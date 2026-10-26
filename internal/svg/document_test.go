package svg

import (
	"bytes"
	"compress/gzip"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/ui"
)

func TestDocumentSize(t *testing.T) {
	tests := []struct {
		name string
		svg  string
		want ui.Size
	}{
		{"viewBox only", `<svg viewBox="0 0 24 48"/>`, ui.Size{Width: 24, Height: 48}},
		{"viewBox with an offset", `<svg viewBox="10 10 24 48"/>`, ui.Size{Width: 24, Height: 48}},
		{"width and height only", `<svg width="30" height="40"/>`, ui.Size{Width: 30, Height: 40}},
		{"width and height win over the viewBox", `<svg width="30" height="40" viewBox="0 0 24 24"/>`, ui.Size{Width: 30, Height: 40}},
		{"width and height with units", `<svg width="1in" height="2cm"/>`, ui.Size{Width: 96, Height: 2 * 96 / 2.54}},
		{"width and height in em use the theme", `<svg width="2em" height="1ex"/>`, ui.Size{Width: 28, Height: 7}},
		{"only width: the height keeps the viewBox's ratio", `<svg width="48" viewBox="0 0 24 12"/>`, ui.Size{Width: 48, Height: 24}},
		{"only height: the width keeps the viewBox's ratio", `<svg height="48" viewBox="0 0 24 12"/>`, ui.Size{Width: 96, Height: 48}},
		{"percentages count as absent", `<svg width="100%" height="50%" viewBox="0 0 24 12"/>`, ui.Size{Width: 24, Height: 12}},
		{"no viewBox and no size: the bounds of the shapes", `<svg><rect x="10" y="10" width="20" height="30"/></svg>`, ui.Size{Width: 30, Height: 40}},
		{"no viewBox, only width", `<svg width="100"><rect x="10" y="10" width="20" height="30"/></svg>`, ui.Size{Width: 100, Height: 40}},
		{"bounds follow transforms", `<svg><g transform="translate(10 10)"><rect width="20" height="30"/></g></svg>`, ui.Size{Width: 30, Height: 40}},
		{"nothing", `<svg/>`, ui.Size{}},
		{"zero-size viewBox", `<svg viewBox="0 0 0 10" width="5" height="6"/>`, ui.Size{Width: 5, Height: 6}},
		{"negative width is ignored", `<svg width="-5" viewBox="0 0 24 12"/>`, ui.Size{Width: 24, Height: 12}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, tt.svg)
			got := d.Size()
			if math.Abs(got.Width-tt.want.Width) > 1e-9 || math.Abs(got.Height-tt.want.Height) > 1e-9 {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
			if d.Root.Viewport.Width != got.Width || d.Root.Viewport.Height != got.Height {
				t.Errorf("the root viewport is %vx%v, not the size", d.Root.Viewport.Width, d.Root.Viewport.Height)
			}
		})
	}
}

func TestDocumentViewBox(t *testing.T) {
	d := mustParse(t, `<svg viewBox="1 2 3 4" preserveAspectRatio="xMinYMax slice"/>`)
	vb, ok := d.ViewBox()
	if !ok || vb != (ViewBox{1, 2, 3, 4}) {
		t.Errorf("ViewBox() = %v, %v", vb, ok)
	}
	if par := d.Root.Viewport.PreserveAspectRatio; par != (PreserveAspectRatio{AlignXMinYMax, true}) {
		t.Errorf("preserveAspectRatio = %v", par)
	}
	if _, ok := mustParse(t, `<svg/>`).ViewBox(); ok {
		t.Error("a document with no viewBox has one")
	}
}

func TestPercentagesUseTheViewBox(t *testing.T) {
	d := mustParse(t, `<svg viewBox="0 0 200 100"><rect id="a" x="10%" y="10%" width="50%" height="50%"/><circle id="b" r="10%"/></svg>`)
	b := mustFind(t, d, "a").Path.GetBounds()
	if b != ui.NewRectFromLTRB(20, 10, 120, 60) {
		t.Errorf("rect bounds = %v", b)
	}
	r := mustFind(t, d, "b").Path.GetBounds()
	want := 0.1 * math.Sqrt((200*200+100*100)/2.0)
	if math.Abs(r.Width()/2-want) > 1e-9 {
		t.Errorf("circle radius = %v, want %v", r.Width()/2, want)
	}
}

func TestPercentagesWithoutAViewport(t *testing.T) {
	// There is no viewport, so a percentage is invalid and the shape has no size.
	d := mustParse(t, `<svg><rect id="a" width="50%" height="10"/><rect id="b" width="5" height="10"/></svg>`)
	if findNode(d.Root, "a") != nil {
		t.Error("a rect with an unresolvable width is in the tree")
	}
	if findNode(d.Root, "b") == nil {
		t.Error("a rect with absolute sizes is not in the tree")
	}
}

func TestShapes(t *testing.T) {
	tests := []struct {
		name   string
		elem   string
		want   string // pathString, "" for not rendered
		bounds ui.Rect
	}{
		{"rect", `<rect x="1" y="2" width="10" height="20"/>`, "M1,2 L11,2 L11,22 L1,22 Z", ui.NewRectFromLTRB(1, 2, 11, 22)},
		{"rect zero width", `<rect width="0" height="10"/>`, "", ui.Rect{}},
		{"rect negative height", `<rect width="10" height="-1"/>`, "", ui.Rect{}},
		{"rect without size", `<rect/>`, "", ui.Rect{}},
		{"rect rx only", `<rect width="20" height="10" rx="3"/>`, "rounded", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"rect ry only", `<rect width="20" height="10" ry="3"/>`, "rounded", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"rect rx and ry", `<rect width="20" height="10" rx="3" ry="2"/>`, "rounded", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"rect rx larger than half is clamped", `<rect width="20" height="10" rx="50"/>`, "rounded", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"rect negative rx is auto", `<rect width="20" height="10" rx="-3"/>`, "M0,0 L20,0 L20,10 L0,10 Z", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"rect rx zero", `<rect width="20" height="10" rx="0" ry="4"/>`, "M0,0 L20,0 L20,10 L0,10 Z", ui.NewRectFromLTRB(0, 0, 20, 10)},
		{"circle", `<circle cx="10" cy="10" r="5"/>`, "oval", ui.NewRectFromLTRB(5, 5, 15, 15)},
		{"circle without radius", `<circle cx="10" cy="10"/>`, "", ui.Rect{}},
		{"circle negative radius", `<circle r="-5"/>`, "", ui.Rect{}},
		{"ellipse", `<ellipse cx="10" cy="10" rx="5" ry="2"/>`, "oval", ui.NewRectFromLTRB(5, 8, 15, 12)},
		{"ellipse rx auto", `<ellipse cx="10" cy="10" ry="4"/>`, "oval", ui.NewRectFromLTRB(6, 6, 14, 14)},
		{"ellipse without radii", `<ellipse cx="10" cy="10"/>`, "", ui.Rect{}},
		{"line", `<line x1="1" y1="2" x2="3" y2="4"/>`, "M1,2 L3,4", ui.NewRectFromLTRB(1, 2, 3, 4)},
		{"polyline", `<polyline points="0,0 10,0 10,10"/>`, "M0,0 L10,0 L10,10", ui.NewRectFromLTRB(0, 0, 10, 10)},
		{"polygon closes", `<polygon points="0 0 10 0 10 10"/>`, "M0,0 L10,0 L10,10 Z", ui.NewRectFromLTRB(0, 0, 10, 10)},
		{"polyline with an odd number of values", `<polyline points="0 0 10 0 10"/>`, "M0,0 L10,0", ui.NewRectFromLTRB(0, 0, 10, 0)},
		{"polyline stops at an error", `<polyline points="0 0 10 0 x 5 20 20"/>`, "M0,0 L10,0", ui.NewRectFromLTRB(0, 0, 10, 0)},
		{"polyline with one point", `<polyline points="0 0"/>`, "", ui.Rect{}},
		{"path", `<path d="M0 0 L5 5"/>`, "M0,0 L5,5", ui.NewRectFromLTRB(0, 0, 5, 5)},
		{"path without d", `<path/>`, "", ui.Rect{}},
		{"path with a bad start", `<path d="L5 5"/>`, "", ui.Rect{}},
		{"lengths with units", `<rect x="1in" width="10mm" height="1em"/>`, "M96,0 L133.8,0 L133.8,14 L96,14 Z", ui.NewRectFromLTRB(96, 0, 96+10*96/25.4, 14)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := strings.Replace(tt.elem, "/>", ` id="t"/>`, 1)
			d := mustParse(t, svgOpen+src+`</svg>`)
			n := findNode(d.Root, "t")
			if tt.want == "" {
				if n != nil {
					t.Fatalf("the element is in the tree: %s", pathString(n.Path))
				}
				return
			}
			if n == nil {
				t.Fatal("the element is not in the tree")
			}
			if n.Kind != NodeShape {
				t.Errorf("kind = %d", n.Kind)
			}
			if tt.want != "rounded" && tt.want != "oval" {
				if got := pathString(n.Path); got != tt.want {
					t.Errorf("path = %q, want %q", got, tt.want)
				}
			}
			b := n.Path.GetBounds()
			if math.Abs(b.Left-tt.bounds.Left) > 1e-3 || math.Abs(b.Top-tt.bounds.Top) > 1e-3 ||
				math.Abs(b.Right-tt.bounds.Right) > 1e-3 || math.Abs(b.Bottom-tt.bounds.Bottom) > 1e-3 {
				t.Errorf("bounds = %v, want %v", b, tt.bounds)
			}
		})
	}
}

func TestRoundedRectStartsAtTheTopLeftAndGoesClockwise(t *testing.T) {
	d := mustParse(t, svgOpen+`<rect id="t" x="0" y="0" width="20" height="10" rx="3" ry="3"/></svg>`)
	segs := mustFind(t, d, "t").Path.Segments()
	if segs[0].Verb != ui.PathVerbMove || segs[0].P0 != (ui.Offset{Dx: 3, Dy: 0}) {
		t.Errorf("starts at %v %v", segs[0].Verb, segs[0].P0)
	}
	if segs[1].Verb != ui.PathVerbLine || segs[1].P1 != (ui.Offset{Dx: 17, Dy: 0}) {
		t.Errorf("then goes along the top edge: %v %v", segs[1].Verb, segs[1].P1)
	}
}

func TestFillRuleSetsThePathFillType(t *testing.T) {
	d := mustParse(t, svgOpen+`<path id="a" d="M0 0h1v1z"/><path id="b" fill-rule="evenodd" d="M0 0h1v1z"/></svg>`)
	if got := mustFind(t, d, "a").Path.FillType(); got != ui.PathFillTypeNonZero {
		t.Errorf("a: %v", got)
	}
	if got := mustFind(t, d, "b").Path.FillType(); got != ui.PathFillTypeEvenOdd {
		t.Errorf("b: %v", got)
	}
}

func TestNestedSVGAndGroups(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<g id="g" transform="translate(5 5)"><a id="a" href="x"><rect id="r" width="1" height="1"/></a></g>`+
		`<svg id="n" x="10" y="20" width="50" height="40" viewBox="0 0 10 8" preserveAspectRatio="none" overflow="visible"><rect id="in" width="10%" height="50%"/></svg>`+
		`<svg id="m" width="25%" height="25%"/>`+
		`</svg>`)
	if a := mustFind(t, d, "a"); a.Kind != NodeGroup || len(a.Children) != 1 {
		t.Errorf("a is not a group with a child: %+v", a)
	}
	n := mustFind(t, d, "n")
	if n.Kind != NodeSvg {
		t.Fatalf("n kind = %d", n.Kind)
	}
	want := Viewport{X: 10, Y: 20, Width: 50, Height: 40, ViewBox: &ViewBox{0, 0, 10, 8}, PreserveAspectRatio: PreserveAspectRatio{Align: AlignNone}, Overflow: true}
	got := *n.Viewport
	if got.X != want.X || got.Y != want.Y || got.Width != want.Width || got.Height != want.Height ||
		*got.ViewBox != *want.ViewBox || got.PreserveAspectRatio != want.PreserveAspectRatio || got.Overflow != want.Overflow || got.Root {
		t.Errorf("viewport = %+v, want %+v", got, want)
	}
	// Percentages inside the nested viewport refer to its viewBox.
	if b := mustFind(t, d, "in").Path.GetBounds(); b != ui.NewRectFromLTRB(0, 0, 1, 4) {
		t.Errorf("inner rect = %v", b)
	}
	m := mustFind(t, d, "m")
	if m.Viewport.Width != 25 || m.Viewport.Height != 25 || m.Viewport.Overflow {
		t.Errorf("m viewport = %+v", *m.Viewport)
	}
	if !d.Root.Viewport.Root || d.Root.Kind != NodeSvg {
		t.Error("the root is not marked")
	}
}

func TestUse(t *testing.T) {
	tests := []struct {
		name  string
		svg   string
		check func(t *testing.T, d *Document)
	}{
		{
			name: "use makes a group at the translation with the use's transform before it",
			svg:  `<defs><rect id="r" width="10" height="10"/></defs><use id="u" href="#r" x="5" y="6" transform="scale(2)"/>`,
			check: func(t *testing.T, d *Document) {
				u := mustFind(t, d, "u")
				if u.Kind != NodeGroup || len(u.Children) != 1 {
					t.Fatalf("u = %+v", u)
				}
				if want := Scale(2, 2).Mul(Translate(5, 6)); u.Transform != want {
					t.Errorf("transform = %v, want %v", u.Transform, want)
				}
				c := u.Children[0]
				if c.Kind != NodeShape || c.ID != "r" {
					t.Errorf("child = %+v", c)
				}
			},
		},
		{
			name: "xlink:href",
			svg:  `<defs><rect id="r" width="10" height="10"/></defs><use id="u" xlink:href="#r"/>`,
			check: func(t *testing.T, d *Document) {
				if u := mustFind(t, d, "u"); len(u.Children) != 1 {
					t.Errorf("no child")
				}
			},
		},
		{
			name: "href wins over xlink:href",
			svg:  `<defs><rect id="r" width="10" height="10"/><circle id="c" r="3"/></defs><use id="u" href="#c" xlink:href="#r"/>`,
			check: func(t *testing.T, d *Document) {
				if u := mustFind(t, d, "u"); u.Children[0].Shape != ShapeCircle {
					t.Errorf("child shape = %d", u.Children[0].Shape)
				}
			},
		},
		{
			name: "the copy inherits from the use, not from the original's parent",
			svg:  `<g fill="blue"><rect id="r" width="10" height="10"/></g><g fill="red"><use id="u" href="#r"/></g>`,
			check: func(t *testing.T, d *Document) {
				u := mustFind(t, d, "u")
				if got := u.Children[0].Style.Fill.Color; got != col("red") {
					t.Errorf("fill = %v, want red", got)
				}
			},
		},
		{
			name: "the original's own fill wins over the use's",
			svg:  `<rect id="r" fill="blue" width="10" height="10"/><use id="u" href="#r" fill="red"/>`,
			check: func(t *testing.T, d *Document) {
				if got := mustFind(t, d, "u").Children[0].Style.Fill.Color; got != col("blue") {
					t.Errorf("fill = %v, want blue", got)
				}
			},
		},
		{
			name: "use of a group",
			svg:  `<g id="g"><rect width="1" height="1"/><rect width="2" height="2"/></g><use id="u" href="#g"/>`,
			check: func(t *testing.T, d *Document) {
				if g := mustFind(t, d, "u").Children[0]; len(g.Children) != 2 {
					t.Errorf("group has %d children", len(g.Children))
				}
			},
		},
		{
			name: "use of a missing id draws nothing",
			svg:  `<use id="u" href="#nope"/><use id="v"/><use id="w" href="elsewhere.svg#x"/>`,
			check: func(t *testing.T, d *Document) {
				for _, id := range []string{"u", "v", "w"} {
					if findNode(d.Root, id) != nil {
						t.Errorf("%s is in the tree", id)
					}
				}
			},
		},
		{
			name: "use of a symbol is a viewport as large as the use",
			svg:  `<symbol id="s" viewBox="0 0 10 10" preserveAspectRatio="xMinYMin"><rect width="10" height="10"/></symbol><use id="u" href="#s" x="5" y="5" width="30" height="40"/>`,
			check: func(t *testing.T, d *Document) {
				u := mustFind(t, d, "u")
				s := u.Children[0]
				if s.Kind != NodeSvg || s.Viewport.Width != 30 || s.Viewport.Height != 40 || s.Viewport.ViewBox == nil {
					t.Fatalf("symbol node = %+v %+v", s, s.Viewport)
				}
				if s.Viewport.PreserveAspectRatio.Align != AlignXMinYMin {
					t.Errorf("par = %v", s.Viewport.PreserveAspectRatio)
				}
				if s.Viewport.X != 0 || u.Transform != Translate(5, 5) {
					t.Errorf("position: viewport x %v, use transform %v", s.Viewport.X, u.Transform)
				}
				if len(s.Children) != 1 {
					t.Errorf("symbol has %d children", len(s.Children))
				}
			},
		},
		{
			name: "use of a symbol without a size is 100% of the viewport",
			svg:  `<symbol id="s" viewBox="0 0 10 10"><rect width="10" height="10"/></symbol><use id="u" href="#s"/>`,
			check: func(t *testing.T, d *Document) {
				s := mustFind(t, d, "u").Children[0]
				if s.Viewport.Width != 100 || s.Viewport.Height != 100 {
					t.Errorf("symbol viewport = %vx%v", s.Viewport.Width, s.Viewport.Height)
				}
			},
		},
		{
			name: "a symbol that is not used is not drawn",
			svg:  `<symbol id="s"><rect id="r" width="10" height="10"/></symbol>`,
			check: func(t *testing.T, d *Document) {
				if findNode(d.Root, "r") != nil || findNode(d.Root, "s") != nil {
					t.Error("the symbol is drawn")
				}
			},
		},
		{
			name: "use of an svg takes the use's size",
			svg:  `<svg id="s" width="10" height="10" viewBox="0 0 5 5"><rect width="5" height="5"/></svg><use id="u" href="#s" width="50"/>`,
			check: func(t *testing.T, d *Document) {
				// Two copies are drawn: the svg itself, and the one of the use.
				c := mustFind(t, d, "u").Children[0]
				if c.Kind != NodeSvg || c.Viewport.Width != 50 || c.Viewport.Height != 10 {
					t.Errorf("viewport = %+v", c.Viewport)
				}
			},
		},
		{
			name: "use of a use",
			svg:  `<rect id="r" width="1" height="1"/><use id="a" href="#r" x="1"/><use id="b" href="#a" x="2"/>`,
			check: func(t *testing.T, d *Document) {
				b := mustFind(t, d, "b")
				a := b.Children[0]
				if a.Kind != NodeGroup || a.Children[0].Kind != NodeShape {
					t.Errorf("chain = %+v", a)
				}
			},
		},
		{
			name: "use of an element with display none draws nothing",
			svg:  `<rect id="r" display="none" width="1" height="1"/><use id="u" href="#r"/>`,
			check: func(t *testing.T, d *Document) {
				if findNode(d.Root, "u") != nil {
					t.Error("u is in the tree")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.svg+`</svg>`)
			tt.check(t, d)
		})
	}
}

func TestUseCycles(t *testing.T) {
	tests := []struct {
		name string
		svg  string
	}{
		{"use of itself", `<use id="u" href="#u"/>`},
		{"two uses of each other", `<g id="a"><use href="#b"/></g><g id="b"><use href="#a"/></g><use href="#a"/>`},
		{"use of its own parent", `<g id="g"><rect width="1" height="1"/><use href="#g"/></g>`},
		{"use of its own grandparent", `<g id="g"><g><use href="#g"/></g></g><use href="#g"/>`},
		{"a symbol that uses itself", `<symbol id="s"><use href="#s"/></symbol><use href="#s"/>`},
		{"a ring of three", `<use id="a" href="#b"/><use id="b" href="#c"/><use id="c" href="#a"/>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.svg+`</svg>`)
			if n := countNodes(d.Root); n > 50 {
				t.Errorf("the cycle expanded to %d nodes", n)
			}
		})
	}
}

func TestUseExpansionIsBounded(t *testing.T) {
	// Each level uses the previous one twice: 2^40 copies if nothing stopped it.
	var sb strings.Builder
	sb.WriteString(svgOpen + `<rect id="l0" width="1" height="1"/>`)
	for i := 1; i <= 40; i++ {
		sb.WriteString(`<g id="l` + itoa(i) + `"><use href="#l` + itoa(i-1) + `"/><use href="#l` + itoa(i-1) + `"/></g>`)
	}
	sb.WriteString(`<use href="#l40"/></svg>`)
	d := mustParse(t, sb.String())
	if n := countNodes(d.Root); n > 2*maxUseNodes+1000 {
		t.Errorf("%d nodes", n)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{byte('0' + i%10)}, b...)
	}
	return string(b)
}

func TestSwitch(t *testing.T) {
	tests := []struct {
		name   string
		locale ui.Locale
		body   string
		want   string // id of the child that is drawn, "" for none
	}{
		{"first child without conditions", ui.Locale{}, `<rect id="a" width="1" height="1"/><rect id="b" width="1" height="1"/>`, "a"},
		{"system language en", ui.Locale{}, `<rect id="a" systemLanguage="fr" width="1" height="1"/><rect id="b" systemLanguage="en" width="1" height="1"/>`, "b"},
		{"a list of languages", ui.NewLocale("de"), `<rect id="a" systemLanguage="fr" width="1" height="1"/><rect id="b" systemLanguage="en, de" width="1" height="1"/>`, "b"},
		{"a region of the user matches a plain tag", ui.NewLocale("en", "US"), `<rect id="a" systemLanguage="fr" width="1" height="1"/><rect id="b" systemLanguage="en" width="1" height="1"/>`, "b"},
		{"a plain user language matches a region tag", ui.NewLocale("en"), `<rect id="a" systemLanguage="en-GB" width="1" height="1"/>`, "a"},
		{"a different region does not match", ui.NewLocale("en", "US"), `<rect id="a" systemLanguage="en-GB" width="1" height="1"/>`, ""},
		{"fallback child", ui.NewLocale("ja"), `<rect id="a" systemLanguage="fr" width="1" height="1"/><rect id="b" width="1" height="1"/>`, "b"},
		{"required extensions are never supported", ui.Locale{}, `<rect id="a" requiredExtensions="http://x" width="1" height="1"/><rect id="b" width="1" height="1"/>`, "b"},
		{"required features: an empty list is false", ui.Locale{}, `<rect id="a" requiredFeatures="" width="1" height="1"/><rect id="b" width="1" height="1"/>`, "b"},
		{"required features: any feature is true", ui.Locale{}, `<rect id="a" requiredFeatures="http://www.w3.org/TR/SVG11/feature#Shape" width="1" height="1"/><rect id="b" width="1" height="1"/>`, "a"},
		{"a non-graphics child is skipped", ui.Locale{}, `<desc>x</desc><rect id="a" width="1" height="1"/>`, "a"},
		{"no child matches", ui.NewLocale("ja"), `<rect id="a" systemLanguage="fr" width="1" height="1"/>`, ""},
		{"only the first match is drawn even if it draws nothing", ui.Locale{}, `<rect id="a" display="none" width="1" height="1"/><rect id="b" width="1" height="1"/>`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<switch id="sw">`+tt.body+`</switch></svg>`, ParseOpts{Locale: tt.locale})
			sw := mustFind(t, d, "sw")
			if tt.want == "" {
				if len(sw.Children) != 0 {
					t.Errorf("switch drew %d children", len(sw.Children))
				}
				return
			}
			if len(sw.Children) != 1 || sw.Children[0].ID != tt.want {
				t.Errorf("switch children = %v, want %q", ids(sw.Children), tt.want)
			}
		})
	}
}

func ids(ns []*Node) []string {
	var out []string
	for _, n := range ns {
		out = append(out, n.ID)
	}
	return out
}

func TestConditionalProcessingOutsideSwitch(t *testing.T) {
	d := mustParse(t, svgOpen+`<rect id="a" systemLanguage="fr" width="1" height="1"/><rect id="b" width="1" height="1"/></svg>`)
	if findNode(d.Root, "a") != nil || findNode(d.Root, "b") == nil {
		t.Error("systemLanguage did not hide the element")
	}
}

func TestNamespacesAndUnknownElements(t *testing.T) {
	src := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"
     xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" xmlns:other="http://example.com/other"
     xmlns:xlink="http://www.w3.org/1999/xlink" inkscape:version="1.0" viewBox="0 0 10 10">
  <sodipodi:namedview id="nv"><rect id="inside-nv" width="1" height="1"/></sodipodi:namedview>
  <other:rect id="foreign" width="5" height="5"/>
  <foo id="unknown"><rect id="inside-unknown" width="1" height="1"/></foo>
  <metadata><rect id="inside-metadata" width="1" height="1"/></metadata>
  <foreignObject id="fo"><div xmlns="http://www.w3.org/1999/xhtml"><rect id="inside-fo" width="1" height="1"/></div></foreignObject>
  <script>alert(1)</script>
  <rect id="ok" inkscape:label="x" other:fill="red" width="1" height="1"/>
  <a xlink:href="#ok"><rect id="in-a" width="1" height="1"/></a>
</svg>`
	d := mustParse(t, src)
	for _, id := range []string{"nv", "inside-nv", "foreign", "unknown", "inside-unknown", "inside-metadata", "fo", "inside-fo"} {
		if findNode(d.Root, id) != nil {
			t.Errorf("%q is in the tree", id)
		}
	}
	ok := mustFind(t, d, "ok")
	if ok.Style.Fill.Color != col("black") {
		t.Errorf("an attribute of another namespace set the fill: %v", ok.Style.Fill)
	}
	if mustFind(t, d, "in-a") == nil {
		t.Error("a is missing")
	}
	if got := len(d.Root.Children); got != 2 {
		t.Errorf("root has %d children, want 2", got)
	}
}

func TestDocumentWithoutNamespace(t *testing.T) {
	d := mustParse(t, `<svg width="5" height="5"><rect id="r" width="5" height="5" fill="red"/></svg>`)
	if mustFind(t, d, "r").Style.Fill.Color != col("red") {
		t.Error("a document with no xmlns is not read")
	}
}

func TestTitleAndDesc(t *testing.T) {
	d := mustParse(t, svgOpen+`<title> My  icon </title><desc>An icon.</desc><g><title>not the root's</title></g><title>second</title></svg>`)
	if d.Title != "My  icon" || d.Desc != "An icon." {
		t.Errorf("title %q desc %q", d.Title, d.Desc)
	}
}

func TestXMLFeatures(t *testing.T) {
	t.Run("entities", func(t *testing.T) {
		src := `<?xml version="1.0"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
	<!ENTITY ns_svg "http://www.w3.org/2000/svg">
	<!ENTITY red "#ff0000">
]>
<svg xmlns="&ns_svg;" viewBox="0 0 1 1"><rect id="r" fill="&red;" width="1" height="1"/><title>a &amp; b &lt; c</title></svg>`
		d := mustParse(t, src)
		if mustFind(t, d, "r").Style.Fill.Color != col("red") {
			t.Error("entity not expanded")
		}
		if d.Title != "a & b < c" {
			t.Errorf("title = %q", d.Title)
		}
	})
	t.Run("CDATA in text", func(t *testing.T) {
		d := mustParse(t, svgOpen+`<text id="t"><![CDATA[a < b]]></text></svg>`)
		if got := mustFind(t, d, "t").Text.Items[0].Text; got != "a < b" {
			t.Errorf("text = %q", got)
		}
	})
	t.Run("byte order mark", func(t *testing.T) {
		mustParse(t, "\xef\xbb\xbf"+svgOpen+`</svg>`)
	})
	t.Run("latin-1", func(t *testing.T) {
		d := mustParse(t, "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>"+svgOpen+"<title>caf\xe9</title></svg>")
		if d.Title != "café" {
			t.Errorf("title = %q", d.Title)
		}
	})
	t.Run("gzip", func(t *testing.T) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(svgOpen + `<rect id="r" width="1" height="1"/></svg>`))
		_ = zw.Close()
		d := mustParse(t, buf.String())
		if findNode(d.Root, "r") == nil {
			t.Error("compressed document not read")
		}
	})
	t.Run("nesting up to the limit", func(t *testing.T) {
		mustParse(t, svgOpen+strings.Repeat("<g>", maxDepth-2)+`<rect width="1" height="1"/>`+strings.Repeat("</g>", maxDepth-2)+`</svg>`)
	})
	t.Run("the first id wins", func(t *testing.T) {
		d := mustParse(t, svgOpen+`<defs><rect id="r" fill="red" width="1" height="1"/><rect id="r" fill="blue" width="1" height="1"/></defs><use id="u" href="#r"/></svg>`)
		if got := mustFind(t, d, "u").Children[0].Style.Fill.Color; got != col("red") {
			t.Errorf("fill = %v", got)
		}
	})
}

func TestImages(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		wantMIME string
		wantData string
		drawn    bool
	}{
		{"base64 png", "data:image/png;base64,aGVsbG8=", "image/png", "hello", true},
		{"base64 with line breaks", "data:image/png;base64,aGVs\nbG8=", "image/png", "hello", true},
		{"base64 without padding", "data:image/png;base64,aGVsbG8", "image/png", "hello", true},
		{"charset parameter", "data:image/svg+xml;charset=utf-8;base64,aGVsbG8=", "image/svg+xml", "hello", true},
		{"percent encoded", "data:image/svg+xml,%3Csvg%20a%3E", "image/svg+xml", "<svg a>", true},
		{"upper case scheme and type", "DATA:IMAGE/JPEG;BASE64,aGVsbG8=", "image/jpeg", "hello", true},
		{"external file", "photo.png", "", "", false},
		{"http", "http://example.com/a.png", "", "", false},
		{"broken base64", "data:image/png;base64,!!!", "", "", false},
		{"no comma", "data:image/png", "", "", false},
		{"empty", "data:image/png;base64,", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+`<image id="i" x="1" y="2" width="30" height="40" preserveAspectRatio="xMinYMin slice" href="`+tt.href+`"/></svg>`)
			n := findNode(d.Root, "i")
			if !tt.drawn {
				if n != nil {
					t.Fatal("the image is in the tree")
				}
				return
			}
			if n == nil {
				t.Fatal("the image is not in the tree")
			}
			im := n.Image
			if n.Kind != NodeImage || im.MIME != tt.wantMIME || string(im.Data) != tt.wantData {
				t.Errorf("image = %q %q", im.MIME, im.Data)
			}
			if im.X != 1 || im.Y != 2 || im.Width != 30 || im.Height != 40 || im.PreserveAspectRatio != (PreserveAspectRatio{AlignXMinYMin, true}) {
				t.Errorf("image = %+v", *im)
			}
		})
	}
}

func TestParseOptsMoreThanOnePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	_, _ = Parse([]byte(`<svg/>`), ParseOpts{}, ParseOpts{})
}

func TestErrorsAndTheirRoutes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		route    string
		tag      string
		metadata map[string]any
	}{
		{"unexpected end", `<svg`, "Parse->readXML.SvgParseFailed", TagSvgParseFailed, map[string]any{"line": 1}},
		{"mismatched tags", "<svg>\n<g></svg>", "Parse->readXML.SvgParseFailed", TagSvgParseFailed, map[string]any{"line": 2}},
		{"empty", ``, "Parse->readXML.SvgParseFailed", TagSvgParseFailed, nil},
		{"not xml", `hello`, "Parse->readXML.SvgParseFailed", TagSvgParseFailed, nil},
		{"unknown entity", `<svg>&nbsp;</svg>`, "Parse->readXML.SvgParseFailed", TagSvgParseFailed, nil},
		{"unsupported encoding", `<?xml version="1.0" encoding="shift_jis"?><svg/>`, "Parse->readXML.SvgParseFailed", TagSvgParseFailed, nil},
		{"broken gzip", "\x1f\x8bbad", "Parse->readXML.SvgParseFailed", TagSvgParseFailed, nil},
		{"elements nested too deeply", svgOpen + strings.Repeat("<g>", 1100), "Parse->readXML.SvgParseFailed", TagSvgParseFailed, map[string]any{"limit": maxDepth}},
		{"the root is html", `<html/>`, "Parse.SvgNotSvg", TagSvgNotSvg, map[string]any{"element": "html"}},
		{"the root is in another namespace", `<svg xmlns="http://example.com/x"/>`, "Parse.SvgNotSvg", TagSvgNotSvg, nil},
		{"the root is a rect", `<rect/>`, "Parse.SvgNotSvg", TagSvgNotSvg, nil},
		{"negative viewBox on the root", `<svg viewBox="0 0 -1 5"/>`, "Parse->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, map[string]any{"attribute": "viewBox"}},
		{"negative viewBox height on the root", `<svg viewBox="0 0 5 -1"/>`, "Parse->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, nil},
		{"negative viewBox on a nested svg", `<svg><svg viewBox="0 0 5 -5"/></svg>`, "Parse->builder.buildChildren->builder.buildElement->builder.buildNestedSVG->builder.viewportOf->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, nil},
		{"negative viewBox on a used symbol", `<svg><symbol id="s" viewBox="0 0 -5 5"/><use href="#s"/></svg>`, "builder.buildUse->builder.viewportOf->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, nil},
		{"negative viewBox on a pattern", `<svg><pattern id="p" width="1" height="1" viewBox="0 0 -1 1"/><rect fill="url(#p)" width="1" height="1"/></svg>`, "builder.pattern->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, nil},
		{"negative viewBox on a marker", `<svg><marker id="m" viewBox="0 0 1 -1"/><path d="M0 0L1 1" marker-end="url(#m)"/></svg>`, "builder.marker->parseViewBox.SvgInvalidValue", TagSvgInvalidValue, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("no error: %+v", d)
			}
			if d != nil {
				t.Error("a document and an error")
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.tag {
				t.Errorf("tag = %s, want %s", appErr.Tag, tt.tag)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s\nwant %s", appErr.Route(), tt.route)
			}
			for k, want := range tt.metadata {
				if got := appErr.Metadata[k]; got != want {
					t.Errorf("metadata %s = %v, want %v", k, got, want)
				}
			}
		})
	}
}

func TestInvalidValuesOtherThanTheViewBoxAreIgnored(t *testing.T) {
	d := mustParse(t, `<svg width="abc" height="" viewBox="not numbers" preserveAspectRatio="bogus" fill="nope">`+
		`<rect id="r" x="x" width="5" height="5" rx="q" transform="rotate(" opacity="much" style="fill: ; stroke"/></svg>`)
	r := mustFind(t, d, "r")
	if r.Opacity != 1 || r.Transform != Identity || r.Style.Fill.Color != col("black") {
		t.Errorf("node = %+v", r)
	}
	if d.Root.Viewport.ViewBox != nil {
		t.Error("invalid viewBox was kept")
	}
}
