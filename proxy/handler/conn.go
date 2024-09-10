package handler

import (
	"net"
	"time"
)

func setConnectionTimeout(conn *net.TCPConn, timeout int) error {
	if timeout <= 0 {
		return nil
	}

	return conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
}
