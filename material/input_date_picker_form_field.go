// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/widgets"
)

// InputDatePickerFormFieldOpts are the optional parameters of
// NewInputDatePickerFormField.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.new
type InputDatePickerFormFieldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialDate is the date that the field starts with; the zero time for none.
	InitialDate time.Time
	// OnDateSubmitted is called when the user submits a valid date.
	OnDateSubmitted func(date time.Time)
	// OnDateSaved is called when the form is saved with a valid date.
	OnDateSaved func(date time.Time)
	// SelectableDayPredicate says which days are valid; the default is all of
	// them.
	SelectableDayPredicate SelectableDayPredicate
	// ErrorFormatText is the error text for an entry that is not a date; the
	// default is the invalid date format label of the localizations.
	ErrorFormatText string
	// ErrorInvalidText is the error text for a date that is out of range or not
	// selectable; the default is the date out of range label.
	ErrorInvalidText string
	// FieldHintText is the hint of the field; the default is the date help text of
	// the calendar.
	FieldHintText string
	// FieldLabelText is the label of the field; the default is the date input
	// label of the localizations.
	FieldLabelText string
	// KeyboardType is the keyboard of the field; the default is datetime.
	KeyboardType foundation.Opt[services.TextInputType]
	// Autofocus gives the field the focus when it is built, with its text
	// selected.
	Autofocus bool
	// AcceptEmptyDate makes an empty entry valid.
	AcceptEmptyDate bool
	// FocusNode is the focus node of the field.
	FocusNode *widgets.FocusNode
	// CalendarDelegate is the calendar system; the default is the Gregorian one.
	CalendarDelegate CalendarDelegate
}

// InputDatePickerFormField is a TextFormField configured to accept and validate
// a date entered by a user.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField
type InputDatePickerFormField struct {
	widgets.StatefulWidget
	initialDate time.Time
	firstDate   time.Time
	lastDate    time.Time
	o           InputDatePickerFormFieldOpts
}

// NewInputDatePickerFormField creates the field. The dates are reduced to the
// date part. In debug builds it panics with the tags
// InputDatePickerLastDateBeforeFirstDate, InputDatePickerInitialDateBeforeFirstDate,
// InputDatePickerInitialDateAfterLastDate and
// InputDatePickerInitialDateNotSelectable when the dates do not fit.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.new
func NewInputDatePickerFormField(firstDate, lastDate time.Time, opts ...InputDatePickerFormFieldOpts) *InputDatePickerFormField {
	o := firstOpts(opts)
	if o.CalendarDelegate == nil {
		o.CalendarDelegate = NewGregorianCalendarDelegate()
	}
	d := o.CalendarDelegate
	w := &InputDatePickerFormField{
		StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		firstDate:      d.DateOnly(firstDate), lastDate: d.DateOnly(lastDate), o: o,
	}
	if !o.InitialDate.IsZero() {
		w.initialDate = d.DateOnly(o.InitialDate)
	}
	if foundation.DebugMode {
		switch {
		case w.lastDate.Before(w.firstDate):
			panic(errors.New(tagInputDatePickerLastDateBeforeFirstDate, "lastDate must be on or after firstDate", "lastDate", w.lastDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.Before(w.firstDate):
			panic(errors.New(tagInputDatePickerInitialDateBeforeFirstDate, "initialDate must be on or after firstDate", "initialDate", w.initialDate, "firstDate", w.firstDate))
		case !w.initialDate.IsZero() && w.initialDate.After(w.lastDate):
			panic(errors.New(tagInputDatePickerInitialDateAfterLastDate, "initialDate must be on or before lastDate", "initialDate", w.initialDate, "lastDate", w.lastDate))
		case o.SelectableDayPredicate != nil && !w.initialDate.IsZero() && !o.SelectableDayPredicate(w.initialDate):
			panic(errors.New(tagInputDatePickerInitialDateNotSelectable, "the initialDate must satisfy the selectableDayPredicate", "initialDate", w.initialDate))
		}
	}
	return w
}

// InitialDate is the date that the field starts with; the zero time for none.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.initialDate
func (w *InputDatePickerFormField) InitialDate() time.Time { return w.initialDate }

// FirstDate is the earliest date that is valid.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.firstDate
func (w *InputDatePickerFormField) FirstDate() time.Time { return w.firstDate }

// LastDate is the latest date that is valid.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.lastDate
func (w *InputDatePickerFormField) LastDate() time.Time { return w.lastDate }

// CreateState creates the state.
//
// Flutter: material/input_date_picker_form_field.dart InputDatePickerFormField.createState
func (w *InputDatePickerFormField) CreateState() widgets.State {
	return &inputDatePickerFormFieldState{}
}

type inputDatePickerFormFieldState struct {
	widgets.StateOf[*InputDatePickerFormField]
	controller   *widgets.TextEditingController
	selectedDate time.Time
	inputText    string
	autoSelected bool
}

func (s *inputDatePickerFormFieldState) InitState() {
	s.controller = widgets.NewTextEditingController()
	s.selectedDate = s.Widget().initialDate
}

func (s *inputDatePickerFormFieldState) Dispose() { s.controller.Dispose() }

func (s *inputDatePickerFormFieldState) DidChangeDependencies() { s.updateValueForSelectedDate() }

func (s *inputDatePickerFormFieldState) DidUpdateWidget(old *InputDatePickerFormField) {
	w := s.Widget()
	if !w.initialDate.Equal(old.initialDate) {
		// Can't update the form field in the middle of a build, so do it in the next frame.
		widgets.WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) {
			if !s.Mounted() {
				return
			}
			s.SetState(func() {
				s.selectedDate = w.initialDate
				s.updateValueForSelectedDate()
			})
		})
	}
}

func (s *inputDatePickerFormFieldState) updateValueForSelectedDate() {
	w := s.Widget()
	if !s.selectedDate.IsZero() {
		localizations := MaterialLocalizationsOf(s.Context())
		s.inputText = w.o.CalendarDelegate.FormatCompactDate(s.selectedDate, localizations)
		value := services.NewTextEditingValue(services.TextEditingValueOpts{Text: s.inputText})
		if w.o.Autofocus && !s.autoSelected {
			value = value.CopyWith(services.TextEditingValueCopy{
				Selection: foundation.Some(services.NewTextSelection(0, len([]rune(s.inputText)))),
			})
			s.autoSelected = true
		}
		s.controller.SetValue(value)
	} else {
		s.inputText = ""
		s.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: s.inputText}))
	}
}

func (s *inputDatePickerFormFieldState) parseDate(text string) (time.Time, bool) {
	return s.Widget().o.CalendarDelegate.ParseCompactDate(text, MaterialLocalizationsOf(s.Context()))
}

func (s *inputDatePickerFormFieldState) isValidAcceptableDate(date time.Time, ok bool) bool {
	w := s.Widget()
	return ok && !date.Before(w.firstDate) && !date.After(w.lastDate) &&
		(w.o.SelectableDayPredicate == nil || w.o.SelectableDayPredicate(date))
}

func (s *inputDatePickerFormFieldState) validateDate(value foundation.Opt[string]) foundation.Opt[string] {
	w := s.Widget()
	text := value.OrElse("")
	if text == "" && w.o.AcceptEmptyDate {
		return foundation.None[string]()
	}
	date, ok := s.parseDate(text)
	switch {
	case !ok:
		return foundation.Some(orDefault(w.o.ErrorFormatText, MaterialLocalizationsOf(s.Context()).InvalidDateFormatLabel()))
	case !s.isValidAcceptableDate(date, ok):
		return foundation.Some(orDefault(w.o.ErrorInvalidText, MaterialLocalizationsOf(s.Context()).DateOutOfRangeLabel()))
	}
	return foundation.None[string]()
}

// orDefault returns the text, or the default for the empty text.
func orDefault(text, def string) string {
	if text == "" {
		return def
	}
	return text
}

func (s *inputDatePickerFormFieldState) updateDate(text string, callback func(time.Time)) {
	date, ok := s.parseDate(text)
	if s.isValidAcceptableDate(date, ok) {
		s.selectedDate = date
		s.inputText = text
		if callback != nil {
			callback(s.selectedDate)
		}
	}
}

func (s *inputDatePickerFormFieldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := ThemeOf(bc)
	localizations := MaterialLocalizationsOf(bc)
	datePickerTheme := theme.DatePickerTheme
	inputTheme := InputDecorationThemeOf(bc)
	var effectiveBorder InputBorder
	switch {
	case datePickerTheme.InputDecorationTheme != nil && datePickerTheme.InputDecorationTheme.Border != nil:
		effectiveBorder = datePickerTheme.InputDecorationTheme.Border
	case inputTheme.Border != nil:
		effectiveBorder = inputTheme.Border
	case theme.UseMaterial3:
		effectiveBorder = NewOutlineInputBorder()
	default:
		effectiveBorder = NewUnderlineInputBorder()
	}
	merged := inputTheme
	if datePickerTheme.InputDecorationTheme != nil {
		merged = inputTheme.Merge(*datePickerTheme.InputDecorationTheme)
	}
	merged = merged.CopyWith(InputDecorationThemeDataCopy{Border: foundation.Some(effectiveBorder)})
	decoration := InputDecoration{
		HintText:  foundation.Some(orDefault(w.o.FieldHintText, w.o.CalendarDelegate.DateHelpText(localizations))),
		LabelText: foundation.Some(orDefault(w.o.FieldLabelText, localizations.DateInputLabel())),
	}.ApplyDefaults(merged)
	return widgets.NewSemantics(NewTextFormField(TextFormFieldOpts{
		TextFieldOpts: TextFieldOpts{
			Decoration:   &decoration,
			KeyboardType: foundation.Some(w.o.KeyboardType.OrElse(services.TextInputTypeDatetime)),
			Autofocus:    w.o.Autofocus,
			Controller:   s.controller,
			FocusNode:    w.o.FocusNode,
		},
		Validator:        s.validateDate,
		OnSaved:          func(v foundation.Opt[string]) { s.updateDate(v.OrElse(""), w.o.OnDateSaved) },
		OnFieldSubmitted: func(text string) { s.updateDate(text, w.o.OnDateSubmitted) },
	}), widgets.SemanticsOpts{Container: true})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagInputDatePickerInitialDateAfterLastDate   = "InputDatePickerInitialDateAfterLastDate"
	tagInputDatePickerInitialDateBeforeFirstDate = "InputDatePickerInitialDateBeforeFirstDate"
	tagInputDatePickerInitialDateNotSelectable   = "InputDatePickerInitialDateNotSelectable"
	tagInputDatePickerLastDateBeforeFirstDate    = "InputDatePickerLastDateBeforeFirstDate"
)
