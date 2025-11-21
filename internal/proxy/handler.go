package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
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

type Handler interface {
	HandleRequest(
		ctx context.Context,
		lConn net.Conn,
		req *HttpRequest,
		domain string,
		dstAddrs []net.IPAddr,
		dstPort int,
		timeout time.Duration,
	)
}

// dialFirstSuccessful attempts to connect to a list of IP addresses,
// returning the first successful connection.
// It respects the context for cancellation and the dialer's timeout.
// It does not handle logging internally; the caller is responsible for logging.
func dialFirstSuccessful(
	ctx context.Context,
	addrs []net.IPAddr,
	port int,
	timeout time.Duration,
) (net.Conn, error) {
	// Create a dialer with the specified connection timeout.
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	var rConn net.Conn
	var lastErr error

	for _, addr := range addrs {
		targetAddr := (&net.TCPAddr{IP: addr.IP, Port: port}).String()

		// Attempt to dial using the context.
		conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
		if err == nil {
			// Connection successful.
			rConn = conn
			break
		}

		// Store the last error for reporting.
		lastErr = err
	}

	if rConn == nil {
		// All attempts failed.
		// Return a clear error if all attempts failed.
		if lastErr == nil {
			// This should theoretically not be hit if addrs is not empty,
			// but as a safeguard:
			return nil, fmt.Errorf("no addresses provided to dial")
		}
		return nil, fmt.Errorf("all connection attempts failed: %w", lastErr)
	}

	return rConn, nil
}

func tunnel(
	_ context.Context,
	logger zerolog.Logger,
	errCh chan<- error,
	dst net.Conn, // Destination connection (io.Writer)
	src net.Conn, // Source connection (io.Reader)
	domain string,
	closeOnReturn bool,
) {
	// The copy routine is responsible for closing both connections
	// when it finishes, which will unblock the peer copy routine.
	if closeOnReturn {
		defer closeConns(dst, src)
	}

	// Get a buffer from the pool (zero allocation).
	bufPtr := bufferPool.Get().(*[]byte)
	// Ensure the buffer is returned to the pool when the tunnel closes.
	defer bufferPool.Put(bufPtr)

	// Copy data from src to dst using the borrowed buffer.
	n, err := io.CopyBuffer(dst, src, *bufPtr)

	// Check for non-EOF and non-net.ErrClosed errors.
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		// Log the error locally for context.
		logger.Debug().Msgf("error while copying data: %s", err)

		// Report the wrapped error back to the main goroutine.
		// Use %w to wrap the original error for inspection by the caller.
		errCh <- err
	} else {
		errCh <- nil
	}

	if n > 0 {
		logger.Trace().Msgf("copied %d bytes from %s to %s",
			n, src.RemoteAddr().String(), dst.RemoteAddr().String(),
		)
	}

	logger.Trace().Msgf("closing tunnel %s -> %s for %s",
		src.RemoteAddr().String(), dst.RemoteAddr().String(), domain,
	)
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
