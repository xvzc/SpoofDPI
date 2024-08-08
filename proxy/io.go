package proxy

import (
	"errors"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

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

func ReadBytes(conn *net.TCPConn, dest []byte) ([]byte, error) {
	n, err := readBytesInternal(conn, dest)
	return dest[:n], err
}

func readBytesInternal(conn *net.TCPConn, dest []byte) (int, error) {
	totalRead, err := conn.Read(dest)
	if err != nil {
		switch err.(type) {
		case *net.OpError:
			return totalRead, errors.New("timed out")
		default:
			return totalRead, err
		}
	}
	return totalRead, nil
}

func Serve(from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int, bufferSize int) {
	proto += " "
	buf := make([]byte, bufferSize)
	for {
		if timeout > 0 {
			from.SetReadDeadline(
				time.Now().Add(time.Millisecond * time.Duration(timeout)),
			)
		}

		bytesRead, err := ReadBytes(from, buf)
		if err != nil {
			if err == io.EOF {
				log.Debug(proto, "Finished ", fd)
				return
			}
			log.Debug(proto, "Error reading from ", fd, " ", err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			log.Debug(proto, "Error Writing to ", td)
			return
		}
	}
}
