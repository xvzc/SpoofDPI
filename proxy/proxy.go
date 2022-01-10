package proxy

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/babolivier/go-doh-client"
	"github.com/pterm/pterm"
	"github.com/xvzc/SpoofDPI/packet"
)

type Proxy struct {
	Port  string
	DNS   doh.Resolver
	OS    string
	Debug bool
}

func New(port string, dns string, os string, debug bool) *Proxy {
	return &Proxy{
		Port:  port,
		DNS:   doh.Resolver{Host: dns, Class: doh.IN},
		OS:    os,
		Debug: debug,
	}
}

func (p *Proxy) PrintWelcome() {
	cyan := pterm.NewLettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := pterm.NewLettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "PORT  : " + p.Port},
		{Level: 0, Text: "DNS   : " + p.DNS.Host},
		{Level: 0, Text: "DEBUG : " + fmt.Sprint(p.Debug)},
	}).Render()
}

func (p *Proxy) Start() {
	listener, err := net.Listen("tcp", ":"+p.Port)
	if err != nil {
		log.Fatal("Error creating listener: ", err)
		os.Exit(1)
	}

	// util.Debug("Created a listener")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Fatal("Error accepting connection: ", err)
			continue
		}

		// util.Debug("Accepted a new connection.", clientConn.RemoteAddr())

		go func() {
			defer clientConn.Close()

			b, err := ReadBytes(clientConn)
			if err != nil {
				return
			}

			// util.Debug("Client sent data: ", len(b))

			r := packet.NewHttpPacket(&b)
			// util.Debug("Request: \n" + string(*r.Raw))

			if !r.IsValidMethod() {
				log.Println("Unsupported method: ", r.Method)
				return
			}

			// Dns lookup over https
			ip, err := p.DnsLookupOverHttps(r.Domain)
			if err != nil {
				log.Println("Error looking up dns: "+r.Domain, err)
				return
			}

			// util.Debug("ip: " + ip)

			if r.IsConnectMethod() {
				// util.Debug("HTTPS Requested")
				HandleHttps(clientConn, ip, &r)
			} else {
				// util.Debug("HTTP Requested.")
				HandleHttp(clientConn, ip, &r)
			}
		}()
	}
}
