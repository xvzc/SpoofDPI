package client

import (
	"context"
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type DNSResult struct {
	msg *dns.Msg
	err error
}

type CustomClient struct {
	server string
}

func NewGeneralClient(server string) *CustomClient {
	return &CustomClient{
		server: server,
	}
}

func (c *CustomClient) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	sendMsg := func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
		clt := &dns.Client{}
		resp, _, err := clt.Exchange(msg, c.server)
		return resp, err
	}

	resultCh := lookup(ctx, host, qTypes, sendMsg)
	addrs, err := processResults(ctx, resultCh)
	return addrs, err
}

func (c *CustomClient) String() string {
	return fmt.Sprintf("custom client(%s)", c.server)
}
