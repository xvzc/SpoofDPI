package packet

import (
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

type HttpPacket struct {
	raw     []byte
	method  string
	domain  string
	path    string
	version string
}

func NewHttpPacket(raw []byte) HttpPacket {
	method, domain, path, version := parse(raw)

	return HttpPacket{
		raw:     raw,
		method:  method,
		domain:  domain,
		path:    path,
		version: version,
	}
}

func (p *HttpPacket) Raw() []byte {
	return p.raw
}
func (p *HttpPacket) Method() string {
	return p.method
}

func (p *HttpPacket) Domain() string {
	return p.domain
}

func (p *HttpPacket) Version() string {
	return p.version
}

func (p *HttpPacket) IsValidMethod() bool {
	if _, exists := validMethod[p.Method()]; exists {
		return true
	}

	return false
}

func (p *HttpPacket) IsConnectMethod() bool {
	return p.Method() == "CONNECT"
}

func (p *HttpPacket) Tidy() {
	s := string(p.raw)

	lines := strings.Split(s, "\n")

	lines[0] = p.method + " " + p.path + " " + p.version

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "Proxy-Connection") {
			lines[i] = ""
		}
	}

	result := ""
	for i := 0; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}

		result += lines[i] + "\n"
	}

	p.raw = []byte(result)
}

func parse(raw []byte) (string, string, string, string) {
	var firstLine string
	for i := 0; i < len(raw); i++ {
		if (raw)[i] == '\n' {
			firstLine = string((raw)[:i])
			break
		}
	}

	tokens := strings.Split(firstLine, " ")

	method := strings.TrimSpace(tokens[0])
	url := strings.TrimSpace(tokens[1])
	version := strings.TrimSpace(tokens[2])

	url = strings.Replace(url, "http://", "", 1)
	url = strings.Replace(url, "https://", "", 1)

	domain := url
	for i := 0; i < len(url); i++ {
		if url[i] == ':' {
			domain = url[:i]
			break
		}

		if url[i] == '/' {
			domain = url[:i]
			break
		}
	}

	path := "/"
	for i := 0; i < len(url); i++ {
		if url[i] == '/' {
			path = url[i:]
			break
		}
	}

	return method, domain, path, version
}
