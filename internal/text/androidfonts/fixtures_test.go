package androidfonts

import (
	"io/fs"
	"log/slog"
	"strconv"
	"strings"
	"testing"
)

// fontsAPI26 is the fonts.xml of Android 8 (version 22): static Roboto, the CJK collection listed once per
// language with its index, the emoji font with no language, a fallbackFor serif font.
const fontsAPI26 = `<?xml version="1.0" encoding="utf-8"?>
<!-- fixture in the format of /system/etc/fonts.xml of API 26 -->
<familyset version="22">
    <family name="sans-serif">
        <font weight="100" style="normal">Roboto-Thin.ttf</font>
        <font weight="100" style="italic">Roboto-ThinItalic.ttf</font>
        <font weight="300" style="normal">Roboto-Light.ttf</font>
        <font weight="400" style="normal">Roboto-Regular.ttf</font>
        <font weight="400" style="italic">Roboto-Italic.ttf</font>
        <font weight="500" style="normal">Roboto-Medium.ttf</font>
        <font weight="700" style="normal">Roboto-Bold.ttf</font>
        <font weight="700" style="italic">Roboto-BoldItalic.ttf</font>
        <font weight="900" style="normal">Roboto-Black.ttf</font>
    </family>
    <alias name="sans-serif-thin" to="sans-serif" weight="100" />
    <alias name="sans-serif-light" to="sans-serif" weight="300" />
    <alias name="sans-serif-medium" to="sans-serif" weight="500" />
    <alias name="sans-serif-black" to="sans-serif" weight="900" />
    <alias name="arial" to="sans-serif" />
    <alias name="Helvetica" to="sans-serif" />
    <alias name="tahoma" to="sans-serif" />
    <family name="sans-serif-condensed">
        <font weight="300" style="normal">RobotoCondensed-Light.ttf</font>
        <font weight="400" style="normal">RobotoCondensed-Regular.ttf</font>
        <font weight="700" style="normal">RobotoCondensed-Bold.ttf</font>
    </family>
    <family name="serif">
        <font weight="400" style="normal">NotoSerif-Regular.ttf</font>
        <font weight="400" style="italic">NotoSerif-Italic.ttf</font>
        <font weight="700" style="normal">NotoSerif-Bold.ttf</font>
        <font weight="700" style="italic">NotoSerif-BoldItalic.ttf</font>
    </family>
    <alias name="times" to="serif" />
    <alias name="times new roman" to="serif" />
    <family name="monospace">
        <font weight="400" style="normal">DroidSansMono.ttf</font>
    </family>
    <alias name="courier" to="monospace" />
    <family name="cursive">
        <font weight="400" style="normal">DancingScript-Regular.ttf</font>
        <font weight="700" style="normal">DancingScript-Bold.ttf</font>
    </family>
    <family name="casual">
        <font weight="400" style="normal">ComingSoon.ttf</font>
    </family>
    <family>
        <font weight="400" style="normal">NotoSansSymbols-Regular-Subsetted.ttf</font>
    </family>
    <family lang="und-Arab" variant="elegant">
        <font weight="400" style="normal">NotoNaskhArabic-Regular.ttf</font>
        <font weight="700" style="normal">NotoNaskhArabic-Bold.ttf</font>
    </family>
    <family lang="und-Arab" variant="compact">
        <font weight="400" style="normal">NotoNaskhArabicUI-Regular.ttf</font>
        <font weight="700" style="normal">NotoNaskhArabicUI-Bold.ttf</font>
    </family>
    <family lang="und-Hebr">
        <font weight="400" style="normal">NotoSansHebrew-Regular.ttf</font>
        <font weight="700" style="normal">NotoSansHebrew-Bold.ttf</font>
    </family>
    <family lang="zh-Hans">
        <font weight="400" style="normal" index="2">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="2" fallbackFor="serif">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="zh-Hant,zh-Bopo">
        <font weight="400" style="normal" index="3">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="3" fallbackFor="serif">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="ja">
        <font weight="400" style="normal" index="0">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="0" fallbackFor="serif">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="ko">
        <font weight="400" style="normal" index="1">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="1" fallbackFor="serif">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family>
        <font weight="400" style="normal">NotoColorEmoji.ttf</font>
    </family>
</familyset>
`

// fontsAPI28 is the format of Android 9 and 10: postScriptName on the fonts, the languages of a family
// separated by blanks, a family-list, the emoji font with its language, a font with an unknown style.
const fontsAPI28 = `<?xml version="1.0" encoding="utf-8"?>
<familyset version="23">
    <family name="sans-serif">
        <font weight="400" style="normal" postScriptName="Roboto-Regular">Roboto-Regular.ttf</font>
        <font weight="700" style="normal" postScriptName="Roboto-Bold">Roboto-Bold.ttf</font>
    </family>
    <alias name="arial" to="sans-serif" />
    <family-list name="Droid Family">
        <family lang="und-Thai" variant="elegant">
            <font>
                NotoSansThai-Regular.ttf
            </font>
        </family>
    </family-list>
    <family lang="und-Thai" variant="compact">
        <font weight="400" style="normal">NotoSansThaiUI-Regular.ttf</font>
    </family>
    <family lang="zh-Hant zh-Bopo">
        <font weight="400" style="normal" index="3">NotoSansCJK-Regular.ttc</font>
    </family>
    <family lang="und-Zsye">
        <font weight="400" style="normal">NotoColorEmoji.ttf</font>
    </family>
    <unknown-element><family name="ignored"><font>Ignored.ttf</font></family></unknown-element>
</familyset>
`

// fontsAPI31 is the format of Android 12 and later: the variable Roboto, listed once per weight with its
// axes, the weight aliases, the COLRv1 emoji font and the CJK fonts with fallbackFor.
const fontsAPI31 = `<?xml version="1.0" encoding="utf-8"?>
<familyset version="23">
    <family name="sans-serif">
        <font weight="100" style="normal">Roboto-Regular.ttf
            <axis tag="wdth" stylevalue="100" />
            <axis tag="wght" stylevalue="100" />
        </font>
        <font weight="400" style="normal">Roboto-Regular.ttf
            <axis tag="wdth" stylevalue="100" />
            <axis tag="wght" stylevalue="400" />
        </font>
        <font weight="700" style="normal">Roboto-Regular.ttf
            <axis tag="wdth" stylevalue="100" />
            <axis tag="wght" stylevalue="700" />
        </font>
        <font weight="400" style="italic">Roboto-Italic.ttf
            <axis tag="wdth" stylevalue="100" />
            <axis tag="wght" stylevalue="400" />
        </font>
        <font weight="700" style="italic">Roboto-Italic.ttf
            <axis tag="wdth" stylevalue="100" />
            <axis tag="wght" stylevalue="700" />
        </font>
    </family>
    <alias name="sans-serif-thin" to="sans-serif" weight="100" />
    <alias name="arial" to="sans-serif" />
    <family name="serif">
        <font weight="400" style="normal">NotoSerif-Regular.ttf</font>
        <font weight="700" style="normal">NotoSerif-Bold.ttf</font>
    </family>
    <family name="monospace">
        <font weight="400" style="normal">DroidSansMono.ttf</font>
    </family>
    <family lang="zh-Hans">
        <font weight="400" style="normal" index="2" postScriptName="NotoSansCJKsc-Regular">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="2" fallbackFor="serif" postScriptName="NotoSerifCJKsc-Regular">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="zh-Hant,zh-Bopo">
        <font weight="400" style="normal" index="3" postScriptName="NotoSansCJKtc-Regular">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="3" fallbackFor="serif" postScriptName="NotoSerifCJKtc-Regular">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="ja">
        <font weight="400" style="normal" index="0" postScriptName="NotoSansCJKjp-Regular">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="0" fallbackFor="serif" postScriptName="NotoSerifCJKjp-Regular">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="ko">
        <font weight="400" style="normal" index="1" postScriptName="NotoSansCJKkr-Regular">NotoSansCJK-Regular.ttc</font>
        <font weight="400" style="normal" index="1" fallbackFor="serif" postScriptName="NotoSerifCJKkr-Regular">NotoSerifCJK-Regular.ttc</font>
    </family>
    <family lang="und-Zsye">
        <font weight="400" style="normal">NotoColorEmojiCompat.ttf</font>
    </family>
</familyset>
`

// customizationXML is the fonts_customization.xml of a device builder.
const customizationXML = `<?xml version="1.0" encoding="utf-8"?>
<fonts-modification version="1">
    <family customizationType="new-named-family" name="brand-sans">
        <font weight="400" style="normal">BrandSans-Regular.ttf</font>
        <font weight="700" style="normal">BrandSans-Bold.ttf</font>
    </family>
    <alias name="brand" to="brand-sans" />
    <alias name="sans-serif-bold" to="sans-serif" weight="700" />
</fonts-modification>
`

// fakeSystem is a device: configuration files and font files by path.
type fakeSystem struct {
	files map[string]string
	fonts map[string]bool
	// glyphs are the runes that each "path#index" has.
	glyphs map[string]string
}

func (f *fakeSystem) readFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(data), nil
}

func (f *fakeSystem) exists(path string) bool { return f.fonts[path] }

func (f *fakeSystem) hasGlyph(path string, index int, r rune) bool {
	return strings.ContainsRune(f.glyphs[path+"#"+strconv.Itoa(index)], r)
}

// withFonts makes every path of the list exist under dir.
func (f *fakeSystem) withFonts(dir string, names ...string) *fakeSystem {
	if f.fonts == nil {
		f.fonts = map[string]bool{}
	}
	for _, n := range names {
		f.fonts[dir+"/"+n] = true
	}
	return f
}

func (f *fakeSystem) options() Options {
	o := DefaultOptions()
	o.ReadFile = f.readFile
	o.FileExists = f.exists
	o.HasGlyph = f.hasGlyph
	o.Logger = slog.New(slog.NewTextHandler(testWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return o
}

// testWriter discards the log.
type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

// allFontsOf lists the font file names of a fixture.
func allFontsOf(t *testing.T, xml string) []string {
	t.Helper()
	var families []*fontFamily
	if _, err := parseConfig(slog.New(slog.NewTextHandler(testWriter{}, nil)), "fixture.xml", []byte(xml), &families, "/system/fonts", false); err != nil {
		t.Fatalf("fixture does not parse: %v", err)
	}
	var out []string
	var visit func(f *fontFamily)
	visit = func(f *fontFamily) {
		for _, ft := range f.Fonts {
			out = append(out, ft.FileName)
		}
		for _, fb := range f.fallbackFamilies {
			visit(fb)
		}
	}
	for _, f := range families {
		visit(f)
	}
	return out
}

// device makes a fake device with the fixture as /system/etc/fonts.xml and all its fonts installed.
func device(t *testing.T, xml string) *fakeSystem {
	t.Helper()
	d := &fakeSystem{files: map[string]string{SystemFontsXML: xml}}
	return d.withFonts("/system/fonts", allFontsOf(t, xml)...)
}
