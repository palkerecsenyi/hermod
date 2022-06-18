package client

import (
	"context"
	"fmt"
	"github.com/palkerecsenyi/hermod/encoder"
	"github.com/palkerecsenyi/hermod/framing"
	"sync"
)

type webSocketRoute struct {
	// mutex prevents session request from being sent multiple times
	sync.Mutex

	endpoint           uint16
	client             uint32
	session            *uint32
	sessionRequestSent bool

	token *string

	websocketIn <-chan *[]byte
	received    chan receiveOutput
	router      *WebSocketRouter
}

const (
	eventData = iota
	eventSessionAck
)

type receiveOutput struct {
	data  []byte
	error error
	event int
}

func (route *webSocketRoute) receive(ctx context.Context) {
	defer func() {
		close(route.received)
	}()

	for {
		select {
		case <-ctx.Done():
			route.received <- receiveOutput{
				error: fmt.Errorf("context ended"),
			}
			return
		case _data, ok := <-route.websocketIn:
			data := *_data

			if !ok {
				route.received <- receiveOutput{
					error: fmt.Errorf("connection closed"),
				}
				return
			}

			flag := data[2]
			if flag == framing.ServerSessionAck {
				client := encoder.SliceToU32(data[3:7])
				if client != route.client {
					continue
				}

				sessionId := encoder.SliceToU32(data[7:11])
				route.router.unlockClientID(route.client)
				route.session = &sessionId

				route.received <- receiveOutput{
					event: eventSessionAck,
				}
				continue
			}

			if flag == framing.ErrorClientID || flag == framing.ErrorSessionID {
				clientOrSession := encoder.SliceToU32(data[3:7])
				if route.session != nil && flag == framing.ErrorSessionID && clientOrSession == *route.session {
					message := data[7:]
					route.received <- receiveOutput{
						error: fmt.Errorf("server (session ID): %s", message),
					}
					return
				}

				if flag == framing.ErrorClientID && clientOrSession == route.client {
					message := data[7:]
					route.received <- receiveOutput{
						error: fmt.Errorf("server (client ID): %s", message),
					}
					return
				}

				continue
			}

			if route.session == nil {
				continue
			}

			session := encoder.SliceToU32(data[3:7])
			if session != *route.session {
				continue
			}

			if flag == framing.Close {
				return
			}

			if flag == framing.Data {
				route.received <- receiveOutput{
					data:  data[7:],
					event: eventData,
				}
			}
		}
	}
}

// open sends an ClientSessionRequest message.
// If the session has already been opened, open returns immediately and without error.
func (route *webSocketRoute) open() error {
	route.Lock()
	defer route.Unlock()

	if route.sessionRequestSent {
		return nil
	}

	var flag uint8 = framing.ClientSessionRequest
	var token string
	if route.token != nil {
		flag = framing.ClientSessionRequestWithAuth
		token = *route.token
	}

	frame := framing.SessionFrame{
		EndpointId: route.endpoint,
		Flag:       flag,
		ClientId:   route.client,
		Token:      token,
	}

	err := route.router.send(frame.Encode())
	if err != nil {
		return fmt.Errorf("sending open frame: %s", err)
	}

	route.sessionRequestSent = true
	return nil
}

func (route *webSocketRoute) send(message []byte) error {
	if route.session == nil {
		return fmt.Errorf("session not open")
	}

	frame := framing.MessageFrame{
		EndpointId: route.endpoint,
		Flag:       framing.Data,
		SessionId:  *route.session,
		Data:       message,
	}
	err := route.router.send(frame.Encode())
	if err != nil {
		return fmt.Errorf("sending messsage: %s", err)
	}

	return nil
}

func (route *webSocketRoute) close() error {
	if route.session == nil {
		return fmt.Errorf("cannot close, session not open")
	}

	frame := framing.MessageFrame{
		EndpointId: route.endpoint,
		SessionId:  *route.session,
	}
	err := route.router.send(frame.Close())
	if err != nil {
		return fmt.Errorf("sending close: %s", err)
	}

	return nil
}
