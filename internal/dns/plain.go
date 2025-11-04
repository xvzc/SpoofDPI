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

func (gr *PlainResolver) String() string {
	return fmt.Sprintf("plain-resolver(%s)", gr.server)
}

func (gr *PlainResolver) Resolve(
	ctx context.Context,
	domain string,
) (RecordSet, error) {
	logger := gr.logger.With().Ctx(ctx).Logger()

	resCh := lookupAllTypes(ctx, domain, gr.qTypes, gr.exchange)
	rSet, err := processMessages(ctx, resCh)

	logger.Debug().Msgf("resolved %d records for %s", rSet.Counts(), domain)

	return rSet, err
}

func (gr *PlainResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
) (*dns.Msg, error) {
	resp, _, err := gr.client.ExchangeContext(ctx, msg, gr.server)
	return resp, err
}
