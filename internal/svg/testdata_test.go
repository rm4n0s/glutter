package svg

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// kindCount counts the nodes of each kind.
func kindCount(n *Node, into map[NodeKind]int) {
	into[n.Kind]++
	for _, c := range n.Children {
		kindCount(c, into)
	}
}

func TestTestdataDocuments(t *testing.T) {
	tests := []struct {
		file   string
		size   ui.Size
		title  string
		nodes  map[NodeKind]int
		checks func(t *testing.T, d *Document)
	}{
		{
			file: "icon.svg", size: ui.Size{Width: 24, Height: 24}, title: "Favorite",
			nodes: map[NodeKind]int{NodeSvg: 1, NodeShape: 1},
			checks: func(t *testing.T, d *Document) {
				p := d.Root.Children[0]
				if p.Style.Fill.Color != ui.NewColorFromARGB(255, 0, 0, 0) {
					t.Errorf("currentColor fill = %v", p.Style.Fill.Color)
				}
				if b := p.Path.GetBounds(); math.Abs(b.Left-2) > 0.01 || math.Abs(b.Right-22) > 0.01 {
					t.Errorf("bounds = %v", b)
				}
			},
		},
		{
			file: "gradients.svg", size: ui.Size{Width: 200, Height: 100},
			nodes: map[NodeKind]int{NodeSvg: 1, NodeShape: 3},
			checks: func(t *testing.T, d *Document) {
				g, ok := d.Root.Children[2].Style.Fill.Server.(*Gradient)
				if !ok || len(g.Stops) != 2 || !affEq(g.Transform, Rotate(90)) {
					t.Errorf("sky2 = %+v", g)
				}
				if s, ok := d.Root.Children[1].Style.Fill.Server.(*Gradient); !ok || !s.Radial || s.FX != (Length{0.4, UnitPx}) {
					t.Errorf("sun = %+v", s)
				}
			},
		},
		{
			file: "styles.svg", size: ui.Size{Width: 100, Height: 100},
			nodes: map[NodeKind]int{NodeSvg: 1, NodeShape: 5, NodeGroup: 1},
			checks: func(t *testing.T, d *Document) {
				bg := d.Root.Children[0]
				if bg.Style.Fill.Color != col("#eee") || bg.Style.StrokeWidth != 2 || bg.Style.Stroke.Color != col("#333") {
					t.Errorf("bg style = %+v", bg.Style)
				}
				g := d.Root.Children[1]
				if g.Children[0].Style.Fill.Color != col("tomato") {
					t.Errorf("g.group > rect fill = %v", g.Children[0].Style.Fill.Color)
				}
				if g.Children[1].Style.Fill.Color != col("gold") {
					t.Errorf("#special (important) fill = %v", g.Children[1].Style.Fill.Color)
				}
				if c := g.Children[2]; c.Style.Fill.Color != col("purple") || c.Style.FillOpacity != 0.5 {
					t.Errorf("circle = %v %v", c.Style.Fill.Color, c.Style.FillOpacity)
				}
				if e := g.Children[3]; e.Style.Fill.Color != col("navy") || e.Style.FillOpacity != 0.5 {
					t.Errorf("ellipse = %v %v", e.Style.Fill.Color, e.Style.FillOpacity)
				}
			},
		},
		{
			file: "structure.svg", size: ui.Size{Width: 120, Height: 80},
			// the root, the symbol's viewport and the nested svg; use, use of g; the 2 circles, the polygon, the text, the rect
			nodes: map[NodeKind]int{NodeSvg: 3, NodeGroup: 4, NodeShape: 4, NodeText: 1},
			checks: func(t *testing.T, d *Document) {
				sw := d.Root.Children[2]
				if len(sw.Children) != 1 || flatten(sw.Children[0].Text) != "Hello" {
					t.Errorf("switch = %+v", sw.Children)
				}
				nested := d.Root.Children[3]
				if nested.Viewport.X != 30 || nested.Viewport.PreserveAspectRatio != (PreserveAspectRatio{AlignXMinYMin, true}) {
					t.Errorf("nested = %+v", nested.Viewport)
				}
			},
		},
		{
			file: "text.svg", size: ui.Size{Width: 200, Height: 60},
			nodes: map[NodeKind]int{NodeSvg: 1, NodeText: 3, NodeShape: 1},
			checks: func(t *testing.T, d *Document) {
				if got := flatten(d.Root.Children[1].Text); got != "Middle anchored text" {
					t.Errorf("text 2 = %q", got)
				}
				if got := flatten(d.Root.Children[0].Text); got != "ABC{def}" {
					t.Errorf("text 1 = %q", got)
				}
				if d.Root.Children[3].Text.Items[0].Span.PathData == nil {
					t.Error("textPath has no path")
				}
			},
		},
		{
			file: "effects.svg", size: ui.Size{Width: 100, Height: 100},
			nodes: map[NodeKind]int{NodeSvg: 1, NodeShape: 4, NodeGroup: 1},
			checks: func(t *testing.T, d *Document) {
				if _, ok := d.Root.Children[0].Style.Fill.Server.(*Pattern); !ok {
					t.Error("pattern fill")
				}
				g := d.Root.Children[1]
				if g.ClipPath == nil || g.Children[0].Mask == nil {
					t.Errorf("clip %v mask %v", g.ClipPath, g.Children[0].Mask)
				}
				if f := d.Root.Children[2].Filters; len(f) != 1 || len(f[0].Primitives) != 3 {
					t.Errorf("filters = %+v", f)
				}
				if d.Root.Children[3].Style.MarkerEnd == nil {
					t.Error("marker")
				}
			},
		},
		{
			file: "inkscape.svg", size: ui.Size{Width: 48 * 96 / 25.4, Height: 48 * 96 / 25.4},
			nodes: map[NodeKind]int{NodeSvg: 1, NodeGroup: 1, NodeShape: 1},
			checks: func(t *testing.T, d *Document) {
				p := mustFind(t, d, "path1")
				if p.Style.Fill.Color != col("#d4145a") {
					t.Errorf("entity fill = %v", p.Style.Fill.Color)
				}
				if b := p.Path.GetBounds(); b != ui.NewRectFromLTRB(8, 8, 40, 40) {
					t.Errorf("bounds = %v", b)
				}
			},
		},
	}
	files, err := filepath.Glob("testdata/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(tests) {
		t.Errorf("%d documents in testdata, %d in the test", len(files), len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if got := d.Size(); math.Abs(got.Width-tt.size.Width) > 1e-9 || math.Abs(got.Height-tt.size.Height) > 1e-9 {
				t.Errorf("size = %v, want %v", got, tt.size)
			}
			if d.Title != tt.title {
				t.Errorf("title = %q", d.Title)
			}
			got := map[NodeKind]int{}
			kindCount(d.Root, got)
			if fmt.Sprint(got) != fmt.Sprint(tt.nodes) {
				t.Errorf("nodes = %v, want %v", got, tt.nodes)
			}
			tt.checks(t, d)
		})
	}
}

// bigDocument makes a document of about n bytes: groups with classes, gradients, paths with long data, text, uses.
func bigDocument(n int) []byte {
	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 1000 1000">`)
	sb.WriteString(`<style>.a{fill:#336699;stroke:#000;stroke-width:1.5}.b{fill:url(#g0)}g.c>path{opacity:.8}#p5{fill:red}</style><defs>`)
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, `<linearGradient id="g%d" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#%06x"/><stop offset="1" stop-color="#%06x"/></linearGradient>`, i, i*0x0b0b0b, 0xffffff-i*0x0a0a0a)
	}
	sb.WriteString(`<path id="shape" d="M0 0 L10 0 L10 10 Z"/></defs>`)
	for i := 0; sb.Len() < n; i++ {
		fmt.Fprintf(&sb, `<g class="c" transform="translate(%d %d) rotate(%d)">`, i%900, (i*7)%900, i%360)
		for j := 0; j < 4; j++ {
			fmt.Fprintf(&sb, `<path id="p%d_%d" class="a" d="M%d.5 %d.25 C%d 10 20.5 30 40 50 S60 70 80 90 l-5 5 q3 4 5 6 a5 5 0 0 1 10 10 h3 v-4 z" fill-opacity="0.%d"/>`, i, j, j, i%10, j+1, j+1)
		}
		fmt.Fprintf(&sb, `<use xlink:href="#shape" x="%d" fill="url(#g%d)"/><text x="5" y="5" font-size="12">Label %d</text><circle class="b" cx="5" cy="5" r="3"/></g>`, i%50, i%20, i)
	}
	sb.WriteString(`</svg>`)
	return []byte(sb.String())
}

func BenchmarkParse(b *testing.B) {
	icon, err := os.ReadFile("testdata/icon.svg")
	if err != nil {
		b.Fatal(err)
	}
	structure, err := os.ReadFile("testdata/structure.svg")
	if err != nil {
		b.Fatal(err)
	}
	big := bigDocument(500 << 10)
	for _, bm := range []struct {
		name string
		data []byte
	}{
		{"icon", icon},
		{"structure", structure},
		{"large", big},
	} {
		b.Run(fmt.Sprintf("%s_%dB", bm.name, len(bm.data)), func(b *testing.B) {
			b.SetBytes(int64(len(bm.data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Parse(bm.data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestLargeDocumentParses(t *testing.T) {
	data := bigDocument(500 << 10)
	d := mustParse(t, string(data))
	if n := countNodes(d.Root); n < 5000 {
		t.Errorf("only %d nodes from %d bytes", n, len(data))
	}
}
