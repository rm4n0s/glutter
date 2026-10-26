package layout

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/fonttest"
	"golang.org/x/image/font/gofont/gomono"
)

// The fonts of these tests share the outlines of Go Mono and differ in their character maps:
//
//   - the first and the last have the printable ASCII characters and nothing else;
//   - the third has Greek (alpha to omega);
//   - the fourth has a Cyrillic letter that nobody else has;
//   - the others have a block of Han each, so that no two of them cover the same character.
const (
	manyFonts     = 60
	thirdFontRune = 'α'
	lastFontRune  = 'ж'
)

// manyFontsMatcher lists the files in order for every family, as fontconfig does for monospace.
type manyFontsMatcher struct{ files []text.FontFile }

func (m manyFontsMatcher) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return m.files, nil
}

// writeManyFonts writes n test fonts and returns their files in order; broken is the index of a font
// whose cmap cannot be read (-1 for none).
func writeManyFonts(tb testing.TB, n, broken int) []text.FontFile {
	return writeManyFontsFrom(tb, gomono.TTF, n, broken)
}

// writeManyFontsFrom is writeManyFonts with the outlines of another TrueType font, such as a large one for the benchmark.
func writeManyFontsFrom(tb testing.TB, base []byte, n, broken int) []text.FontFile {
	tb.Helper()
	ascii := fonttest.Range(' ', 95)
	files := make([]text.FontFile, n)
	for i := range files {
		var data []byte
		switch i {
		case broken:
			data = fonttest.WithBrokenCmap(tb, base)
		case 0:
			data = fonttest.WithCmap(tb, base, ascii)
		case n - 1:
			data = fonttest.WithCmap(tb, base, append(append([]rune{}, ascii...), lastFontRune))
		case 2:
			data = fonttest.WithCmap(tb, base, fonttest.Range('α', 25))
		default:
			data = fonttest.WithCmap(tb, base, fonttest.Range(0x4E00+rune(i)*32, 32))
		}
		path := filepath.Join(tb.TempDir(), "font"+strconv.Itoa(i)+".ttf")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			tb.Fatal(err)
		}
		files[i] = text.FontFile{Path: path, Family: "Test Mono " + strconv.Itoa(i)}
	}
	return files
}

// manyFontsEngine is an engine over the test fonts, with no bundled font and no system font. Font
// errors are collected in errs.
func manyFontsEngine(tb testing.TB, files []text.FontFile, m text.Matcher, errs *[]error) (*Engine, *text.Collection) {
	tb.Helper()
	if m == nil {
		m = manyFontsMatcher{files}
	}
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	e := NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) {
		if errs != nil {
			*errs = append(*errs, err)
		}
	}})
	return e, c
}
