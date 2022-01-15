package net

import (
	"net"

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
			return nil, err
		}
		ret = append(ret, buf[:n]...)

		if n < BUF_SIZE {
			break
		}
	}

	return ret, nil
}

func (lConn *Conn) HandleHttp(p packet.HttpPacket) {
	p.Tidy()

	log.Debug("[HTTP] request: \n\n" + string(p.Raw()))

	ip, err := doh.Lookup(p.Domain())
	if err != nil {
		log.Debug("[HTTP] Error looking up for domain: ", err)
	}
	log.Debug("[HTTP] Found ip over HTTPS: ", ip)

	// Create connection to server
	rConn, err := Dial("tcp", ip+":80")
	if err != nil {
		log.Debug("[HTTPS] ", err)
		return
	}

	log.Debug("[HTTP] Connected to the server.")

	go rConn.Serve(lConn, "[HTTP]")
	go lConn.Serve(rConn, "[HTTP]")

	_, err = rConn.Write(p.Raw())
	if err != nil {
		log.Debug("[HTTP] Error sending request to the server: ", err)
		return
	}
	log.Debug("[HTTP] Sent a request to the server")
}

func (lConn *Conn) HandleHttps(p packet.HttpPacket) {
	log.Debug("[HTTPS] request: \n\n" + string(p.Raw()))

	ip, err := doh.Lookup(p.Domain())
	if err != nil {
		log.Debug("[HTTPS] Error looking up for domain: ", p.Domain(), " ", err)
	}
	log.Debug("[HTTPS] Found ip over HTTPS: ", ip)

	// Create a connection to the requested server
	rConn, err := Dial("tcp", ip+":443")
	if err != nil {
		log.Debug("[HTTPS] ", err)
		return
	}

	log.Debug("[HTTPS] Connected to the server.")

	_, err = lConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Debug("[HTTPS] Error sending client hello: ", err)
	}
	log.Debug("[HTTPS] Sent 200 Connection Estabalished")

	// Read client hello
	clientHello, err := lConn.ReadBytes()
	if err != nil {
		log.Debug("[HTTPS] Error reading client hello: ", err)
		log.Debug("[HTTPS] Closing connection: ", lConn.RemoteAddr())
	}

	log.Debug("[HTTPS] Client "+lConn.RemoteAddr().String()+" sent hello: ", len(clientHello), "bytes")

	// Generate a go routine that reads from the server
	go rConn.Serve(lConn, "[HTTPS]")
	go lConn.Serve(rConn, "[HTTPS]")

	pkt := packet.NewHttpsPacket(clientHello)

	chunks := pkt.SplitInChunks()

	if _, err := rConn.WriteChunks(chunks); err != nil {
		log.Debug("[HTTPS] Error writing client hello: ", err)
		return
	}
}

func (from *Conn) Serve(to *Conn, proto string) {
	defer from.Close()
	defer to.CloseWrite()

	proto += " "

	for {
		buf, err := from.ReadBytes()
		if err != nil {
			log.Debug(proto+"Error reading from ", from.RemoteAddr())
			log.Debug(proto, err)
			log.Debug(proto + "Exiting Serve() method. ")
			break
		}

		log.Debug(proto, from.RemoteAddr(), " sent data: ", len(buf), "bytes")

		if _, err := to.Write(buf); err != nil {
			log.Debug(proto+"Error Writing to ", to.RemoteAddr())
			log.Debug(proto, err)
			log.Debug(proto + "Exiting Serve() method. ")
			break
		}
	}
}
