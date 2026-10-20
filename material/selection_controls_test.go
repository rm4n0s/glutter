// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func someBool(b bool) foundation.Opt[bool] { return foundation.Some(b) }

func TestCheckboxTap(t *testing.T) {
	none := foundation.None[bool]()
	tests := []struct {
		name     string
		value    foundation.Opt[bool]
		tristate bool
		want     foundation.Opt[bool]
	}{
		{"false becomes true", someBool(false), false, someBool(true)},
		{"true becomes false", someBool(true), false, someBool(false)},
		{"true becomes none when tristate", someBool(true), true, none},
		{"none becomes false when tristate", none, true, someBool(false)},
		{"false becomes true when tristate", someBool(false), true, someBool(true)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got foundation.Opt[bool]
			calls := 0
			ts := pumpButton(t, standardTheme(true), NewCheckbox(tc.value, func(v foundation.Opt[bool]) { got = v; calls++ }, CheckboxOpts{Tristate: tc.tristate}))
			ts.Tap(gluttertest.ByType[*Checkbox]())
			ts.Pump()
			if calls != 1 || got != tc.want {
				t.Errorf("onChanged called %d times with %v, want once with %v", calls, got, tc.want)
			}
		})
	}
}

func TestCheckboxDisabledAndSize(t *testing.T) {
	tests := []struct {
		name     string
		tapSize  MaterialTapTargetSize
		density  VisualDensity
		wantSize ui.Size
	}{
		{"padded", MaterialTapTargetSizePadded, VisualDensityStandard(), ui.Size{Width: 48, Height: 48}},
		{"shrink wrap", MaterialTapTargetSizeShrinkWrap, VisualDensityStandard(), ui.Size{Width: 40, Height: 40}},
		{"compact", MaterialTapTargetSizePadded, VisualDensityCompact(), ui.Size{Width: 40, Height: 40}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), NewCheckbox(someBool(true), func(foundation.Opt[bool]) {},
				CheckboxOpts{MaterialTapTargetSize: tc.tapSize, VisualDensity: foundation.Some(tc.density)}))
			if got := ts.Size(gluttertest.ByType[*Checkbox]()); got != tc.wantSize {
				t.Errorf("size = %v, want %v", got, tc.wantSize)
			}
		})
	}
	t.Run("a checkbox without a callback ignores taps", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewCheckbox(someBool(false), nil))
		ts.Tap(gluttertest.ByType[*Checkbox]())
		ts.Pump()
		if n := ts.Count(gluttertest.ByType[*Checkbox]()); n != 1 {
			t.Fatalf("%d checkboxes", n)
		}
	})
}

func TestCheckboxColors(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name    string
		value   bool
		enabled bool
		want    ui.Color
	}{
		{"selected is primary", true, true, cs.Primary()},
		{"selected and disabled is on surface at 38%", true, false, cs.OnSurface().WithOpacity(0.38)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var onChanged func(foundation.Opt[bool])
			if tc.enabled {
				onChanged = func(foundation.Opt[bool]) {}
			}
			ts := pumpButton(t, theme, NewCheckbox(someBool(tc.value), onChanged))
			ts.PumpAndSettle()
			// The corner of the box of 18 by 18 is rounded: sample just inside it.
			c := ts.Center(gluttertest.ByType[*Checkbox]())
			got := pixel(ts, int(c.Dx-7), int(c.Dy-7))
			want := tc.want
			blended := ui.ColorAlphaBlend(want, cs.Surface())
			if int(got.R) != blended.Red() || int(got.G) != blended.Green() || int(got.B) != blended.Blue() {
				t.Errorf("fill = %v, want %v", got, blended)
			}
		})
	}
}

func TestCheckboxAnimatesToSelected(t *testing.T) {
	theme := standardTheme(true)
	value := someBool(false)
	var rebuild func()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			rebuild = func() { setState(func() {}) }
			return NewCheckbox(value, func(v foundation.Opt[bool]) { value = v; rebuild() })
		}))})}))
	c := ts.Center(gluttertest.ByType[*Checkbox]())
	before := pixel(ts, int(c.Dx-7), int(c.Dy-7))
	ts.Tap(gluttertest.ByType[*Checkbox]())
	ts.Pump()
	ts.Pump(ts.Elapsed() * 0) // no time passes
	ts.PumpAndSettle()
	after := pixel(ts, int(c.Dx-7), int(c.Dy-7))
	if before == after {
		t.Errorf("the checkbox did not change its fill: %v", after)
	}
	if value != someBool(true) {
		t.Errorf("value = %v", value)
	}
}

func TestRadioGroupSelection(t *testing.T) {
	var got foundation.Opt[string]
	group := foundation.Some("a")
	build := func(toggleable bool) widgets.Widget {
		return widgets.NewRadioGroup(func(v foundation.Opt[string]) { got = v },
			widgets.NewColumn([]widgets.Widget{
				NewRadio("a", RadioOpts[string]{Toggleable: toggleable}),
				NewRadio("b", RadioOpts[string]{Toggleable: toggleable}),
			}), widgets.RadioGroupOpts[string]{GroupValue: group})
	}
	tests := []struct {
		name       string
		toggleable bool
		tapIndex   int
		want       foundation.Opt[string]
		wantCalled bool
	}{
		{"select another", false, 1, foundation.Some("b"), true},
		{"select the selected", false, 0, foundation.None[string](), false},
		{"toggle the selected", true, 0, foundation.None[string](), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got = foundation.Some("unset")
			ts := pumpButton(t, standardTheme(true), build(tc.toggleable))
			ts.Tap(gluttertest.At(gluttertest.ByType[*Radio[string]](), tc.tapIndex))
			ts.Pump()
			called := got != foundation.Some("unset")
			if called != tc.wantCalled || (called && got != tc.want) {
				t.Errorf("onChanged = %v (called %v), want %v (called %v)", got, called, tc.want, tc.wantCalled)
			}
		})
	}
}

func TestRadioWithoutGroup(t *testing.T) {
	var got foundation.Opt[int]
	ts := pumpButton(t, standardTheme(true), NewRadio(3, RadioOpts[int]{GroupValue: foundation.Some(1), OnChanged: func(v foundation.Opt[int]) { got = v }}))
	ts.Tap(gluttertest.ByType[*Radio[int]]())
	ts.Pump()
	if got != foundation.Some(3) {
		t.Errorf("onChanged got %v, want 3", got)
	}
	// The theme of Linux shrinks the tap targets.
	if size := ts.Size(gluttertest.ByType[*Radio[int]]()); size != (ui.Size{Width: 40, Height: 40}) {
		t.Errorf("size = %v", size)
	}
}

func TestRadioGroupKeyboard(t *testing.T) {
	var got foundation.Opt[string]
	ts := pumpButton(t, standardTheme(true), widgets.NewRadioGroup(func(v foundation.Opt[string]) { got = v },
		widgets.NewColumn([]widgets.Widget{NewRadio("a", RadioOpts[string]{Autofocus: true}), NewRadio("b", RadioOpts[string]{})}),
		widgets.RadioGroupOpts[string]{GroupValue: foundation.Some("a")}))
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	if got != foundation.Some("b") {
		t.Errorf("the arrow down selected %v, want b", got)
	}
}

func TestSwitchTapAndDrag(t *testing.T) {
	t.Run("tap", func(t *testing.T) {
		var got []bool
		ts := pumpButton(t, standardTheme(true), NewSwitch(false, func(v bool) { got = append(got, v) }))
		ts.Tap(gluttertest.ByType[*Switch]())
		ts.Pump()
		if len(got) != 1 || !got[0] {
			t.Errorf("onChanged = %v, want [true]", got)
		}
	})
	t.Run("drag to the end", func(t *testing.T) {
		var got []bool
		ts := pumpButton(t, standardTheme(true), NewSwitch(false, func(v bool) { got = append(got, v) }))
		ts.Drag(gluttertest.ByType[*Switch](), ui.Offset{Dx: 40, Dy: 0})
		ts.Pump()
		if len(got) == 0 || !got[len(got)-1] {
			t.Errorf("onChanged = %v, want it to end with true", got)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewSwitch(false, nil))
		ts.Tap(gluttertest.ByType[*Switch]())
		ts.Pump()
	})
	t.Run("size", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewSwitch(true, func(bool) {}))
		if got := ts.Size(gluttertest.ByType[*Switch]()); got != (ui.Size{Width: 60, Height: 40}) {
			t.Errorf("size = %v, want 60 by 40", got)
		}
	})
}

func TestSwitchColors(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name  string
		value bool
		want  ui.Color
	}{
		{"on track is primary", true, cs.Primary()},
		{"off track is surface container highest", false, cs.SurfaceContainerHighest()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, theme, NewSwitch(tc.value, func(bool) {}))
			ts.PumpAndSettle()
			c := ts.Center(gluttertest.ByType[*Switch]())
			// Sample the track next to the thumb, on the side that the thumb is not.
			x := c.Dx + 18
			if tc.value {
				x = c.Dx - 18
			}
			got := pixel(ts, int(x), int(c.Dy))
			if int(got.R) != tc.want.Red() || int(got.G) != tc.want.Green() || int(got.B) != tc.want.Blue() {
				t.Errorf("track = %v, want %v", got, tc.want)
			}
		})
	}
}
