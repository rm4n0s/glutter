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
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	monthScrollDuration                  = 200 * time.Millisecond
	dayPickerRowHeightM2                 = 42.0
	dayPickerRowHeightM3                 = 48.0
	maxDayPickerRowCount                 = 6 // A 31 day month that starts on Saturday.
	maxDayPickerHeightM2                 = dayPickerRowHeightM2 * (maxDayPickerRowCount + 1)
	maxDayPickerHeightM3                 = dayPickerRowHeightM3 * (maxDayPickerRowCount + 1)
	monthPickerHorizontalPaddingPortrait = 12.0
	monthPickerHorizontalPaddingOther    = 8.0
	yearPickerColumnCount                = 3
	yearPickerPadding                    = 16.0
	yearPickerRowHeight                  = 52.0
	yearPickerRowSpacing                 = 8.0
	subHeaderHeight                      = 52.0
	monthNavButtonsWidth                 = 108.0
	calendarMaxTextScaleFactor           = 3.0
	modeToggleButtonMaxScaleFactor       = 2.0
	dayPickerGridPortraitMaxScaleFactor  = 2.0
	dayPickerGridLandscapeMaxScaleFactor = 1.5
	calendarFontSizeToScale              = 14.0
	daysPerWeek                          = 7
)

// SelectableDayPredicate is the signature of a function that says whether a day
// can be selected in a date picker.
//
// Flutter: material/calendar_date_picker.dart SelectableDayPredicate
type SelectableDayPredicate func(day time.Time) bool

// textScaleFactorOf is the scale factor of the text scaler of the context
// (clamped to maxScale), as the date pickers measure it: by scaling a font size
// of 14.
func textScaleFactorOf(bc widgets.BuildContext, maxScale float64) float64 {
	return widgets.MediaQueryTextScalerOf(bc).Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(maxScale)}).
		Scale(calendarFontSizeToScale) / calendarFontSizeToScale
}

// datePickerColor resolves a color property of the date picker theme for the
// states: the one of the theme if it gives a color, then the one of the defaults.
func datePickerColor(theme, defaults DatePickerThemeData,
	get func(DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]], states widgets.WidgetStates) foundation.Opt[ui.Color] {
	if p := get(theme); p != nil {
		if c := p.Resolve(states); c.IsSome() {
			return c
		}
	}
	if p := get(defaults); p != nil {
		return p.Resolve(states)
	}
	return foundation.None[ui.Color]()
}

// datePickerShape resolves a shape property of the date picker theme for the
// states: the one of the theme if it gives a shape, then the one of the defaults.
func datePickerShape(theme, defaults DatePickerThemeData,
	get func(DatePickerThemeData) widgets.WidgetStateProperty[painting.OutlinedBorder], states widgets.WidgetStates) painting.OutlinedBorder {
	if p := get(theme); p != nil {
		if s := p.Resolve(states); s != nil {
			return s
		}
	}
	if p := get(defaults); p != nil {
		return p.Resolve(states)
	}
	return nil
}

// CalendarDatePickerOpts are the optional parameters of NewCalendarDatePicker.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.new
type CalendarDatePickerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CurrentDate is the date that is shown as today; the default is the current
	// date.
	CurrentDate time.Time
	// OnDisplayedMonthChanged is called when the user navigates to another month
	// in the day picker.
	OnDisplayedMonthChanged func(date time.Time)
	// InitialCalendarMode is the picker that is shown first: the days of a month,
	// or the years. The default is the days.
	InitialCalendarMode DatePickerMode
	// SelectableDayPredicate says which days can be selected; the default is all
	// of them.
	SelectableDayPredicate SelectableDayPredicate
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// CalendarDatePicker displays a scrollable calendar grid that allows a user to
// select a single date.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker
type CalendarDatePicker struct {
	widgets.StatefulWidget
	initialDate time.Time
	firstDate   time.Time
	lastDate    time.Time
	currentDate time.Time
	onChanged   func(time.Time)
	o           CalendarDatePickerOpts
}

// NewCalendarDatePicker creates a calendar. The dates are reduced to the date
// part. The initial date is the zero time for none. In debug builds it panics
// with the tags CalendarDatePickerLastDateBeforeFirstDate,
// CalendarDatePickerInitialDateBeforeFirstDate,
// CalendarDatePickerInitialDateAfterLastDate and
// CalendarDatePickerInitialDateNotSelectable when the dates do not fit.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.new
func NewCalendarDatePicker(initialDate, firstDate, lastDate time.Time, onDateChanged func(time.Time), opts ...CalendarDatePickerOpts) *CalendarDatePicker {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	d := o.CalendarDelegate
	w := &CalendarDatePicker{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		firstDate:      d.DateOnly(firstDate),
		lastDate:       d.DateOnly(lastDate),
		onChanged:      onDateChanged,
		o:              o,
	}
	if !initialDate.IsZero() {
		w.initialDate = d.DateOnly(initialDate)
	}
	current := o.CurrentDate
	if current.IsZero() {
		current = d.Now()
	}
	w.currentDate = d.DateOnly(current)
	if foundation.DebugMode {
		const tagPrefix = "CalendarDatePicker"
		switch {
		case w.lastDate.Before(w.firstDate):
			panic(errors.New(tagPrefix+"LastDateBeforeFirstDate", "lastDate must be on or after firstDate", "lastDate", w.lastDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.Before(w.firstDate):
			panic(errors.New(tagPrefix+"InitialDateBeforeFirstDate", "initialDate must be on or after firstDate", "initialDate", w.initialDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.After(w.lastDate):
			panic(errors.New(tagPrefix+"InitialDateAfterLastDate", "initialDate must be on or before lastDate", "initialDate", w.initialDate, "lastDate", w.lastDate))
		case o.SelectableDayPredicate != nil && !w.initialDate.IsZero() && !o.SelectableDayPredicate(w.initialDate):
			panic(errors.New(tagPrefix+"InitialDateNotSelectable", "the initialDate must satisfy the selectableDayPredicate", "initialDate", w.initialDate))
		}
	}
	return w
}

// InitialDate is the date that is selected first; the zero time for none.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.initialDate
func (w *CalendarDatePicker) InitialDate() time.Time { return w.initialDate }

// FirstDate is the earliest date the user can select.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.firstDate
func (w *CalendarDatePicker) FirstDate() time.Time { return w.firstDate }

// LastDate is the latest date the user can select.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.lastDate
func (w *CalendarDatePicker) LastDate() time.Time { return w.lastDate }

// CurrentDate is the date that is shown as today.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.currentDate
func (w *CalendarDatePicker) CurrentDate() time.Time { return w.currentDate }

// CreateState creates the state.
//
// Flutter: material/calendar_date_picker.dart CalendarDatePicker.createState
func (w *CalendarDatePicker) CreateState() widgets.State { return &calendarDatePickerState{} }

type calendarDatePickerState struct {
	widgets.StateOf[*CalendarDatePicker]
	announcedInitialDate bool
	announcementText     string
	mode                 DatePickerMode
	currentMonthDate     time.Time
	selectedDate         time.Time
	monthPickerKey       *widgets.GlobalKey[widgets.State]
	yearPickerKey        *widgets.GlobalKey[widgets.State]
	localizations        MaterialLocalizations
	textDirection        ui.TextDirection
}

func (s *calendarDatePickerState) delegate() CalendarDelegate { return s.Widget().o.CalendarDelegate }

func (s *calendarDatePickerState) InitState() {
	w := s.Widget()
	s.monthPickerKey = widgets.NewGlobalKey[widgets.State]()
	s.yearPickerKey = widgets.NewGlobalKey[widgets.State]()
	s.mode = w.o.InitialCalendarMode
	displayed := w.currentDate
	if !w.initialDate.IsZero() {
		displayed = w.initialDate
		s.selectedDate = w.initialDate
	}
	s.currentMonthDate = s.delegate().GetMonth(displayed.Year(), int(displayed.Month()))
}

func (s *calendarDatePickerState) DidChangeDependencies() {
	bc := s.Context()
	DebugCheckHasMaterial(bc)
	DebugCheckHasMaterialLocalizations(bc)
	widgets.DebugCheckHasDirectionality(bc)
	s.localizations = MaterialLocalizationsOf(bc)
	s.textDirection = widgets.DirectionalityOf(bc)
	w := s.Widget()
	if !s.announcedInitialDate && !w.initialDate.IsZero() {
		s.announcedInitialDate = true
		suffix := ""
		if s.delegate().IsSameDay(w.currentDate, s.selectedDate) {
			suffix = ", " + s.localizations.CurrentDateLabel()
		}
		s.announce(s.localizations.FormatFullDate(s.selectedDate) + suffix)
	}
}

// announce stores the message as the label of a live region, which is how
// the platform reads it out.
//
// Deviation: Flutter sends the message through SemanticsService.sendAnnouncement
// when the platform supports announcements; Glutter has no such service, and
// uses the path that Flutter takes for the platforms that do not support it (SPEC
// §12.10).
//
// Flutter: material/calendar_date_picker.dart _CalendarDatePickerState._announce
func (s *calendarDatePickerState) announce(message string) { s.announcementText = message }

func (s *calendarDatePickerState) handleModeChanged(mode DatePickerMode) {
	// Deviation: Flutter vibrates on Android, Fuchsia, Linux and Windows; Glutter
	// has no haptic feedback service (SPEC §12.10).
	s.SetState(func() {
		s.mode = mode
		if !s.selectedDate.IsZero() {
			if mode == DatePickerModeDay {
				s.announce(s.delegate().FormatMonthYear(s.selectedDate, s.localizations))
			} else {
				s.announce(s.delegate().FormatYear(s.selectedDate.Year(), s.localizations))
			}
		}
	})
}

func (s *calendarDatePickerState) handleMonthChanged(date time.Time) {
	s.SetState(func() {
		if s.currentMonthDate.Year() != date.Year() || s.currentMonthDate.Month() != date.Month() {
			s.currentMonthDate = s.delegate().GetMonth(date.Year(), int(date.Month()))
			if f := s.Widget().o.OnDisplayedMonthChanged; f != nil {
				f(s.currentMonthDate)
			}
		}
	})
}

func (s *calendarDatePickerState) handleYearChanged(value time.Time) {
	w := s.Widget()
	d := s.delegate()
	daysInMonth := d.DaysInMonth(value.Year(), int(value.Month()))
	preferredDay := 1
	if !s.selectedDate.IsZero() {
		preferredDay = s.selectedDate.Day()
	}
	value = d.GetDay(value.Year(), int(value.Month()), min(preferredDay, daysInMonth))
	if value.Before(w.firstDate) {
		value = w.firstDate
	} else if value.After(w.lastDate) {
		value = w.lastDate
	}
	s.SetState(func() {
		s.mode = DatePickerModeDay
		s.handleMonthChanged(value)
		if s.isSelectable(value) {
			s.selectedDate = value
			w.onChanged(s.selectedDate)
		}
	})
}

func (s *calendarDatePickerState) handleDayChanged(value time.Time) {
	s.SetState(func() {
		s.selectedDate = value
		s.Widget().onChanged(s.selectedDate)
		switch ThemeOf(s.Context()).Platform {
		case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
			suffix := ""
			if s.delegate().IsSameDay(s.Widget().currentDate, s.selectedDate) {
				suffix = ", " + s.localizations.CurrentDateLabel()
			}
			s.announce(s.localizations.SelectedDateLabel() + " " + s.delegate().FormatFullDate(s.selectedDate, s.localizations) + suffix)
		}
	})
}

func (s *calendarDatePickerState) isSelectable(date time.Time) bool {
	p := s.Widget().o.SelectableDayPredicate
	return p == nil || p(date)
}

func (s *calendarDatePickerState) buildPicker() widgets.Widget {
	w := s.Widget()
	if s.mode == DatePickerModeDay {
		return newMonthPicker(monthPickerArgs{
			key: s.monthPickerKey, calendarDelegate: s.delegate(), initialMonth: s.currentMonthDate, currentDate: w.currentDate,
			firstDate: w.firstDate, lastDate: w.lastDate, selectedDate: s.selectedDate, onChanged: s.handleDayChanged,
			onDisplayedMonthChanged: s.handleMonthChanged, selectableDayPredicate: w.o.SelectableDayPredicate,
		})
	}
	return widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, subHeaderHeight, 0, 0), NewYearPicker(w.firstDate, w.lastDate, s.currentMonthDate, s.handleYearChanged, YearPickerOpts{
		Key: s.yearPickerKey, CurrentDate: w.currentDate, CalendarDelegate: s.delegate(),
	}))
}

func (s *calendarDatePickerState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	DebugCheckHasMaterialLocalizations(bc)
	widgets.DebugCheckHasDirectionality(bc)
	textScaleFactor := textScaleFactorOf(bc, calendarMaxTextScaleFactor)
	orientation := widgets.MediaQueryOrientationOf(bc)
	maxDayPickerHeight := maxDayPickerHeightM2
	if ThemeOf(bc).UseMaterial3 && orientation == widgets.OrientationPortrait {
		maxDayPickerHeight = maxDayPickerHeightM3
	}
	scaledMaxDayPickerHeight := float64(maxDayPickerHeight)
	if textScaleFactor > 1.3 {
		scaledMaxDayPickerHeight += (maxDayPickerRowCount + 1) * ((textScaleFactor - 1) * 8)
	}
	picker := widgets.NewSizedBox(s.buildPicker(), widgets.SizedBoxOpts{Height: foundation.Some(subHeaderHeight + scaledMaxDayPickerHeight)})
	// Deviation: the platform is assumed not to support announcements, so the
	// picker is always a live region (see announce).
	live := widgets.NewSemantics(picker, widgets.SemanticsOpts{
		Container: true, LiveRegion: foundation.Some(true),
		AccessibilityFocusBlockType: foundation.Some(semantics.AccessibilityFocusBlockTypeBlockNode),
		Label:                       foundation.Some(s.announcementText),
	})
	title := s.delegate().FormatMonthYear(s.currentMonthDate, s.localizations)
	return widgets.NewStack([]widgets.Widget{
		live,
		widgets.NewMediaQueryWithClampedTextScaling(newDatePickerModeToggleButton(s.mode, title, func() {
			if s.mode == DatePickerModeDay {
				s.handleModeChanged(DatePickerModeYear)
			} else {
				s.handleModeChanged(DatePickerModeDay)
			}
		}), widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: modeToggleButtonMaxScaleFactor}),
	})
}

// datePickerModeToggleButton is the button of the sub header that shows the
// month and the year, and switches between the day picker and the year picker.
//
// Flutter: material/calendar_date_picker.dart _DatePickerModeToggleButton
type datePickerModeToggleButton struct {
	widgets.StatefulWidget
	mode           DatePickerMode
	title          string
	onTitlePressed func()
}

func newDatePickerModeToggleButton(mode DatePickerMode, title string, onTitlePressed func()) *datePickerModeToggleButton {
	return &datePickerModeToggleButton{StatefulWidget: widgets.NewStatefulWidgetBase(), mode: mode, title: title, onTitlePressed: onTitlePressed}
}

func (w *datePickerModeToggleButton) CreateState() widgets.State {
	return &datePickerModeToggleButtonState{}
}

type datePickerModeToggleButtonState struct {
	widgets.StateOf[*datePickerModeToggleButton]
	widgets.TickerProvider
	controller *animation.AnimationController
}

func (s *datePickerModeToggleButtonState) InitState() {
	value := 0.0
	if s.Widget().mode == DatePickerModeYear {
		value = 0.5
	}
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Value: foundation.Some(value), UpperBound: foundation.Some(0.5), Duration: foundation.Some(200 * time.Millisecond)})
}

func (s *datePickerModeToggleButtonState) DidUpdateWidget(old *datePickerModeToggleButton) {
	w := s.Widget()
	if old.mode == w.mode {
		return
	}
	if w.mode == DatePickerModeYear {
		s.controller.Forward()
	} else {
		s.controller.Reverse()
	}
}

func (s *datePickerModeToggleButtonState) Dispose() { s.controller.Dispose() }

func (s *datePickerModeToggleButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	buttonTextStyle := firstNonNil(datePickerTheme.ToggleButtonTextStyle, defaults.ToggleButtonTextStyle)
	subHeaderForeground := firstOpt(datePickerTheme.SubHeaderForegroundColor, defaults.SubHeaderForegroundColor)
	var buttonTextColor foundation.Opt[ui.Color]
	if datePickerTheme.ToggleButtonTextStyle != nil && datePickerTheme.ToggleButtonTextStyle.Color.IsSome() {
		buttonTextColor = datePickerTheme.ToggleButtonTextStyle.Color
	} else if datePickerTheme.SubHeaderForegroundColor.IsSome() {
		buttonTextColor = datePickerTheme.SubHeaderForegroundColor
	} else if defaults.ToggleButtonTextStyle != nil {
		buttonTextColor = defaults.ToggleButtonTextStyle.Color
	}
	var style *painting.TextStyle
	if buttonTextStyle != nil {
		style = buttonTextStyle.Apply(painting.TextStyleApply{Color: buttonTextColor})
	}
	row := []widgets.Widget{
		widgets.NewFlexible(widgets.NewSemantics(
			widgets.NewSizedBox(
				NewInkWell(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 8), widgets.NewRow([]widgets.Widget{
					widgets.NewFlexible(widgets.NewText(w.title, widgets.TextOpts{Overflow: painting.TextOverflowEllipsis, Style: style})),
					widgets.NewRotationTransition(s.controller, widgets.NewIcon(Icons.ArrowDropDown(), widgets.IconOpts{Color: subHeaderForeground})),
				})), InkWellOpts{OnTap: w.onTitlePressed}),
				widgets.SizedBoxOpts{Height: foundation.Some(subHeaderHeight)}),
			widgets.SemanticsOpts{Label: foundation.Some(MaterialLocalizationsOf(bc).SelectYearSemanticsLabel()), Button: foundation.Some(true), Container: true})),
	}
	if w.mode == DatePickerModeDay {
		row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(monthNavButtonsWidth)}))
	}
	return widgets.NewSizedBox(
		widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(16, 0, 4, 0), widgets.NewRow(row)),
		widgets.SizedBoxOpts{Height: foundation.Some(subHeaderHeight)})
}

// monthPickerArgs are the parameters of monthPicker.
type monthPickerArgs struct {
	key                     foundation.Key
	initialMonth            time.Time
	currentDate             time.Time
	firstDate               time.Time
	lastDate                time.Time
	selectedDate            time.Time
	onChanged               func(time.Time)
	onDisplayedMonthChanged func(time.Time)
	selectableDayPredicate  SelectableDayPredicate
	calendarDelegate        CalendarDelegate
}

// monthPicker displays the days of a month and moves between months with a page
// view.
//
// Flutter: material/calendar_date_picker.dart _MonthPicker
type monthPicker struct {
	widgets.StatefulWidget
	a monthPickerArgs
}

func newMonthPicker(a monthPickerArgs) *monthPicker {
	if foundation.DebugMode {
		switch {
		case a.firstDate.After(a.lastDate):
			panic(errors.New("MonthPickerFirstDateAfterLastDate", "the first date of a month picker must not be after the last date"))
		case !a.selectedDate.IsZero() && (a.selectedDate.Before(a.firstDate) || a.selectedDate.After(a.lastDate)):
			panic(errors.New("MonthPickerSelectedDateOutOfRange", "the selected date of a month picker must be between the first and the last date", "selectedDate", a.selectedDate))
		}
	}
	return &monthPicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: a.key}), a: a}
}

func (w *monthPicker) CreateState() widgets.State { return &monthPickerState{} }

// monthPickerGridIntent types for the shortcuts of the day grid.
type monthPickerState struct {
	widgets.StateOf[*monthPicker]
	pageViewKey      *widgets.GlobalKey[widgets.State]
	announcementText string
	currentMonth     time.Time
	pageController   *widgets.PageController
	localizations    MaterialLocalizations
	shortcutMap      widgets.ShortcutMap
	actionMap        widgets.ActionMap
	dayGridFocus     *widgets.FocusNode
	focusedDay       time.Time
}

func (s *monthPickerState) delegate() CalendarDelegate { return s.Widget().a.calendarDelegate }

func (s *monthPickerState) InitState() {
	a := s.Widget().a
	s.pageViewKey = widgets.NewGlobalKey[widgets.State]()
	s.currentMonth = a.initialMonth
	s.pageController = widgets.NewPageController(widgets.PageControllerOpts{
		InitialPage: a.calendarDelegate.MonthDelta(a.firstDate, s.currentMonth)})
	arrow := func(k services.LogicalKeyboardKey, d widgets.TraversalDirection) widgets.ShortcutEntry {
		return widgets.ShortcutEntry{Activator: widgets.NewSingleActivator(k), Intent: widgets.DirectionalFocusIntent{Direction: d}}
	}
	s.shortcutMap = widgets.ShortcutMap{
		arrow(services.LogicalKeyboardKeyArrowLeft, widgets.TraversalDirectionLeft),
		arrow(services.LogicalKeyboardKeyArrowRight, widgets.TraversalDirectionRight),
		arrow(services.LogicalKeyboardKeyArrowDown, widgets.TraversalDirectionDown),
		arrow(services.LogicalKeyboardKeyArrowUp, widgets.TraversalDirectionUp),
	}
	next := widgets.NewCallbackAction(func(widgets.NextFocusIntent) any { s.handleGridNextFocus(); return nil })
	previous := widgets.NewCallbackAction(func(widgets.PreviousFocusIntent) any { s.handleGridPreviousFocus(); return nil })
	directional := widgets.NewCallbackAction(func(i widgets.DirectionalFocusIntent) any { s.handleDirectionFocus(i); return nil })
	s.actionMap = widgets.ActionMap{
		reflect.TypeFor[widgets.NextFocusIntent]():        next,
		reflect.TypeFor[widgets.PreviousFocusIntent]():    previous,
		reflect.TypeFor[widgets.DirectionalFocusIntent](): directional,
	}
	s.dayGridFocus = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "Day Grid"})
}

func (s *monthPickerState) DidChangeDependencies() {
	s.localizations = MaterialLocalizationsOf(s.Context())
}

func (s *monthPickerState) Dispose() {
	s.pageController.Dispose()
	s.dayGridFocus.Dispose()
}

func (s *monthPickerState) handleDateSelected(selected time.Time) {
	s.focusedDay = selected
	s.Widget().a.onChanged(selected)
}

func (s *monthPickerState) announce(message string) { s.announcementText = message }

func (s *monthPickerState) handleMonthPageChanged(monthPage int) {
	a := s.Widget().a
	s.SetState(func() {
		monthDate := a.calendarDelegate.AddMonthsToMonthDate(a.firstDate, monthPage)
		if !a.calendarDelegate.IsSameMonth(s.currentMonth, monthDate) {
			s.currentMonth = a.calendarDelegate.GetMonth(monthDate.Year(), int(monthDate.Month()))
			a.onDisplayedMonthChanged(s.currentMonth)
			if !s.focusedDay.IsZero() && !a.calendarDelegate.IsSameMonth(s.focusedDay, s.currentMonth) {
				// We have navigated to a new month with the grid focused, but the
				// focused day is not in this month. Choose a new one trying to keep
				// the same day of the month.
				s.focusedDay = s.focusableDayForMonth(s.currentMonth, s.focusedDay.Day())
			}
			s.announce(a.calendarDelegate.FormatMonthYear(s.currentMonth, s.localizations))
		}
	})
}

// focusableDayForMonth returns the day of the month to focus: the preferred
// day if it can be selected, otherwise the first day that can; the zero time
// when none can.
//
// Flutter: material/calendar_date_picker.dart _MonthPickerState._focusableDayForMonth
func (s *monthPickerState) focusableDayForMonth(month time.Time, preferredDay int) time.Time {
	d := s.delegate()
	daysInMonth := d.DaysInMonth(month.Year(), int(month.Month()))
	// Can we use the preferred day in this month?
	if preferredDay <= daysInMonth {
		newFocus := d.GetDay(month.Year(), int(month.Month()), preferredDay)
		if s.isSelectable(newFocus) {
			return newFocus
		}
	}
	// Start at the 1st and take the first selectable date.
	for day := 1; day <= daysInMonth; day++ {
		newFocus := d.GetDay(month.Year(), int(month.Month()), day)
		if s.isSelectable(newFocus) {
			return newFocus
		}
	}
	return time.Time{}
}

func (s *monthPickerState) handleNextMonth() {
	if !s.isDisplayingLastMonth() {
		s.pageController.NextPage(monthScrollDuration, animation.Curves.Ease)
	}
}

func (s *monthPickerState) handlePreviousMonth() {
	if !s.isDisplayingFirstMonth() {
		s.pageController.PreviousPage(monthScrollDuration, animation.Curves.Ease)
	}
}

// showMonth jumps or animates to the page of the month.
func (s *monthPickerState) showMonth(month time.Time, jump bool) {
	a := s.Widget().a
	monthPage := a.calendarDelegate.MonthDelta(a.firstDate, month)
	if jump {
		s.pageController.JumpToPage(monthPage)
	} else {
		s.pageController.AnimateToPage(monthPage, monthScrollDuration, animation.Curves.Ease)
	}
}

func (s *monthPickerState) isDisplayingFirstMonth() bool {
	a := s.Widget().a
	return !s.currentMonth.After(a.calendarDelegate.GetMonth(a.firstDate.Year(), int(a.firstDate.Month())))
}

func (s *monthPickerState) isDisplayingLastMonth() bool {
	a := s.Widget().a
	return !s.currentMonth.Before(a.calendarDelegate.GetMonth(a.lastDate.Year(), int(a.lastDate.Month())))
}

func (s *monthPickerState) handleGridFocusChange(focused bool) {
	a := s.Widget().a
	s.SetState(func() {
		if focused && s.focusedDay.IsZero() {
			switch {
			case a.calendarDelegate.IsSameMonth(a.selectedDate, s.currentMonth):
				s.focusedDay = a.selectedDate
			case a.calendarDelegate.IsSameMonth(a.currentDate, s.currentMonth):
				s.focusedDay = s.focusableDayForMonth(s.currentMonth, a.currentDate.Day())
			default:
				s.focusedDay = s.focusableDayForMonth(s.currentMonth, 1)
			}
		}
	})
}

func (s *monthPickerState) handleGridNextFocus() {
	s.dayGridFocus.RequestFocus()
	s.dayGridFocus.NextFocus()
}

func (s *monthPickerState) handleGridPreviousFocus() {
	s.dayGridFocus.RequestFocus()
	s.dayGridFocus.PreviousFocus()
}

func (s *monthPickerState) handleDirectionFocus(intent widgets.DirectionalFocusIntent) {
	s.SetState(func() {
		if next := s.nextDateInDirection(s.focusedDay, intent.Direction); !next.IsZero() {
			s.focusedDay = next
			if !s.delegate().IsSameMonth(s.focusedDay, s.currentMonth) {
				s.showMonth(s.focusedDay, false)
			}
		}
	})
}

// dayDirectionOffset is the number of days that a move in the direction goes.
//
// Flutter: material/calendar_date_picker.dart _MonthPickerState._dayDirectionOffset
func dayDirectionOffset(direction widgets.TraversalDirection, textDirection ui.TextDirection) int {
	if textDirection == ui.TextDirectionRTL {
		switch direction {
		case widgets.TraversalDirectionLeft:
			direction = widgets.TraversalDirectionRight
		case widgets.TraversalDirectionRight:
			direction = widgets.TraversalDirectionLeft
		}
	}
	switch direction {
	case widgets.TraversalDirectionUp:
		return -daysPerWeek
	case widgets.TraversalDirectionRight:
		return 1
	case widgets.TraversalDirectionDown:
		return daysPerWeek
	default:
		return -1
	}
}

// nextDateInDirection returns the first selectable date from the date in the
// direction, or the zero time when there is none in range.
//
// Flutter: material/calendar_date_picker.dart _MonthPickerState._nextDateInDirection
func (s *monthPickerState) nextDateInDirection(date time.Time, direction widgets.TraversalDirection) time.Time {
	a := s.Widget().a
	textDirection := widgets.DirectionalityOf(s.Context())
	nextDate := a.calendarDelegate.AddDaysToDate(date, dayDirectionOffset(direction, textDirection))
	for !nextDate.Before(a.firstDate) && !nextDate.After(a.lastDate) {
		if s.isSelectable(nextDate) {
			return nextDate
		}
		nextDate = a.calendarDelegate.AddDaysToDate(nextDate, dayDirectionOffset(direction, textDirection))
	}
	return time.Time{}
}

func (s *monthPickerState) isSelectable(date time.Time) bool {
	p := s.Widget().a.selectableDayPredicate
	return p == nil || p(date)
}

func (s *monthPickerState) buildItems(_ widgets.BuildContext, index int) widgets.Widget {
	a := s.Widget().a
	month := a.calendarDelegate.AddMonthsToMonthDate(a.firstDate, index)
	return newDayPicker(dayPickerArgs{
		key: foundation.NewValueKey(month.Unix()), calendarDelegate: a.calendarDelegate, selectedDate: a.selectedDate,
		currentDate: a.currentDate, onChanged: s.handleDateSelected, firstDate: a.firstDate, lastDate: a.lastDate,
		displayedMonth: month, selectableDayPredicate: a.selectableDayPredicate,
	})
}

func (s *monthPickerState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	subHeaderForeground := firstOpt(DatePickerThemeOf(bc).SubHeaderForegroundColor, DatePickerThemeDefaults(bc).SubHeaderForegroundColor)
	// Deviation: the platform is assumed not to support announcements, so the
	// month picker is always a live region (see announce).
	previousLabel, nextLabel := s.localizations.PreviousMonthTooltip(), s.localizations.NextMonthTooltip()
	first, last := s.isDisplayingFirstMonth(), s.isDisplayingLastMonth()
	chevron := func(icon widgets.IconData, atEnd bool, label string, onPressed func()) widgets.Widget {
		iconOpts := widgets.IconOpts{}
		tooltip := label
		if atEnd {
			iconOpts.SemanticLabel = foundation.Some(label)
			tooltip = ""
			onPressed = nil
		}
		return NewIconButton(onPressed, widgets.NewIcon(icon, iconOpts), IconButtonOpts{Color: subHeaderForeground, Tooltip: tooltip})
	}
	var focused time.Time
	if s.dayGridFocus.HasFocus() {
		focused = s.focusedDay
	}
	return widgets.NewSemantics(widgets.NewColumn([]widgets.Widget{
		widgets.NewSizedBox(widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(16, 0, 4, 0), widgets.NewRow([]widgets.Widget{
			widgets.NewSpacer(),
			chevron(Icons.ChevronLeft(), first, previousLabel, s.handlePreviousMonth),
			chevron(Icons.ChevronRight(), last, nextLabel, s.handleNextMonth),
		})), widgets.SizedBoxOpts{Height: foundation.Some(subHeaderHeight)}),
		widgets.NewExpanded(widgets.NewFocusableActionDetector(
			newFocusedDate(a.calendarDelegate, focused, NewMaterial(
				widgets.NewPageViewBuilder(s.buildItems, widgets.PageViewOpts{
					Key: s.pageViewKey, Controller: s.pageController, OnPageChanged: s.handleMonthPageChanged,
					ItemCount: foundation.Some(a.calendarDelegate.MonthDelta(a.firstDate, a.lastDate) + 1),
				}), MaterialOpts{Type: MaterialTypeTransparency})),
			widgets.FocusableActionDetectorOpts{
				Shortcuts: s.shortcutMap, Actions: s.actionMap, FocusNode: s.dayGridFocus, OnFocusChange: s.handleGridFocusChange,
			})),
	}), widgets.SemanticsOpts{
		Container: true, ExplicitChildNodes: true, LiveRegion: foundation.Some(true),
		AccessibilityFocusBlockType: foundation.Some(semantics.AccessibilityFocusBlockTypeBlockNode),
		Label:                       foundation.Some(s.announcementText),
	})
}

// focusedDate tells the day pickers which day has the focus.
//
// Flutter: material/calendar_date_picker.dart _FocusedDate
type focusedDate struct {
	widgets.InheritedWidget
	calendarDelegate CalendarDelegate
	date             time.Time
}

func newFocusedDate(d CalendarDelegate, date time.Time, child widgets.Widget) *focusedDate {
	return &focusedDate{InheritedWidget: widgets.NewInheritedWidgetBase(child), calendarDelegate: d, date: date}
}

func (w *focusedDate) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.calendarDelegate.IsSameDay(w.date, old.(*focusedDate).date)
}

// focusedDateMaybeOf returns the focused day below, or the zero time for none.
func focusedDateMaybeOf(bc widgets.BuildContext) time.Time {
	if f, ok := widgets.DependOn[*focusedDate](bc); ok {
		return f.date
	}
	return time.Time{}
}

// dayPickerArgs are the parameters of dayPicker.
type dayPickerArgs struct {
	key                    foundation.Key
	currentDate            time.Time
	displayedMonth         time.Time
	firstDate              time.Time
	lastDate               time.Time
	selectedDate           time.Time
	onChanged              func(time.Time)
	selectableDayPredicate SelectableDayPredicate
	calendarDelegate       CalendarDelegate
}

// dayPicker displays the days of a month as a grid.
//
// Flutter: material/calendar_date_picker.dart _DayPicker
type dayPicker struct {
	widgets.StatefulWidget
	a dayPickerArgs
}

func newDayPicker(a dayPickerArgs) *dayPicker {
	return &dayPicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: a.key}), a: a}
}

func (w *dayPicker) CreateState() widgets.State { return &dayPickerState{} }

type dayPickerState struct {
	widgets.StateOf[*dayPicker]
	dayFocusNodes []*widgets.FocusNode
}

func (s *dayPickerState) InitState() {
	a := s.Widget().a
	daysInMonth := a.calendarDelegate.DaysInMonth(a.displayedMonth.Year(), int(a.displayedMonth.Month()))
	s.dayFocusNodes = make([]*widgets.FocusNode, daysInMonth)
	for i := range s.dayFocusNodes {
		s.dayFocusNodes[i] = widgets.NewFocusNode(widgets.FocusNodeOpts{SkipTraversal: true, DebugLabel: "Day " + strconv.Itoa(i+1)})
	}
}

func (s *dayPickerState) DidChangeDependencies() {
	a := s.Widget().a
	focused := focusedDateMaybeOf(s.Context())
	// Request focus in the day of the month that is focused, if it is in this month.
	if !focused.IsZero() && a.calendarDelegate.IsSameMonth(a.displayedMonth, focused) {
		s.dayFocusNodes[focused.Day()-1].RequestFocus()
	}
}

func (s *dayPickerState) Dispose() {
	for _, n := range s.dayFocusNodes {
		n.Dispose()
	}
}

// dayHeaders builds the narrow weekday names, from the first day of the week.
func (s *dayPickerState) dayHeaders(headerStyle *painting.TextStyle, l MaterialLocalizations) []widgets.Widget {
	var result []widgets.Widget
	for i := l.FirstDayOfWeekIndex(); len(result) < daysPerWeek; i = (i + 1) % daysPerWeek {
		weekday := l.NarrowWeekdays()[i]
		result = append(result, widgets.NewExcludeSemantics(widgets.NewCenter(widgets.NewText(weekday, widgets.TextOpts{Style: headerStyle}))))
	}
	return result
}

func (s *dayPickerState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	localizations := MaterialLocalizationsOf(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	weekdayStyle := firstNonNil(datePickerTheme.WeekdayStyle, defaults.WeekdayStyle)
	isLandscape := widgets.MediaQueryOrientationOf(bc) == widgets.OrientationLandscape
	year, month := a.displayedMonth.Year(), int(a.displayedMonth.Month())
	daysInMonth := a.calendarDelegate.DaysInMonth(year, month)
	dayOffset := a.calendarDelegate.FirstDayOffset(year, month, localizations)
	dayItems := s.dayHeaders(weekdayStyle, localizations)
	// 1-based day of month, e.g. 1-31 for January, and 1-29 for February on a leap year.
	day := -dayOffset
	for day < daysInMonth {
		day++
		if day < 1 {
			dayItems = append(dayItems, widgets.NewSizedBoxShrink(nil))
			continue
		}
		dayToBuild := a.calendarDelegate.GetDay(year, month, day)
		isDisabled := dayToBuild.After(a.lastDate) || dayToBuild.Before(a.firstDate) ||
			(a.selectableDayPredicate != nil && !a.selectableDayPredicate(dayToBuild))
		dayItems = append(dayItems, newDayWidget(dayToBuild, dayWidgetArgs{
			isDisabled: isDisabled, isSelectedDay: a.calendarDelegate.IsSameDay(a.selectedDate, dayToBuild),
			isToday: a.calendarDelegate.IsSameDay(a.currentDate, dayToBuild), onChanged: a.onChanged,
			focusNode: s.dayFocusNodes[day-1], calendarDelegate: a.calendarDelegate,
		}))
	}
	horizontal := monthPickerHorizontalPaddingOther
	if ThemeOf(bc).UseMaterial3 && !isLandscape {
		horizontal = monthPickerHorizontalPaddingPortrait
	}
	maxScale := dayPickerGridPortraitMaxScaleFactor
	if isLandscape {
		maxScale = dayPickerGridLandscapeMaxScaleFactor
	}
	return widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, horizontal), widgets.NewMediaQueryWithClampedTextScaling(
		widgets.NewGridViewCustom(&dayPickerGridDelegate{bc: bc}, widgets.NewSliverChildListDelegate(dayItems, widgets.SliverChildListDelegateOpts{
			AddRepaintBoundaries: foundation.Some(false)}),
			widgets.GridViewOpts{Physics: widgets.NewClampingScrollPhysics(nil)}),
		widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxScale}))
}

// dayWidgetArgs are the parameters of dayWidget.
type dayWidgetArgs struct {
	isDisabled       bool
	isSelectedDay    bool
	isToday          bool
	onChanged        func(time.Time)
	focusNode        *widgets.FocusNode
	calendarDelegate CalendarDelegate
}

// dayWidget is a day of the month in the grid.
//
// Flutter: material/calendar_date_picker.dart _Day
type dayWidget struct {
	widgets.StatefulWidget
	day time.Time
	a   dayWidgetArgs
}

func newDayWidget(day time.Time, a dayWidgetArgs) *dayWidget {
	return &dayWidget{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: foundation.NewValueKey(day.Unix())}), day: day, a: a}
}

func (w *dayWidget) CreateState() widgets.State { return &dayWidgetState{} }

type dayWidgetState struct {
	widgets.StateOf[*dayWidget]
	statesController *widgets.WidgetStatesController
}

func (s *dayWidgetState) InitState() { s.statesController = widgets.NewWidgetStatesController() }

func (s *dayWidgetState) Dispose() { s.statesController.Dispose() }

func (s *dayWidgetState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	a := w.a
	defaults := DatePickerThemeDefaults(bc)
	datePickerTheme := DatePickerThemeOf(bc)
	dayStyle := firstNonNil(datePickerTheme.DayStyle, defaults.DayStyle)
	localizations := MaterialLocalizationsOf(bc)
	suffix := ""
	if a.isToday {
		suffix = ", " + localizations.CurrentDateLabel()
	}
	var states widgets.WidgetStates
	if a.isDisabled {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if a.isSelectedDay {
		states = states.With(widgets.WidgetStateSelected)
	}
	s.statesController.SetValue(states)

	todayOrDay := func(today, day func(DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) func(DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
		if a.isToday {
			return today
		}
		return day
	}
	dayForeground := datePickerColor(datePickerTheme, defaults, todayOrDay(
		func(t DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
			return t.TodayForegroundColor
		},
		func(t DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
			return t.DayForegroundColor
		}), states)
	dayBackground := datePickerColor(datePickerTheme, defaults, todayOrDay(
		func(t DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
			return t.TodayBackgroundColor
		},
		func(t DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
			return t.DayBackgroundColor
		}), states)
	dayOverlay := widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		return datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
			return t.DayOverlayColor
		}, states)
	})
	dayShape := datePickerShape(datePickerTheme, defaults, func(t DatePickerThemeData) widgets.WidgetStateProperty[painting.OutlinedBorder] {
		return t.DayShape
	}, states)

	hasCustomBorderColor := datePickerTheme.TodayBorder.IsSome() && datePickerTheme.TodayBorder.Value().Color.Alpha() != 0
	var todayBorderSide painting.BorderSide
	if hasCustomBorderColor {
		todayBorderSide = datePickerTheme.TodayBorder.Value()
	} else {
		todayBorderSide = firstOpt(datePickerTheme.TodayBorder, defaults.TodayBorder).Value().CopyWith(painting.BorderSideCopy{Color: dayForeground})
	}
	shape := dayShape
	if a.isToday {
		shape = dayShape.CopyWithSide(todayBorderSide)
	}
	decoration := painting.NewShapeDecoration(shape, painting.ShapeDecorationOpts{Color: dayBackground})

	var textStyle *painting.TextStyle
	if dayStyle != nil {
		textStyle = dayStyle.Apply(painting.TextStyleApply{Color: dayForeground})
	}
	var dayWidget widgets.Widget = NewInk(
		widgets.NewCenter(widgets.NewText(localizations.FormatDecimal(w.day.Day()), widgets.TextOpts{Style: textStyle})),
		InkOpts{Decoration: decoration})
	if ThemeOf(bc).UseMaterial3 && widgets.MediaQueryOrientationOf(bc) == widgets.OrientationPortrait {
		dayWidget = widgets.NewPadding(painting.NewEdgeInsetsAll(4), dayWidget)
	}
	dayWidget = widgets.NewSemantics(dayWidget, widgets.SemanticsOpts{
		Label:            foundation.Some(localizations.FormatDecimal(w.day.Day()) + ", " + a.calendarDelegate.FormatFullDate(w.day, localizations) + suffix),
		Button:           foundation.Some(true),
		Selected:         foundation.Some(a.isSelectedDay),
		Enabled:          foundation.Some(!a.isDisabled),
		ExcludeSemantics: true,
	})
	if !a.isDisabled {
		dayWidget = NewInkResponse(dayWidget, InkResponseOpts{
			FocusNode: a.focusNode, OnTap: func() { a.onChanged(w.day) }, StatesController: s.statesController,
			OverlayColor: dayOverlay, CustomBorder: dayShape, ContainedInkWell: true,
		})
	}
	return dayWidget
}

// dayPickerGridDelegate lays the days out in a grid of seven columns.
//
// Flutter: material/calendar_date_picker.dart _DayPickerGridDelegate
type dayPickerGridDelegate struct{ bc widgets.BuildContext }

func (d *dayPickerGridDelegate) GetLayout(constraints rendering.SliverConstraints) rendering.SliverGridLayout {
	textScaleFactor := textScaleFactorOf(d.bc, 3.0)
	rowHeight := dayPickerRowHeightM2
	if ThemeOf(d.bc).UseMaterial3 && widgets.MediaQueryOrientationOf(d.bc) == widgets.OrientationPortrait {
		rowHeight = dayPickerRowHeightM3
	}
	scaledRowHeight := float64(rowHeight)
	if textScaleFactor > 1.3 {
		scaledRowHeight = (textScaleFactor-1)*30 + rowHeight
	}
	const columnCount = daysPerWeek
	tileWidth := constraints.CrossAxisExtent / columnCount
	tileHeight := math.Min(scaledRowHeight, constraints.ViewportMainAxisExtent/(maxDayPickerRowCount+1))
	return rendering.NewSliverGridRegularTileLayout(columnCount, tileHeight, tileWidth, tileHeight, tileWidth,
		painting.AxisDirectionIsReversed(constraints.CrossAxisDirection))
}

func (d *dayPickerGridDelegate) ShouldRelayout(rendering.SliverGridDelegate) bool { return false }

// YearPickerOpts are the optional parameters of NewYearPicker.
//
// Flutter: material/calendar_date_picker.dart YearPicker.new
type YearPickerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CurrentDate is the date that is shown as the current year; the default is
	// the current date.
	CurrentDate time.Time
	// DragStartBehavior determines the way that drag start behavior is handled;
	// the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// YearPicker displays a scrollable list of years to allow picking a year.
//
// Flutter: material/calendar_date_picker.dart YearPicker
type YearPicker struct {
	widgets.StatefulWidget
	currentDate  time.Time
	firstDate    time.Time
	lastDate     time.Time
	selectedDate time.Time
	onChanged    func(time.Time)
	o            YearPickerOpts
}

// NewYearPicker creates a year picker. The selected date is the zero time for
// none. In debug builds it panics with the tag YearPickerFirstDateAfterLastDate
// when the first date is after the last one.
//
// Flutter: material/calendar_date_picker.dart YearPicker.new
func NewYearPicker(firstDate, lastDate, selectedDate time.Time, onChanged func(time.Time), opts ...YearPickerOpts) *YearPicker {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	if foundation.DebugMode && firstDate.After(lastDate) {
		panic(errors.New("YearPickerFirstDateAfterLastDate", "the first date of a year picker must not be after the last date", "firstDate", firstDate, "lastDate", lastDate))
	}
	current := o.CurrentDate
	if current.IsZero() {
		current = time.Now()
	}
	return &YearPicker{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		currentDate:    o.CalendarDelegate.DateOnly(current), firstDate: firstDate, lastDate: lastDate, selectedDate: selectedDate,
		onChanged: onChanged, o: o,
	}
}

// CreateState creates the state.
//
// Flutter: material/calendar_date_picker.dart YearPicker.createState
func (w *YearPicker) CreateState() widgets.State { return &yearPickerState{} }

const yearPickerMinYears = 18

type yearPickerState struct {
	widgets.StateOf[*YearPicker]
	scrollController *widgets.ScrollControllerBase
	statesController *widgets.WidgetStatesController
}

func (s *yearPickerState) InitState() {
	w := s.Widget()
	start := w.firstDate
	if !w.selectedDate.IsZero() {
		start = w.selectedDate
	}
	s.statesController = widgets.NewWidgetStatesController()
	s.scrollController = widgets.NewScrollController(widgets.ScrollControllerOpts{InitialScrollOffset: s.scrollOffsetForYear(start)})
}

func (s *yearPickerState) Dispose() {
	s.scrollController.Dispose()
	s.statesController.Dispose()
}

func (s *yearPickerState) DidUpdateWidget(old *YearPicker) {
	w := s.Widget()
	if !w.selectedDate.Equal(old.selectedDate) && !w.selectedDate.IsZero() {
		s.scrollController.JumpTo(s.scrollOffsetForYear(w.selectedDate))
	}
}

func (s *yearPickerState) itemCount() int {
	w := s.Widget()
	return w.lastDate.Year() - w.firstDate.Year() + 1
}

func (s *yearPickerState) scrollOffsetForYear(date time.Time) float64 {
	w := s.Widget()
	initialYearIndex := date.Year() - w.firstDate.Year()
	initialYearRow := initialYearIndex / yearPickerColumnCount
	// Move the offset down by 2 rows to approximately center it.
	centeredYearRow := initialYearRow - 2
	if s.itemCount() < yearPickerMinYears {
		return 0
	}
	return float64(centeredYearRow) * yearPickerRowHeight
}

func (s *yearPickerState) buildYearItem(bc widgets.BuildContext, index int) widgets.Widget {
	w := s.Widget()
	datePickerTheme := DatePickerThemeOf(bc)
	defaults := DatePickerThemeDefaults(bc)
	textScaleFactor := textScaleFactorOf(bc, 3.0)
	// Backfill the _YearPicker with disabled years if necessary.
	offset := 0
	if s.itemCount() < yearPickerMinYears {
		offset = (yearPickerMinYears - s.itemCount()) / 2
	}
	year := w.firstDate.Year() + index - offset
	isSelected := !w.selectedDate.IsZero() && year == w.selectedDate.Year()
	isCurrentYear := year == w.currentDate.Year()
	isDisabled := year < w.firstDate.Year() || year > w.lastDate.Year()
	decorationHeight := 36.0 * textScaleFactor
	decorationWidth := 72.0 * textScaleFactor

	var states widgets.WidgetStates
	if isDisabled {
		states = states.With(widgets.WidgetStateDisabled)
	}
	if isSelected {
		states = states.With(widgets.WidgetStateSelected)
	}
	type colorProp = widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	textColor := datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp {
		if isCurrentYear {
			return t.TodayForegroundColor
		}
		return t.YearForegroundColor
	}, states)
	background := datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp {
		if isCurrentYear {
			return t.TodayBackgroundColor
		}
		return t.YearBackgroundColor
	}, states)
	overlay := widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		return datePickerColor(datePickerTheme, defaults, func(t DatePickerThemeData) colorProp { return t.YearOverlayColor }, states)
	})
	yearShape := datePickerShape(datePickerTheme, defaults, func(t DatePickerThemeData) widgets.WidgetStateProperty[painting.OutlinedBorder] {
		return t.YearShape
	}, states)

	shape := yearShape
	if isCurrentYear {
		if side, ok := firstOpt(datePickerTheme.TodayBorder, defaults.TodayBorder).Get(); ok {
			shape = yearShape.CopyWithSide(side.CopyWith(painting.BorderSideCopy{Color: textColor}))
		}
	}
	decoration := painting.NewShapeDecoration(shape, painting.ShapeDecorationOpts{Color: background})
	var itemStyle *painting.TextStyle
	if base := firstNonNil(datePickerTheme.YearStyle, defaults.YearStyle); base != nil {
		itemStyle = base.Apply(painting.TextStyleApply{Color: textColor})
	}
	localizations := MaterialLocalizationsOf(bc)
	var yearItem widgets.Widget = widgets.NewCenter(widgets.NewContainer(
		widgets.NewSemantics(widgets.NewText(w.o.CalendarDelegate.FormatYear(year, localizations), widgets.TextOpts{Style: itemStyle}),
			widgets.SemanticsOpts{Selected: foundation.Some(isSelected), Enabled: foundation.Some(!isDisabled), Button: foundation.Some(true)}),
		widgets.ContainerOpts{Decoration: decoration, Height: foundation.Some(decorationHeight), Width: foundation.Some(decorationWidth),
			Alignment: painting.AlignmentCenter}))
	if !isDisabled {
		month := time.January
		if !w.selectedDate.IsZero() {
			month = w.selectedDate.Month()
		}
		d := w.o.CalendarDelegate
		date := d.GetMonth(year, int(month))
		if date.Before(d.GetMonth(w.firstDate.Year(), int(w.firstDate.Month()))) {
			// Ignore firstDate.day because we're just working in years and months here.
			date = d.GetMonth(year, int(w.firstDate.Month()))
		} else if date.After(w.lastDate) {
			// No need to ignore the day here because it can only be the 1st.
			date = d.GetMonth(year, int(w.lastDate.Month()))
		}
		s.statesController.SetValue(states)
		yearItem = NewInkWell(yearItem, InkWellOpts{
			Key: foundation.NewValueKey(year), OnTap: func() { w.onChanged(date) },
			StatesController: s.statesController, OverlayColor: overlay,
		})
	}
	return yearItem
}

func (s *yearPickerState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewColumn([]widgets.Widget{
		NewDivider(DividerOpts{Height: foundation.Some(16.0)}),
		widgets.NewExpanded(NewMaterial(widgets.NewGridViewBuilder(&yearPickerGridDelegate{bc: bc}, s.buildYearItem, widgets.GridViewOpts{
			Controller: s.scrollController, DragStartBehavior: w.o.DragStartBehavior, ItemCount: foundation.Some(max(s.itemCount(), yearPickerMinYears)),
			Padding: painting.NewEdgeInsetsSymmetric(0, yearPickerPadding),
		}), MaterialOpts{Type: MaterialTypeTransparency})),
		NewDivider(DividerOpts{Height: foundation.Some(16.0)}),
	})
}

// yearPickerGridDelegate lays the years out in a grid of three columns.
//
// Flutter: material/calendar_date_picker.dart _YearPickerGridDelegate
type yearPickerGridDelegate struct{ bc widgets.BuildContext }

func (d *yearPickerGridDelegate) GetLayout(constraints rendering.SliverConstraints) rendering.SliverGridLayout {
	textScaleFactor := textScaleFactorOf(d.bc, 3.0)
	columnCount := yearPickerColumnCount
	if textScaleFactor > 1.65 {
		columnCount--
	}
	tileWidth := math.Max((constraints.CrossAxisExtent-float64(columnCount-1)*yearPickerRowSpacing)/float64(columnCount), 0)
	rowHeight := float64(yearPickerRowHeight)
	if textScaleFactor > 1 {
		rowHeight += (textScaleFactor - 1) * 9
	}
	return rendering.NewSliverGridRegularTileLayout(columnCount, rowHeight, tileWidth+yearPickerRowSpacing, rowHeight, tileWidth,
		painting.AxisDirectionIsReversed(constraints.CrossAxisDirection))
}

func (d *yearPickerGridDelegate) ShouldRelayout(rendering.SliverGridDelegate) bool { return false }
