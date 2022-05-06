package service

import (
	"encoding/base64"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
)

type handler struct {
	config *HermodConfig
	path   string
}

// ServeConnection is made public to allow Hermod users to manually choose when to upgrade an HTTP connection to WebSocket/Hermod.
// Pass in the same HermodConfig as with service.StartServer, as well as http.ResponseWriter and http.Request, and ServeConnection
// will attempt to upgrade the HTTP connection to a WebSocket, and start responding to Hermod requests. ServeConnection will
// continue blocking until the WebSocket connection closes (e.g. if the underlying http.Request's context ends)
func ServeConnection(config *HermodConfig, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: config.WSHandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_, _ = w.Write([]byte("This Hermod server only handles WebSocket requests."))
		return
	}

	defer func() {
		_ = conn.Close()
	}()

	ctx := r.Context()
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
		serveWsConnection(&request, &response, r.URL.Query(), config)
		c <- true
	}(done)

	go func(r *Request) {
		for {
			select {
			case <-r.Context.Done():
				return
			case <-done:
				return
			default:
				messageType, data, err := conn.ReadMessage()
				if err != nil {
					wsSendError(conn, errors.New("400: message could not be read"))
					return
				}

				// If the message is text-based (for some reason), assume it's base64-encoded
				if messageType == websocket.TextMessage {
					data, err = base64.StdEncoding.DecodeString(string(data))
					if err != nil {
						wsSendError(conn, errors.New("400: base64-encoded message could not be read"))
						return
					}
				}

				r.Data <- &data
			}
		}
	}(&request)

	<-done
	close(request.Data)
}
