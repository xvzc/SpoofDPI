package config

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func CreateCommand(
	runFunc func(ctx context.Context, configDir string, cfg *Config),
	version string,
	commit string,
	build string,
) *cli.Command {
	cli.RootCommandHelpTemplate = createHelpTemplate()

	argsCfg := NewConfig()
	defaultCfg := getDefault()

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
				Name: "clean",
				Usage: `
				if set, all configuration files will be ignored (default: %v)`,
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

			&cli.Int64Flag{
				Name: "default-ttl",
				Usage: fmt.Sprintf(`
				Default TTL value for manipulated packets. (default: %v)`,
					*defaultCfg.Server.DefaultTTL,
				),
				OnlyOnce:  true,
				Validator: checkUint8NonZero,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Server.DefaultTTL = ptr.FromValue(uint8(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "dns-addr",
				Usage: fmt.Sprintf(`<ip:port>
				Upstream DNS server address for standard UDP queries. (default: %v)`,
					defaultCfg.DNS.Addr.String(),
				),
				OnlyOnce:  true,
				Validator: checkHostPort,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.DNS.Addr = ptr.FromValue(MustParseTCPAddr(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "dns-cache",
				Usage: fmt.Sprintf(`
				If set, DNS records will be cached. (default: %v)`,
					defaultCfg.DNS.Cache,
				),
				Value:    false,
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.DNS.Cache = ptr.FromValue(v)
					return nil
				},
			},

			&cli.StringFlag{
				Name: "dns-mode",
				Usage: fmt.Sprintf(`<"udp"|"doh"|"sys">
				Default resolution mode for domains that do not match any specific rule.
				(default: %q)`,
					defaultCfg.DNS.Mode.String(),
				),
				Value:     "udp",
				OnlyOnce:  true,
				Validator: checkDNSMode,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.DNS.Mode = ptr.FromValue(MustParseDNSModeType(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "dns-https-url",
				Usage: fmt.Sprintf(`<https_url>
				Endpoint URL for DNS over HTTPS (DoH) queries. 
				(default: %q)`,
					*defaultCfg.DNS.HTTPSURL,
				),
				Value:     "https://dns.google/dns-query",
				OnlyOnce:  true,
				Validator: checkHTTPSEndpoint,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.DNS.HTTPSURL = ptr.FromValue(v)
					return nil
				},
			},

			&cli.StringFlag{
				Name: "dns-qtype",
				Usage: fmt.Sprintf(`<"ipv4"|"ipv6"|"all">
				Filters DNS queries by record type (A for IPv4, AAAA for IPv6).
				(default: %q)`,
					defaultCfg.DNS.QType.String(),
				),
				Value:     "ipv4",
				OnlyOnce:  true,
				Validator: checkDNSQueryType,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.DNS.QType = ptr.FromValue(MustParseDNSQueryType(v))
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "https-fake-count",
				Usage: fmt.Sprintf(`
				Number of fake packets to be sent before the Client Hello.
				Requires 'https-chunk-size' > 0 for fragmentation. (default: %v)`,
					defaultCfg.HTTPS.FakeCount,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint8,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.HTTPS.FakeCount = ptr.FromValue(uint8(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "https-fake-packet",
				Usage: `<byte_array>
				Comma-separated hexadecimal byte array used for fake Client Hello. 
				(default: built-in fake packet)`,
				Value:     MustParseHexCSV([]byte(FakeClientHello)),
				OnlyOnce:  true,
				Validator: checkHexBytesStr,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.HTTPS.FakePacket = MustParseBytes(v)
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "https-disorder",
				Usage: fmt.Sprintf(`
				If set, sends fragmented Client Hello packets out-of-order. (default: %v)`,
					defaultCfg.HTTPS.Disorder,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.HTTPS.Disorder = ptr.FromValue(v)
					return nil
				},
			},

			&cli.StringFlag{
				Name: "https-split-mode",
				Usage: fmt.Sprintf(`<"sni"|"random"|"chunk"|"sni"|"none">
				Specifies the default packet fragmentation strategy to use. (default: %q)`,
					defaultCfg.HTTPS.SplitMode.String(),
				),
				Value:     "chunk",
				OnlyOnce:  true,
				Validator: checkHTTPSSplitMode,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.HTTPS.SplitMode = ptr.FromValue(mustParseHTTPSSplitModeType(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "https-skip",
				Usage: fmt.Sprintf(`
				If set, HTTPS traffic will be processed without any DPI bypass techniques. 
				(default: %v)`,
					defaultCfg.HTTPS.Skip,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.HTTPS.Skip = ptr.FromValue(v)
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "https-chunk-size",
				Usage: fmt.Sprintf(`
				The chunk size (in bytes) for packet fragmentation. This value is only applied 
				when 'https-split-default' is 'chunk'. While setting the size to '0' internally 
				disables fragmentation (to avoid division-by-zero errors), you should set 
				'https-split-default' to 'none' to disable the feature cleanly.
				(default: %v, max: %v)`,
					defaultCfg.HTTPS.ChunkSize,
					math.MaxUint8,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint8NonZero,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.HTTPS.ChunkSize = ptr.FromValue(uint8(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "listen-addr",
				Usage: fmt.Sprintf(`
				IP address to listen on (default: %v)`,
					defaultCfg.Server.ListenAddr.String(),
				),
				Value:     "127.0.0.1:8080",
				OnlyOnce:  true,
				Validator: checkHostPort,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.Server.ListenAddr = ptr.FromValue(MustParseTCPAddr(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "log-level",
				Usage: `
				Set log level (default: "info")`,
				OnlyOnce:  true,
				Validator: checkLogLevel,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.General.LogLevel = ptr.FromValue(MustParseLogLevel(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "policy-auto",
				Usage: fmt.Sprintf(`
				Automatically detect the blocked sites and add policies (default: %v)`,
					*defaultCfg.Policy.Auto,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.Policy.Auto = ptr.FromValue(v)
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "silent",
				Usage: fmt.Sprintf(`
				Do not show the banner at start up (default: %v)`,
					defaultCfg.General.Silent,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.General.Silent = ptr.FromValue(v)
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "system-proxy",
				Usage: fmt.Sprintf(`
				Automatically set system-wide proxy configuration (default: %v)`,
					defaultCfg.General.SetSystemProxy,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.General.SetSystemProxy = ptr.FromValue(v)
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "timeout",
				Usage: fmt.Sprintf(`
				Timeout for tcp connection in milliseconds. 
				No effect when the value is 0 (default: %v, max: %v)`,
					defaultCfg.Server.Timeout,
					math.MaxUint16,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint16,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Server.Timeout = ptr.FromValue(
						time.Duration(v * int64(time.Millisecond)),
					)
					return nil
				},
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

			tomlCfg := NewConfig()
			var configDir string
			if !cmd.Bool("clean") {
				configFilename := "spoofdpi.toml"

				configDirs := []string{
					path.Join(string(os.PathSeparator), "etc", configFilename),
					path.Join(os.Getenv("XDG_CONFIG_HOME"), "spoofdpi", configFilename),
					path.Join(os.Getenv("HOME"), ".config", "spoofdpi", configFilename),
				}

				c, err := searchTomlFile(cmd.String("config"), configDirs)
				if err != nil {
					return err
				}

				if c != "" {
					configDir = c
					tomlCfg, err = fromTomlFile(c)
					if err != nil {
						return fmt.Errorf("error parsing '%s': %w", c, err)
					}
				}
			}

			// defaultCfg := getDefault()
			// // argsCfg := fromFlags(cmd)
			//
			finalCfg := getDefault().Merge(tomlCfg.Merge(argsCfg))
			// finalCfg = finalCfg.Merge(argsCfg)

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
