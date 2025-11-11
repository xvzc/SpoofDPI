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
	"github.com/xvzc/SpoofDPI/internal/appctx"
	"github.com/xvzc/SpoofDPI/internal/applog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/datastruct"
	"github.com/xvzc/SpoofDPI/internal/dns"
	"github.com/xvzc/SpoofDPI/internal/packet"
	"github.com/xvzc/SpoofDPI/internal/proxy"
	"github.com/xvzc/SpoofDPI/internal/system"
	"github.com/xvzc/SpoofDPI/version"
)

func main() {
	cmd := config.CreateCommand(runApp)
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

	logger := applog.WithScope(log.Logger, "APP(setup)").With().Ctx(ctx).Logger()
	logger.Info().Msgf("started spoofdpi; %s;", version.Version())
	if configDir != "" {
		logger.Info().Msgf("config file; %s;", configDir)
	}

	resolver := createResolver(logger, cfg)
	p, err := createProxy(logger, cfg, resolver)
	if err != nil {
		logger.Fatal().Msgf("failed to build app: %s", err)
	}

	// start app
	wait := make(chan struct{}) // wait for setup logs to be printed
	go p.ListenAndServe(wait)

	// set system-wide proxy configuration.
	if cfg.SetSystemProxy {
		if err := system.SetProxy(logger, cfg.ListenPort.Value()); err != nil {
			logger.Fatal().Msgf("error while changing proxy settings: %s", err)
		}
		defer func() {
			if err := system.UnsetProxy(logger); err != nil {
				logger.Fatal().Msgf("error while disabling proxy: %s", err)
			}
		}()
	} else {
		logger.Info().Msgf("use '--system-proxy' to automatically set system proxy")
	}

	logger.Info().Msgf("dns info;")
	logger.Info().Msgf(" query type; %d;", len(cfg.GenerateDnsQueryTypes()))
	logger.Info().Msgf(" resolvers;")
	dnsInfo := resolver.Info()
	for i := range dnsInfo {
		logger.Info().Msgf("  • %s", dnsInfo[i].String())
	}

	logger.Info().Msgf("window size; %d;", cfg.WindowSize.Value())
	logger.Info().Msgf(
		"patterns; allow=%d; ignore=%d;", len(cfg.PatternsAllowed), len(cfg.PatternsIgnored),
	)

	if cfg.Timeout.Value() > 0 {
		logger.Info().
			Msgf("connection timeout; %dms;", cfg.Timeout.Value())
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
	// create a local resolver.
	localResolver := dns.NewLocalResolver(
		applog.WithScope(logger, "DNS(local)"),
	)

	// create a plain resolver that uses UDP on port 53.
	plainResolver := dns.NewPlainResolver(
		applog.WithScope(logger, "DNS(plain)"),
		cfg.DnsAddr.Value(),
		cfg.DnsPort.Value(),
	)

	// create a resolver for DNS over HTTPS (DoH).
	httpsResolver := dns.NewHTTPSResolver(
		applog.WithScope(logger, "DNS(https)"),
		cfg.ShouldEnableDOH(),
		cfg.GenerateDOHEndpoint(),
	)

	// create a TTL cache for storing DNS records.
	dnsCache := datastruct.NewTTLCache[dns.RecordSet](
		cfg.CacheShards.Value(),
		time.Duration(1*time.Minute),
	)

	// create a resolver that routes DNS queries based on rules.
	return dns.NewRouteResolver(
		applog.WithScope(logger, "DNS(route)"),
		[]dns.Resolver{
			dns.NewCacheResolver(
				applog.WithScope(logger, "DNS(cache)"),
				dnsCache,
				httpsResolver,
			),
			dns.NewCacheResolver(
				applog.WithScope(logger, "DNS(cache)"),
				dnsCache,
				plainResolver,
			),
			localResolver,
		},
	)
}

func createProxy(
	logger zerolog.Logger,
	cfg *config.Config,
	resolver dns.Resolver,
) (*proxy.Proxy, error) {
	var packetInjector *packet.PacketInjector
	var hopTracker *packet.HopTracker
	if cfg.FakeHTTPSPackets.Value() > 0 {
		// find the default network interface.
		iface, err := system.FindDefaultInterface()
		if err != nil {
			return nil, fmt.Errorf("could not find default interface: %w", err)
		}

		// get the IPv4 address for the default network interface.
		ifaceIP, err := system.GetInterfaceIPv4(iface)
		if err != nil {
			return nil, fmt.Errorf("could not find ip address of nic: %w", err)
		}
		logger.Info().Msgf("interface name; %s;", iface.Name)
		logger.Info().Msgf("interface mac; %s;", iface.HardwareAddr)
		logger.Info().Msgf("interface ip; %s;", ifaceIP)

		// create a pcap handle for packet capturing.
		handle, err := system.CreatePcapHandle(iface)
		if err != nil {
			return nil, fmt.Errorf(
				"error opening pcap handle on interface %s: %w",
				iface.Name,
				err,
			)
		}

		// resolve the MAC address of the gateway.
		gatewayMAC, err := system.ResolveGatewayMACAddr(logger, handle, iface, ifaceIP)
		if err != nil {
			return nil, fmt.Errorf("could not find mac address of gateway: %w", err)
		}
		logger.Info().Msgf("gateway mac; %s;", gatewayMAC.String())

		hopCache := datastruct.NewTTLCache[uint8](
			cfg.CacheShards.Value(),
			time.Duration(3)*time.Minute,
		)
		hopTracker = packet.NewHopTracker(
			applog.WithScope(logger, "PKT(track)"),
			hopCache,
			handle,
		)
		hopTracker.StartCapturing()

		// create a packet injector instance.
		packetInjector, err = packet.NewPacketInjector(
			applog.WithScope(logger, "PKT(write)"),
			gatewayMAC,
			handle,
			iface,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating package injector: %w", err)
		}
	} else {
		packetInjector = nil
	}

	// create an HTTP handler.
	httpHandler := proxy.NewHttpHandler(
		applog.WithScope(logger, "PXY(.main)"),
	)

	// create an HTTPS handler.
	httpsHandler := proxy.NewHttpsHandler(
		applog.WithScope(logger, "HDL(https)"),
		hopTracker,
		packetInjector,
		cfg.WindowSize.Value(),
		cfg.FakeHTTPSPackets.Value(),
	)

	return proxy.NewProxy(
		applog.WithScope(logger, "HDL(.http)"),
		resolver,
		httpHandler,
		httpsHandler,
		cfg.ListenAddr.Value(),
		cfg.ListenPort.Value(),
		config.ParseRegexpSlices(cfg.PatternsAllowed),
		config.ParseRegexpSlices(cfg.PatternsIgnored),
		cfg.GenerateDnsQueryTypes(),
		time.Duration(cfg.Timeout.Value())*time.Millisecond,
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
