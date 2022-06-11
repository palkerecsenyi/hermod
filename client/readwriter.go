package client

import (
	"context"
	"fmt"
	"github.com/palkerecsenyi/hermod/encoder"
)

type DummyOutSample struct{}

func (DummyOutSample) GetDefinition() *encoder.Unit {
	return nil
}
func (DummyOutSample) DecodeAbstract(*[]byte) (encoder.UserFacingHermodUnit, error) {
	return DummyOutSample{}, nil
}

// ServiceReadWriter is the user-facing frontend for the Hermod client and is referenced in generated code.
type ServiceReadWriter[In encoder.UserFacingHermodUnit, Out encoder.UserFacingHermodUnit] struct {
	Router   *WebSocketRouter
	Endpoint uint16

	HasIn     bool
	OutSample Out

	Context context.Context
	cancel  context.CancelFunc
	route   *webSocketRoute
}

// Init is called in generated code and must always be run before any other method will work as expected.
// It calls rw.Router.initRoute to reserve a new client ID and create the route struct.
func (rw *ServiceReadWriter[In, Out]) Init(token ...string) error {
	if rw.Router == nil {
		return fmt.Errorf("router must not be nil")
	}

	route, err := rw.Router.initRoute(rw.Endpoint, token...)
	if err != nil {
		return fmt.Errorf("initing route: %s", err)
	}

	rw.route = route
	return nil
}

// Messages opens the session and returns a pair of channels: one for data values and another for session-specific
// errors. It blocks until the session has been opened, and returns an error (as the 3rd return value) if the session
// handshake times out.
func (rw *ServiceReadWriter[In, Out]) Messages() (<-chan Out, <-chan error, error) {
	if rw.Context == nil {
		rw.Context, rw.cancel = context.WithCancel(rw.Router.context)
	}

	dataChan, err := rw.route.run(rw.Context)
	if err != nil {
		return nil, nil, err
	}

	outputChan := make(chan Out)
	errorChan := make(chan error)
	openChan := make(chan struct{})
	go func() {
		for {
			nextData, open := <-dataChan
			if !open {
				close(outputChan)
				close(errorChan)
				return
			}

			if nextData.event == eventSessionReady {
				close(openChan)
				continue
			}

			if nextData.error != nil {
				errorChan <- nextData.error
				continue
			}

			if nextData.event == eventData {
				decoded, err := encoder.UserDecode(rw.OutSample, &nextData.data)
				if err != nil {
					errorChan <- fmt.Errorf("failed to decode: %s", err)
					continue
				}

				outputChan <- decoded.(Out)
			}
		}
	}()

	timeout, cancel := context.WithTimeout(rw.Context, rw.Router.Timeout)
	defer cancel()

	select {
	case <-timeout.Done():
		return nil, nil, fmt.Errorf("session open timeout")
	case <-openChan:
		rw.Router.unlockClientID(rw.route.client)
		return outputChan, errorChan, nil
	}
}

// NextMessage reads a single new message. readyChan is a channel that gets closed once the session has been opened.
// If NextMessage is called after the session has already been opened, readyChan will be closed instantaneously.
func (rw *ServiceReadWriter[In, Out]) NextMessage(readyChan chan<- interface{}) (*Out, error) {
	dataChan, errorChan, err := rw.Messages()
	if err != nil {
		return nil, err
	}

	close(readyChan)

	select {
	case data, ok := <-dataChan:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return &data, nil
	case sessionError, ok := <-errorChan:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return nil, sessionError
	}
}

// Send sends typed data on the session. The session must be open, or this will return an error.
func (rw *ServiceReadWriter[In, Out]) Send(data In) error {
	if !rw.HasIn {
		return fmt.Errorf("endpoint doesn't have input parameter")
	}

	encoded, err := encoder.UserEncode(data)
	if err != nil {
		return fmt.Errorf("encoding input data: %s", err)
	}

	err = rw.route.send(*encoded)
	if err != nil {
		return fmt.Errorf("sending encoded data: %s", err)
	}

	return nil
}

// Call opens the session, sends one message, and reads one message. Best for unary-in unary-out functions.
func (rw *ServiceReadWriter[In, Out]) Call(data In) (*Out, error) {
	readyChan := make(chan interface{})
	errorChan := make(chan error)
	responseChan := make(chan *Out)

	go func() {
		response, err := rw.NextMessage(readyChan)
		if err != nil {
			errorChan <- err
		} else {
			responseChan <- response
		}
	}()

	<-readyChan
	err := rw.Send(data)
	if err != nil {
		return nil, fmt.Errorf("send: %s", err)
	}

	select {
	case response := <-responseChan:
		return response, nil
	case err = <-errorChan:
		return nil, fmt.Errorf("receive: %s", err)
	}
}

// Close closes the route. Re-opening is not supported and may result in unexpected behaviour.
func (rw *ServiceReadWriter[In, Out]) Close() error {
	if rw.cancel == nil {
		return fmt.Errorf("context has not been initialised")
	}

	rw.cancel()
	return nil
}
