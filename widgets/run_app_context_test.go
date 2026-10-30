package widgets

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/engine"
)

func TestRunGivesTheEngineAChildOfTheContext(t *testing.T) {
	tests := []struct {
		name string
		// parent makes the parent context and the function that ends it.
		parent func() (context.Context, context.CancelFunc)
		// doneAtCall is whether the engine's context has ended when the engine is called.
		doneAtCall bool
	}{
		{"RunApp: a context that does not end", func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, false},
		{"a context that ended before the call", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parent, cancel := tc.parent()
			defer cancel()
			var atCall bool
			var got context.Context
			err := run(parent, NewText("x"), true, RunAppOpts{}, func(ctx context.Context, _ engine.Config) error {
				got, atCall = ctx, ctx.Err() != nil
				return nil
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if atCall != tc.doneAtCall {
				t.Errorf("the engine's context had ended at the call = %v, want %v", atCall, tc.doneAtCall)
			}
			if got.Err() == nil {
				t.Error("the engine's context did not end when run returned")
			}
		})
	}
}

func TestRunEndsTheEngineContextWhenTheParentEnds(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ended := make(chan struct{})
	err := run(parent, NewText("x"), true, RunAppOpts{}, func(ctx context.Context, _ engine.Config) error {
		if ctx.Err() != nil {
			t.Error("the engine's context had ended before the parent did")
		}
		go func() {
			<-ctx.Done()
			close(ended)
		}()
		cancel()
		select {
		case <-ended:
		case <-time.After(5 * time.Second):
			t.Error("the engine's context did not end with its parent")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunReturnsWhatTheEngineReturns(t *testing.T) {
	want := errors.New(tagTestEngineFailed, "the engine failed")
	got := run(context.Background(), NewText("x"), true, RunAppOpts{}, func(context.Context, engine.Config) error { return want })
	if got != want {
		t.Errorf("run = %v, want the engine's error", got)
	}
}

// A context that ended before the call returns nil before the display is opened, so the
// test needs no display (without the check, the call fails with NoDisplay or opens a window).
func TestRunAppContextWithAnEndedContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, Widget, ...RunAppOpts) error
	}{
		{"RunAppContext", RunAppContext},
		{"RunWidgetContext", RunWidgetContext},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			done := make(chan error, 1)
			go func() { done <- tc.run(ctx, NewText("x")) }()
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("%s = %v, want nil", tc.name, err)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("%s did not return", tc.name)
			}
		})
	}
}

// tagTestEngineFailed is the tag of the error that the fake engine of the tests returns.
const tagTestEngineFailed = "TestEngineFailed"
