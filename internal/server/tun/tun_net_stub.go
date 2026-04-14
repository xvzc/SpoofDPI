//go:build !darwin && !linux && !freebsd

package tun

import (
	"context"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/server"
	"golang.zx2c4.com/wireguard/tun"
)

type tunStateStub struct{}

func loadState() (*tunStateStub, bool, error) {
	return nil, false, nil
}

func createState(sysNet TUNSystemNetwork) (*tunStateStub, error) {
	return nil, nil
}

func deleteState() error {
	return nil
}

func saveState(state *tunStateStub) error {
	return nil
}

func createTunDevice() (tun.Device, error) {
	return nil, nil
}

func configurationJobs(
	ctx context.Context,
	logger zerolog.Logger,
	state *tunStateStub,
) []server.ConfigurationJob {
	return nil
}

// tunSystemNetworkStub implements TUNSystemNetwork for unsupported platforms
type tunSystemNetworkStub struct {
	logger zerolog.Logger
}

// NewTUNSystemNetwork creates a new TUNSystemNetwork for TUN mode on unsupported platforms
func NewTUNSystemNetwork(
	logger zerolog.Logger,
	defaultRoute *netutil.Route,
	fibID int,
) (TUNSystemNetwork, error) {
	return &tunSystemNetworkStub{logger: logger}, nil
}

func (n *tunSystemNetworkStub) TunDevice() tun.Device {
	return nil
}

func (n *tunSystemNetworkStub) DefaultRoute() *netutil.Route {
	return nil
}

func (n *tunSystemNetworkStub) FIBID() int {
	return 1
}

func (n *tunSystemNetworkStub) BindDialer(
	dialer *net.Dialer,
	network string,
	targetIP net.IP,
) error {
	return nil
}
