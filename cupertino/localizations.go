package cupertino

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DatePickerDateTimeOrder is the order of the date, the time and the day period (AM or
// PM) in a date and time picker.
//
// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder
type DatePickerDateTimeOrder int

const (
	// DatePickerDateTimeOrderUnspecified is the default order, date_time_dayPeriod.
	//
	// Flutter: no counterpart
	DatePickerDateTimeOrderUnspecified DatePickerDateTimeOrder = iota
	// DatePickerDateTimeOrderDateTimeDayPeriod is date, time, day period: "Mon Jun 6 | 7 : 24 | PM".
	//
	// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder.date_time_dayPeriod
	DatePickerDateTimeOrderDateTimeDayPeriod
	// DatePickerDateTimeOrderDateDayPeriodTime is date, day period, time.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder.date_dayPeriod_time
	DatePickerDateTimeOrderDateDayPeriodTime
	// DatePickerDateTimeOrderTimeDayPeriodDate is time, day period, date.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder.time_dayPeriod_date
	DatePickerDateTimeOrderTimeDayPeriodDate
	// DatePickerDateTimeOrderDayPeriodTimeDate is day period, time, date.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder.dayPeriod_time_date
	DatePickerDateTimeOrderDayPeriodTimeDate
)

var datePickerDateTimeOrderNames = [...]string{"date_time_dayPeriod", "date_time_dayPeriod", "date_dayPeriod_time", "time_dayPeriod_date", "dayPeriod_time_date"}

// Name is the name of the order in Dart.
//
// Flutter: cupertino/localizations.dart DatePickerDateTimeOrder
func (o DatePickerDateTimeOrder) Name() string {
	if o < 0 || int(o) >= len(datePickerDateTimeOrderNames) {
		return fmt.Sprintf("DatePickerDateTimeOrder(%d)", int(o))
	}
	return datePickerDateTimeOrderNames[o]
}

// DatePickerDateOrder is the order of the day, the month and the year in a date picker.
//
// Flutter: cupertino/localizations.dart DatePickerDateOrder
type DatePickerDateOrder int

const (
	// DatePickerDateOrderUnspecified is the default order, mdy.
	//
	// Flutter: no counterpart
	DatePickerDateOrderUnspecified DatePickerDateOrder = iota
	// DatePickerDateOrderDMY is day, month, year.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateOrder.dmy
	DatePickerDateOrderDMY
	// DatePickerDateOrderMDY is month, day, year: the order of the United States.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateOrder.mdy
	DatePickerDateOrderMDY
	// DatePickerDateOrderYMD is year, month, day.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateOrder.ymd
	DatePickerDateOrderYMD
	// DatePickerDateOrderYDM is year, day, month.
	//
	// Flutter: cupertino/localizations.dart DatePickerDateOrder.ydm
	DatePickerDateOrderYDM
)

var datePickerDateOrderNames = [...]string{"mdy", "dmy", "mdy", "ymd", "ydm"}

// Name is the name of the order in Dart.
//
// Flutter: cupertino/localizations.dart DatePickerDateOrder
func (o DatePickerDateOrder) Name() string {
	if o < 0 || int(o) >= len(datePickerDateOrderNames) {
		return fmt.Sprintf("DatePickerDateOrder(%d)", int(o))
	}
	return datePickerDateOrderNames[o]
}

// CupertinoLocalizations defines the localized resource values used by the widgets of
// the Cupertino library: the labels, the semantics labels and the formats of the
// pickers. The values come from the Localizations widget that CupertinoApp makes;
// DefaultCupertinoLocalizations is the English implementation, and it is what custom
// implementations embed to get the strings they do not translate.
//
// Deviation: a method that returns a nullable String returns (string, bool), the optional
// weekday of DatePickerDayOfMonth is a foundation.Opt[int], and DateTime is time.Time
// (SPEC §12.10).
//
// Flutter: cupertino/localizations.dart CupertinoLocalizations
type CupertinoLocalizations interface {
	// DatePickerYear is the term used to refer to the year in a date picker, for the index of the year.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerYear
	DatePickerYear(yearIndex int) string
	// DatePickerMonth is the month in a date picker, for a month from 1 to 12.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerMonth
	DatePickerMonth(monthIndex int) string
	// DatePickerStandaloneMonth is the month in a date picker when it stands alone, for a month from 1 to 12.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerStandaloneMonth
	DatePickerStandaloneMonth(monthIndex int) string
	// DatePickerDayOfMonth is the day of the month in a date picker, with the abbreviated weekday when one is given (1 is Monday).
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerDayOfMonth
	DatePickerDayOfMonth(dayIndex int, weekDay foundation.Opt[int]) string
	// DatePickerMediumDate is the date in the medium form of a date and time picker: "Mon Jun 6".
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerMediumDate
	DatePickerMediumDate(date time.Time) string
	// DatePickerHour is the hour in a date picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerHour
	DatePickerHour(hour int) string
	// DatePickerHourSemanticsLabel is the semantics label of the hour in a date picker; false for none.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerHourSemanticsLabel
	DatePickerHourSemanticsLabel(hour int) (string, bool)
	// DatePickerMinute is the minute in a date picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerMinute
	DatePickerMinute(minute int) string
	// DatePickerMinuteSemanticsLabel is the semantics label of the minute in a date picker; false for none.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerMinuteSemanticsLabel
	DatePickerMinuteSemanticsLabel(minute int) (string, bool)
	// DatePickerDateOrder is the order of the day, month and year in a date picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerDateOrder
	DatePickerDateOrder() DatePickerDateOrder
	// DatePickerDateTimeOrder is the order of the date, time and day period in a date and time picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.datePickerDateTimeOrder
	DatePickerDateTimeOrder() DatePickerDateTimeOrder
	// AnteMeridiemAbbreviation is the abbreviation for ante meridiem (before noon): AM.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.anteMeridiemAbbreviation
	AnteMeridiemAbbreviation() string
	// PostMeridiemAbbreviation is the abbreviation for post meridiem (after noon): PM.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.postMeridiemAbbreviation
	PostMeridiemAbbreviation() string
	// TodayLabel is the term for the current day in a date picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.todayLabel
	TodayLabel() string
	// AlertDialogLabel is the semantics label of an alert dialog.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.alertDialogLabel
	AlertDialogLabel() string
	// TabSemanticsLabel is the semantics label of a tab: both numbers start at 1.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.tabSemanticsLabel
	TabSemanticsLabel(tabIndex, tabCount int) string
	// TimerPickerHour is the hour in a timer picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerHour
	TimerPickerHour(hour int) string
	// TimerPickerMinute is the minute in a timer picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerMinute
	TimerPickerMinute(minute int) string
	// TimerPickerSecond is the second in a timer picker.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerSecond
	TimerPickerSecond(second int) string
	// TimerPickerHourLabel is the label next to the hour in a timer picker; false for none.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerHourLabel
	TimerPickerHourLabel(hour int) (string, bool)
	// TimerPickerHourLabels are all the labels that TimerPickerHourLabel can give.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerHourLabels
	TimerPickerHourLabels() []string
	// TimerPickerMinuteLabel is the label next to the minute in a timer picker; false for none.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerMinuteLabel
	TimerPickerMinuteLabel(minute int) (string, bool)
	// TimerPickerMinuteLabels are all the labels that TimerPickerMinuteLabel can give.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerMinuteLabels
	TimerPickerMinuteLabels() []string
	// TimerPickerSecondLabel is the label next to the second in a timer picker; false for none.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerSecondLabel
	TimerPickerSecondLabel(second int) (string, bool)
	// TimerPickerSecondLabels are all the labels that TimerPickerSecondLabel can give.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.timerPickerSecondLabels
	TimerPickerSecondLabels() []string
	// CutButtonLabel is the label of the cut button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.cutButtonLabel
	CutButtonLabel() string
	// CopyButtonLabel is the label of the copy button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.copyButtonLabel
	CopyButtonLabel() string
	// PasteButtonLabel is the label of the paste button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.pasteButtonLabel
	PasteButtonLabel() string
	// ClearButtonLabel is the semantics label of the clear button of a text field.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.clearButtonLabel
	ClearButtonLabel() string
	// NoSpellCheckReplacementsLabel is the label shown when the spell check toolbar has no replacements.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.noSpellCheckReplacementsLabel
	NoSpellCheckReplacementsLabel() string
	// SelectAllButtonLabel is the label of the select all button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.selectAllButtonLabel
	SelectAllButtonLabel() string
	// LookUpButtonLabel is the label of the look up button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.lookUpButtonLabel
	LookUpButtonLabel() string
	// SearchWebButtonLabel is the label of the search web button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.searchWebButtonLabel
	SearchWebButtonLabel() string
	// ShareButtonLabel is the label of the share button of the text selection toolbar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.shareButtonLabel
	ShareButtonLabel() string
	// SearchTextFieldPlaceholderLabel is the placeholder of a search field.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.searchTextFieldPlaceholderLabel
	SearchTextFieldPlaceholderLabel() string
	// ModalBarrierDismissLabel is the semantics label of the barrier of a modal route, which dismisses it.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.modalBarrierDismissLabel
	ModalBarrierDismissLabel() string
	// MenuDismissLabel is the semantics label of the barrier of a menu, which dismisses it.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.menuDismissLabel
	MenuDismissLabel() string
	// CancelButtonLabel is the label of the cancel button of an action sheet.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.cancelButtonLabel
	CancelButtonLabel() string
	// BackButtonLabel is the label of the back button of a navigation bar.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.backButtonLabel
	BackButtonLabel() string
	// ExpansionTileExpandedHint is the hint that the screen reader reads for an expanded tile.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.expansionTileExpandedHint
	ExpansionTileExpandedHint() string
	// ExpansionTileCollapsedHint is the hint that the screen reader reads for a collapsed tile.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.expansionTileCollapsedHint
	ExpansionTileCollapsedHint() string
	// ExpansionTileExpandedTapHint is the tap hint of an expanded tile.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.expansionTileExpandedTapHint
	ExpansionTileExpandedTapHint() string
	// ExpansionTileCollapsedTapHint is the tap hint of a collapsed tile.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.expansionTileCollapsedTapHint
	ExpansionTileCollapsedTapHint() string
	// ExpandedHint is the hint announced when a tile is collapsed. (Flutter's strings are swapped: the
	// expanded hint says "Collapsed" and the collapsed hint "Expanded"; they are kept.)
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.expandedHint
	ExpandedHint() string
	// CollapsedHint is the hint announced when a tile is expanded.
	//
	// Flutter: cupertino/localizations.dart CupertinoLocalizations.collapsedHint
	CollapsedHint() string
}

// DefaultCupertinoLocalizations is the English implementation of CupertinoLocalizations,
// used when the Localizations widget above has none for the locale. Custom
// implementations embed it for the strings they do not translate.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations
type DefaultCupertinoLocalizations struct{}

var (
	shortWeekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	shortMonths   = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	longMonths    = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
)

// DatePickerYear is the year as a number.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerYear
func (DefaultCupertinoLocalizations) DatePickerYear(yearIndex int) string {
	return strconv.Itoa(yearIndex)
}

// DatePickerMonth is the name of the month.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerMonth
func (DefaultCupertinoLocalizations) DatePickerMonth(monthIndex int) string {
	return longMonths[monthIndex-1]
}

// DatePickerStandaloneMonth is the name of the month.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerStandaloneMonth
func (DefaultCupertinoLocalizations) DatePickerStandaloneMonth(monthIndex int) string {
	return longMonths[monthIndex-1]
}

// DatePickerDayOfMonth is the day, with the short name of the weekday around it when there is one.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerDayOfMonth
func (DefaultCupertinoLocalizations) DatePickerDayOfMonth(dayIndex int, weekDay foundation.Opt[int]) string {
	if w, ok := weekDay.Get(); ok {
		return " " + shortWeekdays[w-1] + " " + strconv.Itoa(dayIndex) + " "
	}
	return strconv.Itoa(dayIndex)
}

// DatePickerHour is the hour as a number.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerHour
func (DefaultCupertinoLocalizations) DatePickerHour(hour int) string { return strconv.Itoa(hour) }

// DatePickerHourSemanticsLabel is "<hour> o'clock".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerHourSemanticsLabel
func (DefaultCupertinoLocalizations) DatePickerHourSemanticsLabel(hour int) (string, bool) {
	return strconv.Itoa(hour) + " o'clock", true
}

// DatePickerMinute is the minute as a number of two digits.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerMinute
func (DefaultCupertinoLocalizations) DatePickerMinute(minute int) string {
	return fmt.Sprintf("%02d", minute)
}

// DatePickerMinuteSemanticsLabel is "1 minute" or "<minute> minutes".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerMinuteSemanticsLabel
func (DefaultCupertinoLocalizations) DatePickerMinuteSemanticsLabel(minute int) (string, bool) {
	if minute == 1 {
		return "1 minute", true
	}
	return strconv.Itoa(minute) + " minutes", true
}

// DatePickerMediumDate is the short weekday, the short month and the day, padded to two characters.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerMediumDate
func (DefaultCupertinoLocalizations) DatePickerMediumDate(date time.Time) string {
	return shortWeekdays[(int(date.Weekday())+6)%7] + " " + shortMonths[date.Month()-time.January] + " " + fmt.Sprintf("%-2d", date.Day())
}

// DatePickerDateOrder is month, day, year.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerDateOrder
func (DefaultCupertinoLocalizations) DatePickerDateOrder() DatePickerDateOrder {
	return DatePickerDateOrderMDY
}

// DatePickerDateTimeOrder is date, time, day period.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.datePickerDateTimeOrder
func (DefaultCupertinoLocalizations) DatePickerDateTimeOrder() DatePickerDateTimeOrder {
	return DatePickerDateTimeOrderDateTimeDayPeriod
}

// AnteMeridiemAbbreviation is "AM".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.anteMeridiemAbbreviation
func (DefaultCupertinoLocalizations) AnteMeridiemAbbreviation() string { return "AM" }

// PostMeridiemAbbreviation is "PM".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.postMeridiemAbbreviation
func (DefaultCupertinoLocalizations) PostMeridiemAbbreviation() string { return "PM" }

// TodayLabel is "Today".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.todayLabel
func (DefaultCupertinoLocalizations) TodayLabel() string { return "Today" }

// AlertDialogLabel is "Alert".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.alertDialogLabel
func (DefaultCupertinoLocalizations) AlertDialogLabel() string { return "Alert" }

// TabSemanticsLabel is "Tab <index> of <count>". In debug builds it panics with the tag
// CupertinoTabSemanticsInvalid when the index or the count is less than 1.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.tabSemanticsLabel
func (DefaultCupertinoLocalizations) TabSemanticsLabel(tabIndex, tabCount int) string {
	if foundation.DebugMode && (tabIndex < 1 || tabCount < 1) {
		panic(errors.New(tagCupertinoTabSemanticsInvalid, "the tab index and the tab count of a tab semantics label start at 1",
			"tabIndex", tabIndex, "tabCount", tabCount))
	}
	return "Tab " + strconv.Itoa(tabIndex) + " of " + strconv.Itoa(tabCount)
}

// TimerPickerHour is the hour as a number.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerHour
func (DefaultCupertinoLocalizations) TimerPickerHour(hour int) string { return strconv.Itoa(hour) }

// TimerPickerMinute is the minute as a number.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerMinute
func (DefaultCupertinoLocalizations) TimerPickerMinute(minute int) string {
	return strconv.Itoa(minute)
}

// TimerPickerSecond is the second as a number.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerSecond
func (DefaultCupertinoLocalizations) TimerPickerSecond(second int) string {
	return strconv.Itoa(second)
}

// TimerPickerHourLabel is "hour" for 1 and "hours" for the rest.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerHourLabel
func (DefaultCupertinoLocalizations) TimerPickerHourLabel(hour int) (string, bool) {
	if hour == 1 {
		return "hour", true
	}
	return "hours", true
}

// TimerPickerHourLabels are "hour" and "hours".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerHourLabels
func (DefaultCupertinoLocalizations) TimerPickerHourLabels() []string {
	return []string{"hour", "hours"}
}

// TimerPickerMinuteLabel is "min.".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerMinuteLabel
func (DefaultCupertinoLocalizations) TimerPickerMinuteLabel(int) (string, bool) { return "min.", true }

// TimerPickerMinuteLabels is "min.".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerMinuteLabels
func (DefaultCupertinoLocalizations) TimerPickerMinuteLabels() []string { return []string{"min."} }

// TimerPickerSecondLabel is "sec.".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerSecondLabel
func (DefaultCupertinoLocalizations) TimerPickerSecondLabel(int) (string, bool) { return "sec.", true }

// TimerPickerSecondLabels is "sec.".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.timerPickerSecondLabels
func (DefaultCupertinoLocalizations) TimerPickerSecondLabels() []string { return []string{"sec."} }

// CutButtonLabel is "Cut".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.cutButtonLabel
func (DefaultCupertinoLocalizations) CutButtonLabel() string { return "Cut" }

// CopyButtonLabel is "Copy".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.copyButtonLabel
func (DefaultCupertinoLocalizations) CopyButtonLabel() string { return "Copy" }

// PasteButtonLabel is "Paste".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.pasteButtonLabel
func (DefaultCupertinoLocalizations) PasteButtonLabel() string { return "Paste" }

// ClearButtonLabel is "Clear".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.clearButtonLabel
func (DefaultCupertinoLocalizations) ClearButtonLabel() string { return "Clear" }

// NoSpellCheckReplacementsLabel is "No Replacements Found".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.noSpellCheckReplacementsLabel
func (DefaultCupertinoLocalizations) NoSpellCheckReplacementsLabel() string {
	return "No Replacements Found"
}

// SelectAllButtonLabel is "Select All".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.selectAllButtonLabel
func (DefaultCupertinoLocalizations) SelectAllButtonLabel() string { return "Select All" }

// LookUpButtonLabel is "Look Up".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.lookUpButtonLabel
func (DefaultCupertinoLocalizations) LookUpButtonLabel() string { return "Look Up" }

// SearchWebButtonLabel is "Search Web".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.searchWebButtonLabel
func (DefaultCupertinoLocalizations) SearchWebButtonLabel() string { return "Search Web" }

// ShareButtonLabel is "Share...".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.shareButtonLabel
func (DefaultCupertinoLocalizations) ShareButtonLabel() string { return "Share..." }

// SearchTextFieldPlaceholderLabel is "Search".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.searchTextFieldPlaceholderLabel
func (DefaultCupertinoLocalizations) SearchTextFieldPlaceholderLabel() string { return "Search" }

// ModalBarrierDismissLabel is "Dismiss".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.modalBarrierDismissLabel
func (DefaultCupertinoLocalizations) ModalBarrierDismissLabel() string { return "Dismiss" }

// MenuDismissLabel is "Dismiss menu".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.menuDismissLabel
func (DefaultCupertinoLocalizations) MenuDismissLabel() string { return "Dismiss menu" }

// CancelButtonLabel is "Cancel".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.cancelButtonLabel
func (DefaultCupertinoLocalizations) CancelButtonLabel() string { return "Cancel" }

// BackButtonLabel is "Back".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.backButtonLabel
func (DefaultCupertinoLocalizations) BackButtonLabel() string { return "Back" }

// ExpansionTileExpandedHint is "double tap to collapse".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.expansionTileExpandedHint
func (DefaultCupertinoLocalizations) ExpansionTileExpandedHint() string {
	return "double tap to collapse"
}

// ExpansionTileCollapsedHint is "double tap to expand".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.expansionTileCollapsedHint
func (DefaultCupertinoLocalizations) ExpansionTileCollapsedHint() string {
	return "double tap to expand"
}

// ExpansionTileExpandedTapHint is "Collapse".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.expansionTileExpandedTapHint
func (DefaultCupertinoLocalizations) ExpansionTileExpandedTapHint() string { return "Collapse" }

// ExpansionTileCollapsedTapHint is "Expand for more details".
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.expansionTileCollapsedTapHint
func (DefaultCupertinoLocalizations) ExpansionTileCollapsedTapHint() string {
	return "Expand for more details"
}

// ExpandedHint is "Collapsed" (as in Flutter, where the two hints are swapped).
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.expandedHint
func (DefaultCupertinoLocalizations) ExpandedHint() string { return "Collapsed" }

// CollapsedHint is "Expanded" (as in Flutter, where the two hints are swapped).
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.collapsedHint
func (DefaultCupertinoLocalizations) CollapsedHint() string { return "Expanded" }

// DefaultCupertinoLocalizationsDelegate is the delegate that loads
// DefaultCupertinoLocalizations for the English locales. It is a read-only value.
//
// Flutter: cupertino/localizations.dart DefaultCupertinoLocalizations.delegate
var DefaultCupertinoLocalizationsDelegate widgets.LocalizationsDelegate = &widgets.FuncLocalizationsDelegate[CupertinoLocalizations]{
	Supported: func(locale ui.Locale) bool { return locale.LanguageCode == "en" },
	LoadFunc: func(context.Context, ui.Locale) (CupertinoLocalizations, error) {
		return DefaultCupertinoLocalizations{}, nil
	},
	Name: "DefaultCupertinoLocalizations.delegate(en_US)",
}

// CupertinoLocalizationsMaybeOf returns the CupertinoLocalizations of the closest
// Localizations above the context, if it has them.
//
// Flutter: cupertino/localizations.dart CupertinoLocalizations.of
func CupertinoLocalizationsMaybeOf(bc widgets.BuildContext) (CupertinoLocalizations, bool) {
	return widgets.LocalizationsOf[CupertinoLocalizations](bc)
}

// CupertinoLocalizationsOf returns the CupertinoLocalizations of the closest Localizations
// above the context. In debug builds it panics with the tag CupertinoLocalizationsMissing
// when there are none.
//
// Flutter: cupertino/localizations.dart CupertinoLocalizations.of
func CupertinoLocalizationsOf(bc widgets.BuildContext) CupertinoLocalizations {
	DebugCheckHasCupertinoLocalizations(bc)
	l, _ := CupertinoLocalizationsMaybeOf(bc)
	return l
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTabSemanticsInvalid = "CupertinoTabSemanticsInvalid"
)
