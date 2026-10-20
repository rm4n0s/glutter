// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DropdownButtonFormFieldOpts are the optional parameters of
// NewDropdownButtonFormField: the ones of DropdownButton, and the ones of a
// FormField.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.new
type DropdownButtonFormFieldOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SelectedItemBuilder builds the widgets that show the selected item.
	SelectedItemBuilder DropdownButtonBuilder
	// InitialValue is the value that is selected at first.
	InitialValue foundation.Opt[T]
	// Hint is shown when no item is selected; the default is the hintText of the
	// decoration.
	Hint widgets.Widget
	// DisabledHint is shown when no item is selected and the button is disabled;
	// the default is the hint.
	DisabledHint widgets.Widget
	// OnTap is called when the button is tapped.
	OnTap func()
	// Elevation is the elevation of the menu; the default is 8.
	Elevation foundation.Opt[int]
	// Style is the text style of the items.
	Style *painting.TextStyle
	// Icon is the icon next to the button; the default is Icons.arrow_drop_down.
	Icon widgets.Widget
	// IconDisabledColor is the color of the icon when the button is disabled.
	IconDisabledColor foundation.Opt[ui.Color]
	// IconEnabledColor is the color of the icon when the button is enabled.
	IconEnabledColor foundation.Opt[ui.Color]
	// IconSize is the size of the icon; the default is 24.
	IconSize foundation.Opt[float64]
	// NotDense makes the button as tall as its items (Flutter's isDense is true
	// by default for the form field).
	NotDense bool
	// IsExpanded makes the button as wide as its parent.
	IsExpanded bool
	// ItemHeight is the height of the items; the default is none, which lets
	// them take the height of their content.
	ItemHeight foundation.Opt[float64]
	// FocusColor is the color of the field while the button has the focus.
	FocusColor foundation.Opt[ui.Color]
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus asks for the focus when the button is built.
	Autofocus bool
	// DropdownColor is the background color of the menu.
	DropdownColor foundation.Opt[ui.Color]
	// Decoration is the decoration of the field; the defaults of the theme apply
	// to it.
	Decoration *InputDecoration
	// OnSaved is called with the final value when the form is saved.
	OnSaved widgets.FormFieldSetter[T]
	// Validator validates the value; it returns the error text, or none if the
	// value is valid.
	Validator widgets.FormFieldValidator[T]
	// ErrorBuilder builds the widget that shows the error; it can't be given with
	// the ErrorText of the decoration.
	ErrorBuilder widgets.FormFieldErrorBuilder
	// ForceErrorText forces an error to be shown, and skips the validator.
	ForceErrorText foundation.Opt[string]
	// AutovalidateMode is when the field validates; the default is disabled.
	AutovalidateMode widgets.AutovalidateMode
	// MenuMaxHeight is the maximum height of the menu.
	MenuMaxHeight foundation.Opt[float64]
	// EnableFeedback says whether the gestures give acoustic and haptic feedback.
	EnableFeedback foundation.Opt[bool]
	// Alignment is how the items are aligned in the button; the default is
	// AlignmentDirectional.centerStart.
	Alignment painting.AlignmentGeometry
	// BorderRadius is the radius of the corners of the menu.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Padding is the space around the button's content.
	Padding painting.EdgeInsetsGeometry
	// NoBarrierDismissible keeps a tap outside the menu from closing it.
	NoBarrierDismissible bool
	// MouseCursor is the cursor over the button.
	MouseCursor services.MouseCursor
	// DropdownMenuItemMouseCursor is the cursor over the items of the menu.
	DropdownMenuItemMouseCursor services.MouseCursor
}

// DropdownButtonFormField is a DropdownButton in a FormField, with an
// InputDecorator: it keeps the selected value as the value of the form field,
// validates it, and shows the error in the decoration.
//
// Deviation: Dart's DropdownButtonFormField is a FormField<T> with its own
// state; this one is a stateful widget that builds a FormField<T> (SPEC §12.10).
//
// Flutter: material/dropdown.dart DropdownButtonFormField
type DropdownButtonFormField[T comparable] struct {
	widgets.StatefulWidget
	items     []*DropdownMenuItem[T]
	onChanged func(value foundation.Opt[T])
	o         DropdownButtonFormFieldOpts[T]
}

// NewDropdownButtonFormField creates the field; a nil list of items or a nil
// onChanged disables it. In debug builds it panics with the tag
// DropdownButtonFormFieldValueNotUnique when the initial value is not the value
// of exactly one item, and with DropdownButtonFormFieldErrorBuilderWithErrorText
// when it has both an error builder and the error text of the decoration.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.new
func NewDropdownButtonFormField[T comparable](items []*DropdownMenuItem[T], onChanged func(value foundation.Opt[T]), opts ...DropdownButtonFormFieldOpts[T]) *DropdownButtonFormField[T] {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if v, ok := o.InitialValue.Get(); ok && len(items) > 0 {
			n := 0
			for _, it := range items {
				if it.o.Value == foundation.Some(v) {
					n++
				}
			}
			if n != 1 {
				panic(errors.NewErrf("DropdownButtonFormFieldValueNotUnique", "there should be exactly one item with the value of the DropdownButtonFormField: %v; either zero or 2 or more DropdownMenuItems were detected with the same value", v))
			}
		}
		if o.ErrorBuilder != nil && o.Decoration != nil && o.Decoration.ErrorText.IsSome() {
			panic(errors.New("DropdownButtonFormFieldErrorBuilderWithErrorText", "declaring both errorBuilder and decoration.errorText is not supported"))
		}
	}
	return &DropdownButtonFormField[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), items: items, onChanged: onChanged, o: o}
}

// OnChanged is called when the user selects an item, and when the field is
// reset.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.onChanged
func (w *DropdownButtonFormField[T]) OnChanged() func(value foundation.Opt[T]) { return w.onChanged }

// Decoration is the decoration of the field.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.decoration
func (w *DropdownButtonFormField[T]) Decoration() InputDecoration {
	if w.o.Decoration == nil {
		return InputDecoration{}
	}
	return *w.o.Decoration
}

// CreateState creates the state.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.createState
func (w *DropdownButtonFormField[T]) CreateState() widgets.State {
	return &dropdownButtonFormFieldState[T]{}
}

// dropdownButtonFormFieldState tells onChanged about the changes of the value of
// the form field.
//
// Flutter: material/dropdown.dart _DropdownButtonFormFieldState
type dropdownButtonFormFieldState[T comparable] struct {
	widgets.StateOf[*DropdownButtonFormField[T]]
	field *widgets.FormFieldState[T]
}

// DidUpdateWidget follows the initial value of the new widget.
//
// Flutter: material/dropdown.dart _DropdownButtonFormFieldState.didUpdateWidget
func (s *dropdownButtonFormFieldState[T]) DidUpdateWidget(old *DropdownButtonFormField[T]) {
	if old.o.InitialValue != s.Widget().o.InitialValue && s.field != nil {
		s.field.SetValue(s.Widget().o.InitialValue)
	}
}

// Build builds the form field.
//
// Flutter: material/dropdown.dart DropdownButtonFormField
func (s *dropdownButtonFormFieldState[T]) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return widgets.NewFormField(s.buildField, widgets.FormFieldOpts[T]{
		OnSaved: w.o.OnSaved,
		OnReset: func() {
			if w.onChanged != nil {
				w.onChanged(s.field.Value())
			}
		},
		OnDidChange: func(value foundation.Opt[T]) {
			if w.onChanged != nil {
				w.onChanged(value)
			}
		},
		ForceErrorText:   w.o.ForceErrorText,
		Validator:        w.o.Validator,
		ErrorBuilder:     w.o.ErrorBuilder,
		InitialValue:     w.o.InitialValue,
		AutovalidateMode: w.o.AutovalidateMode,
	})
}

// buildField builds the button with the decoration of the form field.
//
// Flutter: material/dropdown.dart DropdownButtonFormField.new
func (s *dropdownButtonFormFieldState[T]) buildField(field *widgets.FormFieldState[T]) widgets.Widget {
	s.field = field
	w := s.Widget()
	effectiveDecoration := w.Decoration().ApplyDefaults(InputDecorationThemeOf(field.Context()))
	showSelectedItem := false
	if v, ok := field.Value().Get(); ok {
		for _, it := range w.items {
			if it.o.Value == foundation.Some(v) {
				showSelectedItem = true
			}
		}
	}
	isDropdownEnabled := w.onChanged != nil && len(w.items) > 0
	// If the decoration hintText is provided, use it as the default value for both
	// hint and disabledHint.
	var decorationHint widgets.Widget
	if h, ok := effectiveDecoration.HintText.Get(); ok {
		decorationHint = widgets.NewText(h)
	}
	effectiveHint := w.o.Hint
	if effectiveHint == nil {
		effectiveHint = decorationHint
	}
	effectiveDisabledHint := w.o.DisabledHint
	if effectiveDisabledHint == nil {
		effectiveDisabledHint = effectiveHint
	}
	isHintOrDisabledHintAvailable := effectiveHint != nil || (!isDropdownEnabled && effectiveDisabledHint != nil)
	isEmpty := !showSelectedItem && !isHintOrDisabledHintAvailable

	if errorText, hasError := field.ErrorText().Get(); hasError || effectiveDecoration.HintText.IsSome() {
		var errorWidget widgets.Widget
		if hasError && w.o.ErrorBuilder != nil {
			errorWidget = w.o.ErrorBuilder(field.Context(), errorText)
		}
		c := InputDecorationCopy{Error: foundation.Some(errorWidget)}
		if errorWidget == nil {
			c.ErrorText = field.ErrorText()
		}
		// Clear the decoration hintText because DropdownButton has its own hint logic.
		if effectiveDecoration.HintText.IsSome() {
			c.HintText = foundation.Some("")
		}
		effectiveDecoration = effectiveDecoration.CopyWith(c)
	}

	var onChanged func(foundation.Opt[T])
	if w.onChanged != nil {
		onChanged = field.DidChange
	}
	alignment := w.o.Alignment
	if alignment == nil {
		alignment = painting.AlignmentDirectionalCenterStart
	}
	button := NewDropdownButton(w.items, onChanged, DropdownButtonOpts[T]{
		SelectedItemBuilder:         w.o.SelectedItemBuilder,
		Value:                       field.Value(),
		Hint:                        effectiveHint,
		DisabledHint:                effectiveDisabledHint,
		OnTap:                       w.o.OnTap,
		Elevation:                   foundation.Some(w.o.Elevation.OrElse(8)),
		Style:                       w.o.Style,
		Icon:                        w.o.Icon,
		IconDisabledColor:           w.o.IconDisabledColor,
		IconEnabledColor:            w.o.IconEnabledColor,
		IconSize:                    foundation.Some(w.o.IconSize.OrElse(24.0)),
		IsDense:                     !w.o.NotDense,
		IsExpanded:                  w.o.IsExpanded,
		ItemHeight:                  w.o.ItemHeight,
		NoItemHeight:                w.o.ItemHeight.IsNone(),
		FocusColor:                  w.o.FocusColor,
		FocusNode:                   w.o.FocusNode,
		Autofocus:                   w.o.Autofocus,
		DropdownColor:               w.o.DropdownColor,
		MenuMaxHeight:               w.o.MenuMaxHeight,
		EnableFeedback:              w.o.EnableFeedback,
		Alignment:                   alignment,
		BorderRadius:                w.o.BorderRadius,
		Padding:                     w.o.Padding,
		NoBarrierDismissible:        w.o.NoBarrierDismissible,
		MouseCursor:                 w.o.MouseCursor,
		DropdownMenuItemMouseCursor: w.o.DropdownMenuItemMouseCursor,
	})
	button.inputDecoration = &effectiveDecoration
	button.isEmpty = isEmpty
	// An unfocusable Focus widget so that this widget can detect if its
	// descendants have focus or not.
	return widgets.NewFocus(NewDropdownButtonHideUnderline(button), widgets.FocusOpts{
		CanRequestFocus: foundation.Some(false), SkipTraversal: foundation.Some(true)})
}
