package request

import (
	"strings"
)

type Request struct {
	Raw    *[]byte
	Method string
	Domain string
}

func (r *Request) IsValidMethod() bool {
	if _, exists := getValidMethods()[r.Method]; exists {
		return true
	}

	return false
}

func New(raw *[]byte) Request {
	return Request{
		Raw:    raw,
		Method: extractMethod(raw),
		Domain: extractDomain(raw),
	}
}

func (r *Request) ToChunks() {

}

func extractDomain(request *[]byte) string {
	i := 0
	for ; i < len(*request); i++ {
		if (*request)[i] == ' ' {
			i++
			break
		}
	}

	j := i
	for ; j < len(*request); j++ {
		if (*request)[j] == ' ' {
			break
		}
	}

	domain := string((*request)[i:j])
	domain = strings.Replace(domain, "http://", "", 1)
	domain = strings.Replace(domain, "https://", "", 1)
	domain = strings.Split(domain, ":")[0]
	domain = strings.Split(domain, "/")[0]

	return strings.TrimSpace(domain)
}

func extractMethod(message *[]byte) string {
	i := 0
	for ; i < len(*message); i++ {
		if (*message)[i] == ' ' {
			break
		}
	}

	method := strings.TrimSpace(string((*message)[:i]))

	return strings.ToUpper(method)
}

func getValidMethods() map[string]struct{} {
	return map[string]struct{}{
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
}
