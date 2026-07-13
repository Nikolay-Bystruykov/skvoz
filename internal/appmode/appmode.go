// Package appmode decides whether Skvoz was launched to show its tray GUI or to
// run in command-line mode. A user who double-clicks the exe passes no
// arguments, so an empty argument list means "show the GUI"; any flag (CLI use
// or the Windows service) means "run headless".
package appmode

// IsGUI reports whether Skvoz should start its tray GUI. args is os.Args[1:].
func IsGUI(args []string) bool {
	return len(args) == 0
}
