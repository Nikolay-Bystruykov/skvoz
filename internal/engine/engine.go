// Package engine is the platform-agnostic core: it reads packets from a
// PacketSource, decides what to do with each (pass through, desync, or drop),
// and writes the result back. All decision logic lives here and is exercised by
// tests through a mock PacketSource, so it is fully verifiable without Windows.
package engine

import (
	"log"

	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/desync"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/hostlist"
	"github.com/skvoz/skvoz/internal/packet"
	"github.com/skvoz/skvoz/internal/quic"
	"github.com/skvoz/skvoz/internal/tls"
)

// Engine processes packets according to a Config against a domain List.
type Engine struct {
	cfg      config.Config
	strategy desync.Strategy
	lists    *hostlist.List
	ports    map[uint16]bool
	logger   *log.Logger
}

// New builds an Engine. The strategy name in cfg must already be valid
// (config.Parse guarantees this).
func New(cfg config.Config, lists *hostlist.List, logger *log.Logger) (*Engine, error) {
	s, err := desync.Get(cfg.Strategy)
	if err != nil {
		return nil, err
	}
	ports := make(map[uint16]bool, len(cfg.Ports))
	for _, p := range cfg.Ports {
		ports[p] = true
	}
	return &Engine{cfg: cfg, strategy: s, lists: lists, ports: ports, logger: logger}, nil
}

// Run reads and processes packets until the source returns an error (e.g. EOF
// or a closed handle), which it returns to the caller.
func (e *Engine) Run(src divert.PacketSource) error {
	for {
		pkt, err := src.Recv()
		if err != nil {
			return err
		}
		for _, out := range e.Process(pkt) {
			if err := src.Send(out); err != nil {
				e.logf("send: %v", err)
			}
		}
	}
}

// Process decides the fate of a single captured packet and returns the packets
// to inject. It never returns nil for a packet it should not touch — it returns
// the original — so the engine is fail-open by construction. Returning an empty
// slice means "drop".
func (e *Engine) Process(pkt *divert.Packet) (result []*divert.Packet) {
	// Fail-open: any unexpected panic re-injects the original packet so the
	// user's connectivity is never broken by a bug in a strategy.
	defer func() {
		if r := recover(); r != nil {
			e.logf("recovered while processing packet: %v", r)
			result = []*divert.Packet{pkt}
		}
	}()

	// Only outbound traffic is a candidate for desync.
	if !pkt.Addr.Outbound {
		return []*divert.Packet{pkt}
	}

	// TCP path: target the TLS ClientHello.
	if tcp, err := packet.ParseTCP(pkt.Data); err == nil {
		return e.processTCP(pkt, tcp)
	}

	// UDP path: suppress QUIC so browsers fall back to TLS-over-TCP.
	if udp, err := packet.ParseUDP(pkt.Data); err == nil {
		return e.processUDP(pkt, udp)
	}

	return []*divert.Packet{pkt}
}

func (e *Engine) processTCP(pkt *divert.Packet, tcp *packet.TCPPacket) []*divert.Packet {
	if !e.ports[tcp.DstPort] {
		return []*divert.Packet{pkt}
	}
	if !tls.IsClientHello(tcp.Payload) {
		return []*divert.Packet{pkt}
	}
	info, err := tls.ParseClientHello(tcp.Payload)
	if err != nil {
		return []*divert.Packet{pkt} // no SNI or malformed: leave it alone
	}
	if !e.lists.Match(info.SNI) {
		return []*divert.Packet{pkt} // not a targeted domain
	}

	params := desync.Params{
		SplitPos:    e.splitPos(tcp.Payload, info),
		FakeTTL:     e.cfg.FakeTTL,
		FakePayload: nil, // zero-filled decoy (see desync.makeFake)
	}
	segments := e.strategy.Apply(tcp, params)

	out := make([]*divert.Packet, 0, len(segments))
	for _, seg := range segments {
		out = append(out, &divert.Packet{Data: seg.Serialize(), Addr: pkt.Addr})
	}
	e.logf("desync %s -> %s (%d segments)", e.cfg.Strategy, info.SNI, len(out))
	return out
}

func (e *Engine) processUDP(pkt *divert.Packet, udp *packet.UDPInfo) []*divert.Packet {
	if e.cfg.QUIC != config.QUICDrop {
		return []*divert.Packet{pkt}
	}
	if udp.DstPort == 443 && quic.IsInitial(udp.Payload) {
		e.logf("drop QUIC Initial -> :443 (force TCP fallback)")
		return nil // drop
	}
	return []*divert.Packet{pkt}
}

// splitPos chooses where to cut the ClientHello segment. In SNI mode it cuts in
// the middle of the host name so neither half contains the whole name; in
// middle mode it cuts the payload in half.
func (e *Engine) splitPos(payload []byte, info tls.ClientHelloInfo) int {
	if e.cfg.SplitPos == config.SplitAtSNI && info.SNILength > 0 {
		return info.SNIOffset + info.SNILength/2
	}
	return len(payload) / 2
}

func (e *Engine) logf(format string, args ...any) {
	if e.logger != nil {
		e.logger.Printf(format, args...)
	}
}
