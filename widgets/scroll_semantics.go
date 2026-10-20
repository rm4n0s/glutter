// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// scrollSemantics is the widget that makes the semantics of a Scrollable: the
// scroll position and extents, and the handler for scrolling to an offset.
//
// Flutter: widgets/scrollable.dart _ScrollSemantics
type scrollSemantics struct {
	SingleChildRenderObjectWidget
	position               ScrollPosition
	allowImplicitScrolling bool
	semanticChildCount     foundation.Opt[int]
	axis                   painting.Axis
}

func newScrollSemantics(position ScrollPosition, allowImplicitScrolling bool, axis painting.Axis, semanticChildCount foundation.Opt[int], child Widget, key foundation.Key) *scrollSemantics {
	return &scrollSemantics{
		SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: key}),
		position:                      position, allowImplicitScrolling: allowImplicitScrolling, semanticChildCount: semanticChildCount, axis: axis,
	}
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/scrollable.dart _ScrollSemantics.createRenderObject
func (w *scrollSemantics) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newRenderScrollSemantics(w.position, w.allowImplicitScrolling, w.axis, w.semanticChildCount)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/scrollable.dart _ScrollSemantics.updateRenderObject
func (w *scrollSemantics) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderScrollSemantics)
	r.setAllowImplicitScrolling(w.allowImplicitScrolling)
	r.axis = w.axis
	r.setPosition(w.position)
	r.setSemanticChildCount(w.semanticChildCount)
}

// renderScrollSemantics describes the scrollable to the semantics tree, and puts
// the children of a viewport that uses two-pane semantics under an inner node.
//
// Flutter: widgets/scrollable.dart _RenderScrollSemantics
type renderScrollSemantics struct {
	rendering.RenderProxyBox
	position               ScrollPosition
	positionHandle         foundation.ListenerHandle
	allowImplicitScrolling bool
	axis                   painting.Axis
	semanticChildCount     foundation.Opt[int]
	innerNode              *semantics.SemanticsNode
}

func newRenderScrollSemantics(position ScrollPosition, allowImplicitScrolling bool, axis painting.Axis, semanticChildCount foundation.Opt[int]) *renderScrollSemantics {
	r := &renderScrollSemantics{position: position, allowImplicitScrolling: allowImplicitScrolling, axis: axis, semanticChildCount: semanticChildCount}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	r.positionHandle = position.AddListener(r.MarkNeedsSemanticsUpdate)
	return r
}

func (r *renderScrollSemantics) setPosition(value ScrollPosition) {
	if value == r.position {
		return
	}
	r.position.RemoveListener(r.positionHandle)
	r.position = value
	r.positionHandle = r.position.AddListener(r.MarkNeedsSemanticsUpdate)
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderScrollSemantics) setAllowImplicitScrolling(value bool) {
	if value == r.allowImplicitScrolling {
		return
	}
	r.allowImplicitScrolling = value
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderScrollSemantics) setSemanticChildCount(value foundation.Opt[int]) {
	if value == r.semanticChildCount {
		return
	}
	r.semanticChildCount = value
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderScrollSemantics) onScrollToOffset(targetOffset ui.Offset) {
	if r.axis == painting.AxisHorizontal {
		r.position.JumpTo(targetOffset.Dx)
	} else {
		r.position.JumpTo(targetOffset.Dy)
	}
}

// DescribeSemanticsConfiguration marks the semantic boundary, the implicit
// scrolling and the scroll extents.
//
// Flutter: widgets/scrollable.dart _RenderScrollSemantics.describeSemanticsConfiguration
func (r *renderScrollSemantics) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
	config.SetHasImplicitScrolling(r.allowImplicitScrolling)
	if r.position.HaveDimensions() {
		config.SetScrollPosition(r.position.Pixels())
		config.SetScrollExtentMax(r.position.MaxScrollExtent())
		config.SetScrollExtentMin(r.position.MinScrollExtent())
		config.SetScrollChildCount(r.semanticChildCount)
		if r.position.MaxScrollExtent() > r.position.MinScrollExtent() && r.allowImplicitScrolling {
			config.SetOnScrollToOffset(r.onScrollToOffset)
		}
	}
}

// AssembleSemanticsNode splits the children of a two-pane viewport between the
// node and an inner node, so that the ones that scroll are under the inner node.
//
// Flutter: widgets/scrollable.dart _RenderScrollSemantics.assembleSemanticsNode
func (r *renderScrollSemantics) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	if len(children) == 0 || !children[0].IsTagged(rendering.RenderViewportUseTwoPaneSemantics) {
		r.innerNode = nil
		r.RenderProxyBox.AssembleSemanticsNode(node, config, children)
		return
	}
	if r.innerNode == nil {
		r.innerNode = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{ShowOnScreen: func() { r.ShowOnScreen(rendering.ShowOnScreenOpts{}) }})
	}
	r.innerNode.SetRect(node.Rect())

	var firstVisibleIndex foundation.Opt[int]
	excluded := []*semantics.SemanticsNode{r.innerNode}
	var included []*semantics.SemanticsNode
	for _, child := range children {
		if child.IsTagged(rendering.RenderViewportExcludeFromScrolling) {
			excluded = append(excluded, child)
		} else {
			if !child.FlagsCollection().IsHidden && !firstVisibleIndex.IsSome() {
				firstVisibleIndex = child.IndexInParent
			}
			included = append(included, child)
		}
	}
	config.SetScrollIndex(firstVisibleIndex)
	node.UpdateWith(nil, excluded)
	r.innerNode.UpdateWith(config, included)
}

// ClearSemantics forgets the inner node too.
//
// Flutter: widgets/scrollable.dart _RenderScrollSemantics.clearSemantics
func (r *renderScrollSemantics) ClearSemantics() {
	r.RenderProxyBox.ClearSemantics()
	r.innerNode = nil
}
