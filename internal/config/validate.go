package config

import (
	"fmt"
	"math"
	"net"
	"regexp"
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

func validateRegexpPattern(p string) (err error) {
	_, err = regexp.Compile(p)
	if err != nil {
		return fmt.Errorf("failed to compile regexp")
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
