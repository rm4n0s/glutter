package viewport3d_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func TestSectionPlanesAreMadeUnitPlanesAndTellTheListeners(t *testing.T) {
	tests := []struct {
		name   string
		planes []vecmath.Plane
		want   []vecmath.Plane
	}{
		{"none", nil, []vecmath.Plane{}},
		{"a unit plane stays", []vecmath.Plane{{Normal: vecmath.Vec3X, D: -0.5}}, []vecmath.Plane{{Normal: vecmath.Vec3X, D: -0.5}}},
		{"a long normal is made a unit", []vecmath.Plane{{Normal: vecmath.Vec3{Y: 4}, D: 2}}, []vecmath.Plane{{Normal: vecmath.Vec3Y, D: 0.5}}},
		{"six planes", []vecmath.Plane{{Normal: vecmath.Vec3X}, {Normal: vecmath.Vec3Y}, {Normal: vecmath.Vec3Z}, {Normal: vecmath.Vec3X.Neg()}, {Normal: vecmath.Vec3Y.Neg()}, {Normal: vecmath.Vec3Z.Neg()}},
			[]vecmath.Plane{{Normal: vecmath.Vec3X}, {Normal: vecmath.Vec3Y}, {Normal: vecmath.Vec3Z}, {Normal: vecmath.Vec3X.Neg()}, {Normal: vecmath.Vec3Y.Neg()}, {Normal: vecmath.Vec3Z.Neg()}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := viewport3d.NewScene()
			c := listen(s)
			s.SetSectionPlanes(tt.planes...)
			if c.n != 1 {
				t.Errorf("%d notifications, want 1", c.n)
			}
			got := s.SectionPlanes()
			if len(got) != len(tt.want) {
				t.Fatalf("%d planes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Normal.Sub(tt.want[i].Normal).Length() > 1e-12 || math.Abs(got[i].D-tt.want[i].D) > 1e-12 {
					t.Errorf("plane %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTooManySectionPlanesAndPlanesWithoutANormalPanicInDebugBuilds(t *testing.T) {
	skipUnlessDebug(t)
	seven := make([]vecmath.Plane, viewport3d.MaxSectionPlanes+1)
	for i := range seven {
		seven[i] = vecmath.Plane{Normal: vecmath.Vec3X}
	}
	tests := []struct {
		name   string
		planes []vecmath.Plane
	}{
		{"seven planes", seven},
		{"a normal without length", []vecmath.Plane{{Normal: vecmath.Vec3X}, {}}},
		{"a normal that is not a number", []vecmath.Plane{{Normal: vecmath.Vec3{X: math.NaN()}}}},
		{"a distance that is infinite", []vecmath.Plane{{Normal: vecmath.Vec3X, D: math.Inf(1)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := viewport3d.NewScene()
			err := panicOf(func() { s.SetSectionPlanes(tt.planes...) })
			wantTag(t, err, viewport3d.TagSectionInvalid, "Scene.SetSectionPlanes.SectionInvalid")
		})
	}
}

func TestAMeshIsClosedWhenEveryEdgeBelongsToTwoTriangles(t *testing.T) {
	openBox := scenetest.Box(1, 1, 1).Data()
	openBox.Indices16 = openBox.Indices16[:len(openBox.Indices16)-6]
	open, err := viewport3d.NewMesh(openBox)
	if err != nil {
		t.Fatal(err)
	}
	lone, err := viewport3d.NewMesh(viewport3d.MeshData{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		mesh *viewport3d.Mesh
		want bool
	}{
		{"a box", scenetest.Box(1, 1, 1), true},
		{"a box with no normals (the vertices of a corner are shared)", scenetest.BoxNoNormals(1, 1, 1), true},
		{"a cylinder", scenetest.Cylinder(1, 2, 12), true},
		{"a sphere", scenetest.Sphere(1, 12, 8), true},
		{"a torus", scenetest.Torus(2, 0.5, 16, 8), true},
		{"a box without a face", open, false},
		{"a lone triangle", lone, false},
		{"lines", scenetest.Polyline([3]float32{0, 0, 0}, [3]float32{1, 0, 0}, [3]float32{1, 1, 0}), false},
		{"points", scenetest.Points([3]float32{0, 0, 0}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mesh.Closed(); got != tt.want {
				t.Errorf("Closed() = %v, want %v", got, tt.want)
			}
			if got := tt.mesh.Closed(); got != tt.want {
				t.Errorf("Closed() the second time = %v, want %v", got, tt.want)
			}
		})
	}
}

// triangles returns the triangles of a mesh of triangles in its own coordinates.
func triangles(t *testing.T, m *viewport3d.Mesh) [][3]vecmath.Vec3 {
	t.Helper()
	d := m.Data()
	var idx []int
	switch {
	case d.Indices32 != nil:
		for _, i := range d.Indices32 {
			idx = append(idx, int(i))
		}
	case d.Indices16 != nil:
		for _, i := range d.Indices16 {
			idx = append(idx, int(i))
		}
	default:
		for i := 0; i < m.VertexCount(); i++ {
			idx = append(idx, i)
		}
	}
	pos := func(i int) vecmath.Vec3 {
		return vecmath.Vec3{X: float64(d.Positions[3*i]), Y: float64(d.Positions[3*i+1]), Z: float64(d.Positions[3*i+2])}
	}
	var out [][3]vecmath.Vec3
	for i := 0; i+2 < len(idx); i += 3 {
		out = append(out, [3]vecmath.Vec3{pos(idx[i]), pos(idx[i+1]), pos(idx[i+2])})
	}
	return out
}

// stencilCount is what the count pass of the caps leaves in the stencil for the ray of a pixel (SPEC §27.8, "Section planes"): the
// triangles the ray meets on the side of the plane that is kept add one when they face away from the ray and take one away when they
// face it.
func stencilCount(tris [][3]vecmath.Vec3, ray vecmath.Ray, plane vecmath.Plane) int {
	n := 0
	for _, tr := range tris {
		t, ok := ray.IntersectTriangle(tr[0], tr[1], tr[2])
		if !ok || plane.DistanceTo(ray.At(t)) > 0 {
			continue
		}
		if tr[1].Sub(tr[0]).Cross(tr[2].Sub(tr[0])).Dot(ray.Dir) > 0 {
			n++
		} else {
			n--
		}
	}
	return n
}

// inside says whether a point is inside a closed mesh: a ray from it meets an odd number of its triangles.
func inside(tris [][3]vecmath.Vec3, p vecmath.Vec3) bool {
	ray := vecmath.Ray{Origin: p, Dir: vecmath.Vec3{X: 0.371, Y: 0.529, Z: 0.762}.Normalize()}
	n := 0
	for _, tr := range tris {
		if _, ok := ray.IntersectTriangle(tr[0], tr[1], tr[2]); ok {
			n++
		}
	}
	return n%2 == 1
}

func TestTheCapIsDrawnExactlyWhereTheStencilCountIsNotZero(t *testing.T) {
	tests := []struct {
		name  string
		mesh  *viewport3d.Mesh
		plane vecmath.Plane
	}{
		{"a cube cut through the middle", scenetest.Box(2, 2, 2), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.3}, vecmath.Vec3X)},
		{"a cube cut slantwise", scenetest.Box(2, 2, 2), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{Y: 0.2}, vecmath.Vec3{X: 1, Y: 1, Z: 0.5}.Normalize())},
		{"a cube cut at a corner", scenetest.Box(2, 2, 2), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.9, Y: 0.9, Z: 0.9}, vecmath.Vec3{X: 1, Y: 1, Z: 1}.Normalize())},
		{"a torus through its hole", scenetest.Torus(2, 0.6, 48, 16), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{}, vecmath.Vec3Z)},
		{"a torus across its tube", scenetest.Torus(2, 0.6, 48, 16), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.4}, vecmath.Vec3X)},
		{"a torus slantwise", scenetest.Torus(2, 0.6, 48, 16), vecmath.NewPlaneFromPointNormal(vecmath.Vec3{Y: 0.1}, vecmath.Vec3{X: 0.3, Y: 1, Z: 0.2}.Normalize())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tris := triangles(t, tt.mesh)
			var capped, checked int
			for ix := -30; ix <= 30; ix++ {
				for iy := -30; iy <= 30; iy++ {
					// A ray from the eye 12 units away, through a grid of points around the part, turned a little so that no ray runs
					// along an edge.
					ray := vecmath.Ray{Origin: vecmath.Vec3{X: 1.7, Y: 2.3, Z: 12}, Dir: vecmath.Vec3{X: float64(ix)*0.0931 - 1.7 + 0.013, Y: float64(iy)*0.0937 - 2.3 + 0.017, Z: -12}.Normalize()}
					at, ok := ray.IntersectPlane(tt.plane)
					if !ok {
						continue
					}
					want := inside(tris, ray.At(at))
					got := stencilCount(tris, ray, tt.plane) != 0
					checked++
					if want {
						capped++
					}
					if got != want {
						t.Fatalf("ray %d,%d: the cap is drawn %v, the plane's point is inside the solid %v", ix, iy, got, want)
					}
				}
			}
			if capped == 0 || capped == checked {
				t.Errorf("%d of %d rays have a cap: the test checks nothing", capped, checked)
			}
		})
	}
}

func TestTheTorusIsWoundOutward(t *testing.T) {
	tris := triangles(t, scenetest.Torus(2, 0.6, 24, 12))
	for i, tr := range tris {
		c := tr[0].Add(tr[1]).Add(tr[2]).Scale(1.0 / 3)
		// The nearest point of the ring's circle, and from it the outward direction of the tube.
		r := math.Hypot(c.X, c.Z)
		ring := vecmath.Vec3{X: c.X / r * 2, Z: c.Z / r * 2}
		out := c.Sub(ring)
		n := tr[1].Sub(tr[0]).Cross(tr[2].Sub(tr[0]))
		if n.Dot(out) <= 0 {
			t.Fatalf("triangle %d faces inward", i)
		}
	}
}
