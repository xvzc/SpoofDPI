package util

import (
	"log"
	"net"
	"strings"
    "errors"

	"github.com/babolivier/go-doh-client"
	"github.com/xvzc/SpoofDPI/config"
)

const BUF_SIZE = 1024

func WriteAndRead(conn net.Conn, message []byte) ([]byte, error){
    _, err := conn.Write(message)
    if err != nil {
        log.Fatal("Error writing to client:", err)
        return nil, err
    }
    // defer conn.(*net.TCPConn).CloseWrite()

    buf, err := ReadMessage(conn)
    if err != nil {
        log.Fatal("failed:", err)
        return nil, err
    }

    return buf, nil
}

func ReadMessage(conn net.Conn)([]byte, error) {
    buf := make([]byte, 0) // big buffer
    tmp := make([]byte, BUF_SIZE)     // using small tmo buffer for demonstrating

    for {
        n, err := conn.Read(tmp)
        if err != nil {
            return nil, err
        }
        buf = append(buf, tmp[:n]...)

        if n < BUF_SIZE {
            break
        }
    }

    return buf, nil
}

func ExtractDomain(message *[]byte) (string) {
    i := 0
    for ; i < len(*message); i++ {
        if (*message)[i] == '\n' {
            i++
            break;
        }
    }

    for ; i < len(*message); i++ {
        if (*message)[i] == ' ' {
            i++
            break;
        }
    }

    j := i
    for ; j < len(*message); j++ {
        if (*message)[j] == '\n' {
            break;
        }
    }

    domain := strings.Split(string((*message)[i:j]), ":")[0]

    return strings.TrimSpace(domain)
}

func DnsLookupOverHttps(dns string, domain string)(string, error) {
    // Perform a A lookup on example.com
    resolver := doh.Resolver{
        Host:  dns, // Change this with your favourite DoH-compliant resolver.
        Class: doh.IN,
    }

    log.Println(domain)
    a, _, err := resolver.LookupA(domain)
    if err != nil {
        log.Println("Error looking up dns. ", err)
        return "", err
    }

    ip := a[0].IP4 

    return ip, nil
}

func ExtractMethod(message *[]byte) (string) {
    i := 0
    for ; i < len(*message); i++ {
        if (*message)[i] == ' ' {
            break;
        }
    }

    method := strings.TrimSpace(string((*message)[:i]))
    log.Println(method)

    return strings.ToUpper(method)
}

func SplitSliceInChunks(a []byte, size int) ([][]byte, error) {
	if size < 1 {
		return nil, errors.New("chuckSize must be greater than zero")
	}
	chunks := make([][]byte, 0, (len(a)+size-1)/size)

	for size < len(a) {
		a, chunks = a[size:], append(chunks, a[0:size:size])
	}
	chunks = append(chunks, a)
	return chunks, nil
}

func Debug(v ...interface{}) {
    if config.GetConfig().Debug == false {
        return
    }

    log.Println(v...)
}
