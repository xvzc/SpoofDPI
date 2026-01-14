//go:build darwin

package netutil

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// bindToInterface sets the dialer's Control function to bind the socket
// to a specific network interface using IP_BOUND_IF on Darwin.
func bindToInterface(dialer *net.Dialer, iface *net.Interface, targetIP net.IP) {
	addrs, _ := iface.Addrs()
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
				break
			}
		}
	}
}
