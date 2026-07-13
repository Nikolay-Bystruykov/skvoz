package settings

import "testing"

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := State{YouTube: true, Discord: false, Strategy: "fake", FakeTTL: 6, Autostart: true}
	if err := in.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("round-trip mismatch: %+v != %+v", got, in)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != Default() {
		t.Fatalf("expected defaults for missing file, got %+v", got)
	}
}

func TestDefaultEnablesBothTargets(t *testing.T) {
	d := Default()
	if !d.YouTube || !d.Discord {
		t.Fatalf("both targets should be on by default: %+v", d)
	}
	if d.Autostart {
		t.Fatal("autostart must default to OFF")
	}
	if d.Strategy != "fakedsplit" {
		t.Fatalf("unexpected default strategy %q", d.Strategy)
	}
}
