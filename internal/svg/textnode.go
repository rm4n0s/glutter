package svg

import (
	"strings"
)

// buildText makes the node of a text element: its spans, its positions and the strings with the white space
// handled. Nothing is laid out here (that is the painter's work).
//
// SVG 1.1: text.html#TextElement
func (b *builder) buildText(e *element, st *Style, np *nodeProps, ctx *buildCtx) *Node {
	n := b.newNode(NodeText, e, st, np, ctx)
	ws := &wsState{prevSpace: true}
	preserve := e.attrOr("xml:space", "") == "preserve"
	n.Text = b.buildSpan(e, st, ctx, preserve, ws)
	if !preserve {
		trimTrailingSpace(n.Text)
	}
	return n
}

// wsState is the state of the white space collapsing across the whole text element.
type wsState struct {
	prevSpace bool
}

// buildSpan reads the content of a text, tspan, textPath or tref.
func (b *builder) buildSpan(e *element, st *Style, ctx *buildCtx, preserve bool, ws *wsState) *TextSpan {
	lc := b.lenFor(st.FontSize, ctx)
	s := &TextSpan{Name: e.name, Style: st}
	s.X = lengthList(e, "x", &lc, axisX)
	s.Y = lengthList(e, "y", &lc, axisY)
	s.Dx = lengthList(e, "dx", &lc, axisX)
	s.Dy = lengthList(e, "dy", &lc, axisY)
	if v, ok := e.attr("rotate"); ok {
		if nums, ok := ParseNumberList(v); ok {
			s.Rotate = nums
		}
	}
	if tl, ok := optLengthAttr(e, "textLength", &lc, axisX); ok && tl > 0 {
		s.TextLength = tl
		s.AdjustGlyphs = strings.TrimSpace(e.attrOr("lengthAdjust", "")) == "spacingAndGlyphs"
	}
	if e.name == "textPath" {
		b.textPathData(e, s, &lc)
	}
	if v, ok := e.attr("xml:space"); ok {
		preserve = v == "preserve"
	}
	for _, c := range e.children {
		switch c.name {
		case "#text":
			if t := collapseSpace(c.text, preserve, ws); t != "" {
				s.Items = append(s.Items, TextItem{Text: t})
			}
		case "tspan", "textPath", "a":
			if !b.conditionsHold(c) {
				continue
			}
			cst, cnp := b.computeStyle(c, b.sheet.cascade(c), st, ctx)
			if !cnp.display {
				continue
			}
			child := b.buildSpan(c, cst, ctx, preserve, ws)
			if c.name == "a" {
				child.Name = "tspan"
			}
			s.Items = append(s.Items, TextItem{Span: child})
		case "tref":
			href, ok := c.href()
			if !ok {
				continue
			}
			id, ok := hrefID(href)
			if !ok || b.ids[id] == nil {
				continue
			}
			if t := collapseSpace(allText(b.ids[id]), preserve, ws); t != "" {
				cst, _ := b.computeStyle(c, b.sheet.cascade(c), st, ctx)
				s.Items = append(s.Items, TextItem{Span: &TextSpan{Name: "tref", Style: cst, Items: []TextItem{{Text: t}}}})
			}
		}
	}
	return s
}

// allText returns the text of an element and its descendants.
func allText(e *element) string {
	var sb strings.Builder
	var walk func(*element)
	walk = func(x *element) {
		for _, c := range x.children {
			if c.name == "#text" {
				sb.WriteString(c.text)
			} else {
				walk(c)
			}
		}
	}
	walk(e)
	return sb.String()
}

// textPathData sets the path of a textPath from its href.
func (b *builder) textPathData(e *element, s *TextSpan, lc *lenCtx) {
	if href, ok := e.href(); ok {
		if id, ok := hrefID(href); ok {
			if target := b.ids[id]; target != nil && target.name == "path" {
				tlc := b.lenFor(b.styleOf(target).FontSize, &b.rootCtx)
				if p, _ := shapePath(target, &tlc); p != nil {
					if v, ok := target.attr("transform"); ok {
						if t, ok := ParseTransform(v); ok && !t.IsIdentity() {
							p = p.Transform(t.Matrix4())
						}
					}
					s.PathData = p
				}
			}
		}
	}
	if v, ok := e.attr("startOffset"); ok {
		if l, ok := ParseLength(v); ok {
			if l.Unit == UnitPercent {
				s.StartOffset, s.StartOffsetFraction = l.Value/100, true
			} else if px, ok := lc.px(l, axisX); ok {
				s.StartOffset = px
			}
		}
	}
}

// lengthList reads a list of lengths separated by white space and commas; an invalid list is dropped.
func lengthList(e *element, name string, lc *lenCtx, a axis) []float64 {
	v, ok := e.attr(name)
	if !ok {
		return nil
	}
	var out []float64
	i := skipSpace(v, 0)
	for i < len(v) {
		l, j, ok := scanLength(v, i)
		if !ok {
			return nil
		}
		px, ok := lc.px(l, a)
		if !ok {
			return nil
		}
		out = append(out, px)
		i = skipCommaSpace(v, j)
	}
	return out
}

// collapseSpace handles the white space of a string. In the default mode a run of white space (newlines and tabs
// count as spaces) becomes one space and the space at the start of the text element is dropped; with
// xml:space="preserve" each white space character becomes one space.
//
// SVG 1.1: text.html#WhiteSpace
func collapseSpace(s string, preserve bool, ws *wsState) string {
	if s == "" {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		isWS := c == ' ' || c == '\t' || c == '\n' || c == '\r'
		switch {
		case !isWS:
			sb.WriteByte(c)
			ws.prevSpace = false
		case preserve:
			sb.WriteByte(' ')
		case !ws.prevSpace:
			sb.WriteByte(' ')
			ws.prevSpace = true
		}
	}
	return sb.String()
}

// trimTrailingSpace removes the space at the end of the last string of a text element.
func trimTrailingSpace(s *TextSpan) bool {
	for i := len(s.Items) - 1; i >= 0; i-- {
		it := &s.Items[i]
		if it.Span != nil {
			if trimTrailingSpace(it.Span) {
				return true
			}
			continue
		}
		t := strings.TrimRight(it.Text, " ")
		if t == "" {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			continue
		}
		it.Text = t
		return true
	}
	return false
}
