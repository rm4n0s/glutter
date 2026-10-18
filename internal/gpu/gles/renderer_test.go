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
	if !ok || appErr.Tag != "RendererSurfaceInvalid" || !appErr.HasRoute("Renderer.render.RendererSurfaceInvalid") {
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
		{"zero size", func() error { return r.Render(s, nil, ui.Size{Width: 0, Height: 5}, 1) }, "RendererSizeInvalid", "Renderer.render.RendererSizeInvalid"},
		{"foreign surface", func() error { return r.Render(foreign, nil, ui.Size{Width: 5, Height: 5}, 1) }, "RendererSurfaceInvalid", "Renderer.render.RendererSurfaceInvalid"},
		{"nil surface", func() error { return r.Render(nil, nil, ui.Size{Width: 5, Height: 5}, 1) }, "RendererSurfaceInvalid", "Renderer.render.RendererSurfaceInvalid"},
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

func TestRendererRenderCapture(t *testing.T) {
	r, egl, gl := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	// Two rows of two pixels, bottom row first.
	gl.pixels = []byte{
		1, 2, 3, 255, 4, 5, 6, 255,
		7, 8, 9, 255, 10, 11, 12, 255,
	}
	img, err := r.RenderCapture(s, nil, ui.Size{Width: 2, Height: 2}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if img == nil || img.Pix[0] != 7 || img.Pix[8] != 1 {
		t.Errorf("image = %v, want the rows flipped to top first", img)
	}
	// The frame was read before it was presented.
	read, swap := -1, -1
	for i, c := range gl.log {
		if len(c) >= 10 && c[:10] == "readPixels" {
			read = i
		}
	}
	for i, c := range egl.calls {
		if c == "swapBuffers" {
			swap = i
		}
	}
	if read < 0 || swap < 0 {
		t.Fatalf("read %d, swap %d", read, swap)
	}
	// Render itself reads nothing.
	gl.resetCalls()
	if err := r.Render(s, nil, ui.Size{Width: 2, Height: 2}, 1); err != nil {
		t.Fatal(err)
	}
	if gl.calls["readPixels"] != 0 {
		t.Error("Render read the frame back")
	}
}

func TestRendererRenderCaptureErrors(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, _ := r.Attach(waylandHandle)
	gl.errCode = 0x502
	_, err := r.RenderCapture(s, nil, ui.Size{Width: 2, Height: 2}, 1)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "GLReadPixelsFailed" {
		t.Errorf("RenderCapture = %v, want GLReadPixelsFailed", err)
	}
}

func TestRendererDrawsTheScene(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, _ := r.Attach(waylandHandle)
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	red := ui.NewPaint()
	red.SetColor(ui.NewColor(0xFFFF0000))
	c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), red)
	c.DrawCircle(ui.Offset{Dx: 20, Dy: 20}, 5, red)
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	if err := r.Render(s, sb.Build(), ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	st := r.LastFrame()
	if st.Commands != 2 || st.Drawn != 2 || st.Instances != 2 || st.Batches != 1 || st.Skipped != 0 {
		t.Errorf("stats = %+v", st)
	}
	// One draw call for the batch; the instances were uploaded once, orphaning the old data.
	if gl.calls["drawArraysInstanced"] != 1 || gl.calls["bufferSubData"] != 1 {
		t.Errorf("draw calls %d, uploads %d", gl.calls["drawArraysInstanced"], gl.calls["bufferSubData"])
	}
	// The scissor is off again afterwards, so the next clear covers the surface.
	if err := r.Render(s, nil, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Commands != 0 {
		t.Errorf("an empty frame has stats %+v", st)
	}
}

func TestRendererStrictAndSkippingModes(t *testing.T) {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	blurred := ui.NewPaint()
	blurred.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
	c.DrawArc(ui.NewRectFromLTWH(0, 0, 10, 10), 0, 1, true, blurred)
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	scene := sb.Build()

	strict, egl, _ := newTestRenderer()
	s, _ := strict.Attach(waylandHandle)
	err := strict.Render(s, scene, ui.Size{Width: 8, Height: 8}, 1)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "GLNotImplemented" || !appErr.HasRoute("Renderer.render->Renderer.drawScene->frameBuilder.command") {
		t.Fatalf("Render = %v, want GLNotImplemented from the frame builder", err)
	}
	if contains(egl.calls, "swapBuffers") {
		t.Error("a failed frame was presented")
	}

	skipping, egl2, _ := newTestRenderer(RendererOpts{SkipUnsupported: true})
	s2, _ := skipping.Attach(waylandHandle)
	if err := skipping.Render(s2, scene, ui.Size{Width: 8, Height: 8}, 1); err != nil {
		t.Fatal(err)
	}
	if st := skipping.LastFrame(); st.Skipped != 1 || len(st.Unsupported) != 1 {
		t.Errorf("stats = %+v", st)
	}
	if !contains(egl2.calls, "swapBuffers") {
		t.Error("the frame was not presented")
	}
}

func TestRendererCloseReleasesWhatTheCachesHold(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(5, 5, 50, 50), 10, 10))
	path := ui.NewPath()
	path.MoveTo(0, 0)
	path.LineTo(40, 5)
	path.LineTo(10, 40)
	path.Close()
	c.DrawPath(path, ui.NewPaint())
	c.DrawImage(ui.NewImageFromRGBA(make([]byte, 4*8*8), 8, 8), ui.Offset{}, nil)
	c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, []ui.Offset{{}, {Dx: 10}, {Dy: 10}}), ui.BlendModeSrc, ui.NewPaint())
	c.DrawRect(ui.NewRectFromLTWH(1, 1, 5, 5), ui.NewPaint())
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	if err := r.Render(s, sb.Build(), ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	st := r.LastFrame()
	if st.Skipped != 0 || st.TilesRasterized == 0 || st.ClipMasksBuilt != 1 || st.ImagesUploaded != 1 || st.Vertices != 3 {
		t.Errorf("stats = %+v", st)
	}
	if gl.liveCount("texture") < 4 || gl.liveCount("buffer") == 0 || gl.liveCount("vao") == 0 {
		t.Errorf("live: textures %d, buffers %d, vertex arrays %d", gl.liveCount("texture"), gl.liveCount("buffer"), gl.liveCount("vao"))
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"texture", "buffer", "vao", "program", "shader", "framebuffer", "renderbuffer"} {
		if n := gl.liveCount(kind); n != 0 {
			t.Errorf("%d %s objects leaked after Close", n, kind)
		}
	}
}
