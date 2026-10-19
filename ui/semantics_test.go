package ui_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestTristate(t *testing.T) {
	tests := []struct {
		t    ui.Tristate
		name string
		want *bool
	}{
		{ui.TristateNone, "none", nil},
		{ui.TristateIsTrue, "isTrue", ptr(true)},
		{ui.TristateIsFalse, "isFalse", ptr(false)},
	}
	for _, tc := range tests {
		if tc.t.Name() != tc.name || tc.t.String() != "Tristate."+tc.name {
			t.Errorf("%v: name %q", int(tc.t), tc.t.Name())
		}
		got := tc.t.ToBoolOrNull()
		if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
			t.Errorf("%s: ToBoolOrNull = %v", tc.name, got)
		}
	}
	if ui.TristateFromBool(true) != ui.TristateIsTrue || ui.TristateFromBool(false) != ui.TristateIsFalse {
		t.Error("TristateFromBool")
	}
}

func ptr[T any](v T) *T { return &v }

func TestSemanticsFlagsMergeAndConflicts(t *testing.T) {
	a := ui.SemanticsFlags{IsButton: true, IsSelected: ui.TristateIsFalse, IsChecked: ui.CheckedStateIsFalse}
	b := ui.SemanticsFlags{IsHeader: true, IsSelected: ui.TristateIsTrue, IsChecked: ui.CheckedStateMixed}
	m := a.Merge(b)
	if !m.IsButton || !m.IsHeader || m.IsSelected != ui.TristateIsTrue || m.IsChecked != ui.CheckedStateMixed {
		t.Errorf("merged = %+v", m)
	}
	if !a.HasConflictingFlags(b) {
		t.Error("two selected states should conflict")
	}
	if (ui.SemanticsFlags{IsButton: true}).HasConflictingFlags(ui.SemanticsFlags{IsHeader: true}) {
		t.Error("different flags conflict")
	}
	if got := ui.SemanticsFlagsNone.ToStrings(); len(got) != 0 {
		t.Errorf("none = %v", got)
	}
	want := []string{"hasCheckedState", "isChecked", "isSelected", "isButton", "hasSelectedState"}
	got := ui.SemanticsFlags{IsChecked: ui.CheckedStateIsTrue, IsSelected: ui.TristateIsTrue, IsButton: true}.ToStrings()
	if !slices.Equal(got, want) {
		t.Errorf("strings = %v, want %v", got, want)
	}
}

func TestSemanticsActionValuesAndNames(t *testing.T) {
	values := ui.SemanticsActionValues()
	if len(values) == 0 || values[0] != ui.SemanticsActionTap {
		t.Fatalf("values = %v", values)
	}
	for _, a := range values {
		if a.Name() == "" {
			t.Errorf("action %d has no name", int(a))
		}
	}
	if ui.SemanticsActionTap.Name() != "tap" || ui.SemanticsActionScrollLeft.Name() != "scrollLeft" {
		t.Error("names")
	}
}

func TestSemanticsUpdateBuilder(t *testing.T) {
	b := ui.NewSemanticsUpdateBuilder()
	b.UpdateNode(ui.SemanticsNodeUpdate{ID: 1, Label: "a"})
	b.UpdateCustomAction(ui.SemanticsCustomActionUpdate{ID: 3, Label: "x"})
	u := b.Build()
	if len(u.Nodes) != 1 || u.Nodes[0].Label != "a" || len(u.CustomActions) != 1 {
		t.Errorf("update = %+v", u)
	}
	if again := b.Build(); len(again.Nodes) != 0 {
		t.Error("the builder keeps the nodes after Build")
	}
}
