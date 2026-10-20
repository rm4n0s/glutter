// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// OverflowBarOpts are the optional parameters of NewOverflowBar.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.new
type OverflowBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Spacing is the horizontal space between the children; the default is 0.
	Spacing float64
	// Alignment is how the children are placed in the row when they fit; the
	// default (None) sizes the bar to its children and starts them at the start.
	Alignment foundation.Opt[rendering.MainAxisAlignment]
	// OverflowSpacing is the vertical space between the children when they
	// overflow; the default is 0.
	OverflowSpacing float64
	// OverflowAlignment is how the children are aligned when they overflow; the
	// default is start.
	OverflowAlignment rendering.OverflowBarAlignment
	// OverflowDirection is the direction of the column of overflowing children;
	// the default is down.
	OverflowDirection painting.VerticalDirection
	// TextDirection resolves start and end; the default is the one of the
	// closest Directionality.
	TextDirection ui.TextDirection
}

// OverflowBar lays out its children in a row unless they "overflow" the
// available horizontal space, in which case it lays them out in a column
// instead.
//
// Flutter: widgets/overflow_bar.dart OverflowBar
type OverflowBar struct {
	MultiChildRenderObjectWidget
	o OverflowBarOpts
}

// NewOverflowBar creates the widget.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.new
func NewOverflowBar(children []Widget, opts ...OverflowBarOpts) *OverflowBar {
	o := firstOpts(opts)
	return &OverflowBar{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), o: o}
}

// Spacing is the horizontal space between the children.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.spacing
func (w *OverflowBar) Spacing() float64 { return w.o.Spacing }

// Alignment is how the children are placed in the row, when set.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.alignment
func (w *OverflowBar) Alignment() foundation.Opt[rendering.MainAxisAlignment] { return w.o.Alignment }

// OverflowSpacing is the vertical space between overflowing children.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.overflowSpacing
func (w *OverflowBar) OverflowSpacing() float64 { return w.o.OverflowSpacing }

// OverflowAlignment is how overflowing children are aligned.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.overflowAlignment
func (w *OverflowBar) OverflowAlignment() rendering.OverflowBarAlignment {
	if w.o.OverflowAlignment == rendering.OverflowBarAlignmentUnspecified {
		return rendering.OverflowBarAlignmentStart
	}
	return w.o.OverflowAlignment
}

// OverflowDirection is the direction of the column of overflowing children.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.overflowDirection
func (w *OverflowBar) OverflowDirection() painting.VerticalDirection {
	if w.o.OverflowDirection == painting.VerticalDirectionUnspecified {
		return painting.VerticalDirectionDown
	}
	return w.o.OverflowDirection
}

func (w *OverflowBar) textDirection(bc BuildContext) ui.TextDirection {
	if w.o.TextDirection != ui.TextDirectionUnspecified {
		return w.o.TextDirection
	}
	return DirectionalityOf(bc)
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.createRenderObject
func (w *OverflowBar) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderOverflowBar(nil, rendering.RenderOverflowBarOpts{
		Spacing: w.o.Spacing, Alignment: w.o.Alignment, OverflowSpacing: w.o.OverflowSpacing,
		OverflowAlignment: w.OverflowAlignment(), OverflowDirection: w.OverflowDirection(), TextDirection: w.textDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.updateRenderObject
func (w *OverflowBar) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderOverflowBar)
	r.SetSpacing(w.o.Spacing)
	r.SetAlignment(w.o.Alignment)
	r.SetOverflowSpacing(w.o.OverflowSpacing)
	r.SetOverflowAlignment(w.OverflowAlignment())
	r.SetOverflowDirection(w.OverflowDirection())
	r.SetTextDirection(w.textDirection(bc))
}

// DebugFillProperties adds the configuration.
//
// Flutter: widgets/overflow_bar.dart OverflowBar.debugFillProperties
func (w *OverflowBar) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDoubleProperty("spacing", foundation.Some(w.o.Spacing), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	p.Add(foundation.NewEnumProperty("alignment", w.o.Alignment, foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDoubleProperty("overflowSpacing", foundation.Some(w.o.OverflowSpacing), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	p.Add(foundation.NewEnumProperty("overflowAlignment", foundation.Some(w.OverflowAlignment()), foundation.EnumPropertyOpts{DefaultValue: rendering.OverflowBarAlignmentStart}))
	p.Add(foundation.NewEnumProperty("overflowDirection", foundation.Some(w.OverflowDirection()), foundation.EnumPropertyOpts{DefaultValue: painting.VerticalDirectionDown}))
	if w.o.TextDirection != ui.TextDirectionUnspecified {
		p.Add(foundation.NewEnumProperty("textDirection", foundation.Some(w.o.TextDirection), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	}
}
