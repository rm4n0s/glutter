package windows

import (
	"math"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the windows (SPEC §25.4): the styles that a WindowConfig
// makes, the sizes in physical pixels, the limits of WM_GETMINMAXINFO, full screen, and the
// mapping of the messages to the WindowInfo that the engine hears. The calls to Windows are in
// window_windows.go; everything here takes numbers, so a test runs it on any host.

// styleSet is the style and the extended style of a window.
type styleSet struct {
	style, exStyle uint32
}

// styleOf works out the styles of a window from its config (SPEC §25.4): a regular window is
// WS_OVERLAPPEDWINDOW, one that is not resizable has neither WS_THICKFRAME nor WS_MAXIMIZEBOX,
// one that is not decorated (or is borderless) is WS_POPUP, and a dialog is WS_OVERLAPPED |
// WS_CAPTION | WS_SYSMENU with WS_EX_DLGMODALFRAME. A window whose title bar the app draws
// (DecorationsApp) keeps the caption and the frame of its kind, for the shadow, the rounded
// corners and Aero Snap, and the window procedure removes them from the client area.
//
// Flutter: shell/platform/windows/host_window_regular.cc, host_window_dialog.cc (the window styles)
func styleOf(cfg platform.WindowConfig) styleSet {
	if !cfg.IsDecorated() || cfg.Kind == platform.WindowKindBorderless {
		return styleSet{style: win32.WSPopup}
	}
	if cfg.Kind == platform.WindowKindDialog {
		return styleSet{style: win32.WSOverlapped | win32.WSCaption | win32.WSSysMenu, exStyle: win32.WSExDlgModalFrame}
	}
	s := win32.WSOverlappedWindow
	if !cfg.IsResizable() {
		s &^= win32.WSThickFrame | win32.WSMaximizeBox
	}
	return styleSet{style: s}
}

// ownsClientArea says whether the app's title bar covers the caption: the window keeps the frame
// styles and WM_NCCALCSIZE gives the client area the whole window.
func ownsClientArea(cfg platform.WindowConfig) bool {
	return cfg.Decorations == platform.DecorationsApp && cfg.IsDecorated() && cfg.Kind != platform.WindowKindBorderless
}

// scaleOfDPI is the scale of a DPI: 96 is 1, 144 is 1.5 (1.25, 1.5 and 1.75 are exact).
func scaleOfDPI(dpi uint32) float64 {
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / win32.UserDefaultScreenDPI
}

// physical converts logical pixels to physical ones, rounded.
func physical(logical, scale float64) int32 {
	return int32(math.Round(logical * scale))
}

// insets are the sizes of the non-client area on each side of a window, in physical pixels: what
// AdjustWindowRectExForDpi adds to the client rectangle.
type insets struct {
	left, top, right, bottom int32
}

// insetsOf reads the insets from the rectangle AdjustWindowRectExForDpi made of a client
// rectangle at the origin of the size given.
func insetsOf(adjusted win32.Rect, clientW, clientH int32) insets {
	return insets{
		left: -adjusted.Left, top: -adjusted.Top,
		right: adjusted.Right - clientW, bottom: adjusted.Bottom - clientH,
	}
}

// outer is the size of a window whose client area is the size.
func (in insets) outer(clientW, clientH int32) (w, h int32) {
	return clientW + in.left + in.right, clientH + in.top + in.bottom
}

// width and height are the non-client width and height.
func (in insets) width() int32  { return in.left + in.right }
func (in insets) height() int32 { return in.top + in.bottom }

// minMaxLimits works out the tracking limits that WM_GETMINMAXINFO reports (SPEC §25.4): the
// outer sizes of the minimum and maximum client sizes in logical pixels at the scale. A size of 0
// is no limit, so it is left as Windows set it (hasMin, hasMax false); a limit is not larger
// than the virtual screen.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::HandleMessage (WM_GETMINMAXINFO)
func minMaxLimits(minSize, maxSize ui.Size, scale float64, in insets, screenW, screenH int32) (minTrack, maxTrack win32.Point, hasMin, hasMax bool) {
	clamp := func(v, limit int32) int32 {
		if limit > 0 && v > limit {
			return limit
		}
		return v
	}
	if minSize.Width > 0 || minSize.Height > 0 {
		hasMin = true
		minTrack = win32.Point{
			X: clamp(physical(minSize.Width, scale)+in.width(), screenW),
			Y: clamp(physical(minSize.Height, scale)+in.height(), screenH),
		}
	}
	if maxSize.Width > 0 || maxSize.Height > 0 {
		hasMax = true
		w, h := int32(math.MaxInt32), int32(math.MaxInt32)
		if maxSize.Width > 0 {
			w = physical(maxSize.Width, scale) + in.width()
		}
		if maxSize.Height > 0 {
			h = physical(maxSize.Height, scale) + in.height()
		}
		maxTrack = win32.Point{X: clamp(w, screenW), Y: clamp(h, screenH)}
	}
	return minTrack, maxTrack, hasMin, hasMax
}

// fullscreenStyle is the style of a window in full screen: the frame and the caption removed
// (WS_OVERLAPPEDWINDOW), the window a WS_POPUP, and the borders of the extended style gone.
//
// Flutter: shell/platform/windows/host_window.cc HostWindow::SetFullscreen
func fullscreenStyle(saved styleSet) styleSet {
	return styleSet{
		style:   saved.style&^win32.WSOverlappedWindow&^win32.WSCaption | win32.WSPopup | win32.WSVisible,
		exStyle: saved.exStyle &^ (win32.WSExDlgModalFrame | win32.WSExWindowEdge | win32.WSExClientEdge | win32.WSExStaticEdge),
	}
}

// restoredStyle is the style a window gets back when it leaves full screen: the saved one, shown.
func restoredStyle(saved styleSet) styleSet {
	return styleSet{style: saved.style | win32.WSVisible, exStyle: saved.exStyle}
}

// windowState is the state that WM_SIZE reports, with the full screen the platform keeps.
type windowState struct {
	maximized, minimized, fullscreen bool
}

// stateOfSize reads the state from the wParam of WM_SIZE (SIZE_*). ok is false for the two values
// that say a window was hidden or shown by another window (SIZE_MAXSHOW, SIZE_MAXHIDE), which
// change nothing.
//
// Flutter: no counterpart
func stateOfSize(wparam uintptr, fullscreen bool) (st windowState, ok bool) {
	switch wparam {
	case win32.SizeRestored:
		return windowState{fullscreen: fullscreen}, true
	case win32.SizeMinimized:
		return windowState{minimized: true, fullscreen: fullscreen}, true
	case win32.SizeMaximized:
		return windowState{maximized: true, fullscreen: fullscreen}, true
	}
	return windowState{}, false
}

// decorationOf is who draws the title bar.
func decorationOf(cfg platform.WindowConfig) ui.DecorationMode {
	switch {
	case !cfg.IsDecorated() || cfg.Kind == platform.WindowKindBorderless:
		return ui.DecorationNone
	case cfg.Decorations == platform.DecorationsApp:
		return ui.DecorationApp
	}
	return ui.DecorationServerSide
}

// capabilitiesOf says what the user may do with the window, from its style bits.
func capabilitiesOf(style uint32, cfg platform.WindowConfig) ui.WindowCapabilities {
	return ui.WindowCapabilities{
		WindowMenu: style&win32.WSSysMenu != 0,
		Maximize:   style&win32.WSMaximizeBox != 0,
		Minimize:   style&win32.WSMinimizeBox != 0,
		Fullscreen: cfg.IsResizable() && cfg.Kind != platform.WindowKindDialog,
	}
}

// windowInfoOf is the WindowInfo that the engine hears of a window (SPEC §25.4).
func windowInfoOf(cfg platform.WindowConfig, style uint32, st windowState, activated bool, title string) ui.WindowInfo {
	return ui.WindowInfo{
		Decoration: decorationOf(cfg), Capabilities: capabilitiesOf(style, cfg), Title: title,
		Maximized: st.maximized, Fullscreen: st.fullscreen, Minimized: st.minimized, Activated: activated,
	}
}

// hitTest answers WM_NCHITTEST for a window whose title bar the app draws (SPEC §25.4): the eight
// resize directions in a band of the frame's thickness around the window, and HTCLIENT elsewhere.
// pt and the rectangle are in screen coordinates. A window that is maximized or full screen, or
// one that cannot be resized, has no resize band.
//
// Flutter: no counterpart
func hitTest(pt win32.Point, window win32.Rect, border int32, resizable, maximizedOrFullscreen bool) uintptr {
	if !resizable || maximizedOrFullscreen || border <= 0 {
		return win32.HTClient
	}
	left := pt.X < window.Left+border
	right := pt.X >= window.Right-border
	top := pt.Y < window.Top+border
	bottom := pt.Y >= window.Bottom-border
	switch {
	case top && left:
		return win32.HTTopLeft
	case top && right:
		return win32.HTTopRight
	case bottom && left:
		return win32.HTBottomLeft
	case bottom && right:
		return win32.HTBottomRight
	case left:
		return win32.HTLeft
	case right:
		return win32.HTRight
	case top:
		return win32.HTTop
	case bottom:
		return win32.HTBottom
	}
	return win32.HTClient
}

// edgeHit is the hit-test value that the system's resize loop takes for an edge of the window
// (WM_NCLBUTTONDOWN with HTLEFT and the others).
func edgeHit(e ui.ResizeEdge) uintptr {
	switch e {
	case ui.ResizeEdgeTop:
		return win32.HTTop
	case ui.ResizeEdgeBottom:
		return win32.HTBottom
	case ui.ResizeEdgeLeft:
		return win32.HTLeft
	case ui.ResizeEdgeRight:
		return win32.HTRight
	case ui.ResizeEdgeTopLeft:
		return win32.HTTopLeft
	case ui.ResizeEdgeTopRight:
		return win32.HTTopRight
	case ui.ResizeEdgeBottomLeft:
		return win32.HTBottomLeft
	case ui.ResizeEdgeBottomRight:
		return win32.HTBottomRight
	}
	return win32.HTCaption
}

// maximizedInset is how far the client area of an app-drawn title bar is inset on each side when
// the window is maximized: the frame thickness, which would otherwise reach beyond the screen
// (SM_CXSIZEFRAME + SM_CXPADDEDBORDER).
func maximizedInset(sizeFrame, paddedBorder int32) int32 { return sizeFrame + paddedBorder }

// darkTitleBar reads the registry value AppsUseLightTheme: the apps are dark when it exists and
// is 0. Without the value (Windows before 1809, or Wine) the title bar is light.
//
// Flutter: no counterpart
func darkTitleBar(appsUseLightTheme uint32, found bool) bool {
	return found && appsUseLightTheme == 0
}
