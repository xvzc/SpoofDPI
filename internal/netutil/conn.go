package netutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

// bufferPool is a package-level pool of 32KB buffers used by io.CopyBuffer
// to reduce memory allocations and GC pressure in the tunnel hot path.
var bufferPool = sync.Pool{
	New: func() any {
		// We allocate a pointer to a byte slice.
		// 32KB is the default buffer size for io.Copy.
		b := make([]byte, 32*1024)
		return &b
	},
}

func TunnelConns(
	ctx context.Context,
	logger zerolog.Logger,
	errCh chan<- error,
	dst net.Conn, // Destination connection (io.Writer)
	src net.Conn, // Source connection (io.Reader)
) {
	var n int64
	logger = logging.WithLocalScope(ctx, logger, "tunnel")

	var once sync.Once
	closeOnce := func() {
		once.Do(func() {
			CloseConns(src, dst)
		})
	}

	stop := context.AfterFunc(ctx, closeOnce)

	defer func() {
		stop()
		closeOnce()

		logger.Trace().
			Int64("len", n).
			Str("route", fmt.Sprintf("%s -> %s", src.RemoteAddr(), dst.RemoteAddr())).
			Msgf("done")
	}()

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)

	// Copy data from src to dst using the borrowed buffer.
	n, err := io.CopyBuffer(dst, src, *bufPtr)
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) &&
		!errors.Is(err, syscall.EPIPE) {
		errCh <- err
		return
	}

	errCh <- nil
}

// CloseConns safely closes one or more io.Closer (like net.Conn).
// It is nil-safe and intentionally ignores errors from Close(),
// which is a common pattern in defer statements where handling the
// error is not feasible or desired.
func CloseConns(closers ...io.Closer) {
	for _, c := range closers {
		if c != nil {
			// Intentionally ignore the error.
			_ = c.Close()
		}
	}
}

// SetTTL configures the TTL or Hop Limit depending on the IP version.
func SetTTL(conn net.Conn, isIPv4 bool, ttl uint8) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("failed to cast to TCPConn")
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var level, opt int
	if isIPv4 {
		level = syscall.IPPROTO_IP
		opt = syscall.IP_TTL
	} else {
		level = syscall.IPPROTO_IPV6
		opt = syscall.IPV6_UNICAST_HOPS
	}

	var sysErr error

	// Invoke Control to manipulate file descriptor directly
	err = rawConn.Control(func(fd uintptr) {
		sysErr = syscall.SetsockoptInt(int(fd), level, opt, int(ttl))
	})
	if err != nil {
		return err
	}

	return sysErr
}
