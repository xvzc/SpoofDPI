package dns

import (
	"context"
	"net"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var _ Resolver = (*LocalResolver)(nil)

type LocalResolver struct {
	*net.Resolver

	qTypes []uint16
	logger zerolog.Logger
}

func NewLocalResolver(qTypes []uint16, logger zerolog.Logger) *LocalResolver {
	return &LocalResolver{
		Resolver: &net.Resolver{PreferGo: true},
		qTypes:   qTypes,
		logger:   logger,
	}
}

func (lr *LocalResolver) String() string {
	return "local-resolver"
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
) (RecordSet, error) {
	logger := lr.logger.With().Ctx(ctx).Logger()

	addrs, err := lr.LookupIPAddr(ctx, domain)
	if err != nil {
		return RecordSet{addrs: []net.IPAddr{}, ttl: 0}, err
	}

	logger.Debug().Msgf("resolved %d records for %s", len(addrs), domain)

	return RecordSet{addrs: filtterAddrs(addrs, lr.qTypes), ttl: 0}, nil
}
