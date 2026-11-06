package viewport3d

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/vecmath"
)

// Primitive is what a mesh's vertices make: triangles, lines or points.
//
// Flutter: no counterpart
type Primitive int

const (
	// PrimitiveUnspecified is the zero value: triangles.
	PrimitiveUnspecified Primitive = iota
	// PrimitiveTriangles is a list of triangles: three vertices (or three indices) for each. A triangle's front face is
	// the one whose vertices run counterclockwise.
	PrimitiveTriangles
	// PrimitiveLines is a list of line segments: two vertices (or two indices) for each. A polyline is a list of the
	// segments between its points, and a dash pattern runs on along segments that share an end.
	PrimitiveLines
	// PrimitivePoints is a list of points: one vertex for each.
	PrimitivePoints
)

// String names the primitive.
//
// Flutter: no counterpart
func (p Primitive) String() string {
	switch p {
	case PrimitiveLines:
		return "lines"
	case PrimitivePoints:
		return "points"
	}
	return "triangles"
}

// MeshData is what NewMesh makes a mesh of. The mesh keeps the slices: do not change them afterwards.
//
// Flutter: no counterpart
type MeshData struct {
	// Primitive is what the vertices make; the default is triangles.
	Primitive Primitive
	// Origin is the mesh's origin in float64: the positions are float32 numbers relative to it (SPEC §27.8, D30). A part
	// that is made 50 km from the world's origin keeps its precision by having its origin there and small positions.
	Origin vecmath.Vec3
	// Positions are x, y and z for each vertex, relative to Origin.
	Positions []float32
	// Normals are x, y and z for each vertex, of length 1; optional. Without them a lit material shades each face flat.
	Normals []float32
	// TexCoords are u and v for each vertex; optional.
	TexCoords []float32
	// Colors are red, green, blue and alpha (not premultiplied) for each vertex; optional. A material that uses vertex
	// colors multiplies its color by them.
	Colors []byte
	// Indices16 or Indices32 index the vertices; at most one of them is set. Without indices the vertices are taken in order.
	Indices16 []uint16
	Indices32 []uint32
}

// Mesh is geometry that is uploaded to the GPU once and shared by its items (SPEC §27.8). Its CPU data stays, so that the
// mesh can be uploaded again after a device is lost and its edges can be made. A mesh is used on the UI thread.
//
// Flutter: no counterpart
type Mesh struct {
	data   MeshData
	bounds vecmath.Box3
	local  vecmath.Box3

	// indexed is whether the primitives are named by indices (an edge mesh may have none), not by the vertices in order.
	indexed bool

	// renderers are the renderers that hold GPU resources of the mesh, which Release lets go of, and states what they hold.
	renderers []*Renderer
	states    []*meshGPU

	featureEdges *Mesh
	featureAngle float64
	allEdges     *Mesh
	// closed is whether the mesh is a solid (Closed).
	closed int8
}

// NewMesh checks the data and makes a mesh of it. It fails with MeshInvalid when the lengths do not agree (positions that
// are not a multiple of 3, normals, texture coordinates or colors that are not one for each vertex, both kinds of indices,
// an index outside the vertices, a count of vertices or indices that the primitive does not divide by) or the data holds
// a number that is not finite.
//
// Flutter: no counterpart
func NewMesh(data MeshData) (*Mesh, error) {
	m := &Mesh{data: data, indexed: len(data.Indices16) != 0 || len(data.Indices32) != 0}
	if data.Primitive == PrimitiveUnspecified {
		m.data.Primitive = PrimitiveTriangles
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	m.computeBounds()
	return m, nil
}

// validate checks the data of a new mesh.
func (m *Mesh) validate() error {
	d := &m.data
	n := len(d.Positions) / 3
	prim := d.Primitive.String()
	switch {
	case len(d.Positions)%3 != 0:
		return errors.New(TagMeshInvalid, "the positions are not x, y and z for each vertex", "primitive", prim, "length", len(d.Positions))
	case n == 0:
		return errors.New(TagMeshInvalid, "a mesh has at least one vertex", "primitive", prim)
	case len(d.Normals) != 0 && len(d.Normals) != 3*n:
		return errors.New(TagMeshInvalid, "the normals are not one for each vertex", "primitive", prim, "normals", len(d.Normals), "vertices", n)
	case len(d.TexCoords) != 0 && len(d.TexCoords) != 2*n:
		return errors.New(TagMeshInvalid, "the texture coordinates are not one for each vertex", "primitive", prim, "texCoords", len(d.TexCoords), "vertices", n)
	case len(d.Colors) != 0 && len(d.Colors) != 4*n:
		return errors.New(TagMeshInvalid, "the colors are not one for each vertex", "primitive", prim, "colors", len(d.Colors), "vertices", n)
	case len(d.Indices16) != 0 && len(d.Indices32) != 0:
		return errors.New(TagMeshInvalid, "a mesh has indices of 16 bits or of 32 bits, not both", "primitive", prim)
	case !d.Origin.IsFinite():
		return errors.New(TagMeshInvalid, "the origin is not finite", "primitive", prim)
	}
	for _, v := range d.Positions {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return errors.New(TagMeshInvalid, "a position is not finite", "primitive", prim)
		}
	}
	count := n
	switch {
	case len(d.Indices16) != 0:
		count = len(d.Indices16)
	case len(d.Indices32) != 0:
		count = len(d.Indices32)
	}
	per := 0
	switch d.Primitive {
	case PrimitiveTriangles:
		per = 3
	case PrimitiveLines:
		per = 2
	case PrimitivePoints:
		per = 1
	default:
		return errors.New(TagMeshInvalid, "the primitive is not known", "primitive", int(d.Primitive))
	}
	if count%per != 0 {
		return errors.New(TagMeshInvalid, "the number of vertices or indices is not a multiple of what the primitive takes", "primitive", prim, "count", count, "per", per)
	}
	for _, i := range d.Indices16 {
		if int(i) >= n {
			return errors.New(TagMeshInvalid, "an index is outside the vertices", "primitive", prim, "index", int(i), "vertices", n)
		}
	}
	for _, i := range d.Indices32 {
		if int64(i) >= int64(n) {
			return errors.New(TagMeshInvalid, "an index is outside the vertices", "primitive", prim, "index", int(i), "vertices", n)
		}
	}
	return nil
}

// computeBounds makes the box of the vertices that the primitives use.
func (m *Mesh) computeBounds() {
	box := vecmath.EmptyBox3()
	add := func(i int) {
		box = box.Expand(vecmath.Vec3{X: float64(m.data.Positions[3*i]), Y: float64(m.data.Positions[3*i+1]), Z: float64(m.data.Positions[3*i+2])})
	}
	switch {
	case !m.indexed:
		for i := 0; i < len(m.data.Positions)/3; i++ {
			add(i)
		}
	case m.data.Indices16 != nil:
		for _, i := range m.data.Indices16 {
			add(int(i))
		}
	default:
		for _, i := range m.data.Indices32 {
			add(int(i))
		}
	}
	m.local = box
	m.bounds = vecmath.Box3{Min: box.Min.Add(m.data.Origin), Max: box.Max.Add(m.data.Origin)}
}

// Data is the data the mesh was made of, with the slices that it keeps: do not change them.
//
// Flutter: no counterpart
func (m *Mesh) Data() MeshData { return m.data }

// Primitive is what the mesh's vertices make.
//
// Flutter: no counterpart
func (m *Mesh) Primitive() Primitive { return m.data.Primitive }

// Origin is the mesh's origin: the positions are relative to it.
//
// Flutter: no counterpart
func (m *Mesh) Origin() vecmath.Vec3 { return m.data.Origin }

// Bounds is the box of the vertices that are used, in float64 and with the origin added: in the mesh's own coordinates (an
// item's transform turns them into the world's).
//
// Flutter: no counterpart
func (m *Mesh) Bounds() vecmath.Box3 { return m.bounds }

// VertexCount is the number of vertices.
//
// Flutter: no counterpart
func (m *Mesh) VertexCount() int { return len(m.data.Positions) / 3 }

// IndexCount is the number of indices, or of vertices when there are none: what a draw of the whole mesh takes.
//
// Flutter: no counterpart
func (m *Mesh) IndexCount() int {
	switch {
	case !m.indexed:
		return m.VertexCount()
	case m.data.Indices16 != nil:
		return len(m.data.Indices16)
	}
	return len(m.data.Indices32)
}

// PrimitiveCount is the number of triangles, segments or points.
//
// Flutter: no counterpart
func (m *Mesh) PrimitiveCount() int {
	switch m.data.Primitive {
	case PrimitiveLines:
		return m.IndexCount() / 2
	case PrimitivePoints:
		return m.IndexCount()
	}
	return m.IndexCount() / 3
}

// index returns the vertex that the i-th index (or vertex, without indices) stands for.
func (m *Mesh) index(i int) int {
	switch {
	case !m.indexed:
		return i
	case m.data.Indices16 != nil:
		return int(m.data.Indices16[i])
	}
	return int(m.data.Indices32[i])
}

// position returns the position of a vertex, relative to the origin, in float64.
func (m *Mesh) position(v int) vecmath.Vec3 {
	p := m.data.Positions
	return vecmath.Vec3{X: float64(p[3*v]), Y: float64(p[3*v+1]), Z: float64(p[3*v+2])}
}

// Release frees what the mesh holds on the GPU, in every renderer that drew it (and what its edges hold). The mesh itself stays
// usable: a renderer uploads it again if an item still draws it. Call it when the app is done with the mesh, as with a
// gpu.Buffer.
//
// Flutter: no counterpart
func (m *Mesh) Release() {
	for len(m.renderers) > 0 {
		m.renderers[0].forgetMesh(m)
	}
	if m.featureEdges != nil {
		m.featureEdges.Release()
	}
	if m.allEdges != nil {
		m.allEdges.Release()
	}
}

// TagMeshInvalid is the tag of the error of NewMesh and FeatureEdges for data that cannot make a mesh.
//
// Flutter: no counterpart
const TagMeshInvalid = "MeshInvalid"
