package engine

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// posterConn is a connection that cannot write files, like the web: the engine hands it
// the PNG of the self-capture. The answer comes back through Post, as on the web.
type posterConn struct {
	*conn
	png    []byte
	answer error
}

func (c *posterConn) PostScreenshot(data []byte, done func(error)) {
	c.png = data
	c.Post(func() { done(c.answer) })
}

// lateRenderer is a renderer whose read-back finishes after Draw returned, as WebGPU's does: Draw
// of a capture returns nothing, and the image comes through Config.captured on the UI thread.
type lateRenderer struct {
	*fakeRenderer
	cfg *Config
	err error
}

func (r *lateRenderer) Draw(h gpu.SurfaceHandle, w, ht int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	img, err := r.fakeRenderer.Draw(h, w, ht, scene, capture)
	if capture && err == nil {
		cfg, late := r.cfg, r.err
		cfg.post(func() { cfg.captured(img, late) })
		return nil, nil
	}
	return img, err
}

// runPosterCapture runs the self-capture of a 2x2 frame on a platform that takes the PNG.
func runPosterCapture(t *testing.T, answer error) (*posterConn, error) {
	return runCapture(t, answer, false, nil)
}

// runCapture is runPosterCapture; with late the renderer reads the frame back after Draw returned,
// and readErr is the error of that read.
func runCapture(t *testing.T, answer error, late bool, readErr error) (*posterConn, error) {
	t.Helper()
	// The path is not used: the platform takes the PNG.
	h := newHarness(t, map[string]string{ScreenshotEnv: "1"})
	pc := &posterConn{conn: h.c, answer: answer}
	h.cfg.connect = func() (connection, error) { return pc, nil }
	if late {
		h.cfg.newRenderer = func(cfg *Config, _ *slog.Logger) renderer {
			return &lateRenderer{fakeRenderer: h.r, cfg: cfg, err: readErr}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range 4 {
		img.SetRGBA(i%2, i/2, color.RGBA{0x67, 0x50, 0xA4, 255})
	}
	h.r.img = img
	h.c.Inject(metrics(2, 2))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { return hasEntry(h.c, fake.EntryRequestFrame) })
	h.c.Inject(fake.FrameEvent{View: 0})
	return pc, result(t, done)
}

func TestScreenshotGoesToAPlatformThatTakesIt(t *testing.T) {
	pc, err := runPosterCapture(t, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := png.Decode(bytes.NewReader(pc.png))
	if err != nil {
		t.Fatalf("what the platform got is not a PNG: %v", err)
	}
	if b := got.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Errorf("size = %v, want 2x2", b)
	}
	if r, g, b, _ := got.At(1, 1).RGBA(); r>>8 != 0x67 || g>>8 != 0x50 || b>>8 != 0xA4 {
		t.Errorf("pixel = %x %x %x", r>>8, g>>8, b>>8)
	}
}

// TestScreenshotFromARendererThatReadsBackLater: the frame of the self-capture arrives after Draw
// returned (WebGPU's read-back finishes in a later task); the engine takes it from the renderer's
// callback, hands it to the platform and ends Run, and a read that failed ends Run with its error.
func TestScreenshotFromARendererThatReadsBackLater(t *testing.T) {
	pc, err := runCapture(t, nil, true, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := png.Decode(bytes.NewReader(pc.png))
	if err != nil {
		t.Fatalf("what the platform got is not a PNG: %v", err)
	}
	if b := got.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Errorf("size = %v, want 2x2", b)
	}
	want := errors.New("WebGPUReadFailed", "the browser could not map the buffer of a read-back")
	if _, err := runCapture(t, nil, true, want); err != want {
		t.Fatalf("Run = %v, want the error of the read unchanged", err)
	}
}

func TestScreenshotThatDoesNotArriveEndsRunWithTheError(t *testing.T) {
	want := errors.New("ScreenshotWriteFailed", "the server said no")
	_, err := runPosterCapture(t, want)
	if err != want {
		t.Fatalf("Run = %v, want the error of the platform unchanged", err)
	}
}

func TestEncodeFailsOnAnEmptyImageWithTheRoute(t *testing.T) {
	_, err := (&Screenshot{path: "x"}).Encode(image.NewRGBA(image.Rect(0, 0, 0, 0)))
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "ScreenshotWriteFailed" || !appErr.HasRoute("Screenshot.Encode.ScreenshotWriteFailed") {
		t.Fatalf("Encode = %v, want ScreenshotWriteFailed via Screenshot.Encode", err)
	}
}

// exitlessConn is a connection whose app cannot end itself, like a web page.
type exitlessConn struct{ *conn }

func (exitlessConn) CanExit() bool { return false }

func TestRequestAppExitDoesNothingWhereTheAppCannotExit(t *testing.T) {
	tests := []struct {
		name     string
		conn     func(*conn) connection
		wantExit bool
	}{
		{"a desktop platform", func(c *conn) connection { return c }, true},
		{"a web page", func(c *conn) connection { return exitlessConn{c} }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, nil)
			c := tc.conn(h.c)
			h.cfg.connect = func() (connection, error) { return c, nil }
			var stopped bool
			h.cfg.Start = func(d *ui.PlatformDispatcher) error {
				d.RequestAppExit()
				return nil
			}
			done := h.run(t)
			select {
			case err := <-done:
				stopped = true
				if err != nil {
					t.Fatalf("Run: %v", err)
				}
			case <-time.After(300 * time.Millisecond):
			}
			if stopped != tc.wantExit {
				t.Errorf("Run ended = %v after RequestAppExit, want %v", stopped, tc.wantExit)
			}
			if !stopped {
				h.stopEngine(t)
				if err := result(t, done); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
