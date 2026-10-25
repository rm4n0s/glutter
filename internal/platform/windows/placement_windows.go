//go:build windows

package windows

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// cascadeStep is the step, in logical pixels, of the cascade of windows.
const cascadeStep = 32

// workAreaOf is the work area of the monitor that the window is on, and its rectangle on the screen:
// the visible frame (DWMWA_EXTENDED_FRAME_BOUNDS, without the invisible borders and the shadow that
// Windows 10 and 11 add to the window's rectangle) when asked for, else the window's rectangle.
func workAreaOf(w *window, visibleFrame bool) (rect, work win32.Rect, ok bool) {
	rect, err := win32.GetWindowRect(w.hwnd)
	if err != nil {
		return rect, work, false
	}
	if visibleFrame {
		if frame, err := win32.DwmGetWindowAttributeRect(w.hwnd, win32.DWMWAExtendedFrameBounds); err == nil && frame.Right > frame.Left {
			rect = frame
		}
	}
	mi, err := win32.GetMonitorInfo(win32.MonitorFromWindow(w.hwnd, win32.MonitorDefaultToNearest))
	if err != nil {
		return rect, work, false
	}
	return rect, mi.Work, true
}

// defaultPlace is the place for a new window of the outer size ow by oh (SPEC §25.4): a dialog is
// centered over its owner's frame, and any other window is a step below and to the right of the window made
// before it (the newest one that is open and in its normal state). ok is false when there is no such
// window, and when the window starts maximized or full screen: Windows then chooses (CW_USEDEFAULT).
//
// Flutter: shell/platform/windows/host_window_dialog.cc HostWindowDialog::GetInitialRect
//
// Deviation: Flutter does not move the dialog into the work area, and leaves the other windows to
// CW_USEDEFAULT; the cascade is Glutter's (SPEC §12.10).
func (p *Platform) defaultPlace(w *window, ow, oh int32) (x, y int32, ok bool) {
	if w.initial == platform.WindowStateMaximized || w.initial == platform.WindowStateFullscreen || w.initial == platform.WindowStateMinimized {
		return 0, 0, false
	}
	if w.owner != nil {
		if rect, work, ok := workAreaOf(w.owner, true); ok && !win32.IsIconic(w.owner.hwnd) {
			x, y = centerOn(rect, ow, oh, work)
			return x, y, true
		}
		return 0, 0, false
	}
	windows := p.sortedWindows()
	for i := len(windows) - 1; i >= 0; i-- {
		prev := windows[i]
		if prev == w || prev.closed || prev.owner != nil || prev.state.minimized || prev.state.maximized || prev.fullscreen() || win32.IsIconic(prev.hwnd) {
			continue
		}
		if rect, work, ok := workAreaOf(prev, false); ok {
			x, y = cascadeFrom(rect, physical(cascadeStep, scaleOfDPI(w.dpi)), ow, oh, work)
			return x, y, true
		}
		break
	}
	return 0, 0, false
}
