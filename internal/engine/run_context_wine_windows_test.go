//go:build windows && glutter_wine

package engine

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// TestACancelledContextEndsRunUnderWine runs the engine on the Win32 platform under Wine and
// ends it by cancelling the context after the first frame (step 29.38): Run returns nil within a
// few seconds, and the window is closed. Run it with
// GOOS=windows go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 -run ACancelledContext ./internal/engine
func TestACancelledContextEndsRunUnderWine(t *testing.T) {
	tests := []struct {
		name string
		// ctx makes the context Run gets and the function that ends the run.
		ctx func() (context.Context, context.CancelFunc)
	}{
		{"cancelled after the first frame", func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}},
		{"the parent of the context cancelled after the first frame", func() (context.Context, context.CancelFunc) {
			parent, end := context.WithCancel(context.Background())
			child, stop := context.WithTimeout(parent, 2*time.Minute)
			return child, func() { end(); stop() }
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			ctx, cancel := tc.ctx()
			defer cancel()
			scene := rectangleScene()
			var once sync.Once
			drew := make(chan struct{})
			var cancelledAt atomic.Int64
			go func() {
				select {
				case <-drew:
				case <-time.After(time.Minute):
				}
				time.Sleep(200 * time.Millisecond) // a few more frames
				cancelledAt.Store(time.Now().UnixNano())
				cancel() // from a goroutine that is not the loop's
			}()
			log, buf := recordingLog()
			err := Run(ctx, Config{
				Logger: log, Getenv: func(k string) string {
					if k == RendererEnv {
						return "soft"
					}
					return ""
				},
				Window: platform.WindowConfig{Title: "Context", Size: ui.Size{Width: 200, Height: 100}},
				Scene:  func(ui.Size, float64) *ui.Scene { once.Do(func() { close(drew) }); return scene },
			})
			returned := time.Now()
			if err != nil {
				t.Fatalf("Run = %v, want nil\n%s", err, buf.String())
			}
			at := cancelledAt.Load()
			if at == 0 {
				t.Fatalf("Run returned before the context was cancelled\n%s", buf.String())
			}
			if d := returned.Sub(time.Unix(0, at)); d > 5*time.Second {
				t.Errorf("Run ended %v after the context did, want a few seconds at most", d)
			}
		})
	}
}
