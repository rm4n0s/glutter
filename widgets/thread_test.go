package widgets

import (
	"context"
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
)
