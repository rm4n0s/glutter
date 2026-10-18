package ui

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
)

// Offset is an immutable 2D floating-point offset: a point or a vector in
// logical or physical pixels, depending on where it is used.
//
// Flutter: ui/geometry.dart Offset
type Offset struct {
	// Dx is the horizontal component.
	Dx float64
	// Dy is the vertical component.
	Dy float64
}

// OffsetZero is an offset with zero magnitude.
//
// Flutter: ui/geometry.dart Offset.zero
var OffsetZero = Offset{}

// OffsetInfinite is an offset with infinite x and y components.
//
// Flutter: ui/geometry.dart Offset.infinite
var OffsetInfinite = Offset{math.Inf(1), math.Inf(1)}

// NewOffsetFromDirection returns the offset whose direction is the given angle
// in radians, measured from the positive x axis, and whose length is distance.
// Flutter's default distance is 1.0.
//
// Deviation: distance is a positional argument, because package ui cannot use
// foundation.Opt.
//
// Flutter: ui/geometry.dart Offset.fromDirection
func NewOffsetFromDirection(direction, distance float64) Offset {
	return Offset{distance * math.Cos(direction), distance * math.Sin(direction)}
}

// Distance is the magnitude of this offset.
//
// Flutter: ui/geometry.dart Offset.distance
func (o Offset) Distance() float64 { return math.Sqrt(o.Dx*o.Dx + o.Dy*o.Dy) }

// DistanceSquared is the square of the magnitude of this offset.
//
// Flutter: ui/geometry.dart Offset.distanceSquared
func (o Offset) DistanceSquared() float64 { return o.Dx*o.Dx + o.Dy*o.Dy }

// Direction is the angle of this offset as radians clockwise from the positive
// x axis, in the range -pi to pi.
//
// Flutter: ui/geometry.dart Offset.direction
func (o Offset) Direction() float64 { return math.Atan2(o.Dy, o.Dx) }

// IsInfinite reports whether either component is infinite.
//
// Flutter: ui/geometry.dart OffsetBase.isInfinite
func (o Offset) IsInfinite() bool { return o.Dx >= math.Inf(1) || o.Dy >= math.Inf(1) }

// IsFinite reports whether both components are finite (neither infinite nor
// NaN).
//
// Flutter: ui/geometry.dart OffsetBase.isFinite
func (o Offset) IsFinite() bool { return isFinite(o.Dx) && isFinite(o.Dy) }

// Lt reports whether both components are less than the other's (Dart's <).
//
// Flutter: ui/geometry.dart OffsetBase.operator <
func (o Offset) Lt(other Offset) bool { return o.Dx < other.Dx && o.Dy < other.Dy }

// Le reports whether both components are less than or equal to the other's
// (Dart's <=).
//
// Flutter: ui/geometry.dart OffsetBase.operator <=
func (o Offset) Le(other Offset) bool { return o.Dx <= other.Dx && o.Dy <= other.Dy }

// Gt reports whether both components are greater than the other's (Dart's >).
//
// Flutter: ui/geometry.dart OffsetBase.operator >
func (o Offset) Gt(other Offset) bool { return o.Dx > other.Dx && o.Dy > other.Dy }

// Ge reports whether both components are greater than or equal to the other's
// (Dart's >=).
//
// Flutter: ui/geometry.dart OffsetBase.operator >=
func (o Offset) Ge(other Offset) bool { return o.Dx >= other.Dx && o.Dy >= other.Dy }

// Scale returns an offset with Dx multiplied by scaleX and Dy by scaleY.
//
// Flutter: ui/geometry.dart Offset.scale
func (o Offset) Scale(scaleX, scaleY float64) Offset { return Offset{o.Dx * scaleX, o.Dy * scaleY} }

// Translate returns an offset with translateX added to Dx and translateY to
// Dy.
//
// Flutter: ui/geometry.dart Offset.translate
func (o Offset) Translate(translateX, translateY float64) Offset {
	return Offset{o.Dx + translateX, o.Dy + translateY}
}

// Neg returns the offset with both components negated (Dart's unary -).
//
// Flutter: ui/geometry.dart Offset.operator -
func (o Offset) Neg() Offset { return Offset{-o.Dx, -o.Dy} }

// Sub returns the difference of two offsets (Dart's -).
//
// Flutter: ui/geometry.dart Offset.operator -
func (o Offset) Sub(other Offset) Offset { return Offset{o.Dx - other.Dx, o.Dy - other.Dy} }

// Add returns the sum of two offsets (Dart's +).
//
// Flutter: ui/geometry.dart Offset.operator +
func (o Offset) Add(other Offset) Offset { return Offset{o.Dx + other.Dx, o.Dy + other.Dy} }

// Mul multiplies both components by operand (Dart's *).
//
// Flutter: ui/geometry.dart Offset.operator *
func (o Offset) Mul(operand float64) Offset { return Offset{o.Dx * operand, o.Dy * operand} }

// Div divides both components by operand (Dart's /).
//
// Flutter: ui/geometry.dart Offset.operator /
func (o Offset) Div(operand float64) Offset { return Offset{o.Dx / operand, o.Dy / operand} }

// TruncDiv divides both components by operand and truncates the results
// (Dart's ~/).
//
// Deviation: an infinite or NaN result stays so, where Dart throws.
//
// Flutter: ui/geometry.dart Offset.operator ~/
func (o Offset) TruncDiv(operand float64) Offset {
	return Offset{math.Trunc(o.Dx / operand), math.Trunc(o.Dy / operand)}
}

// Mod returns the remainder of dividing both components by operand, made
// non-negative as Dart does (Dart's %).
//
// Flutter: ui/geometry.dart Offset.operator %
func (o Offset) Mod(operand float64) Offset {
	return Offset{dartMod(o.Dx, operand), dartMod(o.Dy, operand)}
}

// And returns the rectangle whose top left corner is this offset and whose size
// is other (Dart's &).
//
// Flutter: ui/geometry.dart Offset.operator &
func (o Offset) And(other Size) Rect { return NewRectFromLTWH(o.Dx, o.Dy, other.Width, other.Height) }

// Lerp linearly interpolates between o and b: o when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart Offset.lerp
func (o Offset) Lerp(b Offset, t float64) Offset {
	return Offset{lerpDouble(o.Dx, b.Dx, t), lerpDouble(o.Dy, b.Dy, t)}
}

// OffsetLerp linearly interpolates between two offsets, either of which may be
// nil. A nil offset counts as zero, and two nils give nil.
//
// Flutter: ui/geometry.dart Offset.lerp
func OffsetLerp(a, b *Offset, t float64) *Offset {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		r := a.Mul(1.0 - t)
		return &r
	case a == nil:
		r := b.Mul(t)
		return &r
	}
	r := a.Lerp(*b, t)
	return &r
}

// String describes the offset, as Dart's toString does.
//
// Flutter: ui/geometry.dart Offset.toString
func (o Offset) String() string {
	return fmt.Sprintf("Offset(%s, %s)", toStringAsFixed1(o.Dx), toStringAsFixed1(o.Dy))
}

// Size holds a 2D floating-point size: a width and a height.
//
// Flutter: ui/geometry.dart Size
type Size struct {
	// Width is the horizontal extent.
	Width float64
	// Height is the vertical extent.
	Height float64
}

// SizeZero is a size whose width and height are zero.
//
// Flutter: ui/geometry.dart Size.zero
var SizeZero = Size{}

// SizeInfinite is a size whose width and height are infinite.
//
// Flutter: ui/geometry.dart Size.infinite
var SizeInfinite = Size{math.Inf(1), math.Inf(1)}

// NewSizeSquare returns a size with the same width and height.
//
// Flutter: ui/geometry.dart Size.square
func NewSizeSquare(dimension float64) Size { return Size{dimension, dimension} }

// NewSizeFromWidth returns a size with the given width and an infinite height.
//
// Flutter: ui/geometry.dart Size.fromWidth
func NewSizeFromWidth(width float64) Size { return Size{width, math.Inf(1)} }

// NewSizeFromHeight returns a size with the given height and an infinite width.
//
// Flutter: ui/geometry.dart Size.fromHeight
func NewSizeFromHeight(height float64) Size { return Size{math.Inf(1), height} }

// NewSizeFromRadius returns the size of a circle with the given radius.
//
// Flutter: ui/geometry.dart Size.fromRadius
func NewSizeFromRadius(radius float64) Size { return Size{radius * 2.0, radius * 2.0} }

// AspectRatio is the lesser of the ratio between width and height and the ratio
// between height and width: here, width divided by height, with the cases of a
// zero height handled as Flutter does.
//
// Flutter: ui/geometry.dart Size.aspectRatio
func (s Size) AspectRatio() float64 {
	if s.Height != 0.0 {
		return s.Width / s.Height
	}
	if s.Width > 0.0 {
		return math.Inf(1)
	}
	if s.Width < 0.0 {
		return math.Inf(-1)
	}
	return 0.0
}

// IsEmpty reports whether either dimension is zero or negative.
//
// Flutter: ui/geometry.dart Size.isEmpty
func (s Size) IsEmpty() bool { return s.Width <= 0.0 || s.Height <= 0.0 }

// IsInfinite reports whether either dimension is infinite.
//
// Flutter: ui/geometry.dart OffsetBase.isInfinite
func (s Size) IsInfinite() bool { return s.Width >= math.Inf(1) || s.Height >= math.Inf(1) }

// IsFinite reports whether both dimensions are finite.
//
// Flutter: ui/geometry.dart OffsetBase.isFinite
func (s Size) IsFinite() bool { return isFinite(s.Width) && isFinite(s.Height) }

// Lt reports whether both dimensions are less than the other's (Dart's <).
//
// Flutter: ui/geometry.dart OffsetBase.operator <
func (s Size) Lt(other Size) bool { return s.Width < other.Width && s.Height < other.Height }

// Le reports whether both dimensions are less than or equal to the other's
// (Dart's <=).
//
// Flutter: ui/geometry.dart OffsetBase.operator <=
func (s Size) Le(other Size) bool { return s.Width <= other.Width && s.Height <= other.Height }

// Gt reports whether both dimensions are greater than the other's (Dart's >).
//
// Flutter: ui/geometry.dart OffsetBase.operator >
func (s Size) Gt(other Size) bool { return s.Width > other.Width && s.Height > other.Height }

// Ge reports whether both dimensions are greater than or equal to the other's
// (Dart's >=).
//
// Flutter: ui/geometry.dart OffsetBase.operator >=
func (s Size) Ge(other Size) bool { return s.Width >= other.Width && s.Height >= other.Height }

// Sub returns the offset that, added to other, gives s: the difference of two
// sizes (Dart's - with a Size).
//
// Flutter: ui/geometry.dart Size.operator -
func (s Size) Sub(other Size) Offset { return Offset{s.Width - other.Width, s.Height - other.Height} }

// SubOffset returns s made smaller by the offset (Dart's - with an Offset).
//
// Flutter: ui/geometry.dart Size.operator -
func (s Size) SubOffset(other Offset) Size { return Size{s.Width - other.Dx, s.Height - other.Dy} }

// Add returns s made larger by the offset (Dart's +).
//
// Flutter: ui/geometry.dart Size.operator +
func (s Size) Add(other Offset) Size { return Size{s.Width + other.Dx, s.Height + other.Dy} }

// Mul multiplies both dimensions by operand (Dart's *).
//
// Flutter: ui/geometry.dart Size.operator *
func (s Size) Mul(operand float64) Size { return Size{s.Width * operand, s.Height * operand} }

// Div divides both dimensions by operand (Dart's /).
//
// Flutter: ui/geometry.dart Size.operator /
func (s Size) Div(operand float64) Size { return Size{s.Width / operand, s.Height / operand} }

// TruncDiv divides both dimensions by operand and truncates the results
// (Dart's ~/).
//
// Deviation: an infinite or NaN result stays so, where Dart throws.
//
// Flutter: ui/geometry.dart Size.operator ~/
func (s Size) TruncDiv(operand float64) Size {
	return Size{math.Trunc(s.Width / operand), math.Trunc(s.Height / operand)}
}

// Mod returns the remainder of dividing both dimensions by operand, made
// non-negative as Dart does (Dart's %).
//
// Flutter: ui/geometry.dart Size.operator %
func (s Size) Mod(operand float64) Size {
	return Size{dartMod(s.Width, operand), dartMod(s.Height, operand)}
}

// ShortestSide is the lesser of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart Size.shortestSide
func (s Size) ShortestSide() float64 { return math.Min(math.Abs(s.Width), math.Abs(s.Height)) }

// LongestSide is the greater of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart Size.longestSide
func (s Size) LongestSide() float64 { return math.Max(math.Abs(s.Width), math.Abs(s.Height)) }

// TopLeft is the offset to the intersection of the top and left edges of the
// rectangle of this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.topLeft
func (s Size) TopLeft(origin Offset) Offset { return origin }

// TopCenter is the offset to the center of the top edge of the rectangle of this
// size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.topCenter
func (s Size) TopCenter(origin Offset) Offset { return Offset{origin.Dx + s.Width/2.0, origin.Dy} }

// TopRight is the offset to the intersection of the top and right edges of the
// rectangle of this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.topRight
func (s Size) TopRight(origin Offset) Offset { return Offset{origin.Dx + s.Width, origin.Dy} }

// CenterLeft is the offset to the center of the left edge of the rectangle of
// this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.centerLeft
func (s Size) CenterLeft(origin Offset) Offset { return Offset{origin.Dx, origin.Dy + s.Height/2.0} }

// Center is the offset to the point halfway between the left and right and the
// top and bottom edges of the rectangle of this size whose top left corner is
// origin.
//
// Flutter: ui/geometry.dart Size.center
func (s Size) Center(origin Offset) Offset {
	return Offset{origin.Dx + s.Width/2.0, origin.Dy + s.Height/2.0}
}

// CenterRight is the offset to the center of the right edge of the rectangle of
// this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.centerRight
func (s Size) CenterRight(origin Offset) Offset {
	return Offset{origin.Dx + s.Width, origin.Dy + s.Height/2.0}
}

// BottomLeft is the offset to the intersection of the bottom and left edges of
// the rectangle of this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.bottomLeft
func (s Size) BottomLeft(origin Offset) Offset { return Offset{origin.Dx, origin.Dy + s.Height} }

// BottomCenter is the offset to the center of the bottom edge of the rectangle
// of this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.bottomCenter
func (s Size) BottomCenter(origin Offset) Offset {
	return Offset{origin.Dx + s.Width/2.0, origin.Dy + s.Height}
}

// BottomRight is the offset to the intersection of the bottom and right edges of
// the rectangle of this size whose top left corner is origin.
//
// Flutter: ui/geometry.dart Size.bottomRight
func (s Size) BottomRight(origin Offset) Offset {
	return Offset{origin.Dx + s.Width, origin.Dy + s.Height}
}

// Contains reports whether the point is inside the rectangle of this size whose
// top left corner is the origin. The left and top edges are inside, the right
// and bottom edges are not.
//
// Flutter: ui/geometry.dart Size.contains
func (s Size) Contains(offset Offset) bool {
	return offset.Dx >= 0.0 && offset.Dx < s.Width && offset.Dy >= 0.0 && offset.Dy < s.Height
}

// Flipped returns a size with the width and height swapped.
//
// Flutter: ui/geometry.dart Size.flipped
func (s Size) Flipped() Size { return Size{s.Height, s.Width} }

// Lerp linearly interpolates between s and b: s when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart Size.lerp
func (s Size) Lerp(b Size, t float64) Size {
	return Size{lerpDouble(s.Width, b.Width, t), lerpDouble(s.Height, b.Height, t)}
}

// SizeLerp linearly interpolates between two sizes, either of which may be nil.
// A nil size counts as zero, and two nils give nil.
//
// Flutter: ui/geometry.dart Size.lerp
func SizeLerp(a, b *Size, t float64) *Size {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		r := a.Mul(1.0 - t)
		return &r
	case a == nil:
		r := b.Mul(t)
		return &r
	}
	r := a.Lerp(*b, t)
	return &r
}

// String describes the size, as Dart's toString does.
//
// Flutter: ui/geometry.dart Size.toString
func (s Size) String() string {
	return fmt.Sprintf("Size(%s, %s)", toStringAsFixed1(s.Width), toStringAsFixed1(s.Height))
}

// Rect is an immutable, 2D, axis-aligned, floating-point rectangle whose
// coordinates are given relative to an origin.
//
// Flutter: ui/geometry.dart Rect
type Rect struct {
	// Left is the offset of the left edge from the x axis.
	Left float64
	// Top is the offset of the top edge from the y axis.
	Top float64
	// Right is the offset of the right edge from the x axis.
	Right float64
	// Bottom is the offset of the bottom edge from the y axis.
	Bottom float64
}

// giantScalar matches kGiantRect from Flutter's layer.h.
const giantScalar = 1.0e+9

// RectZero is a rectangle with left, top, right and bottom edges all at zero.
//
// Flutter: ui/geometry.dart Rect.zero
var RectZero = Rect{}

// RectLargest is a rectangle that covers the entire coordinate space, or at
// least a very large part of it.
//
// Flutter: ui/geometry.dart Rect.largest
var RectLargest = Rect{-giantScalar, -giantScalar, giantScalar, giantScalar}

// NewRectFromLTRB constructs a rectangle from its left, top, right and bottom
// edges.
//
// Flutter: ui/geometry.dart Rect.fromLTRB
func NewRectFromLTRB(left, top, right, bottom float64) Rect {
	return Rect{left, top, right, bottom}
}

// NewRectFromLTWH constructs a rectangle from its left and top edges, its width
// and its height.
//
// Flutter: ui/geometry.dart Rect.fromLTWH
func NewRectFromLTWH(left, top, width, height float64) Rect {
	return Rect{left, top, left + width, top + height}
}

// NewRectFromCircle constructs the rectangle that bounds a circle with the given
// center and radius.
//
// Flutter: ui/geometry.dart Rect.fromCircle
func NewRectFromCircle(center Offset, radius float64) Rect {
	return NewRectFromCenter(center, radius*2, radius*2)
}

// NewRectFromCenter constructs a rectangle from its center point, width and
// height.
//
// Flutter: ui/geometry.dart Rect.fromCenter
func NewRectFromCenter(center Offset, width, height float64) Rect {
	return Rect{center.Dx - width/2, center.Dy - height/2, center.Dx + width/2, center.Dy + height/2}
}

// NewRectFromPoints constructs the smallest rectangle that contains the two
// points.
//
// Flutter: ui/geometry.dart Rect.fromPoints
func NewRectFromPoints(a, b Offset) Rect {
	return Rect{math.Min(a.Dx, b.Dx), math.Min(a.Dy, b.Dy), math.Max(a.Dx, b.Dx), math.Max(a.Dy, b.Dy)}
}

// Width is the distance between the left and right edges of the rectangle.
//
// Flutter: ui/geometry.dart Rect.width
func (r Rect) Width() float64 { return r.Right - r.Left }

// Height is the distance between the top and bottom edges of the rectangle.
//
// Flutter: ui/geometry.dart Rect.height
func (r Rect) Height() float64 { return r.Bottom - r.Top }

// Size is the distance between the top left and bottom right corners.
//
// Flutter: ui/geometry.dart Rect.size
func (r Rect) Size() Size { return Size{r.Width(), r.Height()} }

// HasNaN reports whether any of the coordinates is NaN.
//
// Flutter: ui/geometry.dart Rect.hasNaN
func (r Rect) HasNaN() bool {
	return math.IsNaN(r.Left) || math.IsNaN(r.Top) || math.IsNaN(r.Right) || math.IsNaN(r.Bottom)
}

// IsInfinite reports whether any of the coordinates is positive infinity.
//
// Flutter: ui/geometry.dart Rect.isInfinite
func (r Rect) IsInfinite() bool {
	inf := math.Inf(1)
	return r.Left >= inf || r.Top >= inf || r.Right >= inf || r.Bottom >= inf
}

// IsFinite reports whether all four coordinates are finite.
//
// Flutter: ui/geometry.dart Rect.isFinite
func (r Rect) IsFinite() bool {
	return isFinite(r.Left) && isFinite(r.Top) && isFinite(r.Right) && isFinite(r.Bottom)
}

// IsEmpty reports whether the rectangle encloses no area: its left edge is not
// to the left of its right edge, or its top is not above its bottom.
//
// Flutter: ui/geometry.dart Rect.isEmpty
func (r Rect) IsEmpty() bool { return r.Left >= r.Right || r.Top >= r.Bottom }

// Shift returns a new rectangle translated by the given offset.
//
// Flutter: ui/geometry.dart Rect.shift
func (r Rect) Shift(offset Offset) Rect {
	return Rect{r.Left + offset.Dx, r.Top + offset.Dy, r.Right + offset.Dx, r.Bottom + offset.Dy}
}

// Translate returns a new rectangle with translateX added to the x components
// and translateY added to the y components.
//
// Flutter: ui/geometry.dart Rect.translate
func (r Rect) Translate(translateX, translateY float64) Rect {
	return Rect{r.Left + translateX, r.Top + translateY, r.Right + translateX, r.Bottom + translateY}
}

// Inflate returns a new rectangle with edges moved outwards by the given delta.
//
// Flutter: ui/geometry.dart Rect.inflate
func (r Rect) Inflate(delta float64) Rect {
	return Rect{r.Left - delta, r.Top - delta, r.Right + delta, r.Bottom + delta}
}

// Deflate returns a new rectangle with edges moved inwards by the given delta.
//
// Flutter: ui/geometry.dart Rect.deflate
func (r Rect) Deflate(delta float64) Rect { return r.Inflate(-delta) }

// Intersect returns a new rectangle that is the intersection of the given
// rectangle and this rectangle. The two rectangles must overlap for the result
// to be meaningful: check IsEmpty on the result.
//
// Flutter: ui/geometry.dart Rect.intersect
func (r Rect) Intersect(other Rect) Rect {
	return Rect{
		math.Max(r.Left, other.Left),
		math.Max(r.Top, other.Top),
		math.Min(r.Right, other.Right),
		math.Min(r.Bottom, other.Bottom),
	}
}

// ExpandToInclude returns a new rectangle which is the bounding box containing
// this rectangle and the given rectangle.
//
// Flutter: ui/geometry.dart Rect.expandToInclude
func (r Rect) ExpandToInclude(other Rect) Rect {
	return Rect{
		math.Min(r.Left, other.Left),
		math.Min(r.Top, other.Top),
		math.Max(r.Right, other.Right),
		math.Max(r.Bottom, other.Bottom),
	}
}

// Overlaps reports whether the two rectangles have a non-zero area of overlap.
//
// Flutter: ui/geometry.dart Rect.overlaps
func (r Rect) Overlaps(other Rect) bool {
	if r.Right <= other.Left || other.Right <= r.Left {
		return false
	}
	if r.Bottom <= other.Top || other.Bottom <= r.Top {
		return false
	}
	return true
}

// ShortestSide is the lesser of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart Rect.shortestSide
func (r Rect) ShortestSide() float64 { return math.Min(math.Abs(r.Width()), math.Abs(r.Height())) }

// LongestSide is the greater of the magnitudes of the width and the height.
//
// Flutter: ui/geometry.dart Rect.longestSide
func (r Rect) LongestSide() float64 { return math.Max(math.Abs(r.Width()), math.Abs(r.Height())) }

// TopLeft is the offset to the intersection of the top and left edges.
//
// Flutter: ui/geometry.dart Rect.topLeft
func (r Rect) TopLeft() Offset { return Offset{r.Left, r.Top} }

// TopCenter is the offset to the center of the top edge.
//
// Flutter: ui/geometry.dart Rect.topCenter
func (r Rect) TopCenter() Offset { return Offset{r.Left + r.Width()/2.0, r.Top} }

// TopRight is the offset to the intersection of the top and right edges.
//
// Flutter: ui/geometry.dart Rect.topRight
func (r Rect) TopRight() Offset { return Offset{r.Right, r.Top} }

// CenterLeft is the offset to the center of the left edge.
//
// Flutter: ui/geometry.dart Rect.centerLeft
func (r Rect) CenterLeft() Offset { return Offset{r.Left, r.Top + r.Height()/2.0} }

// Center is the offset to the point halfway between the left and right and the
// top and bottom edges.
//
// Flutter: ui/geometry.dart Rect.center
func (r Rect) Center() Offset { return Offset{r.Left + r.Width()/2.0, r.Top + r.Height()/2.0} }

// CenterRight is the offset to the center of the right edge.
//
// Flutter: ui/geometry.dart Rect.centerRight
func (r Rect) CenterRight() Offset { return Offset{r.Right, r.Top + r.Height()/2.0} }

// BottomLeft is the offset to the intersection of the bottom and left edges.
//
// Flutter: ui/geometry.dart Rect.bottomLeft
func (r Rect) BottomLeft() Offset { return Offset{r.Left, r.Bottom} }

// BottomCenter is the offset to the center of the bottom edge.
//
// Flutter: ui/geometry.dart Rect.bottomCenter
func (r Rect) BottomCenter() Offset { return Offset{r.Left + r.Width()/2.0, r.Bottom} }

// BottomRight is the offset to the intersection of the bottom and right edges.
//
// Flutter: ui/geometry.dart Rect.bottomRight
func (r Rect) BottomRight() Offset { return Offset{r.Right, r.Bottom} }

// Contains reports whether the point is inside the rectangle. The left and top
// edges are inside, the right and bottom edges are not.
//
// Flutter: ui/geometry.dart Rect.contains
func (r Rect) Contains(offset Offset) bool {
	return offset.Dx >= r.Left && offset.Dx < r.Right && offset.Dy >= r.Top && offset.Dy < r.Bottom
}

// Lerp linearly interpolates between r and b: r when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart Rect.lerp
func (r Rect) Lerp(b Rect, t float64) Rect {
	return Rect{
		lerpDouble(r.Left, b.Left, t),
		lerpDouble(r.Top, b.Top, t),
		lerpDouble(r.Right, b.Right, t),
		lerpDouble(r.Bottom, b.Bottom, t),
	}
}

// RectLerp linearly interpolates between two rectangles, either of which may be
// nil. A nil rectangle counts as an empty one at the origin, and two nils give
// nil.
//
// Flutter: ui/geometry.dart Rect.lerp
func RectLerp(a, b *Rect, t float64) *Rect {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		k := 1.0 - t
		return &Rect{a.Left * k, a.Top * k, a.Right * k, a.Bottom * k}
	case a == nil:
		return &Rect{b.Left * t, b.Top * t, b.Right * t, b.Bottom * t}
	}
	r := a.Lerp(*b, t)
	return &r
}

// String describes the rectangle, as Dart's toString does.
//
// Flutter: ui/geometry.dart Rect.toString
func (r Rect) String() string {
	return fmt.Sprintf("Rect.fromLTRB(%s, %s, %s, %s)",
		toStringAsFixed1(r.Left), toStringAsFixed1(r.Top), toStringAsFixed1(r.Right), toStringAsFixed1(r.Bottom))
}

// Radius is a radius of a circle or ellipse.
//
// Flutter: ui/geometry.dart Radius
type Radius struct {
	// X is the radius value on the horizontal axis.
	X float64
	// Y is the radius value on the vertical axis.
	Y float64
}

// RadiusZero is a radius with zero extent on both axes.
//
// Flutter: ui/geometry.dart Radius.zero
var RadiusZero = Radius{}

// NewRadiusCircular constructs a circular radius: X and Y are both radius.
//
// Flutter: ui/geometry.dart Radius.circular
func NewRadiusCircular(radius float64) Radius { return Radius{radius, radius} }

// NewRadiusElliptical constructs an elliptical radius with the given radii on
// the horizontal and vertical axes.
//
// Flutter: ui/geometry.dart Radius.elliptical
func NewRadiusElliptical(x, y float64) Radius { return Radius{x, y} }

// RadiusClampOpts are the optional arguments of Radius.Clamp. A nil bound
// leaves that side open.
//
// Flutter: ui/geometry.dart Radius.clamp
type RadiusClampOpts struct {
	// Minimum is the lower bound for both axes; nil means no lower bound.
	Minimum *Radius
	// Maximum is the upper bound for both axes; nil means no upper bound.
	Maximum *Radius
}

// Clamp returns this radius with each axis limited to the bounds. Passing more
// than one options value panics.
//
// Flutter: ui/geometry.dart Radius.clamp
func (r Radius) Clamp(opts ...RadiusClampOpts) Radius {
	o := oneOpts(opts)
	minimum := Radius{math.Inf(-1), math.Inf(-1)}
	maximum := Radius{math.Inf(1), math.Inf(1)}
	if o.Minimum != nil {
		minimum = *o.Minimum
	}
	if o.Maximum != nil {
		maximum = *o.Maximum
	}
	return Radius{ClampDouble(r.X, minimum.X, maximum.X), ClampDouble(r.Y, minimum.Y, maximum.Y)}
}

// RadiusClampValuesOpts are the optional arguments of Radius.ClampValues. A nil
// bound leaves that side open.
//
// Flutter: ui/geometry.dart Radius.clampValues
type RadiusClampValuesOpts struct {
	// MinimumX is the lower bound for the horizontal axis.
	MinimumX *float64
	// MinimumY is the lower bound for the vertical axis.
	MinimumY *float64
	// MaximumX is the upper bound for the horizontal axis.
	MaximumX *float64
	// MaximumY is the upper bound for the vertical axis.
	MaximumY *float64
}

// ClampValues returns this radius with each axis limited to its bounds. Passing
// more than one options value panics.
//
// Flutter: ui/geometry.dart Radius.clampValues
func (r Radius) ClampValues(opts ...RadiusClampValuesOpts) Radius {
	o := oneOpts(opts)
	bound := func(v *float64, open float64) float64 {
		if v == nil {
			return open
		}
		return *v
	}
	return Radius{
		ClampDouble(r.X, bound(o.MinimumX, math.Inf(-1)), bound(o.MaximumX, math.Inf(1))),
		ClampDouble(r.Y, bound(o.MinimumY, math.Inf(-1)), bound(o.MaximumY, math.Inf(1))),
	}
}

// Neg returns the radius with both axes negated (Dart's unary -).
//
// Flutter: ui/geometry.dart Radius.operator -
func (r Radius) Neg() Radius { return Radius{-r.X, -r.Y} }

// Sub returns the difference of two radii (Dart's -).
//
// Flutter: ui/geometry.dart Radius.operator -
func (r Radius) Sub(other Radius) Radius { return Radius{r.X - other.X, r.Y - other.Y} }

// Add returns the sum of two radii (Dart's +).
//
// Flutter: ui/geometry.dart Radius.operator +
func (r Radius) Add(other Radius) Radius { return Radius{r.X + other.X, r.Y + other.Y} }

// Mul multiplies both axes by operand (Dart's *).
//
// Flutter: ui/geometry.dart Radius.operator *
func (r Radius) Mul(operand float64) Radius { return Radius{r.X * operand, r.Y * operand} }

// Div divides both axes by operand (Dart's /).
//
// Flutter: ui/geometry.dart Radius.operator /
func (r Radius) Div(operand float64) Radius { return Radius{r.X / operand, r.Y / operand} }

// TruncDiv divides both axes by operand and truncates the results (Dart's ~/).
//
// Deviation: an infinite or NaN result stays so, where Dart throws.
//
// Flutter: ui/geometry.dart Radius.operator ~/
func (r Radius) TruncDiv(operand float64) Radius {
	return Radius{math.Trunc(r.X / operand), math.Trunc(r.Y / operand)}
}

// Mod returns the remainder of dividing both axes by operand, made non-negative
// as Dart does (Dart's %).
//
// Flutter: ui/geometry.dart Radius.operator %
func (r Radius) Mod(operand float64) Radius {
	return Radius{dartMod(r.X, operand), dartMod(r.Y, operand)}
}

// Lerp linearly interpolates between r and b: r when t is 0, b when t is 1.
//
// Flutter: ui/geometry.dart Radius.lerp
func (r Radius) Lerp(b Radius, t float64) Radius {
	return Radius{lerpDouble(r.X, b.X, t), lerpDouble(r.Y, b.Y, t)}
}

// RadiusLerp linearly interpolates between two radii, either of which may be
// nil. A nil radius counts as zero, and two nils give nil.
//
// Flutter: ui/geometry.dart Radius.lerp
func RadiusLerp(a, b *Radius, t float64) *Radius {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		k := 1.0 - t
		return &Radius{a.X * k, a.Y * k}
	case a == nil:
		return &Radius{b.X * t, b.Y * t}
	}
	r := a.Lerp(*b, t)
	return &r
}

// String describes the radius, as Dart's toString does.
//
// Flutter: ui/geometry.dart Radius.toString
func (r Radius) String() string {
	if r.X == r.Y {
		return fmt.Sprintf("Radius.circular(%s)", toStringAsFixed1(r.X))
	}
	return fmt.Sprintf("Radius.elliptical(%s, %s)", toStringAsFixed1(r.X), toStringAsFixed1(r.Y))
}

// isFinite reports whether x is neither infinite nor NaN.
func isFinite(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) }

// oneOpts returns the one options value, or the zero value when there is none.
// More than one panics with the tag MultipleOptions (SPEC §12.1).
//
// TODO(step 5.1): use the shared opts.First helper.
func oneOpts[T any](opts []T) T {
	var zero T
	switch len(opts) {
	case 0:
		return zero
	case 1:
		return opts[0]
	}
	panic(errors.New("MultipleOptions", "more than one options value was passed"))
}
