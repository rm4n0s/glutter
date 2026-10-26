package svg

import (
	"strings"
	"testing"
)

// flatten writes the items of a span as text, with {} around each child span.
func flatten(s *TextSpan) string {
	var sb strings.Builder
	for _, it := range s.Items {
		if it.Span != nil {
			sb.WriteString("{" + flatten(it.Span) + "}")
		} else {
			sb.WriteString(it.Text)
		}
	}
	return sb.String()
}

func TestTextWhiteSpace(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"plain", `<text id="t">Hello</text>`, "Hello"},
		{"leading and trailing space", `<text id="t">  Hello  </text>`, "Hello"},
		{"runs of space collapse", `<text id="t">a   b</text>`, "a b"},
		{"newlines and tabs are spaces", "<text id=\"t\">a\n\t  b\r\nc</text>", "a b c"},
		{"space between tspans stays", `<text id="t"><tspan>a</tspan> <tspan>b</tspan></text>`, "{a} {b}"},
		{"collapse across tspans", `<text id="t">a <tspan> b</tspan></text>`, "a {b}"},
		{"trailing space in the last tspan", `<text id="t"><tspan>a </tspan></text>`, "{a}"},
		{"leading space in the first tspan", `<text id="t"><tspan> a</tspan></text>`, "{a}"},
		{"xml:space preserve", `<text id="t" xml:space="preserve">  a  b </text>`, "  a  b "},
		{"preserve turns newlines into spaces", "<text id=\"t\" xml:space=\"preserve\">a\nb</text>", "a b"},
		{"preserve on a tspan", `<text id="t">a<tspan xml:space="preserve">  b  </tspan>c</text>`, "a{  b  }c"},
		{"entities", `<text id="t">a &amp; b</text>`, "a & b"},
		{"empty", `<text id="t"></text>`, ""},
		{"only space", `<text id="t">   </text>`, ""},
		{"nested tspans", `<text id="t">a<tspan>b<tspan>c</tspan>d</tspan>e</text>`, "a{b{c}d}e"},
		{"a inside text is a span", `<text id="t">a<a href="#x">b</a></text>`, "a{b}"},
		{"tref copies the text", `<text id="src">copy me</text><text id="t"><tref href="#src"/></text>`, "{copy me}"},
		{"unknown children are skipped", `<text id="t">a<title>x</title>b</text>`, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.body+`</svg>`)
			n := mustFind(t, d, "t")
			if n.Kind != NodeText || n.Text.Name != "text" {
				t.Fatalf("node = %+v", n)
			}
			if got := flatten(n.Text); got != tt.want {
				t.Errorf("text = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextPositionsAndStyle(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<text id="t" x="10 20,30" y="5%" dx="1em" dy="-2 2" rotate="45 90" font-size="10" font-family="serif" text-anchor="end" fill="red" textLength="50" lengthAdjust="spacingAndGlyphs">`+
		`a<tspan x="7" dy="1ex" font-size="20" fill="blue" font-weight="bold" style="text-decoration:underline">b</tspan></text>`+
		`<text id="u" x="a b">x</text><text id="v" display="none">x</text></svg>`)
	n := mustFind(t, d, "t")
	s := n.Text
	if !eqf(s.X, 10, 20, 30) || !eqf(s.Y, 5) || !eqf(s.Dx, 10) || !eqf(s.Dy, -2, 2) || !eqf(s.Rotate, 45, 90) {
		t.Errorf("lists: x %v y %v dx %v dy %v rotate %v", s.X, s.Y, s.Dx, s.Dy, s.Rotate)
	}
	if s.TextLength != 50 || !s.AdjustGlyphs {
		t.Errorf("textLength %v adjust %v", s.TextLength, s.AdjustGlyphs)
	}
	if s.Style.FontSize != 10 || s.Style.FontFamily[0] != "serif" || s.Style.TextAnchor != TextAnchorEnd || s.Style.Fill.Color != col("red") {
		t.Errorf("style = %+v", s.Style)
	}
	ts := s.Items[1].Span
	if ts.Name != "tspan" || !eqf(ts.X, 7) || !eqf(ts.Dy, 10) || ts.Style.FontSize != 20 || ts.Style.Fill.Color != col("blue") ||
		ts.Style.FontWeight != 700 || ts.Style.TextDecoration != TextDecorationUnderline || ts.Style.TextAnchor != TextAnchorEnd {
		t.Errorf("tspan = %+v style %+v", ts, ts.Style)
	}
	// 1ex of the tspan: half of its font size, 20.
	if !eqf(mustFind(t, d, "t").Text.Items[1].Span.Dy, 10) {
		t.Error("ex")
	}
	if mustFind(t, d, "u").Text.X != nil {
		t.Error("an invalid list is kept")
	}
	if findNode(d.Root, "v") != nil {
		t.Error("display:none text is in the tree")
	}
}

func TestTextPath(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<path id="p" d="M0 0 L100 0" transform="translate(0 50)"/>`+
		`<text id="t"><textPath id="tp" href="#p" startOffset="25%">along</textPath></text>`+
		`<text id="u"><textPath href="#p" startOffset="7">along</textPath></text>`+
		`<text id="w"><textPath href="#nope">along</textPath></text></svg>`)
	tp := mustFind(t, d, "t").Text.Items[0].Span
	if tp.Name != "textPath" || tp.PathData == nil || !tp.StartOffsetFraction || tp.StartOffset != 0.25 {
		t.Fatalf("textPath = %+v", tp)
	}
	if b := tp.PathData.GetBounds(); b.Top != 50 || b.Right != 100 {
		t.Errorf("path = %v: the transform of the path is not applied", b)
	}
	if u := mustFind(t, d, "u").Text.Items[0].Span; u.StartOffsetFraction || u.StartOffset != 7 {
		t.Errorf("startOffset = %+v", u)
	}
	if w := mustFind(t, d, "w").Text.Items[0].Span; w.PathData != nil {
		t.Error("path for a missing target")
	}
}

func TestTextInAGroupInheritsItsStyle(t *testing.T) {
	d := mustParse(t, svgOpen+`<g font-size="30" fill="green"><text id="t">x</text></g></svg>`)
	s := mustFind(t, d, "t").Text.Style
	if s.FontSize != 30 || s.Fill.Color != col("green") {
		t.Errorf("style = %+v", s)
	}
}
