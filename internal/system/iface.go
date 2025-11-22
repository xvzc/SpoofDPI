package system

import (
	"errors"
	"fmt"
	"net"
)

// FindDefaultInterface attempts to dial a public DNS server via UDP
// to find the default network interface used for internet connection.
// Returns *net.Interface instead of string.
func FindDefaultInterface() (*net.Interface, error) {
	// List of public DNS servers
	dnsServers := []string{
		"8.8.8.8:53",
		"8.8.4.4:53",
		"1.1.1.1:53",
		"1.0.0.1:53",
		"9.9.9.9:53",
	}

	var conn net.Conn
	var err error

	// Try UDP Dial one by one until successful
	for _, server := range dnsServers {
		conn, err = net.Dial("udp", server)
		if err == nil {
			// Success
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf(
			"could not dial any public DNS to determine default interface: %w",
			err,
		)
	}
	defer func() { _ = conn.Close() }()

	// Check the local IP address used for the connection
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf(
			"could not determine local address from UDP connection",
		)
	}

	// Search for the network interface (iface) that has this local IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get network interfaces: %w",
			err,
		)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue // Skip interfaces whose addresses cannot be retrieved
		}
		for _, addr := range addrs {
			// Check if it is of type net.IPNet
			if ipnet, ok := addr.(*net.IPNet); ok {
				// Check if the IP used for Dial matches the interface's IP
				if ipnet.IP.Equal(localAddr.IP) {
					// Return &iface (*net.Interface) instead of iface.Name (string)
					return &iface, nil // Found the default interface
				}
			}
		}
	}

	return nil, fmt.Errorf(
		"failed to find default interface for local IP: %s",
		localAddr.IP,
	)
}

// GetInterfaceIPv4 finds the first valid (non-loopback) IPv4 address
// on a given interface.
func GetInterfaceIPv4(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil {
				if !ip.IsLoopback() {
					return ip, nil
				}
			}
		}
	}
	return nil, errors.New("no non-loopback IPv4 address found on interface")
}
