package web

import (
	"context"
	"sync"
	"sync/atomic"
)

// queue is the queue of the UI goroutine: the browser's callbacks and Platform.Post
// put functions in it, and Platform.Run runs them in order (SPEC §23.3). A callback
// copies what it needs out of the event into Go values and posts a function, then
// returns; Go's wasm runtime runs the UI goroutine, which is ready by then, before it
// goes back to the browser, so the function runs in the same browser task.
//
// Flutter: no counterpart
type queue struct {
	mu    sync.Mutex
	items []func()
	wake  chan struct{} // capacity 1: a pending wakeup for run
	// active is whether run is in the middle of a function.
	active atomic.Bool
}

func newQueue() *queue { return &queue{wake: make(chan struct{}, 1)} }

// post adds fn to the queue and wakes run. It is safe to call from any goroutine.
// Functions posted by one goroutine run in the order that goroutine posted them.
func (q *queue) post(fn func()) {
	q.mu.Lock()
	q.items = append(q.items, fn)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default: // a wakeup is already pending
	}
}

// pop removes and returns the oldest function.
func (q *queue) pop() (func(), bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	fn := q.items[0]
	q.items[0] = nil
	q.items = q.items[1:]
	return fn, true
}

// run runs the queued functions, one at a time, until ctx is done. Functions that are
// queued when ctx ends are not run.
func (q *queue) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if fn, ok := q.pop(); ok {
			q.active.Store(true)
			fn()
			q.active.Store(false)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
		}
	}
}

// busy reports whether run is in the middle of a function. A browser callback that finds it so
// was called from inside that function (the function dispatched an event, and the browser
// runs the listener at once, on the goroutine that dispatched it), and must not wait for the
// queue, which only that function can serve.
func (q *queue) busy() bool { return q.active.Load() }
