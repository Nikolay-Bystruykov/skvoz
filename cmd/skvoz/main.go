// Command skvoz is the Skvoz DPI-bypass daemon. It captures outbound packets
// via WinDivert and applies desync strategies to targeted domains so that
// DPI-based blocking of YouTube and Discord is defeated.
//
// Skvoz is not a VPN or proxy: no traffic leaves your machine to a third party.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/skvoz/skvoz/internal/appmode"
	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/engine"
	"github.com/skvoz/skvoz/internal/hostlist"
	"github.com/skvoz/skvoz/internal/service"
	"github.com/skvoz/skvoz/internal/tray"
	"github.com/skvoz/skvoz/internal/winenv"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// No arguments means a double-click: show the tray GUI. Any flag keeps the
	// original command-line / service behavior.
	if appmode.IsGUI(os.Args[1:]) {
		logger := log.New(os.Stderr, "skvoz: ", log.LstdFlags)
		logger.Printf("Skvoz %s starting (tray mode)", version)
		if relaunched, err := winenv.EnsureElevated(); err != nil {
			logger.Printf("could not elevate: %v", err)
		} else if relaunched {
			return // an elevated copy took over; this instance exits.
		}
		tray.Run(logger)
		return
	}

	// Command-line mode: reattach to the parent terminal so a GUI-subsystem
	// binary still prints output when run from a console.
	winenv.AttachParentConsole()

	cfg, err := config.Parse(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(2)
	}

	logger := log.New(os.Stdout, "skvoz: ", log.LstdFlags)

	if code := run(cfg, logger); code != 0 {
		os.Exit(code)
	}
}

func run(cfg config.Config, logger *log.Logger) int {
	// Service management actions run and exit immediately.
	switch cfg.Service {
	case config.ServiceInstall:
		if err := installService(cfg); err != nil {
			logger.Printf("install failed: %v", err)
			return 1
		}
		logger.Printf("service %q installed and started", service.Name)
		return 0
	case config.ServiceUninstall:
		if err := service.Uninstall(); err != nil {
			logger.Printf("uninstall failed: %v", err)
			return 1
		}
		logger.Printf("service %q removed", service.Name)
		return 0
	}

	if !service.IsElevated() {
		logger.Printf("WARNING: not elevated; packet capture will fail to open. To fix: %s", service.ElevationHint())
	}

	lists, err := loadLists(cfg, logger)
	if err != nil {
		logger.Printf("%v", err)
		return 1
	}

	eng, err := engine.New(cfg, lists, logger)
	if err != nil {
		logger.Printf("engine: %v", err)
		return 1
	}

	runEngine := func(stop <-chan struct{}) error {
		filter := cfg.Filter()
		logger.Printf("opening WinDivert with filter: %s", filter)
		src, err := divert.NewWinDivertSource(filter)
		if err != nil {
			return err
		}
		// Close the handle when asked to stop, which unblocks Recv.
		go func() {
			<-stop
			logger.Printf("stopping...")
			_ = src.Close()
		}()
		return eng.Run(src)
	}

	// Under the service control manager, hand control to the SCM loop.
	if cfg.Service == config.ServiceRunSvc {
		if err := service.RunService(runEngine); err != nil {
			logger.Printf("service run: %v", err)
			return 1
		}
		return 0
	}

	// Foreground: stop on Ctrl+C.
	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		close(stop)
	}()

	logger.Printf("skvoz %s running (strategy=%s, quic=%s, domains=%d). Press Ctrl+C to stop.",
		version, cfg.Strategy, cfg.QUIC, lists.Len())
	if err := runEngine(stop); err != nil {
		// A closed handle after a stop request is the normal shutdown path.
		select {
		case <-stop:
			return 0
		default:
			logger.Printf("fatal: %v", err)
			return 1
		}
	}
	return 0
}

// loadLists loads every configured domain list and reports how many domains
// were loaded. It is an error to end up with zero domains, since Skvoz would
// then do nothing.
func loadLists(cfg config.Config, logger *log.Logger) (*hostlist.List, error) {
	lists := hostlist.New()
	for _, path := range cfg.Lists {
		n, err := lists.LoadFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
		logger.Printf("loaded %d domains from %s", n, path)
	}
	if lists.Len() == 0 {
		return nil, fmt.Errorf("no domains loaded; pass --lists lists/list-youtube.txt,lists/list-discord.txt")
	}
	return lists, nil
}

// installService resolves list paths to absolute (the service runs with a
// different working directory) and registers the service.
func installService(cfg config.Config) error {
	abs := make([]string, len(cfg.Lists))
	for i, p := range cfg.Lists {
		a, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		abs[i] = a
	}
	cfg.Lists = abs
	args := append([]string{"--service", "run"}, cfg.Args()...)
	return service.Install(args)
}
