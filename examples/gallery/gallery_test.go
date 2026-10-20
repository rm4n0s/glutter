package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// useLinux makes the framework behave as on the Linux desktop, which the example
// is for: the tests default to Android.
func useLinux(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 1100, Height: 760}})
	ts.PumpWidget(NewGalleryApp())
	return ts
}

func TestShell(t *testing.T) {
	ts := newTester(t)
	for _, text := range []string{"Glutter gallery", "AppBar", "SliverAppBar: medium and large", "SnackBar and MaterialBanner", "BottomAppBar", "Show SnackBar"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("the text %q is not shown", text)
		}
	}
	// The page list has the name of the first page, and it is the one that is open.
	if n := ts.Count(gluttertest.ByText("App bars & snack bars")); n != 1 {
		t.Errorf("the page name is shown %d times, want 1", n)
	}
}

// openNavigationPage opens the second page through the navigation rail.
func openNavigationPage(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Navigation & tabs"))
	ts.PumpAndSettle()
	return ts
}

func TestRailSwitchesPages(t *testing.T) {
	ts := openNavigationPage(t)
	for _, text := range []string{"NavigationBar", "TabBar and TabBarView", "NavigationRail", "NavigationDrawer", "Explore page"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("the text %q is not shown", text)
		}
	}
	if ts.Exists(gluttertest.ByText("SliverAppBar: medium and large")) {
		t.Error("the first page is still shown")
	}
	ts.Tap(gluttertest.ByText("App bars & snack bars"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("SliverAppBar: medium and large")) {
		t.Error("the first page is not back")
	}
}

func TestNavigationDemos(t *testing.T) {
	ts := openNavigationPage(t)
	// The navigation bar and the rail of the demos select their own pages.
	ts.Tap(gluttertest.Descendant(gluttertest.ByType[*m.NavigationBar](), gluttertest.ByText("Inbox")))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Inbox page")) {
		t.Error("the navigation bar did not select the second page")
	}
	ts.Tap(gluttertest.Descendant(gluttertest.ByType[*m.NavigationRail](), gluttertest.ByText("Settings")))
	ts.PumpAndSettle()
	if n := ts.Count(gluttertest.ByText("Settings page")); n != 1 {
		t.Errorf("the rail of the demo shows %d settings pages, want 1", n)
	}
	// The tabs of the demo select their views.
	ts.Tap(gluttertest.ByText("Trips"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.Descendant(gluttertest.ByType[*m.TabBarView](), gluttertest.ByText("Trips"))) {
		t.Error("the tab bar did not select the second view")
	}
}

func TestSheetDemos(t *testing.T) {
	ts := openNavigationPage(t)
	scroll := gluttertest.ByType[*m.ListView]()
	ts.Drag(gluttertest.First(scroll), m.Offset{Dx: 0, Dy: -300})
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("Modal bottom sheet"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("A modal bottom sheet")) {
		t.Fatal("the modal sheet is not shown")
	}
	ts.Tap(gluttertest.ByText("Close"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("A modal bottom sheet")) {
		t.Error("the modal sheet is still shown")
	}
	ts.Tap(gluttertest.ByText("Persistent bottom sheet"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("A persistent bottom sheet")) {
		t.Fatal("the persistent sheet is not shown")
	}
	ts.Tap(gluttertest.ByText("Close"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("A persistent bottom sheet")) {
		t.Error("the persistent sheet is still shown")
	}
}

func TestThemeToggle(t *testing.T) {
	ts := newTester(t)
	brightness := func() m.Brightness {
		return m.ThemeBrightnessOf(ts.Element(gluttertest.First(gluttertest.ByType[*m.AppBar]())))
	}
	if got := brightness(); got != m.BrightnessLight {
		t.Fatalf("brightness at the start = %v, want light", got)
	}
	toggle := func(tooltip string) gluttertest.Finder {
		return gluttertest.ByPredicate("the theme toggle", func(w m.Widget) bool {
			b, ok := w.(*m.IconButton)
			return ok && b.Tooltip() == tooltip
		})
	}
	ts.Tap(toggle("Switch to the dark theme"))
	ts.PumpAndSettle()
	if got := brightness(); got != m.BrightnessDark {
		t.Errorf("brightness after the toggle = %v, want dark", got)
	}
	ts.Tap(toggle("Switch to the light theme"))
	ts.PumpAndSettle()
	if got := brightness(); got != m.BrightnessLight {
		t.Errorf("brightness after the second toggle = %v, want light", got)
	}
}

func TestSnackBarAndBannerButtons(t *testing.T) {
	tests := []struct {
		name   string
		button string
		want   string
	}{
		{"snack bar", "Show SnackBar", "A fixed snack bar"},
		{"action", "With an action", "Item deleted"},
		{"floating", "Floating", "A floating snack bar"},
		{"close icon", "With a close icon", "It can be closed"},
		{"banner", "Show MaterialBanner", "A banner stays until it is dismissed."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTester(t)
			button := gluttertest.ByText(tc.button)
			ts.Tap(button)
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Fatalf("%q is not shown", tc.want)
			}
			ts.Tap(gluttertest.ByText("Clear all"))
			ts.PumpAndSettle()
			if ts.Exists(gluttertest.ByText(tc.want)) {
				t.Errorf("%q is still shown after Clear all", tc.want)
			}
		})
	}
}

func TestBannerDismiss(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("Show MaterialBanner"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("DISMISS"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("A banner stays until it is dismissed.")) {
		t.Error("the banner is still shown")
	}
}

func TestDevPage(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_PAGE is read in debug builds only")
	}
	pages := []Page{{Slug: "a"}, {Slug: "b"}}
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"unset", "", 0},
		{"second", "b", 1},
		{"unknown", "x", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := devPage(func(k string) string {
				if k != "GLUTTER_DEV_PAGE" {
					t.Errorf("read %q", k)
				}
				return tc.value
			}, pages)
			if got != tc.want {
				t.Errorf("devPage(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestDevThemeMode(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_THEME is read in debug builds only")
	}
	tests := []struct {
		value string
		want  m.ThemeMode
	}{
		{"", m.ThemeModeLight},
		{"light", m.ThemeModeLight},
		{"dark", m.ThemeModeDark},
		{"DARK", m.ThemeModeLight},
	}
	for _, tc := range tests {
		if got := devThemeMode(func(string) string { return tc.value }); got != tc.want {
			t.Errorf("devThemeMode(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestGalleryGolden draws the first page in the light theme, after the snack bar
// of the first button is shown.
func TestGalleryGolden(t *testing.T) {
	ts := newTester(t)
	ts.Tap(gluttertest.ByText("With an action"))
	ts.PumpAndSettle()
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-app-bars")
}

// TestNavigationGolden draws the second page in the light theme.
func TestNavigationGolden(t *testing.T) {
	ts := openNavigationPage(t)
	ts.Pump(time.Second)
	ts.MatchesGolden("gallery-navigation")
}

func TestDevSnackBar(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("GLUTTER_DEV_SNACKBAR is read in debug builds only")
	}
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"1", true},
		{"0", false},
		{"yes", false},
	}
	for _, tc := range tests {
		if got := devSnackBar(func(string) string { return tc.value }); got != tc.want {
			t.Errorf("devSnackBar(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
