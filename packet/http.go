package packet

import (
	"errors"
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
    port    string
	path    string
	version string
}

func ParseUrl(raw []byte) {

}

func NewHttpPacket(raw []byte) (*HttpPacket, error){
    pkt := &HttpPacket{raw: raw}

    pkt.parse()

	return pkt, nil
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

func (p *HttpPacket) Port() string {
	return p.port
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

	lines := strings.Split(s, "\r\n")

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

		result += lines[i] + "\r\n"
	}

    result += "\r\n"

	p.raw = []byte(result)
}

func (p *HttpPacket )parse() error {
	var firstLine string
	for i := 0; i < len(p.raw); i++ {
		if (p.raw)[i] == '\r' {
			firstLine = string(p.raw[:i])
			break
		}
	}

	tokens := strings.Split(firstLine, " ")

    if (len(tokens) < 3) {
        return errors.New("Error parsing http request")
    }

	p.method = tokens[0]
	url := tokens[1]
	p.version = tokens[2]

    if strings.HasPrefix(url, "http://") {
        url = strings.Replace(url, "http://", "", 1)
    }

    if strings.HasPrefix(url, "https://") {
	    url = strings.Replace(url, "https://", "", 1)
    }

    domain := ""
    port := ""
	for i := 0; i < len(url); i++ {
		if url[i] == ':' {
			domain = url[:i]
            port = url[i:]
			break
		}

		if url[i] == '/' {
			domain = url[:i]
			break
		}
	}
    p.domain = domain
    p.port = port

	path := "/"
	for i := 0; i < len(url); i++ {
		if url[i] == '/' {
			path = url[i:]
			break
		}
	}

    p.path = path

    return nil
}
