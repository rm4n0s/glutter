package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// openButtonsPage opens the third page through the navigation rail.
func openButtonsPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Buttons")))
	ts.PumpAndSettle()
	return ts
}

// TestButtonsGolden draws the top of the third page in the light theme.
func TestButtonsGolden(t *testing.T) {
	ts := openButtonsPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-buttons")
}

// TestButtonsPageVariants checks that the page shows every kind of button.
func TestButtonsPageVariants(t *testing.T) {
	ts := openButtonsPage(t)
	tests := []struct {
		name  string
		exist func() bool
	}{
		{"text", func() bool { return ts.Exists(gluttertest.ByType[*m.TextButton]()) }},
		{"elevated", func() bool { return ts.Exists(gluttertest.ByType[*m.ElevatedButton]()) }},
		{"filled", func() bool { return ts.Exists(gluttertest.ByType[*m.FilledButton]()) }},
		{"outlined", func() bool { return ts.Exists(gluttertest.ByType[*m.OutlinedButton]()) }},
		{"icon", func() bool { return ts.Exists(gluttertest.ByType[*m.IconButton]()) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.exist() {
				t.Errorf("the page has no %s button", tt.name)
			}
		})
	}
}

// TestButtonsSegmentedAndLocation uses the segmented buttons of the page: the
// chooser moves the floating action button of the small scaffold.
func TestButtonsSegmentedAndLocation(t *testing.T) {
	ts := openButtonsPage(t)
	scroll := gluttertest.First(gluttertest.ByType[*m.ListView]())
	chooser := gluttertest.First(gluttertest.ByText("centerDocked"))
	for i := 0; i < 20 && (!ts.Exists(chooser) || ts.Center(chooser).Dy > 450); i++ {
		ts.Drag(scroll, m.Offset{Dx: 0, Dy: -200})
		ts.PumpAndSettle()
	}
	if !ts.Exists(gluttertest.ByText("Location: endFloat")) || ts.Center(chooser).Dy > 450 {
		t.Fatal("the floating action button demo cannot be scrolled into view")
	}
	tests := []struct {
		tap  string
		want string
	}{
		{"centerDocked", "Location: centerDocked"},
		{"startTop", "Location: startTop"},
		{"miniStartFloat", "Location: miniStartFloat"},
		{"endFloat", "Location: endFloat"},
	}
	for _, tt := range tests {
		t.Run(tt.tap, func(t *testing.T) {
			ts.Tap(gluttertest.First(gluttertest.ByText(tt.tap)))
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText(tt.want)) {
				t.Errorf("tapping %q did not give %q", tt.tap, tt.want)
			}
		})
	}
}
