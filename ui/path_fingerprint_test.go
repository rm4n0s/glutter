package ui

import (
	"math"
	"sync"
	"testing"
)

// TestPathFingerprintIsKeptUpToDate checks that the hash a path keeps while it is built is the
// hash of its contents, for every operation that changes it (path_fingerprint.go).
func TestPathFingerprintIsKeptUpToDate(t *testing.T) {
	src := NewPath()
	src.MoveTo(1, 2)
	src.ConicTo(5, 0, 5, 5, 0.5)
	src.CubicTo(1, 2, 3, 4, 5, 6)
	src.Close()
	tests := []struct {
		name string
		make func() *Path
	}{
		{"empty", NewPath},
		{"a point that is not finite", func() *Path {
			p := NewPath()
			p.MoveTo(1, 1)
			p.LineTo(math.Inf(1), 2)
			p.LineTo(3, 3)
			return p
		}},
		{"lines", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.LineTo(1, 1)
			p.RelativeLineTo(2, 3)
			p.Close()
			return p
		}},
		{"curves", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.QuadraticBezierTo(1, 1, 2, 0)
			p.RelativeCubicTo(1, 1, 2, 2, 3, 0)
			p.ConicTo(1, 1, 2, 0, 0.3)
			p.RelativeConicTo(1, 1, 2, 0, 2)
			return p
		}},
		{"shapes", func() *Path {
			p := NewPath()
			p.AddRect(NewRectFromLTWH(1, 2, 3, 4))
			p.AddOval(NewRectFromLTWH(0, 0, 9, 5))
			p.AddRRect(NewRRectFromRectXY(NewRectFromLTWH(0, 0, 20, 10), 3, 3))
			return p
		}},
		{"arcs", func() *Path {
			p := NewPath()
			p.MoveTo(0, 0)
			p.ArcToPoint(Offset{10, 10}, ArcToPointOpts{Radius: Radius{X: 8, Y: 8}})
			p.ArcTo(NewRectFromLTWH(0, 0, 8, 8), 0.2, 2, false)
			p.AddArc(NewRectFromLTWH(0, 0, 8, 8), 0, 3)
			return p
		}},
		{"added with an offset", func() *Path { p := NewPath(); p.MoveTo(9, 9); p.AddPath(src, Offset{3, 4}); return p }},
		{"added with a matrix", func() *Path {
			p := NewPath()
			p.AddPath(src, Offset{}, AddPathOpts{Matrix4: NewMatrix4Diagonal3Values(2, 3, 1)})
			return p
		}},
		{"extended", func() *Path { p := NewPath(); p.MoveTo(9, 9); p.ExtendWithPath(src, Offset{3, 4}); return p }},
		{"copied", func() *Path { return NewPathFrom(src) }},
		{"shifted", func() *Path { return src.Shift(Offset{3, 4}) }},
		{"transformed", func() *Path { return src.Transform(NewMatrix4RotationZ(0.5)) }},
		{"transformed with perspective", func() *Path {
			m := NewMatrix4Identity()
			m.SetEntry(3, 0, 0.001)
			return src.Transform(m)
		}},
		{"reset", func() *Path { p := NewPathFrom(src); p.Reset(); p.MoveTo(1, 1); return p }},
		{"combined", func() *Path { return PathCombine(PathOperationUnion, rectPath(0, 0, 10, 10), rectPath(5, 5, 15, 15)) }},
	}
	for _, tt := range tests {
		p := tt.make()
		got := p.derived
		p.rehash()
		if got != p.derived {
			t.Errorf("%s: what the path kept, %+v, is not what its contents give, %+v", tt.name, got, p.derived)
		}
		var acc boundsAcc
		finite := true
		for _, pt := range p.points {
			finite = finite && pt.IsFinite()
			acc.add(pt)
		}
		want := acc.rect()
		if !finite {
			want = Rect{}
		}
		if b := p.GetBounds(); b != want {
			t.Errorf("%s: GetBounds = %v, want %v", tt.name, b, want)
		}
	}
}

// TestPathFingerprintTellsPathsApart checks that paths that differ in one way or another do not
// share a fingerprint: the ways a sum of hashes could be fooled (a mirrored shape, swapped
// points, a move of an element to another index).
func TestPathFingerprintTellsPathsApart(t *testing.T) {
	poly := func(pts ...float64) *Path {
		p := NewPath()
		p.MoveTo(pts[0], pts[1])
		for i := 2; i < len(pts); i += 2 {
			p.LineTo(pts[i], pts[i+1])
		}
		return p
	}
	tests := []struct {
		name string
		a, b *Path
	}{
		{"mirrored in x", poly(1, 2, 3, 4, 5, 6), poly(-1, 2, -3, 4, -5, 6)},
		{"two signs flipped", poly(1, 2, 3, 4), poly(-1, 2, 3, -4)},
		{"mirrored in y", poly(1, 2, 3, 4, 5, 6), poly(1, -2, 3, -4, 5, -6)},
		{"x and y swapped", poly(1, 2, 3, 4), poly(2, 1, 4, 3)},
		{"points in another order", poly(1, 2, 3, 4, 5, 6), poly(5, 6, 3, 4, 1, 2)},
		{"two points swapped between positions", poly(1, 2, 3, 4, 5, 6, 7, 8), poly(1, 2, 5, 6, 3, 4, 7, 8)},
		{"one coordinate nudged", poly(1, 2, 3, 4), poly(1, 2, 3, math.Nextafter(4, 5))},
		{"a longer path", poly(1, 2, 3, 4), poly(1, 2, 3, 4, 3, 4)},
		{"zero and negative zero", poly(0, 0, 1, 1), poly(math.Copysign(0, -1), 0, 1, 1)},
	}
	for _, tt := range tests {
		if tt.a.Fingerprint() == tt.b.Fingerprint() {
			t.Errorf("%s: the paths have the same fingerprint", tt.name)
		}
	}
	// No collisions among many grid-like paths, as icons are.
	seen := map[uint64][2]int{}
	for i := 0; i < 200; i++ {
		for j := 0; j < 200; j++ {
			p := poly(float64(i), float64(j), float64(j), float64(i), 0, 0)
			f := p.Fingerprint()
			if prev, ok := seen[f]; ok {
				t.Fatalf("(%d,%d) and %v have the same fingerprint", i, j, prev)
			}
			seen[f] = [2]int{i, j}
		}
	}
}

// TestPathFingerprintIsNotWrittenByReaders reads the fingerprint of one path from two goroutines;
// run with -race (SPEC §9.9).
func TestPathFingerprintIsNotWrittenByReaders(t *testing.T) {
	p := NewPath()
	p.AddOval(NewRectFromLTWH(0, 0, 10, 10))
	want := p.Fingerprint()
	var wg sync.WaitGroup
	for g := 0; g < 2; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				if p.Fingerprint() != want {
					t.Error("the fingerprint changed")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkPathFingerprint(b *testing.B) {
	p := NewPath()
	p.MoveTo(0, 0)
	for i := 0; i < 500; i++ {
		p.CubicTo(float64(i), 1, 2, float64(i), 3, 4)
	}
	b.ReportAllocs()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink += p.Fingerprint()
	}
	_ = sink
}
