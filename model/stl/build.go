package stl

import (
	"context"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
)

// soup is the triangles of one solid as the file gave them, before welding.
type soup[T float32 | float64] struct {
	// pos are the nine coordinates of each triangle.
	pos []T
	// norm are the three numbers of the normal that the file stores for each triangle (zero when it has none).
	norm []float32
	// color is the packed red, green, blue and alpha of each triangle (red in the lowest byte), or nil when no triangle has one.
	color []uint32
}

// build welds the vertices of the soup, drops the triangles that have no area, turns the triangles whose winding contradicts their
// stored normal, and gives each vertex the normal of its smoothing group: the triangles at a vertex whose normals differ from the
// first one of the group by less than the crease angle (and that have the same color). The normals are weighted by area.
func build[T float32 | float64](ctx context.Context, s *soup[T], o Options) (*model.Mesh, error) {
	ntri := len(s.pos) / 9
	if ntri == 0 {
		return nil, errors.New(TagSTLInvalid, "the solid has no triangles", "reason", "empty")
	}
	workers := workersFor(ntri)
	var idx []uint32
	var verts []T
	if o.Flat {
		verts = s.pos
		idx = make([]uint32, 3*ntri)
		for c := range idx {
			idx[c] = uint32(c)
		}
	} else {
		idx, verts = weld(ctx, s.pos, o.WeldTolerance, workers)
		if err := ctx.Err(); err != nil {
			return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("phase", "weld")
		}
	}

	// The normal and the area of each triangle, in float64; a triangle with none has the area 0 and is dropped below.
	unit := make([]float32, 3*ntri)
	area := make([]float32, ntri)
	turned := make([]int, workers)
	parallel(ntri, workers, func(worker, lo, hi int) {
		for t := lo; t < hi; t++ {
			if t&(minPerWorker-1) == 0 && ctx.Err() != nil {
				return
			}
			i0, i1, i2 := idx[3*t], idx[3*t+1], idx[3*t+2]
			if i0 == i1 || i1 == i2 || i0 == i2 {
				continue
			}
			ax, ay, az := float64(verts[3*i0]), float64(verts[3*i0+1]), float64(verts[3*i0+2])
			e1x, e1y, e1z := float64(verts[3*i1])-ax, float64(verts[3*i1+1])-ay, float64(verts[3*i1+2])-az
			e2x, e2y, e2z := float64(verts[3*i2])-ax, float64(verts[3*i2+1])-ay, float64(verts[3*i2+2])-az
			nx, ny, nz := e1y*e2z-e1z*e2y, e1z*e2x-e1x*e2z, e1x*e2y-e1y*e2x
			l2 := nx*nx + ny*ny + nz*nz
			if !(l2 > 0) || math.IsInf(l2, 0) {
				continue
			}
			// A stored normal that points the other way than the winding says wins: many writers get the winding wrong, and few
			// get the normal wrong.
			if sx, sy, sz := float64(s.norm[3*t]), float64(s.norm[3*t+1]), float64(s.norm[3*t+2]); sx*sx+sy*sy+sz*sz > 0.25 &&
				nx*sx+ny*sy+nz*sz < 0 {
				idx[3*t+1], idx[3*t+2] = i2, i1
				nx, ny, nz = -nx, -ny, -nz
				turned[worker]++
			}
			l := math.Sqrt(l2)
			unit[3*t], unit[3*t+1], unit[3*t+2] = float32(nx/l), float32(ny/l), float32(nz/l)
			area[t] = max(float32(l), math.SmallestNonzeroFloat32)
		}
	})
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("phase", "normals")
	}
	w := 0
	for t := 0; t < ntri; t++ {
		if area[t] == 0 {
			continue
		}
		if w != t {
			idx[3*w], idx[3*w+1], idx[3*w+2] = idx[3*t], idx[3*t+1], idx[3*t+2]
			unit[3*w], unit[3*w+1], unit[3*w+2] = unit[3*t], unit[3*t+1], unit[3*t+2]
			area[w] = area[t]
			if s.color != nil {
				s.color[w] = s.color[t]
			}
		}
		w++
	}
	if dropped := ntri - w; dropped > 0 {
		o.logger().Warn("stl: triangles without area were dropped", "name", o.Name, "dropped", dropped, "kept", w)
	}
	flipped := 0
	for _, n := range turned {
		flipped += n
	}
	if flipped > 0 {
		o.logger().Debug("stl: triangles were turned to agree with their stored normals", "name", o.Name, "turned", flipped)
	}
	if w == 0 {
		return nil, errors.New(TagSTLInvalid, "every triangle of the solid has no area", "triangles", ntri, "reason", "degenerate")
	}
	colored := s.color != nil
	var outPos []T
	var outNormal []float32
	var outColor []byte
	if o.Flat {
		outPos = make([]T, 0, 9*w)
		outNormal = make([]float32, 0, 9*w)
		if colored {
			outColor = make([]byte, 0, 12*w)
		}
		for t := 0; t < w; t++ {
			for j := 0; j < 3; j++ {
				v := idx[3*t+j]
				outPos = append(outPos, verts[3*v], verts[3*v+1], verts[3*v+2])
				outNormal = append(outNormal, unit[3*t], unit[3*t+1], unit[3*t+2])
				if colored {
					c := s.color[t]
					outColor = append(outColor, byte(c), byte(c>>8), byte(c>>16), byte(c>>24))
				}
				idx[3*t+j] = uint32(3*t + j)
			}
		}
	} else {
		nv := len(verts) / 3
		workers = workersFor(w)
		// The corners of each vertex, by counting sort. Each worker owns a range of vertices and looks at all the corners.
		start := make([]uint32, nv+1)
		corners := idx[:3*w]
		parallel(nv, workers, func(_, lo, hi int) {
			l, h := uint32(lo), uint32(hi)
			for _, v := range corners {
				if v >= l && v < h {
					start[v+1]++
				}
			}
		})
		for v := 0; v < nv; v++ {
			start[v+1] += start[v]
		}
		incident := make([]uint32, 3*w)
		parallel(nv, workers, func(_, lo, hi int) {
			l, h := uint32(lo), uint32(hi)
			for c, v := range corners {
				if v >= l && v < h {
					incident[start[v]] = uint32(c)
					start[v]++
				}
			}
		})
		// start[v] is now the end of the corners of v, which is the start of those of v+1.
		cosCrease := o.cosCrease() - 1e-5
		type group struct {
			rep   [3]float32
			color uint32
			id    uint32
		}
		type part struct {
			pos    []T
			normal []float32
			color  []byte
		}
		parts := make([]part, workers)
		parallel(nv, workers, func(worker, lo, hi int) {
			var groups []group
			p := part{pos: make([]T, 0, 3*(hi-lo)), normal: make([]float32, 0, 3*(hi-lo))}
			if colored {
				p.color = make([]byte, 0, 4*(hi-lo))
			}
			from := uint32(0)
			if lo > 0 {
				from = start[lo-1]
			}
			for v := lo; v < hi; v++ {
				if v&(minPerWorker-1) == 0 && ctx.Err() != nil {
					return
				}
				to := start[v]
				groups = groups[:0]
				for _, c := range incident[from:to] {
					t := int(c / 3)
					ux, uy, uz := unit[3*t], unit[3*t+1], unit[3*t+2]
					var col uint32
					if colored {
						col = s.color[t]
					}
					g := -1
					for gi := range groups {
						r := &groups[gi]
						if colored && r.color != col {
							continue
						}
						if float64(ux*r.rep[0]+uy*r.rep[1]+uz*r.rep[2]) >= cosCrease {
							g = gi
							break
						}
					}
					if g < 0 {
						id := uint32(len(p.pos) / 3)
						p.pos = append(p.pos, verts[3*v], verts[3*v+1], verts[3*v+2])
						p.normal = append(p.normal, 0, 0, 0)
						if colored {
							p.color = append(p.color, byte(col), byte(col>>8), byte(col>>16), byte(col>>24))
						}
						groups = append(groups, group{rep: [3]float32{ux, uy, uz}, color: col, id: id})
						g = len(groups) - 1
					}
					id := groups[g].id
					a := area[t]
					p.normal[3*id] += ux * a
					p.normal[3*id+1] += uy * a
					p.normal[3*id+2] += uz * a
					idx[c] = id
				}
				from = to
			}
			parts[worker] = p
		})
		if err := ctx.Err(); err != nil {
			return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("phase", "smoothing")
		}
		// The parts side by side: the ids of the corners of each part move by what the parts before it have.
		offsets := make([]int, workers+1)
		for k, p := range parts {
			offsets[k+1] = offsets[k] + len(p.pos)/3
		}
		outPos = make([]T, 3*offsets[workers])
		outNormal = make([]float32, 3*offsets[workers])
		if colored {
			outColor = make([]byte, 4*offsets[workers])
		}
		parallel(nv, workers, func(worker, lo, hi int) {
			p, off := parts[worker], offsets[worker]
			copy(outPos[3*off:], p.pos)
			if colored {
				copy(outColor[4*off:], p.color)
			}
			for i := 0; i < len(p.normal); i += 3 {
				x, y, z := float64(p.normal[i]), float64(p.normal[i+1]), float64(p.normal[i+2])
				l := math.Sqrt(x*x + y*y + z*z)
				if l == 0 {
					outNormal[3*off+i+2] = 1
					continue
				}
				outNormal[3*off+i], outNormal[3*off+i+1], outNormal[3*off+i+2] = float32(x/l), float32(y/l), float32(z/l)
			}
			if off != 0 && hi > lo {
				from := uint32(0)
				if lo > 0 {
					from = start[lo-1]
				}
				for _, c := range incident[from:start[hi-1]] {
					idx[c] += uint32(off)
				}
			}
		})
	}
	mesh := &model.Mesh{Primitive: model.PrimitiveTriangles, Normals: outNormal, Colors: outColor, Indices: idx[: 3*w : 3*w]}
	setPositions(mesh, outPos)
	return mesh, nil
}

// setPositions puts the positions in float32, or in float64 when they are float64 and some of them do not fit float32.
func setPositions[T float32 | float64](m *model.Mesh, pos []T) {
	switch p := any(pos).(type) {
	case []float32:
		m.Positions = p
	case []float64:
		for _, v := range p {
			if float64(float32(v)) != v {
				m.Positions64 = p
				return
			}
		}
		m.Positions = make([]float32, len(p))
		for i, v := range p {
			m.Positions[i] = float32(v)
		}
	}
}
