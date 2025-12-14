package dns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

type RouteResolver struct {
	logger      zerolog.Logger
	https       Resolver
	udp         Resolver
	system      Resolver
	cache       Resolver
	defaultOpts *config.DNSOptions
}

func NewRouteResolver(
	logger zerolog.Logger,
	doh Resolver,
	udp Resolver,
	sys Resolver,
	cache Resolver,
	defaultOpts *config.DNSOptions,
) *RouteResolver {
	return &RouteResolver{
		logger:      logger,
		https:       doh,
		udp:         udp,
		system:      sys,
		cache:       cache,
		defaultOpts: defaultOpts,
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
	attrs := rr.defaultOpts
	if rule != nil {
		attrs = attrs.Merge(rule.DNS)
	}

	logger := logging.WithLocalScope(ctx, rr.logger, "route")

	// 1. Check for IP address in domain
	if ip, err := parseIpAddr(domain); err == nil {
		return &RecordSet{Addrs: []net.IPAddr{*ip}, TTL: 0}, nil
	}

	// 4. Handle ROUTE rule (or default)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resolver := rr.route(attrs)
	if resolver == nil {
		return nil, fmt.Errorf("no resolver available for spec")
	}

	resolverInfo := resolver.Info()[0]
	logger.Trace().
		Str("name", resolverInfo.Name).
		Bool("cache", ptr.FromPtr(attrs.Cache)).
		Msgf("ready to resolve")

	var rSet *RecordSet
	var err error
	if *attrs.Mode != config.DNSModeSystem && *attrs.Cache {
		rSet, err = rr.cache.Resolve(ctx, domain, resolver, rule)
	} else {
		rSet, err = resolver.Resolve(ctx, domain, nil, rule)
	}

	return rSet, err
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

func parseIpAddr(addr string) (*net.IPAddr, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, fmt.Errorf("%s is not an ip address", addr)
	}

	ipAddr := &net.IPAddr{
		IP: ip,
	}

	return ipAddr, nil
}
