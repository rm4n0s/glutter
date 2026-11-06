package main

import (
	"math"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

// builtInPart makes the part that the example shows before a file is opened, with plain code: a plate with a hollow boss and
// two posts, in millimetres, z up. Each piece is a node with a mesh and a material of its own, so that a pick says which one it hit.
func builtInPart() *model.Model {
	m := &model.Model{Name: "Bracket", Units: model.Units{Name: "mm", MetersPerUnit: 0.001}}
	piece := func(name string, mesh *model.Mesh, color [4]float32) {
		mat := &model.Material{Name: name, BaseColor: color, Specular: [3]float32{0.5, 0.5, 0.5}, Shininess: 48}
		mesh.Name, mesh.Material = name, mat
		node := model.NewNode(name)
		node.Meshes = []*model.Mesh{mesh}
		m.Nodes = append(m.Nodes, node)
		m.Meshes = append(m.Meshes, mesh)
		m.Materials = append(m.Materials, mat)
	}
	piece("plate", box(vecmath.Vec3{X: -50, Y: -30}, vecmath.Vec3{X: 50, Y: 30, Z: 10}), [4]float32{0.62, 0.68, 0.76, 1})
	piece("boss", tube(vecmath.Vec3{X: -10, Z: 10}, 10, 18, 45, 48), [4]float32{0.85, 0.55, 0.25, 1})
	piece("post 1", cylinder(vecmath.Vec3{X: 30, Y: 18, Z: 10}, 6, 24, 24), [4]float32{0.35, 0.65, 0.45, 1})
	piece("post 2", cylinder(vecmath.Vec3{X: 30, Y: -18, Z: 10}, 6, 24, 24), [4]float32{0.35, 0.65, 0.45, 1})
	return m
}

// meshBuilder collects vertices with normals and triangles.
type meshBuilder struct {
	pos, nrm []float32
	idx      []uint32
}

func (b *meshBuilder) vertex(p, n vecmath.Vec3) uint32 {
	b.pos = append(b.pos, float32(p.X), float32(p.Y), float32(p.Z))
	b.nrm = append(b.nrm, float32(n.X), float32(n.Y), float32(n.Z))
	return uint32(len(b.pos)/3 - 1)
}

func (b *meshBuilder) tri(a, c, d uint32) { b.idx = append(b.idx, a, c, d) }

func (b *meshBuilder) mesh() *model.Mesh {
	return &model.Mesh{Positions: b.pos, Normals: b.nrm, Indices: b.idx}
}

// box makes the box between two corners, with a normal on each of its 24 vertices.
func box(lo, hi vecmath.Vec3) *model.Mesh {
	var b meshBuilder
	center, half := lo.Add(hi).Scale(0.5), hi.Sub(lo).Scale(0.5)
	axes := [3]vecmath.Vec3{vecmath.Vec3X, vecmath.Vec3Y, vecmath.Vec3Z}
	h := [3]float64{half.X, half.Y, half.Z}
	for a := 0; a < 3; a++ {
		for _, s := range []float64{1, -1} {
			n := axes[a].Scale(s)
			u, v := (a+1)%3, (a+2)%3
			var c [4]uint32
			for k, uv := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				p := center.Add(n.Scale(h[a])).Add(axes[u].Scale(uv[0] * h[u])).Add(axes[v].Scale(uv[1] * h[v]))
				c[k] = b.vertex(p, n)
			}
			if s > 0 {
				b.tri(c[0], c[1], c[2])
				b.tri(c[0], c[2], c[3])
			} else {
				b.tri(c[0], c[2], c[1])
				b.tri(c[0], c[3], c[2])
			}
		}
	}
	return b.mesh()
}

// ringPoint is the point at the angle i of n on the circle of radius r around the axis through c, at height z.
func ringPoint(c vecmath.Vec3, r, z float64, i, n int) (p, radial vecmath.Vec3) {
	a := 2 * math.Pi * float64(i%n) / float64(n) // the last point is the first one, to the bit
	radial = vecmath.Vec3{X: math.Cos(a), Y: math.Sin(a)}
	return vecmath.Vec3{X: c.X + r*radial.X, Y: c.Y + r*radial.Y, Z: z}, radial
}

// wall adds a cylindrical wall of radius r between two heights; outward says which way the surface faces.
func (b *meshBuilder) wall(c vecmath.Vec3, r, z0, z1 float64, n int, outward bool) {
	first := uint32(len(b.pos) / 3)
	for i := 0; i <= n; i++ {
		p0, radial := ringPoint(c, r, z0, i, n)
		p1, _ := ringPoint(c, r, z1, i, n)
		normal := radial
		if !outward {
			normal = radial.Neg()
		}
		b.vertex(p0, normal)
		b.vertex(p1, normal)
	}
	for i := 0; i < n; i++ {
		b0, t0, b1, t1 := first+uint32(2*i), first+uint32(2*i+1), first+uint32(2*i+2), first+uint32(2*i+3)
		if outward {
			b.tri(b0, b1, t1)
			b.tri(b0, t1, t0)
		} else {
			b.tri(b0, t1, b1)
			b.tri(b0, t0, t1)
		}
	}
}

// disk adds a flat ring between two radii (the inner radius 0 makes a disk) at the height z, facing up or down.
func (b *meshBuilder) disk(c vecmath.Vec3, rIn, rOut, z float64, n int, up bool) {
	normal := vecmath.Vec3{Z: 1}
	if !up {
		normal = normal.Neg()
	}
	first := uint32(len(b.pos) / 3)
	for i := 0; i < n; i++ {
		po, _ := ringPoint(c, rOut, z, i, n)
		pi, _ := ringPoint(c, rIn, z, i, n)
		b.vertex(po, normal)
		b.vertex(pi, normal)
	}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		o0, i0, o1, i1 := first+uint32(2*i), first+uint32(2*i+1), first+uint32(2*j), first+uint32(2*j+1)
		// A disk (the inner radius 0) has one triangle for each step, a ring two.
		switch {
		case up && rIn == 0:
			b.tri(o0, o1, i0)
		case up:
			b.tri(o0, o1, i1)
			b.tri(o0, i1, i0)
		case rIn == 0:
			b.tri(o0, i0, o1)
		default:
			b.tri(o0, i1, o1)
			b.tri(o0, i0, i1)
		}
	}
}

// cylinder makes a solid cylinder standing on the point c, with its two ends.
func cylinder(c vecmath.Vec3, r, h float64, n int) *model.Mesh {
	var b meshBuilder
	b.wall(c, r, c.Z, c.Z+h, n, true)
	b.disk(c, 0, r, c.Z+h, n, true)
	b.disk(c, 0, r, c.Z, n, false)
	return b.mesh()
}

// tube makes a hollow cylinder standing on the point c: its two walls and its two ends.
func tube(c vecmath.Vec3, rIn, rOut, h float64, n int) *model.Mesh {
	var b meshBuilder
	b.wall(c, rOut, c.Z, c.Z+h, n, true)
	b.wall(c, rIn, c.Z, c.Z+h, n, false)
	b.disk(c, rIn, rOut, c.Z+h, n, true)
	b.disk(c, rIn, rOut, c.Z, n, false)
	return b.mesh()
}
