// Package winenv holds the Windows-specific environment glue Skvoz needs when
// it runs as a double-clicked tray app: extracting the embedded driver,
// relaunching elevated, and toggling autostart. The pieces that are pure file
// I/O live here (cross-platform, unit-tested); the Win32 calls live in
// *_windows.go files behind build tags.
package winenv

import (
	"bytes"
	"os"
	"path/filepath"
)

// ExtractDriver writes each named file into dir, creating dir if needed. A file
// whose contents already match is left untouched, so repeated launches don't
// churn the disk or fight a file lock on the loaded DLL. It returns the
// directory the files were written to.
func ExtractDriver(dir string, files map[string][]byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
	}
	return dir, nil
}
