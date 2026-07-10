package desync

import (
	"bytes"
	"testing"

	"github.com/skvoz/skvoz/internal/packet"
)

func orig() *packet.TCPPacket {
	return &packet.TCPPacket{
		TTL:     64,
		SrcIP:   [4]byte{10, 0, 0, 2},
		DstIP:   [4]byte{142, 250, 74, 78},
		SrcPort: 50000,
		DstPort: 443,
		Seq:     1000,
		Flags:   packet.FlagPSH | packet.FlagACK,
		Payload: []byte("ABCDEFGHIJ"), // 10 bytes
	}
}

func TestSplit_ReassemblesToOriginal(t *testing.T) {
	s, _ := Get("split")
	out := s.Apply(orig(), Params{SplitPos: 4})
	if len(out) != 2 {
		t.Fatalf("got %d packets, want 2", len(out))
	}
	if !bytes.Equal(out[0].Payload, []byte("ABCD")) {
		t.Errorf("first payload = %q, want ABCD", out[0].Payload)
	}
	if !bytes.Equal(out[1].Payload, []byte("EFGHIJ")) {
		t.Errorf("second payload = %q, want EFGHIJ", out[1].Payload)
	}
	// Sequence numbers must be contiguous so the server reassembles correctly.
	if out[1].Seq != out[0].Seq+uint32(len(out[0].Payload)) {
		t.Errorf("second Seq = %d, want %d", out[1].Seq, out[0].Seq+4)
	}
	// First segment must not carry PSH; second must.
	if out[0].Flags&packet.FlagPSH != 0 {
		t.Error("first segment should not have PSH")
	}
	if out[1].Flags&packet.FlagPSH == 0 {
		t.Error("second segment should keep PSH")
	}
}

func TestDisorder_ReversesOrderButKeepsSeq(t *testing.T) {
	s, _ := Get("disorder")
	out := s.Apply(orig(), Params{SplitPos: 3})
	if len(out) != 2 {
		t.Fatalf("got %d packets, want 2", len(out))
	}
	// Second half is sent first, but with the higher sequence number.
	if !bytes.Equal(out[0].Payload, []byte("DEFGHIJ")) {
		t.Errorf("first-sent payload = %q, want DEFGHIJ", out[0].Payload)
	}
	if out[0].Seq != 1003 {
		t.Errorf("first-sent Seq = %d, want 1003", out[0].Seq)
	}
	if out[1].Seq != 1000 {
		t.Errorf("second-sent Seq = %d, want 1000", out[1].Seq)
	}
}

func TestFake_DecoyThenReal(t *testing.T) {
	s, _ := Get("fake")
	out := s.Apply(orig(), Params{FakeTTL: 5, FakePayload: []byte("XXXXXXXXXX")})
	if len(out) != 2 {
		t.Fatalf("got %d packets, want 2", len(out))
	}
	fk, real := out[0], out[1]
	if fk.TTL != 5 {
		t.Errorf("fake TTL = %d, want 5", fk.TTL)
	}
	if fk.Seq != real.Seq {
		t.Errorf("fake Seq %d must overlap real Seq %d", fk.Seq, real.Seq)
	}
	if bytes.Contains(fk.Payload, []byte("ABC")) {
		t.Error("fake payload must not leak real ClientHello bytes")
	}
	if !bytes.Equal(real.Payload, []byte("ABCDEFGHIJ")) {
		t.Errorf("real payload altered: %q", real.Payload)
	}
	if real.TTL != 64 {
		t.Errorf("real TTL = %d, want 64", real.TTL)
	}
}

func TestFake_DefaultTTLAndDecoy(t *testing.T) {
	s, _ := Get("fake")
	out := s.Apply(orig(), Params{})
	if out[0].TTL != DefaultFakeTTL {
		t.Errorf("default fake TTL = %d, want %d", out[0].TTL, DefaultFakeTTL)
	}
	if len(out[0].Payload) != len(orig().Payload) {
		t.Errorf("default decoy len = %d, want %d", len(out[0].Payload), len(orig().Payload))
	}
}

func TestFakedSplit_DecoyPlusTwoHalves(t *testing.T) {
	s, _ := Get("fakedsplit")
	out := s.Apply(orig(), Params{SplitPos: 4})
	if len(out) != 3 {
		t.Fatalf("got %d packets, want 3", len(out))
	}
	if out[0].TTL != DefaultFakeTTL {
		t.Error("first packet should be the low-TTL decoy")
	}
	if !bytes.Equal(out[1].Payload, []byte("ABCD")) || !bytes.Equal(out[2].Payload, []byte("EFGHIJ")) {
		t.Errorf("split halves wrong: %q %q", out[1].Payload, out[2].Payload)
	}
}

func TestSplitPos_Clamped(t *testing.T) {
	s, _ := Get("split")
	// Out-of-range split positions must not panic and must still produce two
	// non-empty segments.
	for _, pos := range []int{-5, 0, 1, 9, 10, 100} {
		out := s.Apply(orig(), Params{SplitPos: pos})
		if len(out) != 2 {
			t.Fatalf("pos %d: got %d packets", pos, len(out))
		}
		if len(out[0].Payload) == 0 || len(out[1].Payload) == 0 {
			t.Errorf("pos %d: produced an empty segment", pos)
		}
		joined := append(append([]byte{}, out[0].Payload...), out[1].Payload...)
		if !bytes.Equal(joined, orig().Payload) {
			t.Errorf("pos %d: halves do not reassemble: %q", pos, joined)
		}
	}
}

func TestGet_Unknown(t *testing.T) {
	if _, err := Get("nope"); err == nil {
		t.Error("expected error for unknown strategy")
	}
}
