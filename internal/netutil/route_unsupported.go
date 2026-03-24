//go:build !linux && !darwin && !freebsd

package netutil

import (
	"fmt"
	"net"
)

// bindToInterface is a no-op on unsupported platforms.
func bindToInterface(
	network string,
	dialer *net.Dialer,
	iface *net.Interface,
	targetIP net.IP,
) error {
	return nil
}

// getDefaultGateway is not supported on this platform.
func getDefaultGateway() (string, error) {
	return "", fmt.Errorf("getDefaultGateway not supported on this platform")
}
