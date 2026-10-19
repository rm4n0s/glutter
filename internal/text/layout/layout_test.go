package layout

import (
	"math"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

const (
	// Roboto's hhea ascent and descent in em: 1900 and 500 of 2048 units.
	robotoAscent  = 1900.0 / 2048
	robotoDescent = 500.0 / 2048
	pangram       = "The quick brown fox jumps over the lazy dog"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-3 }

func TestLineBreaksLatin(t *testing.T) {
	e := bundledEngine(t)
	hyphenWidth := wide(t, e, "well-known", 16) - 2
	nbspWidth := wide(t, e, "a\u00a0b", 16) + 1
	tests := []struct {
		name  string
		text  string
		width float64
		want  []string
	}{
		{"fits on one line", pangram, 1000, []string{pangram}},
		{"just fits", pangram, 316, []string{pangram}},
		{"one word wraps", pangram, 315, []string{"The quick brown fox jumps over the lazy ", "dog"}},
		{"two lines", pangram, 200, []string{"The quick brown fox jumps ", "over the lazy dog"}},
		{"three lines", pangram, 150, []string{"The quick brown fox ", "jumps over the lazy ", "dog"}},
		{"four lines", pangram, 100, []string{"The quick ", "brown fox ", "jumps over ", "the lazy dog"}},
		{"one or two words a line", pangram, 60, []string{"The ", "quick ", "brown ", "fox ", "jumps ", "over the ", "lazy dog"}},
		{"a word wider than the line is cut", pangram, 44, []string{"The ", "quick ", "brow", "n fox ", "jumps ", "over ", "the ", "lazy ", "dog"}},
		{"hard breaks", "ab\ncd\n\nef\n", 1000, []string{"ab\n", "cd\n", "\n", "ef\n", ""}},
		{"carriage return and line feed are one break", "ab\r\ncd", 1000, []string{"ab\r\n", "cd"}},
		{"empty text is one empty line", "", 100, []string{""}},
		{"spaces hang at the end", "aaaa      bbbb", 40, []string{"aaaa      ", "bbbb"}},
		{"hyphen is a break opportunity", "well-known fact", hyphenWidth, []string{"well-", "known ", "fact"}},
		{"no-break space does not break", "a b c", nbspWidth, []string{"a b ", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := laidOut(t, e, tt.text, 16, tt.width, nil)
			if got := lineTexts(p); !slices.Equal(got, tt.want) {
				t.Errorf("lines = %q, want %q", got, tt.want)
			}
			for i, l := range p.lines {
				if l.width > tt.width+1e-3 && len(string(p.runes[l.start:l.end])) > 1 {
					t.Errorf("line %d is %v wide, over the width %v", i, l.width, tt.width)
				}
			}
		})
	}
}

func TestLinesCoverTheText(t *testing.T) {
	e := bundledEngine(t)
	texts := []string{pangram, "", "a", "\n", "a\n\nb", strings.Repeat("word ", 50), "ünïcödé wörds hëre", "tab\tseparated\ttext"}
	for _, text := range texts {
		for _, width := range []float64{0, 5, 33, 80, 250, math.Inf(1)} {
			p := laidOut(t, e, text, 14, width, nil)
			var all strings.Builder
			for _, s := range lineTexts(p) {
				all.WriteString(s)
			}
			if all.String() != text {
				t.Errorf("text %q width %v: lines %q do not add up to the text", text, width, lineTexts(p))
			}
			for i := 1; i < len(p.lines); i++ {
				if p.lines[i].start != p.lines[i-1].end {
					t.Errorf("text %q width %v: line %d starts at %d, the line before ends at %d", text, width, i, p.lines[i].start, p.lines[i-1].end)
				}
			}
		}
	}
}

func TestMaxLinesAndEllipsis(t *testing.T) {
	e := bundledEngine(t)
	ellipsis := "…"
	tests := []struct {
		name        string
		maxLines    *int
		ellipsis    string
		text        string
		width       float64
		wantLines   []string // the text kept on each line, without the ellipsis
		wantExceed  bool
		wantEllipsi bool
	}{
		{"no limit, text that wraps", nil, ellipsis, pangram, 150, []string{"The quick brown fo"}, true, true},
		{"no limit, text that fits", nil, ellipsis, "The quick", 150, []string{"The quick"}, false, false},
		{"no limit, no ellipsis", nil, "", pangram, 150, []string{"The quick brown fox ", "jumps over the lazy ", "dog"}, false, false},
		{"limit not reached", ui.Ptr(5), ellipsis, pangram, 150, []string{"The quick brown fox ", "jumps over the lazy ", "dog"}, false, false},
		{"exactly the limit", ui.Ptr(3), ellipsis, pangram, 150, []string{"The quick brown fox ", "jumps over the lazy ", "dog"}, false, false},
		{"two lines with ellipsis", ui.Ptr(2), ellipsis, pangram, 150, []string{"The quick brown fox ", "jumps over the lazy"}, true, true},
		{"one line with ellipsis", ui.Ptr(1), ellipsis, pangram, 150, []string{"The quick brown fo"}, true, true},
		{"two lines no ellipsis", ui.Ptr(2), "", pangram, 150, []string{"The quick brown fox ", "jumps over the lazy "}, true, false},
		{"zero lines", ui.Ptr(0), ellipsis, pangram, 150, nil, true, false},
		{"hard breaks count as lines", ui.Ptr(2), ellipsis, "one\ntwo\nthree", 1000, []string{"one\n", "two"}, true, true},
		{"custom ellipsis", ui.Ptr(1), "...", pangram, 150, []string{"The quick brown fo"}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: tt.maxLines, Ellipsis: tt.ellipsis})
			p := laidOut(t, e, tt.text, 16, tt.width, ps)
			if got := lineTexts(p); !slices.Equal(got, tt.wantLines) {
				t.Errorf("lines = %q, want %q", got, tt.wantLines)
			}
			if p.DidExceedMaxLines() != tt.wantExceed {
				t.Errorf("DidExceedMaxLines = %v, want %v", p.DidExceedMaxLines(), tt.wantExceed)
			}
			if tt.wantEllipsi {
				last := p.lines[len(p.lines)-1]
				ell := last.runs[len(last.runs)-1]
				if !ell.ellipsis {
					t.Fatalf("the last line has no ellipsis run")
				}
				if last.width > tt.width+1e-3 {
					t.Errorf("the ellipsized line is %v wide, over %v", last.width, tt.width)
				}
				// The ellipsis is shaped from its own text, one glyph per character.
				if len(ell.glyphs) != len([]rune(tt.ellipsis)) {
					t.Errorf("ellipsis has %d glyphs, want %d", len(ell.glyphs), len([]rune(tt.ellipsis)))
				}
			}
			for _, l := range p.lines[:max(len(p.lines)-1, 0)] {
				for _, r := range l.runs {
					if r.ellipsis {
						t.Errorf("an ellipsis on a line that is not the last")
					}
				}
			}
		})
	}
}

func TestEllipsisKeepsTheLongestPrefixThatFits(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…"})
	text := strings.Repeat("abcdefghij", 8)
	for _, width := range []float64{20, 55, 100, 200} {
		p := laidOut(t, e, text, 16, width, ps)
		l := p.lines[0]
		ew := l.runs[len(l.runs)-1].width
		kept := p.sum(l.start, l.end)
		if kept+ew > width+1e-3 {
			t.Errorf("width %v: the text %v and the ellipsis %v do not fit", width, kept, ew)
		}
		// One more character would not have fit.
		if l.end < len(p.runes) && kept+p.adv[l.end]+ew <= width+1e-3 {
			t.Errorf("width %v: another character would have fit after %d", width, l.end)
		}
	}
}

func TestIntrinsicWidths(t *testing.T) {
	e := bundledEngine(t)
	hello, world, both := wide(t, e, "hello", 16), wide(t, e, "world", 16), wide(t, e, "hello world", 16)
	space := wide(t, e, "hello world ", 16) - both
	tests := []struct {
		name     string
		text     string
		min, max float64
	}{
		{"empty", "", 0, 0},
		{"one word", "hello", hello, hello},
		{"two words", "hello world", math.Max(hello, world), both},
		{"trailing space is part of the max width only", "hello world ", math.Max(hello, world), both + space},
		{"hard breaks: the widest line", "hello\nhello world\nhi", math.Max(hello, world), both},
		{"a space alone", " ", 0, space},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := laidOut(t, e, tt.text, 16, 1000, nil)
			// The words are measured in context, so allow for kerning at the spaces.
			if math.Abs(p.MinIntrinsicWidth()-tt.min) > 0.5 {
				t.Errorf("MinIntrinsicWidth = %v, want %v", p.MinIntrinsicWidth(), tt.min)
			}
			if math.Abs(p.MaxIntrinsicWidth()-tt.max) > 0.5 {
				t.Errorf("MaxIntrinsicWidth = %v, want %v", p.MaxIntrinsicWidth(), tt.max)
			}
		})
	}
}

func TestIntrinsicWidthsDoNotDependOnLayoutWidth(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, pangram, 16, 1000, nil)
	min, max := p.MinIntrinsicWidth(), p.MaxIntrinsicWidth()
	for _, w := range []float64{10, 100, 400} {
		p.Layout(ui.NewParagraphConstraints(w))
		if p.MinIntrinsicWidth() != min || p.MaxIntrinsicWidth() != max {
			t.Errorf("width %v changed the intrinsic widths", w)
		}
	}
	// At the max intrinsic width the text takes one line, and one pixel less it wraps.
	p.Layout(ui.NewParagraphConstraints(max))
	if len(p.lines) != 1 {
		t.Errorf("%d lines at the max intrinsic width", len(p.lines))
	}
	p.Layout(ui.NewParagraphConstraints(max - 1))
	if len(p.lines) != 2 {
		t.Errorf("%d lines below the max intrinsic width", len(p.lines))
	}
	// At the min intrinsic width no word is cut.
	p.Layout(ui.NewParagraphConstraints(min))
	for _, s := range lineTexts(p) {
		if strings.Contains(strings.TrimSpace(s), " ") && p.sum(0, 0) == 0 {
			continue
		}
	}
	if got := lineTexts(p); len(strings.Fields(strings.Join(got, ""))) != len(strings.Fields(pangram)) {
		t.Errorf("a word was cut at the min intrinsic width: %q", got)
	}
}

func TestSizeFromFontMetrics(t *testing.T) {
	e := bundledEngine(t)
	for _, size := range []float64{10, 14, 16, 20, 33} {
		p := laidOut(t, e, "Hamburgefonstiv", size, 1000, nil)
		m := p.ComputeLineMetrics()
		if len(m) != 1 {
			t.Fatalf("%d lines", len(m))
		}
		asc, desc := robotoAscent*size, robotoDescent*size
		if !near(m[0].Ascent, asc) || !near(m[0].Descent, desc) || !near(m[0].UnscaledAscent, asc) {
			t.Errorf("size %v: ascent %v descent %v unscaled %v, want %v and %v", size, m[0].Ascent, m[0].Descent, m[0].UnscaledAscent, asc, desc)
		}
		if want := math.Round(asc + desc); m[0].Height != want || p.Height() != want {
			t.Errorf("size %v: line height %v, paragraph height %v, want %v", size, m[0].Height, p.Height(), want)
		}
		if !near(m[0].Baseline, asc) || !near(p.AlphabeticBaseline(), asc) {
			t.Errorf("size %v: baseline %v, want %v", size, m[0].Baseline, asc)
		}
		if !near(p.IdeographicBaseline(), asc+desc) {
			t.Errorf("size %v: ideographic baseline %v, want %v", size, p.IdeographicBaseline(), asc+desc)
		}
		if !near(m[0].Width, p.LongestLine()) || !near(m[0].Width, p.MaxIntrinsicWidth()) {
			t.Errorf("size %v: width %v, longest %v", size, m[0].Width, p.LongestLine())
		}
	}
}

func TestLineMetricsOfSeveralLines(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "one\ntwo\nthree", 16, 1000, nil)
	m := p.ComputeLineMetrics()
	if len(m) != 3 {
		t.Fatalf("%d lines", len(m))
	}
	for i, l := range m {
		if l.LineNumber != i || l.Height != 19 || l.HardBreak != (i < 2) {
			t.Errorf("line %d: %+v", i, l)
		}
		if want := 19*float64(i) + robotoAscent*16; !near(l.Baseline, want) {
			t.Errorf("line %d: baseline %v, want %v", i, l.Baseline, want)
		}
	}
	if p.Height() != 57 {
		t.Errorf("height = %v, want 57", p.Height())
	}
}

func TestEmptyParagraphHasTheHeightOfOneLine(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0)})
	tests := []struct {
		name string
		text string
		h    float64
	}{
		{"empty", "", 19},
		{"final newline adds an empty line", "a\n", 38},
		{"two final newlines", "a\n\n", 57},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := build(e, tt.text, nil, ps).(*paragraph)
			p.Layout(ui.NewParagraphConstraints(100))
			if p.Height() != tt.h {
				t.Errorf("height = %v, want %v", p.Height(), tt.h)
			}
		})
	}
	zero := paraStyle(ui.ParagraphStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), MaxLines: ui.Ptr(0)})
	if p := build(e, "text", nil, zero); p.Height() != 0 || !p.DidExceedMaxLines() {
		t.Errorf("maxLines 0: height %v, exceeded %v", p.Height(), p.DidExceedMaxLines())
	}
}

func TestLineHeightAndLeadingDistribution(t *testing.T) {
	e := bundledEngine(t)
	const size = 16.0
	asc, desc := robotoAscent*size, robotoDescent*size
	tests := []struct {
		name         string
		height       *float64
		distribution ui.TextLeadingDistribution
		behavior     *ui.TextHeightBehavior
		lines        int
		ascent       float64 // of the first line
		descent      float64 // of the last line
	}{
		{"no height: the font's own", nil, 0, nil, 2, asc, desc},
		{"height 1.5 proportional", ui.Ptr(1.5), ui.TextLeadingDistributionProportional, nil, 2, asc * 24 / (asc + desc), desc * 24 / (asc + desc)},
		{"height 1.5 even", ui.Ptr(1.5), ui.TextLeadingDistributionEven, nil, 2, asc + (24-asc-desc)/2, desc + (24-asc-desc)/2},
		{"height 1 proportional is the font's own", ui.Ptr(1.0), ui.TextLeadingDistributionProportional, nil, 2, asc * 16 / (asc + desc), desc * 16 / (asc + desc)},
		{"height 2 even", ui.Ptr(2.0), ui.TextLeadingDistributionEven, nil, 2, asc + (32-asc-desc)/2, desc + (32-asc-desc)/2},
		{
			"no height on the first ascent", ui.Ptr(2.0), ui.TextLeadingDistributionEven,
			&ui.TextHeightBehavior{ApplyHeightToFirstAscent: false, ApplyHeightToLastDescent: true, LeadingDistribution: ui.TextLeadingDistributionEven},
			2, asc, desc + (32-asc-desc)/2,
		},
		{
			"no height on the last descent", ui.Ptr(2.0), ui.TextLeadingDistributionEven,
			&ui.TextHeightBehavior{ApplyHeightToFirstAscent: true, ApplyHeightToLastDescent: false, LeadingDistribution: ui.TextLeadingDistributionEven},
			2, asc + (32-asc-desc)/2, desc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(size), Height: tt.height, LeadingDistribution: tt.distribution})
			ps := paraStyle(ui.ParagraphStyleOpts{TextHeightBehavior: tt.behavior})
			p := build(e, "first\nsecond", style, ps).(*paragraph)
			p.Layout(ui.NewParagraphConstraints(1000))
			m := p.ComputeLineMetrics()
			if len(m) != tt.lines {
				t.Fatalf("%d lines", len(m))
			}
			if !near(m[0].Ascent, tt.ascent) {
				t.Errorf("first ascent = %v, want %v", m[0].Ascent, tt.ascent)
			}
			if !near(m[len(m)-1].Descent, tt.descent) {
				t.Errorf("last descent = %v, want %v", m[len(m)-1].Descent, tt.descent)
			}
			if !near(m[0].UnscaledAscent, asc) {
				t.Errorf("unscaled ascent = %v, want %v", m[0].UnscaledAscent, asc)
			}
		})
	}
}

func TestStrut(t *testing.T) {
	e := bundledEngine(t)
	const size = 16.0
	tests := []struct {
		name  string
		strut ui.StrutStyleOpts
		want  float64 // the height of a line
	}{
		{"no strut", ui.StrutStyleOpts{}, 19},
		{"a strut smaller than the text does not shrink it", ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(8.0)}, 19},
		{"a bigger strut raises the line", ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(32.0)}, math.Round((robotoAscent + robotoDescent) * 32)},
		{"a forced strut sets the height", ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(8.0), ForceStrutHeight: ui.Ptr(true)}, math.Round((robotoAscent + robotoDescent) * 8)},
		{"strut height", ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(10.0), Height: ui.Ptr(3.0), ForceStrutHeight: ui.Ptr(true)}, 30},
		{"strut leading adds space", ui.StrutStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(10.0), Leading: ui.Ptr(1.0), ForceStrutHeight: ui.Ptr(true)}, math.Round((robotoAscent+robotoDescent)*10 + 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ps *ui.ParagraphStyle
			if tt.strut.FontFamily != "" {
				ps = paraStyle(ui.ParagraphStyleOpts{StrutStyle: ui.NewStrutStyle(tt.strut)})
			}
			p := laidOut(t, e, "a\nb", size, 1000, ps)
			for i, l := range p.ComputeLineMetrics() {
				if l.Height != tt.want {
					t.Errorf("line %d height = %v, want %v", i, l.Height, tt.want)
				}
			}
		})
	}
}

func TestMixedSizesOnALine(t *testing.T) {
	e := bundledEngine(t)
	b := ui.NewParagraphBuilder(e, nil)
	b.PushStyle(roboto(10))
	b.AddText("small ")
	b.PushStyle(roboto(40))
	b.AddText("BIG")
	b.Pop()
	b.AddText(" small")
	p := b.Build().Layouter().(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	m := p.ComputeLineMetrics()
	if len(m) != 1 {
		t.Fatalf("%d lines", len(m))
	}
	if !near(m[0].Ascent, robotoAscent*40) || !near(m[0].Descent, robotoDescent*40) {
		t.Errorf("the line should take the metrics of the biggest text: %+v", m[0])
	}
	if len(p.lines[0].runs) != 3 {
		t.Errorf("%d runs, want 3", len(p.lines[0].runs))
	}
}

func TestAlignment(t *testing.T) {
	e := bundledEngine(t)
	const width = 200.0
	tests := []struct {
		name  string
		align ui.TextAlign
		dir   ui.TextDirection
		left  func(w float64) float64 // where the first line begins, for a line of width w
	}{
		{"unspecified is start", ui.TextAlignUnspecified, ui.TextDirectionLTR, func(w float64) float64 { return 0 }},
		{"left", ui.TextAlignLeft, ui.TextDirectionLTR, func(w float64) float64 { return 0 }},
		{"right", ui.TextAlignRight, ui.TextDirectionLTR, func(w float64) float64 { return width - w }},
		{"center", ui.TextAlignCenter, ui.TextDirectionLTR, func(w float64) float64 { return (width - w) / 2 }},
		{"start in ltr", ui.TextAlignStart, ui.TextDirectionLTR, func(w float64) float64 { return 0 }},
		{"end in ltr", ui.TextAlignEnd, ui.TextDirectionLTR, func(w float64) float64 { return width - w }},
		{"start in rtl", ui.TextAlignStart, ui.TextDirectionRTL, func(w float64) float64 { return width - w }},
		{"end in rtl", ui.TextAlignEnd, ui.TextDirectionRTL, func(w float64) float64 { return 0 }},
		{"left in rtl", ui.TextAlignLeft, ui.TextDirectionRTL, func(w float64) float64 { return 0 }},
		{"right in rtl", ui.TextAlignRight, ui.TextDirectionRTL, func(w float64) float64 { return width - w }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := paraStyle(ui.ParagraphStyleOpts{TextAlign: tt.align, TextDirection: tt.dir})
			p := laidOut(t, e, "hello world\nhi", 16, width, ps)
			for i, l := range p.lines {
				if !near(l.left, tt.left(l.width)) {
					t.Errorf("line %d: left = %v, want %v (width %v)", i, l.left, tt.left(l.width), l.width)
				}
			}
			if p.Width() != width {
				t.Errorf("Width = %v, want %v", p.Width(), width)
			}
		})
	}
}

func TestTrailingSpacesDoNotMoveTheText(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{TextAlign: ui.TextAlignRight})
	plain := laidOut(t, e, "hello", 16, 200, ps)
	spaced := laidOut(t, e, "hello     ", 16, 200, ps)
	if !near(plain.lines[0].left, spaced.lines[0].left) || !near(plain.lines[0].width, spaced.lines[0].width) {
		t.Errorf("trailing spaces changed the line: left %v/%v width %v/%v", plain.lines[0].left, spaced.lines[0].left, plain.lines[0].width, spaced.lines[0].width)
	}
	if spaced.lines[0].widthWithSpaces <= spaced.lines[0].width {
		t.Errorf("the spaces have no width: %v <= %v", spaced.lines[0].widthWithSpaces, spaced.lines[0].width)
	}
}

func TestJustify(t *testing.T) {
	e := bundledEngine(t)
	const width = 150.0
	ps := paraStyle(ui.ParagraphStyleOpts{TextAlign: ui.TextAlignJustify})
	p := laidOut(t, e, pangram+".\nShort line here\n"+pangram, 16, width, ps)
	if len(p.lines) < 4 {
		t.Fatalf("only %d lines: %q", len(p.lines), lineTexts(p))
	}
	for i, l := range p.lines {
		last := i == len(p.lines)-1
		justified := !last && !l.hardBreak
		switch {
		case justified && (!near(l.width, width) || l.justify <= 0):
			t.Errorf("line %d %q should fill the width: width %v justify %v", i, string(p.runes[l.start:l.end]), l.width, l.justify)
		case !justified && l.justify != 0:
			t.Errorf("line %d %q ends a paragraph and must not be justified", i, string(p.runes[l.start:l.end]))
		}
		if justified {
			// The glyphs reach the right edge.
			end := 0.0
			for _, r := range l.runs {
				end = math.Max(end, r.x+r.width)
			}
			trailing := p.sum(l.visEnd, l.end)
			if math.Abs(end-width) > 1e-3 && math.Abs(end-trailing-width) > 1e-3 {
				t.Errorf("line %d ends at %v, want %v", i, end, width)
			}
		}
	}
}

func TestReuseWithOtherWidths(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, pangram, 16, 1000, nil)
	counts := map[float64]int{}
	for _, w := range []float64{1000, 100, 1000, 60, 100} {
		p.Layout(ui.NewParagraphConstraints(w))
		if n, ok := counts[w]; ok && n != len(p.lines) {
			t.Errorf("width %v gave %d lines, then %d", w, n, len(p.lines))
		}
		counts[w] = len(p.lines)
	}
	if counts[1000] != 1 || counts[100] != 4 || counts[60] != 7 {
		t.Errorf("line counts = %v", counts)
	}
}

func TestInfiniteWidthNeverWrapsSoftly(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, strings.Repeat("word ", 100)+"\nnext", 16, math.Inf(1), nil)
	if len(p.lines) != 2 {
		t.Errorf("%d lines, want 2", len(p.lines))
	}
	if !near(p.Width(), p.LongestLine()) {
		t.Errorf("width %v, longest line %v", p.Width(), p.LongestLine())
	}
}

func TestOffsetsAreUTF8Bytes(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "aé€😀b", 16, 1000, nil)
	want := []int{0, 1, 3, 6, 10, 11}
	if !slices.Equal(p.byteOffset, want) {
		t.Errorf("byteOffset = %v, want %v", p.byteOffset, want)
	}
	if p.byteOf(4) != 10 || utf8.RuneCountInString("aé€😀b") != 5 {
		t.Errorf("byteOf(4) = %d", p.byteOf(4))
	}
}

func TestStyleRunsKeepTheirText(t *testing.T) {
	e := bundledEngine(t)
	b := ui.NewParagraphBuilder(e, nil)
	b.PushStyle(roboto(16))
	b.AddText("plain ")
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontWeight: ui.FontWeightBold}))
	b.AddText("bold ")
	b.Pop()
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontStyle: ui.FontStyleItalic}))
	b.AddText("italic")
	p := b.Build().Layouter().(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	if len(p.styles) != 3 {
		t.Fatalf("%d styles", len(p.styles))
	}
	// Bold text is wider than regular text in the same font.
	regular := wide(t, e, "bold ", 16)
	bold := p.sum(6, 11)
	if bold <= regular {
		t.Errorf("bold %v is not wider than regular %v", bold, regular)
	}
	if got := lineTexts(p); !slices.Equal(got, []string{"plain bold italic"}) {
		t.Errorf("lines = %q", got)
	}
}

func TestMissingFontsDoNotCrash(t *testing.T) {
	c := newEmptyCollection()
	e := NewEngine(t.Context(), c)
	p := build(e, "no fonts at all\nsecond", nil, nil).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(100))
	if len(p.lines) != 2 || p.Height() <= 0 {
		t.Errorf("lines %q height %v", lineTexts(p), p.Height())
	}
}

func TestFontErrorsAreReported(t *testing.T) {
	c := newEmptyCollection()
	var got []error
	e := NewEngine(t.Context(), c, EngineOpts{OnFontError: func(err error) { got = append(got, err) }})
	c2 := newBrokenCollection()
	e2 := NewEngine(t.Context(), c2, EngineOpts{OnFontError: func(err error) { got = append(got, err) }})
	build(e, "x", nil, nil)
	build(e2, "x", nil, nil)
	if len(got) == 0 {
		t.Error("no font error was reported")
	}
}

func TestMultipleOptionsPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewEngine with two options values did not panic")
		}
	}()
	NewEngine(t.Context(), newEmptyCollection(), EngineOpts{}, EngineOpts{})
}
