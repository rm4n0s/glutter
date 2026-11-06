package obj

import (
	"math"
	"math/rand"
	"testing"
)

// polygonArea is the area and the winding of a polygon in the plane z = 0 (positive when counter-clockwise).
func polygonArea(pos []float64, corners []int32) float64 {
	a := 0.0
	for i := range corners {
		p, q := int(corners[i])*3, int(corners[(i+1)%len(corners)])*3
		a += pos[p]*pos[q+1] - pos[q]*pos[p+1]
	}
	return a / 2
}

func TestTriangulateCoversThePolygon(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	var tr triangulator
	tests := []struct {
		name  string
		shape func(n int) []float64 // the positions of the corners, in order
		sizes []int
	}{
		{"convex, counter-clockwise", func(n int) []float64 { return circle(n, func(int) float64 { return 1 }, 1) }, []int{4, 5, 8, 30}},
		{"convex, clockwise", func(n int) []float64 { return circle(n, func(int) float64 { return 1 }, -1) }, []int{4, 5, 8, 30}},
		{"a star, counter-clockwise", func(n int) []float64 {
			return circle(n, func(i int) float64 { return 1 - 0.6*float64(i%2) }, 1)
		}, []int{6, 8, 10, 40}},
		{"a star, clockwise", func(n int) []float64 {
			return circle(n, func(i int) float64 { return 1 - 0.6*float64(i%2) }, -1)
		}, []int{6, 8, 10, 40}},
		{"a random star", func(n int) []float64 {
			r := make([]float64, n)
			for i := range r {
				r[i] = 0.2 + rng.Float64()
			}
			return circle(n, func(i int) float64 { return r[i] }, 1)
		}, []int{5, 9, 17, 100}},
		{"a comb", comb, []int{4, 6, 10}},
	}
	for _, tt := range tests {
		for _, n := range tt.sizes {
			t.Run(tt.name, func(t *testing.T) {
				pos := tt.shape(n)
				count := len(pos) / 3
				corners := make([]int32, count)
				for i := range corners {
					corners[i] = int32(i)
				}
				want := polygonArea(pos, corners)
				tris := tr.cut(pos, corners, nil)
				if len(tris) != 3*(count-2) {
					t.Fatalf("%d corners gave %d triangles, want %d", count, len(tris)/3, count-2)
				}
				sum := 0.0
				for i := 0; i < len(tris); i += 3 {
					a := polygonArea(pos, []int32{corners[tris[i]], corners[tris[i+1]], corners[tris[i+2]]})
					if a*want < 0 {
						t.Errorf("triangle %d turns the other way than the polygon", i/3)
					}
					sum += a
				}
				if math.Abs(sum-want) > 1e-9*math.Max(1, math.Abs(want)) {
					t.Errorf("the triangles cover %v, the polygon is %v", sum, want)
				}
			})
		}
	}
}

// circle makes the corners at the angles of a regular polygon with radius r(i), counter-clockwise or clockwise (dir -1).
func circle(n int, r func(int) float64, dir float64) []float64 {
	pos := make([]float64, 0, 3*n)
	for i := 0; i < n; i++ {
		a := dir * 2 * math.Pi * float64(i) / float64(n)
		pos = append(pos, r(i)*math.Cos(a), r(i)*math.Sin(a), 0)
	}
	return pos
}

// comb is a comb with n teeth, as a polygon: along the bottom, then over each tooth.
func comb(n int) []float64 {
	teeth := n / 2
	var pos []float64
	pos = append(pos, 0, 0, 0, float64(2*teeth), 0, 0)
	for k := teeth - 1; k >= 0; k-- {
		x := float64(2 * k)
		pos = append(pos, x+2, 3, 0)
		if k > 0 {
			pos = append(pos, x+1, 3, 0, x+1, 1, 0)
		} else {
			pos = append(pos, x+1, 3, 0, x+1, 1, 0)
		}
	}
	return pos[:3*(len(pos)/3)]
}

func TestTriangulateInAnyPlane(t *testing.T) {
	// The L-shaped polygon of testdata/concave.obj, turned about two axes and moved: the triangles still cover its area.
	l := [][3]float64{{0, 0, 0}, {2, 0, 0}, {2, 1, 0}, {1, 1, 0}, {1, 2, 0}, {0, 2, 0}}
	var tr triangulator
	for _, angles := range [][2]float64{{0, 0}, {0.7, 0}, {0, 1.2}, {2.5, 0.4}, {math.Pi / 2, math.Pi / 2}} {
		var pos []float64
		for _, p := range l {
			// Turn about x, then about y.
			y, z := p[1]*math.Cos(angles[0])-p[2]*math.Sin(angles[0]), p[1]*math.Sin(angles[0])+p[2]*math.Cos(angles[0])
			x, z2 := p[0]*math.Cos(angles[1])+z*math.Sin(angles[1]), -p[0]*math.Sin(angles[1])+z*math.Cos(angles[1])
			pos = append(pos, x+3, y-2, z2+5)
		}
		corners := []int32{0, 1, 2, 3, 4, 5}
		tris := tr.cut(pos, corners, nil)
		if len(tris) != 12 {
			t.Fatalf("angles %v: %d triangles", angles, len(tris)/3)
		}
		sum := 0.0
		for i := 0; i < len(tris); i += 3 {
			a, b, c := int(tris[i])*3, int(tris[i+1])*3, int(tris[i+2])*3
			u := [3]float64{pos[b] - pos[a], pos[b+1] - pos[a+1], pos[b+2] - pos[a+2]}
			v := [3]float64{pos[c] - pos[a], pos[c+1] - pos[a+1], pos[c+2] - pos[a+2]}
			cross := [3]float64{u[1]*v[2] - u[2]*v[1], u[2]*v[0] - u[0]*v[2], u[0]*v[1] - u[1]*v[0]}
			sum += math.Sqrt(cross[0]*cross[0]+cross[1]*cross[1]+cross[2]*cross[2]) / 2
		}
		if math.Abs(sum-3) > 1e-9 {
			t.Errorf("angles %v: the triangles cover %v, want 3", angles, sum)
		}
	}
}

func TestTriangulateDegenerateAndHugePolygons(t *testing.T) {
	var tr triangulator
	// No area: a fan, with as many triangles as any polygon.
	line := []float64{0, 0, 0, 1, 0, 0, 2, 0, 0, 3, 0, 0, 4, 0, 0}
	if got := len(tr.cut(line, []int32{0, 1, 2, 3, 4}, nil)); got != 9 {
		t.Errorf("a polygon on a line gave %d numbers, want 9", got)
	}
	// A polygon that crosses itself (a bow tie) still gives n-2 triangles.
	bow := []float64{0, 0, 0, 2, 2, 0, 2, 0, 0, 0, 2, 0}
	if got := len(tr.cut(bow, []int32{0, 1, 2, 3}, nil)); got != 6 {
		t.Errorf("a bow tie gave %d numbers, want 6", got)
	}
	// Beyond the size for ear clipping it is a fan, and it is quick.
	pos := circle(maxEarClipping+10, func(i int) float64 { return 1 - 0.5*float64(i%2) }, 1)
	corners := make([]int32, len(pos)/3)
	for i := range corners {
		corners[i] = int32(i)
	}
	if got := len(tr.cut(pos, corners, nil)); got != 3*(len(corners)-2) {
		t.Errorf("a huge polygon gave %d numbers", got)
	}
}
