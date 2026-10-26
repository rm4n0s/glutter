package svg

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestParseSelector(t *testing.T) {
	tests := []struct {
		in   string
		ok   bool
		spec uint32
	}{
		{"rect", true, 1},
		{"*", true, 0},
		{".a", true, 1 << 8},
		{"#a", true, 1 << 16},
		{"rect.a.b", true, 2<<8 | 1},
		{"g rect", true, 2},
		{"g > rect", true, 2},
		{"g>rect", true, 2},
		{"#a .b rect", true, 1<<16 | 1<<8 | 1},
		{"  rect  ", true, 1},
		{"rect:hover", false, 0},
		{"rect:first-child", false, 0},
		{"rect[x]", false, 0},
		{"a + b", false, 0},
		{"a ~ b", false, 0},
		{"a >", false, 0},
		{"> a", false, 0},
		{"a > > b", false, 0},
		{"svg|rect", false, 0},
		{"", false, 0},
		{".", false, 0},
		{"#", false, 0},
	}
	for _, tt := range tests {
		sel, ok := parseSelector(tt.in)
		if ok != tt.ok {
			t.Errorf("parseSelector(%q) ok = %v, want %v", tt.in, ok, tt.ok)
			continue
		}
		if ok && sel.specificity() != tt.spec {
			t.Errorf("parseSelector(%q) specificity = %#x, want %#x", tt.in, sel.specificity(), tt.spec)
		}
	}
}

func TestParseDeclarations(t *testing.T) {
	tests := []struct {
		in   string
		want []declaration
	}{
		{"fill:red", []declaration{{"fill", "red", false}}},
		{" Fill : red ; stroke : blue ;", []declaration{{"fill", "red", false}, {"stroke", "blue", false}}},
		{"fill:red !important", []declaration{{"fill", "red", true}}},
		{"fill:red!IMPORTANT;", []declaration{{"fill", "red", true}}},
		{"fill:url(data:image/svg+xml;base64,AAAA)", []declaration{{"fill", "url(data:image/svg+xml;base64,AAAA)", false}}},
		{"font-family:'a;b', c", []declaration{{"font-family", "'a;b', c", false}}},
		{"/* c */ fill:red; /* d; e */ stroke:blue", []declaration{{"fill", "red", false}, {"stroke", "blue", false}}},
		{"nonsense; fill:red;; :x; y:", []declaration{{"fill", "red", false}}},
		{"--custom:1; fill:red", []declaration{{"fill", "red", false}}},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseDeclarations(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("parseDeclarations(%q) = %v, want %v", tt.in, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseDeclarations(%q)[%d] = %v, want %v", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

// The fill of the node "t" in `<svg><style>css</style>body</svg>` is want.
func TestCSSSelectorsAndCascade(t *testing.T) {
	tests := []struct {
		name string
		css  string
		body string
		want string
	}{
		{"type selector", `rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"class selector", `.a{fill:red}`, `<rect id="t" class="a" width="1" height="1"/>`, "red"},
		{"one of several classes", `.b{fill:red}`, `<rect id="t" class="a b c" width="1" height="1"/>`, "red"},
		{"two classes need both", `.a.z{fill:red}`, `<rect id="t" class="a b" width="1" height="1"/>`, "black"},
		{"id selector", `#t{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"universal", `*{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"selector list", `circle, rect, line{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"descendant", `g rect{fill:red}`, `<g><g><rect id="t" width="1" height="1"/></g></g>`, "red"},
		{"descendant does not match a sibling", `g rect{fill:red}`, `<g/><rect id="t" width="1" height="1"/>`, "black"},
		{"child", `g > rect{fill:red}`, `<g><rect id="t" width="1" height="1"/></g>`, "red"},
		{"child does not match a grandchild", `svg > rect{fill:red}`, `<g><rect id="t" width="1" height="1"/></g>`, "black"},
		{"descendant of a class", `.k rect{fill:red}`, `<g class="k"><g><rect id="t" width="1" height="1"/></g></g>`, "red"},
		{"unsupported selector is dropped, the rest of the list stays", `rect:first-child, .a{fill:red}`, `<rect id="t" class="a" width="1" height="1"/>`, "red"},
		{"unsupported rule does not hide the next rule", `rect:hover{fill:blue} rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"specificity: id over class", `#t{fill:red} .a{fill:blue}`, `<rect id="t" class="a" width="1" height="1"/>`, "red"},
		{"specificity: id over class, either order", `.a{fill:blue} #t{fill:red}`, `<rect id="t" class="a" width="1" height="1"/>`, "red"},
		{"specificity: class over type", `.a{fill:red} rect{fill:blue}`, `<rect id="t" class="a" width="1" height="1"/>`, "red"},
		{"specificity: more types win", `g rect{fill:red} rect{fill:blue}`, `<g><rect id="t" width="1" height="1"/></g>`, "red"},
		{"equal specificity: the later rule wins", `rect{fill:blue} rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"important over specificity", `rect{fill:red !important} #t{fill:blue}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"sheet over presentation attribute", `rect{fill:red}`, `<rect id="t" fill="blue" width="1" height="1"/>`, "red"},
		{"universal sheet rule over presentation attribute", `*{fill:red}`, `<rect id="t" fill="blue" width="1" height="1"/>`, "red"},
		{"style attribute over sheet", `#t{fill:blue}`, `<rect id="t" style="fill:red" width="1" height="1"/>`, "red"},
		{"important in the sheet over the style attribute", `rect{fill:red !important}`, `<rect id="t" style="fill:blue" width="1" height="1"/>`, "red"},
		{"important in the style attribute over important in the sheet", `rect{fill:blue !important}`, `<rect id="t" style="fill:red !important" width="1" height="1"/>`, "red"},
		{"style attribute over presentation attribute", ``, `<rect id="t" fill="blue" style="fill:red" width="1" height="1"/>`, "red"},
		{"presentation attribute is a default", ``, `<rect id="t" fill="red" width="1" height="1"/>`, "red"},
		{"an invalid value falls back to the weaker one", `rect{fill:notacolor}`, `<rect id="t" fill="red" width="1" height="1"/>`, "red"},
		{"at-rule media is skipped", `@media screen { rect{fill:blue} } rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"at-rule media skipped alone", `@media screen { rect{fill:blue} }`, `<rect id="t" width="1" height="1"/>`, "black"},
		{"at-rule import is skipped", `@import url(x.css); rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"at-rule font-face is skipped", `@font-face{font-family:x;src:url(a)} rect{fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"comments", `/* a { */ rect{ /* fill:blue; */ fill:red}`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"cdata-style markers", `<![CDATA[<!-- rect{fill:red} -->]]>`, `<rect id="t" width="1" height="1"/>`, "red"},
		{"shorthand marker", ``, `<rect id="t" style="marker:url(#m)" fill="red" width="1" height="1"/>`, "red"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<style>`+tt.css+`</style>`+tt.body+`</svg>`)
			n := mustFind(t, d, "t")
			if got := n.Style.Fill; got.Kind != PaintKindColor || got.Color != col(tt.want) {
				t.Errorf("fill = %+v, want %s", got, tt.want)
			}
		})
	}
}

func TestStyleElementTypeAndSeveralSheets(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<style type="text/css">.a{fill:red}</style>`+
		`<style type="text/x-other">.a{fill:blue}</style>`+
		`<style>.b{stroke:lime}</style>`+
		`<style><![CDATA[ .c > .d { fill: yellow } ]]></style>`+
		`<rect id="t" class="a b" width="1" height="1"/>`+
		`<g class="c"><rect id="u" class="d" width="1" height="1"/></g></svg>`)
	n := mustFind(t, d, "t")
	if n.Style.Fill.Color != col("red") || n.Style.Stroke.Color != col("lime") {
		t.Errorf("fill %v stroke %v", n.Style.Fill, n.Style.Stroke)
	}
	if u := mustFind(t, d, "u"); u.Style.Fill.Color != col("yellow") {
		t.Errorf("CDATA sheet: fill %v", u.Style.Fill)
	}
}

func TestInheritance(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<g id="g" fill="red" stroke="blue" stroke-width="3" opacity="0.5" fill-opacity="0.25" visibility="hidden" stroke-dasharray="4 2" font-weight="bold" stroke-linecap="round">`+
		`<rect id="a" width="1" height="1"/>`+
		`<rect id="b" width="1" height="1" fill="inherit" stroke="green"/>`+
		`<rect id="c" width="1" height="1" fill="initial" visibility="visible" font-weight="lighter"/>`+
		`<rect id="d" width="1" height="1" opacity="inherit" stroke-width="initial"/>`+
		`</g></svg>`)
	g := mustFind(t, d, "g")
	if g.Opacity != 0.5 {
		t.Errorf("g opacity = %v", g.Opacity)
	}
	a := mustFind(t, d, "a")
	checks := []struct {
		what string
		ok   bool
	}{
		{"fill inherited", a.Style.Fill.Color == col("red")},
		{"stroke inherited", a.Style.Stroke.Color == col("blue")},
		{"stroke-width inherited", a.Style.StrokeWidth == 3},
		{"fill-opacity inherited", a.Style.FillOpacity == 0.25},
		{"visibility inherited", !a.Style.Visible},
		{"dash inherited", len(a.Style.DashArray) == 2 && a.Style.DashArray[0] == 4},
		{"linecap inherited", a.Style.LineCap == LineCapRound},
		{"font-weight inherited", a.Style.FontWeight == 700},
		{"opacity does not inherit", a.Opacity == 1},
		{"the style is shared when nothing changes", a.Style == g.Style},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("%s", c.what)
		}
	}
	b := mustFind(t, d, "b")
	if b.Style.Fill.Color != col("red") || b.Style.Stroke.Color != col("green") {
		t.Errorf("inherit/override: fill %v stroke %v", b.Style.Fill, b.Style.Stroke)
	}
	c := mustFind(t, d, "c")
	if c.Style.Fill.Color != col("black") || !c.Style.Visible || c.Style.FontWeight != 400 {
		t.Errorf("initial: fill %v visible %v weight %d", c.Style.Fill, c.Style.Visible, c.Style.FontWeight)
	}
	dd := mustFind(t, d, "d")
	if dd.Style.StrokeWidth != 1 {
		t.Errorf("stroke-width initial = %v", dd.Style.StrokeWidth)
	}
}

func TestCurrentColorAndFontSize(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		theme     Theme
		wantFill  string
		wantFont  float64
		wantSpace float64
	}{
		{"currentColor is the theme's", `<rect id="t" fill="currentColor" width="1" height="1"/>`, Theme{CurrentColor: col("lime")}, "lime", 14, 0},
		{"default theme is black", `<rect id="t" fill="currentColor" width="1" height="1"/>`, Theme{}, "black", 14, 0},
		{"the color property sets it", `<g color="red"><rect id="t" fill="currentColor" width="1" height="1"/></g>`, Theme{CurrentColor: col("lime")}, "red", 14, 0},
		{"the color property on the element itself", `<rect id="t" color="blue" fill="currentColor" width="1" height="1"/>`, Theme{}, "blue", 14, 0},
		{"stroke currentColor in a style", `<rect id="t" style="fill:currentColor;color:red" width="1" height="1"/>`, Theme{}, "red", 14, 0},
		{"font size px", `<g font-size="20px"><rect id="t" width="1" height="1"/></g>`, Theme{}, "black", 20, 0},
		{"font size em is relative to the parent", `<g font-size="2em"><g font-size="1.5em"><rect id="t" width="1" height="1"/></g></g>`, Theme{FontSize: 10}, "black", 30, 0},
		{"font size percent", `<g font-size="150%"><rect id="t" width="1" height="1"/></g>`, Theme{FontSize: 10}, "black", 15, 0},
		{"font size keyword", `<g font-size="large"><rect id="t" width="1" height="1"/></g>`, Theme{FontSize: 10}, "black", 12, 0},
		{"font size larger", `<g font-size="larger"><rect id="t" width="1" height="1"/></g>`, Theme{FontSize: 10}, "black", 12, 0},
		{"font size ex", `<g font-size="2ex"><rect id="t" width="1" height="1"/></g>`, Theme{FontSize: 10, XHeight: 4}, "black", 8, 0},
		{"letter spacing in em uses the element's font size", `<g font-size="10px" letter-spacing="0.5em"><rect id="t" width="1" height="1"/></g>`, Theme{}, "black", 10, 5},
		{"an invalid font size is ignored", `<g font-size="-3"><rect id="t" width="1" height="1"/></g>`, Theme{FontSize: 10}, "black", 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.body+`</svg>`, ParseOpts{Theme: tt.theme})
			n := mustFind(t, d, "t")
			if n.Style.Fill.Color != col(tt.wantFill) {
				t.Errorf("fill = %v, want %s", n.Style.Fill.Color, tt.wantFill)
			}
			if n.Style.FontSize != tt.wantFont {
				t.Errorf("font size = %v, want %v", n.Style.FontSize, tt.wantFont)
			}
			if n.Style.LetterSpacing != tt.wantSpace {
				t.Errorf("letter spacing = %v, want %v", n.Style.LetterSpacing, tt.wantSpace)
			}
		})
	}
}

func TestPaintValues(t *testing.T) {
	tests := []struct {
		name     string
		fill     string
		wantKind PaintKind
		want     string
	}{
		{"none", "none", PaintKindNone, ""},
		{"color", "#0f0", PaintKindColor, "lime"},
		{"gradient", "url(#g)", PaintKindServer, ""},
		{"gradient with quotes and a space", "url( '#g' )", PaintKindServer, ""},
		{"missing server with a fallback", "url(#nope) red", PaintKindColor, "red"},
		{"missing server with none", "url(#nope) none", PaintKindNone, ""},
		{"missing server without a fallback", "url(#nope)", PaintKindNone, ""},
		{"existing server ignores the fallback", "url(#g) red", PaintKindServer, ""},
		{"icc color keeps the sRGB part", "#f00 icc-color(x, 1)", PaintKindColor, "red"},
		{"an invalid value keeps the default", "bogus", PaintKindColor, "black"},
		{"a url that is not a fragment", "url(x.svg#g)", PaintKindColor, "black"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<linearGradient id="g"><stop offset="0"/></linearGradient><rect id="t" fill="`+tt.fill+`" width="1" height="1"/></svg>`)
			p := mustFind(t, d, "t").Style.Fill
			if p.Kind != tt.wantKind {
				t.Fatalf("kind = %d, want %d", p.Kind, tt.wantKind)
			}
			if tt.want != "" && p.Color != col(tt.want) {
				t.Errorf("color = %v, want %s", p.Color, tt.want)
			}
			if tt.wantKind == PaintKindServer {
				if _, ok := p.Server.(*Gradient); !ok {
					t.Errorf("server = %T", p.Server)
				}
			}
		})
	}
}

func TestStrokeProperties(t *testing.T) {
	tests := []struct {
		name  string
		attrs string
		check func(*Style) bool
	}{
		{"dash list", `stroke-dasharray="4 2"`, func(s *Style) bool { return eqf(s.DashArray, 4, 2) }},
		{"dash list with commas and units", `stroke-dasharray="1in,2px"`, func(s *Style) bool { return eqf(s.DashArray, 96, 2) }},
		{"odd dash list repeats", `stroke-dasharray="5 3 1"`, func(s *Style) bool { return eqf(s.DashArray, 5, 3, 1, 5, 3, 1) }},
		{"dash none", `stroke-dasharray="none"`, func(s *Style) bool { return s.DashArray == nil }},
		{"zero dashes are none", `stroke-dasharray="0 0"`, func(s *Style) bool { return s.DashArray == nil }},
		{"negative dash is invalid", `stroke-dasharray="4 -2"`, func(s *Style) bool { return s.DashArray == nil }},
		{"dash percent", `stroke-dasharray="10%"`, func(s *Style) bool { return eqf(s.DashArray, 10, 10) }},
		{"dash offset", `stroke-dashoffset="3"`, func(s *Style) bool { return s.DashOffset == 3 }},
		{"line cap square", `stroke-linecap="square"`, func(s *Style) bool { return s.LineCap == LineCapSquare }},
		{"line join bevel", `stroke-linejoin="bevel"`, func(s *Style) bool { return s.LineJoin == LineJoinBevel }},
		{"line join round", `stroke-linejoin="round"`, func(s *Style) bool { return s.LineJoin == LineJoinRound }},
		{"line join miter-clip is miter", `stroke-linejoin="miter-clip"`, func(s *Style) bool { return s.LineJoin == LineJoinMiter }},
		{"invalid line cap", `stroke-linecap="pointy"`, func(s *Style) bool { return s.LineCap == LineCapButt }},
		{"miter limit", `stroke-miterlimit="2.5"`, func(s *Style) bool { return s.MiterLimit == 2.5 }},
		{"miter limit below 1 is invalid", `stroke-miterlimit="0.5"`, func(s *Style) bool { return s.MiterLimit == 4 }},
		{"stroke width", `stroke-width="2.5"`, func(s *Style) bool { return s.StrokeWidth == 2.5 }},
		{"stroke width percent", `stroke-width="10%"`, func(s *Style) bool { return s.StrokeWidth == 10 }},
		{"stroke width negative is invalid", `stroke-width="-2"`, func(s *Style) bool { return s.StrokeWidth == 1 }},
		{"fill rule", `fill-rule="evenodd"`, func(s *Style) bool { return s.FillRule == FillRuleEvenOdd }},
		{"clip rule", `clip-rule="evenodd"`, func(s *Style) bool { return s.ClipRule == FillRuleEvenOdd }},
		{"opacity percent", `fill-opacity="50%"`, func(s *Style) bool { return s.FillOpacity == 0.5 }},
		{"opacity is clamped", `stroke-opacity="3"`, func(s *Style) bool { return s.StrokeOpacity == 1 }},
		{"paint order", `paint-order="stroke"`, func(s *Style) bool {
			return s.PaintOrder == [3]PaintLayer{PaintLayerStroke, PaintLayerFill, PaintLayerMarkers}
		}},
		{"paint order normal", `paint-order="normal"`, func(s *Style) bool { return s.PaintOrder == DefaultPaintOrder }},
		{"shape rendering", `shape-rendering="crispEdges"`, func(s *Style) bool { return s.ShapeRendering == ShapeRenderingCrispEdges }},
		{"font family", `font-family="'Open Sans', Arial, sans-serif"`, func(s *Style) bool {
			return len(s.FontFamily) == 3 && s.FontFamily[0] == "Open Sans" && s.FontFamily[2] == "sans-serif"
		}},
		{"font weight number", `font-weight="300"`, func(s *Style) bool { return s.FontWeight == 300 }},
		{"font style", `font-style="italic"`, func(s *Style) bool { return s.FontStyle == FontStyleItalic }},
		{"text anchor", `text-anchor="middle"`, func(s *Style) bool { return s.TextAnchor == TextAnchorMiddle }},
		{"text decoration", `text-decoration="underline line-through"`, func(s *Style) bool {
			return s.TextDecoration == TextDecorationUnderline|TextDecorationLineThrough
		}},
		{"word spacing", `word-spacing="3px"`, func(s *Style) bool { return s.WordSpacing == 3 }},
		{"color interpolation filters", `color-interpolation-filters="sRGB"`, func(s *Style) bool { return s.ColorInterpolationFilters == ColorInterpolationSRGB }},
		{"color interpolation filters default", ``, func(s *Style) bool { return s.ColorInterpolationFilters == ColorInterpolationLinearRGB }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<rect id="t" `+tt.attrs+` width="1" height="1"/></svg>`)
			if s := mustFind(t, d, "t").Style; !tt.check(s) {
				t.Errorf("style = %+v", *s)
			}
		})
	}
}

func eqf(got []float64, want ...float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestNodeProperties(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<rect id="a" opacity="0.5" width="1" height="1"/>`+
		`<rect id="b" style="opacity:25%" opacity="0.5" width="1" height="1"/>`+
		`<rect id="c" display="none" width="1" height="1"/>`+
		`<g id="d" display="none"><rect id="e" width="1" height="1"/></g>`+
		`<rect id="f" vector-effect="non-scaling-stroke" width="1" height="1"/>`+
		`<rect id="g" transform="translate(5 6)" width="1" height="1"/>`+
		`<rect id="h" style="transform:translate(7px,8px)" transform="translate(5 6)" width="1" height="1"/>`+
		`<rect id="i" transform="bogus(1)" width="1" height="1"/>`+
		`</svg>`)
	if got := mustFind(t, d, "a").Opacity; got != 0.5 {
		t.Errorf("opacity attribute = %v", got)
	}
	if got := mustFind(t, d, "b").Opacity; got != 0.25 {
		t.Errorf("opacity style = %v", got)
	}
	for _, id := range []string{"c", "d", "e"} {
		if findNode(d.Root, id) != nil {
			t.Errorf("display:none node %q is in the tree", id)
		}
	}
	if mustFind(t, d, "f").VectorEffect != VectorEffectNonScalingStroke {
		t.Error("vector-effect")
	}
	if got := mustFind(t, d, "g").Transform; got != Translate(5, 6) {
		t.Errorf("transform = %v", got)
	}
	if got := mustFind(t, d, "h").Transform; got != Translate(7, 8) {
		t.Errorf("transform property over attribute = %v", got)
	}
	if got := mustFind(t, d, "i").Transform; got != Identity {
		t.Errorf("invalid transform = %v", got)
	}
}

var _ = ui.Offset{}
