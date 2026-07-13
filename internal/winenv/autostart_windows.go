//go:build windows

package winenv

import (
	"os"
	"os/exec"
)

// EnableAutostart registers the logon Scheduled Task so Skvoz starts elevated
// with Windows. exe should be the absolute path to skvoz.exe.
func EnableAutostart(exe string) error {
	return runSchtasks(autostartArgs("create", exe))
}

// DisableAutostart removes the Scheduled Task.
func DisableAutostart() error {
	return runSchtasks(autostartArgs("delete", ""))
}

// IsAutostartEnabled reports whether the Scheduled Task currently exists.
func IsAutostartEnabled() bool {
	return runSchtasks(autostartArgs("query", "")) == nil
}

func runSchtasks(args []string) error {
	cmd := exec.Command("schtasks", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
