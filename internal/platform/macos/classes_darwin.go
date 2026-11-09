//go:build darwin && !ios

package macos

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// registerClasses registers the classes that Go implements, once per process, under names with the
// Glutter prefix and the process id (SPEC §29.2): the app delegate, the window, the window's delegate and
// the view. Each method finds its Go object by the handle in the instance's ivar, and runs under the
// platform's guard so that a panic never unwinds through AppKit.
func (s *system) registerClasses() error {
	var err error
	if s.appDelegate, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "AppDelegate", Super: "NSObject", Protocols: []string{"NSApplicationDelegate"},
		Methods: []objcx.Method{
			{Name: "applicationDidFinishLaunching:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p := platformOf(self); p != nil {
					p.guard(p.didFinishLaunching)
				}
			}},
			{Name: "applicationShouldTerminate:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) uint64 {
				if p := platformOf(self); p != nil {
					p.guard(p.requestQuit)
				}
				return terminateCancel // Run ends by itself when the windows are closed
			}},
			{Name: "applicationShouldTerminateAfterLastWindowClosed:", Fn: func(objc.ID, objc.SEL, objc.ID) bool { return false }},
			{Name: "applicationSupportsSecureRestorableState:", Fn: func(objc.ID, objc.SEL, objc.ID) bool { return true }},
			{Name: "applicationDidChangeScreenParameters:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p := platformOf(self); p != nil {
					p.guard(p.displaysChanged)
				}
			}},
		},
	}); err != nil {
		return err
	}
	if s.window, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "Window", Super: "NSWindow",
		Methods: []objcx.Method{
			// A borderless window answers NO by default; the app's windows always take the keys.
			{Name: "canBecomeKeyWindow", Fn: func(objc.ID, objc.SEL) bool { return true }},
			{Name: "canBecomeMainWindow", Fn: func(objc.ID, objc.SEL) bool { return true }},
		},
	}); err != nil {
		return err
	}
	if s.windowDelegate, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "WindowDelegate", Super: "NSObject", Protocols: []string{"NSWindowDelegate"},
		Methods: []objcx.Method{
			{Name: "windowShouldClose:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) (ok bool) {
				ok = true
				if w := windowOf(self); w != nil {
					w.p.guard(func() { ok = w.shouldClose() })
				}
				return ok
			}},
			windowEvent("windowWillClose:", (*window).willClose),
			windowEvent("windowDidResize:", (*window).didResize),
			windowEvent("windowDidChangeBackingProperties:", (*window).didChangeBacking),
			windowEvent("windowDidChangeScreen:", (*window).didChangeScreen),
			windowEvent("windowDidBecomeKey:", func(w *window) { w.keyChanged(true) }),
			windowEvent("windowDidResignKey:", func(w *window) { w.keyChanged(false) }),
			windowEvent("windowDidMiniaturize:", (*window).stateChanged),
			windowEvent("windowDidDeminiaturize:", (*window).stateChanged),
			windowEvent("windowDidEnterFullScreen:", (*window).stateChanged),
			windowEvent("windowDidExitFullScreen:", (*window).stateChanged),
			windowEvent("windowWillStartLiveResize:", func(w *window) { w.liveResize(true) }),
			windowEvent("windowDidEndLiveResize:", func(w *window) { w.liveResize(false) }),
		},
	}); err != nil {
		return err
	}
	if s.view, err = objcx.RegisterClass(objcx.ClassDef{
		Name: "View", Super: "NSView",
		Methods: append([]objcx.Method{
			{Name: "acceptsFirstResponder", Fn: func(objc.ID, objc.SEL) bool { return true }},
			{Name: "acceptsFirstMouse:", Fn: func(objc.ID, objc.SEL, objc.ID) bool { return true }},
			{Name: "isOpaque", Fn: func(objc.ID, objc.SEL) bool { return true }},
			{Name: "performKeyEquivalent:", Fn: func(self objc.ID, _ objc.SEL, ev objc.ID) (handled bool) {
				if w := windowOf(self); w != nil && !w.closed {
					w.p.guard(func() { handled = w.performKeyEquivalent(self, ev) })
				}
				return handled
			}},
			{Name: "displayLinkFired:", Fn: func(self objc.ID, _ objc.SEL, link objc.ID) {
				if w := windowOf(self); w != nil {
					w.p.guard(func() { w.linkFired(link) })
				}
			}},
		}, append(viewMethods(), keyMethods()...)...),
	}); err != nil {
		return err
	}
	return nil
}

// windowEvent is a method of the window's delegate that takes a notification and calls a method of the
// window under the guard.
func windowEvent(name string, fn func(*window)) objcx.Method {
	return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
		if w := windowOf(self); w != nil {
			w.p.guard(func() { fn(w) })
		}
	}}
}

// platformOf is the platform of an instance of the app delegate.
func platformOf(self objc.ID) *Platform {
	p, _ := objcx.LookupAs[*Platform](objcx.HandleOf(self))
	return p
}

// windowOf is the window of an instance of the window's delegate or of the view.
func windowOf(self objc.ID) *window {
	w, _ := objcx.LookupAs[*window](objcx.HandleOf(self))
	return w
}

// didFinishLaunching activates a regular app once, so that it is in front when the user runs it from a
// terminal (SPEC §29.1).
func (p *Platform) didFinishLaunching() {
	if send[int64](p.app, "activationPolicy") == activationPolicyRegular {
		p.app.Send(sel("activateIgnoringOtherApps:"), true)
	}
}

// buildMainMenu makes the main menu of an app that has none of its own: the application menu with Hide,
// Hide Others, Show All and Quit (the Edit and Window menus come with step 33.6).
func buildMainMenu(name string) objc.ID {
	menu := class("NSMenu").Send(sel("alloc")).Send(sel("init"))
	appItem := class("NSMenuItem").Send(sel("alloc")).Send(sel("init"))
	menu.Send(sel("addItem:"), appItem)
	appMenu := class("NSMenu").Send(sel("alloc")).Send(sel("init"))
	add := func(title, action, key string, mask uint64) {
		t, k := objcx.NewString(title), objcx.NewString(key)
		item := class("NSMenuItem").Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"), t, sel(action), k)
		if mask != 0 {
			item.Send(sel("setKeyEquivalentModifierMask:"), mask)
		}
		appMenu.Send(sel("addItem:"), item)
		objcx.Release(t)
		objcx.Release(k)
		objcx.Release(item)
	}
	const (
		maskCommand = uint64(1 << 20)
		maskOption  = uint64(1 << 19)
	)
	add("Hide "+name, "hide:", "h", maskCommand)
	add("Hide Others", "hideOtherApplications:", "h", maskCommand|maskOption)
	add("Show All", "unhideAllApplications:", "", 0)
	appMenu.Send(sel("addItem:"), class("NSMenuItem").Send(sel("separatorItem")))
	add("Quit "+name, "terminate:", "q", maskCommand)
	appItem.Send(sel("setSubmenu:"), appMenu)
	objcx.Release(appMenu)
	objcx.Release(appItem)
	return objcx.Autorelease(menu)
}
