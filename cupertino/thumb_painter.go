package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// thumbBorderColor is the color of the thin border around a thumb.
var thumbBorderColor = ui.NewColor(0x0A000000)

func boxShadow(color uint32, dy, blur float64) painting.BoxShadow {
	c := ui.NewColor(color)
	return painting.BoxShadow{Shadow: ui.NewShadow(ui.ShadowOpts{Color: &c, Offset: ui.Offset{Dy: dy}, BlurRadius: blur})}
}

// switchBoxShadows are the shadows of the thumb of a switch.
func switchBoxShadows() []painting.BoxShadow {
	return []painting.BoxShadow{boxShadow(0x26000000, 3, 8.0), boxShadow(0x0F000000, 3, 1.0)}
}

// sliderBoxShadows are the shadows of the thumb of a slider.
func sliderBoxShadows() []painting.BoxShadow {
	return []painting.BoxShadow{boxShadow(0x26000000, 3, 8.0), boxShadow(0x29000000, 1, 1.0), boxShadow(0x1A000000, 3, 1.0)}
}

// CupertinoThumbPainterOpts are the optional parameters of NewCupertinoThumbPainter and of
// NewCupertinoThumbPainterSwitchThumb.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.new
type CupertinoThumbPainterOpts struct {
	// Color is the color of the thumb; the default is white.
	Color foundation.Opt[ui.Color]
	// Shadows are the shadows of the thumb; the default is the slider's, and the switch thumb's for the switch form.
	Shadows []painting.BoxShadow
}

// CupertinoThumbPainter paints the thumb of a switch or a slider: a circle (or a stadium) with shadows under it.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter
type CupertinoThumbPainter struct {
	color   ui.Color
	shadows []painting.BoxShadow
}

// NewCupertinoThumbPainter creates the painter of the thumb of a slider.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.new
func NewCupertinoThumbPainter(opts ...CupertinoThumbPainterOpts) CupertinoThumbPainter {
	o := firstOf(opts)
	color := o.Color.OrElse(CupertinoColors.White)
	shadows := o.Shadows
	if shadows == nil {
		shadows = sliderBoxShadows()
	}
	return CupertinoThumbPainter{color, shadows}
}

// NewCupertinoThumbPainterSwitchThumb creates the painter of the thumb of a switch.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.switchThumb
func NewCupertinoThumbPainterSwitchThumb(opts ...CupertinoThumbPainterOpts) CupertinoThumbPainter {
	o := firstOf(opts)
	if o.Shadows == nil {
		o.Shadows = switchBoxShadows()
	}
	return NewCupertinoThumbPainter(o)
}

// CupertinoThumbPainterRadius is the radius of the thumb.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.radius
const CupertinoThumbPainterRadius = 14.0

// CupertinoThumbPainterExtension is how much the thumb of a switch stretches while pressed.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.extension
const CupertinoThumbPainterExtension = 7.0

// Color is the color of the thumb.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.color
func (p CupertinoThumbPainter) Color() ui.Color { return p.color }

// Shadows are the shadows of the thumb.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.shadows
func (p CupertinoThumbPainter) Shadows() []painting.BoxShadow { return p.shadows }

// Paint paints the thumb in the rect.
//
// Flutter: cupertino/thumb_painter.dart CupertinoThumbPainter.paint
func (p CupertinoThumbPainter) Paint(canvas *ui.Canvas, rect ui.Rect) {
	thumbShape := ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2.0))
	for _, shadow := range p.shadows {
		canvas.DrawRRect(thumbShape.Shift(shadow.Offset), shadow.ToPaint())
	}
	border := ui.NewPaint()
	border.SetColor(thumbBorderColor)
	canvas.DrawRRect(thumbShape.Inflate(0.5), border)
	fill := ui.NewPaint()
	fill.SetColor(p.color)
	canvas.DrawRRect(thumbShape, fill)
}
