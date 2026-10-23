package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
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
	requireTag(t, "ScrollableDragStateInvalid", func() { s.baseDragEnd(gestures.DragEndDetails{}) })
}

func TestViewportElementMustNotBeReentered(t *testing.T) {
	skipUnlessDebug(t)
	e := &viewportElement{doingMountOrUpdate: true}
	requireTag(t, "ViewportReentered", func() { e.Mount(nil, nil) })
	requireTag(t, "ViewportReentered", func() { e.Update(nil) })
}

func TestRawScrollbarDismissedFadeMustBeTransparent(t *testing.T) {
	skipUnlessDebug(t)
	s := &RawScrollbarState{fadeoutOpacityAnimation: animation.NewCurvedAnimation(animation.NewAlwaysStoppedAnimation(0.5), animation.Curves.Linear)}
	requireTag(t, "RawScrollbarFadeInvalid", func() { s.validateInteractions(animation.AnimationStatusDismissed) })
	s.fadeoutOpacityAnimation = animation.NewCurvedAnimation(animation.NewAlwaysStoppedAnimation(0.0), animation.Curves.Linear)
	s.validateInteractions(animation.AnimationStatusDismissed) // fully faded out is valid
}

// stubScrollContext is a scroll context that does nothing: the methods that the tests do not reach are
// those of the nil interface and panic.
type stubScrollContext struct {
	ScrollContext
	storage BuildContext
}

func (c stubScrollContext) StorageContext() BuildContext                      { return c.storage }
func (stubScrollContext) SetIgnorePointer(bool)                               {}
func (stubScrollContext) SetCanDrag(bool)                                     {}
func (stubScrollContext) SetSemanticsActions(map[ui.SemanticsAction]struct{}) {}

// TestScrollPositionAbsorbsOneDragOnly: a position that has a drag cannot absorb another. The public API
// never gets there, since beginning an activity ends the drag, so the position is made by hand.
func TestScrollPositionAbsorbsOneDragOnly(t *testing.T) {
	skipUnlessDebug(t)
	var bc BuildContext
	newTree(t).pump(newBuilder(func(b BuildContext, _ func(func())) Widget { bc = b; return newLeaf() }))
	context := stubScrollContext{storage: bc}
	old := NewScrollPositionWithSingleContext(NewClampingScrollPhysics(nil), context)
	fresh := NewScrollPositionWithSingleContext(NewClampingScrollPhysics(nil), context, ScrollPositionWithSingleContextOpts{NoInitialPixels: true})
	fresh.activity = nil
	fresh.currentDrag = &ScrollDragController{}
	requireTag(t, "ScrollPositionWithSingleContext.Absorb.ScrollPositionDragInvalid", func() { fresh.Absorb(old) })
}
