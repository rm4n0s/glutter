package painting

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// edgeParts are the six distances of an EdgeInsetsGeometry: Flutter's
// protected getters _left, _right, _start, _end, _top and _bottom.
type edgeParts struct{ left, right, start, end, top, bottom float64 }

func (p edgeParts) isNonNegative() bool {
	return p.left >= 0 && p.right >= 0 && p.start >= 0 && p.end >= 0 && p.top >= 0 && p.bottom >= 0
}

func (p edgeParts) horizontal() float64 { return p.left + p.right + p.start + p.end }
func (p edgeParts) vertical() float64   { return p.top + p.bottom }

func (p edgeParts) String() string {
	f := func(v float64) string { return dartfmt.ToStringAsFixed(v, 1) }
	if p.start == 0 && p.end == 0 {
		if p.left == 0 && p.right == 0 && p.top == 0 && p.bottom == 0 {
			return "EdgeInsets.zero"
		}
		if p.left == p.right && p.right == p.top && p.top == p.bottom {
			return "EdgeInsets.all(" + f(p.left) + ")"
		}
		return fmt.Sprintf("EdgeInsets(%s, %s, %s, %s)", f(p.left), f(p.top), f(p.right), f(p.bottom))
	}
	if p.left == 0 && p.right == 0 {
		return fmt.Sprintf("EdgeInsetsDirectional(%s, %s, %s, %s)", f(p.start), f(p.top), f(p.end), f(p.bottom))
	}
	return fmt.Sprintf("EdgeInsets(%s, %s, %s, %s) + EdgeInsetsDirectional(%s, 0.0, %s, 0.0)",
		f(p.left), f(p.top), f(p.right), f(p.bottom), f(p.start), f(p.end))
}

// EdgeInsetsGeometry is a base for EdgeInsets and EdgeInsetsDirectional, which
// describe offsets from each side of a box. Its covariant operations (Flipped,
// Neg, Mul, Div, TruncDiv and Mod) are the functions EdgeInsetsGeometryFlipped,
// EdgeInsetsGeometryNeg, … and the methods of the concrete types.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry
type EdgeInsetsGeometry interface {
	// IsNonNegative reports whether every distance is zero or more.
	IsNonNegative() bool
	// Horizontal is the total offset in the horizontal direction.
	Horizontal() float64
	// Vertical is the total offset in the vertical direction.
	Vertical() float64
	// Along is the total offset along the axis.
	Along(axis Axis) float64
	// CollapsedSize is a size that will contain the offsets: Size(Horizontal, Vertical).
	CollapsedSize() ui.Size
	// InflateSize returns the size grown by the insets.
	InflateSize(size ui.Size) ui.Size
	// DeflateSize returns the size shrunk by the insets.
	DeflateSize(size ui.Size) ui.Size
	// Subtract returns the result of subtracting the distances of other.
	Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry
	// Add returns the result of adding the distances of other.
	Add(other EdgeInsetsGeometry) EdgeInsetsGeometry
	// Clamp clamps each distance between those of min and max.
	Clamp(min, max EdgeInsetsGeometry) EdgeInsetsGeometry
	// Resolve returns an EdgeInsets for the text direction. A directional
	// value panics with the tag TextDirectionRequired in debug builds if the
	// direction is unspecified.
	Resolve(direction ui.TextDirection) EdgeInsets
	// Equal reports whether the six distances are equal.
	Equal(other EdgeInsetsGeometry) bool
	String() string
	edgeParts() edgeParts
}

// EdgeInsetsGeometryInfinity is an EdgeInsetsGeometry with infinite offsets in
// every direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.infinity
var EdgeInsetsGeometryInfinity EdgeInsetsGeometry = mixedEdgeInsets(edgeParts{
	math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1)})

// mixedEdgeInsets is Flutter's _MixedEdgeInsets: both physical and directional
// distances.
type mixedEdgeInsets edgeParts

func (m mixedEdgeInsets) edgeParts() edgeParts { return edgeParts(m) }
func (m mixedEdgeInsets) IsNonNegative() bool  { return edgeParts(m).isNonNegative() }
func (m mixedEdgeInsets) Horizontal() float64  { return edgeParts(m).horizontal() }
func (m mixedEdgeInsets) Vertical() float64    { return edgeParts(m).vertical() }
func (m mixedEdgeInsets) Along(a Axis) float64 { return alongAxis(m, a) }
func (m mixedEdgeInsets) CollapsedSize() ui.Size {
	return ui.Size{Width: m.Horizontal(), Height: m.Vertical()}
}
func (m mixedEdgeInsets) InflateSize(s ui.Size) ui.Size { return inflateSize(m, s) }
func (m mixedEdgeInsets) DeflateSize(s ui.Size) ui.Size { return deflateSize(m, s) }
func (m mixedEdgeInsets) Subtract(o EdgeInsetsGeometry) EdgeInsetsGeometry {
	return subtractEdges(m, o)
}
func (m mixedEdgeInsets) Add(o EdgeInsetsGeometry) EdgeInsetsGeometry { return addEdges(m, o) }
func (m mixedEdgeInsets) Clamp(lo, hi EdgeInsetsGeometry) EdgeInsetsGeometry {
	return clampEdges(m, lo, hi)
}
func (m mixedEdgeInsets) Equal(o EdgeInsetsGeometry) bool {
	return o != nil && edgeParts(m) == o.edgeParts()
}
func (m mixedEdgeInsets) String() string { return edgeParts(m).String() }

// Resolve implements EdgeInsetsGeometry.
//
// Flutter: painting/edge_insets.dart _MixedEdgeInsets.resolve
func (m mixedEdgeInsets) Resolve(direction ui.TextDirection) EdgeInsets {
	debugCheckCanResolveTextDirection(direction, "_MixedEdgeInsets")
	if direction == ui.TextDirectionRTL {
		return EdgeInsets{Left: m.end + m.left, Top: m.top, Right: m.start + m.right, Bottom: m.bottom}
	}
	return EdgeInsets{Left: m.start + m.left, Top: m.top, Right: m.end + m.right, Bottom: m.bottom}
}

func alongAxis(g EdgeInsetsGeometry, axis Axis) float64 {
	if axis == AxisVertical {
		return g.Vertical()
	}
	return g.Horizontal()
}

func inflateSize(g EdgeInsetsGeometry, s ui.Size) ui.Size {
	return ui.Size{Width: s.Width + g.Horizontal(), Height: s.Height + g.Vertical()}
}

func deflateSize(g EdgeInsetsGeometry, s ui.Size) ui.Size {
	return ui.Size{Width: s.Width - g.Horizontal(), Height: s.Height - g.Vertical()}
}

func subtractEdges(a, b EdgeInsetsGeometry) EdgeInsetsGeometry {
	x, y := a.edgeParts(), b.edgeParts()
	return mixedEdgeInsets{x.left - y.left, x.right - y.right, x.start - y.start, x.end - y.end, x.top - y.top, x.bottom - y.bottom}
}

func addEdges(a, b EdgeInsetsGeometry) EdgeInsetsGeometry {
	x, y := a.edgeParts(), b.edgeParts()
	return mixedEdgeInsets{x.left + y.left, x.right + y.right, x.start + y.start, x.end + y.end, x.top + y.top, x.bottom + y.bottom}
}

func clampEdges(a, lo, hi EdgeInsetsGeometry) EdgeInsetsGeometry {
	x, l, h := a.edgeParts(), lo.edgeParts(), hi.edgeParts()
	return mixedEdgeInsets{
		ui.ClampDouble(x.left, l.left, h.left), ui.ClampDouble(x.right, l.right, h.right),
		ui.ClampDouble(x.start, l.start, h.start), ui.ClampDouble(x.end, l.end, h.end),
		ui.ClampDouble(x.top, l.top, h.top), ui.ClampDouble(x.bottom, l.bottom, h.bottom)}
}

func truncDiv(a, b float64) float64 { return math.Trunc(a / b) }

// dartMod is Dart's % on doubles: the result has the sign of the divisor, or
// is NaN when the divisor is 0.
func dartMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		if b < 0 {
			r -= b
		} else {
			r += b
		}
	}
	return r
}

// mapEdges applies f to the six distances of a geometry and returns a value of
// the same kind.
func mapEdges(g EdgeInsetsGeometry, f func(float64) float64) EdgeInsetsGeometry {
	switch v := g.(type) {
	case EdgeInsets:
		return EdgeInsets{f(v.Left), f(v.Top), f(v.Right), f(v.Bottom)}
	case EdgeInsetsDirectional:
		return EdgeInsetsDirectional{f(v.Start), f(v.Top), f(v.End), f(v.Bottom)}
	}
	p := g.edgeParts()
	return mixedEdgeInsets{f(p.left), f(p.right), f(p.start), f(p.end), f(p.top), f(p.bottom)}
}

// EdgeInsetsGeometryNeg negates every distance.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.operator -
func EdgeInsetsGeometryNeg(g EdgeInsetsGeometry) EdgeInsetsGeometry {
	return mapEdges(g, func(v float64) float64 { return -v })
}

// EdgeInsetsGeometryMul multiplies every distance by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.operator *
func EdgeInsetsGeometryMul(g EdgeInsetsGeometry, k float64) EdgeInsetsGeometry {
	return mapEdges(g, func(v float64) float64 { return v * k })
}

// EdgeInsetsGeometryDiv divides every distance by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.operator /
func EdgeInsetsGeometryDiv(g EdgeInsetsGeometry, k float64) EdgeInsetsGeometry {
	return mapEdges(g, func(v float64) float64 { return v / k })
}

// EdgeInsetsGeometryTruncDiv integer-divides every distance by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.operator ~/
func EdgeInsetsGeometryTruncDiv(g EdgeInsetsGeometry, k float64) EdgeInsetsGeometry {
	return mapEdges(g, func(v float64) float64 { return truncDiv(v, k) })
}

// EdgeInsetsGeometryMod takes the modulo k of every distance.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.operator %
func EdgeInsetsGeometryMod(g EdgeInsetsGeometry, k float64) EdgeInsetsGeometry {
	return mapEdges(g, func(v float64) float64 { return dartMod(v, k) })
}

// EdgeInsetsGeometryFlipped returns the insets with left and right swapped, and
// top and bottom.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.flipped
func EdgeInsetsGeometryFlipped(g EdgeInsetsGeometry) EdgeInsetsGeometry {
	switch v := g.(type) {
	case EdgeInsets:
		return v.Flipped()
	case EdgeInsetsDirectional:
		return v.Flipped()
	}
	p := g.edgeParts()
	return mixedEdgeInsets{p.right, p.left, p.end, p.start, p.bottom, p.top}
}

// EdgeInsetsGeometryLerp linearly interpolates between two EdgeInsetsGeometry
// objects. Either may be nil, which counts as zero; a nil result is returned for
// two nil values.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.lerp
func EdgeInsetsGeometryLerp(a, b EdgeInsetsGeometry, t float64) EdgeInsetsGeometry {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && a.Equal(b) && sameEdgeKind(a, b) {
		return a
	}
	if a == nil {
		return EdgeInsetsGeometryMul(b, t)
	}
	if b == nil {
		return EdgeInsetsGeometryMul(a, 1.0-t)
	}
	if x, ok := a.(EdgeInsets); ok {
		if y, ok := b.(EdgeInsets); ok {
			return *EdgeInsetsLerp(&x, &y, t)
		}
	}
	if x, ok := a.(EdgeInsetsDirectional); ok {
		if y, ok := b.(EdgeInsetsDirectional); ok {
			return *EdgeInsetsDirectionalLerp(&x, &y, t)
		}
	}
	x, y := a.edgeParts(), b.edgeParts()
	return mixedEdgeInsets{
		ui.LerpDouble(x.left, y.left, t), ui.LerpDouble(x.right, y.right, t),
		ui.LerpDouble(x.start, y.start, t), ui.LerpDouble(x.end, y.end, t),
		ui.LerpDouble(x.top, y.top, t), ui.LerpDouble(x.bottom, y.bottom, t)}
}

func sameEdgeKind(a, b EdgeInsetsGeometry) bool {
	return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
}

// EdgeInsets is an immutable set of offsets in each of the four cardinal
// directions. Create one with a literal, NewEdgeInsetsAll, NewEdgeInsetsSymmetric,
// NewEdgeInsetsFromLTRB or NewEdgeInsetsFromViewPadding.
//
// Flutter: painting/edge_insets.dart EdgeInsets
type EdgeInsets struct {
	// Left is the offset from the left.
	Left float64
	// Top is the offset from the top.
	Top float64
	// Right is the offset from the right.
	Right float64
	// Bottom is the offset from the bottom.
	Bottom float64
}

// EdgeInsetsZero is an EdgeInsets with zero offsets in each direction.
//
// Flutter: painting/edge_insets.dart EdgeInsets.zero
var EdgeInsetsZero = EdgeInsets{}

// NewEdgeInsetsFromLTRB creates insets from offsets from the left, top, right,
// and bottom.
//
// Flutter: painting/edge_insets.dart EdgeInsets.fromLTRB
func NewEdgeInsetsFromLTRB(left, top, right, bottom float64) EdgeInsets {
	return EdgeInsets{left, top, right, bottom}
}

// NewEdgeInsetsAll creates insets where all the offsets are value.
//
// Flutter: painting/edge_insets.dart EdgeInsets.all
func NewEdgeInsetsAll(value float64) EdgeInsets { return EdgeInsets{value, value, value, value} }

// NewEdgeInsetsSymmetric creates insets with symmetrical vertical and
// horizontal offsets.
//
// Flutter: painting/edge_insets.dart EdgeInsets.symmetric
func NewEdgeInsetsSymmetric(vertical, horizontal float64) EdgeInsets {
	return EdgeInsets{horizontal, vertical, horizontal, vertical}
}

// NewEdgeInsetsFromViewPadding creates insets that match the padding of a view,
// converted from physical to logical pixels.
//
// Flutter: painting/edge_insets.dart EdgeInsets.fromViewPadding
func NewEdgeInsetsFromViewPadding(padding ui.ViewPadding, devicePixelRatio float64) EdgeInsets {
	return EdgeInsets{padding.Left / devicePixelRatio, padding.Top / devicePixelRatio,
		padding.Right / devicePixelRatio, padding.Bottom / devicePixelRatio}
}

func (e EdgeInsets) edgeParts() edgeParts {
	return edgeParts{left: e.Left, right: e.Right, top: e.Top, bottom: e.Bottom}
}

// IsNonNegative reports whether every distance is zero or more.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.isNonNegative
func (e EdgeInsets) IsNonNegative() bool { return e.edgeParts().isNonNegative() }

// Horizontal is the total offset in the horizontal direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.horizontal
func (e EdgeInsets) Horizontal() float64 { return e.Left + e.Right }

// Vertical is the total offset in the vertical direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.vertical
func (e EdgeInsets) Vertical() float64 { return e.Top + e.Bottom }

// Along is the total offset along the axis.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.along
func (e EdgeInsets) Along(axis Axis) float64 { return alongAxis(e, axis) }

// CollapsedSize is a size that will contain the offsets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.collapsedSize
func (e EdgeInsets) CollapsedSize() ui.Size {
	return ui.Size{Width: e.Horizontal(), Height: e.Vertical()}
}

// InflateSize returns the size grown by the insets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.inflateSize
func (e EdgeInsets) InflateSize(s ui.Size) ui.Size { return inflateSize(e, s) }

// DeflateSize returns the size shrunk by the insets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.deflateSize
func (e EdgeInsets) DeflateSize(s ui.Size) ui.Size { return deflateSize(e, s) }

// TopLeft is an Offset describing the vector from the top left of a rectangle
// to the top left of that rectangle inset by this object.
//
// Flutter: painting/edge_insets.dart EdgeInsets.topLeft
func (e EdgeInsets) TopLeft() ui.Offset { return ui.Offset{Dx: e.Left, Dy: e.Top} }

// TopRight is the vector from the top right of a rectangle to the top right of
// that rectangle inset by this object.
//
// Flutter: painting/edge_insets.dart EdgeInsets.topRight
func (e EdgeInsets) TopRight() ui.Offset { return ui.Offset{Dx: -e.Right, Dy: e.Top} }

// BottomLeft is the vector from the bottom left of a rectangle to the bottom
// left of that rectangle inset by this object.
//
// Flutter: painting/edge_insets.dart EdgeInsets.bottomLeft
func (e EdgeInsets) BottomLeft() ui.Offset { return ui.Offset{Dx: e.Left, Dy: -e.Bottom} }

// BottomRight is the vector from the bottom right of a rectangle to the bottom
// right of that rectangle inset by this object.
//
// Flutter: painting/edge_insets.dart EdgeInsets.bottomRight
func (e EdgeInsets) BottomRight() ui.Offset { return ui.Offset{Dx: -e.Right, Dy: -e.Bottom} }

// Flipped is an EdgeInsets with top and bottom, left and right flipped.
//
// Flutter: painting/edge_insets.dart EdgeInsets.flipped
func (e EdgeInsets) Flipped() EdgeInsets { return EdgeInsets{e.Right, e.Bottom, e.Left, e.Top} }

// InflateRect returns a new rect that is bigger than the given rect in each
// direction by the amount of inset in each direction.
//
// Flutter: painting/edge_insets.dart EdgeInsets.inflateRect
func (e EdgeInsets) InflateRect(r ui.Rect) ui.Rect {
	return ui.NewRectFromLTRB(r.Left-e.Left, r.Top-e.Top, r.Right+e.Right, r.Bottom+e.Bottom)
}

// DeflateRect returns a new rect that is smaller than the given rect in each
// direction by the amount of inset in each direction.
//
// Flutter: painting/edge_insets.dart EdgeInsets.deflateRect
func (e EdgeInsets) DeflateRect(r ui.Rect) ui.Rect {
	return ui.NewRectFromLTRB(r.Left+e.Left, r.Top+e.Top, r.Right-e.Right, r.Bottom-e.Bottom)
}

// InflateRRect returns a rounded rect grown by the insets, whose radii grow too.
//
// Flutter: painting/edge_insets.dart EdgeInsets.inflateRRect
func (e EdgeInsets) InflateRRect(r ui.RRect) ui.RRect {
	zero := ui.Radius{}
	return ui.NewRRectFromLTRBAndCorners(r.Left-e.Left, r.Top-e.Top, r.Right+e.Right, r.Bottom+e.Bottom, ui.Corners{
		TopLeft:     r.TlRadius().Add(ui.NewRadiusElliptical(e.Left, e.Top)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		TopRight:    r.TrRadius().Add(ui.NewRadiusElliptical(e.Right, e.Top)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		BottomRight: r.BrRadius().Add(ui.NewRadiusElliptical(e.Right, e.Bottom)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		BottomLeft:  r.BlRadius().Add(ui.NewRadiusElliptical(e.Left, e.Bottom)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
	})
}

// DeflateRRect returns a rounded rect shrunk by the insets, whose radii shrink
// too.
//
// Flutter: painting/edge_insets.dart EdgeInsets.deflateRRect
func (e EdgeInsets) DeflateRRect(r ui.RRect) ui.RRect {
	zero := ui.Radius{}
	return ui.NewRRectFromLTRBAndCorners(r.Left+e.Left, r.Top+e.Top, r.Right-e.Right, r.Bottom-e.Bottom, ui.Corners{
		TopLeft:     r.TlRadius().Sub(ui.NewRadiusElliptical(e.Left, e.Top)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		TopRight:    r.TrRadius().Sub(ui.NewRadiusElliptical(e.Right, e.Top)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		BottomRight: r.BrRadius().Sub(ui.NewRadiusElliptical(e.Right, e.Bottom)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
		BottomLeft:  r.BlRadius().Sub(ui.NewRadiusElliptical(e.Left, e.Bottom)).Clamp(ui.RadiusClampOpts{Minimum: &zero}),
	})
}

// Subtract returns the result of subtracting the distances of other.
//
// Flutter: painting/edge_insets.dart EdgeInsets.subtract
func (e EdgeInsets) Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsets); ok {
		return e.Minus(o)
	}
	return subtractEdges(e, other)
}

// Add returns the result of adding the distances of other.
//
// Flutter: painting/edge_insets.dart EdgeInsets.add
func (e EdgeInsets) Add(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsets); ok {
		return e.Plus(o)
	}
	return addEdges(e, other)
}

// Clamp clamps each distance between those of min and max.
//
// Flutter: painting/edge_insets.dart EdgeInsets.clamp
func (e EdgeInsets) Clamp(lo, hi EdgeInsetsGeometry) EdgeInsetsGeometry {
	l, h := lo.edgeParts(), hi.edgeParts()
	return EdgeInsets{ui.ClampDouble(e.Left, l.left, h.left), ui.ClampDouble(e.Top, l.top, h.top),
		ui.ClampDouble(e.Right, l.right, h.right), ui.ClampDouble(e.Bottom, l.bottom, h.bottom)}
}

// Minus returns an EdgeInsets where the distances of other are subtracted from
// these.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator -
func (e EdgeInsets) Minus(o EdgeInsets) EdgeInsets {
	return EdgeInsets{e.Left - o.Left, e.Top - o.Top, e.Right - o.Right, e.Bottom - o.Bottom}
}

// Plus returns an EdgeInsets where the distances of other are added to these.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator +
func (e EdgeInsets) Plus(o EdgeInsets) EdgeInsets {
	return EdgeInsets{e.Left + o.Left, e.Top + o.Top, e.Right + o.Right, e.Bottom + o.Bottom}
}

// Neg returns the insets with every distance negated.
//
// Flutter: painting/edge_insets.dart EdgeInsets.unary-
func (e EdgeInsets) Neg() EdgeInsets { return EdgeInsets{-e.Left, -e.Top, -e.Right, -e.Bottom} }

// Mul scales the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator *
func (e EdgeInsets) Mul(k float64) EdgeInsets {
	return EdgeInsets{e.Left * k, e.Top * k, e.Right * k, e.Bottom * k}
}

// Div divides the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator /
func (e EdgeInsets) Div(k float64) EdgeInsets {
	return EdgeInsets{e.Left / k, e.Top / k, e.Right / k, e.Bottom / k}
}

// TruncDiv integer-divides the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator ~/
func (e EdgeInsets) TruncDiv(k float64) EdgeInsets {
	return EdgeInsets{truncDiv(e.Left, k), truncDiv(e.Top, k), truncDiv(e.Right, k), truncDiv(e.Bottom, k)}
}

// Mod takes the modulo k of the distances.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator %
func (e EdgeInsets) Mod(k float64) EdgeInsets {
	return EdgeInsets{dartMod(e.Left, k), dartMod(e.Top, k), dartMod(e.Right, k), dartMod(e.Bottom, k)}
}

// Resolve returns the insets themselves: they do not depend on the direction.
//
// Flutter: painting/edge_insets.dart EdgeInsets.resolve
func (e EdgeInsets) Resolve(ui.TextDirection) EdgeInsets { return e }

// CopyWith returns a copy with the given distances replaced.
//
// Flutter: painting/edge_insets.dart EdgeInsets.copyWith
func (e EdgeInsets) CopyWith(c EdgeInsetsCopy) EdgeInsets {
	return EdgeInsets{c.Left.OrElse(e.Left), c.Top.OrElse(e.Top), c.Right.OrElse(e.Right), c.Bottom.OrElse(e.Bottom)}
}

// Equal reports whether the six distances are equal.
//
// Flutter: painting/edge_insets.dart EdgeInsets.operator ==
func (e EdgeInsets) Equal(o EdgeInsetsGeometry) bool {
	return o != nil && e.edgeParts() == o.edgeParts()
}

// String describes the insets, such as "EdgeInsets.all(8.0)".
//
// Flutter: painting/edge_insets.dart EdgeInsets.toString
func (e EdgeInsets) String() string { return e.edgeParts().String() }

// EdgeInsetsLerp linearly interpolates between two EdgeInsets. Either may be
// nil, which counts as zero.
//
// Flutter: painting/edge_insets.dart EdgeInsets.lerp
func EdgeInsetsLerp(a, b *EdgeInsets, t float64) *EdgeInsets {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		r := b.Mul(t)
		return &r
	}
	if b == nil {
		r := a.Mul(1.0 - t)
		return &r
	}
	if *a == *b {
		return a
	}
	return &EdgeInsets{ui.LerpDouble(a.Left, b.Left, t), ui.LerpDouble(a.Top, b.Top, t),
		ui.LerpDouble(a.Right, b.Right, t), ui.LerpDouble(a.Bottom, b.Bottom, t)}
}

// EdgeInsetsCopy holds the distances EdgeInsets.CopyWith replaces; the ones
// that are None keep their value.
//
// Flutter: painting/edge_insets.dart EdgeInsets.copyWith
type EdgeInsetsCopy struct {
	Left, Top, Right, Bottom foundation.Opt[float64]
}

// EdgeInsetsDirectional is an immutable set of offsets in each of the four
// cardinal directions, but whose horizontal components are dependent on the
// writing direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional
type EdgeInsetsDirectional struct {
	// Start is the offset from the start side, the left in left-to-right text.
	Start float64
	// Top is the offset from the top.
	Top float64
	// End is the offset from the end side, the right in left-to-right text.
	End float64
	// Bottom is the offset from the bottom.
	Bottom float64
}

// EdgeInsetsDirectionalCopy holds the distances EdgeInsetsDirectional.CopyWith
// replaces; the ones that are None keep their value.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.copyWith
type EdgeInsetsDirectionalCopy struct {
	Start, Top, End, Bottom foundation.Opt[float64]
}

// EdgeInsetsDirectionalZero is an EdgeInsetsDirectional with zero offsets in
// each direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.zero
var EdgeInsetsDirectionalZero = EdgeInsetsDirectional{}

// NewEdgeInsetsDirectionalFromSTEB creates insets from offsets from the start,
// top, end, and bottom.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.fromSTEB
func NewEdgeInsetsDirectionalFromSTEB(start, top, end, bottom float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{start, top, end, bottom}
}

// NewEdgeInsetsDirectionalSymmetric creates insets with symmetrical vertical
// and horizontal offsets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.symmetric
func NewEdgeInsetsDirectionalSymmetric(horizontal, vertical float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{horizontal, vertical, horizontal, vertical}
}

// NewEdgeInsetsDirectionalAll creates insets where all the offsets are value.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.all
func NewEdgeInsetsDirectionalAll(value float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{value, value, value, value}
}

func (e EdgeInsetsDirectional) edgeParts() edgeParts {
	return edgeParts{start: e.Start, end: e.End, top: e.Top, bottom: e.Bottom}
}

// IsNonNegative reports whether every distance is zero or more.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.isNonNegative
func (e EdgeInsetsDirectional) IsNonNegative() bool { return e.edgeParts().isNonNegative() }

// Horizontal is the total offset in the horizontal direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.horizontal
func (e EdgeInsetsDirectional) Horizontal() float64 { return e.Start + e.End }

// Vertical is the total offset in the vertical direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.vertical
func (e EdgeInsetsDirectional) Vertical() float64 { return e.Top + e.Bottom }

// Along is the total offset along the axis.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.along
func (e EdgeInsetsDirectional) Along(axis Axis) float64 { return alongAxis(e, axis) }

// CollapsedSize is a size that will contain the offsets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.collapsedSize
func (e EdgeInsetsDirectional) CollapsedSize() ui.Size {
	return ui.Size{Width: e.Horizontal(), Height: e.Vertical()}
}

// InflateSize returns the size grown by the insets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.inflateSize
func (e EdgeInsetsDirectional) InflateSize(s ui.Size) ui.Size { return inflateSize(e, s) }

// DeflateSize returns the size shrunk by the insets.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.deflateSize
func (e EdgeInsetsDirectional) DeflateSize(s ui.Size) ui.Size { return deflateSize(e, s) }

// Flipped is an EdgeInsetsDirectional with top and bottom, start and end
// flipped.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.flipped
func (e EdgeInsetsDirectional) Flipped() EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.End, e.Bottom, e.Start, e.Top}
}

// Subtract returns the result of subtracting the distances of other.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.subtract
func (e EdgeInsetsDirectional) Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsetsDirectional); ok {
		return e.Minus(o)
	}
	return subtractEdges(e, other)
}

// Add returns the result of adding the distances of other.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.add
func (e EdgeInsetsDirectional) Add(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsetsDirectional); ok {
		return e.Plus(o)
	}
	return addEdges(e, other)
}

// Clamp clamps each distance between those of min and max.
//
// Flutter: painting/edge_insets.dart EdgeInsetsGeometry.clamp
func (e EdgeInsetsDirectional) Clamp(lo, hi EdgeInsetsGeometry) EdgeInsetsGeometry {
	return clampEdges(e, lo, hi)
}

// Minus returns the insets where the distances of other are subtracted.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator -
func (e EdgeInsetsDirectional) Minus(o EdgeInsetsDirectional) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start - o.Start, e.Top - o.Top, e.End - o.End, e.Bottom - o.Bottom}
}

// Plus returns the insets where the distances of other are added.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator +
func (e EdgeInsetsDirectional) Plus(o EdgeInsetsDirectional) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start + o.Start, e.Top + o.Top, e.End + o.End, e.Bottom + o.Bottom}
}

// Neg returns the insets with every distance negated.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.unary-
func (e EdgeInsetsDirectional) Neg() EdgeInsetsDirectional {
	return EdgeInsetsDirectional{-e.Start, -e.Top, -e.End, -e.Bottom}
}

// Mul scales the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator *
func (e EdgeInsetsDirectional) Mul(k float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start * k, e.Top * k, e.End * k, e.Bottom * k}
}

// Div divides the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator /
func (e EdgeInsetsDirectional) Div(k float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start / k, e.Top / k, e.End / k, e.Bottom / k}
}

// TruncDiv integer-divides the distances by k.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator ~/
func (e EdgeInsetsDirectional) TruncDiv(k float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{truncDiv(e.Start, k), truncDiv(e.Top, k), truncDiv(e.End, k), truncDiv(e.Bottom, k)}
}

// Mod takes the modulo k of the distances.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator %
func (e EdgeInsetsDirectional) Mod(k float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{dartMod(e.Start, k), dartMod(e.Top, k), dartMod(e.End, k), dartMod(e.Bottom, k)}
}

// Resolve returns an EdgeInsets for the text direction.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.resolve
func (e EdgeInsetsDirectional) Resolve(direction ui.TextDirection) EdgeInsets {
	debugCheckCanResolveTextDirection(direction, "EdgeInsetsDirectional")
	if direction == ui.TextDirectionRTL {
		return EdgeInsets{e.End, e.Top, e.Start, e.Bottom}
	}
	return EdgeInsets{e.Start, e.Top, e.End, e.Bottom}
}

// CopyWith returns a copy with the given distances replaced.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.copyWith
func (e EdgeInsetsDirectional) CopyWith(c EdgeInsetsDirectionalCopy) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{c.Start.OrElse(e.Start), c.Top.OrElse(e.Top), c.End.OrElse(e.End), c.Bottom.OrElse(e.Bottom)}
}

// Equal reports whether the six distances are equal.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.operator ==
func (e EdgeInsetsDirectional) Equal(o EdgeInsetsGeometry) bool {
	return o != nil && e.edgeParts() == o.edgeParts()
}

// String describes the insets, such as "EdgeInsetsDirectional(1.0, 2.0, 3.0, 4.0)".
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.toString
func (e EdgeInsetsDirectional) String() string { return e.edgeParts().String() }

// EdgeInsetsDirectionalLerp linearly interpolates between two
// EdgeInsetsDirectional. Either may be nil, which counts as zero.
//
// Flutter: painting/edge_insets.dart EdgeInsetsDirectional.lerp
func EdgeInsetsDirectionalLerp(a, b *EdgeInsetsDirectional, t float64) *EdgeInsetsDirectional {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		r := b.Mul(t)
		return &r
	}
	if b == nil {
		r := a.Mul(1.0 - t)
		return &r
	}
	if *a == *b {
		return a
	}
	return &EdgeInsetsDirectional{ui.LerpDouble(a.Start, b.Start, t), ui.LerpDouble(a.Top, b.Top, t),
		ui.LerpDouble(a.End, b.End, t), ui.LerpDouble(a.Bottom, b.Bottom, t)}
}
