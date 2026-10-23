//go:build linux && !android && glutter_display

package wayland

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplayHidingAndShowingAWindow hides a window and shows it again, twice, on a
// real compositor, with each way of decorating it: the compositor must accept the
// unmap, the new role on the same surface and its first commit (a protocol error
// would end Run), and the window must be configured and drawn again each time.
func TestDisplayHidingAndShowingAWindow(t *testing.T) {
	for _, tt := range []struct {
		name        string
		decorations platform.Decorations
		libdecor    bool
	}{
		{"auto", platform.DecorationsAuto, false},
		{"the app draws the title bar", platform.DecorationsApp, false},
		{"libdecor", platform.DecorationsLibdecor, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := connectToCompositor(t)
			if tt.libdecor && p.libdecorContext() == nil {
				t.Skip("libdecor is not available")
			}
			cfg := platform.DefaultWindowConfig()
			cfg.Title = "Glutter hide test"
			cfg.AppID = "dev.glutter.hidetest"
			cfg.Size = ui.Size{Width: 320, Height: 240}
			cfg.Decorations = tt.decorations
			win, err := p.NewWindow(cfg)
			if err != nil {
				t.Fatal(err)
			}
			h := &recHandler{allowClose: true}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- p.Run(ctx, h) }()
			onLoop := func(fn func()) {
				t.Helper()
				finished := make(chan struct{})
				p.Post(func() { fn(); close(finished) })
				select {
				case <-finished:
				case err := <-done:
					t.Fatalf("Run returned %v", err)
				}
			}
			waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "the window was never configured")

			for cycle := 1; cycle <= 2; cycle++ {
				onLoop(func() { win.SetHidden(true) })
				var mapped bool
				time.Sleep(150 * time.Millisecond) // the compositor handles the unmap
				onLoop(func() { mapped = win.(*window).mapped })
				if mapped {
					t.Fatalf("cycle %d: the hidden window is mapped", cycle)
				}
				seen := len(h.metricsSeen())
				onLoop(func() { win.SetHidden(false) })
				waitUntil(t, func() bool { return len(h.metricsSeen()) > seen }, "the shown window was never configured again")
				waitUntil(t, func() bool {
					onLoop(func() { mapped = win.(*window).mapped })
					return mapped
				}, "the shown window was never drawn again")
				if info, _ := h.lastWindowInfo(); info.Title != cfg.Title {
					t.Errorf("cycle %d: title %q after the show", cycle, info.Title)
				}
			}
			onLoop(win.Close)
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("Run returned %v", err)
				}
			case <-time.After(15 * time.Second):
				t.Fatal("Run did not return after the window closed")
			}
		})
	}
}
