// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// DatePickerEntryMode tells which method the date picker dialog starts with,
// and which the user can switch to.
//
// Flutter: material/date.dart DatePickerEntryMode
type DatePickerEntryMode int

const (
	// DatePickerEntryModeUnspecified is the zero value: the default (calendar).
	//
	// Flutter: no counterpart
	DatePickerEntryModeUnspecified DatePickerEntryMode = iota
	// DatePickerEntryModeCalendar is a calendar grid to pick a date; the user can
	// switch to the input mode.
	//
	// Flutter: material/date.dart DatePickerEntryMode.calendar
	DatePickerEntryModeCalendar
	// DatePickerEntryModeInput is a text field to type a date; the user can switch
	// to the calendar mode.
	//
	// Flutter: material/date.dart DatePickerEntryMode.input
	DatePickerEntryModeInput
	// DatePickerEntryModeCalendarOnly is the calendar grid, without a way to switch
	// to the input mode.
	//
	// Flutter: material/date.dart DatePickerEntryMode.calendarOnly
	DatePickerEntryModeCalendarOnly
	// DatePickerEntryModeInputOnly is the text field, without a way to switch to
	// the calendar mode.
	//
	// Flutter: material/date.dart DatePickerEntryMode.inputOnly
	DatePickerEntryModeInputOnly
)

// Name returns the Dart enum name.
//
// Flutter: material/date.dart DatePickerEntryMode.name
func (m DatePickerEntryMode) Name() string {
	return enumName(int(m), "calendar", "input", "calendarOnly", "inputOnly")
}

// String returns the name Dart prints, such as "DatePickerEntryMode.input".
//
// Flutter: material/date.dart DatePickerEntryMode.toString
func (m DatePickerEntryMode) String() string { return "DatePickerEntryMode." + m.Name() }

// DatePickerMode determines which picker of a date picker is shown: the days of
// a month, or the years.
//
// Flutter: material/date.dart DatePickerMode
type DatePickerMode int

const (
	// DatePickerModeDay shows the days of a month; it is the zero value.
	//
	// Flutter: material/date.dart DatePickerMode.day
	DatePickerModeDay DatePickerMode = iota
	// DatePickerModeYear shows a list of years.
	//
	// Flutter: material/date.dart DatePickerMode.year
	DatePickerModeYear
)

// Name returns the Dart enum name.
//
// Flutter: material/date.dart DatePickerMode.name
func (m DatePickerMode) Name() string {
	if m == DatePickerModeYear {
		return "year"
	}
	return "day"
}

// String returns the name Dart prints, such as "DatePickerMode.year".
//
// Flutter: material/date.dart DatePickerMode.toString
func (m DatePickerMode) String() string { return "DatePickerMode." + m.Name() }

// DateTimeRange is a range of dates: the two ends are part of it.
//
// Flutter: material/date.dart DateTimeRange
type DateTimeRange struct {
	// Start is the start of the range.
	Start time.Time
	// End is the end of the range; it is not before Start.
	End time.Time
}

// NewDateTimeRange creates a range. In debug builds it panics with the tag
// DateTimeRangeEndBeforeStart when the start is after the end.
//
// Flutter: material/date.dart DateTimeRange.new
func NewDateTimeRange(start, end time.Time) DateTimeRange {
	if foundation.DebugMode && start.After(end) {
		panic(errors.New("DateTimeRangeEndBeforeStart", "the start of a date range must not be after its end", "start", start, "end", end))
	}
	return DateTimeRange{Start: start, End: end}
}

// Duration is the time between the start and the end.
//
// Flutter: material/date.dart DateTimeRange.duration
func (r DateTimeRange) Duration() time.Duration { return r.End.Sub(r.Start) }

// IsZero reports whether the range is the zero range: the result of a date range
// picker that was cancelled.
//
// Flutter: no counterpart
func (r DateTimeRange) IsZero() bool { return r.Start.IsZero() && r.End.IsZero() }

// Equal reports whether the two ranges start and end at the same instants.
//
// Flutter: material/date.dart DateTimeRange.==
func (r DateTimeRange) Equal(o DateTimeRange) bool {
	return r.Start.Equal(o.Start) && r.End.Equal(o.End)
}

// String returns the start and the end, as Dart prints them.
//
// Flutter: material/date.dart DateTimeRange.toString
func (r DateTimeRange) String() string { return r.Start.String() + " - " + r.End.String() }

// CalendarDelegate is the interface to the calendar system that a date picker
// works with. GregorianCalendarDelegate is the default.
//
// Deviation: Dart's delegate is generic over a subclass of DateTime, so that a
// calendar can have its own date class. Go's time.Time cannot be extended, so a
// delegate of another calendar system stores the dates as the time.Time of the
// Gregorian date they stand for (SPEC §12.10).
//
// Flutter: material/date.dart CalendarDelegate
type CalendarDelegate interface {
	// Now returns the current date and time.
	//
	// Flutter: material/date.dart CalendarDelegate.now
	Now() time.Time
	// DateOnly returns the date with the time of day removed.
	//
	// Flutter: material/date.dart CalendarDelegate.dateOnly
	DateOnly(date time.Time) time.Time
	// IsSameDay reports whether the two dates are on the same day.
	//
	// Flutter: material/date.dart CalendarDelegate.isSameDay
	IsSameDay(a, b time.Time) bool
	// IsSameMonth reports whether the two dates are in the same month.
	//
	// Flutter: material/date.dart CalendarDelegate.isSameMonth
	IsSameMonth(a, b time.Time) bool
	// MonthDelta returns the number of months between the months of the two
	// dates.
	//
	// Flutter: material/date.dart CalendarDelegate.monthDelta
	MonthDelta(start, end time.Time) int
	// AddMonthsToMonthDate returns the date of the first day of the month that is
	// monthsToAdd months from the one of monthDate.
	//
	// Flutter: material/date.dart CalendarDelegate.addMonthsToMonthDate
	AddMonthsToMonthDate(monthDate time.Time, monthsToAdd int) time.Time
	// AddDaysToDate returns the date that is days days from the date.
	//
	// Flutter: material/date.dart CalendarDelegate.addDaysToDate
	AddDaysToDate(date time.Time, days int) time.Time
	// FirstDayOffset returns the number of empty cells before the first day of the
	// month in a week grid that starts at the first day of the week of the
	// localizations.
	//
	// Flutter: material/date.dart CalendarDelegate.firstDayOffset
	FirstDayOffset(year, month int, localizations MaterialLocalizations) int
	// DaysInMonth returns the number of days of the month.
	//
	// Flutter: material/date.dart CalendarDelegate.getDaysInMonth
	DaysInMonth(year, month int) int
	// GetMonth returns the date of the first day of the month.
	//
	// Flutter: material/date.dart CalendarDelegate.getMonth
	GetMonth(year, month int) time.Time
	// GetDay returns the date of the day.
	//
	// Flutter: material/date.dart CalendarDelegate.getDay
	GetDay(year, month, day int) time.Time
	// FormatMonthYear formats the month and the year of the date, such as
	// "September 2017".
	//
	// Flutter: material/date.dart CalendarDelegate.formatMonthYear
	FormatMonthYear(date time.Time, localizations MaterialLocalizations) string
	// FormatYear formats the year, such as "2017".
	//
	// Flutter: material/date.dart CalendarDelegate.formatYear
	FormatYear(year int, localizations MaterialLocalizations) string
	// FormatMediumDate formats the date as "Sun, Sep 17".
	//
	// Flutter: material/date.dart CalendarDelegate.formatMediumDate
	FormatMediumDate(date time.Time, localizations MaterialLocalizations) string
	// FormatShortMonthDay formats the month and the day, such as "Sep 17".
	//
	// Flutter: material/date.dart CalendarDelegate.formatShortMonthDay
	FormatShortMonthDay(date time.Time, localizations MaterialLocalizations) string
	// FormatShortDate formats the date as "Sep 17, 2017".
	//
	// Flutter: material/date.dart CalendarDelegate.formatShortDate
	FormatShortDate(date time.Time, localizations MaterialLocalizations) string
	// FormatFullDate formats the date as "Sunday, September 17, 2017".
	//
	// Flutter: material/date.dart CalendarDelegate.formatFullDate
	FormatFullDate(date time.Time, localizations MaterialLocalizations) string
	// FormatCompactDate formats the date as "09/17/2017".
	//
	// Flutter: material/date.dart CalendarDelegate.formatCompactDate
	FormatCompactDate(date time.Time, localizations MaterialLocalizations) string
	// ParseCompactDate parses the text that FormatCompactDate makes; the result is
	// false when the text is not a date.
	//
	// Flutter: material/date.dart CalendarDelegate.parseCompactDate
	ParseCompactDate(input string, localizations MaterialLocalizations) (time.Time, bool)
	// DateHelpText is the help text of the date field, such as "mm/dd/yyyy".
	//
	// Flutter: material/date.dart CalendarDelegate.dateHelpText
	DateHelpText(localizations MaterialLocalizations) string
}

// GregorianCalendarDelegate is the CalendarDelegate of the Gregorian calendar.
//
// Flutter: material/date.dart GregorianCalendarDelegate
type GregorianCalendarDelegate struct{}

// NewGregorianCalendarDelegate creates the delegate.
//
// Flutter: material/date.dart GregorianCalendarDelegate.new
func NewGregorianCalendarDelegate() GregorianCalendarDelegate { return GregorianCalendarDelegate{} }

// Now returns the current local time.
//
// Flutter: material/date.dart GregorianCalendarDelegate.now
func (GregorianCalendarDelegate) Now() time.Time { return time.Now() }

// DateOnly returns the date at midnight.
//
// Flutter: material/date.dart GregorianCalendarDelegate.dateOnly
func (GregorianCalendarDelegate) DateOnly(date time.Time) time.Time { return DateUtilsDateOnly(date) }

// IsSameDay reports whether the two dates are on the same day.
//
// Flutter: material/date.dart CalendarDelegate.isSameDay
func (GregorianCalendarDelegate) IsSameDay(a, b time.Time) bool { return DateUtilsIsSameDay(a, b) }

// IsSameMonth reports whether the two dates are in the same month.
//
// Flutter: material/date.dart CalendarDelegate.isSameMonth
func (GregorianCalendarDelegate) IsSameMonth(a, b time.Time) bool { return DateUtilsIsSameMonth(a, b) }

// MonthDelta returns the number of months between the two dates.
//
// Flutter: material/date.dart GregorianCalendarDelegate.monthDelta
func (GregorianCalendarDelegate) MonthDelta(start, end time.Time) int {
	return DateUtilsMonthDelta(start, end)
}

// AddMonthsToMonthDate adds the months to the date of the month.
//
// Flutter: material/date.dart GregorianCalendarDelegate.addMonthsToMonthDate
func (GregorianCalendarDelegate) AddMonthsToMonthDate(monthDate time.Time, monthsToAdd int) time.Time {
	return DateUtilsAddMonthsToMonthDate(monthDate, monthsToAdd)
}

// AddDaysToDate adds the days to the date.
//
// Flutter: material/date.dart GregorianCalendarDelegate.addDaysToDate
func (GregorianCalendarDelegate) AddDaysToDate(date time.Time, days int) time.Time {
	return DateUtilsAddDaysToDate(date, days)
}

// FirstDayOffset returns the number of empty cells before the first day.
//
// Flutter: material/date.dart GregorianCalendarDelegate.firstDayOffset
func (GregorianCalendarDelegate) FirstDayOffset(year, month int, localizations MaterialLocalizations) int {
	return DateUtilsFirstDayOffset(year, month, localizations)
}

// DaysInMonth returns the number of days of the month.
//
// Flutter: material/date.dart GregorianCalendarDelegate.getDaysInMonth
func (GregorianCalendarDelegate) DaysInMonth(year, month int) int {
	return DateUtilsGetDaysInMonth(year, month)
}

// GetMonth returns the first day of the month.
//
// Flutter: material/date.dart GregorianCalendarDelegate.getMonth
func (GregorianCalendarDelegate) GetMonth(year, month int) time.Time {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
}

// GetDay returns the day.
//
// Flutter: material/date.dart GregorianCalendarDelegate.getDay
func (GregorianCalendarDelegate) GetDay(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

// FormatMonthYear formats the month and the year.
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatMonthYear
func (GregorianCalendarDelegate) FormatMonthYear(date time.Time, l MaterialLocalizations) string {
	return l.FormatMonthYear(date)
}

// FormatYear formats the year.
//
// Flutter: material/date.dart CalendarDelegate.formatYear
func (GregorianCalendarDelegate) FormatYear(year int, l MaterialLocalizations) string {
	return l.FormatYear(time.Date(year, 1, 1, 0, 0, 0, 0, time.Local))
}

// FormatMediumDate formats the date as "Sun, Sep 17".
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatMediumDate
func (GregorianCalendarDelegate) FormatMediumDate(date time.Time, l MaterialLocalizations) string {
	return l.FormatMediumDate(date)
}

// FormatShortMonthDay formats the month and the day.
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatShortMonthDay
func (GregorianCalendarDelegate) FormatShortMonthDay(date time.Time, l MaterialLocalizations) string {
	return l.FormatShortMonthDay(date)
}

// FormatShortDate formats the date as "Sep 17, 2017".
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatShortDate
func (GregorianCalendarDelegate) FormatShortDate(date time.Time, l MaterialLocalizations) string {
	return l.FormatShortDate(date)
}

// FormatFullDate formats the date as "Sunday, September 17, 2017".
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatFullDate
func (GregorianCalendarDelegate) FormatFullDate(date time.Time, l MaterialLocalizations) string {
	return l.FormatFullDate(date)
}

// FormatCompactDate formats the date as "09/17/2017".
//
// Flutter: material/date.dart GregorianCalendarDelegate.formatCompactDate
func (GregorianCalendarDelegate) FormatCompactDate(date time.Time, l MaterialLocalizations) string {
	return l.FormatCompactDate(date)
}

// ParseCompactDate parses the compact form of a date.
//
// Flutter: material/date.dart GregorianCalendarDelegate.parseCompactDate
func (GregorianCalendarDelegate) ParseCompactDate(input string, l MaterialLocalizations) (time.Time, bool) {
	return l.ParseCompactDate(input)
}

// DateHelpText is the help text of the date field.
//
// Flutter: material/date.dart GregorianCalendarDelegate.dateHelpText
func (GregorianCalendarDelegate) DateHelpText(l MaterialLocalizations) string {
	return l.DateHelpText()
}

// DateUtilsDateOnly returns the date with the time of day set to midnight, in
// the location of the date.
//
// Flutter: material/date.dart DateUtils.dateOnly
func DateUtilsDateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

// DateUtilsDatesOnly returns the range with the time of day removed from both
// ends.
//
// Flutter: material/date.dart DateUtils.datesOnly
func DateUtilsDatesOnly(r DateTimeRange) DateTimeRange {
	return NewDateTimeRange(DateUtilsDateOnly(r.Start), DateUtilsDateOnly(r.End))
}

// DateUtilsIsSameDay reports whether the two dates are on the same day. A zero
// time stands for null, and is the same day only as another zero time.
//
// Flutter: material/date.dart DateUtils.isSameDay
func DateUtilsIsSameDay(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// DateUtilsIsSameMonth reports whether the two dates are in the same month. A
// zero time stands for null.
//
// Flutter: material/date.dart DateUtils.isSameMonth
func DateUtilsIsSameMonth(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// DateUtilsMonthDelta returns the number of months between the months of the two
// dates; it is negative when the end is before the start.
//
// Flutter: material/date.dart DateUtils.monthDelta
func DateUtilsMonthDelta(start, end time.Time) int {
	return (end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month())
}

// DateUtilsAddMonthsToMonthDate returns the first day of the month that is
// monthsToAdd months from the month of monthDate. The day of monthDate is
// ignored.
//
// Flutter: material/date.dart DateUtils.addMonthsToMonthDate
func DateUtilsAddMonthsToMonthDate(monthDate time.Time, monthsToAdd int) time.Time {
	return time.Date(monthDate.Year(), monthDate.Month()+time.Month(monthsToAdd), 1, 0, 0, 0, 0, monthDate.Location())
}

// DateUtilsAddDaysToDate returns the date that is days days after the date, at
// midnight.
//
// Flutter: material/date.dart DateUtils.addDaysToDate
func DateUtilsAddDaysToDate(date time.Time, days int) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day()+days, 0, 0, 0, 0, date.Location())
}

// DateUtilsFirstDayOffset returns the number of empty cells before the first day
// of the month in a week grid that starts at the first day of the week of the
// localizations.
//
// Flutter: material/date.dart DateUtils.firstDayOffset
func DateUtilsFirstDayOffset(year, month int, localizations MaterialLocalizations) int {
	// 0-based day of week for the month and year, with 0 representing Monday.
	weekdayFromMonday := (int(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).Weekday()) + 6) % 7
	// 0-based start of week depending on the locale, with 0 representing Sunday.
	firstDayOfWeekIndex := localizations.FirstDayOfWeekIndex()
	// firstDayOfWeekIndex recomputed to be Monday-based, in order to compare with
	// weekdayFromMonday.
	firstDayOfWeekIndex = mod(firstDayOfWeekIndex-1, 7)
	// Number of days between the first day of week appearing on the calendar, and
	// the day corresponding to the first of the month.
	return mod(weekdayFromMonday-firstDayOfWeekIndex, 7)
}

// DateUtilsGetDaysInMonth returns the number of days of the month, in the
// Gregorian calendar.
//
// Flutter: material/date.dart DateUtils.getDaysInMonth
func DateUtilsGetDaysInMonth(year, month int) int {
	if month == int(time.February) {
		isLeapYear := (year%4 == 0 && year%100 != 0) || year%400 == 0
		if isLeapYear {
			return 29
		}
		return 28
	}
	return [...]int{31, -1, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[month-1]
}

// mod is the modulo of Dart's % operator on integers: never negative for a
// positive divisor.
func mod(a, n int) int { return ((a % n) + n) % n }
