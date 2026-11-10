package a11y

import (
	"github.com/rm4n0s/glutter/internal/platform/darwin/a11ymodel"
	"github.com/rm4n0s/glutter/ui"
)

// Trait is a UIAccessibilityTrait of the elements. The numbers of UIKit are read from its symbols when the platform starts
// (a11y_ios.go), so these are names, not values.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityTraits
type Trait int

// The traits that the bridge gives.
const (
	// TraitButton is UIAccessibilityTraitButton.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitButton
	TraitButton Trait = 1 << iota
	// TraitLink is UIAccessibilityTraitLink.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitLink
	TraitLink
	// TraitImage is UIAccessibilityTraitImage.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitImage
	TraitImage
	// TraitSelected is UIAccessibilityTraitSelected.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitSelected
	TraitSelected
	// TraitStaticText is UIAccessibilityTraitStaticText.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitStaticText
	TraitStaticText
	// TraitNotEnabled is UIAccessibilityTraitNotEnabled.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitNotEnabled
	TraitNotEnabled
	// TraitUpdatesFrequently is UIAccessibilityTraitUpdatesFrequently.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitUpdatesFrequently
	TraitUpdatesFrequently
	// TraitAdjustable is UIAccessibilityTraitAdjustable.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitAdjustable
	TraitAdjustable
	// TraitHeader is UIAccessibilityTraitHeader.
	//
	// Flutter: SemanticsObject.mm UIAccessibilityTraitHeader
	TraitHeader
	// TraitKeyboardKey is UIAccessibilityTraitKeyboardKey.
	//
	// Flutter: no counterpart
	TraitKeyboardKey
)

// TraitNames are the names of the UIKit symbols that hold the value of each trait.
//
// Flutter: no counterpart
var TraitNames = map[Trait]string{
	TraitButton: "UIAccessibilityTraitButton", TraitLink: "UIAccessibilityTraitLink", TraitImage: "UIAccessibilityTraitImage",
	TraitSelected: "UIAccessibilityTraitSelected", TraitStaticText: "UIAccessibilityTraitStaticText",
	TraitNotEnabled: "UIAccessibilityTraitNotEnabled", TraitUpdatesFrequently: "UIAccessibilityTraitUpdatesFrequently",
	TraitAdjustable: "UIAccessibilityTraitAdjustable", TraitHeader: "UIAccessibilityTraitHeader",
	TraitKeyboardKey: "UIAccessibilityTraitKeyboardKey",
}

// scrollableActions are the actions that scroll; systemActions are the ones that the system gives a node without the app wanting to
// be asked (the focus callbacks).
//
// Flutter: shell/platform/common/... semantics_node.h kScrollableSemanticsActions, kSystemActions
const (
	scrollableActions = ui.SemanticsActionScrollLeft | ui.SemanticsActionScrollRight | ui.SemanticsActionScrollUp | ui.SemanticsActionScrollDown
	systemActions     = ui.SemanticsActionDidGainAccessibilityFocus | ui.SemanticsActionDidLoseAccessibilityFocus
)

// Focusable says whether the node can have the accessibility focus: it is a scrollable that is now hidden but still a target for
// navigation, or it has a label, a value or a hint, or an action that is not a scroll.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm isFocusable
func Focusable(n *a11ymodel.Node) bool {
	return (n.Flags.HasImplicitScrolling && n.Flags.IsHidden) ||
		n.Label != "" || n.Value != "" || n.Hint != "" ||
		n.Actions&^scrollableActions != 0
}

// IsElement says whether the node is an accessibility element: a node that scopes a route is not (its children are), and the others
// are when they are focusable.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm isAccessibilityElement
func IsElement(n *a11ymodel.Node) bool {
	if n.Flags.ScopesRoute {
		return false
	}
	return Focusable(n)
}

// LabelOf is the label that VoiceOver reads: the node's label, with its tooltip under it on a line of its own (or alone).
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityLabel
func LabelOf(n *a11ymodel.Node) string {
	label := n.Label
	if n.Tooltip != "" {
		if label != "" {
			return label + "\n" + n.Tooltip
		}
		return n.Tooltip
	}
	return label
}

// HintOf is the hint that VoiceOver reads after a pause.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityHint
func HintOf(n *a11ymodel.Node) string { return n.Hint }

// ValueOf is the value of the node: its value; or, for a toggle or a check box that is not in a radio group, "1" when it is on and "0"
// when it is off (iOS does not announce the value of a radio button). An empty string is no value.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityValue
func ValueOf(n *a11ymodel.Node) string {
	if n.Value != "" {
		return n.Value
	}
	if n.Flags.IsInMutuallyExclusiveGroup {
		return ""
	}
	switch {
	case n.Flags.IsToggled == ui.TristateIsTrue || n.Flags.IsChecked == ui.CheckedStateIsTrue:
		return "1"
	case n.Flags.IsToggled == ui.TristateIsFalse || n.Flags.IsChecked == ui.CheckedStateIsFalse:
		return "0"
	}
	return ""
}

// TraitsOf are the traits of the node: adjustable for a node that can be increased or decreased, button for a toggle, a check box, a radio
// button and a button, selected, not enabled, header for a heading, image, updates frequently for a live region, link; and static text for
// a leaf that has a label and none of those and is not a text field.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityTraits
func TraitsOf(n *a11ymodel.Node, hasChildren bool) Trait {
	var t Trait
	if n.Actions&(ui.SemanticsActionIncrease|ui.SemanticsActionDecrease) != 0 {
		t |= TraitAdjustable
	}
	if n.Flags.IsToggled != ui.TristateNone || n.Flags.IsChecked != ui.CheckedStateNone {
		t |= TraitButton
	}
	if n.Flags.IsSelected == ui.TristateIsTrue {
		t |= TraitSelected
	}
	if n.Flags.IsButton {
		t |= TraitButton
	}
	if n.Flags.IsEnabled == ui.TristateIsFalse {
		t |= TraitNotEnabled
	}
	if n.HeadingLevel > 0 {
		t |= TraitHeader
	}
	if n.Flags.IsImage {
		t |= TraitImage
	}
	if n.Flags.IsLiveRegion {
		t |= TraitUpdatesFrequently
	}
	if n.Flags.IsLink {
		t |= TraitLink
	}
	if n.Flags.IsKeyboardKey {
		t |= TraitKeyboardKey
	}
	if t == 0 && !hasChildren && LabelOf(n) != "" && !n.Flags.IsTextField {
		t = TraitStaticText
	}
	return t
}

// RespondsToUserInteraction says whether VoiceOver should announce that the element can be used (the "double tap to activate" hint):
// not for a node that blocks the accessibility focus, and only for a node with an action that is not a system one or with custom
// actions.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityRespondsToUserInteraction
func RespondsToUserInteraction(n *a11ymodel.Node) bool {
	if n.Flags.IsAccessibilityFocusBlocked {
		return false
	}
	return n.Actions&^systemActions != 0 || len(n.AdditionalActions) > 0
}

// ScrollDirection is a UIAccessibilityScrollDirection, with UIKit's numbers.
//
// Flutter: no counterpart
type ScrollDirection int

// The scroll directions that VoiceOver sends.
const (
	ScrollRight    ScrollDirection = 1
	ScrollLeft     ScrollDirection = 2
	ScrollUp       ScrollDirection = 3
	ScrollDown     ScrollDirection = 4
	ScrollNext     ScrollDirection = 5
	ScrollPrevious ScrollDirection = 6
)

// ScrollActionOf is the action for a scroll direction. The vertical directions are the opposite of the action's: UIKit says the direction
// that the scroll bar moves in and the action the direction that the finger moves in; the horizontal ones are the same. Next and previous
// are left and right (a right-to-left text is not handled). It is not ok for a direction that is none of them.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm GetSemanticsActionForScrollDirection
func ScrollActionOf(d ScrollDirection) (ui.SemanticsAction, bool) {
	switch d {
	case ScrollRight, ScrollPrevious:
		return ui.SemanticsActionScrollRight, true
	case ScrollLeft, ScrollNext:
		return ui.SemanticsActionScrollLeft, true
	case ScrollUp:
		return ui.SemanticsActionScrollDown, true
	case ScrollDown:
		return ui.SemanticsActionScrollUp, true
	}
	return 0, false
}

// Has says whether the node has the action.
//
// Flutter: no counterpart
func Has(n *a11ymodel.Node, a ui.SemanticsAction) bool { return n.Actions&a != 0 }

// ActivateAction is what an activation (a double tap) does: a tap if the node has one; a slider that has none still answers yes, so
// that VoiceOver does not tap its middle (which would set the value there). ok says that the activation was handled.
//
// Flutter: shell/platform/darwin/ios/framework/Source/SemanticsObject.mm accessibilityActivate
func ActivateAction(n *a11ymodel.Node) (action ui.SemanticsAction, send, ok bool) {
	if Has(n, ui.SemanticsActionTap) {
		return ui.SemanticsActionTap, true, true
	}
	return 0, false, n.Flags.IsSlider
}

// Describe is the a11ymodel.Describer of iOS: the properties whose change is a change of the element. There are no roles: the traits
// stand for them (Flags is watched by the model on its own).
//
// Flutter: no counterpart
func Describe(_ *a11ymodel.Model, n *a11ymodel.Node) a11ymodel.Description {
	d := a11ymodel.Description{
		Label: LabelOf(n), Help: HintOf(n), Value: ValueOf(n),
		Enabled: n.Flags.IsEnabled != ui.TristateIsFalse, Selected: n.Flags.IsSelected == ui.TristateIsTrue,
		Caret: n.TextSelectionExtent, Base: n.TextSelectionBase,
	}
	switch n.Flags.IsExpanded {
	case ui.TristateIsTrue:
		d.Expanded = 2
	case ui.TristateIsFalse:
		d.Expanded = 1
	}
	return d
}
