//go:build !windows && !darwin && !linux

package system

import "github.com/rs/zerolog"

func SetProxy(port uint16, logger zerolog.Logger) error {
	return nil
}

func UnsetProxy(logger zerolog.Logger) error {
	return nil
}
