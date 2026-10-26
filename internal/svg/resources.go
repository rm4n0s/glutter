package svg

import (
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// paintServer returns the gradient or pattern that an id names, or nil.
func (b *builder) paintServer(id string) PaintServer {
	e := b.ids[id]
	if e == nil {
		return nil
	}
	switch e.name {
	case "linearGradient", "radialGradient":
		if g := b.gradient(e); g != nil {
			return g
		}
	case "pattern":
		if p := b.pattern(e); p != nil {
			return p
		}
	}
	return nil
}

// refChain returns the element and the elements it refers to with href (and those, in turn), as long as they are
// accepted; a cycle ends the chain.
//
// SVG 1.1: pservers.html#LinearGradientElementHrefAttribute
func (b *builder) refChain(e *element, accept func(*element) bool) []*element {
	chain := []*element{e}
	seen := map[*element]bool{e: true}
	for cur := e; len(chain) < 32; {
		href, ok := cur.href()
		if !ok {
			break
		}
		id, ok := hrefID(href)
		if !ok {
			break
		}
		next := b.ids[id]
		if next == nil || seen[next] || !accept(next) {
			break
		}
		seen[next] = true
		chain = append(chain, next)
		cur = next
	}
	return chain
}

// chainAttr returns the first value of an attribute along the chain, with the element that has it.
func chainAttr(chain []*element, name string) (string, *element) {
	for _, e := range chain {
		if v, ok := e.attr(name); ok {
			return v, e
		}
	}
	return "", nil
}

func parseUnits(v string, def Units) Units {
	switch strings.TrimSpace(v) {
	case "userSpaceOnUse":
		return UnitsUserSpaceOnUse
	case "objectBoundingBox":
		return UnitsObjectBoundingBox
	}
	return def
}

// chainLength reads a length attribute along the chain; the length keeps its percentage (see Length) and is
// resolved with the font size of the element that has the attribute.
func (b *builder) chainLength(chain []*element, name string, def Length) Length {
	v, owner := chainAttr(chain, name)
	if owner == nil {
		return def
	}
	l, ok := ParseLength(v)
	if !ok {
		return def
	}
	lc := b.lenFor(b.styleOf(owner).FontSize, &b.rootCtx)
	return lc.keep(l)
}

func (b *builder) chainTransform(chain []*element, name string) Affine {
	v, owner := chainAttr(chain, name)
	if owner == nil {
		return Identity
	}
	if t, ok := ParseTransform(v); ok {
		return t
	}
	return Identity
}

func isGradient(e *element) bool { return e.name == "linearGradient" || e.name == "radialGradient" }

// gradient makes a gradient with the attributes and the stops it inherits.
//
// SVG 1.1: pservers.html#Gradients
func (b *builder) gradient(e *element) *Gradient {
	if v, ok := b.res[e]; ok {
		g, _ := v.(*Gradient)
		return g
	}
	chain := b.refChain(e, isGradient)
	g := &Gradient{Radial: e.name == "radialGradient"}
	v, _ := chainAttr(chain, "gradientUnits")
	g.Units = parseUnits(v, UnitsObjectBoundingBox)
	g.Transform = b.chainTransform(chain, "gradientTransform")
	v, _ = chainAttr(chain, "spreadMethod")
	switch strings.TrimSpace(v) {
	case "reflect":
		g.Spread = SpreadReflect
	case "repeat":
		g.Spread = SpreadRepeat
	}
	// The geometry attributes are taken only from gradients of the same kind.
	same := make([]*element, 0, len(chain))
	for _, c := range chain {
		if c.name == e.name {
			same = append(same, c)
		}
	}
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	if g.Radial {
		g.CX = b.chainLength(same, "cx", pct(50))
		g.CY = b.chainLength(same, "cy", pct(50))
		g.R = b.chainLength(same, "r", pct(50))
		g.FX = b.chainLength(same, "fx", g.CX)
		g.FY = b.chainLength(same, "fy", g.CY)
		g.FR = b.chainLength(same, "fr", Length{})
	} else {
		g.X1 = b.chainLength(same, "x1", pct(0))
		g.Y1 = b.chainLength(same, "y1", pct(0))
		g.X2 = b.chainLength(same, "x2", pct(100))
		g.Y2 = b.chainLength(same, "y2", pct(0))
	}
	for _, c := range chain {
		if stops := b.stops(c); len(stops) > 0 {
			g.Stops = stops
			break
		}
	}
	if len(g.Stops) == 0 {
		g = nil
	}
	b.res[e] = g
	return g
}

// stops reads the stop children of a gradient element.
//
// SVG 1.1: pservers.html#GradientStops
func (b *builder) stops(g *element) []Stop {
	var out []Stop
	last := 0.0
	for _, c := range g.children {
		if c.name != "stop" {
			continue
		}
		off := 0.0
		if v, ok := c.attr("offset"); ok {
			if n, ok := parseNumberOrPercent(v); ok {
				off = clamp01(n)
			}
		}
		if off < last {
			off = last
		}
		last = off
		decls := b.sheet.cascade(c)
		st := b.styleOf(c)
		color := ui.NewColorFromARGB(255, 0, 0, 0)
		if v, ok := decls.get("stop-color"); ok {
			if col, cur, ok := ParseColor(v); ok {
				color = col
				if cur {
					color = st.Color
				}
			}
		}
		if v, ok := decls.get("stop-opacity"); ok {
			if o, ok := parseOpacity(v); ok {
				color = scaleColorAlpha(color, o)
			}
		}
		out = append(out, Stop{Offset: off, Color: color})
	}
	return out
}

func scaleColorAlpha(c ui.Color, f float64) ui.Color {
	a := int(float64(c.Alpha())*f + 0.5)
	return ui.NewColorFromARGB(a, c.Red(), c.Green(), c.Blue())
}

func isPattern(e *element) bool { return e.name == "pattern" }

// pattern makes a pattern with the attributes and the content it inherits.
//
// SVG 1.1: pservers.html#Patterns
func (b *builder) pattern(e *element) *Pattern {
	if v, ok := b.res[e]; ok {
		p, _ := v.(*Pattern)
		return p
	}
	if b.busy[e] {
		return nil
	}
	b.busy[e] = true
	defer delete(b.busy, e)
	chain := b.refChain(e, isPattern)
	p := &Pattern{}
	v, _ := chainAttr(chain, "patternUnits")
	p.Units = parseUnits(v, UnitsObjectBoundingBox)
	v, _ = chainAttr(chain, "patternContentUnits")
	p.ContentUnits = parseUnits(v, UnitsUserSpaceOnUse)
	p.Transform = b.chainTransform(chain, "patternTransform")
	p.X = b.chainLength(chain, "x", Length{})
	p.Y = b.chainLength(chain, "y", Length{})
	p.Width = b.chainLength(chain, "width", Length{})
	p.Height = b.chainLength(chain, "height", Length{})
	if v, owner := chainAttr(chain, "viewBox"); owner != nil {
		vb, ok, err := parseViewBox(v)
		if err != nil {
			b.err = err
		} else if ok {
			p.ViewBox = &vb
		}
	}
	if v, owner := chainAttr(chain, "preserveAspectRatio"); owner != nil {
		if par, ok := ParsePreserveAspectRatio(v); ok {
			p.PreserveAspectRatio = par
		}
	}
	if p.Width.Value <= 0 || p.Height.Value <= 0 || (p.ViewBox != nil && (p.ViewBox.Width == 0 || p.ViewBox.Height == 0)) {
		b.res[e] = (*Pattern)(nil)
		return nil
	}
	for _, c := range chain {
		if hasElementChild(c) {
			ctx := b.rootCtx
			if p.ViewBox != nil {
				ctx.vw, ctx.vh, ctx.vpKnown = p.ViewBox.Width, p.ViewBox.Height, true
			}
			p.Children = b.buildChildren(c, b.styleOf(c), &ctx)
			break
		}
	}
	b.res[e] = p
	return p
}

func hasElementChild(e *element) bool {
	for _, c := range e.children {
		if c.name != "#text" {
			return true
		}
	}
	return false
}

// clipPath makes the clip path an id names, or nil.
//
// SVG 1.1: masking.html#EstablishingANewClippingPath
func (b *builder) clipPath(id string) *ClipPath {
	e := b.ids[id]
	if e == nil || e.name != "clipPath" {
		return nil
	}
	if v, ok := b.res[e]; ok {
		c, _ := v.(*ClipPath)
		return c
	}
	if b.busy[e] {
		return nil
	}
	b.busy[e] = true
	defer delete(b.busy, e)
	st := b.styleOf(e)
	decls := b.sheet.cascade(e)
	_, np := b.computeStyle(e, decls, b.styleOf2(e.parent), &b.rootCtx)
	c := &ClipPath{Transform: np.transform}
	if v, ok := e.attr("clipPathUnits"); ok {
		c.Units = parseUnits(v, UnitsUserSpaceOnUse)
	} else {
		c.Units = UnitsUserSpaceOnUse
	}
	if np.clipPath != "" {
		c.ClipPath = b.clipPath(np.clipPath)
	}
	ctx := b.rootCtx
	ctx.inClip = true
	c.Children = b.buildChildren(e, st, &ctx)
	b.res[e] = c
	return c
}

// styleOf2 is styleOf for an optional element: the initial style for nil.
func (b *builder) styleOf2(e *element) *Style {
	if e == nil {
		return b.initialStyle()
	}
	return b.styleOf(e)
}

// mask makes the mask an id names, or nil.
//
// SVG 1.1: masking.html#Masking
func (b *builder) mask(id string) *Mask {
	e := b.ids[id]
	if e == nil || e.name != "mask" {
		return nil
	}
	if v, ok := b.res[e]; ok {
		m, _ := v.(*Mask)
		return m
	}
	if b.busy[e] {
		return nil
	}
	b.busy[e] = true
	defer delete(b.busy, e)
	st := b.styleOf(e)
	decls := b.sheet.cascade(e)
	_, np := b.computeStyle(e, decls, b.styleOf2(e.parent), &b.rootCtx)
	lc := b.lenFor(st.FontSize, &b.rootCtx)
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	m := &Mask{}
	one := []*element{e}
	m.X = b.chainLength(one, "x", pct(-10))
	m.Y = b.chainLength(one, "y", pct(-10))
	m.Width = b.chainLength(one, "width", pct(120))
	m.Height = b.chainLength(one, "height", pct(120))
	_ = lc
	m.Units = parseUnits(e.attrOr("maskUnits", ""), UnitsObjectBoundingBox)
	m.ContentUnits = parseUnits(e.attrOr("maskContentUnits", ""), UnitsUserSpaceOnUse)
	if v, ok := decls.get("mask-type"); ok && strings.TrimSpace(v) == "alpha" {
		m.Type = MaskAlpha
	}
	if np.mask != "" {
		m.Mask = b.mask(np.mask)
	}
	ctx := b.rootCtx
	m.Children = b.buildChildren(e, st, &ctx)
	b.res[e] = m
	return m
}

// marker makes the marker an id names, or nil.
//
// SVG 1.1: painting.html#MarkerElement
func (b *builder) marker(id string) *Marker {
	e := b.ids[id]
	if e == nil || e.name != "marker" {
		return nil
	}
	if v, ok := b.res[e]; ok {
		m, _ := v.(*Marker)
		return m
	}
	if b.busy[e] {
		return nil
	}
	b.busy[e] = true
	defer delete(b.busy, e)
	st := b.styleOf(e)
	decls := b.sheet.cascade(e)
	_, np := b.computeStyle(e, decls, b.styleOf2(e.parent), &b.rootCtx)
	lc := b.lenFor(st.FontSize, &b.rootCtx)
	m := &Marker{
		RefX:             lengthAttr(e, "refX", &lc, axisX, 0),
		RefY:             lengthAttr(e, "refY", &lc, axisY, 0),
		Width:            lengthAttr(e, "markerWidth", &lc, axisX, 3),
		Height:           lengthAttr(e, "markerHeight", &lc, axisY, 3),
		StrokeWidthUnits: strings.TrimSpace(e.attrOr("markerUnits", "strokeWidth")) != "userSpaceOnUse",
		Overflow:         np.overflowSet && np.overflow,
	}
	switch o := strings.TrimSpace(e.attrOr("orient", "0")); o {
	case "auto":
		m.Orient = OrientAuto
	case "auto-start-reverse":
		m.Orient = OrientAutoStartReverse
	default:
		if a, ok := parseAngle(o); ok {
			m.Angle = a
		}
	}
	if v, ok := e.attr("viewBox"); ok {
		vb, good, err := parseViewBox(v)
		if err != nil {
			b.err = err
		} else if good {
			m.ViewBox = &vb
		}
	}
	if v, ok := e.attr("preserveAspectRatio"); ok {
		if par, ok := ParsePreserveAspectRatio(v); ok {
			m.PreserveAspectRatio = par
		}
	}
	if m.Width <= 0 || m.Height <= 0 {
		b.res[e] = (*Marker)(nil)
		return nil
	}
	ctx := b.rootCtx
	if m.ViewBox != nil {
		ctx.vw, ctx.vh, ctx.vpKnown = m.ViewBox.Width, m.ViewBox.Height, true
	} else {
		ctx.vw, ctx.vh, ctx.vpKnown = m.Width, m.Height, true
	}
	m.Children = b.buildChildren(e, st, &ctx)
	b.res[e] = m
	return m
}

// parseAngle reads an angle in degrees (a number, or a number with deg, rad, grad or turn).
func parseAngle(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	v, j, ok := scanNumber(s, 0)
	if !ok {
		return 0, false
	}
	switch s[j:] {
	case "", "deg":
		return v, true
	case "rad":
		return v * 180 / 3.141592653589793, true
	case "grad":
		return v * 360 / 400, true
	case "turn":
		return v * 360, true
	}
	return 0, false
}

func isFilter(e *element) bool { return e.name == "filter" }

// filter makes the filter an id names, or nil.
//
// SVG 1.1: filters.html#FilterElement
func (b *builder) filter(id string) *Filter {
	e := b.ids[id]
	if e == nil || e.name != "filter" {
		return nil
	}
	if v, ok := b.res[e]; ok {
		f, _ := v.(*Filter)
		return f
	}
	if b.busy[e] {
		return nil
	}
	b.busy[e] = true
	defer delete(b.busy, e)
	chain := b.refChain(e, isFilter)
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	f := &Filter{}
	f.X = b.chainLength(chain, "x", pct(-10))
	f.Y = b.chainLength(chain, "y", pct(-10))
	f.Width = b.chainLength(chain, "width", pct(120))
	f.Height = b.chainLength(chain, "height", pct(120))
	v, _ := chainAttr(chain, "filterUnits")
	f.Units = parseUnits(v, UnitsObjectBoundingBox)
	v, _ = chainAttr(chain, "primitiveUnits")
	f.PrimitiveUnits = parseUnits(v, UnitsUserSpaceOnUse)
	for _, c := range chain {
		if hasElementChild(c) {
			for _, child := range c.children {
				if child.name != "#text" && strings.HasPrefix(child.name, "fe") {
					f.Primitives = append(f.Primitives, b.primitive(child))
				}
			}
			break
		}
	}
	b.res[e] = f
	return f
}

// filterCommon are the attributes of a primitive that are fields of FilterPrimitive, not entries of Attrs.
var filterCommon = map[string]bool{
	"in": true, "in2": true, "result": true, "x": true, "y": true, "width": true, "height": true,
	"id": true, "class": true, "style": true,
}

// primitive reads one filter primitive element and its children.
func (b *builder) primitive(e *element) *FilterPrimitive {
	st := b.styleOf(e)
	decls := b.sheet.cascade(e)
	lc := b.lenFor(st.FontSize, &b.rootCtx)
	p := &FilterPrimitive{
		Name:         e.name,
		In:           strings.TrimSpace(e.attrOr("in", "")),
		In2:          strings.TrimSpace(e.attrOr("in2", "")),
		Result:       strings.TrimSpace(e.attrOr("result", "")),
		Style:        st,
		FloodOpacity: 1,
	}
	if e.name == "feDiffuseLighting" || e.name == "feSpecularLighting" {
		p.FloodColor = ui.NewColorFromARGB(255, 255, 255, 255)
		if v, ok := decls.get("lighting-color"); ok {
			if col, cur, ok := ParseColor(v); ok {
				p.FloodColor = col
				if cur {
					p.FloodColor = st.Color
				}
			}
		}
	} else {
		p.FloodColor = ui.NewColorFromARGB(255, 0, 0, 0)
		if v, ok := decls.get("flood-color"); ok {
			if col, cur, ok := ParseColor(v); ok {
				p.FloodColor = col
				if cur {
					p.FloodColor = st.Color
				}
			}
		}
		if v, ok := decls.get("flood-opacity"); ok {
			if o, ok := parseOpacity(v); ok {
				p.FloodOpacity = o
			}
		}
	}
	sub := func(name string) *Length {
		v, ok := e.attr(name)
		if !ok {
			return nil
		}
		l, ok := ParseLength(v)
		if !ok {
			return nil
		}
		k := lc.keep(l)
		return &k
	}
	p.X, p.Y, p.Width, p.Height = sub("x"), sub("y"), sub("width"), sub("height")
	p.Attrs = map[string]string{}
	for _, a := range e.attrs {
		if !filterCommon[a.name] && !presentationProps[a.name] {
			p.Attrs[a.name] = a.value
		}
	}
	for _, c := range e.children {
		if c.name != "#text" {
			p.Children = append(p.Children, b.primitive(c))
		}
	}
	return p
}
