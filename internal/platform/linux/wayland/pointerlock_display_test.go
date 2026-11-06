//go:build linux && !android && glutter_display

package wayland

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplayPointerLockOnSway asks sway for a lock. Sway offers both protocols, but its headless session has no input
// device, so the seat has no pointer (swaymsg -t get_seats says capabilities 0) and there is nothing to lock: the
// window says PointerLockUnavailable, and nothing is sent to the compositor that could be a protocol error. What
// the compositor sends when there is a pointer (locked, unlocked, the relative motion) is tested against the fake
// compositor in pointerlock_test.go; a real pointer is a manual check.
func TestDisplayPointerLockOnSway(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		t.Skip("this test runs under sway (run under scripts/headless-wayland.sh)")
	}
	p := connectToCompositor(t)
	for _, iface := range []string{"zwp_pointer_constraints_v1", "zwp_relative_pointer_manager_v1"} {
		if _, ok := p.Globals(iface); !ok {
			t.Skipf("the compositor has no %s", iface)
		}
	}
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter pointer lock test"
	cfg.AppID = "dev.glutter.locktest"
	cfg.Size = ui.Size{Width: 400, Height: 300}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	}()
	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "no configure arrived")
	var hasPointer bool
	var lockErr error
	ran := make(chan struct{})
	p.Post(func() {
		hasPointer = p.pointer != nil
		lockErr = win.(platform.PointerLocker).LockPointer()
		close(ran)
	})
	<-ran
	if hasPointer {
		t.Skip("this seat has a pointer: lock it by hand (the fake compositor tests the protocol)")
	}
	appErr, ok := errors.FromError(lockErr)
	if !ok || appErr.Tag != ui.TagPointerLockUnavailable || !appErr.HasRoute("window.LockPointer.PointerLockUnavailable") {
		t.Errorf("LockPointer = %v, want PointerLockUnavailable", lockErr)
	}
}
