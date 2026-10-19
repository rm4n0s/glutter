package ui

import "math"

// Quaternion is a rotation as a quaternion with float64 components.
//
// Flutter: package:vector_math Quaternion
type Quaternion struct {
	// X is the first imaginary component.
	X float64
	// Y is the second imaginary component.
	Y float64
	// Z is the third imaginary component.
	Z float64
	// W is the real component.
	W float64
}

// NewQuaternionIdentity returns the quaternion that does not rotate.
//
// Flutter: package:vector_math Quaternion.identity
func NewQuaternionIdentity() Quaternion { return Quaternion{W: 1} }

// Scaled returns the quaternion with every component multiplied by scale.
//
// Flutter: package:vector_math Quaternion.scaled
func (q Quaternion) Scaled(scale float64) Quaternion {
	return Quaternion{q.X * scale, q.Y * scale, q.Z * scale, q.W * scale}
}

// Add returns the sum of the components of the two quaternions.
//
// Flutter: package:vector_math Quaternion.+
func (q Quaternion) Add(o Quaternion) Quaternion {
	return Quaternion{q.X + o.X, q.Y + o.Y, q.Z + o.Z, q.W + o.W}
}

// Length is the length of the quaternion as a four dimensional vector.
//
// Flutter: package:vector_math Quaternion.length
func (q Quaternion) Length() float64 {
	return math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)
}

// Normalized returns the quaternion scaled to length 1; a quaternion of length
// 0 is returned as it is.
//
// Flutter: package:vector_math Quaternion.normalized
func (q Quaternion) Normalized() Quaternion {
	l := q.Length()
	if l == 0 {
		return q
	}
	return q.Scaled(1 / l)
}

// NewQuaternionFromRotation returns the quaternion of the 3x3 rotation matrix,
// which is given in column-major order (the entry of row r and column c is
// m[c*3+r]).
//
// Flutter: package:vector_math Quaternion.setFromRotation
func NewQuaternionFromRotation(m [9]float64) Quaternion {
	var q [4]float64
	trace := m[0] + m[4] + m[8]
	if trace > 0 {
		s := math.Sqrt(trace + 1)
		q[3] = s * 0.5
		s = 0.5 / s
		q[0] = (m[5] - m[7]) * s
		q[1] = (m[6] - m[2]) * s
		q[2] = (m[1] - m[3]) * s
	} else {
		var i int
		if m[0] < m[4] {
			if m[4] < m[8] {
				i = 2
			} else {
				i = 1
			}
		} else if m[0] < m[8] {
			i = 2
		}
		j, k := (i+1)%3, (i+2)%3
		idx := func(r, c int) int { return c*3 + r }
		s := math.Sqrt(m[idx(i, i)] - m[idx(j, j)] - m[idx(k, k)] + 1)
		q[i] = s * 0.5
		s = 0.5 / s
		q[3] = (m[idx(k, j)] - m[idx(j, k)]) * s
		q[j] = (m[idx(j, i)] + m[idx(i, j)]) * s
		q[k] = (m[idx(k, i)] + m[idx(i, k)]) * s
	}
	return Quaternion{q[0], q[1], q[2], q[3]}
}

// Decompose splits m into its translation, rotation and scale. A matrix with a
// negative determinant has a negative x scale.
//
// Flutter: package:vector_math Matrix4.decompose
func (m *Matrix4) Decompose() (translation Vector3, rotation Quaternion, scale Vector3) {
	s := &m.storage
	col := func(i int) (float64, float64, float64) {
		return float64(s[i]), float64(s[i+1]), float64(s[i+2])
	}
	length := func(x, y, z float64) float64 { return math.Sqrt(x*x + y*y + z*z) }
	x0, y0, z0 := col(0)
	x1, y1, z1 := col(4)
	x2, y2, z2 := col(8)
	sx, sy, sz := length(x0, y0, z0), length(x1, y1, z1), length(x2, y2, z2)
	if m.Determinant() < 0 {
		sx = -sx
	}
	translation = Vector3{s[12], s[13], s[14]}
	rot := [9]float64{
		x0 / sx, y0 / sx, z0 / sx,
		x1 / sy, y1 / sy, z1 / sy,
		x2 / sz, y2 / sz, z2 / sz,
	}
	// copyRotation normalizes each column again by its own length.
	for c := 0; c < 3; c++ {
		l := length(rot[c*3], rot[c*3+1], rot[c*3+2])
		for r := 0; r < 3; r++ {
			rot[c*3+r] /= l
		}
	}
	return translation, NewQuaternionFromRotation(rot), Vector3{float32(sx), float32(sy), float32(sz)}
}

// NewMatrix4Compose returns the matrix of the translation, rotation and scale
// (the inverse of Decompose).
//
// Flutter: package:vector_math Matrix4.compose
func NewMatrix4Compose(translation Vector3, rotation Quaternion, scale Vector3) *Matrix4 {
	x, y, z, w := rotation.X, rotation.Y, rotation.Z, rotation.W
	x2, y2, z2 := x+x, y+y, z+z
	xx, xy, xz := x*x2, x*y2, x*z2
	yy, yz, zz := y*y2, y*z2, z*z2
	wx, wy, wz := w*x2, w*y2, w*z2
	m := &Matrix4{[16]float32{
		float32(1 - (yy + zz)), float32(xy + wz), float32(xz - wy), 0,
		float32(xy - wz), float32(1 - (xx + zz)), float32(yz + wx), 0,
		float32(xz + wy), float32(yz - wx), float32(1 - (xx + yy)), 0,
		translation.X, translation.Y, translation.Z, 1,
	}}
	m.Scale(scale.X, scale.Y, scale.Z)
	return m
}
