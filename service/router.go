package service

import (
	"errors"
	"github.com/palkerecsenyi/hermod/encoder"
	"log"
)

func serveWsConnection(req *Request, res *Response) {
	// sessions are specific to a particular WS connection
	sessions := newSessionsStruct()

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
			if !ok {
				res.SendError(errors.New("endpoint not found"))
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
