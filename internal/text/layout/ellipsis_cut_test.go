package layout

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// keptText is the text that an ellipsized last line keeps before its ellipsis.
func keptText(t *testing.T, p *paragraph) string {
	t.Helper()
	l := p.lines[len(p.lines)-1]
	er := ellipsisRun(l)
	if er == nil {
		t.Fatalf("the last line has no ellipsis: %v", p)
	}
	return string(p.runes[l.start:er.glyphs[0].cluster])
}

// ellipsisRun is the run of the ellipsis of a line, or nil (it is last in logical order, and
// first in visual order in a right-to-left paragraph).
func ellipsisRun(l *line) *placedRun {
	for i := range l.runs {
		if l.runs[i].ellipsis {
			return &l.runs[i]
		}
	}
	return nil
}

// checkLongestFit checks that the last line is at most width wide, and that the
// next cluster would not have fitted with the ellipsis (the line keeps every
// character that fits).
func checkLongestFit(t *testing.T, p *paragraph, width float64) {
	t.Helper()
	l := p.lines[len(p.lines)-1]
	er := ellipsisRun(l)
	if er == nil {
		t.Fatalf("the last line has no ellipsis: %v", p)
	}
	ew := er.width
	cut := er.glyphs[0].cluster
	if !p.boundary[cut] && cut != len(p.runes) {
		t.Errorf("the line is cut at %d, inside a cluster", cut)
	}
	if l.width > width+1e-3 {
		t.Errorf("the line is %v wide, over the box's %v", l.width, width)
	}
	// The white space at the end of the kept text is dropped, so the next cluster that
	// matters is the first one that is not white space.
	next := p.nextBoundary(cut)
	for next < len(p.runes) && strings.TrimSpace(string(p.runes[cut:next])) == "" {
		next = p.nextBoundary(next)
	}
	if cut < len(p.runes) && p.span(l.start, next, 0)+ew <= width+1e-6 {
		t.Errorf("the cluster %q after the cut would have fitted", string(p.runes[cut:next]))
	}
}

// widthFor is a box width that fits the text prefix and the ellipsis, and not
// one more character.
func widthFor(t *testing.T, e *Engine, prefix string, ps *ui.ParagraphStyle) float64 {
	t.Helper()
	return build(e, prefix, roboto(16), ps).MaxIntrinsicWidth() + build(e, "…", roboto(16), ps).MaxIntrinsicWidth() + 0.2
}

func TestEllipsisCutsTheLastLineAtACharacter(t *testing.T) {
	e := bundledEngine(t)
	const code = "\tfmt.Println(Shout(Greet(\"ev..."
	const spaced = "Hello world this is long and more"
	const dots = "abc…def ghi jkl mno pqr"
	tests := []struct {
		name     string
		text     string
		maxLines *int
		keep     string // the text kept before the ellipsis
	}{
		{"code, one line", code, ui.Ptr(1), "\tfmt.Printl"},
		{"code, no limit", code, nil, "\tfmt.Printl"},
		{"text with spaces, one line", spaced, ui.Ptr(1), "Hello world this i"},
		{"text with spaces, no limit", spaced, nil, "Hello world this i"},
		{"a break after an ellipsis character", "ab…cdefghijklmnopqrstuvw", ui.Ptr(1), "ab…cdefgh"},
		{"text with an ellipsis character, one line", dots, ui.Ptr(1), "abc…def gh"},
		{"text with an ellipsis character, no limit", dots, nil, "abc…def gh"},
		{"a long word after a break", "Hello world abcdefghijklmnopqrst uvw", ui.Ptr(1), "Hello world abcdef"},
		{"a long word after a break, no limit", "Hello world abcdefghijklmnopqrst uvw", nil, "Hello world abcdef"},
		{"a long word after a short one", "ab cdefghijklmnopqrstuvwxyz", ui.Ptr(1), "ab cdefghij"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: tt.maxLines, Ellipsis: "…"})
			width := widthFor(t, e, tt.keep, ps)
			p := laidOut(t, e, tt.text, 16, width, ps)
			if got := keptText(t, p); got != tt.keep {
				t.Errorf("kept %q, want %q", got, tt.keep)
			}
			if len(p.lines) != 1 || !p.DidExceedMaxLines() && tt.maxLines != nil {
				t.Errorf("lines %v, exceeded %v", p, p.DidExceedMaxLines())
			}
			checkLongestFit(t, p, width)
		})
	}
}

func TestEllipsisWithTwoLines(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(2), Ellipsis: "…"})
	const text = "one two three four sixty seven eight nine ten"
	// The first line breaks at a word; the second is cut inside "sixty", after "three four s".
	width := widthFor(t, e, "three four s", ps)
	p := laidOut(t, e, text, 16, width, ps)
	if got := lineTexts(p); len(got) != 2 || got[0] != "one two " {
		t.Fatalf("lines = %v", p)
	}
	if ellipsisRun(p.lines[0]) != nil {
		t.Errorf("the first line has the ellipsis")
	}
	if got := keptText(t, p); got != "three four s" {
		t.Errorf("second line keeps %q, want %q", got, "three four s")
	}
	checkLongestFit(t, p, width)
}

func TestEllipsisOnHardBreaks(t *testing.T) {
	e := bundledEngine(t)
	ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…"})
	// A line that ends at a hard break keeps all of it when it fits; the text after the break is dropped.
	p := laidOut(t, e, "one\ntwo", 16, 1000, ps)
	if got := keptText(t, p); got != "one" {
		t.Errorf("kept %q, want %q", got, "one")
	}
	// A first line that is too long is cut inside, and never takes the text after the break.
	long := strings.Repeat("abcdefghij", 6)
	width := widthFor(t, e, long[:20], ps)
	p = laidOut(t, e, long+"\n"+long, 16, width, ps)
	if got := keptText(t, p); got != long[:20] {
		t.Errorf("kept %q, want %q", got, long[:20])
	}
}

func TestEllipsisNeverCutsInsideACluster(t *testing.T) {
	e := bundledEngine(t)
	texts := map[string]string{
		"combining marks": strings.Repeat("éäô ", 8),
		"emoji with ZWJ":  strings.Repeat("\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466x", 6),
		"emoji modifier":  strings.Repeat("\U0001F44D\U0001F3FDab", 6),
		"flags":           strings.Repeat("\U0001F1EC\U0001F1F7a", 8),
	}
	for name, text := range texts {
		for _, maxLines := range []*int{ui.Ptr(1), nil} {
			ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: maxLines, Ellipsis: "…"})
			for width := 20.0; width < 260; width += 3.7 {
				p := laidOut(t, e, text, 16, width, ps)
				if !p.DidExceedMaxLines() {
					continue
				}
				l := p.lines[len(p.lines)-1]
				cut := ellipsisRun(l).glyphs[0].cluster
				if cut != len(p.runes) && !p.boundary[cut] {
					t.Fatalf("%s, width %v, maxLines %v: cut at %d, inside a cluster of %q", name, width, maxLines, cut, string(p.runes))
				}
				if l.width > width+1e-3 && cut > l.start {
					t.Fatalf("%s, width %v: the line is %v wide", name, width, l.width)
				}
			}
		}
	}
}

func TestEllipsisInRightToLeftText(t *testing.T) {
	const text = "שלום עולם זה טקסט ארוך מאוד שלא נכנס בתיבה"
	e := systemEngine(t, text)
	ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…", TextDirection: ui.TextDirectionRTL})
	for _, width := range []float64{60, 95, 140, 200} {
		p := laidOut(t, e, text, 16, width, ps)
		if !p.DidExceedMaxLines() || len(p.lines) != 1 {
			t.Fatalf("width %v: %v", width, p)
		}
		if got := keptText(t, p); got == "" || !strings.HasPrefix(text, got) {
			t.Errorf("width %v: kept %q", width, got)
		}
		checkLongestFit(t, p, width)
	}
}

func TestEllipsisInABoxNarrowerThanItself(t *testing.T) {
	e := bundledEngine(t)
	for _, maxLines := range []*int{ui.Ptr(1), nil} {
		ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: maxLines, Ellipsis: "…"})
		p := laidOut(t, e, "\tfmt.Println(Shout", 16, 3, ps)
		if got := keptText(t, p); got != "" {
			t.Errorf("maxLines %v: kept %q, want only the ellipsis", maxLines, got)
		}
		l := p.lines[0]
		if len(l.runs) != 1 || !l.runs[0].ellipsis {
			t.Errorf("maxLines %v: the line has %d runs", maxLines, len(l.runs))
		}
	}
}

func TestEllipsisLeavesFittingTextAlone(t *testing.T) {
	e := bundledEngine(t)
	for _, maxLines := range []*int{ui.Ptr(1), ui.Ptr(2), nil} {
		ps := paraStyle(ui.ParagraphStyleOpts{MaxLines: maxLines, Ellipsis: "…"})
		p := laidOut(t, e, "fits", 16, 500, ps)
		if got := lineTexts(p); !slices.Equal(got, []string{"fits"}) || p.DidExceedMaxLines() {
			t.Errorf("maxLines %v: %v exceeded %v", maxLines, p, p.DidExceedMaxLines())
		}
	}
}
