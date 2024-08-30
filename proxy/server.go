package proxy

import (
	"bytes"
	"context"
	"errors"
	"github.com/xvzc/SpoofDPI/util"
	"io"
	"net"
	"time"

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
		bytesRead = makeRequestURIRelative(bytesRead)

		if _, err := to.Write(bytesRead); err != nil {
			logger.Debug().Msgf("error Writing to %s", td)
			return
		}
	}
}

var postRequestPrefix = []byte{0x50, 0x4f, 0x53, 0x54, 0x20, 0x68, 0x74, 0x74, 0x70} // "POST http"

func makeRequestURIRelative(data []byte) []byte {
	if !bytes.Equal(data[:9], postRequestPrefix) {
		return data
	}

	var slashCount uint8
	var thirdSlashIdx int

	for i := 9; i < len(data); i++ {
		if data[i] == '/' {
			slashCount++
		}
		if slashCount == 3 {
			thirdSlashIdx = i
			break
		}
	}
	for i := 0; i < 5; i++ {
		data[thirdSlashIdx-5+i] = postRequestPrefix[i]
	}
	return data[thirdSlashIdx-5:]
}
