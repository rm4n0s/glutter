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
	// Cov holds the coverage of each pixel of Rect, row by row from the top.
	Cov []float32
}

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
	bounds, ok := contourBounds(contours)
	if !ok {
		return nil
	}
	r := bounds.Intersect(clip)
	if r.Empty() {
		return nil
	}
	ac := newAccum(r.Dx(), r.Dy())
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
// the signed area covered.
type accum struct {
	w, h   int
	stride int
	a      []float64
}

func newAccum(w, h int) *accum {
	s := w + 2
	return &accum{w: w, h: h, stride: s, a: make([]float64, s*h)}
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
}

// mask sums the rows and applies the fill rule.
func (ac *accum) mask(r image.Rectangle, rule FillRule) *Mask {
	cov := make([]float32, ac.w*ac.h)
	any := false
	for y := 0; y < ac.h; y++ {
		line := ac.a[y*ac.stride : y*ac.stride+ac.w]
		acc := 0.0
		for x, v := range line {
			acc += v
			c := fold(acc, rule)
			if c < 1.0/1024 {
				continue
			}
			cov[y*ac.w+x] = float32(c)
			any = true
		}
	}
	if !any {
		return nil
	}
	return &Mask{Rect: r, Cov: cov}
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
