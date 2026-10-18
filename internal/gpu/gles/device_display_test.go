//go:build glutter_display

package gles

import (
	"context"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/proof"
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

// renderGPU draws a scene with a renderer into an offscreen framebuffer of w x h
// pixels cleared to transparent, and reads it back as the software renderer
// returns images: top row first, premultiplied.
func renderGPU(r *Renderer, scene *ui.Scene, w, h int) (*image.RGBA, error) {
	d := r.ctx.dev
	fb, err := d.AcquireFramebuffer(w, h, false)
	if err != nil {
		return nil, err
	}
	defer d.ReleaseFramebuffer(fb)
	d.BindFramebuffer(fb)
	defer d.BindFramebuffer(nil)
	d.Viewport(0, 0, w, h)
	r.ctx.gl.clearColor(0, 0, 0, 0)
	r.ctx.gl.clear(glColorBufferBit)
	if err := r.drawScene(scene, w, h); err != nil {
		return nil, err
	}
	raw, err := readFramebuffer(r.ctx, w, h)
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(img.Pix[y*img.Stride:(y+1)*img.Stride], raw[(h-1-y)*w*4:(h-y)*w*4])
	}
	return img, nil
}

// diffStats compares two images inside a rectangle: the number of pixels whose
// largest channel difference is above each threshold, and the largest difference.
type diffStats struct {
	pixels, over8, over32, over96, max int
}

func diffRect(a, b *image.RGBA, r image.Rectangle) diffStats {
	var s diffStats
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ia, ib := a.PixOffset(x, y), b.PixOffset(x, y)
			m := 0
			for c := 0; c < 4; c++ {
				if d := abs(int(a.Pix[ia+c]) - int(b.Pix[ib+c])); d > m {
					m = d
				}
			}
			s.pixels++
			if m > 8 {
				s.over8++
			}
			if m > 32 {
				s.over32++
			}
			if m > 96 {
				s.over96++
			}
			if m > s.max {
				s.max = m
			}
		}
	}
	return s
}

// dumpImage saves an image into the directory named by GLUTTER_TEST_DUMP, when
// it is set, so that a failing comparison can be looked at.
func dumpImage(t *testing.T, name string, img image.Image) {
	t.Helper()
	dir := os.Getenv("GLUTTER_TEST_DUMP")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Errorf("dump: %v", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Errorf("dump: %v", err)
	}
}

// gpuMatchesSoft renders a proof scene with the GPU and with the software renderer, and calls check for the two images.
func gpuMatchesSoft(t *testing.T, name string, check func(t *testing.T, gpuImg, soft *image.RGBA, r *Renderer)) {
	sc, err := proof.Lookup(name)
	if err != nil {
		t.Fatal(err)
	}
	scene := sc.Build()
	want, err := soft.Render(scene, sc.Width, sc.Height)
	if err != nil {
		t.Fatal(err)
	}
	onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
		r := NewRenderer()
		defer func() { _ = r.Close() }()
		if _, err := r.Attach(h); err != nil {
			return err
		}
		got, err := renderGPU(r, scene, sc.Width, sc.Height)
		if err != nil {
			return err
		}
		dumpImage(t, name+"-gpu", got)
		dumpImage(t, name+"-soft", want)
		check(t, got, want, r)
		return nil
	})
}

// cellNames are the cells of the shapes-clips scene, by grid index.
var cellNames = []string{
	"rect", "rrect", "superellipse", "oval", "circle stroke", "drrect", "arc", "lines and points",
	"vertices", "nine-patch", "image quality", "clipped image",
}

// tolerance is how far the GPU may be from the software renderer in a region: of
// its pixels, at most over8Permille differ by more than 8 levels (the edges, where
// the distance field and the exact area differ a little), at most over32Permille
// by more than 32, and at most over96 pixels by more than 96.
type tolerance struct{ over8Permille, over32Permille, over96 int }

func (tol tolerance) check(t *testing.T, label string, s diffStats) {
	t.Helper()
	t.Logf("%-28s differs: >8 %5d  >32 %4d  >96 %3d  max %3d of %d pixels", label, s.over8, s.over32, s.over96, s.max, s.pixels)
	if s.over8*1000 > tol.over8Permille*s.pixels || s.over32*1000 > tol.over32Permille*s.pixels || s.over96 > tol.over96 {
		t.Errorf("%s: the GPU image is too far from the software image", label)
	}
}

func TestDisplayShapesMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "shapes-clips", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		for i, name := range cellNames {
			cell := image.Rect(i%4*160, i/4*160, i%4*160+160, i/4*160+160)
			tolerance{over8Permille: 30, over32Permille: 3}.check(t, name, diffRect(got, want, cell))
		}
	})
}

func TestDisplayBasicShapesMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "shapes-basic", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		tolerance{over8Permille: 20, over32Permille: 5}.check(t, "shapes-basic", diffRect(got, want, got.Bounds()))
	})
}

func fillP(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func strokeP(argb uint32, width float64, cap ui.StrokeCap, join ui.StrokeJoin) *ui.Paint {
	p := fillP(argb)
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	p.SetStrokeCap(cap)
	p.SetStrokeJoin(join)
	return p
}

func canvasScene(w, h int, fn func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	fn(c)
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build()
}

func TestDisplayShapeScenesMatchSoftware(t *testing.T) {
	r10 := ui.NewRectFromLTWH(10, 10, 60, 40)
	tests := []struct {
		name string
		tol  tolerance
		draw func(c *ui.Canvas)
	}{
		{"rotated rect and rrect", tolerance{25, 3, 0}, func(c *ui.Canvas) {
			c.Save()
			c.Translate(60, 60)
			c.Rotate(0.5)
			c.DrawRect(ui.NewRectFromLTWH(-40, -25, 80, 50), fillP(0xFF2060C0))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(-30, -15, 60, 30), 10, 6), fillP(0xCCF0A000))
			c.Restore()
		}},
		{"non-uniform scale and skew", tolerance{30, 4, 0}, func(c *ui.Canvas) {
			c.Save()
			c.Translate(20, 20)
			c.Scale(1.6, 0.7)
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 30}, 25, strokeP(0xFF802080, 6, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawOval(ui.NewRectFromLTWH(10, 70, 60, 30), fillP(0xFF20A060))
			c.Restore()
			c.Save()
			c.Translate(20, 90)
			c.Skew(0.4, 0)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 70, 30), 8, 8), fillP(0xFFC04020))
			c.Restore()
		}},
		{"mirrored", tolerance{20, 2, 0}, func(c *ui.Canvas) {
			c.Translate(120, 0)
			c.Scale(-1, 1)
			c.DrawRRect(ui.NewRRectFromRectAndCorners(ui.NewRectFromLTWH(10, 10, 70, 50), ui.Corners{TopLeft: ui.Radius{X: 20, Y: 20}, BottomRight: ui.Radius{X: 12, Y: 6}}), fillP(0xFF3050A0))
		}},
		{"fractional anti-aliased clip", tolerance{20, 2, 0}, func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(10.3, 10.6, 60.4, 50.5))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), fillP(0xFF204080))
			c.DrawCircle(ui.Offset{Dx: 70, Dy: 60}, 30, fillP(0xCCFF8000))
		}},
		{"hard clip and nested clips", tolerance{20, 2, 0}, func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(10.3, 10.6, 60.4, 50.5), ui.ClipRectOpts{DoAntiAlias: ui.Ptr(false)})
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 128, 128), fillP(0xFF204080))
			c.Restore()
			c.Translate(5, 70)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 80, 40))
			c.ClipRect(ui.NewRectFromLTWH(20.5, 10.5, 80, 40))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 120, 60), 20, 20), fillP(0xFFA02060))
		}},
		{"translucent overlaps and blend modes", tolerance{10, 1, 0}, func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 50), fillP(0x802060C0))
			c.DrawRect(ui.NewRectFromLTWH(35, 35, 50, 50), fillP(0x80C06020))
			dst := fillP(0xFF30A030)
			dst.SetBlendMode(ui.BlendModeDstOver)
			c.DrawRect(ui.NewRectFromLTWH(0, 80, 128, 20), dst)
			plus := fillP(0x80402010)
			plus.SetBlendMode(ui.BlendModePlus)
			c.DrawCircle(ui.Offset{Dx: 95, Dy: 35}, 25, plus)
		}},
		// At the corners of strokes of a pixel or less the distance field misses the
		// union of two strips, and the software renderer counts both: a few pixels are
		// far from each other, neither being exact.
		{"strokes of every width", tolerance{25, 4, 20}, func(c *ui.Canvas) {
			for i, w := range []float64{0, 0.4, 1, 2.5, 6} {
				x := 8 + float64(i)*24
				c.DrawRect(ui.NewRectFromLTWH(x, 8, 18, 30), strokeP(0xFF000000, w, ui.StrokeCapButt, ui.StrokeJoinMiter))
				c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(x, 46, 18, 30), 6, 6), strokeP(0xFF1030A0, w, ui.StrokeCapButt, ui.StrokeJoinMiter))
				c.DrawRect(ui.NewRectFromLTWH(x, 84, 18, 30), strokeP(0xFFA03010, w, ui.StrokeCapButt, ui.StrokeJoinRound))
			}
		}},
		{"stroke with a radius smaller than the width", tolerance{25, 3, 0}, func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(15, 15, 90, 40), 3, 3), strokeP(0xFF108040, 12, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawOval(ui.NewRectFromLTWH(15, 70, 90, 40), strokeP(0xFF8010A0, 5, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"radii that do not fit", tolerance{20, 2, 0}, func(c *ui.Canvas) {
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 100, 30), 60, 40), fillP(0xFF206080))
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 50, 30, 70), 40, 40), fillP(0xFF806020))
		}},
		{"lines, caps and points", tolerance{30, 3, 0}, func(c *ui.Canvas) {
			caps := []ui.StrokeCap{ui.StrokeCapButt, ui.StrokeCapRound, ui.StrokeCapSquare}
			for i, cp := range caps {
				y := 20 + float64(i)*30
				c.DrawLine(ui.Offset{Dx: 15, Dy: y}, ui.Offset{Dx: 70, Dy: y + 12}, strokeP(0xFF2040A0, 12, cp, ui.StrokeJoinMiter))
				c.DrawLine(ui.Offset{Dx: 90, Dy: y}, ui.Offset{Dx: 90, Dy: y}, strokeP(0xFFA04020, 14, cp, ui.StrokeJoinMiter))
			}
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 20, Dy: 110}, {Dx: 40, Dy: 112}}, strokeP(0xFF008060, 9, ui.StrokeCapRound, ui.StrokeJoinMiter))
			c.DrawPoints(ui.PointModePoints, []ui.Offset{{Dx: 60, Dy: 110}}, strokeP(0xFF008060, 9, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawPoints(ui.PointModeLines, []ui.Offset{{Dx: 80, Dy: 105}, {Dx: 110, Dy: 118}, {Dx: 80, Dy: 118}, {Dx: 110, Dy: 105}}, strokeP(0xFF606060, 2, ui.StrokeCapRound, ui.StrokeJoinMiter))
		}},
		{"colors, paint and tiny shapes", tolerance{20, 2, 0}, func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(r10)
			c.DrawColor(ui.NewColor(0x80FF0000), ui.BlendModeSrcOver)
			c.Restore()
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(80, 10, 30.5, 40.5))
			c.DrawPaint(fillP(0x9900A000))
			c.Restore()
			c.DrawCircle(ui.Offset{Dx: 30, Dy: 90}, 0.5, fillP(0xFF000000))
			c.DrawCircle(ui.Offset{Dx: 50, Dy: 90}, 1.5, fillP(0xFF000000))
			c.DrawRect(ui.NewRectFromLTWH(70.25, 80.25, 0.5, 0.5), fillP(0xFF000000))
		}},
		// Without anti-aliasing a pixel is all or nothing; where the exact coverage is
		// close to half the two renderers may choose differently.
		{"no anti-aliasing", tolerance{20, 20, 8}, func(c *ui.Canvas) {
			p := fillP(0xFF2040C0)
			p.SetIsAntiAlias(false)
			c.DrawCircle(ui.Offset{Dx: 40, Dy: 40}, 30, p)
			c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(60.4, 60.4, 50, 50), 12, 12), p)
			q := strokeP(0xFFC04020, 5, ui.StrokeCapRound, ui.StrokeJoinMiter)
			q.SetIsAntiAlias(false)
			c.DrawLine(ui.Offset{Dx: 10, Dy: 110}, ui.Offset{Dx: 50, Dy: 80}, q)
		}},
		{"superellipse corners", tolerance{30, 3, 0}, func(c *ui.Canvas) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(ui.NewRectFromLTWH(10, 10, 100, 100), 30, 30), fillP(0xFF006A6A))
			c.Translate(64, 64)
			c.Rotate(0.3)
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectAndCorners(ui.NewRectFromLTWH(-30, -20, 60, 40), ui.Corners{TopLeft: ui.Radius{X: 16, Y: 16}, BottomRight: ui.Radius{X: 8, Y: 12}}), fillP(0xAAFFC000))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := canvasScene(128, 128, tt.draw)
			want, err := soft.Render(scene, 128, 128)
			if err != nil {
				t.Fatal(err)
			}
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				if _, err := r.Attach(h); err != nil {
					return err
				}
				got, err := renderGPU(r, scene, 128, 128)
				if err != nil {
					return err
				}
				slug := strings.ReplaceAll(tt.name, " ", "-")
				dumpImage(t, slug+"-gpu", got)
				dumpImage(t, slug+"-soft", want)
				tt.tol.check(t, tt.name, diffRect(got, want, got.Bounds()))
				return nil
			})
		})
	}
}

// pattern is an n x n image of a color gradient with a grid and a transparent
// corner, to show orientation, filtering and alpha.
func pattern(n int) *ui.Image {
	pix := make([]byte, 4*n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			r, g, b, a := byte(x*255/(n-1)), byte(y*255/(n-1)), byte(255-x*255/(n-1)), byte(255)
			if x%8 == 0 || y%8 == 0 {
				r, g, b = 255, 255, 255
			}
			if x < n/4 && y < n/4 {
				a = 128
			}
			// Premultiplied.
			f := float64(a) / 255
			copy(pix[4*(y*n+x):], []byte{byte(float64(r) * f), byte(float64(g) * f), byte(float64(b) * f), a})
		}
	}
	return ui.NewImageFromRGBA(pix, n, n)
}

func starPath() *ui.Path {
	p := ui.NewPath()
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		x, y := 64+50*math.Cos(a), 64+50*math.Sin(a)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

func TestDisplayPathsClipsAndImagesMatchSoftware(t *testing.T) {
	img := pattern(32)
	quality := func(q ui.FilterQuality) *ui.Paint {
		p := ui.NewPaint()
		p.SetFilterQuality(q)
		return p
	}
	curve := func() *ui.Path {
		p := ui.NewPath()
		p.MoveTo(10, 100)
		p.CubicTo(30, 10, 60, 120, 90, 30)
		p.QuadraticBezierTo(110, 5, 120, 60)
		return p
	}
	tests := []struct {
		name string
		tol  tolerance
		draw func(c *ui.Canvas)
	}{
		{"star, both fill rules", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			evenOdd := starPath()
			evenOdd.SetFillType(ui.PathFillTypeEvenOdd)
			c.Save()
			c.Scale(0.5, 0.5)
			c.DrawPath(starPath(), fillP(0xFF8040A0))
			c.Translate(128, 0)
			c.DrawPath(evenOdd, fillP(0xFF40A080))
			c.Restore()
		}},
		{"stroked curves with every join and cap", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			for i, cp := range []ui.StrokeCap{ui.StrokeCapButt, ui.StrokeCapRound, ui.StrokeCapSquare} {
				c.Save()
				c.Translate(0, float64(i)*40-30)
				c.Scale(1, 0.4)
				c.DrawPath(curve(), strokeP(0xFF2040A0, 9, cp, []ui.StrokeJoin{ui.StrokeJoinMiter, ui.StrokeJoinRound, ui.StrokeJoinBevel}[i]))
				c.Restore()
			}
		}},
		{"bevel and miter limit fall back to the atlas", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			p := strokeP(0xFF108040, 12, ui.StrokeCapButt, ui.StrokeJoinBevel)
			c.DrawRect(ui.NewRectFromLTWH(15, 15, 40, 30), p)
			q := strokeP(0xFF801040, 12, ui.StrokeCapButt, ui.StrokeJoinMiter)
			q.SetStrokeMiterLimit(1.1)
			c.DrawRect(ui.NewRectFromLTWH(70, 15, 40, 30), q)
			c.DrawDRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(15, 70, 45, 45), 10, 10), ui.NewRRectFromRectXY(ui.NewRectFromLTWH(25, 80, 25, 25), 4, 4), strokeP(0xFF404040, 3, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.DrawRSuperellipse(ui.NewRSuperellipseFromRectXY(ui.NewRectFromLTWH(70, 70, 45, 45), 12, 12), strokeP(0xFF804000, 4, ui.StrokeCapButt, ui.StrokeJoinMiter))
		}},
		{"polygon points and arcs", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			c.DrawPoints(ui.PointModePolygon, []ui.Offset{{Dx: 10, Dy: 100}, {Dx: 40, Dy: 20}, {Dx: 70, Dy: 90}, {Dx: 118, Dy: 30}}, strokeP(0xFF204080, 7, ui.StrokeCapRound, ui.StrokeJoinRound))
			c.DrawArc(ui.NewRectFromLTWH(20, 30, 90, 80), 0.2, 4.5, false, strokeP(0xFFC04020, 5, ui.StrokeCapRound, ui.StrokeJoinMiter))
			c.DrawArc(ui.NewRectFromLTWH(40, 50, 50, 40), -1, 3, true, fillP(0x99208040))
		}},
		{"hairlines, thin strokes and no anti-aliasing", tolerance{10, 4, 4}, func(c *ui.Canvas) {
			c.DrawPath(curve(), strokeP(0xFF000000, 0, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.Translate(0, 12)
			c.DrawPath(curve(), strokeP(0xFF000000, 0.4, ui.StrokeCapButt, ui.StrokeJoinMiter))
			c.Translate(0, 12)
			p := strokeP(0xFF2020A0, 3, ui.StrokeCapButt, ui.StrokeJoinMiter)
			p.SetIsAntiAlias(false)
			c.DrawPath(curve(), p)
		}},
		{"a path larger than a tile, partly off the surface", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			p := ui.NewPath()
			p.MoveTo(-300, 20)
			p.CubicTo(0, 400, 200, -300, 500, 100)
			p.LineTo(500, 600)
			p.LineTo(-300, 600)
			p.Close()
			c.DrawPath(p, fillP(0xCC2060A0))
		}},
		{"transformed paths", tolerance{4, 1, 0}, func(c *ui.Canvas) {
			c.Translate(64.37, 60.81)
			c.Rotate(0.5)
			c.Scale(0.8, 0.5)
			c.Translate(-64, -64)
			c.DrawPath(starPath(), fillP(0xFFA04020))
			c.DrawPath(curve(), strokeP(0xFF000000, 6, ui.StrokeCapRound, ui.StrokeJoinRound))
		}},
		{"rounded rectangle, superellipse and path clips", tolerance{4, 1, 0}, func(c *ui.Canvas) {
			stripes := func() {
				for i := 0; i < 8; i++ {
					c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16, 128, 8), fillP(0xFF3050C0))
					c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16+8, 128, 8), fillP(0xFFF0A020))
				}
			}
			c.Save()
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(5.5, 5.5, 50, 50), 14, 14))
			stripes()
			c.Restore()
			c.Save()
			c.ClipRSuperellipse(ui.NewRSuperellipseFromRectXY(ui.NewRectFromLTWH(70, 5, 50, 50), 18, 18))
			stripes()
			c.Restore()
			c.Save()
			c.ClipPath(starPath().Shift(ui.Offset{Dx: -40, Dy: 20}))
			stripes()
			c.Restore()
		}},
		{"rotated, difference, hard and nested clips", tolerance{4, 1, 0}, func(c *ui.Canvas) {
			stripes := func() {
				for i := 0; i < 8; i++ {
					c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16, 128, 8), fillP(0xFF3050C0))
					c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16+8, 128, 8), fillP(0xFFF0A020))
				}
			}
			c.Save()
			c.Translate(35, 35)
			c.Rotate(0.5)
			c.ClipRect(ui.NewRectFromLTWH(-25, -20, 50, 40))
			c.Rotate(-0.5)
			c.Translate(-35, -35)
			stripes()
			c.Restore()
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(70, 10, 50, 50), ui.ClipRectOpts{DoAntiAlias: ui.Ptr(true)})
			c.ClipRect(ui.NewRectFromLTWH(85, 25, 20, 20), ui.ClipRectOpts{ClipOp: ui.ClipOpDifference})
			stripes()
			c.Restore()
			c.Save()
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 70, 60, 50), 20, 20), ui.ClipOpts{DoAntiAlias: ui.Ptr(false)})
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(30, 80, 60, 40), 15, 15))
			stripes()
			c.Restore()
		}},
		{"images of every filter quality, up and down", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			for i, q := range []ui.FilterQuality{ui.FilterQualityNone, ui.FilterQualityLow, ui.FilterQualityMedium, ui.FilterQualityHigh} {
				x := float64(i%2) * 64
				y := float64(i/2) * 64
				c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(x+2, y+2, 60, 60), quality(q))
			}
			c.Save()
			c.Translate(5, 5)
			for i, q := range []ui.FilterQuality{ui.FilterQualityNone, ui.FilterQualityLow, ui.FilterQualityMedium, ui.FilterQualityHigh} {
				c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(float64(i)*12, 2, 9.5, 9.5), quality(q))
			}
			c.Restore()
		}},
		{"rotated and scaled images, alpha and blend modes", tolerance{6, 2, 0}, func(c *ui.Canvas) {
			c.Save()
			c.Translate(40, 40)
			c.Rotate(0.4)
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(-25, -25, 50, 50), quality(ui.FilterQualityLow))
			c.Restore()
			faded := quality(ui.FilterQualityLow)
			faded.SetColor(ui.NewColor(0x80FFFFFF))
			c.DrawImage(img, ui.Offset{Dx: 70.5, Dy: 10.25}, faded)
			plus := quality(ui.FilterQualityNone)
			plus.SetBlendMode(ui.BlendModePlus)
			c.DrawImageRect(img, ui.NewRectFromLTWH(8, 8, 40, 40), ui.NewRectFromLTWH(10, 70, 50, 50), plus)
			noAA := quality(ui.FilterQualityLow)
			noAA.SetIsAntiAlias(false)
			c.Save()
			c.Translate(95, 95)
			c.Rotate(-0.3)
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, 32, 32), ui.NewRectFromLTWH(-20, -20, 40, 40), noAA)
			c.Restore()
		}},
		{"nine-patch, squeezed and stretched", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			center := ui.NewRectFromLTWH(8, 8, 16, 16)
			c.DrawImageNine(img, center, ui.NewRectFromLTWH(4, 4, 80, 50), quality(ui.FilterQualityLow))
			c.DrawImageNine(img, center, ui.NewRectFromLTWH(90, 4, 30, 20), quality(ui.FilterQualityNone))
			c.DrawImageNine(img, center, ui.NewRectFromLTWH(4, 70, 100, 40), quality(ui.FilterQualityNone))
		}},
		{"atlas sprites with transforms and colors", tolerance{6, 2, 0}, func(c *ui.Canvas) {
			xf := []ui.RSTransform{
				ui.NewRSTransformFromComponents(0, 1, 0, 0, 30, 30),
				ui.NewRSTransformFromComponents(0.5, 1.5, 8, 8, 90, 30),
				ui.NewRSTransformFromComponents(-0.3, 0.75, 8, 8, 30, 90),
				ui.NewRSTransformFromComponents(0.9, 1, 8, 8, 90, 90),
			}
			rects := []ui.Rect{ui.NewRectFromLTWH(0, 0, 16, 16), ui.NewRectFromLTWH(16, 0, 16, 16), ui.NewRectFromLTWH(0, 16, 16, 16), ui.NewRectFromLTWH(16, 16, 16, 16)}
			c.DrawAtlas(img, xf, rects, nil, ui.BlendModeSrc, nil, quality(ui.FilterQualityLow))
			c.Save()
			c.Translate(0, 4)
			colors := []ui.Color{ui.NewColor(0x80FFFFFF), ui.NewColor(0xFF00FF00), ui.NewColor(0xFFFF0000), ui.NewColor(0x80000000)}
			c.DrawAtlas(img, xf, rects, colors, ui.BlendModeSrcIn, nil, quality(ui.FilterQualityNone))
			c.Restore()
		}},
		{"vertices of every mode", tolerance{3, 1, 0}, func(c *ui.Canvas) {
			colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0x80FFFF00)}
			pos := []ui.Offset{{Dx: 10, Dy: 10}, {Dx: 60, Dy: 15}, {Dx: 20, Dy: 55}, {Dx: 70, Dy: 60}}
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleStrip, pos, ui.VerticesOpts{Colors: colors}), ui.BlendModeDst, fillP(0xFF000000))
			c.Save()
			c.Translate(60, 0)
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos[:3], ui.VerticesOpts{Colors: colors[:3]}), ui.BlendModeModulate, fillP(0xFF80C0FF))
			c.Restore()
			c.Save()
			c.Translate(0, 60)
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangleFan, pos, ui.VerticesOpts{Colors: colors}), ui.BlendModeSrcOver, fillP(0x8000FFFF))
			c.Restore()
			c.Save()
			c.Translate(60, 60)
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, pos, ui.VerticesOpts{Indices: []int{0, 1, 2, 1, 2, 3}}), ui.BlendModeSrc, fillP(0xCC208040))
			c.Restore()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := canvasScene(128, 128, tt.draw)
			want, err := soft.Render(scene, 128, 128)
			if err != nil {
				t.Fatal(err)
			}
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				if _, err := r.Attach(h); err != nil {
					return err
				}
				got, err := renderGPU(r, scene, 128, 128)
				if err != nil {
					return err
				}
				slug := strings.ReplaceAll(strings.ReplaceAll(tt.name, " ", "-"), ",", "")
				dumpImage(t, slug+"-gpu", got)
				dumpImage(t, slug+"-soft", want)
				tt.tol.check(t, tt.name, diffRect(got, want, got.Bounds()))
				return nil
			})
		})
	}
}
