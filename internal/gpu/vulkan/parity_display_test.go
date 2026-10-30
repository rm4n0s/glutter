//go:build linux && glutter_display

package vulkan

import (
	"bytes"
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// The parity test of step 3.6 for Vulkan: every proof scene drawn through the explicit renderer
// on lavapipe (validation on) matches the software renderer within the tolerance the GLES
// renderer has for it, nothing is skipped, and a snapshot of the scene is the same pixels as the
// frame. Then the raster cache, the damage and the snapshots of the renderer, as the tests of
// gles have them.

func TestVulkanEveryProofSceneMatchesSoftware(t *testing.T) {
	lavapipe(t)
	var jobs []*drawJob
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, &drawJob{name: name, scene: sc.Build(), width: sc.Width, height: sc.Height, snapshot: true})
	}
	drawOnWindow(t, jobs)
	for _, j := range jobs {
		t.Run(j.name, func(t *testing.T) {
			tol, ok := gputest.ProofTolerances[j.name]
			if !ok {
				t.Fatalf("the proof scene %q has no tolerance in gputest.ProofTolerances", j.name)
			}
			want, err := soft.Render(j.scene, j.width, j.height)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "vk-parity-"+j.name+"-gpu", j.got)
			dumpImage(t, "vk-parity-"+j.name+"-soft", want)
			t.Logf("frame: %+v", j.stats)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			if j.name == "shapes-clips" {
				// Compared by cell as before: the cells hold the shapes that need the most.
				for i, name := range cellNames {
					cell := image.Rect(i%4*160, i/4*160, i%4*160+160, i/4*160+160)
					gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, name, gputest.DiffRect(j.got, want, cell))
				}
			} else {
				tol.Check(t, j.name, gputest.DiffRect(j.got, want, j.got.Bounds()))
			}
			// A snapshot is the same pixels as the frame, drawn without a window over a clear
			// background; the frame's background is white, so scenes that do not cover it differ
			// in their transparent places only: it is compared where the frame is opaque white
			// or the snapshot is opaque.
			if j.snap == nil || j.snap.Bounds() != j.got.Bounds() {
				t.Fatalf("the snapshot is %v, want %v", j.snap, j.got.Bounds())
			}
			var differ int
			for i := 0; i < len(j.snap.Pix); i += 4 {
				if j.snap.Pix[i+3] != 255 {
					continue // transparent or translucent: the frame has the white behind it
				}
				for k := 0; k < 4; k++ {
					d := int(j.snap.Pix[i+k]) - int(j.got.Pix[i+k])
					if d < -1 || d > 1 {
						differ++
						break
					}
				}
			}
			if differ != 0 {
				t.Errorf("%d opaque pixels of the snapshot differ from the frame by more than 1 level", differ)
			}
		})
	}
}

func TestVulkanSnapshotIsPremultipliedOnTransparent(t *testing.T) {
	lavapipe(t)
	scene := gputest.DamageScene(gputest.PictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(8, 8, 16, 16), gputest.FillPaint(0x80FF0000))
	}))
	jobs := []*drawJob{{name: "half red", scene: scene, width: 32, height: 32, snapshot: true}}
	drawOnWindow(t, jobs)
	snap := jobs[0].snap
	// Premultiplied: half transparent red is (128, 0, 0, 128); outside, nothing.
	if p := snap.Pix[4*(16*32+16) : 4*(16*32+16)+4]; p[0] < 126 || p[0] > 130 || p[1] != 0 || p[2] != 0 || p[3] < 126 || p[3] > 130 {
		t.Errorf("pixel in the rectangle = %v", p)
	}
	if p := snap.Pix[:4]; !bytes.Equal(p, []byte{0, 0, 0, 0}) {
		t.Errorf("pixel outside = %v", p)
	}
}

func TestVulkanToImageThroughTheGPU(t *testing.T) {
	lavapipe(t)
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4, 4, 56, 40), 10, 10), gputest.FillPaint(0xFF3050C0))
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 64}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x2000FF00)}))
	c.DrawCircle(ui.Offset{Dx: 32, Dy: 24}, 16, g)
	pic := rec.EndRecording()
	want, err := pic.ToImage(64, 48, ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}})
	if err != nil {
		t.Fatal(err)
	}
	var got *ui.Image
	var gotErr error
	jobs := []*drawJob{{name: "first frame", scene: gputest.DamageScene(), width: 64, height: 48}}
	drawOnWindowThen(t, jobs, func(r *explicit.Renderer) {
		got, gotErr = pic.ToImage(64, 48, ui.ToImageOpts{Rasterizer: r})
	})
	if gotErr != nil {
		t.Fatal(gotErr)
	}
	if got.Width() != 64 || got.Height() != 48 {
		t.Fatalf("image = %dx%d", got.Width(), got.Height())
	}
	a := &image.RGBA{Pix: got.Pixels(), Stride: 64 * 4, Rect: image.Rect(0, 0, 64, 48)}
	b := &image.RGBA{Pix: want.Pixels(), Stride: 64 * 4, Rect: image.Rect(0, 0, 64, 48)}
	gputest.Tolerance{Over8Permille: 15, Over32Permille: 3}.Check(t, "ToImage", gputest.DiffRect(a, b, a.Bounds()))
	// The image is the pixels of a transparent surface: the corner has none.
	if a.Pix[3] != 0 {
		t.Errorf("the corner has alpha %d", a.Pix[3])
	}
}

// retainedCase is a clip around a retained layer, and how far the cached frames may be from the
// frames drawn without the cache.
type retainedCase struct {
	name string
	clip func(b *ui.SceneBuilder)
	tol  gputest.Tolerance
}

func TestVulkanRasterCacheDrawsWhatTheLayerDrew(t *testing.T) {
	lavapipe(t)
	tests := []retainedCase{
		{"no clip", nil, gputest.Tolerance{}},
		{"a rectangle", func(b *ui.SceneBuilder) { b.PushClipRect(ui.NewRectFromLTWH(10, 10, 100, 90)) }, gputest.Tolerance{}},
		// The clip's edge applies to the cached group as a whole, so where content overlaps at
		// the anti-aliased edge the pixels differ a little.
		{"a rounded clip", func(b *ui.SceneBuilder) {
			b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10.5, 10.5, 100, 90), 20, 20))
		}, gputest.Tolerance{Over8Permille: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenes := gputest.RetainedFrames(5, tt.clip)
			frames := func(extra ...func(*explicit.RendererOpts)) []*drawJob {
				jobs := make([]*drawJob, len(scenes))
				for i, s := range scenes {
					jobs[i] = &drawJob{name: "frame", scene: s, width: 128, height: 128}
				}
				drawOnWindow(t, jobs, extra...)
				return jobs
			}
			plain := frames()
			cached := frames(func(o *explicit.RendererOpts) { o.RasterCacheAfter = 2 })
			var instances []int
			for i, j := range cached {
				instances = append(instances, j.stats.Instances)
				tt.tol.Check(t, "frame "+string(rune('1'+i)), gputest.DiffRect(j.got, plain[i].got, j.got.Bounds()))
				switch {
				case i == 1 && j.stats.CachedLayersMade != 1:
					t.Errorf("frame 2: %+v", j.stats)
				case i >= 2 && j.stats.CachedLayersDrawn != 1:
					t.Errorf("frame %d: %+v", i+1, j.stats)
				}
			}
			// The cached frames draw fewer things: the layer is one instance.
			if instances[2] >= instances[0] || instances[3] != instances[2] {
				t.Errorf("instances per frame = %v", instances)
			}
		})
	}
}

func TestVulkanDamageIsToldToTheCompositor(t *testing.T) {
	lavapipe(t)
	still := gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 50, 50))
	m100 := gputest.DamagePicture(ui.NewRectFromLTWH(100, 20, 30, 30))
	m120 := gputest.DamagePicture(ui.NewRectFromLTWH(120, 20, 30, 30))
	scenes := func() []*ui.Scene {
		return []*ui.Scene{
			gputest.DamageScene(still, m100),
			gputest.DamageScene(still, m100),
			gputest.DamageScene(still, m120),
			gputest.DamageScene(still, m120),
			gputest.DamageScene(still),
		}
	}
	frames := func(extra ...func(*explicit.RendererOpts)) []*drawJob {
		var jobs []*drawJob
		for _, s := range scenes() {
			jobs = append(jobs, &drawJob{name: "frame", scene: s, width: 200, height: 150})
		}
		drawOnWindow(t, jobs, extra...)
		return jobs
	}
	with := frames(func(o *explicit.RendererOpts) { o.Damage = true })
	without := frames()
	for i := range with {
		if !bytes.Equal(with[i].got.Pix, without[i].got.Pix) {
			t.Errorf("frame %d differs from the frame drawn without damage", i)
		}
		if st := without[i].stats; st.DamagedPixels != 200*150 || st.DamageRects != 0 {
			t.Errorf("frame %d: a renderer without the option reports damage %+v", i, st)
		}
	}
	// The tiles are 64 pixels: the pictures at x 99 to 151 are in the tiles of x 64 to 192, and y
	// 19 to 51 in the first row.
	whole, tiles := 200*150, 128*64
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
}

func TestVulkanDamageSwapSurvivesManyFrames(t *testing.T) {
	lavapipe(t)
	still := gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 50, 50))
	var jobs []*drawJob
	for i := 0; i < 60; i++ {
		scene := gputest.DamageScene(still, gputest.DamagePicture(ui.NewRectFromLTWH(float64(i), 20, 30, 30)))
		jobs = append(jobs, &drawJob{name: "frame", scene: scene, width: 128, height: 96})
	}
	drawOnWindow(t, jobs, func(o *explicit.RendererOpts) { o.Damage = true; o.RasterCacheAfter = 3 })
}
