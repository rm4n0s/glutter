package winfonts

import (
	"context"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

// fixture lists the families a stock Windows 11 has, with the faces that matter to the tests. The
// paths are those of the files; the fake HasGlyph says what each family covers.
func windows11() []Face {
	f := func(family, file string, weight int, italic bool) Face {
		return Face{Family: family, Path: `C:\Windows\Fonts\` + file, Weight: weight, Italic: italic}
	}
	return []Face{
		f("Segoe UI", "segoeui.ttf", 400, false), f("Segoe UI", "segoeuib.ttf", 700, false),
		f("Segoe UI", "segoeuii.ttf", 400, true), f("Segoe UI", "segoeuiz.ttf", 700, true),
		f("Segoe UI", "segoeuil.ttf", 300, false), f("Segoe UI", "seguisb.ttf", 600, false),
		f("Segoe UI Variable Text", "SegUIVar.ttf", 400, false),
		f("Arial", "arial.ttf", 400, false), f("Arial", "arialbd.ttf", 700, false),
		f("Times New Roman", "times.ttf", 400, false), f("Times New Roman", "timesbd.ttf", 700, false),
		f("Consolas", "consola.ttf", 400, false), f("Consolas", "consolab.ttf", 700, false),
		f("Segoe Script", "segoesc.ttf", 400, false),
		f("Segoe UI Emoji", "seguiemj.ttf", 400, false), f("Segoe UI Symbol", "seguisym.ttf", 400, false),
		f("Microsoft YaHei", "msyh.ttc", 400, false), f("Microsoft YaHei", "msyhbd.ttc", 700, false),
		f("Microsoft JhengHei", "msjh.ttc", 400, false),
		f("Yu Gothic", "YuGothR.ttc", 400, false), f("Meiryo", "meiryo.ttc", 400, false),
		f("Malgun Gothic", "malgun.ttf", 400, false),
		f("Nirmala UI", "Nirmala.ttf", 400, false), f("Leelawadee UI", "LeelawUI.ttf", 400, false),
		f("Gadugi", "gadugi.ttf", 400, false), f("Ebrima", "ebrima.ttf", 400, false),
		f("Myanmar Text", "mmrtext.ttf", 400, false), f("Javanese Text", "javatext.ttf", 400, false),
		f("Segoe UI Historic", "seguihis.ttf", 400, false),
		f("MS Gothic", "msgothic.ttc", 400, false), f("SimSun", "simsun.ttc", 400, false),
		f("Tahoma", "tahoma.ttf", 400, false), f("Calibri", "calibri.ttf", 400, false),
	}
}

// windows10 is Windows 11 without the variable system font.
func windows10() []Face {
	return slices.DeleteFunc(windows11(), func(f Face) bool { return f.Family == "Segoe UI Variable Text" })
}

// coverage is what each family has a glyph for, by the file name.
var coverage = map[string]func(r rune) bool{
	"segoeui.ttf":  func(r rune) bool { return r < 0x0600 || r >= 0x2000 && r < 0x2100 }, // Latin, Greek, Cyrillic
	"seguiemj.ttf": func(r rune) bool { return r >= 0x1F300 && r < 0x1FA00 || r == 0x2764 || r == 0x263A },
	"seguisym.ttf": func(r rune) bool { return r >= 0x2190 && r < 0x2600 || r == 0x263A },
	"msyh.ttc":     func(r rune) bool { return r >= 0x4E00 && r < 0xA000 },
	"msjh.ttc":     func(r rune) bool { return r >= 0x4E00 && r < 0xA000 },
	"YuGothR.ttc":  func(r rune) bool { return r >= 0x3040 && r < 0x3100 || r >= 0x4E00 && r < 0xA000 },
	"meiryo.ttc":   func(r rune) bool { return r >= 0x3040 && r < 0x3100 || r >= 0x4E00 && r < 0xA000 },
	"malgun.ttf":   func(r rune) bool { return r >= 0xAC00 && r < 0xD7B0 },
	"Nirmala.ttf":  func(r rune) bool { return r >= 0x0900 && r < 0x0D80 },
	"LeelawUI.ttf": func(r rune) bool { return r >= 0x0E00 && r < 0x0E80 },
	"mmrtext.ttf":  func(r rune) bool { return r >= 0x1000 && r < 0x10A0 },
	"msgothic.ttc": func(r rune) bool { return r >= 0x3040 && r < 0x3100 || r >= 0x4E00 && r < 0xA000 },
	"simsun.ttc":   func(r rune) bool { return r >= 0x4E00 && r < 0xA000 },
	"seguihis.ttf": func(r rune) bool { return r >= 0x10300 && r < 0x10330 },
	"gadugi.ttf":   func(r rune) bool { return r >= 0x13A0 && r < 0x1400 },
	"ebrima.ttf":   func(r rune) bool { return r >= 0x1200 && r < 0x1380 },
	"javatext.ttf": func(r rune) bool { return r >= 0xA980 && r < 0xA9E0 },
}

func hasGlyph(p string, _ int, r rune) bool {
	if fn := coverage[path.Base(winSlash(p))]; fn != nil {
		return fn(r)
	}
	return false
}

func winSlash(p string) string {
	out := []byte(p)
	for i, c := range out {
		if c == '\\' {
			out[i] = '/'
		}
	}
	return string(out)
}

func newMatcher(t *testing.T, faces []Face, locale string) *Matcher {
	t.Helper()
	m, err := New(Options{Faces: faces, Locale: locale, HasGlyph: hasGlyph})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// names are the families of the files, in order.
func names(files []text.FontFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Family
	}
	return out
}

func files(files []text.FontFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = path.Base(winSlash(f.Path))
	}
	return out
}

func TestGenericFamilies(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		faces  []Face
		family string
		want   string
	}{
		{"sans-serif is Segoe UI", windows11(), "sans-serif", "Segoe UI"},
		{"an empty family is sans-serif", windows11(), "", "Segoe UI"},
		{"the case does not matter", windows11(), " Sans-Serif ", "Segoe UI"},
		{"serif is Times New Roman", windows11(), "serif", "Times New Roman"},
		{"monospace is Consolas", windows11(), "monospace", "Consolas"},
		{"cursive is Segoe Script", windows11(), "cursive", "Segoe Script"},
		{"system-ui is the variable Segoe UI on Windows 11", windows11(), "system-ui", "Segoe UI Variable Text"},
		{"system-ui is Segoe UI on Windows 10", windows10(), "system-ui", "Segoe UI"},
		{"emoji is Segoe UI Emoji", windows11(), "emoji", "Segoe UI Emoji"},
		{"fantasy has no Impact here and is Segoe UI", windows11(), "fantasy", "Segoe UI"},
		{"math is Segoe UI Symbol without Cambria Math", windows11(), "math", "Segoe UI Symbol"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMatcher(t, tt.faces, "").Match(ctx, tt.family, 400, false, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) == 0 || got[0].Family != tt.want {
				t.Errorf("Match(%q) starts with %v, want %q", tt.family, names(got), tt.want)
			}
		})
	}
}

func TestGenericFamiliesOnAMachineWithoutTheFontsOfWindows(t *testing.T) {
	// The prefix of Wine: Liberation fonts and little else.
	faces := []Face{
		{Family: "Liberation Sans", Path: "Z:\\lib\\ls.ttf", Weight: 400}, {Family: "Liberation Serif", Path: "Z:\\lib\\lse.ttf", Weight: 400},
		{Family: "Liberation Mono", Path: "Z:\\lib\\lm.ttf", Weight: 400}, {Family: "Noto Sans", Path: "Z:\\lib\\n.ttf", Weight: 400},
	}
	m := newMatcher(t, faces, "")
	ctx := context.Background()
	for family, want := range map[string][]string{
		"sans-serif": {"Liberation Sans"},
		"serif":      {"Liberation Serif", "Liberation Sans"}, // then the stand-in of the chain
		"monospace":  {"Liberation Mono", "Liberation Sans"},
		"cursive":    {"Liberation Sans"}, // no cursive font: the chain has only the stand-in
	} {
		got, err := m.Match(ctx, family, 400, false, "")
		if err != nil || !reflect.DeepEqual(names(got), want) {
			t.Errorf("Match(%q) = %v, %v; want %v", family, names(got), err, want)
		}
	}
	if _, ok, _ := m.Fallback(ctx, "sans-serif", 400, false, "", '日'); ok {
		t.Error("found a font for a character that no font of the chain has")
	}
	// Liberation Sans stands in for the fonts of the chain: it is the font of the characters it has.
	g := func(p string, _ int, r rune) bool { return r == 'ש' || r == 'a' }
	m2, err := New(Options{Faces: faces, HasGlyph: g})
	if err != nil {
		t.Fatal(err)
	}
	if f, ok, _ := m2.Fallback(ctx, "sans-serif", 400, false, "", 'ש'); !ok || f.Family != "Liberation Sans" {
		t.Errorf("Fallback(Hebrew) = %v, %v; want Liberation Sans", f.Family, ok)
	}
}

func TestMatchChoosesTheFaceNearestToTheStyle(t *testing.T) {
	m := newMatcher(t, windows11(), "")
	ctx := context.Background()
	tests := []struct {
		name   string
		family string
		weight int
		italic bool
		want   string
	}{
		{"regular", "Segoe UI", 400, false, "segoeui.ttf"},
		{"bold", "Segoe UI", 700, false, "segoeub.ttf"},
		{"bold italic", "Segoe UI", 700, true, "segoeuiz.ttf"},
		{"italic", "Segoe UI", 400, true, "segoeuii.ttf"},
		{"semibold is its own face", "Segoe UI", 600, false, "seguisb.ttf"},
		{"light is its own face", "Segoe UI", 300, false, "segoeuil.ttf"},
		{"thin falls to the lightest", "Segoe UI", 100, false, "segoeuil.ttf"},
		{"black falls to the boldest", "Segoe UI", 900, false, "segoeuib.ttf"},
		{"a family with one face", "Calibri", 700, true, "calibri.ttf"},
		{"the generic family has the same faces", "sans-serif", 700, false, "segoeub.ttf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Match(ctx, tt.family, tt.weight, tt.italic, "")
			if err != nil {
				t.Fatal(err)
			}
			// segoeub.ttf is not in the fixture: bold is segoeuib.ttf.
			want := tt.want
			if want == "segoeub.ttf" {
				want = "segoeuib.ttf"
			}
			if len(got) == 0 || files(got)[0] != want {
				t.Errorf("Match(%q, %d, %v) starts with %v, want %s", tt.family, tt.weight, tt.italic, files(got), want)
			}
		})
	}
}

func TestMatchOfAFamilyThatIsNotGenericHasOnlyThatFamily(t *testing.T) {
	m := newMatcher(t, windows11(), "")
	ctx := context.Background()
	got, err := m.Match(ctx, "Calibri", 400, false, "ja")
	if err != nil || len(got) != 1 || got[0].Family != "Calibri" {
		t.Errorf("Match(Calibri) = %v, %v", names(got), err)
	}
	// The name does not need its spaces or its case.
	got, _ = m.Match(ctx, "segoeui", 400, false, "")
	if len(got) != 1 || got[0].Family != "Segoe UI" {
		t.Errorf("Match(segoeui) = %v", names(got))
	}
	// A family that is not installed gives none and no error, so the caller goes on to the next family.
	got, err = m.Match(ctx, "Comic Neue", 400, false, "")
	if len(got) != 0 || err != nil {
		t.Errorf("Match(Comic Neue) = %v, %v", names(got), err)
	}
}

func TestMatchOfAnAlias(t *testing.T) {
	m := newMatcher(t, windows11(), "")
	ctx := context.Background()
	tests := []struct {
		family string
		want   string
	}{
		{"Helvetica", "Arial"}, {"helvetica neue", "Arial"}, {"Times", "Times New Roman"},
		{"MS Shell Dlg", "Tahoma"}, {"-apple-system", "Segoe UI"}, {"BlinkMacSystemFont", "Segoe UI"},
	}
	for _, tt := range tests {
		got, _ := m.Match(ctx, tt.family, 400, false, "")
		if len(got) != 1 || got[0].Family != tt.want {
			t.Errorf("Match(%q) = %v, want %s", tt.family, names(got), tt.want)
		}
		al, err := m.Aliases(ctx, tt.family)
		if err != nil || len(al) == 0 || !slices.Contains(al, tt.want) {
			t.Errorf("Aliases(%q) = %v, %v; want %s in it", tt.family, al, err, tt.want)
		}
	}
	if al, _ := m.Aliases(ctx, "Arial"); len(al) != 0 {
		t.Errorf("Aliases(Arial) = %v, want none", al)
	}
	if al, _ := m.Aliases(ctx, "Courier"); len(al) != 0 {
		t.Errorf("Aliases(Courier) = %v: Courier New is not installed in the fixture", al)
	}
}

func TestTheFallbackChainFollowsTheLanguage(t *testing.T) {
	ctx := context.Background()
	specOrder := []string{"Segoe UI", "Segoe UI Emoji", "Segoe UI Symbol", "Microsoft YaHei", "Microsoft JhengHei", "Yu Gothic", "Meiryo", "Malgun Gothic",
		"Nirmala UI", "Leelawadee UI", "Gadugi", "Ebrima", "Myanmar Text", "Javanese Text", "Segoe UI Historic", "MS Gothic", "SimSun",
		"Arial", "Tahoma"} // the stand-ins that the fixture has come last
	tests := []struct {
		name   string
		locale string
		lang   string
		first  []string // the families after the head, in order
	}{
		{"no language keeps the order of the list", "", "", specOrder[1:]},
		{"Simplified Chinese", "", "zh-Hans", append([]string{"Microsoft YaHei", "SimSun"}, without(specOrder[1:], "Microsoft YaHei", "SimSun")...)},
		{"zh-CN is Simplified", "", "zh-CN", append([]string{"Microsoft YaHei", "SimSun"}, without(specOrder[1:], "Microsoft YaHei", "SimSun")...)},
		{"zh alone is Simplified", "", "zh", append([]string{"Microsoft YaHei", "SimSun"}, without(specOrder[1:], "Microsoft YaHei", "SimSun")...)},
		{"Traditional Chinese", "", "zh-Hant-TW", append([]string{"Microsoft JhengHei"}, without(specOrder[1:], "Microsoft JhengHei")...)},
		{"zh-HK is Traditional", "", "zh-HK", append([]string{"Microsoft JhengHei"}, without(specOrder[1:], "Microsoft JhengHei")...)},
		{"Japanese", "", "ja", append([]string{"Yu Gothic", "Meiryo", "MS Gothic"}, without(specOrder[1:], "Yu Gothic", "Meiryo", "MS Gothic")...)},
		{"Korean", "", "ko-KR", append([]string{"Malgun Gothic"}, without(specOrder[1:], "Malgun Gothic")...)},
		{"the user's language when the request has none", "ja-JP", "", append([]string{"Yu Gothic", "Meiryo", "MS Gothic"}, without(specOrder[1:], "Yu Gothic", "Meiryo", "MS Gothic")...)},
		{"the request wins over the user's language", "ja-JP", "ko", append([]string{"Malgun Gothic"}, without(specOrder[1:], "Malgun Gothic")...)},
		{"a language with no special font keeps the order", "", "de-DE", specOrder[1:]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMatcher(t, windows11(), tt.locale).Match(ctx, "sans-serif", 400, false, tt.lang)
			if err != nil {
				t.Fatal(err)
			}
			if names(got)[0] != "Segoe UI" {
				t.Fatalf("starts with %v", names(got))
			}
			if !reflect.DeepEqual(names(got)[1:], tt.first) {
				t.Errorf("chain:\n%v\nwant:\n%v", names(got)[1:], tt.first)
			}
		})
	}
}

// without is the list without the names.
func without(list []string, drop ...string) []string {
	return slices.DeleteFunc(slices.Clone(list), func(s string) bool { return slices.Contains(drop, s) })
}

func TestMatchHasEachFontOnce(t *testing.T) {
	// Two families that are the same file (a collection) are two fonts of the file; one face twice is one.
	faces := append(windows11(), windows11()...)
	got, err := newMatcher(t, faces, "").Match(context.Background(), "sans-serif", 400, false, "")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range got {
		key := f.Path
		if seen[key] {
			t.Errorf("%s twice in %v", key, files(got))
		}
		seen[key] = true
	}
}

func TestFallbackFindsAFontForACharacter(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		family string
		lang   string
		r      rune
		want   string
	}{
		{"an emoji", "Segoe UI", "", '\U0001F600', "Segoe UI Emoji"},
		{"an emoji for a family that is not installed", "Comic Neue", "", '\U0001F600', "Segoe UI Emoji"},
		{"a heart is an emoji font glyph before the symbols", "Segoe UI", "", '\u2764', "Segoe UI Emoji"},
		{"a symbol", "Segoe UI", "", '\u2192', "Segoe UI Symbol"},
		{"a Chinese character", "Segoe UI", "", '漢', "Microsoft YaHei"},
		{"a Chinese character in Traditional Chinese", "Segoe UI", "zh-TW", '漢', "Microsoft JhengHei"},
		{"a Chinese character in Japanese", "Segoe UI", "ja", '漢', "Yu Gothic"},
		{"hiragana", "Segoe UI", "", 'あ', "Yu Gothic"},
		{"hangul", "Segoe UI", "", '한', "Malgun Gothic"},
		{"devanagari", "Segoe UI", "", 'अ', "Nirmala UI"},
		{"thai", "Segoe UI", "", 'ก', "Leelawadee UI"},
		{"myanmar", "Segoe UI", "", 'က', "Myanmar Text"},
		{"javanese", "Segoe UI", "", '\uA984', "Javanese Text"},
		{"cherokee", "Segoe UI", "", 'Ꭰ', "Gadugi"},
		{"ethiopic", "Segoe UI", "", 'አ', "Ebrima"},
		{"old italic", "Segoe UI", "", '\U00010300', "Segoe UI Historic"},
		{"a Latin letter the family itself has", "Segoe UI", "", 'a', "Segoe UI"},
	}
	m := newMatcher(t, windows11(), "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, ok, err := m.Fallback(ctx, tt.family, 400, false, tt.lang, tt.r)
			if err != nil || !ok || f.Family != tt.want {
				t.Errorf("Fallback(%q) = %v, %v, %v; want %s", tt.r, f.Family, ok, err, tt.want)
			}
		})
	}
	if _, ok, _ := m.Fallback(ctx, "Segoe UI", 400, false, "", '\U0001E000'); ok {
		t.Error("a character that no font has was found")
	}
}

func TestFamiliesAreSortedAndComplete(t *testing.T) {
	got, err := newMatcher(t, windows11(), "").Families(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Arial", "Consolas", "Segoe UI", "Segoe UI Variable Text", "Tahoma"} {
		if !slices.Contains(got, want) {
			t.Errorf("%s is not listed in %v", want, got)
		}
	}
	if got[0] != "Arial" {
		t.Errorf("the first family is %q, want Arial", got[0])
	}
	if !slices.IsSortedFunc(got, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }) {
		t.Errorf("families are not sorted: %v", got)
	}
}

func TestFailuresHaveTheirRoutes(t *testing.T) {
	assertRoute := func(t *testing.T, err error, tag, route string) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if appErr.Tag != tag || !appErr.HasRoute(route) {
			t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
		}
	}
	t.Run("no faces", func(t *testing.T) {
		_, err := New(Options{})
		assertRoute(t, err, "WindowsFontsUnavailable", "New.WindowsFontsUnavailable")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := newMatcher(t, windows11(), "")
	t.Run("Match", func(t *testing.T) {
		_, err := m.Match(ctx, "Arial", 400, false, "")
		assertRoute(t, err, "WindowsFontsCanceled", "Matcher.Match.WindowsFontsCanceled")
		if appErr, _ := errors.FromError(err); appErr.Metadata["family"] != "Arial" || !errors.Is(err, context.Canceled) {
			t.Errorf("metadata %v, cause %v", appErr.Metadata, errors.Unwrap(err))
		}
	})
	t.Run("Fallback", func(t *testing.T) {
		_, _, err := m.Fallback(ctx, "Arial", 400, false, "", 'a')
		assertRoute(t, err, "WindowsFontsCanceled", "Matcher.Fallback.WindowsFontsCanceled")
	})
	t.Run("Aliases", func(t *testing.T) {
		_, err := m.Aliases(ctx, "Arial")
		assertRoute(t, err, "WindowsFontsCanceled", "Matcher.Aliases.WindowsFontsCanceled")
	})
	t.Run("Families", func(t *testing.T) {
		_, err := m.Families(ctx)
		assertRoute(t, err, "WindowsFontsCanceled", "Matcher.Families.WindowsFontsCanceled")
	})
	t.Run("a font file that is not there", func(t *testing.T) {
		_, err := Describe(t.TempDir() + "/missing.ttf")
		assertRoute(t, err, "WindowsFontUnreadable", "Describe.WindowsFontUnreadable")
	})
}

func TestTheMatcherPlugsIntoACollection(t *testing.T) {
	c := text.NewCollection(text.CollectionOpts{Matcher: newMatcher(t, windows10(), "")})
	ctx := context.Background()
	got, err := c.Match(ctx, "sans-serif", 400, false, "")
	if err != nil || len(got) == 0 || got[0].Family != "Segoe UI" {
		t.Errorf("Match(sans-serif) = %v, %v", names(got), err)
	}
	// A name that stands for another family keeps only the fonts of that family, which the Aliaser names.
	got, err = c.Match(ctx, "Helvetica", 400, false, "")
	if err != nil || len(got) != 1 || got[0].Family != "Arial" {
		t.Errorf("Match(Helvetica) = %v, %v", names(got), err)
	}
	got, err = c.Match(ctx, "Calibri", 700, false, "")
	if err != nil || len(got) != 1 || got[0].Family != "Calibri" {
		t.Errorf("Match(Calibri) = %v, %v", names(got), err)
	}
	fams, err := c.Families(ctx)
	if err != nil || !slices.Contains(fams, "Segoe UI") {
		t.Errorf("Families = %v, %v", fams, err)
	}
}
