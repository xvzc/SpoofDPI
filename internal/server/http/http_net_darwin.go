//go:build darwin

package http

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
)

// httpSystemNetworkDarwin implements HTTPSystemNetwork for HTTP proxy on Darwin
type httpSystemNetworkDarwin struct {
	logger       zerolog.Logger
	port         uint16
	defaultRoute *netutil.Route
	pacServer    *http.Server
}

// NewHTTPSystemNetwork creates a new HTTPSystemNetwork for HTTP proxy on Darwin
func NewHTTPSystemNetwork(
	logger zerolog.Logger,
	port uint16,
	defaultRoute *netutil.Route,
) HTTPSystemNetwork {
	return &httpSystemNetworkDarwin{
		logger:       logger,
		port:         port,
		defaultRoute: defaultRoute,
	}
}

func (n *httpSystemNetworkDarwin) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *httpSystemNetworkDarwin) SetNetworkConfig() error {
	service, err := server.GetDefaultNetworkService()
	if err != nil {
		return err
	}

	pacServer, err := server.SetSystemProxy(service, n.port, "PROXY")
	if err != nil {
		return err
	}
	n.pacServer = pacServer
	return nil
}

func (n *httpSystemNetworkDarwin) UnsetNetworkConfig() error {
	if n.pacServer != nil {
		_ = n.pacServer.Close()
	}
	return nil
}
