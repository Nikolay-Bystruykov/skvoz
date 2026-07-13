// Package prober checks whether a host is reachable over TLS. autopick uses it
// to decide whether the currently-applied desync strategy actually restores
// access: with the engine running, a completed TLS handshake to a target host
// means DPI is no longer blocking it.
package prober

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

// DefaultTimeout bounds a single probe. A failing candidate blocks for the
// full timeout, and a from-scratch sweep tries up to 5 candidates, so this
// directly sets the worst-case disruption of a full re-sweep; kept short so
// that stays bounded while still giving a real handshake a fair chance.
const DefaultTimeout = 2 * time.Second

// TLSProber dials host:443 and performs a TLS handshake with SNI set to host.
type TLSProber struct {
	Timeout time.Duration
}

// Probe returns nil if the TLS handshake to host completes within the timeout.
// Certificate validity is not required — we only care that the handshake
// reaches the real server, which proves DPI let the ClientHello through.
func (p TLSProber) Probe(ctx context.Context, host string) error {
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := tls.Dialer{Config: &tls.Config{ServerName: host, InsecureSkipVerify: true}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, "443"))
	if err != nil {
		return err
	}
	return conn.Close()
}
