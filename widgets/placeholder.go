package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// PlaceholderOpts are the optional parameters of NewPlaceholder.
//
// Flutter: widgets/placeholder.dart Placeholder.new
type PlaceholderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color to draw the placeholder box; the default is 0xFF455A64
	// (blue grey 700).
	Color foundation.Opt[ui.Color]
	// StrokeWidth is the width of the lines in the placeholder box; the default
	// is 2.
	StrokeWidth foundation.Opt[float64]
	// FallbackWidth is the width to use when the placeholder is in a situation
	// with an unbounded width; the default is 400.
	FallbackWidth foundation.Opt[float64]
	// FallbackHeight is the height to use when the placeholder is in a situation
	// with an unbounded height; the default is 400.
	FallbackHeight foundation.Opt[float64]
}

// Placeholder is a box that draws two crossed lines, as a stand-in for a widget
// that is not there yet.
//
// Flutter: widgets/placeholder.dart Placeholder
type Placeholder struct {
	StatelessWidget
	child                                      Widget
	color                                      ui.Color
	strokeWidth, fallbackWidth, fallbackHeight float64
}

// NewPlaceholder creates a placeholder; the child, which may be nil, is drawn
// over the lines.
//
// Flutter: widgets/placeholder.dart Placeholder.new
func NewPlaceholder(child Widget, opts ...PlaceholderOpts) *Placeholder {
	o := firstOpts(opts)
	return &Placeholder{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child: child,
		color: o.Color.OrElse(ui.NewColor(0xFF455A64)), strokeWidth: o.StrokeWidth.OrElse(2),
		fallbackWidth: o.FallbackWidth.OrElse(400), fallbackHeight: o.FallbackHeight.OrElse(400)}
}

// Child is the widget drawn over the lines.
//
// Flutter: widgets/placeholder.dart Placeholder.child
func (w *Placeholder) Child() Widget { return w.child }

// Color is the color of the lines.
//
// Flutter: widgets/placeholder.dart Placeholder.color
func (w *Placeholder) Color() ui.Color { return w.color }

// StrokeWidth is the width of the lines.
//
// Flutter: widgets/placeholder.dart Placeholder.strokeWidth
func (w *Placeholder) StrokeWidth() float64 { return w.strokeWidth }

// FallbackWidth is the width used when the constraints are unbounded.
//
// Flutter: widgets/placeholder.dart Placeholder.fallbackWidth
func (w *Placeholder) FallbackWidth() float64 { return w.fallbackWidth }

// FallbackHeight is the height used when the constraints are unbounded.
//
// Flutter: widgets/placeholder.dart Placeholder.fallbackHeight
func (w *Placeholder) FallbackHeight() float64 { return w.fallbackHeight }

// Build builds the painted box, limited to the fallback size when it is
// unconstrained.
//
// Flutter: widgets/placeholder.dart Placeholder.build
func (w *Placeholder) Build(BuildContext) Widget {
	return NewLimitedBox(NewCustomPaint(w.child, CustomPaintOpts{Size: ui.SizeInfinite,
		Painter: &placeholderPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), color: w.color, strokeWidth: w.strokeWidth}}),
		LimitedBoxOpts{MaxWidth: foundation.Some(w.fallbackWidth), MaxHeight: foundation.Some(w.fallbackHeight)})
}

// placeholderPainter draws the box and its two diagonals.
//
// Flutter: widgets/placeholder.dart _PlaceholderPainter
type placeholderPainter struct {
	rendering.CustomPainterBase
	color       ui.Color
	strokeWidth float64
}

func (p *placeholderPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(p.strokeWidth)
	rect := ui.OffsetZero.And(size)
	path := ui.NewPath()
	path.AddRect(rect)
	path.AddPolygon([]ui.Offset{rect.TopRight(), rect.BottomLeft()}, false)
	path.AddPolygon([]ui.Offset{rect.TopLeft(), rect.BottomRight()}, false)
	canvas.DrawPath(path, paint)
}

func (p *placeholderPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*placeholderPainter)
	return o.color != p.color || o.strokeWidth != p.strokeWidth
}

func (p *placeholderPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.Some(false) }
