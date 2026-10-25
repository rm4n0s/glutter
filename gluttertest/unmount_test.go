package gluttertest

import (
	"context"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/widgets"
)

// lifeWidget is a stateful widget that tells when its state is disposed and
// lets the test see its lifetime.
type lifeWidget struct {
	widgets.StatefulWidget
	onInit    func(lifetime context.Context)
	onDispose func()
	// panicOnDispose makes Dispose fail.
	panicOnDispose bool
}

func (w *lifeWidget) CreateState() widgets.State { return &lifeState{} }

type lifeState struct{ widgets.StateOf[*lifeWidget] }

func (s *lifeState) InitState() {
	if s.Widget().onInit != nil {
		s.Widget().onInit(s.Lifetime())
	}
}

func (s *lifeState) Dispose() {
	if s.Widget().onDispose != nil {
		s.Widget().onDispose()
	}
	if s.Widget().panicOnDispose {
		panic(errors.New(tagLifeDisposeFailed, "dispose failed"))
	}
}

func (s *lifeState) Build(widgets.BuildContext) widgets.Widget { return newSwatchBox() }

const tagLifeDisposeFailed = "LifeDisposeFailed"

func newSwatchBox() widgets.Widget { return widgets.NewSizedBox(nil) }

func newLife(w *lifeWidget) *lifeWidget {
	w.StatefulWidget = widgets.NewStatefulWidgetBase()
	return w
}

func TestTheTreeIsUnmountedAtTheEndOfATest(t *testing.T) {
	var disposed bool
	var lifetime context.Context
	t.Run("body", func(t *testing.T) {
		ts := New(t)
		ts.PumpWidget(newLife(&lifeWidget{
			onInit:    func(l context.Context) { lifetime = l },
			onDispose: func() { disposed = true },
		}))
		if disposed || lifetime.Err() != nil {
			t.Fatalf("the state was disposed before the end of the test (disposed = %v, lifetime = %v)", disposed, lifetime.Err())
		}
	})
	if !disposed {
		t.Error("State.Dispose did not run at the end of the test")
	}
	if lifetime == nil || lifetime.Err() == nil {
		t.Error("State.Lifetime was not cancelled at the end of the test")
	}
}

func TestUnmountingAtTheEndOfATest(t *testing.T) {
	tests := []struct {
		name string
		// run uses the tester and returns what the fake TB should have seen.
		run func(tb *fakeTB, life *lifeWidget, explode func())
		// disposed is whether the state is disposed after the cleanups ran.
		disposed bool
		// failure is a part of the message that the test must fail with, if any.
		failure string
	}{
		{"a passing test", func(*fakeTB, *lifeWidget, func()) {}, true, ""},
		{"a test that failed already is not unmounted", func(tb *fakeTB, _ *lifeWidget, _ func()) {
			tb.Errorf("the test failed")
		}, false, "the test failed"},
		{"an error that was not taken is not unmounted", func(_ *fakeTB, _ *lifeWidget, explode func()) { explode() }, false, "BuildPanicked"},
		{"an error while unmounting fails the test", func(_ *fakeTB, life *lifeWidget, _ func()) {
			life.panicOnDispose = true
		}, true, "LifeDisposeFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &fakeTB{}
			ts := New(tb)
			var disposed bool
			life := newLife(&lifeWidget{onDispose: func() { disposed = true }})
			explode := false
			app := func() widgets.Widget {
				return widgets.NewColumn([]widgets.Widget{life, newBuilder(func(widgets.BuildContext) widgets.Widget {
					if explode {
						panic("boom")
					}
					return newSwatchBox()
				})})
			}
			ts.PumpWidget(app())
			tt.run(tb, life, func() {
				explode = true
				ts.PumpWidget(app()) // the same widgets, so that the state stays
			})
			tb.runCleanups()
			if disposed != tt.disposed {
				t.Errorf("disposed = %v, want %v", disposed, tt.disposed)
			}
			if got := strings.Join(tb.errorfs, "\n"); (tt.failure == "") != (got == "") || !strings.Contains(got, tt.failure) {
				t.Errorf("the test failed with %q, want a failure with %q", got, tt.failure)
			}
		})
	}
}

func TestCleanupsOfTheTestSeeTheTreeBeforeItIsUnmounted(t *testing.T) {
	tb := &fakeTB{}
	ts := New(tb)
	var disposed, disposedInCleanup bool
	ts.PumpWidget(newLife(&lifeWidget{onDispose: func() { disposed = true }}))
	// Registered after New, so that it runs before the unmounting.
	tb.Cleanup(func() {
		disposedInCleanup = disposed
		if len(ts.Elements(ByType[*lifeWidget]())) != 1 {
			t.Error("the tree was gone in a cleanup that the test registered")
		}
	})
	tb.runCleanups()
	if disposedInCleanup || !disposed {
		t.Errorf("disposed in the cleanup = %v, after = %v; want false, true", disposedInCleanup, disposed)
	}
}

func TestNothingToUnmountWithoutAWidget(t *testing.T) {
	tb := &fakeTB{}
	New(tb)
	tb.runCleanups()
	if len(tb.errorfs) != 0 {
		t.Errorf("the test failed: %q", tb.errorfs)
	}
}
