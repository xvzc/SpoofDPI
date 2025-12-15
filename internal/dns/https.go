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
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/logging"
)

var _ Resolver = (*HTTPSResolver)(nil)

type HTTPSResolver struct {
	logger zerolog.Logger

	client       *http.Client
	defaultAttrs *config.DNSOptions
}

func NewHTTPSResolver(
	logger zerolog.Logger,
	defaultAttrs *config.DNSOptions,
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
		defaultAttrs: defaultAttrs,
	}
}

func (dr *HTTPSResolver) Info() []ResolverInfo {
	return []ResolverInfo{
		{
			Name: "https",
			Dst:  *dr.defaultAttrs.HTTPSURL,
		},
	}
}

func (dr *HTTPSResolver) Resolve(
	ctx context.Context,
	domain string,
	fallback Resolver,
	rule *config.Rule,
) (*RecordSet, error) {
	attrs := dr.defaultAttrs
	if rule != nil {
		attrs = attrs.Merge(rule.DNS)
	}

	upstream := *attrs.HTTPSURL
	if !strings.HasPrefix(upstream, "https://") {
		upstream = "https://" + upstream + "/dns-query"
	}

	resCh := lookupAllTypes(
		ctx,
		domain,
		upstream,
		parseQueryTypes(*attrs.QType),
		dr.exchange,
	)
	return processMessages(ctx, resCh)
}

func (dr *HTTPSResolver) exchange(
	ctx context.Context,
	msg *dns.Msg,
	upstream string,
) (*dns.Msg, error) {
	logger := logging.WithLocalScope(ctx, dr.logger, "doh_exchange")

	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s?dns=%s",
		upstream,
		base64.RawURLEncoding.EncodeToString(pack),
		// base64.RawStdEncoding.EncodeToString(pack),
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/dns-message")

	resp, err := dr.client.Do(req)
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
			Str("body", buf.String()).
			Msg("status not ok")
		return nil, fmt.Errorf("doh status code(%d)", resp.StatusCode)
	}

	resultMsg := new(dns.Msg)
	err = resultMsg.Unpack(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Ignore Rcode 3 (NameNotFound) as it's not a critical error.
	if resultMsg.Rcode != dns.RcodeSuccess && resultMsg.Rcode != dns.RcodeNameError {
		logger.Trace().
			Int("rcode", resultMsg.Rcode).
			Str("msg", resultMsg.String()).
			Msg("rcode not ok")
		return nil, fmt.Errorf("doh Rcode(%d)", resultMsg.Rcode)
	}

	return resultMsg, nil
}
