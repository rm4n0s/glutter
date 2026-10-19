package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/drag_boundary_test.dart.

func dragBoundaryApp(key foundation.Key, withBoundary bool) Widget {
	var child Widget = NewSizedBox(nil, SizedBoxOpts{Key: key, Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
	if withBoundary {
		child = NewDragBoundary(child)
	}
	return ltr(NewContainer(child, ContainerOpts{Margin: painting.NewEdgeInsetsFromLTRB(100, 100, 0, 0), Alignment: painting.AlignmentTopLeft}))
}

func TestDragBoundary(t *testing.T) {
	rect := func(l, t, w, h float64) ui.Rect { return ui.NewRectFromLTWH(l, t, w, h) }
	tests := []struct {
		name         string
		withBoundary bool
		local        bool
		within       map[ui.Rect]bool
		nearest      map[ui.Rect]ui.Rect
	}{
		{"with global position", true, false,
			map[ui.Rect]bool{rect(50, 50, 20, 20): false, rect(100, 100, 20, 20): true},
			map[ui.Rect]ui.Rect{rect(50, 50, 20, 20): rect(100, 100, 20, 20), rect(150, 150, 20, 20): rect(150, 150, 20, 20)}},
		{"without global position", true, true,
			map[ui.Rect]bool{rect(50, 50, 20, 20): true, rect(90, 90, 20, 20): false},
			map[ui.Rect]ui.Rect{rect(50, 50, 20, 20): rect(50, 50, 20, 20), rect(90, 90, 20, 20): rect(80, 80, 20, 20)}},
		{"free boundary when no ancestor has a DragBoundary", false, false,
			map[ui.Rect]bool{rect(50, 50, 20, 20): true, rect(100, 100, 20, 20): true, rect(300, 300, 300, 300): true},
			map[ui.Rect]ui.Rect{rect(50, 50, 20, 20): rect(50, 50, 20, 20), rect(150, 150, 20, 20): rect(150, 150, 20, 20),
				rect(300, 300, 300, 300): rect(300, 300, 300, 300)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			key := NewGlobalKey[State]()
			ts.PumpWidget(dragBoundaryApp(key, tc.withBoundary))
			boundary := DragBoundaryForRectOf(key.CurrentContext(), DragBoundaryForRectOfOpts{UseLocalPosition: tc.local})
			if boundary == nil {
				t.Fatal("no boundary")
			}
			for r, want := range tc.within {
				if got := boundary.IsWithinBoundary(r); got != want {
					t.Errorf("IsWithinBoundary(%v) = %v, want %v", r, got, want)
				}
			}
			for r, want := range tc.nearest {
				if got := boundary.NearestPositionWithinBoundary(r); got != want {
					t.Errorf("NearestPositionWithinBoundary(%v) = %v, want %v", r, got, want)
				}
			}
		})
	}
}
