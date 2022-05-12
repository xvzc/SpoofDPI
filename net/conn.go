package net

import (
	"errors"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xvzc/SpoofDPI/doh"
	"github.com/xvzc/SpoofDPI/packet"
)

const BUF_SIZE = 1024

type Conn struct {
	conn net.Conn
}

func (c *Conn) CloseWrite() {
	c.conn.(*net.TCPConn).CloseWrite()
}

func (c *Conn) Close() {
	c.conn.Close()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *Conn) SetReadDeadline(t time.Time) (error) {
    c.conn.SetReadDeadline(t)
    return nil
}

func (c *Conn) SetDeadLine(t time.Time) (error) {
    c.conn.SetDeadline(t)
    return nil
}

func (c *Conn) SetKeepAlive(b bool) (error) {
    c.conn.(*net.TCPConn).SetKeepAlive(b)
    return nil
}

func (conn *Conn) WriteChunks(c [][]byte) (n int, err error) {
	total := 0
	for i := 0; i < len(c); i++ {
		b, err := conn.Write(c[i])
		if err != nil {
			return 0, nil
		}

		b += total
	}

	return total, nil
}

func (conn *Conn) ReadBytes() ([]byte, error) {
	ret := make([]byte, 0)
	buf := make([]byte, BUF_SIZE)

	for {
		n, err := conn.Read(buf)
        if err != nil {
            switch err.(type) {
            case *net.OpError:
                return nil, errors.New("timed out")
            default:
                return nil, err
            }
        }
		ret = append(ret, buf[:n]...)

		if n < BUF_SIZE {
            break
		}
	}

    return ret, nil
}

func (lConn *Conn) HandleHttp(p *packet.HttpPacket) {
    defer func() {
        lConn.Close()
        log.Debug("[HTTP] Closing client Connection.. ", lConn.RemoteAddr())
    }()

	p.Tidy()

	ip, err := doh.Lookup(p.Domain())
	if err != nil {
        log.Error("[HTTP DOH] Error looking up for domain with ", p.Domain() , " ", err)
        lConn.Write([]byte(p.Version() + " 502 Bad Gateway\r\n\r\n"))
        return
	}

	log.Debug("[DOH] Found ", ip, " with ", p.Domain())

	// Create connection to server
    var port = ":80"
    if p.Port() != "" {
        port = ":" + p.Port()
    }

	rConn, err := Dial("tcp", ip + port)
	if err != nil {
		log.Debug("[HTTP] ", err)
		return
	}

    defer func() {
        defer rConn.Close()
        log.Debug("[HTTP] Closing server Connection.. ", p.Domain(), " ", rConn.LocalAddr())
    }()

    log.Debug("[HTTP] New connection to the server ", p.Domain(), " ", rConn.LocalAddr())

	_, err = rConn.Write(p.Raw())
	if err != nil {
		log.Debug("[HTTP] Error sending request to ", p.Domain(), err)
		return
	}

	log.Debug("[HTTP] Sent a request to ", p.Domain())

    go lConn.Serve(rConn, "[HTTP]", lConn.RemoteAddr().String(), p.Domain())
    rConn.Serve(lConn, "[HTTP]", lConn.RemoteAddr().String(), p.Domain())

}

func (lConn *Conn) HandleHttps(p *packet.HttpPacket) {
    defer func() {
        lConn.Close()
        log.Debug("[HTTPS] Closing client Connection.. ", lConn.RemoteAddr())
    }()

	ip, err := doh.Lookup(p.Domain())
	if err != nil {
		log.Error("[HTTPS DOH] Error looking up for domain: ", p.Domain(), " ", err)
        lConn.Write([]byte(p.Version() + " 502 Bad Gateway\r\n\r\n"))
        return
	}

	log.Debug("[DOH] Found ", ip, " with ", p.Domain())

	// Create a connection to the requested server
    var port = ":443"
    if p.Port() != "" {
        port = ":" + p.Port()
    }

	rConn, err := Dial("tcp", ip + port)
	if err != nil {
		log.Debug("[HTTPS] ", err)
		return
	}

    defer func() {
        defer rConn.Close()
        log.Debug("[HTTPS] Closing server Connection.. ", p.Domain(), " ", rConn.LocalAddr())
    }()

    log.Debug("[HTTPS] New connection to the server ", p.Domain(), " ", rConn.LocalAddr())

	_, err = lConn.Write([]byte(p.Version() + " 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Debug("[HTTPS] Error sending 200 Connection Established to the client", err)
        return
	}
	log.Debug("[HTTPS] Sent 200 Connection Estabalished to the client")

	// Read client hello
	clientHello, err := lConn.ReadBytes()
	if err != nil {
		log.Debug("[HTTPS] Error reading client hello from the client", err)
        return
	}

	log.Debug("[HTTPS] Client sent hello ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server

	pkt := packet.NewHttpsPacket(clientHello)

	chunks := pkt.SplitInChunks()

	if _, err := rConn.WriteChunks(chunks); err != nil {
		log.Debug("[HTTPS] Error writing client hello to ", p.Domain(), err)
		return
	}

    go lConn.Serve(rConn, "[HTTPS]", lConn.RemoteAddr().String(), p.Domain())
    rConn.Serve(lConn, "[HTTPS]", lConn.RemoteAddr().String(), p.Domain())
}

func (from *Conn) Serve(to *Conn, proto string, fd string, td string) {
	proto += " "

    for {
        from.conn.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
        buf, err := from.ReadBytes()
        if err != nil {
            log.Debug(proto, "Error reading from ", fd, " ", err)
            return
        } 

        if _, err := to.Write(buf); err != nil {
            log.Debug(proto, "Error Writing to ", td)
            return
        } 
    }
}
