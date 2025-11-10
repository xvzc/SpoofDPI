package dns

import (
	"context"
	"net"
	"strconv"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

var _ Resolver = (*PlainResolver)(nil)

type PlainResolver struct {
	logger zerolog.Logger

	upstream string

	client *dns.Client
}

func NewPlainResolver(
	logger zerolog.Logger,
	server net.IP,
	port uint16,
) *PlainResolver {
	return &PlainResolver{
		client:   &dns.Client{},
		upstream: net.JoinHostPort(server.String(), strconv.Itoa(int(port))),
		logger:   logger,
	}
}

func (pr *PlainResolver) Route(ctx context.Context) (Resolver, bool) {
	patternMatched, ok := appctx.PatternMatchedFrom(ctx)
	if !ok {
		return nil, false
	}

	if patternMatched {
		return pr, true
	}

	return nil, true
}

func (pr *PlainResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "plain",
			Dest:   pr.upstream,
			Cached: CachedStatus{false},
		},
	}
}

func (pr *PlainResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (RecordSet, error) {
	resCh := lookupAllTypes(ctx, domain, qTypes, pr.exchange)
	rSet, err := processMessages(ctx, resCh)

	return rSet, err
}

func (pr *PlainResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
) (*dns.Msg, error) {
	resp, _, err := pr.client.ExchangeContext(ctx, msg, pr.upstream)
	return resp, err
}
