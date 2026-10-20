// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
)

// RowOpts are the optional parameters of NewRow; they are those of NewFlex.
//
// Flutter: widgets/basic.dart Row.new
type RowOpts = FlexOpts

// Row displays its children in a horizontal array.
//
// Flutter: widgets/basic.dart Row
type Row struct{ Flex }

// NewRow creates a horizontal array of children.
//
// Flutter: widgets/basic.dart Row.new
func NewRow(children []Widget, opts ...RowOpts) *Row {
	return &Row{*NewFlex(painting.AxisHorizontal, children, opts...)}
}

// ColumnOpts are the optional parameters of NewColumn; they are those of
// NewFlex.
//
// Flutter: widgets/basic.dart Column.new
type ColumnOpts = FlexOpts

// Column displays its children in a vertical array.
//
// Flutter: widgets/basic.dart Column
type Column struct{ Flex }

// NewColumn creates a vertical array of children.
//
// Flutter: widgets/basic.dart Column.new
func NewColumn(children []Widget, opts ...ColumnOpts) *Column {
	return &Column{*NewFlex(painting.AxisVertical, children, opts...)}
}

// FlexibleOpts are the optional parameters of NewFlexible.
//
// Flutter: widgets/basic.dart Flexible.new
type FlexibleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Flex is the flex factor to use for this child; the default is 1. A child
	// with flex 0 is not flexible.
	Flex foundation.Opt[int]
	// Fit is how a flexible child is inscribed into the available space; the
	// default is FlexFitLoose.
	Fit rendering.FlexFit
}

// Flexible controls how a child of a Row, Column or Flex flexes.
//
// Flutter: widgets/basic.dart Flexible
type Flexible struct {
	ParentDataWidgetOf[*rendering.FlexParentData]
	flex int
	fit  rendering.FlexFit
}

// NewFlexible creates a widget that controls how a child of a Row, Column or Flex
// flexes.
//
// Flutter: widgets/basic.dart Flexible.new
func NewFlexible(child Widget, opts ...FlexibleOpts) *Flexible {
	o := firstOpts(opts)
	if o.Fit == rendering.FlexFitUnspecified {
		o.Fit = rendering.FlexFitLoose
	}
	return &Flexible{NewParentDataWidgetBase[*rendering.FlexParentData](child, WidgetOpts{Key: o.Key}), o.Flex.OrElse(1), o.Fit}
}

// Flex is the flex factor to use for this child.
//
// Flutter: widgets/basic.dart Flexible.flex
func (w *Flexible) Flex() int { return w.flex }

// Fit is how a flexible child is inscribed into the available space.
//
// Flutter: widgets/basic.dart Flexible.fit
func (w *Flexible) Fit() rendering.FlexFit { return w.fit }

// ApplyParentData writes the flex factor and the fit to the parent data of the
// render object.
//
// Flutter: widgets/basic.dart Flexible.applyParentData
func (w *Flexible) ApplyParentData(renderObject rendering.RenderObject) {
	pd := renderObject.ParentData().(*rendering.FlexParentData)
	needsLayout := false
	if pd.Flex != foundation.Some(w.flex) {
		pd.Flex = foundation.Some(w.flex)
		needsLayout = true
	}
	if pd.Fit != w.fit {
		pd.Fit = w.fit
		needsLayout = true
	}
	if needsLayout {
		if p := renderObject.AsRenderObject().Parent(); p != nil {
			p.MarkNeedsLayout()
		}
	}
}

// DebugTypicalAncestorWidgetClass is the name of the widget this one belongs in.
//
// Flutter: widgets/basic.dart Flexible.debugTypicalAncestorWidgetClass
func (*Flexible) DebugTypicalAncestorWidgetClass() string { return "Flex" }

// ExpandedOpts are the optional parameters of NewExpanded.
//
// Flutter: widgets/basic.dart Expanded.new
type ExpandedOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Flex is the flex factor to use for this child; the default is 1.
	Flex foundation.Opt[int]
}

// Expanded expands a child of a Row, Column or Flex so that the child fills the
// available space.
//
// Flutter: widgets/basic.dart Expanded
type Expanded struct{ Flexible }

// NewExpanded creates a widget that expands a child of a Row, Column or Flex.
//
// Flutter: widgets/basic.dart Expanded.new
func NewExpanded(child Widget, opts ...ExpandedOpts) *Expanded {
	o := firstOpts(opts)
	return &Expanded{*NewFlexible(child, FlexibleOpts{Key: o.Key, Flex: o.Flex, Fit: rendering.FlexFitTight})}
}

// SpacerOpts are the optional parameters of NewSpacer.
//
// Flutter: widgets/spacer.dart Spacer.new
type SpacerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Flex is the flex factor to use in determining how much space to take up;
	// the default is 1. It must be above zero.
	Flex foundation.Opt[int]
}

// Spacer creates an adjustable, empty spacer that can be used to tune the
// spacing between widgets in a Flex container, like Row or Column.
//
// Flutter: widgets/spacer.dart Spacer
type Spacer struct {
	StatelessWidget
	flex int
}

// NewSpacer creates a spacer that takes up space proportional to its flex.
//
// Flutter: widgets/spacer.dart Spacer.new
func NewSpacer(opts ...SpacerOpts) *Spacer {
	o := firstOpts(opts)
	flex := o.Flex.OrElse(1)
	if foundation.DebugMode && flex <= 0 {
		panic(errors.New("SpacerFlexInvalid", "the flex of a Spacer must be above zero", "flex", flex))
	}
	return &Spacer{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), flex}
}

// Flex is the flex factor of the space.
//
// Flutter: widgets/spacer.dart Spacer.flex
func (w *Spacer) Flex() int { return w.flex }

// Build builds an Expanded with an empty box.
//
// Flutter: widgets/spacer.dart Spacer.build
func (w *Spacer) Build(BuildContext) Widget {
	return NewExpanded(NewSizedBoxShrink(nil), ExpandedOpts{Flex: foundation.Some(w.flex)})
}
