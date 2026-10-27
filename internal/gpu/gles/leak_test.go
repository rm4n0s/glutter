package gles_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/ui"
)

// imageScene draws one image of 64 x 64 pixels whose pixels are new each time.
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

// TestLeaksRenderer draws frames on the GLES renderer over the fake GL (step 29.21, SPEC §15
// "Memory over time"): the live heap, the goroutines, and the GL objects that were made and not
// deleted.
func TestLeaksRenderer(t *testing.T) {
	tests := []struct {
		name   string
		warmup int
		scene  func(t testing.TB) func(i int) *ui.Scene
		w, h   int
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
			draw, live, closeRig, err := gles.NewLeakRig(gles.RendererOpts{
				Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			if err != nil {
				t.Fatal(err)
			}
			defer closeRig()
			scene := tt.scene(t)
			count := func(kind string) func() int { return func() int { return live(kind) } }
			leaktest.Run(t, "gles/"+tt.name, func() func() {
				i := 0
				return func() {
					w, h := tt.w, tt.h
					if tt.resize {
						w, h = tt.w+(i%8)*40, tt.h+(i%5)*30
					}
					if err := draw(scene(i), w, h); err != nil {
						t.Fatal(err)
					}
					i++
				}
			}, leaktest.Opts{N: 300, Warmup: tt.warmup, Counters: map[string]func() int{
				"textures": count("texture"), "buffers": count("buffer"), "framebuffers": count("framebuffer"),
				"renderbuffers": count("renderbuffer"), "programs": count("program"), "vaos": count("vao"),
			}})
		})
	}
}

// TestRendererCloseDeletesEveryObject checks that closing the renderer deletes the GL objects it
// made, after frames that filled its caches.
func TestRendererCloseDeletesEveryObject(t *testing.T) {
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
			draw, live, closeRig, err := gles.NewLeakRig(gles.RendererOpts{Damage: true, RasterCacheAfter: 3})
			if err != nil {
				t.Fatal(err)
			}
			s := scene.make(t)
			for range 6 {
				if err := draw(s, scene.w, scene.h); err != nil {
					t.Fatal(err)
				}
			}
			kinds := []string{"texture", "buffer", "framebuffer", "renderbuffer", "program", "vao", "shader"}
			total := 0
			for _, k := range kinds {
				total += live(k)
			}
			if total == 0 {
				t.Fatal("the renderer made no GL object")
			}
			closeRig()
			for _, k := range kinds {
				if n := live(k); n != 0 {
					t.Errorf("%d GL objects of kind %q alive after Close", n, k)
				}
			}
		})
	}
}
