//go:build glutter_display

package gles

import (
	"bytes"
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplaySwapsWithDamage presents frames with eglSwapBuffersWithDamageKHR through
// the compositor's EGL (Mesa), and checks that the frames are the same pixels as the
// ones drawn without it, and that the damage the renderer reports is the changed part.
func TestDisplaySwapsWithDamage(t *testing.T) {
	still := damagePicture(ui.NewRectFromLTWH(0, 0, 50, 50))
	m100 := damagePicture(ui.NewRectFromLTWH(100, 20, 30, 30))
	m120 := damagePicture(ui.NewRectFromLTWH(120, 20, 30, 30))
	scenes := func() []*ui.Scene {
		return []*ui.Scene{
			damageScene(still, m100),
			damageScene(still, m100),
			damageScene(still, m120),
			damageScene(still, m120),
			damageScene(still),
		}
	}
	type frame struct {
		pix   []byte
		stats FrameStats
	}
	draw := func(h gpu.SurfaceHandle, opts RendererOpts, w, ht int, support *bool) ([]frame, error) {
		r := NewRenderer(opts)
		defer func() { _ = r.Close() }()
		s, err := r.Attach(h)
		if err != nil {
			return nil, err
		}
		*support = r.ctx.CanSwapWithDamage()
		var out []frame
		for _, scene := range scenes() {
			img, err := r.RenderCapture(s, scene, ui.Size{Width: float64(w), Height: float64(ht)}, 1)
			if err != nil {
				return nil, err
			}
			out = append(out, frame{pix: img.Pix, stats: r.LastFrame()})
		}
		return out, nil
	}
	var supported bool
	onDisplay(t, func(h gpu.SurfaceHandle, w, ht int) error {
		var support bool
		with, err := draw(h, RendererOpts{Damage: true}, w, ht, &support)
		if err != nil {
			return err
		}
		supported = support
		var unused bool
		without, err := draw(h, RendererOpts{}, w, ht, &unused)
		if err != nil {
			return err
		}
		for i := range with {
			if !bytes.Equal(with[i].pix, without[i].pix) {
				t.Errorf("frame %d differs from the frame drawn without damage", i)
			}
			if without[i].stats.DamagedPixels != 0 {
				t.Errorf("frame %d: a renderer without the option reports damage %+v", i, without[i].stats)
			}
		}
		if !support {
			return nil
		}
		// The tiles are 64 pixels: the pictures at x 99 to 151 are in the tiles of x 64 to
		// 192, and y 19 to 51 in the first row.
		whole, tiles := w*ht, 128*64
		want := []struct{ pixels, rects int }{
			{whole, 0}, // the first frame is presented whole
			{1, 1},     // the same pictures: a pixel
			{tiles, 1}, // the picture moved
			{1, 1},     // the same pictures again
			{tiles, 1}, // the picture went away
		}
		for i, f := range with {
			if f.stats.DamagedPixels != want[i].pixels || f.stats.DamageRects != want[i].rects {
				t.Errorf("frame %d: damage %d pixels in %d rectangles, want %d in %d", i, f.stats.DamagedPixels, f.stats.DamageRects, want[i].pixels, want[i].rects)
			}
		}
		return nil
	})
	if !supported {
		t.Log("the display has no swap-with-damage extension: the frames were presented whole")
	}
}

// TestDisplayDamageSwapSurvivesManyFrames presents a long run of frames with damage,
// resizing nothing, and checks that the swaps keep working.
func TestDisplayDamageSwapSurvivesManyFrames(t *testing.T) {
	onDisplay(t, func(h gpu.SurfaceHandle, w, ht int) error {
		r := NewRenderer(RendererOpts{Damage: true, RasterCacheAfter: 3})
		defer func() { _ = r.Close() }()
		s, err := r.Attach(h)
		if err != nil {
			return err
		}
		still := damagePicture(ui.NewRectFromLTWH(0, 0, 50, 50))
		for i := 0; i < 60; i++ {
			scene := damageScene(still, damagePicture(ui.NewRectFromLTWH(float64(i), 20, 30, 30)))
			if err := r.Render(s, scene, ui.Size{Width: float64(w), Height: float64(ht)}, 1); err != nil {
				t.Logf("frame %d failed", i)
				return err
			}
		}
		_ = image.Rectangle{}
		return nil
	})
}
