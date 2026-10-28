package foundation

import (
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/internal/goid"
)

// onNewGoroutine runs fn on a goroutine of its own and waits for it to end.
func onNewGoroutine(fn func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
	wg.Wait()
}

func TestUIThreadIsCurrent(t *testing.T) {
	tests := []struct {
		name string
		// run reports IsCurrent as the case sees it; the test goroutine made ui.
		run  func(ui *UIThread) bool
		want bool
	}{
		{"the goroutine that made it", func(ui *UIThread) bool { return ui.IsCurrent() }, true},
		{"the same goroutine deep in the stack", func(ui *UIThread) bool {
			var got bool
			var deeper func(n int)
			deeper = func(n int) {
				if n > 0 {
					deeper(n - 1)
					return
				}
				got = ui.IsCurrent()
			}
			deeper(300)
			return got
		}, true},
		{"another goroutine", func(ui *UIThread) bool {
			var got bool
			onNewGoroutine(func() { got = ui.IsCurrent() })
			return got
		}, false},
		{"a nil UIThread has no opinion", func(*UIThread) bool {
			var nilThread *UIThread
			var got bool
			onNewGoroutine(func() { got = nilThread.IsCurrent() })
			return got
		}, true},
		{"a new goroutine that adopts", func(ui *UIThread) bool {
			var got bool
			onNewGoroutine(func() {
				ui.Adopt()
				got = ui.IsCurrent()
			})
			return got
		}, true},
		{"the first goroutine after another adopted", func(ui *UIThread) bool {
			onNewGoroutine(ui.Adopt)
			return ui.IsCurrent()
		}, false},
		{"the first goroutine adopts again", func(ui *UIThread) bool {
			onNewGoroutine(ui.Adopt)
			ui.Adopt()
			return ui.IsCurrent()
		}, true},
		{"adopt twice on the same goroutine", func(ui *UIThread) bool {
			ui.Adopt()
			ui.Adopt()
			return ui.IsCurrent()
		}, true},
		{"a zero UIThread is no goroutine's", func(*UIThread) bool { return (&UIThread{}).IsCurrent() }, false},
	}
	for _, tt := range tests {
		// No t.Run: a subtest would run on a goroutine that did not make ui.
		ui := NewUIThread()
		if got := tt.run(ui); got != tt.want {
			t.Errorf("%s: IsCurrent = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestUIThreadManyGoroutinesAtOnce(t *testing.T) {
	ui := NewUIThread()
	const n = 500
	results := make([]bool, n)
	var wg sync.WaitGroup
	release := make(chan struct{})
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release
			results[i] = ui.IsCurrent()
		}()
	}
	close(release)
	wg.Wait()
	for i, got := range results {
		if got {
			t.Errorf("goroutine %d is the UI thread", i)
		}
	}
	if !ui.IsCurrent() {
		t.Error("the goroutine that made the UIThread is no longer current")
	}
}

func TestUIThreadAssertCurrent(t *testing.T) {
	if !DebugMode {
		ui := NewUIThread()
		onNewGoroutine(func() { ui.AssertCurrent("Foo.Bar") }) // must not panic
		return
	}
	tests := []struct {
		name      string
		fromOther bool
		wantPanic bool
	}{
		{"on the UI goroutine", false, false},
		{"on another goroutine", true, true},
	}
	for _, tt := range tests {
		ui := NewUIThread()
		wantUI := goid.ID()
		var panicked any
		var callerID uint64
		call := func() {
			callerID = goid.ID()
			defer func() { panicked = recover() }()
			ui.AssertCurrent("Foo.Bar")
		}
		if tt.fromOther {
			onNewGoroutine(call)
		} else {
			call()
		}
		if !tt.wantPanic {
			if panicked != nil {
				t.Errorf("%s: panicked with %v", tt.name, panicked)
			}
			continue
		}
		err, ok := panicked.(error)
		if !ok {
			t.Errorf("%s: panic = %v, want an error", tt.name, panicked)
			continue
		}
		appErr := routeOf(t, err)
		if !appErr.HasRoute("UIThread.AssertCurrent.WrongThread") {
			t.Errorf("%s: route = %s", tt.name, appErr.Route())
		}
		if appErr.Metadata["call"] != "Foo.Bar" {
			t.Errorf("%s: call = %v", tt.name, appErr.Metadata["call"])
		}
		if appErr.Metadata["uiGoroutine"] != wantUI || appErr.Metadata["callingGoroutine"] != callerID || wantUI == callerID || wantUI == 0 {
			t.Errorf("%s: goroutines = %v and %v, want %d and %d", tt.name,
				appErr.Metadata["uiGoroutine"], appErr.Metadata["callingGoroutine"], wantUI, callerID)
		}
	}
}

func TestUIThreadAdoptKeepsTheNumberOfTheNewGoroutine(t *testing.T) {
	if !DebugMode {
		t.Skip("release builds make no WrongThread error")
	}
	ui := NewUIThread()
	var adopted uint64
	onNewGoroutine(func() {
		ui.Adopt()
		adopted = goid.ID()
	})
	// This goroutine is not the UI thread any more.
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		ui.AssertCurrent("Foo.Bar")
	}()
	err, ok := panicked.(error)
	if !ok {
		t.Fatalf("panic = %v, want an error", panicked)
	}
	if got := routeOf(t, err).Metadata["uiGoroutine"]; got != adopted {
		t.Errorf("uiGoroutine = %v, want %d", got, adopted)
	}
}

// atDepth calls fn n frames deeper than its caller.
//
//go:noinline
func atDepth(n int, fn func()) {
	if n > 0 {
		atDepth(n-1, fn)
		return
	}
	fn()
}

func BenchmarkUIThreadIsCurrent(b *testing.B) {
	ui := NewUIThread()
	for _, depth := range []int{5, 200} {
		name := "depth200"
		if depth == 5 {
			name = "depth5"
		}
		b.Run(name, func(b *testing.B) {
			atDepth(depth, func() {
				ui.Adopt() // a sub-benchmark runs on its own goroutine
				for b.Loop() {
					if !ui.IsCurrent() {
						b.Fatal("not current")
					}
				}
			})
		})
	}
}
