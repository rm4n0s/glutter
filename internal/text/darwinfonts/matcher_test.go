package darwinfonts

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

const fontDir = "/System/Library/Fonts/"

// face makes a face of a file with the fake coverage: every font has the characters below 0x0600, and the
// fonts named in cover have the ranges they are given.
func face(family, file string, index, weight int, italic bool) Face {
	return Face{Family: family, Path: fontDir + file, Index: index, Weight: weight, Italic: italic}
}

// macOS14 is the faces of a Mac, as far as the tests need them.
func macOS14() []Face {
	return []Face{
		face("Helvetica Neue", "HelveticaNeue.ttc", 0, 400, false),
		face("Helvetica Neue", "HelveticaNeue.ttc", 1, 700, false),
		face("Helvetica", "Helvetica.ttc", 0, 400, false),
		face("Arial", "Supplemental/Arial.ttf", 0, 400, false),
		face("Times", "Times.ttc", 0, 400, false),
		face("Menlo", "Menlo.ttc", 0, 400, false),
		face("Menlo", "Menlo.ttc", 1, 700, false),
		face("Apple Color Emoji", "Apple Color Emoji.ttc", 0, 400, false),
		face("Apple Symbols", "Apple Symbols.ttf", 0, 400, false),
		face("PingFang SC", "PingFang.ttc", 0, 400, false),
		face("PingFang TC", "PingFang.ttc", 1, 400, false),
		face("Hiragino Sans", "Hiragino.ttc", 0, 400, false),
		face("Apple SD Gothic Neo", "AppleSDGothicNeo.ttc", 0, 400, false),
	}
}

// fakeCover says which characters a font file has: Latin and the like below 0x0600 everywhere but the CJK
// and emoji fonts; Han (0x4E00 to 0x9FFF) in the CJK fonts; the emoji (0x1F300 to 0x1FAFF) in Apple Color Emoji;
// kana in the Japanese font; Hangul in the Korean one. The system UI font has Latin and the symbols.
func fakeCover(path string, _ int, r rune) bool {
	name := strings.TrimPrefix(path, fontDir)
	switch {
	case strings.Contains(name, "LastResort"):
		return true // the font that has a glyph for everything
	case strings.Contains(name, "Emoji"):
		return r >= 0x1F300 && r <= 0x1FAFF
	case strings.Contains(name, "PingFang"):
		return r >= 0x4E00 && r <= 0x9FFF
	case strings.Contains(name, "Hiragino"):
		return (r >= 0x3040 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF)
	case strings.Contains(name, "AppleSDGothicNeo"):
		return r >= 0xAC00 && r <= 0xD7AF
	case strings.Contains(name, "Symbols"):
		return r >= 0x2000 && r <= 0x2BFF
	}
	return r < 0x0600
}

var sfns = FontFile{Path: fontDir + "SFNS.ttf", Family: ".AppleSystemUIFont"}

func newMatcher(t *testing.T, faces []Face, o Options) *Matcher {
	t.Helper()
	o.Faces = faces
	if o.HasGlyph == nil {
		o.HasGlyph = fakeCover
	}
	m, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func withSystemUI(lang string) func(string) (FontFile, bool) {
	return func(string) (FontFile, bool) { return sfns, true }
}

func families(files []text.FontFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Family
	}
	return out
}

func TestTheSystemUIFontIsTheDefaultFamily(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	ctx := context.Background()
	for _, name := range []string{"", ".AppleSystemUIFont", "system-ui", "sans-serif", "-apple-system", "BlinkMacSystemFont", "SF Pro Text", "Segoe UI"} {
		got, err := m.Match(ctx, name, 400, false, "")
		if err != nil || len(got) == 0 || got[0].Path != sfns.Path || got[0].Family != SystemUIFamily {
			t.Errorf("Match(%q) = %v, %v; want the system font first", name, got, err)
		}
	}
	// A family that is not generic gives its own font only.
	got, _ := m.Match(ctx, "Menlo", 400, false, "")
	if len(got) != 1 || got[0].Family != "Menlo" {
		t.Errorf("Match(Menlo) = %v", families(got))
	}
}

func TestGenericFamilies(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	ctx := context.Background()
	tests := []struct {
		generic, first string
	}{
		{"serif", "Times"}, {"monospace", "Menlo"}, {"emoji", "Apple Color Emoji"},
		{"math", "Apple Symbols"}, {"cursive", ""}, {"fantasy", SystemUIFamily},
	}
	for _, tt := range tests {
		got, err := m.Match(ctx, tt.generic, 400, false, "")
		if err != nil {
			t.Fatal(err)
		}
		if tt.first == "" {
			continue // no installed cursive font on this fake Mac: the chain follows
		}
		if len(got) == 0 || got[0].Family != tt.first {
			t.Errorf("Match(%s) = %v, want %s first", tt.generic, families(got), tt.first)
		}
	}
	// Generic families have the chain behind the head.
	got, _ := m.Match(ctx, "sans-serif", 400, false, "")
	if len(got) < 4 {
		t.Errorf("Match(sans-serif) = %v", families(got))
	}
}

func TestMatchChoosesTheFaceNearestToTheStyle(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{})
	ctx := context.Background()
	// Menlo has a regular face (index 0) and a bold one (index 1).
	for _, tt := range []struct {
		weight    int
		wantIndex int
	}{{400, 0}, {500, 0}, {600, 1}, {700, 1}, {900, 1}, {100, 0}} {
		got, err := m.Match(ctx, "Menlo", tt.weight, false, "")
		if err != nil || len(got) != 1 || got[0].Index != tt.wantIndex {
			t.Errorf("Match(Menlo, %d) = %+v, %v; want the face %d", tt.weight, got, err, tt.wantIndex)
		}
	}
	// A family that is not installed has no fonts and no error.
	if got, err := m.Match(ctx, "No Such Family", 400, false, ""); err != nil || len(got) != 0 {
		t.Errorf("Match(unknown) = %v, %v", got, err)
	}
}

func TestMatchOfAnAliasAndItsAliases(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	ctx := context.Background()
	got, err := m.Aliases(ctx, "Consolas")
	if err != nil || !slices.Equal(got, []string{"Menlo"}) {
		t.Errorf("Aliases(Consolas) = %v, %v", got, err)
	}
	got, _ = m.Aliases(ctx, "segoe ui")
	if !slices.Equal(got, []string{SystemUIFamily, "Helvetica Neue"}) {
		t.Errorf("Aliases(segoe ui) = %v", got)
	}
	for _, name := range []string{"CupertinoSystemText", "CupertinoSystemDisplay"} {
		if got, _ := m.Aliases(ctx, name); !slices.Equal(got, []string{SystemUIFamily}) {
			t.Errorf("Aliases(%s) = %v, want the system UI font", name, got)
		}
	}
	if got, _ := m.Aliases(ctx, "Menlo"); len(got) != 0 {
		t.Errorf("Aliases(Menlo) = %v", got)
	}
	// The alias of a font that is not installed has no target.
	mm := newMatcher(t, []Face{face("Arial", "Arial.ttf", 0, 400, false)}, Options{})
	if got, _ := mm.Aliases(ctx, "Consolas"); len(got) != 0 {
		t.Errorf("Aliases(Consolas) with no Menlo = %v", got)
	}
}

func TestTheFallbackChainFollowsTheLanguage(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	ctx := context.Background()
	pos := func(lang, family string) int {
		got, err := m.Match(ctx, "sans-serif", 400, false, lang)
		if err != nil {
			t.Fatal(err)
		}
		return slices.Index(families(got), family)
	}
	// A Japanese request puts the Japanese font before the Chinese ones; Traditional Chinese before
	// Simplified; the default order has Simplified first.
	if pos("ja", "Hiragino Sans") > pos("ja", "PingFang SC") {
		t.Error("ja: the Japanese font is not before PingFang SC")
	}
	if pos("zh-Hant-TW", "PingFang TC") > pos("zh-Hant-TW", "PingFang SC") {
		t.Error("zh-Hant: Traditional is not before Simplified")
	}
	if pos("zh-HK", "PingFang TC") > pos("zh-HK", "PingFang SC") {
		t.Error("zh-HK is Traditional")
	}
	if pos("", "PingFang SC") > pos("", "PingFang TC") {
		t.Error("no language: Simplified is not first")
	}
	if pos("ko", "Apple SD Gothic Neo") > pos("ko", "Hiragino Sans") {
		t.Error("ko: the Korean font is not before the Japanese one")
	}
}

func TestTheLocaleOrdersTheChainWhenTheRequestNamesNoLanguage(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{Locale: "ja-JP", SystemUI: withSystemUI("")})
	got, _ := m.Match(context.Background(), "sans-serif", 400, false, "")
	names := families(got)
	if slices.Index(names, "Hiragino Sans") > slices.Index(names, "PingFang SC") {
		t.Errorf("the Japanese locale does not put Hiragino first: %v", names)
	}
}

func TestFallbackFindsAFontForACharacter(t *testing.T) {
	ctx := context.Background()
	var asked []string
	system := func(family string, r rune, lang string) (FontFile, bool) {
		asked = append(asked, family+"/"+string(r)+"/"+lang)
		switch r {
		case '한':
			return FontFile{Path: fontDir + "AppleSDGothicNeo.ttc", Family: "Apple SD Gothic Neo"}, true
		case 0x1F4A9, 0x0E02:
			return FontFile{Path: fontDir + "LastResort.otf", Family: "LastResort"}, true
		case 0x2603:
			// Core Text names a font that has no glyph in the file: ignored, the chain goes on.
			return FontFile{Path: fontDir + "Nothing.ttf", Family: "Nothing"}, true
		}
		return FontFile{}, false
	}
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI(""), SystemFallback: system})
	tests := []struct {
		name   string
		family string
		r      rune
		lang   string
		want   string
		found  bool
	}{
		{"Latin is in the system font", "", 'a', "", SystemUIFamily, true},
		{"Latin in an installed family", "Menlo", 'a', "", "Menlo", true},
		{"Han comes from the chain", "", '汉', "zh-Hans", "PingFang SC", true},
		{"Han for Japanese", "", '汉', "ja", "Hiragino Sans", true},
		{"emoji is Apple Color Emoji", "", 0x1F600, "", "Apple Color Emoji", true},
		{"Hangul is what Core Text says", "", '한', "", "Apple SD Gothic Neo", true},
		{"the LastResort font is never used", "", 0x1F4A9, "", "Apple Color Emoji", true},
		{"LastResort for a character nothing has", "", 0x0E02, "", "", false},
		{"a symbol the chain has", "", 0x2603, "", "Apple Symbols", true},
		{"nothing has it", "", 0x0E01, "", "", false},
	}
	for _, tt := range tests {
		got, ok, err := m.Fallback(ctx, tt.family, 400, false, tt.lang, tt.r)
		if err != nil || ok != tt.found || (ok && got.Family != tt.want) {
			t.Errorf("%s: Fallback(%q, %U, %q) = %+v, %v, %v; want %q, %v", tt.name, tt.family, tt.r, tt.lang, got, ok, err, tt.want, tt.found)
		}
	}
	// Core Text is asked for the system font when the request is the system font or generic, and for the
	// family when it names an installed one.
	if !slices.Contains(asked, "/한/") {
		t.Errorf("Core Text was asked %v", asked)
	}
	asked = nil
	if _, _, err := m.Fallback(ctx, "Menlo", 400, false, "", '한'); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != "Menlo/한/" {
		t.Errorf("Core Text was asked %v for Menlo", asked)
	}
}

func TestFamiliesAreSortedAndComplete(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	got, err := m.Families(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Menlo", "Helvetica Neue", "PingFang SC"} {
		if !slices.Contains(got, want) {
			t.Errorf("Families lacks %q: %v", want, got)
		}
	}
	// The fonts of the system itself have a dot before their names and are not listed.
	for _, name := range got {
		if strings.HasPrefix(name, ".") {
			t.Errorf("Families lists the system family %q", name)
		}
	}
	if !slices.IsSortedFunc(got, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }) {
		t.Errorf("families are not sorted: %v", got)
	}
}

func TestTheSystemFontAloneIsEnough(t *testing.T) {
	// A machine whose font files cannot be read still has the system font that Core Text names.
	m, err := New(Options{SystemUI: withSystemUI(""), HasGlyph: fakeCover})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := m.Match(context.Background(), "", 400, false, "")
	if len(got) != 1 || got[0].Family != SystemUIFamily {
		t.Errorf("Match = %v", families(got))
	}
}

func TestCoversKnowsTheListedFacesOnly(t *testing.T) {
	m := newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})
	ctx := context.Background()
	file := func(name string) text.FontFile { return text.FontFile{Path: fontDir + name} }
	for _, tt := range []struct {
		name      string
		f         text.FontFile
		r         rune
		has, know bool
	}{
		{"Latin in the system font", text.FontFile{Path: sfns.Path}, 'a', true, true},
		{"Han in a Latin font", file("Menlo.ttc"), '日', false, true},
		{"Han in a Han font", file("PingFang.ttc"), '日', true, true},
		{"a font that is not listed", file("Private/PingFangUI.ttc"), 'a', false, false},
		{"a listed file at another index", text.FontFile{Path: fontDir + "Menlo.ttc", Index: 7}, 'a', false, false},
	} {
		has, known := m.Covers(ctx, tt.f, tt.r)
		if has != tt.has || known != tt.know {
			t.Errorf("%s: Covers = %t, %t; want %t, %t", tt.name, has, known, tt.has, tt.know)
		}
	}
}

func TestFailuresHaveTheirRoutes(t *testing.T) {
	t.Run("no faces", func(t *testing.T) {
		_, err := New(Options{})
		assertRoute(t, err, "DarwinFontsUnavailable", "New.DarwinFontsUnavailable")
	})
	t.Run("a system font that is not there", func(t *testing.T) {
		_, err := New(Options{SystemUI: func(string) (FontFile, bool) { return FontFile{}, false }})
		assertRoute(t, err, "DarwinFontsUnavailable", "New.DarwinFontsUnavailable")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := newMatcher(t, macOS14(), Options{})
	t.Run("Match", func(t *testing.T) {
		_, err := m.Match(ctx, "Menlo", 400, false, "")
		assertRoute(t, err, "DarwinFontsCanceled", "Matcher.Match.DarwinFontsCanceled")
		if appErr, _ := errors.FromError(err); appErr.Metadata["family"] != "Menlo" || !errors.Is(err, context.Canceled) {
			t.Errorf("metadata %v, cause %v", appErr.Metadata, errors.Unwrap(err))
		}
	})
	t.Run("Fallback", func(t *testing.T) {
		_, _, err := m.Fallback(ctx, "Menlo", 400, false, "", 'a')
		assertRoute(t, err, "DarwinFontsCanceled", "Matcher.Fallback.DarwinFontsCanceled")
	})
	t.Run("Aliases", func(t *testing.T) {
		_, err := m.Aliases(ctx, "Menlo")
		assertRoute(t, err, "DarwinFontsCanceled", "Matcher.Aliases.DarwinFontsCanceled")
	})
	t.Run("Families", func(t *testing.T) {
		_, err := m.Families(ctx)
		assertRoute(t, err, "DarwinFontsCanceled", "Matcher.Families.DarwinFontsCanceled")
	})
}

func TestTheMatcherPlugsIntoACollection(t *testing.T) {
	c := text.NewCollection(text.CollectionOpts{Matcher: newMatcher(t, macOS14(), Options{SystemUI: withSystemUI("")})})
	ctx := context.Background()
	got, err := c.Match(ctx, ".AppleSystemUIFont", 400, false, "")
	if err != nil || len(got) == 0 || got[0].Family != SystemUIFamily {
		t.Errorf("Match(.AppleSystemUIFont) = %v, %v", families(got), err)
	}
	// A name that stands for another family keeps only the fonts of that family, which the Aliaser names.
	got, err = c.Match(ctx, "Segoe UI", 400, false, "")
	if err != nil || len(got) == 0 || got[0].Family != SystemUIFamily {
		t.Errorf("Match(Segoe UI) = %v, %v", families(got), err)
	}
	fams, err := c.Families(ctx)
	if err != nil || !slices.Contains(fams, "Menlo") {
		t.Errorf("Families = %v, %v", fams, err)
	}
}
