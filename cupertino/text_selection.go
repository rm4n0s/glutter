package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the selection handles of iOS (the constants of cupertino/text_selection.dart).
const (
	selectionHandleOverlap = 1.5
	selectionHandleRadius  = 6.0
	// arrowScreenPadding is how far from the sides of the screen the arrow of the toolbar stays.
	arrowScreenPadding = 26.0
)

// textSelectionHandlePainter paints a selection handle: a circle on a line.
//
// Flutter: cupertino/text_selection.dart _CupertinoTextSelectionHandlePainter
type textSelectionHandlePainter struct {
	rendering.CustomPainterBase
	color ui.Color
}

func (p *textSelectionHandlePainter) Paint(canvas *ui.Canvas, size ui.Size) {
	const halfStrokeWidth = 1.0
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	circle := ui.NewRectFromCircle(ui.Offset{Dx: selectionHandleRadius, Dy: selectionHandleRadius}, selectionHandleRadius)
	line := ui.NewRectFromPoints(
		ui.Offset{Dx: selectionHandleRadius - halfStrokeWidth, Dy: 2*selectionHandleRadius - selectionHandleOverlap},
		ui.Offset{Dx: selectionHandleRadius + halfStrokeWidth, Dy: size.Height})
	path := ui.NewPath()
	path.AddOval(circle)
	path.AddRect(line)
	canvas.DrawPath(path, paint)
}

func (p *textSelectionHandlePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*textSelectionHandlePainter)
	return !ok || o.color != p.color
}

func (p *textSelectionHandlePainter) HitTest(ui.Offset) foundation.Opt[bool] {
	return foundation.None[bool]()
}

// CupertinoTextSelectionControls are the controls of the text selection of iOS: the handles, which are a circle on a
// line. The toolbar is not part of the controls (it is the context menu builder of the text field), so Flutter's
// CupertinoTextSelectionHandleControls is the same type.
//
// Deviation: the deprecated buildToolbar, canSelectAll and handleSelectAll of the controls are not ported
// (SPEC §12.10).
//
// Flutter: cupertino/text_selection.dart CupertinoTextSelectionControls
type CupertinoTextSelectionControls struct{}

// CupertinoTextSelectionHandleControls are the controls of the iOS selection, to give to a text field, a
// SelectableRegion or an EditableText. It is read-only.
//
// Flutter: cupertino/text_selection.dart cupertinoTextSelectionHandleControls
var CupertinoTextSelectionHandleControls widgets.TextSelectionControls = CupertinoTextSelectionControls{}

// GetHandleSize is the size of a handle: 12 wide, and as high as the line with the circle.
//
// Flutter: cupertino/text_selection.dart CupertinoTextSelectionControls.getHandleSize
func (CupertinoTextSelectionControls) GetHandleSize(textLineHeight float64) ui.Size {
	return ui.Size{Width: selectionHandleRadius * 2, Height: textLineHeight + selectionHandleRadius*2 - selectionHandleOverlap}
}

// BuildHandle builds a handle: the left one is a circle on a line, the right one is the same turned half a turn, and
// the collapsed one is nothing.
//
// Flutter: cupertino/text_selection.dart CupertinoTextSelectionControls.buildHandle
func (c CupertinoTextSelectionControls) BuildHandle(bc widgets.BuildContext, handleType rendering.TextSelectionHandleType, textLineHeight float64, _ foundation.VoidCallback) widgets.Widget {
	customPaint := widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: &textSelectionHandlePainter{
		CustomPainterBase: rendering.NewCustomPainterBase(nil), color: CupertinoDynamicColorResolve(CupertinoThemeOf(bc).SelectionHandleColor(), bc)}})
	switch handleType {
	case rendering.TextSelectionHandleTypeLeft:
		return widgets.NewSizedBoxFromSize(c.GetHandleSize(textLineHeight), customPaint)
	case rendering.TextSelectionHandleTypeRight:
		desiredSize := c.GetHandleSize(textLineHeight)
		handle := widgets.NewSizedBoxFromSize(desiredSize, customPaint)
		transform := ui.NewMatrix4Identity()
		transform.Translate(float32(desiredSize.Width/2), float32(desiredSize.Height/2), 0)
		transform.RotateZ(math.Pi)
		transform.Translate(float32(-desiredSize.Width/2), float32(-desiredSize.Height/2), 0)
		return widgets.NewTransform(transform, handle)
	}
	return widgets.NewSizedBoxFromSize(c.GetHandleSize(textLineHeight), nil)
}

// GetHandleAnchor is where the handle is held: the bottom of the line for the left one, the top of the circle for
// the right one, and the middle of the line for the collapsed one.
//
// Flutter: cupertino/text_selection.dart CupertinoTextSelectionControls.getHandleAnchor
func (c CupertinoTextSelectionControls) GetHandleAnchor(handleType rendering.TextSelectionHandleType, textLineHeight float64) ui.Offset {
	handleSize := c.GetHandleSize(textLineHeight)
	switch handleType {
	case rendering.TextSelectionHandleTypeLeft:
		return ui.Offset{Dx: handleSize.Width / 2, Dy: handleSize.Height}
	case rendering.TextSelectionHandleTypeRight:
		return ui.Offset{Dx: handleSize.Width / 2, Dy: handleSize.Height - 2*selectionHandleRadius + selectionHandleOverlap}
	}
	return ui.Offset{Dx: handleSize.Width / 2, Dy: textLineHeight + (handleSize.Height-textLineHeight)/2}
}
