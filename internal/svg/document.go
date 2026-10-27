package svg

import (
	"math"
	"strings"
	"sync"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/ui"
)

// ParseOpts are the optional arguments of Parse.
type ParseOpts struct {
	// Theme gives currentColor, the font size and the x-height; the zero value is flutter_svg's default theme.
	Theme Theme
	// Locale picks the child of a switch by its systemLanguage; the zero value is "en".
	Locale ui.Locale
}

// Document is a parsed SVG document: a tree of nodes for one Theme. It is never changed after Parse returns, so it
// can be drawn from several goroutines.
//
// SVG 1.1: struct.html#NewDocument
type Document struct {
	// Root is the root svg element, a NodeSvg whose Viewport has the size of the document.
	Root *Node
	// Theme is the theme that the document was parsed with, with the defaults filled in.
	Theme Theme
	// Title and Desc are the text of the first title and desc elements that are children of the root; the semantics
	// label of the picture when the app gives none.
	Title, Desc string

	// scaleDependent is Parse's answer for ScaleDependent.
	scaleDependent bool

	// engine is the text engine of PictureAt, made on first use (defaultEngine), and engineMu orders its users.
	engineOnce sync.Once
	engine     *layout.Engine
	engineMu   sync.Mutex
}

// Parse reads an SVG document (XML, optionally gzip-compressed) and builds its tree. The errors are SvgParseFailed
// (the XML is not well formed), SvgNotSvg (the root is not an svg element) and SvgInvalidValue (a viewBox with a
// negative size); anything else that is wrong in the document is ignored, as browsers do.
//
// SVG 1.1: struct.html#NewDocument
func Parse(data []byte, opts ...ParseOpts) (*Document, error) {
	var o ParseOpts
	switch len(opts) {
	case 0:
	case 1:
		o = opts[0]
	default:
		panic("svg.Parse: more than one ParseOpts")
	}
	root, err := readXML(data)
	if err != nil {
		return nil, err
	}
	if root.name != "svg" {
		return nil, errors.New(TagSvgNotSvg, "the root element is not svg", "element", root.name)
	}
	b := &builder{
		theme:     o.Theme.withDefaults(),
		langs:     languagesOf(o.Locale),
		root:      root,
		ids:       map[string]*element{},
		sheet:     &stylesheet{},
		styleMemo: map[*element]*Style{},
		res:       map[*element]any{},
		busy:      map[*element]bool{},
	}
	b.index(root)
	doc := &Document{Theme: b.theme}
	doc.Title, doc.Desc = rootTexts(root)

	st, np := b.computeStyle(root, b.sheet.cascade(root), b.initialStyle(), &buildCtx{})
	b.styleMemo[root] = st
	lc := b.lenFor(st.FontSize, &buildCtx{})
	vp := &Viewport{Root: true, Overflow: true}
	if v, ok := root.attr("viewBox"); ok {
		vb, good, err := parseViewBox(v)
		if err != nil {
			return nil, err
		}
		if good {
			vp.ViewBox = &vb
		}
	}
	if v, ok := root.attr("preserveAspectRatio"); ok {
		if par, ok := ParsePreserveAspectRatio(v); ok {
			vp.PreserveAspectRatio = par
		}
	}
	// width and height count only if they are not percentages (those refer to the container, which is the widget's
	// business).
	var w, h float64
	var hasW, hasH bool
	if l, ok := attrLength(root, "width"); ok && l.Unit != UnitPercent {
		if px, _ := lc.px(l, axisX); px >= 0 {
			w, hasW = px, true
		}
	}
	if l, ok := attrLength(root, "height"); ok && l.Unit != UnitPercent {
		if px, _ := lc.px(l, axisY); px >= 0 {
			h, hasH = px, true
		}
	}
	ctx := buildCtx{}
	switch {
	case vp.ViewBox != nil:
		ctx.vw, ctx.vh, ctx.vpKnown = vp.ViewBox.Width, vp.ViewBox.Height, true
	case hasW && hasH:
		ctx.vw, ctx.vh, ctx.vpKnown = w, h, true
	}
	b.rootCtx = ctx
	// Percentages of the root's own lengths were left out above; the style is computed again for the content with the
	// viewport known.
	st, np = b.computeStyle(root, b.sheet.cascade(root), b.initialStyle(), &ctx)
	b.styleMemo[root] = st
	n := &Node{
		Kind:         NodeSvg,
		Name:         "svg",
		ID:           root.id,
		Transform:    np.transform,
		Style:        st,
		Opacity:      np.opacity,
		VectorEffect: np.vectorEffect,
		Viewport:     vp,
	}
	if np.clipPath != "" {
		n.ClipPath = b.clipPath(np.clipPath)
	}
	if np.mask != "" {
		n.Mask = b.mask(np.mask)
	}
	for _, id := range np.filters {
		if f := b.filter(id); f != nil {
			n.Filters = append(n.Filters, f)
		}
	}
	if np.display && b.conditionsHold(root) && !(vp.ViewBox != nil && (vp.ViewBox.Width == 0 || vp.ViewBox.Height == 0)) {
		n.Children = b.buildChildren(root, st, &ctx)
	}
	if b.err != nil {
		return nil, b.err
	}
	doc.Root = n
	doc.scaleDependent = b.scaleDependent
	size := doc.intrinsicSize(w, h, hasW, hasH)
	vp.Width, vp.Height = size.Width, size.Height
	return doc, nil
}

// index fills the id table (the first element with an id wins) and the style sheet.
func (b *builder) index(e *element) {
	if e.id != "" && b.ids[e.id] == nil {
		b.ids[e.id] = e
	}
	if e.name == "style" {
		if t := strings.TrimSpace(e.attrOr("type", "")); t == "" || t == "text/css" {
			b.sheet.add(e.textContent())
		}
	}
	for _, c := range e.children {
		if c.name != "#text" {
			b.index(c)
		}
	}
}

func attrLength(e *element, name string) (Length, bool) {
	v, ok := e.attr(name)
	if !ok {
		return Length{}, false
	}
	return ParseLength(v)
}

// rootTexts returns the text of the first title and desc that are children of the root.
func rootTexts(root *element) (title, desc string) {
	gotT, gotD := false, false
	for _, c := range root.children {
		switch {
		case c.name == "title" && !gotT:
			title, gotT = strings.TrimSpace(c.textContent()), true
		case c.name == "desc" && !gotD:
			desc, gotD = strings.TrimSpace(c.textContent()), true
		}
	}
	return title, desc
}

// Size returns the size of the document in user units: the picture's intrinsic size, as flutter_svg takes it from
// the width and height of the root or, without them, from the viewBox. It is zero if the document draws nothing
// that has a size.
//
// A width or height that is a percentage counts as absent. A missing one comes from the viewBox with its aspect
// ratio; with no viewBox it comes from the bounds of the shapes (without strokes and text).
//
// SVG 1.1: coords.html#IntrinsicSizing; SVG 2: coords.html#SizingSVGInCSS
func (d *Document) Size() ui.Size {
	return ui.Size{Width: d.Root.Viewport.Width, Height: d.Root.Viewport.Height}
}

// ViewBox returns the viewBox of the root, or false if there is none.
func (d *Document) ViewBox() (ViewBox, bool) {
	if d.Root.Viewport.ViewBox == nil {
		return ViewBox{}, false
	}
	return *d.Root.Viewport.ViewBox, true
}

// intrinsicSize works out Size from the root's width and height (when absolute), the viewBox and the content.
func (d *Document) intrinsicSize(w, h float64, hasW, hasH bool) ui.Size {
	vb := d.Root.Viewport.ViewBox
	switch {
	case hasW && hasH:
		return ui.Size{Width: w, Height: h}
	case vb != nil && vb.Width > 0 && vb.Height > 0:
		switch {
		case hasW:
			return ui.Size{Width: w, Height: w * vb.Height / vb.Width}
		case hasH:
			return ui.Size{Width: h * vb.Width / vb.Height, Height: h}
		}
		return ui.Size{Width: vb.Width, Height: vb.Height}
	case vb != nil:
		return ui.Size{Width: w, Height: h}
	}
	bounds := d.ContentBounds()
	if !hasW {
		w = math.Max(0, bounds.Right)
	}
	if !hasH {
		h = math.Max(0, bounds.Bottom)
	}
	return ui.Size{Width: w, Height: h}
}

// ContentBounds returns the bounds of the shapes of the document in the user space of the root (the fill geometry:
// no strokes, no text, no images). It is the empty rectangle for a document with no shape.
func (d *Document) ContentBounds() ui.Rect {
	var r ui.Rect
	first := true
	var walk func(n *Node, m Affine)
	walk = func(n *Node, m Affine) {
		m = m.Mul(n.Transform)
		if n.Kind == NodeShape && n.Path != nil {
			bb := n.Path.GetBounds()
			x0, y0 := m.Apply(bb.Left, bb.Top)
			x1, y1 := m.Apply(bb.Right, bb.Top)
			x2, y2 := m.Apply(bb.Left, bb.Bottom)
			x3, y3 := m.Apply(bb.Right, bb.Bottom)
			t := ui.NewRectFromLTRB(math.Min(math.Min(x0, x1), math.Min(x2, x3)), math.Min(math.Min(y0, y1), math.Min(y2, y3)),
				math.Max(math.Max(x0, x1), math.Max(x2, x3)), math.Max(math.Max(y0, y1), math.Max(y2, y3)))
			if first {
				r, first = t, false
			} else {
				r = r.ExpandToInclude(t)
			}
		}
		for _, c := range n.Children {
			walk(c, m)
		}
	}
	walk(d.Root, Identity)
	return r
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagSvgNotSvg is the tag of the error for a document whose root element is not svg; the metadata is "element".
	TagSvgNotSvg = "SvgNotSvg"
)
