package service

import (
	"fmt"
	"github.com/palkerecsenyi/hermod/encoder"
	"github.com/palkerecsenyi/hermod/framing"
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

			frame := framing.MessageFrame{}

			frame.EndpointId = encoder.SliceToU16(data[0:2])
			endpoint, ok := endpointRegistrations[frame.EndpointId]
			if !ok && frame.EndpointId != framing.AuthenticationEndpoint {
				res.SendError(fmt.Errorf("endpoint %d not found", frame.EndpointId))
				return
			}

			frame.Flag = data[2]
			if frame.Flag == framing.ClientSessionRequest || frame.Flag == framing.ClientSessionRequestWithAuth {
				ack := framing.SessionFrame{}
				ack.EndpointId = frame.EndpointId

				ack.ClientId = encoder.SliceToU32(data[3:7])

				var err error
				frame.SessionId, err = sessions.createNewSession()
				if err != nil {
					errorFrame := framing.CreateErrorClient(ack.EndpointId, ack.ClientId, err.Error())
					res.Send(&errorFrame)
					continue
				}

				if frame.Flag == framing.ClientSessionRequestWithAuth {
					if len(data) == 7 {
						errorFrame := framing.CreateErrorClient(ack.EndpointId, ack.ClientId, "expected token but none specified")
						res.Send(&errorFrame)
						continue
					}

					token := string(data[7:])
					if token == "" {
						errorFrame := framing.CreateErrorClient(ack.EndpointId, ack.ClientId, "token was empty")
						res.Send(&errorFrame)
						continue
					}

					var api *AuthAPI
					api, err = setupRequestAuthentication(token, config)
					if err != nil {
						errorFrame := framing.CreateErrorClient(ack.EndpointId, ack.ClientId, err.Error())
						res.Send(&errorFrame)
						continue
					}

					err = sessions.setSessionAuth(frame.SessionId, api.authProvider)
					if err != nil {
						errorFrame := framing.CreateErrorClient(ack.EndpointId, ack.ClientId, err.Error())
						res.Send(&errorFrame)
						continue
					}
				}

				res.Send(ack.Ack(frame.SessionId))
				sessions.initiateNewSession(req, res, frame, endpoint)
				continue
			}

			if frame.EndpointId == framing.AuthenticationEndpoint {
				if frame.Flag != framing.Authentication {
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

				ackFrame := framing.NewAuthenticationAck(token)
				res.Send(ackFrame.Encode())
				continue
			}

			frame.SessionId = encoder.SliceToU32(data[3:7])

			if frame.Flag == framing.Close {
				// if there's an error, the session has probably been closed automatically
				_ = sessions.endSession(frame.SessionId)
				continue
			}

			sd, err := sessions.getSessionData(frame.SessionId)
			if err != nil {
				errorFrame := framing.CreateErrorSession(frame.EndpointId, frame.SessionId, err.Error())
				res.Send(&errorFrame)
				continue
			}

			if frame.Flag == framing.Data {
				encodedUnit := data[7:]
				if len(encodedUnit) == 0 {
					log.Println("received malformed message with unit size 0")
					continue
				}

				sd.channel <- &encodedUnit
			} else {
				log.Printf("unrecognised flag %b\n", frame.Flag)
			}
		}
	}
}
