// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Sizes of the time picker dialog.
var (
	timePickerPortraitSize       = ui.Size{Width: 310, Height: 468}
	timePickerLandscapeSize      = ui.Size{Width: 524, Height: 342}
	timePickerLandscapeSizeM2    = ui.Size{Width: 508, Height: 300}
	timePickerInputSize          = ui.Size{Width: 312, Height: 252}
	timePickerMinPortraitSize    = ui.Size{Width: 238, Height: 326}
	timePickerMinLandscapeSize   = ui.Size{Width: 416, Height: 248}
	timePickerMinInputSize       = ui.Size{Width: 312, Height: 196}
	timePickerInputMinimumHeight = 216.0
)

// EntryModeChangeCallback is called when the entry mode of the time picker
// changes.
//
// Flutter: material/time_picker.dart EntryModeChangeCallback
type EntryModeChangeCallback func(mode TimePickerEntryMode)

// TimePickerDialogOpts are the optional parameters of NewTimePickerDialog.
//
// Flutter: material/time_picker.dart TimePickerDialog.new
type TimePickerDialogOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CancelText is the label of the cancel button.
	CancelText string
	// ConfirmText is the label of the confirm button.
	ConfirmText string
	// HelpText is the text at the top of the dialog.
	HelpText string
	// ErrorInvalidText is the error text for a time that is not valid.
	ErrorInvalidText string
	// HourLabelText is the label of the hour field.
	HourLabelText string
	// MinuteLabelText is the label of the minute field.
	MinuteLabelText string
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// dial.
	InitialEntryMode TimePickerEntryMode
	// Orientation overrides the orientation of the screen.
	Orientation widgets.Orientation
	// OnEntryModeChanged is called when the entry mode changes.
	OnEntryModeChanged EntryModeChangeCallback
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToTimerEntryModeIcon is the icon of the button that switches to the
	// dial mode.
	SwitchToTimerEntryModeIcon *widgets.Icon
	// EmptyInitialInput leaves the text fields empty at the start of the input
	// mode.
	EmptyInitialInput bool
}

// TimePickerDialog is a Material Design time picker, to be shown in a dialog
// with ShowTimePicker.
//
// Flutter: material/time_picker.dart TimePickerDialog
type TimePickerDialog struct {
	widgets.StatefulWidget
	initialTime TimeOfDay
	o           TimePickerDialogOpts
}

// NewTimePickerDialog creates a time picker dialog that starts at the initial
// time.
//
// Flutter: material/time_picker.dart TimePickerDialog.new
func NewTimePickerDialog(initialTime TimeOfDay, opts ...TimePickerDialogOpts) *TimePickerDialog {
	o := firstOpts(opts)
	if o.InitialEntryMode == TimePickerEntryModeUnspecified {
		o.InitialEntryMode = TimePickerEntryModeDial
	}
	return &TimePickerDialog{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), initialTime: initialTime, o: o}
}

// InitialTime is the time the picker starts at.
//
// Flutter: material/time_picker.dart TimePickerDialog.initialTime
func (w *TimePickerDialog) InitialTime() TimeOfDay { return w.initialTime }

// CreateState creates the state.
//
// Flutter: material/time_picker.dart TimePickerDialog.createState
func (w *TimePickerDialog) CreateState() widgets.State { return &timePickerDialogState{} }

type timePickerDialogState struct {
	widgets.StateOf[*TimePickerDialog]
	entryMode        TimePickerEntryMode
	selectedTime     TimeOfDay
	formKey          *widgets.GlobalKey[*widgets.FormState]
	autovalidateMode widgets.AutovalidateMode
	orientation      widgets.Orientation
}

func (s *timePickerDialogState) InitState() {
	w := s.Widget()
	s.entryMode = w.o.InitialEntryMode
	s.selectedTime = w.initialTime
	s.formKey = widgets.NewGlobalKey[*widgets.FormState]()
	s.autovalidateMode = widgets.AutovalidateModeDisabled
	s.orientation = w.o.Orientation
}

func (s *timePickerDialogState) isInput() bool {
	return s.entryMode == TimePickerEntryModeInput || s.entryMode == TimePickerEntryModeInputOnly
}

func (s *timePickerDialogState) handleTimeChanged(value TimeOfDay) {
	if value != s.selectedTime {
		s.SetState(func() { s.selectedTime = value })
	}
}

func (s *timePickerDialogState) handleEntryModeChanged(value TimePickerEntryMode) {
	if value == s.entryMode {
		return
	}
	s.SetState(func() {
		switch s.entryMode {
		case TimePickerEntryModeDial:
			s.autovalidateMode = widgets.AutovalidateModeDisabled
		case TimePickerEntryModeInput:
			if form, ok := s.formKey.CurrentState(); ok {
				form.Save()
			}
		}
		s.entryMode = value
		if f := s.Widget().o.OnEntryModeChanged; f != nil {
			f(value)
		}
	})
}

func (s *timePickerDialogState) toggleEntryMode() {
	// Deviation: Flutter creates a FlutterError for the dial only and input
	// only modes and drops it, so nothing happens; the same here.
	switch s.entryMode {
	case TimePickerEntryModeDial:
		s.handleEntryModeChanged(TimePickerEntryModeInput)
	case TimePickerEntryModeInput:
		s.handleEntryModeChanged(TimePickerEntryModeDial)
	}
}

func (s *timePickerDialogState) handleCancel() { widgets.NavigatorPop(s.Context(), nil) }

func (s *timePickerDialogState) handleOk() {
	if s.isInput() {
		form, _ := s.formKey.CurrentState()
		if !form.Validate() {
			s.SetState(func() { s.autovalidateMode = widgets.AutovalidateModeAlways })
			return
		}
		form.Save()
	}
	widgets.NavigatorPop(s.Context(), s.selectedTime)
}

func (s *timePickerDialogState) orientationOf(bc widgets.BuildContext) widgets.Orientation {
	if s.orientation != widgets.OrientationUnspecified {
		return s.orientation
	}
	return widgets.MediaQueryOrientationOf(bc)
}

// inputPickerWidth is the width of the dialog in the input modes, given the
// width of the 12 hour layout.
func (s *timePickerDialogState) inputPickerWidth(bc widgets.BuildContext, base float64, useMaterial3 bool) float64 {
	format := MaterialLocalizationsOf(bc).TimeOfDayFormat(widgets.MediaQueryAlwaysUse24HourFormatOf(bc))
	switch format {
	case TimeOfDayFormatHHColonMM, TimeOfDayFormatHHDotMM, TimeOfDayFormatFrenchCanadian, TimeOfDayFormatHColonMM:
		var defaults *timePickerDefaults
		if useMaterial3 {
			defaults = timePickerDefaultsM3(ThemeOf(bc), TimePickerEntryModeDial)
		} else {
			defaults = timePickerDefaultsM2(ThemeOf(bc))
		}
		return base - defaults.dayPeriodPortraitSize.Width - 12
	default:
		if useMaterial3 {
			return base - 32
		}
		return base
	}
}

func (s *timePickerDialogState) minDialogSize(bc widgets.BuildContext, useMaterial3 bool) ui.Size {
	orientation := s.orientationOf(bc)
	if !s.isInput() {
		if orientation == widgets.OrientationPortrait {
			return timePickerMinPortraitSize
		}
		return timePickerMinLandscapeSize
	}
	return ui.Size{Width: s.inputPickerWidth(bc, timePickerMinInputSize.Width, useMaterial3), Height: timePickerMinInputSize.Height}
}

func (s *timePickerDialogState) dialogSize(bc widgets.BuildContext, useMaterial3 bool) ui.Size {
	orientation := s.orientationOf(bc)
	const fontSizeToScale = 14.0
	textScaleFactor := widgets.MediaQueryTextScalerOf(bc).Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(1.1)}).Scale(fontSizeToScale) / fontSizeToScale
	var size ui.Size
	if !s.isInput() {
		if orientation == widgets.OrientationPortrait {
			size = timePickerPortraitSize
		} else {
			height := timePickerLandscapeSizeM2.Height
			if useMaterial3 {
				height = timePickerLandscapeSize.Height
			}
			size = ui.Size{Width: timePickerLandscapeSize.Width * textScaleFactor, Height: height}
		}
	} else {
		size = ui.Size{Width: s.inputPickerWidth(bc, timePickerInputSize.Width, useMaterial3), Height: timePickerInputSize.Height}
	}
	return ui.Size{Width: size.Width, Height: size.Height * textScaleFactor}
}

func (s *timePickerDialogState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	widgets.DebugCheckHasMediaQuery(bc)
	theme := ThemeOf(bc)
	pickerTheme := TimePickerThemeOf(bc)
	var defaultTheme *timePickerDefaults
	if theme.UseMaterial3 {
		defaultTheme = timePickerDefaultsM3(theme, TimePickerEntryModeDial)
	} else {
		defaultTheme = timePickerDefaultsM2(theme)
	}
	shape := firstNonNil(pickerTheme.Shape, defaultTheme.shape)
	entryModeIconColor := firstOpt(pickerTheme.EntryModeIconColor, foundation.Some(defaultTheme.entryModeIconColor)).Value()
	localizations := MaterialLocalizationsOf(bc)

	var actionsChildren []widgets.Widget
	if s.entryMode == TimePickerEntryModeDial || s.entryMode == TimePickerEntryModeInput {
		var icon widgets.Widget
		var tooltip string
		if s.entryMode == TimePickerEntryModeDial {
			icon = iconOr(o.SwitchToInputEntryModeIcon, widgets.NewIcon(Icons.KeyboardOutlined()))
			tooltip = localizations.InputTimeModeButtonLabel()
		} else {
			icon = iconOr(o.SwitchToTimerEntryModeIcon, widgets.NewIcon(Icons.AccessTime()))
			tooltip = localizations.DialModeButtonLabel()
		}
		btn := IconButtonOpts{Tooltip: tooltip}
		if theme.UseMaterial3 {
			btn.Style = IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: foundation.Some(entryModeIconColor)})
		} else {
			btn.Color = foundation.Some(entryModeIconColor)
		}
		actionsChildren = append(actionsChildren, NewIconButton(s.toggleEntryMode, icon, btn))
	}
	actionsChildren = append(actionsChildren, widgets.NewExpanded(
		widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(36.0)}),
			widgets.NewAlign(widgets.NewOverflowBar([]widgets.Widget{
				NewTextButton(s.handleCancel, widgets.NewText(orDefault(o.CancelText, upperIfMaterial2(theme.UseMaterial3, localizations.CancelButtonLabel()))),
					TextButtonOpts{Style: firstNonNil(pickerTheme.CancelButtonStyle, defaultTheme.cancelButtonStyle)}),
				NewTextButton(s.handleOk, widgets.NewText(orDefault(o.ConfirmText, localizations.OkButtonLabel())),
					TextButtonOpts{Style: firstNonNil(pickerTheme.ConfirmButtonStyle, defaultTheme.confirmButtonStyle)}),
			}, widgets.OverflowBarOpts{Spacing: 8, OverflowAlignment: rendering.OverflowBarAlignmentEnd}),
				widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd}))))
	startPad := 4.0
	if theme.UseMaterial3 {
		startPad = 0
	}
	actions := widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(startPad, 0, 0, 0), widgets.NewRow(actionsChildren))

	tapTargetSizeOffset := ui.Offset{}
	if theme.MaterialTapTargetSize == MaterialTapTargetSizeShrinkWrap {
		tapTargetSizeOffset = ui.Offset{Dy: -12}
	}
	dialogSize := s.dialogSize(bc, theme.UseMaterial3)
	dialogSize = ui.Size{Width: dialogSize.Width + tapTargetSizeOffset.Dx, Height: dialogSize.Height + tapTargetSizeOffset.Dy}
	minDialogSize := s.minDialogSize(bc, theme.UseMaterial3)
	minDialogSize = ui.Size{Width: minDialogSize.Width + tapTargetSizeOffset.Dx, Height: minDialogSize.Height + tapTargetSizeOffset.Dy}

	verticalInset := 24.0
	if s.isInput() {
		verticalInset = 0
	}
	elevation := firstOpt(pickerTheme.Elevation, foundation.Some(defaultTheme.elevation)).Value()
	padding := painting.EdgeInsetsGeometry(defaultTheme.padding)
	if pickerTheme.Padding != nil {
		padding = pickerTheme.Padding
	}
	insets := painting.NewEdgeInsetsSymmetric(verticalInset, 16)
	return NewDialog(widgets.NewPadding(padding, widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		constrainedSize := constraints.Constrain(dialogSize)
		allowedSize := ui.Size{Width: math.Max(constrainedSize.Width, minDialogSize.Width), Height: math.Max(constrainedSize.Height, minDialogSize.Height)}
		return widgets.NewSingleChildScrollView(
			widgets.NewSingleChildScrollView(
				widgets.NewAnimatedContainer(timePickerDialogSizeAnimationDuration,
					widgets.NewColumn([]widgets.Widget{
						widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
							var child widgets.Widget = widgets.NewForm(newTimePicker(timePickerArgs{
								time: w.initialTime, onTimeChanged: s.handleTimeChanged, o: o, entryMode: s.entryMode,
								orientation: s.orientation, onEntryModeChanged: s.handleEntryModeChanged,
							}), widgets.FormOpts{Key: s.formKey, AutovalidateMode: s.autovalidateMode})
							if !s.isInput() {
								return widgets.NewFlexible(child)
							}
							return child
						}),
						actions,
					}, widgets.ColumnOpts{
						MainAxisSize: rendering.MainAxisSizeMin, MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween,
						CrossAxisAlignment: rendering.CrossAxisAlignmentStart,
					}),
					widgets.AnimatedContainerOpts{
						Width: foundation.Some(allowedSize.Width), Curve: animation.Curves.EaseIn,
						Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
							MinHeight: foundation.Some(timePickerInputMinimumHeight), MaxHeight: foundation.Some(allowedSize.Height),
						})),
					})),
			widgets.SingleChildScrollViewOpts{ScrollDirection: painting.AxisHorizontal})
	})), DialogOpts{
		Shape: shape, Elevation: foundation.Some(elevation),
		BackgroundColor: firstOpt(pickerTheme.BackgroundColor, foundation.Some(defaultTheme.backgroundColor)),
		InsetPadding:    &insets,
	})
}

// timePickerArgs are the parameters of timePicker.
type timePickerArgs struct {
	time               TimeOfDay
	onTimeChanged      func(TimeOfDay)
	o                  TimePickerDialogOpts
	entryMode          TimePickerEntryMode
	orientation        widgets.Orientation
	onEntryModeChanged func(TimePickerEntryMode)
}

// timePicker is the picker inside the dialog: the dial or the input, and the
// model that its parts depend on.
//
// Flutter: material/time_picker.dart _TimePicker
type timePicker struct {
	widgets.StatefulWidget
	a timePickerArgs
}

func newTimePicker(a timePickerArgs) *timePicker {
	return &timePicker{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *timePicker) CreateState() widgets.State { return &timePickerState{} }

type timePickerState struct {
	widgets.StateOf[*timePicker]
	localizations   MaterialLocalizations
	hourMinuteMode  hourMinuteMode
	autofocusHour   foundation.Opt[bool]
	autofocusMinute foundation.Opt[bool]
	orientation     widgets.Orientation
	selectedTime    TimeOfDay
}

func (s *timePickerState) InitState() {
	a := s.Widget().a
	s.hourMinuteMode = hourMinuteModeHour
	s.orientation = a.orientation
	s.selectedTime = a.time
}

func (s *timePickerState) DidChangeDependencies() {
	s.localizations = MaterialLocalizationsOf(s.Context())
}

func (s *timePickerState) DidUpdateWidget(old *timePicker) {
	a := s.Widget().a
	if old.a.orientation != a.orientation {
		s.orientation = a.orientation
	}
	if old.a.time != a.time {
		s.selectedTime = a.time
	}
}

func (s *timePickerState) handleHourMinuteModeChanged(mode hourMinuteMode) {
	// Deviation: Flutter vibrates on the platforms that do (HapticFeedback);
	// there is no haptic feedback on the desktop (SPEC §12.10).
	s.SetState(func() { s.hourMinuteMode = mode })
}

func (s *timePickerState) handleEntryModeToggle() {
	a := s.Widget().a
	s.SetState(func() {
		newMode := a.entryMode
		switch a.entryMode {
		case TimePickerEntryModeDial:
			newMode = TimePickerEntryModeInput
		case TimePickerEntryModeInput:
			s.autofocusHour = foundation.Some(false)
			s.autofocusMinute = foundation.Some(false)
			newMode = TimePickerEntryModeDial
		}
		if a.onEntryModeChanged != nil {
			a.onEntryModeChanged(newMode)
		}
	})
}

func (s *timePickerState) handleTimeChanged(value TimeOfDay) {
	s.SetState(func() {
		s.selectedTime = value
		if f := s.Widget().a.onTimeChanged; f != nil {
			f(value)
		}
	})
}

func (s *timePickerState) handleHourDoubleTapped() {
	s.autofocusHour = foundation.Some(true)
	s.handleEntryModeToggle()
}

func (s *timePickerState) handleMinuteDoubleTapped() {
	s.autofocusMinute = foundation.Some(true)
	s.handleEntryModeToggle()
}

func (s *timePickerState) handleHourSelected() {
	s.SetState(func() { s.hourMinuteMode = hourMinuteModeMinute })
}

func (s *timePickerState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	widgets.DebugCheckHasMediaQuery(bc)
	timeOfDayFormat := s.localizations.TimeOfDayFormat(widgets.MediaQueryAlwaysUse24HourFormatOf(bc))
	theme := ThemeOf(bc)
	var defaultTheme *timePickerDefaults
	if theme.UseMaterial3 {
		defaultTheme = timePickerDefaultsM3(theme, a.entryMode)
	} else {
		defaultTheme = timePickerDefaultsM2(theme)
	}
	orientation := s.orientation
	if orientation == widgets.OrientationUnspecified {
		orientation = widgets.MediaQueryOrientationOf(bc)
	}
	var hourMode hourDialType
	switch HourFormatOf(timeOfDayFormat) {
	case HourFormatHH, HourFormatH:
		if theme.UseMaterial3 {
			hourMode = hourDialTwentyFourHourDoubleRing
		} else {
			hourMode = hourDialTwentyFourHour
		}
	default:
		hourMode = hourDialTwelveHour
	}

	hSym := 16.0
	if theme.UseMaterial3 {
		hSym = 0
	}
	var picker widgets.Widget
	switch a.entryMode {
	case TimePickerEntryModeDial, TimePickerEntryModeDialOnly:
		helpText := orDefault(a.o.HelpText, upperIfMaterial2(theme.UseMaterial3, s.localizations.TimePickerDialHelpText()))
		portraitMinInteractiveVerticalAdjustment := math.Max(0, 2*MinInteractiveDimension-defaultTheme.dayPeriodPortraitSize.Height)
		var dialPadding painting.EdgeInsetsGeometry
		if orientation == widgets.OrientationPortrait {
			dialPadding = painting.NewEdgeInsetsFromLTRB(12, 36-portraitMinInteractiveVerticalAdjustment/2, 12, 0)
		} else {
			dialPadding = painting.NewEdgeInsetsDirectionalFromSTEB(64, 0, 0, 0)
		}
		dialWidget := widgets.NewPadding(dialPadding, widgets.NewExcludeSemantics(
			widgets.NewSizedBoxFromSize(defaultTheme.dialSize, widgets.NewAspectRatio(1,
				newDial(s.selectedTime, s.hourMinuteMode, hourMode, s.handleTimeChanged, s.handleHourSelected)))))
		if orientation == widgets.OrientationPortrait {
			picker = widgets.NewColumn([]widgets.Widget{
				widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, hSym), newDialTimePickerHeader(helpText)),
				widgets.NewExpanded(widgets.NewColumn([]widgets.Widget{
					widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, hSym), dialWidget)),
				}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})),
			}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
		} else {
			picker = widgets.NewColumn([]widgets.Widget{
				widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, hSym), widgets.NewRow([]widgets.Widget{
					newDialTimePickerHeader(helpText),
					widgets.NewExpanded(dialWidget),
				}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))),
			})
		}
	default:
		helpText := orDefault(a.o.HelpText, upperIfMaterial2(theme.UseMaterial3, s.localizations.TimePickerInputHelpText()))
		picker = widgets.NewColumn([]widgets.Widget{
			&timePickerInput{
				StatefulWidget:      widgets.NewStatefulWidgetBase(),
				initialSelectedTime: s.selectedTime, errorInvalidText: a.o.ErrorInvalidText, hourLabelText: a.o.HourLabelText,
				minuteLabelText: a.o.MinuteLabelText, helpText: helpText, autofocusHour: s.autofocusHour, autofocusMinute: s.autofocusMinute,
				emptyInitialTime: a.o.EmptyInitialInput,
			},
		}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})
	}
	return &timePickerModel{
		InheritedModel: widgets.NewInheritedModelBase(picker),
		entryMode:      a.entryMode, selectedTime: s.selectedTime, hourMinuteMode: s.hourMinuteMode, orientation: orientation,
		onHourMinuteModeChanged: s.handleHourMinuteModeChanged, onHourDoubleTapped: s.handleHourDoubleTapped,
		onMinuteDoubleTapped: s.handleMinuteDoubleTapped, hourDialType: hourMode, onSelectedTimeChanged: s.handleTimeChanged,
		useMaterial3: theme.UseMaterial3, use24HourFormat: widgets.MediaQueryAlwaysUse24HourFormatOf(bc),
		theme: TimePickerThemeOf(bc), defaultTheme: defaultTheme,
	}
}

// ShowTimePickerOpts are the optional parameters of ShowTimePicker.
//
// Flutter: material/time_picker.dart showTimePicker
type ShowTimePickerOpts struct {
	// Builder wraps the dialog, for instance to change its theme.
	Builder func(bc widgets.BuildContext, dialog widgets.Widget) widgets.Widget
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
	// InitialEntryMode is the entry mode the dialog starts with; the default is
	// dial.
	InitialEntryMode TimePickerEntryMode
	// CancelText is the label of the cancel button.
	CancelText string
	// ConfirmText is the label of the confirm button.
	ConfirmText string
	// HelpText is the text at the top of the dialog.
	HelpText string
	// ErrorInvalidText is the error text for a time that is not valid.
	ErrorInvalidText string
	// HourLabelText is the label of the hour field.
	HourLabelText string
	// MinuteLabelText is the label of the minute field.
	MinuteLabelText string
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// OnEntryModeChanged is called when the entry mode changes.
	OnEntryModeChanged EntryModeChangeCallback
	// Orientation overrides the orientation of the screen.
	Orientation widgets.Orientation
	// SwitchToInputEntryModeIcon is the icon of the button that switches to the
	// input mode.
	SwitchToInputEntryModeIcon *widgets.Icon
	// SwitchToTimerEntryModeIcon is the icon of the button that switches to the
	// dial mode.
	SwitchToTimerEntryModeIcon *widgets.Icon
	// EmptyInitialInput leaves the text fields empty at the start of the input
	// mode.
	EmptyInitialInput bool
}

// ShowTimePicker shows a dialog that has a Material Design time picker. The
// channel gets the time the user picked, then is closed; when the user cancels
// the dialog it gets None.
//
// Deviation: Dart's Future<TimeOfDay?> becomes a channel of an Opt, since the
// zero TimeOfDay (midnight) is a time the user can pick (SPEC §12.10).
//
// Flutter: material/time_picker.dart showTimePicker
func ShowTimePicker(bc widgets.BuildContext, initialTime TimeOfDay, opts ...ShowTimePickerOpts) <-chan foundation.Opt[TimeOfDay] {
	o := firstOpts(opts)
	DebugCheckHasMaterialLocalizations(bc)
	var dialog widgets.Widget = NewTimePickerDialog(initialTime, TimePickerDialogOpts{
		InitialEntryMode: o.InitialEntryMode, CancelText: o.CancelText, ConfirmText: o.ConfirmText, HelpText: o.HelpText,
		ErrorInvalidText: o.ErrorInvalidText, HourLabelText: o.HourLabelText, MinuteLabelText: o.MinuteLabelText,
		Orientation: o.Orientation, OnEntryModeChanged: o.OnEntryModeChanged,
		SwitchToInputEntryModeIcon: o.SwitchToInputEntryModeIcon, SwitchToTimerEntryModeIcon: o.SwitchToTimerEntryModeIcon,
		EmptyInitialInput: o.EmptyInitialInput,
	})
	in := showDialogAny(bc, func(bc widgets.BuildContext) widgets.Widget {
		if o.Builder == nil {
			return dialog
		}
		return o.Builder(bc, dialog)
	}, ShowDialogOpts{
		BarrierDismissible: o.BarrierDismissible, BarrierColor: o.BarrierColor, BarrierLabel: o.BarrierLabel,
		UseRootNavigator: o.UseRootNavigator, RouteSettings: o.RouteSettings,
	})
	out := make(chan foundation.Opt[TimeOfDay], 1)
	go func() {
		defer close(out)
		if v, ok := <-in; ok {
			if t, isTime := v.(TimeOfDay); isTime {
				out <- foundation.Some(t)
			} else {
				out <- foundation.None[TimeOfDay]()
			}
		}
	}()
	return out
}
