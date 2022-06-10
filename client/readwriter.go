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

type ServiceReadWriter[In encoder.UserFacingHermodUnit, Out encoder.UserFacingHermodUnit] struct {
	Router   *WebSocketRouter
	Endpoint uint16

	HasIn     bool
	OutSample Out

	route *webSocketRoute
}

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

func (rw *ServiceReadWriter[In, Out]) Messages() (<-chan Out, <-chan error, error) {
	dataChan, err := rw.route.run(rw.Router.context)
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

	timeout, cancel := context.WithTimeout(rw.Router.context, rw.Router.Timeout)
	defer cancel()

	select {
	case <-timeout.Done():
		return nil, nil, fmt.Errorf("session open timeout")
	case <-openChan:
		err = rw.Router.unlockClientID(rw.route.client)
		if err != nil {
			return nil, nil, fmt.Errorf("unlocking client ID: %s", err)
		}

		return outputChan, errorChan, nil
	}
}
