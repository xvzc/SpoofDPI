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
	value uint8
}

func (u *Uint8Number) Value() uint8 {
	return u.value
}

func (u *Uint8Number) UnmarshalText(text []byte) error {
	v, err := strconv.Atoi(string(text))
	if err != nil {
		return fmt.Errorf("wrong value")
	}

	if v < 0 || math.MaxUint8 < v {
		return fmt.Errorf("out of range[%d-%d]", 0, math.MaxUint8)
	}

	u.value = uint8(v)

	return nil
}

type LogLevel struct {
	value string
}

func (l *LogLevel) Value() string {
	return l.value
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	if err := validateLogLevel(string(text)); err != nil {
		return err
	}

	l.value = string(text)

	return nil
}

// ┌────────┐
// │ UINT16 │
// └────────┘
type Uint16Number struct {
	value uint16
}

func (u *Uint16Number) Value() uint16 {
	return u.value
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

	u.value = uint16(v)

	return nil
}

// ┌────────────┐
// │ IP_ADDRESS │
// └────────────┘
type IPAddress struct {
	value net.IP
}

func (a *IPAddress) Value() net.IP {
	return a.value
}

func (c *IPAddress) UnmarshalText(text []byte) error {
	s := string(text)
	err := validateIPAddr(s)
	if err != nil {
		return err
	}

	c.value = net.ParseIP(s)

	return nil
}

// ┌────────────────┐
// │ HTTPS_ENDPOINT │
// └────────────────┘
type HTTPSEndpoint struct {
	value string
}

func (e *HTTPSEndpoint) Value() string {
	return e.value
}

func (e *HTTPSEndpoint) UnmarshalText(text []byte) error {
	err := validateHTTPSEndpoint(string(text))
	if err != nil {
		return err
	}

	e.value = string(text)

	return nil
}

// ┌──────────────┐
// │ DomainPolicy │
// └──────────────┘
// DomainPolicyAction defines the type of action to take for a domain.
type DomainPolicy struct {
	value   string
	include bool
}

func (dp *DomainPolicy) Value() string {
	return dp.value
}

func (dp *DomainPolicy) IsIncluded() bool {
	return dp.include
}

func (dp *DomainPolicy) UnmarshalText(text []byte) error {
	s := string(text)
	err := validatePolicy(s)
	if err != nil {
		return err
	}

	parsed := parseDomainPolicy(string(text))

	dp.value = parsed.Value()
	dp.include = parsed.IsIncluded()

	return nil
}

func parseDomainPolicy(s string) DomainPolicy {
	prefix, value := string(s[0]), s[2:]

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
		value:   value,
		include: action,
	}
}

func parseDomainPolicySlice(ss []string) []DomainPolicy {
	var ret []DomainPolicy
	for _, s := range ss {
		ret = append(ret, parseDomainPolicy(s))
	}

	return ret
}

func ParseDomainSearchTree(ps []DomainPolicy) tree.RadixTree {
	rt := tree.NewDomainSearchTree()
	for _, p := range ps {
		rt.Insert(p.Value(), p.IsIncluded())
	}

	return rt
}
