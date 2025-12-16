//go:build !darwin && !linux

package system

import "github.com/rs/zerolog"

func SetProxy(logger zerolog.Logger, port uint16) error {
	return nil
}

func UnsetProxy(logger zerolog.Logger) error {
	return nil
}
