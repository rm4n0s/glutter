package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// FillRule says which points are inside a shape whose outline crosses itself.
//
// Flutter: ui/painting.dart PathFillType
type FillRule int

// The fill rules.
const (
	// FillNonZero fills a point when the outline winds around it a number of
	// times other than zero.
	FillNonZero FillRule = iota
	// FillEvenOdd fills a point when a ray from it crosses the outline an odd
	// number of times.
	FillEvenOdd
)

// maxCoord bounds the coordinates the rasterizer works with, so that arithmetic
// on them stays finite.
const maxCoord = 1e8

// Mask is the coverage of a shape over a rectangle of pixels: 0 where the shape
// does not touch a pixel and 1 where it covers the pixel fully.
//
// Flutter: no counterpart (SkMask)
type Mask struct {
	// Rect is the area of the surface the mask covers.
	Rect image.Rectangle
	// Cov holds the coverage of each pixel of Rect, row by row from the top. The
	// renderer's own masks of a fill that covers its rectangle fully have none (At and
	// Sum are not used on them).
	Cov []float32

	// rows, when it is not nil, has one entry for each row of Rect: the part of the row,
	// as offsets from Rect.Min.X, where the coverage may be other than 0 (lo inclusive,
	// hi exclusive). It is a hint for the renderer that Rasterize makes; a mask built
	// any other way, or changed after Rasterize made it, has none.
	rows []rowExtent
}

// rowExtent is the part of a row of a mask that has coverage: from lo up to hi.
type rowExtent struct{ lo, hi int32 }

// At returns the coverage of the pixel at x, y, or 0 outside the mask.
func (m *Mask) At(x, y int) float32 {
	if m == nil || !(image.Point{X: x, Y: y}).In(m.Rect) {
		return 0
	}
	return m.Cov[(y-m.Rect.Min.Y)*m.Rect.Dx()+(x-m.Rect.Min.X)]
}

// Sum returns the total coverage: the area of the shape in pixels.
func (m *Mask) Sum() float64 {
	if m == nil {
		return 0
	}
	var s float64
	for _, c := range m.Cov {
		s += float64(c)
	}
	return s
}

// Rasterize computes the coverage of the area enclosed by the contours, each of
// which is closed implicitly, inside the clip rectangle. The coverage of a pixel
// is the exact area of the pixel the shape covers where only one edge crosses it;
// with several edges in one pixel it is the area under the fill rule applied to
// the accumulated winding. It returns nil when nothing is covered. Contours with
// a point that is not finite are skipped.
//
// Flutter: no counterpart (Skia's SkScan_AntiPath)
func Rasterize(contours [][]ui.Offset, rule FillRule, clip image.Rectangle) *Mask {
	return rasterize(contours, rule, clip, nil)
}

// scratch is the memory that rasterize uses between shapes, so that a renderer does not
// allocate and clear a buffer the size of the shape for each one. Its buffer is all zeros
// between calls. A scratch is used by one goroutine at a time.
type scratch struct{ buf []float64 }

// rasterize is Rasterize with the buffer of sc, which can be nil.
func rasterize(contours [][]ui.Offset, rule FillRule, clip image.Rectangle, sc *scratch) *Mask {
	bounds, ok := contourBounds(contours)
	if !ok {
		return nil
	}
	r := bounds.Intersect(clip)
	if r.Empty() {
		return nil
	}
	ac := newAccum(r.Dx(), r.Dy(), sc)
	ox, oy := float64(r.Min.X), float64(r.Min.Y)
	for _, c := range contours {
		if len(c) < 2 || !finite(c) {
			continue
		}
		prev := clampPoint(c[len(c)-1])
		for _, q := range c {
			q = clampPoint(q)
			ac.line(prev.Dx-ox, prev.Dy-oy, q.Dx-ox, q.Dy-oy)
			prev = q
		}
	}
	return ac.mask(r, rule)
}

// contourBounds returns the pixels the contours touch, and false when there is
// no usable point.
func contourBounds(contours [][]ui.Offset) (image.Rectangle, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	any := false
	for _, c := range contours {
		if len(c) < 2 || !finite(c) {
			continue
		}
		for _, p := range c {
			p = clampPoint(p)
			minX, maxX = min(minX, p.Dx), max(maxX, p.Dx)
			minY, maxY = min(minY, p.Dy), max(maxY, p.Dy)
			any = true
		}
	}
	if !any {
		return image.Rectangle{}, false
	}
	return image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))), true
}

func finite(c []ui.Offset) bool {
	for _, p := range c {
		if math.IsNaN(p.Dx) || math.IsNaN(p.Dy) || math.IsInf(p.Dx, 0) || math.IsInf(p.Dy, 0) {
			return false
		}
	}
	return true
}

func clampPoint(p ui.Offset) ui.Offset {
	return ui.Offset{Dx: max(-maxCoord, min(maxCoord, p.Dx)), Dy: max(-maxCoord, min(maxCoord, p.Dy))}
}

// accum is the accumulation buffer: for every pixel, the change of the signed
// area to the left of the edges that cross it, summed along the row it becomes
// the signed area covered. It remembers, for each row, the cells that were written, so
// that the rows with nothing in them cost nothing and the buffer can be cleared again
// cell by cell.
type accum struct {
	w, h   int
	stride int
	a      []float64
	// ext is the part of each row, from lo up to hi (exclusive), that was written.
	ext []rowExtent
	sc  *scratch
}

func newAccum(w, h int, sc *scratch) *accum {
	s := w + 2
	n := s * h
	ac := &accum{w: w, h: h, stride: s, sc: sc, ext: make([]rowExtent, h)}
	if sc != nil && cap(sc.buf) >= n {
		ac.a = sc.buf[:n]
	} else {
		ac.a = make([]float64, n)
		if sc != nil {
			sc.buf = ac.a
		}
	}
	for i := range ac.ext {
		ac.ext[i] = rowExtent{lo: int32(s)}
	}
	return ac
}

// touch records that the cells from lo up to hi of the row were written.
func (ac *accum) touch(y, lo, hi int) {
	e := &ac.ext[y]
	e.lo = min(e.lo, int32(lo))
	e.hi = max(e.hi, int32(hi))
}

// line adds the edge from (x0, y0) to (x1, y1).
func (ac *accum) line(x0, y0, x1, y1 float64) {
	if y0 == y1 {
		return
	}
	dir := 1.0
	if y0 > y1 {
		x0, y0, x1, y1 = x1, y1, x0, y0
		dir = -1
	}
	h := float64(ac.h)
	if y1 <= 0 || y0 >= h {
		return
	}
	dxdy := (x1 - x0) / (y1 - y0)
	x, ys := x0, y0
	if ys < 0 {
		x -= ys * dxdy
		ys = 0
	}
	yEnd := min(y1, h)
	for y := int(ys); float64(y) < yEnd; y++ {
		top := max(float64(y), ys)
		bot := min(float64(y+1), yEnd)
		dy := bot - top
		xn := x + dxdy*dy
		ac.row(y, x, xn, dy*dir)
		x = xn
	}
}

// row adds the part of an edge inside one pixel row, going from xa to xb in x
// and covering d of the row's height. Parts left of the surface count as
// covering the first pixel's left side, parts right of it are dropped.
func (ac *accum) row(y int, xa, xb, d float64) {
	w := float64(ac.w)
	if (xa >= 0 && xa <= w) && (xb >= 0 && xb <= w) {
		ac.inside(y, xa, xb, d)
		return
	}
	// Cut the part where it crosses x = 0 and x = w.
	ts := [4]float64{0, 1, 1, 1}
	n := 1
	if xa != xb {
		var cuts [2]float64
		k := 0
		for _, edge := range [2]float64{0, w} {
			if t := (edge - xa) / (xb - xa); t > 0 && t < 1 {
				cuts[k] = t
				k++
			}
		}
		if k == 2 && cuts[0] > cuts[1] {
			cuts[0], cuts[1] = cuts[1], cuts[0]
		}
		copy(ts[1:], cuts[:k])
		n += k
	}
	ts[n] = 1
	n++
	for i := 0; i+1 < n; i++ {
		t0, t1 := ts[i], ts[i+1]
		if t1 <= t0 {
			continue
		}
		pa, pb := xa+(xb-xa)*t0, xa+(xb-xa)*t1
		dp := d * (t1 - t0)
		switch mid := (pa + pb) / 2; {
		case mid <= 0:
			ac.a[y*ac.stride] += dp
			ac.touch(y, 0, 1)
		case mid >= w:
		default:
			ac.inside(y, max(0, min(w, pa)), max(0, min(w, pb)), dp)
		}
	}
}

// inside adds an edge part wholly inside the surface horizontally.
func (ac *accum) inside(y int, x, xn, d float64) {
	line := ac.a[y*ac.stride : (y+1)*ac.stride]
	x0, x1 := x, xn
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	x0floor := math.Floor(x0)
	x0i := int(x0floor)
	x1ceil := math.Ceil(x1)
	x1i := int(x1ceil)
	if x1i <= x0i+1 {
		xmf := 0.5*(x+xn) - x0floor
		line[x0i] += d - d*xmf
		line[x0i+1] += d * xmf
		ac.touch(y, x0i, x0i+2)
		return
	}
	s := 1 / (x1 - x0)
	x0f := x0 - x0floor
	a0 := 0.5 * s * (1 - x0f) * (1 - x0f)
	x1f := x1 - x1ceil + 1
	am := 0.5 * s * x1f * x1f
	line[x0i] += d * a0
	if x1i == x0i+2 {
		line[x0i+1] += d * (1 - a0 - am)
	} else {
		a1 := s * (1.5 - x0f)
		line[x0i+1] += d * (a1 - a0)
		for xi := x0i + 2; xi < x1i-1; xi++ {
			line[xi] += d * s
		}
		a2 := a1 + float64(x1i-x0i-3)*s
		line[x1i-1] += d * (1 - a2 - am)
	}
	line[x1i] += d * am
	ac.touch(y, x0i, x1i+1)
}

// mask sums the rows and applies the fill rule. It clears the buffer again.
func (ac *accum) mask(r image.Rectangle, rule FillRule) *Mask {
	cov := make([]float32, ac.w*ac.h)
	rows := make([]rowExtent, ac.h)
	any := false
	for y := 0; y < ac.h; y++ {
		e := ac.ext[y]
		if e.hi <= e.lo {
			continue
		}
		base := y * ac.stride
		line := ac.a[base : base+ac.stride]
		out := cov[y*ac.w : (y+1)*ac.w]
		lo, hi := int(e.lo), min(int(e.hi), ac.w)
		acc := 0.0
		first, last := -1, -1
		var prev float32
		for x := lo; x < hi; x++ {
			v := line[x]
			if v != 0 {
				acc += v
				c := fold(acc, rule)
				prev = 0
				if c >= 1.0/1024 {
					prev = float32(c)
				}
			}
			if prev != 0 {
				out[x] = prev
				if first < 0 {
					first = x
				}
				last = x + 1
			}
		}
		// Past the last cell that was written, the area stays what it is.
		if hi < ac.w {
			if c := fold(acc, rule); c >= 1.0/1024 {
				f := float32(c)
				for x := hi; x < ac.w; x++ {
					out[x] = f
				}
				if first < 0 {
					first = hi
				}
				last = ac.w
			}
		}
		if first >= 0 {
			rows[y] = rowExtent{lo: int32(first), hi: int32(last)}
			any = true
		}
		clear(line[e.lo:min(int(e.hi), ac.stride)])
	}
	if !any {
		return nil
	}
	return &Mask{Rect: r, Cov: cov, rows: rows}
}

// fold turns an accumulated signed area into coverage.
func fold(acc float64, rule FillRule) float64 {
	v := math.Abs(acc)
	if rule == FillEvenOdd {
		v -= 2 * math.Floor(v/2)
		if v > 1 {
			v = 2 - v
		}
		return v
	}
	return min(v, 1)
}
