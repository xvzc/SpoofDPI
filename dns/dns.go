package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/dns/client"
	"github.com/xvzc/SpoofDPI/util"
)

type Dns struct {
	host          string
	port          string
	systemClient  client.Client
	generalClient client.Client
	dohClient     client.Client
}

func NewResolver(config *util.Config) *Dns {
	addr := *config.DnsAddr
	port := strconv.Itoa(*config.DnsPort)

	return &Dns{
		host:          *config.DnsAddr,
		port:          port,
		systemClient:  client.NewSystemClient(),
		generalClient: client.NewGeneralClient(net.JoinHostPort(addr, port)),
		dohClient:     client.NewDOHClient(addr),
	}
}

func (d *Dns) ResolveHost(host string, enableDoh bool, useSystemDns bool) (string, error) {
	if ip, err := parseIpAddr(host); err == nil {
		return ip.String(), nil
	}

	clt := d.clientFactory(enableDoh, useSystemDns)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	log.Debugf("[DNS] resolving %s using %s", host, clt)
	t := time.Now()

	addrs, err := clt.Resolve(ctx, host, []uint16{dns.TypeAAAA, dns.TypeA})
	// addrs, err := clt.Resolve(ctx, host, []uint16{dns.TypeAAAA})
	if err != nil {
		return "", fmt.Errorf("%s: %w", clt, err)
	}

	if len(addrs) > 0 {
		d := time.Since(t).Milliseconds()
		log.Debugf("[DNS] resolved %s from %s in %d ms", addrs[0].String(), host, d)
		return addrs[0].String(), nil
	}

	return "", fmt.Errorf("could not resolve %s using %s", host, clt)
}

func (d *Dns) clientFactory(enableDoh bool, useSystemDns bool) client.Client {
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
