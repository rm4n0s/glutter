// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsAnnotationsOpts are the optional parameters of
// InitSemanticsAnnotations.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.initSemanticsAnnotations
type SemanticsAnnotationsOpts struct {
	// Container makes the object a semantic boundary: its subtree forms a node
	// of its own.
	Container bool
	// ExplicitChildNodes makes the children form nodes of their own instead of
	// merging into this one.
	ExplicitChildNodes bool
	// ExcludeSemantics drops the semantics of the subtree.
	ExcludeSemantics bool
	// BlockUserActions blocks the user actions of the subtree.
	BlockUserActions bool
	// LocaleForSubtree is the locale of the strings of the subtree.
	LocaleForSubtree foundation.Opt[ui.Locale]
	// TextDirection is the direction of the labels; Unspecified for none.
	TextDirection ui.TextDirection
}

// SemanticsAnnotations is the state of the render objects that add annotations
// to the semantics node of their subtree, such as RenderSemanticsAnnotations.
// Embed it, and call InitSemanticsAnnotations from the constructor.
//
// Deviation: Flutter's is a mixin on RenderObject (SPEC §12.10).
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin
type SemanticsAnnotations struct {
	self RenderObject

	properties         *semantics.SemanticsProperties
	container          bool
	explicitChildNodes bool
	excludeSemantics   bool
	blockUserActions   bool
	localeForSubtree   foundation.Opt[ui.Locale]
	textDirection      ui.TextDirection

	attributedLabel, attributedValue, attributedIncreased, attributedDecreased, attributedHint *semantics.AttributedString
}

// InitSemanticsAnnotations sets the outer object and the annotations.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.initSemanticsAnnotations
func InitSemanticsAnnotations(a *SemanticsAnnotations, self RenderObject, properties *semantics.SemanticsProperties, opts ...SemanticsAnnotationsOpts) {
	o := firstOpts(opts)
	a.self = self
	a.properties = properties
	a.container = o.Container
	a.explicitChildNodes = o.ExplicitChildNodes
	a.excludeSemantics = o.ExcludeSemantics
	a.blockUserActions = o.BlockUserActions
	a.localeForSubtree = o.LocaleForSubtree
	a.textDirection = o.TextDirection
	a.updateAttributedFields(properties)
}

// Properties are all the semantics properties of the object.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.properties
func (a *SemanticsAnnotations) Properties() *semantics.SemanticsProperties { return a.properties }

// SetProperties replaces the properties; the semantics are updated if they
// are not the same object.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.properties
func (a *SemanticsAnnotations) SetProperties(value *semantics.SemanticsProperties) {
	if a.properties == value {
		return
	}
	a.properties = value
	a.updateAttributedFields(value)
	a.self.MarkNeedsSemanticsUpdate()
}

// Container is whether the object forms a semantic boundary.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.container
func (a *SemanticsAnnotations) Container() bool { return a.container }

// SetContainer changes the flag.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.container
func (a *SemanticsAnnotations) SetContainer(value bool) {
	if a.container == value {
		return
	}
	a.container = value
	a.self.MarkNeedsSemanticsUpdate()
}

// ExplicitChildNodes is whether the children form nodes of their own.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.explicitChildNodes
func (a *SemanticsAnnotations) ExplicitChildNodes() bool { return a.explicitChildNodes }

// SetExplicitChildNodes changes the flag.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.explicitChildNodes
func (a *SemanticsAnnotations) SetExplicitChildNodes(value bool) {
	if a.explicitChildNodes == value {
		return
	}
	a.explicitChildNodes = value
	a.self.MarkNeedsSemanticsUpdate()
}

// ExcludeSemantics is whether the semantics of the subtree are dropped.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.excludeSemantics
func (a *SemanticsAnnotations) ExcludeSemantics() bool { return a.excludeSemantics }

// SetExcludeSemantics changes the flag.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.excludeSemantics
func (a *SemanticsAnnotations) SetExcludeSemantics(value bool) {
	if a.excludeSemantics == value {
		return
	}
	a.excludeSemantics = value
	a.self.MarkNeedsSemanticsUpdate()
}

// BlockUserActions is whether the user actions of the subtree are blocked.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.blockUserActions
func (a *SemanticsAnnotations) BlockUserActions() bool { return a.blockUserActions }

// SetBlockUserActions changes the flag.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.blockUserActions
func (a *SemanticsAnnotations) SetBlockUserActions(value bool) {
	if a.blockUserActions == value {
		return
	}
	a.blockUserActions = value
	a.self.MarkNeedsSemanticsUpdate()
}

// LocaleForSubtree is the locale of the strings of the subtree.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.localeForSubtree
func (a *SemanticsAnnotations) LocaleForSubtree() foundation.Opt[ui.Locale] {
	return a.localeForSubtree
}

// SetLocaleForSubtree changes the locale.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.localeForSubtree
func (a *SemanticsAnnotations) SetLocaleForSubtree(value foundation.Opt[ui.Locale]) {
	if a.localeForSubtree == value {
		return
	}
	a.localeForSubtree = value
	a.self.MarkNeedsSemanticsUpdate()
}

// TextDirection is the direction of the labels; Unspecified for none.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.textDirection
func (a *SemanticsAnnotations) TextDirection() ui.TextDirection { return a.textDirection }

// SetTextDirection changes the direction.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.textDirection
func (a *SemanticsAnnotations) SetTextDirection(value ui.TextDirection) {
	if a.textDirection == value {
		return
	}
	a.textDirection = value
	a.self.MarkNeedsSemanticsUpdate()
}

func (a *SemanticsAnnotations) updateAttributedFields(p *semantics.SemanticsProperties) {
	effective := func(attr *semantics.AttributedString, s foundation.Opt[string]) *semantics.AttributedString {
		if attr != nil {
			return attr
		}
		if v, ok := s.Get(); ok {
			as := semantics.NewAttributedString(v)
			return &as
		}
		return nil
	}
	a.attributedLabel = effective(p.AttributedLabel, p.Label)
	a.attributedValue = effective(p.AttributedValue, p.Value)
	a.attributedIncreased = effective(p.AttributedIncreasedValue, p.IncreasedValue)
	a.attributedDecreased = effective(p.AttributedDecreasedValue, p.DecreasedValue)
	a.attributedHint = effective(p.AttributedHint, p.Hint)
}

// VisitChildrenForSemantics visits no child if the semantics are excluded.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.visitChildrenForSemantics
func (a *SemanticsAnnotations) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if a.excludeSemantics {
		return
	}
	a.self.AsRenderObject().VisitChildrenForSemantics(visitor)
}

// DescribeSemanticsConfiguration copies the properties into the configuration.
//
// Flutter: rendering/object.dart SemanticsAnnotationsMixin.describeSemanticsConfiguration
func (a *SemanticsAnnotations) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	p := a.properties
	config.SetIsSemanticBoundary(a.container || p.Identifier.IsSome())
	config.ExplicitChildNodes = a.explicitChildNodes
	config.IsBlockingUserActions = a.blockUserActions
	if l, ok := a.localeForSubtree.Get(); ok {
		config.SetLocaleForSubtree(l)
	}
	if foundation.DebugMode {
		if p.ScopesRoute.OrElse(false) && !a.explicitChildNodes {
			panic(errors.New("SemanticsScopesRouteNeedsExplicitChildNodes", "explicitChildNodes must be set to true if scopes route is true"))
		}
		if p.Toggled.OrElse(false) && p.Checked.OrElse(false) {
			panic(errors.New("SemanticsToggledAndChecked", "a semantics node cannot be toggled and checked at the same time"))
		}
	}
	setBool := func(v foundation.Opt[bool], set func(bool)) {
		if b, ok := v.Get(); ok {
			set(b)
		}
	}
	setOpt := func(v foundation.Opt[bool], set func(foundation.Opt[bool])) {
		if v.IsSome() {
			set(v)
		}
	}
	setOpt(p.Enabled, config.SetIsEnabled)
	setOpt(p.Checked, config.SetIsChecked)
	setOpt(p.Mixed, config.SetIsCheckStateMixed)
	setOpt(p.Toggled, config.SetIsToggled)
	setBool(p.Selected, config.SetIsSelected)
	setBool(p.Button, config.SetIsButton)
	setOpt(p.Expanded, config.SetIsExpanded)
	setBool(p.Link, config.SetIsLink)
	if p.LinkURL.IsSome() {
		config.SetLinkURL(p.LinkURL)
	}
	setBool(p.Slider, config.SetIsSlider)
	setBool(p.KeyboardKey, config.SetIsKeyboardKey)
	setBool(p.Header, config.SetIsHeader)
	if l, ok := p.HeadingLevel.Get(); ok {
		config.SetHeadingLevel(l)
	}
	setBool(p.TextField, config.SetIsTextField)
	setBool(p.ReadOnly, config.SetIsReadOnly)
	setBool(p.Focusable, config.SetIsFocusable)
	setOpt(p.Focused, config.SetIsFocused)
	if t, ok := p.AccessibilityFocusBlockType.Get(); ok {
		config.SetAccessibilityFocusBlockType(t)
	}
	setBool(p.InMutuallyExclusiveGroup, config.SetIsInMutuallyExclusiveGroup)
	setBool(p.Obscured, config.SetIsObscured)
	setBool(p.Multiline, config.SetIsMultiline)
	setBool(p.Hidden, config.SetIsHidden)
	setBool(p.Image, config.SetIsImage)
	setOpt(p.IsRequired, config.SetIsRequired)
	if id, ok := p.Identifier.Get(); ok {
		config.SetIdentifier(id)
	}
	if p.TraversalParentIdentifier != nil {
		config.SetTraversalParentIdentifier(p.TraversalParentIdentifier)
	}
	if p.TraversalChildIdentifier != nil {
		config.SetTraversalChildIdentifier(p.TraversalChildIdentifier)
	}
	if a.attributedLabel != nil {
		config.SetAttributedLabel(*a.attributedLabel)
	}
	if a.attributedValue != nil {
		config.SetAttributedValue(*a.attributedValue)
	}
	if a.attributedIncreased != nil {
		config.SetAttributedIncreasedValue(*a.attributedIncreased)
	}
	if a.attributedDecreased != nil {
		config.SetAttributedDecreasedValue(*a.attributedDecreased)
	}
	if a.attributedHint != nil {
		config.SetAttributedHint(*a.attributedHint)
	}
	if t, ok := p.Tooltip.Get(); ok {
		config.SetTooltip(t)
	}
	if p.HintOverrides != nil && p.HintOverrides.IsNotEmpty() {
		config.SetHintOverrides(p.HintOverrides)
	}
	setBool(p.ScopesRoute, config.SetScopesRoute)
	setBool(p.NamesRoute, config.SetNamesRoute)
	setBool(p.LiveRegion, config.SetLiveRegion)
	if p.MaxValueLength.IsSome() {
		config.SetMaxValueLength(p.MaxValueLength)
	}
	if p.CurrentValueLength.IsSome() {
		config.SetCurrentValueLength(p.CurrentValueLength)
	}
	if a.textDirection != ui.TextDirectionUnspecified {
		config.SetTextDirection(a.textDirection)
	}
	if p.SortKey != nil {
		config.SetSortKey(p.SortKey)
	}
	if tag, ok := p.TagForChildren.Get(); ok {
		config.AddTagForChildren(tag)
	}
	if r, ok := p.Role.Get(); ok {
		config.SetRole(r)
	}
	if p.ControlsNodes != nil {
		config.SetControlsNodes(p.ControlsNodes)
	}
	if config.ValidationResult() != p.ValidationResult {
		config.SetValidationResult(p.ValidationResult)
	}
	if b, ok := p.HitTestBehavior.Get(); ok {
		config.SetHitTestBehavior(b)
	}
	if t, ok := p.InputType.Get(); ok {
		config.SetInputType(t)
	}
	if p.MinValue.IsSome() {
		config.SetMinValue(p.MinValue)
	}
	if p.MaxValue.IsSome() {
		config.SetMaxValue(p.MaxValue)
	}

	// The configuration gets the handlers of this object instead of the ones
	// that the user provided, so that changing a handler from one non-nil value
	// to another does not need a semantics update.
	void := func(has bool, set func(semantics.VoidCallback), call func(*semantics.SemanticsProperties) semantics.VoidCallback) {
		if has {
			set(func() {
				if cb := call(a.properties); cb != nil {
					cb()
				}
			})
		}
	}
	void(p.OnTap != nil, config.SetOnTap, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnTap })
	void(p.OnLongPress != nil, config.SetOnLongPress, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnLongPress })
	void(p.OnDismiss != nil, config.SetOnDismiss, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnDismiss })
	void(p.OnScrollLeft != nil, config.SetOnScrollLeft, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnScrollLeft })
	void(p.OnScrollRight != nil, config.SetOnScrollRight, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnScrollRight })
	void(p.OnScrollUp != nil, config.SetOnScrollUp, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnScrollUp })
	void(p.OnScrollDown != nil, config.SetOnScrollDown, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnScrollDown })
	void(p.OnIncrease != nil, config.SetOnIncrease, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnIncrease })
	void(p.OnDecrease != nil, config.SetOnDecrease, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnDecrease })
	void(p.OnCopy != nil, config.SetOnCopy, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnCopy })
	void(p.OnCut != nil, config.SetOnCut, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnCut })
	void(p.OnPaste != nil, config.SetOnPaste, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnPaste })
	void(p.OnDidGainAccessibilityFocus != nil, config.SetOnDidGainAccessibilityFocus,
		func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnDidGainAccessibilityFocus })
	void(p.OnDidLoseAccessibilityFocus != nil, config.SetOnDidLoseAccessibilityFocus,
		func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnDidLoseAccessibilityFocus })
	void(p.OnFocus != nil, config.SetOnFocus, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnFocus })
	void(p.OnExpand != nil, config.SetOnExpand, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnExpand })
	void(p.OnCollapse != nil, config.SetOnCollapse, func(p *semantics.SemanticsProperties) semantics.VoidCallback { return p.OnCollapse })

	cursor := func(has bool, set func(semantics.MoveCursorHandler), call func(*semantics.SemanticsProperties) semantics.MoveCursorHandler) {
		if has {
			set(func(extend bool) {
				if cb := call(a.properties); cb != nil {
					cb(extend)
				}
			})
		}
	}
	cursor(p.OnMoveCursorForwardByCharacter != nil, config.SetOnMoveCursorForwardByCharacter,
		func(p *semantics.SemanticsProperties) semantics.MoveCursorHandler {
			return p.OnMoveCursorForwardByCharacter
		})
	cursor(p.OnMoveCursorBackwardByCharacter != nil, config.SetOnMoveCursorBackwardByCharacter,
		func(p *semantics.SemanticsProperties) semantics.MoveCursorHandler {
			return p.OnMoveCursorBackwardByCharacter
		})
	cursor(p.OnMoveCursorForwardByWord != nil, config.SetOnMoveCursorForwardByWord,
		func(p *semantics.SemanticsProperties) semantics.MoveCursorHandler { return p.OnMoveCursorForwardByWord })
	cursor(p.OnMoveCursorBackwardByWord != nil, config.SetOnMoveCursorBackwardByWord,
		func(p *semantics.SemanticsProperties) semantics.MoveCursorHandler {
			return p.OnMoveCursorBackwardByWord
		})
	if p.OnSetSelection != nil {
		config.SetOnSetSelection(func(sel services.TextSelection) {
			if cb := a.properties.OnSetSelection; cb != nil {
				cb(sel)
			}
		})
	}
	if p.OnSetText != nil {
		config.SetOnSetText(func(text string) {
			if cb := a.properties.OnSetText; cb != nil {
				cb(text)
			}
		})
	}
	if p.CustomSemanticsActions != nil {
		config.SetCustomSemanticsActions(p.CustomSemanticsActions)
	}
}

// RenderSemanticsGestureHandler listens for the gestures that the semantics
// server (an accessibility tool) asks for: tap, long press and scrolls.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler
type RenderSemanticsGestureHandler struct {
	RenderProxyBoxWithHitTestBehavior
	validActions           map[ui.SemanticsAction]struct{}
	onTap                  gestures.GestureTapCallback
	onLongPress            gestures.GestureLongPressCallback
	onHorizontalDragUpdate gestures.GestureDragUpdateCallback
	onVerticalDragUpdate   gestures.GestureDragUpdateCallback
	// ScrollFactor is the fraction of the dimension of the box to use when
	// scrolling: with 0.8, a left-scroll action of the accessibility system
	// drags a 200 pixel wide box 160 pixels to the left. The default is 0.8.
	//
	// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.scrollFactor
	ScrollFactor float64
}

// RenderSemanticsGestureHandlerOpts are the optional parameters of
// NewRenderSemanticsGestureHandler.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.new
type RenderSemanticsGestureHandlerOpts struct {
	// OnTap is called when the user taps on the object.
	OnTap gestures.GestureTapCallback
	// OnLongPress is called when the user long presses the object.
	OnLongPress gestures.GestureLongPressCallback
	// OnHorizontalDragUpdate is called when the user scrolls left or right.
	OnHorizontalDragUpdate gestures.GestureDragUpdateCallback
	// OnVerticalDragUpdate is called when the user scrolls up or down.
	OnVerticalDragUpdate gestures.GestureDragUpdateCallback
	// ScrollFactor defaults to 0.8.
	ScrollFactor foundation.Opt[float64]
	// Behavior defaults to HitTestBehaviorDeferToChild.
	Behavior HitTestBehavior
}

// NewRenderSemanticsGestureHandler creates the box.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.new
func NewRenderSemanticsGestureHandler(child Box, opts ...RenderSemanticsGestureHandlerOpts) *RenderSemanticsGestureHandler {
	o := firstOpts(opts)
	r := &RenderSemanticsGestureHandler{
		onTap: o.OnTap, onLongPress: o.OnLongPress, onHorizontalDragUpdate: o.OnHorizontalDragUpdate,
		onVerticalDragUpdate: o.OnVerticalDragUpdate, ScrollFactor: o.ScrollFactor.OrElse(0.8),
	}
	InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, o.Behavior, child)
	return r
}

// ValidActions is the set of actions to allow, or nil for all: the others are
// omitted even if their callback is set.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.validActions
func (r *RenderSemanticsGestureHandler) ValidActions() map[ui.SemanticsAction]struct{} {
	return r.validActions
}

// SetValidActions changes the allowed actions; the semantics are updated if
// the set differs.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.validActions
func (r *RenderSemanticsGestureHandler) SetValidActions(value map[ui.SemanticsAction]struct{}) {
	if (value == nil) == (r.validActions == nil) && len(value) == len(r.validActions) {
		same := true
		for a := range value {
			if _, ok := r.validActions[a]; !ok {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	r.validActions = value
	r.MarkNeedsSemanticsUpdate()
}

// OnTap is called when the user taps on the object.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onTap
func (r *RenderSemanticsGestureHandler) OnTap() gestures.GestureTapCallback { return r.onTap }

// SetOnTap changes the callback; the semantics are updated if it becomes nil or
// stops being nil.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onTap
func (r *RenderSemanticsGestureHandler) SetOnTap(value gestures.GestureTapCallback) {
	had := r.onTap != nil
	r.onTap = value
	if (value != nil) != had {
		r.MarkNeedsSemanticsUpdate()
	}
}

// OnLongPress is called when the user presses on the object for a long time.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onLongPress
func (r *RenderSemanticsGestureHandler) OnLongPress() gestures.GestureLongPressCallback {
	return r.onLongPress
}

// SetOnLongPress changes the callback.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onLongPress
func (r *RenderSemanticsGestureHandler) SetOnLongPress(value gestures.GestureLongPressCallback) {
	had := r.onLongPress != nil
	r.onLongPress = value
	if (value != nil) != had {
		r.MarkNeedsSemanticsUpdate()
	}
}

// OnHorizontalDragUpdate is called when the user scrolls left or right.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onHorizontalDragUpdate
func (r *RenderSemanticsGestureHandler) OnHorizontalDragUpdate() gestures.GestureDragUpdateCallback {
	return r.onHorizontalDragUpdate
}

// SetOnHorizontalDragUpdate changes the callback.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onHorizontalDragUpdate
func (r *RenderSemanticsGestureHandler) SetOnHorizontalDragUpdate(value gestures.GestureDragUpdateCallback) {
	had := r.onHorizontalDragUpdate != nil
	r.onHorizontalDragUpdate = value
	if (value != nil) != had {
		r.MarkNeedsSemanticsUpdate()
	}
}

// OnVerticalDragUpdate is called when the user scrolls up or down.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onVerticalDragUpdate
func (r *RenderSemanticsGestureHandler) OnVerticalDragUpdate() gestures.GestureDragUpdateCallback {
	return r.onVerticalDragUpdate
}

// SetOnVerticalDragUpdate changes the callback.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.onVerticalDragUpdate
func (r *RenderSemanticsGestureHandler) SetOnVerticalDragUpdate(value gestures.GestureDragUpdateCallback) {
	had := r.onVerticalDragUpdate != nil
	r.onVerticalDragUpdate = value
	if (value != nil) != had {
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *RenderSemanticsGestureHandler) isValidAction(a ui.SemanticsAction) bool {
	if r.validActions == nil {
		return true
	}
	_, ok := r.validActions[a]
	return ok
}

// DescribeSemanticsConfiguration adds the handlers of the callbacks that are
// set and valid.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.describeSemanticsConfiguration
func (r *RenderSemanticsGestureHandler) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBoxWithHitTestBehavior.DescribeSemanticsConfiguration(config)
	if r.onTap != nil && r.isValidAction(ui.SemanticsActionTap) {
		config.SetOnTap(semantics.VoidCallback(r.onTap))
	}
	if r.onLongPress != nil && r.isValidAction(ui.SemanticsActionLongPress) {
		config.SetOnLongPress(semantics.VoidCallback(r.onLongPress))
	}
	if r.onHorizontalDragUpdate != nil {
		if r.isValidAction(ui.SemanticsActionScrollRight) {
			config.SetOnScrollRight(func() { r.performScroll(true, r.Size().Width*r.ScrollFactor, r.onHorizontalDragUpdate) })
		}
		if r.isValidAction(ui.SemanticsActionScrollLeft) {
			config.SetOnScrollLeft(func() { r.performScroll(true, r.Size().Width*-r.ScrollFactor, r.onHorizontalDragUpdate) })
		}
	}
	if r.onVerticalDragUpdate != nil {
		if r.isValidAction(ui.SemanticsActionScrollUp) {
			config.SetOnScrollUp(func() { r.performScroll(false, r.Size().Height*-r.ScrollFactor, r.onVerticalDragUpdate) })
		}
		if r.isValidAction(ui.SemanticsActionScrollDown) {
			config.SetOnScrollDown(func() { r.performScroll(false, r.Size().Height*r.ScrollFactor, r.onVerticalDragUpdate) })
		}
	}
}

func (r *RenderSemanticsGestureHandler) performScroll(horizontal bool, primaryDelta float64, cb gestures.GestureDragUpdateCallback) {
	if cb == nil {
		return
	}
	delta := ui.Offset{Dy: primaryDelta}
	if horizontal {
		delta = ui.Offset{Dx: primaryDelta}
	}
	cb(gestures.NewDragUpdateDetails(
		r.LocalToGlobal(r.Size().Center(ui.OffsetZero), nil),
		gestures.DragUpdateOpts{Delta: delta, PrimaryDelta: foundation.Some(primaryDelta)}))
}

// DebugFillProperties adds the gestures that are handled.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsGestureHandler.debugFillProperties
func (r *RenderSemanticsGestureHandler) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBoxWithHitTestBehavior.DebugFillProperties(b)
	var gs []string
	if r.onTap != nil {
		gs = append(gs, "tap")
	}
	if r.onLongPress != nil {
		gs = append(gs, "long press")
	}
	if r.onHorizontalDragUpdate != nil {
		gs = append(gs, "horizontal scroll")
	}
	if r.onVerticalDragUpdate != nil {
		gs = append(gs, "vertical scroll")
	}
	if len(gs) == 0 {
		gs = append(gs, "<none>")
	}
	b.Add(foundation.NewIterableProperty("gestures", gs))
}

// RenderSemanticsAnnotations adds annotations to the semantics node of its
// subtree.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsAnnotations
type RenderSemanticsAnnotations struct {
	RenderProxyBox
	SemanticsAnnotations
}

// NewRenderSemanticsAnnotations creates the box. If the properties have an
// attributed label, a text direction should be given.
//
// Flutter: rendering/proxy_box.dart RenderSemanticsAnnotations.new
func NewRenderSemanticsAnnotations(child Box, properties *semantics.SemanticsProperties, opts ...SemanticsAnnotationsOpts) *RenderSemanticsAnnotations {
	r := &RenderSemanticsAnnotations{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	InitSemanticsAnnotations(&r.SemanticsAnnotations, r, properties, opts...)
	return r
}

// RenderBlockSemantics drops the semantics of all the earlier render objects
// below the same semantic boundary.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics
type RenderBlockSemantics struct {
	RenderProxyBox
	blocking bool
}

// RenderBlockSemanticsOpts are the optional parameters of NewRenderBlockSemantics.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.new
type RenderBlockSemanticsOpts struct {
	// Blocking defaults to true.
	Blocking foundation.Opt[bool]
}

// NewRenderBlockSemantics creates the box.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.new
func NewRenderBlockSemantics(child Box, opts ...RenderBlockSemanticsOpts) *RenderBlockSemantics {
	r := &RenderBlockSemantics{blocking: firstOpts(opts).Blocking.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Blocking is whether the object drops the semantics of the render objects
// painted before it, below a common semantic boundary.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.blocking
func (r *RenderBlockSemantics) Blocking() bool { return r.blocking }

// SetBlocking changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.blocking
func (r *RenderBlockSemantics) SetBlocking(value bool) {
	if value == r.blocking {
		return
	}
	r.blocking = value
	r.MarkNeedsSemanticsUpdate()
}

// DescribeSemanticsConfiguration sets the blocking flag.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.describeSemanticsConfiguration
func (r *RenderBlockSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.IsBlockingSemanticsOfPreviouslyPaintedNodes = r.blocking
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_box.dart RenderBlockSemantics.debugFillProperties
func (r *RenderBlockSemantics) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("blocking", r.blocking))
}

// RenderMergeSemantics merges the semantics of all its descendants into its
// node, so that the whole subtree becomes one leaf of the semantics tree.
//
// Flutter: rendering/proxy_box.dart RenderMergeSemantics
type RenderMergeSemantics struct {
	RenderProxyBox
}

// NewRenderMergeSemantics creates the box.
//
// Flutter: rendering/proxy_box.dart RenderMergeSemantics.new
func NewRenderMergeSemantics(child Box) *RenderMergeSemantics {
	r := &RenderMergeSemantics{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// DescribeSemanticsConfiguration makes the object a boundary that merges its
// descendants.
//
// Flutter: rendering/proxy_box.dart RenderMergeSemantics.describeSemanticsConfiguration
func (r *RenderMergeSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.SetIsMergingSemanticsOfDescendants(true)
}

// RenderExcludeSemantics excludes its subtree from the semantics tree when
// excluding.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics
type RenderExcludeSemantics struct {
	RenderProxyBox
	excluding bool
}

// RenderExcludeSemanticsOpts are the optional parameters of
// NewRenderExcludeSemantics.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.new
type RenderExcludeSemanticsOpts struct {
	// Excluding defaults to true.
	Excluding foundation.Opt[bool]
}

// NewRenderExcludeSemantics creates the box.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.new
func NewRenderExcludeSemantics(child Box, opts ...RenderExcludeSemanticsOpts) *RenderExcludeSemantics {
	r := &RenderExcludeSemantics{excluding: firstOpts(opts).Excluding.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Excluding is whether the subtree is excluded from the semantics tree.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.excluding
func (r *RenderExcludeSemantics) Excluding() bool { return r.excluding }

// SetExcluding changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.excluding
func (r *RenderExcludeSemantics) SetExcluding(value bool) {
	if value == r.excluding {
		return
	}
	r.excluding = value
	r.MarkNeedsSemanticsUpdate()
}

// VisitChildrenForSemantics visits no child when excluding.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.visitChildrenForSemantics
func (r *RenderExcludeSemantics) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.excluding {
		return
	}
	r.RenderProxyBox.VisitChildrenForSemantics(visitor)
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_box.dart RenderExcludeSemantics.debugFillProperties
func (r *RenderExcludeSemantics) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("excluding", r.excluding))
}

// RenderIndexedSemantics annotates the semantics of its child with an index.
// Widgets such as scroll views use the index of the first visible child to set
// the scroll index.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics
type RenderIndexedSemantics struct {
	RenderProxyBox
	index int
}

// NewRenderIndexedSemantics creates the box.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics.new
func NewRenderIndexedSemantics(child Box, index int) *RenderIndexedSemantics {
	r := &RenderIndexedSemantics{index: index}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Index is the index of the child.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics.index
func (r *RenderIndexedSemantics) Index() int { return r.index }

// SetIndex changes the index.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics.index
func (r *RenderIndexedSemantics) SetIndex(value int) {
	if value == r.index {
		return
	}
	r.index = value
	r.MarkNeedsSemanticsUpdate()
}

// DescribeSemanticsConfiguration sets the index in the parent.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics.describeSemanticsConfiguration
func (r *RenderIndexedSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIndexInParent(foundation.Some(r.index))
}

// DebugFillProperties adds the index.
//
// Flutter: rendering/proxy_box.dart RenderIndexedSemantics.debugFillProperties
func (r *RenderIndexedSemantics) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("index", r.index))
}
