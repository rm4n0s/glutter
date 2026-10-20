// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// column is the colors of the screen along a vertical line.
func column(ts *gluttertest.Tester, x, fromY, toY int) []color.RGBA {
	var out []color.RGBA
	for y := fromY; y <= toY; y++ {
		out = append(out, pixelAt(ts, x, y))
	}
	return out
}

func pixelAt(ts *gluttertest.Tester, x, y int) color.RGBA { return ts.Screenshot().RGBAAt(x, y) }

func allEqual(cs []color.RGBA) bool {
	for _, c := range cs {
		if c != cs[0] {
			return false
		}
	}
	return true
}

func TestOutlineInputBorderLeavesAGapForTheFloatingLabel(t *testing.T) {
	tests := []struct {
		name string
		dir  ui.TextDirection
	}{
		{"ltr", ui.TextDirectionLTR},
		{"rtl", ui.TextDirectionRTL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{
				theme:         m3Standard(),
				textDirection: tc.dir,
				isFocused:     true,
				decoration:    InputDecoration{LabelText: someStr(testLabelText), Border: NewOutlineInputBorder()},
			})
			ts.Pump()
			container := containerBox(ts).LocalToGlobal(ui.Offset{}, nil)
			label := labelRect(ts)
			top := int(container.Dy)
			// The gap is the width of the label and the padding on both sides of it,
			// which the border leaves out: the column just before the label (after
			// it in a right to left field) has no border.
			x := int(label.Left) - 2
			if tc.dir == ui.TextDirectionRTL {
				x = int(label.Right) + 2
			}
			inGap := column(ts, x, top, top+3)
			outside := column(ts, int(container.Dx)+int(containerBox(ts).Size().Width)/2, top, top+3)
			if allEqual(outside) {
				t.Fatalf("no border is drawn along the top edge: %v", outside)
			}
			if !allEqual(inGap) {
				t.Errorf("the border is drawn under the label: %v", inGap)
			}
		})
	}
}

func TestInputBordersBehavior(t *testing.T) {
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFFFF0000)), Width: foundation.Some(3.0)})
	outline := NewOutlineInputBorder(OutlineInputBorderOpts{BorderSide: side, GapPadding: foundation.Some(6.0)})
	underline := NewUnderlineInputBorder(UnderlineInputBorderOpts{BorderSide: side})
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"outline side", outline.BorderSide(), side},
		{"outline gap padding", outline.GapPadding(), 6.0},
		{"outline is an outline", outline.IsOutline(), true},
		{"underline is not an outline", underline.IsOutline(), false},
		{"none is not an outline", InputBorderNone.IsOutline(), false},
		{"none has no dimensions", InputBorderNone.Dimensions(), painting.EdgeInsetsGeometry(painting.EdgeInsetsZero)},
		{"copy with side", outline.CopyWithBorderSide(painting.BorderSideNone).BorderSide().Width, 0.0},
		{"scaled outline", outline.Scale(2).(OutlineInputBorder).GapPadding(), 12.0},
		{"underline dimensions", underline.Dimensions(), painting.EdgeInsetsGeometry(painting.EdgeInsets{Bottom: 3})},
		{"outline dimensions", outline.Dimensions(), painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(3))},
		{"none to string", InputBorderNone.String(), "InputBorder.none"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
	// The default border sides are the ones of the specification.
	if w := NewOutlineInputBorder().BorderSide().Width; w != 1.0 {
		t.Errorf("default width = %v, want 1", w)
	}
}

func TestInputBordersInterpolate(t *testing.T) {
	a := NewOutlineInputBorder(OutlineInputBorderOpts{GapPadding: foundation.Some(2.0)})
	b := NewOutlineInputBorder(OutlineInputBorderOpts{GapPadding: foundation.Some(10.0)})
	// The gap padding is the one of the end border, as in Flutter.
	if got := a.LerpTo(b, 0.5).(OutlineInputBorder).GapPadding(); got != 10 {
		t.Errorf("gap padding at the middle = %v, want 10", got)
	}
	u := NewUnderlineInputBorder()
	// There is no interpolation between an outline and an underline: nil asks
	// the other border, and then the scaling of both.
	if got := a.LerpTo(u, 0.25); got != nil {
		t.Errorf("the lerp to an underline is %v, want nil", got)
	}
	if got := u.LerpFrom(nil, 0.5); got == nil {
		t.Error("the lerp from nil is nil")
	}
	if got := a.LerpFrom(nil, 0.5); got == nil {
		t.Error("the lerp of the outline from nil is nil")
	}
}

func TestWidgetStateInputBorder(t *testing.T) {
	focused := NewOutlineInputBorder(OutlineInputBorderOpts{BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(4.0)})})
	b := WidgetStateInputBorderResolveWith(func(states widgets.WidgetStates) InputBorder {
		if states.Has(widgets.WidgetStateFocused) {
			return focused
		}
		return NewOutlineInputBorder()
	})
	if got := b.Resolve(widgets.NewWidgetStates(widgets.WidgetStateFocused)).BorderSide().Width; got != 4 {
		t.Errorf("focused width = %v, want 4", got)
	}
	if got := b.Resolve(widgets.NewWidgetStates()).BorderSide().Width; got != 1 {
		t.Errorf("default width = %v, want 1", got)
	}
}
