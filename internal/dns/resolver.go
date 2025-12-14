package dns

import (
	"context"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"

	"github.com/miekg/dns"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/dns/addrselect"
)

type ResolverKind int

const (
	UDP ResolverKind = iota
	HTTPS
	System
)

type Resolver interface {
	Info() []ResolverInfo
	Resolve(
		ctx context.Context,
		domain string,
		falback Resolver,
		rule *config.Rule,
	) (*RecordSet, error)
}

type ResolverInfo struct {
	Name string `json:"name"`
	Dst  string `json:"dst"`
}

func (i *ResolverInfo) String() string {
	return fmt.Sprintf("name=%s; dst=%s;", i.Name, i.Dst)
}

type exchangeFunc = func(
	ctx context.Context,
	msg *dns.Msg,
	upstream string,
) (*dns.Msg, error)

type MsgChan struct {
	msg *dns.Msg
	err error
}

type RecordSet struct {
	Addrs []net.IPAddr
	TTL   uint32
}

func (rs *RecordSet) Clone() *RecordSet {
	return &RecordSet{
		Addrs: append([]net.IPAddr(nil), rs.Addrs...),
		TTL:   rs.TTL,
	}
}

func parseQueryTypes(qtype config.DNSQueryType) []uint16 {
	switch qtype {
	case config.DNSQueryIPv4:
		return []uint16{dns.TypeA}
	case config.DNSQueryIPv6:
		return []uint16{dns.TypeAAAA}
	case config.DNSQueryAll:
		return []uint16{dns.TypeA, dns.TypeAAAA}
	default:
		return []uint16{dns.TypeA}
	}
}

func newMsg(domain string, qType uint16) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), qType)

	return msg
}

func recordTypeIDToName(id uint16) string {
	switch id {
	case dns.TypeA:
		return "A"
	case dns.TypeAAAA:
		return "AAAA"
	}

	return strconv.FormatUint(uint64(id), 10)
}

func lookupType(
	ctx context.Context,
	domain string,
	upstream string,
	queryType uint16,
	exchange exchangeFunc,
) *MsgChan {
	resMsg, err := exchange(ctx, newMsg(domain, queryType), upstream)
	if err != nil {
		queryName := recordTypeIDToName(queryType)
		err = fmt.Errorf(
			"failed to resolve '%s', query type=%s: %w",
			domain,
			queryName,
			err,
		)

		return &MsgChan{msg: nil, err: err}
	}

	return &MsgChan{msg: resMsg, err: nil}
}

func lookupAllTypes(
	ctx context.Context,
	domain string,
	upstream string,
	qTypes []uint16,
	exchange exchangeFunc,
) <-chan *MsgChan {
	var wg sync.WaitGroup
	resCh := make(chan *MsgChan)

	for _, qType := range qTypes {
		wg.Add(1)

		go func(qType uint16) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case resCh <- lookupType(ctx, domain, upstream, qType, exchange):
			}
		}(qType)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	return resCh
}

func parseMsg(msg *dns.Msg) ([]net.IPAddr, uint32, bool) {
	var addrs []net.IPAddr
	minTTL := uint32(math.MaxUint32)
	ok := false

	for _, record := range msg.Answer {
		switch ipRecord := record.(type) {
		case *dns.A:
			ok = true
			addrs = append(addrs, net.IPAddr{IP: ipRecord.A})
			minTTL = min(minTTL, record.Header().Ttl)
		case *dns.AAAA:
			ok = true
			addrs = append(addrs, net.IPAddr{IP: ipRecord.AAAA})
			minTTL = min(minTTL, record.Header().Ttl)
		}
	}

	return addrs, minTTL, ok
}

func processMessages(
	ctx context.Context,
	resCh <-chan *MsgChan,
) (*RecordSet, error) {
	var errs []error
	var addrs []net.IPAddr

	minTTL := uint32(math.MaxUint32)
	found := false

loop: // Loop until the channel is closed or context is canceled
	for {
		select {
		// Detect context cancellation immediately to prevent blocking
		case <-ctx.Done():
			return nil, ctx.Err()

		case result, ok := <-resCh:
			// If the channel is closed, break the loop
			if !ok {
				break loop
			}

			if result.err != nil {
				errs = append(errs, result.err)
				continue
			}

			// Defensive check for nil msg
			if result.msg == nil {
				continue
			}

			resultAddrs, ttl, ok := parseMsg(result.msg)
			if ok {
				addrs = append(addrs, resultAddrs...)
				minTTL = min(minTTL, ttl)
				found = true
			}
		}
	}

	// If we found any valid addresses,
	// return them even if some errors occurred (Partial Success)
	if len(addrs) > 0 {
		if !found {
			minTTL = 0
		}
		addrselect.SortByRFC6724(addrs)
		return &RecordSet{Addrs: addrs, TTL: minTTL}, nil
	}

	// Only return errors if no addresses were found at all
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to resolve with %d errors", len(errs))
	}

	return nil, fmt.Errorf("record not found")
}
