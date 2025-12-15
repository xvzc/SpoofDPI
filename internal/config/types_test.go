package config

import (
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

// ┌─────────────────┐
// │ GENERAL OPTIONS │
// └─────────────────┘
func TestGeneralOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o GeneralOptions)
	}{
		{
			name: "valid general options",
			input: map[string]any{
				"log-level":    "debug",
				"silent":       true,
				"system-proxy": true,
			},
			wantErr: false,
			assert: func(t *testing.T, o GeneralOptions) {
				assert.Equal(t, zerolog.DebugLevel, *o.LogLevel)
				assert.True(t, *o.Silent)
				assert.True(t, *o.SetSystemProxy)
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var o GeneralOptions
			err := o.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, o)
				}
			}
		})
	}
}

func TestGeneralOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *GeneralOptions
		assert func(t *testing.T, input *GeneralOptions, output *GeneralOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *GeneralOptions, output *GeneralOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &GeneralOptions{
				LogLevel: ptr.FromValue(zerolog.DebugLevel),
				Silent:   ptr.FromValue(true),
			},
			assert: func(t *testing.T, input *GeneralOptions, output *GeneralOptions) {
				assert.NotNil(t, output)
				assert.Equal(t, zerolog.DebugLevel, *output.LogLevel)
				assert.True(t, *output.Silent)
				assert.NotSame(t, input, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

func TestGeneralOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *GeneralOptions
		override *GeneralOptions
		assert   func(t *testing.T, output *GeneralOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &GeneralOptions{Silent: ptr.FromValue(true)},
			assert: func(t *testing.T, output *GeneralOptions) {
				assert.True(t, *output.Silent)
			},
		},
		{
			name:     "nil override",
			base:     &GeneralOptions{Silent: ptr.FromValue(false)},
			override: nil,
			assert: func(t *testing.T, output *GeneralOptions) {
				assert.False(t, *output.Silent)
			},
		},
		{
			name: "merge values",
			base: &GeneralOptions{
				Silent:   ptr.FromValue(false),
				LogLevel: ptr.FromValue(zerolog.InfoLevel),
			},
			override: &GeneralOptions{
				Silent: ptr.FromValue(true),
			},
			assert: func(t *testing.T, output *GeneralOptions) {
				assert.True(t, *output.Silent)
				assert.Equal(t, zerolog.InfoLevel, *output.LogLevel)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.base.Merge(tc.override)
			tc.assert(t, output)
		})
	}
}

// ┌────────────────┐
// │ SERVER OPTIONS │
// └────────────────┘
func TestServerOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o ServerOptions)
	}{
		{
			name: "valid server options",
			input: map[string]any{
				"default-ttl": int64(64),
				"listen-addr": "127.0.0.1:8080",
				"timeout":     int64(1000),
			},
			wantErr: false,
			assert: func(t *testing.T, o ServerOptions) {
				assert.Equal(t, uint8(64), *o.DefaultTTL)
				assert.Equal(t, "127.0.0.1:8080", o.ListenAddr.String())
				assert.Equal(t, 1000*time.Millisecond, *o.Timeout)
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var o ServerOptions
			err := o.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, o)
				}
			}
		})
	}
}

func TestServerOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *ServerOptions
		assert func(t *testing.T, input *ServerOptions, output *ServerOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *ServerOptions, output *ServerOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &ServerOptions{
				DefaultTTL: ptr.FromValue(uint8(64)),
				ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
			},
			assert: func(t *testing.T, input *ServerOptions, output *ServerOptions) {
				assert.NotNil(t, output)
				assert.Equal(t, uint8(64), *output.DefaultTTL)
				assert.Equal(t, "127.0.0.1:8080", output.ListenAddr.String())
				assert.NotSame(t, input, output)
				if output.ListenAddr != nil {
					assert.NotSame(t, input.ListenAddr, output.ListenAddr)
				}
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

func TestServerOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *ServerOptions
		override *ServerOptions
		assert   func(t *testing.T, output *ServerOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &ServerOptions{DefaultTTL: ptr.FromValue(uint8(64))},
			assert: func(t *testing.T, output *ServerOptions) {
				assert.Equal(t, uint8(64), *output.DefaultTTL)
			},
		},
		{
			name:     "nil override",
			base:     &ServerOptions{DefaultTTL: ptr.FromValue(uint8(128))},
			override: nil,
			assert: func(t *testing.T, output *ServerOptions) {
				assert.Equal(t, uint8(128), *output.DefaultTTL)
			},
		},
		{
			name: "merge values",
			base: &ServerOptions{
				DefaultTTL: ptr.FromValue(uint8(64)),
				ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
			},
			override: &ServerOptions{
				DefaultTTL: ptr.FromValue(uint8(128)),
			},
			assert: func(t *testing.T, output *ServerOptions) {
				assert.Equal(t, uint8(128), *output.DefaultTTL)
				assert.Equal(t, "127.0.0.1:8080", output.ListenAddr.String())
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.base.Merge(tc.override)
			tc.assert(t, output)
		})
	}
}

// ┌─────────────┐
// │ DNS OPTIONS │
// └─────────────┘
func TestDNSOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o DNSOptions)
	}{
		{
			name: "valid dns options",
			input: map[string]any{
				"mode":      "https",
				"addr":      "8.8.8.8:53",
				"https-url": "https://dns.google/dns-query",
				"qtype":     "ipv4",
				"cache":     true,
			},
			wantErr: false,
			assert: func(t *testing.T, o DNSOptions) {
				assert.Equal(t, DNSModeHTTPS, *o.Mode)
				assert.Equal(t, "8.8.8.8:53", o.Addr.String())
				assert.Equal(t, "https://dns.google/dns-query", *o.HTTPSURL)
				assert.Equal(t, DNSQueryIPv4, *o.QType)
				assert.True(t, *o.Cache)
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var o DNSOptions
			err := o.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, o)
				}
			}
		})
	}
}

func TestDNSOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *DNSOptions
		assert func(t *testing.T, input *DNSOptions, output *DNSOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *DNSOptions, output *DNSOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &DNSOptions{
				Mode: ptr.FromValue(DNSModeHTTPS),
				Addr: &net.TCPAddr{IP: net.ParseIP("1.1.1.1"), Port: 53},
			},
			assert: func(t *testing.T, input *DNSOptions, output *DNSOptions) {
				assert.NotNil(t, output)
				assert.Equal(t, DNSModeHTTPS, *output.Mode)
				assert.NotSame(t, input, output)
				if output.Addr != nil {
					assert.NotSame(t, input.Addr, output.Addr)
				}
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

func TestDNSOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *DNSOptions
		override *DNSOptions
		assert   func(t *testing.T, output *DNSOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &DNSOptions{Mode: ptr.FromValue(DNSModeHTTPS)},
			assert: func(t *testing.T, output *DNSOptions) {
				assert.Equal(t, DNSModeHTTPS, *output.Mode)
			},
		},
		{
			name:     "nil override",
			base:     &DNSOptions{Mode: ptr.FromValue(DNSModeUDP)},
			override: nil,
			assert: func(t *testing.T, output *DNSOptions) {
				assert.Equal(t, DNSModeUDP, *output.Mode)
			},
		},
		{
			name: "merge values",
			base: &DNSOptions{
				Mode: ptr.FromValue(DNSModeUDP),
				Addr: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
			},
			override: &DNSOptions{
				Mode:     ptr.FromValue(DNSModeUDP),
				HTTPSURL: ptr.FromValue("https://dns.google/test"),
			},
			assert: func(t *testing.T, output *DNSOptions) {
				assert.Equal(t, DNSModeUDP, *output.Mode)
				assert.Equal(t, "8.8.8.8:53", output.Addr.String())
				assert.Equal(t, "https://dns.google/test", *output.HTTPSURL)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.base.Merge(tc.override)
			tc.assert(t, output)
		})
	}
}

// ┌───────────────┐
// │ HTTPS OPTIONS │
// └───────────────┘
func TestHTTPSOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o HTTPSOptions)
	}{
		{
			name: "valid https options",
			input: map[string]any{
				"disorder":    true,
				"fake-count":  int64(5),
				"fake-packet": []any{int64(0x01), int64(0x02)},
				"split-mode":  "chunk",
				"chunk-size":  int64(20),
				"skip":        true,
			},
			wantErr: false,
			assert: func(t *testing.T, o HTTPSOptions) {
				assert.True(t, *o.Disorder)
				assert.Equal(t, uint8(5), *o.FakeCount)
				assert.Equal(t, []byte{0x01, 0x02}, o.FakePacket)
				assert.Equal(t, HTTPSSplitModeChunk, *o.SplitMode)
				assert.Equal(t, uint8(20), *o.ChunkSize)
				assert.True(t, *o.Skip)
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var o HTTPSOptions
			err := o.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, o)
				}
			}
		})
	}
}

func TestHTTPSOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *HTTPSOptions
		assert func(t *testing.T, input *HTTPSOptions, output *HTTPSOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *HTTPSOptions, output *HTTPSOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name:  "non-nil receiver",
			input: &HTTPSOptions{Disorder: ptr.FromValue(true), FakePacket: []byte{0x01}},
			assert: func(t *testing.T, input *HTTPSOptions, output *HTTPSOptions) {
				assert.NotNil(t, output)
				assert.True(t, *output.Disorder)
				assert.NotSame(t, input, output)
				if len(output.FakePacket) > 0 {
					assert.Equal(t, input.FakePacket, output.FakePacket)
					// assert.NotSame(t, &input.FakePacket[0], &output.FakePacket[0]) // Cannot easily check slice backing array address safely
				}
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

func TestHTTPSOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *HTTPSOptions
		override *HTTPSOptions
		assert   func(t *testing.T, output *HTTPSOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &HTTPSOptions{Disorder: ptr.FromValue(true)},
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.True(t, *output.Disorder)
			},
		},
		{
			name:     "nil override",
			base:     &HTTPSOptions{Disorder: ptr.FromValue(false)},
			override: nil,
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.False(t, *output.Disorder)
			},
		},
		{
			name: "merge values",
			base: &HTTPSOptions{
				Disorder:   ptr.FromValue(false),
				ChunkSize:  ptr.FromValue(uint8(10)),
				FakePacket: []byte{0x01},
			},
			override: &HTTPSOptions{
				Disorder:   ptr.FromValue(true),
				FakePacket: []byte{0x02},
			},
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.True(t, *output.Disorder)
				assert.Equal(t, uint8(10), *output.ChunkSize)
				assert.Equal(t, []byte{0x02}, output.FakePacket)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.base.Merge(tc.override)
			tc.assert(t, output)
		})
	}
}

// ┌────────────────┐
// │ POLICY OPTIONS │
// └────────────────┘
func TestPolicyOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o PolicyOptions)
	}{
		{
			name: "valid policy options",
			input: map[string]any{
				"auto": true,
				"overrides": []map[string]any{
					{
						"name": "rule1",
						"match": map[string]any{
							"domain": "example.com",
						},
					},
				},
			},
			wantErr: false,
			assert: func(t *testing.T, o PolicyOptions) {
				assert.True(t, *o.Auto)
				assert.Len(t, o.Overrides, 1)
				assert.Equal(t, "rule1", *o.Overrides[0].Name)
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var o PolicyOptions
			err := o.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, o)
				}
			}
		})
	}
}

func TestPolicyOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *PolicyOptions
		assert func(t *testing.T, input *PolicyOptions, output *PolicyOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *PolicyOptions, output *PolicyOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &PolicyOptions{
				Auto: ptr.FromValue(true),
				Overrides: []Rule{
					{
						Name:  ptr.FromValue("rule1"),
						Match: &MatchAttrs{Domain: ptr.FromValue("example.com")},
					},
				},
			},
			assert: func(t *testing.T, input *PolicyOptions, output *PolicyOptions) {
				assert.NotNil(t, output)
				assert.True(t, *output.Auto)
				assert.Len(t, output.Overrides, 1)
				assert.NotSame(t, input, output)
				// Deep copy check for slice
				assert.Equal(t, "rule1", *output.Overrides[0].Name)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

func TestPolicyOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *PolicyOptions
		override *PolicyOptions
		assert   func(t *testing.T, output *PolicyOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &PolicyOptions{Auto: ptr.FromValue(true)},
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.True(t, *output.Auto)
			},
		},
		{
			name:     "nil override",
			base:     &PolicyOptions{Auto: ptr.FromValue(false)},
			override: nil,
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.False(t, *output.Auto)
			},
		},
		{
			name: "merge values",
			base: &PolicyOptions{
				Auto:      ptr.FromValue(false),
				Overrides: []Rule{{Name: ptr.FromValue("rule1")}},
			},
			override: &PolicyOptions{
				Auto:      ptr.FromValue(true),
				Overrides: []Rule{{Name: ptr.FromValue("rule2")}},
			},
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.True(t, *output.Auto)
				assert.Len(t, output.Overrides, 2)
				assert.Equal(t, "rule1", *output.Overrides[0].Name)
				assert.Equal(t, "rule2", *output.Overrides[1].Name)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.base.Merge(tc.override)
			tc.assert(t, output)
		})
	}
}

// ┌─────────────┐
// │ MATCH ATTRS │
// └─────────────┘
func TestMatchAttrs_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, m MatchAttrs)
	}{
		{
			name: "valid domain",
			input: map[string]any{
				"domain": "example.com",
			},
			wantErr: false,
			assert: func(t *testing.T, m MatchAttrs) {
				assert.Equal(t, "example.com", *m.Domain)
				assert.Nil(t, m.CIDR)
				assert.Nil(t, m.PortFrom)
				assert.Nil(t, m.PortTo)
			},
		},
		{
			name: "valid cidr with port",
			input: map[string]any{
				"cidr": "192.168.1.0/24",
				"port": "80",
			},
			wantErr: false,
			assert: func(t *testing.T, m MatchAttrs) {
				assert.Equal(t, "192.168.1.0/24", m.CIDR.String())
				assert.Equal(t, uint16(80), *m.PortFrom)
				assert.Equal(t, uint16(80), *m.PortTo)
			},
		},
		{
			name: "cidr requires port",
			input: map[string]any{
				"cidr": "192.168.1.0/24",
			},
			wantErr: true,
		},
		{
			name: "port requires cidr",
			input: map[string]any{
				"port": "all",
			},
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var m MatchAttrs
			err := m.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tc.assert != nil {
				tc.assert(t, m)
			}
		})
	}
}

func TestMatchAttrs_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *MatchAttrs
		assert func(t *testing.T, input *MatchAttrs, output *MatchAttrs)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *MatchAttrs, output *MatchAttrs) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &MatchAttrs{
				Domain: ptr.FromValue("example.com"),
			},
			assert: func(t *testing.T, input *MatchAttrs, output *MatchAttrs) {
				assert.NotNil(t, output)
				assert.Equal(t, "example.com", *output.Domain)
				assert.NotSame(t, input, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}

// ┌──────┐
// │ RULE │
// └──────┘
func TestRule_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, r Rule)
	}{
		{
			name: "valid rule",
			input: map[string]any{
				"name": "rule1",
				"match": map[string]any{
					"domain": "example.com",
				},
				"block": true,
			},
			wantErr: false,
			assert: func(t *testing.T, r Rule) {
				assert.Equal(t, "rule1", *r.Name)
				assert.Equal(t, "example.com", *r.Match.Domain)
				assert.True(t, *r.Block)
			},
		},
		{
			name: "invalid rule check",
			input: map[string]any{
				"name": "rule1",
			},
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var r Rule
			err := r.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, r)
				}
			}
		})
	}
}

func TestRule_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *Rule
		assert func(t *testing.T, input *Rule, output *Rule)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *Rule, output *Rule) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &Rule{
				Name:  ptr.FromValue("rule1"),
				Match: &MatchAttrs{Domain: ptr.FromValue("example.com")},
			},
			assert: func(t *testing.T, input *Rule, output *Rule) {
				assert.NotNil(t, output)
				assert.Equal(t, "rule1", *output.Name)
				assert.NotSame(t, input, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.Clone()
			tc.assert(t, tc.input, output)
		})
	}
}
