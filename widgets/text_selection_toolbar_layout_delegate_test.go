// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests port widgets/text_selection_toolbar_layout_delegate_test.dart.

func TestTextSelectionToolbarLayoutDelegatePositionsItselfAtAnchorAboveIfItFits(t *testing.T) {
	ts := gluttertest.New(t)
	const height = 43.0
	const anchorBelowY = 500.0
	anchorAboveY := 0.0
	var setState StateSetter

	ts.PumpWidget(NewMediaQuery(MediaQueryData{}, NewDirectionality(ui.TextDirectionLTR, NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewCustomSingleChildLayout(
			NewTextSelectionToolbarLayoutDelegate(ui.Offset{Dx: 50, Dy: anchorAboveY}, ui.Offset{Dx: 50, Dy: anchorBelowY}),
			NewContainer(nil, ContainerOpts{Width: foundation.Some(200.0), Height: foundation.Some(height), Color: foundation.Some(ui.NewColor(0xffff0000))}))
	}))))

	// When the toolbar doesn't fit above aboveAnchor, it positions itself below
	// belowAnchor.
	if got := ts.TopLeft(gluttertest.ByType[*Container]()).Dy; got != anchorBelowY {
		t.Errorf("toolbar y = %v, want %v", got, anchorBelowY)
	}

	// Even when it barely doesn't fit.
	setState(func() { anchorAboveY = height - 1.0 })
	ts.Pump()
	if got := ts.TopLeft(gluttertest.ByType[*Container]()).Dy; got != anchorBelowY {
		t.Errorf("toolbar y = %v, want %v", got, anchorBelowY)
	}

	// When it does fit above aboveAnchor, it positions itself there.
	setState(func() { anchorAboveY = height })
	ts.Pump()
	if got := ts.TopLeft(gluttertest.ByType[*Container]()).Dy; got != anchorAboveY-height {
		t.Errorf("toolbar y = %v, want %v", got, anchorAboveY-height)
	}
}

func TestTextSelectionToolbarLayoutDelegateCenterOn(t *testing.T) {
	tests := []struct {
		name                       string
		position, width, max, want float64
	}{
		{"centered", 100, 40, 200, 80},
		{"overflows on the left", 10, 40, 200, 0},
		{"overflows on the right", 190, 40, 200, 160},
		{"just fits on the left", 20, 40, 200, 0},
		{"just fits on the right", 180, 40, 200, 160},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TextSelectionToolbarLayoutDelegateCenterOn(tc.position, tc.width, tc.max); got != tc.want {
				t.Errorf("centerOn = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextSelectionToolbarLayoutDelegatePositionForChild(t *testing.T) {
	size, child := ui.Size{Width: 400, Height: 300}, ui.Size{Width: 100, Height: 50}
	above, below := ui.Offset{Dx: 200, Dy: 120}, ui.Offset{Dx: 200, Dy: 180}
	tests := []struct {
		name      string
		fitsAbove foundation.Opt[bool]
		above     ui.Offset
		want      ui.Offset
	}{
		{"calculated: fits above", foundation.None[bool](), above, ui.Offset{Dx: 150, Dy: 70}},
		{"calculated: does not fit above", foundation.None[bool](), ui.Offset{Dx: 200, Dy: 49}, ui.Offset{Dx: 150, Dy: 180}},
		{"forced above even when it does not fit", foundation.Some(true), ui.Offset{Dx: 200, Dy: 20}, ui.Offset{Dx: 150, Dy: 0}},
		{"forced below even when it fits above", foundation.Some(false), above, ui.Offset{Dx: 150, Dy: 180}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewTextSelectionToolbarLayoutDelegate(tc.above, below, TextSelectionToolbarLayoutDelegateOpts{FitsAbove: tc.fitsAbove})
			if got := d.GetPositionForChild(size, child); got != tc.want {
				t.Errorf("position = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextSelectionToolbarLayoutDelegateConstraintsAndRelayout(t *testing.T) {
	a, b := ui.Offset{Dx: 1, Dy: 2}, ui.Offset{Dx: 3, Dy: 4}
	d := NewTextSelectionToolbarLayoutDelegate(a, b)
	tight := rendering.NewBoxConstraintsTight(ui.Size{Width: 50, Height: 60})
	if got := d.GetConstraintsForChild(tight); got.MinWidth != 0 || got.MinHeight != 0 || got.MaxWidth != 50 || got.MaxHeight != 60 {
		t.Errorf("constraints = %v, want them loosened", got)
	}
	if d.AnchorAbove() != a || d.AnchorBelow() != b || d.FitsAbove().IsSome() {
		t.Errorf("accessors = %v %v %v", d.AnchorAbove(), d.AnchorBelow(), d.FitsAbove())
	}
	tests := []struct {
		name  string
		other *TextSelectionToolbarLayoutDelegate
		want  bool
	}{
		{"same", NewTextSelectionToolbarLayoutDelegate(a, b), false},
		{"another anchor above", NewTextSelectionToolbarLayoutDelegate(b, b), true},
		{"another anchor below", NewTextSelectionToolbarLayoutDelegate(a, a), true},
		{"another fitsAbove", NewTextSelectionToolbarLayoutDelegate(a, b, TextSelectionToolbarLayoutDelegateOpts{FitsAbove: foundation.Some(true)}), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := d.ShouldRelayout(tc.other); got != tc.want {
				t.Errorf("ShouldRelayout = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStandardComponentTypeKeys(t *testing.T) {
	types := []StandardComponentType{StandardComponentTypeBackButton, StandardComponentTypeCloseButton, StandardComponentTypeMoreButton, StandardComponentTypeDrawerButton}
	for i, a := range types {
		if a.Key() != a.Key() {
			t.Errorf("the key of %v is not equal to itself", a)
		}
		for _, b := range types[i+1:] {
			if a.Key() == b.Key() {
				t.Errorf("%v and %v have the same key", a, b)
			}
		}
	}
}
