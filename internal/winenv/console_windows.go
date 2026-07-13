//go:build windows

package winenv

import "golang.org/x/sys/windows"

// AttachParentConsole reattaches stdout/stderr to the console of the parent
// process. Skvoz is linked as a GUI ("windowsgui") binary so that a
// double-click shows no console window; that also detaches CLI output. Calling
// this at the start of command-line mode restores output when skvoz.exe is run
// from an existing terminal. It is a harmless no-op when there is no parent
// console (e.g. launched from Explorer).
func AttachParentConsole() {
	const attachParentProcess = ^uintptr(0) // (DWORD)-1
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	if err := kernel32.Load(); err != nil {
		return
	}
	proc := kernel32.NewProc("AttachConsole")
	proc.Call(attachParentProcess)
}
