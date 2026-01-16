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
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/server"
	"github.com/xvzc/SpoofDPI/internal/server/http" // Add http import
	"github.com/xvzc/SpoofDPI/internal/server/socks5"
	"github.com/xvzc/SpoofDPI/internal/server/tun"
	"github.com/xvzc/SpoofDPI/internal/session"
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
	if !*cfg.App.Silent {
		printBanner()
	}

	logging.SetGlobalLogger(ctx, *cfg.App.LogLevel)

	logger := log.Logger.With().Ctx(ctx).Logger()
	logger.Info().Str("version", version).Msg("started spoofdpi")
	if configDir != "" {
		logger.Info().
			Str("dir", configDir).
			Msgf("config file loaded")
	}

	resolver := createResolver(logger, cfg)

	srv, err := createServer(logger, cfg, resolver)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	// Start server
	ready := make(chan struct{})
	go func() {
		if err := srv.Start(ctx, ready); err != nil {
			logger.Fatal().Err(err).Msgf("failed to start server: %T", srv)
		}
	}()

	<-ready

	// System Proxy Config
	if *cfg.App.SetNetworkConfig {
		if err := srv.SetNetworkConfig(); err != nil {
			logger.Fatal().Err(err).Msg("failed to set system network config")
		}
		defer func() {
			if err := srv.UnsetNetworkConfig(); err != nil {
				logger.Error().Err(err).Msg("failed to unset system network config")
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

	if *cfg.Conn.DNSTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.DNSTimeout.Milliseconds())).
			Msgf("dns connection timeout")
	}
	if *cfg.Conn.TCPTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.TCPTimeout.Milliseconds())).
			Msgf("tcp connection timeout")
	}
	if *cfg.Conn.UDPTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.UDPTimeout.Milliseconds())).
			Msgf("udp connection timeout")
	}

	logger.Info().Msgf("app-mode; %s", cfg.App.Mode.String())

	logger.Info().Msgf("server started on %s", srv.Addr())

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

	// Graceful shutdown
	_ = srv.Stop()
}

func createResolver(logger zerolog.Logger, cfg *config.Config) dns.Resolver {
	// create a TTL cache for storing DNS records.

	udpResolver := dns.NewUDPResolver(
		logging.WithScope(logger, "dns"),
		cfg.DNS.Clone(),
		cfg.Conn.Clone(),
	)

	dohResolver := dns.NewHTTPSResolver(
		logging.WithScope(logger, "dns"),
		cfg.DNS.Clone(),
		cfg.Conn.Clone(),
	)

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
) (packet.Sniffer, packet.Writer, packet.Sniffer, packet.Writer, error) {
	// create a network detector for passive discovery
	networkDetector := packet.NewNetworkDetector(
		logging.WithScope(logger, "pkt"),
	)

	if err := networkDetector.Start(context.Background()); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error starting network detector: %w", err)
	}

	// Wait for gateway MAC with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gatewayMAC, err := networkDetector.WaitForGatewayMAC(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf(
			"failed to detect gateway (timeout): %w",
			err,
		)
	}

	iface := networkDetector.GetInterface()

	// create a pcap handle for packet capturing.
	tcpHandle, err := packet.NewHandle(iface)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf(
			"error opening pcap handle on interface %s: %w",
			iface.Name,
			err,
		)
	}

	udpHandle, err := packet.NewHandle(iface)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf(
			"error opening pcap handle on interface %s: %w",
			iface.Name,
			err,
		)
	}

	logger.Info().Msg("network info")
	logger.Info().Str("name", iface.Name).
		Str("mac", iface.HardwareAddr.String()).
		Msg(" interface")

	gatewayMACStr := gatewayMAC.String()
	if gatewayMACStr == "" {
		gatewayMACStr = "none"
	}
	logger.Info().
		Str("mac", gatewayMACStr).
		Msg(" gateway (passive detection)")

	hopCache := cache.NewLRUCache(4096)

	// TCP Objects
	tcpSniffer := packet.NewTCPSniffer(
		logging.WithScope(logger, "pkt"),
		hopCache,
		tcpHandle,
		uint8(*cfg.Conn.DefaultFakeTTL),
	)
	tcpSniffer.StartCapturing()

	tcpWriter := packet.NewTCPWriter(
		logging.WithScope(logger, "pkt"),
		tcpHandle,
		iface,
		gatewayMAC,
	)

	// UDP Objects
	udpSniffer := packet.NewUDPSniffer(
		logging.WithScope(logger, "pkt"),
		hopCache,
		udpHandle,
		uint8(*cfg.Conn.DefaultFakeTTL),
	)
	udpSniffer.StartCapturing()

	udpWriter := packet.NewUDPWriter(
		logging.WithScope(logger, "pkt"),
		udpHandle,
		iface,
		gatewayMAC,
	)

	return tcpSniffer, tcpWriter, udpSniffer, udpWriter, nil
}

func createServer(
	logger zerolog.Logger,
	cfg *config.Config,
	resolver dns.Resolver,
) (server.Server, error) {
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

	var tcpSniffer packet.Sniffer
	var tcpWriter packet.Writer
	var udpSniffer packet.Sniffer
	var udpWriter packet.Writer

	if cfg.ShouldEnablePcap() {
		var err error
		tcpSniffer, tcpWriter, udpSniffer, udpWriter, err = createPacketObjects(
			logger,
			cfg,
		)
		if err != nil {
			return nil, err
		}
	}

	desyncer := desync.NewTLSDesyncer(
		tcpWriter,
		tcpSniffer,
	)

	switch *cfg.App.Mode {
	case config.AppModeHTTP:
		httpHandler := http.NewHTTPHandler(logging.WithScope(logger, "hnd"))
		httpsHandler := http.NewHTTPSHandler(
			logging.WithScope(logger, "hnd"),
			desyncer,
			tcpSniffer,
			cfg.HTTPS.Clone(),
			cfg.Conn.Clone(),
		)

		return http.NewHTTPProxy(
			logging.WithScope(logger, "srv"),
			resolver,
			httpHandler,
			httpsHandler,
			ruleMatcher,
			cfg.App.Clone(),
			cfg.Conn.Clone(),
			cfg.Policy.Clone(),
		), nil
	case config.AppModeSOCKS5:
		connectHandler := socks5.NewConnectHandler(
			logging.WithScope(logger, "hnd"),
			desyncer,
			tcpSniffer,
			cfg.App.Clone(),
			cfg.Conn.Clone(),
			cfg.HTTPS.Clone(),
		)
		udpAssociateHandler := socks5.NewUdpAssociateHandler(
			logging.WithScope(logger, "hnd"),
			netutil.NewConnPool(4096, 60*time.Second),
		)
		bindHandler := socks5.NewBindHandler(logging.WithScope(logger, "hnd"))

		return socks5.NewSOCKS5Proxy(
			logging.WithScope(logger, "srv"),
			resolver,
			ruleMatcher,
			connectHandler,
			bindHandler,
			udpAssociateHandler,
			cfg.App.Clone(),
			cfg.Conn.Clone(),
			cfg.Policy.Clone(),
		), nil
	case config.AppModeTUN:
		tcpHandler := tun.NewTCPHandler(
			logging.WithScope(logger, "hnd"),
			ruleMatcher, // For domain-based TLS matching
			cfg.HTTPS.Clone(),
			cfg.Conn.Clone(),
			desyncer,
			tcpSniffer, // For TTL tracking
			"",         // iface and gateway will be set later
			"",
		)

		udpDesyncer := desync.NewUDPDesyncer(
			logging.WithScope(logger, "hnd"),
			udpWriter,
			udpSniffer,
		)

		udpHandler := tun.NewUDPHandler(
			logging.WithScope(logger, "hnd"),
			udpDesyncer,
			cfg.UDP.Clone(),
			cfg.Conn.Clone(),
			netutil.NewConnPool(4096, 60*time.Second),
		)

		return tun.NewTunServer(
			logging.WithScope(logger, "srv"),
			cfg,
			ruleMatcher, // For IP-based matching in server.go
			tcpHandler,
			udpHandler,
		), nil
	default:
		return nil, fmt.Errorf("unknown server mode: %s", *cfg.App.Mode)
	}
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
