package soft

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/ui"
)

// textScene draws a paragraph of one style at an offset, under a transform, over white.
func textScene(t *testing.T, e *layout.Engine, text string, style ui.TextStyleOpts, ps ui.ParagraphStyleOpts, at ui.Offset, ctm *ui.Matrix4, w, h int) *ui.Scene {
	t.Helper()
	if style.FontFamily == "" {
		style.FontFamily = fonts.RobotoFamily
	}
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle(ps))
	b.PushStyle(ui.NewTextStyle(style))
	b.AddText(text)
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(float64(w)))
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, float64(w), float64(h)))})
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	if ctm != nil {
		c.Transform(ctm)
	}
	c.DrawParagraph(p, at)
	b2 := ui.NewSceneBuilder()
	b2.AddPicture(ui.Offset{}, rec.EndRecording())
	return b2.Build()
}

func renderText(t *testing.T, e *layout.Engine, text string, style ui.TextStyleOpts, ps ui.ParagraphStyleOpts, at ui.Offset, ctm *ui.Matrix4) *image.RGBA {
	t.Helper()
	img, err := Render(textScene(t, e, text, style, ps, at, ctm, 120, 60), 120, 60)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return img
}

func sameImages(a, b *image.RGBA) bool { return string(a.Pix) == string(b.Pix) }

// inkBounds is the box of the pixels that are not white.
func inkBounds(img *image.RGBA) image.Rectangle {
	var r image.Rectangle
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if c := img.RGBAAt(x, y); c != (color.RGBA{255, 255, 255, 255}) {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

func TestGoldenTextScene(t *testing.T) {
	scene := proof.TextScene(proof.BundledTextEngine(), true)
	img, err := Render(scene, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	goldens.Check(t, "proof-text-basic", img)
}

func TestTextIsDrawnInTheTextColorAtTheBaseline(t *testing.T) {
	e := proof.BundledTextEngine()
	red := ui.NewColor(0xFFD32F2F)
	img := renderText(t, e, "HIH", ui.TextStyleOpts{Color: &red, FontSize: ui.Ptr(40.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 10, Dy: 5}, nil)
	ink := inkBounds(img)
	if ink.Empty() {
		t.Fatal("nothing was drawn")
	}
	// Capital letters of Roboto at 40 px are about 28 px high and sit on the baseline.
	ascent := 40 * 1900.0 / 2048
	baseline := 5 + ascent
	if int(baseline)-ink.Max.Y > 1 || ink.Max.Y-int(baseline) > 2 {
		t.Errorf("the letters end at y=%d, the baseline is at %.1f", ink.Max.Y, baseline)
	}
	if h := ink.Dy(); h < 26 || h > 31 {
		t.Errorf("capitals are %d px high, want about 28", h)
	}
	if ink.Min.X < 10 || ink.Min.X > 16 {
		t.Errorf("the text starts at x=%d", ink.Min.X)
	}
	// The stem of the first H is the text color.
	if c := img.RGBAAt(ink.Min.X+3, (ink.Min.Y+ink.Max.Y)/2); c.R < 190 || c.G > 90 || c.B > 90 {
		t.Errorf("stem pixel = %v, want the red of the text", c)
	}
}

func TestGlyphsHaveFourSubpixelPositions(t *testing.T) {
	e := proof.BundledTextEngine()
	render := func(dx, dy float64) *image.RGBA {
		return renderText(t, e, "l", ui.TextStyleOpts{FontSize: ui.Ptr(30.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: dx, Dy: dy}, nil)
	}
	base := render(10, 5)
	tests := []struct {
		name   string
		dx, dy float64
		same   bool
	}{
		{"within an eighth of a pixel is the same position", 10.1, 5, true},
		{"a quarter is another position", 10.25, 5, false},
		{"just under a quarter is the same as it", 10.2, 5, false}, // not the same as 10.0: it rounds to .25
		{"a half", 10.5, 5, false},
		{"three quarters", 10.75, 5, false},
		{"a whole pixel moves it, not blurs it", 11, 5, false},
		{"vertical position snaps to the pixel", 10, 5.3, true},
		{"vertical position rounds up", 10, 5.7, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameImages(render(tt.dx, tt.dy), base); got != tt.same {
				t.Errorf("same as the base image = %v, want %v", got, tt.same)
			}
		})
	}
	// The four positions differ from each other, and a quarter more than a pixel is the first
	// again, one pixel along.
	seen := map[string]bool{}
	for _, dx := range []float64{10, 10.25, 10.5, 10.75} {
		seen[string(render(dx, 5).Pix)] = true
	}
	if len(seen) != 4 {
		t.Errorf("%d different images for four subpixel positions", len(seen))
	}
	if !sameImages(render(10.2, 5), render(10.25, 5)) || !sameImages(render(10.13, 5), render(10.25, 5)) {
		t.Error("positions that round to the same quarter are drawn differently")
	}
}

func TestTextUnderTransforms(t *testing.T) {
	e := proof.BundledTextEngine()
	ctm := ui.NewMatrix4RotationZ(0.3)
	ctm.Translate(30, 5, 0)
	img := renderText(t, e, "Rotated", ui.TextStyleOpts{FontSize: ui.Ptr(20.0)}, ui.ParagraphStyleOpts{}, ui.Offset{}, ctm)
	if inkBounds(img).Empty() {
		t.Error("rotated text was not drawn")
	}
	scaled := ui.NewMatrix4Diagonal3Values(2, 2, 1)
	big := renderText(t, e, "Hi", ui.TextStyleOpts{FontSize: ui.Ptr(10.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 5, Dy: 2}, scaled)
	small := renderText(t, e, "Hi", ui.TextStyleOpts{FontSize: ui.Ptr(20.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 10, Dy: 4}, nil)
	// Drawn at size 10 under a scale of 2 is about the size of drawing at 20 (the font is
	// scaled by the transform, not hinted).
	if b, s := inkBounds(big), inkBounds(small); abs(b.Dx()-s.Dx()) > 2 || abs(b.Dy()-s.Dy()) > 2 {
		t.Errorf("scaled text is %v, text at the final size is %v", b, s)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestShadowsBackgroundsAndDecorations(t *testing.T) {
	e := proof.BundledTextEngine()
	t.Run("shadow", func(t *testing.T) {
		plain := renderText(t, e, "I", ui.TextStyleOpts{FontSize: ui.Ptr(40.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 20, Dy: 5}, nil)
		sh := ui.NewShadow(ui.ShadowOpts{Color: ui.Ptr(ui.NewColor(0xFFFF0000)), Offset: ui.Offset{Dx: 8, Dy: 0}})
		shadowed := renderText(t, e, "I", ui.TextStyleOpts{FontSize: ui.Ptr(40.0), Shadows: []ui.Shadow{sh}}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 20, Dy: 5}, nil)
		pb, sb := inkBounds(plain), inkBounds(shadowed)
		if sb.Max.X-pb.Max.X < 7 || sb.Min.X != pb.Min.X {
			t.Errorf("plain %v, with a shadow %v: the shadow should extend 8 px to the right", pb, sb)
		}
		// The shadow is red and under the text, which stays black.
		y := (sb.Min.Y + sb.Max.Y) / 2
		if c := shadowed.RGBAAt(sb.Max.X-2, y); c.R < 200 || c.G > 60 {
			t.Errorf("shadow pixel = %v", c)
		}
		if c := shadowed.RGBAAt(pb.Min.X+3, y); c.R > 60 || c.G > 60 {
			t.Errorf("text pixel = %v, want black", c)
		}
	})
	t.Run("blurred shadow is softer", func(t *testing.T) {
		sh := ui.NewShadow(ui.ShadowOpts{Color: ui.Ptr(ui.NewColor(0xFF000000)), Offset: ui.Offset{Dx: 0, Dy: 12}, BlurRadius: 6})
		img := renderText(t, e, "I", ui.TextStyleOpts{FontSize: ui.Ptr(24.0), Color: ui.Ptr(ui.NewColor(0xFFFFFFFF)), Shadows: []ui.Shadow{sh}}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 20, Dy: 2}, nil)
		// Below the white letter there are grey pixels that are neither white nor black.
		mid := 0
		for y := 0; y < 60; y++ {
			c := img.RGBAAt(24, y)
			if c.R > 40 && c.R < 215 {
				mid++
			}
		}
		if mid < 3 {
			t.Errorf("only %d grey pixels in the blurred shadow", mid)
		}
	})
	t.Run("background", func(t *testing.T) {
		bg := ui.NewPaint()
		bg.SetColor(ui.NewColor(0xFF00FF00))
		img := renderText(t, e, "ab", ui.TextStyleOpts{FontSize: ui.Ptr(20.0), Background: bg}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 10, Dy: 10}, nil)
		if c := img.RGBAAt(11, 12); c.G != 255 || c.R != 0 {
			t.Errorf("background pixel = %v, want green", c)
		}
		if c := img.RGBAAt(100, 12); c != (color.RGBA{255, 255, 255, 255}) {
			t.Errorf("a pixel after the text = %v, want white", c)
		}
	})
	t.Run("underline is on the line below the baseline", func(t *testing.T) {
		red := ui.NewColor(0xFFFF0000)
		img := renderText(t, e, "xx", ui.TextStyleOpts{FontSize: ui.Ptr(30.0), Color: ui.Ptr(ui.NewColor(0x00000000)), Decoration: ui.TextDecorationUnderline, DecorationColor: &red}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 10, Dy: 5}, nil)
		ink := inkBounds(img)
		if ink.Empty() {
			t.Fatal("nothing drawn")
		}
		baseline := 5 + 30*1900.0/2048
		if float64(ink.Min.Y) < baseline-1 || float64(ink.Min.Y) > baseline+6 || ink.Dy() > 4 {
			t.Errorf("underline at %v, the baseline is at %.1f", ink, baseline)
		}
		// The line is thin and not on the pixel grid: look for the reddest pixel of it.
		best := color.RGBA{255, 255, 255, 255}
		for y := ink.Min.Y; y < ink.Max.Y; y++ {
			if c := img.RGBAAt((ink.Min.X+ink.Max.X)/2, y); int(c.G) < int(best.G) {
				best = c
			}
		}
		if best.R < 240 || best.G > 80 {
			t.Errorf("underline pixel = %v, want red", best)
		}
	})
}

func TestEmojiAreDrawnInColor(t *testing.T) {
	e := proof.SystemTextEngine()
	img := renderText(t, e, "😀", ui.TextStyleOpts{FontSize: ui.Ptr(40.0)}, ui.ParagraphStyleOpts{}, ui.Offset{Dx: 10, Dy: 5}, nil)
	ink := inkBounds(img)
	if ink.Empty() {
		t.Skip("the machine has no font with the emoji")
	}
	// A color emoji has pixels that are not grey.
	colored := 0
	for y := ink.Min.Y; y < ink.Max.Y; y++ {
		for x := ink.Min.X; x < ink.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A == 255 && abs(int(c.R)-int(c.B)) > 60 {
				colored++
			}
		}
	}
	if colored < 50 {
		t.Skipf("the emoji font of the machine is not a color font (%d colored pixels)", colored)
	}
}

func TestParagraphThatCannotDrawDrawsNothing(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, 10, 10))})
	c.DrawParagraph(ui.NewParagraph(nil), ui.Offset{})
	c.DrawParagraph(nil, ui.Offset{})
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	if _, err := Render(b.Build(), 10, 10); err != nil {
		t.Errorf("Render: %v", err)
	}
}
