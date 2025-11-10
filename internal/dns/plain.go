package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var _ Resolver = (*PlainResolver)(nil)

type PlainResolver struct {
	client *dns.Client
	server string
	qTypes []uint16
	logger zerolog.Logger
}

func NewPlainResolver(
	server net.IP,
	port uint16,
	qTypes []uint16,
	logger zerolog.Logger,
) *PlainResolver {
	return &PlainResolver{
		client: &dns.Client{},
		server: net.JoinHostPort(server.String(), strconv.Itoa(int(port))),
		qTypes: qTypes,
		logger: logger,
	}
}

func (pr *PlainResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "plain",
			Dest:   pr.server,
			Cached: false,
		},
	}
}

func (pr *PlainResolver) String() string {
	return fmt.Sprintf("plain-resolver(%s)", pr.server)
}

func (pr *PlainResolver) Resolve(
	ctx context.Context,
	domain string,
) (RecordSet, error) {
	logger := pr.logger.With().Ctx(ctx).Logger()

	resCh := lookupAllTypes(ctx, domain, pr.qTypes, pr.exchange)
	rSet, err := processMessages(ctx, resCh)

	logger.Debug().Msgf("resolved %d records for %s", rSet.Counts(), domain)

	return rSet, err
}

func (pr *PlainResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
) (*dns.Msg, error) {
	resp, _, err := pr.client.ExchangeContext(ctx, msg, pr.server)
	return resp, err
}
