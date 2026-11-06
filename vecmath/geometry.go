package vecmath

import "math"

// Ray is a half line: the points Origin + t*Dir for t >= 0.
//
// Flutter: package:vector_math Ray
type Ray struct {
	// Origin is where the ray starts.
	Origin Vec3
	// Dir is the direction of the ray, normally of length 1; the t of the methods is in units of its length.
	Dir Vec3
}

// At returns the point at t along the ray.
//
// Flutter: package:vector_math Ray.at
func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Dir.Scale(t)) }

// IntersectPlane returns the t where the ray meets the plane, and false when it never does: the ray is parallel to the
// plane, or the plane is behind the origin.
//
// Flutter: no counterpart
func (r Ray) IntersectPlane(p Plane) (float64, bool) {
	d := p.Normal.Dot(r.Dir)
	if d == 0 {
		return 0, false
	}
	t := -p.DistanceTo(r.Origin) / d
	return t, t >= 0
}

// IntersectBox returns the t where the ray enters the box, which is 0 when the origin is inside, and false when the
// ray misses the box.
//
// Flutter: package:vector_math Ray.intersectsWithAabb3
func (r Ray) IntersectBox(b Box3) (float64, bool) {
	tmin, tmax := 0.0, math.Inf(1)
	for axis := 0; axis < 3; axis++ {
		o, d, lo, hi := r.Origin.axis(axis), r.Dir.axis(axis), b.Min.axis(axis), b.Max.axis(axis)
		if d == 0 {
			if o < lo || o > hi {
				return 0, false
			}
			continue
		}
		t0, t1 := (lo-o)/d, (hi-o)/d
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		tmin, tmax = math.Max(tmin, t0), math.Min(tmax, t1)
		if tmin > tmax {
			return 0, false
		}
	}
	return tmin, true
}

// IntersectSphere returns the t where the ray enters the sphere (0 when the origin is inside), and false when it
// misses.
//
// Flutter: package:vector_math Ray.intersectsWithSphere
func (r Ray) IntersectSphere(s Sphere) (float64, bool) {
	oc := r.Origin.Sub(s.Center)
	a := r.Dir.LengthSq()
	if a == 0 {
		return 0, false
	}
	b := oc.Dot(r.Dir)
	c := oc.LengthSq() - s.Radius*s.Radius
	disc := b*b - a*c
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	if t := (-b - root) / a; t >= 0 {
		return t, true
	}
	if t := (-b + root) / a; t >= 0 {
		return 0, true
	}
	return 0, false
}

// IntersectTriangle returns the t where the ray meets the triangle, and false when it misses it or is parallel.
//
// Flutter: package:vector_math Ray.intersectsWithTriangle
func (r Ray) IntersectTriangle(a, b, c Vec3) (float64, bool) {
	// Moller-Trumbore.
	e1, e2 := b.Sub(a), c.Sub(a)
	p := r.Dir.Cross(e2)
	det := e1.Dot(p)
	if det == 0 {
		return 0, false
	}
	inv := 1 / det
	s := r.Origin.Sub(a)
	u := s.Dot(p) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	q := s.Cross(e1)
	v := r.Dir.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	t := e2.Dot(q) * inv
	return t, t >= 0
}

// Transform returns the ray moved by the matrix: the origin as a point and the direction as a vector.
//
// Flutter: no counterpart
func (r Ray) Transform(m Mat4) Ray {
	return Ray{m.TransformPoint(r.Origin), m.TransformDirection(r.Dir)}
}

// axis returns the component 0, 1 or 2.
func (a Vec3) axis(i int) float64 {
	switch i {
	case 0:
		return a.X
	case 1:
		return a.Y
	}
	return a.Z
}

// Plane is the points p with Normal.Dot(p) + D = 0. When Normal has length 1, DistanceTo is the distance in the units of
// the scene, positive on the side that the normal points to.
//
// Flutter: package:vector_math Plane
type Plane struct {
	// Normal is the direction that is perpendicular to the plane.
	Normal Vec3
	// D is minus the dot product of the normal with any point of the plane.
	D float64
}

// NewPlaneFromPointNormal creates the plane through the point, perpendicular to the normal (which is made of length 1).
//
// Flutter: package:vector_math Plane.components
func NewPlaneFromPointNormal(point, normal Vec3) Plane {
	n := normal.Normalize()
	return Plane{n, -n.Dot(point)}
}

// NewPlaneFromPoints creates the plane through the three points; its normal is the one that a counterclockwise
// triangle a, b, c points to. Points on one line give the plane with the zero normal.
//
// Flutter: package:vector_math Plane.setFromPoints
func NewPlaneFromPoints(a, b, c Vec3) Plane {
	return NewPlaneFromPointNormal(a, b.Sub(a).Cross(c.Sub(a)))
}

// DistanceTo returns the signed distance of the point to the plane.
//
// Flutter: package:vector_math Plane.distanceToVector3
func (p Plane) DistanceTo(v Vec3) float64 { return p.Normal.Dot(v) + p.D }

// ClosestPoint returns the point of the plane that is nearest to v.
//
// Flutter: no counterpart
func (p Plane) ClosestPoint(v Vec3) Vec3 { return v.Sub(p.Normal.Scale(p.DistanceTo(v))) }

// Flip returns the plane that faces the other way.
//
// Flutter: no counterpart
func (p Plane) Flip() Plane { return Plane{p.Normal.Neg(), -p.D} }

// Box3 is an axis-aligned box. The empty box has Min above Max, so that Expand and Union of it are the other operand.
//
// Flutter: package:vector_math Aabb3
type Box3 struct {
	// Min is the corner with the smallest coordinates.
	Min Vec3
	// Max is the corner with the largest coordinates.
	Max Vec3
}

// EmptyBox3 returns the box that contains no point.
//
// Flutter: no counterpart
func EmptyBox3() Box3 {
	inf := math.Inf(1)
	return Box3{Vec3{inf, inf, inf}, Vec3{-inf, -inf, -inf}}
}

// NewBox3 creates the box with the two corners, in any order.
//
// Flutter: package:vector_math Aabb3.minMax
func NewBox3(a, b Vec3) Box3 { return Box3{a.Min(b), a.Max(b)} }

// IsEmpty says whether the box contains no point.
//
// Flutter: no counterpart
func (b Box3) IsEmpty() bool { return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y || b.Min.Z > b.Max.Z }

// Center returns the middle of the box.
//
// Flutter: package:vector_math Aabb3.center
func (b Box3) Center() Vec3 { return b.Min.Add(b.Max).Scale(0.5) }

// Size returns the lengths of the three sides.
//
// Flutter: no counterpart
func (b Box3) Size() Vec3 { return b.Max.Sub(b.Min) }

// Expand returns the smallest box that holds the box and the point.
//
// Flutter: package:vector_math Aabb3.hullPoint
func (b Box3) Expand(p Vec3) Box3 { return Box3{b.Min.Min(p), b.Max.Max(p)} }

// Union returns the smallest box that holds both boxes.
//
// Flutter: package:vector_math Aabb3.hull
func (b Box3) Union(o Box3) Box3 { return Box3{b.Min.Min(o.Min), b.Max.Max(o.Max)} }

// Contains says whether the point is in the box, or on its surface.
//
// Flutter: package:vector_math Aabb3.containsVector3
func (b Box3) Contains(p Vec3) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X && p.Y >= b.Min.Y && p.Y <= b.Max.Y && p.Z >= b.Min.Z && p.Z <= b.Max.Z
}

// Intersects says whether the boxes share a point.
//
// Flutter: package:vector_math Aabb3.intersectsWithAabb3
func (b Box3) Intersects(o Box3) bool {
	return b.Min.X <= o.Max.X && b.Max.X >= o.Min.X && b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y &&
		b.Min.Z <= o.Max.Z && b.Max.Z >= o.Min.Z
}

// Corner returns one of the eight corners: bit 0 of i picks the x, bit 1 the y and bit 2 the z of Max over Min.
//
// Flutter: no counterpart
func (b Box3) Corner(i int) Vec3 {
	p := b.Min
	if i&1 != 0 {
		p.X = b.Max.X
	}
	if i&2 != 0 {
		p.Y = b.Max.Y
	}
	if i&4 != 0 {
		p.Z = b.Max.Z
	}
	return p
}

// Transform returns the smallest box that holds the box moved by the matrix.
//
// Flutter: package:vector_math Aabb3.transform
func (b Box3) Transform(m Mat4) Box3 {
	if b.IsEmpty() {
		return b
	}
	out := EmptyBox3()
	for i := 0; i < 8; i++ {
		out = out.Expand(m.TransformPoint(b.Corner(i)))
	}
	return out
}

// BoundingSphere returns the sphere that has the box's center and holds its corners.
//
// Flutter: no counterpart
func (b Box3) BoundingSphere() Sphere {
	if b.IsEmpty() {
		return Sphere{}
	}
	return Sphere{b.Center(), b.Size().Length() / 2}
}

// DistanceTo returns the distance from the point to the box, which is 0 for a point inside.
//
// Flutter: no counterpart
func (b Box3) DistanceTo(p Vec3) float64 {
	d := Vec3{
		math.Max(math.Max(b.Min.X-p.X, 0), p.X-b.Max.X),
		math.Max(math.Max(b.Min.Y-p.Y, 0), p.Y-b.Max.Y),
		math.Max(math.Max(b.Min.Z-p.Z, 0), p.Z-b.Max.Z),
	}
	return d.Length()
}

// Sphere is a ball.
//
// Flutter: package:vector_math Sphere
type Sphere struct {
	// Center is the middle of the ball.
	Center Vec3
	// Radius is the distance from the center to the surface.
	Radius float64
}

// Contains says whether the point is in the ball.
//
// Flutter: package:vector_math Sphere.containsVector3
func (s Sphere) Contains(p Vec3) bool { return s.Center.DistanceSqTo(p) <= s.Radius*s.Radius }

// Intersects says whether the balls share a point.
//
// Flutter: package:vector_math Sphere.intersectsWithSphere
func (s Sphere) Intersects(o Sphere) bool {
	r := s.Radius + o.Radius
	return s.Center.DistanceSqTo(o.Center) <= r*r
}

// Frustum is the six planes of the volume that a view matrix and a projection see; the normals point inwards, so a
// point is inside when its distance to every plane is not negative.
//
// Flutter: package:vector_math Frustum
type Frustum struct {
	// Planes are, in order, the left, right, bottom, top, near and far planes. The far plane of a projection with an
	// infinite far plane has the zero normal and a distance above 0, so that every point is on its inner side.
	Planes [6]Plane
}

// NewFrustum creates the frustum of a matrix that is a projection times a view (the clip-space matrix of a camera), in
// WebGPU's clip space.
//
// Flutter: package:vector_math Frustum.matrix
func NewFrustum(viewProjection Mat4) Frustum {
	m := viewProjection
	row := func(r int) Vec4 { return Vec4{m[r], m[4+r], m[8+r], m[12+r]} }
	r0, r1, r2, r3 := row(0), row(1), row(2), row(3)
	// A clip point is inside when -w <= x <= w, -w <= y <= w and 0 <= z <= w. Whether the near plane is the z = 0 or the
	// z = w one depends on reversed-Z; the two planes are the near and the far one in the order that the depth gives.
	combos := [6]Vec4{r3.Add(r0), r3.Sub(r0), r3.Add(r1), r3.Sub(r1), r2, r3.Sub(r2)}
	var f Frustum
	for i, c := range combos {
		n := Vec3{c.X, c.Y, c.Z}
		l := n.Length()
		if l == 0 {
			f.Planes[i] = Plane{D: c.W}
			continue
		}
		f.Planes[i] = Plane{Vec3{n.X / l, n.Y / l, n.Z / l}, c.W / l}
	}
	return f
}

// ContainsPoint says whether the point is inside the frustum.
//
// Flutter: package:vector_math Frustum.containsVector3
func (f Frustum) ContainsPoint(p Vec3) bool {
	for _, pl := range f.Planes {
		if pl.DistanceTo(p) < 0 {
			return false
		}
	}
	return true
}

// IntersectsSphere says whether the sphere is at least partly inside the frustum. It can say true for a sphere that is
// outside near a corner of the frustum.
//
// Flutter: package:vector_math Frustum.intersectsWithSphere
func (f Frustum) IntersectsSphere(s Sphere) bool {
	for _, pl := range f.Planes {
		if pl.DistanceTo(s.Center) < -s.Radius {
			return false
		}
	}
	return true
}

// IntersectsBox says whether the box is at least partly inside the frustum. It can say true for a box that is outside
// near an edge of the frustum.
//
// Flutter: package:vector_math Frustum.intersectsWithAabb3
func (f Frustum) IntersectsBox(b Box3) bool {
	if b.IsEmpty() {
		return false
	}
	for _, pl := range f.Planes {
		// The corner of the box that is the furthest along the normal: if it is behind the plane, all of the box is.
		p := b.Min
		if pl.Normal.X >= 0 {
			p.X = b.Max.X
		}
		if pl.Normal.Y >= 0 {
			p.Y = b.Max.Y
		}
		if pl.Normal.Z >= 0 {
			p.Z = b.Max.Z
		}
		if pl.DistanceTo(p) < 0 {
			return false
		}
	}
	return true
}
