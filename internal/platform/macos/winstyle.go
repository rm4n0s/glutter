package macos

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The bits of NSWindowStyleMask that the platform uses (AppKit's values; plain Go so that the style is
// tested on every host).
const (
	maskBorderless       uint64 = 0
	maskTitled           uint64 = 1 << 0
	maskClosable         uint64 = 1 << 1
	maskMiniaturizable   uint64 = 1 << 2
	maskResizable        uint64 = 1 << 3
	maskFullSizeContent  uint64 = 1 << 15
	maskNonactivating    uint64 = 1 << 7
	maskUtilityWindow    uint64 = 1 << 4
	maskUnifiedTitleBar  uint64 = 1 << 12
	maskFullScreenWindow uint64 = 1 << 14
)

// windowStyle is what a WindowConfig asks of an NSWindow (SPEC §29.4), as plain data: winstyle.go has
// no call into AppKit.
//
// Flutter: no counterpart
type windowStyle struct {
	// Titled, Closable, Miniaturizable and Resizable are the buttons and the title bar of the window.
	Titled, Closable, Miniaturizable, Resizable bool
	// FullSizeContent puts the content under the title bar, whose background is then transparent: the
	// traffic lights stay over the content of an app that draws its own title bar (DecorationsApp).
	FullSizeContent bool
	// Borderless is a window with no title bar and no border.
	Borderless bool
	// Dialog is a child window of its parent; Modal runs it with the parent disabled.
	Dialog, Modal bool
	// FullScreen is whether the user can make the window a full-screen space.
	FullScreen bool
}

// styleFor is the style of the window that the config describes: a regular window is titled, closable,
// miniaturizable and resizable as the config says; a dialog has no minimize button; a borderless
// window, or one that is not decorated, has no title bar at all; DecorationsApp keeps the traffic lights
// over a full-size content view.
func styleFor(cfg platform.WindowConfig) windowStyle {
	resizable := cfg.IsResizable()
	switch {
	case cfg.Kind == platform.WindowKindBorderless || !cfg.IsDecorated():
		return windowStyle{Borderless: true, Resizable: resizable, Dialog: cfg.Kind == platform.WindowKindDialog, Modal: cfg.Modal && cfg.HasParent}
	case cfg.Kind == platform.WindowKindDialog:
		return windowStyle{
			Titled: true, Closable: true, Resizable: resizable,
			FullSizeContent: cfg.Decorations == platform.DecorationsApp,
			Dialog:          true, Modal: cfg.Modal && cfg.HasParent,
		}
	default:
		return windowStyle{
			Titled: true, Closable: true, Miniaturizable: true, Resizable: resizable,
			FullSizeContent: cfg.Decorations == platform.DecorationsApp,
			FullScreen:      resizable,
		}
	}
}

// mask is the NSWindowStyleMask of the style.
func (s windowStyle) mask() uint64 {
	m := maskBorderless
	if s.Titled {
		m |= maskTitled
	}
	if s.Closable {
		m |= maskClosable
	}
	if s.Miniaturizable {
		m |= maskMiniaturizable
	}
	if s.Resizable {
		m |= maskResizable
	}
	if s.FullSizeContent {
		m |= maskFullSizeContent
	}
	return m
}

// capabilities are what the user can do with the window (ui.WindowCapabilities): the window menu is
// the title bar's, maximize is zoom, fullscreen is the native space.
func (s windowStyle) capabilities() ui.WindowCapabilities {
	return ui.WindowCapabilities{
		WindowMenu: s.Titled,
		Maximize:   s.Titled && s.Resizable,
		Fullscreen: s.FullScreen,
		Minimize:   s.Miniaturizable,
	}
}

// decoration says who draws the title bar (ui.DecorationMode).
func (s windowStyle) decoration() ui.DecorationMode {
	switch {
	case s.Borderless:
		return ui.DecorationNone
	case s.FullSizeContent:
		return ui.DecorationApp
	default:
		return ui.DecorationServerSide
	}
}
