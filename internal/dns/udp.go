package dns

import (
	"context"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var _ Resolver = (*UDPResolver)(nil)

type UDPResolver struct {
	logger zerolog.Logger

	client   *dns.Client
	upstream string
}

func NewUDPResolver(
	logger zerolog.Logger,
	upstream string,
) *UDPResolver {
	return &UDPResolver{
		client:   &dns.Client{},
		upstream: upstream,
		logger:   logger,
	}
}

func (pr *UDPResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "udp",
			Dst:    pr.upstream,
			Cached: CachedStatus{false},
		},
	}
}

func (pr *UDPResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (*RecordSet, error) {
	resCh := lookupAllTypes(ctx, domain, qTypes, pr.exchange)
	return processMessages(ctx, resCh)
}

func (pr *UDPResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
) (*dns.Msg, error) {
	resp, _, err := pr.client.ExchangeContext(ctx, msg, pr.upstream)
	return resp, err
}
