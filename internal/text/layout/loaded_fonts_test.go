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
	if !appErr.HasRoute("Engine.LoadFontFromList->Collection.AddFont.FontDataInvalid") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
