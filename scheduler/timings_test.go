package scheduler

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func oneTiming(n int) ui.FrameTiming {
	return ui.NewFrameTiming(0, 0, time.Millisecond, time.Millisecond, 2*time.Millisecond, 0, n)
}

func TestTimingsCallbacks(t *testing.T) {
	tt := newTester()
	if tt.d.NeedsReportTimings() {
		t.Fatal("the dispatcher is asked for timings with no callback")
	}
	var first, second []int
	h1 := tt.AddTimingsCallback(func(ts []ui.FrameTiming) {
		for _, f := range ts {
			first = append(first, f.FrameNumber())
		}
	})
	h2 := tt.AddTimingsCallback(func(ts []ui.FrameTiming) { second = append(second, len(ts)) })
	if !tt.d.NeedsReportTimings() {
		t.Fatal("the dispatcher is not asked for timings with a callback")
	}
	tt.d.ReportTimings([]ui.FrameTiming{oneTiming(1), oneTiming(2)})
	tt.d.ReportTimings(nil) // an empty batch is not reported
	if len(first) != 2 || first[0] != 1 || first[1] != 2 || len(second) != 1 || second[0] != 2 {
		t.Errorf("first = %v, second = %v", first, second)
	}
	tt.RemoveTimingsCallback(h1)
	tt.RemoveTimingsCallback(h1) // removing twice does nothing
	tt.d.ReportTimings([]ui.FrameTiming{oneTiming(3)})
	if len(first) != 2 || len(second) != 2 {
		t.Errorf("after the first was removed: first = %v, second = %v", first, second)
	}
	tt.RemoveTimingsCallback(h2)
	if tt.d.NeedsReportTimings() {
		t.Error("the dispatcher is still asked for timings after the last callback was removed")
	}
}

func TestTimingsCallbackThatPanicsIsReportedAndTheOthersRun(t *testing.T) {
	tt := newTester()
	var reported []error
	tt.ErrorReporter = func(err error) { reported = append(reported, err) }
	ran := false
	tt.AddTimingsCallback(func([]ui.FrameTiming) { panic("boom") })
	tt.AddTimingsCallback(func([]ui.FrameTiming) { ran = true })
	tt.d.ReportTimings([]ui.FrameTiming{oneTiming(1)})
	if !ran {
		t.Error("the callback after the panicking one did not run")
	}
	if len(reported) != 1 {
		t.Fatalf("%d errors reported, want 1", len(reported))
	}
	if !foundation.IsRecoveredPanic(reported[0]) {
		t.Errorf("the error is not a recovered panic: %v", reported[0])
	}
}
