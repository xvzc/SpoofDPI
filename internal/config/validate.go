package config

import (
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"

	"github.com/rs/zerolog"
)

func validateUint8(v int) error {
	if v < 0 || math.MaxUint8 < v {
		return fmt.Errorf("out of range[%d-%d]", 0, math.MaxUint8)
	}
	return nil
}

func validateUint16(v int) error {
	if v < 0 || math.MaxUint16 < v {
		return fmt.Errorf("out of range[%d-%d]", 0, math.MaxUint16)
	}

	return nil
}

func validateLogLevel(v string) error {
	_, err := zerolog.ParseLevel(v)
	if err != nil {
		return fmt.Errorf("invalid level string %s", v)
	}

	return nil
}

func validatePolicy(v string) error {
	rs := `^([i|x]):([a-zA-Z0-9\-]+|\*{2}|\*)(?:\.([a-zA-Z0-9\-]+|\*{2}|\*))*$`
	r, err := regexp.Compile(rs)
	if err != nil {
		return err
	}

	if !r.MatchString(v) {
		return fmt.Errorf("wrong formatted policy")
	}

	return nil
}

func validateDNSDefaultMode(v string) error {
	if v == "udp" {
		return nil
	}

	if v == "doh" {
		return nil
	}

	if v == "sys" {
		return nil
	}

	return fmt.Errorf("wrong value '%s' for default dns mode", v)
}

func validateDNSMode(v string) error {
	if v == "map" {
		return nil
	}

	return validateDNSDefaultMode(v)
}

func validateDNSQueryType(v string) error {
	if v == "ipv4" {
		return nil
	}

	if v == "ipv6" {
		return nil
	}

	if v == "all" {
		return nil
	}

	return fmt.Errorf("wrong value '%s' for dns query type", v)
}

func validateHostPort(v string) error {
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address format")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	return validateUint16(portInt)
}

func validateHTTPSEndpoint(v string) error {
	if v != "" {
		if ok, err := regexp.MatchString("^https?://", v); !ok ||
			err != nil {
			return fmt.Errorf("should start with 'https://'")
		}
	}

	return nil
}
