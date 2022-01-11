package net

import (
	"net"
)

type Listener struct {
	listener net.Listener
}

func (l *Listener) Accept() (Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return Conn{}, err
	}

	return Conn{conn: conn}, nil
}
