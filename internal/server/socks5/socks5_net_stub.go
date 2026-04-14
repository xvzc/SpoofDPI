//go:build !darwin

package socks5

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
)

type socks5SystemNetworkStub struct{}

func NewSOCKS5SystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
) SOCKS5SystemNetwork {
	return &socks5SystemNetworkStub{}
}

func (n *socks5SystemNetworkStub) DefaultRoute() *netutil.Route {
	return nil
}

type socks5StateDarwin struct {
	Service   string `json:"service"`
	Port      uint16 `json:"port"`
	ProxyType string `json:"proxyType"`
	PACURL    string `json:"pacURL"`
}

func createState(
	defaultRoute *netutil.Route,
	port uint16,
	pacURL string,
) (*socks5StateDarwin, error) {
	return &socks5StateDarwin{}, nil
}

func saveState(state *socks5StateDarwin) error {
	return nil
}

func loadState() (*socks5StateDarwin, bool, error) {
	return nil, false, nil
}

func deleteState() error {
	return nil
}

func configurationJobs(
	ctx context.Context,
	logger zerolog.Logger,
	state *socks5StateDarwin,
) []server.ConfigurationJob {
	return nil
}
