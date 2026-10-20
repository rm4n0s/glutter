package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
)

// openMenusPage opens the menus page through the navigation rail.
func openMenusPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.First(gluttertest.ByText("Menus")))
	ts.PumpAndSettle()
	return ts
}

// TestMenusGolden draws the top of the menus page in the light theme.
func TestMenusGolden(t *testing.T) {
	ts := openMenusPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-menus")
}

// TestMenusOpenGolden draws the page with the File menu and its submenu open.
func TestMenusOpenGolden(t *testing.T) {
	ts := openMenusPage(t)
	ts.Tap(gluttertest.ByText("File"))
	ts.PumpAndSettle()
	ts.Hover(ts.Center(gluttertest.ByText("Open recent")))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("notes.txt")) {
		t.Fatal("the submenu is not open")
	}
	ts.MatchesGolden("gallery-menus-open")
}

// TestMenusDevOpenMenu checks that GLUTTER_DEV_OPEN_MENU opens the menus when the
// page is shown, in a debug build.
func TestMenusDevOpenMenu(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantNew bool
		wantSub bool
	}{
		{"none", "", false, false},
		{"file", "file", true, false},
		{"file and recent", "file/recent", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !foundation.DebugMode {
				t.Skip("the dev variables work only in debug builds")
			}
			t.Setenv("GLUTTER_DEV_OPEN_MENU", tc.value)
			ts := openMenusPage(t)
			ts.Pump(time.Second)
			ts.PumpAndSettle()
			if got := ts.Exists(gluttertest.ByText("Open recent")); got != tc.wantNew {
				t.Errorf("File menu open = %v, want %v", got, tc.wantNew)
			}
			if got := ts.Exists(gluttertest.ByText("notes.txt")); got != tc.wantSub {
				t.Errorf("submenu open = %v, want %v", got, tc.wantSub)
			}
		})
	}
}
