package autopick

import (
	"context"
	"errors"
	"testing"
)

// fakeProber succeeds only when the currently-applied strategy matches want.
type fakeProber struct {
	current *Candidate
	want    Candidate
}

func (f *fakeProber) Probe(ctx context.Context, host string) error {
	if f.current != nil && *f.current == f.want {
		return nil
	}
	return errors.New("blocked")
}

func TestSelectPicksFirstWorkingCandidate(t *testing.T) {
	cands := []Candidate{
		{Strategy: "fakedsplit", FakeTTL: 8},
		{Strategy: "fake", FakeTTL: 6},
		{Strategy: "split", FakeTTL: 0},
	}
	var applied []Candidate
	var current Candidate
	fp := &fakeProber{current: &current, want: cands[1]}

	apply := func(c Candidate) error {
		applied = append(applied, c)
		current = c
		return nil
	}

	got, ok := Select(context.Background(), cands, "youtube.com", apply, fp)
	if !ok {
		t.Fatal("expected a working candidate")
	}
	if got != cands[1] {
		t.Fatalf("picked %+v, want %+v", got, cands[1])
	}
	// It should have tried the first (failed) then the second (worked), and
	// stopped there — never applying the third.
	if len(applied) != 2 || applied[0] != cands[0] || applied[1] != cands[1] {
		t.Fatalf("unexpected apply sequence: %+v", applied)
	}
}

func TestSelectReturnsFalseWhenNoneWork(t *testing.T) {
	cands := []Candidate{{Strategy: "fakedsplit", FakeTTL: 8}, {Strategy: "fake", FakeTTL: 6}}
	var current Candidate
	fp := &fakeProber{current: &current, want: Candidate{Strategy: "nonexistent"}}
	apply := func(c Candidate) error { current = c; return nil }

	_, ok := Select(context.Background(), cands, "youtube.com", apply, fp)
	if ok {
		t.Fatal("expected ok=false when no candidate works")
	}
}

func TestSelectSkipsCandidatesThatFailToApply(t *testing.T) {
	cands := []Candidate{{Strategy: "bad"}, {Strategy: "good"}}
	var current Candidate
	fp := &fakeProber{current: &current, want: cands[1]}
	apply := func(c Candidate) error {
		if c.Strategy == "bad" {
			return errors.New("apply failed")
		}
		current = c
		return nil
	}

	got, ok := Select(context.Background(), cands, "youtube.com", apply, fp)
	if !ok || got != cands[1] {
		t.Fatalf("expected to skip un-appliable candidate and pick good, got %+v ok=%v", got, ok)
	}
}

func TestCandidatesNonEmptyAndStartsWithRecommended(t *testing.T) {
	c := Candidates()
	if len(c) == 0 {
		t.Fatal("expected a non-empty candidate list")
	}
	if c[0].Strategy != "fakedsplit" {
		t.Fatalf("first candidate should be the recommended fakedsplit, got %q", c[0].Strategy)
	}
}
