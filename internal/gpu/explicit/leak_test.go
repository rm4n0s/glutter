package explicit_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/ui"
)

// imageScene draws one image of 64 x 64 pixels whose pixels are new each time, as an app does that
// loads a picture for each row of a list.
func imageScene(seed int) *ui.Scene {
	pix := make([]byte, 4*64*64)
	for i := range pix {
		pix[i] = byte(seed + i)
	}
	img := ui.NewImageFromRGBA(pix, 64, 64)
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawImage(img, ui.Offset{Dx: 10, Dy: 10}, ui.NewPaint())
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}

// TestLeaksRenderer draws frames on the explicit renderer over the fake device (step 29.21, SPEC
// §15 "Memory over time"): the live heap, the goroutines, and the resources that the device made
// and that were not destroyed.
func TestLeaksRenderer(t *testing.T) {
	tests := []struct {
		name   string
		warmup int
		scene  func(t testing.TB) func(i int) *ui.Scene
		w, h   int
		// resize draws the scene at sizes from w x h up, as a window that is dragged larger and smaller.
		resize bool
	}{
		{"editor caret, 50 lines", 20, func(t testing.TB) func(int) *ui.Scene {
			on, off := editorscene.CaretScenes(t, 50)
			return func(i int) *ui.Scene { return [2]*ui.Scene{on, off}[i%2] }
		}, editorscene.Width, editorscene.Height, false},
		{"editor caret, 1500 lines", 20, func(t testing.TB) func(int) *ui.Scene {
			on, off := editorscene.CaretScenes(t, 1500)
			return func(i int) *ui.Scene { return [2]*ui.Scene{on, off}[i%2] }
		}, editorscene.Width, editorscene.Height, false},
		{"scrolled list", 20, func(testing.TB) func(int) *ui.Scene {
			return func(i int) *ui.Scene { return gputest.ScrolledList(400, float64((i*37)%4000)) }
		}, gputest.ScrolledWidth, gputest.ScrolledHeight, false},
		{"new images", gpu.IdleFrames + 60, func(testing.TB) func(int) *ui.Scene {
			return func(i int) *ui.Scene { return imageScene(i) }
		}, 200, 200, false},
		{"resized window", 100, func(t testing.TB) func(int) *ui.Scene {
			on, off := editorscene.CaretScenes(t, 50)
			return func(i int) *ui.Scene { return [2]*ui.Scene{on, off}[i%2] }
		}, editorscene.Width, editorscene.Height, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit, dev.Quiet = true, true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil },
				explicit.RendererOpts{Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			defer r.Close()
			scene := tt.scene(t)
			live := func(f func(fakedevice.Live) int) func() int { return func() int { return f(dev.Live()) } }
			leaktest.Run(t, "explicit/"+tt.name, func() func() {
				i := 0
				return func() {
					w, h := tt.w, tt.h
					if tt.resize {
						w, h = tt.w+(i%8)*40, tt.h+(i%5)*30
					}
					if _, err := r.Draw(window, w, h, scene(i), false); err != nil {
						t.Fatal(err)
					}
					dev.Reset()
					i++
				}
			}, leaktest.Opts{N: 300, Warmup: tt.warmup, Counters: map[string]func() int{
				"buffers":   live(func(l fakedevice.Live) int { return l.Buffers }),
				"textures":  live(func(l fakedevice.Live) int { return l.Textures }),
				"pipelines": live(func(l fakedevice.Live) int { return l.Pipelines }),
			}})
		})
	}
}

// TestRendererCloseDestroysEveryResource checks that closing the renderer destroys what it made on
// the device, after frames that filled its caches.
func TestRendererCloseDestroysEveryResource(t *testing.T) {
	for _, scene := range []struct {
		name string
		make func(testing.TB) *ui.Scene
		w, h int
	}{
		{"editor", func(t testing.TB) *ui.Scene { on, _ := editorscene.CaretScenes(t, 100); return on }, editorscene.Width, editorscene.Height},
		{"icons", func(testing.TB) *ui.Scene { return gputest.ScrolledList(100, 300) }, gputest.ScrolledWidth, gputest.ScrolledHeight},
		{"image", func(testing.TB) *ui.Scene { return imageScene(1) }, 200, 200},
	} {
		t.Run(scene.name, func(t *testing.T) {
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit, dev.Quiet = true, true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil },
				explicit.RendererOpts{Damage: true, RasterCacheAfter: 3})
			s := scene.make(t)
			for range 6 {
				if _, err := r.Draw(window, scene.w, scene.h, s, false); err != nil {
					t.Fatal(err)
				}
			}
			if l := dev.Live(); l == (fakedevice.Live{}) {
				t.Fatalf("the renderer made no resource: %+v", l)
			}
			r.Close()
			if l := dev.Live(); l != (fakedevice.Live{}) {
				t.Errorf("resources alive after Close: %s", fmt.Sprintf("%+v", l))
			}
		})
	}
}
