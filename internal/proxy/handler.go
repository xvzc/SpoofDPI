package proxy

import (
	"context"
	"fmt"
	"net"
	"time"
)

type Handler interface {
	Serve(
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
