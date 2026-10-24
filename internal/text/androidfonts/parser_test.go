package androidfonts

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(testWriter{}, nil)) }

func parse(t *testing.T, xml, base string, isFallback bool) ([]*fontFamily, int) {
	t.Helper()
	var families []*fontFamily
	version, err := parseConfig(quietLogger(), "fixture.xml", []byte(xml), &families, base, isFallback)
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	return families, version
}

// describe writes a family and its fonts on one line.
func describe(f *fontFamily) string {
	var fonts []string
	for _, ft := range f.Fonts {
		style := [...]string{"auto", "normal", "italic"}[ft.Style]
		s := fmt.Sprintf("%s/%d/%d/%s", ft.FileName, ft.Index, ft.Weight, style)
		for _, a := range ft.Axes {
			s += fmt.Sprintf("/%s=%g", a.Tag, a.Value)
		}
		if ft.PostScriptName != "" {
			s += "/ps=" + ft.PostScriptName
		}
		fonts = append(fonts, s)
	}
	variant := map[fontVariant]string{variantDefault: "default", variantCompact: "compact", variantElegant: "elegant"}[f.Variant]
	return fmt.Sprintf("names=%s langs=%s variant=%s fallback=%t for=%q fonts=%s",
		strings.Join(f.Names, ","), strings.Join(f.Languages, ","), variant, f.IsFallbackFont, f.FallbackFor, strings.Join(fonts, ";"))
}

func describeAll(families []*fontFamily) []string {
	var out []string
	for _, f := range families {
		out = append(out, describe(f))
		for _, fb := range f.fallbackFamilies {
			out = append(out, "  "+describe(fb))
		}
	}
	return out
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		isFallback  bool
		wantVersion int
		want        []string
	}{
		{
			name: "API 28: lang with blanks, family-list, postScriptName, a file name with blanks, an unknown element", xml: fontsAPI28, wantVersion: 23,
			want: []string{
				`names=sans-serif,arial langs= variant=default fallback=false for="" fonts=Roboto-Regular.ttf/0/400/normal/ps=Roboto-Regular;Roboto-Bold.ttf/0/700/normal/ps=Roboto-Bold`,
				`names=droid family langs=und-Thai variant=elegant fallback=false for="" fonts=NotoSansThai-Regular.ttf/0/0/auto`,
				`names= langs=und-Thai variant=compact fallback=true for="" fonts=NotoSansThaiUI-Regular.ttf/0/400/normal`,
				`names= langs=zh-Hant,zh-Bopo variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/3/400/normal`,
				`names= langs=und-Zsye variant=default fallback=true for="" fonts=NotoColorEmoji.ttf/0/400/normal`,
			},
		},
		{
			name: "API 31: axes, a weight alias, fallbackFor families next to their family", xml: fontsAPI31, wantVersion: 23,
			want: []string{
				`names=sans-serif,arial langs= variant=default fallback=false for="" fonts=Roboto-Regular.ttf/0/100/normal/wdth=100/wght=100;Roboto-Regular.ttf/0/400/normal/wdth=100/wght=400;Roboto-Regular.ttf/0/700/normal/wdth=100/wght=700;Roboto-Italic.ttf/0/400/italic/wdth=100/wght=400;Roboto-Italic.ttf/0/700/italic/wdth=100/wght=700`,
				`names=sans-serif-thin langs= variant=default fallback=false for="" fonts=Roboto-Regular.ttf/0/100/normal/wdth=100/wght=100`,
				`names=serif langs= variant=default fallback=false for="" fonts=NotoSerif-Regular.ttf/0/400/normal;NotoSerif-Bold.ttf/0/700/normal`,
				`names=monospace langs= variant=default fallback=false for="" fonts=DroidSansMono.ttf/0/400/normal`,
				`names= langs=zh-Hans variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/2/400/normal/ps=NotoSansCJKsc-Regular`,
				`  names= langs=zh-Hans variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/2/400/normal/ps=NotoSerifCJKsc-Regular`,
				`names= langs=zh-Hant,zh-Bopo variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/3/400/normal/ps=NotoSansCJKtc-Regular`,
				`  names= langs=zh-Hant,zh-Bopo variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/3/400/normal/ps=NotoSerifCJKtc-Regular`,
				`names= langs=ja variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/0/400/normal/ps=NotoSansCJKjp-Regular`,
				`  names= langs=ja variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/0/400/normal/ps=NotoSerifCJKjp-Regular`,
				`names= langs=ko variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/1/400/normal/ps=NotoSansCJKkr-Regular`,
				`  names= langs=ko variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/1/400/normal/ps=NotoSerifCJKkr-Regular`,
				`names= langs=und-Zsye variant=default fallback=true for="" fonts=NotoColorEmojiCompat.ttf/0/400/normal`,
			},
		},
		{
			name: "fonts-modification (fonts_customization.xml) is the format of version 21", wantVersion: 21,
			xml: customizationXML,
			want: []string{
				`names=brand-sans,brand langs= variant=default fallback=false for="" fonts=BrandSans-Regular.ttf/0/400/normal;BrandSans-Bold.ttf/0/700/normal`,
			},
			// "sans-serif-bold" has no target in this file alone: it is skipped
		},
		{
			name: "the aliases and the case of names", wantVersion: 22,
			xml: `<familyset version="22">
				<family name="Sans-Serif"><font weight="400">A.ttf</font><font weight="700" style="italic" index="2">B.ttc</font></family>
				<alias name="ARIAL" to="SANS-serif"/>
				<alias name="bold" to="sans-serif" weight="700"/>
				<alias name="missing" to="nothing"/>
			</familyset>`,
			want: []string{
				`names=sans-serif,arial langs= variant=default fallback=false for="" fonts=A.ttf/0/400/auto;B.ttc/2/700/italic`,
				`names=bold langs= variant=default fallback=false for="" fonts=B.ttc/2/700/italic`,
			},
		},
		{
			name: "invalid values are ignored with a warning", wantVersion: 0,
			xml: `<familyset version="x">
				<family name="a" variant="weird" lang="  ja   ko ">
					<font weight="bold" index="-1" style="oblique">A.ttf<axis tag="wght" stylevalue="1.5x"/><axis tag="wghtt" stylevalue="2"/><axis tag="wdth" stylevalue="-.5"/><axis tag="wdth" stylevalue="3"/></font>
				</family>
			</familyset>`,
			want: []string{
				`names=a langs=ja,ko variant=default fallback=false for="" fonts=A.ttf/0/0/auto/wdth=-0.5`,
			},
		},
		{
			name: "the fallback flag of the file and a family that is not at the right place", wantVersion: 22, isFallback: true,
			xml: `<familyset version="22">
				<font>Stray.ttf</font>
				<family><family name="inner"><font>Inner.ttf</font></family><font>Outer.ttf</font></family>
				<alias name="alias2" to="a" weight="400"/>
				<family name="a"><font weight="400">A.ttf</font></family>
				<alias name="alias2" to="a" weight="400"/>
			</familyset>`,
			want: []string{
				`names= langs= variant=default fallback=true for="" fonts=Outer.ttf/0/0/auto`,
				`names=a langs= variant=default fallback=false for="" fonts=A.ttf/0/400/auto`,
				`names=alias2 langs= variant=default fallback=true for="" fonts=A.ttf/0/400/auto`, // a weight alias in a fallback file is a fallback family
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			families, version := parse(t, tt.xml, "/system/fonts", tt.isFallback)
			if version != tt.wantVersion {
				t.Errorf("version = %d, want %d", version, tt.wantVersion)
			}
			got := describeAll(families)
			if !slices.Equal(got, tt.want) {
				t.Errorf("families:\n got  %s\n want %s", strings.Join(got, "\n      "), strings.Join(tt.want, "\n      "))
			}
		})
	}
}

// TestParseConfigAPI26 checks the parts of the largest fixture that the matcher relies on.
func TestParseConfigAPI26(t *testing.T) {
	families, version := parse(t, fontsAPI26, "/system/fonts", false)
	if version != 22 {
		t.Errorf("version = %d, want 22", version)
	}
	byNames := map[string]*fontFamily{}
	for _, f := range families {
		byNames[strings.Join(f.Names, ",")] = f
	}
	tests := []struct {
		names string
		want  string
	}{
		{"sans-serif,arial,helvetica,tahoma", `names=sans-serif,arial,helvetica,tahoma langs= variant=default fallback=false for="" fonts=Roboto-Thin.ttf/0/100/normal;Roboto-ThinItalic.ttf/0/100/italic;Roboto-Light.ttf/0/300/normal;Roboto-Regular.ttf/0/400/normal;Roboto-Italic.ttf/0/400/italic;Roboto-Medium.ttf/0/500/normal;Roboto-Bold.ttf/0/700/normal;Roboto-BoldItalic.ttf/0/700/italic;Roboto-Black.ttf/0/900/normal`},
		{"sans-serif-thin", `names=sans-serif-thin langs= variant=default fallback=false for="" fonts=Roboto-Thin.ttf/0/100/normal;Roboto-ThinItalic.ttf/0/100/italic`},
		{"sans-serif-medium", `names=sans-serif-medium langs= variant=default fallback=false for="" fonts=Roboto-Medium.ttf/0/500/normal`},
		{"serif,times,times new roman", `names=serif,times,times new roman langs= variant=default fallback=false for="" fonts=NotoSerif-Regular.ttf/0/400/normal;NotoSerif-Italic.ttf/0/400/italic;NotoSerif-Bold.ttf/0/700/normal;NotoSerif-BoldItalic.ttf/0/700/italic`},
		{"monospace,courier", `names=monospace,courier langs= variant=default fallback=false for="" fonts=DroidSansMono.ttf/0/400/normal`},
		{"", `names= langs=und-Arab variant=elegant fallback=true for="" fonts=NotoNaskhArabic-Regular.ttf/0/400/normal;NotoNaskhArabic-Bold.ttf/0/700/normal`}, // the first family with no name
	}
	for _, tt := range tests {
		f := byNames[tt.names]
		if tt.names == "" {
			for _, c := range families {
				if len(c.Names) == 0 && c.Languages != nil {
					f = c
					break
				}
			}
		}
		if f == nil {
			t.Errorf("no family %q", tt.names)
			continue
		}
		if got := describe(f); got != tt.want {
			t.Errorf("family %q:\n got  %s\n want %s", tt.names, got, tt.want)
		}
	}
	// The CJK collection is listed once per language, with a family for the serif fonts next to it.
	var cjk []string
	for _, f := range families {
		if slices.Contains(f.Languages, "ja") || slices.Contains(f.Languages, "zh-Hant") {
			cjk = append(cjk, describe(f))
			for _, fb := range f.fallbackFamilies {
				cjk = append(cjk, describe(fb))
			}
		}
	}
	wantCJK := []string{
		`names= langs=zh-Hant,zh-Bopo variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/3/400/normal`,
		`names= langs=zh-Hant,zh-Bopo variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/3/400/normal`,
		`names= langs=ja variant=default fallback=true for="" fonts=NotoSansCJK-Regular.ttc/0/400/normal`,
		`names= langs=ja variant=default fallback=true for="serif" fonts=NotoSerifCJK-Regular.ttc/0/400/normal`,
	}
	if !slices.Equal(cjk, wantCJK) {
		t.Errorf("CJK families:\n got  %s\n want %s", strings.Join(cjk, "\n      "), strings.Join(wantCJK, "\n      "))
	}
}

// TestParseConfigAppends checks that a later file sees the families of the earlier ones: an alias of the
// customization file names a family of fonts.xml.
func TestParseConfigAppends(t *testing.T) {
	var families []*fontFamily
	if _, err := parseConfig(quietLogger(), "fonts.xml", []byte(fontsAPI31), &families, "/system/fonts", false); err != nil {
		t.Fatal(err)
	}
	n := len(families)
	if _, err := parseConfig(quietLogger(), "fonts_customization.xml", []byte(customizationXML), &families, "/product/fonts", false); err != nil {
		t.Fatal(err)
	}
	got := describeAll(families[n:])
	want := []string{
		`names=brand-sans,brand langs= variant=default fallback=false for="" fonts=BrandSans-Regular.ttf/0/400/normal;BrandSans-Bold.ttf/0/700/normal`,
		`names=sans-serif-bold langs= variant=default fallback=false for="" fonts=Roboto-Regular.ttf/0/700/normal/wdth=100/wght=700;Roboto-Italic.ttf/0/700/italic/wdth=100/wght=700`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("customization families:\n got  %s\n want %s", strings.Join(got, "\n      "), strings.Join(want, "\n      "))
	}
	if base := families[n].BasePath; base != "/product/fonts" {
		t.Errorf("base path = %q, want /product/fonts", base)
	}
}

func TestParseConfigErrors(t *testing.T) {
	tests := []struct {
		name      string
		xml       string
		wantRoute string
	}{
		{"element not closed", `<familyset version="22"><family name="a"><font>A.ttf</font></familyset>`, "parseConfig.AndroidFontsXMLInvalid"},
		{"text only", `not xml at all <`, "parseConfig.AndroidFontsXMLInvalid"},
		{"truncated tag", `<familyset version="22"><family name="a"`, "parseConfig.AndroidFontsXMLInvalid"},
		{"empty file", ``, "parseConfig.AndroidFontsXMLInvalid"},
		{"no element", `<?xml version="1.0"?>`, "parseConfig.AndroidFontsXMLInvalid"},
		{"entity declaration", `<?xml version="1.0"?><!DOCTYPE familyset [<!ENTITY a "b">]><familyset version="22"><family name="a"><font>A.ttf</font></family></familyset>`, "parseConfig.AndroidFontsXMLInvalid"},
		{"unknown entity", `<familyset version="22"><family name="a"><font>&unknown;</font></family></familyset>`, "parseConfig.AndroidFontsXMLInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var families []*fontFamily
			_, err := parseConfig(quietLogger(), "fonts.xml", []byte(tt.xml), &families, "/system/fonts", false)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != TagXMLInvalid || !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("tag %s, route %s; want %s", appErr.Tag, appErr.Route(), tt.wantRoute)
			}
			if appErr.Metadata["file"] != "fonts.xml" {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
			if len(families) != 0 {
				t.Errorf("a file with an error added %d families", len(families))
			}
		})
	}
}

func TestParseNumbers(t *testing.T) {
	ints := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true}, {"400", 400, true}, {"007", 7, true}, {"2147483647", 2147483647, true},
		{"2147483648", 0, false}, {"", 0, false}, {"-1", 0, false}, {"1.5", 0, false}, {" 1", 0, false}, {"+1", 0, false}, {"1e3", 0, false},
	}
	for _, tt := range ints {
		got, ok := parseNonNegativeInt(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseNonNegativeInt(%q) = %d, %t; want %d, %t", tt.in, got, ok, tt.want, tt.ok)
		}
	}
	fixed := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"100", 100, true}, {"0", 0, true}, {"1.5", 1.5, true}, {".5", 0.5, true}, {"-.5", -0.5, true}, {"-12.25", -12.25, true},
		{"32767", 32767, true}, {"32768", 0, false}, {"1.", 0, false}, {"", 0, false}, {"-", 0, false}, {"1e3", 0, false},
		{"1.2.3", 0, false}, {"a", 0, false}, {"0.000001", 0, true}, {"0.3", 0.2999877929688, true},
	}
	for _, tt := range fixed {
		got, ok := parseFixed16(tt.in)
		if ok != tt.ok || (ok && (got-tt.want > 1e-9 || tt.want-got > 1e-9)) {
			t.Errorf("parseFixed16(%q) = %v, %t; want %v, %t", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestStringHelpers(t *testing.T) {
	if got := asciiLower("Sans-SERIF Ä"); got != "sans-serif Ä" {
		t.Errorf("asciiLower = %q", got)
	}
	if got := asciiLower("already"); got != "already" {
		t.Errorf("asciiLower = %q", got)
	}
	langs := []struct {
		in   string
		want []string
	}{
		{"ja", []string{"ja"}}, {"  zh-Hant,zh-Bopo ", []string{"zh-Hant", "zh-Bopo"}}, {"a\tb\nc d", []string{"a", "b", "c", "d"}}, {"", nil}, {" ,, ", nil},
	}
	for _, tt := range langs {
		if got := splitLanguages(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("splitLanguages(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
