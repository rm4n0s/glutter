package fontconfig

import (
	"context"
	"testing"

	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

func TestCollectionOverFontconfig(t *testing.T) {
	ctx := context.Background()
	m, err := Open(ctx)
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
	m, err := Open(ctx)
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
