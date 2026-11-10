//go:build darwin && !ios

package macos

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// This file reads the system settings from AppKit and watches them (SPEC §29.8). The conversion into
// platform.Settings is settings.go.

// The notifications that say a setting changed (the strings are the names of the constants), and the
// user defaults of the caret's blink.
const (
	appearanceKeyPath        = "effectiveAppearance"
	systemColorsChanged      = "NSSystemColorsDidChangeNotification"
	localeChanged            = "NSCurrentLocaleDidChangeNotification"
	accessibilityChanged     = "NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification"
	appearanceNameDarkAqua   = "NSAppearanceNameDarkAqua"
	appearanceNameAqua       = "NSAppearanceNameAqua"
	blinkPeriodOnDefault     = "NSTextInsertionPointBlinkPeriodOn"
	blinkPeriodOffDefault    = "NSTextInsertionPointBlinkPeriodOff"
	dateTemplateHourOnly     = "j"
	nsOperationQueueMainName = "NSOperationQueue"
)

// DevAppearanceEnv forces the appearance of the app to "dark" or "light" in debug builds, so that a screenshot
// or a test shows the dark theme without the user changing the system's appearance (SPEC §29.8).
//
// Flutter: no counterpart
const DevAppearanceEnv = "GLUTTER_DEV_APPEARANCE"

// applyDevAppearance sets NSApp.appearance from DevAppearanceEnv; any other value leaves the system's.
func (p *Platform) applyDevAppearance() {
	name := ""
	switch p.getenv(DevAppearanceEnv) {
	case "dark":
		name = appearanceNameDarkAqua
	case "light":
		name = appearanceNameAqua
	default:
		return
	}
	objcx.WithPool(func() {
		ns := objcx.NewString(name)
		p.app.Send(sel("setAppearance:"), class("NSAppearance").Send(sel("appearanceNamed:"), ns))
		objcx.Release(ns)
	})
}

// settingsWatch reads the settings and tells the handler when one changed. It belongs to the UI thread.
type settingsWatch struct {
	p *Platform
	// last is what was read last; have says it was read.
	last settingsInput
	have bool

	// observer is the instance of the KVO observer of NSApp.effectiveAppearance, 0 when not observing.
	observer objc.ID
	// The notification observers: each is the token that addObserverForName gave, with the center that
	// it is in; block is the block they all share.
	tokens  []objc.ID
	centers []objc.ID
	block   objc.Block
}

// newSettingsWatch starts watching the appearance (KVO on NSApp.effectiveAppearance) and the
// notifications of the colors, the accessibility options and the locale.
func newSettingsWatch(p *Platform) *settingsWatch {
	w := &settingsWatch{p: p}
	objcx.WithPool(func() {
		w.observer = p.s.settingsObserver.New(p.handle)
		key := objcx.NewString(appearanceKeyPath)
		p.app.Send(sel("addObserver:forKeyPath:options:context:"), w.observer, key, uint64(0), uintptr(0))
		objcx.Release(key)

		w.block = objcx.NewBlock(func(_ objc.Block, _ objc.ID) {
			if !p.closed {
				p.guard(w.changed)
			}
		})
		queue := class(nsOperationQueueMainName).Send(sel("mainQueue"))
		def := class("NSNotificationCenter").Send(sel("defaultCenter"))
		ws := class("NSWorkspace").Send(sel("sharedWorkspace")).Send(sel("notificationCenter"))
		for _, n := range []struct {
			center objc.ID
			name   string
		}{{def, systemColorsChanged}, {def, localeChanged}, {ws, accessibilityChanged}} {
			name := objcx.NewString(n.name)
			tok := n.center.Send(sel("addObserverForName:object:queue:usingBlock:"), name, objc.ID(0), queue, w.block)
			objcx.Release(name)
			w.tokens = append(w.tokens, objcx.Retain(tok))
			w.centers = append(w.centers, n.center)
		}
	})
	return w
}

// close stops the watching.
func (w *settingsWatch) close() {
	objcx.WithPool(func() {
		for i, tok := range w.tokens {
			w.centers[i].Send(sel("removeObserver:"), tok)
			objcx.Release(tok)
		}
		w.tokens, w.centers = nil, nil
		if w.observer != 0 {
			key := objcx.NewString(appearanceKeyPath)
			w.p.app.Send(sel("removeObserver:forKeyPath:"), w.observer, key)
			objcx.Release(key)
			objcx.Release(w.observer)
			w.observer = 0
		}
	})
	if w.block != 0 {
		w.block.Release()
		w.block = 0
	}
}

// current returns the settings, reading them the first time.
func (w *settingsWatch) current() platform.Settings {
	if !w.have {
		w.last, w.have = readSettings(w.p), true
	}
	return settingsOf(w.last)
}

// changed rereads the settings and gives the handler the new ones when something differs from what it
// was told.
func (w *settingsWatch) changed() {
	in := readSettings(w.p)
	if w.have && reflect.DeepEqual(in, w.last) {
		return
	}
	w.last, w.have = in, true
	if h := w.p.handler; h != nil {
		h.OnSettings(settingsOf(in))
	}
}

// readSettings reads what the settings are made of from AppKit and Foundation.
func readSettings(p *Platform) (in settingsInput) {
	objcx.WithPool(func() {
		in.Dark = isDarkAppearance(send[objc.ID](p.app, "effectiveAppearance"))

		accent := class("NSColor").Send(sel("controlAccentColor"))
		srgb := accent.Send(sel("colorUsingColorSpace:"), class("NSColorSpace").Send(sel("sRGBColorSpace")))
		if srgb != 0 {
			var r, g, b, a float64
			srgb.Send(sel("getRed:green:blue:alpha:"), unsafe.Pointer(&r), unsafe.Pointer(&g), unsafe.Pointer(&b), unsafe.Pointer(&a))
			in.AccentRed, in.AccentGreen, in.AccentBlue, in.HasAccent = r, g, b, true
		}

		ws := class("NSWorkspace").Send(sel("sharedWorkspace"))
		in.ReduceMotion = send[bool](ws, "accessibilityDisplayShouldReduceMotion")
		in.IncreaseContrast = send[bool](ws, "accessibilityDisplayShouldIncreaseContrast")
		in.ReduceTransparency = send[bool](ws, "accessibilityDisplayShouldReduceTransparency")
		in.DifferentiateWithoutColor = send[bool](ws, "accessibilityDisplayShouldDifferentiateWithoutColor")

		in.DoubleClickSeconds = send[float64](class("NSEvent"), "doubleClickInterval")

		defaults := class("NSUserDefaults").Send(sel("standardUserDefaults"))
		on, hasOn := defaultsMillis(defaults, blinkPeriodOnDefault)
		off, hasOff := defaultsMillis(defaults, blinkPeriodOffDefault)
		in.BlinkOnMillis, in.BlinkOffMillis, in.HasBlink = on, off, hasOn || hasOff

		tmpl := objcx.NewString(dateTemplateHourOnly)
		format := class("NSDateFormatter").Send(sel("dateFormatFromTemplate:options:locale:"), tmpl, uint64(0),
			class("NSLocale").Send(sel("currentLocale")))
		objcx.Release(tmpl)
		in.TimeFormat = objcx.String(format)

		langs := class("NSLocale").Send(sel("preferredLanguages"))
		for i := range objcx.ArrayCount(langs) {
			in.Languages = append(in.Languages, objcx.String(objcx.ArrayAt(langs, i)))
		}
	})
	return in
}

// isDarkAppearance says that the appearance is the dark one of the two that an app supports: the
// appearance that best matches Aqua and Dark Aqua.
func isDarkAppearance(appearance objc.ID) bool {
	if appearance == 0 {
		return false
	}
	aqua, dark := objcx.NewString(appearanceNameAqua), objcx.NewString(appearanceNameDarkAqua)
	names := objcx.NewArray(aqua, dark)
	best := appearance.Send(sel("bestMatchFromAppearancesWithNames:"), names)
	isDark := objcx.String(best) == appearanceNameDarkAqua
	objcx.Release(names)
	objcx.Release(aqua)
	objcx.Release(dark)
	return isDark
}

// defaultsMillis reads a number of the user defaults; has is false when the key is not set.
func defaultsMillis(defaults objc.ID, key string) (millis int, has bool) {
	k := objcx.NewString(key)
	defer objcx.Release(k)
	v := defaults.Send(sel("objectForKey:"), k)
	if v == 0 || !send[bool](v, "isKindOfClass:", objc.ID(objcx.MustClass("NSNumber"))) {
		return 0, false
	}
	return int(objcx.Int(v)), true
}

// Settings returns the system settings (SPEC §29.8).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterEngine.mm FlutterEngine onSettingsChanged
func (p *Platform) Settings() platform.Settings {
	if p.watch == nil {
		return settingsOf(settingsInput{})
	}
	return p.watch.current()
}
