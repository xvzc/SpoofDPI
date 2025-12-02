package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v3"
)

func CreateCommand(
	runFunc func(ctx context.Context, configDir string, cfg *Config),
	version string,
	commit string,
	build string,
) *cli.Command {
	cli.RootCommandHelpTemplate = createHelpTemplate()

	cmd := &cli.Command{
		Name:        "spoofdpi",
		Description: "Simple and fast anti-censorship tool to bypass DPI",
		Copyright:   "Apache License, Version 2.0, January 2004",
		ErrWriter:   io.Discard,
		OnUsageError: func(ctx context.Context, cmd *cli.Command, err error, sub bool) error {
			return err
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "auto-policy",
				Usage: `
				Automatically detect the blocked sites and add policies (default: false)`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "cache-shards",
				Usage: `
				Number of shards to use for ttlcache. It is recommended to set this to be 
				at least the number of CPU cores for optimal performance (default: 32, max: 255)`,
				Value:            32,
				OnlyOnce:         true,
				Validator:        validateUint8,
				ValidateDefaults: true,
			},

			&cli.BoolFlag{
				Name: "clean",
				Usage: `
				if set, all configuration files will be ignored`,
				OnlyOnce: true,
			},

			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage: `
				Custom location of the config file to load. Options given through the command 
				line flags will override the options set in this file.`,
				OnlyOnce:         true,
				Sources:          cli.EnvVars("SPOOFDPI_CONFIG"),
				ValidateDefaults: true,
			},

			&cli.IntFlag{
				Name: "default-ttl",
				Usage: `
				Default TTL value for manipulated packets.`,
				Value:            64,
				OnlyOnce:         true,
				Validator:        validateUint8,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "dns-addr",
				Usage: `<ip:port>
				Upstream DNS server address for standard UDP queries.
				(default: 8.8.8.8:53)`,
				Value:            "8.8.8.8:53",
				OnlyOnce:         true,
				Validator:        validateHostPort,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "dns-default",
				Usage: `<'udp'|'doh'|'sys'>
				Default resolution mode for domains that do not match any specific rule.
				(default: "udp")`,
				Value:            "udp",
				OnlyOnce:         true,
				Validator:        validateDNSMode,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "dns-qtype",
				Usage: `<'ipv4'|'ipv6'|'all'>
				Filters DNS queries by record type (A for IPv4, AAAA for IPv6).
				(default: "ipv4")`,
				Value:            "ipv4",
				OnlyOnce:         true,
				Validator:        validateDNSQueryType,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "doh-url",
				Usage: `<https_url>
				Endpoint URL for DNS over HTTPS (DoH) queries.
				(default: "https://dns.google/dns-query")`,
				Value:            "https://dns.google/dns-query",
				OnlyOnce:         true,
				Validator:        validateHTTPSEndpoint,
				ValidateDefaults: true,
			},

			&cli.IntFlag{
				Name: "https-fake-count",
				Usage: `
				Number of fake packets to be sent before the Client Hello.
				Requires 'https-chunk-size' > 0 for fragmentation. (default: 0)`,
				Value:            0,
				OnlyOnce:         true,
				Validator:        validateUint8,
				ValidateDefaults: true,
			},

			&cli.BoolFlag{
				Name: "https-disorder",
				Usage: `
				If set, sends fragmented Client Hello packets out-of-order. (default: false)`,
				OnlyOnce: true,
			},

			&cli.StringFlag{
				Name: "https-split-default",
				Usage: `<'chunk'|'1byte'|'sni'|'none'>
				Specifies the default packet fragmentation strategy to use. (default: 'chunk')`,
				Value:            "chunk",
				OnlyOnce:         true,
				Validator:        validateHTTPSSplitMode,
				ValidateDefaults: true,
			},

			&cli.IntFlag{
				Name: "https-chunk-size",
				Usage: `
				The chunk size (in bytes) for packet fragmentation. Only used when 
				'https-split-default' is 'chunk'. Setting to '0' disables fragmentation 
				regardless of the split mode. (default: 35, max: 255)`,
				Value:            35,
				OnlyOnce:         true,
				Validator:        validateUint8,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "listen-addr",
				Usage: `
				IP address to listen on (default: 127.0.0.1:8080)`,
				Value:            "127.0.0.1:8080",
				OnlyOnce:         true,
				Validator:        validateHostPort,
				ValidateDefaults: true,
			},

			&cli.StringFlag{
				Name: "log-level",
				Usage: `
				Set log level (default: 'info')`,
				Value:            "info",
				OnlyOnce:         true,
				Validator:        validateLogLevel,
				ValidateDefaults: true,
			},

			&cli.StringSliceFlag{
				Name: "policy",
				Usage: `
				Domain rules that determine whether to perform DPI circumvention on match.
        This flag can be given multiple times.`,
				Validator: func(ss []string) error {
					for _, s := range ss {
						if err := validatePolicy(s); err != nil {
							return err
						}
					}

					return nil
				},
			},

			&cli.BoolFlag{
				Name: "silent",
				Usage: `
				Do not show the banner and server information at start up`,
				OnlyOnce: true,
			},

			&cli.BoolFlag{
				Name: "system-proxy",
				Usage: `
				Automatically set system-wide proxy configuration`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "timeout",
				Usage: `
				Timeout for tcp connection in milliseconds. 
				No effect when the value is 0 (default: 0, max: 66535)`,
				Value:     0,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.BoolFlag{
				Name: "version",
				Usage: `
				Print version; this may contain some other relevant information`,
				Aliases:  []string{"v"},
				OnlyOnce: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("version") {
				fmt.Printf("spoofdpi %s %s (%s)\n", version, commit, build)
				fmt.Println("Official docs at https://spoofdpi.xvzc.dev")
				os.Exit(0)
			}

			var tomlCfg *Config
			var configDir string
			if !cmd.Bool("clean") {
				configFilename := "spoofdpi.toml"

				configDirs := []string{
					path.Join(string(os.PathSeparator), "etc", configFilename),
					path.Join(os.Getenv("XDG_CONFIG_HOME"), "spoofdpi", configFilename),
					path.Join(os.Getenv("HOME"), ".config", "spoofdpi", configFilename),
				}

				c, err := findConfigFileToLoad(cmd.String("config"), configDirs)
				if err != nil {
					return err
				}

				if c != "" {
					configDir = c
					tomlCfg, err = parseTomlConfig(c)
					if err != nil {
						return fmt.Errorf("error parsing toml config: %w", err)
					}
				}
			}

			argsCfg, err := parseConfigFromArgs(cmd)
			if err != nil {
				return fmt.Errorf("error parsing config from args: %w", err)
			}

			var finalCfg *Config
			if tomlCfg == nil {
				finalCfg = argsCfg
			} else {
				finalCfg = mergeConfig(argsCfg, tomlCfg, os.Args[1:])
			}

			runFunc(ctx, strings.Replace(configDir, os.Getenv("HOME"), "~", 1), finalCfg)
			return nil
		},
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage: `
        show help`,
	}

	return cmd
}

func createHelpTemplate() string {
	return fmt.Sprintf(`DESCRIPTION:
  %s{{if .Copyright }}
COPYRIGHT:
  {{.Copyright}}{{end}}
USAGE:
  %s {{if .Flags}}%s{{end}}{{if .Commands}}
GLOBAL OPTIONS:
  {{range .VisibleFlags}}%s{{if .Aliases}}{{range .Aliases}}%s{{end}}{{end}} %s %s %s
	{{end}}{{end}}
	`,
		"{{.Name}} - {{.Description}}",
		"{{.Name}}", // spoofdpi
		"[global options]",
		"--{{.Name}}", // --option
		", -{{.}}",    // -o
		"{{.TypeName}}",
		"{{.Usage}}",
		"{{.DefaultText}}",
	)
}

func parseConfigFromArgs(cmd *cli.Command) (*Config, error) {
	listenAddr := HostPort{}
	_ = listenAddr.UnmarshalText([]byte(cmd.String("listen-addr")))

	dnsAddr := HostPort{}
	_ = dnsAddr.UnmarshalText([]byte(cmd.String("dns-addr")))

	cfg := &Config{
		AutoPolicy:        cmd.Bool("auto-policy"),
		CacheShards:       Uint8Number{uint8(cmd.Int("cache-shards"))},
		DefaultTTL:        Uint8Number{uint8(cmd.Int("default-ttl"))},
		DNSAddr:           dnsAddr,
		DNSDefault:        DNSMode{cmd.String("dns-default")},
		DNSQueryType:      DNSQueryType{cmd.String("dns-qtype")},
		DOHURL:            HTTPSEndpoint{cmd.String("doh-url")},
		DomainPolicySlice: parseDomainPolicySlice(cmd.StringSlice("policy")),
		HTTPSChunkSize:    Uint8Number{uint8(cmd.Int("https-chunk-size"))},
		HTTPSDisorder:     cmd.Bool("https-disorder"),
		HTTPSFakeCount:    Uint8Number{uint8(cmd.Int("https-fake-count"))},
		HTTPSSplitDefault: HTTPSSplitMode{cmd.String("https-split-default")},
		ListenAddr:        listenAddr,
		LogLevel:          LogLevel{cmd.String("log-level")},
		SetSystemProxy:    cmd.Bool("system-proxy"),
		Silent:            cmd.Bool("silent"),
		Timeout:           Uint16Number{uint16(cmd.Int("timeout"))},
	}

	return cfg, nil
}
