package layout

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// monoAdvance is the advance of every glyph of Go Mono, in em.
const monoAdvance = 1229.0 / 2048.0

func widthIn(e *Engine, s string, style *ui.TextStyle) float64 {
	p := build(e, s, style, nil)
	p.Layout(ui.NewParagraphConstraints(10000))
	return p.LongestLine()
}

func TestLoadFontFromList(t *testing.T) {
	const size = 20.0
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Go Mono", FontFamilyFallback: []string{"Roboto"}, FontSize: ui.Ptr(size)})
	tests := []struct {
		name   string
		sample string
		style  *ui.TextStyle
		// mono is whether the sample is laid out with Go Mono's advance after the font is loaded.
		mono bool
	}{
		{"the family is the first one", "iiiiiWWW.i", style, true},
		{"the family is a fallback", "iiiiiWWW.i", ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontFamilyFallback: []string{"Go Mono"}, FontSize: ui.Ptr(size)}), false},
		{"the family is a fallback for a glyph the first lacks", "\u2500\u2502\u250c\u2500", ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontFamilyFallback: []string{"Go Mono"}, FontSize: ui.Ptr(size)}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := bundledEngine(t)
			before := widthIn(e, tt.sample, tt.style)
			if err := e.LoadFontFromList(gomono.TTF, "Go Mono"); err != nil {
				t.Fatalf("LoadFontFromList: %v", err)
			}
			after := widthIn(e, tt.sample, tt.style)
			want := float64(len([]rune(tt.sample))) * monoAdvance * size
			if tt.mono {
				if math.Abs(after-want) > 0.01 {
					t.Errorf("width after loading = %v, want %v (monospace)", after, want)
				}
				if math.Abs(before-want) < 0.5 {
					t.Errorf("width before loading = %v, the family was there already", before)
				}
			} else if after != before {
				t.Errorf("width changed from %v to %v: Roboto has every glyph, the fallback must not be used", before, after)
			}
		})
	}
}

func TestLoadedFontWeightAndSlant(t *testing.T) {
	e := bundledEngine(t)
	for _, data := range [][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF} {
		if err := e.LoadFontFromList(data, "Go Mono"); err != nil {
			t.Fatal(err)
		}
	}
	regular := e.fontsFor([]string{"Go Mono"}, ui.FontWeightW400, false, ui.Locale{}, nil).ResolveFace('a')
	bold := e.fontsFor([]string{"Go Mono"}, ui.FontWeightW700, false, ui.Locale{}, nil).ResolveFace('a')
	italic := e.fontsFor([]string{"Go Mono"}, ui.FontWeightW400, true, ui.Locale{}, nil).ResolveFace('a')
	if regular == nil || bold == nil || italic == nil {
		t.Fatal("a style of the loaded family has no face")
	}
	if regular == bold || regular == italic || bold == italic {
		t.Error("regular, bold and italic resolve to the same face")
	}
	for _, fw := range []ui.FontWeight{ui.FontWeightW400, ui.FontWeightW700} {
		st := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Go Mono", FontSize: ui.Ptr(20.0), FontWeight: fw})
		if got, want := widthIn(e, "abc", st), 3*monoAdvance*20; math.Abs(got-want) > 0.01 {
			t.Errorf("weight %v: width = %v, want %v", fw, got, want)
		}
	}
}

func TestLoadFontFromListErrors(t *testing.T) {
	e := bundledEngine(t)
	err := e.LoadFontFromList([]byte("nope"), "X")
	appErr, ok := errorsFrom(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Engine.LoadFontFromList->Collection.AddFont->describeFont.FontDataInvalid") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// TestLoadedSansSerifIsTheDefaultFont: a font loaded under the generic name sans-serif is the font of a
// style that names no family, which an engine without system fonts otherwise draws with the bundled font
// (its last resort, TestAFamilyThatNothingHasIsDrawnWithTheBundledFont).
func TestLoadedSansSerifIsTheDefaultFont(t *testing.T) {
	e := bundledEngine(t)
	style := ui.NewTextStyle(ui.TextStyleOpts{FontSize: ui.Ptr(20.0)})
	roboto := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: ui.Ptr(20.0)})
	if got, want := widthIn(e, "iiii", style), widthIn(e, "iiii", roboto); got != want || got == 0 {
		t.Fatalf("width before = %v, want that of the bundled Roboto, %v", got, want)
	}
	if err := e.LoadFontFromList(gomono.TTF, "sans-serif"); err != nil {
		t.Fatal(err)
	}
	if got, want := widthIn(e, "iiiW", style), 4*monoAdvance*20; math.Abs(got-want) > 0.01 {
		t.Errorf("width after = %v, want %v", got, want)
	}
}

// TestAFamilyThatNothingHasIsDrawnWithTheBundledFont: a style whose families no loaded, bundled or system
// font has (a name that is not installed, a generic name on a machine without system fonts, as in
// gluttertest) is laid out with the first bundled font, as Flutter falls back to its default font, and not
// with no glyph at all. A family that has a font keeps it, and a character that the family lacks and the
// bundled font has is drawn with the bundled one.
func TestAFamilyThatNothingHasIsDrawnWithTheBundledFont(t *testing.T) {
	e := bundledEngine(t)
	if err := e.LoadFontFromList(gomono.TTF, "Go Mono"); err != nil {
		t.Fatal(err)
	}
	size := ui.Ptr(20.0)
	roboto := widthIn(e, "Hello", ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: size}))
	mono := 5 * monoAdvance * 20
	if roboto == 0 || math.Abs(roboto-mono) < 1 {
		t.Fatalf("Roboto lays Hello out %v wide and Go Mono %v: the test needs them to differ", roboto, mono)
	}
	tests := []struct {
		name  string
		style ui.TextStyleOpts
		want  float64
	}{
		{"no family", ui.TextStyleOpts{}, roboto},
		{"a generic family with no system font", ui.TextStyleOpts{FontFamily: "monospace"}, roboto},
		{"a family that is not installed", ui.TextStyleOpts{FontFamily: "No Such Font"}, roboto},
		{"nor are its fallbacks", ui.TextStyleOpts{FontFamily: "No Such Font", FontFamilyFallback: []string{"Neither This", "serif"}}, roboto},
		{"a fallback that is loaded wins", ui.TextStyleOpts{FontFamily: "No Such Font", FontFamilyFallback: []string{"Go Mono"}}, mono},
		{"a family that is loaded keeps its font", ui.TextStyleOpts{FontFamily: "Go Mono"}, mono},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.style.FontSize = size
			if got := widthIn(e, "Hello", ui.NewTextStyle(tt.style)); math.Abs(got-tt.want) > 0.01 {
				t.Errorf("width = %v, want %v", got, tt.want)
			}
		})
	}
}
