package socks5

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/netutil"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type UDPHandler struct {
	logger zerolog.Logger
}

func NewUDPHandler(logger zerolog.Logger) *UDPHandler {
	return &UDPHandler{
		logger: logger,
	}
}

func (h *UDPHandler) Handle(
	ctx context.Context,
	conn net.Conn,
	req *proto.SOCKS5Request,
	dst *netutil.Destination,
	rule *config.Rule,
) error {
	logger := h.logger.With().Ctx(ctx).Logger()

	// 1. Listen on a random UDP port
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create udp listener")
		_ = proto.SOCKS5FailureResponse().Write(conn)
		return err
	}
	netutil.CloseConns(udpConn)

	lAddr := udpConn.LocalAddr().(*net.UDPAddr)

	logger.Debug().
		Str("bind_addr", lAddr.String()).
		Msg("socks5 udp associate established")

	// 2. Reply with the bound address
	if err := proto.SOCKS5SuccessResponse().Bind(lAddr.IP).Port(lAddr.Port).Write(conn); err != nil {
		logger.Error().Err(err).Msg("failed to write socks5 success reply")
		return err
	}

	// 3. Keep TCP Alive & Relay
	// We need to monitor TCP for closure.
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, conn) // Block until TCP closes
		close(done)
	}()

	go func() {
		<-done
		netutil.CloseConns(udpConn)
	}()

	buf := make([]byte, 65535)
	var clientAddr *net.UDPAddr

	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			// Normal closure check
			select {
			case <-done:
				return nil
			default:
				logger.Debug().Err(err).Msg("error reading from udp")
				return err
			}
		}

		// Initial Client Identification
		if clientAddr == nil {
			clientAddr = addr
		}

		if addr.IP.Equal(clientAddr.IP) && addr.Port == clientAddr.Port {
			// Outbound: Client -> Proxy -> Target
			targetAddr, payload, err := parseUDPHeader(buf[:n])
			if err != nil {
				logger.Warn().Err(err).Msg("failed to parse socks5 udp header")
				continue
			}

			// We use the same UDP socket to send to target.
			// The Target will reply to this socket.
			resolvedAddr, err := net.ResolveUDPAddr("udp", targetAddr)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("addr", targetAddr).
					Msg("failed to resolve udp target")
				continue
			}

			if _, err := udpConn.WriteTo(payload, resolvedAddr); err != nil {
				logger.Warn().Err(err).Msg("failed to write udp to target")
			}
		} else {
			// Inbound: Target -> Proxy -> Client
			// Wrap with SOCKS5 Header
			header := createUDPHeaderFromAddr(addr)
			response := append(header, buf[:n]...)

			if _, err := udpConn.WriteToUDP(response, clientAddr); err != nil {
				logger.Warn().Err(err).Msg("failed to write udp to client")
			}
		}
	}
}

func parseUDPHeader(b []byte) (string, []byte, error) {
	if len(b) < 4 {
		return "", nil, fmt.Errorf("header too short")
	}
	// RSV(2) FRAG(1) ATYP(1)
	if b[0] != 0 || b[1] != 0 {
		return "", nil, fmt.Errorf("invalid rsv")
	}
	frag := b[2]
	if frag != 0 {
		return "", nil, fmt.Errorf("fragmentation not supported")
	}

	atyp := b[3]
	var host string
	var pos int

	switch atyp {
	case proto.ATYPIPv4:
		if len(b) < 10 {
			return "", nil, fmt.Errorf("header too short for ipv4")
		}
		host = net.IP(b[4:8]).String()
		pos = 8
	case proto.ATYPIPv6:
		if len(b) < 22 {
			return "", nil, fmt.Errorf("header too short for ipv6")
		}
		host = net.IP(b[4:20]).String()
		pos = 20
	case proto.ATYPFQDN:
		if len(b) < 5 {
			return "", nil, fmt.Errorf("header too short for fqdn")
		}
		l := int(b[4])
		if len(b) < 5+l+2 {
			return "", nil, fmt.Errorf("header too short for fqdn data")
		}
		host = string(b[5 : 5+l])
		pos = 5 + l
	default:
		return "", nil, fmt.Errorf("unsupported atyp: %d", atyp)
	}

	port := binary.BigEndian.Uint16(b[pos : pos+2])
	pos += 2

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	return addr, b[pos:], nil
}

func createUDPHeaderFromAddr(addr *net.UDPAddr) []byte {
	// RSV(2) FRAG(1) ATYP(1) ...
	buf := make([]byte, 0, 24)
	buf = append(buf, 0, 0, 0) // RSV, FRAG

	ip4 := addr.IP.To4()
	if ip4 != nil {
		buf = append(buf, proto.ATYPIPv4)
		buf = append(buf, ip4...)
	} else {
		buf = append(buf, proto.ATYPIPv6)
		buf = append(buf, addr.IP.To16()...)
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(addr.Port))
	buf = append(buf, portBuf...)

	return buf
}
