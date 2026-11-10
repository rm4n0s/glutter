package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/switch_test.dart. The cases that read the calls of
// the canvas with a `paints` matcher (the colors of the track and of the thumb, the labels, the outline) are goldens
// here (TestCupertinoSwitchGoldens), and the cases of the semantics tester are not ported.

// switchApp is an app whose home has a switch of the given value that keeps it, under a Directionality.
func switchApp(value *bool, opts cupertino.CupertinoSwitchOpts, direction ui.TextDirection) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewDirectionality(direction, widgets.NewStatefulBuilder(
		func(bc widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return widgets.NewCenter(cupertino.NewCupertinoSwitch(*value, func(v bool) { setState(func() { *value = v }) }, opts))
		}))})
}

func TestCupertinoSwitch(t *testing.T) {
	sw := gluttertest.ByType[*cupertino.CupertinoSwitch]()
	t.Run("it is 59 by 39", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		if got, want := ts.Size(sw), (ui.Size{Width: 59, Height: 39}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
	})

	t.Run("a tap turns it", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		ts.Tap(sw)
		if !v {
			t.Error("the switch is still off")
		}
		ts.PumpAndSettle()
		ts.Tap(sw)
		if v {
			t.Error("the switch is still on")
		}
	})

	t.Run("the keyboard turns it", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{Autofocus: true}, ui.TextDirectionLTR))
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if !v {
			t.Error("space did not turn the switch on")
		}
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if v {
			t.Error("space did not turn the switch off")
		}
	})

	for _, tt := range []struct {
		name      string
		direction ui.TextDirection
		by        float64
	}{{"a drag to the end turns it (LTR)", ui.TextDirectionLTR, 60}, {"a drag to the end turns it (RTL)", ui.TextDirectionRTL, -60}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			v := false
			ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, tt.direction))
			ts.Drag(sw, ui.Offset{Dx: tt.by})
			ts.PumpAndSettle()
			if !v {
				t.Error("the switch is still off")
			}
			ts.Drag(sw, ui.Offset{Dx: -tt.by})
			ts.PumpAndSettle()
			if v {
				t.Error("the switch is still on")
			}
		})
	}

	t.Run("a drag the wrong way does nothing, and a short one is below the threshold", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		ts.Drag(sw, ui.Offset{Dx: -60})
		ts.PumpAndSettle()
		if v {
			t.Error("a drag to the start turned an off switch on")
		}
		ts.Drag(sw, ui.Offset{Dx: 20}) // 20 of 51 is under 0.7 of the track.
		ts.PumpAndSettle()
		if v {
			t.Error("a short drag turned the switch")
		}
	})

	t.Run("a drag turns back when it goes back past the reverse threshold", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		gesture := ts.StartGesture(ts.Center(sw))
		gesture.MoveBy(ui.Offset{Dx: 45})
		ts.Pump()
		gesture.MoveBy(ui.Offset{Dx: -50})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		if v {
			t.Error("a drag that went back turned the switch")
		}
	})

	t.Run("the app can veto the result of a drag", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := false
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSwitch(v, func(bool) {}))}))
		ts.Drag(sw, ui.Offset{Dx: 60})
		ts.PumpAndSettle()
		if ts.Widget(sw).(*cupertino.CupertinoSwitch).Value() {
			t.Error("the switch changed its value by itself")
		}
	})

	t.Run("a disabled switch is translucent and does not turn", func(t *testing.T) {
		ts := gluttertest.New(t)
		opacityOf := func() float64 {
			return ts.Widget(gluttertest.Descendant(sw, gluttertest.ByType[*widgets.Opacity]())).(*widgets.Opacity).Opacity()
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSwitch(false, nil))}))
		if got := opacityOf(); got != 0.5 {
			t.Errorf("disabled opacity = %v, want 0.5", got)
		}
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSwitch(false, func(bool) {}))}))
		if got := opacityOf(); got != 1 {
			t.Errorf("enabled opacity = %v, want 1", got)
		}
	})

	t.Run("a haptic feedback on iOS when it turns and when a drag crosses", func(t *testing.T) {
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformIOS))
		defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		ts := gluttertest.New(t)
		haptics := 0
		ts.Binding.HapticFeedback().OnPerform = func(ui.HapticFeedbackKind) { haptics++ }
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		ts.Tap(sw)
		if haptics != 1 {
			t.Errorf("haptics after a tap = %d, want 1", haptics)
		}
		ts.PumpAndSettle()
		ts.Drag(sw, ui.Offset{Dx: -60})
		ts.PumpAndSettle()
		if haptics != 2 {
			t.Errorf("haptics after a drag = %d, want 2", haptics)
		}
	})

	t.Run("no haptic feedback away from iOS", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t)
		haptics := 0
		ts.Binding.HapticFeedback().OnPerform = func(ui.HapticFeedbackKind) { haptics++ }
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{}, ui.TextDirectionLTR))
		ts.Tap(sw)
		if haptics != 0 {
			t.Errorf("haptics = %d, want 0", haptics)
		}
	})

	t.Run("the focus and its callback", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Switch"})
		t.Cleanup(node.Dispose)
		focused := false
		v := false
		ts.PumpWidget(switchApp(&v, cupertino.CupertinoSwitchOpts{FocusNode: node, OnFocusChange: func(f bool) { focused = f }}, ui.TextDirectionLTR))
		node.RequestFocus()
		ts.Pump()
		if !focused || !node.HasFocus() {
			t.Errorf("focused %v, has focus %v", focused, node.HasFocus())
		}
		node.Unfocus()
		ts.Pump()
		if focused {
			t.Error("the callback still says focused")
		}
	})

	t.Run("the mouse cursor can be set", func(t *testing.T) {
		ts := gluttertest.New(t)
		cursor := widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) services.MouseCursor {
			if st.Has(widgets.WidgetStateDisabled) {
				return services.SystemMouseCursors.Forbidden
			}
			return services.SystemMouseCursors.Grab
		})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(cupertino.NewCupertinoSwitch(false, nil, cupertino.CupertinoSwitchOpts{MouseCursor: cursor}))}))
		if got := ts.Widgets(gluttertest.Descendant(sw, gluttertest.ByType[*widgets.MouseRegion]()))[0].(*widgets.MouseRegion).Cursor(); got != services.SystemMouseCursors.Forbidden {
			t.Errorf("disabled cursor = %v", got)
		}
	})

	t.Run("it does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoSwitch(true, func(bool) {})))}))
	})
}

func TestCupertinoSwitchErrors(t *testing.T) {
	skipUnlessDebug(t)
	requireErrorTag(t, recoverError(func() {
		cupertino.NewCupertinoSwitch(true, nil, cupertino.CupertinoSwitchOpts{OnActiveThumbImageError: func(error) {}})
	}), "NewCupertinoSwitch.CupertinoSwitchActiveThumbImageErrorWithoutImage")
	requireErrorTag(t, recoverError(func() {
		cupertino.NewCupertinoSwitch(true, nil, cupertino.CupertinoSwitchOpts{OnInactiveThumbImageError: func(error) {}})
	}), "NewCupertinoSwitch.CupertinoSwitchInactiveThumbImageErrorWithoutImage")
}

// TestCupertinoSwitchGoldens draws the switch in its states: off, on, disabled, with the on and off labels, with the
// focus outline, with an icon in the thumb, in the dark, and right to left.
func TestCupertinoSwitchGoldens(t *testing.T) {
	cell := func(value bool, opts cupertino.CupertinoSwitchOpts) widgets.Widget {
		return widgets.NewPadding(painting.NewEdgeInsetsAll(8), cupertino.NewCupertinoSwitch(value, func(bool) {}, opts))
	}
	disabled := widgets.NewPadding(painting.NewEdgeInsetsAll(8), cupertino.NewCupertinoSwitch(true, nil))
	icon := widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) *widgets.Icon {
		if st.Has(widgets.WidgetStateSelected) {
			return widgets.NewIcon(cupertino.CupertinoIcons.CheckMark())
		}
		return widgets.NewIcon(cupertino.CupertinoIcons.Clear())
	})
	row := func(children ...widgets.Widget) widgets.Widget {
		return widgets.NewRow(children, widgets.RowOpts{MainAxisSize: 1})
	}
	for _, tt := range []struct {
		name string
		page widgets.Widget
	}{
		{"cupertino.switch", widgets.NewColumn([]widgets.Widget{
			row(cell(false, cupertino.CupertinoSwitchOpts{}), cell(true, cupertino.CupertinoSwitchOpts{}), disabled),
			row(cell(false, cupertino.CupertinoSwitchOpts{ThumbIcon: icon}), cell(true, cupertino.CupertinoSwitchOpts{ThumbIcon: icon}),
				cell(true, cupertino.CupertinoSwitchOpts{ActiveTrackColor: cupertino.CupertinoColors.SystemPink, ThumbColor: cupertino.CupertinoColors.SystemYellow})),
		}, widgets.ColumnOpts{MainAxisSize: 1})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 240, Height: 130}})
			ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(tt.page))}))
			ts.PumpAndSettle()
			ts.MatchesGolden(tt.name)
		})
	}
	t.Run("cupertino.switch.labels.dark", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 160, Height: 60}})
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Theme: testTheme(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark}),
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewMediaQuery(widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{OnOffSwitchLabels: foundation.Some(true)}),
					cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewRow([]widgets.Widget{
						cell(false, cupertino.CupertinoSwitchOpts{}), cell(true, cupertino.CupertinoSwitchOpts{})}, widgets.RowOpts{MainAxisSize: 1}))))
			})}))
		ts.PumpAndSettle()
		ts.MatchesGolden("cupertino.switch.labels.dark")
	})
}
