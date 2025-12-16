package desync

import (
	"context"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

type TLSDesyncerAttrs struct {
	DefaultTTL uint8
}

// TLSDesyncer splits the data into chunks and optionally
// disorders packets by manipulating TTL.
type TLSDesyncer struct {
	injector   *packet.Injector
	hopTracker *packet.HopTracker
	attrs      *TLSDesyncerAttrs
}

func NewTLSDesyncer(
	injector *packet.Injector,
	hopTracker *packet.HopTracker,
	attrs *TLSDesyncerAttrs,
) *TLSDesyncer {
	return &TLSDesyncer{
		injector:   injector,
		hopTracker: hopTracker,
		attrs:      attrs,
	}
}

func (d *TLSDesyncer) Send(
	ctx context.Context,
	logger zerolog.Logger,
	msg *proto.TLSMessage,
	conn net.Conn,
	httpsOpts *config.HTTPSOptions,
) (int, error) {
	logger = logging.WithLocalScope(ctx, logger, "tls_desync")

	if ptr.FromPtr(httpsOpts.Skip) {
		logger.Trace().Msg("skip desync for this request")
		return d.sendSegments(conn, [][]byte{msg.Raw})
	}

	if d.hopTracker != nil && d.injector != nil && ptr.FromPtr(httpsOpts.FakeCount) > 0 {
		oTTL := d.hopTracker.GetOptimalTTL(conn.RemoteAddr().String())
		n, err := d.sendFakePackets(ctx, conn, oTTL, httpsOpts)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to send fake packets")
		} else {
			logger.Debug().Int("len", n).Uint8("ttl", oTTL).Msg("sent fake packets")
		}
	}

	segments := split(logger, httpsOpts, msg)
	logger.Debug().
		Int("len", len(segments)).
		Str("mode", httpsOpts.SplitMode.String()).
		Msg("segments ready")

	if ptr.FromPtr(httpsOpts.Disorder) {
		return d.sendSegmentsDisorder(conn, logger, segments, httpsOpts)
	}

	return d.sendSegments(conn, segments)
}

// sendSegments sends the segmented Client Hello sequentially.
func (d *TLSDesyncer) sendSegments(conn net.Conn, segments [][]byte) (int, error) {
	total := 0
	for _, chunk := range segments {
		n, err := conn.Write(chunk)
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// sendSegmentsDisorder sends the segmented Client Hello out of order.
// Since performance is prioritized over strict randomness,
// a single 64-bit pattern is generated and reused cyclically
// for sequences exceeding 64 chunks.
func (d *TLSDesyncer) sendSegmentsDisorder(
	conn net.Conn,
	logger zerolog.Logger,
	segments [][]byte,
	opts *config.HTTPSOptions,
) (int, error) {
	var isIPv4 bool
	if tcpAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		isIPv4 = tcpAddr.IP.To4() != nil
	}

	var ttlErrored bool
	setTTLWrap := func(ttl uint8) {
		if err := netutil.SetTTL(conn, isIPv4, ttl); err != nil {
			logger.Warn().Err(err).Msg("failed to set TTL, continuing without modifying ttl")
			ttlErrored = true
		}
	}

	defer setTTLWrap(d.attrs.DefaultTTL) // Restore the default TTL on return

	disorderBits := genPatternMask()
	logger.Debug().
		Str("bits", fmt.Sprintf("%064b", disorderBits)).
		Msgf("disorder ready")
	curBit := uint64(1)
	total := 0
	for _, chunk := range segments {
		if !ttlErrored && disorderBits&curBit == curBit {
			setTTLWrap(1)
		}

		n, err := conn.Write(chunk)
		if err != nil {
			return total, err
		}
		total += n

		if !ttlErrored && disorderBits&curBit == curBit {
			setTTLWrap(d.attrs.DefaultTTL)
		}

		curBit = bits.RotateLeft64(curBit, 1)
	}

	return total, nil
}

func split(
	logger zerolog.Logger,
	attrs *config.HTTPSOptions,
	msg *proto.TLSMessage,
) [][]byte {
	mode := *attrs.SplitMode
	var chunks [][]byte
	var err error
	switch mode {
	case config.HTTPSSplitModeSNI:
		var start, end int
		start, end, err = msg.ExtractSNIOffset()
		if err != nil {
			break
		}
		chunks, err = splitSNI(msg.Raw, start, end)
		logger.Trace().Msgf("extracted SNI is '%s'", msg.Raw[start:end])
	case config.HTTPSSplitModeRandom:
		mask := genPatternMask()
		chunks, err = splitMask(msg.Raw, mask)
	case config.HTTPSSplitModeChunk:
		chunks, err = splitChunks(msg.Raw, int(*attrs.ChunkSize))
	case config.HTTPSSplitModeFirstByte:
		chunks, err = splitFirstByte(msg.Raw)
	case config.HTTPSSplitModeNone:
		return [][]byte{msg.Raw}
	default:
		logger.Debug().Msgf("unsupprted split mode '%s'. proceed without split", mode)
		chunks = [][]byte{msg.Raw}
	}

	if err != nil {
		logger.Debug().Err(err).
			Msgf("error processing split mode '%s', fallback to 'none'", mode)
		chunks = [][]byte{msg.Raw}
	}

	return chunks
}

func splitChunks(raw []byte, size int) ([][]byte, error) {
	lenRaw := len(raw)

	if lenRaw == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if size == 0 {
		return nil, fmt.Errorf("size == 0")
	}

	capacity := (lenRaw + size - 1) / size
	chunks := make([][]byte, 0, capacity)

	for len(raw) > 0 {
		n := min(len(raw), size)
		chunks = append(chunks, raw[:n])
		raw = raw[n:]
	}

	return chunks, nil
}

func splitFirstByte(raw []byte) ([][]byte, error) {
	if len(raw) < 2 {
		return nil, fmt.Errorf("len(raw) is less than 2")
	}

	return [][]byte{raw[:1], raw[1:]}, nil
}

func splitSNI(raw []byte, start, end int) ([][]byte, error) {
	lenRaw := len(raw)

	if lenRaw == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if start > end {
		return nil, fmt.Errorf("invalid start, end pos (start > end)")
	}

	if start < 0 || lenRaw <= start || end < 0 || lenRaw <= end {
		return nil, fmt.Errorf("invalid start, end pos (out of range)")
	}

	segments := make([][]byte, 0, lenRaw)
	segments = append(segments, raw[:start])
	for i := range end - start {
		segments = append(segments, []byte{raw[start+i]})
	}
	segments = append(segments, raw[end:])

	return append([][]byte(nil), segments...), nil
}

func splitMask(raw []byte, mask uint64) ([][]byte, error) {
	lenRaw := len(raw)

	if lenRaw == 0 {
		return nil, fmt.Errorf("empty data")
	}

	segments := make([][]byte, 0, lenRaw)
	start := 0
	curBit := uint64(1)
	for i := range lenRaw {
		if mask&curBit == curBit {
			if i > start {
				segments = append(segments, raw[start:i])
			}
			segments = append(segments, raw[i:i+1])
			start = i + 1
		}

		curBit = bits.RotateLeft64(curBit, 1)
	}

	if lenRaw > start {
		segments = append(segments, raw[start:lenRaw])
	}

	return append([][]byte(nil), segments...), nil
}

func (d *TLSDesyncer) String() string {
	return "split"
}

func (d *TLSDesyncer) sendFakePackets(
	ctx context.Context,
	conn net.Conn,
	oTTL uint8,
	opts *config.HTTPSOptions,
) (int, error) {
	src := conn.LocalAddr().(*net.TCPAddr)
	dst := conn.RemoteAddr().(*net.TCPAddr)

	var totalSent int
	for range *(opts.FakeCount) {
		n, err := d.injector.WriteCraftedPacket(ctx, src, dst, oTTL, opts.FakePacket)
		if err != nil {
			return totalSent, err
		}

		totalSent += n
	}

	return totalSent, nil
}

// --- Helper Functions (Low-level Syscall) ---

// genPatternMask generates a pseudo-random 64-bit mask used for determining
// split points or disorder indices in the packet fragmentation process.
//
// Instead of relying on slow modulo operations or heavy PRNG calls,
// it utilizes a lightweight Xorshift algorithm to mutate the seed for each byte.
// This ensures a high-performance, non-deterministic pattern distribution
// where at least one bit is set in every 8-bit block.
func genPatternMask() uint64 {
	// Initialize the seed using the default PRNG.
	// This is called once per generation, so the cost is negligible.
	seed := rand.Uint()

	var ret uint64 = 0

	// Block 0 [0-7 bits]:
	// Ensure LSB is always 1 to guarantee at least one operation at the start.
	// The second bit is placed randomly within the remaining 7 bits.
	ret |= uint64(0b01010101)

	// Block 1 [8-15 bits]:
	// Place 2 bits randomly within this byte using the mutated seed.
	seed ^= (seed >> 13)
	ret |= uint64(bits.RotateLeft8(0b10000000, int(seed))) << 8
	seed ^= (seed << 11)
	ret |= uint64(bits.RotateLeft8(0b10000000, -int(seed%7)+1)) << 8

	// Block 2 [16-23 bits]:
	seed ^= (seed >> 17)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 16
	// ret |= uint64(0b00000001) << 16
	// ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%3)+2)) << 16
	// ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%4)+4)) << 16

	// Block 3 [24-31 bits]:
	seed ^= (seed << 5)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 24

	// Block 4 [32-39 bits]:
	seed ^= (seed >> 12)
	// ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 32
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%2))) << 32
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%3)+2)) << 32
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%3)+5)) << 32

	// Block 5 [40-47 bits]:
	seed ^= (seed << 25)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 40

	// Block 6 [48-55 bits]:
	seed ^= (seed >> 27)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 48

	// Block 7 [56-63 bits]:
	seed ^= (seed << 13)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 56

	return ret
}
