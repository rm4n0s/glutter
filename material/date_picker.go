// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"strings"
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

var (
	calendarPortraitDialogSizeM2  = ui.Size{Width: 330, Height: 518}
	calendarPortraitDialogSizeM3  = ui.Size{Width: 360, Height: 568}
	calendarLandscapeDialogSize   = ui.Size{Width: 496, Height: 346}
	inputPortraitDialogSizeM2     = ui.Size{Width: 330, Height: 270}
	inputPortraitDialogSizeM3     = ui.Size{Width: 328, Height: 270}
	inputLandscapeDialogSize      = ui.Size{Width: 496, Height: 160}
	inputRangeLandscapeDialogSize = ui.Size{Width: 496, Height: 164}
)

const (
	dialogSizeAnimationDuration       = 200 * time.Millisecond
	inputFormPortraitHeight           = 98.0
	inputFormLandscapeHeight          = 108.0
	datePickerMaxTextScaleFactor      = 3.0
	maxRangeTextScaleFactor           = 1.3
	maxHeaderTextScaleFactor          = 1.6
	maxHeaderWithEntryTextScaleFactor = 1.4
	maxHelpPortraitTextScaleFactor    = 1.6
	maxHelpLandscapeTextScaleFactor   = 1.4
)

// ShowDatePickerOpts are the optional parameters of ShowDatePicker.
//
// Flutter: material/date_picker.dart showDatePicker
type ShowDatePickerOpts struct {
	// InitialDate is the date that is selected first; the zero time for none.
	InitialDate time.Time
	// CurrentDate is the date that is shown as today; the default is the current
	// date.
	CurrentDate time.Time
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// calendar.
	InitialEntryMode DatePickerEntryMode
	// SelectableDayPredicate says which days can be selected.
	SelectableDayPredicate SelectableDayPredicate
	// HelpText is the text at the top of the dialog.
	HelpText string
	// CancelText is the label of the cancel button.
	CancelText string
	// ConfirmText is the label of the confirm button.
	ConfirmText string
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
	// InitialDatePickerMode is the picker that is shown first in the calendar
	// mode: the days, or the years.
	InitialDatePickerMode DatePickerMode
	// ErrorFormatText is the error text for an entry that is not a date.
	ErrorFormatText string
	// ErrorInvalidText is the error text for a date that is out of range or not
	// selectable.
	ErrorInvalidText string
	// FieldHintText is the hint of the field of the input mode.
	FieldHintText string
	// FieldLabelText is the label of the field of the input mode.
	FieldLabelText string
	// KeyboardType is the keyboard of the field of the input mode.
	KeyboardType foundation.Opt[services.TextInputType]
	// OnDatePickerModeChange is called when the user switches between the
	// calendar and the input mode.
	OnDatePickerModeChange func(mode DatePickerEntryMode)
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToCalendarEntryModeIcon is the icon of the button that switches to the
	// calendar mode.
	SwitchToCalendarEntryModeIcon *widgets.Icon
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// ShowDatePicker shows a dialog containing a Material Design date picker. The
// returned channel gets the date that the user confirmed, or the zero time (and
// is then closed) when the dialog was cancelled.
//
// In debug builds it panics with the tags ShowDatePickerLastDateBeforeFirstDate,
// ShowDatePickerInitialDateBeforeFirstDate, ShowDatePickerInitialDateAfterLastDate
// and ShowDatePickerInitialDateNotSelectable when the dates do not fit.
//
// Flutter: material/date_picker.dart showDatePicker
func ShowDatePicker(bc widgets.BuildContext, firstDate, lastDate time.Time, opts ...ShowDatePickerOpts) <-chan time.Time {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	d := o.CalendarDelegate
	initialDate := time.Time{}
	if !o.InitialDate.IsZero() {
		initialDate = d.DateOnly(o.InitialDate)
	}
	firstDate, lastDate = d.DateOnly(firstDate), d.DateOnly(lastDate)
	if foundation.DebugMode {
		switch {
		case lastDate.Before(firstDate):
			panic(errors.New("ShowDatePickerLastDateBeforeFirstDate", "lastDate must be on or after firstDate", "lastDate", lastDate, "firstDate", firstDate))
		case !initialDate.IsZero() && initialDate.Before(firstDate):
			panic(errors.New("ShowDatePickerInitialDateBeforeFirstDate", "initialDate must be on or after firstDate", "initialDate", initialDate, "firstDate", firstDate))
		case !initialDate.IsZero() && initialDate.After(lastDate):
			panic(errors.New("ShowDatePickerInitialDateAfterLastDate", "initialDate must be on or before lastDate", "initialDate", initialDate, "lastDate", lastDate))
		case o.SelectableDayPredicate != nil && !initialDate.IsZero() && !o.SelectableDayPredicate(initialDate):
			panic(errors.New("ShowDatePickerInitialDateNotSelectable", "the initialDate must satisfy the selectableDayPredicate", "initialDate", initialDate))
		}
	}
	DebugCheckHasMaterialLocalizations(bc)

	var dialog widgets.Widget = NewDatePickerDialog(firstDate, lastDate, DatePickerDialogOpts{
		InitialDate: initialDate, CurrentDate: o.CurrentDate, InitialEntryMode: o.InitialEntryMode,
		SelectableDayPredicate: o.SelectableDayPredicate, HelpText: o.HelpText, CancelText: o.CancelText, ConfirmText: o.ConfirmText,
		InitialCalendarMode: o.InitialDatePickerMode, ErrorFormatText: o.ErrorFormatText, ErrorInvalidText: o.ErrorInvalidText,
		FieldHintText: o.FieldHintText, FieldLabelText: o.FieldLabelText, KeyboardType: o.KeyboardType,
		OnDatePickerModeChange: o.OnDatePickerModeChange, SwitchToInputEntryModeIcon: o.SwitchToInputEntryModeIcon,
		SwitchToCalendarEntryModeIcon: o.SwitchToCalendarEntryModeIcon, CalendarDelegate: d,
	})
	if o.TextDirection != ui.TextDirectionUnspecified {
		dialog = widgets.NewDirectionality(o.TextDirection, dialog)
	}
	if locale, ok := o.Locale.Get(); ok {
		dialog = widgets.NewLocalizationsOverride(bc, foundation.Some(locale), nil, dialog)
	} else if locale, ok := DatePickerThemeOf(bc).Locale.Get(); ok {
		dialog = widgets.NewLocalizationsOverride(bc, foundation.Some(locale), nil, dialog)
	}
	return ShowDialog[time.Time](bc, func(bc widgets.BuildContext) widgets.Widget {
		if o.Builder == nil {
			return dialog
		}
		return o.Builder(bc, dialog)
	}, ShowDialogOpts{
		BarrierDismissible: o.BarrierDismissible, BarrierColor: o.BarrierColor, BarrierLabel: o.BarrierLabel,
		UseRootNavigator: o.UseRootNavigator, RouteSettings: o.RouteSettings,
	})
}

// DatePickerDialogOpts are the optional parameters of NewDatePickerDialog.
//
// Flutter: material/date_picker.dart DatePickerDialog.new
type DatePickerDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialDate is the date that is selected first; the zero time for none.
	InitialDate time.Time
	// CurrentDate is the date that is shown as today; the default is the current
	// date.
	CurrentDate time.Time
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// calendar.
	InitialEntryMode DatePickerEntryMode
	// SelectableDayPredicate says which days can be selected.
	SelectableDayPredicate SelectableDayPredicate
	// CancelText is the label of the cancel button.
	CancelText string
	// ConfirmText is the label of the confirm button.
	ConfirmText string
	// HelpText is the text at the top of the dialog.
	HelpText string
	// InitialCalendarMode is the picker that is shown first in the calendar mode.
	InitialCalendarMode DatePickerMode
	// ErrorFormatText is the error text for an entry that is not a date.
	ErrorFormatText string
	// ErrorInvalidText is the error text for a date that is out of range or not
	// selectable.
	ErrorInvalidText string
	// FieldHintText is the hint of the field of the input mode.
	FieldHintText string
	// FieldLabelText is the label of the field of the input mode.
	FieldLabelText string
	// KeyboardType is the keyboard of the field of the input mode.
	KeyboardType foundation.Opt[services.TextInputType]
	// OnDatePickerModeChange is called when the user switches between the
	// calendar and the input mode.
	OnDatePickerModeChange func(mode DatePickerEntryMode)
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToCalendarEntryModeIcon is the icon of the button that switches to the
	// calendar mode.
	SwitchToCalendarEntryModeIcon *widgets.Icon
	// InsetPadding is the distance between the edges of the screen and the dialog;
	// the default is 16 horizontally and 24 vertically.
	InsetPadding *painting.EdgeInsets
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// DatePickerDialog is a Material Design date picker dialog: a calendar and a
// text field to pick a date. Use ShowDatePicker to show it.
//
// Deviation: the state restoration (restorationId) is not ported (SPEC §12.10).
//
// Flutter: material/date_picker.dart DatePickerDialog
type DatePickerDialog struct {
	widgets.StatefulWidget
	initialDate time.Time
	firstDate   time.Time
	lastDate    time.Time
	currentDate time.Time
	o           DatePickerDialogOpts
}

// NewDatePickerDialog creates the dialog. The dates are reduced to the date part.
// In debug builds it panics with the tags DatePickerDialogLastDateBeforeFirstDate,
// DatePickerDialogInitialDateBeforeFirstDate,
// DatePickerDialogInitialDateAfterLastDate and
// DatePickerDialogInitialDateNotSelectable when the dates do not fit.
//
// Flutter: material/date_picker.dart DatePickerDialog.new
func NewDatePickerDialog(firstDate, lastDate time.Time, opts ...DatePickerDialogOpts) *DatePickerDialog {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	if o.InsetPadding == nil {
		p := painting.NewEdgeInsetsSymmetric(24, 16)
		o.InsetPadding = &p
	}
	d := o.CalendarDelegate
	w := &DatePickerDialog{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		firstDate:      d.DateOnly(firstDate), lastDate: d.DateOnly(lastDate), o: o,
	}
	if !o.InitialDate.IsZero() {
		w.initialDate = d.DateOnly(o.InitialDate)
	}
	current := o.CurrentDate
	if current.IsZero() {
		current = d.Now()
	}
	w.currentDate = d.DateOnly(current)
	if foundation.DebugMode {
		switch {
		case w.lastDate.Before(w.firstDate):
			panic(errors.New("DatePickerDialogLastDateBeforeFirstDate", "lastDate must be on or after firstDate", "lastDate", w.lastDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.Before(w.firstDate):
			panic(errors.New("DatePickerDialogInitialDateBeforeFirstDate", "initialDate must be on or after firstDate", "initialDate", w.initialDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.After(w.lastDate):
			panic(errors.New("DatePickerDialogInitialDateAfterLastDate", "initialDate must be on or before lastDate", "initialDate", w.initialDate, "lastDate", w.lastDate))
		case o.SelectableDayPredicate != nil && !w.initialDate.IsZero() && !o.SelectableDayPredicate(w.initialDate):
			panic(errors.New("DatePickerDialogInitialDateNotSelectable", "the initialDate must satisfy the selectableDayPredicate", "initialDate", w.initialDate))
		}
	}
	return w
}

// InitialDate is the date that is selected first; the zero time for none.
//
// Flutter: material/date_picker.dart DatePickerDialog.initialDate
func (w *DatePickerDialog) InitialDate() time.Time { return w.initialDate }

// FirstDate is the earliest date the user can select.
//
// Flutter: material/date_picker.dart DatePickerDialog.firstDate
func (w *DatePickerDialog) FirstDate() time.Time { return w.firstDate }

// LastDate is the latest date the user can select.
//
// Flutter: material/date_picker.dart DatePickerDialog.lastDate
func (w *DatePickerDialog) LastDate() time.Time { return w.lastDate }

// CurrentDate is the date that is shown as today.
//
// Flutter: material/date_picker.dart DatePickerDialog.currentDate
func (w *DatePickerDialog) CurrentDate() time.Time { return w.currentDate }

// CreateState creates the state.
//
// Flutter: material/date_picker.dart DatePickerDialog.createState
func (w *DatePickerDialog) CreateState() widgets.State { return &datePickerDialogState{} }

type datePickerDialogState struct {
	widgets.StateOf[*DatePickerDialog]
	selectedDate     time.Time
	entryMode        DatePickerEntryMode
	autovalidateMode widgets.AutovalidateMode
	calendarKey      *widgets.GlobalKey[widgets.State]
	formKey          *widgets.GlobalKey[*widgets.FormState]
}

func (s *datePickerDialogState) InitState() {
	w := s.Widget()
	s.selectedDate = w.initialDate
	s.entryMode = w.o.InitialEntryMode
	if s.entryMode == DatePickerEntryModeUnspecified {
		s.entryMode = DatePickerEntryModeCalendar
	}
	s.autovalidateMode = widgets.AutovalidateModeDisabled
	s.calendarKey = widgets.NewGlobalKey[widgets.State]()
	s.formKey = widgets.NewGlobalKey[*widgets.FormState]()
}

func (s *datePickerDialogState) isInput() bool {
	return s.entryMode == DatePickerEntryModeInput || s.entryMode == DatePickerEntryModeInputOnly
}

func (s *datePickerDialogState) handleOk() {
	if s.isInput() {
		form, _ := s.formKey.CurrentState()
		if !form.Validate() {
			s.SetState(func() { s.autovalidateMode = widgets.AutovalidateModeAlways })
			return
		}
		form.Save()
	}
	widgets.NavigatorPop(s.Context(), s.selectedDate)
}

func (s *datePickerDialogState) handleCancel() { widgets.NavigatorPop(s.Context(), nil) }

func (s *datePickerDialogState) handleEntryModeChanged() {
	if f := s.Widget().o.OnDatePickerModeChange; f != nil {
		f(s.entryMode)
	}
}

func (s *datePickerDialogState) handleEntryModeToggle() {
	s.SetState(func() {
		switch s.entryMode {
		case DatePickerEntryModeCalendar:
			s.autovalidateMode = widgets.AutovalidateModeDisabled
			s.entryMode = DatePickerEntryModeInput
			s.handleEntryModeChanged()
		case DatePickerEntryModeInput:
			form, _ := s.formKey.CurrentState()
			form.Save()
			s.entryMode = DatePickerEntryModeCalendar
			s.handleEntryModeChanged()
		default:
			if foundation.DebugMode {
				panic(errors.New("DatePickerDialogEntryModeFixed", "the entry mode of a date picker dialog can't change from the calendarOnly and inputOnly modes", "entryMode", s.entryMode))
			}
		}
	})
}

func (s *datePickerDialogState) handleDateChanged(date time.Time) {
	s.SetState(func() { s.selectedDate = date })
}

func (s *datePickerDialogState) dialogSize(bc widgets.BuildContext) ui.Size {
	useMaterial3 := ThemeOf(bc).UseMaterial3
	isCalendar := s.entryMode == DatePickerEntryModeCalendar || s.entryMode == DatePickerEntryModeCalendarOnly
	portrait := widgets.MediaQueryOrientationOf(bc) == widgets.OrientationPortrait
	switch {
	case isCalendar && portrait && useMaterial3:
		return calendarPortraitDialogSizeM3
	case !isCalendar && portrait && useMaterial3:
		return inputPortraitDialogSizeM3
	case isCalendar && portrait:
		return calendarPortraitDialogSizeM2
	case !isCalendar && portrait:
		return inputPortraitDialogSizeM2
	case isCalendar:
		return calendarLandscapeDialogSize
	default:
		return inputLandscapeDialogSize
	}
}

// upperIfMaterial2 is the text in upper case for Material 2, as the labels of the
// dialogs are.
func upperIfMaterial2(useMaterial3 bool, text string) string {
	if useMaterial3 {
		return text
	}
	return strings.ToUpper(text)
}

func (s *datePickerDialogState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)
	useMaterial3 := theme.UseMaterial3
	localizations := MaterialLocalizationsOf(bc)
	orientation := widgets.MediaQueryOrientationOf(bc)
	isLandscape := orientation == widgets.OrientationLandscape
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	textTheme := theme.TextTheme

	var headlineStyle *painting.TextStyle
	if useMaterial3 {
		headlineStyle = firstNonNil(datePickerTheme.HeaderHeadlineStyle, defaults.HeaderHeadlineStyle)
		if s.isInput() && orientation == widgets.OrientationLandscape {
			headlineStyle = textTheme.HeadlineSmall
		}
	} else if isLandscape {
		headlineStyle = textTheme.HeadlineSmall
	} else {
		headlineStyle = textTheme.HeadlineMedium
	}
	headerForeground := firstOpt(datePickerTheme.HeaderForegroundColor, defaults.HeaderForegroundColor)
	if headlineStyle != nil {
		headlineStyle = headlineStyle.CopyWith(painting.TextStyleCopy{Color: headerForeground})
	}

	cancelStyle := firstNonNil(datePickerTheme.CancelButtonStyle, defaults.CancelButtonStyle)
	confirmStyle := firstNonNil(datePickerTheme.ConfirmButtonStyle, defaults.ConfirmButtonStyle)
	actionsScale := datePickerMaxTextScaleFactor
	if isLandscape {
		actionsScale = 1.6
	}
	actions := widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(52.0)}),
		widgets.NewMediaQueryWithClampedTextScaling(
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 8), widgets.NewAlign(
				widgets.NewOverflowBar([]widgets.Widget{
					NewTextButton(s.handleCancel, widgets.NewText(orDefault(o.CancelText, upperIfMaterial2(useMaterial3, localizations.CancelButtonLabel()))), TextButtonOpts{Style: cancelStyle}),
					NewTextButton(s.handleOk, widgets.NewText(orDefault(o.ConfirmText, localizations.OkButtonLabel())), TextButtonOpts{Style: confirmStyle}),
				}, widgets.OverflowBarOpts{Spacing: 8}),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd})),
			widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: actionsScale}))

	calendarDatePicker := func() widgets.Widget {
		return NewCalendarDatePicker(s.selectedDate, w.firstDate, w.lastDate, s.handleDateChanged, CalendarDatePickerOpts{
			Key: s.calendarKey, CalendarDelegate: o.CalendarDelegate, CurrentDate: w.currentDate,
			SelectableDayPredicate: o.SelectableDayPredicate, InitialCalendarMode: o.InitialCalendarMode,
		})
	}
	inputDatePicker := func() widgets.Widget {
		height := inputFormPortraitHeight
		if orientation != widgets.OrientationPortrait {
			height = inputFormLandscapeHeight
		}
		return widgets.NewForm(widgets.NewSizedBox(
			widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 24), widgets.NewShortcuts(
				widgets.ShortcutMap{{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEnter), Intent: widgets.NextFocusIntent{}}},
				widgets.NewColumn([]widgets.Widget{
					widgets.NewFlexible(widgets.NewMediaQueryWithClampedTextScaling(
						NewInputDatePickerFormField(w.firstDate, w.lastDate, InputDatePickerFormFieldOpts{
							CalendarDelegate: o.CalendarDelegate, InitialDate: s.selectedDate,
							OnDateSubmitted: s.handleDateChanged, OnDateSaved: s.handleDateChanged,
							SelectableDayPredicate: o.SelectableDayPredicate, ErrorFormatText: o.ErrorFormatText,
							ErrorInvalidText: o.ErrorInvalidText, FieldHintText: o.FieldHintText, FieldLabelText: o.FieldLabelText,
							KeyboardType: o.KeyboardType, Autofocus: true,
						}), widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: 2.0})),
				}, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentCenter}))),
			widgets.SizedBoxOpts{Height: foundation.Some(height)}),
			widgets.FormOpts{Key: s.formKey, AutovalidateMode: s.autovalidateMode})
	}

	var picker, entryModeButton widgets.Widget
	switch s.entryMode {
	case DatePickerEntryModeCalendar:
		picker = calendarDatePicker()
		entryModeButton = NewIconButton(s.handleEntryModeToggle, iconOr(o.SwitchToInputEntryModeIcon, editIcon(useMaterial3)),
			IconButtonOpts{Color: headerForeground, Tooltip: localizations.InputDateModeButtonLabel()})
	case DatePickerEntryModeCalendarOnly:
		picker = calendarDatePicker()
	case DatePickerEntryModeInput:
		picker = inputDatePicker()
		entryModeButton = NewIconButton(s.handleEntryModeToggle, iconOr(o.SwitchToCalendarEntryModeIcon, widgets.NewIcon(Icons.CalendarToday())),
			IconButtonOpts{Color: headerForeground, Tooltip: localizations.CalendarModeButtonLabel()})
	case DatePickerEntryModeInputOnly:
		picker = inputDatePicker()
	}

	titleText := ""
	if !s.selectedDate.IsZero() {
		titleText = o.CalendarDelegate.FormatMediumDate(s.selectedDate, localizations)
	}
	header := newDatePickerHeader(datePickerHeaderArgs{
		helpText:        orDefault(o.HelpText, upperIfMaterial2(useMaterial3, localizations.DatePickerHelpText())),
		titleText:       titleText,
		titleStyle:      headlineStyle,
		orientation:     orientation,
		isShort:         orientation == widgets.OrientationLandscape,
		entryModeButton: entryModeButton,
	})

	textScaleFactor := textScaleFactorOf(bc, datePickerMaxTextScaleFactor)
	dialogSize := s.dialogSize(bc)
	dialogSize = ui.Size{Width: dialogSize.Width * textScaleFactor, Height: dialogSize.Height * textScaleFactor}
	dialogTheme := theme.DialogTheme

	var elevation float64
	var shape painting.ShapeBorder
	if useMaterial3 {
		elevation = firstOpt(datePickerTheme.Elevation, defaults.Elevation).Value()
		shape = firstNonNil(datePickerTheme.Shape, defaults.Shape)
	} else {
		elevation = firstOpt(datePickerTheme.Elevation, dialogTheme.Elevation, foundation.Some(24.0)).Value()
		shape = firstNonNil(datePickerTheme.Shape, dialogTheme.Shape, defaults.Shape)
	}
	return NewDialog(widgets.NewAnimatedContainer(dialogSizeAnimationDuration,
		widgets.NewMediaQueryWithClampedTextScaling(widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			portraitDialogSize := inputPortraitDialogSizeM2
			if useMaterial3 {
				portraitDialogSize = inputPortraitDialogSizeM3
			}
			isFullyPortrait := constraints.MaxHeight >= math.Min(dialogSize.Height, portraitDialogSize.Height)
			if orientation == widgets.OrientationPortrait {
				showHeader := isFullyPortrait || !s.isInput()
				showPicker := isFullyPortrait || s.isInput()
				var children []widgets.Widget
				if showHeader {
					children = append(children, header)
				}
				if useMaterial3 {
					children = append(children, NewDivider(DividerOpts{Height: foundation.Some(0.0), Color: datePickerTheme.DividerColor}))
				}
				if showPicker {
					children = append(children, widgets.NewExpanded(picker), actions)
				}
				return widgets.NewColumn(children, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
			}
			children := []widgets.Widget{header}
			if useMaterial3 {
				children = append(children, NewVerticalDivider(VerticalDividerOpts{Width: foundation.Some(0.0), Color: datePickerTheme.DividerColor}))
			}
			children = append(children, widgets.NewFlexible(widgets.NewColumn([]widgets.Widget{widgets.NewExpanded(picker), actions},
				widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})))
			return widgets.NewRow(children, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
		}), widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: datePickerMaxTextScaleFactor}),
		widgets.AnimatedContainerOpts{Width: foundation.Some(dialogSize.Width), Height: foundation.Some(dialogSize.Height), Curve: animation.Curves.EaseIn}),
		DialogOpts{
			BackgroundColor:  firstOpt(datePickerTheme.BackgroundColor, defaults.BackgroundColor),
			Elevation:        foundation.Some(elevation),
			ShadowColor:      firstOpt(datePickerTheme.ShadowColor, defaults.ShadowColor),
			SurfaceTintColor: firstOpt(datePickerTheme.SurfaceTintColor, defaults.SurfaceTintColor),
			Shape:            shape,
			InsetPadding:     o.InsetPadding,
			ClipBehavior:     ui.ClipAntiAlias,
		})
}

// iconOr returns the icon, or the default one.
func iconOr(icon *widgets.Icon, def widgets.Widget) widgets.Widget {
	if icon != nil {
		return icon
	}
	return def
}

// editIcon is the icon of the button that switches to the input mode.
func editIcon(useMaterial3 bool) widgets.Widget {
	if useMaterial3 {
		return widgets.NewIcon(Icons.EditOutlined())
	}
	return widgets.NewIcon(Icons.Edit())
}

// datePickerHeaderArgs are the parameters of datePickerHeader.
type datePickerHeaderArgs struct {
	helpText            string
	titleText           string
	titleSemanticsLabel string
	titleStyle          *painting.TextStyle
	orientation         widgets.Orientation
	isShort             bool
	entryModeButton     widgets.Widget
}

// datePickerHeader is the header of the date picker dialog: the help text and
// the selected date, with the button that switches the entry mode.
//
// Flutter: material/date_picker.dart _DatePickerHeader
type datePickerHeader struct {
	widgets.StatelessWidget
	a datePickerHeaderArgs
}

func newDatePickerHeader(a datePickerHeaderArgs) *datePickerHeader {
	return &datePickerHeader{StatelessWidget: widgets.NewStatelessWidgetBase(), a: a}
}

const (
	datePickerHeaderLandscapeWidth   = 152.0
	datePickerHeaderPortraitHeight   = 120.0
	datePickerHeaderPaddingLandscape = 16.0
)

func (w *datePickerHeader) Build(bc widgets.BuildContext) widgets.Widget {
	a := w.a
	theme := ThemeOf(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	backgroundColor := firstOpt(datePickerTheme.HeaderBackgroundColor, defaults.HeaderBackgroundColor)
	foregroundColor := firstOpt(datePickerTheme.HeaderForegroundColor, defaults.HeaderForegroundColor)
	helpStyle := firstNonNil(datePickerTheme.HeaderHelpStyle, defaults.HeaderHelpStyle)
	if helpStyle != nil {
		helpStyle = helpStyle.CopyWith(painting.TextStyleCopy{Color: foregroundColor})
	}

	scaler := widgets.MediaQueryTextScalerOf(bc)
	currentScale := scaler.Scale(calendarFontSizeToScale) / calendarFontSizeToScale
	maxScale := maxHeaderTextScaleFactor
	if a.entryModeButton != nil {
		maxScale = maxHeaderWithEntryTextScaleFactor
	}
	maxHeaderTextScale := math.Min(currentScale, maxScale)
	textScaleFactor := textScaleFactorOf(bc, maxHeaderTextScale)
	titleFontSize := 32.0
	if a.titleStyle != nil && a.titleStyle.FontSize.IsSome() {
		titleFontSize = a.titleStyle.FontSize.Value()
	}
	scaledFontSize := scaler.Scale(titleFontSize)
	headerScaleFactor := 1.0
	if textScaleFactor > 1 {
		headerScaleFactor = textScaleFactor
	}
	helpMax := maxHelpLandscapeTextScaleFactor
	if a.orientation == widgets.OrientationPortrait {
		helpMax = maxHelpPortraitTextScaleFactor
	}
	help := widgets.NewText(a.helpText, widgets.TextOpts{
		Style: helpStyle, MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis,
		TextScaler: scaler.Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(math.Min(textScaleFactor, helpMax))}),
	})
	var maxLines int
	switch {
	case a.orientation == widgets.OrientationPortrait && scaledFontSize > 70:
		maxLines = 2
	case a.orientation == widgets.OrientationPortrait:
		maxLines = 1
	case scaledFontSize > 40:
		maxLines = 3
	default:
		maxLines = 2
	}
	label := a.titleSemanticsLabel
	if label == "" {
		label = a.titleText
	}
	title := widgets.NewText(a.titleText, widgets.TextOpts{
		SemanticsLabel: foundation.Some(label), Style: a.titleStyle, MaxLines: foundation.Some(maxLines), Overflow: painting.TextOverflowEllipsis,
		TextScaler: scaler.Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(textScaleFactor)}),
	})
	fontScaleAdjustedHeaderHeight := 1.0
	if headerScaleFactor > 1.3 {
		fontScaleAdjustedHeaderHeight = headerScaleFactor - 0.2
	}

	if a.orientation == widgets.OrientationPortrait {
		titleRow := []widgets.Widget{widgets.NewExpanded(title)}
		if a.entryModeButton != nil {
			titleRow = append(titleRow, widgets.NewSemantics(a.entryModeButton, widgets.SemanticsOpts{Container: true}))
		}
		return widgets.NewSemantics(widgets.NewSizedBox(NewMaterial(
			widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(24, 0, 12, 12), widgets.NewColumn([]widgets.Widget{
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(16.0)}),
				help,
				widgets.NewFlexible(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(38.0)})),
				widgets.NewRow(titleRow),
			}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})),
			MaterialOpts{Color: backgroundColor}),
			widgets.SizedBoxOpts{Height: foundation.Some(datePickerHeaderPortraitHeight * fontScaleAdjustedHeaderHeight)}),
			widgets.SemanticsOpts{Container: true})
	}
	gap := 56.0
	if a.isShort {
		gap = 16
	}
	horizontal := painting.NewEdgeInsetsSymmetric(0, datePickerHeaderPaddingLandscape)
	children := []widgets.Widget{
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(16.0)}),
		widgets.NewPadding(horizontal, help),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(gap)}),
		widgets.NewExpanded(widgets.NewPadding(horizontal, title)),
	}
	if a.entryModeButton != nil {
		var padding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsSymmetric(0, 4)
		if theme.UseMaterial3 {
			padding = painting.NewEdgeInsetsDirectionalFromSTEB(8, 0, 4, 6)
		}
		children = append(children, widgets.NewPadding(padding, widgets.NewSemantics(a.entryModeButton, widgets.SemanticsOpts{Container: true})))
	}
	return widgets.NewSemantics(widgets.NewSizedBox(NewMaterial(
		widgets.NewColumn(children, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}),
		MaterialOpts{Color: backgroundColor}),
		widgets.SizedBoxOpts{Width: foundation.Some(datePickerHeaderLandscapeWidth)}),
		widgets.SemanticsOpts{Container: true})
}
