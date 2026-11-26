package widgets

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// uiQueue is a Post for tests: it keeps the functions until the test, which is
// the UI thread, runs them.
type uiQueue struct {
	mu  sync.Mutex
	fns []func()
}

func (q *uiQueue) post(fn func()) {
	q.mu.Lock()
	q.fns = append(q.fns, fn)
	q.mu.Unlock()
}

func (q *uiQueue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.fns)
}

// drain runs the posted functions on the calling goroutine.
func (q *uiQueue) drain() {
	q.mu.Lock()
	fns := q.fns
	q.fns = nil
	q.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// waitFor drains the queue until cond holds, or fails the test.
func (q *uiQueue) waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the goroutines")
		}
		q.drain()
		time.Sleep(time.Millisecond)
	}
}

func newThreadedTree(t *testing.T) (*tree, *uiQueue) {
	t.Helper()
	tr := newTree(t)
	q := &uiQueue{}
	tr.owner.Post = q.post
	tr.owner.UIThread = foundation.NewUIThread()
	return tr, q
}

func TestSetStateFromManyGoroutines(t *testing.T) {
	tr, q := newThreadedTree(t)
	count, builds := 0, 0
	var setState func(func())
	tr.pump(newBuilder(func(_ BuildContext, ss func(func())) Widget {
		setState = ss
		builds++
		return newLeaf()
	}))
	builds = 0
	const goroutines = 50
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			setState(func() { count++ })
		}()
	}
	wg.Wait()
	if count != 0 {
		t.Fatalf("count = %d before the UI thread ran the posted functions, want 0", count)
	}
	q.waitFor(t, func() bool { return count == goroutines })
	tr.pumpFrame()
	if builds != 1 {
		t.Errorf("built %d times, want 1: the calls must share one rebuild", builds)
	}
}

func TestSetStateFromAnotherGoroutineIsDroppedAfterUnmount(t *testing.T) {
	tr, q := newThreadedTree(t)
	ran := false
	var setState func(func())
	tr.pump(newMulti(newBuilder(func(_ BuildContext, ss func(func())) Widget { setState = ss; return newLeaf() })))
	done := make(chan struct{})
	go func() { setState(func() { ran = true }); close(done) }()
	<-done
	tr.pump(newMulti())
	if q.len() != 1 {
		t.Fatalf("%d functions posted, want 1", q.len())
	}
	q.drain() // in debug builds SetStateAfterDispose would panic here
	if ran {
		t.Error("the function ran for an unmounted state")
	}
}

type goer struct {
	StatefulWidget
	start func(s *goerState)
}

func (w *goer) CreateState() State { return &goerState{} }

type goerState struct{ StateOf[*goer] }

func (s *goerState) InitState()                { s.Widget().start(s) }
func (s *goerState) Build(BuildContext) Widget { return newLeaf() }

func TestStateGoRecoversPanics(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.collectPanics = true
	var cancelled <-chan struct{}
	tr.pump(&goer{StatefulWidget: NewStatefulWidgetBase(), start: func(s *goerState) {
		s.Go(func(ctx context.Context) {
			cancelled = ctx.Done()
			panic(errors.New(tagGoroutineFailed, "boom in goroutine"))
		})
	}})
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	appErr := requireErrorTag(t, tr.errs[0], "GoroutineFailed")
	if got := appErr.Metadata["context"]; got == nil {
		t.Errorf("metadata = %v, want the context", appErr.Metadata)
	}
	tr.pump(newMulti())
	select {
	case <-cancelled:
	default:
		t.Error("the context of State.Go was not cancelled when the state was disposed")
	}
}

func TestStateGoReportsUntaggedPanicsAsCallbackPanicked(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.collectPanics = true
	tr.pump(&goer{StatefulWidget: NewStatefulWidgetBase(), start: func(s *goerState) {
		s.Go(func(context.Context) { panic("plain") })
	}})
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	_ = requireErrorTag(t, tr.errs[0], foundation.CallbackPanicked)
}

// goErrJob is the function of the tests of GoErr that a state calls: the route of its error must go on with it.
type goErrJob struct {
	run  func(ctx context.Context) error
	done func(err error)
}

func (j *goErrJob) start(s *goerState) { s.GoErr(j.run, j.done) }

// failingWork makes the error of the tests of GoErr, one function below the one that GoErr runs.
func failingWork() error { return errors.New(tagGoErrWorkFailed, "the work failed") }

func startGoErr(tr *tree, j *goErrJob) {
	tr.pump(&goer{StatefulWidget: NewStatefulWidgetBase(), start: j.start})
}

func TestStateGoErrCarriesTheRouteAcrossTheGoroutine(t *testing.T) {
	tr, q := newThreadedTree(t)
	ui := foundation.NewUIThread()
	tr.owner.UIThread = ui
	var got error
	called := 0
	onUI := false
	j := &goErrJob{
		run: func(context.Context) error { return failingWork() },
		done: func(err error) {
			called++
			onUI = ui.IsCurrent()
			got = err
		},
	}
	startGoErr(tr, j)
	q.waitFor(t, func() bool { return called > 0 })
	if called != 1 || !onUI {
		t.Errorf("done ran %d times, on the UI thread: %v; want once, on the UI thread", called, onUI)
	}
	appErr := requireErrorTag(t, got, tagGoErrWorkFailed)
	t.Log(appErr.Route())
	if !appErr.HasRoute("goErrJob.start->StateOf[...].GoErr->StateOf[...].GoErr.func1->TestStateGoErrCarriesTheRouteAcrossTheGoroutine.func1->failingWork." + tagGoErrWorkFailed) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestStateGoErrWithANilError(t *testing.T) {
	tr, q := newThreadedTree(t)
	var got error = errors.New(tagGoErrWorkFailed, "unset")
	called := false
	startGoErr(tr, &goErrJob{
		run:  func(context.Context) error { return nil },
		done: func(err error) { called, got = true, err },
	})
	q.waitFor(t, func() bool { return called })
	if got != nil {
		t.Errorf("done got %v, want nil", got)
	}
}

func TestStateGoErrIsDroppedAfterDispose(t *testing.T) {
	tr, q := newThreadedTree(t)
	release := make(chan struct{})
	finished := make(chan struct{})
	var ctxDone <-chan struct{}
	startGoErr(tr, &goErrJob{
		run: func(ctx context.Context) error {
			ctxDone = ctx.Done()
			<-release
			defer close(finished)
			return failingWork()
		},
		done: func(error) { t.Error("done was called after the state was disposed") },
	})
	tr.pump(newMulti())
	close(release)
	<-finished
	// The goroutine posts done to the UI thread; it is not run for a state that is gone. The error of the work is
	// reported instead (TestStateGoErrDisposedBeforeTheWorkReturnsReportsAnError).
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	select {
	case <-ctxDone:
	default:
		t.Error("the context of State.GoErr was not cancelled when the state was disposed")
	}
}

// settle gives the goroutines of GoErr that have nothing to deliver time to finish, and runs what they posted.
func settle(q *uiQueue) {
	time.Sleep(30 * time.Millisecond)
	q.drain()
}

// requireUndelivered checks the report of an error that GoErr could not give to a callback: its tag is no panic's,
// its reason and state are in the metadata, and the error of the work is reachable with its tag and whole route.
func requireUndelivered(t *testing.T, err error, reason string) {
	t.Helper()
	report := requireErrorTag(t, err, TagGoErrUndelivered)
	if report.Tag != TagGoErrUndelivered || report.Tag == foundation.CallbackPanicked || foundation.IsRecoveredPanic(err) {
		t.Errorf("the report has the tag %s and a recovered panic's metadata: %v; want %s, which is no panic", report.Tag, report.Metadata, TagGoErrUndelivered)
	}
	if report.Metadata["reason"] != reason || report.Metadata["state"] != "goerState" || report.Metadata["library"] != "widgets library" {
		t.Errorf("metadata = %v, want the reason %q and the state goerState", report.Metadata, reason)
	}
	// The report is made inside GoErr, where the work could not be delivered: its own route says so.
	if !report.HasRoute("StateOf[...].GoErr") || !strings.HasSuffix(report.Route(), "."+TagGoErrUndelivered) {
		t.Errorf("unexpected route of the report: %s", report.Route())
	}
	if report.Metadata["originalTag"] != tagGoErrWorkFailed {
		t.Errorf("originalTag = %v, want %s", report.Metadata["originalTag"], tagGoErrWorkFailed)
	}
	if route, _ := report.Metadata["originalRoute"].(string); !strings.Contains(route, "StateOf[...].GoErr->StateOf[...].GoErr.func1->") || !strings.HasSuffix(route, "failingWork."+tagGoErrWorkFailed) {
		t.Errorf("originalRoute = %q, want the route of the error of the work", route)
	}
	all := errors.All(err)
	if len(all) != 2 || all[0] != report {
		t.Fatalf("errors.All gave %d errors, want the report and the error of the work", len(all))
	}
	if all[1].Tag != tagGoErrWorkFailed {
		t.Errorf("the original error has the tag %s, want %s", all[1].Tag, tagGoErrWorkFailed)
	}
	if !all[1].HasRoute("StateOf[...].GoErr->StateOf[...].GoErr.func1->") || !all[1].HasRoute("failingWork."+tagGoErrWorkFailed) {
		t.Errorf("unexpected route of the original error: %s", all[1].Route())
	}
	if !errors.Is(err, &errors.Error{Tag: tagGoErrWorkFailed}) {
		t.Errorf("errors.Is does not reach the original error")
	}
}

func TestStateGoErrWithoutACallbackReportsTheError(t *testing.T) {
	tr, q := newThreadedTree(t)
	ui := foundation.NewUIThread()
	tr.owner.UIThread = ui
	// The report is made by the work's goroutine and posted to the UI thread, where OnError runs.
	onUI := false
	tr.owner.OnError = func(err error) { onUI = ui.IsCurrent(); tr.errs = append(tr.errs, err) }
	startGoErr(tr, &goErrJob{run: namedWork})
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	settle(q)
	if len(tr.errs) != 1 || !onUI {
		t.Fatalf("%d reports, on the UI thread: %v; want one, on the UI thread", len(tr.errs), onUI)
	}
	report := requireErrorTag(t, tr.errs[0], TagGoErrUndelivered)
	if report.Metadata["reason"] != "no callback" {
		t.Errorf("reason = %v, want no callback", report.Metadata["reason"])
	}
	original := errors.All(tr.errs[0])[1]
	if original.Tag != tagGoErrWorkFailed || !original.HasRoute("goErrJob.start->StateOf[...].GoErr->StateOf[...].GoErr.func1->namedWork->workThatFails->failingWork."+tagGoErrWorkFailed) {
		t.Errorf("the original error: %s, route %s", original.Tag, original.Route())
	}
	requireUndelivered(t, tr.errs[0], "no callback")
}

// namedWork is the work of a test that asserts the whole route: it has a name of its own, so that its frame in a route
// does not depend on the function literals around it.
func namedWork(context.Context) error { return workThatFails() }

// workThatFails is a function between the work that GoErr runs and the function that makes its error.
func workThatFails() error { return failingWork() }

func TestStateGoErrWithoutACallbackAndWithANilErrorReportsNothing(t *testing.T) {
	tr, q := newThreadedTree(t)
	ran := make(chan struct{})
	startGoErr(tr, &goErrJob{run: func(context.Context) error { defer close(ran); return nil }})
	<-ran
	settle(q)
	if len(tr.errs) != 0 {
		t.Errorf("reported %v, want nothing for a nil error", tr.errs)
	}
}

func TestStateGoErrDisposedBeforeTheWorkReturnsReportsAnError(t *testing.T) {
	tr, q := newThreadedTree(t)
	release := make(chan struct{})
	finished := make(chan struct{})
	startGoErr(tr, &goErrJob{
		run: func(context.Context) error {
			<-release
			defer close(finished)
			return failingWork()
		},
		done: func(error) { t.Error("done was called after the state was disposed") },
	})
	tr.pump(newMulti())
	close(release)
	<-finished
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	settle(q)
	if len(tr.errs) != 1 {
		t.Fatalf("%d reports, want one", len(tr.errs))
	}
	requireUndelivered(t, tr.errs[0], "disposed")
}

func TestStateGoErrDisposedWithACancellationReportsNothing(t *testing.T) {
	cancelled := []struct {
		name string
		err  func(ctx context.Context) error
	}{
		{"the error of the context", func(ctx context.Context) error { return ctx.Err() }},
		{"the cause of the context", func(ctx context.Context) error { return context.Cause(ctx) }},
		{"wrapped in an *errors.Error", func(ctx context.Context) error { return errors.NewErr(tagGoErrWorkFailed, ctx.Err()) }},
		{"wrapped by fmt %w in an *errors.Error", func(ctx context.Context) error {
			return errors.NewErrf(tagGoErrWorkFailed, "waiting: %w", ctx.Err())
		}},
	}
	for _, tc := range cancelled {
		t.Run(tc.name, func(t *testing.T) {
			tr, q := newThreadedTree(t)
			finished := make(chan struct{})
			startGoErr(tr, &goErrJob{
				run: func(ctx context.Context) error {
					<-ctx.Done()
					defer close(finished)
					return tc.err(ctx)
				},
				done: func(error) { t.Error("done was called after the state was disposed") },
			})
			tr.pump(newMulti())
			<-finished
			q.waitFor(t, func() bool { return q.len() > 0 })
			settle(q)
			if len(tr.errs) != 0 {
				t.Errorf("reported %v, want nothing: the work ended with the cancellation of its context", tr.errs)
			}
		})
	}
}

func TestStateGoErrWithoutACallbackAfterDisposeIsQuietOnlyForACancellation(t *testing.T) {
	tr, q := newThreadedTree(t)
	finished := make(chan struct{}, 2)
	tr.pump(&goer{StatefulWidget: NewStatefulWidgetBase(), start: func(s *goerState) {
		s.GoErr(func(ctx context.Context) error {
			<-ctx.Done()
			defer func() { finished <- struct{}{} }()
			return ctx.Err()
		}, nil)
		s.GoErr(func(ctx context.Context) error {
			<-ctx.Done()
			defer func() { finished <- struct{}{} }()
			return failingWork()
		}, nil)
	}})
	tr.pump(newMulti())
	<-finished
	<-finished
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	settle(q)
	if len(tr.errs) != 1 {
		t.Fatalf("%d reports, want one: the real error, not the cancellation", len(tr.errs))
	}
	requireUndelivered(t, tr.errs[0], "no callback")
}

func TestStateGoErrMountedCallsDoneAndReportsNothing(t *testing.T) {
	tr, q := newThreadedTree(t)
	var got error
	called := 0
	startGoErr(tr, &goErrJob{
		run:  func(context.Context) error { return failingWork() },
		done: func(err error) { called++; got = err },
	})
	q.waitFor(t, func() bool { return called > 0 })
	settle(q)
	if called != 1 || len(tr.errs) != 0 {
		t.Errorf("done ran %d times and %d errors were reported, want 1 and 0", called, len(tr.errs))
	}
	_ = requireErrorTag(t, got, tagGoErrWorkFailed)
}

func TestStateGoErrReportIsToldFromAPanic(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.collectPanics = true
	tr.pump(&goer{StatefulWidget: NewStatefulWidgetBase(), start: func(s *goerState) {
		s.GoErr(func(context.Context) error { panic("plain") }, func(error) {})
		s.GoErr(func(context.Context) error { return failingWork() }, nil)
	}})
	q.waitFor(t, func() bool { return len(tr.errs) == 2 })
	var panics, reports int
	for _, err := range tr.errs {
		appErr, _ := errors.FromError(err)
		switch {
		case appErr.Tag == foundation.CallbackPanicked && foundation.IsRecoveredPanic(err):
			panics++
		case appErr.Tag == TagGoErrUndelivered && !foundation.IsRecoveredPanic(err):
			reports++
		}
	}
	if panics != 1 || reports != 1 {
		t.Errorf("%d panics and %d reports among %v, want one of each", panics, reports, tr.errs)
	}
}

func TestStateGoErrDisposedWithoutAnErrorReporterPanicsWithTheReport(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.owner.OnError = nil
	release := make(chan struct{})
	finished := make(chan struct{})
	startGoErr(tr, &goErrJob{
		run: func(context.Context) error {
			<-release
			defer close(finished)
			return failingWork()
		},
		done: func(error) { t.Error("done was called after the state was disposed") },
	})
	tr.pump(newMulti())
	close(release)
	<-finished
	for deadline := time.Now().Add(5 * time.Second); q.len() == 0; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the goroutine to post")
		}
	}
	// Without OnError the report panics, on the UI thread, as the panic of a callback does without one.
	_ = requireErrorTagFromPanic(t, panicOf(q.drain), TagGoErrUndelivered)
}

func TestEndedByCancel(t *testing.T) {
	cause := errors.New(tagStopForGood, "stopped for good")
	causeCtx, stopWithCause := context.WithCancelCause(context.Background())
	stopWithCause(cause)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	live := context.Background()
	timedOut, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"canceled", cancelled, context.Canceled, true},
		{"canceled, wrapped", cancelled, errors.NewErr(tagGoErrWorkFailed, context.Canceled), true},
		{"canceled, joined", cancelled, errors.Join(failingWork(), context.Canceled), true},
		{"the cause", causeCtx, cause, true},
		{"the cause, wrapped", causeCtx, errors.NewErr(tagGoErrWorkFailed, cause), true},
		{"a real error", cancelled, failingWork(), false},
		{"the tag of a cancellation and no cause", cancelled, errors.New(tagCanceled, "canceled"), false},
		{"the deadline that ended the context", timedOut, context.DeadlineExceeded, true},
		{"a deadline of the work, the context cancelled", cancelled, context.DeadlineExceeded, false},
		{"canceled, but the context is live", live, context.Canceled, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := endedByCancel(tc.ctx, tc.err); got != tc.want {
				t.Errorf("endedByCancel = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStateGoErrPanicInTheWork(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.collectPanics = true
	startGoErr(tr, &goErrJob{
		run:  func(context.Context) error { panic("plain") },
		done: func(error) { t.Error("done was called after a panic in the work") },
	})
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	appErr := requireErrorTag(t, tr.errs[0], foundation.CallbackPanicked)
	if appErr.Tag != foundation.CallbackPanicked {
		t.Errorf("tag = %s, want %s: a panic is not turned into an error of the work", appErr.Tag, foundation.CallbackPanicked)
	}
}

func TestStateGoErrPanicInTheCallback(t *testing.T) {
	tr, q := newThreadedTree(t)
	tr.collectPanics = true
	startGoErr(tr, &goErrJob{
		run:  func(context.Context) error { return nil },
		done: func(error) { panic("plain") },
	})
	q.waitFor(t, func() bool { return len(tr.errs) > 0 })
	_ = requireErrorTag(t, tr.errs[0], foundation.CallbackPanicked)
}

func TestStateGoErrBeforeMount(t *testing.T) {
	skipUnlessDebug(t)
	s := &builderState{}
	requireTag(t, "StateGoBeforeMount", func() { s.GoErr(func(context.Context) error { return nil }, nil) })
}

func TestStateGoBeforeMount(t *testing.T) {
	skipUnlessDebug(t)
	s := &builderState{}
	requireTag(t, "StateGoBeforeMount", func() { s.Go(func(context.Context) {}) })
}

func TestWrongThread(t *testing.T) {
	skipUnlessDebug(t)
	tr, _ := newThreadedTree(t)
	tr.pump(newBuilder(func(BuildContext, func(func())) Widget { return newLeaf() }))
	el := elementOfType[*builder](t, tr)
	got := make(chan any)
	go func() { got <- panicOf(el.MarkNeedsBuild) }()
	appErr := requireErrorTagFromPanic(t, <-got, "WrongThread")
	if appErr.Metadata["call"] != "Element.MarkNeedsBuild" {
		t.Errorf("call = %v", appErr.Metadata["call"])
	}
	// The UI thread is fine.
	el.MarkNeedsBuild()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGoroutineFailed = "GoroutineFailed"
	tagGoErrWorkFailed = "GoErrWorkFailed"
	tagStopForGood     = "StopForGood"
	tagCanceled        = "Canceled"
)
