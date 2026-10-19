package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/gestures"
)

// The checks of the scrolling classes that the public API cannot reach: they
// guard the invariants of the code itself.

type stubDrag struct{}

func (stubDrag) Update(gestures.DragUpdateDetails) {}
func (stubDrag) End(gestures.DragEndDetails)       {}
func (stubDrag) Cancel()                           {}

func TestScrollableEndedDragMustBeDisposed(t *testing.T) {
	skipUnlessDebug(t)
	s := &ScrollableState{drag: stubDrag{}}
	requireTag(t, "ScrollableDragStateInvalid", func() { s.handleDragEnd(gestures.DragEndDetails{}) })
}

func TestViewportElementMustNotBeReentered(t *testing.T) {
	skipUnlessDebug(t)
	e := &viewportElement{doingMountOrUpdate: true}
	requireTag(t, "ViewportReentered", func() { e.Mount(nil, nil) })
	requireTag(t, "ViewportReentered", func() { e.Update(nil) })
}

func TestScrollPositionAbsorbsOneDragOnly(t *testing.T) {
	skipUnlessDebug(t)
	_ = requireErrorTag(t, scrollAbsorbDragError(), "ScrollPositionDragInvalid")
}

func TestRawScrollbarDismissedFadeMustBeTransparent(t *testing.T) {
	skipUnlessDebug(t)
	s := &RawScrollbarState{fadeoutOpacityAnimation: animation.NewCurvedAnimation(animation.NewAlwaysStoppedAnimation(0.5), animation.Curves.Linear)}
	requireTag(t, "RawScrollbarFadeInvalid", func() { s.validateInteractions(animation.AnimationStatusDismissed) })
	s.fadeoutOpacityAnimation = animation.NewCurvedAnimation(animation.NewAlwaysStoppedAnimation(0.0), animation.Curves.Linear)
	s.validateInteractions(animation.AnimationStatusDismissed) // fully faded out is valid
}
