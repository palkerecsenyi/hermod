package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"sync"
	"time"
)

type WebSocketRouter struct {
	URL     url.URL
	Timeout time.Duration
	// routeStore is a map of type map[uint32]*webSocketRoute
	routeStore sync.Map
	connection *websocket.Conn
	data       chan []byte

	context context.Context
	cancel  context.CancelFunc
}

func (router *WebSocketRouter) Connect(token ...string) error {
	if router.URL.Scheme == "" {
		router.URL.Scheme = "ws"
	}

	if router.URL.Path == "" {
		router.URL.Path = "/hermod/"
	}

	if len(token) == 1 {
		router.URL.Query().Add("token", token[0])
	}

	if router.context != nil {
		return fmt.Errorf("connect already called (reconnect not supported)")
	}

	connection, _, err := websocket.DefaultDialer.Dial(router.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("opening websocket: %s", err)
	}

	router.connection = connection
	router.routeStore = sync.Map{}
	router.data = make(chan []byte)

	router.context, router.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-router.context.Done():
				return
			default:
				_, message, err := connection.ReadMessage()
				if err != nil {
					log.Printf("hermod read error: %s\n", err)
					continue
				}

				router.data <- message
			}
		}
	}()
	return nil
}

func (router *WebSocketRouter) Close() error {
	if router.connection == nil {
		return fmt.Errorf("no connection exists")
	}

	err := router.connection.Close()
	if err != nil {
		return err
	}

	if router.cancel != nil {
		router.cancel()
	}

	return nil
}

func (router *WebSocketRouter) send(data interface{}) error {
	if router.connection == nil {
		return fmt.Errorf("connection required to send message")
	}

	if stringData, ok := data.(string); ok {
		return router.connection.WriteMessage(websocket.TextMessage, []byte(stringData))
	}

	if binaryData, ok := data.([]byte); ok {
		return router.connection.WriteMessage(websocket.BinaryMessage, binaryData)
	}

	return fmt.Errorf("unsupported message type")
}

func (router *WebSocketRouter) initRoute(endpoint uint16, token ...string) (*webSocketRoute, error) {
	if router.connection == nil {
		return nil, fmt.Errorf("connection required before opening route")
	}

	// find an unused client ID
	var client uint32 = 0
	for i := uint32(0); i <= uint32(0xffffffff); i++ {
		used := false
		router.routeStore.Range(func(thisClient, _ any) bool {
			if val, ok := thisClient.(uint32); ok && val == client {
				used = true
				return false
			}

			return true
		})

		if !used {
			client = i
			break
		}

		if i+1 > 0xffffffff {
			return nil, fmt.Errorf("no more client IDs remaining")
		}
	}

	route := webSocketRoute{
		client:   client,
		endpoint: endpoint,
		router:   router,
	}

	if len(token) == 1 {
		t := token[0]
		route.token = &t
	}

	router.routeStore.Store(client, &route)
	return &route, nil
}

func (router *WebSocketRouter) unlockClientID(client uint32) {
	router.routeStore.Delete(client)
}
