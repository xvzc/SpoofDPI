//go:build !darwin

package netutil

import "net"

// bindToInterface is a no-op on non-Darwin systems.
// Interface binding is handled differently or not supported on other platforms.
func bindToInterface(dialer *net.Dialer, iface *net.Interface, targetIP net.IP) {
	// No-op: interface binding via IP_BOUND_IF is Darwin-specific
}
