package ui

import (
	"math"
	"testing"
)

// FuzzRectIntersectExpand checks what must hold for any two rectangles without
// NaN: the bounding box contains both, the intersection is inside both, and
// Overlaps agrees with the intersection of non-empty rectangles.
func FuzzRectIntersectExpand(f *testing.F) {
	f.Add(0.0, 0.0, 10.0, 10.0, 5.0, 5.0, 15.0, 15.0)
	f.Add(0.0, 0.0, 10.0, 10.0, 10.0, 0.0, 20.0, 10.0)
	f.Add(0.0, 0.0, 10.0, 10.0, 20.0, 20.0, 30.0, 30.0)
	f.Add(-1e9, -1e9, 1e9, 1e9, 0.0, 0.0, 1.0, 1.0)
	f.Add(5.0, 5.0, 5.0, 5.0, 0.0, 0.0, 10.0, 10.0)
	f.Add(10.0, 10.0, 0.0, 0.0, 0.0, 0.0, 10.0, 10.0)
	f.Fuzz(func(t *testing.T, al, at, ar, ab, bl, bt, br, bb float64) {
		a, b := Rect{al, at, ar, ab}, Rect{bl, bt, br, bb}
		if a.HasNaN() || b.HasNaN() {
			t.Skip()
		}

		u := a.ExpandToInclude(b)
		if u != b.ExpandToInclude(a) {
			t.Errorf("ExpandToInclude is not commutative: %v vs %v", u, b.ExpandToInclude(a))
		}
		if u.Left > a.Left || u.Left > b.Left || u.Top > a.Top || u.Top > b.Top ||
			u.Right < a.Right || u.Right < b.Right || u.Bottom < a.Bottom || u.Bottom < b.Bottom {
			t.Errorf("%v does not bound %v and %v", u, a, b)
		}
		if a.ExpandToInclude(a) != a {
			t.Errorf("ExpandToInclude is not idempotent for %v", a)
		}

		i := a.Intersect(b)
		if i != b.Intersect(a) {
			t.Errorf("Intersect is not commutative: %v vs %v", i, b.Intersect(a))
		}
		if i.Left < a.Left || i.Left < b.Left || i.Top < a.Top || i.Top < b.Top ||
			i.Right > a.Right || i.Right > b.Right || i.Bottom > a.Bottom || i.Bottom > b.Bottom {
			t.Errorf("%v is not inside %v and %v", i, a, b)
		}
		if a.Intersect(a) != a {
			t.Errorf("Intersect is not idempotent for %v", a)
		}
		if a.Overlaps(b) != b.Overlaps(a) {
			t.Errorf("Overlaps is not symmetric for %v and %v", a, b)
		}
		if !a.IsEmpty() && !b.IsEmpty() {
			if a.Overlaps(b) == i.IsEmpty() {
				t.Errorf("Overlaps = %v but the intersection %v of %v and %v has IsEmpty = %v", a.Overlaps(b), i, a, b, i.IsEmpty())
			}
			// Everything in the intersection is in both rectangles.
			if !i.IsEmpty() {
				c := i.TopLeft()
				if !a.Contains(c) || !b.Contains(c) {
					t.Errorf("the top left corner %v of %v is not in both %v and %v", c, i, a, b)
				}
			}
			// The bounding box of two non-empty rectangles is not empty.
			if u.IsEmpty() && !math.IsInf(u.Right-u.Left, 0) && !math.IsInf(u.Bottom-u.Top, 0) {
				t.Errorf("the bounding box %v of non-empty %v and %v is empty", u, a, b)
			}
		}
	})
}
