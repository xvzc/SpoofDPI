package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

func parseTomlConfig(path string) (*Config, error) {
	var cfg *Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func readFirstFoundConfigFiles(customDir string, lookupDirs []string) (*Config, error) {
	if customDir != "" {
		if _, err := os.Stat(customDir); err != nil { // Path exists
			return nil, err
		}

		return parseTomlConfig(customDir)
	}

	for _, p := range lookupDirs {
		if p == "" {
			continue
		}

		if _, err := os.Stat(p); err == nil { // Path exists
			return parseTomlConfig(p)
		}
	}

	// Don't care even if any configuration file in lookupDirs is not found
	return nil, nil
}
