package resolver

import (
	"context"
	"net"
)

type Factory interface {
	Get(ctx context.Context, host string, qTypes []uint16) (Resolver, error)
}

type FactoryFunc func(ctx context.Context, host string, qTypes []uint16) (Resolver, error)

func (f FactoryFunc) Get(ctx context.Context, host string, qTypes []uint16) (Resolver, error) {
	return f(ctx, host, qTypes)
}

type FactoryResolver struct {
	f Factory
}

func NewFactoryResolver(f Factory) *FactoryResolver {
	return &FactoryResolver{f}
}

func (c *FactoryResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	resolver, err := c.f.Get(ctx, host, qTypes)
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(ctx, host, qTypes)
}

func (c *FactoryResolver) String() string {
	return "FactoryResolver"
}
