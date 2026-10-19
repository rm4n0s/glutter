package gluttertest

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestPumpWidgetShowsTheWidget(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newRow(newSwatch(ui.NewColor(0xFFFF0000), 40, 30)))
	if got := ts.Size(ByType[*swatch]()); got != (ui.Size{Width: 40, Height: 30}) {
		t.Errorf("size = %v", got)
	}
	if len(ts.Scenes()) == 0 {
		t.Errorf("no scene was composited")
	}
	if ts.View.PhysicalSize() != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("view size = %v", ts.View.PhysicalSize())
	}
}

func TestPumpAdvancesTheFakeClock(t *testing.T) {
	tests := []struct {
		name  string
		pumps []time.Duration
		want  time.Duration
	}{
		{"none", nil, 0},
		{"one", []time.Duration{time.Second}, time.Second},
		{"several", []time.Duration{time.Second, 500 * time.Millisecond, 0}, 1500 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			ts.PumpWidget(newRow())
			for _, d := range tt.pumps {
				ts.Pump(d)
			}
			if got := ts.Elapsed(); got != tt.want {
				t.Errorf("Elapsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPumpAndSettle(t *testing.T) {
	tests := []struct {
		name      string
		callbacks int
		duration  time.Duration
		wantPumps int
	}{
		{"nothing scheduled", 0, 0, 1},
		{"three animation frames", 3, 0, 3},
		{"custom interval", 2, 50 * time.Millisecond, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t)
			ts.PumpWidget(newRow())
			left := tt.callbacks
			var cb func(time.Duration)
			cb = func(time.Duration) {
				left--
				if left > 0 {
					ts.Binding.Scheduler().ScheduleFrameCallback(cb)
				}
			}
			if tt.callbacks > 0 {
				ts.Binding.Scheduler().ScheduleFrameCallback(cb)
			}
			if got := ts.PumpAndSettle(PumpAndSettleOpts{Duration: tt.duration}); got != tt.wantPumps {
				t.Errorf("PumpAndSettle() = %d pumps, want %d", got, tt.wantPumps)
			}
			if left > 0 {
				t.Errorf("%d callbacks did not run", left)
			}
		})
	}
}

func TestPumpAndSettleTimeout(t *testing.T) {
	appErr := failure(t, func(ts *Tester) {
		ts.PumpWidget(newRow())
		var cb func(time.Duration)
		cb = func(time.Duration) { ts.Binding.Scheduler().ScheduleFrameCallback(cb) }
		ts.Binding.Scheduler().ScheduleFrameCallback(cb)
		ts.PumpAndSettle(PumpAndSettleOpts{Duration: time.Second, Timeout: 5 * time.Second})
	})
	requireRoute(t, appErr, "PumpAndSettleTimeout")
}

func TestPumpFrames(t *testing.T) {
	ts := New(t)
	frames := 0
	var cb func(time.Duration)
	cb = func(time.Duration) { frames++; ts.Binding.Scheduler().ScheduleFrameCallback(cb) }
	ts.Binding.Scheduler().ScheduleFrameCallback(cb)
	ts.PumpFrames(newRow(), time.Second, PumpFramesOpts{Interval: 250 * time.Millisecond})
	if frames != 5 {
		t.Errorf("frames = %d, want 5", frames)
	}
}

func TestScreenshotWithoutFrame(t *testing.T) {
	appErr := failure(t, func(ts *Tester) { ts.Screenshot() })
	requireRoute(t, appErr, "ScreenshotNoFrame")
}

func TestUnreadErrorsFailTheTestAtCleanup(t *testing.T) {
	tb := &fakeTB{}
	ts := New(tb)
	ts.PumpWidget(newBuilder(func(widgets.BuildContext) widgets.Widget { panic("boom") }))
	if len(ts.Errors()) != 1 {
		t.Fatalf("errors = %v", ts.Errors())
	}
	tb.runCleanups()
	if len(tb.errorfs) != 1 || !strings.Contains(tb.errorfs[0], "BuildPanicked") {
		t.Errorf("cleanup did not report the error: %q", tb.errorfs)
	}
}

func TestTakeError(t *testing.T) {
	tb := &fakeTB{}
	ts := New(tb)
	if ts.TakeError() != nil {
		t.Errorf("TakeError() before any error is not nil")
	}
	ts.PumpWidget(newBuilder(func(widgets.BuildContext) widgets.Widget { panic("boom") }))
	err := ts.TakeError()
	if err == nil || !foundation.IsRecoveredPanic(err) {
		t.Errorf("TakeError() = %v, want the recovered panic", err)
	}
	if ts.TakeError() != nil {
		t.Errorf("the error was not forgotten")
	}
	tb.runCleanups()
	if len(tb.errorfs) != 0 {
		t.Errorf("a taken error failed the test: %q", tb.errorfs)
	}
}

func TestViewSettersReachTheFramework(t *testing.T) {
	ts := New(t)
	var seen ui.Size
	ts.PumpWidget(newBuilder(func(bc widgets.BuildContext) widgets.Widget {
		seen = widgets.MediaQuerySizeOf(bc)
		return newRow()
	}))
	if seen != (ui.Size{Width: 800, Height: 600}) {
		t.Fatalf("size = %v", seen)
	}
	ts.SetViewSize(ui.Size{Width: 400, Height: 300})
	ts.Pump()
	if seen != (ui.Size{Width: 400, Height: 300}) {
		t.Errorf("size after SetViewSize = %v", seen)
	}
	ts.SetDevicePixelRatio(2)
	ts.Pump()
	if seen != (ui.Size{Width: 400, Height: 300}) {
		t.Errorf("logical size changed with the pixel ratio: %v", seen)
	}
	if ts.View.DevicePixelRatio() != 2 {
		t.Errorf("pixel ratio = %v", ts.View.DevicePixelRatio())
	}
}

func TestPlatformConfigurationReachesTheFramework(t *testing.T) {
	ts := New(t)
	var scale float64
	ts.PumpWidget(newBuilder(func(bc widgets.BuildContext) widgets.Widget {
		scale = widgets.MediaQueryTextScalerOf(bc).Scale(10) / 10
		return newRow()
	}))
	ts.SetPlatformConfiguration(func(c *ui.PlatformConfiguration) { c.TextScaleFactor = 1.5 })
	ts.Pump()
	if scale != 1.5 {
		t.Errorf("text scale = %v, want 1.5", scale)
	}
}

func TestGolden(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newRow(
		newSwatch(ui.NewColor(0xFF6750A4), 100, 60),
		newSwatch(ui.NewColor(0xFFFFB74D), 60, 100),
	))
	ts.MatchesGolden("tester_row")
}

func TestSemanticsLabelFinders(t *testing.T) {
	skipUnlessDebug(t)
	ts := New(t)
	ts.EnsureSemantics()
	a := newSwatch(ui.NewColor(0xFF000000), 10, 10)
	a.label = "Increment"
	b := newSwatch(ui.NewColor(0xFF000000), 10, 10)
	b.label = "Decrement"
	ts.PumpWidget(newRow(a, b, newSwatch(ui.NewColor(0xFF000000), 10, 10)))
	tests := []struct {
		name string
		f    Finder
		want int
	}{
		{"exact", BySemanticsLabel("Increment"), 1},
		{"no such label", BySemanticsLabel("Nope"), 0},
		{"pattern", BySemanticsLabelMatching(regexp.MustCompile(`crement$`)), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ts.Count(tt.f); got != tt.want {
				t.Errorf("Count(%s) = %d, want %d", tt.f.Describe(), got, tt.want)
			}
		})
	}
}
