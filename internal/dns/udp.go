package dns

import (
	"context"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

var _ Resolver = (*UDPResolver)(nil)

type UDPResolver struct {
	logger zerolog.Logger

	client      *dns.Client
	defaultOpts *config.DNSOptions
}

func NewUDPResolver(
	logger zerolog.Logger,
	defaultOpts *config.DNSOptions,
) *UDPResolver {
	return &UDPResolver{
		client:      &dns.Client{},
		defaultOpts: defaultOpts,
		logger:      logger,
	}
}

func (ur *UDPResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name: "udp",
			Dst:  ur.defaultOpts.Addr.String(),
		},
	}
}

func (ur *UDPResolver) Resolve(
	ctx context.Context,
	domain string,
	fallback Resolver,
	rule *config.Rule,
) (*RecordSet, error) {
	opts := ur.defaultOpts
	if rule != nil {
		opts = opts.Merge(rule.DNS)
	}

	resCh := lookupAllTypes(
		ctx,
		domain,
		opts.Addr.String(),
		parseQueryTypes(*opts.QType),
		ur.exchange,
	)
	return processMessages(ctx, resCh)
}

func (ur *UDPResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
	upstream string,
) (*dns.Msg, error) {
	logger := logging.WithLocalScope(ctx, ur.logger, "udp_exchange")

	resp, _, err := ur.client.ExchangeContext(ctx, msg, upstream)
	if err != nil {
		logger.Trace().Err(err).Msgf("client returned error")
	}

	return resp, err
}
