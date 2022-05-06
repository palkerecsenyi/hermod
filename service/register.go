package service

import (
	"context"
	"net/http"
	"sync"
)

type Request struct {
	Context context.Context
	Data    chan *[]byte
	Headers http.Header

	// Auth will be nil if authentication hasn't been set up in HermodConfig or if the WebSocket connection doesn't have
	// an authentication session assigned to it (either because there was no initial Authorization header or no Hermod
	// messages with the Authentication flag has been received)
	Auth *AuthAPI
}

type Response struct {
	// you're not supposed to write concurrently to a WebSocket connection
	sync.Mutex
	sendFunction func(data *[]byte, error bool)
}

func (res *Response) Send(data *[]byte) {
	res.Lock()
	res.sendFunction(data, false)
	res.Unlock()
}

func (res *Response) SendError(err error) {
	t := []byte(err.Error())
	res.Lock()
	res.sendFunction(&t, true)
	res.Unlock()
}

var endpointRegistrations = map[uint16]func(*Request, *Response){}

func RegisterEndpoint(id uint16, handler func(request *Request, response *Response)) {
	endpointRegistrations[id] = handler
}
