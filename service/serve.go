package service

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024, WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type handler struct {
	config *HermodConfig
}

func serveConnection(config *HermodConfig, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_, _ = w.Write([]byte("This Hermod server only handles WebSocket requests."))
		return
	}

	defer func() {
		_ = conn.Close()
	}()

	path := r.URL.Path
	endpoint, ok := endpointRegistrations[path]
	if !ok {
		wsSendError(conn, errors.New("404: endpoint not found"))
		return
	}

	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(config.Timeout))
	request := Request{
		Context: ctx,
		Headers: r.Header,
		Data:    make(chan *[]byte),
	}
	response := Response{
		sendFunction: func(data *[]byte, error bool) {
			if error {
				wsSendError(conn, errors.New(string(*data)))
			} else {
				wsSendBinary(conn, data)
			}
		},
	}

	done := make(chan bool)
	go func(c chan bool) {
		endpoint(&request, &response)
		c <- true
	}(done)

	go func(r *Request) {
		for {
			select {
			case <-done:
				return
			default:
				messageType, data, err := conn.ReadMessage()
				if err != nil {
					wsSendError(conn, errors.New("400: message could not be read"))
					cancel()
					return
				}

				if messageType == websocket.TextMessage {
					data, err = base64.StdEncoding.DecodeString(string(data))
					if err != nil {
						wsSendError(conn, errors.New("400: base64-encoded message could not be read"))
						cancel()
						return
					}
				}

				r.Data <- &data
			}
		}
	}(&request)

	<-done
	close(request.Data)
	cancel()
}
