package service

import (
	"fmt"
	"github.com/palkerecsenyi/hermod/encoder"
	"log"
	"net/url"
)

func serveWsConnection(req *Request, res *Response, query url.Values, config *HermodConfig) {
	// sessions are specific to a particular WS connection
	sessions := newSessionsStruct()

	if authQuery := query.Get("token"); authQuery != "" {
		err := setupRequestAuthorization(req, authQuery, config)
		if err != nil {
			res.SendError(err)
			return
		}
	}

	var _data *[]byte
	for {
		select {
		case <-req.Context.Done():
			return
		case _data = <-req.Data:
			data := *_data

			frame := messageFrame{}

			frame.endpointId = encoder.SliceToU16(data[0:2])
			endpoint, ok := endpointRegistrations[frame.endpointId]
			if !ok && frame.endpointId != AuthenticationEndpoint {
				res.SendError(fmt.Errorf("endpoint %d not found", frame.endpointId))
				continue
			}

			frame.flag = data[2]
			if frame.flag == ClientSessionRequest {
				ack := sessionFrame{}
				ack.endpointId = frame.endpointId

				ack.clientId = encoder.SliceToU32(data[3:7])

				var err error
				frame.sessionId, err = sessions.createNewSession()
				if err != nil {
					errorFrame := createErrorClient(ack.endpointId, ack.clientId, err.Error())
					res.Send(&errorFrame)
					continue
				}

				res.Send(ack.ack(frame.sessionId))
				sessions.initiateNewSession(req, res, frame, endpoint)
				continue
			}

			if frame.endpointId == AuthenticationEndpoint {
				if frame.flag != Authentication {
					res.SendError(fmt.Errorf("made AuthenticationEndpoint request without using the Authentication flag"))
					return
				}

				token := string(data[3:])
				err := setupRequestAuthorization(req, token, config)
				if err != nil {
					res.SendError(err)
					return
				}

				ackFrame := authenticationAck(token)
				res.Send(ackFrame.encode())
				continue
			}

			frame.sessionId = encoder.SliceToU32(data[3:7])
			sessionChannel, err := sessions.getChannel(frame.sessionId)

			if err != nil {
				errorFrame := createErrorSession(frame.endpointId, frame.sessionId, err.Error())
				res.Send(&errorFrame)
				continue
			}

			if frame.flag == Close {
				// if there's an error, the session has probably been closed automatically
				_ = sessions.endSession(frame.sessionId)
				res.Send(frame.closeAck())
				continue
			}

			if frame.flag == CloseAck {
				continue
			}

			if frame.flag == Data {
				encodedUnit := data[7:]
				if len(encodedUnit) == 0 {
					log.Println("received malformed message with unit size 0")
					continue
				}

				sessionChannel <- &encodedUnit
			} else {
				log.Printf("unrecognised flag %b\n", frame.flag)
			}
		}
	}
}
