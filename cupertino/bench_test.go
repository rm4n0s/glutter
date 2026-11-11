package cupertino_test

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The benchmarks of SPEC §31.5: the UI thread's time and allocations for a frame of the Cupertino page transition, of the
// large title collapsing while a list scrolls, and of the wheel of a picker while it flings, at the size of a 1080p
// display with a pixel ratio of 1.25 (the size of the gallery's benchmarks). Run them without the debug checks:
//
//	go test ./cupertino -run xxx -bench . -benchmem
//
// Each reports the time of one frame; an iteration of the page transition is the transition's frames.

const benchFrame = 16 * time.Millisecond

func benchTester(b *testing.B, home widgets.Widget) *gluttertest.Tester {
	b.Helper()
	ts := gluttertest.New(b, gluttertest.Opts{Size: ui.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true, Fonts: []gluttertest.Font{
		{Family: cupertino.CupertinoSystemTextFamily, Data: gomono.TTF}, {Family: cupertino.CupertinoSystemDisplayFamily, Data: gomono.TTF}}})
	ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: home}))
	ts.PumpAndSettle()
	return ts
}

// benchLongPage is a page with a large title over a list of rows, as the settings of an iOS app.
func benchLongPage(title string) widgets.Widget {
	rows := make([]widgets.Widget, 0, 100)
	for i := 0; i < 100; i++ {
		rows = append(rows, cupertino.NewCupertinoListTileNotched(widgets.NewText(fmt.Sprintf("Row %d", i)), cupertino.CupertinoListTileOpts{
			Trailing: cupertino.NewCupertinoListTileChevron(), AdditionalInfo: widgets.NewText("On")}))
	}
	return cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
		cupertino.NewCupertinoSliverNavigationBar(cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText(title)}),
		widgets.NewSliverList(widgets.NewSliverChildListDelegate(rows)),
	}))
}

// BenchmarkCupertinoPageTransition is a page pushed over a page of the settings, with the parallax slide, the shadow and the
// flight of the bar, and popped again. The metrics are the mean time of a frame while a transition runs ("ns/frame"; the
// first frame of the push, which builds the page, is in it) and the worst one ("worst-ns").
func BenchmarkCupertinoPageTransition(b *testing.B) {
	var bc widgets.BuildContext
	ts := benchTester(b, widgets.NewBuilder(func(c widgets.BuildContext) widgets.Widget {
		bc = c
		return benchLongPage("Settings")
	}))
	var frames int
	var total, worst time.Duration
	run := func() {
		for ts.Binding.Scheduler().HasScheduledFrame() {
			start := time.Now()
			ts.PumpFrame(benchFrame)
			d := time.Since(start)
			frames++
			total += d
			worst = max(worst, d)
		}
	}
	step := func() {
		widgets.NavigatorPush(bc, cupertino.NewCupertinoPageRoute(func(widgets.BuildContext) widgets.Widget { return benchLongPage("Next") }))
		run()
		widgets.NavigatorPop(bc)
		run()
	}
	step()
	frames, total, worst = 0, 0, 0
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step()
	}
	b.StopTimer()
	b.ReportMetric(float64(total.Nanoseconds())/float64(frames), "ns/frame")
	b.ReportMetric(float64(worst.Nanoseconds()), "worst-ns")
	b.ReportMetric(float64(frames)/float64(b.N), "frames/op")
}

// BenchmarkCupertinoLargeTitleCollapse is a frame of the list scrolling by a drag, with the large title collapsing into the
// middle of the bar and the bar's border fading in, over the rows of the settings.
func BenchmarkCupertinoLargeTitleCollapse(b *testing.B) {
	controller := widgets.NewScrollController()
	b.Cleanup(controller.Dispose)
	rows := make([]widgets.Widget, 0, 100)
	for i := 0; i < 100; i++ {
		rows = append(rows, cupertino.NewCupertinoListTileNotched(widgets.NewText(fmt.Sprintf("Row %d", i)), cupertino.CupertinoListTileOpts{Trailing: cupertino.NewCupertinoListTileChevron()}))
	}
	ts := benchTester(b, cupertino.NewCupertinoPageScaffold(widgets.NewCustomScrollView([]widgets.Widget{
		cupertino.NewCupertinoSliverNavigationBar(cupertino.CupertinoSliverNavigationBarOpts{LargeTitle: widgets.NewText("Settings")}),
		widgets.NewSliverList(widgets.NewSliverChildListDelegate(rows)),
	}, widgets.CustomScrollViewOpts{Controller: controller})))
	offset := 0.0
	step := 3.0
	frame := func() {
		offset += step
		if offset > 120 || offset < 0 {
			step = -step
			offset += 2 * step
		}
		controller.JumpTo(offset)
		ts.PumpFrame(benchFrame)
	}
	for i := 0; i < 60; i++ {
		frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame()
	}
}

// BenchmarkCupertinoPickerFling is a frame of a wheel that flings: the magnifier, the squeeze of the rows and the
// selection overlay over 1,000 rows, in the size of a date picker.
func BenchmarkCupertinoPickerFling(b *testing.B) {
	controller := cupertino.NewFixedExtentScrollController()
	b.Cleanup(controller.Dispose)
	children := make([]widgets.Widget, 0, 1000)
	for i := 0; i < 1000; i++ {
		children = append(children, widgets.NewText(fmt.Sprintf("Item %d", i)))
	}
	ts := benchTester(b, cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewSizedBox(
		cupertino.NewCupertinoPicker(32, func(int) {}, children, cupertino.CupertinoPickerOpts{ScrollController: controller}),
		widgets.SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(216.0)}))))
	row := 0
	frame := func() {
		if !ts.Binding.Scheduler().HasScheduledFrame() {
			// The fling ended: start the next one, in the other direction.
			row = (row + 400) % 1000
			controller.AnimateToItem(row, 3*time.Second, nil)
		}
		ts.PumpFrame(benchFrame)
	}
	for i := 0; i < 30; i++ {
		frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame()
	}
}
