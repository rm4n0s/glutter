package gles

import (
	"github.com/rm4n0s/glutter/ui"
)

// NewBenchRig makes a renderer over the fake EGL and GL, with the log of the calls off, and
// returns the function that draws a scene of w x h pixels on it and the one that closes it. The
// benchmarks of package gles_test use it, since they import packages that import gles.
func NewBenchRig(w, h int, opts ...RendererOpts) (draw func(*ui.Scene) error, closeRig func(), err error) {
	r, _, gl := newTestRenderer(opts...)
	gl.quiet = true
	s, err := r.Attach(waylandHandle)
	if err != nil {
		return nil, nil, err
	}
	size := ui.Size{Width: float64(w), Height: float64(h)}
	return func(scene *ui.Scene) error { return r.Render(s, scene, size, 1) }, func() { _ = r.Close() }, nil
}
