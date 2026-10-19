package painting

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// LinearBorderEdge defines the relative size and alignment of one LinearBorder
// edge.
//
// Flutter: painting/linear_border.dart LinearBorderEdge
type LinearBorderEdge struct {
	// Size is the length of the edge as a fraction of the available space,
	// from 0 to 1.
	Size float64
	// Alignment is where the edge is placed, from -1 (start) to 1 (end).
	Alignment float64
}

// LinearBorderEdgeOpts are the optional parameters of NewLinearBorderEdge.
//
// Flutter: painting/linear_border.dart LinearBorderEdge.new
type LinearBorderEdgeOpts struct {
	// Size defaults to 1.0; it must be from 0.0 to 1.0.
	Size foundation.Opt[float64]
	// Alignment defaults to 0.0.
	Alignment float64
}

// NewLinearBorderEdge creates an edge. In debug builds a size outside 0 to 1
// panics with the tag LinearBorderEdgeSizeOutOfRange.
//
// Flutter: painting/linear_border.dart LinearBorderEdge.new
func NewLinearBorderEdge(opts ...LinearBorderEdgeOpts) LinearBorderEdge {
	o := firstOpts(opts)
	size := o.Size.OrElse(1.0)
	if foundation.DebugMode && !(size >= 0.0 && size <= 1.0) {
		panic(errors.New("LinearBorderEdgeSizeOutOfRange", "the size of a LinearBorderEdge must be from 0.0 to 1.0", "size", size))
	}
	return LinearBorderEdge{Size: size, Alignment: o.Alignment}
}

// LinearBorderEdgeLerp linearly interpolates between two edges; a missing edge
// is taken as an empty one with the other's alignment.
//
// Flutter: painting/linear_border.dart LinearBorderEdge.lerp
func LinearBorderEdgeLerp(a, b foundation.Opt[LinearBorderEdge], t float64) foundation.Opt[LinearBorderEdge] {
	if a.IsNone() && b.IsNone() {
		return a
	}
	if a == b {
		return a
	}
	av, bv := a.Value(), b.Value()
	if a.IsNone() {
		av = LinearBorderEdge{Alignment: bv.Alignment}
	}
	if b.IsNone() {
		bv = LinearBorderEdge{Alignment: av.Alignment}
	}
	return foundation.Some(LinearBorderEdge{Size: ui.LerpDouble(av.Size, bv.Size, t), Alignment: ui.LerpDouble(av.Alignment, bv.Alignment, t)})
}

// String describes the edge, such as "LinearBorderEdge(size: 0.5)".
//
// Flutter: painting/linear_border.dart LinearBorderEdge.toString
func (e LinearBorderEdge) String() string {
	s := "LinearBorderEdge("
	if e.Size != 1.0 {
		s += "size: " + dartDouble(e.Size)
	}
	if e.Alignment != 0 {
		comma := ""
		if e.Size != 1.0 {
			comma = ", "
		}
		s += comma + "alignment: " + dartDouble(e.Alignment)
	}
	return s + ")"
}

// LinearBorder defines a border that consists of up to four lines, one for
// each edge: start, end, top and bottom.
//
// Flutter: painting/linear_border.dart LinearBorder
type LinearBorder struct {
	ShapeBorderDefaults
	side BorderSide
	// Start is the edge at the start of the reading direction.
	Start foundation.Opt[LinearBorderEdge]
	// End is the edge at the end of the reading direction.
	End foundation.Opt[LinearBorderEdge]
	// Top is the top edge.
	Top foundation.Opt[LinearBorderEdge]
	// Bottom is the bottom edge.
	Bottom foundation.Opt[LinearBorderEdge]
}

// LinearBorderOpts are the optional parameters of NewLinearBorder.
//
// Flutter: painting/linear_border.dart LinearBorder.new
type LinearBorderOpts struct {
	// Side defaults to BorderSideNone.
	Side   BorderSide
	Start  foundation.Opt[LinearBorderEdge]
	End    foundation.Opt[LinearBorderEdge]
	Top    foundation.Opt[LinearBorderEdge]
	Bottom foundation.Opt[LinearBorderEdge]
}

// NewLinearBorder creates a border with the given edges.
//
// Flutter: painting/linear_border.dart LinearBorder.new
func NewLinearBorder(opts ...LinearBorderOpts) LinearBorder {
	o := firstOpts(opts)
	return LinearBorder{side: o.Side.normalize(), Start: o.Start, End: o.End, Top: o.Top, Bottom: o.Bottom}
}

// LinearBorderEdgeOnlyOpts are the optional parameters of the single-edge
// constructors of LinearBorder.
//
// Flutter: painting/linear_border.dart LinearBorder.start
type LinearBorderEdgeOnlyOpts struct {
	// Side defaults to BorderSideNone.
	Side BorderSide
	// Alignment defaults to 0.0.
	Alignment float64
	// Size defaults to 1.0.
	Size foundation.Opt[float64]
}

func linearEdgeOnly(opts []LinearBorderEdgeOnlyOpts, name string) (BorderSide, foundation.Opt[LinearBorderEdge]) {
	o := firstOpts(opts)
	return o.Side.normalize(), foundation.Some(NewLinearBorderEdge(LinearBorderEdgeOpts{Size: o.Size, Alignment: o.Alignment}))
}

// NewLinearBorderStart creates a border with only a start edge.
//
// Flutter: painting/linear_border.dart LinearBorder.start
func NewLinearBorderStart(opts ...LinearBorderEdgeOnlyOpts) LinearBorder {
	side, e := linearEdgeOnly(opts, "NewLinearBorderStart")
	return LinearBorder{side: side, Start: e}
}

// NewLinearBorderEnd creates a border with only an end edge.
//
// Flutter: painting/linear_border.dart LinearBorder.end
func NewLinearBorderEnd(opts ...LinearBorderEdgeOnlyOpts) LinearBorder {
	side, e := linearEdgeOnly(opts, "NewLinearBorderEnd")
	return LinearBorder{side: side, End: e}
}

// NewLinearBorderTop creates a border with only a top edge.
//
// Flutter: painting/linear_border.dart LinearBorder.top
func NewLinearBorderTop(opts ...LinearBorderEdgeOnlyOpts) LinearBorder {
	side, e := linearEdgeOnly(opts, "NewLinearBorderTop")
	return LinearBorder{side: side, Top: e}
}

// NewLinearBorderBottom creates a border with only a bottom edge.
//
// Flutter: painting/linear_border.dart LinearBorder.bottom
func NewLinearBorderBottom(opts ...LinearBorderEdgeOnlyOpts) LinearBorder {
	side, e := linearEdgeOnly(opts, "NewLinearBorderBottom")
	return LinearBorder{side: side, Bottom: e}
}

// LinearBorderNone is a border without any visible sides.
//
// Flutter: painting/linear_border.dart LinearBorder.none
var LinearBorderNone = NewLinearBorder()

// Side is the border outline's color and weight.
//
// Flutter: painting/borders.dart OutlinedBorder.side
func (b LinearBorder) Side() BorderSide { return b.side.normalize() }

// Scale scales the side by t; the edges are dropped.
//
// Flutter: painting/linear_border.dart LinearBorder.scale
func (b LinearBorder) Scale(t float64) ShapeBorder { return LinearBorder{side: b.Side().Scale(t)} }

// Dimensions is the width of the side on each edge that is present.
//
// Flutter: painting/linear_border.dart LinearBorder.dimensions
func (b LinearBorder) Dimensions() EdgeInsetsGeometry {
	w := b.Side().Width
	pick := func(e foundation.Opt[LinearBorderEdge]) float64 {
		if e.IsNone() {
			return 0.0
		}
		return w
	}
	return NewEdgeInsetsDirectionalFromSTEB(pick(b.Start), pick(b.Top), pick(b.End), pick(b.Bottom))
}

// LerpFrom interpolates from a LinearBorder.
//
// Flutter: painting/linear_border.dart LinearBorder.lerpFrom
func (b LinearBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(LinearBorder); ok {
		return LinearBorder{
			side:  BorderSideLerp(x.Side(), b.Side(), t),
			Start: LinearBorderEdgeLerp(x.Start, b.Start, t), End: LinearBorderEdgeLerp(x.End, b.End, t),
			Top: LinearBorderEdgeLerp(x.Top, b.Top, t), Bottom: LinearBorderEdgeLerp(x.Bottom, b.Bottom, t),
		}
	}
	return ShapeBorderDefaultLerpFrom(b, a, t)
}

// LerpTo interpolates to a LinearBorder.
//
// Flutter: painting/linear_border.dart LinearBorder.lerpTo
func (b LinearBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if x, ok := o.(LinearBorder); ok {
		return LinearBorder{
			side:  BorderSideLerp(b.Side(), x.Side(), t),
			Start: LinearBorderEdgeLerp(b.Start, x.Start, t), End: LinearBorderEdgeLerp(b.End, x.End, t),
			Top: LinearBorderEdgeLerp(b.Top, x.Top, t), Bottom: LinearBorderEdgeLerp(b.Bottom, x.Bottom, t),
		}
	}
	return ShapeBorderDefaultLerpTo(b, o, t)
}

// LinearBorderCopy holds the replacements for LinearBorder.CopyWith.
//
// Flutter: painting/linear_border.dart LinearBorder.copyWith
type LinearBorderCopy struct {
	Side   foundation.Opt[BorderSide]
	Start  foundation.Opt[LinearBorderEdge]
	End    foundation.Opt[LinearBorderEdge]
	Top    foundation.Opt[LinearBorderEdge]
	Bottom foundation.Opt[LinearBorderEdge]
}

func orEdge(n, old foundation.Opt[LinearBorderEdge]) foundation.Opt[LinearBorderEdge] {
	if n.IsSome() {
		return n
	}
	return old
}

// CopyWith returns a copy with the given fields replaced.
//
// Flutter: painting/linear_border.dart LinearBorder.copyWith
func (b LinearBorder) CopyWith(c LinearBorderCopy) LinearBorder {
	return LinearBorder{
		side: c.Side.OrElse(b.Side()), Start: orEdge(c.Start, b.Start), End: orEdge(c.End, b.End),
		Top: orEdge(c.Top, b.Top), Bottom: orEdge(c.Bottom, b.Bottom),
	}
}

// CopyWithSide returns a copy with the given side.
//
// Flutter: painting/borders.dart OutlinedBorder.copyWith
func (b LinearBorder) CopyWithSide(side BorderSide) OutlinedBorder {
	return b.CopyWith(LinearBorderCopy{Side: foundation.Some(side)})
}

// GetInnerPath is the rect, deflated by the dimensions.
//
// Flutter: painting/linear_border.dart LinearBorder.getInnerPath
func (b LinearBorder) GetInnerPath(rect ui.Rect, td ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(b.Dimensions().Resolve(td).DeflateRect(rect))
	return p
}

// GetOuterPath is the rect.
//
// Flutter: painting/linear_border.dart LinearBorder.getOuterPath
func (b LinearBorder) GetOuterPath(rect ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

// Paint draws the edges.
//
// Flutter: painting/linear_border.dart LinearBorder.paint
func (b LinearBorder) Paint(canvas *ui.Canvas, rect ui.Rect, td ui.TextDirection) {
	side := b.Side()
	insets := b.Dimensions().Resolve(td)
	rtl := td == ui.TextDirectionRTL
	path := ui.NewPath()
	paint := ui.NewPaint()
	paint.SetStrokeWidth(0.0)

	drawEdge := func(r ui.Rect, color ui.Color) {
		paint.SetColor(color)
		path.Reset()
		path.MoveTo(r.Left, r.Top)
		switch {
		case r.Width() == 0.0:
			paint.SetStyle(ui.PaintingStyleStroke)
			path.LineTo(r.Left, r.Bottom)
		case r.Height() == 0.0:
			paint.SetStyle(ui.PaintingStyleStroke)
			path.LineTo(r.Right, r.Top)
		default:
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(r.Right, r.Top)
			path.LineTo(r.Right, r.Bottom)
			path.LineTo(r.Left, r.Bottom)
		}
		canvas.DrawPath(path, paint)
	}

	visible := func(e foundation.Opt[LinearBorderEdge]) (LinearBorderEdge, bool) {
		v, ok := e.Get()
		return v, ok && v.Size != 0.0 && side.Style != BorderStyleNone
	}
	if e, ok := visible(b.Start); ok {
		insetRect := ui.NewRectFromLTWH(rect.Left, rect.Top+insets.Top, rect.Width(), rect.Height()-insets.Vertical())
		x, width := rect.Left, insets.Left
		if rtl {
			x, width = rect.Right-insets.Right, insets.Right
		}
		height := insetRect.Height() * e.Size
		y := (insetRect.Height() - height) * ((e.Alignment + 1.0) / 2.0)
		drawEdge(ui.NewRectFromLTWH(x, y, width, height), side.Color)
	}
	if e, ok := visible(b.End); ok {
		insetRect := ui.NewRectFromLTWH(rect.Left, rect.Top+insets.Top, rect.Width(), rect.Height()-insets.Vertical())
		x, width := rect.Right-insets.Right, insets.Right
		if rtl {
			x, width = rect.Left, insets.Left
		}
		height := insetRect.Height() * e.Size
		y := (insetRect.Height() - height) * ((e.Alignment + 1.0) / 2.0)
		drawEdge(ui.NewRectFromLTWH(x, y, width, height), side.Color)
	}
	if e, ok := visible(b.Top); ok {
		width := rect.Width() * e.Size
		startX := (rect.Width() - width) * ((e.Alignment + 1.0) / 2.0)
		x := startX
		if rtl {
			x = rect.Width() - startX - width
		}
		drawEdge(ui.NewRectFromLTWH(x, rect.Top, width, insets.Top), side.Color)
	}
	if e, ok := visible(b.Bottom); ok {
		width := rect.Width() * e.Size
		startX := (rect.Width() - width) * ((e.Alignment + 1.0) / 2.0)
		x := startX
		if rtl {
			x = rect.Width() - startX - width
		}
		drawEdge(ui.NewRectFromLTWH(x, rect.Bottom-insets.Bottom, width, side.Width), side.Color)
	}
}

// String describes the border.
//
// Flutter: painting/linear_border.dart LinearBorder.toString
func (b LinearBorder) String() string {
	if b == LinearBorderNone {
		return "LinearBorder.none"
	}
	s := "LinearBorder(side: " + b.Side().String()
	for _, e := range []struct {
		name string
		edge foundation.Opt[LinearBorderEdge]
	}{{"start", b.Start}, {"end", b.End}, {"top", b.Top}, {"bottom", b.Bottom}} {
		if v, ok := e.edge.Get(); ok {
			s += ", " + e.name + ": " + v.String()
		}
	}
	return s + ")"
}

var _ OutlinedBorder = LinearBorder{}
