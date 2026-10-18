package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

// imageFunc makes an image whose pixel x, y has the color f(x, y), premultiplied.
func imageFunc(w, h int, f func(x, y int) [4]uint8) *ui.Image {
	pix := make([]byte, 4*w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := f(x, y)
			copy(pix[4*(y*w+x):], c[:])
		}
	}
	return ui.NewImageFromRGBA(pix, w, h)
}

// quads is a 4x4 image of four 2x2 blocks: red, green, blue and white.
func quads() *ui.Image {
	return imageFunc(4, 4, func(x, y int) [4]uint8 {
		switch {
		case x < 2 && y < 2:
			return [4]uint8{255, 0, 0, 255}
		case y < 2:
			return [4]uint8{0, 255, 0, 255}
		case x < 2:
			return [4]uint8{0, 0, 255, 255}
		}
		return [4]uint8{255, 255, 255, 255}
	})
}

// ramp is a 16x16 image with a horizontal gradient from black to white.
func ramp() *ui.Image {
	return imageFunc(16, 16, func(x, y int) [4]uint8 {
		v := uint8(x * 255 / 15)
		return [4]uint8{v, v, v, 255}
	})
}

// detail is a 32x32 image of one-pixel black and white checks.
func detail() *ui.Image {
	return imageFunc(32, 32, func(x, y int) [4]uint8 {
		if (x+y)%2 == 0 {
			return [4]uint8{0, 0, 0, 255}
		}
		return [4]uint8{255, 255, 255, 255}
	})
}

func withQuality(q ui.FilterQuality) *ui.Paint {
	p := ui.NewPaint()
	p.SetFilterQuality(q)
	return p
}

func TestDrawImageNearest(t *testing.T) {
	img := renderOf(t, 12, 12, func(c *ui.Canvas) {
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(2, 2, 8, 8), withQuality(ui.FilterQualityNone))
	})
	tests := []struct {
		x, y int
		want [4]uint8
	}{
		{2, 2, [4]uint8{255, 0, 0, 255}},
		{5, 5, [4]uint8{255, 0, 0, 255}},
		{6, 2, [4]uint8{0, 255, 0, 255}},
		{2, 6, [4]uint8{0, 0, 255, 255}},
		{9, 9, [4]uint8{255, 255, 255, 255}},
		{1, 1, [4]uint8{}},
		{10, 5, [4]uint8{}},
	}
	for _, tt := range tests {
		if got := px(img, tt.x, tt.y); got != tt.want {
			t.Errorf("pixel %d,%d = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestDrawImageUnscaled(t *testing.T) {
	// DrawImage at an offset copies the pixels exactly, whatever the quality.
	for _, q := range []ui.FilterQuality{ui.FilterQualityNone, ui.FilterQualityLow, ui.FilterQualityMedium} {
		img := renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawImage(quads(), ui.Offset{Dx: 2, Dy: 3}, withQuality(q)) })
		if got := px(img, 2, 3); got != [4]uint8{255, 0, 0, 255} {
			t.Errorf("quality %v: top left = %v", q, got)
		}
		if got := px(img, 5, 6); got != [4]uint8{255, 255, 255, 255} {
			t.Errorf("quality %v: bottom right = %v", q, got)
		}
		if got := px(img, 4, 3); got != [4]uint8{0, 255, 0, 255} {
			t.Errorf("quality %v: top right block = %v", q, got)
		}
	}
}

func TestDrawImageFiltering(t *testing.T) {
	// A 16-wide ramp drawn 4 times larger: nearest is flat in each 4 px, bilinear
	// is smooth and strictly increasing.
	column := func(q ui.FilterQuality) []uint8 {
		img := renderOf(t, 64, 4, func(c *ui.Canvas) {
			c.DrawImageRect(ramp(), ui.NewRectFromLTWH(0, 0, 16, 16), ui.NewRectFromLTWH(0, 0, 64, 4), withQuality(q))
		})
		out := make([]uint8, 64)
		for x := range out {
			out[x] = px(img, x, 1)[0]
		}
		return out
	}
	near := column(ui.FilterQualityNone)
	for x := 0; x < 64; x++ {
		if near[x] != near[x/4*4] {
			t.Fatalf("nearest is not flat inside a texel at %d: %v", x, near[x-1:x+1])
		}
	}
	for _, q := range []ui.FilterQuality{ui.FilterQualityLow, ui.FilterQualityMedium, ui.FilterQualityHigh} {
		lin := column(q)
		for x := 8; x < 56; x++ {
			if lin[x] <= lin[x-1] && lin[x] != 255 {
				t.Errorf("quality %v: not increasing at %d: %d then %d", q, x, lin[x-1], lin[x])
				break
			}
		}
		// The pixel center at 32.5 maps to image x 8.125, between the centers of
		// texels 7 and 8, whose values are 7 and 8 steps of 255/15.
		if want := (8.125 - 0.5) * 255 / 15; math.Abs(float64(lin[32])-want) > 6 {
			t.Errorf("quality %v: value at 32 = %d", q, lin[32])
		}
	}
}

func TestDrawImageMipmaps(t *testing.T) {
	// The check pattern drawn at 1/8 size averages to gray with mipmaps, but
	// aliases with nearest and bilinear sampling.
	value := func(q ui.FilterQuality) uint8 {
		img := renderOf(t, 4, 4, func(c *ui.Canvas) {
			c.DrawImageRect(detail(), ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(0, 0, 4, 4), withQuality(q))
		})
		return px(img, 1, 1)[0]
	}
	for _, q := range []ui.FilterQuality{ui.FilterQualityMedium, ui.FilterQualityHigh} {
		if got := value(q); got < 120 || got > 136 {
			t.Errorf("quality %v: downscaled checks = %d, want about 128", q, got)
		}
	}
	if got := value(ui.FilterQualityNone); got != 0 && got != 255 {
		t.Errorf("nearest of a check = %d, want a single texel", got)
	}
}

func TestMipLevels(t *testing.T) {
	r := &renderer{mips: map[*ui.Image][]*mipLevel{}}
	img := imageFunc(5, 3, func(x, y int) [4]uint8 { return [4]uint8{100, 100, 100, 100} })
	chain := r.levels(img)
	var sizes [][2]int
	for _, l := range chain {
		sizes = append(sizes, [2]int{l.w, l.h})
	}
	want := [][2]int{{5, 3}, {2, 1}, {1, 1}}
	if len(sizes) != len(want) {
		t.Fatalf("sizes = %v, want %v", sizes, want)
	}
	for i := range want {
		if sizes[i] != want[i] {
			t.Errorf("level %d = %v, want %v", i, sizes[i], want[i])
		}
	}
	for _, l := range chain {
		if got := l.pix[0]; got != 100 {
			t.Errorf("a flat image changed to %d at %dx%d", got, l.w, l.h)
		}
	}
	if again := r.levels(img); &again[0] != &chain[0] {
		t.Error("the chain is not cached")
	}
}

func TestDrawImageRectSource(t *testing.T) {
	// The source rectangle picks the green block and a part outside the image
	// is cut away along with its part of the destination.
	img := renderOf(t, 12, 8, func(c *ui.Canvas) {
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(2, 0, 2, 2), ui.NewRectFromLTWH(0, 0, 8, 8), withQuality(ui.FilterQualityNone))
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(2, 2, 4, 2), ui.NewRectFromLTWH(8, 0, 4, 2), withQuality(ui.FilterQualityNone))
	})
	if got := px(img, 3, 3); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("green block = %v", got)
	}
	// src 2..6 across x is half outside: only the left half (white, 2..4) of the
	// source, drawn in the left half of its destination, 8..10.
	if got := px(img, 8, 0); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("part inside = %v", got)
	}
	if got := px(img, 10, 0); got != [4]uint8{} {
		t.Errorf("part outside the image = %v, want transparent", got)
	}
}

func TestDrawImageTransformed(t *testing.T) {
	// Rotated by a quarter turn: the red block moves from the top left to the top right.
	img := renderOf(t, 8, 8, func(c *ui.Canvas) {
		c.Translate(8, 0)
		c.Rotate(math.Pi / 2)
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(0, 0, 8, 8), withQuality(ui.FilterQualityNone))
	})
	if got := px(img, 7, 0); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("rotated red block = %v", got)
	}
	if got := px(img, 0, 0); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("rotated blue block = %v", got)
	}
}

func TestDrawImagePaintAlphaAndAntiAlias(t *testing.T) {
	p := withQuality(ui.FilterQualityNone)
	p.SetColor(ui.NewColor(0x80FFFFFF)) // half opaque
	img := renderOf(t, 4, 4, func(c *ui.Canvas) {
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 2, 2), ui.NewRectFromLTWH(0, 0, 4, 4), p)
	})
	if got := px(img, 1, 1); got != [4]uint8{128, 0, 0, 128} {
		t.Errorf("half opaque red = %v", got)
	}
	// A fractional edge is anti-aliased, or not.
	edge := func(aa bool) uint8 {
		pp := withQuality(ui.FilterQualityNone)
		pp.SetIsAntiAlias(aa)
		img := renderOf(t, 8, 4, func(c *ui.Canvas) {
			c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(0, 0, 4.25, 4), pp)
		})
		return px(img, 4, 1)[3]
	}
	if got := edge(true); got < 55 || got > 75 {
		t.Errorf("anti-aliased edge alpha = %d, want about 64", got)
	}
	if got := edge(false); got != 0 {
		t.Errorf("hard edge alpha = %d, want 0", got)
	}
}

func TestDrawImageNine(t *testing.T) {
	// A 6x6 image: a red 2 px border, a green 2x2 center.
	nine := imageFunc(6, 6, func(x, y int) [4]uint8 {
		if x >= 2 && x < 4 && y >= 2 && y < 4 {
			return [4]uint8{0, 255, 0, 255}
		}
		return [4]uint8{255, 0, 0, 255}
	})
	center := ui.NewRectFromLTWH(2, 2, 2, 2)
	tests := []struct {
		name string
		dst  ui.Rect
		at   map[image.Point][4]uint8
	}{
		{"stretched", ui.NewRectFromLTWH(1, 1, 14, 10), map[image.Point][4]uint8{
			{1, 1}: {255, 0, 0, 255}, {2, 2}: {255, 0, 0, 255}, {3, 3}: {0, 255, 0, 255}, {12, 8}: {0, 255, 0, 255},
			{13, 9}: {255, 0, 0, 255}, {14, 10}: {255, 0, 0, 255}, {0, 0}: {}, {15, 5}: {},
		}},
		{"smaller than the corners", ui.NewRectFromLTWH(0, 0, 2, 2), map[image.Point][4]uint8{
			{0, 0}: {255, 0, 0, 255}, {1, 1}: {255, 0, 0, 255}, {2, 2}: {},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := withQuality(ui.FilterQualityNone)
			img := renderOf(t, 16, 12, func(c *ui.Canvas) { c.DrawImageNine(nine, center, tt.dst, p) })
			for pt, want := range tt.at {
				if got := px(img, pt.X, pt.Y); got != want {
					t.Errorf("pixel %v = %v, want %v", pt, got, want)
				}
			}
		})
	}
}

func TestNinePieces(t *testing.T) {
	tests := []struct {
		name         string
		origin, size float64
		src          [4]float64
		want         [4]float64
	}{
		{"stretch", 10, 20, [4]float64{0, 2, 4, 6}, [4]float64{10, 12, 28, 30}},
		{"exact", 0, 4, [4]float64{0, 2, 4, 6}, [4]float64{0, 2, 2, 4}},
		{"squeezed", 0, 2, [4]float64{0, 2, 4, 6}, [4]float64{0, 1, 1, 2}},
		{"no fixed part", 5, 10, [4]float64{0, 0, 6, 6}, [4]float64{5, 5, 15, 15}},
	}
	for _, tt := range tests {
		if got := ninePieces(tt.origin, tt.size, tt.src); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDrawAtlas(t *testing.T) {
	// Two sprites from the atlas: the green block unrotated at 1,1, and the blue
	// block rotated a quarter turn, scaled by 2, with its origin at 10, 2.
	tr := []ui.RSTransform{ui.NewRSTransform(1, 0, 1, 1), ui.NewRSTransform(0, 2, 10, 2)}
	rects := []ui.Rect{ui.NewRectFromLTWH(2, 0, 2, 2), ui.NewRectFromLTWH(0, 2, 2, 2)}
	img := renderOf(t, 16, 8, func(c *ui.Canvas) {
		c.DrawAtlas(quads(), tr, rects, nil, ui.BlendModeSrcOver, nil, withQuality(ui.FilterQualityNone))
	})
	if got := px(img, 1, 1); got != [4]uint8{0, 255, 0, 255} {
		t.Errorf("first sprite = %v", got)
	}
	if got := px(img, 3, 1); got != [4]uint8{} {
		t.Errorf("outside the first sprite = %v", got)
	}
	// Rotated by 90 degrees (scos 0, ssin 2): the 2x2 sprite becomes 4x4, with
	// x going down and y going left from the origin: covering x 6..10, y 2..6.
	if got := px(img, 8, 3); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("second sprite = %v", got)
	}
	if got := px(img, 11, 3); got != [4]uint8{} {
		t.Errorf("right of the second sprite = %v", got)
	}
}

func TestImageErrors(t *testing.T) {
	disposed := quads()
	disposed.Dispose()
	tests := []struct {
		name string
		draw func(c *ui.Canvas)
		tag  string
	}{
		{"disposed image", func(c *ui.Canvas) { c.DrawImage(disposed, ui.Offset{}, nil) }, "SoftImageDisposed"},
		{"disposed in rect", func(c *ui.Canvas) {
			c.DrawImageRect(disposed, ui.NewRectFromLTWH(0, 0, 1, 1), ui.NewRectFromLTWH(0, 0, 1, 1), nil)
		}, "SoftImageDisposed"},
		{"disposed nine", func(c *ui.Canvas) {
			c.DrawImageNine(disposed, ui.NewRectFromLTWH(1, 1, 1, 1), ui.NewRectFromLTWH(0, 0, 4, 4), nil)
		}, "SoftImageDisposed"},
		{"disposed atlas", func(c *ui.Canvas) {
			c.DrawAtlas(disposed, []ui.RSTransform{ui.NewRSTransform(1, 0, 0, 0)}, []ui.Rect{ui.NewRectFromLTWH(0, 0, 1, 1)}, nil, ui.BlendModeSrcOver, nil, nil)
		}, "SoftImageDisposed"},
		{"disposed image shader", func(c *ui.Canvas) {
			p := ui.NewPaint()
			p.SetShader(ui.NewImageShader(disposed, ui.TileModeClamp, ui.TileModeClamp, nil))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 4, 4), p)
		}, "SoftImageDisposed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Render(sceneOf(ui.Offset{}, tt.draw), 8, 8)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) {
				t.Errorf("expected %s, got %v", tt.tag, err)
			}
		})
	}
}

func TestDrawImageDegenerate(t *testing.T) {
	for name, draw := range map[string]func(c *ui.Canvas){
		"empty src": func(c *ui.Canvas) {
			c.DrawImageRect(quads(), ui.NewRectFromLTWH(1, 1, 0, 2), ui.NewRectFromLTWH(0, 0, 4, 4), nil)
		},
		"empty dst": func(c *ui.Canvas) {
			c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 2, 2), ui.NewRectFromLTWH(0, 0, 0, 4), nil)
		},
		"src outside the image": func(c *ui.Canvas) {
			c.DrawImageRect(quads(), ui.NewRectFromLTWH(10, 10, 2, 2), ui.NewRectFromLTWH(0, 0, 4, 4), nil)
		},
		"singular transform": func(c *ui.Canvas) {
			c.Scale(0, 1)
			c.DrawImage(quads(), ui.Offset{}, nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := coverageSum(renderOf(t, 8, 8, draw)); got != 0 {
				t.Errorf("drew %v pixels", got)
			}
		})
	}
}

func TestGoldenImages(t *testing.T) {
	smooth := func(c *ui.Canvas) {
		white(c)
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(2, 2, 28, 28), withQuality(ui.FilterQualityLow))
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(34, 2, 28, 28), withQuality(ui.FilterQualityNone))
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(2, 34, 28, 28), withQuality(ui.FilterQualityHigh))
		c.DrawImageRect(detail(), ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(34, 34, 12, 12), withQuality(ui.FilterQualityMedium))
		c.DrawImageRect(detail(), ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(48, 34, 12, 12), withQuality(ui.FilterQualityNone))
	}
	rotated := func(c *ui.Canvas) {
		white(c)
		c.Translate(32, 32)
		c.Rotate(math.Pi / 6)
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(-20, -20, 40, 40), withQuality(ui.FilterQualityLow))
	}
	nine := func(c *ui.Canvas) {
		white(c)
		src := imageFunc(12, 12, func(x, y int) [4]uint8 {
			if x < 4 || x >= 8 || y < 4 || y >= 8 {
				return [4]uint8{0x30, 0x60, 0xC0, 255}
			}
			return [4]uint8{0xF0, 0xA0, 0x20, 255}
		})
		c.DrawImageNine(src, ui.NewRectFromLTWH(4, 4, 4, 4), ui.NewRectFromLTWH(4, 8, 56, 48), withQuality(ui.FilterQualityNone))
	}
	atlas := func(c *ui.Canvas) {
		white(c)
		var tr []ui.RSTransform
		var rects []ui.Rect
		for i := 0; i < 8; i++ {
			a := float64(i) * math.Pi / 4
			tr = append(tr, ui.NewRSTransform(float32(2*math.Cos(a)), float32(2*math.Sin(a)), float32(32+20*math.Cos(a)), float32(32+20*math.Sin(a))))
			rects = append(rects, ui.NewRectFromLTWH(float64(i%2)*2, float64(i/2%2)*2, 2, 2))
		}
		c.DrawAtlas(quads(), tr, rects, nil, ui.BlendModeSrcOver, nil, withQuality(ui.FilterQualityLow))
	}
	clipped := func(c *ui.Canvas) {
		white(c)
		c.Translate(32, 32)
		c.Rotate(math.Pi / 8)
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromCenter(ui.Offset{}, 44, 36), 14, 14))
		c.Rotate(-math.Pi / 8)
		c.DrawImageRect(quads(), ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(-30, -30, 60, 60), withQuality(ui.FilterQualityLow))
	}
	for name, draw := range map[string]func(*ui.Canvas){
		"filter-qualities": smooth, "rotated": rotated, "nine": nine, "atlas": atlas, "clipped-by-rotated-rrect": clipped,
	} {
		t.Run(name, func(t *testing.T) {
			goldens.Check(t, "image-"+name, renderOf(t, 64, 64, draw))
		})
	}
}
