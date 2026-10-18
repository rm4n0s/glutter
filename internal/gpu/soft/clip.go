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
	return c.combine(b, func(x, y int) float32 { return shape.At(x, y) })
}

// subtract returns the clip with the shape cut out of it. A nil shape cuts
// nothing.
func (c clip) subtract(shape *Mask) clip {
	if c.empty() || shape == nil {
		return c
	}
	return c.combine(c.bounds, func(x, y int) float32 { return 1 - shape.At(x, y) })
}

// combine multiplies the coverage of the clip inside b by f, and drops the mask
// when everything in the result is covered fully or not at all along its edge.
func (c clip) combine(b image.Rectangle, f func(x, y int) float32) clip {
	cov := make([]float32, b.Dx()*b.Dy())
	full := true
	nonzero := false
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := f(x, y) * c.at(x, y)
			cov[i] = v
			i++
			if v != 1 {
				full = false
			}
			if v != 0 {
				nonzero = true
			}
		}
	}
	switch {
	case !nonzero:
		return clip{}
	case full:
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
	out := &Mask{Rect: m.Rect, Cov: make([]float32, len(m.Cov))}
	for i, c := range m.Cov {
		if c >= 0.5 {
			out.Cov[i] = 1
		}
	}
	return out
}

// limit returns the part of the rectangle that the clip reaches.
func (c clip) limit(r image.Rectangle) image.Rectangle { return r.Intersect(c.bounds) }
