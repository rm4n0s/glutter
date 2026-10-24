// Command android-fonts is a development example (step 25.10): a Material page with a line of text in
// each of several scripts (Latin, Greek, Cyrillic, Arabic, Hebrew, Devanagari, Thai, Chinese, Japanese,
// Korean and emoji) in the system's fonts. At the start it also asks the system font matcher of the
// platform (the one the engine uses, internal/engine.SystemFontMatcher) for a font with a glyph for
// every character of every line, and logs what it found, so that the device tests can check that no
// character is left to a missing-glyph box. It may use internal packages (it is a development example).
//
// GLUTTER_DEMO is a BCP 47 language tag (for instance ja-JP): the locale of the app. The default is
// the system's.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-fonts
//	go run ./cmd/glutter-android run -env GLUTTER_DEMO=ja-JP ./examples/dev/android-fonts
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/text"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

// lines are the sample texts: a name and the characters.
// A line with a locale is drawn with it, so that the Han characters take the forms of that language.
var lines = []struct{ name, text, locale string }{
	{"latin", "Hamburgefonstiv Ægir çà ñ", ""},
	{"greek", "Ηλιοβασίλεμα ΩΣ άέ", ""},
	{"cyrillic", "Съешь же ещё этих мягких", ""},
	{"arabic", "مرحبا بالعالم", ""},
	{"hebrew", "שלום עולם", ""},
	{"devanagari", "नमस्ते दुनिया", ""},
	{"thai", "สวัสดีชาวโลก", ""},
	{"chinese", "你好，世界。直骨", "zh-CN"},
	{"japanese", "こんにちは世界。直骨", "ja-JP"},
	{"korean", "안녕하세요 세계", "ko-KR"},
	{"emoji", "😀🎉👍❤️🚀", ""},
}

func main() {
	locale := os.Getenv("GLUTTER_DEMO")
	logCoverage(locale)
	err := m.RunApp(newApp(locale), m.RunAppOpts{
		AppID:  "dev.glutter.androidfonts",
		Window: m.WindowConfig{Title: "Android fonts", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// logCoverage logs, for each line, the characters that no font of the system matcher has a glyph for
// (missing=), with the locale's language.
func logCoverage(locale string) {
	ctx := context.Background()
	lang := strings.SplitN(locale, "-", 2)[0]
	if locale == "" {
		lang = "en"
	}
	c := text.NewCollection(text.CollectionOpts{Matcher: engine.SystemFontMatcher(ctx, slog.Default())})
	for _, l := range lines {
		var missing []rune
		for _, r := range l.text {
			if !covered(ctx, c, r, lineLocale(l.locale, locale)) {
				missing = append(missing, r)
			}
		}
		slog.Info("coverage", "line", l.name, "lang", lang, "missing", len(missing), "chars", string(missing))
	}
}

// lineLocale is the locale of a line: its own, or the app's.
func lineLocale(own, app string) string {
	if own != "" {
		return own
	}
	return app
}

// covered says whether some system font of the sans-serif chain for the locale, or the matcher's
// fallback, has a glyph for r. The format controls and the variation selectors need none.
func covered(ctx context.Context, c *text.Collection, r rune, lang string) bool {
	if r == 0xFE0F || r == 0x200D || r == ' ' || r == '\n' {
		return true
	}
	files, err := c.Match(ctx, "sans-serif", 400, false, lang)
	if err != nil {
		slog.Error("match", "err", err)
		return false
	}
	if f, ok, err := c.Fallback(ctx, "sans-serif", 400, false, lang, r); err == nil && ok {
		files = append([]text.FontFile{f}, files...)
	}
	for _, f := range files {
		face, err := c.Face(ctx, f)
		if err != nil {
			continue
		}
		if _, ok := face.NominalGlyph(r); ok {
			return true
		}
	}
	return false
}

func newApp(locale string) m.Widget {
	var loc m.Opt[ui.Locale]
	if l, ok := ui.ParseLocale(locale); ok {
		loc = m.Some(l)
	}
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android fonts",
		DebugShowCheckedModeBanner: m.Some(false),
		Locale:                     loc,
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct{ m.StateOf[*page] }

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	var rows []m.Widget
	for _, l := range lines {
		rows = append(rows, m.NewPadding(m.NewEdgeInsetsSymmetric(4, 16),
			m.NewText(l.text, m.TextOpts{Style: &m.TextStyle{FontSize: m.Some(22.0)}, Locale: lineLocaleOpt(l.locale)})))
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android fonts")}),
		Body:   m.NewSingleChildScrollView(m.NewColumn(rows, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})),
	})
}

// lineLocaleOpt is the locale of a line for Text, or none.
func lineLocaleOpt(tag string) m.Opt[ui.Locale] {
	if l, ok := ui.ParseLocale(tag); ok {
		return m.Some(l)
	}
	return m.Opt[ui.Locale]{}
}
