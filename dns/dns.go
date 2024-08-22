package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/xvzc/SpoofDPI/dns/resolver"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

const scopeDNS = "DNS"

type Resolver interface {
	Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error)
	String() string
}

type Dns struct {
	host          string
	port          string
	systemClient  Resolver
	generalClient Resolver
	dohClient     Resolver
}

func NewDns(config *util.Config) *Dns {
	addr := *config.DnsAddr
	port := strconv.Itoa(*config.DnsPort)

	return &Dns{
		host:          *config.DnsAddr,
		port:          port,
		systemClient:  resolver.NewSystemResolver(),
		generalClient: resolver.NewGeneralResolver(net.JoinHostPort(addr, port)),
		dohClient:     resolver.NewDOHResolver(addr),
	}
}

func (d *Dns) ResolveHost(ctx context.Context, host string, enableDoh bool, useSystemDns bool) (string, error) {
	ctx = util.GetCtxWithScope(ctx, scopeDNS)
	logger := log.GetCtxLogger(ctx)

	if ip, err := parseIpAddr(host); err == nil {
		return ip.String(), nil
	}

	clt := d.clientFactory(enableDoh, useSystemDns)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	logger.Debug().Msgf("resolving %s using %s", host, clt)

	t := time.Now()

	addrs, err := clt.Resolve(ctx, host, []uint16{dns.TypeAAAA, dns.TypeA})
	// addrs, err := clt.Resolve(ctx, host, []uint16{dns.TypeAAAA})
	if err != nil {
		return "", fmt.Errorf("%s: %w", clt, err)
	}

	if len(addrs) > 0 {
		d := time.Since(t).Milliseconds()
		logger.Debug().Msgf("resolved %s from %s in %d ms", addrs[0].String(), host, d)
		return addrs[0].String(), nil
	}

	return "", fmt.Errorf("could not resolve %s using %s", host, clt)
}

func (d *Dns) clientFactory(enableDoh bool, useSystemDns bool) Resolver {
	if useSystemDns {
		return d.systemClient
	}

	if enableDoh {
		return d.dohClient
	}

	return d.generalClient
}

func parseIpAddr(addr string) (*net.IPAddr, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, fmt.Errorf("%s is not an ip address", addr)
	}

	ipAddr := &net.IPAddr{
		IP: ip,
	}

	return ipAddr, nil
}
