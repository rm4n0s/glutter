package gles

import (
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// newTestResources makes the caches on a device with the fake GL.
func newTestResources(t *testing.T) (*resources, *Device, *fakeGL) {
	t.Helper()
	d, gl := newTestDevice(t)
	res, err := newResources(d, gl)
	if err != nil {
		t.Fatal(err)
	}
	return res, d, gl
}

// frameWith builds a frame in a new frame of the caches.
func frameWith(t *testing.T, res *resources, fn func(c *ui.Canvas)) *frameBuilder {
	t.Helper()
	res.beginFrame()
	defer res.endFrame()
	b, err := buildWith(t, res, false, fn)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func wave() *ui.Path {
	p := ui.NewPath()
	p.MoveTo(10, 30)
	p.CubicTo(40, -10, 70, 70, 100, 20)
	p.LineTo(100, 50)
	p.Close()
	return p
}

func TestPathsAreDrawnFromTheAtlasAndCached(t *testing.T) {
	res, _, gl := newTestResources(t)
	paint := solidPaint(0xFF204080)
	draw := func(dx float64) func(c *ui.Canvas) {
		return func(c *ui.Canvas) {
			c.Translate(dx, 5)
			c.DrawPath(wave(), paint)
		}
	}
	b := frameWith(t, res, draw(0))
	if b.frame.items(pipeMask) != 1 || b.frame.items(pipeShape) != 0 || res.tilesRasterized < 1 {
		t.Fatalf("mask items %d, rasterized %d", b.frame.items(pipeMask), res.tilesRasterized)
	}
	first := res.tilesRasterized
	in := b.frame.data[pipeMask]
	// The tile's pixels are a whole-number rectangle, its coverage is in the atlas page,
	// and the instance holds the premultiplied color and the clip.
	if in[0] != float32(int(in[0])) || in[2] <= in[0] || in[3] <= in[1] {
		t.Errorf("quad = %v", in[0:4])
	}
	if in[8] != 0x20/255.0*1 && in[11] != 1 {
		t.Errorf("color = %v", in[8:12])
	}
	if in[12] != 0 || in[14] != 200 {
		t.Errorf("clip = %v", in[12:16])
	}
	if gl.calls["texSubImage2D"] == 0 {
		t.Error("the tile was not uploaded")
	}
	uploads := gl.calls["texSubImage2D"]

	// The same path at another whole-pixel offset, in a later frame: no new tile.
	b = frameWith(t, res, draw(37))
	if res.tilesRasterized != first || b.frame.items(pipeMask) != 1 {
		t.Errorf("after a whole-pixel move: rasterized %d, want %d, items %d", res.tilesRasterized, first, b.frame.items(pipeMask))
	}
	if gl.calls["texSubImage2D"] != uploads {
		t.Error("a cached tile was uploaded again")
	}
	// The tile moved with the path.
	if got := b.frame.data[pipeMask][0] - in[0]; got != 37 {
		t.Errorf("the tile moved by %v, want 37", got)
	}
	// Another fractional position is another coverage.
	frameWith(t, res, draw(0.5))
	second := res.tilesRasterized
	if second <= first {
		t.Errorf("after a half-pixel move: rasterized %d, want more than %d", second, first)
	}
	// A path with the same contents built again, with the same paint: cached.
	frameWith(t, res, draw(0))
	if res.tilesRasterized != second {
		t.Errorf("a path built again was rasterized again: %d", res.tilesRasterized)
	}
	// Another stroke or color changes what is cached only when it changes the coverage.
	frameWith(t, res, func(c *ui.Canvas) { c.Translate(0, 5); c.DrawPath(wave(), solidPaint(0xFFFF0000)) })
	if res.tilesRasterized != second {
		t.Errorf("another color was rasterized again: %d", res.tilesRasterized)
	}
	frameWith(t, res, func(c *ui.Canvas) {
		c.Translate(0, 5)
		c.DrawPath(wave(), linePaint(0xFF204080, 3, ui.StrokeCapButt, ui.StrokeJoinMiter))
	})
	if res.tilesRasterized <= second {
		t.Errorf("a stroke should be rasterized: %d", res.tilesRasterized)
	}
}

func TestLargePathsAreTiledAndOnlyVisibleTilesAreMade(t *testing.T) {
	res, _, _ := newTestResources(t)
	big := ui.NewPath()
	big.AddRect(ui.NewRectFromLTWH(-100, -100, 2000, 2000))
	p := solidPaint(0xFF000000)
	// The visible surface is 200 x 100: one tile of 256 covers it.
	b := frameWith(t, res, func(c *ui.Canvas) { c.DrawPath(big, p) })
	if b.frame.items(pipeMask) != 1 || res.tilesRasterized != 1 {
		t.Errorf("tiles %d, rasterized %d", b.frame.items(pipeMask), res.tilesRasterized)
	}
	// Scrolled so that the visible part crosses tile boundaries.
	b = frameWith(t, res, func(c *ui.Canvas) { c.Translate(-200, -50); c.DrawPath(big, p) })
	if n := b.frame.items(pipeMask); n != 2 {
		t.Errorf("the surface spans %d tiles, want 2", n)
	}
	// A tile's quad never reaches beyond the tile size, and the tiles do not overlap.
	data := b.frame.data[pipeMask]
	for i := 0; i < b.frame.items(pipeMask); i++ {
		q := data[i*16 : i*16+4]
		if q[2]-q[0] > float32(gpu.MaxAtlasTile) || q[3]-q[1] > float32(gpu.MaxAtlasTile) {
			t.Errorf("tile %d is %v wide", i, q)
		}
	}
	if a, c := data[0:4], data[16:20]; !(a[2] <= c[0] || c[2] <= a[0] || a[3] <= c[1] || c[3] <= a[1]) {
		t.Errorf("tiles overlap: %v %v", a, c)
	}
	// An empty tile is remembered: nothing covered, nothing drawn, nothing rasterized twice.
	off := ui.NewPath()
	off.AddRect(ui.NewRectFromLTWH(1000, 1000, 10, 10))
	before := res.tilesRasterized
	b = frameWith(t, res, func(c *ui.Canvas) { c.DrawPath(off, p) })
	if b.frame.items(pipeMask) != 0 || res.tilesRasterized != before {
		t.Errorf("off the surface: %d tiles, rasterized %d more", b.frame.items(pipeMask), res.tilesRasterized-before)
	}
}

func TestAtlasPagesAreTexturesOfTheGPU(t *testing.T) {
	res, d, gl := newTestResources(t)
	frameWith(t, res, func(c *ui.Canvas) { c.DrawPath(wave(), solidPaint(0xFF000000)) })
	if len(res.pages) != 1 || res.pages[0].Width() != atlasPageSize || res.pages[0].Format() != TextureFormatR8 {
		t.Fatalf("pages = %v", res.pages)
	}
	res.release()
	if gl.liveCount("texture") != int(d.Stats().FreeTextures) && d.Stats().Textures != d.Stats().FreeTextures {
		t.Errorf("textures still in use after release: %+v", d.Stats())
	}
	if len(res.pages) != 0 || res.white != nil {
		t.Error("release kept what it should have given back")
	}
}

func TestAtlasFullIsAnError(t *testing.T) {
	res, _, _ := newTestResources(t)
	res.atlas = gpu.NewAtlas[maskKey](256, 1) // four slots of 128 or one of 256 in one page
	many := func(c *ui.Canvas) {
		// Each of these needs the whole 256 pixel page.
		for i := 0; i < 2; i++ {
			p := ui.NewPath()
			p.AddRect(ui.NewRectFromLTWH(-20, -20, 400+float64(i), 400))
			c.DrawPath(p, solidPaint(0xFF000000))
		}
	}
	res.beginFrame()
	_, err := buildWith(t, res, false, many)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "AtlasFull" {
		t.Fatalf("error = %v, want AtlasFull", err)
	}
	// With skipping it is an unsupported feature.
	res.beginFrame()
	b, err := buildWith(t, res, true, many)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range b.stats.Unsupported {
		found = found || u == "coverage atlas full"
	}
	if !found {
		t.Errorf("unsupported = %v", b.stats.Unsupported)
	}
}

func TestClipMasks(t *testing.T) {
	res, _, gl := newTestResources(t)
	clipped := func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20.5, 10.5, 100, 60), 20, 20))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 200, 100), solidPaint(0xFF000000))
		c.DrawCircle(ui.Offset{Dx: 60, Dy: 40}, 20, solidPaint(0xFFFF0000))
	}
	b := frameWith(t, res, clipped)
	if res.clipMasksBuilt != 1 || len(res.clipMasks) != 1 {
		t.Fatalf("masks built %d, cached %d", res.clipMasksBuilt, len(res.clipMasks))
	}
	var mask *clipMask
	for _, m := range res.clipMasks {
		mask = m
	}
	// The mask covers the pixels the shape reaches, once for both draws, and the
	// draws that share it are one call.
	if mask.x != 20 || mask.y != 10 || mask.w != 101 || mask.h != 61 {
		t.Errorf("mask rectangle = %d,%d %dx%d", mask.x, mask.y, mask.w, mask.h)
	}
	if b.frame.callsOf(pipeShape) != 1 || b.frame.calls[0].clip != mask {
		t.Errorf("calls %d, clip %v", b.frame.callsOf(pipeShape), b.frame.calls[0].clip)
	}
	if mask.tex.Format() != TextureFormatR8 {
		t.Errorf("mask format = %v", mask.tex.Format())
	}
	uploads := gl.calls["texSubImage2D"]
	// Later frames find the mask.
	frameWith(t, res, clipped)
	if res.clipMasksBuilt != 1 || gl.calls["texSubImage2D"] != uploads {
		t.Errorf("the mask was made again: built %d", res.clipMasksBuilt)
	}
	// A different shape is a different mask; a clip that leaves nothing draws nothing.
	frameWith(t, res, func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 10, 100, 60), 20, 20))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 200, 100), solidPaint(0xFF000000))
	})
	if res.clipMasksBuilt != 2 {
		t.Errorf("masks built = %d", res.clipMasksBuilt)
	}
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		c.Translate(5, 5)
		c.Rotate(0.3)
		c.ClipRect(ui.NewRectFromLTWH(100, 50, 10, 10)) // rotated away from the first
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 200, 100), solidPaint(0xFF000000))
	})
	if b.frame.items(pipeShape) != 0 {
		t.Errorf("a clip that leaves nothing drew %d shapes", b.frame.items(pipeShape))
	}
}

func TestClipMaskDifferenceAndCombination(t *testing.T) {
	res, _, _ := newTestResources(t)
	area := image.Rect(0, 0, 200, 100)
	hole := gpu.Clip{Bounds: ui.NewRectFromLTWH(0, 0, 200, 100), Shapes: []gpu.ClipShape{{
		Kind: gpu.ClipShapeRect, Rect: ui.NewRectFromLTWH(50, 20, 40, 40), Difference: true, AntiAlias: true, Transform: *ui.NewMatrix4Identity(),
	}}}
	res.beginFrame()
	m, empty, err := res.clipMask(hole, 200, 100)
	if err != nil || empty || m == nil {
		t.Fatalf("clipMask = %v, %v, %v", m, empty, err)
	}
	if m.w != area.Dx() || m.h != area.Dy() {
		t.Errorf("a difference keeps the whole clip: %dx%d", m.w, m.h)
	}
	// A hole as big as the clip leaves nothing.
	all := hole
	all.Shapes = []gpu.ClipShape{{Kind: gpu.ClipShapeRect, Rect: ui.NewRectFromLTWH(-10, -10, 300, 200), Difference: true, AntiAlias: true, Transform: *ui.NewMatrix4Identity()}}
	if m, empty, _ := res.clipMask(all, 200, 100); !empty || m != nil {
		t.Errorf("a hole over everything: %v, %v", m, empty)
	}
	// A shape outside the clip's bounds leaves nothing.
	out := gpu.Clip{Bounds: ui.NewRectFromLTWH(0, 0, 50, 50), Shapes: []gpu.ClipShape{{
		Kind: gpu.ClipShapeRRect, RRect: ui.NewRRectFromRectXY(ui.NewRectFromLTWH(100, 100, 30, 30), 5, 5), AntiAlias: true, Transform: *ui.NewMatrix4Identity(),
	}}}
	if _, empty, _ := res.clipMask(out, 200, 100); !empty {
		t.Error("a shape outside the bounds should leave nothing")
	}
	// A clip with no area.
	if _, empty, _ := res.clipMask(gpu.Clip{Bounds: ui.Rect{Left: 5, Top: 5, Right: 5, Bottom: 5}}, 200, 100); !empty {
		t.Error("an empty clip should be empty")
	}
}

func TestClipMasksAreDroppedWhenIdle(t *testing.T) {
	res, d, _ := newTestResources(t)
	frameWith(t, res, func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 10, 100, 60), 20, 20))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 200, 100), solidPaint(0xFF000000))
	})
	if len(res.clipMasks) != 1 || res.clipBytes == 0 {
		t.Fatalf("masks %d, bytes %d", len(res.clipMasks), res.clipBytes)
	}
	textures := d.Stats().Textures
	for i := 0; i < idleFrames+2; i++ {
		res.beginFrame()
		res.endFrame()
	}
	if len(res.clipMasks) != 0 || res.clipBytes != 0 {
		t.Errorf("after idling: masks %d, bytes %d", len(res.clipMasks), res.clipBytes)
	}
	if got := d.Stats(); got.FreeTextures == 0 || got.Textures != textures {
		t.Errorf("the mask's texture should be back in the pool: %+v", got)
	}
	// Over the budget the least recently used goes, but never one of this frame.
	res.clipBytes = clipMaskBudget + 1
	res.beginFrame()
	m := &clipMask{lastUsed: res.frame}
	tex, _ := d.AcquireTexture(4, 4, TextureFormatR8)
	m.tex = tex
	res.clipMasks[1] = m
	res.endFrame()
	if _, ok := res.clipMasks[1]; !ok {
		t.Error("a mask used in this frame was dropped over the budget")
	}
	res.beginFrame()
	res.endFrame()
	if _, ok := res.clipMasks[1]; ok {
		t.Error("a mask of an earlier frame stayed over the budget")
	}
}

func testImage(w, h int) *ui.Image { return ui.NewImageFromRGBA(make([]byte, 4*w*h), w, h) }

func TestImagesAreUploadedOnceWithMipmaps(t *testing.T) {
	res, _, gl := newTestResources(t)
	img := testImage(64, 32)
	draw := func(c *ui.Canvas) {
		c.DrawImage(img, ui.Offset{Dx: 5, Dy: 5}, nil)
		c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(100, 5, 20, 20), nil)
		c.DrawImage(img.Clone(), ui.Offset{Dx: 5, Dy: 50}, nil) // a clone shares the pixels
	}
	b := frameWith(t, res, draw)
	if res.imagesUploaded != 1 || b.frame.items(pipeImage) != 3 {
		t.Fatalf("uploads %d, instances %d", res.imagesUploaded, b.frame.items(pipeImage))
	}
	if gl.calls["generateMipmap"] != 1 {
		t.Errorf("generateMipmap calls = %d", gl.calls["generateMipmap"])
	}
	// The three draws share a texture and a state: one call.
	if b.frame.callsOf(pipeImage) != 1 || b.frame.calls[0].tex == nil {
		t.Errorf("calls = %d", b.frame.callsOf(pipeImage))
	}
	frameWith(t, res, draw)
	if res.imagesUploaded != 1 {
		t.Errorf("the image was uploaded again: %d", res.imagesUploaded)
	}
	// Another image is another texture and another call.
	other := testImage(16, 16)
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.DrawImage(img, ui.Offset{}, nil)
		c.DrawImage(other, ui.Offset{}, nil)
		c.DrawImage(img, ui.Offset{}, nil)
	})
	if b.frame.callsOf(pipeImage) != 3 || res.imagesUploaded != 2 {
		t.Errorf("calls %d, uploads %d", b.frame.callsOf(pipeImage), res.imagesUploaded)
	}
	// An image that is not used for long is dropped and its texture goes to the pool.
	for i := 0; i < idleFrames+2; i++ {
		res.beginFrame()
		res.endFrame()
	}
	if len(res.images) != 0 || res.imageBytes != 0 {
		t.Errorf("after idling: images %d, bytes %d", len(res.images), res.imageBytes)
	}
}

func TestImageInstances(t *testing.T) {
	res, _, _ := newTestResources(t)
	img := testImage(64, 64)
	q := func(quality ui.FilterQuality) *ui.Paint {
		p := ui.NewPaint()
		p.SetFilterQuality(quality)
		return p
	}
	tests := []struct {
		name    string
		draw    func(c *ui.Canvas)
		quality float32
		level   float32
		box     bool
	}{
		{"nearest", func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, q(ui.FilterQualityNone)) }, 0, 0, true},
		{"low", func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, q(ui.FilterQualityLow)) }, 1, 0, true},
		{"medium at full size", func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, q(ui.FilterQualityMedium)) }, 2, 0, true},
		{"medium at a quarter", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 16, 16), q(ui.FilterQualityMedium))
		}, 2, 2, true},
		{"high at an eighth", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 8, 8), q(ui.FilterQualityHigh))
		}, 3, 3, true},
		{"low never uses a mip level", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 8, 8), q(ui.FilterQualityLow))
		}, 1, 0, true},
		{"upscaled medium", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 128, 128), q(ui.FilterQualityMedium))
		}, 2, 0, true},
		{"rotated is not a box", func(c *ui.Canvas) {
			c.Rotate(0.3)
			c.DrawImage(img, ui.Offset{}, q(ui.FilterQualityLow))
		}, 1, 0, false},
		{"a quarter turn is a box", func(c *ui.Canvas) {
			c.Translate(100, 0)
			c.Rotate(1.5707963267948966)
			c.DrawImage(img, ui.Offset{}, q(ui.FilterQualityLow))
		}, 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := frameWith(t, res, tt.draw)
			if b.frame.items(pipeImage) != 1 {
				t.Fatalf("%d instances", b.frame.items(pipeImage))
			}
			in := b.frame.data[pipeImage]
			if in[25] != tt.quality || in[24] != tt.level {
				t.Errorf("quality %v level %v, want %v and %v", in[25], in[24], tt.quality, tt.level)
			}
			if got := int(in[10])&imageFlagBox != 0; got != tt.box {
				t.Errorf("box flag = %v, want %v", got, tt.box)
			}
			if in[26] != -1 {
				t.Errorf("blend index = %v, want none", in[26])
			}
		})
	}
}

func TestImageDrawingEdgeCases(t *testing.T) {
	res, _, _ := newTestResources(t)
	img := testImage(32, 32)
	count := func(fn func(c *ui.Canvas)) int { return frameWith(t, res, fn).frame.items(pipeImage) }
	// Parts of the source outside the image are cut, with the matching part of the destination.
	b := frameWith(t, res, func(c *ui.Canvas) {
		c.DrawImageRect(img, ui.NewRectFromLTWH(16, 16, 32, 32), ui.NewRectFromLTWH(0, 0, 64, 64), nil)
	})
	in := b.frame.data[pipeImage]
	if b.frame.items(pipeImage) != 1 || in[20] != 0 || in[22] != 32 {
		t.Errorf("the destination was not cut with the source: %v", in[20:24])
	}
	for name, n := range map[string]int{
		"an empty source": count(func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 0, 5), ui.NewRectFromLTWH(0, 0, 9, 9), nil)
		}),
		"an empty destination": count(func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 5, 5), ui.NewRectFromLTWH(0, 0, 0, 9), nil)
		}),
		"a source off the image": count(func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(40, 40, 5, 5), ui.NewRectFromLTWH(0, 0, 9, 9), nil)
		}),
		"off the surface":      count(func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{Dx: 500, Dy: 500}, nil) }),
		"a singular transform": count(func(c *ui.Canvas) { c.Scale(0, 1); c.DrawImage(img, ui.Offset{}, nil) }),
	} {
		if n != 0 {
			t.Errorf("%s drew %d instances", name, n)
		}
	}
	// The paint's alpha scales the image; a shader and mask filter do not apply to images.
	faded := ui.NewPaint()
	faded.SetColor(ui.NewColor(0x80FFFFFF))
	faded.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 5}, []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)}))
	faded.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 2))
	b = frameWith(t, res, func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, faded) })
	if got := b.frame.data[pipeImage][12]; got < 0.49 || got > 0.51 {
		t.Errorf("alpha multiplier = %v", got)
	}
}

func TestImageErrorsAndUnsupported(t *testing.T) {
	res, _, _ := newTestResources(t)
	disposed := testImage(4, 4)
	disposed.Dispose()
	for name, fn := range map[string]func(c *ui.Canvas){
		"image": func(c *ui.Canvas) { c.DrawImage(disposed, ui.Offset{}, nil) },
		"rect": func(c *ui.Canvas) {
			c.DrawImageRect(disposed, ui.NewRectFromLTWH(0, 0, 4, 4), ui.NewRectFromLTWH(0, 0, 8, 8), nil)
		},
		"nine": func(c *ui.Canvas) {
			c.DrawImageNine(disposed, ui.NewRectFromLTWH(1, 1, 2, 2), ui.NewRectFromLTWH(0, 0, 8, 8), nil)
		},
		"atlas": func(c *ui.Canvas) { c.DrawAtlas(disposed, nil, nil, nil, ui.BlendModeSrc, nil, nil) },
		"nil":   func(c *ui.Canvas) { c.DrawImage(nil, ui.Offset{}, nil) },
	} {
		res.beginFrame()
		_, err := buildWith(t, res, false, fn)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GLImageDisposed" {
			t.Errorf("%s: error = %v, want GLImageDisposed", name, err)
		}
	}
}

func TestNinePatchAndAtlasInstances(t *testing.T) {
	res, _, _ := newTestResources(t)
	img := testImage(24, 24)
	center := ui.NewRectFromLTWH(8, 8, 8, 8)
	b := frameWith(t, res, func(c *ui.Canvas) { c.DrawImageNine(img, center, ui.NewRectFromLTWH(10, 10, 80, 50), nil) })
	if n := b.frame.items(pipeImage); n != 9 {
		t.Errorf("a nine-patch is %d instances, want 9", n)
	}
	// Smaller than its corners the middle vanishes.
	b = frameWith(t, res, func(c *ui.Canvas) { c.DrawImageNine(img, center, ui.NewRectFromLTWH(10, 10, 8, 8), nil) })
	if n := b.frame.items(pipeImage); n != 4 {
		t.Errorf("a squeezed nine-patch is %d instances, want 4", n)
	}
	// The pieces of one axis.
	if got := ninePieces(10, 100, [4]float64{0, 8, 16, 24}); got != [4]float64{10, 18, 102, 110} {
		t.Errorf("ninePieces = %v", got)
	}
	if got := ninePieces(0, 8, [4]float64{0, 8, 16, 24}); got != [4]float64{0, 4, 4, 8} {
		t.Errorf("squeezed ninePieces = %v", got)
	}
	// Sprites, with and without colors.
	xf := []ui.RSTransform{ui.NewRSTransformFromComponents(0, 1, 0, 0, 10, 10), ui.NewRSTransformFromComponents(0.5, 2, 4, 4, 50, 10)}
	rects := []ui.Rect{ui.NewRectFromLTWH(0, 0, 8, 8), ui.NewRectFromLTWH(8, 8, 8, 8)}
	b = frameWith(t, res, func(c *ui.Canvas) { c.DrawAtlas(img, xf, rects, nil, ui.BlendModeSrc, nil, nil) })
	data := b.frame.data[pipeImage]
	if b.frame.items(pipeImage) != 2 || data[26] != -1 {
		t.Errorf("sprites: %d instances, blend %v", b.frame.items(pipeImage), data[26])
	}
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.DrawAtlas(img, xf, rects, []ui.Color{ui.NewColor(0x80FF0000), ui.NewColor(0xFF00FF00)}, ui.BlendModeSrcIn, nil, nil)
	})
	data = b.frame.data[pipeImage]
	if data[26] != 5 { // srcIn
		t.Errorf("blend index = %v, want 5", data[26])
	}
	if data[28] < 0.49 || data[28] > 0.51 || data[29] != 0 || data[31] < 0.49 || data[31] > 0.51 { // premultiplied 50% red
		t.Errorf("tint = %v", data[28:32])
	}
	// A sprite outside the atlas is skipped.
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.DrawAtlas(img, xf[:1], []ui.Rect{ui.NewRectFromLTWH(100, 100, 8, 8)}, nil, ui.BlendModeSrc, nil, nil)
	})
	if b.frame.items(pipeImage) != 0 {
		t.Error("a sprite outside the atlas was drawn")
	}
}

func TestBlendIndex(t *testing.T) {
	// Every mode is its place in the order of BlendMode, as shaders/blend.glsl lists them.
	names := []string{
		"clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn", "srcOut", "dstOut", "srcATop", "dstATop", "xor",
		"plus", "modulate", "screen", "overlay", "darken", "lighten", "colorDodge", "colorBurn", "hardLight", "softLight",
		"difference", "exclusion", "multiply", "hue", "saturation", "color", "luminosity",
	}
	for i, name := range names {
		m := ui.BlendModeClear + ui.BlendMode(i)
		if got := m.String(); got != "BlendMode."+name {
			t.Fatalf("mode %d is %s, want %s", i, got, name)
		}
		if got, ok := blendIndex(m); !ok || got != i {
			t.Errorf("blendIndex(%v) = %d, %v, want %d", m, got, ok, i)
		}
	}
	if got, ok := blendIndex(ui.BlendModeUnspecified); !ok || got != 3 {
		t.Errorf("an unspecified mode is srcOver: %d, %v", got, ok)
	}
	if _, ok := blendIndex(ui.BlendModeLuminosity + 1); ok {
		t.Error("a mode after luminosity has an index")
	}
}

func TestVerticesInstances(t *testing.T) {
	res, _, _ := newTestResources(t)
	pos := []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 60, Dy: 10}, {Dx: 10, Dy: 60}, {Dx: 60, Dy: 60}, {Dx: 80, Dy: 30}}
	colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0x80FFFF00), ui.NewColor(0xFFFFFFFF)}
	tests := []struct {
		name      string
		v         *ui.Vertices
		triangles int
	}{
		{"triangles", ui.NewVertices(ui.VertexModeTriangles, pos[:6-3], ui.VerticesOpts{}), 1},
		{"extra vertices of triangles", ui.NewVertices(ui.VertexModeTriangles, pos, ui.VerticesOpts{}), 1},
		{"strip", ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{}), 3},
		{"fan", ui.NewVertices(ui.VertexModeTriangleFan, pos, ui.VerticesOpts{}), 3},
		{"indices", ui.NewVertices(ui.VertexModeTriangles, pos, ui.VerticesOpts{Indices: []int{0, 1, 2, 1, 2, 3, 4, 3, 1}}), 3},
		{"too few vertices", ui.NewVertices(ui.VertexModeTriangleStrip, pos[:2], ui.VerticesOpts{}), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := frameWith(t, res, func(c *ui.Canvas) { c.DrawVertices(tt.v, ui.BlendModeDst, solidPaint(0xFF000000)) })
			if got := b.frame.items(pipeVertices); got != tt.triangles*3 {
				t.Errorf("%d vertices, want %d", got, tt.triangles*3)
			}
		})
	}
	// Positions move with the transform; colors ride along; the call carries the paint.
	b := frameWith(t, res, func(c *ui.Canvas) {
		c.Translate(100, 5)
		c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{Colors: colors[:3]}), ui.BlendModeModulate, solidPaint(0x80FF0000))
	})
	v := b.frame.data[pipeVertices]
	if v[0] != 110 || v[1] != 15 || v[2] != 1 || v[3] != 0 || v[5] != 1 {
		t.Errorf("first vertex = %v", v[:6])
	}
	call := b.frame.calls[0]
	if call.mode != 13 || !call.hasColors || call.paint[3] < 0.49 || call.paint[3] > 0.51 {
		t.Errorf("call = %+v", call.callKey)
	}
	// Two draws with the same state are one call; without colors the mode does not matter.
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{}), ui.BlendModeModulate, solidPaint(0xFF000000))
		c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{}), ui.BlendModeModulate, solidPaint(0xFF000000))
	})
	if b.frame.callsOf(pipeVertices) != 1 || b.frame.items(pipeVertices) != 6 {
		t.Errorf("calls %d, vertices %d", b.frame.callsOf(pipeVertices), b.frame.items(pipeVertices))
	}
	// Any blend mode of the call blends the paint with the vertex colors, and a
	// shader with texture coordinates is evaluated at them.
	b = frameWith(t, res, func(c *ui.Canvas) {
		c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{Colors: colors[:3]}), ui.BlendModeHardLight, solidPaint(0xFF000000))
		shader := ui.NewPaint()
		shader.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 5}, []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)}))
		c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{TextureCoordinates: pos[:3]}), ui.BlendModeSrc, shader)
	})
	if b.frame.calls[0].mode != 20 || !b.frame.calls[0].hasColors || b.frame.calls[0].atTexture {
		t.Errorf("first call = %+v", b.frame.calls[0].callKey)
	}
	if call := b.frame.calls[1]; !call.atTexture || call.spec.shader != shaderLinear || call.spec.xf != [6]float32{1, 0, 0, 1, 0, 0} {
		t.Errorf("second call = %+v, spec %+v", call.callKey, call.spec)
	}
}

func TestShapesFallBackToTheAtlas(t *testing.T) {
	res, _, _ := newTestResources(t)
	rect := ui.NewRectFromLTWH(20, 20, 60, 40)
	b := frameWith(t, res, func(c *ui.Canvas) {
		c.DrawRect(rect, linePaint(0xFF000000, 8, ui.StrokeCapButt, ui.StrokeJoinBevel))
		c.DrawDRRect(ui.NewRRectFromRectXY(rect, 8, 8), ui.NewRRectFromRectXY(rect.Deflate(8), 4, 4), linePaint(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(rect, 8, 8), linePaint(0xFF000000, 2, ui.StrokeCapButt, ui.StrokeJoinMiter))
		c.DrawPoints(ui.PointModePolygon, []ui.Offset{{Dx: 5, Dy: 5}, {Dx: 50, Dy: 40}, {Dx: 90, Dy: 10}}, linePaint(0xFF000000, 4, ui.StrokeCapRound, ui.StrokeJoinRound))
		p := linePaint(0xFF000000, 8, ui.StrokeCapButt, ui.StrokeJoinMiter)
		p.SetStrokeMiterLimit(1.1)
		c.DrawRect(rect, p)
	})
	if b.frame.items(pipeShape) != 0 || b.frame.items(pipeMask) < 5 || b.stats.Skipped != 0 {
		t.Errorf("shapes %d, tiles %d, skipped %d", b.frame.items(pipeShape), b.frame.items(pipeMask), b.stats.Skipped)
	}
	if b.stats.Drawn != 5 {
		t.Errorf("drawn = %d", b.stats.Drawn)
	}
	// Features of the paint still stop a fallback.
	unfit := manyColorsPaint()
	unfit.SetStyle(ui.PaintingStyleStroke)
	unfit.SetStrokeJoin(ui.StrokeJoinBevel)
	res.beginFrame()
	_, err := buildWith(t, res, false, func(c *ui.Canvas) { c.DrawRect(rect, unfit) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GLNotImplemented" || appErr.Metadata["feature"] != "gradient with more than 32 colors" {
		t.Errorf("error = %v", err)
	}
}

func TestMaskKeysAreStableAndDistinct(t *testing.T) {
	a, ok := shapeOf(ui.OpDrawPath{Path: wave(), Paint: *solidPaint(0xFF000000)})
	b, _ := shapeOf(ui.OpDrawPath{Path: wave(), Paint: *solidPaint(0xFFFF0000)})
	if !ok || a.geom != b.geom {
		t.Error("the same path should have the same geometry hash")
	}
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	seen := map[uint64]string{}
	for name, op := range map[string]ui.DisplayOp{
		"path":   ui.OpDrawPath{Path: wave()},
		"rect":   ui.OpDrawRect{Rect: rect},
		"rrect":  ui.OpDrawRRect{RRect: ui.NewRRectFromRectXY(rect, 2, 2)},
		"super":  ui.OpDrawRSuperellipse{RSuperellipse: ui.NewRSuperellipseFromRectXY(rect, 2, 2)},
		"drrect": ui.OpDrawDRRect{Outer: ui.NewRRectFromRectXY(rect, 2, 2), Inner: ui.NewRRectFromRectXY(rect.Deflate(2), 1, 1)},
		"oval":   ui.OpDrawOval{Rect: rect},
		"circle": ui.OpDrawCircle{Radius: 5},
		"arc":    ui.OpDrawArc{Rect: rect, SweepAngle: 1},
		"arc2":   ui.OpDrawArc{Rect: rect, SweepAngle: 2},
		"arc3":   ui.OpDrawArc{Rect: rect, SweepAngle: 2, UseCenter: true},
		"line":   ui.OpDrawLine{P2: ui.Offset{Dx: 5, Dy: 5}},
		"points": ui.OpDrawPoints{Points: []ui.Offset{{Dx: 1, Dy: 1}}},
		"lines":  ui.OpDrawPoints{Mode: ui.PointModeLines, Points: []ui.Offset{{Dx: 1, Dy: 1}}},
	} {
		sh, ok := shapeOf(op)
		if !ok {
			t.Fatalf("%s: unknown", name)
		}
		if other, dup := seen[sh.geom]; dup {
			t.Errorf("%s and %s have the same geometry hash", name, other)
		}
		seen[sh.geom] = name
	}
	if _, ok := shapeOf(ui.OpSave{}); ok {
		t.Error("Save is not a shape")
	}
	// Strokes and anti-aliasing change the style, a color does not.
	fill, stroke, aliased := solidPaint(0xFF000000), linePaint(0xFF000000, 3, ui.StrokeCapButt, ui.StrokeJoinMiter), solidPaint(0xFF000000)
	aliased.SetIsAntiAlias(false)
	if styleKey(fill, false) == styleKey(stroke, true) || styleKey(fill, false) == styleKey(aliased, false) {
		t.Error("style keys of different coverage are equal")
	}
	if styleKey(fill, false) != styleKey(solidPaint(0xFF123456), false) {
		t.Error("a color changed the style key")
	}
	wide := linePaint(0xFF000000, 5, ui.StrokeCapButt, ui.StrokeJoinMiter)
	if styleKey(stroke, true) == styleKey(wide, true) {
		t.Error("stroke widths share a style key")
	}
}

func TestFloorDiv(t *testing.T) {
	for _, tt := range []struct{ a, b, want int }{{7, 4, 1}, {8, 4, 2}, {0, 4, 0}, {-1, 4, -1}, {-4, 4, -1}, {-5, 4, -2}, {255, 256, 0}, {256, 256, 1}} {
		if got := floorDiv(tt.a, tt.b); got != tt.want {
			t.Errorf("floorDiv(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
