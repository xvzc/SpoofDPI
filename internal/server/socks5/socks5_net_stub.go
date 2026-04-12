//go:build !darwin

package socks5

import (
	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
)

// socks5SystemNetworkStub implements SOCKS5SystemNetwork for SOCKS5 proxy on unsupported platforms
type socks5SystemNetworkStub struct{}

// NewSOCKS5SystemNetwork creates a new SOCKS5SystemNetwork for SOCKS5 proxy on unsupported platforms
func NewSOCKS5SystemNetwork(
	logger zerolog.Logger,
	port uint16,
	defaultRoute *netutil.Route,
) SOCKS5SystemNetwork {
	return &socks5SystemNetworkStub{}
}

func (n *socks5SystemNetworkStub) DefaultRoute() *netutil.Route {
	return nil
}

func (n *socks5SystemNetworkStub) SetNetworkConfig() error {
	return nil
}

func (n *socks5SystemNetworkStub) UnsetNetworkConfig() error {
	return nil
}
