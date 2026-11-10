package a11y

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/darwin/a11ymodel"
	"github.com/rm4n0s/glutter/ui"
)

// node makes a node with the update that sets it up.
func node(set func(n *ui.SemanticsNodeUpdate)) *a11ymodel.Node {
	n := &a11ymodel.Node{}
	if set != nil {
		set(&n.SemanticsNodeUpdate)
	}
	return n
}

func TestIsElement(t *testing.T) {
	tests := []struct {
		name string
		set  func(n *ui.SemanticsNodeUpdate)
		want bool
	}{
		{"nothing", nil, false},
		{"a label", func(n *ui.SemanticsNodeUpdate) { n.Label = "a" }, true},
		{"a value", func(n *ui.SemanticsNodeUpdate) { n.Value = "1" }, true},
		{"a hint", func(n *ui.SemanticsNodeUpdate) { n.Hint = "h" }, true},
		{"a tap", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionTap }, true},
		{"only scrolling is not enough", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionScrollUp | ui.SemanticsActionScrollDown }, false},
		{"a scroll and a tap", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionScrollUp | ui.SemanticsActionTap }, true},
		{"a hidden scrollable still is", func(n *ui.SemanticsNodeUpdate) { n.Flags.HasImplicitScrolling, n.Flags.IsHidden = true, true }, true},
		{"a scrollable that is not hidden is not", func(n *ui.SemanticsNodeUpdate) { n.Flags.HasImplicitScrolling = true }, false},
		{"a route scope is not, whatever it has", func(n *ui.SemanticsNodeUpdate) { n.Flags.ScopesRoute, n.Label = true, "page" }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsElement(node(tc.set)); got != tc.want {
				t.Errorf("IsElement = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLabelHintAndValue(t *testing.T) {
	tests := []struct {
		name               string
		set                func(n *ui.SemanticsNodeUpdate)
		label, hint, value string
	}{
		{"nothing", nil, "", "", ""},
		{"label hint value", func(n *ui.SemanticsNodeUpdate) { n.Label, n.Hint, n.Value = "L", "H", "V" }, "L", "H", "V"},
		{"a tooltip under the label", func(n *ui.SemanticsNodeUpdate) { n.Label, n.Tooltip = "L", "T" }, "L\nT", "", ""},
		{"a tooltip alone", func(n *ui.SemanticsNodeUpdate) { n.Tooltip = "T" }, "T", "", ""},
		{"a switch that is on", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsTrue }, "", "", "1"},
		{"a switch that is off", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsFalse }, "", "", "0"},
		{"a check box that is checked", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsTrue }, "", "", "1"},
		{"a check box that is not", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsFalse }, "", "", "0"},
		{"the value wins over the state", func(n *ui.SemanticsNodeUpdate) { n.Value, n.Flags.IsToggled = "On", ui.TristateIsTrue }, "", "", "On"},
		{"a radio button has no value", func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked, n.Flags.IsInMutuallyExclusiveGroup = ui.CheckedStateIsTrue, true
		}, "", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := node(tc.set)
			if got := LabelOf(n); got != tc.label {
				t.Errorf("label %q, want %q", got, tc.label)
			}
			if got := HintOf(n); got != tc.hint {
				t.Errorf("hint %q, want %q", got, tc.hint)
			}
			if got := ValueOf(n); got != tc.value {
				t.Errorf("value %q, want %q", got, tc.value)
			}
		})
	}
}

func TestTraitsOf(t *testing.T) {
	tests := []struct {
		name        string
		set         func(n *ui.SemanticsNodeUpdate)
		hasChildren bool
		want        Trait
	}{
		{"nothing", nil, false, 0},
		{"a button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true }, false, TraitButton},
		{"a switch is a button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsFalse }, false, TraitButton},
		{"a check box is a button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsTrue }, false, TraitButton},
		{"a slider is adjustable", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionIncrease }, false, TraitAdjustable},
		{"decrease alone is adjustable too", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionDecrease }, false, TraitAdjustable},
		{"selected", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsTrue }, false, TraitSelected},
		{"not selected has none", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsFalse }, false, 0},
		{"a disabled button", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton, n.Flags.IsEnabled = true, ui.TristateIsFalse }, false, TraitButton | TraitNotEnabled},
		{"a heading", func(n *ui.SemanticsNodeUpdate) { n.HeadingLevel = 2 }, false, TraitHeader},
		{"an image", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsImage = true }, false, TraitImage},
		{"a live region", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLiveRegion = true }, false, TraitUpdatesFrequently},
		{"a link", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLink = true }, false, TraitLink},
		{"a key", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsKeyboardKey = true }, false, TraitKeyboardKey},
		{"a text leaf is static text", func(n *ui.SemanticsNodeUpdate) { n.Label = "Hello" }, false, TraitStaticText},
		{"a text with children is not", func(n *ui.SemanticsNodeUpdate) { n.Label = "Hello" }, true, 0},
		{"a text field is not static text", func(n *ui.SemanticsNodeUpdate) { n.Label, n.Flags.IsTextField = "Name", true }, false, 0},
		{"a tooltip alone is static text", func(n *ui.SemanticsNodeUpdate) { n.Tooltip = "T" }, false, TraitStaticText},
		{"a button with a label is not static text", func(n *ui.SemanticsNodeUpdate) { n.Label, n.Flags.IsButton = "OK", true }, false, TraitButton},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TraitsOf(node(tc.set), tc.hasChildren); got != tc.want {
				t.Errorf("TraitsOf = %b, want %b", got, tc.want)
			}
		})
	}
}

func TestEveryTraitHasASymbol(t *testing.T) {
	for _, tr := range []Trait{TraitButton, TraitLink, TraitImage, TraitSelected, TraitStaticText, TraitNotEnabled, TraitUpdatesFrequently,
		TraitAdjustable, TraitHeader, TraitKeyboardKey} {
		if TraitNames[tr] == "" {
			t.Errorf("trait %b has no symbol name", tr)
		}
	}
	// Every trait is its own bit, so that a set of them can be told apart.
	seen := Trait(0)
	for tr := range TraitNames {
		if seen&tr != 0 {
			t.Errorf("trait %b shares a bit", tr)
		}
		seen |= tr
	}
}

func TestRespondsToUserInteraction(t *testing.T) {
	tests := []struct {
		name string
		set  func(n *ui.SemanticsNodeUpdate)
		want bool
	}{
		{"nothing", nil, false},
		{"a tap", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionTap }, true},
		{"only the focus callbacks", func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionDidGainAccessibilityFocus | ui.SemanticsActionDidLoseAccessibilityFocus
		}, false},
		{"custom actions", func(n *ui.SemanticsNodeUpdate) { n.AdditionalActions = []int{3} }, true},
		{"blocked", func(n *ui.SemanticsNodeUpdate) {
			n.Actions, n.Flags.IsAccessibilityFocusBlocked = ui.SemanticsActionTap, true
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RespondsToUserInteraction(node(tc.set)); got != tc.want {
				t.Errorf("RespondsToUserInteraction = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScrollActionOf(t *testing.T) {
	tests := []struct {
		d    ScrollDirection
		want ui.SemanticsAction
		ok   bool
	}{
		// The vertical directions are reversed, the horizontal ones are not.
		{ScrollUp, ui.SemanticsActionScrollDown, true},
		{ScrollDown, ui.SemanticsActionScrollUp, true},
		{ScrollLeft, ui.SemanticsActionScrollLeft, true},
		{ScrollRight, ui.SemanticsActionScrollRight, true},
		{ScrollNext, ui.SemanticsActionScrollLeft, true},
		{ScrollPrevious, ui.SemanticsActionScrollRight, true},
		{0, 0, false},
		{7, 0, false},
	}
	for _, tc := range tests {
		got, ok := ScrollActionOf(tc.d)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ScrollActionOf(%d) = %v, %v; want %v, %v", tc.d, got, ok, tc.want, tc.ok)
		}
	}
	// UIKit's numbers: right 1, left 2, up 3, down 4, next 5, previous 6.
	if ScrollRight != 1 || ScrollLeft != 2 || ScrollUp != 3 || ScrollDown != 4 || ScrollNext != 5 || ScrollPrevious != 6 {
		t.Error("the scroll directions are not UIKit's numbers")
	}
}

func TestActivateAction(t *testing.T) {
	tests := []struct {
		name     string
		set      func(n *ui.SemanticsNodeUpdate)
		send, ok bool
	}{
		{"a tap", func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionTap }, true, true},
		{"nothing", nil, false, false},
		{"a slider without a tap is handled and does nothing", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true }, false, true},
		{"a slider with a tap sends it", func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider, n.Actions = true, ui.SemanticsActionTap }, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action, send, ok := ActivateAction(node(tc.set))
			if send != tc.send || ok != tc.ok || (send && action != ui.SemanticsActionTap) {
				t.Errorf("ActivateAction = %v, %v, %v; want send %v, ok %v", action, send, ok, tc.send, tc.ok)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	n := node(func(n *ui.SemanticsNodeUpdate) {
		n.Label, n.Hint, n.Flags.IsToggled = "L", "H", ui.TristateIsTrue
		n.Flags.IsExpanded, n.TextSelectionBase, n.TextSelectionExtent = ui.TristateIsFalse, 1, 4
	})
	d := Describe(nil, n)
	if d.Label != "L" || d.Help != "H" || d.Value != "1" || !d.Enabled || d.Selected || d.Expanded != 1 || d.Base != 1 || d.Caret != 4 {
		t.Errorf("Describe = %+v", d)
	}
	if d := Describe(nil, node(func(n *ui.SemanticsNodeUpdate) {
		n.Flags.IsExpanded, n.Flags.IsEnabled = ui.TristateIsTrue, ui.TristateIsFalse
	})); d.Expanded != 2 || d.Enabled {
		t.Errorf("an expanded, disabled node: %+v", d)
	}
}
