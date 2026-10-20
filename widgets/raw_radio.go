// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
)

// RadioBuilder builds the painting of a RawRadio from the state of the toggle.
//
// Flutter: widgets/raw_radio.dart RadioBuilder
type RadioBuilder func(bc BuildContext, state *ToggleableStateMixin) Widget

// RawRadioOpts are the optional parameters of NewRawRadio.
//
// Flutter: widgets/raw_radio.dart RawRadio.new
type RawRadioOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// RawRadio is a radio button without a look: it has the behavior of a radio
// (the toggle animations, the focus, the hover, the keys and the registry of
// the group) and builds the painting with a builder.
//
// Flutter: widgets/raw_radio.dart RawRadio
type RawRadio[T comparable] struct {
	StatefulWidget
	value         T
	mouseCursor   WidgetStateProperty[services.MouseCursor]
	toggleable    bool
	focusNode     *FocusNode
	autofocus     bool
	groupRegistry RadioGroupRegistry[T]
	enabled       bool
	builder       RadioBuilder
}

// NewRawRadio creates the widget. In debug builds it panics with the tag
// RawRadioNoRegistry when it is enabled and has no registry.
//
// Flutter: widgets/raw_radio.dart RawRadio.new
func NewRawRadio[T comparable](value T, mouseCursor WidgetStateProperty[services.MouseCursor], toggleable bool,
	focusNode *FocusNode, autofocus bool, groupRegistry RadioGroupRegistry[T], enabled bool,
	builder RadioBuilder, opts ...RawRadioOpts) *RawRadio[T] {
	if foundation.DebugMode && enabled && groupRegistry == nil {
		panic(errors.New("RawRadioNoRegistry", "an enabled raw radio must have a registry"))
	}
	o := firstOpts(opts)
	return &RawRadio[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts(o)), value: value, mouseCursor: mouseCursor,
		toggleable: toggleable, focusNode: focusNode, autofocus: autofocus, groupRegistry: groupRegistry,
		enabled: enabled, builder: builder}
}

// Value is the value that this radio stands for.
//
// Flutter: widgets/raw_radio.dart RawRadio.value
func (w *RawRadio[T]) Value() T { return w.value }

// MouseCursor is the cursor over the radio, in each state.
//
// Flutter: widgets/raw_radio.dart RawRadio.mouseCursor
func (w *RawRadio[T]) MouseCursor() WidgetStateProperty[services.MouseCursor] { return w.mouseCursor }

// Toggleable is whether the radio can be deselected by selecting it again.
//
// Flutter: widgets/raw_radio.dart RawRadio.toggleable
func (w *RawRadio[T]) Toggleable() bool { return w.toggleable }

// FocusNode is the focus node of the radio.
//
// Flutter: widgets/raw_radio.dart RawRadio.focusNode
func (w *RawRadio[T]) FocusNode() *FocusNode { return w.focusNode }

// Autofocus is whether the radio asks for the focus when it is built.
//
// Flutter: widgets/raw_radio.dart RawRadio.autofocus
func (w *RawRadio[T]) Autofocus() bool { return w.autofocus }

// GroupRegistry is the registry of the group of the radio.
//
// Flutter: widgets/raw_radio.dart RawRadio.groupRegistry
func (w *RawRadio[T]) GroupRegistry() RadioGroupRegistry[T] { return w.groupRegistry }

// Enabled is whether the radio responds to input.
//
// Flutter: widgets/raw_radio.dart RawRadio.enabled
func (w *RawRadio[T]) Enabled() bool { return w.enabled }

// Builder builds the painting of the radio.
//
// Flutter: widgets/raw_radio.dart RawRadio.builder
func (w *RawRadio[T]) Builder() RadioBuilder { return w.builder }

// CreateState creates the state.
//
// Flutter: widgets/raw_radio.dart RawRadio.createState
func (w *RawRadio[T]) CreateState() State { return &rawRadioState[T]{} }

type rawRadioState[T comparable] struct {
	StateOf[*RawRadio[T]]
	TickerProvider
	ToggleableStateMixin
	RadioClientRegistration[T]
}

func (s *rawRadioState[T]) InitState() {
	s.SetRegistry(s, s.Widget().groupRegistry)
	s.Init(s, s, ToggleableStateMixinOpts{IsInteractive: func() bool { return s.Widget().enabled }})
}

func (s *rawRadioState[T]) DidUpdateWidget(*RawRadio[T]) {
	s.SetRegistry(s, s.Widget().groupRegistry)
	s.AnimateToValue() // The registry's group value may have changed
}

func (s *rawRadioState[T]) Dispose() {
	s.DisposeToggleable()
	s.SetRegistry(s, nil)
}

// FocusNode is the focus node of the radio.
//
// Flutter: widgets/raw_radio.dart _RawRadioState.focusNode
func (s *rawRadioState[T]) FocusNode() *FocusNode { return s.Widget().focusNode }

// Enabled is whether the radio responds to input.
//
// Flutter: widgets/raw_radio.dart _RawRadioState.enabled
func (s *rawRadioState[T]) Enabled() bool { return s.IsInteractive() }

// RadioValue is the value of the radio.
//
// Flutter: widgets/raw_radio.dart _RawRadioState.radioValue
func (s *rawRadioState[T]) RadioValue() T { return s.Widget().value }

// Tristate is whether the radio can be deselected.
//
// Flutter: widgets/raw_radio.dart _RawRadioState.tristate
func (s *rawRadioState[T]) Tristate() bool { return s.Widget().toggleable }

func (s *rawRadioState[T]) handleChanged(selected foundation.Opt[bool]) {
	registry := s.Registry()
	if v, ok := selected.Get(); ok && !v {
		return
	}
	if selected == foundation.Some(true) {
		registry.OnChanged(foundation.Some(s.Widget().value))
	} else {
		registry.OnChanged(foundation.None[T]())
	}
}

func (s *rawRadioState[T]) ToggleableOnChanged() func(foundation.Opt[bool]) {
	if s.Registry() != nil {
		return s.handleChanged
	}
	return nil
}

func (s *rawRadioState[T]) ToggleableTristate() bool { return s.Widget().toggleable }

func (s *rawRadioState[T]) ToggleableValue() foundation.Opt[bool] {
	registry := s.Registry()
	if registry == nil {
		return foundation.Some(false)
	}
	return foundation.Some(registry.GroupValue() == foundation.Some(s.Widget().value))
}

func (s *rawRadioState[T]) Build(bc BuildContext) Widget {
	w := s.Widget()
	accessibilitySelected := foundation.None[bool]()
	semanticsHint := foundation.None[string]()
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		value := s.ToggleableValue()
		accessibilitySelected = value
		if v, _ := value.Get(); !v {
			semanticsHint = foundation.Some(WidgetsLocalizationsOf(bc).RadioButtonUnselectedLabel())
		}
	}
	return NewSemantics(
		s.BuildToggleableWithChild(w.builder(bc, &s.ToggleableStateMixin),
			ToggleableOpts{FocusNode: w.focusNode, Autofocus: w.autofocus, MouseCursor: w.mouseCursor}),
		SemanticsOpts{
			InMutuallyExclusiveGroup: foundation.Some(true),
			Checked:                  s.ToggleableValue(),
			Selected:                 accessibilitySelected,
			Hint:                     semanticsHint,
		})
}
