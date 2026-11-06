package vecmath

import (
	"math"
	"math/rand"
	"testing"
)

const eps = 1e-9

func near(a, b, e float64) bool { return math.Abs(a-b) <= e }

func nearVec(a, b Vec3, e float64) bool { return a.NearlyEqual(b, e) }

// randVec returns a vector with components in [-scale, scale].
func randVec(r *rand.Rand, scale float64) Vec3 {
	return Vec3{(r.Float64()*2 - 1) * scale, (r.Float64()*2 - 1) * scale, (r.Float64()*2 - 1) * scale}
}

// randRotation returns a random rotation.
func randRotation(r *rand.Rand) Quat {
	return NewQuatAxisAngle(randVec(r, 1).Add(Vec3{X: 0.01}), r.Float64()*2*math.Pi)
}

func TestVectorsKnownValues(t *testing.T) {
	tests := []struct {
		name string
		got  Vec3
		want Vec3
	}{
		{"add", Vec3{1, 2, 3}.Add(Vec3{4, 5, 6}), Vec3{5, 7, 9}},
		{"sub", Vec3{1, 2, 3}.Sub(Vec3{4, 5, 6}), Vec3{-3, -3, -3}},
		{"scale", Vec3{1, -2, 3}.Scale(2), Vec3{2, -4, 6}},
		{"cross x y", Vec3X.Cross(Vec3Y), Vec3Z},
		{"cross y x", Vec3Y.Cross(Vec3X), Vec3{Z: -1}},
		{"cross of parallel vectors", Vec3{1, 2, 3}.Cross(Vec3{2, 4, 6}), Vec3Zero},
		{"normalize", Vec3{0, 3, 4}.Normalize(), Vec3{0, 0.6, 0.8}},
		{"normalize of zero", Vec3Zero.Normalize(), Vec3Zero},
		{"lerp", Vec3{0, 0, 0}.Lerp(Vec3{10, 20, 30}, 0.25), Vec3{2.5, 5, 7.5}},
		{"min", Vec3{1, 5, 3}.Min(Vec3{2, 4, 6}), Vec3{1, 4, 3}},
		{"max", Vec3{1, 5, 3}.Max(Vec3{2, 4, 6}), Vec3{2, 5, 6}},
		{"reflect", Vec3{1, -1, 0}.Reflect(Vec3Y), Vec3{1, 1, 0}},
		{"project onto", Vec3{3, 4, 0}.ProjectOnto(Vec3X), Vec3{3, 0, 0}},
		{"project onto zero", Vec3{3, 4, 0}.ProjectOnto(Vec3Zero), Vec3Zero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !nearVec(tt.got, tt.want, eps) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	scalars := []struct {
		name      string
		got, want float64
	}{
		{"dot", Vec3{1, 2, 3}.Dot(Vec3{4, -5, 6}), 12},
		{"length", Vec3{2, 3, 6}.Length(), 7},
		{"distance", Vec3{1, 1, 1}.DistanceTo(Vec3{4, 5, 1}), 5},
		{"angle", Vec3X.AngleTo(Vec3Y), math.Pi / 2},
		{"angle of opposites", Vec3X.AngleTo(Vec3X.Neg()), math.Pi},
		{"angle with zero", Vec3X.AngleTo(Vec3Zero), 0},
		{"vec2 cross", Vec2{1, 0}.Cross(Vec2{0, 1}), 1},
		{"vec2 length", Vec2{3, 4}.Length(), 5},
		{"vec4 dot", Vec4{1, 2, 3, 4}.Dot(Vec4{1, 1, 1, 1}), 10},
	}
	for _, tt := range scalars {
		t.Run(tt.name, func(t *testing.T) {
			if !near(tt.got, tt.want, eps) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestPerpendicularIsPerpendicularAndUnit(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		v := randVec(r, 100)
		p := v.Perpendicular()
		if !near(p.Length(), 1, 1e-9) || !near(p.Dot(v), 0, 1e-7) {
			t.Fatalf("Perpendicular(%v) = %v: length %v, dot %v", v, p, p.Length(), p.Dot(v))
		}
	}
	for _, axis := range []Vec3{Vec3X, Vec3Y, Vec3Z, Vec3X.Neg()} {
		if p := axis.Perpendicular(); !near(p.Length(), 1, eps) || !near(p.Dot(axis), 0, eps) {
			t.Errorf("Perpendicular(%v) = %v", axis, p)
		}
	}
}

func TestMatricesKnownValues(t *testing.T) {
	tests := []struct {
		name string
		got  Mat4
		want Mat4
	}{
		{"translation", NewMat4Translation(Vec3{1, 2, 3}), Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 1, 2, 3, 1}},
		{"scale then move", NewMat4Translation(Vec3{1, 2, 3}).Mul(NewMat4Scale(Vec3{2, 2, 2})),
			Mat4{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 1, 2, 3, 1}},
		{"move then scale", NewMat4Scale(Vec3{2, 2, 2}).Mul(NewMat4Translation(Vec3{1, 2, 3})),
			Mat4{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 2, 4, 6, 1}},
		{"rotation z by a quarter turn", NewMat4RotationZ(math.Pi / 2), Mat4{0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}},
		{"rotation x by a quarter turn", NewMat4RotationX(math.Pi / 2), Mat4{1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 1}},
		{"rotation y by a quarter turn", NewMat4RotationY(math.Pi / 2), Mat4{0, 0, -1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1}},
		{"look at from +z", LookAt(Vec3{0, 0, 5}, Vec3Zero, Vec3Y), NewMat4Translation(Vec3{0, 0, -5})},
		{"look at from +x", LookAt(Vec3{5, 0, 0}, Vec3Zero, Vec3Y),
			Mat4{0, 0, 1, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, -5, 1}},
		{"look at with eye = center", LookAt(Vec3{1, 1, 1}, Vec3{1, 1, 1}, Vec3Y), Mat4Identity()},
		{"look at with up along the view", LookAt(Vec3{0, 5, 0}, Vec3Zero, Vec3Y), Mat4Identity()},
		{"orthographic of a unit cube", Orthographic(-1, 1, -1, 1, 0, 1), Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1}},
		{"transpose", Mat4{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}.Transpose(),
			Mat4{1, 5, 9, 13, 2, 6, 10, 14, 3, 7, 11, 15, 4, 8, 12, 16}},
		{"with translation", Mat4Identity().WithTranslation(Vec3{7, 8, 9}), NewMat4Translation(Vec3{7, 8, 9})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got.NearlyEqual(tt.want, eps) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestMatrixTransformsKnownValues(t *testing.T) {
	m := NewMat4Translation(Vec3{10, 0, 0}).Mul(NewMat4RotationZ(math.Pi / 2))
	tests := []struct {
		name string
		got  Vec3
		want Vec3
	}{
		{"point", m.TransformPoint(Vec3{1, 0, 0}), Vec3{10, 1, 0}},
		{"direction ignores the translation", m.TransformDirection(Vec3{1, 0, 0}), Vec3{0, 1, 0}},
		{"look at moves the eye to the origin", LookAt(Vec3{3, 4, 5}, Vec3Zero, Vec3Y).TransformPoint(Vec3{3, 4, 5}), Vec3Zero},
		{"look at sends the target down -z", LookAt(Vec3{0, 0, 5}, Vec3{0, 0, 1}, Vec3Y).TransformPoint(Vec3{0, 0, 1}), Vec3{0, 0, -4}},
		{"trs", NewMat4TRS(Vec3{1, 2, 3}, NewQuatAxisAngle(Vec3Z, math.Pi/2), Vec3{2, 2, 2}).TransformPoint(Vec3{1, 0, 0}), Vec3{1, 4, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !nearVec(tt.got, tt.want, eps) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestDeterminantsKnownValues(t *testing.T) {
	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"identity", Mat4Identity().Det(), 1},
		{"scale", NewMat4Scale(Vec3{2, 3, 4}).Det(), 24},
		{"rotation", NewMat4RotationY(1.2).Det(), 1},
		{"singular", Mat4{1, 2, 3, 4, 2, 4, 6, 8, 0, 0, 1, 0, 0, 0, 0, 1}.Det(), 0},
		{"mat3 scale", Mat3{2, 0, 0, 0, 3, 0, 0, 0, 4}.Det(), 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !near(tt.got, tt.want, 1e-9) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestInverseOfASingularMatrixSaysSo(t *testing.T) {
	nan := math.NaN()
	for name, m := range map[string]Mat4{
		"zero":         {},
		"repeated":     {1, 2, 3, 4, 1, 2, 3, 4, 0, 0, 1, 0, 0, 0, 0, 1},
		"flat scale":   NewMat4Scale(Vec3{1, 1, 0}),
		"not a number": {nan, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1},
	} {
		if _, ok := m.Inverse(); ok {
			t.Errorf("%s: Inverse reported a singular matrix as invertible", name)
		}
	}
	if _, ok := (Mat3{1, 2, 3, 2, 4, 6, 0, 0, 1}).Inverse(); ok {
		t.Error("Mat3.Inverse reported a singular matrix as invertible")
	}
	if got := (Mat4{}).NormalMatrix(); got != Mat3Identity() {
		t.Errorf("NormalMatrix of a singular matrix = %v, want the identity", got)
	}
}

// randMat4 returns a random invertible transform: a rotation, a scale and a translation, sometimes with a projection.
func randMat4(r *rand.Rand, projective bool) Mat4 {
	m := NewMat4TRS(randVec(r, 50), randRotation(r), Vec3{0.5 + r.Float64()*3, 0.5 + r.Float64()*3, 0.5 + r.Float64()*3})
	if projective {
		m = Perspective(0.3+r.Float64(), 0.5+r.Float64()*2, 0.1+r.Float64()).Mul(LookAt(randVec(r, 20).Add(Vec3{Z: 30}), randVec(r, 5), Vec3Y)).Mul(m)
	}
	return m
}

func TestInverseTimesMatrixIsIdentity(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 500; i++ {
		m := randMat4(r, i%2 == 1)
		inv, ok := m.Inverse()
		if !ok {
			t.Fatalf("Inverse(%v) reported singular", m)
		}
		// The infinite far plane makes a projection badly scaled: compare in relative terms.
		scale := 1.0
		for _, v := range m {
			scale = math.Max(scale, math.Abs(v))
		}
		if p := m.Mul(inv); !p.NearlyEqual(Mat4Identity(), 1e-9*scale*scale) {
			t.Fatalf("m * inverse = %v for m = %v", p, m)
		}
		if p := inv.Mul(m); !p.NearlyEqual(Mat4Identity(), 1e-9*scale*scale) {
			t.Fatalf("inverse * m = %v for m = %v", p, m)
		}
	}
	for i := 0; i < 200; i++ {
		m := randRotation(r).Mat3().Mul(Mat3{1 + r.Float64(), 0, 0, 0, 1 + r.Float64(), 0, 0, 0, 1 + r.Float64()})
		inv, ok := m.Inverse()
		if !ok {
			t.Fatalf("Mat3.Inverse(%v) reported singular", m)
		}
		if p := m.Mul(inv); p != Mat3Identity() {
			for j, v := range p {
				if !near(v, Mat3Identity()[j], 1e-12) {
					t.Fatalf("Mat3: m * inverse = %v", p)
				}
			}
		}
	}
}

func TestDeterminantOfAProductIsTheProductOfTheDeterminants(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < 200; i++ {
		a, b := randMat4(r, false), randMat4(r, false)
		want := a.Det() * b.Det()
		if got := a.Mul(b).Det(); !near(got, want, 1e-9*math.Abs(want)) {
			t.Fatalf("det(a*b) = %v, want %v", got, want)
		}
	}
}

func TestMultiplyingMatricesIsAssociative(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		a, b, c := randMat4(r, false), randMat4(r, false), randMat4(r, false)
		l, rr := a.Mul(b).Mul(c), a.Mul(b.Mul(c))
		scale := 1.0
		for _, v := range l {
			scale = math.Max(scale, math.Abs(v))
		}
		if !l.NearlyEqual(rr, 1e-9*scale) {
			t.Fatalf("(ab)c = %v, a(bc) = %v", l, rr)
		}
	}
}

func TestRotationsKeepLengthsAndAnglesAndAgreeWithMatrices(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for i := 0; i < 300; i++ {
		q := randRotation(r)
		a, b := randVec(r, 10), randVec(r, 10)
		ra, rb := q.Rotate(a), q.Rotate(b)
		if !near(ra.Length(), a.Length(), 1e-9) {
			t.Fatalf("a rotation changed the length of %v from %v to %v", a, a.Length(), ra.Length())
		}
		if !near(ra.Dot(rb), a.Dot(b), 1e-8) {
			t.Fatalf("a rotation changed the dot product: %v to %v", a.Dot(b), ra.Dot(rb))
		}
		if m := q.Mat3().MulVec3(a); !nearVec(m, ra, 1e-9) {
			t.Fatalf("Mat3 turns %v to %v, Rotate to %v", a, m, ra)
		}
		if m := q.Mat4().TransformPoint(a); !nearVec(m, ra, 1e-9) {
			t.Fatalf("Mat4 turns %v to %v, Rotate to %v", a, m, ra)
		}
		if back := q.Inverse().Rotate(ra); !nearVec(back, a, 1e-9) {
			t.Fatalf("the inverse rotation gave %v, want %v", back, a)
		}
		if d := q.Mat3().Det(); !near(d, 1, 1e-9) {
			t.Fatalf("the determinant of a rotation is %v", d)
		}
		// Composition: (p*q) applies q first.
		p := randRotation(r)
		if got, want := p.Mul(q).Rotate(a), p.Rotate(q.Rotate(a)); !nearVec(got, want, 1e-9) {
			t.Fatalf("(p*q) a = %v, p(q a) = %v", got, want)
		}
		if got, want := p.Mat3().Mul(q.Mat3()).MulVec3(a), p.Mul(q).Rotate(a); !nearVec(got, want, 1e-9) {
			t.Fatalf("matrix product %v, quaternion product %v", got, want)
		}
	}
}

func TestQuaternionsKnownValues(t *testing.T) {
	tests := []struct {
		name string
		got  Vec3
		want Vec3
	}{
		{"a quarter turn about z", NewQuatAxisAngle(Vec3Z, math.Pi/2).Rotate(Vec3X), Vec3Y},
		{"a half turn about y", NewQuatAxisAngle(Vec3Y, math.Pi).Rotate(Vec3X), Vec3{X: -1}},
		{"an axis that is not a unit vector", NewQuatAxisAngle(Vec3{0, 0, 7}, math.Pi/2).Rotate(Vec3X), Vec3Y},
		{"no axis", NewQuatAxisAngle(Vec3Zero, 1).Rotate(Vec3{1, 2, 3}), Vec3{1, 2, 3}},
		{"from to", NewQuatFromTo(Vec3X, Vec3Y).Rotate(Vec3X), Vec3Y},
		{"from to, opposite", NewQuatFromTo(Vec3X, Vec3X.Neg()).Rotate(Vec3X), Vec3X.Neg()},
		{"from to, the same", NewQuatFromTo(Vec3Z, Vec3{0, 0, 5}).Rotate(Vec3{1, 2, 3}), Vec3{1, 2, 3}},
		{"from to, zero", NewQuatFromTo(Vec3Zero, Vec3Z).Rotate(Vec3{1, 2, 3}), Vec3{1, 2, 3}},
		{"slerp halfway", NewQuatAxisAngle(Vec3Z, 0).Slerp(NewQuatAxisAngle(Vec3Z, math.Pi), 0.5).Rotate(Vec3X), Vec3Y},
		{"slerp of the same", QuatIdentity().Slerp(QuatIdentity(), 0.3).Rotate(Vec3X), Vec3X},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !nearVec(tt.got, tt.want, 1e-9) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestQuaternionFromTwoDirectionsAndBasisAndAxisAngleRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(6))
	for i := 0; i < 300; i++ {
		a, b := randVec(r, 5), randVec(r, 5)
		if got := NewQuatFromTo(a, b).Rotate(a.Normalize()); !nearVec(got, b.Normalize(), 1e-9) {
			t.Fatalf("FromTo(%v, %v) turned a into %v", a, b, got)
		}
		q := randRotation(r)
		x, y, z := q.Rotate(Vec3X), q.Rotate(Vec3Y), q.Rotate(Vec3Z)
		got := NewQuatBasis(x, y, z)
		if v := got.Rotate(Vec3{1, 2, 3}); !nearVec(v, q.Rotate(Vec3{1, 2, 3}), 1e-9) {
			t.Fatalf("NewQuatBasis gave a rotation that turns (1,2,3) to %v, want %v", v, q.Rotate(Vec3{1, 2, 3}))
		}
		axis, angle := q.AxisAngle()
		if v := NewQuatAxisAngle(axis, angle).Rotate(Vec3{3, 2, 1}); !nearVec(v, q.Rotate(Vec3{3, 2, 1}), 1e-9) {
			t.Fatalf("AxisAngle round trip: %v, want %v", v, q.Rotate(Vec3{3, 2, 1}))
		}
		// Slerp keeps unit length and meets both ends.
		p := randRotation(r)
		for _, tt := range []float64{0, 0.25, 0.5, 1} {
			if l := p.Slerp(q, tt).Length(); !near(l, 1, 1e-9) {
				t.Fatalf("Slerp(%v) has length %v", tt, l)
			}
		}
		if v := p.Slerp(q, 0).Rotate(a); !nearVec(v, p.Rotate(a), 1e-9) {
			t.Fatalf("Slerp(0) = %v, want %v", v, p.Rotate(a))
		}
		if v := p.Slerp(q, 1).Rotate(a); !nearVec(v, q.Rotate(a), 1e-9) {
			t.Fatalf("Slerp(1) = %v, want %v", v, q.Rotate(a))
		}
	}
	if axis, angle := QuatIdentity().AxisAngle(); axis != Vec3X || angle != 0 {
		t.Errorf("the identity's axis and angle = %v, %v", axis, angle)
	}
	if q := (Quat{}).Normalize(); q != QuatIdentity() {
		t.Errorf("Normalize of the zero quaternion = %v, want the identity", q)
	}
	if q := (Quat{}).Inverse(); q != QuatIdentity() {
		t.Errorf("Inverse of the zero quaternion = %v, want the identity", q)
	}
}

func TestLookAtIsARigidMotionThatPutsTheEyeAtTheOriginAndTheTargetOnTheAxis(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 300; i++ {
		eye, center := randVec(r, 100), randVec(r, 100)
		if eye.DistanceTo(center) < 1e-3 {
			continue
		}
		up := Vec3Y
		if d := center.Sub(eye).Normalize(); math.Abs(d.Dot(up)) > 0.999 {
			continue
		}
		v := LookAt(eye, center, up)
		if got := v.TransformPoint(eye); !nearVec(got, Vec3Zero, 1e-9) {
			t.Fatalf("the eye is at %v in view space", got)
		}
		c := v.TransformPoint(center)
		if want := (Vec3{Z: -eye.DistanceTo(center)}); !nearVec(c, want, 1e-8) {
			t.Fatalf("the center is at %v in view space, want %v", c, want)
		}
		if d := v.Det(); !near(d, 1, 1e-9) {
			t.Fatalf("a view matrix has the determinant %v", d)
		}
		// Up in the world has no negative y in the view: the camera is not upside down.
		if got := v.TransformDirection(up); got.Y < 0 {
			t.Fatalf("world up is %v in view space", got)
		}
	}
}

// ndcOf projects a view-space point and returns its normalized device coordinates.
func ndcOf(p Mat4, v Vec3) Vec3 {
	n, _ := p.ProjectPoint(v)
	return n
}

func TestProjectionsMapTheirFrustumCornersOntoClipSpace(t *testing.T) {
	const fov, aspect, nearZ, far = math.Pi / 3, 1.6, 0.5, 80.0
	th := math.Tan(fov / 2)
	corner := func(d float64, sx, sy float64) Vec3 { return Vec3{sx * d * th * aspect, sy * d * th, -d} }
	t.Run("perspective with a far plane", func(t *testing.T) {
		p := PerspectiveFinite(fov, aspect, nearZ, far)
		for _, c := range []struct{ d, z float64 }{{nearZ, 1}, {far, 0}} {
			for _, sx := range []float64{-1, 1} {
				for _, sy := range []float64{-1, 1} {
					if got := ndcOf(p, corner(c.d, sx, sy)); !nearVec(got, Vec3{sx, sy, c.z}, 1e-9) {
						t.Errorf("corner (%v, %v) at %v maps to %v, want %v", sx, sy, c.d, got, Vec3{sx, sy, c.z})
					}
				}
			}
		}
	})
	t.Run("perspective with the far plane at infinity", func(t *testing.T) {
		p := Perspective(fov, aspect, nearZ)
		for _, sx := range []float64{-1, 1} {
			for _, sy := range []float64{-1, 1} {
				if got := ndcOf(p, corner(nearZ, sx, sy)); !nearVec(got, Vec3{sx, sy, 1}, 1e-9) {
					t.Errorf("nearZ corner (%v, %v) maps to %v", sx, sy, got)
				}
				if got := ndcOf(p, corner(1e9, sx, sy)); !nearVec(got, Vec3{sx, sy, 0}, 1e-6) {
					t.Errorf("a far corner (%v, %v) maps to %v, want a depth of about 0", sx, sy, got)
				}
			}
		}
		// The depth falls as 1/d: a point twice as far has half the depth.
		if z := ndcOf(p, Vec3{0, 0, -2 * nearZ}).Z; !near(z, 0.5, 1e-12) {
			t.Errorf("depth at twice the nearZ distance = %v, want 0.5", z)
		}
	})
	t.Run("orthographic", func(t *testing.T) {
		const l, r, b, tp = -4.0, 6.0, -1.0, 3.0
		p := Orthographic(l, r, b, tp, 0.5, 20)
		for _, c := range []struct{ d, z float64 }{{0.5, 1}, {20, 0}} {
			for _, sx := range []float64{l, r} {
				for _, sy := range []float64{b, tp} {
					want := Vec3{-1, -1, c.z}
					if sx == r {
						want.X = 1
					}
					if sy == tp {
						want.Y = 1
					}
					if got := ndcOf(p, Vec3{sx, sy, -c.d}); !nearVec(got, want, 1e-9) {
						t.Errorf("corner (%v, %v) at %v maps to %v, want %v", sx, sy, c.d, got, want)
					}
				}
			}
		}
		// The depth is linear.
		if z := ndcOf(p, Vec3{0, 0, -(0.5 + 20) / 2}).Z; !near(z, 0.5, 1e-12) {
			t.Errorf("depth halfway = %v, want 0.5", z)
		}
		// A nearZ plane behind the eye is allowed.
		q := Orthographic(-1, 1, -1, 1, -10, 10)
		if z := ndcOf(q, Vec3{}).Z; !near(z, 0.5, 1e-12) {
			t.Errorf("depth of the eye with nearZ -10 and far 10 = %v, want 0.5", z)
		}
	})
	t.Run("a projection is undone by its inverse", func(t *testing.T) {
		r := rand.New(rand.NewSource(8))
		for _, p := range []Mat4{Perspective(fov, aspect, nearZ), PerspectiveFinite(fov, aspect, nearZ, far), Orthographic(-3, 3, -2, 2, 0.1, 50)} {
			inv, ok := p.Inverse()
			if !ok {
				t.Fatalf("a projection %v is singular", p)
			}
			for i := 0; i < 100; i++ {
				v := Vec3{(r.Float64()*2 - 1) * 2, (r.Float64()*2 - 1) * 2, -(nearZ + r.Float64()*10)}
				n, w := p.ProjectPoint(v)
				back := inv.MulVec4(Vec4{n.X, n.Y, n.Z, 1}).PerspectiveDivide()
				if !nearVec(back, v, 1e-7) || w == 0 {
					t.Fatalf("unprojecting %v gave %v", v, back)
				}
			}
		}
	})
}

func TestARayMadeFromAPlaneMeetsThePlane(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	for i := 0; i < 300; i++ {
		point, normal := randVec(r, 50), randVec(r, 1)
		if normal.Length() < 0.1 {
			continue
		}
		pl := NewPlaneFromPointNormal(point, normal)
		if d := pl.DistanceTo(point); !near(d, 0, 1e-9) {
			t.Fatalf("a point of the plane is %v from it", d)
		}
		// A ray from a point on the front side to a point on the plane.
		target := point.Add(normal.Perpendicular().Scale(r.Float64() * 20))
		origin := target.Add(pl.Normal.Scale(1 + r.Float64()*30))
		ray := Ray{origin, target.Sub(origin).Normalize()}
		tt, ok := ray.IntersectPlane(pl)
		if !ok {
			t.Fatalf("the ray %v missed the plane %v", ray, pl)
		}
		if got := ray.At(tt); !nearVec(got, target, 1e-8) {
			t.Fatalf("the ray meets the plane at %v, want %v", got, target)
		}
		if d := pl.DistanceTo(ray.At(tt)); !near(d, 0, 1e-8) {
			t.Fatalf("the hit is %v from the plane", d)
		}
		if got := pl.ClosestPoint(origin); !nearVec(got, pl.ClosestPoint(target.Add(pl.Normal.Scale(pl.DistanceTo(origin)))), 1e-8) {
			t.Fatalf("ClosestPoint = %v", got)
		}
		// The same ray pointing the other way never meets it.
		if _, ok := (Ray{origin, ray.Dir.Neg()}).IntersectPlane(pl); ok {
			t.Fatalf("a ray that points away met the plane")
		}
		if flipped := pl.Flip(); !near(flipped.DistanceTo(origin), -pl.DistanceTo(origin), 1e-9) {
			t.Fatalf("Flip did not negate the distance")
		}
	}
	if _, ok := (Ray{Vec3{0, 1, 0}, Vec3X}).IntersectPlane(NewPlaneFromPointNormal(Vec3Zero, Vec3Y)); ok {
		t.Error("a ray parallel to the plane met it")
	}
}

func TestRayIntersectionsKnownValues(t *testing.T) {
	box := NewBox3(Vec3{-1, -1, -1}, Vec3{1, 1, 1})
	sphere := Sphere{Vec3{0, 0, 0}, 2}
	tests := []struct {
		name string
		t    float64
		ok   bool
		got  func() (float64, bool)
	}{
		{"box hit", 4, true, func() (float64, bool) { return Ray{Vec3{0, 0, 5}, Vec3{Z: -1}}.IntersectBox(box) }},
		{"box hit from inside", 0, true, func() (float64, bool) { return Ray{Vec3{}, Vec3{Z: -1}}.IntersectBox(box) }},
		{"box miss", 0, false, func() (float64, bool) { return Ray{Vec3{3, 0, 5}, Vec3{Z: -1}}.IntersectBox(box) }},
		{"box behind", 0, false, func() (float64, bool) { return Ray{Vec3{0, 0, 5}, Vec3{Z: 1}}.IntersectBox(box) }},
		{"box on the diagonal", 2 * math.Sqrt(3), true, func() (float64, bool) {
			return Ray{Vec3{3, 3, 3}, Vec3{-1, -1, -1}.Normalize()}.IntersectBox(box)
		}},
		{"sphere hit", 3, true, func() (float64, bool) { return Ray{Vec3{0, 0, 5}, Vec3{Z: -1}}.IntersectSphere(sphere) }},
		{"sphere miss", 0, false, func() (float64, bool) { return Ray{Vec3{3, 0, 5}, Vec3{Z: -1}}.IntersectSphere(sphere) }},
		{"sphere from inside", 0, true, func() (float64, bool) { return Ray{Vec3{}, Vec3{Z: -1}}.IntersectSphere(sphere) }},
		{"sphere behind", 0, false, func() (float64, bool) { return Ray{Vec3{0, 0, 5}, Vec3{Z: 1}}.IntersectSphere(sphere) }},
		{"triangle hit", 2, true, func() (float64, bool) {
			return Ray{Vec3{0.2, 0.2, 2}, Vec3{Z: -1}}.IntersectTriangle(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		}},
		{"triangle miss", 0, false, func() (float64, bool) {
			return Ray{Vec3{0.8, 0.8, 2}, Vec3{Z: -1}}.IntersectTriangle(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		}},
		{"triangle parallel", 0, false, func() (float64, bool) {
			return Ray{Vec3{0.2, 0.2, 2}, Vec3{X: 1}}.IntersectTriangle(Vec3{0, 0, 0}, Vec3{1, 0, 0}, Vec3{0, 1, 0})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.got()
			if ok != tt.ok || (ok && !near(got, tt.t, 1e-9)) {
				t.Errorf("got %v, %v; want %v, %v", got, ok, tt.t, tt.ok)
			}
		})
	}
}

func TestBoxesAndSpheresKnownValues(t *testing.T) {
	b := NewBox3(Vec3{1, 2, 3}, Vec3{-1, 0, 5})
	if b.Min != (Vec3{-1, 0, 3}) || b.Max != (Vec3{1, 2, 5}) {
		t.Fatalf("NewBox3 = %v", b)
	}
	if c := b.Center(); c != (Vec3{0, 1, 4}) {
		t.Errorf("Center = %v", c)
	}
	if s := b.Size(); s != (Vec3{2, 2, 2}) {
		t.Errorf("Size = %v", s)
	}
	if !b.Contains(Vec3{0, 1, 4}) || !b.Contains(b.Min) || b.Contains(Vec3{0, 1, 6}) {
		t.Error("Contains is wrong")
	}
	if !b.Intersects(NewBox3(Vec3{0, 0, 0}, Vec3{1, 1, 3})) || b.Intersects(NewBox3(Vec3{2, 0, 0}, Vec3{3, 1, 3})) {
		t.Error("Intersects is wrong")
	}
	if got := b.DistanceTo(Vec3{4, 1, 4}); !near(got, 3, eps) {
		t.Errorf("DistanceTo = %v, want 3", got)
	}
	if got := b.DistanceTo(Vec3{0, 1, 4}); got != 0 {
		t.Errorf("DistanceTo of an inner point = %v, want 0", got)
	}
	e := EmptyBox3()
	if !e.IsEmpty() || b.IsEmpty() {
		t.Error("IsEmpty is wrong")
	}
	if u := e.Union(b); u != b {
		t.Errorf("empty.Union(b) = %v, want b", u)
	}
	if u := e.Expand(Vec3{1, 2, 3}); u.Min != u.Max || u.Min != (Vec3{1, 2, 3}) {
		t.Errorf("empty.Expand(p) = %v", u)
	}
	if s := b.BoundingSphere(); !near(s.Radius, math.Sqrt(3), eps) || s.Center != b.Center() {
		t.Errorf("BoundingSphere = %v", s)
	}
	if s := e.BoundingSphere(); s != (Sphere{}) {
		t.Errorf("the bounding sphere of an empty box = %v", s)
	}
	if got := e.Transform(Mat4Identity()); !got.IsEmpty() {
		t.Errorf("the transform of an empty box = %v", got)
	}
	for i := 0; i < 8; i++ {
		c := b.Corner(i)
		if !b.Contains(c) || (c.X == b.Max.X) != (i&1 != 0) || (c.Y == b.Max.Y) != (i&2 != 0) || (c.Z == b.Max.Z) != (i&4 != 0) {
			t.Errorf("Corner(%d) = %v", i, c)
		}
	}
	s := Sphere{Vec3{0, 0, 0}, 1}
	if !s.Contains(Vec3{0, 1, 0}) || s.Contains(Vec3{0, 1.1, 0}) {
		t.Error("Sphere.Contains is wrong")
	}
	if !s.Intersects(Sphere{Vec3{1.5, 0, 0}, 0.5}) || s.Intersects(Sphere{Vec3{3, 0, 0}, 1}) {
		t.Error("Sphere.Intersects is wrong")
	}
}

func TestATransformedBoxHoldsTheTransformedCorners(t *testing.T) {
	r := rand.New(rand.NewSource(10))
	for i := 0; i < 200; i++ {
		b := NewBox3(randVec(r, 10), randVec(r, 10))
		m := randMat4(r, false)
		tb := b.Transform(m)
		for j := 0; j < 8; j++ {
			c := m.TransformPoint(b.Corner(j))
			if !tb.Contains(c.Add(Vec3{})) && tb.DistanceTo(c) > 1e-9 {
				t.Fatalf("the transformed box %v does not hold %v", tb, c)
			}
		}
	}
}

func TestFrustumContainment(t *testing.T) {
	view := LookAt(Vec3{0, 0, 10}, Vec3Zero, Vec3Y)
	tests := []struct {
		name string
		proj Mat4
		in   []Vec3
		out  []Vec3
	}{
		{"perspective", Perspective(math.Pi/2, 1, 1),
			[]Vec3{{0, 0, 0}, {0, 0, 5}, {8, 0, 0}, {-8, 8, 0}, {0, 0, 8.9}, {0, 0, -1000}},
			[]Vec3{{0, 0, 9.5}, {0, 0, 11}, {11, 0, 0}, {0, -11, 0}, {0, 0, 10}}},
		{"finite perspective", PerspectiveFinite(math.Pi/2, 1, 1, 20),
			[]Vec3{{0, 0, 0}, {0, 0, -9}},
			[]Vec3{{0, 0, -11}, {0, 0, 9.5}, {20, 0, 0}}},
		{"orthographic", Orthographic(-2, 2, -2, 2, 1, 20),
			[]Vec3{{0, 0, 0}, {1.9, -1.9, -9}},
			[]Vec3{{2.3, 0, 0}, {0, 0, 9.5}, {0, 0, -11}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFrustum(tt.proj.Mul(view))
			for _, p := range tt.in {
				if !f.ContainsPoint(p) {
					t.Errorf("the frustum does not hold %v", p)
				}
				if !f.IntersectsSphere(Sphere{p, 0.1}) || !f.IntersectsBox(NewBox3(p.Sub(Vec3{0.1, 0.1, 0.1}), p.Add(Vec3{0.1, 0.1, 0.1}))) {
					t.Errorf("the frustum does not meet a small ball and box around %v", p)
				}
			}
			for _, p := range tt.out {
				if f.ContainsPoint(p) {
					t.Errorf("the frustum holds %v", p)
				}
				if f.IntersectsSphere(Sphere{p, 0.1}) || f.IntersectsBox(NewBox3(p.Sub(Vec3{0.1, 0.1, 0.1}), p.Add(Vec3{0.1, 0.1, 0.1}))) {
					t.Errorf("the frustum meets a small ball or box around %v", p)
				}
			}
			if f.IntersectsBox(EmptyBox3()) {
				t.Error("the frustum meets an empty box")
			}
		})
	}
}

func TestConversionsToFloat32AndToMatrix4(t *testing.T) {
	m := Mat4{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16.5}
	f := m.Float32()
	for i, v := range m {
		if float64(f[i]) != v {
			t.Errorf("Float32()[%d] = %v, want %v", i, f[i], v)
		}
	}
	u := m.Matrix4()
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if got, want := float64(u.Entry(r, c)), m[c*4+r]; got != want {
				t.Errorf("Matrix4().Entry(%d, %d) = %v, want %v", r, c, got, want)
			}
		}
	}
	if back := NewMat4FromMatrix4(&u); back != m {
		t.Errorf("round trip = %v, want %v", back, m)
	}
	if got := (Vec3{1.5, -2, 1e-3}).Float32(); got != [3]float32{1.5, -2, 1e-3} {
		t.Errorf("Vec3.Float32 = %v", got)
	}
	if got := (Vec2{1.5, -2}).Float32(); got != [2]float32{1.5, -2} {
		t.Errorf("Vec2.Float32 = %v", got)
	}
	if got := (Vec4{1, 2, 3, 4}).Float32(); got != [4]float32{1, 2, 3, 4} {
		t.Errorf("Vec4.Float32 = %v", got)
	}
}

func TestOperationsDoNotAllocate(t *testing.T) {
	a, b := Vec3{1, 2, 3}, Vec3{4, 5, 6}
	m := NewMat4TRS(Vec3{1, 2, 3}, NewQuatAxisAngle(Vec3Y, 0.3), Vec3{1, 2, 3})
	p := PerspectiveFinite(1, 1.5, 0.1, 100)
	box := NewBox3(Vec3{-1, -1, -1}, Vec3{1, 1, 1})
	var sink float64
	var sinkV Vec3
	var sinkM Mat4
	var sinkF [16]float32
	allocs := testing.AllocsPerRun(100, func() {
		sinkV = a.Add(b).Sub(a).Scale(2).Cross(b).Normalize().Lerp(a, 0.5)
		sink += a.Dot(b) + a.Length() + a.DistanceTo(b) + a.AngleTo(b)
		q := NewQuatAxisAngle(Vec3Y, 0.3).Mul(NewQuatFromTo(a, b)).Normalize()
		sinkV = q.Slerp(QuatIdentity(), 0.5).Rotate(a)
		sinkM = m.Mul(p).Mul(LookAt(a, b, Vec3Y))
		inv, _ := sinkM.Inverse()
		sinkM = inv.Transpose().Mul(NewMat4RotationX(0.2))
		sink += sinkM.Det() + p.MulVec4(Vec4{1, 2, 3, 1}).W
		sinkV = m.TransformPoint(a)
		sinkV, _ = p.ProjectPoint(a)
		sinkV = m.NormalMatrix().MulVec3(a)
		sinkF = sinkM.Float32()
		u := sinkM.Matrix4()
		sink += float64(u.Entry(0, 0))
		f := NewFrustum(p)
		if f.IntersectsBox(box.Transform(m)) {
			sink++
		}
		r := Ray{a, b.Normalize()}
		_, _ = r.IntersectBox(box)
		_, _ = r.IntersectPlane(NewPlaneFromPointNormal(a, b))
		_, _ = r.IntersectSphere(box.BoundingSphere())
		_, _ = r.IntersectTriangle(a, b, Vec3Z)
	})
	_, _, _ = sink, sinkV, sinkF
	if allocs != 0 {
		t.Errorf("%v allocations per run, want 0", allocs)
	}
}
