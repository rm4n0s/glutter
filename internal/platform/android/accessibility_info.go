package android

import (
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/rm4n0s/glutter/ui"
)

// The bits of androidNode.Flags: Flutter's SemanticsFlag values (AccessibilityBridge.Flag).
const (
	flagHasCheckedState = 1 << iota
	flagIsChecked
	flagIsSelected
	flagIsButton
	flagIsTextField
	flagIsFocused
	flagHasEnabledState
	flagIsEnabled
	flagIsInMutuallyExclusiveGroup
	flagIsHeader
	flagIsObscured
	flagScopesRoute
	flagNamesRoute
	flagIsHidden
	flagIsImage
	flagIsLiveRegion
	flagHasToggledState
	flagIsToggled
	flagHasImplicitScrolling
	flagIsMultiline
	flagIsReadOnly
	flagIsFocusable
	flagIsLink
	flagIsSlider
	flagIsKeyboardKey
	flagIsCheckStateMixed
	flagHasExpandedState
	flagIsExpanded
	flagHasSelectedState
	flagHasRequiredState
	flagIsRequired
	flagIsAccessibilityFocusBlocked
)

// focusableFlags are the flags that make a node accessibility focusable
// (AccessibilityBridge.FOCUSABLE_FLAGS).
const focusableFlags = flagHasCheckedState | flagIsChecked | flagIsSelected | flagIsTextField | flagIsFocused |
	flagHasEnabledState | flagIsEnabled | flagIsInMutuallyExclusiveGroup | flagHasToggledState | flagIsToggled |
	flagIsFocusable | flagIsSlider

// scrollableActions are the actions that do not make a node focusable on their own
// (AccessibilityBridge.SCROLLABLE_ACTIONS).
const scrollableActions = ui.SemanticsActionScrollRight | ui.SemanticsActionScrollLeft | ui.SemanticsActionScrollUp |
	ui.SemanticsActionScrollDown

// The ids of the AccessibilityNodeInfo.ACTION_* constants that the nodes use.
const (
	androidActionClick                         = 0x10
	androidActionLongClick                     = 0x20
	androidActionNextAtMovementGranularity     = 0x100
	androidActionPreviousAtMovementGranularity = 0x200
	androidActionScrollForward                 = 0x1000
	androidActionScrollBackward                = 0x2000
	androidActionCopy                          = 0x4000
	androidActionPaste                         = 0x8000
	androidActionCut                           = 0x10000
	androidActionSetSelection                  = 0x20000
	androidActionExpand                        = 0x40000
	androidActionCollapse                      = 0x80000
	androidActionDismiss                       = 0x100000
	androidActionSetText                       = 0x200000
)

// firstResourceID is the id of the AccessibilityAction of the custom action number 0
// (AccessibilityBridge.FIRST_RESOURCE_ID).
const firstResourceID = 267386881

// The granularities of AccessibilityNodeInfo.MOVEMENT_GRANULARITY_*.
const (
	granularityCharacter = 1
	granularityWord      = 2
)

// The class names of the views that nodes pretend to be.
const (
	classView             = "android.view.View"
	classEditText         = "android.widget.EditText"
	classButton           = "android.widget.Button"
	classImageView        = "android.widget.ImageView"
	classSeekBar          = "android.widget.SeekBar"
	classScrollView       = "android.widget.ScrollView"
	classHorizontalScroll = "android.widget.HorizontalScrollView"
	classRadioButton      = "android.widget.RadioButton"
	classCheckBox         = "android.widget.CheckBox"
	classSwitch           = "android.widget.Switch"
	classProgressBar      = "android.widget.ProgressBar"
	classSpinner          = "android.widget.Spinner"
	classListView         = "android.widget.ListView"
	classRadioGroup       = "android.widget.RadioGroup"
	classMenuItem         = "android.view.MenuItem"
)

// flagBits turns the flags of the framework into Flutter's bit field (AccessibilityBridge.Flag).
func flagBits(f ui.SemanticsFlags) uint32 {
	var bits uint32
	set := func(on bool, bit uint32) {
		if on {
			bits |= bit
		}
	}
	set(f.IsChecked != ui.CheckedStateNone, flagHasCheckedState)
	set(f.IsChecked == ui.CheckedStateIsTrue, flagIsChecked)
	set(f.IsSelected == ui.TristateIsTrue, flagIsSelected)
	set(f.IsButton, flagIsButton)
	set(f.IsTextField, flagIsTextField)
	set(f.IsFocused == ui.TristateIsTrue, flagIsFocused)
	set(f.IsEnabled != ui.TristateNone, flagHasEnabledState)
	set(f.IsEnabled == ui.TristateIsTrue, flagIsEnabled)
	set(f.IsInMutuallyExclusiveGroup, flagIsInMutuallyExclusiveGroup)
	set(f.IsHeader, flagIsHeader)
	set(f.IsObscured, flagIsObscured)
	set(f.ScopesRoute, flagScopesRoute)
	set(f.NamesRoute, flagNamesRoute)
	set(f.IsHidden, flagIsHidden)
	set(f.IsImage, flagIsImage)
	set(f.IsLiveRegion, flagIsLiveRegion)
	set(f.IsToggled != ui.TristateNone, flagHasToggledState)
	set(f.IsToggled == ui.TristateIsTrue, flagIsToggled)
	set(f.HasImplicitScrolling, flagHasImplicitScrolling)
	set(f.IsMultiline, flagIsMultiline)
	set(f.IsReadOnly, flagIsReadOnly)
	set(f.IsFocused != ui.TristateNone, flagIsFocusable)
	set(f.IsLink, flagIsLink)
	set(f.IsSlider, flagIsSlider)
	set(f.IsKeyboardKey, flagIsKeyboardKey)
	set(f.IsChecked == ui.CheckedStateMixed, flagIsCheckStateMixed)
	set(f.IsExpanded != ui.TristateNone, flagHasExpandedState)
	set(f.IsExpanded == ui.TristateIsTrue, flagIsExpanded)
	set(f.IsSelected != ui.TristateNone, flagHasSelectedState)
	set(f.IsRequired != ui.TristateNone, flagHasRequiredState)
	set(f.IsRequired == ui.TristateIsTrue, flagIsRequired)
	set(f.IsAccessibilityFocusBlocked, flagIsAccessibilityFocusBlocked)
	return bits
}

// utf16Len is the length of s in UTF-16 code units, which is what Java's String.length is. A byte
// that is not valid UTF-8 counts as one unit (U+FFFD).
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// utf16Offset turns a byte offset into s (the framework's text offsets, SPEC §12.10) into the
// UTF-16 offset of the same place. A negative offset stays -1; one past the end is the end.
func utf16Offset(s string, byteOffset int) int {
	if byteOffset < 0 {
		return -1
	}
	if byteOffset > len(s) {
		byteOffset = len(s)
	}
	n := 0
	for i := 0; i < byteOffset; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if i+size > byteOffset {
			break
		}
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
		i += size
	}
	return n
}

// styled makes the text of a node's string with the spans of AccessibilityStringBuilder: the
// attributes of the framework (their ranges are byte offsets), the URL over the whole text, and the
// locale of the node over the whole text. An empty string is no string (Flutter's engine sends -1
// for it), so it gives no text.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityStringBuilder.java AccessibilityStringBuilder.build
func styled(kind uint8, s string, attrs []ui.StringAttribute, url, locale string) (styledText, bool) {
	if s == "" {
		return styledText{}, false
	}
	t := styledText{Kind: kind, Text: s, URL: url, Locale: locale}
	for _, a := range attrs {
		r := a.Range()
		sa := styledAttr{Start: utf16Offset(s, r.Start), End: utf16Offset(s, r.End)}
		switch v := a.(type) {
		case ui.SpellOutStringAttribute:
			sa.Type = attrSpellOut
		case ui.LocaleStringAttribute:
			sa.Type = attrLocale
			sa.Locale = v.Locale().LanguageTag()
		default:
			continue
		}
		if sa.Start < 0 || sa.End < sa.Start {
			continue
		}
		t.Attrs = append(t.Attrs, sa)
	}
	return t, true
}

// joinStyled gathers the parts that exist, in order, as the pieces that the Java side joins with
// ", " (AccessibilityBridge.SemanticsNode.getValueLabelHint).
func joinStyled(parts ...styledPart) []styledText {
	var out []styledText
	for _, p := range parts {
		if p.ok {
			out = append(out, p.text)
		}
	}
	return out
}

// styledPart is a styledText that may not exist.
type styledPart struct {
	text styledText
	ok   bool
}

func part(kind uint8, s string, attrs []ui.StringAttribute, url, locale string) styledPart {
	t, ok := styled(kind, s, attrs, url, locale)
	return styledPart{t, ok}
}

// isFocusable says whether the node can have the accessibility focus, as the framework does not
// say yet (AccessibilityBridge.SemanticsNode.isFocusable).
func isFocusable(flags uint32, actions ui.SemanticsAction, label, value, hint string) bool {
	if flags&flagScopesRoute != 0 {
		return false
	}
	if flags&flagIsFocusable != 0 {
		return true
	}
	if flags&flagIsAccessibilityFocusBlocked != 0 {
		return false
	}
	return actions&^scrollableActions != 0 || flags&focusableFlags != 0 || label != "" || value != "" || hint != ""
}

// rangeOf works out the RangeInfo of a progress bar from its value and its limits: a number as
// the value gives a float range (the limits default to minus and plus infinity), anything else the
// indeterminate one (ProgressBarRoleConfigurator).
func rangeOf(value, minValue, maxValue string) (kind uint8, min, max, current float32) {
	if value == "" {
		return rangeNone, 0, 0, 0
	}
	min, max = float32(math.Inf(-1)), float32(math.Inf(1))
	if f, err := strconv.ParseFloat(minValue, 32); err == nil {
		min = float32(f)
	}
	if f, err := strconv.ParseFloat(maxValue, 32); err == nil {
		max = float32(f)
	}
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return rangeIndeterminate, 0, 0, 0
	}
	return rangeFloat, min, max, float32(f)
}

// roleClass is the class name that a role gives its node, and whether the node can open a popup
// (RoleConfiguratorFactory). A role without one leaves the class name as it was.
func roleClass(role ui.SemanticsRole) (class string, popup bool) {
	switch role {
	case ui.SemanticsRoleProgressBar:
		return classProgressBar, false
	case ui.SemanticsRoleComboBox, ui.SemanticsRoleMenu:
		return classSpinner, true
	case ui.SemanticsRoleList:
		return classListView, false
	case ui.SemanticsRoleRadioGroup:
		return classRadioGroup, false
	case ui.SemanticsRoleMenuItem, ui.SemanticsRoleMenuItemCheckbox, ui.SemanticsRoleMenuItemRadio:
		return classMenuItem, false
	}
	return "", false
}

// newAndroidNode works out everything that the Java side needs of a node of the semantics update:
// the port of the createAccessibilityNodeInfo of Flutter's AccessibilityBridge and of the
// configurators of BaseRoleConfigurator and its subclasses, in their order. What depends on the
// state of the tree or of the focus (the collection info of a list that holds the accessibility
// focus, the item info, the focus action, whether a text field is a live region) is left to the
// Java side. custom holds the custom actions that the framework defined.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.createAccessibilityNodeInfo
func newAndroidNode(n ui.SemanticsNodeUpdate, custom map[int]ui.SemanticsCustomActionUpdate) androidNode {
	flags := flagBits(n.Flags)
	actions := n.Actions
	has := func(a ui.SemanticsAction) bool { return actions&a != 0 }
	hasFlag := func(f uint32) bool { return flags&f != 0 }

	locale := ""
	if n.Locale != nil {
		locale = n.Locale.LanguageTag()
	}
	// The label has the URL of a link as a span; the value and the hint do not.
	value := part(kindValue, n.Value, n.ValueAttributes, "", locale)
	label := part(kindLabel, n.Label, n.LabelAttributes, n.LinkURL, locale)
	hint := part(kindHint, n.Hint, n.HintAttributes, "", locale)

	out := androidNode{
		ID: n.ID, Flags: flags, FlutterActions: uint32(actions),
		ClassName: classView, Identifier: n.Identifier, Value: n.Value, Label: n.Label, Tooltip: n.Tooltip,
		MaxTextLength: -1, ScrollChildren: n.ScrollChildren, ScrollIndex: n.ScrollIndex,
		SelectionBase: -1, SelectionExtent: -1,
		ScrollPosition: float32(n.ScrollPosition), ScrollExtentMax: float32(n.ScrollExtentMax), ScrollExtentMin: float32(n.ScrollExtentMin),
		Left: float32(n.Rect.Left), Top: float32(n.Rect.Top), Right: float32(n.Rect.Right), Bottom: float32(n.Rect.Bottom),
		Transform: n.Transform, HitTestTransform: n.HitTestTransform,
		Text: joinStyled(value), Content: joinStyled(value, label, hint), Hint: joinStyled(label, hint),
		TraversalChildren: n.ChildrenInTraversalOrder, HitTestChildren: n.ChildrenInHitTestOrder,
	}
	if has(ui.SemanticsActionIncrease) {
		out.Increased = joinStyled(part(kindValue, n.IncreasedValue, n.IncreasedValueAttributes, "", locale))
	}
	if has(ui.SemanticsActionDecrease) {
		out.Decreased = joinStyled(part(kindValue, n.DecreasedValue, n.DecreasedValueAttributes, "", locale))
	}

	var props uint32
	set := func(on bool, bit uint32) {
		if on {
			props |= bit
		}
	}
	var acts []androidAction
	add := func(id int) {
		for _, a := range acts {
			if a.ID == id {
				return
			}
		}
		acts = append(acts, androidAction{ID: id})
	}
	addLabelled := func(id int, label string) {
		for i, a := range acts {
			if a.ID == id {
				acts[i].Label = label
				return
			}
		}
		acts = append(acts, androidAction{ID: id, Label: label})
	}

	// isImportant: a node with something to say, or with an action of the user. Flutter's
	// systemAction is the AND of three different bits, which is 0, so every action counts.
	set(!hasFlag(flagScopesRoute) && (len(out.Content) > 0 || actions != 0), propImportant)
	set(isFocusable(flags, actions, n.Label, n.Value, n.Hint), propFocusable)
	set(n.Flags.IsEnabled != ui.TristateIsFalse, propEnabled)

	// configureClipboard
	if has(ui.SemanticsActionSetSelection) {
		add(androidActionSetSelection)
	}
	if has(ui.SemanticsActionCopy) {
		add(androidActionCopy)
	}
	if has(ui.SemanticsActionCut) {
		add(androidActionCut)
	}
	if has(ui.SemanticsActionPaste) {
		add(androidActionPaste)
	}
	if has(ui.SemanticsActionSetText) {
		add(androidActionSetText)
	}
	// configureDismissable
	if has(ui.SemanticsActionDismiss) {
		props |= propDismissable
		add(androidActionDismiss)
	}
	// The custom actions: the ones that override the tap and the long press only give the hint of
	// the standard action (configureTappable); the others are actions of their own
	// (configureCustomActions).
	tapHint, longHint := "", ""
	tapOverride, longOverride := false, false
	var customActs []androidAction
	for _, id := range n.AdditionalActions {
		def, ok := custom[id]
		if !ok {
			continue
		}
		switch def.OverrideID {
		case int(ui.SemanticsActionTap):
			tapOverride, tapHint = true, def.Hint
		case int(ui.SemanticsActionLongPress):
			longOverride, longHint = true, def.Hint
		default:
			customActs = append(customActs, androidAction{ID: id + firstResourceID, Label: def.Label})
		}
	}
	// configureTappable
	if has(ui.SemanticsActionTap) {
		if tapOverride {
			addLabelled(androidActionClick, tapHint)
		} else {
			add(androidActionClick)
		}
		props |= propClickable
	}
	if has(ui.SemanticsActionLongPress) {
		if longOverride {
			addLabelled(androidActionLongClick, longHint)
		} else {
			add(androidActionLongClick)
		}
		props |= propLongClickable
	}
	// configureLiveRegion, configureSelectable, configureHeading
	set(hasFlag(flagIsLiveRegion), propLivePolite)
	set(hasFlag(flagIsSelected), propSelected)
	set(n.HeadingLevel > 0, propHeading)
	for _, a := range customActs {
		addLabelled(a.ID, a.Label)
	}

	// configureTextField
	if hasFlag(flagIsTextField) {
		props |= propTextField
		set(hasFlag(flagIsObscured), propPassword)
		if !hasFlag(flagIsReadOnly) {
			out.ClassName = classEditText
			props |= propEditable
		}
		out.SelectionBase = utf16Offset(n.Value, n.TextSelectionBase)
		out.SelectionExtent = utf16Offset(n.Value, n.TextSelectionExtent)
		var gran uint8
		if has(ui.SemanticsActionMoveCursorForwardByCharacter) {
			add(androidActionNextAtMovementGranularity)
			gran |= granularityCharacter
		}
		if has(ui.SemanticsActionMoveCursorBackwardByCharacter) {
			add(androidActionPreviousAtMovementGranularity)
			gran |= granularityCharacter
		}
		if has(ui.SemanticsActionMoveCursorForwardByWord) {
			add(androidActionNextAtMovementGranularity)
			gran |= granularityWord
		}
		if has(ui.SemanticsActionMoveCursorBackwardByWord) {
			add(androidActionPreviousAtMovementGranularity)
			gran |= granularityWord
		}
		out.Granularities = gran
		if n.MaxValueLength >= 0 {
			out.MaxTextLength = utf16Len(n.Value) - n.CurrentValueLength + n.MaxValueLength
		}
	}
	// configureButton: a link is a button when it has no URL (the URL is a span of the label).
	if hasFlag(flagIsButton) || (n.LinkURL == "" && hasFlag(flagIsLink)) {
		out.ClassName = classButton
	}
	// configureImage
	if hasFlag(flagIsImage) {
		out.ClassName = classImageView
	}
	// configureSlider
	if !has(ui.SemanticsActionTap) && hasFlag(flagIsSlider) {
		add(androidActionClick)
		props |= propClickable
	}
	if has(ui.SemanticsActionIncrease) || has(ui.SemanticsActionDecrease) {
		out.ClassName = classSeekBar
		if has(ui.SemanticsActionIncrease) {
			add(androidActionScrollForward)
		}
		if has(ui.SemanticsActionDecrease) {
			add(androidActionScrollBackward)
		}
	}
	// configureScrollable
	horizontal := has(ui.SemanticsActionScrollLeft) || has(ui.SemanticsActionScrollRight)
	if horizontal || has(ui.SemanticsActionScrollUp) || has(ui.SemanticsActionScrollDown) {
		props |= propScrollable
		if hasFlag(flagHasImplicitScrolling) {
			if horizontal {
				out.ClassName = classHorizontalScroll
			} else {
				out.ClassName = classScrollView
			}
		}
	}
	if has(ui.SemanticsActionScrollLeft) || has(ui.SemanticsActionScrollUp) {
		add(androidActionScrollForward)
	}
	if has(ui.SemanticsActionScrollRight) || has(ui.SemanticsActionScrollDown) {
		add(androidActionScrollBackward)
	}
	// configureCollection: the Java side decides whether the list holds the focus; the shape is
	// worked out here (a list with a single item is no collection).
	if n.ScrollChildren > 1 {
		if horizontal {
			out.CollectionRows, out.CollectionColumns = 1, n.ScrollChildren
		} else {
			out.CollectionRows, out.CollectionColumns = n.ScrollChildren, 1
		}
	}
	// configureCheckable
	hasChecked, hasToggled := hasFlag(flagHasCheckedState), hasFlag(flagHasToggledState)
	set(hasChecked || hasToggled, propCheckable)
	switch {
	case hasChecked:
		if hasFlag(flagIsInMutuallyExclusiveGroup) {
			out.ClassName = classRadioButton
		} else {
			out.ClassName = classCheckBox
		}
		switch {
		case hasFlag(flagIsCheckStateMixed):
			out.Checked = 2
		case hasFlag(flagIsChecked):
			out.Checked = 1
		}
	case hasToggled:
		out.ClassName = classSwitch
		if hasFlag(flagIsToggled) {
			out.Checked = 1
		}
	}
	// configureExpanded
	if hasFlag(flagHasExpandedState) {
		out.Expanded = 1
		if hasFlag(flagIsExpanded) {
			out.Expanded = 2
		}
		if has(ui.SemanticsActionExpand) {
			add(androidActionExpand)
		}
		if has(ui.SemanticsActionCollapse) {
			add(androidActionCollapse)
		}
	}
	// configureRole
	if class, popup := roleClass(n.Role); class != "" {
		out.ClassName = class
		set(popup, propCanOpenPopup)
	}
	if n.Role == ui.SemanticsRoleProgressBar {
		out.RangeType, out.RangeMin, out.RangeMax, out.RangeCurrent = rangeOf(n.Value, n.MinValue, n.MaxValue)
	}

	out.Props = props
	out.AndroidActions = acts
	return out
}
