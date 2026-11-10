package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoTimerPickerMode is the mode of a CupertinoTimerPicker: the columns that it shows.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPickerMode
type CupertinoTimerPickerMode int

const (
	// CupertinoTimerPickerModeUnspecified is the default mode, HMS.
	//
	// Flutter: no counterpart
	CupertinoTimerPickerModeUnspecified CupertinoTimerPickerMode = iota
	// CupertinoTimerPickerModeHM shows the hours and the minutes.
	//
	// Flutter: cupertino/date_picker.dart CupertinoTimerPickerMode.hm
	CupertinoTimerPickerModeHM
	// CupertinoTimerPickerModeMS shows the minutes and the seconds.
	//
	// Flutter: cupertino/date_picker.dart CupertinoTimerPickerMode.ms
	CupertinoTimerPickerModeMS
	// CupertinoTimerPickerModeHMS shows the hours, the minutes and the seconds.
	//
	// Flutter: cupertino/date_picker.dart CupertinoTimerPickerMode.hms
	CupertinoTimerPickerModeHMS
)

// String is the name of the mode, like Dart's enum toString.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPickerMode
func (m CupertinoTimerPickerMode) String() string {
	switch m {
	case CupertinoTimerPickerModeHM:
		return "CupertinoTimerPickerMode.hm"
	case CupertinoTimerPickerModeMS:
		return "CupertinoTimerPickerMode.ms"
	}
	return "CupertinoTimerPickerMode.hms"
}

// CupertinoTimerPickerOpts are the optional parameters of NewCupertinoTimerPicker.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker.new
type CupertinoTimerPickerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Mode is the columns that the picker shows; the default is HMS.
	Mode CupertinoTimerPickerMode
	// InitialTimerDuration is the duration that the picker starts at, less than a day; the default is zero.
	InitialTimerDuration time.Duration
	// MinuteInterval is the difference between two minutes of the minute column; it must be a factor of 60. The default is 1.
	MinuteInterval int
	// SecondInterval is the difference between two seconds of the second column; it must be a factor of 60. The default is 1.
	SecondInterval int
	// Alignment is where the picker is in the space that it is given; the default is the center.
	Alignment painting.AlignmentGeometry
	// BackgroundColor is the background of the picker.
	BackgroundColor ui.ColorValue
	// ItemExtent is the height of an item; the default is 32.
	ItemExtent float64
	// ChangeReportingBehavior is when OnTimerDurationChanged is called; the default is on every scroll update.
	ChangeReportingBehavior widgets.ChangeReportingBehavior
	// SelectionOverlayBuilder builds the selection overlay of each column; it may return nil.
	SelectionOverlayBuilder SelectionOverlayBuilder
}

// CupertinoTimerPicker is an iOS style countdown timer picker: wheels of hours, minutes and seconds with their labels.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker
type CupertinoTimerPicker struct {
	widgets.StatefulWidget
	onTimerDurationChanged func(time.Duration)
	o                      CupertinoTimerPickerOpts
}

// NewCupertinoTimerPicker creates a timer picker that calls onTimerDurationChanged with each duration that the user selects. In
// debug builds it panics with the tag CupertinoTimerPickerDurationOutOfRange, CupertinoTimerPickerMinuteIntervalInvalid,
// CupertinoTimerPickerSecondIntervalInvalid, CupertinoTimerPickerDurationNotOnInterval or CupertinoTimerPickerItemExtentNotPositive
// when an argument breaks the rule of that name.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker.new
func NewCupertinoTimerPicker(onTimerDurationChanged func(time.Duration), opts ...CupertinoTimerPickerOpts) *CupertinoTimerPicker {
	o := firstOf(opts)
	if o.Mode == CupertinoTimerPickerModeUnspecified {
		o.Mode = CupertinoTimerPickerModeHMS
	}
	if o.MinuteInterval == 0 {
		o.MinuteInterval = 1
	}
	if o.SecondInterval == 0 {
		o.SecondInterval = 1
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.ItemExtent == 0 {
		o.ItemExtent = pickerItemExtent
	}
	if foundation.DebugMode {
		fail := func(tag, message string, kv ...any) { panic(errors.New(tag, message, kv...)) }
		switch {
		case o.InitialTimerDuration < 0 || o.InitialTimerDuration >= 24*time.Hour:
			fail(tagCupertinoTimerPickerDurationOutOfRange, "initialTimerDuration is not at least zero and less than a day", "initialTimerDuration", o.InitialTimerDuration.String())
		case o.MinuteInterval <= 0 || 60%o.MinuteInterval != 0:
			fail(tagCupertinoTimerPickerMinuteIntervalInvalid, "minute interval is not a positive integer factor of 60", "minuteInterval", o.MinuteInterval)
		case o.SecondInterval <= 0 || 60%o.SecondInterval != 0:
			fail(tagCupertinoTimerPickerSecondIntervalInvalid, "second interval is not a positive integer factor of 60", "secondInterval", o.SecondInterval)
		case int(o.InitialTimerDuration.Minutes())%o.MinuteInterval != 0:
			fail(tagCupertinoTimerPickerDurationNotOnInterval, "initial minute is not divisible by minute interval", "minuteInterval", o.MinuteInterval)
		case int(o.InitialTimerDuration.Seconds())%o.SecondInterval != 0:
			fail(tagCupertinoTimerPickerDurationNotOnInterval, "initial second is not divisible by second interval", "secondInterval", o.SecondInterval)
		case o.ItemExtent <= 0:
			fail(tagCupertinoTimerPickerItemExtentNotPositive, "item extent should be greater than 0", "itemExtent", o.ItemExtent)
		}
	}
	return &CupertinoTimerPicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onTimerDurationChanged: onTimerDurationChanged, o: o}
}

// Mode is the columns that the picker shows.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker.mode
func (w *CupertinoTimerPicker) Mode() CupertinoTimerPickerMode { return w.o.Mode }

// ItemExtent is the height of an item.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker.itemExtent
func (w *CupertinoTimerPicker) ItemExtent() float64 { return w.o.ItemExtent }

// CreateState creates the state.
//
// Flutter: cupertino/date_picker.dart CupertinoTimerPicker.createState
func (w *CupertinoTimerPicker) CreateState() widgets.State { return &timerPickerState{} }

// timerPickerState is the state of the timer picker.
//
// Flutter: cupertino/date_picker.dart _CupertinoTimerPickerState
type timerPickerState struct {
	widgets.StateOf[*CupertinoTimerPicker]
	textDirection       ui.TextDirection
	localizations       CupertinoLocalizations
	selectedHour        foundation.Opt[int]
	selectedMinute      int
	selectedSecond      foundation.Opt[int]
	lastSelectedHour    foundation.Opt[int]
	lastSelectedMinute  foundation.Opt[int]
	lastSelectedSecond  foundation.Opt[int]
	textPainter         *painting.TextPainter
	numbers             []string
	numberLabelWidth    float64
	numberLabelHeight   float64
	numberLabelBaseline float64
	hourLabelWidth      float64
	minuteLabelWidth    float64
	secondLabelWidth    float64
	totalWidth          float64
	pickerColumnWidth   float64

	hourController, minuteController, secondController *widgets.FixedExtentScrollController
	fontsHandle                                        foundation.ListenerHandle
}

func (s *timerPickerState) textDirectionFactor() int {
	if s.textDirection == ui.TextDirectionRTL {
		return -1
	}
	return 1
}

func (s *timerPickerState) InitState() {
	w := s.Widget()
	s.selectedMinute = int(w.o.InitialTimerDuration.Minutes()) % 60
	if w.o.Mode != CupertinoTimerPickerModeMS {
		s.selectedHour = foundation.Some(int(w.o.InitialTimerDuration.Hours()))
	}
	if w.o.Mode != CupertinoTimerPickerModeHM {
		s.selectedSecond = foundation.Some(int(w.o.InitialTimerDuration.Seconds()) % 60)
	}
	for i := range 10 {
		s.numbers = append(s.numbers, string(rune('0'+9-i)))
	}
	s.textPainter = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: widgets.WidgetsBindingOf(s.Context()).Painting().TextEngine()})
	s.fontsHandle = widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().AddListener(func() {
		s.SetState(func() {
			s.textPainter.MarkNeedsLayout()
			s.measureLabelMetrics()
		})
	})
}

func (s *timerPickerState) Dispose() {
	widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().RemoveListener(s.fontsHandle)
	s.textPainter.Dispose()
	for _, c := range []*widgets.FixedExtentScrollController{s.hourController, s.minuteController, s.secondController} {
		if c != nil {
			c.Dispose()
		}
	}
}

func (s *timerPickerState) DidUpdateWidget(old *CupertinoTimerPicker) {
	if foundation.DebugMode && old.o.Mode != s.Widget().o.Mode {
		panic(errors.New(tagCupertinoTimerPickerModeChanged, "The CupertinoTimerPicker's mode cannot change once it's built", "was", old.o.Mode.String(), "is", s.Widget().o.Mode.String()))
	}
}

func (s *timerPickerState) DidChangeDependencies() {
	s.textDirection = widgets.DirectionalityOf(s.Context())
	s.localizations = CupertinoLocalizationsOf(s.Context())
	s.measureLabelMetrics()
}

func (s *timerPickerState) measureLabelMetrics() {
	mode := s.Widget().o.Mode
	s.textPainter.SetTextDirection(s.textDirection)
	style := s.textStyleFrom(timerPickerMagnification)
	maxWidth := math.Inf(-1)
	widest := ""
	for _, input := range s.numbers {
		s.textPainter.SetText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(input), Style: style}))
		s.textPainter.Layout(0, math.Inf(1))
		if w := s.textPainter.MaxIntrinsicWidth(); w > maxWidth {
			maxWidth, widest = w, input
		}
	}
	s.textPainter.SetText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(widest + widest), Style: style}))
	s.textPainter.Layout(0, math.Inf(1))
	s.numberLabelWidth = s.textPainter.MaxIntrinsicWidth()
	s.numberLabelHeight = s.textPainter.Height()
	s.numberLabelBaseline = s.textPainter.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic)
	s.minuteLabelWidth = s.measureLabelsMaxWidth(s.localizations.TimerPickerMinuteLabels(), style)
	if mode != CupertinoTimerPickerModeMS {
		s.hourLabelWidth = s.measureLabelsMaxWidth(s.localizations.TimerPickerHourLabels(), style)
	}
	if mode != CupertinoTimerPickerModeHM {
		s.secondLabelWidth = s.measureLabelsMaxWidth(s.localizations.TimerPickerSecondLabels(), style)
	}
}

func (s *timerPickerState) measureLabelsMaxWidth(labels []string, style *painting.TextStyle) float64 {
	maxWidth := math.Inf(-1)
	for _, label := range labels {
		s.textPainter.SetText(painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(label), Style: style}))
		s.textPainter.Layout(0, math.Inf(1))
		maxWidth = math.Max(maxWidth, s.textPainter.MaxIntrinsicWidth())
	}
	return maxWidth
}

func (s *timerPickerState) buildLabel(text string, pickerPadding painting.EdgeInsetsDirectional) widgets.Widget {
	padding := painting.NewEdgeInsetsDirectionalFromSTEB(s.numberLabelWidth+timerPickerLabelPadSize+pickerPadding.Start, 0, 0, 0)
	return widgets.NewIgnorePointer(widgets.NewPadding(padding.Resolve(s.textDirection),
		widgets.NewAlign(widgets.NewSizedBox(widgets.NewBaseline(s.numberLabelBaseline, ui.TextBaselineAlphabetic,
			widgets.NewText(text, widgets.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(timerPickerLabelFontSize), FontWeight: ui.FontWeightW600}),
				MaxLines: foundation.Some(1), SoftWrap: foundation.Some(false)})),
			widgets.SizedBoxOpts{Height: foundation.Some(s.numberLabelHeight)}),
			widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart.Resolve(s.textDirection)})))
}

func (s *timerPickerState) buildPickerNumberLabel(text string, padding painting.EdgeInsetsDirectional) widgets.Widget {
	return widgets.NewSizedBox(widgets.NewPadding(padding.Resolve(s.textDirection),
		widgets.NewAlign(widgets.NewSizedBox(widgets.NewAlign(widgets.NewText(text, widgets.TextOpts{SoftWrap: foundation.Some(false), MaxLines: foundation.Some(1), Overflow: painting.TextOverflowVisible}),
			widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd.Resolve(s.textDirection)}),
			widgets.SizedBoxOpts{Width: foundation.Some(s.numberLabelWidth)}),
			widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart.Resolve(s.textDirection)})),
		widgets.SizedBoxOpts{Width: foundation.Some(timerPickerColumnIntrinsicWidth + padding.Horizontal())})
}

// semanticsItem is an item of a column, whose semantics label has the number and its unit in the order of the text direction.
func (s *timerPickerState) semanticsItem(number string, label string, padding painting.EdgeInsetsDirectional) widgets.Widget {
	semanticsLabel := number + label
	if s.textDirectionFactor() != 1 {
		semanticsLabel = label + number
	}
	return widgets.NewSemantics(s.buildPickerNumberLabel(number, padding), widgets.SemanticsOpts{Label: foundation.Some(semanticsLabel), ExcludeSemantics: true})
}

func padStart(padding painting.EdgeInsetsDirectional) painting.EdgeInsetsDirectional {
	return painting.NewEdgeInsetsDirectionalFromSTEB(math.Max(padding.Start, 0), 0, math.Max(padding.End, 0), 0)
}

// column is a column of the picker: a wheel with the label of its unit over it. The label follows the last item that a
// scroll ended on.
func (s *timerPickerState) column(additionalPadding painting.EdgeInsetsDirectional, wheel func(painting.EdgeInsetsDirectional) widgets.Widget, label func() string, scrollEnded func()) widgets.Widget {
	additionalPadding = padStart(additionalPadding)
	return widgets.NewStack([]widgets.Widget{
		widgets.NewNotificationListener(func(*widgets.ScrollEndNotification) bool {
			s.SetState(scrollEnded)
			return false
		}, wheel(additionalPadding)),
		s.buildLabel(label(), additionalPadding),
	})
}

func (s *timerPickerState) wheelOpts(offAxisFraction float64, overlay widgets.Widget, controller *widgets.FixedExtentScrollController, looping bool) CupertinoPickerOpts {
	w := s.Widget()
	return CupertinoPickerOpts{ScrollController: controller, Magnification: foundation.Some(datePickerMagnification), OffAxisFraction: offAxisFraction,
		BackgroundColor: w.o.BackgroundColor, Squeeze: foundation.Some(datePickerSqueeze), Looping: looping,
		ChangeReportingBehavior: w.o.ChangeReportingBehavior, SelectionOverlay: overlay, NoSelectionOverlay: overlay == nil}
}

func (s *timerPickerState) report() {
	s.Widget().onTimerDurationChanged(time.Duration(s.selectedHour.OrElse(0))*time.Hour + time.Duration(s.selectedMinute)*time.Minute +
		time.Duration(s.selectedSecond.OrElse(0))*time.Second)
}

func (s *timerPickerState) buildHourPicker(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	if s.hourController == nil {
		s.hourController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedHour.Value()})
	}
	children := make([]widgets.Widget, 24)
	for index := range children {
		label, _ := s.localizations.TimerPickerHourLabel(index)
		children[index] = s.semanticsItem(s.localizations.TimerPickerHour(index), label, additionalPadding)
	}
	return NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.SetState(func() {
			s.selectedHour = foundation.Some(index)
			s.report()
		})
	}, children, s.wheelOpts(s.calculateOffAxisFraction(additionalPadding.Start, 0), overlay, s.hourController, false))
}

func (s *timerPickerState) buildHourColumn(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	return s.column(additionalPadding, func(p painting.EdgeInsetsDirectional) widgets.Widget { return s.buildHourPicker(p, overlay) },
		func() string {
			l, _ := s.localizations.TimerPickerHourLabel(s.lastSelectedHour.OrElse(s.selectedHour.Value()))
			return l
		}, func() { s.lastSelectedHour = s.selectedHour })
}

func (s *timerPickerState) buildMinutePicker(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	if s.minuteController == nil {
		s.minuteController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedMinute / w.o.MinuteInterval})
	}
	position := 1
	if w.o.Mode == CupertinoTimerPickerModeMS {
		position = 0
	}
	children := make([]widgets.Widget, 60/w.o.MinuteInterval)
	for index := range children {
		minute := index * w.o.MinuteInterval
		label, _ := s.localizations.TimerPickerMinuteLabel(minute)
		children[index] = s.semanticsItem(s.localizations.TimerPickerMinute(minute), label, additionalPadding)
	}
	return NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.SetState(func() {
			s.selectedMinute = index * w.o.MinuteInterval
			s.report()
		})
	}, children, s.wheelOpts(s.calculateOffAxisFraction(additionalPadding.Start, position), overlay, s.minuteController, true))
}

func (s *timerPickerState) buildMinuteColumn(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	return s.column(additionalPadding, func(p painting.EdgeInsetsDirectional) widgets.Widget { return s.buildMinutePicker(p, overlay) },
		func() string {
			l, _ := s.localizations.TimerPickerMinuteLabel(s.lastSelectedMinute.OrElse(s.selectedMinute))
			return l
		}, func() { s.lastSelectedMinute = foundation.Some(s.selectedMinute) })
}

func (s *timerPickerState) buildSecondPicker(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	if s.secondController == nil {
		s.secondController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedSecond.Value() / w.o.SecondInterval})
	}
	position := 2
	if w.o.Mode == CupertinoTimerPickerModeMS {
		position = 1
	}
	children := make([]widgets.Widget, 60/w.o.SecondInterval)
	for index := range children {
		second := index * w.o.SecondInterval
		label, _ := s.localizations.TimerPickerSecondLabel(second)
		children[index] = s.semanticsItem(s.localizations.TimerPickerSecond(second), label, additionalPadding)
	}
	return NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.SetState(func() {
			s.selectedSecond = foundation.Some(index * w.o.SecondInterval)
			s.report()
		})
	}, children, s.wheelOpts(s.calculateOffAxisFraction(additionalPadding.Start, position), overlay, s.secondController, true))
}

func (s *timerPickerState) buildSecondColumn(additionalPadding painting.EdgeInsetsDirectional, overlay widgets.Widget) widgets.Widget {
	return s.column(additionalPadding, func(p painting.EdgeInsetsDirectional) widgets.Widget { return s.buildSecondPicker(p, overlay) },
		func() string {
			l, _ := s.localizations.TimerPickerSecondLabel(s.lastSelectedSecond.OrElse(s.selectedSecond.Value()))
			return l
		}, func() { s.lastSelectedSecond = s.selectedSecond })
}

// textStyleFrom is the theme's picker style with its size magnified.
func (s *timerPickerState) textStyleFrom(magnification float64) *painting.TextStyle {
	style := CupertinoThemeOf(s.Context()).TextTheme().PickerTextStyle()
	return style.CopyWith(painting.TextStyleCopy{FontSize: foundation.Some(style.FontSize.Value() * magnification)})
}

func (s *timerPickerState) calculateOffAxisFraction(paddingStart float64, position int) float64 {
	centerPoint := paddingStart + s.numberLabelWidth/2
	pickerColumnOffAxisFraction := 0.5 - centerPoint/s.pickerColumnWidth
	timerPickerOffAxisFraction := 0.5 - (centerPoint+s.pickerColumnWidth*float64(position))/s.totalWidth
	return (pickerColumnOffAxisFraction - timerPickerOffAxisFraction) * float64(s.textDirectionFactor())
}

func (s *timerPickerState) overlay(bc widgets.BuildContext, def func() widgets.Widget, index, count int) widgets.Widget {
	if b := s.Widget().o.SelectionOverlayBuilder; b != nil {
		return b(bc, count, index)
	}
	return def()
}

func (s *timerPickerState) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		w := s.Widget()
		if w.o.Mode == CupertinoTimerPickerModeHMS {
			s.pickerColumnWidth = timerPickerColumnIntrinsicWidth + timerPickerHalfColumnPadding*2
			s.totalWidth = s.pickerColumnWidth * 3
		} else {
			s.totalWidth = pickerWidth
			s.pickerColumnWidth = s.totalWidth / 2
		}
		if constraints.MaxWidth < s.totalWidth {
			s.totalWidth = constraints.MaxWidth
			if w.o.Mode == CupertinoTimerPickerModeHMS {
				s.pickerColumnWidth = s.totalWidth / 3
			} else {
				s.pickerColumnWidth = s.totalWidth / 2
			}
		}
		baseLabelContentWidth := s.numberLabelWidth + timerPickerLabelPadSize
		minuteLabelContentWidth := baseLabelContentWidth + s.minuteLabelWidth
		directional := painting.NewEdgeInsetsDirectionalFromSTEB
		var columns []widgets.Widget
		switch w.o.Mode {
		case CupertinoTimerPickerModeHM:
			hourLabelContentWidth := baseLabelContentWidth + s.hourLabelWidth
			hourColumnStartPadding := math.Max(s.pickerColumnWidth-hourLabelContentWidth-timerPickerHalfColumnPadding, timerPickerMinHorizontalPadding)
			minuteColumnEndPadding := math.Max(s.pickerColumnWidth-minuteLabelContentWidth-timerPickerHalfColumnPadding, timerPickerMinHorizontalPadding)
			columns = []widgets.Widget{
				s.buildHourColumn(directional(hourColumnStartPadding, 0, s.pickerColumnWidth-hourColumnStartPadding-hourLabelContentWidth, 0),
					s.overlay(bc, startSelectionOverlay, 0, 2)),
				s.buildMinuteColumn(directional(s.pickerColumnWidth-minuteColumnEndPadding-minuteLabelContentWidth, 0, minuteColumnEndPadding, 0),
					s.overlay(bc, endSelectionOverlay, 1, 2)),
			}
		case CupertinoTimerPickerModeMS:
			secondLabelContentWidth := baseLabelContentWidth + s.secondLabelWidth
			secondColumnEndPadding := math.Max(s.pickerColumnWidth-secondLabelContentWidth-timerPickerHalfColumnPadding, timerPickerMinHorizontalPadding)
			minuteColumnStartPadding := math.Max(s.pickerColumnWidth-minuteLabelContentWidth-timerPickerHalfColumnPadding, timerPickerMinHorizontalPadding)
			columns = []widgets.Widget{
				s.buildMinuteColumn(directional(minuteColumnStartPadding, 0, s.pickerColumnWidth-minuteColumnStartPadding-minuteLabelContentWidth, 0),
					s.overlay(bc, startSelectionOverlay, 0, 2)),
				s.buildSecondColumn(directional(s.pickerColumnWidth-secondColumnEndPadding-minuteLabelContentWidth, 0, secondColumnEndPadding, 0),
					s.overlay(bc, endSelectionOverlay, 1, 2)),
			}
		default:
			hourColumnEndPadding := s.pickerColumnWidth - baseLabelContentWidth - s.hourLabelWidth - timerPickerMinHorizontalPadding
			minuteColumnPadding := (s.pickerColumnWidth - minuteLabelContentWidth) / 2
			secondColumnStartPadding := s.pickerColumnWidth - baseLabelContentWidth - s.secondLabelWidth - timerPickerMinHorizontalPadding
			columns = []widgets.Widget{
				s.buildHourColumn(directional(timerPickerMinHorizontalPadding, 0, math.Max(hourColumnEndPadding, 0), 0), s.overlay(bc, startSelectionOverlay, 0, 3)),
				s.buildMinuteColumn(directional(minuteColumnPadding, 0, minuteColumnPadding, 0), s.overlay(bc, centerSelectionOverlay, 1, 3)),
				s.buildSecondColumn(directional(math.Max(secondColumnStartPadding, 0), 0, timerPickerMinHorizontalPadding, 0), s.overlay(bc, endSelectionOverlay, 2, 3)),
			}
		}
		expanded := make([]widgets.Widget, len(columns))
		for i, c := range columns {
			expanded[i] = widgets.NewExpanded(c)
		}
		var contents widgets.Widget = widgets.NewSizedBox(widgets.NewDefaultTextStyle(s.textStyleFrom(1), widgets.NewRow(expanded)),
			widgets.SizedBoxOpts{Width: foundation.Some(s.totalWidth), Height: foundation.Some(pickerHeight)})
		if color, ok := CupertinoDynamicColorMaybeResolve(w.o.BackgroundColor, bc).Get(); ok {
			contents = widgets.NewColoredBox(color, contents)
		}
		theme := CupertinoThemeOf(bc)
		return widgets.NewMediaQueryWithNoTextScaling(NewCupertinoTheme(theme.CopyWith(CupertinoThemeDataOpts{
			TextTheme: theme.TextTheme().CopyWith(CupertinoTextThemeDataOpts{PickerTextStyle: s.textStyleFrom(timerPickerMagnification)})}),
			widgets.NewAlign(contents, widgets.AlignOpts{Alignment: w.o.Alignment})))
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTimerPickerDurationOutOfRange    = "CupertinoTimerPickerDurationOutOfRange"
	tagCupertinoTimerPickerMinuteIntervalInvalid = "CupertinoTimerPickerMinuteIntervalInvalid"
	tagCupertinoTimerPickerSecondIntervalInvalid = "CupertinoTimerPickerSecondIntervalInvalid"
	tagCupertinoTimerPickerDurationNotOnInterval = "CupertinoTimerPickerDurationNotOnInterval"
	tagCupertinoTimerPickerItemExtentNotPositive = "CupertinoTimerPickerItemExtentNotPositive"
	tagCupertinoTimerPickerModeChanged           = "CupertinoTimerPickerModeChanged"
)
