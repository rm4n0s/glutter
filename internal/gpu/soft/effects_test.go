package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func alphaAt(img *image.RGBA, x, y int) int { return int(px(img, x, y)[3]) }

func TestGaussianKernel(t *testing.T) {
	for _, sigma := range []float64{0.5, 1, 2.5, 8} {
		k := gaussianKernel(sigma)
		if want := 2*int(math.Ceil(3*sigma)) + 1; len(k) != want {
			t.Errorf("sigma %v: %d taps, want %d", sigma, len(k), want)
		}
		var sum float64
		for i, w := range k {
			sum += float64(w)
			if mirror := k[len(k)-1-i]; w != mirror {
				t.Errorf("sigma %v: kernel is not symmetric at %d", sigma, i)
			}
		}
		if math.Abs(sum-1) > 1e-5 {
			t.Errorf("sigma %v: weights sum to %v", sigma, sum)
		}
		if k[len(k)/2] < k[len(k)/2-1] {
			t.Errorf("sigma %v: the middle is not the largest weight", sigma)
		}
	}
	for _, sigma := range []float64{0, -1, 0.005, math.NaN(), math.Inf(1)} {
		if gaussianKernel(sigma) != nil {
			t.Errorf("sigma %v should not blur", sigma)
		}
	}
}

func TestBlurAgainstTheClosedForm(t *testing.T) {
	// The blur of a pixel-aligned rectangle is a product of error functions.
	const sigma = 4.0
	rect := image.Rect(20, 20, 60, 50)
	src := &Mask{Rect: rect, Cov: ones(rect.Dx() * rect.Dy())}
	out := image.Rect(0, 0, 80, 70)
	got := blurredMask(src, sigma, sigma, out)
	span := func(v, lo, hi float64) float64 {
		return 0.5 * (math.Erf((v-lo)/(sigma*math.Sqrt2)) - math.Erf((v-hi)/(sigma*math.Sqrt2)))
	}
	var worst float64
	for y := out.Min.Y; y < out.Max.Y; y++ {
		for x := out.Min.X; x < out.Max.X; x++ {
			want := span(float64(x)+0.5, 20, 60) * span(float64(y)+0.5, 20, 50)
			worst = math.Max(worst, math.Abs(float64(got.At(x, y))-want))
		}
	}
	if worst > 0.01 {
		t.Errorf("largest difference from the closed form is %v, want under 0.01", worst)
	}
	// The blur keeps the total coverage: the area of the rectangle.
	if sum := got.Sum(); math.Abs(sum-float64(rect.Dx()*rect.Dy())) > 1 {
		t.Errorf("total coverage = %v, want %v", sum, rect.Dx()*rect.Dy())
	}
}

func TestBlurDataTileModes(t *testing.T) {
	// A row of 8 pixels, the first one set. The blur leaks across the edge in
	// each mode as the tile mode says.
	row := []float32{1, 0, 0, 0, 0, 0, 0, 0}
	at := func(tile ui.TileMode) []float32 { return blurData(row, 8, 1, 1, 1, 0, tile) }
	sum := func(v []float32) float64 {
		var s float64
		for _, x := range v {
			s += float64(x)
		}
		return s
	}
	if s := sum(at(ui.TileModeDecal)); s >= 0.999 {
		t.Errorf("decal keeps %v of the coverage, want a loss at the edge", s)
	}
	for _, tile := range []ui.TileMode{ui.TileModeRepeated, ui.TileModeMirror} {
		if s := sum(at(tile)); math.Abs(s-1) > 1e-4 {
			t.Errorf("%v keeps %v of the coverage, want 1", tile, s)
		}
	}
	// Clamp repeats the edge pixel, so what is set at the edge counts extra.
	if s := sum(at(ui.TileModeClamp)); s <= 1.01 {
		t.Errorf("clamp holds %v, want more than 1 for a set edge pixel", s)
	}
	// Repeated and mirrored put the leaked weight on the other side or back.
	if got := at(ui.TileModeRepeated); got[7] < 0.1 {
		t.Errorf("repeated: the last pixel = %v, want leaked weight", got[7])
	}
	if got := at(ui.TileModeClamp); got[0] < at(ui.TileModeDecal)[0] {
		t.Errorf("clamp should hold more at the edge than decal: %v vs %v", got[0], at(ui.TileModeDecal)[0])
	}
	// No blur is the data itself.
	if got := blurData(row, 8, 1, 1, 0, 0, ui.TileModeDecal); &got[0] != &row[0] {
		t.Error("a zero sigma should return the data unchanged")
	}
}

func TestMaskFilterStyles(t *testing.T) {
	draw := func(style ui.BlurStyle) *image.RGBA {
		return renderOf(t, 60, 40, func(c *ui.Canvas) {
			p := paint(0xFF000000)
			p.SetMaskFilter(ui.NewMaskFilterBlur(style, 3))
			c.DrawRect(rectAt(15, 10, 30, 20), p)
		})
	}
	// inside is the middle of the rect, edge a pixel just outside it, far well outside.
	type probe struct{ inside, justOutside, justInside, far int }
	get := func(img *image.RGBA) probe {
		return probe{alphaAt(img, 30, 20), alphaAt(img, 13, 20), alphaAt(img, 16, 20), alphaAt(img, 2, 2)}
	}
	normal := get(draw(ui.BlurStyleNormal))
	if normal.inside < 250 || normal.justOutside < 20 || normal.justOutside > 200 || normal.justInside < 100 || normal.justInside > 235 || normal.far != 0 {
		t.Errorf("normal = %+v", normal)
	}
	solid := get(draw(ui.BlurStyleSolid))
	if solid.inside != 255 || solid.justInside != 255 || solid.justOutside != normal.justOutside || solid.far != 0 {
		t.Errorf("solid = %+v, normal = %+v", solid, normal)
	}
	outer := get(draw(ui.BlurStyleOuter))
	if outer.inside != 0 || outer.justInside != 0 || outer.justOutside != normal.justOutside || outer.far != 0 {
		t.Errorf("outer = %+v", outer)
	}
	inner := get(draw(ui.BlurStyleInner))
	if inner.inside < 250 || inner.justOutside != 0 || inner.justInside != normal.justInside || inner.far != 0 {
		t.Errorf("inner = %+v", inner)
	}
	if got := get(draw(ui.BlurStyleUnspecified)); got != normal {
		t.Errorf("unspecified = %+v, want the normal style %+v", got, normal)
	}
}

func TestMaskFilterSigmaFollowsTheTransform(t *testing.T) {
	// A blur of sigma 2 drawn under a 2x scale blurs like sigma 4 on a rect twice as big.
	small := renderOf(t, 80, 40, func(c *ui.Canvas) {
		p := paint(0xFF000000)
		p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 4))
		c.DrawRect(rectAt(20, 10, 40, 20), p)
	})
	scaled := renderOf(t, 80, 40, func(c *ui.Canvas) {
		c.Scale(2, 2)
		p := paint(0xFF000000)
		p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 2))
		c.DrawRect(rectAt(10, 5, 20, 10), p)
	})
	for x := 10; x < 70; x++ {
		if d := absInt(alphaAt(small, x, 20) - alphaAt(scaled, x, 20)); d > 2 {
			t.Fatalf("x=%d: %d vs %d", x, alphaAt(small, x, 20), alphaAt(scaled, x, 20))
		}
	}
}

func TestMaskFilterBlursOntoTheSurfaceFromOutside(t *testing.T) {
	img := renderOf(t, 20, 20, func(c *ui.Canvas) {
		p := paint(0xFF000000)
		p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 4))
		c.DrawRect(rectAt(-30, 5, 28, 10), p) // the rect ends 2 pixels left of the surface
	})
	if got := alphaAt(img, 0, 10); got < 20 {
		t.Errorf("alpha at the edge = %d, want some blur to reach it", got)
	}
	if got := alphaAt(img, 19, 10); got != 0 {
		t.Errorf("alpha far away = %d", got)
	}
}

func TestMaskFilterWithClipAndStroke(t *testing.T) {
	img := renderOf(t, 40, 40, func(c *ui.Canvas) {
		c.ClipRect(rectAt(0, 0, 20, 40))
		p := stroked(0xFF000000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter)
		p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 2))
		c.DrawCircle(ui.Offset{Dx: 20, Dy: 20}, 12, p)
	})
	if alphaAt(img, 8, 20) == 0 {
		t.Error("the stroke is missing inside the clip")
	}
	if alphaAt(img, 31, 20) != 0 {
		t.Error("the blur was not clipped")
	}
}

func TestImageFilterBlurTileModes(t *testing.T) {
	// A layer filled to its left edge: how it blurs there depends on the tile mode.
	draw := func(tile ui.TileMode) *image.RGBA {
		return renderOf(t, 40, 20, func(c *ui.Canvas) {
			p := ui.NewPaint()
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 3, SigmaY: 0, TileMode: tile}))
			c.SaveLayer(nil, p)
			c.DrawRect(rectAt(0, 0, 20, 20), paint(0xFFFF0000))
			c.Restore()
		})
	}
	decal, clamp := draw(ui.TileModeDecal), draw(ui.TileModeClamp)
	if got := alphaAt(decal, 0, 10); got > 200 {
		t.Errorf("decal at the left edge = %d, want a fade", got)
	}
	if got := alphaAt(clamp, 0, 10); got < 250 {
		t.Errorf("clamp at the left edge = %d, want it held", got)
	}
	// Both blur the right edge of the rect the same way.
	if d := absInt(alphaAt(decal, 20, 10) - alphaAt(clamp, 20, 10)); d > 3 {
		t.Errorf("the free edge differs by %d", d)
	}
	if got := alphaAt(clamp, 20, 10); got < 100 || got > 160 {
		t.Errorf("alpha at the rect's edge = %d, want about half", got)
	}
	// The default of a layer's blur is clamp: unspecified matches it.
	if d := absInt(alphaAt(draw(ui.TileModeUnspecified), 0, 10) - alphaAt(clamp, 0, 10)); d > 0 {
		t.Errorf("unspecified differs from clamp by %d", d)
	}
}

func TestImageFilterOnShapes(t *testing.T) {
	// A paint with an image filter draws the shape blurred, with the paint's blend mode.
	blur := ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 3, SigmaY: 3})
	p := paint(0xFF0000FF)
	p.SetImageFilter(blur)
	img := renderOf(t, 40, 40, func(c *ui.Canvas) { c.DrawRect(rectAt(10, 10, 20, 20), p) })
	if got := alphaAt(img, 20, 20); got < 250 {
		t.Errorf("inside = %d", got)
	}
	if got := alphaAt(img, 9, 20); got < 30 || got > 150 {
		t.Errorf("just outside = %d, want a soft edge", got)
	}
	if got := alphaAt(img, 2, 2); got != 0 {
		t.Errorf("far = %d", got)
	}
	// With dstOut the blurred shape erases, with soft edges.
	q := paint(0xFF000000)
	q.SetImageFilter(blur)
	q.SetBlendMode(ui.BlendModeDstOut)
	img = renderOf(t, 40, 40, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 40, 40), paint(0xFFFF0000))
		c.DrawRect(rectAt(10, 10, 20, 20), q)
	})
	if got := alphaAt(img, 20, 20); got > 5 {
		t.Errorf("cleared = %d", got)
	}
	if got := alphaAt(img, 9, 20); got < 50 || got > 220 {
		t.Errorf("soft edge of the hole = %d", got)
	}
	if got := alphaAt(img, 2, 2); got != 255 {
		t.Errorf("untouched = %d", got)
	}
	// The mode clear takes the whole of every pixel the filtered shape reaches,
	// and nothing beyond it, as in Skia.
	z := paint(0xFF000000)
	z.SetImageFilter(blur)
	z.SetBlendMode(ui.BlendModeClear)
	img = renderOf(t, 40, 40, func(c *ui.Canvas) {
		c.DrawRect(rectAt(0, 0, 40, 40), paint(0xFFFF0000))
		c.DrawRect(rectAt(10, 10, 20, 20), z)
	})
	if alphaAt(img, 9, 20) != 0 || alphaAt(img, 2, 2) != 255 || alphaAt(img, 20, 20) != 0 {
		t.Errorf("clear: %d %d %d", alphaAt(img, 9, 20), alphaAt(img, 2, 2), alphaAt(img, 20, 20))
	}
	// Images, points, vertices and atlases take the filter too.
	for name, draw := range map[string]func(c *ui.Canvas, p *ui.Paint){
		"image": func(c *ui.Canvas, p *ui.Paint) { c.DrawImage(quads(), ui.Offset{Dx: 16, Dy: 16}, p) },
		"points": func(c *ui.Canvas, p *ui.Paint) {
			p.SetStyle(ui.PaintingStyleStroke)
			p.SetStrokeWidth(6)
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 20, Dy: 20}}, p)
		},
		"vertices": func(c *ui.Canvas, p *ui.Paint) {
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pts(10, 10, 30, 10, 10, 30)), ui.BlendModeSrcOver, p)
		},
		"nine": func(c *ui.Canvas, p *ui.Paint) {
			c.DrawImageNine(quads(), rectAt(1, 1, 2, 2), rectAt(12, 12, 16, 16), p)
		},
	} {
		t.Run(name, func(t *testing.T) {
			fp := withQuality(ui.FilterQualityNone)
			fp.SetImageFilter(blur)
			img := renderOf(t, 40, 40, func(c *ui.Canvas) { draw(c, fp) })
			sharp := renderOf(t, 40, 40, func(c *ui.Canvas) { draw(c, withQuality(ui.FilterQualityNone)) })
			if coverageSum(img) <= coverageSum(sharp)*0.9 {
				t.Errorf("blurred coverage %v is below the sharp %v", coverageSum(img), coverageSum(sharp))
			}
			// The blur spreads the coverage outside the sharp drawing.
			spread := false
			for y := 0; y < 40 && !spread; y++ {
				for x := 0; x < 40; x++ {
					if alphaAt(sharp, x, y) == 0 && alphaAt(img, x, y) > 20 {
						spread = true
						break
					}
				}
			}
			if !spread {
				t.Error("the filter did not spread the drawing")
			}
		})
	}
}

func TestMorphology(t *testing.T) {
	dot := func(c *ui.Canvas, f ui.ImageFilter) {
		c.SaveLayer(nil, func() *ui.Paint { p := ui.NewPaint(); p.SetImageFilter(f); return p }())
		c.DrawRect(rectAt(10, 10, 1, 1), paint(0xFF000000))
		c.Restore()
	}
	dilated := renderOf(t, 21, 21, func(c *ui.Canvas) {
		dot(c, ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2}))
	})
	if got := coverageSum(dilated); got != 25 {
		t.Errorf("a dot dilated by 2 covers %v pixels, want 25", got)
	}
	if alphaAt(dilated, 8, 8) != 255 || alphaAt(dilated, 12, 12) != 255 || alphaAt(dilated, 7, 10) != 0 {
		t.Error("the dilated square is not 5x5 around the dot")
	}
	wide := renderOf(t, 21, 21, func(c *ui.Canvas) {
		dot(c, ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 3, RadiusY: 0}))
	})
	if got := coverageSum(wide); got != 7 {
		t.Errorf("a dot dilated by 3 on x covers %v pixels, want 7", got)
	}
	// Erode: a 5x5 square becomes 3x3.
	eroded := renderOf(t, 21, 21, func(c *ui.Canvas) {
		p := ui.NewPaint()
		p.SetImageFilter(ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 1}))
		c.SaveLayer(nil, p)
		c.DrawRect(rectAt(8, 8, 5, 5), paint(0xFF000000))
		c.Restore()
	})
	if got := coverageSum(eroded); got != 9 {
		t.Errorf("a 5x5 square eroded by 1 covers %v pixels, want 9", got)
	}
	// The radius follows the transform's scale.
	scaled := renderOf(t, 40, 40, func(c *ui.Canvas) {
		c.Scale(2, 2)
		p := ui.NewPaint()
		p.SetImageFilter(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 1}))
		c.SaveLayer(nil, p)
		c.DrawRect(rectAt(5, 5, 1, 1), paint(0xFF000000)) // a 2x2 device square
		c.Restore()
	})
	if got := coverageSum(scaled); got != 36 {
		t.Errorf("under a 2x scale: %v pixels, want 36 (a 2x2 square grown by 2 each side)", got)
	}
	// No radius, no change.
	same := renderOf(t, 21, 21, func(c *ui.Canvas) { dot(c, ui.NewImageFilterDilate()) })
	if got := coverageSum(same); got != 1 {
		t.Errorf("zero radius covers %v pixels", got)
	}
}

func TestMatrixAndComposeFilters(t *testing.T) {
	square := func(c *ui.Canvas, f ui.ImageFilter) {
		p := ui.NewPaint()
		p.SetImageFilter(f)
		c.SaveLayer(nil, p)
		c.DrawRect(rectAt(4, 4, 4, 4), paint(0xFFFF0000))
		c.Restore()
	}
	shift := ui.NewMatrix4Identity()
	shift.Translate(6, 2, 0)
	moved := renderOf(t, 24, 16, func(c *ui.Canvas) {
		square(c, ui.NewImageFilterMatrix(shift, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}))
	})
	if alphaAt(moved, 11, 7) != 255 || alphaAt(moved, 5, 5) != 0 || alphaAt(moved, 10, 6) != 255 {
		t.Errorf("translated square: %d %d %d", alphaAt(moved, 11, 7), alphaAt(moved, 5, 5), alphaAt(moved, 10, 6))
	}
	grow := ui.NewMatrix4Identity()
	grow.Scale(2, 2, 1)
	big := renderOf(t, 24, 16, func(c *ui.Canvas) {
		square(c, ui.NewImageFilterMatrix(grow, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}))
	})
	if got := coverageSum(big); got != 64 {
		t.Errorf("a 4x4 square scaled by 2 covers %v pixels, want 64", got)
	}
	// The matrix acts in the canvas's coordinates: under a 2x scale a translation of 3 moves 6 pixels.
	scaled := renderOf(t, 40, 20, func(c *ui.Canvas) {
		c.Scale(2, 2)
		m := ui.NewMatrix4Identity()
		m.Translate(3, 0, 0)
		p := ui.NewPaint()
		p.SetImageFilter(ui.NewImageFilterMatrix(m, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}))
		c.SaveLayer(nil, p)
		c.DrawRect(rectAt(1, 1, 2, 2), paint(0xFFFF0000))
		c.Restore()
	})
	if alphaAt(scaled, 2, 2) != 0 || alphaAt(scaled, 9, 3) != 255 || alphaAt(scaled, 14, 3) != 0 {
		t.Errorf("the matrix did not follow the canvas scale: %d %d %d", alphaAt(scaled, 2, 2), alphaAt(scaled, 9, 3), alphaAt(scaled, 14, 3))
	}
	// Compose applies the inner filter first: dilating a dot and then blurring
	// is wider than blurring and then dilating by nothing.
	dil := ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2})
	blur := ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 1, SigmaY: 1})
	inner := renderOf(t, 24, 16, func(c *ui.Canvas) { square(c, ui.NewImageFilterCompose(blur, dil)) })
	if got := alphaAt(inner, 2, 8); got == 0 {
		t.Errorf("dilate then blur: alpha at 2 pixels outside = %d, want some", got)
	}
	// A color filter is an image filter too.
	gray := renderOf(t, 8, 8, func(c *ui.Canvas) {
		p := ui.NewPaint()
		p.SetImageFilter(ui.NewColorFilterSaturation(0))
		c.SaveLayer(nil, p)
		c.DrawRect(rectAt(0, 0, 8, 8), paint(0xFFFF0000))
		c.Restore()
	})
	if got := px(gray, 3, 3); got[0] != got[1] || got[1] != got[2] || got[0] < 50 {
		t.Errorf("red through a saturation filter = %v, want gray", got)
	}
}

func TestShadows(t *testing.T) {
	card := ui.NewPath()
	card.AddRRect(ui.NewRRectFromRectXY(rectAt(30, 30, 60, 40), 6, 6))
	shadow := func(elevation float64, transparent bool, scale float64) *image.RGBA {
		return renderOf(t, int(120*scale), int(100*scale), func(c *ui.Canvas) {
			white(c)
			c.Scale(scale, scale)
			c.DrawShadow(card, ui.NewColor(0xFF000000), elevation, transparent)
		})
	}
	dark := func(img *image.RGBA, x, y int) int { return 255 - int(px(img, x, y)[0]) }

	if got := coverageSum(shadow(0, false, 1)); math.Abs(got-120*100) > 0.5 {
		t.Errorf("elevation 0 changed the picture")
	}
	e4 := shadow(4, false, 1)
	// The spot shadow falls below the card: darker under than above, and equal left and right.
	if below, above := dark(e4, 60, 74), dark(e4, 60, 26); below <= above {
		t.Errorf("below = %d, above = %d: the shadow should fall down", below, above)
	}
	if l, r := dark(e4, 25, 50), dark(e4, 94, 50); absInt(l-r) > 2 {
		t.Errorf("left %d and right %d should match", l, r)
	}
	// An opaque occluder is not shadowed inside.
	if got := dark(e4, 60, 50); got != 0 {
		t.Errorf("inside the opaque occluder = %d, want 0", got)
	}
	trans := shadow(4, true, 1)
	if got := dark(trans, 60, 50); got == 0 {
		t.Error("a transparent occluder shows its shadow inside")
	}
	// The darkest the shadow gets is the sum of the two alphas, 0.039 + 0.25.
	var peak int
	for y := 0; y < 100; y++ {
		for x := 0; x < 120; x++ {
			peak = max(peak, dark(e4, x, y))
		}
	}
	if peak > 80 || peak < 30 {
		t.Errorf("darkest shadow pixel = %d of 255, want at most 0.289 * 255 = 74", peak)
	}
	// A higher card casts a wider and lower shadow.
	e12 := shadow(12, false, 1)
	reach := func(img *image.RGBA) int {
		n := 0
		for y := 70; y < 100; y++ {
			if dark(img, 60, y) > 2 {
				n++
			}
		}
		return n
	}
	if reach(e12) <= reach(e4) {
		t.Errorf("shadow reach: elevation 12 gives %d, elevation 4 gives %d", reach(e12), reach(e4))
	}
	e1 := shadow(1, false, 1)
	if reach(e1) >= reach(e4) {
		t.Errorf("shadow reach: elevation 1 gives %d, elevation 4 gives %d", reach(e1), reach(e4))
	}
	// Under a 2x scale (a device pixel ratio of 2) the shadow is 2x as big.
	e4x2 := shadow(4, false, 2)
	if a, b := coverageDark(e4), coverageDark(e4x2); b < a*3 || b > a*5 {
		t.Errorf("total shadow at 2x = %v, at 1x = %v, want about 4x", b, a)
	}
	// The shadow color tints it, and its alpha scales it.
	red := renderOf(t, 120, 100, func(c *ui.Canvas) {
		white(c)
		c.DrawShadow(card, ui.NewColor(0xFFFF0000), 8, false)
	})
	if p := px(red, 60, 76); p[1] >= p[0] || p[2] >= p[0] {
		t.Errorf("a red shadow = %v", p)
	}
	faint := renderOf(t, 120, 100, func(c *ui.Canvas) {
		white(c)
		c.DrawShadow(card, ui.NewColor(0x40000000), 8, false)
	})
	strong := renderOf(t, 120, 100, func(c *ui.Canvas) {
		white(c)
		c.DrawShadow(card, ui.NewColor(0xFF000000), 8, false)
	})
	if coverageDark(faint) >= coverageDark(strong)/2 {
		t.Errorf("a quarter alpha shadow = %v, a full one = %v", coverageDark(faint), coverageDark(strong))
	}
	// Any path casts one: a circle.
	circle := ui.NewPath()
	circle.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 60, Dy: 50}, 20))
	disc := renderOf(t, 120, 100, func(c *ui.Canvas) {
		white(c)
		c.DrawShadow(circle, ui.NewColor(0xFF000000), 6, false)
	})
	if dark(disc, 60, 74) == 0 || dark(disc, 60, 50) != 0 {
		t.Errorf("circle shadow below = %d, inside = %d", dark(disc, 60, 74), dark(disc, 60, 50))
	}
	// A shadow clipped, or off the surface, or of an empty path, does not fail.
	for name, draw := range map[string]func(c *ui.Canvas){
		"clipped": func(c *ui.Canvas) {
			c.ClipRect(rectAt(0, 0, 40, 40))
			c.DrawShadow(card, ui.NewColor(0xFF000000), 6, false)
		},
		"far away": func(c *ui.Canvas) { c.Translate(1000, 1000); c.DrawShadow(card, ui.NewColor(0xFF000000), 6, false) },
		"empty":    func(c *ui.Canvas) { c.DrawShadow(ui.NewPath(), ui.NewColor(0xFF000000), 6, false) },
		"too high": func(c *ui.Canvas) { c.DrawShadow(card, ui.NewColor(0xFF000000), 700, false) },
	} {
		if _, err := Render(sceneOf(ui.Offset{}, draw), 120, 100); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// coverageDark adds up how dark a picture on white is.
func coverageDark(img *image.RGBA) float64 {
	var s float64
	for i := 0; i < len(img.Pix); i += 4 {
		s += float64(255-img.Pix[i]) / 255
	}
	return s
}

func stripesPicture(w, h float64) *ui.Picture {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	for i := 0; i < int(w/8); i++ {
		col := uint32(0xFF3060C0)
		if i%2 == 1 {
			col = 0xFFF0A020
		}
		c.DrawRect(rectAt(float64(i)*8, 0, 8, h), paint(col))
	}
	return rec.EndRecording()
}

func TestBackdropFilterLayer(t *testing.T) {
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, stripesPicture(64, 32))
	b.PushClipRect(rectAt(16, 4, 32, 24))
	b.PushBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 4, SigmaY: 4}))
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rectAt(20, 8, 8, 8), paint(0xFF00FF00))
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := Render(b.Build(), 64, 32)
	if err != nil {
		t.Fatal(err)
	}
	sharp := px(img, 7, 2) // outside the clip: a stripe edge at x=8 stays crisp
	crisp := px(img, 8, 2)
	if sharp == crisp {
		t.Error("the stripes outside the clip are not crisp")
	}
	// Inside the clip the stripe edge at x=24 is blurred: its two sides are close.
	left, right := px(img, 23, 20), px(img, 24, 20)
	d := absInt(int(left[0])-int(right[0])) + absInt(int(left[2])-int(right[2]))
	if d > 80 {
		t.Errorf("stripe edge inside the backdrop filter still differs by %d: %v %v", d, left, right)
	}
	// The child is drawn over the filtered backdrop, unfiltered.
	if got := px(img, 24, 12); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("child = %v", got)
	}
	// Outside the clip, away from the child, the picture is untouched.
	if px(img, 4, 20) != px(img, 4, 2) {
		t.Errorf("outside the clip changed: %v vs %v", px(img, 4, 20), px(img, 4, 2))
	}
}

func TestBackdropMatrixFilterReadsOutsideTheClip(t *testing.T) {
	// A filter that moves pixels by 20 up or down: the lens at (8, 8) to (24, 16) shows pixels
	// that are not under it.
	tests := []struct {
		name  string
		shift float32
		// want is the color the lens shows at (16, 12), where the picture has a red square at
		// (12, 28) to (20, 36) and nothing else.
		want [4]uint8
	}{
		{"it shows what is 20 below the lens", -20, [4]uint8{255, 0, 0, 255}},
		{"it shows what is 20 above the lens", 20, [4]uint8{255, 255, 255, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := ui.NewSceneBuilder()
			rec := ui.NewPictureRecorder()
			c := ui.NewCanvas(rec)
			c.DrawRect(rectAt(0, 0, 32, 48), paint(0xFFFFFFFF))
			c.DrawRect(rectAt(12, 28, 8, 8), paint(0xFFFF0000))
			b.AddPicture(ui.Offset{}, rec.EndRecording())
			b.PushClipRect(rectAt(8, 8, 16, 8))
			b.PushBackdropFilter(ui.NewImageFilterMatrix(ui.NewMatrix4TranslationValues(0, tt.shift, 0)))
			img, err := Render(b.Build(), 32, 48)
			if err != nil {
				t.Fatal(err)
			}
			if got := px(img, 16, 12); got != tt.want {
				t.Errorf("the lens shows %v, want %v", got, tt.want)
			}
			// Outside the clip nothing changed.
			if got := px(img, 16, 30); got != [4]uint8{255, 0, 0, 255} {
				t.Errorf("outside the clip = %v", got)
			}
		})
	}
}

func TestBackdropFilterBlendAndId(t *testing.T) {
	// A backdrop with the dstOver blend: the original stays over the filtered image.
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, stripesPicture(32, 16))
	b.PushBackdropFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrc), ui.PushBackdropFilterOpts{BlendMode: ui.BlendModeSrcOver})
	img, err := Render(b.Build(), 32, 16)
	if err != nil {
		t.Fatal(err)
	}
	if got := px(img, 4, 4); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("a white color filter over the backdrop = %v", got)
	}
	id := 7
	b = ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, stripesPicture(32, 16))
	b.PushBackdropFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrc), ui.PushBackdropFilterOpts{BackdropID: &id})
	if _, err := Render(b.Build(), 32, 16); err != nil {
		t.Errorf("a backdrop id must not fail: %v", err)
	}
}

func TestShaderMaskLayer(t *testing.T) {
	// A white rect masked by a gradient that goes from opaque to transparent
	// across the mask rectangle: dstIn keeps the rect where the shader is opaque.
	b := ui.NewSceneBuilder()
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rectAt(0, 0, 64, 16), paint(0xFFFF0000))
	grad := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 32}, []ui.Color{ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF)})
	b.PushShaderMask(grad, rectAt(16, 0, 32, 16), ui.BlendModeDstIn)
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := Render(b.Build(), 64, 16)
	if err != nil {
		t.Fatal(err)
	}
	// Outside the mask rectangle the layer is untouched (dstIn only draws inside it).
	if alphaAt(img, 4, 8) != 255 || alphaAt(img, 60, 8) != 255 {
		t.Errorf("outside the mask rect: %d %d, want 255", alphaAt(img, 4, 8), alphaAt(img, 60, 8))
	}
	// The gradient's coordinates start at the mask rect's left: opaque at 16, clear at 48.
	a16, a32, a47 := alphaAt(img, 16, 8), alphaAt(img, 32, 8), alphaAt(img, 47, 8)
	if a16 < 240 || a32 < 100 || a32 > 150 || a47 > 15 {
		t.Errorf("alpha at 16, 32, 47 = %d, %d, %d", a16, a32, a47)
	}
}

func TestImageFilterLayer(t *testing.T) {
	b := ui.NewSceneBuilder()
	b.PushImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}), ui.PushImageFilterOpts{Offset: ui.Offset{Dx: 8, Dy: 4}})
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rectAt(0, 0, 8, 8), paint(0xFF000000))
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	img, err := Render(b.Build(), 32, 24)
	if err != nil {
		t.Fatal(err)
	}
	// The center of an 8x8 box blurred with sigma 2 keeps erf(4/(2*sqrt 2))^2 = 91%.
	if got := alphaAt(img, 12, 8); got < 225 || got > 240 {
		t.Errorf("center of the offset square = %d, want about 232", got)
	}
	if got := alphaAt(img, 7, 8); got == 0 || got > 120 {
		t.Errorf("just left of the offset square = %d, want a soft edge", got)
	}
	if got := alphaAt(img, 2, 2); got != 0 {
		t.Errorf("not at the offset = %d", got)
	}
}

func TestUnsupportedFiltersFail(t *testing.T) {
	// There are no other kinds of image filter or shader; the errors exist for
	// ones added to ui later.
	if _, err := applyImageFilter(image.NewRGBA(image.Rect(0, 0, 1, 1)), nil, affine{a: 1, d: 1}, ui.TileModeClamp); err == nil {
		t.Error("a nil filter should be an error")
	}
	if _, _, err := (&renderer{}).shaderEval(nil, affine{a: 1, d: 1}); err == nil {
		t.Error("a nil shader should be an error")
	}
}

func TestGoldenEffects(t *testing.T) {
	t.Run("mask-filter-styles", func(t *testing.T) {
		img := renderOf(t, 160, 40, func(c *ui.Canvas) {
			white(c)
			for i, style := range []ui.BlurStyle{ui.BlurStyleNormal, ui.BlurStyleSolid, ui.BlurStyleOuter, ui.BlurStyleInner} {
				p := paint(0xFF3060C0)
				p.SetMaskFilter(ui.NewMaskFilterBlur(style, 3))
				c.DrawRRect(ui.NewRRectFromRectXY(rectAt(float64(i)*40+8, 8, 26, 24), 6, 6), p)
			}
		})
		goldens.Check(t, "effects-mask-filter-styles", img)
	})
	t.Run("image-blur-tile-modes", func(t *testing.T) {
		img := renderOf(t, 160, 40, func(c *ui.Canvas) {
			white(c)
			for i, tile := range []ui.TileMode{ui.TileModeClamp, ui.TileModeRepeated, ui.TileModeMirror, ui.TileModeDecal} {
				c.Save()
				c.ClipRect(rectAt(float64(i)*40+2, 2, 36, 36))
				c.Translate(float64(i)*40+2, 2)
				p := ui.NewPaint()
				p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 3, SigmaY: 3, TileMode: tile}))
				b := rectAt(0, 0, 36, 36)
				c.SaveLayer(&b, p)
				c.DrawRect(rectAt(0, 8, 14, 20), paint(0xFFE53935))
				c.DrawCircle(ui.Offset{Dx: 26, Dy: 18}, 6, paint(0xFF1E88E5))
				c.Restore()
				c.Restore()
			}
		})
		goldens.Check(t, "effects-image-blur-tile-modes", img)
	})
	t.Run("shadows", func(t *testing.T) {
		img := renderOf(t, 240, 100, func(c *ui.Canvas) {
			c.DrawColor(ui.NewColor(0xFFF5F5F5), ui.BlendModeSrcOver)
			for i, e := range []float64{1, 4, 12} {
				card := ui.NewRRectFromRectXY(rectAt(float64(i)*80+16, 20, 48, 48), 8, 8)
				p := ui.NewPath()
				p.AddRRect(card)
				c.DrawShadow(p, ui.NewColor(0xFF000000), e, false)
				c.DrawRRect(card, paint(0xFFFFFFFF))
			}
		})
		goldens.Check(t, "effects-shadows", img)
	})
	t.Run("shadow-shapes", func(t *testing.T) {
		img := renderOf(t, 200, 80, func(c *ui.Canvas) {
			c.DrawColor(ui.NewColor(0xFFF5F5F5), ui.BlendModeSrcOver)
			circle := ui.NewPath()
			circle.AddOval(ui.NewRectFromCircle(ui.Offset{Dx: 40, Dy: 40}, 22))
			c.DrawShadow(circle, ui.NewColor(0xFF000000), 6, false)
			c.DrawPath(circle, paint(0xFFFFFFFF))
			star := ui.NewPath()
			for i := 0; i < 5; i++ {
				a := -math.Pi/2 + float64(i)*4*math.Pi/5
				x, y := 100+24*math.Cos(a), 40+24*math.Sin(a)
				if i == 0 {
					star.MoveTo(x, y)
				} else {
					star.LineTo(x, y)
				}
			}
			star.Close()
			c.DrawShadow(star, ui.NewColor(0xFF000000), 6, true) // transparent occluder
			glass := ui.NewPath()
			glass.AddRRect(ui.NewRRectFromRectXY(rectAt(136, 18, 48, 44), 10, 10))
			c.DrawShadow(glass, ui.NewColor(0xFF1E88E5), 8, false)
			c.DrawPath(glass, paint(0xFFFFFFFF))
		})
		goldens.Check(t, "effects-shadow-shapes", img)
	})
	t.Run("backdrop-blur", func(t *testing.T) {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, stripesPicture(96, 64))
		b.PushClipRRect(ui.NewRRectFromRectXY(rectAt(20, 12, 56, 40), 10, 10))
		b.PushBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 5, SigmaY: 5}))
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawRect(rectAt(0, 0, 96, 64), paint(0x40FFFFFF))
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		img, err := Render(b.Build(), 96, 64)
		if err != nil {
			t.Fatal(err)
		}
		goldens.Check(t, "effects-backdrop-blur", img)
	})
	t.Run("shader-mask", func(t *testing.T) {
		b := ui.NewSceneBuilder()
		bg := ui.NewPictureRecorder()
		ui.NewCanvas(bg).DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
		b.AddPicture(ui.Offset{}, bg.EndRecording())
		b.PushShaderMask(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 80}, []ui.Color{ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF)}),
			rectAt(8, 8, 80, 48), ui.BlendModeDstIn)
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		c.DrawRRect(ui.NewRRectFromRectXY(rectAt(8, 8, 80, 48), 8, 8), paint(0xFF6750A4))
		c.DrawCircle(ui.Offset{Dx: 30, Dy: 32}, 12, paint(0xFFF2B600))
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		img, err := Render(b.Build(), 96, 64)
		if err != nil {
			t.Fatal(err)
		}
		goldens.Check(t, "effects-shader-mask", img)
	})
	t.Run("filters", func(t *testing.T) {
		img := renderOf(t, 160, 40, func(c *ui.Canvas) {
			white(c)
			shift := ui.NewMatrix4Identity()
			shift.Translate(6, 6, 0)
			shift.Scale(1.0, 0.6, 1)
			filters := []ui.ImageFilter{
				ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2}),
				ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2}),
				ui.NewImageFilterMatrix(shift),
				ui.NewImageFilterCompose(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}), ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 2})),
			}
			for i, f := range filters {
				c.Save()
				c.ClipRect(rectAt(float64(i)*40+2, 2, 36, 36))
				c.Translate(float64(i)*40+2, 2)
				p := ui.NewPaint()
				p.SetImageFilter(f)
				c.SaveLayer(nil, p)
				c.DrawRRect(ui.NewRRectFromRectXY(rectAt(8, 8, 18, 18), 4, 4), paint(0xFF3060C0))
				c.DrawRect(rectAt(12, 12, 3, 3), paint(0xFFFFFFFF))
				c.Restore()
				c.Restore()
			}
		})
		goldens.Check(t, "effects-filters", img)
	})
	t.Run("proof-effects", func(t *testing.T) {
		goldens.Check(t, "proof-effects", renderProof(t, "effects"))
	})
}
