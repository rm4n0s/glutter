// Package poly cuts the polygons of a model file into triangles (a fan when the polygon is convex, ear clipping when it is not); the
// loaders of OBJ and PLY share it.
package poly

import "math"

// maxEarClipping is the number of corners above which a concave polygon is cut as a fan instead of by ear clipping, which takes
// time that grows with the cube of it in the worst case.
const maxEarClipping = 2048

// Triangulator cuts polygons into triangles. It keeps its work arrays between polygons.
type Triangulator struct {
	u, v   []float64 // the corners in the plane of the polygon
	next   []int     // the corner after each corner, in the ring that is left
	prev   []int
	remain int
}

// Cut appends the triangles of the polygon to out, as numbers of its corners: a triangle is three consecutive numbers, in the
// winding of the polygon. pos are the positions of all the vertices of the file, three numbers each, and corners the vertices of
// the polygon in them. A convex polygon is cut as a fan from its first corner; a concave one by ear clipping on the plane that is
// the most face-on to the polygon. A polygon with no area is cut as a fan too, and the triangles that have none are the business of
// the renderer.
func (tr *Triangulator) Cut(pos []float64, corners []int32, out []int32) []int32 {
	n := len(corners)
	if n == 3 {
		return append(out, 0, 1, 2)
	}
	at := func(i int) (x, y, z float64) {
		p := int(corners[i]) * 3
		return pos[p], pos[p+1], pos[p+2]
	}
	// The normal of the polygon, by the method of Newell: the sum of the cross products of the neighbours.
	var nx, ny, nz float64
	for i := 0; i < n; i++ {
		x0, y0, z0 := at(i)
		x1, y1, z1 := at((i + 1) % n)
		nx += (y0 - y1) * (z0 + z1)
		ny += (z0 - z1) * (x0 + x1)
		nz += (x0 - x1) * (y0 + y1)
	}
	if !(nx*nx+ny*ny+nz*nz > 0) {
		return fan(out, n)
	}
	// Convex: every turn is to the same side as the normal.
	convex := true
	for i := 0; i < n && convex; i++ {
		x0, y0, z0 := at((i + n - 1) % n)
		x1, y1, z1 := at(i)
		x2, y2, z2 := at((i + 1) % n)
		ax, ay, az := x1-x0, y1-y0, z1-z0
		bx, by, bz := x2-x1, y2-y1, z2-z1
		cx, cy, cz := ay*bz-az*by, az*bx-ax*bz, ax*by-ay*bx
		if cx*nx+cy*ny+cz*nz < 0 {
			convex = false
		}
	}
	if convex || n > maxEarClipping {
		return fan(out, n)
	}

	// Ear clipping, on the plane of the two axes that are not the largest of the normal. The sign keeps the winding: a turn is to
	// the left, in (u, v), when it is positive times orient.
	axis := 2
	var orient float64
	switch ax, ay, az := math.Abs(nx), math.Abs(ny), math.Abs(nz); {
	case ax >= ay && ax >= az:
		axis = 0
	case ay >= az:
		axis = 1
	}
	switch axis {
	case 0:
		orient = math.Copysign(1, nx)
	case 1:
		orient = math.Copysign(1, ny)
	default:
		orient = math.Copysign(1, nz)
	}
	tr.u, tr.v = tr.u[:0], tr.v[:0]
	for i := 0; i < n; i++ {
		x, y, z := at(i)
		switch axis {
		case 0:
			tr.u, tr.v = append(tr.u, y), append(tr.v, z)
		case 1:
			tr.u, tr.v = append(tr.u, z), append(tr.v, x)
		default:
			tr.u, tr.v = append(tr.u, x), append(tr.v, y)
		}
	}
	tr.next, tr.prev = tr.next[:0], tr.prev[:0]
	for i := 0; i < n; i++ {
		tr.next, tr.prev = append(tr.next, (i+1)%n), append(tr.prev, (i+n-1)%n)
	}
	tr.remain = n
	i := 0
	for tr.remain > 3 {
		found := false
		for tries := 0; tries < tr.remain; tries++ {
			if tr.isEar(i, orient) {
				found = true
				break
			}
			i = tr.next[i]
		}
		if !found {
			// A polygon that crosses itself has no ear: the rest is cut as a fan from the corner where we are.
			break
		}
		p, q := tr.prev[i], tr.next[i]
		out = append(out, int32(p), int32(i), int32(q))
		tr.next[p], tr.prev[q] = q, p
		tr.remain--
		i = q
	}
	// What is left: a triangle, or a ring that has no ear.
	first := i
	for k, c := tr.next[first], 0; c < tr.remain-2; k, c = tr.next[k], c+1 {
		out = append(out, int32(first), int32(k), int32(tr.next[k]))
	}
	return out
}

func fan(out []int32, n int) []int32 {
	for i := 1; i+1 < n; i++ {
		out = append(out, 0, int32(i), int32(i+1))
	}
	return out
}

// isEar says whether the corner i makes a triangle with its neighbours that turns the right way and has no other corner of the
// ring inside it.
func (tr *Triangulator) isEar(i int, orient float64) bool {
	p, q := tr.prev[i], tr.next[i]
	ax, ay := tr.u[p], tr.v[p]
	bx, by := tr.u[i], tr.v[i]
	cx, cy := tr.u[q], tr.v[q]
	if ((bx-ax)*(cy-by)-(by-ay)*(cx-bx))*orient <= 0 {
		return false
	}
	for k := tr.next[q]; k != p; k = tr.next[k] {
		px, py := tr.u[k], tr.v[k]
		// Inside, or on the border: the triangle then covers part of what is left of the polygon.
		d1 := ((bx-ax)*(py-ay) - (by-ay)*(px-ax)) * orient
		d2 := ((cx-bx)*(py-by) - (cy-by)*(px-bx)) * orient
		d3 := ((ax-cx)*(py-cy) - (ay-cy)*(px-cx)) * orient
		if d1 >= 0 && d2 >= 0 && d3 >= 0 {
			// A corner at the same place as one of the three does not stop the ear: it is the same point visited twice.
			if (px == ax && py == ay) || (px == bx && py == by) || (px == cx && py == cy) {
				continue
			}
			return false
		}
	}
	return true
}
