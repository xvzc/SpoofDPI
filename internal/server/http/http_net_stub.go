//go:build !darwin

package http

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
)

type httpSystemNetworkStub struct{}

func NewHTTPSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
) HTTPSystemNetwork {
	return &httpSystemNetworkStub{}
}

func (n *httpSystemNetworkStub) DefaultRoute() *netutil.Route {
	return nil
}

type httpStateDarwin struct {
	Service   string `json:"service"`
	Port      uint16 `json:"port"`
	ProxyType string `json:"proxyType"`
	PACURL    string `json:"pacURL"`
}

func createState(
	defaultRoute *netutil.Route,
	port uint16,
	pacURL string,
) (*httpStateDarwin, error) {
	return &httpStateDarwin{}, nil
}

func saveState(state *httpStateDarwin) error {
	return nil
}

func loadState() (*httpStateDarwin, bool, error) {
	return nil, false, nil
}

func deleteState() error {
	return nil
}

func configurationJobs(
	ctx context.Context,
	logger zerolog.Logger,
	state *httpStateDarwin,
) []server.ConfigurationJob {
	return nil
}
