package proto

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// SOCKS5 Protocol Constants
const (
	// Version
	SOCKSVersion = 0x05

	// Auth
	AuthNone     = 0x00
	AuthGSSAPI   = 0x01
	AuthUserPass = 0x02
	AuthNoAccept = 0xFF

	// Command
	CmdConnect      = 0x01
	CmdBind         = 0x02
	CmdUDPAssociate = 0x03

	// ATYP
	ATYPIPv4 = 0x01
	ATYPFQDN = 0x03
	ATYPIPv6 = 0x04

	// Reply codes
	ReplyCodeSuccess        = 0x00
	ReplyCodeGenFailure     = 0x01
	ReplyCodeCmdNotSupport  = 0x07
	ReplyCodeAddrNotSupport = 0x08
)

type SOCKS5Request struct {
	Cmd    byte
	Domain string
	IP     net.IP
	Port   int
}

func (r *SOCKS5Request) Host() string {
	ret := r.Domain
	if ret == "" {
		ret = r.IP.String()
	}

	return ret
}

// ReadSocks5Request parses the SOCKS5 request details.
func ReadSocks5Request(conn net.Conn) (*SOCKS5Request, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	if header[0] != SOCKSVersion {
		return nil, fmt.Errorf(
			"version mismatch: expected %x, got %x",
			SOCKSVersion,
			header[0],
		)
	}

	cmd := header[1]
	atyp := header[3]

	var domain string
	var ip net.IP

	switch atyp {
	case ATYPIPv4:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, err
		}
		ip = net.IP(buf)

	case ATYPFQDN:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return nil, err
		}
		domainLen := int(lenBuf[0])
		domainBuf := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domainBuf); err != nil {
			return nil, err
		}
		domain = string(domainBuf)

	case ATYPIPv6:
		buf := make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, err
		}
		ip = net.IP(buf)

	default:
		return nil, fmt.Errorf("unsupported atyp: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return nil, err
	}
	port := int(binary.BigEndian.Uint16(portBuf))

	return &SOCKS5Request{
		Cmd:    cmd,
		Domain: domain,
		IP:     ip,
		Port:   port,
	}, nil
}

type SOCKS5Reply struct {
	Rep      byte
	BindIP   net.IP
	BindPort int
}

func NewSOCKS5Reply(rep byte) *SOCKS5Reply {
	return &SOCKS5Reply{
		Rep:      rep,
		BindIP:   net.IPv4zero,
		BindPort: 0,
	}
}

func SOCKS5SuccessResponse() *SOCKS5Reply {
	return NewSOCKS5Reply(ReplyCodeSuccess)
}

func SOCKS5FailureResponse() *SOCKS5Reply {
	return NewSOCKS5Reply(ReplyCodeGenFailure)
}

func SOCKS5CommandNotSupportedResponse() *SOCKS5Reply {
	return NewSOCKS5Reply(ReplyCodeCmdNotSupport)
}

func (r *SOCKS5Reply) Bind(ip net.IP) *SOCKS5Reply {
	if ip != nil {
		r.BindIP = ip
	}
	return r
}

func (r *SOCKS5Reply) Port(port int) *SOCKS5Reply {
	r.BindPort = port
	return r
}

func (r *SOCKS5Reply) Write(w io.Writer) error {
	buf := make([]byte, 0, 10)
	buf = append(buf, SOCKSVersion, r.Rep, 0x00, ATYPIPv4)

	// Use To4() to ensure 4 bytes if it's an IPv4 address stored in IPv6 format
	if ip4 := r.BindIP.To4(); ip4 != nil {
		buf = append(buf, ip4...)
	} else {
		// Fallback or Handle IPv6
		buf = append(buf, make([]byte, 4)...)
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(r.BindPort))
	buf = append(buf, portBuf...)

	_, err := w.Write(buf)
	return err
}
