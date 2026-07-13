//go:build !windows

package winenv

// RelaunchElevated is a no-op off Windows: there is no UAC to satisfy.
func RelaunchElevated() error { return nil }

// EnsureElevated reports that no relaunch happened, so callers proceed.
func EnsureElevated() (relaunched bool, err error) { return false, nil }
