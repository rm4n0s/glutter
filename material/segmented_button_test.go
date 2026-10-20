// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func textSegments(values ...string) []ButtonSegment[string] {
	out := make([]ButtonSegment[string], len(values))
	for i, v := range values {
		out[i] = ButtonSegment[string]{Value: v, Label: widgets.NewText(v)}
	}
	return out
}

func TestSegmentedButtonSelection(t *testing.T) {
	tests := []struct {
		name     string
		opts     SegmentedButtonOpts
		selected []string
		press    string
		want     []string
		changed  bool
	}{
		{"single selection picks the pressed one", SegmentedButtonOpts{}, []string{"A"}, "B", []string{"B"}, true},
		{"single selection keeps the only one", SegmentedButtonOpts{}, []string{"A"}, "A", nil, false},
		{"empty selection allowed toggles it off", SegmentedButtonOpts{EmptySelectionAllowed: true}, []string{"A"}, "A", []string{}, true},
		{"empty selection allowed picks one", SegmentedButtonOpts{EmptySelectionAllowed: true}, nil, "B", []string{"B"}, true},
		{"empty selection allowed replaces", SegmentedButtonOpts{EmptySelectionAllowed: true}, []string{"A"}, "B", []string{"B"}, true},
		{"multi selection adds", SegmentedButtonOpts{MultiSelectionEnabled: true}, []string{"A"}, "B", []string{"A", "B"}, true},
		{"multi selection removes", SegmentedButtonOpts{MultiSelectionEnabled: true}, []string{"A", "B"}, "A", []string{"B"}, true},
		{"multi selection keeps the last one", SegmentedButtonOpts{MultiSelectionEnabled: true}, []string{"A"}, "A", nil, false},
		{"multi selection empty allowed removes the last", SegmentedButtonOpts{MultiSelectionEnabled: true, EmptySelectionAllowed: true}, []string{"A"}, "A", []string{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			changed := false
			ts := pumpButton(t, standardTheme(true), NewSegmentedButton(textSegments("A", "B", "C"), tc.selected,
				func(v []string) { got, changed = v, true }, tc.opts))
			ts.Tap(gluttertest.ByText(tc.press))
			ts.Pump()
			if changed != tc.changed {
				t.Fatalf("changed = %v, want %v", changed, tc.changed)
			}
			if tc.changed && !slices.Equal(got, tc.want) {
				t.Errorf("selection = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSegmentedButtonDisabled(t *testing.T) {
	t.Run("without a callback", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewSegmentedButton(textSegments("A", "B"), []string{"A"}, nil))
		tb := gluttertest.WidgetAs[*TextButton](ts, gluttertest.First(gluttertest.ByType[*TextButton]()))
		if tb.Enabled() {
			t.Error("the segments of a button without a callback are enabled")
		}
	})
	t.Run("a disabled segment", func(t *testing.T) {
		pressed := ""
		segments := textSegments("A", "B")
		segments[1].Disabled = true
		ts := pumpButton(t, standardTheme(true), NewSegmentedButton(segments, []string{"A"}, func(v []string) { pressed = v[0] }))
		ts.Tap(gluttertest.ByText("B"))
		ts.Pump()
		if pressed != "" {
			t.Errorf("the disabled segment selected %q", pressed)
		}
		buttons := gluttertest.ByType[*TextButton]()
		if gluttertest.WidgetAs[*TextButton](ts, gluttertest.At(buttons, 0)).Enabled() == false || gluttertest.WidgetAs[*TextButton](ts, gluttertest.At(buttons, 1)).Enabled() {
			t.Error("the segments do not have the right enabled state")
		}
	})
}

func TestSegmentedButtonLayoutAndColors(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	ts := pumpButton(t, theme, NewSegmentedButton(textSegments("One", "Two", "Three"), []string{"Two"}, func([]string) {}))
	button := gluttertest.ByType[*SegmentedButton[string]]()
	size := ts.Size(button)
	if size.Height != 40 {
		t.Errorf("height = %v, want 40", size.Height)
	}
	buttons := gluttertest.ByType[*TextButton]()
	var widths []float64
	for i := 0; i < ts.Count(buttons); i++ {
		w := ts.Size(gluttertest.At(buttons, i)).Width
		widths = append(widths, w)
		if w != widths[0] {
			t.Errorf("segment %d is %v wide, the first is %v: they have the same size", i, w, widths[0])
		}
	}
	if len(widths) != 3 || size.Width != widths[0]*3 {
		t.Errorf("widths = %v for a button %v wide, want three of a third", widths, size.Width)
	}
	// The selected segment shows the check icon and has the secondary container
	// background.
	if n := ts.Count(gluttertest.ByType[*widgets.Icon]()); n != 1 {
		t.Errorf("%d icons, want the check of the selected segment", n)
	}
	mat := gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Descendant(gluttertest.At(buttons, 1), gluttertest.ByType[*Material]())))
	if mat.Color() != foundation.Some(cs.SecondaryContainer()) {
		t.Errorf("selected background = %v, want the secondary container", mat.Color())
	}
	mat = gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Descendant(gluttertest.At(buttons, 0), gluttertest.ByType[*Material]())))
	if got := mat.TextStyle().Color; got != foundation.Some(cs.OnSurface()) {
		t.Errorf("unselected text color = %v, want on surface", got)
	}
}

func TestSegmentedButtonSelectedIcon(t *testing.T) {
	tests := []struct {
		name      string
		opts      SegmentedButtonOpts
		segments  []ButtonSegment[string]
		wantIcons int
	}{
		{"the check of the selected segment", SegmentedButtonOpts{}, textSegments("A", "B"), 1},
		{"no selected icon", SegmentedButtonOpts{NoShowSelectedIcon: true}, textSegments("A", "B"), 0},
		{"icons of the segments with labels", SegmentedButtonOpts{NoShowSelectedIcon: true},
			[]ButtonSegment[string]{{Value: "A", Label: widgets.NewText("A"), Icon: NewIcon(Icons.Add())}, {Value: "B", Label: widgets.NewText("B"), Icon: NewIcon(Icons.Add())}}, 2},
		{"selected icon replaces the icon", SegmentedButtonOpts{},
			[]ButtonSegment[string]{{Value: "A", Label: widgets.NewText("A"), Icon: NewIcon(Icons.Add())}, {Value: "B", Label: widgets.NewText("B"), Icon: NewIcon(Icons.Add())}}, 2},
		{"icon only segments show their icon", SegmentedButtonOpts{NoShowSelectedIcon: true},
			[]ButtonSegment[string]{{Value: "A", Icon: NewIcon(Icons.Add())}, {Value: "B", Icon: NewIcon(Icons.Send())}}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), NewSegmentedButton(tc.segments, []string{"A"}, func([]string) {}, tc.opts))
			if n := ts.Count(gluttertest.ByType[*widgets.Icon]()); n != tc.wantIcons {
				t.Errorf("%d icons, want %d", n, tc.wantIcons)
			}
		})
	}
	t.Run("custom selected icon", func(t *testing.T) {
		custom := NewIcon(Icons.Favorite())
		ts := pumpButton(t, standardTheme(true), NewSegmentedButton(textSegments("A", "B"), []string{"A"}, func([]string) {}, SegmentedButtonOpts{SelectedIcon: custom}))
		if !ts.Exists(gluttertest.ByWidget(custom)) {
			t.Error("the custom selected icon is not shown")
		}
	})
}

func TestSegmentedButtonDirectionAndExpansion(t *testing.T) {
	t.Run("vertical", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewSegmentedButton(textSegments("One", "Two"), []string{"One"}, func([]string) {}, SegmentedButtonOpts{Direction: painting.AxisVertical}))
		buttons := gluttertest.ByType[*TextButton]()
		first, second := ts.TopLeft(gluttertest.At(buttons, 0)), ts.TopLeft(gluttertest.At(buttons, 1))
		if first.Dx != second.Dx || second.Dy <= first.Dy {
			t.Errorf("positions %v and %v: the second goes below the first", first, second)
		}
	})
	t.Run("expanded", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewSegmentedButton(textSegments("One", "Two"), []string{"One"}, func([]string) {}, SegmentedButtonOpts{ExpandedInsets: foundation.Some(painting.NewEdgeInsetsAll(10))}))
		if got := ts.Size(gluttertest.ByType[*SegmentedButton[string]]()).Width; got < 380 {
			t.Errorf("width = %v, want the width of the window less the insets", got)
		}
	})
	t.Run("right to left", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: widgets.NewDirectionality(ui.TextDirectionRTL, NewScaffold(ScaffoldOpts{
			Body: widgets.NewCenter(NewSegmentedButton(textSegments("One", "Two"), []string{"One"}, func([]string) {}))}))}))
		buttons := gluttertest.ByType[*TextButton]()
		if ts.TopLeft(gluttertest.At(buttons, 0)).Dx <= ts.TopLeft(gluttertest.At(buttons, 1)).Dx {
			t.Error("the first segment is not at the right")
		}
	})
}

func TestSegmentedButtonStatesControllers(t *testing.T) {
	segments := textSegments("A", "B")
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	build := func(selected string, segs []ButtonSegment[string]) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewSegmentedButton(segs, []string{selected}, func([]string) {}))})})
	}
	ts.PumpWidget(build("A", segments))
	state := gluttertest.StateOf[*SegmentedButtonState[string]](ts, gluttertest.ByType[*SegmentedButton[string]]())
	if n := len(state.StatesControllers()); n != 2 {
		t.Fatalf("%d states controllers, want 2", n)
	}
	if !state.StatesControllers()["A"].Value().Has(widgets.WidgetStateSelected) || state.StatesControllers()["B"].Value().Has(widgets.WidgetStateSelected) {
		t.Error("the selected state is not on the selected segment only")
	}
	ts.PumpWidget(build("B", segments))
	if state.StatesControllers()["A"].Value().Has(widgets.WidgetStateSelected) || !state.StatesControllers()["B"].Value().Has(widgets.WidgetStateSelected) {
		t.Error("the selected state did not move")
	}
	// A segment that goes leaves its controller.
	ts.PumpWidget(build("B", segments[1:]))
	if n := len(state.StatesControllers()); n != 1 {
		t.Errorf("%d states controllers after a segment went, want 1", n)
	}
}

func TestSegmentedButtonStyleFrom(t *testing.T) {
	blue, red := ui.NewColor(0xFF0000FF), ui.NewColor(0xFFFF0000)
	s := SegmentedButtonStyleFrom(SegmentedButtonStyleFromOpts{
		ButtonStyleFromOpts:     ButtonStyleFromOpts{ForegroundColor: foundation.Some(blue), BackgroundColor: foundation.Some(red), DisabledForegroundColor: foundation.Some(Colors.Transparent)},
		SelectedForegroundColor: foundation.Some(red), SelectedBackgroundColor: foundation.Some(blue)})
	states := widgets.NewWidgetStates
	tests := []struct {
		name   string
		prop   widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
		states widgets.WidgetStates
		want   ui.Color
	}{
		{"foreground", s.ForegroundColor, states(), blue},
		{"selected foreground", s.ForegroundColor, states(widgets.WidgetStateSelected), red},
		{"disabled foreground", s.ForegroundColor, states(widgets.WidgetStateDisabled), Colors.Transparent},
		{"background", s.BackgroundColor, states(), red},
		{"selected background", s.BackgroundColor, states(widgets.WidgetStateSelected), blue},
		{"hover overlay", s.OverlayColor, states(widgets.WidgetStateHovered), blue.WithOpacity(0.08)},
		{"selected press overlay", s.OverlayColor, states(widgets.WidgetStateSelected, widgets.WidgetStatePressed), red.WithOpacity(0.1)},
		{"overlay at rest", s.OverlayColor, states(), Colors.Transparent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.prop.Resolve(tc.states); got != foundation.Some(tc.want) {
				t.Errorf("color = %v, want %v", got, tc.want)
			}
		})
	}
	if SegmentedButtonStyleFrom(SegmentedButtonStyleFromOpts{}).OverlayColor != nil {
		t.Error("a style without colors has an overlay")
	}
}

func TestSegmentedButtonErrors(t *testing.T) {
	skipUnlessDebug(t)
	one := textSegments("A")
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"no segments", func() { NewSegmentedButton[string](nil, nil, nil, SegmentedButtonOpts{EmptySelectionAllowed: true}) }, "SegmentedButtonNoSegments"},
		{"no selection", func() { NewSegmentedButton(one, nil, nil) }, "SegmentedButtonNoSelection"},
		{"several selected", func() { NewSegmentedButton(textSegments("A", "B"), []string{"A", "B"}, nil) }, "SegmentedButtonMultipleSelected"},
		{"empty segment", func() { NewSegmentedButton([]ButtonSegment[string]{{Value: "A"}}, []string{"A"}, nil) }, "ButtonSegmentEmpty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.route)
		})
	}
}
