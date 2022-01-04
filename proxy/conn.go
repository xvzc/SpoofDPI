package proxy

import (
	"net"

	"github.com/xvzc/SpoofDPI/util"
)

const BUF_SIZE = 1024

func ReadBytes(conn net.Conn) ([]byte, error) {
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

func Serve(from net.Conn, to net.Conn, proto string) {
	for {
		buf, err := ReadBytes(from)
		if err != nil {
			util.Debug("["+proto+"]"+"Error reading from ", from.RemoteAddr())
			util.Debug(err, " Closing the connection.. ")
			break
		}

		util.Debug(from.RemoteAddr(), "sent data", len(buf))

		_, write_err := to.Write(buf)
		if write_err != nil {
			util.Debug("["+proto+"]"+"Error reading from ", to.RemoteAddr())
			util.Debug(err, " Closing the connection.. ")
			break
		}
	}
}
