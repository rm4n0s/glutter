// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package semantics

import (
	"maps"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsConfiguration describes the semantic information of a render
// object: the flags, the strings, the actions, and how the object takes part in
// the semantics tree.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration
type SemanticsConfiguration struct {
	// IsBlockingUserActions is whether the node and its subtree can not
	// perform user actions.
	//
	// Flutter: semantics/semantics.dart SemanticsConfiguration.isBlockingUserActions
	IsBlockingUserActions bool
	// ExplicitChildNodes is whether the children form their own nodes instead of
	// merging into this one.
	//
	// Flutter: semantics/semantics.dart SemanticsConfiguration.explicitChildNodes
	ExplicitChildNodes bool
	// IsBlockingSemanticsOfPreviouslyPaintedNodes hides the semantics of the
	// nodes painted before this one.
	//
	// Flutter: semantics/semantics.dart SemanticsConfiguration.isBlockingSemanticsOfPreviouslyPaintedNodes
	IsBlockingSemanticsOfPreviouslyPaintedNodes bool
	// Locale is the locale of the strings of the node.
	//
	// Flutter: semantics/semantics.dart SemanticsConfiguration.locale
	Locale foundation.Opt[ui.Locale]

	isSemanticBoundary              bool
	isMergingSemanticsOfDescendants bool
	hasBeenAnnotated                bool
	localeForSubtree                foundation.Opt[ui.Locale]
	actions                         map[ui.SemanticsAction]SemanticsActionHandler
	actionsAsBits                   ui.SemanticsAction
	customSemanticsActions          map[CustomSemanticsAction]VoidCallback
	childConfigurationsDelegate     ChildSemanticsConfigurationsDelegate
	sortKey                         SemanticsSortKey
	indexInParent                   foundation.Opt[int]
	scrollChildCount, scrollIndex   foundation.Opt[int]
	platformViewID                  foundation.Opt[int]
	maxValueLength, currentValueLen foundation.Opt[int]
	identifier                      string
	traversalParent, traversalChild any
	role                            ui.SemanticsRole
	attributedLabel                 AttributedString
	attributedValue                 AttributedString
	attributedIncreased             AttributedString
	attributedDecreased             AttributedString
	attributedHint                  AttributedString
	tooltip                         string
	hintOverrides                   *SemanticsHintOverrides
	textDirection                   ui.TextDirection
	flags                           ui.SemanticsFlags
	accessibilityFocusBlockType     AccessibilityFocusBlockType
	linkURL                         foundation.Opt[string]
	headingLevel                    int
	textSelection                   foundation.Opt[services.TextSelection]
	scrollPosition, scrollExtentMax foundation.Opt[float64]
	scrollExtentMin                 foundation.Opt[float64]
	controlsNodes                   []string
	validationResult                ui.SemanticsValidationResult
	hitTestBehavior                 ui.SemanticsHitTestBehavior
	inputType                       ui.SemanticsInputType
	minValue, maxValue              foundation.Opt[string]
	tagsForChildren                 map[SemanticsTag]struct{}

	onTap, onLongPress, onScrollLeft, onScrollRight, onScrollUp, onScrollDown VoidCallback
	onDismiss, onIncrease, onDecrease, onCopy, onCut, onPaste, onShowOnScreen VoidCallback
	onDidGainAccessibilityFocus, onDidLoseAccessibilityFocus, onFocus         VoidCallback
	onExpand, onCollapse                                                      VoidCallback
	onScrollToOffset                                                          ScrollToOffsetHandler
	onMoveCursorForwardByCharacter, onMoveCursorBackwardByCharacter           MoveCursorHandler
	onMoveCursorForwardByWord, onMoveCursorBackwardByWord                     MoveCursorHandler
	onSetSelection                                                            SetSelectionHandler
	onSetText                                                                 SetTextHandler
}

// NewSemanticsConfiguration creates an empty configuration.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.new
func NewSemanticsConfiguration() *SemanticsConfiguration {
	return &SemanticsConfiguration{
		actions:                map[ui.SemanticsAction]SemanticsActionHandler{},
		customSemanticsActions: map[CustomSemanticsAction]VoidCallback{},
		hitTestBehavior:        ui.SemanticsHitTestBehaviorDefer,
	}
}

func (c *SemanticsConfiguration) annotate() { c.hasBeenAnnotated = true }

// IsSemanticBoundary is whether the object forms a node of its own.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSemanticBoundary
func (c *SemanticsConfiguration) IsSemanticBoundary() bool { return c.isSemanticBoundary }

// SetIsSemanticBoundary sets whether the object forms a node of its own. In
// debug builds clearing it while the configuration merges the semantics of its
// descendants panics with the tag SemanticsBoundaryRequired.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSemanticBoundary
func (c *SemanticsConfiguration) SetIsSemanticBoundary(value bool) {
	if foundation.DebugMode && c.isMergingSemanticsOfDescendants && !value {
		panic(errors.New("SemanticsBoundaryRequired", "a configuration that merges the semantics of its descendants must be a semantic boundary"))
	}
	c.isSemanticBoundary = value
}

// LocaleForSubtree is the locale that the node and its subtree use.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.localeForSubtree
func (c *SemanticsConfiguration) LocaleForSubtree() foundation.Opt[ui.Locale] {
	return c.localeForSubtree
}

// SetLocaleForSubtree sets the locale of the node and its subtree.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.localeForSubtree
func (c *SemanticsConfiguration) SetLocaleForSubtree(value ui.Locale) {
	c.localeForSubtree = foundation.Some(value)
	c.annotate()
}

// HasBeenAnnotated is whether any property was set.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hasBeenAnnotated
func (c *SemanticsConfiguration) HasBeenAnnotated() bool { return c.hasBeenAnnotated }

// IsMergingSemanticsOfDescendants is whether the node merges the semantics of
// its descendants into itself.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isMergingSemanticsOfDescendants
func (c *SemanticsConfiguration) IsMergingSemanticsOfDescendants() bool {
	return c.isMergingSemanticsOfDescendants
}

// SetIsMergingSemanticsOfDescendants sets whether the node merges its
// descendants. In debug builds it panics with the tag SemanticsBoundaryRequired
// if the configuration is not a semantic boundary.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isMergingSemanticsOfDescendants
func (c *SemanticsConfiguration) SetIsMergingSemanticsOfDescendants(value bool) {
	if foundation.DebugMode && !c.isSemanticBoundary {
		panic(errors.New("SemanticsBoundaryRequired", "only a semantic boundary can merge the semantics of its descendants"))
	}
	c.isMergingSemanticsOfDescendants = value
	c.annotate()
}

func (c *SemanticsConfiguration) effectiveActionsAsBits() ui.SemanticsAction {
	if c.IsBlockingUserActions {
		return c.actionsAsBits & unblockedUserActions
	}
	return c.actionsAsBits
}

func (c *SemanticsConfiguration) addAction(action ui.SemanticsAction, handler SemanticsActionHandler) {
	c.actions[action] = handler
	c.actionsAsBits |= action
	c.annotate()
}

func (c *SemanticsConfiguration) addArgumentlessAction(action ui.SemanticsAction, handler VoidCallback) {
	c.assertHandler(handler != nil)
	c.addAction(action, func(args any) {
		if foundation.DebugMode && args != nil {
			panic(errors.New("SemanticsActionArgumentsUnexpected", "this action takes no arguments", "action", action.Name()))
		}
		handler()
	})
}

func (c *SemanticsConfiguration) assertHandler(ok bool) {
	if foundation.DebugMode && !ok {
		panic(errors.New("SemanticsHandlerNil", "the handler of a semantics action must not be nil"))
	}
}

// OnTap is the handler of the tap action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onTap
func (c *SemanticsConfiguration) OnTap() VoidCallback { return c.onTap }

// SetOnTap sets the handler of the tap action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onTap
func (c *SemanticsConfiguration) SetOnTap(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionTap, v)
	c.onTap = v
}

// OnLongPress is the handler of the long press action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onLongPress
func (c *SemanticsConfiguration) OnLongPress() VoidCallback { return c.onLongPress }

// SetOnLongPress sets the handler of the long press action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onLongPress
func (c *SemanticsConfiguration) SetOnLongPress(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionLongPress, v)
	c.onLongPress = v
}

// OnScrollLeft is the handler of the scroll left action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollLeft
func (c *SemanticsConfiguration) OnScrollLeft() VoidCallback { return c.onScrollLeft }

// SetOnScrollLeft sets the handler of the scroll left action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollLeft
func (c *SemanticsConfiguration) SetOnScrollLeft(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionScrollLeft, v)
	c.onScrollLeft = v
}

// OnScrollRight is the handler of the scroll right action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollRight
func (c *SemanticsConfiguration) OnScrollRight() VoidCallback { return c.onScrollRight }

// SetOnScrollRight sets the handler of the scroll right action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollRight
func (c *SemanticsConfiguration) SetOnScrollRight(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionScrollRight, v)
	c.onScrollRight = v
}

// OnScrollUp is the handler of the scroll up action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollUp
func (c *SemanticsConfiguration) OnScrollUp() VoidCallback { return c.onScrollUp }

// SetOnScrollUp sets the handler of the scroll up action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollUp
func (c *SemanticsConfiguration) SetOnScrollUp(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionScrollUp, v)
	c.onScrollUp = v
}

// OnScrollDown is the handler of the scroll down action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollDown
func (c *SemanticsConfiguration) OnScrollDown() VoidCallback { return c.onScrollDown }

// SetOnScrollDown sets the handler of the scroll down action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollDown
func (c *SemanticsConfiguration) SetOnScrollDown(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionScrollDown, v)
	c.onScrollDown = v
}

// OnDismiss is the handler of the dismiss action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDismiss
func (c *SemanticsConfiguration) OnDismiss() VoidCallback { return c.onDismiss }

// SetOnDismiss sets the handler of the dismiss action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDismiss
func (c *SemanticsConfiguration) SetOnDismiss(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionDismiss, v)
	c.onDismiss = v
}

// OnScrollToOffset is the handler of the scroll to offset action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollToOffset
func (c *SemanticsConfiguration) OnScrollToOffset() ScrollToOffsetHandler { return c.onScrollToOffset }

// SetOnScrollToOffset sets the handler of the scroll to offset action. The
// argument of the action is a []float64 with the offset.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onScrollToOffset
func (c *SemanticsConfiguration) SetOnScrollToOffset(v ScrollToOffsetHandler) {
	c.assertHandler(v != nil)
	c.addAction(ui.SemanticsActionScrollToOffset, func(args any) {
		list := args.([]float64)
		v(ui.Offset{Dx: list[0], Dy: list[1]})
	})
	c.onScrollToOffset = v
}

// OnIncrease is the handler of the increase action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onIncrease
func (c *SemanticsConfiguration) OnIncrease() VoidCallback { return c.onIncrease }

// SetOnIncrease sets the handler of the increase action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onIncrease
func (c *SemanticsConfiguration) SetOnIncrease(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionIncrease, v)
	c.onIncrease = v
}

// OnDecrease is the handler of the decrease action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDecrease
func (c *SemanticsConfiguration) OnDecrease() VoidCallback { return c.onDecrease }

// SetOnDecrease sets the handler of the decrease action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDecrease
func (c *SemanticsConfiguration) SetOnDecrease(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionDecrease, v)
	c.onDecrease = v
}

// OnCopy is the handler of the copy action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCopy
func (c *SemanticsConfiguration) OnCopy() VoidCallback { return c.onCopy }

// SetOnCopy sets the handler of the copy action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCopy
func (c *SemanticsConfiguration) SetOnCopy(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionCopy, v)
	c.onCopy = v
}

// OnCut is the handler of the cut action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCut
func (c *SemanticsConfiguration) OnCut() VoidCallback { return c.onCut }

// SetOnCut sets the handler of the cut action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCut
func (c *SemanticsConfiguration) SetOnCut(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionCut, v)
	c.onCut = v
}

// OnPaste is the handler of the paste action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onPaste
func (c *SemanticsConfiguration) OnPaste() VoidCallback { return c.onPaste }

// SetOnPaste sets the handler of the paste action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onPaste
func (c *SemanticsConfiguration) SetOnPaste(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionPaste, v)
	c.onPaste = v
}

// OnShowOnScreen is the handler of the show on screen action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onShowOnScreen
func (c *SemanticsConfiguration) OnShowOnScreen() VoidCallback { return c.onShowOnScreen }

// SetOnShowOnScreen sets the handler of the show on screen action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onShowOnScreen
func (c *SemanticsConfiguration) SetOnShowOnScreen(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionShowOnScreen, v)
	c.onShowOnScreen = v
}

func (c *SemanticsConfiguration) addMoveCursor(action ui.SemanticsAction, v MoveCursorHandler) {
	c.assertHandler(v != nil)
	c.addAction(action, func(args any) { v(args.(bool)) })
}

// OnMoveCursorForwardByCharacter is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorForwardByCharacter
func (c *SemanticsConfiguration) OnMoveCursorForwardByCharacter() MoveCursorHandler {
	return c.onMoveCursorForwardByCharacter
}

// SetOnMoveCursorForwardByCharacter sets the handler of the action; its
// argument is whether the selection extends.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorForwardByCharacter
func (c *SemanticsConfiguration) SetOnMoveCursorForwardByCharacter(v MoveCursorHandler) {
	c.addMoveCursor(ui.SemanticsActionMoveCursorForwardByCharacter, v)
	c.onMoveCursorForwardByCharacter = v
}

// OnMoveCursorBackwardByCharacter is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorBackwardByCharacter
func (c *SemanticsConfiguration) OnMoveCursorBackwardByCharacter() MoveCursorHandler {
	return c.onMoveCursorBackwardByCharacter
}

// SetOnMoveCursorBackwardByCharacter sets the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorBackwardByCharacter
func (c *SemanticsConfiguration) SetOnMoveCursorBackwardByCharacter(v MoveCursorHandler) {
	c.addMoveCursor(ui.SemanticsActionMoveCursorBackwardByCharacter, v)
	c.onMoveCursorBackwardByCharacter = v
}

// OnMoveCursorForwardByWord is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorForwardByWord
func (c *SemanticsConfiguration) OnMoveCursorForwardByWord() MoveCursorHandler {
	return c.onMoveCursorForwardByWord
}

// SetOnMoveCursorForwardByWord sets the handler of the action.
//
// Deviation: Flutter stores the handler in the by-character field, a typo; here
// each setter has its own field (SPEC §12.10).
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorForwardByWord
func (c *SemanticsConfiguration) SetOnMoveCursorForwardByWord(v MoveCursorHandler) {
	c.addMoveCursor(ui.SemanticsActionMoveCursorForwardByWord, v)
	c.onMoveCursorForwardByWord = v
}

// OnMoveCursorBackwardByWord is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorBackwardByWord
func (c *SemanticsConfiguration) OnMoveCursorBackwardByWord() MoveCursorHandler {
	return c.onMoveCursorBackwardByWord
}

// SetOnMoveCursorBackwardByWord sets the handler of the action.
//
// Deviation: see SetOnMoveCursorForwardByWord.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onMoveCursorBackwardByWord
func (c *SemanticsConfiguration) SetOnMoveCursorBackwardByWord(v MoveCursorHandler) {
	c.addMoveCursor(ui.SemanticsActionMoveCursorBackwardByWord, v)
	c.onMoveCursorBackwardByWord = v
}

// OnSetSelection is the handler of the set selection action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onSetSelection
func (c *SemanticsConfiguration) OnSetSelection() SetSelectionHandler { return c.onSetSelection }

// SetOnSetSelection sets the handler of the set selection action. The argument
// of the action is a map[string]int with "base" and "extent".
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onSetSelection
func (c *SemanticsConfiguration) SetOnSetSelection(v SetSelectionHandler) {
	c.assertHandler(v != nil)
	c.addAction(ui.SemanticsActionSetSelection, func(args any) {
		sel := args.(map[string]int)
		v(services.NewTextSelection(sel["base"], sel["extent"]))
	})
	c.onSetSelection = v
}

// OnSetText is the handler of the set text action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onSetText
func (c *SemanticsConfiguration) OnSetText() SetTextHandler { return c.onSetText }

// SetOnSetText sets the handler of the set text action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onSetText
func (c *SemanticsConfiguration) SetOnSetText(v SetTextHandler) {
	c.assertHandler(v != nil)
	c.addAction(ui.SemanticsActionSetText, func(args any) { v(args.(string)) })
	c.onSetText = v
}

// OnDidGainAccessibilityFocus is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDidGainAccessibilityFocus
func (c *SemanticsConfiguration) OnDidGainAccessibilityFocus() VoidCallback {
	return c.onDidGainAccessibilityFocus
}

// SetOnDidGainAccessibilityFocus sets the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDidGainAccessibilityFocus
func (c *SemanticsConfiguration) SetOnDidGainAccessibilityFocus(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionDidGainAccessibilityFocus, v)
	c.onDidGainAccessibilityFocus = v
}

// OnDidLoseAccessibilityFocus is the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDidLoseAccessibilityFocus
func (c *SemanticsConfiguration) OnDidLoseAccessibilityFocus() VoidCallback {
	return c.onDidLoseAccessibilityFocus
}

// SetOnDidLoseAccessibilityFocus sets the handler of the action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onDidLoseAccessibilityFocus
func (c *SemanticsConfiguration) SetOnDidLoseAccessibilityFocus(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionDidLoseAccessibilityFocus, v)
	c.onDidLoseAccessibilityFocus = v
}

// OnFocus is the handler of the focus action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onFocus
func (c *SemanticsConfiguration) OnFocus() VoidCallback { return c.onFocus }

// SetOnFocus sets the handler of the focus action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onFocus
func (c *SemanticsConfiguration) SetOnFocus(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionFocus, v)
	c.onFocus = v
}

// OnExpand is the handler of the expand action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onExpand
func (c *SemanticsConfiguration) OnExpand() VoidCallback { return c.onExpand }

// SetOnExpand sets the handler of the expand action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onExpand
func (c *SemanticsConfiguration) SetOnExpand(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionExpand, v)
	c.onExpand = v
}

// OnCollapse is the handler of the collapse action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCollapse
func (c *SemanticsConfiguration) OnCollapse() VoidCallback { return c.onCollapse }

// SetOnCollapse sets the handler of the collapse action.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.onCollapse
func (c *SemanticsConfiguration) SetOnCollapse(v VoidCallback) {
	c.addArgumentlessAction(ui.SemanticsActionCollapse, v)
	c.onCollapse = v
}

// ChildConfigurationsDelegate is the delegate that decides how the
// configurations of the children merge.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.childConfigurationsDelegate
func (c *SemanticsConfiguration) ChildConfigurationsDelegate() ChildSemanticsConfigurationsDelegate {
	return c.childConfigurationsDelegate
}

// SetChildConfigurationsDelegate sets the delegate.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.childConfigurationsDelegate
func (c *SemanticsConfiguration) SetChildConfigurationsDelegate(v ChildSemanticsConfigurationsDelegate) {
	c.assertHandler(v != nil)
	c.childConfigurationsDelegate = v
}

// GetActionHandler is the handler of the action, or nil.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.getActionHandler
func (c *SemanticsConfiguration) GetActionHandler(action ui.SemanticsAction) SemanticsActionHandler {
	return c.actions[action]
}

// SortKey is the sort key of the node.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.sortKey
func (c *SemanticsConfiguration) SortKey() SemanticsSortKey { return c.sortKey }

// SetSortKey sets the sort key.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.sortKey
func (c *SemanticsConfiguration) SetSortKey(v SemanticsSortKey) {
	c.sortKey = v
	c.annotate()
}

// IndexInParent is the index among the children of the parent that scroll.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.indexInParent
func (c *SemanticsConfiguration) IndexInParent() foundation.Opt[int] { return c.indexInParent }

// SetIndexInParent sets the index in the parent.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.indexInParent
func (c *SemanticsConfiguration) SetIndexInParent(v foundation.Opt[int]) {
	c.indexInParent = v
	c.annotate()
}

func setOptInt(dst *foundation.Opt[int], v foundation.Opt[int], c *SemanticsConfiguration) {
	if optEqual(*dst, v) {
		return
	}
	*dst = v
	c.annotate()
}

// ScrollChildCount is the number of children of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollChildCount
func (c *SemanticsConfiguration) ScrollChildCount() foundation.Opt[int] { return c.scrollChildCount }

// SetScrollChildCount sets the number of children of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollChildCount
func (c *SemanticsConfiguration) SetScrollChildCount(v foundation.Opt[int]) {
	setOptInt(&c.scrollChildCount, v, c)
}

// ScrollIndex is the index of the first visible child of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollIndex
func (c *SemanticsConfiguration) ScrollIndex() foundation.Opt[int] { return c.scrollIndex }

// SetScrollIndex sets the index of the first visible child.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollIndex
func (c *SemanticsConfiguration) SetScrollIndex(v foundation.Opt[int]) {
	setOptInt(&c.scrollIndex, v, c)
}

// PlatformViewID is the platform view that the node stands for.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.platformViewId
func (c *SemanticsConfiguration) PlatformViewID() foundation.Opt[int] { return c.platformViewID }

// SetPlatformViewID sets the platform view.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.platformViewId
func (c *SemanticsConfiguration) SetPlatformViewID(v foundation.Opt[int]) {
	setOptInt(&c.platformViewID, v, c)
}

// MaxValueLength is the longest value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.maxValueLength
func (c *SemanticsConfiguration) MaxValueLength() foundation.Opt[int] { return c.maxValueLength }

// SetMaxValueLength sets the longest value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.maxValueLength
func (c *SemanticsConfiguration) SetMaxValueLength(v foundation.Opt[int]) {
	setOptInt(&c.maxValueLength, v, c)
}

// CurrentValueLength is the length of the value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.currentValueLength
func (c *SemanticsConfiguration) CurrentValueLength() foundation.Opt[int] {
	return c.currentValueLen
}

// SetCurrentValueLength sets the length of the value of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.currentValueLength
func (c *SemanticsConfiguration) SetCurrentValueLength(v foundation.Opt[int]) {
	setOptInt(&c.currentValueLen, v, c)
}

// CustomSemanticsActions are the custom actions with their handlers. The map is
// the configuration's own: changing it changes the configuration.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.customSemanticsActions
func (c *SemanticsConfiguration) CustomSemanticsActions() map[CustomSemanticsAction]VoidCallback {
	return c.customSemanticsActions
}

// SetCustomSemanticsActions replaces the custom actions.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.customSemanticsActions
func (c *SemanticsConfiguration) SetCustomSemanticsActions(v map[CustomSemanticsAction]VoidCallback) {
	c.annotate()
	c.actionsAsBits |= ui.SemanticsActionCustomAction
	c.customSemanticsActions = v
	c.actions[ui.SemanticsActionCustomAction] = c.onCustomSemanticsAction
}

func (c *SemanticsConfiguration) onCustomSemanticsAction(args any) {
	action, ok := CustomSemanticsActionGetAction(args.(int))
	if !ok {
		return
	}
	if cb := c.customSemanticsActions[action]; cb != nil {
		cb()
	}
}

// Identifier is the identifier for tests.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.identifier
func (c *SemanticsConfiguration) Identifier() string { return c.identifier }

// SetIdentifier sets the identifier.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.identifier
func (c *SemanticsConfiguration) SetIdentifier(v string) {
	c.identifier = v
	c.annotate()
}

// TraversalParentIdentifier ties this node to the nodes with the same child
// identifier.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.traversalParentIdentifier
func (c *SemanticsConfiguration) TraversalParentIdentifier() any { return c.traversalParent }

// SetTraversalParentIdentifier sets the traversal parent identifier.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.traversalParentIdentifier
func (c *SemanticsConfiguration) SetTraversalParentIdentifier(v any) {
	if v == c.traversalParent {
		return
	}
	c.traversalParent = v
	c.annotate()
}

// TraversalChildIdentifier ties this node to the node with the same parent
// identifier.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.traversalChildIdentifier
func (c *SemanticsConfiguration) TraversalChildIdentifier() any { return c.traversalChild }

// SetTraversalChildIdentifier sets the traversal child identifier.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.traversalChildIdentifier
func (c *SemanticsConfiguration) SetTraversalChildIdentifier(v any) {
	if v == c.traversalChild {
		return
	}
	c.traversalChild = v
	c.annotate()
}

// Role is the role.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.role
func (c *SemanticsConfiguration) Role() ui.SemanticsRole { return c.role }

// SetRole sets the role.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.role
func (c *SemanticsConfiguration) SetRole(v ui.SemanticsRole) {
	c.role = v
	c.annotate()
}

// Label is the text of the label.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.label
func (c *SemanticsConfiguration) Label() string { return c.attributedLabel.String }

// SetLabel sets the label.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.label
func (c *SemanticsConfiguration) SetLabel(v string) {
	c.attributedLabel = NewAttributedString(v)
	c.annotate()
}

// AttributedLabel is the label with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedLabel
func (c *SemanticsConfiguration) AttributedLabel() AttributedString { return c.attributedLabel }

// SetAttributedLabel sets the label with attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedLabel
func (c *SemanticsConfiguration) SetAttributedLabel(v AttributedString) {
	c.attributedLabel = v
	c.annotate()
}

// Value is the text of the value.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.value
func (c *SemanticsConfiguration) Value() string { return c.attributedValue.String }

// SetValue sets the value.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.value
func (c *SemanticsConfiguration) SetValue(v string) {
	c.attributedValue = NewAttributedString(v)
	c.annotate()
}

// AttributedValue is the value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedValue
func (c *SemanticsConfiguration) AttributedValue() AttributedString { return c.attributedValue }

// SetAttributedValue sets the value with attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedValue
func (c *SemanticsConfiguration) SetAttributedValue(v AttributedString) {
	c.attributedValue = v
	c.annotate()
}

// IncreasedValue is the text of the value after an increase.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.increasedValue
func (c *SemanticsConfiguration) IncreasedValue() string { return c.attributedIncreased.String }

// SetIncreasedValue sets the value after an increase.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.increasedValue
func (c *SemanticsConfiguration) SetIncreasedValue(v string) {
	c.attributedIncreased = NewAttributedString(v)
	c.annotate()
}

// AttributedIncreasedValue is the increased value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedIncreasedValue
func (c *SemanticsConfiguration) AttributedIncreasedValue() AttributedString {
	return c.attributedIncreased
}

// SetAttributedIncreasedValue sets the increased value with attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedIncreasedValue
func (c *SemanticsConfiguration) SetAttributedIncreasedValue(v AttributedString) {
	c.attributedIncreased = v
	c.annotate()
}

// DecreasedValue is the text of the value after a decrease.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.decreasedValue
func (c *SemanticsConfiguration) DecreasedValue() string { return c.attributedDecreased.String }

// SetDecreasedValue sets the value after a decrease.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.decreasedValue
func (c *SemanticsConfiguration) SetDecreasedValue(v string) {
	c.attributedDecreased = NewAttributedString(v)
	c.annotate()
}

// AttributedDecreasedValue is the decreased value with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedDecreasedValue
func (c *SemanticsConfiguration) AttributedDecreasedValue() AttributedString {
	return c.attributedDecreased
}

// SetAttributedDecreasedValue sets the decreased value with attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedDecreasedValue
func (c *SemanticsConfiguration) SetAttributedDecreasedValue(v AttributedString) {
	c.attributedDecreased = v
	c.annotate()
}

// Hint is the text of the hint.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hint
func (c *SemanticsConfiguration) Hint() string { return c.attributedHint.String }

// SetHint sets the hint.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hint
func (c *SemanticsConfiguration) SetHint(v string) {
	c.attributedHint = NewAttributedString(v)
	c.annotate()
}

// AttributedHint is the hint with its attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedHint
func (c *SemanticsConfiguration) AttributedHint() AttributedString { return c.attributedHint }

// SetAttributedHint sets the hint with attributes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.attributedHint
func (c *SemanticsConfiguration) SetAttributedHint(v AttributedString) {
	c.attributedHint = v
	c.annotate()
}

// Tooltip is the tooltip.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.tooltip
func (c *SemanticsConfiguration) Tooltip() string { return c.tooltip }

// SetTooltip sets the tooltip.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.tooltip
func (c *SemanticsConfiguration) SetTooltip(v string) {
	c.tooltip = v
	c.annotate()
}

// HintOverrides are the hints that replace the default hints of tap and long
// press.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hintOverrides
func (c *SemanticsConfiguration) HintOverrides() *SemanticsHintOverrides { return c.hintOverrides }

// SetHintOverrides sets the hint overrides; nil is ignored.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hintOverrides
func (c *SemanticsConfiguration) SetHintOverrides(v *SemanticsHintOverrides) {
	if v == nil {
		return
	}
	c.hintOverrides = v
	c.annotate()
}

func (c *SemanticsConfiguration) setFlags(f func(*ui.SemanticsFlags)) {
	f(&c.flags)
	c.annotate()
}

// ScopesRoute is whether the node is the root of a route.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scopesRoute
func (c *SemanticsConfiguration) ScopesRoute() bool { return c.flags.ScopesRoute }

// SetScopesRoute sets whether the node is the root of a route.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scopesRoute
func (c *SemanticsConfiguration) SetScopesRoute(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.ScopesRoute = v })
}

// NamesRoute is whether the node names the route.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.namesRoute
func (c *SemanticsConfiguration) NamesRoute() bool { return c.flags.NamesRoute }

// SetNamesRoute sets whether the node names the route.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.namesRoute
func (c *SemanticsConfiguration) SetNamesRoute(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.NamesRoute = v })
}

// IsImage is whether the node is an image.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isImage
func (c *SemanticsConfiguration) IsImage() bool { return c.flags.IsImage }

// SetIsImage sets whether the node is an image.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isImage
func (c *SemanticsConfiguration) SetIsImage(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsImage = v })
}

// LiveRegion is whether changes of the node are announced.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.liveRegion
func (c *SemanticsConfiguration) LiveRegion() bool { return c.flags.IsLiveRegion }

// SetLiveRegion sets whether changes of the node are announced.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.liveRegion
func (c *SemanticsConfiguration) SetLiveRegion(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsLiveRegion = v })
}

// TextDirection is the direction of the strings.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.textDirection
func (c *SemanticsConfiguration) TextDirection() ui.TextDirection { return c.textDirection }

// SetTextDirection sets the direction of the strings.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.textDirection
func (c *SemanticsConfiguration) SetTextDirection(v ui.TextDirection) {
	c.textDirection = v
	c.annotate()
}

// IsSelected is whether the node is selected.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSelected
func (c *SemanticsConfiguration) IsSelected() bool { return c.flags.IsSelected == ui.TristateIsTrue }

// SetIsSelected sets whether the node is selected.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSelected
func (c *SemanticsConfiguration) SetIsSelected(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsSelected = ui.TristateFromBool(v) })
}

func optBoolOf(t ui.Tristate) foundation.Opt[bool] {
	if b := t.ToBoolOrNull(); b != nil {
		return foundation.Some(*b)
	}
	return foundation.None[bool]()
}

func tristateOf(v foundation.Opt[bool]) ui.Tristate {
	if b, ok := v.Get(); ok {
		return ui.TristateFromBool(b)
	}
	return ui.TristateNone
}

// IsExpanded is whether an expandable node is expanded; None if it is not
// expandable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isExpanded
func (c *SemanticsConfiguration) IsExpanded() foundation.Opt[bool] {
	return optBoolOf(c.flags.IsExpanded)
}

// SetIsExpanded sets whether the node is expanded.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isExpanded
func (c *SemanticsConfiguration) SetIsExpanded(v foundation.Opt[bool]) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsExpanded = tristateOf(v) })
}

// IsEnabled is whether the node is enabled; None if it has no such state.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isEnabled
func (c *SemanticsConfiguration) IsEnabled() foundation.Opt[bool] {
	return optBoolOf(c.flags.IsEnabled)
}

// SetIsEnabled sets whether the node is enabled.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isEnabled
func (c *SemanticsConfiguration) SetIsEnabled(v foundation.Opt[bool]) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsEnabled = tristateOf(v) })
}

// IsChecked is whether the node is checked; None if it is not checkable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isChecked
func (c *SemanticsConfiguration) IsChecked() foundation.Opt[bool] {
	if c.flags.IsChecked == ui.CheckedStateNone {
		return foundation.None[bool]()
	}
	return foundation.Some(c.flags.IsChecked == ui.CheckedStateIsTrue)
}

// SetIsChecked sets whether the node is checked; None leaves the state.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isChecked
func (c *SemanticsConfiguration) SetIsChecked(v foundation.Opt[bool]) {
	if b, ok := v.Get(); ok {
		state := ui.CheckedStateIsFalse
		if b {
			state = ui.CheckedStateIsTrue
		}
		c.flags.IsChecked = state
	}
	c.annotate()
}

// IsCheckStateMixed is whether the check state is mixed; None if it is not
// checkable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isCheckStateMixed
func (c *SemanticsConfiguration) IsCheckStateMixed() foundation.Opt[bool] {
	if c.flags.IsChecked == ui.CheckedStateNone {
		return foundation.None[bool]()
	}
	return foundation.Some(c.flags.IsChecked == ui.CheckedStateMixed)
}

// SetIsCheckStateMixed makes the check state mixed when true.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isCheckStateMixed
func (c *SemanticsConfiguration) SetIsCheckStateMixed(v foundation.Opt[bool]) {
	if v.OrElse(false) {
		c.flags.IsChecked = ui.CheckedStateMixed
	}
	c.annotate()
}

// IsToggled is whether a toggle is on; None if the node is not a toggle.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isToggled
func (c *SemanticsConfiguration) IsToggled() foundation.Opt[bool] {
	return optBoolOf(c.flags.IsToggled)
}

// SetIsToggled sets whether a toggle is on.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isToggled
func (c *SemanticsConfiguration) SetIsToggled(v foundation.Opt[bool]) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsToggled = tristateOf(v) })
}

// IsInMutuallyExclusiveGroup is whether the node is in a radio group.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isInMutuallyExclusiveGroup
func (c *SemanticsConfiguration) IsInMutuallyExclusiveGroup() bool {
	return c.flags.IsInMutuallyExclusiveGroup
}

// SetIsInMutuallyExclusiveGroup sets whether the node is in a radio group.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isInMutuallyExclusiveGroup
func (c *SemanticsConfiguration) SetIsInMutuallyExclusiveGroup(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsInMutuallyExclusiveGroup = v })
}

// IsFocusable is whether the node can have the input focus.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isFocusable
func (c *SemanticsConfiguration) IsFocusable() bool { return c.flags.IsFocused != ui.TristateNone }

// SetIsFocusable sets whether the node can have the input focus. Flutter
// deprecates it: set IsFocused.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isFocusable
func (c *SemanticsConfiguration) SetIsFocusable(v bool) {
	if !v {
		c.flags.IsFocused = ui.TristateNone
	} else if c.flags.IsFocused == ui.TristateNone {
		c.flags.IsFocused = ui.TristateIsFalse
	}
	c.annotate()
}

// IsFocused is whether the node has the input focus; None if it can not.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isFocused
func (c *SemanticsConfiguration) IsFocused() foundation.Opt[bool] {
	return optBoolOf(c.flags.IsFocused)
}

// SetIsFocused sets whether the node has the input focus.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isFocused
func (c *SemanticsConfiguration) SetIsFocused(v foundation.Opt[bool]) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsFocused = tristateOf(v) })
}

// AccessibilityFocusBlockType is what the accessibility focus is blocked for.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.accessibilityFocusBlockType
func (c *SemanticsConfiguration) AccessibilityFocusBlockType() AccessibilityFocusBlockType {
	return c.accessibilityFocusBlockType
}

// SetAccessibilityFocusBlockType sets what the accessibility focus is blocked
// for, and the flag that says it is blocked.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.accessibilityFocusBlockType
func (c *SemanticsConfiguration) SetAccessibilityFocusBlockType(v AccessibilityFocusBlockType) {
	c.accessibilityFocusBlockType = v
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsAccessibilityFocusBlocked = v != AccessibilityFocusBlockTypeNone })
}

// IsButton is whether the node is a button.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isButton
func (c *SemanticsConfiguration) IsButton() bool { return c.flags.IsButton }

// SetIsButton sets whether the node is a button.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isButton
func (c *SemanticsConfiguration) SetIsButton(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsButton = v })
}

// IsLink is whether the node is a link.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isLink
func (c *SemanticsConfiguration) IsLink() bool { return c.flags.IsLink }

// SetIsLink sets whether the node is a link.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isLink
func (c *SemanticsConfiguration) SetIsLink(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsLink = v })
}

// LinkURL is the URL of a link.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.linkUrl
func (c *SemanticsConfiguration) LinkURL() foundation.Opt[string] { return c.linkURL }

// SetLinkURL sets the URL of a link.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.linkUrl
func (c *SemanticsConfiguration) SetLinkURL(v foundation.Opt[string]) {
	if optEqual(v, c.linkURL) {
		return
	}
	c.linkURL = v
	c.annotate()
}

// IsHeader is whether the node is a header.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isHeader
func (c *SemanticsConfiguration) IsHeader() bool { return c.flags.IsHeader }

// SetIsHeader sets whether the node is a header.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isHeader
func (c *SemanticsConfiguration) SetIsHeader(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsHeader = v })
}

// HeadingLevel is the level of a heading, 0 to 6.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.headingLevel
func (c *SemanticsConfiguration) HeadingLevel() int { return c.headingLevel }

// SetHeadingLevel sets the heading level; in debug builds a level outside 0 to
// 6 panics with the tag SemanticsHeadingLevelInvalid.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.headingLevel
func (c *SemanticsConfiguration) SetHeadingLevel(v int) {
	if foundation.DebugMode && (v < 0 || v > 6) {
		panic(errors.New("SemanticsHeadingLevelInvalid", "the heading level must be between 0 and 6", "level", v))
	}
	if v == c.headingLevel {
		return
	}
	c.headingLevel = v
	c.annotate()
}

// IsSlider is whether the node is a slider.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSlider
func (c *SemanticsConfiguration) IsSlider() bool { return c.flags.IsSlider }

// SetIsSlider sets whether the node is a slider.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isSlider
func (c *SemanticsConfiguration) SetIsSlider(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsSlider = v })
}

// IsKeyboardKey is whether the node is a key of an on screen keyboard.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isKeyboardKey
func (c *SemanticsConfiguration) IsKeyboardKey() bool { return c.flags.IsKeyboardKey }

// SetIsKeyboardKey sets whether the node is a key of an on screen keyboard.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isKeyboardKey
func (c *SemanticsConfiguration) SetIsKeyboardKey(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsKeyboardKey = v })
}

// IsHidden is whether the node is hidden from the user.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isHidden
func (c *SemanticsConfiguration) IsHidden() bool { return c.flags.IsHidden }

// SetIsHidden sets whether the node is hidden from the user.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isHidden
func (c *SemanticsConfiguration) SetIsHidden(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsHidden = v })
}

// IsTextField is whether the node is a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isTextField
func (c *SemanticsConfiguration) IsTextField() bool { return c.flags.IsTextField }

// SetIsTextField sets whether the node is a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isTextField
func (c *SemanticsConfiguration) SetIsTextField(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsTextField = v })
}

// IsReadOnly is whether the text field is read only.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isReadOnly
func (c *SemanticsConfiguration) IsReadOnly() bool { return c.flags.IsReadOnly }

// SetIsReadOnly sets whether the text field is read only.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isReadOnly
func (c *SemanticsConfiguration) SetIsReadOnly(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsReadOnly = v })
}

// IsObscured is whether the text is obscured.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isObscured
func (c *SemanticsConfiguration) IsObscured() bool { return c.flags.IsObscured }

// SetIsObscured sets whether the text is obscured.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isObscured
func (c *SemanticsConfiguration) SetIsObscured(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsObscured = v })
}

// IsMultiline is whether the text field has several lines.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isMultiline
func (c *SemanticsConfiguration) IsMultiline() bool { return c.flags.IsMultiline }

// SetIsMultiline sets whether the text field has several lines.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isMultiline
func (c *SemanticsConfiguration) SetIsMultiline(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsMultiline = v })
}

// IsRequired is whether a form field is required; None if the node is not a
// field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isRequired
func (c *SemanticsConfiguration) IsRequired() foundation.Opt[bool] {
	return optBoolOf(c.flags.IsRequired)
}

// SetIsRequired sets whether a form field is required.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isRequired
func (c *SemanticsConfiguration) SetIsRequired(v foundation.Opt[bool]) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.IsRequired = tristateOf(v) })
}

// HasImplicitScrolling is whether the node scrolls when a child gets the
// accessibility focus.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hasImplicitScrolling
func (c *SemanticsConfiguration) HasImplicitScrolling() bool { return c.flags.HasImplicitScrolling }

// SetHasImplicitScrolling sets whether the node scrolls implicitly.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hasImplicitScrolling
func (c *SemanticsConfiguration) SetHasImplicitScrolling(v bool) {
	c.setFlags(func(f *ui.SemanticsFlags) { f.HasImplicitScrolling = v })
}

// TextSelection is the selection of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.textSelection
func (c *SemanticsConfiguration) TextSelection() foundation.Opt[services.TextSelection] {
	return c.textSelection
}

// SetTextSelection sets the selection of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.textSelection
func (c *SemanticsConfiguration) SetTextSelection(v services.TextSelection) {
	c.textSelection = foundation.Some(v)
	c.annotate()
}

// ScrollPosition is the scroll position of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollPosition
func (c *SemanticsConfiguration) ScrollPosition() foundation.Opt[float64] { return c.scrollPosition }

// SetScrollPosition sets the scroll position.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollPosition
func (c *SemanticsConfiguration) SetScrollPosition(v float64) {
	c.scrollPosition = foundation.Some(v)
	c.annotate()
}

// ScrollExtentMax is the highest scroll position of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollExtentMax
func (c *SemanticsConfiguration) ScrollExtentMax() foundation.Opt[float64] { return c.scrollExtentMax }

// SetScrollExtentMax sets the highest scroll position.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollExtentMax
func (c *SemanticsConfiguration) SetScrollExtentMax(v float64) {
	c.scrollExtentMax = foundation.Some(v)
	c.annotate()
}

// ScrollExtentMin is the lowest scroll position of a scrollable.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollExtentMin
func (c *SemanticsConfiguration) ScrollExtentMin() foundation.Opt[float64] { return c.scrollExtentMin }

// SetScrollExtentMin sets the lowest scroll position.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.scrollExtentMin
func (c *SemanticsConfiguration) SetScrollExtentMin(v float64) {
	c.scrollExtentMin = foundation.Some(v)
	c.annotate()
}

// ControlsNodes are the identifiers of the nodes that this one controls; nil
// if none.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.controlsNodes
func (c *SemanticsConfiguration) ControlsNodes() []string { return c.controlsNodes }

// SetControlsNodes sets the controlled nodes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.controlsNodes
func (c *SemanticsConfiguration) SetControlsNodes(v []string) {
	c.controlsNodes = v
	c.annotate()
}

// ValidationResult is the validation result.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.validationResult
func (c *SemanticsConfiguration) ValidationResult() ui.SemanticsValidationResult {
	return c.validationResult
}

// SetValidationResult sets the validation result.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.validationResult
func (c *SemanticsConfiguration) SetValidationResult(v ui.SemanticsValidationResult) {
	c.validationResult = v
	c.annotate()
}

// HitTestBehavior is the hit test behavior.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hitTestBehavior
func (c *SemanticsConfiguration) HitTestBehavior() ui.SemanticsHitTestBehavior {
	return c.hitTestBehavior
}

// SetHitTestBehavior sets the hit test behavior.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.hitTestBehavior
func (c *SemanticsConfiguration) SetHitTestBehavior(v ui.SemanticsHitTestBehavior) {
	c.hitTestBehavior = v
	c.annotate()
}

// InputType is the input type of a text field.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.inputType
func (c *SemanticsConfiguration) InputType() ui.SemanticsInputType { return c.inputType }

// SetInputType sets the input type.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.inputType
func (c *SemanticsConfiguration) SetInputType(v ui.SemanticsInputType) {
	c.inputType = v
	c.annotate()
}

// MaxValue is the highest value, as text.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.maxValue
func (c *SemanticsConfiguration) MaxValue() foundation.Opt[string] { return c.maxValue }

// SetMaxValue sets the highest value.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.maxValue
func (c *SemanticsConfiguration) SetMaxValue(v foundation.Opt[string]) {
	c.maxValue = v
	c.annotate()
}

// MinValue is the lowest value, as text.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.minValue
func (c *SemanticsConfiguration) MinValue() foundation.Opt[string] { return c.minValue }

// SetMinValue sets the lowest value.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.minValue
func (c *SemanticsConfiguration) SetMinValue(v foundation.Opt[string]) {
	c.minValue = v
	c.annotate()
}

// TagsForChildren are the tags that the node gives to its children.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.tagsForChildren
func (c *SemanticsConfiguration) TagsForChildren() map[SemanticsTag]struct{} {
	return c.tagsForChildren
}

// TagsChildrenWith is whether the children get the tag.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.tagsChildrenWith
func (c *SemanticsConfiguration) TagsChildrenWith(tag SemanticsTag) bool {
	_, ok := c.tagsForChildren[tag]
	return ok
}

// AddTagForChildren makes the children get the tag.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.addTagForChildren
func (c *SemanticsConfiguration) AddTagForChildren(tag SemanticsTag) {
	if c.tagsForChildren == nil {
		c.tagsForChildren = map[SemanticsTag]struct{}{}
	}
	c.tagsForChildren[tag] = struct{}{}
}

// hasExplicitRole is whether the node has a role of its own.
func (c *SemanticsConfiguration) hasExplicitRole() bool {
	f := c.flags
	return c.role != ui.SemanticsRoleNone || f.IsTextField || f.IsSlider || f.IsLink || f.ScopesRoute || f.IsImage || f.IsKeyboardKey
}

// IsCompatibleWith reports whether the configuration can merge with the other
// one without losing information; nil is compatible.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.isCompatibleWith
func (c *SemanticsConfiguration) IsCompatibleWith(other *SemanticsConfiguration) bool {
	if other == nil || !other.hasBeenAnnotated {
		return true
	}
	if c.traversalChild != other.traversalChild {
		return false
	}
	if !c.hasBeenAnnotated {
		return true
	}
	switch {
	case c.actionsAsBits&other.actionsAsBits != 0,
		c.flags.HasConflictingFlags(other.flags),
		c.platformViewID.IsSome() && other.platformViewID.IsSome(),
		c.maxValueLength.IsSome() && other.maxValueLength.IsSome(),
		c.currentValueLen.IsSome() && other.currentValueLen.IsSome(),
		c.attributedValue.String != "" && other.attributedValue.String != "",
		!optEqual(c.localeForSubtree, other.localeForSubtree),
		c.hasExplicitRole() && other.hasExplicitRole(),
		c.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer || other.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer,
		c.minValue.IsSome() && other.minValue.IsSome(),
		c.maxValue.IsSome() && other.maxValue.IsSome():
		return false
	}
	return true
}

func mergeHeadingLevels(source, target int) int {
	if target == 0 {
		return source
	}
	return target
}

func unionStrings(a, b []string) []string {
	out := slices.Clone(a)
	for _, s := range b {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// Absorb merges the configuration of a child into this one. In debug builds it
// panics with the tag SemanticsAbsorbWithExplicitChildren if the configuration
// has explicit child nodes.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.absorb
func (c *SemanticsConfiguration) Absorb(child *SemanticsConfiguration) {
	if foundation.DebugMode && c.ExplicitChildNodes {
		panic(errors.New("SemanticsAbsorbWithExplicitChildren", "a configuration with explicit child nodes can not absorb a child"))
	}
	if !child.hasBeenAnnotated {
		return
	}
	if child.IsBlockingUserActions {
		for key, value := range child.actions {
			if unblockedUserActions&key > 0 {
				c.actions[key] = value
			}
		}
	} else {
		maps.Copy(c.actions, child.actions)
	}
	c.actionsAsBits |= child.effectiveActionsAsBits()
	maps.Copy(c.customSemanticsActions, child.customSemanticsActions)
	c.flags = c.flags.Merge(child.flags)
	if c.linkURL.IsNone() {
		c.linkURL = child.linkURL
	}
	if c.textSelection.IsNone() {
		c.textSelection = child.textSelection
	}
	if c.scrollPosition.IsNone() {
		c.scrollPosition = child.scrollPosition
	}
	if c.scrollExtentMax.IsNone() {
		c.scrollExtentMax = child.scrollExtentMax
	}
	if c.scrollExtentMin.IsNone() {
		c.scrollExtentMin = child.scrollExtentMin
	}
	if c.hintOverrides == nil {
		c.hintOverrides = child.hintOverrides
	}
	for _, p := range []struct{ dst, src *foundation.Opt[int] }{
		{&c.indexInParent, &child.indexInParent}, {&c.scrollIndex, &child.scrollIndex},
		{&c.scrollChildCount, &child.scrollChildCount}, {&c.platformViewID, &child.platformViewID},
		{&c.maxValueLength, &child.maxValueLength}, {&c.currentValueLen, &child.currentValueLen},
	} {
		if p.dst.IsNone() {
			*p.dst = *p.src
		}
	}
	if c.traversalChild == nil && c.traversalParent == nil {
		c.traversalParent = child.traversalParent
	}
	if c.traversalChild == nil {
		c.traversalChild = child.traversalChild
	}
	c.headingLevel = mergeHeadingLevels(child.headingLevel, c.headingLevel)
	if c.textDirection == ui.TextDirectionUnspecified {
		c.textDirection = child.textDirection
	}
	if c.sortKey == nil {
		c.sortKey = child.sortKey
	}
	if c.identifier == "" {
		c.identifier = child.identifier
	}
	c.attributedLabel = concatAttributedString(c.attributedLabel, c.textDirection, child.attributedLabel, child.textDirection)
	if c.attributedValue.String == "" {
		c.attributedValue = child.attributedValue
	}
	if c.attributedIncreased.String == "" {
		c.attributedIncreased = child.attributedIncreased
	}
	if c.attributedDecreased.String == "" {
		c.attributedDecreased = child.attributedDecreased
	}
	if c.role == ui.SemanticsRoleNone {
		c.role = child.role
	}
	if c.inputType == ui.SemanticsInputTypeNone {
		c.inputType = child.inputType
	}
	c.attributedHint = concatAttributedString(c.attributedHint, c.textDirection, child.attributedHint, child.textDirection)
	if c.tooltip == "" {
		c.tooltip = child.tooltip
	}
	if c.controlsNodes == nil {
		c.controlsNodes = child.controlsNodes
	} else if child.controlsNodes != nil {
		c.controlsNodes = unionStrings(c.controlsNodes, child.controlsNodes)
	}
	if child.validationResult != c.validationResult {
		if child.validationResult == ui.SemanticsValidationResultInvalid {
			c.validationResult = ui.SemanticsValidationResultInvalid
		} else if c.validationResult == ui.SemanticsValidationResultNone {
			c.validationResult = child.validationResult
		}
	}
	c.accessibilityFocusBlockType = c.accessibilityFocusBlockType.merge(child.accessibilityFocusBlockType)
	if c.minValue.IsNone() {
		c.minValue = child.minValue
	}
	if c.maxValue.IsNone() {
		c.maxValue = child.maxValue
	}
	if c.hitTestBehavior == ui.SemanticsHitTestBehaviorDefer && child.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer {
		c.hitTestBehavior = child.hitTestBehavior
	}
	c.hasBeenAnnotated = c.hasBeenAnnotated || child.hasBeenAnnotated
}

// Copy returns a copy of the configuration. Like Flutter's, it keeps the
// actions with their handlers but not the getters of the handlers (OnTap and
// the like), the delegate, or the locales.
//
// Flutter: semantics/semantics.dart SemanticsConfiguration.copy
func (c *SemanticsConfiguration) Copy() *SemanticsConfiguration {
	n := NewSemanticsConfiguration()
	n.isSemanticBoundary = c.isSemanticBoundary
	n.ExplicitChildNodes = c.ExplicitChildNodes
	n.IsBlockingSemanticsOfPreviouslyPaintedNodes = c.IsBlockingSemanticsOfPreviouslyPaintedNodes
	n.hasBeenAnnotated = c.hasBeenAnnotated
	n.isMergingSemanticsOfDescendants = c.isMergingSemanticsOfDescendants
	n.textDirection = c.textDirection
	n.sortKey = c.sortKey
	n.identifier = c.identifier
	n.traversalParent, n.traversalChild = c.traversalParent, c.traversalChild
	n.attributedLabel, n.attributedIncreased, n.attributedValue = c.attributedLabel, c.attributedIncreased, c.attributedValue
	n.attributedDecreased, n.attributedHint = c.attributedDecreased, c.attributedHint
	n.accessibilityFocusBlockType = c.accessibilityFocusBlockType
	n.hintOverrides = c.hintOverrides
	n.tooltip = c.tooltip
	n.flags = c.flags
	n.tagsForChildren = c.tagsForChildren
	n.textSelection = c.textSelection
	n.scrollPosition, n.scrollExtentMax, n.scrollExtentMin = c.scrollPosition, c.scrollExtentMax, c.scrollExtentMin
	n.actionsAsBits = c.actionsAsBits
	n.indexInParent = c.indexInParent
	n.scrollIndex, n.scrollChildCount = c.scrollIndex, c.scrollChildCount
	n.platformViewID = c.platformViewID
	n.maxValueLength, n.currentValueLen = c.maxValueLength, c.currentValueLen
	maps.Copy(n.actions, c.actions)
	maps.Copy(n.customSemanticsActions, c.customSemanticsActions)
	n.IsBlockingUserActions = c.IsBlockingUserActions
	n.headingLevel = c.headingLevel
	n.linkURL = c.linkURL
	n.role = c.role
	n.controlsNodes = c.controlsNodes
	n.validationResult = c.validationResult
	n.inputType = c.inputType
	n.hitTestBehavior = c.hitTestBehavior
	n.minValue, n.maxValue = c.minValue, c.maxValue
	return n
}
