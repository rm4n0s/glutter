package main

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/svg"
)

// useLinux makes the framework behave as on the Linux desktop, which the example is for: the
// tests default to Android.
func useLinux(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

// titles are the labels that the pictures get from the title of their files: they are in the
// semantics tree once a picture has loaded.
var titles = []string{"The Glutter logo", "A card with SVG text and a drop shadow"}

// newTester shows the page and waits until every picture has loaded.
func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 760, Height: 560}, Assets: Assets()})
	ts.EnsureSemantics()
	ts.PumpWidget(NewSvgApp())
	deadline := time.Now().Add(10 * time.Second)
	for !allLoaded(ts) {
		if time.Now().After(deadline) {
			t.Fatal("the pictures did not load")
		}
		ts.Pump(16 * time.Millisecond)
		time.Sleep(time.Millisecond)
	}
	ts.PumpAndSettle()
	return ts
}

func allLoaded(ts *gluttertest.Tester) bool {
	for _, name := range append(append([]string{}, iconNames...), titles...) {
		if !ts.Exists(gluttertest.BySemanticsLabel(name)) {
			return false
		}
	}
	return true
}

func TestEveryPictureLoads(t *testing.T) {
	ts := newTester(t)
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("errors reported: %v", errs)
	}
	// 8 plain icons, 8 tinted, 3 sizes, the logo and the card.
	if n := ts.Count(gluttertest.ByType[*svg.SvgPicture]()); n != 21 {
		t.Errorf("%d pictures, want 21", n)
	}
	// Each icon is in the page twice (plain and tinted); its title is its label.
	for _, name := range iconNames {
		if n := ts.Count(gluttertest.BySemanticsLabel(name)); n < 2 {
			t.Errorf("the label %q is in the page %d times, want at least 2", name, n)
		}
	}
}

func TestOneIconAtThreeSizes(t *testing.T) {
	ts := newTester(t)
	var widths []float64
	for _, w := range ts.Widgets(gluttertest.ByType[*svg.SvgPicture]()) {
		pic := w.(*svg.SvgPicture)
		if pic.Width() == foundation.Some(24.0) || pic.Width() == foundation.Some(48.0) || pic.Width() == foundation.Some(96.0) {
			size := ts.Size(gluttertest.ByWidget(w))
			if size.Width != size.Height {
				t.Errorf("the star %v is not square: %v", pic.Width(), size)
			}
			widths = append(widths, size.Width)
		}
	}
	if len(widths) != 3 || widths[0] != 24 || widths[1] != 48 || widths[2] != 96 {
		t.Errorf("widths = %v, want 24, 48 and 96", widths)
	}
}

func TestGolden(t *testing.T) {
	ts := newTester(t)
	ts.MatchesGolden("svg")
}
