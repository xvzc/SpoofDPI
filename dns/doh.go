package dns

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type DOHClient struct {
	upstream string
	client   *http.Client
}

var client *DOHClient
var clientOnce sync.Once

func GetDOHClient(upstream string) *DOHClient {
	clientOnce.Do(func() {
		if client == nil {
			if !strings.HasPrefix(upstream, "https://") {
				upstream = "https://" + upstream
			}

			if !strings.HasSuffix(upstream, "/dns-query") {
				upstream = upstream + "/dns-query"
			}

			c := &http.Client{
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

			client = &DOHClient{
				upstream: upstream,
				client:   c,
			}
		}
	})

	return client
}

func (d *DOHClient) query(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
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

	resp, err := d.client.Do(req)
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

func (d *DOHClient) Exchange(ctx context.Context, domain string, msg *dns.Msg) (*dns.Msg, error) {
	res, err := d.query(ctx, msg)
	if err != nil {
		return nil, err
	}

	if res.Rcode != dns.RcodeSuccess {
		return nil, errors.New("doh rcode wasn't successful")
	}

	return res, nil
}
