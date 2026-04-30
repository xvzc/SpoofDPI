package config

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/proto"
)

// Config groups configuration by lifecycle:
//
//   - Startup is read once at boot (logger setup, mode selection, building
//     the matcher from policy overrides) and is not needed at request time.
//   - Runtime is read on every request (per-traffic decisions about DPI
//     bypass, DNS resolution, timeouts) and travels with handlers.
type Config struct {
	Startup StartupConfig
	Runtime RuntimeConfig

	// WarnMsgs accumulates non-fatal advisories produced during Load
	// (deprecations, transitional behaviors, etc.). main surfaces them
	// through the configured logger after TUI/log setup, so they don't
	// race the TUI taking over stdout/stderr.
	WarnMsgs []string
}

// StartupConfig holds the sections consumed only during server bootstrap.
// After the server is up and the matcher is built, this can be discarded.
type StartupConfig struct {
	App    AppOptions
	Policy PolicyOptions
}

// RuntimeConfig holds the sections accessed on the request hot path.
// Handlers take a pointer to RuntimeConfig (not the individual *XOptions)
// so adding a new section doesn't require touching every signature, and
// rule overrides can swap the whole RuntimeConfig in a single assignment.
type RuntimeConfig struct {
	Conn  ConnOptions
	DNS   DNSOptions
	HTTPS HTTPSOptions
	UDP   UDPOptions
}

func (c *Config) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type config file")
	}

	if app := findStructFrom[AppOptions](m, "app", &err); app != nil {
		c.Startup.App = *app
	}
	if conn := findStructFrom[ConnOptions](m, "connection", &err); conn != nil {
		c.Runtime.Conn = *conn
	}
	if dns := findStructFrom[DNSOptions](m, "dns", &err); dns != nil {
		c.Runtime.DNS = *dns
	}
	if https := findStructFrom[HTTPSOptions](m, "https", &err); https != nil {
		c.Runtime.HTTPS = *https
	}
	if udp := findStructFrom[UDPOptions](m, "udp", &err); udp != nil {
		c.Runtime.UDP = *udp
	}
	if policy := findStructFrom[PolicyOptions](m, "policy", &err); policy != nil {
		c.Startup.Policy = *policy
	}

	if policyMap, ok := m["policy"].(map[string]any); ok {
		if _, hasTemplate := policyMap["template"]; hasTemplate {
			c.WarnMsgs = append(
				c.WarnMsgs,
				"'policy.template' is deprecated and ignored; move template fields to top-level [app]/[connection]/[dns]/[https]/[udp] sections",
			)
		}
	}

	return
}

func (c *Config) ShouldEnablePcap() bool {
	if c.Runtime.HTTPS.FakeCount > 0 {
		return true
	}
	if c.Runtime.UDP.FakeCount > 0 {
		return true
	}
	for _, r := range c.Startup.Policy.Overrides {
		if r.Runtime.HTTPS.FakeCount > 0 {
			return true
		}
		if r.Runtime.UDP.FakeCount > 0 {
			return true
		}
	}
	return false
}

// DefaultConfig returns a fully-populated Config with default values for
// every field. Used as the starting point of the load pipeline
// (defaults → TOML → CLI → Finalize → Validate).
func DefaultConfig() *Config { //exhaustruct:enforce
	return &Config{
		Startup:  DefaultStartupConfig(),
		Runtime:  DefaultRuntimeConfig(),
		WarnMsgs: nil,
	}
}

// DefaultStartupConfig returns the default startup-time configuration.
func DefaultStartupConfig() StartupConfig { //exhaustruct:enforce
	return StartupConfig{
		App:    DefaultAppOptions(),
		Policy: DefaultPolicyOptions(),
	}
}

// DefaultRuntimeConfig returns the default runtime configuration.
func DefaultRuntimeConfig() RuntimeConfig { //exhaustruct:enforce
	return RuntimeConfig{
		Conn:  DefaultConnOptions(),
		DNS:   DefaultDNSOptions(),
		HTTPS: DefaultHTTPSOptions(),
		UDP:   DefaultUDPOptions(),
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

// Finalize applies defaults that depend on other fields (e.g. ListenAddr
// per Mode). Called after defaults+TOML+CLI layers are merged, before
// Validate. Rule resolution is handled separately by Load via
// resolveRules so PolicyOptions doesn't need to carry load-time scratch
// state into the runtime config.
func (c *Config) Finalize() error {
	if c.Startup.App.ListenAddr.IP == nil && c.Startup.App.ListenAddr.Port == 0 {
		port := 8080
		if c.Startup.App.Mode == AppModeSOCKS5 {
			port = 1080
		}
		c.Startup.App.ListenAddr = net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: port,
		}
	}
	return nil
}
