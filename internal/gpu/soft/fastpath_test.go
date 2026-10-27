package soft

import (
	"image"
	"math/rand"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// randomImage returns an image whose pixels are valid premultiplied colors.
func randomImage(rng *rand.Rand, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		a := byte(rng.Intn(256))
		if rng.Intn(3) == 0 {
			a = 255
		}
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = byte(rng.Intn(int(a)+1)), byte(rng.Intn(int(a)+1)), byte(rng.Intn(int(a)+1)), a
	}
	return img
}

// TestSolidRowMatchesMix checks that the row loop for a color that is the same everywhere
// writes the bytes that the blend of one pixel after another does.
func TestSolidRowMatchesMix(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	colors := []ui.Color{
		ui.NewColor(0xFF336699), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0x80FF8000),
		ui.NewColor(0x01010101), ui.NewColor(0xFE123456), ui.NewColor(0x00FFFFFF), ui.NewColor(0xB4B4B4B4),
	}
	tests := []struct {
		name      string
		useCov    bool
		useClip   bool
		fullCover bool
	}{
		{"no coverage", false, false, true},
		{"coverage", true, false, false},
		{"clip", false, true, false},
		{"coverage and clip", true, true, false},
	}
	const w = 211
	for _, tt := range tests {
		for _, c := range colors {
			t.Run(tt.name, func(t *testing.T) {
				want := randomImage(rng, w, 1)
				got := &image.RGBA{Pix: append([]byte(nil), want.Pix...), Stride: want.Stride, Rect: want.Rect}
				var cov, clipRow []float32
				if tt.useCov {
					cov = make([]float32, w)
					for i := range cov {
						cov[i] = []float32{0, 1, 1, 0.5, rng.Float32(), 0.999, 0.001}[rng.Intn(7)]
					}
				}
				if tt.useClip {
					clipRow = make([]float32, w)
					for i := range clipRow {
						clipRow[i] = []float32{0, 1, 1, 0.25, rng.Float32()}[rng.Intn(5)]
					}
				}
				s := solidPx(c)
				rw := &renderer{img: want}
				for i := 0; i < w; i++ {
					cv := float32(1)
					if cov != nil {
						cv = cov[i]
					}
					if cv == 0 {
						continue
					}
					if clipRow != nil {
						cv *= clipRow[i]
						if cv == 0 {
							continue
						}
					}
					if s[3] == 0 {
						continue
					}
					rw.mix(i, 0, s, ui.BlendModeSrcOver, cv, 0)
				}
				(&renderer{img: got}).solidRow(0, 0, w, cov, clipRow, s)
				for i := range want.Pix {
					if want.Pix[i] != got.Pix[i] {
						t.Fatalf("color %v: byte %d = %d, want %d", c, i, got.Pix[i], want.Pix[i])
					}
				}
			})
		}
	}
}

// reference draws the scene whole, as RenderInto draws a region of it.
func TestRenderIntoRegionMatchesWhole(t *testing.T) {
	scene := sceneOf(ui.Offset{}, func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 80), paint(0xFFE0E8F0))
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10.3, 12.6, 60.4, 40.2), 9, 9), paint(0xCC2060C0))
		c.DrawCircle(ui.Offset{Dx: 70, Dy: 40}, 25, stroked(0xFFC04020, 5.5, ui.StrokeCapRound, ui.StrokeJoinRound))
		c.Save()
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 50, 50), 12, 12))
		c.DrawColor(ui.NewColor(0x6600AA00), ui.BlendModeSrcOver)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 30), paint(0x88FFFFFF))
		c.Restore()
	})
	whole, err := Render(scene, 100, 80)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		region image.Rectangle
	}{
		{"all", image.Rect(0, 0, 100, 80)},
		{"inside", image.Rect(13, 17, 61, 55)},
		{"a row", image.Rect(0, 40, 100, 41)},
		{"a column", image.Rect(70, 0, 71, 80)},
		{"a corner", image.Rect(60, 50, 120, 120)},
		{"off the surface", image.Rect(200, 200, 300, 300)},
		{"empty", image.Rect(5, 5, 5, 9)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start from other pixels, clear the region, and draw into it: the region
			// then holds the pixels of the whole drawing and the rest is untouched.
			dst := randomImage(rand.New(rand.NewSource(2)), 100, 80)
			before := append([]byte(nil), dst.Pix...)
			r := tt.region.Intersect(dst.Rect)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				i := dst.PixOffset(r.Min.X, y)
				clear(dst.Pix[i : i+4*r.Dx()])
			}
			cleared := append([]byte(nil), dst.Pix...)
			if err := RenderInto(dst, scene, tt.region); err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 80; y++ {
				for x := 0; x < 100; x++ {
					i := dst.PixOffset(x, y)
					want := cleared[i : i+4]
					if (image.Point{X: x, Y: y}).In(r) {
						want = whole.Pix[i : i+4]
					} else if string(cleared[i:i+4]) != string(before[i:i+4]) {
						t.Fatalf("test bug at %d,%d", x, y)
					}
					if string(dst.Pix[i:i+4]) != string(want) {
						t.Fatalf("pixel %d,%d = %v, want %v", x, y, dst.Pix[i:i+4], want)
					}
				}
			}
		})
	}
}

func TestRenderIntoErrors(t *testing.T) {
	scene := sceneOf(ui.Offset{}, func(c *ui.Canvas) {})
	tests := []struct {
		name  string
		dst   *image.RGBA
		scene *ui.Scene
		route string
	}{
		{"nil scene", image.NewRGBA(image.Rect(0, 0, 4, 4)), nil, "RenderInto.SoftSceneNil"},
		{"nil image", nil, scene, "RenderInto.SoftTargetInvalid"},
		{"empty image", &image.RGBA{}, scene, "RenderInto.SoftTargetInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RenderInto(tt.dst, tt.scene, image.Rect(0, 0, 4, 4))
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.route) {
				t.Fatalf("expected %s, got %v", tt.route, err)
			}
		})
	}
}

// TestRasterizeScratch checks that the memory shapes are rasterized in is reused without
// changing the masks, and is all zeros again after each.
func TestRasterizeScratch(t *testing.T) {
	shapes := [][][]ui.Offset{
		{poly(0.25, 0.5, 40.75, 0.5, 40.75, 30.25, 0.25, 30.25)},
		{poly(5, 5, 90, 20, 30, 80)},
		{poly(0, 0, 10, 0, 10, 10, 0, 10), poly(2, 2, 8, 2, 8, 8, 2, 8)},
		{poly(-20, -20, 200, -20, 200, 200, -20, 200)},
		{poly(3, 3, 5, 3, 5, 5)},
	}
	clip := image.Rect(0, 0, 100, 90)
	sc := &scratch{}
	for round := 0; round < 2; round++ {
		for i, contours := range shapes {
			for _, rule := range []FillRule{FillNonZero, FillEvenOdd} {
				want := Rasterize(contours, rule, clip)
				got := rasterize(contours, rule, clip, sc)
				if (want == nil) != (got == nil) {
					t.Fatalf("shape %d: nil mismatch", i)
				}
				if want != nil {
					if want.Rect != got.Rect || len(want.Cov) != len(got.Cov) {
						t.Fatalf("shape %d: rect %v vs %v", i, want.Rect, got.Rect)
					}
					for j := range want.Cov {
						if want.Cov[j] != got.Cov[j] {
							t.Fatalf("shape %d: coverage %d = %v, want %v", i, j, got.Cov[j], want.Cov[j])
						}
					}
				}
				for j, v := range sc.buf {
					if v != 0 {
						t.Fatalf("shape %d: scratch cell %d is %v after use", i, j, v)
					}
				}
			}
		}
	}
}

// TestMaskRowExtents checks that the extents of a rasterized mask hold all its coverage and
// start and end on it.
func TestMaskRowExtents(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for n := 0; n < 200; n++ {
		var contour []ui.Offset
		for i := 3 + rng.Intn(5); i > 0; i-- {
			contour = append(contour, ui.Offset{Dx: rng.Float64()*120 - 10, Dy: rng.Float64()*90 - 10})
		}
		m := Rasterize([][]ui.Offset{contour}, FillRule(rng.Intn(2)), image.Rect(0, 0, 100, 80))
		if m == nil {
			continue
		}
		w := m.Rect.Dx()
		for y := 0; y < m.Rect.Dy(); y++ {
			e := m.rows[y]
			first, last := -1, -1
			for x := 0; x < w; x++ {
				if m.Cov[y*w+x] != 0 {
					if first < 0 {
						first = x
					}
					last = x + 1
				}
			}
			if first < 0 {
				if e.hi > e.lo {
					t.Fatalf("round %d row %d: extent %v for an empty row", n, y, e)
				}
				continue
			}
			if int(e.lo) != first || int(e.hi) != last {
				t.Fatalf("round %d row %d: extent %v, coverage from %d to %d", n, y, e, first, last)
			}
		}
	}
}
