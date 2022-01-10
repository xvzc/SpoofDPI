package net

import (
	"net"
)

func Listen(network, address string) (Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return Listener{}, err
	}

	return Listener{Listener: l}, nil
}

func Dial(network, address string) (Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return Conn{}, err
	}

	return Conn{Conn: conn}, nil
}
