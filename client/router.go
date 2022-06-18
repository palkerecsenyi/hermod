package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/palkerecsenyi/hermod/encoder"
	"log"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type WebSocketRouter struct {
	URL     url.URL
	Timeout time.Duration

	routeStore      map[uint32]*webSocketRoute
	routeStoreMutex sync.Mutex

	connectionMutex sync.Mutex
	connection      *websocket.Conn

	openMutex sync.Mutex

	data    chan []byte
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
	router.routeStore = map[uint32]*webSocketRoute{}
	router.data = make(chan []byte)

	router.context, router.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-router.context.Done():
				close(router.data)
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

	router.connectionMutex.Lock()
	defer router.connectionMutex.Unlock()

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

	router.routeStoreMutex.Lock()
	defer router.routeStoreMutex.Unlock()

	// find an unused client ID
	var client uint32 = 0
	for i := uint32(0); i <= uint32(0xffffffff); i++ {
		_, clientInUse := router.routeStore[i]
		if !clientInUse {
			client = i
			break
		}

		if i+1 > 0xffffffff {
			return nil, fmt.Errorf("no more client IDs remaining")
		}
	}

	routeChan := make(chan *[]byte)
	receiveDoneChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-router.context.Done():
				close(routeChan)
				return
			case <-receiveDoneChan:
				close(routeChan)
				return
			case data, ok := <-router.data:
				if !ok {
					close(routeChan)
					return
				}

				// a lightweight check to reduce the load on individual routes.
				// obviously this doesn't filter multiple same-endpoint sessions though.
				dataEndpoint := encoder.SliceToU16(data[0:2])
				if dataEndpoint != endpoint {
					continue
				}

				routeChan <- &data
			}
		}
	}()

	received := make(chan receiveOutput)
	route := webSocketRoute{
		client:      client,
		endpoint:    endpoint,
		router:      router,
		websocketIn: routeChan,
		received:    received,
	}

	if len(token) == 1 {
		t := token[0]
		route.token = &t
	}

	router.routeStore[client] = &route

	go func() {
		route.receive(router.context)
		// let the for-loop in the goroutine above know to stop listening
		close(receiveDoneChan)
	}()

	// make sure the route.receive call in the goroutine is actually ready before continuing
	runtime.Gosched()
	return &route, nil
}

func (router *WebSocketRouter) unlockClientID(client uint32) {
	router.routeStoreMutex.Lock()
	defer router.routeStoreMutex.Unlock()
	delete(router.routeStore, client)
}
