package proxy

import (
	"errors"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const BUF_SIZE = 1024

func WriteChunks(conn *net.TCPConn, c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			return 0, nil
		}

		total += b
	}

	return total, nil
}

func ReadBytes(conn *net.TCPConn) ([]byte, error) {
	ret := make([]byte, 0)
	buf := make([]byte, BUF_SIZE)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			switch err.(type) {
			case *net.OpError:
				return nil, errors.New("timed out")
			default:
				return nil, err
			}
		}
		ret = append(ret, buf[:n]...)

		if n < BUF_SIZE {
			break
		}
	}

	if len(ret) == 0 {
		return nil, io.EOF
	}

	return ret, nil
}

func Serve(from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int) {
	proto += " "

	for {
		from.SetReadDeadline(
			time.Now().Add(time.Millisecond * time.Duration(timeout)),
		)

		buf, err := ReadBytes(from)
		if err != nil {
			if err == io.EOF {
				log.Debug(proto, "Finished ", fd)
				return
			}
			log.Debug(proto, "Error reading from ", fd, " ", err)
			return
		}

		if _, err := to.Write(buf); err != nil {
			log.Debug(proto, "Error Writing to ", td)
			return
		}
	}
}
