package main

import (
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func TestCreateResolver(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DNS = &config.DNSOptions{
		Mode:     ptr.FromValue(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: ptr.FromValue("https://dns.google/dns-query"),
		QType:    ptr.FromValue(config.DNSQueryIPv4),
		Cache:    ptr.FromValue(true),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	assert.NotNil(t, resolver)
}

func TestCreateProxy_NoPcap(t *testing.T) {
	// Setup configuration that doesn't require PCAP (root privileges)
	cfg := config.NewConfig()

	// Server Config
	cfg.Server = &config.ServerOptions{
		ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
		DefaultTTL: ptr.FromValue(uint8(64)),
		Timeout:    ptr.FromValue(time.Duration(1 * time.Second)),
	}

	// HTTPS Config (Ensure FakeCount is 0 to disable PCAP)
	cfg.HTTPS = &config.HTTPSOptions{
		Disorder:   ptr.FromValue(false),
		FakeCount:  ptr.FromValue(uint8(0)),
		FakePacket: []byte{},
		SplitMode:  ptr.FromValue(config.HTTPSSplitModeChunk),
		ChunkSize:  ptr.FromValue(uint8(10)),
		Skip:       ptr.FromValue(false),
	}

	// Policy Config
	cfg.Policy = &config.PolicyOptions{
		Auto: ptr.FromValue(false),
	}

	// DNS Config
	cfg.DNS = &config.DNSOptions{
		Mode:     ptr.FromValue(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: ptr.FromValue("https://dns.google/dns-query"),
		QType:    ptr.FromValue(config.DNSQueryIPv4),
		Cache:    ptr.FromValue(false),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	p, err := createProxy(logger, cfg, resolver)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestCreateProxy_WithPolicy(t *testing.T) {
	cfg := config.NewConfig()

	// Server Config
	cfg.Server = &config.ServerOptions{
		ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
		DefaultTTL: ptr.FromValue(uint8(64)),
		Timeout:    ptr.FromValue(time.Duration(0)),
	}

	// HTTPS Config
	cfg.HTTPS = &config.HTTPSOptions{
		FakeCount: ptr.FromValue(uint8(0)),
	}

	// Policy Config with one override
	cfg.Policy = &config.PolicyOptions{
		Auto: ptr.FromValue(false),
		Overrides: []config.Rule{
			{
				Name: ptr.FromValue("test-rule"),
				Match: &config.MatchAttrs{
					Domain: ptr.FromValue("example.com"),
				},
				DNS: &config.DNSOptions{
					Mode: ptr.FromValue(config.DNSModeSystem),
				},
				HTTPS: &config.HTTPSOptions{
					Skip: ptr.FromValue(true),
				},
			},
		},
	}

	// DNS Config
	cfg.DNS = &config.DNSOptions{
		Mode:     ptr.FromValue(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: ptr.FromValue("https://dns.google/dns-query"),
		QType:    ptr.FromValue(config.DNSQueryIPv4),
		Cache:    ptr.FromValue(false),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	p, err := createProxy(logger, cfg, resolver)
	require.NoError(t, err)
	assert.NotNil(t, p)
}
