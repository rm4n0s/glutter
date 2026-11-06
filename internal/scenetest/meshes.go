// Package scenetest has the scenes that the display tests of viewport3d draw on each backend (SPEC §27.12): meshes made by plain
// code, the scenes, and a function that draws one into a texture and reads the pixels back. Tests import it; no program does.
//
// Flutter: no counterpart
package scenetest

import (
	"math"

	"github.com/rm4n0s/glutter/viewport3d"
)

// mustMesh makes a mesh of data that the generators below made right.
func mustMesh(d viewport3d.MeshData) *viewport3d.Mesh {
	m, err := viewport3d.NewMesh(d)
	if err != nil {
		panic(err)
	}
	return m
}

// Box makes a box of the sizes centered on the origin, with a normal and texture coordinates for each of its 24 vertices.
func Box(w, h, d float32) *viewport3d.Mesh {
	hx, hy, hz := w/2, h/2, d/2
	type face struct {
		n    [3]float32
		u, v [3]float32
	}
	faces := []face{
		{[3]float32{1, 0, 0}, [3]float32{0, 0, -1}, [3]float32{0, 1, 0}},
		{[3]float32{-1, 0, 0}, [3]float32{0, 0, 1}, [3]float32{0, 1, 0}},
		{[3]float32{0, 1, 0}, [3]float32{1, 0, 0}, [3]float32{0, 0, -1}},
		{[3]float32{0, -1, 0}, [3]float32{1, 0, 0}, [3]float32{0, 0, 1}},
		{[3]float32{0, 0, 1}, [3]float32{1, 0, 0}, [3]float32{0, 1, 0}},
		{[3]float32{0, 0, -1}, [3]float32{-1, 0, 0}, [3]float32{0, 1, 0}},
	}
	half := [3]float32{hx, hy, hz}
	var data viewport3d.MeshData
	for fi, f := range faces {
		base := uint16(4 * fi)
		for _, c := range [][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			for i := 0; i < 3; i++ {
				data.Positions = append(data.Positions, half[i]*(f.n[i]+c[0]*f.u[i]+c[1]*f.v[i]))
			}
			data.Normals = append(data.Normals, f.n[:]...)
			data.TexCoords = append(data.TexCoords, (c[0]+1)/2, (1-c[1])/2)
		}
		data.Indices16 = append(data.Indices16, base, base+1, base+2, base, base+2, base+3)
	}
	return mustMesh(data)
}

// Sphere makes a sphere of the radius around the origin, with the poles on y, a normal and texture coordinates for each vertex.
func Sphere(r float32, segments, rings int) *viewport3d.Mesh {
	var data viewport3d.MeshData
	for i := 0; i <= rings; i++ {
		phi := math.Pi * float64(i) / float64(rings)
		sinPhi, cosPhi := math.Sin(phi), math.Cos(phi)
		if i == 0 || i == rings { // the poles are exact points
			sinPhi, cosPhi = 0, 1-2*float64(i/rings)
		}
		for j := 0; j <= segments; j++ {
			// The last column has the angles of the first, so that the vertices of the seam are at the same positions.
			theta := 2 * math.Pi * float64(j%segments) / float64(segments)
			n := [3]float32{float32(sinPhi * math.Cos(theta)), float32(cosPhi), float32(sinPhi * math.Sin(theta))}
			data.Positions = append(data.Positions, r*n[0], r*n[1], r*n[2])
			data.Normals = append(data.Normals, n[:]...)
			data.TexCoords = append(data.TexCoords, float32(j)/float32(segments), float32(i)/float32(rings))
		}
	}
	stride := segments + 1
	for i := 0; i < rings; i++ {
		for j := 0; j < segments; j++ {
			a, b, c, d := uint32(i*stride+j), uint32((i+1)*stride+j), uint32((i+1)*stride+j+1), uint32(i*stride+j+1)
			data.Indices32 = append(data.Indices32, a, d, b, d, c, b)
		}
	}
	return mustMesh(data)
}

// Cylinder makes a cylinder along y, of the radius and height, centered on the origin, with its caps. Each side and each cap has its
// own vertices, so that the edge between them is sharp.
func Cylinder(r, h float32, segments int) *viewport3d.Mesh {
	var data viewport3d.MeshData
	add := func(p, n [3]float32) uint32 {
		data.Positions = append(data.Positions, p[:]...)
		data.Normals = append(data.Normals, n[:]...)
		return uint32(len(data.Positions)/3 - 1)
	}
	for j := 0; j < segments; j++ {
		// The last side ends at the angle of the first, exactly, so that the seam welds.
		a0, a1 := 2*math.Pi*float64(j)/float64(segments), 2*math.Pi*float64((j+1)%segments)/float64(segments)
		c0, s0, c1, s1 := float32(math.Cos(a0)), float32(math.Sin(a0)), float32(math.Cos(a1)), float32(math.Sin(a1))
		n0, n1 := [3]float32{c0, 0, s0}, [3]float32{c1, 0, s1}
		b0 := add([3]float32{r * c0, -h / 2, r * s0}, n0)
		b1 := add([3]float32{r * c1, -h / 2, r * s1}, n1)
		t1 := add([3]float32{r * c1, h / 2, r * s1}, n1)
		t0 := add([3]float32{r * c0, h / 2, r * s0}, n0)
		data.Indices32 = append(data.Indices32, b0, t1, b1, b0, t0, t1)
		// The caps.
		up, down := [3]float32{0, 1, 0}, [3]float32{0, -1, 0}
		tc := add([3]float32{0, h / 2, 0}, up)
		tp0 := add([3]float32{r * c0, h / 2, r * s0}, up)
		tp1 := add([3]float32{r * c1, h / 2, r * s1}, up)
		data.Indices32 = append(data.Indices32, tc, tp1, tp0)
		bc := add([3]float32{0, -h / 2, 0}, down)
		bp0 := add([3]float32{r * c0, -h / 2, r * s0}, down)
		bp1 := add([3]float32{r * c1, -h / 2, r * s1}, down)
		data.Indices32 = append(data.Indices32, bc, bp0, bp1)
	}
	return mustMesh(data)
}

// BoxNoNormals is a box of 8 shared vertices with no normals, drawn flat; its edges are where it is not flat.
func BoxNoNormals(w, h, d float32) *viewport3d.Mesh {
	hx, hy, hz := w/2, h/2, d/2
	data := viewport3d.MeshData{
		Positions: []float32{-hx, -hy, -hz, hx, -hy, -hz, hx, hy, -hz, -hx, hy, -hz, -hx, -hy, hz, hx, -hy, hz, hx, hy, hz, -hx, hy, hz},
		Indices16: []uint16{
			4, 5, 6, 4, 6, 7, // +z
			1, 0, 3, 1, 3, 2, // -z
			5, 1, 2, 5, 2, 6, // +x
			0, 4, 7, 0, 7, 3, // -x
			7, 6, 2, 7, 2, 3, // +y
			0, 1, 5, 0, 5, 4, // -y
		},
	}
	return mustMesh(data)
}

// Segments makes a mesh of line segments from pairs of points.
func Segments(points ...[3]float32) *viewport3d.Mesh {
	var data viewport3d.MeshData
	data.Primitive = viewport3d.PrimitiveLines
	for _, p := range points {
		data.Positions = append(data.Positions, p[:]...)
	}
	return mustMesh(data)
}

// Polyline makes a mesh of the segments between consecutive points.
func Polyline(points ...[3]float32) *viewport3d.Mesh {
	var data viewport3d.MeshData
	data.Primitive = viewport3d.PrimitiveLines
	for i, p := range points {
		data.Positions = append(data.Positions, p[:]...)
		if i > 0 {
			data.Indices32 = append(data.Indices32, uint32(i-1), uint32(i))
		}
	}
	return mustMesh(data)
}

// Points makes a mesh of points.
func Points(points ...[3]float32) *viewport3d.Mesh {
	var data viewport3d.MeshData
	data.Primitive = viewport3d.PrimitivePoints
	for _, p := range points {
		data.Positions = append(data.Positions, p[:]...)
	}
	return mustMesh(data)
}

// Torus makes a torus around the y axis, of the radius of its ring and of its tube, with a normal for each vertex; the vertices of the
// seams are shared, so that it is a closed mesh.
func Torus(ring, tube float32, segments, rings int) *viewport3d.Mesh {
	var data viewport3d.MeshData
	for i := 0; i < segments; i++ {
		u := 2 * math.Pi * float64(i) / float64(segments)
		for j := 0; j < rings; j++ {
			v := 2 * math.Pi * float64(j) / float64(rings)
			cu, su, cv, sv := math.Cos(u), math.Sin(u), math.Cos(v), math.Sin(v)
			data.Positions = append(data.Positions, float32((float64(ring)+float64(tube)*cv)*cu), float32(float64(tube)*sv), float32((float64(ring)+float64(tube)*cv)*su))
			data.Normals = append(data.Normals, float32(cv*cu), float32(sv), float32(cv*su))
		}
	}
	at := func(i, j int) uint32 { return uint32((i%segments)*rings + j%rings) }
	for i := 0; i < segments; i++ {
		for j := 0; j < rings; j++ {
			a, b, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			data.Indices32 = append(data.Indices32, a, d, b, b, d, c)
		}
	}
	return mustMesh(data)
}
