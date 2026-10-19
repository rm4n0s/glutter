package scheduler

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// reported collects the errors a binding reports.
type reported []error

func (r *reported) report(err error) { *r = append(*r, err) }

func (r reported) check(t *testing.T, context string) {
	t.Helper()
	if len(r) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(r), []error(r))
	}
	appErr, ok := errors.FromError(r[0])
	if !ok || !appErr.HasRoute(foundation.CallbackPanicked) {
		t.Fatalf("error = %v, want CallbackPanicked", r[0])
	}
	if got, _ := appErr.Metadata["context"].(string); !strings.Contains(got, context) {
		t.Errorf("context = %q, want %q", got, context)
	}
}

func TestPanicInATaskIsReportedAndTheNextTaskRuns(t *testing.T) {
	tt := newTester()
	var errs reported
	tt.ErrorReporter = errs.report
	var ran []string
	ScheduleTask(tt.SchedulerBinding, func() int { panic("bad task") }, PriorityIdle)
	ScheduleTask(tt.SchedulerBinding, func() string { ran = append(ran, "second"); return "" }, PriorityIdle)
	tt.host.runPosted()
	for tt.HandleEventLoopCallback() {
	}
	errs.check(t, "during a task callback")
	if !slices.Equal(ran, []string{"second"}) {
		t.Errorf("ran = %v, want the second task", ran)
	}
	if errs[0].(*errors.Error).Metadata["panic"] != "bad task" {
		t.Errorf("metadata = %v", errs[0].(*errors.Error).Metadata)
	}
}

func TestPanicInATaskWithoutAReporterGoesOn(t *testing.T) {
	tt := newTester()
	ScheduleTask(tt.SchedulerBinding, func() int { panic("bad task") }, PriorityIdle)
	defer func() {
		if r := recover(); r != "bad task" {
			t.Errorf("recovered %v, want the panic of the task", r)
		}
	}()
	tt.HandleEventLoopCallback()
}

func TestPanicInFrameCallbacksIsReportedAndTheOthersRun(t *testing.T) {
	tests := []struct {
		name string
		add  func(tt *tester, body func())
	}{
		{"transient", func(tt *tester, body func()) {
			tt.ScheduleFrameCallback(func(time.Duration) { body() })
		}},
		{"persistent", func(tt *tester, body func()) {
			tt.AddPersistentFrameCallback(func(time.Duration) { body() })
		}},
		{"post-frame", func(tt *tester, body func()) {
			tt.AddPostFrameCallback(func(time.Duration) { body() })
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTester()
			var errs reported
			tt.ErrorReporter = errs.report
			var ran []int
			tc.add(tt, func() { ran = append(ran, 1) })
			tc.add(tt, func() { panic("bad callback") })
			tc.add(tt, func() { ran = append(ran, 3) })
			tt.tick(time.Millisecond)
			errs.check(t, "during a scheduler callback")
			if !slices.Equal(ran, []int{1, 3}) {
				t.Errorf("ran = %v, want [1 3]", ran)
			}
			if tt.SchedulerPhase() != SchedulerPhaseIdle {
				t.Errorf("phase = %v after the frame", tt.SchedulerPhase())
			}
		})
	}
}

func TestPanicInAFrameCallbackWithoutAReporterGoesOn(t *testing.T) {
	tt := newTester()
	tt.ScheduleFrameCallback(func(time.Duration) { panic("bad callback") })
	defer func() {
		if r := recover(); r != "bad callback" {
			t.Errorf("recovered %v, want the panic of the callback", r)
		}
	}()
	tt.tick(time.Millisecond)
}
