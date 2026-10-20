package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openDataPage opens the Data & progress page through the navigation rail.
func openDataPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Data & progress")))
	ts.PumpAndSettle()
	return ts
}

// TestDataGolden draws the top of the page.
func TestDataGolden(t *testing.T) {
	ts := openDataPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-data")
}

func TestDataTableSortsAndSelects(t *testing.T) {
	ts := openDataPage(t)
	first := func() float64 { return ts.Center(gluttertest.ByText("Cupcake")).Dy }
	before := first()
	ts.Tap(gluttertest.ByText("Calories"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("Calories"))
	ts.PumpAndSettle()
	if first() >= before {
		t.Errorf("Cupcake did not move up when sorted by calories, descending: %v then %v", before, first())
	}
	ts.Tap(gluttertest.ByText("Eclair"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Selected rows: 1")) {
		t.Error("the row was not selected")
	}
}

func TestDataProgressSwitch(t *testing.T) {
	ts := openDataPage(t)
	if n := ts.Count(gluttertest.ByType[*m.LinearProgressIndicator]()); n != 2 {
		t.Fatalf("%d linear indicators, want 2", n)
	}
	ts.Tap(gluttertest.ByType[*m.Switch]())
	ts.Pump(100 * time.Millisecond)
	// The indeterminate indicators animate for ever; the page is still there.
	if !ts.Exists(gluttertest.ByType[*m.CircularProgressIndicator]()) {
		t.Error("the circular indicator is gone")
	}
}

func TestDataStepperContinue(t *testing.T) {
	ts := openDataPage(t)
	scrollPageDown(ts)
	ts.Tap(gluttertest.First(gluttertest.ByText("Continue")))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Current step: 2 of 3")) {
		t.Error("the stepper did not open the second step")
	}
}

func TestDataPaginatedTablePages(t *testing.T) {
	ts := openDataPage(t)
	for range 2 {
		ts.Drag(gluttertest.First(gluttertest.ByType[*m.ListView]()), m.Offset{Dx: 0, Dy: -1000})
		ts.PumpAndSettle()
	}
	if !ts.Exists(gluttertest.ByText("Dessert 1")) || ts.Exists(gluttertest.ByText("Dessert 6")) {
		t.Fatal("the first page does not show the first five rows")
	}
	ts.Tap(gluttertest.ByTooltip("Next page"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Dessert 6")) {
		t.Error("the next page does not show the sixth row")
	}
}
