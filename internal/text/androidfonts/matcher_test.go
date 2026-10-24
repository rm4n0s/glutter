package androidfonts

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/text"
)

func mustNew(t *testing.T, d *fakeSystem) *Matcher {
	t.Helper()
	m, err := New(d.options())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// brief writes a font as its file name, with #index when the index is not 0.
func brief(f text.FontFile) string {
	s := path.Base(f.Path)
	if f.Index != 0 {
		s += fmt.Sprintf("#%d", f.Index)
	}
	return s
}

func briefs(files []text.FontFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = brief(f)
	}
	return out
}

type matchCase struct {
	name   string
	family string
	weight int
	italic bool
	lang   string
	// prefix are the first fonts of the answer; wantLen is the number of fonts, -1 when it is not checked.
	prefix  []string
	wantLen int
	// family0 is the Family of the first font, "" when it is not checked.
	family0 string
}

func runMatchCases(t *testing.T, m *Matcher, tests []matchCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := m.Match(context.Background(), tt.family, tt.weight, tt.italic, tt.lang)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			got := briefs(files)
			if len(got) < len(tt.prefix) || !slices.Equal(got[:len(tt.prefix)], tt.prefix) {
				t.Errorf("Match(%q, %d, %t, %q) = %v\n want it to start with %v", tt.family, tt.weight, tt.italic, tt.lang, got, tt.prefix)
			}
			if tt.wantLen >= 0 && len(got) != tt.wantLen {
				t.Errorf("Match(%q) gave %d fonts %v, want %d", tt.family, len(got), got, tt.wantLen)
			}
			if tt.family0 != "" && len(files) > 0 && files[0].Family != tt.family0 {
				t.Errorf("Family of the first font = %q, want %q", files[0].Family, tt.family0)
			}
			seen := map[string]bool{}
			for _, f := range files {
				key := fmt.Sprintf("%s#%d", f.Path, f.Index)
				if seen[key] {
					t.Errorf("%s is in the answer twice", brief(f))
				}
				seen[key] = true
			}
		})
	}
}

// api26Chain is Match("sans-serif", 400, false, "") on the API 26 device: the family, then the families with no
// name that have no fallbackFor (the elegant ones before the compact), the other named families, and the fallback
// families for serif.
var api26Chain = []string{
	"Roboto-Regular.ttf",
	"NotoSansSymbols-Regular-Subsetted.ttf", "NotoNaskhArabic-Regular.ttf", "NotoSansHebrew-Regular.ttf",
	"NotoSansCJK-Regular.ttc#2", "NotoSansCJK-Regular.ttc#3", "NotoSansCJK-Regular.ttc", "NotoSansCJK-Regular.ttc#1",
	"NotoColorEmoji.ttf",
	"Roboto-Thin.ttf", "Roboto-Light.ttf", "Roboto-Medium.ttf", "Roboto-Black.ttf",
	"RobotoCondensed-Regular.ttf", "NotoSerif-Regular.ttf", "DroidSansMono.ttf", "DancingScript-Regular.ttf", "ComingSoon.ttf",
	"NotoSerifCJK-Regular.ttc#2", "NotoSerifCJK-Regular.ttc#3", "NotoSerifCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc#1",
	"NotoNaskhArabicUI-Regular.ttf",
}

func TestMatchAPI26(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI26))
	tests := []matchCase{
		{name: "sans-serif, the whole chain", family: "sans-serif", weight: 400, prefix: api26Chain, wantLen: len(api26Chain), family0: "sans-serif"},
		{name: "sans-serif bold", family: "sans-serif", weight: 700, prefix: []string{"Roboto-Bold.ttf", "NotoSansSymbols-Regular-Subsetted.ttf", "NotoNaskhArabic-Bold.ttf"}, wantLen: -1},
		{name: "sans-serif bold italic", family: "sans-serif", weight: 700, italic: true, prefix: []string{"Roboto-BoldItalic.ttf"}, wantLen: -1},
		{name: "sans-serif italic", family: "sans-serif", weight: 400, italic: true, prefix: []string{"Roboto-Italic.ttf"}, wantLen: -1},
		{name: "sans-serif light", family: "sans-serif", weight: 300, prefix: []string{"Roboto-Light.ttf"}, wantLen: -1},
		{name: "sans-serif 600 is bold", family: "sans-serif", weight: 600, prefix: []string{"Roboto-Bold.ttf"}, wantLen: -1},
		{name: "sans-serif 200 is thin", family: "sans-serif", weight: 200, prefix: []string{"Roboto-Thin.ttf"}, wantLen: -1},
		{name: "empty family is sans-serif", family: "", weight: 400, prefix: api26Chain, wantLen: len(api26Chain)},
		{name: "case and blanks of the name do not matter", family: " SANS-Serif ", weight: 400, prefix: api26Chain, wantLen: len(api26Chain)},
		{name: "alias arial", family: "arial", weight: 700, prefix: []string{"Roboto-Bold.ttf", "NotoSansSymbols-Regular-Subsetted.ttf"}, wantLen: -1, family0: "sans-serif"},
		{name: "alias with capitals in the file", family: "HELVETICA", weight: 400, prefix: []string{"Roboto-Regular.ttf"}, wantLen: -1, family0: "sans-serif"},
		{name: "serif", family: "serif", weight: 400, prefix: []string{"NotoSerif-Regular.ttf", "NotoSerifCJK-Regular.ttc#2", "NotoSerifCJK-Regular.ttc#3", "NotoSerifCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc#1", "NotoSansSymbols-Regular-Subsetted.ttf"}, wantLen: len(api26Chain), family0: "serif"},
		{name: "serif bold italic", family: "serif", weight: 700, italic: true, prefix: []string{"NotoSerif-BoldItalic.ttf"}, wantLen: -1},
		{name: "alias times new roman", family: "Times New Roman", weight: 400, prefix: []string{"NotoSerif-Regular.ttf"}, wantLen: -1, family0: "serif"},
		{name: "serif puts the serif CJK fonts before the sans ones", family: "serif", weight: 400, lang: "ja", prefix: []string{"NotoSerif-Regular.ttf", "NotoSerifCJK-Regular.ttc", "NotoSansCJK-Regular.ttc"}, wantLen: -1},
		{name: "monospace", family: "monospace", weight: 400, prefix: []string{"DroidSansMono.ttf"}, wantLen: -1, family0: "monospace"},
		{name: "alias courier", family: "courier", weight: 400, prefix: []string{"DroidSansMono.ttf"}, wantLen: -1, family0: "monospace"},
		{name: "cursive bold", family: "cursive", weight: 700, prefix: []string{"DancingScript-Bold.ttf"}, wantLen: -1},
		{name: "condensed bold", family: "sans-serif-condensed", weight: 700, prefix: []string{"RobotoCondensed-Bold.ttf"}, wantLen: -1, family0: "sans-serif-condensed"},
		{name: "weight alias", family: "sans-serif-medium", weight: 400, prefix: []string{"Roboto-Medium.ttf"}, wantLen: -1, family0: "sans-serif-medium"},
		{name: "fantasy is casual", family: "fantasy", weight: 400, prefix: []string{"ComingSoon.ttf"}, wantLen: -1},
		{name: "math is sans-serif", family: "math", weight: 400, prefix: []string{"Roboto-Regular.ttf"}, wantLen: -1},
		{name: "emoji puts the emoji font first", family: "emoji", weight: 400, prefix: []string{"NotoColorEmoji.ttf", "Roboto-Regular.ttf"}, wantLen: len(api26Chain)},
		{name: "unknown family", family: "Comic Sans", weight: 400, wantLen: 0},
		// The CJK fonts: one collection listed once per language, with the index of the language.
		{name: "ja", family: "sans-serif", weight: 400, lang: "ja", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc"}, wantLen: len(api26Chain)},
		{name: "ja-JP", family: "sans-serif", weight: 400, lang: "ja-JP", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc"}, wantLen: -1},
		{name: "ko", family: "sans-serif", weight: 400, lang: "ko", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#1", "NotoSerifCJK-Regular.ttc#1"}, wantLen: -1},
		{name: "ko-KR", family: "sans-serif", weight: 400, lang: "ko-KR", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#1"}, wantLen: -1},
		{name: "zh-Hans", family: "sans-serif", weight: 400, lang: "zh-Hans", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#2", "NotoSerifCJK-Regular.ttc#2"}, wantLen: -1},
		{name: "zh-CN is simplified", family: "sans-serif", weight: 400, lang: "zh-CN", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#2"}, wantLen: -1},
		{name: "zh-Hant", family: "sans-serif", weight: 400, lang: "zh-Hant", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#3", "NotoSerifCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "zh-TW is traditional", family: "sans-serif", weight: 400, lang: "zh-TW", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "zh-HK is traditional", family: "sans-serif", weight: 400, lang: "zh-HK", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "zh-Bopo is traditional", family: "sans-serif", weight: 400, lang: "zh-Bopo", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "zh alone is the first, simplified", family: "sans-serif", weight: 400, lang: "zh", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#2", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "underscore tag", family: "sans-serif", weight: 400, lang: "zh_TW", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "en-US keeps the order of the file", family: "sans-serif", weight: 400, lang: "en-US", prefix: api26Chain, wantLen: len(api26Chain)},
		{name: "a script puts its fonts first", family: "sans-serif", weight: 400, lang: "he", prefix: []string{"Roboto-Regular.ttf", "NotoSansHebrew-Regular.ttf", "NotoSansSymbols-Regular-Subsetted.ttf"}, wantLen: -1},
		{name: "the elegant variant comes before the compact one", family: "sans-serif", weight: 400, lang: "ar", prefix: []string{"Roboto-Regular.ttf", "NotoNaskhArabic-Regular.ttf", "NotoSansSymbols-Regular-Subsetted.ttf"}, wantLen: -1},
	}
	runMatchCases(t, m, tests)
}

func TestMatchAPI31(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI31))
	chain := []string{
		"Roboto-Regular.ttf", // the variable font of every weight
		"NotoSansCJK-Regular.ttc#2", "NotoSansCJK-Regular.ttc#3", "NotoSansCJK-Regular.ttc", "NotoSansCJK-Regular.ttc#1",
		"NotoColorEmojiCompat.ttf",
		"NotoSerif-Bold.ttf", "DroidSansMono.ttf",
		"NotoSerifCJK-Regular.ttc#2", "NotoSerifCJK-Regular.ttc#3", "NotoSerifCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc#1",
	}
	tests := []matchCase{
		{name: "the variable font is one font, whatever the weight", family: "sans-serif", weight: 700, prefix: chain, wantLen: len(chain), family0: "sans-serif"},
		{name: "italic", family: "sans-serif", weight: 400, italic: true, prefix: []string{"Roboto-Italic.ttf", "NotoSansCJK-Regular.ttc#2"}, wantLen: len(chain) + 1},
		{name: "arial", family: "arial", weight: 400, prefix: []string{"Roboto-Regular.ttf"}, wantLen: -1, family0: "sans-serif"},
		{name: "weight alias of the variable font", family: "sans-serif-thin", weight: 400, prefix: []string{"Roboto-Regular.ttf"}, wantLen: -1, family0: "sans-serif-thin"},
		{name: "ja, sans", family: "sans-serif", weight: 400, lang: "ja", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc", "NotoSerifCJK-Regular.ttc"}, wantLen: -1},
		{name: "ja, serif", family: "serif", weight: 400, lang: "ja", prefix: []string{"NotoSerif-Regular.ttf", "NotoSerifCJK-Regular.ttc", "NotoSansCJK-Regular.ttc"}, wantLen: -1},
		{name: "ko, serif", family: "serif", weight: 400, lang: "ko", prefix: []string{"NotoSerif-Regular.ttf", "NotoSerifCJK-Regular.ttc#1", "NotoSansCJK-Regular.ttc#1"}, wantLen: -1},
		{name: "zh-Hant-TW, serif", family: "serif", weight: 400, lang: "zh-Hant-TW", prefix: []string{"NotoSerif-Regular.ttf", "NotoSerifCJK-Regular.ttc#3", "NotoSansCJK-Regular.ttc#3"}, wantLen: -1},
		{name: "emoji", family: "emoji", weight: 400, prefix: []string{"NotoColorEmojiCompat.ttf", "Roboto-Regular.ttf"}, wantLen: len(chain)},
		{name: "monospace", family: "monospace", weight: 400, prefix: []string{"DroidSansMono.ttf"}, wantLen: -1},
		{name: "the locale of the options when the request has no language", family: "sans-serif", weight: 400, lang: "", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#2"}, wantLen: -1},
	}
	runMatchCases(t, m, tests)
}

func TestMatchLocaleOption(t *testing.T) {
	d := device(t, fontsAPI31)
	o := d.options()
	o.Locale = "ja-JP"
	m, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	runMatchCases(t, m, []matchCase{
		{name: "the locale decides when the request has no language", family: "sans-serif", weight: 400, prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc"}, wantLen: -1},
		{name: "the request decides over the locale", family: "sans-serif", weight: 400, lang: "ko", prefix: []string{"Roboto-Regular.ttf", "NotoSansCJK-Regular.ttc#1"}, wantLen: -1},
	})
}

func TestMatchAPI28(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI28))
	chain := []string{
		"Roboto-Regular.ttf",
		"NotoSansCJK-Regular.ttc#3", "NotoColorEmoji.ttf",
		"NotoSansThai-Regular.ttf", // the family-list family has a name: it comes with the named ones
		"NotoSansThaiUI-Regular.ttf",
	}
	tests := []matchCase{
		{name: "sans-serif", family: "sans-serif", weight: 400, prefix: chain, wantLen: len(chain)},
		{name: "th puts the Thai fonts first, the elegant before the compact", family: "sans-serif", weight: 400, lang: "th", prefix: []string{"Roboto-Regular.ttf", "NotoSansThai-Regular.ttf", "NotoSansCJK-Regular.ttc#3", "NotoColorEmoji.ttf", "NotoSansThaiUI-Regular.ttf"}, wantLen: -1},
		{name: "the family of a family-list", family: "Droid Family", weight: 400, prefix: []string{"NotoSansThai-Regular.ttf"}, wantLen: -1, family0: "droid family"},
		{name: "a file name with blanks around it", family: "droid family", weight: 400, prefix: []string{"NotoSansThai-Regular.ttf"}, wantLen: -1},
		{name: "bold", family: "sans-serif", weight: 800, prefix: []string{"Roboto-Bold.ttf"}, wantLen: -1},
		{name: "emoji with a language", family: "emoji", weight: 400, prefix: []string{"NotoColorEmoji.ttf"}, wantLen: len(chain)},
		{name: "ignored element", family: "ignored", weight: 400, wantLen: 0},
	}
	runMatchCases(t, m, tests)
}

func TestMatchMissingFiles(t *testing.T) {
	d := device(t, fontsAPI26)
	delete(d.fonts, "/system/fonts/Roboto-Bold.ttf")
	delete(d.fonts, "/system/fonts/Roboto-BoldItalic.ttf")
	delete(d.fonts, "/system/fonts/NotoSerif-Regular.ttf")
	delete(d.fonts, "/system/fonts/NotoSerif-Italic.ttf")
	delete(d.fonts, "/system/fonts/NotoSerif-Bold.ttf")
	delete(d.fonts, "/system/fonts/NotoSerif-BoldItalic.ttf")
	m := mustNew(t, d)
	runMatchCases(t, m, []matchCase{
		{name: "bold falls back to the nearest font that exists", family: "sans-serif", weight: 700, prefix: []string{"Roboto-Black.ttf"}, wantLen: -1},
		{name: "a generic family with no font that exists is the default family", family: "serif", weight: 400, prefix: []string{"Roboto-Regular.ttf", "NotoSerifCJK-Regular.ttc#2"}, wantLen: -1},
		{name: "the alias of that family neither", family: "times", weight: 400, wantLen: 0},
	})
	families, err := m.Families(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(families, "serif") || slices.Contains(families, "times") {
		t.Errorf("Families lists serif or times: %v", families)
	}
	// The serif CJK fonts are still the fallback of the serif family, because their files exist.
	files, err := m.Match(context.Background(), "sans-serif", 400, false, "ja")
	if err != nil {
		t.Fatal(err)
	}
	if got := briefs(files); !slices.Contains(got, "NotoSerifCJK-Regular.ttc") {
		t.Errorf("the serif CJK fonts are not in %v", got)
	}
}

func TestMatchCustomization(t *testing.T) {
	d := device(t, fontsAPI31)
	d.files["/product/etc/fonts_customization.xml"] = customizationXML
	d.files["/vendor/etc/fonts_customization.xml"] = `<fonts-modification version="1"><family name="vendor-mono"><font>VendorMono.ttf</font></family></fonts-modification>`
	d.files["/oem/etc/fonts_customization.xml"] = `<broken`
	d.withFonts("/product/fonts", "BrandSans-Regular.ttf", "BrandSans-Bold.ttf")
	d.withFonts("/vendor/fonts", "VendorMono.ttf")
	m := mustNew(t, d)
	tests := []struct {
		family string
		weight int
		want   string // the path of the first font
	}{
		{"brand-sans", 700, "/product/fonts/BrandSans-Bold.ttf"},
		{"brand", 400, "/product/fonts/BrandSans-Regular.ttf"},
		{"vendor-mono", 400, "/vendor/fonts/VendorMono.ttf"},
		{"sans-serif-bold", 400, "/system/fonts/Roboto-Regular.ttf"},
		{"sans-serif", 400, "/system/fonts/Roboto-Regular.ttf"},
	}
	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			files, err := m.Match(context.Background(), tt.family, tt.weight, false, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(files) == 0 || files[0].Path != tt.want {
				t.Errorf("Match(%q, %d) starts with %v, want %s", tt.family, tt.weight, files, tt.want)
			}
		})
	}
}

func TestAliasesAndFamilies(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI26))
	aliases := []struct {
		family string
		want   []string
	}{
		{"arial", []string{"sans-serif", "helvetica", "tahoma"}},
		{"Arial", []string{"sans-serif", "helvetica", "tahoma"}},
		{"sans-serif", []string{"arial", "helvetica", "tahoma"}},
		{"times new roman", []string{"serif", "times"}},
		{"monospace", []string{"courier"}},
		{"cursive", nil},
		{"sans-serif-thin", nil},
		{"unknown", nil},
		{"", nil},
	}
	for _, tt := range aliases {
		got, err := m.Aliases(context.Background(), tt.family)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("Aliases(%q) = %q, want %q", tt.family, got, tt.want)
		}
	}
	got, err := m.Families(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"sans-serif", "arial", "helvetica", "tahoma", "sans-serif-thin", "sans-serif-light", "sans-serif-medium", "sans-serif-black",
		"sans-serif-condensed", "serif", "times", "times new roman", "monospace", "courier", "cursive", "casual",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Families = %q\n want %q", got, want)
	}
}

func TestFallback(t *testing.T) {
	// Every CJK index has the kana and the Han characters, as the Noto Sans CJK collection does; the emoji
	// font has the emoji; the Latin fonts have Latin.
	glyphs := map[string]string{}
	const latin = "abcABC"
	for _, f := range []string{"Roboto-Regular", "Roboto-Bold", "NotoSerif-Regular"} {
		glyphs["/system/fonts/"+f+".ttf#0"] = latin
	}
	for i := 0; i < 4; i++ {
		glyphs[fmt.Sprintf("/system/fonts/NotoSansCJK-Regular.ttc#%d", i)] = "あ骨한" + latin
		glyphs[fmt.Sprintf("/system/fonts/NotoSerifCJK-Regular.ttc#%d", i)] = "あ骨한"
	}
	glyphs["/system/fonts/NotoColorEmoji.ttf#0"] = "\U0001F600❤"
	glyphs["/system/fonts/NotoSansSymbols-Regular-Subsetted.ttf#0"] = "❤☃"
	glyphs["/system/fonts/NotoNaskhArabic-Regular.ttf#0"] = "ب"
	glyphs["/system/fonts/NotoNaskhArabic-Bold.ttf#0"] = "ب"
	glyphs["/system/fonts/NotoNaskhArabicUI-Regular.ttf#0"] = "بج"
	glyphs["/system/fonts/NotoSansHebrew-Regular.ttf#0"] = "א"

	d := device(t, fontsAPI26)
	d.glyphs = glyphs
	m := mustNew(t, d)
	tests := []struct {
		name   string
		family string
		weight int
		italic bool
		lang   string
		r      rune
		want   string // brief of the font, "" for none
	}{
		{"emoji", "sans-serif", 400, false, "", '\U0001F600', "NotoColorEmoji.ttf"},
		{"emoji with the Japanese locale", "sans-serif", 400, false, "ja-JP", '\U0001F600', "NotoColorEmoji.ttf"},
		{"a character that the symbols font has before the emoji font", "sans-serif", 400, false, "", '❤', "NotoSansSymbols-Regular-Subsetted.ttf"},
		{"Latin is in the family first", "sans-serif", 400, false, "", 'a', "Roboto-Regular.ttf"},
		{"Latin of a bold request", "sans-serif", 700, false, "", 'a', "Roboto-Bold.ttf"},
		{"Latin of serif", "serif", 400, false, "", 'a', "NotoSerif-Regular.ttf"},
		{"Hebrew", "sans-serif", 400, false, "", 'א', "NotoSansHebrew-Regular.ttf"},
		{"Arabic, the elegant font", "sans-serif", 400, false, "ar", 'ب', "NotoNaskhArabic-Regular.ttf"},
		{"Arabic bold", "sans-serif", 700, false, "ar", 'ب', "NotoNaskhArabic-Bold.ttf"},
		{"Arabic, only the compact font has the character", "sans-serif", 400, false, "ar", 'ج', "NotoNaskhArabicUI-Regular.ttf"},
		{"Han without a language is the first CJK font", "sans-serif", 400, false, "", '骨', "NotoSansCJK-Regular.ttc#2"},
		{"Han, ja", "sans-serif", 400, false, "ja", '骨', "NotoSansCJK-Regular.ttc"},
		{"Han, ja-JP", "sans-serif", 400, false, "ja-JP", '骨', "NotoSansCJK-Regular.ttc"},
		{"Han, ko", "sans-serif", 400, false, "ko", '骨', "NotoSansCJK-Regular.ttc#1"},
		{"Han, zh-Hans", "sans-serif", 400, false, "zh-Hans", '骨', "NotoSansCJK-Regular.ttc#2"},
		{"Han, zh-CN", "sans-serif", 400, false, "zh-CN", '骨', "NotoSansCJK-Regular.ttc#2"},
		{"Han, zh-Hant", "sans-serif", 400, false, "zh-Hant", '骨', "NotoSansCJK-Regular.ttc#3"},
		{"Han, zh-TW", "sans-serif", 400, false, "zh-TW", '骨', "NotoSansCJK-Regular.ttc#3"},
		{"Han in a serif family, ja: the serif CJK font", "serif", 400, false, "ja", '骨', "NotoSerifCJK-Regular.ttc"},
		{"Han in a serif family, ko", "serif", 400, false, "ko", '骨', "NotoSerifCJK-Regular.ttc#1"},
		{"Han in an unknown family, ja: the sans font", "Unknown Family", 400, false, "ja", '骨', "NotoSansCJK-Regular.ttc"},
		{"Hangul, a language that does not fit: the character decides", "sans-serif", 400, false, "ja", '한', "NotoSansCJK-Regular.ttc"},
		{"nobody has it", "sans-serif", 400, false, "", '\U0001F9FF', ""},
		{"the family emoji", "emoji", 400, false, "", '\U0001F600', "NotoColorEmoji.ttf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, ok, err := m.Fallback(context.Background(), tt.family, tt.weight, tt.italic, tt.lang, tt.r)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				if tt.want != "" {
					t.Errorf("Fallback(%q, %q, %U) found nothing, want %s", tt.family, tt.lang, tt.r, tt.want)
				}
				return
			}
			if got := brief(f); got != tt.want {
				t.Errorf("Fallback(%q, %q, %U) = %s, want %s", tt.family, tt.lang, tt.r, got, tt.want)
			}
		})
	}
}

func TestNewErrors(t *testing.T) {
	const noFamilies = `<familyset version="22"></familyset>`
	tests := []struct {
		name      string
		files     map[string]string
		fonts     []string
		wantTag   string
		wantRoute string
	}{
		{"no file", map[string]string{}, nil, TagUnavailable, "New.AndroidFontsUnavailable"},
		{"a file with no family", map[string]string{SystemFontsXML: noFamilies}, nil, TagUnavailable, "New.AndroidFontsUnavailable"},
		{"no font file exists", map[string]string{SystemFontsXML: fontsAPI31}, nil, TagUnavailable, "New.AndroidFontsUnavailable"},
		{"malformed XML", map[string]string{SystemFontsXML: `<familyset><family></familyset>`}, nil, TagXMLInvalid, "New->parseConfig.AndroidFontsXMLInvalid"},
		{"an entity", map[string]string{SystemFontsXML: `<!DOCTYPE a [<!ENTITY x "y">]><familyset/>`}, nil, TagXMLInvalid, "New->parseConfig.AndroidFontsXMLInvalid"},
		{"only the customization file", map[string]string{"/product/etc/fonts_customization.xml": customizationXML}, []string{"BrandSans-Regular.ttf"}, TagUnavailable, "New.AndroidFontsUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &fakeSystem{files: tt.files}
			d.withFonts("/system/fonts", tt.fonts...)
			d.withFonts("/product/fonts", tt.fonts...)
			m, err := New(d.options())
			if err == nil {
				t.Fatalf("New succeeded: %v", m)
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag || !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("tag %s, route %s; want %s", appErr.Tag, appErr.Route(), tt.wantRoute)
			}
		})
	}
}

func TestNewSkipsFilesWithProblems(t *testing.T) {
	good := `<familyset version="22"><family name="sans-serif"><font>A.ttf</font></family></familyset>`
	tests := []struct {
		name  string
		files []string // FontsXML, in order
	}{
		{"the first file is missing", []string{"/missing.xml", "/good.xml"}},
		{"the first file is malformed", []string{"/bad.xml", "/good.xml"}},
		{"the first file has no family", []string{"/empty.xml", "/good.xml"}},
		{"the last file is malformed", []string{"/good.xml", "/bad.xml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &fakeSystem{files: map[string]string{
				"/good.xml": good, "/bad.xml": "<familyset", "/empty.xml": `<familyset version="22"/>`,
			}}
			d.withFonts("/system/fonts", "A.ttf")
			o := d.options()
			o.FontsXML = tt.files
			m, err := New(o)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			files, err := m.Match(context.Background(), "sans-serif", 400, false, "")
			if err != nil || len(files) != 1 || brief(files[0]) != "A.ttf" {
				t.Errorf("Match = %v, %v", files, err)
			}
		})
	}
}

func TestFilesExtendEachOther(t *testing.T) {
	d := &fakeSystem{files: map[string]string{
		"/a.xml": `<familyset version="22"><family name="sans-serif"><font>A.ttf</font></family></familyset>`,
		"/b.xml": `<familyset version="22"><alias name="Arial" to="sans-serif"/><family name="serif"><font>B.ttf</font></family></familyset>`,
	}}
	d.withFonts("/system/fonts", "A.ttf", "B.ttf")
	o := d.options()
	o.FontsXML = []string{"/a.xml", "/b.xml"}
	m, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	runMatchCases(t, m, []matchCase{
		{name: "alias of the second file to the first", family: "arial", weight: 400, prefix: []string{"A.ttf", "B.ttf"}, wantLen: 2},
		{name: "family of the second file", family: "serif", weight: 400, prefix: []string{"B.ttf", "A.ttf"}, wantLen: 2},
	})
}

func TestCanceled(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI26))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := []struct {
		name      string
		call      func() error
		wantRoute string
	}{
		{"Match", func() error { _, err := m.Match(ctx, "sans-serif", 400, false, ""); return err }, "Matcher.Match.AndroidFontsCanceled"},
		{"Fallback", func() error { _, _, err := m.Fallback(ctx, "sans-serif", 400, false, "", 'a'); return err }, "Matcher.Fallback.AndroidFontsCanceled"},
		{"Aliases", func() error { _, err := m.Aliases(ctx, "arial"); return err }, "Matcher.Aliases.AndroidFontsCanceled"},
		{"Families", func() error { _, err := m.Families(ctx); return err }, "Matcher.Families.AndroidFontsCanceled"},
	}
	for _, tt := range calls {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != "AndroidFontsCanceled" || !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("tag %s, route %s; want %s", appErr.Tag, appErr.Route(), tt.wantRoute)
			}
			if !errors.Is(err, context.Canceled) {
				t.Errorf("the error does not wrap context.Canceled")
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	o := DefaultOptions()
	if !slices.Equal(o.FontsXML, []string{"/system/etc/fonts.xml"}) {
		t.Errorf("FontsXML = %v", o.FontsXML)
	}
	wantCustom := []string{"/oem/etc/fonts_customization.xml", "/product/etc/fonts_customization.xml", "/vendor/etc/fonts_customization.xml"}
	if !slices.Equal(o.Customization, wantCustom) {
		t.Errorf("Customization = %v", o.Customization)
	}
	if o.FontDir != "/system/fonts" {
		t.Errorf("FontDir = %q", o.FontDir)
	}
	o.Customization[0] = "changed"
	if DefaultOptions().Customization[0] == "changed" {
		t.Error("DefaultOptions shares its slice")
	}
	if got := (Options{}).withDefaults(); got.FontDir != "/system/fonts" || got.ReadFile == nil || got.FileExists == nil || got.HasGlyph == nil || got.Logger == nil {
		t.Errorf("withDefaults left a field empty: %+v", got)
	}
}

func TestCustomizationDir(t *testing.T) {
	tests := []struct{ file, want string }{
		{"/product/etc/fonts_customization.xml", "/product/fonts"},
		{"/vendor/etc/fonts_customization.xml", "/vendor/fonts"},
		{"/oem/etc/fonts_customization.xml", "/oem/fonts"},
		{"/data/test/fonts_customization.xml", "/system/fonts"},
		{"/etc/fonts_customization.xml", "/system/fonts"},
		{"fonts_customization.xml", "/system/fonts"},
	}
	for _, tt := range tests {
		if got := customizationDir(tt.file, "/system/fonts"); got != tt.want {
			t.Errorf("customizationDir(%q) = %q, want %q", tt.file, got, tt.want)
		}
	}
}

func TestIsEmoji(t *testing.T) {
	m := mustNew(t, device(t, fontsAPI26))
	var emoji []string
	for _, s := range m.sets {
		if s.isEmoji() {
			emoji = append(emoji, path.Base(s.fonts[0].path))
		}
	}
	if want := []string{"NotoColorEmoji.ttf"}; !slices.Equal(emoji, want) {
		t.Errorf("emoji families = %v, want %v", emoji, want)
	}
	if got := strings.Join(m.byName["sans-serif"].names, ","); got != "sans-serif,arial,helvetica,tahoma" {
		t.Errorf("names = %s", got)
	}
}
