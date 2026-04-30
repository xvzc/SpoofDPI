package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			c := DefaultConfig()
			err := c.UnmarshalTOML(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			// Finalize promotes captured raw overrides into resolved Rules.
			require.NoError(t, c.Finalize())
			if tc.assert != nil {
				tc.assert(t, *c)
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

func TestConfig_resolveRules_inheritsFromBase(t *testing.T) {
	c := DefaultConfig()
	c.HTTPS.FakeCount = 5
	c.HTTPS.SplitMode = HTTPSSplitModeChunk
	c.HTTPS.ChunkSize = 99
	c.Policy.rawOverrides = []map[string]any{
		{
			"name": "rule1",
			"match": map[string]any{
				"domain": []any{"example.com"},
			},
			"https": map[string]any{
				// Only override fake-count; other HTTPS fields should
				// inherit from the base config (eager-resolve at load).
				"fake-count": int64(2),
			},
		},
	}

	require.NoError(t, c.Finalize())
	require.Len(t, c.Policy.Overrides, 1)
	rule := c.Policy.Overrides[0]
	assert.Equal(t, "rule1", rule.Name)
	assert.Equal(t, uint8(2), rule.HTTPS.FakeCount, "rule overrides fake-count")
	assert.Equal(
		t, HTTPSSplitModeChunk, rule.HTTPS.SplitMode,
		"rule inherits split-mode from base",
	)
	assert.Equal(t, uint8(99), rule.HTTPS.ChunkSize, "rule inherits chunk-size from base")
	// rawOverrides should be cleared after resolution
	assert.Nil(t, c.Policy.rawOverrides)
}
