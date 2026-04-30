package config

import (
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v3"
)

// Load assembles the effective Config from three sources, in precedence
// order: defaults → TOML → CLI flags. After all layers are merged it
// runs Finalize (cross-field defaults + rule resolution) and Validate
// (semantic checks). Returns the resolved Config and the directory of
// the TOML file that was used (or "" if none).
//
// argsCfg holds the CLI overrides accumulated by Flag.Action callbacks
// during cmd.Run; only fields whose flags are reported set by cmd.IsSet
// are copied onto the merged Config.
func Load(cmd *cli.Command, argsCfg *Config) (*Config, string, error) {
	cfg := DefaultConfig()

	configDir, err := loadTOML(cmd, cfg)
	if err != nil {
		return nil, "", err
	}

	applyCLIOverrides(cfg, cmd, argsCfg)

	if err := cfg.Finalize(); err != nil {
		return nil, "", err
	}
	// Validate is wired in a follow-up commit.

	return cfg, configDir, nil
}

// loadTOML resolves the TOML config path (custom --config flag, env var,
// or one of the default locations), then decodes it onto cfg if found.
// The decode-into-defaults trick preserves cfg's pre-populated values
// for any TOML key that's absent. Returns the directory of the decoded
// file (or "" if none was loaded).
func loadTOML(cmd *cli.Command, cfg *Config) (string, error) {
	if cmd.Bool("clean") {
		return "", nil
	}

	const configFilename = "spoofdpi.toml"
	configDirs := []string{
		path.Join(string(os.PathSeparator), "etc", configFilename),
		path.Join(os.Getenv("XDG_CONFIG_HOME"), "spoofdpi", configFilename),
		path.Join(determineRealHome(), ".config", "spoofdpi", configFilename),
	}

	configPath, err := searchTomlFile(cmd.String("config"), configDirs)
	if err != nil {
		return "", err
	}
	if configPath == "" {
		return "", nil
	}

	if err := fromTomlFile(configPath, cfg); err != nil {
		return "", fmt.Errorf("error parsing '%s': %w", configPath, err)
	}
	return configPath, nil
}
