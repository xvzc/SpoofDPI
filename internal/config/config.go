package config

import (
	"reflect"
	"strings"

	"github.com/miekg/dns"
)

type Config struct {
	AutoPolicy        bool           `toml:"auto-policy"`
	CacheShards       Uint8Number    `toml:"cache-shards"`
	DefaultTTL        Uint8Number    `toml:"default-ttl"`
	Disorder          bool           `toml:"disorder"`
	DNSAddr           HostPort       `toml:"dns-addr"`
	DNSDefault        DNSMode        `toml:"dns-default"`
	DNSQueryType      DNSQueryType   `toml:"dns-qtype"`
	DOHURL            HTTPSEndpoint  `toml:"doh-url"`
	FakeCount         Uint8Number    `toml:"fake-count"`
	ListenAddr        HostPort       `toml:"listen-addr"`
	LogLevel          LogLevel       `toml:"log-level"`
	DomainPolicySlice []DomainPolicy `toml:"policy"`
	SetSystemProxy    bool           `toml:"system-proxy"`
	Silent            bool           `toml:"silent"`
	Timeout           Uint16Number   `toml:"timeout"`
	WindowSize        Uint8Number    `toml:"window-size"`
}

func (c *Config) GenerateDnsQueryTypes() []uint16 {
	switch c.DNSQueryType.Value {
	case "ipv4":
		return []uint16{dns.TypeA}
	case "ipv6":
		return []uint16{dns.TypeAAAA}
	case "all":
		return []uint16{dns.TypeA, dns.TypeAAAA}
	default:
		return []uint16{}
	}
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
