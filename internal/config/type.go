package config

import (
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/xvzc/SpoofDPI/internal/datastruct/tree"
)

// ┌───────┐
// │ UINT8 │
// └───────┘
type Uint8Number struct {
	Value uint8
}

func (u *Uint8Number) UnmarshalText(text []byte) error {
	v, err := strconv.Atoi(string(text))
	if err != nil {
		return fmt.Errorf("wrong value")
	}

	if v < 0 || math.MaxUint8 < v {
		return fmt.Errorf("out of range[%d-%d]", 0, math.MaxUint8)
	}

	u.Value = uint8(v)

	return nil
}

// ┌────────┐
// │ UINT16 │
// └────────┘
type Uint16Number struct {
	Value uint16
}

func (u *Uint16Number) UnmarshalText(text []byte) error {
	v, err := strconv.Atoi(string(text))
	if err != nil {
		return fmt.Errorf("wrong value")
	}
	err = validateUint16(v)
	if err != nil {
		return err
	}

	u.Value = uint16(v)

	return nil
}

// ┌───────────┐
// │ LOG_LEVEL │
// └───────────┘
type LogLevel struct {
	Value string
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	if err := validateLogLevel(string(text)); err != nil {
		return err
	}

	l.Value = string(text)

	return nil
}

func (l *LogLevel) String() string {
	return l.Value
}

// ┌─────────────┐
// │ TCP_ADDRESS │
// └─────────────┘
type HostPort struct {
	net.TCPAddr
}

func (a *HostPort) UnmarshalText(text []byte) error {
	s := string(text)
	err := validateHostPort(s)
	if err != nil {
		return err
	}

	host, port, _ := net.SplitHostPort(s)
	a.IP = net.ParseIP(host)
	a.Port, _ = strconv.Atoi(port)

	return nil
}

// ┌──────────┐
// │ DNS_MODE │
// └──────────┘
type DNSMode struct {
	Value string
}

func (a *DNSMode) UnmarshalText(text []byte) error {
	if err := validateDNSDefaultMode(string(text)); err != nil {
		return err
	}

	a.Value = string(text)

	return nil
}

// ┌────────────────┐
// │ DNS_QUERY_TYPE │
// └────────────────┘
type DNSQueryType struct {
	Value string
}

func (a *DNSQueryType) UnmarshalText(text []byte) error {
	if err := validateDNSQueryType(string(text)); err != nil {
		return err
	}

	a.Value = string(text)

	return nil
}

// ┌────────────────┐
// │ HTTPS_ENDPOINT │
// └────────────────┘
type HTTPSEndpoint struct {
	Value string
}

func (e *HTTPSEndpoint) UnmarshalText(text []byte) error {
	err := validateHTTPSEndpoint(string(text))
	if err != nil {
		return err
	}

	e.Value = string(text)

	return nil
}

// ┌──────────────┐
// │ DomainPolicy │
// └──────────────┘
// DomainPolicyAction defines the type of action to take for a domain.
type DomainPolicy struct {
	Rule    string
	Include bool
}

func (dp *DomainPolicy) UnmarshalText(text []byte) error {
	s := string(text)
	err := validatePolicy(s)
	if err != nil {
		return err
	}

	parsed := parseDomainPolicy(string(text))

	dp.Rule = parsed.Rule
	dp.Include = parsed.Include

	return nil
}

func parseDomainPolicy(s string) DomainPolicy {
	prefix, rule := string(s[0]), s[2:]

	var action bool
	switch prefix {
	case "i":
		action = true
	case "x":
		action = false
	default:
		action = false
	}

	return DomainPolicy{
		Rule:    rule,
		Include: action,
	}
}

func parseDomainPolicySlice(ss []string) []DomainPolicy {
	var ret []DomainPolicy
	for _, s := range ss {
		ret = append(ret, parseDomainPolicy(s))
	}

	return ret
}

func ParseDomainSearchTree(ps []DomainPolicy) tree.SearchTree {
	rt := tree.NewDomainSearchTree()
	for _, p := range ps {
		rt.Insert(p.Rule, p.Include)
	}

	return rt
}
