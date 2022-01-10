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
	Raw     []byte
	Method  string
	Domain  string
	Version string
}

func NewHttpPacket(raw []byte) HttpPacket {
	method, domain, version := parse(raw)
	return HttpPacket{
		Raw:     raw,
		Method:  method,
		Domain:  domain,
		Version: version,
	}
}

func (r *HttpPacket) IsValidMethod() bool {
	if _, exists := validMethod[r.Method]; exists {
		return true
	}

	return false
}

func (r *HttpPacket) IsConnectMethod() bool {
	return r.Method == "CONNECT"
}

func parse(raw []byte) (string, string, string) {
	var firstLine string
	for i := 0; i < len(raw); i++ {
		if (raw)[i] == '\n' {
			firstLine = string((raw)[:i])
		}
	}

	tokens := strings.Split(firstLine, " ")

	method := strings.TrimSpace(tokens[0])
	domain := strings.TrimSpace(tokens[1])
	version := strings.TrimSpace(tokens[2])

	domain = strings.Replace(domain, "http://", "", 1)
	domain = strings.Replace(domain, "https://", "", 1)
	domain = strings.Split(domain, ":")[0]
	domain = strings.Split(domain, "/")[0]

	return method, domain, version
}
