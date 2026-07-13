// Package tray hosts the system-tray GUI and the engine-lifecycle glue behind
// it. Manager lives here (rather than in the GUI file) so the start/stop/
// reconfigure logic is cross-platform and unit-tested, while the Win32 systray
// wiring stays isolated in tray_windows.go.
package tray

import (
	"log"
	"sync"

	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/engine"
	"github.com/skvoz/skvoz/internal/hostlist"
)

// SourceOpener opens a packet source for the given WinDivert filter. Production
// passes divert.NewWinDivertSource; tests pass a fake.
type SourceOpener func(filter string) (divert.PacketSource, error)

// Manager owns the currently-running engine and lets the tray (re)start it with
// a new configuration or domain set. All methods are safe for concurrent use.
type Manager struct {
	open SourceOpener
	log  *log.Logger

	mu      sync.Mutex
	src     divert.PacketSource
	running bool
}

// New returns a Manager that opens sources via open.
func New(open SourceOpener, logger *log.Logger) *Manager {
	return &Manager{open: open, log: logger}
}

// Apply (re)starts the engine with cfg matching against lists. Any previously
// running engine is stopped first. It returns an error if the engine or source
// cannot be created, in which case nothing is left running.
func (m *Manager) Apply(cfg config.Config, lists *hostlist.List) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopLocked()

	eng, err := engine.New(cfg, lists, m.log)
	if err != nil {
		return err
	}
	src, err := m.open(cfg.Filter())
	if err != nil {
		return err
	}
	m.src = src
	m.running = true
	go func() {
		// Run blocks until the source is closed (by stopLocked) or errors.
		if err := eng.Run(src); err != nil {
			m.log.Printf("engine stopped: %v", err)
		}
	}()
	return nil
}

// Stop halts the running engine, if any.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

// Running reports whether an engine is currently active.
func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// stopLocked closes the current source (unblocking Run) and clears state. The
// caller must hold m.mu.
func (m *Manager) stopLocked() {
	if m.src != nil {
		_ = m.src.Close()
		m.src = nil
	}
	m.running = false
}
