package handler

import (
	"context"
	"errors"
	"net"
)

func ReadBytes(ctx context.Context, conn *net.TCPConn, dest []byte) ([]byte, error) {
	n, err := readBytesInternal(ctx, conn, dest)
	return dest[:n], err
}

func readBytesInternal(ctx context.Context, conn *net.TCPConn, dest []byte) (int, error) {
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
