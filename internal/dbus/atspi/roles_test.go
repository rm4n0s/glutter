package atspi

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestRoleOf(t *testing.T) {
	flags := func(edit func(f *ui.SemanticsFlags)) ui.SemanticsFlags {
		var f ui.SemanticsFlags
		edit(&f)
		return f
	}
	tests := []struct {
		name   string
		role   ui.SemanticsRole
		flags  ui.SemanticsFlags
		label  string
		isRoot bool
		want   Role
	}{
		{"the root is the frame", ui.SemanticsRoleNone, ui.SemanticsFlags{}, "", true, RoleFrame},
		{"the root with a role is still the frame", ui.SemanticsRoleMain, ui.SemanticsFlags{}, "", true, RoleFrame},
		{"a plain node", ui.SemanticsRoleNone, ui.SemanticsFlags{}, "", false, RolePanel},
		{"text is a label", ui.SemanticsRoleNone, ui.SemanticsFlags{}, "Hello", false, RoleLabel},
		{"button", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsButton = true }), "", false, RolePushButton},
		{"check box", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsButton = true; f.IsChecked = ui.CheckedStateIsFalse }), "", false, RoleCheckBox},
		{"check box without the button flag", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsChecked = ui.CheckedStateMixed }), "", false, RoleCheckBox},
		{"radio button", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) {
			f.IsButton = true
			f.IsChecked = ui.CheckedStateIsTrue
			f.IsInMutuallyExclusiveGroup = true
		}), "", false, RoleRadioButton},
		{"toggle button", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsButton = true; f.IsToggled = ui.TristateIsTrue }), "", false, RoleToggleButton},
		{"switch", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsToggled = ui.TristateIsFalse }), "", false, RoleToggleButton},
		{"text field", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsTextField = true }), "", false, RoleEntry},
		{"password field", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsTextField = true; f.IsObscured = true }), "", false, RolePasswordText},
		{"slider", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsSlider = true }), "", false, RoleSlider},
		{"link", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsLink = true }), "", false, RoleLink},
		{"heading", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsHeader = true }), "", false, RoleHeading},
		{"image", ui.SemanticsRoleNone, flags(func(f *ui.SemanticsFlags) { f.IsImage = true }), "", false, RoleImage},
		{"the role beats the flags", ui.SemanticsRoleTab, flags(func(f *ui.SemanticsFlags) { f.IsButton = true }), "", false, RolePageTab},
		{"tab bar", ui.SemanticsRoleTabBar, ui.SemanticsFlags{}, "", false, RolePageTabList},
		{"tab panel", ui.SemanticsRoleTabPanel, ui.SemanticsFlags{}, "", false, RolePanel},
		{"dialog", ui.SemanticsRoleDialog, ui.SemanticsFlags{}, "", false, RoleDialog},
		{"alert dialog", ui.SemanticsRoleAlertDialog, ui.SemanticsFlags{}, "", false, RoleAlert},
		{"table", ui.SemanticsRoleTable, ui.SemanticsFlags{}, "", false, RoleTable},
		{"cell", ui.SemanticsRoleCell, ui.SemanticsFlags{}, "", false, RoleTableCell},
		{"row", ui.SemanticsRoleRow, ui.SemanticsFlags{}, "", false, RoleTableRow},
		{"column header", ui.SemanticsRoleColumnHeader, ui.SemanticsFlags{}, "", false, RoleTableColumnHeader},
		{"drag handle", ui.SemanticsRoleDragHandle, ui.SemanticsFlags{}, "", false, RolePushButton},
		{"spin button", ui.SemanticsRoleSpinButton, ui.SemanticsFlags{}, "", false, RoleSpinButton},
		{"combo box", ui.SemanticsRoleComboBox, ui.SemanticsFlags{}, "", false, RoleComboBox},
		{"menu bar", ui.SemanticsRoleMenuBar, ui.SemanticsFlags{}, "", false, RoleMenuBar},
		{"menu", ui.SemanticsRoleMenu, ui.SemanticsFlags{}, "", false, RoleMenu},
		{"menu item", ui.SemanticsRoleMenuItem, ui.SemanticsFlags{}, "", false, RoleMenuItem},
		{"menu item checkbox", ui.SemanticsRoleMenuItemCheckbox, ui.SemanticsFlags{}, "", false, RoleCheckMenuItem},
		{"menu item radio", ui.SemanticsRoleMenuItemRadio, ui.SemanticsFlags{}, "", false, RoleRadioMenuItem},
		{"alert", ui.SemanticsRoleAlert, ui.SemanticsFlags{}, "", false, RoleNotification},
		{"status", ui.SemanticsRoleStatus, ui.SemanticsFlags{}, "", false, RoleStatusBar},
		{"list", ui.SemanticsRoleList, ui.SemanticsFlags{}, "", false, RoleList},
		{"list item", ui.SemanticsRoleListItem, ui.SemanticsFlags{}, "", false, RoleListItem},
		{"complementary", ui.SemanticsRoleComplementary, ui.SemanticsFlags{}, "", false, RoleLandmark},
		{"content info", ui.SemanticsRoleContentInfo, ui.SemanticsFlags{}, "", false, RoleLandmark},
		{"main", ui.SemanticsRoleMain, ui.SemanticsFlags{}, "", false, RoleLandmark},
		{"navigation", ui.SemanticsRoleNavigation, ui.SemanticsFlags{}, "", false, RoleLandmark},
		{"region", ui.SemanticsRoleRegion, ui.SemanticsFlags{}, "", false, RoleLandmark},
		{"form", ui.SemanticsRoleForm, ui.SemanticsFlags{}, "", false, RoleForm},
		{"loading spinner", ui.SemanticsRoleLoadingSpinner, ui.SemanticsFlags{}, "", false, RoleProgressBar},
		{"progress bar", ui.SemanticsRoleProgressBar, ui.SemanticsFlags{}, "", false, RoleProgressBar},
		{"radio group", ui.SemanticsRoleRadioGroup, ui.SemanticsFlags{}, "", false, RolePanel},
		{"tooltip", ui.SemanticsRoleTooltip, ui.SemanticsFlags{}, "", false, RoleToolTip},
		{"hot key", ui.SemanticsRoleHotKey, ui.SemanticsFlags{}, "", false, RoleAcceleratorLabel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{SemanticsNodeUpdate: ui.SemanticsNodeUpdate{Role: tt.role, Flags: tt.flags, Label: tt.label}}
			if got := roleOf(n, tt.isRoot); got != tt.want {
				t.Errorf("role = %s (%d), want %s (%d)", got.Name(), got, tt.want.Name(), tt.want)
			}
		})
	}
}

// TestEverySemanticsRoleIsMapped makes sure that a role that is added to the framework is
// given a place in the table.
func TestEverySemanticsRoleIsMapped(t *testing.T) {
	for r := ui.SemanticsRoleNone + 1; r.Name() != "" && r <= ui.SemanticsRoleHotKey; r++ {
		if _, ok := semanticsRoles[r]; !ok {
			t.Errorf("%s has no AT-SPI2 role", r)
		}
	}
	for r, name := range roleNames {
		if name == "" || (r != RoleInvalid && r.Name() == "unknown") {
			t.Errorf("role %d has no name", r)
		}
	}
	if Role(9999).Name() != "unknown" {
		t.Error("an unknown role is not unknown")
	}
}

func TestStatesOf(t *testing.T) {
	tests := []struct {
		name   string
		edit   func(n *ui.SemanticsNodeUpdate)
		isRoot bool
		active bool
		want   []State
		not    []State
	}{
		{"a plain node", func(*ui.SemanticsNodeUpdate) {}, false, false,
			[]State{StateEnabled, StateSensitive, StateShowing, StateVisible}, []State{StateFocusable, StateChecked, StateEditable}},
		{"disabled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsEnabled = ui.TristateIsFalse }, false, false,
			[]State{StateShowing}, []State{StateEnabled, StateSensitive}},
		{"enabled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsEnabled = ui.TristateIsTrue }, false, false,
			[]State{StateEnabled, StateSensitive}, nil},
		{"hidden", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHidden = true }, false, false,
			[]State{StateEnabled}, []State{StateShowing, StateVisible}},
		{"focused", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsFocused = ui.TristateIsTrue }, false, false,
			[]State{StateFocusable, StateFocused}, nil},
		{"focusable", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsFocused = ui.TristateIsFalse }, false, false,
			[]State{StateFocusable}, []State{StateFocused}},
		{"can take the focus", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionFocus }, false, false,
			[]State{StateFocusable}, []State{StateFocused}},
		{"checked", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsTrue }, false, false,
			[]State{StateCheckable, StateChecked}, []State{StateIndeterminate}},
		{"not checked", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsFalse }, false, false,
			[]State{StateCheckable}, []State{StateChecked}},
		{"mixed", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateMixed }, false, false,
			[]State{StateCheckable, StateIndeterminate}, []State{StateChecked}},
		{"toggled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsTrue }, false, false,
			[]State{StateCheckable, StateChecked}, nil},
		{"not toggled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsFalse }, false, false,
			[]State{StateCheckable}, []State{StateChecked}},
		{"selected", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsTrue }, false, false,
			[]State{StateSelectable, StateSelected}, nil},
		{"selectable", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsFalse }, false, false,
			[]State{StateSelectable}, []State{StateSelected}},
		{"expanded", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsExpanded = ui.TristateIsTrue }, false, false,
			[]State{StateExpandable, StateExpanded}, []State{StateCollapsed}},
		{"collapsed", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsExpanded = ui.TristateIsFalse }, false, false,
			[]State{StateExpandable, StateCollapsed}, []State{StateExpanded}},
		{"required", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsRequired = ui.TristateIsTrue }, false, false,
			[]State{StateRequired}, nil},
		{"single-line field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, false, false,
			[]State{StateSingleLine, StateEditable, StateSelectableText}, []State{StateMultiLine, StateReadOnly}},
		{"multi-line field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsMultiline = true }, false, false,
			[]State{StateMultiLine, StateEditable}, []State{StateSingleLine}},
		{"read-only field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsReadOnly = true }, false, false,
			[]State{StateReadOnly}, []State{StateEditable}},
		{"modal dialog", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleDialog; n.Flags.ScopesRoute = true }, false, false,
			[]State{StateModal}, nil},
		{"route that is not a dialog", func(n *ui.SemanticsNodeUpdate) { n.Flags.ScopesRoute = true }, false, false,
			nil, []State{StateModal}},
		{"invalid entry", func(n *ui.SemanticsNodeUpdate) { n.ValidationResult = ui.SemanticsValidationResultInvalid }, false, false,
			[]State{StateInvalidEntry}, nil},
		{"tooltip", func(n *ui.SemanticsNodeUpdate) { n.Tooltip = "tip" }, false, false,
			[]State{StateHasTooltip}, nil},
		{"the active frame", func(*ui.SemanticsNodeUpdate) {}, true, true, []State{StateActive}, nil},
		{"the frame of a window that is not active", func(*ui.SemanticsNodeUpdate) {}, true, false, nil, []State{StateActive}},
		{"a node is never active", func(*ui.SemanticsNodeUpdate) {}, false, true, nil, []State{StateActive}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{}
			tt.edit(&n.SemanticsNodeUpdate)
			set := statesOf(n, tt.isRoot, tt.active)
			for _, s := range tt.want {
				if !set.Has(s) {
					t.Errorf("%s is not set", s.Name())
				}
			}
			for _, s := range tt.not {
				if set.Has(s) {
					t.Errorf("%s is set", s.Name())
				}
			}
		})
	}
}

func TestStateSetWords(t *testing.T) {
	set := StateSet(0).With(StateEnabled).With(StateCheckable).With(StateReadOnly)
	words := set.Words()
	if len(words) != 2 || words[0] != 1<<StateEnabled || words[1] != 1<<(StateCheckable-32)|1<<(StateReadOnly-32) {
		t.Errorf("words = %b", words)
	}
	for s := State(1); s <= stateLast; s++ {
		if _, ok := stateNames[s]; ok && stateNames[s] == "" {
			t.Errorf("state %d has an empty name", s)
		}
	}
	if State(2).Name() != "" {
		t.Error("a state that is not mapped has a name")
	}
}

func TestLandmarkNames(t *testing.T) {
	for r, name := range landmarkNames {
		if semanticsRoles[r] != RoleLandmark || name == "" {
			t.Errorf("%s: role %v, name %q", r, semanticsRoles[r], name)
		}
	}
}
