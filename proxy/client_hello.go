package proxy

import (
	"encoding/binary"
	"io"
)

const headerLen = 5

type ClientHello struct {
	Header     ClientHelloHeader
	Raw        []byte //Header + Payload
	RawHeader  []byte
	RawPayload []byte
}

type ClientHelloHeader struct {
	Type         byte
	ProtoVersion uint16
	PayloadLen   uint16
}

func ReadClientHello(r io.Reader) (*ClientHello, error) {
	var rawHeader [5]byte
	_, err := io.ReadFull(r, rawHeader[:])
	if err != nil {
		return nil, err
	}

	header := ClientHelloHeader{
		Type:         rawHeader[0],
		ProtoVersion: binary.BigEndian.Uint16(rawHeader[1:3]),
		PayloadLen:   binary.BigEndian.Uint16(rawHeader[3:5]),
	}
	raw := make([]byte, header.PayloadLen+headerLen)
	copy(raw[0:headerLen], rawHeader[:])
	_, err = io.ReadFull(r, raw[headerLen:])
	if err != nil {
		return nil, err
	}
	hello := &ClientHello{
		Header: header,
		Raw:    raw,
	}
	hello.RawHeader = hello.Raw[:headerLen]
	hello.RawPayload = hello.Raw[headerLen:]
	return hello, nil
}
