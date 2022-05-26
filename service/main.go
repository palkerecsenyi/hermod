// Package service provides a server and an application-layer protocol for transmitting encoded Hermod units. It does
// not provide a client (at the moment).
//
// You can use `service` by adding service endpoints to Hermod YAML files and then calling the generated Register<name>()
// function to register an endpoint handler. This will get called whenever a client requests that endpoint. Endpoints
// have fairly simple interfaces that work in a similar way to net/http, giving you a request and response object. Any
// errors returned within endpoint handlers are sent as encoded error objects to the client.
//
// The package also provides a JWT-based authentication system. This allows clients to send a JWT along with their request,
// either as a query parameter in the initial session establishment request, or as a separate authentication packet at any
// time during the request. For more details see HermodAuthenticationConfig.
//
// The server provided by the package is designed to be used to open a single WebSocket connection between a client and the
// server. All Hermod requests should be transmitted over that single connection to avoid the overhead of performing a
// WebSocket handshake. To that effect, you can either call StartServer to start a full HTTP server or you can use
// ServeConnection to manually perform WebSocket upgrades using your existing HTTP server, allowing you to use conventional
// HTTP endpoints at the same time as your Hermod endpoint.
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
	// AuthenticationConfig is optional, you can use it to enable a highly opinionated JWT-based authentication system.
	// If you want anything custom, you'll need to build it on your own for now!
	AuthenticationConfig *HermodAuthenticationConfig
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
