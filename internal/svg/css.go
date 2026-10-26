package svg

import (
	"sort"
	"strings"
)

// A declaration is one "property: value" of a style attribute or a rule.
type declaration struct {
	name      string
	value     string
	important bool
}

// stripComments removes /* */ comments and the <!-- --> markers that old style sheets are wrapped in.
func stripComments(s string) string {
	if !strings.Contains(s, "/*") && !strings.Contains(s, "<!--") && !strings.Contains(s, "-->") {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); {
		switch {
		case strings.HasPrefix(s[i:], "/*"):
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return sb.String()
			}
			i += end + 4
			sb.WriteByte(' ')
		case strings.HasPrefix(s[i:], "<!--"):
			i += 4
		case strings.HasPrefix(s[i:], "-->"):
			i += 3
		default:
			sb.WriteByte(s[i])
			i++
		}
	}
	return sb.String()
}

// splitTop splits s at the byte sep where it is not inside parentheses, brackets or quotes.
func splitTop(s string, sep byte) []string {
	var out []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// parseDeclarations parses the declarations of a style attribute or the body of a rule. A declaration that has no
// colon is dropped. Custom properties (--name) are ignored: var() is not supported.
//
// CSS 2.1: syndata.html#declaration
func parseDeclarations(s string) []declaration {
	s = stripComments(s)
	var out []declaration
	for _, part := range splitTop(s, ';') {
		colon := strings.IndexByte(part, ':')
		if colon < 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(part[:colon]))
		value := strings.TrimSpace(part[colon+1:])
		if name == "" || value == "" || strings.HasPrefix(name, "--") {
			continue
		}
		important := false
		if i := strings.LastIndexByte(value, '!'); i >= 0 && strings.EqualFold(strings.TrimSpace(value[i+1:]), "important") {
			important = true
			value = strings.TrimSpace(value[:i])
		}
		if value == "" {
			continue
		}
		out = append(out, declaration{name, value, important})
	}
	return out
}

// A compound is a simple selector: an optional type, an optional id and any number of classes.
type compound struct {
	tag     string // "" or "*" for any
	id      string
	classes []string
}

// A selector is compounds joined by combinators, from the left: combs[i] joins compounds[i] and compounds[i+1] and
// is ' ' (descendant) or '>' (child).
type selector struct {
	compounds []compound
	combs     []byte
}

// specificity returns the specificity of the selector as a·2^16 + b·2^8 + c, with a the number of ids, b of classes
// and c of types.
//
// CSS 2.1: cascade.html#specificity
func (s *selector) specificity() uint32 {
	var a, b, c uint32
	for _, cp := range s.compounds {
		if cp.id != "" {
			a++
		}
		b += uint32(len(cp.classes))
		if cp.tag != "" && cp.tag != "*" {
			c++
		}
	}
	return a<<16 | b<<8 | c
}

// parseSelector parses a selector of the supported subset (type, class, id, universal, descendant and child
// combinators). It returns false for anything else, which makes the whole selector (not the rule) unused.
//
// CSS 2.1: selector.html
func parseSelector(s string) (selector, bool) {
	var sel selector
	s = strings.TrimSpace(s)
	if s == "" {
		return sel, false
	}
	i := 0
	for i < len(s) {
		var cp compound
		start := i
		for i < len(s) {
			c := s[i]
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' {
				break
			}
			switch c {
			case '#', '.':
				j := i + 1
				for j < len(s) && isIdentByte(s[j]) {
					j++
				}
				if j == i+1 {
					return sel, false
				}
				if c == '#' {
					if cp.id != "" {
						return sel, false
					}
					cp.id = s[i+1 : j]
				} else {
					cp.classes = append(cp.classes, s[i+1:j])
				}
				i = j
			case '*':
				if i != start {
					return sel, false
				}
				cp.tag = "*"
				i++
			default:
				if !isIdentByte(c) || i != start {
					return sel, false
				}
				j := i
				for j < len(s) && isIdentByte(s[j]) {
					j++
				}
				cp.tag = s[i:j]
				i = j
			}
		}
		if i == start {
			return sel, false
		}
		sel.compounds = append(sel.compounds, cp)
		// A combinator, or the end.
		comb := byte(0)
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '>') {
			if s[i] == '>' {
				if comb == '>' {
					return sel, false
				}
				comb = '>'
			} else if comb == 0 {
				comb = ' '
			}
			i++
		}
		if i >= len(s) {
			if comb == '>' {
				return sel, false
			}
			break
		}
		if comb == 0 {
			return sel, false
		}
		sel.combs = append(sel.combs, comb)
	}
	return sel, len(sel.compounds) > 0
}

func isIdentByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c >= 0x80
}

// matchCompound reports whether an element matches a compound.
func matchCompound(cp *compound, e *element) bool {
	if cp.tag != "" && cp.tag != "*" && cp.tag != e.name {
		return false
	}
	if cp.id != "" && cp.id != e.id {
		return false
	}
	if len(cp.classes) > 0 {
		cls, _ := e.attr("class")
		for _, want := range cp.classes {
			if !hasClass(cls, want) {
				return false
			}
		}
	}
	return true
}

// hasClass reports whether the white-space separated list contains the class.
func hasClass(list, want string) bool {
	for {
		list = strings.TrimLeft(list, " \t\r\n\f")
		if list == "" {
			return false
		}
		end := strings.IndexAny(list, " \t\r\n\f")
		if end < 0 {
			end = len(list)
		}
		if list[:end] == want {
			return true
		}
		list = list[end:]
	}
}

// matches reports whether the element matches the selector, matching the compounds from the right.
func (s *selector) matches(e *element) bool { return s.matchFrom(len(s.compounds)-1, e) }

func (s *selector) matchFrom(i int, e *element) bool {
	if !matchCompound(&s.compounds[i], e) {
		return false
	}
	if i == 0 {
		return true
	}
	switch s.combs[i-1] {
	case '>':
		return e.parent != nil && s.matchFrom(i-1, e.parent)
	default:
		for p := e.parent; p != nil; p = p.parent {
			if s.matchFrom(i-1, p) {
				return true
			}
		}
	}
	return false
}

// A rule is a selector with the declarations of its block.
type rule struct {
	sel   selector
	spec  uint32
	order int
	decls []declaration
}

// stylesheet holds the rules of all <style> elements, indexed by the rightmost compound of the selector so that an
// element looks only at the rules that can match it.
type stylesheet struct {
	byID      map[string][]*rule
	byClass   map[string][]*rule
	byTag     map[string][]*rule
	universal []*rule
	order     int
}

// add parses a style sheet and adds its rules. The at-rules are skipped: @import, @media (with its block),
// @font-face, @keyframes and the rest. A rule with a selector that is not supported is skipped for that selector.
//
// CSS 2.1: syndata.html
func (sh *stylesheet) add(src string) {
	s := stripComments(src)
	i := 0
	for i < len(s) {
		i = skipCSSSpace(s, i)
		if i >= len(s) {
			break
		}
		if s[i] == '@' {
			i = skipAtRule(s, i)
			continue
		}
		open := strings.IndexByte(s[i:], '{')
		if open < 0 {
			break
		}
		prelude := s[i : i+open]
		end := findBlockEnd(s, i+open)
		body := s[i+open+1 : end]
		if end < len(s) {
			end++
		}
		i = end
		decls := parseDeclarations(body)
		if len(decls) == 0 {
			continue
		}
		for _, one := range splitTop(prelude, ',') {
			sel, ok := parseSelector(one)
			if !ok {
				continue
			}
			sh.order++
			sh.insert(&rule{sel: sel, spec: sel.specificity(), order: sh.order, decls: decls})
		}
	}
}

func (sh *stylesheet) insert(r *rule) {
	last := &r.sel.compounds[len(r.sel.compounds)-1]
	switch {
	case last.id != "":
		if sh.byID == nil {
			sh.byID = map[string][]*rule{}
		}
		sh.byID[last.id] = append(sh.byID[last.id], r)
	case len(last.classes) > 0:
		if sh.byClass == nil {
			sh.byClass = map[string][]*rule{}
		}
		sh.byClass[last.classes[0]] = append(sh.byClass[last.classes[0]], r)
	case last.tag != "" && last.tag != "*":
		if sh.byTag == nil {
			sh.byTag = map[string][]*rule{}
		}
		sh.byTag[last.tag] = append(sh.byTag[last.tag], r)
	default:
		sh.universal = append(sh.universal, r)
	}
}

func skipCSSSpace(s string, i int) int {
	for i < len(s) && (isSpace(s[i])) {
		i++
	}
	return i
}

// skipAtRule returns the index after the at-rule that starts at s[i]: up to a semicolon, or after its block.
func skipAtRule(s string, i int) int {
	depth := 0
	var quote byte
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ';' && depth == 0:
			return i + 1
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth <= 0 {
				return i + 1
			}
		}
	}
	return len(s)
}

// findBlockEnd returns the index of the } that closes the { at s[open], or len(s).
func findBlockEnd(s string, open int) int {
	depth := 0
	var quote byte
	for i := open; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

// The origins of the cascade, from the weakest.
const (
	originPresentation = iota
	originSheet
	originInline
)

// ranked is a declaration with its place in the cascade.
type ranked struct {
	declaration
	rank uint64
}

func makeRank(important bool, origin int, spec uint32, order int) uint64 {
	var r uint64
	if important {
		r = 1 << 62
	}
	return r | uint64(origin)<<60 | uint64(spec)<<32 | uint64(uint32(order))
}

// collect adds the declarations of the sheet rules that match the element.
func (sh *stylesheet) collect(e *element, out []ranked) []ranked {
	emit := func(rules []*rule) {
		for _, r := range rules {
			if !r.sel.matches(e) {
				continue
			}
			for _, d := range r.decls {
				out = append(out, ranked{d, makeRank(d.important, originSheet, r.spec, r.order)})
			}
		}
	}
	if sh == nil {
		return out
	}
	if e.id != "" {
		emit(sh.byID[e.id])
	}
	if len(sh.byClass) > 0 {
		if cls, ok := e.attr("class"); ok {
			for _, c := range strings.Fields(cls) {
				emit(sh.byClass[c])
			}
		}
	}
	if len(sh.byTag) > 0 {
		emit(sh.byTag[e.name])
	}
	emit(sh.universal)
	return out
}

// presentationProps are the properties that can be written as attributes (presentation attributes).
//
// SVG 1.1: styling.html#SVGStylingProperties
var presentationProps = map[string]bool{
	"fill": true, "fill-opacity": true, "fill-rule": true, "stroke": true, "stroke-width": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-miterlimit": true, "stroke-dasharray": true,
	"stroke-dashoffset": true, "stroke-opacity": true, "opacity": true, "color": true, "display": true,
	"visibility": true, "clip-path": true, "clip-rule": true, "mask": true, "filter": true,
	"font-family": true, "font-size": true, "font-weight": true, "font-style": true, "font-variant": true,
	"text-anchor": true, "text-decoration": true, "letter-spacing": true, "word-spacing": true,
	"color-interpolation-filters": true, "color-interpolation": true, "stop-color": true, "stop-opacity": true,
	"flood-color": true, "flood-opacity": true, "lighting-color": true, "marker-start": true, "marker-mid": true,
	"marker-end": true, "overflow": true, "vector-effect": true, "paint-order": true, "shape-rendering": true,
	"text-rendering": true, "image-rendering": true, "mask-type": true, "transform": true, "direction": true,
	"dominant-baseline": true, "alignment-baseline": true, "baseline-shift": true, "writing-mode": true,
	"mix-blend-mode": true, "isolation": true,
}

// declList is the declarations that apply to one element, weakest first. Applying them in order gives the cascade's
// winner for every property.
type declList []ranked

// get returns the winning value of a property.
func (l declList) get(name string) (string, bool) {
	for i := len(l) - 1; i >= 0; i-- {
		if l[i].name == name {
			return l[i].value, true
		}
	}
	return "", false
}

// cascade returns the declarations that apply to the element: the presentation attributes, the rules of the sheet and
// the style attribute, ordered by the cascade of CSS: !important first, then the origin, the specificity and the
// order in the sheet.
//
// CSS 2.1: cascade.html#cascade; SVG 1.1: styling.html#UsingPresentationAttributes
func (sh *stylesheet) cascade(e *element) declList {
	var out []ranked
	for _, a := range e.attrs {
		if presentationProps[a.name] {
			out = append(out, ranked{declaration{name: a.name, value: strings.TrimSpace(a.value)}, makeRank(false, originPresentation, 0, 0)})
		}
	}
	out = sh.collect(e, out)
	if st, ok := e.attr("style"); ok {
		for i, d := range parseDeclarations(st) {
			out = append(out, ranked{d, makeRank(d.important, originInline, 0, i)})
		}
	}
	out = expandShorthands(out)
	if len(out) > 1 {
		sort.SliceStable(out, func(i, j int) bool { return out[i].rank < out[j].rank })
	}
	return out
}

// expandShorthands adds the longhands of the shorthands that SVG styles use: marker and the font shorthand is not
// supported.
func expandShorthands(in []ranked) []ranked {
	n := len(in)
	for i := 0; i < n; i++ {
		if in[i].name == "marker" {
			for _, p := range [...]string{"marker-start", "marker-mid", "marker-end"} {
				in = append(in, ranked{declaration{p, in[i].value, in[i].important}, in[i].rank})
			}
		}
	}
	return in
}
