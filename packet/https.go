package packet

import (
	"encoding/binary"
	"fmt"
	"io"
)

type TLSMessageType byte

const (
	TLSMaxPayloadLen    uint16         = 16384 // 16 KB
	TLSHeaderLen                       = 5
	TLSInvalid          TLSMessageType = 0x0
	TLSChangeCipherSpec TLSMessageType = 0x14
	TLSAlert            TLSMessageType = 0x15
	TLSHandshake        TLSMessageType = 0x16
	TLSApplicationData  TLSMessageType = 0x17
	TLSHeartbeat        TLSMessageType = 0x18
)

type TLSMessage struct {
	Header     TLSHeader
	Raw        []byte //Header + Payload
	RawHeader  []byte
	RawPayload []byte
}

type TLSHeader struct {
	Type         TLSMessageType
	ProtoVersion uint16 // major | minor
	PayloadLen   uint16
}

func ReadTLSMessage(r io.Reader) (*TLSMessage, error) {
	var rawHeader [TLSHeaderLen]byte
	_, err := io.ReadFull(r, rawHeader[:])
	if err != nil {
		return nil, err
	}

	header := TLSHeader{
		Type:         TLSMessageType(rawHeader[0]),
		ProtoVersion: binary.BigEndian.Uint16(rawHeader[1:3]),
		PayloadLen:   binary.BigEndian.Uint16(rawHeader[3:5]),
	}
	if header.PayloadLen > TLSMaxPayloadLen {
		// Corrupted header? Check integer overflow
		return nil, fmt.Errorf("invalid TLS header. Type: %x, ProtoVersion: %x, PayloadLen: %x", header.Type, header.ProtoVersion, header.PayloadLen)
	}
	raw := make([]byte, header.PayloadLen+TLSHeaderLen)
	copy(raw[0:TLSHeaderLen], rawHeader[:])
	_, err = io.ReadFull(r, raw[TLSHeaderLen:])
	if err != nil {
		return nil, err
	}

	hello := &TLSMessage{
		Header:     header,
		Raw:        raw,
		RawHeader:  raw[:TLSHeaderLen],
		RawPayload: raw[TLSHeaderLen:],
	}
	return hello, nil
}

func (m *TLSMessage) IsClientHello() bool {
	// According to RFC 8446 section 4.
	// first byte (Raw[5]) of handshake message should be 0x1 - means client_hello
	return len(m.Raw) > TLSHeaderLen &&
		m.Header.Type == TLSHandshake &&
		m.Raw[5] == 0x01
}
