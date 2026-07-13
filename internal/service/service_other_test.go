//go:build !windows

package service

import (
	"os"
	"testing"
)

func TestIsElevatedMatchesEUID(t *testing.T) {
	want := os.Geteuid() == 0
	if got := IsElevated(); got != want {
		t.Fatalf("IsElevated() = %v, want %v (euid=%d)", got, want, os.Geteuid())
	}
}

func TestElevationHintMentionsSudo(t *testing.T) {
	if got := ElevationHint(); got == "" {
		t.Fatal("ElevationHint() must not be empty")
	}
}
