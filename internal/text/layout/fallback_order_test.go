package layout

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// fontconfigLike is a matcher that answers like fontconfig: the font of the family when the
// system has one, then every other font, and for a family it does not know the default sans
// font first. Arial stands for Go Mono (an alias).
type fontconfigLike struct {
	sans, mono text.FontFile
}

func (m fontconfigLike) Match(_ context.Context, family string, _ int, _ bool, _ string) ([]text.FontFile, error) {
	switch family {
	case "monospace", "Go Mono":
		return []text.FontFile{m.mono, m.sans}, nil
	default: // sans-serif, and a family that is not installed
		return []text.FontFile{m.sans, m.mono}, nil
	}
}

func (m fontconfigLike) Aliases(_ context.Context, family string) ([]string, error) {
	if family == "Arial" {
		return []string{"Go Mono"}, nil
	}
	return nil, nil
}

func systemFontFiles(t *testing.T) fontconfigLike {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	return fontconfigLike{
		sans: text.FontFile{Path: write("GoRegular.ttf", goregular.TTF), Family: "Go Regular"},
		mono: text.FontFile{Path: write("GoMono.ttf", gomono.TTF), Family: "Go Mono"},
	}
}

func TestFallbackOrder(t *testing.T) {
	sys := systemFontFiles(t)
	tests := []struct {
		name     string
		family   string
		fallback []string
		// want is the file whose face draws a Latin letter.
		want text.FontFile
	}{
		{"a missing family with monospace as the fallback", "Missing", []string{"monospace"}, sys.mono},
		{"a missing family and nothing else uses the default", "Missing", nil, sys.sans},
		{"missing entries are skipped, the first family that has a font wins", "Missing", []string{"Also Missing", "monospace"}, sys.mono},
		{"the first family that has a font wins over the later ones", "Go Mono", []string{"sans-serif"}, sys.mono},
		{"a system family before the generic one", "Go Regular", []string{"monospace"}, sys.sans},
		{"an entry of an installed family", "Missing", []string{"Go Mono", "sans-serif"}, sys.mono},
		{"an alias", "Arial", nil, sys.mono},
		{"a family listed after sans-serif is still before the default", "Missing", []string{"Go Mono"}, sys.mono},
		{"no family uses the default", "", nil, sys.sans},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := text.NewCollection(text.CollectionOpts{Matcher: sys})
			if err := fonts.Register(c); err != nil {
				t.Fatal(err)
			}
			e := NewEngine(context.Background(), c, EngineOpts{OnFontError: func(err error) { t.Errorf("font error: %v", err) }})
			var families []string
			if tt.family != "" {
				families = append(families, tt.family)
			}
			families = append(families, tt.fallback...)
			got := e.fontsFor(families, ui.FontWeightW400, false, ui.Locale{}, nil).ResolveFace('a')
			want, err := c.Face(context.Background(), tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.Font != want.Font {
				t.Errorf("the letter is drawn with another font than %s", tt.want.Family)
			}
		})
	}
}

// TestFallbackOrderInAParagraph checks the same through a paragraph's width: Go Mono's
// advance is the same for every letter.
func TestFallbackOrderInAParagraph(t *testing.T) {
	sys := systemFontFiles(t)
	c := text.NewCollection(text.CollectionOpts{Matcher: sys})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(context.Background(), c)
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Missing", FontFamilyFallback: []string{"monospace"}, FontSize: ui.Ptr(20.0)})
	i, w := widthIn(e, "iiii", style), widthIn(e, "WWWW", style)
	if i != w {
		t.Errorf("width of iiii = %v, of WWWW = %v: the text is not monospace", i, w)
	}
}

// TestMissingFamilyFallsBackToMonospaceOverFontscan is the same over the font scan, which
// has no fontconfig to put the default font first but finds the default font for a missing family.
func TestMissingFamilyFallsBackToMonospaceOverFontscan(t *testing.T) {
	e := systemEngine(t, "iW")
	lineWidth := func(s string, fallback ...string) float64 {
		style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Glutter No Such Family", FontFamilyFallback: fallback, FontSize: ui.Ptr(20.0)})
		return widthIn(e, s, style)
	}
	if i, w := lineWidth("iiiiiiii", "monospace"), lineWidth("WWWWWWWW", "monospace"); i != w {
		t.Errorf("with the monospace fallback the text is not monospace: iiiiiiii is %v wide, WWWWWWWW is %v", i, w)
	}
}
