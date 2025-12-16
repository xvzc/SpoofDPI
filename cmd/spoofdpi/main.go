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
	"github.com/xvzc/SpoofDPI/internal/handler"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/matcher"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proxy"
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
		Str("default", cfg.HTTPS.SplitMode.String()).
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
) (*packet.HopTracker, *packet.Injector, error) {
	// find the default network interface.
	iface, err := system.FindDefaultInterface()
	if err != nil {
		return nil, nil, fmt.Errorf("could not find default interface: %w", err)
	}

	// get the IPv4 address for the default network interface.
	ifaceIP, err := system.GetInterfaceIPv4(iface)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find IP address of NIC: %w", err)
	}

	// create a pcap handle for packet capturing.
	handle, err := packet.NewHandle(iface)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"error opening pcap handle on interface %s: %w",
			iface.Name,
			err,
		)
	}

	gatewayIP, err := system.FindGatewayIPAddr()
	if err != nil {
		return nil, nil, fmt.Errorf("could not find IP address of gateway: %w", err)
	}

	// resolve the MAC address of the gateway.
	gatewayMAC, err := system.ResolveGatewayMACAddr(
		logger,
		handle,
		gatewayIP,
		iface,
		ifaceIP,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find MAC address of gateway: %w", err)
	}

	logger.Info().Msg("network info")
	logger.Info().Str("name", iface.Name).
		Str("mac", iface.HardwareAddr.String()).
		Str("ip", ifaceIP.String()).
		Msg(" interface")
	logger.Info().
		Str("mac", gatewayMAC.String()).
		Str("ip", gatewayIP.String()).
		Msgf(" gateway")

	hopCache := cache.NewLRUCache(4096)
	hopTracker := packet.NewHopTracker(
		logging.WithScope(logger, "pkt"),
		hopCache,
		handle,
		packet.HopTrackerAttrs{
			DefaultTTL: uint8(*cfg.Server.DefaultTTL),
		},
	)
	hopTracker.StartCapturing()

	// create a packet injector instance.
	packetInjector, err := packet.NewPacketInjector(
		logging.WithScope(logger, "pkt"),
		handle,
		iface,
		gatewayMAC,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating packet injector: %w", err)
	}

	return hopTracker, packetInjector, nil
}

func createProxy(
	logger zerolog.Logger,
	cfg *config.Config,
	resolver dns.Resolver,
) (proxy.Proxy, error) {
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
	httpHandler := handler.NewHTTPHandler(logging.WithScope(logger, "hnd"))

	var hopTracker *packet.HopTracker
	var packetInjector *packet.Injector
	if cfg.ShouldEnablePcap() {
		var err error
		hopTracker, packetInjector, err = createPacketObjects(logger, cfg)
		if err != nil {
			return nil, err
		}
	}

	httpsHandler := handler.NewHTTPSHandler(
		logging.WithScope(logger, "hnd"),
		desync.NewTLSDesyncer(
			packetInjector,
			hopTracker,
			&desync.TLSDesyncerAttrs{DefaultTTL: *cfg.Server.DefaultTTL},
		),
		hopTracker,
		cfg.HTTPS.Clone(),
	)

	return proxy.NewHTTPProxy(
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
