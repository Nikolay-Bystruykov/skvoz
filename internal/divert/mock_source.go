package divert

import (
	"io"
	"sync"
)

// MockSource is an in-memory PacketSource for tests. It yields a fixed list of
// packets from Recv and records everything passed to Send.
type MockSource struct {
	mu   sync.Mutex
	in   []*Packet
	idx  int
	Sent []*Packet
}

// NewMockSource returns a MockSource that will Recv the given packets in order,
// then return io.EOF.
func NewMockSource(in ...*Packet) *MockSource {
	return &MockSource{in: in}
}

// Recv returns the next queued packet, or io.EOF when exhausted.
func (m *MockSource) Recv() (*Packet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.idx >= len(m.in) {
		return nil, io.EOF
	}
	p := m.in[m.idx]
	m.idx++
	return p, nil
}

// Send records the packet.
func (m *MockSource) Send(p *Packet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, p)
	return nil
}

// Close is a no-op.
func (m *MockSource) Close() error { return nil }

// OutboundPacket is a helper for building a captured outbound packet.
func OutboundPacket(data []byte) *Packet {
	return &Packet{Data: data, Addr: Addr{Outbound: true}}
}
