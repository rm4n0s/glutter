package offscreen

import (
	"math"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/textdraw"
	"github.com/rm4n0s/glutter/ui"
)

// Detacher copies scenes, on the UI thread, into scenes that hold nothing the
// framework changes afterwards, so that any goroutine can draw the copy while the UI
// thread goes on with the next frame. A scene is not safe to draw on another goroutine
// for these reasons:
//
//   - a paragraph draws by asking the text engine that laid it out, whose caches have no
//     lock, and the framework lays the same paragraph out again in a later frame;
//   - the framework disposes the handle of an image when it lets go of it, and the
//     renderer refuses an image that is disposed.
//
// The copy replaces each paragraph by the drawing the paragraph makes now, and each image
// by a handle of its own on the same pixels. Everything else a scene holds (layers,
// pictures without a paragraph or an image, paths, paints, gradients, filters) is
// immutable once it is recorded, and is shared with the source.
//
// The copy of a picture is reused when the next scene has the same picture, and a layer
// whose pictures were all kept is the same layer in both scenes, so that the damage
// tracker of a Surface still sees what did not change. A Detacher keeps the copies of the
// last scene only, and is used by one goroutine at a time (the UI thread).
//
// Flutter: no counterpart
type Detacher struct {
	cur, prev copies
}

// copies are the copies a Detacher made for one scene, by the object they copy.
type copies struct {
	pictures map[*ui.Picture]*ui.Picture
	images   map[*ui.Image]*ui.Image
	shaders  map[*ui.ImageShader]*ui.ImageShader
}

func newCopies() copies {
	return copies{
		pictures: map[*ui.Picture]*ui.Picture{},
		images:   map[*ui.Image]*ui.Image{},
		shaders:  map[*ui.ImageShader]*ui.ImageShader{},
	}
}

// NewDetacher returns a detacher that has seen no scene.
//
// Flutter: no counterpart
func NewDetacher() *Detacher { return &Detacher{cur: newCopies(), prev: newCopies()} }

// Detach returns a scene that draws the same pixels as the scene, and that any goroutine
// can draw while the framework goes on. Call it on the UI thread, after the scene is
// built and before the framework changes what the scene refers to (lays a paragraph out
// again, disposes an image). A nil scene gives nil.
//
// Flutter: no counterpart
func (d *Detacher) Detach(scene *ui.Scene) *ui.Scene {
	if scene == nil {
		return nil
	}
	d.cur = newCopies()
	out := scene.MapContent(d.picture, d.shader)
	d.prev, d.cur = d.cur, copies{}
	return out
}

// picture returns the copy of a picture: the picture itself when it holds nothing to
// detach.
func (d *Detacher) picture(p *ui.Picture) *ui.Picture {
	if p == nil {
		return nil
	}
	if c, ok := d.cur.pictures[p]; ok {
		return c
	}
	if c, ok := d.prev.pictures[p]; ok {
		d.cur.pictures[p] = c
		return c
	}
	// Record the picture first as its own copy, which a picture that refers to itself
	// (it cannot) would find.
	d.cur.pictures[p] = p
	ops := p.DisplayList().Ops()
	var out []ui.DisplayOp
	for i, op := range ops {
		n, changed := d.op(op)
		if changed && out == nil {
			out = make([]ui.DisplayOp, len(ops))
			copy(out, ops[:i])
		}
		if out != nil {
			if !changed {
				n = op
			}
			out[i] = n
		}
	}
	c := p
	if out != nil {
		c = ui.NewPictureFromOps(p.CullRect(), out)
	}
	d.cur.pictures[p] = c
	return c
}

// image returns a handle of the detacher's own on the pixels of the image. A handle that
// is disposed stays disposed in the copy, so that drawing it fails as it does in the
// source.
func (d *Detacher) image(img *ui.Image) *ui.Image {
	if img == nil {
		return nil
	}
	if c, ok := d.cur.images[img]; ok {
		return c
	}
	c, ok := d.prev.images[img]
	if !ok {
		c = img.Clone()
		if img.IsDisposed() {
			c.Dispose()
		}
	}
	d.cur.images[img] = c
	return c
}

// shader returns the copy of an image shader, which refers to a copy of its image; any
// other shader (a gradient) is immutable and is returned as it is.
func (d *Detacher) shader(sh ui.Shader) ui.Shader {
	is, ok := sh.(*ui.ImageShader)
	if !ok || is == nil {
		return sh
	}
	if c, ok := d.cur.shaders[is]; ok {
		return c
	}
	c, ok := d.prev.shaders[is]
	if !ok {
		c = ui.NewImageShader(d.image(is.Image()), is.TileModeX(), is.TileModeY(), is.Matrix4(), ui.ImageShaderOpts{FilterQuality: is.FilterQuality()})
	}
	d.cur.shaders[is] = c
	return c
}

// paint returns the paint with its shader detached, and whether it changed.
func (d *Detacher) paint(p ui.Paint) (ui.Paint, bool) {
	is, ok := p.Shader().(*ui.ImageShader)
	if !ok || is == nil {
		return p, false
	}
	p.SetShader(d.shader(is))
	return p, true
}

// op returns the copy of an operation, and whether it differs from the operation.
func (d *Detacher) op(op ui.DisplayOp) (ui.DisplayOp, bool) {
	switch v := op.(type) {
	case ui.OpDrawParagraph:
		return d.paragraph(v)
	case ui.OpDrawPicture:
		if c := d.picture(v.Picture); c != v.Picture {
			v.Picture = c
			return v, true
		}
	case ui.OpDrawPaint:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawLine:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawRect:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawRRect:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawDRRect:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawRSuperellipse:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawOval:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawCircle:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawArc:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawPath:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawPoints:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawVertices:
		if p, ch := d.paint(v.Paint); ch {
			v.Paint = p
			return v, true
		}
	case ui.OpDrawImage:
		p, _ := d.paint(v.Paint)
		v.Image, v.Paint = d.image(v.Image), p
		return v, true
	case ui.OpDrawImageRect:
		p, _ := d.paint(v.Paint)
		v.Image, v.Paint = d.image(v.Image), p
		return v, true
	case ui.OpDrawImageNine:
		p, _ := d.paint(v.Paint)
		v.Image, v.Paint = d.image(v.Image), p
		return v, true
	case ui.OpDrawAtlas:
		p, _ := d.paint(v.Paint)
		v.Atlas, v.Paint = d.image(v.Atlas), p
		return v, true
	}
	return op, false
}

// paragraph returns the operation that draws what the paragraph draws now. A paragraph
// that cannot draw itself (it has no painter) draws nothing and is kept as it is.
func (d *Detacher) paragraph(v ui.OpDrawParagraph) (ui.DisplayOp, bool) {
	if v.Paragraph == nil {
		return v, false
	}
	painter, ok := v.Paragraph.Layouter().(ui.ParagraphPainter)
	if !ok {
		return v, false
	}
	f := &frozenParagraph{width: v.Paragraph.Width(), height: v.Paragraph.Height()}
	painter.PaintParagraph(func(draw ui.ParagraphDraw) {
		if o, changed := d.op(draw.Op); changed {
			draw.Op = o
		}
		if draw.Matrix != nil {
			draw.Matrix = draw.Matrix.Clone()
		}
		f.add(draw)
	})
	f.finish()
	v.Paragraph = ui.NewParagraph(f)
	return v, true
}

// frozenParagraph is the layout of a paragraph that was detached: it draws what the
// paragraph drew, and knows nothing else. Calling another method of the layout panics.
//
// The draws are kept in groups of consecutive draws, each with the rows it reaches, so that a
// renderer that draws a screenful of a long text skips the groups outside it (the same as the
// lines of the layout of the text engine, textdraw.RowPainter), and with the box of them all
// (textdraw.BoundsPainter), so that the damage tracker need not look at each draw.
type frozenParagraph struct {
	ui.ParagraphLayout
	width, height float64
	draws         []ui.ParagraphDraw
	groups        []drawGroup
	// open is the group being filled: the draws from the end of the last group on.
	open drawGroup
	// bounds is the box of all the draws, in the paragraph's coordinates; infinite when a draw
	// reaches further than its geometry (a blur) or cannot be bounded.
	bounds ui.Rect
}

// drawGroup is the draws up to end (and from the end of the group before it) with the rows,
// in the paragraph's coordinates, that they reach.
type drawGroup struct {
	end         int
	top, bottom float64
}

var (
	_ ui.ParagraphPainter    = (*frozenParagraph)(nil)
	_ textdraw.RowPainter    = (*frozenParagraph)(nil)
	_ textdraw.BoundsPainter = (*frozenParagraph)(nil)
)

// groupSize is how many draws make a group.
const groupSize = 64

// add keeps a draw and widens its group and the box to what it reaches.
func (f *frozenParagraph) add(draw ui.ParagraphDraw) {
	r := drawReach(draw)
	if len(f.draws) == 0 {
		f.open.top, f.open.bottom = math.Inf(1), math.Inf(-1)
		f.bounds = ui.Rect{Left: math.Inf(1), Top: math.Inf(1), Right: math.Inf(-1), Bottom: math.Inf(-1)}
	}
	f.draws = append(f.draws, draw)
	f.open.top, f.open.bottom = min(f.open.top, r.Top), max(f.open.bottom, r.Bottom)
	f.bounds = ui.Rect{Left: min(f.bounds.Left, r.Left), Top: min(f.bounds.Top, r.Top), Right: max(f.bounds.Right, r.Right), Bottom: max(f.bounds.Bottom, r.Bottom)}
	if len(f.draws)-f.groupStart() >= groupSize {
		f.close()
	}
}

func (f *frozenParagraph) groupStart() int {
	if n := len(f.groups); n > 0 {
		return f.groups[n-1].end
	}
	return 0
}

func (f *frozenParagraph) close() {
	f.open.end = len(f.draws)
	f.groups = append(f.groups, f.open)
	f.open = drawGroup{top: math.Inf(1), bottom: math.Inf(-1)}
}

// finish closes the last group.
func (f *frozenParagraph) finish() {
	if len(f.draws) > f.groupStart() {
		f.close()
	}
}

// drawReach is the rectangle, in the paragraph's coordinates, that a draw can change; it is
// infinite for a draw that cannot be bounded or that spreads past its geometry.
func drawReach(draw ui.ParagraphDraw) ui.Rect {
	everything := ui.Rect{Left: math.Inf(-1), Top: math.Inf(-1), Right: math.Inf(1), Bottom: math.Inf(1)}
	if img, mask := gpu.PaintFilters(draw.Op); img || mask {
		return everything
	}
	var local ui.Rect
	if pic, ok := draw.Op.(ui.OpDrawPicture); ok {
		if pic.Picture == nil {
			return ui.Rect{}
		}
		local = pic.Picture.CullRect()
	} else if r, ok := gpu.ShapeBounds(draw.Op); ok {
		local = r
	} else {
		return everything
	}
	if draw.Matrix != nil {
		local = draw.Matrix.TransformRect(local)
	}
	if !local.IsFinite() {
		return everything
	}
	// Two pixels more, for the edge that is anti-aliased.
	return local.Shift(draw.Offset).Inflate(2)
}

// Layout does nothing: the paragraph stays as it was when it was detached.
func (*frozenParagraph) Layout(ui.ParagraphConstraints) {}

// Width is the width of the paragraph when it was detached.
func (f *frozenParagraph) Width() float64 { return f.width }

// Height is the height of the paragraph when it was detached.
func (f *frozenParagraph) Height() float64 { return f.height }

// PaintParagraph emits the drawing the paragraph made when it was detached.
func (f *frozenParagraph) PaintParagraph(emit func(ui.ParagraphDraw)) {
	for _, d := range f.draws {
		emit(d)
	}
}

// PaintParagraphRows emits the draws of the groups that reach the rows from top to bottom, and
// no others, in the order they were made.
func (f *frozenParagraph) PaintParagraphRows(top, bottom float64, emit func(ui.ParagraphDraw)) {
	start := 0
	for _, g := range f.groups {
		if g.bottom >= top && g.top <= bottom {
			for _, d := range f.draws[start:g.end] {
				emit(d)
			}
		}
		start = g.end
	}
}

// PaintBounds is the box of everything the paragraph draws.
func (f *frozenParagraph) PaintBounds() ui.Rect {
	if len(f.draws) == 0 {
		return ui.Rect{}
	}
	return f.bounds
}
