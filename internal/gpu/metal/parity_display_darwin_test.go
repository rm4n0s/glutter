//go:build darwin && !ios && glutter_display

package metal

import (
	"image"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// The parity test of step 33.8, as the other explicit backends have (vulkan's parity_display_test.go, d3d12's
// parity_wine_test.go): every proof scene drawn through the explicit renderer on Metal matches the software
// renderer within the tolerance the other backends have, nothing is skipped, and a snapshot of the scene is the
// same pixels as the frame. The swap chain has its own tests, so the windows of these tests have a swap chain
// that is a texture and shows nothing: the frame is read back from the frame texture (SPEC §9.3 "Frames").

// offscreenSwapchain is a swap chain that is one BGRA texture: Acquire returns it, Present does nothing.
type offscreenSwapchain struct {
	d    *Device
	tex  explicit.Texture
	w, h int
}

func (s *offscreenSwapchain) Configure(w, h int) error {
	if s.tex != nil && s.w == w && s.h == h {
		return nil
	}
	if s.tex != nil {
		s.tex.Destroy()
		s.tex = nil
	}
	tex, err := s.d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatBGRA8Unorm, MipLevels: 1,
		Usage: explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource})
	if err != nil {
		return err
	}
	s.tex, s.w, s.h = tex, w, h
	return nil
}
func (s *offscreenSwapchain) PreRotation() explicit.Rotation         { return explicit.Rotation0 }
func (s *offscreenSwapchain) Stale() bool                            { return false }
func (s *offscreenSwapchain) Acquire() (explicit.Texture, error)     { return s.tex, nil }
func (s *offscreenSwapchain) Present(damage []image.Rectangle) error { return nil }
func (s *offscreenSwapchain) Destroy() {
	if s.tex != nil {
		s.tex.Destroy()
		s.tex = nil
	}
}

// offscreenDevice is a Device whose swap chains are offscreenSwapchains.
type offscreenDevice struct{ *Device }

func (o offscreenDevice) NewSwapchain(gpu.SurfaceHandle) (explicit.Swapchain, error) {
	return &offscreenSwapchain{d: o.Device}, nil
}

// rig is a renderer over Metal devices that it opens itself, on the thread of the test.
type rig struct {
	t    *testing.T
	r    *explicit.Renderer
	devs []*Device
}

// newRig makes the renderer; the device is opened at the first frame, as the engine does.
func newRig(t *testing.T, o Options, extra ...func(*explicit.RendererOpts)) *rig {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	g := &rig{t: t}
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	opts := explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true, SkipUnsupported: true}
	for _, f := range extra {
		f(&opts)
	}
	g.r = explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := Open(o)
		if err != nil {
			return nil, err
		}
		g.devs = append(g.devs, d)
		return offscreenDevice{d}, nil
	}, opts)
	t.Cleanup(g.r.Close)
	return g
}

// dev is the device the renderer has now.
func (g *rig) dev() *Device { return g.devs[len(g.devs)-1] }

// drawn is a scene drawn at a size, and what came of it.
type drawn struct {
	got   *image.RGBA
	snap  *image.RGBA
	stats explicit.FrameStats
}

// draw draws the scene into the window and reads the frame back, and then the snapshot.
func (g *rig) draw(scene *ui.Scene, w, h int, snapshot bool) drawn {
	g.t.Helper()
	img, err := g.r.Draw(gpu.SurfaceHandle{}, w, h, scene, true)
	if err != nil {
		g.t.Fatalf("Draw: %+v", err)
	}
	if img == nil {
		g.t.Fatal("Draw gave no image")
	}
	out := drawn{got: img, stats: g.r.LastFrame()}
	if snapshot {
		out.snap, err = g.r.Snapshot(scene, ui.Size{Width: float64(w), Height: float64(h)})
		if err != nil {
			g.t.Fatalf("Snapshot: %+v", err)
		}
	}
	return out
}

// dumpImage saves an image into the directory named by GLUTTER_TEST_DUMP, when it is set.
func dumpImage(t *testing.T, name string, img image.Image) {
	t.Helper()
	dir := os.Getenv("GLUTTER_TEST_DUMP")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Errorf("dump: %v", err)
		return
	}
	if err := png.Encode(f, img); err != nil {
		t.Errorf("dump: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("dump: %v", err)
	}
}

func slug(s string) string { return strings.NewReplacer(" ", "-", ",", "", ":", "").Replace(s) }

// cellNames are the cells of the shapes-clips scene, by grid index.
var cellNames = []string{
	"rect", "rrect", "superellipse", "oval", "circle stroke", "drrect", "arc", "lines and points",
	"vertices", "nine-patch", "image quality", "clipped image",
}

func TestEveryProofSceneMatchesSoft(t *testing.T) {
	g := newRig(t, Options{})
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) {
			g.t = t
			scene := sc.Build()
			tol, ok := gputest.ProofTolerances[name]
			if !ok {
				t.Fatalf("the proof scene %q has no tolerance in gputest.ProofTolerances", name)
			}
			j := g.draw(scene, sc.Width, sc.Height, true)
			want, err := soft.Render(scene, sc.Width, sc.Height)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "metal-parity-"+name+"-gpu", j.got)
			dumpImage(t, "metal-parity-"+name+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			if name == "shapes-clips" {
				for i, cell := range cellNames {
					r := image.Rect(i%4*160, i/4*160, i%4*160+160, i/4*160+160)
					gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, cell, gputest.DiffRect(j.got, want, r))
				}
			} else {
				tol.Check(t, name, gputest.DiffRect(j.got, want, j.got.Bounds()))
			}
			if j.snap == nil || j.snap.Bounds() != j.got.Bounds() {
				t.Fatalf("the snapshot is %v, want %v", j.snap, j.got.Bounds())
			}
			differ := 0
			for i := 0; i < len(j.snap.Pix); i += 4 {
				if j.snap.Pix[i+3] != 255 {
					continue
				}
				for k := 0; k < 4; k++ {
					if d := int(j.snap.Pix[i+k]) - int(j.got.Pix[i+k]); d < -1 || d > 1 {
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

func TestShapeScenesMatchSoft(t *testing.T) {
	g := newRig(t, Options{})
	for _, sc := range gputest.ShapeScenes() {
		t.Run(sc.Name, func(t *testing.T) {
			g.t = t
			scene := gputest.CanvasScene(sc.Draw)
			j := g.draw(scene, gputest.ShapeSize, gputest.ShapeSize, false)
			want, err := soft.Render(scene, gputest.ShapeSize, gputest.ShapeSize)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "metal-shape-"+slug(sc.Name)+"-gpu", j.got)
			dumpImage(t, "metal-shape-"+slug(sc.Name)+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance().Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}

// TestEffectScenesMatchSoft draws the scenes of the paint shaders, the layers, the backdrop filters, the blends
// that read the destination and the mask filters.
func TestEffectScenesMatchSoft(t *testing.T) {
	g := newRig(t, Options{})
	for _, sc := range gputest.EffectScenes() {
		t.Run(sc.Name, func(t *testing.T) {
			g.t = t
			scene := sc.Build()
			j := g.draw(scene, gputest.EffectSize, gputest.EffectSize, false)
			want, err := soft.Render(scene, gputest.EffectSize, gputest.EffectSize)
			if err != nil {
				t.Fatal(err)
			}
			dumpImage(t, "metal-effect-"+slug(sc.Name)+"-gpu", j.got)
			dumpImage(t, "metal-effect-"+slug(sc.Name)+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance.Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}

// TestSecondRunCompilesNoPipeline makes the renderer twice over the same shader cache directory: the first run
// compiles the pipelines of the proof scenes into the binary archive, the second takes every one from it.
func TestSecondRunCompilesNoPipeline(t *testing.T) {
	if os.Getenv("MTL_SHADER_VALIDATION") == "1" {
		t.Skip("the pipelines are instrumented under MTL_SHADER_VALIDATION=1, and the archive does not hold them")
	}
	dir := t.TempDir()
	scenes := []string{"shapes-basic", "gradients", "effects"}
	run := func() (Stats, *Device) {
		g := newRig(t, Options{ShaderCacheDir: dir})
		for _, name := range scenes {
			sc, err := proof.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			g.draw(sc.Build(), sc.Width, sc.Height, false)
		}
		return g.dev().Stats(), g.dev()
	}
	first, d1 := run()
	t.Logf("first run: %+v", first)
	if first.PipelinesCompiled == 0 || first.PipelinesFromCache != 0 {
		t.Fatalf("the first run with an empty cache: %+v", first)
	}
	if err := d1.Close(); err != nil { // saves the archive
		t.Fatal(err)
	}
	second, _ := run()
	t.Logf("second run: %+v", second)
	if second.PipelinesCompiled != 0 {
		t.Errorf("the second run compiled %d pipelines, want none: %+v", second.PipelinesCompiled, second)
	}
	if second.PipelinesFromCache != first.PipelinesCompiled {
		t.Errorf("the second run took %d pipelines from the archive, the first compiled %d", second.PipelinesFromCache, first.PipelinesCompiled)
	}
}
