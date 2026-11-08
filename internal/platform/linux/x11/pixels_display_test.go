//go:build linux && !android && glutter_display

package x11

import (
	"fmt"
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/shmwindow"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/ui"
)

// These tests run the software renderer over the pixel windows of the X11 backend (SPEC §9.6, step
// 32.2) on a real X server (scripts/headless-x11.sh), and read the window back from the server with
// XGetImage: what the server shows must equal what the renderer drew, frame after frame, with the
// damage of a moving caret only, with MIT-SHM (and its completion events) and with XPutImage.

// getImage is XGetImage, bound for the tests: the pixels that the server shows in a window.
var getImage = func() func(display uintptr, drawable uint64, x, y int32, w, h uint32, planeMask uint64, format int32) uintptr {
	lib, err := dl.Open(libX11, "libX11.so")
	if err != nil {
		return nil
	}
	var f func(display uintptr, drawable uint64, x, y int32, w, h uint32, planeMask uint64, format int32) uintptr
	if err := lib.Bind(&f, "XGetImage"); err != nil {
		return nil
	}
	return f
}()

// moveWindow is XMoveWindow, bound for the tests: the second window of a test moves off the first,
// which it would hide (there is no window manager to place them, and XGetImage returns what is on
// the screen).
var moveWindow = func() func(display uintptr, window uint64, x, y int32) int32 {
	lib, err := dl.Open(libX11, "libX11.so")
	if err != nil {
		return nil
	}
	var f func(display uintptr, window uint64, x, y int32) int32
	if err := lib.Bind(&f, "XMoveWindow"); err != nil {
		return nil
	}
	return f
}()

// pixelEnv is a platform of pixel windows with a renderer over them.
type pixelEnv struct {
	*x11Env
	r *shmwindow.Renderer
	// lost is the count of OnContentLost that the renderer has been told of.
	lost int
}

func startPixelDisplay(t *testing.T, noSHM bool, opts Options) *pixelEnv {
	t.Helper()
	if getImage == nil || moveWindow == nil {
		t.Skip("XGetImage is not available")
	}
	e := startDisplayOptions(t, windowConfig(), opts, func(p *Platform) {
		// A window that shows pixels never asks EGL for a visual: with no GPU that is where Mesa prints
		// its DRI3 warning and starts a software rasterizer for nothing.
		if opts.PixelWindows {
			p.visualFor = func(uintptr) (uint32, error) {
				t.Error("EGL was asked for a visual for a window that shows pixels")
				return 0, &errors.Error{Tag: tagPixelsNoEGL}
			}
		}
		p.getenv = func(k string) string {
			if k == NoSHMEnv && noSHM {
				return "1"
			}
			return ""
		}
	})
	e.waitMapped()
	pe := &pixelEnv{x11Env: e}
	pe.r = shmwindow.NewRenderer(func(h gpu.SurfaceHandle) (gpu.PixelTarget, bool) {
		w := e.p.windows[uint64(h.Window)]
		return w, w != nil
	}, shmwindow.Options{Background: ui.NewColor(0xFFFFFFFF)})
	return pe
}

// tagPixelsNoEGL is the tag of the error that a test gives when it must not be asked for a visual.
const tagPixelsNoEGL = "PixelsNoEGL"

// pixelScene is what a window shows in the tests: a background, bars like lines of text, and a caret at
// a place that changes from frame to frame.
type pixelScene struct {
	back, bars *ui.Picture
	// carets are the pictures of the caret by place: the same place is the same picture, so that a
	// frame that shows the same is a frame that changed nothing.
	carets map[float64]*ui.Picture
}

func newPixelScene(color uint32) *pixelScene {
	return &pixelScene{
		carets: map[float64]*ui.Picture{},
		back: gputest.PictureOf(func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 4000, 4000), gputest.FillPaint(color))
		}),
		bars: gputest.PictureOf(func(c *ui.Canvas) {
			for i := 0; i < 8; i++ {
				c.DrawRect(ui.NewRectFromLTWH(20, 20+float64(i)*30, 300-float64(i)*20, 12), gputest.FillPaint(0xFF3366CC))
			}
		}),
	}
}

func (s *pixelScene) at(caretX float64) *ui.Scene {
	caret, ok := s.carets[caretX]
	if !ok {
		caret = gputest.PictureOf(func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(caretX, 130, 2, 18), gputest.FillPaint(0xFF000000))
		})
		s.carets[caretX] = caret
	}
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, s.back)
	b.AddPicture(ui.Offset{}, s.bars)
	b.AddPicture(ui.Offset{}, caret)
	return b.Build()
}

// recordTarget forwards to the window and records the damage of each presentation.
type recordTarget struct {
	gpu.PixelTarget
	damage *[][]image.Rectangle
}

func (r recordTarget) PresentPixels(d []image.Rectangle) error {
	*r.damage = append(*r.damage, append([]image.Rectangle(nil), d...))
	return r.PixelTarget.PresentPixels(d)
}

// serverPixels reads the window back from the server: width x height pixels, as RGBA.
func serverPixels(t *testing.T, p *Platform, w *window, width, height int) *image.RGBA {
	t.Helper()
	img := getImage(p.display, w.xid, 0, 0, uint32(width), uint32(height), ^uint64(0), zPixmap)
	if img == 0 {
		t.Fatal("XGetImage failed")
	}
	defer p.x.destroyImage(img)
	im := view[xImage](img)
	if !pixelLayoutOK(im) {
		t.Fatalf("the server gave an image of depth %d with %d bits a pixel", im.depth, im.bitsPerPixel)
	}
	data := dl.Bytes(im.data, int(im.bytesPerLine)*height)
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := data[y*int(im.bytesPerLine):]
		for x := 0; x < width; x++ {
			o := out.PixOffset(x, y)
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = row[4*x+2], row[4*x+1], row[4*x], 0xFF
		}
	}
	return out
}

// firstDifference describes the first pixel in which two images differ, or "" when they are equal.
func firstDifference(a, b *image.RGBA) string {
	if a.Rect != b.Rect {
		return fmt.Sprintf("sizes %v and %v", a.Rect, b.Rect)
	}
	for y := a.Rect.Min.Y; y < a.Rect.Max.Y; y++ {
		for x := a.Rect.Min.X; x < a.Rect.Max.X; x++ {
			o := a.PixOffset(x, y)
			if a.Pix[o] != b.Pix[o] || a.Pix[o+1] != b.Pix[o+1] || a.Pix[o+2] != b.Pix[o+2] {
				return fmt.Sprintf("pixel %d,%d is %v on the server, %v when drawn", x, y, a.Pix[o:o+3], b.Pix[o:o+3])
			}
		}
	}
	return ""
}

// finish destroys the surfaces and closes the renderer, on the UI thread.
func (e *pixelEnv) finish(surfaces ...gpu.Surface) {
	e.t.Helper()
	e.onLoop(func() {
		for _, s := range surfaces {
			s.Destroy()
		}
		if err := e.r.Close(); err != nil {
			e.t.Errorf("Close: %v", err)
		}
	})
}

// render draws the scene into the window, with the content-lost rule of the engine (a window that
// the server lost the content of is drawn whole), and checks what the server shows.
func (e *pixelEnv) render(w *window, surface gpu.Surface, scene *ui.Scene) {
	e.t.Helper()
	e.onLoop(func() {
		if lost := e.h.contentLost(); lost != e.lost {
			e.lost = lost
			e.r.Invalidate(surface)
		}
		size := ui.Size{Width: float64(w.width), Height: float64(w.height)}
		if err := e.r.Render(surface, scene, size, 1); err != nil {
			e.t.Errorf("Render: %v", err)
			return
		}
		want, err := e.r.Snapshot(scene, size)
		if err != nil {
			e.t.Errorf("Snapshot: %v", err)
			return
		}
		if diff := firstDifference(serverPixels(e.t, e.p, w, int(w.width), int(w.height)), want); diff != "" {
			e.t.Errorf("the server does not show what was drawn: %s", diff)
		}
	})
	// Let the loop dispatch the events that came meanwhile: the completions of MIT-SHM.
	e.onLoop(func() {})
}

func TestDisplayPixelWindowsShowWhatTheRendererDrew(t *testing.T) {
	for _, mode := range []struct {
		name  string
		noSHM bool
		path  string
	}{{"MIT-SHM", false, "MIT-SHM"}, {"XPutImage", true, "XPutImage"}} {
		t.Run(mode.name, func(t *testing.T) {
			e := startPixelDisplay(t, mode.noSHM, Options{PixelWindows: true})
			w := e.win.(*window)
			var damage [][]image.Rectangle
			var surface gpu.Surface
			e.onLoop(func() {
				// The renderer draws into the window through a target that records the damage.
				e.r = shmwindow.NewRenderer(func(h gpu.SurfaceHandle) (gpu.PixelTarget, bool) {
					return recordTarget{PixelTarget: w, damage: &damage}, true
				}, shmwindow.Options{Background: ui.NewColor(0xFFFFFFFF)})
				var err error
				if surface, err = e.r.Attach(w.NativeSurface()); err != nil {
					t.Fatal(err)
				}
			})
			scene := newPixelScene(0xFFFAFAFA)
			e.onLoop(func() {
				if w.depth != 24 || w.visual == 0 || w.colormap == 0 {
					t.Errorf("the window has visual %#x of depth %d with colormap %#x, want a 24-bit visual of its own choice", w.visual, w.depth, w.colormap)
				}
			})

			// The first frame draws the whole window.
			e.render(w, surface, scene.at(20))
			if got := w.PixelPath(); got != mode.path {
				t.Errorf("the pixels go with %q, want %q", got, mode.path)
			}
			whole := image.Rect(0, 0, int(w.width), int(w.height))
			if len(damage) != 1 || len(damage[0]) != 1 || damage[0][0] != whole {
				t.Errorf("the first frame reported %v, want the whole window %v", damage, whole)
			}

			// A caret that moves reports its old and new places and nothing else, and the server shows
			// the whole picture every time: the buffers that the pool gives back are made right by the
			// damage of the frames since they were shown (buffer age).
			damage = nil
			for i := 1; i <= 14; i++ {
				e.render(w, surface, scene.at(20+float64(i)*24))
			}
			if len(damage) != 14 {
				t.Fatalf("%d presentations for 14 frames", len(damage))
			}
			for i, d := range damage {
				area := 0
				for _, r := range d {
					area += r.Dx() * r.Dy()
				}
				if area == 0 || area > 4*64*64 {
					t.Errorf("frame %d reported %v (%d pixels): more than the caret's old and new tiles", i+1, d, area)
				}
			}
			if mode.path == "MIT-SHM" && len(e.p.shmSegs) > 4 {
				t.Errorf("%d shared segments alive after 15 frames: the completion events do not give the buffers back", len(e.p.shmSegs))
			}

			// A frame that changes nothing sends nothing.
			damage = nil
			e.render(w, surface, scene.at(20+14*24))
			if len(damage) != 1 || len(damage[0]) != 0 {
				t.Errorf("a frame with no change reported %v, want an empty presentation", damage)
			}

			// A resize: the window gets the new size, and the next frame is drawn whole at it.
			e.onLoop(func() { w.SetSize(ui.Size{Width: 560, Height: 380}) })
			waitFor(t, func() bool {
				var ok bool
				e.onLoop(func() { ok = w.width == 560 && w.height == 380 })
				return ok
			}, "the window was not resized")
			damage = nil
			e.render(w, surface, scene.at(100))
			if len(damage) != 1 || len(damage[0]) != 1 || damage[0][0] != image.Rect(0, 0, 560, 380) {
				t.Errorf("the first frame after the resize reported %v, want the whole window", damage)
			}
			e.render(w, surface, scene.at(130))

			// A window that is withdrawn and shown again has lost its content: the next frame is
			// drawn whole.
			before := e.h.contentLost()
			e.onLoop(func() { w.SetHidden(true) })
			e.onLoop(func() { w.SetHidden(false) })
			waitFor(t, func() bool { return e.h.contentLost() > before }, "the window got no expose after it was shown again")
			e.render(w, surface, scene.at(160))

			e.finish(surface)
		})
	}
}

func TestDisplayTwoPixelWindowsHaveBuffersOfTheirOwn(t *testing.T) {
	e := startPixelDisplay(t, false, Options{PixelWindows: true})
	first := e.win.(*window)
	var second *window
	var s1, s2 gpu.Surface
	e.onLoop(func() {
		cfg := windowConfig()
		cfg.Size = ui.Size{Width: 300, Height: 200}
		win, err := e.p.NewWindow(cfg)
		if err != nil {
			t.Fatal(err)
		}
		second = win.(*window)
		moveWindow(e.p.display, second.xid, 700, 400)
		e.p.x.flush(e.p.display)
	})
	waitFor(t, func() bool {
		var ok bool
		e.onLoop(func() { ok = second.mapped })
		return ok
	}, "the second window was not mapped")
	e.onLoop(func() {
		var err error
		if s1, err = e.r.Attach(first.NativeSurface()); err != nil {
			t.Fatal(err)
		}
		if s2, err = e.r.Attach(second.NativeSurface()); err != nil {
			t.Fatal(err)
		}
	})
	one, two := newPixelScene(0xFFFAFAFA), newPixelScene(0xFFE0F0E0)
	for i := 0; i < 4; i++ {
		e.render(first, s1, one.at(20+float64(i)*30))
		e.render(second, s2, two.at(40+float64(i)*20))
	}
	e.onLoop(func() {
		if first.pix == nil || second.pix == nil || first.pix.pool == second.pix.pool || first.pix.gc == second.pix.gc {
			t.Error("two windows share a pool or a graphics context")
		}
		before := len(e.p.shmSegs)
		second.Close()
		if after := len(e.p.shmSegs); after >= before || second.pix != nil {
			t.Errorf("closing a window left its %d segments registered (%d before)", after, before)
		}
	})
	e.render(first, s1, one.at(300)) // the other window goes on
	e.finish(s1, s2)
}

// TestDisplayPixelsOnTheVisualThatEGLChose is the window of an app whose renderer is chosen at the
// first frame (step 32.3 makes auto choose soft with no GPU): it was made with the visual that EGL
// wants, and shows pixels when that visual's layout allows, or the first frame says it does not.
func TestDisplayPixelsOnTheVisualThatEGLChose(t *testing.T) {
	e := startPixelDisplay(t, false, Options{})
	w := e.win.(*window)
	var surface gpu.Surface
	e.onLoop(func() {
		var err error
		if surface, err = e.r.Attach(w.NativeSurface()); err != nil {
			t.Fatal(err)
		}
		t.Logf("the window has visual %#x of depth %d", w.visual, w.depth)
	})
	scene := newPixelScene(0xFFFAFAFA)
	var failed error
	e.onLoop(func() {
		size := ui.Size{Width: float64(w.width), Height: float64(w.height)}
		failed = e.r.Render(surface, scene.at(20), size, 1)
	})
	if failed != nil {
		appErr, ok := errors.FromError(failed)
		if !ok || appErr.Tag != tagX11PixelFormatUnsupported {
			t.Fatalf("Render: %v", failed)
		}
		t.Skipf("the visual of EGL is not one that shows pixels: %v", appErr.Metadata)
	}
	e.render(w, surface, scene.at(60))
	e.finish(surface)
}
