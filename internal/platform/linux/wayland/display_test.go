//go:build glutter_display

package wayland

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// These tests need a Wayland compositor. Run them under the headless one:
//
//	scripts/headless-wayland.sh -- go test -race -tags glutter_display ./internal/platform/linux/wayland/

func connectToCompositor(t *testing.T) *Platform {
	t.Helper()
	// These tests check the shared-memory path; the EGL path is tested in
	// internal/gpu/gles.
	p, err := New("", Options{ShmFill: true})
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("could not connect to the compositor: %s %v (is WAYLAND_DISPLAY set? run under scripts/headless-wayland.sh)", appErr.Tag, appErr.Metadata)
		}
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return p
}

func TestDisplayConnectListsRequiredGlobals(t *testing.T) {
	p := connectToCompositor(t)

	var all []string
	for iface := range p.sess.advertised {
		all = append(all, iface)
	}
	sort.Strings(all)
	t.Logf("the compositor offers: %v", all)

	for _, r := range required {
		globals, ok := p.Globals(r.iface.Name)
		if !ok {
			t.Errorf("required global %s is not offered", r.iface.Name)
			continue
		}
		t.Logf("%s: offered up to version %d, bound at %d", r.iface.Name, globals[0].Version, bindVersion(globals[0], r.max))
	}
	for name, g := range p.sess.optionalPresent {
		t.Logf("optional %s: version %d", name, g.Version)
	}

	s := p.sess
	if s.compositor == nil || s.subcompositor == nil || s.shm == nil || s.seat == nil || s.wmBase == nil || len(s.outputs) == 0 {
		t.Errorf("not every required global is bound: %+v", s)
	}
	if v := s.compositor.ProxyVersion(); v < 4 {
		t.Errorf("wl_compositor was bound at version %d, want at least 4", v)
	}
}

func TestDisplayRunAndPost(t *testing.T) {
	p := connectToCompositor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	posted := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Post(func() { close(posted) })
	}()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, nopHandler{}) }()

	select {
	case <-posted:
	case <-time.After(5 * time.Second):
		t.Fatal("a posted function did not run")
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v, want nil", err)
	}
}

// TestDisplayWindowMapsAndCloses opens a window on the real compositor and
// checks the whole path: the first configure arrives, the window is mapped
// with a buffer of the asked size, and a close request ends Run.
func TestDisplayWindowMapsAndCloses(t *testing.T) {
	p := connectToCompositor(t)

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter display test"
	cfg.AppID = "dev.glutter.displaytest"
	cfg.Size = ui.Size{Width: 640, Height: 400}
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}

	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()

	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "no configure arrived, so the window never got a size")
	var configures int
	var states WindowStates
	var w *window
	finished := make(chan struct{})
	p.Post(func() {
		w = p.windows[0]
		configures, states = w.configures, w.states
		if !w.mapped {
			t.Error("the window is not mapped after the first configure")
		}
		close(finished)
	})
	<-finished
	if configures < 1 {
		t.Errorf("configures = %d, want at least 1", configures)
	}
	m := h.metricsSeen()[0]
	if m.PhysicalSize.Width <= 0 || m.PhysicalSize.Height <= 0 || m.DevicePixelRatio < 1 {
		t.Errorf("metrics = %+v, want a positive size at a ratio of at least 1", m)
	}
	t.Logf("first configure: %vx%v, states %+v", m.PhysicalSize.Width, m.PhysicalSize.Height, states)

	// The compositor owns the close button; ask the window to close the way the
	// toolkit would on a close request.
	closed := make(chan struct{})
	p.Post(func() {
		w.onClose()
		close(closed)
	})
	<-closed
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v after the window closed", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the last window closed")
	}
	if got := h.closeRequests(); len(got) != 1 || got[0] != 0 {
		t.Errorf("close requests = %v, want [0]", got)
	}
}

// outputScale is the scale of the headless compositor's output: HEADLESS_SCALE, as
// scripts/headless-wayland.sh reads it, or 1.
func outputScale(t *testing.T) float64 {
	t.Helper()
	v := os.Getenv("HEADLESS_SCALE")
	if v == "" {
		return 1
	}
	scale, err := strconv.ParseFloat(v, 64)
	if err != nil || scale < 1 {
		t.Fatalf("HEADLESS_SCALE = %q", v)
	}
	return scale
}

// TestDisplayScaleSizesTheBuffer checks, against the real compositor, that a window's
// buffer is ceil(logical size x scale) and that the metrics say so. Run it with
// HEADLESS_SCALE=1.5 for a fractional scale (SPEC §8.11).
func TestDisplayScaleSizesTheBuffer(t *testing.T) {
	scale := outputScale(t)
	p := connectToCompositor(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter display test"
	cfg.Size = ui.Size{Width: 333, Height: 201} // sizes that do not scale to whole pixels
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()

	wantW, wantH := math.Ceil(333*scale), math.Ceil(201*scale)
	waitUntil(t, func() bool {
		m := h.metricsSeen()
		return len(m) > 0 && m[len(m)-1].PhysicalSize == (ui.Size{Width: wantW, Height: wantH})
	}, "the metrics never reached the size of a buffer at the output's scale")
	if m := h.metricsSeen(); m[len(m)-1].DevicePixelRatio != scale {
		t.Errorf("device pixel ratio = %v, want %v", m[len(m)-1].DevicePixelRatio, scale)
	}
	var bufW, bufH int32
	var viewport bool
	finished := make(chan struct{})
	p.Post(func() {
		w := p.windows[0]
		bufW, bufH, viewport = w.buffer.width, w.buffer.height, w.viewportUsed
		close(finished)
	})
	<-finished
	if float64(bufW) != wantW || float64(bufH) != wantH {
		t.Errorf("the buffer is %dx%d pixels, want %vx%v", bufW, bufH, wantW, wantH)
	}
	// A fractional scale goes through a viewport; an integer one through the buffer scale.
	if _, ok := p.Globals("wp_fractional_scale_manager_v1"); ok && scale != math.Trunc(scale) && !viewport {
		t.Error("a fractional scale did not use a viewport")
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// TestDisplayResizeNeverCommitsTheOldSize resizes a floating window through sway's IPC
// and checks that after each acknowledged configure the next buffer committed has the
// size that was acknowledged, and that the buffer follows the logical size and the
// scale. The sizes are the container's, which sway makes the window smaller than by its
// borders and title bar, so the test follows what the compositor decides.
func TestDisplayResizeNeverCommitsTheOldSize(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		t.Skip("this test resizes the window through sway (run under scripts/headless-wayland.sh)")
	}
	scale := outputScale(t)
	p := connectToCompositor(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter resize test"
	cfg.AppID = "dev.glutter.resizetest"
	cfg.Size = ui.Size{Width: 400, Height: 300}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var trace []string
	win.(*window).trace = func(e string) { mu.Lock(); trace = append(trace, e); mu.Unlock() }

	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	// Whatever happens, Run ends before the platform is closed.
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	}()

	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "no configure arrived")
	last := h.metricsSeen()[len(h.metricsSeen())-1].PhysicalSize
	for _, size := range [][2]int{{500, 350}, {321, 201}, {640, 480}, {400, 300}} {
		out, err := exec.Command("swaymsg", "[app_id=dev.glutter.resizetest]", "resize", "set", "width", strconv.Itoa(size[0]), "px", "height", strconv.Itoa(size[1]), "px").CombinedOutput()
		if err != nil {
			t.Fatalf("swaymsg: %v: %s", err, out)
		}
		waitUntil(t, func() bool {
			m := h.metricsSeen()
			return m[len(m)-1].PhysicalSize != last
		}, fmt.Sprintf("the window did not change size after resizing to %v (metrics: %v)", size, h.metricsSeen()))
		m := h.metricsSeen()
		last = m[len(m)-1].PhysicalSize
		// The buffer is the logical size times the scale, rounded up.
		var width, height int32
		finished := make(chan struct{})
		p.Post(func() { w := p.windows[0]; width, height = w.width, w.height; close(finished) })
		<-finished
		if want := (ui.Size{Width: math.Ceil(float64(width) * scale), Height: math.Ceil(float64(height) * scale)}); last != want {
			t.Errorf("after resizing to %v: the buffer is %v for a window of %dx%d at scale %v, want %v", size, last, width, height, scale, want)
		}
	}
	cancel()

	mu.Lock()
	defer mu.Unlock()
	acks := 0
	for i, e := range trace {
		if !strings.HasPrefix(e, "ack ") {
			continue
		}
		acks++
		if i+1 >= len(trace) || trace[i+1] != "commit "+strings.TrimPrefix(e, "ack ") {
			t.Errorf("the buffer committed after %q is not that size: %v", e, trace[i:min(i+2, len(trace))])
		}
	}
	if acks < 5 {
		t.Errorf("only %d configures were acknowledged: %v", acks, trace)
	}
}
