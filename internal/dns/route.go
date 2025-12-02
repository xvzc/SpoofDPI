package dns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/logging"
	"github.com/xvzc/SpoofDPI/internal/session"
)

type RouteResolver struct {
	logger zerolog.Logger

	mainResolver  Resolver
	localResolver Resolver
}

func NewRouteResolver(
	logger zerolog.Logger,
	mainResolver Resolver,
	localResolver Resolver,
) *RouteResolver {
	return &RouteResolver{
		logger:        logger,
		mainResolver:  mainResolver,
		localResolver: localResolver,
	}
}

func (rr *RouteResolver) Info() []ResolverInfo {
	return slices.Concat(rr.mainResolver.Info(), rr.localResolver.Info())
}

func (rr *RouteResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (*RecordSet, error) {
	logger := logging.WithLocalScope(rr.logger, ctx, "route")

	if ip, err := parseIpAddr(domain); err == nil {
		return &RecordSet{addrs: []net.IPAddr{(*ip)}, ttl: 0}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resolver := rr.route(ctx)
	if resolver == nil {
		return nil, fmt.Errorf(
			"error routing dns resolver",
		)
	}

	resolverInfo := resolver.Info()[0]
	logger.Trace().
		Str("cached", resolverInfo.Cached.String()).
		Str("name", resolverInfo.Name).
		Msgf("next resolver info")

	rSet, err := resolver.Resolve(ctx, domain, qTypes)
	if err != nil {
		return nil, err
	}

	return rSet, nil
}

func (rr *RouteResolver) route(ctx context.Context) Resolver {
	policyIncluded, _ := session.PolicyIncludedFrom(ctx)
	if policyIncluded {
		return rr.mainResolver
	}

	return rr.localResolver
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
