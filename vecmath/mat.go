package vecmath

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Mat3 is a 3x3 matrix, stored column-major: the entry in row r and column c is m[c*3+r].
//
// Flutter: package:vector_math Matrix3
type Mat3 [9]float64

// Mat4 is a 4x4 matrix, stored column-major as WGSL reads it: the entry in row r and column c is m[c*4+r]. It applies
// to column vectors, so a.Mul(b) applies b first.
//
// Flutter: package:vector_math Matrix4
type Mat4 [16]float64

// Mat3Identity returns the identity matrix.
//
// Flutter: package:vector_math Matrix3.identity
func Mat3Identity() Mat3 { return Mat3{1, 0, 0, 0, 1, 0, 0, 0, 1} }

// Mul returns the product: the matrix that applies b first, then a.
//
// Flutter: package:vector_math Matrix3.operator*
func (a Mat3) Mul(b Mat3) Mat3 {
	var r Mat3
	for c := 0; c < 3; c++ {
		for row := 0; row < 3; row++ {
			r[c*3+row] = a[row]*b[c*3] + a[3+row]*b[c*3+1] + a[6+row]*b[c*3+2]
		}
	}
	return r
}

// MulVec3 returns the vector transformed by the matrix.
//
// Flutter: package:vector_math Matrix3.transform
func (a Mat3) MulVec3(v Vec3) Vec3 {
	return Vec3{
		a[0]*v.X + a[3]*v.Y + a[6]*v.Z,
		a[1]*v.X + a[4]*v.Y + a[7]*v.Z,
		a[2]*v.X + a[5]*v.Y + a[8]*v.Z,
	}
}

// Transpose returns the matrix with rows and columns exchanged.
//
// Flutter: package:vector_math Matrix3.transposed
func (a Mat3) Transpose() Mat3 {
	return Mat3{a[0], a[3], a[6], a[1], a[4], a[7], a[2], a[5], a[8]}
}

// Det returns the determinant.
//
// Flutter: package:vector_math Matrix3.determinant
func (a Mat3) Det() float64 {
	return a[0]*(a[4]*a[8]-a[5]*a[7]) - a[3]*(a[1]*a[8]-a[2]*a[7]) + a[6]*(a[1]*a[5]-a[2]*a[4])
}

// Inverse returns the inverse matrix, and false when the matrix is singular (its determinant is 0 or not a number).
//
// Flutter: package:vector_math Matrix3.invert
func (a Mat3) Inverse() (Mat3, bool) {
	det := a.Det()
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return Mat3{}, false
	}
	k := 1 / det
	return Mat3{
		(a[4]*a[8] - a[5]*a[7]) * k, (a[2]*a[7] - a[1]*a[8]) * k, (a[1]*a[5] - a[2]*a[4]) * k,
		(a[5]*a[6] - a[3]*a[8]) * k, (a[0]*a[8] - a[2]*a[6]) * k, (a[2]*a[3] - a[0]*a[5]) * k,
		(a[3]*a[7] - a[4]*a[6]) * k, (a[1]*a[6] - a[0]*a[7]) * k, (a[0]*a[4] - a[1]*a[3]) * k,
	}, true
}

// Float32 returns the entries as float32, in the same order.
//
// Flutter: no counterpart
func (a Mat3) Float32() [9]float32 {
	var r [9]float32
	for i, v := range a {
		r[i] = float32(v)
	}
	return r
}

// Mat4Identity returns the identity matrix.
//
// Flutter: package:vector_math Matrix4.identity
func Mat4Identity() Mat4 {
	return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// NewMat4Translation creates the matrix that moves points by v.
//
// Flutter: package:vector_math Matrix4.translation
func NewMat4Translation(v Vec3) Mat4 {
	return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, v.X, v.Y, v.Z, 1}
}

// NewMat4Scale creates the matrix that scales by v along the axes.
//
// Flutter: package:vector_math Matrix4.diagonal3
func NewMat4Scale(v Vec3) Mat4 {
	return Mat4{v.X, 0, 0, 0, 0, v.Y, 0, 0, 0, 0, v.Z, 0, 0, 0, 0, 1}
}

// NewMat4RotationX creates the rotation around the x axis by the angle in radians.
//
// Flutter: package:vector_math Matrix4.rotationX
func NewMat4RotationX(angle float64) Mat4 {
	s, c := math.Sincos(angle)
	return Mat4{1, 0, 0, 0, 0, c, s, 0, 0, -s, c, 0, 0, 0, 0, 1}
}

// NewMat4RotationY creates the rotation around the y axis by the angle in radians.
//
// Flutter: package:vector_math Matrix4.rotationY
func NewMat4RotationY(angle float64) Mat4 {
	s, c := math.Sincos(angle)
	return Mat4{c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1}
}

// NewMat4RotationZ creates the rotation around the z axis by the angle in radians.
//
// Flutter: package:vector_math Matrix4.rotationZ
func NewMat4RotationZ(angle float64) Mat4 {
	s, c := math.Sincos(angle)
	return Mat4{c, s, 0, 0, -s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// NewMat4TRS creates the matrix that scales, then rotates, then moves.
//
// Flutter: package:vector_math Matrix4.compose
func NewMat4TRS(translation Vec3, rotation Quat, scale Vec3) Mat4 {
	r := rotation.Mat3()
	return Mat4{
		r[0] * scale.X, r[1] * scale.X, r[2] * scale.X, 0,
		r[3] * scale.Y, r[4] * scale.Y, r[5] * scale.Y, 0,
		r[6] * scale.Z, r[7] * scale.Z, r[8] * scale.Z, 0,
		translation.X, translation.Y, translation.Z, 1,
	}
}

// NewMat4FromMatrix4 creates the matrix from a float32 matrix of the engine API.
//
// Flutter: no counterpart
func NewMat4FromMatrix4(m *ui.Matrix4) Mat4 {
	var r Mat4
	for i, v := range m.Storage() {
		r[i] = float64(v)
	}
	return r
}

// LookAt creates the view matrix of a camera at eye that looks at center, with up as its upward direction: the matrix
// that turns world space into the space of the camera, which looks down -z. It returns the identity when eye and
// center are the same point, or when up is along the view.
//
// Flutter: package:vector_math setViewMatrix
func LookAt(eye, center, up Vec3) Mat4 {
	z := eye.Sub(center).Normalize()
	x := up.Cross(z).Normalize()
	if z.LengthSq() == 0 || x.LengthSq() == 0 {
		return Mat4Identity()
	}
	y := z.Cross(x)
	return Mat4{
		x.X, y.X, z.X, 0,
		x.Y, y.Y, z.Y, 0,
		x.Z, y.Z, z.Z, 0,
		-x.Dot(eye), -y.Dot(eye), -z.Dot(eye), 1,
	}
}

// Perspective creates a projection with the vertical field of view fovY, in radians, the aspect (width over height) and
// the near plane, in WebGPU's clip space with reversed-Z and the far plane at infinity: a point at the near plane gets the
// depth 1 and a point infinitely far gets 0.
//
// Flutter: package:vector_math setPerspectiveMatrix
func Perspective(fovY, aspect, near float64) Mat4 {
	f := 1 / math.Tan(fovY/2)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, 0, -1,
		0, 0, near, 0,
	}
}

// PerspectiveFinite is Perspective with a far plane: the depth is 1 at near and 0 at far.
//
// Flutter: package:vector_math setPerspectiveMatrix
func PerspectiveFinite(fovY, aspect, near, far float64) Mat4 {
	f := 1 / math.Tan(fovY/2)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, near / (far - near), -1,
		0, 0, near * far / (far - near), 0,
	}
}

// Orthographic creates a projection of the box that goes from left to right, bottom to top, and from near to far in
// front of the eye, in WebGPU's clip space with reversed-Z: the depth is 1 at near and 0 at far. Near may be negative
// (a plane behind the eye).
//
// Flutter: package:vector_math setOrthographicMatrix
func Orthographic(left, right, bottom, top, near, far float64) Mat4 {
	return Mat4{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, 1 / (far - near), 0,
		-(right + left) / (right - left), -(top + bottom) / (top - bottom), far / (far - near), 1,
	}
}

// Mul returns the product: the matrix that applies b first, then a.
//
// Flutter: package:vector_math Matrix4.operator*
func (a Mat4) Mul(b Mat4) Mat4 {
	var r Mat4
	for c := 0; c < 4; c++ {
		b0, b1, b2, b3 := b[c*4], b[c*4+1], b[c*4+2], b[c*4+3]
		for row := 0; row < 4; row++ {
			r[c*4+row] = a[row]*b0 + a[4+row]*b1 + a[8+row]*b2 + a[12+row]*b3
		}
	}
	return r
}

// MulVec4 returns the vector transformed by the matrix.
//
// Flutter: package:vector_math Matrix4.transform
func (a Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		a[0]*v.X + a[4]*v.Y + a[8]*v.Z + a[12]*v.W,
		a[1]*v.X + a[5]*v.Y + a[9]*v.Z + a[13]*v.W,
		a[2]*v.X + a[6]*v.Y + a[10]*v.Z + a[14]*v.W,
		a[3]*v.X + a[7]*v.Y + a[11]*v.Z + a[15]*v.W,
	}
}

// TransformPoint returns the point moved by the matrix, with w taken as 1 and the result not divided by its w: right
// for the matrices of an object, not for a projection (use ProjectPoint).
//
// Flutter: package:vector_math Matrix4.transform3
func (a Mat4) TransformPoint(p Vec3) Vec3 {
	return Vec3{
		a[0]*p.X + a[4]*p.Y + a[8]*p.Z + a[12],
		a[1]*p.X + a[5]*p.Y + a[9]*p.Z + a[13],
		a[2]*p.X + a[6]*p.Y + a[10]*p.Z + a[14],
	}
}

// TransformDirection returns the vector turned and scaled by the matrix, without its translation.
//
// Flutter: package:vector_math Matrix4.rotate3
func (a Mat4) TransformDirection(v Vec3) Vec3 {
	return Vec3{
		a[0]*v.X + a[4]*v.Y + a[8]*v.Z,
		a[1]*v.X + a[5]*v.Y + a[9]*v.Z,
		a[2]*v.X + a[6]*v.Y + a[10]*v.Z,
	}
}

// ProjectPoint returns the point transformed by the matrix and divided by its w: the point of normalized device
// coordinates for a projection matrix, and the point behind the eye gets the negative w of the clip point in the
// second result.
//
// Flutter: package:vector_math Matrix4.perspectiveTransform
func (a Mat4) ProjectPoint(p Vec3) (ndc Vec3, w float64) {
	v := a.MulVec4(Vec4{p.X, p.Y, p.Z, 1})
	return Vec3{v.X / v.W, v.Y / v.W, v.Z / v.W}, v.W
}

// Transpose returns the matrix with rows and columns exchanged.
//
// Flutter: package:vector_math Matrix4.transposed
func (a Mat4) Transpose() Mat4 {
	return Mat4{
		a[0], a[4], a[8], a[12],
		a[1], a[5], a[9], a[13],
		a[2], a[6], a[10], a[14],
		a[3], a[7], a[11], a[15],
	}
}

// minors are the 2x2 minors that the determinant and the inverse are made of: s of the first two rows and c of the last
// two. The matrix is read as if it were row-major, which gives the transpose: the determinant is the same, and the
// inverse of the transpose is the transpose of the inverse, so the entries come out in the same order they go in.
type minors struct{ s0, s1, s2, s3, s4, s5, c0, c1, c2, c3, c4, c5 float64 }

func (a *Mat4) minors() minors {
	return minors{
		s0: a[0]*a[5] - a[4]*a[1],
		s1: a[0]*a[6] - a[4]*a[2],
		s2: a[0]*a[7] - a[4]*a[3],
		s3: a[1]*a[6] - a[5]*a[2],
		s4: a[1]*a[7] - a[5]*a[3],
		s5: a[2]*a[7] - a[6]*a[3],
		c5: a[10]*a[15] - a[14]*a[11],
		c4: a[9]*a[15] - a[13]*a[11],
		c3: a[9]*a[14] - a[13]*a[10],
		c2: a[8]*a[15] - a[12]*a[11],
		c1: a[8]*a[14] - a[12]*a[10],
		c0: a[8]*a[13] - a[12]*a[9],
	}
}

// Det returns the determinant.
//
// Flutter: package:vector_math Matrix4.determinant
func (a Mat4) Det() float64 {
	m := a.minors()
	return m.s0*m.c5 - m.s1*m.c4 + m.s2*m.c3 + m.s3*m.c2 - m.s4*m.c1 + m.s5*m.c0
}

// Inverse returns the inverse matrix, and false when the matrix is singular (its determinant is 0 or not a number).
//
// Flutter: package:vector_math Matrix4.invert
func (a Mat4) Inverse() (Mat4, bool) {
	m := a.minors()
	det := m.s0*m.c5 - m.s1*m.c4 + m.s2*m.c3 + m.s3*m.c2 - m.s4*m.c1 + m.s5*m.c0
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return Mat4{}, false
	}
	k := 1 / det
	return Mat4{
		(a[5]*m.c5 - a[6]*m.c4 + a[7]*m.c3) * k,
		(-a[1]*m.c5 + a[2]*m.c4 - a[3]*m.c3) * k,
		(a[13]*m.s5 - a[14]*m.s4 + a[15]*m.s3) * k,
		(-a[9]*m.s5 + a[10]*m.s4 - a[11]*m.s3) * k,

		(-a[4]*m.c5 + a[6]*m.c2 - a[7]*m.c1) * k,
		(a[0]*m.c5 - a[2]*m.c2 + a[3]*m.c1) * k,
		(-a[12]*m.s5 + a[14]*m.s2 - a[15]*m.s1) * k,
		(a[8]*m.s5 - a[10]*m.s2 + a[11]*m.s1) * k,

		(a[4]*m.c4 - a[5]*m.c2 + a[7]*m.c0) * k,
		(-a[0]*m.c4 + a[1]*m.c2 - a[3]*m.c0) * k,
		(a[12]*m.s4 - a[13]*m.s2 + a[15]*m.s0) * k,
		(-a[8]*m.s4 + a[9]*m.s2 - a[11]*m.s0) * k,

		(-a[4]*m.c3 + a[5]*m.c1 - a[6]*m.c0) * k,
		(a[0]*m.c3 - a[1]*m.c1 + a[2]*m.c0) * k,
		(-a[12]*m.s3 + a[13]*m.s1 - a[14]*m.s0) * k,
		(a[8]*m.s3 - a[9]*m.s1 + a[10]*m.s0) * k,
	}, true
}

// Translation returns the last column's first three entries: where the matrix moves the origin.
//
// Flutter: package:vector_math Matrix4.getTranslation
func (a Mat4) Translation() Vec3 { return Vec3{a[12], a[13], a[14]} }

// WithTranslation returns the matrix with its translation set to t.
//
// Flutter: package:vector_math Matrix4.setTranslation
func (a Mat4) WithTranslation(t Vec3) Mat4 {
	a[12], a[13], a[14] = t.X, t.Y, t.Z
	return a
}

// Mat3 returns the upper left 3x3 block: the rotation and the scale.
//
// Flutter: package:vector_math Matrix4.getRotation
func (a Mat4) Mat3() Mat3 {
	return Mat3{a[0], a[1], a[2], a[4], a[5], a[6], a[8], a[9], a[10]}
}

// NormalMatrix returns the matrix for the normals of a surface: the inverse transpose of the upper left block. It is the
// identity block when that one is singular.
//
// Flutter: package:vector_math Matrix4.copyNormalMatrix
func (a Mat4) NormalMatrix() Mat3 {
	inv, ok := a.Mat3().Inverse()
	if !ok {
		return Mat3Identity()
	}
	return inv.Transpose()
}

// Float32 returns the entries as float32, in the same order (what a uniform buffer takes).
//
// Flutter: no counterpart
func (a Mat4) Float32() [16]float32 {
	var r [16]float32
	for i, v := range a {
		r[i] = float32(v)
	}
	return r
}

// Matrix4 returns the matrix as the float32 matrix of the engine API (what a Canvas takes).
//
// Flutter: no counterpart
func (a Mat4) Matrix4() ui.Matrix4 { return *ui.NewMatrix4FromList(a.Float32()) }

// NearlyEqual says whether every entry of a is within eps of b's.
//
// Flutter: no counterpart
func (a Mat4) NearlyEqual(b Mat4, eps float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}
