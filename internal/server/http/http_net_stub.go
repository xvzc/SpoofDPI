//go:build !darwin

package http

import (
	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
)

// httpSystemNetworkStub implements HTTPSystemNetwork for HTTP proxy on unsupported platforms
type httpSystemNetworkStub struct{}

// NewHTTPSystemNetwork creates a new HTTPSystemNetwork for HTTP proxy on unsupported platforms
func NewHTTPSystemNetwork(
	logger zerolog.Logger,
	port uint16,
	defaultRoute *netutil.Route,
) HTTPSystemNetwork {
	return &httpSystemNetworkStub{}
}

func (n *httpSystemNetworkStub) DefaultRoute() *netutil.Route {
	return nil
}

func (n *httpSystemNetworkStub) SetNetworkConfig() error {
	return nil
}

func (n *httpSystemNetworkStub) UnsetNetworkConfig() error {
	return nil
}
