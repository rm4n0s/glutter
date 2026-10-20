// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"math"
	"math/rand"
	"testing"
)

// near reports whether a and b differ by at most eps.
func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func matrixNear(t *testing.T, got, want *Matrix4, eps float64) {
	t.Helper()
	for i := range got.storage {
		if !near(float64(got.storage[i]), float64(want.storage[i]), eps) {
			t.Errorf("storage[%d] = %v, want %v (±%g)\n got  %v\n want %v", i, got.storage[i], want.storage[i], eps, got.storage, want.storage)
			return
		}
	}
}

func rectNear(t *testing.T, got, want Rect, eps float64) {
	t.Helper()
	if !near(got.Left, want.Left, eps) || !near(got.Top, want.Top, eps) ||
		!near(got.Right, want.Right, eps) || !near(got.Bottom, want.Bottom, eps) {
		t.Errorf("got %v, want %v (±%g)", got, want, eps)
	}
}

func TestMatrix4Constructors(t *testing.T) {
	tests := []struct {
		name string
		got  *Matrix4
		want [16]float32
	}{
		{"identity", NewMatrix4Identity(), [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"zero", NewMatrix4Zero(), [16]float32{}},
		{"translation", NewMatrix4TranslationValues(1, 2, 3), [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 1, 2, 3, 1}},
		{"diagonal", NewMatrix4Diagonal3Values(2, 3, 4), [16]float32{2, 0, 0, 0, 0, 3, 0, 0, 0, 0, 4, 0, 0, 0, 0, 1}},
		{"rotation x by 90°", NewMatrix4RotationX(math.Pi / 2), [16]float32{1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1}},
		{"rotation y by 90°", NewMatrix4RotationY(math.Pi / 2), [16]float32{0, 0, -1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1}},
		{"rotation z by 90°", NewMatrix4RotationZ(math.Pi / 2), [16]float32{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"skew x", NewMatrix4SkewX(math.Pi / 4), [16]float32{1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"skew y", NewMatrix4SkewY(math.Pi / 4), [16]float32{1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"skew", NewMatrix4Skew(math.Pi/4, math.Pi/4), [16]float32{1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"from list", NewMatrix4FromList([16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), [16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{"force to point", NewMatrix4ForceToPoint(Offset{20, -30}), [16]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 20, -30, 0, 1}},
	}
	for _, tt := range tests {
		// The rotation entries are cos(pi/2) = 6e-17, not exactly zero.
		matrixNear(t, tt.got, &Matrix4{tt.want}, 1e-6)
	}
	// Row and column access: the entry in row r, column c is storage[c*4+r].
	m := NewMatrix4FromList([16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	if m.Entry(1, 0) != 2 || m.Entry(0, 1) != 5 || m.Entry(3, 3) != 16 || m.Entry(2, 3) != 15 {
		t.Error("Entry reads the wrong storage slots")
	}
	m.SetEntry(0, 3, 99)
	if m.storage[12] != 99 {
		t.Error("SetEntry writes the wrong storage slot")
	}
	if m.Storage() != &m.storage {
		t.Error("Storage must return the matrix's own storage")
	}
}

func TestMatrix4CopySetAndEquals(t *testing.T) {
	m := NewMatrix4TranslationValues(1, 2, 3)
	c := m.Clone()
	c.SetEntry(0, 3, 50)
	if m.Entry(0, 3) != 1 {
		t.Error("Clone must not share storage")
	}
	c.SetFrom(m)
	if !c.Equals(m) {
		t.Error("SetFrom must copy the values")
	}
	c.SetIdentity()
	if !c.IsIdentity() {
		t.Error("SetIdentity must give the identity")
	}
	c.SetZero()
	if c.IsIdentity() || c.storage != [16]float32{} {
		t.Error("SetZero must give zeros")
	}

	var nilMatrix *Matrix4
	tests := []struct {
		name string
		a, b *Matrix4
		want bool
	}{
		{"same", m, m, true},
		{"equal values", m, m.Clone(), true},
		{"different", m, NewMatrix4Identity(), false},
		{"nil and nil", nilMatrix, nilMatrix, true},
		{"nil is the identity", nilMatrix, NewMatrix4Identity(), true},
		{"identity is nil", NewMatrix4Identity(), nilMatrix, true},
		{"nil is not a translation", nilMatrix, m, false},
		{"a translation is not nil", m, nilMatrix, false},
	}
	for _, tt := range tests {
		if got := tt.a.Equals(tt.b); got != tt.want {
			t.Errorf("%s: Equals = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMatrix4TranslateScaleRotate(t *testing.T) {
	// Translate, Scale and the rotations multiply on the right: m = m * T.
	t.Run("translate", func(t *testing.T) {
		m := NewMatrix4Diagonal3Values(2, 2, 2)
		m.Translate(1, 2, 3)
		matrixNear(t, m, NewMatrix4Diagonal3Values(2, 2, 2).Multiply(NewMatrix4TranslationValues(1, 2, 3)), 0)
		// A point is translated first, then scaled: (0,0,0) -> (1,2,3) -> (2,4,6).
		if got := m.Transform3(Vector3{}); got != (Vector3{2, 4, 6}) {
			t.Errorf("point = %v, want {2 4 6}", got)
		}
	})
	t.Run("scale", func(t *testing.T) {
		m := NewMatrix4TranslationValues(10, 20, 30)
		m.Scale(2, 3, 4)
		if got := m.Transform3(Vector3{1, 1, 1}); got != (Vector3{12, 23, 34}) {
			t.Errorf("point = %v, want {12 23 34}", got)
		}
		m.ScaleUniform(2)
		if m.Entry(0, 0) != 4 || m.Entry(1, 1) != 6 || m.Entry(2, 2) != 8 {
			t.Errorf("ScaleUniform gave %v", m.storage)
		}
	})
	t.Run("rotations", func(t *testing.T) {
		for _, tt := range []struct {
			name   string
			rotate func(*Matrix4, float64)
			make   func(float64) *Matrix4
		}{
			{"x", (*Matrix4).RotateX, NewMatrix4RotationX},
			{"y", (*Matrix4).RotateY, NewMatrix4RotationY},
			{"z", (*Matrix4).RotateZ, NewMatrix4RotationZ},
		} {
			m := NewMatrix4TranslationValues(3, 4, 5)
			tt.rotate(m, 0.7)
			matrixNear(t, m, NewMatrix4TranslationValues(3, 4, 5).Multiply(tt.make(0.7)), 1e-6)
		}
	})
	t.Run("rotating z by 90° sends x to y", func(t *testing.T) {
		got := NewMatrix4RotationZ(math.Pi / 2).Transform3(Vector3{1, 0, 0})
		if !near(float64(got.X), 0, 1e-6) || !near(float64(got.Y), 1, 1e-6) {
			t.Errorf("got %v", got)
		}
	})
}

func TestMatrix4Multiply(t *testing.T) {
	a := NewMatrix4FromList([16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	b := NewMatrix4FromList([16]float32{2, 0, 1, 0, 1, 3, 0, 0, 0, 1, 4, 0, 5, 0, 0, 1})
	// Computed by hand for the first column of a*b: a * (2,0,1,0).
	got := a.Multiply(b)
	wantCol0 := [4]float32{2*1 + 0*5 + 1*9, 2*2 + 0*6 + 1*10, 2*3 + 0*7 + 1*11, 2*4 + 0*8 + 1*12}
	for r := 0; r < 4; r++ {
		if got.Entry(r, 0) != wantCol0[r] {
			t.Errorf("(a*b)[%d][0] = %v, want %v", r, got.Entry(r, 0), wantCol0[r])
		}
	}
	if !a.Multiply(NewMatrix4Identity()).Equals(a) || !NewMatrix4Identity().Multiply(a).Equals(a) {
		t.Error("the identity must be neutral")
	}
	if a.Multiply(b).Equals(b.Multiply(a)) {
		t.Error("matrix multiplication is not commutative")
	}

	// MultiplyBy changes the receiver only.
	c := a.Clone()
	c.MultiplyBy(b)
	if !c.Equals(a.Multiply(b)) {
		t.Error("MultiplyBy must set the receiver to the product")
	}
}

// Ported from packages/flutter/test/painting/matrix_utils_test.dart.
func TestMatrixUtilsMultiplyInPlace(t *testing.T) {
	originalA := NewMatrix4Identity()
	originalA.Translate(10, 0, 0)
	a := originalA.Clone()
	b := NewMatrix4Identity()
	b.Translate(20, 0, 0)
	result := a.Multiply(b)
	MatrixMultiplyInPlace(a, b)
	if a.storage != originalA.storage {
		t.Error("the first argument must not change")
	}
	if b.storage != result.storage {
		t.Error("the second argument must hold the product")
	}
	if b.GetTranslation().X == 20 {
		t.Error("the translation of b must have changed")
	}
}

// Ported from packages/flutter/test/painting/matrix_utils_test.dart.
func TestMatrixUtilsAsTranslation(t *testing.T) {
	tests := []struct {
		name string
		m    func() *Matrix4
		want *Offset
	}{
		{"identity", NewMatrix4Identity, ptr(OffsetZero)},
		{"zero", NewMatrix4Zero, nil},
		{"rotation x", func() *Matrix4 { return NewMatrix4RotationX(1) }, nil},
		{"rotation z", func() *Matrix4 { return NewMatrix4RotationZ(1) }, nil},
		{"translation in the plane", func() *Matrix4 { return NewMatrix4TranslationValues(1, 2, 0) }, ptr(Offset{1, 2})},
		{"translation with z", func() *Matrix4 { return NewMatrix4TranslationValues(1, 2, 3) }, nil},
		{"rotated about x", func() *Matrix4 { m := NewMatrix4Identity(); m.RotateX(2); return m }, nil},
		{"scaled", func() *Matrix4 { m := NewMatrix4Identity(); m.ScaleUniform(2); return m }, nil},
		{"translated once", func() *Matrix4 { m := NewMatrix4Identity(); m.Translate(2, -2, 0); return m }, ptr(Offset{2, -2})},
		{"translated twice", func() *Matrix4 {
			m := NewMatrix4Identity()
			m.Translate(2, -2, 0)
			m.Translate(4, 8, 0)
			return m
		}, ptr(Offset{6, 6})},
		{"NaN", func() *Matrix4 { return NewMatrix4TranslationValues(float32(math.NaN()), 0, 0) }, nil},
	}
	for _, tt := range tests {
		got, ok := tt.m().AsTranslation()
		if tt.want == nil {
			if ok {
				t.Errorf("%s: got translation %v, want none", tt.name, got)
			}
			continue
		}
		if !ok || got != *tt.want {
			t.Errorf("%s: got %v %v, want %v", tt.name, got, ok, *tt.want)
		}
	}
}

func TestMatrixUtilsAsScale(t *testing.T) {
	tests := []struct {
		name string
		m    *Matrix4
		want float64
		ok   bool
	}{
		{"identity", NewMatrix4Identity(), 1, true},
		{"uniform", NewMatrix4Diagonal3Values(3, 3, 1), 3, true},
		{"non-uniform", NewMatrix4Diagonal3Values(3, 2, 1), 0, false},
		{"scales z", NewMatrix4Diagonal3Values(3, 3, 3), 0, false},
		{"translated", NewMatrix4TranslationValues(1, 0, 0), 0, false},
		{"rotated", NewMatrix4RotationZ(0.5), 0, false},
	}
	for _, tt := range tests {
		got, ok := tt.m.AsScale()
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: got %v %v, want %v %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

// Ported from packages/flutter/test/painting/matrix_utils_test.dart.
func TestCylindricalProjectionTransform(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		got := NewMatrix4CylindricalProjection(0, 0, 0, true)
		matrixNear(t, got, NewMatrix4Identity(), 0)
	})
	t.Run("rotate with no radius", func(t *testing.T) {
		got := NewMatrix4CylindricalProjection(0, math.Pi/2, 0, true)
		matrixNear(t, got, NewMatrix4RotationX(math.Pi/2), 0)
	})
	t.Run("radius does not change scale", func(t *testing.T) {
		got := NewMatrix4CylindricalProjection(1000000, 0, 0, true)
		matrixNear(t, got, NewMatrix4Identity(), 0)
	})
	t.Run("calculation spot check", func(t *testing.T) {
		got := NewMatrix4CylindricalProjection(100, math.Pi/3, 0.001, true)
		want := NewMatrix4FromList([16]float32{
			1, 0, 0, 0,
			0, 0.5, 0.8660254037844386, -0.0008660254037844386,
			0, -0.8660254037844386, 0.5, -0.0005,
			0, -86.60254037844386, -50.0, 1.05,
		})
		// Flutter compares to 1e-10 in float64; float32 keeps 7 digits.
		matrixNear(t, got, want, 1e-5)
	})
	t.Run("horizontal turns around y", func(t *testing.T) {
		got := NewMatrix4CylindricalProjection(0, 0.3, 0, false)
		matrixNear(t, got, NewMatrix4RotationY(0.3), 1e-7)
	})
}

// Ported from packages/flutter/test/painting/matrix_utils_test.dart.
func TestMatrixUtilsForceToPoint(t *testing.T) {
	forced := Offset{20, -30}
	m := NewMatrix4ForceToPoint(forced)
	for _, p := range []Offset{forced, OffsetZero, {1, 1}, {-1, -1}, {-20, 30}, {-1.2344, 1422434.23}} {
		if got := m.TransformPoint(p); got != forced {
			t.Errorf("TransformPoint(%v) = %v, want %v", p, got, forced)
		}
	}
}

// Ported from packages/flutter/test/painting/matrix_utils_test.dart.
func TestMatrixUtilsTransformRect(t *testing.T) {
	t.Run("very large finite values", func(t *testing.T) {
		evil := Rect{0, -1.7976931348623157e+308, 800, 1.7976931348623157e+308}
		m := NewMatrix4Identity()
		m.Translate(10, 0, 0)
		if got := m.TransformRect(evil); !got.IsFinite() {
			t.Errorf("got %v, want a finite rectangle", got)
		}
	})
	t.Run("no perspective", func(t *testing.T) {
		rect := Rect{10, 20, 30, 40}
		matrixRect := []struct {
			name string
			m    *Matrix4
			want Rect
		}{
			{"identity", NewMatrix4Identity(), rect},
			{"2D scaling", NewMatrix4Diagonal3Values(2, 2, 2), Rect{20, 40, 60, 80}},
			{"rotation", NewMatrix4RotationZ(math.Pi / 2), Rect{-40, 10, -20, 30}},
			{"negative scale flips", NewMatrix4Diagonal3Values(-1, 1, 1), Rect{-30, 20, -10, 40}},
			{"translation", NewMatrix4TranslationValues(5, -5, 0), Rect{15, 15, 35, 35}},
		}
		for _, tt := range matrixRect {
			rectNear(t, tt.m.TransformRect(rect), tt.want, 1e-5)
		}
	})
	t.Run("with perspective", func(t *testing.T) {
		m := NewMatrix4CylindricalProjection(10, math.Pi/8, 0.3, true)
		for i := 1; i < 10000; i++ {
			f := float64(i)
			rect := Rect{11 * f, 12 * f, 15 * f, 18 * f}
			golden := vectorWiseTransformRect(m, rect)
			rectNear(t, m.TransformRect(rect), golden, 1e-9*f)
			if t.Failed() {
				t.Fatalf("failed at i = %d", i)
			}
		}
	})
	t.Run("an infinite rectangle stays infinite, not NaN", func(t *testing.T) {
		m := NewMatrix4TranslationValues(10, 5, 0)
		got := m.TransformRect(Rect{0, 0, math.Inf(1), 10})
		if got.HasNaN() || got.Left != 10 || got.Top != 5 || !math.IsInf(got.Right, 1) || got.Bottom != 15 {
			t.Errorf("got %v", got)
		}
	})
	t.Run("inverse", func(t *testing.T) {
		rect := Rect{10, 20, 30, 40}
		m := NewMatrix4Diagonal3Values(2, 4, 1)
		rectNear(t, m.InverseTransformRect(rect), Rect{5, 5, 15, 10}, 1e-6)
		rectNear(t, NewMatrix4Identity().InverseTransformRect(rect), rect, 0)
		trans := NewMatrix4TranslationValues(10, 10, 0)
		rectNear(t, trans.InverseTransformRect(rect), Rect{0, 10, 20, 30}, 1e-6)
	})
}

// vectorWiseTransformRect transforms the corners one at a time with the
// homogeneous coordinates in float64: the golden implementation of TransformRect.
func vectorWiseTransformRect(m *Matrix4, rect Rect) Rect {
	point := func(p Offset) Offset {
		s := m.storage
		x, y := p.Dx, p.Dy
		w := float64(s[3])*x + float64(s[7])*y + float64(s[15])
		return Offset{(float64(s[0])*x + float64(s[4])*y + float64(s[12])) / w, (float64(s[1])*x + float64(s[5])*y + float64(s[13])) / w}
	}
	p1, p2, p3, p4 := point(rect.TopLeft()), point(rect.TopRight()), point(rect.BottomLeft()), point(rect.BottomRight())
	return Rect{
		math.Min(p1.Dx, math.Min(p2.Dx, math.Min(p3.Dx, p4.Dx))),
		math.Min(p1.Dy, math.Min(p2.Dy, math.Min(p3.Dy, p4.Dy))),
		math.Max(p1.Dx, math.Max(p2.Dx, math.Max(p3.Dx, p4.Dx))),
		math.Max(p1.Dy, math.Max(p2.Dy, math.Max(p3.Dy, p4.Dy))),
	}
}

func TestMatrix4TransformPoint(t *testing.T) {
	m := NewMatrix4TranslationValues(10, 20, 0)
	m.RotateZ(math.Pi / 2)
	// Rotate (1, 0) to (0, 1), then translate.
	got := m.TransformPoint(Offset{1, 0})
	if !near(got.Dx, 10, 1e-6) || !near(got.Dy, 21, 1e-6) {
		t.Errorf("got %v, want Offset(10, 21)", got)
	}
	// With perspective, TransformPoint divides by w like PerspectiveTransform.
	p := NewMatrix4CylindricalProjection(10, 0.4, 0.3, true)
	for _, pt := range []Offset{{1, 2}, {-30, 40}, {100, -7}} {
		want := p.PerspectiveTransform(Vector3{float32(pt.Dx), float32(pt.Dy), 0})
		got := p.TransformPoint(pt)
		if !near(got.Dx, float64(want.X), 1e-4*math.Max(1, math.Abs(got.Dx))) || !near(got.Dy, float64(want.Y), 1e-4*math.Max(1, math.Abs(got.Dy))) {
			t.Errorf("TransformPoint(%v) = %v, PerspectiveTransform = %v", pt, got, want)
		}
	}
}

func TestMatrix4TransformVectors(t *testing.T) {
	m := NewMatrix4TranslationValues(1, 2, 3)
	m.Scale(2, 2, 2)
	if got := m.Transform(Vector4{1, 1, 1, 1}); got != (Vector4{3, 4, 5, 1}) {
		t.Errorf("Transform = %v", got)
	}
	if got := m.Transform(Vector4{1, 1, 1, 0}); got != (Vector4{2, 2, 2, 0}) {
		t.Errorf("Transform of a direction = %v", got)
	}
	if got := m.Transform3(Vector3{1, 1, 1}); got != (Vector3{3, 4, 5}) {
		t.Errorf("Transform3 = %v", got)
	}
	m.SetTranslationRaw(7, 8, 9)
	if m.GetTranslation() != (Vector3{7, 8, 9}) {
		t.Errorf("GetTranslation = %v", m.GetTranslation())
	}
}

func TestMatrix4Determinant(t *testing.T) {
	tests := []struct {
		name string
		m    *Matrix4
		want float64
	}{
		{"identity", NewMatrix4Identity(), 1},
		{"zero", NewMatrix4Zero(), 0},
		{"scale", NewMatrix4Diagonal3Values(2, 3, 4), 24},
		{"translation", NewMatrix4TranslationValues(5, 6, 7), 1},
		{"rotation", NewMatrix4RotationY(1.1), 1},
		{"mirror", NewMatrix4Diagonal3Values(-1, 1, 1), -1},
		{"singular", NewMatrix4Diagonal3Values(1, 0, 1), 0},
	}
	for _, tt := range tests {
		if got := tt.m.Determinant(); !near(got, tt.want, 1e-6) {
			t.Errorf("%s: det = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMatrix4Invert(t *testing.T) {
	t.Run("known inverses", func(t *testing.T) {
		m := NewMatrix4Diagonal3Values(2, 4, 5)
		m.SetTranslationRaw(10, 20, 30)
		det := m.Invert()
		if !near(det, 40, 1e-6) {
			t.Errorf("determinant = %v, want 40", det)
		}
		want := NewMatrix4Diagonal3Values(0.5, 0.25, 0.2)
		want.SetTranslationRaw(-5, -5, -6)
		matrixNear(t, m, want, 1e-6)
	})
	t.Run("singular matrices are left alone", func(t *testing.T) {
		m := NewMatrix4Diagonal3Values(1, 0, 1)
		before := m.storage
		if det := m.Invert(); det != 0 {
			t.Errorf("determinant = %v, want 0", det)
		}
		if m.storage != before {
			t.Error("a singular matrix must not change")
		}
		if got := m.Inverted(); got.storage != before {
			t.Error("Inverted of a singular matrix must be a copy of it")
		}
	})
	t.Run("random matrices times their inverse are the identity", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1))
		checked := 0
		for i := 0; i < 500; i++ {
			var v [16]float32
			for j := range v {
				v[j] = float32(rng.Float64()*4 - 2)
			}
			m := NewMatrix4FromList(v)
			if math.Abs(m.Determinant()) < 0.1 {
				continue // too close to singular for float32
			}
			checked++
			matrixNear(t, m.Multiply(m.Inverted()), NewMatrix4Identity(), 1e-4)
			matrixNear(t, m.Inverted().Multiply(m), NewMatrix4Identity(), 1e-4)
		}
		if checked < 100 {
			t.Fatalf("only %d matrices were invertible enough", checked)
		}
	})
}

func TestMatrix4Transpose(t *testing.T) {
	m := NewMatrix4FromList([16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	tr := m.Transposed()
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if tr.Entry(r, c) != m.Entry(c, r) {
				t.Fatalf("entry (%d, %d) = %v, want %v", r, c, tr.Entry(r, c), m.Entry(c, r))
			}
		}
	}
	if m.Entry(0, 1) != 5 {
		t.Error("Transposed must not change the receiver")
	}
	tr.Transpose()
	if !tr.Equals(m) {
		t.Error("transposing twice must give the matrix back")
	}
}

func TestMatrix4GetMaxScaleOnAxis(t *testing.T) {
	tests := []struct {
		name string
		m    *Matrix4
		want float64
	}{
		{"identity", NewMatrix4Identity(), 1},
		{"scale", NewMatrix4Diagonal3Values(2, 5, 3), 5},
		{"rotation keeps scale", NewMatrix4RotationZ(0.7), 1},
		{"scaled rotation", func() *Matrix4 { m := NewMatrix4RotationZ(0.7); m.ScaleUniform(3); return m }(), 3},
		{"translation is ignored", NewMatrix4TranslationValues(100, 100, 100), 1},
	}
	for _, tt := range tests {
		if got := tt.m.GetMaxScaleOnAxis(); !near(got, tt.want, 1e-6) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMatrix4IsIdentity(t *testing.T) {
	if !NewMatrix4Identity().IsIdentity() || NewMatrix4Zero().IsIdentity() || NewMatrix4TranslationValues(0, 0, 1e-9).IsIdentity() {
		t.Error("IsIdentity wrong")
	}
}
