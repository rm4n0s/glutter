// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func pickerDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// TestPickerInternalErrors checks the failures of the parts of the pickers that
// an app cannot reach directly.
func TestPickerInternalErrors(t *testing.T) {
	skipUnlessDebug(t)
	first, last, outside := pickerDay(2025, 1, 1), pickerDay(2025, 12, 31), pickerDay(2026, 6, 1)
	d := NewGregorianCalendarDelegate()
	tests := []struct {
		name  string
		f     func()
		route string
	}{
		{"month picker first after last", func() { newMonthPicker(monthPickerArgs{firstDate: last, lastDate: first, calendarDelegate: d}) }, "MonthPickerFirstDateAfterLastDate"},
		{"month picker selection out of range", func() {
			newMonthPicker(monthPickerArgs{firstDate: first, lastDate: last, selectedDate: outside, calendarDelegate: d})
		}, "MonthPickerSelectedDateOutOfRange"},
		{"month item first after last", func() { newMonthItem(monthItemArgs{firstDate: last, lastDate: first, calendarDelegate: d}) }, "MonthItemFirstDateAfterLastDate"},
		{"month item selection out of range", func() {
			newMonthItem(monthItemArgs{firstDate: first, lastDate: last, selectedDateStart: outside, calendarDelegate: d})
		}, "MonthItemSelectionOutOfRange"},
		{"month item selection reversed", func() {
			newMonthItem(monthItemArgs{firstDate: first, lastDate: last, selectedDateStart: pickerDay(2025, 6, 5), selectedDateEnd: pickerDay(2025, 6, 1), calendarDelegate: d})
		}, "MonthItemSelectionReversed"},
		{"range picker start after end", func() {
			newCalendarDateRangePicker(calendarDateRangePickerArgs{initialStartDate: pickerDay(2025, 6, 5), initialEndDate: pickerDay(2025, 6, 1), firstDate: first, lastDate: last, calendarDelegate: d})
		}, "CalendarDateRangePickerStartAfterEnd"},
		{"range picker last before first", func() {
			newCalendarDateRangePicker(calendarDateRangePickerArgs{firstDate: last, lastDate: first, calendarDelegate: d})
		}, "CalendarDateRangePickerLastDateBeforeFirstDate"},
		{"date picker dialog entry mode fixed", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewDatePickerDialog(first, last, DatePickerDialogOpts{
				InitialDate: pickerDay(2025, 6, 1), InitialEntryMode: DatePickerEntryModeCalendarOnly})})}))
			ts.PumpAndSettle()
			gluttertest.StateOf[*datePickerDialogState](ts, gluttertest.ByType[*DatePickerDialog]()).handleEntryModeToggle()
			takeAndPanic(ts)
		}, "DatePickerDialogEntryModeFixed"},
		{"date range picker dialog entry mode fixed", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 900}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDateRangePickerDialog(first, last, DateRangePickerDialogOpts{InitialEntryMode: DatePickerEntryModeCalendarOnly})}))
			ts.PumpAndSettle()
			gluttertest.StateOf[*dateRangePickerDialogState](ts, gluttertest.ByType[*DateRangePickerDialog]()).handleEntryModeToggle()
			takeAndPanic(ts)
		}, "DateRangePickerDialogEntryModeFixed"},
		{"dial header outside the dial modes", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTimePickerDialog(TimeOfDay{Hour: 1}, TimePickerDialogOpts{InitialEntryMode: TimePickerEntryModeInput})})}))
			ts.PumpAndSettle()
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				theme := ThemeOf(bc)
				return &timePickerModel{
					InheritedModel: widgets.NewInheritedModelBase(newDialTimePickerHeader("help")),
					entryMode:      TimePickerEntryModeInput, defaultTheme: timePickerDefaultsM3(theme, TimePickerEntryModeInput),
				}
			})})}))
			takeAndPanic(ts)
		}, "TimePickerNotInDialMode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRoute(t, recoverError(tt.f), tt.route)
		})
	}
}

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only checks of the framework")
	}
}

func takeAndPanic(ts *gluttertest.Tester) {
	if errs := ts.TakeErrors(); len(errs) > 0 {
		panic(errs[0])
	}
}

func recoverError(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	f()
	return nil
}

func wantRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), route)
	}
}
