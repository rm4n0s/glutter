package gpu

import (
	"image"

	"github.com/rm4n0s/glutter/ui"
)

// A clip mask is the coverage of the clip shapes of a clip (everything that is not an
// axis-aligned rectangle), drawn into an 8-bit texture over a rectangle of the surface. A
// draw under the clip multiplies its coverage by the mask's, as the software renderer does
// (SPEC §9.1, "Clips": antialiased through coverage masks).

// ClipKey hashes what a clip mask is made of: its area and its shapes, with the transform
// each was clipped under.
//
// Flutter: no counterpart
func ClipKey(area image.Rectangle, shapes []ClipShape) uint64 {
	h := NewHasher()
	h.I(area.Min.X)
	h.I(area.Min.Y)
	h.I(area.Max.X)
	h.I(area.Max.Y)
	for i := range shapes {
		s := &shapes[i]
		h.I(int(s.Kind))
		h.B(s.AntiAlias)
		h.B(s.Difference)
		for _, v := range s.Transform.Storage() {
			h.F64(float64(v))
		}
		switch s.Kind {
		case ClipShapeRect:
			h.F64(s.Rect.Left)
			h.F64(s.Rect.Top)
			h.F64(s.Rect.Right)
			h.F64(s.Rect.Bottom)
		case ClipShapeRRect:
			rr := s.RRect
			for _, v := range [...]float64{rr.Left, rr.Top, rr.Right, rr.Bottom, rr.TlRadiusX, rr.TlRadiusY, rr.TrRadiusX, rr.TrRadiusY, rr.BrRadiusX, rr.BrRadiusY, rr.BlRadiusX, rr.BlRadiusY} {
				h.F64(v)
			}
		case ClipShapeRSuperellipse:
			rr := s.RSuperellipse
			for _, v := range [...]float64{rr.Left, rr.Top, rr.Right, rr.Bottom, rr.TlRadiusX, rr.TlRadiusY, rr.TrRadiusX, rr.TrRadiusY, rr.BrRadiusX, rr.BrRadiusY, rr.BlRadiusX, rr.BlRadiusY} {
				h.F64(v)
			}
		case ClipShapePath:
			h.U64(s.Path.Fingerprint())
		}
	}
	return h.Sum()
}

// Outline is the outline of a clip shape.
//
// Flutter: no counterpart
func (s *ClipShape) Outline() *ui.Path {
	p := ui.NewPath()
	switch s.Kind {
	case ClipShapeRect:
		p.AddRect(s.Rect)
	case ClipShapeRRect:
		p.AddRRect(s.RRect)
	case ClipShapeRSuperellipse:
		p.AddRSuperellipse(s.RSuperellipse)
	case ClipShapePath:
		return s.Path
	}
	return p
}

// ClipCoverage makes the coverage of the shapes of the clip over the area, as the bytes of an
// 8-bit texture, row by row from the top. empty says that the shapes leave nothing (no bytes).
// rasterize fills the shape's outline under its transform, with or without anti-aliasing, over
// the area: one coverage from 0 to 1 per pixel of rect, or nil for a shape that covers nothing.
//
// Flutter: no counterpart
func ClipCoverage(c Clip, area image.Rectangle, rasterize func(path *ui.Path, m *ui.Matrix4, antiAlias bool, area image.Rectangle) (rect image.Rectangle, cov []float32)) (data []byte, empty bool) {
	w := area.Dx()
	cov := make([]float32, w*area.Dy())
	for i := range cov {
		cov[i] = 1
	}
	for i := range c.Shapes {
		s := &c.Shapes[i]
		rect, mask := rasterize(s.Outline(), &s.Transform, s.AntiAlias, area)
		if s.Difference {
			// Cut the shape out: where it covers, the clip is left by the rest.
			if mask != nil {
				cut(cov, area, rect, mask, true)
			}
			continue
		}
		if mask == nil {
			return nil, true
		}
		cut(cov, area, rect, mask, false)
	}
	data = CoverageBytes(cov)
	for _, b := range data {
		if b != 0 {
			return data, false
		}
	}
	return nil, true
}

// cut multiplies the coverage of a clip, over the area, by a shape's mask, or by what the
// mask leaves when subtract is set.
func cut(cov []float32, area, rect image.Rectangle, mask []float32, subtract bool) {
	w := area.Dx()
	mw := rect.Dx()
	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := cov[(y-area.Min.Y)*w : (y-area.Min.Y+1)*w]
		if y < rect.Min.Y || y >= rect.Max.Y {
			// Above or below the mask: it covers nothing here, so a cut keeps the row and a cut-out
			// (subtract) has nothing to take out of it.
			if !subtract {
				clear(row)
			}
			continue
		}
		mrow := mask[(y-rect.Min.Y)*mw : (y-rect.Min.Y+1)*mw]
		for x := range row {
			var v float32
			if mx := area.Min.X + x; mx >= rect.Min.X && mx < rect.Max.X {
				v = mrow[mx-rect.Min.X]
			}
			if subtract {
				v = 1 - v
			}
			row[x] *= v
		}
	}
}
