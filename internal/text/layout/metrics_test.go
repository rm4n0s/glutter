package layout

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
)

// TestFontMetricsAreThoseOfAOneLineParagraph compares the metrics of a style with what a paragraph of one
// line in that style measures: the line's ascent, descent and height, the baseline, and the width of "0".
func TestFontMetricsAreThoseOfAOneLineParagraph(t *testing.T) {
	e := bundledEngine(t)
	if err := e.LoadFontFromList(gomono.TTF, "Test Mono"); err != nil {
		t.Fatal(err)
	}
	style := func(family string, size float64, height float64) *ui.TextStyle {
		o := ui.TextStyleOpts{FontFamily: family, FontSize: ui.Ptr(size)}
		if height != 0 {
			o.Height = ui.Ptr(height)
		}
		return ui.NewTextStyle(o)
	}
	tests := []struct {
		name  string
		style *ui.TextStyle
		mono  bool
	}{
		{"Roboto at 14", style(fonts.RobotoFamily, 14, 0), false},
		{"Roboto at 37.5", style(fonts.RobotoFamily, 37.5, 0), false},
		{"Roboto with a line height", style(fonts.RobotoFamily, 20, 1.5), false},
		{"the icon font at 24 (all its glyphs have one advance)", style(fonts.MaterialIconsFamily, 24, 0), true},
		{"a loaded monospaced font at 13", style("Test Mono", 13, 0), true},
		{"a loaded monospaced font at 21", style("Test Mono", 21, 0), true},
		{"a loaded monospaced font with a line height", style("Test Mono", 16, 1.2), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := tt.style.Resolve(ui.NewParagraphStyle().BaseTextStyle())
			got := e.FontMetrics(&resolved)

			for _, text := range []string{"0", "00"} {
				p := build(e, text, tt.style, nil)
				p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
				line := p.ComputeLineMetrics()[0]
				if got.Ascent != line.Ascent || got.Descent != line.Descent || got.Height != line.Height || got.Height != p.Height() {
					t.Errorf("paragraph %q: metrics = ascent %v descent %v height %v, paragraph has %v %v %v (height %v)",
						text, got.Ascent, got.Descent, got.Height, line.Ascent, line.Descent, line.Height, p.Height())
				}
				if got.Ascent != p.AlphabeticBaseline() {
					t.Errorf("paragraph %q: ascent %v, baseline of the paragraph %v", text, got.Ascent, p.AlphabeticBaseline())
				}
			}
			zero := build(e, "0", tt.style, nil)
			if got.ZeroAdvance != zero.MaxIntrinsicWidth() || got.ZeroAdvance <= 0 {
				t.Errorf("advance of 0 = %v, paragraph is %v wide", got.ZeroAdvance, zero.MaxIntrinsicWidth())
			}
			if got.Monospaced != tt.mono {
				t.Errorf("Monospaced = %v, want %v", got.Monospaced, tt.mono)
			}
			if got.Leading < 0 || got.Ascent+got.Descent < got.Leading {
				t.Errorf("leading %v does not fit in the line: ascent %v descent %v", got.Leading, got.Ascent, got.Descent)
			}
		})
	}
}

// TestFontMetricsScaleWithTheSize checks that a size twice as large measures twice as much, up to the
// rounding of the height to whole pixels.
func TestFontMetricsScaleWithTheSize(t *testing.T) {
	e := bundledEngine(t)
	at := func(size float64) ui.FontMetrics {
		r := roboto(size).Resolve(ui.NewParagraphStyle().BaseTextStyle())
		return e.FontMetrics(&r)
	}
	small, large := at(16), at(32)
	for _, c := range []struct {
		name         string
		small, large float64
	}{
		{"ascent", small.Ascent, large.Ascent}, {"descent", small.Descent, large.Descent},
		{"leading", small.Leading, large.Leading}, {"advance of 0", small.ZeroAdvance, large.ZeroAdvance},
	} {
		if math.Abs(2*c.small-c.large) > 1e-6 {
			t.Errorf("%s: %v at 16, %v at 32", c.name, c.small, c.large)
		}
	}
	if math.Abs(2*small.Height-large.Height) > 1 {
		t.Errorf("height: %v at 16, %v at 32", small.Height, large.Height)
	}
}

// TestFontMetricsWithoutFonts checks that an engine with no font at all still answers, with the
// fallback metrics a paragraph uses, and says the font is not monospaced.
func TestFontMetricsWithoutFonts(t *testing.T) {
	e := NewEngine(t.Context(), newEmptyCollection())
	r := roboto(10).Resolve(ui.NewParagraphStyle().BaseTextStyle())
	got := e.FontMetrics(&r)
	p := build(e, "0", roboto(10), nil)
	p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	if got.Height != p.Height() || got.Monospaced {
		t.Errorf("metrics = %+v, paragraph height %v", got, p.Height())
	}
}
