package config

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand_Flags(t *testing.T) {
	tcs := []struct {
		name   string
		args   []string
		assert func(t *testing.T, cfg *Config)
	}{
		{
			name: "default values (no flags)",
			args: []string{"spoofdpi", "--clean"},
			assert: func(t *testing.T, cfg *Config) {
				// Verify defaults are preserved
				assert.Equal(t, zerolog.InfoLevel, *cfg.General.LogLevel)
				assert.False(t, *cfg.General.Silent)
				assert.False(t, *cfg.General.SetSystemProxy)
				assert.Equal(t, "127.0.0.1:8080", cfg.Server.ListenAddr.String())
				assert.Equal(t, uint8(64), *cfg.Server.DefaultTTL)
				assert.Equal(t, time.Duration(0), *cfg.Server.Timeout)
				assert.Equal(t, "8.8.8.8:53", cfg.DNS.Addr.String())
				assert.Equal(t, DNSModeUDP, *cfg.DNS.Mode)
				assert.Equal(t, "https://dns.google/dns-query", *cfg.DNS.HTTPSURL)
				assert.Equal(t, DNSQueryIPv4, *cfg.DNS.QType)
				assert.False(t, *cfg.DNS.Cache)
				assert.Equal(t, uint8(0), *cfg.HTTPS.FakeCount)
				assert.False(t, *cfg.HTTPS.Disorder)
				assert.Equal(t, HTTPSSplitModeSNI, *cfg.HTTPS.SplitMode)
				assert.Equal(t, uint8(0), *cfg.HTTPS.ChunkSize)
				assert.False(t, *cfg.HTTPS.Skip)
				assert.False(t, *cfg.Policy.Auto)
			},
		},
		{
			name: "all flags set with custom values",
			args: []string{
				"spoofdpi",
				"--clean", // Ensure no config file interferes
				"--log-level", "debug",
				"--silent",
				"--system-proxy",
				"--listen-addr", "127.0.0.1:9090",
				"--default-ttl", "128",
				"--timeout", "5000",
				"--dns-addr", "1.1.1.1:53",
				"--dns-mode", "https",
				"--dns-https-url", "https://cloudflare-dns.com/dns-query",
				"--dns-qtype", "ipv6",
				"--dns-cache",
				"--https-fake-count", "10",
				"--https-fake-packet", "0x16, 0x03",
				"--https-disorder",
				"--https-split-mode", "chunk",
				"--https-chunk-size", "50",
				"--https-skip",
				"--policy-auto",
			},
			assert: func(t *testing.T, cfg *Config) {
				// General
				assert.Equal(t, zerolog.DebugLevel, *cfg.General.LogLevel)
				assert.True(t, *cfg.General.Silent)
				assert.True(t, *cfg.General.SetSystemProxy)

				// Server
				assert.Equal(t, "127.0.0.1:9090", cfg.Server.ListenAddr.String())
				assert.Equal(t, uint8(128), *cfg.Server.DefaultTTL)
				assert.Equal(t, 5000*time.Millisecond, *cfg.Server.Timeout)

				// DNS
				assert.Equal(t, "1.1.1.1:53", cfg.DNS.Addr.String())
				assert.Equal(t, DNSModeHTTPS, *cfg.DNS.Mode)
				assert.Equal(t, "https://cloudflare-dns.com/dns-query", *cfg.DNS.HTTPSURL)
				assert.Equal(t, DNSQueryIPv6, *cfg.DNS.QType)
				assert.True(t, *cfg.DNS.Cache)

				// HTTPS
				assert.Equal(t, uint8(10), *cfg.HTTPS.FakeCount)
				assert.Equal(t, []byte{0x16, 0x03}, cfg.HTTPS.FakePacket)
				assert.True(t, *cfg.HTTPS.Disorder)
				assert.Equal(t, HTTPSSplitModeChunk, *cfg.HTTPS.SplitMode)
				assert.Equal(t, uint8(50), *cfg.HTTPS.ChunkSize)
				assert.True(t, *cfg.HTTPS.Skip)

				// Policy
				assert.True(t, *cfg.Policy.Auto)
			},
		},
		{
			name: "alternative values",
			args: []string{
				"spoofdpi",
				"--clean",
				"--log-level", "error",
				"--dns-mode", "system",
				"--dns-qtype", "all",
				"--https-split-mode", "random",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, zerolog.ErrorLevel, *cfg.General.LogLevel)
				assert.Equal(t, DNSModeSystem, *cfg.DNS.Mode)
				assert.Equal(t, DNSQueryAll, *cfg.DNS.QType)
				assert.Equal(t, HTTPSSplitModeRandom, *cfg.HTTPS.SplitMode)
			},
		},
		{
			name: "ipv6 listen addr",
			args: []string{
				"spoofdpi",
				"--clean",
				"--listen-addr", "[::1]:1080",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "[::1]:1080", cfg.Server.ListenAddr.String())
				ip := net.ParseIP("::1")
				assert.True(t, cfg.Server.ListenAddr.IP.Equal(ip))
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var capturedCfg *Config
			runFunc := func(ctx context.Context, configDir string, cfg *Config) {
				capturedCfg = cfg
			}

			cmd := CreateCommand(runFunc, "v0.0.0", "commit", "build")
			// We need to suppress stdout/stderr for cleaner test output,
			// or we can let it be.
			// cmd.Writer = io.Discard
			// cmd.ErrWriter = io.Discard

			err := cmd.Run(context.Background(), tc.args)
			require.NoError(t, err)
			require.NotNil(t, capturedCfg, "Run function was not called")

			tc.assert(t, capturedCfg)
		})
	}
}

func TestCreateCommand_OverrideTOML(t *testing.T) {
	tomlContent := `
[general]
    log-level = "debug"
    silent = true
    system-proxy = true

[server]
    listen-addr = "127.0.0.1:8080"
    timeout = 1000
    default-ttl = 100

[dns]
    addr = "8.8.8.8:53"
    cache = true
    mode = "https"
    https-url = "https://1.1.1.1/dns-query"
    qtype = "ipv4"

[https]
    disorder = true
    fake-count = 5
    fake-packet = [0x01, 0x02, 0x03]
    split-mode = "chunk"
    chunk-size = 20
    skip = true

[policy]
    auto = true
    [[policy.overrides]]
        name = "test-rule"
        priority = 100
        block = true
        match = { 
            domain = "example.com", 
            cidr = "192.168.1.0/24", 
            port = "80-443",
        }
        dns = { 
            mode = "udp", 
            addr = "8.8.4.4:53",
            https-url = "https://8.8.8.8/dns-query", 
            qtype = "ipv6", 
            block = true, 
            cache = false,
        }
        https = { 
            disorder = false, 
            fake-count = 2, 
            fake-packet = [0xAA, 0xBB], 
            split-mode = "sni",
            chunk-size = 10, 
            skip = true,
        }
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spoofdpi.toml")
	err := os.WriteFile(configPath, []byte(tomlContent), 0o644)
	require.NoError(t, err)

	var capturedCfg *Config
	runFunc := func(ctx context.Context, configDir string, cfg *Config) {
		capturedCfg = cfg
	}

	cmd := CreateCommand(runFunc, "v0.0.0", "commit", "build")

	args := []string{
		"spoofdpi",
		"--config", configPath,
		"--log-level", "error",
		"--silent=false",
		"--system-proxy=false",
		"--listen-addr", "127.0.0.1:9090",
		"--timeout", "2000",
		"--default-ttl", "200",
		"--dns-addr", "1.1.1.1:53",
		"--dns-cache=false",
		"--dns-mode", "udp",
		"--dns-https-url", "https://8.8.8.8/dns-query",
		"--dns-qtype", "ipv6",
		"--https-disorder=false",
		"--https-fake-count", "10",
		"--https-fake-packet", "0xff,0xff",
		"--https-split-mode", "sni",
		"--https-chunk-size", "10",
		"--https-skip=false",
		"--policy-auto=false",
	}

	err = cmd.Run(context.Background(), args)
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)

	// Verify Overrides
	// General
	assert.Equal(t, zerolog.ErrorLevel, *capturedCfg.General.LogLevel)
	assert.False(t, *capturedCfg.General.Silent)
	assert.False(t, *capturedCfg.General.SetSystemProxy)

	// Server
	assert.Equal(t, "127.0.0.1:9090", capturedCfg.Server.ListenAddr.String())
	assert.Equal(t, 2000*time.Millisecond, *capturedCfg.Server.Timeout)
	assert.Equal(t, uint8(200), *capturedCfg.Server.DefaultTTL)

	// DNS
	assert.Equal(t, "1.1.1.1:53", capturedCfg.DNS.Addr.String())
	assert.False(t, *capturedCfg.DNS.Cache)
	assert.Equal(t, DNSModeUDP, *capturedCfg.DNS.Mode)
	assert.Equal(t, "https://8.8.8.8/dns-query", *capturedCfg.DNS.HTTPSURL)
	assert.Equal(t, DNSQueryIPv6, *capturedCfg.DNS.QType)

	// HTTPS
	assert.False(t, *capturedCfg.HTTPS.Disorder)
	assert.Equal(t, uint8(10), *capturedCfg.HTTPS.FakeCount)
	assert.Equal(t, []byte{0xff, 0xff}, capturedCfg.HTTPS.FakePacket)
	assert.Equal(t, HTTPSSplitModeSNI, *capturedCfg.HTTPS.SplitMode)
	assert.Equal(t, uint8(10), *capturedCfg.HTTPS.ChunkSize)
	assert.False(t, *capturedCfg.HTTPS.Skip)

	// Policy
	assert.False(t, *capturedCfg.Policy.Auto)

	// Verify TOML-only fields are preserved
	require.Len(t, capturedCfg.Policy.Overrides, 1)
	override := capturedCfg.Policy.Overrides[0]
	assert.Equal(t, "test-rule", *override.Name)
	assert.Equal(t, "example.com", *override.Match.Domain)
}
