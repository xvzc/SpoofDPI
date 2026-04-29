package netutil

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type Destination struct {
	Domain  string
	Addrs   []net.IP
	Port    int
	Timeout time.Duration
}

func (d *Destination) String() string {
	return net.JoinHostPort(d.Domain, strconv.Itoa(d.Port))
}

func (d *Destination) IsValid(listenAddr *net.TCPAddr) (bool, error) {
	if d.Port != listenAddr.Port {
		return true, nil
	}

	ifAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return false, err
	}

	for _, dstAddr := range d.Addrs {
		ip := dstAddr
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

	return true, nil
}
