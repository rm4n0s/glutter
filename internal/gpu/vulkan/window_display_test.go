//go:build linux && glutter_display

package vulkan

import (
	"context"
	"image"
	"image/color"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland"
	"github.com/rm4n0s/glutter/internal/platform/linux/x11"
	"github.com/rm4n0s/glutter/ui"
)

// These tests open real windows and present through Vulkan on lavapipe, with validation
// on. Run them under the headless compositor and server:
//
//	scripts/headless-wayland.sh -- go test -tags glutter_display ./internal/gpu/vulkan/
//	scripts/headless-x11.sh -- go test -tags glutter_display ./internal/gpu/vulkan/

// recDevice wraps the Vulkan device to record the size of every swapchain image that a
// frame draws into.
type recDevice struct {
	*Device
	mu       sync.Mutex
	acquired []image.Point
	opened   int
}

func (d *recDevice) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	sc, err := d.Device.NewSwapchain(h)
	if err != nil {
		return nil, err
	}
	return &recSwapchain{Swapchain: sc, d: d}, nil
}

type recSwapchain struct {
	explicit.Swapchain
	d *recDevice
}

func (s *recSwapchain) Acquire() (explicit.Texture, error) {
	t, err := s.Swapchain.Acquire()
	if err == nil {
		s.d.mu.Lock()
		s.d.acquired = append(s.d.acquired, image.Pt(t.Desc().Width, t.Desc().Height))
		s.d.mu.Unlock()
	}
	return t, err
}

func (d *recDevice) lastAcquired() image.Point {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.acquired) == 0 {
		return image.Point{}
	}
	return d.acquired[len(d.acquired)-1]
}

// newWindowRenderer makes the explicit renderer over a Vulkan device on lavapipe, with
// validation; the device is opened at the first frame, from the first window.
func newWindowRenderer(t *testing.T) (*explicit.Renderer, *recDevice) {
	t.Helper()
	lavapipe(t)
	dev := &recDevice{}
	r := explicit.NewRenderer(func(h gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := Open(Options{AllowCPU: true, Validation: true, Surface: h, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
		if err != nil {
			return nil, err
		}
		dev.Device = d
		dev.opened++
		return dev, nil
	}, explicit.RendererOpts{Background: ui.NewColor(0xFF6750A4), HasBackground: true})
	t.Cleanup(func() {
		r.Close()
	})
	return r, dev
}

// checkValidation closes the renderer, which destroys the swapchains and the device (the
// path of shutdown), and fails the test if the validation layer reported anything.
func checkValidation(t *testing.T, r *explicit.Renderer, dev *recDevice) {
	t.Helper()
	r.Close()
	if dev.Device == nil {
		return
	}
	if errs := dev.ValidationErrors(); len(errs) > 0 {
		t.Errorf("the validation layer reported %d errors:\n%v", len(errs), errs)
	}
}

// windowHandler draws each frame of its windows with the renderer, and records what it
// drew.
type windowHandler struct {
	platform.NopHandler
	r    *explicit.Renderer
	dev  *recDevice
	wins map[ui.ViewID]platform.Window

	mu      sync.Mutex
	sizes   map[ui.ViewID]image.Point // the latest size the platform reported
	frames  map[ui.ViewID][]frame
	capture bool
	errs    []error
}

type frame struct {
	metrics  image.Point // the size the platform said the window has
	presents image.Point // the size of the swapchain image the frame was drawn into
	shot     *image.RGBA
}

func newWindowHandler(r *explicit.Renderer, dev *recDevice, capture bool, wins ...platform.Window) *windowHandler {
	h := &windowHandler{r: r, dev: dev, wins: map[ui.ViewID]platform.Window{}, sizes: map[ui.ViewID]image.Point{}, frames: map[ui.ViewID][]frame{}, capture: capture}
	for _, w := range wins {
		h.wins[w.ID()] = w
	}
	return h
}

func (h *windowHandler) OnMetrics(v ui.ViewID, m platform.ViewMetrics) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sizes[v] = image.Pt(int(m.PhysicalSize.Width), int(m.PhysicalSize.Height))
}

func (h *windowHandler) OnFrame(v ui.ViewID, _ time.Duration) {
	h.mu.Lock()
	size := h.sizes[v]
	h.mu.Unlock()
	win, ok := h.wins[v]
	if !ok || size.X <= 0 || size.Y <= 0 {
		return
	}
	shot, err := h.r.Draw(win.NativeSurface(), size.X, size.Y, nil, h.capture)
	h.mu.Lock()
	defer h.mu.Unlock()
	if err != nil {
		h.errs = append(h.errs, err)
		return
	}
	h.frames[v] = append(h.frames[v], frame{metrics: size, presents: h.dev.lastAcquired(), shot: shot})
}

func (h *windowHandler) framesOf(v ui.ViewID) []frame {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]frame(nil), h.frames[v]...)
}

func (h *windowHandler) failed() []error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]error(nil), h.errs...)
}

// waitUntil polls the condition for up to 20 seconds.
func waitUntil(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// runPlatform runs the platform's loop and returns a function that stops it.
func runPlatform(t *testing.T, p platform.Platform, h platform.Handler) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	stopped := false
	stop = func() {
		if stopped {
			return
		}
		stopped = true
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run: %v", err)
		}
	}
	t.Cleanup(stop)
	return stop
}

func TestWaylandWindowPresentsItsBackground(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	// The renderer is closed before the platform, as the engine does: its swapchains
	// draw into the windows.
	r, dev := newWindowRenderer(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "vulkan display test"
	cfg.Size = ui.Size{Width: 200, Height: 100}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := newWindowHandler(r, dev, true, win)
	stop := runPlatform(t, p, h)
	waitUntil(t, func() bool { return len(h.framesOf(win.ID())) > 0 || len(h.failed()) > 0 }, "no frame was presented")
	stop()
	if errs := h.failed(); len(errs) > 0 {
		t.Fatalf("a frame failed: %v", errs[0])
	}
	f := h.framesOf(win.ID())[0]
	if f.shot == nil || f.shot.Rect.Dx() != f.metrics.X || f.shot.Rect.Dy() != f.metrics.Y {
		t.Fatalf("capture = %v for a window of %v", f.shot, f.metrics)
	}
	for _, p := range []image.Point{{0, 0}, {f.metrics.X - 1, f.metrics.Y - 1}, {f.metrics.X / 2, f.metrics.Y / 2}} {
		if got := f.shot.RGBAAt(p.X, p.Y); got != (color.RGBA{0x67, 0x50, 0xA4, 0xFF}) {
			t.Errorf("pixel %v = %v, want #6750A4", p, got)
		}
	}
	if f.presents != f.metrics {
		t.Errorf("the swapchain image is %v for a window of %v", f.presents, f.metrics)
	}
	if dev.opened != 1 || r.Describe() == "" {
		t.Errorf("devices opened %d, description %q", dev.opened, r.Describe())
	}
	checkValidation(t, r, dev)
}

// TestWaylandResizeNeverPresentsTheOldSize is the resize test of step 3.7 with the Vulkan
// renderer: a floating window is resized through sway's IPC, and each frame is drawn
// into a swapchain image of the size the platform reported for it, never of the one
// before (SPEC §8.11: ack_configure, then the swapchain at the new size, then the
// present).
func TestWaylandResizeNeverPresentsTheOldSize(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		t.Skip("this test resizes the window through sway (run under scripts/headless-wayland.sh)")
	}
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	// The renderer is closed before the platform, as the engine does: its swapchains
	// draw into the windows.
	r, dev := newWindowRenderer(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter vulkan resize test"
	cfg.AppID = "dev.glutter.vkresizetest"
	cfg.Size = ui.Size{Width: 400, Height: 300}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := newWindowHandler(r, dev, false, win)
	stop := runPlatform(t, p, h)
	waitUntil(t, func() bool { return len(h.framesOf(win.ID())) > 0 || len(h.failed()) > 0 }, "no frame was presented")
	seen := map[image.Point]bool{}
	for _, size := range [][2]int{{500, 350}, {321, 201}, {640, 480}, {400, 300}, {350, 260}} {
		before := len(h.framesOf(win.ID()))
		out, err := exec.Command("swaymsg", "[app_id=dev.glutter.vkresizetest]", "resize", "set", "width", strconv.Itoa(size[0]), "px", "height", strconv.Itoa(size[1]), "px").CombinedOutput()
		if err != nil {
			t.Fatalf("swaymsg: %v: %s", err, out)
		}
		waitUntil(t, func() bool {
			fs := h.framesOf(win.ID())
			return len(fs) > before && fs[len(fs)-1].metrics != fs[before-1].metrics
		}, "the window did not present a frame of a new size after resizing")
		// Let the frames of this size settle.
		time.Sleep(50 * time.Millisecond)
	}
	stop()
	if errs := h.failed(); len(errs) > 0 {
		t.Fatalf("a frame failed: %v", errs[0])
	}
	for i, f := range h.framesOf(win.ID()) {
		seen[f.metrics] = true
		if f.presents != f.metrics {
			t.Errorf("frame %d was drawn into a swapchain image of %v, but the window had %v", i, f.presents, f.metrics)
		}
	}
	if len(seen) < 4 {
		t.Errorf("only %d sizes were presented: %v", len(seen), seen)
	}
	checkValidation(t, r, dev)
}

func TestWaylandTwoWindowsShareTheDevice(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("there is no Wayland compositor (run under scripts/headless-wayland.sh)")
	}
	p, err := wayland.New("")
	if err != nil {
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	// The renderer is closed before the platform, as the engine does: its swapchains
	// draw into the windows.
	r, dev := newWindowRenderer(t)
	var wins []platform.Window
	for i, size := range []ui.Size{{Width: 200, Height: 120}, {Width: 260, Height: 140}} {
		cfg := platform.DefaultWindowConfig()
		cfg.Title = "vulkan window " + strconv.Itoa(i)
		cfg.Size = size
		w, err := p.NewWindow(cfg)
		if err != nil {
			t.Fatal(err)
		}
		wins = append(wins, w)
	}
	h := newWindowHandler(r, dev, true, wins...)
	stop := runPlatform(t, p, h)
	waitUntil(t, func() bool {
		return len(h.framesOf(wins[0].ID())) > 0 && len(h.framesOf(wins[1].ID())) > 0 || len(h.failed()) > 0
	}, "both windows did not present")
	stop()
	if errs := h.failed(); len(errs) > 0 {
		t.Fatalf("a frame failed: %v", errs[0])
	}
	for i, w := range wins {
		f := h.framesOf(w.ID())[0]
		if got := f.shot.RGBAAt(1, 1); got != (color.RGBA{0x67, 0x50, 0xA4, 0xFF}) {
			t.Errorf("window %d: pixel = %v", i, got)
		}
	}
	if dev.opened != 1 {
		t.Errorf("%d devices were opened for two windows", dev.opened)
	}
	// Closing one window and drawing the other goes on.
	r.Release(wins[0].NativeSurface())
	checkValidation(t, r, dev)
}

func TestX11WindowPresents(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("there is no X server (run under scripts/headless-x11.sh)")
	}
	p, err := x11.New("")
	if err != nil {
		t.Fatalf("could not connect to the X server: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	// The renderer is closed before the platform, as the engine does: its swapchains
	// draw into the windows.
	r, dev := newWindowRenderer(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "vulkan x11 display test"
	cfg.Size = ui.Size{Width: 200, Height: 100}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := newWindowHandler(r, dev, true, win)
	stop := runPlatform(t, p, h)
	waitUntil(t, func() bool { return len(h.framesOf(win.ID())) > 0 || len(h.failed()) > 0 }, "no frame was presented")
	stop()
	if errs := h.failed(); len(errs) > 0 {
		t.Fatalf("a frame failed: %v", errs[0])
	}
	f := h.framesOf(win.ID())[0]
	if got := f.shot.RGBAAt(f.metrics.X/2, f.metrics.Y/2); got != (color.RGBA{0x67, 0x50, 0xA4, 0xFF}) {
		t.Errorf("pixel = %v, want #6750A4", got)
	}
	checkValidation(t, r, dev)
}
