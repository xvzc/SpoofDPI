package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
)

// validMethods contains the set of HTTP methods that are considered valid
var validMethods = map[string]bool{
	"DELETE":      true,
	"GET":         true,
	"HEAD":        true,
	"POST":        true,
	"PUT":         true,
	"CONNECT":     true,
	"OPTIONS":     true,
	"TRACE":       true,
	"COPY":        true,
	"LOCK":        true,
	"MKCOL":       true,
	"MOVE":        true,
	"PROPFIND":    true,
	"PROPPATCH":   true,
	"SEARCH":      true,
	"UNLOCK":      true,
	"BIND":        true,
	"REBIND":      true,
	"UNBIND":      true,
	"ACL":         true,
	"REPORT":      true,
	"MKACTIVITY":  true,
	"CHECKOUT":    true,
	"MERGE":       true,
	"M-SEARCH":    true,
	"NOTIFY":      true,
	"SUBSCRIBE":   true,
	"UNSUBSCRIBE": true,
	"PATCH":       true,
	"PURGE":       true,
	"MKCALENDAR":  true,
	"LINK":        true,
	"UNLINK":      true,
}

// HttpRequest wraps the standard http.Request with additional functionality
type HttpRequest struct {
	*http.Request
}

// NewHttpRequest creates a new HttpRequest from an http.Request
func NewHttpRequest(req *http.Request) *HttpRequest {
	return &HttpRequest{Request: req}
}

// readHttpRequest reads and parses an HTTP request from the given reader
func readHttpRequest(rdr io.Reader) (*HttpRequest, error) {
	req, err := http.ReadRequest(bufio.NewReader(rdr))
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		return nil, fmt.Errorf("failed to read HTTP request: %w", err)
	}
	return NewHttpRequest(req), nil
}

// ExtractDomain returns the host without port information
func (r *HttpRequest) ExtractDomain() string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		return r.Host
	}
	return host
}

// ExtractPort returns the port from the host or empty string if not specified
func (r *HttpRequest) ExtractPort() (int, error) {
	_, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		if r.Method == http.MethodConnect {
			return 443, nil
		} else {
			return 80, nil
		}
	}

	return strconv.Atoi(port)
}

// IsValidMethod returns true if the request method is a valid HTTP method
func (r *HttpRequest) IsValidMethod() bool {
	return validMethods[r.Method]
}

// IsConnectMethod returns true if the request method is CONNECT
func (r *HttpRequest) IsConnectMethod() bool {
	return r.Method == http.MethodConnect
}

func (r *HttpRequest) ResBadGateway() []byte {
	return []byte(r.Proto + " 502 Bad Gateway\r\n\r\n")
}

func (r *HttpRequest) ResConnectionEstablished() []byte {
	return []byte(r.Proto + " 200 Connection Established\r\n\r\n")
}
