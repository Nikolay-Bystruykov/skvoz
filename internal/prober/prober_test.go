package prober

import (
	"context"
	"testing"
	"time"

	"github.com/skvoz/skvoz/internal/autopick"
)

// TLSProber must satisfy the autopick.Prober interface.
var _ autopick.Prober = TLSProber{}

func TestProbeCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: no network needed, must fail fast
	if err := (TLSProber{Timeout: time.Second}).Probe(ctx, "example.com"); err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestProbeReachableHost(t *testing.T) {
	if testing.Short() {
		t.Skip("network test skipped in -short")
	}
	if err := (TLSProber{Timeout: 5 * time.Second}).Probe(context.Background(), "example.com"); err != nil {
		t.Fatalf("expected example.com reachable over TLS: %v", err)
	}
}
