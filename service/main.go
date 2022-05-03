package service

import (
	"crypto/tls"
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
	if path != h.path {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("Hermod does not run on this path"))
		return
	}

	ServeConnection(h.config, w, r)
}

type HermodConfig struct {
	WSHandshakeTimeout time.Duration
}

type HermodHTTPConfig struct {
	// TLSConfig specifies an optional tls.Config to use with the HTTP server
	TLSConfig *tls.Config
	// UseWSS specifies whether to use a wss:// connection (WebSocket over HTTPS)
	UseWSS bool
	// CertFile and KeyFile are paths to the certificate files if UseWSS is true
	CertFile, KeyFile string

	// Path is the path at which Hermod should respond to WebSocket Upgrade requests. Requests to any other path
	// will result in a 404 response. Default value is "/hermod"
	Path string
}

// StartServer starts a full HTTP server which responds to WebSocket connections at
// HermodConfig.Path
func StartServer(addr string, config *HermodConfig, httpConfig *HermodHTTPConfig) error {
	if config == nil {
		config = &HermodConfig{
			WSHandshakeTimeout: 10 * time.Second,
		}
	}

	if httpConfig == nil {
		httpConfig = &HermodHTTPConfig{
			Path: "/hermod",
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		}
	}

	server := &http.Server{
		Handler: &handler{
			config: config,
			path:   httpConfig.Path,
		},
		Addr:      addr,
		TLSConfig: httpConfig.TLSConfig,
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if httpConfig.UseWSS {
		err = server.ServeTLS(lis, httpConfig.CertFile, httpConfig.KeyFile)
	} else {
		err = server.Serve(lis)
	}
	if err != nil {
		return err
	}

	return nil
}
