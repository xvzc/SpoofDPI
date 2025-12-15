package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

var errBlocked = errors.New("request blocked")

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

type dialResult struct {
	conn net.Conn
	err  error
}

// dialFirstSuccessful attempts robust connections to the server
// and returns the first successful conn. All the other connections will be
// automatically canceled by calling `cancel()`
func dialFirstSuccessful(
	ctx context.Context,
	addrs []net.IPAddr,
	port int,
	timeout time.Duration,
) (net.Conn, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses provided to dial")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan dialResult, len(addrs))

	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency) // semaphore

	go func() {
		for _, addr := range addrs {
			// Get semaphore
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}

			go func(ip net.IP) {
				defer func() { <-sem }() // Return semaphore

				targetAddr := net.JoinHostPort(ip.String(), strconv.Itoa(port))
				dialer := &net.Dialer{}
				if timeout > 0 {
					dialer.Deadline = time.Now().Add(timeout)
				}

				conn, err := dialer.DialContext(ctx, "tcp", targetAddr)

				select {
				case results <- dialResult{conn: conn, err: err}:
				case <-ctx.Done():
					if conn != nil {
						_ = conn.Close() // Close on context cancel
					}
				}
			}(addr.IP)
		}
	}()

	var firstError error
	failureCount := 0

	for range addrs {
		select {
		case res := <-results:
			if res.err == nil {
				return res.conn, nil
			}
			if firstError == nil {
				firstError = res.err
			}
			failureCount++
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf(
		"all connection attempts failed (total %d): %w",
		failureCount,
		firstError,
	)
}

func tunnelConns(
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
			closeConns(src, dst)
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

// closeConns safely closes one or more io.Closer (like net.Conn).
// It is nil-safe and intentionally ignores errors from Close(),
// which is a common pattern in defer statements where handling the
// error is not feasible or desired.
func closeConns(closers ...io.Closer) {
	for _, c := range closers {
		if c != nil {
			// Intentionally ignore the error.
			_ = c.Close()
		}
	}
}

func isConnectionResetByPeer(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr syscall.Errno
		if errors.As(opErr.Err, &sysErr) {
			return sysErr == syscall.ECONNRESET
		}
	}

	return false
}
