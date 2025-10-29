package handler

import (
	"context"
	"errors"
	"net"
	"time"
)

func ReadBytes(ctx context.Context, conn *net.TCPConn, dest []byte) ([]byte, error) {
	n, err := readBytesInternal(conn, dest)
	return dest[:n], err
}

func readBytesInternal(conn *net.TCPConn, dest []byte) (int, error) {
	totalRead, err := conn.Read(dest)
	if err != nil {
		var opError *net.OpError
		switch {
		case errors.As(err, &opError) && opError.Timeout():
			return totalRead, errors.New("timed out")
		default:
			return totalRead, err
		}
	}
	return totalRead, nil
}

func SetConnectionTimeout(conn *net.TCPConn, timeout int) error {
	if timeout <= 0 {
		return nil
	}

	return conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
}
