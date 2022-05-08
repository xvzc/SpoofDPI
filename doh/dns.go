package doh

import (
	"errors"
	"sync"

    "regexp"
	"github.com/babolivier/go-doh-client"
)

var resolver *doh.Resolver
var once sync.Once

func Init(dns string) {
	getInstance().Host = dns
}

func Lookup(domain string) (string, error) {
    ipRegex := "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
    
    if r, _ := regexp.MatchString(ipRegex, domain); r {
        return domain, nil
    }


	a, _, err := resolver.LookupA(domain)
	if err != nil {
		return "", err
	}

    if len(a) < 1 {
        return "", errors.New(" couldn't resolve the domain")
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
