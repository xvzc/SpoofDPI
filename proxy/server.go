package proxy

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/xvzc/SpoofDPI/util/log"
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

func Serve(from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int) {
	defer func() {
		from.Close()
		to.Close()

		log.Logger.Debug().
			Str(log.ScopeFieldName, proto).
			Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

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
				log.Logger.Debug().
					Str(log.ScopeFieldName, proto).
					Msgf("finished reading from %s", fd)
				return
			}
			log.Logger.Debug().
				Str(log.ScopeFieldName, proto).
				Msgf("error reading from %s: %s", fd, err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			log.Logger.Debug().
				Str(log.ScopeFieldName, proto).
				Msgf("error Writing to %s", td)
			return
		}
	}
}
