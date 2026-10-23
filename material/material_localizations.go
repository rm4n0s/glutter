package material

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// This file was made from material/material_localizations.dart: the interface
// by a script, the English strings of DefaultMaterialLocalizations too.

// MaterialLocalizations defines the localized resource values used by the
// widgets of the Material library: the labels, the tooltips, the semantics
// labels, and the formats of dates and times. The values come from the
// Localizations widget that MaterialApp makes; DefaultMaterialLocalizations is
// the English implementation, and it is what custom implementations embed to
// get the strings they do not translate.
//
// Deviation: Dart's optional named parameter alwaysUse24HourFormat is a
// positional bool, and a method that returns a nullable DateTime returns
// (time.Time, bool).
//
// Flutter: material/material_localizations.dart MaterialLocalizations
type MaterialLocalizations interface {
	// OpenAppDrawerTooltip: the tooltip for the leading AppBar menu (a.k.a. 'hamburger') button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.openAppDrawerTooltip
	OpenAppDrawerTooltip() string
	// BackButtonTooltip: the BackButton's tooltip.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.backButtonTooltip
	BackButtonTooltip() string
	// ClearButtonTooltip: the tooltip for the clear button to clear text on SearchAnchor's search view.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.clearButtonTooltip
	ClearButtonTooltip() string
	// CloseButtonTooltip: the CloseButton's tooltip.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.closeButtonTooltip
	CloseButtonTooltip() string
	// DeleteButtonTooltip: the tooltip for the delete button on a Chip.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.deleteButtonTooltip
	DeleteButtonTooltip() string
	// MoreButtonTooltip: the tooltip for the more button on an overflowing text selection menu.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.moreButtonTooltip
	MoreButtonTooltip() string
	// NextMonthTooltip: the tooltip for the CalendarDatePicker's "next month" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.nextMonthTooltip
	NextMonthTooltip() string
	// PreviousMonthTooltip: the tooltip for the CalendarDatePicker's "previous month" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.previousMonthTooltip
	PreviousMonthTooltip() string
	// FirstPageTooltip: the tooltip for the PaginatedDataTable's "first page" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.firstPageTooltip
	FirstPageTooltip() string
	// LastPageTooltip: the tooltip for the PaginatedDataTable's "last page" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.lastPageTooltip
	LastPageTooltip() string
	// NextPageTooltip: the tooltip for the PaginatedDataTable's "next page" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.nextPageTooltip
	NextPageTooltip() string
	// PreviousPageTooltip: the tooltip for the PaginatedDataTable's "previous page" button.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.previousPageTooltip
	PreviousPageTooltip() string
	// ShowMenuTooltip: the default PopupMenuButton tooltip.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.showMenuTooltip
	ShowMenuTooltip() string
	// AboutListTileTitle: the default title for AboutListTile.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.aboutListTileTitle
	AboutListTileTitle(applicationName string) string
	// LicensesPageTitle: title for the LicensePage widget.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.licensesPageTitle
	LicensesPageTitle() string
	// LicensesPackageDetailText: subtitle for a package in the LicensePage widget.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.licensesPackageDetailText
	LicensesPackageDetailText(licenseCount int) string
	// PageRowsInfoTitle: title for the PaginatedDataTable's row info footer.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.pageRowsInfoTitle
	PageRowsInfoTitle(firstRow int, lastRow int, rowCount int, rowCountIsApproximate bool) string
	// RowsPerPageTitle: title for the PaginatedDataTable's "rows per page" footer.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.rowsPerPageTitle
	RowsPerPageTitle() string
	// TabLabel: the accessibility label used on a tab in a TabBar. This message describes the index of the selected tab and how many tabs there are, e.g.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.tabLabel
	TabLabel(tabIndex int, tabCount int) string
	// SelectedRowCountTitle: title for the PaginatedDataTable's selected row count header.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.selectedRowCountTitle
	SelectedRowCountTitle(selectedRowCount int) string
	// CancelButtonLabel: label for "cancel" buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.cancelButtonLabel
	CancelButtonLabel() string
	// CloseButtonLabel: label for "close" buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.closeButtonLabel
	CloseButtonLabel() string
	// ContinueButtonLabel: label for "continue" buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.continueButtonLabel
	ContinueButtonLabel() string
	// CopyButtonLabel: label for "copy" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.copyButtonLabel
	CopyButtonLabel() string
	// CutButtonLabel: label for "cut" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.cutButtonLabel
	CutButtonLabel() string
	// ScanTextButtonLabel: label for "scan text" OCR edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.scanTextButtonLabel
	ScanTextButtonLabel() string
	// OkButtonLabel: label for OK buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.okButtonLabel
	OkButtonLabel() string
	// PasteButtonLabel: label for "paste" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.pasteButtonLabel
	PasteButtonLabel() string
	// SelectAllButtonLabel: label for "select all" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.selectAllButtonLabel
	SelectAllButtonLabel() string
	// LookUpButtonLabel: label for "look up" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.lookUpButtonLabel
	LookUpButtonLabel() string
	// SearchWebButtonLabel: label for "search web" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.searchWebButtonLabel
	SearchWebButtonLabel() string
	// ShareButtonLabel: label for "share" edit buttons and menu items.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.shareButtonLabel
	ShareButtonLabel() string
	// ViewLicensesButtonLabel: label for the AboutDialog button that shows the LicensePage.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.viewLicensesButtonLabel
	ViewLicensesButtonLabel() string
	// AnteMeridiemAbbreviation: the abbreviation for ante meridiem (before noon) shown in the time picker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.anteMeridiemAbbreviation
	AnteMeridiemAbbreviation() string
	// PostMeridiemAbbreviation: the abbreviation for post meridiem (after noon) shown in the time picker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.postMeridiemAbbreviation
	PostMeridiemAbbreviation() string
	// TimePickerHourModeAnnouncement: the text-to-speech announcement made when a time picker invoked using showTimePicker is set to the hour picker mode.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerHourModeAnnouncement
	TimePickerHourModeAnnouncement() string
	// TimePickerMinuteModeAnnouncement: the text-to-speech announcement made when a time picker invoked using showTimePicker is set to the minute picker mode.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerMinuteModeAnnouncement
	TimePickerMinuteModeAnnouncement() string
	// ModalBarrierDismissLabel: label read out by accessibility tools (TalkBack or VoiceOver) for a modal barrier to indicate that a tap dismisses the barrier. A modal barrier can for example be found behind an alert or popup to block user interaction with elements behind it.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.modalBarrierDismissLabel
	ModalBarrierDismissLabel() string
	// MenuDismissLabel: label read out by accessibility tools (TalkBack or VoiceOver) for a context menu to indicate that a tap dismisses the context menu.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.menuDismissLabel
	MenuDismissLabel() string
	// DrawerLabel: label read out by accessibility tools (TalkBack or VoiceOver) when a drawer widget is opened.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.drawerLabel
	DrawerLabel() string
	// PopupMenuLabel: label read out by accessibility tools (TalkBack or VoiceOver) when a popup menu widget is opened.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.popupMenuLabel
	PopupMenuLabel() string
	// MenuBarMenuLabel: label read out by accessibility tools (TalkBack or VoiceOver) when a MenuBarMenu widget is opened.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.menuBarMenuLabel
	MenuBarMenuLabel() string
	// DialogLabel: label read out by accessibility tools (TalkBack or VoiceOver) when a dialog widget is opened.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dialogLabel
	DialogLabel() string
	// AlertDialogLabel: label read out by accessibility tools (TalkBack or VoiceOver) when an alert dialog widget is opened.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.alertDialogLabel
	AlertDialogLabel() string
	// SearchFieldLabel: label indicating that a text field is a search field. This will be used as a hint text in the text field.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.searchFieldLabel
	SearchFieldLabel() string
	// CurrentDateLabel: label indicating that a given date is the current date.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.currentDateLabel
	CurrentDateLabel() string
	// SelectedDateLabel: the semantics label to describe the selected date in the calendar picker invoked using showDatePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.selectedDateLabel
	SelectedDateLabel() string
	// ScrimLabel: label for the scrim rendered underneath a BottomSheet.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.scrimLabel
	ScrimLabel() string
	// BottomSheetLabel: label for a BottomSheet, used as the modalRouteContentName of the scrimOnTapHint.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.bottomSheetLabel
	BottomSheetLabel() string
	// ScrimOnTapHint: hint text announced when tapping on the scrim underneath the content of a modal route.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.scrimOnTapHint
	ScrimOnTapHint(modalRouteContentName string) string
	// TimeOfDayFormat: the format used to lay out the time picker. The documentation for TimeOfDayFormat enum values provides details on each supported layout.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timeOfDayFormat
	TimeOfDayFormat(alwaysUse24HourFormat bool) TimeOfDayFormat
	// ScriptCategory: defines the localized TextStyle geometry for ThemeData.textTheme. The scriptCategory defines the overall geometry of a TextTheme for the Typography.geometryThemeFor method in terms of the three language categories defined in https://material.io/go/design-typography.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.scriptCategory
	ScriptCategory() ScriptCategory
	// FormatDecimal: formats number as a decimal, inserting locale-appropriate thousands separators as necessary.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatDecimal
	FormatDecimal(number int) string
	// FormatHour: formats TimeOfDay.hour in the given time of day according to the value of timeOfDayFormat. If alwaysUse24HourFormat is true, formats hour using HourFormat.HH rather than the default for the current locale.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatHour
	FormatHour(timeOfDay TimeOfDay, alwaysUse24HourFormat bool) string
	// FormatMinute: formats TimeOfDay.minute in the given time of day according to the value of timeOfDayFormat.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatMinute
	FormatMinute(timeOfDay TimeOfDay) string
	// FormatTimeOfDay: formats timeOfDay according to the value of timeOfDayFormat. If alwaysUse24HourFormat is true, formats hour using HourFormat.HH rather than the default for the current locale.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatTimeOfDay
	FormatTimeOfDay(timeOfDay TimeOfDay, alwaysUse24HourFormat bool) string
	// FormatYear: full unabbreviated year format, e.g. 2017 rather than 17.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatYear
	FormatYear(date time.Time) string
	// FormatCompactDate: formats the date in a compact format. Usually just the numeric values for the for day, month and year are used.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatCompactDate
	FormatCompactDate(date time.Time) string
	// FormatShortDate: formats the date using a short-width format. Includes the abbreviation of the month, the day and year.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatShortDate
	FormatShortDate(date time.Time) string
	// FormatMediumDate: formats the date using a medium-width format. Abbreviates month and days of week.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatMediumDate
	FormatMediumDate(date time.Time) string
	// FormatFullDate: formats day of week, month, day of month and year in a long-width format. Does not abbreviate names.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatFullDate
	FormatFullDate(date time.Time) string
	// FormatMonthYear: formats the month and the year of the given date. The returned string does not contain the day of the month.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatMonthYear
	FormatMonthYear(date time.Time) string
	// FormatShortMonthDay: formats the month and day of the given date. Examples:  - US English: Feb 21 - Russian: 21 февр.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.formatShortMonthDay
	FormatShortMonthDay(date time.Time) string
	// ParseCompactDate: converts the given compact date formatted string into a DateTime. The format of the string must be a valid compact date format for the given locale.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.parseCompactDate
	ParseCompactDate(inputString string) (time.Time, bool)
	// NarrowWeekdays: list of week day names in narrow format, usually 1- or 2-letter abbreviations of full names. The list begins with the value corresponding to Sunday and ends with Saturday.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.narrowWeekdays
	NarrowWeekdays() []string
	// FirstDayOfWeekIndex: index of the first day of week, where 0 points to Sunday, and 6 points to Saturday. This getter is compatible with narrowWeekdays.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.firstDayOfWeekIndex
	FirstDayOfWeekIndex() int
	// DateSeparator: the character string used to separate the parts of a compact date format (i.e. mm/dd/yyyy has a separator of '/').
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateSeparator
	DateSeparator() string
	// DateHelpText: the help text used on an empty InputDatePickerFormField to indicate to the user the date format being asked for.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateHelpText
	DateHelpText() string
	// SelectYearSemanticsLabel: the semantic label used to announce when the user has entered the year selection mode of the CalendarDatePicker which is used in the data picker dialog created with showDatePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.selectYearSemanticsLabel
	SelectYearSemanticsLabel() string
	// UnspecifiedDate: the label used to indicate a date that has not been entered or selected yet in the date picker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.unspecifiedDate
	UnspecifiedDate() string
	// UnspecifiedDateRange: the label used to indicate a date range that has not been entered or selected yet in the date range picker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.unspecifiedDateRange
	UnspecifiedDateRange() string
	// DateInputLabel: the label used to describe the text field used in an InputDatePickerFormField.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateInputLabel
	DateInputLabel() string
	// DateRangeStartLabel: the label used for the starting date input field in the date range picker created with showDateRangePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateRangeStartLabel
	DateRangeStartLabel() string
	// DateRangeEndLabel: the label used for the ending date input field in the date range picker created with showDateRangePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateRangeEndLabel
	DateRangeEndLabel() string
	// DateRangeStartDateSemanticLabel: the semantics label used for the selected start date in the date range picker's day grid.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateRangeStartDateSemanticLabel
	DateRangeStartDateSemanticLabel(formattedDate string) string
	// DateRangeEndDateSemanticLabel: the semantics label used for the selected end date in the date range picker's day grid.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateRangeEndDateSemanticLabel
	DateRangeEndDateSemanticLabel(formattedDate string) string
	// InvalidDateFormatLabel: error message displayed to the user when they have entered a text string in an InputDatePickerFormField that is not in a valid date format.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.invalidDateFormatLabel
	InvalidDateFormatLabel() string
	// InvalidDateRangeLabel: error message displayed to the user when they have entered an invalid date range in the input mode of the date range picker created with showDateRangePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.invalidDateRangeLabel
	InvalidDateRangeLabel() string
	// DateOutOfRangeLabel: error message displayed to the user when they have entered a date that is outside the valid range for the date picker. showDateRangePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateOutOfRangeLabel
	DateOutOfRangeLabel() string
	// SaveButtonLabel: label for a 'SAVE' button. Currently used by the full screen mode of the date range picker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.saveButtonLabel
	SaveButtonLabel() string
	// DatePickerHelpText: label used in the header of the date picker dialog created with showDatePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.datePickerHelpText
	DatePickerHelpText() string
	// DateRangePickerHelpText: label used in the header of the date range picker dialog created with showDateRangePicker.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dateRangePickerHelpText
	DateRangePickerHelpText() string
	// CalendarModeButtonLabel: tooltip used for the calendar mode button of the date pickers.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.calendarModeButtonLabel
	CalendarModeButtonLabel() string
	// InputDateModeButtonLabel: tooltip used for the text input mode button of the date pickers.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.inputDateModeButtonLabel
	InputDateModeButtonLabel() string
	// TimePickerDialHelpText: label used in the header of the time picker dialog created with showTimePicker when in TimePickerEntryMode.dial.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerDialHelpText
	TimePickerDialHelpText() string
	// TimePickerInputHelpText: label used in the header of the time picker dialog created with showTimePicker when in TimePickerEntryMode.input.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerInputHelpText
	TimePickerInputHelpText() string
	// TimePickerHourLabel: label used below the hour text field of the time picker dialog created with showTimePicker when in TimePickerEntryMode.input.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerHourLabel
	TimePickerHourLabel() string
	// TimePickerMinuteLabel: label used below the minute text field of the time picker dialog created with showTimePicker when in TimePickerEntryMode.input.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.timePickerMinuteLabel
	TimePickerMinuteLabel() string
	// InvalidTimeLabel: error message for the time picker dialog created with showTimePicker when in TimePickerEntryMode.input.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.invalidTimeLabel
	InvalidTimeLabel() string
	// DialModeButtonLabel: tooltip used to put the time picker into TimePickerEntryMode.dial.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.dialModeButtonLabel
	DialModeButtonLabel() string
	// InputTimeModeButtonLabel: tooltip used to put the time picker into TimePickerEntryMode.input.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.inputTimeModeButtonLabel
	InputTimeModeButtonLabel() string
	// SignedInLabel: the semantics label used to indicate which account is signed in the UserAccountsDrawerHeader widget.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.signedInLabel
	SignedInLabel() string
	// HideAccountsLabel: the semantics label used for the button on UserAccountsDrawerHeader that hides the list of accounts.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.hideAccountsLabel
	HideAccountsLabel() string
	// ShowAccountsLabel: the semantics label used for the button on UserAccountsDrawerHeader that shows the list of accounts.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.showAccountsLabel
	ShowAccountsLabel() string
	// ReorderItemToStart: the semantics label used for ReorderableListView to reorder an item in the list to the start of the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemToStart
	ReorderItemToStart() string
	// ReorderItemToEnd: the semantics label used for ReorderableListView to reorder an item in the list to the end of the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemToEnd
	ReorderItemToEnd() string
	// ReorderItemUp: the semantics label used for ReorderableListView to reorder an item in the list one space up the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemUp
	ReorderItemUp() string
	// ReorderItemDown: the semantics label used for ReorderableListView to reorder an item in the list one space down the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemDown
	ReorderItemDown() string
	// ReorderItemLeft: the semantics label used for ReorderableListView to reorder an item in the list one space left in the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemLeft
	ReorderItemLeft() string
	// ReorderItemRight: the semantics label used for ReorderableListView to reorder an item in the list one space right in the list.
	//
	// Deprecated in Flutter: use the one of widgets.WidgetsLocalizations.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.reorderItemRight
	ReorderItemRight() string
	// ExpandedIconTapHint: the semantics hint to describe the tap action on an expanded ExpandIcon.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expandedIconTapHint
	ExpandedIconTapHint() string
	// CollapsedIconTapHint: the semantics hint to describe the tap action on a collapsed ExpandIcon.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.collapsedIconTapHint
	CollapsedIconTapHint() string
	// ExpansionTileExpandedHint: the semantics hint to describe the tap action on an expanded ExpansionTile on iOS and macOS. This is appended to the collapsedHint hint to provide a more detailed description of the action, e.g.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expansionTileExpandedHint
	ExpansionTileExpandedHint() string
	// ExpansionTileCollapsedHint: the semantics hint to describe the tap action on a collapsed ExpansionTile on iOS and macOS. This is appended to the expandedHint hint to provide a more detailed description of the action, e.g.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expansionTileCollapsedHint
	ExpansionTileCollapsedHint() string
	// ExpansionTileExpandedTapHint: the semantics hint to describe the tap action on an expanded ExpansionTile.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expansionTileExpandedTapHint
	ExpansionTileExpandedTapHint() string
	// ExpansionTileCollapsedTapHint: the semantics hint to describe the tap action on a collapsed ExpansionTile.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expansionTileCollapsedTapHint
	ExpansionTileCollapsedTapHint() string
	// ExpandedHint: the semantics hint to describe the ExpansionTile expanded state.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.expandedHint
	ExpandedHint() string
	// CollapsedHint: the semantics hint to describe the ExpansionTile collapsed state.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.collapsedHint
	CollapsedHint() string
	// RemainingTextFieldCharacterCount: the label for the TextField's character counter.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.remainingTextFieldCharacterCount
	RemainingTextFieldCharacterCount(remaining int) string
	// RefreshIndicatorSemanticLabel: the default semantics label for a RefreshIndicator.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.refreshIndicatorSemanticLabel
	RefreshIndicatorSemanticLabel() string
	// KeyboardKeyAlt: the shortcut label for the keyboard key LogicalKeyboardKey.alt.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyAlt
	KeyboardKeyAlt() string
	// KeyboardKeyAltGraph: the shortcut label for the keyboard key LogicalKeyboardKey.altGraph.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyAltGraph
	KeyboardKeyAltGraph() string
	// KeyboardKeyBackspace: the shortcut label for the keyboard key LogicalKeyboardKey.backspace.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyBackspace
	KeyboardKeyBackspace() string
	// KeyboardKeyCapsLock: the shortcut label for the keyboard key LogicalKeyboardKey.capsLock.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyCapsLock
	KeyboardKeyCapsLock() string
	// KeyboardKeyChannelDown: the shortcut label for the keyboard key LogicalKeyboardKey.channelDown.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyChannelDown
	KeyboardKeyChannelDown() string
	// KeyboardKeyChannelUp: the shortcut label for the keyboard key LogicalKeyboardKey.channelUp.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyChannelUp
	KeyboardKeyChannelUp() string
	// KeyboardKeyControl: the shortcut label for the keyboard key LogicalKeyboardKey.control.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyControl
	KeyboardKeyControl() string
	// KeyboardKeyDelete: the shortcut label for the keyboard key LogicalKeyboardKey.delete.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyDelete
	KeyboardKeyDelete() string
	// KeyboardKeyEject: the shortcut label for the keyboard key LogicalKeyboardKey.eject.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyEject
	KeyboardKeyEject() string
	// KeyboardKeyEnd: the shortcut label for the keyboard key LogicalKeyboardKey.end.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyEnd
	KeyboardKeyEnd() string
	// KeyboardKeyEscape: the shortcut label for the keyboard key LogicalKeyboardKey.escape.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyEscape
	KeyboardKeyEscape() string
	// KeyboardKeyFn: the shortcut label for the keyboard key LogicalKeyboardKey.fn.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyFn
	KeyboardKeyFn() string
	// KeyboardKeyHome: the shortcut label for the keyboard key LogicalKeyboardKey.home.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyHome
	KeyboardKeyHome() string
	// KeyboardKeyInsert: the shortcut label for the keyboard key LogicalKeyboardKey.insert.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyInsert
	KeyboardKeyInsert() string
	// KeyboardKeyMeta: the shortcut label for the keyboard key LogicalKeyboardKey.meta.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyMeta
	KeyboardKeyMeta() string
	// KeyboardKeyMetaMacOs: the shortcut label for the keyboard key LogicalKeyboardKey.meta on macOS.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyMetaMacOs
	KeyboardKeyMetaMacOs() string
	// KeyboardKeyMetaWindows: the shortcut label for the keyboard key LogicalKeyboardKey.meta on Windows.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyMetaWindows
	KeyboardKeyMetaWindows() string
	// KeyboardKeyNumLock: the shortcut label for the keyboard key LogicalKeyboardKey.numLock.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumLock
	KeyboardKeyNumLock() string
	// KeyboardKeyNumpad1: the shortcut label for the keyboard key LogicalKeyboardKey.numpad1.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad1
	KeyboardKeyNumpad1() string
	// KeyboardKeyNumpad2: the shortcut label for the keyboard key LogicalKeyboardKey.numpad2.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad2
	KeyboardKeyNumpad2() string
	// KeyboardKeyNumpad3: the shortcut label for the keyboard key LogicalKeyboardKey.numpad3.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad3
	KeyboardKeyNumpad3() string
	// KeyboardKeyNumpad4: the shortcut label for the keyboard key LogicalKeyboardKey.numpad4.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad4
	KeyboardKeyNumpad4() string
	// KeyboardKeyNumpad5: the shortcut label for the keyboard key LogicalKeyboardKey.numpad5.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad5
	KeyboardKeyNumpad5() string
	// KeyboardKeyNumpad6: the shortcut label for the keyboard key LogicalKeyboardKey.numpad6.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad6
	KeyboardKeyNumpad6() string
	// KeyboardKeyNumpad7: the shortcut label for the keyboard key LogicalKeyboardKey.numpad7.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad7
	KeyboardKeyNumpad7() string
	// KeyboardKeyNumpad8: the shortcut label for the keyboard key LogicalKeyboardKey.numpad8.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad8
	KeyboardKeyNumpad8() string
	// KeyboardKeyNumpad9: the shortcut label for the keyboard key LogicalKeyboardKey.numpad9.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad9
	KeyboardKeyNumpad9() string
	// KeyboardKeyNumpad0: the shortcut label for the keyboard key LogicalKeyboardKey.numpad0.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpad0
	KeyboardKeyNumpad0() string
	// KeyboardKeyNumpadAdd: the shortcut label for the keyboard key LogicalKeyboardKey.numpadAdd.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadAdd
	KeyboardKeyNumpadAdd() string
	// KeyboardKeyNumpadComma: the shortcut label for the keyboard key LogicalKeyboardKey.numpadComma.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadComma
	KeyboardKeyNumpadComma() string
	// KeyboardKeyNumpadDecimal: the shortcut label for the keyboard key LogicalKeyboardKey.numpadDecimal.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadDecimal
	KeyboardKeyNumpadDecimal() string
	// KeyboardKeyNumpadDivide: the shortcut label for the keyboard key LogicalKeyboardKey.numpadDivide.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadDivide
	KeyboardKeyNumpadDivide() string
	// KeyboardKeyNumpadEnter: the shortcut label for the keyboard key LogicalKeyboardKey.numpadEnter.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadEnter
	KeyboardKeyNumpadEnter() string
	// KeyboardKeyNumpadEqual: the shortcut label for the keyboard key LogicalKeyboardKey.numpadEqual.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadEqual
	KeyboardKeyNumpadEqual() string
	// KeyboardKeyNumpadMultiply: the shortcut label for the keyboard key LogicalKeyboardKey.numpadMultiply.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadMultiply
	KeyboardKeyNumpadMultiply() string
	// KeyboardKeyNumpadParenLeft: the shortcut label for the keyboard key LogicalKeyboardKey.numpadParenLeft.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadParenLeft
	KeyboardKeyNumpadParenLeft() string
	// KeyboardKeyNumpadParenRight: the shortcut label for the keyboard key LogicalKeyboardKey.numpadParenRight.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadParenRight
	KeyboardKeyNumpadParenRight() string
	// KeyboardKeyNumpadSubtract: the shortcut label for the keyboard key LogicalKeyboardKey.numpadSubtract.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyNumpadSubtract
	KeyboardKeyNumpadSubtract() string
	// KeyboardKeyPageDown: the shortcut label for the keyboard key LogicalKeyboardKey.pageDown.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyPageDown
	KeyboardKeyPageDown() string
	// KeyboardKeyPageUp: the shortcut label for the keyboard key LogicalKeyboardKey.pageUp.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyPageUp
	KeyboardKeyPageUp() string
	// KeyboardKeyPower: the shortcut label for the keyboard key LogicalKeyboardKey.power.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyPower
	KeyboardKeyPower() string
	// KeyboardKeyPowerOff: the shortcut label for the keyboard key LogicalKeyboardKey.powerOff.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyPowerOff
	KeyboardKeyPowerOff() string
	// KeyboardKeyPrintScreen: the shortcut label for the keyboard key LogicalKeyboardKey.printScreen.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyPrintScreen
	KeyboardKeyPrintScreen() string
	// KeyboardKeyScrollLock: the shortcut label for the keyboard key LogicalKeyboardKey.scrollLock.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyScrollLock
	KeyboardKeyScrollLock() string
	// KeyboardKeySelect: the shortcut label for the keyboard key LogicalKeyboardKey.select.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeySelect
	KeyboardKeySelect() string
	// KeyboardKeyShift: the shortcut label for the keyboard key LogicalKeyboardKey.shift.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeyShift
	KeyboardKeyShift() string
	// KeyboardKeySpace: the shortcut label for the keyboard key LogicalKeyboardKey.space.
	//
	// Flutter: material/material_localizations.dart MaterialLocalizations.keyboardKeySpace
	KeyboardKeySpace() string
}

// DefaultMaterialLocalizations is the American English implementation of
// MaterialLocalizations. It is the one that MaterialApp uses when the app does
// not give other delegates.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations
type DefaultMaterialLocalizations struct{}

// Ordered to match time.Monday=1, time.Sunday=7.
var (
	shortWeekdays  = [...]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	weekdays       = [...]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	narrowWeekdays = []string{"S", "M", "T", "W", "T", "F", "S"}
	shortMonths    = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	months         = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
)

func daysInMonth(year, month int) int {
	if month == int(time.February) {
		if (year%4 == 0 && year%100 != 0) || year%400 == 0 {
			return 29
		}
		return 28
	}
	return [...]int{31, -1, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[month-1]
}

func formatTwoDigitZeroPad(number int) string {
	if foundation.DebugMode && (number < 0 || number >= 100) {
		panic(errors.New(tagMaterialLocalizationsNumberRange, "a two digit number must be from 0 to 99", "number", number))
	}
	return fmt.Sprintf("%02d", number)
}

// FormatHour formats the hour in the format of the locale.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatHour
func (l DefaultMaterialLocalizations) FormatHour(timeOfDay TimeOfDay, alwaysUse24HourFormat bool) string {
	return l.formatHourInFormat(timeOfDay, l.TimeOfDayFormat(alwaysUse24HourFormat))
}

// formatHourInFormat formats the hour in the time format.
func (l DefaultMaterialLocalizations) formatHourInFormat(timeOfDay TimeOfDay, format TimeOfDayFormat) string {
	switch format {
	case TimeOfDayFormatHColonMMSpaceA:
		if timeOfDay.HourOfPeriod() == 0 {
			return l.FormatDecimal(12)
		}
		return l.FormatDecimal(timeOfDay.HourOfPeriod())
	case TimeOfDayFormatHHColonMM:
		return formatTwoDigitZeroPad(timeOfDay.Hour)
	}
	panic(errors.New(tagMaterialLocalizationsTimeFormat, "DefaultMaterialLocalizations does not support the time format", "format", int(format)))
}

// FormatMinute formats the minute as two digits.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatMinute
func (DefaultMaterialLocalizations) FormatMinute(timeOfDay TimeOfDay) string {
	if timeOfDay.Minute < 10 {
		return "0" + strconv.Itoa(timeOfDay.Minute)
	}
	return strconv.Itoa(timeOfDay.Minute)
}

// FormatYear formats the year.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatYear
func (DefaultMaterialLocalizations) FormatYear(date time.Time) string {
	return strconv.Itoa(date.Year())
}

// FormatCompactDate formats the date as mm/dd/yyyy, the American format.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatCompactDate
func (DefaultMaterialLocalizations) FormatCompactDate(date time.Time) string {
	return fmt.Sprintf("%s/%s/%04d", formatTwoDigitZeroPad(int(date.Month())), formatTwoDigitZeroPad(date.Day()), date.Year())
}

// FormatShortDate formats the date as "Jan 2, 2024".
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatShortDate
func (DefaultMaterialLocalizations) FormatShortDate(date time.Time) string {
	return fmt.Sprintf("%s %d, %d", shortMonths[date.Month()-time.January], date.Day(), date.Year())
}

// FormatMediumDate formats the date as "Tue, Jan 2".
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatMediumDate
func (DefaultMaterialLocalizations) FormatMediumDate(date time.Time) string {
	return fmt.Sprintf("%s, %s %d", shortWeekdays[weekdayIndex(date)], shortMonths[date.Month()-time.January], date.Day())
}

// weekdayIndex is the weekday of the date with Monday as 0 (Dart's weekday - DateTime.monday).
func weekdayIndex(date time.Time) int { return (int(date.Weekday()) + 6) % 7 }

// FormatFullDate formats the date as "Tuesday, January 2, 2024".
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatFullDate
func (DefaultMaterialLocalizations) FormatFullDate(date time.Time) string {
	return fmt.Sprintf("%s, %s %d, %d", weekdays[weekdayIndex(date)], months[date.Month()-time.January], date.Day(), date.Year())
}

// FormatMonthYear formats the date as "January 2024".
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatMonthYear
func (l DefaultMaterialLocalizations) FormatMonthYear(date time.Time) string {
	return months[date.Month()-time.January] + " " + l.FormatYear(date)
}

// FormatShortMonthDay formats the date as "Jan 2".
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatShortMonthDay
func (DefaultMaterialLocalizations) FormatShortMonthDay(date time.Time) string {
	return fmt.Sprintf("%s %d", shortMonths[date.Month()-time.January], date.Day())
}

// ParseCompactDate parses a mm/dd/yyyy string; the result is false when the text is not a valid date.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.parseCompactDate
func (DefaultMaterialLocalizations) ParseCompactDate(inputString string) (time.Time, bool) {
	// Assumes US mm/dd/yyyy format.
	parts := strings.Split(inputString, "/")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	year, err := strconv.Atoi(parts[2])
	if err != nil || year < 1 {
		return time.Time{}, false
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, false
	}
	day, err := strconv.Atoi(parts[1])
	if err != nil || day < 1 || day > daysInMonth(year, month) {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), true
}

// NarrowWeekdays are the one-letter weekdays, starting with Sunday.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.narrowWeekdays
func (DefaultMaterialLocalizations) NarrowWeekdays() []string { return narrowWeekdays }

// FirstDayOfWeekIndex is 0: the first of NarrowWeekdays is Sunday.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.firstDayOfWeekIndex
func (DefaultMaterialLocalizations) FirstDayOfWeekIndex() int { return 0 }

// DateRangeStartDateSemanticLabel is the semantics label of the start date of a range.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateRangeStartDateSemanticLabel
func (DefaultMaterialLocalizations) DateRangeStartDateSemanticLabel(formattedDate string) string {
	return "Start date " + formattedDate
}

// DateRangeEndDateSemanticLabel is the semantics label of the end date of a range.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateRangeEndDateSemanticLabel
func (DefaultMaterialLocalizations) DateRangeEndDateSemanticLabel(formattedDate string) string {
	return "End date " + formattedDate
}

func (DefaultMaterialLocalizations) formatDayPeriod(timeOfDay TimeOfDay, l MaterialLocalizations) string {
	if timeOfDay.Period() == DayPeriodAM {
		return l.AnteMeridiemAbbreviation()
	}
	return l.PostMeridiemAbbreviation()
}

// FormatDecimal formats the number with a comma as the thousands separator.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatDecimal
func (DefaultMaterialLocalizations) FormatDecimal(number int) string {
	if number > -1000 && number < 1000 {
		return strconv.Itoa(number)
	}
	abs := number
	if abs < 0 {
		abs = -abs
	}
	digits := strconv.Itoa(abs)
	var result strings.Builder
	if number < 0 {
		result.WriteByte('-')
	}
	maxDigitIndex := len(digits) - 1
	for i := 0; i <= maxDigitIndex; i++ {
		result.WriteByte(digits[i])
		if i < maxDigitIndex && (maxDigitIndex-i)%3 == 0 {
			result.WriteByte(',')
		}
	}
	return result.String()
}

// FormatTimeOfDay formats the time as "1:04 PM", or "13:04" with the 24 hour format.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.formatTimeOfDay
func (l DefaultMaterialLocalizations) FormatTimeOfDay(timeOfDay TimeOfDay, alwaysUse24HourFormat bool) string {
	// Not using a date format for two reasons: it supports more formats than the
	// time picker does, and it operates on a time that is sensitive to time eras
	// and time zones, while here the hour and the minute are within one day no
	// matter what date the day falls on.
	s := l.FormatHour(timeOfDay, alwaysUse24HourFormat) + ":" + l.FormatMinute(timeOfDay)
	if alwaysUse24HourFormat {
		// There is no AM/PM indicator in 24-hour format.
		return s
	}
	return s + " " + l.formatDayPeriod(timeOfDay, l)
}

// AboutListTileTitle is the title of the about list tile.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.aboutListTileTitle
func (DefaultMaterialLocalizations) AboutListTileTitle(applicationName string) string {
	return "About " + applicationName
}

// ScrimOnTapHint is the hint of the tap action of a scrim.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.scrimOnTapHint
func (DefaultMaterialLocalizations) ScrimOnTapHint(modalRouteContentName string) string {
	return "Close " + modalRouteContentName
}

// LicensesPackageDetailText is the text that says how many licenses a package has.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.licensesPackageDetailText
func (DefaultMaterialLocalizations) LicensesPackageDetailText(licenseCount int) string {
	switch licenseCount {
	case 0:
		return "No licenses."
	case 1:
		return "1 license."
	}
	return fmt.Sprintf("%d licenses.", licenseCount)
}

// PageRowsInfoTitle is the text of the footer of a paginated data table.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.pageRowsInfoTitle
func (DefaultMaterialLocalizations) PageRowsInfoTitle(firstRow, lastRow, rowCount int, rowCountIsApproximate bool) string {
	if rowCountIsApproximate {
		return fmt.Sprintf("%d\u2013%d of about %d", firstRow, lastRow, rowCount)
	}
	return fmt.Sprintf("%d\u2013%d of %d", firstRow, lastRow, rowCount)
}

// TabLabel is the semantics label of a tab, such as "Tab 1 of 3"; both numbers must be at least 1.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.tabLabel
func (DefaultMaterialLocalizations) TabLabel(tabIndex, tabCount int) string {
	if foundation.DebugMode && (tabIndex < 1 || tabCount < 1) {
		panic(errors.New(tagMaterialLocalizationsTabLabel, "the index and the count of a tab label must be at least 1", "tabIndex", tabIndex, "tabCount", tabCount))
	}
	return fmt.Sprintf("Tab %d of %d", tabIndex, tabCount)
}

// SelectedRowCountTitle is the title of a data table with selected rows.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.selectedRowCountTitle
func (DefaultMaterialLocalizations) SelectedRowCountTitle(selectedRowCount int) string {
	switch selectedRowCount {
	case 0:
		return "No items selected"
	case 1:
		return "1 item selected"
	}
	return fmt.Sprintf("%d items selected", selectedRowCount)
}

// RemainingTextFieldCharacterCount is the semantics label of the number of characters left in a text field.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.remainingTextFieldCharacterCount
func (DefaultMaterialLocalizations) RemainingTextFieldCharacterCount(remaining int) string {
	switch remaining {
	case 0:
		return "No characters remaining"
	case 1:
		return "1 character remaining"
	}
	return fmt.Sprintf("%d characters remaining", remaining)
}

// ScriptCategory is English-like.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.scriptCategory
func (DefaultMaterialLocalizations) ScriptCategory() ScriptCategory { return ScriptCategoryEnglishLike }

// TimeOfDayFormat is "h:mm a", or "HH:mm" with the 24 hour format.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timeOfDayFormat
func (DefaultMaterialLocalizations) TimeOfDayFormat(alwaysUse24HourFormat bool) TimeOfDayFormat {
	if alwaysUse24HourFormat {
		return TimeOfDayFormatHHColonMM
	}
	return TimeOfDayFormatHColonMMSpaceA
}

// DefaultMaterialLocalizationsDelegate is the delegate that loads
// DefaultMaterialLocalizations for the English locales. It is a read-only value.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.delegate
var DefaultMaterialLocalizationsDelegate widgets.LocalizationsDelegate = &widgets.FuncLocalizationsDelegate[MaterialLocalizations]{
	Supported: func(locale ui.Locale) bool { return locale.LanguageCode == "en" },
	LoadFunc: func(context.Context, ui.Locale) (MaterialLocalizations, error) {
		return DefaultMaterialLocalizations{}, nil
	},
	Name: "DefaultMaterialLocalizations.delegate(en_US)",
}

// MaterialLocalizationsMaybeOf returns the MaterialLocalizations of the closest
// Localizations above the context, if it has them.
//
// Flutter: material/material_localizations.dart MaterialLocalizations.of
func MaterialLocalizationsMaybeOf(bc widgets.BuildContext) (MaterialLocalizations, bool) {
	return widgets.LocalizationsOf[MaterialLocalizations](bc)
}

// MaterialLocalizationsOf returns the MaterialLocalizations of the closest
// Localizations above the context. In debug builds it panics with the tag
// MaterialLocalizationsMissing when there are none.
//
// Flutter: material/material_localizations.dart MaterialLocalizations.of
func MaterialLocalizationsOf(bc widgets.BuildContext) MaterialLocalizations {
	DebugCheckHasMaterialLocalizations(bc)
	l, _ := MaterialLocalizationsMaybeOf(bc)
	return l
}

// OpenAppDrawerTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.openAppDrawerTooltip
func (DefaultMaterialLocalizations) OpenAppDrawerTooltip() string { return "Open navigation menu" }

// BackButtonTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.backButtonTooltip
func (DefaultMaterialLocalizations) BackButtonTooltip() string { return "Back" }

// ClearButtonTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.clearButtonTooltip
func (DefaultMaterialLocalizations) ClearButtonTooltip() string { return "Clear text" }

// CloseButtonTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.closeButtonTooltip
func (DefaultMaterialLocalizations) CloseButtonTooltip() string { return "Close" }

// DeleteButtonTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.deleteButtonTooltip
func (DefaultMaterialLocalizations) DeleteButtonTooltip() string { return "Delete" }

// MoreButtonTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.moreButtonTooltip
func (DefaultMaterialLocalizations) MoreButtonTooltip() string { return "More" }

// NextMonthTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.nextMonthTooltip
func (DefaultMaterialLocalizations) NextMonthTooltip() string { return "Next month" }

// PreviousMonthTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.previousMonthTooltip
func (DefaultMaterialLocalizations) PreviousMonthTooltip() string { return "Previous month" }

// FirstPageTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.firstPageTooltip
func (DefaultMaterialLocalizations) FirstPageTooltip() string { return "First page" }

// LastPageTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.lastPageTooltip
func (DefaultMaterialLocalizations) LastPageTooltip() string { return "Last page" }

// NextPageTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.nextPageTooltip
func (DefaultMaterialLocalizations) NextPageTooltip() string { return "Next page" }

// PreviousPageTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.previousPageTooltip
func (DefaultMaterialLocalizations) PreviousPageTooltip() string { return "Previous page" }

// ShowMenuTooltip is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.showMenuTooltip
func (DefaultMaterialLocalizations) ShowMenuTooltip() string { return "Show menu" }

// LicensesPageTitle is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.licensesPageTitle
func (DefaultMaterialLocalizations) LicensesPageTitle() string { return "Licenses" }

// RowsPerPageTitle is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.rowsPerPageTitle
func (DefaultMaterialLocalizations) RowsPerPageTitle() string { return "Rows per page:" }

// CancelButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.cancelButtonLabel
func (DefaultMaterialLocalizations) CancelButtonLabel() string { return "Cancel" }

// CloseButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.closeButtonLabel
func (DefaultMaterialLocalizations) CloseButtonLabel() string { return "Close" }

// ContinueButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.continueButtonLabel
func (DefaultMaterialLocalizations) ContinueButtonLabel() string { return "Continue" }

// CopyButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.copyButtonLabel
func (DefaultMaterialLocalizations) CopyButtonLabel() string { return "Copy" }

// CutButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.cutButtonLabel
func (DefaultMaterialLocalizations) CutButtonLabel() string { return "Cut" }

// ScanTextButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.scanTextButtonLabel
func (DefaultMaterialLocalizations) ScanTextButtonLabel() string { return "Scan text" }

// OkButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.okButtonLabel
func (DefaultMaterialLocalizations) OkButtonLabel() string { return "OK" }

// PasteButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.pasteButtonLabel
func (DefaultMaterialLocalizations) PasteButtonLabel() string { return "Paste" }

// SelectAllButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.selectAllButtonLabel
func (DefaultMaterialLocalizations) SelectAllButtonLabel() string { return "Select all" }

// LookUpButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.lookUpButtonLabel
func (DefaultMaterialLocalizations) LookUpButtonLabel() string { return "Look Up" }

// SearchWebButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.searchWebButtonLabel
func (DefaultMaterialLocalizations) SearchWebButtonLabel() string { return "Search Web" }

// ShareButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.shareButtonLabel
func (DefaultMaterialLocalizations) ShareButtonLabel() string { return "Share" }

// ViewLicensesButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.viewLicensesButtonLabel
func (DefaultMaterialLocalizations) ViewLicensesButtonLabel() string { return "View licenses" }

// AnteMeridiemAbbreviation is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.anteMeridiemAbbreviation
func (DefaultMaterialLocalizations) AnteMeridiemAbbreviation() string { return "AM" }

// PostMeridiemAbbreviation is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.postMeridiemAbbreviation
func (DefaultMaterialLocalizations) PostMeridiemAbbreviation() string { return "PM" }

// TimePickerHourModeAnnouncement is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerHourModeAnnouncement
func (DefaultMaterialLocalizations) TimePickerHourModeAnnouncement() string { return "Select hours" }

// TimePickerMinuteModeAnnouncement is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerMinuteModeAnnouncement
func (DefaultMaterialLocalizations) TimePickerMinuteModeAnnouncement() string {
	return "Select minutes"
}

// ModalBarrierDismissLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.modalBarrierDismissLabel
func (DefaultMaterialLocalizations) ModalBarrierDismissLabel() string { return "Dismiss" }

// MenuDismissLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.menuDismissLabel
func (DefaultMaterialLocalizations) MenuDismissLabel() string { return "Dismiss menu" }

// DrawerLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.drawerLabel
func (DefaultMaterialLocalizations) DrawerLabel() string { return "Navigation menu" }

// PopupMenuLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.popupMenuLabel
func (DefaultMaterialLocalizations) PopupMenuLabel() string { return "Popup menu" }

// MenuBarMenuLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.menuBarMenuLabel
func (DefaultMaterialLocalizations) MenuBarMenuLabel() string { return "Menu bar menu" }

// DialogLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dialogLabel
func (DefaultMaterialLocalizations) DialogLabel() string { return "Dialog" }

// AlertDialogLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.alertDialogLabel
func (DefaultMaterialLocalizations) AlertDialogLabel() string { return "Alert" }

// SearchFieldLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.searchFieldLabel
func (DefaultMaterialLocalizations) SearchFieldLabel() string { return "Search" }

// CurrentDateLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.currentDateLabel
func (DefaultMaterialLocalizations) CurrentDateLabel() string { return "Today" }

// SelectedDateLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.selectedDateLabel
func (DefaultMaterialLocalizations) SelectedDateLabel() string { return "Selected" }

// ScrimLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.scrimLabel
func (DefaultMaterialLocalizations) ScrimLabel() string { return "Scrim" }

// BottomSheetLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.bottomSheetLabel
func (DefaultMaterialLocalizations) BottomSheetLabel() string { return "Bottom Sheet" }

// DateSeparator is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateSeparator
func (DefaultMaterialLocalizations) DateSeparator() string { return "/" }

// DateHelpText is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateHelpText
func (DefaultMaterialLocalizations) DateHelpText() string { return "mm/dd/yyyy" }

// SelectYearSemanticsLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.selectYearSemanticsLabel
func (DefaultMaterialLocalizations) SelectYearSemanticsLabel() string { return "Select year" }

// UnspecifiedDate is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.unspecifiedDate
func (DefaultMaterialLocalizations) UnspecifiedDate() string { return "Date" }

// UnspecifiedDateRange is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.unspecifiedDateRange
func (DefaultMaterialLocalizations) UnspecifiedDateRange() string { return "Date Range" }

// DateInputLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateInputLabel
func (DefaultMaterialLocalizations) DateInputLabel() string { return "Enter Date" }

// DateRangeStartLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateRangeStartLabel
func (DefaultMaterialLocalizations) DateRangeStartLabel() string { return "Start Date" }

// DateRangeEndLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateRangeEndLabel
func (DefaultMaterialLocalizations) DateRangeEndLabel() string { return "End Date" }

// InvalidDateFormatLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.invalidDateFormatLabel
func (DefaultMaterialLocalizations) InvalidDateFormatLabel() string { return "Invalid format." }

// InvalidDateRangeLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.invalidDateRangeLabel
func (DefaultMaterialLocalizations) InvalidDateRangeLabel() string { return "Invalid range." }

// DateOutOfRangeLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateOutOfRangeLabel
func (DefaultMaterialLocalizations) DateOutOfRangeLabel() string { return "Out of range." }

// SaveButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.saveButtonLabel
func (DefaultMaterialLocalizations) SaveButtonLabel() string { return "Save" }

// DatePickerHelpText is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.datePickerHelpText
func (DefaultMaterialLocalizations) DatePickerHelpText() string { return "Select date" }

// DateRangePickerHelpText is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dateRangePickerHelpText
func (DefaultMaterialLocalizations) DateRangePickerHelpText() string { return "Select range" }

// CalendarModeButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.calendarModeButtonLabel
func (DefaultMaterialLocalizations) CalendarModeButtonLabel() string { return "Switch to calendar" }

// InputDateModeButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.inputDateModeButtonLabel
func (DefaultMaterialLocalizations) InputDateModeButtonLabel() string { return "Switch to input" }

// TimePickerDialHelpText is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerDialHelpText
func (DefaultMaterialLocalizations) TimePickerDialHelpText() string { return "Select time" }

// TimePickerInputHelpText is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerInputHelpText
func (DefaultMaterialLocalizations) TimePickerInputHelpText() string { return "Enter time" }

// TimePickerHourLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerHourLabel
func (DefaultMaterialLocalizations) TimePickerHourLabel() string { return "Hour" }

// TimePickerMinuteLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.timePickerMinuteLabel
func (DefaultMaterialLocalizations) TimePickerMinuteLabel() string { return "Minute" }

// InvalidTimeLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.invalidTimeLabel
func (DefaultMaterialLocalizations) InvalidTimeLabel() string { return "Enter a valid time" }

// DialModeButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.dialModeButtonLabel
func (DefaultMaterialLocalizations) DialModeButtonLabel() string { return "Switch to dial picker mode" }

// InputTimeModeButtonLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.inputTimeModeButtonLabel
func (DefaultMaterialLocalizations) InputTimeModeButtonLabel() string {
	return "Switch to text input mode"
}

// SignedInLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.signedInLabel
func (DefaultMaterialLocalizations) SignedInLabel() string { return "Signed in" }

// HideAccountsLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.hideAccountsLabel
func (DefaultMaterialLocalizations) HideAccountsLabel() string { return "Hide accounts" }

// ShowAccountsLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.showAccountsLabel
func (DefaultMaterialLocalizations) ShowAccountsLabel() string { return "Show accounts" }

// ReorderItemToStart is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemToStart
func (DefaultMaterialLocalizations) ReorderItemToStart() string { return "Move to the start" }

// ReorderItemToEnd is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemToEnd
func (DefaultMaterialLocalizations) ReorderItemToEnd() string { return "Move to the end" }

// ReorderItemUp is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemUp
func (DefaultMaterialLocalizations) ReorderItemUp() string { return "Move up" }

// ReorderItemDown is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemDown
func (DefaultMaterialLocalizations) ReorderItemDown() string { return "Move down" }

// ReorderItemLeft is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemLeft
func (DefaultMaterialLocalizations) ReorderItemLeft() string { return "Move left" }

// ReorderItemRight is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.reorderItemRight
func (DefaultMaterialLocalizations) ReorderItemRight() string { return "Move right" }

// ExpandedIconTapHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expandedIconTapHint
func (DefaultMaterialLocalizations) ExpandedIconTapHint() string { return "Collapse" }

// CollapsedIconTapHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.collapsedIconTapHint
func (DefaultMaterialLocalizations) CollapsedIconTapHint() string { return "Expand" }

// ExpansionTileExpandedHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expansionTileExpandedHint
func (DefaultMaterialLocalizations) ExpansionTileExpandedHint() string {
	return "double tap to collapse"
}

// ExpansionTileCollapsedHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expansionTileCollapsedHint
func (DefaultMaterialLocalizations) ExpansionTileCollapsedHint() string {
	return "double tap to expand"
}

// ExpansionTileExpandedTapHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expansionTileExpandedTapHint
func (DefaultMaterialLocalizations) ExpansionTileExpandedTapHint() string { return "Collapse" }

// ExpansionTileCollapsedTapHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expansionTileCollapsedTapHint
func (DefaultMaterialLocalizations) ExpansionTileCollapsedTapHint() string {
	return "Expand for more details"
}

// ExpandedHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.expandedHint
func (DefaultMaterialLocalizations) ExpandedHint() string { return "Collapsed" }

// CollapsedHint is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.collapsedHint
func (DefaultMaterialLocalizations) CollapsedHint() string { return "Expanded" }

// RefreshIndicatorSemanticLabel is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.refreshIndicatorSemanticLabel
func (DefaultMaterialLocalizations) RefreshIndicatorSemanticLabel() string { return "Refresh" }

// KeyboardKeyAlt is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyAlt
func (DefaultMaterialLocalizations) KeyboardKeyAlt() string { return "Alt" }

// KeyboardKeyAltGraph is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyAltGraph
func (DefaultMaterialLocalizations) KeyboardKeyAltGraph() string { return "AltGr" }

// KeyboardKeyBackspace is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyBackspace
func (DefaultMaterialLocalizations) KeyboardKeyBackspace() string { return "Backspace" }

// KeyboardKeyCapsLock is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyCapsLock
func (DefaultMaterialLocalizations) KeyboardKeyCapsLock() string { return "Caps Lock" }

// KeyboardKeyChannelDown is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyChannelDown
func (DefaultMaterialLocalizations) KeyboardKeyChannelDown() string { return "Channel Down" }

// KeyboardKeyChannelUp is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyChannelUp
func (DefaultMaterialLocalizations) KeyboardKeyChannelUp() string { return "Channel Up" }

// KeyboardKeyControl is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyControl
func (DefaultMaterialLocalizations) KeyboardKeyControl() string { return "Ctrl" }

// KeyboardKeyDelete is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyDelete
func (DefaultMaterialLocalizations) KeyboardKeyDelete() string { return "Del" }

// KeyboardKeyEject is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyEject
func (DefaultMaterialLocalizations) KeyboardKeyEject() string { return "Eject" }

// KeyboardKeyEnd is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyEnd
func (DefaultMaterialLocalizations) KeyboardKeyEnd() string { return "End" }

// KeyboardKeyEscape is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyEscape
func (DefaultMaterialLocalizations) KeyboardKeyEscape() string { return "Esc" }

// KeyboardKeyFn is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyFn
func (DefaultMaterialLocalizations) KeyboardKeyFn() string { return "Fn" }

// KeyboardKeyHome is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyHome
func (DefaultMaterialLocalizations) KeyboardKeyHome() string { return "Home" }

// KeyboardKeyInsert is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyInsert
func (DefaultMaterialLocalizations) KeyboardKeyInsert() string { return "Insert" }

// KeyboardKeyMeta is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyMeta
func (DefaultMaterialLocalizations) KeyboardKeyMeta() string { return "Meta" }

// KeyboardKeyMetaMacOs is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyMetaMacOs
func (DefaultMaterialLocalizations) KeyboardKeyMetaMacOs() string { return "Command" }

// KeyboardKeyMetaWindows is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyMetaWindows
func (DefaultMaterialLocalizations) KeyboardKeyMetaWindows() string { return "Win" }

// KeyboardKeyNumLock is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumLock
func (DefaultMaterialLocalizations) KeyboardKeyNumLock() string { return "Num Lock" }

// KeyboardKeyNumpad1 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad1
func (DefaultMaterialLocalizations) KeyboardKeyNumpad1() string { return "Num 1" }

// KeyboardKeyNumpad2 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad2
func (DefaultMaterialLocalizations) KeyboardKeyNumpad2() string { return "Num 2" }

// KeyboardKeyNumpad3 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad3
func (DefaultMaterialLocalizations) KeyboardKeyNumpad3() string { return "Num 3" }

// KeyboardKeyNumpad4 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad4
func (DefaultMaterialLocalizations) KeyboardKeyNumpad4() string { return "Num 4" }

// KeyboardKeyNumpad5 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad5
func (DefaultMaterialLocalizations) KeyboardKeyNumpad5() string { return "Num 5" }

// KeyboardKeyNumpad6 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad6
func (DefaultMaterialLocalizations) KeyboardKeyNumpad6() string { return "Num 6" }

// KeyboardKeyNumpad7 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad7
func (DefaultMaterialLocalizations) KeyboardKeyNumpad7() string { return "Num 7" }

// KeyboardKeyNumpad8 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad8
func (DefaultMaterialLocalizations) KeyboardKeyNumpad8() string { return "Num 8" }

// KeyboardKeyNumpad9 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad9
func (DefaultMaterialLocalizations) KeyboardKeyNumpad9() string { return "Num 9" }

// KeyboardKeyNumpad0 is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpad0
func (DefaultMaterialLocalizations) KeyboardKeyNumpad0() string { return "Num 0" }

// KeyboardKeyNumpadAdd is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadAdd
func (DefaultMaterialLocalizations) KeyboardKeyNumpadAdd() string { return "Num +" }

// KeyboardKeyNumpadComma is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadComma
func (DefaultMaterialLocalizations) KeyboardKeyNumpadComma() string { return "Num ," }

// KeyboardKeyNumpadDecimal is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadDecimal
func (DefaultMaterialLocalizations) KeyboardKeyNumpadDecimal() string { return "Num ." }

// KeyboardKeyNumpadDivide is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadDivide
func (DefaultMaterialLocalizations) KeyboardKeyNumpadDivide() string { return "Num /" }

// KeyboardKeyNumpadEnter is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadEnter
func (DefaultMaterialLocalizations) KeyboardKeyNumpadEnter() string { return "Num Enter" }

// KeyboardKeyNumpadEqual is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadEqual
func (DefaultMaterialLocalizations) KeyboardKeyNumpadEqual() string { return "Num =" }

// KeyboardKeyNumpadMultiply is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadMultiply
func (DefaultMaterialLocalizations) KeyboardKeyNumpadMultiply() string { return "Num *" }

// KeyboardKeyNumpadParenLeft is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadParenLeft
func (DefaultMaterialLocalizations) KeyboardKeyNumpadParenLeft() string { return "Num (" }

// KeyboardKeyNumpadParenRight is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadParenRight
func (DefaultMaterialLocalizations) KeyboardKeyNumpadParenRight() string { return "Num )" }

// KeyboardKeyNumpadSubtract is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyNumpadSubtract
func (DefaultMaterialLocalizations) KeyboardKeyNumpadSubtract() string { return "Num -" }

// KeyboardKeyPageDown is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyPageDown
func (DefaultMaterialLocalizations) KeyboardKeyPageDown() string { return "PgDown" }

// KeyboardKeyPageUp is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyPageUp
func (DefaultMaterialLocalizations) KeyboardKeyPageUp() string { return "PgUp" }

// KeyboardKeyPower is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyPower
func (DefaultMaterialLocalizations) KeyboardKeyPower() string { return "Power" }

// KeyboardKeyPowerOff is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyPowerOff
func (DefaultMaterialLocalizations) KeyboardKeyPowerOff() string { return "Power Off" }

// KeyboardKeyPrintScreen is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyPrintScreen
func (DefaultMaterialLocalizations) KeyboardKeyPrintScreen() string { return "Print Screen" }

// KeyboardKeyScrollLock is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyScrollLock
func (DefaultMaterialLocalizations) KeyboardKeyScrollLock() string { return "Scroll Lock" }

// KeyboardKeySelect is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeySelect
func (DefaultMaterialLocalizations) KeyboardKeySelect() string { return "Select" }

// KeyboardKeyShift is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeyShift
func (DefaultMaterialLocalizations) KeyboardKeyShift() string { return "Shift" }

// KeyboardKeySpace is the English string.
//
// Flutter: material/material_localizations.dart DefaultMaterialLocalizations.keyboardKeySpace
func (DefaultMaterialLocalizations) KeyboardKeySpace() string { return "Space" }

var _ MaterialLocalizations = DefaultMaterialLocalizations{}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMaterialLocalizationsNumberRange = "MaterialLocalizationsNumberRange"
	tagMaterialLocalizationsTabLabel    = "MaterialLocalizationsTabLabel"
	tagMaterialLocalizationsTimeFormat  = "MaterialLocalizationsTimeFormat"
)
