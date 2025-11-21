package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/miekg/dns"
)

type Config struct {
	AutoPolicy        bool           `toml:"auto-policy"`
	CacheShards       Uint8Number    `toml:"cache-shards"`
	DnsAddr           IPAddress      `toml:"dns-addr"`
	DnsIPv4Only       bool           `toml:"dns-ipv4-only"`
	DnsPort           Uint16Number   `toml:"dns-port"`
	DOHEndpoint       HTTPSEndpoint  `toml:"doh-endpoint"`
	EnableDOH         bool           `toml:"endble-doh"`
	FakeHTTPSPackets  Uint8Number    `toml:"fake-https-packets"`
	ListenAddr        IPAddress      `toml:"listen-addr"`
	ListenPort        Uint16Number   `toml:"listen-port"`
	LogLevel          LogLevel       `toml:"log-level"`
	DomainPolicySlice []DomainPolicy `toml:"policy"`
	SetSystemProxy    bool           `toml:"system-proxy"`
	Silent            bool           `toml:"silent"`
	Timeout           Uint16Number   `toml:"timeout"`
	WindowSize        Uint8Number    `toml:"window-size"`
}

func (c *Config) GenerateDnsQueryTypes() []uint16 {
	if c.DnsIPv4Only {
		return []uint16{dns.TypeA}
	} else {
		return []uint16{dns.TypeA, dns.TypeAAAA}
	}
}

func (c *Config) GenerateDOHEndpoint() string {
	if c.DOHEndpoint.Value() == "" {
		return fmt.Sprintf("https://%s/dns-query", c.DnsAddr.value.String())
	} else {
		return c.DOHEndpoint.Value()
	}
}

func (c *Config) ShouldEnableDOH() bool {
	return c.EnableDOH || (c.DOHEndpoint.Value() != "")
}

func mergeConfig(argsCfg *Config, tomlCfg *Config, args []string) *Config {
	final := tomlCfg

	finalVal := reflect.ValueOf(final).Elem()
	argsVal := reflect.ValueOf(argsCfg).Elem()
	structType := finalVal.Type()

	for i := 0; i < finalVal.NumField(); i++ {
		tag := structType.Field(i).Tag.Get("toml")

		finalField := finalVal.Field(i)
		argsField := argsVal.Field(i)

		if finalField.CanSet() && finalField.IsZero() {
			finalField.Set(argsField)
		}

		for i := range args {
			if strings.Contains(args[i], tag) {
				finalField.Set(argsField)
				break
			}
		}
	}

	return final
}
