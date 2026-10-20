// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	timePickerDialogSizeAnimationDuration = 200 * time.Millisecond
	timePickerDialAnimateDuration         = 200 * time.Millisecond
	twoPi                                 = 2 * math.Pi
	timePickerHeaderLandscapeWidth        = 216.0
	timePickerInnerDialOffset             = 28.0
	timePickerDialMinRadius               = 50.0
	timePickerDialPadding                 = 28.0
)

// TimePickerEntryMode describes how the time picker dialog starts, and how the
// user can change it.
//
// Flutter: material/time_picker.dart TimePickerEntryMode
type TimePickerEntryMode int

const (
	// TimePickerEntryModeUnspecified is the zero value: the default (dial).
	//
	// Flutter: no counterpart
	TimePickerEntryModeUnspecified TimePickerEntryMode = iota
	// TimePickerEntryModeDial is a dial to pick the time; the user can switch to
	// the input mode.
	//
	// Flutter: material/time_picker.dart TimePickerEntryMode.dial
	TimePickerEntryModeDial
	// TimePickerEntryModeInput is text fields to type the time; the user can switch
	// to the dial mode.
	//
	// Flutter: material/time_picker.dart TimePickerEntryMode.input
	TimePickerEntryModeInput
	// TimePickerEntryModeDialOnly is the dial, without a way to switch to the input
	// mode.
	//
	// Flutter: material/time_picker.dart TimePickerEntryMode.dialOnly
	TimePickerEntryModeDialOnly
	// TimePickerEntryModeInputOnly is the text fields, without a way to switch to
	// the dial mode.
	//
	// Flutter: material/time_picker.dart TimePickerEntryMode.inputOnly
	TimePickerEntryModeInputOnly
)

// Name returns the Dart enum name.
//
// Flutter: material/time_picker.dart TimePickerEntryMode.name
func (m TimePickerEntryMode) Name() string {
	return enumName(int(m), "dial", "input", "dialOnly", "inputOnly")
}

// String returns the name Dart prints, such as "TimePickerEntryMode.input".
//
// Flutter: material/time_picker.dart TimePickerEntryMode.toString
func (m TimePickerEntryMode) String() string { return "TimePickerEntryMode." + m.Name() }

// isDial is whether the mode shows the dial.
func (m TimePickerEntryMode) isDial() bool {
	return m == TimePickerEntryModeDial || m == TimePickerEntryModeDialOnly || m == TimePickerEntryModeUnspecified
}

// hourMinuteMode is which of the hour and the minute the dial selects.
//
// Flutter: material/time_picker.dart _HourMinuteMode
type hourMinuteMode int

const (
	hourMinuteModeHour hourMinuteMode = iota
	hourMinuteModeMinute
)

// hourDialType is the kind of dial of the hours.
//
// Flutter: material/time_picker.dart _HourDialType
type hourDialType int

const (
	hourDialTwentyFourHour hourDialType = iota
	hourDialTwentyFourHourDoubleRing
	hourDialTwelveHour
)

// timePickerAspect is an aspect of the time picker model that a widget depends
// on.
//
// Flutter: material/time_picker.dart _TimePickerAspect
type timePickerAspect int

const (
	aspectUse24HourFormat timePickerAspect = iota
	aspectUseMaterial3
	aspectEntryMode
	aspectHourMinuteMode
	aspectOnHourMinuteModeChanged
	aspectOnHourDoubleTapped
	aspectOnMinuteDoubleTapped
	aspectHourDialType
	aspectSelectedTime
	aspectOnSelectedTimeChanged
	aspectOrientation
	aspectTheme
	aspectDefaultTheme
)

// timePickerModel is the inherited model that gives the parts of the time picker
// the state of the picker.
//
// Flutter: material/time_picker.dart _TimePickerModel
type timePickerModel struct {
	widgets.InheritedModel
	entryMode               TimePickerEntryMode
	hourMinuteMode          hourMinuteMode
	onHourMinuteModeChanged func(hourMinuteMode)
	onHourDoubleTapped      func()
	onMinuteDoubleTapped    func()
	selectedTime            TimeOfDay
	onSelectedTimeChanged   func(TimeOfDay)
	use24HourFormat         bool
	useMaterial3            bool
	hourDialType            hourDialType
	orientation             widgets.Orientation
	theme                   TimePickerThemeData
	defaultTheme            *timePickerDefaults
}

func (m *timePickerModel) UpdateShouldNotify(old widgets.Widget) bool {
	return m.changedAspects(old.(*timePickerModel), nil)
}

func (m *timePickerModel) UpdateShouldNotifyDependent(old widgets.Widget, deps *widgets.AspectSet) bool {
	return m.changedAspects(old.(*timePickerModel), deps)
}

// changedAspects is whether an aspect that the dependencies include (any
// aspect for nil dependencies) is different from the one of the old model.
//
// Deviation: Dart compares the callbacks too, which are the tear-offs of
// methods of the state, and so are always equal; Go cannot compare functions,
// so they are not compared (SPEC §12.10).
func (m *timePickerModel) changedAspects(o *timePickerModel, deps *widgets.AspectSet) bool {
	changed := func(aspect timePickerAspect, differs bool) bool {
		return differs && (deps == nil || deps.Contains(aspect))
	}
	return changed(aspectUse24HourFormat, m.use24HourFormat != o.use24HourFormat) ||
		changed(aspectUseMaterial3, m.useMaterial3 != o.useMaterial3) ||
		changed(aspectEntryMode, m.entryMode != o.entryMode) ||
		changed(aspectHourMinuteMode, m.hourMinuteMode != o.hourMinuteMode) ||
		changed(aspectHourDialType, m.hourDialType != o.hourDialType) ||
		changed(aspectSelectedTime, m.selectedTime != o.selectedTime) ||
		changed(aspectOrientation, m.orientation != o.orientation) ||
		changed(aspectTheme, !m.theme.Equal(o.theme)) ||
		changed(aspectDefaultTheme, m.defaultTheme.source != o.defaultTheme.source && !m.defaultTheme.source.Equal(o.defaultTheme.source))
}

// timePickerModelOf is the model for the aspect.
func timePickerModelOf(bc widgets.BuildContext, aspect timePickerAspect) *timePickerModel {
	m, _ := widgets.DependOnModel[*timePickerModel](bc, aspect)
	return m
}

func timePickerEntryModeOf(bc widgets.BuildContext) TimePickerEntryMode {
	return timePickerModelOf(bc, aspectEntryMode).entryMode
}

func timePickerHourMinuteModeOf(bc widgets.BuildContext) hourMinuteMode {
	return timePickerModelOf(bc, aspectHourMinuteMode).hourMinuteMode
}

func timePickerSelectedTimeOf(bc widgets.BuildContext) TimeOfDay {
	return timePickerModelOf(bc, aspectSelectedTime).selectedTime
}

func timePickerUse24HourFormatOf(bc widgets.BuildContext) bool {
	return timePickerModelOf(bc, aspectUse24HourFormat).use24HourFormat
}

func timePickerUseMaterial3Of(bc widgets.BuildContext) bool {
	return timePickerModelOf(bc, aspectUseMaterial3).useMaterial3
}

func timePickerHourDialTypeOf(bc widgets.BuildContext) hourDialType {
	return timePickerModelOf(bc, aspectHourDialType).hourDialType
}

func timePickerOrientationOf(bc widgets.BuildContext) widgets.Orientation {
	return timePickerModelOf(bc, aspectOrientation).orientation
}

func timePickerThemeOf(bc widgets.BuildContext) TimePickerThemeData {
	return timePickerModelOf(bc, aspectTheme).theme
}

func timePickerDefaultThemeOf(bc widgets.BuildContext) *timePickerDefaults {
	return timePickerModelOf(bc, aspectDefaultTheme).defaultTheme
}

func setTimePickerSelectedTime(bc widgets.BuildContext, value TimeOfDay) {
	timePickerModelOf(bc, aspectOnSelectedTimeChanged).onSelectedTimeChanged(value)
}

func setTimePickerHourMinuteMode(bc widgets.BuildContext, value hourMinuteMode) {
	timePickerModelOf(bc, aspectOnHourMinuteModeChanged).onHourMinuteModeChanged(value)
}

// debugDialTimePickerEntryMode panics with the tag TimePickerNotInDialMode when
// the time picker is not in a dial mode.
//
// Flutter: material/time_picker.dart _debugDialTimePickerEntryMode
func debugDialTimePickerEntryMode(bc widgets.BuildContext) {
	if foundation.DebugMode && !timePickerEntryModeOf(bc).isDial() {
		panic(errors.New("TimePickerNotInDialMode", "this part of the time picker is only for the dial entry modes"))
	}
}

// timePickerHeader is the header of the dial modes: the help text, the hour,
// the minute and the day period.
//
// Flutter: material/time_picker.dart _DialTimePickerHeader
type dialTimePickerHeader struct {
	widgets.StatelessWidget
	helpText string
}

func newDialTimePickerHeader(helpText string) *dialTimePickerHeader {
	return &dialTimePickerHeader{StatelessWidget: widgets.NewStatelessWidgetBase(), helpText: helpText}
}

func (w *dialTimePickerHeader) Build(bc widgets.BuildContext) widgets.Widget {
	debugDialTimePickerEntryMode(bc)
	l := MaterialLocalizationsOf(bc)
	timeOfDayFormat := l.TimeOfDayFormat(timePickerUse24HourFormatOf(bc))
	defaultTheme := timePickerDefaultThemeOf(bc)
	orientation := timePickerOrientationOf(bc)
	dayPeriodHeight := defaultTheme.dayPeriodLandscapeSize.Height
	var minInteractiveVerticalPadding float64
	if orientation == widgets.OrientationPortrait {
		dayPeriodHeight = defaultTheme.dayPeriodPortraitSize.Height
		minInteractiveVerticalPadding = math.Max(0, 2*MinInteractiveDimension-dayPeriodHeight)
	} else {
		minInteractiveVerticalPadding = math.Max(0, MinInteractiveDimension-dayPeriodHeight)
	}
	hourDialType := timePickerHourDialTypeOf(bc)
	helpStyle := firstNonNil(timePickerThemeOf(bc).HelpTextStyle, defaultTheme.helpTextStyle)
	hourMinuteRow := func() *widgets.Row {
		return widgets.NewRow([]widgets.Widget{
			widgets.NewExpanded(newDialHourControl()),
			newTimeSelectorSeparator(timeOfDayFormat),
			widgets.NewExpanded(newDialMinuteControl()),
		}, widgets.RowOpts{TextDirection: ui.TextDirectionLTR})
	}
	var header widgets.Widget
	if orientation == widgets.OrientationPortrait {
		bottom := 24.0
		if timePickerUseMaterial3Of(bc) {
			bottom = 20
		}
		direction := ui.TextDirectionLTR
		if timeOfDayFormat == TimeOfDayFormatASpaceHColonMM {
			direction = ui.TextDirectionRTL
		}
		row := []widgets.Widget{widgets.NewExpanded(hourMinuteRow())}
		if hourDialType == hourDialTwelveHour {
			row = append(row, newDayPeriodControl(nil))
		}
		header = widgets.NewColumn([]widgets.Widget{
			widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, 0, bottom-minInteractiveVerticalPadding/2),
				widgets.NewText(w.helpText, widgets.TextOpts{Style: helpStyle})),
			widgets.NewRow(row, widgets.RowOpts{TextDirection: direction, Spacing: 12}),
		}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart})
	} else {
		verticalDirection := painting.VerticalDirectionDown
		if timeOfDayFormat == TimeOfDayFormatASpaceHColonMM {
			verticalDirection = painting.VerticalDirectionUp
		}
		children := []widgets.Widget{hourMinuteRow()}
		if hourDialType == hourDialTwelveHour {
			children = append(children, newDayPeriodControl(nil))
		}
		header = widgets.NewSizedBox(widgets.NewStack([]widgets.Widget{
			widgets.NewText(w.helpText, widgets.TextOpts{Style: helpStyle}),
			widgets.NewColumn(children, widgets.ColumnOpts{
				VerticalDirection: verticalDirection, MainAxisAlignment: rendering.MainAxisAlignmentCenter,
				CrossAxisAlignment: rendering.CrossAxisAlignmentStart, Spacing: math.Max(0, 16-minInteractiveVerticalPadding/2),
			}),
		}), widgets.SizedBoxOpts{Width: foundation.Some(timePickerHeaderLandscapeWidth)})
	}
	return widgets.NewSemantics(header, widgets.SemanticsOpts{
		Label: foundation.Some(l.FormatTimeOfDay(timePickerSelectedTimeOf(bc), widgets.MediaQueryAlwaysUse24HourFormatOf(bc))),
	})
}

// dialTimeSelectorControl is the hour or the minute in the header of the dial.
//
// Flutter: material/time_picker.dart _DialTimeSelectorControl
type dialTimeSelectorControl struct {
	widgets.StatelessWidget
	text        string
	onTap       func()
	onDoubleTap func()
	isSelected  bool
}

func (w *dialTimeSelectorControl) Build(bc widgets.BuildContext) widgets.Widget {
	debugDialTimePickerEntryMode(bc)
	timePickerTheme := timePickerThemeOf(bc)
	defaultTheme := timePickerDefaultThemeOf(bc)
	var states widgets.WidgetStates
	if w.isSelected {
		states = states.With(widgets.WidgetStateSelected)
	}
	backgroundColor := timePickerThemeColor(timePickerTheme.HourMinuteColor, timePickerTheme.HourMinuteStateColor, defaultTheme.hourMinuteColor, states)
	shape := firstNonNil(timePickerTheme.HourMinuteShape, defaultTheme.hourMinuteShape)
	effectiveTextColor := timePickerThemeColor(timePickerTheme.HourMinuteTextColor, timePickerTheme.HourMinuteTextStateColor, defaultTheme.hourMinuteTextColor, states)
	effectiveStyle := firstNonNil(timePickerTheme.HourMinuteTextStyle, defaultTheme.hourMinuteTextStyle).
		CopyWith(painting.TextStyleCopy{Color: foundation.Some(effectiveTextColor)})
	var onDoubleTap func()
	if w.isSelected {
		onDoubleTap = w.onDoubleTap
	}
	return widgets.NewSizedBox(NewMaterial(
		NewInkWell(widgets.NewCenter(widgets.NewText(w.text, widgets.TextOpts{Style: effectiveStyle, TextScaler: painting.TextScalerNoScaling})),
			InkWellOpts{OnTap: w.onTap, OnDoubleTap: onDoubleTap}),
		MaterialOpts{Color: foundation.Some(backgroundColor), ClipBehavior: ui.ClipAntiAlias, Shape: shape}),
		widgets.SizedBoxOpts{Height: foundation.Some(defaultTheme.hourMinuteSize.Height)})
}

// dialHourControl is the hour in the header of the dial.
//
// Flutter: material/time_picker.dart _DialHourControl
type dialHourControl struct{ widgets.StatelessWidget }

func newDialHourControl() *dialHourControl {
	return &dialHourControl{StatelessWidget: widgets.NewStatelessWidgetBase()}
}

func (w *dialHourControl) Build(bc widgets.BuildContext) widgets.Widget {
	debugDialTimePickerEntryMode(bc)
	widgets.DebugCheckHasMediaQuery(bc)
	alwaysUse24HourFormat := widgets.MediaQueryAlwaysUse24HourFormatOf(bc)
	selectedTime := timePickerSelectedTimeOf(bc)
	l := MaterialLocalizationsOf(bc)
	formattedHour := l.FormatHour(selectedTime, timePickerUse24HourFormatOf(bc))
	dialType := timePickerHourDialTypeOf(bc)
	hoursFromSelected := func(hoursToAdd int) TimeOfDay {
		if dialType == hourDialTwelveHour {
			periodOffset := selectedTime.PeriodOffset()
			hours := selectedTime.HourOfPeriod()
			return selectedTime.Replacing(TimeOfDayOpts{Hour: foundation.Some(periodOffset + mod(hours+hoursToAdd, HoursPerPeriod))})
		}
		return selectedTime.Replacing(TimeOfDayOpts{Hour: foundation.Some(mod(selectedTime.Hour+hoursToAdd, HoursPerDay))})
	}
	nextHour := hoursFromSelected(1)
	previousHour := hoursFromSelected(-1)
	return widgets.NewSemantics(newDialTimeSelectorControlWidget(dialTimeSelectorControl{
		isSelected:  timePickerHourMinuteModeOf(bc) == hourMinuteModeHour,
		text:        formattedHour,
		onTap:       func() { setTimePickerHourMinuteMode(bc, hourMinuteModeHour) },
		onDoubleTap: timePickerModelOf(bc, aspectOnHourDoubleTapped).onHourDoubleTapped,
	}), widgets.SemanticsOpts{
		Value:            foundation.Some(l.TimePickerHourModeAnnouncement() + " " + formattedHour),
		ExcludeSemantics: true,
		IncreasedValue:   foundation.Some(l.FormatHour(nextHour, alwaysUse24HourFormat)),
		OnIncrease:       func() { setTimePickerSelectedTime(bc, nextHour) },
		DecreasedValue:   foundation.Some(l.FormatHour(previousHour, alwaysUse24HourFormat)),
		OnDecrease:       func() { setTimePickerSelectedTime(bc, previousHour) },
	})
}

func newDialTimeSelectorControlWidget(c dialTimeSelectorControl) *dialTimeSelectorControl {
	c.StatelessWidget = widgets.NewStatelessWidgetBase()
	return &c
}

// timeSelectorSeparator is the colon between the hour and the minute.
//
// Flutter: material/time_picker.dart _TimeSelectorSeparator
type timeSelectorSeparator struct {
	widgets.StatelessWidget
	timeOfDayFormat TimeOfDayFormat
}

func newTimeSelectorSeparator(f TimeOfDayFormat) *timeSelectorSeparator {
	return &timeSelectorSeparator{StatelessWidget: widgets.NewStatelessWidgetBase(), timeOfDayFormat: f}
}

func (w *timeSelectorSeparator) value() string {
	switch w.timeOfDayFormat {
	case TimeOfDayFormatHHDotMM:
		return "."
	case TimeOfDayFormatFrenchCanadian:
		return "h"
	}
	return ":"
}

func (w *timeSelectorSeparator) Build(bc widgets.BuildContext) widgets.Widget {
	timePickerTheme := TimePickerThemeOf(bc)
	defaultTheme := timePickerDefaultsOf(bc, timePickerEntryModeOf(bc))
	var states widgets.WidgetStates
	// The color: the separator color of the theme, then the hour and minute text
	// color of the theme, then the separator color of the defaults, then the
	// hour and minute text color of the defaults.
	var color ui.Color
	switch {
	case timePickerTheme.TimeSelectorSeparatorColor != nil && timePickerTheme.TimeSelectorSeparatorColor.Resolve(states).IsSome():
		color = timePickerTheme.TimeSelectorSeparatorColor.Resolve(states).Value()
	case timePickerTheme.HourMinuteTextStateColor != nil || timePickerTheme.HourMinuteTextColor.IsSome():
		color = timePickerThemeColor(timePickerTheme.HourMinuteTextColor, timePickerTheme.HourMinuteTextStateColor, defaultTheme.hourMinuteTextColor, states)
	case defaultTheme.separatorColor != nil:
		color = defaultTheme.separatorColor.Resolve(states)
	default:
		color = defaultTheme.hourMinuteTextColor.Resolve(states)
	}
	// The style: the separator style of the theme, then the hour and minute style
	// of the theme, then the separator style of the defaults, then the hour and
	// minute style of the defaults.
	var style *painting.TextStyle
	switch {
	case timePickerTheme.TimeSelectorSeparatorTextStyle != nil && timePickerTheme.TimeSelectorSeparatorTextStyle.Resolve(states) != nil:
		style = timePickerTheme.TimeSelectorSeparatorTextStyle.Resolve(states)
	case timePickerTheme.HourMinuteTextStyle != nil:
		style = timePickerTheme.HourMinuteTextStyle
	case defaultTheme.separatorTextStyle != nil:
		style = defaultTheme.separatorTextStyle
	default:
		style = defaultTheme.hourMinuteTextStyle
	}
	style = style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(color), Height: foundation.Some(1.0)})
	height := defaultTheme.hourMinuteSize.Height
	if !timePickerEntryModeOf(bc).isDial() {
		height = defaultTheme.hourMinuteInputSize.Height
	}
	width := 24.0
	if w.timeOfDayFormat == TimeOfDayFormatFrenchCanadian {
		width = 36
	}
	return widgets.NewExcludeSemantics(widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewText(w.value(), widgets.TextOpts{Style: style, TextScaler: painting.TextScalerNoScaling})),
		widgets.SizedBoxOpts{Width: foundation.Some(width), Height: foundation.Some(height)}))
}

// dialMinuteControl is the minute in the header of the dial.
//
// Flutter: material/time_picker.dart _DialMinuteControl
type dialMinuteControl struct{ widgets.StatelessWidget }

func newDialMinuteControl() *dialMinuteControl {
	return &dialMinuteControl{StatelessWidget: widgets.NewStatelessWidgetBase()}
}

func (w *dialMinuteControl) Build(bc widgets.BuildContext) widgets.Widget {
	debugDialTimePickerEntryMode(bc)
	l := MaterialLocalizationsOf(bc)
	selectedTime := timePickerSelectedTimeOf(bc)
	formattedMinute := l.FormatMinute(selectedTime)
	nextMinute := selectedTime.Replacing(TimeOfDayOpts{Minute: foundation.Some(mod(selectedTime.Minute+1, MinutesPerHour))})
	previousMinute := selectedTime.Replacing(TimeOfDayOpts{Minute: foundation.Some(mod(selectedTime.Minute-1, MinutesPerHour))})
	return widgets.NewSemantics(newDialTimeSelectorControlWidget(dialTimeSelectorControl{
		isSelected:  timePickerHourMinuteModeOf(bc) == hourMinuteModeMinute,
		text:        formattedMinute,
		onTap:       func() { setTimePickerHourMinuteMode(bc, hourMinuteModeMinute) },
		onDoubleTap: timePickerModelOf(bc, aspectOnMinuteDoubleTapped).onMinuteDoubleTapped,
	}), widgets.SemanticsOpts{
		ExcludeSemantics: true,
		Value:            foundation.Some(l.TimePickerMinuteModeAnnouncement() + " " + formattedMinute),
		IncreasedValue:   foundation.Some(l.FormatMinute(nextMinute)),
		OnIncrease:       func() { setTimePickerSelectedTime(bc, nextMinute) },
		DecreasedValue:   foundation.Some(l.FormatMinute(previousMinute)),
		OnDecrease:       func() { setTimePickerSelectedTime(bc, previousMinute) },
	})
}

// dayPeriodControl is the AM/PM toggle.
//
// Flutter: material/time_picker.dart _DayPeriodControl
type dayPeriodControl struct {
	widgets.StatelessWidget
	onPeriodChanged func(TimeOfDay)
}

func newDayPeriodControl(onPeriodChanged func(TimeOfDay)) *dayPeriodControl {
	return &dayPeriodControl{StatelessWidget: widgets.NewStatelessWidgetBase(), onPeriodChanged: onPeriodChanged}
}

func (w *dayPeriodControl) togglePeriod(bc widgets.BuildContext) {
	selectedTime := timePickerSelectedTimeOf(bc)
	newHour := (selectedTime.Hour + HoursPerPeriod) % HoursPerDay
	newTime := selectedTime.Replacing(TimeOfDayOpts{Hour: foundation.Some(newHour)})
	if w.onPeriodChanged != nil {
		w.onPeriodChanged(newTime)
	} else {
		setTimePickerSelectedTime(bc, newTime)
	}
}

func (w *dayPeriodControl) setAm(bc widgets.BuildContext) {
	if timePickerSelectedTimeOf(bc).Period() == DayPeriodAM {
		return
	}
	w.togglePeriod(bc)
}

func (w *dayPeriodControl) setPm(bc widgets.BuildContext) {
	if timePickerSelectedTimeOf(bc).Period() == DayPeriodPM {
		return
	}
	w.togglePeriod(bc)
}

func (w *dayPeriodControl) Build(bc widgets.BuildContext) widgets.Widget {
	l := MaterialLocalizationsOf(bc)
	timePickerTheme := timePickerThemeOf(bc)
	defaultTheme := timePickerDefaultThemeOf(bc)
	selectedTime := timePickerSelectedTimeOf(bc)
	amSelected := selectedTime.Period() == DayPeriodAM
	pmSelected := !amSelected
	resolvedSide := defaultTheme.dayPeriodBorderSide
	if s, ok := timePickerTheme.DayPeriodBorderSide.Get(); ok {
		resolvedSide = s
	}
	resolvedShape := firstNonNil(timePickerTheme.DayPeriodShape, defaultTheme.dayPeriodShape).CopyWithSide(resolvedSide)

	var dayPeriodSize ui.Size
	orientation := widgets.OrientationPortrait
	if timePickerEntryModeOf(bc).isDial() {
		orientation = timePickerOrientationOf(bc)
		if orientation == widgets.OrientationPortrait {
			dayPeriodSize = defaultTheme.dayPeriodPortraitSize
		} else {
			dayPeriodSize = defaultTheme.dayPeriodLandscapeSize
		}
	} else {
		dayPeriodSize = defaultTheme.dayPeriodInputSize
	}
	amShape, pmShape := resolvedShape, resolvedShape
	rounded, hasRoundedBorder := resolvedShape.(painting.RoundedRectangleBorder)
	var borderRadius painting.BorderRadius
	if hasRoundedBorder {
		borderRadius, hasRoundedBorder = rounded.BorderRadius.(painting.BorderRadius)
	}
	if orientation == widgets.OrientationPortrait {
		if hasRoundedBorder {
			amShape = rounded.CopyWith(painting.RoundedRectangleBorderCopy{BorderRadius: painting.BorderRadius{
				TopLeft: borderRadius.TopLeft, TopRight: borderRadius.TopRight}})
			pmShape = rounded.CopyWith(painting.RoundedRectangleBorderCopy{BorderRadius: painting.BorderRadius{
				BottomLeft: borderRadius.BottomLeft, BottomRight: borderRadius.BottomRight}})
		}
		minInteractiveSize := ui.Size{Width: dayPeriodSize.Width, Height: math.Max(dayPeriodSize.Height, 2*MinInteractiveDimension)}
		extra := (minInteractiveSize.Height - dayPeriodSize.Height) / 2
		amButton := newAmPmButton(amSelected, func() { w.setAm(bc) }, l.AnteMeridiemAbbreviation(), painting.NewEdgeInsetsFromLTRB(0, extra, 0, 0), amShape)
		pmButton := newAmPmButton(pmSelected, func() { w.setPm(bc) }, l.PostMeridiemAbbreviation(), painting.NewEdgeInsetsFromLTRB(0, 0, 0, extra), pmShape)
		return newDayPeriodInputPadding(minInteractiveSize, orientation, widgets.NewSizedBoxFromSize(minInteractiveSize,
			widgets.NewColumn([]widgets.Widget{widgets.NewExpanded(amButton), widgets.NewExpanded(pmButton)})))
	}
	if hasRoundedBorder {
		amShape = rounded.CopyWith(painting.RoundedRectangleBorderCopy{BorderRadius: painting.BorderRadius{
			TopLeft: borderRadius.TopLeft, BottomLeft: borderRadius.BottomLeft}})
		pmShape = rounded.CopyWith(painting.RoundedRectangleBorderCopy{BorderRadius: painting.BorderRadius{
			TopRight: borderRadius.TopRight, BottomRight: borderRadius.BottomRight}})
	}
	minInteractiveSize := ui.Size{Width: dayPeriodSize.Width, Height: math.Max(dayPeriodSize.Height, MinInteractiveDimension)}
	padding := painting.NewEdgeInsetsSymmetric((minInteractiveSize.Height-dayPeriodSize.Height)/2, 0)
	amButton := newAmPmButton(amSelected, func() { w.setAm(bc) }, l.AnteMeridiemAbbreviation(), padding, amShape)
	pmButton := newAmPmButton(pmSelected, func() { w.setPm(bc) }, l.PostMeridiemAbbreviation(), padding, pmShape)
	return newDayPeriodInputPadding(minInteractiveSize, orientation, widgets.NewSizedBox(
		widgets.NewRow([]widgets.Widget{widgets.NewExpanded(amButton), widgets.NewExpanded(pmButton)}),
		widgets.SizedBoxOpts{Height: foundation.Some(minInteractiveSize.Height)}))
}

// amPmButton is a button of the AM/PM toggle.
//
// Flutter: material/time_picker.dart _AmPmButton
type amPmButton struct {
	widgets.StatelessWidget
	selected  bool
	onPressed func()
	label     string
	padding   painting.EdgeInsets
	shape     painting.OutlinedBorder
}

func newAmPmButton(selected bool, onPressed func(), label string, padding painting.EdgeInsets, shape painting.OutlinedBorder) *amPmButton {
	return &amPmButton{StatelessWidget: widgets.NewStatelessWidgetBase(), selected: selected, onPressed: onPressed, label: label, padding: padding, shape: shape}
}

func (w *amPmButton) Build(bc widgets.BuildContext) widgets.Widget {
	var states widgets.WidgetStates
	if w.selected {
		states = states.With(widgets.WidgetStateSelected)
	}
	timePickerTheme := timePickerThemeOf(bc)
	defaultTheme := timePickerDefaultThemeOf(bc)
	// Deviation: a plain dayPeriodColor of the theme is transparent unless the
	// state is selected; Flutter's getter wraps it in a WidgetStateColor.
	var background ui.Color
	switch {
	case timePickerTheme.DayPeriodStateColor != nil && timePickerTheme.DayPeriodStateColor.Resolve(states).IsSome():
		background = timePickerTheme.DayPeriodStateColor.Resolve(states).Value()
	case timePickerTheme.DayPeriodColor.IsSome():
		if w.selected {
			background = timePickerTheme.DayPeriodColor.Value()
		} else {
			background = Colors.Transparent
		}
	default:
		background = defaultTheme.dayPeriodColor.Resolve(states)
	}
	resolvedTextColor := timePickerThemeColor(timePickerTheme.DayPeriodTextColor, timePickerTheme.DayPeriodTextStateColor, defaultTheme.dayPeriodTextColor, states)
	resolvedTextStyle := firstNonNil(timePickerTheme.DayPeriodTextStyle, defaultTheme.dayPeriodTextStyle)
	if resolvedTextStyle != nil {
		resolvedTextStyle = resolvedTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(resolvedTextColor)})
	}
	buttonTextScaler := widgets.MediaQueryTextScalerOf(bc).Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(2.0)})
	semantics := widgets.SemanticsOpts{InMutuallyExclusiveGroup: foundation.Some(true), Button: foundation.Some(true)}
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS {
		semantics.Selected = foundation.Some(w.selected)
	} else {
		semantics.Checked = foundation.Some(w.selected)
	}
	return widgets.NewSemantics(widgets.NewPadding(w.padding, NewMaterial(
		NewInkWell(widgets.NewCenter(widgets.NewText(w.label, widgets.TextOpts{Style: resolvedTextStyle, TextScaler: buttonTextScaler})), InkWellOpts{OnTap: w.onPressed}),
		MaterialOpts{ClipBehavior: ui.ClipAntiAlias, Color: foundation.Some(background), Shape: w.shape})), semantics)
}

// dayPeriodInputPadding gives the toggle a tap target of a minimum size, and
// passes the taps next to it to the nearest button.
//
// Flutter: material/time_picker.dart _DayPeriodInputPadding
type dayPeriodInputPadding struct {
	widgets.SingleChildRenderObjectWidget
	minSize     ui.Size
	orientation widgets.Orientation
}

func newDayPeriodInputPadding(minSize ui.Size, orientation widgets.Orientation, child widgets.Widget) *dayPeriodInputPadding {
	return &dayPeriodInputPadding{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), minSize: minSize, orientation: orientation}
}

func (w *dayPeriodInputPadding) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return newRenderDayPeriodInputPadding(w.minSize, w.orientation)
}

func (w *dayPeriodInputPadding) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderDayPeriodInputPadding)
	r.setMinSize(w.minSize)
	r.setOrientation(w.orientation)
}

// renderDayPeriodInputPadding is the render object of dayPeriodInputPadding.
//
// Flutter: material/time_picker.dart _RenderInputPadding
type renderDayPeriodInputPadding struct {
	rendering.RenderShiftedBox
	minSize     ui.Size
	orientation widgets.Orientation
}

func newRenderDayPeriodInputPadding(minSize ui.Size, orientation widgets.Orientation) *renderDayPeriodInputPadding {
	r := &renderDayPeriodInputPadding{minSize: minSize, orientation: orientation}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (r *renderDayPeriodInputPadding) setMinSize(v ui.Size) {
	if r.minSize == v {
		return
	}
	r.minSize = v
	r.MarkNeedsLayout()
}

func (r *renderDayPeriodInputPadding) setOrientation(v widgets.Orientation) {
	if r.orientation == v {
		return
	}
	r.orientation = v
	r.MarkNeedsLayout()
}

func (r *renderDayPeriodInputPadding) ComputeMinIntrinsicWidth(height float64) float64 {
	if c := r.Child(); c != nil {
		return math.Max(c.GetMinIntrinsicWidth(height), r.minSize.Width)
	}
	return 0
}

func (r *renderDayPeriodInputPadding) ComputeMinIntrinsicHeight(width float64) float64 {
	if c := r.Child(); c != nil {
		return math.Max(c.GetMinIntrinsicHeight(width), r.minSize.Height)
	}
	return 0
}

func (r *renderDayPeriodInputPadding) ComputeMaxIntrinsicWidth(height float64) float64 {
	if c := r.Child(); c != nil {
		return math.Max(c.GetMaxIntrinsicWidth(height), r.minSize.Width)
	}
	return 0
}

func (r *renderDayPeriodInputPadding) ComputeMaxIntrinsicHeight(width float64) float64 {
	if c := r.Child(); c != nil {
		return math.Max(c.GetMaxIntrinsicHeight(width), r.minSize.Height)
	}
	return 0
}

func (r *renderDayPeriodInputPadding) computeSize(constraints rendering.BoxConstraints, layoutChild func(rendering.Box, rendering.BoxConstraints) ui.Size) ui.Size {
	if c := r.Child(); c != nil {
		childSize := layoutChild(c, constraints)
		return constraints.Constrain(ui.Size{Width: math.Max(childSize.Width, r.minSize.Width), Height: math.Max(childSize.Height, r.minSize.Height)})
	}
	return ui.Size{}
}

func (r *renderDayPeriodInputPadding) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return r.computeSize(constraints, func(c rendering.Box, cs rendering.BoxConstraints) ui.Size { return c.GetDryLayout(cs) })
}

func (r *renderDayPeriodInputPadding) PerformLayout() {
	r.SetSize(r.computeSize(r.BoxConstraints(), func(c rendering.Box, cs rendering.BoxConstraints) ui.Size {
		c.Layout(cs, rendering.LayoutOpts{ParentUsesSize: true})
		return c.Size()
	}))
	if c := r.Child(); c != nil {
		c.ParentData().(*rendering.BoxParentData).Offset = painting.AlignmentCenter.AlongOffset(
			ui.Offset{Dx: r.Size().Width - c.Size().Width, Dy: r.Size().Height - c.Size().Height})
	}
}

func (r *renderDayPeriodInputPadding) HitTest(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	if r.RenderShiftedBox.HitTest(result, position) {
		return true
	}
	child := r.Child()
	if child == nil {
		return false
	}
	if position.Dx < 0 || position.Dx > math.Max(child.Size().Width, r.minSize.Width) ||
		position.Dy < 0 || position.Dy > math.Max(child.Size().Height, r.minSize.Height) {
		return false
	}
	newPosition := child.Size().Center(ui.OffsetZero)
	switch {
	case r.orientation == widgets.OrientationPortrait && position.Dy > newPosition.Dy:
		newPosition = newPosition.Add(ui.Offset{Dx: 0, Dy: 1})
	case r.orientation == widgets.OrientationLandscape && position.Dx > newPosition.Dx:
		newPosition = newPosition.Add(ui.Offset{Dx: 1, Dy: 0})
	case r.orientation == widgets.OrientationPortrait:
		newPosition = newPosition.Add(ui.Offset{Dx: 0, Dy: -1})
	default:
		newPosition = newPosition.Add(ui.Offset{Dx: -1, Dy: 0})
	}
	return result.AddWithRawTransform(ui.NewMatrix4ForceToPoint(newPosition), newPosition, func(result *rendering.BoxHitTestResult, position ui.Offset) bool {
		return child.HitTest(result, newPosition)
	})
}

// animationStandardEasing is the standard easing of the dial.
//
// Flutter: material/curves.dart standardEasing
func animationStandardEasing() animation.Curve { return animation.Curves.FastOutSlowIn }
