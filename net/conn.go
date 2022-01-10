package net

import (
	"net"
)

const BUF_SIZE = 1024

type Conn struct {
	Conn net.Conn
}

func (conn *Conn) Close() {
	conn.Conn.Close()
}

func (conn *Conn) Read(b []byte) (n int, err error) {
	return conn.Conn.Read(b)
}

func (conn *Conn) Write(b []byte) (n int, err error) {
	return conn.Conn.Write(b)
}

func (conn *Conn) WriteChunks(c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			return 0, nil
		}

		b += total
	}

	return total, nil
}

func (conn *Conn) ReadBytes() ([]byte, error) {
	ret := make([]byte, 0)
	buf := make([]byte, BUF_SIZE)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, err
		}
		ret = append(ret, buf[:n]...)

		if n < BUF_SIZE {
			break
		}
	}

	return ret, nil
}

func (from *Conn) Serve(to Conn, proto string) {
	for {
		buf, err := from.ReadBytes()
		if err != nil {
			// util.Debug("["+proto+"]"+"Error reading from ", from.RemoteAddr())
			// util.Debug(err, " Closing the connection.. ")
			break
		}

		// util.Debug(from.RemoteAddr(), "sent data", len(buf))

		_, write_err := to.Write(buf)
		if write_err != nil {
			// util.Debug("["+proto+"]"+"Error reading from ", to.RemoteAddr())
			// util.Debug(err, " Closing the connection.. ")
			break
		}
	}
}
