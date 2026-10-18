package engine

// Frames (SPEC §8.11, "Frame pacing"): a window draws at most one frame at a time, and
// the input that arrives between two frames is not handled at once: it is queued, and
// the next frame handles all of it before it builds its scene. A burst of pointer
// moves, or a key and the click after it, then costs one frame, which already shows
// all of them. Handling the input at the frame's start also gives it the frame's
// time, as Flutter's input resampling does.

// inputQueue holds what arrived since the last frame, in the order it arrived.
type inputQueue struct {
	pending []func()
}

// add queues an event's handling.
func (q *inputQueue) add(handle func()) { q.pending = append(q.pending, handle) }

// flush handles everything that is queued, in order, and empties the queue. What the
// handling queues in turn waits for the next frame.
func (q *inputQueue) flush() {
	pending := q.pending
	q.pending = nil
	for _, handle := range pending {
		handle()
	}
}
