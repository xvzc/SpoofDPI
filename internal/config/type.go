package config

import (
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
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
// │ RegexPattern │
// └──────────────┘
type RegexPattern struct {
	value *regexp.Regexp
}

func (r *RegexPattern) Value() *regexp.Regexp {
	return r.value
}

func (r *RegexPattern) UnmarshalText(text []byte) error {
	s := string(text)
	err := validateRegexpPattern(s)
	if err != nil {
		return err
	}

	r.value = regexp.MustCompile(s)

	return nil
}

func ParseRegexPatterns(ss []string) []RegexPattern {
	var ret []RegexPattern
	for i := range ss {
		ret = append(ret, RegexPattern{regexp.MustCompile(ss[i])})
	}

	return ret
}

func ParseRegexpSlices(rs []RegexPattern) []*regexp.Regexp {
	var ret []*regexp.Regexp
	for i := range rs {
		ret = append(ret, rs[i].Value())
	}

	return ret
}
