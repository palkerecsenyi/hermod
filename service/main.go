package service

import (
	"hermod/encoder"
	"net"
	"net/http"
	"time"
)

type Service struct {
	Name string
}

type EndpointArgument struct {
	Unit     encoder.Unit
	Streamed bool
}

type Endpoint struct {
	Service *Service
	Path    string
	In      EndpointArgument
	Out     EndpointArgument
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveConnection(h.config, w, r)
}

type HermodConfig struct {
	Timeout time.Duration
}

func StartServer(addr string, config *HermodConfig) error {
	if config == nil {
		config = &HermodConfig{
			Timeout: 30 * time.Second,
		}
	}

	server := &http.Server{
		Handler: &handler{config: config},
		Addr:    addr,
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	err = server.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}
