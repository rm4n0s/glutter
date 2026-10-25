package engine

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gdiwindow"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/ui"
)

// gdiRecorder is a gdiwindow.Presenter that keeps the frames and the damage.
type gdiRecorder struct {
	frames   int
	damage   []gpu.Damage
	released []gpu.SurfaceHandle
}

func (r *gdiRecorder) Present(_ gpu.SurfaceHandle, _ *image.RGBA, d gpu.Damage) error {
	r.frames++
	r.damage = append(r.damage, d)
	return nil
}
func (r *gdiRecorder) Release(h gpu.SurfaceHandle) { r.released = append(r.released, h) }

func TestGDIRendererDrawsInvalidatesAndReleases(t *testing.T) {
	rec := &gdiRecorder{}
	made := 0
	r := newGDIRenderer(func() gdiwindow.Presenter { made++; return rec }, quiet())
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 0x77}
	scene := gputest.CanvasScene(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), gputest.FillPaint(0xFF3366CC)) })

	r.Invalidate(h) // a handle that was never drawn into is ignored
	if img, err := r.Draw(h, 40, 30, scene, true); err != nil || img == nil || img.Bounds().Dx() != 40 {
		t.Fatalf("Draw with capture = %v, %v", img, err)
	}
	if img, err := r.Draw(h, 40, 30, scene, false); err != nil || img != nil {
		t.Fatalf("Draw without capture = %v, %v", img, err)
	}
	if made != 1 || rec.frames != 2 {
		t.Errorf("made %d presenters and %d frames, want 1 and 2", made, rec.frames)
	}
	if !rec.damage[0].Full {
		t.Errorf("the first frame has damage %+v, want Full", rec.damage[0])
	}
	r.Invalidate(h)
	if _, err := r.Draw(h, 40, 30, scene, false); err != nil {
		t.Fatal(err)
	}
	if !rec.damage[2].Full {
		t.Errorf("the frame after Invalidate has damage %+v, want Full", rec.damage[2])
	}
	if got := r.Describe(); got == "" {
		t.Error("Describe is empty")
	}
	r.Release(h)
	r.Release(h)
	if len(rec.released) != 1 {
		t.Errorf("released %d times, want 1", len(rec.released))
	}
	r.Close()
	if _, err := r.Draw(h, 4, 4, scene, false); err == nil {
		t.Error("Draw after Close succeeded")
	}
}

func TestGDIRendererRefusesAnotherKindOfSurface(t *testing.T) {
	r := newGDIRenderer(func() gdiwindow.Presenter { return &gdiRecorder{} }, quiet())
	_, err := r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Window: 3}, 4, 4, nil, false)
	assertRoute(t, err, "SurfaceKindUnsupported", "gdiRenderer.Draw->Renderer.Attach.SurfaceKindUnsupported")
}
