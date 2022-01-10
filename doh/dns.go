package doh

import (
	"sync"

	"github.com/babolivier/go-doh-client"
)

var resolver *doh.Resolver
var once sync.Once

func Init(dns string) {
	getInstance().Host = dns
}

func Lookup(domain string) (string, error) {
	a, _, err := resolver.LookupA(domain)
	if err != nil {
		return "", err
	}

	ip := a[0].IP4

	return ip, nil
}

func getInstance() *doh.Resolver {
	once.Do(func() {
		resolver = &doh.Resolver{
			Host:  "",
			Class: doh.IN,
		}
	})

	return resolver
}
