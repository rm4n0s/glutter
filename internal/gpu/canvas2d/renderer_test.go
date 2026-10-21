package canvas2d

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// recorder is a Presenter that keeps the frames it is given.
type recorder struct {
	frames   []*image.RGBA
	handles  []gpu.SurfaceHandle
	released []gpu.SurfaceHandle
	fail     error
}

func (r *recorder) Present(h gpu.SurfaceHandle, img *image.RGBA) error {
	if r.fail != nil {
		return r.fail
	}
	r.frames = append(r.frames, img)
	r.handles = append(r.handles, h)
	return nil
}

func (r *recorder) Release(h gpu.SurfaceHandle) { r.released = append(r.released, h) }

var webHandle = gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 0}

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
		{"transparent over black", color.RGBA{}, [3]uint8{}, color.RGBA{0, 0, 0, 255}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 2, 1))
			img.SetRGBA(0, 0, tc.in)
			img.SetRGBA(1, 0, tc.in)
			overBackground(img, tc.bg)
			for x := 0; x < 2; x++ {
				if got := img.RGBAAt(x, 0); got != tc.want {
					t.Errorf("pixel %d = %v, want %v", x, got, tc.want)
				}
			}
		})
	}
}

func TestRenderPresentsWhatSoftDraws(t *testing.T) {
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(4, 4, 20, 10), gputest.FillPaint(0xFF3366CC))
	})
	rec := &recorder{}
	r := NewRenderer(rec, Options{Background: ui.NewColor(0xFF6750A4)})
	s, err := r.Attach(webHandle)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	size := ui.Size{Width: 40, Height: 30}
	if err := r.Render(s, scene, size, 1); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(rec.frames) != 1 || rec.handles[0] != webHandle {
		t.Fatalf("presented %d frames to %v, want 1 to %v", len(rec.frames), rec.handles, webHandle)
	}
	want, err := soft.Render(scene, 40, 30)
	if err != nil {
		t.Fatalf("soft.Render: %v", err)
	}
	// The scene covers the whole surface with white, so the background never shows and the
	// frame is exactly what soft draws.
	if d := gputest.DiffRect(want, rec.frames[0], want.Rect); d.Max != 0 {
		t.Errorf("the frame differs from soft's by up to %d", d.Max)
	}
}

func TestRenderCaptureAndBackground(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec, Options{Background: ui.NewColor(0xFF6750A4)})
	s, err := r.Attach(webHandle)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	// A nil scene draws the background only.
	img, err := r.RenderCapture(s, nil, ui.Size{Width: 8, Height: 6}, 1)
	if err != nil {
		t.Fatalf("RenderCapture: %v", err)
	}
	if got, want := img.RGBAAt(3, 3), (color.RGBA{0x67, 0x50, 0xA4, 255}); got != want {
		t.Errorf("an empty scene shows %v, want the background %v", got, want)
	}
	if len(rec.frames) != 1 || rec.frames[0] != img {
		t.Errorf("the captured image is not the one presented")
	}
}

func TestSnapshotHasNoBackground(t *testing.T) {
	r := NewRenderer(&recorder{}, Options{Background: ui.NewColor(0xFF6750A4)})
	img, err := r.Snapshot(nil, ui.Size{Width: 4, Height: 4})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if got := img.RGBAAt(1, 1); got != (color.RGBA{}) {
		t.Errorf("a snapshot of nothing is %v, want transparent", got)
	}
}

func TestDestroyAndClose(t *testing.T) {
	rec := &recorder{}
	r := NewRenderer(rec)
	s, err := r.Attach(webHandle)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	s.Destroy()
	s.Destroy()
	if len(rec.released) != 1 {
		t.Errorf("released %d times, want once", len(rec.released))
	}
	if err := r.Render(s, nil, ui.Size{Width: 2, Height: 2}, 1); err == nil {
		t.Error("Render into a destroyed surface worked")
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := r.Attach(webHandle); err == nil {
		t.Error("Attach after Close worked")
	}
}

func TestErrorRoutes(t *testing.T) {
	newRenderer := func() (*Renderer, gpu.Surface) {
		r := NewRenderer(&recorder{})
		s, err := r.Attach(webHandle)
		if err != nil {
			t.Fatalf("Attach: %v", err)
		}
		return r, s
	}
	closed := NewRenderer(&recorder{})
	_ = closed.Close()
	tests := []struct {
		name string
		do   func() error
		tag  string
		fn   string
	}{
		{"wrong kind", func() error {
			_, err := NewRenderer(&recorder{}).Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland})
			return err
		}, "SurfaceKindUnsupported", "Attach"},
		{"closed", func() error { _, err := closed.Attach(webHandle); return err }, "RendererClosed", "Attach"},
		{"foreign surface", func() error {
			r, _ := newRenderer()
			_, other := newRenderer()
			return r.Render(other, nil, ui.Size{Width: 2, Height: 2}, 1)
		}, "SurfaceInvalid", "RenderCapture"},
		{"bad size", func() error {
			r, s := newRenderer()
			return r.Render(s, nil, ui.Size{Width: 0, Height: 2}, 1)
		}, "SoftSizeInvalid", "Render"},
		{"presenter fails", func() error {
			rec := &recorder{fail: errors.New("PresentFailed", "no")}
			r := NewRenderer(rec)
			s, _ := r.Attach(webHandle)
			return r.Render(s, nil, ui.Size{Width: 2, Height: 2}, 1)
		}, "PresentFailed", ""},
		{"canvas not found", func() error { return errCanvasNotFound(3) }, "CanvasNotFound", "errCanvasNotFound"},
		{"no context", func() error { return errContextUnavailable(3) }, "Canvas2DUnavailable", "errContextUnavailable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.do()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tc.tag || !appErr.HasRoute(tc.fn+"."+tc.tag) && tc.fn != "" {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tc.tag, tc.fn)
			}
		})
	}
}

func TestTwoOptionsPanic(t *testing.T) {
	defer func() {
		r := recover()
		appErr, ok := r.(*errors.Error)
		if !ok || appErr.Tag != "MultipleOptions" {
			t.Errorf("recovered %v, want a MultipleOptions error", r)
		}
	}()
	NewRenderer(&recorder{}, Options{}, Options{})
}
