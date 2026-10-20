package ui

import (
	"math"
	"testing"
)

// buildFuzzPath makes a path from fuzz input: each byte of ops picks a segment,
// and the numbers fill its coordinates in order.
func buildFuzzPath(ops []byte, nums []float64) *Path {
	p := NewPath()
	n := 0
	next := func() float64 {
		if len(nums) == 0 {
			return 0
		}
		v := nums[n%len(nums)]
		n++
		return v
	}
	for _, op := range ops {
		switch op % 9 {
		case 0:
			p.MoveTo(next(), next())
		case 1:
			p.LineTo(next(), next())
		case 2:
			p.QuadraticBezierTo(next(), next(), next(), next())
		case 3:
			p.CubicTo(next(), next(), next(), next(), next(), next())
		case 4:
			p.ConicTo(next(), next(), next(), next(), math.Abs(next()))
		case 5:
			p.Close()
		case 6:
			p.AddArc(Rect{next(), next(), next(), next()}, next(), next())
		case 7:
			p.ArcToPoint(Offset{next(), next()}, ArcToPointOpts{Radius: Radius{next(), next()}, Rotation: next(), LargeArc: op&16 != 0})
		case 8:
			p.AddRRect(NewRRectFromLTRBXY(next(), next(), next(), next(), next(), next()))
		}
	}
	return p
}

func finiteInputs(nums []float64) bool {
	for _, v := range nums {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e6 {
			return false
		}
	}
	return true
}

// FuzzPathFlatten checks that flattening a path made of finite numbers gives
// finite points, a bounded number of them, and never panics; and that Contains,
// the bounds and the metrics are fine with the same path.
func FuzzPathFlatten(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5}, 1.0, 2.0, 30.0, 40.0, 5.0, 6.0, 0.5, 100.0)
	f.Add([]byte{0, 6, 7, 8}, 0.0, 0.0, 10.0, 10.0, 3.0, 4.0, 1.0, 20.0)
	f.Add([]byte{3, 3, 3, 5, 1}, -100.0, 50.0, 1e5, -1e5, 0.0, 0.0, 0.0, 1e-3)
	f.Add([]byte{2, 4, 4, 0}, 1e-9, 1e-9, 1e-9, 1e-9, 1e-9, 1e-9, 1e-9, 1e-9)
	f.Fuzz(func(t *testing.T, ops []byte, a, b, c, d, e, g, h, tol float64) {
		nums := []float64{a, b, c, d, e, g, h}
		if !finiteInputs(nums) || len(ops) > 64 {
			t.Skip()
		}
		p := buildFuzzPath(ops, nums)
		contours := p.Flatten(nil, math.Abs(tol))
		total := 0
		for _, ct := range contours {
			total += len(ct.Points)
			for _, pt := range ct.Points {
				if math.IsNaN(pt.Dx) || math.IsNaN(pt.Dy) || math.IsInf(pt.Dx, 0) || math.IsInf(pt.Dy, 0) {
					t.Fatalf("non-finite point %v in %v", pt, ct)
				}
			}
		}
		if limit := (len(ops) + 1) * (maxFlattenSplits + 2) * 8; total > limit {
			t.Fatalf("%d points from %d segments", total, len(ops))
		}
		p.Contains(Offset{a, b})
		_ = p.GetBounds()
		_ = p.GetTightBounds()
		for _, m := range p.ComputeMetrics() {
			if math.IsNaN(m.Length) || m.Length < 0 {
				t.Fatalf("bad length %v", m.Length)
			}
			m.GetTangentForOffset(m.Length / 2)
			m.ExtractPath(m.Length/4, m.Length*3/4)
		}
	})
}

// FuzzPathCombine checks that combining two paths made of finite numbers never panics,
// gives finite points inside the box of the points of the inputs, and a bounded number
// of verbs, whatever the operation and the fill types.
func FuzzPathCombine(f *testing.F) {
	f.Add([]byte{0, 1, 1, 5}, []byte{0, 1, 1, 5}, byte(0), 0.0, 0.0, 10.0, 0.0, 10.0, 10.0, 5.0, 5.0)
	f.Add([]byte{0, 2, 5}, []byte{0, 3, 5}, byte(1), 0.0, 0.0, 50.0, 100.0, 100.0, 0.0, 20.0, 80.0)
	f.Add([]byte{6, 6}, []byte{8, 4, 5}, byte(2), 0.0, 0.0, 100.0, 100.0, 1.0, 5.0, 60.0, 30.0)
	f.Add([]byte{0, 1, 1, 1, 1, 5}, []byte{0, 1, 1, 1, 1, 5}, byte(3), 0.0, 0.0, 100.0, 0.0, 0.0, 100.0, 100.0, 100.0)
	f.Add([]byte{0, 1, 5}, []byte{0, 1, 5}, byte(4), 1e-7, 1e-7, 1e-7, 2e-7, 3e-7, 1e-7, 2e-7, 3e-7)
	f.Fuzz(func(t *testing.T, opsA, opsB []byte, mode byte, n0, n1, n2, n3, n4, n5, n6, n7 float64) {
		nums := []float64{n0, n1, n2, n3, n4, n5, n6, n7}
		if !finiteInputs(nums) || len(opsA) > 24 || len(opsB) > 24 {
			t.Skip()
		}
		a := buildFuzzPath(opsA, nums)
		b := buildFuzzPath(opsB, nums[len(nums)/2:])
		if mode&16 != 0 {
			a.SetFillType(PathFillTypeEvenOdd)
		}
		if mode&32 != 0 {
			b.SetFillType(PathFillTypeEvenOdd)
		}
		op := PathOperation(1 + int(mode)%5)
		res := PathCombine(op, a, b)
		if len(res.verbs) > 1<<20 {
			t.Fatalf("%d verbs", len(res.verbs))
		}
		box, ok := combineBounds(a, b)
		if !ok {
			if !res.IsEmpty() {
				t.Fatal("a result from nothing")
			}
			return
		}
		slack := 1e-6 * math.Max(box.Right-box.Left, box.Bottom-box.Top)
		for _, pt := range res.points {
			if !pt.IsFinite() || pt.Dx < box.Left-slack || pt.Dx > box.Right+slack || pt.Dy < box.Top-slack || pt.Dy > box.Bottom+slack {
				t.Fatalf("the point %v is outside %v", pt, box)
			}
		}
		for _, w := range res.weights {
			if !(w >= 0) || math.IsInf(w, 0) {
				t.Fatalf("a conic weight of %v", w)
			}
		}
		// The result is a path that the rest of the package can use.
		res.GetBounds()
		res.Contains(Offset{box.Left, box.Top})
		res.Flatten(nil, 0)
	})
}
