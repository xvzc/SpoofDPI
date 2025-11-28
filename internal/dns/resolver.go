package dns

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"

	"github.com/miekg/dns"
	"github.com/xvzc/SpoofDPI/internal/dns/addrselect"
)

type Resolver interface {
	Info() []ResolverInfo
	Resolve(ctx context.Context, domain string, qTypes []uint16) (RecordSet, error)
}

type ResolverInfo struct {
	Name   string       `json:"name"`
	Dst    string       `json:"dst"`
	Cached CachedStatus `json:"cached"`
}

func (i *ResolverInfo) String() string {
	return fmt.Sprintf("name=%s; cached=%s; dst=%s;", i.Name, i.Cached.String(), i.Dst)
}

type CachedStatus struct {
	bool
}

func (s *CachedStatus) String() string {
	if s.bool {
		return "1"
	} else {
		return "0"
	}
}

type exchangeFunc = func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error)

type MsgEnvelope struct {
	msg *dns.Msg
	err error
}

type RecordSet struct {
	addrs []net.IPAddr
	ttl   uint32
}

func (rs *RecordSet) CopyAddrs() []net.IPAddr {
	return rs.addrs
}

func (rs *RecordSet) TTL() uint32 {
	return rs.ttl
}

func (rs *RecordSet) Count() int {
	return len(rs.addrs)
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
	queryType uint16,
	exchange exchangeFunc,
) *MsgEnvelope {
	resMsg, err := exchange(ctx, newMsg(domain, queryType))
	if err != nil {
		queryName := recordTypeIDToName(queryType)
		err = fmt.Errorf(
			"failed to resolve '%s', query type=%s: %w",
			domain,
			queryName,
			err,
		)

		return &MsgEnvelope{msg: nil, err: err}
	}

	return &MsgEnvelope{msg: resMsg, err: nil}
}

func lookupAllTypes(
	ctx context.Context,
	domain string,
	qTypes []uint16,
	exchange exchangeFunc,
) <-chan *MsgEnvelope {
	var wg sync.WaitGroup
	resCh := make(chan *MsgEnvelope)

	for _, qType := range qTypes {
		wg.Add(1)

		go func(qType uint16) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case resCh <- lookupType(ctx, domain, qType, exchange):
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
	resCh <-chan *MsgEnvelope,
) (RecordSet, error) {
	var errs []error
	var addrs []net.IPAddr

	minTTL := uint32(math.MaxUint32)
	found := false

	for result := range resCh {
		if result.err != nil {
			errs = append(errs, result.err)

			continue
		}

		resultAddrs, ttl, ok := parseMsg(result.msg)
		if ok {
			addrs = append(addrs, resultAddrs...)
			minTTL = min(minTTL, ttl)
			found = true
		}
	}

	select {
	case <-ctx.Done():
		return RecordSet{}, fmt.Errorf("context is canceled")
	default:
		if len(addrs) == 0 {
			return RecordSet{}, errors.Join(errs...)
		}
	}

	if !found {
		minTTL = 0
	}

	addrselect.SortByRFC6724(addrs)

	return RecordSet{addrs: addrs, ttl: minTTL}, nil
}
