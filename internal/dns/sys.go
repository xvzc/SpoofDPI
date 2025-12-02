package dns

import (
	"context"
	"net"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var _ Resolver = (*SysResolver)(nil)

type SysResolver struct {
	logger zerolog.Logger

	*net.Resolver
}

func NewSystemResolver(
	logger zerolog.Logger,
) *SysResolver {
	return &SysResolver{
		logger:   logger,
		Resolver: &net.Resolver{PreferGo: true},
	}
}

func (lr *SysResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "sys",
			Dst:    "system-resolver",
			Cached: CachedStatus{false},
		},
	}
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

func (lr *SysResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (*RecordSet, error) {
	addrs, err := lr.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}

	return &RecordSet{addrs: filtterAddrs(addrs, qTypes), ttl: 0}, nil
}
