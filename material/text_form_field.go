// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/widgets"
)

// TextFormFieldOpts are the optional parameters of NewTextFormField: every
// parameter of TextField, and the ones of a FormField.
//
// Flutter: material/text_form_field.dart TextFormField.new
type TextFormFieldOpts struct {
	TextFieldOpts
	// InitialValue is the initial text; it can't be given with a Controller (the
	// text of the controller is the initial one then).
	InitialValue foundation.Opt[string]
	// ForceErrorText forces an error to be shown, and skips the validator.
	ForceErrorText foundation.Opt[string]
	// OnFieldSubmitted is called with the text when the user finishes editing.
	OnFieldSubmitted func(text string)
	// OnSaved is called with the final value when the form is saved.
	OnSaved widgets.FormFieldSetter[string]
	// Validator validates the text; it returns the error text, or none if the
	// text is valid.
	Validator widgets.FormFieldValidator[string]
	// ErrorBuilder builds the widget that shows the error; it can't be given with
	// the ErrorText of the decoration.
	ErrorBuilder widgets.FormFieldErrorBuilder
	// AutovalidateMode is when the field validates; the default is disabled.
	AutovalidateMode widgets.AutovalidateMode
}

// TextFormField is a TextField in a FormField: it keeps its text as the value of
// the form field, validates it, and shows the error in the decoration.
//
// Deviation: Dart's TextFormField is a FormField<String> with its own state;
// this one is a stateful widget that builds a FormField<String> and keeps the
// controller in sync with it (SPEC §12.10). The restoration of the state is not
// ported.
//
// Flutter: material/text_form_field.dart TextFormField
type TextFormField struct {
	widgets.StatefulWidget
	o TextFormFieldOpts
}

// NewTextFormField creates a text form field. In debug builds it panics with
// the tag TextFormFieldInitialValueWithController when it has both an initial
// value and a controller, TextFormFieldErrorBuilderWithErrorText when it has
// both an error builder and the error text of the decoration, and with the tags
// of NewTextField for the other parameters.
//
// Flutter: material/text_form_field.dart TextFormField.new
func NewTextFormField(opts ...TextFormFieldOpts) *TextFormField {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.InitialValue.IsSome() && o.Controller != nil {
			panic(errors.New("TextFormFieldInitialValueWithController", "a TextFormField can't have both an initialValue and a controller"))
		}
		if o.ErrorBuilder != nil && o.Decoration != nil && o.Decoration.ErrorText.IsSome() {
			panic(errors.New("TextFormFieldErrorBuilderWithErrorText", "declaring both errorBuilder and decoration.errorText is not supported"))
		}
	}
	// Check the parameters of the text field now, as Flutter's constructor does.
	NewTextField(o.TextFieldOpts)
	return &TextFormField{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Controller is the controller of the text; nil when the field makes its own.
//
// Flutter: material/text_form_field.dart TextFormField.controller
func (w *TextFormField) Controller() *widgets.TextEditingController { return w.o.Controller }

// InitialValue is the text that the field starts with, and is reset to.
//
// Flutter: material/text_form_field.dart TextFormField.initialValue
func (w *TextFormField) InitialValue() string {
	if w.o.Controller != nil {
		return w.o.Controller.Text()
	}
	return w.o.InitialValue.OrElse("")
}

func (w *TextFormField) enabled() bool {
	if e, ok := w.o.Enabled.Get(); ok {
		return e
	}
	return w.o.Decoration == nil || !w.o.Decoration.Disabled
}

// CreateState creates the state.
//
// Flutter: material/text_form_field.dart TextFormField.createState
func (w *TextFormField) CreateState() widgets.State { return &textFormFieldState{} }

// textFormFieldState keeps the controller of the text and the form field in
// sync.
//
// Flutter: material/text_form_field.dart _TextFormFieldState
type textFormFieldState struct {
	widgets.StateOf[*TextFormField]
	controller     *widgets.TextEditingController
	field          *widgets.FormFieldState[string]
	initialValue   string
	controllerSeen foundation.ListenerHandle
	listeningTo    *widgets.TextEditingController
}

func (s *textFormFieldState) effectiveController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.controller
}

// InitState creates the local controller, or listens to the given one.
//
// Flutter: material/text_form_field.dart _TextFormFieldState.initState
func (s *textFormFieldState) InitState() {
	w := s.Widget()
	if w.o.Controller == nil {
		s.controller = widgets.NewTextEditingController(w.o.InitialValue.OrElse(""))
	} else {
		s.listenTo(w.o.Controller)
	}
	s.initialValue = w.InitialValue()
}

func (s *textFormFieldState) listenTo(c *widgets.TextEditingController) {
	s.listeningTo = c
	s.controllerSeen = c.AddListener(s.handleControllerChanged)
}

func (s *textFormFieldState) stopListening() {
	if s.listeningTo != nil {
		s.listeningTo.RemoveListener(s.controllerSeen)
		s.listeningTo = nil
	}
}

// DidUpdateWidget follows the controller of the new widget.
//
// Flutter: material/text_form_field.dart _TextFormFieldState.didUpdateWidget
func (s *textFormFieldState) DidUpdateWidget(old *TextFormField) {
	w := s.Widget()
	if w.o.Controller == old.o.Controller {
		return
	}
	s.stopListening()
	if w.o.Controller != nil {
		s.listenTo(w.o.Controller)
	}
	if old.o.Controller != nil && w.o.Controller == nil {
		s.controller = widgets.NewTextEditingControllerFromValue(old.o.Controller.Value())
	}
	if w.o.Controller != nil {
		if s.field != nil {
			s.field.SetValue(foundation.Some(w.o.Controller.Text()))
		}
		if old.o.Controller == nil && s.controller != nil {
			s.controller.Dispose()
			s.controller = nil
		}
	}
}

// Dispose stops listening and releases the local controller.
//
// Flutter: material/text_form_field.dart _TextFormFieldState.dispose
func (s *textFormFieldState) Dispose() {
	s.stopListening()
	if s.controller != nil {
		s.controller.Dispose()
	}
}

// handleDidChange puts the new value in the controller.
//
// Flutter: material/text_form_field.dart _TextFormFieldState.didChange
func (s *textFormFieldState) handleDidChange(value foundation.Opt[string]) {
	c := s.effectiveController()
	if text := value.OrElse(""); c.Text() != text {
		c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: text}))
	}
}

// handleReset puts the initial value in the controller and tells onChanged.
//
// Flutter: material/text_form_field.dart _TextFormFieldState.reset
func (s *textFormFieldState) handleReset() {
	c := s.effectiveController()
	c.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: s.initialValue}))
	if s.field != nil {
		s.field.SetValue(foundation.Some(s.initialValue))
	}
	if f := s.Widget().o.OnChanged; f != nil {
		f(c.Text())
	}
}

func (s *textFormFieldState) handleControllerChanged() {
	if s.field == nil {
		return
	}
	if text := s.effectiveController().Text(); s.field.Value() != foundation.Some(text) {
		s.field.DidChange(foundation.Some(text))
	}
}

// Build builds the form field.
//
// Flutter: material/text_form_field.dart TextFormField
func (s *textFormFieldState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewFormField(s.buildField, widgets.FormFieldOpts[string]{
		OnSaved:          w.o.OnSaved,
		OnReset:          s.handleReset,
		OnDidChange:      s.handleDidChange,
		ForceErrorText:   w.o.ForceErrorText,
		Validator:        w.o.Validator,
		ErrorBuilder:     w.o.ErrorBuilder,
		InitialValue:     foundation.Some(w.InitialValue()),
		Disabled:         !w.enabled(),
		AutovalidateMode: w.o.AutovalidateMode,
	})
}

// buildField builds the text field of the form field.
//
// Flutter: material/text_form_field.dart TextFormField.new
func (s *textFormFieldState) buildField(field *widgets.FormFieldState[string]) widgets.Widget {
	s.field = field
	w := s.Widget()
	decoration := InputDecoration{}
	if w.o.Decoration != nil {
		decoration = *w.o.Decoration
	}
	effective := decoration.ApplyDefaults(InputDecorationThemeOf(field.Context()))
	if errorText, ok := field.ErrorText().Get(); ok {
		if w.o.ErrorBuilder != nil {
			effective = effective.CopyWith(InputDecorationCopy{Error: foundation.Some(w.o.ErrorBuilder(field.Context(), errorText))})
		} else {
			effective = effective.CopyWith(InputDecorationCopy{ErrorText: foundation.Some(errorText)})
		}
	}
	o := w.o.TextFieldOpts
	o.Key = nil
	o.Controller = s.effectiveController()
	o.Decoration = &effective
	o.NoDecoration = false
	o.Enabled = foundation.Some(w.enabled())
	o.OnSubmitted = w.o.OnFieldSubmitted
	o.OnChanged = func(value string) {
		field.DidChange(foundation.Some(value))
		if w.o.OnChanged != nil {
			w.o.OnChanged(value)
		}
	}
	return NewTextField(o)
}
