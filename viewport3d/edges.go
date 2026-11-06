package viewport3d

import (
	"cmp"
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/vecmath"
)

// edgeEntry is one side of a triangle: the undirected edge between two welded positions, the triangle, and the vertices
// that the triangle's indices name (the vertices of the first triangle that has the edge are the ones the edge mesh uses).
type edgeEntry struct {
	key    uint64
	face   int32
	v0, v1 int32
}

// weldedPositions gives each vertex the number of its position: vertices whose three float32 coordinates are equal (-0 equals 0) share
// it, so that the triangles of a mesh whose faces do not share vertices (as an STL file's) still meet along their edges.
func weldedPositions(m *Mesh) []int32 {
	n := m.VertexCount()
	ids := make([]int32, n)
	seen := make(map[[3]uint32]int32, n)
	p := m.data.Positions
	// bits is the pattern of a coordinate, where -0 is 0: they are the same position.
	bits := func(f float32) uint32 {
		if f == 0 {
			return 0
		}
		return math.Float32bits(f)
	}
	for v := 0; v < n; v++ {
		k := [3]uint32{bits(p[3*v]), bits(p[3*v+1]), bits(p[3*v+2])}
		id, ok := seen[k]
		if !ok {
			id = int32(len(seen))
			seen[k] = id
		}
		ids[v] = id
	}
	return ids
}

// triangleEdges lists the edges of the triangles of a mesh, sorted by edge. Degenerate triangles (two equal positions, or
// no area) are left out. normals has the unit normal of each triangle, and a zero vector for one that is left out.
func triangleEdges(m *Mesh) (entries []edgeEntry, normals []vecmath.Vec3) {
	ids := weldedPositions(m)
	tris := m.IndexCount() / 3
	entries = make([]edgeEntry, 0, 3*tris)
	normals = make([]vecmath.Vec3, tris)
	for f := 0; f < tris; f++ {
		v := [3]int{m.index(3 * f), m.index(3*f + 1), m.index(3*f + 2)}
		p := [3]int32{ids[v[0]], ids[v[1]], ids[v[2]]}
		if p[0] == p[1] || p[1] == p[2] || p[0] == p[2] {
			continue
		}
		a, b, c := m.position(v[0]), m.position(v[1]), m.position(v[2])
		n := b.Sub(a).Cross(c.Sub(a)).Normalize()
		if n.LengthSq() == 0 {
			continue
		}
		normals[f] = n
		for e := 0; e < 3; e++ {
			i, j := e, (e+1)%3
			lo, hi := p[i], p[j]
			if lo > hi {
				lo, hi = hi, lo
			}
			entries = append(entries, edgeEntry{key: uint64(lo)<<32 | uint64(hi), face: int32(f), v0: int32(v[i]), v1: int32(v[j])})
		}
	}
	slices.SortFunc(entries, func(a, b edgeEntry) int {
		if c := cmp.Compare(a.key, b.key); c != 0 {
			return c
		}
		return cmp.Compare(a.face, b.face)
	})
	return entries, normals
}

// edgeMesh makes a mesh of line segments that share the positions of m, from pairs of its vertices.
func edgeMesh(m *Mesh, pairs []uint32) *Mesh {
	if pairs == nil {
		pairs = []uint32{}
	}
	e := &Mesh{data: MeshData{Primitive: PrimitiveLines, Origin: m.data.Origin, Positions: m.data.Positions, Indices32: pairs}, indexed: true}
	e.computeBounds()
	return e
}

// FeatureEdges makes the edges that show the shape of a triangle mesh: the boundary edges (one triangle), the edges that
// more than two triangles share, and the edges where the angle between the normals of the two triangles is more than
// angle, in radians. Vertices at the same position count as one, so a mesh whose faces do not share vertices has its
// edges where they meet. The result is a mesh of line segments that uses the positions of m, so it costs the indices only;
// it is made once for each angle and kept in m (the next call with the same angle gives the same mesh), and Release of m
// releases it. It fails with MeshInvalid when m is not made of triangles.
//
// Flutter: no counterpart
func FeatureEdges(m *Mesh, angle float64) (*Mesh, error) {
	if m.data.Primitive != PrimitiveTriangles {
		return nil, errors.New(TagMeshInvalid, "feature edges are made of a mesh of triangles", "primitive", m.data.Primitive.String())
	}
	if m.featureEdges != nil && m.featureAngle == angle {
		return m.featureEdges, nil
	}
	entries, normals := triangleEdges(m)
	cosAngle := math.Cos(angle)
	var pairs []uint32
	for i := 0; i < len(entries); {
		j := i + 1
		for j < len(entries) && entries[j].key == entries[i].key {
			j++
		}
		switch count := j - i; {
		case count == 1, count > 2:
			pairs = append(pairs, uint32(entries[i].v0), uint32(entries[i].v1))
		default:
			if normals[entries[i].face].Dot(normals[entries[i+1].face]) < cosAngle {
				pairs = append(pairs, uint32(entries[i].v0), uint32(entries[i].v1))
			}
		}
		i = j
	}
	m.featureEdges, m.featureAngle = edgeMesh(m, pairs), angle
	return m.featureEdges, nil
}

// allEdges makes the mesh of every edge of a triangle mesh, once, for the wireframe display mode.
func allEdges(m *Mesh) *Mesh {
	if m.allEdges != nil {
		return m.allEdges
	}
	entries, _ := triangleEdges(m)
	var pairs []uint32
	for i := 0; i < len(entries); i++ {
		if i == 0 || entries[i].key != entries[i-1].key {
			pairs = append(pairs, uint32(entries[i].v0), uint32(entries[i].v1))
		}
	}
	m.allEdges = edgeMesh(m, pairs)
	return m.allEdges
}

// segmentFloats is the number of float32 numbers of a line segment as the GPU reads it: two positions, the distance
// along the polyline at the first of them, and the number of the segment.
const segmentFloats = 8

// segments lists the segments of a line mesh as the GPU reads them: p0, p1 and the length of the polyline up to p0, in the
// units of the mesh. A segment that starts where the previous one ended continues its length, so that a dash pattern
// runs on along a polyline; another starts at 0. The positions are float32, relative to the origin.
func (m *Mesh) segments(dst []float32) []float32 {
	n := m.IndexCount() / 2
	dst = slices.Grow(dst[:0], n*segmentFloats)
	prevEnd := -1
	var dist float32
	for s := 0; s < n; s++ {
		a, b := m.index(2*s), m.index(2*s+1)
		if a != prevEnd {
			dist = 0
		}
		pa, pb := m.data.Positions[3*a:3*a+3], m.data.Positions[3*b:3*b+3]
		dst = append(dst, pa[0], pa[1], pa[2], pb[0], pb[1], pb[2], dist, float32(s))
		dx, dy, dz := float64(pb[0]-pa[0]), float64(pb[1]-pa[1]), float64(pb[2]-pa[2])
		dist += float32(math.Sqrt(dx*dx + dy*dy + dz*dz))
		prevEnd = b
	}
	return dst
}
