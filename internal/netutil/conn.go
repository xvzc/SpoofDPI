package netutil

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

type TunnelDirType int

const (
	TunnelDirOut TunnelDirType = iota
	TunnelDirIn
)

// TransferResult holds the result of a unidirectional tunnel transfer.
type TransferResult struct {
	Written int64
	Dir     TunnelDirType
	Err     error
}

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

// TunnelConns copies data from src to dst.
// It sends the result to resCh upon completion.
// It filters out benign errors like EOF, pipe closed, or read timeouts (for UDP).
func TunnelConns(
	ctx context.Context,
	resCh chan<- TransferResult,
	src net.Conn, // Source connection (io.Reader)
	dst net.Conn, // Destination connection (io.Writer)
	dir TunnelDirType,
) {
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
	}()

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)

	// Copy data from src to dst using the borrowed buffer.
	n, err := io.CopyBuffer(dst, src, *bufPtr)

	// Filter benign errors.
	// os.IsTimeout is checked to handle UDP idle timeouts gracefully.
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) &&
		!errors.Is(err, syscall.EPIPE) && !os.IsTimeout(err) {
		resCh <- TransferResult{Written: n, Dir: dir, Err: err}
		return
	}

	resCh <- TransferResult{Written: n, Dir: dir, Err: nil}
}

// WaitAndLogTunnel aggregates results and logs the summary.
// errHandler processes the list of errors to determine the final error.
func WaitAndLogTunnel(
	ctx context.Context,
	logger zerolog.Logger,
	resCh <-chan TransferResult,
	startedAt time.Time,
	route string,
	errHandler func(errs []error) error, // [Modified] Accepts slice handler
) error {
	var (
		outBytes int64
		inBytes  int64
		errs     []error // Collect all errors
	)

	// Wait for exactly 2 results.
	for range 2 {
		res := <-resCh

		switch res.Dir {
		case TunnelDirOut:
			outBytes = res.Written
		case TunnelDirIn:
			inBytes = res.Written
		default:
			return fmt.Errorf("invalid tunnel dir")
		}

		if res.Err != nil {
			errs = append(errs, res.Err)
		}
	}

	duration := float64(time.Since(startedAt).Microseconds()) / 1000.0
	logger.Trace().
		Int64("out", outBytes).
		Int64("in", inBytes).
		Str("took", fmt.Sprintf("%.3fms", duration)).
		Str("route", route).
		Int("errs", len(errs)).
		Msg("tunnel closed")

	if errHandler != nil {
		return errHandler(errs)
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func DescribeRoute(src, dst net.Conn) string {
	return fmt.Sprintf("%s(%s) -> %s(%s)",
		src.RemoteAddr(),
		src.RemoteAddr().Network(),
		dst.RemoteAddr(),
		dst.RemoteAddr().Network(),
	)
}

// CloseConns safely closes one or more io.Closer (like net.Conn).
// It is nil-safe and intentionally ignores errors from Close(),
// which is a common pattern in defer statements where handling the
// error is not feasible or desired.
func CloseConns(closers ...io.Closer) {
	for _, c := range closers {
		if c != nil {
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

	// Invoke Control to manipulate file descriptor directly.
	err = rawConn.Control(func(fd uintptr) {
		sysErr = syscall.SetsockoptInt(int(fd), level, opt, int(ttl))
	})
	if err != nil {
		return err
	}

	return sysErr
}

// BufferedConn wraps a net.Conn with a bufio.Reader to support peeking.
type BufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func NewBufferedConn(c net.Conn) *BufferedConn {
	return &BufferedConn{
		r:    bufio.NewReader(c),
		Conn: c,
	}
}

func (b *BufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func (b *BufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

// TimeoutConn wraps a net.Conn to update the read deadline on every Read call.
// This is useful for UDP sessions which do not have a natural EOF.
type TimeoutConn struct {
	net.Conn
	Timeout    time.Duration
	LastActive time.Time
	ExpiredAt  time.Time // Calculated expiration time for cleanup
}

func (c *TimeoutConn) Read(b []byte) (int, error) {
	c.ExtendDeadline()
	return c.Conn.Read(b)
}

func (c *TimeoutConn) Write(b []byte) (int, error) {
	c.ExtendDeadline()
	return c.Conn.Write(b)
}

// ExtendDeadline attempts to extend the connection's deadline.
// Returns false if the connection was already expired.
func (c *TimeoutConn) ExtendDeadline() bool {
	now := time.Now()

	// Check if already expired
	if !c.ExpiredAt.IsZero() && now.After(c.ExpiredAt) {
		return false
	}

	c.LastActive = now
	if c.Timeout > 0 {
		c.ExpiredAt = now.Add(c.Timeout)
		_ = c.SetReadDeadline(c.ExpiredAt)
		_ = c.SetWriteDeadline(c.ExpiredAt)
	}
	return true
}
