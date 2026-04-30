package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
				"app": map[string]any{
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
								"domain": []any{"example.com"},
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
				assert.Equal(t, "127.0.0.1:9090", c.App.ListenAddr.String())
				assert.Equal(t, "1.1.1.1:53", c.DNS.Addr.String())
				if assert.Len(t, c.Policy.Overrides, 1) {
					assert.Equal(t, "test", c.Policy.Overrides[0].Name)
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
				"app": map[string]any{
					"listen-addr": "invalid-addr",
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
				HTTPS: HTTPSOptions{
					FakeCount: uint8(1),
				},
			},
			expect: true,
		},
		{
			name: "rule fake count > 0",
			config: Config{
				HTTPS: HTTPSOptions{
					FakeCount: uint8(0),
				},
				Policy: PolicyOptions{
					Overrides: []Rule{
						{
							HTTPS: HTTPSOptions{
								FakeCount: uint8(1),
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
				HTTPS: HTTPSOptions{
					FakeCount: uint8(0),
				},
				Policy: PolicyOptions{
					Overrides: []Rule{
						{
							HTTPS: HTTPSOptions{
								FakeCount: uint8(0),
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
