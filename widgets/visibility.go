// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// VisibilityOpts are the optional parameters of NewVisibility.
//
// Flutter: widgets/indexed_stack.dart Visibility.new
type VisibilityOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Replacement is the widget to use when the child is not visible and its
	// state is not maintained; the default is a SizedBox.shrink.
	Replacement Widget
	// Visible is whether the child is visible; the default is true.
	Visible foundation.Opt[bool]
	// MaintainState is whether to maintain the state of the child when it is not
	// visible.
	MaintainState bool
	// MaintainAnimation is whether to keep the animations of the child running
	// when it is not visible; it needs MaintainState.
	MaintainAnimation bool
	// MaintainSize is whether to keep the space of the child when it is not
	// visible; it needs MaintainAnimation.
	MaintainSize bool
	// MaintainSemantics is whether to keep the semantics of the child when it
	// is not visible; it needs MaintainSize.
	MaintainSemantics bool
	// MaintainInteractivity is whether the child still takes pointers when it is
	// not visible; it needs MaintainSize.
	MaintainInteractivity bool
	// MaintainFocusability is whether the child can still take the focus when it
	// is not visible; it needs MaintainState.
	MaintainFocusability bool
}

// Visibility controls whether its child is included in the subtree or not.
//
// Flutter: widgets/indexed_stack.dart Visibility
type Visibility struct {
	StatelessWidget
	child                                                                                                          Widget
	replacement                                                                                                    Widget
	visible                                                                                                        bool
	maintainState, maintainAnimation, maintainSize, maintainSemantics, maintainInteractivity, maintainFocusability bool
}

// NewVisibility creates a widget that shows or hides its child. In debug builds
// it panics with the tag VisibilityMaintainInvalid when a Maintain flag is set
// without the one it needs.
//
// Flutter: widgets/indexed_stack.dart Visibility.new
func NewVisibility(child Widget, opts ...VisibilityOpts) *Visibility {
	o := firstOpts(opts)
	if foundation.DebugMode {
		switch {
		case o.MaintainAnimation && !o.MaintainState:
			panic(visibilityInvalid("cannot maintain animations if the state is not also maintained"))
		case o.MaintainSize && !o.MaintainAnimation:
			panic(visibilityInvalid("cannot maintain size if animations are not maintained"))
		case o.MaintainSemantics && !o.MaintainSize:
			panic(visibilityInvalid("cannot maintain semantics if size is not maintained"))
		case o.MaintainInteractivity && !o.MaintainSize:
			panic(visibilityInvalid("cannot maintain interactivity if size is not maintained"))
		case o.MaintainFocusability && !o.MaintainState:
			panic(visibilityInvalid("cannot maintain focusability if the state is not also maintained"))
		}
	}
	if o.Replacement == nil {
		o.Replacement = NewSizedBoxShrink(nil)
	}
	return &Visibility{StatelessWidget: NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child: child, replacement: o.Replacement,
		visible: o.Visible.OrElse(true), maintainState: o.MaintainState, maintainAnimation: o.MaintainAnimation, maintainSize: o.MaintainSize,
		maintainSemantics: o.MaintainSemantics, maintainInteractivity: o.MaintainInteractivity, maintainFocusability: o.MaintainFocusability}
}

func visibilityInvalid(msg string) error {
	return errors.New("VisibilityMaintainInvalid", msg)
}

// NewVisibilityMaintain creates a Visibility that keeps everything of the child
// when it is hidden: its state, animations, size, semantics, interactivity and
// focusability.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintain
func NewVisibilityMaintain(child Widget, opts ...VisibilityOpts) *Visibility {
	o := firstOpts(opts)
	o.MaintainState, o.MaintainAnimation, o.MaintainSize = true, true, true
	o.MaintainSemantics, o.MaintainInteractivity, o.MaintainFocusability = true, true, true
	return NewVisibility(child, o)
}

// Child is the widget to show or hide.
//
// Flutter: widgets/indexed_stack.dart Visibility.child
func (w *Visibility) Child() Widget { return w.child }

// Replacement is the widget that takes the place of the child when it is
// hidden and its state is not maintained.
//
// Flutter: widgets/indexed_stack.dart Visibility.replacement
func (w *Visibility) Replacement() Widget { return w.replacement }

// Visible is whether the child is visible.
//
// Flutter: widgets/indexed_stack.dart Visibility.visible
func (w *Visibility) Visible() bool { return w.visible }

// MaintainState is whether the state of the child is kept when it is hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainState
func (w *Visibility) MaintainState() bool { return w.maintainState }

// MaintainAnimation is whether the animations of the child keep running when it
// is hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainAnimation
func (w *Visibility) MaintainAnimation() bool { return w.maintainAnimation }

// MaintainSize is whether the space of the child is kept when it is hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainSize
func (w *Visibility) MaintainSize() bool { return w.maintainSize }

// MaintainSemantics is whether the semantics of the child are kept when it is
// hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainSemantics
func (w *Visibility) MaintainSemantics() bool { return w.maintainSemantics }

// MaintainInteractivity is whether the child takes pointers when it is hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainInteractivity
func (w *Visibility) MaintainInteractivity() bool { return w.maintainInteractivity }

// MaintainFocusability is whether the child can take the focus when it is
// hidden.
//
// Flutter: widgets/indexed_stack.dart Visibility.maintainFocusability
func (w *Visibility) MaintainFocusability() bool { return w.maintainFocusability }

// Build builds the child for the current configuration.
//
// Flutter: widgets/indexed_stack.dart Visibility.build
func (w *Visibility) Build(BuildContext) Widget {
	var result Widget = NewExcludeFocus(w.child, ExcludeFocusOpts{NotExcluding: w.visible || w.maintainFocusability})
	switch {
	case w.maintainSize:
		result = newRenderVisibility(w.visible, w.maintainSemantics,
			NewIgnorePointer(result, IgnorePointerOpts{Ignoring: foundation.Some(!w.visible && !w.maintainInteractivity)}))
	case w.maintainState:
		if !w.maintainAnimation {
			result = NewTickerMode(w.visible, result)
		}
		result = NewOffstage(result, OffstageOpts{Offstage: foundation.Some(!w.visible)})
	case w.visible:
		result = w.child
	default:
		result = w.replacement
	}
	return newVisibilityScope(w.visible, result)
}

// VisibilityOf is whether the closest Visibility or IndexedStack above the
// context shows it: false if any of the Visibility widgets above hides it. The
// context depends on all of them.
//
// Flutter: widgets/indexed_stack.dart Visibility.of
func VisibilityOf(bc BuildContext) bool {
	visible := true
	ancestorContext := bc
	ancestor := GetElementForInheritedWidget[*visibilityScope](ancestorContext)
	for visible && ancestor != nil {
		scope := bc.DependOnInheritedElement(ancestor, nil).(*visibilityScope)
		visible = scope.isVisible
		ancestor.VisitAncestorElements(func(parent Element) bool {
			ancestorContext = parent
			return false
		})
		ancestor = GetElementForInheritedWidget[*visibilityScope](ancestorContext)
	}
	return visible
}

// visibilityScope tells the descendants whether they are visible.
//
// Flutter: widgets/indexed_stack.dart _VisibilityScope
type visibilityScope struct {
	InheritedWidget
	isVisible bool
}

func newVisibilityScope(isVisible bool, child Widget) *visibilityScope {
	return &visibilityScope{NewInheritedWidgetBase(child), isVisible}
}

func (w *visibilityScope) UpdateShouldNotify(old Widget) bool {
	return w.isVisible != old.(*visibilityScope).isVisible
}

// renderVisibility is a widget that hides its child without the forced
// compositing of an Opacity.
//
// Flutter: widgets/indexed_stack.dart _Visibility
type renderVisibility struct {
	SingleChildRenderObjectWidget
	visible, maintainSemantics bool
}

func newRenderVisibility(visible, maintainSemantics bool, child Widget) *renderVisibility {
	return &renderVisibility{NewSingleChildRenderObjectWidgetBase(child), visible, maintainSemantics}
}

func (w *renderVisibility) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &visibilityBox{visible: w.visible, maintainSemantics: w.maintainSemantics}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *renderVisibility) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*visibilityBox)
	r.setVisible(w.visible)
	r.setMaintainSemantics(w.maintainSemantics)
}

// visibilityBox paints its child only while it is visible.
//
// Flutter: widgets/indexed_stack.dart _RenderVisibility
type visibilityBox struct {
	rendering.RenderProxyBox
	visible, maintainSemantics bool
}

func (r *visibilityBox) setVisible(v bool) {
	if r.visible != v {
		r.visible = v
		r.MarkNeedsPaint()
	}
}

func (r *visibilityBox) setMaintainSemantics(v bool) {
	if r.maintainSemantics != v {
		r.maintainSemantics = v
		r.MarkNeedsSemanticsUpdate()
	}
}

// VisitChildrenForSemantics visits the child if it is visible or its semantics
// are maintained.
func (r *visibilityBox) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	if (r.maintainSemantics || r.visible) && r.Child() != nil {
		visitor(r.Child())
	}
}

// Paint paints the child if it is visible.
func (r *visibilityBox) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if !r.visible {
		return
	}
	r.RenderProxyBox.Paint(context, offset)
}
