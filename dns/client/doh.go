package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/miekg/dns"
)

type DOHClient struct {
	upstream   string
	httpClient *http.Client
}

func NewDOHClient(host string) *DOHClient {
	h := &http.Client{
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
	}

	host = regexp.MustCompile(`^https:\/\/|\/dns-query$`).ReplaceAllString(host, "")
	return &DOHClient{
		upstream:   "https://" + host + "/dns-query",
		httpClient: h,
	}
}

func (c *DOHClient) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	sendMsg := func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
		clt := NewDOHClient(net.JoinHostPort(host, "443"))
		return clt.dohExchange(ctx, msg)
	}

	resultCh := lookup(ctx, host, qTypes, sendMsg)
	addrs, err := processResults(ctx, resultCh)
	return addrs, err
}

func (c *DOHClient) String() string {
	return fmt.Sprintf("doh client(%s)", c.upstream)
}

func (d *DOHClient) dohQuery(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s?dns=%s", d.upstream, base64.RawStdEncoding.EncodeToString(pack))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/dns-message")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	return resultMsg, nil
}

func (d *DOHClient) dohExchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	res, err := d.dohQuery(ctx, msg)
	if err != nil {
		return nil, err
	}

	if res.Rcode != dns.RcodeSuccess {
		return nil, errors.New("doh rcode wasn't successful")
	}

	return res, nil
}
