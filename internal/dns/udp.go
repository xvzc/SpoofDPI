package dns

import (
	"context"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/spoofdpi/internal/config"
	"github.com/xvzc/spoofdpi/internal/logging"
)

var _ Resolver = (*UDPResolver)(nil)

type UDPResolver struct {
	logger zerolog.Logger

	client          *dns.Client
	defaultDNSOpts  *config.DNSOptions
	defaultConnOpts *config.ConnOptions
}

func NewUDPResolver(
	logger zerolog.Logger,
	defaultDNSOpts *config.DNSOptions,
	defaultConnOpts *config.ConnOptions,
) *UDPResolver {
	return &UDPResolver{
		client: &dns.Client{
			Timeout: defaultConnOpts.DNSTimeout,
		},
		defaultDNSOpts:  defaultDNSOpts,
		defaultConnOpts: defaultConnOpts,
		logger:          logger,
	}
}

func (ur *UDPResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name: "udp",
			Dst:  ur.defaultDNSOpts.Addr.String(),
		},
	}
}

func (ur *UDPResolver) Resolve(
	ctx context.Context,
	domain string,
	fallback Resolver,
	rule *config.Rule,
) (*RecordSet, error) {
	opts := ur.defaultDNSOpts
	if rule != nil {
		opts = &rule.DNS
	}

	resCh := lookupAllTypes(
		ctx,
		domain,
		opts.Addr.String(),
		parseQueryTypes(opts.QType),
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
