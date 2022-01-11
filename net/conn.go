package net

import (
	"net"

	log "github.com/sirupsen/logrus"
)

const BUF_SIZE = 1024

type Conn struct {
	Conn net.Conn
}

func (conn *Conn) Close() {
	conn.Conn.Close()
}

func (conn *Conn) RemoteAddr() net.Addr {
	return conn.Conn.RemoteAddr()
}

func (conn *Conn) LocalAddr() net.Addr {
	return conn.Conn.LocalAddr()
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
			log.Debug("["+proto+"]"+" Error reading from ", from.RemoteAddr())
			log.Debug("["+proto+"]", err)
			log.Debug("[" + proto + "]" + " Exiting Serve() method. ")
			break
		}
		log.Debug(from.RemoteAddr(), " sent data: ", len(buf), "bytes")

		if _, err := to.Write(buf); err != nil {
			log.Debug("["+proto+"]"+"Error Writing to ", to.RemoteAddr())
			log.Debug("["+proto+"]", err)
			log.Debug("[" + proto + "]" + " Exiting Serve() method. ")
			break
		}
	}
}
