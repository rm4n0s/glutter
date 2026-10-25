package desktop

import (
	"context"
	"path/filepath"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/widgets"
)

// windowsDesktop is what package desktop asks of Windows (SPEC §25.9): the platform's own services, which the
// engine hands out through the dispatcher (internal/platform/windows.Desktop). It has the interface of Android's
// Desktop (open a URI, show and close a notification) and the optional interfaces of a tray and of file dialogs.
// The tests of the package replace it with a fake. The session bus does not exist on Windows, and D-Bus is
// never tried there.
type windowsDesktop interface {
	androidDesktop
	platform.TrayHost
	platform.FileDialogHost
}

// newWindowsDesktop returns the services of the Windows platform that runs the binding, or nil on every other
// platform. It asks the engine through the dispatcher and does not look at the target platform, which a test can
// set to Windows on any machine. The services of Android have neither a tray nor file dialogs, so they are not
// taken for Windows'.
func newWindowsDesktop(b *widgets.WidgetsBinding) windowsDesktop {
	svc, _ := b.PlatformDispatcher().PlatformService(mobile.ServiceDesktop).(windowsDesktop)
	return svc
}

// windowsReason is why the service behind tag does not use D-Bus on Windows.
func windowsReason(tag string) string {
	switch tag {
	case "TrayUnavailable":
		return "the system tray of Windows is not served over D-Bus: D-Bus is never tried on Windows"
	case "FileChooserUnavailable":
		return "the file dialogs of Windows are not served over D-Bus: D-Bus is never tried on Windows"
	}
	return "Windows has no session bus: D-Bus is never tried on Windows"
}

// winTray is the tray icon of the app on Windows: the platform's Tray, made and updated on the UI thread by
// the state of the SystemTray widget.
type winTray struct {
	s *services
	h platform.Tray
}

// newWinTray reserves the tray of the app and puts its icon in the notification area. It fails with
// TrayDuplicate when the app has a tray, and with the platform's error (TrayRegisterFailed) when the icon cannot
// be added. It must be called on the UI thread.
func newWinTray(s *services, spec platform.TraySpec) (*winTray, error) {
	s.tray.mu.Lock()
	if s.tray.active {
		s.tray.mu.Unlock()
		return nil, errors.New(tagTrayDuplicate, "the app already has a system tray; an app has one at a time")
	}
	s.tray.active = true
	s.tray.mu.Unlock()
	h, err := s.windows.NewTray(spec)
	if err != nil {
		s.tray.mu.Lock()
		s.tray.active = false
		s.tray.mu.Unlock()
		return nil, err
	}
	return &winTray{s: s, h: h}, nil
}

// update hands the platform the new model.
func (t *winTray) update(spec platform.TraySpec) { t.h.Update(spec) }

// release takes the icon out of the tray and lets a new tray be made.
func (t *winTray) release() {
	t.h.Close()
	t.s.tray.mu.Lock()
	t.s.tray.active = false
	t.s.tray.mu.Unlock()
}

// guarded wraps a callback of the tray: the platform calls it on the UI thread, where a panic is reported
// through the error reporter of the binding and does not unwind the platform's message loop.
func (s *services) guarded(what string, fn func()) func() {
	if fn == nil {
		return nil
	}
	b := s.binding
	return func() {
		foundation.Guard(b.ErrorReporter(), foundation.CallbackPanicked, "while handling "+what+" of the system tray", nil, fn)
	}
}

// traySpec is the model of the SystemTray widget as the platform's tray takes it: the icon, the tip, the
// menu and the two clicks. The status, the attention icon and the scroll have no counterpart on Windows.
func (s *systemTrayState) traySpec(svc *services) platform.TraySpec {
	w := s.Widget()
	id, title := w.o.ID, w.o.Title
	if id == "" {
		id = s.appID
	}
	if title == "" {
		title = s.appID
	}
	return platform.TraySpec{
		ID: id, Title: title, Tooltip: w.o.Tooltip, TooltipBody: w.o.TooltipBody, Icon: w.icon,
		Menu:                trayItemsOf(svc, w.o.Menu),
		OnActivate:          svc.guarded("a click", w.o.OnActivate),
		OnSecondaryActivate: svc.guarded("a middle click", w.o.OnSecondaryActivate),
	}
}

// trayItemsOf turns the entries of the menu into the platform's.
func trayItemsOf(svc *services, entries []TrayMenuEntry) []platform.TrayMenuItem {
	var out []platform.TrayMenuItem
	for _, e := range entries {
		switch e := e.(type) {
		case *TrayMenuSeparator:
			out = append(out, platform.TrayMenuItem{Separator: true})
		case *TrayMenuItem:
			if e == nil {
				continue
			}
			out = append(out, platform.TrayMenuItem{
				Label: e.label, Disabled: e.o.Disabled, Hidden: e.o.Hidden, Checkable: e.o.Checkable,
				Radio: e.o.Radio, Checked: e.o.Checked,
				Children:  trayItemsOf(svc, e.children),
				OnPressed: svc.guarded("a menu item", e.onPressed),
			})
		}
	}
	return out
}

// filtersOf turns the filters of a file dialog into the platform's: the MIME types have no counterpart.
func filtersOf(filters []FileFilter) []platform.FileFilter {
	if len(filters) == 0 {
		return nil
	}
	out := make([]platform.FileFilter, len(filters))
	for i, f := range filters {
		out[i] = platform.FileFilter{Name: f.Name, Patterns: f.Patterns}
	}
	return out
}

// showFileDialog shows the common item dialog of Windows and turns the answer into a result. The dialog is
// modal to the window of the call unless the app asked for NotModal.
func (c *call) showFileDialog(ctx context.Context, spec platform.FileDialogSpec, notModal bool) (FilesResult, error) {
	if !notModal {
		spec.Owner = c.handle
	}
	res, err := c.s.windows.ShowFileDialog(ctx, spec)
	if err != nil {
		return FilesResult{}, err
	}
	return FilesResult{Cancelled: res.Cancelled, Paths: res.Paths, Filter: res.Filter}, nil
}

// currentFileParts splits the path of CurrentFile into the folder and the name the save dialog starts with.
func currentFileParts(path string) (folder, name string) {
	return filepath.Dir(path), filepath.Base(path)
}
