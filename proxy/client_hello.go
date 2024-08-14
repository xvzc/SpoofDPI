package proxy

import (
	"encoding/binary"
	"io"
)

const headerLen = 5

type TLSMessageType byte

const (
	TLSInvalid          TLSMessageType = 0x0
	TLSChangeCipherSpec TLSMessageType = 0x14
	TLSAlert            TLSMessageType = 0x15
	TLSHandshake        TLSMessageType = 0x16
	TLSApplicationData  TLSMessageType = 0x17
	TLSHeartbeat        TLSMessageType = 0x18
)

type TlsMessage struct {
	Header     TlsHeader
	Raw        []byte //Header + Payload
	RawHeader  []byte
	RawPayload []byte
}

type TlsHeader struct {
	Type         TLSMessageType
	ProtoVersion uint16 // major | minor
	PayloadLen   uint16
}

func ReadTlsMessage(r io.Reader) (*TlsMessage, error) {
	var rawHeader [5]byte
	_, err := io.ReadFull(r, rawHeader[:])
	if err != nil {
		return nil, err
	}

	header := TlsHeader{
		Type:         TLSMessageType(rawHeader[0]),
		ProtoVersion: binary.BigEndian.Uint16(rawHeader[1:3]),
		PayloadLen:   binary.BigEndian.Uint16(rawHeader[3:5]),
	}
	raw := make([]byte, header.PayloadLen+headerLen)
	copy(raw[0:headerLen], rawHeader[:])
	_, err = io.ReadFull(r, raw[headerLen:])
	if err != nil {
		return nil, err
	}
	hello := &TlsMessage{
		Header:     header,
		Raw:        raw,
		RawHeader:  raw[:headerLen],
		RawPayload: raw[headerLen:],
	}
	return hello, nil
}

func IsClientHello(message *TlsMessage) bool {
	// According to RFC 8446 section 4.
	// first byte (Raw[5]) of handshake message should be 0x1 - means client_hello
	return message.Header.Type == TLSHandshake &&
		message.Raw[5] == 0x1
}
