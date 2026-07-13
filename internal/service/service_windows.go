//go:build windows

// Package service installs, removes and runs Skvoz as a Windows service so it
// can start on boot without a console window.
package service

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Name is the Windows service identifier.
const Name = "Skvoz"

// DisplayName is shown in the Services console.
const DisplayName = "Skvoz DPI bypass"

// IsInteractive reports whether the process runs in an interactive session
// (i.e. not under the service control manager).
func IsInteractive() (bool, error) {
	isSvc, err := svc.IsWindowsService()
	return !isSvc, err
}

// IsElevated reports whether the current process is running with administrator
// rights, which WinDivert requires.
func IsElevated() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY, 2,
		windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.GetCurrentProcessToken()
	member, err := token.IsMember(sid)
	return err == nil && member
}

// ElevationHint is shown to the user when IsElevated() is false.
func ElevationHint() string {
	return "right-click skvoz.exe and choose 'Run as administrator'"
}

// Install registers the service to run the current executable with the given
// arguments and start automatically at boot.
func Install(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager (run as administrator?): %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(Name); err == nil {
		s.Close()
		return fmt.Errorf("service %q is already installed", Name)
	}
	s, err := m.CreateService(Name, exe, mgr.Config{
		DisplayName: DisplayName,
		Description: "Bypasses DPI blocking for YouTube and Discord.",
		StartType:   mgr.StartAutomatic,
	}, args...)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

// Uninstall stops and removes the service.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(Name)
	if err != nil {
		return fmt.Errorf("service %q is not installed", Name)
	}
	defer s.Close()
	_, _ = s.Control(svc.Stop)
	return s.Delete()
}

// RunService runs under the service control manager, invoking run in a
// goroutine and closing the stop channel when a Stop/Shutdown is requested.
func RunService(run func(stop <-chan struct{}) error) error {
	return svc.Run(Name, &handler{run: run})
}

type handler struct {
	run func(stop <-chan struct{}) error
}

func (h *handler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- h.run(stop) }()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				close(stop)
				s <- svc.Status{State: svc.StopPending}
				<-done
				return false, 0
			}
		case <-done:
			return false, 0
		}
	}
}
