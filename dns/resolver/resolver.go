package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/miekg/dns"
	"github.com/xvzc/SpoofDPI/dns/addrselect"
)

type exchangeFunc = func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error)

type DNSResult struct {
	msg *dns.Msg
	err error
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

func sortAddrs(addrs []net.IPAddr) {
	addrselect.SortByRFC6724(addrs)
}

func lookupAllTypes(ctx context.Context, host string, qTypes []uint16, exchange exchangeFunc) <-chan *DNSResult {
	var wg sync.WaitGroup
	resCh := make(chan *DNSResult)

	for _, qType := range qTypes {
		wg.Add(1)
		go func(qType uint16) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case resCh <- lookupType(ctx, host, qType, exchange):
			}
		}(qType)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	return resCh
}

func lookupType(ctx context.Context, host string, queryType uint16, exchange exchangeFunc) *DNSResult {
	msg := newMsg(host, queryType)
	resp, err := exchange(ctx, msg)
	if err != nil {
		queryName := recordTypeIDToName(queryType)
		err = fmt.Errorf("resolving %s, query type %s: %w", host, queryName, err)
		return &DNSResult{err: err}
	}
	return &DNSResult{msg: resp}
}

func newMsg(host string, qType uint16) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(host), qType)
	return msg
}

func processResults(ctx context.Context, resCh <-chan *DNSResult) ([]net.IPAddr, error) {
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
		return nil, errors.New("canceled")
	default:
		if len(addrs) == 0 {
			return addrs, errors.Join(errs...)
		}
	}

	sortAddrs(addrs)
	return addrs, nil
}
