# Skvoz — Design Spec (v1)

Date: 2026-07-10
Status: Approved (design), pending implementation plan

## 1. Purpose

**Skvoz** is an open-source DPI-bypass (anti-censorship) tool for **Windows**, written in **Go**, built on top of the **WinDivert** packet-interception driver — the same architectural approach as the original `zapret`/`winws`.

It is **not** a VPN or proxy: no traffic is tunneled through a remote server and nothing is encrypted/re-routed. Skvoz intercepts the user's own outbound packets and manipulates them so that DPI (Deep Packet Inspection) systems fail to classify and block them, restoring access to **YouTube** and **Discord**.

Audience: ordinary Windows users in censored networks who want a "download → run → works" experience.

## 2. Data Flow

```
Outbound packet ─▶ WinDivert intercepts (filter: TCP 80/443 out, UDP 443 out)
        ─▶ Is it a TLS ClientHello (TCP 443/80) or a QUIC Initial (UDP 443)?
        ─▶ Extract SNI (server name) from the packet
        ─▶ Is SNI in a target list (youtube / discord)?
             ├─ yes ─▶ apply desync strategy ─▶ re-inject modified/fake packets
             └─ no  ─▶ re-inject unchanged (never touch unrelated traffic)
```

Only targeted domains are modified. All other traffic passes through untouched — critical for stability and performance.

## 3. Bypass Strategies (core engine)

Implement proven zapret-style techniques, selectable per profile (different ISPs need different tricks):

- **split** — split the TCP segment carrying the ClientHello at a configurable position (e.g. at the SNI offset), so the DPI cannot reassemble the hostname.
- **fake** — send a decoy ClientHello with a low TTL (dies before reaching the real server, but the DPI sees it and desyncs its state), then send the real one.
- **disorder** — send the split segments in reverse order.
- **fakedsplit** — combination of fake + split.
- **QUIC handling** — drop/mangle the QUIC Initial so the browser falls back to TLS-over-TCP, which the strategies above already handle. (v1 default: suppress QUIC to YouTube domains and let TLS path do the work.)

Strategies are configured per profile. Each strategy is a pure transformation from an intercepted packet to a list of packets to inject, so it is unit-testable without WinDivert.

## 4. Components (Go packages)

| Package | Responsibility |
|---|---|
| `internal/divert` | Thin WinDivert wrapper (open handle, recv, send/re-inject). Exposes a `PacketSource` interface so the engine can run against a mock in tests. Pure `syscall`/`golang.org/x/sys/windows` — **no cgo** (enables cross-compiling the .exe from macOS). |
| `internal/tls` | Parse TLS record + handshake, locate ClientHello, extract SNI and its byte offset. |
| `internal/quic` | Parse/identify QUIC Initial packets; extract SNI where feasible. |
| `internal/desync` | The strategies (split / fake / disorder / fakedsplit). Pure functions over packet bytes. |
| `internal/hostlist` | Load domain lists; match a SNI against them (exact + subdomain suffix match). |
| `internal/config` | CLI flags, profiles, preset parsing. |
| `internal/service` | Install / run as a Windows service (`golang.org/x/sys/windows/svc`). |
| `cmd/skvoz` | CLI entry point; wires config → divert → engine. |

### PacketSource interface (test seam)

```go
type Packet struct {
    Data []byte
    Addr WinDivertAddress // direction, iface, etc.
}

type PacketSource interface {
    Recv() (Packet, error)   // blocking read of next intercepted packet
    Send(Packet) error       // re-inject
    Close() error
}
```

The real implementation wraps WinDivert; a `mockSource` feeds recorded packet bytes for tests.

## 5. Distribution Contents ("download and run")

Release zip contains:
- `skvoz.exe`
- WinDivert files (`WinDivert.dll`, `WinDivert64.sys`)
- `lists/list-youtube.txt`, `lists/list-discord.txt`
- one-click presets: `youtube.bat`, `discord.bat`, `general.bat`
- `service-install.bat` (autostart as Windows service), `service-uninstall.bat`
- `README.md` (Russian + English quick start)

## 6. Reliability & Error Handling

- No admin rights / WinDivert driver missing → clear, actionable error + install link. Detect elevation up front.
- **Fail-open**: if the engine errors while processing a packet, re-inject the original unmodified packet so the user's connectivity never breaks.
- Graceful shutdown: on Ctrl+C / service stop, close the WinDivert handle so no packets are left queued.

## 7. Testing Strategy

- All "pure" logic is unit-tested and **runs on macOS** (dev machine): ClientHello/SNI parsing, packet-split math, disorder/fake construction, hostlist matching, config/preset parsing.
- The WinDivert layer is isolated behind `PacketSource`; the engine is tested by feeding recorded packet bytes through a mock and asserting the emitted packet set.
- Because the binding is cgo-free, `GOOS=windows GOARCH=amd64 go build` produces `skvoz.exe` from macOS. Live "does YouTube open" verification is done by the user on Windows.

## 8. Build & Distribution

- License: **MIT** for Skvoz's own code. WinDivert ships under its own license (LGPLv3 / GPLv3 dual) — documented and attributed in README/NOTICE; WinDivert binaries are downloaded/bundled, not modified.
- GitHub Actions: on tag push, build `skvoz.exe`, assemble the release zip (exe + WinDivert + lists + presets + README), and publish to GitHub Releases.
- `go test ./...` runs in CI on every push.

## 9. Repo Layout

```
skvoz/
  cmd/skvoz/main.go
  internal/divert/      windivert.go  source.go  mock_source.go
  internal/tls/         clienthello.go
  internal/quic/        initial.go
  internal/desync/      strategies.go
  internal/hostlist/    hostlist.go
  internal/config/      config.go  presets.go
  internal/service/     service.go
  lists/                list-youtube.txt  list-discord.txt
  presets/              youtube.bat  discord.bat  general.bat  service-install.bat
  third_party/windivert/ (fetched at build; license + NOTICE)
  .github/workflows/    ci.yml  release.yml
  README.md  LICENSE  NOTICE  go.mod
```

## 10. Out of Scope for v1 (deferred to phase 2)

- GUI / system-tray app
- Automatic strategy auto-discovery (probe which desync works on the user's ISP)
- Linux (NFQUEUE) and macOS engines
- Domain lists beyond YouTube / Discord (structure stays extensible)

## 11. Legal / Ethical Note

Skvoz is an anti-censorship tool that restores access to lawful services. The README states that users are responsible for compliance with their local laws. No traffic interception of other users, no server-side component, no data collection.
