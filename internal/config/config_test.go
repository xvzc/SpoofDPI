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
				assert.Equal(t, "127.0.0.1:9090", c.Startup.App.ListenAddr.String())
				assert.Equal(t, "1.1.1.1:53", c.Runtime.DNS.Addr.String())
				// Policy.Overrides is populated by Load via resolveRules,
				// not by Config.UnmarshalTOML; see TestResolveRules_*.
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
				Runtime: RuntimeConfig{
					HTTPS: HTTPSOptions{FakeCount: uint8(1)},
				},
			},
			expect: true,
		},
		{
			name: "rule fake count > 0",
			config: Config{
				Runtime: RuntimeConfig{
					HTTPS: HTTPSOptions{FakeCount: uint8(0)},
				},
				Startup: StartupConfig{
					Policy: PolicyOptions{
						Overrides: []Rule{
							{
								Runtime: RuntimeConfig{
									HTTPS: HTTPSOptions{FakeCount: uint8(1)},
								},
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
				Runtime: RuntimeConfig{
					HTTPS: HTTPSOptions{FakeCount: uint8(0)},
				},
				Startup: StartupConfig{
					Policy: PolicyOptions{
						Overrides: []Rule{
							{
								Runtime: RuntimeConfig{
									HTTPS: HTTPSOptions{FakeCount: uint8(0)},
								},
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

func TestConfig_Validate_rejectsRuleWithoutMatch(t *testing.T) {
	c := DefaultConfig()
	c.Startup.Policy.Overrides = []Rule{
		{
			Name:  "no-match",
			Match: nil, // missing match attribute
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy.overrides[0]")
	assert.Contains(t, err.Error(), "match attribute")
}

func TestConfig_Validate_acceptsValidRule(t *testing.T) {
	c := DefaultConfig()
	c.Startup.Policy.Overrides = []Rule{
		{
			Name: "ok",
			Match: &MatchAttrs{
				Domains: []string{"example.com"},
			},
		},
	}
	assert.NoError(t, c.Validate())
}

func TestResolveRules_inheritsFromBase(t *testing.T) {
	base := DefaultRuntimeConfig()
	base.HTTPS.FakeCount = 5
	base.HTTPS.SplitMode = HTTPSSplitModeChunk
	base.HTTPS.ChunkSize = 99

	raw := []map[string]any{
		{
			"name": "rule1",
			"match": map[string]any{
				"domain": []any{"example.com"},
			},
			"https": map[string]any{
				// Only override fake-count; other HTTPS fields should
				// inherit from the base RuntimeConfig (eager-resolve at load).
				"fake-count": int64(2),
			},
		},
	}

	rules, err := resolveRules(raw, base)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	rule := rules[0]
	assert.Equal(t, "rule1", rule.Name)
	assert.Equal(t, uint8(2), rule.Runtime.HTTPS.FakeCount, "rule overrides fake-count")
	assert.Equal(
		t, HTTPSSplitModeChunk, rule.Runtime.HTTPS.SplitMode,
		"rule inherits split-mode from base",
	)
	assert.Equal(
		t,
		uint8(99),
		rule.Runtime.HTTPS.ChunkSize,
		"rule inherits chunk-size from base",
	)
}

func TestResolveRules_skipAutoResetWhenBaseSkipTrue(t *testing.T) {
	tcs := []struct {
		name      string
		baseSkip  bool
		ruleHTTPS map[string]any
		wantSkip  bool
	}{
		{
			name:      "base skip=true, rule omits skip → reset to false",
			baseSkip:  true,
			ruleHTTPS: map[string]any{"chunk-size": int64(8)},
			wantSkip:  false,
		},
		{
			name:      "base skip=true, rule has no https section → reset to false",
			baseSkip:  true,
			ruleHTTPS: nil,
			wantSkip:  false,
		},
		{
			name:      "base skip=true, rule explicitly skip=true → kept",
			baseSkip:  true,
			ruleHTTPS: map[string]any{"skip": true},
			wantSkip:  true,
		},
		{
			name:      "base skip=true, rule explicitly skip=false → kept",
			baseSkip:  true,
			ruleHTTPS: map[string]any{"skip": false},
			wantSkip:  false,
		},
		{
			name:      "base skip=false, rule omits skip → false (no warning)",
			baseSkip:  false,
			ruleHTTPS: map[string]any{"chunk-size": int64(8)},
			wantSkip:  false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			base := DefaultRuntimeConfig()
			base.HTTPS.Skip = tc.baseSkip

			item := map[string]any{
				"name":  "r",
				"match": map[string]any{"domain": []any{"example.com"}},
			}
			if tc.ruleHTTPS != nil {
				item["https"] = tc.ruleHTTPS
			}

			rules, err := resolveRules([]map[string]any{item}, base)
			require.NoError(t, err)
			require.Len(t, rules, 1)
			assert.Equal(t, tc.wantSkip, rules[0].Runtime.HTTPS.Skip)
		})
	}
}
