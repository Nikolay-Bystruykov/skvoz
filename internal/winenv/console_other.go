//go:build !windows

package winenv

// AttachParentConsole is a Windows-only concern; a no-op elsewhere.
func AttachParentConsole() {}
