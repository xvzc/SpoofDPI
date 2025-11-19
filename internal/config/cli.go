package config

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/xvzc/SpoofDPI/version"
)

func CreateCommand(
	runFunc func(ctx context.Context, configDir string, cfg *Config),
) *cli.Command {
	cli.RootCommandHelpTemplate = createHelpTemplate()

	cmd := &cli.Command{
		Name:        "spoofdpi",
		Description: "Simple and fast anti-censorship tool to bypass DPI",
		Copyright:   "Apache License, Version 2.0, January 2004",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name: "cache-shards",
				Usage: `
				number of shards to use for ttlcache. it is recommended to set this to be 
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
				custom location of the config file to load. options given through the command 
				line flags will override the options set in this file. when not given, it will 
				search sequentially in the following locations: 
				$SPOOFDPI_CONFIG, /etc/spoofdpi.toml, $XDG_CONFIG_HOME/spoofdpi/spoofdpi.toml 
				and $HOME/.config/spoofdpi/spoofdpi.toml`,
				OnlyOnce: true,
				Sources:  cli.EnvVars("SPOOFDPI_CONFIG"),
			},

			&cli.StringFlag{
				Name: "dns-addr",
				Usage: `
				dns address (default: 8.8.8.8)`,
				Value:     "8.8.8.8",
				OnlyOnce:  true,
				Validator: validateIPAddr,
			},

			&cli.BoolFlag{
				Name: "dns-ipv4-only",
				Usage: `
				resolve only IPv4 addresses`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "dns-port",
				Usage: `
				port number for dns (default: 53)`,
				Value:     53,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.StringFlag{
				Name: "doh-endpoint",
				Usage: `
				endpoint for 'dns over https' (default: "https://${DNS_ADDR}/dns-query")`,
				Value:     "",
				OnlyOnce:  true,
				Validator: validateHTTPSEndpoint,
			},

			&cli.BoolFlag{
				Name: "enable-doh",
				Usage: `
				enable 'dns-over-https'`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "fake-https-packets",
				Usage: `
				number of fake packets to send before the client hello, higher values
        may increase success, but the lowest possible value is recommended.
        try this if tcp-level fragmentation (via --window-size) does not
        work. this feature requires root privilege and the 'libpcap'
				dependency (default: 0, max: 255)`,
				Value:     0,
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
				port number to listen on (default: 8080)`,
				Value:     8080,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.StringFlag{
				Name: "log-level",
				Usage: `
				set log level (default: 'info')`,
				Value:     "info",
				OnlyOnce:  true,
				Validator: validateLogLevel,
			},

			&cli.StringSliceFlag{
				Name: "policy",
				Usage: `
				domain rules that determine whether to perform DPI circumvention on match.
        supports wildcards, but the main domain name must not contain a wildcard.
        this flag can be given multiple times. policies start with 'i:' to include 
				or 'x:' to exclude the matching domain. when rules overlap, a more specific 
				rule (e.g., static) overrides a less specific one (e.g., wildcard).
        e.g. Given 'i:*.discordapp.com' and 'x:cdn.discordapp.com', traffic for
        'api.discordapp.com' and 'www.discordapp.com' will be circumvented, 
				but 'cdn.discordapp.com' will be passed through.`,
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
				do not show the banner and server information at start up`,
				OnlyOnce: true,
			},

			&cli.BoolFlag{
				Name: "system-proxy",
				Usage: `
				automatically set system-wide proxy configuration`,
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "timeout",
				Usage: `
				timeout for tcp connection in milliseconds. 
				no effect when the value is 0 (default: 0, max: 66535)`,
				Value:     0,
				OnlyOnce:  true,
				Validator: validateUint16,
			},

			&cli.BoolFlag{
				Name: "version",
				Usage: `
				print version; this may contain some other relevant information`,
				Aliases:  []string{"v"},
				OnlyOnce: true,
			},

			&cli.IntFlag{
				Name: "window-size",
				Usage: `
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
				fragmentation for the first data packet and the rest (default: 0, max: 255)`,
				Value:     0,
				OnlyOnce:  true,
				Validator: validateUint8,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("version") {
				version.PrintVersion()
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
						return fmt.Errorf("error parsing toml config: %s", err)
					}
				}
			}

			argsCfg, err := parseConfigFromArgs(cmd)
			if err != nil {
				return fmt.Errorf("error parsing config from args: %s", err)
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
		CacheShards:       Uint8Number{uint8(cmd.Int("cache-shards"))},
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
		FakeHTTPSPackets:  Uint8Number{uint8(cmd.Int("fake-https-packets"))},
	}

	return cfg, nil
}
