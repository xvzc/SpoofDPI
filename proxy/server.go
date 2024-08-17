package proxy

import (
	"errors"
	"io"
	"net"
	"time"
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

		log.Debug("[HTTPS] closing proxy connection: ", fd, " -> ", td)
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
				log.Debug(proto, "finished reading from", fd)
				return
			}
			log.Debug(proto, "error reading from ", fd, " ", err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			log.Debug(proto, "error Writing to ", td)
			return
		}
	}
}
