// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// AnnotatedRegionOpts are the optional parameters of NewAnnotatedRegion.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.new
type AnnotatedRegionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// NotSized says that the region is not limited to the size of the child but
	// has no bounds (Flutter's sized: false).
	NotSized bool
}

// AnnotatedRegion marks a region of the layer tree with a value, which an
// ancestor finds with a layer search (a Scene's AnnotatedRegionLayer). AppBar
// uses it for its system overlay style.
//
// Deviation: the value is an any, not a type parameter (SPEC §12.10).
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion
type AnnotatedRegion struct {
	SingleChildRenderObjectWidget
	value any
	sized bool
}

// NewAnnotatedRegion creates the widget.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.new
func NewAnnotatedRegion(value any, child Widget, opts ...AnnotatedRegionOpts) *AnnotatedRegion {
	o := firstOpts(opts)
	return &AnnotatedRegion{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), value, !o.NotSized}
}

// Value is the value that is provided for the region.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.value
func (w *AnnotatedRegion) Value() any { return w.value }

// Sized is whether the region is limited to the size of the child.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.sized
func (w *AnnotatedRegion) Sized() bool { return w.sized }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.createRenderObject
func (w *AnnotatedRegion) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderAnnotatedRegion(w.value, w.sized, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/annotated_region.dart AnnotatedRegion.updateRenderObject
func (w *AnnotatedRegion) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderAnnotatedRegion)
	r.SetValue(w.value)
	r.SetSized(w.sized)
}
