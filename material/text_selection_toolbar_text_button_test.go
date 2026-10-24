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

// These tests port material/text_selection_toolbar_text_button_test.dart.

func TestTextSelectionToolbarTextButtonPositionChangesWidth(t *testing.T) {
	ts := newToolbarTester(t)
	index, total := 1, 3
	setState := pumpToolbarApp(ts, ui.TextDirectionLTR, func() widgets.Widget {
		return widgets.NewCenter(NewTextSelectionToolbarTextButton(TextSelectionToolbarTextButtonGetPadding(index, total), widgets.NewText("button")))
	})
	find := gluttertest.ByType[*TextSelectionToolbarTextButton]()

	middleSize := ts.Size(find)

	setState(func() { index, total = 0, 3 })
	ts.Pump()
	firstSize := ts.Size(find)
	if !(firstSize.Width > middleSize.Width) {
		t.Errorf("first %v is not wider than middle %v", firstSize.Width, middleSize.Width)
	}

	setState(func() { index, total = 2, 3 })
	ts.Pump()
	lastSize := ts.Size(find)
	if !(lastSize.Width > middleSize.Width) {
		t.Errorf("last %v is not wider than middle %v", lastSize.Width, middleSize.Width)
	}
	if lastSize.Width != firstSize.Width {
		t.Errorf("last %v is not as wide as first %v", lastSize.Width, firstSize.Width)
	}

	setState(func() { index, total = 0, 1 })
	ts.Pump()
	onlySize := ts.Size(find)
	if !(onlySize.Width > middleSize.Width && onlySize.Width > firstSize.Width && onlySize.Width > lastSize.Width) {
		t.Errorf("only %v is not the widest (middle %v, first %v, last %v)", onlySize.Width, middleSize.Width, firstSize.Width, lastSize.Width)
	}
}

func TestTextSelectionToolbarTextButtonColors(t *testing.T) {
	schemes := []struct {
		name          string
		scheme        func() *ColorScheme
		wantForegound ui.Color
	}{
		{"light", func() *ColorScheme { return NewThemeData().ColorScheme }, ui.NewColor(0xff000000)},
		{"dark", func() *ColorScheme { return NewThemeDataDark().ColorScheme }, ui.NewColor(0xffffffff)},
	}
	pump := func(t *testing.T, theme *ThemeData) *TextButton {
		ts := newToolbarTester(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewTextSelectionToolbarTextButton(TextSelectionToolbarTextButtonGetPadding(0, 1), widgets.NewText("button")))})}))
		if got := ts.Count(gluttertest.ByType[*TextButton]()); got != 1 {
			t.Fatalf("text buttons = %d, want 1", got)
		}
		return ts.Widget(gluttertest.ByType[*TextButton]()).(*TextButton)
	}
	for _, tc := range schemes {
		t.Run("foreground color by default "+tc.name, func(t *testing.T) {
			button := pump(t, NewThemeData(ThemeDataOpts{ColorScheme: tc.scheme()}))
			// The foreground color is hardcoded to black or white by default, not
			// the default value from ColorScheme.onSurface.
			if got := button.Style().ForegroundColor.Resolve(0); got != foundation.Some(tc.wantForegound) {
				t.Errorf("foreground color = %v, want %v", got, tc.wantForegound)
			}
		})
		t.Run("custom foreground color "+tc.name, func(t *testing.T) {
			custom := Colors.Red.Shade500()
			button := pump(t, NewThemeData(ThemeDataOpts{ColorScheme: tc.scheme().CopyWith(ColorSchemeCopy{OnSurface: foundation.Some(custom)})}))
			if got := button.Style().ForegroundColor.Resolve(0); got != foundation.Some(custom) {
				t.Errorf("foreground color = %v, want %v", got, custom)
			}
		})
		t.Run("background color by default "+tc.name, func(t *testing.T) {
			// Regression test for https://github.com/flutter/flutter/issues/133027
			button := pump(t, NewThemeData(ThemeDataOpts{ColorScheme: tc.scheme()}))
			// The background color is hardcoded to transparent by default so the
			// buttons are the color of the container behind them. For example
			// TextSelectionToolbar hardcodes the color value, and
			// TextSelectionToolbarTextButton that are its children should be that
			// color.
			if got := button.Style().BackgroundColor.Resolve(0); got != foundation.Some(ui.NewColor(0x00000000)) {
				t.Errorf("background color = %v, want transparent", got)
			}
		})
		t.Run("textButtonTheme should not override default background color "+tc.name, func(t *testing.T) {
			// Regression test for https://github.com/flutter/flutter/issues/133027
			blue := Colors.Blue.Shade500()
			button := pump(t, NewThemeData(ThemeDataOpts{ColorScheme: tc.scheme(), TextButtonTheme: NewTextButtonThemeData(TextButtonThemeDataOpts{
				Style: &ButtonStyle{BackgroundColor: widgets.NewWidgetStatePropertyAll(foundation.Some(blue))}})}))
			if got := button.Style().BackgroundColor.Resolve(0); got != foundation.Some(ui.NewColor(0x00000000)) {
				t.Errorf("background color = %v, want transparent", got)
			}
		})
	}
}

func TestTextSelectionToolbarTextButtonDoesNotCrashAtZeroArea(t *testing.T) {
	ts := newToolbarTester(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewCenter(widgets.NewSizedBoxShrink(
		NewTextSelectionToolbarTextButton(painting.NewEdgeInsetsAll(5), widgets.NewText("X"))))}))
	if got := ts.Size(gluttertest.ByType[*TextSelectionToolbarTextButton]()); got != ui.SizeZero {
		t.Errorf("size = %v, want zero", got)
	}
}

func TestTextSelectionToolbarTextButtonGetPadding(t *testing.T) {
	const middle, end = 9.5, 14.5
	tests := []struct {
		name         string
		index, total int
		start, end   float64
	}{
		{"only", 0, 1, end, end},
		{"first", 0, 3, end, middle},
		{"middle", 1, 3, middle, middle},
		{"last", 2, 3, middle, end},
		{"first of two", 0, 2, end, middle},
		{"last of two", 1, 2, middle, end},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := painting.NewEdgeInsetsDirectionalFromSTEB(tc.start, 0, tc.end, 0)
			if got := TextSelectionToolbarTextButtonGetPadding(tc.index, tc.total); got != painting.EdgeInsetsGeometry(want) {
				t.Errorf("padding = %v, want %v", got, want)
			}
		})
	}
}

func TestTextSelectionToolbarTextButtonGetPaddingNeedsAValidPosition(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name         string
		index, total int
	}{
		{"no buttons", 0, 0}, {"negative index", -1, 2}, {"index past the end", 2, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(func() { TextSelectionToolbarTextButtonGetPadding(tc.index, tc.total) }),
				"TextSelectionToolbarTextButtonGetPadding.TextSelectionToolbarTextButtonPosition")
		})
	}
}

func TestTextSelectionToolbarTextButtonCopyWith(t *testing.T) {
	pressed := ""
	original := NewTextSelectionToolbarTextButton(painting.NewEdgeInsetsAll(1), widgets.NewText("one"),
		TextSelectionToolbarTextButtonOpts{OnPressed: func() { pressed = "original" }, Alignment: painting.AlignmentCenterLeft})
	newChild := widgets.NewText("two")
	tests := []struct {
		name          string
		copy          TextSelectionToolbarTextButtonCopy
		wantChild     widgets.Widget
		wantPadding   painting.EdgeInsetsGeometry
		wantAlignment painting.AlignmentGeometry
		wantPressed   string
	}{
		{"nothing", TextSelectionToolbarTextButtonCopy{}, original.Child(), painting.NewEdgeInsetsAll(1), painting.AlignmentCenterLeft, "original"},
		{"everything", TextSelectionToolbarTextButtonCopy{Child: newChild, Padding: painting.NewEdgeInsetsAll(2), Alignment: painting.AlignmentCenter,
			OnPressed: func() { pressed = "copy" }}, newChild, painting.NewEdgeInsetsAll(2), painting.AlignmentCenter, "copy"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := original.CopyWith(tc.copy)
			if got.Child() != tc.wantChild || got.Padding() != tc.wantPadding || got.Alignment() != tc.wantAlignment {
				t.Errorf("copy = child %v padding %v alignment %v", got.Child(), got.Padding(), got.Alignment())
			}
			got.OnPressed()()
			if pressed != tc.wantPressed {
				t.Errorf("pressed = %q, want %q", pressed, tc.wantPressed)
			}
		})
	}
}

func TestTextSelectionToolbarTextButtonPressAndDisable(t *testing.T) {
	taps := 0
	tests := []struct {
		name      string
		onPressed func()
		wantTaps  int
	}{
		{"enabled", func() { taps++ }, 1},
		{"disabled without a callback", nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			taps = 0
			ts := newToolbarTester(t)
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				NewTextSelectionToolbarTextButton(painting.NewEdgeInsetsAll(5), widgets.NewText("press"), TextSelectionToolbarTextButtonOpts{OnPressed: tc.onPressed}))})}))
			ts.Tap(gluttertest.ByText("press"))
			ts.Pump()
			if taps != tc.wantTaps {
				t.Errorf("taps = %d, want %d", taps, tc.wantTaps)
			}
		})
	}
}
