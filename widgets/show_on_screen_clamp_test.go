package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestShowOnScreenClampsToTheScrollExtents checks that showing a rect that
// reaches past the scroll extents moves the scroll position only as far as the
// extents (Flutter's RenderViewportBase.showInViewport calls offset.moveTo, which
// ScrollPosition.moveTo clamps). A view whose content fits starts no animation, so
// the scrollable does not ignore the pointers that come during it: before the
// clamp a focused text field's caret, with its scroll padding, started one and
// the first tap beside the field in a scrolling dialog was lost.
func TestShowOnScreenClampsToTheScrollExtents(t *testing.T) {
	const viewHeight = 300.0
	tests := []struct {
		name string
		// list is a ListView (a RenderViewport) instead of a SingleChildScrollView.
		list bool
		// content is the height of the box in the view.
		content float64
		// rect is what is shown, in the box's coordinates.
		rect ui.Rect
		// want is the position at the end; scrolls is whether it got there with an
		// animation.
		want    float64
		scrolls bool
	}{
		{name: "fits, above the top", content: 100, rect: ui.NewRectFromLTRB(0, -20, 100, 40), want: 0},
		{name: "fits, below the bottom", content: 100, rect: ui.NewRectFromLTRB(0, 60, 100, viewHeight+20), want: 0},
		{name: "overflows, above the top", content: 1000, rect: ui.NewRectFromLTRB(0, -20, 100, 40), want: 0},
		{name: "overflows, past the end", content: 1000, rect: ui.NewRectFromLTRB(0, 980, 100, 1020), want: 1000 - viewHeight, scrolls: true},
		{name: "list fits, above the top", list: true, content: 100, rect: ui.NewRectFromLTRB(0, -20, 100, 40), want: 0},
		{name: "list overflows, past the end", list: true, content: 1000, rect: ui.NewRectFromLTRB(0, 980, 100, 1020), want: 1000 - viewHeight, scrolls: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: viewHeight}})
			controller := NewScrollController()
			key := foundation.NewValueKey("box")
			box := NewSizedBox(nil, SizedBoxOpts{Key: key, Height: foundation.Some(tt.content)})
			var view Widget = NewSingleChildScrollView(box, SingleChildScrollViewOpts{Controller: controller})
			if tt.list {
				view = NewListView([]Widget{box}, ListViewOpts{Controller: controller})
			}
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, view))
			position := controller.Position()
			scrolled := false
			position.IsScrollingNotifier().AddListener(func() { scrolled = scrolled || position.IsScrollingNotifier().Value() })

			ts.RenderObject(gluttertest.ByKey(key)).ShowOnScreen(rendering.ShowOnScreenOpts{Rect: foundation.Some(tt.rect), Duration: 100 * time.Millisecond})
			ts.Pump()
			if !tt.scrolls && position.IsScrollingNotifier().Value() {
				t.Errorf("the position is scrolling (ignoring pointers) after showing a rect it cannot scroll to")
			}
			ts.PumpAndSettle()
			if got := position.Pixels(); got != tt.want {
				t.Errorf("pixels = %v, want %v", got, tt.want)
			}
			if scrolled != tt.scrolls {
				t.Errorf("scrolled = %v, want %v", scrolled, tt.scrolls)
			}
		})
	}
}
