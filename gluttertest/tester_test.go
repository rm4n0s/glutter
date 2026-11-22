package gluttertest

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
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

// TestPumpRunsPostedWorkBetweenBeginAndDrawFrame checks that the work a transient
// callback posts (a microtask in Flutter) runs before the draw phase of the same
// frame, as flutter_test flushes microtasks between the two.
func TestPumpRunsPostedWorkBetweenBeginAndDrawFrame(t *testing.T) {
	ts := New(t)
	ts.PumpWidget(newRow(newSwatch(ui.NewColor(0xFFFF0000), 40, 30)))
	var order []string
	sched := ts.Binding.Scheduler()
	sched.ScheduleFrameCallback(func(time.Duration) {
		order = append(order, "begin")
		ts.Binding.Post(func() { order = append(order, "posted") })
	})
	sched.AddPostFrameCallback(func(time.Duration) { order = append(order, "post frame") })
	ts.Pump()
	want := []string{"begin", "posted", "post frame"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestClearLogsForgetsWhatTheFakePlatformKept(t *testing.T) {
	ts := New(t, Opts{})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewSizedBox(nil)))
	ts.host.Lock()
	ts.host.TextInputStates = append(ts.host.TextInputStates, ui.TextInputState{})
	ts.host.ActivatedCursors = append(ts.host.ActivatedCursors, renderhost.CursorActivation{Device: 1, Kind: "click"})
	ts.host.WindowActions = append(ts.host.WindowActions, ui.WindowAction{Kind: ui.WindowActionMinimize})
	ts.host.Windows = append(ts.host.Windows, &renderhost.FakeWindow{View: 7, Destroyed: true}, &renderhost.FakeWindow{View: 8})
	ts.host.Unlock()
	ts.ClearLogs()
	if n := len(ts.TextInputStates()); n != 0 {
		t.Errorf("%d text input states stay", n)
	}
	if n := len(ts.ActivatedCursors()); n != 0 {
		t.Errorf("%d cursor activations stay", n)
	}
	if n := len(ts.WindowActions()); n != 0 {
		t.Errorf("%d window actions stay", n)
	}
	// A window that is alive is not a log: the app can still reach it.
	if ws := ts.Windows(); len(ws) != 1 || ws[0].View != 8 {
		t.Errorf("windows = %+v, want only the one of view 8", ws)
	}
}

// TestWindowActionsKeepTheActivationToken shows a test of an app what token the app activated its window
// with (step 54.3: a single-instance app hands the token of its second instance to the running one).
func TestWindowActionsKeepTheActivationToken(t *testing.T) {
	ts := New(t, Opts{})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
	view := widgets.ViewOf(ts.Element(ByText("main")))
	view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate, Token: "token-of-the-second-instance"})
	view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionActivate})
	got := ts.WindowActions()
	want := []ui.WindowAction{
		{Kind: ui.WindowActionActivate, Token: "token-of-the-second-instance"},
		{Kind: ui.WindowActionActivate},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("window actions = %+v, want %+v", got, want)
	}
}

// openListener listens for open requests from its InitState, as an app does, and shows how many it got.
type openListener struct {
	widgets.StatefulWidget
	got *[]ui.OpenRequest
}

func newOpenListener(got *[]ui.OpenRequest) *openListener {
	return &openListener{widgets.NewStatefulWidgetBase(), got}
}

func (w *openListener) CreateState() widgets.State { return &openListenerState{} }

type openListenerState struct {
	widgets.StateOf[*openListener]
	remove func()
}

func (s *openListenerState) InitState() {
	d := widgets.WidgetsBindingOf(s.Context()).PlatformDispatcher()
	s.remove = d.AddOpenRequestListener(func(r ui.OpenRequest) {
		s.SetState(func() { *s.Widget().got = append(*s.Widget().got, r) })
	})
}

func (s *openListenerState) Dispose() {
	s.remove()
}

func (s *openListenerState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText(fmt.Sprintf("opened %d", len(*s.Widget().got))))
}

// TestOpenRequestReachesTheApp is Theatro's issue (step 56.1) in the test process of an app, with each target
// platform: a request sent before the app is pumped (the one that starts an app) waits for the listener of its
// first widget; one sent later reaches it and the app shows it; after the widget is gone nothing is called.
func TestOpenRequestReachesTheApp(t *testing.T) {
	started := ui.OpenRequest{Files: []string{"/films/A Film.torrent"}}
	clicked := ui.OpenRequest{URLs: []string{"magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a"}}
	for _, p := range everyTargetPlatform {
		t.Run(p.String(), func(t *testing.T) {
			ts := New(t)
			runAs(t, p)
			var got []ui.OpenRequest
			ts.SendOpenRequest(started)
			ts.PumpWidget(newOpenListener(&got))
			ts.Pump(0)
			if want := []ui.OpenRequest{started}; !reflect.DeepEqual(got, want) {
				t.Fatalf("after the first frame the app has %+v, want the request that started it: %+v", got, want)
			}
			if err := FindsOneWidget(ts, ByText("opened 1")); err != nil {
				t.Error(err)
			}

			ts.SendOpenRequest(clicked)
			ts.Pump(0)
			if want := []ui.OpenRequest{started, clicked}; !reflect.DeepEqual(got, want) {
				t.Errorf("the app has %+v, want %+v", got, want)
			}
			if err := FindsOneWidget(ts, ByText("opened 2")); err != nil {
				t.Error(err)
			}

			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("gone")))
			ts.SendOpenRequest(clicked)
			ts.Pump(0)
			if len(got) != 2 {
				t.Errorf("a listener that was removed got a request: %+v", got)
			}
		})
	}
}
