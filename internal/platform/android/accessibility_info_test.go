package android

import (
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// update makes a node update with the "unset" values the framework gives, changed by f.
func update(f func(n *ui.SemanticsNodeUpdate)) ui.SemanticsNodeUpdate {
	n := ui.SemanticsNodeUpdate{
		TextSelectionBase: -1, TextSelectionExtent: -1, PlatformViewID: -1, MaxValueLength: -1, CurrentValueLength: -1,
		ScrollPosition: math.NaN(), ScrollExtentMax: math.NaN(), ScrollExtentMin: math.NaN(),
		Transform: identity(), HitTestTransform: identity(),
	}
	f(&n)
	return n
}

func texts(l []styledText) []string {
	var out []string
	for _, s := range l {
		out = append(out, s.Text)
	}
	return out
}

func hasAction(n androidNode, id int) bool {
	for _, a := range n.AndroidActions {
		if a.ID == id {
			return true
		}
	}
	return false
}

func actionLabel(n androidNode, id int) string {
	for _, a := range n.AndroidActions {
		if a.ID == id {
			return a.Label
		}
	}
	return "?"
}

// The decisions of createAccessibilityNodeInfo that need no tree, as the cases of
// AccessibilityBridgeTest and the role configurators have them.
func TestNewAndroidNode(t *testing.T) {
	custom := map[int]ui.SemanticsCustomActionUpdate{
		3: {ID: 3, Label: "Archive", Hint: "archives it", OverrideID: -1},
		4: {ID: 4, Label: "Open", Hint: "open it", OverrideID: int(ui.SemanticsActionTap)},
		5: {ID: 5, Label: "More", Hint: "more of it", OverrideID: int(ui.SemanticsActionLongPress)},
	}
	tests := []struct {
		name  string
		n     ui.SemanticsNodeUpdate
		check func(t *testing.T, n androidNode)
	}{
		{"a label is a content description, not a text", update(func(n *ui.SemanticsNodeUpdate) { n.Label = "Hello, World" }),
			func(t *testing.T, n androidNode) {
				if !reflect.DeepEqual(texts(n.Content), []string{"Hello, World"}) || len(n.Text) != 0 || len(n.Hint) != 1 || n.ClassName != classView {
					t.Errorf("content %v, text %v, hint %v, class %s", n.Content, n.Text, n.Hint, n.ClassName)
				}
				if n.Props&propImportant == 0 || n.Props&propFocusable == 0 || n.Props&propEnabled == 0 {
					t.Errorf("a node with a label is important, focusable and enabled: %b", n.Props)
				}
			}},
		{"value, label and hint are joined in that order", update(func(n *ui.SemanticsNodeUpdate) { n.Value, n.Label, n.Hint = "v", "l", "h" }),
			func(t *testing.T, n androidNode) {
				if !reflect.DeepEqual(texts(n.Content), []string{"v", "l", "h"}) || n.Content[0].Kind != kindValue || n.Content[1].Kind != kindLabel || n.Content[2].Kind != kindHint {
					t.Errorf("content %+v", n.Content)
				}
			}},
		{"a text field has text and a hint", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsTextField = true
			n.Value, n.Label, n.Hint = "Hello, World", "some label", "some hint"
		}), func(t *testing.T, n androidNode) {
			if !reflect.DeepEqual(texts(n.Text), []string{"Hello, World"}) || !reflect.DeepEqual(texts(n.Hint), []string{"some label", "some hint"}) {
				t.Errorf("text %v, hint %v", texts(n.Text), texts(n.Hint))
			}
			if n.ClassName != classEditText || n.Props&propEditable == 0 || n.Props&propTextField == 0 || n.Props&propPassword != 0 {
				t.Errorf("class %s, props %b", n.ClassName, n.Props)
			}
		}},
		{"a read only text field is no EditText", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField, n.Flags.IsReadOnly = true, true }),
			func(t *testing.T, n androidNode) {
				if n.ClassName != classView || n.Props&propEditable != 0 {
					t.Errorf("class %s, props %b", n.ClassName, n.Props)
				}
			}},
		{"an obscured text field is a password", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsTextField, n.Flags.IsObscured = true, true }),
			func(t *testing.T, n androidNode) {
				if n.Props&propPassword == 0 {
					t.Errorf("props %b", n.Props)
				}
			}},
		{"a text field's selection is in UTF-16 units, its granularities and length limit", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsTextField = true
			n.Value = "a\U0001F600b" // 6 bytes, 4 units
			n.TextSelectionBase, n.TextSelectionExtent = 5, 6
			n.MaxValueLength, n.CurrentValueLength = 10, 4
			n.Actions = ui.SemanticsActionMoveCursorForwardByCharacter | ui.SemanticsActionMoveCursorBackwardByWord | ui.SemanticsActionSetSelection |
				ui.SemanticsActionSetText | ui.SemanticsActionCopy | ui.SemanticsActionCut | ui.SemanticsActionPaste
		}), func(t *testing.T, n androidNode) {
			if n.SelectionBase != 3 || n.SelectionExtent != 4 {
				t.Errorf("selection %d, %d", n.SelectionBase, n.SelectionExtent)
			}
			if n.Granularities != granularityCharacter|granularityWord || n.MaxTextLength != 10 {
				t.Errorf("granularities %d, max length %d", n.Granularities, n.MaxTextLength)
			}
			for _, id := range []int{androidActionNextAtMovementGranularity, androidActionPreviousAtMovementGranularity, androidActionSetSelection,
				androidActionSetText, androidActionCopy, androidActionCut, androidActionPaste} {
				if !hasAction(n, id) {
					t.Errorf("action %#x is missing", id)
				}
			}
		}},
		{"a scope of a route has no content description to give but is not important", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.ScopesRoute = true
			n.Label = "Route"
		}), func(t *testing.T, n androidNode) {
			if n.Props&propImportant != 0 || n.Props&propFocusable != 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a node without text or actions is not important", update(func(n *ui.SemanticsNodeUpdate) {}), func(t *testing.T, n androidNode) {
			if n.Props&propImportant != 0 || n.Props&propFocusable != 0 || n.Props&propEnabled == 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"an action alone makes a node important and focusable", update(func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionTap }),
			func(t *testing.T, n androidNode) {
				if n.Props&propImportant == 0 || n.Props&propFocusable == 0 || n.Props&propClickable == 0 || !hasAction(n, androidActionClick) {
					t.Errorf("props %b, actions %v", n.Props, n.AndroidActions)
				}
			}},
		{"scrolling alone is not focusable", update(func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionScrollDown | ui.SemanticsActionScrollUp }),
			func(t *testing.T, n androidNode) {
				if n.Props&propFocusable != 0 || n.Props&propScrollable == 0 {
					t.Errorf("props %b", n.Props)
				}
			}},
		{"a focusable flag makes a node focusable, unless the focus is blocked", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsFocused = ui.TristateIsFalse
			n.Flags.IsAccessibilityFocusBlocked = true
		}), func(t *testing.T, n androidNode) {
			if n.Props&propFocusable == 0 {
				t.Errorf("IS_FOCUSABLE wins over the block: props %b", n.Props)
			}
		}},
		{"a blocked focus is not focusable", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsAccessibilityFocusBlocked = true
			n.Label = "x"
		}), func(t *testing.T, n androidNode) {
			if n.Props&propFocusable != 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a tooltip is its own string and gives no content", update(func(n *ui.SemanticsNodeUpdate) { n.Tooltip = "tooltip" }),
			func(t *testing.T, n androidNode) {
				if n.Tooltip != "tooltip" || len(n.Content) != 0 {
					t.Errorf("tooltip %q, content %v", n.Tooltip, n.Content)
				}
			}},
		{"an identifier", update(func(n *ui.SemanticsNodeUpdate) { n.Identifier = "my-id" }), func(t *testing.T, n androidNode) {
			if n.Identifier != "my-id" {
				t.Errorf("identifier %q", n.Identifier)
			}
		}},
		{"a link with a URL is no button, and the label has the URL", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsLink = true
			n.Label, n.LinkURL = "site", "https://example.com"
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classView || n.Content[0].URL != "https://example.com" {
				t.Errorf("class %s, content %+v", n.ClassName, n.Content)
			}
		}},
		{"a link without a URL is a button", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLink = true }), func(t *testing.T, n androidNode) {
			if n.ClassName != classButton {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a button", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsButton = true }), func(t *testing.T, n androidNode) {
			if n.ClassName != classButton {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"an image", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsImage = true }), func(t *testing.T, n androidNode) {
			if n.ClassName != classImageView {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a slider gets a click action", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSlider = true }), func(t *testing.T, n androidNode) {
			if !hasAction(n, androidActionClick) || n.Props&propClickable == 0 {
				t.Errorf("props %b, actions %v", n.Props, n.AndroidActions)
			}
		}},
		{"increase and decrease make a SeekBar that scrolls", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsSlider = true
			n.Actions = ui.SemanticsActionIncrease | ui.SemanticsActionDecrease
			n.Value, n.IncreasedValue, n.DecreasedValue = "50%", "60%", "40%"
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classSeekBar || !hasAction(n, androidActionScrollForward) || !hasAction(n, androidActionScrollBackward) {
				t.Errorf("class %s, actions %v", n.ClassName, n.AndroidActions)
			}
			if !reflect.DeepEqual(texts(n.Increased), []string{"60%"}) || !reflect.DeepEqual(texts(n.Decreased), []string{"40%"}) {
				t.Errorf("increased %v, decreased %v", n.Increased, n.Decreased)
			}
		}},
		{"scrolling up and left scrolls forward, down and right backward", update(func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionScrollUp | ui.SemanticsActionScrollRight
		}), func(t *testing.T, n androidNode) {
			if !hasAction(n, androidActionScrollForward) || !hasAction(n, androidActionScrollBackward) || n.Props&propScrollable == 0 {
				t.Errorf("props %b, actions %v", n.Props, n.AndroidActions)
			}
		}},
		{"an implicitly scrolling list is a ScrollView", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.HasImplicitScrolling = true
			n.Actions = ui.SemanticsActionScrollDown
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classScrollView {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a horizontal one is a HorizontalScrollView", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.HasImplicitScrolling = true
			n.Actions = ui.SemanticsActionScrollLeft
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classHorizontalScroll {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a list of several children is a collection of rows", update(func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionScrollDown
			n.ScrollChildren = 7
		}), func(t *testing.T, n androidNode) {
			if n.CollectionRows != 7 || n.CollectionColumns != 1 || n.ScrollChildren != 7 {
				t.Errorf("rows %d, columns %d", n.CollectionRows, n.CollectionColumns)
			}
		}},
		{"a horizontal one is a collection of columns", update(func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionScrollLeft
			n.ScrollChildren = 3
		}), func(t *testing.T, n androidNode) {
			if n.CollectionRows != 1 || n.CollectionColumns != 3 {
				t.Errorf("rows %d, columns %d", n.CollectionRows, n.CollectionColumns)
			}
		}},
		{"a list of one child is no collection", update(func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionScrollDown
			n.ScrollChildren = 1
		}), func(t *testing.T, n androidNode) {
			if n.CollectionRows != 0 || n.CollectionColumns != 0 {
				t.Errorf("rows %d, columns %d", n.CollectionRows, n.CollectionColumns)
			}
		}},
		{"a checked check box", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsTrue }), func(t *testing.T, n androidNode) {
			if n.ClassName != classCheckBox || n.Props&propCheckable == 0 || n.Checked != 1 {
				t.Errorf("class %s, props %b, checked %d", n.ClassName, n.Props, n.Checked)
			}
		}},
		{"an unchecked check box", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateIsFalse }), func(t *testing.T, n androidNode) {
			if n.ClassName != classCheckBox || n.Props&propCheckable == 0 || n.Checked != 0 {
				t.Errorf("class %s, props %b, checked %d", n.ClassName, n.Props, n.Checked)
			}
		}},
		{"a mixed check box", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsChecked = ui.CheckedStateMixed }), func(t *testing.T, n androidNode) {
			if n.Checked != 2 || n.Flags&flagIsCheckStateMixed == 0 || n.Flags&flagIsChecked != 0 {
				t.Errorf("checked %d, flags %b", n.Checked, n.Flags)
			}
		}},
		{"a radio button", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsChecked = ui.CheckedStateIsTrue
			n.Flags.IsInMutuallyExclusiveGroup = true
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classRadioButton || n.Checked != 1 {
				t.Errorf("class %s, checked %d", n.ClassName, n.Checked)
			}
		}},
		{"a switch", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsTrue }), func(t *testing.T, n androidNode) {
			if n.ClassName != classSwitch || n.Props&propCheckable == 0 || n.Checked != 1 {
				t.Errorf("class %s, props %b, checked %d", n.ClassName, n.Props, n.Checked)
			}
		}},
		{"a switch that is off", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsToggled = ui.TristateIsFalse }), func(t *testing.T, n androidNode) {
			if n.ClassName != classSwitch || n.Checked != 0 {
				t.Errorf("class %s, checked %d", n.ClassName, n.Checked)
			}
		}},
		{"a selected node", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsSelected = ui.TristateIsTrue }), func(t *testing.T, n androidNode) {
			if n.Props&propSelected == 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a heading has a level", update(func(n *ui.SemanticsNodeUpdate) { n.HeadingLevel = 2 }), func(t *testing.T, n androidNode) {
			if n.Props&propHeading == 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a header flag is no heading", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHeader = true }), func(t *testing.T, n androidNode) {
			if n.Props&propHeading != 0 || n.Flags&flagIsHeader == 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a live region is polite", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsLiveRegion = true }), func(t *testing.T, n androidNode) {
			if n.Props&propLivePolite == 0 {
				t.Errorf("props %b", n.Props)
			}
		}},
		{"a disabled node", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsEnabled = ui.TristateIsFalse }), func(t *testing.T, n androidNode) {
			if n.Props&propEnabled != 0 || n.Flags&flagHasEnabledState == 0 || n.Flags&flagIsEnabled != 0 {
				t.Errorf("props %b, flags %b", n.Props, n.Flags)
			}
		}},
		{"a hidden node keeps its flag", update(func(n *ui.SemanticsNodeUpdate) { n.Flags.IsHidden = true }), func(t *testing.T, n androidNode) {
			if n.Flags&flagIsHidden == 0 {
				t.Errorf("flags %b", n.Flags)
			}
		}},
		{"dismiss", update(func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionDismiss | ui.SemanticsActionLongPress }), func(t *testing.T, n androidNode) {
			if !hasAction(n, androidActionDismiss) || n.Props&propDismissable == 0 || !hasAction(n, androidActionLongClick) || n.Props&propLongClickable == 0 {
				t.Errorf("props %b, actions %v", n.Props, n.AndroidActions)
			}
		}},
		{"an expanded node with the collapse action", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsExpanded = ui.TristateIsTrue
			n.Actions = ui.SemanticsActionCollapse
		}), func(t *testing.T, n androidNode) {
			if n.Expanded != 2 || !hasAction(n, androidActionCollapse) || hasAction(n, androidActionExpand) {
				t.Errorf("expanded %d, actions %v", n.Expanded, n.AndroidActions)
			}
		}},
		{"a collapsed node with the expand action", update(func(n *ui.SemanticsNodeUpdate) {
			n.Flags.IsExpanded = ui.TristateIsFalse
			n.Actions = ui.SemanticsActionExpand
		}), func(t *testing.T, n androidNode) {
			if n.Expanded != 1 || !hasAction(n, androidActionExpand) {
				t.Errorf("expanded %d, actions %v", n.Expanded, n.AndroidActions)
			}
		}},
		{"expand without an expanded state is not offered", update(func(n *ui.SemanticsNodeUpdate) { n.Actions = ui.SemanticsActionExpand }), func(t *testing.T, n androidNode) {
			if n.Expanded != 0 || hasAction(n, androidActionExpand) {
				t.Errorf("expanded %d, actions %v", n.Expanded, n.AndroidActions)
			}
		}},
		{"custom actions have the id plus the first resource id", update(func(n *ui.SemanticsNodeUpdate) { n.AdditionalActions = []int{3, 99} }),
			func(t *testing.T, n androidNode) {
				if len(n.AndroidActions) != 1 || n.AndroidActions[0] != (androidAction{ID: 3 + firstResourceID, Label: "Archive"}) {
					t.Errorf("actions %v (an unknown id is skipped)", n.AndroidActions)
				}
			}},
		{"an action that overrides the tap gives the click action its hint", update(func(n *ui.SemanticsNodeUpdate) {
			n.Actions = ui.SemanticsActionTap | ui.SemanticsActionLongPress
			n.AdditionalActions = []int{4, 5}
		}), func(t *testing.T, n androidNode) {
			if actionLabel(n, androidActionClick) != "open it" || actionLabel(n, androidActionLongClick) != "more of it" || len(n.AndroidActions) != 2 {
				t.Errorf("actions %v", n.AndroidActions)
			}
		}},
		{"a progress bar with a float value", update(func(n *ui.SemanticsNodeUpdate) {
			n.Role = ui.SemanticsRoleProgressBar
			n.Value, n.MinValue, n.MaxValue = "0.5", "0", "1"
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classProgressBar || n.RangeType != rangeFloat || n.RangeMin != 0 || n.RangeMax != 1 || n.RangeCurrent != 0.5 {
				t.Errorf("class %s, range %d %v %v %v", n.ClassName, n.RangeType, n.RangeMin, n.RangeMax, n.RangeCurrent)
			}
		}},
		{"a progress bar without limits", update(func(n *ui.SemanticsNodeUpdate) {
			n.Role = ui.SemanticsRoleProgressBar
			n.Value = "3"
		}), func(t *testing.T, n androidNode) {
			if n.RangeType != rangeFloat || !math.IsInf(float64(n.RangeMin), -1) || !math.IsInf(float64(n.RangeMax), 1) || n.RangeCurrent != 3 {
				t.Errorf("range %d %v %v %v", n.RangeType, n.RangeMin, n.RangeMax, n.RangeCurrent)
			}
		}},
		{"a progress bar with limits that are no numbers", update(func(n *ui.SemanticsNodeUpdate) {
			n.Role = ui.SemanticsRoleProgressBar
			n.Value, n.MinValue, n.MaxValue = "3", "low", "high"
		}), func(t *testing.T, n androidNode) {
			if n.RangeType != rangeFloat || !math.IsInf(float64(n.RangeMin), -1) || !math.IsInf(float64(n.RangeMax), 1) {
				t.Errorf("range %d %v %v", n.RangeType, n.RangeMin, n.RangeMax)
			}
		}},
		{"a progress bar whose value is no number is indeterminate", update(func(n *ui.SemanticsNodeUpdate) {
			n.Role = ui.SemanticsRoleProgressBar
			n.Value = "loading"
		}), func(t *testing.T, n androidNode) {
			if n.RangeType != rangeIndeterminate {
				t.Errorf("range %d", n.RangeType)
			}
		}},
		{"a progress bar without a value has no range", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleProgressBar }), func(t *testing.T, n androidNode) {
			if n.ClassName != classProgressBar || n.RangeType != rangeNone {
				t.Errorf("class %s, range %d", n.ClassName, n.RangeType)
			}
		}},
		{"a combo box can open a popup", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleComboBox }), func(t *testing.T, n androidNode) {
			if n.ClassName != classSpinner || n.Props&propCanOpenPopup == 0 {
				t.Errorf("class %s, props %b", n.ClassName, n.Props)
			}
		}},
		{"a menu can open a popup", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenu }), func(t *testing.T, n androidNode) {
			if n.ClassName != classSpinner || n.Props&propCanOpenPopup == 0 {
				t.Errorf("class %s, props %b", n.ClassName, n.Props)
			}
		}},
		{"menu items", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleMenuItem }), func(t *testing.T, n androidNode) {
			if n.ClassName != classMenuItem {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a menu item check box keeps the class of its role over the check box", update(func(n *ui.SemanticsNodeUpdate) {
			n.Role = ui.SemanticsRoleMenuItemCheckbox
			n.Flags.IsChecked = ui.CheckedStateIsTrue
		}), func(t *testing.T, n androidNode) {
			if n.ClassName != classMenuItem || n.Checked != 1 {
				t.Errorf("class %s, checked %d", n.ClassName, n.Checked)
			}
		}},
		{"a list and a radio group", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleList }), func(t *testing.T, n androidNode) {
			if n.ClassName != classListView {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a radio group", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleRadioGroup }), func(t *testing.T, n androidNode) {
			if n.ClassName != classRadioGroup {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"a role without a class leaves the class alone", update(func(n *ui.SemanticsNodeUpdate) { n.Role = ui.SemanticsRoleAlert }), func(t *testing.T, n androidNode) {
			if n.ClassName != classView {
				t.Errorf("class %s", n.ClassName)
			}
		}},
		{"spell out and locale attributes are in UTF-16 units of their text", update(func(n *ui.SemanticsNodeUpdate) {
			n.Label = "\U0001F600ab"
			n.LabelAttributes = []ui.StringAttribute{
				ui.NewSpellOutStringAttribute(ui.TextRange{Start: 4, End: 6}),
				ui.NewLocaleStringAttribute(ui.TextRange{Start: 0, End: 4}, ui.Locale{LanguageCode: "el", CountryCode: "GR"}),
			}
		}), func(t *testing.T, n androidNode) {
			want := []styledAttr{{Start: 2, End: 4, Type: attrSpellOut}, {Start: 0, End: 2, Type: attrLocale, Locale: "el-GR"}}
			if !reflect.DeepEqual(n.Content[0].Attrs, want) {
				t.Errorf("attrs %+v, want %+v", n.Content[0].Attrs, want)
			}
		}},
		{"the locale of the node is on every text", update(func(n *ui.SemanticsNodeUpdate) {
			n.Value, n.Label = "v", "l"
			l := ui.Locale{LanguageCode: "fr", CountryCode: "FR"}
			n.Locale = &l
		}), func(t *testing.T, n androidNode) {
			if n.Content[0].Locale != "fr-FR" || n.Content[1].Locale != "fr-FR" {
				t.Errorf("content %+v", n.Content)
			}
		}},
		{"the geometry and the children are passed on", update(func(n *ui.SemanticsNodeUpdate) {
			n.Rect = ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}
			n.ChildrenInTraversalOrder = []int{5, 6}
			n.ChildrenInHitTestOrder = []int{6, 5}
			n.ScrollPosition, n.ScrollExtentMax, n.ScrollExtentMin = 10, math.Inf(1), 0
		}), func(t *testing.T, n androidNode) {
			if n.Left != 1 || n.Top != 2 || n.Right != 3 || n.Bottom != 4 || !reflect.DeepEqual(n.TraversalChildren, []int{5, 6}) ||
				!reflect.DeepEqual(n.HitTestChildren, []int{6, 5}) || n.ScrollPosition != 10 || !math.IsInf(float64(n.ScrollExtentMax), 1) {
				t.Errorf("node %+v", n)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, newAndroidNode(tt.n, custom))
		})
	}
}

func TestFlagBits(t *testing.T) {
	tests := []struct {
		name  string
		flags ui.SemanticsFlags
		want  uint32
	}{
		{"none", ui.SemanticsFlags{}, 0},
		{"a button that is enabled", ui.SemanticsFlags{IsButton: true, IsEnabled: ui.TristateIsTrue}, flagIsButton | flagHasEnabledState | flagIsEnabled},
		{"a disabled one", ui.SemanticsFlags{IsButton: true, IsEnabled: ui.TristateIsFalse}, flagIsButton | flagHasEnabledState},
		{"a checked box", ui.SemanticsFlags{IsChecked: ui.CheckedStateIsTrue}, flagHasCheckedState | flagIsChecked},
		{"a mixed box", ui.SemanticsFlags{IsChecked: ui.CheckedStateMixed}, flagHasCheckedState | flagIsCheckStateMixed},
		{"focus", ui.SemanticsFlags{IsFocused: ui.TristateIsTrue}, flagIsFocused | flagIsFocusable},
		{"focusable", ui.SemanticsFlags{IsFocused: ui.TristateIsFalse}, flagIsFocusable},
		{"expanded", ui.SemanticsFlags{IsExpanded: ui.TristateIsTrue}, flagHasExpandedState | flagIsExpanded},
		{"required", ui.SemanticsFlags{IsRequired: ui.TristateIsTrue}, flagHasRequiredState | flagIsRequired},
		{"selected state", ui.SemanticsFlags{IsSelected: ui.TristateIsFalse}, flagHasSelectedState},
		{"blocked", ui.SemanticsFlags{IsAccessibilityFocusBlocked: true}, flagIsAccessibilityFocusBlocked},
		{"toggled", ui.SemanticsFlags{IsToggled: ui.TristateIsTrue}, flagHasToggledState | flagIsToggled},
	}
	for _, tt := range tests {
		if got := flagBits(tt.flags); got != tt.want {
			t.Errorf("%s: flagBits = %#x, want %#x", tt.name, got, tt.want)
		}
	}
	// The values of the bits are Flutter's (AccessibilityBridge.Flag).
	for name, c := range map[string][2]uint32{"HAS_CHECKED_STATE": {flagHasCheckedState, 1 << 0}, "IS_HIDDEN": {flagIsHidden, 1 << 13},
		"IS_FOCUSABLE": {flagIsFocusable, 1 << 21}, "IS_EXPANDED": {flagIsExpanded, 1 << 27}, "IS_ACCESSIBILITY_FOCUS_BLOCKED": {flagIsAccessibilityFocusBlocked, 1 << 31}} {
		if c[0] != c[1] {
			t.Errorf("%s = %#x, want %#x", name, c[0], c[1])
		}
	}
}

func TestUTF16Offsets(t *testing.T) {
	tests := []struct {
		s      string
		offset int
		want   int
	}{
		{"abc", 0, 0}, {"abc", 2, 2}, {"abc", 3, 3}, {"abc", 9, 3}, {"abc", -1, -1},
		{"é", 2, 1}, {"é", 1, 0}, // inside a character
		{"a\U0001F600b", 1, 1}, {"a\U0001F600b", 5, 3}, {"a\U0001F600b", 6, 4},
		{"\xff\xfe", 2, 2}, // bytes that are no UTF-8 count as one unit each
	}
	for _, tt := range tests {
		if got := utf16Offset(tt.s, tt.offset); got != tt.want {
			t.Errorf("utf16Offset(%q, %d) = %d, want %d", tt.s, tt.offset, got, tt.want)
		}
	}
	if utf16Len("a\U0001F600b") != 4 || utf16Len("") != 0 || utf16Len("é") != 1 {
		t.Error("utf16Len")
	}
}
