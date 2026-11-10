package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The goldens of the pickers (step 34.7): the date picker in its four modes, light and dark, and the timer picker. The
// text is Go Mono, which the test engine has in place of San Francisco.

func pickerGolden(t *testing.T, name string, dark bool, size ui.Size, picker widgets.Widget, before func(ts *gluttertest.Tester)) {
	t.Helper()
	ts := dialogTester(t, size)
	theme := cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessLight})
	if dark {
		theme = cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: ui.BrightnessDark})
	}
	ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false), Theme: theme,
		Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(picker))}))
	ts.PumpAndSettle()
	if before != nil {
		before(ts)
	}
	ts.MatchesGolden(name)
}

func TestCupertinoDatePickerGoldens(t *testing.T) {
	initial := foundation.Some(dts(2019, 1, 1, 4, 12, 30))
	size := ui.Size{Width: 500, Height: 216}
	picker := func(mode cupertino.CupertinoDatePickerMode) widgets.Widget {
		return widgets.NewSizedBox(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: mode, InitialDateTime: initial}),
			widgets.SizedBoxOpts{Width: foundation.Some(500.0), Height: foundation.Some(216.0)})
	}
	for _, tt := range []struct {
		name string
		mode cupertino.CupertinoDatePickerMode
		dark bool
	}{
		{"cupertino.datepicker.time.light", cupertino.CupertinoDatePickerModeTime, false},
		{"cupertino.datepicker.date.light", cupertino.CupertinoDatePickerModeDate, false},
		{"cupertino.datepicker.monthyear.light", cupertino.CupertinoDatePickerModeMonthYear, false},
		{"cupertino.datepicker.datetime.light", cupertino.CupertinoDatePickerModeDateAndTime, false},
		{"cupertino.datepicker.datetime.dark", cupertino.CupertinoDatePickerModeDateAndTime, true},
	} {
		t.Run(tt.name, func(t *testing.T) { pickerGolden(t, tt.name, tt.dark, size, picker(tt.mode), nil) })
	}
	t.Run("cupertino.datepicker.datetime.drag", func(t *testing.T) {
		// Slightly drag the hour column to make the current hour off-center.
		pickerGolden(t, "cupertino.datepicker.datetime.drag", false, size, picker(cupertino.CupertinoDatePickerModeDateAndTime), func(ts *gluttertest.Tester) {
			ts.Drag(gluttertest.ByText("4"), ui.Offset{Dy: rowOffset.Dy / 2})
			ts.Pump()
		})
	})
}

func TestCupertinoTimerPickerGoldens(t *testing.T) {
	size := ui.Size{Width: 400, Height: 216}
	picker := func(mode cupertino.CupertinoTimerPickerMode, d time.Duration) widgets.Widget {
		return widgets.NewSizedBox(cupertino.NewCupertinoTimerPicker(timerNoop, cupertino.CupertinoTimerPickerOpts{Mode: mode, InitialTimerDuration: d}),
			widgets.SizedBoxOpts{Width: foundation.Some(320.0), Height: foundation.Some(216.0)})
	}
	t.Run("cupertino.timerpicker.hm.light", func(t *testing.T) {
		pickerGolden(t, "cupertino.timerpicker.hm.light", false, size, picker(cupertino.CupertinoTimerPickerModeHM, 23*time.Hour+59*time.Minute), nil)
	})
	t.Run("cupertino.timerpicker.hm.drag", func(t *testing.T) {
		// Slightly drag the minute column to make the current minute off-center.
		pickerGolden(t, "cupertino.timerpicker.hm.drag", false, size, picker(cupertino.CupertinoTimerPickerModeHM, 23*time.Hour+59*time.Minute), func(ts *gluttertest.Tester) {
			ts.Drag(gluttertest.ByText("59"), ui.Offset{Dy: rowOffset.Dy / 2})
			ts.Pump()
		})
	})
	t.Run("cupertino.timerpicker.hms.dark", func(t *testing.T) {
		pickerGolden(t, "cupertino.timerpicker.hms.dark", true, ui.Size{Width: 420, Height: 216}, picker(cupertino.CupertinoTimerPickerModeHMS, 5*time.Hour+17*time.Minute+19*time.Second), nil)
	})
}

func TestCupertinoWheelPickerGoldens(t *testing.T) {
	items := func() []widgets.Widget {
		var out []widgets.Widget
		for _, s := range []string{"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape"} {
			out = append(out, widgets.NewCenter(widgets.NewText(s)))
		}
		return out
	}
	size := ui.Size{Width: 240, Height: 216}
	picker := func() widgets.Widget {
		return widgets.NewSizedBox(cupertino.NewCupertinoPicker(32, pickerNoop, items(), cupertino.CupertinoPickerOpts{UseMagnifier: true,
			Magnification: foundation.Some(1.2), Squeeze: foundation.Some(1.25), ScrollController: widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: 2})}),
			widgets.SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(216.0)})
	}
	t.Run("cupertino.picker.wheel.light", func(t *testing.T) { pickerGolden(t, "cupertino.picker.wheel.light", false, size, picker(), nil) })
	t.Run("cupertino.picker.wheel.dark", func(t *testing.T) { pickerGolden(t, "cupertino.picker.wheel.dark", true, size, picker(), nil) })
}
