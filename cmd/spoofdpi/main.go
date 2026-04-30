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

type SwitchableWriter struct {
	// target is a pointer to an interface, or just the interface itself.
	// We use a pointer to the interface for direct updates.
	target io.Writer
}

func (sw *SwitchableWriter) SetWriter(w io.Writer) {
	// Update the underlying value that the pointer references
	sw.target = w
}

func (sw *SwitchableWriter) Write(p []byte) (n int, err error) {
	// Access the current writer through the pointer
	return sw.target.Write(p)
}

type DelayedWriter struct {
	writer io.Writer
	delay  time.Duration
}

// DelayedWriter is stateless, so value receiver is technically fine,
// but pointer receiver is preferred for consistency in Go.
func (dw *DelayedWriter) Write(p []byte) (n int, err error) {
	if dw.delay > 0 {
		time.Sleep(dw.delay)
	}
	return dw.writer.Write(p)
}

func main() {
	cmd := config.CreateCommand(runApp, version, commit, build)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println("application failed to start")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runApp(mainctx context.Context, configDir string, cfg *config.Config) error {
	appctx, cancel := signal.NotifyContext(
		session.WithNewTraceID(mainctx),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
	)
	defer cancel()

	var writer io.Writer
	// Channel to capture critical TUI execution failures
	if !cfg.App.NoTUI {
		if err := startTUI(cancel); err != nil {
			return fmt.Errorf("failed to start tui: %w", err)
		}
		writer = TUIWriter{}
	} else {
		writer = os.Stdout
	}

	dw := &DelayedWriter{
		writer: writer,
		delay:  29 * time.Millisecond,
	}
	sw := &SwitchableWriter{target: dw}

	logging.SetGlobalLogger(appctx, cfg.App.LogLevel, sw)
	logger := log.Logger.With().Ctx(appctx).Logger()

	logger.Info().Str("version", version).Msg("spoofdpi")
	if configDir != "" {
		logger.Info().
			Str("dir", configDir).
			Msgf("loaded config file")
	} else {
		logger.Warn().
			Msg("config file not found")
		logger.Warn().
			Msg(" please try 'sudo -E spoofdpi' if you expect a configuration to be loaded")
	}

	logger.Info().Str("mode", cfg.App.Mode.String()).Msgf("app")

	switch cfg.App.Mode {
	case config.AppModeSOCKS5:
		logger.Warn().Msg(" 'socks5' mode is an experimental feature")
	case config.AppModeTUN:
		logger.Warn().Msg(" 'tun' mode is an experimental feature")
	}

	resolver := createResolver(logger, cfg)
	srv, err := createServer(appctx, logger, cfg, resolver)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		return err
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
		Uint8("chunk-size", uint8(cfg.HTTPS.ChunkSize)).
		Bool("disorder", cfg.HTTPS.Disorder).
		Msg(" split")

	logger.Info().
		Uint8("count", uint8(cfg.HTTPS.FakeCount)).
		Msg(" fake")

	if cfg.Conn.DNSTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.DNSTimeout.Milliseconds())).
			Msgf("dns connection timeout")
	}
	if cfg.Conn.TCPTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.TCPTimeout.Milliseconds())).
			Msgf("tcp connection timeout")
	}
	if cfg.Conn.UDPIdleTimeout > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Conn.UDPIdleTimeout.Milliseconds())).
			Msgf("udp idle timeout")
	}

	time.Sleep(300 * time.Millisecond)
	err = srv.ListenAndServe(appctx)
	if err != nil {
		logger.Error().Err(err).Msg("server failed to start")
	} else {
		logger.Info().Msgf("server started on %s", srv.Addr())
		if cfg.App.AutoConfigureNetwork {
			unset, err := srv.AutoConfigureNetwork(appctx)
			if err != nil {
				logger.Error().Err(err).Msg("failed to set system network config")
			} else if unset != nil {
				defer unset()
			}
		}
	}

	sw.SetWriter(writer)

	<-appctx.Done()

	return nil
}

func createResolver(logger zerolog.Logger, cfg *config.Config) dns.Resolver {
	// create a TTL cache for storing DNS records.

	udpResolver := dns.NewUDPResolver(
		logging.WithScope(logger, "dns"),
		&cfg.DNS,
		&cfg.Conn,
	)

	dohResolver := dns.NewHTTPSResolver(
		logging.WithScope(logger, "dns"),
		&cfg.DNS,
		&cfg.Conn,
	)

	sysResolver := dns.NewSystemResolver(
		logging.WithScope(logger, "dns"),
		&cfg.DNS,
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
		&cfg.DNS,
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
		uint8(cfg.Conn.DefaultFakeTTL),
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
		uint8(cfg.Conn.DefaultFakeTTL),
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

	defaultRoute, err := netutil.DefaultRoute()
	if err != nil {
		return nil, fmt.Errorf("failed to find default route: %w", err)
	}

	switch cfg.App.Mode {
	case config.AppModeHTTP:
		httpHandler := http.NewHTTPHandler(logging.WithScope(logger, "hnd"))
		httpsHandler := http.NewHTTPSHandler(
			logging.WithScope(logger, "hnd"),
			desyncer,
			tcpSniffer,
			&cfg.HTTPS,
			&cfg.Conn,
		)

		sysNet := http.NewHTTPSystemNetwork(
			logging.WithScope(logger, "sys"),
			defaultRoute,
		)

		return http.NewHTTPProxy(
			logging.WithScope(logger, "srv"),
			resolver,
			httpHandler,
			httpsHandler,
			ruleMatcher,
			sysNet,
			&cfg.App,
			&cfg.Conn,
			&cfg.Policy,
		), nil
	case config.AppModeSOCKS5:
		connectHandler := socks5.NewConnectHandler(
			logging.WithScope(logger, "hnd"),
			desyncer,
			tcpSniffer,
			&cfg.App,
			&cfg.Conn,
			&cfg.HTTPS,
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
			&cfg.UDP,
		)
		bindHandler := socks5.NewBindHandler(logging.WithScope(logger, "hnd"))

		return socks5.NewSOCKS5Proxy(
			logging.WithScope(logger, "srv"),
			resolver,
			ruleMatcher,
			connectHandler,
			bindHandler,
			udpAssociateHandler,
			socks5.NewSOCKS5SystemNetwork(
				logging.WithScope(logger, "sys"),
				defaultRoute,
			),
			&cfg.App,
			&cfg.Conn,
			&cfg.Policy,
		), nil
	case config.AppModeTUN:
		if err != nil {
			return nil, fmt.Errorf("failed to get default route: %w", err)
		}
		logger.Info().
			Str("interface", defaultRoute.Iface.Name).
			Str("gateway", defaultRoute.Gateway.String()).
			Msg("determined default interface and gateway")

		// Get FIB ID from config (FreeBSD only, default to 1)
		fibID := cfg.App.FreebsdFIB

		tcpHandler := tun.NewTCPHandler(
			logging.WithScope(logger, "hnd"),
			ruleMatcher, // For domain-based TLS matching
			&cfg.HTTPS,
			&cfg.Conn,
			desyncer,
			tcpSniffer, // For TTL tracking
		)

		udpDesyncer := desync.NewUDPDesyncer(
			logging.WithScope(logger, "hnd"),
			udpWriter,
			udpSniffer,
		)

		udpHandler := tun.NewUDPHandler(
			logging.WithScope(logger, "hnd"),
			udpDesyncer,
			&cfg.UDP,
			&cfg.Conn,
		)

		sysNet, err := tun.NewTUNSystemNetwork(
			logging.WithScope(logger, "sys"),
			defaultRoute,
			fibID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create sysnet: %w", err)
		}

		return tun.NewTUNServer(
			logging.WithScope(logger, "srv"),
			cfg,
			ruleMatcher, // For IP-based matching in server.go
			tcpHandler,
			udpHandler,
			sysNet,
		), nil
	default:
		return nil, fmt.Errorf("unknown server mode: %s", cfg.App.Mode)
	}
}
