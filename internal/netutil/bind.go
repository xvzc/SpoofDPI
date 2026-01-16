//go:build !linux && !darwin && !freebsd

package netutil

import (
	"net"
)

// bindToInterface is a no-op on unsupported platforms.
func bindToInterface(dialer *net.Dialer, iface *net.Interface, targetIP net.IP) error {
	return nil
}
