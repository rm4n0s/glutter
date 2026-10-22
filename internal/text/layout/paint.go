package layout

import (
	"math"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/rm4n0s/glutter/ui"
)

// glyphKey identifies the outline of a glyph at a size.
type glyphKey struct {
	face *font.Face
	gid  font.GID
	size float64
}

// colorKey identifies the picture of a color glyph with a foreground color.
type colorKey struct {
	face *font.Face
	gid  font.GID
	fg   ui.Color
}

// maxCachedGlyphs bounds each of the glyph caches; when one is full, the entry that was
// used least recently is dropped.
const maxCachedGlyphs = 1 << 14

// glyphPath returns the outline of a glyph as a path with its origin at the pen position and
// y growing down, scaled to the size; nil if the glyph has no outline. The same path is
// returned for the same glyph and size, so that the renderers recognize it.
func (e *Engine) glyphPath(face *font.Face, gid font.GID, size float64) *ui.Path {
	key := glyphKey{face, gid, size}
	if p, ok := e.paths.get(key); ok {
		return p
	}
	var path *ui.Path
	if out, ok := face.GlyphDataOutline(tables.GlyphID(gid)); ok && len(out.Segments) > 0 {
		path = outlinePath(out.Segments, size/float64(face.Upem()), -size/float64(face.Upem()))
	}
	e.paths.put(key, path)
	return path
}

// outlinePath converts a glyph outline in font units to a path, scaling x by sx and y by sy.
func outlinePath(segments []font.Segment, sx, sy float64) *ui.Path {
	path := ui.NewPath()
	open := false
	pt := func(i int, s font.Segment) (float64, float64) {
		return float64(s.Args[i].X) * sx, float64(s.Args[i].Y) * sy
	}
	for _, s := range segments {
		switch s.Op {
		case ot.SegmentOpMoveTo:
			if open {
				path.Close()
			}
			x, y := pt(0, s)
			path.MoveTo(x, y)
			open = true
		case ot.SegmentOpLineTo:
			x, y := pt(0, s)
			path.LineTo(x, y)
		case ot.SegmentOpQuadTo:
			x1, y1 := pt(0, s)
			x2, y2 := pt(1, s)
			path.QuadraticBezierTo(x1, y1, x2, y2)
		case ot.SegmentOpCubeTo:
			x1, y1 := pt(0, s)
			x2, y2 := pt(1, s)
			x3, y3 := pt(2, s)
			path.CubicTo(x1, y1, x2, y2, x3, y3)
		}
	}
	if open {
		path.Close()
	}
	return path
}

// foreground is the paint that draws the text of a style.
func foreground(st *ui.ResolvedTextStyle) *ui.Paint {
	if st.Foreground != nil {
		return ui.NewPaintFrom(st.Foreground)
	}
	p := ui.NewPaint()
	p.SetColor(st.Color)
	return p
}

// PaintParagraph implements ui.ParagraphPainter: for each line, the backgrounds, the shadows,
// the glyphs and the decorations of its runs.
//
// Flutter: engine/src/flutter/third_party/skia/modules/skparagraph/src/TextLine.cpp TextLine::paint
func (p *paragraph) PaintParagraph(emit func(ui.ParagraphDraw)) {
	byLine := map[int][]DecorationSegment{}
	for _, d := range p.Decorations() {
		byLine[d.Line] = append(byLine[d.Line], d)
	}
	for li, l := range p.lines {
		p.paintBackgrounds(l, emit)
		p.paintShadows(l, emit)
		p.paintGlyphs(l, emit)
		for _, d := range byLine[li] {
			paintDecoration(d, emit)
		}
	}
}

// paintBackgrounds draws the background paints of the runs of a line.
func (p *paragraph) paintBackgrounds(l *line, emit func(ui.ParagraphDraw)) {
	geo := p.geometry(l)
	for ri := range l.runs {
		pr := &l.runs[ri]
		st := &p.styles[pr.run.style]
		if st.Background == nil || pr.run.ph != nil {
			continue
		}
		x0, x1 := math.Inf(1), math.Inf(-1)
		for r := pr.start; r < pr.end && !pr.ellipsis; r++ {
			if b := geo[r-l.start]; b.run == ri {
				x0, x1 = min(x0, b.x0), max(x1, b.x1)
			}
		}
		if pr.ellipsis {
			x0, x1 = l.left+pr.x, l.left+pr.x+pr.width
		}
		if x1 <= x0 {
			continue
		}
		rect := ui.Rect{Left: x0, Top: l.baseline - pr.run.ascent, Right: x1, Bottom: l.baseline + pr.run.descent}
		emit(ui.ParagraphDraw{Op: ui.OpDrawRect{Rect: rect, Paint: *ui.NewPaintFrom(st.Background)}})
	}
}

// paintShadows draws the text shadows of the runs of a line: the glyph outlines in the color
// of the shadow, blurred, displaced.
func (p *paragraph) paintShadows(l *line, emit func(ui.ParagraphDraw)) {
	for ri := range l.runs {
		pr := &l.runs[ri]
		st := &p.styles[pr.run.style]
		if pr.run.ph != nil || pr.run.face == nil {
			continue
		}
		for _, sh := range st.Shadows {
			paint := ui.NewPaint()
			paint.SetColor(sh.Color)
			if sigma := sh.BlurSigma(); sigma > 0 {
				paint.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, sigma))
			}
			for _, g := range pr.glyphs {
				path := p.eng.glyphPath(pr.run.face, g.id, pr.run.size)
				if path == nil || p.drawsNothing(pr, &g) {
					continue
				}
				emit(ui.ParagraphDraw{
					Op:     ui.OpDrawPath{Path: path, Paint: *paint},
					Offset: ui.Offset{Dx: l.left + pr.x + g.x + sh.Offset.Dx, Dy: l.baseline + g.y + sh.Offset.Dy},
					Snap:   true,
				})
			}
		}
	}
}

// paintGlyphs draws the glyphs of a line: outlines filled with the text paint, and color
// glyphs as their own pictures.
func (p *paragraph) paintGlyphs(l *line, emit func(ui.ParagraphDraw)) {
	for ri := range l.runs {
		pr := &l.runs[ri]
		st := &p.styles[pr.run.style]
		face := pr.run.face
		if pr.run.ph != nil || face == nil {
			continue
		}
		paint := foreground(st)
		for _, g := range pr.glyphs {
			if p.drawsNothing(pr, &g) {
				continue
			}
			origin := ui.Offset{Dx: l.left + pr.x + g.x, Dy: l.baseline + g.y}
			if pic := p.eng.colorGlyph(face, g.id, st.Color); pic != nil {
				s := pr.run.size / float64(face.Upem())
				m := ui.NewMatrix4Diagonal3Values(float32(s), float32(-s), 1)
				emit(ui.ParagraphDraw{Op: ui.OpDrawPicture{Picture: pic}, Offset: origin, Matrix: m, Snap: true})
				continue
			}
			if path := p.eng.glyphPath(face, g.id, pr.run.size); path != nil {
				emit(ui.ParagraphDraw{Op: ui.OpDrawPath{Path: path, Paint: *paint}, Offset: origin, Snap: true})
			}
		}
	}
}

// paintDecoration draws one decoration line.
func paintDecoration(d DecorationSegment, emit func(ui.ParagraphDraw)) {
	paint := ui.NewPaint()
	paint.SetColor(d.Color)
	th := max(d.Thickness, 0.5)
	rect := func(x0, x1, y float64) {
		emit(ui.ParagraphDraw{Op: ui.OpDrawRect{Rect: ui.Rect{Left: x0, Top: y - th/2, Right: x1, Bottom: y + th/2}, Paint: *paint}})
	}
	dashes := func(on, off float64) {
		for x := d.X0; x < d.X1; x += on + off {
			rect(x, min(x+on, d.X1), d.Y)
		}
	}
	switch d.Style {
	case ui.TextDecorationStyleDouble:
		rect(d.X0, d.X1, d.Y)
		rect(d.X0, d.X1, d.Y+DoubleDecorationSpacing)
	case ui.TextDecorationStyleDotted:
		dashes(th, th)
	case ui.TextDecorationStyleDashed:
		dashes(4*th, 2*th)
	case ui.TextDecorationStyleWavy:
		emit(ui.ParagraphDraw{Op: ui.OpDrawPath{Path: wavyPath(d.X0, d.X1, d.Y, th), Paint: *strokePaint(d.Color, th)}})
	default:
		rect(d.X0, d.X1, d.Y)
	}
}

// strokePaint is a paint that strokes with a color and a width.
func strokePaint(c ui.Color, width float64) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(c)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	return p
}

// wavyPath is the sine-like line of a wavy decoration, from x0 to x1 around y; its quarter
// wave is as long as the thickness.
//
// Flutter: engine/src/flutter/third_party/skia/modules/skparagraph/src/Decorations.cpp Decorations::paint
func wavyPath(x0, x1, y, thickness float64) *ui.Path {
	path := ui.NewPath()
	path.MoveTo(x0, y)
	quarter := thickness
	x, cy, wave := x0, y, 0
	for x+quarter*2 < x1 {
		dy := -quarter
		if wave%2 != 0 {
			dy = quarter
		}
		path.QuadraticBezierTo(x+quarter, cy+dy, x+quarter*2, cy)
		x += quarter * 2
		wave++
	}
	if remaining := x1 - x; remaining > 0 {
		sign := 1.0
		if wave%2 == 0 {
			sign = -1
		}
		path.QuadraticBezierTo(x+remaining/2, cy+remaining/2*sign, x+remaining, cy+(remaining-remaining*remaining/(quarter*2))*sign)
	}
	return path
}
