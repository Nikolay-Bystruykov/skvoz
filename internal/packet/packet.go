// Package packet parses and rebuilds IPv4 TCP and UDP packets. It is the
// low-level substrate the desync strategies operate on: decode an intercepted
// packet, produce modified variants, and re-serialize them with correct
// lengths and checksums.
//
// v1 handles IPv4 only. IPv6 support is deferred (see design spec); callers
// treat a parse error as "pass through unchanged".
package packet

import "errors"

// TCP flag bits.
const (
	FlagFIN = 0x01
	FlagSYN = 0x02
	FlagRST = 0x04
	FlagPSH = 0x08
	FlagACK = 0x10
	FlagURG = 0x20
)

const (
	protoTCP = 6
	protoUDP = 17
)

var (
	// ErrNotIPv4 means the buffer is not an IPv4 packet.
	ErrNotIPv4 = errors.New("packet: not IPv4")
	// ErrNotTCP means the IPv4 packet does not carry TCP.
	ErrNotTCP = errors.New("packet: not TCP")
	// ErrNotUDP means the IPv4 packet does not carry UDP.
	ErrNotUDP = errors.New("packet: not UDP")
	// ErrTooShort means the buffer is shorter than its headers claim.
	ErrTooShort = errors.New("packet: truncated")
)

// TCPPacket is a decoded IPv4 + TCP packet. Strategies copy it (via Clone),
// adjust Seq/Payload/TTL and re-serialize.
type TCPPacket struct {
	// IPv4 fields
	TOS     uint8
	ID      uint16
	IPFlags uint16 // 3-bit flags + 13-bit fragment offset (e.g. 0x4000 = DF)
	TTL     uint8
	SrcIP   [4]byte
	DstIP   [4]byte
	IPOpts  []byte // IP options (usually empty)

	// TCP fields
	SrcPort uint16
	DstPort uint16
	Seq     uint32
	Ack     uint32
	Flags   uint8
	Window  uint16
	Urgent  uint16
	TCPOpts []byte // TCP options (already padded to a 4-byte boundary)
	Payload []byte
}

// ParseTCP decodes an IPv4 TCP packet. The returned struct references freshly
// copied byte slices, so mutating the input buffer afterwards is safe.
func ParseTCP(buf []byte) (*TCPPacket, error) {
	if len(buf) < 20 {
		return nil, ErrTooShort
	}
	if buf[0]>>4 != 4 {
		return nil, ErrNotIPv4
	}
	ihl := int(buf[0]&0x0f) * 4
	if ihl < 20 || len(buf) < ihl {
		return nil, ErrTooShort
	}
	if buf[9] != protoTCP {
		return nil, ErrNotTCP
	}
	totalLen := int(buf[2])<<8 | int(buf[3])
	if totalLen < ihl || totalLen > len(buf) {
		// Trust the buffer length if the header lies (some capture paths pad).
		totalLen = len(buf)
	}
	tcp := buf[ihl:totalLen]
	if len(tcp) < 20 {
		return nil, ErrTooShort
	}
	dataOff := int(tcp[12]>>4) * 4
	if dataOff < 20 || dataOff > len(tcp) {
		return nil, ErrTooShort
	}

	p := &TCPPacket{
		TOS:     buf[1],
		ID:      uint16(buf[4])<<8 | uint16(buf[5]),
		IPFlags: uint16(buf[6])<<8 | uint16(buf[7]),
		TTL:     buf[8],
		SrcPort: uint16(tcp[0])<<8 | uint16(tcp[1]),
		DstPort: uint16(tcp[2])<<8 | uint16(tcp[3]),
		Seq:     uint32(tcp[4])<<24 | uint32(tcp[5])<<16 | uint32(tcp[6])<<8 | uint32(tcp[7]),
		Ack:     uint32(tcp[8])<<24 | uint32(tcp[9])<<16 | uint32(tcp[10])<<8 | uint32(tcp[11]),
		Flags:   tcp[13],
		Window:  uint16(tcp[14])<<8 | uint16(tcp[15]),
		Urgent:  uint16(tcp[18])<<8 | uint16(tcp[19]),
	}
	copy(p.SrcIP[:], buf[12:16])
	copy(p.DstIP[:], buf[16:20])
	p.IPOpts = cloneBytes(buf[20:ihl])
	p.TCPOpts = cloneBytes(tcp[20:dataOff])
	p.Payload = cloneBytes(tcp[dataOff:])
	return p, nil
}

// Clone returns a deep copy safe to mutate independently.
func (p *TCPPacket) Clone() *TCPPacket {
	c := *p
	c.IPOpts = cloneBytes(p.IPOpts)
	c.TCPOpts = cloneBytes(p.TCPOpts)
	c.Payload = cloneBytes(p.Payload)
	return &c
}

// Serialize builds a complete IPv4 TCP packet with correct total length,
// data offset and both checksums.
func (p *TCPPacket) Serialize() []byte {
	ihl := 20 + len(p.IPOpts)
	tcpHdr := 20 + len(p.TCPOpts)
	total := ihl + tcpHdr + len(p.Payload)
	buf := make([]byte, total)

	// IPv4 header
	buf[0] = 4<<4 | byte(ihl/4)
	buf[1] = p.TOS
	buf[2] = byte(total >> 8)
	buf[3] = byte(total)
	buf[4] = byte(p.ID >> 8)
	buf[5] = byte(p.ID)
	buf[6] = byte(p.IPFlags >> 8)
	buf[7] = byte(p.IPFlags)
	buf[8] = p.TTL
	buf[9] = protoTCP
	// checksum bytes 10-11 left zero for now
	copy(buf[12:16], p.SrcIP[:])
	copy(buf[16:20], p.DstIP[:])
	copy(buf[20:ihl], p.IPOpts)
	ipChk := checksum(buf[:ihl])
	buf[10] = byte(ipChk >> 8)
	buf[11] = byte(ipChk)

	// TCP header
	tcp := buf[ihl:]
	tcp[0] = byte(p.SrcPort >> 8)
	tcp[1] = byte(p.SrcPort)
	tcp[2] = byte(p.DstPort >> 8)
	tcp[3] = byte(p.DstPort)
	tcp[4] = byte(p.Seq >> 24)
	tcp[5] = byte(p.Seq >> 16)
	tcp[6] = byte(p.Seq >> 8)
	tcp[7] = byte(p.Seq)
	tcp[8] = byte(p.Ack >> 24)
	tcp[9] = byte(p.Ack >> 16)
	tcp[10] = byte(p.Ack >> 8)
	tcp[11] = byte(p.Ack)
	tcp[12] = byte(tcpHdr/4) << 4
	tcp[13] = p.Flags
	tcp[14] = byte(p.Window >> 8)
	tcp[15] = byte(p.Window)
	tcp[18] = byte(p.Urgent >> 8)
	tcp[19] = byte(p.Urgent)
	copy(tcp[20:tcpHdr], p.TCPOpts)
	copy(tcp[tcpHdr:], p.Payload)

	tcpChk := tcpChecksum(p.SrcIP, p.DstIP, tcp)
	tcp[16] = byte(tcpChk >> 8)
	tcp[17] = byte(tcpChk)
	return buf
}

// UDPInfo is a minimal decode of an IPv4 UDP packet, enough to identify QUIC.
type UDPInfo struct {
	SrcPort uint16
	DstPort uint16
	Payload []byte
}

// ParseUDP decodes an IPv4 UDP packet.
func ParseUDP(buf []byte) (*UDPInfo, error) {
	if len(buf) < 20 {
		return nil, ErrTooShort
	}
	if buf[0]>>4 != 4 {
		return nil, ErrNotIPv4
	}
	ihl := int(buf[0]&0x0f) * 4
	if ihl < 20 || len(buf) < ihl+8 {
		return nil, ErrTooShort
	}
	if buf[9] != protoUDP {
		return nil, ErrNotUDP
	}
	udp := buf[ihl:]
	length := int(udp[4])<<8 | int(udp[5])
	end := ihl + length
	if length < 8 || end > len(buf) {
		end = len(buf)
	}
	return &UDPInfo{
		SrcPort: uint16(udp[0])<<8 | uint16(udp[1]),
		DstPort: uint16(udp[2])<<8 | uint16(udp[3]),
		Payload: cloneBytes(buf[ihl+8 : end]),
	}, nil
}

func cloneBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// checksum computes the 16-bit one's-complement sum used by IPv4/TCP/UDP.
func checksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(b); i += 2 {
		sum += uint32(b[i])<<8 | uint32(b[i+1])
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

// tcpChecksum computes the TCP checksum over the pseudo-header + segment.
// The segment's own checksum field must be zero on entry.
func tcpChecksum(src, dst [4]byte, tcp []byte) uint16 {
	pseudo := make([]byte, 12+len(tcp))
	copy(pseudo[0:4], src[:])
	copy(pseudo[4:8], dst[:])
	pseudo[8] = 0
	pseudo[9] = protoTCP
	pseudo[10] = byte(len(tcp) >> 8)
	pseudo[11] = byte(len(tcp))
	copy(pseudo[12:], tcp)
	// Ensure checksum field is zero before summing.
	pseudo[12+16] = 0
	pseudo[12+17] = 0
	return checksum(pseudo)
}
