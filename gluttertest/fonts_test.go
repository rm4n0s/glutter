package gluttertest

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// monoAdvance is the advance of a glyph of Go Mono, in em.
const monoAdvance = 1229.0 / 2048.0

func monoText(s string, size float64, weight ui.FontWeight, italic bool) widgets.Widget {
	fontStyle := ui.FontStyleNormal
	if italic {
		fontStyle = ui.FontStyleItalic
	}
	style := painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: "Go Mono", FontFamilyFallback: []string{"Roboto"}, FontSize: foundation.Some(size),
		FontWeight: weight, FontStyle: fontStyle,
	})
	return widgets.NewText(s, widgets.TextOpts{Style: style})
}

func widthOfText(ts *Tester, w widgets.Widget) float64 {
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewAlign(w, widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})))
	return ts.Size(ByType[*widgets.Text]()).Width
}

func TestOptsFonts(t *testing.T) {
	const size = 20.0
	want := 8 * monoAdvance * size
	tests := []struct {
		name  string
		fonts []Font
		mono  bool
	}{
		{"no fonts: Roboto draws the text", nil, false},
		{"one font", []Font{{Family: "Go Mono", Data: gomono.TTF}}, true},
		{"the family inside the font", []Font{{Data: gomono.TTF}}, true},
		{"four styles of one family", []Font{
			{Family: "Go Mono", Data: gomono.TTF}, {Family: "Go Mono", Data: gomonobold.TTF},
			{Family: "Go Mono", Data: gomonoitalic.TTF}, {Family: "Go Mono", Data: gomonobolditalic.TTF},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t, Opts{Fonts: tt.fonts})
			got := widthOfText(ts, monoText("iiiiWWWW", size, ui.FontWeightW400, false))
			if tt.mono && math.Abs(got-want) > 0.5 {
				t.Errorf("width = %v, want %v (monospace)", got, want)
			}
			if !tt.mono && math.Abs(got-want) < 0.5 {
				t.Errorf("width = %v: the font is there without being asked for", got)
			}
		})
	}
}

func TestOptsFontsDoNotChangeTheSharedEngine(t *testing.T) {
	const size = 20.0
	want := 8 * monoAdvance * size
	with := New(t, Opts{Fonts: []Font{{Family: "Go Mono", Data: gomono.TTF}}})
	if got := widthOfText(with, monoText("iiiiWWWW", size, ui.FontWeightW400, false)); math.Abs(got-want) > 0.5 {
		t.Fatalf("with the font: width = %v, want %v", got, want)
	}
	without := New(t)
	if got := widthOfText(without, monoText("iiiiWWWW", size, ui.FontWeightW400, false)); math.Abs(got-want) < 0.5 {
		t.Errorf("a tester without Fonts saw the font of another: width = %v", got)
	}
}

func TestOptsFontsFailures(t *testing.T) {
	tests := []struct {
		name  string
		opts  Opts
		route string
	}{
		{"data that is no font", Opts{Fonts: []Font{{Family: "X", Data: []byte("nope")}}}, "FontDataInvalid"},
		{"Fonts with TextEngine", Opts{Fonts: []Font{{Family: "Go Mono", Data: gomono.TTF}}, TextEngine: nopEngine{}}, "TesterFontsWithTextEngine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &fakeTB{}
			func() {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(fatalStop); !ok {
							panic(r)
						}
					}
				}()
				New(tb, tt.opts)
			}()
			if len(tb.fatals) == 0 {
				t.Fatal("New did not fail the test")
			}
			err, _ := tb.fatals[0].(error)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", tb.fatals[0])
			}
			requireRoute(t, appErr, tt.route)
		})
	}
}

// nopEngine is a text engine that builds nothing.
type nopEngine struct{}

func (nopEngine) BuildParagraph(*ui.ParagraphContent) ui.ParagraphLayout { return nil }

func TestMonospaceGolden(t *testing.T) {
	ts := New(t, Opts{Size: ui.Size{Width: 420, Height: 220}, Fonts: []Font{
		{Family: "Go Mono", Data: gomono.TTF}, {Family: "Go Mono", Data: gomonobold.TTF},
		{Family: "Go Mono", Data: gomonoitalic.TTF}, {Family: "Go Mono", Data: gomonobolditalic.TTF},
	}})
	line := func(s string, size float64, weight ui.FontWeight, italic bool) widgets.Widget {
		return monoText(s, size, weight, italic)
	}
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColoredBox(ui.NewColor(0xFFFFFFFF),
		widgets.NewPadding(painting.NewEdgeInsetsAll(8), widgets.NewColumn([]widgets.Widget{
			line("func main() { fmt.Println(1) }", 14, ui.FontWeightW400, false),
			line("func main() { fmt.Println(1) }", 14, ui.FontWeightW700, false),
			line("func main() { fmt.Println(1) }", 14, ui.FontWeightW400, true),
			line("func main() { fmt.Println(1) }", 14, ui.FontWeightW700, true),
			line("iiiiiiiiii WWWWWWWWWW", 20, ui.FontWeightW400, false),
			line("0123456789 {}[]()<>", 28, ui.FontWeightW400, false),
		}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})))))
	ts.MatchesGolden("monospace_text")
}

func TestSetPlatformConfigurationDesktopFonts(t *testing.T) {
	ts := New(t)
	d := ts.Binding.PlatformDispatcher()
	if f, _ := d.MonospaceFont(); f != "" {
		t.Fatalf("MonospaceFont = %q before the test says", f)
	}
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) {
		c.MonospaceFontFamily, c.MonospaceFontSize = "Go Mono", 12
		c.UIFontFamily, c.UIFontSize = "Roboto", 11
	})
	if f, s := d.MonospaceFont(); f != "Go Mono" || s != 12 {
		t.Errorf("MonospaceFont = %q, %v", f, s)
	}
	if f, s := d.UIFont(); f != "Roboto" || s != 11 {
		t.Errorf("UIFont = %q, %v", f, s)
	}
}

func TestTestEngineAnswersFontQueries(t *testing.T) {
	ts := New(t, Opts{Fonts: []Font{{Family: "Go Mono", Data: gomono.TTF}}})
	d := ts.Binding.PlatformDispatcher()
	got, ok := d.ResolveFont("Go Mono", ui.FontWeightW400, ui.FontStyleNormal)
	if !ok || got.Origin != ui.FontOriginLoaded || got.Family != "Go Mono" {
		t.Errorf("ResolveFont(Go Mono) = %+v, %v", got, ok)
	}
	if got, ok := d.ResolveFont("Roboto", ui.FontWeightW400, ui.FontStyleNormal); !ok || got.Origin != ui.FontOriginBundled {
		t.Errorf("ResolveFont(Roboto) = %+v, %v", got, ok)
	}
	if got := d.InstalledFontFamilies(); len(got) != 3 || got[0] != "Go Mono" {
		t.Errorf("InstalledFontFamilies = %v, want the loaded and the bundled families", got)
	}
}
