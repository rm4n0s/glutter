package svg

import (
	"context"
	"math"
	"sort"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

// maxTextRunes is the most characters of one text element that are laid out; the rest is dropped. It keeps a document
// that holds a huge text from making the layout take time and memory without bound.
const maxTextRunes = 50_000

// textBlock is a text element laid out and placed: the outlines of its glyphs in the user space of the element, grouped
// by the span whose style paints them.
type textBlock struct {
	groups []*textGroup
	// outline is all the glyphs and decorations as one path (for a clip path).
	outline *ui.Path
	// cells is the bounding box of the text: the cells of its glyphs, from the ascent to the descent of their fonts. It
	// is the box that objectBoundingBox units refer to for text.
	cells bounds
	// items are the placed clusters, in the order of the text, for the tests.
	items []textItem
}

// textGroup is the glyphs and the decoration lines of one span, painted with the span's fill and stroke.
type textGroup struct {
	style *Style
	path  *ui.Path
	// pics are the color glyphs (emoji), which are pictures.
	pics []textPicture
}

type textPicture struct {
	pic *ui.Picture
	m   Affine
	// local is the transform from the font's units to the glyph's space.
	local *ui.Matrix4
}

// bundledMatcher is the font matcher of the engine that a document makes when the app gives none: it finds no system
// font, so that the bundled fonts alone draw the text, the same on every machine.
type bundledMatcher struct{}

func (bundledMatcher) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// defaultEngine returns the text engine that a document uses when the app gives none: the fonts that Glutter bundles.
// It is made on first use, and it is used one layout at a time.
func (d *Document) defaultEngine() *layout.Engine {
	d.engineOnce.Do(func() {
		c := text.NewCollection(text.CollectionOpts{Matcher: bundledMatcher{}})
		if err := fonts.Register(c); err != nil {
			return
		}
		d.engine = layout.NewEngine(context.Background(), c)
	})
	return d.engine
}

// textBlockOf returns the layout of a text node, made on first use and kept for the picture being recorded; nil if the
// text has nothing to draw or there is no text engine.
func (p *painter) textBlockOf(n *Node) *textBlock {
	if tb, ok := p.texts[n]; ok {
		return tb
	}
	tb := p.layoutText(n)
	p.texts[n] = tb
	return tb
}

// drawText draws a text node: for each span, the fill and the stroke of its glyphs and decorations.
//
// SVG 1.1: text.html#TextRenderingOrder
func (p *painter) drawText(n *Node, alpha float64) {
	tb := p.textBlockOf(n)
	if tb == nil {
		return
	}
	bb := &lazyBBox{measure: func() (ui.Rect, bool) {
		return tb.cells.r, tb.cells.ok && tb.cells.r.Width() > 0 && tb.cells.r.Height() > 0
	}}
	for _, g := range tb.groups {
		if !g.style.Visible {
			continue
		}
		if !g.path.IsEmpty() {
			p.paintGeometry(g.path, g.style, n.VectorEffect, bb, alpha, nil)
		}
		for _, pic := range g.pics {
			p.save()
			p.concat(pic.m)
			if pic.local != nil {
				p.c.Transform(pic.local)
			}
			p.c.DrawPicture(pic.pic)
			p.restore()
		}
	}
}

// flatText is a text element as a string with its runs and the attributes of each character.
type flatText struct {
	text string
	runs []flatRun
	// The attributes of each character (code point), NaN when not given.
	x, y, dx, dy, rot []float64
	// pathOf is the textPath that holds each character, or nil.
	pathOf []*TextSpan
	count  int
}

// flatRun is a string of one span.
type flatRun struct {
	span       *TextSpan
	start, end int // byte offsets
}

// flattenText walks the spans of a text element in order and collects its content.
func flattenText(root *TextSpan) *flatText {
	f := &flatText{}
	type entry struct {
		span       *TextSpan
		start, end int
		path       *TextSpan
	}
	var entries []entry
	var sb []byte
	var walk func(s *TextSpan, path *TextSpan)
	walk = func(s *TextSpan, path *TextSpan) {
		if s.Name == "textPath" && path == nil {
			path = s
		}
		ei := len(entries)
		entries = append(entries, entry{span: s, start: f.count, path: path})
		for _, it := range s.Items {
			if it.Span != nil {
				walk(it.Span, path)
				continue
			}
			t := it.Text
			room := maxTextRunes - f.count
			if room <= 0 {
				continue
			}
			if utf8.RuneCountInString(t) > room {
				i, k := 0, 0
				for i < len(t) && k < room {
					_, w := utf8.DecodeRuneInString(t[i:])
					i += w
					k++
				}
				t = t[:i]
			}
			start := len(sb)
			sb = append(sb, t...)
			f.runs = append(f.runs, flatRun{span: s, start: start, end: len(sb)})
			f.count += utf8.RuneCountInString(t)
		}
		entries[ei].end = f.count
	}
	walk(root, nil)
	f.text = string(sb)
	nan := math.NaN()
	mk := func() []float64 {
		a := make([]float64, f.count)
		for i := range a {
			a[i] = nan
		}
		return a
	}
	f.x, f.y, f.dx, f.dy, f.rot = mk(), mk(), mk(), mk(), mk()
	f.pathOf = make([]*TextSpan, f.count)
	for _, e := range entries {
		s := e.span
		for i := e.start; i < e.end; i++ {
			f.pathOf[i] = e.path
		}
		set := func(dst []float64, src []float64) {
			for j, v := range src {
				if e.start+j < e.end {
					dst[e.start+j] = v
				}
			}
		}
		set(f.x, s.X)
		set(f.y, s.Y)
		set(f.dx, s.Dx)
		set(f.dy, s.Dy)
		if len(s.Rotate) > 0 {
			for i := e.start; i < e.end; i++ {
				j := i - e.start
				if j < len(s.Rotate) {
					f.rot[i] = s.Rotate[j]
				} else {
					f.rot[i] = s.Rotate[len(s.Rotate)-1]
				}
			}
		}
	}
	return f
}

// textStyleOf makes the ui text style that shapes a span: the font, its size, weight and slant, and the spacing.
func textStyleOf(st *Style) *ui.TextStyle {
	size := st.FontSize
	ls, ws := st.LetterSpacing, st.WordSpacing
	weight := ui.FontWeight(min(max(st.FontWeight, 1), 1000))
	fs := ui.FontStyleNormal
	if st.FontStyle != FontStyleNormal {
		fs = ui.FontStyleItalic
	}
	o := ui.TextStyleOpts{
		FontSize: &size, FontWeight: weight, FontStyle: fs, LetterSpacing: &ls, WordSpacing: &ws,
		Color: ptr(ui.NewColorFromARGB(255, 0, 0, 0)),
	}
	// The families of the style, then sans-serif when it names none, and the bundled Roboto last, so that text is
	// drawn even with no font of the machine that fits.
	families := append([]string(nil), st.FontFamily...)
	if len(families) == 0 {
		families = append(families, "sans-serif")
	}
	families = append(families, fonts.RobotoFamily)
	o.FontFamily = families[0]
	o.FontFamilyFallback = families[1:]
	return ui.NewTextStyle(o)
}

func ptr[T any](v T) *T { return &v }

// textItem is a cluster with the place the layout of the text gives it.
type textItem struct {
	c    layout.Cluster
	span *TextSpan
	// g maps the cluster's own space (the origin is the start of its advance, on the baseline) to the user space; placed
	// is false for a cluster that is not drawn (a character of a textPath that is beyond the path).
	g      Affine
	placed bool
}

// layoutText lays out the text element of a node, places each character and makes its glyph outlines.
//
// Text is laid out as SVG does it, not by paragraph wrapping: the whole element is one line, shaped as one paragraph
// of the engine so that kerning and joining work across spans, and then the clusters are put where the x, y, dx, dy,
// rotate and text-anchor of their text chunk and the textPath say.
//
// SVG 1.1: text.html#TextLayout
func (p *painter) layoutText(n *Node) *textBlock {
	root := n.Text
	if root == nil {
		return nil
	}
	flat := flattenText(root)
	if flat.count == 0 {
		return nil
	}
	factory := p.text
	if factory == nil && p.doc != nil {
		if e := p.doc.defaultEngine(); e != nil {
			factory = e
			p.doc.engineMu.Lock()
			defer p.doc.engineMu.Unlock()
		}
	}
	if factory == nil {
		return nil
	}
	pb := ui.NewParagraphBuilder(factory, ui.NewParagraphStyle())
	for _, r := range flat.runs {
		pb.PushStyle(textStyleOf(r.span.Style))
		pb.AddText(flat.text[r.start:r.end])
		pb.Pop()
	}
	para := pb.Build()
	para.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	cl, ok := para.Layouter().(layout.Clusterer)
	if !ok {
		return nil
	}
	clusters := cl.Clusters()
	if len(clusters) == 0 {
		return nil
	}
	// cpAt[b] is the index of the character that starts at byte b (valid at the start of a character).
	cpAt := make([]int32, len(flat.text)+1)
	k := int32(0)
	for b := range flat.text {
		cpAt[b] = k
		k++
	}
	cpAt[len(flat.text)] = k
	items := make([]textItem, len(clusters))
	runIdx := 0
	for i, c := range clusters {
		for runIdx+1 < len(flat.runs) && flat.runs[runIdx].end <= c.Start {
			runIdx++
		}
		items[i] = textItem{c: c, span: flat.runs[runIdx].span}
	}
	placeText(items, flat, cpAt)
	return buildTextBlock(items)
}

// placement is a cluster in a chunk, with the dx and dy that came before it in the chunk, summed, and its rotate.
type placement struct {
	item       int
	offX, offY float64
	rot        float64
	hasRot     bool
}

// textChunk is a run of characters that follow each other from one starting position, to which text-anchor applies; or
// the characters of a textPath.
type textChunk struct {
	sx, sy float64
	anchor TextAnchor
	places []placement
	// path is the textPath that the characters follow, or nil.
	path *TextSpan
}

// extent returns the left edge of the chunk's clusters in the layout and the width of the chunk: the clusters' advances
// with the dx that moved them.
func (ch *textChunk) extent(items []textItem) (minX0, width float64) {
	minX0 = math.Inf(1)
	for _, pl := range ch.places {
		minX0 = math.Min(minX0, items[pl.item].c.X0)
	}
	end := 0.0
	for _, pl := range ch.places {
		c := items[pl.item].c
		end = math.Max(end, (c.X0-minX0)+pl.offX+(c.X1-c.X0))
	}
	return minX0, end
}

// anchorShift is the shift that text-anchor makes of a chunk of the width w.
func anchorShift(a TextAnchor, w float64) float64 {
	switch a {
	case TextAnchorMiddle:
		return -w / 2
	case TextAnchorEnd:
		return -w
	}
	return 0
}

// placeText gives each cluster its transform: the chunks of text, positioned by their x and y and moved by the anchor,
// and the text on paths.
//
// SVG 1.1: text.html#TextLayoutIntroduction
func placeText(items []textItem, flat *flatText, cpAt []int32) {
	var chunks []*textChunk
	var cur *textChunk
	curX, curY := 0.0, 0.0
	var accDx, accDy float64
	for i := range items {
		c := items[i].c
		cp0, cp1 := int(cpAt[c.Start]), int(cpAt[min(c.End, len(cpAt)-1)])
		cp1 = max(cp1, cp0+1)
		x, y := math.NaN(), math.NaN()
		var dx, dy, rot float64
		hasRot := false
		for cp := cp0; cp < cp1 && cp < flat.count; cp++ {
			if math.IsNaN(x) && !math.IsNaN(flat.x[cp]) {
				x = flat.x[cp]
			}
			if math.IsNaN(y) && !math.IsNaN(flat.y[cp]) {
				y = flat.y[cp]
			}
			if !math.IsNaN(flat.dx[cp]) {
				dx += flat.dx[cp]
			}
			if !math.IsNaN(flat.dy[cp]) {
				dy += flat.dy[cp]
			}
			if cp == cp0 && !math.IsNaN(flat.rot[cp]) {
				rot, hasRot = flat.rot[cp], true
			}
		}
		var path *TextSpan
		if cp0 < len(flat.pathOf) {
			path = flat.pathOf[cp0]
		}
		newChunk := cur == nil || cur.path != path || (path == nil && (!math.IsNaN(x) || !math.IsNaN(y)))
		if newChunk {
			if cur != nil && cur.path == nil {
				curX, curY = cur.end(items)
			}
			cur = &textChunk{path: path, anchor: items[i].span.Style.TextAnchor, sx: curX, sy: curY}
			if !math.IsNaN(x) {
				cur.sx = x
			}
			if !math.IsNaN(y) {
				cur.sy = y
			}
			chunks = append(chunks, cur)
			accDx, accDy = 0, 0
		}
		accDx += dx
		accDy += dy
		cur.places = append(cur.places, placement{item: i, offX: accDx, offY: accDy, rot: rot, hasRot: hasRot})
	}
	for _, ch := range chunks {
		if ch.path != nil {
			ch.layoutOnPath(items)
			continue
		}
		minX0, w := ch.extent(items)
		shift := anchorShift(ch.anchor, w)
		for _, pl := range ch.places {
			it := &items[pl.item]
			it.g = Translate(ch.sx+(it.c.X0-minX0)+pl.offX+shift, ch.sy+pl.offY)
			if pl.hasRot {
				it.g = it.g.Mul(Rotate(pl.rot))
			}
			it.placed = true
		}
	}
}

// end returns the current text position after the chunk: where a character with no position of its own goes next.
func (ch *textChunk) end(items []textItem) (float64, float64) {
	if len(ch.places) == 0 {
		return ch.sx, ch.sy
	}
	_, w := ch.extent(items)
	return ch.sx + w, ch.sy + ch.places[len(ch.places)-1].offY
}

// layoutOnPath places the characters of a textPath along its path: the middle of each character's advance is on the
// path, with the baseline on it and the glyph turned to the direction of the path there. A character whose middle is
// beyond the ends of the path is not drawn. startOffset moves the start of the text along the path, and the anchor
// shifts the text by its width; dx moves a character along the path and dy off it.
//
// SVG 1.1: text.html#TextpathElement
func (ch *textChunk) layoutOnPath(items []textItem) {
	span := ch.path
	if span == nil || span.PathData == nil {
		return
	}
	metrics := span.PathData.ComputeMetrics()
	cum := make([]float64, len(metrics)+1)
	for i, m := range metrics {
		cum[i+1] = cum[i] + m.Length
	}
	total := cum[len(metrics)]
	if !(total > 0) {
		return
	}
	start := span.StartOffset
	if span.StartOffsetFraction {
		start *= total
	}
	minX0, w := ch.extent(items)
	start += anchorShift(ch.anchor, w)
	for _, pl := range ch.places {
		it := &items[pl.item]
		adv := it.c.X1 - it.c.X0
		mid := start + (it.c.X0 - minX0) + pl.offX + adv/2
		if mid < 0 || mid > total {
			continue
		}
		ci := sort.Search(len(metrics), func(k int) bool { return cum[k+1] >= mid })
		if ci >= len(metrics) {
			ci = len(metrics) - 1
		}
		tan, ok := metrics[ci].GetTangentForOffset(mid - cum[ci])
		if !ok {
			continue
		}
		angle := math.Atan2(tan.Vector.Dy, tan.Vector.Dx) * 180 / math.Pi
		g := Translate(tan.Position.Dx, tan.Position.Dy).Mul(Rotate(angle))
		if pl.hasRot {
			g = g.Mul(Rotate(pl.rot))
		}
		it.g = g.Mul(Translate(-adv/2, pl.offY))
		it.placed = true
	}
}

// buildTextBlock makes the outlines of the placed clusters: the glyphs and the decoration lines of each span in one path,
// the bounds of the text, and the outline of everything.
func buildTextBlock(items []textItem) *textBlock {
	tb := &textBlock{outline: ui.NewPath(), items: items}
	groups := map[*TextSpan]*textGroup{}
	for i := range items {
		it := &items[i]
		if !it.placed {
			continue
		}
		c := it.c
		grp := groups[it.span]
		if grp == nil {
			grp = &textGroup{style: it.span.Style, path: ui.NewPath()}
			groups[it.span] = grp
			tb.groups = append(tb.groups, grp)
		}
		adv := c.X1 - c.X0
		tb.cells.addTransformed(ui.NewRectFromLTRB(0, -c.Ascent, adv, c.Descent), it.g)
		for _, gl := range c.Glyphs {
			m := it.g.Mul(Translate(gl.Origin.Dx-c.X0, gl.Origin.Dy-c.Baseline))
			switch {
			case gl.Path != nil:
				grp.path.AddPath(gl.Path.Transform(m.Matrix4()), ui.Offset{})
			case gl.Picture != nil:
				grp.pics = append(grp.pics, textPicture{pic: gl.Picture, m: m, local: gl.Matrix})
			}
		}
		if dec := it.span.Style.TextDecoration; dec != 0 && adv > 0 {
			line := func(y, th float64) {
				r := ui.NewRectFromLTRB(0, y-th/2, adv, y+th/2)
				rp := ui.NewPath()
				rp.AddRect(r)
				grp.path.AddPath(rp.Transform(it.g.Matrix4()), ui.Offset{})
			}
			if dec&TextDecorationUnderline != 0 {
				line(c.Underline, c.UnderlineThickness)
			}
			if dec&TextDecorationOverline != 0 {
				line(c.Overline, c.UnderlineThickness)
			}
			if dec&TextDecorationLineThrough != 0 {
				line(c.Strike, c.StrikeThickness)
			}
		}
	}
	for _, g := range tb.groups {
		tb.outline.AddPath(g.path, ui.Offset{})
	}
	if len(tb.groups) == 0 {
		return nil
	}
	return tb
}
