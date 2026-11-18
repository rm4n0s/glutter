package soft

import (
	"bytes"
	"image"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// picture is an image of w x h pixels of blocks of 16 pixels in two colors, which seed changes.
func picture(w, h int, seed uint8) *ui.Image {
	return imageFunc(w, h, func(x, y int) [4]uint8 {
		if (x/16+y/16)%2 == 0 {
			return [4]uint8{200, seed, 40, 255}
		}
		return [4]uint8{20, 90, seed, 255}
	})
}

// pictureScene draws the image at half its size with mipmaps, and a small square turned by turn
// quarter turns over it: a spinner over a picture.
func pictureScene(img *ui.Image, turn int) *ui.Scene {
	return sceneOf(ui.Offset{}, func(c *ui.Canvas) {
		w, h := float64(img.Width()), float64(img.Height())
		c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(0, 0, w/2, h/2), withQuality(ui.FilterQualityMedium))
		c.Save()
		c.Translate(w/4, h/4)
		c.Rotate(float64(turn) * 0.4)
		c.DrawRect(ui.NewRectFromLTWH(-10, -10, 20, 20), paint(0xFF2060FF))
		c.Restore()
	})
}

// TestCacheKeepsTheMipmapsOfAPicture draws the frames of a spinner over a picture that is scaled
// down, as a window's damage tracker does (the spinner's region only), and checks that the picture's
// chain is made once and kept, and that the pixels are those of a drawing without a cache. Before
// step 49.1 every frame made the chain again (the issue of Fishball).
func TestCacheKeepsTheMipmapsOfAPicture(t *testing.T) {
	img := picture(1024, 768, 0)
	c := NewCache()
	dst := image.NewRGBA(image.Rect(0, 0, 512, 384))
	spin := image.Rect(226, 162, 286, 222)
	for frame := 0; frame < 6; frame++ {
		scene := pictureScene(img, frame)
		region := spin
		if frame == 0 {
			region = dst.Rect
		}
		clear(dst.Pix) // the region is drawn over transparent pixels, as a surface does
		if err := c.RenderInto(dst, scene, region); err != nil {
			t.Fatal(err)
		}
		c.EndFrame()
		st := c.Stats()
		// Level 1 only: 512 x 384 pixels.
		if st.Built != 1 || st.Chains != 1 || st.Bytes != 4*512*384 || st.Dropped != 0 {
			t.Errorf("frame %d: stats %+v, want one chain built, kept, with level 1 only", frame, st)
		}
		want, err := Render(scene, 512, 384)
		if err != nil {
			t.Fatal(err)
		}
		for y := region.Min.Y; y < region.Max.Y; y++ {
			o := dst.PixOffset(region.Min.X, y)
			if !bytes.Equal(dst.Pix[o:o+4*region.Dx()], want.Pix[o:o+4*region.Dx()]) {
				t.Fatalf("frame %d: row %d differs from a drawing without the cache", frame, y)
			}
		}
	}
}

func TestCacheMakesOnlyTheLevelsADrawNeeds(t *testing.T) {
	tests := []struct {
		name     string
		dst      ui.Rect
		quality  ui.FilterQuality
		wantByte int
	}{
		{"the image's own size", ui.NewRectFromLTWH(0, 0, 64, 64), ui.FilterQualityMedium, 0},
		{"larger", ui.NewRectFromLTWH(0, 0, 128, 128), ui.FilterQualityHigh, 0},
		{"smaller without mipmaps", ui.NewRectFromLTWH(0, 0, 16, 16), ui.FilterQualityLow, 0},
		{"half", ui.NewRectFromLTWH(0, 0, 32, 32), ui.FilterQualityMedium, 4 * 32 * 32},
		{"a quarter", ui.NewRectFromLTWH(0, 0, 16, 16), ui.FilterQualityMedium, 4 * (32*32 + 16*16)},
		{"one pixel", ui.NewRectFromLTWH(0, 0, 1, 1), ui.FilterQualityHigh, 4 * (32*32 + 16*16 + 8*8 + 4*4 + 2*2 + 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := picture(64, 64, 1)
			c := NewCache()
			scene := sceneOf(ui.Offset{}, func(cv *ui.Canvas) {
				cv.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 64, 64), tt.dst, withQuality(tt.quality))
			})
			if _, err := c.Render(scene, 128, 128); err != nil {
				t.Fatal(err)
			}
			if st := c.Stats(); st.Bytes != tt.wantByte || st.Chains != 1 {
				t.Errorf("stats %+v, want %d bytes of levels in one chain", st, tt.wantByte)
			}
		})
	}
}

func TestCacheSharesAChainBetweenClones(t *testing.T) {
	img := picture(64, 64, 2)
	clone := img.Clone()
	c := NewCache()
	for _, im := range []*ui.Image{img, clone, img} {
		scene := sceneOf(ui.Offset{}, func(cv *ui.Canvas) {
			cv.DrawImageRect(im, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 32, 32), withQuality(ui.FilterQualityMedium))
		})
		if _, err := c.Render(scene, 32, 32); err != nil {
			t.Fatal(err)
		}
		c.EndFrame()
	}
	if st := c.Stats(); st.Built != 1 || st.Chains != 1 {
		t.Errorf("stats %+v, want one chain for the image and its clone", st)
	}
	// Another image of the same size has a chain of its own.
	other := picture(64, 64, 3)
	scene := sceneOf(ui.Offset{}, func(cv *ui.Canvas) {
		cv.DrawImageRect(other, ui.NewRectFromLTWH(0, 0, 64, 64), ui.NewRectFromLTWH(0, 0, 32, 32), withQuality(ui.FilterQualityMedium))
	})
	if _, err := c.Render(scene, 32, 32); err != nil {
		t.Fatal(err)
	}
	if st := c.Stats(); st.Built != 2 || st.Chains != 2 {
		t.Errorf("stats %+v, want a second chain for another image", st)
	}
}

// TestCacheDropsChains checks when EndFrame drops a chain: an image not drawn for gpu.IdleFrames
// frames, one whose handle was disposed, and the oldest while the levels are over the budget.
func TestCacheDropsChains(t *testing.T) {
	half := func(img *ui.Image) *ui.Scene {
		return sceneOf(ui.Offset{}, func(cv *ui.Canvas) {
			w, h := float64(img.Width()), float64(img.Height())
			cv.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(0, 0, w/2, h/2), withQuality(ui.FilterQualityMedium))
		})
	}
	draw := func(t *testing.T, c *Cache, imgs ...*ui.Image) {
		t.Helper()
		for _, img := range imgs {
			if _, err := c.Render(half(img), 32, 32); err != nil {
				t.Fatal(err)
			}
		}
		c.EndFrame()
	}
	tests := []struct {
		name string
		run  func(t *testing.T, c *Cache)
		want CacheStats
	}{
		{"kept for IdleFrames frames", func(t *testing.T, c *Cache) {
			draw(t, c, picture(64, 64, 4))
			for i := 0; i < gpu.IdleFrames; i++ {
				c.EndFrame()
			}
		}, CacheStats{Chains: 1, Bytes: 4 * 32 * 32, Built: 1}},
		{"dropped after IdleFrames frames", func(t *testing.T, c *Cache) {
			draw(t, c, picture(64, 64, 4))
			for i := 0; i < gpu.IdleFrames+1; i++ {
				c.EndFrame()
			}
		}, CacheStats{Built: 1, Dropped: 1}},
		{"kept while drawn", func(t *testing.T, c *Cache) {
			img := picture(64, 64, 4)
			for i := 0; i < 2*gpu.IdleFrames; i++ {
				draw(t, c, img)
			}
		}, CacheStats{Chains: 1, Bytes: 4 * 32 * 32, Built: 1}},
		{"dropped when its handle is disposed", func(t *testing.T, c *Cache) {
			img := picture(64, 64, 5)
			draw(t, c, img)
			img.Dispose()
			c.EndFrame()
		}, CacheStats{Built: 1, Dropped: 1}},
		{"dropped in the frame its handle is disposed", func(t *testing.T, c *Cache) {
			img := picture(64, 64, 5)
			if _, err := c.Render(half(img), 32, 32); err != nil {
				t.Fatal(err)
			}
			img.Dispose()
			c.EndFrame()
		}, CacheStats{Built: 1, Dropped: 1}},
		{"a clone drawn after the dispose makes it again", func(t *testing.T, c *Cache) {
			img := picture(64, 64, 5)
			clone := img.Clone()
			draw(t, c, img)
			img.Dispose()
			c.EndFrame()
			draw(t, c, clone)
		}, CacheStats{Chains: 1, Bytes: 4 * 32 * 32, Built: 2, Dropped: 1}},
		{"the oldest over the budget", func(t *testing.T, c *Cache) {
			c.budget = 2 * 4 * 32 * 32
			a, b, d := picture(64, 64, 6), picture(64, 64, 7), picture(64, 64, 8)
			draw(t, c, a)
			draw(t, c, b)
			draw(t, c, d) // three chains: a goes
			draw(t, c, b) // b is drawn again: d is now the oldest
			draw(t, c, a) // a again: d goes
		}, CacheStats{Chains: 2, Bytes: 2 * 4 * 32 * 32, Built: 4, Dropped: 2}},
		{"none drawn in the frame over the budget", func(t *testing.T, c *Cache) {
			c.budget = 4 * 32 * 32
			draw(t, c, picture(64, 64, 6), picture(64, 64, 7), picture(64, 64, 8))
		}, CacheStats{Chains: 3, Bytes: 3 * 4 * 32 * 32, Built: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCache()
			tt.run(t, c)
			if got := c.Stats(); got != tt.want {
				t.Errorf("stats %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestCacheOnSeveralGoroutines draws the same images with one cache on several goroutines at once
// (run with -race).
func TestCacheOnSeveralGoroutines(t *testing.T) {
	imgs := []*ui.Image{picture(128, 96, 9), picture(128, 96, 10)}
	c := NewCache()
	want := make([]*image.RGBA, len(imgs))
	for i, img := range imgs {
		w, err := Render(pictureScene(img, 1), 64, 48)
		if err != nil {
			t.Fatal(err)
		}
		want[i] = w
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := g % len(imgs)
			got, err := c.Render(pictureScene(imgs[i], 1), 64, 48)
			if err != nil {
				errs <- err
				return
			}
			if !bytes.Equal(got.Pix, want[i].Pix) {
				t.Errorf("goroutine %d drew other pixels", g)
			}
			c.EndFrame()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if st := c.Stats(); st.Built != len(imgs) {
		t.Errorf("stats %+v, want one chain for each image", st)
	}
}

func TestCacheFailsAsTheFunctionsDo(t *testing.T) {
	scene := sceneOf(ui.Offset{}, func(*ui.Canvas) {})
	tests := []struct {
		name  string
		do    func(c *Cache) error
		route string
	}{
		{"RenderInto without a scene", func(c *Cache) error {
			return c.RenderInto(image.NewRGBA(image.Rect(0, 0, 1, 1)), nil, image.Rect(0, 0, 1, 1))
		}, "Cache.RenderInto.SoftSceneNil"},
		{"RenderInto into nothing", func(c *Cache) error { return c.RenderInto(nil, scene, image.Rect(0, 0, 1, 1)) }, "Cache.RenderInto.SoftTargetInvalid"},
		{"Render without a scene", func(c *Cache) error { _, err := c.Render(nil, 1, 1); return err }, "Cache.Render.SoftSceneNil"},
		{"Render of no size", func(c *Cache) error { _, err := c.Render(scene, 0, 1); return err }, "Cache.Render.SoftSizeInvalid"},
		{"a disposed image", func(c *Cache) error {
			img := picture(4, 4, 0)
			img.Dispose()
			_, err := c.Render(sceneOf(ui.Offset{}, func(cv *ui.Canvas) { cv.DrawImage(img, ui.Offset{}, ui.NewPaint()) }), 4, 4)
			return err
		}, "Cache.Render->renderInto->renderer.layer->renderer.children->renderer.layer->renderer.picture->renderer.draw->renderer.drawOp->renderer.drawImageRect->checkImage.SoftImageDisposed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.do(NewCache())
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

// BenchmarkSpinnerOverAPicture draws the spinner's region of a frame over a picture of 1024 x 768
// scaled to half its size: with a cache kept across frames (the fix of step 49.1), and with a cache
// for each frame (as every frame was drawn before it).
func BenchmarkSpinnerOverAPicture(b *testing.B) {
	img := picture(1024, 768, 0)
	spin := image.Rect(226, 162, 286, 222)
	for _, kept := range []bool{true, false} {
		name := "kept"
		if !kept {
			name = "each frame"
		}
		b.Run(name, func(b *testing.B) {
			dst := image.NewRGBA(image.Rect(0, 0, 512, 384))
			c := NewCache()
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				if !kept {
					c = NewCache()
				}
				if err := c.RenderInto(dst, pictureScene(img, i), spin); err != nil {
					b.Fatal(err)
				}
				c.EndFrame()
			}
		})
	}
}
