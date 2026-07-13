//go:build windows

package divert

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WinDivert layer / flag constants (WinDivert 2.x).
const (
	layerNetwork  = 0
	priorityHi    = 0
	flagsNone     = 0
	maxPacketSize = 65535

	invalidHandle = ^uintptr(0)

	// Offset of the packed flags byte within WINDIVERT_ADDRESS and the
	// Outbound bit inside it.
	addrFlagsByte = 10
	outboundBit   = 0x02
)

var (
	dll               = windows.NewLazyDLL("WinDivert.dll")
	procOpen          = dll.NewProc("WinDivertOpen")
	procRecv          = dll.NewProc("WinDivertRecv")
	procSend          = dll.NewProc("WinDivertSend")
	procClose         = dll.NewProc("WinDivertClose")
	procCalcChecksums = dll.NewProc("WinDivertHelperCalcChecksums")
)

// SetDriverDir points the loader at the directory that holds the WinDivert
// binaries. The single-exe build extracts the embedded WinDivert.dll and
// WinDivert64.sys into a per-user folder at runtime and calls this so the DLL
// (and the .sys it loads next to itself) are found there. SetDllDirectory
// ensures WinDivert64.sys is located beside the DLL. Call before opening a
// source; the default (bare "WinDivert.dll" on the search path) is kept for the
// classic layout where the binaries sit next to skvoz.exe.
func SetDriverDir(dir string) {
	_ = windows.SetDllDirectory(dir)
	dll = windows.NewLazyDLL(filepath.Join(dir, "WinDivert.dll"))
	procOpen = dll.NewProc("WinDivertOpen")
	procRecv = dll.NewProc("WinDivertRecv")
	procSend = dll.NewProc("WinDivertSend")
	procClose = dll.NewProc("WinDivertClose")
	procCalcChecksums = dll.NewProc("WinDivertHelperCalcChecksums")
}

// winDivertSource is the production PacketSource backed by the WinDivert driver.
type winDivertSource struct {
	handle uintptr
	buf    []byte
}

// NewWinDivertSource opens a WinDivert handle on the NETWORK layer with the
// given filter. It requires administrator privileges and the WinDivert driver
// files (WinDivert.dll + WinDivert64.sys) alongside the executable.
func NewWinDivertSource(filter string) (PacketSource, error) {
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("cannot load WinDivert.dll (is it next to skvoz.exe?): %w", err)
	}
	fptr, err := windows.BytePtrFromString(filter)
	if err != nil {
		return nil, err
	}
	h, _, callErr := procOpen.Call(
		uintptr(unsafe.Pointer(fptr)),
		uintptr(layerNetwork),
		uintptr(priorityHi),
		uintptr(flagsNone),
	)
	if h == invalidHandle {
		return nil, fmt.Errorf("WinDivertOpen failed (run as administrator?): %w", callErr)
	}
	return &winDivertSource{handle: h, buf: make([]byte, maxPacketSize)}, nil
}

func (w *winDivertSource) Recv() (*Packet, error) {
	var recvLen uint32
	var addr Addr
	ok, _, callErr := procRecv.Call(
		w.handle,
		uintptr(unsafe.Pointer(&w.buf[0])),
		uintptr(len(w.buf)),
		uintptr(unsafe.Pointer(&recvLen)),
		uintptr(unsafe.Pointer(&addr.raw[0])),
	)
	if ok == 0 {
		return nil, fmt.Errorf("WinDivertRecv: %w", callErr)
	}
	addr.Outbound = addr.raw[addrFlagsByte]&outboundBit != 0
	data := make([]byte, recvLen)
	copy(data, w.buf[:recvLen])
	return &Packet{Data: data, Addr: addr}, nil
}

func (w *winDivertSource) Send(p *Packet) error {
	if len(p.Data) == 0 {
		return nil
	}
	addr := p.Addr // copy so we can hand a stable pointer and let the helper set flags
	// Recompute IP/TCP/UDP checksums on the (possibly modified) buffer and set
	// the address checksum flags accordingly, matching WinDivert's expectations
	// for re-injected packets.
	procCalcChecksums.Call(
		uintptr(unsafe.Pointer(&p.Data[0])),
		uintptr(len(p.Data)),
		uintptr(unsafe.Pointer(&addr.raw[0])),
		0,
	)
	var sentLen uint32
	ok, _, callErr := procSend.Call(
		w.handle,
		uintptr(unsafe.Pointer(&p.Data[0])),
		uintptr(len(p.Data)),
		uintptr(unsafe.Pointer(&sentLen)),
		uintptr(unsafe.Pointer(&addr.raw[0])),
	)
	if ok == 0 {
		return fmt.Errorf("WinDivertSend: %w", callErr)
	}
	return nil
}

func (w *winDivertSource) Close() error {
	if w.handle == 0 || w.handle == invalidHandle {
		return nil
	}
	procClose.Call(w.handle)
	w.handle = 0
	return nil
}
