package vecmath

import "math"

// Quat is a quaternion x*i + y*j + z*k + w. A rotation is a quaternion of length 1 (the identity has w = 1), and a.Mul(b)
// applies b first, as for matrices.
//
// Flutter: package:vector_math Quaternion
type Quat struct {
	// X is the i component.
	X float64
	// Y is the j component.
	Y float64
	// Z is the k component.
	Z float64
	// W is the real component.
	W float64
}

// QuatIdentity is the rotation that does nothing.
//
// Flutter: package:vector_math Quaternion.identity
func QuatIdentity() Quat { return Quat{W: 1} }

// NewQuatAxisAngle creates the rotation of the angle, in radians, around the axis (which need not have length 1). An axis
// without length gives the identity.
//
// Flutter: package:vector_math Quaternion.axisAngle
func NewQuatAxisAngle(axis Vec3, angle float64) Quat {
	l := axis.Length()
	if l == 0 {
		return QuatIdentity()
	}
	s, c := math.Sincos(angle / 2)
	k := s / l
	return Quat{axis.X * k, axis.Y * k, axis.Z * k, c}
}

// NewQuatFromTo creates the shortest rotation that turns the direction from into the direction to. Directions without
// length give the identity.
//
// Flutter: package:vector_math Quaternion.fromTwoVectors
func NewQuatFromTo(from, to Vec3) Quat {
	a, b := from.Normalize(), to.Normalize()
	if a.LengthSq() == 0 || b.LengthSq() == 0 {
		return QuatIdentity()
	}
	d := a.Dot(b)
	if d < -1+1e-12 {
		// Opposite directions: a half turn around any axis that is perpendicular to them.
		return NewQuatAxisAngle(a.Perpendicular(), math.Pi)
	}
	c := a.Cross(b)
	return Quat{c.X, c.Y, c.Z, 1 + d}.Normalize()
}

// NewQuatBasis creates the rotation that turns the x, y and z axes into the three vectors, which must be perpendicular
// and of length 1.
//
// Flutter: no counterpart
func NewQuatBasis(x, y, z Vec3) Quat {
	// The matrix with the vectors as its columns, as a quaternion (Shepperd's method).
	m00, m10, m20 := x.X, x.Y, x.Z
	m01, m11, m21 := y.X, y.Y, y.Z
	m02, m12, m22 := z.X, z.Y, z.Z
	var q Quat
	switch tr := m00 + m11 + m22; {
	case tr > 0:
		s := math.Sqrt(tr+1) * 2
		q = Quat{(m21 - m12) / s, (m02 - m20) / s, (m10 - m01) / s, s / 4}
	case m00 > m11 && m00 > m22:
		s := math.Sqrt(1+m00-m11-m22) * 2
		q = Quat{s / 4, (m01 + m10) / s, (m02 + m20) / s, (m21 - m12) / s}
	case m11 > m22:
		s := math.Sqrt(1+m11-m00-m22) * 2
		q = Quat{(m01 + m10) / s, s / 4, (m12 + m21) / s, (m02 - m20) / s}
	default:
		s := math.Sqrt(1+m22-m00-m11) * 2
		q = Quat{(m02 + m20) / s, (m12 + m21) / s, s / 4, (m10 - m01) / s}
	}
	return q.Normalize()
}

// Mul returns the product: the rotation that applies b first, then a.
//
// Flutter: package:vector_math Quaternion.operator*
func (a Quat) Mul(b Quat) Quat {
	return Quat{
		a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
		a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
		a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
		a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
	}
}

// Conjugate returns the quaternion with the vector part negated, which is the inverse of a rotation.
//
// Flutter: package:vector_math Quaternion.conjugated
func (a Quat) Conjugate() Quat { return Quat{-a.X, -a.Y, -a.Z, a.W} }

// Dot returns the dot product, as of 4D vectors.
//
// Flutter: package:vector_math Quaternion.dot
func (a Quat) Dot(b Quat) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W }

// Length returns the length.
//
// Flutter: package:vector_math Quaternion.length
func (a Quat) Length() float64 { return math.Sqrt(a.Dot(a)) }

// Normalize returns the quaternion of length 1 in the direction of a, or the identity when a has no length.
//
// Flutter: package:vector_math Quaternion.normalized
func (a Quat) Normalize() Quat {
	l := a.Length()
	if l == 0 {
		return QuatIdentity()
	}
	return Quat{a.X / l, a.Y / l, a.Z / l, a.W / l}
}

// Inverse returns the inverse of the quaternion, which is the conjugate for a rotation. A quaternion without length
// gives the identity.
//
// Flutter: package:vector_math Quaternion.inverted
func (a Quat) Inverse() Quat {
	d := a.Dot(a)
	if d == 0 {
		return QuatIdentity()
	}
	return Quat{-a.X / d, -a.Y / d, -a.Z / d, a.W / d}
}

// Rotate returns the vector turned by the rotation.
//
// Flutter: package:vector_math Quaternion.rotate
func (a Quat) Rotate(v Vec3) Vec3 {
	// v + 2w(u x v) + 2u x (u x v), with u the vector part.
	u := Vec3{a.X, a.Y, a.Z}
	t := u.Cross(v).Scale(2)
	return v.Add(t.Scale(a.W)).Add(u.Cross(t))
}

// Slerp returns the rotation at t of the way from a to b on the shortest arc.
//
// Flutter: no counterpart
func (a Quat) Slerp(b Quat, t float64) Quat {
	d := a.Dot(b)
	if d < 0 {
		b = Quat{-b.X, -b.Y, -b.Z, -b.W}
		d = -d
	}
	if d > 1-1e-10 {
		// Nearly the same rotation: a straight line is as good, and it does not divide by a tiny sine.
		return Quat{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t, a.W + (b.W-a.W)*t}.Normalize()
	}
	theta := math.Acos(d)
	s := math.Sin(theta)
	ka, kb := math.Sin((1-t)*theta)/s, math.Sin(t*theta)/s
	return Quat{a.X*ka + b.X*kb, a.Y*ka + b.Y*kb, a.Z*ka + b.Z*kb, a.W*ka + b.W*kb}
}

// AxisAngle returns the axis, of length 1, and the angle, from 0 to 2 pi, of the rotation. The identity has the axis
// (1, 0, 0) and the angle 0.
//
// Flutter: package:vector_math Quaternion.radians
func (a Quat) AxisAngle() (Vec3, float64) {
	a = a.Normalize()
	s := math.Sqrt(1 - a.W*a.W)
	if s < 1e-12 {
		return Vec3{X: 1}, 0
	}
	return Vec3{a.X / s, a.Y / s, a.Z / s}, 2 * math.Acos(math.Max(-1, math.Min(1, a.W)))
}

// Mat3 returns the rotation as a matrix. The quaternion should have length 1.
//
// Flutter: package:vector_math Quaternion.asRotationMatrix
func (a Quat) Mat3() Mat3 {
	x2, y2, z2 := a.X+a.X, a.Y+a.Y, a.Z+a.Z
	xx, xy, xz := a.X*x2, a.X*y2, a.X*z2
	yy, yz, zz := a.Y*y2, a.Y*z2, a.Z*z2
	wx, wy, wz := a.W*x2, a.W*y2, a.W*z2
	return Mat3{
		1 - (yy + zz), xy + wz, xz - wy,
		xy - wz, 1 - (xx + zz), yz + wx,
		xz + wy, yz - wx, 1 - (xx + yy),
	}
}

// Mat4 returns the rotation as a matrix. The quaternion should have length 1.
//
// Flutter: no counterpart
func (a Quat) Mat4() Mat4 {
	r := a.Mat3()
	return Mat4{
		r[0], r[1], r[2], 0,
		r[3], r[4], r[5], 0,
		r[6], r[7], r[8], 0,
		0, 0, 0, 1,
	}
}
