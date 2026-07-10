//go:build !windows

package divert

import "errors"

// NewWinDivertSource is a stub on non-Windows platforms so the rest of the
// program still builds and its logic can be tested. The real implementation
// lives in windivert_windows.go.
func NewWinDivertSource(filter string) (PacketSource, error) {
	return nil, errors.New("WinDivert packet capture is only available on Windows")
}
