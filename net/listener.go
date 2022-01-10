package net

import (
	"net"
)

type Listener struct {
	Listener net.Listener
}

func (l *Listener) Accept() (Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return Conn{}, err
	}

	return Conn{Conn: conn}, nil
}
