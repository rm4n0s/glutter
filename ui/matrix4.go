package ui

import "math"

// Vector3 is a 3D vector of float32 components.
//
// Flutter: package:vector_math Vector3
type Vector3 struct {
	// X is the first component.
	X float32
	// Y is the second component.
	Y float32
	// Z is the third component.
	Z float32
}

// Vector4 is a 4D vector of float32 components.
//
// Flutter: package:vector_math Vector4
type Vector4 struct {
	// X is the first component.
	X float32
	// Y is the second component.
	Y float32
	// Z is the third component.
	Z float32
	// W is the fourth component.
	W float32
}

// Matrix4 is a 4x4 matrix of float32 values, stored in column-major order: the
// entry in row r and column c is Storage()[c*4+r]. Transforms are applied to
// column vectors, so m.Multiply(n) applies n first.
//
// Unlike most of package ui, Matrix4 is mutable, as vector_math's is: methods
// such as Translate and Scale change the receiver. Clone makes a copy.
//
// Deviation: the storage is float32 (SPEC §7), where Flutter's vector_math_64
// uses float64.
//
// Flutter: package:vector_math Matrix4
type Matrix4 struct {
	storage [16]float32
}

// NewMatrix4Identity returns the identity matrix.
//
// Flutter: package:vector_math Matrix4.identity
func NewMatrix4Identity() *Matrix4 {
	return &Matrix4{[16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}}
}

// NewMatrix4Zero returns the zero matrix.
//
// Flutter: package:vector_math Matrix4.zero
func NewMatrix4Zero() *Matrix4 { return &Matrix4{} }

// NewMatrix4FromList returns a matrix with the 16 values, in column-major
// order, as its storage.
//
// Flutter: package:vector_math Matrix4.fromList
func NewMatrix4FromList(values [16]float32) *Matrix4 { return &Matrix4{values} }

// NewMatrix4TranslationValues returns the matrix that translates by (x, y, z).
//
// Flutter: package:vector_math Matrix4.translationValues
func NewMatrix4TranslationValues(x, y, z float32) *Matrix4 {
	m := NewMatrix4Identity()
	m.storage[12], m.storage[13], m.storage[14] = x, y, z
	return m
}

// NewMatrix4Diagonal3Values returns the matrix that scales by (x, y, z).
//
// Flutter: package:vector_math Matrix4.diagonal3Values
func NewMatrix4Diagonal3Values(x, y, z float32) *Matrix4 {
	m := NewMatrix4Identity()
	m.storage[0], m.storage[5], m.storage[10] = x, y, z
	return m
}

// NewMatrix4RotationX returns the matrix that rotates by radians around the x
// axis.
//
// Flutter: package:vector_math Matrix4.rotationX
func NewMatrix4RotationX(radians float64) *Matrix4 {
	c, s := float32(math.Cos(radians)), float32(math.Sin(radians))
	m := NewMatrix4Identity()
	m.storage[5], m.storage[6], m.storage[9], m.storage[10] = c, s, -s, c
	return m
}

// NewMatrix4RotationY returns the matrix that rotates by radians around the y
// axis.
//
// Flutter: package:vector_math Matrix4.rotationY
func NewMatrix4RotationY(radians float64) *Matrix4 {
	c, s := float32(math.Cos(radians)), float32(math.Sin(radians))
	m := NewMatrix4Identity()
	m.storage[0], m.storage[2], m.storage[8], m.storage[10] = c, -s, s, c
	return m
}

// NewMatrix4RotationZ returns the matrix that rotates by radians around the z
// axis.
//
// Flutter: package:vector_math Matrix4.rotationZ
func NewMatrix4RotationZ(radians float64) *Matrix4 {
	c, s := float32(math.Cos(radians)), float32(math.Sin(radians))
	m := NewMatrix4Identity()
	m.storage[0], m.storage[1], m.storage[4], m.storage[5] = c, s, -s, c
	return m
}

// NewMatrix4SkewX returns the matrix that skews the x coordinate by the tangent
// of alpha times y.
//
// Flutter: package:vector_math Matrix4.skewX
func NewMatrix4SkewX(alpha float64) *Matrix4 {
	m := NewMatrix4Identity()
	m.storage[4] = float32(math.Tan(alpha))
	return m
}

// NewMatrix4SkewY returns the matrix that skews the y coordinate by the tangent
// of beta times x.
//
// Flutter: package:vector_math Matrix4.skewY
func NewMatrix4SkewY(beta float64) *Matrix4 {
	m := NewMatrix4Identity()
	m.storage[1] = float32(math.Tan(beta))
	return m
}

// NewMatrix4Skew returns the matrix that skews both coordinates.
//
// Flutter: package:vector_math Matrix4.skew
func NewMatrix4Skew(alpha, beta float64) *Matrix4 {
	m := NewMatrix4Identity()
	m.storage[4] = float32(math.Tan(alpha))
	m.storage[1] = float32(math.Tan(beta))
	return m
}

// NewMatrix4CylindricalProjection returns the transform that makes a plane look
// like it is on a cylinder with the given radius, turned by angle radians. The
// perspective is from 0 to 1 (Flutter's default is 0.001). With vertical true
// the turn is around the x axis (Flutter's Axis.vertical, its default), and with
// false around the y axis (Axis.horizontal).
//
// Flutter: painting/matrix_utils.dart MatrixUtils.createCylindricalProjectionTransform
func NewMatrix4CylindricalProjection(radius, angle, perspective float64, vertical bool) *Matrix4 {
	result := NewMatrix4Identity()
	result.SetEntry(3, 2, float32(-perspective))
	result.SetEntry(2, 3, float32(-radius))
	result.SetEntry(3, 3, float32(perspective*radius+1.0))
	var rotation *Matrix4
	if vertical {
		rotation = NewMatrix4RotationX(angle)
	} else {
		rotation = NewMatrix4RotationY(angle)
	}
	return result.Multiply(rotation.Multiply(NewMatrix4TranslationValues(0, 0, float32(radius))))
}

// NewMatrix4ForceToPoint returns a matrix that sends every point to offset: a
// transform that collapses its input.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.forceToPoint
func NewMatrix4ForceToPoint(offset Offset) *Matrix4 {
	m := NewMatrix4Zero()
	m.storage[10] = 1 // Row 2, Column 2: z' = z
	m.storage[12] = float32(offset.Dx)
	m.storage[13] = float32(offset.Dy)
	m.storage[15] = 1 // Row 3, Column 3: w' = w
	return m
}

// Clone returns a copy of the matrix.
//
// Flutter: package:vector_math Matrix4.clone
func (m *Matrix4) Clone() *Matrix4 { c := *m; return &c }

// Storage returns the 16 values in column-major order. The array is the
// matrix's own storage: changing it changes the matrix.
//
// Flutter: package:vector_math Matrix4.storage
func (m *Matrix4) Storage() *[16]float32 { return &m.storage }

// Entry returns the value in the given row and column.
//
// Flutter: package:vector_math Matrix4.entry
func (m *Matrix4) Entry(row, col int) float32 { return m.storage[col*4+row] }

// SetEntry sets the value in the given row and column.
//
// Flutter: package:vector_math Matrix4.setEntry
func (m *Matrix4) SetEntry(row, col int, v float32) { m.storage[col*4+row] = v }

// SetIdentity makes m the identity matrix.
//
// Flutter: package:vector_math Matrix4.setIdentity
func (m *Matrix4) SetIdentity() { *m = *NewMatrix4Identity() }

// SetZero makes m the zero matrix.
//
// Flutter: package:vector_math Matrix4.setZero
func (m *Matrix4) SetZero() { m.storage = [16]float32{} }

// SetFrom copies the values of other into m.
//
// Flutter: package:vector_math Matrix4.setFrom
func (m *Matrix4) SetFrom(other *Matrix4) { m.storage = other.storage }

// IsIdentity reports whether m is exactly the identity matrix.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.isIdentity
func (m *Matrix4) IsIdentity() bool { return m.storage == NewMatrix4Identity().storage }

// Equals reports whether the two matrices hold the same values. A nil matrix
// counts as the identity, and two nils are equal.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.matrixEquals
func (m *Matrix4) Equals(other *Matrix4) bool {
	switch {
	case m == other:
		return true
	case m == nil:
		return other.IsIdentity()
	case other == nil:
		return m.IsIdentity()
	}
	return m.storage == other.storage
}

// Translate translates m by (x, y, z), as if multiplied on the right by the
// translation: m = m * T.
//
// Flutter: package:vector_math Matrix4.translate
func (m *Matrix4) Translate(x, y, z float32) {
	s := &m.storage
	for r := 0; r < 4; r++ {
		s[12+r] = float32(float64(s[r])*float64(x) + float64(s[4+r])*float64(y) + float64(s[8+r])*float64(z) + float64(s[12+r]))
	}
}

// Scale scales m by (x, y, z): m = m * S.
//
// Flutter: package:vector_math Matrix4.scale
func (m *Matrix4) Scale(x, y, z float32) {
	s := &m.storage
	for r := 0; r < 4; r++ {
		s[r] *= x
		s[4+r] *= y
		s[8+r] *= z
	}
}

// ScaleUniform scales m by v on all three axes: m = m * S.
//
// Flutter: package:vector_math Matrix4.scale
func (m *Matrix4) ScaleUniform(v float32) { m.Scale(v, v, v) }

// RotateX rotates m by radians around the x axis: m = m * R.
//
// Flutter: package:vector_math Matrix4.rotateX
func (m *Matrix4) RotateX(radians float64) { m.MultiplyBy(NewMatrix4RotationX(radians)) }

// RotateY rotates m by radians around the y axis: m = m * R.
//
// Flutter: package:vector_math Matrix4.rotateY
func (m *Matrix4) RotateY(radians float64) { m.MultiplyBy(NewMatrix4RotationY(radians)) }

// RotateZ rotates m by radians around the z axis: m = m * R.
//
// Flutter: package:vector_math Matrix4.rotateZ
func (m *Matrix4) RotateZ(radians float64) { m.MultiplyBy(NewMatrix4RotationZ(radians)) }

// Multiply returns the product m * n (Dart's *). Applied to a point, n acts
// first.
//
// Flutter: package:vector_math Matrix4.operator *
func (m *Matrix4) Multiply(n *Matrix4) *Matrix4 {
	var out Matrix4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			var sum float64
			for k := 0; k < 4; k++ {
				sum += float64(m.storage[k*4+r]) * float64(n.storage[c*4+k])
			}
			out.storage[c*4+r] = float32(sum)
		}
	}
	return &out
}

// MultiplyBy sets m to the product m * n.
//
// Flutter: package:vector_math Matrix4.multiply
func (m *Matrix4) MultiplyBy(n *Matrix4) { m.storage = m.Multiply(n).storage }

// MatrixMultiplyInPlace sets b to the product a * b, and leaves a alone.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.multiplyInPlace
func MatrixMultiplyInPlace(a, b *Matrix4) { b.storage = a.Multiply(b).storage }

// Transpose transposes m in place.
//
// Flutter: package:vector_math Matrix4.transpose
func (m *Matrix4) Transpose() {
	s := &m.storage
	s[1], s[4] = s[4], s[1]
	s[2], s[8] = s[8], s[2]
	s[3], s[12] = s[12], s[3]
	s[6], s[9] = s[9], s[6]
	s[7], s[13] = s[13], s[7]
	s[11], s[14] = s[14], s[11]
}

// Transposed returns the transpose of m.
//
// Flutter: package:vector_math Matrix4.transposed
func (m *Matrix4) Transposed() *Matrix4 {
	c := m.Clone()
	c.Transpose()
	return c
}

// cofactors returns the 2x2 sub-determinants that the determinant and the
// inverse are made of, computed in float64.
func (m *Matrix4) cofactors() (a [16]float64, b [12]float64, det float64) {
	for i, v := range m.storage {
		a[i] = float64(v)
	}
	b[0] = a[0]*a[5] - a[1]*a[4]
	b[1] = a[0]*a[6] - a[2]*a[4]
	b[2] = a[0]*a[7] - a[3]*a[4]
	b[3] = a[1]*a[6] - a[2]*a[5]
	b[4] = a[1]*a[7] - a[3]*a[5]
	b[5] = a[2]*a[7] - a[3]*a[6]
	b[6] = a[8]*a[13] - a[9]*a[12]
	b[7] = a[8]*a[14] - a[10]*a[12]
	b[8] = a[8]*a[15] - a[11]*a[12]
	b[9] = a[9]*a[14] - a[10]*a[13]
	b[10] = a[9]*a[15] - a[11]*a[13]
	b[11] = a[10]*a[15] - a[11]*a[14]
	det = b[0]*b[11] - b[1]*b[10] + b[2]*b[9] + b[3]*b[8] - b[4]*b[7] + b[5]*b[6]
	return a, b, det
}

// Determinant returns the determinant of m.
//
// Flutter: package:vector_math Matrix4.determinant
func (m *Matrix4) Determinant() float64 {
	_, _, det := m.cofactors()
	return det
}

// Invert inverts m in place and returns its determinant. A matrix whose
// determinant is zero cannot be inverted: it is left as it was.
//
// Flutter: package:vector_math Matrix4.invert
func (m *Matrix4) Invert() float64 {
	a, b, det := m.cofactors()
	if det == 0 {
		return 0
	}
	inv := 1.0 / det
	m.storage = [16]float32{
		float32((a[5]*b[11] - a[6]*b[10] + a[7]*b[9]) * inv),
		float32((-a[1]*b[11] + a[2]*b[10] - a[3]*b[9]) * inv),
		float32((a[13]*b[5] - a[14]*b[4] + a[15]*b[3]) * inv),
		float32((-a[9]*b[5] + a[10]*b[4] - a[11]*b[3]) * inv),
		float32((-a[4]*b[11] + a[6]*b[8] - a[7]*b[7]) * inv),
		float32((a[0]*b[11] - a[2]*b[8] + a[3]*b[7]) * inv),
		float32((-a[12]*b[5] + a[14]*b[2] - a[15]*b[1]) * inv),
		float32((a[8]*b[5] - a[10]*b[2] + a[11]*b[1]) * inv),
		float32((a[4]*b[10] - a[5]*b[8] + a[7]*b[6]) * inv),
		float32((-a[0]*b[10] + a[1]*b[8] - a[3]*b[6]) * inv),
		float32((a[12]*b[4] - a[13]*b[2] + a[15]*b[0]) * inv),
		float32((-a[8]*b[4] + a[9]*b[2] - a[11]*b[0]) * inv),
		float32((-a[4]*b[9] + a[5]*b[7] - a[6]*b[6]) * inv),
		float32((a[0]*b[9] - a[1]*b[7] + a[2]*b[6]) * inv),
		float32((-a[12]*b[3] + a[13]*b[1] - a[14]*b[0]) * inv),
		float32((a[8]*b[3] - a[9]*b[1] + a[10]*b[0]) * inv),
	}
	return det
}

// Inverted returns the inverse of m, or a copy of m when it has none.
//
// Flutter: package:vector_math Matrix4.copyInverse
func (m *Matrix4) Inverted() *Matrix4 {
	c := m.Clone()
	c.Invert()
	return c
}

// GetTranslation returns the translation part of m.
//
// Flutter: package:vector_math Matrix4.getTranslation
func (m *Matrix4) GetTranslation() Vector3 {
	return Vector3{m.storage[12], m.storage[13], m.storage[14]}
}

// SetTranslationRaw sets the translation part of m.
//
// Flutter: package:vector_math Matrix4.setTranslationRaw
func (m *Matrix4) SetTranslationRaw(x, y, z float32) {
	m.storage[12], m.storage[13], m.storage[14] = x, y, z
}

// GetMaxScaleOnAxis returns the largest factor by which m scales along an axis.
//
// Flutter: package:vector_math Matrix4.getMaxScaleOnAxis
func (m *Matrix4) GetMaxScaleOnAxis() float64 {
	s := &m.storage
	sq := func(i int) float64 {
		x, y, z := float64(s[i]), float64(s[i+1]), float64(s[i+2])
		return x*x + y*y + z*z
	}
	return math.Sqrt(math.Max(sq(0), math.Max(sq(4), sq(8))))
}

// Transform3 transforms the vector as the point (x, y, z, 1), ignoring the
// perspective row.
//
// Flutter: package:vector_math Matrix4.transform3
func (m *Matrix4) Transform3(v Vector3) Vector3 {
	s := &m.storage
	x, y, z := float64(v.X), float64(v.Y), float64(v.Z)
	return Vector3{
		float32(float64(s[0])*x + float64(s[4])*y + float64(s[8])*z + float64(s[12])),
		float32(float64(s[1])*x + float64(s[5])*y + float64(s[9])*z + float64(s[13])),
		float32(float64(s[2])*x + float64(s[6])*y + float64(s[10])*z + float64(s[14])),
	}
}

// Transform transforms the 4D vector.
//
// Flutter: package:vector_math Matrix4.transform
func (m *Matrix4) Transform(v Vector4) Vector4 {
	s := &m.storage
	x, y, z, w := float64(v.X), float64(v.Y), float64(v.Z), float64(v.W)
	return Vector4{
		float32(float64(s[0])*x + float64(s[4])*y + float64(s[8])*z + float64(s[12])*w),
		float32(float64(s[1])*x + float64(s[5])*y + float64(s[9])*z + float64(s[13])*w),
		float32(float64(s[2])*x + float64(s[6])*y + float64(s[10])*z + float64(s[14])*w),
		float32(float64(s[3])*x + float64(s[7])*y + float64(s[11])*z + float64(s[15])*w),
	}
}

// PerspectiveTransform transforms the vector as the point (x, y, z, 1) and
// divides the result by its w.
//
// Flutter: package:vector_math Matrix4.perspectiveTransform
func (m *Matrix4) PerspectiveTransform(v Vector3) Vector3 {
	s := &m.storage
	x, y, z := float64(v.X), float64(v.Y), float64(v.Z)
	w := 1.0 / (float64(s[3])*x + float64(s[7])*y + float64(s[11])*z + float64(s[15]))
	return Vector3{
		float32((float64(s[0])*x + float64(s[4])*y + float64(s[8])*z + float64(s[12])) * w),
		float32((float64(s[1])*x + float64(s[5])*y + float64(s[9])*z + float64(s[13])) * w),
		float32((float64(s[2])*x + float64(s[6])*y + float64(s[10])*z + float64(s[14])) * w),
	}
}

// AsTranslation returns the translation (dx, dy) when m is exactly a
// translation in the plane, and false otherwise.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.getAsTranslation
func (m *Matrix4) AsTranslation() (Offset, bool) {
	s := &m.storage
	want := [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, s[12], s[13], 0, 1}
	if *s != want {
		return Offset{}, false
	}
	return Offset{float64(s[12]), float64(s[13])}, true
}

// AsScale returns the factor when m is exactly a uniform scale in the plane
// (the same x and y factor, nothing else), and false otherwise.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.getAsScale
func (m *Matrix4) AsScale() (float64, bool) {
	s := &m.storage
	want := [16]float32{s[0], 0, 0, 0, 0, s[5], 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if s[0] != s[5] || *s != want {
		return 0, false
	}
	return float64(s[0]), true
}

// TransformPoint transforms the point (x, y) as the vector (x, y, 0, 1) and
// drops the resulting z. It divides by w only when w is not 1.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.transformPoint
func (m *Matrix4) TransformPoint(point Offset) Offset {
	s := &m.storage
	x, y := point.Dx, point.Dy
	rx := float64(s[0])*x + float64(s[4])*y + float64(s[12])
	ry := float64(s[1])*x + float64(s[5])*y + float64(s[13])
	rw := float64(s[3])*x + float64(s[7])*y + float64(s[15])
	if rw == 1.0 {
		return Offset{rx, ry}
	}
	return Offset{rx / rw, ry / rw}
}

// TransformRect returns the smallest rectangle that contains the transformed
// rect.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.transformRect
func (m *Matrix4) TransformRect(rect Rect) Rect {
	s := &m.storage
	x, y := rect.Left, rect.Top
	w, h := rect.Right-x, rect.Bottom-y
	// We want to avoid turning a finite rect into an infinite one if we can.
	if !isFinite(w) || !isFinite(h) {
		return m.safeTransformRect(rect)
	}
	// Without perspective, the rectangle becomes a parallelogram with a corner at
	// (rx, ry) and edge vectors (wx, wy) and (hx, hy).
	wx, hx := float64(s[0])*w, float64(s[4])*h
	rx := float64(s[0])*x + float64(s[4])*y + float64(s[12])
	wy, hy := float64(s[1])*w, float64(s[5])*h
	ry := float64(s[1])*x + float64(s[5])*y + float64(s[13])
	if s[3] == 0.0 && s[7] == 0.0 && s[15] == 1.0 {
		left, right := rx, rx
		if wx < 0 {
			left += wx
		} else {
			right += wx
		}
		if hx < 0 {
			left += hx
		} else {
			right += hx
		}
		top, bottom := ry, ry
		if wy < 0 {
			top += wy
		} else {
			bottom += wy
		}
		if hy < 0 {
			top += hy
		} else {
			bottom += hy
		}
		return Rect{left, top, right, bottom}
	}
	ww, hw := float64(s[3])*w, float64(s[7])*h
	rw := float64(s[3])*x + float64(s[7])*y + float64(s[15])
	ulx, uly := rx/rw, ry/rw
	urx, ury := (rx+wx)/(rw+ww), (ry+wy)/(rw+ww)
	llx, lly := (rx+hx)/(rw+hw), (ry+hy)/(rw+hw)
	lrx, lry := (rx+wx+hx)/(rw+ww+hw), (ry+wy+hy)/(rw+ww+hw)
	return Rect{
		min4(ulx, urx, llx, lrx),
		min4(uly, ury, lly, lry),
		max4(ulx, urx, llx, lrx),
		max4(uly, ury, lly, lry),
	}
}

// safeTransformRect transforms the four corners one by one, for rectangles that
// are not finite.
func (m *Matrix4) safeTransformRect(rect Rect) Rect {
	s := &m.storage
	isAffine := s[3] == 0.0 && s[7] == 0.0 && s[15] == 1.0
	corner := func(x, y float64) (float64, float64) {
		w := 1.0
		if !isAffine {
			w = 1.0 / (float64(s[3])*x + float64(s[7])*y + float64(s[15]))
		}
		return (float64(s[0])*x + float64(s[4])*y + float64(s[12])) * w,
			(float64(s[1])*x + float64(s[5])*y + float64(s[13])) * w
	}
	minX, minY := corner(rect.Left, rect.Top)
	maxX, maxY := minX, minY
	for _, c := range [3][2]float64{{rect.Right, rect.Top}, {rect.Left, rect.Bottom}, {rect.Right, rect.Bottom}} {
		tx, ty := corner(c[0], c[1])
		if tx < minX {
			minX = tx
		}
		if ty < minY {
			minY = ty
		}
		if tx > maxX {
			maxX = tx
		}
		if ty > maxY {
			maxY = ty
		}
	}
	return Rect{minX, minY, maxX, maxY}
}

// InverseTransformRect returns the smallest rectangle that contains rect
// transformed by the inverse of m.
//
// Flutter: painting/matrix_utils.dart MatrixUtils.inverseTransformRect
func (m *Matrix4) InverseTransformRect(rect Rect) Rect {
	if m.IsIdentity() {
		return rect
	}
	return m.Inverted().TransformRect(rect)
}

func min4(a, b, c, d float64) float64 {
	e := b
	if a < b {
		e = a
	}
	f := d
	if c < d {
		f = c
	}
	if e < f {
		return e
	}
	return f
}

func max4(a, b, c, d float64) float64 {
	e := b
	if a > b {
		e = a
	}
	f := d
	if c > d {
		f = c
	}
	if e > f {
		return e
	}
	return f
}
