//go:build darwin || freebsd

package netutil

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// bindToInterface sets the dialer's Control function to bind the socket
// to a specific network interface using IP_BOUND_IF on BSD systems.
func bindToInterface(dialer *net.Dialer, iface *net.Interface, targetIP net.IP) error {
	if iface == nil {
		return nil
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if targetIP.To4() != nil && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				ifaceIndex := iface.Index
				dialer.Control = func(network, address string, c syscall.RawConn) error {
					var setsockoptErr error
					err := c.Control(func(fd uintptr) {
						setsockoptErr = unix.SetsockoptInt(
							int(fd),
							unix.IPPROTO_IP,
							unix.IP_BOUND_IF,
							ifaceIndex,
						)
					})
					if err != nil {
						return err
					}

					return setsockoptErr
				}
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable IP address found on interface %s for target %s", iface.Name, targetIP)
}
