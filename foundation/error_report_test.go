package foundation

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"
)

func routeOf(t *testing.T, err error) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	return appErr
}

func TestPanicError(t *testing.T) {
	tagged := errors.New("SomeDebugCheck", "a check failed")
	var runtimeErr error
	runtimeErrIndex := []int{3}
	func() {
		defer func() { runtimeErr = recover().(runtime.Error) }()
		var s []int
		_ = s[len(runtimeErrIndex)]
	}()
	tests := []struct {
		name      string
		recovered any
		wantTag   string
		wantMsg   string
	}{
		{"a string", "boom", BuildPanicked, "boom"},
		{"a number", 42, BuildPanicked, "42"},
		{"a runtime error", runtimeErr, BuildPanicked, "index out of range"},
		{"a tagged error keeps its tag", tagged, "SomeDebugCheck", "a check failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := PanicError(BuildPanicked, "while building Foo", tc.recovered, "path", "Foo ← Bar")
			appErr := routeOf(t, err)
			if !appErr.HasRoute(tc.wantTag) {
				t.Errorf("route = %s, want %s", appErr.Route(), tc.wantTag)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("message = %q, want it to contain %q", err.Error(), tc.wantMsg)
			}
			for key, want := range map[string]string{"context": "while building Foo", "path": "Foo ← Bar"} {
				if appErr.Metadata[key] != want {
					t.Errorf("metadata %s = %v, want %v", key, appErr.Metadata[key], want)
				}
			}
			if stack, _ := appErr.Metadata["stack"].(string); !strings.Contains(stack, "TestPanicError") {
				t.Errorf("the stack does not show the caller: %q", stack)
			}
		})
	}
}

func TestPanicErrorTags(t *testing.T) {
	for _, tag := range []string{BuildPanicked, LayoutPanicked, PaintPanicked, CallbackPanicked} {
		err := PanicError(tag, "while testing", "x")
		if !routeOf(t, err).HasRoute(tag) {
			t.Errorf("the error for %s has route %s", tag, routeOf(t, err).Route())
		}
	}
}

func TestGuard(t *testing.T) {
	t.Run("reports a panic and goes on", func(t *testing.T) {
		var got []error
		panicked := Guard(func(err error) { got = append(got, err) }, CallbackPanicked, "while testing", func() []any { return []any{"extra", 1} }, func() { panic("bad") })
		if !panicked || len(got) != 1 {
			t.Fatalf("panicked = %v, reports = %d", panicked, len(got))
		}
		appErr := routeOf(t, got[0])
		if !appErr.HasRoute(CallbackPanicked) || appErr.Metadata["extra"] != 1 {
			t.Errorf("error = %v (%v)", appErr.Route(), appErr.Metadata)
		}
	})
	t.Run("does not call the pair function without a panic", func(t *testing.T) {
		called := false
		if Guard(func(error) { t.Error("reported") }, CallbackPanicked, "x", func() []any { called = true; return nil }, func() {}) {
			t.Error("panicked")
		}
		if called {
			t.Error("the pairs were made without a panic")
		}
	})
	t.Run("without a reporter the panic goes on", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "bad" {
				t.Errorf("recovered %v, want bad", r)
			}
		}()
		Guard(nil, CallbackPanicked, "x", nil, func() { panic("bad") })
	})
}

func TestUIThread(t *testing.T) {
	ui := NewUIThread()
	if !ui.IsCurrent() {
		t.Fatal("the goroutine that made the UIThread is not current")
	}
	var nilThread *UIThread
	if !nilThread.IsCurrent() {
		t.Error("a nil UIThread says no")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var other bool
	var panicked any
	go func() {
		defer wg.Done()
		other = ui.IsCurrent()
		func() {
			defer func() { panicked = recover() }()
			ui.AssertCurrent("Foo.Bar")
		}()
	}()
	wg.Wait()
	if other {
		t.Error("another goroutine is the UI thread")
	}
	if DebugMode {
		appErr := routeOf(t, panicked.(error))
		if !appErr.HasRoute("WrongThread") || appErr.Metadata["call"] != "Foo.Bar" {
			t.Errorf("error = %v (%v)", appErr.Route(), appErr.Metadata)
		}
	} else if panicked != nil {
		t.Errorf("release builds do not check: %v", panicked)
	}
	// Adopt moves the UI thread to the calling goroutine.
	done := make(chan bool)
	go func() {
		ui.Adopt()
		done <- ui.IsCurrent()
	}()
	if !<-done || ui.IsCurrent() {
		t.Error("Adopt did not move the UI thread")
	}
}

func TestChangeNotifierReportsPanics(t *testing.T) {
	t.Run("with a reporter the others still run", func(t *testing.T) {
		n := &ChangeNotifier{}
		var reports []error
		n.SetErrorReporter(func(err error) { reports = append(reports, err) })
		var calls []int
		n.AddListener(func() { calls = append(calls, 1) })
		n.AddListener(func() { panic("bad") })
		n.AddListener(func() { calls = append(calls, 3) })
		n.NotifyListeners()
		if len(calls) != 2 || calls[0] != 1 || calls[1] != 3 {
			t.Errorf("calls = %v, want [1 3]", calls)
		}
		if len(reports) != 1 || !routeOf(t, reports[0]).HasRoute(CallbackPanicked) {
			t.Fatalf("reports = %v", reports)
		}
		if !strings.Contains(routeOf(t, reports[0]).Metadata["context"].(string), "dispatching notifications") {
			t.Errorf("context = %v", routeOf(t, reports[0]).Metadata["context"])
		}
		// The notifier is usable afterwards.
		n.NotifyListeners()
		if len(reports) != 2 {
			t.Errorf("%d reports after two notifications, want 2", len(reports))
		}
	})
	t.Run("without a reporter the first panic goes on after the others ran", func(t *testing.T) {
		n := &ChangeNotifier{}
		var calls []int
		n.AddListener(func() { panic("first") })
		n.AddListener(func() { calls = append(calls, 2); panic("second") })
		n.AddListener(func() { calls = append(calls, 3) })
		func() {
			defer func() {
				if r := recover(); r != "first" {
					t.Errorf("recovered %v, want the first panic", r)
				}
			}()
			n.NotifyListeners()
		}()
		if len(calls) != 2 {
			t.Errorf("calls = %v, want the other listeners to run", calls)
		}
		if n.notifyDepth != 0 {
			t.Error("the notifier still thinks it is notifying")
		}
	})
	t.Run("a notifier with a UI thread is checked", func(t *testing.T) {
		if !DebugMode {
			t.Skip("needs the debug-only check")
		}
		n := &ChangeNotifier{}
		n.SetUIThread(NewUIThread())
		n.NotifyListeners() // on the UI thread: fine
		var panicked any
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { panicked = recover() }()
			n.NotifyListeners()
		}()
		wg.Wait()
		if panicked == nil || !routeOf(t, panicked.(error)).HasRoute("WrongThread") {
			t.Errorf("panic = %v, want WrongThread", panicked)
		}
	})
}
