package cupertino_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The tests of CupertinoDatePicker, ported from the group "Date picker" and the cases about the date picker of
// cupertino/date_picker_test.dart (step 34.7). The goldens are in picker_golden_test.go.

// dt is DateTime(year, month, day, hour, minute) of Dart: a local time.
func dt(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}

func dateNoop(time.Time) {}

func centeredSquare(size float64, picker widgets.Widget) widgets.Widget {
	return widgets.NewCenter(square(size, picker))
}

func TestCupertinoDatePickerArguments(t *testing.T) {
	skipUnlessDebug(t)
	d := func(o cupertino.CupertinoDatePickerOpts) func() {
		return func() { cupertino.NewCupertinoDatePicker(dateNoop, o) }
	}
	for _, tt := range []struct {
		name string
		make func()
		tag  string
	}{
		{"an item extent of zero", d(cupertino.CupertinoDatePickerOpts{ItemExtent: -1}), "CupertinoDatePickerItemExtentNotPositive"},
		{"a minute interval of 7", d(cupertino.CupertinoDatePickerOpts{MinuteInterval: 7}), "CupertinoDatePickerMinuteIntervalInvalid"},
		{"an initial date before the minimum date", d(cupertino.CupertinoDatePickerOpts{
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0)), MinimumDate: foundation.Some(dt(2018, 10, 11, 0, 0))}), "CupertinoDatePickerInitialBeforeMinimum"},
		{"an initial date after the maximum date", d(cupertino.CupertinoDatePickerOpts{
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0)), MaximumDate: foundation.Some(dt(2018, 10, 9, 0, 0))}), "CupertinoDatePickerInitialAfterMaximum"},
		{"a date before the minimum date in date mode", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate,
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0)), MinimumDate: foundation.Some(dt(2018, 10, 11, 0, 0))}), "CupertinoDatePickerInitialBeforeMinimum"},
		{"an initial year before the minimum year", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate,
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0)), MinimumYear: 2019}), "CupertinoDatePickerMinimumYearInvalid"},
		{"an initial year after the maximum year", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeMonthYear,
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0)), MaximumYear: foundation.Some(2017)}), "CupertinoDatePickerInitialYearAfterMaximumYear"},
		{"the day of the week in time mode", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeTime, ShowDayOfWeek: true}), "CupertinoDatePickerDayOfWeekOutsideDateMode"},
		{"the day of the week in month and year mode", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeMonthYear, ShowDayOfWeek: true}), "CupertinoDatePickerDayOfWeekOutsideDateMode"},
		{"the day of the week in the default mode", d(cupertino.CupertinoDatePickerOpts{ShowDayOfWeek: true}), "CupertinoDatePickerDayOfWeekOutsideDateMode"},
		{"a minute that is not on the interval", d(cupertino.CupertinoDatePickerOpts{
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 10, 4)), MinuteInterval: 3}), "CupertinoDatePickerMinuteNotDivisible"},
		{"a time separator in date mode", d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate, ShowTimeSeparator: true}), "CupertinoDatePickerTimeSeparatorMode"},
		{"an initial date that cannot be selected", d(cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(dt(2018, 10, 10, 10, 0)),
			SelectableDayPredicate: func(day time.Time) bool { return day.Day() != 10 }}), "CupertinoDatePickerInitialNotSelectable"},
	} {
		t.Run(tt.name, func(t *testing.T) { requireErrorTag(t, recoverError(tt.make), tt.tag) })
	}
	t.Run("the day of the week is fine in date mode", func(t *testing.T) {
		if err := recoverError(d(cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate, ShowDayOfWeek: true})); err != nil {
			t.Error(err)
		}
	})
	t.Run("the initial date is now by default", func(t *testing.T) {
		if cupertino.NewCupertinoDatePicker(dateNoop).InitialDateTime().IsZero() {
			t.Error("no initial date")
		}
	})
	t.Run("the mode cannot change", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate})))
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate + 1})))
		requireErrorTag(t, firstError(t, ts), "CupertinoDatePickerModeChanged")
	})
}

func TestCupertinoDatePickerAppearance(t *testing.T) {
	t.Run("the background color is none by default", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoDatePicker(dateNoop)))
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
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{BackgroundColor: cupertino.CupertinoColors.Black})))
		for _, p := range allPickers(ts) {
			if p.BackgroundColor() != ui.ColorValue(cupertino.CupertinoColors.Black) {
				t.Errorf("a wheel has the background %v", p.BackgroundColor())
			}
		}
	})
	t.Run("the item extent is applied", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{ItemExtent: 55})))
		for _, p := range allPickers(ts) {
			if p.ItemExtent() != 55 {
				t.Errorf("item extent %v", p.ItemExtent())
			}
		}
	})
}

func TestCupertinoDatePickerStrings(t *testing.T) {
	for _, tt := range []struct {
		name  string
		opts  cupertino.CupertinoDatePickerOpts
		texts []string
	}{
		{"date", cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0))}, []string{"September", "15", "2018"}},
		{"date and time", cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(dt(2018, 9, 15, 3, 14))}, []string{"Sat Sep 15", "3", "14", "AM"}},
		{"month and year", cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeMonthYear, InitialDateTime: foundation.Some(dt(2018, 9, 1, 0, 0))}, []string{"September", "2018"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, tt.opts))))
			for _, s := range tt.texts {
				if n := len(ts.Elements(gluttertest.ByText(s))); n != 1 {
					t.Errorf("%q is shown %d times, want once", s, n)
				}
			}
		})
	}
	t.Run("the day of the week", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate,
			ShowDayOfWeek: true, InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0))}))))
		if len(ts.Elements(gluttertest.ByText(" Sat 15 "))) != 1 {
			t.Error("the day of the month has no weekday")
		}
	})
}

func TestCupertinoDatePickerReporting(t *testing.T) {
	t.Run("the initial minute honors the minute interval", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var got time.Time
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { got = d }, cupertino.CupertinoDatePickerOpts{
			InitialDateTime: foundation.Some(dt(2018, 10, 10, 10, 3)), MinuteInterval: 3}))))
		ts.Drag(gluttertest.ByText("03"), rowOffset)
		ts.Pump()
		if got.Minute() != 6 {
			t.Errorf("minute = %d, want 6", got.Minute())
		}
	})
	t.Run("a new initial date after the first build changes nothing", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var got time.Time
		build := func(initial time.Time) widgets.Widget {
			return pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { got = d },
				cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(initial)})))
		}
		slop := gluttertest.GestureOpts{Pointer: 1, TouchSlopY: foundation.Some(0.0)}
		ts.PumpWidget(build(dt(2018, 1, 1, 10, 30)))
		ts.Drag(gluttertest.ByText("10"), ui.Offset{Dy: 32}, slop)
		ts.Pump()
		ts.Pump(500 * time.Millisecond)
		if !got.Equal(dt(2018, 1, 1, 9, 30)) {
			t.Fatalf("got %v, want 2018-01-01 09:30", got)
		}
		ts.PumpWidget(build(dt(2016, 4, 5, 15, 0)))
		ts.Drag(gluttertest.ByText("9"), ui.Offset{Dy: 32}, slop)
		ts.Pump()
		ts.Pump(500 * time.Millisecond)
		if !got.Equal(dt(2018, 1, 1, 8, 30)) {
			t.Errorf("got %v, want 2018-01-01 08:30", got)
		}
	})
}

func TestCupertinoDatePickerWidth(t *testing.T) {
	for _, tt := range []struct {
		name         string
		mode         cupertino.CupertinoDatePickerMode
		first, last  string
		exactDiffers bool
	}{
		{"date and time", cupertino.CupertinoDatePickerModeDateAndTime, "Mon Jan 1 ", "AM", false},
		{"date", cupertino.CupertinoDatePickerModeDate, "January", "2018", false},
		{"time", cupertino.CupertinoDatePickerModeTime, "10", "AM", true},
		{"month and year", cupertino.CupertinoDatePickerModeMonthYear, "January", "2018", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			opts := cupertino.CupertinoDatePickerOpts{Mode: tt.mode, InitialDateTime: foundation.Some(dt(2018, 1, 1, 10, 30))}
			if tt.mode == cupertino.CupertinoDatePickerModeMonthYear {
				opts.InitialDateTime = foundation.Some(dt(2018, 1, 1, 0, 0))
			}
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, opts))))
			distance := ts.Center(gluttertest.ByText(tt.first)).Dx - ts.Center(gluttertest.ByText(tt.last)).Dx
			ts.PumpWidget(pickerApp(widgets.NewCenter(widgets.NewSizedBox(cupertino.NewCupertinoDatePicker(dateNoop, opts),
				widgets.SizedBoxOpts{Width: foundation.Some(800.0), Height: foundation.Some(400.0)}))))
			got := ts.Center(gluttertest.ByText(tt.first)).Dx - ts.Center(gluttertest.ByText(tt.last)).Dx
			if d := got - distance; d > 1e-9 || d < -1e-9 {
				t.Errorf("distance = %v, want %v", got, distance)
			}
		})
	}
}

// dts is dt with seconds.
func dts(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.Local)
}

// textColor is the color of the style of the text widget that shows the string.
func textColor(t *testing.T, ts *gluttertest.Tester, text string) ui.Color {
	t.Helper()
	w, ok := ts.Widget(gluttertest.First(gluttertest.ByText(text))).(*widgets.Text)
	if !ok {
		t.Fatalf("%q is not a Text", text)
	}
	return w.Style().Color.Value()
}

func isGrayedOut(t *testing.T, ts *gluttertest.Tester, text string) bool {
	t.Helper()
	gray := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.InactiveGray, ts.Element(gluttertest.ByType[*cupertino.CupertinoDatePicker]()))
	return textColor(t, ts, text) == gray
}

func TestCupertinoDatePickerGrayedOutDays(t *testing.T) {
	for _, tt := range []struct {
		name              string
		min, max          foundation.Opt[time.Time]
		initial           time.Time
		grayed, notGrayed []string
	}{
		{"an unconstrained minimum", foundation.None[time.Time](), foundation.Some(dt(2018, 6, 15, 0, 0)), dt(2018, 6, 15, 0, 0), []string{"16"}, []string{"14", "15"}},
		{"an unconstrained maximum", foundation.Some(dt(2018, 6, 15, 0, 0)), foundation.None[time.Time](), dt(2018, 6, 15, 0, 0), []string{"14"}, []string{"15", "16"}},
		{"the months", foundation.Some(dt(2018, 5, 15, 0, 0)), foundation.Some(dt(2018, 7, 15, 0, 0)), dt(2018, 6, 15, 0, 0), nil, []string{"14", "16"}},
		{"the years", foundation.Some(dt(2017, 6, 15, 0, 0)), foundation.Some(dt(2019, 6, 15, 0, 0)), dt(2018, 6, 15, 0, 0), nil, []string{"14", "16"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
				Mode: cupertino.CupertinoDatePickerModeDate, MinimumDate: tt.min, MaximumDate: tt.max, InitialDateTime: foundation.Some(tt.initial)}))))
			for _, s := range tt.grayed {
				if !isGrayedOut(t, ts, s) {
					t.Errorf("%s is not grayed out", s)
				}
			}
			for _, s := range tt.notGrayed {
				if isGrayedOut(t, ts, s) {
					t.Errorf("%s is grayed out", s)
				}
			}
		})
	}
}

func TestCupertinoDatePickerWheelDoesNotBendOutwards(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 3200, Height: 600})
	picker := cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate,
		InitialDateTime: foundation.Some(dt(2018, 1, 1, 10, 30))})
	for _, width := range []float64{300, 3000} {
		ts.PumpWidget(pickerApp(cupertino.NewCupertinoPageScaffold(widgets.NewCenter(widgets.NewSizedBox(picker,
			widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(200.0)})))))
		center := ts.TopLeft(gluttertest.ByText("January")).Dx
		for _, month := range []string{"September", "October", "November", "December", "February", "March", "April", "May"} {
			// The bottom left of the text is at the left of its box.
			if other := ts.TopLeft(gluttertest.ByText(month)).Dx; !(center < other) {
				t.Errorf("width %v: January at %v is not left of %s at %v", width, center, month, other)
			}
		}
	}
}

func TestCupertinoDatePickerScrollsAwayFromInvalidDates(t *testing.T) {
	slop := gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)}
	t.Run("on a month change", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 3, 30, 0, 0))}))))
		ts.Drag(gluttertest.ByText("March"), ui.Offset{Dy: 32}, slop)
		// Momentarily, the 2018 and the incorrect 30 of February are aligned.
		if a, b := ts.TopLeft(gluttertest.ByText("2018")).Dy, ts.TopLeft(gluttertest.ByText("30")).Dy; a != b {
			t.Errorf("2018 at %v, 30 at %v", a, b)
		}
		ts.Pump() // the post frame callback that starts the animation
		ts.Pump() // the start of the activity
		ts.Pump(500 * time.Millisecond)
		if !date.Equal(dt(2018, 2, 28, 0, 0)) {
			t.Errorf("date = %v, want 2018-02-28", date)
		}
		if a, b := ts.TopLeft(gluttertest.ByText("2018")).Dy, ts.TopLeft(gluttertest.ByText("28")).Dy; a != b {
			t.Errorf("2018 at %v, 28 at %v", a, b)
		}
	})
	t.Run("on a day change", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 2, 27, 0, 0))}))))
		pointer := gluttertest.GestureOpts{Pointer: 1, TouchSlopY: foundation.Some(0.0)}
		ts.Drag(gluttertest.ByText("27"), ui.Offset{Dy: -32}, pointer)
		ts.Pump()
		if !date.Equal(dt(2018, 2, 28, 0, 0)) {
			t.Fatalf("date = %v, want 2018-02-28", date)
		}
		ts.Drag(gluttertest.ByText("28"), ui.Offset{Dy: -32}, pointer)
		ts.Pump()
		if !date.Equal(dt(2018, 2, 28, 0, 0)) {
			t.Errorf("the callback went through an invalid date: %v", date)
		}
		if a, b := ts.TopLeft(gluttertest.ByText("2018")).Dy, ts.TopLeft(gluttertest.ByText("29")).Dy; a != b {
			t.Errorf("the invalid 29 is not in the middle: 2018 at %v, 29 at %v", a, b)
		}
		ts.Pump()
		ts.Pump(500 * time.Millisecond)
		if !date.Equal(dt(2018, 2, 28, 0, 0)) {
			t.Errorf("date = %v, want 2018-02-28", date)
		}
		if a, b := ts.TopLeft(gluttertest.ByText("2018")).Dy, ts.TopLeft(gluttertest.ByText("28")).Dy; a != b {
			t.Errorf("2018 at %v, 28 at %v", a, b)
		}
	})
	t.Run("in date mode, and the callback does not report invalid dates", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		min, max := dt(2016, 2, 29, 0, 0), dt(2018, 12, 31, 0, 0)
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) {
			date = d
			if d.Before(min) || d.After(max) {
				t.Errorf("the callback reported %v", d)
			}
		}, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate, MinimumDate: foundation.Some(min), MaximumDate: foundation.Some(max),
			InitialDateTime: foundation.Some(dt(2017, 2, 28, 0, 0))}))))
		if !isGrayedOut(t, ts, "29") {
			t.Error("29 of February 2017 is not grayed out")
		}
		ts.Drag(gluttertest.ByText("2017"), ui.Offset{Dy: 32}, slop)
		ts.Pump()
		ts.PumpAndSettle()
		if !date.Equal(min) {
			t.Fatalf("date = %v, want %v", date, min)
		}
		if isGrayedOut(t, ts, "29") {
			t.Error("29 of February 2016 is grayed out")
		}
		ts.Drag(gluttertest.ByText("2016"), ui.Offset{Dy: -32}, slop)
		ts.Pump()
		ts.PumpAndSettle()
		if !date.Equal(dt(2017, 2, 28, 0, 0)) {
			t.Errorf("date = %v, want 2017-02-28", date)
		}
		if !isGrayedOut(t, ts, "29") {
			t.Error("29 of February 2017 is not grayed out after the return")
		}
	})
	for _, mode := range []struct {
		name string
		mode cupertino.CupertinoDatePickerMode
	}{{"date and time", cupertino.CupertinoDatePickerModeDateAndTime}, {"time", cupertino.CupertinoDatePickerModeTime}} {
		t.Run("in "+mode.name+" mode", func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			var date time.Time
			min, max := dt(2019, 11, 11, 3, 30), dts(2019, 11, 11, 14, 59, 59)
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) {
				date = d
				if min.After(d) || max.Before(d) {
					t.Errorf("the callback reported %v", d)
				}
			}, cupertino.CupertinoDatePickerOpts{Mode: mode.mode, MinimumDate: foundation.Some(min), MaximumDate: foundation.Some(max),
				InitialDateTime: foundation.Some(dt(2019, 11, 11, 4, 0))}))))
			if isGrayedOut(t, ts, "3") || !isGrayedOut(t, ts, "2") || !isGrayedOut(t, ts, "PM") {
				t.Fatal("the hours before the minimum and the PM are not the only ones grayed out")
			}
			ts.Drag(gluttertest.ByText("AM"), ui.Offset{Dy: -32}, slop)
			ts.Pump()
			ts.PumpAndSettle()
			if !date.Equal(dt(2019, 11, 11, 14, 59)) {
				t.Fatalf("date = %v, want 14:59", date)
			}
			if !isGrayedOut(t, ts, "AM") || !isGrayedOut(t, ts, "3") {
				t.Error("AM and 3 are not grayed out at 14:59")
			}
			ts.Drag(gluttertest.ByText("PM"), ui.Offset{Dy: 32}, slop)
			ts.Pump()
			ts.PumpAndSettle()
			if !date.Equal(min) {
				t.Errorf("date = %v, want %v", date, min)
			}
		})
	}
	t.Run("in month and year mode", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		min, max := dt(2016, 2, 1, 0, 0), dt(2018, 12, 1, 0, 0)
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) {
			date = d
			if d.Before(min) || d.After(max) {
				t.Errorf("the callback reported %v", d)
			}
		}, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeMonthYear, MinimumDate: foundation.Some(min), MaximumDate: foundation.Some(max),
			InitialDateTime: foundation.Some(dt(2017, 2, 1, 0, 0))}))))
		for _, step := range []struct {
			text string
			dy   float64
			want time.Time
		}{{"2017", 100, min}, {"2016", -100, max}, {"2016", 32, dt(2017, 12, 1, 0, 0)}} {
			ts.Drag(gluttertest.ByText(step.text), ui.Offset{Dy: step.dy}, slop)
			ts.Pump()
			ts.PumpAndSettle()
			if !date.Equal(step.want) {
				t.Errorf("after dragging %s by %v: date = %v, want %v", step.text, step.dy, date, step.want)
			}
		}
	})
}

func TestCupertinoDatePickerMinimumDate(t *testing.T) {
	t.Run("only its date part counts in date mode", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		minDate := dt(2020, 1, 1, 12, 0)
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeDate, MinimumDate: foundation.Some(minDate), InitialDateTime: foundation.Some(dt(2020, 1, 12, 0, 0))}))))
		ts.Drag(gluttertest.ByText("2020"), ui.Offset{Dy: 32}, gluttertest.GestureOpts{TouchSlopY: foundation.Some(0.0)})
		ts.Pump()
		ts.PumpAndSettle()
		if date.Year() != minDate.Year() || date.Month() != minDate.Month() || date.Day() != minDate.Day() {
			t.Errorf("date = %v, want the day of %v", date, minDate)
		}
	})
	t.Run("the day before a minimum at midnight is not shown", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		minDate := dt(2019, 12, 31, 0, 0)
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			MinimumDate: foundation.Some(minDate), InitialDateTime: foundation.Some(minDate.AddDate(0, 0, 1))}))))
		if n := len(ts.Elements(gluttertest.ByText("Mon Dec 30"))); n != 0 {
			t.Errorf("Mon Dec 30 is shown %d times", n)
		}
	})
}

func TestCupertinoDatePickerTimes(t *testing.T) {
	drag := func(ts *gluttertest.Tester, text string, by ui.Offset) {
		ts.Drag(gluttertest.ByText(text), by)
		ts.Pump()
		ts.Pump(500 * time.Millisecond)
	}
	for _, tt := range []struct {
		name string
		opts cupertino.CupertinoDatePickerOpts
		text string
		want time.Time
	}{
		{"midnight", cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeTime, InitialDateTime: foundation.Some(dt(2019, 1, 1, 0, 15))}, "15", dt(2019, 1, 1, 0, 16)},
		{"noon", cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeTime, InitialDateTime: foundation.Some(dt(2019, 1, 1, 12, 15))}, "15", dt(2019, 1, 1, 12, 16)},
		{"noon in 24 hour time", cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeTime, Use24hFormat: true,
			InitialDateTime: foundation.Some(dt(2019, 1, 1, 12, 25))}, "25", dt(2019, 1, 1, 12, 26)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			var date time.Time
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, tt.opts))))
			drag(ts, tt.text, rowOffset)
			if !date.Equal(tt.want) {
				t.Errorf("date = %v, want %v", date, tt.want)
			}
		})
	}
	t.Run("the picker keeps the am/pm value when the hours scroll", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeTime, InitialDateTime: foundation.Some(dt(2019, 1, 1, 3, 0))}))))
		up := ui.Offset{Dy: -rowOffset.Dy}
		for _, step := range []struct {
			text string
			by   ui.Offset
			want time.Time
		}{{"AM", rowOffset, dt(2019, 1, 1, 15, 0)}, {"3", rowOffset, dt(2019, 1, 1, 16, 0)}, {"PM", up, dt(2019, 1, 1, 4, 0)}, {"4", up, dt(2019, 1, 1, 3, 0)}} {
			drag(ts, step.text, step.by)
			if !date.Equal(step.want) {
				t.Errorf("after dragging %s: date = %v, want %v", step.text, date, step.want)
			}
		}
	})
	t.Run("the am/pm column scrolls when the hour column changes enough", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		var date time.Time
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { date = d }, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeTime, InitialDateTime: foundation.Some(dt(2018, 1, 1, 11, 59))}))))
		delta := ui.Offset{Dy: -18}
		sub := func(a, b ui.Offset) ui.Offset { return ui.Offset{Dx: a.Dx - b.Dx, Dy: a.Dy - b.Dy} }
		scale := func(a ui.Offset, k float64) ui.Offset { return ui.Offset{Dx: a.Dx * k, Dy: a.Dy * k} }
		add := func(a, b ui.Offset) ui.Offset { return ui.Offset{Dx: a.Dx + b.Dx, Dy: a.Dy + b.Dy} }
		neg := func(a ui.Offset) ui.Offset { return scale(a, -1) }
		for _, step := range []struct {
			text string
			by   ui.Offset
			want time.Time
		}{
			{"11", rowOffset, dt(2018, 1, 1, 12, 59)},
			{"12", neg(rowOffset), dt(2018, 1, 1, 11, 59)},
			{"11", neg(add(scale(sub(rowOffset, delta), 2), delta)), dt(2018, 1, 1, 9, 59)},
			{"9", add(scale(sub(rowOffset, delta), 6), delta), dt(2018, 1, 1, 15, 59)},
		} {
			drag(ts, step.text, step.by)
			if !date.Equal(step.want) {
				t.Errorf("after dragging %s: date = %v, want %v", step.text, date, step.want)
			}
		}
	})
}

func TestCupertinoDatePickerOrder(t *testing.T) {
	t.Run("the date order of the date picker", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			DateOrder: cupertino.DatePickerDateOrderYDM, Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 1, 14, 10, 30))}))))
		x := func(s string) float64 { return ts.TopLeft(gluttertest.ByText(s)).Dx }
		if !(x("2018") < x("14") && x("14") < x("January")) {
			t.Errorf("2018 at %v, 14 at %v, January at %v", x("2018"), x("14"), x("January"))
		}
	})
	t.Run("the date order of the month and year picker", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		for _, tt := range []struct {
			order     cupertino.DatePickerDateOrder
			yearFirst bool
		}{{cupertino.DatePickerDateOrderDMY, false}, {cupertino.DatePickerDateOrderMDY, false}, {cupertino.DatePickerDateOrderYDM, true}, {cupertino.DatePickerDateOrderYMD, true}} {
			ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
				Key: foundation.NewValueKey(tt.order), DateOrder: tt.order, Mode: cupertino.CupertinoDatePickerModeMonthYear, InitialDateTime: foundation.Some(dt(2018, 1, 14, 10, 30))}))))
			month, year := ts.TopLeft(gluttertest.ByText("January")).Dx, ts.TopLeft(gluttertest.ByText("2018")).Dx
			if (year < month) != tt.yearFirst {
				t.Errorf("%v: January at %v, 2018 at %v", tt.order.Name(), month, year)
			}
		}
	})
	t.Run("the hours and minutes in RTL", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(widgets.NewDirectionality(ui.TextDirectionRTL, widgets.NewCenter(widgets.NewSizedBox(
			cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(dt(2019, 1, 1, 4, 0))}),
			widgets.SizedBoxOpts{Width: foundation.Some(500.0), Height: foundation.Some(400.0)})))))
		if hour, minute := ts.TopLeft(gluttertest.ByText("4")).Dx, ts.TopLeft(gluttertest.ByText("00")).Dx; !(hour < minute) {
			t.Errorf("hour at %v is not left of minute at %v", hour, minute)
		}
	})
	t.Run("the values share a baseline", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2025, 2, 14, 0, 0))}))))
		y := func(s string) float64 { return ts.TopLeft(gluttertest.ByText(s)).Dy }
		if y("November") != y("11") || y("11") != y("2022") {
			t.Errorf("November at %v, 11 at %v, 2022 at %v", y("November"), y("11"), y("2022"))
		}
	})
}

func TestCupertinoDatePickerScrollEnd(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	var got []time.Time
	ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(func(d time.Time) { got = append(got, d) }, cupertino.CupertinoDatePickerOpts{
		Mode: cupertino.CupertinoDatePickerModeDate, ChangeReportingBehavior: widgets.ChangeReportingBehaviorOnScrollEnd,
		InitialDateTime: foundation.Some(dt(2025, 1, 1, 0, 0))}))))
	g := ts.StartGesture(ts.TopLeft(gluttertest.ByText("2025")))
	g.MoveBy(ui.Offset{Dy: -32})
	g.MoveBy(ui.Offset{Dy: -32})
	if len(got) != 0 {
		t.Fatalf("reported %v before the end", got)
	}
	g.Up()
	ts.PumpAndSettle()
	if len(got) != 1 || !got[0].Equal(dt(2027, 1, 1, 0, 0)) {
		t.Errorf("reported %v, want 2027-01-01 once", got)
	}
}

func TestCupertinoDatePickerSelectionOverlayBuilder(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mode    cupertino.CupertinoDatePickerMode
		overlay int
	}{{"month and year", cupertino.CupertinoDatePickerModeMonthYear, 2}, {"date", cupertino.CupertinoDatePickerModeDate, 3},
		{"time", cupertino.CupertinoDatePickerModeTime, 3}, {"date and time", cupertino.CupertinoDatePickerModeDateAndTime, 4}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			overlay := widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(ui.NewColor(0x12345678))})
			var counts []int
			ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: tt.mode,
				InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0)),
				SelectionOverlayBuilder: func(bc widgets.BuildContext, columnCount, selectedIndex int) widgets.Widget {
					counts = append(counts, columnCount)
					return overlay
				}}))))
			if n := len(ts.Elements(gluttertest.ByWidget(overlay))); n != tt.overlay {
				t.Errorf("%d overlays, want %d", n, tt.overlay)
			}
			for _, c := range counts {
				if c != tt.overlay {
					t.Errorf("columnCount = %d, want %d", c, tt.overlay)
				}
			}
		})
	}
	t.Run("a builder that returns nil leaves no overlay", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			InitialDateTime:         foundation.Some(dt(2018, 9, 15, 0, 0)),
			SelectionOverlayBuilder: func(widgets.BuildContext, int, int) widgets.Widget { return nil }}))))
		if n := len(ts.Elements(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]())); n != 0 {
			t.Errorf("%d default overlays", n)
		}
	})
	t.Run("the default overlays cap the first and the last column", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0))}))))
		var caps [][2]bool
		for _, w := range ts.Widgets(gluttertest.ByType[*cupertino.CupertinoPickerDefaultSelectionOverlay]()) {
			o := w.(*cupertino.CupertinoPickerDefaultSelectionOverlay)
			caps = append(caps, [2]bool{o.CapStartEdge(), o.CapEndEdge()})
		}
		want := [][2]bool{{true, false}, {false, false}, {false, true}}
		if len(caps) != 3 || caps[0] != want[0] || caps[1] != want[1] || caps[2] != want[2] {
			t.Errorf("caps = %v, want %v", caps, want)
		}
	})
}

func TestCupertinoDatePickerTimeSeparator(t *testing.T) {
	for _, show := range []bool{true, false} {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			Mode: cupertino.CupertinoDatePickerModeTime, ShowTimeSeparator: show}))))
		if n := len(ts.Elements(gluttertest.ByText(":"))); (n == 1) != show || n > 1 {
			t.Errorf("showTimeSeparator %v: %d colons", show, n)
		}
	}
}

func TestCupertinoDatePickerSelectableDayPredicate(t *testing.T) {
	t.Run("argument validation", func(t *testing.T) {
		skipUnlessDebug(t)
		in2025 := func(d time.Time) bool { return d.Year() == 2025 }
		for _, o := range []cupertino.CupertinoDatePickerOpts{{},
			{InitialDateTime: foundation.Some(dt(2025, 1, 1, 0, 0)), SelectableDayPredicate: in2025}, {SelectableDayPredicate: in2025}} {
			if err := recoverError(func() { cupertino.NewCupertinoDatePicker(dateNoop, o) }); err != nil {
				t.Errorf("%v", err)
			}
		}
		requireErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{InitialDateTime: foundation.Some(dt(2025, 7, 4, 0, 0)),
				SelectableDayPredicate: func(d time.Time) bool { return d.Month() == time.June }})
		}), "CupertinoDatePickerInitialNotSelectable")
	})
	t.Run("workdays: a scroll to Saturday goes on to Monday", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		initial := dt(2025, 6, 13, 0, 0)
		selected := initial
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(func(d time.Time) { selected = d }, cupertino.CupertinoDatePickerOpts{
			InitialDateTime:        foundation.Some(initial),
			SelectableDayPredicate: func(d time.Time) bool { return d.Weekday() >= time.Monday && d.Weekday() <= time.Friday }}))))
		ts.Drag(gluttertest.ByText("Sat Jun 14"), ui.Offset{Dy: -100})
		if !selected.Equal(dt(2025, 6, 16, 0, 0)) {
			t.Errorf("selected = %v, want 2025-06-16", selected)
		}
	})
	t.Run("weekend: a press on Friday changes nothing", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		initial := dt(2025, 6, 14, 0, 0)
		selected := initial
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(func(d time.Time) { selected = d }, cupertino.CupertinoDatePickerOpts{
			InitialDateTime:        foundation.Some(initial),
			SelectableDayPredicate: func(d time.Time) bool { return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday }}))))
		g := ts.StartGesture(ts.Center(gluttertest.ByText("Fri Jun 13")))
		ts.Pump()
		g.Up()
		ts.Pump()
		if !selected.Equal(initial) {
			t.Errorf("selected = %v, want %v", selected, initial)
		}
	})
	t.Run("custom: a drag onto an unselectable day is not reported", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		initial := dt(2025, 6, 16, 0, 0)
		selected := initial
		ts.PumpWidget(pickerApp(widgets.NewCenter(cupertino.NewCupertinoDatePicker(func(d time.Time) { selected = d }, cupertino.CupertinoDatePickerOpts{
			InitialDateTime:        foundation.Some(initial),
			SelectableDayPredicate: func(d time.Time) bool { return d.Day() >= 16 }}))))
		ts.Drag(gluttertest.ByText("Sun Jun 15"), ui.Offset{Dy: 64})
		ts.Pump()
		if !selected.Equal(initial) {
			t.Errorf("selected = %v, want %v", selected, initial)
		}
	})
}

func TestCupertinoDatePickerSemantics(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	ts.EnsureSemantics()
	minimum, maximum := dt(2018, 6, 10, 0, 0), dt(2018, 6, 20, 0, 0)
	ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
		MinimumDate: foundation.Some(minimum), MaximumDate: foundation.Some(maximum), InitialDateTime: foundation.Some(minimum),
		Mode: cupertino.CupertinoDatePickerModeDate}))))
	ts.Pump(time.Millisecond)
	// The column of the days has the value 10, and cannot decrease because 9 is disabled and excluded.
	n := pickerSemanticsNode(ts, "10")
	if n == nil {
		t.Fatal("no node has the value 10")
	}
	d := n.GetSemanticsData()
	if d.HasAction(ui.SemanticsActionDecrease) {
		t.Error("the day column at the minimum date can decrease")
	}
	if !d.HasAction(ui.SemanticsActionIncrease) {
		t.Error("the day column at the minimum date cannot increase")
	}
}

func TestCupertinoDatePickerDarkMode(t *testing.T) {
	for _, tt := range []struct {
		name       string
		brightness ui.Brightness
	}{{"light", ui.BrightnessLight}, {"dark", ui.BrightnessDark}} {
		t.Run(tt.name, func(t *testing.T) {
			ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
			ts.PumpWidget(cupertino.NewCupertinoApp(cupertino.CupertinoAppOpts{DebugShowCheckedModeBanner: foundation.Some(false),
				Theme: cupertino.NewCupertinoThemeData(cupertino.CupertinoThemeDataOpts{Brightness: tt.brightness}),
				Home: cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate,
					InitialDateTime: foundation.Some(dt(2018, 10, 10, 0, 0))})}))
			want := cupertino.CupertinoDynamicColorResolve(cupertino.CupertinoColors.Label, ts.Element(gluttertest.ByType[*cupertino.CupertinoDatePicker]()))
			if got := textColor(t, ts, "October"); got != want {
				t.Errorf("color = %v, want %v", got, want)
			}
		})
	}
}

func TestCupertinoTimerPickerZeroArea(t *testing.T) {
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	ts.PumpWidget(pickerApp(widgets.NewCenter(widgets.NewSizedBoxShrink(cupertino.NewCupertinoTimerPicker(timerNoop)))))
	if got := ts.Size(gluttertest.ByType[*cupertino.CupertinoTimerPicker]()); got != (ui.Size{}) {
		t.Errorf("size = %v", got)
	}
}

func TestCupertinoDatePickerScrollControllersAreIndependentOfTheWidget(t *testing.T) {
	// The wheels of a picker keep their own controllers: a new picker widget with the same mode keeps the state.
	ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
	ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
		Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0))}))))
	ts.PumpWidget(pickerApp(centeredSquare(400, cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
		Mode: cupertino.CupertinoDatePickerModeDate, InitialDateTime: foundation.Some(dt(2018, 9, 15, 0, 0)), Use24hFormat: true}))))
	if n := len(ts.Elements(gluttertest.ByText("September"))); n != 1 {
		t.Errorf("%d Septembers", n)
	}
}

// emptyYear is localizations that have no text for the year.
type emptyYear struct {
	cupertino.DefaultCupertinoLocalizations
}

func (emptyYear) DatePickerYear(int) string { return "" }

func TestCupertinoDatePickerReportsItsErrors(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("a column with no text", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		delegate := &widgets.FuncLocalizationsDelegate[cupertino.CupertinoLocalizations]{
			LoadFunc: func(context.Context, ui.Locale) (cupertino.CupertinoLocalizations, error) { return emptyYear{}, nil }}
		ts.PumpWidget(pickerApp(widgets.NewLocalizations(ui.NewLocale("en", "US"),
			[]widgets.LocalizationsDelegate{cupertino.DefaultWidgetsLocalizationsDelegate, delegate},
			cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{Mode: cupertino.CupertinoDatePickerModeDate}))))
		ts.Pump()
		requireErrorTag(t, firstError(t, ts), "CupertinoDatePickerColumnType")
	})
	t.Run("too narrow a space", func(t *testing.T) {
		ts := dialogTester(t, ui.Size{Width: 800, Height: 600})
		ts.PumpWidget(pickerApp(widgets.NewCenter(widgets.NewSizedBox(cupertino.NewCupertinoDatePicker(dateNoop, cupertino.CupertinoDatePickerOpts{
			InitialDateTime: foundation.Some(dt(2019, 1, 1, 4, 0))}), widgets.SizedBoxOpts{Width: foundation.Some(100.0)}))))
		appErr := requireErrorTag(t, firstError(t, ts), "CupertinoDatePickerTooNarrow")
		if got := appErr.Message; !strings.Contains(got, "Insufficient horizontal space to render the CupertinoDatePicker") {
			t.Errorf("message = %q", got)
		}
	})
}
