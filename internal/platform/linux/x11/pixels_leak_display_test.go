//go:build linux && !android && glutter_display

package x11

import (
	"image"
	"os"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// sysvMappings is how many mappings of this process are segments of System V shared memory (the
// buffers of MIT-SHM; the kernel names them /SYSV<key>).
func sysvMappings() int {
	maps, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return 0
	}
	return strings.Count(string(maps), "/SYSV")
}

// TestLeaksPixelBuffers sends frames of the software renderer to a window, again and again, at three
// sizes in turn and with frames that change nothing in between (step 32.2, SPEC §15 "Memory over
// time"), with MIT-SHM and with XPutImage: the buffers of a size that is no longer asked for are
// freed when the server releases them, the pool keeps three, and the heap, the blocks of C memory,
// the mappings of segments and the segments registered with the platform do not grow.
func TestLeaksPixelBuffers(t *testing.T) {
	for _, mode := range []struct {
		name  string
		noSHM bool
	}{{"MIT-SHM", false}, {"XPutImage", true}} {
		t.Run(mode.name, func(t *testing.T) {
			leaktest.Run(t, "x11/pixel buffers/"+mode.name, func() func() {
				e := startPixelDisplay(t, mode.noSHM, Options{PixelWindows: true})
				w := e.win.(*window)
				sizes := [][2]int{{640, 400}, {800, 500}, {320, 200}}
				i := 0
				return func() {
					i++
					size := sizes[i%len(sizes)]
					e.onLoop(func() {
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
					e.onLoop(func() { // the server's completion events arrive
						e.p.x.sync(e.p.display, 0)
						e.p.dispatchEvents()
					})
				}
			}, leaktest.Opts{N: 120, CounterSlack: 8, Counters: map[string]func() int{
				"cblocks":  func() int { return int(dl.LiveBlocks()) },
				"mappings": sysvMappings,
			}})
		})
	}
}
