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

// NewLeakRig is NewBenchRig with a function that draws a scene of any size, and one that counts
// the objects of the fake GL that were made and not deleted (kinds: "texture", "buffer",
// "framebuffer", "renderbuffer", "program", "vao"). The memory tests of package gles_test use it.
func NewLeakRig(opts ...RendererOpts) (draw func(s *ui.Scene, w, h int) error, live func(kind string) int, closeRig func(), err error) {
	r, egl, gl := newTestRenderer(opts...)
	gl.quiet = true
	s, err := r.Attach(waylandHandle)
	if err != nil {
		return nil, nil, nil, err
	}
	return func(scene *ui.Scene, w, h int) error {
			err := r.Render(s, scene, ui.Size{Width: float64(w), Height: float64(h)}, 1)
			egl.calls = egl.calls[:0] // the log of the fake EGL is not the memory of the renderer
			return err
		},
		func(kind string) int { return gl.liveCount(kind) },
		func() { _ = r.Close() }, nil
}
