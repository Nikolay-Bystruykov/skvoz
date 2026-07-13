package tray

import (
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/hostlist"
)

// blockingSource mimics the real WinDivert source: Recv blocks until Close, so
// the engine goroutine stays alive until the manager stops it.
type blockingSource struct {
	closed     chan struct{}
	mu         sync.Mutex
	closeCount int
}

func newBlockingSource() *blockingSource { return &blockingSource{closed: make(chan struct{})} }

func (b *blockingSource) Recv() (*divert.Packet, error) {
	<-b.closed
	return nil, io.EOF
}
func (b *blockingSource) Send(*divert.Packet) error { return nil }
func (b *blockingSource) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.closed:
	default:
		close(b.closed)
	}
	b.closeCount++
	return nil
}
func (b *blockingSource) CloseCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closeCount
}

func testLists() *hostlist.List {
	l := hostlist.New()
	l.Add("youtube.com")
	return l
}

func TestApplyStartsEngineWithExpectedFilter(t *testing.T) {
	var gotFilter string
	src := newBlockingSource()
	open := func(filter string) (divert.PacketSource, error) {
		gotFilter = filter
		return src, nil
	}
	m := New(open, log.New(io.Discard, "", 0))

	cfg := config.Default()
	if err := m.Apply(cfg, testLists()); err != nil {
		t.Fatal(err)
	}
	if gotFilter != cfg.Filter() {
		t.Fatalf("opened with filter %q, want %q", gotFilter, cfg.Filter())
	}
	if !m.Running() {
		t.Fatal("expected Running() true after Apply")
	}

	m.Stop()
	if m.Running() {
		t.Fatal("expected Running() false after Stop")
	}
	if src.CloseCount() == 0 {
		t.Fatal("source was not closed on Stop")
	}
}

func TestReapplyClosesPreviousSource(t *testing.T) {
	srcs := []*blockingSource{newBlockingSource(), newBlockingSource()}
	var i int
	open := func(string) (divert.PacketSource, error) {
		s := srcs[i]
		i++
		return s, nil
	}
	m := New(open, log.New(io.Discard, "", 0))

	if err := m.Apply(config.Default(), testLists()); err != nil {
		t.Fatal(err)
	}
	if err := m.Apply(config.Default(), testLists()); err != nil {
		t.Fatal(err)
	}
	// Give the first engine goroutine a moment to unwind after its source closed.
	time.Sleep(50 * time.Millisecond)
	if srcs[0].CloseCount() == 0 {
		t.Fatal("first source should be closed when Apply is called again")
	}
	m.Stop()
}
