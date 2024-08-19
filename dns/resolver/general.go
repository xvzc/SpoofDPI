package resolver

import (
	"context"
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type GeneralResolver struct {
	client *dns.Client
	server string
}

func NewGeneralResolver(server string) *GeneralResolver {
	return &GeneralResolver{
		client: &dns.Client{},
		server: server,
	}
}

func (r *GeneralResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	resultCh := lookupAllTypes(ctx, host, qTypes, r.exchange)
	addrs, err := processResults(ctx, resultCh)
	return addrs, err
}

func (r *GeneralResolver) String() string {
	return fmt.Sprintf("general resolver(%s)", r.server)
}

func (r *GeneralResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	resp, _, err := r.client.Exchange(msg, r.server)
	return resp, err
}
