package config

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func TestConfig_UnmarshalTOML(t *testing.T) {
	tcs := []struct {
		name    string
		input   any
		wantErr bool
		assert  func(t *testing.T, c Config)
	}{
		{
			name: "valid config",
			input: map[string]any{
				"server": map[string]any{
					"listen-addr": "127.0.0.1:9090",
				},
				"dns": map[string]any{
					"addr": "1.1.1.1:53",
				},
				"policy": map[string]any{
					"overrides": []map[string]any{
						{
							"name": "test",
							"match": map[string]any{
								"domain": "example.com",
							},
							"dns": map[string]any{
								"route": "doh",
							},
						},
					},
				},
			},
			wantErr: false,
			assert: func(t *testing.T, c Config) {
				assert.Equal(t, "127.0.0.1:9090", c.Server.ListenAddr.String())
				assert.Equal(t, "1.1.1.1:53", c.DNS.Addr.String())
				if assert.Len(t, c.Policy.Overrides, 1) {
					assert.Equal(t, "test", *c.Policy.Overrides[0].Name)
				}
			},
		},
		{
			name:    "invalid type",
			input:   "invalid",
			wantErr: true,
		},
		{
			name: "validation error",
			input: map[string]any{
				"server": map[string]any{
					"listen-addr": "invalid-addr",
				},
			},
			wantErr: true,
		},
		{
			name: "nested validation error (rule)",
			input: map[string]any{
				"policy": map[string]any{
					"overrides": []map[string]any{
						{
							"name": "invalid rule",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var c Config
			err := c.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.assert != nil {
					tc.assert(t, c)
				}
			}
		})
	}
}

func TestConfig_ShouldEnablePcap(t *testing.T) {
	tcs := []struct {
		name   string
		config Config
		expect bool
	}{
		{
			name: "global fake count > 0",
			config: Config{
				HTTPS: &HTTPSOptions{
					FakeCount: ptr.FromValue(uint8(1)),
				},
			},
			expect: true,
		},
		{
			name: "rule fake count > 0",
			config: Config{
				HTTPS: &HTTPSOptions{
					FakeCount: ptr.FromValue(uint8(0)),
				},
				Policy: &PolicyOptions{
					Overrides: []Rule{
						{
							HTTPS: &HTTPSOptions{
								FakeCount: ptr.FromValue(uint8(1)),
							},
						},
					},
				},
			},
			expect: true,
		},
		{
			name: "none",
			config: Config{
				HTTPS: &HTTPSOptions{
					FakeCount: ptr.FromValue(uint8(0)),
				},
				Policy: &PolicyOptions{
					Overrides: []Rule{
						{
							HTTPS: &HTTPSOptions{
								FakeCount: ptr.FromValue(uint8(0)),
							},
						},
					},
				},
			},
			expect: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, tc.config.ShouldEnablePcap())
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	tcs := []struct {
		name    string
		tomlCfg *Config
		argsCfg *Config
		assert  func(t *testing.T, merged *Config)
	}{
		{
			name: "keep toml if arg is nil",
			tomlCfg: &Config{
				Server: &ServerOptions{
					ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
				},
			},
			argsCfg: &Config{
				Server: &ServerOptions{
					ListenAddr: nil,
				},
			},
			assert: func(t *testing.T, merged *Config) {
				assert.Equal(t, "127.0.0.1:8080", merged.Server.ListenAddr.String())
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, tc.tomlCfg.Merge(tc.argsCfg))
		})
	}
}

func TestConfig_Clone(t *testing.T) {
	tcs := []struct {
		name   string
		input  *Config
		assert func(t *testing.T, input *Config, output *Config)
	}{
		{
			name:  "nil receiver",
			input: nil,
			assert: func(t *testing.T, input *Config, output *Config) {
				assert.Nil(t, output)
			},
		},
		{
			name: "non-nil receiver",
			input: &Config{
				General: &GeneralOptions{},
				Server:  &ServerOptions{},
				DNS:     &DNSOptions{},
				HTTPS:   &HTTPSOptions{},
				Policy:  &PolicyOptions{},
			},
			assert: func(t *testing.T, input *Config, output *Config) {
				assert.NotNil(t, output)
				assert.NotSame(t, input, output)
				assert.NotSame(t, input.General, output.General)
				assert.NotSame(t, input.Server, output.Server)
				assert.NotSame(t, input.DNS, output.DNS)
				assert.NotSame(t, input.HTTPS, output.HTTPS)
				assert.NotSame(t, input.Policy, output.Policy)
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
