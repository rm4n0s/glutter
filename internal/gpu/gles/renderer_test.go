package gles

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

func newTestRenderer(opts ...RendererOpts) (*Renderer, *fakeEGL, *fakeGL) {
	egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.0 Mesa"}
	r := newRenderer(func(h gpu.SurfaceHandle) (*Context, *Surface, error) {
		return openWith(egl, fakeGLLoader(gl), h)
	}, opts...)
	return r, egl, gl
}

func TestRendererRenderClearsAndPresents(t *testing.T) {
	r, egl, gl := newTestRenderer(RendererOpts{Background: ui.NewColor(0x80FF0000), HasBackground: true})
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Render(s, nil, ui.Size{Width: 320, Height: 200}, 1); err != nil {
		t.Fatal(err)
	}
	if gl.viewportW != 320 || gl.viewportH != 200 {
		t.Errorf("viewport = %dx%d", gl.viewportW, gl.viewportH)
	}
	// 50% red, premultiplied.
	want := [4]float32{0.5019608 * 1, 0, 0, 0.5019608}
	want[0] = want[3]
	for i, w := range want {
		if d := gl.cleared[i] - w; d > 0.01 || d < -0.01 {
			t.Errorf("clear color = %v, want about %v", gl.cleared, want)
			break
		}
	}
	if gl.clears != 1 || !contains(egl.calls, "swapBuffers") {
		t.Errorf("clears %d, EGL calls %v", gl.clears, egl.calls)
	}
	// A second frame at another size.
	if err := r.Render(s, nil, ui.Size{Width: 100, Height: 50}, 2); err != nil {
		t.Fatal(err)
	}
	if gl.viewportW != 100 || gl.clears != 2 {
		t.Errorf("second frame: viewport %dx%d, clears %d", gl.viewportW, gl.viewportH, gl.clears)
	}
}

func TestRendererDefaultBackgroundIsOpaqueWhite(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, _ := r.Attach(waylandHandle)
	if err := r.Render(s, nil, ui.Size{Width: 4, Height: 4}, 1); err != nil {
		t.Fatal(err)
	}
	if gl.cleared != [4]float32{1, 1, 1, 1} {
		t.Errorf("clear color = %v", gl.cleared)
	}
}

func TestRendererSecondSurfaceSharesContext(t *testing.T) {
	r, egl, _ := newTestRenderer()
	a, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range egl.calls {
		if c == "createContext" {
			n++
		}
	}
	if n != 1 || a == b {
		t.Errorf("%d contexts for two surfaces", n)
	}
	a.Destroy()
	a.Destroy()
	appErr, ok := errors.FromError(r.Render(a, nil, ui.Size{Width: 4, Height: 4}, 1))
	if !ok || appErr.Tag != "RendererSurfaceInvalid" || !appErr.HasRoute("Renderer.Render.RendererSurfaceInvalid") {
		t.Errorf("Render on a destroyed surface = %v", appErr)
	}
}

func TestRendererErrors(t *testing.T) {
	r, egl, _ := newTestRenderer()
	s, _ := r.Attach(waylandHandle)
	other, _, _ := newTestRenderer()
	foreign, _ := other.Attach(waylandHandle)
	tests := []struct {
		name  string
		do    func() error
		tag   string
		route string
	}{
		{"zero size", func() error { return r.Render(s, nil, ui.Size{Width: 0, Height: 5}, 1) }, "RendererSizeInvalid", "Renderer.Render.RendererSizeInvalid"},
		{"foreign surface", func() error { return r.Render(foreign, nil, ui.Size{Width: 5, Height: 5}, 1) }, "RendererSurfaceInvalid", "Renderer.Render.RendererSurfaceInvalid"},
		{"nil surface", func() error { return r.Render(nil, nil, ui.Size{Width: 5, Height: 5}, 1) }, "RendererSurfaceInvalid", "Renderer.Render.RendererSurfaceInvalid"},
		{"snapshot", func() error { _, err := r.Snapshot(nil, ui.Size{Width: 5, Height: 5}); return err }, "GLNotImplemented", "Renderer.Snapshot.GLNotImplemented"},
		{"swap fails", func() error { egl.swapFails = true; return r.Render(s, nil, ui.Size{Width: 5, Height: 5}, 1) }, "EGLSwapFailed", "Context.SwapBuffers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(tt.do())
			if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %v, want %s via %s", appErr, tt.tag, tt.route)
			}
		})
	}
}

func TestRendererAttachFailure(t *testing.T) {
	egl, gl := newFakeEGL(), &fakeGL{versionString: "OpenGL ES 3.0 Mesa"}
	egl.noConfig = true
	r := newRenderer(func(h gpu.SurfaceHandle) (*Context, *Surface, error) { return openWith(egl, fakeGLLoader(gl), h) })
	_, err := r.Attach(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "EGLNoConfig" {
		t.Fatalf("Attach = %v, want EGLNoConfig", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("Close without a context: %v", err)
	}
}

func TestRendererClose(t *testing.T) {
	r, egl, gl := newTestRenderer()
	if _, err := r.Attach(waylandHandle); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	for _, want := range []string{"destroySurface", "destroyContext", "terminate"} {
		if !contains(egl.calls, want) {
			t.Errorf("Close did not call %s: %v", want, egl.calls)
		}
	}
	if gl.liveCount("program") != 0 {
		t.Errorf("%d programs leaked", gl.liveCount("program"))
	}
	_, err := r.Attach(waylandHandle)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "RendererClosed" || !appErr.HasRoute("Renderer.Attach.RendererClosed") {
		t.Errorf("Attach after Close = %v", err)
	}
}

func TestNewRendererMultipleOptions(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(recoverError(recover()))
		if !ok || appErr.Tag != "MultipleOptions" || !appErr.HasRoute("newRenderer.MultipleOptions") {
			t.Errorf("expected MultipleOptions, got %v", appErr)
		}
	}()
	NewRenderer(RendererOpts{}, RendererOpts{})
}
