package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
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
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runApp(ctx context.Context, cfg *config.Config) {
	if !cfg.Silent {
		printBanner()
	}

	baseLogger := applog.NewLogger(cfg.Debug)
	logger := applog.WithScope(baseLogger, "SETUP")
	logger.Info().Msgf("started spoofdpi %s", version.Version())

	resolver := createResolver(cfg, baseLogger)
	p, err := createProxy(cfg, resolver, baseLogger, logger)
	if err != nil {
		logger.Fatal().Msgf("failed to build app: %s", err)
	}

	// start app
	wait := make(chan struct{}) // wait for setup logs to be printed
	go p.Start(wait)

	// set system-wide proxy configuration.
	if cfg.SetSystemProxy {
		if err := system.SetProxy(cfg.ListenPort.Value(), logger); err != nil {
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

	if cfg.Timeout.Value() > 0 {
		logger.Info().
			Msgf("connection timeout is set to %d ms", cfg.Timeout.Value())
	}

	logger.Info().Msgf(
		"patterns: allow %d, ignore %d", len(cfg.PatternsAllowed), len(cfg.PatternsIgnored),
	)

	logger.Info().Msgf("dns resolvers;")
	dnsInfo := resolver.Info()
	for i := range dnsInfo {
		logger.Info().Msgf(" • %s", dnsInfo[i].String())
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

func createResolver(cfg *config.Config, baseLogger zerolog.Logger) dns.Resolver {
	// create a local resolver.
	localResolver := dns.NewLocalResolver(
		cfg.GenerateDnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(LOCAL)"),
	)

	// create a plain resolver that uses UDP on port 53.
	plainResolver := dns.NewPlainResolver(
		cfg.DnsAddr.Value(),
		cfg.DnsPort.Value(),
		cfg.GenerateDnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(PLAIN)"),
	)

	// create a resolver for DNS over HTTPS (DoH).
	httpsResolver := dns.NewHTTPSResolver(
		cfg.GenerateDOHEndpoint(),
		cfg.GenerateDnsQueryTypes(),
		applog.WithScope(baseLogger, "DNS(HTTPS)"),
	)

	// create a TTL cache for storing DNS records.
	dnsCache := datastruct.NewTTLCache[dns.RecordSet](
		cfg.CacheShards.Value(),
		time.Duration(1*time.Minute),
	)

	// create a resolver that routes DNS queries based on rules.
	return dns.NewRouteResolver(
		cfg.ShouldEnableDOH(),
		localResolver,
		dns.NewCacheResolver(
			dnsCache,
			plainResolver,
			applog.WithScope(baseLogger, "DNS(CACHE)"),
		),
		dns.NewCacheResolver(
			dnsCache,
			httpsResolver,
			applog.WithScope(baseLogger, "DNS(CACHE)"),
		),
		applog.WithScope(baseLogger, "DNS(ROUTE)"),
	)
}

func createProxy(
	cfg *config.Config,
	resolver dns.Resolver,
	baseLogger zerolog.Logger,
	logger zerolog.Logger,
) (*proxy.Proxy, error) {
	var packetInjector *packet.PacketInjector
	var hopTracker *packet.HopTracker
	if cfg.FakeHTTPSPackets.Value() > 0 {
		// find the default network interface.
		iface, err := system.FindDefaultInterface()
		if err != nil {
			return nil, fmt.Errorf("could not find default interface: %s", err)
		}

		logger.Debug().Msgf("interface name is %s", iface.Name)

		// get the IPv4 address for the default network interface.
		ifaceIP, err := system.GetInterfaceIPv4(iface)
		if err != nil {
			return nil, fmt.Errorf("could not find ip address of nic: %s", err)
		}

		// create a pcap handle for packet capturing.
		handle, err := system.CreatePcapHandle(iface)
		if err != nil {
			return nil, fmt.Errorf(
				"error opening pcap handle on interface %s: %s",
				iface.Name,
				err,
			)
		}

		// resolve the MAC address of the gateway.
		gatewayMAC, err := system.ResolveGatewayMACAddr(handle, iface, ifaceIP, logger)
		if err != nil {
			return nil, fmt.Errorf("could not find mac address of gateway: %s", err)
		}

		hopCache := datastruct.NewTTLCache[uint8](
			cfg.CacheShards.Value(),
			time.Duration(3)*time.Minute,
		)
		hopTracker = packet.NewHopTracker(handle, hopCache,
			applog.WithScope(baseLogger, "HOP_TRACK"),
		)
		hopTracker.StartCapturing()

		// create a packet injector instance.
		packetInjector, err = packet.NewPacketInjector(handle, iface, gatewayMAC,
			applog.WithScope(baseLogger, "INJECT"),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating package injector: %s", err)
		}
	} else {
		packetInjector = nil
	}

	// create an HTTP handler.
	httpHandler := proxy.NewHttpHandler(
		applog.WithScope(baseLogger, "HTTP"),
	)

	// create an HTTPS handler.
	httpsHandler := proxy.NewHttpsHandler(
		cfg.WindowSize.Value(),
		cfg.FakeHTTPSPackets.Value(),
		hopTracker,
		packetInjector,
		applog.WithScope(baseLogger, "HTTPS"),
	)

	return proxy.NewProxy(
		cfg.ListenAddr.Value(),
		cfg.ListenPort.Value(),
		config.ParseRegexpSlices(cfg.PatternsAllowed),
		config.ParseRegexpSlices(cfg.PatternsIgnored),
		time.Duration(cfg.Timeout.Value())*time.Millisecond,
		resolver,
		httpHandler,
		httpsHandler,
		applog.WithScope(baseLogger, "PROXY"),
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
