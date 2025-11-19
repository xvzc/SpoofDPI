package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog"
)

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

// tunnel handles the bidirectional io.Copy between the client and server.
func tunnel(
	_ context.Context,
	logger zerolog.Logger,
	dst net.Conn, // Renamed for io.Copy clarity (Destination)
	src net.Conn, // Renamed for io.Copy clarity (Source)
	domain string,
	closeOnReturn bool,
) {
	// The client-to-server goroutine is responsible for closing both connections
	// when it finishes, which will unblock the server-to-client copy.
	if closeOnReturn {
		defer closeConns(dst, src)
	}

	// Use a buffer from the pool to reduce allocations.
	// 1. Get a buffer from the pool (zero allocation).
	bufPtr := bufferPool.Get().(*[]byte)
	// 2. Ensure the buffer is returned to the pool when the tunnel closes.
	defer bufferPool.Put(bufPtr)

	// 3. Use the borrowed buffer with io.CopyBuffer.
	// This copies from src to dst.
	n, err := io.CopyBuffer(dst, src, *bufPtr)
	if err != nil {
		if !errors.Is(err, net.ErrClosed) && err != io.EOF {
			logger.Debug().Msgf("error while copying data: %s", err)
		}
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
