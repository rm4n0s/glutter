package services

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// These tests are ported from packages/flutter/test/services/predictive_back_event_test.dart;
// the maps of the platform channel are ui.BackGestureEvent values.

func TestPredictiveBackEventFromUI(t *testing.T) {
	tests := []struct {
		name         string
		event        ui.BackGestureEvent
		wantEdge     SwipeEdge
		wantIsButton bool
	}{
		{"swipe edge left", ui.BackGestureEvent{HasTouchOffset: true, TouchY: 100, SwipeEdge: ui.BackSwipeEdgeLeft}, SwipeEdgeLeft, false},
		{"swipe edge right", ui.BackGestureEvent{HasTouchOffset: true, TouchY: 100, SwipeEdge: ui.BackSwipeEdgeRight}, SwipeEdgeRight, false},
		{"isButtonEvent zero position", ui.BackGestureEvent{HasTouchOffset: true, SwipeEdge: ui.BackSwipeEdgeRight}, SwipeEdgeRight, true},
		{"isButtonEvent null position", ui.BackGestureEvent{SwipeEdge: ui.BackSwipeEdgeRight}, SwipeEdgeRight, true},
		{"zero position with progress is a touch", ui.BackGestureEvent{HasTouchOffset: true, Progress: 0.5, SwipeEdge: ui.BackSwipeEdgeLeft}, SwipeEdgeLeft, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := NewPredictiveBackEventFromUI(tc.event)
			if e.SwipeEdge() != tc.wantEdge {
				t.Errorf("SwipeEdge = %v, want %v", e.SwipeEdge(), tc.wantEdge)
			}
			if e.IsButtonEvent() != tc.wantIsButton {
				t.Errorf("IsButtonEvent = %v, want %v", e.IsButtonEvent(), tc.wantIsButton)
			}
			if e.Progress() != tc.event.Progress {
				t.Errorf("Progress = %v, want %v", e.Progress(), tc.event.Progress)
			}
			if off, ok := e.TouchOffset().Get(); ok != tc.event.HasTouchOffset || (ok && off != (ui.Offset{Dx: tc.event.TouchX, Dy: tc.event.TouchY})) {
				t.Errorf("TouchOffset = %v, %v", off, ok)
			}
		})
	}
}

func TestPredictiveBackEventPanicsOnInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		event   ui.BackGestureEvent
		wantTag string
	}{
		{"progress above 1", ui.BackGestureEvent{HasTouchOffset: true, TouchY: 100, Progress: 2, SwipeEdge: ui.BackSwipeEdgeRight}, tagPredictiveBackProgressOutOfRange},
		{"negative progress", ui.BackGestureEvent{Progress: -0.1, SwipeEdge: ui.BackSwipeEdgeRight}, tagPredictiveBackProgressOutOfRange},
		{"unspecified swipe edge", ui.BackGestureEvent{HasTouchOffset: true, TouchY: 100}, tagPredictiveBackSwipeEdgeInvalid},
		{"unknown swipe edge", ui.BackGestureEvent{SwipeEdge: ui.BackSwipeEdge(3)}, tagPredictiveBackSwipeEdgeInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got any
			func() {
				defer func() { got = recover() }()
				NewPredictiveBackEventFromUI(tc.event)
			}()
			err, ok := got.(error)
			if !ok {
				t.Fatalf("recovered %v (%T), want an error", got, got)
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("NewPredictiveBackEventFromUI->NewPredictiveBackEvent." + tc.wantTag) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestPredictiveBackEventEquality(t *testing.T) {
	mk := func(x float64, progress float64, edge SwipeEdge) PredictiveBackEvent {
		return NewPredictiveBackEvent(foundation.Some(ui.Offset{Dx: x, Dy: 100}), progress, edge)
	}
	tests := []struct {
		name      string
		a, b      PredictiveBackEvent
		wantEqual bool
	}{
		{"same parameters", mk(0, 0, SwipeEdgeLeft), mk(0, 0, SwipeEdgeLeft), true},
		{"different touch offset", mk(0, 0, SwipeEdgeLeft), mk(1, 0, SwipeEdgeLeft), false},
		{"different progress", mk(0, 0, SwipeEdgeLeft), mk(0, 0.5, SwipeEdgeLeft), false},
		{"different edge", mk(0, 0, SwipeEdgeLeft), mk(0, 0, SwipeEdgeRight), false},
		{"none and some offset", NewPredictiveBackEvent(foundation.None[ui.Offset](), 0, SwipeEdgeLeft), mk(0, 0, SwipeEdgeLeft), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.wantEqual {
				t.Errorf("Equal = %v, want %v", got, tc.wantEqual)
			}
			if got := tc.a.String() == tc.b.String(); got != tc.wantEqual {
				t.Errorf("String equality = %v, want %v (%s, %s)", got, tc.wantEqual, tc.a, tc.b)
			}
		})
	}
}

func TestPredictiveBackEventString(t *testing.T) {
	tests := []struct {
		name string
		e    PredictiveBackEvent
		want string
	}{
		{"touch", NewPredictiveBackEvent(foundation.Some(ui.Offset{Dy: 100}), 0, SwipeEdgeLeft),
			"PredictiveBackEvent{touchOffset: Offset(0.0, 100.0), progress: 0.0, swipeEdge: SwipeEdge.left}"},
		{"button", NewPredictiveBackEvent(foundation.None[ui.Offset](), 0.35, SwipeEdgeRight),
			"PredictiveBackEvent{touchOffset: null, progress: 0.35, swipeEdge: SwipeEdge.right}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.e.String(); got != tc.want {
				t.Errorf("String = %q, want %q", got, tc.want)
			}
		})
	}
}
