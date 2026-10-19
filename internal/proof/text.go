package proof

import (
	"context"
	"strconv"
	"sync"

	"github.com/rm4n0s/glutter/internal/text"
	textfontconfig "github.com/rm4n0s/glutter/internal/text/fontconfig"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name: "text-tour",
		Description: "text: Latin in several sizes and weights, italic, Arabic and Hebrew (right to left), a CJK line, an emoji line, every " +
			"decoration style, text shadows, an ellipsized paragraph and a justified one; it uses the fonts of the machine for the scripts " +
			"that Roboto lacks",
		Width:  640,
		Height: 480,
		Build:  func() *ui.Scene { return TextScene(SystemTextEngine(), false) },
	})
}

var (
	systemEngineOnce sync.Once
	systemEngine     *layout.Engine
)

// SystemTextEngine returns the text engine of the proof scenes: the bundled fonts and the fonts
// of the machine, found through fontconfig if it is there and by scanning the font directories
// if not. The engine is made once.
//
// Flutter: no counterpart
func SystemTextEngine() *layout.Engine {
	systemEngineOnce.Do(func() {
		ctx := context.Background()
		opts := text.CollectionOpts{}
		if m, err := textfontconfig.Open(ctx); err == nil {
			opts.Matcher = m
		}
		c := text.NewCollection(opts)
		if err := fonts.Register(c); err != nil {
			panic(err)
		}
		systemEngine = layout.NewEngine(ctx, c)
	})
	return systemEngine
}

// BundledTextEngine returns a text engine that knows only the bundled fonts, so that what it
// draws is the same on every machine.
//
// Flutter: no counterpart
func BundledTextEngine() *layout.Engine {
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		panic(err)
	}
	return layout.NewEngine(context.Background(), c)
}

// noSystemFonts is a font matcher that finds nothing.
type noSystemFonts struct{}

func (noSystemFonts) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// TextScene draws the text proof: the scene of the text-tour proof, made with the given engine.
// With latinOnly it leaves out the right-to-left, CJK and emoji text, which need fonts that
// the bundled ones lack; what is left is the same on every machine, which is what the golden
// tests need.
//
// Flutter: no counterpart
func TextScene(e *layout.Engine, latinOnly bool) *ui.Scene {
	return singlePicture(640, 480, func(c *ui.Canvas) { drawText(c, e, latinOnly) })
}

// textCtx is what the helpers that draw text share.
type textCtx struct {
	c *ui.Canvas
	e *layout.Engine
}

// run is a piece of text in a style.
type run struct {
	text  string
	style ui.TextStyleOpts
}

// draw lays out the runs in a paragraph of the given width and draws it at (x, y); it returns
// the height.
func (t textCtx) draw(x, y, width float64, ps ui.ParagraphStyleOpts, runs ...run) float64 {
	if ps.FontFamily == "" {
		ps.FontFamily = fonts.RobotoFamily
	}
	b := ui.NewParagraphBuilder(t.e, ui.NewParagraphStyle(ps))
	for _, r := range runs {
		b.PushStyle(ui.NewTextStyle(r.style))
		b.AddText(r.text)
		b.Pop()
	}
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(width))
	t.c.DrawParagraph(p, ui.Offset{Dx: x, Dy: y})
	return p.Height()
}

// inkSize is the style of black text at a size.
func inkSize(s float64) ui.TextStyleOpts {
	return ui.TextStyleOpts{Color: colorOf(black), FontSize: &s}
}

func colorOf(v uint32) *ui.Color { c := ui.NewColor(v); return &c }

func size(v float64) *float64 { return &v }

const (
	black  = 0xFF212121
	grey   = 0xFF757575
	accent = 0xFF1565C0
)

func drawText(c *ui.Canvas, e *layout.Engine, latinOnly bool) {
	t := textCtx{c, e}
	label := func(x, y float64, s string) {
		t.draw(x, y, 300, ui.ParagraphStyleOpts{FontSize: size(9)}, run{s, ui.TextStyleOpts{Color: colorOf(grey)}})
	}

	// Left column: Latin text.
	const lx, lw = 16.0, 296.0
	y := 8.0
	label(lx, y, "SIZES")
	y += 12
	for _, s := range []float64{12, 16, 24, 36} {
		y += t.draw(lx, y, lw, ui.ParagraphStyleOpts{}, run{"Glutter text " + strconv.Itoa(int(s)), ui.TextStyleOpts{Color: colorOf(black), FontSize: size(s)}})
	}
	y += 6
	label(lx, y, "WEIGHTS AND ITALIC")
	y += 12
	var weights []run
	for _, w := range []struct {
		name   string
		weight ui.FontWeight
	}{{"Thin ", ui.FontWeightW100}, {"Light ", ui.FontWeightW300}, {"Regular ", ui.FontWeightW400}, {"Medium ", ui.FontWeightW500}, {"Bold ", ui.FontWeightW700}, {"Black", ui.FontWeightW900}} {
		weights = append(weights, run{w.name, ui.TextStyleOpts{Color: colorOf(black), FontSize: size(18), FontWeight: w.weight}})
	}
	y += t.draw(lx, y, lw, ui.ParagraphStyleOpts{}, weights...)
	y += t.draw(lx, y, lw, ui.ParagraphStyleOpts{}, run{"Italic text in Roboto", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(18), FontStyle: ui.FontStyleItalic}},
		run{" bold italic", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(18), FontStyle: ui.FontStyleItalic, FontWeight: ui.FontWeightBold}})
	y += 6
	label(lx, y, "DECORATIONS")
	y += 12
	deco := func(word string, d ui.TextDecoration, st ui.TextDecorationStyle) run {
		return run{word + " ", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(16), Decoration: ui.Ptr(d), DecorationStyle: st, DecorationColor: colorOf(0xFFD81B60)}}
	}
	y += 4 + t.draw(lx, y, lw, ui.ParagraphStyleOpts{},
		deco("solid", ui.TextDecorationUnderline, ui.TextDecorationStyleSolid),
		deco("double", ui.TextDecorationUnderline, ui.TextDecorationStyleDouble),
		deco("dotted", ui.TextDecorationUnderline, ui.TextDecorationStyleDotted),
		deco("dashed", ui.TextDecorationUnderline, ui.TextDecorationStyleDashed),
		deco("wavy", ui.TextDecorationUnderline, ui.TextDecorationStyleWavy))
	y += 6 + t.draw(lx, y, lw, ui.ParagraphStyleOpts{},
		deco("overline", ui.TextDecorationOverline, ui.TextDecorationStyleSolid),
		deco("line-through", ui.TextDecorationLineThrough, ui.TextDecorationStyleSolid),
		deco("all three", ui.CombineTextDecorations(ui.TextDecorationUnderline, ui.TextDecorationOverline, ui.TextDecorationLineThrough), ui.TextDecorationStyleWavy))
	y += 6
	label(lx, y, "SHADOWS AND BACKGROUND")
	y += 12
	shadows := []ui.Shadow{ui.NewShadow(ui.ShadowOpts{Color: colorOf(0x99000000), Offset: ui.Offset{Dx: 2, Dy: 3}, BlurRadius: 4}),
		ui.NewShadow(ui.ShadowOpts{Color: colorOf(0x661565C0), Offset: ui.Offset{Dx: -2, Dy: -1}, BlurRadius: 0})}
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFFF59D))
	t.draw(lx, y, lw, ui.ParagraphStyleOpts{},
		run{"Shadow ", ui.TextStyleOpts{Color: colorOf(accent), FontSize: size(30), FontWeight: ui.FontWeightBold, Shadows: shadows}},
		run{"marked", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(20), Background: bg}})

	// Right column: other scripts and paragraphs.
	const rx, rw = 328.0, 296.0
	y = 8
	if !latinOnly {
		label(rx, y, "RIGHT TO LEFT")
		y += 12
		y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionRTL}, run{"مرحبا بالعالم", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(24)}})
		y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{TextDirection: ui.TextDirectionRTL}, run{"שלום עולם", ui.TextStyleOpts{Color: colorOf(black), FontSize: size(22)}})
		y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{}, run{"abc שלום 123 مرحبا def", inkSize(18)})
		y += 6
		label(rx, y, "CJK AND EMOJI")
		y += 12
		y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{}, run{"日本語のテキストを折り返す", inkSize(20)})
		y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{}, run{"😀 🚀 🌈 👨‍👩‍👧 ❤️", inkSize(26)})
		y += 6
	}
	label(rx, y, "ELLIPSIS (2 LINES)")
	y += 12
	const long = "Glutter lays text out with HarfBuzz shaping, bidi reordering and UAX #14 line breaking, then draws the glyphs on the GPU."
	y += t.draw(rx, y, rw, ui.ParagraphStyleOpts{MaxLines: ui.Ptr(2), Ellipsis: "…"}, run{long, inkSize(14)})
	y += 6
	label(rx, y, "JUSTIFIED")
	y += 12
	t.draw(rx, y, rw, ui.ParagraphStyleOpts{TextAlign: ui.TextAlignJustify}, run{long + " The last line is not stretched.", inkSize(14)})
}
