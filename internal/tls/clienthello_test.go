package tls

import (
	"testing"
)

// buildClientHello is a thin alias so existing tests keep reading naturally.
func buildClientHello(sni string) []byte { return BuildClientHello(sni) }

func TestParseClientHello_SNI(t *testing.T) {
	for _, sni := range []string{"www.youtube.com", "discord.gg", "a.b.c.example.org"} {
		payload := buildClientHello(sni)
		info, err := ParseClientHello(payload)
		if err != nil {
			t.Fatalf("ParseClientHello(%q) error: %v", sni, err)
		}
		if info.SNI != sni {
			t.Errorf("SNI = %q, want %q", info.SNI, sni)
		}
		if got := string(payload[info.SNIOffset : info.SNIOffset+info.SNILength]); got != sni {
			t.Errorf("payload at SNIOffset = %q, want %q", got, sni)
		}
		if info.RecordLength != len(payload) {
			t.Errorf("RecordLength = %d, want %d", info.RecordLength, len(payload))
		}
	}
}

func TestParseClientHello_NoSNI(t *testing.T) {
	payload := buildClientHello("")
	if _, err := ParseClientHello(payload); err != ErrNoSNI {
		t.Errorf("err = %v, want ErrNoSNI", err)
	}
}

func TestParseClientHello_NotHandshake(t *testing.T) {
	// Application data record, not a handshake.
	payload := []byte{0x17, 0x03, 0x03, 0x00, 0x01, 0xff}
	if _, err := ParseClientHello(payload); err != ErrNotClientHello {
		t.Errorf("err = %v, want ErrNotClientHello", err)
	}
	if IsClientHello(payload) {
		t.Error("IsClientHello = true, want false")
	}
}

func TestParseClientHello_Truncated(t *testing.T) {
	full := buildClientHello("www.youtube.com")
	// Truncate mid-extensions; parser must not panic and must error cleanly.
	for n := 0; n < len(full); n++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on truncated len %d: %v", n, r)
				}
			}()
			_, _ = ParseClientHello(full[:n])
		}()
	}
}
