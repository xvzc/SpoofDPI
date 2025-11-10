package dns

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var _ Resolver = (*HTTPSResolver)(nil)

type HTTPSResolver struct {
	endpoint string
	client   *http.Client
	qTypes   []uint16
	logger   zerolog.Logger
}

func (hr *HTTPSResolver) Upstream() string {
	return hr.endpoint
}

func NewHTTPSResolver(
	endpoint string,
	qTypes []uint16,
	logger zerolog.Logger,
) *HTTPSResolver {
	return &HTTPSResolver{
		endpoint: endpoint,
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
		qTypes: qTypes,
		logger: logger,
	}
}

func (hr *HTTPSResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name:   "https",
			Dest:   hr.endpoint,
			Cached: false,
		},
	}
}

func (hr *HTTPSResolver) String() string {
	return fmt.Sprintf("https-resolver(%s)", hr.endpoint)
}

func (hr *HTTPSResolver) Resolve(
	ctx context.Context,
	host string,
) (RecordSet, error) {
	logger := hr.logger.With().Ctx(ctx).Logger()
	logger.Debug().Msgf("resolving %s", host)

	resCh := lookupAllTypes(ctx, host, hr.qTypes, hr.exchange)
	rSet, err := processMessages(ctx, resCh)
	return rSet, err
}

func (hr *HTTPSResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s?dns=%s",
		hr.endpoint,
		base64.RawStdEncoding.EncodeToString(pack),
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

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("doh status error")
	}

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	resultMsg := new(dns.Msg)
	err = resultMsg.Unpack(buf.Bytes())
	if err != nil {
		return nil, err
	}

	if resultMsg.Rcode != dns.RcodeSuccess {
		return nil, errors.New("doh rcode wasn't successful")
	}

	return resultMsg, nil
}
