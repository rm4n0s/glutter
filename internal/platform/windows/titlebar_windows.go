//go:build windows

package windows

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// applyTitleBarTheme makes the title bar dark when the system's app mode is dark
// (DWMWA_USE_IMMERSIVE_DARK_MODE: attribute 20 on Windows 10 version 2004 and 11, else 19 on
// versions 1809 to 1909). The setting is read once, when the platform starts; step 28.6 follows
// it, and then the bar changes live (SPEC §25.4). A Windows or a Wine that does not know the
// attribute leaves the bar as it is.
//
// Flutter: shell/platform/windows/host_window.cc UpdateTheme
func (w *window) applyTitleBarTheme() {
	if w.closed {
		return
	}
	var v int32
	if w.p.dark {
		v = 1
	}
	if err := win32.DwmSetWindowAttribute(w.hwnd, win32.DWMWAUseImmersiveDarkMode, v); err != nil {
		if err := win32.DwmSetWindowAttribute(w.hwnd, win32.DWMWAUseImmersiveDarkModeBefore20H1, v); err != nil {
			w.p.log.Debug("the title bar theme was not set", "err", err)
		}
	}
}
