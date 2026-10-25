//go:build windows

package windows

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// winShell is shellAPI over the Win32 calls (SPEC §25.9). Its parts are in tray_windows.go (the notify
// icon and the icons), menu_windows.go (the popup menu), openuri_windows.go (ShellExecuteExW),
// filedialog_windows.go (the item dialogs) and worker_windows.go (the threads of the calls that wait).
type winShell struct{ p *Platform }

var _ shellAPI = winShell{}

// Post runs fn on the UI thread.
func (s winShell) Post(fn func()) { s.p.Post(fn) }

// After arms a timer of the platform's heap; it belongs to the UI thread.
func (s winShell) After(d time.Duration, fn func()) func() {
	id := s.p.timers.add(time.Now().Add(d), fn)
	return func() { s.p.timers.cancel(id) }
}

var _ platform.DesktopProvider = (*Platform)(nil)

// Desktop returns the desktop services of Windows (SPEC §25.9): the value implements mobile.Services,
// platform.TrayHost and platform.FileDialogHost. It is made on the first call, which may come from any
// goroutine, and lives as long as the platform.
//
// Flutter: no counterpart
func (p *Platform) Desktop() mobile.Services {
	p.deskOnce.Do(func() {
		created, err := win32.RegisterWindowMessage("TaskbarCreated")
		if err != nil {
			p.log.Warn("the TaskbarCreated message could not be registered; a restart of Explorer takes the tray icon away", "err", err)
			created = 0
		}
		p.desk.Store(newDesktop(winShell{p}, p.log, created))
	})
	return p.desk.Load()
}

// PortalHandle says how the dialogs name the window as their owner: "win32:" and its HWND in
// hexadecimal (SPEC §25.9). It calls done at once, on the UI thread.
//
// Flutter: no counterpart
func (w *window) PortalHandle(done func(handle string)) { done(portalHandleOf(w.hwnd)) }

var _ platform.PortalHandleProvider = (*window)(nil)
