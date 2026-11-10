package cupertino

import (
	"math"
	"strconv"
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
	pickerItemExtent                  = 32.0
	pickerWidth                       = 320.0
	pickerHeight                      = 216.0
	datePickerMagnification           = 2.35 / 2.1
	datePickerPadSize                 = 12.0
	datePickerSqueeze                 = 1.25
	datePickerMaximumOffAxisFraction  = 0.45
	timerPickerMagnification          = 34.0 / 32.0
	timerPickerMinHorizontalPadding   = 30.0
	timerPickerHalfColumnPadding      = 4.0
	timerPickerLabelPadSize           = 6.0
	timerPickerLabelFontSize          = 17.0
	timerPickerColumnIntrinsicWidth   = 106.0
	datePickerScrollAnimationDuration = 200 * time.Millisecond
)

// datePickerTextStyle is the style of the items of a date picker: the theme's date and time style, in the
// gray of the disabled items when the item is not valid.
//
// Flutter: cupertino/date_picker.dart _themeTextStyle
func datePickerTextStyle(bc widgets.BuildContext, valid bool) *painting.TextStyle {
	style := CupertinoThemeOf(bc).TextTheme().DateTimePickerTextStyle()
	if valid {
		// Deviation: the colors of the theme's styles are resolved by CupertinoThemeOf, so there is nothing for
		// CupertinoDynamicColor.maybeResolve to resolve here.
		return style
	}
	return style.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.InactiveGray, bc))})
}

// datePickerDefaultTextStyle is the style that the columns of a date picker add to the text style around them.
//
// Flutter: cupertino/date_picker.dart _kDefaultPickerTextStyle
func datePickerDefaultTextStyle() *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{LetterSpacing: foundation.Some(-0.83)})
}

func animateColumnControllerToItem(controller *widgets.FixedExtentScrollController, targetItem int) {
	controller.AnimateToItem(targetItem, datePickerScrollAnimationDuration, animation.Curves.EaseInOut)
}

// SelectionOverlayBuilder builds the selection overlay of a column of a date or timer picker, given the number of
// columns and the index of the column; nil means no overlay.
//
// Flutter: cupertino/date_picker.dart SelectionOverlayBuilder
type SelectionOverlayBuilder func(bc widgets.BuildContext, columnCount, selectedIndex int) widgets.Widget

func startSelectionOverlay() widgets.Widget {
	return NewCupertinoPickerDefaultSelectionOverlay(CupertinoPickerDefaultSelectionOverlayOpts{NotCapEndEdge: true})
}

func centerSelectionOverlay() widgets.Widget {
	return NewCupertinoPickerDefaultSelectionOverlay(CupertinoPickerDefaultSelectionOverlayOpts{NotCapStartEdge: true, NotCapEndEdge: true})
}

func endSelectionOverlay() widgets.Widget {
	return NewCupertinoPickerDefaultSelectionOverlay(CupertinoPickerDefaultSelectionOverlayOpts{NotCapStartEdge: true})
}

// columnSelectionOverlay is the overlay of column i of count: the one that the widget's builder makes, or the
// default one that caps the first and the last column.
func columnSelectionOverlay(bc widgets.BuildContext, builder SelectionOverlayBuilder, i, count int) widgets.Widget {
	if builder != nil {
		return builder(bc, count, i)
	}
	switch {
	case i == 0:
		return startSelectionOverlay()
	case i == count-1:
		return endSelectionOverlay()
	}
	return centerSelectionOverlay()
}

// pickerOpts are the options that every wheel of a date picker has.
func datePickerWheelOpts(offAxisFraction float64, overlay widgets.Widget, background ui.ColorValue, behavior widgets.ChangeReportingBehavior, looping bool) CupertinoPickerOpts {
	return CupertinoPickerOpts{
		OffAxisFraction: offAxisFraction, UseMagnifier: true, Magnification: foundation.Some(datePickerMagnification),
		BackgroundColor: background, Squeeze: foundation.Some(datePickerSqueeze), ChangeReportingBehavior: behavior,
		SelectionOverlay: overlay, NoSelectionOverlay: overlay == nil, Looping: looping,
	}
}

// datePickerLayout lays the columns of a date picker out: each one is as wide as its widest text plus the padding,
// and the first and the last one share the width that is left.
//
// Flutter: cupertino/date_picker.dart _DatePickerLayoutDelegate
type datePickerLayout struct {
	rendering.MultiChildLayoutDelegateBase
	columnWidths        []float64
	textDirectionFactor int
	maxWidth            float64
	reportError         foundation.ErrorReporter
}

func (d *datePickerLayout) PerformLayout(size ui.Size) {
	remainingWidth := math.Min(d.maxWidth, size.Width)
	currentHorizontalOffset := (size.Width - remainingWidth) / 2
	for _, w := range d.columnWidths {
		remainingWidth -= w + datePickerPadSize*2
	}
	for i := range d.columnWidths {
		index := i
		if d.textDirectionFactor != 1 {
			index = len(d.columnWidths) - i - 1
		}
		childWidth := d.columnWidths[index] + datePickerPadSize*2
		if index == 0 || index == len(d.columnWidths)-1 {
			childWidth += remainingWidth / 2
		}
		if foundation.DebugMode && childWidth < 0 && d.reportError != nil {
			d.reportError(errors.New(tagCupertinoDatePickerTooNarrow, "Insufficient horizontal space to render the CupertinoDatePicker because the parent is too narrow at "+
				strconv.FormatFloat(size.Width, 'f', -1, 64)+"px.\nAn additional "+strconv.FormatFloat(-remainingWidth, 'f', -1, 64)+"px is needed to avoid overlapping columns.",
				"width", size.Width, "missing", -remainingWidth))
		}
		d.LayoutChild(index, rendering.NewBoxConstraintsTight(ui.Size{Width: math.Max(0, childWidth), Height: size.Height}))
		d.PositionChild(index, ui.Offset{Dx: currentHorizontalOffset})
		currentHorizontalOffset += childWidth
	}
}

func (d *datePickerLayout) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o := old.(*datePickerLayout)
	if d.textDirectionFactor != o.textDirectionFactor || len(d.columnWidths) != len(o.columnWidths) {
		return true
	}
	// Deviation: Dart compares the lists by identity, and every build makes a new one, so it always lays out again; the
	// widths are compared here, which gives the same layouts.
	for i, w := range d.columnWidths {
		if w != o.columnWidths[i] {
			return true
		}
	}
	return false
}

// CupertinoDatePickerMode is the mode of a CupertinoDatePicker: the columns that it shows.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode
type CupertinoDatePickerMode int

const (
	// CupertinoDatePickerModeUnspecified is the default mode, DateAndTime.
	//
	// Flutter: no counterpart
	CupertinoDatePickerModeUnspecified CupertinoDatePickerMode = iota
	// CupertinoDatePickerModeTime shows the hour and the minute, and the AM or PM unless the format has 24 hours.
	//
	// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode.time
	CupertinoDatePickerModeTime
	// CupertinoDatePickerModeDate shows the day, the month and the year.
	//
	// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode.date
	CupertinoDatePickerModeDate
	// CupertinoDatePickerModeDateAndTime shows the day, the hour and the minute.
	//
	// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode.dateAndTime
	CupertinoDatePickerModeDateAndTime
	// CupertinoDatePickerModeMonthYear shows the month and the year.
	//
	// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode.monthYear
	CupertinoDatePickerModeMonthYear
)

// String is the name of the mode, like Dart's enum toString.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePickerMode
func (m CupertinoDatePickerMode) String() string {
	switch m {
	case CupertinoDatePickerModeTime:
		return "CupertinoDatePickerMode.time"
	case CupertinoDatePickerModeDate:
		return "CupertinoDatePickerMode.date"
	case CupertinoDatePickerModeMonthYear:
		return "CupertinoDatePickerMode.monthYear"
	}
	return "CupertinoDatePickerMode.dateAndTime"
}

type pickerColumnType int

const (
	pickerColumnDayOfMonth pickerColumnType = iota
	pickerColumnMonth
	pickerColumnYear
	pickerColumnDate
	pickerColumnHour
	pickerColumnMinute
	pickerColumnDayPeriod
	pickerColumnTimeSeparator
)

// CupertinoDatePickerOpts are the optional parameters of NewCupertinoDatePicker.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.new
type CupertinoDatePickerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Mode is the columns that the picker shows; the default is DateAndTime.
	Mode CupertinoDatePickerMode
	// InitialDateTime is the date that the picker starts at; the default is now. The picker does not follow later
	// changes of it.
	InitialDateTime foundation.Opt[time.Time]
	// MinimumDate is the earliest date that can be selected.
	MinimumDate foundation.Opt[time.Time]
	// MaximumDate is the latest date that can be selected.
	MaximumDate foundation.Opt[time.Time]
	// MinimumYear is the first year of the year column in date and month and year mode; the default is 1.
	MinimumYear int
	// MaximumYear is the last year of the year column; there is none by default.
	MaximumYear foundation.Opt[int]
	// MinuteInterval is the difference between two minutes of the minute column; it must be a factor of 60. The default is 1.
	MinuteInterval int
	// Use24hFormat shows the hours from 0 to 23, with no AM or PM column.
	Use24hFormat bool
	// DateOrder is the order of the day, month and year columns; the default is the localization's.
	DateOrder DatePickerDateOrder
	// BackgroundColor is the background of the picker.
	BackgroundColor ui.ColorValue
	// ShowDayOfWeek shows the weekday in the day column (date mode only).
	ShowDayOfWeek bool
	// ShowTimeSeparator shows a colon between the hour and the minute columns (time and date and time modes only).
	ShowTimeSeparator bool
	// ItemExtent is the height of an item; the default is 32.
	ItemExtent float64
	// SelectionOverlayBuilder builds the selection overlay of each column; it may return nil.
	SelectionOverlayBuilder SelectionOverlayBuilder
	// SelectableDayPredicate says whether a day can be selected; the days that it rejects are grayed out and skipped.
	SelectableDayPredicate widgets.SelectableDayPredicate
	// ChangeReportingBehavior is when OnDateTimeChanged is called; the default is on every scroll update.
	ChangeReportingBehavior widgets.ChangeReportingBehavior
}

// CupertinoDatePicker is an iOS style date and time picker: wheels of days, months, years, hours and minutes.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker
type CupertinoDatePicker struct {
	widgets.StatefulWidget
	onDateTimeChanged func(time.Time)
	initialDateTime   time.Time
	o                 CupertinoDatePickerOpts
}

// NewCupertinoDatePicker creates a date picker that calls onDateTimeChanged with each date that the user selects. In
// debug builds it panics with the tag CupertinoDatePickerItemExtentNotPositive, CupertinoDatePickerMinuteIntervalInvalid,
// CupertinoDatePickerInitialBeforeMinimum, CupertinoDatePickerInitialAfterMaximum, CupertinoDatePickerMinimumYearInvalid,
// CupertinoDatePickerInitialYearAfterMaximumYear, CupertinoDatePickerDayOfWeekOutsideDateMode,
// CupertinoDatePickerMinuteNotDivisible, CupertinoDatePickerTimeSeparatorMode or CupertinoDatePickerInitialNotSelectable
// when an argument breaks the rule of that name.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.new
func NewCupertinoDatePicker(onDateTimeChanged func(time.Time), opts ...CupertinoDatePickerOpts) *CupertinoDatePicker {
	o := firstOf(opts)
	if o.Mode == CupertinoDatePickerModeUnspecified {
		o.Mode = CupertinoDatePickerModeDateAndTime
	}
	if o.MinimumYear == 0 {
		o.MinimumYear = 1
	}
	if o.MinuteInterval == 0 {
		o.MinuteInterval = 1
	}
	if o.ItemExtent == 0 {
		o.ItemExtent = pickerItemExtent
	}
	initial, given := o.InitialDateTime.Get()
	if !given {
		initial = time.Now()
	}
	checkDatePickerArguments(o, initial, given)
	return &CupertinoDatePicker{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onDateTimeChanged: onDateTimeChanged, initialDateTime: initial, o: o}
}

func checkDatePickerArguments(o CupertinoDatePickerOpts, initial time.Time, given bool) {
	if !foundation.DebugMode {
		return
	}
	yearMode := o.Mode == CupertinoDatePickerModeDate || o.Mode == CupertinoDatePickerModeMonthYear
	min, hasMin := o.MinimumDate.Get()
	max, hasMax := o.MaximumDate.Get()
	fail := func(tag, message string, kv ...any) { panic(errors.New(tag, message, kv...)) }
	switch {
	case o.ItemExtent <= 0:
		fail(tagCupertinoDatePickerItemExtentNotPositive, "item extent should be greater than 0", "itemExtent", o.ItemExtent)
	case o.MinuteInterval <= 0 || 60%o.MinuteInterval != 0:
		fail(tagCupertinoDatePickerMinuteIntervalInvalid, "minute interval is not a positive integer factor of 60", "minuteInterval", o.MinuteInterval)
	case o.Mode == CupertinoDatePickerModeDateAndTime && hasMin && initial.Before(min):
		fail(tagCupertinoDatePickerInitialBeforeMinimum, "initial date is before minimum date", "initialDateTime", initial.String(), "minimumDate", min.String())
	case o.Mode == CupertinoDatePickerModeDateAndTime && hasMax && initial.After(max):
		fail(tagCupertinoDatePickerInitialAfterMaximum, "initial date is after maximum date", "initialDateTime", initial.String(), "maximumDate", max.String())
	case yearMode && (o.MinimumYear < 1 || initial.Year() < o.MinimumYear):
		fail(tagCupertinoDatePickerMinimumYearInvalid, "initial year is not greater than minimum year, or minimum year is not positive", "minimumYear", o.MinimumYear)
	case yearMode && o.MaximumYear.IsSome() && initial.Year() > o.MaximumYear.Value():
		fail(tagCupertinoDatePickerInitialYearAfterMaximumYear, "initial year is not smaller than maximum year", "maximumYear", o.MaximumYear.Value())
	case yearMode && hasMin && min.After(initial):
		fail(tagCupertinoDatePickerInitialBeforeMinimum, "initial date "+initial.String()+" is not greater than or equal to minimumDate "+min.String())
	case yearMode && hasMax && max.Before(initial):
		fail(tagCupertinoDatePickerInitialAfterMaximum, "initial date "+initial.String()+" is not less than or equal to maximumDate "+max.String())
	case o.Mode != CupertinoDatePickerModeDate && o.ShowDayOfWeek:
		fail(tagCupertinoDatePickerDayOfWeekOutsideDateMode, "showDayOfWeek is only supported in date mode")
	case initial.Minute()%o.MinuteInterval != 0:
		fail(tagCupertinoDatePickerMinuteNotDivisible, "initial minute is not divisible by minute interval", "minute", initial.Minute(), "minuteInterval", o.MinuteInterval)
	case o.ShowTimeSeparator && o.Mode != CupertinoDatePickerModeDateAndTime && o.Mode != CupertinoDatePickerModeTime:
		fail(tagCupertinoDatePickerTimeSeparatorMode, "showTimeSeparator is only supported in time or dateAndTime modes")
	case o.SelectableDayPredicate != nil && given && !o.SelectableDayPredicate(initial):
		fail(tagCupertinoDatePickerInitialNotSelectable, initial.String()+" must satisfy provided selectableDayPredicate.")
	}
}

// Mode is the columns that the picker shows.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.mode
func (w *CupertinoDatePicker) Mode() CupertinoDatePickerMode { return w.o.Mode }

// InitialDateTime is the date that the picker started at.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.initialDateTime
func (w *CupertinoDatePicker) InitialDateTime() time.Time { return w.initialDateTime }

// MinuteInterval is the difference between two minutes of the minute column.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.minuteInterval
func (w *CupertinoDatePicker) MinuteInterval() int { return w.o.MinuteInterval }

// ItemExtent is the height of an item.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.itemExtent
func (w *CupertinoDatePicker) ItemExtent() float64 { return w.o.ItemExtent }

// CreateState creates the state of the mode.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.createState
func (w *CupertinoDatePicker) CreateState() widgets.State {
	switch w.o.Mode {
	case CupertinoDatePickerModeDate:
		return &datePickerDateState{}
	case CupertinoDatePickerModeMonthYear:
		return &datePickerMonthYearState{}
	}
	return &datePickerDateTimeState{}
}

// datePickerColumnWidth is the width of the widest text that a column shows.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker._getColumnWidth
func datePickerColumnWidth(columnType pickerColumnType, l CupertinoLocalizations, bc widgets.BuildContext, showDayOfWeek, standaloneMonth bool) float64 {
	var texts []string
	switch columnType {
	case pickerColumnDate:
		for i := 1; i <= 12; i++ {
			texts = append(texts, l.DatePickerMediumDate(time.Date(2018, time.Month(i), 25, 0, 0, 0, 0, time.Local)))
		}
	case pickerColumnHour:
		for i := range 24 {
			texts = append(texts, l.DatePickerHour(i))
		}
	case pickerColumnMinute:
		for i := range 60 {
			texts = append(texts, l.DatePickerMinute(i))
		}
	case pickerColumnDayPeriod:
		texts = append(texts, l.AnteMeridiemAbbreviation(), l.PostMeridiemAbbreviation())
	case pickerColumnDayOfMonth:
		for i := 1; i <= 31; i++ {
			texts = append(texts, l.DatePickerDayOfMonth(i, foundation.None[int]()))
		}
		if showDayOfWeek {
			for wd := 1; wd < 7; wd++ {
				texts = append(texts, l.DatePickerDayOfMonth(31, foundation.Some(wd)))
			}
		}
	case pickerColumnMonth:
		for i := 1; i <= 12; i++ {
			if standaloneMonth {
				texts = append(texts, l.DatePickerStandaloneMonth(i))
			} else {
				texts = append(texts, l.DatePickerMonth(i))
			}
		}
	case pickerColumnYear:
		texts = append(texts, l.DatePickerYear(2018))
	case pickerColumnTimeSeparator:
		texts = append(texts, ":")
	}
	if foundation.DebugMode {
		if len(texts) == 0 {
			panic(errors.New(tagCupertinoDatePickerColumnType, "column type is not appropriate", "columnType", int(columnType)))
		}
		for _, t := range texts {
			if t == "" {
				panic(errors.New(tagCupertinoDatePickerColumnType, "column type is not appropriate", "columnType", int(columnType)))
			}
		}
	}
	return CupertinoDatePickerGetColumnWidth(texts, bc, nil)
}

// CupertinoDatePickerGetColumnWidth is the width of the widest of the texts in the style (the theme's date and
// time style when it is nil): what the picker gives a column.
//
// Flutter: cupertino/date_picker.dart CupertinoDatePicker.getColumnWidth
func CupertinoDatePickerGetColumnWidth(texts []string, bc widgets.BuildContext, textStyle *painting.TextStyle) float64 {
	if textStyle == nil {
		textStyle = datePickerTextStyle(bc, true)
	}
	engine := widgets.WidgetsBindingOf(bc).Painting().TextEngine()
	direction := widgets.DirectionalityOf(bc)
	widest := math.Inf(-1)
	for _, text := range texts {
		widest = math.Max(widest, painting.TextPainterComputeMaxIntrinsicWidth(
			painting.NewTextSpan(painting.TextSpanOpts{Style: textStyle, Text: foundation.Some(text)}), direction,
			painting.TextPainterComputeWidthOpts{TextEngine: engine}))
	}
	return widest
}

// itemPositioner places the text of an item in its column.
type itemPositioner func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget

// columnBuilder builds one column of a picker.
type columnBuilder func(offAxisFraction float64, positioning itemPositioner, selectionOverlay widgets.Widget) widgets.Widget

// scrollTracker keeps whether a column is being scrolled, from the notifications of its scroll view.
func scrollTracker(scrolling *bool, didStop func(), child widgets.Widget) widgets.Widget {
	return widgets.NewNotificationListener(func(n widgets.ScrollNotification) bool {
		switch n.(type) {
		case *widgets.ScrollStartNotification:
			*scrolling = true
		case *widgets.ScrollEndNotification:
			*scrolling = false
			didStop()
		}
		return false
	}, child)
}

// dateIn is the date and time at the day, which may be outside the month, in the location of the date.
func dateIn(loc *time.Location, year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

// weekdayNumber is the weekday of Dart's DateTime: 1 for Monday to 7 for Sunday.
func weekdayNumber(t time.Time) int {
	if wd := int(t.Weekday()); wd != 0 {
		return wd
	}
	return 7
}

func columnPadding(textDirectionFactor int, lastColumn bool) painting.EdgeInsets {
	padding := painting.EdgeInsets{Right: datePickerPadSize}
	if lastColumn {
		padding = padding.Flipped()
	}
	if textDirectionFactor == -1 {
		padding = padding.Flipped()
	}
	return padding
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoDatePickerItemExtentNotPositive       = "CupertinoDatePickerItemExtentNotPositive"
	tagCupertinoDatePickerMinuteIntervalInvalid       = "CupertinoDatePickerMinuteIntervalInvalid"
	tagCupertinoDatePickerInitialBeforeMinimum        = "CupertinoDatePickerInitialBeforeMinimum"
	tagCupertinoDatePickerInitialAfterMaximum         = "CupertinoDatePickerInitialAfterMaximum"
	tagCupertinoDatePickerMinimumYearInvalid          = "CupertinoDatePickerMinimumYearInvalid"
	tagCupertinoDatePickerInitialYearAfterMaximumYear = "CupertinoDatePickerInitialYearAfterMaximumYear"
	tagCupertinoDatePickerDayOfWeekOutsideDateMode    = "CupertinoDatePickerDayOfWeekOutsideDateMode"
	tagCupertinoDatePickerMinuteNotDivisible          = "CupertinoDatePickerMinuteNotDivisible"
	tagCupertinoDatePickerTimeSeparatorMode           = "CupertinoDatePickerTimeSeparatorMode"
	tagCupertinoDatePickerInitialNotSelectable        = "CupertinoDatePickerInitialNotSelectable"
	tagCupertinoDatePickerColumnType                  = "CupertinoDatePickerColumnType"
	tagCupertinoDatePickerTooNarrow                   = "CupertinoDatePickerTooNarrow"
	tagCupertinoDatePickerModeChanged                 = "CupertinoDatePickerModeChanged"
)
