package widgets_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The benchmarks of the numbers in SPEC §15 that the framework alone decides: the UI
// thread's time and allocations for a frame, with no rendering (the renderers have
// their own benchmarks). Run them with
//
//	go test ./widgets -run xxx -bench Frame -benchmem
//
// and scripts/perf.sh collects them with the numbers of the examples.

// wavePainter paints a progress-indicator-like shape that follows the animation: a
// track and an arc, a few rounded rectangles. It is what an animation that repaints
// (a ripple, an indicator) does each frame.
type wavePainter struct {
	rendering.CustomPainterBase
	value animation.Animation[float64]
	track *ui.Paint
	bar   *ui.Paint
}

func newWavePainter(value animation.Animation[float64]) *wavePainter {
	track, bar := ui.NewPaint(), ui.NewPaint()
	track.SetColor(ui.NewColor(0xFFE8DEF8))
	bar.SetColor(ui.NewColor(0xFF6750A4))
	return &wavePainter{CustomPainterBase: rendering.NewCustomPainterBase(value), value: value, track: track, bar: bar}
}

func (p *wavePainter) Paint(c *ui.Canvas, size ui.Size) {
	c.DrawRRect(ui.NewRRectFromRectAndRadius(ui.NewRectFromLTWH(0, 0, size.Width, size.Height), ui.Radius{X: 8, Y: 8}), p.track)
	v := p.value.Value()
	for i := 0; i < 8; i++ {
		x := size.Width * float64(i) / 8
		h := size.Height * (0.2 + 0.8*math.Abs(math.Sin(v*math.Pi+float64(i))))
		c.DrawRRect(ui.NewRRectFromRectAndRadius(ui.NewRectFromLTWH(x+2, size.Height-h, size.Width/8-4, h), ui.Radius{X: 3, Y: 3}), p.bar)
	}
}

func (p *wavePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	return old.(*wavePainter).value != p.value
}

// screen is a gallery-like screen of about 500 elements: two columns of 18 rows, each a
// card with a title, a subtitle, a colored dot and three chips, and a panel in the top
// right corner that has one animation in it. It is built twice for the full-rebuild
// benchmark, from the same description.
type screen struct {
	widgets.StatefulWidget
	// started is called with the controller of the animation, or nil.
	started func(*animation.AnimationController)
	// repaints makes the animation repaint a painter instead of changing the opacity
	// of a layer.
	repaints bool
}

func newScreen(started func(*animation.AnimationController)) *screen {
	return &screen{StatefulWidget: widgets.NewStatefulWidgetBase(), started: started}
}

func (w *screen) CreateState() widgets.State { return &screenState{} }

type screenState struct {
	widgets.StateOf[*screen]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
	opacity    animation.Animation[float64]
	generation int
}

func (s *screenState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(time.Second)})
	s.opacity = animation.NewTween(0.3, 1.0).Animate(s.controller)
	if s.Widget().started != nil {
		s.Widget().started(s.controller)
	}
}

func (s *screenState) Dispose() { s.controller.Dispose() }

var (
	benchText  = painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(14.0), Color: foundation.Some(ui.NewColor(0xFF1C1B1F))})
	benchSmall = painting.NewTextStyle(painting.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(12.0), Color: foundation.Some(ui.NewColor(0xFF49454F))})
)

func benchRow(i, generation int) widgets.Widget {
	chip := func(j int) widgets.Widget {
		return widgets.NewPadding(painting.NewEdgeInsetsAll(2), widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFE8DEF8)), BorderRadius: painting.NewBorderRadiusCircular(8)}),
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(2, 6),
				widgets.NewText("chip "+strconv.Itoa(j), widgets.TextOpts{Style: benchSmall}))))
	}
	return widgets.NewPadding(painting.NewEdgeInsetsSymmetric(2, 8),
		widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFF3EDF7)), BorderRadius: painting.NewBorderRadiusCircular(12)}),
			widgets.NewPadding(painting.NewEdgeInsetsAll(6), widgets.NewRow([]widgets.Widget{
				widgets.NewSizedBox(widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{
					Color: foundation.Some(ui.NewColor(0xFF6750A4)), Shape: painting.BoxShapeCircle}), nil),
					widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(8.0)}),
				widgets.NewExpanded(widgets.NewColumn([]widgets.Widget{
					widgets.NewText("Row "+strconv.Itoa(i)+" generation "+strconv.Itoa(generation), widgets.TextOpts{Style: benchText}),
					widgets.NewText("subtitle of the row", widgets.TextOpts{Style: benchSmall}),
				}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})),
				chip(1), chip(2), chip(3),
			}))))
}

func (s *screenState) Build(widgets.BuildContext) widgets.Widget {
	column := func(first int) widgets.Widget {
		rows := make([]widgets.Widget, 0, 18)
		for i := 0; i < 18; i++ {
			rows = append(rows, benchRow(first+i, s.generation))
		}
		return widgets.NewExpanded(widgets.NewColumn(rows))
	}
	// The animation: a panel whose opacity follows the controller, in a repaint boundary
	// of its own, so that a frame of the animation repaints only it.
	var panel widgets.Widget
	if s.Widget().repaints {
		panel = widgets.NewRepaintBoundary(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{
			Painter: newWavePainter(s.opacity), Size: ui.Size{Width: 120, Height: 60}}))
	} else {
		panel = widgets.NewRepaintBoundary(widgets.NewFadeTransition(s.opacity, widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFF6750A4)), BorderRadius: painting.NewBorderRadiusCircular(16)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(120.0), Height: foundation.Some(60.0)}))))
	}
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewStack([]widgets.Widget{
		widgets.NewRow([]widgets.Widget{column(0), column(18)}),
		widgets.NewPositioned(panel, widgets.PositionedOpts{Top: foundation.Some(8.0), Right: foundation.Some(8.0)}),
	}))
}

// benchTester pumps the screen to its first frame at the size of a 1080p display at
// a pixel ratio of 1.25 (1536 x 864 logical pixels).
func benchTester(b *testing.B, started func(*animation.AnimationController), repaints ...bool) *gluttertest.Tester {
	b.Helper()
	ts := gluttertest.New(b, gluttertest.Opts{Size: ui.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true})
	w := newScreen(started)
	w.repaints = len(repaints) > 0 && repaints[0]
	ts.PumpWidget(w)
	return ts
}

// countElements is the number of elements in the tree.
func countElements(ts *gluttertest.Tester) int {
	n := 0
	var visit widgets.ElementVisitor
	visit = func(e widgets.Element) {
		n++
		e.VisitChildren(visit)
	}
	ts.Binding.RootElement().VisitChildren(visit)
	return n
}

// BenchmarkAnimationFrame is a frame in which only an animation advances: no widget
// builds, the panel's opacity changes and the panel is painted again. SPEC §15: under
// 2 ms of the UI thread and 1 KB of allocations.
func BenchmarkAnimationFrame(b *testing.B) {
	var controller *animation.AnimationController
	ts := benchTester(b, func(c *animation.AnimationController) { controller = c })
	controller.Repeat(animation.RepeatOpts{Reverse: true})
	// Warm up the caches and the allocator.
	for i := 0; i < 20; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}

// BenchmarkAnimationFramePaint is an animation frame that repaints: a painter draws its
// shapes again into the picture of its repaint boundary.
func BenchmarkAnimationFramePaint(b *testing.B) {
	var controller *animation.AnimationController
	ts := benchTester(b, func(c *animation.AnimationController) { controller = c }, true)
	controller.Repeat(animation.RepeatOpts{Reverse: true})
	for i := 0; i < 20; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}

// BenchmarkFullRebuildFrame is a frame in which the whole screen builds again: every
// widget is new and every element updates. SPEC §15: under 8 ms of the UI thread.
func BenchmarkFullRebuildFrame(b *testing.B) {
	ts := benchTester(b, nil)
	state := ts.Element(gluttertest.ByType[*screen]()).(*widgets.StatefulElement).State().(*screenState)
	for i := 0; i < 5; i++ {
		state.SetState(func() { state.generation++ })
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetState(func() { state.generation++ })
		ts.PumpFrame(16 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}

// BenchmarkIdleFrame is what an idle app costs per Pump: no frame is scheduled, so
// nothing runs. SPEC §15: idle CPU is ~0.
func BenchmarkIdleFrame(b *testing.B) {
	ts := benchTester(b, nil)
	ts.Pump()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.PumpFrame(16 * time.Millisecond)
	}
}

// BenchmarkCaretFrame is a frame in which only the caret of a focused text field blinks, in a
// screen of about 500 elements (the screen of the other benchmarks, with an EditableText in its
// corner): nothing builds, the caret is painted again and the layers above it are made again
// (step 29.15). The allocations must not grow with the rest of the screen.
func BenchmarkCaretFrame(b *testing.B) {
	ts := gluttertest.New(b, gluttertest.Opts{Size: ui.Size{Width: 1536, Height: 864}, DevicePixelRatio: 1.25, DiscardScenes: true})
	controller := widgets.NewTextEditingController()
	focus := widgets.NewFocusNode()
	field := widgets.NewEditableText(controller, focus, benchText, ui.NewColor(0xFF6750A4), ui.NewColor(0xFFCAC4D0),
		widgets.EditableTextOpts{Autofocus: true})
	page := widgets.NewStack([]widgets.Widget{
		newScreen(nil),
		widgets.NewPositioned(widgets.NewSizedBox(field, widgets.SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(30.0)}),
			widgets.PositionedOpts{Top: foundation.Some(8.0), Left: foundation.Some(8.0)}),
	})
	ts.PumpWidget(widgets.NewWidgetsApp(ui.NewColor(0xFFFFFFFF), widgets.WidgetsAppOpts{
		DebugShowCheckedModeBanner: foundation.Some(false),
		OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
			return widgets.NewPageRouteBuilder(
				func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
					return page
				},
				widgets.PageRouteBuilderOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
		},
	}))
	for i := 0; i < 6; i++ {
		ts.PumpFrame(500 * time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.PumpFrame(500 * time.Millisecond)
	}
	b.ReportMetric(float64(countElements(ts)), "elements")
}
