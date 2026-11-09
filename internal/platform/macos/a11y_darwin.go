//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/coreaudio"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/macos/a11y"
	"github.com/rm4n0s/glutter/internal/platform/macos/iokit"
	"github.com/rm4n0s/glutter/ui"
)

// The glue of accessibility (SPEC §29.10): the services of the platform give the engine the bridge of package a11y,
// and the view hands the assistive technologies the root element of its tree.

// a11yHost is what the bridge asks of the platform.
type a11yHost struct{ p *Platform }

var _ a11y.Host = a11yHost{}

// ViewObjects are the NSView and the NSWindow of the view, and the scale of the window.
func (h a11yHost) ViewObjects(view ui.ViewID) (nsView, nsWindow objc.ID, scale float64, ok bool) {
	w, found := h.p.windows[view]
	if !found || w.closed {
		return 0, 0, 0, false
	}
	return w.view, w.ns, w.scale, true
}

// Guard runs fn under the platform's guard.
func (h a11yHost) Guard(fn func()) { h.p.guard(fn) }

// macServices is platform.Services of macOS: the bridge of accessibility, the gamepads (package iokit) and the audio (package coreaudio); there
// are no motion sensors on a Mac (SPEC §29.11).
type macServices struct {
	acc   platform.Accessibility
	pads  platform.Gamepads
	audio platform.Audio
}

var _ platform.Services = macServices{}

func (s macServices) Accessibility() platform.Accessibility { return s.acc }
func (s macServices) Gamepads() platform.Gamepads           { return s.pads }
func (s macServices) Audio() platform.Audio                 { return s.audio }
func (s macServices) Sensors() platform.Sensors             { return nil }

// a11yState is the platform's field for accessibility.
type a11yState struct {
	accOnce sync.Once
	acc     *a11y.Bridge
	// pads is the gamepad service (SPEC §29.11), made with the services.
	padsOnce sync.Once
	pads     *iokit.Service
	// audio is the audio service (SPEC §29.11), made with the services.
	audioOnce sync.Once
	audio     *coreaudio.Service
}

// Services returns the services of the platform: the accessibility bridge (SPEC §29.10) the gamepads and the audio (SPEC §29.11). It is made at the first
// call, which is the engine's.
//
// Flutter: no counterpart
func (p *Platform) Services() platform.Services {
	p.accOnce.Do(func() { p.acc = a11y.NewBridge(a11yHost{p}, p.log) })
	p.padsOnce.Do(func() {
		p.pads = iokit.New(iokit.Options{Manager: iokit.NewManager(p.Post, iokit.ManagerOptions{Guard: p.guard}), Post: p.Post, Logger: p.log})
	})
	p.audioOnce.Do(func() {
		p.audio = coreaudio.New(coreaudio.Options{Permissions: coreaudio.NewPermissions(), Log: p.log})
	})
	return macServices{acc: p.acc, pads: p.pads, audio: p.audio}
}

// accessibilityMethods are the methods of the view that make it the parent of the root element: the first question of
// an assistive technology turns the semantics tree on.
func accessibilityMethods() []objcx.Method {
	return []objcx.Method{
		{Name: "isAccessibilityElement", Fn: func(objc.ID, objc.SEL) bool { return false }},
		{Name: "accessibilityChildren", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			w := windowOf(self)
			if w == nil || w.closed || w.p.acc == nil {
				return 0
			}
			w.p.acc.Touch()
			root := w.p.acc.RootElement(w.id)
			if root == 0 {
				return 0
			}
			return objcx.Autorelease(objcx.NewArray(root))
		}},
		{Name: "accessibilityFocusedUIElement", Fn: func(self objc.ID, _ objc.SEL) objc.ID {
			w := windowOf(self)
			if w == nil || w.closed || w.p.acc == nil {
				return 0
			}
			return w.p.acc.FocusedElement(w.id)
		}},
		// The point is in the screen's coordinates (the origin at its bottom left).
		{Name: "accessibilityHitTest:", Fn: func(self objc.ID, _ objc.SEL, x, y float64) objc.ID {
			w := windowOf(self)
			if w == nil || w.closed || w.p.acc == nil {
				return 0
			}
			px, py := w.viewPixelsOfScreenPoint(x, y)
			return w.p.acc.ElementAt(w.id, px, py)
		}},
	}
}

// viewPixelsOfScreenPoint turns a point of the screen into the pixels of the view from its top left.
func (w *window) viewPixelsOfScreenPoint(x, y float64) (float64, float64) {
	screenRect := nsRect{Origin: nsPoint{X: x, Y: y}}
	inWindow := objc.Send[nsRect](w.ns, sel("convertRectFromScreen:"), screenRect)
	pt := objc.Send[nsPoint](w.view, sel("convertPoint:fromView:"), inWindow.Origin, objc.ID(0))
	h := send[nsRect](w.view, "bounds").Size.Height
	return pt.X * w.scale, (h - pt.Y) * w.scale
}
