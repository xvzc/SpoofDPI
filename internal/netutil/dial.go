package netutil

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

type dialResult struct {
	conn net.Conn
	err  error
}

// DialFirstSuccessful attempts robust connections to the server
// and returns the first successful conn. All the other connections will be
// automatically canceled by calling `cancel()`
func DialFirstSuccessful(
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
