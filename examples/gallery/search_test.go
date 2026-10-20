package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

// openSearchPage opens the search page through the navigation rail.
func openSearchPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Search"))
	ts.PumpAndSettle()
	return ts
}

// TestSearchGolden draws the search page in the light theme.
func TestSearchGolden(t *testing.T) {
	ts := openSearchPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-search")
}

// TestSearchViewGolden draws the search view that GLUTTER_DEV_QUERY opens.
func TestSearchViewGolden(t *testing.T) {
	t.Setenv("GLUTTER_DEV_QUERY", "gr")
	ts := openSearchPage(t)
	ts.Pump(time.Second)
	ts.PumpAndSettle()
	for _, want := range []string{"Greece", "Grenada", "Guatemala"} {
		if want == "Guatemala" {
			continue
		}
		if !ts.Exists(gluttertest.ByText(want)) {
			t.Errorf("the country %q is not in the view", want)
		}
	}
	if ts.Exists(gluttertest.ByText("Egypt")) {
		t.Error("a country that does not match is in the view")
	}
	ts.MatchesGolden("gallery-search-view")
}

func TestSearchViewSelectsACountry(t *testing.T) {
	ts := openSearchPage(t)
	ts.Tap(gluttertest.First(gluttertest.ByType[*m.SearchBar]()))
	ts.PumpAndSettle()
	ts.EnterText(gluttertest.Last(gluttertest.ByType[*m.SearchBar]()), "sw")
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Sweden")) || !ts.Exists(gluttertest.ByText("Switzerland")) {
		t.Fatal("the countries with sw are not suggested")
	}
	ts.Tap(gluttertest.ByText("Sweden"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Switzerland")) {
		t.Error("the view is still open")
	}
}

func TestSearchAutocomplete(t *testing.T) {
	ts := openSearchPage(t)
	ts.EnterText(gluttertest.ByType[*m.TextFormField](), "po")
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Poland")) || !ts.Exists(gluttertest.ByText("Portugal")) {
		t.Fatal("the countries with po are not suggested")
	}
	ts.Tap(gluttertest.ByText("Portugal"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Selected: Portugal")) {
		t.Error("the selection is not shown")
	}
}

func TestSearchBarSubmit(t *testing.T) {
	ts := openSearchPage(t)
	bar := gluttertest.At(gluttertest.ByType[*m.SearchBar](), 1)
	ts.EnterText(bar, "hello")
	ts.PumpAndSettle()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Submitted: hello")) {
		t.Error("the submitted text is not shown")
	}
}
