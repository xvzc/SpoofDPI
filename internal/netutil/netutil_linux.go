//go:build linux

package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
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

	return fmt.Errorf(
		"no suitable IP address found on interface %s for target %s",
		iface.Name,
		targetIP,
	)
}

// getDefaultGateway parses the system route table to find the default gateway on Linux
func getDefaultGateway() (string, error) {
	// Use ip route to get the default route on Linux
	cmd := exec.Command("ip", "route", "show", "default")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output: "default via 192.168.0.1 dev enp12s0 ..."
	fields := strings.Fields(string(out))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("could not parse gateway from ip route output: %s", string(out))
}
