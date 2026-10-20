// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dartfmt"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// WidgetState is an interactive state that a widget can be in: a button is
// pressed, a text field has an error.
//
// Flutter: widgets/widget_state.dart WidgetState
type WidgetState int

// The widget states, in the order of Flutter's enum. WidgetStateUnspecified is
// the zero value and is not a state.
const (
	// WidgetStateUnspecified is the zero value.
	//
	// Flutter: no counterpart
	WidgetStateUnspecified WidgetState = iota
	// WidgetStateHovered: the state when the user drags their mouse cursor over
	// the given widget.
	//
	// Flutter: widgets/widget_state.dart WidgetState.hovered
	WidgetStateHovered
	// WidgetStateFocused: the state when the user navigates with the keyboard to
	// a given widget.
	//
	// Flutter: widgets/widget_state.dart WidgetState.focused
	WidgetStateFocused
	// WidgetStatePressed: the state when the user is actively pressing down on
	// the given widget.
	//
	// Flutter: widgets/widget_state.dart WidgetState.pressed
	WidgetStatePressed
	// WidgetStateDragged: the state when this widget is being dragged from one
	// place to another by the user.
	//
	// Flutter: widgets/widget_state.dart WidgetState.dragged
	WidgetStateDragged
	// WidgetStateSelected: the state when this item has been selected.
	//
	// Flutter: widgets/widget_state.dart WidgetState.selected
	WidgetStateSelected
	// WidgetStateScrolledUnder: the state when this widget overlaps the content of
	// a scrollable below.
	//
	// Flutter: widgets/widget_state.dart WidgetState.scrolledUnder
	WidgetStateScrolledUnder
	// WidgetStateDisabled: the state when this widget is disabled and cannot be
	// interacted with.
	//
	// Flutter: widgets/widget_state.dart WidgetState.disabled
	WidgetStateDisabled
	// WidgetStateError: the state when the widget has entered some form of
	// invalid state.
	//
	// Flutter: widgets/widget_state.dart WidgetState.error
	WidgetStateError
)

var widgetStateNames = [...]string{
	WidgetStateUnspecified: "unspecified", WidgetStateHovered: "hovered", WidgetStateFocused: "focused",
	WidgetStatePressed: "pressed", WidgetStateDragged: "dragged", WidgetStateSelected: "selected",
	WidgetStateScrolledUnder: "scrolledUnder", WidgetStateDisabled: "disabled", WidgetStateError: "error",
}

// String is the name of the state, as Dart prints it: WidgetState.hovered.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetState) String() string {
	if s < 0 || int(s) >= len(widgetStateNames) {
		return "WidgetState(" + strconv.Itoa(int(s)) + ")"
	}
	return "WidgetState." + widgetStateNames[s]
}

// IsSatisfiedBy reports whether the state is in the set.
//
// Flutter: widgets/widget_state.dart WidgetState.isSatisfiedBy
func (s WidgetState) IsSatisfiedBy(states WidgetStates) bool { return states.Has(s) }

// WidgetStates is a set of widget states (Dart's Set<WidgetState>). It is a
// value that can be compared with ==. The zero value is the empty set.
//
// Flutter: widgets/widget_state.dart WidgetState
type WidgetStates uint16

// NewWidgetStates makes the set of the states.
//
// Flutter: widgets/widget_state.dart WidgetState
func NewWidgetStates(states ...WidgetState) WidgetStates {
	var s WidgetStates
	for _, st := range states {
		s = s.With(st)
	}
	return s
}

// Has reports whether the state is in the set.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) Has(state WidgetState) bool { return s&(1<<uint(state)) != 0 }

// With is the set with the state added.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) With(state WidgetState) WidgetStates { return s | 1<<uint(state) }

// Without is the set with the state removed.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) Without(state WidgetState) WidgetStates { return s &^ (1 << uint(state)) }

// Len is the number of states in the set.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) Len() int {
	n := 0
	for st := WidgetStateHovered; st <= WidgetStateError; st++ {
		if s.Has(st) {
			n++
		}
	}
	return n
}

// Slice is the states in the order of the enum.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) Slice() []WidgetState {
	var out []WidgetState
	for st := WidgetStateHovered; st <= WidgetStateError; st++ {
		if s.Has(st) {
			out = append(out, st)
		}
	}
	return out
}

// String lists the states, as Dart prints a set: {WidgetState.hovered, WidgetState.pressed}.
//
// Flutter: widgets/widget_state.dart WidgetState
func (s WidgetStates) String() string {
	names := make([]string, 0, s.Len())
	for _, st := range s.Slice() {
		names = append(names, st.String())
	}
	return "{" + strings.Join(names, ", ") + "}"
}

// WidgetStatesConstraint is a condition on a set of widget states: a state (it
// must be in the set), WidgetStateAny, or a combination made with
// WidgetStatesAnd, WidgetStatesOr and WidgetStatesNot (Dart's operators &, | and
// ~). Constraints can be compared with ==.
//
// Flutter: widgets/widget_state.dart WidgetStatesConstraint
type WidgetStatesConstraint interface {
	// IsSatisfiedBy reports whether the constraint holds for the set of states.
	//
	// Flutter: widgets/widget_state.dart WidgetStatesConstraint.isSatisfiedBy
	IsSatisfiedBy(states WidgetStates) bool
}

type widgetStateAnd struct{ first, second WidgetStatesConstraint }

func (c widgetStateAnd) IsSatisfiedBy(states WidgetStates) bool {
	return c.first.IsSatisfiedBy(states) && c.second.IsSatisfiedBy(states)
}
func (c widgetStateAnd) String() string {
	return "(" + describeConstraint(c.first) + " & " + describeConstraint(c.second) + ")"
}

type widgetStateOr struct{ first, second WidgetStatesConstraint }

func (c widgetStateOr) IsSatisfiedBy(states WidgetStates) bool {
	return c.first.IsSatisfiedBy(states) || c.second.IsSatisfiedBy(states)
}
func (c widgetStateOr) String() string {
	return "(" + describeConstraint(c.first) + " | " + describeConstraint(c.second) + ")"
}

type widgetStateNot struct{ value WidgetStatesConstraint }

func (c widgetStateNot) IsSatisfiedBy(states WidgetStates) bool {
	return !c.value.IsSatisfiedBy(states)
}
func (c widgetStateNot) String() string { return "~" + describeConstraint(c.value) }

type anyWidgetStates struct{}

func (anyWidgetStates) IsSatisfiedBy(WidgetStates) bool { return true }
func (anyWidgetStates) String() string                  { return "WidgetState.any" }

func describeConstraint(c WidgetStatesConstraint) string {
	if s, ok := c.(interface{ String() string }); ok {
		return s.String()
	}
	return "<constraint>"
}

// WidgetStatesAnd is the constraint that holds when both constraints do (Dart's
// operator &).
//
// Flutter: widgets/widget_state.dart WidgetStatesConstraint.operator &
func WidgetStatesAnd(a, b WidgetStatesConstraint) WidgetStatesConstraint { return widgetStateAnd{a, b} }

// WidgetStatesOr is the constraint that holds when either constraint does (Dart's
// operator |).
//
// Flutter: widgets/widget_state.dart WidgetStatesConstraint.operator |
func WidgetStatesOr(a, b WidgetStatesConstraint) WidgetStatesConstraint { return widgetStateOr{a, b} }

// WidgetStatesNot is the constraint that holds when the constraint does not
// (Dart's operator ~).
//
// Flutter: widgets/widget_state.dart WidgetStatesConstraint.operator ~
func WidgetStatesNot(c WidgetStatesConstraint) WidgetStatesConstraint { return widgetStateNot{c} }

// WidgetStateAny is the constraint that every set of states satisfies, the
// empty set too. It is read-only.
//
// Flutter: widgets/widget_state.dart WidgetState.any
var WidgetStateAny WidgetStatesConstraint = anyWidgetStates{}

// WidgetPropertyResolver is the signature for the functions that resolve a
// property from a set of states.
//
// Flutter: widgets/widget_state.dart WidgetPropertyResolver
type WidgetPropertyResolver[T any] func(states WidgetStates) T

// WidgetStateProperty is an interface for classes that resolve to a value of
// type T based on a widget's interactive state. Widgets whose parameter is
// documented to be a WidgetStateProperty resolve it with the states they are in.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty
type WidgetStateProperty[T any] interface {
	// Resolve is the value for the set of states.
	//
	// Flutter: widgets/widget_state.dart WidgetStateProperty.resolve
	Resolve(states WidgetStates) T
}

// WidgetStateResolveAs resolves the value for the states: when the value is a
// WidgetStateProperty[T] it is resolved, otherwise it must be a T and is returned
// as it is (Dart's WidgetStateProperty.resolveAs, where a Color can also be a
// WidgetStateColor). A value that is neither panics with the tag
// WidgetStateResolveAsType.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.resolveAs
func WidgetStateResolveAs[T any](value any, states WidgetStates) T {
	switch v := value.(type) {
	case WidgetStateProperty[T]:
		return v.Resolve(states)
	case T:
		return v
	case nil:
		var zero T
		return zero
	}
	panic(errors.New("WidgetStateResolveAsType", "the value is neither a WidgetStateProperty nor the type that is resolved",
		"value", reflect.TypeOf(value).String(), "type", reflect.TypeFor[T]().String()))
}

type widgetStatePropertyWith[T any] struct{ resolve WidgetPropertyResolver[T] }

func (p *widgetStatePropertyWith[T]) Resolve(states WidgetStates) T { return p.resolve(states) }

// WidgetStatePropertyResolveWith is a property that resolves with the callback.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.resolveWith
func WidgetStatePropertyResolveWith[T any](callback WidgetPropertyResolver[T]) WidgetStateProperty[T] {
	return &widgetStatePropertyWith[T]{callback}
}

// WidgetStatePropertyAll is a property that resolves to the same value for every
// set of states.
//
// Flutter: widgets/widget_state.dart WidgetStatePropertyAll
type WidgetStatePropertyAll[T any] struct {
	// Value is the value that every set of states resolves to.
	//
	// Flutter: widgets/widget_state.dart WidgetStatePropertyAll.value
	Value T
}

// NewWidgetStatePropertyAll creates a property that always resolves to the value
// (Flutter's WidgetStateProperty.all).
//
// Flutter: widgets/widget_state.dart WidgetStatePropertyAll.new
func NewWidgetStatePropertyAll[T any](value T) *WidgetStatePropertyAll[T] {
	return &WidgetStatePropertyAll[T]{Value: value}
}

// Resolve is the value.
//
// Flutter: widgets/widget_state.dart WidgetStatePropertyAll.resolve
func (p *WidgetStatePropertyAll[T]) Resolve(WidgetStates) T { return p.Value }

// String describes the property: WidgetStatePropertyAll(value), a double with
// the digits that Flutter's debugFormatDouble keeps.
//
// Flutter: widgets/widget_state.dart WidgetStatePropertyAll.toString
func (p *WidgetStatePropertyAll[T]) String() string {
	if f, ok := any(p.Value).(float64); ok {
		return "WidgetStatePropertyAll(" + foundation.DebugFormatDouble(foundation.Some(f)) + ")"
	}
	return "WidgetStatePropertyAll(" + formatValue(p.Value) + ")"
}

// Equal reports whether the other property holds an equal value.
//
// Flutter: widgets/widget_state.dart WidgetStatePropertyAll.==
func (p *WidgetStatePropertyAll[T]) Equal(other *WidgetStatePropertyAll[T]) bool {
	return other != nil && valuesEqual(p.Value, other.Value)
}

// formatValue is how Dart prints the value in string interpolation.
func formatValue(v any) string {
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	if f, ok := v.(float64); ok {
		return dartfmt.Double(f)
	}
	return fmt.Sprint(v)
}

// valuesEqual is Dart's == on two values: the Equal method when there is one, == for
// comparable types, and a deep comparison for the rest.
func valuesEqual(a, b any) bool {
	if e, ok := a.(interface{ Equal(any) bool }); ok {
		return e.Equal(b)
	}
	return reflect.DeepEqual(a, b)
}

// WidgetStateMapEntry is an entry of a WidgetStateMap.
//
// Flutter: widgets/widget_state.dart WidgetStateMap
type WidgetStateMapEntry[T any] struct {
	// Key is the constraint of the entry.
	Key WidgetStatesConstraint
	// Value is the value of the entry.
	Value T
}

// WidgetStateMap is a map of constraints to values, kept in order: the first entry
// whose constraint holds for a set of states is the value of that set (Dart's
// Map<WidgetStatesConstraint, T> keeps the order of insertion).
//
// Flutter: widgets/widget_state.dart WidgetStateMap
type WidgetStateMap[T any] []WidgetStateMapEntry[T]

// WidgetStateMapper is a property that resolves with the first entry of a map
// whose constraint the states satisfy. When none does, the value is the zero
// value of a nullable T (a pointer, an interface, a slice, a map, a function or a
// foundation.Opt), and for any other T it panics with the tag
// WidgetStateMapperNoMatch: add the WidgetStateAny key to the map.
//
// Flutter: widgets/widget_state.dart WidgetStateMapper
type WidgetStateMapper[T any] struct {
	m WidgetStateMap[T]
}

// NewWidgetStateMapper creates a mapper (Flutter's WidgetStateProperty.fromMap).
//
// Flutter: widgets/widget_state.dart WidgetStateMapper.new
func NewWidgetStateMapper[T any](m WidgetStateMap[T]) *WidgetStateMapper[T] {
	return &WidgetStateMapper[T]{m: append(WidgetStateMap[T](nil), m...)}
}

// Resolve is the value of the first entry whose constraint the states satisfy.
//
// Flutter: widgets/widget_state.dart WidgetStateMapper.resolve
func (p *WidgetStateMapper[T]) Resolve(states WidgetStates) T {
	for _, e := range p.m {
		if e.Key.IsSatisfiedBy(states) {
			return e.Value
		}
	}
	var zero T
	if isNullable(reflect.TypeFor[T]()) {
		return zero
	}
	panic(errors.New("WidgetStateMapperNoMatch",
		"The current set of widget states is "+states.String()+".\nNone of the provided map keys matched this set, "+
			"and the type "+reflect.TypeFor[T]().String()+" is non-nullable.\n"+
			"Consider using a nullable type (a pointer or a foundation.Opt), or adding the WidgetStateAny key to this map.",
		"states", states.String(), "type", reflect.TypeFor[T]().String()))
}

func isNullable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return true
	}
	return t.PkgPath() == "github.com/rm4n0s/glutter/foundation" && strings.HasPrefix(t.Name(), "Opt[")
}

// Equal reports whether the other mapper has the same entries, in the same order.
//
// Flutter: widgets/widget_state.dart WidgetStateMapper.==
func (p *WidgetStateMapper[T]) Equal(other *WidgetStateMapper[T]) bool {
	if other == nil || len(p.m) != len(other.m) {
		return false
	}
	for i, e := range p.m {
		o := other.m[i]
		if e.Key != o.Key || !valuesEqual(e.Value, o.Value) {
			return false
		}
	}
	return true
}

// String describes the mapper.
//
// Flutter: widgets/widget_state.dart WidgetStateMapper.toString
func (p *WidgetStateMapper[T]) String() string {
	parts := make([]string, len(p.m))
	for i, e := range p.m {
		parts[i] = describeConstraint(e.Key) + ": " + formatValue(e.Value)
	}
	return "WidgetStateMapper<" + reflect.TypeFor[T]().String() + ">({" + strings.Join(parts, ", ") + "})"
}

// DebugFillProperties adds the map.
//
// Flutter: widgets/widget_state.dart WidgetStateMapper.debugFillProperties
func (p *WidgetStateMapper[T]) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	parts := make([]string, len(p.m))
	for i, e := range p.m {
		parts[i] = describeConstraint(e.Key) + ": " + formatValue(e.Value)
	}
	properties.Add(foundation.NewDiagnosticsProperty[string]("map", "{"+strings.Join(parts, ", ")+"}"))
}

type lerpProperties[T any] struct {
	a, b WidgetStateProperty[T]
	t    float64
	lerp func(a, b foundation.Opt[T], t float64) foundation.Opt[T]
}

func (p *lerpProperties[T]) Resolve(states WidgetStates) foundation.Opt[T] {
	var a, b foundation.Opt[T]
	if p.a != nil {
		a = foundation.Some(p.a.Resolve(states))
	}
	if p.b != nil {
		b = foundation.Some(p.b.Resolve(states))
	}
	return p.lerp(a, b, p.t)
}

// WidgetStatePropertyLerp linearly interpolates between two properties, with the
// lerp function of their values. A nil property resolves to None, which the lerp
// function handles as Dart's null. It is nil when both are nil.
//
// Flutter: widgets/widget_state.dart WidgetStateProperty.lerp
func WidgetStatePropertyLerp[T any](a, b WidgetStateProperty[T], t float64,
	lerp func(a, b foundation.Opt[T], t float64) foundation.Opt[T]) WidgetStateProperty[foundation.Opt[T]] {
	// Avoid creating a lerpProperties value for a common case.
	if a == nil && b == nil {
		return nil
	}
	return &lerpProperties[T]{a, b, t, lerp}
}

// WidgetStateColorResolveWith is a property of colors that resolves with the
// callback. Dart's WidgetStateColor is a Color that is also a property; ui.Color
// is a plain value, so the colors are properties only (SPEC §12.10).
//
// Flutter: widgets/widget_state.dart WidgetStateColor.resolveWith
func WidgetStateColorResolveWith(callback WidgetPropertyResolver[ui.Color]) WidgetStateProperty[ui.Color] {
	return WidgetStatePropertyResolveWith(callback)
}

// WidgetStateColorFromMap is a property of colors that resolves with a map.
//
// Flutter: widgets/widget_state.dart WidgetStateColor.fromMap
func WidgetStateColorFromMap(m WidgetStateMap[ui.Color]) WidgetStateProperty[ui.Color] {
	return NewWidgetStateMapper(m)
}

// WidgetStateColorTransparent is the property that resolves to transparent black
// for every state. It is read-only.
//
// Flutter: widgets/widget_state.dart WidgetStateColor.transparent
var WidgetStateColorTransparent WidgetStateProperty[ui.Color] = NewWidgetStatePropertyAll(ui.NewColor(0x00000000))

// WidgetStateMouseCursor is a mouse cursor that is also a property that resolves
// to a mouse cursor for a set of states. As a cursor, it is the cursor of the
// empty set of states.
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor
type WidgetStateMouseCursor interface {
	services.MouseCursor
	WidgetStateProperty[services.MouseCursor]
}

type widgetStateMouseCursor struct {
	resolve     WidgetPropertyResolver[services.MouseCursor]
	description string
}

func (c *widgetStateMouseCursor) CreateSession(device int) services.MouseCursorSession {
	return c.resolve(0).CreateSession(device)
}
func (c *widgetStateMouseCursor) DebugDescription() string                    { return c.description }
func (c *widgetStateMouseCursor) Resolve(s WidgetStates) services.MouseCursor { return c.resolve(s) }

// WidgetStateMouseCursorResolveWith is a mouse cursor that resolves with the
// callback. The description is the debug description of the cursor; the default
// is WidgetStateMouseCursor().
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor.resolveWith
func WidgetStateMouseCursorResolveWith(callback WidgetPropertyResolver[services.MouseCursor], description ...string) WidgetStateMouseCursor {
	desc := "WidgetStateMouseCursor()"
	if len(description) > 0 {
		desc = description[0]
	}
	return &widgetStateMouseCursor{callback, desc}
}

// WidgetStateMouseCursorFromMap is a mouse cursor that resolves with a map.
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor.fromMap
func WidgetStateMouseCursorFromMap(m WidgetStateMap[services.MouseCursor]) WidgetStateMouseCursor {
	mapper := NewWidgetStateMapper(m)
	return &widgetStateMouseCursor{mapper.Resolve, mapper.String()}
}

// WidgetStateMouseCursorClickable is the click cursor, or the basic one when the
// widget is disabled. It is read-only.
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor.clickable
var WidgetStateMouseCursorClickable = WidgetStateMouseCursorResolveWith(func(states WidgetStates) services.MouseCursor {
	if states.Has(WidgetStateDisabled) {
		return services.SystemMouseCursors.Basic
	}
	return services.SystemMouseCursors.Click
}, "WidgetStateMouseCursor(clickable)")

// WidgetStateMouseCursorAdaptiveClickable is the basic cursor on desktop, as
// Flutter's native platforms use it, and the click cursor on the web, which Glutter
// does not target; it is the basic cursor when the widget is disabled too. It is
// read-only.
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor.adaptiveClickable
var WidgetStateMouseCursorAdaptiveClickable = WidgetStateMouseCursorResolveWith(func(WidgetStates) services.MouseCursor {
	return services.SystemMouseCursors.Basic
}, "WidgetStateMouseCursor(adaptiveClickable)")

// WidgetStateMouseCursorTextable is the text cursor, or the basic one when the
// widget is disabled. It is read-only.
//
// Flutter: widgets/widget_state.dart WidgetStateMouseCursor.textable
var WidgetStateMouseCursorTextable = WidgetStateMouseCursorResolveWith(func(states WidgetStates) services.MouseCursor {
	if states.Has(WidgetStateDisabled) {
		return services.SystemMouseCursors.Basic
	}
	return services.SystemMouseCursors.Text
}, "WidgetStateMouseCursor(textable)")

// WidgetStateBorderSideResolveWith is a property of border sides (None is Dart's
// null) that resolves with the callback.
//
// Flutter: widgets/widget_state.dart WidgetStateBorderSide.resolveWith
func WidgetStateBorderSideResolveWith(callback WidgetPropertyResolver[foundation.Opt[painting.BorderSide]]) WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
	return WidgetStatePropertyResolveWith(callback)
}

// WidgetStateBorderSideFromMap is a property of border sides that resolves with a map.
//
// Flutter: widgets/widget_state.dart WidgetStateBorderSide.fromMap
func WidgetStateBorderSideFromMap(m WidgetStateMap[foundation.Opt[painting.BorderSide]]) WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
	return NewWidgetStateMapper(m)
}

type lerpSides struct {
	a, b WidgetStateProperty[foundation.Opt[painting.BorderSide]]
	t    float64
}

func (p *lerpSides) Resolve(states WidgetStates) foundation.Opt[painting.BorderSide] {
	var resolvedA, resolvedB foundation.Opt[painting.BorderSide]
	if p.a != nil {
		resolvedA = p.a.Resolve(states)
	}
	if p.b != nil {
		resolvedB = p.b.Resolve(states)
	}
	a, aOK := resolvedA.Get()
	b, bOK := resolvedB.Get()
	switch {
	case !aOK && !bOK:
		return foundation.None[painting.BorderSide]()
	case !aOK:
		return foundation.Some(painting.BorderSideLerp(painting.NewBorderSide(painting.BorderSideOpts{
			Width: foundation.Some(0.0), Color: foundation.Some(b.Color.WithAlpha(0))}), b, p.t))
	case !bOK:
		return foundation.Some(painting.BorderSideLerp(a, painting.NewBorderSide(painting.BorderSideOpts{
			Width: foundation.Some(0.0), Color: foundation.Some(a.Color.WithAlpha(0))}), p.t))
	}
	return foundation.Some(painting.BorderSideLerp(a, b, p.t))
}

// WidgetStateBorderSideLerp linearly interpolates between two properties of border
// sides. A side that is None grows from, or shrinks to, a transparent side of
// width 0. It is nil when both properties are nil, and a when they are the same.
//
// Flutter: widgets/widget_state.dart WidgetStateBorderSide.lerp
func WidgetStateBorderSideLerp(a, b WidgetStateProperty[foundation.Opt[painting.BorderSide]], t float64) WidgetStateProperty[foundation.Opt[painting.BorderSide]] {
	// Avoid creating a lerpSides value for a common case.
	if a == nil && b == nil {
		return nil
	}
	if a == b {
		return a
	}
	return &lerpSides{a, b, t}
}

// WidgetStateOutlinedBorderResolveWith is a property of outlined borders (nil is
// Dart's null) that resolves with the callback.
//
// Flutter: widgets/widget_state.dart WidgetStateOutlinedBorder.resolveWith
func WidgetStateOutlinedBorderResolveWith(callback WidgetPropertyResolver[painting.OutlinedBorder]) WidgetStateProperty[painting.OutlinedBorder] {
	return WidgetStatePropertyResolveWith(callback)
}

// WidgetStateOutlinedBorderFromMap is a property of outlined borders that resolves with a map.
//
// Flutter: widgets/widget_state.dart WidgetStateOutlinedBorder.fromMap
func WidgetStateOutlinedBorderFromMap(m WidgetStateMap[painting.OutlinedBorder]) WidgetStateProperty[painting.OutlinedBorder] {
	return NewWidgetStateMapper(m)
}

// WidgetStateTextStyleResolveWith is a property of text styles that resolves with the callback.
//
// Flutter: widgets/widget_state.dart WidgetStateTextStyle.resolveWith
func WidgetStateTextStyleResolveWith(callback WidgetPropertyResolver[*painting.TextStyle]) WidgetStateProperty[*painting.TextStyle] {
	return WidgetStatePropertyResolveWith(callback)
}

// WidgetStateTextStyleFromMap is a property of text styles that resolves with a map.
//
// Flutter: widgets/widget_state.dart WidgetStateTextStyle.fromMap
func WidgetStateTextStyleFromMap(m WidgetStateMap[*painting.TextStyle]) WidgetStateProperty[*painting.TextStyle] {
	return NewWidgetStateMapper(m)
}

// WidgetStatesController manages a set of WidgetState s and notifies listeners
// of changes.
//
// Flutter: widgets/widget_state.dart WidgetStatesController
type WidgetStatesController struct {
	*foundation.ValueNotifier[WidgetStates]
}

// NewWidgetStatesController creates a controller with the states.
//
// Flutter: widgets/widget_state.dart WidgetStatesController.new
func NewWidgetStatesController(states ...WidgetState) *WidgetStatesController {
	return &WidgetStatesController{foundation.NewValueNotifier(NewWidgetStates(states...))}
}

// Update adds the state to the set, or removes it, and notifies the listeners when
// that changes the set.
//
// Flutter: widgets/widget_state.dart WidgetStatesController.update
func (c *WidgetStatesController) Update(state WidgetState, add bool) {
	old := c.Value()
	value := old.Without(state)
	if add {
		value = old.With(state)
	}
	if value != old {
		c.SetValue(value)
	}
}
