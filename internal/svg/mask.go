package svg

import (
	"github.com/rm4n0s/glutter/ui"
)

// luminanceToAlpha is the color matrix of a luminance mask: the alpha of the result is the luminance of the color, with
// the coefficients of the specification, and the color is dropped. The mask content is drawn over black first, so
// that its transparency counts as the luminance it hides.
//
// SVG 1.1: filters.html#feColorMatrixElement (luminanceToAlpha), masking.html#EstablishingANewMask
var luminanceToAlpha = ui.NewColorFilterMatrix([20]float64{
	0, 0, 0, 0, 0,
	0, 0, 0, 0, 0,
	0, 0, 0, 0, 0,
	0.2125, 0.7154, 0.0721, 0, 0,
})

// maskUsable reports whether the mask can be drawn for the node: a mask in objectBoundingBox units needs a box with an
// area, and a mask that is being drawn already (it refers to itself) is an error, which leaves the element out.
//
// SVG 1.1: masking.html#EstablishingANewMask
func (p *painter) maskUsable(m *Mask, n *Node) bool {
	if p.active[m] {
		return false
	}
	for ; m != nil; m = m.Mask {
		if m.Units == UnitsObjectBoundingBox || m.ContentUnits == UnitsObjectBoundingBox {
			if _, ok := p.objectBBox(n); !ok {
				return false
			}
		}
	}
	return true
}

// beginMask opens the layer that holds the content that a mask applies to. endMask closes it.
func (p *painter) beginMask() bool {
	p.c.SaveLayer(nil, nil)
	return true
}

// endMask applies the mask to the content of the layer that beginMask opened, with BlendMode.dstIn: where the mask's
// alpha is 0 the content is gone, and where it is 1 the content stays.
//
// SVG 1.1: masking.html#EstablishingANewMask
func (p *painter) endMask(m *Mask, n *Node) {
	p.active[m] = true
	defer delete(p.active, m)
	bbox, hasBBox := p.objectBBox(n)
	paint := ui.NewPaint()
	paint.SetBlendMode(ui.BlendModeDstIn)
	p.c.SaveLayer(nil, paint)
	p.maskInto(m, bbox, hasBBox)
	p.c.Restore()
	p.c.Restore()
}

// maskInto draws the alpha of a mask into the current layer: the content of the mask inside its region, as luminance
// or as alpha, times the mask that the mask element has itself.
func (p *painter) maskInto(m *Mask, bbox ui.Rect, hasBBox bool) {
	region, ok := p.maskRegion(m, bbox, hasBBox)
	if ok {
		p.save()
		p.c.ClipRect(region)
		if m.Type == MaskLuminance {
			p.c.SaveLayer(nil, lumPaint())
			black := ui.NewPaint()
			black.SetColor(ui.NewColorFromARGB(255, 0, 0, 0))
			p.c.DrawPaint(black)
		}
		if m.ContentUnits == UnitsObjectBoundingBox {
			p.concat(Affine{A: bbox.Width(), D: bbox.Height(), E: bbox.Left, F: bbox.Top})
		}
		for _, c := range m.Children {
			p.drawNode(c, 1)
		}
		if m.Type == MaskLuminance {
			p.c.Restore()
		}
		p.restore()
	}
	if m.Mask != nil && ok {
		if p.active[m.Mask] {
			// A mask that refers to itself: nothing gets through.
			p.c.DrawColor(ui.NewColorFromARGB(0, 0, 0, 0), ui.BlendModeClear)
			return
		}
		p.active[m.Mask] = true
		inner := ui.NewPaint()
		inner.SetBlendMode(ui.BlendModeDstIn)
		p.c.SaveLayer(nil, inner)
		p.maskInto(m.Mask, bbox, hasBBox)
		p.c.Restore()
		delete(p.active, m.Mask)
	}
}

func lumPaint() *ui.Paint {
	paint := ui.NewPaint()
	paint.SetColorFilter(luminanceToAlpha)
	return paint
}

// maskRegion returns the region of a mask in the user space of the element: x, y, width and height in the units of the
// mask. It reports false for a region with no area.
func (p *painter) maskRegion(m *Mask, bbox ui.Rect, hasBBox bool) (ui.Rect, bool) {
	var r ui.Rect
	if m.Units == UnitsObjectBoundingBox {
		if !hasBBox {
			return r, false
		}
		r = ui.NewRectFromLTWH(bbox.Left+m.X.Resolve(1)*bbox.Width(), bbox.Top+m.Y.Resolve(1)*bbox.Height(),
			m.Width.Resolve(1)*bbox.Width(), m.Height.Resolve(1)*bbox.Height())
	} else {
		r = ui.NewRectFromLTWH(m.X.Resolve(p.vw), m.Y.Resolve(p.vh), m.Width.Resolve(p.vw), m.Height.Resolve(p.vh))
	}
	return r, r.Width() > 0 && r.Height() > 0
}
