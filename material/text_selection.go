// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// materialHandleSize is the size of a selection handle.
//
// Flutter: material/text_selection.dart _kHandleSize
const materialHandleSize = 22.0

// MaterialTextSelectionControls builds the selection handles of the Material
// design: a drop that points at the text. The context menu is not a part of the
// controls (it is the contextMenuBuilder of the text field), so Flutter's
// MaterialTextSelectionHandleControls is the same type.
//
// Deviation: the deprecated buildToolbar, canCut, canCopy, canPaste and
// canSelectAll of the controls are not ported (SPEC §12.10).
//
// Flutter: material/text_selection.dart MaterialTextSelectionControls
type MaterialTextSelectionControls struct{}

// MaterialTextSelectionHandleControls is the controls of the selection handles
// of the Material design, to give to a text field, a SelectableRegion or an
// EditableText. It is read-only.
//
// Flutter: material/text_selection.dart materialTextSelectionHandleControls
var MaterialTextSelectionHandleControls widgets.TextSelectionControls = MaterialTextSelectionControls{}

// GetHandleSize is a square of 22 logical pixels.
//
// Flutter: material/text_selection.dart MaterialTextSelectionControls.getHandleSize
func (MaterialTextSelectionControls) GetHandleSize(float64) ui.Size {
	return ui.Size{Width: materialHandleSize, Height: materialHandleSize}
}

// BuildHandle builds a handle that is a circle with one square corner, rotated
// to point at the text.
//
// Flutter: material/text_selection.dart MaterialTextSelectionControls.buildHandle
func (MaterialTextSelectionControls) BuildHandle(bc widgets.BuildContext, handleType rendering.TextSelectionHandleType, _ float64, onTap foundation.VoidCallback) widgets.Widget {
	theme := ThemeOf(bc)
	handleColor := TextSelectionThemeOf(bc).SelectionHandleColor.OrElse(theme.ColorScheme.Primary())
	handle := widgets.Widget(widgets.NewSizedBoxSquare(foundation.Some(materialHandleSize),
		widgets.NewCustomPaint(widgets.NewGestureDetector(nil, widgets.GestureDetectorOpts{OnTap: gestures.GestureTapCallback(onTap), Behavior: rendering.HitTestBehaviorTranslucent}),
			widgets.CustomPaintOpts{Painter: &textSelectionHandlePainter{color: handleColor}})))
	switch handleType {
	case rendering.TextSelectionHandleTypeLeft: // points up-right
		return widgets.NewTransformRotate(math.Pi/2.0, handle)
	case rendering.TextSelectionHandleTypeCollapsed: // points up
		return widgets.NewTransformRotate(math.Pi/4.0, handle)
	}
	return handle // points up-left
}

// GetHandleAnchor is where the tip of the handle is, for each type.
//
// Flutter: material/text_selection.dart MaterialTextSelectionControls.getHandleAnchor
func (MaterialTextSelectionControls) GetHandleAnchor(handleType rendering.TextSelectionHandleType, _ float64) ui.Offset {
	switch handleType {
	case rendering.TextSelectionHandleTypeCollapsed:
		return ui.Offset{Dx: materialHandleSize / 2, Dy: -4}
	case rendering.TextSelectionHandleTypeLeft:
		return ui.Offset{Dx: materialHandleSize}
	}
	return ui.OffsetZero
}

// textSelectionHandlePainter paints a selection handle: a circle and the
// square at its top left corner.
//
// Flutter: material/text_selection.dart _TextSelectionHandlePainter
type textSelectionHandlePainter struct {
	rendering.CustomPainterBase
	color ui.Color
}

// Paint draws the circle and the square in a path, so that they are one shape.
//
// Flutter: material/text_selection.dart _TextSelectionHandlePainter.paint
func (p *textSelectionHandlePainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	radius := size.Width / 2.0
	circle := ui.NewRectFromCircle(ui.Offset{Dx: radius, Dy: radius}, radius)
	point := ui.NewRectFromLTWH(0, 0, radius, radius)
	path := ui.NewPath()
	path.AddOval(circle)
	path.AddRect(point)
	canvas.DrawPath(path, paint)
}

// ShouldRepaint is true when the color changed.
//
// Flutter: material/text_selection.dart _TextSelectionHandlePainter.shouldRepaint
func (p *textSelectionHandlePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	return p.color != old.(*textSelectionHandlePainter).color
}

// HitTest defers to the default.
func (p *textSelectionHandlePainter) HitTest(ui.Offset) foundation.Opt[bool] {
	return foundation.None[bool]()
}
