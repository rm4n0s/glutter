package layout

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

const (
	family = "👨‍👩‍👧"
	// "e" followed by a combining acute accent: two runes, one grapheme cluster.
	accented = "é"
)

// paragraphIn builds and lays out a paragraph from one text in a style.
func paragraphIn(t *testing.T, e *Engine, text string, style *ui.TextStyle, ps *ui.ParagraphStyle, width float64) *paragraph {
	t.Helper()
	p := build(e, text, style, ps).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(width))
	return p
}

// box makes a left-to-right text box.
func box(l, t, r, b float64) ui.TextBox {
	return ui.NewTextBoxFromLTRBD(l, t, r, b, ui.TextDirectionLTR)
}

// rng makes a text range.
func rng(start, end int) ui.TextRange { return ui.NewTextRange(start, end) }

func TestLetterAndWordSpacing(t *testing.T) {
	e := bundledEngine(t)
	plain := laidOut(t, e, "ab cd ef", 16, 1000, nil)
	tests := []struct {
		name         string
		letter, word float64
		extra        float64 // total extra width: 8 clusters, 2 spaces
	}{
		{"letter spacing", 2, 0, 8 * 2},
		{"word spacing", 0, 5, 2 * 5},
		{"both", 1.5, 4, 8*1.5 + 2*4},
		{"negative", -0.5, 0, 8 * -0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), LetterSpacing: ui.Ptr(tt.letter), WordSpacing: ui.Ptr(tt.word)})
			p := paragraphIn(t, e, "ab cd ef", style, nil, 1000)
			if got, want := p.MaxIntrinsicWidth(), plain.MaxIntrinsicWidth()+tt.extra; !near(got, want) {
				t.Errorf("width = %v, want %v", got, want)
			}
		})
	}
	t.Run("spacing changes the line breaks", func(t *testing.T) {
		wide := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), WordSpacing: ui.Ptr(60.0)})
		w := plain.MaxIntrinsicWidth()
		if n := len(paragraphIn(t, e, "ab cd ef", roboto(16), nil, w).lines); n != 1 {
			t.Fatalf("plain text takes %d lines", n)
		}
		if n := len(paragraphIn(t, e, "ab cd ef", wide, nil, w).lines); n != 3 {
			t.Errorf("with word spacing it takes %d lines, want 3", n)
		}
	})
	t.Run("in a right-to-left run the room is on the left of the glyph", func(t *testing.T) {
		e := systemEngine(t, hebrew)
		style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), LetterSpacing: ui.Ptr(10.0)})
		p := paragraphIn(t, e, "של", style, nil, 1000)
		r := p.lines[0].runs[0]
		if len(r.glyphs) != 2 {
			t.Fatalf("%d glyphs", len(r.glyphs))
		}
		for i, g := range r.glyphs {
			if !near(g.x-g.pen, 10) {
				t.Errorf("glyph %d is %v from the start of its advance, want 10", i, g.x-g.pen)
			}
		}
	})
}

func TestFontFeaturesAndVariations(t *testing.T) {
	e := bundledEngine(t)
	opts := func(o ui.TextStyleOpts) *ui.TextStyle {
		o.FontFamily, o.FontSize = fonts.RobotoFamily, ui.Ptr(24.0)
		return ui.NewTextStyle(o)
	}
	width := func(text string, o ui.TextStyleOpts) float64 {
		return paragraphIn(t, e, text, opts(o), nil, 10000).MaxIntrinsicWidth()
	}
	glyphs := func(text string, o ui.TextStyleOpts) int {
		return len(paragraphIn(t, e, text, opts(o), nil, 10000).runs[0].glyphs)
	}
	if on, off := glyphs("office", ui.TextStyleOpts{}), glyphs("office", ui.TextStyleOpts{FontFeatures: []ui.FontFeature{ui.NewFontFeature("liga", 0)}}); off <= on {
		t.Errorf("turning ligatures off: %d glyphs, with them %d", off, on)
	}
	if on, off := glyphs("office", ui.TextStyleOpts{FontFeatures: []ui.FontFeature{ui.NewFontFeature("liga", 1)}}), glyphs("office", ui.TextStyleOpts{FontFeatures: []ui.FontFeature{ui.NewFontFeature("liga", 0)}}); on >= off {
		t.Errorf("liga 1: %d glyphs, liga 0: %d", on, off)
	}
	regular := width("Hamburgefonstiv", ui.TextStyleOpts{})
	if bold := width("Hamburgefonstiv", ui.TextStyleOpts{FontWeight: ui.FontWeightBold}); bold <= regular {
		t.Errorf("bold %v is not wider than regular %v", bold, regular)
	}
	if black := width("Hamburgefonstiv", ui.TextStyleOpts{FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 900)}}); black <= regular {
		t.Errorf("wght 900 %v is not wider than regular %v", black, regular)
	}
	if narrow := width("Hamburgefonstiv", ui.TextStyleOpts{FontVariations: []ui.FontVariation{ui.NewFontVariation("wdth", 75)}}); narrow >= regular {
		t.Errorf("wdth 75 %v is not narrower than regular %v", narrow, regular)
	}
	// A variation overrides the one that the weight gives.
	if v := width("Hamburgefonstiv", ui.TextStyleOpts{FontWeight: ui.FontWeightBold, FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 400)}}); !near(v, regular) {
		t.Errorf("wght 400 over bold: %v, regular %v", v, regular)
	}
	if italic := width("Hamburgefonstiv", ui.TextStyleOpts{FontStyle: ui.FontStyleItalic}); italic == regular {
		t.Errorf("italic has the width of regular: %v", italic)
	}
}

func TestBoxesForRange(t *testing.T) {
	e := bundledEngine(t)
	const size = 16.0
	p := laidOut(t, e, "hello world", size, 1000, nil)
	asc, desc := robotoAscent*size, robotoDescent*size
	hello := p.sum(0, 5)
	tests := []struct {
		name       string
		start, end int
		want       []ui.TextBox
	}{
		{"first word", 0, 5, []ui.TextBox{box(0, 0, hello, asc+desc)}},
		{"all of it", 0, 11, []ui.TextBox{box(0, 0, p.sum(0, 11), asc+desc)}},
		{"middle", 6, 8, []ui.TextBox{box(p.sum(0, 6), 0, p.sum(0, 8), asc+desc)}},
		{"empty range", 3, 3, nil},
		{"backwards range", 5, 3, nil},
		{"past the end is cut", 8, 100, []ui.TextBox{box(p.sum(0, 8), 0, p.sum(0, 11), asc+desc)}},
		{"negative start", -3, 2, nil},
		{"start past the end", 30, 40, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetBoxesForRange(tt.start, tt.end, ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
			if len(got) != len(tt.want) {
				t.Fatalf("boxes = %v, want %v", got, tt.want)
			}
			for i := range got {
				w := tt.want[i]
				if !near(got[i].Left, w.Left) || !near(got[i].Right, w.Right) || !near(got[i].Top, w.Top) || !near(got[i].Bottom, w.Bottom) || got[i].Direction != w.Direction {
					t.Errorf("box %d = %v, want %v", i, got[i], w)
				}
			}
		})
	}
}

func TestBoxHeightStyles(t *testing.T) {
	e := bundledEngine(t)
	const size = 16.0
	// Two lines of 32 px with the glyphs 18.75 px high: 13.25 px of space between them.
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(size), Height: ui.Ptr(2.0), LeadingDistribution: ui.TextLeadingDistributionEven})
	p := paragraphIn(t, e, "one\ntwo\nthree", style, nil, 1000)
	if len(p.lines) != 3 || p.Height() != 96 {
		t.Fatalf("lines %v, height %v", lineTexts(p), p.Height())
	}
	asc, desc := robotoAscent*size, robotoDescent*size
	gap := 32 - (asc + desc)
	baseline := func(line int) float64 { return p.lines[line].baseline }
	tests := []struct {
		name        string
		style       ui.BoxHeightStyle
		line        int
		top, bottom float64
	}{
		{"tight, first line", ui.BoxHeightStyleTight, 0, baseline(0) - asc, baseline(0) + desc},
		{"tight, second line", ui.BoxHeightStyleTight, 1, baseline(1) - asc, baseline(1) + desc},
		{"max is the whole line", ui.BoxHeightStyleMax, 1, 32, 64},
		{"top: spacing above, none above the first line", ui.BoxHeightStyleIncludeLineSpacingTop, 0, baseline(0) - asc, baseline(0) + desc},
		{"top: second line reaches the line above", ui.BoxHeightStyleIncludeLineSpacingTop, 1, baseline(1) - asc - gap, baseline(1) + desc},
		{"bottom: first line reaches the line below", ui.BoxHeightStyleIncludeLineSpacingBottom, 0, baseline(0) - asc, baseline(0) + desc + gap},
		{"bottom: none below the last line", ui.BoxHeightStyleIncludeLineSpacingBottom, 2, baseline(2) - asc, baseline(2) + desc},
		{"middle: half each way", ui.BoxHeightStyleIncludeLineSpacingMiddle, 1, baseline(1) - asc - gap/2, baseline(1) + desc + gap/2},
		{"middle: first line only below", ui.BoxHeightStyleIncludeLineSpacingMiddle, 0, baseline(0) - asc, baseline(0) + desc + gap/2},
		{"no strut: the strut style is the tight one", ui.BoxHeightStyleStrut, 1, baseline(1) - asc, baseline(1) + desc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := p.lines[tt.line]
			bs := p.GetBoxesForRange(p.byteOf(l.start), p.byteOf(l.start+1), tt.style, ui.BoxWidthStyleTight)
			if len(bs) != 1 || !near(bs[0].Top, tt.top) || !near(bs[0].Bottom, tt.bottom) {
				t.Errorf("boxes = %v, want top %v bottom %v", bs, tt.top, tt.bottom)
			}
		})
	}
	// Middle boxes of neighbouring lines meet.
	var edges [][2]float64
	for i := range p.lines {
		l := p.lines[i]
		b := p.GetBoxesForRange(p.byteOf(l.start), p.byteOf(l.start+1), ui.BoxHeightStyleIncludeLineSpacingMiddle, ui.BoxWidthStyleTight)[0]
		edges = append(edges, [2]float64{b.Top, b.Bottom})
	}
	for i := 1; i < len(edges); i++ {
		if !near(edges[i][0], edges[i-1][1]) {
			t.Errorf("line %d starts at %v, the line above ends at %v", i, edges[i][0], edges[i-1][1])
		}
	}
	t.Run("strut", func(t *testing.T) {
		ps := paraStyle(ui.ParagraphStyleOpts{StrutStyle: ui.NewStrutStyle(ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(10.0), Height: ui.Ptr(3.0), ForceStrutHeight: ui.Ptr(true)})})
		sp := paragraphIn(t, e, "a\nb", roboto(16), ps, 1000)
		bs := sp.GetBoxesForRange(0, 1, ui.BoxHeightStyleStrut, ui.BoxWidthStyleTight)
		if len(bs) != 1 || !near(bs[0].Bottom-bs[0].Top, 30) {
			t.Errorf("strut box = %v, want 30 high", bs)
		}
	})
}

func TestBoxWidthStyleMax(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "a long first line\nshort", 16, 1000, nil)
	long := p.lines[0].width
	// The tight box of "short" is as wide as the text; the max box reaches the widest line.
	tight := p.GetBoxesForRange(p.byteOf(p.lines[1].start), len(p.content.Text), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
	max := p.GetBoxesForRange(p.byteOf(p.lines[1].start), len(p.content.Text), ui.BoxHeightStyleTight, ui.BoxWidthStyleMax)
	if len(tight) != 1 || len(max) != 2 {
		t.Fatalf("tight %v, max %v", tight, max)
	}
	if !near(max[0].Left, tight[0].Left) || !near(max[1].Right, long) || !near(max[1].Left, tight[0].Right) {
		t.Errorf("max boxes %v do not extend %v to the widest line %v", max, tight, long)
	}
	// The line that is the widest gets no extra box.
	if bs := p.GetBoxesForRange(0, 6, ui.BoxHeightStyleTight, ui.BoxWidthStyleMax); len(bs) != 2 {
		t.Errorf("boxes of the widest line = %v", bs)
	}
}

func TestBoxesOverSeveralLines(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, pangram, 16, 150, nil)
	bs := p.GetBoxesForRange(0, len(pangram), ui.BoxHeightStyleTight, ui.BoxWidthStyleTight)
	if len(bs) != len(p.lines) {
		t.Fatalf("%d boxes for %d lines", len(bs), len(p.lines))
	}
	for i, b := range bs {
		l := p.lines[i]
		// A box covers the line's text, including the space that hangs at its end.
		if !near(b.Left, l.left) || !near(b.Right, l.left+l.widthWithSpaces) {
			t.Errorf("box %d = %v, line is [%v,%v]", i, b, l.left, l.left+l.widthWithSpaces)
		}
		if i > 0 && b.Top <= bs[i-1].Top {
			t.Errorf("box %d is not below box %d", i, i-1)
		}
	}
}

func TestBoxesInsideLigaturesAndGraphemes(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "office", 24, 1000, nil)
	if len(p.runs[0].glyphs) >= 6 {
		t.Skip("no ligature in the font")
	}
	whole := p.GetBoxesForRange(0, 6, 0, 0)[0]
	// Every character of the text has a box and together they are the whole.
	var sum float64
	var prevRight float64
	for i := 0; i < 6; i++ {
		b := p.GetBoxesForRange(i, i+1, 0, 0)
		if len(b) != 1 {
			t.Fatalf("character %d has boxes %v", i, b)
		}
		if i > 0 && !near(b[0].Left, prevRight) {
			t.Errorf("character %d starts at %v, the one before ends at %v", i, b[0].Left, prevRight)
		}
		prevRight = b[0].Right
		sum += b[0].Right - b[0].Left
	}
	if !near(sum, whole.Right-whole.Left) {
		t.Errorf("the boxes of the characters add up to %v, the whole is %v", sum, whole.Right-whole.Left)
	}
	// The three characters of the "ffi" ligature share its width.
	ff, f2, i3 := p.GetBoxesForRange(1, 2, 0, 0)[0], p.GetBoxesForRange(2, 3, 0, 0)[0], p.GetBoxesForRange(3, 4, 0, 0)[0]
	if !near(ff.Right-ff.Left, f2.Right-f2.Left) || !near(f2.Right-f2.Left, i3.Right-i3.Left) {
		t.Errorf("the characters of a ligature differ in width: %v %v %v", ff, f2, i3)
	}

	t.Run("grapheme cluster", func(t *testing.T) {
		g := laidOut(t, e, "a"+accented+"b", 24, 1000, nil)
		if len(g.runes) != 4 {
			t.Fatalf("%d runes", len(g.runes))
		}
		// The accent has no width of its own: it is in the box of the "e".
		e1 := g.GetBoxesForRange(1, 3, 0, 0)
		all := g.GetBoxesForRange(1, 4, 0, 0)
		if len(e1) != 1 || len(all) != 1 || e1[0].Right > all[0].Right {
			t.Errorf("e with accent: %v, with b: %v", e1, all)
		}
	})
}

func TestBoxesAcrossABidiBoundary(t *testing.T) {
	text := "abc " + hebrew + " def"
	e := systemEngine(t, text)
	p := build(e, text, roboto(16), nil).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	n := len([]rune(text))
	// Byte offsets: the Hebrew letters are two bytes each.
	hebStart, hebEnd := 4, 4+len(hebrew)
	t.Run("the whole text is three boxes", func(t *testing.T) {
		bs := p.GetBoxesForRange(0, len(text), 0, 0)
		if len(bs) != 3 {
			t.Fatalf("boxes = %v", bs)
		}
		dirs := []ui.TextDirection{bs[0].Direction, bs[1].Direction, bs[2].Direction}
		if !slices.Equal(dirs, []ui.TextDirection{ui.TextDirectionLTR, ui.TextDirectionRTL, ui.TextDirectionLTR}) {
			t.Errorf("directions = %v", dirs)
		}
		for i := 1; i < 3; i++ {
			if !near(bs[i].Left, bs[i-1].Right) {
				t.Errorf("box %d starts at %v, box %d ends at %v", i, bs[i].Left, i-1, bs[i-1].Right)
			}
		}
	})
	t.Run("a range inside the hebrew is one right-to-left box", func(t *testing.T) {
		bs := p.GetBoxesForRange(hebStart, hebEnd, 0, 0)
		if len(bs) != 1 || bs[0].Direction != ui.TextDirectionRTL {
			t.Fatalf("boxes = %v", bs)
		}
		// Start is the right edge for right-to-left text.
		if bs[0].Start() != bs[0].Right || bs[0].End() != bs[0].Left {
			t.Errorf("start %v end %v of %v", bs[0].Start(), bs[0].End(), bs[0])
		}
	})
	t.Run("a range from english into hebrew is two boxes", func(t *testing.T) {
		bs := p.GetBoxesForRange(2, hebStart+4, 0, 0)
		if len(bs) != 2 || bs[0].Direction != ui.TextDirectionLTR || bs[1].Direction != ui.TextDirectionRTL {
			t.Fatalf("boxes = %v", bs)
		}
	})
	t.Run("the first hebrew letter is on the right of the hebrew", func(t *testing.T) {
		first := p.GetBoxesForRange(hebStart, hebStart+2, 0, 0)[0]
		last := p.GetBoxesForRange(hebEnd-2, hebEnd, 0, 0)[0]
		if first.Left <= last.Left {
			t.Errorf("first letter %v should be right of the last letter %v", first, last)
		}
	})
	t.Run("positions follow the boxes", func(t *testing.T) {
		_ = n
		first := p.GetBoxesForRange(hebStart, hebStart+2, 0, 0)[0]
		// The right edge of the first Hebrew letter is where the text of that letter starts;
		// a point just inside it is closer to its start.
		pos := p.GetPositionForOffset(ui.Offset{Dx: first.Right - 0.5, Dy: 5})
		if pos.Offset != hebStart {
			t.Errorf("position at the right edge of the first letter = %v, want offset %d", pos, hebStart)
		}
		pos = p.GetPositionForOffset(ui.Offset{Dx: first.Left + 0.5, Dy: 5})
		if pos.Offset != hebStart+2 {
			t.Errorf("position at the left edge of the first letter = %v, want offset %d", pos, hebStart+2)
		}
	})
}

func TestPositionForOffset(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "abc def ghi", 16, 1000, nil)
	// x of the left edge of each character.
	edge := func(i int) float64 { return p.sum(0, i) }
	tests := []struct {
		name string
		at   ui.Offset
		want int
	}{
		{"far left", ui.Offset{Dx: -50, Dy: 5}, 0},
		{"left edge", ui.Offset{Dx: 0, Dy: 5}, 0},
		{"just right of the left edge", ui.Offset{Dx: edge(1) * 0.25, Dy: 5}, 0},
		{"past the middle of the first character", ui.Offset{Dx: edge(1) * 0.75, Dy: 5}, 1},
		{"on the boundary of two characters", ui.Offset{Dx: edge(4), Dy: 5}, 4},
		{"before the middle of d", ui.Offset{Dx: edge(4) + (edge(5)-edge(4))*0.4, Dy: 5}, 4},
		{"after the middle of d", ui.Offset{Dx: edge(4) + (edge(5)-edge(4))*0.6, Dy: 5}, 5},
		{"far right", ui.Offset{Dx: 5000, Dy: 5}, 11},
		{"above the text", ui.Offset{Dx: edge(2), Dy: -40}, 2},
		{"below the text", ui.Offset{Dx: edge(2), Dy: 400}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetPositionForOffset(tt.at)
			if got.Offset != tt.want || got.Affinity.IsUpstream() {
				t.Errorf("position = %v, want offset %d downstream", got, tt.want)
			}
		})
	}
	t.Run("empty paragraph", func(t *testing.T) {
		empty := laidOut(t, e, "", 16, 100, nil)
		if got := empty.GetPositionForOffset(ui.Offset{Dx: 50, Dy: 50}); !got.Equal(ui.NewTextPosition(0)) {
			t.Errorf("position = %v", got)
		}
	})
}

func TestPositionForOffsetOnSeveralLines(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "one two\nthree four five", 16, 80, nil)
	lines := lineTexts(p)
	if len(lines) < 3 {
		t.Fatalf("lines = %q", lines)
	}
	mid := func(i int) float64 { return p.lines[i].top + p.lines[i].height/2 }
	t.Run("each line", func(t *testing.T) {
		for i, l := range p.lines {
			got := p.GetPositionForOffset(ui.Offset{Dx: -1, Dy: mid(i)})
			if got.Offset != p.byteOf(l.start) {
				t.Errorf("line %d: start position = %v, want %d", i, got, p.byteOf(l.start))
			}
		}
	})
	t.Run("the end of a line that was broken softly is upstream", func(t *testing.T) {
		li := 1 // "three four " breaks softly
		l := p.lines[li]
		if l.hardBreak {
			t.Skip("not a soft break with these fonts")
		}
		got := p.GetPositionForOffset(ui.Offset{Dx: 5000, Dy: mid(li)})
		if got.Offset != p.byteOf(l.end) || !got.Affinity.IsUpstream() {
			t.Errorf("position = %v, want offset %d upstream", got, p.byteOf(l.end))
		}
		// The start of the next line is the same offset, downstream.
		next := p.GetPositionForOffset(ui.Offset{Dx: -1, Dy: mid(li + 1)})
		if next.Offset != p.byteOf(l.end) || next.Affinity.IsUpstream() {
			t.Errorf("next line start = %v", next)
		}
	})
	t.Run("the end of a line with a newline is before the newline", func(t *testing.T) {
		got := p.GetPositionForOffset(ui.Offset{Dx: 5000, Dy: mid(0)})
		if got.Offset != 7 || got.Affinity.IsUpstream() {
			t.Errorf("position = %v, want offset 7 (before the newline)", got)
		}
	})
}

func TestPositionsNeverSplitAGraphemeCluster(t *testing.T) {
	e := systemEngine(t, family)
	text := "a" + family + accented + "b"
	p := build(e, text, roboto(24), nil).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	valid := map[int]bool{}
	for _, r := range p.graphemes {
		valid[p.byteOf(r)] = true
	}
	if len(p.graphemes) != 5 { // a, family, e+accent, b, and the end
		t.Fatalf("graphemes = %v", p.graphemes)
	}
	w := p.sum(0, len(p.runes))
	for x := -5.0; x < w+5; x += 0.5 {
		pos := p.GetPositionForOffset(ui.Offset{Dx: x, Dy: 5})
		if !valid[pos.Offset] {
			t.Fatalf("x=%v: offset %d is inside a grapheme cluster", x, pos.Offset)
		}
	}
	info, ok := p.GetGlyphInfoAt(1 + 4) // inside the family emoji
	if !ok {
		t.Fatal("no glyph info inside the emoji")
	}
	if info.GraphemeClusterRange != (ui.TextRange{Start: 1, End: 1 + len(family)}) {
		t.Errorf("range = %v, want the whole family emoji [1,%d)", info.GraphemeClusterRange, 1+len(family))
	}
	if info.WritingDirection != ui.TextDirectionLTR || info.GraphemeClusterLayoutBounds.Width() <= 0 {
		t.Errorf("info = %+v", info)
	}
}

func TestGlyphInfo(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "ab\ncd", 16, 1000, nil)
	b := p.GetBoxesForRange(1, 2, 0, 0)[0]
	info, ok := p.GetGlyphInfoAt(1)
	if !ok || info.GraphemeClusterRange != (ui.TextRange{Start: 1, End: 2}) {
		t.Fatalf("info = %+v, %v", info, ok)
	}
	r := info.GraphemeClusterLayoutBounds
	if !near(r.Left, b.Left) || !near(r.Right, b.Right) || !near(r.Top, 0) || !near(r.Bottom, 19) {
		t.Errorf("bounds = %v, box = %v", r, b)
	}
	if info2, ok := p.GetGlyphInfoAt(3); !ok || info2.GraphemeClusterLayoutBounds.Top != 19 {
		t.Errorf("second line: %+v, %v", info2, ok)
	}
	for _, off := range []int{-1, 5, 99} {
		if _, ok := p.GetGlyphInfoAt(off); ok {
			t.Errorf("GetGlyphInfoAt(%d) found a glyph", off)
		}
	}
	closest, ok := p.GetClosestGlyphInfoForOffset(ui.Offset{Dx: b.Left + 1, Dy: 25})
	if !ok || closest.GraphemeClusterRange != (ui.TextRange{Start: 3, End: 4}) && closest.GraphemeClusterRange != (ui.TextRange{Start: 4, End: 5}) {
		t.Errorf("closest = %+v, %v", closest, ok)
	}
	if _, ok := laidOut(t, e, "", 16, 100, nil).GetClosestGlyphInfoForOffset(ui.Offset{}); ok {
		t.Error("the empty paragraph has a glyph")
	}
	// Glyphs cut away by maxLines are not found.
	cut := laidOut(t, e, pangram, 16, 100, paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…"}))
	if _, ok := cut.GetGlyphInfoAt(len(pangram) - 2); ok {
		t.Error("a glyph that is cut away was found")
	}
	if _, ok := cut.GetGlyphInfoAt(0); !ok {
		t.Error("a visible glyph was not found")
	}
}

func TestWordBoundary(t *testing.T) {
	e := bundledEngine(t)
	text := "Hello word, it's 3.5 o'clock! 日本"
	p := laidOut(t, e, text, 16, 1000, nil)
	up, down := ui.TextAffinityUpstream, ui.TextAffinityDownstream
	tests := []struct {
		name     string
		off      int
		affinity ui.TextAffinity
		want     ui.TextRange
	}{
		{"start of a word", 0, down, rng(0, 5)},
		{"inside a word", 3, down, rng(0, 5)},
		{"end of the word, downstream points at the space", 5, down, rng(5, 6)},
		{"end of the word, upstream points at the o", 5, up, rng(0, 5)},
		{"a space is its own range", 5, down, rng(5, 6)},
		{"the second word", 7, down, rng(6, 10)},
		{"punctuation is its own range", 10, down, rng(10, 11)},
		{"an apostrophe stays in the word", 14, down, rng(12, 16)},
		{"a decimal number is one word", 18, down, rng(17, 20)},
		{"upstream at the very start", 0, up, rng(0, 0)},
		{"past the end", 500, down, rng(len(text), len(text))},
		{"end of the text, upstream, is the last word", len(text), up, rng(len(text)-3, len(text))},
		{"exclamation mark", 28, down, rng(28, 29)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetWordBoundary(ui.TextPosition{Offset: tt.off, Affinity: tt.affinity})
			if tt.name == "end of the text, upstream, is the last word" {
				// The last "word" is the two ideographs, each a word of its own by UAX #29 segmentation of
				// this engine; accept a range that ends the text and holds at least one.
				if got.End != len(text) || got.Start >= len(text) {
					t.Errorf("range = %v", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("range = %v, want %v (%q)", got, tt.want, text[got.Start:got.End])
			}
		})
	}
	if got := laidOut(t, e, "", 16, 100, nil).GetWordBoundary(ui.NewTextPosition(0)); got != (ui.TextRange{}) {
		t.Errorf("empty text: %v", got)
	}
}

func TestLineBoundaryAndNumbers(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "one two\nthree four five", 16, 80, nil)
	lines := lineTexts(p)
	if len(lines) < 3 {
		t.Fatalf("lines = %q", lines)
	}
	ranges := make([]ui.TextRange, len(p.lines))
	for i, l := range p.lines {
		ranges[i] = ui.TextRange{Start: p.byteOf(l.start), End: p.byteOf(l.end)}
	}
	down, up := ui.TextAffinityDownstream, ui.TextAffinityUpstream
	tests := []struct {
		name string
		pos  ui.TextPosition
		want ui.TextRange
	}{
		{"start of the text", ui.TextPosition{Offset: 0, Affinity: down}, ranges[0]},
		{"inside the first line", ui.TextPosition{Offset: 3, Affinity: down}, ranges[0]},
		{"the newline belongs to its line", ui.TextPosition{Offset: 7, Affinity: down}, ranges[0]},
		{"inside the second line", ui.TextPosition{Offset: ranges[1].Start + 2, Affinity: down}, ranges[1]},
		{"at a soft break, downstream is the next line", ui.TextPosition{Offset: ranges[1].End, Affinity: down}, ranges[2]},
		{"at a soft break, upstream is the line before", ui.TextPosition{Offset: ranges[1].End, Affinity: up}, ranges[1]},
		{"end of the text", ui.TextPosition{Offset: len(p.content.Text), Affinity: down}, ranges[len(ranges)-1]},
		{"past the end", ui.TextPosition{Offset: 999, Affinity: down}, ranges[len(ranges)-1]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.GetLineBoundary(tt.pos); got != tt.want {
				t.Errorf("boundary = %v, want %v", got, tt.want)
			}
		})
	}
	if p.NumberOfLines() != len(p.lines) {
		t.Errorf("NumberOfLines = %d", p.NumberOfLines())
	}
	for off, want := range map[int]int{0: 0, 6: 0, 7: 0, 8: 1} {
		if got, ok := p.GetLineNumberAt(off); !ok || got != want {
			t.Errorf("GetLineNumberAt(%d) = %d, %v, want %d", off, got, ok, want)
		}
	}
	for _, off := range []int{-1, len(p.content.Text), 100} {
		if _, ok := p.GetLineNumberAt(off); ok {
			t.Errorf("GetLineNumberAt(%d) found a line", off)
		}
	}
	if m, ok := p.GetLineMetricsAt(1); !ok || m.LineNumber != 1 {
		t.Errorf("GetLineMetricsAt(1) = %+v, %v", m, ok)
	}
	for _, n := range []int{-1, len(p.lines)} {
		if _, ok := p.GetLineMetricsAt(n); ok {
			t.Errorf("GetLineMetricsAt(%d) found a line", n)
		}
	}
	if got := laidOut(t, e, "", 16, 100, nil).GetLineBoundary(ui.NewTextPosition(0)); got != (ui.TextRange{}) {
		t.Errorf("empty text: %v", got)
	}
	// Text cut by maxLines is on no visible line.
	cut := laidOut(t, e, pangram, 16, 100, paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(2)}))
	if _, ok := cut.GetLineNumberAt(len(pangram) - 1); ok {
		t.Error("a cut-off offset is on a line")
	}
	if cut.NumberOfLines() != 2 {
		t.Errorf("NumberOfLines = %d, want 2", cut.NumberOfLines())
	}
}

func TestEllipsisWithoutAMaximumNumberOfLines(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{Ellipsis: "…"})
	t.Run("text that wraps is not cut", func(t *testing.T) {
		p := laidOut(t, e, pangram, 16, 100, ps)
		if len(p.lines) != 4 || p.DidExceedMaxLines() {
			t.Errorf("lines %q, exceeded %v", lineTexts(p), p.DidExceedMaxLines())
		}
	})
	t.Run("a word wider than the line is ellipsized and ends the text", func(t *testing.T) {
		p := laidOut(t, e, "short supercalifragilisticexpialidocious tail", 16, 100, ps)
		if got := lineTexts(p); len(got) != 2 || got[0] != "short " || !strings.HasPrefix(got[1], "supercal") || strings.Contains(got[1], "tail") {
			t.Fatalf("lines = %q", got)
		}
		if !p.DidExceedMaxLines() {
			t.Error("DidExceedMaxLines is false")
		}
		last := p.lines[1]
		if !last.runs[len(last.runs)-1].ellipsis || last.width > 100+1e-3 {
			t.Errorf("the last line is not an ellipsized line that fits: width %v", last.width)
		}
	})
	t.Run("without an ellipsis the word is cut", func(t *testing.T) {
		p := laidOut(t, e, "short supercalifragilisticexpialidocious tail", 16, 100, nil)
		if p.DidExceedMaxLines() || len(p.lines) < 3 {
			t.Errorf("lines = %q", lineTexts(p))
		}
	})
}

func TestPlaceholders(t *testing.T) {
	e := bundledEngine(t)
	const size = 16.0
	asc, desc := robotoAscent*size, robotoDescent*size
	build := func(w, h float64, al ui.PlaceholderAlignment, opts ...ui.AddPlaceholderOpts) (*paragraph, ui.TextBox) {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(roboto(size))
		b.AddText("ab")
		b.AddPlaceholder(w, h, al, opts...)
		b.AddText("cd")
		p := b.Build().Layouter().(*paragraph)
		p.Layout(ui.NewParagraphConstraints(1000))
		bs := p.GetBoxesForPlaceholders()
		if len(bs) != 1 {
			t.Fatalf("placeholder boxes = %v", bs)
		}
		return p, bs[0]
	}
	t.Run("baseline default: the bottom edge sits on the baseline", func(t *testing.T) {
		p, b := build(30, 20, ui.PlaceholderAlignmentBaseline)
		m := p.ComputeLineMetrics()[0]
		if !near(b.Right-b.Left, 30) || !near(b.Bottom-b.Top, 20) || !near(b.Bottom, m.Baseline) {
			t.Errorf("box = %v, baseline %v", b, m.Baseline)
		}
		// 20 above the baseline: the line grows upwards.
		if !near(m.Ascent, 20) || !near(m.Descent, desc) {
			t.Errorf("ascent %v descent %v", m.Ascent, m.Descent)
		}
		if !near(b.Left, p.sum(0, 2)) {
			t.Errorf("left = %v, want after the text before it (%v)", b.Left, p.sum(0, 2))
		}
	})
	t.Run("baseline offset", func(t *testing.T) {
		p, b := build(30, 20, ui.PlaceholderAlignmentBaseline, ui.AddPlaceholderOpts{BaselineOffset: ui.Ptr(5.0)})
		m := p.ComputeLineMetrics()[0]
		if !near(b.Top, m.Baseline-5) || !near(b.Bottom, m.Baseline+15) {
			t.Errorf("box = %v, baseline %v", b, m.Baseline)
		}
		if !near(m.Ascent, asc) || !near(m.Descent, 15) {
			t.Errorf("ascent %v descent %v", m.Ascent, m.Descent)
		}
	})
	t.Run("above the baseline", func(t *testing.T) {
		p, b := build(30, 40, ui.PlaceholderAlignmentAboveBaseline)
		m := p.ComputeLineMetrics()[0]
		if !near(b.Bottom, m.Baseline) || !near(b.Top, m.Baseline-40) || !near(m.Descent, desc) {
			t.Errorf("box = %v, baseline %v descent %v", b, m.Baseline, m.Descent)
		}
	})
	t.Run("below the baseline", func(t *testing.T) {
		p, b := build(30, 40, ui.PlaceholderAlignmentBelowBaseline)
		m := p.ComputeLineMetrics()[0]
		if !near(b.Top, m.Baseline) || !near(b.Bottom, m.Baseline+40) || !near(m.Ascent, asc) || !near(m.Descent, 40) {
			t.Errorf("box = %v, baseline %v ascent %v descent %v", b, m.Baseline, m.Ascent, m.Descent)
		}
	})
	t.Run("top of a tall placeholder", func(t *testing.T) {
		p, b := build(30, 50, ui.PlaceholderAlignmentTop)
		m := p.ComputeLineMetrics()[0]
		if !near(b.Top, 0) || !near(b.Bottom, 50) || !near(m.Ascent, asc) || !near(m.Descent, 50-asc) {
			t.Errorf("box = %v, ascent %v descent %v", b, m.Ascent, m.Descent)
		}
	})
	t.Run("bottom of a tall placeholder", func(t *testing.T) {
		p, b := build(30, 50, ui.PlaceholderAlignmentBottom)
		m := p.ComputeLineMetrics()[0]
		if !near(b.Bottom, m.Baseline+m.Descent) || !near(b.Top, m.Baseline+m.Descent-50) || !near(m.Descent, desc) || !near(m.Ascent, 50-desc) {
			t.Errorf("box = %v, ascent %v descent %v baseline %v", b, m.Ascent, m.Descent, m.Baseline)
		}
	})
	t.Run("middle of a tall placeholder", func(t *testing.T) {
		p, b := build(30, 50, ui.PlaceholderAlignmentMiddle)
		m := p.ComputeLineMetrics()[0]
		if !near(m.Ascent+m.Descent, 50) || !near(b.Top, 0) || !near(b.Bottom, 50) {
			t.Errorf("box = %v, ascent %v descent %v", b, m.Ascent, m.Descent)
		}
	})
	t.Run("a small placeholder aligned to the top, bottom or middle does not change the line", func(t *testing.T) {
		plain := laidOut(t, e, "abcd", size, 1000, nil).ComputeLineMetrics()[0]
		for _, al := range []ui.PlaceholderAlignment{ui.PlaceholderAlignmentTop, ui.PlaceholderAlignmentBottom, ui.PlaceholderAlignmentMiddle} {
			p, b := build(10, 8, al)
			m := p.ComputeLineMetrics()[0]
			if !near(m.Ascent, plain.Ascent) || !near(m.Descent, plain.Descent) || !near(b.Bottom-b.Top, 8) {
				t.Errorf("%v: metrics %+v box %v", al, m, b)
			}
			switch al {
			case ui.PlaceholderAlignmentTop:
				if !near(b.Top, m.Baseline-m.Ascent) {
					t.Errorf("top: %v", b)
				}
			case ui.PlaceholderAlignmentBottom:
				if !near(b.Bottom, m.Baseline+m.Descent) {
					t.Errorf("bottom: %v", b)
				}
			default:
				if !near((b.Top+b.Bottom)/2, m.Baseline-m.Ascent+(m.Ascent+m.Descent)/2) {
					t.Errorf("middle: %v", b)
				}
			}
		}
	})
	t.Run("scale", func(t *testing.T) {
		_, b := build(30, 20, ui.PlaceholderAlignmentBaseline, ui.AddPlaceholderOpts{Scale: ui.Ptr(2.0)})
		if !near(b.Right-b.Left, 60) || !near(b.Bottom-b.Top, 40) {
			t.Errorf("box = %v", b)
		}
	})
	t.Run("the text around it keeps its place", func(t *testing.T) {
		p, b := build(30, 20, ui.PlaceholderAlignmentBaseline)
		after := p.GetBoxesForRange(5, 6, 0, 0) // "c": the placeholder character has three bytes
		if len(after) != 1 || !near(after[0].Left, b.Right) {
			t.Errorf("text after the placeholder starts at %v, the placeholder ends at %v", after, b.Right)
		}
		// The placeholder is one character of the text, and it has a range box.
		own := p.GetBoxesForRange(2, 2+len("￼"), 0, 0)
		if len(own) != 1 || !near(own[0].Left, b.Left) || !near(own[0].Right, b.Right) {
			t.Errorf("range box of the placeholder = %v, placeholder box %v", own, b)
		}
	})
	t.Run("several, in order, on several lines", func(t *testing.T) {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(roboto(size))
		for i := 0; i < 4; i++ {
			b.AddText("word ")
			b.AddPlaceholder(40, 10+float64(i), ui.PlaceholderAlignmentBaseline)
			b.AddText(" ")
		}
		if b.PlaceholderCount() != 4 || !slices.Equal(b.PlaceholderScales(), []float64{1, 1, 1, 1}) {
			t.Fatalf("count %d scales %v", b.PlaceholderCount(), b.PlaceholderScales())
		}
		p := b.Build().Layouter().(*paragraph)
		p.Layout(ui.NewParagraphConstraints(120))
		bs := p.GetBoxesForPlaceholders()
		if len(bs) != 4 || len(p.lines) < 2 {
			t.Fatalf("%d boxes on %d lines", len(bs), len(p.lines))
		}
		for i, bx := range bs {
			if !near(bx.Bottom-bx.Top, 10+float64(i)) || !near(bx.Right-bx.Left, 40) {
				t.Errorf("box %d = %v", i, bx)
			}
			if i > 0 && bx.Top < bs[i-1].Top-1 && bx.Left <= bs[i-1].Left {
				t.Errorf("box %d is before box %d", i, i-1)
			}
		}
	})
	t.Run("a placeholder cut by maxLines has no box", func(t *testing.T) {
		b := ui.NewParagraphBuilder(e, paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1)}))
		b.PushStyle(roboto(size))
		b.AddPlaceholder(10, 10, ui.PlaceholderAlignmentBaseline)
		b.AddText(strings.Repeat("word ", 20))
		b.AddPlaceholder(10, 10, ui.PlaceholderAlignmentBaseline)
		p := b.Build().Layouter().(*paragraph)
		p.Layout(ui.NewParagraphConstraints(80))
		if n := len(p.GetBoxesForPlaceholders()); n != 1 {
			t.Errorf("%d boxes, want 1", n)
		}
	})
	t.Run("negative size panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("no panic")
			}
		}()
		ui.NewParagraphBuilder(e, nil).AddPlaceholder(-1, 1, ui.PlaceholderAlignmentBaseline)
	})
}

func TestDecorations(t *testing.T) {
	e := bundledEngine(t)
	red := ui.NewColor(0xFFFF0000)
	deco := func(d ui.TextDecoration, st ui.TextDecorationStyle, thickness float64) *ui.TextStyle {
		o := ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(20.0), Decoration: ui.Ptr(d), DecorationStyle: st, DecorationColor: &red}
		if thickness != 0 {
			o.DecorationThickness = ui.Ptr(thickness)
		}
		return ui.NewTextStyle(o)
	}
	t.Run("none draws nothing", func(t *testing.T) {
		if got := paragraphIn(t, e, "plain", roboto(20), nil, 1000).Decorations(); len(got) != 0 {
			t.Errorf("decorations = %+v", got)
		}
	})
	t.Run("underline", func(t *testing.T) {
		p := paragraphIn(t, e, "underlined text", deco(ui.TextDecorationUnderline, ui.TextDecorationStyleDashed, 0), nil, 1000)
		ds := p.Decorations()
		if len(ds) != 1 {
			t.Fatalf("decorations = %+v", ds)
		}
		d := ds[0]
		l := p.lines[0]
		if d.Kind != ui.TextDecorationUnderline || d.Style != ui.TextDecorationStyleDashed || d.Color != red {
			t.Errorf("segment = %+v", d)
		}
		if !near(d.X0, 0) || !near(d.X1, l.width) {
			t.Errorf("extent [%v,%v], want [0,%v]", d.X0, d.X1, l.width)
		}
		// Below the baseline, by about the font's underline position, and as thick as its underline.
		if d.Y <= l.baseline || d.Y > l.baseline+20*0.3 || d.Thickness <= 0 || d.Thickness > 3 {
			t.Errorf("y %v (baseline %v) thickness %v", d.Y, l.baseline, d.Thickness)
		}
	})
	t.Run("thickness multiplier", func(t *testing.T) {
		one := paragraphIn(t, e, "text", deco(ui.TextDecorationUnderline, 0, 0), nil, 1000).Decorations()[0]
		three := paragraphIn(t, e, "text", deco(ui.TextDecorationUnderline, 0, 3), nil, 1000).Decorations()[0]
		if !near(three.Thickness, 3*one.Thickness) {
			t.Errorf("thickness %v, three times %v", three.Thickness, 3*one.Thickness)
		}
	})
	t.Run("overline above and line through in the middle", func(t *testing.T) {
		p := paragraphIn(t, e, "text", deco(ui.CombineTextDecorations(ui.TextDecorationOverline, ui.TextDecorationLineThrough, ui.TextDecorationUnderline), 0, 0), nil, 1000)
		by := map[ui.TextDecoration]DecorationSegment{}
		for _, d := range p.Decorations() {
			by[d.Kind] = d
		}
		if len(by) != 3 {
			t.Fatalf("kinds = %v", by)
		}
		base := p.lines[0].baseline
		if !(by[ui.TextDecorationOverline].Y < by[ui.TextDecorationLineThrough].Y && by[ui.TextDecorationLineThrough].Y < base && base < by[ui.TextDecorationUnderline].Y) {
			t.Errorf("overline %v, line through %v, baseline %v, underline %v", by[ui.TextDecorationOverline].Y, by[ui.TextDecorationLineThrough].Y, base, by[ui.TextDecorationUnderline].Y)
		}
		if top := p.lines[0].baseline - p.lines[0].ascent; by[ui.TextDecorationOverline].Y < top-1 {
			t.Errorf("overline above the line: %v < %v", by[ui.TextDecorationOverline].Y, top)
		}
	})
	t.Run("white space at the end of a line is not decorated", func(t *testing.T) {
		p := paragraphIn(t, e, "word    ", deco(ui.TextDecorationUnderline, 0, 0), nil, 1000)
		d := p.Decorations()[0]
		if !near(d.X1, p.sum(0, 4)) {
			t.Errorf("underline ends at %v, the word at %v", d.X1, p.sum(0, 4))
		}
	})
	t.Run("pieces of one style are one line, pieces of two styles are two", func(t *testing.T) {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(deco(ui.TextDecorationUnderline, 0, 0))
		b.AddText("one ")
		b.AddText("two ")
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{Color: &red}))
		b.AddText("three ")
		b.Pop()
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{DecorationColor: ui.Ptr(ui.NewColor(0xFF00FF00))}))
		b.AddText("four")
		p := b.Build().Layouter().(*paragraph)
		p.Layout(ui.NewParagraphConstraints(1000))
		ds := p.Decorations()
		if len(ds) != 2 {
			t.Fatalf("decorations = %+v", ds)
		}
		if ds[1].Color == red || !near(ds[0].X1, p.sum(0, 14)) {
			t.Errorf("segments = %+v", ds)
		}
	})
	t.Run("a decorated line of each wrapped line", func(t *testing.T) {
		p := paragraphIn(t, e, pangram, deco(ui.TextDecorationLineThrough, 0, 0), nil, 120)
		ds := p.Decorations()
		if len(ds) != len(p.lines) {
			t.Fatalf("%d segments for %d lines", len(ds), len(p.lines))
		}
		for i, d := range ds {
			if d.Line != i {
				t.Errorf("segment %d is on line %d", i, d.Line)
			}
		}
	})
	t.Run("the ellipsis is decorated", func(t *testing.T) {
		ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…"})
		p := paragraphIn(t, e, pangram, deco(ui.TextDecorationUnderline, 0, 0), ps, 120)
		ds := p.Decorations()
		if len(ds) != 1 || !near(ds[0].X1, p.lines[0].width) {
			t.Errorf("segments = %+v, line width %v", ds, p.lines[0].width)
		}
	})
}

func TestQueriesOnTextStyles(t *testing.T) {
	e := bundledEngine(t)
	// Mixed sizes: the tight box of the small text is shorter than that of the big text, and
	// the max box of both is the same.
	b := ui.NewParagraphBuilder(e, nil)
	b.PushStyle(roboto(10))
	b.AddText("ab")
	b.PushStyle(roboto(30))
	b.AddText("cd")
	p := b.Build().Layouter().(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	small := p.GetBoxesForRange(0, 2, ui.BoxHeightStyleTight, 0)[0]
	big := p.GetBoxesForRange(2, 4, ui.BoxHeightStyleTight, 0)[0]
	if small.Bottom-small.Top >= big.Bottom-big.Top || !near(small.Bottom-small.Top, 10*(robotoAscent+robotoDescent)) {
		t.Errorf("small %v big %v", small, big)
	}
	if !near(small.Bottom, big.Bottom-(30-10)*robotoDescent) {
		t.Errorf("the boxes do not share a baseline: %v %v", small, big)
	}
	smallMax := p.GetBoxesForRange(0, 2, ui.BoxHeightStyleMax, 0)[0]
	bigMax := p.GetBoxesForRange(2, 4, ui.BoxHeightStyleMax, 0)[0]
	if !near(smallMax.Top, bigMax.Top) || !near(smallMax.Bottom, bigMax.Bottom) {
		t.Errorf("max boxes differ: %v %v", smallMax, bigMax)
	}
	// Both boxes are next to each other.
	if !near(small.Right, big.Left) || math.IsNaN(small.Left) {
		t.Errorf("small %v big %v", small, big)
	}
}
