package util

import (
	"log"
	"strings"

	"github.com/xvzc/SpoofDPI/config"
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

func IsValidMethod(name string) bool {
	if _, exists := validMethod[name]; exists {
		return true
	}

	return false
}

func ExtractDomain(message *[]byte) string {
	i := 0
	for ; i < len(*message); i++ {
		if (*message)[i] == ' ' {
			i++
			break
		}
	}

	j := i
	for ; j < len(*message); j++ {
		if (*message)[j] == ' ' {
			break
		}
	}

	domain := string((*message)[i:j])
	domain = strings.Replace(domain, "http://", "", 1)
	domain = strings.Replace(domain, "https://", "", 1)
	domain = strings.Split(domain, ":")[0]
	domain = strings.Split(domain, "/")[0]

	return strings.TrimSpace(domain)
}

func ExtractMethod(message *[]byte) string {
	i := 0
	for ; i < len(*message); i++ {
		if (*message)[i] == ' ' {
			break
		}
	}

	method := strings.TrimSpace(string((*message)[:i]))
	Debug(method)

	return strings.ToUpper(method)
}

func Debug(v ...interface{}) {
	if config.GetConfig().Debug == false {
		return
	}

	log.Println(v...)
}

func BytesToChunks(buf []byte) [][]byte {
	if len(buf) < 1 {
		return [][]byte{buf}
	}

	return [][]byte{buf[:1], buf[1:]}
}
