package config

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/xvzc/SpoofDPI/internal/proto"
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
			&cli.StringFlag{
				Name: "app-mode",
				Usage: fmt.Sprintf(`<"http"|"socks5"|"tun">
				Specifies the proxy mode. (default: %q)`,
					defaultCfg.App.Mode.String(),
				),
				OnlyOnce:  true,
				Validator: checkAppMode,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.App.Mode = lo.ToPtr(MustParseServerModeType(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "clean",
				Usage: `
				if set, all configuration files will be ignored (default: false)`,
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
				Name: "default-fake-ttl",
				Usage: fmt.Sprintf(`
				Default TTL value for fake packets. (default: %v)`,
					*defaultCfg.Conn.DefaultFakeTTL,
				),
				OnlyOnce:  true,
				Validator: checkUint8NonZero,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Conn.DefaultFakeTTL = lo.ToPtr(uint8(v))
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
					argsCfg.DNS.Addr = lo.ToPtr(MustParseTCPAddr(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "dns-cache",
				Usage: fmt.Sprintf(`
				If set, DNS records will be cached. (default: %v)`,
					*defaultCfg.DNS.Cache,
				),
				Value:    false,
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.DNS.Cache = lo.ToPtr(v)
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
					argsCfg.DNS.Mode = lo.ToPtr(MustParseDNSModeType(v))
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
					argsCfg.DNS.HTTPSURL = lo.ToPtr(v)
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
					argsCfg.DNS.QType = lo.ToPtr(MustParseDNSQueryType(v))
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "dns-timeout",
				Usage: fmt.Sprintf(`
				Timeout for dns connection in milliseconds. 
				No effect when the value is 0 (default: %v, max: %v)`,
					defaultCfg.Conn.DNSTimeout.Milliseconds(),
					math.MaxUint16,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint16,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Conn.DNSTimeout = lo.ToPtr(
						time.Duration(v * int64(time.Millisecond)),
					)
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "https-fake-count",
				Usage: fmt.Sprintf(`
				Number of fake packets to be sent before the Client Hello.
				Requires 'https-chunk-size' > 0 for fragmentation. (default: %v)`,
					*defaultCfg.HTTPS.FakeCount,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint8,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.HTTPS.FakeCount = lo.ToPtr(uint8(v))
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
					argsCfg.HTTPS.FakePacket = proto.NewFakeTLSMessage(MustParseBytes(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "https-disorder",
				Usage: fmt.Sprintf(`
				If set, sends fragmented Client Hello packets out-of-order. (default: %v)`,
					*defaultCfg.HTTPS.Disorder,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.HTTPS.Disorder = lo.ToPtr(v)
					return nil
				},
			},

			&cli.StringFlag{
				Name: "https-split-mode",
				Usage: fmt.Sprintf(`<"sni"|"random"|"chunk"|"sni"|"custom"|"none">
				Specifies the default packet fragmentation strategy to use. (default: %q)`,
					defaultCfg.HTTPS.SplitMode.String(),
				),
				Value:     "chunk",
				OnlyOnce:  true,
				Validator: checkHTTPSSplitMode,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.HTTPS.SplitMode = lo.ToPtr(mustParseHTTPSSplitModeType(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "https-skip",
				Usage: fmt.Sprintf(`
				If set, HTTPS traffic will be processed without any DPI bypass techniques. 
				(default: %v)`,
					*defaultCfg.HTTPS.Skip,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.HTTPS.Skip = lo.ToPtr(v)
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
					*defaultCfg.HTTPS.ChunkSize,
					math.MaxUint8,
				),
				OnlyOnce:  true,
				Validator: checkUint8NonZero,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.HTTPS.ChunkSize = lo.ToPtr(uint8(v))
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "udp-fake-count",
				Usage: fmt.Sprintf(`
				Number of fake packets to be sent. (default: %v)`,
					*defaultCfg.UDP.FakeCount,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: int64Range(0, math.MaxInt),
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.UDP.FakeCount = lo.ToPtr(int(v))
					return nil
				},
			},

			&cli.StringFlag{
				Name: "udp-fake-packet",
				Usage: `<byte_array>
				Comma-separated hexadecimal byte array used for fake packet. 
				(default: built-in fake packet)`,
				Value:     MustParseHexCSV(defaultCfg.UDP.FakePacket),
				OnlyOnce:  true,
				Validator: checkHexBytesStr,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					argsCfg.UDP.FakePacket = MustParseBytes(v)
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "udp-idle-timeout",
				Usage: fmt.Sprintf(`
				Idle timeout for udp connection in milliseconds. 
				No effect when the value is 0 (default: %v, max: %v)`,
					defaultCfg.Conn.UDPIdleTimeout.Milliseconds(),
					math.MaxUint16,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint16,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Conn.UDPIdleTimeout = lo.ToPtr(
						time.Duration(v * int64(time.Millisecond)),
					)
					return nil
				},
			},

			&cli.StringFlag{
				Name: "listen-addr",
				Usage: `
				IP address to listen on (default: 127.0.0.1:8080 for http, or 127.0.0.1:1080 for socks5)`,
				OnlyOnce:  true,
				Validator: checkHostPort,
				Action: func(ctx context.Context, cmd *cli.Command, v string) error {
					if v == "" {
						return nil
					}
					argsCfg.App.ListenAddr = lo.ToPtr(MustParseTCPAddr(v))
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
					argsCfg.App.LogLevel = lo.ToPtr(MustParseLogLevel(v))
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "silent",
				Usage: fmt.Sprintf(`
				Do not show the banner at start up (default: %v)`,
					*defaultCfg.App.Silent,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.App.Silent = lo.ToPtr(v)
					return nil
				},
			},

			&cli.BoolFlag{
				Name: "auto-configure-network",
				Usage: fmt.Sprintf(`
				Automatically set system-wide proxy configuration (default: %v)`,
					*defaultCfg.App.AutoConfigureNetwork,
				),
				OnlyOnce: true,
				Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
					argsCfg.App.AutoConfigureNetwork = lo.ToPtr(v)
					return nil
				},
			},

			&cli.Int64Flag{
				Name: "tcp-timeout",
				Usage: fmt.Sprintf(`
				Timeout for tcp connection in milliseconds. 
				No effect when the value is 0 (default: %v, max: %v)`,
					defaultCfg.Conn.TCPTimeout.Milliseconds(),
					math.MaxUint16,
				),
				Value:     0,
				OnlyOnce:  true,
				Validator: checkUint16,
				Action: func(ctx context.Context, cmd *cli.Command, v int64) error {
					argsCfg.Conn.TCPTimeout = lo.ToPtr(
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

			finalCfg := defaultCfg.Merge(tomlCfg.Merge(argsCfg))

			if finalCfg.App.ListenAddr == nil {
				port := 8080
				if *finalCfg.App.Mode == AppModeSOCKS5 {
					port = 1080
				}
				finalCfg.App.ListenAddr = &net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: port,
				}
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
