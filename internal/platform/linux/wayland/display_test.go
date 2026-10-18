//go:build glutter_display

package wayland

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// These tests need a Wayland compositor. Run them under the headless one:
//
//	scripts/headless-wayland.sh -- go test -race -tags glutter_display ./internal/platform/linux/wayland/

func connectToCompositor(t *testing.T) *Platform {
	t.Helper()
	p, err := New("")
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
