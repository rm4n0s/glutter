package soft

import (
	"image"
)

// clip is the part of the surface that drawing may change: a rectangle of pixels
// and, when the edge of the clip is not on pixel boundaries or the clip is not a
// rectangle, the coverage of each pixel in it. A clip is a value; its mask is
// never changed once made.
type clip struct {
	// bounds is the area of the clip in device pixels; empty means that nothing
	// can be drawn.
	bounds image.Rectangle
	// mask is the coverage inside bounds, or nil when every pixel of bounds is
	// covered fully. When it is not nil its Rect equals bounds.
	mask *Mask
}

func newClip(r image.Rectangle) clip { return clip{bounds: r} }

// empty says that the clip lets nothing through.
func (c clip) empty() bool { return c.bounds.Empty() }

// at returns how much of the pixel the clip lets through.
func (c clip) at(x, y int) float32 {
	if !(image.Point{X: x, Y: y}).In(c.bounds) {
		return 0
	}
	if c.mask == nil {
		return 1
	}
	return c.mask.Cov[(y-c.bounds.Min.Y)*c.bounds.Dx()+(x-c.bounds.Min.X)]
}

// intersect returns the clip narrowed to the shape whose coverage is shape. A
// nil shape is the empty shape.
func (c clip) intersect(shape *Mask) clip {
	if c.empty() || shape == nil {
		return clip{}
	}
	b := c.bounds.Intersect(shape.Rect)
	if b.Empty() {
		return clip{}
	}
	cov := make([]float32, b.Dx()*b.Dy())
	st := coverState{full: true}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		out := cov[(y-b.Min.Y)*b.Dx() : (y-b.Min.Y+1)*b.Dx()]
		sh := shape.Cov[(y-shape.Rect.Min.Y)*shape.Rect.Dx()+(b.Min.X-shape.Rect.Min.X):][:len(out)]
		if c.mask == nil {
			copy(out, sh)
		} else {
			cl := c.mask.Cov[(y-c.bounds.Min.Y)*c.bounds.Dx()+(b.Min.X-c.bounds.Min.X):][:len(out)]
			for i, v := range sh {
				out[i] = v * cl[i]
			}
		}
		st.add(out)
	}
	return c.made(b, cov, st)
}

// subtract returns the clip with the shape cut out of it. A nil shape cuts
// nothing.
func (c clip) subtract(shape *Mask) clip {
	if c.empty() || shape == nil {
		return c
	}
	b := c.bounds
	cov := make([]float32, b.Dx()*b.Dy())
	st := coverState{full: true}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		out := cov[(y-b.Min.Y)*b.Dx() : (y-b.Min.Y+1)*b.Dx()]
		for i := range out {
			out[i] = 1
		}
		// Where the shape has coverage, 1 - coverage is what is left.
		if y >= shape.Rect.Min.Y && y < shape.Rect.Max.Y {
			x0, x1 := max(b.Min.X, shape.Rect.Min.X), min(b.Max.X, shape.Rect.Max.X)
			if x0 < x1 {
				sh := shape.Cov[(y-shape.Rect.Min.Y)*shape.Rect.Dx()+(x0-shape.Rect.Min.X):][:x1-x0]
				for i, v := range sh {
					out[x0-b.Min.X+i] = 1 - v
				}
			}
		}
		if c.mask != nil {
			cl := c.mask.Cov[(y-c.bounds.Min.Y)*c.bounds.Dx():][:len(out)]
			for i := range out {
				out[i] *= cl[i]
			}
		}
		st.add(out)
	}
	return c.made(b, cov, st)
}

// coverState tells, for the coverage made so far, whether every value is 1 and
// whether any is not 0.
type coverState struct{ full, nonzero bool }

func (s *coverState) add(row []float32) {
	for _, v := range row {
		if v != 1 {
			s.full = false
		}
		if v != 0 {
			s.nonzero = true
		}
	}
}

// made returns the clip over b with the coverage cov, and drops the mask when everything
// in the result is covered fully or not at all.
func (c clip) made(b image.Rectangle, cov []float32, st coverState) clip {
	switch {
	case !st.nonzero:
		return clip{}
	case st.full:
		return clip{bounds: b}
	}
	return clip{bounds: b, mask: &Mask{Rect: b, Cov: cov}}
}

// threshold turns a mask into one without partial coverage: the pixels covered
// by half or more are covered fully. It is how a clip without anti-aliasing
// samples the shape at the pixel centers.
func threshold(m *Mask) *Mask {
	if m == nil {
		return nil
	}
	out := &Mask{Rect: m.Rect, Cov: make([]float32, len(m.Cov)), rows: m.rows}
	for i, c := range m.Cov {
		if c >= 0.5 {
			out.Cov[i] = 1
		}
	}
	return out
}

// limit returns the part of the rectangle that the clip reaches.
func (c clip) limit(r image.Rectangle) image.Rectangle { return r.Intersect(c.bounds) }
