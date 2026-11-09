package cupertino_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/cupertino/localizations_test.dart: the English
// translations exist for every property, and CupertinoLocalizations.of fails without them
// (the second test of Flutter uses a CupertinoApp and is in app_test.go, step 34.2).
func TestEnglishTranslationsExistForAllCupertinoLocalizationProperties(t *testing.T) {
	var l cupertino.CupertinoLocalizations = cupertino.DefaultCupertinoLocalizations{}
	strings := map[string]string{
		"datePickerYear": l.DatePickerYear(2018), "datePickerMonth": l.DatePickerMonth(1), "datePickerStandaloneMonth": l.DatePickerStandaloneMonth(1),
		"datePickerDayOfMonth": l.DatePickerDayOfMonth(1, foundation.None[int]()), "datePickerDayOfMonth with a weekday": l.DatePickerDayOfMonth(1, foundation.Some(1)),
		"datePickerHour": l.DatePickerHour(0), "datePickerMinute": l.DatePickerMinute(0), "datePickerMediumDate": l.DatePickerMediumDate(time.Now()),
		"anteMeridiemAbbreviation": l.AnteMeridiemAbbreviation(), "postMeridiemAbbreviation": l.PostMeridiemAbbreviation(),
		"todayLabel": l.TodayLabel(), "alertDialogLabel": l.AlertDialogLabel(), "tabSemanticsLabel": l.TabSemanticsLabel(1, 2),
		"timerPickerHour": l.TimerPickerHour(0), "timerPickerMinute": l.TimerPickerMinute(0), "timerPickerSecond": l.TimerPickerSecond(0),
		"modalBarrierDismissLabel": l.ModalBarrierDismissLabel(), "menuDismissLabel": l.MenuDismissLabel(),
		"searchTextFieldPlaceholderLabel": l.SearchTextFieldPlaceholderLabel(), "noSpellCheckReplacementsLabel": l.NoSpellCheckReplacementsLabel(),
		"clearButtonLabel": l.ClearButtonLabel(), "cancelButtonLabel": l.CancelButtonLabel(), "backButtonLabel": l.BackButtonLabel(),
		"cutButtonLabel": l.CutButtonLabel(), "copyButtonLabel": l.CopyButtonLabel(), "pasteButtonLabel": l.PasteButtonLabel(),
		"selectAllButtonLabel": l.SelectAllButtonLabel(), "lookUpButtonLabel": l.LookUpButtonLabel(), "searchWebButtonLabel": l.SearchWebButtonLabel(),
		"shareButtonLabel": l.ShareButtonLabel(), "expansionTileExpandedHint": l.ExpansionTileExpandedHint(),
		"expansionTileCollapsedHint": l.ExpansionTileCollapsedHint(), "expansionTileExpandedTapHint": l.ExpansionTileExpandedTapHint(),
		"expansionTileCollapsedTapHint": l.ExpansionTileCollapsedTapHint(), "expandedHint": l.ExpandedHint(), "collapsedHint": l.CollapsedHint(),
	}
	for name, got := range strings {
		if got == "" {
			t.Errorf("%s is empty", name)
		}
	}
	for name, f := range map[string]func() (string, bool){
		"datePickerHourSemanticsLabel":   func() (string, bool) { return l.DatePickerHourSemanticsLabel(0) },
		"datePickerMinuteSemanticsLabel": func() (string, bool) { return l.DatePickerMinuteSemanticsLabel(0) },
		"timerPickerHourLabel":           func() (string, bool) { return l.TimerPickerHourLabel(0) },
		"timerPickerMinuteLabel":         func() (string, bool) { return l.TimerPickerMinuteLabel(0) },
		"timerPickerSecondLabel":         func() (string, bool) { return l.TimerPickerSecondLabel(0) },
	} {
		if got, ok := f(); !ok || got == "" {
			t.Errorf("%s = %q, %v", name, got, ok)
		}
	}
	if l.DatePickerDateOrder() != cupertino.DatePickerDateOrderMDY || l.DatePickerDateTimeOrder() != cupertino.DatePickerDateTimeOrderDateTimeDayPeriod {
		t.Error("the date orders are not the ones of the United States")
	}
}

func TestDefaultCupertinoLocalizationsFormats(t *testing.T) {
	var l cupertino.CupertinoLocalizations = cupertino.DefaultCupertinoLocalizations{}
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"month", l.DatePickerMonth(3), "March"},
		{"day of the month with a weekday", l.DatePickerDayOfMonth(7, foundation.Some(2)), " Tue 7 "},
		{"minute is padded", l.DatePickerMinute(5), "05"},
		{"a medium date of a Monday pads the day", l.DatePickerMediumDate(time.Date(2024, time.June, 3, 0, 0, 0, 0, time.UTC)), "Mon Jun 3 "},
		{"a medium date of a Sunday", l.DatePickerMediumDate(time.Date(2024, time.June, 30, 0, 0, 0, 0, time.UTC)), "Sun Jun 30"},
		{"tab label", l.TabSemanticsLabel(2, 5), "Tab 2 of 5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
	if s, _ := l.DatePickerMinuteSemanticsLabel(1); s != "1 minute" {
		t.Errorf("one minute is %q", s)
	}
	if s, _ := l.DatePickerMinuteSemanticsLabel(2); s != "2 minutes" {
		t.Errorf("two minutes are %q", s)
	}
	if s, _ := l.TimerPickerHourLabel(1); s != "hour" {
		t.Errorf("one hour is %q", s)
	}
	if s, _ := l.TimerPickerHourLabel(2); s != "hours" {
		t.Errorf("two hours are %q", s)
	}
}

func TestCupertinoLocalizationsOfFailsWithoutLocalizations(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		cupertino.CupertinoLocalizationsOf(bc)
		return cupertino.NewSizedBox(nil)
	}))
	appErr := requireErrorTag(t, firstError(t, ts), "CupertinoLocalizationsMissing")
	if got := appErr.Message; len(got) < 30 || got[:len("No CupertinoLocalizations found")] != "No CupertinoLocalizations found" {
		t.Errorf("message = %q", got)
	}
}

func TestCupertinoLocalizationsOfFindsTheOnesOfADelegate(t *testing.T) {
	ts := gluttertest.New(t)
	var got cupertino.CupertinoLocalizations
	ts.PumpWidget(cupertino.NewLocalizations(ui.NewLocale("en", "US"),
		[]cupertino.LocalizationsDelegate{cupertino.DefaultWidgetsLocalizationsDelegate, cupertino.DefaultCupertinoLocalizationsDelegate},
		cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
			got, _ = cupertino.CupertinoLocalizationsMaybeOf(bc)
			return cupertino.NewSizedBox(nil)
		})))
	ts.Pump()
	if got == nil {
		t.Fatal("no CupertinoLocalizations below a Localizations with the default delegate")
	}
	if got.CancelButtonLabel() != "Cancel" {
		t.Errorf("cancel = %q", got.CancelButtonLabel())
	}
}

func TestTabSemanticsLabelAssertsItsArguments(t *testing.T) {
	skipUnlessDebug(t)
	defer func() {
		appErr := requireErrorTagFromPanic(t, recover(), "CupertinoTabSemanticsInvalid")
		if appErr.Metadata["tabIndex"] != 0 {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
	}()
	cupertino.DefaultCupertinoLocalizations{}.TabSemanticsLabel(0, 1)
}

func TestDateOrderNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{cupertino.DatePickerDateOrderDMY.Name(), "dmy"}, {cupertino.DatePickerDateOrderYDM.Name(), "ydm"},
		{cupertino.DatePickerDateTimeOrderDayPeriodTimeDate.Name(), "dayPeriod_time_date"},
		{cupertino.DatePickerDateTimeOrderUnspecified.Name(), "date_time_dayPeriod"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("name = %q, want %q", tt.got, tt.want)
		}
	}
}
