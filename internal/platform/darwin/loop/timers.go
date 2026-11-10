package loop

import (
	"container/heap"
	"time"
)

// TimerID names a timer of a Timers, to cancel it.
type TimerID uint64

// timerItem is one armed timer.
type timerItem struct {
	id    TimerID
	at    time.Time
	seq   uint64 // the order of arming, to run the timers of one instant in that order
	fn    func()
	index int
}

// timerQueue is the ordered storage of Timers, for container/heap.
type timerQueue []*timerItem

func (q timerQueue) Len() int { return len(q) }
func (q timerQueue) Less(i, j int) bool {
	if !q[i].at.Equal(q[j].at) {
		return q[i].at.Before(q[j].at)
	}
	return q[i].seq < q[j].seq
}
func (q timerQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index, q[j].index = i, j
}
func (q *timerQueue) Push(x any) {
	it := x.(*timerItem)
	it.index = len(*q)
	*q = append(*q, it)
}
func (q *timerQueue) Pop() any {
	old := *q
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*q = old[:n-1]
	return it
}

// Timers is the platform's own heap of timers (SPEC §29.3): one CFRunLoopTimer is set to the
// earliest deadline (SetNextFire) and the heap runs what is due when it fires. It belongs to the UI
// thread, so it needs no lock.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterRunLoop.swift FlutterRunLoop (its
// scheduled blocks run in the main run loop)
type Timers struct {
	q    timerQueue
	byID map[TimerID]*timerItem
	next TimerID
	seq  uint64
}

// Add arms fn to run at the time and returns the id to cancel it with.
//
// Flutter: no counterpart
func (h *Timers) Add(at time.Time, fn func()) TimerID {
	if h.byID == nil {
		h.byID = map[TimerID]*timerItem{}
	}
	h.next++
	h.seq++
	it := &timerItem{id: h.next, at: at, seq: h.seq, fn: fn}
	h.byID[it.id] = it
	heap.Push(&h.q, it)
	return it.id
}

// Cancel disarms a timer and reports whether it was still armed.
//
// Flutter: no counterpart
func (h *Timers) Cancel(id TimerID) bool {
	it, ok := h.byID[id]
	if !ok {
		return false
	}
	heap.Remove(&h.q, it.index)
	delete(h.byID, id)
	return true
}

// Deadline is when the earliest timer is due; ok is false when none is armed, which makes the
// loop wait for ever.
//
// Flutter: no counterpart
func (h *Timers) Deadline() (at time.Time, ok bool) {
	if len(h.q) == 0 {
		return time.Time{}, false
	}
	return h.q[0].at, true
}

// RunDue runs the timers that are due at now, the earliest first and those of one instant in the
// order they were armed, and returns how many ran. A function may arm and cancel timers; one it
// arms for a time that is already due also runs.
//
// Flutter: no counterpart
func (h *Timers) RunDue(now time.Time) int {
	n := 0
	for len(h.q) > 0 && !h.q[0].at.After(now) {
		it := heap.Pop(&h.q).(*timerItem)
		delete(h.byID, it.id)
		it.fn()
		n++
	}
	return n
}

// WaitFor is how long the loop may wait from now: the time to the earliest timer (0 when it is
// due), and false when none is armed.
//
// Flutter: no counterpart
func (h *Timers) WaitFor(now time.Time) (d time.Duration, ok bool) {
	at, ok := h.Deadline()
	if !ok {
		return 0, false
	}
	return max(at.Sub(now), 0), true
}
