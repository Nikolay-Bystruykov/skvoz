// Package desync implements the DPI-desynchronization strategies. Each strategy
// is a pure transformation: given the intercepted TCP segment that carries a
// TLS ClientHello, it returns the sequence of packets to inject in its place.
//
// The strategies mirror the well-known zapret techniques:
//
//	split       - cut the segment in two at SplitPos so DPI cannot reassemble
//	              the host name.
//	disorder    - send those two halves in reverse order.
//	fake        - send a decoy ClientHello (low TTL, dies before the server but
//	              is seen by DPI), then the real segment.
//	fakedsplit  - decoy first, then the real segment split in two.
package desync

import (
	"fmt"

	"github.com/skvoz/skvoz/internal/packet"
)

// DefaultFakeTTL is the TTL applied to decoy packets: large enough to reach the
// DPI middlebox, small enough to expire before the real server.
const DefaultFakeTTL = 8

// Params controls how a strategy rewrites a segment.
type Params struct {
	// SplitPos is the byte offset within the segment payload to split at.
	// It is clamped into [1, len(payload)-1].
	SplitPos int
	// FakeTTL is the TTL for decoy packets. Zero means DefaultFakeTTL.
	FakeTTL uint8
	// FakePayload is the decoy application data sent by fake/fakedsplit. If
	// empty, a zero-filled buffer the size of the real payload is used.
	FakePayload []byte
}

func (p Params) fakeTTL() uint8 {
	if p.FakeTTL == 0 {
		return DefaultFakeTTL
	}
	return p.FakeTTL
}

// Strategy rewrites a ClientHello-bearing segment into the packets to inject.
type Strategy interface {
	Name() string
	Apply(orig *packet.TCPPacket, p Params) []*packet.TCPPacket
}

// Get returns the strategy registered under name, or an error.
func Get(name string) (Strategy, error) {
	if s, ok := registry[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("desync: unknown strategy %q", name)
}

// Names lists the available strategy names.
func Names() []string {
	return []string{"split", "disorder", "fake", "fakedsplit"}
}

var registry = map[string]Strategy{
	"split":      split{},
	"disorder":   disorder{},
	"fake":       fake{},
	"fakedsplit": fakedSplit{},
}

// splitAt cuts orig into two consecutive segments at pos (clamped). The first
// segment loses its PSH flag so the receiver waits for more data; the second
// carries the adjusted sequence number.
func splitAt(orig *packet.TCPPacket, pos int) (first, second *packet.TCPPacket) {
	pos = clamp(pos, 1, len(orig.Payload)-1)

	first = orig.Clone()
	first.Payload = clone(orig.Payload[:pos])
	first.Flags = orig.Flags &^ packet.FlagPSH

	second = orig.Clone()
	second.Payload = clone(orig.Payload[pos:])
	second.Seq = orig.Seq + uint32(pos)
	return first, second
}

// makeFake builds a decoy segment: same sequence position as the real one so
// DPI associates it with the connection, but a low TTL so it never reaches the
// server, and decoy payload so the real SNI is never exposed in it.
func makeFake(orig *packet.TCPPacket, p Params) *packet.TCPPacket {
	f := orig.Clone()
	f.TTL = p.fakeTTL()
	if len(p.FakePayload) > 0 {
		f.Payload = clone(p.FakePayload)
	} else {
		f.Payload = make([]byte, len(orig.Payload)) // zero-filled decoy
	}
	return f
}

type split struct{}

func (split) Name() string { return "split" }
func (split) Apply(orig *packet.TCPPacket, p Params) []*packet.TCPPacket {
	if len(orig.Payload) < 2 {
		return []*packet.TCPPacket{orig.Clone()}
	}
	a, b := splitAt(orig, p.SplitPos)
	return []*packet.TCPPacket{a, b}
}

type disorder struct{}

func (disorder) Name() string { return "disorder" }
func (disorder) Apply(orig *packet.TCPPacket, p Params) []*packet.TCPPacket {
	if len(orig.Payload) < 2 {
		return []*packet.TCPPacket{orig.Clone()}
	}
	a, b := splitAt(orig, p.SplitPos)
	return []*packet.TCPPacket{b, a} // second half first
}

type fake struct{}

func (fake) Name() string { return "fake" }
func (fake) Apply(orig *packet.TCPPacket, p Params) []*packet.TCPPacket {
	return []*packet.TCPPacket{makeFake(orig, p), orig.Clone()}
}

type fakedSplit struct{}

func (fakedSplit) Name() string { return "fakedsplit" }
func (fakedSplit) Apply(orig *packet.TCPPacket, p Params) []*packet.TCPPacket {
	if len(orig.Payload) < 2 {
		return []*packet.TCPPacket{makeFake(orig, p), orig.Clone()}
	}
	a, b := splitAt(orig, p.SplitPos)
	return []*packet.TCPPacket{makeFake(orig, p), a, b}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clone(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
