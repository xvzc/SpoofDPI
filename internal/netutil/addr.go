package netutil

import (
	"fmt"
	"net"
)

func ValidateDestination(
	dstAddrs []net.IPAddr,
	dstPort int,
	listenAddr *net.TCPAddr,
) (bool, error) {
	if dstPort != int(listenAddr.Port) {
		return true, nil
	}

	var err error
	var ifAddrs []net.Addr
	ifAddrs, err = net.InterfaceAddrs()

	for _, dstAddr := range dstAddrs {
		ip := dstAddr.IP
		if ip.IsLoopback() {
			return false, fmt.Errorf("loopback addr detected %v", ip.String())
		}

		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(ip) {
					return false, fmt.Errorf("interface addr detected %v", ipnet.String())
				}
			}
		}
	}

	return true, err
}
