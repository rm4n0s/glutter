//go:build linux && !android

package wayland

import (
	"image"
	"os"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
)

// sharedMappings is how many mappings of this process are the memory of a wl_shm buffer (the
// memfd that newShmBufferFormat makes is called glutter-shm).
func sharedMappings() int {
	maps, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return 0
	}
	return strings.Count(string(maps), "memfd:glutter-shm")
}

// TestLeaksPixelBuffers draws frames of the software renderer into a window, again and again, at
// three sizes in turn and with frames that change nothing in between (step 32.1, SPEC §15 "Memory
// over time"): the buffers of a size that is no longer asked for are unmapped and destroyed when
// the compositor releases them, the pool keeps three, and the heap, the blocks of C memory and the
// mappings do not grow.
func TestLeaksPixelBuffers(t *testing.T) {
	leaktest.Run(t, "wayland/pixel buffers", func() func() {
		h := &pixelHandler{}
		h.script = []func(*window){h.frame(640, 400, 1, image.Rect(0, 0, 640, 400))}
		e, srv := startPixelWindow(t, fakecomp.Config{}, h)
		waitUntil(t, func() bool { return len(srv.RequestsWith("wl_surface.commit buffer=")) >= 1 }, "the window was not mapped")
		sizes := [][2]int{{640, 400}, {800, 500}, {320, 200}}
		i := 0
		return func() {
			i++
			size := sizes[i%len(sizes)]
			e.onLoop(func() {
				w := e.p.windows[0]
				b, err := w.AcquirePixels(size[0], size[1])
				if err != nil {
					t.Error(err)
					return
				}
				b.Pix[0] = byte(i)
				if err := w.PresentPixels([]image.Rectangle{image.Rect(0, 0, 64, 64)}); err != nil {
					t.Error(err)
				}
				if i%4 == 0 {
					_ = w.PresentPixels(nil) // a frame that changed nothing
				}
			})
			e.onLoop(func() { // the compositor's releases arrive
				if err := e.p.sess.conn.Roundtrip(); err != nil {
					t.Error(err)
				}
			})
			srv.ClearRequests()
		}
	}, leaktest.Opts{N: 120, CounterSlack: 8, Counters: map[string]func() int{
		"cblocks":  func() int { return int(dl.LiveBlocks()) },
		"mappings": sharedMappings,
	}})
}
