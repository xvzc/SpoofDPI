package dns

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

var _ Resolver = (*HTTPSResolver)(nil)

type HTTPSResolver struct {
	logger zerolog.Logger

	client *http.Client

	enabled  bool
	endpoint string
}

func (hr *HTTPSResolver) Upstream() string {
	return hr.endpoint
}

func NewHTTPSResolver(
	logger zerolog.Logger,
	enabled bool,
	endpoint string,
) *HTTPSResolver {
	return &HTTPSResolver{
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
		enabled:  enabled,
		endpoint: endpoint,
	}
}

func (hr *HTTPSResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "https",
			Dest:   hr.endpoint,
			Cached: CachedStatus{false},
		},
	}
}

func (hr *HTTPSResolver) Route(ctx context.Context) Resolver {
	if include, ok := appctx.DomainIncludedFrom(ctx); ok && include && hr.enabled {
		return hr
	}

	return nil
}

func (hr *HTTPSResolver) Resolve(
	ctx context.Context,
	domain string,
	qTypes []uint16,
) (RecordSet, error) {
	resCh := lookupAllTypes(ctx, domain, qTypes, hr.exchange)
	rSet, err := processMessages(ctx, resCh)
	return rSet, err
}

func (hr *HTTPSResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	logger := hr.logger.With().Ctx(ctx).Logger()

	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s?dns=%s",
		hr.endpoint,
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
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Trace().
			Msgf("dns error; resolver=https; body=%s", buf.String())
		return nil, fmt.Errorf("doh status code(%d)", resp.StatusCode)
	}

	resultMsg := new(dns.Msg)
	err = resultMsg.Unpack(buf.Bytes())
	if err != nil {
		return nil, err
	}

	if resultMsg.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("doh Rcode(%d)", resultMsg.Rcode)
	}

	return resultMsg, nil
}
