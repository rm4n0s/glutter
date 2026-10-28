package layout_test

import (
	"context"
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
)

// referenceOf draws the cluster the way an app does, with a paragraph under a canvas scaled by the ratio,
// its origin on the whole pixel origin, in white, and returns the pixels with the paragraph's baseline
// in pixels below the origin.
func referenceOf(t *testing.T, e *layout.Engine, cluster string, style ui.TextStyleOpts, ratio float64, origin image.Point, w, h int) (*image.RGBA, int) {
	t.Helper()
	white := ui.NewColor(0xFFFFFFFF)
	style.Color = &white
	b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle())
	b.PushStyle(ui.NewTextStyle(style))
	b.AddText(cluster)
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.Translate(float64(origin.X), float64(origin.Y))
	c.Scale(ratio, ratio)
	c.DrawParagraph(p, ui.Offset{})
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := soft.Render(sb.Build(), w, h)
	if err != nil {
		t.Fatal(err)
	}
	return img, int(math.Round(ratio * p.AlphabeticBaseline()))
}

func sheetPixels(sheet *offscreen.GlyphSheet) *image.RGBA {
	return &image.RGBA{Pix: sheet.Image.Pixels(), Stride: 4 * sheet.Image.Width(), Rect: image.Rect(0, 0, sheet.Image.Width(), sheet.Image.Height())}
}

func rectOf(r ui.Rect) image.Rectangle {
	return image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom))
}

// systemEngine is an engine with Roboto and the fonts of the machine; the test is skipped when the machine
// has no font for the rune that the bundled ones lack.
func systemEngine(t *testing.T, r rune) *layout.Engine {
	t.Helper()
	c := text.NewCollection()
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := c.Fallback(context.Background(), fonts.RobotoFamily, 400, false, "", r); err != nil || !ok {
		t.Skipf("the system has no font for %q", r)
	}
	return layout.NewEngine(context.Background(), c)
}

func bundledEngine(t *testing.T) *layout.Engine {
	t.Helper()
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		t.Fatal(err)
	}
	return layout.NewEngine(context.Background(), c)
}

type noSystemFonts struct{}

func (noSystemFonts) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// TestGlyphSheetMatchesParagraphs checks each cluster of a sheet against the same cluster drawn by a
// paragraph with the software renderer at a pixel-aligned origin: the pixels of the rectangle are the same, and
// the paragraph has no pixel outside it.
func TestGlyphSheetMatchesParagraphs(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	roboto := func(size float64) ui.TextStyleOpts {
		return ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(size), Color: &red}
	}
	tests := []struct {
		name     string
		engine   func(t *testing.T) *layout.Engine
		style    ui.TextStyleOpts
		ratio    float64
		clusters []string
		colored  bool   // the clusters draw with colors of their own
		empty    string // the cluster that draws nothing, if any
	}{
		{"ASCII", func(t *testing.T) *layout.Engine { return bundledEngine(t) }, roboto(14), 1,
			[]string{"A", "g", "W", "i", ".", "@", "0", "j", "Q"}, false, ""},
		{"ASCII at a ratio of 1.5", func(t *testing.T) *layout.Engine { return bundledEngine(t) }, roboto(13), 1.5,
			[]string{"A", "g", "W", "i", ".", "@", "0", "j", "Q"}, false, ""},
		{"ASCII at a ratio of 2 and a large size", func(t *testing.T) *layout.Engine { return bundledEngine(t) }, roboto(33), 2,
			[]string{"A", "g", "W", "R", "&", "%"}, false, ""},
		{"accented letters, composed and with a mark", func(t *testing.T) *layout.Engine { return bundledEngine(t) }, roboto(20), 1.25,
			[]string{"é", "é", "ñ", "Å", "ü"}, false, ""},
		{"a space and letters", func(t *testing.T) *layout.Engine { return bundledEngine(t) }, roboto(16), 1,
			[]string{" ", "x", " ", "y"}, false, " "},
		{"a CJK character from a fallback font", func(t *testing.T) *layout.Engine { return systemEngine(t, '漢') }, roboto(18), 1,
			[]string{"漢", "字", "A"}, false, ""},
		{"a color emoji", layout.CBDTEngine, ui.TextStyleOpts{FontFamily: layout.CBDTFamily, FontSize: ui.Ptr(20.0), Color: &red}, 1,
			[]string{"\U0001F600", "\U0001F601", "A"}, true, ""},
		{"a color emoji at a ratio of 2", layout.CBDTEngine, ui.TextStyleOpts{FontFamily: layout.CBDTFamily, FontSize: ui.Ptr(12.0), Color: &red}, 2,
			[]string{"\U0001F600", "A"}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.engine(t)
			sheet, err := offscreen.NewGlyphSheet(e, ui.NewTextStyle(tt.style), tt.ratio, tt.clusters)
			if err != nil {
				t.Fatalf("NewGlyphSheet: %v", err)
			}
			defer sheet.Image.Dispose()
			if len(sheet.Glyphs) != len(tt.clusters) {
				t.Fatalf("%d glyphs for %d clusters", len(sheet.Glyphs), len(tt.clusters))
			}
			pix := sheetPixels(sheet)
			bounds := pix.Rect
			covered := map[image.Point]bool{}
			var rects []image.Rectangle
			for i, g := range sheet.Glyphs {
				if g.Cluster != tt.clusters[i] {
					t.Errorf("glyph %d is %q, want %q", i, g.Cluster, tt.clusters[i])
				}
				// The reference: a paragraph drawn in white with its origin on a whole pixel.
				const w, h = 160, 160
				origin := image.Pt(40, 20)
				ref, pen := referenceOf(t, e, g.Cluster, tt.style, tt.ratio, origin, w, h)
				pPen := image.Pt(origin.X, origin.Y+pen)

				if g.Advance <= 0 {
					t.Errorf("%q: advance %v", g.Cluster, g.Advance)
				}
				if (g.Cluster == tt.empty) != g.Rect.IsEmpty() {
					t.Errorf("%q: rect %v, empty cluster is %q", g.Cluster, g.Rect, tt.empty)
				}
				if g.Colored != (tt.colored && !g.Rect.IsEmpty()) {
					t.Errorf("%q: Colored = %v", g.Cluster, g.Colored)
				}
				r := rectOf(g.Rect)
				if r.Empty() {
					if ink := inkOf(ref); !ink.Empty() {
						t.Errorf("%q: the sheet has no rectangle but a paragraph draws in %v", g.Cluster, ink)
					}
					continue
				}
				if !r.In(bounds) {
					t.Errorf("%q: rect %v is outside the image %v", g.Cluster, r, bounds)
				}
				for _, prev := range rects {
					if prev.Overlaps(r) {
						t.Errorf("%q: rect %v overlaps %v", g.Cluster, r, prev)
					}
				}
				rects = append(rects, r)
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						covered[image.Pt(x, y)] = true
					}
				}
				// The paragraph's ink is where the rectangle says, relative to the pen.
				from := image.Pt(pPen.X+int(g.Offset.Dx), pPen.Y+int(g.Offset.Dy))
				if g.Offset.Dx != math.Trunc(g.Offset.Dx) || g.Offset.Dy != math.Trunc(g.Offset.Dy) {
					t.Errorf("%q: offset %v is not whole pixels", g.Cluster, g.Offset)
				}
				if ink := inkOf(ref); !ink.Empty() && !ink.In(image.Rectangle{Min: from, Max: from.Add(r.Size())}) {
					t.Errorf("%q: the paragraph draws in %v, outside the rect that starts at %v", g.Cluster, ink, from)
				}
				for y := 0; y < r.Dy(); y++ {
					for x := 0; x < r.Dx(); x++ {
						got, want := pix.RGBAAt(r.Min.X+x, r.Min.Y+y), ref.RGBAAt(from.X+x, from.Y+y)
						if got != want {
							t.Fatalf("%q: pixel (%d,%d) of the glyph is %v in the sheet and %v in the paragraph", g.Cluster, x, y, got, want)
						}
					}
				}
				// Outline glyphs are white coverage.
				if !g.Colored {
					for y := r.Min.Y; y < r.Max.Y; y++ {
						for x := r.Min.X; x < r.Max.X; x++ {
							c := pix.RGBAAt(x, y)
							if c.R != c.A || c.G != c.A || c.B != c.A {
								t.Fatalf("%q: pixel (%d,%d) is %v, not white coverage", g.Cluster, x, y, c)
							}
						}
					}
				}
			}
			// Nothing is drawn outside the rectangles.
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					if c := pix.RGBAAt(x, y); !covered[image.Pt(x, y)] && c.A != 0 {
						t.Fatalf("pixel (%d,%d) is %v, outside every rectangle", x, y, c)
					}
				}
			}
		})
	}
}

// inkOf is the box of the pixels that are not transparent.
func inkOf(img *image.RGBA) image.Rectangle {
	var r image.Rectangle
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if img.RGBAAt(x, y).A != 0 {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

// TestGlyphSheetAdvanceAndDuplicates checks the advance against a paragraph and that a cluster given
// twice is drawn once.
func TestGlyphSheetAdvanceAndDuplicates(t *testing.T) {
	e := bundledEngine(t)
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(15.0)})
	clusters := []string{"M", "i", "M", " ", "i", ""}
	sheet, err := offscreen.NewGlyphSheet(e, style, 2, clusters)
	if err != nil {
		t.Fatal(err)
	}
	defer sheet.Image.Dispose()
	if sheet.DevicePixelRatio != 2 {
		t.Errorf("DevicePixelRatio = %v", sheet.DevicePixelRatio)
	}
	for i, g := range sheet.Glyphs {
		want := 0.0
		if g.Cluster != "" {
			b := ui.NewParagraphBuilder(e, ui.NewParagraphStyle())
			b.PushStyle(style)
			b.AddText(g.Cluster)
			want = 2 * b.Build().MaxIntrinsicWidth()
		}
		if math.Abs(g.Advance-want) > 1e-9 {
			t.Errorf("%d %q: advance %v, want %v", i, g.Cluster, g.Advance, want)
		}
	}
	if sheet.Glyphs[0] != sheet.Glyphs[2] || sheet.Glyphs[1] != sheet.Glyphs[4] {
		t.Errorf("a cluster given twice has two places: %+v", sheet.Glyphs)
	}
	if !sheet.Glyphs[3].Rect.IsEmpty() || !sheet.Glyphs[5].Rect.IsEmpty() || sheet.Glyphs[5].Advance != 0 {
		t.Errorf("the space and the empty cluster draw nothing: %+v %+v", sheet.Glyphs[3], sheet.Glyphs[5])
	}
}

// TestGlyphSheetOfClustersThatDrawNothing checks that a sheet of spaces is a transparent pixel.
func TestGlyphSheetOfClustersThatDrawNothing(t *testing.T) {
	e := bundledEngine(t)
	sheet, err := offscreen.NewGlyphSheet(e, ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily}), 1, []string{" ", "\t"})
	if err != nil {
		t.Fatal(err)
	}
	defer sheet.Image.Dispose()
	if sheet.Image.Width() != 1 || sheet.Image.Height() != 1 || inkOf(sheetPixels(sheet)).Size() != (image.Point{}) {
		t.Errorf("the sheet is %dx%d with ink %v", sheet.Image.Width(), sheet.Image.Height(), inkOf(sheetPixels(sheet)))
	}
}
