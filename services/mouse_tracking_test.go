package services

import (
	"testing"

	"github.com/rm4n0s/glutter/gestures"
)

func TestMouseTrackerAnnotationDefaults(t *testing.T) {
	a := NewMouseTrackerAnnotation()
	if a.OnEnter() != nil || a.OnExit() != nil || a.Cursor() != MouseCursorDefer || !a.ValidForMouseTracker() {
		t.Errorf("unexpected defaults: %+v", a)
	}
	entered := false
	b := NewMouseTrackerAnnotation(MouseTrackerAnnotationOpts{
		OnEnter: func(*gestures.PointerEnterEvent) { entered = true }, Cursor: MouseCursorUncontrolled, Invalid: true,
	})
	b.OnEnter()(nil)
	if !entered || b.Cursor() != MouseCursorUncontrolled || b.ValidForMouseTracker() {
		t.Errorf("unexpected values: %+v", b)
	}
}
