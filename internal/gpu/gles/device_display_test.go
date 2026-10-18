//go:build glutter_display

package gles

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/ui"
)

// These tests need a Wayland compositor and a working EGL driver. Run them under
// the headless compositor:
//
//	scripts/headless-wayland.sh -- go test -race -tags glutter_display ./internal/gpu/gles/

// frameHandler runs fn once, on the UI thread, at the first frame of the window.
type frameHandler struct {
	platform.NopHandler
	win  platform.Window
	fn   func(h gpu.SurfaceHandle, w, hgt int) error
	done chan<- error
	w, h int
	ran  bool
}

func (f *frameHandler) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) {
	f.w, f.h = int(m.PhysicalSize.Width), int(m.PhysicalSize.Height)
}

func (f *frameHandler) OnFrame(ui.ViewID, time.Duration) {
	if f.ran {
		return
	}
	f.ran = true
	f.done <- f.fn(f.win.NativeSurface(), f.w, f.h)
}

// onDisplay opens a window and runs fn at its first frame, with the native
// surface and the window's size in pixels.
func onDisplay(t *testing.T, fn func(h gpu.SurfaceHandle, w, hgt int) error) {
	t.Helper()
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v (run under scripts/headless-wayland.sh)", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "gles device test"
	cfg.Size = ui.Size{Width: 200, Height: 100}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runErr := make(chan error, 1)
	go func() { runErr <- p.Run(ctx, &frameHandler{win: win, fn: fn, done: done}) }()
	var res error
	select {
	case res = <-done:
	case <-ctx.Done():
		t.Fatal("no frame was drawn within 20 seconds")
	}
	cancel()
	if err := <-runErr; err != nil {
		t.Errorf("Run: %v", err)
	}
	if res != nil {
		if appErr, ok := errors.FromError(res); ok {
			t.Fatalf("%s: %s %v (%s)", appErr.Tag, appErr.Message, appErr.Metadata, appErr.Route())
		}
		t.Fatal(res)
	}
}

// readFramebuffer reads w x h pixels of the bound framebuffer, bottom row first.
func readFramebuffer(c *Context, w, h int) ([]byte, error) {
	raw := make([]byte, w*h*4)
	c.gl.readPixels(0, 0, int32(w), int32(h), glRgba, glUnsignedByte, raw)
	return raw, c.dev.checkError("readPixels")
}

func TestDisplayDeviceCompilesEveryShader(t *testing.T) {
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		c, s, err := Open(h)
		if err != nil {
			return err
		}
		defer c.Close()
		defer s.Destroy()
		names, err := ShaderNames()
		if err != nil {
			return err
		}
		caps := c.Device().Caps()
		t.Logf("%s | %s | %s | max texture %d, %d extensions", caps.Version, caps.Vendor, caps.Renderer, caps.MaxTextureSize, len(caps.Extensions))
		if got := c.Device().Stats().Programs; got != len(names) {
			return errors.NewErrf("TestPrograms", "%d programs for %d shader pairs", got, len(names))
		}
		for _, name := range names {
			if _, err := c.Device().Program(name); err != nil {
				return err
			}
		}
		// The driver rejects bad GLSL with a readable log.
		_, err = c.Device().NewProgram("broken", "#version 300 es\nvoid main() { gl_Position = nope; }", "#version 300 es\nvoid main() {}")
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "ShaderCompileFailed" || appErr.Metadata["log"] == "" {
			return errors.NewErrf("TestBrokenShader", "expected ShaderCompileFailed with a log, got %v", err)
		}
		t.Logf("driver log for a broken shader: %v", appErr.Metadata["log"])
		return c.Device().checkError("end")
	})
}

func TestDisplayDrawSolidIntoFramebuffer(t *testing.T) {
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		c, s, err := Open(h)
		if err != nil {
			return err
		}
		defer c.Close()
		defer s.Destroy()
		d := c.Device()
		fb, err := d.AcquireFramebuffer(8, 8, true)
		if err != nil {
			return err
		}
		d.BindFramebuffer(fb)
		d.Viewport(0, 0, 8, 8)
		c.gl.clearColor(0, 0, 0, 0)
		c.gl.clear(glColorBufferBit)
		p, err := d.Program("solid")
		if err != nil {
			return err
		}
		d.UseProgram(p)
		c.gl.uniform4f(p.Uniform("uColor"), 0.5, 0, 0, 0.5) // premultiplied 50% red
		c.gl.drawArrays(glTriangles, 0, 3)
		raw, err := readFramebuffer(c, 8, 8)
		if err != nil {
			return err
		}
		for i := 0; i < len(raw); i += 4 {
			px := [4]uint8{raw[i], raw[i+1], raw[i+2], raw[i+3]}
			if abs(int(px[0])-128) > 1 || px[1] != 0 || px[2] != 0 || abs(int(px[3])-128) > 1 {
				return errors.NewErrf("TestPixel", "pixel %d = %v, want about [128 0 0 128]", i/4, px)
			}
		}
		d.ReleaseFramebuffer(fb)
		d.BindFramebuffer(nil)
		return d.checkError("end")
	})
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestDisplayTextureUploadAndBlit(t *testing.T) {
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		c, s, err := Open(h)
		if err != nil {
			return err
		}
		defer c.Close()
		defer s.Destroy()
		d := c.Device()
		tex, err := d.AcquireTexture(2, 2, TextureFormatRGBA8)
		if err != nil {
			return err
		}
		d.SetTextureParams(tex, glNearest, glNearest, glClampToEdge, glClampToEdge)
		// Row 0 of the upload is the bottom row of the texture: red, green; then
		// blue, white.
		pixels := []byte{
			255, 0, 0, 255, 0, 255, 0, 255,
			0, 0, 255, 255, 255, 255, 255, 255,
		}
		if err := d.UploadTexture(tex, 0, 0, 2, 2, pixels); err != nil {
			return err
		}
		fb, err := d.AcquireFramebuffer(4, 4, false)
		if err != nil {
			return err
		}
		d.BindFramebuffer(fb)
		d.Viewport(0, 0, 4, 4)
		c.gl.clearColor(0, 0, 0, 0)
		c.gl.clear(glColorBufferBit)
		p, err := d.Program("blit")
		if err != nil {
			return err
		}
		d.UseProgram(p)
		d.BindTexture(0, tex)
		c.gl.uniform1i(p.Uniform("uTexture"), 0)
		c.gl.drawArrays(glTriangles, 0, 3)
		raw, err := readFramebuffer(c, 4, 4)
		if err != nil {
			return err
		}
		want := [4][4]uint8{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}}
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				i := (y*4 + x) * 4
				got := [4]uint8{raw[i], raw[i+1], raw[i+2], raw[i+3]}
				if w := want[(y/2)*2+x/2]; got != w {
					return errors.NewErrf("TestPixel", "pixel %d,%d = %v, want %v", x, y, got, w)
				}
			}
		}
		// The framebuffer's own texture can be sampled in turn.
		d.BindFramebuffer(nil)
		d.ReleaseFramebuffer(fb)
		d.ReleaseTexture(tex)
		return d.checkError("end")
	})
}

func TestDisplayRendererRendersBackground(t *testing.T) {
	onDisplay(t, func(h gpu.SurfaceHandle, w, hgt int) error {
		r := NewRenderer(RendererOpts{Background: ui.NewColor(0xFF3366CC), HasBackground: true})
		defer func() { _ = r.Close() }()
		var raw []byte
		r.beforePresent = func(w, h int) error {
			var err error
			raw, err = readFramebuffer(r.ctx, w, h)
			return err
		}
		s, err := r.Attach(h)
		if err != nil {
			return err
		}
		for frame := 0; frame < 2; frame++ { // the second frame reuses everything
			if err := r.Render(s, nil, ui.Size{Width: float64(w), Height: float64(hgt)}, 1); err != nil {
				return err
			}
			if len(raw) != w*hgt*4 {
				return errors.NewErrf("TestSize", "read %d bytes for %dx%d", len(raw), w, hgt)
			}
			for i := 0; i < len(raw); i += 4 {
				got := [4]uint8{raw[i], raw[i+1], raw[i+2], raw[i+3]}
				if want := [4]uint8{0x33, 0x66, 0xCC, 0xFF}; got != want {
					return errors.NewErrf("TestPixel", "frame %d pixel %d = %v, want %v", frame, i/4, got, want)
				}
			}
		}
		s.Destroy()
		return nil
	})
}
