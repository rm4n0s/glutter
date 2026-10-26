package uia

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/uia/semtree"
	"github.com/rm4n0s/glutter/ui"
)

// nodeWith makes the model of a root and one child that the edit shapes, and returns the child.
func nodeWith(t *testing.T, edit func(n *ui.SemanticsNodeUpdate)) (*Model, *Node) {
	t.Helper()
	child := semtree.N(1, "", 0, 0, 10, 10)
	edit(&child)
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 100, 100, 1), child))
	return m, m.Node(NodeRef{1, 1})
}

func TestControlTypeTable(t *testing.T) {
	tests := []struct {
		name string
		edit func(n *ui.SemanticsNodeUpdate)
		want ControlType
	}{
		{"a button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true }, ControlTypeButton},
		{"a check box", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true; n.Flags.IsChecked = ui.CheckedStateIsFalse }, ControlTypeCheckBox},
		{"a check box that is not a button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateMixed }, ControlTypeCheckBox},
		{"a radio button", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Flags.IsInMutuallyExclusiveGroup = true
		}, ControlTypeRadioButton},
		{"a toggle button is a button with the Toggle pattern", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true; n.Flags.IsToggled = ui.TristateIsTrue }, ControlTypeButton},
		{"a text field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, ControlTypeEdit},
		{"a password field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsObscured = true }, ControlTypeEdit},
		{"a slider", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true }, ControlTypeSlider},
		{"a link", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLink = true }, ControlTypeHyperlink},
		{"an image", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsImage = true }, ControlTypeImage},
		{"a header", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHeader = true }, ControlTypeText},
		{"a label", func(n *ui.SemanticsNodeUpdate) { n.Label = "Hello" }, ControlTypeText},
		{"a value only", func(n *ui.SemanticsNodeUpdate) { n.Value = "7" }, ControlTypeText},
		{"nothing at all", func(n *ui.SemanticsNodeUpdate) {}, ControlTypeGroup},
		{"a tab", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTab }, ControlTypeTabItem},
		{"a tab bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTabBar }, ControlTypeTab},
		{"a dialog", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleDialog }, ControlTypeWindow},
		{"a table", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTable }, ControlTypeTable},
		{"a cell", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleCell }, ControlTypeDataItem},
		{"a row", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleRow }, ControlTypeDataItem},
		{"a column header", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleColumnHeader }, ControlTypeHeaderItem},
		{"a combo box", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleComboBox }, ControlTypeComboBox},
		{"a menu bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenuBar }, ControlTypeMenuBar},
		{"a menu", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenu }, ControlTypeMenu},
		{"a menu item", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenuItem }, ControlTypeMenuItem},
		{"a list", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleList }, ControlTypeList},
		{"a list item", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleListItem }, ControlTypeListItem},
		{"a progress bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleProgressBar }, ControlTypeProgressBar},
		{"a spin button", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleSpinButton }, ControlTypeSpinner},
		{"a tooltip", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleTooltip }, ControlTypeToolTip},
		{"a status bar", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleStatus }, ControlTypeStatusBar},
		{"a landmark", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMain }, ControlTypeGroup},
		{"the role wins over the flags", func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleList; n.Flags.IsButton = true }, ControlTypeList},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, n := nodeWith(t, tt.edit)
			if got := ControlTypeOf(n); got != tt.want {
				t.Errorf("control type = %d, want %d", got, tt.want)
			}
		})
	}
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10)))
	if got := ControlTypeOf(m.Root(1)); got != ControlTypePane {
		t.Errorf("the root = %d, want a pane", got)
	}
}

func TestNameAndHelpText(t *testing.T) {
	tests := []struct {
		name     string
		edit     func(n *ui.SemanticsNodeUpdate)
		wantName string
		wantHelp string
	}{
		{"the label", func(n *ui.SemanticsNodeUpdate) { n.Label = "L"; n.Value = "V"; n.Hint = "H" }, "L", "H"},
		{"the value when there is no label", func(n *ui.SemanticsNodeUpdate) { n.Value = "V"; n.Hint = "H" }, "V", ""},
		{"the hint when there is nothing else", func(n *ui.SemanticsNodeUpdate) { n.Hint = "H" }, "H", ""},
		{"a text field's text is not its name", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Value = "typed"; n.Hint = "Search" }, "Search", ""},
		{"a slider's value is not its name", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "50%" }, "", ""},
		{"the tooltip is the help text", func(n *ui.SemanticsNodeUpdate) { n.Label = "L"; n.Tooltip = "T"; n.Hint = "H" }, "L", "T"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n := nodeWith(t, tt.edit)
			if got := m.Name(n); got != tt.wantName {
				t.Errorf("name = %q, want %q", got, tt.wantName)
			}
			if got := HelpText(n); got != tt.wantHelp {
				t.Errorf("help = %q, want %q", got, tt.wantHelp)
			}
		})
	}
}

func TestAutomationID(t *testing.T) {
	_, n := nodeWith(t, func(n *ui.SemanticsNodeUpdate) {})
	if got := AutomationID(n); got != "1" {
		t.Errorf("id = %q, want the node's number", got)
	}
	_, n = nodeWith(t, func(n *ui.SemanticsNodeUpdate) { n.Identifier = "save-button" })
	if got := AutomationID(n); got != "save-button" {
		t.Errorf("id = %q, want the identifier the app gave", got)
	}
}

func TestPropertyValues(t *testing.T) {
	tests := []struct {
		name string
		edit func(n *ui.SemanticsNodeUpdate)
		prop PropertyID
		want Value
	}{
		{"enabled by default", func(n *ui.SemanticsNodeUpdate) {}, PropIsEnabled, Bool(true)},
		{"disabled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsEnabled = ui.TristateIsFalse }, PropIsEnabled, Bool(false)},
		{"focusable by the flag", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsFocused = ui.TristateIsFalse }, PropIsKeyboardFocusable, Bool(true)},
		{"focusable by the action", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionFocus }, PropIsKeyboardFocusable, Bool(true)},
		{"not focusable", func(n *ui.SemanticsNodeUpdate) {}, PropIsKeyboardFocusable, Bool(false)},
		{"offscreen when hidden", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHidden = true }, PropIsOffscreen, Bool(true)},
		{"a password", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsObscured = true }, PropIsPassword, Bool(true)},
		{"a live region", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLiveRegion = true }, PropLiveSetting, Int(LivePolite)},
		{"not a live region", func(n *ui.SemanticsNodeUpdate) {}, PropLiveSetting, Int(LiveOff)},
		{"the framework", func(n *ui.SemanticsNodeUpdate) {}, PropFrameworkId, String("Glutter")},
		{"the class", func(n *ui.SemanticsNodeUpdate) {}, PropClassName, String("GlutterSemanticsNode")},
		{"a heading level", func(n *ui.SemanticsNodeUpdate) { n.HeadingLevel = 2 }, PropHeadingLevel, Int(2)},
		{"checked", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Actions = ui.SemanticsActionTap
		}, PropToggleToggleState, Int(ToggleOn)},
		{"unchecked", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsFalse }, PropToggleToggleState, Int(ToggleOff)},
		{"mixed", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateMixed }, PropToggleToggleState, Int(ToggleIndeterminate)},
		{"toggled", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsTrue }, PropToggleToggleState, Int(ToggleOn)},
		{"no toggle state", func(n *ui.SemanticsNodeUpdate) {}, PropToggleToggleState, Value{}},
		{"a radio button has no toggle state", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Flags.IsInMutuallyExclusiveGroup = true
		}, PropToggleToggleState, Value{}},
		{"a radio button is selected when checked", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Flags.IsInMutuallyExclusiveGroup = true
		}, PropSelectionItemIsSelected, Bool(true)},
		{"selected", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsTrue }, PropSelectionItemIsSelected, Bool(true)},
		{"expanded", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsExpanded = ui.TristateIsTrue }, PropExpandCollapseExpandCollapseState, Int(ExpandCollapseExpanded)},
		{"collapsed", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsExpanded = ui.TristateIsFalse }, PropExpandCollapseExpandCollapseState, Int(ExpandCollapseCollapsed)},
		{"collapsed by the action", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionExpand }, PropExpandCollapseExpandCollapseState, Int(ExpandCollapseCollapsed)},
		{"a text field's value", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Value = "héllo" }, PropValueValue, String("héllo")},
		{"a password field says nothing", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsTextField = true
			n.Flags.IsObscured = true
			n.Value = "secret"
		}, PropValueValue, String("")},
		{"a read-only field", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true; n.Flags.IsReadOnly = true }, PropValueIsReadOnly, Bool(true)},
		{"a field you can edit", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, PropValueIsReadOnly, Bool(false)},
		{"no value pattern", func(n *ui.SemanticsNodeUpdate) { n.Value = "x" }, PropValueValue, Value{}},
		{"a slider's value in percent", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "37.5%" }, PropRangeValueValue, Float(37.5)},
		{"a slider's minimum", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsSlider = true
			n.Value = "5"
			n.MinValue = "-10"
			n.MaxValue = "10"
		}, PropRangeValueMinimum, Float(-10)},
		{"a slider's maximum", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsSlider = true
			n.Value = "5"
			n.MinValue = "-10"
			n.MaxValue = "10"
		}, PropRangeValueMaximum, Float(10)},
		{"a percent slider goes from 0 to 100", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "50%"; n.MaxValue = "7" }, PropRangeValueMaximum, Float(100)},
		{"the small change", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "50%" }, PropRangeValueSmallChange, Float(1)},
		{"a small range has a small step", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "0.5"; n.MaxValue = "1" }, PropRangeValueSmallChange, Float(0.01)},
		{"the large change", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "50%" }, PropRangeValueLargeChange, Float(10)},
		{"a slider you cannot move", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true; n.Value = "50%" }, PropRangeValueIsReadOnly, Bool(true)},
		{"a slider you can move", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsSlider = true
			n.Value = "50%"
			n.Actions = ui.SemanticsActionIncrease
		}, PropRangeValueIsReadOnly, Bool(false)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n := nodeWith(t, tt.edit)
			got := m.Property(n, tt.prop)
			if got.Kind == ValueFloat && tt.want.Kind == ValueFloat {
				if math.Abs(got.F-tt.want.F) > 1e-9 {
					t.Errorf("value = %v, want %v", got.F, tt.want.F)
				}
				return
			}
			if got != tt.want {
				t.Errorf("value = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestScrollProperties(t *testing.T) {
	tests := []struct {
		name         string
		actions      ui.SemanticsAction
		pos, lo, hi  float64
		wantV, wantH float64
		wantVSize    float64
	}{
		{"half way down a vertical list", ui.SemanticsActionScrollUp | ui.SemanticsActionScrollDown, 50, 0, 100, 50, NoScroll, 50},
		{"at the top", ui.SemanticsActionScrollUp, 0, 0, 100, 0, NoScroll, 50},
		{"a horizontal list", ui.SemanticsActionScrollLeft, 25, 0, 100, NoScroll, 25, 100},
		{"extents not known", ui.SemanticsActionScrollUp, math.NaN(), math.NaN(), math.NaN(), 0, NoScroll, 100},
		{"nothing to scroll", ui.SemanticsActionScrollUp, 0, 0, 0, 0, NoScroll, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n := nodeWith(t, func(n *ui.SemanticsNodeUpdate) {
				n.Rect = ui.NewRectFromLTWH(0, 0, 100, 100)
				n.Actions = tt.actions
				n.ScrollPosition, n.ScrollExtentMin, n.ScrollExtentMax = tt.pos, tt.lo, tt.hi
			})
			if got := m.Property(n, PropScrollVerticalScrollPercent); got.F != tt.wantV {
				t.Errorf("vertical percent = %v, want %v", got.F, tt.wantV)
			}
			if got := m.Property(n, PropScrollHorizontalScrollPercent); got.F != tt.wantH {
				t.Errorf("horizontal percent = %v, want %v", got.F, tt.wantH)
			}
			if tt.wantV != NoScroll {
				if got := m.Property(n, PropScrollVerticalViewSize); math.Abs(got.F-tt.wantVSize) > 1e-9 {
					t.Errorf("vertical view size = %v, want %v", got.F, tt.wantVSize)
				}
			}
		})
	}
}

func TestControlAndContentElements(t *testing.T) {
	tests := []struct {
		name        string
		edit        func(n *ui.SemanticsNodeUpdate)
		control     bool
		content     bool
		controlRoot bool
	}{
		{"a plain container is neither", func(n *ui.SemanticsNodeUpdate) {}, false, false, false},
		{"a scrolling container with nothing else", func(n *ui.SemanticsNodeUpdate) { n.Flags.HasImplicitScrolling = true }, false, false, false},
		{"a label is both", func(n *ui.SemanticsNodeUpdate) { n.Label = "x" }, true, true, false},
		{"a button is a control", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true; n.Actions = ui.SemanticsActionTap }, true, false, false},
		{"a text field is both", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField = true }, true, true, false},
		{"an action alone makes a control", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionTap }, true, false, false},
		{"show on screen alone does not", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionShowOnScreen }, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n := nodeWith(t, tt.edit)
			if got := m.Property(n, PropIsControlElement).B; got != tt.control {
				t.Errorf("control = %v, want %v", got, tt.control)
			}
			if got := m.Property(n, PropIsContentElement).B; got != tt.content {
				t.Errorf("content = %v, want %v", got, tt.content)
			}
		})
	}
	m := NewModel()
	m.Apply(1, semtree.Update(semtree.N(0, "", 0, 0, 10, 10)))
	if !m.Property(m.Root(1), PropIsControlElement).B {
		t.Error("the root is not a control")
	}
}
