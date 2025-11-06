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
	upstream string
	client   *http.Client
	qTypes   []uint16
	logger   zerolog.Logger
}

func NewHTTPSResolver(
	endpoint string,
	server net.IP,
	qTypes []uint16,
	logger zerolog.Logger,
) *HTTPSResolver {
	var upstream string
	if endpoint != "" {
		upstream = endpoint
	} else {
		upstream = "https://" + server.String() + "/dns-query"
	}

	return &HTTPSResolver{
		upstream: upstream,
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

func (dr *HTTPSResolver) String() string {
	return fmt.Sprintf("https-resolver(%s)", dr.upstream)
}

func (dr *HTTPSResolver) Resolve(
	ctx context.Context,
	host string,
) (RecordSet, error) {
	logger := dr.logger.With().Ctx(ctx).Logger()
	logger.Debug().Msgf("resolving %s", host)

	resCh := lookupAllTypes(ctx, host, dr.qTypes, dr.exchange)
	rSet, err := processMessages(ctx, resCh)
	return rSet, err
}

func (dr *HTTPSResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s?dns=%s",
		dr.upstream,
		base64.RawStdEncoding.EncodeToString(pack),
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
