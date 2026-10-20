package material

import (
	"fmt"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/widgets"
)

// DayPeriod is a value that indicates whether a time of day is before or after
// midday.
//
// Flutter: material/time.dart DayPeriod
type DayPeriod int

const (
	// DayPeriodAM is the time between 0:00 and 11:59.
	//
	// Flutter: material/time.dart DayPeriod.am
	DayPeriodAM DayPeriod = iota
	// DayPeriodPM is the time between 12:00 and 23:59.
	//
	// Flutter: material/time.dart DayPeriod.pm
	DayPeriodPM
)

// Name returns the name of the day period, as Dart's Enum.name.
func (p DayPeriod) Name() string {
	if p == DayPeriodPM {
		return "pm"
	}
	return "am"
}

// Time constants of TimeOfDay.
const (
	// HoursPerDay is the number of hours in one day, i.e. 24.
	//
	// Flutter: material/time.dart TimeOfDay.hoursPerDay
	HoursPerDay = 24
	// HoursPerPeriod is the number of hours in one day period (see DayPeriod), i.e. 12.
	//
	// Flutter: material/time.dart TimeOfDay.hoursPerPeriod
	HoursPerPeriod = 12
	// MinutesPerHour is the number of minutes in one hour, i.e. 60.
	//
	// Flutter: material/time.dart TimeOfDay.minutesPerHour
	MinutesPerHour = 60
)

// TimeOfDay is a value representing a time during the day, independent of the
// date that day might fall on or the time zone.
//
// Flutter: material/time.dart TimeOfDay
type TimeOfDay struct {
	// Hour is the selected hour, in 24 hour time from 0..23.
	//
	// Flutter: material/time.dart TimeOfDay.hour
	Hour int
	// Minute is the selected minute.
	//
	// Flutter: material/time.dart TimeOfDay.minute
	Minute int
}

// NewTimeOfDayFromDateTime returns a time of day from the hour and the minute
// of a time.
//
// Flutter: material/time.dart TimeOfDay.fromDateTime
func NewTimeOfDayFromDateTime(t time.Time) TimeOfDay { return TimeOfDay{t.Hour(), t.Minute()} }

// TimeOfDayOpts are the replacements of TimeOfDay.Replacing.
//
// Flutter: material/time.dart TimeOfDay.replacing
type TimeOfDayOpts struct {
	// Hour replaces the hour.
	Hour foundation.Opt[int]
	// Minute replaces the minute.
	Minute foundation.Opt[int]
}

// Replacing returns a new TimeOfDay with the hour or the minute replaced. In
// debug builds an hour outside 0..23 or a minute outside 0..59 panics with the
// tag TimeOfDayInvalid.
//
// Flutter: material/time.dart TimeOfDay.replacing
func (t TimeOfDay) Replacing(o TimeOfDayOpts) TimeOfDay {
	if foundation.DebugMode {
		if h, ok := o.Hour.Get(); ok && (h < 0 || h >= HoursPerDay) {
			panic(errors.New("TimeOfDayInvalid", "the hour must be from 0 to 23", "hour", h))
		}
		if m, ok := o.Minute.Get(); ok && (m < 0 || m >= MinutesPerHour) {
			panic(errors.New("TimeOfDayInvalid", "the minute must be from 0 to 59", "minute", m))
		}
	}
	return TimeOfDay{o.Hour.OrElse(t.Hour), o.Minute.OrElse(t.Minute)}
}

// Period is whether this time of day is before or after midday.
//
// Flutter: material/time.dart TimeOfDay.period
func (t TimeOfDay) Period() DayPeriod {
	if t.Hour < HoursPerPeriod {
		return DayPeriodAM
	}
	return DayPeriodPM
}

// HourOfPeriod is which hour of the current period (e.g., am or pm) this time is.
// For 12AM (midnight) and 12PM (noon) it returns 12.
//
// Flutter: material/time.dart TimeOfDay.hourOfPeriod
func (t TimeOfDay) HourOfPeriod() int {
	if t.Hour == 0 || t.Hour == 12 {
		return 12
	}
	return t.Hour - t.PeriodOffset()
}

// PeriodOffset is the number of hours that the period adds to the hour of the period.
//
// Flutter: material/time.dart TimeOfDay.periodOffset
func (t TimeOfDay) PeriodOffset() int {
	if t.Period() == DayPeriodAM {
		return 0
	}
	return HoursPerPeriod
}

// Format returns the localized string representation of this time of day, with
// the 24 hour setting of the MediaQuery.
//
// Flutter: material/time.dart TimeOfDay.format
func (t TimeOfDay) Format(bc widgets.BuildContext) string {
	widgets.DebugCheckHasMediaQuery(bc)
	DebugCheckHasMaterialLocalizations(bc)
	return MaterialLocalizationsOf(bc).FormatTimeOfDay(t, widgets.MediaQueryAlwaysUse24HourFormatOf(bc))
}

// IsBefore returns true if this time of day is before the other.
//
// Flutter: material/time.dart TimeOfDay.isBefore
func (t TimeOfDay) IsBefore(other TimeOfDay) bool { return t.CompareTo(other) < 0 }

// IsAfter returns true if this time of day is after the other.
//
// Flutter: material/time.dart TimeOfDay.isAfter
func (t TimeOfDay) IsAfter(other TimeOfDay) bool { return t.CompareTo(other) > 0 }

// IsAtSameTimeAs returns true if this time of day is the same as the other.
//
// Flutter: material/time.dart TimeOfDay.isAtSameTimeAs
func (t TimeOfDay) IsAtSameTimeAs(other TimeOfDay) bool { return t.CompareTo(other) == 0 }

// CompareTo compares the hour and then the minute.
//
// Flutter: material/time.dart TimeOfDay.compareTo
func (t TimeOfDay) CompareTo(other TimeOfDay) int {
	if t.Hour != other.Hour {
		if t.Hour < other.Hour {
			return -1
		}
		return 1
	}
	switch {
	case t.Minute < other.Minute:
		return -1
	case t.Minute > other.Minute:
		return 1
	}
	return 0
}

// String returns "TimeOfDay(09:05)".
//
// Flutter: material/time.dart TimeOfDay.toString
func (t TimeOfDay) String() string { return fmt.Sprintf("TimeOfDay(%02d:%02d)", t.Hour, t.Minute) }

// TimeOfDayFormat describes how to format a TimeOfDay as a string.
//
// Flutter: material/time.dart TimeOfDayFormat
type TimeOfDayFormat int

const (
	// TimeOfDayFormatHHColonMM is the 24-hour format: two digits for the hour, a colon, two digits for the minute ("13:04").
	//
	// Flutter: material/time.dart TimeOfDayFormat.HH_colon_mm
	TimeOfDayFormatHHColonMM TimeOfDayFormat = iota
	// TimeOfDayFormatHHDotMM is the 24-hour format with a dot ("13.04").
	//
	// Flutter: material/time.dart TimeOfDayFormat.HH_dot_mm
	TimeOfDayFormatHHDotMM
	// TimeOfDayFormatFrenchCanadian is the format used in French Canadian ("13 h 04").
	//
	// Flutter: material/time.dart TimeOfDayFormat.frenchCanadian
	TimeOfDayFormatFrenchCanadian
	// TimeOfDayFormatHColonMM is the 24-hour format without a leading zero ("3:04").
	//
	// Flutter: material/time.dart TimeOfDayFormat.H_colon_mm
	TimeOfDayFormatHColonMM
	// TimeOfDayFormatHColonMMSpaceA is the 12-hour format with the day period after ("1:04 PM").
	//
	// Flutter: material/time.dart TimeOfDayFormat.h_colon_mm_space_a
	TimeOfDayFormatHColonMMSpaceA
	// TimeOfDayFormatASpaceHColonMM is the 12-hour format with the day period before ("PM 1:04").
	//
	// Flutter: material/time.dart TimeOfDayFormat.a_space_h_colon_mm
	TimeOfDayFormatASpaceHColonMM
)

// HourFormat describes how to format the hour part of a time of day.
//
// Flutter: material/time.dart HourFormat
type HourFormat int

const (
	// HourFormatHH is two digits ("03").
	//
	// Flutter: material/time.dart HourFormat.HH
	HourFormatHH HourFormat = iota
	// HourFormatH is one or two digits ("3").
	//
	// Flutter: material/time.dart HourFormat.H
	HourFormatH
	// HourFormatLowerH is the hour of a 12-hour clock ("3").
	//
	// Flutter: material/time.dart HourFormat.h
	HourFormatLowerH
)

// HourFormatOf returns the hour format of a time of day format.
//
// Flutter: material/time.dart hourFormat
func HourFormatOf(of TimeOfDayFormat) HourFormat {
	switch of {
	case TimeOfDayFormatHColonMMSpaceA, TimeOfDayFormatASpaceHColonMM:
		return HourFormatLowerH
	case TimeOfDayFormatHColonMM:
		return HourFormatH
	}
	return HourFormatHH
}
