// Package tls parses just enough of the TLS handshake to locate the
// ClientHello and extract the Server Name Indication (SNI). It deliberately
// does not implement TLS — it only reads the plaintext handshake bytes that
// travel in the clear at the start of a connection, which is exactly what a
// DPI engine inspects and what Skvoz needs to target and split.
package tls

import "errors"

const (
	recordTypeHandshake  = 0x16
	handshakeTypeHello   = 0x01
	extensionServerName  = 0x0000
	serverNameTypeHost   = 0x00
	tlsRecordHeaderLen   = 5
	handshakeHeaderLen   = 4
	clientRandomLen      = 32
	minClientHelloRecord = tlsRecordHeaderLen + handshakeHeaderLen + 2 + clientRandomLen
)

// ErrNotClientHello is returned when the payload is not a TLS ClientHello.
var ErrNotClientHello = errors.New("tls: not a ClientHello record")

// ErrNoSNI is returned when the ClientHello carries no SNI extension.
var ErrNoSNI = errors.New("tls: no SNI extension present")

// ClientHelloInfo describes a parsed ClientHello.
type ClientHelloInfo struct {
	// SNI is the requested host name (lowercased as sent).
	SNI string
	// SNIOffset is the byte offset, within the payload passed to
	// ParseClientHello, where the SNI host name bytes begin. This is the
	// natural place to split a TCP segment so DPI cannot reassemble the name.
	SNIOffset int
	// SNILength is the length in bytes of the SNI host name.
	SNILength int
	// RecordLength is the total length of the TLS record (header + body).
	RecordLength int
}

// IsClientHello reports whether payload begins with a TLS handshake record
// whose handshake type is ClientHello. It is a cheap pre-filter.
func IsClientHello(payload []byte) bool {
	if len(payload) < tlsRecordHeaderLen+1 {
		return false
	}
	return payload[0] == recordTypeHandshake && payload[tlsRecordHeaderLen] == handshakeTypeHello
}

// ParseClientHello parses a TLS ClientHello contained at the start of payload
// (a single TCP segment's application data) and returns its SNI and the offset
// of the SNI host name within payload.
//
// It tolerates a payload longer than one record and ignores trailing bytes.
// It returns ErrNotClientHello if the payload is not a ClientHello, and ErrNoSNI
// if there is no server name extension.
func ParseClientHello(payload []byte) (ClientHelloInfo, error) {
	var info ClientHelloInfo
	if !IsClientHello(payload) {
		return info, ErrNotClientHello
	}
	if len(payload) < minClientHelloRecord {
		return info, ErrNotClientHello
	}

	recordLen := int(payload[3])<<8 | int(payload[4])
	info.RecordLength = tlsRecordHeaderLen + recordLen

	// Work over the handshake body but keep offsets relative to payload so the
	// caller can split the original segment at the returned SNIOffset.
	p := &parser{buf: payload, pos: tlsRecordHeaderLen}

	// Handshake header: type(1) + length(3). Type already checked.
	if _, ok := p.skip(handshakeHeaderLen); !ok {
		return info, ErrNotClientHello
	}
	// client_version(2) + random(32)
	if _, ok := p.skip(2 + clientRandomLen); !ok {
		return info, ErrNotClientHello
	}
	// session_id: length(1) + bytes
	if !p.skipVector8() {
		return info, ErrNotClientHello
	}
	// cipher_suites: length(2) + bytes
	if !p.skipVector16() {
		return info, ErrNotClientHello
	}
	// compression_methods: length(1) + bytes
	if !p.skipVector8() {
		return info, ErrNotClientHello
	}
	// extensions: length(2) + bytes
	extTotal, ok := p.readUint16()
	if !ok {
		return info, ErrNoSNI
	}
	extEnd := p.pos + int(extTotal)
	if extEnd > len(payload) {
		extEnd = len(payload)
	}

	for p.pos+4 <= extEnd {
		extType, _ := p.readUint16()
		extLen, _ := p.readUint16()
		extBodyStart := p.pos
		extBodyEnd := extBodyStart + int(extLen)
		if extBodyEnd > extEnd {
			return info, ErrNoSNI
		}
		if extType == extensionServerName {
			if off, length, ok := parseSNI(payload, extBodyStart, extBodyEnd); ok {
				info.SNI = string(payload[off : off+length])
				info.SNIOffset = off
				info.SNILength = length
				return info, nil
			}
			return info, ErrNoSNI
		}
		p.pos = extBodyEnd
	}
	return info, ErrNoSNI
}

// parseSNI parses a server_name extension body and returns the absolute offset
// and length of the first host_name entry.
func parseSNI(buf []byte, start, end int) (off, length int, ok bool) {
	// server_name_list: length(2) then entries.
	if start+2 > end {
		return 0, 0, false
	}
	listLen := int(buf[start])<<8 | int(buf[start+1])
	pos := start + 2
	listEnd := pos + listLen
	if listEnd > end {
		listEnd = end
	}
	for pos+3 <= listEnd {
		nameType := buf[pos]
		nameLen := int(buf[pos+1])<<8 | int(buf[pos+2])
		nameStart := pos + 3
		nameEnd := nameStart + nameLen
		if nameEnd > listEnd {
			return 0, 0, false
		}
		if nameType == serverNameTypeHost && nameLen > 0 {
			return nameStart, nameLen, true
		}
		pos = nameEnd
	}
	return 0, 0, false
}

// BuildClientHello constructs a minimal but structurally valid TLS ClientHello
// record carrying the given SNI. It is used to synthesize decoy handshakes and
// in tests. An empty sni produces a ClientHello with no SNI extension.
func BuildClientHello(sni string) []byte {
	var ext []byte
	if sni != "" {
		host := []byte(sni)
		entry := append([]byte{serverNameTypeHost, byte(len(host) >> 8), byte(len(host))}, host...)
		list := append([]byte{byte(len(entry) >> 8), byte(len(entry))}, entry...)
		ext = append([]byte{0x00, 0x00, byte(len(list) >> 8), byte(len(list))}, list...)
	}

	body := []byte{0x03, 0x03}                          // client_version TLS 1.2
	body = append(body, make([]byte, clientRandomLen)...) // random
	body = append(body, 0x00)                            // session_id length 0
	body = append(body, 0x00, 0x02, 0x13, 0x01)          // cipher_suites: len 2 + one suite
	body = append(body, 0x01, 0x00)                      // compression_methods: len 1 + null
	body = append(body, byte(len(ext)>>8), byte(len(ext)))
	body = append(body, ext...)

	hs := []byte{handshakeTypeHello, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}
	hs = append(hs, body...)

	rec := []byte{recordTypeHandshake, 0x03, 0x01, byte(len(hs) >> 8), byte(len(hs))}
	rec = append(rec, hs...)
	return rec
}

// parser is a tiny bounds-checked cursor over a byte slice.
type parser struct {
	buf []byte
	pos int
}

func (p *parser) skip(n int) (int, bool) {
	if p.pos+n > len(p.buf) {
		return p.pos, false
	}
	start := p.pos
	p.pos += n
	return start, true
}

func (p *parser) readUint16() (uint16, bool) {
	if p.pos+2 > len(p.buf) {
		return 0, false
	}
	v := uint16(p.buf[p.pos])<<8 | uint16(p.buf[p.pos+1])
	p.pos += 2
	return v, true
}

func (p *parser) skipVector8() bool {
	if p.pos+1 > len(p.buf) {
		return false
	}
	n := int(p.buf[p.pos])
	p.pos++
	_, ok := p.skip(n)
	return ok
}

func (p *parser) skipVector16() bool {
	n, ok := p.readUint16()
	if !ok {
		return false
	}
	_, ok = p.skip(int(n))
	return ok
}
