package engine

import (
	"bytes"
	"testing"

	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/hostlist"
	"github.com/skvoz/skvoz/internal/packet"
	"github.com/skvoz/skvoz/internal/tls"
)

// tcpTo builds a serialized outbound IPv4/TCP packet to dstPort with payload.
func tcpTo(dstPort uint16, payload []byte) []byte {
	p := &packet.TCPPacket{
		TTL:     64,
		SrcIP:   [4]byte{10, 0, 0, 5},
		DstIP:   [4]byte{142, 250, 74, 78},
		SrcPort: 55555,
		DstPort: dstPort,
		Seq:     5000,
		Flags:   packet.FlagPSH | packet.FlagACK,
		Payload: payload,
	}
	return p.Serialize()
}

// udpQUIC builds a serialized outbound IPv4/UDP packet to dstPort carrying a
// QUIC v1 Initial payload.
func udpQUIC(dstPort uint16) []byte {
	quicInitial := []byte{0xC3, 0x00, 0x00, 0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF}
	total := 20 + 8 + len(quicInitial)
	buf := make([]byte, total)
	buf[0] = 0x45
	buf[2], buf[3] = byte(total>>8), byte(total)
	buf[8] = 64
	buf[9] = 17 // UDP
	copy(buf[12:16], []byte{10, 0, 0, 5})
	copy(buf[16:20], []byte{142, 250, 74, 78})
	buf[20], buf[21] = 0xD9, 0x00 // src port
	buf[22], buf[23] = byte(dstPort>>8), byte(dstPort)
	udpLen := 8 + len(quicInitial)
	buf[24], buf[25] = byte(udpLen>>8), byte(udpLen)
	copy(buf[28:], quicInitial)
	return buf
}

func newEngine(t *testing.T, cfg config.Config, domains ...string) *Engine {
	t.Helper()
	l := hostlist.New()
	for _, d := range domains {
		l.Add(d)
	}
	e, err := New(cfg, l, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestProcess_TargetClientHelloIsSplit(t *testing.T) {
	cfg := config.Default()
	cfg.Strategy = "split"
	e := newEngine(t, cfg, "youtube.com")

	hello := tls.BuildClientHello("www.youtube.com")
	pkt := divert.OutboundPacket(tcpTo(443, hello))

	out := e.Process(pkt)
	if len(out) != 2 {
		t.Fatalf("got %d packets, want 2 (split)", len(out))
	}

	// The two emitted segments must reassemble to the original ClientHello.
	var joined []byte
	for _, p := range out {
		tcp, err := packet.ParseTCP(p.Data)
		if err != nil {
			t.Fatalf("emitted packet does not parse: %v", err)
		}
		joined = append(joined, tcp.Payload...)
		if !p.Addr.Outbound {
			t.Error("emitted packet lost outbound addr")
		}
	}
	if !bytes.Equal(joined, hello) {
		t.Errorf("reassembled payload != original ClientHello")
	}
}

func TestProcess_NonTargetPassesThrough(t *testing.T) {
	e := newEngine(t, config.Default(), "youtube.com")

	hello := tls.BuildClientHello("www.example.com") // not targeted
	orig := tcpTo(443, hello)
	out := e.Process(divert.OutboundPacket(orig))

	if len(out) != 1 || !bytes.Equal(out[0].Data, orig) {
		t.Fatalf("non-target traffic must pass through unchanged")
	}
}

func TestProcess_NonClientHelloPassesThrough(t *testing.T) {
	e := newEngine(t, config.Default(), "youtube.com")
	// Application data to a targeted port but not a ClientHello.
	appData := []byte{0x17, 0x03, 0x03, 0x00, 0x05, 1, 2, 3, 4, 5}
	orig := tcpTo(443, appData)
	out := e.Process(divert.OutboundPacket(orig))
	if len(out) != 1 || !bytes.Equal(out[0].Data, orig) {
		t.Fatal("non-ClientHello must pass through unchanged")
	}
}

func TestProcess_NonTargetPortPassesThrough(t *testing.T) {
	e := newEngine(t, config.Default(), "youtube.com")
	hello := tls.BuildClientHello("www.youtube.com")
	orig := tcpTo(22, hello) // SSH port, not targeted
	out := e.Process(divert.OutboundPacket(orig))
	if len(out) != 1 || !bytes.Equal(out[0].Data, orig) {
		t.Fatal("non-target port must pass through unchanged")
	}
}

func TestProcess_QUICInitialDropped(t *testing.T) {
	e := newEngine(t, config.Default(), "youtube.com") // default QUIC=drop
	out := e.Process(divert.OutboundPacket(udpQUIC(443)))
	if len(out) != 0 {
		t.Fatalf("QUIC Initial to :443 must be dropped, got %d packets", len(out))
	}
}

func TestProcess_QUICKeptWhenOff(t *testing.T) {
	cfg := config.Default()
	cfg.QUIC = config.QUICOff
	e := newEngine(t, cfg, "youtube.com")
	q := udpQUIC(443)
	out := e.Process(divert.OutboundPacket(q))
	if len(out) != 1 || !bytes.Equal(out[0].Data, q) {
		t.Fatal("with quic=off the Initial must pass through")
	}
}

func TestProcess_InboundPassesThrough(t *testing.T) {
	e := newEngine(t, config.Default(), "youtube.com")
	hello := tls.BuildClientHello("www.youtube.com")
	pkt := &divert.Packet{Data: tcpTo(443, hello), Addr: divert.Addr{Outbound: false}}
	out := e.Process(pkt)
	if len(out) != 1 || !bytes.Equal(out[0].Data, pkt.Data) {
		t.Fatal("inbound packets must not be desynced")
	}
}

func TestRun_ProcessesAllAndStopsOnEOF(t *testing.T) {
	cfg := config.Default()
	cfg.Strategy = "split"
	e := newEngine(t, cfg, "youtube.com")

	hello := tls.BuildClientHello("www.youtube.com")
	src := divert.NewMockSource(
		divert.OutboundPacket(tcpTo(443, hello)),
		divert.OutboundPacket(udpQUIC(443)), // dropped
	)
	if err := e.Run(src); err == nil {
		t.Fatal("Run should return the source's EOF error")
	}
	// Two split segments from the ClientHello; QUIC produced nothing.
	if len(src.Sent) != 2 {
		t.Fatalf("sent %d packets, want 2", len(src.Sent))
	}
}
