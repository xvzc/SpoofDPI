package handler

import (
	"context"
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/dns/resolver"
)

type DnsHandler interface {
	DoHandle(ctx context.Context, host string, qTypes []uint16, next resolver.Resolver) ([]net.IPAddr, error)
}
type DnsHandlerFunc func(ctx context.Context, host string, qTypes []uint16, r resolver.Resolver) ([]net.IPAddr, error)
type wrappedResolver struct {
	next resolver.Resolver
	h    DnsHandler
}

func (d *wrappedResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	return d.h.DoHandle(ctx, host, qTypes, d.next)
}

func Apply(r resolver.Resolver, h ...DnsHandler) resolver.Resolver {
	for i := len(h) - 1; i >= 0; i-- {
		r = wrap(r, h[i])
	}
	return r
}

func wrap(r resolver.Resolver, handler DnsHandler) resolver.Resolver {
	return &wrappedResolver{r, handler}
}

func (d *wrappedResolver) String() string {
	return fmt.Sprintf("DNSWrapper(%v)", d.next)
}
