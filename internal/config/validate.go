package config

import (
	"fmt"
	"math"
	"net"
	"regexp"

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
	rs := `^(i|x):((\*|[a-zA-Z0-9-]+)\.)*[a-zA-Z0-9-]+\.[a-zA-Z0-9-]+$`
	r, err := regexp.Compile(rs)
	if err != nil {
		return fmt.Errorf("wrong format")
	}

	if !r.MatchString(v) {
		return fmt.Errorf("wrong format")
	}

	return nil
}

func validateIPAddr(v string) error {
	ip := net.ParseIP(v)
	if ip == nil {
		return fmt.Errorf("wrong format")
	}

	return nil
}

func validateHTTPSEndpoint(v string) error {
	if v != "" {
		if ok, err := regexp.MatchString("^https?://", v); !ok ||
			err != nil {
			return fmt.Errorf("should start with 'https://")
		}
	}

	return nil
}
