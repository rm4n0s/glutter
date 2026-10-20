// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// RadioGroupRegistry is what the radio buttons of a group register with: it has the
// value of the group and the function to call when another radio is selected.
//
// Flutter: widgets/radio_group.dart RadioGroupRegistry
type RadioGroupRegistry[T comparable] interface {
	// GroupValue is the value of the selected radio, or none.
	//
	// Flutter: widgets/radio_group.dart RadioGroupRegistry.groupValue
	GroupValue() foundation.Opt[T]
	// RegisterClient registers a radio with the group.
	//
	// Flutter: widgets/radio_group.dart RadioGroupRegistry.registerClient
	RegisterClient(radio RadioClient[T])
	// UnregisterClient takes a radio out of the group.
	//
	// Flutter: widgets/radio_group.dart RadioGroupRegistry.unregisterClient
	UnregisterClient(radio RadioClient[T])
	// OnChanged is called with the value of the radio that is selected, or none
	// when a toggleable radio is deselected.
	//
	// Flutter: widgets/radio_group.dart RadioGroupRegistry.onChanged
	OnChanged(value foundation.Opt[T])
}

// RadioClient is a radio button that can register with a RadioGroupRegistry.
//
// Flutter: widgets/radio_group.dart RadioClient
type RadioClient[T comparable] interface {
	// Tristate is whether the radio can be deselected by selecting it again.
	//
	// Flutter: widgets/radio_group.dart RadioClient.tristate
	Tristate() bool
	// RadioValue is the value that the radio stands for.
	//
	// Flutter: widgets/radio_group.dart RadioClient.radioValue
	RadioValue() T
	// Enabled is whether the radio responds to input.
	//
	// Flutter: widgets/radio_group.dart RadioClient.enabled
	Enabled() bool
	// FocusNode is the focus node of the radio.
	//
	// Flutter: widgets/radio_group.dart RadioClient.focusNode
	FocusNode() *FocusNode
}

// RadioClientRegistration keeps the registry that a radio belongs to; embed it
// in the state of the radio.
//
// Flutter: widgets/radio_group.dart RadioClient
type RadioClientRegistration[T comparable] struct {
	registry RadioGroupRegistry[T]
}

// Registry is the registry that the radio belongs to, or nil.
//
// Flutter: widgets/radio_group.dart RadioClient.registry
func (r *RadioClientRegistration[T]) Registry() RadioGroupRegistry[T] { return r.registry }

// SetRegistry moves the radio, which is the client, to the registry (nil for
// none).
//
// Flutter: widgets/radio_group.dart RadioClient.registry
func (r *RadioClientRegistration[T]) SetRegistry(client RadioClient[T], registry RadioGroupRegistry[T]) {
	if r.registry != registry && r.registry != nil {
		r.registry.UnregisterClient(client)
	}
	r.registry = registry
	if registry != nil {
		registry.RegisterClient(client)
	}
}

// RadioGroupOpts are the optional parameters of NewRadioGroup.
//
// Flutter: widgets/radio_group.dart RadioGroup.new
type RadioGroupOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// GroupValue is the value of the currently selected radio; none for none.
	GroupValue foundation.Opt[T]
}

// RadioGroup is a widget that manages a group of radio buttons below it: it holds
// the value of the group, calls OnChanged when the user selects another radio, and
// moves the selection with the arrow keys.
//
// Flutter: widgets/radio_group.dart RadioGroup
type RadioGroup[T comparable] struct {
	StatefulWidget
	o         RadioGroupOpts[T]
	onChanged func(value foundation.Opt[T])
	child     Widget
}

// NewRadioGroup creates the widget.
//
// Flutter: widgets/radio_group.dart RadioGroup.new
func NewRadioGroup[T comparable](onChanged func(value foundation.Opt[T]), child Widget, opts ...RadioGroupOpts[T]) *RadioGroup[T] {
	o := firstOpts(opts)
	return &RadioGroup[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), o: o, onChanged: onChanged, child: child}
}

// GroupValue is the value of the currently selected radio.
//
// Flutter: widgets/radio_group.dart RadioGroup.groupValue
func (w *RadioGroup[T]) GroupValue() foundation.Opt[T] { return w.o.GroupValue }

// OnChanged is called when the user selects a radio.
//
// Flutter: widgets/radio_group.dart RadioGroup.onChanged
func (w *RadioGroup[T]) OnChanged() func(value foundation.Opt[T]) { return w.onChanged }

// Child is the widget below this one.
//
// Flutter: widgets/radio_group.dart RadioGroup.child
func (w *RadioGroup[T]) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/radio_group.dart RadioGroup.createState
func (w *RadioGroup[T]) CreateState() State { return &radioGroupState[T]{} }

// RadioGroupMaybeOf returns the registry of the closest RadioGroup above the
// context that has the value type T, if there is one. The context depends on
// it.
//
// Flutter: widgets/radio_group.dart RadioGroup.maybeOf
func RadioGroupMaybeOf[T comparable](bc BuildContext) (RadioGroupRegistry[T], bool) {
	if s, ok := DependOn[*radioGroupStateScope[T]](bc); ok {
		return s.state, true
	}
	return nil, false
}

type radioGroupState[T comparable] struct {
	StateOf[*RadioGroup[T]]
	manager                            *radioGroupShortcutManager[T]
	radios                             []RadioClient[T]
	debugSingleSelectionCheckScheduled bool
}

func (s *radioGroupState[T]) InitState() {
	s.manager = &radioGroupShortcutManager[T]{state: s}
	InitShortcutManagerBase(&s.manager.ShortcutManagerBase, s.manager, ShortcutManagerOpts{Shortcuts: ShortcutMap{
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: VoidCallbackIntent{Callback: s.selectPreviousRadio}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: VoidCallbackIntent{Callback: s.selectNextRadio}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: VoidCallbackIntent{Callback: s.selectNextRadio}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: VoidCallbackIntent{Callback: s.selectPreviousRadio}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeySpace), Intent: VoidCallbackIntent{Callback: s.toggleFocusedRadio}},
	}})
}

func (s *radioGroupState[T]) Dispose() { s.manager.Dispose() }

// debugScheduleSingleSelectionCheck checks after the frame that no more than one
// radio has the value of the group.
func (s *radioGroupState[T]) debugScheduleSingleSelectionCheck() {
	if !foundation.DebugMode || s.debugSingleSelectionCheckScheduled {
		return
	}
	WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		s.debugSingleSelectionCheckScheduled = false
		if !s.Mounted() || s.debugCheckOnlySingleSelection() {
			return
		}
		panic(errors.New("RadioGroupMultipleSelection",
			"RadioGroupPolicy can't be used for a radio group that allows multiple selection"))
	})
	s.debugSingleSelectionCheckScheduled = true
}

func (s *radioGroupState[T]) debugCheckOnlySingleSelection() bool {
	n := 0
	for _, radio := range s.radios {
		if s.GroupValue() == foundation.Some(radio.RadioValue()) {
			n++
		}
	}
	return n < 2
}

func (s *radioGroupState[T]) GroupValue() foundation.Opt[T] { return s.Widget().o.GroupValue }

func (s *radioGroupState[T]) RegisterClient(radio RadioClient[T]) {
	if !slices.Contains(s.radios, radio) {
		s.radios = append(s.radios, radio)
	}
	s.debugScheduleSingleSelectionCheck()
}

func (s *radioGroupState[T]) UnregisterClient(radio RadioClient[T]) {
	if i := slices.Index(s.radios, radio); i >= 0 {
		s.radios = slices.Delete(s.radios, i, i+1)
	}
}

func (s *radioGroupState[T]) OnChanged(value foundation.Opt[T]) { s.Widget().onChanged(value) }

func (s *radioGroupState[T]) focusedRadio() RadioClient[T] {
	for _, radio := range s.radios {
		if radio.FocusNode().HasFocus() {
			return radio
		}
	}
	return nil
}

func (s *radioGroupState[T]) toggleFocusedRadio() {
	radio := s.focusedRadio()
	if radio == nil {
		return
	}
	if s.Widget().o.GroupValue != foundation.Some(radio.RadioValue()) {
		s.OnChanged(foundation.Some(radio.RadioValue()))
		return
	}
	if radio.Tristate() {
		s.OnChanged(foundation.None[T]())
	}
}

func (s *radioGroupState[T]) selectNextRadio()     { s.selectRadioInDirection(true) }
func (s *radioGroupState[T]) selectPreviousRadio() { s.selectRadioInDirection(false) }

func (s *radioGroupState[T]) selectRadioInDirection(forward bool) {
	if len(s.radios) < 2 {
		return
	}
	current := s.focusedRadio()
	if current == nil {
		return
	}
	currentFocus := current.FocusNode()
	var enabled []*FocusNode
	for _, radio := range s.radios {
		if radio.Enabled() {
			enabled = append(enabled, radio.FocusNode())
		}
	}
	sorted := ReadingOrderSort(enabled)
	if len(sorted) == 0 {
		return
	}
	if !forward {
		sorted = slices.Clone(sorted)
		slices.Reverse(sorted)
	}
	var next *FocusNode
	if i := slices.Index(sorted, currentFocus); i >= 0 && i+1 < len(sorted) {
		next = sorted[i+1]
	}
	if next == nil {
		next = sorted[0]
	}
	for _, radio := range s.radios {
		if radio.FocusNode() == next {
			s.OnChanged(foundation.Some(radio.RadioValue()))
			break
		}
	}
	next.RequestFocus()
}

func (s *radioGroupState[T]) Build(bc BuildContext) Widget {
	s.debugScheduleSingleSelectionCheck()
	w := s.Widget()
	return NewSemantics(
		NewShortcutsManager(s.manager,
			NewFocusTraversalGroup(
				&radioGroupStateScope[T]{InheritedWidget: NewInheritedWidgetBase(w.child), state: s, groupValue: w.o.GroupValue},
				FocusTraversalGroupOpts{Policy: newSkipUnselectedRadioPolicy(&s.radios, w.o.GroupValue)})),
		SemanticsOpts{Container: true, Role: foundation.Some(ui.SemanticsRoleRadioGroup)})
}

// radioGroupShortcutManager is the manager of the shortcuts of a radio group: it
// handles the keys only while a radio of the group has the focus.
//
// Flutter: widgets/radio_group.dart _RadioGroupShortcutManager
type radioGroupShortcutManager[T comparable] struct {
	ShortcutManagerBase
	state *radioGroupState[T]
}

func (m *radioGroupShortcutManager[T]) HandleKeypress(bc BuildContext, event services.KeyEvent) KeyEventResult {
	if m.state.focusedRadio() == nil {
		return KeyEventResultIgnored
	}
	return m.ShortcutManagerBase.HandleKeypress(bc, event)
}

// radioGroupStateScope tells the radios below a group about its registry.
//
// Flutter: widgets/radio_group.dart _RadioGroupStateScope
type radioGroupStateScope[T comparable] struct {
	InheritedWidget
	state      *radioGroupState[T]
	groupValue foundation.Opt[T]
}

func (w *radioGroupStateScope[T]) UpdateShouldNotify(old Widget) bool {
	o := old.(*radioGroupStateScope[T])
	return w.state != o.state || w.groupValue != o.groupValue
}

// skipUnselectedRadioPolicy is the traversal policy of a radio group: the focus
// goes to the selected radio only, as Tab moves through the group.
//
// Flutter: widgets/radio_group.dart _SkipUnselectedRadioPolicy
type skipUnselectedRadioPolicy[T comparable] struct {
	ReadingOrderTraversalPolicy
	radios     *[]RadioClient[T]
	groupValue foundation.Opt[T]
}

func newSkipUnselectedRadioPolicy[T comparable](radios *[]RadioClient[T], groupValue foundation.Opt[T]) *skipUnselectedRadioPolicy[T] {
	p := &skipUnselectedRadioPolicy[T]{radios: radios, groupValue: groupValue}
	InitFocusTraversalPolicyBase(&p.FocusTraversalPolicyBase, p, FocusTraversalPolicyBaseOpts{Directional: true})
	return p
}

func (p *skipUnselectedRadioPolicy[T]) SortDescendants(descendants []*FocusNode, currentNode *FocusNode) []*FocusNode {
	inReadOrder := p.ReadingOrderTraversalPolicy.SortDescendants(descendants, currentNode)
	var selected RadioClient[T]
	for _, radio := range *p.radios {
		if p.groupValue == foundation.Some(radio.RadioValue()) {
			selected = radio
			break
		}
	}
	if selected == nil {
		byNode := map[*FocusNode]RadioClient[T]{}
		for _, radio := range *p.radios {
			byNode[radio.FocusNode()] = radio
		}
		for _, node := range inReadOrder {
			if r, ok := byNode[node]; ok {
				selected = r
				break
			}
		}
	}
	if selected == nil {
		return inReadOrder
	}
	skip := map[*FocusNode]bool{}
	for _, radio := range *p.radios {
		if radio != selected && radio.FocusNode() != currentNode {
			skip[radio.FocusNode()] = true
		}
	}
	var kept []*FocusNode
	for _, node := range descendants {
		if !skip[node] {
			kept = append(kept, node)
		}
	}
	return p.ReadingOrderTraversalPolicy.SortDescendants(kept, currentNode)
}
