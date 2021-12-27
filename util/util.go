package util

import (
	"log"
	"net"
	"strings"
    "io"
    "github.com/babolivier/go-doh-client"
)

func ExtractDomainAndPort(s string) (string, string) {
    lines := strings.Split(s, "\n")
    hostPart := strings.Split(lines[1], " ")[1]
    tokens := strings.Split(hostPart, ":")
    if len(tokens) == 1 {
        return strings.TrimSpace(string(tokens[0])), "80"
    }

    return string(tokens[0]), string(tokens[1])
}

func ReadBytes(conn net.Conn)([]byte, error) {
    buf := make([]byte, 0, 4096) // big buffer
    tmp := make([]byte, 1024)     // using small tmo buffer for demonstrating
    for {
        n, err := conn.Read(tmp)
        if err != nil {
            if err != io.EOF {
                log.Fatal("ReadRequest error:", err)
            }
            return nil, err
        }
        log.Println("##### got", n, "bytes.")
        buf = append(buf, tmp[:n]...)

        if n < 1024 {
            break
        }
    }

    return buf, nil
}

func DnsLookupOverHttps(addr string, domain string)(string, error) {
    // Perform a A lookup on example.com
    resolver := doh.Resolver{
        Host:  addr, // Change this with your favourite DoH-compliant resolver.
        Class: doh.IN,
    }

    a, _, err := resolver.LookupA(domain)
    if err != nil {
        return "", err
    }

    ip := a[0].IP4 

    return ip, nil
}
