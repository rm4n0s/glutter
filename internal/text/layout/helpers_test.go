package layout

import (
	"context"
	"fmt"
	"github.com/rm4n0s/errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

// TestMain points the cache of the font scan to a directory of its own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "glutter-layout-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CACHE_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(code)
}

// noSystemFonts is a text.Matcher that knows no system font: with it the only fonts are
// the bundled ones, so that the tests do not depend on the machine.
type noSystemFonts struct{}

func (noSystemFonts) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// bundledEngine is an engine with Roboto and Material Icons only.
func bundledEngine(t *testing.T) *Engine {
	t.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	return NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) { t.Errorf("font error: %v", err) }})
}

// systemEngine is an engine with Roboto and the fonts of the machine, or skips the test
// when the machine has none that cover the sample text.
func systemEngine(t *testing.T, sample string) *Engine {
	t.Helper()
	c := text.NewCollection()
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(context.Background(), c)
	st := ui.NewTextStyle()
	p := build(e, sample, st, nil)
	pp := p.(*paragraph)
	for _, r := range pp.runs {
		for _, g := range r.glyphs {
			if g.id == 0 {
				t.Skipf("the system has no font for %q", sample)
			}
		}
	}
	return e
}

// build makes a laid-out paragraph in one style.
func build(e *Engine, s string, style *ui.TextStyle, ps *ui.ParagraphStyle) ui.ParagraphLayout {
	b := ui.NewParagraphBuilder(e, ps)
	if style != nil {
		b.PushStyle(style)
	}
	b.AddText(s)
	return b.Build().Layouter()
}

// roboto is a text style in Roboto at a size.
func roboto(size float64) *ui.TextStyle {
	return ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(size)})
}

// laidOut builds a paragraph in Roboto and lays it out.
func laidOut(t *testing.T, e *Engine, s string, size, width float64, ps *ui.ParagraphStyle) *paragraph {
	t.Helper()
	p := build(e, s, roboto(size), ps).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(width))
	return p
}

// lineTexts is the text of each line.
func lineTexts(p *paragraph) []string {
	out := make([]string, len(p.lines))
	for i, l := range p.lines {
		out[i] = string(p.runes[l.start:l.end])
	}
	return out
}

func (p *paragraph) String() string { return fmt.Sprintf("%q", lineTexts(p)) }

// wide returns the width of a string in Roboto at a size.
func wide(t *testing.T, e *Engine, s string, size float64) float64 {
	t.Helper()
	return build(e, s, roboto(size), nil).MaxIntrinsicWidth()
}

func paraStyle(o ui.ParagraphStyleOpts) *ui.ParagraphStyle { return ui.NewParagraphStyle(o) }

var _ = filepath.Join
var _ = strings.Join

// newEmptyCollection is a collection with no fonts at all.
func newEmptyCollection() *text.Collection {
	return text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
}

// brokenMatcher names a font file that is not there.
type brokenMatcher struct{}

func (brokenMatcher) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return []text.FontFile{{Path: "/nonexistent/glutter/font.ttf"}}, nil
}

// newBrokenCollection is a collection whose only font file is missing.
func newBrokenCollection() *text.Collection {
	return text.NewCollection(text.CollectionOpts{Matcher: brokenMatcher{}})
}

// errorsFrom is errors.FromError.
func errorsFrom(err error) (*errors.Error, bool) { return errors.FromError(err) }
