package widgets

import (
	"testing"

	"github.com/rm4n0s/glutter/rendering"
)

// Route tests of the failure tags of step 10.1 that guard the invariants of
// the private overlay machinery (Flutter's asserts). The public ones are tested
// next to the behavior they belong to.

func TestOverlayEntryLocationFailures(t *testing.T) {
	skipUnlessDebug(t)
	box := &renderDeferredLayoutBox{}
	other := &renderDeferredLayoutBox{}
	t.Run("occupied", func(t *testing.T) {
		l := &overlayEntryLocation{overlayChildRenderBox: box}
		requireTag(t, "OverlayLocationOccupied", func() { l.addToChildModel(other) })
		requireTag(t, "OverlayLocationOccupied", func() { l.reattachFromLayoutSurrogate(other) })
	})
	t.Run("occupant mismatch", func(t *testing.T) {
		l := &overlayEntryLocation{overlayChildRenderBox: box}
		requireTag(t, "OverlayLocationOccupantMismatch", func() { l.removeFromChildModel(other) })
	})
	t.Run("invalid", func(t *testing.T) {
		l := &overlayEntryLocation{}
		l.debugMarkLocationInvalid()
		requireTag(t, "OverlayLocationInvalid", func() { l.debugIsLocationValid() })
		requireTag(t, "OverlayLocationInvalid", func() { l.debugMarkLocationInvalid() })
	})
}

func TestOverlayChildModelFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("add to an unmounted entry", func(t *testing.T) {
		s := &overlayEntryWidgetState{}
		requireTag(t, "OverlayChildModelInvalid", func() { s.add(&overlayEntryLocation{}) })
	})
	t.Run("remove a location that is not in the list", func(t *testing.T) {
		s := &overlayEntryWidgetState{}
		requireTag(t, "OverlayChildModelInvalid", func() { s.remove(&overlayEntryLocation{}) })
	})
}

func TestDeferredBoxParentFailures(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("deferred box without a theater", func(t *testing.T) {
		r := &renderDeferredLayoutBox{}
		rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
		requireTag(t, "OverlayDeferredBoxParent", func() { r.theaterOf() })
	})
	t.Run("layout builder without a deferred box", func(t *testing.T) {
		r := &renderOverlayChildLayoutBuilder{}
		rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
		requireTag(t, "OverlayLayoutBuilderParent", func() { r.theaterOf() })
	})
	t.Run("layout info is read before the layout callback", func(t *testing.T) {
		r := &renderOverlayChildLayoutBuilder{}
		requireTag(t, "OverlayChildLayoutInfoMissing", func() { r.layoutInfo() })
	})
}

func TestOverlayPortalElementFailures(t *testing.T) {
	skipUnlessDebug(t)
	e := &overlayPortalElement{}
	requireTag(t, "OverlayPortalForgetChild", func() { e.ForgetChild(&overlayPortalElement{}) })
}

func TestTheaterSkipCountFailure(t *testing.T) {
	skipUnlessDebug(t)
	e := &theaterElement{}
	e.widget = &theater{skipCount: 1}
	requireTag(t, "TheaterSkipCountInvalid", func() { e.DebugVisitOnstageChildren(func(Element) {}) })
}
