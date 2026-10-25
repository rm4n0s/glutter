package gdiwindow

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/ui"
)

// recorder is a Presenter that keeps what it is given.
type recorder struct {
	frames   []*image.RGBA
	damages  []gpu.Damage
	handles  []gpu.SurfaceHandle
	released []gpu.SurfaceHandle
	fail     error
}

func (r *recorder) Present(h gpu.SurfaceHandle, img *image.RGBA, d gpu.Damage) error {
	if r.fail != nil {
		return r.fail
	}
	r.frames = append(r.frames, img)
	r.damages = append(r.damages, d)
	r.handles = append(r.handles, h)
	return nil
}

func (r *recorder) Release(h gpu.SurfaceHandle) { r.released = append(r.released, h) }

// fillScene is a scene with a rectangle of the color.
func fillScene(argb uint32, rect ui.Rect) *ui.Scene {
	return gputest.CanvasScene(func(cv *ui.Canvas) { cv.DrawRect(rect, gputest.FillPaint(argb)) })
}

func TestOverBackground(t *testing.T) {
	tests := []struct {
		name string
		in   color.RGBA // premultiplied
		bg   [3]uint8
		want color.RGBA
	}{
		{"opaque stays", color.RGBA{10, 20, 30, 255}, [3]uint8{200, 100, 50}, color.RGBA{10, 20, 30, 255}},
		{"transparent shows the background", color.RGBA{0, 0, 0, 0}, [3]uint8{200, 100, 50}, color.RGBA{200, 100, 50, 255}},
		{"half white over black", color.RGBA{128, 128, 128, 128}, [3]uint8{0, 0, 0}, color.RGBA{128, 128, 128, 255}},
		{"half black over white", color.RGBA{0, 0, 0, 128}, [3]uint8{255, 255, 255}, color.RGBA{127, 127, 127, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 2, 1))
			img.SetRGBA(0, 0, tt.in)
			img.SetRGBA(1, 0, tt.in)
			overBackground(img, tt.bg)
			for x := 0; x < 2; x++ {
				if got := img.RGBAAt(x, 0); got != tt.want {
					t.Errorf("pixel %d = %v, want %v", x, got, tt.want)
				}
			}
		})
	}
}

func TestRendererPresentsWhatSoftDraws(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec, Options{Background: ui.NewColor(0xFF102030)})
	s, err := r.Attach(win)
	if err != nil {
		t.Fatal(err)
	}
	scene := fillScene(0xFFFF0000, ui.NewRectFromLTWH(0, 0, 8, 8))
	img, err := r.RenderCapture(s, scene, ui.Size{Width: 16, Height: 16}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.frames) != 1 || rec.handles[0] != win {
		t.Fatalf("presented %d frames to %v", len(rec.frames), rec.handles)
	}
	if got := rec.frames[0].RGBAAt(2, 2); got != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("a pixel of the rectangle = %v, want red", got)
	}
	if got := rec.frames[0].RGBAAt(12, 12); got.A != 255 {
		t.Errorf("a pixel outside = %v: the frame of an opaque window must be opaque", got)
	}
	if img != rec.frames[0] {
		t.Error("RenderCapture did not return the frame it presented")
	}
	if !rec.damages[0].Full {
		t.Errorf("the first frame's damage = %+v, want Full", rec.damages[0])
	}
}

func TestRendererDamageAndInvalidate(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec)
	s, err := r.Attach(win)
	if err != nil {
		t.Fatal(err)
	}
	size := ui.Size{Width: 200, Height: 200}
	a := fillScene(0xFF00FF00, ui.NewRectFromLTWH(0, 0, 200, 200))
	b := fillScene(0xFF0000FF, ui.NewRectFromLTWH(0, 0, 200, 200))
	for i, scene := range []*ui.Scene{a, b, a} {
		if err := r.Render(s, scene, size, 1); err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
	}
	if !rec.damages[0].Full {
		t.Error("the first frame is not whole")
	}
	r.Invalidate(s)
	if err := r.Render(s, fillScene(0xFFFF00FF, ui.NewRectFromLTWH(0, 0, 10, 10)), size, 1); err != nil {
		t.Fatal(err)
	}
	if !rec.damages[len(rec.damages)-1].Full {
		t.Errorf("the frame after Invalidate has damage %+v, want Full", rec.damages[len(rec.damages)-1])
	}
	// A frame of another size is whole too.
	if err := r.Render(s, a, ui.Size{Width: 100, Height: 100}, 1); err != nil {
		t.Fatal(err)
	}
	if !rec.damages[len(rec.damages)-1].Full {
		t.Error("a new size is not a whole frame")
	}
}

func TestRendererNilSceneAndSnapshot(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec, Options{Background: ui.NewColor(0xFF010203)})
	s, _ := r.Attach(win)
	if err := r.Render(s, nil, ui.Size{Width: 4, Height: 4}, 1); err != nil {
		t.Fatal(err)
	}
	if got := rec.frames[0].RGBAAt(1, 1); got != (color.RGBA{1, 2, 3, 255}) {
		t.Errorf("a nil scene draws %v, want the background", got)
	}
	snap, err := r.Snapshot(nil, ui.Size{Width: 3, Height: 2})
	if err != nil || snap.Bounds().Dx() != 3 || snap.Bounds().Dy() != 2 {
		t.Errorf("Snapshot = %v, %v", snap, err)
	}
	if len(rec.frames) != 1 {
		t.Error("Snapshot presented a frame")
	}
}

func TestRendererFailuresHaveRoutes(t *testing.T) {
	assertRoute := func(t *testing.T, err error, want string) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if !appErr.HasRoute(want) {
			t.Errorf("route %s, want %s", appErr.Route(), want)
		}
	}
	t.Run("a surface of another kind", func(t *testing.T) {
		_, err := NewRenderer(&recorder{}).Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Window: 5})
		assertRoute(t, err, "Renderer.Attach.SurfaceKindUnsupported")
	})
	t.Run("a surface with no window", func(t *testing.T) {
		_, err := NewRenderer(&recorder{}).Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32})
		assertRoute(t, err, "Renderer.Attach.SurfaceKindUnsupported")
	})
	t.Run("after Close", func(t *testing.T) {
		r := NewRenderer(&recorder{})
		_ = r.Close()
		_, err := r.Attach(win)
		assertRoute(t, err, "Renderer.Attach.RendererClosed")
	})
	t.Run("a surface of another renderer", func(t *testing.T) {
		s, _ := NewRenderer(&recorder{}).Attach(win)
		assertRoute(t, NewRenderer(&recorder{}).Render(s, nil, ui.Size{Width: 2, Height: 2}, 1), "Renderer.Render->Renderer.RenderCapture.SurfaceInvalid")
	})
	t.Run("a destroyed surface", func(t *testing.T) {
		r := NewRenderer(&recorder{})
		s, _ := r.Attach(win)
		s.Destroy()
		s.Destroy() // twice is fine
		assertRoute(t, r.Render(s, nil, ui.Size{Width: 2, Height: 2}, 1), "Renderer.Render->Renderer.RenderCapture.SurfaceInvalid")
	})
	t.Run("two options", func(t *testing.T) {
		defer func() {
			v := recover()
			err, _ := v.(error)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != "MultipleOptions" || !appErr.HasRoute("NewRenderer.MultipleOptions") {
				t.Errorf("panic = %v", v)
			}
		}()
		NewRenderer(&recorder{}, Options{}, Options{})
	})
	t.Run("the presenter fails and the next frame is whole", func(t *testing.T) {
		rec := &recorder{fail: errors.New(TagGDIPresentFailed, "x")}
		r := NewRenderer(rec)
		s, _ := r.Attach(win)
		scene := fillScene(0xFF00FF00, ui.NewRectFromLTWH(0, 0, 100, 100))
		if _, err := r.RenderCapture(s, scene, ui.Size{Width: 100, Height: 100}, 1); err == nil {
			t.Fatal("no error")
		}
		rec.fail = nil
		_ = r.Render(s, scene, ui.Size{Width: 100, Height: 100}, 1)
		_ = r.Render(s, fillScene(0xFF0000FF, ui.NewRectFromLTWH(0, 0, 10, 10)), ui.Size{Width: 100, Height: 100}, 1)
		if !rec.damages[0].Full {
			t.Error("the frame after a failed one is not whole")
		}
	})
}

func TestSurfaceDestroyReleasesThePresenter(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec)
	s, _ := r.Attach(win)
	s.Destroy()
	s.Destroy()
	if len(rec.released) != 1 || rec.released[0] != win {
		t.Errorf("released = %v", rec.released)
	}
}
