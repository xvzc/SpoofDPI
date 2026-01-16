//go:build linux

package netutil

import (
	"fmt"
	"net"
)

// bindToInterface sets the dialer's LocalAddr to use the interface's IP as the source address.
// On Linux, we only set LocalAddr because SO_BINDTODEVICE can cause issues with
// socket lookup for incoming packets.
func bindToInterface(dialer *net.Dialer, iface *net.Interface, targetIP net.IP) error {
	if iface == nil {
		return nil
	}

	// Find the interface's IP address to use as source
	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			// Match IP version: use IPv4 source for IPv4 target, IPv6 for IPv6
			if targetIP.To4() != nil && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				dialer.LocalAddr = &net.TCPAddr{IP: ipnet.IP}
				return nil
			} else if targetIP.To4() == nil && ipnet.IP.To4() == nil && !ipnet.IP.IsLoopback() {
				dialer.LocalAddr = &net.TCPAddr{IP: ipnet.IP}
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable IP address found on interface %s for target %s", iface.Name, targetIP)
}
