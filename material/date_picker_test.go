// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.Local) }

func TestDatePickerSmoke(t *testing.T) {
	var result <-chan time.Time
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: linuxTheme(), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			result = ShowDatePicker(bc, day(2020, 1, 1), day(2030, 12, 31), ShowDatePickerOpts{InitialDate: day(2025, 6, 15), CurrentDate: day(2025, 6, 10)})
		}}))
	})})}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	ts.Tap(gluttertest.ByText("20"))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	got := <-result
	if !got.Equal(day(2025, 6, 20)) {
		t.Errorf("picked %v", got)
	}
	_ = foundation.DebugMode
}

func someRange(a, b time.Time) foundation.Opt[DateTimeRange] {
	return foundation.Some(NewDateTimeRange(a, b))
}

// pumpApp pumps a Material app whose home is a button that runs show.
func pumpPickerApp(t *testing.T, size ui.Size, show func(bc widgets.BuildContext)) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: size})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: linuxTheme(), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() { show(bc) }}))
	})})}))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	return ts
}

func TestDateUtils(t *testing.T) {
	t.Run("DateOnly", func(t *testing.T) {
		got := DateUtilsDateOnly(time.Date(2025, 3, 4, 13, 14, 15, 0, time.Local))
		if !got.Equal(day(2025, 3, 4)) {
			t.Errorf("got %v", got)
		}
	})
	t.Run("IsSameDay", func(t *testing.T) {
		tests := []struct {
			name string
			a, b time.Time
			want bool
		}{
			{"same", day(2025, 3, 4), time.Date(2025, 3, 4, 23, 0, 0, 0, time.Local), true},
			{"next day", day(2025, 3, 4), day(2025, 3, 5), false},
			{"next year", day(2025, 3, 4), day(2026, 3, 4), false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := DateUtilsIsSameDay(tt.a, tt.b); got != tt.want {
					t.Errorf("got %v", got)
				}
			})
		}
	})
	t.Run("IsSameMonth", func(t *testing.T) {
		if !DateUtilsIsSameMonth(day(2025, 3, 1), day(2025, 3, 31)) || DateUtilsIsSameMonth(day(2025, 3, 1), day(2026, 3, 1)) {
			t.Error("wrong")
		}
	})
	t.Run("MonthDelta", func(t *testing.T) {
		tests := []struct {
			a, b time.Time
			want int
		}{
			{day(2025, 3, 1), day(2025, 3, 20), 0},
			{day(2025, 3, 1), day(2025, 5, 1), 2},
			{day(2025, 3, 1), day(2024, 12, 1), -3},
			{day(2020, 1, 1), day(2025, 1, 1), 60},
		}
		for _, tt := range tests {
			if got := DateUtilsMonthDelta(tt.a, tt.b); got != tt.want {
				t.Errorf("MonthDelta(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		}
	})
	t.Run("AddMonthsToMonthDate", func(t *testing.T) {
		tests := []struct {
			a    time.Time
			n    int
			want time.Time
		}{
			{day(2025, 11, 1), 3, day(2026, 2, 1)},
			{day(2025, 2, 1), -3, day(2024, 11, 1)},
			{day(2025, 2, 1), 0, day(2025, 2, 1)},
		}
		for _, tt := range tests {
			if got := DateUtilsAddMonthsToMonthDate(tt.a, tt.n); !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		}
	})
	t.Run("AddDaysToDate", func(t *testing.T) {
		tests := []struct {
			a    time.Time
			n    int
			want time.Time
		}{
			{day(2025, 2, 27), 3, day(2025, 3, 2)},
			{day(2024, 2, 27), 3, day(2024, 3, 1)},
			{day(2025, 1, 1), -1, day(2024, 12, 31)},
		}
		for _, tt := range tests {
			if got := DateUtilsAddDaysToDate(tt.a, tt.n); !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		}
	})
	t.Run("GetDaysInMonth", func(t *testing.T) {
		tests := []struct{ y, m, want int }{{2025, 1, 31}, {2025, 2, 28}, {2024, 2, 29}, {2100, 2, 28}, {2000, 2, 29}, {2025, 4, 30}, {2025, 12, 31}}
		for _, tt := range tests {
			if got := DateUtilsGetDaysInMonth(tt.y, tt.m); got != tt.want {
				t.Errorf("DaysInMonth(%d, %d) = %d, want %d", tt.y, tt.m, got, tt.want)
			}
		}
	})
	t.Run("FirstDayOffset", func(t *testing.T) {
		l := DefaultMaterialLocalizations{}
		// June 1, 2025 is a Sunday, and the first day of the week is Sunday.
		if got := DateUtilsFirstDayOffset(2025, 6, l); got != 0 {
			t.Errorf("June 2025 offset = %d", got)
		}
		// July 1, 2025 is a Tuesday.
		if got := DateUtilsFirstDayOffset(2025, 7, l); got != 2 {
			t.Errorf("July 2025 offset = %d", got)
		}
	})
}

func TestDateTimeRange(t *testing.T) {
	r := NewDateTimeRange(day(2025, 1, 1), day(2025, 1, 11))
	if r.Duration() != 10*24*time.Hour {
		t.Errorf("duration %v", r.Duration())
	}
	if r.IsZero() || !(DateTimeRange{}).IsZero() {
		t.Error("IsZero")
	}
	if !r.Equal(NewDateTimeRange(day(2025, 1, 1), day(2025, 1, 11))) || r.Equal(NewDateTimeRange(day(2025, 1, 1), day(2025, 1, 12))) {
		t.Error("Equal")
	}
	if got := DateUtilsDatesOnly(NewDateTimeRange(time.Date(2025, 1, 1, 5, 0, 0, 0, time.Local), time.Date(2025, 1, 2, 5, 0, 0, 0, time.Local))); !got.Start.Equal(day(2025, 1, 1)) || !got.End.Equal(day(2025, 1, 2)) {
		t.Errorf("DatesOnly = %v", got)
	}
}

func TestShowDatePickerResults(t *testing.T) {
	first, last := day(2020, 1, 1), day(2030, 12, 31)
	tests := []struct {
		name string
		opts ShowDatePickerOpts
		act  func(ts *gluttertest.Tester)
		want time.Time
	}{
		{name: "cancel", act: func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Cancel")) }},
		{name: "ok keeps the initial date", act: func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("OK")) }, want: day(2025, 6, 15)},
		{name: "pick a day", act: func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("3"))
			ts.Tap(gluttertest.ByText("OK"))
		}, want: day(2025, 6, 3)},
		{name: "next month", act: func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByIcon(Icons.ChevronRight()))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("9"))
			ts.Tap(gluttertest.ByText("OK"))
		}, want: day(2025, 7, 9)},
		{name: "predicate blocks days", opts: ShowDatePickerOpts{SelectableDayPredicate: func(d time.Time) bool { return d.Day() != 3 }}, act: func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("3"))
			ts.Tap(gluttertest.ByText("OK"))
		}, want: day(2025, 6, 15)},
		{name: "year picker", act: func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.ByText("June 2025"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("2027"))
			ts.PumpAndSettle()
			ts.Tap(gluttertest.ByText("OK"))
		}, want: day(2027, 6, 15)},
		{name: "input mode", opts: ShowDatePickerOpts{InitialEntryMode: DatePickerEntryModeInput}, act: func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*widgets.EditableText](), "07/04/2026")
			ts.Tap(gluttertest.ByText("OK"))
		}, want: day(2026, 7, 4)},
		{name: "input mode rejects an out of range date", opts: ShowDatePickerOpts{InitialEntryMode: DatePickerEntryModeInput}, act: func(ts *gluttertest.Tester) {
			ts.EnterText(gluttertest.ByType[*widgets.EditableText](), "07/04/2040")
			ts.Tap(gluttertest.ByText("OK"))
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText("Out of range.")) {
				t.Error("no error text")
			}
			ts.Tap(gluttertest.ByText("Cancel"))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var result <-chan time.Time
			opts := tc.opts
			opts.InitialDate, opts.CurrentDate = day(2025, 6, 15), day(2025, 6, 10)
			ts := pumpPickerApp(t, ui.Size{Width: 800, Height: 800}, func(bc widgets.BuildContext) { result = ShowDatePicker(bc, first, last, opts) })
			tc.act(ts)
			ts.PumpAndSettle()
			if errs := ts.Errors(); len(errs) > 0 {
				t.Fatalf("errors: %v", errs)
			}
			if got := <-result; !got.Equal(tc.want) {
				t.Errorf("picked %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDatePickerEntryModeToggle(t *testing.T) {
	var modes []DatePickerEntryMode
	var result <-chan time.Time
	ts := pumpPickerApp(t, ui.Size{Width: 800, Height: 800}, func(bc widgets.BuildContext) {
		result = ShowDatePicker(bc, day(2020, 1, 1), day(2030, 1, 1), ShowDatePickerOpts{InitialDate: day(2025, 6, 15), CurrentDate: day(2025, 6, 10),
			OnDatePickerModeChange: func(m DatePickerEntryMode) { modes = append(modes, m) }})
	})
	ts.Tap(gluttertest.ByIcon(Icons.EditOutlined()))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByType[*widgets.EditableText]()) != 1 {
		t.Fatal("input mode has no text field")
	}
	ts.Tap(gluttertest.ByIcon(Icons.CalendarToday()))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByType[*widgets.EditableText]()) != 0 {
		t.Fatal("calendar mode has a text field")
	}
	if len(modes) != 2 || modes[0] != DatePickerEntryModeInput || modes[1] != DatePickerEntryModeCalendar {
		t.Errorf("modes %v", modes)
	}
	ts.Tap(gluttertest.ByText("OK"))
	ts.PumpAndSettle()
	if got := <-result; !got.Equal(day(2025, 6, 15)) {
		t.Errorf("got %v", got)
	}
}

func TestShowDateRangePickerResults(t *testing.T) {
	first, last := day(2020, 1, 1), day(2030, 12, 31)
	tests := []struct {
		name string
		opts ShowDateRangePickerOpts
		act  func(ts *gluttertest.Tester)
		want foundation.Opt[DateTimeRange]
	}{
		{name: "close", act: func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByIcon(Icons.Close())) }, want: foundation.None[DateTimeRange]()},
		{name: "initial range", opts: ShowDateRangePickerOpts{InitialDateRange: someRange(day(2025, 6, 10), day(2025, 6, 12))},
			act: func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Save")) }, want: someRange(day(2025, 6, 10), day(2025, 6, 12))},
		{name: "pick two days", opts: ShowDateRangePickerOpts{InitialDateRange: someRange(day(2025, 6, 10), day(2025, 6, 12))}, act: func(ts *gluttertest.Tester) {
			ts.Tap(gluttertest.First(gluttertest.ByText("20")))
			ts.Tap(gluttertest.First(gluttertest.ByText("23")))
			ts.Tap(gluttertest.ByText("Save"))
		}, want: someRange(day(2025, 6, 20), day(2025, 6, 23))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var result <-chan DateTimeRange
			opts := tc.opts
			opts.CurrentDate = day(2025, 6, 10)
			ts := pumpPickerApp(t, ui.Size{Width: 800, Height: 900}, func(bc widgets.BuildContext) { result = ShowDateRangePicker(bc, first, last, opts) })
			tc.act(ts)
			ts.PumpAndSettle()
			if errs := ts.Errors(); len(errs) > 0 {
				t.Fatalf("errors: %v", errs)
			}
			got := <-result
			want, ok := tc.want.Get()
			if ok && !got.Equal(want) || !ok && !got.IsZero() {
				t.Errorf("picked %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCalendarDatePickerAndYearPicker(t *testing.T) {
	var picked, year time.Time
	var displayed []time.Time
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 700}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: linuxTheme(), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewColumn([]widgets.Widget{
		widgets.NewSizedBox(NewCalendarDatePicker(day(2025, 6, 15), day(2020, 1, 1), day(2030, 12, 31), func(d time.Time) { picked = d },
			CalendarDatePickerOpts{CurrentDate: day(2025, 6, 10), OnDisplayedMonthChanged: func(d time.Time) { displayed = append(displayed, d) }}),
			widgets.SizedBoxOpts{Height: foundation.Some(350.0)}),
		widgets.NewSizedBox(NewYearPicker(day(2020, 1, 1), day(2030, 12, 31), day(2025, 6, 15), func(d time.Time) { year = d },
			YearPickerOpts{CurrentDate: day(2025, 6, 10)}), widgets.SizedBoxOpts{Height: foundation.Some(300.0)}),
	})})}))
	ts.PumpAndSettle()
	if errs := ts.Errors(); len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	ts.Tap(gluttertest.First(gluttertest.ByText("21")))
	if !picked.Equal(day(2025, 6, 21)) {
		t.Errorf("picked %v", picked)
	}
	ts.Tap(gluttertest.ByIcon(Icons.ChevronLeft()))
	ts.PumpAndSettle()
	if len(displayed) != 1 || !displayed[0].Equal(day(2025, 5, 1)) {
		t.Errorf("displayed %v", displayed)
	}
	ts.Tap(gluttertest.ByText("2023"))
	if year.Year() != 2023 || year.Month() != 6 {
		t.Errorf("year picked %v", year)
	}
}

func TestInputDatePickerFormField(t *testing.T) {
	var submitted, saved time.Time
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 500, Height: 400}})
	formKey := widgets.NewGlobalKey[*widgets.FormState]()
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: linuxTheme(), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewForm(
		NewInputDatePickerFormField(day(2020, 1, 1), day(2030, 12, 31), InputDatePickerFormFieldOpts{
			InitialDate: day(2025, 6, 15), OnDateSubmitted: func(d time.Time) { submitted = d }, OnDateSaved: func(d time.Time) { saved = d },
		}), widgets.FormOpts{Key: formKey})})}))
	ts.PumpAndSettle()
	tests := []struct {
		name  string
		input string
		valid bool
		want  time.Time
	}{
		{"valid", "07/04/2026", true, day(2026, 7, 4)},
		{"not a date", "hello", false, time.Time{}},
		{"out of range", "01/01/2019", false, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved = time.Time{}
			ts.EnterText(gluttertest.ByType[*widgets.EditableText](), tt.input)
			form, _ := formKey.CurrentState()
			if got := form.Validate(); got != tt.valid {
				t.Errorf("Validate = %v", got)
			}
			form.Save()
			ts.PumpAndSettle()
			if tt.valid && !saved.Equal(tt.want) {
				t.Errorf("saved %v, want %v", saved, tt.want)
			}
			if !tt.valid && !saved.IsZero() {
				t.Errorf("saved %v for an invalid entry", saved)
			}
		})
	}
	_ = submitted
}

func TestPickerThemeData(t *testing.T) {
	t.Run("date picker", func(t *testing.T) {
		a := DatePickerThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(0.0)}
		b := DatePickerThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(10.0)}
		got := DatePickerThemeDataLerp(a, b, 0.5)
		if got.Elevation != foundation.Some(5.0) || got.BackgroundColor != foundation.Some(lerpColor(a.BackgroundColor.Value(), b.BackgroundColor.Value(), 0.5)) {
			t.Errorf("lerp = %v", got)
		}
		if !a.Equal(a.CopyWith(DatePickerThemeDataCopy{})) || a.Equal(a.CopyWith(DatePickerThemeDataCopy{Elevation: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
		if !NewThemeData().DatePickerTheme.Equal(DatePickerThemeData{}) {
			t.Error("the default theme has a date picker theme")
		}
	})
	t.Run("time picker", func(t *testing.T) {
		a := TimePickerThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(0.0)}
		b := TimePickerThemeData{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(10.0)}
		got := TimePickerThemeDataLerp(a, b, 0.5)
		if got.Elevation != foundation.Some(5.0) || got.BackgroundColor != foundation.Some(lerpColor(a.BackgroundColor.Value(), b.BackgroundColor.Value(), 0.5)) {
			t.Errorf("lerp = %v", got)
		}
		if !a.Equal(a.CopyWith(TimePickerThemeDataCopy{})) || a.Equal(a.CopyWith(TimePickerThemeDataCopy{Elevation: foundation.Some(1.0)})) {
			t.Error("Equal / CopyWith")
		}
		if !NewThemeData().TimePickerTheme.Equal(TimePickerThemeData{}) {
			t.Error("the default theme has a time picker theme")
		}
	})
}

func TestPickerErrors(t *testing.T) {
	skipUnlessDebug(t)
	a, b, c := day(2025, 1, 1), day(2025, 12, 31), day(2026, 6, 1)
	noop := func(time.Time) {}
	build := func(mk func() widgets.Widget) func() {
		return func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: mk()})}))
			takeAndPanic(ts)
		}
	}
	show := func(f func(bc widgets.BuildContext)) func() {
		return func() {
			var captured widgets.BuildContext
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				captured = bc
				return widgets.NewText("HOME")
			})}))
			f(captured)
		}
	}
	never := func(time.Time) bool { return false }
	tests := []struct {
		name  string
		f     func()
		route string
	}{
		{"range end before start", func() { NewDateTimeRange(b, a) }, "DateTimeRangeEndBeforeStart"},
		{"calendar last before first", build(func() widgets.Widget { return NewCalendarDatePicker(a, b, a, noop) }), "CalendarDatePickerLastDateBeforeFirstDate"},
		{"calendar initial before first", build(func() widgets.Widget { return NewCalendarDatePicker(a, day(2025, 6, 1), b, noop) }), "CalendarDatePickerInitialDateBeforeFirstDate"},
		{"calendar initial after last", build(func() widgets.Widget { return NewCalendarDatePicker(c, a, b, noop) }), "CalendarDatePickerInitialDateAfterLastDate"},
		{"calendar initial not selectable", build(func() widgets.Widget {
			return NewCalendarDatePicker(day(2025, 6, 1), a, b, noop, CalendarDatePickerOpts{SelectableDayPredicate: never})
		}), "CalendarDatePickerInitialDateNotSelectable"},
		{"year picker first after last", build(func() widgets.Widget { return NewYearPicker(b, a, a, noop) }), "YearPickerFirstDateAfterLastDate"},
		{"input last before first", build(func() widgets.Widget { return NewInputDatePickerFormField(b, a) }), "InputDatePickerLastDateBeforeFirstDate"},
		{"input initial before first", build(func() widgets.Widget {
			return NewInputDatePickerFormField(day(2025, 6, 1), b, InputDatePickerFormFieldOpts{InitialDate: a})
		}), "InputDatePickerInitialDateBeforeFirstDate"},
		{"input initial after last", build(func() widgets.Widget {
			return NewInputDatePickerFormField(a, b, InputDatePickerFormFieldOpts{InitialDate: c})
		}), "InputDatePickerInitialDateAfterLastDate"},
		{"input initial not selectable", build(func() widgets.Widget {
			return NewInputDatePickerFormField(a, b, InputDatePickerFormFieldOpts{InitialDate: day(2025, 6, 1), SelectableDayPredicate: never})
		}), "InputDatePickerInitialDateNotSelectable"},
		{"dialog last before first", func() { NewDatePickerDialog(b, a) }, "DatePickerDialogLastDateBeforeFirstDate"},
		{"dialog initial before first", func() { NewDatePickerDialog(day(2025, 6, 1), b, DatePickerDialogOpts{InitialDate: a}) }, "DatePickerDialogInitialDateBeforeFirstDate"},
		{"dialog initial after last", func() { NewDatePickerDialog(a, b, DatePickerDialogOpts{InitialDate: c}) }, "DatePickerDialogInitialDateAfterLastDate"},
		{"dialog initial not selectable", func() {
			NewDatePickerDialog(a, b, DatePickerDialogOpts{InitialDate: day(2025, 6, 1), SelectableDayPredicate: never})
		}, "DatePickerDialogInitialDateNotSelectable"},
		{"show last before first", show(func(bc widgets.BuildContext) { ShowDatePicker(bc, b, a) }), "ShowDatePickerLastDateBeforeFirstDate"},
		{"show initial before first", show(func(bc widgets.BuildContext) {
			ShowDatePicker(bc, day(2025, 6, 1), b, ShowDatePickerOpts{InitialDate: a})
		}), "ShowDatePickerInitialDateBeforeFirstDate"},
		{"show initial after last", show(func(bc widgets.BuildContext) { ShowDatePicker(bc, a, b, ShowDatePickerOpts{InitialDate: c}) }), "ShowDatePickerInitialDateAfterLastDate"},
		{"show initial not selectable", show(func(bc widgets.BuildContext) {
			ShowDatePicker(bc, a, b, ShowDatePickerOpts{InitialDate: day(2025, 6, 1), SelectableDayPredicate: never})
		}), "ShowDatePickerInitialDateNotSelectable"},
		{"range show last before first", show(func(bc widgets.BuildContext) { ShowDateRangePicker(bc, b, a) }), "ShowDateRangePickerLastDateBeforeFirstDate"},
		{"range show out of bounds", show(func(bc widgets.BuildContext) {
			ShowDateRangePicker(bc, a, b, ShowDateRangePickerOpts{InitialDateRange: someRange(a, c)})
		}), "ShowDateRangePickerInitialRangeOutOfBounds"},
		{"range show not selectable", show(func(bc widgets.BuildContext) {
			ShowDateRangePicker(bc, a, b, ShowDateRangePickerOpts{InitialDateRange: someRange(a, day(2025, 6, 1)),
				SelectableDayPredicate: func(d, s, e time.Time) bool { return false }})
		}), "ShowDateRangePickerInitialRangeNotSelectable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.f), tt.route)
		})
	}
}
