package layout

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

const (
	arabic = "مرحبا بالعالم"
	hebrew = "שלום עולם"
)

func rtlStyle() *ui.ParagraphStyle {
	return paraStyle(ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionRTL})
}

// visualRuns returns the rune ranges of the runs of a line from left to right.
func visualRuns(l *line) [][2]int {
	out := make([][2]int, len(l.runs))
	for i, r := range l.runs {
		out[i] = [2]int{r.start, r.end}
	}
	return out
}

// checkContiguous checks that the runs of every line follow each other with no gap.
func checkContiguous(t *testing.T, p *paragraph) {
	t.Helper()
	for i, l := range p.lines {
		x := l.runs[0].x
		for _, r := range l.runs {
			if !near(r.x, x) {
				t.Errorf("line %d: a run starts at %v, the one before ends at %v", i, r.x, x)
			}
			x = r.x + r.width
			for k := 1; k < len(r.glyphs); k++ {
				if r.glyphs[k].x < r.glyphs[k-1].x-1e-6 && r.glyphs[k].xOffset == r.glyphs[k-1].xOffset {
					t.Errorf("line %d: glyph %d is left of glyph %d", i, k, k-1)
				}
			}
		}
	}
}

func TestRightToLeftRunsAreInVisualOrder(t *testing.T) {
	tests := []struct {
		name, text string
	}{
		{"arabic", arabic},
		{"hebrew", hebrew},
	}
	for _, tt := range tests {
		for _, dir := range []ui.TextDirection{ui.TextDirectionLTR, ui.TextDirectionRTL} {
			t.Run(tt.name+" "+dir.String(), func(t *testing.T) {
				e := systemEngine(t, tt.text)
				p := build(e, tt.text, roboto(16), paraStyle(ui.ParagraphStyleOpts{TextDirection: dir})).(*paragraph)
				p.Layout(ui.NewParagraphConstraints(1000))
				if len(p.lines) != 1 || len(p.lines[0].runs) != 1 {
					t.Fatalf("lines %v, runs %v", lineTexts(p), visualRuns(p.lines[0]))
				}
				r := p.lines[0].runs[0]
				if !r.run.rtl {
					t.Fatalf("the run is not right to left")
				}
				// The glyphs are in visual order: left to right, so the last letter of the text
				// comes first.
				for i := 1; i < len(r.glyphs); i++ {
					if r.glyphs[i].cluster >= r.glyphs[i-1].cluster {
						t.Errorf("glyph %d has cluster %d after cluster %d", i, r.glyphs[i].cluster, r.glyphs[i-1].cluster)
					}
					if r.glyphs[i].x < r.glyphs[i-1].x {
						t.Errorf("glyph %d is left of glyph %d", i, i-1)
					}
				}
				if last := len([]rune(tt.text)) - 1; r.glyphs[0].cluster != last {
					t.Errorf("the leftmost glyph is cluster %d, want the last letter %d", r.glyphs[0].cluster, last)
				}
				checkContiguous(t, p)
			})
		}
	}
}

func TestBidiRunOrder(t *testing.T) {
	// The runs of "abc שלום def", from left to right, as rune ranges.
	tests := []struct {
		name string
		text string
		dir  ui.TextDirection
		want [][2]int
	}{
		{"hebrew in an english sentence", "abc שלום def", ui.TextDirectionLTR, [][2]int{{0, 4}, {4, 8}, {8, 12}}},
		{"english in a hebrew sentence", "abc שלום def", ui.TextDirectionRTL, [][2]int{{9, 12}, {3, 9}, {0, 3}}},
		{"numbers in a hebrew sentence", "שלום 123 עולם", ui.TextDirectionRTL, [][2]int{{9, 13}, {5, 8}, {0, 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := systemEngine(t, tt.text)
			p := build(e, tt.text, roboto(16), paraStyle(ui.ParagraphStyleOpts{TextDirection: tt.dir})).(*paragraph)
			p.Layout(ui.NewParagraphConstraints(1000))
			got := visualRuns(p.lines[0])
			// Fonts may split the runs further; compare the order of what each holds.
			if !slices.Equal(got, tt.want) && !sameOrder(got, tt.want) {
				t.Errorf("runs left to right = %v, want %v", got, tt.want)
			}
			checkContiguous(t, p)
		})
	}
}

// sameOrder reports whether two lists of ranges hold the same number of ranges in the same
// order along the text, ignoring a small shift of the boundaries (a space belongs to either
// neighbour).
func sameOrder(a, b [][2]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if abs(a[i][0]-b[i][0]) > 1 || abs(a[i][1]-b[i][1]) > 1 {
			return false
		}
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestRightToLeftLinesBreakAndAlign(t *testing.T) {
	e := systemEngine(t, hebrew)
	p := build(e, hebrew, roboto(16), rtlStyle()).(*paragraph)
	width := p.MinIntrinsicWidth() + 1
	p.Layout(ui.NewParagraphConstraints(width))
	got := lineTexts(p)
	if len(got) != 2 || got[0] != "שלום " || got[1] != "עולם" {
		t.Fatalf("lines = %q", got)
	}
	for i, l := range p.lines {
		// Start alignment of a right-to-left paragraph is on the right.
		if !near(l.left+l.width, width) {
			t.Errorf("line %d ends at %v, want the right edge %v", i, l.left+l.width, width)
		}
	}
	// The first line holds the first word: the logical start is on the right.
	if p.lines[0].start != 0 {
		t.Errorf("the first line starts at %d", p.lines[0].start)
	}
}

func TestCJKBreaksBetweenCharacters(t *testing.T) {
	const text = "日本語のテキストを折り返す、そして続ける。"
	e := systemEngine(t, text)
	p := build(e, text, roboto(16), nil).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(16 * 5))
	got := lineTexts(p)
	if len(got) < 3 {
		t.Fatalf("lines = %q", got)
	}
	for i, l := range got {
		if n := len([]rune(l)); n > 5 {
			t.Errorf("line %d %q has %d characters in a width of 5", i, l, n)
		}
		// No line starts with punctuation that must not start a line.
		if r := []rune(l)[0]; r == '、' || r == '。' {
			t.Errorf("line %d %q starts with %q", i, l, r)
		}
	}
	// Every full-width character is as wide as the font size.
	if w := p.lines[0].width; w < 16*4 {
		t.Errorf("a line of CJK characters is %v wide", w)
	}
}

func TestEmojiUseAnotherFont(t *testing.T) {
	e := systemEngine(t, "😀")
	p := build(e, "a😀b", roboto(16), nil).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(1000))
	if len(p.runs) != 3 {
		t.Fatalf("%d runs, want 3", len(p.runs))
	}
	if p.runs[1].face == p.runs[0].face {
		t.Error("the emoji was shaped with the font of the letters")
	}
	if p.runs[0].face != p.runs[2].face {
		t.Error("the letters before and after the emoji have different fonts")
	}
	if p.adv[1] <= 0 {
		t.Errorf("the emoji has no width")
	}
}

func TestLigaturesAreOneCluster(t *testing.T) {
	e := bundledEngine(t)
	p := laidOut(t, e, "office", 16, 1000, nil)
	if len(p.runs) != 1 {
		t.Fatalf("%d runs", len(p.runs))
	}
	glyphs := len(p.runs[0].glyphs)
	if glyphs >= 6 {
		t.Skipf("the font has no ligature for \"ffi\": %d glyphs", glyphs)
	}
	// "ffi" is shaped to a single glyph: the caret cannot be inside it, and neither can a break.
	boundaries := 0
	for i := 0; i < 6; i++ {
		if p.boundary[i] {
			boundaries++
		}
	}
	if boundaries != glyphs {
		t.Errorf("%d cluster boundaries for %d glyphs", boundaries, glyphs)
	}
	narrow := laidOut(t, e, "office", 16, 1, nil)
	for _, l := range narrow.lines {
		if !p.boundary[l.start] || !p.boundary[l.end] {
			t.Errorf("a line [%d,%d) starts or ends inside a ligature", l.start, l.end)
		}
	}
	// The width of the ligature is shared by its characters.
	total := p.sum(1, 4)
	if !near(p.adv[1], p.adv[2]) && p.boundary[2] == false {
		t.Errorf("advances inside the ligature differ: %v %v (total %v)", p.adv[1], p.adv[2], total)
	}
}
