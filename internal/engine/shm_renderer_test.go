package engine

import (
	"image"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/ui"
)

// pixelRecorder is a gpu.PixelTarget over one buffer, which keeps what it is told.
type pixelRecorder struct {
	buf      []byte
	width    int
	acquired int
	damages  [][]image.Rectangle
}

func (p *pixelRecorder) AcquirePixels(width, height int) (gpu.PixelBuffer, error) {
	p.acquired++
	if p.buf == nil {
		p.buf, p.width = make([]byte, 4*width*height), width
	}
	age := 0
	if len(p.damages) > 0 {
		age = 1
	}
	return gpu.PixelBuffer{Pix: p.buf, Stride: 4 * width, Width: width, Height: height, Age: age}, nil
}

func (p *pixelRecorder) PresentPixels(d []image.Rectangle) error {
	p.damages = append(p.damages, d)
	return nil
}

func TestSHMRendererDrawsInvalidatesAndReleases(t *testing.T) {
	rec := &pixelRecorder{}
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Display: 1, Surface: 0x77}
	targets := func(got gpu.SurfaceHandle) (gpu.PixelTarget, bool) { return rec, got == h }
	log, buf := recordingLog()
	r := newSHMRenderer(targets, log)
	back := gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 128, 128))
	scene := func(x float64) *ui.Scene {
		return gputest.DamageScene(back, gputest.PictureOf(func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(x, 70, 20, 20), gputest.FillPaint(0xFFE53935))
		}))
	}

	r.Invalidate(h) // a handle that was never drawn into is ignored
	img, err := r.Draw(h, 128, 128, scene(2), true)
	if err != nil || img == nil || img.Bounds().Dx() != 128 {
		t.Fatalf("Draw with capture = %v, %v", img, err)
	}
	// The window is shown in the colors that the other renderers clear to.
	if got := img.RGBAAt(100, 10); got.A != 255 {
		t.Errorf("a pixel of the capture is %v, want opaque", got)
	}
	if img, err := r.Draw(h, 128, 128, scene(66), false); err != nil || img != nil {
		t.Fatalf("Draw without capture = %v, %v", img, err)
	}
	if len(rec.damages) != 2 || len(rec.damages[0]) != 1 || rec.damages[0][0] != image.Rect(0, 0, 128, 128) {
		t.Fatalf("damages = %v, want the whole window and then a part", rec.damages)
	}
	if d := rec.damages[1]; len(d) == 0 || d[0] == image.Rect(0, 0, 128, 128) {
		t.Errorf("the second frame reported %v, want the tiles of the square", d)
	}
	r.Invalidate(h)
	if _, err := r.Draw(h, 128, 128, scene(66), false); err != nil {
		t.Fatal(err)
	}
	if d := rec.damages[2]; len(d) != 1 || d[0] != image.Rect(0, 0, 128, 128) {
		t.Errorf("the frame after Invalidate reported %v, want the whole window", d)
	}
	if got := r.Describe(); !strings.Contains(got, "wl_shm") {
		t.Errorf("Describe = %q, want it to name wl_shm", got)
	}
	if n := strings.Count(buf.String(), "software renderer ready"); n != 1 {
		t.Errorf("the log says the renderer is ready %d times:\n%s", n, buf.String())
	}
	r.Release(h)
	r.Release(h)
	r.Close()
	if _, err := r.Draw(h, 4, 4, scene(2), false); err == nil {
		t.Error("Draw after Close succeeded")
	}
}

func TestSHMRendererRefusesAWindowWithNoPixels(t *testing.T) {
	r := newSHMRenderer(func(gpu.SurfaceHandle) (gpu.PixelTarget, bool) { return nil, false }, quiet())
	_, err := r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Surface: 3}, 4, 4, nil, false)
	assertRoute(t, err, "PixelTargetMissing", "shmRenderer.Draw->Renderer.Attach.PixelTargetMissing")
	_, err = r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 3}, 4, 4, nil, false)
	assertRoute(t, err, "SurfaceKindUnsupported", "shmRenderer.Draw->Renderer.Attach.SurfaceKindUnsupported")
}
