//go:build !windows

package winenv

// Autostart is a Windows-only feature; these are no-ops elsewhere so the tray
// package compiles and unit-tests run on any OS.

func EnableAutostart(exe string) error { return nil }

func DisableAutostart() error { return nil }

func IsAutostartEnabled() bool { return false }
