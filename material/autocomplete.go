// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// AutocompleteOpts are the optional parameters of NewAutocomplete.
//
// Flutter: material/autocomplete.dart Autocomplete.new
type AutocompleteOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// DisplayStringForOption is the string that the field shows for an option; the
	// default is the default string of the option.
	DisplayStringForOption widgets.AutocompleteOptionToString[T]
	// FieldViewBuilder builds the field; the default is a TextFormField.
	FieldViewBuilder widgets.AutocompleteFieldViewBuilder
	// FocusNode is the focus node of the field; give it with TextEditingController.
	FocusNode *widgets.FocusNode
	// OnSelected is called when the user selects an option.
	OnSelected widgets.AutocompleteOnSelected[T]
	// OptionsMaxHeight is the maximum height of the options; the default is 200.
	OptionsMaxHeight foundation.Opt[float64]
	// OptionsViewBuilder builds the view of the options; the default is a list in
	// a Material.
	OptionsViewBuilder widgets.AutocompleteOptionsViewBuilder[T]
	// OptionsViewOpenDirection is the direction in which the options open; the
	// default is down.
	OptionsViewOpenDirection widgets.OptionsViewOpenDirection
	// TextEditingController is the controller of the field; give it with FocusNode.
	TextEditingController *widgets.TextEditingController
	// InitialValue is the initial value of the field.
	InitialValue foundation.Opt[services.TextEditingValue]
}

// Autocomplete is a widget for helping the user make a selection by entering some
// text and choosing from among a list of options.
//
// Flutter: material/autocomplete.dart Autocomplete
type Autocomplete[T comparable] struct {
	widgets.StatelessWidget
	optionsBuilder widgets.AutocompleteOptionsBuilder[T]
	o              AutocompleteOpts[T]
}

// NewAutocomplete creates the widget. The optionsBuilder makes the options for the
// text of the field.
//
// Flutter: material/autocomplete.dart Autocomplete.new
func NewAutocomplete[T comparable](optionsBuilder widgets.AutocompleteOptionsBuilder[T], opts ...AutocompleteOpts[T]) *Autocomplete[T] {
	o := firstOpts(opts)
	if o.DisplayStringForOption == nil {
		o.DisplayStringForOption = func(option T) string { return fmt.Sprint(option) }
	}
	return &Autocomplete[T]{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), optionsBuilder: optionsBuilder, o: o}
}

// Build builds a RawAutocomplete with a field and a view of the options that are
// Material.
//
// Flutter: material/autocomplete.dart Autocomplete.build
func (w *Autocomplete[T]) Build(widgets.BuildContext) widgets.Widget {
	o := w.o
	direction := o.OptionsViewOpenDirection
	if direction == widgets.OptionsViewOpenDirectionUnspecified {
		direction = widgets.OptionsViewOpenDirectionDown
	}
	fieldViewBuilder := o.FieldViewBuilder
	if fieldViewBuilder == nil {
		fieldViewBuilder = defaultAutocompleteFieldViewBuilder
	}
	optionsViewBuilder := o.OptionsViewBuilder
	if optionsViewBuilder == nil {
		display := o.DisplayStringForOption
		maxHeight := o.OptionsMaxHeight.OrElse(200.0)
		optionsViewBuilder = func(_ widgets.BuildContext, onSelected widgets.AutocompleteOnSelected[T], options []T) widgets.Widget {
			return &autocompleteOptions[T]{StatelessWidget: widgets.NewStatelessWidgetBase(), displayStringForOption: display, onSelected: onSelected,
				options: options, optionsMaxHeight: maxHeight}
		}
	}
	return widgets.NewRawAutocomplete(optionsViewBuilder, w.optionsBuilder, widgets.RawAutocompleteOpts[T]{
		DisplayStringForOption: o.DisplayStringForOption, FieldViewBuilder: fieldViewBuilder, FocusNode: o.FocusNode,
		TextEditingController: o.TextEditingController, InitialValue: o.InitialValue, OptionsViewOpenDirection: direction,
		OnSelected: o.OnSelected,
	})
}

// defaultAutocompleteFieldViewBuilder is the field of an Autocomplete: a text form
// field.
//
// Flutter: material/autocomplete.dart Autocomplete._defaultFieldViewBuilder
func defaultAutocompleteFieldViewBuilder(_ widgets.BuildContext, controller *widgets.TextEditingController, focusNode *widgets.FocusNode, onFieldSubmitted foundation.VoidCallback) widgets.Widget {
	return NewTextFormField(TextFormFieldOpts{Controller: controller, FocusNode: focusNode, OnFieldSubmitted: func(string) { onFieldSubmitted() }})
}

// autocompleteOptions is the view of the options of an Autocomplete.
//
// Flutter: material/autocomplete.dart _AutocompleteOptions
type autocompleteOptions[T comparable] struct {
	widgets.StatelessWidget
	displayStringForOption widgets.AutocompleteOptionToString[T]
	onSelected             widgets.AutocompleteOnSelected[T]
	options                []T
	optionsMaxHeight       float64
}

func (w *autocompleteOptions[T]) Build(bc widgets.BuildContext) widgets.Widget {
	highlightedIndex := widgets.AutocompleteHighlightedOptionOf(bc)
	return NewMaterial(widgets.NewConstrainedBox(
		rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxHeight: foundation.Some(w.optionsMaxHeight)}),
		&autocompleteOptionsList[T]{StatefulWidget: widgets.NewStatefulWidgetBase(), displayStringForOption: w.displayStringForOption,
			highlightedIndex: highlightedIndex, onSelected: w.onSelected, options: w.options}),
		MaterialOpts{Elevation: 4.0})
}

// autocompleteOptionsList is the list of the options: it scrolls the highlighted
// option into view.
//
// Flutter: material/autocomplete.dart _AutocompleteOptionsList
type autocompleteOptionsList[T comparable] struct {
	widgets.StatefulWidget
	displayStringForOption widgets.AutocompleteOptionToString[T]
	highlightedIndex       int
	onSelected             widgets.AutocompleteOnSelected[T]
	options                []T
}

func (w *autocompleteOptionsList[T]) CreateState() widgets.State {
	return &autocompleteOptionsListState[T]{keys: map[T]*widgets.GlobalKey[widgets.State]{}}
}

type autocompleteOptionsListState[T comparable] struct {
	widgets.StateOf[*autocompleteOptionsList[T]]
	scrollController *widgets.ScrollControllerBase
	// keys are the keys of the options, which stand for Flutter's GlobalObjectKey of
	// each option (a GlobalObjectKey cannot find its context in Go).
	keys map[T]*widgets.GlobalKey[widgets.State]
}

func (s *autocompleteOptionsListState[T]) key(option T) *widgets.GlobalKey[widgets.State] {
	k, ok := s.keys[option]
	if !ok {
		k = widgets.NewGlobalKey[widgets.State]()
		s.keys[option] = k
	}
	return k
}

func (s *autocompleteOptionsListState[T]) InitState() {
	s.scrollController = widgets.NewScrollController()
}

func (s *autocompleteOptionsListState[T]) DidUpdateWidget(old *autocompleteOptionsList[T]) {
	w := s.Widget()
	if w.highlightedIndex == old.highlightedIndex {
		return
	}
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		if !s.Mounted() || w.highlightedIndex < 0 || w.highlightedIndex >= len(w.options) {
			return
		}
		highlighted := s.key(w.options[w.highlightedIndex]).CurrentContext()
		if highlighted == nil {
			if w.highlightedIndex == 0 {
				s.scrollController.JumpTo(0.0)
			} else {
				s.scrollController.JumpTo(s.scrollController.Position().MaxScrollExtent())
			}
		} else {
			widgets.ScrollableEnsureVisible(highlighted, widgets.EnsureVisibleOpts{Alignment: 0.5})
		}
	})
}

func (s *autocompleteOptionsListState[T]) Dispose() { s.scrollController.Dispose() }

func (s *autocompleteOptionsListState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	highlightedIndex := widgets.AutocompleteHighlightedOptionOf(bc)
	return widgets.NewListViewBuilder(func(_ widgets.BuildContext, index int) widgets.Widget {
		if index < 0 || index >= len(w.options) {
			return nil
		}
		option := w.options[index]
		return widgets.NewSemantics(NewInkWell(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			var color foundation.Opt[ui.Color]
			if highlightedIndex == index {
				color = foundation.Some(ThemeOf(bc).FocusColor)
			}
			return widgets.NewContainer(widgets.NewText(w.displayStringForOption(option)),
				widgets.ContainerOpts{Color: color, Padding: painting.NewEdgeInsetsAll(16.0)})
		}), InkWellOpts{Key: s.key(option), OnTap: func() { w.onSelected(option) }}), widgets.SemanticsOpts{Button: foundation.Some(true)})
	}, widgets.ListViewOpts{Padding: painting.EdgeInsetsZero, ShrinkWrap: true, Controller: s.scrollController, ItemCount: foundation.Some(len(w.options))})
}
