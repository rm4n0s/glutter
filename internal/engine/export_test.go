package engine

import (
	"context"
	"fmt"
	"image"
	"log/slog"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// RunOnFake runs the engine on the fake platform with a renderer that draws
// nothing, for the tests of package engine_test that run the framework in the
// real loop (they cannot be in package engine: the framework imports it).
func RunOnFake(ctx context.Context, cfg Config, p *fake.Platform) error {
	cfg.connectors = map[string]func() (connection, error){"": func() (connection, error) { return &conn{Platform: p}, nil }}
	cfg.newRenderer = func(*Config, *slog.Logger) renderer { return &fakeRenderer{} }
	return Run(ctx, cfg)
}

// SoftStats is what a checked software renderer saw (CheckSoftRenderer).
type SoftStats struct {
	// Frames is how many frames were drawn, Acquires how many took a buffer, and Presents the
	// damage of each presentation (nil for a frame with no buffer).
	Frames, Acquires int
	Presents         [][]image.Rectangle
	// Sizes are the sizes of the buffers that were acquired.
	Sizes [][2]int
	// Mismatches lists the frames whose buffer was not what a whole redraw of the scene gives.
	Mismatches []string
	// Describe is the name of the renderer once it drew, as the log says it.
	Describe string
}

// CheckSoftRenderer replaces the renderer of cfg by the software renderer of the Linux desktop with
// a check: after each frame it compares the memory of the buffer that is on display with a whole
// redraw of the scene by soft over the background, and counts what the renderer asked of the window.
func CheckSoftRenderer(cfg *Config, stats *SoftStats) {
	cfg.newRenderer = func(c *Config, log *slog.Logger) renderer {
		rec := &checkTarget{stats: stats}
		inner := newSHMRenderer(func(h gpu.SurfaceHandle) (gpu.PixelTarget, bool) {
			t, ok := c.pixelTarget(h)
			rec.inner = t
			return rec, ok
		}, log)
		return &checkedSoft{inner: inner, rec: rec, stats: stats}
	}
}

// checkTarget is a gpu.PixelTarget that counts what goes through it, and keeps the memory of the
// buffer that was presented last.
type checkTarget struct {
	inner gpu.PixelTarget
	stats *SoftStats
	cur   []byte
	shown []byte
	w, h  int
}

func (t *checkTarget) AcquirePixels(w, h int) (gpu.PixelBuffer, error) {
	b, err := t.inner.AcquirePixels(w, h)
	if err == nil {
		t.stats.Acquires++
		t.stats.Sizes = append(t.stats.Sizes, [2]int{w, h})
		t.cur, t.w, t.h = b.Pix, w, h
	}
	return b, err
}

// PixelPath forwards what the window says of how its pixels reach the display server.
func (t *checkTarget) PixelPath() string {
	if p, ok := t.inner.(pixelPather); ok {
		return p.PixelPath()
	}
	return ""
}

func (t *checkTarget) PresentPixels(d []image.Rectangle) error {
	t.stats.Presents = append(t.stats.Presents, append([]image.Rectangle(nil), d...))
	if t.cur != nil {
		t.shown, t.cur = t.cur, nil
	}
	return t.inner.PresentPixels(d)
}

// checkedSoft is a renderer over the software renderer that checks every frame.
type checkedSoft struct {
	inner *shmRenderer
	rec   *checkTarget
	stats *SoftStats
}

func (r *checkedSoft) Describe() string { return r.inner.Describe() }

func (r *checkedSoft) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	img, err := r.inner.Draw(h, width, height, scene, capture)
	if err != nil {
		return img, err
	}
	r.stats.Frames++
	r.stats.Describe = r.inner.Describe()
	ref := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(ref.Pix); i += 4 {
		ref.Pix[i], ref.Pix[i+1], ref.Pix[i+2], ref.Pix[i+3] = 0x67, 0x50, 0xA4, 0xFF // developmentColor
	}
	if scene != nil {
		if err := soft.RenderInto(ref, scene, ref.Rect); err != nil {
			return img, err
		}
	}
	shown := r.rec.shown
	if len(shown) < 4*width*height || r.rec.w != width || r.rec.h != height {
		r.stats.Mismatches = append(r.stats.Mismatches, fmt.Sprintf("frame %d: no buffer of %dx%d on display", r.stats.Frames, width, height))
		return img, nil
	}
	for i := 0; i < 4*width*height; i += 4 {
		if shown[i] != ref.Pix[i+2] || shown[i+1] != ref.Pix[i+1] || shown[i+2] != ref.Pix[i] {
			p := i / 4
			r.stats.Mismatches = append(r.stats.Mismatches, fmt.Sprintf("frame %d: pixel %d,%d is %v, want blue %d green %d red %d",
				r.stats.Frames, p%width, p/width, shown[i:i+3], ref.Pix[i+2], ref.Pix[i+1], ref.Pix[i]))
			break
		}
	}
	return img, nil
}

func (r *checkedSoft) Release(h gpu.SurfaceHandle)    { r.inner.Release(h) }
func (r *checkedSoft) Invalidate(h gpu.SurfaceHandle) { r.inner.Invalidate(h) }
func (r *checkedSoft) Close()                         { r.inner.Close() }
