package svg

import (
	"encoding/base64"
	"math"
	"net/url"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// maxUseDepth is how deep use elements may nest, and maxUseNodes how many nodes the expansion of use elements may
// make in a document (a chain of use elements that each use the previous one twice doubles at each level).
const (
	maxUseDepth = 32
	maxUseNodes = 250_000
)

// buildCtx is what changes while the builder goes down the tree.
type buildCtx struct {
	// vw and vh are the size of the viewport that percentages refer to; vpKnown is false if there is none.
	vw, vh  float64
	vpKnown bool
	// inClip is true inside a clipPath: only shapes, text and uses of them are built.
	inClip bool
	// inUse is true inside the expansion of a use, whose nodes count against maxUseNodes.
	inUse bool
}

// builder makes the tree from the element tree.
type builder struct {
	theme Theme
	langs []string
	root  *element
	ids   map[string]*element
	sheet *stylesheet
	// styleMemo is the memo of styleOf.
	styleMemo map[*element]*Style
	rootCtx   buildCtx
	// err is the first error that building found (an invalid viewBox); Parse returns it.
	err error
	// resources made so far, by element; busy marks the ones being made (a reference to one of them is a cycle).
	res      map[*element]any
	busy     map[*element]bool
	useStack []*element
	useNodes int
}

// languagesOf returns the language tag of the locale, in lower case ("en" for the zero locale). A tag in
// systemLanguage matches if it is the user's, if it is a prefix of the user's ("en" for "en-US"), or, when the user
// names no region, if the user's language is a prefix of it ("en-GB" for "en").
//
// SVG 1.1: struct.html#SystemLanguageAttribute
func languagesOf(l ui.Locale) []string {
	if l.LanguageCode == "" {
		return []string{"en"}
	}
	tag := strings.ToLower(l.LanguageCode)
	if l.CountryCode != "" {
		tag += "-" + strings.ToLower(l.CountryCode)
	}
	return []string{tag}
}

// conditionsHold evaluates requiredFeatures, requiredExtensions and systemLanguage.
//
// SVG 1.1: struct.html#ConditionalProcessing
func (b *builder) conditionsHold(e *element) bool {
	if v, ok := e.attr("requiredFeatures"); ok && strings.TrimSpace(v) == "" {
		return false
	}
	if _, ok := e.attr("requiredExtensions"); ok {
		return false
	}
	if v, ok := e.attr("systemLanguage"); ok {
		match := false
		for _, tag := range strings.Split(v, ",") {
			tag = strings.ToLower(strings.TrimSpace(tag))
			if tag == "" {
				continue
			}
			for _, user := range b.langs {
				if tag == user || strings.HasPrefix(user, tag+"-") || (!strings.Contains(user, "-") && strings.HasPrefix(tag, user+"-")) {
					match = true
				}
			}
		}
		return match
	}
	return true
}

// lengthAttr reads a length attribute of the element and resolves it to user units. It returns def if the
// attribute is missing or invalid.
func lengthAttr(e *element, name string, lc *lenCtx, a axis, def float64) float64 {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	l, ok := ParseLength(v)
	if !ok {
		return def
	}
	px, ok := lc.px(l, a)
	if !ok {
		return def
	}
	return px
}

// optLengthAttr is lengthAttr that tells whether the attribute gave a value.
func optLengthAttr(e *element, name string, lc *lenCtx, a axis) (float64, bool) {
	v, ok := e.attr(name)
	if !ok {
		return 0, false
	}
	l, ok := ParseLength(v)
	if !ok {
		return 0, false
	}
	return lc.px(l, a)
}

// buildChildren builds the children of an element, in order, under the given style.
func (b *builder) buildChildren(e *element, st *Style, ctx *buildCtx) []*Node {
	var out []*Node
	for _, c := range e.children {
		if c.name == "#text" {
			continue
		}
		if n := b.buildElement(c, st, ctx); n != nil {
			out = append(out, n)
		}
	}
	return out
}

func isShapeName(n string) bool {
	switch n {
	case "path", "rect", "circle", "ellipse", "line", "polyline", "polygon":
		return true
	}
	return false
}

func isGraphicOrContainer(n string) bool {
	return isShapeName(n) || n == "g" || n == "svg" || n == "a" || n == "switch" || n == "use" || n == "text" || n == "image"
}

// buildElement makes the node of an element and of its children, or nil if the element draws nothing.
func (b *builder) buildElement(e *element, parent *Style, ctx *buildCtx) *Node {
	if !isGraphicOrContainer(e.name) {
		return nil
	}
	if ctx.inClip && !isShapeName(e.name) && e.name != "text" && e.name != "use" {
		return nil
	}
	if !b.conditionsHold(e) {
		return nil
	}
	st, np := b.computeStyle(e, b.sheet.cascade(e), parent, ctx)
	if !np.display {
		return nil
	}
	if ctx.inUse {
		b.useNodes++
		if b.useNodes > maxUseNodes {
			return nil
		}
	}
	switch e.name {
	case "g", "a":
		n := b.newNode(NodeGroup, e, st, &np, ctx)
		n.Children = b.buildChildren(e, st, ctx)
		return n
	case "switch":
		n := b.newNode(NodeGroup, e, st, &np, ctx)
		for _, c := range e.children {
			if c.name == "#text" || !isGraphicOrContainer(c.name) || !b.conditionsHold(c) {
				continue
			}
			if child := b.buildElement(c, st, ctx); child != nil {
				n.Children = append(n.Children, child)
			}
			break
		}
		return n
	case "svg":
		return b.buildNestedSVG(e, st, &np, ctx, nil)
	case "use":
		return b.buildUse(e, st, &np, ctx)
	case "text":
		return b.buildText(e, st, &np, ctx)
	case "image":
		return b.buildImage(e, st, &np, ctx)
	}
	return b.buildShape(e, st, &np, ctx)
}

// newNode makes a node with the common fields and resolves its references.
func (b *builder) newNode(kind NodeKind, e *element, st *Style, np *nodeProps, ctx *buildCtx) *Node {
	n := &Node{
		Kind:         kind,
		Name:         e.name,
		ID:           e.id,
		Transform:    np.transform,
		Style:        st,
		Opacity:      np.opacity,
		VectorEffect: np.vectorEffect,
	}
	if ctx.inClip {
		// A clip path has no opacity, mask or filter.
		n.Opacity = 1
	}
	if np.clipPath != "" {
		n.ClipPath = b.clipPath(np.clipPath)
	}
	if !ctx.inClip {
		if np.mask != "" {
			n.Mask = b.mask(np.mask)
		}
		for _, id := range np.filters {
			if f := b.filter(id); f != nil {
				n.Filters = append(n.Filters, f)
			}
		}
	}
	return n
}

// buildShape makes the node of a shape element.
func (b *builder) buildShape(e *element, st *Style, np *nodeProps, ctx *buildCtx) *Node {
	lc := b.lenFor(st.FontSize, ctx)
	path, kind := shapePath(e, &lc)
	if path == nil {
		return nil
	}
	rule := st.FillRule
	if ctx.inClip {
		if !st.Visible {
			return nil
		}
		rule = st.ClipRule
	}
	if rule == FillRuleEvenOdd {
		path.SetFillType(ui.PathFillTypeEvenOdd)
	} else {
		path.SetFillType(ui.PathFillTypeNonZero)
	}
	n := b.newNode(NodeShape, e, st, np, ctx)
	n.Path = path
	n.Shape = kind
	return n
}

// shapePath makes the geometry of a shape element, or nil if the element is not rendered (an invalid size, or no
// segment).
//
// SVG 1.1: shapes.html; SVG 2: shapes.html
func shapePath(e *element, lc *lenCtx) (*ui.Path, ShapeKind) {
	switch e.name {
	case "path":
		d, ok := e.attr("d")
		if !ok {
			return nil, ShapePath
		}
		p, _ := ParsePathData(d)
		if p.IsEmpty() {
			return nil, ShapePath
		}
		return p, ShapePath
	case "rect":
		x := lengthAttr(e, "x", lc, axisX, 0)
		y := lengthAttr(e, "y", lc, axisY, 0)
		w := lengthAttr(e, "width", lc, axisX, 0)
		h := lengthAttr(e, "height", lc, axisY, 0)
		if !(w > 0 && h > 0) {
			return nil, ShapeRect
		}
		rx, hasRx := optLengthAttr(e, "rx", lc, axisX)
		ry, hasRy := optLengthAttr(e, "ry", lc, axisY)
		if hasRx && rx < 0 {
			hasRx = false
		}
		if hasRy && ry < 0 {
			hasRy = false
		}
		switch {
		case !hasRx && !hasRy:
			rx, ry = 0, 0
		case !hasRx:
			rx = ry
		case !hasRy:
			ry = rx
		}
		rx, ry = math.Min(rx, w/2), math.Min(ry, h/2)
		p := ui.NewPath()
		if rx <= 0 || ry <= 0 {
			p.MoveTo(x, y)
			p.LineTo(x+w, y)
			p.LineTo(x+w, y+h)
			p.LineTo(x, y+h)
			p.Close()
			return p, ShapeRect
		}
		cw := true
		arc := func(px, py float64) {
			p.ArcToPoint(ui.Offset{Dx: px, Dy: py}, ui.ArcToPointOpts{Radius: ui.Radius{X: rx, Y: ry}, Clockwise: &cw})
		}
		p.MoveTo(x+rx, y)
		p.LineTo(x+w-rx, y)
		arc(x+w, y+ry)
		p.LineTo(x+w, y+h-ry)
		arc(x+w-rx, y+h)
		p.LineTo(x+rx, y+h)
		arc(x, y+h-ry)
		p.LineTo(x, y+ry)
		arc(x+rx, y)
		p.Close()
		return p, ShapeRect
	case "circle":
		cx := lengthAttr(e, "cx", lc, axisX, 0)
		cy := lengthAttr(e, "cy", lc, axisY, 0)
		r := lengthAttr(e, "r", lc, axisXY, 0)
		if !(r > 0) {
			return nil, ShapeCircle
		}
		p := ui.NewPath()
		p.AddOval(ui.NewRectFromLTRB(cx-r, cy-r, cx+r, cy+r))
		return p, ShapeCircle
	case "ellipse":
		cx := lengthAttr(e, "cx", lc, axisX, 0)
		cy := lengthAttr(e, "cy", lc, axisY, 0)
		rx, hasRx := optLengthAttr(e, "rx", lc, axisX)
		ry, hasRy := optLengthAttr(e, "ry", lc, axisY)
		if hasRx && rx < 0 {
			hasRx = false
		}
		if hasRy && ry < 0 {
			hasRy = false
		}
		switch {
		case !hasRx && !hasRy:
			return nil, ShapeEllipse
		case !hasRx:
			rx = ry
		case !hasRy:
			ry = rx
		}
		if !(rx > 0 && ry > 0) {
			return nil, ShapeEllipse
		}
		p := ui.NewPath()
		p.AddOval(ui.NewRectFromLTRB(cx-rx, cy-ry, cx+rx, cy+ry))
		return p, ShapeEllipse
	case "line":
		p := ui.NewPath()
		p.MoveTo(lengthAttr(e, "x1", lc, axisX, 0), lengthAttr(e, "y1", lc, axisY, 0))
		p.LineTo(lengthAttr(e, "x2", lc, axisX, 0), lengthAttr(e, "y2", lc, axisY, 0))
		return p, ShapeLine
	case "polyline", "polygon":
		kind := ShapePolyline
		if e.name == "polygon" {
			kind = ShapePolygon
		}
		pts, _ := e.attr("points")
		nums := parsePoints(pts)
		if len(nums) < 4 {
			return nil, kind
		}
		p := ui.NewPath()
		p.MoveTo(nums[0], nums[1])
		for i := 2; i+1 < len(nums); i += 2 {
			p.LineTo(nums[i], nums[i+1])
		}
		if kind == ShapePolygon {
			p.Close()
		}
		return p, kind
	}
	return nil, ShapePath
}

// parsePoints reads a points list, up to the first error and without a last unpaired number, as the specification
// says.
//
// SVG 1.1: shapes.html#PointsBNF
func parsePoints(s string) []float64 {
	var out []float64
	i := skipSpace(s, 0)
	for i < len(s) {
		v, j, ok := scanNumber(s, i)
		if !ok {
			break
		}
		out = append(out, v)
		i = skipCommaSpace(s, j)
	}
	if len(out)%2 == 1 {
		out = out[:len(out)-1]
	}
	return out
}

// viewportOf reads the viewport attributes of an svg or a symbol: x, y, width, height, viewBox,
// preserveAspectRatio. The width and the height are 100% unless the use gives them.
func (b *builder) viewportOf(e *element, np *nodeProps, lc *lenCtx, ctx *buildCtx, w, h *float64) (*Viewport, error) {
	vp := &Viewport{Overflow: np.overflowSet && np.overflow}
	vp.X = lengthAttr(e, "x", lc, axisX, 0)
	vp.Y = lengthAttr(e, "y", lc, axisY, 0)
	vp.Width = lengthAttr(e, "width", lc, axisX, ctx.vw)
	vp.Height = lengthAttr(e, "height", lc, axisY, ctx.vh)
	if !ctx.vpKnown {
		if _, ok := optLengthAttr(e, "width", lc, axisX); !ok {
			vp.Width = 0
		}
		if _, ok := optLengthAttr(e, "height", lc, axisY); !ok {
			vp.Height = 0
		}
	}
	if w != nil {
		vp.Width = *w
	}
	if h != nil {
		vp.Height = *h
	}
	if v, ok := e.attr("viewBox"); ok {
		vb, good, err := parseViewBox(v)
		if err != nil {
			return nil, err
		}
		if good {
			vp.ViewBox = &vb
		}
	}
	if v, ok := e.attr("preserveAspectRatio"); ok {
		if par, ok := ParsePreserveAspectRatio(v); ok {
			vp.PreserveAspectRatio = par
		}
	}
	return vp, nil
}

// innerCtx is the context for the content of a viewport.
func innerCtx(vp *Viewport, ctx *buildCtx) buildCtx {
	in := *ctx
	switch {
	case vp.ViewBox != nil:
		in.vw, in.vh, in.vpKnown = vp.ViewBox.Width, vp.ViewBox.Height, true
	case vp.Width > 0 && vp.Height > 0:
		in.vw, in.vh, in.vpKnown = vp.Width, vp.Height, true
	}
	return in
}

// buildNestedSVG makes the node of an svg element that is not the root, or of a symbol that a use instantiates (w
// and h are the use's width and height, or nil).
//
// SVG 1.1: struct.html#NewDocument, struct.html#SymbolElement
func (b *builder) buildNestedSVG(e *element, st *Style, np *nodeProps, ctx *buildCtx, size *[2]*float64) *Node {
	lc := b.lenFor(st.FontSize, ctx)
	var w, h *float64
	if size != nil {
		w, h = size[0], size[1]
	}
	vp, err := b.viewportOf(e, np, &lc, ctx, w, h)
	if err != nil {
		b.err = err
		return nil
	}
	if vp.ViewBox != nil && (vp.ViewBox.Width == 0 || vp.ViewBox.Height == 0) {
		return nil
	}
	if vp.Width <= 0 && ctx.vpKnown || vp.Height <= 0 && ctx.vpKnown {
		return nil
	}
	n := b.newNode(NodeSvg, e, st, np, ctx)
	n.Viewport = vp
	in := innerCtx(vp, ctx)
	n.Children = b.buildChildren(e, st, &in)
	return n
}

// buildUse expands a use element.
//
// SVG 1.1: struct.html#UseElement
func (b *builder) buildUse(e *element, st *Style, np *nodeProps, ctx *buildCtx) *Node {
	href, ok := e.href()
	if !ok {
		return nil
	}
	id, ok := hrefID(href)
	if !ok {
		return nil
	}
	target := b.ids[id]
	if target == nil || target == e {
		return nil
	}
	// A use that refers to one of its own ancestors, or to an element that is being expanded, is a cycle.
	for p := e.parent; p != nil; p = p.parent {
		if p == target {
			return nil
		}
	}
	for _, u := range b.useStack {
		if u == target {
			return nil
		}
	}
	if len(b.useStack) >= maxUseDepth {
		return nil
	}
	if ctx.inClip && !isShapeName(target.name) && target.name != "text" {
		return nil
	}
	lc := b.lenFor(st.FontSize, ctx)
	x := lengthAttr(e, "x", &lc, axisX, 0)
	y := lengthAttr(e, "y", &lc, axisY, 0)
	n := b.newNode(NodeGroup, e, st, np, ctx)
	n.Transform = np.transform.Mul(Translate(x, y))

	b.useStack = append(b.useStack, e, target)
	defer func() { b.useStack = b.useStack[:len(b.useStack)-2] }()
	in := *ctx
	in.inUse = true
	var child *Node
	switch target.name {
	case "symbol", "svg":
		if !b.conditionsHold(target) {
			break
		}
		tst, tnp := b.computeStyle(target, b.sheet.cascade(target), st, &in)
		if !tnp.display {
			break
		}
		var size [2]*float64
		if v, ok := optLengthAttr(e, "width", &lc, axisX); ok {
			size[0] = &v
		}
		if v, ok := optLengthAttr(e, "height", &lc, axisY); ok {
			size[1] = &v
		}
		if target.name == "symbol" {
			// A symbol is a viewport at the use's position, as large as the use says (100% by default).
			tnp.transform = Identity
			sc := b.lenFor(tst.FontSize, &in)
			vp, err := b.viewportOf(target, &tnp, &sc, &in, size[0], size[1])
			if err != nil {
				b.err = err
				break
			}
			vp.X, vp.Y = 0, 0
			if vp.ViewBox != nil && (vp.ViewBox.Width == 0 || vp.ViewBox.Height == 0) {
				break
			}
			child = b.newNode(NodeSvg, target, tst, &tnp, &in)
			child.Viewport = vp
			inner := innerCtx(vp, &in)
			child.Children = b.buildChildren(target, tst, &inner)
		} else {
			if size[0] == nil && size[1] == nil {
				child = b.buildNestedSVG(target, tst, &tnp, &in, nil)
			} else {
				child = b.buildNestedSVG(target, tst, &tnp, &in, &size)
			}
		}
	default:
		child = b.buildElement(target, st, &in)
	}
	if child == nil {
		return nil
	}
	n.Children = []*Node{child}
	return n
}

// buildImage makes the node of an image element with a data: URL.
//
// SVG 1.1: struct.html#ImageElement
func (b *builder) buildImage(e *element, st *Style, np *nodeProps, ctx *buildCtx) *Node {
	href, ok := e.href()
	if !ok {
		return nil
	}
	mime, data, ok := decodeDataURL(href)
	if !ok || len(data) == 0 {
		return nil
	}
	lc := b.lenFor(st.FontSize, ctx)
	img := &Image{
		X:      lengthAttr(e, "x", &lc, axisX, 0),
		Y:      lengthAttr(e, "y", &lc, axisY, 0),
		Width:  lengthAttr(e, "width", &lc, axisX, 0),
		Height: lengthAttr(e, "height", &lc, axisY, 0),
		MIME:   mime,
		Data:   data,
	}
	if img.Width < 0 || img.Height < 0 {
		return nil
	}
	if v, ok := e.attr("preserveAspectRatio"); ok {
		if par, ok := ParsePreserveAspectRatio(v); ok {
			img.PreserveAspectRatio = par
		}
	}
	n := b.newNode(NodeImage, e, st, np, ctx)
	n.Image = img
	return n
}

// decodeDataURL decodes a data: URL into its media type and its content.
//
// RFC 2397
func decodeDataURL(s string) (mime string, data []byte, ok bool) {
	s = strings.TrimSpace(s)
	if len(s) < 5 || !strings.EqualFold(s[:5], "data:") {
		return "", nil, false
	}
	s = s[5:]
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return "", nil, false
	}
	meta, payload := s[:comma], s[comma+1:]
	isBase64 := false
	parts := strings.Split(meta, ";")
	mime = strings.ToLower(strings.TrimSpace(parts[0]))
	for _, p := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(p), "base64") {
			isBase64 = true
		}
	}
	if mime == "" {
		mime = "text/plain"
	}
	if isBase64 {
		clean := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, payload)
		if p, err := url.PathUnescape(clean); err == nil {
			clean = p
		}
		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
			if d, err := enc.DecodeString(clean); err == nil {
				return mime, d, true
			}
		}
		return "", nil, false
	}
	d, err := url.PathUnescape(payload)
	if err != nil {
		return "", nil, false
	}
	return mime, []byte(d), true
}
