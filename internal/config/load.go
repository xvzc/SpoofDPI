package config

import (
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v3"
)

// Load assembles the effective Config from three sources, in precedence
// order: defaults → TOML → CLI flags. After all layers are merged it
// runs Finalize (cross-field defaults), resolveRules (eager-resolve
// policy overrides on top of the finalized base RuntimeConfig), and
// Validate (semantic checks). Returns the resolved Config and the
// path of the TOML file that was used (or "" if none).
//
// cliOverrides is the slice of closures appended by Flag.Action
// callbacks during cmd.Run — one per flag the user actually set.
// Applying them after loadTOML is what makes CLI win over TOML.
func Load(cmd *cli.Command, cliOverrides []func(*Config)) (*Config, string, error) {
	cfg := DefaultConfig()

	configPath, rawRules, err := loadTOML(cmd, cfg)
	if err != nil {
		return nil, "", err
	}

	for _, apply := range cliOverrides {
		apply(cfg)
	}

	if err := cfg.Finalize(); err != nil {
		return nil, "", err
	}

	rules, err := resolveRules(rawRules, cfg.Runtime, &cfg.WarnMsgs)
	if err != nil {
		return nil, "", err
	}
	cfg.Startup.Policy.Overrides = rules

	if err := cfg.Validate(); err != nil {
		return nil, "", err
	}

	return cfg, configPath, nil
}

// loadTOML resolves the TOML config path (custom --config flag, env var,
// or one of the default locations), then decodes it onto cfg if found.
// The decode-into-defaults trick preserves cfg's pre-populated values
// for any TOML key that's absent. Also extracts the raw
// [[policy.overrides]] entries separately so resolveRules can apply
// them on top of the finalized base RuntimeConfig later. Returns the
// path of the decoded file (or "" if none was loaded) and the captured
// raw rules.
func loadTOML(cmd *cli.Command, cfg *Config) (string, []map[string]any, error) {
	if cmd.Bool("clean") {
		return "", nil, nil
	}

	const configFilename = "spoofdpi.toml"
	configDirs := []string{
		path.Join(string(os.PathSeparator), "etc", configFilename),
		path.Join(os.Getenv("XDG_CONFIG_HOME"), "spoofdpi", configFilename),
		path.Join(determineRealHome(), ".config", "spoofdpi", configFilename),
	}

	configPath, err := searchTomlFile(cmd.String("config"), configDirs)
	if err != nil {
		return "", nil, err
	}
	if configPath == "" {
		return "", nil, nil
	}

	if err := fromTomlFile(configPath, cfg); err != nil {
		return "", nil, fmt.Errorf("error parsing '%s': %w", configPath, err)
	}

	rawRules, err := extractRawOverrides(configPath)
	if err != nil {
		return "", nil, fmt.Errorf(
			"error reading policy overrides from '%s': %w",
			configPath,
			err,
		)
	}

	return configPath, rawRules, nil
}

// extractRawOverrides re-decodes the TOML file into a small helper struct
// just to capture the raw [[policy.overrides]] entries as a slice of maps.
// Doing it as a separate pass means the regular decode-into-Config
// pipeline doesn't have to know anything about deferred rule resolution,
// and PolicyOptions can stay free of load-time scratch state.
func extractRawOverrides(path string) ([]map[string]any, error) {
	var helper struct { //exhaustruct:enforce
		Policy struct { //exhaustruct:enforce
			Overrides []map[string]any `toml:"overrides"`
		} `toml:"policy"`
	}
	if _, err := toml.DecodeFile(path, &helper); err != nil {
		return nil, err
	}
	return helper.Policy.Overrides, nil
}

// resolveRules expands raw [[policy.overrides]] tables into a slice of
// fully-populated Rules. Each rule's Runtime is pre-filled from the
// finalized base RuntimeConfig and then overlaid with whatever the
// rule's own TOML supplies. Because each section's UnmarshalTOML
// preserves existing values for absent keys, sparse rule overrides
// inherit unset fields from base — that's the point of doing this
// after Finalize rather than at decode time.
func resolveRules(
	raw []map[string]any,
	base RuntimeConfig,
	warnMsgs *[]string,
) ([]Rule, error) {
	rules := make([]Rule, 0, len(raw))
	for i, item := range raw {
		r := Rule{ //exhaustruct:enforce
			Name:     "",
			Priority: 0,
			Block:    false,
			Match:    nil,
			Runtime:  base,
		}

		if v, ok := item["name"].(string); ok {
			r.Name = v
		}
		if v, ok := item["priority"]; ok {
			pv, perr := parseIntFn[uint16](checkUint16)(v)
			if perr != nil {
				return nil, fmt.Errorf("rule %d: priority: %w", i, perr)
			}
			r.Priority = pv
		}
		if v, ok := item["block"].(bool); ok {
			r.Block = v
		}
		if v, ok := item["match"]; ok {
			r.Match = &MatchAttrs{} //exhaustruct:enforce
			if err := r.Match.UnmarshalTOML(v); err != nil {
				return nil, fmt.Errorf("rule %d: match: %w", i, err)
			}
		}
		if v, ok := item["dns"]; ok {
			if err := r.Runtime.DNS.UnmarshalTOML(v); err != nil {
				return nil, fmt.Errorf("rule %d: dns: %w", i, err)
			}
		}
		if v, ok := item["https"]; ok {
			if err := r.Runtime.HTTPS.UnmarshalTOML(v); err != nil {
				return nil, fmt.Errorf("rule %d: https: %w", i, err)
			}
		}
		if v, ok := item["udp"]; ok {
			if err := r.Runtime.UDP.UnmarshalTOML(v); err != nil {
				return nil, fmt.Errorf("rule %d: udp: %w", i, err)
			}
		}
		if v, ok := item["connection"]; ok {
			if err := r.Runtime.Conn.UnmarshalTOML(v); err != nil {
				return nil, fmt.Errorf("rule %d: connection: %w", i, err)
			}
		}

		// Transitional: rules currently inherit base's https.skip when they
		// don't set it explicitly, which makes a global skip=true silently
		// disable desync inside otherwise-tuned rules. Force-reset to false
		// when no explicit skip is present and warn the user — eventually
		// resolveRules will require https.skip to be set explicitly.
		if !hasExplicitKey(item, "https", "skip") && base.HTTPS.Skip {
			label := r.Name
			if label == "" {
				label = fmt.Sprintf("#%d", i)
			}
			*warnMsgs = append(*warnMsgs, fmt.Sprintf(
				"policy override %q inherits https.skip=true from base; auto-resetting to false. Set [policy.overrides.https].skip explicitly — this auto-reset will be removed in a future version.",
				label,
			))
			r.Runtime.HTTPS.Skip = false
		}

		rules = append(rules, r)
	}
	return rules, nil
}

// hasExplicitKey reports whether item[section] is a table that contains key.
func hasExplicitKey(item map[string]any, section, key string) bool {
	sub, ok := item[section].(map[string]any)
	if !ok {
		return false
	}
	_, present := sub[key]
	return present
}
