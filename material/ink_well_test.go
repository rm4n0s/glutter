package material_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// canvasPixel is the color of the Material of the default theme: M2's canvas color.
func inkApp(theme *ThemeData, child widgets.Widget) widgets.Widget {
	return materialApp(theme, NewMaterial(widgets.NewCenter(child)))
}

func square(child widgets.Widget) widgets.Widget {
	return widgets.NewSizedBoxSquare(foundation.Some(100.0), child)
}

var m2 = NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false)})

func TestInkWellGesturesControl(t *testing.T) {
	var log []string
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, NewInkWell(nil, InkWellOpts{
		OnTap:         func() { log = append(log, "tap") },
		OnDoubleTap:   func() { log = append(log, "double-tap") },
		OnLongPress:   func() { log = append(log, "long-press") },
		OnLongPressUp: func() { log = append(log, "long-press-up") },
		OnTapDown:     func(gestures.TapDownDetails) { log = append(log, "tap-down") },
		OnTapUp:       func(gestures.TapUpDetails) { log = append(log, "tap-up") },
		OnTapCancel:   func() { log = append(log, "tap-cancel") },
	})))
	check := func(name string, want ...string) {
		t.Helper()
		if !slices.Equal(log, want) {
			t.Errorf("%s: log = %v, want %v", name, log, want)
		}
		log = nil
	}
	finder := gluttertest.ByType[*InkWell]()

	ts.Tap(finder)
	check("tap, before the double tap timeout")
	ts.Pump(time.Second)
	check("tap", "tap-down", "tap-up", "tap")

	ts.Tap(finder)
	ts.Pump(100 * time.Millisecond)
	ts.Tap(finder)
	check("double tap", "double-tap")

	ts.LongPress(finder)
	check("long press", "tap-down", "tap-cancel", "long-press", "long-press-up")

	g := ts.StartGesture(ts.Center(finder))
	ts.Pump(100 * time.Millisecond)
	check("press", "tap-down")
	g.Up()
	ts.Pump(time.Second)
	check("release", "tap-up", "tap")

	g = ts.StartGesture(ts.Center(finder))
	ts.Pump(100 * time.Millisecond)
	g.MoveBy(ui.Offset{Dx: 0, Dy: 200})
	g.Cancel()
	check("cancel", "tap-down", "tap-cancel")
}

func TestInkWellOnlyOnTapDownEnablesGestures(t *testing.T) {
	downTapped := false
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, NewInkWell(nil, InkWellOpts{OnTapDown: func(gestures.TapDownDetails) { downTapped = true }})))
	ts.Tap(gluttertest.ByType[*InkWell]())
	if !downTapped {
		t.Error("onTapDown was not called")
	}
}

func TestInkWellInvokesActivationActions(t *testing.T) {
	var log []string
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(),
		widgets.NewShortcuts(widgets.ShortcutMap{
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeySpace), Intent: widgets.ActivateIntent{}},
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEnter), Intent: widgets.ButtonActivateIntent{}},
		}, NewMaterial(widgets.NewCenter(NewInkWell(nil, InkWellOpts{Autofocus: true, OnTap: func() { log = append(log, "tap") }})))))))
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
	ts.Pump()
	if !slices.Equal(log, []string{"tap"}) {
		t.Errorf("after space: %v", log)
	}
	log = nil
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.Pump()
	if !slices.Equal(log, []string{"tap"}) {
		t.Errorf("after enter: %v", log)
	}
	ts.PumpAndSettle()
}

func TestInkWellActivationDoesNotEndImmediately(t *testing.T) {
	var states []widgets.WidgetStates
	ctl := widgets.NewWidgetStatesController()
	ctl.AddListener(func() { states = append(states, ctl.Value()) })
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(),
		widgets.NewShortcuts(widgets.ShortcutMap{
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeySpace), Intent: widgets.ActivateIntent{}},
		}, NewMaterial(widgets.NewCenter(NewInkWell(nil, InkWellOpts{Autofocus: true, StatesController: ctl, OnTap: func() {}})))))))
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
	ts.Pump()
	if !ctl.Value().Has(widgets.WidgetStatePressed) {
		t.Error("the pressed state ended immediately")
	}
	ts.Pump(200 * time.Millisecond)
	if ctl.Value().Has(widgets.WidgetStatePressed) {
		t.Error("the pressed state never ended")
	}
}

func TestInkWellOnLongPressUp(t *testing.T) {
	called := false
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnLongPress: func() {}, OnLongPressUp: func() { called = true }}))))
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*InkWell]()))
	ts.Pump(time.Second)
	g.Up()
	ts.PumpAndSettle()
	if !called {
		t.Error("onLongPressUp was not called")
	}
}

func TestInkWellDisabledDoesNotThrow(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, NewInkWell(nil)))
	finder := gluttertest.ByType[*InkWell]()
	ts.Tap(finder)
	ts.Pump(time.Second)
	ts.LongPress(finder)
	ts.Pump(time.Second)
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestInkWellChangesColorOnHover(t *testing.T) {
	green, blue := ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)
	tests := []struct {
		name string
		o    InkWellOpts
	}{
		{"hoverColor", InkWellOpts{
			HoverColor: foundation.Some(green), SplashColor: foundation.Some(red), FocusColor: foundation.Some(blue),
			HighlightColor: foundation.Some(ui.NewColor(0xF00FFFFF)),
		}},
		{"overlayColor", InkWellOpts{OverlayColor: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case s.Has(widgets.WidgetStateHovered):
				return foundation.Some(green)
			case s.Has(widgets.WidgetStateFocused):
				return foundation.Some(blue)
			case s.Has(widgets.WidgetStatePressed):
				return foundation.Some(ui.NewColor(0xF00FFFFF))
			}
			return foundation.Some(ui.NewColor(0xFFBADBAD))
		})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hovers []bool
			o := tt.o
			o.OnTap, o.OnLongPress, o.OnLongPressUp = func() {}, func() {}, func() {}
			o.OnHover = func(h bool) { hovers = append(hovers, h) }
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, o))))
			base := pixel(ts, 400, 300)
			ts.Hover(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
			ts.PumpAndSettle()
			if got := pixel(ts, 400, 300); got != rgba(green) {
				t.Errorf("hovered = %v, want %v (base %v)", got, rgba(green), base)
			}
			if got := pixel(ts, 340, 300); got != base {
				t.Errorf("outside the ink well = %v, want %v", got, base)
			}
			ts.Hover(ui.Offset{Dx: 10, Dy: 10})
			ts.PumpAndSettle()
			if got := pixel(ts, 400, 300); got != base {
				t.Errorf("after the pointer left = %v, want %v", got, base)
			}
			if !slices.Equal(hovers, []bool{true, false}) {
				t.Errorf("onHover calls = %v", hovers)
			}
		})
	}
}

func TestInkWellChangesColorOnFocus(t *testing.T) {
	blue := ui.NewColor(0xFF0000FF)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.Binding.FocusManager().SetHighlightStrategy(widgets.FocusHighlightStrategyAlwaysTraditional)
	node := widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Ink Focus"})
	defer node.Dispose()
	var focus []bool
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{
		FocusNode: node, FocusColor: foundation.Some(blue), OnTap: func() {}, OnFocusChange: func(f bool) { focus = append(focus, f) },
	}))))
	ts.PumpAndSettle()
	base := pixel(ts, 400, 300)
	node.RequestFocus()
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != rgba(blue) {
		t.Errorf("focused = %v, want %v (base %v)", got, rgba(blue), base)
	}
	node.Unfocus()
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != base {
		t.Errorf("unfocused = %v, want %v", got, base)
	}
	if !slices.Equal(focus, []bool{true, false}) {
		t.Errorf("onFocusChange calls = %v", focus)
	}
}

func TestInkWellDoesNotFocusOrHoverWhenDisabled(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.Binding.FocusManager().SetHighlightStrategy(widgets.FocusHighlightStrategyAlwaysTraditional)
	node := widgets.NewFocusNode()
	defer node.Dispose()
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{FocusNode: node, FocusColor: foundation.Some(green), HoverColor: foundation.Some(green)}))))
	base := pixel(ts, 400, 300)
	node.RequestFocus()
	ts.PumpAndSettle()
	if node.HasFocus() {
		t.Error("a disabled ink well took the focus")
	}
	ts.Hover(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.PumpAndSettle()
	if got := pixel(ts, 400, 300); got != base {
		t.Errorf("hovered while disabled = %v, want %v", got, base)
	}
}

func TestInkWellSplashIsPainted(t *testing.T) {
	tests := []struct {
		name  string
		theme *ThemeData
	}{
		{"ripple (Material 3)", NewThemeData()},
		{"splash (Material 2)", m2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(inkApp(tt.theme, square(NewInkWell(nil, InkWellOpts{OnTap: func() {}, SplashColor: foundation.Some(ui.NewColor(0xFFFF0000)), HighlightColor: foundation.Some(ui.NewColor(0x00000000))}))))
			base := pixel(ts, 400, 300)
			g := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
			ts.Pump(150 * time.Millisecond) // the tap down fires, the splash starts
			ts.Pump(150 * time.Millisecond)
			mid := pixel(ts, 400, 300)
			if mid == base || mid.R <= mid.G {
				t.Errorf("mid splash = %v, base %v: expected a reddish splash", mid, base)
			}
			// The splash is clipped to the ink well.
			if got := pixel(ts, 330, 300); got != base {
				t.Errorf("outside = %v, want %v", got, base)
			}
			g.Up()
			ts.PumpAndSettle()
			if got := pixel(ts, 400, 300); got != base {
				t.Errorf("after the splash = %v, want %v", got, base)
			}
		})
	}
}

func TestInkWellSplashUsesThemeFactoryAndColor(t *testing.T) {
	var created int
	factory := countingFactory{InkRippleFactory, &created}
	theme := NewThemeData(ThemeDataOpts{SplashFactory: factory, SplashColor: foundation.Some(ui.NewColor(0xFF00FF00))})
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(theme, square(NewInkWell(nil, InkWellOpts{OnTap: func() {}}))))
	ts.Tap(gluttertest.ByType[*widgets.SizedBox]())
	if created != 1 {
		t.Errorf("the theme's factory made %d splashes, want 1", created)
	}
	ts.PumpAndSettle()
}

type countingFactory struct {
	InteractiveInkFeatureFactory
	n *int
}

func (f countingFactory) Create(o InkFeatureCreateOpts) InteractiveInkFeature {
	*f.n++
	return f.InteractiveInkFeatureFactory.Create(o)
}

func TestInkWellSplashColorFromOverlayColor(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(m2, square(NewInkWell(nil, InkWellOpts{OnTap: func() {}, HighlightColor: foundation.Some(ui.NewColor(0x00000000)),
		OverlayColor: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
			if s.Has(widgets.WidgetStatePressed) {
				return foundation.Some(ui.NewColor(0xFF0000FF))
			}
			return foundation.None[ui.Color]()
		})}))))
	ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(150 * time.Millisecond)
	ts.Pump(150 * time.Millisecond)
	if got := pixel(ts, 400, 300); got.B <= got.R {
		t.Errorf("splash = %v, expected bluish", got)
	}
}

func TestInkWellStatesController(t *testing.T) {
	ctl := widgets.NewWidgetStatesController()
	enabled := true
	build := func() widgets.Widget {
		var tap func()
		if enabled {
			tap = func() {}
		}
		return inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnTap: tap, StatesController: ctl})))
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(build())
	if ctl.Value().Has(widgets.WidgetStateDisabled) {
		t.Errorf("states = %v", ctl.Value())
	}
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(50 * time.Millisecond)
	if !ctl.Value().Has(widgets.WidgetStatePressed) {
		t.Errorf("pressed missing from %v", ctl.Value())
	}
	g.Up()
	ts.PumpAndSettle()
	if ctl.Value().Has(widgets.WidgetStatePressed) {
		t.Errorf("pressed left in %v", ctl.Value())
	}
	enabled = false
	ts.PumpWidget(build())
	if !ctl.Value().Has(widgets.WidgetStateDisabled) {
		t.Errorf("disabled missing from %v", ctl.Value())
	}
}

func TestInkWellMouseCursor(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		cursor  services.MouseCursor
		want    string
	}{
		{"enabled, adaptive default", true, nil, "basic"},
		{"enabled, clickable", true, widgets.WidgetStateMouseCursorClickable, "click"},
		{"disabled, clickable", false, widgets.WidgetStateMouseCursorClickable, "basic"},
		{"enabled, explicit", true, services.SystemMouseCursors.Text, "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tap func()
			if tt.enabled {
				tap = func() {}
			}
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnTap: tap, MouseCursor: tt.cursor}))))
			ts.Hover(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
			ts.Pump()
			cursors := ts.ActivatedCursors()
			if len(cursors) == 0 || cursors[len(cursors)-1].Kind != tt.want {
				t.Errorf("cursors = %v, want the last to be %q", cursors, tt.want)
			}
		})
	}
}

func TestInkWellNestedOnlyInnerIsTriggered(t *testing.T) {
	var highlights []string
	padded := func(name string, child widgets.Widget) widgets.Widget {
		return NewInkWell(widgets.NewPadding(painting.NewEdgeInsetsAll(50), child), InkWellOpts{
			OnTap:              func() {},
			OnHighlightChanged: func(v bool) { highlights = append(highlights, name+map[bool]string{true: " on", false: " off"}[v]) },
		})
	}
	inner := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)})
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(m2, padded("outer", padded("middle", padded("inner", inner)))))
	g := ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(200 * time.Millisecond)
	if !slices.Equal(highlights, []string{"inner on"}) {
		t.Errorf("after the press: %v", highlights)
	}
	g.Up()
	ts.PumpAndSettle()
	if !slices.Equal(highlights, []string{"inner on", "inner off"}) {
		t.Errorf("after the release: %v", highlights)
	}

	// Cancelling takes the highlight down too.
	highlights = nil
	g.Down(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(200 * time.Millisecond)
	g.Cancel()
	ts.PumpAndSettle()
	if !slices.Equal(highlights, []string{"inner on", "inner off"}) {
		t.Errorf("after the cancel: %v", highlights)
	}
}

func TestInkWellParentCountsPressedChildren(t *testing.T) {
	// A parent ink well does not splash while any child ink well is pressed, and
	// splashes again afterwards.
	var parent []bool
	child := func(w float64) widgets.Widget {
		return NewInkWell(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(w)}), InkWellOpts{OnTap: func() {}})
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(m2, NewInkWell(widgets.NewPadding(painting.NewEdgeInsetsAll(40), child(40)), InkWellOpts{
		OnTap: func() {}, OnHighlightChanged: func(v bool) { parent = append(parent, v) },
	})))
	g := ts.StartGesture(ts.Center(gluttertest.Last(gluttertest.ByType[*widgets.SizedBox]())))
	ts.Pump(200 * time.Millisecond)
	g.Up()
	ts.PumpAndSettle()
	if len(parent) != 0 {
		t.Errorf("the parent highlighted while its child was pressed: %v", parent)
	}
	// Tapping the padding of the parent works.
	ts.TapAt(ui.Offset{Dx: 400 - 60, Dy: 300})
	ts.PumpAndSettle()
	if len(parent) == 0 {
		t.Error("the parent never highlighted")
	}
}

func TestInkWellSecondaryTap(t *testing.T) {
	var log []string
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{
		OnSecondaryTap:       func() { log = append(log, "secondary tap") },
		OnSecondaryTapDown:   func(gestures.TapDownDetails) { log = append(log, "secondary tap down") },
		OnSecondaryTapUp:     func(gestures.TapUpDetails) { log = append(log, "secondary tap up") },
		OnSecondaryTapCancel: func() { log = append(log, "secondary tap cancel") },
	}))))
	finder := gluttertest.ByType[*widgets.SizedBox]()
	g := ts.StartGesture(ts.Center(finder), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(int(gestures.SecondaryButton))})
	g.Up()
	ts.PumpAndSettle()
	want := []string{"secondary tap down", "secondary tap up", "secondary tap"}
	if !slices.Equal(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
}

func TestInkWellSurvivesDisposalWhileSplashing(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, square(NewInkWell(nil, InkWellOpts{OnTap: func() {}}))))
	ts.StartGesture(ts.Center(gluttertest.ByType[*widgets.SizedBox]()))
	ts.Pump(100 * time.Millisecond)
	ts.PumpWidget(materialApp(nil, NewMaterial(widgets.NewSizedBox(nil))))
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestInkResponseDoesNotCrashInZeroArea(t *testing.T) {
	tests := []struct {
		name string
		w    widgets.Widget
	}{
		{"InkResponse", NewInkResponse(nil, InkResponseOpts{OnTap: func() {}})},
		{"InkWell", NewInkWell(nil, InkWellOpts{OnTap: func() {}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(inkApp(nil, widgets.NewSizedBox(tt.w, widgets.SizedBoxOpts{Width: foundation.Some(0.0), Height: foundation.Some(0.0)})))
			ts.TapAt(ui.Offset{Dx: 400, Dy: 300})
			ts.PumpAndSettle()
			if errs := ts.Errors(); len(errs) != 0 {
				t.Errorf("errors: %v", errs)
			}
		})
	}
}

func TestInkResponseWithoutMaterial(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(materialApp(nil, NewInkWell(nil, InkWellOpts{OnTap: func() {}})))
	errs := ts.TakeErrors()
	if len(errs) == 0 {
		t.Fatal("expected an error")
	}
	found := false
	for _, err := range errs {
		if e, ok := asAppError(err); ok && e.HasRoute("DebugCheckHasMaterial.MaterialMissing") {
			found = true
		}
	}
	if !found {
		t.Errorf("no MaterialMissing error in %v", errs)
	}
}

func TestInkResponseDebugDescription(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(inkApp(nil, NewInkWell(nil, InkWellOpts{OnTap: func() {}, OnLongPress: func() {}})))
	var desc string
	ts.Element(gluttertest.ByType[*InkWell]()).VisitChildren(func(c widgets.Element) {
		desc = foundation.DiagnosticableString(c.Widget().(foundation.Diagnosticable))
	})
	for _, want := range []string{"gestures: [tap, long press]", "clipped to BoxShape.rectangle"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description %q lacks %q", desc, want)
		}
	}
}
