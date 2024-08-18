package client

import (
	"context"
	"net"
)

type SystemClient struct {
	client *net.Resolver
}

func NewSystemClient() *SystemClient {
	return &SystemClient{
		client: &net.Resolver{PreferGo: true},
	}
}

func (c *SystemClient) String() string {
	return "system client"
}

func (c *SystemClient) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	addrs, err := c.client.LookupIPAddr(ctx, host)
	if err != nil {
		return []net.IPAddr{}, err
	}
	return addrs, nil
}
