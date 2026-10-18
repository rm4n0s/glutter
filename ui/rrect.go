package ui

import (
	"fmt"
	"math"
)

// rrectLike holds what RRect and RSuperellipse share: a rectangle with an
// elliptical radius on each corner. Both types are defined on it, so that their
// fields are the same and the shared algorithms live in one place.
//
// Flutter: ui/geometry.dart _RRectLike
type rrectLike struct {
	// Left is the offset of the left edge from the x axis.
	Left float64
	// Top is the offset of the top edge from the y axis.
	Top float64
	// Right is the offset of the right edge from the x axis.
	Right float64
	// Bottom is the offset of the bottom edge from the y axis.
	Bottom float64
	// TlRadiusX is the top left corner's radius on the x axis.
	TlRadiusX float64
	// TlRadiusY is the top left corner's radius on the y axis.
	TlRadiusY float64
	// TrRadiusX is the top right corner's radius on the x axis.
	TrRadiusX float64
	// TrRadiusY is the top right corner's radius on the y axis.
	TrRadiusY float64
	// BrRadiusX is the bottom right corner's radius on the x axis.
	BrRadiusX float64
	// BrRadiusY is the bottom right corner's radius on the y axis.
	BrRadiusY float64
	// BlRadiusX is the bottom left corner's radius on the x axis.
	BlRadiusX float64
	// BlRadiusY is the bottom left corner's radius on the y axis.
	BlRadiusY float64
}

// Corners holds the radius of each corner, for the constructors that take the
// corners separately. A zero Radius is a square corner.
//
// Flutter: ui/geometry.dart RRect.fromLTRBAndCorners
type Corners struct {
	// TopLeft is the radius of the top left corner.
	TopLeft Radius
	// TopRight is the radius of the top right corner.
	TopRight Radius
	// BottomRight is the radius of the bottom right corner.
	BottomRight Radius
	// BottomLeft is the radius of the bottom left corner.
	BottomLeft Radius
}

func newRRectLikeLTRBXY(left, top, right, bottom, radiusX, radiusY float64) rrectLike {
	return rrectLike{left, top, right, bottom, radiusX, radiusY, radiusX, radiusY, radiusX, radiusY, radiusX, radiusY}
}

func newRRectLikeCorners(left, top, right, bottom float64, c Corners) rrectLike {
	return rrectLike{
		Left: left, Top: top, Right: right, Bottom: bottom,
		TlRadiusX: c.TopLeft.X, TlRadiusY: c.TopLeft.Y,
		TrRadiusX: c.TopRight.X, TrRadiusY: c.TopRight.Y,
		BrRadiusX: c.BottomRight.X, BrRadiusY: c.BottomRight.Y,
		BlRadiusX: c.BottomLeft.X, BlRadiusY: c.BottomLeft.Y,
	}
}

func (r rrectLike) tlRadius() Radius { return Radius{r.TlRadiusX, r.TlRadiusY} }
func (r rrectLike) trRadius() Radius { return Radius{r.TrRadiusX, r.TrRadiusY} }
func (r rrectLike) brRadius() Radius { return Radius{r.BrRadiusX, r.BrRadiusY} }
func (r rrectLike) blRadius() Radius { return Radius{r.BlRadiusX, r.BlRadiusY} }

func (r rrectLike) shift(offset Offset) rrectLike {
	r.Left += offset.Dx
	r.Top += offset.Dy
	r.Right += offset.Dx
	r.Bottom += offset.Dy
	return r
}

func (r rrectLike) inflate(delta float64) rrectLike {
	return rrectLike{
		r.Left - delta, r.Top - delta, r.Right + delta, r.Bottom + delta,
		math.Max(0, r.TlRadiusX+delta), math.Max(0, r.TlRadiusY+delta),
		math.Max(0, r.TrRadiusX+delta), math.Max(0, r.TrRadiusY+delta),
		math.Max(0, r.BrRadiusX+delta), math.Max(0, r.BrRadiusY+delta),
		math.Max(0, r.BlRadiusX+delta), math.Max(0, r.BlRadiusY+delta),
	}
}

func (r rrectLike) width() float64  { return r.Right - r.Left }
func (r rrectLike) height() float64 { return r.Bottom - r.Top }

func (r rrectLike) outerRect() Rect { return Rect{r.Left, r.Top, r.Right, r.Bottom} }

func (r rrectLike) safeInnerRect() Rect {
	const kInsetFactor = 0.29289321881 // 1-cos(pi/4)
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{
		r.Left + leftRadius*kInsetFactor,
		r.Top + topRadius*kInsetFactor,
		r.Right - rightRadius*kInsetFactor,
		r.Bottom - bottomRadius*kInsetFactor,
	}
}

func (r rrectLike) middleRect() Rect {
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{r.Left + leftRadius, r.Top + topRadius, r.Right - rightRadius, r.Bottom - bottomRadius}
}

func (r rrectLike) wideMiddleRect() Rect {
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{r.Left, r.Top + topRadius, r.Right, r.Bottom - bottomRadius}
}

func (r rrectLike) tallMiddleRect() Rect {
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	return Rect{r.Left + leftRadius, r.Top, r.Right - rightRadius, r.Bottom}
}

func (r rrectLike) isEmpty() bool { return r.Left >= r.Right || r.Top >= r.Bottom }

func (r rrectLike) isFinite() bool {
	return isFinite(r.Left) && isFinite(r.Top) && isFinite(r.Right) && isFinite(r.Bottom)
}

func (r rrectLike) isRect() bool {
	return (r.TlRadiusX == 0.0 || r.TlRadiusY == 0.0) &&
		(r.TrRadiusX == 0.0 || r.TrRadiusY == 0.0) &&
		(r.BlRadiusX == 0.0 || r.BlRadiusY == 0.0) &&
		(r.BrRadiusX == 0.0 || r.BrRadiusY == 0.0)
}

func (r rrectLike) isStadium() bool {
	return r.tlRadius() == r.trRadius() &&
		r.trRadius() == r.brRadius() &&
		r.brRadius() == r.blRadius() &&
		(r.width() <= 2.0*r.TlRadiusX || r.height() <= 2.0*r.TlRadiusY)
}

func (r rrectLike) isEllipse() bool {
	return r.tlRadius() == r.trRadius() &&
		r.trRadius() == r.brRadius() &&
		r.brRadius() == r.blRadius() &&
		r.width() <= 2.0*r.TlRadiusX &&
		r.height() <= 2.0*r.TlRadiusY
}

func (r rrectLike) isCircle() bool { return r.width() == r.height() && r.isEllipse() }

func (r rrectLike) shortestSide() float64 { return math.Min(math.Abs(r.width()), math.Abs(r.height())) }
func (r rrectLike) longestSide() float64  { return math.Max(math.Abs(r.width()), math.Abs(r.height())) }

func (r rrectLike) hasNaN() bool {
	return math.IsNaN(r.Left) || math.IsNaN(r.Top) || math.IsNaN(r.Right) || math.IsNaN(r.Bottom) ||
		math.IsNaN(r.TrRadiusX) || math.IsNaN(r.TrRadiusY) || math.IsNaN(r.TlRadiusX) || math.IsNaN(r.TlRadiusY) ||
		math.IsNaN(r.BrRadiusX) || math.IsNaN(r.BrRadiusY) || math.IsNaN(r.BlRadiusX) || math.IsNaN(r.BlRadiusY)
}

func (r rrectLike) center() Offset { return Offset{r.Left + r.width()/2.0, r.Top + r.height()/2.0} }

// getMin returns the minimum between min and the scale to which radius1 and
// radius2 should be scaled in order not to exceed the limit.
func getMin(min, radius1, radius2, limit float64) float64 {
	sum := radius1 + radius2
	if sum > limit && sum != 0.0 {
		return math.Min(min, limit/sum)
	}
	return min
}

func (r rrectLike) scaleRadii() rrectLike {
	scale := 1.0
	scale = getMin(scale, r.BlRadiusY, r.TlRadiusY, r.height())
	scale = getMin(scale, r.TlRadiusX, r.TrRadiusX, r.width())
	scale = getMin(scale, r.TrRadiusY, r.BrRadiusY, r.height())
	scale = getMin(scale, r.BrRadiusX, r.BlRadiusX, r.width())
	if scale < 1.0 {
		r.TlRadiusX *= scale
		r.TlRadiusY *= scale
		r.TrRadiusX *= scale
		r.TrRadiusY *= scale
		r.BlRadiusX *= scale
		r.BlRadiusY *= scale
		r.BrRadiusX *= scale
		r.BrRadiusY *= scale
	}
	return r
}

// lerpTo interpolates between r and b, or between r and nothing when b is nil.
func (r rrectLike) lerpTo(b *rrectLike, t float64) rrectLike {
	if b == nil {
		k := 1.0 - t
		return rrectLike{
			r.Left * k, r.Top * k, r.Right * k, r.Bottom * k,
			math.Max(0, r.TlRadiusX*k), math.Max(0, r.TlRadiusY*k),
			math.Max(0, r.TrRadiusX*k), math.Max(0, r.TrRadiusY*k),
			math.Max(0, r.BrRadiusX*k), math.Max(0, r.BrRadiusY*k),
			math.Max(0, r.BlRadiusX*k), math.Max(0, r.BlRadiusY*k),
		}
	}
	return rrectLike{
		lerpDouble(r.Left, b.Left, t), lerpDouble(r.Top, b.Top, t),
		lerpDouble(r.Right, b.Right, t), lerpDouble(r.Bottom, b.Bottom, t),
		math.Max(0, lerpDouble(r.TlRadiusX, b.TlRadiusX, t)), math.Max(0, lerpDouble(r.TlRadiusY, b.TlRadiusY, t)),
		math.Max(0, lerpDouble(r.TrRadiusX, b.TrRadiusX, t)), math.Max(0, lerpDouble(r.TrRadiusY, b.TrRadiusY, t)),
		math.Max(0, lerpDouble(r.BrRadiusX, b.BrRadiusX, t)), math.Max(0, lerpDouble(r.BrRadiusY, b.BrRadiusY, t)),
		math.Max(0, lerpDouble(r.BlRadiusX, b.BlRadiusX, t)), math.Max(0, lerpDouble(r.BlRadiusY, b.BlRadiusY, t)),
	}
}

// lerpLike is the static lerp of RRect and RSuperellipse: nil counts as an empty
// shape at the origin.
func lerpLike(a, b *rrectLike, t float64) *rrectLike {
	if a == nil {
		if b == nil {
			return nil
		}
		r := b.lerpTo(nil, 1-t)
		return &r
	}
	r := a.lerpTo(b, t)
	return &r
}

// contains reports whether the point is inside the shape, whose corners are the
// superellipse |x|^n + |y|^n = 1 inside a box of (rx*extent) by (ry*extent), as
// far as the box fits in the shape. With extent 1 and n 2 the corners are the
// ellipses of an RRect.
func (r rrectLike) contains(point Offset, extent, n float64, clamp bool) bool {
	if point.Dx < r.Left || point.Dx >= r.Right || point.Dy < r.Top || point.Dy >= r.Bottom {
		return false // outside bounding box
	}
	scaled := r.scaleRadii()
	halfW, halfH := r.width()/2, r.height()/2
	box := func(rx, ry float64) (float64, float64) {
		rx, ry = rx*extent, ry*extent
		if clamp {
			rx, ry = math.Min(rx, halfW), math.Min(ry, halfH)
		}
		return rx, ry
	}
	var x, y, radiusX, radiusY float64
	// check whether point is in one of the rounded corner areas
	// x, y -> translate to ellipse center
	if rx, ry := box(scaled.TlRadiusX, scaled.TlRadiusY); point.Dx < r.Left+rx && point.Dy < r.Top+ry {
		x, y, radiusX, radiusY = point.Dx-r.Left-rx, point.Dy-r.Top-ry, rx, ry
	} else if rx, ry := box(scaled.TrRadiusX, scaled.TrRadiusY); point.Dx > r.Right-rx && point.Dy < r.Top+ry {
		x, y, radiusX, radiusY = point.Dx-r.Right+rx, point.Dy-r.Top-ry, rx, ry
	} else if rx, ry := box(scaled.BrRadiusX, scaled.BrRadiusY); point.Dx > r.Right-rx && point.Dy > r.Bottom-ry {
		x, y, radiusX, radiusY = point.Dx-r.Right+rx, point.Dy-r.Bottom+ry, rx, ry
	} else if rx, ry := box(scaled.BlRadiusX, scaled.BlRadiusY); point.Dx < r.Left+rx && point.Dy > r.Bottom-ry {
		x, y, radiusX, radiusY = point.Dx-r.Left-rx, point.Dy-r.Bottom+ry, rx, ry
	} else {
		return true // inside and not within the rounded corner area
	}
	x /= radiusX
	y /= radiusY
	// check if the point is outside the unit superellipse
	if n == 2 {
		return x*x+y*y <= 1.0
	}
	return math.Pow(math.Abs(x), n)+math.Pow(math.Abs(y), n) <= 1.0
}

func (r rrectLike) toString(className string) string {
	rect := fmt.Sprintf("%s, %s, %s, %s",
		toStringAsFixed1(r.Left), toStringAsFixed1(r.Top), toStringAsFixed1(r.Right), toStringAsFixed1(r.Bottom))
	tl, tr, br, bl := r.tlRadius(), r.trRadius(), r.brRadius(), r.blRadius()
	if tl == tr && tr == br && br == bl {
		if tl.X == tl.Y {
			return fmt.Sprintf("%s.fromLTRBR(%s, %s)", className, rect, toStringAsFixed1(tl.X))
		}
		return fmt.Sprintf("%s.fromLTRBXY(%s, %s, %s)", className, rect, toStringAsFixed1(tl.X), toStringAsFixed1(tl.Y))
	}
	return fmt.Sprintf("%s.fromLTRBAndCorners(%s, topLeft: %s, topRight: %s, bottomRight: %s, bottomLeft: %s)",
		className, rect, tl, tr, br, bl)
}

// RRect is an immutable rectangle with elliptical rounded corners.
//
// Its fields are the rectangle's edges and the radius of each corner on each
// axis.
//
// Flutter: ui/geometry.dart RRect
type RRect rrectLike

// RRectZero is a RRect with all of its fields zero.
//
// Flutter: ui/geometry.dart RRect.zero
var RRectZero = RRect{}

// NewRRectFromLTRBXY constructs a RRect from the left, top, right and bottom edges
// and the same x and y radius for all four corners.
//
// Flutter: ui/geometry.dart RRect.fromLTRBXY
func NewRRectFromLTRBXY(left, top, right, bottom, radiusX, radiusY float64) RRect {
	return RRect(newRRectLikeLTRBXY(left, top, right, bottom, radiusX, radiusY))
}

// NewRRectFromLTRBR constructs a RRect from the left, top, right and bottom edges
// and the same radius for all four corners.
//
// Flutter: ui/geometry.dart RRect.fromLTRBR
func NewRRectFromLTRBR(left, top, right, bottom float64, radius Radius) RRect {
	return RRect(newRRectLikeLTRBXY(left, top, right, bottom, radius.X, radius.Y))
}

// NewRRectFromRectXY constructs a RRect from a rectangle and the same x and y radius
// for all four corners.
//
// Flutter: ui/geometry.dart RRect.fromRectXY
func NewRRectFromRectXY(rect Rect, radiusX, radiusY float64) RRect {
	return RRect(newRRectLikeLTRBXY(rect.Left, rect.Top, rect.Right, rect.Bottom, radiusX, radiusY))
}

// NewRRectFromRectAndRadius constructs a RRect from a rectangle and the same radius
// for all four corners.
//
// Flutter: ui/geometry.dart RRect.fromRectAndRadius
func NewRRectFromRectAndRadius(rect Rect, radius Radius) RRect {
	return RRect(newRRectLikeLTRBXY(rect.Left, rect.Top, rect.Right, rect.Bottom, radius.X, radius.Y))
}

// NewRRectFromLTRBAndCorners constructs a RRect from the left, top, right and bottom
// edges and a radius for each corner. Passing more than one Corners panics.
//
// Flutter: ui/geometry.dart RRect.fromLTRBAndCorners
func NewRRectFromLTRBAndCorners(left, top, right, bottom float64, corners ...Corners) RRect {
	return RRect(newRRectLikeCorners(left, top, right, bottom, oneOpts(corners)))
}

// NewRRectFromRectAndCorners constructs a RRect from a rectangle and a radius for
// each corner. Passing more than one Corners panics.
//
// Flutter: ui/geometry.dart RRect.fromRectAndCorners
func NewRRectFromRectAndCorners(rect Rect, corners ...Corners) RRect {
	return RRect(newRRectLikeCorners(rect.Left, rect.Top, rect.Right, rect.Bottom, oneOpts(corners)))
}

// TlRadius is the top left corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.tlRadius
func (r RRect) TlRadius() Radius { return rrectLike(r).tlRadius() }

// TrRadius is the top right corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.trRadius
func (r RRect) TrRadius() Radius { return rrectLike(r).trRadius() }

// BrRadius is the bottom right corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.brRadius
func (r RRect) BrRadius() Radius { return rrectLike(r).brRadius() }

// BlRadius is the bottom left corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.blRadius
func (r RRect) BlRadius() Radius { return rrectLike(r).blRadius() }

// Shift returns a new RRect translated by the given offset.
//
// Flutter: ui/geometry.dart _RRectLike.shift
func (r RRect) Shift(offset Offset) RRect { return RRect(rrectLike(r).shift(offset)) }

// Inflate returns a new RRect with edges and radii moved outwards by the given
// delta.
//
// Flutter: ui/geometry.dart _RRectLike.inflate
func (r RRect) Inflate(delta float64) RRect { return RRect(rrectLike(r).inflate(delta)) }

// Deflate returns a new RRect with edges and radii moved inwards by the given
// delta.
//
// Flutter: ui/geometry.dart _RRectLike.deflate
func (r RRect) Deflate(delta float64) RRect { return r.Inflate(-delta) }

// Width is the distance between the left and right edges.
//
// Flutter: ui/geometry.dart _RRectLike.width
func (r RRect) Width() float64 { return rrectLike(r).width() }

// Height is the distance between the top and bottom edges.
//
// Flutter: ui/geometry.dart _RRectLike.height
func (r RRect) Height() float64 { return rrectLike(r).height() }

// OuterRect is the outer rectangle: the bounding box of the RRect.
//
// Flutter: ui/geometry.dart _RRectLike.outerRect
func (r RRect) OuterRect() Rect { return rrectLike(r).outerRect() }

// SafeInnerRect is the biggest rectangle that is entirely inside the RRect and has
// the same center, found with a safe inset at each corner.
//
// Flutter: ui/geometry.dart _RRectLike.safeInnerRect
func (r RRect) SafeInnerRect() Rect { return rrectLike(r).safeInnerRect() }

// MiddleRect is the rectangle that remains after the corner radii are taken off
// all four sides.
//
// Flutter: ui/geometry.dart _RRectLike.middleRect
func (r RRect) MiddleRect() Rect { return rrectLike(r).middleRect() }

// WideMiddleRect is the full-width rectangle between the top and bottom corner
// radii.
//
// Flutter: ui/geometry.dart _RRectLike.wideMiddleRect
func (r RRect) WideMiddleRect() Rect { return rrectLike(r).wideMiddleRect() }

// TallMiddleRect is the full-height rectangle between the left and right corner
// radii.
//
// Flutter: ui/geometry.dart _RRectLike.tallMiddleRect
func (r RRect) TallMiddleRect() Rect { return rrectLike(r).tallMiddleRect() }

// IsEmpty reports whether the RRect encloses no area.
//
// Flutter: ui/geometry.dart _RRectLike.isEmpty
func (r RRect) IsEmpty() bool { return rrectLike(r).isEmpty() }

// IsFinite reports whether the four edges are finite.
//
// Flutter: ui/geometry.dart _RRectLike.isFinite
func (r RRect) IsFinite() bool { return rrectLike(r).isFinite() }

// IsRect reports whether the RRect is a simple rectangle with no rounded corner.
//
// Flutter: ui/geometry.dart _RRectLike.isRect
func (r RRect) IsRect() bool { return rrectLike(r).isRect() }

// IsStadium reports whether the RRect has the shape of a stadium (two semicircles
// connected by a rectangle).
//
// Flutter: ui/geometry.dart _RRectLike.isStadium
func (r RRect) IsStadium() bool { return rrectLike(r).isStadium() }

// IsEllipse reports whether the RRect has the shape of an ellipse.
//
// Flutter: ui/geometry.dart _RRectLike.isEllipse
func (r RRect) IsEllipse() bool { return rrectLike(r).isEllipse() }

// IsCircle reports whether the RRect has the shape of a circle.
//
// Flutter: ui/geometry.dart _RRectLike.isCircle
func (r RRect) IsCircle() bool { return rrectLike(r).isCircle() }

// ShortestSide is the lesser of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart _RRectLike.shortestSide
func (r RRect) ShortestSide() float64 { return rrectLike(r).shortestSide() }

// LongestSide is the greater of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart _RRectLike.longestSide
func (r RRect) LongestSide() float64 { return rrectLike(r).longestSide() }

// HasNaN reports whether any of the edges or radii is NaN.
//
// Flutter: ui/geometry.dart _RRectLike.hasNaN
func (r RRect) HasNaN() bool { return rrectLike(r).hasNaN() }

// Center is the offset to the point halfway between the left and right and the
// top and bottom edges.
//
// Flutter: ui/geometry.dart _RRectLike.center
func (r RRect) Center() Offset { return rrectLike(r).center() }

// ScaleRadii scales all radii so that on each side their sum will not exceed the
// size of the width or height.
//
// Flutter: ui/geometry.dart _RRectLike.scaleRadii
func (r RRect) ScaleRadii() RRect { return RRect(rrectLike(r).scaleRadii()) }

// Lerp linearly interpolates between r and b: r when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart RRect.lerp
func (r RRect) Lerp(b RRect, t float64) RRect {
	bl := rrectLike(b)
	return RRect(rrectLike(r).lerpTo(&bl, t))
}

// RRectLerp linearly interpolates between two shapes, either of which may be nil.
// A nil shape counts as an empty one at the origin, and two nils give nil.
//
// Flutter: ui/geometry.dart RRect.lerp
func RRectLerp(a, b *RRect, t float64) *RRect {
	var al, bl *rrectLike
	if a != nil {
		v := rrectLike(*a)
		al = &v
	}
	if b != nil {
		v := rrectLike(*b)
		bl = &v
	}
	res := lerpLike(al, bl, t)
	if res == nil {
		return nil
	}
	out := RRect(*res)
	return &out
}

// String describes the RRect, as Dart's toString does.
//
// Flutter: ui/geometry.dart RRect.toString
func (r RRect) String() string { return rrectLike(r).toString("RRect") }

// Contains reports whether the point is inside the rounded rectangle. The left
// and top edges are inside, the right and bottom edges are not. The radii are
// scaled down first when they do not fit (see ScaleRadii).
//
// Flutter: ui/geometry.dart RRect.contains
func (r RRect) Contains(point Offset) bool { return rrectLike(r).contains(point, 1, 2, false) }

// RSuperellipse is an immutable rectangle with rounded superellipse corners: the
// continuous corners of iOS, which curve more gradually than circular ones.
//
// Its fields are the rectangle's edges and the radius of each corner on each
// axis.
//
// Flutter: ui/geometry.dart RSuperellipse
type RSuperellipse rrectLike

// RSuperellipseZero is a RSuperellipse with all of its fields zero.
//
// Flutter: ui/geometry.dart RSuperellipse.zero
var RSuperellipseZero = RSuperellipse{}

// NewRSuperellipseFromLTRBXY constructs a RSuperellipse from the left, top, right and bottom edges
// and the same x and y radius for all four corners.
//
// Flutter: ui/geometry.dart RSuperellipse.fromLTRBXY
func NewRSuperellipseFromLTRBXY(left, top, right, bottom, radiusX, radiusY float64) RSuperellipse {
	return RSuperellipse(newRRectLikeLTRBXY(left, top, right, bottom, radiusX, radiusY))
}

// NewRSuperellipseFromLTRBR constructs a RSuperellipse from the left, top, right and bottom edges
// and the same radius for all four corners.
//
// Flutter: ui/geometry.dart RSuperellipse.fromLTRBR
func NewRSuperellipseFromLTRBR(left, top, right, bottom float64, radius Radius) RSuperellipse {
	return RSuperellipse(newRRectLikeLTRBXY(left, top, right, bottom, radius.X, radius.Y))
}

// NewRSuperellipseFromRectXY constructs a RSuperellipse from a rectangle and the same x and y radius
// for all four corners.
//
// Flutter: ui/geometry.dart RSuperellipse.fromRectXY
func NewRSuperellipseFromRectXY(rect Rect, radiusX, radiusY float64) RSuperellipse {
	return RSuperellipse(newRRectLikeLTRBXY(rect.Left, rect.Top, rect.Right, rect.Bottom, radiusX, radiusY))
}

// NewRSuperellipseFromRectAndRadius constructs a RSuperellipse from a rectangle and the same radius
// for all four corners.
//
// Flutter: ui/geometry.dart RSuperellipse.fromRectAndRadius
func NewRSuperellipseFromRectAndRadius(rect Rect, radius Radius) RSuperellipse {
	return RSuperellipse(newRRectLikeLTRBXY(rect.Left, rect.Top, rect.Right, rect.Bottom, radius.X, radius.Y))
}

// NewRSuperellipseFromLTRBAndCorners constructs a RSuperellipse from the left, top, right and bottom
// edges and a radius for each corner. Passing more than one Corners panics.
//
// Flutter: ui/geometry.dart RSuperellipse.fromLTRBAndCorners
func NewRSuperellipseFromLTRBAndCorners(left, top, right, bottom float64, corners ...Corners) RSuperellipse {
	return RSuperellipse(newRRectLikeCorners(left, top, right, bottom, oneOpts(corners)))
}

// NewRSuperellipseFromRectAndCorners constructs a RSuperellipse from a rectangle and a radius for
// each corner. Passing more than one Corners panics.
//
// Flutter: ui/geometry.dart RSuperellipse.fromRectAndCorners
func NewRSuperellipseFromRectAndCorners(rect Rect, corners ...Corners) RSuperellipse {
	return RSuperellipse(newRRectLikeCorners(rect.Left, rect.Top, rect.Right, rect.Bottom, oneOpts(corners)))
}

// TlRadius is the top left corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.tlRadius
func (r RSuperellipse) TlRadius() Radius { return rrectLike(r).tlRadius() }

// TrRadius is the top right corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.trRadius
func (r RSuperellipse) TrRadius() Radius { return rrectLike(r).trRadius() }

// BrRadius is the bottom right corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.brRadius
func (r RSuperellipse) BrRadius() Radius { return rrectLike(r).brRadius() }

// BlRadius is the bottom left corner's radius.
//
// Flutter: ui/geometry.dart _RRectLike.blRadius
func (r RSuperellipse) BlRadius() Radius { return rrectLike(r).blRadius() }

// Shift returns a new RSuperellipse translated by the given offset.
//
// Flutter: ui/geometry.dart _RRectLike.shift
func (r RSuperellipse) Shift(offset Offset) RSuperellipse {
	return RSuperellipse(rrectLike(r).shift(offset))
}

// Inflate returns a new RSuperellipse with edges and radii moved outwards by the given
// delta.
//
// Flutter: ui/geometry.dart _RRectLike.inflate
func (r RSuperellipse) Inflate(delta float64) RSuperellipse {
	return RSuperellipse(rrectLike(r).inflate(delta))
}

// Deflate returns a new RSuperellipse with edges and radii moved inwards by the given
// delta.
//
// Flutter: ui/geometry.dart _RRectLike.deflate
func (r RSuperellipse) Deflate(delta float64) RSuperellipse { return r.Inflate(-delta) }

// Width is the distance between the left and right edges.
//
// Flutter: ui/geometry.dart _RRectLike.width
func (r RSuperellipse) Width() float64 { return rrectLike(r).width() }

// Height is the distance between the top and bottom edges.
//
// Flutter: ui/geometry.dart _RRectLike.height
func (r RSuperellipse) Height() float64 { return rrectLike(r).height() }

// OuterRect is the outer rectangle: the bounding box of the RSuperellipse.
//
// Flutter: ui/geometry.dart _RRectLike.outerRect
func (r RSuperellipse) OuterRect() Rect { return rrectLike(r).outerRect() }

// SafeInnerRect is the biggest rectangle that is entirely inside the RSuperellipse and has
// the same center, found with a safe inset at each corner.
//
// Flutter: ui/geometry.dart _RRectLike.safeInnerRect
func (r RSuperellipse) SafeInnerRect() Rect { return rrectLike(r).safeInnerRect() }

// MiddleRect is the rectangle that remains after the corner radii are taken off
// all four sides.
//
// Flutter: ui/geometry.dart _RRectLike.middleRect
func (r RSuperellipse) MiddleRect() Rect { return rrectLike(r).middleRect() }

// WideMiddleRect is the full-width rectangle between the top and bottom corner
// radii.
//
// Flutter: ui/geometry.dart _RRectLike.wideMiddleRect
func (r RSuperellipse) WideMiddleRect() Rect { return rrectLike(r).wideMiddleRect() }

// TallMiddleRect is the full-height rectangle between the left and right corner
// radii.
//
// Flutter: ui/geometry.dart _RRectLike.tallMiddleRect
func (r RSuperellipse) TallMiddleRect() Rect { return rrectLike(r).tallMiddleRect() }

// IsEmpty reports whether the RSuperellipse encloses no area.
//
// Flutter: ui/geometry.dart _RRectLike.isEmpty
func (r RSuperellipse) IsEmpty() bool { return rrectLike(r).isEmpty() }

// IsFinite reports whether the four edges are finite.
//
// Flutter: ui/geometry.dart _RRectLike.isFinite
func (r RSuperellipse) IsFinite() bool { return rrectLike(r).isFinite() }

// IsRect reports whether the RSuperellipse is a simple rectangle with no rounded corner.
//
// Flutter: ui/geometry.dart _RRectLike.isRect
func (r RSuperellipse) IsRect() bool { return rrectLike(r).isRect() }

// IsStadium reports whether the RSuperellipse has the shape of a stadium (two semicircles
// connected by a rectangle).
//
// Flutter: ui/geometry.dart _RRectLike.isStadium
func (r RSuperellipse) IsStadium() bool { return rrectLike(r).isStadium() }

// IsEllipse reports whether the RSuperellipse has the shape of an ellipse.
//
// Flutter: ui/geometry.dart _RRectLike.isEllipse
func (r RSuperellipse) IsEllipse() bool { return rrectLike(r).isEllipse() }

// IsCircle reports whether the RSuperellipse has the shape of a circle.
//
// Flutter: ui/geometry.dart _RRectLike.isCircle
func (r RSuperellipse) IsCircle() bool { return rrectLike(r).isCircle() }

// ShortestSide is the lesser of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart _RRectLike.shortestSide
func (r RSuperellipse) ShortestSide() float64 { return rrectLike(r).shortestSide() }

// LongestSide is the greater of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart _RRectLike.longestSide
func (r RSuperellipse) LongestSide() float64 { return rrectLike(r).longestSide() }

// HasNaN reports whether any of the edges or radii is NaN.
//
// Flutter: ui/geometry.dart _RRectLike.hasNaN
func (r RSuperellipse) HasNaN() bool { return rrectLike(r).hasNaN() }

// Center is the offset to the point halfway between the left and right and the
// top and bottom edges.
//
// Flutter: ui/geometry.dart _RRectLike.center
func (r RSuperellipse) Center() Offset { return rrectLike(r).center() }

// ScaleRadii scales all radii so that on each side their sum will not exceed the
// size of the width or height.
//
// Flutter: ui/geometry.dart _RRectLike.scaleRadii
func (r RSuperellipse) ScaleRadii() RSuperellipse { return RSuperellipse(rrectLike(r).scaleRadii()) }

// Lerp linearly interpolates between r and b: r when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart RSuperellipse.lerp
func (r RSuperellipse) Lerp(b RSuperellipse, t float64) RSuperellipse {
	bl := rrectLike(b)
	return RSuperellipse(rrectLike(r).lerpTo(&bl, t))
}

// RSuperellipseLerp linearly interpolates between two shapes, either of which may be nil.
// A nil shape counts as an empty one at the origin, and two nils give nil.
//
// Flutter: ui/geometry.dart RSuperellipse.lerp
func RSuperellipseLerp(a, b *RSuperellipse, t float64) *RSuperellipse {
	var al, bl *rrectLike
	if a != nil {
		v := rrectLike(*a)
		al = &v
	}
	if b != nil {
		v := rrectLike(*b)
		bl = &v
	}
	res := lerpLike(al, bl, t)
	if res == nil {
		return nil
	}
	out := RSuperellipse(*res)
	return &out
}

// String describes the RSuperellipse, as Dart's toString does.
//
// Flutter: ui/geometry.dart RSuperellipse.toString
func (r RSuperellipse) String() string { return rrectLike(r).toString("RSuperellipse") }

// superellipseExtent and superellipseExponent shape the corners of
// RSuperellipse.Contains: a corner is the curve |x|^n + |y|^n = 1 inside a box
// that is superellipseExtent times the radius.
const (
	superellipseExtent   = 1.2924
	superellipseExponent = 5
)

// Contains reports whether the point is inside the shape. The left and top edges
// are inside, the right and bottom edges are not.
//
// Deviation: Flutter asks Impeller's RoundSuperellipse (impeller/geometry),
// which is not in reference/. This approximates its corners with the
// superellipse of exponent 5 in a box 1.2924 times the radius, so a point near
// a corner can be classified differently by a few thousandths of the radius.
// SPEC §12.10 lists it.
//
// Flutter: ui/geometry.dart RSuperellipse.contains
func (r RSuperellipse) Contains(point Offset) bool {
	return rrectLike(r).contains(point, superellipseExtent, superellipseExponent, true)
}

// RSTransform is a rotation, uniform scale and translation in the form the
// atlas drawing calls take, modeled after Impeller's RSTransform: the matrix
//
//	[ scos -ssin  tx ]
//	[ ssin  scos  ty ]
//	[    0     0   1 ]
//
// Flutter: ui/geometry.dart RSTransform
type RSTransform struct {
	// Scos is the cosine of the rotation, multiplied by the scale.
	Scos float32
	// Ssin is the sine of the rotation, multiplied by the scale.
	Ssin float32
	// Tx is the horizontal translation.
	Tx float32
	// Ty is the vertical translation.
	Ty float32
}

// NewRSTransform creates an RSTransform from the matrix entries.
//
// Flutter: ui/geometry.dart RSTransform
func NewRSTransform(scos, ssin, tx, ty float32) RSTransform { return RSTransform{scos, ssin, tx, ty} }

// NewRSTransformFromComponents creates an RSTransform from a rotation in
// radians, a uniform scale, the anchor point of the source rectangle that stays
// fixed, and the point it moves to.
//
// Flutter: ui/geometry.dart RSTransform.fromComponents
func NewRSTransformFromComponents(rotation, scale, anchorX, anchorY, translateX, translateY float64) RSTransform {
	scos := math.Cos(rotation) * scale
	ssin := math.Sin(rotation) * scale
	tx := translateX + -scos*anchorX + ssin*anchorY
	ty := translateY + -ssin*anchorX - scos*anchorY
	return RSTransform{float32(scos), float32(ssin), float32(tx), float32(ty)}
}
