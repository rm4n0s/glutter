//go:build windows && glutter_wine

package windows

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// TestWineARunThatMessagesFloodStillEnds checks that the loop comes back to its context between turns even when the queue of the
// thread never runs empty (step 31.11): frames that take longer than the display's interval, as a game on Direct3D 12 over a software
// rasterizer under Wine has, bring the next frame's messages while the last are handled, and a loop that dispatched until the queue was
// empty never looked at the context again (examples/game3d did not end after its self-capture). Here goroutines post messages to
// the window as fast as they can, and the context ends after a moment: Run must return soon after.
func TestWineARunThatMessagesFloodStillEnds(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var stop atomic.Bool
	var posted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				if err := win32.PostMessage(w.hwnd, win32.WMApp, 0, 0); err != nil {
					time.Sleep(50 * time.Microsecond) // the queue of the thread is full: that is the flood
					continue
				}
				posted.Add(1)
			}
		}()
	}
	defer func() {
		stop.Store(true)
		wg.Wait()
	}()
	var cancelled time.Time
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	p.Post(func() {
		after(p, 300*time.Millisecond, func() {
			cancelled = time.Now()
			cancel()
		})
	})
	if err := p.Run(ctx, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if cancelled.IsZero() {
		t.Fatal("the context was not cancelled by the test: Run ended by its timeout, or too early")
	}
	if took := time.Since(cancelled); took > 3*time.Second {
		t.Errorf("Run took %v to end after its context did, with %d messages posted", took, posted.Load())
	}
	if posted.Load() < 1000 {
		t.Errorf("only %d messages were posted: the flood was no flood", posted.Load())
	}
}
