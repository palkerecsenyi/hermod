package service

import (
	"github.com/palkerecsenyi/hermod/encoder"
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
	path := r.URL.Path
	if path != h.config.Path {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("Hermod does not run on this path"))
		return
	}

	serveConnection(h.config, w, r)
}

type HermodConfig struct {
	WSHandshakeTimeout time.Duration
	Path               string
}

func StartServer(addr string, config *HermodConfig) error {
	if config == nil {
		config = &HermodConfig{
			WSHandshakeTimeout: 10 * time.Second,
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
