package svg

import (
	"math"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// nodeProps are the properties of an element that do not inherit.
type nodeProps struct {
	display      bool // false for display: none
	opacity      float64
	clipPath     string // the id in clip-path: url(#id), or ""
	mask         string
	filters      []string
	vectorEffect VectorEffect
	overflow     bool // overflow: visible or auto
	overflowSet  bool
	transform    Affine
}

func defaultNodeProps() nodeProps {
	return nodeProps{display: true, opacity: 1, transform: Identity}
}

// initialStyle is the style of the properties at the root before any declaration: the initial values of SVG.
func (b *builder) initialStyle() *Style {
	return &Style{
		Color:                     b.theme.CurrentColor,
		Fill:                      Paint{Kind: PaintKindColor, Color: ui.NewColorFromARGB(255, 0, 0, 0)},
		FillOpacity:               1,
		StrokeOpacity:             1,
		StrokeWidth:               1,
		MiterLimit:                4,
		Visible:                   true,
		PaintOrder:                DefaultPaintOrder,
		FontSize:                  b.theme.FontSize,
		FontWeight:                400,
		ColorInterpolationFilters: ColorInterpolationLinearRGB,
	}
}

// lenFor returns the context that resolves the lengths of an element whose font size is fontSize.
func (b *builder) lenFor(fontSize float64, ctx *buildCtx) lenCtx {
	return lenCtx{
		fontSize: fontSize,
		rootFont: b.theme.FontSize,
		xRatio:   b.theme.XHeight / b.theme.FontSize,
		vw:       ctx.vw, vh: ctx.vh, vpKnown: ctx.vpKnown,
	}
}

// styleOf returns the computed style of an element for the uses that are not part of a use expansion (resources,
// gradient stops, filter primitives): it inherits from the element's parent in the document.
func (b *builder) styleOf(e *element) *Style {
	if st, ok := b.styleMemo[e]; ok {
		return st
	}
	var parent *Style
	if e.parent == nil {
		parent = b.initialStyle()
	} else {
		parent = b.styleOf(e.parent)
	}
	st, _ := b.computeStyle(e, b.sheet.cascade(e), parent, &b.rootCtx)
	b.styleMemo[e] = st
	return st
}

// computeStyle applies the declarations of an element on the style of its parent. It returns the computed style of
// the properties that inherit (the parent's, itself, when no declaration changes any) and the properties that do
// not inherit.
//
// CSS 2.1: cascade.html#computed-value
func (b *builder) computeStyle(e *element, decls declList, parent *Style, ctx *buildCtx) (*Style, nodeProps) {
	np := defaultNodeProps()
	st := parent
	changed := false
	mut := func() *Style {
		if !changed {
			c := *parent
			st = &c
			changed = true
		}
		return st
	}
	if len(decls) == 0 {
		return st, np
	}
	init := b.initialStyle()
	// The properties that other values depend on come first: color for currentColor, font-size for em.
	for i := range decls {
		d := &decls[i]
		switch d.name {
		case "color":
			switch d.value {
			case "inherit":
				mut().Color = parent.Color
			case "initial":
				mut().Color = init.Color
			default:
				if c, cur, ok := ParseColor(d.value); ok {
					if cur {
						c = parent.Color
					}
					mut().Color = c
				}
			}
		case "font-size":
			if fs, ok := b.fontSize(d.value, parent, ctx); ok {
				mut().FontSize = fs
			}
		}
	}
	lc := b.lenFor(st.FontSize, ctx)
	for i := range decls {
		d := &decls[i]
		b.applyProperty(d.name, d.value, parent, init, mut, &lc, &np, ctx)
	}
	return st, np
}

// applyProperty applies one declaration.
func (b *builder) applyProperty(name, v string, parent, init *Style, mut func() *Style, lc *lenCtx, np *nodeProps, ctx *buildCtx) {
	if v == "inherit" || v == "initial" || v == "unset" {
		src := parent
		if v == "initial" {
			src = init
		}
		// The properties that do not inherit start at their initial values in nodeProps, so there is nothing to copy
		// for them.
		copyProperty(name, mut, src)
		return
	}
	st := mut
	switch name {
	case "fill":
		if p, ok := b.paintValue(v, mut().Color); ok {
			st().Fill = p
		}
	case "stroke":
		if p, ok := b.paintValue(v, mut().Color); ok {
			st().Stroke = p
		}
	case "fill-opacity":
		if o, ok := parseOpacity(v); ok {
			st().FillOpacity = o
		}
	case "stroke-opacity":
		if o, ok := parseOpacity(v); ok {
			st().StrokeOpacity = o
		}
	case "opacity":
		if o, ok := parseOpacity(v); ok {
			np.opacity = o
		}
	case "fill-rule":
		switch v {
		case "nonzero":
			st().FillRule = FillRuleNonZero
		case "evenodd":
			st().FillRule = FillRuleEvenOdd
		}
	case "clip-rule":
		switch v {
		case "nonzero":
			st().ClipRule = FillRuleNonZero
		case "evenodd":
			st().ClipRule = FillRuleEvenOdd
		}
	case "stroke-width":
		if l, ok := ParseLength(v); ok {
			if px, ok := lc.px(l, axisXY); ok && px >= 0 {
				st().StrokeWidth = px
			}
		}
	case "stroke-linecap":
		switch v {
		case "butt":
			st().LineCap = LineCapButt
		case "round":
			st().LineCap = LineCapRound
		case "square":
			st().LineCap = LineCapSquare
		}
	case "stroke-linejoin":
		switch v {
		case "miter", "miter-clip", "arcs":
			st().LineJoin = LineJoinMiter
		case "round":
			st().LineJoin = LineJoinRound
		case "bevel":
			st().LineJoin = LineJoinBevel
		}
	case "stroke-miterlimit":
		if n, ok := parseNumber(v); ok && n >= 1 {
			st().MiterLimit = n
		}
	case "stroke-dasharray":
		if arr, ok := parseDashArray(v, lc); ok {
			st().DashArray = arr
		}
	case "stroke-dashoffset":
		if l, ok := ParseLength(v); ok {
			if px, ok := lc.px(l, axisXY); ok {
				st().DashOffset = px
			}
		}
	case "visibility":
		switch v {
		case "visible":
			st().Visible = true
		case "hidden", "collapse":
			st().Visible = false
		}
	case "shape-rendering":
		switch v {
		case "crispEdges":
			st().ShapeRendering = ShapeRenderingCrispEdges
		case "auto", "optimizeSpeed", "geometricPrecision":
			st().ShapeRendering = ShapeRenderingAuto
		}
	case "paint-order":
		if po, ok := parsePaintOrder(v); ok {
			st().PaintOrder = po
		}
	case "color-interpolation-filters":
		switch v {
		case "linearRGB":
			st().ColorInterpolationFilters = ColorInterpolationLinearRGB
		case "sRGB", "auto":
			st().ColorInterpolationFilters = ColorInterpolationSRGB
		}
	case "marker-start":
		if id, ok := markerRef(v); ok {
			st().MarkerStart = b.marker(id)
		} else if v == "none" {
			st().MarkerStart = nil
		}
	case "marker-mid":
		if id, ok := markerRef(v); ok {
			st().MarkerMid = b.marker(id)
		} else if v == "none" {
			st().MarkerMid = nil
		}
	case "marker-end":
		if id, ok := markerRef(v); ok {
			st().MarkerEnd = b.marker(id)
		} else if v == "none" {
			st().MarkerEnd = nil
		}
	case "font-family":
		if fam := parseFontFamily(v); len(fam) > 0 {
			st().FontFamily = fam
		}
	case "font-weight":
		if w, ok := parseFontWeight(v, parent.FontWeight); ok {
			st().FontWeight = w
		}
	case "font-style":
		switch v {
		case "normal":
			st().FontStyle = FontStyleNormal
		case "italic":
			st().FontStyle = FontStyleItalic
		case "oblique":
			st().FontStyle = FontStyleOblique
		}
	case "text-anchor":
		switch v {
		case "start":
			st().TextAnchor = TextAnchorStart
		case "middle":
			st().TextAnchor = TextAnchorMiddle
		case "end":
			st().TextAnchor = TextAnchorEnd
		}
	case "text-decoration":
		st().TextDecoration = parseTextDecoration(v)
	case "letter-spacing":
		if px, ok := parseSpacing(v, lc); ok {
			st().LetterSpacing = px
		}
	case "word-spacing":
		if px, ok := parseSpacing(v, lc); ok {
			st().WordSpacing = px
		}
	case "display":
		np.display = v != "none"
	case "clip-path":
		if v == "none" {
			np.clipPath = ""
		} else if id, _, ok := parseURLRef(v); ok {
			np.clipPath = id
		}
	case "mask":
		if v == "none" {
			np.mask = ""
		} else if id, _, ok := parseURLRef(v); ok {
			np.mask = id
		}
	case "filter":
		np.filters = parseFilterRefs(v)
	case "vector-effect":
		if strings.Contains(v, "non-scaling-stroke") {
			np.vectorEffect = VectorEffectNonScalingStroke
			b.scaleDependent = true
		} else {
			np.vectorEffect = VectorEffectNone
		}
	case "overflow":
		switch v {
		case "visible", "auto":
			np.overflow, np.overflowSet = true, true
		case "hidden", "scroll", "clip":
			np.overflow, np.overflowSet = false, true
		}
	case "transform":
		// The attribute has the SVG syntax (rotate(a x y), no units) and the CSS property has the CSS one (angles with
		// units); both come here, so a value is read as CSS and, if that fails, as SVG.
		if t, ok := parseCSSTransform(v); ok {
			np.transform = t
		} else if t, ok := ParseTransform(v); ok {
			np.transform = t
		}
	}
}

// copyProperty sets a property of the style from the same property of src (the value of inherit and initial).
func copyProperty(name string, mut func() *Style, src *Style) {
	var dst *Style
	get := func() *Style {
		if dst == nil {
			dst = mut()
		}
		return dst
	}
	switch name {
	case "fill":
		get().Fill = src.Fill
	case "stroke":
		get().Stroke = src.Stroke
	case "fill-opacity":
		get().FillOpacity = src.FillOpacity
	case "stroke-opacity":
		get().StrokeOpacity = src.StrokeOpacity
	case "fill-rule":
		get().FillRule = src.FillRule
	case "clip-rule":
		get().ClipRule = src.ClipRule
	case "stroke-width":
		get().StrokeWidth = src.StrokeWidth
	case "stroke-linecap":
		get().LineCap = src.LineCap
	case "stroke-linejoin":
		get().LineJoin = src.LineJoin
	case "stroke-miterlimit":
		get().MiterLimit = src.MiterLimit
	case "stroke-dasharray":
		get().DashArray = src.DashArray
	case "stroke-dashoffset":
		get().DashOffset = src.DashOffset
	case "visibility":
		get().Visible = src.Visible
	case "shape-rendering":
		get().ShapeRendering = src.ShapeRendering
	case "paint-order":
		get().PaintOrder = src.PaintOrder
	case "color-interpolation-filters":
		get().ColorInterpolationFilters = src.ColorInterpolationFilters
	case "marker-start":
		get().MarkerStart = src.MarkerStart
	case "marker-mid":
		get().MarkerMid = src.MarkerMid
	case "marker-end":
		get().MarkerEnd = src.MarkerEnd
	case "font-family":
		get().FontFamily = src.FontFamily
	case "font-weight":
		get().FontWeight = src.FontWeight
	case "font-style":
		get().FontStyle = src.FontStyle
	case "text-anchor":
		get().TextAnchor = src.TextAnchor
	case "text-decoration":
		get().TextDecoration = src.TextDecoration
	case "letter-spacing":
		get().LetterSpacing = src.LetterSpacing
	case "word-spacing":
		get().WordSpacing = src.WordSpacing
	}
}

// parseOpacity reads a number or a percentage, clamped to [0, 1].
//
// SVG 1.1: masking.html#OpacityProperty
func parseOpacity(v string) (float64, bool) {
	n, ok := parseNumberOrPercent(v)
	if !ok {
		return 0, false
	}
	return clamp01(n), true
}

// paintValue parses the value of fill or stroke. A url() to a gradient or pattern that exists is a PaintServer; one
// that does not exist is replaced by its fallback, or none.
//
// SVG 1.1: painting.html#SpecifyingPaint
func (b *builder) paintValue(v string, current ui.Color) (Paint, bool) {
	v = strings.TrimSpace(v)
	switch {
	case v == "none":
		return Paint{}, true
	case strings.HasPrefix(strings.ToLower(v), "url("):
		id, rest, ok := parseURLRef(v)
		if !ok {
			return Paint{}, false
		}
		if srv := b.paintServer(id); srv != nil {
			return Paint{Kind: PaintKindServer, Server: srv}, true
		}
		rest = strings.TrimSpace(rest)
		if rest == "" || rest == "none" {
			return Paint{}, true
		}
		return b.paintValue(rest, current)
	case v == "context-fill" || v == "context-stroke":
		return Paint{}, true
	}
	// "#fff icc-color(...)": the fallback sRGB color comes first.
	if i := strings.Index(v, "icc-color("); i > 0 {
		v = strings.TrimSpace(v[:i])
	}
	c, cur, ok := ParseColor(v)
	if !ok {
		return Paint{}, false
	}
	if cur {
		c = current
	}
	return Paint{Kind: PaintKindColor, Color: c}, true
}

func markerRef(v string) (string, bool) {
	id, _, ok := parseURLRef(v)
	return id, ok
}

func parseFilterRefs(v string) []string {
	if strings.TrimSpace(v) == "none" {
		return nil
	}
	var out []string
	for {
		v = strings.TrimSpace(v)
		if v == "" {
			return out
		}
		if strings.HasPrefix(strings.ToLower(v), "url(") {
			id, rest, ok := parseURLRef(v)
			if ok {
				out = append(out, id)
			}
			v = rest
			continue
		}
		// A filter function such as blur(2px): not supported, skipped.
		open := strings.IndexByte(v, '(')
		end := strings.IndexByte(v, ')')
		if open < 0 || end < open {
			return out
		}
		v = v[end+1:]
	}
}

// parseDashArray parses stroke-dasharray.
//
// SVG 1.1: painting.html#StrokeDasharrayProperty
func parseDashArray(v string, lc *lenCtx) ([]float64, bool) {
	if strings.TrimSpace(v) == "none" {
		return nil, true
	}
	var out []float64
	sum := 0.0
	i := skipSpace(v, 0)
	for i < len(v) {
		l, j, ok := scanLength(v, i)
		if !ok {
			return nil, false
		}
		px, ok := lc.px(l, axisXY)
		if !ok || px < 0 {
			return nil, false
		}
		out = append(out, px)
		sum += px
		i = skipCommaSpace(v, j)
	}
	if len(out) == 0 {
		return nil, false
	}
	if sum <= 0 {
		return nil, true
	}
	if len(out)%2 == 1 {
		out = append(out, out...)
	}
	return out, true
}

func parsePaintOrder(v string) ([3]PaintLayer, bool) {
	f := strings.Fields(v)
	if len(f) == 1 && f[0] == "normal" {
		return DefaultPaintOrder, true
	}
	var out [3]PaintLayer
	var seen [3]bool
	n := 0
	for _, w := range f {
		var l PaintLayer
		switch w {
		case "fill":
			l = PaintLayerFill
		case "stroke":
			l = PaintLayerStroke
		case "markers":
			l = PaintLayerMarkers
		default:
			return out, false
		}
		if seen[l] {
			return out, false
		}
		seen[l] = true
		out[n] = l
		n++
	}
	if n == 0 {
		return out, false
	}
	for _, l := range DefaultPaintOrder {
		if !seen[l] {
			out[n] = l
			n++
		}
	}
	return out, true
}

func parseFontFamily(v string) []string {
	var out []string
	for _, part := range splitTop(v, ',') {
		part = strings.TrimSpace(part)
		if len(part) >= 2 && (part[0] == '\'' || part[0] == '"') && part[len(part)-1] == part[0] {
			part = part[1 : len(part)-1]
		}
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// parseFontWeight parses font-weight; parent is the weight bolder and lighter are relative to.
//
// CSS Fonts 3: font-weight
func parseFontWeight(v string, parent int) (int, bool) {
	switch v {
	case "normal":
		return 400, true
	case "bold":
		return 700, true
	case "bolder":
		switch {
		case parent < 400:
			return 400, true
		case parent < 600:
			return 700, true
		}
		return 900, true
	case "lighter":
		switch {
		case parent < 600:
			return 100, true
		case parent < 800:
			return 400, true
		}
		return 700, true
	}
	n, ok := parseNumber(v)
	if !ok || n < 1 || n > 1000 {
		return 0, false
	}
	return int(n), true
}

func parseTextDecoration(v string) TextDecoration {
	var d TextDecoration
	for _, w := range strings.Fields(v) {
		switch w {
		case "underline":
			d |= TextDecorationUnderline
		case "overline":
			d |= TextDecorationOverline
		case "line-through":
			d |= TextDecorationLineThrough
		}
	}
	return d
}

func parseSpacing(v string, lc *lenCtx) (float64, bool) {
	if strings.TrimSpace(v) == "normal" {
		return 0, true
	}
	l, ok := ParseLength(v)
	if !ok {
		return 0, false
	}
	return lc.px(l, axisX)
}

// fontSize parses font-size against the parent's: a relative unit is relative to the parent's size, the keywords
// are scaled from the theme's font size (medium).
//
// CSS Fonts 3: font-size
func (b *builder) fontSize(v string, parent *Style, ctx *buildCtx) (float64, bool) {
	medium := b.theme.FontSize
	switch strings.TrimSpace(v) {
	case "inherit":
		return parent.FontSize, true
	case "initial", "medium":
		return medium, true
	case "xx-small":
		return medium * 3 / 5, true
	case "x-small":
		return medium * 3 / 4, true
	case "small":
		return medium * 8 / 9, true
	case "large":
		return medium * 6 / 5, true
	case "x-large":
		return medium * 3 / 2, true
	case "xx-large":
		return medium * 2, true
	case "larger":
		return parent.FontSize * 1.2, true
	case "smaller":
		return parent.FontSize / 1.2, true
	}
	l, ok := ParseLength(v)
	if !ok || l.Value < 0 {
		return 0, false
	}
	lc := b.lenFor(parent.FontSize, ctx)
	if l.Unit == UnitPercent {
		return l.Value / 100 * parent.FontSize, true
	}
	px, ok := lc.px(l, axisX)
	if !ok || math.IsNaN(px) {
		return 0, false
	}
	return px, true
}
