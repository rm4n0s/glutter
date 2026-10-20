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
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func chipLabel(s string) widgets.Widget { return widgets.NewText(s) }

// chipBox is the Material of the chip: the visible rectangle of the chip.
func chipBox(ts *gluttertest.Tester, chip gluttertest.Finder) gluttertest.Finder {
	return gluttertest.First(gluttertest.Descendant(chip, gluttertest.ByType[*Material]()))
}

func TestChipSize(t *testing.T) {
	tests := []struct {
		name      string
		tapTarget MaterialTapTargetSize
		// wantTarget is the height of the tap target; zero for the one of the box.
		wantTarget float64
	}{
		{"padded", MaterialTapTargetSizePadded, 48},
		{"shrink wrap", MaterialTapTargetSizeShrinkWrap, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), NewChip(chipLabel("Chip"), ChipOpts{MaterialTapTargetSize: tc.tapTarget}))
			chip := gluttertest.ByType[*Chip]()
			// The content is the height of the label, inside the padding of 8 on
			// each side and the border of 1: the Ink pads its child by the
			// dimensions of the shape.
			wantBox := ts.Size(gluttertest.ByText("Chip")).Height + 2*8 + 2*1
			if got := ts.Size(chipBox(ts, chip)).Height; got != wantBox {
				t.Errorf("chip height = %v, want %v", got, wantBox)
			}
			wantTarget := tc.wantTarget
			if wantTarget == 0 {
				wantTarget = wantBox
			}
			if got := ts.Size(chip).Height; got != wantTarget {
				t.Errorf("tap target height = %v, want %v", got, wantTarget)
			}
		})
	}
}

func TestChipWidthFollowsContent(t *testing.T) {
	// 8 of padding, 8 of label padding and 1 of border on each side of the label.
	ts := pumpButton(t, standardTheme(true), NewChip(chipLabel("Chip")))
	label := ts.Size(gluttertest.ByText("Chip")).Width
	box := ts.Size(chipBox(ts, gluttertest.ByType[*Chip]())).Width
	if want := label + 2*8 + 2*8 + 2*1; box != want {
		t.Errorf("width = %v, want %v (label %v)", box, want, label)
	}
	// An avatar and a delete icon add their 24 pixels (the content height is 32 - 16 + 16 = 32).
	ts = pumpButton(t, standardTheme(true), NewChip(chipLabel("Chip"), ChipOpts{
		Avatar: widgets.NewSizedBox(nil), OnDeleted: func() {}}))
	ts.PumpAndSettle()
	withBoth := ts.Size(chipBox(ts, gluttertest.ByType[*Chip]())).Width
	if withBoth <= box+32 {
		t.Errorf("width with an avatar and a delete icon = %v, want more than %v", withBoth, box+32)
	}
}

func TestActionChip(t *testing.T) {
	pressed := 0
	ts := pumpButton(t, standardTheme(true), NewActionChip(chipLabel("Action"), ActionChipOpts{OnPressed: func() { pressed++ }}))
	ts.Tap(gluttertest.ByText("Action"))
	ts.PumpAndSettle()
	if pressed != 1 {
		t.Errorf("pressed %d times, want 1", pressed)
	}
	// Taps in the padding of the tap target reach the chip.
	chip := gluttertest.ByType[*ActionChip]()
	top := ts.TopLeft(chip)
	ts.TapAt(ui.Offset{Dx: ts.Center(chip).Dx, Dy: top.Dy + 2})
	ts.PumpAndSettle()
	if pressed != 2 {
		t.Errorf("a tap in the padding of the tap target pressed %d times, want 2", pressed)
	}
	t.Run("disabled", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewActionChip(chipLabel("Action")))
		ts.Tap(gluttertest.ByText("Action"))
		ts.PumpAndSettle()
		if gluttertest.WidgetAs[*ActionChip](ts, gluttertest.ByType[*ActionChip]()).IsEnabled() {
			t.Error("a chip without OnPressed is enabled")
		}
	})
}

func TestChipVariantsColorsAndElevation(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name          string
		chip          widgets.Widget
		wantElevation float64
		wantColor     foundation.Opt[ui.Color]
	}{
		{"flat action chip", NewActionChip(chipLabel("A"), ActionChipOpts{OnPressed: func() {}}), 0, foundation.None[ui.Color]()},
		{"elevated action chip", NewActionChipElevated(chipLabel("A"), ActionChipOpts{OnPressed: func() {}}), 1, foundation.Some(cs.SurfaceContainerLow())},
		{"disabled elevated action chip", NewActionChipElevated(chipLabel("A")), 0, foundation.Some(cs.OnSurface().WithOpacity(0.12))},
		{"flat filter chip", NewFilterChip(chipLabel("A"), FilterChipOpts{OnSelected: func(bool) {}}), 0, foundation.None[ui.Color]()},
		{"elevated filter chip", NewFilterChipElevated(chipLabel("A"), FilterChipOpts{OnSelected: func(bool) {}}), 1, foundation.Some(cs.SurfaceContainerLow())},
		{"selected filter chip", NewFilterChip(chipLabel("A"), FilterChipOpts{Selected: true, OnSelected: func(bool) {}}), 0, foundation.Some(cs.SecondaryContainer())},
		{"selected choice chip", NewChoiceChip(chipLabel("A"), true, ChoiceChipOpts{OnSelected: func(bool) {}}), 0, foundation.Some(cs.SecondaryContainer())},
		{"elevated choice chip", NewChoiceChipElevated(chipLabel("A"), false, ChoiceChipOpts{OnSelected: func(bool) {}}), 1, foundation.Some(cs.SurfaceContainerLow())},
		{"selected input chip", NewInputChip(chipLabel("A"), InputChipOpts{Selected: true, OnSelected: func(bool) {}}), 0, foundation.Some(cs.SecondaryContainer())},
		{"chip", NewChip(chipLabel("A")), 0, foundation.None[ui.Color]()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, theme, tc.chip)
			ts.PumpAndSettle()
			raw := gluttertest.ByType[*RawChip]()
			box := gluttertest.WidgetAs[*Material](ts, chipBox(ts, raw))
			if box.Elevation() != tc.wantElevation {
				t.Errorf("elevation = %v, want %v", box.Elevation(), tc.wantElevation)
			}
			decoration := gluttertest.WidgetAs[*Ink](ts, gluttertest.First(gluttertest.Descendant(raw, gluttertest.ByType[*Ink]()))).Decoration()
			shape, ok := decoration.(*painting.ShapeDecoration)
			if !ok {
				t.Fatalf("the decoration is a %T", decoration)
			}
			if shape.Color != tc.wantColor {
				t.Errorf("color = %v, want %v", shape.Color, tc.wantColor)
			}
		})
	}
}

func TestFilterChipSelection(t *testing.T) {
	var got []bool
	selected := false
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return NewFilterChip(chipLabel("Filter"), FilterChipOpts{Selected: selected, OnSelected: func(v bool) {
				got = append(got, v)
				setState(func() { selected = v })
			}})
		}))})}))
	chip := gluttertest.ByType[*FilterChip]()
	unselected := ts.Size(chipBox(ts, chip)).Width
	ts.Tap(gluttertest.ByText("Filter"))
	ts.PumpAndSettle()
	if len(got) != 1 || !got[0] {
		t.Fatalf("onSelected = %v, want [true]", got)
	}
	// The check mark makes room for itself: the 18 pixels of the content height,
	// with a gap of 0.
	if width := ts.Size(chipBox(ts, chip)).Width; width <= unselected {
		t.Errorf("a selected chip is %v wide, the unselected one %v: the check mark has no room", width, unselected)
	}
	ts.Tap(gluttertest.ByText("Filter"))
	ts.PumpAndSettle()
	if len(got) != 2 || got[1] {
		t.Errorf("onSelected = %v, want [true false]", got)
	}
	if width := ts.Size(chipBox(ts, chip)).Width; width != unselected {
		t.Errorf("width after unselecting = %v, want %v", width, unselected)
	}
}

func TestInputChipDelete(t *testing.T) {
	deleted, pressed := 0, 0
	ts := pumpButton(t, standardTheme(true), NewInputChip(chipLabel("Input"), InputChipOpts{
		OnDeleted: func() { deleted++ }, OnPressed: func() { pressed++ },
		Avatar: widgets.NewSizedBox(nil),
	}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByIcon(Icons.Clear()))
	ts.PumpAndSettle()
	if deleted != 1 || pressed != 0 {
		t.Errorf("deleted %d times and pressed %d times, want 1 and 0", deleted, pressed)
	}
	ts.Tap(gluttertest.ByText("Input"))
	ts.PumpAndSettle()
	if deleted != 1 || pressed != 1 {
		t.Errorf("deleted %d times and pressed %d times, want 1 and 1", deleted, pressed)
	}
	// A tap on the right edge of the chip is for the delete icon.
	chip := gluttertest.ByType[*InputChip]()
	box := chipBox(ts, chip)
	right := ts.BottomRight(box)
	ts.TapAt(ui.Offset{Dx: right.Dx - 10, Dy: ts.Center(box).Dy})
	ts.PumpAndSettle()
	if deleted != 2 {
		t.Errorf("a tap near the right edge deleted %d times in all, want 2", deleted)
	}
}

func TestChipDeleteIconTooltipAndSemantics(t *testing.T) {
	ts := pumpButton(t, standardTheme(true), NewChip(chipLabel("Chip"), ChipOpts{OnDeleted: func() {}}))
	ts.PumpAndSettle()
	handle := ts.EnsureSemantics()
	defer handle.Dispose()
	ts.Pump()
	if !ts.Exists(gluttertest.ByIcon(Icons.Cancel())) {
		t.Error("the default delete icon is not shown")
	}
}

func TestRawChipRejectsBothCallbacks(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(materialApp(nil, NewRawChip(chipLabel("x"), RawChipOpts{OnPressed: func() {}, OnSelected: func(bool) {}})))
	wantRoute(t, recoverError(func() { takeAndPanic(ts) }), "RawChipSelectedAndPressed")
}

func TestChipElevationErrors(t *testing.T) {
	skipUnlessDebug(t)
	neg := foundation.Some(-1.0)
	l := chipLabel("x")
	tests := []struct {
		name      string
		build     func()
		wantRoute string
	}{
		{"chip", func() { NewChip(l, ChipOpts{Elevation: neg}) }, "ChipElevationNegative"},
		{"raw chip", func() { NewRawChip(l, RawChipOpts{Elevation: neg}) }, "RawChipElevationNegative"},
		{"raw chip press", func() { NewRawChip(l, RawChipOpts{PressElevation: neg}) }, "RawChipPressElevationNegative"},
		{"action chip", func() { NewActionChip(l, ActionChipOpts{Elevation: neg}) }, "ActionChipElevationNegative"},
		{"action chip press", func() { NewActionChipElevated(l, ActionChipOpts{PressElevation: neg}) }, "ActionChipPressElevationNegative"},
		{"choice chip", func() { NewChoiceChip(l, false, ChoiceChipOpts{Elevation: neg}) }, "ChoiceChipElevationNegative"},
		{"choice chip press", func() { NewChoiceChipElevated(l, false, ChoiceChipOpts{PressElevation: neg}) }, "ChoiceChipPressElevationNegative"},
		{"filter chip", func() { NewFilterChip(l, FilterChipOpts{Elevation: neg}) }, "FilterChipElevationNegative"},
		{"filter chip press", func() { NewFilterChipElevated(l, FilterChipOpts{PressElevation: neg}) }, "FilterChipPressElevationNegative"},
		{"input chip", func() { NewInputChip(l, InputChipOpts{Elevation: neg}) }, "InputChipElevationNegative"},
		{"input chip press", func() { NewInputChip(l, InputChipOpts{PressElevation: neg}) }, "InputChipPressElevationNegative"},
		{"theme defaults without either", func() { ChipThemeDataFromDefaults(ui.Color{}, nil, ChipThemeDataFromDefaultsOpts{}) }, "ChipThemeBrightnessAndColor"},
		{"theme defaults with both", func() {
			ChipThemeDataFromDefaults(ui.Color{}, nil, ChipThemeDataFromDefaultsOpts{Brightness: ui.BrightnessDark, PrimaryColor: foundation.Some(ui.NewColor(0xff000000))})
		}, "ChipThemeBrightnessAndColor"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.wantRoute)
		})
	}
}

func TestChipsInM2(t *testing.T) {
	// The Material 2 chips take their colors from the original defaults.
	theme := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false), VisualDensity: foundation.Some(VisualDensityStandard())})
	ts := pumpButton(t, theme, NewChoiceChip(chipLabel("Choice"), true, ChoiceChipOpts{OnSelected: func(bool) {}}))
	ts.PumpAndSettle()
	if h := ts.Size(chipBox(ts, gluttertest.ByType[*ChoiceChip]())).Height; h != 32 {
		t.Errorf("height = %v, want 32", h)
	}
}
