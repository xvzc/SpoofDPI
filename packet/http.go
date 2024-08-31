package packet

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
)

var validMethod = map[string]struct{}{
	"DELETE":      {},
	"GET":         {},
	"HEAD":        {},
	"POST":        {},
	"PUT":         {},
	"CONNECT":     {},
	"OPTIONS":     {},
	"TRACE":       {},
	"COPY":        {},
	"LOCK":        {},
	"MKCOL":       {},
	"MOVE":        {},
	"PROPFIND":    {},
	"PROPPATCH":   {},
	"SEARCH":      {},
	"UNLOCK":      {},
	"BIND":        {},
	"REBIND":      {},
	"UNBIND":      {},
	"ACL":         {},
	"REPORT":      {},
	"MKACTIVITY":  {},
	"CHECKOUT":    {},
	"MERGE":       {},
	"M-SEARCH":    {},
	"NOTIFY":      {},
	"SUBSCRIBE":   {},
	"UNSUBSCRIBE": {},
	"PATCH":       {},
	"PURGE":       {},
	"MKCALENDAR":  {},
	"LINK":        {},
	"UNLINK":      {},
}

type HttpRequest struct {
	raw     []byte
	method  string
	domain  string
	port    string
	path    string
	version string
}

func ReadHttpRequest(rdr io.Reader) (*HttpRequest, error) {
	p, err := parse(rdr)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *HttpRequest) Raw() []byte {
	return p.raw
}
func (p *HttpRequest) Method() string {
	return p.method
}

func (p *HttpRequest) Domain() string {
	return p.domain
}

func (p *HttpRequest) Port() string {
	return p.port
}

func (p *HttpRequest) Version() string {
	return p.version
}

func (p *HttpRequest) IsValidMethod() bool {
	if _, exists := validMethod[p.Method()]; exists {
		return true
	}

	return false
}

func (p *HttpRequest) IsConnectMethod() bool {
	return p.Method() == "CONNECT"
}

func (p *HttpRequest) Tidy() {
	s := string(p.raw)

	parts := strings.Split(s, "\r\n\r\n")
	meta := strings.Split(parts[0], "\r\n")

	meta[0] = p.method + " " + p.path + " " + p.version

	var buf bytes.Buffer
	buf.Grow(len(p.raw))

	crLF := []byte{0xD, 0xA}
	for _, m := range meta {
		if strings.HasPrefix(m, "Proxy-Connection") {
			continue
		}
		buf.WriteString(m)
		buf.Write(crLF)
	}
	buf.Write(crLF)
	buf.WriteString(parts[1])

	p.raw = buf.Bytes()
}

func parse(rdr io.Reader) (*HttpRequest, error) {
	sb := strings.Builder{}
	tee := io.TeeReader(rdr, &sb)
	request, err := http.ReadRequest(bufio.NewReader(tee))
	if err != nil {
		return nil, err
	}

	p := &HttpRequest{}
	p.raw = []byte(sb.String())

	p.domain, p.port, err = net.SplitHostPort(request.Host)
	if err != nil {
		p.domain = request.Host
		p.port = ""
	}

	p.method = request.Method
	p.version = request.Proto
	p.path = request.URL.Path

	if request.URL.RawQuery != "" {
		p.path += "?" + request.URL.RawQuery
	}

	if request.URL.RawFragment != "" {
		p.path += "#" + request.URL.RawFragment
	}
	if p.path == "" {
		p.path = "/"
	}

	request.Body.Close()
	return p, nil
}
