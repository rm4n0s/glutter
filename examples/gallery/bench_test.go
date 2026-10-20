package main

import (
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
