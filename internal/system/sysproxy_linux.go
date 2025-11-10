//go:build linux

package system

import "github.com/rs/zerolog"

func SetProxy(logger zerolog.Logger, port uint16) error {
	logger.Info().Msgf("automatic system-wide proxy setup is not implemented on Linux")
	return nil
}

func UnsetProxy(logger zerolog.Logger) error {
	return nil
}
