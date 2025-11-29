package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/applog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/datastruct/cache"
	"github.com/xvzc/SpoofDPI/internal/datastruct/tree"
	"github.com/xvzc/SpoofDPI/internal/desync"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proto"
	"github.com/xvzc/SpoofDPI/internal/proxy"
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
	ctx := appctx.WithNewTraceID(context.Background())
	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runApp(ctx context.Context, configDir string, cfg *config.Config) {
	if !cfg.Silent {
		printBanner()
	}

	applog.SetGlobalLogger(ctx, cfg.LogLevel.Value())

	logger := log.Logger.With().Ctx(ctx).Logger()
	logger.Info().Str("version", version).Msg("started spoofdpi")
	if configDir != "" {
		logger.Info().
			Str("dir", configDir).
			Msgf("config file loaded")
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
	if cfg.SetSystemProxy {
		if err := system.SetProxy(logger, cfg.ListenPort.Value()); err != nil {
			logger.Fatal().Err(err).Msg("failed to enable system proxy")
		}
		defer func() {
			if err := system.UnsetProxy(logger); err != nil {
				logger.Fatal().Err(err).Msg("failed to disable system proxy")
			}
		}()
	} else {
		logger.Info().Bool("enabled", cfg.SetSystemProxy).
			Msg("use `--system-proxy` to automatically set system proxy")
	}

	logger.Info().Msg("dns info")
	logger.Info().
		Int("len", len(cfg.GenerateDnsQueryTypes())).
		Msg(" query type")
	logger.Info().Msgf(" resolvers")
	dnsInfo := resolver.Info()
	for i := range dnsInfo {
		logger.Info().
			Str("alias", dnsInfo[i].Name).
			Str("cached", dnsInfo[i].Cached.String()).
			Str("dst", dnsInfo[i].Dst).
			Msgf("  item")
	}

	logger.Info().
		Uint8("value", cfg.WindowSize.Value()).
		Msgf("window size")
	logger.Info().
		Int("len", len(cfg.DomainPolicySlice)).
		Bool("auto", cfg.AutoPolicy).
		Msgf("policy")

	if cfg.Timeout.Value() > 0 {
		logger.Info().
			Str("value", fmt.Sprintf("%dms", cfg.Timeout.Value())).
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
	dnsCache := cache.NewTTLCache(
		cfg.CacheShards.Value(),
		time.Duration(1*time.Minute),
	)

	var mainResolver dns.Resolver
	if cfg.ShouldEnableDOH() { // create a cached DOH resolver if enabled.
		mainResolver = dns.NewCacheResolver(
			applog.WithScope(logger, "dns"),
			dnsCache,
			dns.NewHTTPSResolver(
				applog.WithScope(logger, "dns"),
				cfg.GenerateDOHEndpoint(),
			),
		)
	} else { // create a cached plain resolver if DOH is disabled.
		mainResolver = dns.NewCacheResolver(
			applog.WithScope(logger, "dns"),
			dnsCache,
			dns.NewPlainResolver(
				applog.WithScope(logger, "dns"),
				cfg.DnsAddr.Value(),
				cfg.DnsPort.Value(),
			),
		)
	}

	// create a non-cached local resolver.
	localResolver := dns.NewLocalResolver(applog.WithScope(logger, "dns"))

	// create a resolver that routes DNS queries based on rules.
	return dns.NewRouteResolver(
		applog.WithScope(logger, "dns"),
		mainResolver,
		localResolver,
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
	logger.Info().Msg("interface info")
	logger.Info().Str("name", iface.Name).Msg(" item")
	logger.Info().Str("mac", iface.HardwareAddr.String()).Msg(" item")
	logger.Info().Str("ip", ifaceIP.String()).Msg(" item")

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
	logger.Info().Msgf("gateway info")
	logger.Info().Str("mac", gatewayMAC.String()).Msgf(" item")
	logger.Info().Str("ip", gatewayIP.String()).Msgf(" item")

	hopCache := cache.NewLRUCache(4096)
	hopTracker := packet.NewHopTracker(
		applog.WithScope(logger, "pkt"),
		hopCache,
		handle,
		cfg.DefaultTTL.Value(),
	)
	hopTracker.StartCapturing()

	// create a packet injector instance.
	packetInjector, err := packet.NewPacketInjector(
		applog.WithScope(logger, "pkt"),
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
) (*proxy.Proxy, error) {
	var domainTree tree.SearchTree
	if len(cfg.DomainPolicySlice) > 0 || cfg.AutoPolicy {
		domainTree = config.ParseDomainSearchTree(cfg.DomainPolicySlice)
	}

	// create an HTTP handler.
	httpHandler := proxy.NewHTTPHandler(applog.WithScope(logger, "hnd"))

	// create an HTTPS handler.
	tlsDefault := desync.NewTLSDefault()
	tlsBypass := tlsDefault
	if cfg.WindowSize.Value() > 0 {
		tlsBypass = desync.NewTLSSplit(
			cfg.Disorder,
			cfg.DefaultTTL.Value(),
			cfg.WindowSize.Value(),
		)
	}

	var hopTracker *packet.HopTracker
	var packetInjector *packet.Injector
	if cfg.FakeCount.Value() > 0 {
		var err error
		hopTracker, packetInjector, err = createPacketObjects(logger, cfg)
		if err != nil {
			return nil, err
		}

		fakeMsg, err := proto.ReadTLSMessage(bytes.NewReader(desync.FakeClientHello))
		if err != nil {
			return nil, err
		}

		tlsBypass = desync.NewTLSFake(
			tlsBypass,
			hopTracker,
			packetInjector,
			cfg.WindowSize.Value(),
			cfg.FakeCount.Value(),
			fakeMsg,
		)
	}

	httpsHandler := proxy.NewHTTPSHandler(
		applog.WithScope(logger, "hnd"),
		tlsDefault,
		tlsBypass,
	)

	return proxy.NewProxy(
		applog.WithScope(logger, "pxy"),
		resolver,
		httpHandler,
		httpsHandler,
		domainTree,
		hopTracker,
		proxy.ProxyOptions{
			AutoPolicy:    cfg.AutoPolicy,
			ListenAddr:    cfg.ListenAddr.Value(),
			ListenPort:    cfg.ListenPort.Value(),
			DNSQueryTypes: cfg.GenerateDnsQueryTypes(),
			Timeout:       time.Duration(cfg.Timeout.Value()) * time.Millisecond,
		},
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
