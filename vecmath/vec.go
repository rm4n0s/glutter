package vecmath

import "math"

// Vec2 is a 2D vector.
//
// Flutter: package:vector_math Vector2
type Vec2 struct {
	// X is the first component.
	X float64
	// Y is the second component.
	Y float64
}

// Vec3 is a 3D vector or point.
//
// Flutter: package:vector_math Vector3
type Vec3 struct {
	// X is the first component.
	X float64
	// Y is the second component.
	Y float64
	// Z is the third component.
	Z float64
}

// Vec4 is a 4D vector, or a point in homogeneous coordinates.
//
// Flutter: package:vector_math Vector4
type Vec4 struct {
	// X is the first component.
	X float64
	// Y is the second component.
	Y float64
	// Z is the third component.
	Z float64
	// W is the fourth component.
	W float64
}

// The unit vectors and the zero vector. They are values, so no program can change them.
var (
	// Vec3Zero is the vector (0, 0, 0).
	//
	// Flutter: no counterpart
	Vec3Zero = Vec3{}
	// Vec3X is the unit vector of the x axis.
	//
	// Flutter: no counterpart
	Vec3X = Vec3{X: 1}
	// Vec3Y is the unit vector of the y axis.
	//
	// Flutter: no counterpart
	Vec3Y = Vec3{Y: 1}
	// Vec3Z is the unit vector of the z axis.
	//
	// Flutter: no counterpart
	Vec3Z = Vec3{Z: 1}
)

// NewVec2 creates the vector.
//
// Flutter: package:vector_math Vector2.new
func NewVec2(x, y float64) Vec2 { return Vec2{x, y} }

// NewVec3 creates the vector.
//
// Flutter: package:vector_math Vector3.new
func NewVec3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

// NewVec4 creates the vector.
//
// Flutter: package:vector_math Vector4.new
func NewVec4(x, y, z, w float64) Vec4 { return Vec4{x, y, z, w} }

// Add returns the sum of the vectors.
//
// Flutter: package:vector_math Vector2.operator+
func (a Vec2) Add(b Vec2) Vec2 { return Vec2{a.X + b.X, a.Y + b.Y} }

// Sub returns the difference of the vectors.
//
// Flutter: package:vector_math Vector2.operator-
func (a Vec2) Sub(b Vec2) Vec2 { return Vec2{a.X - b.X, a.Y - b.Y} }

// Scale returns the vector times the number.
//
// Flutter: package:vector_math Vector2.scale
func (a Vec2) Scale(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }

// Neg returns the opposite vector.
//
// Flutter: package:vector_math Vector2.operator-
func (a Vec2) Neg() Vec2 { return Vec2{-a.X, -a.Y} }

// Dot returns the dot product.
//
// Flutter: package:vector_math Vector2.dot
func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }

// Cross returns the z component of the cross product of the vectors seen as 3D.
//
// Flutter: package:vector_math Vector2.cross
func (a Vec2) Cross(b Vec2) float64 { return a.X*b.Y - a.Y*b.X }

// LengthSq returns the squared length.
//
// Flutter: package:vector_math Vector2.length2
func (a Vec2) LengthSq() float64 { return a.X*a.X + a.Y*a.Y }

// Length returns the length.
//
// Flutter: package:vector_math Vector2.length
func (a Vec2) Length() float64 { return math.Hypot(a.X, a.Y) }

// Normalize returns the vector of length 1 with the direction of a, or the zero vector when a has no length.
//
// Flutter: package:vector_math Vector2.normalized
func (a Vec2) Normalize() Vec2 {
	l := a.Length()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{a.X / l, a.Y / l}
}

// Lerp returns the point at t of the way from a to b.
//
// Flutter: package:vector_math Vector2.mix
func (a Vec2) Lerp(b Vec2, t float64) Vec2 { return Vec2{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t} }

// Float32 returns the components as float32.
//
// Flutter: no counterpart
func (a Vec2) Float32() [2]float32 { return [2]float32{float32(a.X), float32(a.Y)} }

// Add returns the sum of the vectors.
//
// Flutter: package:vector_math Vector3.operator+
func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// Sub returns the difference of the vectors.
//
// Flutter: package:vector_math Vector3.operator-
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

// Scale returns the vector times the number.
//
// Flutter: package:vector_math Vector3.scale
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

// MulComponents returns the product of the components, one by one.
//
// Flutter: package:vector_math Vector3.operator*
func (a Vec3) MulComponents(b Vec3) Vec3 { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }

// Neg returns the opposite vector.
//
// Flutter: package:vector_math Vector3.operator-
func (a Vec3) Neg() Vec3 { return Vec3{-a.X, -a.Y, -a.Z} }

// Dot returns the dot product.
//
// Flutter: package:vector_math Vector3.dot
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// Cross returns the cross product, which is perpendicular to both vectors, with the right-hand rule.
//
// Flutter: package:vector_math Vector3.cross
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// LengthSq returns the squared length.
//
// Flutter: package:vector_math Vector3.length2
func (a Vec3) LengthSq() float64 { return a.X*a.X + a.Y*a.Y + a.Z*a.Z }

// Length returns the length.
//
// Flutter: package:vector_math Vector3.length
func (a Vec3) Length() float64 { return math.Sqrt(a.LengthSq()) }

// Normalize returns the vector of length 1 with the direction of a, or the zero vector when a has no length.
//
// Flutter: package:vector_math Vector3.normalized
func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l == 0 {
		return Vec3{}
	}
	return Vec3{a.X / l, a.Y / l, a.Z / l}
}

// DistanceTo returns the distance between the points.
//
// Flutter: package:vector_math Vector3.distanceTo
func (a Vec3) DistanceTo(b Vec3) float64 { return a.Sub(b).Length() }

// DistanceSqTo returns the squared distance between the points.
//
// Flutter: package:vector_math Vector3.distanceToSquared
func (a Vec3) DistanceSqTo(b Vec3) float64 { return a.Sub(b).LengthSq() }

// Lerp returns the point at t of the way from a to b.
//
// Flutter: package:vector_math Vector3.mix
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return Vec3{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t}
}

// Min returns the smallest of each component.
//
// Flutter: package:vector_math Vector3.min
func (a Vec3) Min(b Vec3) Vec3 {
	return Vec3{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)}
}

// Max returns the largest of each component.
//
// Flutter: package:vector_math Vector3.max
func (a Vec3) Max(b Vec3) Vec3 {
	return Vec3{math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)}
}

// AngleTo returns the angle between the vectors, from 0 to pi. It is 0 when one of them has no length.
//
// Flutter: package:vector_math Vector3.angleTo
func (a Vec3) AngleTo(b Vec3) float64 {
	d := a.Length() * b.Length()
	if d == 0 {
		return 0
	}
	return math.Acos(math.Max(-1, math.Min(1, a.Dot(b)/d)))
}

// ProjectOnto returns the part of a that is along b, or the zero vector when b has no length.
//
// Flutter: no counterpart
func (a Vec3) ProjectOnto(b Vec3) Vec3 {
	d := b.LengthSq()
	if d == 0 {
		return Vec3{}
	}
	return b.Scale(a.Dot(b) / d)
}

// Reflect returns the vector reflected by the surface with the normal n, which has length 1.
//
// Flutter: package:vector_math Vector3.reflect
func (a Vec3) Reflect(n Vec3) Vec3 { return a.Sub(n.Scale(2 * a.Dot(n))) }

// Perpendicular returns a unit vector that is perpendicular to a, or the zero vector when a has no length.
//
// Flutter: no counterpart
func (a Vec3) Perpendicular() Vec3 {
	if a.LengthSq() == 0 {
		return Vec3{}
	}
	// Cross with the axis that a is the least along: the result is never short.
	ax, ay, az := math.Abs(a.X), math.Abs(a.Y), math.Abs(a.Z)
	switch {
	case ax <= ay && ax <= az:
		return a.Cross(Vec3{X: 1}).Normalize()
	case ay <= az:
		return a.Cross(Vec3{Y: 1}).Normalize()
	}
	return a.Cross(Vec3{Z: 1}).Normalize()
}

// NearlyEqual says whether every component of a is within eps of b's.
//
// Flutter: no counterpart
func (a Vec3) NearlyEqual(b Vec3, eps float64) bool {
	return math.Abs(a.X-b.X) <= eps && math.Abs(a.Y-b.Y) <= eps && math.Abs(a.Z-b.Z) <= eps
}

// IsFinite says whether no component is infinite or NaN.
//
// Flutter: no counterpart
func (a Vec3) IsFinite() bool {
	return !math.IsNaN(a.X+a.Y+a.Z) && !math.IsInf(a.X+a.Y+a.Z, 0)
}

// Vec4 makes a vector of four components with w.
//
// Flutter: no counterpart
func (a Vec3) Vec4(w float64) Vec4 { return Vec4{a.X, a.Y, a.Z, w} }

// Float32 returns the components as float32.
//
// Flutter: no counterpart
func (a Vec3) Float32() [3]float32 { return [3]float32{float32(a.X), float32(a.Y), float32(a.Z)} }

// Add returns the sum of the vectors.
//
// Flutter: package:vector_math Vector4.operator+
func (a Vec4) Add(b Vec4) Vec4 { return Vec4{a.X + b.X, a.Y + b.Y, a.Z + b.Z, a.W + b.W} }

// Sub returns the difference of the vectors.
//
// Flutter: package:vector_math Vector4.operator-
func (a Vec4) Sub(b Vec4) Vec4 { return Vec4{a.X - b.X, a.Y - b.Y, a.Z - b.Z, a.W - b.W} }

// Scale returns the vector times the number.
//
// Flutter: package:vector_math Vector4.scale
func (a Vec4) Scale(s float64) Vec4 { return Vec4{a.X * s, a.Y * s, a.Z * s, a.W * s} }

// Dot returns the dot product.
//
// Flutter: package:vector_math Vector4.dot
func (a Vec4) Dot(b Vec4) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W }

// Lerp returns the point at t of the way from a to b.
//
// Flutter: package:vector_math Vector4.mix
func (a Vec4) Lerp(b Vec4, t float64) Vec4 {
	return Vec4{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t, a.W + (b.W-a.W)*t}
}

// XYZ returns the first three components.
//
// Flutter: package:vector_math Vector4.xyz
func (a Vec4) XYZ() Vec3 { return Vec3{a.X, a.Y, a.Z} }

// PerspectiveDivide returns the first three components divided by w: the point of normalized device coordinates that
// a clip-space point stands for. A w of 0 gives infinities.
//
// Flutter: no counterpart
func (a Vec4) PerspectiveDivide() Vec3 { return Vec3{a.X / a.W, a.Y / a.W, a.Z / a.W} }

// Float32 returns the components as float32.
//
// Flutter: no counterpart
func (a Vec4) Float32() [4]float32 {
	return [4]float32{float32(a.X), float32(a.Y), float32(a.Z), float32(a.W)}
}
