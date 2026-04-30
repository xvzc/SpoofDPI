package config

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/proto"
)

type Config struct {
	App    AppOptions    `toml:"app"`
	Conn   ConnOptions   `toml:"connection"`
	DNS    DNSOptions    `toml:"dns"`
	HTTPS  HTTPSOptions  `toml:"https"`
	UDP    UDPOptions    `toml:"udp"`
	Policy PolicyOptions `toml:"policy"`
}

func (c *Config) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type config file")
	}

	if app := findStructFrom[AppOptions](m, "app", &err); app != nil {
		c.App = *app
	}
	if conn := findStructFrom[ConnOptions](m, "connection", &err); conn != nil {
		c.Conn = *conn
	}
	if dns := findStructFrom[DNSOptions](m, "dns", &err); dns != nil {
		c.DNS = *dns
	}
	if https := findStructFrom[HTTPSOptions](m, "https", &err); https != nil {
		c.HTTPS = *https
	}
	if udp := findStructFrom[UDPOptions](m, "udp", &err); udp != nil {
		c.UDP = *udp
	}
	if policy := findStructFrom[PolicyOptions](m, "policy", &err); policy != nil {
		c.Policy = *policy
	}

	return
}

func (c *Config) ShouldEnablePcap() bool {
	if c.HTTPS.FakeCount > 0 {
		return true
	}
	if c.UDP.FakeCount > 0 {
		return true
	}
	for _, r := range c.Policy.Overrides {
		if r.HTTPS.FakeCount > 0 {
			return true
		}
		if r.UDP.FakeCount > 0 {
			return true
		}
	}
	return false
}

// DefaultConfig returns a fully-populated Config with default values for
// every field across every section. Used as the starting point of the
// load pipeline (defaults → TOML → CLI → Finalize → Validate).
func DefaultConfig() *Config { //exhaustruct:enforce
	return &Config{
		App:    DefaultAppOptions(),
		Conn:   DefaultConnOptions(),
		DNS:    DefaultDNSOptions(),
		HTTPS:  DefaultHTTPSOptions(),
		UDP:    DefaultUDPOptions(),
		Policy: DefaultPolicyOptions(),
	}
}

// DefaultAppOptions returns the default values for the [app] section.
func DefaultAppOptions() AppOptions { //exhaustruct:enforce
	return AppOptions{
		NoTUI:                false,
		LogLevel:             zerolog.InfoLevel,
		Silent:               false,
		AutoConfigureNetwork: false,
		Mode:                 AppModeHTTP,
		ListenAddr:           net.TCPAddr{},
		FreebsdFIB:           1,
	}
}

// DefaultConnOptions returns the default values for the [connection] section.
func DefaultConnOptions() ConnOptions { //exhaustruct:enforce
	return ConnOptions{
		DefaultFakeTTL: 8,
		DNSTimeout:     5000 * time.Millisecond,
		TCPTimeout:     10000 * time.Millisecond,
		UDPIdleTimeout: 25000 * time.Millisecond,
	}
}

// DefaultDNSOptions returns the default values for the [dns] section.
func DefaultDNSOptions() DNSOptions { //exhaustruct:enforce
	return DNSOptions{
		Mode:     DNSModeUDP,
		Addr:     net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53, Zone: ""},
		HTTPSURL: "https://dns.google/dns-query",
		QType:    DNSQueryIPv4,
		Cache:    false,
	}
}

// DefaultHTTPSOptions returns the default values for the [https] section.
func DefaultHTTPSOptions() HTTPSOptions { //exhaustruct:enforce
	return HTTPSOptions{
		Disorder:           false,
		FakeCount:          0,
		FakePacket:         proto.NewFakeTLSMessage([]byte(FakeClientHello)),
		SplitMode:          HTTPSSplitModeSNI,
		ChunkSize:          35,
		CustomSegmentPlans: []SegmentPlan{},
		Skip:               false,
	}
}

// DefaultUDPOptions returns the default values for the [udp] section.
func DefaultUDPOptions() UDPOptions { //exhaustruct:enforce
	return UDPOptions{
		FakeCount:  0,
		FakePacket: make([]byte, 64),
	}
}

// DefaultPolicyOptions returns the default values for the [policy] section.
func DefaultPolicyOptions() PolicyOptions { //exhaustruct:enforce
	return PolicyOptions{
		Overrides: []Rule{},
	}
}

// Finalize applies defaults that depend on other fields. Called after
// defaults+TOML+CLI layers are merged, before Validate.
func (c *Config) Finalize() {
	if c.App.ListenAddr.IP == nil && c.App.ListenAddr.Port == 0 {
		port := 8080
		if c.App.Mode == AppModeSOCKS5 {
			port = 1080
		}
		c.App.ListenAddr = net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: port,
		}
	}
}
