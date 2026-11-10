package cupertino

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

// datePickerDateTimeState is the state of the date picker in time and date and time modes.
//
// Flutter: cupertino/date_picker.dart _CupertinoDatePickerDateTimeState
type datePickerDateTimeState struct {
	widgets.StateOf[*CupertinoDatePicker]
	textDirectionFactor int
	localizations       CupertinoLocalizations
	alignCenterLeft     painting.Alignment
	alignCenterRight    painting.Alignment
	initialDateTime     time.Time

	dateController     *widgets.FixedExtentScrollController
	hourController     *widgets.FixedExtentScrollController
	minuteController   *widgets.FixedExtentScrollController
	meridiemController *widgets.FixedExtentScrollController
	selectedAmPm       int
	meridiemRegion     int

	datePickerScrolling, hourPickerScrolling, minutePickerScrolling, meridiemPickerScrolling bool

	estimatedColumnWidths map[pickerColumnType]float64
	fontsHandle           foundation.ListenerHandle
}

func (s *datePickerDateTimeState) selectedDayFromInitial() int {
	if s.Widget().o.Mode == CupertinoDatePickerModeDateAndTime && s.dateController.HasClients() {
		return s.dateController.SelectedItem()
	}
	return 0
}

func (s *datePickerDateTimeState) selectedHourIndex() int {
	if s.hourController.HasClients() {
		return s.hourController.SelectedItem() % 24
	}
	return s.initialDateTime.Hour()
}

func (s *datePickerDateTimeState) selectedHour() int {
	return s.hourFor(s.selectedAmPm, s.selectedHourIndex())
}

func (s *datePickerDateTimeState) hourFor(selectedAmPm, selectedHour int) int {
	if s.isHourRegionFlipped(selectedAmPm) {
		return (selectedHour + 12) % 24
	}
	return selectedHour
}

func (s *datePickerDateTimeState) isHourRegionFlipped(selectedAmPm int) bool {
	return selectedAmPm != s.meridiemRegion
}

func (s *datePickerDateTimeState) selectedMinute() int {
	if s.minuteController.HasClients() {
		return s.minuteController.SelectedItem() * s.Widget().o.MinuteInterval % 60
	}
	return s.initialDateTime.Minute()
}

func (s *datePickerDateTimeState) isScrolling() bool {
	return s.datePickerScrolling || s.hourPickerScrolling || s.minutePickerScrolling || s.meridiemPickerScrolling
}

func (s *datePickerDateTimeState) InitState() {
	w := s.Widget()
	s.initialDateTime = w.initialDateTime
	s.selectedAmPm = s.initialDateTime.Hour() / 12
	s.meridiemRegion = s.selectedAmPm
	s.meridiemController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedAmPm})
	s.hourController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.initialDateTime.Hour()})
	s.minuteController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.initialDateTime.Minute() / w.o.MinuteInterval})
	s.dateController = widgets.NewFixedExtentScrollController()
	s.estimatedColumnWidths = map[pickerColumnType]float64{}
	s.fontsHandle = widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().AddListener(func() {
		s.SetState(func() { clear(s.estimatedColumnWidths) })
	})
}

func (s *datePickerDateTimeState) Dispose() {
	s.dateController.Dispose()
	s.hourController.Dispose()
	s.minuteController.Dispose()
	s.meridiemController.Dispose()
	widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().RemoveListener(s.fontsHandle)
}

func (s *datePickerDateTimeState) DidUpdateWidget(old *CupertinoDatePicker) {
	w := s.Widget()
	if foundation.DebugMode && old.o.Mode != w.o.Mode {
		panic(errors.New(tagCupertinoDatePickerModeChanged, "The mode of a CupertinoDatePicker cannot change once it's built.", "was", old.o.Mode.String(), "is", w.o.Mode.String()))
	}
	if !w.o.Use24hFormat && old.o.Use24hFormat {
		s.meridiemController.Dispose()
		s.meridiemController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedAmPm})
	}
}

func (s *datePickerDateTimeState) DidChangeDependencies() {
	bc := s.Context()
	s.textDirectionFactor = 1
	if widgets.DirectionalityOf(bc) != ui.TextDirectionLTR {
		s.textDirectionFactor = -1
	}
	s.localizations = CupertinoLocalizationsOf(bc)
	s.alignCenterLeft, s.alignCenterRight = painting.AlignmentCenterLeft, painting.AlignmentCenterRight
	if s.textDirectionFactor != 1 {
		s.alignCenterLeft, s.alignCenterRight = painting.AlignmentCenterRight, painting.AlignmentCenterLeft
	}
	clear(s.estimatedColumnWidths)
}

func (s *datePickerDateTimeState) estimatedColumnWidth(columnType pickerColumnType) float64 {
	if w, ok := s.estimatedColumnWidths[columnType]; ok {
		return w
	}
	w := datePickerColumnWidth(columnType, s.localizations, s.Context(), s.Widget().o.ShowDayOfWeek, false)
	s.estimatedColumnWidths[columnType] = w
	return w
}

func (s *datePickerDateTimeState) selectedDateTime() time.Time {
	return dateIn(s.initialDateTime.Location(), s.initialDateTime.Year(), s.initialDateTime.Month(), s.initialDateTime.Day()+s.selectedDayFromInitial(),
		s.selectedHour(), s.selectedMinute())
}

func (s *datePickerDateTimeState) onSelectedItemChange(int) {
	w := s.Widget()
	selected := s.selectedDateTime()
	if min, ok := w.o.MinimumDate.Get(); ok && min.After(selected) {
		return
	}
	if max, ok := w.o.MaximumDate.Get(); ok && max.Before(selected) {
		return
	}
	if !s.isSelectableDate(selected) {
		return
	}
	w.onDateTimeChanged(selected)
}

func (s *datePickerDateTimeState) isSelectableDate(date time.Time) bool {
	p := s.Widget().o.SelectableDayPredicate
	return p == nil || p(date)
}

func (s *datePickerDateTimeState) buildMediumDatePicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	return scrollTracker(&s.datePickerScrolling, s.pickerDidStopScrolling, NewCupertinoPickerBuilder(w.o.ItemExtent, s.onSelectedItemChange,
		func(bc widgets.BuildContext, index int) widgets.Widget {
			loc := s.initialDateTime.Location()
			y, m, d := s.initialDateTime.Date()
			rangeStart := dateIn(loc, y, m, d+index, 0, 0)
			rangeEnd := dateIn(loc, y, m, d+index+1, 0, 0)
			now := time.Now().In(loc)
			if min, ok := w.o.MinimumDate.Get(); ok && !min.Before(rangeEnd) {
				return nil
			}
			if max, ok := w.o.MaximumDate.Get(); ok && !max.After(rangeStart) {
				return nil
			}
			dateText := s.localizations.DatePickerMediumDate(rangeStart)
			if rangeStart.Equal(dateIn(loc, now.Year(), now.Month(), now.Day(), 0, 0)) {
				dateText = s.localizations.TodayLabel()
			}
			disabled := !s.isSelectableDate(rangeStart)
			child := positioning(bc, widgets.NewText(dateText, widgets.TextOpts{Style: datePickerTextStyle(bc, !disabled)}))
			if disabled {
				return widgets.NewExcludeSemantics(child)
			}
			return child
		}, withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, false), s.dateController)))
}

func withController(o CupertinoPickerOpts, c *widgets.FixedExtentScrollController) CupertinoPickerOpts {
	o.ScrollController = c
	return o
}

func (s *datePickerDateTimeState) isValidHour(meridiemIndex, hourIndex int) bool {
	w := s.Widget()
	y, m, d := s.initialDateTime.Date()
	rangeStart := dateIn(s.initialDateTime.Location(), y, m, d+s.selectedDayFromInitial(), s.hourFor(meridiemIndex, hourIndex), 0)
	rangeEnd := rangeStart.Add(time.Hour)
	if min, ok := w.o.MinimumDate.Get(); ok && !min.Before(rangeEnd) {
		return false
	}
	if max, ok := w.o.MaximumDate.Get(); ok && max.Before(rangeStart) {
		return false
	}
	return true
}

// pickerText is the text of an item with the semantics label that the localization gives it.
func pickerText(text string, label func() (string, bool), style *painting.TextStyle) widgets.Widget {
	o := widgets.TextOpts{Style: style}
	if label != nil {
		if l, ok := label(); ok {
			o.SemanticsLabel = foundation.Some(l)
		}
	}
	return widgets.NewText(text, o)
}

func (s *datePickerDateTimeState) buildHourPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	bc := s.Context()
	children := make([]widgets.Widget, 24)
	for index := range children {
		hour := index
		if s.isHourRegionFlipped(s.selectedAmPm) {
			hour = (index + 12) % 24
		}
		displayHour := hour
		if !w.o.Use24hFormat {
			displayHour = (hour+11)%12 + 1
		}
		disabled := !s.isValidHour(s.selectedAmPm, index)
		child := positioning(bc, pickerText(s.localizations.DatePickerHour(displayHour),
			func() (string, bool) { return s.localizations.DatePickerHourSemanticsLabel(displayHour) }, datePickerTextStyle(bc, !disabled)))
		if disabled {
			child = widgets.NewExcludeSemantics(child)
		}
		children[index] = child
	}
	return scrollTracker(&s.hourPickerScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		regionChanged := s.meridiemRegion != index/12
		if regionChanged {
			s.meridiemRegion = index / 12
			s.selectedAmPm = 1 - s.selectedAmPm
		}
		if !w.o.Use24hFormat && regionChanged {
			s.meridiemController.AnimateToItem(s.selectedAmPm, 300*time.Millisecond, animation.Curves.EaseOut)
		} else {
			s.onSelectedItemChange(index)
		}
	}, children, withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, true), s.hourController)))
}

func (s *datePickerDateTimeState) buildMinutePicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	bc := s.Context()
	children := make([]widgets.Widget, 60/w.o.MinuteInterval)
	for index := range children {
		minute := index * w.o.MinuteInterval
		y, m, d := s.initialDateTime.Date()
		date := dateIn(s.initialDateTime.Location(), y, m, d+s.selectedDayFromInitial(), s.selectedHour(), minute)
		invalid := false
		if min, ok := w.o.MinimumDate.Get(); ok && min.After(date) {
			invalid = true
		}
		if max, ok := w.o.MaximumDate.Get(); ok && max.Before(date) {
			invalid = true
		}
		child := positioning(bc, pickerText(s.localizations.DatePickerMinute(minute),
			func() (string, bool) { return s.localizations.DatePickerMinuteSemanticsLabel(minute) }, datePickerTextStyle(bc, !invalid)))
		if invalid {
			child = widgets.NewExcludeSemantics(child)
		}
		children[index] = child
	}
	return scrollTracker(&s.minutePickerScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(w.o.ItemExtent, s.onSelectedItemChange, children,
		withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, true), s.minuteController)))
}

func (s *datePickerDateTimeState) buildAmPmPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	bc := s.Context()
	children := make([]widgets.Widget, 2)
	for index := range children {
		disabled := !s.isValidHour(index, s.selectedHourIndex())
		text := s.localizations.AnteMeridiemAbbreviation()
		if index != 0 {
			text = s.localizations.PostMeridiemAbbreviation()
		}
		child := positioning(bc, widgets.NewText(text, widgets.TextOpts{Style: datePickerTextStyle(bc, !disabled)}))
		if disabled {
			child = widgets.NewExcludeSemantics(child)
		}
		children[index] = child
	}
	return scrollTracker(&s.meridiemPickerScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.selectedAmPm = index
		s.onSelectedItemChange(index)
	}, children, withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, false), s.meridiemController)))
}

func (s *datePickerDateTimeState) buildTimeSeparator(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	bc := s.Context()
	o := datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, widgets.ChangeReportingBehaviorUnspecified, false)
	return widgets.NewExcludeSemantics(NewCupertinoPicker(w.o.ItemExtent, func(int) {},
		[]widgets.Widget{positioning(bc, widgets.NewText(":", widgets.TextOpts{Style: datePickerTextStyle(bc, true)}))}, o))
}

func (s *datePickerDateTimeState) scrollToFirstSelectableDate() {
	selected := s.selectedDateTime()
	if !s.isSelectableDate(selected) {
		const daysThreshold = 1
		s.scrollToDate(selected.AddDate(0, 0, daysThreshold), selected, false, foundation.Some(s.dateController.SelectedItem()+daysThreshold))
	}
}

func (s *datePickerDateTimeState) pickerDidStopScrolling() {
	s.SetState(func() {})
	if s.isScrolling() {
		return
	}
	w := s.Widget()
	selected := s.selectedDateTime()
	minCheck, maxCheck := false, false
	if min, ok := w.o.MinimumDate.Get(); ok {
		minCheck = min.After(selected)
	}
	if max, ok := w.o.MaximumDate.Get(); ok {
		maxCheck = max.Before(selected)
	}
	s.scrollToFirstSelectableDate()
	if minCheck || maxCheck {
		target := w.o.MaximumDate.Value()
		if minCheck {
			target = w.o.MinimumDate.Value()
		}
		s.scrollToDate(target, selected, minCheck, foundation.None[int]())
	}
}

func (s *datePickerDateTimeState) scrollToDate(newDate, fromDate time.Time, minCheck bool, focusedIndex foundation.Opt[int]) {
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		w := s.Widget()
		if fromDate.Year() != newDate.Year() || fromDate.Month() != newDate.Month() || fromDate.Day() != newDate.Day() {
			animateColumnControllerToItem(s.dateController, focusedIndex.OrElse(s.selectedDayFromInitial()))
		}
		if fromDate.Hour() != newDate.Hour() {
			needsMeridiemChange := !w.o.Use24hFormat && fromDate.Hour()/12 != newDate.Hour()/12
			if needsMeridiemChange {
				animateColumnControllerToItem(s.meridiemController, 1-s.meridiemController.SelectedItem())
				item := s.hourController.SelectedItem()
				animateColumnControllerToItem(s.hourController, (item/12)*12+floorMod(item+newDate.Hour()-fromDate.Hour(), 12))
			} else {
				animateColumnControllerToItem(s.hourController, s.hourController.SelectedItem()+newDate.Hour()-fromDate.Hour())
			}
		}
		if fromDate.Minute() != newDate.Minute() {
			position := float64(newDate.Minute()) / float64(w.o.MinuteInterval)
			if minCheck {
				animateColumnControllerToItem(s.minuteController, int(math.Ceil(position)))
			} else {
				animateColumnControllerToItem(s.minuteController, int(math.Floor(position)))
			}
		}
	})
}

// floorMod is Dart's remainder operator %, whose result is never negative for a positive divisor.
func floorMod(a, b int) int { return ((a % b) + b) % b }

func (s *datePickerDateTimeState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	columnWidths := []float64{s.estimatedColumnWidth(pickerColumnHour), s.estimatedColumnWidth(pickerColumnMinute)}
	pickerBuilders := []columnBuilder{s.buildHourPicker, s.buildMinutePicker}
	if widgets.DirectionalityOf(bc) == ui.TextDirectionRTL {
		pickerBuilders = []columnBuilder{s.buildMinutePicker, s.buildHourPicker}
	}
	insert := func(i int, width float64, b columnBuilder) {
		columnWidths = append(columnWidths[:i], append([]float64{width}, columnWidths[i:]...)...)
		pickerBuilders = append(pickerBuilders[:i], append([]columnBuilder{b}, pickerBuilders[i:]...)...)
	}
	order := s.localizations.DatePickerDateTimeOrder()
	if order == DatePickerDateTimeOrderUnspecified {
		order = DatePickerDateTimeOrderDateTimeDayPeriod
	}
	if w.o.ShowTimeSeparator {
		insert(1, s.estimatedColumnWidth(pickerColumnTimeSeparator), s.buildTimeSeparator)
	}
	if !w.o.Use24hFormat {
		switch order {
		case DatePickerDateTimeOrderDateTimeDayPeriod, DatePickerDateTimeOrderTimeDayPeriodDate:
			insert(len(columnWidths), s.estimatedColumnWidth(pickerColumnDayPeriod), s.buildAmPmPicker)
		case DatePickerDateTimeOrderDateDayPeriodTime, DatePickerDateTimeOrderDayPeriodTimeDate:
			insert(0, s.estimatedColumnWidth(pickerColumnDayPeriod), s.buildAmPmPicker)
		}
	}
	if w.o.Mode == CupertinoDatePickerModeDateAndTime {
		switch order {
		case DatePickerDateTimeOrderTimeDayPeriodDate, DatePickerDateTimeOrderDayPeriodTimeDate:
			insert(len(columnWidths), s.estimatedColumnWidth(pickerColumnDate), s.buildMediumDatePicker)
		case DatePickerDateTimeOrderDateTimeDayPeriod, DatePickerDateTimeOrderDateDayPeriodTime:
			insert(0, s.estimatedColumnWidth(pickerColumnDate), s.buildMediumDatePicker)
		}
	}
	pickers := make([]widgets.Widget, 0, len(columnWidths))
	totalColumnWidths := 4 * datePickerPadSize
	for i, width := range columnWidths {
		firstColumn, lastColumn := i == 0, i == len(columnWidths)-1
		offAxisFraction := 0.0
		overlay := columnSelectionOverlay(bc, w.o.SelectionOverlayBuilder, i, len(columnWidths))
		if firstColumn {
			offAxisFraction = -datePickerMaximumOffAxisFraction * float64(s.textDirectionFactor)
		} else if i >= 2 || len(columnWidths) == 2 {
			offAxisFraction = datePickerMaximumOffAxisFraction * float64(s.textDirectionFactor)
		}
		padding := columnPadding(s.textDirectionFactor, lastColumn)
		totalColumnWidths += width + 2*datePickerPadSize
		width := width
		pickers = append(pickers, widgets.NewLayoutId(i, pickerBuilders[i](offAxisFraction, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
			var contents widgets.Widget = child
			if firstColumn || lastColumn {
				contents = widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: width + datePickerPadSize, MaxHeight: math.Inf(1)}, child)
			}
			alignment := s.alignCenterRight
			if lastColumn {
				alignment = s.alignCenterLeft
			}
			return widgets.NewPadding(padding, widgets.NewAlign(contents, widgets.AlignOpts{Alignment: alignment}))
		}, overlay)))
	}
	maxPickerWidth := math.Max(totalColumnWidths, pickerWidth)
	return datePickerLayoutWidget(bc, columnWidths, s.textDirectionFactor, maxPickerWidth, pickers)
}

// datePickerLayoutWidget is the columns in a layout of their widths, with the text of the picker unscaled.
func datePickerLayoutWidget(bc widgets.BuildContext, columnWidths []float64, textDirectionFactor int, maxWidth float64, pickers []widgets.Widget) widgets.Widget {
	delegate := &datePickerLayout{columnWidths: columnWidths, textDirectionFactor: textDirectionFactor, maxWidth: maxWidth,
		reportError: widgets.WidgetsBindingOf(bc).ErrorReporter()}
	return widgets.NewMediaQueryWithNoTextScaling(widgets.NewDefaultTextStyleMerge(widgets.NewCustomMultiChildLayout(delegate, pickers),
		widgets.DefaultTextStyleMergeOpts{Style: datePickerDefaultTextStyle()}))
}
