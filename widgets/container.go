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
	"github.com/rm4n0s/glutter/ui"
)

// ContainerOpts are the optional parameters of NewContainer.
//
// Flutter: widgets/container.dart Container.new
type ContainerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Alignment aligns the child within the container; when it is set a
	// container without a child expands to fill the parent.
	Alignment painting.AlignmentGeometry
	// Padding is the empty space to inscribe inside the decoration.
	Padding painting.EdgeInsetsGeometry
	// Color is the background color; it is a shorthand for a BoxDecoration with
	// the color, so it cannot be given together with Decoration.
	Color foundation.Opt[ui.Color]
	// IsAntiAlias is whether to paint the color with anti-aliasing; the default is
	// true.
	IsAntiAlias foundation.Opt[bool]
	// Decoration is the decoration to paint behind the child.
	Decoration painting.Decoration
	// ForegroundDecoration is the decoration to paint in front of the child.
	ForegroundDecoration painting.Decoration
	// Width is a tight width for the container, merged into Constraints.
	Width foundation.Opt[float64]
	// Height is a tight height for the container, merged into Constraints.
	Height foundation.Opt[float64]
	// Constraints are additional constraints to apply to the child.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Margin is the empty space to surround the decoration and the child.
	Margin painting.EdgeInsetsGeometry
	// Transform is the transformation matrix to apply before painting the
	// container.
	Transform *ui.Matrix4
	// TransformAlignment is the alignment of the origin of the transform, relative
	// to the size of the container.
	TransformAlignment painting.AlignmentGeometry
	// ClipBehavior is whether to clip the contents to the decoration's shape
	// (ClipNone, the default, does not clip); it needs a Decoration.
	ClipBehavior ui.Clip
}

// Container is a convenience widget that combines common painting, positioning
// and sizing widgets.
//
// Flutter: widgets/container.dart Container
type Container struct {
	StatelessWidget
	child                Widget
	alignment            painting.AlignmentGeometry
	padding              painting.EdgeInsetsGeometry
	color                foundation.Opt[ui.Color]
	isAntiAlias          bool
	decoration           painting.Decoration
	foregroundDecoration painting.Decoration
	constraints          foundation.Opt[rendering.BoxConstraints]
	margin               painting.EdgeInsetsGeometry
	transform            *ui.Matrix4
	transformAlignment   painting.AlignmentGeometry
	clipBehavior         ui.Clip
}

// NewContainer creates a container. In debug builds it panics with the tags
// ContainerColorAndDecoration (both given), ContainerClipWithoutDecoration, and
// ContainerNegativePadding or the tags of the invalid decoration and
// constraints.
//
// Flutter: widgets/container.dart Container.new
func NewContainer(child Widget, opts ...ContainerOpts) *Container {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if foundation.DebugMode {
		if o.Margin != nil && !o.Margin.IsNonNegative() {
			panic(errors.New("ContainerNegativePadding", "the margin of a Container must not be negative"))
		}
		if o.Padding != nil && !o.Padding.IsNonNegative() {
			panic(errors.New("ContainerNegativePadding", "the padding of a Container must not be negative"))
		}
		if o.Decoration != nil {
			o.Decoration.DebugAssertIsValid()
		}
		if c, ok := o.Constraints.Get(); ok {
			c.DebugAssertIsValid()
		}
		if o.Decoration == nil && o.ClipBehavior != ui.ClipNone {
			panic(errors.New("ContainerClipWithoutDecoration", "a Container with a clipBehavior needs a decoration to clip to"))
		}
		if o.Color.IsSome() && o.Decoration != nil {
			panic(errors.New("ContainerColorAndDecoration", "cannot provide both a color and a decoration; the color argument is just a shorthand for a decoration with the color; to use both a color and other decoration properties, set the color in the BoxDecoration instead"))
		}
	}
	constraints := o.Constraints
	if o.Width.IsSome() || o.Height.IsSome() {
		if c, ok := constraints.Get(); ok {
			constraints = foundation.Some(c.Tighten(rendering.TightenOpts{Width: o.Width, Height: o.Height}))
		} else {
			constraints = foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: o.Width, Height: o.Height}))
		}
	}
	return &Container{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child: child, alignment: o.Alignment,
		padding: o.Padding, color: o.Color, isAntiAlias: o.IsAntiAlias.OrElse(true), decoration: o.Decoration,
		foregroundDecoration: o.ForegroundDecoration, constraints: constraints, margin: o.Margin, transform: o.Transform,
		transformAlignment: o.TransformAlignment, clipBehavior: o.ClipBehavior}
}

// Child is the widget below this one in the tree.
//
// Flutter: widgets/container.dart Container.child
func (w *Container) Child() Widget { return w.child }

// Alignment is how the child is aligned within the container.
//
// Flutter: widgets/container.dart Container.alignment
func (w *Container) Alignment() painting.AlignmentGeometry { return w.alignment }

// Padding is the empty space inside the decoration.
//
// Flutter: widgets/container.dart Container.padding
func (w *Container) Padding() painting.EdgeInsetsGeometry { return w.padding }

// Color is the background color, if any.
//
// Flutter: widgets/container.dart Container.color
func (w *Container) Color() foundation.Opt[ui.Color] { return w.color }

// IsAntiAlias is whether the color is painted with anti-aliasing.
//
// Flutter: widgets/container.dart Container.isAntiAlias
func (w *Container) IsAntiAlias() bool { return w.isAntiAlias }

// Decoration is the decoration painted behind the child.
//
// Flutter: widgets/container.dart Container.decoration
func (w *Container) Decoration() painting.Decoration { return w.decoration }

// ForegroundDecoration is the decoration painted in front of the child.
//
// Flutter: widgets/container.dart Container.foregroundDecoration
func (w *Container) ForegroundDecoration() painting.Decoration { return w.foregroundDecoration }

// Constraints are the additional constraints applied to the child, including
// the tight ones that Width and Height made.
//
// Flutter: widgets/container.dart Container.constraints
func (w *Container) Constraints() foundation.Opt[rendering.BoxConstraints] { return w.constraints }

// Margin is the empty space around the decoration.
//
// Flutter: widgets/container.dart Container.margin
func (w *Container) Margin() painting.EdgeInsetsGeometry { return w.margin }

// Transform is the matrix applied before painting the container.
//
// Flutter: widgets/container.dart Container.transform
func (w *Container) Transform() *ui.Matrix4 { return w.transform }

// TransformAlignment is the alignment of the origin of the transform.
//
// Flutter: widgets/container.dart Container.transformAlignment
func (w *Container) TransformAlignment() painting.AlignmentGeometry { return w.transformAlignment }

// ClipBehavior is whether the contents are clipped to the decoration's shape.
//
// Flutter: widgets/container.dart Container.clipBehavior
func (w *Container) ClipBehavior() ui.Clip { return w.clipBehavior }

func (w *Container) paddingIncludingDecoration() painting.EdgeInsetsGeometry {
	var dp painting.EdgeInsetsGeometry
	if w.decoration != nil {
		dp = w.decoration.Padding()
	}
	switch {
	case w.padding == nil:
		return dp
	case dp == nil:
		return w.padding
	}
	return w.padding.Add(dp)
}

// Build builds the widgets that the configuration needs, from the inside out:
// the alignment, the padding, the color, the clip, the decoration, the
// foreground decoration, the constraints, the margin and the transform.
//
// Flutter: widgets/container.dart Container.build
func (w *Container) Build(bc BuildContext) Widget {
	current := w.child
	if c, ok := w.constraints.Get(); w.child == nil && (!ok || !c.IsTight()) {
		current = NewLimitedBox(NewConstrainedBox(rendering.NewBoxConstraintsExpand(), nil),
			LimitedBoxOpts{MaxWidth: foundation.Some(0.0), MaxHeight: foundation.Some(0.0)})
	} else if w.alignment != nil {
		current = NewAlign(current, AlignOpts{Alignment: w.alignment})
	}
	if p := w.paddingIncludingDecoration(); p != nil {
		current = NewPadding(p, current)
	}
	if c, ok := w.color.Get(); ok {
		current = NewColoredBox(c, current, ColoredBoxOpts{IsAntiAlias: foundation.Some(w.isAntiAlias)})
	}
	if w.clipBehavior != ui.ClipNone {
		current = NewClipPath(current, ClipPathOpts{Clipper: newDecorationClipper(maybeTextDirection(bc), w.decoration),
			ClipBehavior: w.clipBehavior})
	}
	if w.decoration != nil {
		current = NewDecoratedBox(w.decoration, current)
	}
	if w.foregroundDecoration != nil {
		current = NewDecoratedBox(w.foregroundDecoration, current, DecoratedBoxOpts{Position: rendering.DecorationPositionForeground})
	}
	if c, ok := w.constraints.Get(); ok {
		current = NewConstrainedBox(c, current)
	}
	if w.margin != nil {
		current = NewPadding(w.margin, current)
	}
	if w.transform != nil {
		current = NewTransform(w.transform, current, TransformOpts{Alignment: w.transformAlignment})
	}
	return current
}

// decorationClipper clips to the path of a decoration.
//
// Flutter: widgets/container.dart _DecorationClipper
type decorationClipper struct {
	rendering.CustomClipperBase[*ui.Path]
	textDirection ui.TextDirection
	decoration    painting.Decoration
}

func newDecorationClipper(textDirection ui.TextDirection, decoration painting.Decoration) *decorationClipper {
	if textDirection == ui.TextDirectionUnspecified {
		textDirection = ui.TextDirectionLTR
	}
	return &decorationClipper{rendering.NewCustomClipperBase[*ui.Path](nil), textDirection, decoration}
}

func (c *decorationClipper) GetClip(size ui.Size) *ui.Path {
	return c.decoration.GetClipPath(ui.OffsetZero.And(size), c.textDirection)
}

func (c *decorationClipper) ShouldReclip(old rendering.CustomClipper[*ui.Path]) bool {
	o := old.(*decorationClipper)
	return o.decoration != c.decoration || o.textDirection != c.textDirection
}
