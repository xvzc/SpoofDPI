package netutil

import (
	"fmt"
	"net"
)

// FindSafeSubnet scans the 10.0.0.0/8 range to find an unused /30 subnet
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

// GetDefaultInterface returns the name of the default network interface
func GetDefaultInterface() (string, error) {
	ifaceName, _, err := GetDefaultInterfaceAndGateway()
	return ifaceName, err
}
