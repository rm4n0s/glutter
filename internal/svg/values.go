package svg

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/ui"
)

// Unit is the unit of a Length.
//
// SVG 1.1: types.html#DataTypeLength
type Unit uint8

// The units of lengths. After parsing, a Length in the tree has the unit UnitPx (a number of user units) or
// UnitPercent; the others are converted when the length is parsed (see the package documentation).
const (
	// UnitPx is a user unit: a plain number, or a length in px.
	UnitPx Unit = iota
	// UnitPercent is a percentage of the reference length of the attribute, as Value/100 of it.
	UnitPercent
	// UnitPt is 1/72 inch.
	UnitPt
	// UnitPc is 1/6 inch.
	UnitPc
	// UnitMm is a millimeter.
	UnitMm
	// UnitCm is a centimeter.
	UnitCm
	// UnitIn is an inch, 96 user units.
	UnitIn
	// UnitEm is the computed font size of the element.
	UnitEm
	// UnitEx is the x-height of the element's font.
	UnitEx
	// UnitRem is the font size of the root.
	UnitRem
)

// Length is a number with a unit.
//
// SVG 1.1: types.html#DataTypeLength
type Length struct {
	Value float64
	Unit  Unit
}

// Resolve returns the length in user units. ref is what a percentage is a percentage of (the width of the viewport,
// or 1 for the fraction of a bounding box); a length with another unit than a percentage ignores it. A relative
// unit that is still in the length (em, ex, rem) resolves as the parser would with the default theme, which only
// happens for a Length made by ParseLength.
//
// SVG 1.1: coords.html#Units
func (l Length) Resolve(ref float64) float64 {
	switch l.Unit {
	case UnitPercent:
		return l.Value / 100 * ref
	case UnitPt:
		return l.Value * 96 / 72
	case UnitPc:
		return l.Value * 16
	case UnitMm:
		return l.Value * 96 / 25.4
	case UnitCm:
		return l.Value * 96 / 2.54
	case UnitIn:
		return l.Value * 96
	case UnitEm, UnitRem:
		return l.Value * 14
	case UnitEx:
		return l.Value * 7
	}
	return l.Value
}

// ParseLength parses a length with any unit (a number, or a number with px, pt, pc, mm, cm, in, em, ex, rem or %).
// White space around it is allowed.
//
// SVG 1.1: types.html#DataTypeLength
func ParseLength(s string) (Length, bool) {
	i := skipSpace(s, 0)
	l, j, ok := scanLength(s, i)
	if !ok || skipSpace(s, j) != len(s) {
		return Length{}, false
	}
	return l, true
}

// scanLength reads a length at s[i:].
func scanLength(s string, i int) (Length, int, bool) {
	v, j, ok := scanNumber(s, i)
	if !ok {
		return Length{}, i, false
	}
	u := UnitPx
	rest := s[j:]
	switch {
	case strings.HasPrefix(rest, "%"):
		u, j = UnitPercent, j+1
	case len(rest) >= 2:
		switch strings.ToLower(rest[:2]) {
		case "px":
			j += 2
		case "pt":
			u, j = UnitPt, j+2
		case "pc":
			u, j = UnitPc, j+2
		case "mm":
			u, j = UnitMm, j+2
		case "cm":
			u, j = UnitCm, j+2
		case "in":
			u, j = UnitIn, j+2
		case "em":
			u, j = UnitEm, j+2
		case "ex":
			u, j = UnitEx, j+2
		default:
			if len(rest) >= 3 && strings.EqualFold(rest[:3], "rem") {
				u, j = UnitRem, j+3
			}
		}
	}
	// A letter after the number that is not a unit makes the length invalid.
	if j < len(s) && (s[j] >= 'a' && s[j] <= 'z' || s[j] >= 'A' && s[j] <= 'Z' || s[j] == '%') {
		return Length{}, i, false
	}
	return Length{v, u}, j, true
}

// axis is the dimension that a percentage is a percentage of.
type axis uint8

const (
	axisX axis = iota
	axisY
	// axisXY is the normalized diagonal, sqrt((w²+h²)/2), for lengths such as r and stroke-width.
	axisXY
)

// lenCtx resolves the lengths of one element: its font size and the viewport around it.
//
// SVG 1.1: coords.html#Units
type lenCtx struct {
	fontSize float64
	rootFont float64
	xRatio   float64 // the x-height as a fraction of the font size
	vw, vh   float64
	vpKnown  bool
}

// px converts a length to user units; false if it is a percentage and the viewport is unknown.
func (c *lenCtx) px(l Length, a axis) (float64, bool) {
	switch l.Unit {
	case UnitPx:
		return l.Value, true
	case UnitEm:
		return l.Value * c.fontSize, true
	case UnitEx:
		return l.Value * c.fontSize * c.xRatio, true
	case UnitRem:
		return l.Value * c.rootFont, true
	case UnitPercent:
		if !c.vpKnown {
			return 0, false
		}
		switch a {
		case axisX:
			return l.Value / 100 * c.vw, true
		case axisY:
			return l.Value / 100 * c.vh, true
		}
		return l.Value / 100 * math.Sqrt((c.vw*c.vw+c.vh*c.vh)/2), true
	}
	return l.Resolve(0), true
}

// keep converts a length to user units unless it is a percentage, which stays for the painter. Lengths that depend on
// the object a resource is applied to (gradient coordinates, mask and filter regions) are kept this way.
func (c *lenCtx) keep(l Length) Length {
	if l.Unit == UnitPercent {
		return l
	}
	v, _ := c.px(l, axisX)
	return Length{v, UnitPx}
}

// ViewBox is the user-space rectangle that a viewport shows.
//
// SVG 1.1: coords.html#ViewBoxAttribute
type ViewBox struct {
	X, Y, Width, Height float64
}

// Rect returns the view box as a ui.Rect.
func (v ViewBox) Rect() ui.Rect { return ui.NewRectFromLTWH(v.X, v.Y, v.Width, v.Height) }

// Align is the alignment part of preserveAspectRatio. The zero value is xMidYMid, the default.
//
// SVG 1.1: coords.html#PreserveAspectRatioAttribute
type Align uint8

// The alignments of preserveAspectRatio.
const (
	// AlignXMidYMid is xMidYMid, the default.
	AlignXMidYMid Align = iota
	// AlignNone is none: the content is stretched to the viewport.
	AlignNone
	// AlignXMinYMin is xMinYMin.
	AlignXMinYMin
	// AlignXMidYMin is xMidYMin.
	AlignXMidYMin
	// AlignXMaxYMin is xMaxYMin.
	AlignXMaxYMin
	// AlignXMinYMid is xMinYMid.
	AlignXMinYMid
	// AlignXMaxYMid is xMaxYMid.
	AlignXMaxYMid
	// AlignXMinYMax is xMinYMax.
	AlignXMinYMax
	// AlignXMidYMax is xMidYMax.
	AlignXMidYMax
	// AlignXMaxYMax is xMaxYMax.
	AlignXMaxYMax
)

var alignNames = map[string]Align{
	"none": AlignNone, "xminymin": AlignXMinYMin, "xmidymin": AlignXMidYMin, "xmaxymin": AlignXMaxYMin,
	"xminymid": AlignXMinYMid, "xmidymid": AlignXMidYMid, "xmaxymid": AlignXMaxYMid,
	"xminymax": AlignXMinYMax, "xmidymax": AlignXMidYMax, "xmaxymax": AlignXMaxYMax,
}

// PreserveAspectRatio says how a viewBox fits its viewport. The zero value is "xMidYMid meet".
//
// SVG 1.1: coords.html#PreserveAspectRatioAttribute
type PreserveAspectRatio struct {
	Align Align
	// Slice is "slice" (the viewBox covers the viewport); false is "meet" (the viewBox fits in it).
	Slice bool
}

// ParsePreserveAspectRatio parses the value of preserveAspectRatio: an optional "defer", the alignment, and an
// optional "meet" or "slice". The text is case sensitive in SVG, but the alignment is matched without regard to case
// here, as browsers' tolerance goes no further than that.
//
// SVG 1.1: coords.html#PreserveAspectRatioAttribute
func ParsePreserveAspectRatio(s string) (PreserveAspectRatio, bool) {
	f := strings.Fields(s)
	if len(f) > 0 && f[0] == "defer" {
		f = f[1:]
	}
	if len(f) == 0 || len(f) > 2 {
		return PreserveAspectRatio{}, false
	}
	a, ok := alignNames[strings.ToLower(f[0])]
	if !ok {
		return PreserveAspectRatio{}, false
	}
	par := PreserveAspectRatio{Align: a}
	if len(f) == 2 {
		switch f[1] {
		case "meet":
		case "slice":
			par.Slice = true
		default:
			return PreserveAspectRatio{}, false
		}
	}
	return par, true
}

// parseViewBox parses a viewBox attribute. A value that is not four numbers is ignored (ok false, no error). A
// negative width or height is an error, as the specification says; a width or height of zero is valid and disables
// the rendering of the element.
//
// SVG 1.1: coords.html#ViewBoxAttribute
func parseViewBox(s string) (vb ViewBox, ok bool, err error) {
	n, good := ParseNumberList(s)
	if !good || len(n) != 4 {
		return ViewBox{}, false, nil
	}
	if n[2] < 0 || n[3] < 0 {
		return ViewBox{}, false, errors.New(TagSvgInvalidValue, "a viewBox has a negative width or height", "attribute", "viewBox", "value", s)
	}
	return ViewBox{n[0], n[1], n[2], n[3]}, true, nil
}

// parseURLRef reads "url(#id)" at the start of s (white space and quotes inside the parentheses are allowed) and
// returns the id and the text after the closing parenthesis.
//
// SVG 1.1: linking.html#IRIandURI
func parseURLRef(s string) (id, rest string, ok bool) {
	s = strings.TrimLeft(s, " \t\r\n\f")
	if len(s) < 5 || !strings.EqualFold(s[:4], "url(") {
		return "", s, false
	}
	end := strings.IndexByte(s, ')')
	if end < 0 {
		return "", s, false
	}
	inner := strings.TrimSpace(s[4:end])
	if len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') && inner[len(inner)-1] == inner[0] {
		inner = strings.TrimSpace(inner[1 : len(inner)-1])
	}
	if !strings.HasPrefix(inner, "#") {
		return "", s[end+1:], false
	}
	return inner[1:], s[end+1:], true
}

// hrefID returns the id in a reference written as "#id", the value of href.
func hrefID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) > 1 && s[0] == '#' {
		return s[1:], true
	}
	return "", false
}

// ParseColor parses a color: a keyword, #rgb, #rgba, #rrggbb, #rrggbbaa, rgb()/rgba(), hsl()/hsla() or
// "transparent". currentColor is reported by the second result and has no color.
//
// SVG 1.1: types.html#DataTypeColor; SVG 2: painting.html#SpecifyingPaint
func ParseColor(s string) (c ui.Color, current, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ui.Color{}, false, false
	}
	if s[0] == '#' {
		c, ok = parseHexColor(s[1:])
		return c, false, ok
	}
	lower := strings.ToLower(s)
	if lower == "currentcolor" {
		return ui.Color{}, true, true
	}
	if lower == "transparent" {
		return ui.NewColorFromARGB(0, 0, 0, 0), false, true
	}
	if v, found := namedColors[lower]; found {
		return ui.NewColorFromARGB(255, int(v>>16&0xff), int(v>>8&0xff), int(v&0xff)), false, true
	}
	open := strings.IndexByte(lower, '(')
	if open < 0 || !strings.HasSuffix(lower, ")") {
		return ui.Color{}, false, false
	}
	name := strings.TrimSpace(lower[:open])
	args := strings.FieldsFunc(lower[open+1:len(lower)-1], func(r rune) bool { return r == ',' || r == '/' || r == ' ' || r == '\t' || r == '\n' || r == '\r' })
	switch name {
	case "rgb", "rgba":
		c, ok = parseRGBArgs(args)
	case "hsl", "hsla":
		c, ok = parseHSLArgs(args)
	}
	return c, false, ok
}

func parseHexColor(h string) (ui.Color, bool) {
	var d [8]int
	n := len(h)
	if n != 3 && n != 4 && n != 6 && n != 8 {
		return ui.Color{}, false
	}
	for i := 0; i < n; i++ {
		c := h[i]
		switch {
		case c >= '0' && c <= '9':
			d[i] = int(c - '0')
		case c >= 'a' && c <= 'f':
			d[i] = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d[i] = int(c-'A') + 10
		default:
			return ui.Color{}, false
		}
	}
	switch n {
	case 3:
		return ui.NewColorFromARGB(255, d[0]*17, d[1]*17, d[2]*17), true
	case 4:
		return ui.NewColorFromARGB(d[3]*17, d[0]*17, d[1]*17, d[2]*17), true
	case 6:
		return ui.NewColorFromARGB(255, d[0]*16+d[1], d[2]*16+d[3], d[4]*16+d[5]), true
	}
	return ui.NewColorFromARGB(d[6]*16+d[7], d[0]*16+d[1], d[2]*16+d[3], d[4]*16+d[5]), true
}

// colorChannel reads a color channel: a number of 0 to 255, or a percentage.
func colorChannel(s string) (int, bool) {
	v, ok := parseColorNumber(s, 255)
	if !ok {
		return 0, false
	}
	return int(math.Round(math.Min(255, math.Max(0, v)))), true
}

// parseColorNumber reads a number or a percentage; a percentage is of full.
func parseColorNumber(s string, full float64) (float64, bool) {
	v, j, ok := scanNumber(s, 0)
	if !ok {
		return 0, false
	}
	if j < len(s) && s[j] == '%' {
		return v / 100 * full, j+1 == len(s)
	}
	return v, j == len(s)
}

func parseAlpha(s string) (int, bool) {
	v, ok := parseColorNumber(s, 1)
	if !ok {
		return 0, false
	}
	return int(math.Round(clamp01(v) * 255)), true
}

func parseRGBArgs(a []string) (ui.Color, bool) {
	if len(a) != 3 && len(a) != 4 {
		return ui.Color{}, false
	}
	var ch [3]int
	for i := 0; i < 3; i++ {
		v, ok := colorChannel(a[i])
		if !ok {
			return ui.Color{}, false
		}
		ch[i] = v
	}
	alpha := 255
	if len(a) == 4 {
		var ok bool
		if alpha, ok = parseAlpha(a[3]); !ok {
			return ui.Color{}, false
		}
	}
	return ui.NewColorFromARGB(alpha, ch[0], ch[1], ch[2]), true
}

func parseHSLArgs(a []string) (ui.Color, bool) {
	if len(a) != 3 && len(a) != 4 {
		return ui.Color{}, false
	}
	h, j, ok := scanNumber(a[0], 0)
	if !ok {
		return ui.Color{}, false
	}
	switch a[0][j:] {
	case "", "deg":
	case "rad":
		h = h * 180 / math.Pi
	case "grad":
		h = h * 360 / 400
	case "turn":
		h *= 360
	default:
		return ui.Color{}, false
	}
	sat, ok1 := parseColorNumber(a[1], 100)
	light, ok2 := parseColorNumber(a[2], 100)
	if !ok1 || !ok2 {
		return ui.Color{}, false
	}
	s, l := clamp01(sat/100), clamp01(light/100)
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	h /= 360
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	r, g, b := hue(h+1.0/3), hue(h), hue(h-1.0/3)
	alpha := 255
	if len(a) == 4 {
		if alpha, ok = parseAlpha(a[3]); !ok {
			return ui.Color{}, false
		}
	}
	return ui.NewColorFromARGB(alpha, int(math.Round(r*255)), int(math.Round(g*255)), int(math.Round(b*255))), true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagSvgInvalidValue is the tag of the error for a value that must be valid and is not, such as a viewBox with a
	// negative size; the metadata are "attribute" and "value". Invalid values of other attributes are ignored.
	TagSvgInvalidValue = "SvgInvalidValue"
)
