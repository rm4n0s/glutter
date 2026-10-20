package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openSelectionPage opens the selection page through the navigation rail.
func openSelectionPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Selection & chips"))
	ts.PumpAndSettle()
	return ts
}

// TestSelectionGolden draws the top of the selection page in the light theme.
func TestSelectionGolden(t *testing.T) {
	ts := openSelectionPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-selection")
}

// TestSelectionDarkGolden draws the page in the dark theme.
func TestSelectionDarkGolden(t *testing.T) {
	ts := openSelectionPage(t)
	ts.Tap(gluttertest.ByIcon(m.Icons.DarkMode()))
	ts.PumpAndSettle()
	ts.MatchesGolden("gallery-selection-dark")
}

func TestSelectionRadioGroup(t *testing.T) {
	ts := openSelectionPage(t)
	if !ts.Exists(gluttertest.ByText("Selected: tea")) {
		t.Fatal("the group does not start with tea")
	}
	ts.Tap(gluttertest.ByText("Coffee"))
	ts.PumpAndSettle()
	// The label is not part of the radio: tap the radio itself.
	radios := gluttertest.ByType[*m.Radio[string]]()
	ts.Tap(gluttertest.At(radios, 1))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Selected: coffee")) {
		t.Error("the radio did not select coffee")
	}
}

func TestSelectionInputChipDelete(t *testing.T) {
	ts := openSelectionPage(t)
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Grace")) {
		t.Fatal("the chip of Grace is not shown")
	}
	ts.Tap(gluttertest.First(gluttertest.Descendant(gluttertest.Ancestor(gluttertest.ByText("Grace"), gluttertest.ByType[*m.InputChip]()),
		gluttertest.ByIcon(m.Icons.Clear()))))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Grace")) {
		t.Error("the chip of Grace is still shown after it was deleted")
	}
	if !ts.Exists(gluttertest.ByText("Ada")) {
		t.Error("the other chips went away too")
	}
}

func TestSelectionFilterAndActionChips(t *testing.T) {
	ts := openSelectionPage(t)
	filter := gluttertest.Ancestor(gluttertest.ByText("Gluten free"), gluttertest.ByType[*m.FilterChip]())
	if gluttertest.WidgetAs[*m.FilterChip](ts, filter).IsEnabled() == false {
		t.Fatal("the filter chip is disabled")
	}
	ts.Tap(gluttertest.ByText("Gluten free"))
	ts.PumpAndSettle()
	scrollPageDown(ts)
	ts.Tap(gluttertest.ByText("Add"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Pressed 1 times")) {
		t.Error("the action chip did not count the press")
	}
}
