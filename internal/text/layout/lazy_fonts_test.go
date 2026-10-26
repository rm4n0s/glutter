package layout

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/fonttest"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/goregular"
)

func monoStyle(e *Engine) *styleFonts {
	return e.fontsFor([]string{"monospace"}, ui.FontWeightW400, false, ui.Locale{}, nil)
}

func TestFallbackParsesOnlyTheFontsThatCanHelp(t *testing.T) {
	files := writeManyFonts(t, manyFonts, -1)
	tests := []struct {
		name string
		r    rune
		// parsed is how many files may be parsed at most; want is the index of the file whose face is chosen.
		parsed int
		want   int
	}{
		{"a letter is in the first font", 'a', 1, 0},
		{"a tab", '\t', 1, 0},
		{"a carriage return", '\r', 1, 0},
		{"a line feed", '\n', 1, 0},
		{"the delete control", 0x7F, 1, 0},
		{"a C1 control", 0x85, 1, 0},
		{"a zero width space", 0x200B, 1, 0},
		{"a zero width joiner", 0x200D, 1, 0},
		{"the byte order mark", 0xFEFF, 1, 0},
		{"a variation selector", 0xFE0F, 1, 0},
		{"a soft hyphen", 0x00AD, 1, 0},
		{"an unassigned code point", 0x0378, 1, 0},
		{"a code point beyond Unicode's planes in use", 0x10FFFD, 1, 0},
		{"a Greek letter that only the third font has", thirdFontRune, 2, 2},
		{"a Han character that the fifth font has", 0x4E00 + 4*32 + 1, 1, 4},
		{"a Cyrillic letter that only the last font has", lastFontRune, 1, manyFonts - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errs []error
			e, c := manyFontsEngine(t, files, nil, &errs)
			got := monoStyle(e).ResolveFace(tt.r)
			if n := c.ParsedFonts(); n > tt.parsed {
				t.Errorf("%d fonts were parsed for %U, want at most %d", n, tt.r, tt.parsed)
			}
			want, err := c.Face(context.Background(), files[tt.want])
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.Font != want.Font {
				t.Errorf("%U is drawn with another font than #%d", tt.r, tt.want)
			}
			if len(errs) != 0 {
				t.Errorf("font errors: %v", errs)
			}
		})
	}
}

func TestFallbackKeepsTheAnswerPerRune(t *testing.T) {
	files := writeManyFonts(t, manyFonts, -1)
	for _, r := range []rune{'a', '\t', thirdFontRune, 0x0378} {
		counter := &countingCoverer{manyFontsMatcher: manyFontsMatcher{files}}
		e, c := manyFontsEngine(t, files, counter, nil)
		style := monoStyle(e)
		first := style.ResolveFace(r)
		parsed, asked := c.ParsedFonts(), counter.calls
		for i := 0; i < 3; i++ {
			if again := style.ResolveFace(r); again != first {
				t.Errorf("%U: the face changed", r)
			}
		}
		if c.ParsedFonts() != parsed || counter.calls != asked {
			t.Errorf("%U: the repeated lookups parsed %d more fonts and asked %d more times", r, c.ParsedFonts()-parsed, counter.calls-asked)
		}
	}
}

// countingCoverer is a matcher that counts the questions to its coverage and answers none.
type countingCoverer struct {
	manyFontsMatcher
	calls int
}

func (m *countingCoverer) Covers(context.Context, text.FontFile, rune) (has, known bool) {
	m.calls++
	return false, false // unknown: the collection reads the cmap itself
}

// tableCoverer is a matcher that knows what its fonts have; its files are not all there.
type tableCoverer struct {
	manyFontsMatcher
	has map[string]string
	// asked are the files the collection asked about.
	asked map[string]bool
}

func (m *tableCoverer) Covers(_ context.Context, f text.FontFile, r rune) (has, known bool) {
	m.asked[f.Path] = true
	runes, ok := m.has[f.Path]
	if !ok {
		return false, false
	}
	for _, c := range runes {
		if c == r {
			return true, true
		}
	}
	return false, true
}

func TestFallbackTrustsTheMatcherCoverage(t *testing.T) {
	// 60 files named by the matcher, of which only the third is there. The matcher says which have Greek.
	files := make([]text.FontFile, manyFonts)
	have := map[string]string{}
	real := writeManyFonts(t, 4, -1) // the third is the Greek one
	for i := range files {
		files[i] = text.FontFile{Path: "/nowhere/font" + strconv.Itoa(i) + ".ttf", Family: "F" + strconv.Itoa(i)}
		have[files[i].Path] = "abc"
	}
	files[2], have[real[2].Path] = real[2], "αβγ"
	delete(have, "/nowhere/font2.ttf")
	m := &tableCoverer{manyFontsMatcher: manyFontsMatcher{files}, has: have, asked: map[string]bool{}}
	var errs []error
	e, c := manyFontsEngine(t, files, m, &errs)
	got := monoStyle(e).ResolveFace('β')
	want, err := c.Face(context.Background(), real[2])
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Font != want.Font {
		t.Error("the Greek letter is not drawn with the one font that has it")
	}
	if n := c.ParsedFonts(); n != 1 {
		t.Errorf("%d fonts were parsed, want 1", n)
	}
	if len(errs) != 0 {
		t.Errorf("font errors (a font that the matcher ruled out was opened?): %v", errs)
	}
	if len(m.asked) != 3 { // the first three: the search stops at the third, which has the letter
		t.Errorf("the matcher was asked about %d files, want 3", len(m.asked))
	}
}

func TestFallbackSkipsAFontWhoseCmapCannotBeRead(t *testing.T) {
	files := writeManyFonts(t, 10, 1) // the second font is broken
	var errs []error
	e, c := manyFontsEngine(t, files, nil, &errs)
	style := monoStyle(e)
	for _, r := range []rune{thirdFontRune, 'β', 'γ', 0x4E00 + 5*32, 0x0378} {
		style.ResolveFace(r)
	}
	failed := 0
	for _, err := range errs {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == "FontLoadFailed" {
			failed++
			if !appErr.HasRoute("Collection.Covers->Collection.loadCmap->readCmap.FontLoadFailed") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if failed != 1 {
		t.Errorf("FontLoadFailed was reported %d times for the broken font, want once", failed)
	}
	if got := style.ResolveFace(thirdFontRune); got == nil {
		t.Error("the Greek letter of the third font was not found")
	}
	if n := c.ParsedFonts(); n > 3 {
		t.Errorf("%d fonts were parsed", n)
	}
}

func TestFirstFaceIsTheFirstThatLoads(t *testing.T) {
	// The first file is not there: the first face is the second font's.
	files := writeManyFonts(t, 5, -1)
	files = append([]text.FontFile{{Path: filepath.Join(t.TempDir(), "gone.ttf")}}, files...)
	var errs []error
	e, c := manyFontsEngine(t, files, nil, &errs)
	got := monoStyle(e).ResolveFace('\t')
	want, err := c.Face(context.Background(), files[1])
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Font != want.Font {
		t.Error("a tab is not drawn with the first font that loads")
	}
	if len(errs) == 0 {
		t.Error("the font that is not there was not reported")
	}
	if n := c.ParsedFonts(); n != 2 { // the file that is not there, then the first that loads
		t.Errorf("%d fonts were read, want 2", n)
	}
}

// TestTabsMeasureTheSameWithManyFonts checks the tab stops of step 1.4 against a style of many fonts: the
// tab stops use the space of the paragraph font, which does not depend on the fonts that follow it.
func TestTabsMeasureTheSameWithManyFonts(t *testing.T) {
	files := writeManyFonts(t, manyFonts, -1)
	text1 := "func main() {\n\tx := 1\n\t\ty := \"a\\tb\"\n\tz\t=\t3\n}\n"
	measure := func(files []text.FontFile) (width, height float64, lines []string) {
		e, _ := manyFontsEngine(t, files, nil, nil)
		style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "monospace", FontSize: ui.Ptr(16.0)})
		b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{TabSize: 4}))
		b.PushStyle(style)
		b.AddText(text1)
		p := b.Build().Layouter().(*paragraph)
		p.Layout(ui.NewParagraphConstraints(1000))
		return p.MaxIntrinsicWidth(), p.Height(), lineTexts(p)
	}
	w1, h1, l1 := measure(files[:1])
	wN, hN, lN := measure(files)
	if w1 != wN || h1 != hN {
		t.Errorf("a style of %d fonts measures %v x %v, a style of one font %v x %v", len(files), wN, hN, w1, h1)
	}
	if len(l1) != len(lN) {
		t.Errorf("lines: %q and %q", l1, lN)
	}
	if w1 <= 0 {
		t.Error("the paragraph has no width")
	}
}

// TestControlCharactersChooseTheFirstFaceOfTheStyle checks the other order: a font that has a glyph for the
// tab, listed after one that has none, is not looked for.
func TestControlCharactersChooseTheFirstFaceOfTheStyle(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) text.FontFile {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return text.FontFile{Path: path, Family: name}
	}
	noTab := write("notab.ttf", fonttest.WithCmap(t, goregular.TTF, fonttest.Range(' ', 95)))
	withTab := write("withtab.ttf", fonttest.WithCmap(t, goregular.TTF, append([]rune{'\t'}, fonttest.Range(' ', 95)...)))
	e, c := manyFontsEngine(t, []text.FontFile{noTab, withTab}, nil, nil)
	got := monoStyle(e).ResolveFace('\t')
	want, err := c.Face(context.Background(), noTab)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Font != want.Font {
		t.Error("the tab is drawn with the second font, which has a glyph for it")
	}
	if n := c.ParsedFonts(); n != 1 {
		t.Errorf("%d fonts were parsed, want 1", n)
	}
}

// TestTabsGolden draws a paragraph with tab stops through the software renderer. It was made with the
// font search of step 28.15, which loaded a font for the tab; the picture did not change.
func TestTabsGolden(t *testing.T) {
	e := codeEngine(t)
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{FontFamily: "Go Mono", FontSize: ui.Ptr(16.0), TabSize: 4}))
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Go Mono", Color: ptrColor(0xFF212121)}))
	b.AddText("func f() {\n")
	b.Pop()
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Go Mono", Color: ptrColor(0xFF8C8C8C)}))
	b.AddText("\t\n\t\t")
	b.Pop()
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Go Mono", Color: ptrColor(0xFF1750EB)}))
	b.AddText("x := 1\t// one\n\ty\t=\t2\n}")
	para := b.Build()
	para.Layout(ui.NewParagraphConstraints(300))
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFFFFFF))
	c.DrawRect(ui.Rect{Right: 300, Bottom: 140}, bg)
	c.DrawParagraph(para, ui.Offset{Dx: 8, Dy: 8})
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(sb.Build(), 300, 140)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	goldens.Check(t, "tabs-paragraph", img)
}
