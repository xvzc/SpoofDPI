package proxy

import (
	"errors"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	BufferSize   = 1024
	TLSHeaderLen = 5
)

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

func Serve(from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int) {
	defer func() {
		from.Close()
		to.Close()

		log.Debugf("%s closing proxy connection: %s -> %s", proto, fd, td)
	}()

	proto += " "
	buf := make([]byte, BufferSize)
	for {
		if timeout > 0 {
			from.SetReadDeadline(
				time.Now().Add(time.Millisecond * time.Duration(timeout)),
			)
		}

		bytesRead, err := ReadBytes(from, buf)
		if err != nil {
			if err == io.EOF {
				log.Debugf("%s finished reading from %s", proto, fd)
				return
			}
			log.Debugf("%s error reading from %s: %s", proto, fd, err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			log.Debugf("%s error Writing to %s", proto, td)
			return
		}
	}
}
