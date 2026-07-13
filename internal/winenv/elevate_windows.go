//go:build windows

package winenv

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"

	"github.com/skvoz/skvoz/internal/service"
)

// RelaunchElevated starts a new instance of the current executable with the
// "runas" verb, triggering the UAC prompt. It passes no arguments, so the new
// instance launches in tray-GUI mode just like the original double-click.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	cwd, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, cwd, windows.SW_NORMAL)
}

// EnsureElevated returns relaunched=true when it has spawned an elevated copy
// and the caller should exit. When Skvoz is already elevated it returns
// (false, nil) and the caller continues.
func EnsureElevated() (relaunched bool, err error) {
	if service.IsElevated() {
		return false, nil
	}
	if err := RelaunchElevated(); err != nil {
		return false, err
	}
	return true, nil
}
