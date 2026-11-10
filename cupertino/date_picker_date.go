package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// lastDayInMonth is the last day of the month, in the location of the date.
func lastDayInMonth(loc *time.Location, year, month int) time.Time {
	return dateIn(loc, year, time.Month(month+1), 0, 0, 0)
}

// datePickerDateState is the state of the date picker in date mode.
//
// Flutter: cupertino/date_picker.dart _CupertinoDatePickerDateState
type datePickerDateState struct {
	widgets.StateOf[*CupertinoDatePicker]
	textDirectionFactor int
	localizations       CupertinoLocalizations
	alignCenterLeft     painting.Alignment
	alignCenterRight    painting.Alignment

	selectedDay, selectedMonth, selectedYear       int
	dayController, monthController, yearController *widgets.FixedExtentScrollController
	dayScrolling, monthScrolling, yearScrolling    bool

	estimatedColumnWidths map[pickerColumnType]float64
	fontsHandle           foundation.ListenerHandle
}

func (s *datePickerDateState) isScrolling() bool {
	return s.dayScrolling || s.monthScrolling || s.yearScrolling
}

func (s *datePickerDateState) loc() *time.Location { return s.Widget().initialDateTime.Location() }

func (s *datePickerDateState) InitState() {
	w := s.Widget()
	s.selectedDay, s.selectedMonth, s.selectedYear = w.initialDateTime.Day(), int(w.initialDateTime.Month()), w.initialDateTime.Year()
	s.dayController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedDay - 1})
	s.monthController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedMonth - 1})
	s.yearController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedYear})
	s.estimatedColumnWidths = map[pickerColumnType]float64{}
	s.fontsHandle = widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().AddListener(func() {
		s.SetState(s.refreshEstimatedColumnWidths)
	})
}

func (s *datePickerDateState) Dispose() {
	s.dayController.Dispose()
	s.monthController.Dispose()
	s.yearController.Dispose()
	widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().RemoveListener(s.fontsHandle)
}

func (s *datePickerDateState) DidUpdateWidget(old *CupertinoDatePicker) {
	checkDatePickerModeUnchanged(old, s.Widget())
}

func checkDatePickerModeUnchanged(old, w *CupertinoDatePicker) {
	if foundation.DebugMode && old.o.Mode != w.o.Mode {
		panic(errors.New(tagCupertinoDatePickerModeChanged, "The mode of a CupertinoDatePicker cannot change once it's built.", "was", old.o.Mode.String(), "is", w.o.Mode.String()))
	}
}

func (s *datePickerDateState) DidChangeDependencies() {
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
	s.refreshEstimatedColumnWidths()
}

func (s *datePickerDateState) refreshEstimatedColumnWidths() {
	bc, show := s.Context(), s.Widget().o.ShowDayOfWeek
	s.estimatedColumnWidths[pickerColumnDayOfMonth] = datePickerColumnWidth(pickerColumnDayOfMonth, s.localizations, bc, show, false)
	s.estimatedColumnWidths[pickerColumnMonth] = datePickerColumnWidth(pickerColumnMonth, s.localizations, bc, show, false)
	s.estimatedColumnWidths[pickerColumnYear] = datePickerColumnWidth(pickerColumnYear, s.localizations, bc, show, false)
}

func (s *datePickerDateState) buildDayPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	bc := s.Context()
	daysInCurrentMonth := lastDayInMonth(s.loc(), s.selectedYear, s.selectedMonth).Day()
	children := make([]widgets.Widget, 31)
	for index := range children {
		day := index + 1
		dayOfWeek := foundation.None[int]()
		if w.o.ShowDayOfWeek {
			dayOfWeek = foundation.Some(weekdayNumber(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), day, 0, 0)))
		}
		invalid := day > daysInCurrentMonth
		if min, ok := w.o.MinimumDate.Get(); ok && min.Year() == s.selectedYear && int(min.Month()) == s.selectedMonth && min.Day() > day {
			invalid = true
		}
		if max, ok := w.o.MaximumDate.Get(); ok && max.Year() == s.selectedYear && int(max.Month()) == s.selectedMonth && max.Day() < day {
			invalid = true
		}
		child := positioning(bc, widgets.NewText(s.localizations.DatePickerDayOfMonth(day, dayOfWeek), widgets.TextOpts{Style: datePickerTextStyle(bc, !invalid)}))
		if invalid {
			child = widgets.NewExcludeSemantics(child)
		}
		children[index] = child
	}
	return scrollTracker(&s.dayScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.selectedDay = index + 1
		if s.isCurrentDateValid() {
			w.onDateTimeChanged(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay, 0, 0))
		}
	}, children, withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, true), s.dayController)))
}

// monthItems are the twelve months of a month column; standalone months are the names that stand alone.
func monthItems(bc widgets.BuildContext, w *CupertinoDatePicker, l CupertinoLocalizations, selectedYear int, positioning itemPositioner) []widgets.Widget {
	children := make([]widgets.Widget, 12)
	for index := range children {
		month := index + 1
		invalid := false
		if min, ok := w.o.MinimumDate.Get(); ok && min.Year() == selectedYear && int(min.Month()) > month {
			invalid = true
		}
		if max, ok := w.o.MaximumDate.Get(); ok && max.Year() == selectedYear && int(max.Month()) < month {
			invalid = true
		}
		name := l.DatePickerMonth(month)
		if w.o.Mode == CupertinoDatePickerModeMonthYear {
			name = l.DatePickerStandaloneMonth(month)
		}
		child := positioning(bc, widgets.NewText(name, widgets.TextOpts{Style: datePickerTextStyle(bc, !invalid)}))
		if invalid {
			child = widgets.NewExcludeSemantics(child)
		}
		children[index] = child
	}
	return children
}

func (s *datePickerDateState) buildMonthPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	return scrollTracker(&s.monthScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(w.o.ItemExtent, func(index int) {
		s.selectedMonth = index + 1
		if s.isCurrentDateValid() {
			w.onDateTimeChanged(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay, 0, 0))
		}
	}, monthItems(s.Context(), w, s.localizations, s.selectedYear, positioning),
		withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, true), s.monthController)))
}

// yearItem is the item of the year column of a picker, or nil outside the years of the picker.
func yearItem(bc widgets.BuildContext, w *CupertinoDatePicker, l CupertinoLocalizations, year int, positioning itemPositioner) widgets.Widget {
	if year < w.o.MinimumYear {
		return nil
	}
	if max, ok := w.o.MaximumYear.Get(); ok && year > max {
		return nil
	}
	valid := true
	if min, ok := w.o.MinimumDate.Get(); ok && min.Year() > year {
		valid = false
	}
	if max, ok := w.o.MaximumDate.Get(); ok && max.Year() < year {
		valid = false
	}
	child := positioning(bc, widgets.NewText(l.DatePickerYear(year), widgets.TextOpts{Style: datePickerTextStyle(bc, valid)}))
	if valid {
		return child
	}
	return widgets.NewExcludeSemantics(child)
}

func (s *datePickerDateState) buildYearPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	return scrollTracker(&s.yearScrolling, s.pickerDidStopScrolling, NewCupertinoPickerBuilder(w.o.ItemExtent, func(index int) {
		s.selectedYear = index
		if s.isCurrentDateValid() {
			w.onDateTimeChanged(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay, 0, 0))
		}
	}, func(bc widgets.BuildContext, year int) widgets.Widget {
		return yearItem(bc, w, s.localizations, year, positioning)
	}, withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, false), s.yearController)))
}

func (s *datePickerDateState) isCurrentDateValid() bool {
	w := s.Widget()
	minSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay, 0, 0)
	maxSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay+1, 0, 0)
	minCheck := true
	if min, ok := w.o.MinimumDate.Get(); ok {
		minCheck = min.Before(maxSelected)
	}
	maxCheck := false
	if max, ok := w.o.MaximumDate.Get(); ok {
		maxCheck = max.Before(minSelected)
	}
	return minCheck && !maxCheck && minSelected.Day() == s.selectedDay
}

func (s *datePickerDateState) pickerDidStopScrolling() {
	s.SetState(func() {})
	if s.isScrolling() {
		return
	}
	w := s.Widget()
	minSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay, 0, 0)
	maxSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), s.selectedDay+1, 0, 0)
	minCheck := true
	if min, ok := w.o.MinimumDate.Get(); ok {
		minCheck = min.Before(maxSelected)
	}
	maxCheck := false
	if max, ok := w.o.MaximumDate.Get(); ok {
		maxCheck = max.Before(minSelected)
	}
	if !minCheck || maxCheck {
		target := w.o.MinimumDate.Value()
		if minCheck {
			target = w.o.MaximumDate.Value()
		}
		s.scrollToDate(target)
		return
	}
	if minSelected.Day() != s.selectedDay {
		s.scrollToDate(lastDayInMonth(s.loc(), s.selectedYear, s.selectedMonth))
	}
}

func (s *datePickerDateState) scrollToDate(newDate time.Time) {
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if s.selectedYear != newDate.Year() {
			animateColumnControllerToItem(s.yearController, newDate.Year())
		}
		if s.selectedMonth != int(newDate.Month()) {
			animateColumnControllerToItem(s.monthController, int(newDate.Month())-1)
		}
		if s.selectedDay != newDate.Day() {
			animateColumnControllerToItem(s.dayController, newDate.Day()-1)
		}
	})
}

func (s *datePickerDateState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	order := w.o.DateOrder
	if order == DatePickerDateOrderUnspecified {
		order = s.localizations.DatePickerDateOrder()
	}
	width := func(t pickerColumnType) float64 { return s.estimatedColumnWidths[t] }
	var pickerBuilders []columnBuilder
	var columnWidths []float64
	switch order {
	case DatePickerDateOrderDMY:
		pickerBuilders = []columnBuilder{s.buildDayPicker, s.buildMonthPicker, s.buildYearPicker}
		columnWidths = []float64{width(pickerColumnDayOfMonth), width(pickerColumnMonth), width(pickerColumnYear)}
	case DatePickerDateOrderYMD:
		pickerBuilders = []columnBuilder{s.buildYearPicker, s.buildMonthPicker, s.buildDayPicker}
		columnWidths = []float64{width(pickerColumnYear), width(pickerColumnMonth), width(pickerColumnDayOfMonth)}
	case DatePickerDateOrderYDM:
		pickerBuilders = []columnBuilder{s.buildYearPicker, s.buildDayPicker, s.buildMonthPicker}
		columnWidths = []float64{width(pickerColumnYear), width(pickerColumnDayOfMonth), width(pickerColumnMonth)}
	default:
		pickerBuilders = []columnBuilder{s.buildMonthPicker, s.buildDayPicker, s.buildYearPicker}
		columnWidths = []float64{width(pickerColumnMonth), width(pickerColumnDayOfMonth), width(pickerColumnYear)}
	}
	pickers := make([]widgets.Widget, 0, len(columnWidths))
	totalColumnWidths := 4 * datePickerPadSize
	for i, colWidth := range columnWidths {
		firstColumn, lastColumn := i == 0, i == len(columnWidths)-1
		offAxisFraction := float64(i-1) * 0.3 * float64(s.textDirectionFactor)
		padding := painting.EdgeInsets{Right: datePickerPadSize}
		if s.textDirectionFactor == -1 {
			padding = painting.EdgeInsets{Left: datePickerPadSize}
		}
		overlay := columnSelectionOverlay(bc, w.o.SelectionOverlayBuilder, i, len(columnWidths))
		totalColumnWidths += colWidth + 2*datePickerPadSize
		colWidth := colWidth
		pickers = append(pickers, widgets.NewLayoutId(i, pickerBuilders[i](offAxisFraction, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
			inner := s.alignCenterRight
			if firstColumn {
				inner = s.alignCenterLeft
			}
			outer := s.alignCenterRight
			if lastColumn {
				outer = s.alignCenterLeft
			}
			itemPadding := padding
			if firstColumn {
				itemPadding = painting.EdgeInsetsZero
			}
			return widgets.NewPadding(itemPadding, widgets.NewAlign(widgets.NewSizedBox(widgets.NewAlign(child, widgets.AlignOpts{Alignment: inner}),
				widgets.SizedBoxOpts{Width: foundation.Some(colWidth + datePickerPadSize)}), widgets.AlignOpts{Alignment: outer}))
		}, overlay)))
	}
	return datePickerLayoutWidget(bc, columnWidths, s.textDirectionFactor, math.Max(totalColumnWidths, pickerWidth), pickers)
}

// datePickerMonthYearState is the state of the date picker in month and year mode.
//
// Flutter: cupertino/date_picker.dart _CupertinoDatePickerMonthYearState
type datePickerMonthYearState struct {
	widgets.StateOf[*CupertinoDatePicker]
	textDirectionFactor int
	localizations       CupertinoLocalizations
	alignCenterLeft     painting.Alignment
	alignCenterRight    painting.Alignment

	selectedYear, selectedMonth     int
	monthController, yearController *widgets.FixedExtentScrollController
	monthScrolling, yearScrolling   bool
	estimatedColumnWidths           map[pickerColumnType]float64
	fontsHandle                     foundation.ListenerHandle
}

func (s *datePickerMonthYearState) isScrolling() bool { return s.monthScrolling || s.yearScrolling }

func (s *datePickerMonthYearState) loc() *time.Location { return s.Widget().initialDateTime.Location() }

func (s *datePickerMonthYearState) InitState() {
	w := s.Widget()
	s.selectedMonth, s.selectedYear = int(w.initialDateTime.Month()), w.initialDateTime.Year()
	s.monthController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedMonth - 1})
	s.yearController = widgets.NewFixedExtentScrollController(widgets.FixedExtentScrollControllerOpts{InitialItem: s.selectedYear})
	s.estimatedColumnWidths = map[pickerColumnType]float64{}
	s.fontsHandle = widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().AddListener(func() {
		s.SetState(s.refreshEstimatedColumnWidths)
	})
}

func (s *datePickerMonthYearState) Dispose() {
	s.monthController.Dispose()
	s.yearController.Dispose()
	widgets.WidgetsBindingOf(s.Context()).Painting().SystemFonts().RemoveListener(s.fontsHandle)
}

func (s *datePickerMonthYearState) DidUpdateWidget(old *CupertinoDatePicker) {
	checkDatePickerModeUnchanged(old, s.Widget())
}

func (s *datePickerMonthYearState) DidChangeDependencies() {
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
	s.refreshEstimatedColumnWidths()
}

func (s *datePickerMonthYearState) refreshEstimatedColumnWidths() {
	bc, standalone := s.Context(), s.Widget().o.Mode == CupertinoDatePickerModeMonthYear
	s.estimatedColumnWidths[pickerColumnMonth] = datePickerColumnWidth(pickerColumnMonth, s.localizations, bc, false, standalone)
	s.estimatedColumnWidths[pickerColumnYear] = datePickerColumnWidth(pickerColumnYear, s.localizations, bc, false, false)
}

func (s *datePickerMonthYearState) buildMonthPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	return scrollTracker(&s.monthScrolling, s.pickerDidStopScrolling, NewCupertinoPicker(pickerItemExtent, func(index int) {
		s.selectedMonth = index + 1
		if s.isCurrentDateValid() {
			w.onDateTimeChanged(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), 1, 0, 0))
		}
	}, monthItems(s.Context(), w, s.localizations, s.selectedYear, positioning),
		withController(datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, true), s.monthController)))
}

func (s *datePickerMonthYearState) buildYearPicker(offAxisFraction float64, positioning itemPositioner, overlay widgets.Widget) widgets.Widget {
	w := s.Widget()
	o := datePickerWheelOpts(offAxisFraction, overlay, w.o.BackgroundColor, w.o.ChangeReportingBehavior, false)
	// Flutter's month and year picker passes no squeeze to its year wheel, so it has the default one.
	o.Squeeze = foundation.None[float64]()
	return scrollTracker(&s.yearScrolling, s.pickerDidStopScrolling, NewCupertinoPickerBuilder(pickerItemExtent, func(index int) {
		s.selectedYear = index
		if s.isCurrentDateValid() {
			w.onDateTimeChanged(dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), 1, 0, 0))
		}
	}, func(bc widgets.BuildContext, year int) widgets.Widget {
		return yearItem(bc, w, s.localizations, year, positioning)
	}, withController(o, s.yearController)))
}

func (s *datePickerMonthYearState) isCurrentDateValid() bool {
	w := s.Widget()
	minSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), 1, 0, 0)
	maxSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), w.initialDateTime.Day()+1, 0, 0)
	minCheck := true
	if min, ok := w.o.MinimumDate.Get(); ok {
		minCheck = min.Before(maxSelected)
	}
	maxCheck := false
	if max, ok := w.o.MaximumDate.Get(); ok {
		maxCheck = max.Before(minSelected)
	}
	return minCheck && !maxCheck
}

func (s *datePickerMonthYearState) pickerDidStopScrolling() {
	s.SetState(func() {})
	if s.isScrolling() {
		return
	}
	w := s.Widget()
	minSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), 1, 0, 0)
	maxSelected := dateIn(s.loc(), s.selectedYear, time.Month(s.selectedMonth), w.initialDateTime.Day()+1, 0, 0)
	minCheck := true
	if min, ok := w.o.MinimumDate.Get(); ok {
		minCheck = min.Before(maxSelected)
	}
	maxCheck := false
	if max, ok := w.o.MaximumDate.Get(); ok {
		maxCheck = max.Before(minSelected)
	}
	if !minCheck || maxCheck {
		target := w.o.MinimumDate.Value()
		if minCheck {
			target = w.o.MaximumDate.Value()
		}
		s.scrollToDate(target)
	}
}

func (s *datePickerMonthYearState) scrollToDate(newDate time.Time) {
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if s.selectedYear != newDate.Year() {
			animateColumnControllerToItem(s.yearController, newDate.Year())
		}
		if s.selectedMonth != int(newDate.Month()) {
			animateColumnControllerToItem(s.monthController, int(newDate.Month())-1)
		}
	})
}

func (s *datePickerMonthYearState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	order := w.o.DateOrder
	if order == DatePickerDateOrderUnspecified {
		order = s.localizations.DatePickerDateOrder()
	}
	var pickerBuilders []columnBuilder
	var columnWidths []float64
	switch order {
	case DatePickerDateOrderYMD, DatePickerDateOrderYDM:
		pickerBuilders = []columnBuilder{s.buildYearPicker, s.buildMonthPicker}
		columnWidths = []float64{s.estimatedColumnWidths[pickerColumnYear], s.estimatedColumnWidths[pickerColumnMonth]}
	default:
		pickerBuilders = []columnBuilder{s.buildMonthPicker, s.buildYearPicker}
		columnWidths = []float64{s.estimatedColumnWidths[pickerColumnMonth], s.estimatedColumnWidths[pickerColumnYear]}
	}
	pickers := make([]widgets.Widget, 0, len(columnWidths))
	totalColumnWidths := 3 * datePickerPadSize
	for i, colWidth := range columnWidths {
		firstColumn, lastColumn := i == 0, i == len(columnWidths)-1
		offAxisFraction := float64(s.textDirectionFactor) * 0.5
		if firstColumn {
			offAxisFraction = float64(s.textDirectionFactor) * -0.3
		}
		totalColumnWidths += colWidth + 2*datePickerPadSize
		overlay := columnSelectionOverlay(bc, w.o.SelectionOverlayBuilder, i, len(columnWidths))
		colWidth := colWidth
		pickers = append(pickers, widgets.NewLayoutId(i, pickerBuilders[i](offAxisFraction, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
			inner := s.alignCenterRight
			if firstColumn {
				inner = s.alignCenterLeft
			}
			outer := s.alignCenterRight
			if lastColumn {
				outer = s.alignCenterLeft
			}
			contents := widgets.NewAlign(widgets.NewSizedBox(widgets.NewAlign(child, widgets.AlignOpts{Alignment: inner}),
				widgets.SizedBoxOpts{Width: foundation.Some(colWidth + datePickerPadSize)}), widgets.AlignOpts{Alignment: outer})
			if firstColumn {
				return contents
			}
			padding := painting.EdgeInsets{Right: datePickerPadSize}
			if s.textDirectionFactor == -1 {
				padding = padding.Flipped()
			}
			return widgets.NewPadding(padding, contents)
		}, overlay)))
	}
	return datePickerLayoutWidget(bc, columnWidths, s.textDirectionFactor, math.Max(totalColumnWidths, pickerWidth), pickers)
}
