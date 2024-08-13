package dns

import (
	"context"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/dns/addrselect"
	"github.com/xvzc/SpoofDPI/util"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

type DnsResolver struct {
	host      string
	port      string
	enableDoh bool
}

func NewResolver(config *util.Config) *DnsResolver {
	return &DnsResolver{
		host:      *config.DnsAddr,
		port:      strconv.Itoa(*config.DnsPort),
		enableDoh: *config.EnableDoh,
	}
}

type client interface {
	Resolve(ctx context.Context, host string) ([]net.IPAddr, error)
	String() string
}

func (d *DnsResolver) Lookup(domain string, useSystemDns bool) (string, error) {
	if _, err := parseAddr(domain); err == nil {
		return domain, nil
	}
	server := net.JoinHostPort(d.host, d.port)

	var resolver client
	if useSystemDns {
		resolver = NewSystemClient()
	} else if d.enableDoh {
		resolver = NewDoHClient(d.host)
	} else {
		resolver = NewCustomClient(server)
	}

	log.Debugf("[DNS] Resolving %s using %s", domain, resolver)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	addrs, err := resolver.Resolve(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("%s: %w", resolver, err)
	}
	return addrs[0].String(), nil
}

type SystemClient struct {
	client *net.Resolver
}

func NewSystemClient() *SystemClient {
	return &SystemClient{
		client: &net.Resolver{PreferGo: true},
	}
}

func (c *SystemClient) String() string {
	return "SystemClient"
}

func (c *SystemClient) Resolve(ctx context.Context, host string) ([]net.IPAddr, error) {
	addrs, err := c.client.LookupIPAddr(ctx, host)
	if err != nil {
		return []net.IPAddr{}, err
	}
	return addrs, nil
}

type sendMsgFunc = func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error)

type customDNSResult struct {
	msg *dns.Msg
	err error
}

type CustomClient struct {
	server    string
	sendMsgFn sendMsgFunc
}

func (c *CustomClient) Resolve(ctx context.Context, host string) ([]net.IPAddr, error) {
	queryTypes := []uint16{dns.TypeAAAA, dns.TypeA}
	resultCh := c.makeLookups(ctx, host, queryTypes)

	addrs, err := c.processResults(ctx, resultCh)
	return addrs, err
}

func (c *CustomClient) makeLookups(ctx context.Context, host string, queryTypes []uint16) <-chan *customDNSResult {
	var wg sync.WaitGroup
	resCh := make(chan *customDNSResult)

	lookup := func(qType uint16) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		case resCh <- c.makeLookup(ctx, host, qType):
		}
	}
	for _, queryType := range queryTypes {
		wg.Add(1)
		go lookup(queryType)
	}
	go func() {
		wg.Wait()
		close(resCh)
	}()
	return resCh
}

func (c *CustomClient) makeLookup(ctx context.Context, host string, queryType uint16) *customDNSResult {
	msg := c.newMsg(host, queryType)
	resp, err := c.sendMsg(ctx, msg)
	if err != nil {
		queryName := recordTypeIDToName(queryType)
		err = fmt.Errorf("resolving %s, query type %s: %w", host, queryName, err)
		return &customDNSResult{err: err}
	}
	return &customDNSResult{msg: resp}
}

func (c *CustomClient) newMsg(host string, qType uint16) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(host), qType)
	return msg
}

func (c *CustomClient) sendMsg(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	resp, err := c.sendMsgFn(ctx, msg)
	return resp, err
}

func (c *CustomClient) processResults(ctx context.Context, resCh <-chan *customDNSResult) ([]net.IPAddr, error) {
	var errs []error
	var addrs []net.IPAddr

	for result := range resCh {
		if result.err != nil {
			errs = append(errs, result.err)
			continue
		}
		resultAddrs := parseAddrsFromMsg(result.msg)
		addrs = append(addrs, resultAddrs...)
	}
	select {
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	default:
		if len(addrs) == 0 {
			return addrs, errors.Join(errs...)
		}
	}
	sortAddrs(addrs)
	return addrs, nil
}

func (c *CustomClient) String() string {
	return fmt.Sprintf("CustomClient for %s", c.server)
}

func NewCustomClient(server string) *CustomClient {
	clt := &dns.Client{}
	return &CustomClient{
		server: server,
		sendMsgFn: func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			resp, _, err := clt.Exchange(msg, server)
			return resp, err
		},
	}
}

func NewDoHClient(host string) *CustomClient {
	server := net.JoinHostPort(host, "443")
	clt := getDOHClient(server)
	return &CustomClient{
		server: server,
		sendMsgFn: func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
			return clt.dohExchange(ctx, msg)
		},
	}
}

func recordTypeIDToName(id uint16) string {
	switch id {
	case 1:
		return "A"
	case 28:
		return "AAAA"
	}
	return strconv.FormatUint(uint64(id), 10)
}

func parseAddrsFromMsg(msg *dns.Msg) []net.IPAddr {
	var addrs []net.IPAddr

	for _, record := range msg.Answer {
		switch ipRecord := record.(type) {
		case *dns.A:
			addrs = append(addrs, net.IPAddr{IP: ipRecord.A})
		case *dns.AAAA:
			addrs = append(addrs, net.IPAddr{IP: ipRecord.AAAA})
		}
	}
	return addrs
}

func parseAddr(addr string) (net.IP, error) {
	parsed, err := netip.ParseAddr(addr)
	if err != nil {
		return net.IP{}, fmt.Errorf("parsing %s as an IP address: %w", addr, err)
	}
	return parsed.AsSlice(), nil
}

func sortAddrs(addrs []net.IPAddr) {
	addrselect.SortByRFC6724(addrs)
}
