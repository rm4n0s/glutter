package widgets_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// The widget tests of the scrolling step (9.2) run on a scrollable made of
// what step 9.2 has: a Scrollable, a Viewport and one sliver per child. The
// ListView, SingleChildScrollView and the sliver widgets come with step 9.3;
// the Dart tests that need their lazy building wait for it.

// sliverBox is a SliverToBoxAdapter: a sliver with one box child.
type sliverBox struct {
	SingleChildRenderObjectWidget
}

func newSliverBox(child Widget, opts ...WidgetOpts) *sliverBox {
	return &sliverBox{NewSingleChildRenderObjectWidgetBase(child, opts...)}
}

func (w *sliverBox) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderSliverToBoxAdapter(nil)
}

// listOpts are what testList takes besides the scrollable's options.
type listOpts struct {
	ScrollableOpts
	crossAxisDirection painting.AxisDirection
	anchor             float64
	center             foundation.Key
}

// testList is a ListView of the children: a Scrollable with a Viewport whose
// slivers are one sliver box per child.
func testList(children []Widget, opts ...listOpts) Widget {
	var o listOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	slivers := make([]Widget, len(children))
	for i, c := range children {
		slivers[i] = newSliverBox(c)
	}
	return NewScrollable(func(_ BuildContext, position rendering.ViewportOffset) Widget {
		return NewViewport(position, slivers, ViewportOpts{AxisDirection: o.AxisDirection, CrossAxisDirection: o.crossAxisDirection,
			Anchor: o.anchor, Center: o.center})
	}, o.ScrollableOpts)
}

// items are n SizedBoxes of the height, each with a Text "Item i" (the states
// of Flutter's states.dart in the Dart tests).
func items(n int, height float64) []Widget {
	out := make([]Widget, n)
	for i := range out {
		out[i] = NewSizedBox(NewText(fmt.Sprintf("Item %d", i)), SizedBoxOpts{Height: foundation.Some(height)})
	}
	return out
}

// itemFinder finds the SizedBox of "Item i".
func itemFinder(i int) gluttertest.Finder {
	return gluttertest.Ancestor(gluttertest.ByText(fmt.Sprintf("Item %d", i)), gluttertest.ByType[*SizedBox]())
}

func scrollableOf(ts *gluttertest.Tester) *ScrollableState {
	return gluttertest.StateOf[*ScrollableState](ts, gluttertest.ByType[*Scrollable]())
}

func realOffset(ts *gluttertest.Tester) float64 { return scrollableOf(ts).Position().Pixels() }

func requirePanic(t *testing.T, call func()) {
	t.Helper()
	skipUnlessDebug(t) // the checks are debug-only
	if panicOf(call) == nil {
		t.Fatal("expected a panic")
	}
}

func stringsContain(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

// useAndroid makes the platform Android for the test, as it is in Flutter's
// tests: the scrollables have no scrollbar and the clamping physics.
func useAndroid(t *testing.T) {
	t.Helper()
	skipUnlessDebug(t) // the platform override is a debug-only feature
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}
