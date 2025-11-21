package proxy

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/datastruct/tree"
	"github.com/xvzc/SpoofDPI/internal/packet"
)

var _ Handler = (*HTTPSHandler)(nil)

type fragmentationFunc func(bytes []byte, size int) [][]byte

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
	logger zerolog.Logger

	hopTracker       *packet.HopTracker
	packetInjector   *packet.PacketInjector
	domainSearchTree tree.SearchTree

	autoPolicy       bool
	windowSize       uint8
	fakeHTTPSPackets uint8
}

func NewHttpsHandler(
	logger zerolog.Logger,
	hopTrakcer *packet.HopTracker,
	packetInjector *packet.PacketInjector,
	domainSearchTree tree.SearchTree,

	autoPolicy bool,
	windowSize uint8,
	fakeHTTPSPackets uint8,
) *HTTPSHandler {
	return &HTTPSHandler{
		logger:           logger,
		hopTracker:       hopTrakcer,
		packetInjector:   packetInjector,
		domainSearchTree: domainSearchTree,
		autoPolicy:       autoPolicy,
		windowSize:       windowSize,
		fakeHTTPSPackets: fakeHTTPSPackets,
	}
}

func (h *HTTPSHandler) HandleRequest(
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
		logger.Debug().Msgf("all dial attempts to %s failed: %s", domain, err)

		return
	}

	// The remote connection must be closed as soon as it's successfully dialed.
	defer closeConns(rConn)

	logger.Debug().Msgf("new conn; https; %s -> %s(%s);",
		rConn.LocalAddr(), domain, rConn.RemoteAddr(),
	)

	// Send "200 Connection Established" to the client.
	_, err = lConn.Write(req.ResConnectionEstablished())
	if err != nil {
		logger.Debug().Msgf("error sending 200 conn established to the client: %s", err)
		return // Both connections are closed by their defers.
	}

	logger.Debug().Msgf("connection established sent; %s;", lConn.RemoteAddr())

	// Read the client hello, which is specific to SpoofDPI logic.
	tlsMsg, err := readTLSMessage(lConn)
	if err != nil {
		if err != io.EOF {
			logger.Debug().Msgf("error reading client hello: %s", err)
		}

		return
	}

	if !tlsMsg.IsClientHello() {
		logger.Debug().Msgf("received unknown packet; %s; abort;",
			lConn.RemoteAddr().String(),
		)

		return
	}

	clientHello := tlsMsg.Raw
	logger.Debug().Msgf("received client hello; %d bytes;", len(clientHello))

	shouldExploit, ok := appctx.ShouldExploitFrom(ctx)
	if !ok {
		logger.Error().Msg("error retrieving 'shouldExploit' value from ctx")
	}

	// The Client Hello must be sent to the server before starting the
	// bidirectional copy tunnel.
	if shouldExploit {
		// ┌──────────────────┐
		// │ SEND FAKE_PACKET │
		// └──────────────────┘
		if h.hopTracker != nil && h.packetInjector != nil {
			src := rConn.LocalAddr().(*net.TCPAddr)
			dst := rConn.RemoteAddr().(*net.TCPAddr)

			nhops := h.hopTracker.GetOptimalTTL(dst.String())
			err := h.packetInjector.WriteCraftedPacket(
				ctx, src, dst, nhops, packet.FakeClientHello, h.fakeHTTPSPackets,
			)
			if err != nil {
				logger.Debug().Msgf("error sending fake packets to %s: %s", domain, err)
			}
		}

		// ┌───────────────────────────┐
		// │ SEND CHUNKED_CLIENT_HELLO │
		// └───────────────────────────┘
		var fragmentationStrategy string
		var cFunc fragmentationFunc
		if h.windowSize == 0 {
			fragmentationStrategy = "legacy"
			cFunc = legacyFragmentationStrategy
			logger.Debug().Msgf("fragmentation strategy; %s;", fragmentationStrategy)
		} else {
			fragmentationStrategy = "chunk"
			cFunc = chunkFragmentationStrategy
			logger.Debug().Msgf("fragmentation strategy; %s;", fragmentationStrategy)
		}

		if _, err := writeChunks(rConn, cFunc(clientHello, int(h.windowSize))); err != nil {
			logger.Debug().Msgf("error writing chunked client hello to %s: %s",
				domain, err,
			)
			return
		}

		logger.Debug().Msgf("client hello sent; strategy=%s; dst=%s",
			fragmentationStrategy, domain,
		)
	} else {
		if _, err := rConn.Write(clientHello); err != nil {
			logger.Debug().Msgf("error writing plain client hello to %s: %s", domain, err)
			return
		}

		logger.Debug().Msgf("client hello sent; strategy=plain; dst=%s", domain)
	}

	// Start the tunnel using the refactored helper function.
	errCh := make(chan error, 1)
	go tunnel(ctx, logger, nil, rConn, lConn, domain, true)
	go tunnel(ctx, logger, errCh, lConn, rConn, domain, false)

	err = <-errCh
	// Handle the error from the goroutine
	if err != nil {
		// Auto Policy Logic (copied from user query)
		if h.autoPolicy && h.domainSearchTree != nil {
			// Check if the domain is already known
			_, found := h.domainSearchTree.Search(domain)
			if !found {
				// Insert the domain as blocked
				h.domainSearchTree.Insert(domain, true)
				logger.Info().Msgf("auto policy; name=%s; added;", domain)
			}
		}
	}
}

func chunkFragmentationStrategy(
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
	for i := range c {
		b, err := conn.Write(c[i])
		if err != nil {
			// Return the actual error.
			return total, err
		}

		total += b
	}

	return total, nil
}
