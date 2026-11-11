package cupertino_test

import (
	"math"
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

// These tests are ported from packages/flutter/test/cupertino/{checkbox,radio,slider,activity_indicator}_test.dart.
// The cases that read the calls of the canvas with a `paints` matcher are goldens here (TestCupertinoControlsGoldens),
// and the cases of the semantics tester are not ported.

func centered(w widgets.Widget) widgets.Widget {
	return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(w)})
}

func TestCupertinoCheckbox(t *testing.T) {
	cb := gluttertest.ByType[*cupertino.CupertinoCheckbox]()
	checkboxApp := func(value *foundation.Opt[bool], opts cupertino.CupertinoCheckboxOpts) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return widgets.NewCenter(cupertino.NewCupertinoCheckbox(*value, func(v foundation.Opt[bool]) { setState(func() { *value = v }) }, opts))
		})})
	}
	t.Run("a tap turns it on and off", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := foundation.Some(false)
		ts.PumpWidget(checkboxApp(&v, cupertino.CupertinoCheckboxOpts{}))
		ts.Tap(cb)
		if v != foundation.Some(true) {
			t.Errorf("after the first tap: %v", v)
		}
		ts.PumpAndSettle()
		ts.Tap(cb)
		if v != foundation.Some(false) {
			t.Errorf("after the second tap: %v", v)
		}
	})
	t.Run("a tristate checkbox goes through the mixed value", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := foundation.Some(false)
		ts.PumpWidget(checkboxApp(&v, cupertino.CupertinoCheckboxOpts{Tristate: true}))
		var seen []foundation.Opt[bool]
		for range 3 {
			ts.Tap(cb)
			ts.PumpAndSettle()
			seen = append(seen, v)
		}
		want := []foundation.Opt[bool]{foundation.Some(true), foundation.None[bool](), foundation.Some(false)}
		for i := range want {
			if seen[i] != want[i] {
				t.Errorf("value after tap %d = %v, want %v", i+1, seen[i], want[i])
			}
		}
	})
	t.Run("a disabled checkbox ignores the taps", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoCheckbox(foundation.Some(false), nil)))
		ts.Tap(cb)
		ts.PumpAndSettle()
		if ts.Widget(cb).(*cupertino.CupertinoCheckbox).Value() != foundation.Some(false) {
			t.Error("a disabled checkbox changed")
		}
	})
	t.Run("the keyboard turns it", func(t *testing.T) {
		ts := gluttertest.New(t)
		v := foundation.Some(false)
		ts.PumpWidget(checkboxApp(&v, cupertino.CupertinoCheckboxOpts{Autofocus: true}))
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if v != foundation.Some(true) {
			t.Errorf("space left the value at %v", v)
		}
	})
	t.Run("the tap target is the size that is asked for", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoCheckbox(foundation.Some(true), func(foundation.Opt[bool]) {})))
		if got := ts.Size(cb); got.Width < cupertino.CupertinoCheckboxWidth || got.Height < cupertino.CupertinoCheckboxWidth {
			t.Errorf("size = %v", got)
		}
		ts.PumpWidget(centered(cupertino.NewCupertinoCheckbox(foundation.Some(true), func(foundation.Opt[bool]) {},
			cupertino.CupertinoCheckboxOpts{TapTargetSize: foundation.Some(ui.Size{Width: 60, Height: 70})})))
		if got, want := ts.Size(cb), (ui.Size{Width: 60, Height: 70}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
	})
	t.Run("the focus node is the one that is given", func(t *testing.T) {
		ts := gluttertest.New(t)
		node := widgets.NewFocusNode()
		t.Cleanup(node.Dispose)
		ts.PumpWidget(centered(cupertino.NewCupertinoCheckbox(foundation.Some(true), func(foundation.Opt[bool]) {}, cupertino.CupertinoCheckboxOpts{FocusNode: node})))
		node.RequestFocus()
		ts.Pump()
		if !node.HasFocus() {
			t.Error("the node has no focus")
		}
	})
	t.Run("the error of a null value", func(t *testing.T) {
		skipUnlessDebug(t)
		requireErrorTag(t, recoverError(func() { cupertino.NewCupertinoCheckbox(foundation.None[bool](), nil) }), "NewCupertinoCheckbox.CupertinoCheckboxValueNull")
	})
}

func TestCupertinoRadio(t *testing.T) {
	type choice int
	radio := func(v choice, opts ...cupertino.CupertinoRadioOpts[choice]) widgets.Widget {
		return cupertino.NewCupertinoRadio(v, opts...)
	}
	group := func(value *foundation.Opt[choice], children ...widgets.Widget) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return widgets.NewCenter(widgets.NewRadioGroup(func(v foundation.Opt[choice]) { setState(func() { *value = v }) },
				widgets.NewRow(children, widgets.RowOpts{MainAxisSize: 1}), widgets.RadioGroupOpts[choice]{GroupValue: *value}))
		})})
	}
	first := gluttertest.ByKey(foundation.NewValueKey("first"))
	second := gluttertest.ByKey(foundation.NewValueKey("second"))
	keyed := func(name string, v choice, opts cupertino.CupertinoRadioOpts[choice]) widgets.Widget {
		opts.Key = foundation.NewValueKey(name)
		return radio(v, opts)
	}
	t.Run("a tap selects it", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.None[choice]()
		ts.PumpWidget(group(&sel, keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{}), keyed("second", 2, cupertino.CupertinoRadioOpts[choice]{})))
		ts.Tap(first)
		if sel != foundation.Some(choice(1)) {
			t.Errorf("selected %v", sel)
		}
		ts.Tap(second)
		if sel != foundation.Some(choice(2)) {
			t.Errorf("selected %v", sel)
		}
		ts.PumpAndSettle()
		ts.Tap(second) // not toggleable: the value stays.
		if sel != foundation.Some(choice(2)) {
			t.Errorf("a second tap changed the value to %v", sel)
		}
	})
	t.Run("a toggleable radio deselects", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.Some(choice(1))
		ts.PumpWidget(group(&sel, keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{Toggleable: true}), keyed("second", 2, cupertino.CupertinoRadioOpts[choice]{})))
		ts.Tap(first)
		if sel.IsSome() {
			t.Errorf("selected %v, want none", sel)
		}
	})
	t.Run("the deprecated arguments work without a group", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.None[choice]()
		app := func() widgets.Widget {
			return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewCenter(widgets.NewRow([]widgets.Widget{
				keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{GroupValue: sel, OnChanged: func(v foundation.Opt[choice]) { sel = v }}),
			}, widgets.RowOpts{MainAxisSize: 1}))})
		}
		ts.PumpWidget(app())
		ts.Tap(first)
		if sel != foundation.Some(choice(1)) {
			t.Errorf("selected %v", sel)
		}
	})
	t.Run("a disabled radio ignores the taps", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.None[choice]()
		ts.PumpWidget(group(&sel, keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{Enabled: foundation.Some(false)})))
		ts.Tap(first)
		if sel.IsSome() {
			t.Errorf("selected %v", sel)
		}
	})
	t.Run("it is 18 by 18 inside its tap target", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.None[choice]()
		ts.PumpWidget(group(&sel, keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{})))
		paint := gluttertest.Descendant(first, gluttertest.ByType[*widgets.CustomPaint]())
		if got, want := ts.Size(paint), (ui.Size{Width: 18, Height: 18}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
	})
	t.Run("the keyboard selects it", func(t *testing.T) {
		ts := gluttertest.New(t)
		sel := foundation.None[choice]()
		ts.PumpWidget(group(&sel, keyed("first", 1, cupertino.CupertinoRadioOpts[choice]{Autofocus: true})))
		ts.Pump()
		ts.SendKeyEvent(services.LogicalKeyboardKeySpace)
		ts.Pump()
		if sel != foundation.Some(choice(1)) {
			t.Errorf("selected %v", sel)
		}
	})
	t.Run("the error of an enabled radio that has nowhere to report", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(radio(1, cupertino.CupertinoRadioOpts[choice]{Enabled: foundation.Some(true)})))
		err := firstError(t, ts)
		requireErrorTag(t, err, "cupertinoRadioState[...].Build.CupertinoRadioNoRegistry")
	})
}

func TestCupertinoSlider(t *testing.T) {
	sl := gluttertest.ByType[*cupertino.CupertinoSlider]()
	type sliderValues struct {
		value        float64
		starts, ends []float64
	}
	sliderApp := func(s *sliderValues, opts cupertino.CupertinoSliderOpts, direction ui.TextDirection) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Home: widgets.NewDirectionality(direction, widgets.NewStatefulBuilder(
			func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				opts.OnChangeStart = func(v float64) { s.starts = append(s.starts, v) }
				opts.OnChangeEnd = func(v float64) { s.ends = append(s.ends, v) }
				return widgets.NewCenter(cupertino.NewCupertinoSlider(s.value, func(v float64) { setState(func() { s.value = v }) }, opts))
			}))})
	}
	// thumb is where the thumb of a slider of the value (from 0 to 1) is: the track has 8 of padding and the thumb 14 of
	// radius, so the thumb moves over 132.
	thumb := func(ts *gluttertest.Tester, value float64) ui.Offset {
		c := ts.Center(sl)
		return ui.Offset{Dx: c.Dx - 88 + 22 + value*132, Dy: c.Dy}
	}
	t.Run("it is 176 by 44", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoSlider(0.5, func(float64) {})))
		if got, want := ts.Size(sl), (ui.Size{Width: 176, Height: 44}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
	})
	t.Run("a drag moves the thumb in proportion", func(t *testing.T) {
		ts := gluttertest.New(t)
		s := &sliderValues{value: 0.5}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{}, ui.TextDirectionLTR))
		start := ts.Center(sl)
		gesture := ts.StartGesture(start)
		gesture.MoveBy(ui.Offset{Dx: 40})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		// The extent is 176 - 2*(8+14) = 132.
		if want := 0.5 + 40.0/132.0; math.Abs(s.value-want) > 0.02 {
			t.Errorf("value = %v, want about %v", s.value, want)
		}
		if len(s.starts) != 1 || len(s.ends) != 1 {
			t.Errorf("starts %v, ends %v", s.starts, s.ends)
		}
	})
	t.Run("a drag the other way in RTL", func(t *testing.T) {
		ts := gluttertest.New(t)
		s := &sliderValues{value: 0.5}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{}, ui.TextDirectionRTL))
		gesture := ts.StartGesture(ts.Center(sl))
		gesture.MoveBy(ui.Offset{Dx: 40})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		if s.value >= 0.5 {
			t.Errorf("value = %v, want less than 0.5", s.value)
		}
	})
	t.Run("a drag stops at the ends and is in divisions", func(t *testing.T) {
		ts := gluttertest.New(t)
		s := &sliderValues{value: 0.5}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{Divisions: foundation.Some(4)}, ui.TextDirectionLTR))
		ts.DragFrom(thumb(ts, 0.5), ui.Offset{Dx: 30})
		ts.PumpAndSettle()
		if s.value != 0.75 {
			t.Errorf("value = %v, want 0.75", s.value)
		}
		ts.DragFrom(thumb(ts, 0.75), ui.Offset{Dx: 400})
		ts.PumpAndSettle()
		if s.value != 1 {
			t.Errorf("value = %v, want 1", s.value)
		}
	})
	t.Run("a minimum and a maximum", func(t *testing.T) {
		ts := gluttertest.New(t)
		s := &sliderValues{value: 50}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{Min: 0, Max: foundation.Some(100.0)}, ui.TextDirectionLTR))
		ts.Drag(sl, ui.Offset{Dx: 600})
		ts.PumpAndSettle()
		if s.value != 100 {
			t.Errorf("value = %v, want 100", s.value)
		}
	})
	t.Run("a disabled slider ignores the drag", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoSlider(0.5, nil)))
		ts.Drag(sl, ui.Offset{Dx: 40})
		ts.PumpAndSettle()
		if ts.Widget(sl).(*cupertino.CupertinoSlider).Value() != 0.5 {
			t.Error("a disabled slider changed")
		}
	})
	t.Run("the thumb takes the drag only near it", func(t *testing.T) {
		ts := gluttertest.New(t)
		s := &sliderValues{value: 0.0}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{}, ui.TextDirectionLTR))
		far := ts.Center(sl).Add(ui.Offset{Dx: 60})
		gesture := ts.StartGesture(far)
		gesture.MoveBy(ui.Offset{Dx: 20})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		if s.value != 0 {
			t.Errorf("a drag far from the thumb moved it to %v", s.value)
		}
	})
	t.Run("a haptic feedback on iOS at an end", func(t *testing.T) {
		skipUnlessDebug(t) // the platform override is a debug-only feature
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformIOS))
		defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		ts := gluttertest.New(t)
		var kinds []ui.HapticFeedbackKind
		ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) { kinds = append(kinds, k) }
		s := &sliderValues{value: 0.9}
		ts.PumpWidget(sliderApp(s, cupertino.CupertinoSliderOpts{}, ui.TextDirectionLTR))
		ts.DragFrom(thumb(ts, 0.9), ui.Offset{Dx: 300})
		ts.PumpAndSettle()
		if len(kinds) == 0 {
			t.Error("no haptic feedback at the end of the track")
		}
	})
	t.Run("the errors of the arguments", func(t *testing.T) {
		skipUnlessDebug(t)
		requireErrorTag(t, recoverError(func() { cupertino.NewCupertinoSlider(2, nil) }), "NewCupertinoSlider.CupertinoSliderValueOutOfRange")
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSlider(0.5, nil, cupertino.CupertinoSliderOpts{Divisions: foundation.Some(0)})
		}), "NewCupertinoSlider.CupertinoSliderDivisionsInvalid")
	})
}

func TestCupertinoActivityIndicator(t *testing.T) {
	t.Run("it is 20 by 20 and spins forever", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoActivityIndicator()))
		ind := gluttertest.ByType[*cupertino.CupertinoActivityIndicator]()
		if got, want := ts.Size(ind), (ui.Size{Width: 20, Height: 20}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
		ts.Pump(300 * time.Millisecond)
		if ts.Binding.Scheduler().TransientCallbackCount() == 0 {
			t.Error("a spinning indicator has no ticker running")
		}
	})
	t.Run("the radius sets the size", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoActivityIndicator(cupertino.CupertinoActivityIndicatorOpts{Radius: foundation.Some(30.0)})))
		if got, want := ts.Size(gluttertest.ByType[*cupertino.CupertinoActivityIndicator]()), (ui.Size{Width: 60, Height: 60}); got != want {
			t.Errorf("size = %v, want %v", got, want)
		}
	})
	t.Run("a partially revealed one does not animate", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(cupertino.NewCupertinoActivityIndicatorPartiallyRevealed()))
		ts.Pump(300 * time.Millisecond)
		if n := ts.Binding.Scheduler().TransientCallbackCount(); n != 0 {
			t.Errorf("a partially revealed indicator has %d tickers running", n)
		}
	})
	t.Run("a linear one is as wide as the space", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(centered(widgets.NewSizedBox(cupertino.NewCupertinoLinearActivityIndicator(0.5), widgets.SizedBoxOpts{Width: foundation.Some(200.0)})))
		got := ts.Size(gluttertest.ByType[*cupertino.CupertinoLinearActivityIndicator]())
		if got.Width != 200 {
			t.Errorf("width = %v, want 200", got.Width)
		}
	})
	t.Run("the errors of the arguments", func(t *testing.T) {
		skipUnlessDebug(t)
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoActivityIndicator(cupertino.CupertinoActivityIndicatorOpts{Radius: foundation.Some(0.0)})
		}), "NewCupertinoActivityIndicator.CupertinoActivityIndicatorRadiusInvalid")
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoActivityIndicatorPartiallyRevealed(cupertino.CupertinoActivityIndicatorPartiallyRevealedOpts{Progress: foundation.Some(2.0)})
		}), "NewCupertinoActivityIndicatorPartiallyRevealed.CupertinoActivityIndicatorProgressInvalid")
		requireErrorTag(t, recoverError(func() { cupertino.NewCupertinoLinearActivityIndicator(2) }), "NewCupertinoLinearActivityIndicator.CupertinoActivityIndicatorProgressInvalid")
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoLinearActivityIndicator(0.5, cupertino.CupertinoLinearActivityIndicatorOpts{Height: foundation.Some(0.0)})
		}), "NewCupertinoLinearActivityIndicator.CupertinoActivityIndicatorHeightInvalid")
	})
}

var _ = painting.EdgeInsetsZero
