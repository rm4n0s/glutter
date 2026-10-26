package svg

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func gradientOf(t *testing.T, d *Document, id string) *Gradient {
	t.Helper()
	g, ok := mustFind(t, d, id).Style.Fill.Server.(*Gradient)
	if !ok {
		t.Fatalf("%s: fill is %+v", id, mustFind(t, d, id).Style.Fill)
	}
	return g
}

func TestLinearGradient(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<linearGradient id="g" x1="10%" y1="0" x2="1" y2="0.5" gradientUnits="userSpaceOnUse" gradientTransform="rotate(90)" spreadMethod="reflect">`+
		`<stop offset="0" stop-color="red"/><stop offset="50%" style="stop-color:#00f;stop-opacity:0.5"/><stop offset="0.25" stop-color="lime" stop-opacity="50%"/><stop offset="2" stop-color="currentColor"/>`+
		`</linearGradient>`+
		`<rect id="r" fill="url(#g)" color="yellow" width="1" height="1"/></svg>`)
	g := gradientOf(t, d, "r")
	if g.Radial || g.Units != UnitsUserSpaceOnUse || g.Spread != SpreadReflect {
		t.Errorf("gradient = %+v", g)
	}
	if g.X1 != (Length{10, UnitPercent}) || g.Y1 != (Length{0, UnitPx}) || g.X2 != (Length{1, UnitPx}) || g.Y2 != (Length{0.5, UnitPx}) {
		t.Errorf("vector = %v %v %v %v", g.X1, g.Y1, g.X2, g.Y2)
	}
	if !affEq(g.Transform, Rotate(90)) {
		t.Errorf("transform = %v", g.Transform)
	}
	want := []Stop{
		{0, col("red")},
		{0.5, scaleColorAlpha(col("#00f"), 0.5)},
		{0.5, scaleColorAlpha(col("lime"), 0.5)}, // offsets never decrease
		{1, ui.NewColorFromARGB(255, 0, 0, 0)},   // currentColor of the stop: black (the gradient's color is inherited)
	}
	if len(g.Stops) != len(want) {
		t.Fatalf("stops = %v", g.Stops)
	}
	for i := range want {
		if g.Stops[i] != want[i] {
			t.Errorf("stop %d = %v, want %v", i, g.Stops[i], want[i])
		}
	}
}

func TestLinearGradientDefaults(t *testing.T) {
	d := mustParse(t, svgOpen+`<linearGradient id="g"><stop offset="0"/></linearGradient><rect id="r" fill="url(#g)" width="1" height="1"/></svg>`)
	g := gradientOf(t, d, "r")
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	if g.Units != UnitsObjectBoundingBox || g.Spread != SpreadPad || g.Transform != Identity {
		t.Errorf("gradient = %+v", g)
	}
	if g.X1 != pct(0) || g.Y1 != pct(0) || g.X2 != pct(100) || g.Y2 != pct(0) {
		t.Errorf("vector = %v %v %v %v", g.X1, g.Y1, g.X2, g.Y2)
	}
	if len(g.Stops) != 1 || g.Stops[0].Color != col("black") {
		t.Errorf("stops = %v", g.Stops)
	}
}

func TestRadialGradient(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<radialGradient id="a" cx="30%" cy="40%" r="20%"><stop offset="0"/></radialGradient>`+
		`<radialGradient id="b" cx="2em" fx="0.1" fr="5" r="1in"><stop offset="0"/></radialGradient>`+
		`<rect id="ra" fill="url(#a)" width="1" height="1"/><rect id="rb" fill="url(#b)" width="1" height="1"/></svg>`)
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	a := gradientOf(t, d, "ra")
	if !a.Radial || a.CX != pct(30) || a.CY != pct(40) || a.R != pct(20) || a.FX != pct(30) || a.FY != pct(40) || a.FR != (Length{}) {
		t.Errorf("a = %+v", a)
	}
	b := gradientOf(t, d, "rb")
	if b.CX != (Length{28, UnitPx}) || b.FX != (Length{0.1, UnitPx}) || b.FY != pct(50) || b.FR != (Length{5, UnitPx}) || b.R != (Length{96, UnitPx}) {
		t.Errorf("b = %+v", b)
	}
}

func TestGradientHrefInheritance(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<linearGradient id="base" x1="0" x2="0.5" gradientUnits="userSpaceOnUse" spreadMethod="repeat"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`+
		`<linearGradient id="child" xlink:href="#base" x2="1"/>`+
		`<linearGradient id="grandchild" href="#child" y2="1" gradientTransform="scale(2)"><stop offset="0.5" stop-color="lime"/></linearGradient>`+
		`<radialGradient id="radial" href="#base" r="0.3"/>`+
		`<rect id="c" fill="url(#child)" width="1" height="1"/>`+
		`<rect id="g" fill="url(#grandchild)" width="1" height="1"/>`+
		`<rect id="r" fill="url(#radial)" width="1" height="1"/></svg>`)
	c := gradientOf(t, d, "c")
	if c.X2 != (Length{1, UnitPx}) || c.X1 != (Length{0, UnitPx}) || c.Units != UnitsUserSpaceOnUse || c.Spread != SpreadRepeat || len(c.Stops) != 2 {
		t.Errorf("child = %+v", c)
	}
	g := gradientOf(t, d, "g")
	if len(g.Stops) != 1 || g.Stops[0].Color != col("lime") {
		t.Errorf("a gradient with stops keeps its own: %v", g.Stops)
	}
	if g.Y2 != (Length{1, UnitPx}) || g.X2 != (Length{1, UnitPx}) || g.Transform != Scale(2, 2) || g.Units != UnitsUserSpaceOnUse {
		t.Errorf("grandchild = %+v", g)
	}
	r := gradientOf(t, d, "r")
	if !r.Radial || r.R != (Length{0.3, UnitPx}) || len(r.Stops) != 2 || r.Units != UnitsUserSpaceOnUse {
		t.Errorf("radial from linear = %+v", r)
	}
	if r.CX != (Length{50, UnitPercent}) {
		t.Errorf("a radial gradient takes nothing of the linear one's geometry: cx %v", r.CX)
	}
}

func TestGradientEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		defs string
		fill string
		want PaintKind
	}{
		{"no stops is no paint", `<linearGradient id="g"/>`, "url(#g)", PaintKindNone},
		{"no stops with a fallback", `<linearGradient id="g"/>`, "url(#g) red", PaintKindColor},
		{"cycle of hrefs", `<linearGradient id="g" href="#h"><stop offset="0"/></linearGradient><linearGradient id="h" href="#g"/>`, "url(#g)", PaintKindServer},
		{"href to a missing id", `<linearGradient id="g" href="#nope"><stop offset="0"/></linearGradient>`, "url(#g)", PaintKindServer},
		{"href to something else", `<rect id="x" width="1" height="1"/><linearGradient id="g" href="#x"><stop offset="0"/></linearGradient>`, "url(#g)", PaintKindServer},
		{"fill refers to a rect", `<rect id="x" width="1" height="1"/>`, "url(#x)", PaintKindNone},
		{"pattern with a zero size is no paint", `<pattern id="g"/>`, "url(#g)", PaintKindNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.defs+`<rect id="r" fill="`+tt.fill+`" width="1" height="1"/></svg>`)
			if got := mustFind(t, d, "r").Style.Fill.Kind; got != tt.want {
				t.Errorf("kind = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<pattern id="base" x="1" y="2" width="10" height="20" patternUnits="userSpaceOnUse" patternTransform="rotate(45)" viewBox="0 0 5 5" preserveAspectRatio="none"><rect id="in" width="5" height="5" fill="red"/><circle r="1"/></pattern>`+
		`<pattern id="child" href="#base" width="30"/>`+
		`<pattern id="own" href="#base" patternContentUnits="objectBoundingBox"><rect id="own-rect" width="1" height="1"/></pattern>`+
		`<rect id="a" fill="url(#base)" width="1" height="1"/><rect id="b" fill="url(#child)" width="1" height="1"/><rect id="c" fill="url(#own)" width="1" height="1"/>`+
		`<rect id="loop" fill="url(#self)" width="1" height="1"/>`+
		`<pattern id="self" width="4" height="4"><rect id="self-rect" fill="url(#self)" width="4" height="4"/></pattern></svg>`)
	a := mustFind(t, d, "a").Style.Fill.Server.(*Pattern)
	if a.Units != UnitsUserSpaceOnUse || a.ContentUnits != UnitsUserSpaceOnUse || a.X.Value != 1 || a.Y.Value != 2 || a.Width.Value != 10 || a.Height.Value != 20 {
		t.Errorf("pattern = %+v", a)
	}
	if !affEq(a.Transform, Rotate(45)) || a.ViewBox == nil || *a.ViewBox != (ViewBox{0, 0, 5, 5}) || a.PreserveAspectRatio.Align != AlignNone {
		t.Errorf("pattern = %+v", a)
	}
	if len(a.Children) != 2 || a.Children[0].Style.Fill.Color != col("red") {
		t.Errorf("children = %v", a.Children)
	}
	b := mustFind(t, d, "b").Style.Fill.Server.(*Pattern)
	if b.Width.Value != 30 || b.Height.Value != 20 || len(b.Children) != 2 || b.Units != UnitsUserSpaceOnUse {
		t.Errorf("child pattern = %+v", b)
	}
	c := mustFind(t, d, "c").Style.Fill.Server.(*Pattern)
	if len(c.Children) != 1 || c.ContentUnits != UnitsObjectBoundingBox {
		t.Errorf("own pattern = %+v", c)
	}
	// A pattern that paints with itself: the inner reference is dropped.
	loop := mustFind(t, d, "loop").Style.Fill.Server.(*Pattern)
	if len(loop.Children) != 1 || loop.Children[0].Style.Fill.Kind != PaintKindNone {
		t.Errorf("self-referencing pattern = %+v", loop.Children)
	}
}

func TestClipPath(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<clipPath id="c" clipPathUnits="objectBoundingBox" transform="scale(2)" clip-rule="evenodd">`+
		`<rect id="in" width="1" height="1" fill="red" opacity="0.2"/><circle r="1" display="none"/><circle r="1" visibility="hidden"/><g><rect width="1" height="1"/></g><text>t</text>`+
		`<use id="uu" href="#shape"/><use href="#grp"/></clipPath>`+
		`<rect id="shape" width="2" height="2"/><g id="grp"><rect width="1" height="1"/></g>`+
		`<clipPath id="outer" clip-path="url(#c)"><rect width="1" height="1"/></clipPath>`+
		`<clipPath id="self" clip-path="url(#self)"><rect width="1" height="1"/></clipPath>`+
		`<rect id="r" clip-path="url(#c)" width="10" height="10"/>`+
		`<rect id="o" clip-path="url(#outer)" width="10" height="10"/>`+
		`<rect id="s" clip-path="url(#self)" width="10" height="10"/>`+
		`<rect id="missing" clip-path="url(#nope)" width="10" height="10"/>`+
		`<rect id="wrong" clip-path="url(#shape)" width="10" height="10"/>`+
		`<rect id="none" clip-path="none" width="10" height="10"/></svg>`)
	r := mustFind(t, d, "r")
	if r.ClipPath == nil {
		t.Fatal("no clip path")
	}
	c := r.ClipPath
	if c.Units != UnitsObjectBoundingBox || c.Transform != Scale(2, 2) {
		t.Errorf("clip = %+v", c)
	}
	// rect, text, use of a shape: the group, the hidden ones and the use of a group are not in a clip path.
	if len(c.Children) != 3 {
		t.Fatalf("%d clip children: %v", len(c.Children), ids(c.Children))
	}
	if c.Children[0].Path.FillType() != ui.PathFillTypeEvenOdd {
		t.Error("the clip-rule is not the fill type")
	}
	if c.Children[0].Opacity != 1 {
		t.Error("opacity in a clip path")
	}
	if c.Children[1].Kind != NodeText || c.Children[2].Kind != NodeGroup || c.Children[2].Children[0].Kind != NodeShape {
		t.Errorf("clip children kinds = %d %d", c.Children[1].Kind, c.Children[2].Kind)
	}
	if o := mustFind(t, d, "o").ClipPath; o == nil || o.ClipPath != c {
		t.Errorf("outer clip = %+v", o)
	}
	if s := mustFind(t, d, "s").ClipPath; s == nil || s.ClipPath != nil {
		t.Errorf("a clip path that refers to itself: %+v", s)
	}
	for _, id := range []string{"missing", "wrong", "none"} {
		if mustFind(t, d, id).ClipPath != nil {
			t.Errorf("%s has a clip", id)
		}
	}
	// The default units of a clip path are userSpaceOnUse.
	d2 := mustParse(t, svgOpen+`<clipPath id="c"><rect width="1" height="1"/></clipPath><rect id="r" clip-path="url(#c)" width="1" height="1"/></svg>`)
	if mustFind(t, d2, "r").ClipPath.Units != UnitsUserSpaceOnUse {
		t.Error("default clipPathUnits")
	}
}

func TestMask(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<mask id="m" x="1" y="2" width="30%" height="4" maskUnits="userSpaceOnUse" maskContentUnits="objectBoundingBox" style="mask-type:alpha"><rect id="mr" width="1" height="1" fill="white"/></mask>`+
		`<mask id="d"><rect width="1" height="1"/></mask>`+
		`<mask id="loop" mask="url(#loop)"><rect width="1" height="1"/></mask>`+
		`<rect id="a" mask="url(#m)" width="10" height="10"/><rect id="b" mask="url(#d)" width="10" height="10"/>`+
		`<rect id="c" mask="url(#loop)" width="10" height="10"/><rect id="e" mask="url(#nope)" width="10" height="10"/></svg>`)
	m := mustFind(t, d, "a").Mask
	if m == nil {
		t.Fatal("no mask")
	}
	if m.Units != UnitsUserSpaceOnUse || m.ContentUnits != UnitsObjectBoundingBox || m.Type != MaskAlpha ||
		m.X != (Length{1, UnitPx}) || m.Y != (Length{2, UnitPx}) || m.Width != (Length{30, UnitPercent}) || m.Height != (Length{4, UnitPx}) {
		t.Errorf("mask = %+v", m)
	}
	if len(m.Children) != 1 {
		t.Errorf("children = %v", m.Children)
	}
	dm := mustFind(t, d, "b").Mask
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	if dm.Units != UnitsObjectBoundingBox || dm.ContentUnits != UnitsUserSpaceOnUse || dm.Type != MaskLuminance ||
		dm.X != pct(-10) || dm.Y != pct(-10) || dm.Width != pct(120) || dm.Height != pct(120) {
		t.Errorf("default mask = %+v", dm)
	}
	if c := mustFind(t, d, "c").Mask; c == nil || c.Mask != nil {
		t.Errorf("a mask that refers to itself: %+v", c)
	}
	if mustFind(t, d, "e").Mask != nil {
		t.Error("a missing mask")
	}
}

func TestFilter(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<filter id="f" x="1" y="2" width="30%" height="4" filterUnits="userSpaceOnUse" primitiveUnits="objectBoundingBox" color-interpolation-filters="sRGB">`+
		`<feGaussianBlur in="SourceGraphic" stdDeviation="2 3" result="blur"/>`+
		`<feFlood flood-color="red" flood-opacity="0.5" x="1" width="50%"/>`+
		`<feOffset in="blur" dx="2" dy="3" result="o"/>`+
		`<feMerge><feMergeNode in="o"/><feMergeNode in="SourceGraphic"/></feMerge>`+
		`<feComponentTransfer><feFuncR type="linear" slope="2"/></feComponentTransfer>`+
		`<feDiffuseLighting lighting-color="blue"><fePointLight x="1" y="2" z="3"/></feDiffuseLighting></filter>`+
		`<filter id="base"><feOffset dx="1"/></filter><filter id="ref" href="#base" x="5"/>`+
		`<filter id="loop" href="#loop"><feOffset/></filter>`+
		`<rect id="a" filter="url(#f)" width="10" height="10"/>`+
		`<rect id="b" filter="url(#base) url(#nope) blur(3px) url(#ref)" width="10" height="10"/>`+
		`<rect id="c" filter="none" width="10" height="10"/><rect id="e" filter="url(#loop)" width="10" height="10"/>`+
		`<rect id="g" filter="blur(3px)" width="10" height="10"/></svg>`)
	f := mustFind(t, d, "a").Filters
	if len(f) != 1 {
		t.Fatalf("filters = %v", f)
	}
	flt := f[0]
	if flt.Units != UnitsUserSpaceOnUse || flt.PrimitiveUnits != UnitsObjectBoundingBox || flt.X != (Length{1, UnitPx}) || flt.Width != (Length{30, UnitPercent}) {
		t.Errorf("filter = %+v", flt)
	}
	if len(flt.Primitives) != 6 {
		t.Fatalf("%d primitives", len(flt.Primitives))
	}
	blur := flt.Primitives[0]
	if blur.Name != "feGaussianBlur" || blur.In != "SourceGraphic" || blur.Result != "blur" || blur.Attrs["stdDeviation"] != "2 3" {
		t.Errorf("blur = %+v", blur)
	}
	if blur.Style.ColorInterpolationFilters != ColorInterpolationSRGB {
		t.Error("color-interpolation-filters is not inherited from the filter")
	}
	if _, ok := blur.Attrs["in"]; ok {
		t.Error("in is in Attrs")
	}
	flood := flt.Primitives[1]
	if flood.FloodColor != col("red") || flood.FloodOpacity != 0.5 || flood.X == nil || *flood.X != (Length{1, UnitPx}) || flood.Y != nil || *flood.Width != (Length{50, UnitPercent}) {
		t.Errorf("flood = %+v", flood)
	}
	off := flt.Primitives[2]
	if off.In != "blur" || off.Result != "o" || off.Attrs["dx"] != "2" {
		t.Errorf("offset = %+v", off)
	}
	merge := flt.Primitives[3]
	if len(merge.Children) != 2 || merge.Children[1].In != "SourceGraphic" {
		t.Errorf("merge = %+v", merge)
	}
	if ct := flt.Primitives[4]; len(ct.Children) != 1 || ct.Children[0].Name != "feFuncR" || ct.Children[0].Attrs["slope"] != "2" {
		t.Errorf("component transfer = %+v", ct)
	}
	if li := flt.Primitives[5]; li.FloodColor != col("blue") || len(li.Children) != 1 || li.Children[0].Name != "fePointLight" {
		t.Errorf("lighting = %+v", li)
	}
	// Defaults, and the chain of hrefs.
	b := mustFind(t, d, "b").Filters
	if len(b) != 2 {
		t.Fatalf("b filters = %d", len(b))
	}
	if b[1].X != (Length{5, UnitPx}) || len(b[1].Primitives) != 1 || b[1].Primitives[0].Attrs["dx"] != "1" {
		t.Errorf("filter with href = %+v", b[1])
	}
	pct := func(v float64) Length { return Length{v, UnitPercent} }
	if b[0].X != pct(-10) || b[0].Width != pct(120) || b[0].Units != UnitsObjectBoundingBox || b[0].PrimitiveUnits != UnitsUserSpaceOnUse {
		t.Errorf("default filter = %+v", b[0])
	}
	if len(mustFind(t, d, "c").Filters) != 0 || len(mustFind(t, d, "g").Filters) != 0 {
		t.Error("none and filter functions leave filters")
	}
	if e := mustFind(t, d, "e").Filters; len(e) != 1 || len(e[0].Primitives) != 1 {
		t.Errorf("looping filter = %+v", e)
	}
	flood0 := mustParse(t, svgOpen+`<filter id="f"><feFlood/></filter><rect id="r" filter="url(#f)" width="1" height="1"/></svg>`)
	if p := mustFind(t, flood0, "r").Filters[0].Primitives[0]; p.FloodColor != col("black") || p.FloodOpacity != 1 {
		t.Errorf("flood defaults = %+v", p)
	}
}

func TestMarkers(t *testing.T) {
	d := mustParse(t, svgOpen+
		`<marker id="m" refX="2" refY="3" markerWidth="6" markerHeight="7" markerUnits="userSpaceOnUse" orient="auto" viewBox="0 0 4 4" overflow="visible"><path d="M0 0L4 4" stroke="red"/></marker>`+
		`<marker id="d" orient="30deg"/><marker id="r" orient="auto-start-reverse" markerWidth="0"/>`+
		`<path id="p" d="M0 0L10 10" marker-start="url(#m)" marker-mid="url(#d)" marker-end="url(#m)"/>`+
		`<path id="q" d="M0 0L10 10" style="marker:url(#d)"/>`+
		`<path id="n" d="M0 0L10 10" marker-start="url(#nope)" marker-end="url(#r)"/>`+
		`<g marker-start="url(#m)"><path id="inh" d="M0 0L1 1"/><path id="off" marker-start="none" d="M0 0L1 1"/></g></svg>`)
	p := mustFind(t, d, "p").Style
	m := p.MarkerStart
	if m == nil || m != p.MarkerEnd {
		t.Fatalf("markers = %v %v", p.MarkerStart, p.MarkerEnd)
	}
	if m.RefX != 2 || m.RefY != 3 || m.Width != 6 || m.Height != 7 || m.StrokeWidthUnits || m.Orient != OrientAuto || !m.Overflow ||
		m.ViewBox == nil || *m.ViewBox != (ViewBox{0, 0, 4, 4}) || len(m.Children) != 1 {
		t.Errorf("marker = %+v", m)
	}
	if got := m.Children[0].Style.Stroke.Color; got != col("red") {
		t.Errorf("marker content stroke = %v", got)
	}
	dm := p.MarkerMid
	if dm == nil || dm.Orient != OrientAngle || dm.Angle != 30 || dm.Width != 3 || dm.Height != 3 || !dm.StrokeWidthUnits || dm.Overflow {
		t.Errorf("default marker = %+v", dm)
	}
	if q := mustFind(t, d, "q").Style; q.MarkerStart != dm || q.MarkerMid != dm || q.MarkerEnd != dm {
		t.Errorf("marker shorthand: %v %v %v", q.MarkerStart, q.MarkerMid, q.MarkerEnd)
	}
	n := mustFind(t, d, "n").Style
	if n.MarkerStart != nil || n.MarkerEnd != nil {
		t.Errorf("missing or empty markers: %v %v", n.MarkerStart, n.MarkerEnd)
	}
	if mustFind(t, d, "inh").Style.MarkerStart != m || mustFind(t, d, "off").Style.MarkerStart != nil {
		t.Error("marker inheritance")
	}
}

func TestMarkerThatContainsItself(t *testing.T) {
	d := mustParse(t, svgOpen+`<marker id="m"><path d="M0 0L1 1" marker-end="url(#m)"/></marker><path id="p" d="M0 0L5 5" marker-end="url(#m)"/></svg>`)
	m := mustFind(t, d, "p").Style.MarkerEnd
	if m == nil || len(m.Children) != 1 || m.Children[0].Style.MarkerEnd != nil {
		t.Errorf("marker = %+v", m)
	}
}
