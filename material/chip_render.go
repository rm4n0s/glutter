// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// chipRedirectingHitDetection widget constrains its child to at least the size of
// the tap target, and sends the taps in the padding to the chip.
//
// Flutter: material/chip.dart _ChipRedirectingHitDetectionWidget
type chipRedirectingHitDetection struct {
	widgets.SingleChildRenderObjectWidget
	constraints rendering.BoxConstraints
}

func newChipRedirectingHitDetection(constraints rendering.BoxConstraints, child widgets.Widget) *chipRedirectingHitDetection {
	return &chipRedirectingHitDetection{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), constraints: constraints}
}

func (w *chipRedirectingHitDetection) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderChipRedirectingHitDetection{}
	rendering.InitRenderConstrainedBox(&r.RenderConstrainedBox, r, w.constraints, nil)
	return r
}

func (w *chipRedirectingHitDetection) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	ro.(*renderChipRedirectingHitDetection).SetAdditionalConstraints(w.constraints)
}

// renderChipRedirectingHitDetection is a constrained box that hits its child at
// the vertical middle of the position.
//
// Flutter: material/chip.dart _RenderChipRedirectingHitDetection
type renderChipRedirectingHitDetection struct {
	rendering.RenderConstrainedBox
}

// HitTest hits the child at the position in its middle row, so that the taps above
// and below the chip, in the padding of the tap target, reach it.
//
// Flutter: material/chip.dart _RenderChipRedirectingHitDetection.hitTest
func (r *renderChipRedirectingHitDetection) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if !r.Size().Contains(position) {
		return false
	}
	offset := ui.Offset{Dx: position.Dx, Dy: r.Size().Height / 2}
	child := r.Child()
	return result.AddWithRawTransform(ui.NewMatrix4ForceToPoint(offset), position, func(result *rendering.BoxHitTestResult, _ ui.Offset) bool {
		return child.HitTest(result, offset)
	})
}

// chipSlot identifies the children of a renderChip.
//
// Flutter: material/chip.dart _ChipSlot
type chipSlot int

const (
	chipSlotLabel chipSlot = iota
	chipSlotAvatar
	chipSlotDeleteIcon
)

var chipSlotNames = [...]string{"label", "avatar", "deleteIcon"}

func (s chipSlot) String() string { return chipSlotNames[s] }

var allChipSlots = []any{chipSlotLabel, chipSlotAvatar, chipSlotDeleteIcon}

// chipRenderTheme are the parts and the settings of a chip.
//
// Flutter: material/chip.dart _ChipRenderTheme
type chipRenderTheme struct {
	avatar         widgets.Widget
	label          widgets.Widget
	deleteIcon     widgets.Widget
	brightness     ui.Brightness
	padding        painting.EdgeInsets
	visualDensity  VisualDensity
	labelPadding   painting.EdgeInsets
	showAvatar     bool
	showCheckmark  bool
	checkmarkColor foundation.Opt[ui.Color]
	canTapBody     bool
}

// equal is Flutter's ==, which leaves the visual density out.
func (t chipRenderTheme) equal(o chipRenderTheme) bool {
	return t.avatar == o.avatar && t.label == o.label && t.deleteIcon == o.deleteIcon && t.brightness == o.brightness &&
		t.padding == o.padding && t.labelPadding == o.labelPadding && t.showAvatar == o.showAvatar &&
		t.showCheckmark == o.showCheckmark && t.checkmarkColor == o.checkmarkColor && t.canTapBody == o.canTapBody
}

// chipRenderWidget is the slotted widget that creates the render object of a chip.
//
// Flutter: material/chip.dart _ChipRenderWidget
type chipRenderWidget struct {
	widgets.SlottedMultiChildRenderObjectWidget
	theme                    chipRenderTheme
	value                    bool
	isEnabled                bool
	checkmarkAnimation       animation.Animation[float64]
	avatarDrawerAnimation    animation.Animation[float64]
	deleteDrawerAnimation    animation.Animation[float64]
	enableAnimation          animation.Animation[float64]
	avatarBorder             painting.ShapeBorder
	avatarBoxConstraints     foundation.Opt[rendering.BoxConstraints]
	deleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]
}

// Slots are the slots of the chip.
//
// Flutter: material/chip.dart _ChipRenderWidget.slots
func (w *chipRenderWidget) Slots() []any { return allChipSlots }

// ChildForSlot is the part of the chip in the slot.
//
// Flutter: material/chip.dart _ChipRenderWidget.childForSlot
func (w *chipRenderWidget) ChildForSlot(slot any) widgets.Widget {
	switch slot.(chipSlot) {
	case chipSlotLabel:
		return w.theme.label
	case chipSlotAvatar:
		return w.theme.avatar
	case chipSlotDeleteIcon:
		return w.theme.deleteIcon
	}
	return nil
}

func (w *chipRenderWidget) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderChip{
		theme: w.theme, textDirection: widgets.DirectionalityOf(bc), value: w.value, isEnabled: w.isEnabled,
		checkmarkAnimation: w.checkmarkAnimation, avatarDrawerAnimation: w.avatarDrawerAnimation,
		deleteDrawerAnimation: w.deleteDrawerAnimation, enableAnimation: w.enableAnimation, avatarBorder: w.avatarBorder,
		avatarBoxConstraints: w.avatarBoxConstraints, deleteIconBoxConstraints: w.deleteIconBoxConstraints,
		avatarOpacityLayer:     rendering.NewLayerHandle[*rendering.OpacityLayer](nil),
		labelOpacityLayer:      rendering.NewLayerHandle[*rendering.OpacityLayer](nil),
		deleteIconOpacityLayer: rendering.NewLayerHandle[*rendering.OpacityLayer](nil),
	}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitSlottedContainer(&r.SlottedContainer, r)
	return r
}

func (w *chipRenderWidget) UpdateRenderObject(bc widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderChip)
	r.setTheme(w.theme)
	r.setTextDirection(widgets.DirectionalityOf(bc))
	r.value = w.value
	r.isEnabled = w.isEnabled
	r.checkmarkAnimation = w.checkmarkAnimation
	r.avatarDrawerAnimation = w.avatarDrawerAnimation
	r.deleteDrawerAnimation = w.deleteDrawerAnimation
	r.enableAnimation = w.enableAnimation
	r.avatarBorder = w.avatarBorder
	r.setAvatarBoxConstraints(w.avatarBoxConstraints)
	r.setDeleteIconBoxConstraints(w.deleteIconBoxConstraints)
}

// chipSizes are the sizes that the layout of a chip works out.
//
// Flutter: material/chip.dart _ChipSizes
type chipSizes struct {
	size              ui.Size
	overall           ui.Size
	content           float64
	avatar            ui.Size
	labelConstraints  rendering.BoxConstraints
	label             ui.Size
	deleteIcon        ui.Size
	densityAdjustment ui.Offset
}

// renderChip lays out and paints the avatar, the label and the delete icon of a
// chip.
//
// Flutter: material/chip.dart _RenderChip
type renderChip struct {
	rendering.RenderBox
	rendering.SlottedContainer

	value                 bool
	isEnabled             bool
	deleteButtonRect      ui.Rect
	pressRect             ui.Rect
	checkmarkAnimation    animation.Animation[float64]
	avatarDrawerAnimation animation.Animation[float64]
	deleteDrawerAnimation animation.Animation[float64]
	enableAnimation       animation.Animation[float64]
	avatarBorder          painting.ShapeBorder

	theme                    chipRenderTheme
	textDirection            ui.TextDirection
	avatarBoxConstraints     foundation.Opt[rendering.BoxConstraints]
	deleteIconBoxConstraints foundation.Opt[rendering.BoxConstraints]

	avatarOpacityLayer     *rendering.LayerHandle[*rendering.OpacityLayer]
	labelOpacityLayer      *rendering.LayerHandle[*rendering.OpacityLayer]
	deleteIconOpacityLayer *rendering.LayerHandle[*rendering.OpacityLayer]

	handles [4]foundation.ListenerHandle
}

func (r *renderChip) slotBox(s chipSlot) rendering.Box {
	c := r.SlotChild(s)
	if c == nil {
		return nil
	}
	return c.(rendering.Box)
}

func (r *renderChip) avatar() rendering.Box     { return r.slotBox(chipSlotAvatar) }
func (r *renderChip) deleteIcon() rendering.Box { return r.slotBox(chipSlotDeleteIcon) }
func (r *renderChip) label() rendering.Box      { return r.slotBox(chipSlotLabel) }

func (r *renderChip) setTheme(v chipRenderTheme) {
	if r.theme.equal(v) {
		r.theme = v
		return
	}
	r.theme = v
	r.MarkNeedsLayout()
}

func (r *renderChip) setTextDirection(v ui.TextDirection) {
	if r.textDirection == v {
		return
	}
	r.textDirection = v
	r.MarkNeedsLayout()
}

func (r *renderChip) setAvatarBoxConstraints(v foundation.Opt[rendering.BoxConstraints]) {
	if r.avatarBoxConstraints == v {
		return
	}
	r.avatarBoxConstraints = v
	r.MarkNeedsLayout()
}

func (r *renderChip) setDeleteIconBoxConstraints(v foundation.Opt[rendering.BoxConstraints]) {
	if r.deleteIconBoxConstraints == v {
		return
	}
	r.deleteIconBoxConstraints = v
	r.MarkNeedsLayout()
}

func (r *renderChip) isDrawingCheckmark() bool {
	return r.theme.showCheckmark && r.checkmarkAnimation.Status() != animation.AnimationStatusDismissed
}

func (r *renderChip) deleteIconShowing() bool {
	return r.deleteDrawerAnimation.Status() != animation.AnimationStatusDismissed
}

func chipBoxRect(box rendering.Box) ui.Rect {
	return boxData(box).Offset.And(box.Size())
}

// ComputeMinIntrinsicWidth is the sum of the parts and the paddings.
//
// Flutter: material/chip.dart _RenderChip.computeMinIntrinsicWidth
func (r *renderChip) ComputeMinIntrinsicWidth(height float64) float64 {
	overallPadding := r.theme.padding.Horizontal() + r.theme.labelPadding.Horizontal()
	return overallPadding + r.avatar().GetMinIntrinsicWidth(height) + r.label().GetMinIntrinsicWidth(height) + r.deleteIcon().GetMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is the sum of the parts and the paddings.
//
// Flutter: material/chip.dart _RenderChip.computeMaxIntrinsicWidth
func (r *renderChip) ComputeMaxIntrinsicWidth(height float64) float64 {
	overallPadding := r.theme.padding.Horizontal() + r.theme.labelPadding.Horizontal()
	return overallPadding + r.avatar().GetMaxIntrinsicWidth(height) + r.label().GetMaxIntrinsicWidth(height) + r.deleteIcon().GetMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is the chip height, or more for a tall label.
//
// Flutter: material/chip.dart _RenderChip.computeMinIntrinsicHeight
func (r *renderChip) ComputeMinIntrinsicHeight(width float64) float64 {
	return math.Max(chipHeight, r.theme.padding.Vertical()+r.theme.labelPadding.Vertical()+r.label().GetMinIntrinsicHeight(width))
}

// ComputeMaxIntrinsicHeight is the minimum one.
//
// Flutter: material/chip.dart _RenderChip.computeMaxIntrinsicHeight
func (r *renderChip) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.GetMinIntrinsicHeight(width)
}

// ComputeDistanceToActualBaseline is the baseline of the label.
//
// Flutter: material/chip.dart _RenderChip.computeDistanceToActualBaseline
func (r *renderChip) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	label := r.label()
	b, ok := label.GetDistanceToActualBaseline(baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(b + boxData(label).Offset.Dy)
}

func (r *renderChip) labelConstraintsFrom(contentConstraints rendering.BoxConstraints, iconWidth, contentSize float64, rawLabelSize ui.Size) rendering.BoxConstraints {
	freeSpace := contentConstraints.MaxWidth - iconWidth - r.theme.labelPadding.Horizontal() - r.theme.padding.Horizontal()
	maxLabelWidth := math.Max(0.0, freeSpace)
	maxWidth := rawLabelSize.Width
	if !math.IsInf(maxLabelWidth, 0) && !math.IsNaN(maxLabelWidth) {
		maxWidth = maxLabelWidth
	}
	return rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
		MinHeight: foundation.Some(rawLabelSize.Height), MaxHeight: foundation.Some(contentSize), MaxWidth: foundation.Some(maxWidth)})
}

func (r *renderChip) layoutAvatar(contentSize float64, layoutChild rendering.ChildLayouter) ui.Size {
	avatarConstraints := r.avatarBoxConstraints.OrElse(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
		Width: foundation.Some(contentSize), Height: foundation.Some(contentSize)}))
	avatarBoxSize := layoutChild(r.avatar(), avatarConstraints)
	if !r.theme.showCheckmark && !r.theme.showAvatar {
		return ui.Size{Width: 0.0, Height: contentSize}
	}
	avatarFullWidth := contentSize
	if r.theme.showAvatar {
		avatarFullWidth = avatarBoxSize.Width
	}
	return ui.Size{Width: avatarFullWidth * r.avatarDrawerAnimation.Value(), Height: avatarBoxSize.Height}
}

func (r *renderChip) layoutDeleteIcon(contentSize float64, layoutChild rendering.ChildLayouter) ui.Size {
	deleteIconConstraints := r.deleteIconBoxConstraints.OrElse(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
		Width: foundation.Some(contentSize), Height: foundation.Some(contentSize)}))
	boxSize := layoutChild(r.deleteIcon(), deleteIconConstraints)
	if !r.deleteIconShowing() {
		return ui.Size{Width: 0.0, Height: contentSize}
	}
	return ui.Size{Width: r.deleteDrawerAnimation.Value() * boxSize.Width, Height: boxSize.Height}
}

// HitTest sends the hits to the delete icon or to the label.
//
// Flutter: material/chip.dart _RenderChip.hitTest
func (r *renderChip) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if !r.Size().Contains(position) {
		return false
	}
	hitIsOnDeleteIcon := chipHitIsOnDeleteIcon(r.theme.padding, r.theme.labelPadding, position, r.Size(), r.deleteIcon().Size(), r.textDirection)
	hitTestChild := r.label()
	if hitIsOnDeleteIcon {
		hitTestChild = r.deleteIcon()
	}
	center := hitTestChild.Size().Center(ui.Offset{})
	return result.AddWithRawTransform(ui.NewMatrix4ForceToPoint(center), position, func(result *rendering.BoxHitTestResult, _ ui.Offset) bool {
		return hitTestChild.HitTest(result, center)
	})
}

// ComputeDryLayout is the size of the chip for the constraints.
//
// Flutter: material/chip.dart _RenderChip.computeDryLayout
func (r *renderChip) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeSizes(constraints, rendering.ChildLayoutHelper.DryLayoutChild).size
}

// ComputeDryBaseline is the baseline of the label for the constraints.
//
// Flutter: material/chip.dart _RenderChip.computeDryBaseline
func (r *renderChip) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	sizes := r.computeSizes(constraints, rendering.ChildLayoutHelper.DryLayoutChild)
	b, ok := r.label().GetDryBaseline(sizes.labelConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(b + (sizes.content-sizes.label.Height+sizes.densityAdjustment.Dy)/2 + r.theme.padding.Top + r.theme.labelPadding.Top)
}

func (r *renderChip) computeSizes(constraints rendering.BoxConstraints, layoutChild rendering.ChildLayouter) chipSizes {
	contentConstraints := constraints.Loosen()
	rawLabelSize := r.label().GetDryLayout(contentConstraints)
	contentSize := math.Max(chipHeight-r.theme.padding.Vertical()+r.theme.labelPadding.Vertical(), rawLabelSize.Height+r.theme.labelPadding.Vertical())
	avatarSize := r.layoutAvatar(contentSize, layoutChild)
	deleteIconSize := r.layoutDeleteIcon(contentSize, layoutChild)
	labelConstraints := r.labelConstraintsFrom(contentConstraints, avatarSize.Width+deleteIconSize.Width, contentSize, rawLabelSize)
	labelSize := r.theme.labelPadding.InflateSize(layoutChild(r.label(), labelConstraints))
	densityAdjustment := ui.Offset{Dx: 0.0, Dy: r.theme.visualDensity.BaseSizeAdjustment().Dy / 2.0}
	overallSize := ui.Size{Width: avatarSize.Width + labelSize.Width + deleteIconSize.Width + densityAdjustment.Dx, Height: contentSize + densityAdjustment.Dy}
	paddedSize := ui.Size{Width: overallSize.Width + r.theme.padding.Horizontal(), Height: overallSize.Height + r.theme.padding.Vertical()}
	return chipSizes{
		size: constraints.Constrain(paddedSize), overall: overallSize, content: contentSize, densityAdjustment: densityAdjustment,
		avatar: avatarSize, labelConstraints: labelConstraints, label: labelSize, deleteIcon: deleteIconSize,
	}
}

// PerformLayout lays out the parts of the chip and works out the rectangles of the
// delete button and of the body.
//
// Flutter: material/chip.dart _RenderChip.performLayout
func (r *renderChip) PerformLayout() {
	constraints := r.BoxConstraints()
	sizes := r.computeSizes(constraints, rendering.ChildLayoutHelper.LayoutChild)
	padding := r.theme.padding
	const left = 0.0
	right := sizes.overall.Width

	centerLayout := func(boxSize ui.Size, x float64) ui.Offset {
		if r.textDirection == ui.TextDirectionRTL {
			x -= boxSize.Width
		}
		return ui.Offset{Dx: x, Dy: (sizes.content - boxSize.Height + sizes.densityAdjustment.Dy) / 2.0}
	}

	var avatarOffset, labelOffset, deleteIconOffset ui.Offset
	avatar, label, deleteIcon := r.avatar(), r.label(), r.deleteIcon()
	if r.textDirection == ui.TextDirectionRTL {
		start := right
		if r.theme.showCheckmark || r.theme.showAvatar {
			avatarOffset = centerLayout(sizes.avatar, start)
			start -= sizes.avatar.Width
		}
		labelOffset = centerLayout(sizes.label, start)
		start -= sizes.label.Width
		if r.deleteIconShowing() {
			r.deleteButtonRect = ui.NewRectFromLTWH(0.0, 0.0, sizes.deleteIcon.Width+padding.Right, sizes.overall.Height+padding.Vertical())
			deleteIconOffset = centerLayout(sizes.deleteIcon, start)
		} else {
			r.deleteButtonRect = ui.Rect{}
		}
		if r.theme.canTapBody {
			r.pressRect = ui.NewRectFromLTWH(r.deleteButtonRect.Width(), 0.0,
				sizes.overall.Width-r.deleteButtonRect.Width()+padding.Horizontal(), sizes.overall.Height+padding.Vertical())
		} else {
			r.pressRect = ui.Rect{}
		}
	} else {
		start := left
		if r.theme.showCheckmark || r.theme.showAvatar {
			avatarOffset = centerLayout(sizes.avatar, start-avatar.Size().Width+sizes.avatar.Width)
			start += sizes.avatar.Width
		}
		labelOffset = centerLayout(sizes.label, start)
		start += sizes.label.Width
		if r.theme.canTapBody {
			width := sizes.overall.Width + padding.Horizontal()
			if r.deleteIconShowing() {
				width = start + padding.Left
			}
			r.pressRect = ui.NewRectFromLTWH(0.0, 0.0, width, sizes.overall.Height+padding.Vertical())
		} else {
			r.pressRect = ui.Rect{}
		}
		start -= deleteIcon.Size().Width - sizes.deleteIcon.Width
		if r.deleteIconShowing() {
			deleteIconOffset = centerLayout(sizes.deleteIcon, start)
			r.deleteButtonRect = ui.NewRectFromLTWH(start+padding.Left, 0.0, sizes.deleteIcon.Width+padding.Right, sizes.overall.Height+padding.Vertical())
		} else {
			r.deleteButtonRect = ui.Rect{}
		}
	}
	labelOffset = labelOffset.Add(ui.Offset{Dx: 0.0, Dy: ((sizes.label.Height - r.theme.labelPadding.Vertical()) - label.Size().Height) / 2.0})
	boxData(avatar).Offset = padding.TopLeft().Add(avatarOffset)
	boxData(label).Offset = padding.TopLeft().Add(labelOffset).Add(r.theme.labelPadding.TopLeft())
	boxData(deleteIcon).Offset = padding.TopLeft().Add(deleteIconOffset)
	paddedSize := ui.Size{Width: sizes.overall.Width + padding.Horizontal(), Height: sizes.overall.Height + padding.Vertical()}
	r.SetSize(constraints.Constrain(paddedSize))
}

// disabledColor is the color that fades the content of a disabled chip.
//
// Flutter: material/chip.dart _RenderChip._disabledColor
func (r *renderChip) disabledColor() ui.Color {
	if r.enableAnimation.Status() == animation.AnimationStatusCompleted {
		return Colors.White
	}
	color := Colors.White
	if r.theme.brightness == ui.BrightnessDark {
		color = Colors.Black
	}
	return animation.NewColorTween(foundation.Some(color.WithAlpha(chipDisabledAlpha)), foundation.Some(color)).Evaluate(r.enableAnimation).Value()
}

// Flutter: material/chip.dart _RenderChip._paintCheck
func (r *renderChip) paintCheck(canvas *ui.Canvas, origin ui.Offset, size float64) {
	var paintColor ui.Color
	if c, ok := r.theme.checkmarkColor.Get(); ok {
		paintColor = c
	} else {
		switch {
		case r.theme.brightness != ui.BrightnessDark && r.theme.showAvatar:
			paintColor = Colors.White
		case r.theme.brightness != ui.BrightnessDark:
			paintColor = Colors.Black.WithAlpha(chipCheckmarkAlpha)
		case r.theme.showAvatar:
			paintColor = Colors.Black
		default:
			paintColor = Colors.White.WithAlpha(chipCheckmarkAlpha)
		}
	}
	reversing := r.checkmarkAnimation.Status() == animation.AnimationStatusReverse
	if reversing {
		paintColor = animation.NewColorTween(foundation.Some(Colors.Transparent), foundation.Some(paintColor)).Evaluate(r.checkmarkAnimation).Value()
	}
	paint := ui.NewPaint()
	paint.SetColor(paintColor)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(chipCheckmarkStrokeWidth * r.avatar().Size().Height / 24.0)
	t := r.checkmarkAnimation.Value()
	if reversing {
		t = 1.0
	}
	if t == 0.0 {
		return
	}
	path := ui.NewPath()
	start := ui.Offset{Dx: size * 0.15, Dy: size * 0.45}
	mid := ui.Offset{Dx: size * 0.4, Dy: size * 0.7}
	end := ui.Offset{Dx: size * 0.85, Dy: size * 0.25}
	lerp := func(a, b ui.Offset, t float64) ui.Offset { return *ui.OffsetLerp(&a, &b, t) }
	if t < 0.5 {
		drawMid := lerp(start, mid, t*2.0)
		path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
		path.LineTo(origin.Dx+drawMid.Dx, origin.Dy+drawMid.Dy)
	} else {
		drawEnd := lerp(mid, end, (t-0.5)*2.0)
		path.MoveTo(origin.Dx+start.Dx, origin.Dy+start.Dy)
		path.LineTo(origin.Dx+mid.Dx, origin.Dy+mid.Dy)
		path.LineTo(origin.Dx+drawEnd.Dx, origin.Dy+drawEnd.Dy)
	}
	canvas.DrawPath(path, paint)
}

// Flutter: material/chip.dart _RenderChip._paintSelectionOverlay
func (r *renderChip) paintSelectionOverlay(context *rendering.PaintingContext, offset ui.Offset) {
	if !r.isDrawingCheckmark() {
		return
	}
	avatar := r.avatar()
	if r.theme.showAvatar {
		avatarRect := chipBoxRect(avatar).Shift(offset)
		scrim := animation.NewColorTween(foundation.Some(Colors.Transparent), foundation.Some(ui.NewColor(chipSelectScrimColor))).Evaluate(r.checkmarkAnimation)
		darkenPaint := ui.NewPaint()
		darkenPaint.SetColor(scrim.Value())
		darkenPaint.SetBlendMode(ui.BlendModeSrcATop)
		if r.avatarBorder.PreferPaintInterior() {
			r.avatarBorder.PaintInterior(context.Canvas(), avatarRect, darkenPaint, r.textDirection)
		} else {
			context.Canvas().DrawPath(r.avatarBorder.GetOuterPath(avatarRect, r.textDirection), darkenPaint)
		}
	}
	checkSize := avatar.Size().Height * 0.75
	checkOffset := boxData(avatar).Offset.Add(ui.Offset{Dx: avatar.Size().Height * 0.125, Dy: avatar.Size().Height * 0.125})
	r.paintCheck(context.Canvas(), offset.Add(checkOffset), checkSize)
}

// Flutter: material/chip.dart _RenderChip._paintAvatar
func (r *renderChip) paintAvatar(context *rendering.PaintingContext, offset ui.Offset) {
	avatar := r.avatar()
	paintWithOverlay := func(context *rendering.PaintingContext, offset ui.Offset) {
		context.PaintChild(avatar, boxData(avatar).Offset.Add(offset))
		r.paintSelectionOverlay(context, offset)
	}
	if !r.theme.showAvatar && r.avatarDrawerAnimation.Status() == animation.AnimationStatusDismissed {
		r.avatarOpacityLayer.SetLayer(nil)
		return
	}
	disabledColor := r.disabledColor()
	disabledColorAlpha := disabledColor.Alpha()
	if r.NeedsCompositing() {
		r.avatarOpacityLayer.SetLayer(context.PushOpacity(offset, disabledColorAlpha, paintWithOverlay, r.avatarOpacityLayer.Layer()))
		return
	}
	r.avatarOpacityLayer.SetLayer(nil)
	if disabledColorAlpha != 0xff {
		rect := chipBoxRect(avatar).Shift(offset).Inflate(20.0)
		layerPaint := ui.NewPaint()
		layerPaint.SetColor(disabledColor)
		context.Canvas().SaveLayer(&rect, layerPaint)
	}
	paintWithOverlay(context, offset)
	if disabledColorAlpha != 0xff {
		context.Canvas().Restore()
	}
}

// Flutter: material/chip.dart _RenderChip._paintChild
func (r *renderChip) paintChild(context *rendering.PaintingContext, offset ui.Offset, child rendering.Box, isDeleteIcon bool) {
	if child == nil {
		r.labelOpacityLayer.SetLayer(nil)
		r.deleteIconOpacityLayer.SetLayer(nil)
		return
	}
	paintIt := func(context *rendering.PaintingContext, offset ui.Offset) {
		context.PaintChild(child, boxData(child).Offset.Add(offset))
	}
	if r.enableAnimation.Status() == animation.AnimationStatusCompleted {
		paintIt(context, offset)
		return
	}
	disabledColor := r.disabledColor()
	disabledColorAlpha := disabledColor.Alpha()
	if r.NeedsCompositing() {
		r.labelOpacityLayer.SetLayer(context.PushOpacity(offset, disabledColorAlpha, paintIt, r.labelOpacityLayer.Layer()))
		if isDeleteIcon {
			r.deleteIconOpacityLayer.SetLayer(context.PushOpacity(offset, disabledColorAlpha, paintIt, r.deleteIconOpacityLayer.Layer()))
		}
		return
	}
	r.labelOpacityLayer.SetLayer(nil)
	r.deleteIconOpacityLayer.SetLayer(nil)
	childRect := chipBoxRect(child).Shift(offset).Inflate(20.0)
	layerPaint := ui.NewPaint()
	layerPaint.SetColor(disabledColor)
	context.Canvas().SaveLayer(&childRect, layerPaint)
	paintIt(context, offset)
	context.Canvas().Restore()
}

// Attach listens to the animations.
//
// Flutter: material/chip.dart _RenderChip.attach
func (r *renderChip) Attach(owner *rendering.PipelineOwner) {
	r.RenderBox.Attach(owner)
	r.handles[0] = r.checkmarkAnimation.AddListener(r.MarkNeedsPaint)
	r.handles[1] = r.avatarDrawerAnimation.AddListener(r.MarkNeedsLayout)
	r.handles[2] = r.deleteDrawerAnimation.AddListener(r.MarkNeedsLayout)
	r.handles[3] = r.enableAnimation.AddListener(r.MarkNeedsPaint)
}

// Detach stops listening to the animations.
//
// Flutter: material/chip.dart _RenderChip.detach
func (r *renderChip) Detach() {
	r.checkmarkAnimation.RemoveListener(r.handles[0])
	r.avatarDrawerAnimation.RemoveListener(r.handles[1])
	r.deleteDrawerAnimation.RemoveListener(r.handles[2])
	r.enableAnimation.RemoveListener(r.handles[3])
	r.RenderBox.Detach()
}

// Dispose releases the layers.
//
// Flutter: material/chip.dart _RenderChip.dispose
func (r *renderChip) Dispose() {
	r.labelOpacityLayer.SetLayer(nil)
	r.deleteIconOpacityLayer.SetLayer(nil)
	r.avatarOpacityLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

// Paint paints the avatar, the delete icon and the label.
//
// Flutter: material/chip.dart _RenderChip.paint
func (r *renderChip) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	r.paintAvatar(context, offset)
	if r.deleteIconShowing() {
		r.paintChild(context, offset, r.deleteIcon(), true)
	}
	r.paintChild(context, offset, r.label(), false)
}

// HitTestSelf is true inside the delete button and the body.
//
// Flutter: material/chip.dart _RenderChip.hitTestSelf
func (r *renderChip) HitTestSelf(position ui.Offset) bool {
	return r.deleteButtonRect.Contains(position) || r.pressRect.Contains(position)
}

// chipHitIsOnDeleteIcon says whether a tap at the position is meant for the delete
// icon: the touch target of the icon is larger than the icon.
//
// Flutter: material/chip.dart _hitIsOnDeleteIcon
func chipHitIsOnDeleteIcon(padding, labelPadding painting.EdgeInsets, tapPosition ui.Offset, chipSize, deleteButtonSize ui.Size, textDirection ui.TextDirection) bool {
	deflatedSize := padding.DeflateSize(chipSize)
	adjustedPosition := tapPosition.Sub(ui.Offset{Dx: padding.Left, Dy: padding.Top})
	accessibleDeleteButtonWidth := math.Min(deflatedSize.Width*0.499, math.Min(labelPadding.Right+deleteButtonSize.Width, 24.0+deleteButtonSize.Width/2.0))
	if textDirection == ui.TextDirectionRTL {
		return adjustedPosition.Dx <= accessibleDeleteButtonWidth
	}
	return adjustedPosition.Dx >= deflatedSize.Width-accessibleDeleteButtonWidth
}

// ensureMinSemanticsSize gives its child the semantics bounds of at least a size.
//
// Flutter: material/chip.dart _EnsureMinSemanticsSize
type ensureMinSemanticsSize struct {
	widgets.SingleChildRenderObjectWidget
	semanticSize ui.Size
}

func newEnsureMinSemanticsSize(semanticSize ui.Size, child widgets.Widget) *ensureMinSemanticsSize {
	return &ensureMinSemanticsSize{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), semanticSize: semanticSize}
}

func (w *ensureMinSemanticsSize) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderEnsureMinSemanticsSize{semanticSize: w.semanticSize}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *ensureMinSemanticsSize) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderEnsureMinSemanticsSize)
	if r.semanticSize == w.semanticSize {
		return
	}
	r.semanticSize = w.semanticSize
	r.MarkNeedsSemanticsUpdate()
}

// renderEnsureMinSemanticsSize is a proxy box that widens its semantic bounds.
//
// Flutter: material/chip.dart _RenderEnsureMinSemanticsSize
type renderEnsureMinSemanticsSize struct {
	rendering.RenderProxyBox
	semanticSize ui.Size
}

// DescribeSemanticsConfiguration makes the box a semantic boundary and a button.
//
// Flutter: material/chip.dart _RenderEnsureMinSemanticsSize.describeSemanticsConfiguration
func (r *renderEnsureMinSemanticsSize) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.SetIsButton(true)
}

// SemanticBounds are the bounds around the center, at least the semantic size.
//
// Flutter: material/chip.dart _RenderEnsureMinSemanticsSize.semanticBounds
func (r *renderEnsureMinSemanticsSize) SemanticBounds() ui.Rect {
	return ui.NewRectFromCenter(r.PaintBounds().Center(), math.Max(r.semanticSize.Width, r.Size().Width), math.Max(r.semanticSize.Height, r.Size().Height))
}
