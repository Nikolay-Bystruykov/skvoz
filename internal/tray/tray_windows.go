//go:build windows

package tray

import (
	"context"
	"log"
	"os"
	"time"

	"fyne.io/systray"

	"github.com/skvoz/skvoz/internal/assets"
	"github.com/skvoz/skvoz/internal/autopick"
	"github.com/skvoz/skvoz/internal/config"
	"github.com/skvoz/skvoz/internal/divert"
	"github.com/skvoz/skvoz/internal/hostlist"
	"github.com/skvoz/skvoz/internal/prober"
	"github.com/skvoz/skvoz/internal/settings"
	"github.com/skvoz/skvoz/internal/winenv"
)

// autopickTimeout bounds the whole strategy-selection sweep at startup and on
// "Проверить сейчас".
const autopickTimeout = 60 * time.Second

// Run shows the Skvoz tray icon and blocks until the user quits. It extracts
// the embedded driver, starts the engine with the first working strategy, and
// wires the menu to the engine manager. It assumes the process is already
// elevated (main ensures this before calling Run).
func Run(logger *log.Logger) {
	ui := &trayUI{log: logger, prober: prober.TLSProber{Timeout: prober.DefaultTimeout}}
	systray.Run(ui.onReady, func() {})
}

type trayUI struct {
	log    *log.Logger
	prober autopick.Prober
	mgr    *Manager
	dir    string
	state  settings.State

	mStatus    *systray.MenuItem
	mYouTube   *systray.MenuItem
	mDiscord   *systray.MenuItem
	mAutostart *systray.MenuItem
	mCheck     *systray.MenuItem
	mQuit      *systray.MenuItem
}

func (u *trayUI) onReady() {
	systray.SetIcon(iconICO)
	systray.SetTitle("Skvoz")
	systray.SetTooltip("Skvoz")

	dir, err := settings.Dir()
	if err != nil {
		u.log.Printf("settings dir: %v", err)
	}
	u.dir = dir
	u.state, _ = settings.Load(dir)
	u.log.Printf("tray onReady: dir=%s targets(yt=%v,dc=%v)", dir, u.state.YouTube, u.state.Discord)

	// Unpack the embedded WinDivert driver next to the settings and load it.
	if _, err := winenv.ExtractDriver(dir, assets.DriverFiles()); err != nil {
		u.log.Printf("extract driver FAILED: %v", err)
	} else {
		u.log.Printf("driver extracted to %s", dir)
	}
	divert.SetDriverDir(dir)

	u.mgr = New(divert.NewWinDivertSource, u.log)

	u.mStatus = systray.AddMenuItem("Запуск…", "")
	u.mStatus.Disable()
	systray.AddSeparator()
	u.mYouTube = systray.AddMenuItemCheckbox("YouTube", "Разблокировать YouTube", u.state.YouTube)
	u.mDiscord = systray.AddMenuItemCheckbox("Discord", "Разблокировать Discord", u.state.Discord)
	systray.AddSeparator()
	u.mAutostart = systray.AddMenuItemCheckbox("Запускать с Windows", "Автозапуск при входе в систему", winenv.IsAutostartEnabled())
	u.mCheck = systray.AddMenuItem("Проверить сейчас", "Заново подобрать рабочую стратегию")
	systray.AddSeparator()
	u.mQuit = systray.AddMenuItem("Выход", "Остановить Skvoz и выйти")

	u.log.Printf("tray ready (icon shown); starting engine")
	go u.loop()
	go u.reconfigure()
}

// loop dispatches menu clicks until the user quits.
func (u *trayUI) loop() {
	for {
		select {
		case <-u.mYouTube.ClickedCh:
			u.toggleTarget(u.mYouTube, &u.state.YouTube)
		case <-u.mDiscord.ClickedCh:
			u.toggleTarget(u.mDiscord, &u.state.Discord)
		case <-u.mAutostart.ClickedCh:
			u.toggleAutostart()
		case <-u.mCheck.ClickedCh:
			go u.reconfigure()
		case <-u.mQuit.ClickedCh:
			u.mgr.Stop()
			systray.Quit()
			return
		}
	}
}

// toggleTarget flips a YouTube/Discord checkbox, persists it, and restarts the
// engine for the new target set.
func (u *trayUI) toggleTarget(item *systray.MenuItem, field *bool) {
	*field = !*field
	if *field {
		item.Check()
	} else {
		item.Uncheck()
	}
	_ = u.state.Save(u.dir)
	go u.reconfigure()
}

// toggleAutostart creates or removes the logon Scheduled Task and reflects the
// resulting real state in the checkbox.
func (u *trayUI) toggleAutostart() {
	want := !u.state.Autostart
	exe, _ := os.Executable()
	var err error
	if want {
		err = winenv.EnableAutostart(exe)
	} else {
		err = winenv.DisableAutostart()
	}
	if err != nil {
		u.log.Printf("autostart toggle: %v", err)
	}
	u.state.Autostart = winenv.IsAutostartEnabled()
	if u.state.Autostart {
		u.mAutostart.Check()
	} else {
		u.mAutostart.Uncheck()
	}
	_ = u.state.Save(u.dir)
}

// reconfigure (re)starts the engine, auto-selecting the first strategy that
// actually reaches an enabled target. It updates the status line throughout.
func (u *trayUI) reconfigure() {
	lists, host, ok := u.enabledLists()
	if !ok {
		u.mgr.Stop()
		u.setStatus("выключено (нет целей)")
		return
	}

	// Try the last known-good strategy first. A full re-sweep restarts the
	// packet-capture handle once per candidate, which briefly disrupts any
	// in-flight connections (e.g. a video mid-load) — worth avoiding when we
	// already know a strategy that works.
	known := autopick.Candidate{Strategy: u.state.Strategy, FakeTTL: u.state.FakeTTL}
	candidates := autopick.PreferredFirst(known, autopick.Candidates())

	u.setStatus("подбираю стратегию…")
	u.log.Printf("reconfigure: probing host=%s, trying known-good strategy=%s first, then %d more candidates", host, known.Strategy, len(candidates)-1)
	cfg := config.Default()
	apply := func(c autopick.Candidate) error {
		cfg.Strategy = c.Strategy
		cfg.FakeTTL = c.FakeTTL
		u.log.Printf("trying strategy=%s ttl=%d", c.Strategy, c.FakeTTL)
		return u.mgr.Apply(cfg, lists)
	}

	ctx, cancel := context.WithTimeout(context.Background(), autopickTimeout)
	defer cancel()

	picked, ok := autopick.Select(ctx, candidates, host, apply, loggingProber{inner: u.prober, log: u.log})
	if !ok {
		// Nothing probed clean — keep the engine running on the saved strategy
		// so the user still has a chance rather than nothing at all.
		u.log.Printf("no strategy passed the probe; falling back to saved strategy=%s", u.state.Strategy)
		cfg.Strategy = u.state.Strategy
		cfg.FakeTTL = u.state.FakeTTL
		if err := u.mgr.Apply(cfg, lists); err != nil {
			u.setStatus("ошибка запуска")
			u.log.Printf("apply fallback FAILED: %v", err)
			return
		}
		u.setStatus("работает (стратегия по умолчанию)")
		return
	}

	u.log.Printf("selected working strategy=%s ttl=%d for host=%s", picked.Strategy, picked.FakeTTL, host)
	u.state.Strategy = picked.Strategy
	u.state.FakeTTL = picked.FakeTTL
	_ = u.state.Save(u.dir)
	u.setStatus("работает ✓ (" + picked.Strategy + ")")
}

// loggingProber wraps a Prober to record each probe's outcome in the log, so a
// support log shows exactly which strategy made a target reachable.
type loggingProber struct {
	inner autopick.Prober
	log   *log.Logger
}

func (l loggingProber) Probe(ctx context.Context, host string) error {
	err := l.inner.Probe(ctx, host)
	if err != nil {
		l.log.Printf("probe %s: FAIL (%v)", host, err)
	} else {
		l.log.Printf("probe %s: OK", host)
	}
	return err
}

// enabledLists returns the domain list for the currently-enabled targets plus a
// representative host to probe. ok is false when no target is enabled.
func (u *trayUI) enabledLists() (list *hostlist.List, probeHost string, ok bool) {
	switch {
	case u.state.YouTube && u.state.Discord:
		l, err := assets.Lists()
		return l, "youtube.com", err == nil
	case u.state.YouTube:
		l, err := assets.YouTubeList()
		return l, "youtube.com", err == nil
	case u.state.Discord:
		l, err := assets.DiscordList()
		return l, "discord.com", err == nil
	default:
		return nil, "", false
	}
}

func (u *trayUI) setStatus(s string) {
	if u.mStatus != nil {
		u.mStatus.SetTitle("● Skvoz — " + s)
	}
	systray.SetTooltip("Skvoz — " + s)
}
