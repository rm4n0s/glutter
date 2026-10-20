// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"strconv"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// tappableLabel is a label of the dial.
//
// Flutter: material/time_picker.dart _TappableLabel
type tappableLabel struct {
	value   int
	inner   bool
	painter *painting.TextPainter
	onTap   func()
}

// dialPainter paints the dial: the background, the labels, the hand and the
// selected labels clipped to the knob.
//
// Flutter: material/time_picker.dart _DialPainter
type dialPainter struct {
	rendering.CustomPainterBase
	primaryLabels   []*tappableLabel
	selectedLabels  []*tappableLabel
	backgroundColor ui.Color
	handColor       ui.Color
	handWidth       float64
	dotColor        ui.Color
	dotRadius       float64
	centerRadius    float64
	theta           float64
	radius          float64
	textDirection   ui.TextDirection
	selectedValue   int
}

func (p *dialPainter) dispose() {
	for _, l := range p.primaryLabels {
		l.painter.Dispose()
	}
	for _, l := range p.selectedLabels {
		l.painter.Dispose()
	}
	p.primaryLabels, p.selectedLabels = nil, nil
}

func (p *dialPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	dialRadius := ui.ClampDouble(size.ShortestSide()/2, timePickerDialMinRadius+p.dotRadius, math.Inf(1))
	labelRadius := ui.ClampDouble(dialRadius-timePickerDialPadding, timePickerDialMinRadius, math.Inf(1))
	innerLabelRadius := ui.ClampDouble(labelRadius-timePickerInnerDialOffset, 0, math.Inf(1))
	inner := 0.0
	if p.radius < 0.5 {
		inner = 1
	}
	handleRadius := ui.ClampDouble(labelRadius-inner*(labelRadius-innerLabelRadius), timePickerDialMinRadius, math.Inf(1))
	center := ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2}

	background := ui.NewPaint()
	background.SetColor(p.backgroundColor)
	canvas.DrawCircle(center, dialRadius, background)

	getOffsetForTheta := func(theta, radius float64) ui.Offset {
		return center.Add(ui.Offset{Dx: radius * math.Cos(theta), Dy: -radius * math.Sin(theta)})
	}
	paintLabels := func(labels []*tappableLabel, radius float64) {
		if len(labels) == 0 {
			return
		}
		labelThetaIncrement := -twoPi / float64(len(labels))
		labelTheta := math.Pi / 2
		for _, label := range labels {
			labelPainter := label.painter
			labelOffset := ui.Offset{Dx: -labelPainter.Width() / 2, Dy: -labelPainter.Height() / 2}
			labelPainter.Paint(canvas, getOffsetForTheta(labelTheta, radius).Add(labelOffset))
			labelTheta += labelThetaIncrement
		}
	}
	paintInnerOuterLabels := func(labels []*tappableLabel) {
		if labels == nil {
			return
		}
		var outer, inner []*tappableLabel
		for _, l := range labels {
			if l.inner {
				inner = append(inner, l)
			} else {
				outer = append(outer, l)
			}
		}
		paintLabels(outer, labelRadius)
		paintLabels(inner, innerLabelRadius)
	}
	paintInnerOuterLabels(p.primaryLabels)

	selectorPaint := ui.NewPaint()
	selectorPaint.SetColor(p.handColor)
	focusedPoint := getOffsetForTheta(p.theta, handleRadius)
	canvas.DrawCircle(center, p.centerRadius, selectorPaint)
	canvas.DrawCircle(focusedPoint, p.dotRadius, selectorPaint)
	selectorPaint.SetStrokeWidth(p.handWidth)
	canvas.DrawLine(center, focusedPoint, selectorPaint)

	// Add a dot inside the selector but only when it isn't over the labels.
	// This checks that the selector's theta is between two labels. A remainder
	// between 0.1 and 0.45 indicates that the selector is roughly not above any
	// labels. The values were derived by manually testing the dial.
	labelThetaIncrement := -twoPi / float64(len(p.primaryLabels))
	remainder := dartModFloat(p.theta, labelThetaIncrement)
	if remainder > 0.1 && remainder < 0.45 {
		selectorPaint.SetColor(p.dotColor)
		canvas.DrawCircle(focusedPoint, 2, selectorPaint)
	}

	focusedRect := ui.NewRectFromCircle(focusedPoint, p.dotRadius)
	canvas.Save()
	path := ui.NewPath()
	path.AddOval(focusedRect)
	canvas.ClipPath(path)
	paintInnerOuterLabels(p.selectedLabels)
	canvas.Restore()
}

func (p *dialPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*dialPainter)
	return !sameLabels(o.primaryLabels, p.primaryLabels) || !sameLabels(o.selectedLabels, p.selectedLabels) ||
		o.backgroundColor != p.backgroundColor || o.handColor != p.handColor || o.theta != p.theta
}

func sameLabels(a, b []*tappableLabel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// dartModFloat is Dart's % operator on doubles: the result has the sign of the
// divisor's magnitude, never negative for a positive one.
func dartModFloat(a, n float64) float64 {
	r := math.Mod(a, n)
	if r < 0 {
		r += math.Abs(n)
	}
	return r
}

// dial is the dial that selects the hour or the minute.
//
// Flutter: material/time_picker.dart _Dial
type dial struct {
	widgets.StatefulWidget
	selectedTime   TimeOfDay
	hourMinuteMode hourMinuteMode
	hourDialType   hourDialType
	onChanged      func(TimeOfDay)
	onHourSelected func()
}

func newDial(selectedTime TimeOfDay, mode hourMinuteMode, dialType hourDialType, onChanged func(TimeOfDay), onHourSelected func()) *dial {
	return &dial{StatefulWidget: widgets.NewStatefulWidgetBase(), selectedTime: selectedTime, hourMinuteMode: mode,
		hourDialType: dialType, onChanged: onChanged, onHourSelected: onHourSelected}
}

func (w *dial) CreateState() widgets.State { return &dialState{} }

type dialState struct {
	widgets.StateOf[*dial]
	widgets.TickerProvider
	themeData           *ThemeData
	localizations       MaterialLocalizations
	painter             *dialPainter
	animationController *animation.AnimationController
	thetaTween          *animation.Tween[float64]
	theta               animation.Animation[float64]
	radiusTween         *animation.Tween[float64]
	radius              animation.Animation[float64]
	dragging            bool
	position            ui.Offset
	center              ui.Offset
	dialSize            ui.Size
}

func (s *dialState) InitState() {
	w := s.Widget()
	s.animationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(timePickerDialAnimateDuration)})
	s.thetaTween = animation.NewTween(s.getThetaForTime(w.selectedTime), 0)
	s.radiusTween = animation.NewTween(s.getRadiusForTime(w.selectedTime), 0)
	s.thetaTween.End = s.thetaTween.Begin
	s.radiusTween.End = s.radiusTween.Begin
	eased := animation.Drive[float64](s.animationController, animation.NewCurveTween(animationStandardEasing()))
	s.theta = animation.Drive[float64](eased, s.thetaTween)
	s.radius = animation.Drive[float64](eased, s.radiusTween)
	s.theta.AddListener(func() { s.SetState(func() {}) })
	s.radius.AddListener(func() { s.SetState(func() {}) })
}

func (s *dialState) DidChangeDependencies() {
	widgets.DebugCheckHasMediaQuery(s.Context())
	s.themeData = ThemeOf(s.Context())
	s.localizations = MaterialLocalizationsOf(s.Context())
}

func (s *dialState) DidUpdateWidget(old *dial) {
	w := s.Widget()
	if w.hourMinuteMode != old.hourMinuteMode || w.selectedTime != old.selectedTime {
		if !s.dragging {
			s.animateTo(s.getThetaForTime(w.selectedTime), s.getRadiusForTime(w.selectedTime))
		}
	}
}

func (s *dialState) Dispose() {
	s.animationController.Dispose()
	if s.painter != nil {
		s.painter.dispose()
	}
}

func nearest(target, a, b float64) float64 {
	if math.Abs(target-a) < math.Abs(target-b) {
		return a
	}
	return b
}

func (s *dialState) animateTo(targetTheta, targetRadius float64) {
	animateToValue := func(target float64, anim animation.Animation[float64], tween *animation.Tween[float64], lo, hi float64) {
		beginValue := nearest(target, anim.Value(), hi)
		beginValue = nearest(target, beginValue, lo)
		tween.Begin = beginValue
		tween.End = target
		s.animationController.SetValue(0)
		s.animationController.Forward()
	}
	animateToValue(targetTheta, s.theta, s.thetaTween, s.theta.Value()-twoPi, s.theta.Value()+twoPi)
	animateToValue(targetRadius, s.radius, s.radiusTween, 0, 1)
}

func (s *dialState) getRadiusForTime(time TimeOfDay) float64 {
	w := s.Widget()
	if w.hourMinuteMode == hourMinuteModeHour && w.hourDialType == hourDialTwentyFourHourDoubleRing && time.Hour >= 12 {
		return 0
	}
	return 1
}

func (s *dialState) getThetaForTime(time TimeOfDay) float64 {
	w := s.Widget()
	hoursFactor := HoursPerPeriod
	if w.hourDialType == hourDialTwentyFourHour {
		hoursFactor = HoursPerDay
	}
	var fraction float64
	if w.hourMinuteMode == hourMinuteModeHour {
		fraction = dartModFloat(float64(time.Hour)/float64(hoursFactor), float64(hoursFactor))
	} else {
		fraction = dartModFloat(float64(time.Minute)/MinutesPerHour, MinutesPerHour)
	}
	return dartModFloat(math.Pi/2-fraction*twoPi, twoPi)
}

func (s *dialState) getTimeForTheta(theta float64, roundMinutes bool, radius float64) TimeOfDay {
	w := s.Widget()
	fraction := dartModFloat(0.25-dartModFloat(theta, twoPi)/twoPi, 1)
	if w.hourMinuteMode == hourMinuteModeHour {
		var newHour int
		switch w.hourDialType {
		case hourDialTwentyFourHour:
			newHour = mod(int(math.Round(fraction*HoursPerDay)), HoursPerDay)
		case hourDialTwentyFourHourDoubleRing:
			newHour = mod(int(math.Round(fraction*HoursPerPeriod)), HoursPerPeriod)
			if radius < 0.5 {
				newHour += HoursPerPeriod
			}
		default:
			newHour = mod(int(math.Round(fraction*HoursPerPeriod)), HoursPerPeriod)
			newHour += w.selectedTime.PeriodOffset()
		}
		return w.selectedTime.Replacing(TimeOfDayOpts{Hour: foundation.Some(newHour)})
	}
	minute := mod(int(math.Round(fraction*MinutesPerHour)), MinutesPerHour)
	if roundMinutes {
		// Round the minutes to the nearest five minute interval.
		minute = ((minute + 2) / 5 * 5) % MinutesPerHour
	}
	return w.selectedTime.Replacing(TimeOfDayOpts{Minute: foundation.Some(minute)})
}

func (s *dialState) notifyOnChangedIfNeeded(roundMinutes bool) TimeOfDay {
	w := s.Widget()
	current := s.getTimeForTheta(s.theta.Value(), roundMinutes, s.radius.Value())
	if w.onChanged == nil {
		return current
	}
	if current != w.selectedTime {
		w.onChanged(current)
	}
	return current
}

func (s *dialState) updateThetaForPan(roundMinutes bool) {
	s.SetState(func() {
		offset := s.position.Sub(s.center)
		labelRadius := s.dialSize.ShortestSide()/2 - timePickerDialPadding
		innerRadius := labelRadius - timePickerInnerDialOffset
		angle := dartModFloat(math.Atan2(offset.Dx, offset.Dy)-math.Pi/2, twoPi)
		radius := ui.ClampDouble((offset.Distance()-innerRadius)/timePickerInnerDialOffset, 0, 1)
		if roundMinutes {
			angle = s.getThetaForTime(s.getTimeForTheta(angle, roundMinutes, radius))
		}
		s.thetaTween.Begin, s.thetaTween.End = angle, angle
		s.radiusTween.Begin, s.radiusTween.End = radius, radius
	})
}

func (s *dialState) box() rendering.Box { return s.Context().FindRenderObject().(rendering.Box) }

func (s *dialState) handlePanStart(details gestures.DragStartDetails) {
	s.dragging = true
	box := s.box()
	s.position = box.GlobalToLocal(details.GlobalPosition, nil)
	s.dialSize = box.Size()
	s.center = s.dialSize.Center(ui.OffsetZero)
	s.updateThetaForPan(false)
	s.notifyOnChangedIfNeeded(false)
}

func (s *dialState) handlePanUpdate(details gestures.DragUpdateDetails) {
	s.position = s.position.Add(details.Delta)
	s.updateThetaForPan(false)
	s.notifyOnChangedIfNeeded(false)
}

func (s *dialState) handlePanEnd(gestures.DragEndDetails) {
	w := s.Widget()
	s.dragging = false
	s.animateTo(s.getThetaForTime(w.selectedTime), s.getRadiusForTime(w.selectedTime))
	if w.hourMinuteMode == hourMinuteModeHour && w.onHourSelected != nil {
		w.onHourSelected()
	}
}

func (s *dialState) handleTapUp(details gestures.TapUpDetails) {
	w := s.Widget()
	box := s.box()
	s.position = box.GlobalToLocal(details.GlobalPosition, nil)
	s.center = box.Size().Center(ui.OffsetZero)
	s.dialSize = box.Size()
	s.updateThetaForPan(true)
	s.notifyOnChangedIfNeeded(true)
	if w.hourMinuteMode == hourMinuteModeHour && w.onHourSelected != nil {
		w.onHourSelected()
	}
	time := s.getTimeForTheta(s.theta.Value(), true, s.radius.Value())
	s.animateTo(s.getThetaForTime(time), s.getRadiusForTime(time))
	s.dragging = false
}

func (s *dialState) selectHour(hour int) {
	w := s.Widget()
	amPmTime := func() TimeOfDay {
		if w.selectedTime.Period() == DayPeriodAM {
			return TimeOfDay{Hour: hour, Minute: w.selectedTime.Minute}
		}
		return TimeOfDay{Hour: hour + HoursPerPeriod, Minute: w.selectedTime.Minute}
	}
	var time TimeOfDay
	if w.hourMinuteMode == hourMinuteModeHour && w.hourDialType != hourDialTwelveHour {
		time = TimeOfDay{Hour: hour, Minute: w.selectedTime.Minute}
	} else {
		time = amPmTime()
	}
	angle := s.getThetaForTime(time)
	s.thetaTween.Begin, s.thetaTween.End = angle, angle
	s.notifyOnChangedIfNeeded(false)
}

func (s *dialState) selectMinute(minute int) {
	time := TimeOfDay{Hour: s.Widget().selectedTime.Hour, Minute: minute}
	angle := s.getThetaForTime(time)
	s.thetaTween.Begin, s.thetaTween.End = angle, angle
	s.notifyOnChangedIfNeeded(false)
}

var (
	// dialAmHours are the hours of the 12 hour dial.
	dialAmHours = []TimeOfDay{{12, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}, {7, 0}, {8, 0}, {9, 0}, {10, 0}, {11, 0}}
	// dialTwentyFourHoursM2 are the hours of the 24 hour dial of Material 2.
	dialTwentyFourHoursM2 = []TimeOfDay{{0, 0}, {2, 0}, {4, 0}, {6, 0}, {8, 0}, {10, 0}, {12, 0}, {14, 0}, {16, 0}, {18, 0}, {20, 0}, {22, 0}}
)

func (s *dialState) buildTappableLabel(bc widgets.BuildContext, textStyle *painting.TextStyle, value int, inner bool, label string, onTap func()) *tappableLabel {
	painter := painting.NewTextPainter(painting.TextPainterOpts{
		TextEngine:    textEngineOf(bc),
		Text:          painting.NewTextSpan(painting.TextSpanOpts{Style: textStyle, Text: foundation.Some(label)}),
		TextDirection: ui.TextDirectionLTR,
		TextScaler:    widgets.MediaQueryTextScalerOf(bc).Clamp(painting.TextScalerClampOpts{MaxScaleFactor: foundation.Some(2.0)}),
	})
	painter.Layout(0, math.Inf(1))
	return &tappableLabel{value: value, inner: inner, painter: painter, onTap: onTap}
}

func (s *dialState) build24HourRing(bc widgets.BuildContext, textStyle *painting.TextStyle) []*tappableLabel {
	var labels []*tappableLabel
	if s.themeData.UseMaterial3 {
		for hour := 0; hour < HoursPerDay; hour++ {
			timeOfDay := TimeOfDay{Hour: hour}
			label := s.localizations.FormatDecimal(hour)
			if hour == 0 {
				label = s.localizations.FormatHour(timeOfDay, true)
			}
			h := hour
			labels = append(labels, s.buildTappableLabel(bc, textStyle, hour, hour >= 12, label, func() { s.selectHour(h) }))
		}
		return labels
	}
	for _, timeOfDay := range dialTwentyFourHoursM2 {
		h := timeOfDay.Hour
		labels = append(labels, s.buildTappableLabel(bc, textStyle, h, false, s.localizations.FormatHour(timeOfDay, true), func() { s.selectHour(h) }))
	}
	return labels
}

func (s *dialState) build12HourRing(bc widgets.BuildContext, textStyle *painting.TextStyle) []*tappableLabel {
	var labels []*tappableLabel
	for _, timeOfDay := range dialAmHours {
		h := timeOfDay.Hour
		labels = append(labels, s.buildTappableLabel(bc, textStyle, h, false,
			s.localizations.FormatHour(timeOfDay, widgets.MediaQueryAlwaysUse24HourFormatOf(bc)), func() { s.selectHour(h) }))
	}
	return labels
}

func (s *dialState) buildMinutes(bc widgets.BuildContext, textStyle *painting.TextStyle) []*tappableLabel {
	var labels []*tappableLabel
	for minute := 0; minute < MinutesPerHour; minute += 5 {
		m := minute
		labels = append(labels, s.buildTappableLabel(bc, textStyle, minute, false, s.localizations.FormatMinute(TimeOfDay{Minute: minute}), func() { s.selectMinute(m) }))
	}
	return labels
}

func (s *dialState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	debugDialTimePickerEntryMode(bc)
	theme := ThemeOf(bc)
	timePickerTheme := TimePickerThemeOf(bc)
	defaultTheme := timePickerDefaultsOf(bc, TimePickerEntryModeDial)
	backgroundColor := firstOpt(timePickerTheme.DialBackgroundColor, foundation.Some(defaultTheme.dialBackgroundColor)).Value()
	dialHandColor := firstOpt(timePickerTheme.DialHandColor, foundation.Some(defaultTheme.dialHandColor)).Value()
	labelStyle := firstNonNil(timePickerTheme.DialTextStyle, defaultTheme.dialTextStyle)
	selected := widgets.NewWidgetStates(widgets.WidgetStateSelected)
	dialTextUnselectedColor := timePickerThemeColor(timePickerTheme.DialTextColor, timePickerTheme.DialTextStateColor, defaultTheme.dialTextColor, 0)
	dialTextSelectedColor := timePickerThemeColor(timePickerTheme.DialTextColor, timePickerTheme.DialTextStateColor, defaultTheme.dialTextColor, selected)
	unselectedStyle := labelStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(dialTextUnselectedColor)})
	selectedStyle := labelStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(dialTextSelectedColor)})

	var primaryLabels, selectedLabels []*tappableLabel
	var selectedDialValue int
	radiusValue := 1.0
	switch {
	case w.hourMinuteMode == hourMinuteModeHour && w.hourDialType != hourDialTwelveHour:
		selectedDialValue = w.selectedTime.Hour
		primaryLabels = s.build24HourRing(bc, unselectedStyle)
		selectedLabels = s.build24HourRing(bc, selectedStyle)
		if theme.UseMaterial3 {
			radiusValue = s.radius.Value()
		}
	case w.hourMinuteMode == hourMinuteModeHour:
		selectedDialValue = w.selectedTime.HourOfPeriod()
		primaryLabels = s.build12HourRing(bc, unselectedStyle)
		selectedLabels = s.build12HourRing(bc, selectedStyle)
	default:
		selectedDialValue = w.selectedTime.Minute
		primaryLabels = s.buildMinutes(bc, unselectedStyle)
		selectedLabels = s.buildMinutes(bc, selectedStyle)
	}
	if s.painter != nil {
		s.painter.dispose()
	}
	s.painter = &dialPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(nil),
		selectedValue:     selectedDialValue, primaryLabels: primaryLabels, selectedLabels: selectedLabels,
		backgroundColor: backgroundColor, handColor: dialHandColor, handWidth: defaultTheme.handWidth, dotColor: dialTextSelectedColor,
		dotRadius: defaultTheme.dotRadius, centerRadius: defaultTheme.centerRadius, theta: s.theta.Value(), radius: radiusValue,
		textDirection: widgets.DirectionalityOf(bc),
	}
	return widgets.NewGestureDetector(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: s.painter}), widgets.GestureDetectorOpts{
		ExcludeFromSemantics: true, OnPanStart: s.handlePanStart, OnPanUpdate: s.handlePanUpdate, OnPanEnd: s.handlePanEnd, OnTapUp: s.handleTapUp,
	})
}

// timePickerInput is the input mode of the time picker: two text fields.
//
// Flutter: material/time_picker.dart _TimePickerInput
type timePickerInput struct {
	widgets.StatefulWidget
	initialSelectedTime TimeOfDay
	errorInvalidText    string
	hourLabelText       string
	minuteLabelText     string
	helpText            string
	autofocusHour       foundation.Opt[bool]
	autofocusMinute     foundation.Opt[bool]
	emptyInitialTime    bool
}

func (w *timePickerInput) CreateState() widgets.State { return &timePickerInputState{} }

type timePickerInputState struct {
	widgets.StateOf[*timePickerInput]
	selectedTime   TimeOfDay
	hourHasError   bool
	minuteHasError bool
}

func (s *timePickerInputState) InitState() { s.selectedTime = s.Widget().initialSelectedTime }

func (s *timePickerInputState) parseHour(value string) (int, bool) {
	newHour, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	if widgets.MediaQueryAlwaysUse24HourFormatOf(s.Context()) {
		if newHour >= 0 && newHour < 24 {
			return newHour, true
		}
		return 0, false
	}
	if newHour > 0 && newHour < 13 {
		if (s.selectedTime.Period() == DayPeriodPM && newHour != 12) || (s.selectedTime.Period() == DayPeriodAM && newHour == 12) {
			newHour = (newHour + HoursPerPeriod) % HoursPerDay
		}
		return newHour, true
	}
	return 0, false
}

func parseMinute(value string) (int, bool) {
	newMinute, err := strconv.Atoi(value)
	if err != nil || newMinute < 0 || newMinute >= 60 {
		return 0, false
	}
	return newMinute, true
}

func (s *timePickerInputState) handleHourSavedSubmitted(value foundation.Opt[string]) {
	if newHour, ok := s.parseHour(value.OrElse("")); ok {
		s.selectedTime = TimeOfDay{Hour: newHour, Minute: s.selectedTime.Minute}
		setTimePickerSelectedTime(s.Context(), s.selectedTime)
		widgets.FocusScopeOf(s.Context()).RequestFocus()
	}
}

func (s *timePickerInputState) handleHourChanged(value string) {
	if _, ok := s.parseHour(value); ok && len(value) == 2 {
		widgets.FocusScopeOf(s.Context()).NextFocus()
	}
}

func (s *timePickerInputState) handleMinuteSavedSubmitted(value foundation.Opt[string]) {
	if newMinute, ok := parseMinute(value.OrElse("")); ok {
		s.selectedTime = TimeOfDay{Hour: s.selectedTime.Hour, Minute: newMinute}
		setTimePickerSelectedTime(s.Context(), s.selectedTime)
		widgets.FocusScopeOf(s.Context()).Unfocus()
	}
}

func (s *timePickerInputState) handleDayPeriodChanged(value TimeOfDay) {
	s.selectedTime = value
	setTimePickerSelectedTime(s.Context(), s.selectedTime)
}

func (s *timePickerInputState) validateHour(value foundation.Opt[string]) foundation.Opt[string] {
	_, ok := s.parseHour(value.OrElse(""))
	s.SetState(func() { s.hourHasError = !ok })
	if !ok {
		return foundation.Some("")
	}
	return foundation.None[string]()
}

func (s *timePickerInputState) validateMinute(value foundation.Opt[string]) foundation.Opt[string] {
	_, ok := parseMinute(value.OrElse(""))
	s.SetState(func() { s.minuteHasError = !ok })
	if !ok {
		return foundation.Some("")
	}
	return foundation.None[string]()
}

func (s *timePickerInputState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	widgets.DebugCheckHasMediaQuery(bc)
	l := MaterialLocalizationsOf(bc)
	timeOfDayFormat := l.TimeOfDayFormat(timePickerUse24HourFormatOf(bc))
	use24HourDials := HourFormatOf(timeOfDayFormat) != HourFormatLowerH
	theme := ThemeOf(bc)
	timePickerTheme := timePickerThemeOf(bc)
	defaultTheme := timePickerDefaultThemeOf(bc)
	hourMinuteStyle := firstNonNil(timePickerTheme.HourMinuteTextStyle, defaultTheme.hourMinuteTextStyle)
	minInteractiveVerticalPadding := math.Max(0, 2*MinInteractiveDimension-defaultTheme.dayPeriodInputSize.Height)
	useMaterial3 := timePickerUseMaterial3Of(bc)
	helpBottom := 24.0
	if useMaterial3 {
		helpBottom = 20
	}
	var outerPadding painting.EdgeInsetsGeometry = painting.NewEdgeInsetsSymmetric(0, 16)
	if useMaterial3 {
		outerPadding = painting.EdgeInsetsZero
	}
	hasError := s.hourHasError || s.minuteHasError

	field := func(isHour bool, labelText string, defaultLabel string, autofocus foundation.Opt[bool]) widgets.Widget {
		var f widgets.Widget
		if isHour {
			f = newHourMinuteTextField(hourMinuteTextFieldArgs{
				selectedTime: s.selectedTime, isHour: true, autofocus: autofocus, inputAction: services.TextInputActionNext, style: hourMinuteStyle,
				semanticHintText: orDefault(labelText, defaultLabel), validator: s.validateHour, onSavedSubmitted: s.handleHourSavedSubmitted,
				emptyInitialTime: w.emptyInitialTime, onChanged: s.handleHourChanged,
			})
		} else {
			f = newHourMinuteTextField(hourMinuteTextFieldArgs{
				selectedTime: s.selectedTime, isHour: false, autofocus: autofocus, inputAction: services.TextInputActionDone, style: hourMinuteStyle,
				semanticHintText: orDefault(labelText, defaultLabel), validator: s.validateMinute, onSavedSubmitted: s.handleMinuteSavedSubmitted,
				emptyInitialTime: w.emptyInitialTime,
			})
		}
		children := []widgets.Widget{widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, 0, 0, 10), f)}
		if !hasError {
			children = append(children, widgets.NewExcludeSemantics(widgets.NewText(orDefault(labelText, defaultLabel),
				widgets.TextOpts{Style: theme.TextTheme.BodySmall, MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis})))
		}
		return widgets.NewExpanded(widgets.NewColumn(children, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}))
	}

	var row []widgets.Widget
	if !use24HourDials && timeOfDayFormat == TimeOfDayFormatASpaceHColonMM {
		row = append(row, widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, 12, 0), newDayPeriodControl(s.handleDayPeriodChanged)))
	}
	row = append(row, widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, minInteractiveVerticalPadding/2, 0, 0),
		widgets.NewRow([]widgets.Widget{
			field(true, w.hourLabelText, l.TimePickerHourLabel(), w.autofocusHour),
			newTimeSelectorSeparator(timeOfDayFormat),
			field(false, w.minuteLabelText, l.TimePickerMinuteLabel(), w.autofocusMinute),
		}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart, TextDirection: ui.TextDirectionLTR}))))
	if !use24HourDials && timeOfDayFormat != TimeOfDayFormatASpaceHColonMM {
		row = append(row, widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(12, 0, 0, 0), newDayPeriodControl(s.handleDayPeriodChanged)))
	}
	var footer widgets.Widget = widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2.0)})
	if hasError {
		footer = widgets.NewText(orDefault(w.errorInvalidText, l.InvalidTimeLabel()), widgets.TextOpts{
			Style: theme.TextTheme.BodyMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.ColorScheme.Error())})})
	}
	return widgets.NewPadding(outerPadding, widgets.NewColumn([]widgets.Widget{
		widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(0, 0, 0, helpBottom-minInteractiveVerticalPadding/2),
			widgets.NewText(w.helpText, widgets.TextOpts{Style: firstNonNil(timePickerTheme.HelpTextStyle, defaultTheme.helpTextStyle)})),
		widgets.NewRow(row, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}),
		footer,
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStart}))
}

// hourMinuteTextFieldArgs are the parameters of hourMinuteTextField.
type hourMinuteTextFieldArgs struct {
	selectedTime     TimeOfDay
	isHour           bool
	autofocus        foundation.Opt[bool]
	inputAction      services.TextInputAction
	style            *painting.TextStyle
	semanticHintText string
	validator        widgets.FormFieldValidator[string]
	onSavedSubmitted func(foundation.Opt[string])
	onChanged        func(string)
	emptyInitialTime bool
}

// hourMinuteTextField is the text field of the hour or the minute.
//
// Flutter: material/time_picker.dart _HourMinuteTextField
type hourMinuteTextField struct {
	widgets.StatefulWidget
	a hourMinuteTextFieldArgs
}

func newHourMinuteTextField(a hourMinuteTextFieldArgs) *hourMinuteTextField {
	return &hourMinuteTextField{StatefulWidget: widgets.NewStatefulWidgetBase(), a: a}
}

func (w *hourMinuteTextField) CreateState() widgets.State { return &hourMinuteTextFieldState{} }

type hourMinuteTextFieldState struct {
	widgets.StateOf[*hourMinuteTextField]
	controller           *widgets.TextEditingController
	controllerHasBeenSet bool
	focusNode            *widgets.FocusNode
}

func (s *hourMinuteTextFieldState) InitState() {
	s.controller = widgets.NewTextEditingController()
	s.focusNode = widgets.NewFocusNode()
	s.focusNode.AddListener(func() { s.SetState(func() {}) })
}

func (s *hourMinuteTextFieldState) DidChangeDependencies() {
	if !s.controllerHasBeenSet {
		s.controllerHasBeenSet = true
		text := ""
		if !s.Widget().a.emptyInitialTime {
			text = s.formattedValue()
		}
		s.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text}))
	}
}

func (s *hourMinuteTextFieldState) Dispose() {
	s.controller.Dispose()
	s.focusNode.Dispose()
}

func (s *hourMinuteTextFieldState) formattedValue() string {
	a := s.Widget().a
	l := MaterialLocalizationsOf(s.Context())
	if !a.isHour {
		return l.FormatMinute(a.selectedTime)
	}
	return l.FormatHour(a.selectedTime, widgets.MediaQueryAlwaysUse24HourFormatOf(s.Context()))
}

func (s *hourMinuteTextFieldState) Build(bc widgets.BuildContext) widgets.Widget {
	a := s.Widget().a
	theme := ThemeOf(bc)
	timePickerTheme := TimePickerThemeOf(bc)
	defaultTheme := timePickerDefaultsOf(bc, TimePickerEntryModeInput)
	alwaysUse24HourFormat := widgets.MediaQueryAlwaysUse24HourFormatOf(bc)
	inputDecorationTheme := defaultTheme.inputDecorationTheme
	if timePickerTheme.InputDecorationTheme != nil {
		inputDecorationTheme = *timePickerTheme.InputDecorationTheme
	}
	inputDecoration := InputDecoration{ErrorStyle: defaultTheme.inputDecorationTheme.ErrorStyle}.ApplyDefaults(inputDecorationTheme)
	hasFocus := s.focusNode.HasFocus()
	if !hasFocus && !a.emptyInitialTime {
		inputDecoration.HintText = foundation.Some(s.formattedValue())
	} else {
		inputDecoration.HintText = foundation.None[string]()
	}
	var startingFillColor widgets.WidgetStateProperty[ui.Color]
	switch {
	case timePickerTheme.InputDecorationTheme != nil && timePickerTheme.InputDecorationTheme.FillColor != nil:
		startingFillColor = timePickerTheme.InputDecorationTheme.FillColor
	case timePickerTheme.HourMinuteStateColor != nil || timePickerTheme.HourMinuteColor.IsSome():
		startingFillColor = stateColor(func(states widgets.WidgetStates) ui.Color {
			return timePickerThemeColor(timePickerTheme.HourMinuteColor, timePickerTheme.HourMinuteStateColor, defaultTheme.hourMinuteColor, states)
		})
	default:
		startingFillColor = defaultTheme.hourMinuteColor
	}
	var states widgets.WidgetStates
	if hasFocus {
		states = states.With(widgets.WidgetStateFocused).With(widgets.WidgetStateSelected)
	}
	var fillColor ui.Color
	if theme.UseMaterial3 {
		fillColor = startingFillColor.Resolve(states)
	} else if hasFocus {
		fillColor = Colors.Transparent
	} else {
		fillColor = startingFillColor.Resolve(0)
	}
	inputDecoration.FillColor = widgets.NewWidgetStatePropertyAll(fillColor)
	effectiveTextColor := timePickerThemeColor(timePickerTheme.HourMinuteTextColor, timePickerTheme.HourMinuteTextStateColor, defaultTheme.hourMinuteTextColor, states)
	effectiveStyle := a.style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(effectiveTextColor)})
	size := defaultTheme.hourMinuteInputSize
	if alwaysUse24HourFormat {
		size = defaultTheme.hourMinuteInputSize24
	}
	return widgets.NewSizedBoxFromSize(size, widgets.NewMediaQueryWithNoTextScaling(widgets.NewSemantics(NewTextFormField(TextFormFieldOpts{
		TextFieldOpts: TextFieldOpts{
			Autofocus: a.autofocus.OrElse(false), Expands: true, MaxLines: foundation.Some(rendering.MaxLinesUnlimited),
			InputFormatters: []services.TextInputFormatter{services.NewLengthLimitingTextInputFormatter(foundation.Some(2))},
			FocusNode:       s.focusNode, TextAlign: ui.TextAlignCenter, TextInputAction: a.inputAction,
			KeyboardType: foundation.Some(services.TextInputTypeNumber), Style: effectiveStyle, Controller: s.controller,
			Decoration: &inputDecoration, OnEditingComplete: func() { a.onSavedSubmitted(foundation.Some(s.controller.Text())) },
			OnChanged: a.onChanged,
		},
		Validator: a.validator, OnSaved: a.onSavedSubmitted,
		OnFieldSubmitted: func(text string) { a.onSavedSubmitted(foundation.Some(text)) },
	}), widgets.SemanticsOpts{Label: foundation.Some(a.semanticHintText)})))
}
