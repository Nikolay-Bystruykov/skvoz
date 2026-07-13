// Package settings persists Skvoz's runtime state (which targets are enabled,
// the chosen desync strategy, and whether to auto-start) so the tray app
// remembers the user's choices between launches. State lives in a single JSON
// file under %LOCALAPPDATA%\Skvoz on Windows.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const fileName = "config.json"

// State is the persisted runtime configuration controlled from the tray.
type State struct {
	YouTube   bool   `json:"youtube"`
	Discord   bool   `json:"discord"`
	Strategy  string `json:"strategy"`
	FakeTTL   uint8  `json:"fake_ttl"`
	Autostart bool   `json:"autostart"`
}

// Default returns the state a fresh install starts from: both targets on, the
// recommended strategy, and autostart deliberately OFF (the user opts in).
func Default() State {
	return State{
		YouTube:   true,
		Discord:   true,
		Strategy:  "fakedsplit",
		FakeTTL:   8,
		Autostart: false,
	}
}

// Dir returns the per-user directory Skvoz stores its state in, creating it if
// necessary. On Windows os.UserConfigDir resolves to %AppData%; we prefer
// %LOCALAPPDATA% when available so the data is machine-local.
func Dir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	dir := filepath.Join(base, "Skvoz")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// Load reads the state file from dir. A missing file is not an error: it yields
// Default(), so first launch behaves as a fresh install.
func Load(dir string) (State, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return State{}, err
	}
	s := Default()
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

// Save writes the state to dir as pretty-printed JSON.
func (s State) Save(dir string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fileName), data, 0o644)
}
