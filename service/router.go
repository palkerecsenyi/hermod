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
		api, err := setupRequestAuthentication(authQuery, config)
		if err != nil {
			res.SendError(err)
			return
		}

		req.Auth = api
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
				return
			}

			frame.flag = data[2]
			if frame.flag == ClientSessionRequest || frame.flag == ClientSessionRequestWithAuth {
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

				if frame.flag == ClientSessionRequestWithAuth {
					if len(data) == 7 {
						errorFrame := createErrorClient(ack.endpointId, ack.clientId, "expected token but none specified")
						res.Send(&errorFrame)
						continue
					}

					token := string(data[7:])
					if token == "" {
						errorFrame := createErrorClient(ack.endpointId, ack.clientId, "token was empty")
						res.Send(&errorFrame)
						continue
					}

					var api *AuthAPI
					api, err = setupRequestAuthentication(token, config)
					if err != nil {
						errorFrame := createErrorClient(ack.endpointId, ack.clientId, err.Error())
						res.Send(&errorFrame)
						continue
					}

					err = sessions.setSessionAuth(frame.sessionId, api.authProvider)
					if err != nil {
						errorFrame := createErrorClient(ack.endpointId, ack.clientId, err.Error())
						res.Send(&errorFrame)
						continue
					}
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
				api, err := setupRequestAuthentication(token, config)
				if err != nil {
					res.SendError(err)
					return
				}

				req.Auth = api

				ackFrame := authenticationAck(token)
				res.Send(ackFrame.encode())
				continue
			}

			frame.sessionId = encoder.SliceToU32(data[3:7])

			if frame.flag == Close {
				// if there's an error, the session has probably been closed automatically
				_ = sessions.endSession(frame.sessionId)
				continue
			}

			sd, err := sessions.getSessionData(frame.sessionId)
			if err != nil {
				errorFrame := createErrorSession(frame.endpointId, frame.sessionId, err.Error())
				res.Send(&errorFrame)
				continue
			}

			if frame.flag == Data {
				encodedUnit := data[7:]
				if len(encodedUnit) == 0 {
					log.Println("received malformed message with unit size 0")
					continue
				}

				sd.channel <- &encodedUnit
			} else {
				log.Printf("unrecognised flag %b\n", frame.flag)
			}
		}
	}
}
