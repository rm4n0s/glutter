package svg

import (
	"math"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// maxImageDepth is how deep an <image> that holds an SVG document may nest (an SVG image whose own image is an SVG
// image, and so on).
const maxImageDepth = 3

// Picture records the document into a ui.Picture in the document's own size (Size): the root viewport transform
// (viewBox and preserveAspectRatio) is applied inside it, so whoever draws the picture scales it to a widget. The
// picture is not clipped to the document; the widget clips it (SPEC §26.1).
//
// It is PictureAt with a scale of 1.
//
// SVG 1.1: render.html
func (d *Document) Picture() *ui.Picture { return d.PictureAt(1) }

// PictureAt is Picture for a picture that will be drawn scale device pixels to one unit of the document. The scale
// matters only to what the picture rasterizes (the tiles of patterns are made with that many pixels, so that they stay
// sharp) and to vector-effect: non-scaling-stroke (a stroke is that many device pixels wide); everything else is
// vector and scales by itself. A scale that is not positive is 1.
//
// SVG 1.1: pservers.html#Patterns; SVG 2: coords.html#VectorEffectProperty
func (d *Document) PictureAt(scale float64) *ui.Picture {
	if !(scale > 0) || math.IsInf(scale, 0) {
		scale = 1
	}
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	p := newPainter(c, scale)
	if d != nil && d.Root != nil {
		p.drawDocument(d)
	}
	return rec.EndRecording()
}

// painter records the nodes of a tree on a canvas.
type painter struct {
	c *ui.Canvas
	// scale is the number of device pixels to a unit of the picture.
	scale float64
	// ras makes the tiles of patterns.
	ras ui.Rasterizer
	// ctm is the transform of the canvas, from the user space of the node being drawn to the picture.
	ctm   Affine
	stack []Affine
	// vw and vh are the size of the viewport that percentages refer to, in user units.
	vw, vh float64
	// depth is how many SVG images deep the painter is.
	depth int
	// tiles keeps the tile images of the patterns that were made, by pattern and size.
	tiles map[tileKey]*ui.Image
}

func newPainter(c *ui.Canvas, scale float64) *painter {
	return &painter{c: c, scale: scale, ras: soft.ImageRasterizer{}, ctm: Identity, tiles: map[tileKey]*ui.Image{}}
}

func (p *painter) save() {
	p.c.Save()
	p.stack = append(p.stack, p.ctm)
}

func (p *painter) restore() {
	p.c.Restore()
	p.ctm = p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
}

// concat applies a transform to the canvas.
func (p *painter) concat(a Affine) {
	if a.IsIdentity() {
		return
	}
	p.c.Transform(a.Matrix4())
	p.ctm = p.ctm.Mul(a)
}

// drawDocument draws a document: the root node, whose viewport transform places the viewBox in the document's size.
func (p *painter) drawDocument(d *Document) {
	vp := d.Root.Viewport
	oldW, oldH := p.vw, p.vh
	p.vw, p.vh = vp.Width, vp.Height
	if vp.ViewBox != nil {
		p.vw, p.vh = vp.ViewBox.Width, vp.ViewBox.Height
	}
	p.drawNode(d.Root, 1)
	p.vw, p.vh = oldW, oldH
}

// ViewBoxTransform returns the transform that places a viewBox in a viewport of w x h, as preserveAspectRatio says:
// none stretches it, meet scales it to fit inside, and slice to cover the viewport, and the alignment places the
// viewBox in the room that is left.
//
// SVG 1.1: coords.html#PreserveAspectRatioAttribute
func ViewBoxTransform(vb ViewBox, par PreserveAspectRatio, w, h float64) Affine {
	if !(vb.Width > 0 && vb.Height > 0) {
		return Identity
	}
	sx, sy := w/vb.Width, h/vb.Height
	if par.Align == AlignNone {
		return Affine{A: sx, D: sy, E: -vb.X * sx, F: -vb.Y * sy}
	}
	s := math.Min(sx, sy)
	if par.Slice {
		s = math.Max(sx, sy)
	}
	ax, ay := alignFractions(par.Align)
	return Affine{
		A: s, D: s,
		E: -vb.X*s + ax*(w-vb.Width*s),
		F: -vb.Y*s + ay*(h-vb.Height*s),
	}
}

// alignFractions returns where the alignment places the content in the room that is left: 0 for min, 0.5 for mid and 1
// for max, on x and on y.
func alignFractions(a Align) (ax, ay float64) {
	switch a {
	case AlignXMinYMin:
		return 0, 0
	case AlignXMidYMin:
		return 0.5, 0
	case AlignXMaxYMin:
		return 1, 0
	case AlignXMinYMid:
		return 0, 0.5
	case AlignXMaxYMid:
		return 1, 0.5
	case AlignXMinYMax:
		return 0, 1
	case AlignXMidYMax:
		return 0.5, 1
	case AlignXMaxYMax:
		return 1, 1
	}
	return 0.5, 0.5
}

// innerTransform returns the transform from the content of a viewport to the user space around it, for the part the
// painter applies after Node.Transform: the position of a nested viewport and its viewBox.
func innerTransform(vp *Viewport) Affine {
	m := Identity
	if !vp.Root {
		m = Translate(vp.X, vp.Y)
	}
	if vp.ViewBox != nil {
		m = m.Mul(ViewBoxTransform(*vp.ViewBox, vp.PreserveAspectRatio, vp.Width, vp.Height))
	}
	return m
}

// drawNode draws a node and its descendants. alpha is the opacity that the parents handed down (they fold their
// opacity into the paints of a single shape instead of making a layer).
//
// SVG 1.1: render.html#RenderingOrder
func (p *painter) drawNode(n *Node, alpha float64) {
	a := alpha * n.Opacity
	if !(a > 0) {
		return
	}
	p.save()
	defer p.restore()
	p.concat(n.Transform)
	if n.ClipPath != nil && !p.applyClip(n.ClipPath, n) {
		return
	}
	// 29.3: n.Mask and n.Filters are applied here, to the layer that holds the content.
	layer := false
	if a < 1 && !foldable(n) {
		p.c.SaveLayer(nil, alphaPaint(a))
		layer, a = true, 1
	}
	switch n.Kind {
	case NodeShape:
		p.drawShape(n, a)
	case NodeImage:
		p.drawImage(n, a)
	case NodeText:
		// 29.3: text.
	case NodeSvg:
		p.drawViewport(n, a)
	default:
		for _, c := range n.Children {
			p.drawNode(c, a)
		}
	}
	if layer {
		p.c.Restore()
	}
}

// drawViewport draws the content of a node that is a viewport: the root svg, a nested svg, or a symbol.
//
// SVG 1.1: coords.html#EstablishingANewSVGViewport
func (p *painter) drawViewport(n *Node, a float64) {
	vp := n.Viewport
	if vp == nil {
		return
	}
	if !vp.Root {
		p.concat(Translate(vp.X, vp.Y))
		if !vp.Overflow && vp.Width > 0 && vp.Height > 0 {
			p.c.ClipRect(ui.NewRectFromLTWH(0, 0, vp.Width, vp.Height))
		}
	}
	oldW, oldH := p.vw, p.vh
	if vp.ViewBox != nil {
		p.concat(ViewBoxTransform(*vp.ViewBox, vp.PreserveAspectRatio, vp.Width, vp.Height))
		p.vw, p.vh = vp.ViewBox.Width, vp.ViewBox.Height
	} else if vp.Width > 0 && vp.Height > 0 {
		p.vw, p.vh = vp.Width, vp.Height
	}
	for _, c := range n.Children {
		p.drawNode(c, a)
	}
	p.vw, p.vh = oldW, oldH
}

// foldable reports whether the opacity of a node can be multiplied into the alpha of the paints it draws, which
// is the same as drawing it into a layer and compositing the layer with that opacity: a shape that paints only its
// fill or only its stroke, or a group of one such shape. Anything that can overlap itself needs the layer.
func foldable(n *Node) bool {
	if n.Mask != nil || len(n.Filters) > 0 {
		return false
	}
	switch n.Kind {
	case NodeShape:
		st := n.Style
		f := paintedWith(st.Fill)
		s := paintedWith(st.Stroke) && st.StrokeWidth > 0
		return (!f || !s) && !isPatternPaint(st.Fill) && !isPatternPaint(st.Stroke)
	case NodeGroup:
		return len(n.Children) == 0 || len(n.Children) == 1 && n.Children[0].Opacity == 1 && foldable(n.Children[0])
	}
	return false
}

func paintedWith(pt Paint) bool { return pt.Kind != PaintKindNone }

func isPatternPaint(pt Paint) bool {
	if pt.Kind != PaintKindServer {
		return false
	}
	_, ok := pt.Server.(*Pattern)
	return ok
}

// alphaPaint is the paint of a layer that is composited with an opacity.
func alphaPaint(a float64) *ui.Paint {
	paint := ui.NewPaint()
	paint.SetColor(ui.NewColorFromARGB(alpha8(a), 0, 0, 0))
	return paint
}

// alpha8 converts an opacity to a channel value.
func alpha8(a float64) int {
	return int(math.Round(math.Max(0, math.Min(1, a)) * 255))
}

// bounds accumulates the object bounding box of a node (the fill geometry: no strokes, no text).
type bounds struct {
	r  ui.Rect
	ok bool
}

func (b *bounds) add(r ui.Rect) {
	if b.ok {
		b.r = b.r.ExpandToInclude(r)
		return
	}
	b.r, b.ok = r, true
}

// addNodeBounds adds the bounds of the content of a node, whose user space m maps to the target space.
func addNodeBounds(n *Node, m Affine, b *bounds) {
	switch n.Kind {
	case NodeShape:
		if n.Path == nil || n.Path.IsEmpty() {
			return
		}
		path := n.Path
		if !m.IsIdentity() {
			path = path.Transform(m.Matrix4())
		}
		b.add(path.GetTightBounds())
	case NodeImage:
		im := n.Image
		w, h := im.Width, im.Height
		if w <= 0 || h <= 0 {
			return
		}
		var corners [4][2]float64
		corners[0][0], corners[0][1] = m.Apply(im.X, im.Y)
		corners[1][0], corners[1][1] = m.Apply(im.X+w, im.Y)
		corners[2][0], corners[2][1] = m.Apply(im.X, im.Y+h)
		corners[3][0], corners[3][1] = m.Apply(im.X+w, im.Y+h)
		r := ui.NewRectFromLTRB(corners[0][0], corners[0][1], corners[0][0], corners[0][1])
		for _, c := range corners[1:] {
			r = r.ExpandToInclude(ui.NewRectFromLTRB(c[0], c[1], c[0], c[1]))
		}
		b.add(r)
	case NodeText:
		// 29.3: the bounds of text.
	default:
		inner := m
		if n.Kind == NodeSvg && n.Viewport != nil {
			inner = m.Mul(innerTransform(n.Viewport))
		}
		for _, c := range n.Children {
			addNodeBounds(c, inner.Mul(c.Transform), b)
		}
	}
}

// objectBBox returns the object bounding box of a node in its own user space (inside its transform), and whether it
// has an area: a paint server in objectBoundingBox units, and a clip path, are not drawn for a box with no width or
// height.
//
// SVG 1.1: coords.html#ObjectBoundingBox
func objectBBox(n *Node) (ui.Rect, bool) {
	var b bounds
	inner := Identity
	if n.Kind == NodeSvg && n.Viewport != nil {
		inner = innerTransform(n.Viewport)
	}
	if n.Kind == NodeShape || n.Kind == NodeImage {
		addNodeBounds(n, Identity, &b)
	} else {
		for _, c := range n.Children {
			addNodeBounds(c, inner.Mul(c.Transform), &b)
		}
	}
	return b.r, b.ok && b.r.Width() > 0 && b.r.Height() > 0
}
