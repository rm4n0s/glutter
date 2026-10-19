package painting

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/ui"
)

// radiusParts are the eight corner radii of a BorderRadiusGeometry: Flutter's
// protected getters _topLeft … _bottomEnd.
type radiusParts struct {
	topLeft, topRight, bottomLeft, bottomRight ui.Radius
	topStart, topEnd, bottomStart, bottomEnd   ui.Radius
}

func (p radiusParts) String() string {
	var visual, logical string
	if p.topLeft == p.topRight && p.topRight == p.bottomLeft && p.bottomLeft == p.bottomRight {
		if p.topLeft != (ui.Radius{}) {
			if p.topLeft.X == p.topLeft.Y {
				visual = "BorderRadius.circular(" + dartfmt.ToStringAsFixed(p.topLeft.X, 1) + ")"
			} else {
				visual = "BorderRadius.all(" + p.topLeft.String() + ")"
			}
		}
	} else {
		visual = "BorderRadius.only(" + describeCorners(
			[]string{"topLeft", "topRight", "bottomLeft", "bottomRight"},
			[]ui.Radius{p.topLeft, p.topRight, p.bottomLeft, p.bottomRight}) + ")"
	}
	if p.topStart == p.topEnd && p.topEnd == p.bottomEnd && p.bottomEnd == p.bottomStart {
		if p.topStart != (ui.Radius{}) {
			if p.topStart.X == p.topStart.Y {
				logical = "BorderRadiusDirectional.circular(" + dartfmt.ToStringAsFixed(p.topStart.X, 1) + ")"
			} else {
				logical = "BorderRadiusDirectional.all(" + p.topStart.String() + ")"
			}
		}
	} else {
		logical = "BorderRadiusDirectional.only(" + describeCorners(
			[]string{"topStart", "topEnd", "bottomStart", "bottomEnd"},
			[]ui.Radius{p.topStart, p.topEnd, p.bottomStart, p.bottomEnd}) + ")"
	}
	switch {
	case visual != "" && logical != "":
		return visual + " + " + logical
	case visual != "":
		return visual
	case logical != "":
		return logical
	}
	return "BorderRadius.zero"
}

func describeCorners(names []string, radii []ui.Radius) string {
	var parts []string
	for i, r := range radii {
		if r != (ui.Radius{}) {
			parts = append(parts, names[i]+": "+r.String())
		}
	}
	return strings.Join(parts, ", ")
}

// BorderRadiusGeometry is a base class for BorderRadius and
// BorderRadiusDirectional, which describe the corners of a box. Its covariant
// operations (Neg, Mul, Div, TruncDiv and Mod) are the functions
// BorderRadiusGeometryNeg, … and the methods of the concrete types.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry
type BorderRadiusGeometry interface {
	// Subtract returns the result of subtracting the radii of other.
	Subtract(other BorderRadiusGeometry) BorderRadiusGeometry
	// Add returns the result of adding the radii of other.
	Add(other BorderRadiusGeometry) BorderRadiusGeometry
	// Resolve returns a BorderRadius for the text direction. A directional value
	// panics with the tag TextDirectionRequired in debug builds if the direction
	// is unspecified.
	Resolve(direction ui.TextDirection) BorderRadius
	// Equal reports whether the eight radii are equal and the types match.
	Equal(other BorderRadiusGeometry) bool
	String() string
	radiusParts() radiusParts
}

// BorderRadiusGeometryZero is the BorderRadiusGeometry without any rounding.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.zero
var BorderRadiusGeometryZero BorderRadiusGeometry = BorderRadius{}

type mixedBorderRadius radiusParts

func (m mixedBorderRadius) radiusParts() radiusParts { return radiusParts(m) }
func (m mixedBorderRadius) Subtract(o BorderRadiusGeometry) BorderRadiusGeometry {
	return subtractRadii(m, o)
}
func (m mixedBorderRadius) Add(o BorderRadiusGeometry) BorderRadiusGeometry { return addRadii(m, o) }
func (m mixedBorderRadius) Equal(o BorderRadiusGeometry) bool {
	_, same := o.(mixedBorderRadius)
	return same && radiusParts(m) == o.radiusParts()
}
func (m mixedBorderRadius) String() string { return radiusParts(m).String() }

// Resolve implements BorderRadiusGeometry.
//
// Flutter: painting/border_radius.dart _MixedBorderRadius.resolve
func (m mixedBorderRadius) Resolve(direction ui.TextDirection) BorderRadius {
	debugCheckCanResolveTextDirection(direction, "_MixedBorderRadius")
	if direction == ui.TextDirectionRTL {
		return BorderRadius{TopLeft: m.topLeft.Add(m.topEnd), TopRight: m.topRight.Add(m.topStart),
			BottomLeft: m.bottomLeft.Add(m.bottomEnd), BottomRight: m.bottomRight.Add(m.bottomStart)}
	}
	return BorderRadius{TopLeft: m.topLeft.Add(m.topStart), TopRight: m.topRight.Add(m.topEnd),
		BottomLeft: m.bottomLeft.Add(m.bottomStart), BottomRight: m.bottomRight.Add(m.bottomEnd)}
}

func zipRadii(a, b radiusParts, f func(x, y ui.Radius) ui.Radius) mixedBorderRadius {
	return mixedBorderRadius{
		f(a.topLeft, b.topLeft), f(a.topRight, b.topRight), f(a.bottomLeft, b.bottomLeft), f(a.bottomRight, b.bottomRight),
		f(a.topStart, b.topStart), f(a.topEnd, b.topEnd), f(a.bottomStart, b.bottomStart), f(a.bottomEnd, b.bottomEnd)}
}

func subtractRadii(a, b BorderRadiusGeometry) BorderRadiusGeometry {
	return zipRadii(a.radiusParts(), b.radiusParts(), func(x, y ui.Radius) ui.Radius { return x.Sub(y) })
}

func addRadii(a, b BorderRadiusGeometry) BorderRadiusGeometry {
	return zipRadii(a.radiusParts(), b.radiusParts(), func(x, y ui.Radius) ui.Radius { return x.Add(y) })
}

func mapRadii(g BorderRadiusGeometry, f func(ui.Radius) ui.Radius) BorderRadiusGeometry {
	switch v := g.(type) {
	case BorderRadius:
		return BorderRadius{f(v.TopLeft), f(v.TopRight), f(v.BottomLeft), f(v.BottomRight)}
	case BorderRadiusDirectional:
		return BorderRadiusDirectional{f(v.TopStart), f(v.TopEnd), f(v.BottomStart), f(v.BottomEnd)}
	}
	p := g.radiusParts()
	return mixedBorderRadius{f(p.topLeft), f(p.topRight), f(p.bottomLeft), f(p.bottomRight),
		f(p.topStart), f(p.topEnd), f(p.bottomStart), f(p.bottomEnd)}
}

// BorderRadiusGeometryNeg negates every radius.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.operator -
func BorderRadiusGeometryNeg(g BorderRadiusGeometry) BorderRadiusGeometry {
	return mapRadii(g, func(r ui.Radius) ui.Radius { return r.Neg() })
}

// BorderRadiusGeometryMul multiplies every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.operator *
func BorderRadiusGeometryMul(g BorderRadiusGeometry, k float64) BorderRadiusGeometry {
	return mapRadii(g, func(r ui.Radius) ui.Radius { return r.Mul(k) })
}

// BorderRadiusGeometryDiv divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.operator /
func BorderRadiusGeometryDiv(g BorderRadiusGeometry, k float64) BorderRadiusGeometry {
	return mapRadii(g, func(r ui.Radius) ui.Radius { return r.Div(k) })
}

// BorderRadiusGeometryTruncDiv integer-divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.operator ~/
func BorderRadiusGeometryTruncDiv(g BorderRadiusGeometry, k float64) BorderRadiusGeometry {
	return mapRadii(g, func(r ui.Radius) ui.Radius { return r.TruncDiv(k) })
}

// BorderRadiusGeometryMod takes the modulo k of every radius.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.operator %
func BorderRadiusGeometryMod(g BorderRadiusGeometry, k float64) BorderRadiusGeometry {
	return mapRadii(g, func(r ui.Radius) ui.Radius { return r.Mod(k) })
}

// BorderRadiusGeometryLerp linearly interpolates between two
// BorderRadiusGeometry objects. Either may be nil, which counts as zero.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.lerp
func BorderRadiusGeometryLerp(a, b BorderRadiusGeometry, t float64) BorderRadiusGeometry {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && a.Equal(b) {
		return a
	}
	if a == nil {
		a = BorderRadius{}
	}
	if b == nil {
		b = BorderRadius{}
	}
	return a.Add(BorderRadiusGeometryMul(b.Subtract(a), t))
}

// BorderRadiusHorizontalGeometry creates a border radius with the left and right
// radii (which make a BorderRadius), or the start and end radii (which make a
// BorderRadiusDirectional). Mixing them panics with the tag
// BorderRadiusMixedHorizontal in debug builds.
//
// Flutter: painting/border_radius.dart BorderRadiusGeometry.horizontal
func BorderRadiusHorizontalGeometry(left, right, start, end foundation.Opt[ui.Radius]) BorderRadiusGeometry {
	if foundation.DebugMode && (!left.IsNone() || !right.IsNone()) && (!start.IsNone() || !end.IsNone()) {
		panic(errors.New("BorderRadiusMixedHorizontal",
			"the left and right values cannot be used in conjunction with start and end"))
	}
	if start.IsSome() || end.IsSome() {
		return NewBorderRadiusDirectionalHorizontal(start.Value(), end.Value())
	}
	return NewBorderRadiusHorizontal(left.Value(), right.Value())
}

// BorderRadius is an immutable set of radii for each corner of a rectangle.
// Create one with a literal, NewBorderRadiusAll, NewBorderRadiusCircular,
// NewBorderRadiusVertical or NewBorderRadiusHorizontal.
//
// Flutter: painting/border_radius.dart BorderRadius
type BorderRadius struct {
	// TopLeft is the top-left [ui.Radius].
	TopLeft ui.Radius
	// TopRight is the top-right [ui.Radius].
	TopRight ui.Radius
	// BottomLeft is the bottom-left [ui.Radius].
	BottomLeft ui.Radius
	// BottomRight is the bottom-right [ui.Radius].
	BottomRight ui.Radius
}

// BorderRadiusZero is a border radius with no rounding.
//
// Flutter: painting/border_radius.dart BorderRadius.zero
var BorderRadiusZero = BorderRadius{}

// NewBorderRadiusAll creates a border radius where all the corners have the
// same radius.
//
// Flutter: painting/border_radius.dart BorderRadius.all
func NewBorderRadiusAll(radius ui.Radius) BorderRadius {
	return BorderRadius{radius, radius, radius, radius}
}

// NewBorderRadiusCircular creates a border radius where all the corners are
// circular with the same radius.
//
// Flutter: painting/border_radius.dart BorderRadius.circular
func NewBorderRadiusCircular(radius float64) BorderRadius {
	return NewBorderRadiusAll(ui.NewRadiusCircular(radius))
}

// NewBorderRadiusVertical creates a vertically symmetric border radius.
//
// Flutter: painting/border_radius.dart BorderRadius.vertical
func NewBorderRadiusVertical(top, bottom ui.Radius) BorderRadius {
	return BorderRadius{top, top, bottom, bottom}
}

// NewBorderRadiusHorizontal creates a horizontally symmetric border radius.
//
// Flutter: painting/border_radius.dart BorderRadius.horizontal
func NewBorderRadiusHorizontal(left, right ui.Radius) BorderRadius {
	return BorderRadius{left, right, left, right}
}

func (b BorderRadius) radiusParts() radiusParts {
	return radiusParts{topLeft: b.TopLeft, topRight: b.TopRight, bottomLeft: b.BottomLeft, bottomRight: b.BottomRight}
}

// CopyWith returns a copy with the given radii replaced.
//
// Flutter: painting/border_radius.dart BorderRadius.copyWith
func (b BorderRadius) CopyWith(c BorderRadiusCopy) BorderRadius {
	return BorderRadius{c.TopLeft.OrElse(b.TopLeft), c.TopRight.OrElse(b.TopRight),
		c.BottomLeft.OrElse(b.BottomLeft), c.BottomRight.OrElse(b.BottomRight)}
}

// BorderRadiusCopy holds the radii BorderRadius.CopyWith replaces.
//
// Flutter: painting/border_radius.dart BorderRadius.copyWith
type BorderRadiusCopy struct {
	TopLeft, TopRight, BottomLeft, BottomRight foundation.Opt[ui.Radius]
}

func (b BorderRadius) corners() ui.Corners {
	zero := ui.Radius{}
	opts := ui.RadiusClampOpts{Minimum: &zero}
	return ui.Corners{TopLeft: b.TopLeft.Clamp(opts), TopRight: b.TopRight.Clamp(opts),
		BottomLeft: b.BottomLeft.Clamp(opts), BottomRight: b.BottomRight.Clamp(opts)}
}

// ToRRect creates a rounded rectangle from the rectangle and these radii, with
// negative radii clamped to zero.
//
// Flutter: painting/border_radius.dart BorderRadius.toRRect
func (b BorderRadius) ToRRect(rect ui.Rect) ui.RRect {
	return ui.NewRRectFromRectAndCorners(rect, b.corners())
}

// ToRSuperellipse creates a rounded superellipse from the rectangle and these
// radii, with negative radii clamped to zero.
//
// Flutter: painting/border_radius.dart BorderRadius.toRSuperellipse
func (b BorderRadius) ToRSuperellipse(rect ui.Rect) ui.RSuperellipse {
	return ui.NewRSuperellipseFromRectAndCorners(rect, b.corners())
}

// Subtract returns the result of subtracting the radii of other.
//
// Flutter: painting/border_radius.dart BorderRadius.subtract
func (b BorderRadius) Subtract(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadius); ok {
		return b.Minus(o)
	}
	return subtractRadii(b, other)
}

// Add returns the result of adding the radii of other.
//
// Flutter: painting/border_radius.dart BorderRadius.add
func (b BorderRadius) Add(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadius); ok {
		return b.Plus(o)
	}
	return addRadii(b, other)
}

// Minus returns the radii of b less those of o.
//
// Flutter: painting/border_radius.dart BorderRadius.operator -
func (b BorderRadius) Minus(o BorderRadius) BorderRadius {
	return BorderRadius{b.TopLeft.Sub(o.TopLeft), b.TopRight.Sub(o.TopRight), b.BottomLeft.Sub(o.BottomLeft), b.BottomRight.Sub(o.BottomRight)}
}

// Plus returns the radii of b plus those of o.
//
// Flutter: painting/border_radius.dart BorderRadius.operator +
func (b BorderRadius) Plus(o BorderRadius) BorderRadius {
	return BorderRadius{b.TopLeft.Add(o.TopLeft), b.TopRight.Add(o.TopRight), b.BottomLeft.Add(o.BottomLeft), b.BottomRight.Add(o.BottomRight)}
}

func (b BorderRadius) mapAll(f func(ui.Radius) ui.Radius) BorderRadius {
	return BorderRadius{f(b.TopLeft), f(b.TopRight), f(b.BottomLeft), f(b.BottomRight)}
}

// Neg negates every radius.
//
// Flutter: painting/border_radius.dart BorderRadius.unary-
func (b BorderRadius) Neg() BorderRadius { return b.mapAll(ui.Radius.Neg) }

// Mul scales every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadius.operator *
func (b BorderRadius) Mul(k float64) BorderRadius {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Mul(k) })
}

// Div divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadius.operator /
func (b BorderRadius) Div(k float64) BorderRadius {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Div(k) })
}

// TruncDiv integer-divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadius.operator ~/
func (b BorderRadius) TruncDiv(k float64) BorderRadius {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.TruncDiv(k) })
}

// Mod takes the modulo k of every radius.
//
// Flutter: painting/border_radius.dart BorderRadius.operator %
func (b BorderRadius) Mod(k float64) BorderRadius {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Mod(k) })
}

// Resolve returns the radii themselves: they do not depend on the direction.
//
// Flutter: painting/border_radius.dart BorderRadius.resolve
func (b BorderRadius) Resolve(ui.TextDirection) BorderRadius { return b }

// Equal reports whether the radii are equal and o is a BorderRadius.
//
// Flutter: painting/border_radius.dart BorderRadius.operator ==
func (b BorderRadius) Equal(o BorderRadiusGeometry) bool {
	x, ok := o.(BorderRadius)
	return ok && x == b
}

// String describes the border radius, such as "BorderRadius.circular(8.0)".
//
// Flutter: painting/border_radius.dart BorderRadius.toString
func (b BorderRadius) String() string { return b.radiusParts().String() }

// BorderRadiusLerp linearly interpolates between two BorderRadius. Either may be
// nil, which counts as zero.
//
// Flutter: painting/border_radius.dart BorderRadius.lerp
func BorderRadiusLerp(a, b *BorderRadius, t float64) *BorderRadius {
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
	return &BorderRadius{a.TopLeft.Lerp(b.TopLeft, t), a.TopRight.Lerp(b.TopRight, t),
		a.BottomLeft.Lerp(b.BottomLeft, t), a.BottomRight.Lerp(b.BottomRight, t)}
}

// BorderRadiusDirectional is an immutable set of radii for each corner of a
// rectangle, but with the corners on the left and right depending on the
// writing direction.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional
type BorderRadiusDirectional struct {
	// TopStart is the top-start [ui.Radius].
	TopStart ui.Radius
	// TopEnd is the top-end [ui.Radius].
	TopEnd ui.Radius
	// BottomStart is the bottom-start [ui.Radius].
	BottomStart ui.Radius
	// BottomEnd is the bottom-end [ui.Radius].
	BottomEnd ui.Radius
}

// BorderRadiusDirectionalZero is a border radius with no rounding.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.zero
var BorderRadiusDirectionalZero = BorderRadiusDirectional{}

// NewBorderRadiusDirectionalAll creates a border radius where all the corners
// have the same radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.all
func NewBorderRadiusDirectionalAll(radius ui.Radius) BorderRadiusDirectional {
	return BorderRadiusDirectional{radius, radius, radius, radius}
}

// NewBorderRadiusDirectionalCircular creates a border radius where all the
// corners are circular with the same radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.circular
func NewBorderRadiusDirectionalCircular(radius float64) BorderRadiusDirectional {
	return NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(radius))
}

// NewBorderRadiusDirectionalVertical creates a vertically symmetric border
// radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.vertical
func NewBorderRadiusDirectionalVertical(top, bottom ui.Radius) BorderRadiusDirectional {
	return BorderRadiusDirectional{top, top, bottom, bottom}
}

// NewBorderRadiusDirectionalHorizontal creates a horizontally symmetric border
// radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.horizontal
func NewBorderRadiusDirectionalHorizontal(start, end ui.Radius) BorderRadiusDirectional {
	return BorderRadiusDirectional{start, end, start, end}
}

func (b BorderRadiusDirectional) radiusParts() radiusParts {
	return radiusParts{topStart: b.TopStart, topEnd: b.TopEnd, bottomStart: b.BottomStart, bottomEnd: b.BottomEnd}
}

// Subtract returns the result of subtracting the radii of other.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.subtract
func (b BorderRadiusDirectional) Subtract(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadiusDirectional); ok {
		return b.Minus(o)
	}
	return subtractRadii(b, other)
}

// Add returns the result of adding the radii of other.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.add
func (b BorderRadiusDirectional) Add(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadiusDirectional); ok {
		return b.Plus(o)
	}
	return addRadii(b, other)
}

// Minus returns the radii of b less those of o.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator -
func (b BorderRadiusDirectional) Minus(o BorderRadiusDirectional) BorderRadiusDirectional {
	return BorderRadiusDirectional{b.TopStart.Sub(o.TopStart), b.TopEnd.Sub(o.TopEnd), b.BottomStart.Sub(o.BottomStart), b.BottomEnd.Sub(o.BottomEnd)}
}

// Plus returns the radii of b plus those of o.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator +
func (b BorderRadiusDirectional) Plus(o BorderRadiusDirectional) BorderRadiusDirectional {
	return BorderRadiusDirectional{b.TopStart.Add(o.TopStart), b.TopEnd.Add(o.TopEnd), b.BottomStart.Add(o.BottomStart), b.BottomEnd.Add(o.BottomEnd)}
}

func (b BorderRadiusDirectional) mapAll(f func(ui.Radius) ui.Radius) BorderRadiusDirectional {
	return BorderRadiusDirectional{f(b.TopStart), f(b.TopEnd), f(b.BottomStart), f(b.BottomEnd)}
}

// Neg negates every radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.unary-
func (b BorderRadiusDirectional) Neg() BorderRadiusDirectional { return b.mapAll(ui.Radius.Neg) }

// Mul scales every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator *
func (b BorderRadiusDirectional) Mul(k float64) BorderRadiusDirectional {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Mul(k) })
}

// Div divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator /
func (b BorderRadiusDirectional) Div(k float64) BorderRadiusDirectional {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Div(k) })
}

// TruncDiv integer-divides every radius by k.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator ~/
func (b BorderRadiusDirectional) TruncDiv(k float64) BorderRadiusDirectional {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.TruncDiv(k) })
}

// Mod takes the modulo k of every radius.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator %
func (b BorderRadiusDirectional) Mod(k float64) BorderRadiusDirectional {
	return b.mapAll(func(r ui.Radius) ui.Radius { return r.Mod(k) })
}

// Resolve returns a BorderRadius for the text direction.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.resolve
func (b BorderRadiusDirectional) Resolve(direction ui.TextDirection) BorderRadius {
	debugCheckCanResolveTextDirection(direction, "BorderRadiusDirectional")
	if direction == ui.TextDirectionRTL {
		return BorderRadius{b.TopEnd, b.TopStart, b.BottomEnd, b.BottomStart}
	}
	return BorderRadius{b.TopStart, b.TopEnd, b.BottomStart, b.BottomEnd}
}

// Equal reports whether the radii are equal and o is a BorderRadiusDirectional.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.operator ==
func (b BorderRadiusDirectional) Equal(o BorderRadiusGeometry) bool {
	x, ok := o.(BorderRadiusDirectional)
	return ok && x == b
}

// String describes the border radius, such as "BorderRadiusDirectional.circular(8.0)".
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.toString
func (b BorderRadiusDirectional) String() string { return b.radiusParts().String() }

// BorderRadiusDirectionalLerp linearly interpolates between two
// BorderRadiusDirectional. Either may be nil, which counts as zero.
//
// Flutter: painting/border_radius.dart BorderRadiusDirectional.lerp
func BorderRadiusDirectionalLerp(a, b *BorderRadiusDirectional, t float64) *BorderRadiusDirectional {
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
	return &BorderRadiusDirectional{a.TopStart.Lerp(b.TopStart, t), a.TopEnd.Lerp(b.TopEnd, t),
		a.BottomStart.Lerp(b.BottomStart, t), a.BottomEnd.Lerp(b.BottomEnd, t)}
}

var (
	_ EdgeInsetsGeometry   = EdgeInsets{}
	_ EdgeInsetsGeometry   = EdgeInsetsDirectional{}
	_ AlignmentGeometry    = Alignment{}
	_ AlignmentGeometry    = AlignmentDirectional{}
	_ BorderRadiusGeometry = BorderRadius{}
	_ BorderRadiusGeometry = BorderRadiusDirectional{}
	_ fmt.Stringer         = BorderRadius{}
)
