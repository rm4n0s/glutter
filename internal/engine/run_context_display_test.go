//go:build linux && !android && glutter_display

package engine_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// These tests make a real window with engine.Run and end it by cancelling the context
// after the first frame (step 29.38): Run returns nil within a few seconds, and the
// window and the renderer are closed. They run in a headless compositor
// (scripts/headless-wayland.sh) or X server (scripts/headless-x11.sh) with the default
// backend, and skip without a display. The app is a process of its own, as in the
// other display tests (Mesa's EGL does not like a second connection in one process).

const contextChildEnv = "GLUTTER_CONTEXT_CHILD"

// TestDisplayContextChild is the app, started by TestDisplayACancelledContextEndsRun; it
// prints what happened. It does nothing in a normal run.
func TestDisplayContextChild(t *testing.T) {
	mode := os.Getenv(contextChildEnv)
	if mode == "" {
		t.Skip("run by TestDisplayACancelledContextEndsRun")
	}
	// EGL binds the context to the thread that made it current: Run draws on this
	// goroutine, which must stay on one thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// "cancel" ends the context Run gets; "parent" ends the parent of a child that Run gets.
	cancel, ctx := func() (context.CancelFunc, context.Context) {
		parent, end := context.WithCancel(context.Background())
		if mode == "parent" {
			child, stop := context.WithTimeout(parent, time.Minute)
			return func() { end(); stop() }, child
		}
		return end, parent
	}()
	defer cancel()
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 50, 30), gputest.FillPaint(0xFF3366CC))
	})
	var once sync.Once
	drew := make(chan struct{})
	cancelled := make(chan time.Time, 1)
	go func() {
		<-drew
		time.Sleep(200 * time.Millisecond) // a few more frames
		cancelled <- time.Now()
		cancel() // from a goroutine that is not the loop's
	}()
	cfg := engine.Config{
		Window: platform.WindowConfig{Title: "context", AppID: "dev.glutter.context", Size: ui.Size{Width: 200, Height: 100}},
		Scene: func(ui.Size, float64) *ui.Scene {
			once.Do(func() { close(drew) })
			return scene
		},
	}
	err := engine.Run(ctx, cfg)
	returned := time.Now()
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("Run: %v (route %s, metadata %v)", err, appErr.Route(), appErr.Metadata)
		}
		t.Fatalf("Run: %v", err)
	}
	select {
	case at := <-cancelled:
		fmt.Printf("RUN-ENDED-AFTER %d ms\n", returned.Sub(at).Milliseconds())
	default:
		t.Fatal("Run returned before the context was cancelled")
	}
}

func TestDisplayACancelledContextEndsRun(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") == "" {
		t.Skip("there is no display (run under scripts/headless-wayland.sh or scripts/headless-x11.sh)")
	}
	tests := []struct {
		name string
		mode string
	}{
		{"cancelled after the first frame", "cancel"},
		{"the parent of the context cancelled after the first frame", "parent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run", "^TestDisplayContextChild$", "-test.count=1", "-test.v")
			cmd.WaitDelay = 5 * time.Second
			cmd.Env = append(os.Environ(), contextChildEnv+"="+tc.mode)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("the app failed: %v\n%s", err, out)
			}
			var ms int
			i := strings.Index(string(out), "RUN-ENDED-AFTER ")
			if i < 0 {
				t.Fatalf("Run did not end after the context did:\n%s", out)
			}
			if _, err := fmt.Sscanf(string(out)[i:], "RUN-ENDED-AFTER %d ms", &ms); err != nil {
				t.Fatalf("cannot read the time: %v\n%s", err, out)
			}
			if ms > 5000 {
				t.Errorf("Run ended %d ms after the context did, want a few seconds at most", ms)
			}
		})
	}
}
