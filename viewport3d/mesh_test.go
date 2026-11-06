package viewport3d_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func TestNewMeshChecksItsData(t *testing.T) {
	tri := []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}
	nan := float32(math.NaN())
	tests := []struct {
		name string
		data viewport3d.MeshData
	}{
		{"positions that are not triples", viewport3d.MeshData{Positions: []float32{0, 0, 0, 1}}},
		{"no vertices", viewport3d.MeshData{}},
		{"too few normals", viewport3d.MeshData{Positions: tri, Normals: []float32{0, 0, 1}}},
		{"too few texture coordinates", viewport3d.MeshData{Positions: tri, TexCoords: []float32{0, 0}}},
		{"too few colors", viewport3d.MeshData{Positions: tri, Colors: []byte{255, 255, 255, 255}}},
		{"both kinds of indices", viewport3d.MeshData{Positions: tri, Indices16: []uint16{0, 1, 2}, Indices32: []uint32{0, 1, 2}}},
		{"an origin that is not finite", viewport3d.MeshData{Positions: tri, Origin: vecmath.Vec3{X: math.Inf(1)}}},
		{"a position that is not finite", viewport3d.MeshData{Positions: []float32{nan, 0, 0, 1, 0, 0, 0, 1, 0}}},
		{"two vertices of a triangle", viewport3d.MeshData{Positions: tri[:6]}},
		{"an odd number of line indices", viewport3d.MeshData{Primitive: viewport3d.PrimitiveLines, Positions: tri, Indices16: []uint16{0, 1, 2}}},
		{"an index of 16 bits outside the vertices", viewport3d.MeshData{Positions: tri, Indices16: []uint16{0, 1, 3}}},
		{"an index of 32 bits outside the vertices", viewport3d.MeshData{Positions: tri, Indices32: []uint32{0, 1, 1 << 31}}},
		{"a primitive that is not known", viewport3d.MeshData{Primitive: viewport3d.Primitive(9), Positions: tri}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := viewport3d.NewMesh(tt.data)
			if m != nil {
				t.Errorf("got a mesh with an error: %v", err)
			}
			wantTag(t, err, viewport3d.TagMeshInvalid, "NewMesh->Mesh.validate.MeshInvalid")
		})
	}
}

func TestAMeshKnowsItsSize(t *testing.T) {
	tests := []struct {
		name                    string
		mesh                    *viewport3d.Mesh
		vertices, indices, prim int
		min, max                vecmath.Vec3
	}{
		{"a box", scenetest.Box(2, 4, 6), 24, 36, 12, vecmath.Vec3{X: -1, Y: -2, Z: -3}, vecmath.Vec3{X: 1, Y: 2, Z: 3}},
		{"a box of shared vertices", scenetest.BoxNoNormals(2, 2, 2), 8, 36, 12, vecmath.Vec3{X: -1, Y: -1, Z: -1}, vecmath.Vec3{X: 1, Y: 1, Z: 1}},
		{"segments", scenetest.Segments([3]float32{0, 0, 0}, [3]float32{1, 2, 3}, [3]float32{4, 4, 4}, [3]float32{5, 5, 5}), 4, 4, 2, vecmath.Vec3{}, vecmath.Vec3{X: 5, Y: 5, Z: 5}},
		{"a polyline", scenetest.Polyline([3]float32{0, 0, 0}, [3]float32{1, 0, 0}, [3]float32{1, 1, 0}), 3, 4, 2, vecmath.Vec3{}, vecmath.Vec3{X: 1, Y: 1}},
		{"points", scenetest.Points([3]float32{-1, 0, 0}, [3]float32{1, 0, 0}), 2, 2, 2, vecmath.Vec3{X: -1}, vecmath.Vec3{X: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.mesh
			if m.VertexCount() != tt.vertices || m.IndexCount() != tt.indices || m.PrimitiveCount() != tt.prim {
				t.Errorf("vertices %d, indices %d, primitives %d; want %d, %d, %d", m.VertexCount(), m.IndexCount(), m.PrimitiveCount(), tt.vertices, tt.indices, tt.prim)
			}
			if b := m.Bounds(); b.Min != tt.min || b.Max != tt.max {
				t.Errorf("bounds = %v, want %v to %v", b, tt.min, tt.max)
			}
		})
	}
}

// edgesOf is the number of feature edges of a mesh at the angle.
func edgesOf(t *testing.T, m *viewport3d.Mesh, angle float64) int {
	t.Helper()
	e, err := viewport3d.FeatureEdges(m, angle)
	if err != nil {
		t.Fatal(err)
	}
	if e.Primitive() != viewport3d.PrimitiveLines {
		t.Fatalf("the edges are %v", e.Primitive())
	}
	return e.PrimitiveCount()
}

func TestFeatureEdges(t *testing.T) {
	deg := math.Pi / 180
	fan := func() *viewport3d.Mesh {
		// Three triangles around the edge from (0,0,0) to (1,0,0), each with its own vertices.
		d := viewport3d.MeshData{}
		for _, apex := range [][3]float32{{0.5, 1, 0}, {0.5, -1, 0}, {0.5, 0, 1}} {
			d.Positions = append(d.Positions, 0, 0, 0, 1, 0, 0, apex[0], apex[1], apex[2])
		}
		m, err := viewport3d.NewMesh(d)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	tests := []struct {
		name  string
		mesh  *viewport3d.Mesh
		angle float64
		want  int
	}{
		{"a box with a normal for each face", scenetest.Box(1, 1, 1), 30 * deg, 12},
		{"a box of 8 shared vertices", scenetest.BoxNoNormals(1, 1, 1), 30 * deg, 12},
		{"a box, seen over its right angles", scenetest.Box(1, 1, 1), 100 * deg, 0},
		{"a box, just under its right angles", scenetest.Box(1, 1, 1), 89 * deg, 12},
		{"a cylinder of 36 sides at 30 degrees: its two rims", scenetest.Cylinder(1, 2, 36), 30 * deg, 72},
		{"a cylinder at 5 degrees: the rims and every seam of its side", scenetest.Cylinder(1, 2, 36), 5 * deg, 108},
		{"a cylinder at 95 degrees: nothing", scenetest.Cylinder(1, 2, 36), 95 * deg, 0},
		{"three triangles on one edge: that edge and 6 boundary edges", fan(), 30 * deg, 1 + 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := edgesOf(t, tt.mesh, tt.angle); got != tt.want {
				t.Errorf("%d edges, want %d", got, tt.want)
			}
		})
	}
	t.Run("a lone triangle", func(t *testing.T) {
		m, err := viewport3d.NewMesh(viewport3d.MeshData{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}})
		if err != nil {
			t.Fatal(err)
		}
		if got := edgesOf(t, m, 30*deg); got != 3 {
			t.Errorf("%d edges, want its 3 boundary edges", got)
		}
	})
}

func TestFeatureEdgesAreMadeOnceForEachAngle(t *testing.T) {
	m := scenetest.Cylinder(1, 2, 16)
	a, _ := viewport3d.FeatureEdges(m, 0.5)
	b, _ := viewport3d.FeatureEdges(m, 0.5)
	if a != b {
		t.Errorf("the same angle made two meshes")
	}
	c, _ := viewport3d.FeatureEdges(m, 0.1)
	if c == a || c.PrimitiveCount() <= a.PrimitiveCount() {
		t.Errorf("a smaller angle gave %d edges, the larger %d", c.PrimitiveCount(), a.PrimitiveCount())
	}
	// The edges use the positions of the mesh and nothing else.
	if &a.Data().Positions[0] != &m.Data().Positions[0] || a.Origin() != m.Origin() {
		t.Errorf("the edges copied the positions")
	}
}

func TestFeatureEdgesOfAMeshThatIsNotTriangles(t *testing.T) {
	for _, m := range []*viewport3d.Mesh{scenetest.Segments([3]float32{0, 0, 0}, [3]float32{1, 1, 1}), scenetest.Points([3]float32{0, 0, 0})} {
		e, err := viewport3d.FeatureEdges(m, 0.5)
		if e != nil {
			t.Errorf("%v: got edges", m.Primitive())
		}
		wantTag(t, err, viewport3d.TagMeshInvalid, "FeatureEdges.MeshInvalid")
	}
}

func TestNegativeZeroIsTheSamePosition(t *testing.T) {
	negZero := float32(math.Copysign(0, -1))
	// Two triangles that meet on the edge from (0,0,0) to (1,0,0), which the second one gives with -0: they are coplanar, so the
	// edge is not a feature edge; if -0 were another position it would be a boundary of each.
	m, err := viewport3d.NewMesh(viewport3d.MeshData{Positions: []float32{
		0, 0, 0, 1, 0, 0, 0, 1, 0,
		negZero, negZero, negZero, 1, -1, negZero, 1, 0, 0,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := edgesOf(t, m, 30*math.Pi/180); got != 4 {
		t.Errorf("%d edges, want the 4 boundary edges of the two triangles", got)
	}
}
