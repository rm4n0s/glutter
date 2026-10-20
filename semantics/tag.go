// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package semantics

import (
	"sync"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// VoidCallback is a function without parameters or result.
//
// Flutter: dart:ui VoidCallback
type VoidCallback = func()

// SemanticsNodeVisitor visits a node; it returns false to stop the walk.
//
// Flutter: semantics/semantics.dart SemanticsNodeVisitor
type SemanticsNodeVisitor func(node *SemanticsNode) bool

// MoveCursorHandler moves the cursor; extendSelection says whether the
// selection grows.
//
// Flutter: semantics/semantics.dart MoveCursorHandler
type MoveCursorHandler func(extendSelection bool)

// SetSelectionHandler sets the selection of a text field.
//
// Flutter: semantics/semantics.dart SetSelectionHandler
type SetSelectionHandler func(selection services.TextSelection)

// SetTextHandler replaces the text of a text field.
//
// Flutter: semantics/semantics.dart SetTextHandler
type SetTextHandler func(text string)

// ScrollToOffsetHandler scrolls to an offset.
//
// Flutter: semantics/semantics.dart ScrollToOffsetHandler
type ScrollToOffsetHandler func(targetOffset ui.Offset)

// SemanticsActionHandler handles an action; args are its arguments (see
// ui.SemanticsActionEvent).
//
// Flutter: semantics/semantics.dart SemanticsActionHandler
type SemanticsActionHandler func(args any)

// SemanticsUpdateCallback receives the update that the owner builds.
//
// Flutter: semantics/semantics.dart SemanticsUpdateCallback
type SemanticsUpdateCallback func(update *ui.SemanticsUpdate)

// ChildSemanticsConfigurationsDelegate decides which configurations of the
// children merge into the node and which form sibling groups.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsDelegate
type ChildSemanticsConfigurationsDelegate func(configs []*SemanticsConfiguration) *ChildSemanticsConfigurationsResult

// AccessibilityFocusBlockType says what the accessibility focus is blocked
// for.
//
// Flutter: semantics/semantics.dart AccessibilityFocusBlockType
type AccessibilityFocusBlockType int

// The values of AccessibilityFocusBlockType. The zero value is
// AccessibilityFocusBlockTypeNone.
const (
	// AccessibilityFocusBlockTypeNone does not block the accessibility focus.
	//
	// Flutter: semantics/semantics.dart AccessibilityFocusBlockType.none
	AccessibilityFocusBlockTypeNone AccessibilityFocusBlockType = iota
	// AccessibilityFocusBlockTypeBlockSubtree blocks the node and its subtree.
	//
	// Flutter: semantics/semantics.dart AccessibilityFocusBlockType.blockSubtree
	AccessibilityFocusBlockTypeBlockSubtree
	// AccessibilityFocusBlockTypeBlockNode blocks the node only.
	//
	// Flutter: semantics/semantics.dart AccessibilityFocusBlockType.blockNode
	AccessibilityFocusBlockTypeBlockNode
)

// Name returns the name of the value, such as "blockNode".
//
// Flutter: semantics/semantics.dart AccessibilityFocusBlockType.name
func (t AccessibilityFocusBlockType) Name() string {
	switch t {
	case AccessibilityFocusBlockTypeNone:
		return "none"
	case AccessibilityFocusBlockTypeBlockSubtree:
		return "blockSubtree"
	case AccessibilityFocusBlockTypeBlockNode:
		return "blockNode"
	}
	return "AccessibilityFocusBlockType(" + itoa(int(t)) + ")"
}

// String is "AccessibilityFocusBlockType.blockNode".
//
// Flutter: semantics/semantics.dart AccessibilityFocusBlockType.toString
func (t AccessibilityFocusBlockType) String() string {
	return "AccessibilityFocusBlockType." + t.Name()
}

// merge is the block type when two nodes get merged.
func (t AccessibilityFocusBlockType) merge(o AccessibilityFocusBlockType) AccessibilityFocusBlockType {
	if t == AccessibilityFocusBlockTypeBlockSubtree || o == AccessibilityFocusBlockTypeBlockSubtree {
		return AccessibilityFocusBlockTypeBlockSubtree
	}
	if t == AccessibilityFocusBlockTypeBlockNode || o == AccessibilityFocusBlockTypeBlockNode {
		return AccessibilityFocusBlockTypeBlockNode
	}
	return AccessibilityFocusBlockTypeNone
}

// unblockedUserActions are the actions that stay when the user actions of a
// node are blocked.
const unblockedUserActions = ui.SemanticsActionDidGainAccessibilityFocus | ui.SemanticsActionDidLoseAccessibilityFocus

// SemanticsTag is a tag that a node gives to its children, which a parent can
// use to find them. Tags with the same name are equal, like Dart's constant
// tags.
//
// Flutter: semantics/semantics.dart SemanticsTag
type SemanticsTag struct{ name string }

// NewSemanticsTag creates a tag with the name.
//
// Flutter: semantics/semantics.dart SemanticsTag.new
func NewSemanticsTag(name string) SemanticsTag { return SemanticsTag{name} }

// Name is the name of the tag.
//
// Flutter: semantics/semantics.dart SemanticsTag.name
func (t SemanticsTag) Name() string { return t.name }

// String is "SemanticsTag(name)".
//
// Flutter: semantics/semantics.dart SemanticsTag.toString
func (t SemanticsTag) String() string { return "SemanticsTag(" + t.name + ")" }

// ChildSemanticsConfigurationsResult is the decision of a
// ChildSemanticsConfigurationsDelegate.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResult
type ChildSemanticsConfigurationsResult struct {
	// MergeUp are the configurations that merge into the node.
	MergeUp []*SemanticsConfiguration
	// SiblingMergeGroups are groups of configurations that form a sibling node
	// of this one each.
	SiblingMergeGroups [][]*SemanticsConfiguration
}

// ChildSemanticsConfigurationsResultBuilder builds a
// ChildSemanticsConfigurationsResult.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResultBuilder
type ChildSemanticsConfigurationsResultBuilder struct {
	mergeUp            []*SemanticsConfiguration
	siblingMergeGroups [][]*SemanticsConfiguration
}

// NewChildSemanticsConfigurationsResultBuilder creates an empty builder.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResultBuilder.new
func NewChildSemanticsConfigurationsResultBuilder() *ChildSemanticsConfigurationsResultBuilder {
	return &ChildSemanticsConfigurationsResultBuilder{}
}

// MarkAsMergeUp makes the configuration merge into the node.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResultBuilder.markAsMergeUp
func (b *ChildSemanticsConfigurationsResultBuilder) MarkAsMergeUp(config *SemanticsConfiguration) {
	b.mergeUp = append(b.mergeUp, config)
}

// MarkAsSiblingMergeGroup makes the configurations merge into one sibling
// node.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResultBuilder.markAsSiblingMergeGroup
func (b *ChildSemanticsConfigurationsResultBuilder) MarkAsSiblingMergeGroup(configs []*SemanticsConfiguration) {
	b.siblingMergeGroups = append(b.siblingMergeGroups, configs)
}

// Build returns the result. In debug builds it panics with the tag
// SemanticsConfigurationDuplicated if a configuration was marked twice.
//
// Flutter: semantics/semantics.dart ChildSemanticsConfigurationsResultBuilder.build
func (b *ChildSemanticsConfigurationsResultBuilder) Build() *ChildSemanticsConfigurationsResult {
	assertNoDuplicates(b.mergeUp, b.siblingMergeGroups)
	return &ChildSemanticsConfigurationsResult{MergeUp: b.mergeUp, SiblingMergeGroups: b.siblingMergeGroups}
}

// CustomSemanticsAction is an action that an assistive technology can invoke
// on a node, with a label of its own, or the hint that replaces the one of a
// standard action.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction
type CustomSemanticsAction struct {
	label, hint string
	action      ui.SemanticsAction
	overriding  bool
}

// NewCustomSemanticsAction creates an action with a label, which must not be
// empty (SemanticsActionLabelEmpty).
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.new
func NewCustomSemanticsAction(label string) CustomSemanticsAction {
	assertNotEmpty("SemanticsActionLabelEmpty", "label", label)
	return CustomSemanticsAction{label: label}
}

// NewCustomSemanticsActionOverriding creates an action that gives the hint to
// a standard action; the hint must not be empty (SemanticsActionHintEmpty).
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.overridingAction
func NewCustomSemanticsActionOverriding(hint string, action ui.SemanticsAction) CustomSemanticsAction {
	assertNotEmpty("SemanticsActionHintEmpty", "hint", hint)
	return CustomSemanticsAction{hint: hint, action: action, overriding: true}
}

// Label is the label of the action; "" for an overriding action.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.label
func (a CustomSemanticsAction) Label() string { return a.label }

// Hint is the hint of an overriding action.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.hint
func (a CustomSemanticsAction) Hint() string { return a.hint }

// Action is the standard action that an overriding action gives a hint to;
// zero if there is none.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.action
func (a CustomSemanticsAction) Action() ui.SemanticsAction { return a.action }

// IsOverriding is whether it overrides the hint of a standard action.
func (a CustomSemanticsAction) IsOverriding() bool { return a.overriding }

// String describes the action, with its ID.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.toString
func (a CustomSemanticsAction) String() string {
	label, hint, action := "null", "null", "null"
	if !a.overriding {
		label = a.label
	} else {
		hint, action = a.hint, a.action.String()
	}
	return "CustomSemanticsAction(" + itoa(customActions.lookup(a)) + ", label:" + label + ", hint:" + hint + ", action:" + action + ")"
}

// customActionRegistry gives each distinct custom action a stable ID.
type customActionRegistry struct {
	mu      sync.Mutex
	next    int
	actions map[int]CustomSemanticsAction
	ids     map[CustomSemanticsAction]int
}

// customActions is the registry of IDs. Like Flutter's static maps it is shared
// by the process, only grows, and holds no framework state (SPEC §12.10).
var customActions = &customActionRegistry{actions: map[int]CustomSemanticsAction{}, ids: map[CustomSemanticsAction]int{}}

func (r *customActionRegistry) lookup(a CustomSemanticsAction) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.ids[a]; ok {
		return id
	}
	id := r.next
	r.next++
	r.ids[a] = id
	r.actions[id] = a
	return id
}

// CustomSemanticsActionGetIdentifier is the ID of the action; the same action
// always has the same ID.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.getIdentifier
func CustomSemanticsActionGetIdentifier(a CustomSemanticsAction) int { return customActions.lookup(a) }

// CustomSemanticsActionGetAction is the action with the ID, if there is one.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.getAction
func CustomSemanticsActionGetAction(id int) (CustomSemanticsAction, bool) {
	customActions.mu.Lock()
	defer customActions.mu.Unlock()
	a, ok := customActions.actions[id]
	return a, ok
}

// ResetCustomSemanticsActionsForTests forgets all the IDs.
//
// Flutter: semantics/semantics.dart CustomSemanticsAction.resetForTests
func ResetCustomSemanticsActionsForTests() {
	customActions.mu.Lock()
	defer customActions.mu.Unlock()
	customActions.next = 0
	clear(customActions.actions)
	clear(customActions.ids)
}
