package net

import (
	"net"
	"strconv"
)

func ListenTCP(network string, addr *TCPAddr) (Listener, error) {
	l, err := net.ListenTCP(network, addr.Addr)
	if err != nil {
		return Listener{}, err
	}

	return Listener{listener: l}, nil
}

func DialTCP(network string, ip string, port string) (*Conn, error) {
	p, _ := strconv.Atoi(port)

	addr := &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: p,
	}

	conn, err := net.DialTCP(network, nil, addr)
	if err != nil {
		return &Conn{}, err
	}

	return &Conn{*conn}, nil
}
