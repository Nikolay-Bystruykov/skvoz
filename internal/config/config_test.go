package config

import (
	"io"
	"reflect"
	"testing"
)

func TestParse_Defaults(t *testing.T) {
	cfg, err := Parse(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Strategy != "fakedsplit" {
		t.Errorf("default strategy = %q", cfg.Strategy)
	}
	if cfg.QUIC != QUICDrop || cfg.SplitPos != SplitAtSNI {
		t.Errorf("unexpected defaults: quic=%q split=%q", cfg.QUIC, cfg.SplitPos)
	}
	if !reflect.DeepEqual(cfg.Ports, []uint16{80, 443}) {
		t.Errorf("default ports = %v", cfg.Ports)
	}
}

func TestParse_Flags(t *testing.T) {
	args := []string{
		"--strategy", "fake",
		"--fake-ttl", "6",
		"--lists", "a.txt, b.txt ,",
		"--quic", "off",
		"--split", "middle",
		"--ports", "443",
	}
	cfg, err := Parse(args, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Strategy != "fake" || cfg.FakeTTL != 6 {
		t.Errorf("strategy/ttl wrong: %q %d", cfg.Strategy, cfg.FakeTTL)
	}
	if !reflect.DeepEqual(cfg.Lists, []string{"a.txt", "b.txt"}) {
		t.Errorf("lists = %v", cfg.Lists)
	}
	if cfg.QUIC != QUICOff || cfg.SplitPos != SplitAtMiddle {
		t.Errorf("quic/split wrong: %q %q", cfg.QUIC, cfg.SplitPos)
	}
	if !reflect.DeepEqual(cfg.Ports, []uint16{443}) {
		t.Errorf("ports = %v", cfg.Ports)
	}
}

func TestFilter(t *testing.T) {
	cfg := Default() // ports 80,443, quic=drop
	got := cfg.Filter()
	want := "outbound and ((tcp and (tcp.DstPort == 80 or tcp.DstPort == 443)) or (udp and udp.DstPort == 443))"
	if got != want {
		t.Errorf("Filter() =\n  %q\nwant\n  %q", got, want)
	}

	cfg.QUIC = QUICOff
	cfg.Ports = []uint16{443}
	got = cfg.Filter()
	want = "outbound and (tcp and (tcp.DstPort == 443))"
	if got != want {
		t.Errorf("Filter(quic off) =\n  %q\nwant\n  %q", got, want)
	}
}

func TestParse_Invalid(t *testing.T) {
	bad := [][]string{
		{"--strategy", "bogus"},
		{"--quic", "maybe"},
		{"--split", "left"},
		{"--fake-ttl", "0"},
		{"--fake-ttl", "999"},
		{"--ports", "0"},
		{"--ports", "70000"},
		{"--ports", "abc"},
		{"--service", "restart"},
	}
	for _, args := range bad {
		if _, err := Parse(args, io.Discard); err == nil {
			t.Errorf("Parse(%v) = nil error, want error", args)
		}
	}
}
