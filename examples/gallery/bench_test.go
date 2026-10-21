package main

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
)

// The benchmarks of SPEC §15's gallery screen: the UI thread's time and allocations for
// a frame of the real gallery, at the size of a 1080p display with a pixel ratio of
// 1.25. Run them without the debug checks, which cost about ten times as much:
//
//	go test -tags glutter_release ./examples/gallery -run xxx -bench . -benchmem

func benchGallery(b *testing.B, page string, animate bool) *gluttertest.Tester {
	b.Helper()
	b.Setenv("GLUTTER_GALLERY_PAGE", page)
	if animate {
		b.Setenv("GLUTTER_GALLERY_ANIMATE", "1")
	}
	ts := gluttertest.New(b, gluttertest.Opts{Size: m.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true})
	ts.PumpWidget(NewGalleryApp())
	if animate {
		// An animation never settles: a second of frames lets the page build and the
		// first things that happen on it finish.
		for i := 0; i < 60; i++ {
			ts.PumpFrame(16 * time.Millisecond)
		}
	} else {
		ts.PumpAndSettle()
	}
	return ts
}

func countElements(ts *gluttertest.Tester) int {
	n := 0
	var visit m.ElementVisitor
	visit = func(e m.Element) {
		n++
		e.VisitChildren(visit)
	}
	ts.Binding.RootElement().VisitChildren(visit)
	return n
}

// BenchmarkGalleryAnimationFrame is a frame of the data page, where the progress
// indicators are indeterminate: only the animations advance, nothing builds.
func BenchmarkGalleryAnimationFrame(b *testing.B) {
	ts := benchGallery(b, "data", true)
	for i := 0; i < 30; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}

// BenchmarkGalleryFullRebuildFrame is a frame in which the whole app builds again, as
// when the theme changes.
func BenchmarkGalleryFullRebuildFrame(b *testing.B) {
	ts := benchGallery(b, "buttons", false)
	state := ts.Element(gluttertest.ByType[*GalleryApp]()).(*m.StatefulElement).State().(*galleryAppState)
	for i := 0; i < 5; i++ {
		state.SetState(func() {})
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetState(func() {})
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}

// TestRebuildEnvironmentKeepsFramesComing: without GLUTTER_GALLERY_REBUILD the gallery settles, and
// with it every frame asks for the next one (the whole app builds again in each), which is what the
// web measurements of SPEC §15 use for "a full rebuild of the gallery screen".
func TestRebuildEnvironmentKeepsFramesComing(t *testing.T) {
	for _, tt := range []struct {
		name    string
		rebuild string
		busy    bool
	}{
		{"off", "", false},
		{"on", "1", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GLUTTER_GALLERY_REBUILD", tt.rebuild)
			ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 1100, Height: 760}, DiscardScenes: true})
			ts.PumpWidget(NewGalleryApp())
			for range 20 {
				ts.PumpFrame(16 * time.Millisecond)
			}
			if got := ts.Binding.Scheduler().HasScheduledFrame(); got != tt.busy {
				t.Errorf("a frame is scheduled after 20 frames = %v, want %v", got, tt.busy)
			}
		})
	}
}

// BenchmarkGalleryPageSwitch is a click on an entry of the navigation rail, then the frame that builds the
// page: each operation goes to the next page of the gallery, so the whole screen is built new (every element,
// render object and state of the page is made, not updated). The click is the owner's measure of "the
// navigation feels slow" on the web (step 21.7); the pages are all visited once before the timing starts, so
// the caches of the text and the fonts are warm.
func BenchmarkGalleryPageSwitch(b *testing.B) {
	ts := gluttertest.New(b, gluttertest.Opts{Size: m.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true})
	ts.PumpWidget(NewGalleryApp())
	ts.PumpAndSettle()
	open := func(i int) {
		ts.Tap(gluttertest.First(gluttertest.ByText(galleryPages[i%len(galleryPages)].Name)))
		ts.PumpFrame(16 * time.Millisecond)
	}
	for i := range 2 * len(galleryPages) {
		open(i + 1)
	}
	ts.PumpAndSettle()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ts.PumpAndSettle()
		b.StartTimer()
		ts.Tap(gluttertest.First(gluttertest.ByText(galleryPages[(i+1)%len(galleryPages)].Name)))
		ts.PumpFrame(16 * time.Millisecond)
	}
}

// TestPerfWebClicksHitTheNavigationRail keeps the points that scripts/perf-web.sh clicks (a view of 1536x864 CSS
// pixels) on the entries of the navigation rail: each click opens the page whose entry is at that height (the
// entries are 44 px apart from y = 86), and together they visit every page.
func TestPerfWebClicksHitTheNavigationRail(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "perf-web.sh"))
	if err != nil {
		t.Fatal(err)
	}
	m1 := regexp.MustCompile(`GALLERY_CLICKS:-([0-9,;]+)\}`).FindSubmatch(script)
	if m1 == nil {
		t.Fatal("scripts/perf-web.sh has no default for GALLERY_CLICKS")
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: m.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true})
	ts.PumpWidget(NewGalleryApp())
	ts.PumpAndSettle()
	state := ts.Element(gluttertest.ByType[*GalleryApp]()).(*m.StatefulElement).State().(*galleryAppState)
	visited := map[int]bool{}
	for _, point := range strings.Split(string(m1[1]), ";") {
		xs, ys, _ := strings.Cut(point, ",")
		x, errX := strconv.ParseFloat(xs, 64)
		y, errY := strconv.ParseFloat(ys, 64)
		if errX != nil || errY != nil {
			t.Fatalf("bad point %q", point)
		}
		want := int(math.Round((y - 86) / 44))
		ts.TapAt(m.Offset{Dx: x, Dy: y})
		ts.PumpAndSettle()
		if state.page != want {
			t.Errorf("a click at %s opened page %d (%s), want %d (%s)", point, state.page, galleryPages[state.page].Slug, want, galleryPages[want].Slug)
		}
		visited[state.page] = true
	}
	if len(visited) != len(galleryPages) {
		t.Errorf("the clicks visit %d pages of %d", len(visited), len(galleryPages))
	}
}
