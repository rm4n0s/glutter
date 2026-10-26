package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

// goSource makes the text of a Go file of n lines with tabs: the indentation of 0 to 3 tabs, a tab before a
// trailing comment, and a tab inside a string. The lines repeat a few shapes with the counter changed, as a
// source file does.
func goSource(n int) []string {
	shapes := []string{
		"func (s *server) handle%d(w http.ResponseWriter, r *http.Request) {",
		"\tif err := s.check(r, %d); err != nil {",
		"\t\treturn fmt.Errorf(\"check %d: %%w\", err)",
		"\t}",
		"\tfor i := 0; i < len(items); i++ {\t// item %d",
		"\t\tswitch items[i].kind {",
		"\t\tcase kindA:",
		"\t\t\ttotal += items[i].size * %d",
		"\t\t}",
		"\t}",
		"\tw.Header().Set(\"X-Table\", \"a\\tb %d\")",
		"}",
		"",
	}
	lines := make([]string, n)
	for i := range lines {
		lines[i] = shapes[i%len(shapes)]
		if strings.Contains(lines[i], "%d") {
			lines[i] = fmt.Sprintf(lines[i], i)
		}
	}
	return lines
}

// tabSpans cuts a line into the spans a highlighter makes of it: the runs of tabs, which are spans of
// their own (the indentation and the tab before a comment), and the text between them.
func tabSpans(line string) (spans []string) {
	for len(line) > 0 {
		n := 1
		for n < len(line) && (line[n] == '\t') == (line[0] == '\t') {
			n++
		}
		spans = append(spans, line[:n])
		line = line[n:]
	}
	return spans
}

// BenchmarkFirstLayoutManyFonts is the first layout of a Go file of 10 000 lines with tabs, in a style whose
// families list 60 fonts of which only the first and the last have the Latin letters (step 28.16). The
// tabs are spans of their own, as a highlighter makes them, so that a run has nothing but a tab in it and the
// layout has to choose a font for the tab: it parsed all the fonts of the style, one after the other, to look
// for a glyph of it. Each iteration starts with a font collection and an engine that have parsed nothing; the
// fonts are the files that writeManyFontsFrom wrote, with the outlines of Roboto. fonts-parsed/op is how many fonts the collection read.
func BenchmarkFirstLayoutManyFonts(b *testing.B) {
	files := writeManyFontsFrom(b, fonts.Roboto(), manyFonts, -1) // 2.3 MB each, as the files of a system are
	var lines [][]string
	for _, line := range goSource(10000) {
		lines = append(lines, tabSpans(line))
	}
	text := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "monospace", FontSize: ui.Ptr(14.0), Color: ptrColor(0xFF212121)})
	space := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "monospace", FontSize: ui.Ptr(14.0), Color: ptrColor(0xFF8C8C8C)})
	ps := ui.NewParagraphStyle(ui.ParagraphStyleOpts{TabSize: 4, MaxLines: ui.Ptr(1)})
	b.ReportAllocs()
	parsed := 0
	for i := 0; i < b.N; i++ {
		e, c := manyFontsEngine(b, files, nil, nil)
		for _, spans := range lines {
			bld := ui.NewParagraphBuilder(e, ps)
			for _, sp := range spans {
				st := text
				if sp[0] == '\t' {
					st = space
				}
				bld.PushStyle(st)
				bld.AddText(sp)
				bld.Pop()
			}
			bld.Build().Layout(ui.NewParagraphConstraints(2000))
		}
		parsed = c.ParsedFonts()
	}
	b.ReportMetric(float64(parsed), "fonts-parsed/op")
}
