package config

import (
	"context"
	"fmt"
	"net"
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
				Value:     32,
				OnlyOnce:  true,
				Validator: validateUint8,
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
				OnlyOnce: true,
				Sources:  cli.EnvVars("SPOOFDPI_CONFIG"),
			},

			&cli.IntFlag{
				Name: "default-ttl",
				Usage: `
				Default TTL value for manipulated packets.`,
				Value:     64,
				OnlyOnce:  true,
				Validator: validateUint8,
			},

			&cli.BoolFlag{
				Name: "disorder",
				Usage: `
				If set, the fragmented Client Hello packets will be sent out-of-order.`,
				OnlyOnce: true,
			},

			&cli.StringFlag{
				Name: "dns-addr",
				Usage: `
				DNS address (default: 8.8.8.8)`,
				Value:     "8.8.8.8",
				OnlyOnce:  true,
				Validator: validateIPAddr,
			},

			&cli.BoolFlag{
				Name: "dns-ipv4-only",
				Usage: `
				Resolve only IPv4 addresses`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "dns-port",
				Usage: `
				Port number for dns (default: 53)`,
				Value:     53,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.StringFlag{
				Name: "doh-endpoint",
				Usage: `
				Endpoint for 'dns over https' (default: "https://${DNS_ADDR}/dns-query")`,
				Value:     "",
				OnlyOnce:  true,
				Validator: validateHTTPSEndpoint,
			},

			&cli.BoolFlag{
				Name: "enable-doh",
				Usage: `
				Enable 'dns-over-https'`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "fake-count",
				Usage: `
				Number of fake packets to be sent before Client Hello. 
				If 'window-size' is greater than 0, each fake packet will be 
				fragmented into segments of the specified window size. (default: 0)`,
				OnlyOnce:  true,
				Validator: validateUint8,
			},

			&cli.StringFlag{
				Name: "listen-addr",
				Usage: `
				IP address to listen on (default: 127.0.0.1)`,
				Value:    "127.0.0.1",
				OnlyOnce: true,
				Validator: func(v string) error {
					err := validateIPAddr(v)
					if err != nil {
						return err
					}

					return nil
				},
			},

			&cli.IntFlag{
				Name: "listen-port",
				Usage: `
				Port number to listen on (default: 8080)`,
				Value:     8080,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.StringFlag{
				Name: "log-level",
				Usage: `
				Set log level (default: 'info')`,
				Value:     "info",
				OnlyOnce:  true,
				Validator: validateLogLevel,
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

			&cli.IntFlag{
				Name: "window-size",
				Usage: `
				Specifies the chunk size in bytes for the Client Hello packet.
				Try lower values if the default fails to bypass the DPI.
				Setting this to 0 disables fragmentation. (default: 35, max: 255)`,
				Value:     35,
				OnlyOnce:  true,
				Validator: validateUint8,
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
		"{{.Name}}",
		"[global options]",
		"--{{.Name}}",
		", -{{.}}",
		"{{.TypeName}}",
		"{{.Usage}}",
		"{{.DefaultText}}",
	)
}

func parseConfigFromArgs(cmd *cli.Command) (*Config, error) {
	cfg := &Config{
		AutoPolicy:        cmd.Bool("auto-policy"),
		CacheShards:       Uint8Number{uint8(cmd.Int("cache-shards"))},
		DefaultTTL:        Uint8Number{uint8(cmd.Int("default-ttl"))},
		Disorder:          cmd.Bool("disorder"),
		DnsAddr:           IPAddress{net.ParseIP(cmd.String("dns-addr"))},
		DnsPort:           Uint16Number{uint16(cmd.Int("dns-port"))},
		DnsIPv4Only:       cmd.Bool("dns-ipv4-only"),
		DOHEndpoint:       HTTPSEndpoint{cmd.String("doh-endpoint")},
		DomainPolicySlice: parseDomainPolicySlice(cmd.StringSlice("policy")),
		EnableDOH:         cmd.Bool("enable-doh"),
		ListenAddr:        IPAddress{net.ParseIP(cmd.String("listen-addr"))},
		ListenPort:        Uint16Number{uint16(cmd.Int("listen-port"))},
		LogLevel:          LogLevel{cmd.String("log-level")},
		SetSystemProxy:    cmd.Bool("system-proxy"),
		Silent:            cmd.Bool("silent"),
		Timeout:           Uint16Number{uint16(cmd.Int("timeout"))},
		WindowSize:        Uint8Number{uint8(cmd.Int("window-size"))},
		FakeCount:         Uint8Number{uint8(cmd.Int("fake-count"))},
	}

	return cfg, nil
}
