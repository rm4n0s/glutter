package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// alertDialogActionsLayout lays the actions of an alert dialog out: in a column, with the dividers between them;
// or, when there are two actions that fit side by side, in a row with one divider.
//
// Flutter: cupertino/dialog.dart _AlertDialogActionsLayout
type alertDialogActionsLayout struct {
	widgets.MultiChildRenderObjectWidget
	dividerThickness float64
}

func newAlertDialogActionsLayout(dividerThickness float64, children []widgets.Widget) *alertDialogActionsLayout {
	return &alertDialogActionsLayout{MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(children), dividerThickness: dividerThickness}
}

func (w *alertDialogActionsLayout) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderAlertDialogActionsLayout{dividerThickness: w.dividerThickness}
	rendering.InitRenderFlex(&r.RenderFlex, r, nil, rendering.RenderFlexOpts{Direction: painting.AxisVertical,
		MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch, TextDirection: widgets.DirectionalityOf(bc)})
	return r
}

func (w *alertDialogActionsLayout) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderAlertDialogActionsLayout)
	if r.dividerThickness != w.dividerThickness {
		r.dividerThickness = w.dividerThickness
		r.MarkNeedsLayout()
	}
	r.SetTextDirection(widgets.DirectionalityOf(bc))
}

// renderAlertDialogActionsLayout is a vertical flex that becomes a row when it has two actions that fit.
//
// Flutter: cupertino/dialog.dart _RenderAlertDialogActionsLayout
type renderAlertDialogActionsLayout struct {
	rendering.RenderFlex
	dividerThickness float64
}

func (r *renderAlertDialogActionsLayout) horizontalSlotWidthFor(overallWidth float64) float64 {
	return (overallWidth - r.dividerThickness) / 2
}

func (r *renderAlertDialogActionsLayout) forEachSlot(action func(rendering.Box)) {
	slot := r.FirstChild()
	for {
		action(slot)
		divider := r.ChildAfter(slot)
		if divider == nil {
			break
		}
		slot = r.ChildAfter(divider)
	}
}

func (r *renderAlertDialogActionsLayout) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.useHorizontalLayout(width) {
		return r.RenderFlex.ComputeMinIntrinsicHeight(width)
	}
	slotWidth := r.horizontalSlotWidthFor(width)
	height := 0.0
	r.forEachSlot(func(slot rendering.Box) { height = math.Max(height, slot.GetMinIntrinsicHeight(slotWidth)) })
	return height
}

func (r *renderAlertDialogActionsLayout) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !r.useHorizontalLayout(width) {
		return r.RenderFlex.ComputeMaxIntrinsicHeight(width)
	}
	slotWidth := r.horizontalSlotWidthFor(width)
	height := 0.0
	r.forEachSlot(func(slot rendering.Box) { height = math.Max(height, slot.GetMaxIntrinsicHeight(slotWidth)) })
	return height
}

func (r *renderAlertDialogActionsLayout) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if !r.hasValidConstraints(constraints) {
		return ui.Size{}
	}
	overallWidth := constraints.MaxWidth
	if !r.useHorizontalLayout(overallWidth) {
		return r.RenderFlex.ComputeDryLayout(constraints)
	}
	return ui.Size{Width: overallWidth, Height: r.GetMinIntrinsicHeight(overallWidth)}
}

func (r *renderAlertDialogActionsLayout) PerformLayout() {
	if r.FirstChild() == nil {
		r.SetSize(r.BoxConstraints().Smallest())
		return
	}
	if !r.hasValidConstraints(r.BoxConstraints()) {
		r.SetSize(r.BoxConstraints().Smallest())
		return
	}
	overallWidth := r.BoxConstraints().MaxWidth
	if !r.useHorizontalLayout(overallWidth) {
		r.RenderFlex.PerformLayout()
		return
	}
	slotWidth := r.horizontalSlotWidthFor(overallWidth)
	height := r.GetMinIntrinsicHeight(overallWidth)
	r.SetSize(ui.Size{Width: overallWidth, Height: height})
	ltr := r.TextDirection() == ui.TextDirectionLTR
	slot := r.FirstChild()
	x := 0.0
	if !ltr {
		x = overallWidth - slotWidth
	}
	for {
		slot.Layout(rendering.NewBoxConstraintsTight(ui.Size{Width: slotWidth, Height: height}), rendering.LayoutOpts{ParentUsesSize: true})
		slot.ParentData().(*rendering.FlexParentData).Offset = ui.Offset{Dx: x}
		if ltr {
			x += slot.Size().Width
		} else {
			x -= slot.Size().Width
		}
		divider := r.ChildAfter(slot)
		if divider == nil {
			break
		}
		divider.Layout(rendering.NewBoxConstraintsTight(ui.Size{Width: r.dividerThickness, Height: height}), rendering.LayoutOpts{})
		divider.ParentData().(*rendering.FlexParentData).Offset = ui.Offset{Dx: x}
		if ltr {
			x += r.dividerThickness
		} else {
			x -= r.dividerThickness
		}
		slot = r.ChildAfter(divider)
	}
}

// hasValidConstraints panics with CupertinoAlertDialogActionsWidthUnbounded in debug builds when the width is
// unbounded.
func (r *renderAlertDialogActionsLayout) hasValidConstraints(constraints rendering.BoxConstraints) bool {
	if foundation.DebugMode && math.IsInf(constraints.MaxWidth, 1) {
		panic(errors.New(tagCupertinoAlertDialogActionsWidthUnbounded, "The incoming width constraints are unbounded.", "constraints", constraints.String()))
	}
	return true
}

// useHorizontalLayout is whether the actions go side by side: there are two of them (and a divider), and each
// fits in a slot.
func (r *renderAlertDialogActionsLayout) useHorizontalLayout(overallWidth float64) bool {
	if r.ChildCount() != 3 {
		return false
	}
	slotWidth := r.horizontalSlotWidthFor(overallWidth)
	child := r.FirstChild()
	for {
		if child.GetMaxIntrinsicWidth(math.Inf(1)) > slotWidth {
			return false
		}
		divider := r.ChildAfter(child)
		if divider == nil {
			break
		}
		child = r.ChildAfter(divider)
	}
	return true
}

// priorityColumn is a column of two children where the top one takes what it needs and the bottom one keeps at
// least a minimum height.
//
// Flutter: cupertino/dialog.dart _PriorityColumn
type priorityColumn struct {
	widgets.MultiChildRenderObjectWidget
	bottomMinHeight float64
}

func newPriorityColumn(top, bottom widgets.Widget, bottomMinHeight float64) *priorityColumn {
	return &priorityColumn{MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase([]widgets.Widget{top, bottom}), bottomMinHeight: bottomMinHeight}
}

func (w *priorityColumn) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderPriorityColumn{bottomMinHeight: w.bottomMinHeight}
	rendering.InitRenderFlex(&r.RenderFlex, r, nil, rendering.RenderFlexOpts{Direction: painting.AxisVertical,
		MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	return r
}

func (w *priorityColumn) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderPriorityColumn)
	if r.bottomMinHeight != w.bottomMinHeight {
		r.bottomMinHeight = w.bottomMinHeight
		r.MarkNeedsLayout()
	}
}

// renderPriorityColumn lays its two children out as the priority column says.
//
// Flutter: cupertino/dialog.dart _RenderPriorityColumn
type renderPriorityColumn struct {
	rendering.RenderFlex
	bottomMinHeight float64
}

func (r *renderPriorityColumn) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.FirstChild().GetMinIntrinsicHeight(width) + r.LastChild().GetMinIntrinsicHeight(width)
}

func (r *renderPriorityColumn) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.FirstChild().GetMaxIntrinsicHeight(width) + r.LastChild().GetMaxIntrinsicHeight(width)
}

func (r *renderPriorityColumn) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	width, maxHeight := constraints.MaxWidth, constraints.MaxHeight
	top, bottom := r.childrenHeights(width, maxHeight)
	return ui.Size{Width: width, Height: top + bottom}
}

func (r *renderPriorityColumn) PerformLayout() {
	width, maxHeight := r.BoxConstraints().MaxWidth, r.BoxConstraints().MaxHeight
	top, bottom := r.childrenHeights(width, maxHeight)
	r.SetSize(ui.Size{Width: width, Height: top + bottom})
	r.FirstChild().Layout(rendering.NewBoxConstraintsTight(ui.Size{Width: width, Height: top}), rendering.LayoutOpts{ParentUsesSize: true})
	r.FirstChild().ParentData().(*rendering.FlexParentData).Offset = ui.Offset{}
	r.LastChild().Layout(rendering.NewBoxConstraintsTight(ui.Size{Width: width, Height: bottom}), rendering.LayoutOpts{ParentUsesSize: true})
	r.LastChild().ParentData().(*rendering.FlexParentData).Offset = ui.Offset{Dy: top}
}

// childrenHeights is the heights of the two children: both as they are if they fit; otherwise the bottom one
// keeps its minimum, and the top one takes the rest (or nothing).
func (r *renderPriorityColumn) childrenHeights(width, maxHeight float64) (top, bottom float64) {
	topIntrinsic := r.FirstChild().GetMinIntrinsicHeight(width)
	bottomIntrinsic := r.LastChild().GetMinIntrinsicHeight(width)
	if topIntrinsic+bottomIntrinsic <= maxHeight {
		return topIntrinsic, bottomIntrinsic
	}
	effectiveBottomMinHeight := math.Min(r.bottomMinHeight, bottomIntrinsic)
	if maxHeight-topIntrinsic >= effectiveBottomMinHeight {
		return topIntrinsic, maxHeight - topIntrinsic
	}
	if maxHeight >= effectiveBottomMinHeight {
		return maxHeight - effectiveBottomMinHeight, effectiveBottomMinHeight
	}
	return 0, maxHeight
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoAlertDialogActionsWidthUnbounded = "CupertinoAlertDialogActionsWidthUnbounded"
