package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openListsPage opens the Lists & cards page through the navigation rail.
func openListsPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Lists & cards")))
	ts.PumpAndSettle()
	return ts
}

// TestListsGolden draws the top of the page.
func TestListsGolden(t *testing.T) {
	ts := openListsPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-lists")
}

func TestListsTileTaps(t *testing.T) {
	ts := openListsPage(t)
	if !ts.Exists(gluttertest.ByText("Taps on the first tile: 0")) {
		t.Fatal("the counter is not shown")
	}
	ts.Tap(gluttertest.ByText("One line"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Taps on the first tile: 1")) {
		t.Error("the tile did not count the tap")
	}
}

func TestListsExpansionTile(t *testing.T) {
	ts := openListsPage(t)
	scrollPageDown(ts)
	height := func() float64 {
		return ts.Size(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Collapsed"), gluttertest.ByType[*m.ExpansionTile]()))).Height
	}
	before := height()
	ts.Tap(gluttertest.ByText("Collapsed"))
	ts.PumpAndSettle()
	if height() <= before {
		t.Errorf("the tile did not expand: %v then %v", before, height())
	}
}

func TestListsSelectionTiles(t *testing.T) {
	ts := openListsPage(t)
	scrollPageDown(ts)
	ts.Tap(gluttertest.ByText("Large"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Selected: large")) {
		t.Error("the large radio list tile is not selected")
	}
}

func TestListsTooltipShowsOnHover(t *testing.T) {
	ts := openListsPage(t)
	// Tooltips show on hover after the wait duration.
	ts.Hover(ts.Center(gluttertest.ByText("Hover me")))
	ts.Pump(100 * time.Millisecond)
	if ts.Exists(gluttertest.ByText("Tooltips show after the Material delay")) {
		t.Fatal("the tooltip is shown before the delay")
	}
	ts.Pump(600 * time.Millisecond)
	ts.Pump()
	if !ts.Exists(gluttertest.ByText("Tooltips show after the Material delay")) {
		t.Error("the tooltip is not shown after the delay")
	}
}

// TestListsReorderableListView drags the handle of the first item two rows down
// and checks that the item is below the next two.
func TestListsReorderableListView(t *testing.T) {
	ts := openListsPage(t)
	scrollPageDown(ts)
	scrollPageDown(ts)
	y := func(title string) float64 { return ts.TopLeft(gluttertest.ByText(title)).Dy }
	if y("Item 1") >= y("Item 2") {
		t.Fatalf("Item 1 is at %v and Item 2 at %v at the start", y("Item 1"), y("Item 2"))
	}
	drag := ts.StartGesture(ts.Center(gluttertest.First(gluttertest.ByIcon(m.Icons.DragHandle()))))
	drag.MoveBy(m.Offset{Dy: 20})
	for range 10 {
		drag.MoveBy(m.Offset{Dy: 12})
		ts.Pump(50 * time.Millisecond)
	}
	drag.Up()
	ts.PumpAndSettle()
	if !(y("Item 2") < y("Item 3") && y("Item 3") < y("Item 1") && y("Item 1") < y("Item 4")) {
		t.Errorf("after the drag: item 1 at %v, item 2 at %v, item 3 at %v, item 4 at %v", y("Item 1"), y("Item 2"), y("Item 3"), y("Item 4"))
	}
}
