//go:build darwin

package socks5

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
)

// socks5SystemNetworkDarwin implements SOCKS5SystemNetwork for SOCKS5 proxy on Darwin
type socks5SystemNetworkDarwin struct {
	logger       zerolog.Logger
	port         uint16
	defaultRoute *netutil.Route
	pacServer    *http.Server
}

// NewSOCKS5SystemNetwork creates a new SOCKS5SystemNetwork for SOCKS5 proxy on Darwin
func NewSOCKS5SystemNetwork(
	logger zerolog.Logger,
	port uint16,
	defaultRoute *netutil.Route,
) SOCKS5SystemNetwork {
	return &socks5SystemNetworkDarwin{
		logger:       logger,
		port:         port,
		defaultRoute: defaultRoute,
	}
}

func (n *socks5SystemNetworkDarwin) DefaultRoute() *netutil.Route {
	return n.defaultRoute
}

func (n *socks5SystemNetworkDarwin) SetNetworkConfig() error {
	service, err := server.GetDefaultNetworkService()
	if err != nil {
		return err
	}

	pacServer, err := server.SetSystemProxy(service, n.port, "SOCKS5")
	if err != nil {
		return err
	}
	n.pacServer = pacServer
	return nil
}

func (n *socks5SystemNetworkDarwin) UnsetNetworkConfig() error {
	if n.pacServer != nil {
		_ = n.pacServer.Close()
	}
	return nil
}
