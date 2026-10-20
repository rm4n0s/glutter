// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// dropdownMenuBody lays out the field of a DropdownMenu and sizes it to the
// widest of the entries, the label and the icons that come after it.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuBody
type dropdownMenuBody struct {
	widgets.MultiChildRenderObjectWidget
	width foundation.Opt[float64]
}

func newDropdownMenuBody(width foundation.Opt[float64], children []widgets.Widget) *dropdownMenuBody {
	return &dropdownMenuBody{MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(children), width: width}
}

// CreateRenderObject creates the render object.
func (w *dropdownMenuBody) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return newRenderDropdownMenuBody(w.width)
}

// UpdateRenderObject copies the width to the render object.
func (w *dropdownMenuBody) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderDropdownMenuBody).setWidth(w.width)
}

// dropdownMenuBodyParentData is the parent data of the children of the body.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuBodyParentData
type dropdownMenuBodyParentData struct {
	rendering.ContainerBoxParentData[rendering.Box]
}

// renderDropdownMenuBody is the render object of dropdownMenuBody: the first
// child is the field and is the only one that is painted and hit tested; the
// others only give the width.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody
type renderDropdownMenuBody struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	width foundation.Opt[float64]
}

func newRenderDropdownMenuBody(width foundation.Opt[float64]) *renderDropdownMenuBody {
	r := &renderDropdownMenuBody{width: width}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func (r *renderDropdownMenuBody) setWidth(value foundation.Opt[float64]) {
	if r.width == value {
		return
	}
	r.width = value
	r.MarkNeedsLayout()
}

// SetupParentData gives the child a dropdownMenuBodyParentData if it has none.
func (r *renderDropdownMenuBody) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*dropdownMenuBodyParentData); !ok {
		child.AsRenderObject().SetParentData(&dropdownMenuBodyParentData{})
	}
}

func bodyData(child rendering.Box) *dropdownMenuBodyParentData {
	return child.ParentData().(*dropdownMenuBodyParentData)
}

func (r *renderDropdownMenuBody) innerConstraints(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	intrinsicWidth := r.width.OrElse(r.GetMaxIntrinsicWidth(constraints.MaxHeight))
	widthConstraint := math.Min(intrinsicWidth, constraints.MaxWidth)
	return rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
		MaxWidth:  foundation.Some(widthConstraint),
		MaxHeight: foundation.Some(r.GetMaxIntrinsicHeight(widthConstraint)),
	})
}

// PerformLayout lays out the field, and sizes the body to the widest of the
// others.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.performLayout
func (r *renderDropdownMenuBody) PerformLayout() {
	constraints := r.BoxConstraints()
	maxWidth := 0.0
	maxHeight := foundation.None[float64]()
	inner := r.innerConstraints(constraints)
	first := r.FirstChild()
	for child := first; child != nil; {
		child.Layout(inner, rendering.LayoutOpts{ParentUsesSize: true})
		data := bodyData(child)
		if child == first {
			if maxHeight.IsNone() {
				maxHeight = foundation.Some(child.Size().Height)
			}
			child = data.NextSibling
			continue
		}
		data.Offset = ui.OffsetZero
		maxWidth = math.Max(maxWidth, child.Size().Width)
		if maxHeight.IsNone() {
			maxHeight = foundation.Some(child.Size().Height)
		}
		child = data.NextSibling
	}
	maxWidth = math.Max(dropdownMenuMinimumWidth, maxWidth)
	r.SetSize(constraints.Constrain(ui.Size{Width: r.width.OrElse(maxWidth), Height: maxHeight.Value()}))
}

// Paint paints the field.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.paint
func (r *renderDropdownMenuBody) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if child := r.FirstChild(); child != nil {
		context.PaintChild(child, bodyData(child).Offset.Add(offset))
	}
}

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.computeDryLayout
func (r *renderDropdownMenuBody) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	maxWidth := 0.0
	maxHeight := foundation.None[float64]()
	inner := r.innerConstraints(constraints)
	first := r.FirstChild()
	for child := first; child != nil; child = bodyData(child).NextSibling {
		childSize := child.GetDryLayout(inner)
		if child != first {
			maxWidth = math.Max(maxWidth, childSize.Width)
		}
		if maxHeight.IsNone() {
			maxHeight = foundation.Some(childSize.Height)
		}
	}
	maxWidth = math.Max(dropdownMenuMinimumWidth, maxWidth)
	return constraints.Constrain(ui.Size{Width: r.width.OrElse(maxWidth), Height: maxHeight.Value()})
}

// intrinsicWidth is the widest of the children but the field, with the last two
// added to it.
func (r *renderDropdownMenuBody) intrinsicWidth(of func(rendering.Box) float64) float64 {
	width := 0.0
	first, last := r.FirstChild(), r.LastChild()
	before := rendering.Box(nil)
	if last != nil {
		before = r.ChildBefore(last)
	}
	for child := first; child != nil; child = bodyData(child).NextSibling {
		if child == first {
			continue
		}
		w := of(child)
		if child == last {
			width += w
		}
		if child == before {
			width += w
		}
		width = math.Max(width, w)
	}
	return math.Max(width, dropdownMenuMinimumWidth)
}

// ComputeMinIntrinsicWidth is the minimum width of the children but the field.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.computeMinIntrinsicWidth
func (r *renderDropdownMenuBody) ComputeMinIntrinsicWidth(height float64) float64 {
	return r.intrinsicWidth(func(c rendering.Box) float64 { return c.GetMinIntrinsicWidth(height) })
}

// ComputeMaxIntrinsicWidth is the maximum width of the children but the field.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.computeMaxIntrinsicWidth
func (r *renderDropdownMenuBody) ComputeMaxIntrinsicWidth(height float64) float64 {
	return r.intrinsicWidth(func(c rendering.Box) float64 { return c.GetMaxIntrinsicWidth(height) })
}

// ComputeMinIntrinsicHeight is the minimum height of the field.
//
// Deviation: none; Flutter's code shadows the width parameter with a local that
// is zero, so the field is asked for its height at the width zero.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.computeMinIntrinsicHeight
func (r *renderDropdownMenuBody) ComputeMinIntrinsicHeight(float64) float64 {
	if child := r.FirstChild(); child != nil {
		return math.Max(0, child.GetMinIntrinsicHeight(0))
	}
	return 0
}

// ComputeMaxIntrinsicHeight is the maximum height of the field.
//
// Deviation: none; see ComputeMinIntrinsicHeight.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.computeMaxIntrinsicHeight
func (r *renderDropdownMenuBody) ComputeMaxIntrinsicHeight(float64) float64 {
	if child := r.FirstChild(); child != nil {
		return math.Max(0, child.GetMaxIntrinsicHeight(0))
	}
	return 0
}

// HitTestChildren hit tests the field.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.hitTestChildren
func (r *renderDropdownMenuBody) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	child := r.FirstChild()
	if child == nil {
		return false
	}
	offset := bodyData(child).Offset
	return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
		return child.HitTest(result, transformed)
	})
}

// VisitChildrenForSemantics visits the field only.
//
// Flutter: material/dropdown_menu.dart _RenderDropdownMenuBody.visitChildrenForSemantics
func (r *renderDropdownMenuBody) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	if child := r.FirstChild(); child != nil {
		visitor(child)
	}
}
