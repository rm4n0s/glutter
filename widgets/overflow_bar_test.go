// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/overflow_bar_test.dart.

func sizedKeyedBox(key foundation.Key, w, h float64) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Key: key, Width: foundation.Some(w), Height: foundation.Some(h)})
}

func overflowRectOfKey(ts *gluttertest.Tester, key foundation.Key) ui.Rect {
	f := gluttertest.ByKey(key)
	tl, s := ts.TopLeft(f), ts.Size(f)
	return ui.NewRectFromLTWH(tl.Dx, tl.Dy, s.Width, s.Height)
}

func TestOverflowBarDefaults(t *testing.T) {
	bar := NewOverflowBar(nil)
	if bar.Spacing() != 0 || bar.Alignment().IsSome() || bar.OverflowSpacing() != 0 ||
		bar.OverflowDirection() != painting.VerticalDirectionDown || bar.OverflowAlignment() != rendering.OverflowBarAlignmentStart {
		t.Errorf("defaults = %v %v %v %v", bar.Spacing(), bar.Alignment(), bar.OverflowSpacing(), bar.OverflowDirection())
	}
	if len(bar.Children()) != 0 {
		t.Errorf("children = %v", bar.Children())
	}
}

func TestEmptyOverflowBar(t *testing.T) {
	ts := gluttertest.New(t)
	size := ui.Size{Width: 16, Height: 24}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewConstrainedBox(rendering.NewBoxConstraintsTight(size), NewOverflowBar(nil)))))
	if got := ts.Size(gluttertest.ByType[*OverflowBar]()); got != size {
		t.Errorf("size = %v, want %v", got, size)
	}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewOverflowBar(nil))))
	if got := ts.Size(gluttertest.ByType[*OverflowBar]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}

func TestOverflowBarHorizontalLayout(t *testing.T) {
	tests := []struct {
		name    string
		spacing float64
		dir     ui.TextDirection
		want    [3]ui.Rect // children 1, 2 and 3
	}{
		{"ltr", 0, ui.TextDirectionLTR, [3]ui.Rect{ui.NewRectFromLTRB(0, 8, 48, 56), ui.NewRectFromLTRB(48, 0, 112, 64), ui.NewRectFromLTRB(112, 16, 144, 48)}},
		{"ltr spaced", 10, ui.TextDirectionLTR, [3]ui.Rect{ui.NewRectFromLTRB(0, 8, 48, 56), ui.NewRectFromLTRB(58, 0, 122, 64), ui.NewRectFromLTRB(132, 16, 164, 48)}},
		{"rtl", 0, ui.TextDirectionRTL, [3]ui.Rect{ui.NewRectFromLTRB(96, 8, 144, 56), ui.NewRectFromLTRB(32, 0, 96, 64), ui.NewRectFromLTRB(0, 16, 32, 48)}},
		{"rtl spaced", 10, ui.TextDirectionRTL, [3]ui.Rect{ui.NewRectFromLTRB(116, 8, 164, 56), ui.NewRectFromLTRB(42, 0, 106, 64), ui.NewRectFromLTRB(0, 16, 32, 48)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			k := [3]foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
			ts.PumpWidget(NewDirectionality(tc.dir, NewAlign(NewOverflowBar([]Widget{sizedKeyedBox(k[0], 48, 48), sizedKeyedBox(k[1], 64, 64), sizedKeyedBox(k[2], 32, 32)},
				OverflowBarOpts{Spacing: tc.spacing}), AlignOpts{Alignment: painting.AlignmentTopLeft})))
			for i := range k {
				if got := overflowRectOfKey(ts, k[i]); got != tc.want[i] {
					t.Errorf("child %d = %v, want %v", i+1, got, tc.want[i])
				}
			}
		})
	}
}

func TestOverflowBarVerticalLayout(t *testing.T) {
	tests := []struct {
		name string
		o    OverflowBarOpts
		want [3]ui.Rect
	}{
		{"left aligned", OverflowBarOpts{}, [3]ui.Rect{ui.NewRectFromLTRB(0, 0, 48, 48), ui.NewRectFromLTRB(0, 48, 64, 112), ui.NewRectFromLTRB(0, 112, 32, 144)}},
		{"end aligned in rtl is left", OverflowBarOpts{OverflowAlignment: rendering.OverflowBarAlignmentEnd, TextDirection: ui.TextDirectionRTL},
			[3]ui.Rect{ui.NewRectFromLTRB(0, 0, 48, 48), ui.NewRectFromLTRB(0, 48, 64, 112), ui.NewRectFromLTRB(0, 112, 32, 144)}},
		{"spaced", OverflowBarOpts{OverflowSpacing: 10}, [3]ui.Rect{ui.NewRectFromLTRB(0, 0, 48, 48), ui.NewRectFromLTRB(0, 58, 64, 122), ui.NewRectFromLTRB(0, 132, 32, 164)}},
		{"up", OverflowBarOpts{OverflowDirection: painting.VerticalDirectionUp}, [3]ui.Rect{ui.NewRectFromLTRB(0, 96, 48, 144), ui.NewRectFromLTRB(0, 32, 64, 96), ui.NewRectFromLTRB(0, 0, 32, 32)}},
		{"up spaced", OverflowBarOpts{OverflowSpacing: 10, OverflowDirection: painting.VerticalDirectionUp},
			[3]ui.Rect{ui.NewRectFromLTRB(0, 116, 48, 164), ui.NewRectFromLTRB(0, 42, 64, 106), ui.NewRectFromLTRB(0, 0, 32, 32)}},
		{"right aligned", OverflowBarOpts{OverflowAlignment: rendering.OverflowBarAlignmentEnd},
			[3]ui.Rect{ui.NewRectFromLTRB(52, 0, 100, 48), ui.NewRectFromLTRB(36, 48, 100, 112), ui.NewRectFromLTRB(68, 112, 100, 144)}},
		{"start aligned in rtl is right", OverflowBarOpts{TextDirection: ui.TextDirectionRTL},
			[3]ui.Rect{ui.NewRectFromLTRB(52, 0, 100, 48), ui.NewRectFromLTRB(36, 48, 100, 112), ui.NewRectFromLTRB(68, 112, 100, 144)}},
		{"centered", OverflowBarOpts{OverflowAlignment: rendering.OverflowBarAlignmentCenter},
			[3]ui.Rect{ui.NewRectFromLTRB(26, 0, 74, 48), ui.NewRectFromLTRB(18, 48, 82, 112), ui.NewRectFromLTRB(34, 112, 66, 144)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			k := [3]foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewConstrainedBox(
				rendering.NewBoxConstraintsLoose(ui.Size{Width: 100, Height: math.Inf(1)}),
				NewOverflowBar([]Widget{sizedKeyedBox(k[0], 48, 48), sizedKeyedBox(k[1], 64, 64), sizedKeyedBox(k[2], 32, 32)}, tc.o)),
				AlignOpts{Alignment: painting.AlignmentTopLeft})))
			for i := range k {
				if got := overflowRectOfKey(ts, k[i]); got != tc.want[i] {
					t.Errorf("child %d = %v, want %v", i+1, got, tc.want[i])
				}
			}
		})
	}
}

func TestOverflowBarIntrinsicSizes(t *testing.T) {
	children := func() []Widget {
		return []Widget{
			NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(50.0)}),
			NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(64.0), Height: foundation.Some(25.0)}),
			NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(32.0), Height: foundation.Some(75.0)}),
		}
	}
	bar := func() Widget { return NewOverflowBar(children(), OverflowBarOpts{Spacing: 4, OverflowSpacing: 8}) }
	tests := []struct {
		name  string
		child func() Widget
		width float64
		want  func(ui.Size) float64
		value float64
	}{
		{"intrinsic width fits", func() Widget { return NewIntrinsicWidth(bar()) }, 800, func(s ui.Size) float64 { return s.Width }, 152}, // 48 + 4 + 64 + 4 + 32
		{"intrinsic width overflows", func() Widget { return NewIntrinsicWidth(bar()) }, 150, func(s ui.Size) float64 { return s.Width }, 150},
		{"intrinsic height fits", func() Widget { return NewIntrinsicHeight(bar()) }, 800, func(s ui.Size) float64 { return s.Height }, 75},
		{"intrinsic height overflows", func() Widget { return NewIntrinsicHeight(bar()) }, 150, func(s ui.Size) float64 { return s.Height }, 166}, // 50 + 8 + 25 + 8 + 75
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewContainer(tc.child(),
				ContainerOpts{Width: foundation.Some(tc.width), Alignment: painting.AlignmentTopLeft}))))
			if got := tc.want(ts.Size(gluttertest.ByType[*OverflowBar]())); got != tc.value {
				t.Errorf("got %v, want %v", got, tc.value)
			}
		})
	}
}

func TestOverflowBarWiderThanItsIntrinsicWidth(t *testing.T) {
	tests := []struct {
		dir  ui.TextDirection
		want [3]float64
	}{
		{ui.TextDirectionLTR, [3]float64{0, 60, 130}},
		{ui.TextDirectionRTL, [3]float64{750, 680, 600}},
	}
	for _, tc := range tests {
		ts := gluttertest.New(t)
		k := [3]foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
		ts.PumpWidget(NewDirectionality(tc.dir, NewSizedBox(NewOverflowBar([]Widget{sizedKeyedBox(k[0], 50, 50), sizedKeyedBox(k[1], 60, 50), sizedKeyedBox(k[2], 70, 50)},
			OverflowBarOpts{Spacing: 10}), SizedBoxOpts{Width: foundation.Some(800.0)})))
		if got := ts.Size(gluttertest.ByType[*OverflowBar]()); got != (ui.Size{Width: 800, Height: 600}) {
			t.Errorf("%v: size = %v", tc.dir, got)
		}
		for i := range k {
			if got := ts.TopLeft(gluttertest.ByKey(k[i])).Dx; got != tc.want[i] {
				t.Errorf("%v: child %d at %v, want %v", tc.dir, i, got, tc.want[i])
			}
		}
	}
}

// TestOverflowBarAlignmentMatchesRow checks that an OverflowBar with an
// alignment places its children as a Row with the same mainAxisAlignment does.
func TestOverflowBarAlignmentMatchesRow(t *testing.T) {
	alignments := []rendering.MainAxisAlignment{
		rendering.MainAxisAlignmentStart, rendering.MainAxisAlignmentCenter, rendering.MainAxisAlignmentEnd,
		rendering.MainAxisAlignmentSpaceBetween, rendering.MainAxisAlignmentSpaceAround, rendering.MainAxisAlignmentSpaceEvenly,
	}
	for _, alignment := range alignments {
		for _, dir := range []ui.TextDirection{ui.TextDirectionLTR, ui.TextDirectionRTL} {
			t.Run(alignment.String()+"/"+dir.String(), func(t *testing.T) {
				ts := gluttertest.New(t)
				barKeys := [3]foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
				rowKeys := [3]foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
				sizes := [3]float64{50, 70, 80}
				build := func(k [3]foundation.Key) []Widget {
					return []Widget{sizedKeyedBox(k[0], sizes[0], 50), sizedKeyedBox(k[1], sizes[1], 50), sizedKeyedBox(k[2], sizes[2], 50)}
				}
				ts.PumpWidget(NewDirectionality(dir, NewColumn([]Widget{
					NewOverflowBar(build(barKeys), OverflowBarOpts{Alignment: foundation.Some(alignment)}),
					NewRow(build(rowKeys), RowOpts{MainAxisAlignment: alignment}),
				})))
				if got := ts.Size(gluttertest.ByType[*OverflowBar]()); got != (ui.Size{Width: 800, Height: 50}) {
					t.Errorf("size = %v, want 800 x 50", got)
				}
				for i := range barKeys {
					a, b := overflowRectOfKey(ts, barKeys[i]), overflowRectOfKey(ts, rowKeys[i])
					if a.Left != b.Left || a.Right != b.Right || a.Width() != b.Width() || a.Height() != b.Height() {
						t.Errorf("child %d: bar %v, row %v", i, a, b)
					}
				}
			})
		}
	}
}

func TestOverflowBarDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBoxShrink(NewOverflowBar([]Widget{NewText("X"), NewText("Y")})))))
	if got := ts.Size(gluttertest.ByType[*OverflowBar]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
}
