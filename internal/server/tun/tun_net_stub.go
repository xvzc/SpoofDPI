//go:build !darwin && !linux && !freebsd

package tun

import (
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"golang.zx2c4.com/wireguard/tun"
)

func SetRoute(iface string, subnets []string) error {
	return nil
}

func UnsetRoute(iface string, subnets []string) error {
	return nil
}

func SetInterfaceAddress(iface string, local string, remote string) error {
	return nil
}

func createTunDevice() (tun.Device, error) {
	return nil, nil
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

func (n *tunSystemNetworkStub) SetNetworkConfig() error {
	return nil
}

func (n *tunSystemNetworkStub) UnsetNetworkConfig() error {
	return nil
}

func (n *tunSystemNetworkStub) BindDialer(
	dialer *net.Dialer,
	network string,
	targetIP net.IP,
) error {
	return nil
}
