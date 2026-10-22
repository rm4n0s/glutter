//go:build linux

package engine

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

func TestCollectionOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := openFontconfig(ctx)
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	defer m.Close()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})

	tests := []struct {
		name   string
		family string
		weight int
		italic bool
		lang   string
	}{
		{"sans-serif", "sans-serif", 400, false, ""},
		{"bold serif", "serif", 700, false, ""},
		{"japanese", "sans-serif", 400, false, "ja"},
		{"monospace italic", "monospace", 400, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := c.Match(ctx, tt.family, tt.weight, tt.italic, tt.lang)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("Match returned no font file")
			}
			// The first files load, including faces of collections (.ttc).
			for _, f := range files[:min(len(files), 5)] {
				if _, err := c.Face(ctx, f); err != nil {
					t.Errorf("Face(%+v): %v", f, err)
				}
			}
		})
	}
}

func TestEngineLaysOutMixedScriptsOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := openFontconfig(ctx)
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	defer m.Close()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	var fontErrors []error
	e := layout.NewEngine(ctx, c, layout.EngineOpts{OnFontError: func(err error) { fontErrors = append(fontErrors, err) }})
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0)})
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionLTR}))
	b.PushStyle(style)
	b.AddText("Hello שלום مرحبا 日本語 world")
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(200))
	if len(fontErrors) != 0 {
		t.Errorf("font errors: %v", fontErrors)
	}
	if p.Height() <= 0 || p.LongestLine() <= 0 || p.LongestLine() > 200+1e-6 {
		t.Errorf("height %v, longest line %v", p.Height(), p.LongestLine())
	}
	if p.MaxIntrinsicWidth() < p.MinIntrinsicWidth() {
		t.Errorf("max %v < min %v", p.MaxIntrinsicWidth(), p.MinIntrinsicWidth())
	}
}

// TestMissingFamilyFallsBackToMonospaceOverFontconfig is the case that fontconfig's own order got
// wrong: for a family that is not installed it lists the default sans font first, so a
// FontFamilyFallback of monospace never came first.
func TestMissingFamilyFallsBackToMonospaceOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := openFontconfig(ctx)
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	defer m.Close()
	e := layout.NewEngine(ctx, text.NewCollection(text.CollectionOpts{Matcher: m}))
	lineWidth := func(s string, fallback ...string) float64 {
		b := ui.NewParagraphBuilder(e, nil)
		b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Glutter No Such Family", FontFamilyFallback: fallback, FontSize: ui.Ptr(20.0)}))
		b.AddText(s)
		p := b.Build()
		p.Layout(ui.NewParagraphConstraints(10000))
		return p.LongestLine()
	}
	if i, w := lineWidth("iiiiiiii", "monospace"), lineWidth("WWWWWWWW", "monospace"); i != w {
		t.Errorf("with the monospace fallback the text is not monospace: iiiiiiii is %v wide, WWWWWWWW is %v", i, w)
	}
	if i, w := lineWidth("iiiiiiii"), lineWidth("WWWWWWWW"); i == w {
		t.Skipf("the system's default font is monospace (%v wide for both)", i)
	}
}

func TestFontQueriesOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := openFontconfig(ctx)
	if err != nil {
		t.Skipf("fontconfig is not available: %v", err)
	}
	defer m.Close()
	c := text.NewCollection(text.CollectionOpts{Matcher: m})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	e := layout.NewEngine(ctx, c)
	mono, ok := e.ResolveFont("monospace", ui.FontWeightW400, ui.FontStyleNormal)
	if !ok || mono.Origin != ui.FontOriginSystem || mono.Path == "" || mono.Family == "" {
		t.Fatalf("ResolveFont(monospace) = %+v, %v; want a system font", mono, ok)
	}
	if _, err := os.Stat(mono.Path); err != nil {
		t.Errorf("the file of the monospace font: %v", err)
	}
	if got, ok := e.ResolveFont("Glutter No Such Family", ui.FontWeightW400, ui.FontStyleNormal); ok {
		t.Errorf("a family that is not installed resolved to %+v", got)
	}
	if got, ok := e.ResolveFont(fonts.RobotoFamily, ui.FontWeightW400, ui.FontStyleNormal); !ok || got.Origin != ui.FontOriginBundled {
		t.Errorf("ResolveFont(Roboto) = %+v, %v; want the bundled font", got, ok)
	}
	families := e.InstalledFontFamilies()
	for _, want := range []string{fonts.RobotoFamily, fonts.MaterialIconsFamily, mono.Family} {
		if !slices.ContainsFunc(families, func(f string) bool { return strings.EqualFold(f, want) }) {
			t.Errorf("InstalledFontFamilies lacks %q", want)
		}
	}
	if !slices.IsSortedFunc(families, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }) {
		t.Error("the families are not sorted")
	}
}
