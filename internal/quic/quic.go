// Package quic identifies QUIC Initial packets. Skvoz does not decrypt QUIC in
// v1 (extracting the SNI would require removing header protection and AEAD-
// decrypting with the derived initial secrets). Instead it detects QUIC
// Initials on UDP/443 so the engine can suppress them, which makes browsers
// fall back to TLS-over-TCP — the path the desync strategies already handle.
package quic

// QUIC long-header first-byte layout (RFC 9000):
//
//	bit 7 (0x80) Header Form  = 1 (long header)
//	bit 6 (0x40) Fixed Bit    = 1
//	bits 5-4     Packet Type  = 00 for Initial (QUIC v1)
//
// Bytes 1..4 carry the 32-bit version; 0x00000001 is QUIC v1. Version 0 is a
// Version Negotiation packet.
const (
	longHeaderMask  = 0x80
	fixedBitMask    = 0x40
	packetTypeMask  = 0x30
	packetTypeShift = 4

	typeInitial = 0x0

	versionV1 = 0x00000001
)

// IsLongHeader reports whether payload begins with a QUIC long-header packet.
func IsLongHeader(payload []byte) bool {
	if len(payload) < 5 {
		return false
	}
	return payload[0]&longHeaderMask != 0 && payload[0]&fixedBitMask != 0
}

// Version returns the 32-bit QUIC version from a long-header packet, or 0 if the
// payload is too short.
func Version(payload []byte) uint32 {
	if len(payload) < 5 {
		return 0
	}
	return uint32(payload[1])<<24 | uint32(payload[2])<<16 | uint32(payload[3])<<8 | uint32(payload[4])
}

// IsInitial reports whether payload is a QUIC v1 Initial packet. This is the
// first packet of a QUIC handshake and the one carrying the (encrypted)
// ClientHello, so suppressing it is enough to defeat the QUIC path.
func IsInitial(payload []byte) bool {
	if !IsLongHeader(payload) {
		return false
	}
	if Version(payload) != versionV1 {
		return false
	}
	return (payload[0]&packetTypeMask)>>packetTypeShift == typeInitial
}
