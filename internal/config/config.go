package config

import (
	"math"
	"net"
	"regexp"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

type Config struct {
	allowedPatterns  []*regexp.Regexp
	cacheShards      uint64
	debug            bool
	dnsAddr          net.IP
	dnsPort          uint16
	dnsQueryTypes    []uint16
	dohEndpoint      string
	enableDOH        bool
	fakeHTTPSPackets uint8
	listenAddr       net.IP
	listenPort       uint16
	setSystemProxy   bool
	silent           bool
	timeout          uint16
	windowSize       uint16
}

func LoadConfigurationFromArgs(args *Args, logger zerolog.Logger) *Config {
	dnsAddr := net.ParseIP(args.DnsAddr)
	if dnsAddr == nil {
		logger.Fatal().Msgf("invalid dns addr: %s", args.DnsAddr)
	}

	listenAddr := net.ParseIP(args.ListenAddr)
	if listenAddr == nil {
		logger.Fatal().Msgf("invalid listen addr: %s", args.ListenAddr)
	}

	if args.ListenPort > math.MaxUint16 {
		logger.Fatal().Msgf("listen-port value %d is out of range", args.ListenPort)
	}

	if args.DnsPort > math.MaxUint16 {
		logger.Fatal().Msgf("dns-port value %d is out of range", args.DnsPort)
	}

	if args.Timeout > math.MaxUint16 {
		logger.Fatal().Msgf("timeout value %d is out of range", args.Timeout)
	}

	if args.WindowSize > math.MaxUint16 {
		logger.Fatal().Msgf("window-size value %d is out of range", args.WindowSize)
	}

	if args.CacheShards < 1 || args.CacheShards > 256 {
		logger.Fatal().
			Msgf("cache-shards value %d is out of range, it must be between 1 and 256", args.CacheShards)
	}

	if args.FakeHTTPSPackets > 50 {
		logger.Fatal().
			Msgf("fake-https-packets value %d is out of range, it must be between 0 and 50", args.FakeHTTPSPackets)
	}

	if args.DOHEndpoint != "" {
		if ok, err := regexp.MatchString("^https?://", args.DOHEndpoint); !ok ||
			err != nil {
			logger.Fatal().
				Msgf("doh-enpoint value should be https scheme: '%s' does not start with 'https://'", args.DOHEndpoint)
		}
	}

	cfg := &Config{
		allowedPatterns:  parseAllowedPatterns(args.AllowedPattern),
		cacheShards:      uint64(args.CacheShards),
		debug:            args.Debug,
		dnsAddr:          dnsAddr,
		dnsPort:          uint16(args.DnsPort),
		enableDOH:        args.EnableDOH,
		listenAddr:       listenAddr,
		listenPort:       uint16(args.ListenPort),
		setSystemProxy:   args.SystemProxy,
		silent:           args.Silent,
		timeout:          uint16(args.Timeout),
		windowSize:       uint16(args.WindowSize),
		fakeHTTPSPackets: uint8(args.FakeHTTPSPackets),
	}

	if args.DnsIPv4Only {
		cfg.dnsQueryTypes = []uint16{dns.TypeA}
	} else {
		cfg.dnsQueryTypes = []uint16{dns.TypeA, dns.TypeAAAA}
	}

	return cfg
}

func (c *Config) AllowedPatterns() []*regexp.Regexp {
	return c.allowedPatterns
}

func (c *Config) CacheShards() uint64 {
	return c.cacheShards
}

func (c *Config) Debug() bool {
	return c.debug
}

func (c *Config) DnsAddr() net.IP {
	return c.dnsAddr
}

func (c *Config) DnsPort() uint16 {
	return c.dnsPort
}

func (c *Config) DnsQueryTypes() []uint16 {
	return c.dnsQueryTypes
}

func (c *Config) DOHEndpoint() string {
	return c.dohEndpoint
}

func (c *Config) EnableDOH() bool {
	return c.enableDOH
}

func (c *Config) FakeHTTPSPackets() uint8 {
	return c.fakeHTTPSPackets
}

func (c *Config) ListenAddr() net.IP {
	return c.listenAddr
}

func (c *Config) ListenPort() uint16 {
	return c.listenPort
}

func (c *Config) SetSystemProxy() bool {
	return c.setSystemProxy
}

func (c *Config) Silent() bool {
	return c.silent
}

func (c *Config) Timeout() time.Duration {
	return time.Duration(c.timeout) * time.Millisecond
}

func (c *Config) WindowSize() uint16 {
	return c.windowSize
}
