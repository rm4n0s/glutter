package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// ctxKey is the key of a value in a context.
type ctxKey struct{}

// A context that ends while the loop runs ends Run like an exit request: Run returns nil,
// and the renderer and the connection are closed.
func TestRunEndsWhenTheContextEnds(t *testing.T) {
	tests := []struct {
		name string
		// ctx makes the context Run gets, from the parent the test ends.
		ctx func(parent context.Context) (context.Context, context.CancelFunc)
	}{
		{"the context itself", func(p context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(p)
		}},
		{"a child of a context", func(p context.Context) (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.WithValue(p, ctxKey{}, 1), time.Minute)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, map[string]string{"DISPLAY": ":0"})
			asked := make(chan struct{}, 1)
			h.cfg.Start = func(d *ui.PlatformDispatcher) error {
				// The observers and OnExitRequested are not asked when the context ends.
				d.OnExitRequested = func(ui.ViewID) ui.AppExitResponse {
					asked <- struct{}{}
					return ui.AppExitResponseCancel
				}
				return nil
			}
			h.c.Inject(metrics(64, 48))
			h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})

			parent, end := context.WithCancel(context.Background())
			defer end()
			ctx, cancel := tc.ctx(parent)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Run(ctx, h.cfg) }()
			// The loop runs and has drawn the first frame.
			waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) > 0 })
			select {
			case err := <-done:
				t.Fatalf("Run returned before the context ended: %v", err)
			default:
			}

			end() // from this goroutine, not the loop's
			if err := result(t, done); err != nil {
				t.Fatalf("Run = %v, want nil", err)
			}
			if _, closed := h.r.snapshot(); closed != 1 {
				t.Errorf("the renderer was closed %d times, want once", closed)
			}
			if !h.c.closed {
				t.Error("the connection was not closed")
			}
			select {
			case <-asked:
				t.Error("OnExitRequested was asked when the context ended")
			default:
			}
		})
	}
}

// A context that has ended before the call returns nil before the display is connected: the
// connector never runs and no window is made.
func TestRunWithAnEndedContextOpensNothing(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
	}{
		{"cancelled", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}},
		{"past its deadline", func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}},
		{"a child of a cancelled context", func() (context.Context, context.CancelFunc) {
			parent, end := context.WithCancel(context.Background())
			end()
			return context.WithCancel(parent)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, map[string]string{"DISPLAY": ":0"})
			var connected atomic.Int32
			h.cfg.connectors = map[string]func() (connection, error){"": func() (connection, error) {
				connected.Add(1)
				return h.c, nil
			}}
			ctx, cancel := tc.ctx()
			defer cancel()
			if err := Run(ctx, h.cfg); err != nil {
				t.Fatalf("Run = %v, want nil", err)
			}
			if n := connected.Load(); n != 0 {
				t.Errorf("the connector ran %d times, want 0", n)
			}
			if n := len(h.c.Windows()); n != 0 {
				t.Errorf("%d windows were made, want none", n)
			}
			if _, _, count := h.hooked(); count != 0 {
				t.Errorf("the Window hook ran %d times, want 0", count)
			}
			if calls, closed := h.r.snapshot(); len(calls) != 0 || closed != 0 {
				t.Errorf("the renderer got %d draws and %d closes, want none", len(calls), closed)
			}
		})
	}
}

// The other ways to end Run still return nil and close what they opened. (The last window
// closed ends the platform's loop, which the fake does not do: the platforms' tests cover it.)
func TestTheExitPathsStillEndRun(t *testing.T) {
	tests := []struct {
		name string
		// end ends the run, with the harness already running and the first frame drawn.
		end func(t *testing.T, h *harness, d <-chan *ui.PlatformDispatcher)
	}{
		{"the Window hook's stop", func(t *testing.T, h *harness, _ <-chan *ui.PlatformDispatcher) { h.stopEngine(t) }},
		{"RequestAppExit", func(t *testing.T, h *harness, d <-chan *ui.PlatformDispatcher) {
			disp := <-d
			h.c.Post(disp.RequestAppExit)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, map[string]string{"DISPLAY": ":0"})
			disp := make(chan *ui.PlatformDispatcher, 1)
			h.cfg.Start = func(d *ui.PlatformDispatcher) error { disp <- d; return nil }
			h.c.Inject(metrics(64, 48))
			h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
			done := h.run(t)
			waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) > 0 })
			tc.end(t, h, disp)
			if err := result(t, done); err != nil {
				t.Fatalf("Run = %v, want nil", err)
			}
			if _, closed := h.r.snapshot(); closed != 1 {
				t.Errorf("the renderer was closed %d times, want once", closed)
			}
			if !h.c.closed {
				t.Error("the connection was not closed")
			}
		})
	}
}
