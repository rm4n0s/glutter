package offscreen_test

import (
	"image"
	"image/color"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// imageOf wraps the pixels of a ui.Image for the comparison.
func imageOf(img *ui.Image) *image.RGBA {
	return &image.RGBA{Pix: img.Pixels(), Stride: 4 * img.Width(), Rect: image.Rect(0, 0, img.Width(), img.Height())}
}

// testScene is a scene to compare, with the size to draw it at.
type testScene struct {
	name string
	make func() *ui.Scene
	w, h int
}

// gpuScenes are the proof scenes and the scenes the GPU renderers are compared with the
// software renderer on.
func gpuScenes(t *testing.T) []testScene {
	t.Helper()
	var scenes []testScene
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			t.Fatal(err)
		}
		scenes = append(scenes, testScene{"proof " + name, sc.Build, sc.Width, sc.Height})
	}
	for _, s := range gputest.ShapeScenes() {
		scenes = append(scenes, testScene{"shapes " + s.Name, func() *ui.Scene { return gputest.CanvasScene(s.Draw) }, gputest.ShapeSize, gputest.ShapeSize})
	}
	for _, s := range gputest.EffectScenes() {
		scenes = append(scenes, testScene{"effects " + s.Name, s.Build, gputest.EffectSize, gputest.EffectSize})
	}
	return scenes
}

func TestRasterizerMatchesSoftRender(t *testing.T) {
	r := offscreen.NewRasterizer()
	for _, tt := range gpuScenes(t) {
		t.Run(tt.name, func(t *testing.T) {
			sc := tt.make()
			want, err := soft.Render(sc, tt.w, tt.h)
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.RasterizeScene(sc, tt.w, tt.h)
			if err != nil {
				t.Fatal(err)
			}
			if got.Width() != tt.w || got.Height() != tt.h {
				t.Fatalf("the image is %dx%d, want %dx%d", got.Width(), got.Height(), tt.w, tt.h)
			}
			samePixels(t, tt.name, imageOf(got), want)
		})
	}
}

// TestSceneIsReadOnlyWhileDrawn draws each scene on several goroutines at once: what the
// renderer reads of a detached scene (paths, paints, gradients, images, filters) is not changed by reading it, so
// the race detector reports nothing and the pixels agree.
func TestSceneIsReadOnlyWhileDrawn(t *testing.T) {
	r := offscreen.NewRasterizer()
	for _, tt := range gpuScenes(t) {
		t.Run(tt.name, func(t *testing.T) {
			live := tt.make()
			want, err := soft.Render(live, tt.w, tt.h)
			if err != nil {
				t.Fatal(err)
			}
			// The paragraphs of a scene are drawn through the text engine, which is not
			// safe for two goroutines: draw the detached scene.
			sc := offscreen.NewDetacher().Detach(live)
			var wg sync.WaitGroup
			results := make([]*ui.Image, 3)
			errs := make([]error, len(results))
			for i := range results {
				wg.Add(1)
				go func() {
					defer wg.Done()
					results[i], errs[i] = r.RasterizeScene(sc, tt.w, tt.h)
				}()
			}
			wg.Wait()
			for i, img := range results {
				if errs[i] != nil {
					t.Fatal(errs[i])
				}
				samePixels(t, "goroutine", imageOf(img), want)
			}
		})
	}
}

func TestRasterizerMatchesScreenshotOfAWidgetTree(t *testing.T) {
	tests := []struct {
		name string
		body func() widgets.Widget
	}{
		{"scaffold with text", textBody},
		{"scaffold without text", plainBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, scene := pumpFrame(t, scaffoldApp(tt.body()))
			got, err := scene.ToImage(frameW, frameH, ui.ToImageOpts{Rasterizer: offscreen.NewRasterizer()})
			if err != nil {
				t.Fatal(err)
			}
			samePixels(t, tt.name, imageOf(got), ts.Screenshot())
		})
	}
}

func TestRasterizerToImageOfAPicture(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFFFF0000))
	c.DrawRect(ui.NewRectFromLTWH(1, 1, 2, 2), p)
	img, err := rec.EndRecording().ToImage(4, 4, ui.ToImageOpts{Rasterizer: offscreen.NewRasterizer()})
	if err != nil {
		t.Fatal(err)
	}
	px := imageOf(img)
	if got := px.RGBAAt(1, 1); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("pixel 1,1 = %v, want opaque red", got)
	}
	if got := px.RGBAAt(0, 0); got.A != 0 {
		t.Errorf("pixel 0,0 = %v, want transparent", got)
	}
}

func TestRasterizerErrors(t *testing.T) {
	scene := gputest.CanvasScene(func(c *ui.Canvas) {})
	tests := []struct {
		name  string
		scene *ui.Scene
		w, h  int
		tag   string
		route string
	}{
		{"nil scene", nil, 4, 4, "SoftSceneNil", "Rasterizer.RasterizeScene->Render.SoftSceneNil"},
		{"zero size", scene, 0, 4, "SoftSizeInvalid", "Rasterizer.RasterizeScene->Render.SoftSizeInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := offscreen.NewRasterizer().RasterizeScene(tt.scene, tt.w, tt.h)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) || img != nil {
				t.Errorf("got %v, %v; want %s via %s", img, err, tt.tag, tt.route)
			}
		})
	}
}

// TestRasterizerRunsOnManyGoroutines draws one scene on several goroutines with one
// rasterizer: the race detector reports nothing and the pixels agree.
func TestRasterizerRunsOnManyGoroutines(t *testing.T) {
	_, live := pumpFrame(t, scaffoldApp(textBody()))
	scene := offscreen.NewDetacher().Detach(live)
	want, err := soft.Render(scene, frameW, frameH)
	if err != nil {
		t.Fatal(err)
	}
	r := offscreen.NewRasterizer()
	var wg sync.WaitGroup
	results := make([]*ui.Image, 4)
	errs := make([]error, len(results))
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = r.RasterizeScene(scene, frameW, frameH)
		}()
	}
	wg.Wait()
	for i, img := range results {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		samePixels(t, "goroutine", imageOf(img), want)
	}
}
