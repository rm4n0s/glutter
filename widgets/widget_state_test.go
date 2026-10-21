package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

var allWidgetStates = []WidgetState{
	WidgetStateHovered, WidgetStateFocused, WidgetStatePressed, WidgetStateDragged,
	WidgetStateSelected, WidgetStateDisabled, WidgetStateError,
}

// Ported from material_state_property_test.dart.
func TestWidgetStatePropertyResolveWith(t *testing.T) {
	value := WidgetStatePropertyResolveWith(func(states WidgetStates) WidgetState { return states.Slice()[0] })
	for _, s := range allWidgetStates {
		if got := value.Resolve(NewWidgetStates(s)); got != s {
			t.Errorf("resolve(%v) = %v", s, got)
		}
	}
}

func TestWidgetStatePropertyAll(t *testing.T) {
	value := NewWidgetStatePropertyAll(123)
	if value.Resolve(0) != 123 {
		t.Error("the empty set")
	}
	for _, s := range allWidgetStates {
		if got := value.Resolve(NewWidgetStates(s)); got != 123 {
			t.Errorf("resolve(%v) = %v", s, got)
		}
	}
	if !value.Equal(NewWidgetStatePropertyAll(123)) || value.Equal(NewWidgetStatePropertyAll(124)) {
		t.Error("equality")
	}
}

func TestWidgetStatePropertyAllToString(t *testing.T) {
	if got := NewWidgetStatePropertyAll(ui.NewColor(0xFFFFFFFF)).String(); got != "WidgetStatePropertyAll("+ui.NewColor(0xFFFFFFFF).String()+")" {
		t.Errorf("color: %q", got)
	}
	if got, want := NewWidgetStatePropertyAll(33+1.0/3).String(), "WidgetStatePropertyAll(33.3)"; got != want {
		t.Errorf("double: %q, want %q", got, want)
	}
}

func TestWidgetStatePropertyLerp(t *testing.T) {
	style := func(size float64) WidgetStateProperty[*painting.TextStyle] {
		return NewWidgetStatePropertyAll(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(size)}))
	}
	lerp := func(a, b foundation.Opt[*painting.TextStyle], t float64) foundation.Opt[*painting.TextStyle] {
		return foundation.Some(painting.TextStyleLerp(a.OrElse(nil), b.OrElse(nil), t))
	}
	for _, tc := range []struct{ t, want float64 }{{0, 14}, {0.5, 17}, {1, 20}} {
		prop := WidgetStatePropertyLerp(style(14), style(20), tc.t, lerp)
		if got := prop.Resolve(0).Value().FontSize.Value(); got != tc.want {
			t.Errorf("lerp at %v = %v, want %v", tc.t, got, tc.want)
		}
	}
	if WidgetStatePropertyLerp[*painting.TextStyle](nil, nil, 0.5, lerp) != nil {
		t.Error("lerp of two nils is not nil")
	}
}

func TestWidgetStateConstraints(t *testing.T) {
	pressed, hovered := WidgetStatePressed, WidgetStateHovered
	tests := []struct {
		name       string
		constraint WidgetStatesConstraint
		states     WidgetStates
		want       bool
	}{
		{"state in set", pressed, NewWidgetStates(pressed, hovered), true},
		{"state not in set", pressed, NewWidgetStates(hovered), false},
		{"any holds for the empty set", WidgetStateAny, 0, true},
		{"and", WidgetStatesAnd(pressed, hovered), NewWidgetStates(pressed, hovered), true},
		{"and fails", WidgetStatesAnd(pressed, hovered), NewWidgetStates(pressed), false},
		{"or", WidgetStatesOr(pressed, hovered), NewWidgetStates(hovered), true},
		{"or fails", WidgetStatesOr(pressed, hovered), 0, false},
		{"not", WidgetStatesNot(pressed), NewWidgetStates(hovered), true},
		{"not fails", WidgetStatesNot(pressed), NewWidgetStates(pressed), false},
		{"nested", WidgetStatesAnd(hovered, WidgetStatesNot(WidgetStateDisabled)), NewWidgetStates(hovered), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.constraint.IsSatisfiedBy(tt.states); got != tt.want {
				t.Errorf("IsSatisfiedBy(%v) = %v, want %v", tt.states, got, tt.want)
			}
		})
	}
	if WidgetStatesAnd(pressed, hovered) != WidgetStatesAnd(pressed, hovered) || WidgetStatesAnd(pressed, hovered) == WidgetStatesOr(pressed, hovered) {
		t.Error("constraints are compared by structure")
	}
}

func TestWidgetStatesSet(t *testing.T) {
	s := NewWidgetStates(WidgetStatePressed, WidgetStateHovered, WidgetStatePressed)
	if s.Len() != 2 || !s.Has(WidgetStateHovered) || s.Has(WidgetStateError) {
		t.Errorf("set = %v", s)
	}
	if got, want := s.String(), "{WidgetState.hovered, WidgetState.pressed}"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if s.Without(WidgetStateHovered).Has(WidgetStateHovered) || NewWidgetStates().String() != "{}" {
		t.Error("without or empty")
	}
}

func TestWidgetStateMapper(t *testing.T) {
	// The first entry that holds wins, in the order of the map.
	m := NewWidgetStateMapper(WidgetStateMap[int]{
		{Key: WidgetStatesAnd(WidgetStateHovered, WidgetStatePressed), Value: 1},
		{Key: WidgetStateHovered, Value: 2},
		{Key: WidgetStateAny, Value: 3},
	})
	for states, want := range map[WidgetStates]int{
		NewWidgetStates(WidgetStateHovered, WidgetStatePressed): 1, NewWidgetStates(WidgetStateHovered): 2, 0: 3,
		NewWidgetStates(WidgetStateError): 3,
	} {
		if got := m.Resolve(states); got != want {
			t.Errorf("resolve(%v) = %d, want %d", states, got, want)
		}
	}
	same := NewWidgetStateMapper(WidgetStateMap[int]{
		{Key: WidgetStatesAnd(WidgetStateHovered, WidgetStatePressed), Value: 1}, {Key: WidgetStateHovered, Value: 2}, {Key: WidgetStateAny, Value: 3},
	})
	other := NewWidgetStateMapper(WidgetStateMap[int]{{Key: WidgetStateAny, Value: 3}})
	if !m.Equal(same) || m.Equal(other) || m.Equal(nil) {
		t.Error("equality of mappers")
	}
	if got := other.String(); got != "WidgetStateMapper<int>({WidgetState.any: 3})" {
		t.Errorf("String = %q", got)
	}
}

func TestWidgetStateMapperWithoutAMatch(t *testing.T) {
	// A nullable type resolves to null.
	nullable := NewWidgetStateMapper(WidgetStateMap[foundation.Opt[int]]{{Key: WidgetStatePressed, Value: foundation.Some(1)}})
	if nullable.Resolve(0).IsSome() {
		t.Error("an Opt resolved to a value without a match")
	}
	pointer := NewWidgetStateMapper(WidgetStateMap[*int]{})
	if pointer.Resolve(0) != nil {
		t.Error("a pointer resolved to a value without a match")
	}
	// Any other type is an error.
	skipUnlessDebug(t)
	defer func() { _ = requireErrorTagFromPanic(t, recover(), "WidgetStateMapperNoMatch") }()
	NewWidgetStateMapper(WidgetStateMap[int]{{Key: WidgetStatePressed, Value: 1}}).Resolve(NewWidgetStates(WidgetStateHovered))
}

func TestWidgetStateResolveAs(t *testing.T) {
	prop := WidgetStatePropertyResolveWith(func(states WidgetStates) ui.Color {
		if states.Has(WidgetStatePressed) {
			return ui.NewColor(0xFF0000FF)
		}
		return ui.NewColor(0xFF00FF00)
	})
	if got := WidgetStateResolveAs[ui.Color](prop, NewWidgetStates(WidgetStatePressed)); got != ui.NewColor(0xFF0000FF) {
		t.Errorf("property = %v", got)
	}
	if got := WidgetStateResolveAs[ui.Color](ui.NewColor(0xFFFF0000), 0); got != ui.NewColor(0xFFFF0000) {
		t.Errorf("value = %v", got)
	}
	if got := WidgetStateResolveAs[*int](nil, 0); got != nil {
		t.Error("nil")
	}
	skipUnlessDebug(t)
	defer func() { _ = requireErrorTagFromPanic(t, recover(), "WidgetStateResolveAsType") }()
	WidgetStateResolveAs[ui.Color]("not a color", 0)
}

func TestWidgetStateColorsAndCursors(t *testing.T) {
	if WidgetStateColorTransparent.Resolve(NewWidgetStates(WidgetStatePressed)) != ui.NewColor(0) {
		t.Error("transparent")
	}
	c := WidgetStateColorFromMap(WidgetStateMap[ui.Color]{{Key: WidgetStateDisabled, Value: ui.NewColor(0xFF111111)}, {Key: WidgetStateAny, Value: ui.NewColor(0xFF222222)}})
	if c.Resolve(NewWidgetStates(WidgetStateDisabled)) != ui.NewColor(0xFF111111) || c.Resolve(0) != ui.NewColor(0xFF222222) {
		t.Error("color map")
	}
	if WidgetStateColorResolveWith(func(WidgetStates) ui.Color { return ui.NewColor(1) }).Resolve(0) != ui.NewColor(1) {
		t.Error("color resolveWith")
	}
	for _, tc := range []struct {
		cursor WidgetStateMouseCursor
		states WidgetStates
		want   services.MouseCursor
	}{
		{WidgetStateMouseCursorClickable, 0, services.SystemMouseCursors.Click},
		{WidgetStateMouseCursorClickable, NewWidgetStates(WidgetStateDisabled), services.SystemMouseCursors.Basic},
		// The pointing hand on the web (this test also runs under Node), the arrow elsewhere and when disabled.
		{WidgetStateMouseCursorAdaptiveClickable, 0, adaptiveClickableEnabled()},
		{WidgetStateMouseCursorAdaptiveClickable, NewWidgetStates(WidgetStateDisabled), services.SystemMouseCursors.Basic},
		{WidgetStateMouseCursorTextable, 0, services.SystemMouseCursors.Text},
		{WidgetStateMouseCursorTextable, NewWidgetStates(WidgetStateDisabled), services.SystemMouseCursors.Basic},
	} {
		if got := tc.cursor.Resolve(tc.states); got != tc.want {
			t.Errorf("%s with %v = %v, want %v", tc.cursor.DebugDescription(), tc.states, got, tc.want)
		}
	}
	// As a cursor, it is the cursor of the empty set.
	if WidgetStateMouseCursorClickable.CreateSession(1).Cursor() != services.SystemMouseCursors.Click {
		t.Error("the session of a state cursor is not the one of its cursor")
	}
	if WidgetStateMouseCursorClickable.DebugDescription() != "WidgetStateMouseCursor(clickable)" {
		t.Error("description")
	}
}

// adaptiveClickableEnabled is the cursor of an enabled adaptive clickable widget: the pointing hand
// on the web and the arrow on the other platforms.
func adaptiveClickableEnabled() services.MouseCursor {
	if foundation.IsWeb {
		return services.SystemMouseCursors.Click
	}
	return services.SystemMouseCursors.Basic
}

func TestWidgetStateBorderSideLerp(t *testing.T) {
	side := func(w float64) foundation.Opt[painting.BorderSide] {
		return foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(w), Color: foundation.Some(ui.NewColor(0xFF000000))}))
	}
	a := WidgetStateBorderSideResolveWith(func(WidgetStates) foundation.Opt[painting.BorderSide] { return side(2) })
	none := WidgetStateBorderSideResolveWith(func(WidgetStates) foundation.Opt[painting.BorderSide] { return foundation.None[painting.BorderSide]() })
	if WidgetStateBorderSideLerp(nil, nil, 0.5) != nil {
		t.Error("two nils")
	}
	if got := WidgetStateBorderSideLerp(a, none, 0.5).Resolve(0).Value(); got.Width != 1 || got.Color.Alpha() != 128 {
		t.Errorf("fading out: width %v alpha %d", got.Width, got.Color.Alpha())
	}
	if got := WidgetStateBorderSideLerp(none, a, 0.5).Resolve(0).Value(); got.Width != 1 {
		t.Errorf("fading in: width %v", got.Width)
	}
	if WidgetStateBorderSideLerp(none, none, 0.5).Resolve(0).IsSome() {
		t.Error("two nones")
	}
	if got := WidgetStateBorderSideLerp(a, a, 0.3); got != a {
		t.Error("the same property")
	}
}

func TestWidgetStatesController(t *testing.T) {
	c := NewWidgetStatesController(WidgetStateHovered)
	calls := 0
	c.AddListener(func() { calls++ })
	c.Update(WidgetStatePressed, true)
	c.Update(WidgetStatePressed, true) // no change
	c.Update(WidgetStateHovered, false)
	c.Update(WidgetStateError, false) // no change
	if calls != 2 || c.Value() != NewWidgetStates(WidgetStatePressed) {
		t.Errorf("calls = %d, value = %v", calls, c.Value())
	}
	c.Dispose()
}

func TestWidgetStateTextStyleAndOutlinedBorder(t *testing.T) {
	style := WidgetStateTextStyleResolveWith(func(states WidgetStates) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(10.0 + float64(states.Len()))})
	})
	if style.Resolve(NewWidgetStates(WidgetStateHovered)).FontSize.Value() != 11 {
		t.Error("text style")
	}
	fromMap := WidgetStateTextStyleFromMap(WidgetStateMap[*painting.TextStyle]{})
	if fromMap.Resolve(0) != nil {
		t.Error("an empty map resolves to nil")
	}
	border := WidgetStateOutlinedBorderFromMap(WidgetStateMap[painting.OutlinedBorder]{{Key: WidgetStateAny, Value: painting.NewRoundedRectangleBorder()}})
	if border.Resolve(0) == nil || WidgetStateOutlinedBorderResolveWith(func(WidgetStates) painting.OutlinedBorder { return nil }).Resolve(0) != nil {
		t.Error("outlined borders")
	}
}
