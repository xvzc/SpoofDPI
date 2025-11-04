package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
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
		applog.WithScope(applog.NewLogger(false), "CONFIG"),
	)

	if !cfg.Silent() {
		printColoredBanner(cfg)
	}

	logger := applog.WithScope(applog.NewLogger(cfg.Debug()), "MAIN")

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
		applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(HTTPS)"),
	)
	localResolver := dns.NewLocalResolver(
		cfg.DnsQueryTypes(),
		applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(LOCAL)"),
	)
	plainResolver := dns.NewPlainResolver(
		cfg.DnsAddr(),
		cfg.DnsPort(),
		cfg.DnsQueryTypes(),
		applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(PLAIN)"),
	)

	cache := datastruct.NewTTLCache[dns.RecordSet](32, time.Duration(1*time.Minute))

	routeResolver := dns.NewRouteResolver(
		cfg.EnableDOH(),
		localResolver,
		dns.NewCacheResolver(
			cache,
			plainResolver,
			applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(CACHE)"),
		),
		dns.NewCacheResolver(
			cache,
			httpsResolver,
			applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(CACHE)"),
		),
		applog.WithScope(applog.NewLogger(cfg.Debug()), "DNS(ROUTE)"),
	)

	// Create Proxy handlers with scoped loggers
	httpHandler := proxy.NewHttpHandler(
		applog.WithScope(applog.NewLogger(cfg.Debug()), "HTTP"),
	)
	httpsHandler := proxy.NewHttpsHandler(
		cfg.WindowSize(),
		applog.WithScope(applog.NewLogger(cfg.Debug()), "HTTPS"),
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
		applog.WithScope(applog.NewLogger(cfg.Debug()), "PROXY"),
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

func printColoredBanner(cfg *config.Config) {
	cyan := putils.LettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := putils.LettersFromStringWithStyle(
		"DPI",
		pterm.NewStyle(pterm.FgLightMagenta),
	)
	_ = pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	_ = pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR    : " + fmt.Sprint(cfg.ListenAddr())},
		{Level: 0, Text: "PORT    : " + fmt.Sprint(cfg.ListenPort())},
		{Level: 0, Text: "DNS     : " + fmt.Sprint(cfg.DnsAddr())},
		{Level: 0, Text: "DEBUG   : " + fmt.Sprint(cfg.Debug())},
	}).Render()

	pterm.DefaultBasicText.Println("Press 'CTRL + c' to quit")
}
