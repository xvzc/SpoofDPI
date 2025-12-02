package desync

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"net"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

// TLSSplit splits the data into chunks and optionally
// disorders packets by manipulating TTL.
type TLSSplit struct {
	disorder   bool
	defaultTTL uint8
	windowSize uint8
}

func NewTLSSplit(
	disorder bool,
	defaultTTL uint8,
	windowSize uint8,
) TLSDesyncer {
	return &TLSSplit{
		disorder:   disorder,
		defaultTTL: defaultTTL,
		windowSize: windowSize,
	}
}

func (s *TLSSplit) Send(
	ctx context.Context,
	logger zerolog.Logger,
	conn net.Conn,
	msg *proto.TLSMessage,
) (int, error) {
	chunks := split(msg.Raw, int(s.windowSize))

	if s.disorder {
		return s.sendSegmentsDisorder(conn, logger, chunks)
	}

	return s.sendSegments(conn, chunks)
}

// sendSegments sends the segmented Client Hello sequentially.
func (s *TLSSplit) sendSegments(conn net.Conn, chunks [][]byte) (int, error) {
	total := 0
	for _, chunk := range chunks {
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
func (s *TLSSplit) sendSegmentsDisorder(
	conn net.Conn,
	logger zerolog.Logger,
	segments [][]byte,
) (int, error) {
	var isIPv4 bool
	if tcpAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		isIPv4 = tcpAddr.IP.To4() != nil
	}

	var ttlErrored bool
	setTTLWrap := func(ttl uint8) {
		if err := setTTL(conn, isIPv4, ttl); err != nil {
			logger.Warn().Err(err).Msg("failed to set TTL, continuing without modifying ttl")
			ttlErrored = true
		}
	}

	defer setTTLWrap(s.defaultTTL) // Restore the default TTL on return

	disorderBits := genPatternMask()
	logger.Trace().
		Int("chunks", len(segments)).
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
			setTTLWrap(s.defaultTTL)
		}

		curBit = bits.RotateLeft64(curBit, 1)
	}

	return total, nil
}

func split(data []byte, size int) [][]byte {
	if len(data) == 0 {
		return [][]byte{}
	}

	if size == 0 {
		return [][]byte{data}
	}

	capacity := (len(data) + size - 1) / size
	chunks := make([][]byte, 0, capacity)

	for len(data) > 0 {
		n := min(len(data), size)
		chunks = append(chunks, data[:n])
		data = data[n:]
	}
	return chunks
}

func (s *TLSSplit) String() string {
	return "chunk"
}

// --- Helper Functions (Low-level Syscall) ---

// setTTL configures the TTL or Hop Limit depending on the IP version.
func setTTL(conn net.Conn, isIPv4 bool, ttl uint8) error {
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

	// Invoke Control to manipulate file descriptor directly
	err = rawConn.Control(func(fd uintptr) {
		sysErr = syscall.SetsockoptInt(int(fd), level, opt, int(ttl))
	})
	if err != nil {
		return err
	}

	return sysErr
}

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
	ret |= uint64(0b00000001)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed%5)+2))

	// Block 1 [8-15 bits]:
	// Place 2 bits randomly within this byte using the mutated seed.
	seed ^= (seed >> 13)
	ret |= uint64(bits.RotateLeft8(0b10000000, int(seed))) << 8
	seed ^= (seed << 11)
	ret |= uint64(bits.RotateLeft8(0b10000000, -int(seed%7)+1)) << 8

	// Block 2 [16-23 bits]:
	seed ^= (seed >> 17)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 16

	// Block 3 [24-31 bits]:
	seed ^= (seed << 5)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 24

	// Block 4 [32-39 bits]:
	seed ^= (seed >> 12)
	ret |= uint64(bits.RotateLeft8(0b00000001, int(seed))) << 32

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
