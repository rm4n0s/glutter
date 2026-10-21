package web

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestQueueRunsInOrderAndStopsWithTheContext(t *testing.T) {
	q := newQueue()
	ctx, cancel := context.WithCancel(context.Background())
	var got []int
	for i := range 5 {
		q.post(func() { got = append(got, i) })
	}
	// A function that posts another one: the second runs after everything queued before.
	q.post(func() { q.post(func() { got = append(got, 99); cancel() }) })
	q.post(func() { got = append(got, 5) })
	q.run(ctx)
	if want := []int{0, 1, 2, 3, 4, 5, 99}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestQueueWakesFromAnotherGoroutine(t *testing.T) {
	q := newQueue()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.run(ctx)
	}()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var got []int
	wg.Add(3)
	for i := range 3 {
		q.post(func() {
			mu.Lock()
			got = append(got, i)
			mu.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after the context ended")
	}
	if !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("ran %v, want [0 1 2]", got)
	}
}

func TestQueueDoesNotRunAfterCancel(t *testing.T) {
	q := newQueue()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	q.post(func() { ran = true })
	q.run(ctx)
	if ran {
		t.Error("a function ran after the context ended")
	}
}
