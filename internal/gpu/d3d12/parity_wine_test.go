//go:build windows && glutter_wine

package d3d12

import (
	"bytes"
	"image"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/ui"
)

// The parity test of step 28.9, as vulkan's (parity_display_test.go): every proof scene drawn through
// the explicit renderer on Direct3D 12 (vkd3d over lavapipe under Wine) matches the software renderer
// within the tolerance the other backends have, nothing is skipped, and a snapshot of the scene is the
// same pixels as the frame. The swap chain is step 28.10's, so the windows of these tests have a
// swap chain of their own that is a texture and shows nothing: the frame is read back from the frame
// texture, which does not depend on the swap chain (SPEC §9.3 "Frames").

// offscreenSwapchain is a swap chain that is one BGRA texture: Acquire returns it, Present does
// nothing. It stands in for step 28.10's swap chain, so that the renderer can draw whole frames.
type offscreenSwapchain struct {
	d        *Device
	tex      explicit.Texture
	presents int
	damage   [][]image.Rectangle
	w, h     int
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
func (s *offscreenSwapchain) PreRotation() explicit.Rotation { return explicit.Rotation0 }
func (s *offscreenSwapchain) Stale() bool                    { return false }
func (s *offscreenSwapchain) Acquire() (explicit.Texture, error) {
	return s.tex, nil
}
func (s *offscreenSwapchain) Present(damage []image.Rectangle) error {
	s.presents++
	s.damage = append(s.damage, damage)
	return nil
}
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

// rig is a renderer over Direct3D 12 devices that it opens itself, on the thread of the test.
type rig struct {
	t    *testing.T
	r    *explicit.Renderer
	devs []*Device
	rec  *logRecorder
}

// newRig makes the renderer; the device is opened at the first frame (as the engine does), from the
// options. The test keeps its OS thread from here to the end (SPEC §9.8: a device is used from the
// thread that made it).
func newRig(t *testing.T, o Options, extra ...func(*explicit.RendererOpts)) *rig {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	g := &rig{t: t, rec: &logRecorder{}}
	if o.Logger == nil {
		o.Logger = slog.New(g.rec)
	}
	if o.Getenv == nil {
		o.Getenv = env(nil)
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
	took  time.Duration
}

// draw draws the scene into the window and reads the frame back, and then the snapshot.
func (g *rig) draw(scene *ui.Scene, w, h int, snapshot bool) drawn {
	g.t.Helper()
	start := time.Now()
	img, err := g.r.Draw(gpu.SurfaceHandle{}, w, h, scene, true)
	if err != nil {
		g.t.Fatalf("Draw: %+v", err)
	}
	out := drawn{got: img, stats: g.r.LastFrame(), took: time.Since(start)}
	if img == nil {
		g.t.Fatal("Draw gave no image")
	}
	if snapshot {
		out.snap, err = g.r.Snapshot(scene, ui.Size{Width: float64(w), Height: float64(h)})
		if err != nil {
			g.t.Fatalf("Snapshot: %+v", err)
		}
	}
	return out
}

// dumpImage saves an image into the directory named by GLUTTER_TEST_DUMP, when it is set, so that a
// failing comparison can be looked at.
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

// slug makes a file name of a scene name.
func slug(s string) string {
	return strings.NewReplacer(" ", "-", ",", "", ":", "").Replace(s)
}

// cellNames are the cells of the shapes-clips scene, by grid index.
var cellNames = []string{
	"rect", "rrect", "superellipse", "oval", "circle stroke", "drrect", "arc", "lines and points",
	"vertices", "nine-patch", "image quality", "clipped image",
}

func TestWineEveryProofSceneMatchesSoft(t *testing.T) {
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
			dumpImage(t, "d3d12-parity-"+name+"-gpu", j.got)
			dumpImage(t, "d3d12-parity-"+name+"-soft", want)
			st := g.dev().PipelineStats()
			t.Logf("first frame %v; frame %+v; pipelines so far: %+v, shaders: %+v", j.took, j.stats, st, g.dev().CompilerStats())
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			if name == "shapes-clips" {
				// Compared by cell as before: the cells hold the shapes that need the most.
				for i, cell := range cellNames {
					r := image.Rect(i%4*160, i/4*160, i%4*160+160, i/4*160+160)
					gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, cell, gputest.DiffRect(j.got, want, r))
				}
			} else {
				tol.Check(t, name, gputest.DiffRect(j.got, want, j.got.Bounds()))
			}
			// A snapshot is the same pixels as the frame, drawn without a window over a clear
			// background; it is compared where the snapshot is opaque.
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

func TestWineShapeScenesMatchSoft(t *testing.T) {
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
			dumpImage(t, "d3d12-shape-"+slug(sc.Name)+"-gpu", j.got)
			dumpImage(t, "d3d12-shape-"+slug(sc.Name)+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance().Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}

// TestWineEffectScenesMatchSoft draws the scenes of the paint shaders, the layers (more than one, nested),
// the backdrop filters, the blends that read the destination and the mask filters.
func TestWineEffectScenesMatchSoft(t *testing.T) {
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
			dumpImage(t, "d3d12-effect-"+slug(sc.Name)+"-gpu", j.got)
			dumpImage(t, "d3d12-effect-"+slug(sc.Name)+"-soft", want)
			if j.stats.Skipped != 0 {
				t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
			}
			sc.Tolerance.Check(t, sc.Name, gputest.DiffRect(j.got, want, j.got.Bounds()))
		})
	}
}

// TestWineSecondRunCompilesNothing makes the renderer twice over the same shader cache: the first
// compiles the shaders of the proof scenes, the second finds every one in the cache and does not call
// D3DCompile, and the pipeline library, where the device has one, makes no pipeline.
func TestWineSecondRunCompilesNothing(t *testing.T) {
	dir := t.TempDir()
	cache := shadercache.New(dir, func(err error) { t.Errorf("the shader cache: %v", err) })
	scenes := []string{"shapes-basic", "gradients", "effects"}
	run := func() (CompilerStats, PipelineStats, time.Duration, *Device) {
		g := newRig(t, Options{ShaderCache: cache})
		start := time.Now()
		for _, name := range scenes {
			sc, err := proof.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			g.draw(sc.Build(), sc.Width, sc.Height, false)
		}
		d := g.dev()
		return d.CompilerStats(), d.PipelineStats(), time.Since(start), d
	}
	first, firstPipes, firstTook, d1 := run()
	t.Logf("first run: %v; shaders %+v; pipelines %+v; library: %v", firstTook, first, firstPipes, !d1.library.IsNil())
	if first.Compiled == 0 || first.FromCache != 0 {
		t.Fatalf("the first run with an empty cache: %+v", first)
	}
	if err := d1.Close(); err != nil { // saves the pipeline library
		t.Fatal(err)
	}
	second, secondPipes, secondTook, d2 := run()
	t.Logf("second run: %v; shaders %+v; pipelines %+v; library: %v (%d bytes loaded)", secondTook, second, secondPipes, !d2.library.IsNil(), d2.library.loaded)
	if second.Compiled != 0 {
		t.Errorf("the second run compiled %d shaders, want none: %+v", second.Compiled, second)
	}
	if second.FromCache != first.Compiled {
		t.Errorf("the second run took %d shaders from the cache, the first compiled %d", second.FromCache, first.Compiled)
	}
	if d2.library.IsNil() {
		t.Logf("this device has no pipeline library (vkd3d at this version): the pipelines are made again at each start, %d here", secondPipes.Created)
	} else if secondPipes.Created != 0 || secondPipes.FromLibrary == 0 {
		t.Errorf("a pipeline library is there and the second run made %d pipelines and loaded %d", secondPipes.Created, secondPipes.FromLibrary)
	}
}

// lostRig is a rig with a clock that the test moves.
func lostRig(t *testing.T, now *time.Time) *rig {
	return newRig(t, Options{}, func(o *explicit.RendererOpts) { o.Now = func() time.Time { return *now } })
}

// lose makes the current device report that it was removed.
func (g *rig) lose() {
	g.dev().removed = func() win32.HRESULT { return win32.DXGIErrorDeviceRemoved }
}

func TestWineDeviceLostIsRecoveredOnceAndASecondLossEnds(t *testing.T) {
	now := time.Unix(1000, 0)
	g := lostRig(t, &now)
	scene := gputest.DamageScene(gputest.DamagePicture(ui.NewRectFromLTWH(2, 2, 20, 20)))
	before := g.draw(scene, 64, 64, false)
	if len(g.devs) != 1 {
		t.Fatalf("%d devices", len(g.devs))
	}
	// The first loss: a new device is opened, the old one closed, and the frame is drawn again.
	now = now.Add(time.Minute)
	g.lose()
	first := g.dev()
	j := g.draw(scene, 64, 64, false)
	if len(g.devs) != 2 || !first.closed || g.dev().closed {
		t.Fatalf("%d devices; first closed %v", len(g.devs), first.closed)
	}
	if st := j.stats; st.Skipped != 0 || st.Instances == 0 {
		t.Errorf("the frame is drawn whole on the new device: %+v", st)
	}
	if !bytes.Equal(j.got.Pix, before.got.Pix) {
		t.Error("the frame on the new device differs from the frame on the lost one")
	}
	// A second loss within ten seconds ends the app with the error.
	now = now.Add(9 * time.Second)
	g.lose()
	_, err := g.r.Draw(gpu.SurfaceHandle{}, 64, 64, scene, true)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagGPUDeviceLost {
		t.Fatalf("a second loss: got %v, want GPUDeviceLost", err)
	}
	if !appErr.HasRoute("Device.Begin.GPUDeviceLost") && !appErr.HasRoute("Device.checkRemoved.GPUDeviceLost") && !appErr.HasRoute(".GPUDeviceLost") {
		t.Errorf("route %s", appErr.Route())
	}
	if len(g.devs) != 2 {
		t.Errorf("a second loss within ten seconds opened a device: %d", len(g.devs))
	}
	if v := appErr.Metadata["reason"]; v == nil {
		t.Errorf("the loss has no reason: %v", appErr.Metadata)
	}
}

func TestWineDeviceLostAfterTenSecondsIsRecoveredAgain(t *testing.T) {
	now := time.Unix(1000, 0)
	g := lostRig(t, &now)
	scene := gputest.DamageScene(gputest.DamagePicture(ui.NewRectFromLTWH(2, 2, 20, 20)))
	g.draw(scene, 48, 48, false)
	for i := 0; i < 3; i++ {
		now = now.Add(11 * time.Second)
		g.lose()
		g.draw(scene, 48, 48, false)
	}
	if len(g.devs) != 4 {
		t.Errorf("%d devices for three losses spaced over ten seconds", len(g.devs))
	}
	// Everything the lost devices made is released: the only live resources are the new device's.
	for i, d := range g.devs[:3] {
		if !d.closed {
			t.Errorf("device %d was not closed", i)
		}
	}
}

// TestWineNoLeaksAcross200Frames draws a scene of layers, images and blends 200 times and compares the
// resource counts of the device after the warm-up with those at the end.
func TestWineNoLeaksAcross200Frames(t *testing.T) {
	g := newRig(t, Options{})
	sc, err := proof.Lookup("effects")
	if err != nil {
		t.Fatal(err)
	}
	scene := sc.Build()
	for i := 0; i < 5; i++ {
		g.draw(scene, sc.Width, sc.Height, false)
	}
	d := g.dev()
	before := d.Stats()
	for i := 0; i < 200; i++ {
		g.draw(scene, sc.Width, sc.Height, false)
	}
	after := d.Stats()
	t.Logf("before %+v\nafter  %+v", before, after)
	same := func(what string, a, b int) {
		if a != b {
			t.Errorf("%s: %d before, %d after 200 frames", what, a, b)
		}
	}
	same("COM objects", before.Objects, after.Objects)
	same("buffers", before.Buffers, after.Buffers)
	same("textures", before.Textures, after.Textures)
	same("samplers", before.Samplers, after.Samplers)
	same("pipelines", before.Pipelines, after.Pipelines)
	same("root signatures", before.RootSignatures, after.RootSignatures)
	same("CPU descriptors", before.Descriptors, after.Descriptors)
	same("view blocks", before.ViewBlocks, after.ViewBlocks)
	same("sampler blocks", before.SamplerBlocks, after.SamplerBlocks)
	if before.CommittedBytes != after.CommittedBytes {
		t.Errorf("committed bytes: %d before, %d after", before.CommittedBytes, after.CommittedBytes)
	}
	g.r.Close()
	for i, dev := range g.devs {
		if dev.Leaks() != 0 {
			t.Errorf("device %d left %d objects after Close", i, dev.Leaks())
		}
	}
}

// TestWineLayersBackdropReadingBlendAndMaskFilterInOneScene draws what SPEC §9.3 builds on layers in one
// scene: an opacity layer that holds a layer with a backdrop filter, a draw whose blend mode reads the
// destination (color dodge, which the blend unit cannot make), and a rectangle with a mask filter (a
// blur), over a striped background; it matches the software renderer.
func TestWineLayersBackdropReadingBlendAndMaskFilterInOneScene(t *testing.T) {
	g := newRig(t, Options{})
	back := gputest.PictureOf(func(c *ui.Canvas) { gputest.StripesAndDisc(c) })
	dodge := gputest.PictureOf(func(c *ui.Canvas) {
		p := gputest.FillPaint(0xFF7090B0)
		p.SetBlendMode(ui.BlendModeColorDodge)
		c.DrawRect(ui.NewRectFromLTWH(8, 8, 60, 50), p)
		m := gputest.FillPaint(0xFF204080)
		m.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 4))
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(40, 44, 60, 50), 10, 10), m)
	})
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, back)
	b.PushOpacity(200)
	b.PushOffset(6, 6)
	b.PushBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 3, SigmaY: 3}))
	b.AddPicture(ui.Offset{}, dodge)
	b.Pop()
	b.Pop()
	b.Pop()
	scene := b.Build()
	j := g.draw(scene, gputest.EffectSize, gputest.EffectSize, true)
	want, err := soft.Render(scene, gputest.EffectSize, gputest.EffectSize)
	if err != nil {
		t.Fatal(err)
	}
	dumpImage(t, "d3d12-combined-gpu", j.got)
	dumpImage(t, "d3d12-combined-soft", want)
	if j.stats.Skipped != 0 {
		t.Errorf("the GPU skipped %d commands: %v", j.stats.Skipped, j.stats.Unsupported)
	}
	gputest.Tolerance{Over8Permille: 15, Over32Permille: 5}.Check(t, "layers, backdrop, dodge and mask filter", gputest.DiffRect(j.got, want, j.got.Bounds()))
}

// TestWineSmallHeapBlocksDrawTheSamePixels draws a scene whose bind groups do not fit one block of the
// shader-visible heaps (a block of 24 view descriptors for the 30 that the scene's tables take): the
// encoder copies the groups that are bound into the latest block and sets the heaps again. The pixels are
// those of a device whose blocks are big, in every one of 20 frames, and the blocks that the two frame
// slots made after their first frame are all there are: later frames reuse them. (A heap of vkd3d takes
// about 120 MB whatever its size, so this test keeps the number of heaps low, and the big device is closed
// before the small one is made.)
func TestWineSmallHeapBlocksDrawTheSamePixels(t *testing.T) {
	sc, err := proof.Lookup("effects")
	if err != nil {
		t.Fatal(err)
	}
	scene := sc.Build()
	big := newRig(t, Options{})
	want := big.draw(scene, sc.Width, sc.Height, false).got
	big.r.Close()

	small := newRig(t, Options{viewBlock: 24, samplerBlock: 2})
	var viewBlocks, samplerBlocks int
	for i := 0; i < 20; i++ {
		j := small.draw(scene, sc.Width, sc.Height, false)
		if !bytes.Equal(j.got.Pix, want.Pix) {
			t.Fatalf("frame %d with small heap blocks differs from the frame with big blocks", i)
		}
		if j.stats.Skipped != 0 {
			t.Fatalf("frame %d skipped %v", i, j.stats.Unsupported)
		}
		st := small.dev().Stats()
		if i == 1 { // both frame slots have drawn once
			viewBlocks, samplerBlocks = st.ViewBlocks, st.SamplerBlocks
			t.Logf("after the first frame of each slot: %d view blocks and %d sampler blocks", viewBlocks, samplerBlocks)
			if viewBlocks <= framesInFlight || viewBlocks > framesInFlight*maxHeapBlocks {
				t.Fatalf("%d view blocks: the blocks of 24 should have been added to, and capped", viewBlocks)
			}
		}
		if i > 1 && (st.ViewBlocks != viewBlocks || st.SamplerBlocks != samplerBlocks) {
			t.Fatalf("frame %d: %d view blocks and %d sampler blocks, after frame 1 there were %d and %d", i, st.ViewBlocks, st.SamplerBlocks, viewBlocks, samplerBlocks)
		}
	}
}
