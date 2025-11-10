package dns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

type RouteResolver struct {
	enableDOH bool
	local     Resolver
	plain     Resolver
	https     Resolver
	logger    zerolog.Logger
}

func NewRouteResolver(
	enableDOH bool,
	local Resolver,
	plain Resolver,
	https Resolver,
	logger zerolog.Logger,
) *RouteResolver {
	return &RouteResolver{
		enableDOH: enableDOH,
		local:     local,
		plain:     plain,
		https:     https,
		logger:    logger,
	}
}

func (rr *RouteResolver) Info() []ResolverInfo {
	return slices.Concat(rr.local.Info(), rr.plain.Info(), rr.https.Info())
}

func (rr *RouteResolver) String() string {
	return "route-resolver"
}

func (rr *RouteResolver) Resolve(
	ctx context.Context,
	domain string,
) (RecordSet, error) {
	logger := rr.logger.With().Ctx(ctx).Logger()

	if ip, err := parseIpAddr(domain); err == nil {
		return RecordSet{addrs: []net.IPAddr{(*ip)}, ttl: 0}, nil
	}

	patternMatched, ok := appctx.PatternMatchedFrom(ctx)
	if !ok {
		logger.Debug().Msg("failed to retrieve 'patternMatched' value from ctx")
	}

	useSystemDns := !patternMatched

	logger.Debug().
		Msgf("value of 'useSystemDns' is '%s'", strconv.FormatBool(useSystemDns))

	resolver := rr.route(useSystemDns)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	logger.Debug().Msgf("routing %s to %s", domain, resolver)

	startedTime := time.Now()
	rSet, err := resolver.Resolve(ctx, domain)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, fmt.Errorf("%s: %w", rr, err)
	}

	if rSet.Counts() > 0 {
		deltaTime := time.Since(startedTime).Milliseconds()
		logger.Debug().Msgf("dns resolution took %d ms for %s", deltaTime, domain)
		return rSet, nil
	}

	return rSet, fmt.Errorf("could not resolve %s using %s", domain, resolver)
}

func (rr *RouteResolver) route(useSystemDns bool) Resolver {
	if useSystemDns {
		return rr.local
	}

	if rr.enableDOH {
		return rr.https
	}

	return rr.plain
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
