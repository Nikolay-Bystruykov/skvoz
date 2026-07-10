// Package config parses Skvoz's command-line configuration. The one-click .bat
// presets simply invoke skvoz.exe with these flags, so this is the single
// source of truth for runtime options.
package config

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/skvoz/skvoz/internal/desync"
)

// QUIC modes.
const (
	QUICDrop = "drop" // suppress QUIC Initials so browsers fall back to TCP
	QUICOff  = "off"  // leave QUIC untouched
)

// Split positions.
const (
	SplitAtSNI    = "sni"    // split inside the SNI host name
	SplitAtMiddle = "middle" // split at the midpoint of the payload
)

// Service actions.
const (
	ServiceRun       = ""          // foreground (default)
	ServiceInstall   = "install"   // register the Windows service
	ServiceUninstall = "uninstall" // remove the Windows service
	ServiceRunSvc    = "run"       // run under the service control manager
)

// Config is the fully validated runtime configuration.
type Config struct {
	Strategy string   // desync strategy name
	FakeTTL  uint8    // TTL for decoy packets
	Lists    []string // domain-list file paths
	QUIC     string   // QUICDrop | QUICOff
	SplitPos string   // SplitAtSNI | SplitAtMiddle
	Service  string   // ServiceRun | ServiceInstall | ServiceUninstall | ServiceRunSvc
	Ports    []uint16 // target TCP destination ports
	ShowHelp bool
}

// Default returns the recommended default configuration.
func Default() Config {
	return Config{
		Strategy: "fakedsplit",
		FakeTTL:  desync.DefaultFakeTTL,
		Lists:    nil,
		QUIC:     QUICDrop,
		SplitPos: SplitAtSNI,
		Service:  ServiceRun,
		Ports:    []uint16{80, 443},
	}
}

// Parse builds a Config from command-line arguments (excluding argv[0]).
// Diagnostics are written to out. It returns an error on invalid input.
func Parse(args []string, out io.Writer) (Config, error) {
	cfg := Default()

	fs := flag.NewFlagSet("skvoz", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		fmt.Fprint(out, usage)
		fs.PrintDefaults()
	}

	var lists, ports string
	fs.StringVar(&cfg.Strategy, "strategy", cfg.Strategy, "desync strategy: "+strings.Join(desync.Names(), ", "))
	fakeTTL := fs.Uint("fake-ttl", uint(cfg.FakeTTL), "TTL for decoy (fake) packets")
	fs.StringVar(&lists, "lists", "", "comma-separated domain-list files (e.g. lists/list-youtube.txt)")
	fs.StringVar(&cfg.QUIC, "quic", cfg.QUIC, "QUIC handling: drop | off")
	fs.StringVar(&cfg.SplitPos, "split", cfg.SplitPos, "split position: sni | middle")
	fs.StringVar(&cfg.Service, "service", cfg.Service, "service action: install | uninstall | run")
	fs.StringVar(&ports, "ports", "80,443", "comma-separated target TCP ports")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	if *fakeTTL == 0 || *fakeTTL > 255 {
		return cfg, fmt.Errorf("fake-ttl must be 1..255, got %d", *fakeTTL)
	}
	cfg.FakeTTL = uint8(*fakeTTL)

	if lists != "" {
		cfg.Lists = splitCSV(lists)
	}

	parsedPorts, err := parsePorts(ports)
	if err != nil {
		return cfg, err
	}
	cfg.Ports = parsedPorts

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Validate checks that all enumerated fields hold legal values.
func (c Config) Validate() error {
	if _, err := desync.Get(c.Strategy); err != nil {
		return err
	}
	switch c.QUIC {
	case QUICDrop, QUICOff:
	default:
		return fmt.Errorf("quic must be drop|off, got %q", c.QUIC)
	}
	switch c.SplitPos {
	case SplitAtSNI, SplitAtMiddle:
	default:
		return fmt.Errorf("split must be sni|middle, got %q", c.SplitPos)
	}
	switch c.Service {
	case ServiceRun, ServiceInstall, ServiceUninstall, ServiceRunSvc:
	default:
		return fmt.Errorf("service must be install|uninstall|run, got %q", c.Service)
	}
	if len(c.Ports) == 0 {
		return fmt.Errorf("at least one target port is required")
	}
	return nil
}

// Filter returns the WinDivert filter string selecting the traffic Skvoz must
// inspect: outbound TCP to the target ports, plus (when QUIC dropping is on)
// outbound UDP to 443.
func (c Config) Filter() string {
	var tcp strings.Builder
	for i, p := range c.Ports {
		if i > 0 {
			tcp.WriteString(" or ")
		}
		fmt.Fprintf(&tcp, "tcp.DstPort == %d", p)
	}
	filter := fmt.Sprintf("outbound and (tcp and (%s))", tcp.String())
	if c.QUIC == QUICDrop {
		filter = fmt.Sprintf("outbound and ((tcp and (%s)) or (udp and udp.DstPort == 443))", tcp.String())
	}
	return filter
}

// Args reproduces the command-line flags (except --service) that recreate this
// configuration. It is used to build the argument list for the installed
// Windows service.
func (c Config) Args() []string {
	args := []string{
		"--strategy", c.Strategy,
		"--fake-ttl", strconv.Itoa(int(c.FakeTTL)),
		"--quic", c.QUIC,
		"--split", c.SplitPos,
		"--ports", joinPorts(c.Ports),
	}
	if len(c.Lists) > 0 {
		args = append(args, "--lists", strings.Join(c.Lists, ","))
	}
	return args
}

func joinPorts(ports []uint16) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = strconv.Itoa(int(p))
	}
	return strings.Join(parts, ",")
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parsePorts(s string) ([]uint16, error) {
	var out []uint16
	for _, p := range splitCSV(s) {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("invalid port %q", p)
		}
		out = append(out, uint16(n))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ports specified")
	}
	return out, nil
}

const usage = `Skvoz - DPI bypass for YouTube and Discord (Windows).

Usage:
  skvoz.exe [flags]

Examples:
  skvoz.exe --lists lists/list-youtube.txt,lists/list-discord.txt
  skvoz.exe --strategy fake --fake-ttl 6 --lists lists/list-discord.txt
  skvoz.exe --service install --lists lists/list-youtube.txt

Flags:
`
