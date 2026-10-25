package windows

import (
	"container/heap"
	"time"
)

// timerID names a timer of a timerHeap, to cancel it.
type timerID uint64

// timerItem is one armed timer.
type timerItem struct {
	id    timerID
	at    time.Time
	seq   uint64 // the order of arming, to run the timers of one instant in that order
	fn    func()
	index int
}

// timerQueue is the ordered storage of timerHeap, for container/heap.
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

// timerHeap is the platform's own heap of timers (SPEC §25.3): the loop waits until the earliest
// is due instead of using SetTimer, whose resolution is 10 to 16 ms and whose priority is low. It
// belongs to the UI thread, so it needs no lock.
//
// Flutter: no counterpart (Flutter's embedder has a timer thread that posts WM_NULL,
// shell/platform/windows/task_runner_window.cc TimerThread)
type timerHeap struct {
	q    timerQueue
	byID map[timerID]*timerItem
	next timerID
	seq  uint64
}

// add arms fn to run at the time and returns the id to cancel it with.
func (h *timerHeap) add(at time.Time, fn func()) timerID {
	if h.byID == nil {
		h.byID = map[timerID]*timerItem{}
	}
	h.next++
	h.seq++
	it := &timerItem{id: h.next, at: at, seq: h.seq, fn: fn}
	h.byID[it.id] = it
	heap.Push(&h.q, it)
	return it.id
}

// cancel disarms a timer and reports whether it was still armed.
func (h *timerHeap) cancel(id timerID) bool {
	it, ok := h.byID[id]
	if !ok {
		return false
	}
	heap.Remove(&h.q, it.index)
	delete(h.byID, id)
	return true
}

// deadline is when the earliest timer is due; ok is false when none is armed, which makes the
// loop wait for ever.
func (h *timerHeap) deadline() (at time.Time, ok bool) {
	if len(h.q) == 0 {
		return time.Time{}, false
	}
	return h.q[0].at, true
}

// runDue runs the timers that are due at now, the earliest first and those of one instant in the
// order they were armed, and returns how many ran. A function may arm and cancel timers; one it
// arms for a time that is already due also runs.
func (h *timerHeap) runDue(now time.Time) int {
	n := 0
	for len(h.q) > 0 && !h.q[0].at.After(now) {
		it := heap.Pop(&h.q).(*timerItem)
		delete(h.byID, it.id)
		it.fn()
		n++
	}
	return n
}

// waitFor is how long the loop may wait from now: the time to the earliest timer (0 when it is
// due), and false when none is armed.
func (h *timerHeap) waitFor(now time.Time) (d time.Duration, ok bool) {
	at, ok := h.deadline()
	if !ok {
		return 0, false
	}
	return max(at.Sub(now), 0), true
}
