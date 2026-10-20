package foundation

import (
	"context"
	"runtime/trace"
)

// TraceRegion is a span of the runtime execution trace (go tool trace, the
// "User-defined regions" view): the framework wraps the phases of a frame in
// regions named as Flutter's timeline events are (BUILD, LAYOUT, PAINT, and so on).
// A region costs one atomic load while no trace is being recorded.
//
// Flutter: dart:developer Timeline.startSync
type TraceRegion struct {
	r *trace.Region
}

// StartTraceRegion starts a region with the name on the calling goroutine. End it on
// the same goroutine, usually with
//
//	defer foundation.StartTraceRegion("LAYOUT").End()
//
// Flutter: dart:developer Timeline.startSync
func StartTraceRegion(name string) TraceRegion {
	if !trace.IsEnabled() {
		return TraceRegion{}
	}
	return TraceRegion{r: trace.StartRegion(context.Background(), name)}
}

// End ends the region; ending the region of a trace that was not recording does
// nothing.
//
// Flutter: dart:developer TimelineTask.finish
func (r TraceRegion) End() {
	if r.r != nil {
		r.r.End()
	}
}

// TraceEnabled says whether a trace is being recorded, so that the arguments of a
// TraceLog are made only when they will be kept.
//
// Flutter: no counterpart
func TraceEnabled() bool { return trace.IsEnabled() }

// TraceLog adds a message with a category to the trace, if one is recording.
//
// Flutter: dart:developer Timeline.instantSync
func TraceLog(category, message string) {
	if trace.IsEnabled() {
		trace.Log(context.Background(), category, message)
	}
}
