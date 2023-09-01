package net

import (
	"net"
)

type Listener struct {
	listener *net.TCPListener
}

func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.listener.AcceptTCP()
	if err != nil {
		return &Conn{}, err
	}

	return &Conn{*conn}, nil
}
