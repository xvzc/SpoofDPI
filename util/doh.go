package util

import (
	"github.com/babolivier/go-doh-client"
)

func DnsLookupOverHttps(dns string, domain string) (string, error) {
	// Perform a A lookup on example.com
	resolver := doh.Resolver{
		Host:  dns, // Change this with your favourite DoH-compliant resolver.
		Class: doh.IN,
	}

	Debug(domain)
	a, _, err := resolver.LookupA(domain)
	if err != nil {
		return "", err
	}

	ip := a[0].IP4

	return ip, nil
}
