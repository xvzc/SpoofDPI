package desync

import (
	"fmt"
	"slices"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func TestSplit(t *testing.T) {
	logger := zerolog.Nop()

	tcs := []struct {
		name   string
		opts   *config.HTTPSOptions
		msg    *proto.TLSMessage
		assert func(t *testing.T, chunks [][]byte)
	}{
		{
			name: "none",
			opts: &config.HTTPSOptions{SplitMode: ptr.FromValue(config.HTTPSSplitModeNone)},
			msg:  &proto.TLSMessage{Raw: []byte("12345")},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.Len(t, chunks, 1)
				assert.Equal(t, []byte("12345"), chunks[0])
			},
		},
		{
			name: "chunk size 3",
			msg:  &proto.TLSMessage{Raw: []byte("1234567890")},
			opts: &config.HTTPSOptions{
				SplitMode: ptr.FromValue(config.HTTPSSplitModeChunk),
				ChunkSize: ptr.FromValue(uint8(3)),
			},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.Len(t, chunks, 4)
				assert.Equal(t, []byte("123"), chunks[0])
				assert.Equal(t, []byte("0"), chunks[3])
			},
		},
		{
			name: "chunk size 0 (fallback)",
			msg:  &proto.TLSMessage{Raw: []byte("1234567890")},
			opts: &config.HTTPSOptions{
				SplitMode: ptr.FromValue(config.HTTPSSplitModeChunk),
				ChunkSize: ptr.FromValue(uint8(0)),
			},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.Len(t, chunks, 1)
				assert.Equal(t, []byte("1234567890"), chunks[0])
			},
		},
		{
			name: "first-byte",
			msg:  &proto.TLSMessage{Raw: []byte("1234567890")},
			opts: &config.HTTPSOptions{
				SplitMode: ptr.FromValue(config.HTTPSSplitModeFirstByte),
			},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.Len(t, chunks, 2)
				assert.Equal(t, []byte("1"), chunks[0])
				assert.Equal(t, []byte("234567890"), chunks[1])
			},
		},
		{
			name: "first-byte (fallback)",
			msg:  &proto.TLSMessage{Raw: []byte("1")},
			opts: &config.HTTPSOptions{
				SplitMode: ptr.FromValue(config.HTTPSSplitModeFirstByte),
			},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.Len(t, chunks, 1)
				assert.Equal(t, []byte("1"), chunks[0])
			},
		},
		{
			name: "valid sni",
			msg:  &proto.TLSMessage{Raw: []byte(config.FakeClientHello)},
			opts: &config.HTTPSOptions{SplitMode: ptr.FromValue(config.HTTPSSplitModeSNI)},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.GreaterOrEqual(t, len(chunks), 1)
				assert.Equal(t, string("www.w3.org"), string(slices.Concat(chunks[1:11]...)))
			},
		},
		{
			name: "sni (fallback)",
			msg:  &proto.TLSMessage{Raw: []byte("1234567890")},
			opts: &config.HTTPSOptions{SplitMode: ptr.FromValue(config.HTTPSSplitModeSNI)},
			assert: func(t *testing.T, chunks [][]byte) {
				// Fallback to no split on error
				assert.Len(t, chunks, 1)
				assert.Equal(t, []byte("1234567890"), chunks[0])
			},
		},
		{
			name: "random",
			msg:  &proto.TLSMessage{Raw: []byte(config.FakeClientHello)},
			opts: &config.HTTPSOptions{
				SplitMode: ptr.FromValue(config.HTTPSSplitModeRandom),
			},
			assert: func(t *testing.T, chunks [][]byte) {
				assert.GreaterOrEqual(t, len(chunks), 1)
				var joined []byte
				for _, c := range chunks {
					joined = append(joined, c...)
				}
				assert.Equal(t, []byte(config.FakeClientHello), joined)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			chunks := split(logger, tc.opts, tc.msg)
			tc.assert(t, chunks)
		})
	}
}

func TestSplitChunks(t *testing.T) {
	tcs := []struct {
		name    string
		raw     []byte
		size    int
		wantErr bool
		expect  [][]byte
	}{
		{
			name:    "size 2",
			raw:     []byte("12345"),
			size:    2,
			wantErr: false,
			expect:  [][]byte{[]byte("12"), []byte("34"), []byte("5")},
		},
		{
			name:    "size larger than len",
			raw:     []byte("123"),
			size:    5,
			wantErr: false,
			expect:  [][]byte{[]byte("123")},
		},
		{
			name:    "size 0",
			raw:     []byte("12345"),
			size:    0,
			wantErr: true,
		},
		{
			name:    "len(raw) is 0",
			raw:     []byte(""),
			size:    3,
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := splitChunks(tc.raw, tc.size)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expect, chunks)
		})
	}
}

func TestSplitFirstByte(t *testing.T) {
	tcs := []struct {
		name    string
		raw     []byte
		wantErr bool
		expect  [][]byte
	}{
		{
			name:    "size 2",
			raw:     []byte("12"),
			wantErr: false,
			expect:  [][]byte{[]byte("1"), []byte("2")},
		},
		{
			name:    "size 3",
			raw:     []byte("123"),
			wantErr: false,
			expect:  [][]byte{[]byte("1"), []byte("23")},
		},
		{
			name:    "size 10",
			raw:     []byte("1234567890"),
			wantErr: false,
			expect:  [][]byte{[]byte("1"), []byte("234567890")},
		},
		{
			name:    "len(data) is 0",
			raw:     []byte(""),
			wantErr: true,
		},
		{
			name:    "len(data) is 1",
			raw:     []byte(""),
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := splitFirstByte(tc.raw)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, chunks, tc.expect)
		})
	}
}

func TestSplitSNI(t *testing.T) {
	tcs := []struct {
		name    string
		raw     []byte
		start   int
		end     int
		wantErr bool
		expect  [][]byte
	}{
		{
			name:    "size 0",
			raw:     []byte("PREFIX_SNI_SUFFIX"),
			start:   7,
			end:     10,
			wantErr: false,
			expect: [][]byte{
				[]byte("PREFIX_"),
				[]byte("S"),
				[]byte("N"),
				[]byte("I"),
				[]byte("_SUFFIX"),
			},
		},
		{
			name:    "start out of range (start > len)",
			raw:     []byte("1"),
			start:   3,
			end:     3,
			wantErr: true,
			expect:  [][]byte{[]byte("1")},
		},
		{
			name:    "start out of range (start < 0)",
			raw:     []byte("1"),
			start:   -1,
			end:     5,
			wantErr: true,
			expect:  [][]byte{[]byte("1")},
		},
		{
			name:    "end out of range (end > len)",
			raw:     []byte("1"),
			start:   0,
			end:     5,
			wantErr: true,
			expect:  [][]byte{[]byte("1")},
		},
		{
			name:    "end out of range (end < 0)",
			raw:     []byte("1"),
			start:   -1,
			end:     -1,
			wantErr: true,
			expect:  [][]byte{[]byte("1")},
		},
		{
			name:    "invalid start, end pos (start > end)",
			raw:     []byte("12345"),
			start:   4,
			end:     3,
			wantErr: true,
			expect:  [][]byte{[]byte("1")},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := splitSNI(tc.raw, tc.start, tc.end)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, chunks, tc.expect)
			// Debug print if fail
			if !assert.Equal(t, tc.expect, chunks) {
				for i, c := range chunks {
					fmt.Printf("chunk[%d]: %s\n", i, c)
				}
			}
		})
	}
}

func TestSplitMask(t *testing.T) {
	tcs := []struct {
		name    string
		raw     []byte
		mask    uint64
		wantErr bool
		expect  [][]byte
	}{
		{
			name:    "mask with some bits set (137 = 10001001)",
			raw:     []byte("12345678"),
			mask:    137, //
			wantErr: false,
			expect: [][]byte{
				[]byte("1"),
				[]byte("23"),
				[]byte("4"),
				[]byte("567"),
				[]byte("8"),
			},
		},
		{
			name:    "mask 0 (no split)",
			raw:     []byte("123"),
			mask:    0,
			wantErr: false,
			expect:  [][]byte{[]byte("123")},
		},
		{
			name:    "len(data) is 0",
			raw:     []byte(""),
			mask:    123,
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := splitMask(tc.raw, tc.mask)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, chunks, tc.expect)
		})
	}
}
