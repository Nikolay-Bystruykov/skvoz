package winenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractDriverWritesFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Skvoz")
	files := map[string][]byte{"WinDivert.dll": []byte("dll"), "WinDivert64.sys": []byte("sys")}

	got, err := ExtractDriver(dir, files)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("returned dir %q, want %q", got, dir)
	}
	for name, want := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if string(data) != string(want) {
			t.Fatalf("%s = %q, want %q", name, data, want)
		}
	}
}

func TestExtractDriverIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{"WinDivert.dll": []byte("dll")}
	if _, err := ExtractDriver(dir, files); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "WinDivert.dll")
	info1, _ := os.Stat(path)

	// Second extraction with identical content must not rewrite the file.
	if _, err := ExtractDriver(dir, files); err != nil {
		t.Fatal(err)
	}
	info2, _ := os.Stat(path)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatal("identical file was rewritten; extraction is not idempotent")
	}
}
