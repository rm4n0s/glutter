// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// Size is the size of the render box of this element, or false when the render
// object is not a box. In debug builds it panics when the size cannot be known:
// SizeOfInactiveElement, SizeDuringBuild, SizeWithoutRenderObject,
// SizeOfNonBox, SizeBeforeLayout and SizeOfDirtyLayout.
//
// Flutter: widgets/framework.dart Element.size
func (b *ElementBase) Size() (ui.Size, bool) {
	if foundation.DebugMode {
		if b.lifecycle != lifecycleActive {
			panic(errors.New("SizeOfInactiveElement",
				"cannot get the size of an inactive element; in order for an element to have a valid size, it must be active, which means it is part of the tree",
				"lifecycle", b.lifecycle.String(), "element", b.ToStringShort()))
		}
		if b.owner.debugBuilding {
			panic(errors.New("SizeDuringBuild",
				"cannot get the size during build; the size of this render object has not yet been determined because the framework is still in the process of building widgets, so the size should only be read from paint callbacks or interaction event handlers; if you need sizing information during build to decide which widgets to build, consider using a LayoutBuilder widget",
				"element", b.ToStringShort()))
		}
	}
	renderObject := b.FindRenderObject()
	if foundation.DebugMode {
		if renderObject == nil {
			panic(errors.New("SizeWithoutRenderObject",
				"cannot get the size without a render object; the element must have an associated render object, which typically means that the size was read too early in the pipeline, before the framework has created the render tree",
				"element", b.ToStringShort()))
		}
		box, ok := renderObject.(rendering.Box)
		if !ok {
			panic(errors.New("SizeOfNonBox",
				"cannot get the size from a render object that is not a RenderBox; if this type of render object has a size, call FindRenderObject and extract it manually",
				"element", b.ToStringShort(), "renderObject", foundation.RuntimeTypeName(renderObject)))
		}
		if !box.HasSize() {
			panic(errors.New("SizeBeforeLayout",
				"cannot get the size from a render object that has not been through layout, which typically means that the size was read too early in the pipeline, before the framework has determined the size and position of the render objects",
				"element", b.ToStringShort(), "renderObject", foundation.RuntimeTypeName(renderObject)))
		}
		if box.DebugNeedsLayout() {
			panic(errors.New("SizeOfDirtyLayout",
				"cannot get the size from a render object that has been marked dirty for layout; the size is ambiguous because the render object has been modified since it was last laid out, which typically means that the size was read too early in the pipeline",
				"element", b.ToStringShort(), "renderObject", foundation.RuntimeTypeName(renderObject)))
		}
	}
	if box, ok := renderObject.(rendering.Box); ok {
		return box.Size(), true
	}
	return ui.Size{}, false
}
