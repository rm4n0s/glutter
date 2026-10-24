package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

func buildDialogsPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewWrap(children)
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "Dialogs", NewDialogDemos()), section(bc, "Date and time pickers", NewPickerDemos())),
		top(section(bc, "CalendarDatePicker", NewCalendarDemo()), section(bc, "YearPicker", NewYearPickerDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// devOpenDelay is how long GLUTTER_DEV_OPEN waits before it opens a dialog: the
// window gets its size after the first frame.
const devOpenDelay = 400 * time.Millisecond

// devOpen is the dialog that GLUTTER_DEV_OPEN asks to open, in a debug build:
// alert, simple, fullscreen, datepicker, daterange, datepickerinput,
// timepicker or timepickerinput.
func devOpen(getenv func(string) string) string {
	if !foundation.DebugMode {
		return ""
	}
	return getenv("GLUTTER_DEV_OPEN")
}

// awaitResult calls then on the UI thread with the value that arrives on the
// channel, if the state is still mounted.
func awaitResult[T any](bc m.BuildContext, mounted func() bool, ch <-chan T, then func(T)) {
	binding := m.WidgetsBindingOf(bc)
	go func() {
		v, ok := <-ch
		if !ok {
			return
		}
		binding.Post(func() {
			if mounted() {
				then(v)
			}
		})
	}()
}

// DialogDemos has a button for each kind of dialog, and the result of the last one.
type DialogDemos struct{ m.StatefulWidget }

// NewDialogDemos creates the demo.
func NewDialogDemos() *DialogDemos { return &DialogDemos{} }

// CreateState creates the state.
func (w *DialogDemos) CreateState() m.State { return &dialogDemosState{} }

type dialogDemosState struct {
	m.StateOf[*DialogDemos]
	result string
}

// InitState opens the dialog that GLUTTER_DEV_OPEN asks for.
func (s *dialogDemosState) InitState() {
	s.result = "none yet"
	var open func()
	switch devOpen(os.Getenv) {
	case "alert":
		open = s.showAlert
	case "simple":
		open = s.showSimple
	case "fullscreen":
		open = s.showFullscreen
	default:
		return
	}
	m.WidgetsBindingOf(s.Context()).Gestures().NewTimer(devOpenDelay, func() {
		if s.Mounted() {
			open()
		}
	})
}

func (s *dialogDemosState) setResult(text string) { s.SetState(func() { s.result = text }) }

func (s *dialogDemosState) showAlert() {
	ch := m.ShowDialog[string](s.Context(), func(bc m.BuildContext) m.Widget {
		return m.NewAlertDialog(m.AlertDialogOpts{
			Icon:    m.NewIcon(m.Icons.DeleteOutline()),
			Title:   m.NewText("Discard draft?"),
			Content: m.NewText("The draft will be deleted. This cannot be undone."),
			Actions: []m.Widget{
				m.NewTextButton(func() { m.NavigatorPop(bc, "Cancel") }, m.NewText("Cancel")),
				m.NewTextButton(func() { m.NavigatorPop(bc, "Discard") }, m.NewText("Discard")),
			},
		})
	})
	awaitResult(s.Context(), s.Mounted, ch, func(v string) { s.setResult("AlertDialog: " + orNone(v)) })
}

func (s *dialogDemosState) showSimple() {
	option := func(bc m.BuildContext, label string) m.Widget {
		return m.NewSimpleDialogOption(func() { m.NavigatorPop(bc, label) }, m.NewText(label))
	}
	ch := m.ShowDialog[string](s.Context(), func(bc m.BuildContext) m.Widget {
		return m.NewSimpleDialog([]m.Widget{option(bc, "Ada"), option(bc, "Grace"), option(bc, "Margaret")},
			m.SimpleDialogOpts{Title: m.NewText("Choose an account")})
	})
	awaitResult(s.Context(), s.Mounted, ch, func(v string) { s.setResult("SimpleDialog: " + orNone(v)) })
}

func (s *dialogDemosState) showFullscreen() {
	ch := m.ShowDialog[string](s.Context(), func(bc m.BuildContext) m.Widget {
		return m.NewDialogFullscreen(m.NewScaffold(m.ScaffoldOpts{
			AppBar: m.NewAppBar(m.AppBarOpts{
				Leading: m.NewCloseButton(),
				Title:   m.NewText("New event"),
				Actions: []m.Widget{m.NewTextButton(func() { m.NavigatorPop(bc, "Saved") }, m.NewText("Save"))},
			}),
			Body: m.NewCenter(m.NewText("A fullscreen dialog fills the window.")),
		}))
	}, m.ShowDialogOpts{FullscreenDialog: true})
	awaitResult(s.Context(), s.Mounted, ch, func(v string) { s.setResult("Fullscreen: " + orNone(v)) })
}

func orNone(v string) string {
	if v == "" {
		return "dismissed"
	}
	return v
}

func (s *dialogDemosState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewWrap([]m.Widget{
			demoButton(bc, "AlertDialog", s.showAlert),
			demoButton(bc, "SimpleDialog", s.showSimple),
			demoButton(bc, "Fullscreen", s.showFullscreen),
		}, m.WrapOpts{Spacing: 8, RunSpacing: 8}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		m.NewText("Result: "+s.result, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodyMedium}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// PickerDemos has the buttons of the date, the date range and the time pickers.
type PickerDemos struct{ m.StatefulWidget }

// NewPickerDemos creates the demo.
func NewPickerDemos() *PickerDemos { return &PickerDemos{} }

// CreateState creates the state.
func (w *PickerDemos) CreateState() m.State { return &pickerDemosState{} }

type pickerDemosState struct {
	m.StateOf[*PickerDemos]
	date      time.Time
	rng       m.DateTimeRange
	time      m.Opt[m.TimeOfDay]
	firstDate time.Time
	lastDate  time.Time
}

// pickerToday is the date that the pickers show as today, so that the demo looks
// the same on every day.
func pickerToday() time.Time { return time.Date(2025, time.June, 12, 0, 0, 0, 0, time.Local) }

// InitState opens the picker that GLUTTER_DEV_OPEN asks for.
func (s *pickerDemosState) InitState() {
	s.date = time.Date(2025, time.June, 18, 0, 0, 0, 0, time.Local)
	s.firstDate = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local)
	s.lastDate = time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local)
	var open func()
	switch devOpen(os.Getenv) {
	case "datepicker":
		open = s.showDate
	case "datepickerinput":
		open = func() { s.showDateMode(m.DatePickerEntryModeInput) }
	case "daterange":
		open = s.showRange
	case "timepicker":
		open = s.showTime
	case "timepickerinput":
		open = func() { s.showTimeMode(m.TimePickerEntryModeInput) }
	default:
		return
	}
	m.WidgetsBindingOf(s.Context()).Gestures().NewTimer(devOpenDelay, func() {
		if s.Mounted() {
			open()
		}
	})
}

func (s *pickerDemosState) showDate() { s.showDateMode(m.DatePickerEntryModeCalendar) }

func (s *pickerDemosState) showDateMode(mode m.DatePickerEntryMode) {
	ch := m.ShowDatePicker(s.Context(), s.firstDate, s.lastDate, m.ShowDatePickerOpts{
		InitialDate: s.date, CurrentDate: pickerToday(), InitialEntryMode: mode, HelpText: "Select a date",
	})
	awaitResult(s.Context(), s.Mounted, ch, func(d time.Time) {
		if !d.IsZero() {
			s.SetState(func() { s.date = d })
		}
	})
}

func (s *pickerDemosState) showRange() {
	ch := m.ShowDateRangePicker(s.Context(), s.firstDate, s.lastDate, m.ShowDateRangePickerOpts{
		CurrentDate:      pickerToday(),
		InitialDateRange: m.Some(m.NewDateTimeRange(time.Date(2025, time.June, 10, 0, 0, 0, 0, time.Local), time.Date(2025, time.June, 17, 0, 0, 0, 0, time.Local))),
	})
	awaitResult(s.Context(), s.Mounted, ch, func(r m.DateTimeRange) {
		if !r.IsZero() {
			s.SetState(func() { s.rng = r })
		}
	})
}

func (s *pickerDemosState) showTime() { s.showTimeMode(m.TimePickerEntryModeDial) }

func (s *pickerDemosState) showTimeMode(mode m.TimePickerEntryMode) {
	initial := s.time.OrElse(m.TimeOfDay{Hour: 10, Minute: 30})
	ch := m.ShowTimePicker(s.Context(), initial, m.ShowTimePickerOpts{InitialEntryMode: mode})
	awaitResult(s.Context(), s.Mounted, ch, func(t m.Opt[m.TimeOfDay]) {
		if v, ok := t.Get(); ok {
			s.SetState(func() { s.time = m.Some(v) })
		}
	})
}

func (s *pickerDemosState) Build(bc m.BuildContext) m.Widget {
	style := m.ThemeOf(bc).TextTheme.BodyMedium
	rangeText := "none yet"
	if !s.rng.IsZero() {
		rangeText = fmt.Sprintf("%s to %s", s.rng.Start.Format("Jan 2, 2006"), s.rng.End.Format("Jan 2, 2006"))
	}
	timeText := "none yet"
	if t, ok := s.time.Get(); ok {
		timeText = t.Format(bc)
	}
	return m.NewColumn([]m.Widget{
		m.NewWrap([]m.Widget{
			demoButton(bc, "Date", s.showDate),
			demoButton(bc, "Date range", s.showRange),
			demoButton(bc, "Time", s.showTime),
		}, m.WrapOpts{Spacing: 8, RunSpacing: 8}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		m.NewText("Date: "+s.date.Format("Jan 2, 2006"), m.TextOpts{Style: style}),
		m.NewText("Range: "+rangeText, m.TextOpts{Style: style}),
		m.NewText("Time: "+timeText, m.TextOpts{Style: style}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// CalendarDemo is a CalendarDatePicker that shows the date that is selected.
type CalendarDemo struct{ m.StatefulWidget }

// NewCalendarDemo creates the demo.
func NewCalendarDemo() *CalendarDemo { return &CalendarDemo{} }

// CreateState creates the state.
func (w *CalendarDemo) CreateState() m.State { return &calendarDemoState{} }

type calendarDemoState struct {
	m.StateOf[*CalendarDemo]
	date time.Time
}

func (s *calendarDemoState) InitState() {
	s.date = time.Date(2025, time.June, 18, 0, 0, 0, 0, time.Local)
}

func (s *calendarDemoState) Build(bc m.BuildContext) m.Widget {
	return framed(bc, m.NewSizedBox(m.NewCalendarDatePicker(s.date,
		time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local),
		func(d time.Time) { s.SetState(func() { s.date = d }) },
		m.CalendarDatePickerOpts{CurrentDate: pickerToday(),
			SelectableDayPredicate: func(d time.Time) bool { return d.Weekday() != time.Saturday && d.Weekday() != time.Sunday }}),
		m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(380.0)}))
}

// YearPickerDemo is a YearPicker.
type YearPickerDemo struct{ m.StatefulWidget }

// NewYearPickerDemo creates the demo.
func NewYearPickerDemo() *YearPickerDemo { return &YearPickerDemo{} }

// CreateState creates the state.
func (w *YearPickerDemo) CreateState() m.State { return &yearPickerDemoState{} }

type yearPickerDemoState struct {
	m.StateOf[*YearPickerDemo]
	date time.Time
}

func (s *yearPickerDemoState) InitState() {
	s.date = time.Date(2025, time.June, 18, 0, 0, 0, 0, time.Local)
}

func (s *yearPickerDemoState) Build(bc m.BuildContext) m.Widget {
	return framed(bc, m.NewSizedBox(m.NewYearPicker(
		time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2030, time.December, 31, 0, 0, 0, 0, time.Local), s.date,
		func(d time.Time) { s.SetState(func() { s.date = d }) }, m.YearPickerOpts{CurrentDate: pickerToday()}),
		m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(380.0)}))
}
