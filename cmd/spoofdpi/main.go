package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xvzc/SpoofDPI/internal/applog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/proxy"
	"github.com/xvzc/SpoofDPI/internal/system"
	"github.com/xvzc/SpoofDPI/version"
)

func main() {
	args := config.ParseArgs()
	if args.Version {
		version.PrintVersion()
		os.Exit(0)
	}

	cfg := config.LoadConfigurationFromArgs(
		args,
		applog.WithScope(applog.NewLogger(args.Debug), "CONFIG"),
	)

	if !cfg.Silent() {
		printBanner(cfg)
	}

	baseLogger := applog.NewLogger(cfg.Debug())
	logger := applog.WithScope(baseLogger, "MAIN")

	if cfg.SetSystemProxy() {
		if err := system.SetProxy(cfg.ListenPort()); err != nil {
			logger.Fatal().Msgf("error while changing proxy settings: %s", err)
		}
		defer func() {
			if err := system.UnsetProxy(); err != nil {
				logger.Fatal().Msgf("error while disabling proxy: %s", err)
			}
		}()
	}

	// Create DNS resolvers with scoped loggers
	httpsResolver := dns.NewHTTPSResolver(
		cfg.DnsAddr(),
		cfg.DnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(HTTPS)"),
	)
	localResolver := dns.NewLocalResolver(
		cfg.DnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(LOCAL)"),
	)
	plainResolver := dns.NewPlainResolver(
		cfg.DnsAddr(),
		cfg.DnsPort(),
		cfg.DnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(PLAIN)"),
	)

	cache := datastruct.NewTTLCache[dns.RecordSet](
		cfg.CacheShards(),
		time.Duration(1*time.Minute),
	)

	routeResolver := dns.NewRouteResolver(
		cfg.EnableDOH(),
		localResolver,
		dns.NewCacheResolver(
			cache,
			plainResolver,
			applog.WithScope(baseLogger, "DNS(CACHE)"),
		),
		dns.NewCacheResolver(
			cache,
			httpsResolver,
			applog.WithScope(baseLogger, "DNS(CACHE)"),
		),
		applog.WithScope(baseLogger, "DNS(ROUTE)"),
	)

	// Create Proxy handlers with scoped loggers
	httpHandler := proxy.NewHttpHandler(
		applog.WithScope(baseLogger, "HTTP"),
	)
	httpsHandler := proxy.NewHttpsHandler(
		cfg.WindowSize(),
		applog.WithScope(baseLogger, "HTTPS"),
	)

	// Create Proxy with a scoped logger
	p := proxy.NewProxy(
		cfg.ListenAddr(),
		cfg.ListenPort(),
		cfg.Timeout(),
		cfg.AllowedPatterns(),
		routeResolver,
		httpHandler,
		httpsHandler,
		applog.WithScope(baseLogger, "PROXY"),
	)

	go p.Start()

	// Handle signals
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(
		sigs,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP)

	go func() {
		<-sigs
		done <- true
	}()

	<-done
}

func printBanner(cfg *config.Config) {
	const banner = `
 .d8888b.                              .d888 8888888b.  8888888b. 8888888
d88P  Y88b                            d88P"  888  "Y88b 888   Y88b  888
Y88b.                                 888    888    888 888    888  888
 "Y888b.   88888b.   .d88b.   .d88b.  888888 888    888 888   d88P  888
    "Y88b. 888 "88b d88""88b d88""88b 888    888    888 8888888P"   888
      "888 888  888 888  888 888  888 888    888    888 888         888
Y88b  d88P 888 d88P Y88..88P Y88..88P 888    888  .d88P 888         888
 "Y8888P"  88888P"   "Y88P"   "Y88P"  888    8888888P"  888       8888888
           888
           888
           888
`

	fmt.Print(banner)
	fmt.Printf("\n")
	fmt.Printf(" • LISTEN_ADDR : %s\n", fmt.Sprint(cfg.ListenAddr()))
	fmt.Printf(" • LISTEN_PORT : %s\n", fmt.Sprint(cfg.ListenPort()))
	fmt.Printf(" • DNS_ADDR    : %s\n", fmt.Sprint(cfg.DnsAddr()))
	fmt.Printf(" • DNS_PORT    : %s\n", fmt.Sprint(cfg.DnsPort()))
	fmt.Printf(" • DEBUG       : %s\n", fmt.Sprint(cfg.Debug()))
	fmt.Printf("\n")
	fmt.Printf("Press 'CTRL + c' to quit\n")
	fmt.Printf("\n")
}
