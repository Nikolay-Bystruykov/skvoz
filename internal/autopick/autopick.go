// Package autopick chooses, at startup, the first desync strategy that actually
// restores access to a target host. Providers block differently and change over
// time, so rather than hard-coding one strategy Skvoz probes several and keeps
// the first that works. Everything here is transport-agnostic and unit-tested;
// the real network probe is supplied by internal/prober.
package autopick

import "context"

// Candidate is one desync configuration to try.
type Candidate struct {
	Strategy string
	FakeTTL  uint8
}

// Candidates returns the strategies to try, in preference order: the
// recommended fakedsplit first, then fakes with lower TTLs, then the simpler
// split/disorder variants as fallbacks.
func Candidates() []Candidate {
	return []Candidate{
		{Strategy: "fakedsplit", FakeTTL: 8},
		{Strategy: "fake", FakeTTL: 6},
		{Strategy: "fake", FakeTTL: 3},
		{Strategy: "split", FakeTTL: 0},
		{Strategy: "disorder", FakeTTL: 0},
	}
}

// PreferredFirst reorders all so preferred is tried first, ahead of the usual
// sweep. The caller uses this to retry the last strategy known to work before
// falling back to a full re-sweep: probing every candidate from scratch on
// every reconfiguration (e.g. a target checkbox toggle) closes and reopens the
// packet-capture handle once per candidate, which briefly disrupts any
// in-flight connections — disruptive enough to stall a video mid-load. If
// preferred already restores access, that churn is skipped entirely.
func PreferredFirst(preferred Candidate, all []Candidate) []Candidate {
	out := make([]Candidate, 0, len(all)+1)
	out = append(out, preferred)
	for _, c := range all {
		if c != preferred {
			out = append(out, c)
		}
	}
	return out
}

// Prober reports whether host is reachable under whatever strategy is currently
// applied. A nil error means the connection (e.g. a TLS handshake) succeeded.
type Prober interface {
	Probe(ctx context.Context, host string) error
}

// Select applies each candidate via apply and probes host; it returns the first
// candidate that both applies and probes cleanly. If apply fails for a
// candidate it is skipped. ok is false when none succeed or ctx is cancelled.
func Select(ctx context.Context, cands []Candidate, host string, apply func(Candidate) error, p Prober) (Candidate, bool) {
	for _, c := range cands {
		if ctx.Err() != nil {
			return Candidate{}, false
		}
		if err := apply(c); err != nil {
			continue
		}
		if err := p.Probe(ctx, host); err == nil {
			return c, true
		}
	}
	return Candidate{}, false
}
