package dns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/rs/zerolog"
)

type RouteResolver struct {
	logger zerolog.Logger

	neighbors []Resolver
}

func NewRouteResolver(
	logger zerolog.Logger,
	neighbors []Resolver,
) *RouteResolver {
	return &RouteResolver{
		logger:    logger,
		neighbors: neighbors,
	}
}

func (rr *RouteResolver) Info() []ResolverInfo {
	var ret []ResolverInfo
	for _, v := range rr.neighbors {
		ret = slices.Concat(ret, v.Info())
	}
	return ret
}

func (rr *RouteResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (RecordSet, error) {
	logger := rr.logger.With().Ctx(ctx).Logger()

	if ip, err := parseIpAddr(domain); err == nil {
		return RecordSet{addrs: []net.IPAddr{(*ip)}, ttl: 0}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resolver := rr.Route(ctx)
	if resolver == nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, fmt.Errorf(
			"error routing dns resolver",
		)
	}

	destInfo := resolver.Info()[0]
	logger.Debug().Msgf("route dns; name=%s; dest=%s; cached=%s;",
		domain, destInfo.Name, destInfo.Cached.String(),
	)

	rSet, err := resolver.Resolve(ctx, domain, qTypes)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	return rSet, nil
}

func (rr *RouteResolver) Route(ctx context.Context) Resolver {
	for _, r := range rr.neighbors {
		if next := r.Route(ctx); next != nil {
			return next
		}
	}

	return nil
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
