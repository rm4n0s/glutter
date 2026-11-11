package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/button_test.dart. A box of 10 by 10 pixels per
// character stands in for the text of the Ahem font, which the test engine does not have. Not ported: the
// semantics tester cases (the button is a Semantics with Button), the case of the web cursor (no web test
// here), and the case of the icon theme that falls back when the text has no size.

// ahem is a box of one character of the Ahem font: 10 by 10 pixels.
func ahem(characters int) widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(10.0 * float64(characters)), Height: foundation.Some(10.0)})
}

// boilerplate is the widget in a Directionality, centered, in a view of 800 x 600.
func boilerplate(child widgets.Widget) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(
		widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1}, widgets.NewCenter(child)))
}

func buttonSize(ts *gluttertest.Tester) ui.Size {
	return ts.Size(gluttertest.ByType[*cupertino.CupertinoButton]())
}

func TestCupertinoButtonLayout(t *testing.T) {
	for _, tt := range []struct {
		name   string
		button func() widgets.Widget
		want   ui.Size
	}{
		{"the default minimum size", func() widgets.Widget { return cupertino.NewCupertinoButton(nil, ahem(1)) }, ui.Size{Width: 50, Height: 44}},
		{"a minimum size", func() widgets.Widget {
			return cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{MinimumSize: foundation.Some(ui.Size{Width: 60, Height: 100})})
		}, ui.Size{Width: 60, Height: 100}},
		{"the size grows with the child", func() widgets.Widget { return cupertino.NewCupertinoButton(nil, ahem(4)) }, ui.Size{Width: 80, Height: 44}},
		{"the small size", func() widgets.Widget {
			return cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{SizeStyle: cupertino.CupertinoButtonSizeSmall})
		}, ui.Size{Width: 34, Height: 28}},
		{"the medium size", func() widgets.Widget {
			return cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{SizeStyle: cupertino.CupertinoButtonSizeMedium})
		}, ui.Size{Width: 40, Height: 32}},
		{"the large size", func() widgets.Widget {
			return cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{SizeStyle: cupertino.CupertinoButtonSizeLarge})
		}, ui.Size{Width: 50, Height: 44}},
		{"custom padding", func() widgets.Widget {
			return cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{Padding: painting.NewEdgeInsetsAll(100)})
		}, ui.Size{Width: 210, Height: 210}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(boilerplate(tt.button()))
			if got := buttonSize(ts); got != tt.want {
				t.Errorf("size = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("the size changes with the size style", func(t *testing.T) {
		ts := gluttertest.New(t)
		for _, s := range []struct {
			size cupertino.CupertinoButtonSize
			want ui.Size
		}{{cupertino.CupertinoButtonSizeSmall, ui.Size{Width: 34, Height: 28}}, {cupertino.CupertinoButtonSizeMedium, ui.Size{Width: 40, Height: 32}}, {cupertino.CupertinoButtonSizeUnspecified, ui.Size{Width: 50, Height: 44}}} {
			ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{SizeStyle: s.size})))
			if got := buttonSize(ts); got != s.want {
				t.Errorf("%v: size = %v, want %v", s.size, got, s.want)
			}
		}
	})

	t.Run("the child is aligned", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{
			MinimumSize: foundation.Some(ui.Size{Width: 200, Height: 100}), Alignment: painting.AlignmentTopLeft, Padding: painting.NewEdgeInsetsAll(0)})))
		button := ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoButton]())
		child := ts.TopLeft(gluttertest.ByType[*widgets.SizedBox]())
		if child != button {
			t.Errorf("the child is at %v, want the top left of the button %v", child, button)
		}
	})
}

func TestCupertinoButtonEnabled(t *testing.T) {
	for _, tt := range []struct {
		name   string
		button *cupertino.CupertinoButton
		want   bool
	}{
		{"neither callback", cupertino.NewCupertinoButton(nil, ahem(4)), false},
		{"onPressed", cupertino.NewCupertinoButton(func() {}, ahem(4)), true},
		{"onLongPress", cupertino.NewCupertinoButton(nil, ahem(4), cupertino.CupertinoButtonOpts{OnLongPress: func() {}}), true},
	} {
		if tt.button.Enabled() != tt.want {
			t.Errorf("%s: enabled = %v, want %v", tt.name, tt.button.Enabled(), tt.want)
		}
	}
}

func TestCupertinoButtonGestures(t *testing.T) {
	t.Run("a long press works", func(t *testing.T) {
		ts := gluttertest.New(t)
		value := false
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(4), cupertino.CupertinoButtonOpts{OnLongPress: func() { value = !value }})))
		ts.Pump()
		ts.LongPress(gluttertest.ByType[*cupertino.CupertinoButton]())
		if !value {
			t.Error("the long press was not called")
		}
	})

	t.Run("it takes taps and animates", func(t *testing.T) {
		ts := gluttertest.New(t)
		value := false
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() { value = true }, widgets.NewText("Tap me"))))
		sched := ts.Binding.Scheduler()
		if value || sched.TransientCallbackCount() != 0 {
			t.Fatalf("value %v, %d transient callbacks before the tap", value, sched.TransientCallbackCount())
		}
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoButton]())
		if !value {
			t.Error("the button was not pressed")
		}
		if sched.TransientCallbackCount() != 1 {
			t.Errorf("%d transient callbacks after the tap, want 1", sched.TransientCallbackCount())
		}
	})

	t.Run("a disabled button does not animate", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, widgets.NewText("Tap me"))))
		sched := ts.Binding.Scheduler()
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoButton]())
		if sched.TransientCallbackCount() != 0 {
			t.Errorf("%d transient callbacks after the tap, want 0", sched.TransientCallbackCount())
		}
	})

	t.Run("an enabled button animates", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, widgets.NewText("Tap me"))))
		ts.Tap(gluttertest.ByType[*cupertino.CupertinoButton]())
		ts.Pump() // The enter animation.
		for _, s := range []struct {
			ms   int
			want float64
		}{{50, 0.403}, {100, 0.400}, {50, 0.650}, {50, 0.894}, {50, 0.988}, {50, 1.0}} {
			ts.Pump(time.Duration(s.ms) * time.Millisecond)
			opacity := ts.Widget(gluttertest.ByType[*widgets.FadeTransition]()).(*widgets.FadeTransition).Opacity().Value()
			requireNear(t, "the opacity", opacity, s.want, 0.001)
		}
	})

	for _, tt := range []struct {
		name    string
		opacity foundation.Opt[float64]
		want    float64
	}{{"pressedOpacity defaults to 0.4", foundation.None[float64](), 0.4}, {"pressedOpacity can be given", foundation.Some(0.5), 0.5}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, widgets.NewText("Tap me"), cupertino.CupertinoButtonOpts{PressedOpacity: tt.opacity})))
			gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*cupertino.CupertinoButton]()))
			ts.PumpAndSettle()
			opacity := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.FadeTransition]())).(*widgets.FadeTransition)
			if got := opacity.Opacity().Value(); got != tt.want {
				t.Errorf("opacity = %v, want %v", got, tt.want)
			}
			gesture.Up()
			ts.PumpAndSettle()
		})
	}

	t.Run("pressing and moving out of the button, and back", func(t *testing.T) {
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformIOS} {
			skipUnlessDebug(t)
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
			ts := gluttertest.New(t)
			ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, widgets.NewText("Tap me"))))
			gesture := ts.StartGesture(ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoButton]()))
			opacity := func() float64 {
				return ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.FadeTransition]())).(*widgets.FadeTransition).Opacity().Value()
			}
			ts.PumpAndSettle()
			if got := opacity(); got != 0.4 {
				t.Errorf("%v: opacity = %v, want 0.4", platform, got)
			}
			slop := cupertino.CupertinoButtonTapMoveSlopOf()
			gesture.MoveBy(ui.Offset{Dy: -slop + 1})
			ts.PumpAndSettle()
			if got := opacity(); got != 0.4 {
				t.Errorf("%v: opacity inside the slop = %v, want 0.4", platform, got)
			}
			gesture.MoveBy(ui.Offset{Dy: -2})
			ts.PumpAndSettle()
			if got := opacity(); got != 1.0 {
				t.Errorf("%v: opacity outside the slop = %v, want 1.0", platform, got)
			}
			gesture.MoveBy(ui.Offset{Dy: 1})
			ts.PumpAndSettle()
			if got := opacity(); got != 0.4 {
				t.Errorf("%v: opacity back inside = %v, want 0.4", platform, got)
			}
			gesture.Up()
			ts.PumpAndSettle()
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		}
	})

	t.Run("a press that drags out within a ListView does not leave the button pressed", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(boilerplate(widgets.NewListView([]widgets.Widget{cupertino.NewCupertinoButton(func() {}, widgets.NewText("Tap me")),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)})})))
		opacity := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.FadeTransition]())).(*widgets.FadeTransition)
		gesture := ts.StartGesture(ts.TopLeft(gluttertest.ByType[*cupertino.CupertinoButton]()))
		gesture.MoveBy(ui.Offset{Dx: 1, Dy: 1})
		gesture.MoveBy(ui.Offset{Dy: -cupertino.CupertinoButtonTapMoveSlopOf() - 5})
		ts.PumpAndSettle()
		if got := opacity.Opacity().Value(); got != 1.0 {
			t.Errorf("opacity = %v, want 1.0", got)
		}
		gesture.Up()
	})

	t.Run("onPressed takes the move slop into account", func(t *testing.T) {
		skipUnlessDebug(t)
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
		defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		ts := gluttertest.New(t)
		value := false
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() { value = true }, widgets.NewText("Tap me"))))
		button := gluttertest.ByType[*cupertino.CupertinoButton]()
		gesture := ts.StartGesture(ts.Center(button))
		gesture.MoveTo(ts.BottomRight(button).Add(ui.Offset{Dy: cupertino.CupertinoButtonTapMoveSlopOf()}))
		gesture.Up()
		if value {
			t.Error("the button was pressed outside the slop")
		}
		gesture = ts.StartGesture(ts.TopLeft(button))
		gesture.MoveTo(ts.BottomRight(button).Add(ui.Offset{Dy: cupertino.CupertinoButtonTapMoveSlopOf()}))
		gesture.MoveBy(ui.Offset{Dy: -1})
		gesture.Up()
		if !value {
			t.Error("the button was not pressed inside the slop")
		}
	})
}

func TestCupertinoButtonColors(t *testing.T) {
	decorationOf := func(ts *gluttertest.Tester) *painting.ShapeDecoration {
		box := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
		return box.Decoration().(*painting.ShapeDecoration)
	}
	t.Run("the colors can be given", func(t *testing.T) {
		ts := gluttertest.New(t)
		opts := cupertino.CupertinoButtonOpts{Color: ui.NewColor(0x000000FF), DisabledColor: ui.NewColor(0x0000FF00)}
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, ahem(1), opts)))
		if got := decorationOf(ts).Color; got != foundation.Some(ui.NewColor(0x000000FF)) {
			t.Errorf("enabled color = %v", got)
		}
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(1), opts)))
		if got := decorationOf(ts).Color; got != foundation.Some(ui.NewColor(0x0000FF00)) {
			t.Errorf("disabled color = %v", got)
		}
	})

	t.Run("the colors can be dynamic", func(t *testing.T) {
		ts := gluttertest.New(t)
		bg := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF123456), ui.NewColor(0xFF654321))
		inactive := cupertino.NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF111111), ui.NewColor(0xFF222222))
		opts := cupertino.CupertinoButtonOpts{Color: bg, DisabledColor: inactive}
		dark := widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewMediaQuery(
			widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1, PlatformBrightness: ui.BrightnessDark},
			widgets.NewCenter(cupertino.NewCupertinoButton(func() {}, ahem(1), opts))))
		ts.PumpWidget(dark)
		if got := decorationOf(ts).Color; got != foundation.Some(ui.NewColor(0xFF654321)) {
			t.Errorf("dark enabled color = %v", got)
		}
		ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(1), opts)))
		if got := decorationOf(ts).Color; got != foundation.Some(ui.NewColor(0xFF111111)) {
			t.Errorf("light disabled color = %v", got)
		}
	})

	t.Run("the button respects the theme", func(t *testing.T) {
		blue, systemBlue := cupertino.CupertinoColors.ActiveBlue, cupertino.CupertinoColors.SystemBlue
		white := cupertino.CupertinoColors.White
		for _, tt := range []struct {
			name       string
			brightness ui.Brightness
			make       func(onPressed func(), child widgets.Widget, opts ...cupertino.CupertinoButtonOpts) *cupertino.CupertinoButton
			text       ui.Color
			background foundation.Opt[ui.Color]
		}{
			{"plain, light", ui.BrightnessLight, cupertino.NewCupertinoButton, blue.Color, foundation.None[ui.Color]()},
			{"tinted, light", ui.BrightnessLight, cupertino.NewCupertinoButtonTinted, blue.Color, foundation.Some(blue.Color.WithOpacity(0.12))},
			{"filled, light", ui.BrightnessLight, cupertino.NewCupertinoButtonFilled, white, foundation.Some(blue.Color)},
			{"plain, dark", ui.BrightnessDark, cupertino.NewCupertinoButton, systemBlue.DarkColor, foundation.None[ui.Color]()},
			{"tinted, dark", ui.BrightnessDark, cupertino.NewCupertinoButtonTinted, systemBlue.DarkColor, foundation.Some(blue.DarkColor.WithOpacity(0.26))},
			{"filled, dark", ui.BrightnessDark, cupertino.NewCupertinoButtonFilled, white, foundation.Some(systemBlue.DarkColor)},
		} {
			t.Run(tt.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				var textStyle *painting.TextStyle
				ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
					Home: tt.make(func() {}, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
						textStyle = widgets.DefaultTextStyleOf(bc).Style()
						return widgets.NewPlaceholder(nil)
					}))}))
				if got := textStyle.Color; got != foundation.Some(tt.text) {
					t.Errorf("text color = %v, want %v", got, tt.text)
				}
				if got := decorationOf(ts).Color; got != tt.background {
					t.Errorf("background = %v, want %v", got, tt.background)
				}
			})
		}
	})

	t.Run("a filled button has the color it is given", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoButtonFilled(func() {}, widgets.NewPlaceholder(nil),
			cupertino.CupertinoButtonOpts{Color: cupertino.CupertinoColors.SystemRed})}))
		if got, want := decorationOf(ts).Color, foundation.Some(cupertino.CupertinoColors.SystemRed.Color); got != want {
			t.Errorf("background = %v, want %v", got, want)
		}
	})

	t.Run("the foreground color is the one of the text and of the icons", func(t *testing.T) {
		ts := gluttertest.New(t)
		var style *painting.TextStyle
		var icon widgets.IconThemeData
		for _, tt := range []struct {
			name string
			make func(func(), widgets.Widget, ...cupertino.CupertinoButtonOpts) *cupertino.CupertinoButton
			on   bool
		}{{"plain", cupertino.NewCupertinoButton, true}, {"filled", cupertino.NewCupertinoButtonFilled, true}, {"disabled", cupertino.NewCupertinoButton, false}} {
			var onPressed func()
			if tt.on {
				onPressed = func() {}
			}
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: tt.make(onPressed, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				style, icon = widgets.DefaultTextStyleOf(bc).Style(), widgets.IconThemeOf(bc)
				return widgets.NewPlaceholder(nil)
			}), cupertino.CupertinoButtonOpts{ForegroundColor: ui.NewColor(0xFF00FF00)})}))
			if style.Color != foundation.Some(ui.NewColor(0xFF00FF00)) || icon.Color() != foundation.Some(ui.NewColor(0xFF00FF00)) {
				t.Errorf("%s: text color %v, icon color %v", tt.name, style.Color, icon.Color())
			}
		}
	})
}

func TestCupertinoButtonFocus(t *testing.T) {
	setup := func(t *testing.T, opts cupertino.CupertinoButtonOpts, onPressed func()) (*gluttertest.Tester, *widgets.FocusNode) {
		ts := gluttertest.New(t)
		focusNode := widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Button"})
		t.Cleanup(focusNode.Dispose)
		ts.Binding.FocusManager().SetHighlightStrategy(widgets.FocusHighlightStrategyAlwaysTraditional)
		opts.FocusNode = focusNode
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoButton(onPressed, widgets.NewText("Tap me"), opts))}))
		return ts, focusNode
	}
	borderOf := func(ts *gluttertest.Tester) painting.BorderSide {
		box := ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.DecoratedBox]())).(*widgets.DecoratedBox)
		return box.Decoration().(*painting.ShapeDecoration).Shape.(painting.OutlinedBorder).Side()
	}

	t.Run("it can be focused and has the default colors", func(t *testing.T) {
		ts, node := setup(t, cupertino.CupertinoButtonOpts{Autofocus: true}, func() {})
		if !node.HasPrimaryFocus() {
			t.Fatal("the button has no focus")
		}
		if got := borderOf(ts); got != painting.BorderSideNone {
			t.Errorf("the button has the border %v, want none", got)
		}
		node.RequestFocus()
		ts.PumpAndSettle()
		want := painting.NewBorderSide(painting.BorderSideOpts{
			Color: foundation.Some(painting.NewHSLColorFromColor(cupertino.CupertinoColors.ActiveBlue.Color.WithOpacity(cupertino.CupertinoFocusColorOpacity)).
				WithLightness(cupertino.CupertinoFocusColorBrightness).WithSaturation(cupertino.CupertinoFocusColorSaturation).ToColor()),
			Width: foundation.Some(3.5), StrokeAlign: foundation.Some(painting.StrokeAlignOutside)})
		if got := borderOf(ts); got != want {
			t.Errorf("the focused button has the border %v, want %v", got, want)
		}
	})

	t.Run("the focus color can be given", func(t *testing.T) {
		ts, node := setup(t, cupertino.CupertinoButtonOpts{Autofocus: true, FocusColor: cupertino.CupertinoColors.SystemGreen}, func() {})
		node.RequestFocus()
		ts.PumpAndSettle()
		if got, want := borderOf(ts).Color, cupertino.CupertinoColors.SystemGreen.Color; got != want {
			t.Errorf("the border color is %v, want %v", got, want)
		}
	})

	t.Run("OnFocusChange is called", func(t *testing.T) {
		focused := false
		ts, node := setup(t, cupertino.CupertinoButtonOpts{OnFocusChange: func(v bool) { focused = v }}, func() {})
		node.RequestFocus()
		ts.Pump()
		if !focused || !node.HasFocus() {
			t.Errorf("focused %v, has focus %v", focused, node.HasFocus())
		}
		node.Unfocus()
		ts.Pump()
		if focused || node.HasFocus() {
			t.Errorf("focused %v, has focus %v", focused, node.HasFocus())
		}
	})

	t.Run("it is activated by the enter key (not on the web) and the space key", func(t *testing.T) {
		value := true
		var ts *gluttertest.Tester
		ts, _ = setup(t, cupertino.CupertinoButtonOpts{Autofocus: true}, func() { value = !value })
		ts.Pump()
		// On the web Flutter's buttons do not answer the enter key, as its default shortcuts there have no
		// enter; the web shortcuts of WidgetsApp are not ported (SPEC §12.10), so the web skips the two enters.
		if !foundation.IsWeb {
			ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
			ts.Pump()
			if value {
				t.Errorf("after enter: %v, want false", value)
			}
			ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
			ts.Pump()
			if !value {
				t.Error("after a second enter the value is false")
			}
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if value {
			t.Error("after space the value is true")
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if !value {
			t.Error("after a second space the value is false")
		}
	})
}

func TestCupertinoButtonMouseCursor(t *testing.T) {
	cursorOf := func(ts *gluttertest.Tester) services.MouseCursor {
		return ts.Widget(gluttertest.Descendant(gluttertest.ByType[*cupertino.CupertinoButton](), gluttertest.ByType[*widgets.MouseRegion]())).(*widgets.MouseRegion).Cursor()
	}
	ts := gluttertest.New(t)
	ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, ahem(1))))
	// The region is above the button's descendants, so look it up from its own type.
	region := ts.Widgets(gluttertest.ByType[*widgets.MouseRegion]())[0].(*widgets.MouseRegion)
	if got, want := region.Cursor(), services.MouseCursorDefer; got != want && !foundation.IsWeb {
		t.Errorf("the default cursor is %v, want %v", got, want)
	}
	_ = cursorOf
	cursor := widgets.WidgetStateMouseCursorResolveWith(func(states widgets.WidgetStates) services.MouseCursor {
		switch {
		case states.Has(widgets.WidgetStateDisabled):
			return services.SystemMouseCursors.Forbidden
		case states.Has(widgets.WidgetStatePressed):
			return services.SystemMouseCursors.Grab
		case states.Has(widgets.WidgetStateFocused):
			return services.SystemMouseCursors.Text
		}
		return services.SystemMouseCursors.Click
	})
	ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{MouseCursor: cursor})))
	if got := ts.Widgets(gluttertest.ByType[*widgets.MouseRegion]())[0].(*widgets.MouseRegion).Cursor(); got != services.SystemMouseCursors.Forbidden {
		t.Errorf("the disabled cursor is %v", got)
	}
	ts.PumpWidget(boilerplate(cupertino.NewCupertinoButton(func() {}, ahem(1), cupertino.CupertinoButtonOpts{MouseCursor: cursor})))
	if got := ts.Widgets(gluttertest.ByType[*widgets.MouseRegion]())[0].(*widgets.MouseRegion).Cursor(); got != services.SystemMouseCursors.Click {
		t.Errorf("the enabled cursor is %v", got)
	}
	gesture := ts.StartGesture(ts.Center(gluttertest.ByType[*cupertino.CupertinoButton]()))
	ts.Pump()
	if got := ts.Widgets(gluttertest.ByType[*widgets.MouseRegion]())[0].(*widgets.MouseRegion).Cursor(); got != services.SystemMouseCursors.Grab {
		t.Errorf("the pressed cursor is %v", got)
	}
	gesture.Up()
}

func TestCupertinoButtonSizeMaps(t *testing.T) {
	for _, size := range []cupertino.CupertinoButtonSize{cupertino.CupertinoButtonSizeUnspecified, cupertino.CupertinoButtonSizeSmall, cupertino.CupertinoButtonSizeMedium, cupertino.CupertinoButtonSizeLarge} {
		if p := cupertino.CupertinoButtonPadding(size); p == (painting.EdgeInsets{}) {
			t.Errorf("%v: no padding", size)
		}
		if r := cupertino.CupertinoButtonSizeBorderRadius(size); r == painting.BorderRadiusZero {
			t.Errorf("%v: no border radius", size)
		}
		if cupertino.CupertinoButtonMinSize(size) <= 0 {
			t.Errorf("%v: no minimum size", size)
		}
	}
	if cupertino.CupertinoButtonSizeUnspecified.Name() != "large" || cupertino.CupertinoButtonSizeSmall.Name() != "small" {
		t.Error("the names of the sizes are wrong")
	}
}

func TestCupertinoButtonPressedOpacityMustBeInRange(t *testing.T) {
	skipUnlessDebug(t)
	for _, v := range []float64{-0.1, 1.1} {
		err := recoverError(func() {
			cupertino.NewCupertinoButton(nil, ahem(1), cupertino.CupertinoButtonOpts{PressedOpacity: foundation.Some(v)})
		})
		checkErrorTag(t, err, "newCupertinoButton.CupertinoButtonPressedOpacityInvalid")
	}
}

func TestCupertinoButtonDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(boilerplate(widgets.NewSizedBoxShrink(cupertino.NewCupertinoButton(func() {}, widgets.NewText("X")))))
	if size := buttonSize(ts); size != (ui.Size{}) {
		t.Errorf("size = %v, want zero", size)
	}
}
