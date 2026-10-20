package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	. "github.com/rm4n0s/glutter/material"
)

func TestTimeOfDay(t *testing.T) {
	tests := []struct {
		name         string
		t            TimeOfDay
		period       DayPeriod
		hourOfPeriod int
		offset       int
		str          string
	}{
		{"midnight", TimeOfDay{0, 0}, DayPeriodAM, 12, 0, "TimeOfDay(00:00)"},
		{"morning", TimeOfDay{9, 5}, DayPeriodAM, 9, 0, "TimeOfDay(09:05)"},
		{"noon", TimeOfDay{12, 0}, DayPeriodPM, 12, 12, "TimeOfDay(12:00)"},
		{"afternoon", TimeOfDay{15, 30}, DayPeriodPM, 3, 12, "TimeOfDay(15:30)"},
		{"last minute", TimeOfDay{23, 59}, DayPeriodPM, 11, 12, "TimeOfDay(23:59)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.Period(); got != tt.period {
				t.Errorf("Period = %v, want %v", got, tt.period)
			}
			if got := tt.t.HourOfPeriod(); got != tt.hourOfPeriod {
				t.Errorf("HourOfPeriod = %d, want %d", got, tt.hourOfPeriod)
			}
			if got := tt.t.PeriodOffset(); got != tt.offset {
				t.Errorf("PeriodOffset = %d, want %d", got, tt.offset)
			}
			if got := tt.t.String(); got != tt.str {
				t.Errorf("String = %q, want %q", got, tt.str)
			}
		})
	}
}

func TestTimeOfDayCompareAndReplace(t *testing.T) {
	a, b := TimeOfDay{9, 30}, TimeOfDay{10, 0}
	if !a.IsBefore(b) || a.IsAfter(b) || a.IsAtSameTimeAs(b) || !b.IsAfter(a) || !a.IsAtSameTimeAs(TimeOfDay{9, 30}) {
		t.Error("comparison of times of day is wrong")
	}
	if got := a.Replacing(TimeOfDayOpts{Hour: foundation.Some(11)}); got != (TimeOfDay{11, 30}) {
		t.Errorf("Replacing hour = %v", got)
	}
	if got := a.Replacing(TimeOfDayOpts{Minute: foundation.Some(1)}); got != (TimeOfDay{9, 1}) {
		t.Errorf("Replacing minute = %v", got)
	}
	if got := NewTimeOfDayFromDateTime(time.Date(2020, 1, 2, 13, 14, 15, 0, time.UTC)); got != (TimeOfDay{13, 14}) {
		t.Errorf("FromDateTime = %v", got)
	}
}

func TestTimeOfDayReplacingInvalid(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		o    TimeOfDayOpts
	}{
		{"hour too big", TimeOfDayOpts{Hour: foundation.Some(24)}},
		{"hour negative", TimeOfDayOpts{Hour: foundation.Some(-1)}},
		{"minute too big", TimeOfDayOpts{Minute: foundation.Some(60)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recoverError(func() { TimeOfDay{1, 1}.Replacing(tt.o) })
			wantRoute(t, err, "Replacing.TimeOfDayInvalid")
		})
	}
}

func TestHourFormatOf(t *testing.T) {
	tests := []struct {
		in   TimeOfDayFormat
		want HourFormat
	}{
		{TimeOfDayFormatHHColonMM, HourFormatHH}, {TimeOfDayFormatHHDotMM, HourFormatHH}, {TimeOfDayFormatFrenchCanadian, HourFormatHH},
		{TimeOfDayFormatHColonMM, HourFormatH},
		{TimeOfDayFormatHColonMMSpaceA, HourFormatLowerH}, {TimeOfDayFormatASpaceHColonMM, HourFormatLowerH},
	}
	for _, tt := range tests {
		if got := HourFormatOf(tt.in); got != tt.want {
			t.Errorf("HourFormatOf(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
