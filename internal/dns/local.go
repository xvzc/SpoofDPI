package dns

import (
	"context"
	"net"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

var _ Resolver = (*LocalResolver)(nil)

type LocalResolver struct {
	logger zerolog.Logger

	*net.Resolver
}

func NewLocalResolver(
	logger zerolog.Logger,
) *LocalResolver {
	return &LocalResolver{
		logger:   logger,
		Resolver: &net.Resolver{PreferGo: true},
	}
}

func (lr *LocalResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "local",
			Dest:   "system-dns",
			Cached: CachedStatus{false},
		},
	}
}

func (lr *LocalResolver) Route(ctx context.Context) (Resolver, bool) {
	patternMatched, ok := appctx.PatternMatchedFrom(ctx)
	if !ok {
		return nil, false
	}

	if !patternMatched {
		return lr, true
	}

	return nil, true
}

func filtterAddrs(addrs []net.IPAddr, qTypes []uint16) []net.IPAddr {
	wantsA, wantsAAAA := false, false
	for _, qType := range qTypes {
		switch qType {
		case dns.TypeA:
			wantsA = true
		case dns.TypeAAAA:
			wantsAAAA = true
		}

		if wantsA && wantsAAAA {
			break
		}
	}

	if !wantsA && !wantsAAAA {
		return []net.IPAddr{}
	}

	filteredMap := make(map[string]net.IPAddr)

	for _, addr := range addrs {
		addrStr := addr.IP.String()
		if _, exists := filteredMap[addrStr]; exists {
			continue
		}

		isIPv4 := addr.IP.To4() != nil

		if wantsA && isIPv4 {
			filteredMap[addrStr] = addr
		}

		if wantsAAAA && !isIPv4 {
			filteredMap[addrStr] = addr
		}
	}

	filtered := make([]net.IPAddr, 0, len(filteredMap))
	for _, addr := range filteredMap {
		filtered = append(filtered, addr)
	}

	return filtered
}

func (lr *LocalResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (RecordSet, error) {
	addrs, err := lr.LookupIPAddr(ctx, domain)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	return RecordSet{addrs: filtterAddrs(addrs, qTypes), ttl: 0}, nil
}
