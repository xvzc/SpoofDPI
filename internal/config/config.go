package config

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/ptr"
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
	c.Policy = findStructFrom[PolicyOptions](m, "policy", &err)

	return
}

func NewConfig() *Config {
	return &Config{
		General: &GeneralOptions{},
		Server:  &ServerOptions{},
		DNS:     &DNSOptions{},
		HTTPS:   &HTTPSOptions{},
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
		Policy:  origin.Policy.Merge(overrides.Policy),
	}
}

func (c *Config) ShouldEnablePcap() bool {
	if *c.HTTPS.FakeCount > 0 {
		return true
	}

	if c.Policy == nil {
		return false
	}

	if c.Policy.Template != nil {
		template := c.Policy.Template
		if template.HTTPS != nil && ptr.FromPtr(template.HTTPS.FakeCount) > 0 {
			return true
		}
	}

	if c.Policy.Overrides != nil {
		rules := c.Policy.Overrides
		for _, r := range rules {
			if r.HTTPS == nil {
				continue
			}

			if r.HTTPS.FakeCount == nil {
				continue
			}

			if *r.HTTPS.FakeCount > 0 {
				return true
			}
		}
	}

	return false
}

func getDefault() *Config { //exhaustruct:enforce
	return &Config{
		General: &GeneralOptions{
			LogLevel:       ptr.FromValue(zerolog.InfoLevel),
			Silent:         ptr.FromValue(false),
			SetSystemProxy: ptr.FromValue(false),
		},
		Server: &ServerOptions{
			DefaultTTL: ptr.FromValue(uint8(64)),
			ListenAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080, Zone: ""},
			Timeout:    ptr.FromValue(time.Duration(0)),
		},
		DNS: &DNSOptions{
			Mode:     ptr.FromValue(DNSModeUDP),
			Addr:     &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53, Zone: ""},
			HTTPSURL: ptr.FromValue("https://dns.google/dns-query"),
			QType:    ptr.FromValue(DNSQueryIPv4),
			Cache:    ptr.FromValue(false),
		},
		HTTPS: &HTTPSOptions{
			Disorder:   ptr.FromValue(false),
			FakeCount:  ptr.FromValue(uint8(0)),
			FakePacket: []byte(FakeClientHello),
			SplitMode:  ptr.FromValue(HTTPSSplitModeSNI),
			ChunkSize:  ptr.FromValue(uint8(0)),
			Skip:       ptr.FromValue(false),
		},
		Policy: &PolicyOptions{
			Auto:      ptr.FromValue(false),
			Template:  &Rule{},
			Overrides: []Rule{},
		},
	}
}
