package scheduler

import (
	"slices"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// TimingsCallback is called with the timings of the frames that the engine drew since
// the last call.
//
// Flutter: dart:ui TimingsCallback
type TimingsCallback func(timings []ui.FrameTiming)

// TimingsHandle identifies a callback added with AddTimingsCallback, to remove it.
//
// Flutter: no counterpart
type TimingsHandle struct{ id int }

type timingsEntry struct {
	id       int
	callback TimingsCallback
}

// AddTimingsCallback adds a callback for the timings of the frames. The engine
// measures them only while a callback is set, so adding one has a cost: remove it when
// the timings are not used any more. A callback that panics is reported to the
// ErrorReporter (CallbackPanicked, "while reporting frame timings") and the others
// still run.
//
// Deviation: Flutter removes a callback by the function; functions are not comparable
// in Go, so the add gives a handle (SPEC §12.10).
//
// Flutter: scheduler/binding.dart SchedulerBinding.addTimingsCallback
func (b *SchedulerBinding) AddTimingsCallback(callback TimingsCallback) TimingsHandle {
	b.nextTimingsID++
	b.timingsCallbacks = append(b.timingsCallbacks, timingsEntry{id: b.nextTimingsID, callback: callback})
	if len(b.timingsCallbacks) == 1 {
		b.d.OnReportTimings = b.executeTimingsCallbacks
	}
	return TimingsHandle{id: b.nextTimingsID}
}

// RemoveTimingsCallback removes the callback; a handle that was removed already does
// nothing. When the last one is removed the engine stops measuring.
//
// Flutter: scheduler/binding.dart SchedulerBinding.removeTimingsCallback
func (b *SchedulerBinding) RemoveTimingsCallback(handle TimingsHandle) {
	b.timingsCallbacks = slices.DeleteFunc(b.timingsCallbacks, func(e timingsEntry) bool { return e.id == handle.id })
	if len(b.timingsCallbacks) == 0 {
		b.d.OnReportTimings = nil
	}
}

func (b *SchedulerBinding) executeTimingsCallbacks(timings []ui.FrameTiming) {
	for _, e := range slices.Clone(b.timingsCallbacks) {
		foundation.Guard(b.ErrorReporter, foundation.CallbackPanicked, "while reporting frame timings", nil, func() { e.callback(timings) })
	}
}
