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

	router *WebSocketRouter
}

const (
	eventData = iota
	eventSessionAck
	eventSessionReady
)

type receiveOutput struct {
	data  []byte
	error error
	event int
}

func (route *webSocketRoute) receive(ctx context.Context, output chan<- receiveOutput, sessionAck bool) {
	defer close(output)

	for {
		select {
		case <-ctx.Done():
			output <- receiveOutput{
				error: fmt.Errorf("context ended"),
			}
			return
		case data, ok := <-route.router.data:
			if !ok {
				output <- receiveOutput{
					error: fmt.Errorf("connection closed"),
				}
				return
			}

			endpoint := encoder.SliceToU16(data[0:2])
			if endpoint != route.endpoint {
				continue
			}

			flag := data[2]
			if flag == framing.ServerSessionAck {
				if sessionAck {
					client := encoder.SliceToU32(data[3:7])
					if client != route.client {
						continue
					}

					output <- receiveOutput{
						data:  data[7:11],
						event: eventSessionAck,
					}
				}
				continue
			}

			if flag == framing.ErrorClientID || flag == framing.ErrorSessionID {
				clientOrSession := encoder.SliceToU32(data[3:7])
				if route.session != nil && flag == framing.ErrorSessionID && clientOrSession == *route.session {
					message := data[7:]
					output <- receiveOutput{
						error: fmt.Errorf("server (session ID): %s", message),
					}
					return
				}

				if flag == framing.ErrorClientID && clientOrSession == route.client {
					message := data[7:]
					output <- receiveOutput{
						error: fmt.Errorf("server (client ID): %s", message),
					}
					return
				}

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
				output <- receiveOutput{
					data:  data[7:],
					event: eventData,
				}
			}
		}
	}
}

func (route *webSocketRoute) run(ctx context.Context) (chan receiveOutput, error) {
	route.Lock()
	defer route.Unlock()

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

	input := make(chan receiveOutput)
	go func() {
		route.receive(ctx, input, true)
	}()

	output := make(chan receiveOutput)
	go func() {
		for data := range input {
			if data.error != nil {
				output <- data
				break
			}

			if data.event == eventSessionAck {
				id := encoder.SliceToU32(data.data)
				route.session = &id
				output <- receiveOutput{
					event: eventSessionReady,
				}
				continue
			}

			if data.event == eventData {
				output <- data
				continue
			}
		}
		close(output)
	}()

	// this allows for multiple calls to run(), only sending an open packet if this hasn't already been done
	if !route.sessionRequestSent {
		route.sessionRequestSent = true

		var err error
		err = route.router.send(frame.Encode())
		if err != nil {
			return nil, fmt.Errorf("sending open frame: %s", err)
		}
	}

	return output, nil
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
