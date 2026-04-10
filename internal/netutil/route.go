package netutil

import (
	"fmt"
	"net"
	"strings"
)

func FindSafeSubnet() (string, string, error) {
	// Retrieve all active interface addresses to prevent CIDR overlapping.
	// Checking against existing networks is faster than sending probe packets.
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	var existingNets []*net.IPNet
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			existingNets = append(existingNets, ipnet)
		}
	}

	// Iterate through the 10.0.0.0/8 private range with a /30 step.
	// A /30 subnet provides exactly two usable end-point IP addresses.
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			// Construct candidate IP pair: 10.i.j.1 and 10.i.j.2
			local := net.IPv4(10, byte(i), byte(j), 1)
			remote := net.IPv4(10, byte(i), byte(j), 2)

			conflict := false
			for _, ipnet := range existingNets {
				if ipnet.Contains(local) || ipnet.Contains(remote) {
					conflict = true
					break
				}
			}

			if !conflict {
				return local.String(), remote.String(), nil
			}
		}
	}

	return "", "", fmt.Errorf("failed to find an available address in 10.0.0.0/8")
}

// bindToInterface sets the dialer's LocalAddr to use the interface's IP as the source address.
// On Linux, we only set LocalAddr because SO_BINDTODEVICE can cause issues with
// socket lookup for incoming packets.
func bindToInterface(
	network string,
	dialer *net.Dialer,
	iface *net.Interface,
	targetIP net.IP,
) error {
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
				if strings.HasPrefix(network, "tcp") {
					dialer.LocalAddr = &net.TCPAddr{IP: ipnet.IP}
				} else if strings.HasPrefix(network, "udp") {
					dialer.LocalAddr = &net.UDPAddr{IP: ipnet.IP}
				} else {
					dialer.LocalAddr = &net.IPAddr{IP: ipnet.IP}
				}
				return nil
			} else if targetIP.To4() == nil && ipnet.IP.To4() == nil && !ipnet.IP.IsLoopback() {
				if strings.HasPrefix(network, "tcp") {
					dialer.LocalAddr = &net.TCPAddr{IP: ipnet.IP}
				} else if strings.HasPrefix(network, "udp") {
					dialer.LocalAddr = &net.UDPAddr{IP: ipnet.IP}
				} else {
					dialer.LocalAddr = &net.IPAddr{IP: ipnet.IP}
				}
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

// GetDefaultInterfaceAndGateway returns the name of the default network interface and the gateway IP
func GetDefaultInterfaceAndGateway() (string, string, error) {
	// Dial a public DNS server to determine the default interface
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = conn.Close() }()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", err
	}

	var ifaceName string
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(localAddr.IP) {
					ifaceName = iface.Name
					break
				}
			}
		}
		if ifaceName != "" {
			break
		}
	}

	if ifaceName == "" {
		return "", "", fmt.Errorf("default interface not found")
	}

	// Get gateway by parsing route table
	gateway, err := getDefaultGateway()
	if err != nil {
		return "", "", fmt.Errorf("failed to get default gateway: %w", err)
	}

	return ifaceName, gateway, nil
}
