package offscreen

import (
	"github.com/rm4n0s/glutter/internal/fakeclock"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// NewViewWithClock is NewView with the clock that paces the frames.
func NewViewWithClock(parent *widgets.WidgetsBinding, size ui.Size, root widgets.Widget, onFrame func(Frame), clock fakeclock.Clock, o ...ViewOpts) (*View, error) {
	var opt ViewOpts
	if len(o) > 0 {
		opt = o[0]
	}
	return newView(parent, size, root, onFrame, clock, opt)
}

// WaitIdle waits until the raster goroutine drew every scene that was given to it.
func (v *View) WaitIdle() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for v.pending != nil || v.busy {
		v.idle.Wait()
	}
}

// Held is what the view keeps of the frames: the scenes that wait for the goroutine (at
// most one), and the copies the detacher remembers of its last scene.
func (v *View) Held() (waiting, copies int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.pending != nil {
		waiting = 1
	}
	return waiting, len(v.detacher.prev.pictures) + len(v.detacher.prev.images) + len(v.detacher.prev.shaders)
}

// Scheduled is whether a frame is asked for and has not begun.
func (v *View) Scheduled() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.scheduled
}

// IsClosed is whether Close began.
func IsClosed(v *View) bool { return v.isClosed() }
