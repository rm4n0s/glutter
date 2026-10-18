//go:build glutter_display

package gles

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// proofTolerances is how far the GPU may be from the software renderer in each proof
// scene (the scenes are the goldens of the proof tool, SPEC §9.2). A scene that is
// registered without an entry here fails the parity test, so a step that adds one
// states its tolerance.
var proofTolerances = map[string]tolerance{
	"shapes-basic": {over8Permille: 20, over32Permille: 5},
	"shapes-clips": {over8Permille: 30, over32Permille: 3},
	"gradients":    {over8Permille: 10, over32Permille: 3},
	"blend-modes":  {over8Permille: 15, over32Permille: 5},
	"effects":      {over8Permille: 1},
}

func TestDisplayEveryProofSceneMatchesSoftware(t *testing.T) {
	for _, name := range proof.Names() {
		tol, ok := proofTolerances[name]
		if !ok {
			t.Errorf("the proof scene %q has no tolerance in proofTolerances", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			sc, err := proof.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			scene := sc.Build()
			want, err := soft.Render(scene, sc.Width, sc.Height)
			if err != nil {
				t.Fatal(err)
			}
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				surface, err := r.Attach(h)
				if err != nil {
					return err
				}
				got, err := renderGPU(r, scene, sc.Width, sc.Height)
				if err != nil {
					return err
				}
				if st := r.LastFrame(); st.Skipped != 0 {
					t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
				}
				tol.check(t, name, diffRect(got, want, got.Bounds()))
				// A snapshot is the same pixels as the frame, drawn without a window.
				snap, err := r.Snapshot(scene, ui.Size{Width: float64(sc.Width), Height: float64(sc.Height)})
				if err != nil {
					return err
				}
				if s := diffRect(snap, got, got.Bounds()); s.max != 0 {
					t.Errorf("the snapshot differs from the frame by up to %d levels", s.max)
				}
				// And with no window at all.
				surface.Destroy()
				bare, err := r.Snapshot(scene, ui.Size{Width: float64(sc.Width), Height: float64(sc.Height)})
				if err != nil {
					return err
				}
				if s := diffRect(bare, got, got.Bounds()); s.max != 0 {
					t.Errorf("the snapshot without a window differs by up to %d levels", s.max)
				}
				return nil
			})
		})
	}
}

func TestDisplayToImageThroughTheGPU(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4, 4, 56, 40), 10, 10), fillP(0xFF3050C0))
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 64}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x2000FF00)}))
	c.DrawCircle(ui.Offset{Dx: 32, Dy: 24}, 16, g)
	pic := rec.EndRecording()
	want, err := pic.ToImage(64, 48, ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}})
	if err != nil {
		t.Fatal(err)
	}
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(h); err != nil {
			return err
		}
		got, err := pic.ToImage(64, 48, ui.ToImageOpts{Rasterizer: r})
		if err != nil {
			return err
		}
		if got.Width() != 64 || got.Height() != 48 {
			t.Errorf("image = %dx%d", got.Width(), got.Height())
		}
		a := &image.RGBA{Pix: got.Pixels(), Stride: 64 * 4, Rect: image.Rect(0, 0, 64, 48)}
		b := &image.RGBA{Pix: want.Pixels(), Stride: 64 * 4, Rect: image.Rect(0, 0, 64, 48)}
		tolerance{over8Permille: 15, over32Permille: 3}.check(t, "ToImage", diffRect(a, b, a.Bounds()))
		// The image is the pixels of a transparent surface: the corner has none.
		if a.Pix[3] != 0 {
			t.Errorf("the corner has alpha %d", a.Pix[3])
		}
		return nil
	})
}

// retainedFrames makes the scenes of a retained layer: the layer is built in the first
// and added again in the others, as a RepaintBoundary that does not change.
func retainedFrames(n int, clip func(b *ui.SceneBuilder)) []*ui.Scene {
	pic := func(fn func(c *ui.Canvas)) *ui.Picture { return pictureOf(fn) }
	inside := pic(func(c *ui.Canvas) {
		stripesAndDisc(c)
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 60, 40), 8, 8), fillP(0x80FFFFFF))
	})
	under := pic(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) })
	var scenes []*ui.Scene
	var kept *ui.OffsetEngineLayer
	for i := 0; i < n; i++ {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, under)
		if clip != nil {
			clip(b)
		}
		b.PushOffset(6.5, 9)
		if kept == nil {
			kept = b.PushOffset(0, 0)
			b.AddPicture(ui.Offset{}, inside)
			b.Pop()
		} else {
			b.AddRetained(kept)
		}
		b.Pop()
		if clip != nil {
			b.Pop()
		}
		// Something that changes in every frame, over the retained layer.
		b.AddPicture(ui.Offset{}, pic(func(c *ui.Canvas) {
			c.DrawCircle(ui.Offset{Dx: 10 + float64(i)*12, Dy: 110}, 6, fillP(0xFFE53935))
		}))
		scenes = append(scenes, b.Build())
	}
	return scenes
}

func TestDisplayRasterCacheDrawsWhatTheLayerDrew(t *testing.T) {
	tests := []struct {
		name string
		clip func(b *ui.SceneBuilder)
		tol  tolerance
	}{
		{"no clip", nil, tolerance{}},
		{"a rectangle", func(b *ui.SceneBuilder) { b.PushClipRect(ui.NewRectFromLTWH(10, 10, 100, 90)) }, tolerance{}},
		// The clip's edge applies to the cached group as a whole, so where content
		// overlaps at the anti-aliased edge the pixels differ a little.
		{"a rounded clip", func(b *ui.SceneBuilder) {
			b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10.5, 10.5, 100, 90), 20, 20))
		}, tolerance{over8Permille: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenes := retainedFrames(5, tt.clip)
			// What the frames look like without the cache.
			var want []*image.RGBA
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				plain := NewRenderer()
				defer func() { _ = plain.Close() }()
				if _, err := plain.Attach(h); err != nil {
					return err
				}
				for _, scene := range scenes {
					img, err := renderGPU(plain, scene, 128, 128)
					if err != nil {
						return err
					}
					want = append(want, img)
				}
				return nil
			})
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				cached := NewRenderer(RendererOpts{RasterCacheAfter: 2})
				defer func() { _ = cached.Close() }()
				if _, err := cached.Attach(h); err != nil {
					return err
				}
				var instances []int
				for i, scene := range scenes {
					got, err := renderGPU(cached, scene, 128, 128)
					if err != nil {
						return err
					}
					st := cached.LastFrame()
					instances = append(instances, st.Instances)
					tt.tol.check(t, "frame "+string(rune('1'+i)), diffRect(got, want[i], got.Bounds()))
					switch {
					case i == 1 && st.CachedLayersMade != 1:
						t.Errorf("frame 2: %+v", st)
					case i >= 2 && st.CachedLayersDrawn != 1:
						t.Errorf("frame %d: %+v", i+1, st)
					}
				}
				// The cached frames draw fewer things: the layer is one instance.
				if !(instances[2] < instances[0] && instances[3] == instances[2]) {
					t.Errorf("instances per frame = %v", instances)
				}
				return nil
			})
		})
	}
}
