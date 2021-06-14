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
}

type Response struct {
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

var endpointRegistrations = map[string]func(*Request, *Response){}

func RegisterEndpoint(path string, handler func(request *Request, response *Response)) {
	endpointRegistrations[path] = handler
}
