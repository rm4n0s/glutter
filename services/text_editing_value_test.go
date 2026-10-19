package services_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// panicTag runs fn and returns the tag of the error it panicked with, or "".
func panicTag(t *testing.T, fn func()) (tag string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic with %T, want an error", r)
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("panic with %T, want *errors.Error", err)
		}
		tag = appErr.Tag
	}()
	fn()
	return ""
}

func TestTextSelectionEquality(t *testing.T) {
	tests := []struct {
		name string
		a, b services.TextSelection
		want bool
	}{
		{"the invalid selection is a singleton",
			services.NewTextSelection(-1, 0, services.TextSelectionOpts{IsDirectional: true}),
			services.NewTextSelection(123, -1, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream}), true},
		{"affinity does not matter when the selection is not collapsed",
			services.NewTextSelection(1, 2),
			services.NewTextSelection(1, 2, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream}), true},
		{"affinity matters when it is collapsed",
			services.NewTextSelectionCollapsed(1),
			services.NewTextSelectionCollapsed(1, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream}), false},
		{"unspecified affinity is downstream",
			services.NewTextSelectionCollapsed(1),
			services.NewTextSelectionCollapsed(1, services.TextSelectionOpts{Affinity: ui.TextAffinityDownstream}), true},
		{"directional matters",
			services.NewTextSelection(1, 2),
			services.NewTextSelection(1, 2, services.TextSelectionOpts{IsDirectional: true}), false},
		{"offsets matter", services.NewTextSelection(1, 2), services.NewTextSelection(1, 3), false},
		{"valid is not invalid", services.NewTextSelection(1, 2), services.NewTextSelection(-1, -1), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal = %v, want %v", got, tc.want)
			}
			if got := tc.b.Equal(tc.a); got != tc.want {
				t.Errorf("reversed Equal = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextSelectionBaseAndExtent(t *testing.T) {
	tests := []struct {
		name         string
		sel          services.TextSelection
		base, extent ui.TextPosition
	}{
		{"forward", services.NewTextSelection(2, 5),
			ui.TextPosition{Offset: 2, Affinity: ui.TextAffinityDownstream}, ui.TextPosition{Offset: 5, Affinity: ui.TextAffinityUpstream}},
		{"backward", services.NewTextSelection(5, 2),
			ui.TextPosition{Offset: 5, Affinity: ui.TextAffinityUpstream}, ui.TextPosition{Offset: 2, Affinity: ui.TextAffinityDownstream}},
		{"collapsed keeps the affinity", services.NewTextSelectionCollapsed(3, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream}),
			ui.TextPosition{Offset: 3, Affinity: ui.TextAffinityUpstream}, ui.TextPosition{Offset: 3, Affinity: ui.TextAffinityUpstream}},
		{"from a position", services.NewTextSelectionFromPosition(ui.TextPosition{Offset: 4, Affinity: ui.TextAffinityUpstream}),
			ui.TextPosition{Offset: 4, Affinity: ui.TextAffinityUpstream}, ui.TextPosition{Offset: 4, Affinity: ui.TextAffinityUpstream}},
		{"invalid keeps the affinity", services.NewTextSelection(-1, 2),
			ui.TextPosition{Offset: -1}, ui.TextPosition{Offset: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sel.Base(); got != tc.base {
				t.Errorf("Base = %v, want %v", got, tc.base)
			}
			if got := tc.sel.Extent(); got != tc.extent {
				t.Errorf("Extent = %v, want %v", got, tc.extent)
			}
		})
	}
}

func TestTextSelectionExpandAndExtend(t *testing.T) {
	pos := func(o int) ui.TextPosition { return ui.NewTextPosition(o) }
	tests := []struct {
		name string
		got  services.TextSelection
		want services.TextSelection
	}{
		{"expand inside", services.NewTextSelection(3, 6).ExpandTo(pos(4)), services.NewTextSelection(3, 6)},
		{"expand before, normalized", services.NewTextSelection(3, 6).ExpandTo(pos(1)), services.NewTextSelection(1, 6)},
		{"expand before, reversed", services.NewTextSelection(6, 3).ExpandTo(pos(1)), services.NewTextSelection(6, 1)},
		{"expand after, normalized", services.NewTextSelection(3, 6).ExpandTo(pos(9)), services.NewTextSelection(3, 9)},
		{"expand after, reversed", services.NewTextSelection(6, 3).ExpandTo(pos(9)), services.NewTextSelection(9, 3)},
		{"expand before, extent at the index", services.NewTextSelection(3, 6).ExpandTo(pos(1), services.TextSelectionExpandOpts{ExtentAtIndex: true}), services.NewTextSelection(6, 1)},
		{"expand after, extent at the index", services.NewTextSelection(6, 3).ExpandTo(pos(9), services.TextSelectionExpandOpts{ExtentAtIndex: true}), services.NewTextSelection(3, 9)},
		{"extend", services.NewTextSelection(3, 6).ExtendTo(pos(9)), services.NewTextSelection(3, 9)},
		{"extend backward", services.NewTextSelection(3, 6).ExtendTo(pos(1)), services.NewTextSelection(3, 1)},
		{"extend to the same extent", services.NewTextSelection(3, 6).ExtendTo(ui.TextPosition{Offset: 6, Affinity: ui.TextAffinityUpstream}), services.NewTextSelection(3, 6)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.got.Equal(tc.want) {
				t.Errorf("selection = %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestTextSelectionCopyWithAndString(t *testing.T) {
	s := services.NewTextSelection(1, 4)
	c := s.CopyWith(services.TextSelectionCopy{ExtentOffset: foundation.Some(9), IsDirectional: foundation.Some(true)})
	if want := services.NewTextSelection(1, 9, services.TextSelectionOpts{IsDirectional: true}); !c.Equal(want) {
		t.Errorf("CopyWith = %v, want %v", c, want)
	}
	if got := s.CopyWith(services.TextSelectionCopy{Affinity: ui.TextAffinityUpstream}); got.Affinity != ui.TextAffinityUpstream {
		t.Errorf("affinity = %v", got.Affinity)
	}
	strs := map[string]services.TextSelection{
		"TextSelection.invalid": services.NewTextSelection(-1, -1),
		"TextSelection.collapsed(offset: 3, affinity: TextAffinity.upstream, isDirectional: false)": services.NewTextSelectionCollapsed(3, services.TextSelectionOpts{Affinity: ui.TextAffinityUpstream}),
		"TextSelection(baseOffset: 1, extentOffset: 4, isDirectional: false)":                       s,
	}
	for want, sel := range strs {
		if got := sel.String(); got != want {
			t.Errorf("String = %q, want %q", got, want)
		}
	}
}

func TestTextEditingValueReplaced(t *testing.T) {
	const testText = "From a false proposition, anything follows."
	value := func(sel services.TextSelection) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: testText, Selection: foundation.Some(sel)})
	}
	expect := func(text string, sel services.TextSelection) services.TextEditingValue {
		return services.NewTextEditingValue(services.TextEditingValueOpts{Text: text, Selection: foundation.Some(sel)})
	}
	rng := func(s, e int) ui.TextRange { return ui.TextRange{Start: s, End: e} }
	forward, reversed := services.NewTextSelection(5, 13), services.NewTextSelection(13, 5)
	tests := []struct {
		name string
		got  services.TextEditingValue
		want services.TextEditingValue
	}{
		{"selection deletion", value(forward).Replaced(forward.Range(), ""),
			expect("From proposition, anything follows.", services.NewTextSelectionCollapsed(5))},
		{"reversed selection deletion", value(reversed).Replaced(reversed.Range(), ""),
			expect("From proposition, anything follows.", services.NewTextSelectionCollapsed(5))},
		{"insert", value(services.NewTextSelectionCollapsed(5)).Replaced(rng(5, 5), "AA"),
			// The caret moves to the end of the text inserted.
			expect("From AAa false proposition, anything follows.", services.NewTextSelectionCollapsed(7))},
		{"replace before selection", value(reversed).Replaced(rng(4, 5), "AA"),
			expect("FromAAa false proposition, anything follows.", services.NewTextSelection(14, 6))},
		{"replace after selection", value(reversed).Replaced(rng(13, 14), "AA"),
			expect("From a false AAroposition, anything follows.", reversed)},
		{"replace inside selection - start boundary", value(reversed).Replaced(rng(5, 6), "AA"),
			expect("From AA false proposition, anything follows.", services.NewTextSelection(14, 5))},
		{"replace inside selection - end boundary", value(reversed).Replaced(rng(12, 13), "AA"),
			expect("From a falseAAproposition, anything follows.", services.NewTextSelection(14, 5))},
		{"delete after selection", value(reversed).Replaced(rng(13, 14), ""),
			expect("From a false roposition, anything follows.", reversed)},
		{"delete inside selection - start boundary", value(reversed).Replaced(rng(5, 6), ""),
			expect("From  false proposition, anything follows.", services.NewTextSelection(12, 5))},
		{"delete inside selection - end boundary", value(reversed).Replaced(rng(12, 13), ""),
			expect("From a falseproposition, anything follows.", services.NewTextSelection(12, 5))},
		{"same length", value(forward).Replaced(rng(0, 4), "Xxxx"),
			expect("Xxxx a false proposition, anything follows.", forward)},
		{"invalid range changes nothing", value(forward).Replaced(ui.TextRangeEmpty, "x"), value(forward)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.got.Equal(tc.want) {
				t.Errorf("value = %v\nwant %v", tc.got, tc.want)
			}
		})
	}
}

func TestTextEditingValueReplacedMovesTheComposingRange(t *testing.T) {
	v := services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: "hello world", Selection: foundation.Some(services.NewTextSelectionCollapsed(11)),
		Composing: foundation.Some(ui.TextRange{Start: 6, End: 11}),
	})
	got := v.Replaced(ui.TextRange{Start: 0, End: 5}, "hi")
	if got.Text != "hi world" || got.Composing != (ui.TextRange{Start: 3, End: 8}) || got.Selection.BaseOffset != 8 {
		t.Errorf("value = %v", got)
	}
}

func TestTextEditingValueReplacedOutsideTheTextPanics(t *testing.T) {
	v := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "abc"})
	for _, r := range []ui.TextRange{{Start: 2, End: 9}, {Start: 3, End: 1}} {
		if tag := panicTag(t, func() { v.Replaced(r, "x") }); tag != "TextEditingValueRangeOutOfBounds" {
			t.Errorf("Replaced(%v) panicked with %q", r, tag)
		}
	}
}

func TestTextEditingValueIsComposingRangeValid(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		composing ui.TextRange
		want      bool
	}{
		{"empty range", "abc", ui.TextRangeEmpty, false},
		{"inside", "abc", ui.TextRange{Start: 1, End: 3}, true},
		{"collapsed inside", "abc", ui.TextRange{Start: 1, End: 1}, true},
		{"past the end", "abc", ui.TextRange{Start: 1, End: 4}, false},
		{"not normalized", "abc", ui.TextRange{Start: 3, End: 1}, false},
		{"empty text", "", ui.TextRange{Start: 0, End: 0}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := services.NewTextEditingValue(services.TextEditingValueOpts{Text: tc.text, Composing: foundation.Some(tc.composing)})
			if got := v.IsComposingRangeValid(); got != tc.want {
				t.Errorf("IsComposingRangeValid = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTextEditingValueDefaultsCopyAndString(t *testing.T) {
	e := services.TextEditingValueEmpty
	if e.Text != "" || e.Selection.IsValid() || e.Composing != ui.TextRangeEmpty {
		t.Errorf("empty = %v", e)
	}
	v := services.NewTextEditingValue(services.TextEditingValueOpts{Text: "abc"})
	c := v.CopyWith(services.TextEditingValueCopy{
		Selection: foundation.Some(services.NewTextSelectionCollapsed(1)), Composing: foundation.Some(ui.TextRange{Start: 0, End: 2}),
	})
	if c.Text != "abc" || c.Selection.BaseOffset != 1 || c.Composing.End != 2 {
		t.Errorf("CopyWith = %v", c)
	}
	want := "TextEditingValue(text: ┤abc├, selection: TextSelection.collapsed(offset: 1, affinity: TextAffinity.downstream, isDirectional: false), composing: TextRange(start: 0, end: 2))"
	if got := c.String(); got != want {
		t.Errorf("String = %q\nwant %q", got, want)
	}
	if c.Equal(v) || !c.Equal(c) {
		t.Error("Equal is wrong")
	}
}
