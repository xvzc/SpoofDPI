package netutil

import (
	"net"
	"sync/atomic"
)

var (
	txBytes uint64
	rxBytes uint64
)

type TrackingConn struct {
	net.Conn
}

func (c *TrackingConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&rxBytes, uint64(n))
	}
	return
}

func (c *TrackingConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&txBytes, uint64(n))
	}
	return
}

func GetRxBytes() uint64 {
	return atomic.LoadUint64(&rxBytes)
}

func GetTxBytes() uint64 {
	return atomic.LoadUint64(&txBytes)
}
