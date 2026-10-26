//go:build windows

package windows

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/uia"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// The accessibility of Windows is UI Automation (SPEC §25.11): package uia holds the bridge, and this file
// is what the platform gives it: Services (the engine asks for the bridge there), the window of each
// view for the rectangles and the HWND, and the answer to WM_GETOBJECT.

// accWindows is the uia.Window of the platform. The views and their HWNDs are kept in a table that
// any goroutine may read, because UI Automation can call a provider from a thread of a client.
type accWindows struct {
	p     *Platform
	hwnds sync.Map // ui.ViewID -> uintptr
}

var _ uia.Window = (*accWindows)(nil)

// Post runs fn on the UI thread.
func (a *accWindows) Post(fn func()) { a.p.Post(fn) }

// HWND is the window of the view, 0 when it is gone.
func (a *accWindows) HWND(view ui.ViewID) uintptr {
	if h, ok := a.hwnds.Load(view); ok {
		return h.(uintptr)
	}
	return 0
}

// ClientRect is the client area of the view's window on the screen, in physical pixels.
func (a *accWindows) ClientRect(view ui.ViewID) (x, y, width, height float64, ok bool) {
	hwnd := a.HWND(view)
	if hwnd == 0 {
		return 0, 0, 0, 0, false
	}
	rc, err := win32.GetClientRect(hwnd)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	origin, ok := win32.ClientToScreen(hwnd, win32.Point{})
	if !ok {
		return 0, 0, 0, 0, false
	}
	return float64(origin.X), float64(origin.Y), float64(rc.Right - rc.Left), float64(rc.Bottom - rc.Top), true
}

// accessibilityServices is the platform.Services of Windows: the accessibility bridge.
type accessibilityServices struct{ acc *uia.Bridge }

var _ platform.Services = accessibilityServices{}

// Accessibility implements platform.Services.
func (s accessibilityServices) Accessibility() platform.Accessibility { return s.acc }

// Services returns the services of the platform: the UI Automation bridge (SPEC §25.11). It is made on
// the first call. When the COM objects of the providers cannot be made the log says so and there is no
// accessibility (nil), as on a desktop without a service on Linux. The tray, the notifications, the file
// dialogs and the URIs are the desktop services of Desktop.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services {
	p.accOnce.Do(func() {
		env, err := uia.Shared()
		if err != nil {
			p.log.Warn("accessibility is not available: the COM objects of UI Automation could not be made", "err", err)
			return
		}
		p.accWin = &accWindows{p: p}
		for _, w := range p.sortedWindows() {
			p.accWin.hwnds.Store(w.id, w.hwnd)
		}
		p.acc = uia.NewBridge(env, p.accWin, p.log)
	})
	if p.acc == nil {
		return nil
	}
	return accessibilityServices{p.acc}
}

// accessibilityAdd tells the bridge that a window exists.
func (w *window) accessibilityAdd() {
	if w.p.accWin != nil {
		w.p.accWin.hwnds.Store(w.id, w.hwnd)
	}
}

// accessibilityRemove tells the bridge that the window is gone: its HWND is no longer answered.
func (w *window) accessibilityRemove() {
	if w.p.accWin != nil {
		w.p.accWin.hwnds.Delete(w.id)
	}
}

// onGetObject handles WM_GETOBJECT (SPEC §25.11): a client that asks for the UI Automation provider
// (UiaRootObjectId) turns the bridge on and gets the provider of the window. Any other request is for
// DefWindowProc, which answers MSAA clients through the proxy of Windows.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::OnGetObject
func (w *window) onGetObject(hwnd, wparam, lparam uintptr) (uintptr, bool) {
	if w.p.acc == nil {
		return 0, false
	}
	return w.p.acc.Answer(w.id, hwnd, wparam, lparam)
}
