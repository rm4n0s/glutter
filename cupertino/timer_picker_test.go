package cupertino_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoTimerPicker, ported from the group "Countdown timer picker" and the cases about the timer
// picker of cupertino/date_picker_test.dart (step 34.7). The goldens are in picker_golden_test.go.

// rowOffset is what moves a picker to its next item (_kRowOffset of the tests of Flutter).
var rowOffset = ui.Offset{Dy: -50}

func pickerApp(home widgets.Widget) *cupertino.CupertinoApp {
	return cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false), Home: home})
}

func square(dimension float64, child widgets.Widget) widgets.Widget {
	return widgets.NewSizedBox(child, widgets.SizedBoxOpts{Width: foundation.Some(dimension), Height: foundation.Some(dimension)})
}

func timerNoop(time.Duration) {}

func allPickers(ts *gluttertest.Tester) []*cupertino.CupertinoPicker {
	var out []*cupertino.CupertinoPicker
	for _, w := range ts.Widgets(gluttertest.ByType[*cupertino.CupertinoPicker]()) {
		out = append(out, w.(*cupertino.CupertinoPicker))
	}
	return out
}

func TestCupertinoTimerPickerArguments(t *testing.T) {
	skipUnlessDebug(t)
	for _, tt := range []struct {
		name string
		make func()
		tag  string
	}{
		{"a day", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{InitialTimerDuration: 24 * time.Hour})
		}, "CupertinoTimerPickerDurationOutOfRange"},
		{"a negative duration", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{InitialTimerDuration: -time.Second})
		}, "CupertinoTimerPickerDurationOutOfRange"},
		{"a negative minute interval", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{MinuteInterval: -1})
		}, "CupertinoTimerPickerMinuteIntervalInvalid"},
		{"a minute interval that is not a factor of 60", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{MinuteInterval: 7})
		}, "CupertinoTimerPickerMinuteIntervalInvalid"},
		{"a negative second interval", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{SecondInterval: -1})
		}, "CupertinoTimerPickerSecondIntervalInvalid"},
		{"a second interval that is not a factor of 60", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{SecondInterval: 7})
		}, "CupertinoTimerPickerSecondIntervalInvalid"},
		{"a duration off the minute interval", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{MinuteInterval: 10, InitialTimerDuration: 5 * time.Minute})
		}, "CupertinoTimerPickerDurationNotOnInterval"},
		{"a duration off the second interval", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{SecondInterval: 10, InitialTimerDuration: 5 * time.Second})
		}, "CupertinoTimerPickerDurationNotOnInterval"},
		{"an item extent of zero", func() {
			cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{ItemExtent: -1})
		}, "CupertinoTimerPickerItemExtentNotPositive"},
	} {
		t.Run(tt.name, func(t *testing.T) { checkErrorTag(t, recoverError(tt.make), tt.tag) })
	}
	t.Run("the mode cannot change", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop)))
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{Mode: cupertino.CupertinoTimerPickerModeHM})))
		checkErrorTag(t, firstError(t, ts), "CupertinoTimerPickerModeChanged")
	})
}

func TestCupertinoTimerPickerAppearance(t *testing.T) {
	t.Run("the background color is none by default", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop)))
		for _, p := range allPickers(ts) {
			if p.BackgroundColor() != nil {
				t.Errorf("a wheel has the background %v", p.BackgroundColor())
			}
		}
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("the given background color is applied", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{BackgroundColor: cupertino.CupertinoColors.Black})))
		for _, p := range allPickers(ts) {
			if p.BackgroundColor() != ui.ColorValue(cupertino.CupertinoColors.Black) {
				t.Errorf("a wheel has the background %v", p.BackgroundColor())
			}
		}
	})
	t.Run("the item extent is applied", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{ItemExtent: 42})))
		if len(allPickers(ts)) != 3 {
			t.Fatalf("%d wheels, want 3", len(allPickers(ts)))
		}
		for _, p := range allPickers(ts) {
			if p.ItemExtent() != 42 {
				t.Errorf("item extent %v", p.ItemExtent())
			}
		}
	})
}

func TestCupertinoTimerPickerColumnOrder(t *testing.T) {
	for _, tt := range []struct {
		name string
		rtl  bool
	}{{"ltr", false}, {"rtl", true}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			var home widgets.Widget = cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{InitialTimerDuration: 12*time.Hour + 30*time.Minute + 59*time.Second})
			if tt.rtl {
				home = widgets.NewDirectionality(ui.TextDirectionRTL, home)
			}
			ts.PumpWidget(pickerApp(home))
			last := ts.TopLeft(gluttertest.ByText("12")).Dx
			for _, s := range []string{"hours", "30", "min.", "59", "sec."} {
				x := ts.TopLeft(gluttertest.ByText(s)).Dx
				if (x > last) == tt.rtl {
					t.Errorf("%q at %v after %v: wrong order", s, x, last)
				}
				last = x
			}
		})
	}
}

func TestCupertinoTimerPickerWidth(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	initial := cupertino.CupertinoTimerPickerOpts{InitialTimerDuration: 12*time.Hour + 30*time.Minute + 59*time.Second}
	ts.PumpWidget(pickerApp(square(400, cupertino.NewCupertinoTimerPicker(timerNoop, initial))))
	distance := ts.Center(gluttertest.ByText("sec.")).Dx - ts.Center(gluttertest.ByText("12")).Dx
	ts.PumpWidget(pickerApp(widgets.NewSizedBox(cupertino.NewCupertinoTimerPicker(timerNoop, initial),
		widgets.SizedBoxOpts{Width: foundation.Some(800.0), Height: foundation.Some(400.0)})))
	if got := ts.Center(gluttertest.ByText("sec.")).Dx - ts.Center(gluttertest.ByText("12")).Dx; got != distance {
		t.Errorf("distance = %v, want %v", got, distance)
	}
}

func TestCupertinoTimerPickerReporting(t *testing.T) {
	t.Run("onScrollEnd reports only the last change", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var durations []time.Duration
		ts.PumpWidget(pickerApp(widgets.NewCenter(square(400, cupertino.NewCupertinoTimerPicker(func(d time.Duration) { durations = append(durations, d) },
			cupertino.CupertinoTimerPickerOpts{InitialTimerDuration: time.Hour + 30*time.Minute + 15*time.Second,
				ChangeReportingBehavior: widgets.ChangeReportingBehaviorOnScrollEnd})))))
		g := ts.StartGesture(ts.TopLeft(gluttertest.ByText("30")))
		g.MoveBy(ui.Offset{Dy: 32})
		if len(durations) != 0 {
			t.Fatalf("reported %v before the end", durations)
		}
		g.MoveBy(ui.Offset{Dy: 32})
		if len(durations) != 0 {
			t.Fatalf("reported %v before the end", durations)
		}
		g.Up()
		ts.PumpAndSettle()
		if len(durations) != 1 || durations[0] != time.Hour+28*time.Minute+15*time.Second {
			t.Errorf("durations = %v, want one of 1h28m15s", durations)
		}
	})
	t.Run("the intervals are honored", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var duration time.Duration
		ts.PumpWidget(pickerApp(square(400, cupertino.NewCupertinoTimerPicker(func(d time.Duration) { duration = d },
			cupertino.CupertinoTimerPickerOpts{MinuteInterval: 10, SecondInterval: 12, InitialTimerDuration: 10*time.Hour + 40*time.Minute + 48*time.Second}))))
		ts.Drag(gluttertest.ByText("40"), rowOffset)
		ts.Pump()
		ts.Drag(gluttertest.ByText("48"), ui.Offset{Dy: -rowOffset.Dy})
		ts.Pump()
		ts.Pump(500 * time.Millisecond)
		if want := 10*time.Hour + 50*time.Minute + 36*time.Second; duration != want {
			t.Errorf("duration = %v, want %v", duration, want)
		}
	})
}

func TestCupertinoTimerPickerLabels(t *testing.T) {
	t.Run("the hour label changes only after scrolling stops", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var duration foundation.Opt[time.Duration]
		ts.PumpWidget(pickerApp(widgets.NewCenter(widgets.NewSizedBox(cupertino.NewCupertinoTimerPicker(func(d time.Duration) { duration = foundation.Some(d) },
			cupertino.CupertinoTimerPickerOpts{Mode: cupertino.CupertinoTimerPickerModeHM, InitialTimerDuration: 2*time.Hour + 30*time.Minute}),
			widgets.SizedBoxOpts{Width: foundation.Some(320.0), Height: foundation.Some(216.0)}))))
		count := func(text string) int { return len(ts.Elements(gluttertest.ByText(text))) }
		if duration.IsSome() || count("hour") != 0 || count("hours") != 1 {
			t.Fatalf("at the start: duration %v, hour %d, hours %d", duration, count("hour"), count("hours"))
		}
		ts.Drag(gluttertest.ByText("2"), ui.Offset{Dy: -rowOffset.Dy})
		if int(duration.Value().Hours()) != 1 || count("hour") != 0 || count("hours") != 1 {
			t.Fatalf("after the drag: duration %v, hour %d, hours %d", duration, count("hour"), count("hours"))
		}
		ts.PumpAndSettle()
		if int(duration.Value().Hours()) != 1 || count("hours") != 0 || count("hour") != 1 {
			t.Errorf("after the scroll: duration %v, hour %d, hours %d", duration, count("hour"), count("hours"))
		}
	})
}

func TestCupertinoTimerPickerSize(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	key := foundation.NewValueKey("key")
	row := gluttertest.Descendant(gluttertest.ByKey(key), gluttertest.ByType[*widgets.Row]())
	for _, tt := range []struct {
		name string
		opts cupertino.CupertinoTimerPickerOpts
		want ui.Size
	}{
		{"hm", cupertino.CupertinoTimerPickerOpts{Key: key, Mode: cupertino.CupertinoTimerPickerModeHM, InitialTimerDuration: 2*time.Hour + 30*time.Minute}, ui.Size{Width: 320, Height: 216}},
		{"ms", cupertino.CupertinoTimerPickerOpts{Key: key, Mode: cupertino.CupertinoTimerPickerModeMS, InitialTimerDuration: 30*time.Minute + 3*time.Second}, ui.Size{Width: 320, Height: 216}},
		{"hms", cupertino.CupertinoTimerPickerOpts{Key: key, InitialTimerDuration: 5*time.Hour + 17*time.Minute + 19*time.Second}, ui.Size{Width: 342, Height: 216}},
	} {
		// Different modes shouldn't share state.
		ts.PumpWidget(widgets.NewPlaceholder(nil))
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, tt.opts)))
		if got := ts.Size(row); got != tt.want {
			t.Errorf("%s: size %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCupertinoTimerPickerSelectionOverlayBuilder(t *testing.T) {
	for _, tt := range []struct {
		mode    cupertino.CupertinoTimerPickerMode
		columns int
	}{{cupertino.CupertinoTimerPickerModeHMS, 3}, {cupertino.CupertinoTimerPickerModeMS, 2}, {cupertino.CupertinoTimerPickerModeHM, 2}} {
		t.Run(fmt.Sprint(tt.mode), func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			var calls [][2]int
			ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{Mode: tt.mode,
				SelectionOverlayBuilder: func(bc widgets.BuildContext, columnCount, selectedIndex int) widgets.Widget {
					calls = append(calls, [2]int{columnCount, selectedIndex})
					return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey(fmt.Sprintf("overlay%d", selectedIndex))})
				}})))
			for i := range tt.columns {
				if len(ts.Elements(gluttertest.ByKey(foundation.NewValueKey(fmt.Sprintf("overlay%d", i))))) != 1 {
					t.Errorf("no overlay of column %d", i)
				}
			}
			if len(ts.Elements(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]())) != 0 {
				t.Error("the default overlay is built next to the custom ones")
			}
			for _, c := range calls {
				if c[0] != tt.columns {
					t.Errorf("columnCount = %d, want %d", c[0], tt.columns)
				}
			}
		})
	}
	t.Run("a builder that returns nil leaves the wheels with no overlay", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{
			SelectionOverlayBuilder: func(widgets.BuildContext, int, int) widgets.Widget { return nil }})))
		if n := len(ts.Elements(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]())); n != 0 {
			t.Errorf("%d default overlays", n)
		}
	})
}
