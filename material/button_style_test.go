// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	colorA = ui.NewColor(0xFF102030)
	colorB = ui.NewColor(0xFF708090)
)

func TestButtonStyleCopyWithAndMerge(t *testing.T) {
	a := NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorA), IconAlignment: IconAlignmentStart, AnimationDuration: foundation.Some(time.Second)})
	b := NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorB), ForegroundColor: allOpt(colorB), IconAlignment: IconAlignmentEnd})

	t.Run("copyWith: the argument wins", func(t *testing.T) {
		got := a.CopyWith(ButtonStyleOpts{BackgroundColor: allOpt(colorB), TapTargetSize: MaterialTapTargetSizePadded})
		if v := got.BackgroundColor.Resolve(widgets.NewWidgetStates()); v != foundation.Some(colorB) {
			t.Errorf("background = %v, want %v", v, colorB)
		}
		if got.TapTargetSize != MaterialTapTargetSizePadded || got.IconAlignment != IconAlignmentStart || got.AnimationDuration != foundation.Some(time.Second) {
			t.Errorf("copy lost or changed properties: %v", got)
		}
		if v := a.BackgroundColor.Resolve(widgets.NewWidgetStates()); v != foundation.Some(colorA) {
			t.Error("the original style was changed")
		}
	})
	t.Run("merge: the receiver wins", func(t *testing.T) {
		got := a.Merge(b)
		if v := got.BackgroundColor.Resolve(widgets.NewWidgetStates()); v != foundation.Some(colorA) {
			t.Errorf("background = %v, want the one of the receiver", v)
		}
		if v := got.ForegroundColor.Resolve(widgets.NewWidgetStates()); v != foundation.Some(colorB) {
			t.Errorf("foreground = %v, want the one of the argument", v)
		}
		if got.IconAlignment != IconAlignmentStart {
			t.Errorf("icon alignment = %v, want the one of the receiver", got.IconAlignment)
		}
	})
	t.Run("merge with nil", func(t *testing.T) {
		if got := a.Merge(nil); !got.Equal(a) {
			t.Errorf("a.Merge(nil) = %v, want a", got)
		}
	})
}

func TestButtonStyleEqual(t *testing.T) {
	prop := allOpt(colorA)
	tests := []struct {
		name string
		a, b *ButtonStyle
		want bool
	}{
		{"both nil", nil, nil, true},
		{"nil and style", nil, NewButtonStyle(), false},
		{"empty styles", NewButtonStyle(), NewButtonStyle(), true},
		{"same property", NewButtonStyle(ButtonStyleOpts{BackgroundColor: prop}), NewButtonStyle(ButtonStyleOpts{BackgroundColor: prop}), true},
		{"equal all properties", NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorA)}), NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorA)}), true},
		{"other color", NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorA)}), NewButtonStyle(ButtonStyleOpts{BackgroundColor: allOpt(colorB)}), false},
		{"other enum", NewButtonStyle(ButtonStyleOpts{IconAlignment: IconAlignmentEnd}), NewButtonStyle(), false},
		{"other density", NewButtonStyle(ButtonStyleOpts{VisualDensity: foundation.Some(VisualDensityCompact())}), NewButtonStyle(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestButtonStyleLerp(t *testing.T) {
	states := widgets.NewWidgetStates()
	a := NewButtonStyle(ButtonStyleOpts{
		BackgroundColor: allOpt(ui.NewColor(0xFF000000)), Elevation: allOpt(2.0), IconSize: allOpt(10.0),
		MinimumSize: allOpt(ui.Size{Width: 10, Height: 20}), TapTargetSize: MaterialTapTargetSizePadded, IconAlignment: IconAlignmentStart,
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsAll(10))})
	b := NewButtonStyle(ButtonStyleOpts{
		BackgroundColor: allOpt(ui.NewColor(0xFFFFFFFF)), Elevation: allOpt(6.0), IconSize: allOpt(30.0),
		MinimumSize: allOpt(ui.Size{Width: 30, Height: 60}), TapTargetSize: MaterialTapTargetSizeShrinkWrap, IconAlignment: IconAlignmentEnd,
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsAll(30))})
	tests := []struct {
		name      string
		t         float64
		wantElev  float64
		wantIcon  float64
		wantMin   ui.Size
		wantBg    ui.Color
		wantTap   MaterialTapTargetSize
		wantAlign IconAlignment
		wantPad   float64
	}{
		{"start", 0, 2, 10, ui.Size{Width: 10, Height: 20}, ui.NewColor(0xFF000000), MaterialTapTargetSizePadded, IconAlignmentStart, 10},
		{"middle", 0.5, 4, 20, ui.Size{Width: 20, Height: 40}, ui.NewColor(0xFF000000).Lerp(ui.NewColor(0xFFFFFFFF), 0.5), MaterialTapTargetSizeShrinkWrap, IconAlignmentEnd, 20},
		{"end", 1, 6, 30, ui.Size{Width: 30, Height: 60}, ui.NewColor(0xFFFFFFFF), MaterialTapTargetSizeShrinkWrap, IconAlignmentEnd, 30},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ButtonStyleLerp(a, b, tc.t)
			if v := got.Elevation.Resolve(states); v != foundation.Some(tc.wantElev) {
				t.Errorf("elevation = %v, want %v", v, tc.wantElev)
			}
			if v := got.IconSize.Resolve(states); v != foundation.Some(tc.wantIcon) {
				t.Errorf("icon size = %v, want %v", v, tc.wantIcon)
			}
			if v := got.MinimumSize.Resolve(states); v != foundation.Some(tc.wantMin) {
				t.Errorf("minimum size = %v, want %v", v, tc.wantMin)
			}
			if v := got.BackgroundColor.Resolve(states); v != foundation.Some(tc.wantBg) {
				t.Errorf("background = %v, want %v", v, tc.wantBg)
			}
			if got.TapTargetSize != tc.wantTap || got.IconAlignment != tc.wantAlign {
				t.Errorf("tap target %v and icon alignment %v, want %v and %v", got.TapTargetSize, got.IconAlignment, tc.wantTap, tc.wantAlign)
			}
			if v := got.Padding.Resolve(states); v.Horizontal() != tc.wantPad*2 {
				t.Errorf("padding = %v, want %v on all sides", v, tc.wantPad)
			}
		})
	}
	if got := ButtonStyleLerp(nil, nil, 0.5); got != nil {
		t.Errorf("lerp of two nils = %v, want nil", got)
	}
	// With one style missing the properties of the other fade in and out.
	if got := ButtonStyleLerp(nil, b, 1); got.Elevation.Resolve(states) != foundation.Some(6.0) {
		t.Errorf("lerp(nil, b, 1) elevation = %v", got.Elevation.Resolve(states))
	}
}

func TestButtonStyleDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	s := NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(2.0), IconAlignment: IconAlignmentEnd, TapTargetSize: MaterialTapTargetSizePadded})
	text := s.String()
	for _, want := range []string{"elevation", "iconAlignment: IconAlignment.end", "tapTargetSize: MaterialTapTargetSize.padded"} {
		if !strings.Contains(text, want) {
			t.Errorf("the description of the style does not have %q:\n%s", want, text)
		}
	}
	if s.ToStringShort() != "ButtonStyle" {
		t.Errorf("ToStringShort = %q", s.ToStringShort())
	}
}

func TestButtonStyleButtonHelpers(t *testing.T) {
	t.Run("AllOrNull", func(t *testing.T) {
		if ButtonStyleButtonAllOrNull(foundation.None[float64]()) != nil {
			t.Error("a property was made of no value")
		}
		if got := ButtonStyleButtonAllOrNull(foundation.Some(3.0)).Resolve(widgets.NewWidgetStates(widgets.WidgetStateHovered)); got != foundation.Some(3.0) {
			t.Errorf("value = %v", got)
		}
	})
	t.Run("DefaultColor", func(t *testing.T) {
		if ButtonStyleButtonDefaultColor(foundation.None[ui.Color](), foundation.None[ui.Color]()) != nil {
			t.Error("a property was made of no colors")
		}
		p := ButtonStyleButtonDefaultColor(foundation.Some(colorA), foundation.Some(colorB))
		if got := p.Resolve(widgets.NewWidgetStates()); got != foundation.Some(colorA) {
			t.Errorf("enabled = %v", got)
		}
		if got := p.Resolve(widgets.NewWidgetStates(widgets.WidgetStateDisabled)); got != foundation.Some(colorB) {
			t.Errorf("disabled = %v", got)
		}
		// Without a disabled color the disabled button has none, so that the next
		// style gives it.
		p = ButtonStyleButtonDefaultColor(foundation.Some(colorA), foundation.None[ui.Color]())
		if got := p.Resolve(widgets.NewWidgetStates(widgets.WidgetStateDisabled)); got.IsSome() {
			t.Errorf("disabled = %v, want none", got)
		}
	})
	t.Run("ScaledPadding", func(t *testing.T) {
		p1, p2, p3 := painting.NewEdgeInsetsAll(10), painting.NewEdgeInsetsAll(20), painting.NewEdgeInsetsAll(40)
		tests := []struct {
			scale float64
			want  float64
		}{{0.5, 10}, {1, 10}, {1.5, 15}, {2, 20}, {2.5, 30}, {3, 40}, {5, 40}}
		for _, tc := range tests {
			if got := ButtonStyleButtonScaledPadding(p1, p2, p3, tc.scale).(painting.EdgeInsets).Left; got != tc.want {
				t.Errorf("padding at a scale of %v = %v, want %v", tc.scale, got, tc.want)
			}
		}
	})
}

func TestButtonStyleFrom(t *testing.T) {
	states := widgets.NewWidgetStates
	hovered, pressed, focused := states(widgets.WidgetStateHovered), states(widgets.WidgetStatePressed), states(widgets.WidgetStateFocused)
	disabled := states(widgets.WidgetStateDisabled)
	tests := []struct {
		name  string
		style *ButtonStyle
		check func(t *testing.T, s *ButtonStyle)
	}{
		{"empty style from sets nothing", TextButtonStyleFrom(ButtonStyleFromOpts{}), func(t *testing.T, s *ButtonStyle) {
			if s.ForegroundColor != nil || s.BackgroundColor != nil || s.OverlayColor != nil || s.MouseCursor != nil || s.Elevation != nil {
				t.Errorf("an empty style has properties: %v", s)
			}
		}},
		{"foreground makes the states of the overlay", TextButtonStyleFrom(ButtonStyleFromOpts{ForegroundColor: foundation.Some(colorA)}), func(t *testing.T, s *ButtonStyle) {
			for _, tc := range []struct {
				st   widgets.WidgetStates
				want foundation.Opt[ui.Color]
			}{{hovered, foundation.Some(colorA.WithOpacity(0.08))}, {pressed, foundation.Some(colorA.WithOpacity(0.1))},
				{focused, foundation.Some(colorA.WithOpacity(0.1))}, {states(), foundation.None[ui.Color]()}} {
				if got := s.OverlayColor.Resolve(tc.st); got != tc.want {
					t.Errorf("overlay in %v = %v, want %v", tc.st, got, tc.want)
				}
			}
		}},
		{"transparent overlay is in every state", TextButtonStyleFrom(ButtonStyleFromOpts{ForegroundColor: foundation.Some(colorA), OverlayColor: foundation.Some(ui.NewColor(0x00FFFFFF))}), func(t *testing.T, s *ButtonStyle) {
			if got := s.OverlayColor.Resolve(hovered); got != foundation.Some(ui.NewColor(0x00FFFFFF)) {
				t.Errorf("overlay = %v", got)
			}
		}},
		{"text: a background without a disabled one is also the disabled", TextButtonStyleFrom(ButtonStyleFromOpts{BackgroundColor: foundation.Some(colorA)}), func(t *testing.T, s *ButtonStyle) {
			if got := s.BackgroundColor.Resolve(disabled); got != foundation.Some(colorA) {
				t.Errorf("disabled background = %v", got)
			}
		}},
		{"elevated: a background without a disabled one leaves the disabled", ElevatedButtonStyleFrom(ButtonStyleFromOpts{BackgroundColor: foundation.Some(colorA)}), func(t *testing.T, s *ButtonStyle) {
			if got := s.BackgroundColor.Resolve(disabled); got.IsSome() {
				t.Errorf("disabled background = %v", got)
			}
		}},
		{"elevated: the elevations of the states", ElevatedButtonStyleFrom(ButtonStyleFromOpts{Elevation: foundation.Some(2.0)}), func(t *testing.T, s *ButtonStyle) {
			for _, tc := range []struct {
				st   widgets.WidgetStates
				want float64
			}{{disabled, 0}, {pressed, 8}, {hovered, 4}, {focused, 4}, {states(), 2}} {
				if got := s.Elevation.Resolve(tc.st); got != foundation.Some(tc.want) {
					t.Errorf("elevation in %v = %v, want %v", tc.st, got, tc.want)
				}
			}
		}},
		{"filled: the elevation is the same in every state", FilledButtonStyleFrom(ButtonStyleFromOpts{Elevation: foundation.Some(2.0)}), func(t *testing.T, s *ButtonStyle) {
			if got := s.Elevation.Resolve(hovered); got != foundation.Some(2.0) {
				t.Errorf("elevation = %v", got)
			}
		}},
		{"icon colors", OutlinedButtonStyleFrom(ButtonStyleFromOpts{IconColor: foundation.Some(colorA), DisabledIconColor: foundation.Some(colorB)}), func(t *testing.T, s *ButtonStyle) {
			if s.IconColor.Resolve(states()) != foundation.Some(colorA) || s.IconColor.Resolve(disabled) != foundation.Some(colorB) {
				t.Error("the icon colors are wrong")
			}
		}},
		{"mouse cursors", TextButtonStyleFrom(ButtonStyleFromOpts{EnabledMouseCursor: widgets.WidgetStateMouseCursorClickable, DisabledMouseCursor: widgets.WidgetStateMouseCursorAdaptiveClickable}), func(t *testing.T, s *ButtonStyle) {
			if s.MouseCursor.Resolve(states()) != widgets.WidgetStateMouseCursorClickable || s.MouseCursor.Resolve(disabled) != widgets.WidgetStateMouseCursorAdaptiveClickable {
				t.Error("the cursors are wrong")
			}
		}},
		{"plain values", TextButtonStyleFrom(ButtonStyleFromOpts{
			Side: foundation.Some(painting.NewBorderSide()), MinimumSize: foundation.Some(ui.Size{Width: 1, Height: 2}), IconSize: foundation.Some(9.0),
			VisualDensity: foundation.Some(VisualDensityCompact()), TapTargetSize: MaterialTapTargetSizePadded, EnableFeedback: foundation.Some(false),
			IconAlignment: IconAlignmentEnd}), func(t *testing.T, s *ButtonStyle) {
			if s.MinimumSize.Resolve(states()) != foundation.Some(ui.Size{Width: 1, Height: 2}) || s.IconSize.Resolve(states()) != foundation.Some(9.0) || s.Side == nil {
				t.Error("the properties are wrong")
			}
			if s.VisualDensity != foundation.Some(VisualDensityCompact()) || s.TapTargetSize != MaterialTapTargetSizePadded || s.EnableFeedback != foundation.Some(false) || s.IconAlignment != IconAlignmentEnd {
				t.Error("the values are wrong")
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { tc.check(t, tc.style) })
	}
}
