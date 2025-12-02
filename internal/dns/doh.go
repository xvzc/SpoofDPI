package dns

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

var _ Resolver = (*DOHResolver)(nil)

type DOHResolver struct {
	logger zerolog.Logger

	client   *http.Client
	upstream string
}

func NewDOHResolver(
	logger zerolog.Logger,
	upstream string,
) *DOHResolver {
	var dohUpstream string
	if strings.HasPrefix(upstream, "https://") {
		dohUpstream = upstream
	} else {
		dohUpstream = "https://" + upstream + "/dns-query"
	}

	return &DOHResolver{
		logger: logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 5 * time.Second,
				MaxIdleConnsPerHost: 100,
				MaxIdleConns:        100,
			},
		},
		upstream: dohUpstream,
	}
}

func (hr *DOHResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "doh",
			Dst:    hr.upstream,
			Cached: CachedStatus{false},
		},
	}
}

func (hr *DOHResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (*RecordSet, error) {
	resCh := lookupAllTypes(ctx, domain, qTypes, hr.exchange)
	return processMessages(ctx, resCh)
}

func (hr *DOHResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	logger := logging.WithLocalScope(hr.logger, ctx, "dns-over-https")

	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s?dns=%s",
		hr.upstream,
		base64.RawURLEncoding.EncodeToString(pack),
		// base64.RawStdEncoding.EncodeToString(pack),
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/dns-message")

	resp, err := hr.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	buf := bytes.Buffer{}
	bodyLen, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Trace().
			Int64("len", bodyLen).
			Int("status", resp.StatusCode).
			Str("value", buf.String()).
			Msg("status not ok")
		return nil, fmt.Errorf("doh status code(%d)", resp.StatusCode)
	}

	resultMsg := new(dns.Msg)
	err = resultMsg.Unpack(buf.Bytes())
	if err != nil {
		return nil, err
	}

	if resultMsg.Rcode != dns.RcodeSuccess {
		logger.Trace().
			Int("rcode", resultMsg.Rcode).
			Str("msg", resultMsg.String()).
			Msg("rcode not ok")
		return nil, fmt.Errorf("doh Rcode(%d)", resultMsg.Rcode)
	}

	return resultMsg, nil
}
