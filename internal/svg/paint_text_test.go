package svg

import (
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/ui"
)

// layoutOf lays out the first text node of a document body with the bundled fonts.
func layoutOf(t *testing.T, body string) *textBlock {
	t.Helper()
	d := mustParse(t, svgOpen+body+`</svg>`)
	n := mustFind2(t, d)
	p := newPainter(ui.NewCanvas(ui.NewPictureRecorder()), 1)
	p.doc = d
	tb := p.textBlockOf(n)
	if tb == nil {
		t.Fatalf("the text %q has no layout", body)
	}
	return tb
}

// xs returns the x of the origin of each cluster and its advance.
func xs(tb *textBlock) (x, adv []float64) {
	for _, it := range tb.items {
		x = append(x, it.g.E)
		adv = append(adv, it.c.X1-it.c.X0)
	}
	return
}

func approx(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestTextPlacement(t *testing.T) {
	tests := []struct {
		name string
		body string
		// check gets the positions and the advances of the clusters.
		check func(t *testing.T, tb *textBlock, x, adv []float64)
	}{
		{"the first character is at x and y", `<text x="10" y="40" font-size="10">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(x[0], 10, 1e-9) || !approx(tb.items[0].g.F, 40, 1e-9) {
				t.Errorf("origin = %v, %v", x[0], tb.items[0].g.F)
			}
			if !approx(x[1], 10+adv[0], 1e-6) || !approx(x[2], 10+adv[0]+adv[1], 1e-6) {
				t.Errorf("the characters do not follow each other: %v advances %v", x, adv)
			}
		}},
		{"no position is the origin", `<text font-size="10">ab</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[0] != 0 || tb.items[0].g.F != 0 {
				t.Errorf("origin = %v, %v", x[0], tb.items[0].g.F)
			}
		}},
		{"a list of x", `<text x="10 30 50" y="20" font-size="10">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[0] != 10 || x[1] != 30 || x[2] != 50 {
				t.Errorf("x = %v", x)
			}
		}},
		{"a list of y", `<text x="5" y="10 20 30" font-size="10">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			for i, want := range []float64{10, 20, 30} {
				if tb.items[i].g.F != want {
					t.Errorf("y[%d] = %v, want %v", i, tb.items[i].g.F, want)
				}
			}
		}},
		{"a shorter list: the rest follow", `<text x="10 50" y="20" font-size="10">abcd</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[1] != 50 || !approx(x[2], 50+adv[1], 1e-6) || !approx(x[3], 50+adv[1]+adv[2], 1e-6) {
				t.Errorf("x = %v", x)
			}
		}},
		{"dx moves the character and the ones after it", `<text x="10" y="20" dx="0 5 10" font-size="10">abcd</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			want := []float64{10, 10 + adv[0] + 5, 10 + adv[0] + 5 + adv[1] + 10, 10 + adv[0] + 5 + adv[1] + 10 + adv[2]}
			for i := range want {
				if !approx(x[i], want[i], 1e-6) {
					t.Errorf("x[%d] = %v, want %v", i, x[i], want[i])
				}
			}
		}},
		{"dy moves the character and the ones after it", `<text x="10" y="20" dy="0 5 5" font-size="10">abcd</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			for i, want := range []float64{20, 25, 30, 30} {
				if tb.items[i].g.F != want {
					t.Errorf("y[%d] = %v, want %v", i, tb.items[i].g.F, want)
				}
			}
		}},
		{"a tspan with x starts a chunk", `<text x="10" y="20" font-size="10">ab<tspan x="60" y="40">cd</tspan>ef</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[2] != 60 || tb.items[2].g.F != 40 || !approx(x[3], 60+adv[2], 1e-6) {
				t.Errorf("x = %v", x)
			}
			if !approx(x[4], 60+adv[2]+adv[3], 1e-6) || tb.items[4].g.F != 40 {
				t.Errorf("the text after the tspan: x = %v y = %v", x[4], tb.items[4].g.F)
			}
		}},
		{"a tspan with dx only", `<text x="10" y="20" font-size="10">ab<tspan dx="7" dy="3">cd</tspan></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(x[2], 10+adv[0]+adv[1]+7, 1e-6) || tb.items[2].g.F != 23 {
				t.Errorf("x = %v y = %v", x[2], tb.items[2].g.F)
			}
		}},
		{"anchor start", `<text x="50" y="20" font-size="10" text-anchor="start">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[0] != 50 {
				t.Errorf("x = %v", x[0])
			}
		}},
		{"anchor middle", `<text x="50" y="20" font-size="10" text-anchor="middle">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			w := adv[0] + adv[1] + adv[2]
			if !approx(x[0], 50-w/2, 1e-6) {
				t.Errorf("x = %v, want %v", x[0], 50-w/2)
			}
		}},
		{"anchor end", `<text x="50" y="20" font-size="10" text-anchor="end">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			w := adv[0] + adv[1] + adv[2]
			if !approx(x[0], 50-w, 1e-6) {
				t.Errorf("x = %v, want %v", x[0], 50-w)
			}
		}},
		{"anchor per chunk", `<text y="20" font-size="10" text-anchor="end"><tspan x="40">ab</tspan><tspan x="90">cde</tspan></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(x[0], 40-adv[0]-adv[1], 1e-6) || !approx(x[2], 90-adv[2]-adv[3]-adv[4], 1e-6) {
				t.Errorf("x = %v advances %v", x, adv)
			}
		}},
		{"the anchor is the one of the first character of the chunk", `<text x="50" y="20" font-size="10" text-anchor="middle">ab<tspan text-anchor="end">cd</tspan></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			w := adv[0] + adv[1] + adv[2] + adv[3]
			if !approx(x[0], 50-w/2, 1e-6) {
				t.Errorf("x = %v, want %v", x[0], 50-w/2)
			}
		}},
		{"anchor with dx counts the dx in the width", `<text x="50" y="20" dx="10" font-size="10" text-anchor="end">ab</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			w := 10 + adv[0] + adv[1]
			if !approx(x[0], 50+10-w, 1e-6) {
				t.Errorf("x = %v, want %v", x[0], 60-w)
			}
		}},
		{"rotate turns each character about its origin", `<text x="10 40 70" y="50" rotate="0 90 180" font-size="10">abc</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			for i, want := range []float64{0, 90, 180} {
				g := tb.items[i].g
				got := math.Atan2(g.B, g.A) * 180 / math.Pi
				if d := math.Mod(math.Abs(got-want), 360); d > 1e-6 && d < 360-1e-6 {
					t.Errorf("rotation of %d = %v, want %v", i, got, want)
				}
				if !approx(g.E, []float64{10, 40, 70}[i], 1e-9) || !approx(g.F, 50, 1e-9) {
					t.Errorf("the origin of %d moved: %v, %v", i, g.E, g.F)
				}
			}
		}},
		{"the last rotate value repeats", `<text x="10" y="50" rotate="30 60" font-size="10">abcd</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			for i, want := range []float64{30, 60, 60, 60} {
				g := tb.items[i].g
				if got := math.Atan2(g.B, g.A) * 180 / math.Pi; !approx(got, want, 1e-6) {
					t.Errorf("rotation of %d = %v, want %v", i, got, want)
				}
			}
		}},
		{"a tspan's rotate is for its own characters", `<text x="10" y="50" font-size="10">a<tspan rotate="45">bc</tspan>d</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			for i, want := range []float64{0, 45, 45, 0} {
				g := tb.items[i].g
				if got := math.Atan2(g.B, g.A) * 180 / math.Pi; !approx(got, want, 1e-6) {
					t.Errorf("rotation of %d = %v, want %v", i, got, want)
				}
			}
		}},
		{"a tspan's x list counts from its first character", `<text y="20" font-size="10">ab<tspan x="30 60">cd</tspan></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if x[2] != 30 || x[3] != 60 {
				t.Errorf("x = %v", x)
			}
		}},
		{"letter spacing widens the advance", `<text x="0" y="20" font-size="10" letter-spacing="3">ab</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			plain := layoutOf(t, `<text x="0" y="20" font-size="10">ab</text>`)
			_, pa := xs(plain)
			if !approx(adv[0], pa[0]+3, 1e-6) {
				t.Errorf("advance = %v, want %v", adv[0], pa[0]+3)
			}
		}},
		{"word spacing widens the space", `<text x="0" y="20" font-size="10" word-spacing="5">a b</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			plain := layoutOf(t, `<text x="0" y="20" font-size="10">a b</text>`)
			_, pa := xs(plain)
			if !approx(adv[1], pa[1]+5, 1e-6) {
				t.Errorf("advance of the space = %v, want %v", adv[1], pa[1]+5)
			}
		}},
		{"the font size scales the advances", `<text x="0" y="20" font-size="20">ab</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			small := layoutOf(t, `<text x="0" y="20" font-size="10">ab</text>`)
			_, sa := xs(small)
			if !approx(adv[0], 2*sa[0], 1e-3) {
				t.Errorf("advance = %v, want %v", adv[0], 2*sa[0])
			}
		}},
		{"white space collapses before the layout", `<text x="0" y="20" font-size="10">a    b</text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if len(tb.items) != 3 {
				t.Errorf("%d clusters, want 3", len(tb.items))
			}
		}},
		{"a text on a horizontal path", `<defs><path id="p" d="M 10 50 L 110 50"/></defs><text font-size="10"><textPath href="#p">abc</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			// The middle of each character is on the path, and the origin is half an advance before it.
			sum := 0.0
			for i := range adv {
				g := tb.items[i].g
				if !approx(g.E, 10+sum, 1e-6) || !approx(g.F, 50, 1e-6) {
					t.Errorf("origin of %d = %v, %v, want %v, 50", i, g.E, g.F, 10+sum)
				}
				sum += adv[i]
			}
		}},
		{"startOffset moves the text along the path", `<defs><path id="p" d="M 10 50 L 110 50"/></defs><text font-size="10"><textPath href="#p" startOffset="20">ab</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(x[0], 30, 1e-6) {
				t.Errorf("x = %v, want 30", x[0])
			}
		}},
		{"a startOffset in percent is of the length", `<defs><path id="p" d="M 10 50 L 110 50"/></defs><text font-size="10"><textPath href="#p" startOffset="50%">ab</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(x[0], 60, 1e-6) {
				t.Errorf("x = %v, want 60", x[0])
			}
		}},
		{"the anchor shifts the text on a path", `<defs><path id="p" d="M 10 50 L 110 50"/></defs><text font-size="10" text-anchor="middle"><textPath href="#p" startOffset="50%">ab</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			w := adv[0] + adv[1]
			if !approx(x[0], 60-w/2, 1e-6) {
				t.Errorf("x = %v, want %v", x[0], 60-w/2)
			}
		}},
		{"on a vertical path the characters are turned a quarter", `<defs><path id="p" d="M 50 10 L 50 110"/></defs><text font-size="10"><textPath href="#p">ab</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			g := tb.items[0].g
			if got := math.Atan2(g.B, g.A) * 180 / math.Pi; !approx(got, 90, 1e-6) {
				t.Errorf("rotation = %v, want 90", got)
			}
			if !approx(g.E, 50, 1e-6) || !approx(g.F, 10, 1e-6) {
				t.Errorf("origin = %v, %v", g.E, g.F)
			}
		}},
		{"characters beyond the end of the path are not drawn", `<defs><path id="p" d="M 10 50 L 30 50"/></defs><text font-size="10"><textPath href="#p">abcdefghij</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			placed := 0
			for _, it := range tb.items {
				if it.placed {
					placed++
				}
			}
			if placed == 0 || placed >= len(tb.items) {
				t.Errorf("%d of %d characters are placed", placed, len(tb.items))
			}
		}},
		{"dy moves a character on a path off it", `<defs><path id="p" d="M 10 50 L 110 50"/></defs><text font-size="10"><textPath href="#p"><tspan dy="-4">a</tspan></textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !approx(tb.items[0].g.F, 46, 1e-6) {
				t.Errorf("y = %v, want 46", tb.items[0].g.F)
			}
		}},
		{"a curved path turns the characters", `<defs><path id="p" d="M 10 50 Q 35 10 60 50"/></defs><text font-size="10"><textPath href="#p">abcdefghijk</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			first, last := tb.items[0].g, tb.items[len(tb.items)-1].g
			a0 := math.Atan2(first.B, first.A)
			a1 := math.Atan2(last.B, last.A)
			if !(a0 < -0.1 && a1 > 0.1) {
				t.Errorf("angles %v and %v: the text should rise and then fall", a0, a1)
			}
		}},
		{"text on a path that does not exist is not drawn", `<text font-size="10">ab<textPath href="#nothing">cd</textPath></text>`, func(t *testing.T, tb *textBlock, x, adv []float64) {
			if !tb.items[0].placed || tb.items[2].placed || tb.items[3].placed {
				t.Errorf("placed: %v %v %v %v", tb.items[0].placed, tb.items[1].placed, tb.items[2].placed, tb.items[3].placed)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := layoutOf(t, tt.body)
			x, adv := xs(tb)
			tt.check(t, tb, x, adv)
		})
	}
}

func TestTextRightToLeft(t *testing.T) {
	// The characters are in logical order; in right-to-left text the first one is the rightmost.
	tb := layoutOf(t, `<text x="10" y="20" font-size="12">`+"אבגד"+`</text>`)
	if len(tb.items) != 4 {
		t.Fatalf("%d clusters", len(tb.items))
	}
	for i := 1; i < 4; i++ {
		if !(tb.items[i].g.E < tb.items[i-1].g.E) {
			t.Errorf("cluster %d at %v is not left of cluster %d at %v", i, tb.items[i].g.E, i-1, tb.items[i-1].g.E)
		}
		if !tb.items[i].c.RTL {
			t.Errorf("cluster %d is not right to left", i)
		}
	}
	// The chunk starts at x with its left edge, and the anchor works on the whole width.
	left := tb.items[3].g.E
	if !approx(left, 10, 1e-6) {
		t.Errorf("the left edge is at %v, want 10", left)
	}
	end := layoutOf(t, `<text x="100" y="20" font-size="12" text-anchor="end">`+"אבגד"+`</text>`)
	right := end.items[0].g.E + (end.items[0].c.X1 - end.items[0].c.X0)
	if !approx(right, 100, 1e-6) {
		t.Errorf("the right edge is at %v, want 100", right)
	}
	mid := layoutOf(t, `<text x="100" y="20" font-size="12" text-anchor="middle">`+"אבגד"+`</text>`)
	l, r := mid.items[3].g.E, mid.items[0].g.E+(mid.items[0].c.X1-mid.items[0].c.X0)
	if !approx((l+r)/2, 100, 1e-6) {
		t.Errorf("the middle is at %v, want 100", (l+r)/2)
	}
	// Mixed text: the Latin letters keep their order, left to right, inside the right-to-left text.
	mixed := layoutOf(t, `<text x="10" y="20" font-size="12">`+"אב abc ג"+`</text>`)
	var latin []float64
	for _, it := range mixed.items {
		if !it.c.RTL && it.c.End-it.c.Start == 1 {
			latin = append(latin, it.g.E)
		}
	}
	for i := 1; i < len(latin); i++ {
		if !(latin[i] > latin[i-1]) {
			t.Errorf("the Latin characters are not left to right: %v", latin)
			break
		}
	}
}

func TestTextBounds(t *testing.T) {
	tb := layoutOf(t, `<text x="10" y="40" font-size="20">abc</text>`)
	x, adv := xs(tb)
	_ = x
	w := adv[0] + adv[1] + adv[2]
	r := tb.cells.r
	if !approx(r.Left, 10, 1e-6) || !approx(r.Width(), w, 1e-6) {
		t.Errorf("cells = %v, want left 10 and width %v", r, w)
	}
	if r.Top >= 40 || r.Bottom <= 40 || r.Height() < 20 || r.Height() > 30 {
		t.Errorf("cells = %v: the box should reach above and below the baseline", r)
	}
	// The bounds of a text element are those of its cells, for the units of paint servers, clips and filters.
	d := mustParse(t, svgOpen+`<text id="t" x="10" y="40" font-size="20">abc</text></svg>`)
	p := newPainter(ui.NewCanvas(ui.NewPictureRecorder()), 1)
	p.doc = d
	bb, ok := p.objectBBox(mustFind(t, d, "t"))
	if !ok || !approx(bb.Width(), w, 1e-6) {
		t.Errorf("objectBBox = %v, %v", bb, ok)
	}
}

func TestTextDrawing(t *testing.T) {
	const white = `<rect width="100" height="100" fill="#fff"/>`
	count := func(get func(x, y int) [4]uint8, pred func(p [4]uint8) bool) int {
		n := 0
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				if pred(get(x, y)) {
					n++
				}
			}
		}
		return n
	}
	dark := func(p [4]uint8) bool { return p[0] < 128 && p[1] < 128 && p[2] < 128 }
	red := func(p [4]uint8) bool { return p[0] > 200 && p[1] < 80 }
	blue := func(p [4]uint8) bool { return p[2] > 200 && p[0] < 80 }
	tests := []struct {
		name string
		body string
		pred func(p [4]uint8) bool
		min  int
		max  int
	}{
		{"text is drawn with the fill", white + `<text x="10" y="50" font-size="30" fill="#f00">Hi</text>`, red, 100, 1500},
		{"fill none draws nothing", white + `<text x="10" y="50" font-size="30" fill="none">Hi</text>`, dark, 0, 0},
		{"visibility hidden draws nothing", white + `<text x="10" y="50" font-size="30" visibility="hidden">Hi</text>`, dark, 0, 0},
		{"display none draws nothing", white + `<text x="10" y="50" font-size="30" display="none">Hi</text>`, dark, 0, 0},
		{"a stroke draws the outline", white + `<text x="10" y="50" font-size="30" fill="none" stroke="#00f" stroke-width="2">Hi</text>`, blue, 50, 1500},
		{"a tspan has its own fill", white + `<text x="10" y="50" font-size="30" fill="#f00">H<tspan fill="#00f">i</tspan></text>`, blue, 20, 800},
		{"a tspan with fill none leaves its characters out", white + `<text x="10" y="50" font-size="30" fill="#f00">H<tspan fill="none">i</tspan></text>`, red, 20, 700},
		{"a text with no characters draws nothing", white + `<text x="10" y="50" font-size="30"></text>`, dark, 0, 0},
		{"text outside the picture does not draw into it", white + `<text x="500" y="50" font-size="30">Hi</text>`, dark, 0, 0},
		{"underline draws a line", white + `<text x="10" y="50" font-size="30" fill="#f00" text-decoration="underline">I</text>`, red, 50, 1500},
		{"a clip path of text", white + `<clipPath id="c"><text x="10" y="50" font-size="40">I</text></clipPath><rect width="100" height="100" fill="#f00" clip-path="url(#c)"/>`, red, 40, 800},
		{"a mask of text", white + `<mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><text x="10" y="50" font-size="40" fill="#fff">I</text></mask><rect width="100" height="100" fill="#f00" mask="url(#m)"/>`, red, 40, 800},
		{"a filter on text", white + `<filter id="f" filterUnits="userSpaceOnUse" x="0" y="0" width="100" height="100"><feOffset dx="30"/></filter><text x="10" y="50" font-size="40" fill="#f00" filter="url(#f)">I</text>`, red, 40, 800},
		{"text with a gradient with object bounding box units", white + `<linearGradient id="g"><stop offset="0" stop-color="#f00"/><stop offset="1" stop-color="#f00"/></linearGradient><text x="10" y="50" font-size="40" fill="url(#g)">I</text>`, red, 40, 800},
		{"text with a pattern", white + `<pattern id="p" width="4" height="4" patternUnits="userSpaceOnUse"><rect width="4" height="4" fill="#f00"/></pattern><text x="10" y="50" font-size="40" fill="url(#p)">I</text>`, red, 40, 800},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			get := docAt(t, tt.body)
			n := count(get, tt.pred)
			if n < tt.min || n > tt.max {
				t.Errorf("%d matching pixels, want %d to %d", n, tt.min, tt.max)
			}
		})
	}
}

func TestPictureWithTheAppsTextEngine(t *testing.T) {
	d := mustParse(t, svgOpen+`<text x="10" y="50" font-size="30">Hi</text></svg>`)
	// A text engine of the app is used instead of the bundled one.
	spy := &spyEngine{inner: d.defaultEngine()}
	pic := d.PictureWith(PictureOpts{Scale: 1, Text: spy})
	if spy.built == 0 {
		t.Error("the engine of the options was not used")
	}
	if len(pic.DisplayList().Ops()) == 0 {
		t.Error("the picture is empty")
	}
	// An engine whose paragraphs give no clusters draws nothing, and does not panic.
	pic = d.PictureWith(PictureOpts{Text: bareEngine{}})
	if n := countOps[ui.OpDrawPath](pic.DisplayList().Ops()); n != 0 {
		t.Errorf("%d paths from an engine with no clusters", n)
	}
}

type spyEngine struct {
	inner ui.ParagraphFactory
	built int
}

func (s *spyEngine) BuildParagraph(c *ui.ParagraphContent) ui.ParagraphLayout {
	s.built++
	return s.inner.BuildParagraph(c)
}

// bareEngine makes paragraphs that cannot say where their glyphs are.
type bareEngine struct{}

func (bareEngine) BuildParagraph(c *ui.ParagraphContent) ui.ParagraphLayout { return bareLayout{} }

type bareLayout struct{ ui.ParagraphLayout }

func (bareLayout) Layout(ui.ParagraphConstraints) {}

func TestPictureAtIsSafeToCallFromManyGoroutines(t *testing.T) {
	d := mustParse(t, svgOpen+`<text x="10" y="50" font-size="30">Hi there</text></svg>`)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if len(d.PictureAt(1).DisplayList().Ops()) == 0 {
				t.Error("empty picture")
			}
		}()
	}
	wg.Wait()
}

func TestTextOfADocumentWithManySpans(t *testing.T) {
	// A span for each character: the groups are made once each, and the outline holds them all.
	var sb strings.Builder
	sb.WriteString(`<text x="2" y="50" font-size="8">`)
	for i := 0; i < 200; i++ {
		sb.WriteString(`<tspan fill="#f00">a</tspan>`)
	}
	sb.WriteString(`</text>`)
	tb := layoutOf(t, sb.String())
	if len(tb.groups) != 200 {
		t.Errorf("%d groups, want 200", len(tb.groups))
	}
	if tb.outline == nil || tb.outline.IsEmpty() {
		t.Error("no outline")
	}
}

func TestClustersOfTheEngine(t *testing.T) {
	d := mustParse(t, svgOpen+`<text x="0" y="0">a</text></svg>`)
	e := d.defaultEngine()
	style := ui.NewParagraphStyle()
	b := ui.NewParagraphBuilder(e, style)
	size := 20.0
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontSize: &size, FontFamily: "Roboto"}))
	b.AddText("fi aéx")
	para := b.Build()
	para.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	cl, ok := para.Layouter().(layout.Clusterer)
	if !ok {
		t.Fatal("the paragraph has no clusters")
	}
	cs := cl.Clusters()
	if len(cs) == 0 {
		t.Fatal("no clusters")
	}
	end := 0
	for i, c := range cs {
		if c.Start != end {
			t.Errorf("cluster %d starts at %d, want %d (no gaps, no overlaps)", i, c.Start, end)
		}
		end = c.End
		if !(c.X1 >= c.X0) {
			t.Errorf("cluster %d has width %v", i, c.X1-c.X0)
		}
		if c.Size != size || c.Ascent <= 0 || c.Descent <= 0 {
			t.Errorf("cluster %d: size %v ascent %v descent %v", i, c.Size, c.Ascent, c.Descent)
		}
	}
	if end != len("fi aéx") {
		t.Errorf("the clusters end at %d, want %d", end, len("fi aéx"))
	}
	// The cluster of "e" and its accent is one cluster of three bytes, with one or two glyphs.
	found := false
	for _, c := range cs {
		if c.End-c.Start == 3 {
			found = true
		}
	}
	if !found {
		t.Error("no cluster holds the letter and its accent")
	}
}
