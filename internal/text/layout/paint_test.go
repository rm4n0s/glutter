package layout

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// paintOf returns what a laid-out paragraph emits to be drawn.
func paintOf(p ui.ParagraphLayout) []ui.ParagraphDraw {
	var out []ui.ParagraphDraw
	p.(ui.ParagraphPainter).PaintParagraph(func(d ui.ParagraphDraw) { out = append(out, d) })
	return out
}

func paintColor(p ui.Paint) ui.Color      { return p.Color() }
func maskOf(p ui.Paint) *ui.MaskFilter    { return p.MaskFilter() }
func styleOf(p ui.Paint) ui.PaintingStyle { return p.Style() }
func widthOf(p ui.Paint) float64          { return p.StrokeWidth() }

func styled(e *Engine, text string, size float64, opts ui.TextStyleOpts, ps *ui.ParagraphStyle) *paragraph {
	opts.FontFamily = "Roboto"
	opts.FontSize = ui.Ptr(size)
	p := build(e, text, ui.NewTextStyle(opts), ps).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(400))
	return p
}

func TestPaintEmitsOneSnappedPathPerGlyphWithAnOutline(t *testing.T) {
	e := bundledEngine(t)
	// The space has no outline: three glyphs, two paths.
	got := paintOf(styled(e, "H I", 20, ui.TextStyleOpts{}, nil))
	if len(got) != 2 {
		t.Fatalf("%d pieces, want 2", len(got))
	}
	for i, d := range got {
		op, ok := d.Op.(ui.OpDrawPath)
		if !ok || !d.Snap || d.Matrix != nil {
			t.Errorf("piece %d: %T snap=%v matrix=%v", i, d.Op, d.Snap, d.Matrix)
		}
		if paintColor(op.Paint) != ui.NewColor(0xFF000000) {
			t.Errorf("piece %d: color %v", i, paintColor(op.Paint))
		}
	}
	if got[1].Offset.Dx <= got[0].Offset.Dx || got[0].Offset.Dy != got[1].Offset.Dy {
		t.Errorf("the glyphs go right along one baseline: %v then %v", got[0].Offset, got[1].Offset)
	}
	// The same glyph at the same size is the same path, so the renderers can cache it.
	again := paintOf(styled(e, "H", 20, ui.TextStyleOpts{}, nil))
	if again[0].Op.(ui.OpDrawPath).Path != got[0].Op.(ui.OpDrawPath).Path {
		t.Error("a glyph's path is made once")
	}
	other := paintOf(styled(e, "H", 21, ui.TextStyleOpts{}, nil))
	if other[0].Op.(ui.OpDrawPath).Path == got[0].Op.(ui.OpDrawPath).Path {
		t.Error("another size has another path")
	}
}

func TestPaintTheForegroundPaintReplacesTheColor(t *testing.T) {
	e := bundledEngine(t)
	fg := ui.NewPaint()
	fg.SetColor(ui.NewColor(0xFF112233))
	got := paintOf(styled(e, "H", 20, ui.TextStyleOpts{Foreground: fg}, nil))
	if c := paintColor(got[0].Op.(ui.OpDrawPath).Paint); c != ui.NewColor(0xFF112233) {
		t.Errorf("color %v", c)
	}
	red := ui.NewColor(0xFFFF0000)
	got = paintOf(styled(e, "H", 20, ui.TextStyleOpts{Color: &red}, nil))
	if c := paintColor(got[0].Op.(ui.OpDrawPath).Paint); c != red {
		t.Errorf("color %v", c)
	}
}

func TestPaintOrderIsBackgroundShadowGlyphsDecoration(t *testing.T) {
	e := bundledEngine(t)
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFFFF00))
	blue := ui.NewColor(0xFF0000FF)
	p := styled(e, "HH", 20, ui.TextStyleOpts{
		Background: bg,
		Decoration: ui.Ptr(ui.TextDecorationUnderline),
		Shadows:    []ui.Shadow{ui.NewShadow(ui.ShadowOpts{Color: &blue, Offset: ui.Offset{Dx: 2, Dy: 3}, BlurRadius: 4})},
	}, nil)
	got := paintOf(p)
	// background, 2 shadows, 2 glyphs, underline
	if len(got) != 6 {
		t.Fatalf("%d pieces: %v", len(got), got)
	}
	rect, ok := got[0].Op.(ui.OpDrawRect)
	if !ok || paintColor(rect.Paint) != ui.NewColor(0xFFFFFF00) || got[0].Snap {
		t.Errorf("background: %+v", got[0])
	}
	l := p.lines[0]
	if rect.Rect.Top != l.baseline-l.runs[0].run.ascent || rect.Rect.Bottom != l.baseline+l.runs[0].run.descent || rect.Rect.Width() <= 0 {
		t.Errorf("the background covers the run: %v", rect.Rect)
	}
	shadow, glyph := got[1], got[3]
	sp := shadow.Op.(ui.OpDrawPath)
	if paintColor(sp.Paint) != blue || maskOf(sp.Paint) == nil {
		t.Errorf("shadow paint: %+v", sp.Paint)
	}
	if !shadow.Snap || shadow.Offset.Dx != glyph.Offset.Dx+2 || shadow.Offset.Dy != glyph.Offset.Dy+3 {
		t.Errorf("the shadow is the glyph moved by the offset: %v vs %v", shadow.Offset, glyph.Offset)
	}
	if sp.Path != glyph.Op.(ui.OpDrawPath).Path {
		t.Error("the shadow draws the glyph's outline")
	}
	if line, ok := got[5].Op.(ui.OpDrawRect); !ok || got[5].Snap || line.Rect.Width() <= 0 {
		t.Errorf("underline: %+v", got[5])
	}
	// A shadow without blur has no mask filter.
	flat := paintOf(styled(e, "H", 20, ui.TextStyleOpts{Shadows: []ui.Shadow{ui.NewShadow()}}, nil))
	if maskOf(flat[0].Op.(ui.OpDrawPath).Paint) != nil {
		t.Error("a shadow with no blur is not blurred")
	}
}

func TestPaintDecorationStyles(t *testing.T) {
	e := bundledEngine(t)
	tests := []struct {
		name  string
		style ui.TextDecorationStyle
		check func(t *testing.T, got []ui.ParagraphDraw)
	}{
		{"solid", ui.TextDecorationStyleSolid, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) != 1 {
				t.Errorf("%d lines", len(got))
			}
		}},
		{"unspecified is solid", ui.TextDecorationStyleUnspecified, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) != 1 {
				t.Errorf("%d lines", len(got))
			}
		}},
		{"double", ui.TextDecorationStyleDouble, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) != 2 {
				t.Fatalf("%d lines", len(got))
			}
			a, b := got[0].Op.(ui.OpDrawRect).Rect, got[1].Op.(ui.OpDrawRect).Rect
			if b.Top-a.Top != DoubleDecorationSpacing || a.Left != b.Left || a.Right != b.Right {
				t.Errorf("%v and %v", a, b)
			}
		}},
		{"dotted", ui.TextDecorationStyleDotted, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) < 10 {
				t.Fatalf("%d dots", len(got))
			}
			a, b := got[0].Op.(ui.OpDrawRect).Rect, got[1].Op.(ui.OpDrawRect).Rect
			if a.Width() != a.Height() || b.Left-a.Left != 2*a.Width() {
				t.Errorf("dots %v %v", a, b)
			}
		}},
		{"dashed", ui.TextDecorationStyleDashed, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) < 3 {
				t.Fatalf("%d dashes", len(got))
			}
			a, b := got[0].Op.(ui.OpDrawRect).Rect, got[1].Op.(ui.OpDrawRect).Rect
			if a.Width() != 4*a.Height() || b.Left-a.Left != 6*a.Height() {
				t.Errorf("dashes %v %v", a, b)
			}
			last := got[len(got)-1].Op.(ui.OpDrawRect).Rect
			if last.Width() > a.Width() {
				t.Errorf("the last dash is cut at the end: %v", last)
			}
		}},
		{"wavy", ui.TextDecorationStyleWavy, func(t *testing.T, got []ui.ParagraphDraw) {
			if len(got) != 1 {
				t.Fatalf("%d pieces", len(got))
			}
			op, ok := got[0].Op.(ui.OpDrawPath)
			if !ok || styleOf(op.Paint) != ui.PaintingStyleStroke || widthOf(op.Paint) <= 0 {
				t.Errorf("wavy: %+v", got[0].Op)
			}
		}},
	}
	for _, tt := range tests {
		p := styled(e, "Wave", 40, ui.TextStyleOpts{Decoration: ui.Ptr(ui.TextDecorationUnderline), DecorationStyle: tt.style}, nil)
		// The four glyphs come first, then the decoration.
		decor := paintOf(p)[4:]
		t.Run(tt.name, func(t *testing.T) { tt.check(t, decor) })
	}
}

func TestPaintLineThroughAndOverline(t *testing.T) {
	e := bundledEngine(t)
	red := ui.NewColor(0xFFFF0000)
	p := styled(e, "H", 40, ui.TextStyleOpts{
		Decoration:      ui.Ptr(ui.CombineTextDecorations(ui.TextDecorationOverline, ui.TextDecorationLineThrough, ui.TextDecorationUnderline)),
		DecorationColor: &red,
	}, nil)
	got := paintOf(p)[1:]
	if len(got) != 3 {
		t.Fatalf("%d lines", len(got))
	}
	ys := map[float64]bool{}
	for _, d := range got {
		r := d.Op.(ui.OpDrawRect)
		if paintColor(r.Paint) != red {
			t.Errorf("color %v", paintColor(r.Paint))
		}
		ys[r.Rect.Top] = true
	}
	if len(ys) != 3 {
		t.Errorf("the three lines are at three heights: %v", ys)
	}
}

func TestPaintSkipsPlaceholders(t *testing.T) {
	e := bundledEngine(t)
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFF0000))
	b := ui.NewParagraphBuilder(e, nil)
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: ui.Ptr(20.0), Background: bg}))
	b.AddPlaceholder(30, 30, ui.PlaceholderAlignmentBaseline, ui.AddPlaceholderOpts{Baseline: ui.TextBaselineAlphabetic})
	p := b.Build().Layouter()
	p.Layout(ui.NewParagraphConstraints(400))
	if got := paintOf(p); len(got) != 0 {
		t.Errorf("a placeholder is left to the widget that fills it: %v", got)
	}
}

func TestPaintEllipsisBackground(t *testing.T) {
	e := bundledEngine(t)
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFFFFFF00))
	p := build(e, "A long text that does not fit on the line", ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "Roboto", FontSize: ui.Ptr(20.0), Background: bg}),
		ui.NewParagraphStyle(ui.ParagraphStyleOpts{MaxLines: ui.Ptr(1), Ellipsis: "…"})).(*paragraph)
	p.Layout(ui.NewParagraphConstraints(120))
	var right float64
	for _, d := range paintOf(p) {
		if r, ok := d.Op.(ui.OpDrawRect); ok && r.Rect.Right > right {
			right = r.Rect.Right
		}
	}
	if right < 100 || right > 120.01 {
		t.Errorf("the background reaches the ellipsis at the end of the line: %v", right)
	}
}

func TestPaintGlyphCachesAreBounded(t *testing.T) {
	e := bundledEngine(t)
	p := styled(e, "H", 20, ui.TextStyleOpts{}, nil)
	face, gid := p.runs[0].face, p.runs[0].glyphs[0].id
	for i := 0; i < maxCachedGlyphs+10; i++ {
		e.glyphPath(face, gid, 10+float64(i)/100)
	}
	if len(e.paths) > maxCachedGlyphs {
		t.Errorf("%d cached paths", len(e.paths))
	}
}

func TestPaintColorEmojiIsAPicture(t *testing.T) {
	e := systemEngine(t, "😀")
	p := styled(e, "😀", 32, ui.TextStyleOpts{}, nil)
	var pics int
	for _, d := range paintOf(p) {
		if pic, ok := d.Op.(ui.OpDrawPicture); ok {
			pics++
			if pic.Picture == nil || d.Matrix == nil || !d.Snap {
				t.Errorf("color glyph: %+v", d)
			}
			// The glyph is flipped and scaled from font units to the size.
			if m := d.Matrix.Storage(); m[5] >= 0 || m[0] <= 0 {
				t.Errorf("matrix %v", m)
			}
		}
	}
	if pics == 0 {
		t.Skip("the emoji font of the machine has no COLR glyphs")
	}
	again := paintOf(p)
	first := paintOf(p)
	if again[0].Op.(ui.OpDrawPicture).Picture != first[0].Op.(ui.OpDrawPicture).Picture {
		t.Error("a color glyph's picture is made once")
	}
}
