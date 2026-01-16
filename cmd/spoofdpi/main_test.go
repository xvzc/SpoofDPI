package main

import (
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

func TestCreateResolver(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DNS = &config.DNSOptions{
		Mode:     lo.ToPtr(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: lo.ToPtr("https://dns.google/dns-query"),
		QType:    lo.ToPtr(config.DNSQueryIPv4),
		Cache:    lo.ToPtr(true),
	}
	cfg.Conn = &config.ConnOptions{
		DNSTimeout: lo.ToPtr(time.Duration(0)),
		TCPTimeout: lo.ToPtr(time.Duration(0)),
		UDPTimeout: lo.ToPtr(time.Duration(0)),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	assert.NotNil(t, resolver)
}

func TestCreateProxy_NoPcap(t *testing.T) {
	// Setup configuration that dAppModeHTTP PCAP (root privileges)
	cfg := config.NewConfig()

	// App Config
	cfg.App = &config.AppOptions{
		Mode:       lo.ToPtr(config.AppModeHTTP),
		ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
	}

	// Conn Config
	cfg.Conn = &config.ConnOptions{
		DefaultFakeTTL: lo.ToPtr(uint8(64)),
		DNSTimeout:     lo.ToPtr(time.Duration(0)),
		TCPTimeout:     lo.ToPtr(time.Duration(0)),
		UDPTimeout:     lo.ToPtr(time.Duration(0)),
	}

	// HTTPS Config (Ensure FakeCount is 0 to disable PCAP)
	cfg.HTTPS = &config.HTTPSOptions{
		Disorder:   lo.ToPtr(false),
		FakeCount:  lo.ToPtr(uint8(0)),
		FakePacket: proto.NewFakeTLSMessage([]byte{}),
		SplitMode:  lo.ToPtr(config.HTTPSSplitModeChunk),
		ChunkSize:  lo.ToPtr(uint8(10)),
		Skip:       lo.ToPtr(false),
	}

	// Policy Config
	cfg.Policy = &config.PolicyOptions{
		Auto: lo.ToPtr(false),
	}

	// DNS Config
	cfg.DNS = &config.DNSOptions{
		Mode:     lo.ToPtr(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: lo.ToPtr("https://dns.google/dns-query"),
		QType:    lo.ToPtr(config.DNSQueryIPv4),
		Cache:    lo.ToPtr(false),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	p, err := createServer(logger, cfg, resolver)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestCreateProxy_WithPolicy(t *testing.T) {
	cfg := config.NewConfig()

	// App Config
	cfg.App = &config.AppOptions{
		Mode:       lo.ToPtr(config.AppModeHTTP),
		ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0},
	}

	// Conn Config
	cfg.Conn = &config.ConnOptions{
		DefaultFakeTTL: lo.ToPtr(uint8(64)),
		DNSTimeout:     lo.ToPtr(time.Duration(0)),
		TCPTimeout:     lo.ToPtr(time.Duration(0)),
		UDPTimeout:     lo.ToPtr(time.Duration(0)),
	}

	// HTTPS Config
	cfg.HTTPS = &config.HTTPSOptions{
		FakeCount: lo.ToPtr(uint8(0)),
	}

	// Policy Config with one override
	cfg.Policy = &config.PolicyOptions{
		Auto: lo.ToPtr(false),
		Overrides: []config.Rule{
			{
				Name: lo.ToPtr("test-rule"),
				Match: &config.MatchAttrs{
					Domains: []string{"example.com"},
				},
				DNS: &config.DNSOptions{
					Mode: lo.ToPtr(config.DNSModeSystem),
				},
				HTTPS: &config.HTTPSOptions{
					Skip: lo.ToPtr(true),
				},
			},
		},
	}

	// DNS Config
	cfg.DNS = &config.DNSOptions{
		Mode:     lo.ToPtr(config.DNSModeUDP),
		Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		HTTPSURL: lo.ToPtr("https://dns.google/dns-query"),
		QType:    lo.ToPtr(config.DNSQueryIPv4),
		Cache:    lo.ToPtr(false),
	}

	logger := zerolog.Nop()
	resolver := createResolver(logger, cfg)

	p, err := createServer(logger, cfg, resolver)
	require.NoError(t, err)
	assert.NotNil(t, p)
}
