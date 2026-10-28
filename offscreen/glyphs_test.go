package offscreen_test

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
)

// robotoStyle is a style in the bundled Roboto.
func robotoStyle(size float64) *ui.TextStyle {
	return ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(size)})
}

// unpaintable is a text engine whose paragraphs cannot paint themselves.
type unpaintable struct{}

func (unpaintable) BuildParagraph(*ui.ParagraphContent) ui.ParagraphLayout {
	return unpaintableLayout{}
}

type unpaintableLayout struct{ ui.ParagraphLayout }

func (unpaintableLayout) Layout(ui.ParagraphConstraints) {}
func (unpaintableLayout) MaxIntrinsicWidth() float64     { return 1 }
func (unpaintableLayout) AlphabeticBaseline() float64    { return 1 }

// latinClusters are n different letters of Roboto, from "!" on, with the control characters left out.
func latinClusters(n int) []string {
	var out []string
	for r := rune('!'); len(out) < n; r++ {
		if r >= 0x7F && r <= 0xA0 {
			continue
		}
		out = append(out, string(r))
	}
	return out
}

func TestNewGlyphSheetErrors(t *testing.T) {
	engine := proof.BundledTextEngine()
	tests := []struct {
		name      string
		engine    ui.ParagraphFactory
		style     *ui.TextStyle
		ratio     float64
		clusters  []string
		wantRoute string
	}{
		{"no engine", nil, nil, 1, []string{"A"}, "NewGlyphSheet.GlyphEngineMissing"},
		{"no cluster", engine, nil, 1, nil, "NewGlyphSheet.GlyphClustersMissing"},
		{"an empty list", engine, nil, 1, []string{}, "NewGlyphSheet.GlyphClustersMissing"},
		{"a ratio of zero", engine, nil, 0, []string{"A"}, "NewGlyphSheet.GlyphPixelRatioInvalid"},
		{"a negative ratio", engine, nil, -2, []string{"A"}, "NewGlyphSheet.GlyphPixelRatioInvalid"},
		{"a ratio that is not a number", engine, nil, math.NaN(), []string{"A"}, "NewGlyphSheet.GlyphPixelRatioInvalid"},
		{"an infinite ratio", engine, nil, math.Inf(1), []string{"A"}, "NewGlyphSheet.GlyphPixelRatioInvalid"},
		{"an engine that cannot paint", unpaintable{}, nil, 1, []string{"A"}, "NewGlyphSheet->clusterShape.measure.GlyphEngineUnsupported"},
		{"a glyph that is too large", engine, robotoStyle(5000), 4, []string{"W"}, "NewGlyphSheet->clusterShape.measure.GlyphSheetTooLarge"},
		{"a sheet that is too large", engine, robotoStyle(2000), 1, []string{"A", "B", "C", "D", "E", "F", "G", "H", "K", "M", "N", "O", "P", "Q", "R", "S", "U", "V", "W", "X", "Y", "Z", "a", "b", "d", "g"}, "NewGlyphSheet->pack.GlyphSheetTooLarge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet, err := offscreen.NewGlyphSheet(tt.engine, tt.style, tt.ratio, tt.clusters)
			if err == nil {
				sheet.Image.Dispose()
				t.Fatal("NewGlyphSheet succeeded")
			}
			if sheet != nil {
				t.Errorf("sheet = %v with an error", sheet)
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// drawRow draws a row of clusters with one DrawAtlas call, as the guide does; the canvas is in logical
// pixels, and the pen of the row starts at the baseline.
func drawRow(c *ui.Canvas, sheet *offscreen.GlyphSheet, metrics ui.FontMetrics, row []string, pen ui.Offset, fg ui.Color) {
	byCluster := make(map[string]offscreen.Glyph, len(sheet.Glyphs))
	for _, g := range sheet.Glyphs {
		byCluster[g.Cluster] = g
	}
	ratio := sheet.DevicePixelRatio
	var transforms []ui.RSTransform
	var rects []ui.Rect
	var colors []ui.Color
	for x, cluster := range row {
		g := byCluster[cluster]
		if g.Rect.IsEmpty() {
			continue
		}
		cellX := math.Round(float64(x)*metrics.ZeroAdvance*ratio) / ratio
		transforms = append(transforms, ui.RSTransform{
			Scos: float32(1 / ratio),
			Tx:   float32(pen.Dx + cellX + g.Offset.Dx/ratio),
			Ty:   float32(pen.Dy + g.Offset.Dy/ratio),
		})
		rects = append(rects, g.Rect)
		colors = append(colors, fg)
	}
	c.DrawAtlas(sheet.Image, transforms, rects, colors, ui.BlendModeModulate, nil, nil)
}

// TestGlyphSheetDrawsWithDrawAtlas draws a row with one DrawAtlas call, tinted, and compares it with the
// same row drawn by a paragraph for each cell in the same color.
func TestGlyphSheetDrawsWithDrawAtlas(t *testing.T) {
	engine := proof.BundledTextEngine()
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0)})
	d := ui.NewPlatformDispatcher(nil)
	d.SetTextEngine(engine)
	metrics, err := d.FontMetrics(style)
	if err != nil {
		t.Fatal(err)
	}
	row := []string{"G", "l", "u", "t", "t", "e", "r", " ", "0", "1", "é"}
	fg := ui.NewColor(0xFF2E7D32)
	for _, ratio := range []float64{1, 2} {
		sheet, err := offscreen.NewGlyphSheet(engine, style, ratio, row)
		if err != nil {
			t.Fatal(err)
		}
		const w, h = 220, 40
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		c.Scale(ratio, ratio)
		pen := ui.Offset{Dx: 4, Dy: 4 + math.Round(metrics.Ascent*ratio)/ratio}
		drawRow(c, sheet, metrics, row, pen, fg)
		got := renderScene(t, rec.EndRecording(), w*int(ratio), h*int(ratio))

		rec = ui.NewPictureRecorder()
		c = ui.NewCanvas(rec)
		c.Scale(ratio, ratio)
		for x, cluster := range row {
			b := ui.NewParagraphBuilder(engine, ui.NewParagraphStyle())
			b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(16.0), Color: &fg}))
			b.AddText(cluster)
			p := b.Build()
			p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
			cellX := math.Round(float64(x)*metrics.ZeroAdvance*ratio) / ratio
			c.DrawParagraph(p, ui.Offset{Dx: 4 + cellX, Dy: 4 + math.Round(metrics.Ascent*ratio)/ratio - p.AlphabeticBaseline()})
		}
		want := renderScene(t, rec.EndRecording(), w*int(ratio), h*int(ratio))

		worst, bad := 0, 0
		for i := range got.Pix {
			if diff := absDiff(got.Pix[i], want.Pix[i]); diff > 0 {
				bad++
				worst = max(worst, diff)
			}
		}
		if worst > 2 {
			t.Errorf("ratio %v: the atlas row differs from the paragraphs by %d levels in %d channels", ratio, worst, bad)
		}
		if bad == len(got.Pix) {
			t.Errorf("ratio %v: nothing matches", ratio)
		}
		sheet.Image.Dispose()
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func renderScene(t *testing.T, p *ui.Picture, w, h int) *image.RGBA {
	t.Helper()
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, p)
	img, err := soft.Render(sb.Build(), w, h)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// BenchmarkGlyphSheet rasterizes 256 clusters: with NewGlyphSheet, and the way an app had to before it, with
// a paragraph for each cluster, drawn on a picture and read back with Picture.ToImage.
func BenchmarkGlyphSheet(b *testing.B) {
	engine := proof.BundledTextEngine()
	style := robotoStyle(14)
	clusters := latinClusters(256)
	d := ui.NewPlatformDispatcher(nil)
	d.SetTextEngine(engine)
	metrics, err := d.FontMetrics(style)
	if err != nil {
		b.Fatal(err)
	}
	cellW, cellH := int(math.Ceil(metrics.ZeroAdvance*2)), int(math.Ceil(metrics.Height))
	white := ui.NewColor(0xFFFFFFFF)
	b.Run("NewGlyphSheet", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sheet, err := offscreen.NewGlyphSheet(engine, style, 1, clusters)
			if err != nil {
				b.Fatal(err)
			}
			sheet.Image.Dispose()
		}
	})
	b.Run("ParagraphAndToImage", func(b *testing.B) {
		rasterizer := offscreen.NewRasterizer()
		const perRow = 16
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// The sprite sheet of a grid of cells, filled one cluster at a time.
			sheet := make([]byte, 4*perRow*cellW*perRow*cellH)
			for k, cluster := range clusters {
				pb := ui.NewParagraphBuilder(engine, ui.NewParagraphStyle())
				pb.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(14.0), Color: &white}))
				pb.AddText(cluster)
				p := pb.Build()
				p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
				rec := ui.NewPictureRecorder()
				ui.NewCanvas(rec).DrawParagraph(p, ui.Offset{})
				img, err := rec.EndRecording().ToImage(cellW, cellH, ui.ToImageOpts{Rasterizer: rasterizer})
				if err != nil {
					b.Fatal(err)
				}
				px, py := (k%perRow)*cellW, (k/perRow)*cellH
				for y := 0; y < cellH; y++ {
					copy(sheet[4*((py+y)*perRow*cellW+px):], img.Pixels()[4*y*cellW:4*(y+1)*cellW])
				}
				img.Dispose()
			}
		}
	})
}
