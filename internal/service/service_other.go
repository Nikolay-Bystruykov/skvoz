//go:build !windows

package service

import "errors"

// Name is the service identifier (used in messages on all platforms).
const Name = "Skvoz"

var errUnsupported = errors.New("Windows services are only available on Windows")

// IsInteractive always reports true off Windows.
func IsInteractive() (bool, error) { return true, nil }

// IsElevated is a no-op that reports true off Windows (WinDivert itself is
// unavailable there, so the caller fails later with a clear message).
func IsElevated() bool { return true }

// Install is unsupported off Windows.
func Install(args []string) error { return errUnsupported }

// Uninstall is unsupported off Windows.
func Uninstall() error { return errUnsupported }

// RunService is unsupported off Windows.
func RunService(run func(stop <-chan struct{}) error) error { return errUnsupported }
