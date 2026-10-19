package painting

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// StadiumBorder is a border that fits a stadium-shaped border (a box with
// semicircles on the ends) within the rectangle of the widget it is applied to.
//
// Flutter: painting/stadium_border.dart StadiumBorder
type StadiumBorder struct {
	ShapeBorderDefaults
	side BorderSide
}

// StadiumBorderOpts are the optional parameters of NewStadiumBorder.
//
// Flutter: painting/stadium_border.dart StadiumBorder.new
type StadiumBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
}

// NewStadiumBorder creates a stadium border.
//
// Flutter: painting/stadium_border.dart StadiumBorder.new
func NewStadiumBorder(opts ...StadiumBorderOpts) StadiumBorder {
	o := firstOpts(opts)
	return StadiumBorder{side: o.Side.normalize()}
}

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b StadiumBorder) Side() BorderSide { return b.side.normalize() }

// Dimensions is the stroke inset on all sides.
//
// Flutter: painting/borders.dart OutlinedBorder.dimensions
func (b StadiumBorder) Dimensions() EdgeInsetsGeometry { return OutlinedBorderDimensions(b.Side()) }

// Scale scales the side by t.
//
// Flutter: painting/stadium_border.dart StadiumBorder.scale
func (b StadiumBorder) Scale(t float64) ShapeBorder { return StadiumBorder{side: b.Side().Scale(t)} }

// LerpFrom interpolates from a StadiumBorder, CircleBorder or
// RoundedRectangleBorder.
//
// Flutter: painting/stadium_border.dart StadiumBorder.lerpFrom
func (b StadiumBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if s, ok := a.(StadiumBorder); ok {
		return StadiumBorder{side: BorderSideLerp(s.Side(), b.Side(), t)}
	}
	if c, ok := asCircleBorder(a); ok {
		return stadiumToCircleBorder{side: BorderSideLerp(c.Side(), b.Side(), t), circularity: 1.0 - t, eccentricity: c.Eccentricity}
	}
	if r, ok := a.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{side: BorderSideLerp(r.Side(), b.Side(), t), borderRadius: r.radius(), rectilinearity: 1.0 - t}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a StadiumBorder, CircleBorder or RoundedRectangleBorder.
//
// Flutter: painting/stadium_border.dart StadiumBorder.lerpTo
func (b StadiumBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if s, ok := o.(StadiumBorder); ok {
		return StadiumBorder{side: BorderSideLerp(b.Side(), s.Side(), t)}
	}
	if c, ok := asCircleBorder(o); ok {
		return stadiumToCircleBorder{side: BorderSideLerp(b.Side(), c.Side(), t), circularity: t, eccentricity: c.Eccentricity}
	}
	if r, ok := o.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{side: BorderSideLerp(b.Side(), r.Side(), t), borderRadius: r.radius(), rectilinearity: t}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// StadiumBorderCopy holds the replacements for StadiumBorder.CopyWith.
//
// Flutter: painting/stadium_border.dart StadiumBorder.copyWith
type StadiumBorderCopy struct {
	Side foundation.Opt[BorderSide]
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/stadium_border.dart StadiumBorder.copyWith
func (b StadiumBorder) CopyWith(c StadiumBorderCopy) StadiumBorder {
	return StadiumBorder{side: c.Side.OrElse(b.Side())}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b StadiumBorder) CopyWithSide(side BorderSide) OutlinedBorder { return StadiumBorder{side: side} }

func stadiumRRect(rect ui.Rect) ui.RRect {
	return ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2.0))
}

// GetInnerPath is the stadium, deflated by the stroke inset.
//
// Flutter: painting/stadium_border.dart StadiumBorder.getInnerPath
func (b StadiumBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(stadiumRRect(rect).Deflate(b.Side().StrokeInset()))
	return p
}

// GetOuterPath is the stadium.
//
// Flutter: painting/stadium_border.dart StadiumBorder.getOuterPath
func (b StadiumBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(stadiumRRect(rect))
	return p
}

// PaintInterior fills the stadium.
//
// Flutter: painting/stadium_border.dart StadiumBorder.paintInterior
func (b StadiumBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	canvas.DrawRRect(stadiumRRect(rect), paint)
}

// PreferPaintInterior is true.
//
// Flutter: painting/stadium_border.dart StadiumBorder.preferPaintInterior
func (b StadiumBorder) PreferPaintInterior() bool { return true }

// Paint strokes the stadium.
//
// Flutter: painting/stadium_border.dart StadiumBorder.paint
func (b StadiumBorder) Paint(canvas *ui.Canvas, rect ui.Rect, _ ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	canvas.DrawRRect(stadiumRRect(rect).Inflate(side.StrokeOffset()/2), side.ToPaint())
}

// String describes the border.
//
// Flutter: painting/stadium_border.dart StadiumBorder.toString
func (b StadiumBorder) String() string { return "StadiumBorder(" + b.Side().String() + ")" }

// stadiumToCircleBorder is Flutter's _StadiumToCircleBorder.
type stadiumToCircleBorder struct {
	ShapeBorderDefaults
	side         BorderSide
	circularity  float64
	eccentricity float64
}

func (b stadiumToCircleBorder) Side() BorderSide { return b.side.normalize() }

func (b stadiumToCircleBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

func (b stadiumToCircleBorder) Scale(t float64) ShapeBorder {
	return stadiumToCircleBorder{side: b.Side().Scale(t), circularity: t, eccentricity: b.eccentricity}
}

func (b stadiumToCircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if s, ok := a.(StadiumBorder); ok {
		return stadiumToCircleBorder{side: BorderSideLerp(s.Side(), b.Side(), t), circularity: b.circularity * t, eccentricity: b.eccentricity}
	}
	if c, ok := asCircleBorder(a); ok {
		return stadiumToCircleBorder{
			side: BorderSideLerp(c.Side(), b.Side(), t), circularity: b.circularity + (1.0-b.circularity)*(1.0-t), eccentricity: c.Eccentricity,
		}
	}
	if o, ok := a.(stadiumToCircleBorder); ok {
		return stadiumToCircleBorder{
			side:        BorderSideLerp(o.Side(), b.Side(), t),
			circularity: ui.LerpDouble(o.circularity, b.circularity, t), eccentricity: ui.LerpDouble(o.eccentricity, b.eccentricity, t),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

func (b stadiumToCircleBorder) LerpTo(x ShapeBorder, t float64) ShapeBorder {
	if s, ok := x.(StadiumBorder); ok {
		return stadiumToCircleBorder{side: BorderSideLerp(b.Side(), s.Side(), t), circularity: b.circularity * (1.0 - t), eccentricity: b.eccentricity}
	}
	if c, ok := asCircleBorder(x); ok {
		return stadiumToCircleBorder{
			side: BorderSideLerp(b.Side(), c.Side(), t), circularity: b.circularity + (1.0-b.circularity)*t, eccentricity: c.Eccentricity,
		}
	}
	if o, ok := x.(stadiumToCircleBorder); ok {
		return stadiumToCircleBorder{
			side:        BorderSideLerp(b.Side(), o.Side(), t),
			circularity: ui.LerpDouble(b.circularity, o.circularity, t), eccentricity: ui.LerpDouble(b.eccentricity, o.eccentricity, t),
		}
	}
	return ShapeBorderDefaultLerpTo(b, x, t)
}

func (b stadiumToCircleBorder) adjustRect(rect ui.Rect) ui.Rect {
	return adjustCircularRect(rect, b.circularity, b.eccentricity)
}

func (b stadiumToCircleBorder) adjustBorderRadius(rect ui.Rect) BorderRadius {
	circle := NewBorderRadiusCircular(rect.ShortestSide() / 2)
	if b.eccentricity != 0.0 {
		oval := ovalBorderRadius(rect, b.eccentricity)
		return *BorderRadiusLerp(&circle, &oval, b.circularity)
	}
	return circle
}

func (b stadiumToCircleBorder) rrect(rect ui.Rect) ui.RRect {
	return b.adjustBorderRadius(rect).ToRRect(b.adjustRect(rect))
}

func (b stadiumToCircleBorder) GetInnerPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.rrect(rect).Deflate(b.Side().StrokeInset()))
	return p
}

func (b stadiumToCircleBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.rrect(rect))
	return p
}

func (b stadiumToCircleBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, _ ui.TextDirection) {
	canvas.DrawRRect(b.rrect(rect), paint)
}

func (b stadiumToCircleBorder) PreferPaintInterior() bool { return true }

func (b stadiumToCircleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return stadiumToCircleBorder{side: side, circularity: b.circularity, eccentricity: b.eccentricity}
}

func (b stadiumToCircleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, _ ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	canvas.DrawRRect(b.rrect(rect).Inflate(side.StrokeOffset()/2), side.ToPaint())
}

func (b stadiumToCircleBorder) String() string {
	pct := dartfmt.ToStringAsFixed(b.circularity*100, 1)
	if b.eccentricity != 0.0 {
		return "StadiumBorder(" + b.Side().String() + ", " + pct + "% of the way to being a CircleBorder that is " +
			dartfmt.ToStringAsFixed(b.eccentricity*100, 1) + "% oval)"
	}
	return "StadiumBorder(" + b.Side().String() + ", " + pct + "% of the way to being a CircleBorder)"
}

// stadiumToRoundedRectangleBorder is Flutter's _StadiumToRoundedRectangleBorder.
type stadiumToRoundedRectangleBorder struct {
	ShapeBorderDefaults
	side           BorderSide
	borderRadius   BorderRadiusGeometry
	rectilinearity float64
}

func (b stadiumToRoundedRectangleBorder) Side() BorderSide { return b.side.normalize() }

func (b stadiumToRoundedRectangleBorder) radius() BorderRadiusGeometry {
	if b.borderRadius == nil {
		return BorderRadiusZero
	}
	return b.borderRadius
}

func (b stadiumToRoundedRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return OutlinedBorderDimensions(b.Side())
}

func (b stadiumToRoundedRectangleBorder) Scale(t float64) ShapeBorder {
	return stadiumToRoundedRectangleBorder{side: b.Side().Scale(t), borderRadius: BorderRadiusGeometryMul(b.radius(), t), rectilinearity: t}
}

func (b stadiumToRoundedRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if s, ok := a.(StadiumBorder); ok {
		return stadiumToRoundedRectangleBorder{side: BorderSideLerp(s.Side(), b.Side(), t), borderRadius: b.radius(), rectilinearity: b.rectilinearity * t}
	}
	if r, ok := a.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{
			side: BorderSideLerp(r.Side(), b.Side(), t), borderRadius: b.radius(), rectilinearity: b.rectilinearity + (1.0-b.rectilinearity)*(1.0-t),
		}
	}
	if o, ok := a.(stadiumToRoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{
			side: BorderSideLerp(o.Side(), b.Side(), t), borderRadius: BorderRadiusGeometryLerp(o.radius(), b.radius(), t),
			rectilinearity: ui.LerpDouble(o.rectilinearity, b.rectilinearity, t),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

func (b stadiumToRoundedRectangleBorder) LerpTo(x ShapeBorder, t float64) ShapeBorder {
	if s, ok := x.(StadiumBorder); ok {
		return stadiumToRoundedRectangleBorder{
			side: BorderSideLerp(b.Side(), s.Side(), t), borderRadius: b.radius(), rectilinearity: b.rectilinearity * (1.0 - t),
		}
	}
	if r, ok := x.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{
			side: BorderSideLerp(b.Side(), r.Side(), t), borderRadius: b.radius(), rectilinearity: b.rectilinearity + (1.0-b.rectilinearity)*t,
		}
	}
	if o, ok := x.(stadiumToRoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{
			side: BorderSideLerp(b.Side(), o.Side(), t), borderRadius: BorderRadiusGeometryLerp(b.radius(), o.radius(), t),
			rectilinearity: ui.LerpDouble(b.rectilinearity, o.rectilinearity, t),
		}
	}
	return ShapeBorderDefaultLerpTo(b, x, t)
}

func (b stadiumToRoundedRectangleBorder) adjustBorderRadius(rect ui.Rect) BorderRadiusGeometry {
	return BorderRadiusGeometryLerp(b.radius(), NewBorderRadiusAll(ui.NewRadiusCircular(rect.ShortestSide()/2.0)), 1.0-b.rectilinearity)
}

func (b stadiumToRoundedRectangleBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	side := b.Side()
	borderRect := b.adjustBorderRadius(rect).Resolve(td).ToRRect(rect)
	p := ui.NewPath()
	p.AddRRect(borderRect.Deflate(ui.LerpDouble(side.Width, 0, side.StrokeAlign)))
	return p
}

func (b stadiumToRoundedRectangleBorder) GetOuterPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.adjustBorderRadius(rect).Resolve(td).ToRRect(rect))
	return p
}

func (b stadiumToRoundedRectangleBorder) PaintInterior(canvas *ui.Canvas, rect ui.Rect, paint *ui.Paint, td ui.TextDirection) {
	adjusted := b.adjustBorderRadius(rect)
	if adjusted.Equal(BorderRadiusZero) {
		canvas.DrawRect(rect, paint)
		return
	}
	canvas.DrawRRect(adjusted.Resolve(td).ToRRect(rect), paint)
}

func (b stadiumToRoundedRectangleBorder) PreferPaintInterior() bool { return true }

func (b stadiumToRoundedRectangleBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return stadiumToRoundedRectangleBorder{side: side, borderRadius: b.radius(), rectilinearity: b.rectilinearity}
}

func (b stadiumToRoundedRectangleBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	side := b.Side()
	if side.Style != BorderStyleSolid {
		return
	}
	borderRect := b.adjustBorderRadius(rect).Resolve(td).ToRRect(rect)
	canvas.DrawRRect(borderRect.Inflate(side.StrokeOffset()/2), side.ToPaint())
}

func (b stadiumToRoundedRectangleBorder) String() string {
	return "StadiumBorder(" + b.Side().String() + ", " + b.radius().String() + ", " +
		dartfmt.ToStringAsFixed(b.rectilinearity*100, 1) + "% of the way to being a RoundedRectangleBorder)"
}

var (
	_ OutlinedBorder = StadiumBorder{}
	_ OutlinedBorder = stadiumToCircleBorder{}
	_ OutlinedBorder = stadiumToRoundedRectangleBorder{}
)
