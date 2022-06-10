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

type routeStore struct {
	sync.Mutex
	routes []webSocketRoute
}

type WebSocketRouter struct {
	URL        url.URL
	Timeout    time.Duration
	routeStore routeStore
	connection *websocket.Conn
	data       chan []byte

	context context.Context
	close   chan struct{}
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

	connection, _, err := websocket.DefaultDialer.Dial(router.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("opening websocket: %s", err)
	}

	router.connection = connection
	router.routeStore = routeStore{
		routes: []webSocketRoute{},
	}
	router.data = make(chan []byte)

	var cancel context.CancelFunc
	router.context, cancel = context.WithCancel(context.Background())

	router.close = make(chan struct{})
	go func(close <-chan struct{}) {
		for {
			select {
			case <-router.context.Done():
				return
			case _, open := <-close:
				if !open {
					cancel()
				}
			}
		}
	}(router.close)

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

	// notifies the goroutine in Connect() to cancel the context
	close(router.close)

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

	router.routeStore.Lock()
	defer router.routeStore.Unlock()

	// find an unused client ID
	var client uint32 = 0
	for i := uint32(0); i <= uint32(0xffffffff); i++ {
		used := false
		for _, route := range router.routeStore.routes {
			if route.client == i {
				used = true
				break
			}
		}

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

	router.routeStore.routes = append(router.routeStore.routes, route)

	return &route, nil
}

func (router *WebSocketRouter) unlockClientID(client uint32) error {
	router.routeStore.Lock()
	defer router.routeStore.Unlock()

	routeIndex := -1
	for i, route := range router.routeStore.routes {
		if route.client == client {
			routeIndex = i
			break
		}
	}

	if routeIndex == -1 {
		return fmt.Errorf("client ID not in use")
	}

	routeStoreSize := len(router.routeStore.routes)
	router.routeStore.routes[routeIndex] = router.routeStore.routes[routeStoreSize-1]
	router.routeStore.routes = router.routeStore.routes[:routeStoreSize-1]

	return nil
}
