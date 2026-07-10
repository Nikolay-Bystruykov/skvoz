package quic

import "testing"

func TestIsInitial(t *testing.T) {
	// Long header + fixed bit + type Initial (00) => 0xC0..0xC3; version 1.
	initial := []byte{0xC3, 0x00, 0x00, 0x00, 0x01, 0xAA, 0xBB}
	if !IsInitial(initial) {
		t.Error("expected QUIC v1 Initial to be detected")
	}
	if !IsLongHeader(initial) {
		t.Error("expected long header")
	}
	if Version(initial) != versionV1 {
		t.Errorf("Version = %#x, want 1", Version(initial))
	}
}

func TestIsInitial_Rejects(t *testing.T) {
	cases := map[string][]byte{
		"short header (0x40)":  {0x40, 0x00, 0x00, 0x00, 0x01},
		"handshake type (10)":  {0xE0, 0x00, 0x00, 0x00, 0x01}, // type bits = 10
		"version negotiation":  {0xC0, 0x00, 0x00, 0x00, 0x00},
		"too short":            {0xC0, 0x00},
		"not quic (TLS byte)":  {0x16, 0x03, 0x03, 0x00, 0x01},
	}
	for name, p := range cases {
		if IsInitial(p) {
			t.Errorf("%s: IsInitial = true, want false", name)
		}
	}
}
