package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

type chunkingFunc func(bytes []byte, size int) [][]byte

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

type HTTPSHandler struct {
	windowSize uint16
	logger     zerolog.Logger
}

func NewHttpsHandler(
	windowSize uint16,
	logger zerolog.Logger,
) *HTTPSHandler {
	return &HTTPSHandler{
		windowSize: windowSize,
		logger:     logger,
	}
}

func (h *HTTPSHandler) Serve(
	ctx context.Context,
	lConn net.Conn,
	req *HttpRequest,
	domain string,
	dstAddrs []net.IPAddr,
	dstPort int,
	timeout time.Duration,
) {
	logger := h.logger.With().Ctx(ctx).Logger()

	// We are responsible for the client connection, so we must close it when done.
	defer closeConns(lConn)

	rConn, err := dialFirstSuccessful(ctx, dstAddrs, dstPort, timeout)
	if err != nil {
		logger.Debug().
			Msgf("dial to %s failed: %s", domain, err)

		return
	}

	// The remote connection must be closed as soon as it's successfully dialed.
	defer closeConns(rConn)

	logger.Debug().
		Msgf("new conn to the server %s -> %s", rConn.LocalAddr(), domain)

	// Send "200 Connection Established" to the client.
	_, err = lConn.Write([]byte(req.Proto + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.Debug().Msgf("error sending 200 conn established to the client: %s", err)
		return // Both connections are closed by their defers.
	}

	logger.Debug().Msgf("sent a conn established to %s", lConn.RemoteAddr())

	// Read the client hello, which is specific to SpoofDPI logic.
	tlsMsg, err := readTLSMessage(lConn)
	if err != nil {
		logger.Debug().Msgf("error reading client hello from %s: %s",
			lConn.RemoteAddr().String(), err,
		)

		return
	}

	if !tlsMsg.IsClientHello() {
		logger.Debug().Msgf("received non-client-hello message from %s, skipping..",
			lConn.RemoteAddr().String(),
		)

		return
	}

	clientHello := tlsMsg.Raw
	logger.Debug().Msgf("client sent hello %d bytes", len(clientHello))

	patternMatched, ok := appctx.PatternMatchedFrom(ctx)
	if !ok {
		logger.Debug().Msg("failed to retrieve patternMatched value from ctx")
	}

	shouldExploit := patternMatched

	logger.Debug().
		Msgf("value of 'shouldExploit' is %s", strconv.FormatBool(shouldExploit))

	// The Client Hello must be sent to the server before starting the
	// bidirectional copy tunnel.
	if shouldExploit {
		logger.Debug().Msgf("writing chunked client hello to %s", domain)

		var cFunc chunkingFunc
		if h.windowSize == 0 {
			logger.Debug().Msgf("using legacy fragmentation strategy")
			cFunc = legacyFragmentationStrategy
		} else {
			logger.Debug().Msgf("using modern fragmentation strategy")
			cFunc = modernFragmentaionStrategy
		}

		if _, err := writeChunks(rConn, cFunc(clientHello, int(h.windowSize))); err != nil {
			logger.Debug().Msgf("error writing chunked client hello to %s: %s",
				domain, err,
			)
			return
		}
	} else {
		logger.Debug().Msgf("writing plain client hello to %s", domain)
		if _, err := rConn.Write(clientHello); err != nil {
			logger.Debug().Msgf("error writing plain client hello to %s: %s", domain, err)
			return
		}
	}

	// Start the tunnel using the refactored helper function.
	go h.tunnel(ctx, rConn, lConn, domain, true)
	h.tunnel(ctx, lConn, rConn, domain, false)
}

// tunnel handles the bidirectional io.Copy between the client and server.
func (h *HTTPSHandler) tunnel(
	ctx context.Context,
	dst net.Conn, // Renamed for io.Copy clarity (Destination)
	src net.Conn, // Renamed for io.Copy clarity (Source)
	domain string,
	closeOnReturn bool,
) {
	logger := h.logger.With().Ctx(ctx).Logger()

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
			logger.Debug().Msgf("error while copying data from %s to %s: %s",
				src.RemoteAddr().String(), dst.RemoteAddr().String(), err,
			)
		}
	}

	if n > 0 {
		logger.Debug().Msgf("copied %d bytes from %s to %s",
			n, src.RemoteAddr().String(), dst.RemoteAddr().String(),
		)
	}

	logger.Debug().Msgf("closing tunnel %s -> %s for %s",
		src.RemoteAddr().String(), dst.RemoteAddr().String(), domain,
	)
}

func modernFragmentaionStrategy(
	bytes []byte,
	size int,
) [][]byte {
	if len(bytes) == 0 {
		return nil
	}

	var chunks [][]byte
	raw := bytes
	for len(raw) != 0 {
		currentSize := min(size, len(raw))
		chunks = append(chunks, raw[0:currentSize])
		raw = raw[currentSize:]
	}
	return chunks
}

// legacyFragmentationStrategy implements the "legacy" strategy (1 byte, then the rest)
// and matches the chunkingFunc type.
func legacyFragmentationStrategy(
	bytes []byte,
	_ int,
) [][]byte {
	if len(bytes) == 0 {
		return nil // No bytes to send.
	}

	if len(bytes) == 1 {
		return [][]byte{bytes[:1]}
	}

	return [][]byte{bytes[:1], bytes[1:]}
}

func writeChunks(conn net.Conn, c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			// Return the actual error.
			return total, err
		}
		total += b
	}

	return total, nil
}
