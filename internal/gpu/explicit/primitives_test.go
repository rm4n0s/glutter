package explicit_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

func star() *ui.Path {
	p := ui.NewPath()
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		x, y := 50+40*math.Cos(a), 50+40*math.Sin(a)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

// calls returns the calls of the device with the op.
func calls(dev *fakedevice.Device, op string) []string {
	var out []string
	for _, c := range dev.Calls() {
		if c.Op == op {
			out = append(out, c.Args)
		}
	}
	return out
}

// pipelinesOfScenePass returns the shaders of the pipelines the scene pass set.
func pipelinesOfScenePass(dev *fakedevice.Device) []string {
	var out []string
	for _, c := range scenePass(dev) {
		if c.Op == "SetPipeline" {
			out = append(out, strings.TrimPrefix(c.Args, "shader="))
		}
	}
	return out
}

func TestPathsAreDrawnFromTheAtlasAndCached(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) { c.DrawPath(star(), solid(0xFF6750A4)) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"mask"}) {
		t.Errorf("pipelines %v, want the mask pipeline", got)
	}
	first := r.LastFrame()
	if first.TilesRasterized < 1 || first.Instances < 1 {
		t.Errorf("stats %+v, want tiles rasterized", first)
	}
	// One atlas page of 2048 x 2048 8-bit pixels, and the tile written into it.
	var pages int
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "2048x2048 format=3 levels=1 usage=1") {
			pages++
		}
	}
	if pages != 1 {
		t.Errorf("%d atlas pages made, want 1: %v", pages, calls(dev, "NewTexture"))
	}
	written := len(calls(dev, "WriteTexture"))
	if written < 1 {
		t.Error("the tile was not uploaded")
	}
	// The same path again, and in another color, comes from the atlas.
	other := sceneOf(func(c *ui.Canvas) { c.DrawPath(star(), solid(0xFF00A000)) })
	for _, sc := range []*ui.Scene{scene, other} {
		if _, err := r.Draw(window, 100, 100, sc, false); err != nil {
			t.Fatal(err)
		}
		if st := r.LastFrame(); st.TilesRasterized != 0 || st.Instances != first.Instances {
			t.Errorf("the path was rasterized again: %+v", st)
		}
	}
	if got := len(calls(dev, "WriteTexture")); got != written {
		t.Errorf("a cached tile was uploaded again: %d writes, was %d", got, written)
	}
}

func TestTextIsDrawnThroughTheAtlas(t *testing.T) {
	r, dev, _ := newRenderer(t)
	// A paragraph is expanded into the paths of its glyphs by the flattening; a path stands for
	// one here, drawn at the places of two glyphs.
	scene := sceneOf(func(c *ui.Canvas) {
		for _, x := range []float64{0, 60} {
			c.Save()
			c.Translate(x, 0)
			c.DrawPath(star(), solid(0xFF000000))
			c.Restore()
		}
	})
	if _, err := r.Draw(window, 128, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.TilesRasterized != 1 || st.Instances != 2 {
		t.Errorf("stats %+v, want the one tile drawn twice", st)
	}
	if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"mask"}) {
		t.Errorf("pipelines %v", got)
	}
}

// uniformFloats decodes n floats of the Draw uniform of the call of the last frame: the last
// write into a uniform buffer, whose first entry is the frame's and then one for each call.
func uniformFloats(t *testing.T, dev *fakedevice.Device, call, n int) []float32 {
	t.Helper()
	for i := len(dev.Writes) - 1; i >= 0; i-- {
		w := dev.Writes[i]
		if w.Usage&explicit.BufferUniform == 0 {
			continue
		}
		off := 256 * (call + 1)
		if len(w.Data) < off+4*n {
			t.Fatalf("the ring has %d bytes, want an entry at %d", len(w.Data), off)
		}
		return floatsOf(w.Data[off : off+4*n])
	}
	t.Fatal("no uniform data was written")
	return nil
}

func TestShapeClipsMakeMasks(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 20, 60, 40), 12, 12))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 100), solid(0xFF204080))
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.ClipMasksBuilt != 1 || st.Skipped != 0 || st.Drawn != 1 {
		t.Errorf("stats %+v, want one mask built and the rect drawn", st)
	}
	// The mask is an 8-bit texture over the clip's bounds, and the draw's uniform says where.
	var mask int
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "60x40 format=3 levels=1 usage=1") {
			mask++
		}
	}
	if mask != 1 {
		t.Errorf("%d mask textures, want 1: %v", mask, calls(dev, "NewTexture"))
	}
	if got := uniformFloats(t, dev, 0, 4); !reflect.DeepEqual(got, []float32{10, 20, 60, 40}) {
		t.Errorf("clip mask rectangle %v, want 10 20 60 40", got)
	}
	// The same clip again reuses the mask.
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.ClipMasksBuilt != 0 {
		t.Errorf("the clip mask was built again: %+v", st)
	}
	// A clip that leaves nothing draws nothing.
	empty := sceneOf(func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(50, 50, 20, 20), 4, 4))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 100), solid(0xFF000000))
	})
	if _, err := r.Draw(window, 100, 100, empty, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Instances != 0 || st.Skipped != 0 {
		t.Errorf("an empty clip drew %+v", st)
	}
}

func floatsOf(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(uint32(b[4*i]) | uint32(b[4*i+1])<<8 | uint32(b[4*i+2])<<16 | uint32(b[4*i+3])<<24)
	}
	return out
}

func TestImagesAreUploadedOnceWithMipmaps(t *testing.T) {
	r, dev, _ := newRenderer(t)
	img := testImage(32)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(10, 10, 16, 16), nil)
		c.DrawImage(img, ui.Offset{Dx: 50, Dy: 10}, nil)
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	// 32 x 32 has six levels, drawn into by the downsampling pass: sampled and a render target.
	var textures []string
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "32x32 ") {
			textures = append(textures, a)
		}
	}
	if !reflect.DeepEqual(textures, []string{"32x32 format=1 levels=6 usage=3"}) {
		t.Errorf("image textures %v", textures)
	}
	if st := r.LastFrame(); st.ImagesUploaded != 1 || st.Instances != 2 {
		t.Errorf("stats %+v, want one upload and two rectangles", st)
	}
	// Five passes of the downsampling, then the scene's and the blit's.
	var passes []string
	for _, c := range dev.Calls() {
		if c.Op == "BeginPass" {
			passes = append(passes, c.Args)
		}
	}
	if len(passes) != 5+2 {
		t.Errorf("%d passes, want 5 for the levels, the scene and the blit: %v", len(passes), passes)
	}
	if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"image"}) {
		t.Errorf("scene pipelines %v", got)
	}
	// The second frame uploads nothing and makes no levels.
	before := len(calls(dev, "BeginPass"))
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.ImagesUploaded != 0 {
		t.Errorf("the image was uploaded again: %+v", st)
	}
	if got := len(calls(dev, "BeginPass")) - before; got != 2 {
		t.Errorf("the second frame has %d passes, want the scene and the blit", got)
	}
}

func TestImagesOfEveryKind(t *testing.T) {
	img := testImage(24)
	tests := []struct {
		name      string
		draw      func(c *ui.Canvas)
		instances int
	}{
		{"an image", func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{Dx: 5, Dy: 5}, nil) }, 1},
		{"a rectangle of an image", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(4, 4, 8, 8), ui.NewRectFromLTWH(0, 0, 40, 40), nil)
		}, 1},
		{"a rectangle outside the image is cut", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(20, 20, 20, 20), ui.NewRectFromLTWH(0, 0, 40, 40), nil)
		}, 1},
		{"a rectangle with nothing of the image", func(c *ui.Canvas) {
			c.DrawImageRect(img, ui.NewRectFromLTWH(100, 100, 5, 5), ui.NewRectFromLTWH(0, 0, 40, 40), nil)
		}, 0},
		{"a nine-patch", func(c *ui.Canvas) {
			c.DrawImageNine(img, ui.NewRectFromLTWH(8, 8, 8, 8), ui.NewRectFromLTWH(0, 0, 60, 60), nil)
		}, 9},
		{"sprites", func(c *ui.Canvas) {
			c.DrawAtlas(img, []ui.RSTransform{ui.NewRSTransformFromComponents(0, 1, 0, 0, 5, 5), ui.NewRSTransformFromComponents(0.5, 0.5, 0, 0, 50, 5)},
				[]ui.Rect{ui.NewRectFromLTWH(0, 0, 8, 8), ui.NewRectFromLTWH(8, 8, 8, 8)}, nil, ui.BlendModeSrc, nil, nil)
		}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, dev, _ := newRenderer(t)
			if _, err := r.Draw(window, 100, 100, sceneOf(tt.draw), false); err != nil {
				t.Fatal(err)
			}
			if st := r.LastFrame(); st.Instances != tt.instances || st.Skipped != 0 {
				t.Errorf("stats %+v, want %d instances", st, tt.instances)
			}
			if tt.instances > 0 {
				if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"image"}) {
					t.Errorf("pipelines %v", got)
				}
			}
		})
	}
}

func TestImageErrors(t *testing.T) {
	t.Run("a disposed image", func(t *testing.T) {
		img := testImage(8)
		scene := sceneOf(func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{}, nil) })
		img.Dispose()
		r, _, _ := newRenderer(t)
		_, err := r.Draw(window, 50, 50, scene, false)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GPUImageDisposed" || !appErr.HasRoute("Renderer.build->FlattenFunc") {
			t.Errorf("got %v, want GPUImageDisposed through Renderer.build", err)
		}
	})
	t.Run("an image larger than the textures", func(t *testing.T) {
		scene := sceneOf(func(c *ui.Canvas) { c.DrawImage(testImage(64), ui.Offset{}, nil) })
		strict, dev, _ := newRenderer(t)
		limits := dev.Limits()
		limits.MaxTextureSize = 32
		dev.SetLimits(limits)
		_, err := strict.Draw(window, 32, 32, scene, false)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "GPUImageTooLarge" || !appErr.HasRoute("Renderer.build->FlattenFunc") {
			t.Fatalf("got %v, want GPUImageTooLarge through Renderer.build", err)
		}
		skipping, dev2, _ := newRenderer(t, explicit.RendererOpts{SkipUnsupported: true})
		dev2.SetLimits(limits)
		if _, err := skipping.Draw(window, 32, 32, scene, false); err != nil {
			t.Fatal(err)
		}
		if st := skipping.LastFrame(); !reflect.DeepEqual(st.Unsupported, []string{"image larger than the textures of the device"}) {
			t.Errorf("stats %+v", st)
		}
	})
}

func TestVerticesAreDrawnAsTheyAre(t *testing.T) {
	r, dev, _ := newRenderer(t)
	colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0xFFFFFFFF)}
	v := ui.NewVertices(ui.VertexModeTriangleFan, []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 50, Dy: 10}, {Dx: 50, Dy: 50}, {Dx: 10, Dy: 50}}, ui.VerticesOpts{Colors: colors})
	scene := sceneOf(func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeDst, solid(0xFF000000)) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	// A fan of four vertices is two triangles: six vertices, drawn once.
	if st := r.LastFrame(); st.Vertices != 6 || st.Skipped != 0 {
		t.Errorf("stats %+v, want 6 vertices", st)
	}
	var draws []string
	for _, c := range scenePass(dev) {
		if c.Op == "Draw" {
			draws = append(draws, c.Args)
		}
	}
	if len(draws) != 1 || !strings.HasPrefix(draws[0], "vertices=6 instances=1 ") {
		t.Errorf("draws %v", draws)
	}
	if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"vertices"}) {
		t.Errorf("pipelines %v", got)
	}
	// The draw uniform holds the clip mask (none), the paint, the clip and the blend index of dst.
	if got := uniformFloats(t, dev, 0, 16); got[4] != 0 || got[7] != 1 || got[8] != 0 || got[10] != 100 || got[12] != 2 || got[13] != 1 {
		t.Errorf("draw uniform %v", got)
	}
}

func TestEveryPipelineOfAFrameIsMadeOnce(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
		c.DrawPath(star(), solid(0xFF000000))
		c.DrawImage(testImage(8), ui.Offset{Dx: 50}, nil)
		c.DrawRect(ui.NewRectFromLTWH(20, 0, 10, 10), solid(0xFF000000))
	})
	for i := 0; i < 3; i++ {
		if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
			t.Fatal(err)
		}
	}
	made := map[string]int{}
	for _, a := range calls(dev, "NewPipeline") {
		made[strings.Fields(a)[0]]++
	}
	want := map[string]int{"shader=blit": 2, "shader=shape": 1, "shader=mask": 1, "shader=image": 1}
	// The blit is made for the swapchain's format and for the mip levels.
	if !reflect.DeepEqual(made, want) {
		t.Errorf("pipelines made %v, want %v", made, want)
	}
	// The calls alternate between pipelines and every frame has the same draws.
	if got := pipelinesOfScenePass(dev); !reflect.DeepEqual(got, []string{"shape", "mask", "image", "shape"}) {
		t.Errorf("scene pipelines %v", got)
	}
	_ = gpu.ShapeFloats
}
