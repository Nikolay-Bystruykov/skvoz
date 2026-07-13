//go:build !windows

package service

import (
	"errors"
	"os"
)

// Name is the service identifier (used in messages on all platforms).
const Name = "Skvoz"

var errUnsupported = errors.New("Windows services are only available on Windows")

// IsInteractive always reports true off Windows.
func IsInteractive() (bool, error) { return true, nil }

// IsElevated reports whether the process is running as root, which the
// packet-capture backend (BSD divert socket + pf) requires on this platform.
func IsElevated() bool { return os.Geteuid() == 0 }

// ElevationHint is shown to the user when IsElevated() is false.
func ElevationHint() string {
	return "run with root privileges, e.g.: sudo skvoz ..."
}

// Install is unsupported off Windows.
func Install(args []string) error { return errUnsupported }

// Uninstall is unsupported off Windows.
func Uninstall() error { return errUnsupported }

// RunService is unsupported off Windows.
func RunService(run func(stop <-chan struct{}) error) error { return errUnsupported }
