package services

// BrowserContextMenuBackend is what BrowserContextMenu asks to show or hide the browser's
// context menu: the platform dispatcher (ui.PlatformDispatcher) does. It reports whether the
// platform has such a menu; the desktop does not.
//
// Flutter: no counterpart (the flutter/contextmenu channel)
type BrowserContextMenuBackend interface {
	// SetBrowserContextMenuEnabled shows (true) or hides (false) the browser's context menu
	// and reports whether it did.
	SetBrowserContextMenuEnabled(enabled bool) bool
}

// BrowserContextMenu controls the browser's context menu on the web: the menu that appears on
// a right click or on selected text, for example. On the web the browser's context menu is
// enabled by default and Flutter's own context menus are hidden. On every other platform it
// does nothing, and Enabled stays true.
//
// Deviation: Flutter's class has static members and a hidden singleton; here the binding makes
// one (WidgetsBinding.BrowserContextMenu), and the calls are done when they return, not
// futures that complete later (SPEC §12.10).
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu
type BrowserContextMenu struct {
	backend BrowserContextMenuBackend
	enabled bool
}

// NewBrowserContextMenu makes the controller of the browser's context menu, which is enabled
// to begin with. The backend may be nil, as on a platform without such a menu.
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu._
func NewBrowserContextMenu(backend BrowserContextMenuBackend) *BrowserContextMenu {
	return &BrowserContextMenu{backend: backend, enabled: true}
}

// Enabled is whether showing the browser's context menu is enabled: any event that the
// browser uses to open its context menu (a right click) does so. It only says what
// DisableContextMenu and EnableContextMenu did; the page's own handlers of the contextmenu
// event are not known to it. It is true until the context menu is disabled.
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu.enabled
func (m *BrowserContextMenu) Enabled() bool { return m.enabled }

// DisableContextMenu disables the browser's context menu. By default, when the app starts, it
// is enabled. It does nothing where there is no browser.
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu.disableContextMenu
func (m *BrowserContextMenu) DisableContextMenu() {
	if m.backend != nil && m.backend.SetBrowserContextMenuEnabled(false) {
		m.enabled = false
	}
}

// EnableContextMenu enables the browser's context menu, typically after DisableContextMenu.
// It does nothing where there is no browser.
//
// Flutter: services/browser_context_menu.dart BrowserContextMenu.enableContextMenu
func (m *BrowserContextMenu) EnableContextMenu() {
	if m.backend != nil && m.backend.SetBrowserContextMenuEnabled(true) {
		m.enabled = true
	}
}
