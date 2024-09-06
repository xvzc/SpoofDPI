package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

const (
	BufferSize   = 1024
	TLSHeaderLen = 5
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

func Serve(ctx context.Context, from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int) {
	ctx = util.GetCtxWithScope(ctx, proto)
	logger := log.GetCtxLogger(ctx)

	defer func() {
		from.Close()
		to.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	buf := make([]byte, BufferSize)
	for {
		if timeout > 0 {
			from.SetReadDeadline(
				time.Now().Add(time.Millisecond * time.Duration(timeout)),
			)
		}

		bytesRead, err := ReadBytes(ctx, from, buf)
		if err != nil {
			if err == io.EOF {
				logger.Debug().Msgf("finished reading from %s", fd)
				return
			}
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		if _, err := to.Write(bytesRead); err != nil {
			logger.Debug().Msgf("error writing to %s", td)
			return
		}
	}
}
