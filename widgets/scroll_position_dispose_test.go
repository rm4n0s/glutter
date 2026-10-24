package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// unmountingContext is the scroll context of a scrollable whose state leaves the
// tree: asking for its storage context then panics, as ScrollableState does in
// debug builds, and it counts the notification contexts that were asked for.
type unmountingContext struct {
	*ScrollableState
	unmounted     *bool
	notifications *int
}

func (c unmountingContext) StorageContext() BuildContext {
	if *c.unmounted {
		panic("the storage context of an unmounted state was asked for")
	}
	return c.ScrollableState.StorageContext()
}

// The gesture detector accepts these in the layout phase only.
func (unmountingContext) SetCanDrag(bool)                                     {}
func (unmountingContext) SetIgnorePointer(bool)                               {}
func (unmountingContext) SetSemanticsActions(map[ui.SemanticsAction]struct{}) {}

func (c unmountingContext) NotificationContext() BuildContext {
	*c.notifications++
	return c.ScrollableState.NotificationContext()
}

// What a position posted, or does after Dispose, never reaches the storage
// context of a scrollable that left the tree (step 27.1). gluttertest runs the
// posted functions inside Pump, so the position here is made by hand and disposed
// before the Pump that runs what it posted.
func TestScrollPositionAfterDispose(t *testing.T) {
	tests := []struct {
		name  string
		after func(p *ScrollPositionWithSingleContext)
	}{
		{"posted DidUpdateScrollMetrics", func(*ScrollPositionWithSingleContext) {}},
		{"SaveScrollOffset", func(p *ScrollPositionWithSingleContext) { p.SaveScrollOffset() }},
		{"SaveScrollOffset and RestoreScrollOffset", func(p *ScrollPositionWithSingleContext) {
			p.SaveScrollOffset()
			p.RestoreScrollOffset()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(testList(items(1, 100))))
			var unmounted bool
			var notifications int
			context := unmountingContext{ScrollableState: scrollableOf(ts), unmounted: &unmounted, notifications: &notifications}
			p := NewScrollPositionWithSingleContext(NewClampingScrollPhysics(nil), context)
			// A change of the metrics during layout posts DidUpdateScrollMetrics.
			p.ApplyViewportDimension(100)
			p.ApplyContentDimensions(0, 500)
			p.Dispose()
			unmounted = true
			tc.after(p)
			ts.Pump()
			if notifications != 0 {
				t.Errorf("the disposed position asked for the notification context %d times", notifications)
			}
		})
	}
}
