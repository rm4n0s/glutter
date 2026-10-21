//go:build glutter_display

package gles

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// The tests of this file draw into the window's framebuffer, as the engine does, and not
// into a framebuffer of their own as renderGPU does. A shader that works out where a pixel
// is from gl_FragCoord drew a layer's contents in the wrong rows when the window's
// framebuffer was the one under the layer (Mesa's llvmpipe), so every cached layer came out
// empty and a search box flickered when its layer was made and used.

// windowFrames makes the scenes of a retained layer over a window of w x h pixels: the layer
// is built in the first and added again in the others, under something that changes in
// every frame.
func windowFrames(n, w, h int, inside func(b *ui.SceneBuilder)) []*ui.Scene {
	under := pictureOf(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) })
	var scenes []*ui.Scene
	var kept ui.EngineLayer
	for i := 0; i < n; i++ {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, under)
		if kept == nil {
			kept = b.PushOffset(12, 10)
			inside(b)
			b.Pop()
		} else {
			b.AddRetained(kept)
		}
		x := 8 + float64(i)*float64(w-16)/float64(n)
		b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
			c.DrawCircle(ui.Offset{Dx: x, Dy: float64(h) - 10}, 5, fillP(0xFF1E88E5))
		}))
		scenes = append(scenes, b.Build())
	}
	return scenes
}

// frameDiff is the difference of the window's frame from the software renderer's.
func frameDiff(got *image.RGBA, scene *ui.Scene) (diffStats, error) {
	want, err := soft.Render(scene, got.Bounds().Dx(), got.Bounds().Dy())
	if err != nil {
		return diffStats{}, err
	}
	return diffRect(got, want, got.Bounds()), nil
}

func TestDisplayWindowRasterCacheMatchesSoftware(t *testing.T) {
	tests := []struct {
		name   string
		inside func(b *ui.SceneBuilder)
		tol    tolerance
	}{
		{"a rectangle", func(b *ui.SceneBuilder) {
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 30), fillP(0xFFE53935)) }))
		}, tolerance{over8Permille: 1}},
		{"a rounded rectangle and a circle", func(b *ui.SceneBuilder) {
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) {
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 70, 30), 8, 8), fillP(0xFF43A047))
				c.DrawCircle(ui.Offset{Dx: 100, Dy: 20}, 12, fillP(0xCCFB8C00))
			}))
		}, tolerance{over8Permille: 8, over32Permille: 2}},
		{"inside an opacity layer", func(b *ui.SceneBuilder) {
			b.PushOpacity(140)
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 30), fillP(0xFFE53935)) }))
			b.Pop()
		}, tolerance{over8Permille: 1}},
		{"under a clip", func(b *ui.SceneBuilder) {
			b.PushClipRect(ui.NewRectFromLTWH(0, 0, 40, 20))
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 60, 30), fillP(0xFF8E24AA)) }))
			b.Pop()
		}, tolerance{over8Permille: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onDisplay(t, func(h gpu.SurfaceHandle, w, hgt int) error {
				scenes := windowFrames(5, w, hgt, tt.inside)
				r := NewRenderer(RendererOpts{RasterCacheAfter: 2})
				defer func() { _ = r.Close() }()
				surface, err := r.Attach(h)
				if err != nil {
					return err
				}
				for i, scene := range scenes {
					got, err := r.RenderCapture(surface, scene, ui.Size{Width: float64(w), Height: float64(hgt)}, 1)
					if err != nil {
						return err
					}
					st := r.LastFrame()
					switch {
					case i == 1 && st.CachedLayersMade != 1:
						t.Errorf("frame 2: %+v", st)
					case i >= 2 && st.CachedLayersDrawn != 1:
						t.Errorf("frame %d: %+v", i+1, st)
					}
					d, err := frameDiff(got, scene)
					if err != nil {
						return err
					}
					tt.tol.check(t, "frame "+string(rune('1'+i)), d)
				}
				return nil
			})
		})
	}
}

func TestDisplayWindowLayersMatchSoftware(t *testing.T) {
	blur := ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 4, SigmaY: 4, TileMode: ui.TileModeDecal})
	tests := []struct {
		name  string
		build func(b *ui.SceneBuilder)
		tol   tolerance
	}{
		{"an opacity layer", func(b *ui.SceneBuilder) {
			b.PushOpacity(128)
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(10, 10, 80, 40), fillP(0xFFE53935)) }))
			b.Pop()
		}, tolerance{over8Permille: 1}},
		{"a blurred layer", func(b *ui.SceneBuilder) {
			b.PushImageFilter(blur)
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(30, 20, 60, 40), fillP(0xFF1E88E5)) }))
			b.Pop()
		}, tolerance{over8Permille: 15, over32Permille: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onDisplay(t, func(h gpu.SurfaceHandle, w, hgt int) error {
				b := ui.NewSceneBuilder()
				b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) }))
				tt.build(b)
				scene := b.Build()
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				surface, err := r.Attach(h)
				if err != nil {
					return err
				}
				got, err := r.RenderCapture(surface, scene, ui.Size{Width: float64(w), Height: float64(hgt)}, 1)
				if err != nil {
					return err
				}
				d, err := frameDiff(got, scene)
				if err != nil {
					return err
				}
				tt.tol.check(t, tt.name, d)
				return nil
			})
		})
	}
}
