//go:build darwin && !ios

package engine

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
)

// These tests read the real fonts of the Mac through Core Text (SPEC §29.8): the system font, a Han character
// and an emoji. They need no window.

func systemFonts(t *testing.T) (*text.Collection, text.Matcher) {
	t.Helper()
	m := SystemFontMatcher(context.Background(), quiet())
	if m == nil {
		t.Fatal("SystemFontMatcher gave no matcher")
	}
	return text.NewCollection(text.CollectionOpts{Matcher: m}), m
}

func TestTheDefaultFamilyIsTheSystemFont(t *testing.T) {
	c, _ := systemFonts(t)
	ctx := context.Background()
	for _, name := range []string{".AppleSystemUIFont", "system-ui", "sans-serif"} {
		got, err := c.Match(ctx, name, 400, false, "")
		if err != nil || len(got) == 0 {
			t.Fatalf("Match(%q) = %v, %v", name, got, err)
		}
		if !strings.HasSuffix(got[0].Path, "SFNS.ttf") && !strings.Contains(got[0].Path, "SF") {
			t.Errorf("Match(%q) starts with %+v, want San Francisco", name, got[0])
		}
		if _, err := os.Stat(got[0].Path); err != nil {
			t.Errorf("Match(%q): %v", name, err)
		}
		face, err := c.Face(ctx, got[0])
		if err != nil {
			t.Fatalf("Face(%+v): %v", got[0], err)
		}
		if gid, ok := face.NominalGlyph('A'); !ok || gid == 0 {
			t.Errorf("the system font has no glyph for A")
		}
	}
	for _, name := range []string{"Helvetica Neue", "Menlo", "Times New Roman"} {
		got, err := c.Match(ctx, name, 400, false, "")
		if err != nil || len(got) != 1 || got[0].Family != name {
			t.Errorf("Match(%q) = %+v, %v", name, got, err)
		}
	}
}

func TestAHanCharacterAndAnEmojiFindFonts(t *testing.T) {
	c, m := systemFonts(t)
	fb, ok := m.(text.Fallbacker)
	if !ok {
		t.Fatal("the matcher is not a Fallbacker")
	}
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		r    rune
		lang string
	}{{"han", '汉', "zh-Hans"}, {"han with no language", '汉', ""}, {"kana", 'あ', "ja"}, {"hangul", '한', "ko"}, {"emoji", 0x1F600, ""}, {"arrow", '→', ""}} {
		f, ok, err := fb.Fallback(ctx, "", 400, false, tt.lang, tt.r)
		if err != nil || !ok {
			t.Errorf("%s: Fallback = %+v, %v, %v", tt.name, f, ok, err)
			continue
		}
		face, err := c.Face(ctx, f)
		if err != nil {
			t.Errorf("%s: Face(%+v): %v", tt.name, f, err)
			continue
		}
		if gid, ok := face.NominalGlyph(tt.r); !ok || gid == 0 {
			t.Errorf("%s: %+v has no glyph for %U", tt.name, f, tt.r)
		}
		if tt.name == "emoji" && len(face.BitmapSizes()) == 0 {
			t.Errorf("the emoji font %+v has no bitmap strikes (sbix)", f)
		}
	}
}
