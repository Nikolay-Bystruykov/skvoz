//go:build !windows

package tray

import "log"

// Run is the tray entry point. The GUI is Windows-only; on other platforms it
// just reports that, so the package builds and its logic is unit-testable
// everywhere.
func Run(logger *log.Logger) {
	logger.Printf("Skvoz tray GUI is only supported on Windows")
}
