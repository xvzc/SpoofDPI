package proto

import (
	"encoding/binary"
	"errors"
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
	Raw        []byte // Header + Payload
	RawHeader  []byte
	RawPayload []byte
}

type TLSHeader struct {
	Type         TLSMessageType
	ProtoVersion uint16 // major | minor
	PayloadLen   uint16
}

func (h *TLSHeader) Bytes() []byte {
	buf := make([]byte, 5)

	// Type (1 byte)
	buf[0] = byte(h.Type)

	binary.BigEndian.PutUint16(buf[1:3], h.ProtoVersion)

	binary.BigEndian.PutUint16(buf[3:5], h.PayloadLen)

	return buf
}

func (m *TLSMessage) IsClientHello() bool {
	// According to RFC 8446 section 4,
	// the first byte (Raw[5]) of a handshake message should be 0x1 for a Client Hello.
	return len(m.Raw) > TLSHeaderLen &&
		m.Header.Type == TLSHandshake &&
		m.Raw[5] == 0x01
}

// ExtractSNIOffset parses the Client Hello and returns the indices of the SNI field.
// The returned `start` and `end` values are absolute offsets into `m.Raw []bytes`.
// [start:end] covers only the SNI hostname value (i.e., the actual server name string),
// and does NOT include the SNI extension header, type, or length fields.
// If SNI is not found or the packet is invalid, returns (0, 0, error).
//
// Example:
//
//	  Suppose m.Raw contains the following (hex, simplified):
//		   ... [TLS headers] ... [Extensions] ...
//		   00 00    // SNI extension type (0x00 0x00)
//		   00 0e    // SNI extension length (0x00 0x0e)
//		   00 0c    // SNI list length (0x00 0x0c)
//		   00       // SNI type (0x00)
//		   00 09    // SNI value length (0x00 0x09)
//		   65 78 61 6d 70 6c 65 2e 63 6f 6d // SNI value ("example.com")
//		   ...
//
//		ExtractSNIOffset will return start=position of '65', end=position after '6d',
//		covering only "example.com".
//
// This allows callers to extract the SNI hostname directly via m.Raw[start:end].
func (m *TLSMessage) ExtractSNIOffset() (start int, end int, err error) {
	// 1. Basic Length Check
	if len(m.Raw) < 43 { // Minimal headers size
		return 0, 0, errors.New("packet too short")
	}

	// 2. Skip Record Header (5 bytes) + Handshake Header (4 bytes)
	// ContentType(1) + Ver(2) + Len(2) + MsgType(1) + Len(3) = 9 bytes usually.
	// But strictly speaking:
	// Record Layer: 0-4
	// Handshake: 5-8
	// Client Hello Body starts at 9

	// Fast-forward pointer
	curr := 0

	// Check Content Type (Must be Handshake 0x16)
	if m.Raw[curr] != 0x16 {
		return 0, 0, errors.New("not a handshake packet")
	}
	curr += 5 // Skip Record Header

	// Check Handshake Type (Must be Client Hello 0x01)
	if m.Raw[curr] != 0x01 {
		return 0, 0, errors.New("not a client hello")
	}
	curr += 4 // Skip Handshake Header

	// Skip Protocol Version (2) + Random (32)
	curr += 34
	if curr >= len(m.Raw) {
		return 0, 0, errors.New("packet too short after random")
	}

	// 3. Skip Session ID
	sessionIDLen := int(m.Raw[curr])
	curr += 1 + sessionIDLen
	if curr >= len(m.Raw) {
		return 0, 0, errors.New("packet too short after session id")
	}

	// 4. Skip Cipher Suites
	if curr+2 > len(m.Raw) {
		return 0, 0, errors.New("packet too short for cipher suites len")
	}
	cipherSuitesLen := int(binary.BigEndian.Uint16(m.Raw[curr : curr+2]))
	curr += 2 + cipherSuitesLen
	if curr >= len(m.Raw) {
		return 0, 0, errors.New("packet too short after cipher suites")
	}

	// 5. Skip Compression Methods
	if curr+1 > len(m.Raw) {
		return 0, 0, errors.New("packet too short for compression len")
	}
	compressionMethodsLen := int(m.Raw[curr])
	curr += 1 + compressionMethodsLen
	if curr >= len(m.Raw) {
		return 0, 0, errors.New("packet too short after compression")
	}

	// 6. Parse Extensions
	if curr+2 > len(m.Raw) {
		// No extensions present
		return 0, 0, errors.New("no extensions")
	}
	extensionsLen := int(binary.BigEndian.Uint16(m.Raw[curr : curr+2]))
	curr += 2

	extensionsEnd := curr + extensionsLen
	if extensionsEnd > len(m.Raw) {
		return 0, 0, errors.New("extensions length overflow")
	}

	// Loop through extensions
	for curr < extensionsEnd {
		if curr+4 > extensionsEnd {
			break
		}

		extType := binary.BigEndian.Uint16(m.Raw[curr : curr+2])
		extLen := int(binary.BigEndian.Uint16(m.Raw[curr+2 : curr+4]))
		curr += 4

		if curr+extLen > extensionsEnd {
			break
		}

		// Found SNI Extension (Type 0x0000)
		if extType == 0x0000 {
			// SNI structure: ListLen(2) + Type(1) + NameLen(2) + Hostname(...)
			// What we need is the actual location of the 'Hostname' string.
			// Typically, at curr+3 (after ListLen 2 + Type 1) is NameLen,
			// and from curr+5 the actual domain string begins.

			// To target only the domain string precisely:
			sniDataStart := curr
			if extLen < 5 {
				return 0, 0, errors.New("malformed sni extension")
			}

			// Calculate the actual domain string start position
			// List Length (2 bytes) skip
			// Name Type (1 byte) skip
			// Name Length (2 bytes) read
			nameLen := int(binary.BigEndian.Uint16(m.Raw[sniDataStart+3 : sniDataStart+5]))

			realStart := sniDataStart + 5
			realEnd := realStart + nameLen

			if realEnd > curr+extLen {
				return 0, 0, errors.New("malformed sni length")
			}

			return realStart, realEnd, nil
		}

		curr += extLen
	}

	return 0, 0, errors.New("sni not found")
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
		// Corrupted header? Check for integer overflow.
		return nil, fmt.Errorf(
			"invalid TLS header; Type=%x, ProtoVersion: %x, PayloadLen: %x",
			header.Type,
			header.ProtoVersion,
			header.PayloadLen,
		)
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
