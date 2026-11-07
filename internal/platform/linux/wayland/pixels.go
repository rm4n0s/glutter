//go:build linux && !android

package wayland

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// This file makes a window a gpu.PixelTarget, for the software renderer (SPEC §9.6): the renderer
// draws into a wl_shm buffer of the window's pool and the window attaches it, reports the damage
// and commits, as the swap of EGL does.

var _ gpu.PixelTarget = (*window)(nil)

// pixels is the memory of the buffer (poolBuffer).
func (b *shmBuffer) pixels() []byte { return b.data }

// pixelPoolOf returns the buffer pool of the window, made at the first frame.
func (w *window) pixelPoolOf() *pixelPool {
	if w.pixelPool == nil {
		shm := w.p.sess.shm
		w.pixelPool = newPixelPool(func(width, height int, release func()) (poolBuffer, error) {
			b, err := newShmBufferFormat(shm, int32(width), int32(height), wl.WlShmFormatXrgb8888, func([]byte) {}, func(*shmBuffer) {
				if !w.closed {
					release()
				}
			})
			if err != nil {
				return nil, err // not b: a nil *shmBuffer in the interface would not be nil
			}
			return b, nil
		})
	}
	return w.pixelPool
}

// AcquirePixels returns the buffer of the next frame: a wl_shm buffer of the window's pool that the
// compositor does not read, of width x height pixels in XRGB8888, with its age.
//
// Flutter: no counterpart
func (w *window) AcquirePixels(width, height int) (gpu.PixelBuffer, error) {
	pool := w.pixelPoolOf()
	slot, age, err := pool.acquire(width, height)
	if err != nil {
		return gpu.PixelBuffer{}, err
	}
	w.pixelCur = slot
	return gpu.PixelBuffer{
		Pix: slot.buf.pixels(), Stride: 4 * width, Width: width, Height: height,
		Format: gpu.PixelFormatBGRX8888, Age: age,
	}, nil
}

// PresentPixels attaches the buffer that AcquirePixels returned last, reports each damaged
// rectangle (in buffer pixels, which damage_buffer takes, so the scale does not matter) and commits.
// The frame callback that renderNow asked for goes with this commit. With no buffer acquired it
// only commits, which the compositor answers with the frame callback, so that the frame pacing of
// an idle frame goes on.
//
// Flutter: no counterpart
func (w *window) PresentPixels(damage []image.Rectangle) error {
	slot := w.pixelCur
	w.pixelCur = nil
	if w.closed || w.hidden || w.remapping {
		return nil // a window that is not on the screen shows nothing; it is drawn whole when shown
	}
	if slot == nil {
		w.surface.Commit()
		return nil
	}
	w.note("commit", int32(slot.width), int32(slot.height))
	b := slot.buf.(*shmBuffer)
	w.surface.Attach(b.buffer, 0, 0)
	if len(damage) == 0 {
		damage = []image.Rectangle{image.Rect(0, 0, slot.width, slot.height)}
	}
	for _, r := range damage {
		w.surface.DamageBuffer(int32(r.Min.X), int32(r.Min.Y), int32(r.Dx()), int32(r.Dy()))
	}
	w.surface.Commit()
	w.pixelPool.present(slot)
	w.mapped = true
	return nil
}
