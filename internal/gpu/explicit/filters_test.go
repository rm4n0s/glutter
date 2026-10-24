package explicit_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

// pipelinesOfLastFrame lists the shaders of the SetPipeline calls of the last frame.
func pipelinesOfLastFrame(dev *fakedevice.Device) []string {
	var out []string
	for _, c := range opsOfLastFrame(dev) {
		if c.Op == "SetPipeline" {
			out = append(out, strings.TrimPrefix(c.Args, "shader="))
		}
	}
	return out
}

func countOf(list []string, s string) int {
	n := 0
	for _, v := range list {
		if v == s {
			n++
		}
	}
	return n
}

func sceneWithLayer(push func(b *ui.SceneBuilder), draw func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	draw(ui.NewCanvas(rec))
	b := ui.NewSceneBuilder()
	push(b)
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	b.Pop()
	return b.Build()
}

func TestMaskFilterOnARectIsOneShadowInstance(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xFF204080)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleOuter, 4))
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 40, 30), 6, 6), p) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 || st.Instances != 1 || st.Batches != 1 {
		t.Fatalf("stats %+v", st)
	}
	if got := pipelinesOfLastFrame(dev); !reflect.DeepEqual(got, []string{"shape", "blit"}) {
		t.Errorf("pipelines %v: the closed form needs no layer", got)
	}
	in := instances(t, dev)[0]
	// The kind (shadow, 6) is float 10 of the instance, the deviation and the style 44 and 45.
	if in[10] != 6 || in[44] != 4 || in[45] != 2 {
		t.Errorf("kind %v, sigma %v, style %v", in[10], in[44], in[45])
	}
	// Half the shape in pixels, padded by three deviations: the quad is bigger than the shape.
	if in[0] > 20-12 || in[2] < 60+12 {
		t.Errorf("quad %v does not reach three deviations out", in[0:4])
	}
}

func TestMaskFilterOnOtherShapesUsesABlurredLayer(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xFF204080)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleSolid, 3))
	scene := sceneOf(func(c *ui.Canvas) { c.DrawPath(star(), p) })
	if _, err := r.Draw(window, 128, 128, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	// The path is drawn white into a layer, which is blurred in two passes along its axes and
	// composited with the paint's color.
	got := pipelinesOfLastFrame(dev)
	if want := []string{"mask", "filter", "filter", "layer", "blit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pipelines %v, want %v (a filter pass for each axis)", got, want)
	}
}

func TestMaskFilterWithNoBlurDrawsTheShape(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xFF204080)
	p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 0))
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(10, 10, 20, 20), p) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if in := instances(t, dev); len(in) != 1 || in[0][10] == 6 {
		t.Errorf("instances %v: a blur of nothing is the shape", in)
	}
}

func TestImageFilterOfASaveLayerRunsPassesOverTheLayer(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xC0000000)
	p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 3}))
	scene := sceneOf(func(c *ui.Canvas) {
		c.SaveLayer(&ui.Rect{Left: 10, Top: 10, Right: 60, Bottom: 60}, p)
		c.DrawRect(ui.NewRectFromLTWH(20, 20, 20, 20), solid(0xFFFF0000))
		c.Restore()
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	got := pipelinesOfLastFrame(dev)
	if want := []string{"shape", "filter", "filter", "layer", "blit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pipelines %v, want %v", got, want)
	}
	// The pass uniform: a blur (kind 1) along x then y, of the source's size (50 x 50), with the
	// deviations scaled by the transform (2 and 3 in pixels) and a kernel of 3 deviations.
	var params [][]float32
	for _, w := range dev.Writes {
		if w.Usage&explicit.BufferUniform == 0 {
			continue
		}
		for off := 0; off+96 <= len(w.Data); off += 256 {
			f := floatsOf(w.Data[off : off+96])
			if f[0] == 1 && f[4] == 50 && f[5] == 50 {
				params = append(params, f)
			}
		}
	}
	if len(params) != 2 {
		t.Fatalf("%d blur passes in the ring", len(params))
	}
	if params[0][1] != 0 || params[1][1] != 1 || params[0][8] != 2 || params[1][8] != 3 || params[0][9] != 6 || params[1][9] != 9 {
		t.Errorf("passes %v / %v", params[0], params[1])
	}
}

func TestImageFilterOfAPaintFiltersWhatItDrew(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xFF204080)
	p.SetImageFilter(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 0}))
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(20, 20, 30, 30), p) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if got, want := pipelinesOfLastFrame(dev), []string{"shape", "filter", "layer", "blit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pipelines %v, want %v", got, want)
	}
}

func TestBackdropFilterCopiesWhatIsUnderTheLayer(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneWithLayer(func(b *ui.SceneBuilder) {
		b.PushBackdropFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
	}, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(10, 10, 20, 20), solid(0x40FFFFFF)) })
	if _, err := r.Draw(window, 64, 64, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	// The pixels under the layer are copied, then blurred into the layer's texture, which the
	// layer's contents are drawn over (the pass keeps what the filter made), then composited.
	got := pipelinesOfLastFrame(dev)
	if want := []string{"filter", "filter", "shape", "layer", "blit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pipelines %v, want %v: two filter passes before the layer's contents", got, want)
	}
	var copies int
	var keepsBackdrop bool
	var sawFilter bool
	for _, c := range opsOfLastFrame(dev) {
		switch c.Op {
		case "CopyTexture":
			copies++
		case "SetPipeline":
			sawFilter = sawFilter || strings.Contains(c.Args, "filter")
		case "BeginPass":
			if sawFilter && strings.Contains(c.Args, "load=1") {
				keepsBackdrop = true
			}
		}
	}
	if copies != 1 || !keepsBackdrop {
		t.Errorf("%d copies, the layer keeps the filtered backdrop: %v", copies, keepsBackdrop)
	}
}

func TestShaderMaskPaintsTheShaderIntoTheLayer(t *testing.T) {
	r, dev, _ := newRenderer(t)
	g := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 40}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)})
	scene := sceneWithLayer(func(b *ui.SceneBuilder) {
		b.PushShaderMask(g, ui.NewRectFromLTWH(10, 10, 40, 40), ui.BlendModeSrcIn)
	}, func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 64, 64), solid(0xFF00FF00)) })
	if _, err := r.Draw(window, 64, 64, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	// The contents and then the gradient's rectangle are drawn into the layer, with the blend that
	// reads what is under it, so there is a copy.
	if got := pipelinesOfLastFrame(dev); countOf(got, "layer") != 1 || countOf(got, "shape") < 2 {
		t.Errorf("pipelines %v", got)
	}
}

func TestShadowIsMadeOfBlurredLayers(t *testing.T) {
	r, dev, _ := newRenderer(t)
	path := ui.NewPath()
	path.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(30, 30, 60, 40), 8, 8))
	scene := sceneOf(func(c *ui.Canvas) { c.DrawShadow(path, ui.NewColor(0xFF000000), 6, false) })
	if _, err := r.Draw(window, 128, 128, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	// An ambient and a spot shadow, each a blurred layer in the layer of the whole shadow, and
	// the occluder cut out of it.
	got := pipelinesOfLastFrame(dev)
	if countOf(got, "layer") != 3 || countOf(got, "filter") < 2 {
		t.Errorf("pipelines %v", got)
	}
}

func TestEveryFilterPipelineIsMadeOnce(t *testing.T) {
	r, dev, _ := newRenderer(t)
	p := solid(0xC0000000)
	p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
	scene := sceneOf(func(c *ui.Canvas) {
		c.SaveLayer(nil, p)
		c.DrawRect(ui.NewRectFromLTWH(20, 20, 20, 20), solid(0xFFFF0000))
		c.Restore()
		c.SaveLayer(nil, p)
		c.DrawRect(ui.NewRectFromLTWH(50, 50, 20, 20), solid(0xFF0000FF))
		c.Restore()
	})
	for i := 0; i < 3; i++ {
		if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
			t.Fatal(err)
		}
	}
	filters := 0
	for _, a := range calls(dev, "NewPipeline") {
		if strings.Contains(a, "filter") {
			filters++
		}
	}
	if filters != 1 {
		t.Errorf("%d filter pipelines made over three frames", filters)
	}
}

func TestRasterCacheKeepsRetainedLayers(t *testing.T) {
	r, dev, _ := newRenderer(t, explicit.RendererOpts{RasterCacheAfter: 2})
	var kept *ui.OffsetEngineLayer
	frame := func() *ui.Scene {
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 20, 20), solid(0xFFFF0000))
		b := ui.NewSceneBuilder()
		b.PushOffset(5, 5)
		if kept == nil {
			kept = b.PushOffset(0, 0)
			b.AddPicture(ui.Offset{}, rec.EndRecording())
			b.Pop()
		} else {
			b.AddRetained(kept)
		}
		b.Pop()
		return b.Build()
	}
	type result struct {
		made, drawn, instances int
		pipelines              []string
	}
	var results []result
	for i := 0; i < 5; i++ {
		if _, err := r.Draw(window, 100, 100, frame(), false); err != nil {
			t.Fatal(err)
		}
		st := r.LastFrame()
		results = append(results, result{st.CachedLayersMade, st.CachedLayersDrawn, st.Instances, pipelinesOfLastFrame(dev)})
	}
	// Frame 1 draws the layer as it is; frame 2 draws it into a texture that the cache keeps;
	// from frame 3 on one instance of the layer pipeline draws the texture.
	if results[0].made != 0 || results[1].made != 1 || results[1].drawn != 0 {
		t.Errorf("frames 1 and 2: %+v %+v", results[0], results[1])
	}
	for i := 2; i < 5; i++ {
		if results[i].made != 0 || results[i].drawn != 1 {
			t.Errorf("frame %d: %+v", i+1, results[i])
		}
		if want := []string{"layer", "blit"}; !reflect.DeepEqual(results[i].pipelines, want) {
			t.Errorf("frame %d pipelines %v, want %v", i+1, results[i].pipelines, want)
		}
	}
	// The layer's texture was made once and is kept, not made again for each frame.
	layers := 0
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "128x128 format=1 levels=1 usage=7") {
			layers++
		}
	}
	if layers != 1 {
		t.Errorf("%d layer textures made over five frames, want 1: %v", layers, calls(dev, "NewTexture"))
	}
}

func TestRasterCacheIsOffByDefault(t *testing.T) {
	r, _, _ := newRenderer(t)
	b := ui.NewSceneBuilder()
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 20, 20), solid(0xFFFF0000))
	pic := rec.EndRecording()
	kept := b.PushOffset(0, 0)
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	first := b.Build()
	b2 := ui.NewSceneBuilder()
	b2.AddRetained(kept)
	second := b2.Build()
	for i := 0; i < 4; i++ {
		scene := second
		if i == 0 {
			scene = first
		}
		if _, err := r.Draw(window, 64, 64, scene, false); err != nil {
			t.Fatal(err)
		}
		if st := r.LastFrame(); st.CachedLayersMade != 0 || st.CachedLayersDrawn != 0 {
			t.Errorf("frame %d used the cache: %+v", i+1, st)
		}
	}
}

func TestSnapshotDrawsWithoutAWindow(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(2, 2, 6, 6), solid(0x80FF0000)) })
	if _, err := r.Draw(window, 32, 32, scene, false); err != nil {
		t.Fatal(err)
	}
	frames := r.LastFrame()
	swapchains := len(calls(dev, "NewSwapchain"))
	img, err := r.Snapshot(scene, ui.Size{Width: 16, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 12 {
		t.Errorf("snapshot is %v", img.Bounds())
	}
	if got := len(calls(dev, "NewSwapchain")); got != swapchains {
		t.Errorf("a snapshot made a swapchain")
	}
	if !reflect.DeepEqual(r.LastFrame(), frames) {
		t.Errorf("a snapshot changed the last frame's numbers: %+v, was %+v", r.LastFrame(), frames)
	}
	// ToImage goes through the renderer as a ui.Rasterizer.
	im, err := r.RasterizeScene(scene, 16, 12)
	if err != nil || im.Width() != 16 || im.Height() != 12 {
		t.Errorf("RasterizeScene = %v, %v", im, err)
	}
}

func TestSnapshotFailures(t *testing.T) {
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 4, 4), solid(0xFF000000)) })
	tests := []struct {
		name  string
		setup func(r *explicit.Renderer)
		size  ui.Size
		tag   string
	}{
		{"before the first frame", func(*explicit.Renderer) {}, ui.Size{Width: 8, Height: 8}, "RendererNotAttached"},
		{"an empty size", func(r *explicit.Renderer) { _, _ = r.Draw(window, 8, 8, scene, false) }, ui.Size{Width: 0, Height: 8}, "RendererSizeInvalid"},
		{"bigger than the textures", func(r *explicit.Renderer) { _, _ = r.Draw(window, 8, 8, scene, false) }, ui.Size{Width: 100000, Height: 8}, "RendererSizeInvalid"},
		{"after Close", func(r *explicit.Renderer) { r.Close() }, ui.Size{Width: 8, Height: 8}, "RendererClosed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, _ := newRenderer(t)
			tt.setup(r)
			_, err := r.Snapshot(scene, tt.size)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("Renderer.Snapshot."+tt.tag) {
				t.Errorf("got %v, want %s through Renderer.Snapshot", err, tt.tag)
			}
		})
	}
}

func TestDamageIsCountedInTheFrameStats(t *testing.T) {
	r, _, _ := newRenderer(t, explicit.RendererOpts{Damage: true})
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 20, 20), solid(0xFF000000))
	pic := rec.EndRecording()
	for i, want := range []struct{ pixels, rects int }{{64 * 64, 0}, {1, 1}} {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, pic)
		if _, err := r.Draw(window, 64, 64, b.Build(), false); err != nil {
			t.Fatal(err)
		}
		if st := r.LastFrame(); st.DamagedPixels != want.pixels || st.DamageRects != want.rects {
			t.Errorf("frame %d: damage %d pixels in %d rectangles, want %d in %d", i+1, st.DamagedPixels, st.DamageRects, want.pixels, want.rects)
		}
	}
}

func TestBackdropMatrixFilterCopiesWhereItReadsFrom(t *testing.T) {
	r, dev, _ := newRenderer(t)
	b := ui.NewSceneBuilder()
	// A filter that moves the pixels down by 25: the layer at (20, 20) to (40, 40) shows the
	// pixels that are 25 above it, which are not under the layer.
	b.PushClipRect(ui.NewRectFromLTWH(20, 20, 20, 20))
	b.PushBackdropFilter(ui.NewImageFilterMatrix(ui.NewMatrix4TranslationValues(0, 25, 0)))
	b.Pop()
	b.Pop()
	if _, err := r.Draw(window, 64, 64, b.Build(), false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 {
		t.Fatalf("stats %+v", st)
	}
	var copies []string
	for _, c := range opsOfLastFrame(dev) {
		if c.Op == "CopyTexture" {
			copies = append(copies, c.Args)
		}
	}
	// The pixels it reads (19, -6) to (41, 16) are cut to the surface; the copy is of those.
	if want := []string{"rect=(19,0)-(41,16) at=(0,0)"}; !reflect.DeepEqual(copies, want) {
		t.Errorf("copies %v, want %v", copies, want)
	}
}
