package layout

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// The tests of the tab stops use Go Mono, whose glyphs all advance the same, and measure what
// the text before a tab takes with paragraphs that have no tabs, so that the expected stops do
// not come from the code that is tested.

const tabFontSize = 10.0

// tabSpan is a piece of text in a style (nil is the paragraph's own).
type tabSpan struct {
	text  string
	style *ui.TextStyleOpts
}

// tabCase is a paragraph that the tests lay out in the ways they need.
type tabCase struct {
	spans    []tabSpan
	tabSize  float64
	width    float64
	dir      ui.TextDirection
	align    ui.TextAlign
	maxLines *int
	ellipsis string
}

func (c tabCase) paragraphStyle(tabSize float64) *ui.ParagraphStyle {
	return ui.NewParagraphStyle(ui.ParagraphStyleOpts{
		FontFamily: "Go Mono", FontSize: ui.Ptr(tabFontSize), TabSize: tabSize, TextDirection: c.dir, TextAlign: c.align,
		MaxLines: c.maxLines, Ellipsis: c.ellipsis,
	})
}

func (c tabCase) build(e *Engine, tabSize float64) *paragraph {
	b := ui.NewParagraphBuilder(e, c.paragraphStyle(tabSize))
	for _, s := range c.spans {
		if s.style == nil {
			b.AddText(s.text)
			continue
		}
		st := *s.style
		st.FontFamily = "Go Mono"
		b.PushStyle(ui.NewTextStyle(st))
		b.AddText(s.text)
		b.Pop()
	}
	p := b.Build().Layouter().(*paragraph)
	width := c.width
	if width == 0 {
		width = math.Inf(1)
	}
	p.Layout(ui.NewParagraphConstraints(width))
	return p
}

func spansOf(spans ...tabSpan) []tabSpan { return spans }

func plain(s string) tabSpan { return tabSpan{text: s} }

// refWidth is the width of the text in a style with no tab stops: a paragraph of the chunk alone.
func refWidth(t *testing.T, e *Engine, s string, style *ui.TextStyleOpts) float64 {
	t.Helper()
	if s == "" {
		return 0
	}
	c := tabCase{spans: spansOf(tabSpan{s, style})}
	p := c.build(e, 0)
	return p.MaxIntrinsicWidth()
}

// tabStop is where a tab of the oracle starts and ends.
type tabStop struct{ x0, x1 float64 }

// oracle is the expected tab stops of the spans on one line of a left-to-right paragraph, and the
// width of the line, from the widths of the pieces between the tabs.
func oracle(t *testing.T, e *Engine, spans []tabSpan, tabW float64) (stops []tabStop, total float64) {
	t.Helper()
	x := 0.0
	for _, sp := range spans {
		pieces := strings.Split(sp.text, "\t")
		for i, piece := range pieces {
			x += refWidth(t, e, piece, sp.style)
			if i == len(pieces)-1 {
				break
			}
			r := math.Mod(x, tabW)
			if r < 1e-6 || tabW-r < 1e-6 {
				r = 0
			}
			end := x + tabW - r
			if sp.style != nil && sp.style.LetterSpacing != nil {
				end += *sp.style.LetterSpacing
			}
			if sp.style != nil && sp.style.WordSpacing != nil {
				end += *sp.style.WordSpacing
			}
			stops = append(stops, tabStop{x, end})
			x = end
		}
	}
	return stops, x
}

// tabRunes are the indices of the tabs of the text.
func tabRunes(p *paragraph) []int {
	var out []int
	for i, r := range p.runes {
		if r == '\t' {
			out = append(out, i)
		}
	}
	return out
}

// boxOf is the box that the public query gives for the rune i.
func boxOf(t *testing.T, p *paragraph, i int) ui.TextBox {
	t.Helper()
	bs := p.GetBoxesForRange(p.byteOf(i), p.byteOf(i+1), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
	if len(bs) != 1 {
		t.Fatalf("rune %d has %d boxes, want 1", i, len(bs))
	}
	return bs[0]
}

func spaceUnit(t *testing.T, e *Engine) float64 {
	t.Helper()
	return refWidth(t, e, "a", nil)
}

func TestTabStopsAtColumns(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	if u <= 0 {
		t.Fatalf("a glyph of Go Mono has the advance %v", u)
	}
	tests := []struct {
		name    string
		tabSize float64
	}{
		{"1", 1}, {"2", 2}, {"4", 4}, {"8", 8}, {"2.5", 2.5},
	}
	for _, tt := range tests {
		for col := 0; col < 10; col++ {
			// A tab that begins on a stop advances a whole tab width, never 0.
			name := tt.name + "/column " + string(rune('0'+col))
			t.Run(name, func(t *testing.T) {
				c := tabCase{spans: spansOf(plain(strings.Repeat("a", col) + "\tX")), tabSize: tt.tabSize}
				p := c.build(e, tt.tabSize)
				end := (math.Floor(float64(col)/tt.tabSize+1e-9) + 1) * tt.tabSize * u
				b := boxOf(t, p, col)
				if math.Abs(b.Left-float64(col)*u) > 1e-3 || math.Abs(b.Right-end) > 1e-3 {
					t.Errorf("tab box = [%v, %v], want [%v, %v]", b.Left, b.Right, float64(col)*u, end)
				}
				if x := boxOf(t, p, col+1); math.Abs(x.Left-end) > 1e-3 {
					t.Errorf("the text after the tab starts at %v, want the stop %v", x.Left, end)
				}
				if math.Abs(p.MaxIntrinsicWidth()-(end+u)) > 1e-3 {
					t.Errorf("max intrinsic width = %v, want %v", p.MaxIntrinsicWidth(), end+u)
				}
			})
		}
	}
}

func TestTabStopsSeveralTabs(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	c := tabCase{spans: spansOf(plain("a\t\tb\tcd\t\te")), tabSize: 4}
	p := c.build(e, 4)
	stops, total := oracle(t, e, c.spans, 4*u)
	tabs := tabRunes(p)
	if len(tabs) != len(stops) {
		t.Fatalf("%d tabs, oracle has %d", len(tabs), len(stops))
	}
	for i, r := range tabs {
		b := boxOf(t, p, r)
		if math.Abs(b.Left-stops[i].x0) > 1e-3 || math.Abs(b.Right-stops[i].x1) > 1e-3 {
			t.Errorf("tab %d is [%v, %v], want [%v, %v]", i, b.Left, b.Right, stops[i].x0, stops[i].x1)
		}
	}
	if math.Abs(p.MaxIntrinsicWidth()-total) > 1e-3 || math.Abs(p.LongestLine()-total) > 1e-3 {
		t.Errorf("widths %v and %v, want %v", p.MaxIntrinsicWidth(), p.LongestLine(), total)
	}
}

func TestTabStopsWithWideCharacters(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	bold := &ui.TextStyleOpts{FontWeight: ui.FontWeightW700}
	big := &ui.TextStyleOpts{FontSize: ui.Ptr(20.0)}
	spaced := &ui.TextStyleOpts{LetterSpacing: ui.Ptr(1.5), WordSpacing: ui.Ptr(2.0)}
	tests := []struct {
		name  string
		spans []tabSpan
	}{
		{"CJK", spansOf(plain("日本語\tx\t語"))},
		{"CJK and Latin", spansOf(plain("ab日\tcd本\tef"))},
		{"emoji", spansOf(plain("😀\tx\t😀😀\ty"))},
		{"emoji sequence", spansOf(plain("👨‍👩‍👧\ta\t"))},
		{"combining marks", spansOf(plain("é́\tab\t"))},
		{"styled spans", spansOf(plain("ab"), tabSpan{"cd\te", bold}, plain("\tfg"))},
		{"a tab in a bigger span", spansOf(plain("abc"), tabSpan{"\t", big}, plain("d\te"))},
		{"letter and word spacing", spansOf(plain("ab"), tabSpan{"c\td \te", spaced}, plain("\tf"))},
		{"a span that is all tabs", spansOf(plain("a"), tabSpan{"\t\t", bold}, plain("b"))},
	}
	for _, tt := range tests {
		for _, tabSize := range []float64{2, 4, 8} {
			t.Run(tt.name+"/tab size "+string(rune('0'+int(tabSize))), func(t *testing.T) {
				c := tabCase{spans: tt.spans, tabSize: tabSize}
				p := c.build(e, tabSize)
				stops, total := oracle(t, e, tt.spans, tabSize*u)
				tabs := tabRunes(p)
				if len(tabs) != len(stops) {
					t.Fatalf("%d tabs, oracle has %d", len(tabs), len(stops))
				}
				for i, r := range tabs {
					b := boxOf(t, p, r)
					if math.Abs(b.Left-stops[i].x0) > 1e-2 || math.Abs(b.Right-stops[i].x1) > 1e-2 {
						t.Errorf("tab %d is [%v, %v], want [%v, %v]", i, b.Left, b.Right, stops[i].x0, stops[i].x1)
					}
				}
				if math.Abs(p.MaxIntrinsicWidth()-total) > 1e-2 {
					t.Errorf("max intrinsic width = %v, want %v", p.MaxIntrinsicWidth(), total)
				}
			})
		}
	}
}

func TestTabStopsUnderWrapping(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	tests := []struct {
		name     string
		text     string
		tabSize  float64
		width    float64 // in glyph advances
		want     []string
		wantEnds []float64 // where each tab ends, in advances from the start of its line
	}{
		// The tab hangs at the end of its line: the word after it does not fit, so it starts a line,
		// and its tab stops start again from the line start.
		{"the tab ends the first line", "aaaa\tbbbb", 4, 6, []string{"aaaa\t", "bbbb"}, []float64{8}},
		{"stops again on the next line", "aaaa\tbb\tcc", 4, 6, []string{"aaaa\t", "bb\tcc"}, []float64{8, 4}},
		{"a tab in the middle of a line", "a\tb\tc", 4, 20, []string{"a\tb\tc"}, []float64{4, 8}},
		// The second tab starts on a stop, so it advances a whole tab width, and hangs.
		{"a tab on a stop", "ab\tcdef\tg", 4, 9, []string{"ab\tcdef\t", "g"}, []float64{4, 12}},
		{"wrapped after a space", "ab \tcd", 4, 5, []string{"ab \t", "cd"}, []float64{4}},
		{"many tabs", "\t\t\tx", 2, 5, []string{"\t\t\t", "x"}, []float64{2, 4, 6}},
		{"a word wider than the line", "ab\tcdefghij\tk", 4, 6, []string{"ab\t", "cdefgh", "ij\tk"}, []float64{4, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tabCase{spans: spansOf(plain(tt.text)), tabSize: tt.tabSize, width: tt.width * u}
			p := c.build(e, tt.tabSize)
			if got := lineTexts(p); strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("lines = %q, want %q", got, tt.want)
			}
			ends := []float64{}
			for _, r := range tabRunes(p) {
				b := boxOf(t, p, r)
				li, _ := p.GetLineNumberAt(p.byteOf(r))
				ends = append(ends, (b.Right-p.lines[li].left)/u)
			}
			if len(ends) != len(tt.wantEnds) {
				t.Fatalf("tab ends = %v, want %v", ends, tt.wantEnds)
			}
			for i := range ends {
				if math.Abs(ends[i]-tt.wantEnds[i]) > 1e-3 {
					t.Errorf("tab ends = %v, want %v", ends, tt.wantEnds)
					break
				}
			}
			// The tab at the end of a line hangs: it adds nothing to the width of its line.
			for li, l := range p.lines {
				lineText := lineTexts(p)[li]
				visible := strings.TrimRight(lineText, " \t")
				if len(visible) != len(lineText) && l.width > l.widthWithSpaces+1e-6 {
					t.Errorf("line %d: width %v is more than %v with its spaces", li, l.width, l.widthWithSpaces)
				}
			}
		})
	}
}

func TestTabAtTheEndOfALineHangs(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	c := tabCase{spans: spansOf(plain("ab\tc")), tabSize: 4, width: 3 * u}
	p := c.build(e, 4)
	if got := lineTexts(p); strings.Join(got, "|") != "ab\t|c" {
		t.Fatalf("lines = %q", got)
	}
	if w := p.lines[0].width; math.Abs(w-2*u) > 1e-3 {
		t.Errorf("the line is %v wide, want the 2 glyphs %v: the tab hangs", w, 2*u)
	}
	if w := p.lines[0].widthWithSpaces; math.Abs(w-4*u) > 1e-3 {
		t.Errorf("the line with its tab is %v wide, want the stop %v", w, 4*u)
	}
	if p.LongestLine() > 3*u {
		t.Errorf("longest line = %v, the tab must not count", p.LongestLine())
	}
}

func TestTabStopsMinIntrinsicWidth(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	// The widest unbreakable word is "abcdef"; the tab is a break opportunity that hangs.
	c := tabCase{spans: spansOf(plain("ab\tabcdef\tc")), tabSize: 4}
	p := c.build(e, 4)
	if got := p.MinIntrinsicWidth(); math.Abs(got-6*u) > 1e-3 {
		t.Errorf("min intrinsic width = %v, want %v", got, 6*u)
	}
}

func TestTabStopsRightToLeft(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	t.Run("a right-to-left paragraph measures from the right edge", func(t *testing.T) {
		c := tabCase{spans: spansOf(plain("abc\tdef\tg")), tabSize: 4, dir: ui.TextDirectionRTL}
		p := c.build(e, 4)
		stops, total := oracle(t, e, c.spans, 4*u)
		if math.Abs(p.Width()-total) > 1e-3 {
			t.Fatalf("width = %v, want %v", p.Width(), total)
		}
		for i, r := range tabRunes(p) {
			b := boxOf(t, p, r)
			// The distance from the start edge, which is on the right.
			near, far := p.Width()-b.Right, p.Width()-b.Left
			if math.Abs(near-stops[i].x0) > 1e-3 || math.Abs(far-stops[i].x1) > 1e-3 {
				t.Errorf("tab %d is %v to %v from the right edge, want %v to %v", i, near, far, stops[i].x0, stops[i].x1)
			}
		}
	})
	t.Run("a tab is not inside a right-to-left run", func(t *testing.T) {
		c := tabCase{spans: spansOf(plain("ab אב\tגד cd\tef")), tabSize: 4}
		p := c.build(e, 4)
		stops, _ := oracle(t, e, c.spans, 4*u)
		for i, r := range tabRunes(p) {
			if p.levels[r] != 0 {
				t.Errorf("tab %d has the level %d, want the paragraph's 0", i, p.levels[r])
			}
			b := boxOf(t, p, r)
			if math.Abs(b.Left-stops[i].x0) > 1e-3 || math.Abs(b.Right-stops[i].x1) > 1e-3 {
				t.Errorf("tab %d is [%v, %v], want [%v, %v]", i, b.Left, b.Right, stops[i].x0, stops[i].x1)
			}
		}
	})
	t.Run("a right-to-left paragraph with a left-to-right run", func(t *testing.T) {
		c := tabCase{spans: spansOf(plain("אב\tab cd\tג")), tabSize: 4, dir: ui.TextDirectionRTL}
		p := c.build(e, 4)
		stops, total := oracle(t, e, c.spans, 4*u)
		for i, r := range tabRunes(p) {
			if p.levels[r] != 1 {
				t.Errorf("tab %d has the level %d, want the paragraph's 1", i, p.levels[r])
			}
			b := boxOf(t, p, r)
			near, far := p.Width()-b.Right, p.Width()-b.Left
			if math.Abs(near-stops[i].x0) > 1e-3 || math.Abs(far-stops[i].x1) > 1e-3 {
				t.Errorf("tab %d is %v to %v from the right edge, want %v to %v", i, near, far, stops[i].x0, stops[i].x1)
			}
		}
		if math.Abs(p.Width()-total) > 1e-3 {
			t.Errorf("width = %v, want %v", p.Width(), total)
		}
	})
}

func TestTabStopsQueries(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	c := tabCase{spans: spansOf(plain("ab\tcd\t")), tabSize: 4}
	p := c.build(e, 4)
	tab := boxOf(t, p, 2)
	if math.Abs(tab.Left-2*u) > 1e-3 || math.Abs(tab.Right-4*u) > 1e-3 {
		t.Fatalf("tab box = [%v, %v], want [%v, %v]", tab.Left, tab.Right, 2*u, 4*u)
	}
	y := (tab.Top + tab.Bottom) / 2

	t.Run("GetBoxesForRange of a range with tabs is one box", func(t *testing.T) {
		bs := p.GetBoxesForRange(0, len(p.runes), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
		if len(bs) != 1 || bs[0].Left != 0 || math.Abs(bs[0].Right-8*u) > 1e-3 {
			t.Errorf("boxes = %v, want one box from 0 to %v", bs, 8*u)
		}
	})
	t.Run("GetPositionForOffset", func(t *testing.T) {
		tests := []struct {
			name string
			x    float64
			want int
		}{
			{"before the tab", 2*u + 0.1, 2},
			{"in the first half of the tab", 2.5 * u, 2},
			{"in the second half of the tab", 3.5 * u, 3},
			{"at the stop", 4 * u, 3},
			{"after the last tab", 9 * u, 6},
			{"at the start", -5, 0},
		}
		for _, tt := range tests {
			if got := p.GetPositionForOffset(ui.Offset{Dx: tt.x, Dy: y}); got.Offset != tt.want {
				t.Errorf("%s: position at x=%v is %d, want %d", tt.name, tt.x, got.Offset, tt.want)
			}
		}
	})
	t.Run("glyph info", func(t *testing.T) {
		gi, ok := p.GetGlyphInfoAt(2)
		if !ok || gi.GraphemeClusterRange != rng(2, 3) || math.Abs(gi.GraphemeClusterLayoutBounds.Left-2*u) > 1e-3 || math.Abs(gi.GraphemeClusterLayoutBounds.Right-4*u) > 1e-3 {
			t.Errorf("glyph info at the tab = %+v", gi)
		}
		gi, ok = p.GetClosestGlyphInfoForOffset(ui.Offset{Dx: 3 * u, Dy: y})
		if !ok || gi.GraphemeClusterRange != rng(2, 3) || math.Abs(gi.GraphemeClusterLayoutBounds.Right-4*u) > 1e-3 {
			t.Errorf("closest glyph info in the tab = %+v", gi)
		}
		gi, ok = p.GetGlyphInfoAt(5)
		if !ok || math.Abs(gi.GraphemeClusterLayoutBounds.Left-6*u) > 1e-3 || math.Abs(gi.GraphemeClusterLayoutBounds.Right-8*u) > 1e-3 {
			t.Errorf("glyph info at the last tab = %+v", gi)
		}
	})
	t.Run("the caret after a tab is at the stop", func(t *testing.T) {
		// A caret is the left edge of the box of the next rune, or the right edge of the box of
		// the rune before it at the end of the text.
		after := boxOf(t, p, 3)
		if math.Abs(after.Left-4*u) > 1e-3 {
			t.Errorf("the rune after the tab starts at %v, want the stop %v", after.Left, 4*u)
		}
		last := boxOf(t, p, 5)
		if math.Abs(last.Right-8*u) > 1e-3 {
			t.Errorf("the caret at the end of the text is at %v, want the stop %v", last.Right, 8*u)
		}
	})
	t.Run("line metrics see the width without the hanging tab", func(t *testing.T) {
		lm := p.ComputeLineMetrics()
		if len(lm) != 1 || math.Abs(lm[0].Width-6*u) > 1e-3 {
			t.Errorf("line metrics = %+v, want the width %v (the last tab hangs)", lm, 6*u)
		}
	})
}

func TestTabStopsDrawNothing(t *testing.T) {
	e := codeEngine(t)
	for _, tabSize := range []float64{4, 0} {
		c := tabCase{spans: spansOf(plain("a\tb")), tabSize: tabSize}
		p := c.build(e, tabSize)
		var glyphs int
		p.PaintParagraph(func(d ui.ParagraphDraw) {
			if _, ok := d.Op.(ui.OpDrawPath); ok {
				glyphs++
			}
		})
		// a and b always; the tab's glyph is drawn only when tabs have no stops (as before).
		if tabSize > 0 && glyphs != 2 {
			t.Errorf("tab size %v: %d glyphs drawn, want 2", tabSize, glyphs)
		}
	}
}

func TestTabStopsOff(t *testing.T) {
	e := codeEngine(t)
	c := tabCase{spans: spansOf(plain("a\tb\tc"))}
	p := c.build(e, 0)
	if p.tabW != 0 {
		t.Fatalf("tabW = %v without a tab size", p.tabW)
	}
	// A tab is a glyph of the font, with its own advance.
	b := boxOf(t, p, 1)
	var adv float64
	for _, r := range p.runs {
		for _, g := range r.glyphs {
			if g.cluster == 1 {
				adv = g.advance
			}
		}
	}
	if math.Abs((b.Right-b.Left)-adv) > 1e-6 {
		t.Errorf("the tab box is %v wide, the glyph advances %v", b.Right-b.Left, adv)
	}
	// A paragraph without tabs is the same with a tab size.
	q := tabCase{spans: spansOf(plain("abc")), tabSize: 4}.build(e, 4)
	r := tabCase{spans: spansOf(plain("abc"))}.build(e, 0)
	if q.MaxIntrinsicWidth() != r.MaxIntrinsicWidth() || q.tabW != 0 {
		t.Errorf("a tab size changed a paragraph with no tab: %v and %v", q.MaxIntrinsicWidth(), r.MaxIntrinsicWidth())
	}
}

func TestTabStopsEllipsisAndJustify(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	t.Run("ellipsis", func(t *testing.T) {
		one := 1
		c := tabCase{spans: spansOf(plain("ab\tcd\tef\tgh\tij")), tabSize: 4, width: 11 * u, maxLines: &one, ellipsis: "…"}
		p := c.build(e, 4)
		if !p.DidExceedMaxLines() || len(p.lines) != 1 {
			t.Fatalf("lines = %d, exceeded = %v", len(p.lines), p.DidExceedMaxLines())
		}
		if p.lines[0].width > 11*u+1e-3 {
			t.Errorf("the line is %v wide, more than %v", p.lines[0].width, 11*u)
		}
		for _, r := range tabRunes(p) {
			if r >= p.lines[0].end {
				break
			}
			b := boxOf(t, p, r)
			if k := b.Right / (4 * u); math.Abs(k-math.Round(k)) > 1e-3 {
				t.Errorf("a tab ends at %v, not on a stop of %v", b.Right, 4*u)
			}
		}
	})
	t.Run("justify widens the tab like a space", func(t *testing.T) {
		c := tabCase{spans: spansOf(plain("ab\tcd ef gh ij kl mn op qr")), tabSize: 4, width: 20 * u, align: ui.TextAlignJustify}
		p := c.build(e, 4)
		if len(p.lines) < 2 {
			t.Fatalf("lines = %q", lineTexts(p))
		}
		first := p.lines[0]
		if first.justify <= 0 || math.Abs(first.width-20*u) > 1e-3 {
			t.Errorf("the first line is %v wide with %v justification, want %v", first.width, first.justify, 20*u)
		}
		b := boxOf(t, p, 2)
		if math.Abs((b.Right-b.Left)-(2*u+first.justify)) > 1e-3 {
			t.Errorf("the tab is %v wide, want %v (2 advances and the justification)", b.Right-b.Left, 2*u+first.justify)
		}
	})
}

func TestTabSizeInvalid(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks")
	}
	for _, size := range []float64{-1, math.NaN(), math.Inf(1)} {
		func() {
			defer func() {
				v := recover()
				err, _ := v.(error)
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute("TabSizeInvalid") {
					t.Errorf("tab size %v: recovered %v, want the tag TabSizeInvalid", size, v)
				}
			}()
			ui.NewParagraphStyle(ui.ParagraphStyleOpts{TabSize: size})
		}()
	}
}

// TestTabStopsAreConsistent lays out random text with tabs and checks what must hold for any
// text: every tab ends on a stop measured from the left of its line (letters are the same size
// in a monospace font, so the stops are whole advances), the boxes of a line do not overlap and
// the lines are never wider than the width unless a single glyph is.
func TestTabStopsAreConsistent(t *testing.T) {
	e := codeEngine(t)
	u := spaceUnit(t, e)
	rng := rand.New(rand.NewSource(7))
	alphabet := []string{"a", "b", "c", " ", "\t", "\t", "\n", "ab", "é", "日", "😀"}
	for iter := 0; iter < 300; iter++ {
		var sb strings.Builder
		for n := rng.Intn(24); n >= 0; n-- {
			sb.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		tabSize := []float64{1, 2, 3, 4, 8}[rng.Intn(5)]
		width := float64(3+rng.Intn(30)) * u
		c := tabCase{spans: spansOf(plain(sb.String())), tabSize: tabSize, width: width}
		p := c.build(e, tabSize)
		for _, r := range tabRunes(p) {
			li, ok := p.GetLineNumberAt(p.byteOf(r))
			if !ok {
				continue
			}
			l := p.lines[li]
			bs := p.GetBoxesForRange(p.byteOf(r), p.byteOf(r+1), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
			if len(bs) != 1 {
				t.Fatalf("%q: tab %d has %d boxes", sb.String(), r, len(bs))
			}
			b := bs[0]
			if b.Right-b.Left <= 0 {
				t.Fatalf("%q (tab size %v): the tab at %d has the width %v", sb.String(), tabSize, r, b.Right-b.Left)
			}
			if k := (b.Right - l.left) / (tabSize * u); math.Abs(k-math.Round(k)) > 1e-3 {
				t.Fatalf("%q (tab size %v, width %v): the tab at %d ends at %v: not on a stop of %v (line %q)", sb.String(), tabSize, width/u, r, b.Right-l.left, tabSize*u, lineTexts(p)[li])
			}
		}
		// The boxes of the whole text, line by line, do not overlap.
		last := math.Inf(-1)
		lastTop := math.Inf(-1)
		for _, b := range p.GetBoxesForRange(0, len(p.content.Text), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight) {
			if b.Top != lastTop {
				last, lastTop = math.Inf(-1), b.Top
			}
			if b.Left < last-1e-3 {
				t.Fatalf("%q: boxes overlap at %v < %v", sb.String(), b.Left, last)
			}
			last = b.Right
		}
	}
}
