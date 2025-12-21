package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xvzc/SpoofDPI/internal/cache"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proxy"
	"github.com/xvzc/SpoofDPI/internal/proxy/http"
	"github.com/xvzc/SpoofDPI/internal/session"
	"github.com/xvzc/SpoofDPI/internal/system"
)

// Version and commit are set at build time.
var (
	version = "dev"
	commit  = "unknown"
	build   = "unknown"
)

func main() {
	cmd := config.CreateCommand(runApp, version, commit, build)
	ctx := session.WithNewTraceID(context.Background())
	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(ctx context.Context, configDir string, cfg *config.Config) {
	if !*cfg.General.Silent {
		printBanner()
	}

	logging.SetGlobalLogger(ctx, *cfg.General.LogLevel)

	logger := log.Logger.With().Ctx(ctx).Logger()
	logger.Info().Str("version", version).Msg("started spoofdpi")
	if configDir != "" {
		logger.Info().
			Str("dir", configDir).
			Msgf("config file loaded")
	}

	// set system-wide proxy configuration.
	if !*cfg.General.SetSystemProxy {
		logger.Info().Msg("use `--system-proxy` to automatically set system proxy")
	}

	resolver := createResolver(logger, cfg)
	p, err := createProxy(logger, cfg, resolver)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("failed to create proxy")
	}

	// start app
	wait := make(chan struct{}) // wait for setup logs to be printed
	go p.ListenAndServe(ctx, wait)

	// set system-wide proxy configuration.
	if *cfg.General.SetSystemProxy {
		port := cfg.Server.ListenAddr.Port
		if err := system.SetProxy(logger, uint16(port)); err != nil {
			logger.Fatal().Err(err).Msg("failed to enable system proxy")
		}
		defer func() {
			if err := system.UnsetProxy(logger); err != nil {
				logger.Fatal().Err(err).Msg("failed to disable system proxy")
			}
		}()
	}

	logger.Info().Msg("dns info")
	logger.Info().Msgf(" query type '%s'", cfg.DNS.QType.String())
	logger.Info().Msgf(" resolvers")
	dnsInfo := resolver.Info()
	for i := range dnsInfo {
		logger.Info().Str("dst", dnsInfo[i].Dst).Msgf("  %s", dnsInfo[i].Name)
	}

	logger.Info().Msg("https info")
	logger.Info().
		Str("split-mode", cfg.HTTPS.SplitMode.String()).
		Uint8("chunk-size", uint8(*cfg.HTTPS.ChunkSize)).
		Bool("disorder", *cfg.HTTPS.Disorder).
		Msg(" split")

	logger.Info().
		Uint8("count", uint8(*cfg.HTTPS.FakeCount)).
		Msg(" fake")

	logger.Info().
		Bool("auto", *cfg.Policy.Auto).
		Msgf("policy")

	if *cfg.Server.Timeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Server.Timeout)).
			Msgf("connection timeout")
	}

	wait <- struct{}{}

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

func createResolver(logger zerolog.Logger, cfg *config.Config) dns.Resolver {
	// create a TTL cache for storing DNS records.

	udpResolver := dns.NewUDPResolver(logging.WithScope(logger, "dns"), cfg.DNS.Clone())

	dohResolver := dns.NewHTTPSResolver(logging.WithScope(logger, "dns"), cfg.DNS.Clone())

	sysResolver := dns.NewSystemResolver(
		logging.WithScope(logger, "dns"),
		cfg.DNS.Clone(),
	)

	cacheResolver := dns.NewCacheResolver(
		logging.WithScope(logger, "dns"),
		cache.NewTTLCache(
			cache.TTLCacheAttrs{
				NumOfShards:     64,
				CleanupInterval: time.Duration(3 * time.Minute),
			},
		),
	)

	// create a resolver that routes DNS queries based on rules.
	return dns.NewRouteResolver(
		logging.WithScope(logger, "dns"),
		dohResolver,
		udpResolver,
		sysResolver,
		cacheResolver,
		cfg.DNS.Clone(),
	)
}

func createPacketObjects(
	logger zerolog.Logger,
	cfg *config.Config,
) (packet.Sniffer, packet.Writer, error) {
	// create a network detector for passive discovery
	networkDetector := packet.NewNetworkDetector(
		logging.WithScope(logger, "pkt"),
	)

	if err := networkDetector.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("error starting network detector: %w", err)
	}

	// Wait for gateway MAC with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gatewayMAC, err := networkDetector.WaitForGatewayMAC(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect gateway (timeout): %w", err)
	}

	iface := networkDetector.GetInterface()

	// create a pcap handle for packet capturing.
	handle, err := packet.NewHandle(iface)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"error opening pcap handle on interface %s: %w",
			iface.Name,
			err,
		)
	}

	logger.Info().Msg("network info")
	logger.Info().Str("name", iface.Name).
		Str("mac", iface.HardwareAddr.String()).
		Msg(" interface")
	logger.Info().
		Str("mac", gatewayMAC.String()).
		Msg(" gateway (passive detection)")

	hopCache := cache.NewLRUCache(4096)
	sniffer := packet.NewTCPSniffer(
		logging.WithScope(logger, "pkt"),
		hopCache,
		handle,
		uint8(*cfg.Server.DefaultTTL),
	)
	sniffer.StartCapturing()

	writer := packet.NewTCPWriter(
		logging.WithScope(logger, "pkt"),
		handle,
		iface,
		gatewayMAC,
	)

	return sniffer, writer, nil
}

func createProxy(
	logger zerolog.Logger,
	cfg *config.Config,
	resolver dns.Resolver,
) (proxy.ProxyServer, error) {
	ruleMatcher := matcher.NewRuleMatcher(
		matcher.NewAddrMatcher(),
		matcher.NewDomainMatcher(),
	)
	if cfg.Policy.Overrides != nil {
		for _, r := range cfg.Policy.Overrides {
			if err := ruleMatcher.Add(&r); err != nil {
				return nil, err
			}
		}
	}

	// create an HTTP handler.
	httpHandler := http.NewHTTPHandler(logging.WithScope(logger, "hnd"))

	var sniffer packet.Sniffer
	var writer packet.Writer

	if cfg.ShouldEnablePcap() {
		var err error
		sniffer, writer, err = createPacketObjects(logger, cfg)
		if err != nil {
			return nil, err
		}
	}

	httpsHandler := http.NewHTTPSHandler(
		logging.WithScope(logger, "hnd"),
		desync.NewTLSDesyncer(
			writer,
			sniffer,
			&desync.TLSDesyncerAttrs{DefaultTTL: *cfg.Server.DefaultTTL},
		),
		sniffer,
		cfg.HTTPS.Clone(),
	)

	// if cfg.Server.EnableSocks5 != nil && *cfg.Server.EnableSocks5 {
	// 	return socks5.NewSocks5Proxy(
	// 		logging.WithScope(logger, "pxy"),
	// 		resolver,
	// 		httpsHandler,
	// 		ruleMatcher,
	// 		cfg.Server.Clone(),
	// 		cfg.Policy.Clone(),
	// 	), nil
	// }

	return http.NewHTTPProxy(
		logging.WithScope(logger, "pxy"),
		resolver,
		httpHandler,
		httpsHandler,
		ruleMatcher,
		cfg.Server.Clone(),
		cfg.Policy.Clone(),
	), nil
}

func printBanner() {
	const banner = `
 .d8888b.                              .d888 8888888b.  8888888b. 8888888
d88P  Y88b                            d88P'  888  'Y88b 888   Y88b  888
Y88b.                                 888    888    888 888    888  888
 'Y888b.   88888b.   .d88b.   .d88b.  888888 888    888 888   d88P  888
    'Y88b. 888 '88b d88''88b d88''88b 888    888    888 8888888P'   888
      '888 888  888 888  888 888  888 888    888    888 888         888
Y88b  d88P 888 d88P Y88..88P Y88..88P 888    888  .d88P 888         888
 'Y8888P'  88888P'   'Y88P'   'Y88P'  888    8888888P'  888       8888888
           888
           888
           888

`

	fmt.Print(banner)
	fmt.Printf("Press 'CTRL + c' to quit\n")
	fmt.Printf("\n")
}
