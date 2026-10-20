// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	monthItemHeaderHeight     = 58.0
	monthItemFooterHeight     = 12.0
	monthItemRowHeight        = 42.0
	monthItemSpaceBetweenRows = 8.0
	rangeHorizontalPadding    = 8.0
	maxCalendarWidthLandscape = 384.0
	maxCalendarWidthPortrait  = 480.0
)

// SelectableDayForRangePredicate is the signature of a function that says whether
// a day can be selected in a date range picker, given the days that are
// selected now (the zero time for none).
//
// Flutter: material/date_picker.dart SelectableDayForRangePredicate
type SelectableDayForRangePredicate func(day, selectedStartDay, selectedEndDay time.Time) bool

// ShowDateRangePickerOpts are the optional parameters of ShowDateRangePicker.
//
// Flutter: material/date_picker.dart showDateRangePicker
type ShowDateRangePickerOpts struct {
	// InitialDateRange is the range that is selected first.
	InitialDateRange foundation.Opt[DateTimeRange]
	// CurrentDate is the date that is shown as today; the default is the current
	// date.
	CurrentDate time.Time
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// calendar.
	InitialEntryMode DatePickerEntryMode
	// HelpText is the text at the top of the dialog.
	HelpText string
	// CancelText is the label of the cancel button of the input mode.
	CancelText string
	// ConfirmText is the label of the confirm button of the input mode.
	ConfirmText string
	// SaveText is the label of the save button of the calendar mode.
	SaveText string
	// ErrorFormatText is the error text for an entry that is not a date.
	ErrorFormatText string
	// ErrorInvalidText is the error text for a date that is out of range or not
	// selectable.
	ErrorInvalidText string
	// ErrorInvalidRangeText is the error text for a range that ends before it
	// starts.
	ErrorInvalidRangeText string
	// FieldStartHintText is the hint of the start field of the input mode.
	FieldStartHintText string
	// FieldEndHintText is the hint of the end field of the input mode.
	FieldEndHintText string
	// FieldStartLabelText is the label of the start field of the input mode.
	FieldStartLabelText string
	// FieldEndLabelText is the label of the end field of the input mode.
	FieldEndLabelText string
	// Locale overrides the locale of the dialog.
	Locale foundation.Opt[ui.Locale]
	// BarrierDismissible is whether tapping the barrier cancels the dialog; the
	// default is true.
	BarrierDismissible foundation.Opt[bool]
	// BarrierColor is the color of the barrier.
	BarrierColor foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier.
	BarrierLabel string
	// UseRootNavigator pushes the dialog onto the root navigator; the default is
	// true.
	UseRootNavigator foundation.Opt[bool]
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// TextDirection overrides the text direction of the dialog.
	TextDirection ui.TextDirection
	// Builder wraps the dialog, for instance to change its theme.
	Builder func(bc widgets.BuildContext, dialog widgets.Widget) widgets.Widget
	// KeyboardType is the keyboard of the fields of the input mode; the default is
	// datetime.
	KeyboardType foundation.Opt[services.TextInputType]
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToCalendarEntryModeIcon is the icon of the button that switches to the
	// calendar mode.
	SwitchToCalendarEntryModeIcon *widgets.Icon
	// SelectableDayPredicate says which days can be selected.
	SelectableDayPredicate SelectableDayForRangePredicate
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// ShowDateRangePicker shows a full screen modal dialog that contains a Material
// Design date range picker. The returned channel gets the range that the user
// saved, or the zero DateTimeRange (and is then closed) when the dialog was
// cancelled.
//
// In debug builds it panics with the tags ShowDateRangePickerLastDateBeforeFirstDate,
// ShowDateRangePickerInitialRangeOutOfBounds and
// ShowDateRangePickerInitialRangeNotSelectable when the dates do not fit.
//
// Flutter: material/date_picker.dart showDateRangePicker
func ShowDateRangePicker(bc widgets.BuildContext, firstDate, lastDate time.Time, opts ...ShowDateRangePickerOpts) <-chan DateTimeRange {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	d := o.CalendarDelegate
	firstDate, lastDate = d.DateOnly(firstDate), d.DateOnly(lastDate)
	initial := foundation.None[DateTimeRange]()
	if r, ok := o.InitialDateRange.Get(); ok {
		initial = foundation.Some(NewDateTimeRange(d.DateOnly(r.Start), d.DateOnly(r.End)))
	}
	if foundation.DebugMode {
		if lastDate.Before(firstDate) {
			panic(errors.New("ShowDateRangePickerLastDateBeforeFirstDate", "lastDate must be on or after firstDate", "lastDate", lastDate, "firstDate", firstDate))
		}
		if r, ok := initial.Get(); ok {
			if r.Start.Before(firstDate) || r.End.Before(firstDate) || r.Start.After(lastDate) || r.End.After(lastDate) {
				panic(errors.New("ShowDateRangePickerInitialRangeOutOfBounds", "the initial date range must be between firstDate and lastDate", "initialDateRange", r, "firstDate", firstDate, "lastDate", lastDate))
			}
			if p := o.SelectableDayPredicate; p != nil && (!p(r.Start, r.Start, r.End) || !p(r.End, r.Start, r.End)) {
				panic(errors.New("ShowDateRangePickerInitialRangeNotSelectable", "the start and the end of the initial date range must be selectable", "initialDateRange", r))
			}
		}
	}
	DebugCheckHasMaterialLocalizations(bc)

	var dialog widgets.Widget = NewDateRangePickerDialog(firstDate, lastDate, DateRangePickerDialogOpts{
		InitialDateRange: initial, CurrentDate: d.DateOnly(firstNonZeroTime(o.CurrentDate, d.Now())), SelectableDayPredicate: o.SelectableDayPredicate,
		InitialEntryMode: o.InitialEntryMode, HelpText: o.HelpText, CancelText: o.CancelText, ConfirmText: o.ConfirmText, SaveText: o.SaveText,
		ErrorFormatText: o.ErrorFormatText, ErrorInvalidText: o.ErrorInvalidText, ErrorInvalidRangeText: o.ErrorInvalidRangeText,
		FieldStartHintText: o.FieldStartHintText, FieldEndHintText: o.FieldEndHintText,
		FieldStartLabelText: o.FieldStartLabelText, FieldEndLabelText: o.FieldEndLabelText, KeyboardType: o.KeyboardType,
		SwitchToInputEntryModeIcon: o.SwitchToInputEntryModeIcon, SwitchToCalendarEntryModeIcon: o.SwitchToCalendarEntryModeIcon,
		CalendarDelegate: d,
	})
	if o.TextDirection != ui.TextDirectionUnspecified {
		dialog = widgets.NewDirectionality(o.TextDirection, dialog)
	}
	if locale, ok := o.Locale.Get(); ok {
		dialog = widgets.NewLocalizationsOverride(bc, foundation.Some(locale), nil, dialog)
	}
	return ShowDialog[DateTimeRange](bc, func(bc widgets.BuildContext) widgets.Widget {
		if o.Builder == nil {
			return dialog
		}
		return o.Builder(bc, dialog)
	}, ShowDialogOpts{
		BarrierDismissible: o.BarrierDismissible, BarrierColor: o.BarrierColor, BarrierLabel: o.BarrierLabel,
		UseRootNavigator: o.UseRootNavigator, RouteSettings: o.RouteSettings, UseSafeArea: foundation.Some(false),
	})
}

// firstNonZeroTime returns the first time that is not the zero time.
func firstNonZeroTime(times ...time.Time) time.Time {
	for _, t := range times {
		if !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

// formatRangeStartDate formats the start of a range for the header.
//
// Flutter: material/date_picker.dart _formatRangeStartDate
func formatRangeStartDate(l MaterialLocalizations, d CalendarDelegate, start, end time.Time) string {
	switch {
	case start.IsZero():
		return l.DateRangeStartLabel()
	case end.IsZero() || start.Year() == end.Year():
		return d.FormatShortMonthDay(start, l)
	}
	return d.FormatShortDate(start, l)
}

// formatRangeEndDate formats the end of a range for the header.
//
// Flutter: material/date_picker.dart _formatRangeEndDate
func formatRangeEndDate(l MaterialLocalizations, d CalendarDelegate, start, end, currentDate time.Time) string {
	switch {
	case end.IsZero():
		return l.DateRangeEndLabel()
	case !start.IsZero() && start.Year() == end.Year() && start.Year() == currentDate.Year():
		return d.FormatShortMonthDay(end, l)
	}
	return d.FormatShortDate(end, l)
}

// DateRangePickerDialogOpts are the optional parameters of NewDateRangePickerDialog.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.new
type DateRangePickerDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialDateRange is the range that is selected first.
	InitialDateRange foundation.Opt[DateTimeRange]
	// CurrentDate is the date that is shown as today; the default is the current
	// date.
	CurrentDate time.Time
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// calendar.
	InitialEntryMode DatePickerEntryMode
	// HelpText is the text at the top of the dialog.
	HelpText string
	// CancelText is the label of the cancel button of the input mode.
	CancelText string
	// ConfirmText is the label of the confirm button of the input mode.
	ConfirmText string
	// SaveText is the label of the save button of the calendar mode.
	SaveText string
	// ErrorInvalidRangeText is the error text for a range that ends before it
	// starts.
	ErrorInvalidRangeText string
	// ErrorFormatText is the error text for an entry that is not a date.
	ErrorFormatText string
	// ErrorInvalidText is the error text for a date that is out of range or not
	// selectable.
	ErrorInvalidText string
	// FieldStartHintText is the hint of the start field of the input mode.
	FieldStartHintText string
	// FieldEndHintText is the hint of the end field of the input mode.
	FieldEndHintText string
	// FieldStartLabelText is the label of the start field of the input mode.
	FieldStartLabelText string
	// FieldEndLabelText is the label of the end field of the input mode.
	FieldEndLabelText string
	// KeyboardType is the keyboard of the fields of the input mode; the default is
	// datetime.
	KeyboardType foundation.Opt[services.TextInputType]
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToCalendarEntryModeIcon is the icon of the button that switches to the
	// calendar mode.
	SwitchToCalendarEntryModeIcon *widgets.Icon
	// SelectableDayPredicate says which days can be selected.
	SelectableDayPredicate SelectableDayForRangePredicate
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// DateRangePickerDialog is a full screen Material Design date range picker
// dialog. Use ShowDateRangePicker to show it.
//
// Deviation: the state restoration (restorationId) is not ported (SPEC §12.10).
//
// Flutter: material/date_picker.dart DateRangePickerDialog
type DateRangePickerDialog struct {
	widgets.StatefulWidget
	firstDate time.Time
	lastDate  time.Time
	o         DateRangePickerDialogOpts
}

// NewDateRangePickerDialog creates the dialog.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.new
func NewDateRangePickerDialog(firstDate, lastDate time.Time, opts ...DateRangePickerDialogOpts) *DateRangePickerDialog {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	return &DateRangePickerDialog{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), firstDate: firstDate, lastDate: lastDate, o: o}
}

// FirstDate is the earliest date the user can select.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.firstDate
func (w *DateRangePickerDialog) FirstDate() time.Time { return w.firstDate }

// LastDate is the latest date the user can select.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.lastDate
func (w *DateRangePickerDialog) LastDate() time.Time { return w.lastDate }

// CurrentDate is the date that is shown as today: the one given, or the current
// date, reduced to the date part.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.currentDate
func (w *DateRangePickerDialog) CurrentDate() time.Time {
	d := w.o.CalendarDelegate
	return d.DateOnly(firstNonZeroTime(w.o.CurrentDate, d.Now()))
}

// CreateState creates the state.
//
// Flutter: material/date_picker.dart DateRangePickerDialog.createState
func (w *DateRangePickerDialog) CreateState() widgets.State { return &dateRangePickerDialogState{} }

type dateRangePickerDialogState struct {
	widgets.StateOf[*DateRangePickerDialog]
	entryMode      DatePickerEntryMode
	selectedStart  time.Time
	selectedEnd    time.Time
	autoValidate   bool
	calendarKey    *widgets.GlobalKey[widgets.State]
	inputPickerKey *widgets.GlobalKey[*inputDateRangePickerState]
}

func (s *dateRangePickerDialogState) InitState() {
	w := s.Widget()
	s.entryMode = w.o.InitialEntryMode
	if s.entryMode == DatePickerEntryModeUnspecified {
		s.entryMode = DatePickerEntryModeCalendar
	}
	if r, ok := w.o.InitialDateRange.Get(); ok {
		s.selectedStart, s.selectedEnd = r.Start, r.End
	}
	s.calendarKey = widgets.NewGlobalKey[widgets.State]()
	s.inputPickerKey = widgets.NewGlobalKey[*inputDateRangePickerState]()
}

func (s *dateRangePickerDialogState) hasSelectedDateRange() bool {
	return !s.selectedStart.IsZero() && !s.selectedEnd.IsZero()
}

func (s *dateRangePickerDialogState) handleOk() {
	if s.entryMode == DatePickerEntryModeInput || s.entryMode == DatePickerEntryModeInputOnly {
		picker, _ := s.inputPickerKey.CurrentState()
		if !picker.validate() {
			s.SetState(func() { s.autoValidate = true })
			return
		}
	}
	var selected any
	if s.hasSelectedDateRange() {
		selected = NewDateTimeRange(s.selectedStart, s.selectedEnd)
	}
	widgets.NavigatorPop(s.Context(), selected)
}

func (s *dateRangePickerDialogState) handleCancel() { widgets.NavigatorPop(s.Context(), nil) }

func (s *dateRangePickerDialogState) handleEntryModeToggle() {
	s.SetState(func() {
		switch s.entryMode {
		case DatePickerEntryModeCalendar:
			s.autoValidate = false
			s.entryMode = DatePickerEntryModeInput
		case DatePickerEntryModeInput:
			// If invalid range (start after end), then just use the start date.
			if !s.selectedStart.IsZero() && !s.selectedEnd.IsZero() && s.selectedStart.After(s.selectedEnd) {
				s.selectedEnd = time.Time{}
			}
			// Have to check that the dates are selectable.
			if !s.selectedStart.IsZero() && !s.isDaySelectable(s.selectedStart) {
				s.selectedStart, s.selectedEnd = time.Time{}, time.Time{}
			} else if !s.selectedEnd.IsZero() && !s.isDaySelectable(s.selectedEnd) {
				s.selectedEnd = time.Time{}
			}
			s.entryMode = DatePickerEntryModeCalendar
		default:
			if foundation.DebugMode {
				panic(errors.New("DateRangePickerDialogEntryModeFixed", "the entry mode of a date range picker dialog can't change from the calendarOnly and inputOnly modes", "entryMode", s.entryMode))
			}
		}
	})
}

func (s *dateRangePickerDialogState) isDaySelectable(day time.Time) bool {
	w := s.Widget()
	if day.Before(w.firstDate) || day.After(w.lastDate) {
		return false
	}
	if w.o.SelectableDayPredicate == nil {
		return true
	}
	return w.o.SelectableDayPredicate(day, s.selectedStart, s.selectedEnd)
}

func (s *dateRangePickerDialogState) handleStartDateChanged(date time.Time) {
	s.SetState(func() { s.selectedStart = date })
}

func (s *dateRangePickerDialogState) handleEndDateChanged(date time.Time) {
	s.SetState(func() { s.selectedEnd = date })
}

func (s *dateRangePickerDialogState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	useMaterial3 := theme.UseMaterial3
	orientation := widgets.MediaQueryOrientationOf(bc)
	localizations := MaterialLocalizationsOf(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	showEntryModeButton := s.entryMode == DatePickerEntryModeCalendar || s.entryMode == DatePickerEntryModeInput

	var contents widgets.Widget
	var size ui.Size
	var elevation float64
	var shadowColor, surfaceTintColor foundation.Opt[ui.Color]
	var shape painting.ShapeBorder
	insetPadding := painting.EdgeInsets{}
	switch s.entryMode {
	case DatePickerEntryModeCalendar, DatePickerEntryModeCalendarOnly:
		var entryModeButton widgets.Widget
		if showEntryModeButton {
			entryModeButton = NewIconButton(s.handleEntryModeToggle, iconOr(o.SwitchToInputEntryModeIcon, editIcon(useMaterial3)),
				IconButtonOpts{Padding: painting.EdgeInsetsZero, Tooltip: localizations.InputDateModeButtonLabel()})
		}
		contents = newCalendarRangePickerDialog(calendarRangePickerDialogArgs{
			key: s.calendarKey, calendarDelegate: o.CalendarDelegate, selectedStartDate: s.selectedStart, selectedEndDate: s.selectedEnd,
			firstDate: w.firstDate, lastDate: w.lastDate, selectableDayPredicate: o.SelectableDayPredicate, currentDate: w.CurrentDate(),
			onStartDateChanged: s.handleStartDateChanged, onEndDateChanged: s.handleEndDateChanged,
			onConfirm: s.confirmIfRangeSelected(), onCancel: s.handleCancel, entryModeButton: entryModeButton,
			confirmText: orDefault(o.SaveText, upperIfMaterial2(useMaterial3, localizations.SaveButtonLabel())),
			helpText:    orDefault(o.HelpText, upperIfMaterial2(useMaterial3, localizations.DateRangePickerHelpText())),
		})
		size = widgets.MediaQuerySizeOf(bc)
		elevation = firstOpt(datePickerTheme.RangePickerElevation, defaults.RangePickerElevation).Value()
		shadowColor = firstOpt(datePickerTheme.RangePickerShadowColor, defaults.RangePickerShadowColor)
		surfaceTintColor = firstOpt(datePickerTheme.RangePickerSurfaceTintColor, defaults.RangePickerSurfaceTintColor)
		shape = firstNonNil(datePickerTheme.RangePickerShape, defaults.RangePickerShape)
	default:
		var entryModeButton widgets.Widget
		if showEntryModeButton {
			entryModeButton = NewIconButton(s.handleEntryModeToggle, iconOr(o.SwitchToCalendarEntryModeIcon, widgets.NewIcon(Icons.CalendarToday())),
				IconButtonOpts{Padding: painting.EdgeInsetsZero, Tooltip: localizations.CalendarModeButtonLabel()})
		}
		height := inputFormPortraitHeight
		if orientation != widgets.OrientationPortrait {
			height = inputFormLandscapeHeight
		}
		contents = newInputDateRangePickerDialog(inputDateRangePickerDialogArgs{
			calendarDelegate: o.CalendarDelegate, selectedStartDate: s.selectedStart, selectedEndDate: s.selectedEnd, currentDate: w.CurrentDate(),
			picker: widgets.NewSizedBox(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 24), widgets.NewColumn([]widgets.Widget{
				widgets.NewSpacer(),
				newInputDateRangePicker(inputDateRangePickerArgs{
					key: s.inputPickerKey, calendarDelegate: o.CalendarDelegate, initialStartDate: s.selectedStart, initialEndDate: s.selectedEnd,
					firstDate: w.firstDate, lastDate: w.lastDate, selectableDayPredicate: o.SelectableDayPredicate,
					onStartDateChanged: s.handleStartDateChanged, onEndDateChanged: s.handleEndDateChanged, autofocus: true,
					autovalidate: s.autoValidate, helpText: o.HelpText, errorInvalidRangeText: o.ErrorInvalidRangeText,
					errorFormatText: o.ErrorFormatText, errorInvalidText: o.ErrorInvalidText,
					fieldStartHintText: o.FieldStartHintText, fieldEndHintText: o.FieldEndHintText,
					fieldStartLabelText: o.FieldStartLabelText, fieldEndLabelText: o.FieldEndLabelText, keyboardType: o.KeyboardType,
				}),
				widgets.NewSpacer(),
			})), widgets.SizedBoxOpts{Height: foundation.Some(height)}),
			onConfirm: s.handleOk, onCancel: s.handleCancel, entryModeButton: entryModeButton,
			confirmText: o.ConfirmText, helpText: o.HelpText,
			cancelText: orDefault(o.CancelText, upperIfMaterial2(useMaterial3, localizations.CancelButtonLabel())),
		})
		dialogTheme := theme.DialogTheme
		switch {
		case orientation != widgets.OrientationPortrait:
			size = inputRangeLandscapeDialogSize
		case useMaterial3:
			size = inputPortraitDialogSizeM3
		default:
			size = inputPortraitDialogSizeM2
		}
		if useMaterial3 {
			elevation = firstOpt(datePickerTheme.Elevation, defaults.Elevation).Value()
			shape = firstNonNil(datePickerTheme.Shape, defaults.Shape)
		} else {
			elevation = firstOpt(datePickerTheme.Elevation, dialogTheme.Elevation, foundation.Some(24.0)).Value()
			shape = firstNonNil(datePickerTheme.Shape, dialogTheme.Shape, defaults.Shape)
		}
		shadowColor = firstOpt(datePickerTheme.ShadowColor, defaults.ShadowColor)
		surfaceTintColor = firstOpt(datePickerTheme.SurfaceTintColor, defaults.SurfaceTintColor)
		insetPadding = painting.NewEdgeInsetsSymmetric(24, 16)
	}
	return NewDialog(widgets.NewAnimatedContainer(dialogSizeAnimationDuration,
		widgets.NewMediaQueryWithClampedTextScaling(widgets.NewBuilder(func(widgets.BuildContext) widgets.Widget { return contents }),
			widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxRangeTextScaleFactor}),
		widgets.AnimatedContainerOpts{Width: foundation.Some(size.Width), Height: foundation.Some(size.Height), Curve: animation.Curves.EaseIn}),
		DialogOpts{
			InsetPadding: &insetPadding, BackgroundColor: firstOpt(datePickerTheme.BackgroundColor, defaults.BackgroundColor),
			Elevation: foundation.Some(elevation), ShadowColor: shadowColor, SurfaceTintColor: surfaceTintColor, Shape: shape,
			ClipBehavior: ui.ClipAntiAlias,
		})
}

// confirmIfRangeSelected is the confirm callback of the calendar mode: nil until
// a range is selected.
func (s *dateRangePickerDialogState) confirmIfRangeSelected() func() {
	if s.hasSelectedDateRange() {
		return s.handleOk
	}
	return nil
}

// calendarRangePickerDialogArgs are the parameters of calendarRangePickerDialog.
type calendarRangePickerDialogArgs struct {
	key                    foundation.Key
	selectedStartDate      time.Time
	selectedEndDate        time.Time
	firstDate              time.Time
	lastDate               time.Time
	selectableDayPredicate SelectableDayForRangePredicate
	currentDate            time.Time
	onStartDateChanged     func(time.Time)
	onEndDateChanged       func(time.Time)
	onConfirm              func()
	onCancel               func()
	confirmText            string
	helpText               string
	calendarDelegate       CalendarDelegate
	entryModeButton        widgets.Widget
}

// calendarRangePickerDialog is the full screen calendar of the date range
// picker, with a header that shows the range.
//
// Flutter: material/date_picker.dart _CalendarRangePickerDialog
type calendarRangePickerDialog struct {
	widgets.StatelessWidget
	a calendarRangePickerDialogArgs
}

func newCalendarRangePickerDialog(a calendarRangePickerDialogArgs) *calendarRangePickerDialog {
	return &calendarRangePickerDialog{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: a.key}), a: a}
}

func (w *calendarRangePickerDialog) Build(bc widgets.BuildContext) widgets.Widget {
	a := w.a
	theme := ThemeOf(bc)
	useMaterial3 := theme.UseMaterial3
	localizations := MaterialLocalizationsOf(bc)
	orientation := widgets.MediaQueryOrientationOf(bc)
	themeData := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	dialogBackground := firstOpt(themeData.RangePickerBackgroundColor, defaults.RangePickerBackgroundColor)
	headerBackground := firstOpt(themeData.RangePickerHeaderBackgroundColor, defaults.RangePickerHeaderBackgroundColor)
	headerForeground := firstOpt(themeData.RangePickerHeaderForegroundColor, defaults.RangePickerHeaderForegroundColor)
	var headerDisabledForeground foundation.Opt[ui.Color]
	if c, ok := headerForeground.Get(); ok {
		headerDisabledForeground = foundation.Some(c.WithOpacity(0.38))
	}
	headlineStyle := firstNonNil(themeData.RangePickerHeaderHeadlineStyle, defaults.RangePickerHeaderHeadlineStyle)
	headlineHelpStyle := firstNonNil(themeData.RangePickerHeaderHelpStyle, defaults.RangePickerHeaderHelpStyle)
	if headlineHelpStyle != nil {
		headlineHelpStyle = headlineHelpStyle.Apply(painting.TextStyleApply{Color: headerForeground})
	}
	startDateText := formatRangeStartDate(localizations, a.calendarDelegate, a.selectedStartDate, a.selectedEndDate)
	endDateText := formatRangeEndDate(localizations, a.calendarDelegate, a.selectedStartDate, a.selectedEndDate, a.calendarDelegate.Now())
	applyColor := func(set bool) *painting.TextStyle {
		if headlineStyle == nil {
			return nil
		}
		if set {
			return headlineStyle.Apply(painting.TextStyleApply{Color: headerForeground})
		}
		return headlineStyle.Apply(painting.TextStyleApply{Color: headerDisabledForeground})
	}
	startDateStyle := applyColor(!a.selectedStartDate.IsZero())
	endDateStyle := applyColor(!a.selectedEndDate.IsZero())
	buttonStyle := TextButtonStyleFrom(ButtonStyleFromOpts{ForegroundColor: headerForeground, DisabledForegroundColor: headerDisabledForeground})
	iconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: headerForeground})

	actions := []widgets.Widget{}
	if orientation == widgets.OrientationLandscape && a.entryModeButton != nil {
		actions = append(actions, a.entryModeButton)
	}
	actions = append(actions,
		NewTextButton(a.onConfirm, widgets.NewText(a.confirmText), TextButtonOpts{Style: buttonStyle}),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(8.0)}))

	leadingWidth := 72.0
	if widgets.MediaQueryWidthOf(bc) < 360 {
		leadingWidth = 42
	}
	bottomRow := []widgets.Widget{
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(leadingWidth)}),
		widgets.NewExpanded(widgets.NewSemantics(widgets.NewColumn([]widgets.Widget{
			widgets.NewText(a.helpText, widgets.TextOpts{Style: headlineHelpStyle, MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(8.0)}),
			widgets.NewRow([]widgets.Widget{
				widgets.NewText(startDateText, widgets.TextOpts{Style: startDateStyle, MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}),
				widgets.NewText(" – ", widgets.TextOpts{Style: startDateStyle}),
				widgets.NewFlexible(widgets.NewText(endDateText, widgets.TextOpts{Style: endDateStyle, MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})),
			}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(16.0)}),
		}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}),
			widgets.SemanticsOpts{Label: foundation.Some(a.helpText + " " + startDateText + " to " + endDateText), ExcludeSemantics: true})),
	}
	if orientation == widgets.OrientationPortrait && a.entryModeButton != nil {
		bottomRow = append(bottomRow, widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 8), widgets.NewIconTheme(iconTheme, a.entryModeButton)))
	}
	appBarOpts := AppBarOpts{
		IconTheme: foundation.Some(iconTheme), ActionsIconTheme: foundation.Some(iconTheme), BackgroundColor: headerBackground,
		Leading: NewCloseButton(ActionButtonOpts{OnPressed: a.onCancel}), Actions: actions,
		Bottom: widgets.NewPreferredSize(ui.Size{Width: math.Inf(1), Height: 64}, widgets.NewRow(bottomRow)),
	}
	if useMaterial3 {
		appBarOpts.Elevation = foundation.Some(0.0)
		appBarOpts.ScrolledUnderElevation = foundation.Some(0.0)
	}
	return widgets.NewSafeArea(NewScaffold(ScaffoldOpts{
		AppBar:          NewAppBar(appBarOpts),
		BackgroundColor: dialogBackground,
		Body: newCalendarDateRangePicker(calendarDateRangePickerArgs{
			initialStartDate: a.selectedStartDate, initialEndDate: a.selectedEndDate, firstDate: a.firstDate, lastDate: a.lastDate,
			currentDate: a.currentDate, onStartDateChanged: a.onStartDateChanged, onEndDateChanged: a.onEndDateChanged,
			selectableDayPredicate: a.selectableDayPredicate, calendarDelegate: a.calendarDelegate,
		}),
	}), widgets.SafeAreaOpts{Top: foundation.Some(false), Left: foundation.Some(false), Right: foundation.Some(false)})
}

// calendarDateRangePickerArgs are the parameters of calendarDateRangePicker.
type calendarDateRangePickerArgs struct {
	initialStartDate       time.Time
	initialEndDate         time.Time
	firstDate              time.Time
	lastDate               time.Time
	selectableDayPredicate SelectableDayForRangePredicate
	currentDate            time.Time
	onStartDateChanged     func(time.Time)
	onEndDateChanged       func(time.Time)
	calendarDelegate       CalendarDelegate
}

// calendarDateRangePicker displays a scrollable calendar grid that allows a user
// to select a range of dates.
//
// Flutter: material/date_picker.dart _CalendarDateRangePicker
type calendarDateRangePicker struct {
	widgets.StatefulWidget
	a calendarDateRangePickerArgs
}

func newCalendarDateRangePicker(a calendarDateRangePickerArgs) *calendarDateRangePicker {
	d := a.calendarDelegate
	dateOnly := func(t time.Time) time.Time {
		if t.IsZero() {
			return t
		}
		return d.DateOnly(t)
	}
	a.initialStartDate, a.initialEndDate = dateOnly(a.initialStartDate), dateOnly(a.initialEndDate)
	a.firstDate, a.lastDate = d.DateOnly(a.firstDate), d.DateOnly(a.lastDate)
	a.currentDate = d.DateOnly(firstNonZeroTime(a.currentDate, d.Now()))
	if foundation.DebugMode {
		switch {
		case !a.initialStartDate.IsZero() && !a.initialEndDate.IsZero() && a.initialStartDate.After(a.initialEndDate):
			panic(errors.New("CalendarDateRangePickerStartAfterEnd", "initialStartDate must be on or before initialEndDate"))
		case a.lastDate.Before(a.firstDate):
			panic(errors.New("CalendarDateRangePickerLastDateBeforeFirstDate", "firstDate must be on or before lastDate"))
		}
	}
	return &calendarDateRangePicker{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *calendarDateRangePicker) CreateState() widgets.State { return &calendarDateRangePickerState{} }

type calendarDateRangePickerState struct {
	widgets.StateOf[*calendarDateRangePicker]
	scrollViewKey         *widgets.GlobalKey[widgets.State]
	sliverAfterKey        foundation.Key
	startDate, endDate    time.Time
	initialMonthIndex     int
	controller            *widgets.ScrollControllerBase
	showWeekBottomDivider bool
	listenerHandle        foundation.ListenerHandle
}

func (s *calendarDateRangePickerState) InitState() {
	a := s.Widget().a
	s.scrollViewKey = widgets.NewGlobalKey[widgets.State]()
	s.sliverAfterKey = foundation.NewUniqueKey()
	s.controller = widgets.NewScrollController()
	s.listenerHandle = s.controller.AddListener(s.scrollListener)
	s.startDate, s.endDate = a.initialStartDate, a.initialEndDate
	// Calculate the index for the initially displayed month. This is needed to
	// divide the list of months into two `SliverList`s.
	initialDate := firstNonZeroTime(a.initialStartDate, a.currentDate)
	if !initialDate.Before(a.firstDate) && !initialDate.After(a.lastDate) {
		s.initialMonthIndex = a.calendarDelegate.MonthDelta(a.firstDate, initialDate)
	}
	s.showWeekBottomDivider = s.initialMonthIndex != 0
}

func (s *calendarDateRangePickerState) Dispose() {
	s.controller.RemoveListener(s.listenerHandle)
	s.controller.Dispose()
}

func (s *calendarDateRangePickerState) scrollListener() {
	if s.controller.Offset() <= s.controller.Position().MinScrollExtent() {
		s.SetState(func() { s.showWeekBottomDivider = false })
	} else if !s.showWeekBottomDivider {
		s.SetState(func() { s.showWeekBottomDivider = true })
	}
}

func (s *calendarDateRangePickerState) numberOfMonths() int {
	a := s.Widget().a
	return a.calendarDelegate.MonthDelta(a.firstDate, a.lastDate) + 1
}

func (s *calendarDateRangePickerState) updateSelection(date time.Time) {
	// Deviation: Flutter vibrates on Android and Fuchsia; Glutter has no haptic
	// feedback service (SPEC §12.10).
	a := s.Widget().a
	s.SetState(func() {
		if !s.startDate.IsZero() && s.endDate.IsZero() && !date.Before(s.startDate) {
			s.endDate = date
			if a.onEndDateChanged != nil {
				a.onEndDateChanged(s.endDate)
			}
		} else {
			s.startDate = date
			if a.onStartDateChanged != nil {
				a.onStartDateChanged(s.startDate)
			}
			if !s.endDate.IsZero() {
				s.endDate = time.Time{}
				if a.onEndDateChanged != nil {
					a.onEndDateChanged(s.endDate)
				}
			}
		}
	})
}

func (s *calendarDateRangePickerState) buildMonthItem(index int, beforeInitialMonth bool) widgets.Widget {
	a := s.Widget().a
	monthIndex := s.initialMonthIndex + index
	if beforeInitialMonth {
		monthIndex = s.initialMonthIndex - index - 1
	}
	month := a.calendarDelegate.AddMonthsToMonthDate(a.firstDate, monthIndex)
	return newMonthItem(monthItemArgs{
		calendarDelegate: a.calendarDelegate, selectedDateStart: s.startDate, selectedDateEnd: s.endDate, currentDate: a.currentDate,
		firstDate: a.firstDate, lastDate: a.lastDate, displayedMonth: month, onChanged: s.updateSelection,
		selectableDayPredicate: a.selectableDayPredicate,
	})
}

func (s *calendarDateRangePickerState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	children := []widgets.Widget{newDayHeaders()}
	if s.showWeekBottomDivider {
		children = append(children, NewDivider(DividerOpts{Height: foundation.Some(0.0)}))
	}
	children = append(children, widgets.NewExpanded(newCalendarKeyboardNavigator(calendarKeyboardNavigatorArgs{
		calendarDelegate: a.calendarDelegate, firstDate: a.firstDate, lastDate: a.lastDate,
		initialFocusedDay: firstNonZeroTime(s.startDate, a.initialStartDate, a.currentDate),
		child: widgets.NewCustomScrollView([]widgets.Widget{
			widgets.NewSliverListBuilder(func(_ widgets.BuildContext, index int) widgets.Widget { return s.buildMonthItem(index, true) },
				widgets.SliverBuilderOpts{ItemCount: foundation.Some(s.initialMonthIndex)}),
			widgets.NewSliverListBuilder(func(_ widgets.BuildContext, index int) widgets.Widget { return s.buildMonthItem(index, false) },
				widgets.SliverBuilderOpts{Key: s.sliverAfterKey, ItemCount: foundation.Some(s.numberOfMonths() - s.initialMonthIndex)}),
		}, widgets.CustomScrollViewOpts{Key: s.scrollViewKey, Controller: s.controller, Center: s.sliverAfterKey}),
	})))
	return widgets.NewColumn(children)
}

// calendarKeyboardNavigatorArgs are the parameters of calendarKeyboardNavigator.
type calendarKeyboardNavigatorArgs struct {
	child             widgets.Widget
	firstDate         time.Time
	lastDate          time.Time
	initialFocusedDay time.Time
	calendarDelegate  CalendarDelegate
}

// calendarKeyboardNavigator handles the arrow keys in the calendar.
//
// Flutter: material/date_picker.dart _CalendarKeyboardNavigator
type calendarKeyboardNavigator struct {
	widgets.StatefulWidget
	a calendarKeyboardNavigatorArgs
}

func newCalendarKeyboardNavigator(a calendarKeyboardNavigatorArgs) *calendarKeyboardNavigator {
	return &calendarKeyboardNavigator{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *calendarKeyboardNavigator) CreateState() widgets.State {
	return &calendarKeyboardNavigatorState{}
}

type calendarKeyboardNavigatorState struct {
	widgets.StateOf[*calendarKeyboardNavigator]
	shortcutMap           widgets.ShortcutMap
	actionMap             widgets.ActionMap
	dayGridFocus          *widgets.FocusNode
	dayTraversalDirection foundation.Opt[widgets.TraversalDirection]
	focusedDay            time.Time
}

func (s *calendarKeyboardNavigatorState) InitState() {
	arrow := func(k services.LogicalKeyboardKey, d widgets.TraversalDirection) widgets.ShortcutEntry {
		return widgets.ShortcutEntry{Activator: widgets.NewSingleActivator(k), Intent: widgets.DirectionalFocusIntent{Direction: d}}
	}
	s.shortcutMap = widgets.ShortcutMap{
		arrow(services.LogicalKeyboardKeyArrowLeft, widgets.TraversalDirectionLeft),
		arrow(services.LogicalKeyboardKeyArrowRight, widgets.TraversalDirectionRight),
		arrow(services.LogicalKeyboardKeyArrowDown, widgets.TraversalDirectionDown),
		arrow(services.LogicalKeyboardKeyArrowUp, widgets.TraversalDirectionUp),
	}
	next := widgets.NewCallbackAction(func(widgets.NextFocusIntent) any {
		s.dayGridFocus.RequestFocus()
		s.dayGridFocus.NextFocus()
		return nil
	})
	previous := widgets.NewCallbackAction(func(widgets.PreviousFocusIntent) any {
		s.dayGridFocus.RequestFocus()
		s.dayGridFocus.PreviousFocus()
		return nil
	})
	directional := widgets.NewCallbackAction(func(i widgets.DirectionalFocusIntent) any { s.handleDirectionFocus(i); return nil })
	s.actionMap = widgets.ActionMap{
		reflect.TypeFor[widgets.NextFocusIntent]():        next,
		reflect.TypeFor[widgets.PreviousFocusIntent]():    previous,
		reflect.TypeFor[widgets.DirectionalFocusIntent](): directional,
	}
	s.dayGridFocus = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Day Grid"})
}

func (s *calendarKeyboardNavigatorState) Dispose() { s.dayGridFocus.Dispose() }

func (s *calendarKeyboardNavigatorState) handleGridFocusChange(focused bool) {
	s.SetState(func() {
		if focused && s.focusedDay.IsZero() {
			s.focusedDay = s.Widget().a.initialFocusedDay
		}
	})
}

func (s *calendarKeyboardNavigatorState) handleDirectionFocus(intent widgets.DirectionalFocusIntent) {
	s.SetState(func() {
		if next := s.nextDateInDirection(s.focusedDay, intent.Direction); !next.IsZero() {
			s.focusedDay = next
			s.dayTraversalDirection = foundation.Some(intent.Direction)
		}
	})
}

func (s *calendarKeyboardNavigatorState) nextDateInDirection(date time.Time, direction widgets.TraversalDirection) time.Time {
	a := s.Widget().a
	next := a.calendarDelegate.AddDaysToDate(date, dayDirectionOffset(direction, widgets.DirectionalityOf(s.Context())))
	if !next.Before(a.firstDate) && !next.After(a.lastDate) {
		return next
	}
	return time.Time{}
}

func (s *calendarKeyboardNavigatorState) Build(widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	var date time.Time
	var scroll foundation.Opt[widgets.TraversalDirection]
	if s.dayGridFocus.HasFocus() {
		date, scroll = s.focusedDay, s.dayTraversalDirection
	}
	return widgets.NewFocusableActionDetector(
		&rangeFocusedDate{InheritedWidget: widgets.NewInheritedWidgetBase(a.child), calendarDelegate: a.calendarDelegate, date: date, scrollDirection: scroll},
		widgets.FocusableActionDetectorOpts{Shortcuts: s.shortcutMap, Actions: s.actionMap, FocusNode: s.dayGridFocus, OnFocusChange: s.handleGridFocusChange})
}

// rangeFocusedDate tells the days of a month item which day has the focus, and
// from which direction it came.
//
// Flutter: material/date_picker.dart _FocusedDate
type rangeFocusedDate struct {
	widgets.InheritedWidget
	calendarDelegate CalendarDelegate
	date             time.Time
	scrollDirection  foundation.Opt[widgets.TraversalDirection]
}

func (w *rangeFocusedDate) UpdateShouldNotify(old widgets.Widget) bool {
	o := old.(*rangeFocusedDate)
	return !w.calendarDelegate.IsSameDay(w.date, o.date) || w.scrollDirection != o.scrollDirection
}

// dayHeaders is the row of the narrow weekday names above the months.
//
// Flutter: material/date_picker.dart _DayHeaders
type dayHeaders struct{ widgets.StatelessWidget }

func newDayHeaders() *dayHeaders {
	return &dayHeaders{StatelessWidget: widgets.NewStatelessWidgetBase()}
}

func (w *dayHeaders) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	textStyle := theme.TextTheme.TitleSmall.Apply(painting.TextStyleApply{Color: foundation.Some(theme.ColorScheme.OnSurface())})
	l := MaterialLocalizationsOf(bc)
	labels := []widgets.Widget{widgets.NewSizedBoxShrink(nil)}
	for i := l.FirstDayOfWeekIndex(); len(labels) < daysPerWeek+1; i = (i + 1) % daysPerWeek {
		labels = append(labels, widgets.NewExcludeSemantics(widgets.NewCenter(widgets.NewText(l.NarrowWeekdays()[i], widgets.TextOpts{Style: textStyle}))))
	}
	labels = append(labels, widgets.NewSizedBoxShrink(nil))
	maxWidth := maxCalendarWidthPortrait
	if widgets.MediaQueryOrientationOf(bc) == widgets.OrientationLandscape {
		maxWidth = maxCalendarWidthLandscape
	}
	return widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(maxWidth), MaxHeight: foundation.Some(monthItemRowHeight)}),
		widgets.NewGridViewCustom(monthItemGridDelegate{}, widgets.NewSliverChildListDelegate(labels, widgets.SliverChildListDelegateOpts{AddRepaintBoundaries: foundation.Some(false)}),
			widgets.GridViewOpts{ShrinkWrap: true}))
}

// monthItemGridDelegate lays a month out in a grid of seven day columns with an
// edge column on each side.
//
// Flutter: material/date_picker.dart _MonthItemGridDelegate
type monthItemGridDelegate struct{}

func (monthItemGridDelegate) GetLayout(constraints rendering.SliverConstraints) rendering.SliverGridLayout {
	tileWidth := math.Max((constraints.CrossAxisExtent-2*rangeHorizontalPadding)/daysPerWeek, 0)
	return monthSliverGridLayout{
		crossAxisCount: daysPerWeek + 2, dayChildWidth: tileWidth, edgeChildWidth: rangeHorizontalPadding,
		reverseCrossAxis: painting.AxisDirectionIsReversed(constraints.CrossAxisDirection),
	}
}

func (monthItemGridDelegate) ShouldRelayout(rendering.SliverGridDelegate) bool { return false }

// monthSliverGridLayout is the layout of the grid of a month: the cells of the
// edge columns are narrower than the cells of the days.
//
// Flutter: material/date_picker.dart _MonthSliverGridLayout
type monthSliverGridLayout struct {
	crossAxisCount   int
	dayChildWidth    float64
	edgeChildWidth   float64
	reverseCrossAxis bool
}

func (l monthSliverGridLayout) rowHeight() float64 {
	return monthItemRowHeight + monthItemSpaceBetweenRows
}

func (l monthSliverGridLayout) GetMinChildIndexForScrollOffset(scrollOffset float64) int {
	return l.crossAxisCount * int(scrollOffset/l.rowHeight())
}

func (l monthSliverGridLayout) GetMaxChildIndexForScrollOffset(scrollOffset float64) int {
	mainAxisCount := int(math.Ceil(scrollOffset / l.rowHeight()))
	return max(0, l.crossAxisCount*mainAxisCount-1)
}

func (l monthSliverGridLayout) crossAxisOffset(crossAxisStart float64, isPadding bool) float64 {
	if l.reverseCrossAxis {
		extent := l.dayChildWidth
		if isPadding {
			extent = l.edgeChildWidth
		}
		return (float64(l.crossAxisCount-2)*l.dayChildWidth + 2*l.edgeChildWidth) - crossAxisStart - extent
	}
	return crossAxisStart
}

func (l monthSliverGridLayout) GetGeometryForChildIndex(index int) rendering.SliverGridGeometry {
	adjustedIndex := index % l.crossAxisCount
	isEdge := adjustedIndex == 0 || adjustedIndex == l.crossAxisCount-1
	crossAxisStart := math.Max(0, float64(adjustedIndex-1)*l.dayChildWidth+l.edgeChildWidth)
	crossAxisExtent := l.dayChildWidth
	if isEdge {
		crossAxisExtent = l.edgeChildWidth
	}
	return rendering.SliverGridGeometry{
		ScrollOffset:    float64(index/l.crossAxisCount) * l.rowHeight(),
		CrossAxisOffset: l.crossAxisOffset(crossAxisStart, isEdge),
		MainAxisExtent:  monthItemRowHeight,
		CrossAxisExtent: crossAxisExtent,
	}
}

func (l monthSliverGridLayout) ComputeMaxScrollOffset(childCount int) float64 {
	mainAxisCount := (childCount-1)/l.crossAxisCount + 1
	mainAxisSpacing := l.rowHeight() - monthItemRowHeight
	return l.rowHeight()*float64(mainAxisCount) - mainAxisSpacing
}

// monthItemArgs are the parameters of monthItem.
type monthItemArgs struct {
	selectedDateStart      time.Time
	selectedDateEnd        time.Time
	currentDate            time.Time
	onChanged              func(time.Time)
	firstDate              time.Time
	lastDate               time.Time
	displayedMonth         time.Time
	selectableDayPredicate SelectableDayForRangePredicate
	calendarDelegate       CalendarDelegate
}

// monthItem displays the days of a month, with the selected range highlighted.
//
// Flutter: material/date_picker.dart _MonthItem
type monthItem struct {
	widgets.StatefulWidget
	a monthItemArgs
}

func newMonthItem(a monthItemArgs) *monthItem {
	if foundation.DebugMode {
		outside := func(t time.Time) bool { return !t.IsZero() && (t.Before(a.firstDate) || t.After(a.lastDate)) }
		switch {
		case a.firstDate.After(a.lastDate):
			panic(errors.New("MonthItemFirstDateAfterLastDate", "the first date of a month item must not be after the last date"))
		case outside(a.selectedDateStart) || outside(a.selectedDateEnd):
			panic(errors.New("MonthItemSelectionOutOfRange", "the selected dates of a month item must be between the first and the last date"))
		case !a.selectedDateStart.IsZero() && !a.selectedDateEnd.IsZero() && a.selectedDateStart.After(a.selectedDateEnd):
			panic(errors.New("MonthItemSelectionReversed", "the selected start date of a month item must not be after the selected end date"))
		}
	}
	return &monthItem{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *monthItem) CreateState() widgets.State { return &monthItemState{} }

type monthItemState struct {
	widgets.StateOf[*monthItem]
	dayFocusNodes []*widgets.FocusNode
}

func (s *monthItemState) InitState() {
	a := s.Widget().a
	daysInMonth := a.calendarDelegate.DaysInMonth(a.displayedMonth.Year(), int(a.displayedMonth.Month()))
	s.dayFocusNodes = make([]*widgets.FocusNode, daysInMonth)
	for i := range s.dayFocusNodes {
		s.dayFocusNodes[i] = widgets.NewFocusNode(widgets.FocusNodeOpts{SkipTraversal: true, DebugLabel: "Day " + strconv.Itoa(i+1)})
	}
}

func (s *monthItemState) DidChangeDependencies() {
	a := s.Widget().a
	// Check to see if the focused date is in this month, if so focus it.
	if f, ok := widgets.DependOn[*rangeFocusedDate](s.Context()); ok && !f.date.IsZero() &&
		a.calendarDelegate.IsSameMonth(a.displayedMonth, f.date) {
		s.dayFocusNodes[f.date.Day()-1].RequestFocus()
	}
}

func (s *monthItemState) Dispose() {
	for _, n := range s.dayFocusNodes {
		n.Dispose()
	}
}

func (s *monthItemState) highlightColor(bc widgets.BuildContext) ui.Color {
	return firstOpt(DatePickerThemeOf(bc).RangeSelectionBackgroundColor, DatePickerThemeDefaults(bc).RangeSelectionBackgroundColor).Value()
}

func (s *monthItemState) dayFocusChanged(focused bool) {
	if !focused {
		return
	}
	f, ok := widgets.DependOn[*rangeFocusedDate](s.Context())
	if !ok {
		return
	}
	direction, ok := f.scrollDirection.Get()
	if !ok {
		return
	}
	policy := widgets.ScrollPositionAlignmentPolicyKeepVisibleAtEnd
	if direction == widgets.TraversalDirectionUp || direction == widgets.TraversalDirectionLeft {
		policy = widgets.ScrollPositionAlignmentPolicyKeepVisibleAtStart
	}
	if primary := widgets.WidgetsBindingOf(s.Context()).FocusManager().PrimaryFocus(); primary != nil && primary.Context() != nil {
		widgets.ScrollableEnsureVisible(primary.Context(), widgets.EnsureVisibleOpts{Duration: monthScrollDuration, AlignmentPolicy: policy})
	}
}

func (s *monthItemState) buildDayItem(bc widgets.BuildContext, dayToBuild time.Time) widgets.Widget {
	a := s.Widget().a
	start, end := a.selectedDateStart, a.selectedDateEnd
	isDisabled := dayToBuild.After(a.lastDate) || dayToBuild.Before(a.firstDate) ||
		(a.selectableDayPredicate != nil && !a.selectableDayPredicate(dayToBuild, start, end))
	isRangeSelected := !start.IsZero() && !end.IsZero()
	return newDayItem(dayItemArgs{
		calendarDelegate: a.calendarDelegate, day: dayToBuild, focusNode: s.dayFocusNodes[dayToBuild.Day()-1],
		onChanged: a.onChanged, onFocusChange: s.dayFocusChanged, highlightColor: s.highlightColor(bc), isDisabled: isDisabled,
		isRangeSelected:    isRangeSelected,
		isSelectedDayStart: !start.IsZero() && dayToBuild.Equal(start),
		isSelectedDayEnd:   !end.IsZero() && dayToBuild.Equal(end),
		isInRange:          isRangeSelected && dayToBuild.After(start) && dayToBuild.Before(end),
		isOneDayRange:      isRangeSelected && start.Equal(end),
		isToday:            a.calendarDelegate.IsSameDay(a.currentDate, dayToBuild),
	})
}

func (s *monthItemState) buildEdgeBox(bc widgets.BuildContext, isHighlighted bool) widgets.Widget {
	var empty widgets.Widget = widgets.NewLimitedBox(widgets.NewSizedBoxExpand(nil), widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0), MaxHeight: foundation.Some(0.0)})
	if isHighlighted {
		return widgets.NewColoredBox(s.highlightColor(bc), empty)
	}
	return empty
}

func (s *monthItemState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	theme := ThemeOf(bc)
	textTheme := theme.TextTheme
	localizations := MaterialLocalizationsOf(bc)
	d := a.calendarDelegate
	year, month := a.displayedMonth.Year(), int(a.displayedMonth.Month())
	daysInMonth := d.DaysInMonth(year, month)
	dayOffset := d.FirstDayOffset(year, month, localizations)
	weeks := int(math.Ceil(float64(daysInMonth+dayOffset) / daysPerWeek))
	gridHeight := float64(weeks)*monthItemRowHeight + float64(weeks-1)*monthItemSpaceBetweenRows
	start, end := a.selectedDateStart, a.selectedDateEnd
	hasRange := !start.IsZero() && !end.IsZero()

	var dayItems []widgets.Widget
	for day := 1 - dayOffset; day <= daysInMonth; day++ {
		if day < 1 {
			dayItems = append(dayItems, widgets.NewLimitedBox(widgets.NewSizedBoxExpand(nil), widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0), MaxHeight: foundation.Some(0.0)}))
		} else {
			dayItems = append(dayItems, s.buildDayItem(bc, d.GetDay(year, month, day)))
		}
	}
	// Add the leading/trailing edge containers to each week in order to correctly
	// extend the range highlight.
	var paddedDayItems []widgets.Widget
	for i := 0; i < weeks; i++ {
		first := i * daysPerWeek
		last := min(first+daysPerWeek, len(dayItems))
		week := append([]widgets.Widget(nil), dayItems[first:last]...)
		dateAfterLeadingPadding := d.GetDay(year, month, first-dayOffset+1)
		isLeadingInRange := (dayOffset <= 0 || i != 0) && hasRange && dateAfterLeadingPadding.After(start) && !dateAfterLeadingPadding.After(end)
		week = append([]widgets.Widget{s.buildEdgeBox(bc, isLeadingInRange)}, week...)
		// Only add a trailing edge box if it is not the end of the month.
		if last < len(dayItems) || (last == len(dayItems) && len(dayItems)%daysPerWeek == 0) {
			dateBeforeTrailingPadding := d.GetDay(year, month, last-dayOffset)
			isTrailingInRange := hasRange && !dateBeforeTrailingPadding.Before(start) && dateBeforeTrailingPadding.Before(end)
			week = append(week, s.buildEdgeBox(bc, isTrailingInRange))
		}
		paddedDayItems = append(paddedDayItems, week...)
	}
	maxWidth := maxCalendarWidthPortrait
	if widgets.MediaQueryOrientationOf(bc) == widgets.OrientationLandscape {
		maxWidth = maxCalendarWidthLandscape
	}
	return widgets.NewColumn([]widgets.Widget{
		widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(maxWidth)}).Tighten(rendering.TightenOpts{Height: foundation.Some(monthItemHeaderHeight)}),
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 16), widgets.NewAlign(
				widgets.NewExcludeSemantics(widgets.NewText(d.FormatMonthYear(a.displayedMonth, localizations),
					widgets.TextOpts{Style: textTheme.BodyMedium.Apply(painting.TextStyleApply{Color: foundation.Some(theme.ColorScheme.OnSurface())})})),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))),
		widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(maxWidth), MaxHeight: foundation.Some(gridHeight)}),
			widgets.NewGridViewCustom(monthItemGridDelegate{}, widgets.NewSliverChildListDelegate(paddedDayItems, widgets.SliverChildListDelegateOpts{AddRepaintBoundaries: foundation.Some(false)}),
				widgets.GridViewOpts{Physics: widgets.NewNeverScrollableScrollPhysics(nil)})),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(monthItemFooterHeight)}),
	})
}

// dayItemArgs are the parameters of dayItem.
type dayItemArgs struct {
	day                time.Time
	focusNode          *widgets.FocusNode
	onChanged          func(time.Time)
	onFocusChange      func(bool)
	highlightColor     ui.Color
	isDisabled         bool
	isRangeSelected    bool
	isSelectedDayStart bool
	isSelectedDayEnd   bool
	isInRange          bool
	isOneDayRange      bool
	isToday            bool
	calendarDelegate   CalendarDelegate
}

// dayItem is a day in a month item of the range picker.
//
// Flutter: material/date_picker.dart _DayItem
type dayItem struct {
	widgets.StatefulWidget
	a dayItemArgs
}

func newDayItem(a dayItemArgs) *dayItem {
	return &dayItem{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *dayItem) CreateState() widgets.State { return &dayItemState{} }

type dayItemState struct {
	widgets.StateOf[*dayItem]
	statesController *widgets.WidgetStatesController
}

func (s *dayItemState) InitState() { s.statesController = widgets.NewWidgetStatesController() }

func (s *dayItemState) Dispose() { s.statesController.Dispose() }

func (s *dayItemState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	localizations := MaterialLocalizationsOf(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	textDirection := widgets.DirectionalityOf(bc)

	var decoration painting.Decoration
	itemStyle := textTheme.BodyMedium
	var states widgets.WidgetStates
	if a.isDisabled {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if a.isSelectedDayStart || a.isSelectedDayEnd {
		states = states.With(widgets.WidgetStateSelected)
	}
	s.statesController.SetValue(states)

	type colorProp = widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	dayForeground := datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp { return t.DayForegroundColor }, states)
	dayBackground := datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp { return t.DayBackgroundColor }, states)
	dayOverlay := widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		return datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp {
			if a.isInRange {
				return t.RangeSelectionOverlayColor
			}
			return t.DayOverlayColor
		}, states)
	})
	dayShape := datePickerShape(datePickerTheme, defaults, func(t DatePickerThemeData) widgets.WidgetStateProperty[painting.OutlinedBorder] { return t.DayShape }, states)
	if dayShape == nil {
		dayShape = painting.NewCircleBorder()
	}
	apply := func(c ui.Color) {
		if itemStyle != nil {
			itemStyle = itemStyle.Apply(painting.TextStyleApply{Color: foundation.Some(c)})
		}
	}
	var highlight *highlightPainter
	switch {
	case a.isSelectedDayStart || a.isSelectedDayEnd:
		if itemStyle != nil {
			itemStyle = itemStyle.Apply(painting.TextStyleApply{Color: dayForeground})
		}
		decoration = painting.NewShapeDecoration(dayShape, painting.ShapeDecorationOpts{Color: dayBackground})
		if a.isRangeSelected && !a.isOneDayRange {
			style := highlightLeading
			if a.isSelectedDayStart {
				style = highlightTrailing
			}
			highlight = newHighlightPainter(a.highlightColor, style, textDirection)
		}
	case a.isInRange:
		highlight = newHighlightPainter(a.highlightColor, highlightAll, textDirection)
		if a.isDisabled {
			apply(colors.OnSurface().WithOpacity(0.38))
		}
	case a.isDisabled:
		apply(colors.OnSurface().WithOpacity(0.38))
	case a.isToday:
		apply(colors.Primary())
		todaySide := firstOpt(datePickerTheme.TodayBorder, defaults.TodayBorder).Value().CopyWith(painting.BorderSideCopy{Color: foundation.Some(colors.Primary())})
		decoration = painting.NewShapeDecoration(dayShape.CopyWithSide(todaySide))
	}

	dayText := localizations.FormatDecimal(a.day.Day())
	suffix := ""
	if a.isToday {
		suffix = ", " + localizations.CurrentDateLabel()
	}
	semanticLabel := dayText + ", " + a.calendarDelegate.FormatFullDate(a.day, localizations) + suffix
	if a.isSelectedDayStart {
		semanticLabel = localizations.DateRangeStartDateSemanticLabel(semanticLabel)
	} else if a.isSelectedDayEnd {
		semanticLabel = localizations.DateRangeEndDateSemanticLabel(semanticLabel)
	}
	var dayWidget widgets.Widget = widgets.NewContainer(widgets.NewSemantics(
		widgets.NewExcludeSemantics(widgets.NewText(dayText, widgets.TextOpts{Style: itemStyle})),
		widgets.SemanticsOpts{Label: foundation.Some(semanticLabel), Selected: foundation.Some(a.isSelectedDayStart || a.isSelectedDayEnd)}),
		widgets.ContainerOpts{Decoration: decoration, Alignment: painting.AlignmentCenter})
	if highlight != nil {
		dayWidget = widgets.NewCustomPaint(dayWidget, widgets.CustomPaintOpts{Painter: highlight})
	}
	if !a.isDisabled {
		dayWidget = NewInkResponse(dayWidget, InkResponseOpts{
			FocusNode: a.focusNode, OnTap: func() { a.onChanged(a.day) }, CustomBorder: dayShape, ContainedInkWell: true,
			StatesController: s.statesController, OverlayColor: dayOverlay, OnFocusChange: a.onFocusChange,
		})
	}
	return dayWidget
}

// highlightPainterStyle is which part of a day cell the range highlight covers.
//
// Flutter: material/date_picker.dart _HighlightPainterStyle
type highlightPainterStyle int

const (
	highlightNone highlightPainterStyle = iota
	highlightLeading
	highlightTrailing
	highlightAll
)

// highlightPainter paints the highlight of the range behind a day.
//
// Flutter: material/date_picker.dart _HighlightPainter
type highlightPainter struct {
	rendering.CustomPainterBase
	color         ui.Color
	style         highlightPainterStyle
	textDirection ui.TextDirection
}

func newHighlightPainter(color ui.Color, style highlightPainterStyle, textDirection ui.TextDirection) *highlightPainter {
	return &highlightPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), color: color, style: style, textDirection: textDirection}
}

func (p *highlightPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	if p.style == highlightNone {
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	paint.SetStyle(ui.PaintingStyleFill)
	rtl := p.textDirection != ui.TextDirectionLTR
	switch {
	case (p.style == highlightLeading && rtl) || (p.style == highlightTrailing && !rtl):
		canvas.DrawRect(ui.NewRectFromLTWH(size.Width/2, 0, size.Width/2, size.Height), paint)
	case p.style == highlightLeading || p.style == highlightTrailing:
		canvas.DrawRect(ui.NewRectFromLTWH(0, 0, size.Width/2, size.Height), paint)
	case p.style == highlightAll:
		canvas.DrawRect(ui.NewRectFromLTWH(0, 0, size.Width, size.Height), paint)
	}
}

func (p *highlightPainter) ShouldRepaint(rendering.CustomPainter) bool { return false }

// inputDateRangePickerDialogArgs are the parameters of inputDateRangePickerDialog.
type inputDateRangePickerDialogArgs struct {
	selectedStartDate time.Time
	selectedEndDate   time.Time
	currentDate       time.Time
	picker            widgets.Widget
	onConfirm         func()
	onCancel          func()
	confirmText       string
	cancelText        string
	helpText          string
	entryModeButton   widgets.Widget
	calendarDelegate  CalendarDelegate
}

// inputDateRangePickerDialog is the contents of the date range picker dialog in
// the input mode: the header, the fields and the buttons.
//
// Flutter: material/date_picker.dart _InputDateRangePickerDialog
type inputDateRangePickerDialog struct {
	widgets.StatelessWidget
	a inputDateRangePickerDialogArgs
}

func newInputDateRangePickerDialog(a inputDateRangePickerDialogArgs) *inputDateRangePickerDialog {
	return &inputDateRangePickerDialog{StatelessWidget: widgets.NewStatelessWidgetBase(), a: a}
}

func (w *inputDateRangePickerDialog) formatDateRange(bc widgets.BuildContext, start, end, now time.Time) string {
	l := MaterialLocalizationsOf(bc)
	startText := formatRangeStartDate(l, w.a.calendarDelegate, start, end)
	endText := formatRangeEndDate(l, w.a.calendarDelegate, start, end, now)
	if start.IsZero() || end.IsZero() {
		return l.UnspecifiedDateRange()
	}
	if widgets.DirectionalityOf(bc) == ui.TextDirectionRTL {
		return endText + " – " + startText
	}
	return startText + " – " + endText
}

func (w *inputDateRangePickerDialog) Build(bc widgets.BuildContext) widgets.Widget {
	a := w.a
	theme := ThemeOf(bc)
	useMaterial3 := theme.UseMaterial3
	localizations := MaterialLocalizationsOf(bc)
	orientation := widgets.MediaQueryOrientationOf(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	headlineStyle := theme.TextTheme.HeadlineSmall
	if orientation == widgets.OrientationPortrait {
		headlineStyle = firstNonNil(datePickerTheme.HeaderHeadlineStyle, defaults.HeaderHeadlineStyle)
	}
	headerForeground := firstOpt(datePickerTheme.HeaderForegroundColor, defaults.HeaderForegroundColor)
	if headlineStyle != nil {
		headlineStyle = headlineStyle.CopyWith(painting.TextStyleCopy{Color: headerForeground})
	}
	dateText := w.formatDateRange(bc, a.selectedStartDate, a.selectedEndDate, a.currentDate)
	semanticDateText := ""
	if !a.selectedStartDate.IsZero() && !a.selectedEndDate.IsZero() {
		semanticDateText = a.calendarDelegate.FormatMediumDate(a.selectedStartDate, localizations) + " – " + a.calendarDelegate.FormatMediumDate(a.selectedEndDate, localizations)
	}
	header := newDatePickerHeader(datePickerHeaderArgs{
		helpText:            orDefault(a.helpText, upperIfMaterial2(useMaterial3, localizations.DateRangePickerHelpText())),
		titleText:           dateText,
		titleSemanticsLabel: semanticDateText,
		titleStyle:          headlineStyle,
		orientation:         orientation,
		isShort:             orientation == widgets.OrientationLandscape,
		entryModeButton:     a.entryModeButton,
	})
	actions := widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(52.0)}),
		widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 8), widgets.NewAlign(widgets.NewOverflowBar([]widgets.Widget{
			NewTextButton(a.onCancel, widgets.NewText(orDefault(a.cancelText, upperIfMaterial2(useMaterial3, localizations.CancelButtonLabel())))),
			NewTextButton(a.onConfirm, widgets.NewText(orDefault(a.confirmText, localizations.OkButtonLabel()))),
		}, widgets.OverflowBarOpts{Spacing: 8}), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd})))
	textScaleFactor := textScaleFactorOf(bc, maxRangeTextScaleFactor)
	portraitSize := inputPortraitDialogSizeM2
	if useMaterial3 {
		portraitSize = inputPortraitDialogSizeM3
	}
	dialogSize := ui.Size{Width: portraitSize.Width * textScaleFactor, Height: portraitSize.Height * textScaleFactor}
	if orientation == widgets.OrientationPortrait {
		return widgets.NewLayoutBuilder(func(_ widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			isFullyPortrait := constraints.MaxHeight >= math.Min(dialogSize.Height, portraitSize.Height)
			var children []widgets.Widget
			if isFullyPortrait {
				children = append(children, header)
			}
			children = append(children, widgets.NewExpanded(a.picker), actions)
			return widgets.NewColumn(children, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
		})
	}
	return widgets.NewRow([]widgets.Widget{
		header,
		widgets.NewFlexible(widgets.NewColumn([]widgets.Widget{widgets.NewExpanded(a.picker), actions},
			widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})),
	}, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

// inputDateRangePickerArgs are the parameters of inputDateRangePicker.
type inputDateRangePickerArgs struct {
	key                    foundation.Key
	initialStartDate       time.Time
	initialEndDate         time.Time
	firstDate              time.Time
	lastDate               time.Time
	onStartDateChanged     func(time.Time)
	onEndDateChanged       func(time.Time)
	helpText               string
	errorFormatText        string
	errorInvalidText       string
	errorInvalidRangeText  string
	fieldStartHintText     string
	fieldEndHintText       string
	fieldStartLabelText    string
	fieldEndLabelText      string
	autofocus              bool
	autovalidate           bool
	keyboardType           foundation.Opt[services.TextInputType]
	selectableDayPredicate SelectableDayForRangePredicate
	calendarDelegate       CalendarDelegate
}

// inputDateRangePicker is the two text fields of the input mode of the date
// range picker.
//
// Flutter: material/date_picker.dart _InputDateRangePicker
type inputDateRangePicker struct {
	widgets.StatefulWidget
	a inputDateRangePickerArgs
}

func newInputDateRangePicker(a inputDateRangePickerArgs) *inputDateRangePicker {
	d := a.calendarDelegate
	dateOnly := func(t time.Time) time.Time {
		if t.IsZero() {
			return t
		}
		return d.DateOnly(t)
	}
	a.initialStartDate, a.initialEndDate = dateOnly(a.initialStartDate), dateOnly(a.initialEndDate)
	a.firstDate, a.lastDate = d.DateOnly(a.firstDate), d.DateOnly(a.lastDate)
	return &inputDateRangePicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: a.key}), a: a}
}

func (w *inputDateRangePicker) CreateState() widgets.State { return &inputDateRangePickerState{} }

type inputDateRangePickerState struct {
	widgets.StateOf[*inputDateRangePicker]
	startDate, endDate time.Time
	startController    *widgets.TextEditingController
	endController      *widgets.TextEditingController
	startErrorText     string
	endErrorText       string
	autoSelected       bool
}

func (s *inputDateRangePickerState) InitState() {
	a := s.Widget().a
	s.startDate, s.endDate = a.initialStartDate, a.initialEndDate
	s.startController = widgets.NewTextEditingController()
	s.endController = widgets.NewTextEditingController()
}

func (s *inputDateRangePickerState) Dispose() {
	s.startController.Dispose()
	s.endController.Dispose()
}

func (s *inputDateRangePickerState) DidChangeDependencies() {
	a := s.Widget().a
	localizations := MaterialLocalizationsOf(s.Context())
	if !s.startDate.IsZero() {
		text := a.calendarDelegate.FormatCompactDate(s.startDate, localizations)
		selectText := a.autofocus && !s.autoSelected
		s.updateController(s.startController, text, selectText)
		s.autoSelected = selectText
	}
	if !s.endDate.IsZero() {
		s.updateController(s.endController, a.calendarDelegate.FormatCompactDate(s.endDate, localizations), false)
	}
}

// validate validates the two dates and the range, and shows the errors. It
// returns whether they are valid.
//
// Flutter: material/date_picker.dart _InputDateRangePickerState.validate
func (s *inputDateRangePickerState) validate() bool {
	a := s.Widget().a
	startError := s.validateDate(s.startDate)
	endError := s.validateDate(s.endDate)
	if startError == "" && endError == "" && s.startDate.After(s.endDate) {
		startError = orDefault(a.errorInvalidRangeText, MaterialLocalizationsOf(s.Context()).InvalidDateRangeLabel())
	}
	s.SetState(func() {
		s.startErrorText = startError
		s.endErrorText = endError
	})
	return startError == "" && endError == ""
}

func (s *inputDateRangePickerState) parseDate(text string) time.Time {
	date, ok := s.Widget().a.calendarDelegate.ParseCompactDate(text, MaterialLocalizationsOf(s.Context()))
	if !ok {
		return time.Time{}
	}
	return date
}

func (s *inputDateRangePickerState) validateDate(date time.Time) string {
	a := s.Widget().a
	switch {
	case date.IsZero():
		return orDefault(a.errorFormatText, MaterialLocalizationsOf(s.Context()).InvalidDateFormatLabel())
	case !s.isDaySelectable(date):
		return orDefault(a.errorInvalidText, MaterialLocalizationsOf(s.Context()).DateOutOfRangeLabel())
	}
	return ""
}

func (s *inputDateRangePickerState) isDaySelectable(day time.Time) bool {
	a := s.Widget().a
	if day.Before(a.firstDate) || day.After(a.lastDate) {
		return false
	}
	if a.selectableDayPredicate == nil {
		return true
	}
	return a.selectableDayPredicate(day, s.startDate, s.endDate)
}

func (s *inputDateRangePickerState) updateController(controller *widgets.TextEditingController, text string, selectText bool) {
	value := controller.Value().CopyWith(services.TextEditingValueCopy{Text: foundation.Some(text)})
	if selectText {
		value = value.CopyWith(services.TextEditingValueCopy{Selection: foundation.Some(services.NewTextSelection(0, len([]rune(text))))})
	}
	controller.SetValue(value)
}

func (s *inputDateRangePickerState) handleStartChanged(text string) {
	a := s.Widget().a
	s.SetState(func() {
		s.startDate = s.parseDate(text)
		if a.onStartDateChanged != nil {
			a.onStartDateChanged(s.startDate)
		}
	})
	if a.autovalidate {
		s.validate()
	}
}

func (s *inputDateRangePickerState) handleEndChanged(text string) {
	a := s.Widget().a
	s.SetState(func() {
		s.endDate = s.parseDate(text)
		if a.onEndDateChanged != nil {
			a.onEndDateChanged(s.endDate)
		}
	})
	if a.autovalidate {
		s.validate()
	}
}

func (s *inputDateRangePickerState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	theme := ThemeOf(bc)
	localizations := MaterialLocalizationsOf(bc)
	inputTheme := InputDecorationThemeOf(bc)
	inputBorder := inputTheme.Border
	if inputBorder == nil {
		if theme.UseMaterial3 {
			inputBorder = NewOutlineInputBorder()
		} else {
			inputBorder = NewUnderlineInputBorder()
		}
	}
	keyboard := foundation.Some(a.keyboardType.OrElse(services.TextInputTypeDatetime))
	field := func(controller *widgets.TextEditingController, hint, label, errorText string, onChanged func(string), autofocus bool) widgets.Widget {
		decoration := InputDecoration{
			Border: inputBorder, Filled: foundation.Some(inputTheme.Filled),
			HintText:  foundation.Some(orDefault(hint, a.calendarDelegate.DateHelpText(localizations))),
			LabelText: foundation.Some(label),
		}
		if errorText != "" {
			decoration.ErrorText = foundation.Some(errorText)
		}
		return widgets.NewExpanded(NewTextField(TextFieldOpts{
			Controller: controller, Decoration: &decoration, KeyboardType: keyboard, OnChanged: onChanged, Autofocus: autofocus,
		}))
	}
	return widgets.NewRow([]widgets.Widget{
		field(s.startController, a.fieldStartHintText, orDefault(a.fieldStartLabelText, localizations.DateRangeStartLabel()), s.startErrorText, s.handleStartChanged, a.autofocus),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(8.0)}),
		field(s.endController, a.fieldEndHintText, orDefault(a.fieldEndLabelText, localizations.DateRangeEndLabel()), s.endErrorText, s.handleEndChanged, false),
	}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
}
