// Package divert is the packet interception boundary. The PacketSource
// interface lets the engine run against a real WinDivert handle in production
// and against an in-memory mock in tests, so the engine's logic is fully
// testable on any OS.
package divert

// addrSize is the size in bytes of a WINDIVERT_ADDRESS (WinDivert 2.x). The
// real Windows implementation stores the opaque address here so a re-injected
// packet keeps the interface/direction metadata of the packet it replaces.
const addrSize = 64

// Addr carries the metadata needed to re-inject a packet on the same path it
// was captured from. It is copied by value so transformed packets can reuse the
// original's address.
type Addr struct {
	Outbound bool
	raw      [addrSize]byte
}

// Packet is a captured or to-be-injected network packet.
type Packet struct {
	Data []byte
	Addr Addr
}

// PacketSource captures outbound packets and re-injects them.
type PacketSource interface {
	// Recv returns the next captured packet. It blocks until one is available.
	Recv() (*Packet, error)
	// Send injects a packet. For a captured packet, this forwards it; for a
	// crafted packet it originates it on the same path.
	Send(*Packet) error
	// Close releases the underlying handle.
	Close() error
}
