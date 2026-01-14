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
	General *GeneralOptions `toml:"general"`
	Server  *ServerOptions  `toml:"server"`
	DNS     *DNSOptions     `toml:"dns"`
	HTTPS   *HTTPSOptions   `toml:"https"`
	UDP     *UDPOptions     `toml:"udp"`
	Policy  *PolicyOptions  `toml:"policy"`
}

func (c *Config) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type config file")
	}

	c.General = findStructFrom[GeneralOptions](m, "general", &err)
	c.Server = findStructFrom[ServerOptions](m, "server", &err)
	c.DNS = findStructFrom[DNSOptions](m, "dns", &err)
	c.HTTPS = findStructFrom[HTTPSOptions](m, "https", &err)
	c.UDP = findStructFrom[UDPOptions](m, "udp", &err)
	c.Policy = findStructFrom[PolicyOptions](m, "policy", &err)

	return
}

func NewConfig() *Config {
	return &Config{
		General: &GeneralOptions{},
		Server:  &ServerOptions{},
		DNS:     &DNSOptions{},
		HTTPS:   &HTTPSOptions{},
		UDP:     &UDPOptions{},
		Policy:  &PolicyOptions{},
	}
}

func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	return &Config{
		General: c.General.Clone(),
		Server:  c.Server.Clone(),
		DNS:     c.DNS.Clone(),
		HTTPS:   c.HTTPS.Clone(),
		UDP:     c.UDP.Clone(),
		Policy:  c.Policy.Clone(),
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
		General: origin.General.Merge(overrides.General),
		Server:  origin.Server.Merge(overrides.Server),
		DNS:     origin.DNS.Merge(overrides.DNS),
		HTTPS:   origin.HTTPS.Merge(overrides.HTTPS),
		UDP:     origin.UDP.Merge(overrides.UDP),
		Policy:  origin.Policy.Merge(overrides.Policy),
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
		General: &GeneralOptions{
			LogLevel:         lo.ToPtr(zerolog.InfoLevel),
			Silent:           lo.ToPtr(false),
			SetNetworkConfig: lo.ToPtr(false),
		},
		Server: &ServerOptions{
			Mode:       lo.ToPtr(ServerModeHTTP),
			DefaultTTL: lo.ToPtr(uint8(64)),
			ListenAddr: nil,
			Timeout:    lo.ToPtr(time.Duration(0)),
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
			Timeout:    lo.ToPtr(time.Duration(0)),
		},
		Policy: &PolicyOptions{
			Auto:      lo.ToPtr(false),
			Template:  &Rule{},
			Overrides: []Rule{},
		},
	}
}
