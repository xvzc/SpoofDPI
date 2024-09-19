package handler

import (
	"context"
	"io"
	"net"
	"regexp"
	"strconv"
	"sync"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/packet"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

type HttpsHandler struct {
	bufferSize      int
	protocol        string
	port            int
	timeout         int
	windowsize      int
	exploit         bool
	allowedPatterns []*regexp.Regexp
}

func NewHttpsHandler(timeout int, windowSize int, allowedPatterns []*regexp.Regexp, exploit bool) *HttpsHandler {
	return &HttpsHandler{
		bufferSize:      1024,
		protocol:        "HTTPS",
		port:            443,
		timeout:         timeout,
		windowsize:      windowSize,
		allowedPatterns: allowedPatterns,
		exploit:         exploit,
	}
}

func (h *HttpsHandler) Serve(ctx context.Context, lConn *net.TCPConn, initPkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	// Create a connection to the requested server
	var err error
	if initPkt.Port() != "" {
		h.port, err = strconv.Atoi(initPkt.Port())
		if err != nil {
			logger.Debug().Msgf("error parsing port for %s aborting..", initPkt.Domain())
		}
	}

	rConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(ip), Port: h.port})
	if err != nil {
		lConn.Close()
		logger.Debug().Msgf("%s", err)
		return
	}

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), initPkt.Domain())

	_, err = lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.Debug().Msgf("error sending 200 connection established to the client: %s", err)
		return
	}

	logger.Debug().Msgf("sent connection established to %s", lConn.RemoteAddr())

	// Read client hello
	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		logger.Debug().Msgf("error reading client hello from %s: %s", lConn.RemoteAddr().String(), err)
		return
	}
	clientHello := m.Raw

	logger.Debug().Msgf("client sent hello %d bytes", len(clientHello))

	// Generate a go routine that reads from the server
	closeWg := sync.WaitGroup{}
	closeWg.Add(2)
	done := make(chan struct{})
	doneOnce := sync.Once{}
	closeDoneFunc := func() {
		close(done)
	}
	go func() {
		defer closeWg.Done()
		h.communicate(ctx, rConn, lConn, initPkt.Domain(), lConn.RemoteAddr().String())
	}()
	go func() {
		defer closeWg.Done()
		h.communicate(ctx, lConn, rConn, lConn.RemoteAddr().String(), initPkt.Domain())
	}()
	go func(wg *sync.WaitGroup) {
		wg.Wait()
		doneOnce.Do(closeDoneFunc)
	}(&closeWg)

	if h.exploit {
		logger.Debug().Msgf("writing chunked client hello to %s", initPkt.Domain())
		chunks := splitInChunks(ctx, clientHello, h.windowsize)
		if _, err := writeChunks(rConn, chunks); err != nil {
			logger.Debug().Msgf("error writing chunked client hello to %s: %s", initPkt.Domain(), err)
			doneOnce.Do(closeDoneFunc)
		}
	} else {
		logger.Debug().Msgf("writing plain client hello to %s", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			logger.Debug().Msgf("error writing plain client hello to %s: %s", initPkt.Domain(), err)
			doneOnce.Do(closeDoneFunc)
		}
	}
	// wait while conn handling routines stop, then close both connections
	// current routine will sleep until conn handling routines works
	<-done
	_ = lConn.Close()
	_ = rConn.Close()

	logger.Debug().Msgf("closing proxy connection: %s -> %s", lConn.RemoteAddr(), initPkt.Domain())
}

func (h *HttpsHandler) communicate(ctx context.Context, from *net.TCPConn, to *net.TCPConn, fd string, td string) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	buf := make([]byte, h.bufferSize)
	for {
		err := setConnectionTimeout(from, h.timeout)
		if err != nil {
			logger.Debug().Msgf("error while setting connection deadline for %s: %s", fd, err)
		}

		bytesRead, err := ReadBytes(ctx, from, buf)
		if err != nil {
			if len(bytesRead) > 0 {
				h.write(logger, td, to, bytesRead)
			}
			logger.Debug().Msgf("error reading from %s: %s", fd, err)
			return
		}

		if !h.write(logger, td, to, bytesRead) {
			return
		}
	}
}

func (h *HttpsHandler) write(logger zerolog.Logger, connName string, to io.Writer, bytes []byte) (ok bool) {
	ok = true
	if _, err := to.Write(bytes); err != nil {
		logger.Debug().Msgf("error writing to %s", connName)
		ok = false
	}
	return
}

func splitInChunks(ctx context.Context, bytes []byte, size int) [][]byte {
	logger := log.GetCtxLogger(ctx)

	var chunks [][]byte
	var raw []byte = bytes

	logger.Debug().Msgf("window-size: %d", size)

	if size > 0 {
		for {
			if len(raw) == 0 {
				break
			}

			// necessary check to avoid slicing beyond
			// slice capacity
			if len(raw) < size {
				size = len(raw)
			}

			chunks = append(chunks, raw[0:size])
			raw = raw[size:]
		}

		return chunks
	}

	// When the given window-size <= 0

	if len(raw) < 1 {
		return [][]byte{raw}
	}

	logger.Debug().Msg("using legacy fragmentation")

	return [][]byte{raw[:1], raw[1:]}
}

func writeChunks(conn *net.TCPConn, c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			return 0, nil
		}

		total += b
	}

	return total, nil
}
