package config

import (
	"net"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

// ┌─────────────────┐
// │ GENERAL OPTIONS │
// └─────────────────┘
func TestAppOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o AppOptions)
	}{
		{
			name: "valid general options",
			input: map[string]any{
				"log-level":              "debug",
				"silent":                 true,
				"auto-configure-network": true,
				"mode":                   "socks5",
			},
			wantErr: false,
			assert: func(t *testing.T, o AppOptions) {
				assert.Equal(t, zerolog.DebugLevel, *o.LogLevel)
				assert.True(t, *o.Silent)
				assert.True(t, *o.AutoConfigureNetwork)
				assert.Equal(t, AppModeSOCKS5, *o.Mode)
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
			var o AppOptions
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

func TestAppOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *AppOptions
		assert func(t *testing.T, input *AppOptions, output *AppOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *AppOptions, output *AppOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &AppOptions{
				LogLevel: lo.ToPtr(zerolog.DebugLevel),
				Silent:   lo.ToPtr(true),
			},
			assert: func(t *testing.T, input *AppOptions, output *AppOptions) {
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

func TestAppOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *AppOptions
		override *AppOptions
		assert   func(t *testing.T, output *AppOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &AppOptions{Silent: lo.ToPtr(true)},
			assert: func(t *testing.T, output *AppOptions) {
				assert.True(t, *output.Silent)
			},
		},
		{
			name:     "nil override",
			base:     &AppOptions{Silent: lo.ToPtr(false)},
			override: nil,
			assert: func(t *testing.T, output *AppOptions) {
				assert.False(t, *output.Silent)
			},
		},
		{
			name: "merge values",
			base: &AppOptions{
				Silent:   lo.ToPtr(false),
				LogLevel: lo.ToPtr(zerolog.InfoLevel),
			},
			override: &AppOptions{
				Silent: lo.ToPtr(true),
			},
			assert: func(t *testing.T, output *AppOptions) {
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
func TestConnOptions_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, o ConnOptions)
	}{
		{
			name: "valid server options",
			input: map[string]any{
				"default-fake-ttl": int64(64),
				"dns-timeout":      int64(1000),
				"tcp-timeout":      int64(1000),
				"udp-idle-timeout": int64(1000),
			},
			wantErr: false,
			assert: func(t *testing.T, o ConnOptions) {
				assert.Equal(t, uint8(64), *o.DefaultFakeTTL)
				assert.Equal(t, 1000*time.Millisecond, *o.DNSTimeout)
				assert.Equal(t, 1000*time.Millisecond, *o.TCPTimeout)
				assert.Equal(t, 1000*time.Millisecond, *o.UDPIdleTimeout)
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
			var o ConnOptions
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

func TestConnOptions_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *ConnOptions
		assert func(t *testing.T, input *ConnOptions, output *ConnOptions)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *ConnOptions, output *ConnOptions) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &ConnOptions{
				DefaultFakeTTL: lo.ToPtr(uint8(64)),
				DNSTimeout:     lo.ToPtr(time.Duration(1000) * time.Millisecond),
				TCPTimeout:     lo.ToPtr(time.Duration(1000) * time.Millisecond),
				UDPIdleTimeout: lo.ToPtr(time.Duration(1000) * time.Millisecond),
			},
			assert: func(t *testing.T, input *ConnOptions, output *ConnOptions) {
				assert.NotNil(t, output)
				assert.Equal(t, uint8(64), *output.DefaultFakeTTL)
				assert.Equal(t, 1000*time.Millisecond, *output.DNSTimeout)
				assert.Equal(t, 1000*time.Millisecond, *output.TCPTimeout)
				assert.Equal(t, 1000*time.Millisecond, *output.UDPIdleTimeout)
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

func TestConnOptions_Merge(t *testing.T) {
	tcs := []struct {
		name     string
		base     *ConnOptions
		override *ConnOptions
		assert   func(t *testing.T, output *ConnOptions)
	}{
		{
			name:     "nil receiver",
			base:     nil,
			override: &ConnOptions{DefaultFakeTTL: lo.ToPtr(uint8(64))},
			assert: func(t *testing.T, output *ConnOptions) {
				assert.Equal(t, uint8(64), *output.DefaultFakeTTL)
			},
		},
		{
			name:     "nil override",
			base:     &ConnOptions{DefaultFakeTTL: lo.ToPtr(uint8(128))},
			override: nil,
			assert: func(t *testing.T, output *ConnOptions) {
				assert.Equal(t, uint8(128), *output.DefaultFakeTTL)
			},
		},
		{
			name: "merge values",
			base: &ConnOptions{
				DefaultFakeTTL: lo.ToPtr(uint8(64)),
				DNSTimeout:     lo.ToPtr(time.Duration(1000) * time.Millisecond),
				TCPTimeout:     lo.ToPtr(time.Duration(1000) * time.Millisecond),
				UDPIdleTimeout: lo.ToPtr(time.Duration(1000) * time.Millisecond),
			},
			override: &ConnOptions{
				DefaultFakeTTL: lo.ToPtr(uint8(128)),
			},
			assert: func(t *testing.T, output *ConnOptions) {
				assert.Equal(t, uint8(128), *output.DefaultFakeTTL)
				assert.Equal(t, 1000*time.Millisecond, *output.DNSTimeout)
				assert.Equal(t, 1000*time.Millisecond, *output.TCPTimeout)
				assert.Equal(t, 1000*time.Millisecond, *output.UDPIdleTimeout)
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
				Mode: lo.ToPtr(DNSModeHTTPS),
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
			override: &DNSOptions{Mode: lo.ToPtr(DNSModeHTTPS)},
			assert: func(t *testing.T, output *DNSOptions) {
				assert.Equal(t, DNSModeHTTPS, *output.Mode)
			},
		},
		{
			name:     "nil override",
			base:     &DNSOptions{Mode: lo.ToPtr(DNSModeUDP)},
			override: nil,
			assert: func(t *testing.T, output *DNSOptions) {
				assert.Equal(t, DNSModeUDP, *output.Mode)
			},
		},
		{
			name: "merge values",
			base: &DNSOptions{
				Mode: lo.ToPtr(DNSModeUDP),
				Addr: &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
			},
			override: &DNSOptions{
				Mode:     lo.ToPtr(DNSModeUDP),
				HTTPSURL: lo.ToPtr("https://dns.google/test"),
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
				assert.Equal(t, []byte{0x01, 0x02}, o.FakePacket.Raw())
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
			name: "non-nil receiver",
			input: &HTTPSOptions{
				Disorder:   lo.ToPtr(true),
				FakePacket: proto.NewFakeTLSMessage([]byte{0x01}),
			},
			assert: func(t *testing.T, input *HTTPSOptions, output *HTTPSOptions) {
				assert.NotNil(t, output)
				assert.True(t, *output.Disorder)
				assert.NotSame(t, input, output)
				if output.FakePacket != nil {
					assert.Equal(t, input.FakePacket.Raw(), output.FakePacket.Raw())
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
			override: &HTTPSOptions{Disorder: lo.ToPtr(true)},
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.True(t, *output.Disorder)
			},
		},
		{
			name:     "nil override",
			base:     &HTTPSOptions{Disorder: lo.ToPtr(false)},
			override: nil,
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.False(t, *output.Disorder)
			},
		},
		{
			name: "merge values",
			base: &HTTPSOptions{
				Disorder:   lo.ToPtr(false),
				ChunkSize:  lo.ToPtr(uint8(10)),
				FakePacket: proto.NewFakeTLSMessage([]byte{0x01}),
			},
			override: &HTTPSOptions{
				Disorder:   lo.ToPtr(true),
				FakePacket: proto.NewFakeTLSMessage([]byte{0x02}),
			},
			assert: func(t *testing.T, output *HTTPSOptions) {
				assert.True(t, *output.Disorder)
				assert.Equal(t, uint8(10), *output.ChunkSize)
				assert.Equal(t, []byte{0x02}, output.FakePacket.Raw())
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
				"overrides": []map[string]any{
					{
						"name": "rule1",
						"match": map[string]any{
							"domain": []any{"example.com"},
						},
					},
				},
			},
			wantErr: false,
			assert: func(t *testing.T, o PolicyOptions) {
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
				Overrides: []Rule{
					{
						Name:  lo.ToPtr("rule1"),
						Match: &MatchAttrs{Domains: []string{"example.com"}},
					},
				},
			},
			assert: func(t *testing.T, input *PolicyOptions, output *PolicyOptions) {
				assert.NotNil(t, output)
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
			override: &PolicyOptions{Overrides: []Rule{{Name: lo.ToPtr("rule1")}}},
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.Len(t, output.Overrides, 1)
			},
		},
		{
			name:     "nil override",
			base:     &PolicyOptions{Overrides: []Rule{{Name: lo.ToPtr("rule1")}}},
			override: nil,
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.Len(t, output.Overrides, 1)
			},
		},
		{
			name: "merge values",
			base: &PolicyOptions{
				Overrides: []Rule{{Name: lo.ToPtr("rule1")}},
			},
			override: &PolicyOptions{
				Overrides: []Rule{{Name: lo.ToPtr("rule2")}},
			},
			assert: func(t *testing.T, output *PolicyOptions) {
				assert.Len(t, output.Overrides, 1)
				assert.Equal(t, "rule2", *output.Overrides[0].Name)
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
				"domain": []any{"example.com"},
			},
			wantErr: false,
			assert: func(t *testing.T, m MatchAttrs) {
				assert.Len(t, m.Domains, 1)
				assert.Equal(t, "example.com", m.Domains[0])
				assert.Empty(t, m.Addrs)
			},
		},
		{
			name: "valid cidr with port",
			input: map[string]any{
				"addr": []any{
					map[string]any{
						"cidr": "192.168.1.0/24",
						"port": "80",
					},
				},
			},
			wantErr: false,
			assert: func(t *testing.T, m MatchAttrs) {
				assert.Len(t, m.Addrs, 1)
				assert.Equal(t, "192.168.1.0/24", m.Addrs[0].CIDR.String())
				assert.Equal(t, uint16(80), *m.Addrs[0].PortFrom)
				assert.Equal(t, uint16(80), *m.Addrs[0].PortTo)
			},
		},
		{
			name: "cidr requires port",
			input: map[string]any{
				"addr": []any{
					map[string]any{
						"cidr": "192.168.1.0/24",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "port requires cidr",
			input: map[string]any{
				"addr": []any{
					map[string]any{
						"port": "all",
					},
				},
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
				Domains: []string{"example.com"},
			},
			assert: func(t *testing.T, input *MatchAttrs, output *MatchAttrs) {
				assert.NotNil(t, output)
				assert.Equal(t, "example.com", output.Domains[0])
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
					"domain": []any{"example.com"},
				},
				"block": true,
			},
			wantErr: false,
			assert: func(t *testing.T, r Rule) {
				assert.Equal(t, "rule1", *r.Name)
				assert.Equal(t, "example.com", r.Match.Domains[0])
				assert.True(t, *r.Block)
			},
		},
		{
			name: "valid rule with connection options",
			input: map[string]any{
				"name": "rule2",
				"connection": map[string]any{
					"tcp-timeout": int64(500),
				},
			},
			wantErr: false,
			assert: func(t *testing.T, r Rule) {
				assert.Equal(t, "rule2", *r.Name)
				assert.Equal(t, time.Duration(500*time.Millisecond), *r.Conn.TCPTimeout)
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
				Name:  lo.ToPtr("rule1"),
				Match: &MatchAttrs{Domains: []string{"example.com"}},
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

func TestSegmentPlan_UnmarshalTOML(t *testing.T) {
	t.Run("valid segment head", func(t *testing.T) {
		input := `
from = "head"
at = 10
lazy = true
noise = 1
`
		var s SegmentPlan
		err := toml.Unmarshal([]byte(input), &s)
		assert.NoError(t, err)
		assert.Equal(t, SegmentFromHead, s.From)
		assert.Equal(t, 10, s.At)
		assert.True(t, s.Lazy)
		assert.Equal(t, 1, s.Noise)
	})

	t.Run("valid segment sni", func(t *testing.T) {
		input := `
from = "sni"
at = -5
`
		var s SegmentPlan
		err := toml.Unmarshal([]byte(input), &s)
		assert.NoError(t, err)
		assert.Equal(t, SegmentFromSNI, s.From)
		assert.Equal(t, -5, s.At)
	})

	t.Run("missing required field from", func(t *testing.T) {
		input := `
at = 5
`
		var s SegmentPlan
		err := toml.Unmarshal([]byte(input), &s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field 'from' is required")
	})

	t.Run("missing required field at", func(t *testing.T) {
		input := `
from = "head"
`
		var s SegmentPlan
		err := toml.Unmarshal([]byte(input), &s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field 'at' is required")
	})

	t.Run("invalid from value", func(t *testing.T) {
		input := `
from = "invalid"
at = 5
`
		var s SegmentPlan
		err := toml.Unmarshal([]byte(input), &s)
		assert.Error(t, err)
	})
}

func TestHTTPSOptions_CustomSegmentPlans(t *testing.T) {
	t.Run("valid custom config", func(t *testing.T) {
		input := `
split-mode = "custom"
custom-segments = [
	{ from = "head", at = 2 },
	{ from = "sni", at = 0 }
]
`
		var opts HTTPSOptions
		err := toml.Unmarshal([]byte(input), &opts)
		assert.NoError(t, err)
		assert.Equal(t, HTTPSSplitModeCustom, *opts.SplitMode)
		assert.Len(t, opts.CustomSegmentPlans, 2)
		assert.Equal(t, SegmentFromHead, opts.CustomSegmentPlans[0].From)
		assert.Equal(t, 2, opts.CustomSegmentPlans[0].At)
	})

	t.Run("missing custom segments", func(t *testing.T) {
		input := `
split-mode = "custom"
`
		var opts HTTPSOptions
		err := toml.Unmarshal([]byte(input), &opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "custom-segments must be provided")
	})
}
