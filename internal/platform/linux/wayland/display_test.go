//go:build glutter_display

package wayland

import (
	"context"
	"sort"
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
	if m.PhysicalSize.Width <= 0 || m.PhysicalSize.Height <= 0 || m.DevicePixelRatio != 1 {
		t.Errorf("metrics = %+v, want a positive size at ratio 1", m)
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
