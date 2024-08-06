package dns

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/likexian/doh"
	dohDns "github.com/likexian/doh/dns"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/util"
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

func (d *DnsResolver) Lookup(domain string, useSystemDns bool) (string, error) {
	ipRegex := "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"

	if r, _ := regexp.MatchString(ipRegex, domain); r {
		return domain, nil
	}

	if useSystemDns {
    log.Debug("[DNS] ", domain, " resolving with system dns")
		return systemLookup(domain)
	}

	if d.enableDoh {
    log.Debug("[DNS] ", domain, " resolving with dns over https")
		return dohLookup(domain)
	}

  log.Debug("[DNS] ", domain, " resolving with custom dns")
  return customLookup(d.host, d.port, domain) 
}

func customLookup(host string, port string, domain string) (string, error) {

	dnsServer := host + ":" + port

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	c := new(dns.Client)

	response, _, err := c.Exchange(msg, dnsServer)
	if err != nil {
		return "", errors.New("couldn not resolve the domain(custom)")
	}

	for _, answer := range response.Answer {
		if record, ok := answer.(*dns.A); ok {
			return record.A.String(), nil
		}
	}

	return "", errors.New("no record found(custom)")

}

func systemLookup(domain string) (string, error) {
	systemResolver := net.Resolver{PreferGo: true}
	ips, err := systemResolver.LookupIPAddr(context.Background(), domain)
	if err != nil {
		return "", errors.New("couldn not resolve the domain(system)")
	}

	for _, ip := range ips {
		return ip.String(), nil
	}

	return "", errors.New("no record found(system)")
}

func dohLookup(domain string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c := doh.Use(doh.CloudflareProvider, doh.GoogleProvider)

	rsp, err := c.Query(ctx, dohDns.Domain(domain), dohDns.TypeA)
	if err != nil {
		return "", errors.New("could not resolve the domain(doh)")
	}
	// doh dns answer
	answer := rsp.Answer

	// print all answer
	for _, a := range answer {
		if a.Type != 1 { // Type == 1 -> A Record
			continue
		}

		return a.Data, nil
	}

	// close the client
	c.Close()

	return "", errors.New("no record found(doh)")
}
