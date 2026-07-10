package packet

import (
	"bytes"
	"testing"
)

func samplePacket(payload []byte) *TCPPacket {
	return &TCPPacket{
		TOS:     0,
		ID:      0x1234,
		IPFlags: 0x4000, // DF
		TTL:     64,
		SrcIP:   [4]byte{192, 168, 1, 10},
		DstIP:   [4]byte{142, 250, 74, 78},
		SrcPort: 51000,
		DstPort: 443,
		Seq:     0xdeadbeef,
		Ack:     0x11223344,
		Flags:   FlagPSH | FlagACK,
		Window:  64240,
		Payload: payload,
	}
}

func TestSerializeParse_RoundTrip(t *testing.T) {
	orig := samplePacket([]byte("hello world payload"))
	raw := orig.Serialize()

	got, err := ParseTCP(raw)
	if err != nil {
		t.Fatalf("ParseTCP: %v", err)
	}
	if got.SrcPort != orig.SrcPort || got.DstPort != orig.DstPort {
		t.Errorf("ports = %d/%d, want %d/%d", got.SrcPort, got.DstPort, orig.SrcPort, orig.DstPort)
	}
	if got.Seq != orig.Seq || got.Ack != orig.Ack {
		t.Errorf("seq/ack mismatch")
	}
	if got.TTL != orig.TTL || got.Flags != orig.Flags || got.Window != orig.Window {
		t.Errorf("ttl/flags/window mismatch")
	}
	if got.SrcIP != orig.SrcIP || got.DstIP != orig.DstIP {
		t.Errorf("ip mismatch")
	}
	if !bytes.Equal(got.Payload, orig.Payload) {
		t.Errorf("payload = %q, want %q", got.Payload, orig.Payload)
	}
}

func TestSerialize_ChecksumsValid(t *testing.T) {
	raw := samplePacket([]byte("check my sums")).Serialize()

	ihl := int(raw[0]&0x0f) * 4
	if c := checksum(raw[:ihl]); c != 0 {
		t.Errorf("IPv4 header checksum does not validate: got %#04x, want 0", c)
	}

	var src, dst [4]byte
	copy(src[:], raw[12:16])
	copy(dst[:], raw[16:20])
	if c := verifyTCPChecksum(src, dst, raw[ihl:]); c != 0 {
		t.Errorf("TCP checksum does not validate: got %#04x, want 0", c)
	}
}

// verifyTCPChecksum recomputes the checksum over the pseudo-header + segment
// WITHOUT zeroing the checksum field; a valid packet yields 0.
func verifyTCPChecksum(src, dst [4]byte, tcp []byte) uint16 {
	pseudo := make([]byte, 12+len(tcp))
	copy(pseudo[0:4], src[:])
	copy(pseudo[4:8], dst[:])
	pseudo[9] = protoTCP
	pseudo[10] = byte(len(tcp) >> 8)
	pseudo[11] = byte(len(tcp))
	copy(pseudo[12:], tcp)
	return checksum(pseudo)
}

func TestParseTCP_Rejects(t *testing.T) {
	if _, err := ParseTCP([]byte{0x60, 0, 0, 0}); err == nil {
		t.Error("expected error for IPv6/short buffer")
	}
	// UDP packet must be rejected by ParseTCP.
	udp := make([]byte, 28)
	udp[0] = 0x45
	udp[9] = protoUDP
	if _, err := ParseTCP(udp); err != ErrNotTCP {
		t.Errorf("err = %v, want ErrNotTCP", err)
	}
}

func TestParseUDP(t *testing.T) {
	buf := make([]byte, 20+8+5)
	buf[0] = 0x45
	buf[2] = byte(len(buf) >> 8)
	buf[3] = byte(len(buf))
	buf[9] = protoUDP
	// UDP header at offset 20
	buf[20], buf[21] = 0xC0, 0x00 // src port 49152
	buf[22], buf[23] = 0x01, 0xBB // dst port 443
	udpLen := 8 + 5
	buf[24], buf[25] = byte(udpLen>>8), byte(udpLen)
	copy(buf[28:], []byte("quic!"))

	info, err := ParseUDP(buf)
	if err != nil {
		t.Fatal(err)
	}
	if info.DstPort != 443 {
		t.Errorf("DstPort = %d, want 443", info.DstPort)
	}
	if !bytes.Equal(info.Payload, []byte("quic!")) {
		t.Errorf("payload = %q", info.Payload)
	}
}
