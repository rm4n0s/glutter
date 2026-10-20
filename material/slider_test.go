// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// sliderTrackX is the x of the fraction of the track of the slider that the finder
// finds: the track is inset by the radius of the overlay (24) on both sides.
func sliderTrackX(ts *gluttertest.Tester, f gluttertest.Finder, fraction float64) float64 {
	left, width := ts.TopLeft(f).Dx, ts.Size(f).Width
	return left + 24 + fraction*(width-48)
}

func within(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestSliderTapAndDrag(t *testing.T) {
	slider := gluttertest.ByType[*Slider]()
	tests := []struct {
		name string
		opts SliderOpts
		// start is the value of the slider; at is the fraction of the track that
		// is tapped; want is the value that onChanged gets.
		start, at, want float64
	}{
		{"tap in the middle", SliderOpts{}, 0.2, 0.5, 0.5},
		{"tap near the start", SliderOpts{}, 0.5, 0.1, 0.1},
		{"tap the end", SliderOpts{}, 0.5, 1.0, 1.0},
		{"tap a range", SliderOpts{Min: 10, Max: foundation.Some(20.0)}, 12, 0.5, 15},
		{"tap rounds to a division", SliderOpts{Divisions: foundation.Some(4)}, 0.0, 0.6, 0.5},
		{"tap rounds up to a division", SliderOpts{Divisions: foundation.Some(4)}, 0.0, 0.7, 0.75},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []float64
			ts := pumpButton(t, standardTheme(true), NewSlider(tc.start, func(v float64) { got = append(got, v) }, tc.opts))
			ts.TapAt(ui.Offset{Dx: sliderTrackX(ts, slider, tc.at), Dy: ts.Center(slider).Dy})
			ts.PumpAndSettle()
			if len(got) == 0 || !within(got[len(got)-1], tc.want, 0.01*math.Max(1, math.Abs(tc.want))) {
				t.Errorf("onChanged = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSliderDragCallbacks(t *testing.T) {
	var events []string
	var last float64
	value := 0.5
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	var setState widgets.StateSetter
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		widgets.NewStatefulBuilder(func(_ widgets.BuildContext, s widgets.StateSetter) widgets.Widget {
			setState = s
			return NewSlider(value, func(v float64) {
				events = append(events, "changed")
				last = v
				setState(func() { value = v })
			}, SliderOpts{
				OnChangeStart: func(float64) { events = append(events, "start") },
				OnChangeEnd:   func(v float64) { events = append(events, "end") },
			})
		}))})}))
	slider := gluttertest.ByType[*Slider]()
	c := ts.Center(slider)
	ts.DragFrom(ui.Offset{Dx: c.Dx, Dy: c.Dy}, ui.Offset{Dx: 80, Dy: 0})
	ts.PumpAndSettle()
	if len(events) < 3 || events[0] != "start" || events[len(events)-1] != "end" {
		t.Fatalf("events = %v, want start, changes, end", events)
	}
	want := 0.5 + 80.0/(ts.Size(slider).Width-48)
	if !within(last, want, 0.03) {
		t.Errorf("last value = %v, want about %v", last, want)
	}
}

func TestSliderDisabled(t *testing.T) {
	ts := pumpButton(t, standardTheme(true), NewSlider(0.5, nil))
	slider := gluttertest.ByType[*Slider]()
	ts.Tap(slider)
	ts.PumpAndSettle()
	if got := gluttertest.WidgetAs[*Slider](ts, slider).Value(); got != 0.5 {
		t.Errorf("value = %v", got)
	}
}

func TestSliderSizeAndColors(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	ts := pumpButton(t, theme, widgets.NewSizedBox(NewSlider(0.5, func(float64) {}), widgets.SizedBoxOpts{Height: foundation.Some(48.0)}))
	slider := gluttertest.ByType[*Slider]()
	size := ts.Size(slider)
	if size.Width != 400 || size.Height != 48 {
		t.Errorf("size = %v, want 400 by 48", size)
	}
	// The active part of the track is primary, the inactive one is the surface
	// container highest (the 2023 look).
	c := ts.Center(slider)
	for _, tc := range []struct {
		name string
		x    float64
		want ui.Color
	}{
		{"active", c.Dx - 60, cs.Primary()},
		{"inactive", c.Dx + 60, cs.SurfaceContainerHighest()},
	} {
		got := pixel(ts, int(tc.x), int(c.Dy))
		if int(got.R) != tc.want.Red() || int(got.G) != tc.want.Green() || int(got.B) != tc.want.Blue() {
			t.Errorf("%s track = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestSliderValueIndicator(t *testing.T) {
	// Dragging a slider with a label shows the value indicator in the overlay.
	value := 0.5
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return NewSlider(value, func(v float64) { setState(func() { value = v }) },
				SliderOpts{Divisions: foundation.Some(10), Label: foundation.Some("half")})
		}))})}))
	slider := gluttertest.ByType[*Slider]()
	c := ts.Center(slider)
	before := ts.Screenshot()
	g := ts.StartGesture(c)
	g.MoveBy(ui.Offset{Dx: 4, Dy: 0})
	ts.Pump()
	ts.Pump(300 * 1e6)
	during := ts.Screenshot()
	same := true
	for i := range before.Pix {
		if before.Pix[i] != during.Pix[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("the screen did not change while the slider was pressed")
	}
	g.Up()
	ts.PumpAndSettle()
}

func TestSliderErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		build     func()
		wantRoute string
	}{
		{"min greater than max", func() { NewSlider(0, nil, SliderOpts{Min: 2, Max: foundation.Some(1.0)}) }, "SliderMinGreaterThanMax"},
		{"value below the range", func() { NewSlider(-1, nil) }, "SliderValueOutOfRange"},
		{"value above the range", func() { NewSlider(2, nil) }, "SliderValueOutOfRange"},
		{"secondary value out of range", func() { NewSlider(0.5, nil, SliderOpts{SecondaryTrackValue: foundation.Some(3.0)}) }, "SliderSecondaryValueOutOfRange"},
		{"divisions", func() { NewSlider(0.5, nil, SliderOpts{Divisions: foundation.Some(0)}) }, "SliderDivisionsInvalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.wantRoute)
		})
	}
}

func TestRangeSliderDrag(t *testing.T) {
	var changes []RangeValues
	values := RangeValues{Start: 0.2, End: 0.8}
	var starts, ends []RangeValues
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return NewRangeSlider(values, func(v RangeValues) { changes = append(changes, v); setState(func() { values = v }) },
				RangeSliderOpts{
					OnChangeStart: func(v RangeValues) { starts = append(starts, v) },
					OnChangeEnd:   func(v RangeValues) { ends = append(ends, v) },
				})
		}))})}))
	slider := gluttertest.ByType[*RangeSlider]()
	y := ts.Center(slider).Dy
	// Drag the end thumb to the left.
	ts.DragFrom(ui.Offset{Dx: sliderTrackX(ts, slider, 0.8), Dy: y}, ui.Offset{Dx: -100, Dy: 0})
	ts.PumpAndSettle()
	if values.Start != 0.2 || !(values.End < 0.8-0.2) {
		t.Errorf("values = %v, want the end moved to the left and the start kept", values)
	}
	if len(starts) != 1 || len(ends) != 1 {
		t.Errorf("onChangeStart %d times and onChangeEnd %d times, want once each", len(starts), len(ends))
	}
	// Drag the start thumb to the left edge.
	ts.DragFrom(ui.Offset{Dx: sliderTrackX(ts, slider, 0.2), Dy: y}, ui.Offset{Dx: -200, Dy: 0})
	ts.PumpAndSettle()
	if values.Start != 0 {
		t.Errorf("start = %v, want 0", values.Start)
	}
}

func TestRangeSliderTap(t *testing.T) {
	tests := []struct {
		name      string
		at        float64
		wantStart bool
	}{
		{"tap near the start moves the start", 0.1, true},
		{"tap near the end moves the end", 0.9, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			values := RangeValues{Start: 0.3, End: 0.7}
			var got RangeValues
			ts := pumpButton(t, standardTheme(true), NewRangeSlider(values, func(v RangeValues) { got = v }))
			slider := gluttertest.ByType[*RangeSlider]()
			ts.TapAt(ui.Offset{Dx: sliderTrackX(ts, slider, tc.at), Dy: ts.Center(slider).Dy})
			ts.PumpAndSettle()
			if tc.wantStart && !(within(got.Start, tc.at, 0.02) && got.End == 0.7) {
				t.Errorf("values = %v, want the start at %v", got, tc.at)
			}
			if !tc.wantStart && !(within(got.End, tc.at, 0.02) && got.Start == 0.3) {
				t.Errorf("values = %v, want the end at %v", got, tc.at)
			}
		})
	}
}

func TestRangeSliderErrors(t *testing.T) {
	skipUnlessDebug(t)
	ok := RangeValues{Start: 0.2, End: 0.8}
	tests := []struct {
		name      string
		build     func()
		wantRoute string
	}{
		{"min greater than max", func() { NewRangeSlider(ok, nil, RangeSliderOpts{Min: 2, Max: foundation.Some(1.0)}) }, "RangeSliderMinGreaterThanMax"},
		{"start after end", func() { NewRangeSlider(RangeValues{Start: 0.8, End: 0.2}, nil) }, "RangeSliderValuesInvalid"},
		{"end out of range", func() { NewRangeSlider(RangeValues{Start: 0.2, End: 2}, nil) }, "RangeSliderValuesInvalid"},
		{"divisions", func() { NewRangeSlider(ok, nil, RangeSliderOpts{Divisions: foundation.Some(-1)}) }, "RangeSliderDivisionsInvalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.wantRoute)
		})
	}
}
