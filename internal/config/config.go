package config

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type merger[T any] interface {
	Clone() T
	Merge(T) T
}

type cloner[T any] interface {
	Clone() T
}

var _ merger[*Config] = (*Config)(nil)

type Config struct {
	App    *AppOptions    `toml:"general"`
	Conn   *ConnOptions   `toml:"connection"`
	DNS    *DNSOptions    `toml:"dns"`
	HTTPS  *HTTPSOptions  `toml:"https"`
	UDP    *UDPOptions    `toml:"udp"`
	Policy *PolicyOptions `toml:"policy"`
}

func (c *Config) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type config file")
	}

	c.App = findStructFrom[AppOptions](m, "general", &err)
	c.Conn = findStructFrom[ConnOptions](m, "connection", &err)
	c.DNS = findStructFrom[DNSOptions](m, "dns", &err)
	c.HTTPS = findStructFrom[HTTPSOptions](m, "https", &err)
	c.UDP = findStructFrom[UDPOptions](m, "udp", &err)
	c.Policy = findStructFrom[PolicyOptions](m, "policy", &err)

	return
}

func NewConfig() *Config {
	return &Config{
		App:    &AppOptions{},
		Conn:   &ConnOptions{},
		DNS:    &DNSOptions{},
		HTTPS:  &HTTPSOptions{},
		UDP:    &UDPOptions{},
		Policy: &PolicyOptions{},
	}
}

func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	return &Config{
		App:    c.App.Clone(),
		Conn:   c.Conn.Clone(),
		DNS:    c.DNS.Clone(),
		HTTPS:  c.HTTPS.Clone(),
		UDP:    c.UDP.Clone(),
		Policy: c.Policy.Clone(),
	}
}

func (origin *Config) Merge(overrides *Config) *Config {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &Config{
		App:    origin.App.Merge(overrides.App),
		Conn:   origin.Conn.Merge(overrides.Conn),
		DNS:    origin.DNS.Merge(overrides.DNS),
		HTTPS:  origin.HTTPS.Merge(overrides.HTTPS),
		UDP:    origin.UDP.Merge(overrides.UDP),
		Policy: origin.Policy.Merge(overrides.Policy),
	}
}

func (c *Config) ShouldEnablePcap() bool {
	if *c.HTTPS.FakeCount > 0 {
		return true
	}

	if c.UDP != nil && c.UDP.FakeCount != nil && *c.UDP.FakeCount > 0 {
		return true
	}

	if c.Policy == nil {
		return false
	}

	if c.Policy.Template != nil {
		template := c.Policy.Template
		if template.HTTPS != nil && lo.FromPtr(template.HTTPS.FakeCount) > 0 {
			return true
		}
		if template.UDP != nil && lo.FromPtr(template.UDP.FakeCount) > 0 {
			return true
		}
	}

	if c.Policy.Overrides != nil {
		rules := c.Policy.Overrides
		for _, r := range rules {
			if r.HTTPS != nil && r.HTTPS.FakeCount != nil && *r.HTTPS.FakeCount > 0 {
				return true
			}

			if r.UDP != nil && r.UDP.FakeCount != nil && *r.UDP.FakeCount > 0 {
				return true
			}
		}
	}

	return false
}

func getDefault() *Config { //exhaustruct:enforce
	return &Config{
		App: &AppOptions{
			LogLevel:         lo.ToPtr(zerolog.InfoLevel),
			Silent:           lo.ToPtr(false),
			SetNetworkConfig: lo.ToPtr(false),
			Mode:             lo.ToPtr(AppModeHTTP),
			ListenAddr:       nil,
		},
		Conn: &ConnOptions{
			DefaultFakeTTL: lo.ToPtr(uint8(8)),
			DNSTimeout:     lo.ToPtr(time.Duration(5000) * time.Millisecond),
			TCPTimeout:     lo.ToPtr(time.Duration(10000) * time.Millisecond),
			UDPIdleTimeout: lo.ToPtr(time.Duration(25000) * time.Millisecond),
		},
		DNS: &DNSOptions{
			Mode:     lo.ToPtr(DNSModeUDP),
			Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53, Zone: ""},
			HTTPSURL: lo.ToPtr("https://dns.google/dns-query"),
			QType:    lo.ToPtr(DNSQueryIPv4),
			Cache:    lo.ToPtr(false),
		},
		HTTPS: &HTTPSOptions{
			Disorder:           lo.ToPtr(false),
			FakeCount:          lo.ToPtr(uint8(0)),
			FakePacket:         proto.NewFakeTLSMessage([]byte(FakeClientHello)),
			SplitMode:          lo.ToPtr(HTTPSSplitModeSNI),
			ChunkSize:          lo.ToPtr(uint8(35)),
			CustomSegmentPlans: []SegmentPlan{},
			Skip:               lo.ToPtr(false),
		},
		UDP: &UDPOptions{
			FakeCount:  lo.ToPtr(0),
			FakePacket: make([]byte, 64),
		},
		Policy: &PolicyOptions{
			Auto:      lo.ToPtr(false),
			Template:  &Rule{},
			Overrides: []Rule{},
		},
	}
}
