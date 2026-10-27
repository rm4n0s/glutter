package offscreen_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// seen is what the probe's state read from the framework.
type seen struct {
	scale      float64
	brightness ui.Brightness
	locale     ui.Locale
	window     ui.WindowInfo
	// What the build read from the MediaQuery.
	mqScale      float64
	mqBrightness ui.Brightness
	mq24h        bool
	mqBold       bool
}

// probeLog is how a test reaches the state of the probe.
type probeLog struct {
	s      *probeState
	builds int
	init   seen
	built  seen
}

// probe is a widget whose state reads the platform configuration and the window in InitState,
// as an app's root widget does, and builds a MediaQuery reader. It also rebuilds when the
// locales change.
type probe struct {
	widgets.StatefulWidget
	log *probeLog
}

func newProbe(log *probeLog) *probe {
	return &probe{StatefulWidget: widgets.NewStatefulWidgetBase(), log: log}
}

func (p *probe) CreateState() widgets.State { return &probeState{} }

type probeState struct {
	widgets.StateOf[*probe]
	widgets.WidgetsBindingObserverBase
}

func (s *probeState) InitState() {
	b := widgets.WidgetsBindingOf(s.Context())
	d := b.PlatformDispatcher()
	log := s.Widget().log
	log.s = s
	log.init = seen{scale: d.TextScaleFactor(), brightness: d.PlatformBrightness(), locale: d.Locale()}
	if v := d.View(ui.ImplicitViewID); v != nil {
		log.init.window = v.Window()
	}
	b.AddObserver(s)
}

func (s *probeState) Dispose() {
	if b := widgets.MaybeWidgetsBindingOf(s.Context()); b != nil {
		b.RemoveObserver(s)
	}
}

func (s *probeState) DidChangeLocales([]ui.Locale) { s.SetState(func() {}) }

// bump makes the state build again, which asks the view for a frame.
func (s *probeState) bump() { s.SetState(func() {}) }

func (s *probeState) Build(bc widgets.BuildContext) widgets.Widget {
	log := s.Widget().log
	mq := widgets.MediaQueryOf(bc)
	d := widgets.WidgetsBindingOf(bc).PlatformDispatcher()
	log.builds++
	log.built = seen{
		scale: d.TextScaleFactor(), brightness: d.PlatformBrightness(), locale: d.Locale(),
		mqScale: mq.TextScaler.Scale(10), mqBrightness: mq.PlatformBrightness, mq24h: mq.AlwaysUse24HourFormat, mqBold: mq.BoldText,
	}
	if v := d.View(ui.ImplicitViewID); v != nil {
		log.built.window = v.Window()
	}
	return widgets.NewSizedBoxShrink(nil)
}

var (
	german    = ui.Locale{LanguageCode: "de", CountryCode: "DE"}
	greek     = ui.Locale{LanguageCode: "el", CountryCode: "GR"}
	windowOn  = ui.WindowInfo{Decoration: ui.DecorationApp, Capabilities: ui.AllWindowCapabilities, Title: "Remote", Activated: true}
	windowOff = ui.WindowInfo{Decoration: ui.DecorationApp, Capabilities: ui.AllWindowCapabilities, Title: "Remote"}
	viewSize  = ui.Size{Width: 240, Height: 120}
)

func TestViewStartsWithTheParentConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		parent func(c *ui.PlatformConfiguration)
		opts   offscreen.ViewOpts
		want   seen
	}{
		{"defaults", func(c *ui.PlatformConfiguration) {}, offscreen.ViewOpts{},
			seen{scale: 1, brightness: ui.BrightnessLight, mqScale: 10, mqBrightness: ui.BrightnessLight}},
		{"text scale", func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 1.5 }, offscreen.ViewOpts{},
			seen{scale: 1.5, brightness: ui.BrightnessLight, mqScale: 15, mqBrightness: ui.BrightnessLight}},
		{"brightness", func(c *ui.PlatformConfiguration) { c.PlatformBrightness = ui.BrightnessDark }, offscreen.ViewOpts{},
			seen{scale: 1, brightness: ui.BrightnessDark, mqScale: 10, mqBrightness: ui.BrightnessDark}},
		{"locale", func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{german, greek} }, offscreen.ViewOpts{},
			seen{scale: 1, brightness: ui.BrightnessLight, locale: german, mqScale: 10, mqBrightness: ui.BrightnessLight}},
		{"24-hour format and bold text", func(c *ui.PlatformConfiguration) {
			c.AlwaysUse24HourFormat = true
			c.AccessibilityFeatures.BoldText = true
		}, offscreen.ViewOpts{},
			seen{scale: 1, brightness: ui.BrightnessLight, mqScale: 10, mqBrightness: ui.BrightnessLight, mq24h: true, mqBold: true}},
		{"window", func(c *ui.PlatformConfiguration) {}, offscreen.ViewOpts{Window: windowOn},
			seen{scale: 1, brightness: ui.BrightnessLight, window: windowOn, mqScale: 10, mqBrightness: ui.BrightnessLight}},
		{"everything", func(c *ui.PlatformConfiguration) {
			c.TextScaleFactor = 2
			c.PlatformBrightness = ui.BrightnessDark
			c.Locales = []ui.Locale{greek}
			c.AlwaysUse24HourFormat = true
		}, offscreen.ViewOpts{Window: windowOff},
			seen{scale: 2, brightness: ui.BrightnessDark, locale: greek, window: windowOff, mqScale: 20, mqBrightness: ui.BrightnessDark, mq24h: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &probeLog{}
			h := newHarnessWith(t, func(parent *gluttertest.Tester) { parent.SetPlatformConfiguration(tt.parent) },
				newProbe(log), viewSize, tt.opts)
			// The tree was mounted inside NewView, and it is never built a second time.
			if log.builds != 1 {
				t.Fatalf("the probe built %d times when the view was made, want 1", log.builds)
			}
			h.settle()
			if log.builds != 1 {
				t.Errorf("the probe built %d times after the first frames, want 1 (no second build)", log.builds)
			}
			if log.init != (seen{scale: tt.want.scale, brightness: tt.want.brightness, locale: tt.want.locale, window: tt.want.window}) {
				t.Errorf("InitState saw %+v, want the parent's values %+v", log.init, tt.want)
			}
			if log.built != tt.want {
				t.Errorf("the first build saw %+v, want %+v", log.built, tt.want)
			}
		})
	}
}

// configCase changes one field of the parent's configuration, and says how the view shows it.
type configCase struct {
	name string
	// field is the field of ui.PlatformConfiguration that the case covers.
	field  string
	change func(c *ui.PlatformConfiguration)
	// check is whether the view's dispatcher has the new value.
	check func(d *ui.PlatformDispatcher) bool
	// rebuilds is whether the probe builds again by itself, with no other reason to: the
	// parent's binding tells its observers of the change. A field with no observer method is
	// seen by the next frame of the view, which the test asks for with a bump.
	rebuilds bool
	// built is whether the build read the new value.
	built func(s seen) bool
}

func configCases() []configCase {
	accent := ui.NewColor(0xFF336699)
	return []configCase{
		{"locales", "Locales", func(c *ui.PlatformConfiguration) { c.Locales = []ui.Locale{german, greek} },
			func(d *ui.PlatformDispatcher) bool { return d.Locale() == german && len(d.Locales()) == 2 },
			true, func(s seen) bool { return s.locale == german }},
		{"text scale", "TextScaleFactor", func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 1.75 },
			func(d *ui.PlatformDispatcher) bool { return d.TextScaleFactor() == 1.75 },
			true, func(s seen) bool { return s.mqScale == 17.5 }},
		{"text scaler", "ScaleFontSize", func(c *ui.PlatformConfiguration) {
			c.TextScaleFactor = 2
			c.ScaleFontSize = func(size float64) float64 { return size*2 + 1 }
		}, func(d *ui.PlatformDispatcher) bool { return d.ScaleFontSize(14) == 29 && d.ScaleFontSize(14.5) == 30 },
			true, func(s seen) bool { return s.mqScale == 21 }},
		{"brightness", "PlatformBrightness", func(c *ui.PlatformConfiguration) { c.PlatformBrightness = ui.BrightnessDark },
			func(d *ui.PlatformDispatcher) bool { return d.PlatformBrightness() == ui.BrightnessDark },
			true, func(s seen) bool { return s.mqBrightness == ui.BrightnessDark }},
		{"accessibility features", "AccessibilityFeatures", func(c *ui.PlatformConfiguration) { c.AccessibilityFeatures.BoldText = true },
			func(d *ui.PlatformDispatcher) bool { return d.AccessibilityFeatures().BoldText },
			true, func(s seen) bool { return s.mqBold }},
		{"accent color", "AccentColor", func(c *ui.PlatformConfiguration) { c.AccentColor = &accent },
			func(d *ui.PlatformDispatcher) bool { a := d.AccentColor(); return a != nil && *a == accent },
			false, nil},
		{"24-hour format", "AlwaysUse24HourFormat", func(c *ui.PlatformConfiguration) { c.AlwaysUse24HourFormat = true },
			func(d *ui.PlatformDispatcher) bool { return d.AlwaysUse24HourFormat() },
			false, func(s seen) bool { return s.mq24h }},
		{"interface font", "UIFontFamily", func(c *ui.PlatformConfiguration) { c.UIFontFamily, c.UIFontSize = "Cantarell", 11 },
			func(d *ui.PlatformDispatcher) bool { f, s := d.UIFont(); return f == "Cantarell" && s == 11 },
			false, nil},
		{"monospace font", "MonospaceFontFamily", func(c *ui.PlatformConfiguration) { c.MonospaceFontFamily, c.MonospaceFontSize = "Hack", 12 },
			func(d *ui.PlatformDispatcher) bool { f, s := d.MonospaceFont(); return f == "Hack" && s == 12 },
			false, nil},
		{"caret blink time", "CaretBlinkTime", func(c *ui.PlatformConfiguration) { c.CaretBlinkTime = 800 * time.Millisecond },
			func(d *ui.PlatformDispatcher) bool { return d.CaretBlinkTime() == 800*time.Millisecond },
			false, nil},
		{"double click time", "DoubleClickTime", func(c *ui.PlatformConfiguration) { c.DoubleClickTime = 350 * time.Millisecond },
			func(d *ui.PlatformDispatcher) bool { return d.DoubleClickTime() == 350*time.Millisecond },
			false, nil},
	}
}

// otherFields are the fields that the cases above cover through another field of the same
// group: the size of a font goes with its family.
var otherFields = map[string]bool{"UIFontSize": true, "MonospaceFontSize": true}

func TestViewFollowsEveryFieldOfTheParentConfiguration(t *testing.T) {
	// Every field of the struct has a case, so a field added later is not forgotten.
	covered := map[string]bool{}
	for _, tt := range configCases() {
		covered[tt.field] = true
	}
	typ := reflect.TypeOf(ui.PlatformConfiguration{})
	for i := 0; i < typ.NumField(); i++ {
		if name := typ.Field(i).Name; !covered[name] && !otherFields[name] {
			t.Errorf("no case follows the field PlatformConfiguration.%s", name)
		}
	}

	for _, tt := range configCases() {
		t.Run(tt.name, func(t *testing.T) {
			log := &probeLog{}
			h := newHarness(t, newProbe(log), viewSize)
			h.settle()
			d := h.view.Binding().PlatformDispatcher()
			if tt.check(d) {
				t.Fatal("the view already has the new value")
			}
			before := log.builds

			h.parent.SetPlatformConfiguration(tt.change)
			if !tt.rebuilds {
				// The parent's binding hears of these fields through no callback: the view
				// looks at them at the start of its next frame.
				log.s.bump()
			}
			h.settle()

			if !tt.check(d) {
				t.Errorf("the view's dispatcher does not have the parent's new value")
			}
			if log.builds <= before {
				t.Errorf("the probe built %d times, was %d: no rebuild", log.builds, before)
			}
			if tt.built != nil && !tt.built(log.built) {
				t.Errorf("the rebuild did not see the new value: %+v", log.built)
			}
		})
	}
}

func TestViewKeepsAFixedConfiguration(t *testing.T) {
	fixed := ui.PlatformConfiguration{
		Locales: []ui.Locale{greek}, TextScaleFactor: 2, PlatformBrightness: ui.BrightnessDark, CaretBlinkTime: 700 * time.Millisecond,
	}
	log := &probeLog{}
	h := newHarnessWith(t, func(parent *gluttertest.Tester) {
		parent.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 1.25 })
	}, newProbe(log), viewSize, offscreen.ViewOpts{PlatformConfiguration: foundation.Some(fixed)})
	h.settle()
	d := h.view.Binding().PlatformDispatcher()
	want := seen{scale: 2, brightness: ui.BrightnessDark, locale: greek, mqScale: 20, mqBrightness: ui.BrightnessDark}
	if log.builds != 1 || log.init != (seen{scale: 2, brightness: ui.BrightnessDark, locale: greek}) || log.built != want {
		t.Fatalf("the first build saw %+v after %d builds (InitState %+v), want %+v once", log.built, log.builds, log.init, want)
	}

	// The caller's slice is not the view's.
	fixed.Locales[0] = german
	if d.Locale() != greek {
		t.Errorf("the view's locale is %v after the caller changed its slice, want %v", d.Locale(), greek)
	}

	// The parent's changes do not reach the view, in a callback or at a frame.
	h.parent.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) {
		c.TextScaleFactor = 3
		c.PlatformBrightness = ui.BrightnessLight
		c.Locales = []ui.Locale{german}
		c.CaretBlinkTime = time.Second
		c.AlwaysUse24HourFormat = true
	})
	log.s.bump()
	h.settle()
	if d.TextScaleFactor() != 2 || d.PlatformBrightness() != ui.BrightnessDark || d.Locale() != greek ||
		d.CaretBlinkTime() != 700*time.Millisecond || d.AlwaysUse24HourFormat() {
		t.Errorf("the view followed the parent: scale %v, brightness %v, locale %v, blink %v, 24h %v",
			d.TextScaleFactor(), d.PlatformBrightness(), d.Locale(), d.CaretBlinkTime(), d.AlwaysUse24HourFormat())
	}
	if log.built.mqScale != 20 || log.built.mq24h {
		t.Errorf("a rebuild saw %+v, want the fixed configuration", log.built)
	}

	// The app may still change it through the view's dispatcher.
	d.UpdatePlatformConfiguration(ui.PlatformConfiguration{TextScaleFactor: 1.5})
	h.settle()
	if log.built.mqScale != 15 {
		t.Errorf("after the app changed the view's configuration the build saw a scale of %v, want 1.5 (15 for a size of 10)", log.built.mqScale)
	}
	h.view.Close()
	if h.view.Following() {
		t.Error("a view with a fixed configuration registered an observer on the parent")
	}
}

func TestViewStopsFollowingAfterClose(t *testing.T) {
	tests := []struct {
		name  string
		close bool
	}{
		{"open view follows", false},
		{"closed view does not", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &probeLog{}
			h := newHarness(t, newProbe(log), viewSize)
			h.settle()
			d := h.view.Binding().PlatformDispatcher()
			if tt.close {
				h.view.Close()
			}
			h.parent.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) {
				c.TextScaleFactor = 2
				c.Locales = []ui.Locale{german}
			})
			h.parent.Pump(pace)
			if followed := d.TextScaleFactor() == 2 && d.Locale() == german; followed == tt.close {
				t.Errorf("the view's dispatcher followed the parent: %v, want %v", followed, !tt.close)
			}
			if registered := h.view.Following(); registered == tt.close {
				t.Errorf("the view's observer is registered on the parent: %v, want %v", registered, !tt.close)
			}
		})
	}
}

// decorated is a WindowDecoration around a plain body, with a title the view's window gives.
func decorated() widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewWindowDecoration(
		widgets.NewColoredBox(ui.NewColor(0xFFE8EAF6), widgets.NewSizedBoxExpand(nil))))
}

func TestViewWindow(t *testing.T) {
	tests := []struct {
		name string
		from ui.WindowInfo
		to   ui.WindowInfo
		// differs is whether the pixels of the two windows differ.
		differs bool
	}{
		{"active to inactive", windowOn, windowOff, true},
		{"inactive to active", windowOff, windowOn, true},
		{"no decoration to the app's", ui.WindowInfo{}, windowOn, true},
		{"the same window", windowOn, windowOn, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, decorated(), viewSize, offscreen.ViewOpts{Window: tt.from})
			h.settle()
			first := h.last().img
			if got := h.view.Binding().PlatformDispatcher().View(ui.ImplicitViewID).Window(); got != tt.from {
				t.Fatalf("the window of the view is %+v, want %+v", got, tt.from)
			}

			if err := h.view.SetWindow(tt.to); err != nil {
				t.Fatal(err)
			}
			h.settle()
			second := h.last().img
			if got := h.view.Binding().PlatformDispatcher().View(ui.ImplicitViewID).Window(); got != tt.to {
				t.Errorf("after SetWindow the window is %+v, want %+v", got, tt.to)
			}
			if differs := !reflect.DeepEqual(first.Pix, second.Pix); differs != tt.differs {
				t.Errorf("the pixels of the two windows differ: %v, want %v", differs, tt.differs)
			}

			// Going back draws the first window again: the window reached the tree, nothing else changed.
			if err := h.view.SetWindow(tt.from); err != nil {
				t.Fatal(err)
			}
			h.settle()
			samePixels(t, "the window set back", h.last().img, first)
		})
	}
}

func TestViewConfigurationSurvivesSetSizeAndSetWindow(t *testing.T) {
	size := ui.Size{Width: 200, Height: 100}
	padding := ui.ViewPadding{Left: 3, Top: 20, Right: 5, Bottom: 7}
	insets := ui.ViewPadding{Bottom: 40}
	display := ui.Display{ID: 7, DevicePixelRatio: 1, Size: ui.Size{Width: 1920, Height: 1080}, RefreshRate: 60}
	tests := []struct {
		name string
		act  func(t *testing.T, v *offscreen.View)
		// wantSize and wantWindow are the size in pixels and the window after the action.
		wantSize   ui.Size
		wantWindow ui.WindowInfo
	}{
		{"SetSize", func(t *testing.T, v *offscreen.View) {
			if err := v.SetSize(ui.Size{Width: 300, Height: 150}); err != nil {
				t.Fatal(err)
			}
		}, ui.Size{Width: 600, Height: 300}, windowOn},
		{"SetWindow", func(t *testing.T, v *offscreen.View) {
			if err := v.SetWindow(windowOff); err != nil {
				t.Fatal(err)
			}
		}, ui.Size{Width: 400, Height: 200}, windowOff},
		{"both", func(t *testing.T, v *offscreen.View) {
			if err := v.SetWindow(windowOff); err != nil {
				t.Fatal(err)
			}
			if err := v.SetSize(ui.Size{Width: 50, Height: 60}); err != nil {
				t.Fatal(err)
			}
			if err := v.SetSize(ui.Size{Width: 80, Height: 90}); err != nil {
				t.Fatal(err)
			}
		}, ui.Size{Width: 160, Height: 180}, windowOff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &probeLog{}
			h := newHarness(t, newProbe(log), size, offscreen.ViewOpts{DevicePixelRatio: 2, Window: windowOn})
			h.settle()
			d := h.view.Binding().PlatformDispatcher()
			// The app sets padding, insets and a display with UpdateViewConfiguration.
			cfg := ui.ViewConfiguration{
				PhysicalSize: ui.Size{Width: 400, Height: 200}, DevicePixelRatio: 2, ViewPadding: padding, ViewInsets: insets,
				Padding: padding, Display: display, Window: windowOn, GestureSettings: ui.GestureSettings{PhysicalTouchSlop: 36},
			}
			d.UpdateViewConfiguration(ui.ImplicitViewID, cfg)

			tt.act(t, h.view)
			h.settle()

			u := d.View(ui.ImplicitViewID)
			if u.PhysicalSize() != tt.wantSize {
				t.Errorf("the size is %v, want %v", u.PhysicalSize(), tt.wantSize)
			}
			if u.Window() != tt.wantWindow {
				t.Errorf("the window is %+v, want %+v", u.Window(), tt.wantWindow)
			}
			if u.DevicePixelRatio() != 2 || u.ViewPadding() != padding || u.Padding() != padding || u.ViewInsets() != insets ||
				u.Display() != display || u.GestureSettings() != cfg.GestureSettings {
				t.Errorf("the view lost what the app set: ratio %v, view padding %+v, padding %+v, insets %+v, display %+v, gestures %+v",
					u.DevicePixelRatio(), u.ViewPadding(), u.Padding(), u.ViewInsets(), u.Display(), u.GestureSettings())
			}
			// The constraints were not set by the app, so they follow the size.
			if u.PhysicalConstraints() != ui.NewViewConstraintsTight(tt.wantSize) {
				t.Errorf("the constraints are %+v, want tight around %v", u.PhysicalConstraints(), tt.wantSize)
			}
		})
	}
}

func TestViewSetSizeKeepsConstraintsTheAppSet(t *testing.T) {
	h := newHarness(t, newProbe(&probeLog{}), viewSize)
	h.settle()
	d := h.view.Binding().PlatformDispatcher()
	limits := ui.ViewConstraints{MinWidth: 10, MaxWidth: 1000, MinHeight: 10, MaxHeight: 1000}
	d.UpdateViewConfiguration(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: viewSize, DevicePixelRatio: 1, PhysicalConstraints: limits})
	if err := h.view.SetSize(ui.Size{Width: 100, Height: 50}); err != nil {
		t.Fatal(err)
	}
	if got := d.View(ui.ImplicitViewID).PhysicalConstraints(); got != limits {
		t.Errorf("the constraints are %+v, want %+v", got, limits)
	}
}

func TestViewSetWindowErrors(t *testing.T) {
	h := newHarness(t, newProbe(&probeLog{}), viewSize)
	h.view.Close()
	err := h.view.SetWindow(windowOn)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("View.SetWindow.ViewClosed") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
