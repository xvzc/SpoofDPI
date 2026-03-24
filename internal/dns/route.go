package dns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

type RouteResolver struct {
	logger         zerolog.Logger
	https          Resolver
	udp            Resolver
	system         Resolver
	cache          Resolver
	defaultDNSOpts *config.DNSOptions
}

func NewRouteResolver(
	logger zerolog.Logger,
	doh Resolver,
	udp Resolver,
	sys Resolver,
	cache Resolver,
	defaultDNSOpts *config.DNSOptions,
) *RouteResolver {
	return &RouteResolver{
		logger:         logger,
		https:          doh,
		udp:            udp,
		system:         sys,
		cache:          cache,
		defaultDNSOpts: defaultDNSOpts,
	}
}

func (rr *RouteResolver) Info() []ResolverInfo {
	return slices.Concat(
		rr.udp.Info(),
		rr.https.Info(),
		rr.system.Info(),
		rr.cache.Info(),
	)
}

func (rr *RouteResolver) Resolve(
	ctx context.Context,
	domain string,
	fallback Resolver,
	rule *config.Rule,
) (*RecordSet, error) {
	opts := rr.defaultDNSOpts.Clone()
	if rule != nil {
		opts = opts.Merge(rule.DNS)
	}

	logger := logging.WithLocalScope(ctx, rr.logger, "route")

	// 1. Check for IP address in domain
	if ip, err := parseIpAddr(domain); err == nil {
		return &RecordSet{Addrs: []net.IP{ip}, TTL: 0}, nil
	}

	// 4. Handle ROUTE rule (or default)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resolver := rr.route(opts)
	if resolver == nil {
		return nil, fmt.Errorf("no resolver available for spec")
	}

	resolverInfo := resolver.Info()[0]
	logger.Debug().Str("mode", resolverInfo.Name).Bool("cache", lo.FromPtr(opts.Cache)).
		Msgf("ready to resolve")

	t1 := time.Now()
	var rSet *RecordSet
	var err error
	if *opts.Mode != config.DNSModeSystem && *opts.Cache {
		rSet, err = rr.cache.Resolve(ctx, domain, resolver, rule)
	} else {
		rSet, err = resolver.Resolve(ctx, domain, nil, rule)
	}

	if err != nil {
		return nil, err
	}

	logger.Debug().
		Str("domain", domain).
		Int("len", len(rSet.Addrs)).
		Str("took", fmt.Sprintf("%.3fms", float64(time.Since(t1).Microseconds())/1000.0)).
		Msgf("dns lookup ok")

	return rSet, nil
}

func (rr *RouteResolver) route(attrs *config.DNSOptions) Resolver {
	switch *attrs.Mode {
	case config.DNSModeHTTPS:
		return rr.https
	case config.DNSModeUDP:
		return rr.udp
	case config.DNSModeSystem:
		return rr.system
	default:
		return rr.system
	}
}

func parseIpAddr(addr string) (net.IP, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, fmt.Errorf("%s is not an ip address", addr)
	}

	return ip, nil
}
