package net

import (
	"net"
)

type TCPAddr struct {
	Addr *net.TCPAddr
}

func TcpAddr(ip string, port int) *TCPAddr {
	addr := &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	return &TCPAddr{
		Addr: addr,
	}
}
