package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xvzc/spoofdpi/internal/cache"
	"github.com/xvzc/spoofdpi/internal/config"
	"github.com/xvzc/spoofdpi/internal/desync"
	"github.com/xvzc/spoofdpi/internal/dns"
	"github.com/xvzc/spoofdpi/internal/logging"
	"github.com/xvzc/spoofdpi/internal/matcher"
	"github.com/xvzc/spoofdpi/internal/netutil"
	"github.com/xvzc/spoofdpi/internal/packet"
	"github.com/xvzc/spoofdpi/internal/server"
	"github.com/xvzc/spoofdpi/internal/server/http"
	"github.com/xvzc/spoofdpi/internal/server/socks5"
	"github.com/xvzc/spoofdpi/internal/server/tun"
	"github.com/xvzc/spoofdpi/internal/session"
)

// Version and commit are set at build time.
var (
	version = "dev"
	commit  = "unknown"
	build   = "unknown"
)

func main() {
	cmd := config.CreateCommand(runApp, version, commit, build)
	appctx, cancel := signal.NotifyContext(
		session.WithNewTraceID(context.Background()),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
	)
	defer cancel()
	if err := cmd.Run(appctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(appctx context.Context, configDir string, cfg *config.Config) {
	var logW io.Writer = os.Stdout
	if !*cfg.App.Silent {
		logW = TuiWriter{}
	}
	logging.SetGlobalLogger(appctx, *cfg.App.LogLevel, logW)

	if !*cfg.App.Silent {
		ctx, cancel := context.WithCancel(appctx)
		defer cancel()

		errChan := make(chan error, 1)

		go func() {
			errChan <- startServer(ctx, configDir, cfg)
		}()

		if err := startTUI(); err != nil {
			logger := log.Logger.With().Ctx(appctx).Logger()
			logger.Error().Err(err).Msg("tui error")
		}

		cancel()
		<-errChan
	} else {
		if err := startServer(appctx, configDir, cfg); err != nil {
			os.Exit(1)
		}
	}
}

func startServer(appctx context.Context, configDir string, cfg *config.Config) error {
	logger := log.Logger.With().Ctx(appctx).Logger()
	logger.Info().Str("version", version).Msg("started spoofdpi")
	if configDir != "" {
		logger.Info().
			Str("dir", configDir).
			Msgf("config file loaded")
	}

	logger.Info().Msgf("app-mode: %s", cfg.App.Mode.String())

	resolver := createResolver(logger, cfg)

	srv, err := createServer(appctx, logger, cfg, resolver)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		return err
	}

	// Start server
	ready := make(chan struct{})
	go func() {
		if err := srv.ListenAndServe(appctx, ready); err != nil {
			logger.Error().Err(err).Msgf("failed to start server: %T", srv)
			// Return to avoid hanging if TUI is waiting?
			// Serve errors are just logged, and TUI stays up.
		}
	}()

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
	if *cfg.Conn.UDPIdleTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.UDPIdleTimeout.Milliseconds())).
			Msgf("udp idle timeout")
	}

	switch *cfg.App.Mode {
	case config.AppModeSOCKS5:
		logger.Warn().Msg("SOCKS5 mode is an EXPERIMENTAL feature")
	case config.AppModeTUN:
		logger.Warn().Msg("TUN mode is an EXPERIMENTAL feature")
	}

	logger.Info().Msgf("server started on %s", srv.Addr())

	<-ready

	// System Proxy Config
	if *cfg.App.AutoConfigureNetwork {
		unset, err := srv.SetNetworkConfig()
		if err != nil {
			logger.Error().Err(err).Msg("failed to set system network config")
			return err
		}
		if unset != nil {
			defer func() {
				if err := unset(); err != nil {
					logger.Error().Err(err).Msg("failed to unset system network config")
				}
			}()
		}
	}

	<-appctx.Done()
	return nil
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
		cache.NewTTLCache[string](
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

	hopCache := cache.NewLRUCache[netutil.IPKey](4096, nil)

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
	appctx context.Context,
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
		udpDesyncer := desync.NewUDPDesyncer(
			logging.WithScope(logger, "dsn"),
			udpWriter,
			udpSniffer,
		)
		udpPool := netutil.NewConnRegistry[netutil.NATKey](4096, 60*time.Second)
		udpPool.RunCleanupLoop(appctx)
		udpAssociateHandler := socks5.NewUdpAssociateHandler(
			logging.WithScope(logger, "hnd"),
			udpPool,
			udpDesyncer,
			cfg.UDP.Clone(),
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
		// Find default interface and gateway before modifying routes
		defaultIface, defaultGateway, err := netutil.GetDefaultInterfaceAndGateway()
		if err != nil {
			return nil, fmt.Errorf("failed to get default interface: %w", err)
		}
		logger.Info().
			Str("interface", defaultIface).
			Str("gateway", defaultGateway).
			Msg("determined default interface and gateway")
		// s.defaultIface = defaultIface
		// s.defaultGateway = defaultGateway

		// Update handlers with network info
		// s.tcpHandler.SetNetworkInfo(defaultIface, defaultGateway)
		// s.udpHandler.SetNetworkInfo(defaultIface, defaultGateway)
		//
		tcpHandler := tun.NewTCPHandler(
			logging.WithScope(logger, "hnd"),
			ruleMatcher, // For domain-based TLS matching
			cfg.HTTPS.Clone(),
			cfg.Conn.Clone(),
			desyncer,
			tcpSniffer, // For TTL tracking
			defaultIface,
			defaultGateway,
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
			defaultIface,
			defaultGateway,
		)

		return tun.NewTunServer(
			logging.WithScope(logger, "srv"),
			cfg,
			ruleMatcher, // For IP-based matching in server.go
			tcpHandler,
			udpHandler,
			defaultIface,
			defaultGateway,
		), nil
	default:
		return nil, fmt.Errorf("unknown server mode: %s", *cfg.App.Mode)
	}
}
