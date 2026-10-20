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

// ViewOpts are the optional parameters of NewView.
//
// Flutter: widgets/view.dart View.new
type ViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// View is the bridge between the widget tree and a view of the platform: it
// shows its child in the view (a window), makes the view available to the
// descendants with ViewOf and MaybeViewOf, and provides the MediaQuery of the
// view. RunApp wraps the app in a View for the first window.
//
// Deviation: the view focus events (ViewFocusEvent, requestViewFocusChange) are
// not ported: nothing tells the view that its window got or lost the keyboard
// focus, so the view scope takes the focus only when a widget asks for it
// (TODO(step 14.2): the windows). The key of the internal widget is an ObjectKey
// of the view, not a GlobalKey (SPEC §12.10).
//
// Flutter: widgets/view.dart View
type View struct {
	StatefulWidget
	view  *ui.View
	child Widget
}

// NewView creates a View that shows the child in the view.
//
// Flutter: widgets/view.dart View.new
func NewView(view *ui.View, child Widget, opts ...ViewOpts) *View {
	o := firstOpts(opts)
	return &View{StatefulWidget: NewStatefulWidgetBase(WidgetOpts(o)), view: view, child: child}
}

// UIView is the view the widget shows its child in.
//
// Flutter: widgets/view.dart View.view
func (w *View) UIView() *ui.View { return w.view }

// Child is the widget shown in the view.
//
// Flutter: widgets/view.dart View.child
func (w *View) Child() Widget { return w.child }

// CreateState creates the state of the View.
//
// Flutter: widgets/view.dart View.createState
func (w *View) CreateState() State { return &viewState{} }

// MaybeViewOf is the view that the closest View or RawView above the context
// shows its child in, or nil. The context depends on it. The lookup does not go
// past a LookupBoundary.
//
// Flutter: widgets/view.dart View.maybeOf
func MaybeViewOf(bc BuildContext) *ui.View {
	if scope, ok := LookupBoundaryDependOnInheritedWidgetOfExactType[*viewScope](bc); ok {
		return scope.view
	}
	return nil
}

// ViewOf is MaybeViewOf that panics with the tag ViewOfWithoutView in debug
// builds, and in release builds too, when there is no View above the context.
//
// Flutter: widgets/view.dart View.of
func ViewOf(bc BuildContext) *ui.View {
	v := MaybeViewOf(bc)
	if v == nil {
		msg := "ViewOf was called with a context that does not contain a View widget; no View widget ancestor could be found starting from the context that was passed to ViewOf; this usually means that the provided context is not associated with a View"
		if LookupBoundaryDebugIsHidingAncestorWidgetOfExactType[*viewScope](bc) {
			msg = "ViewOf was called with a context that does not have access to a View widget; the context provided to ViewOf does have a View widget ancestor, but it is hidden by a LookupBoundary; this usually means that the provided context is not associated with a View"
		}
		panic(errors.New("ViewOfWithoutView", msg, "context", bc.(Element).String()))
	}
	return v
}

// PipelineOwnerOf is the pipeline owner that the render view of the closest
// View above the context uses; the root pipeline owner of the renderer binding
// when there is no View above.
//
// Flutter: widgets/view.dart View.pipelineOwnerOf
func PipelineOwnerOf(bc BuildContext) *rendering.PipelineOwner {
	if scope, ok := DependOn[*pipelineOwnerScope](bc); ok {
		return scope.pipelineOwner
	}
	return WidgetsBindingOf(bc).Renderer().RootPipelineOwner()
}

type viewState struct {
	StateOf[*View]
	scopeNode *FocusScopeNode
	policy    FocusTraversalPolicy
}

func (s *viewState) InitState() {
	label := ""
	if foundation.DebugMode {
		label = "View Scope"
	}
	s.scopeNode = NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: label}})
	s.policy = NewReadingOrderTraversalPolicy()
}

func (s *viewState) Dispose() { s.scopeNode.Dispose() }

// Build shows the child in a RawView with the MediaQuery of the view, a
// FocusTraversalGroup and the view's focus scope.
func (s *viewState) Build(bc BuildContext) Widget {
	w := s.Widget()
	return NewRawView(w.view, NewMediaQueryFromView(w.view,
		// This view's focus subtree is attached directly to the root scope rather
		// than nested under the scope of an enclosing view (which happens when a
		// view is rendered inside another via a ViewAnchor): each view is an
		// independent focus root.
		NewFocusTraversalGroup(
			NewFocusScopeWithExternalFocusNode(w.child, s.scopeNode, FocusScopeWithExternalFocusNodeOpts{ExcludeSemantics: true}),
			FocusTraversalGroupOpts{Policy: s.policy, ParentNode: &bc.Owner().FocusManager().rootScope.FocusNode},
		)))
}

// RawView is a View without the MediaQuery and the focus scope: the bridge between the widget tree and a view, nothing else. It
// creates the render view of the view and owns the pipeline owner of its render
// tree.
//
// Flutter: widgets/view.dart RawView
type RawView struct {
	StatelessWidget
	view  *ui.View
	child Widget
}

// NewRawView creates a RawView that shows the child in the view.
//
// Flutter: widgets/view.dart RawView.new
func NewRawView(view *ui.View, child Widget, opts ...ViewOpts) *RawView {
	o := firstOpts(opts)
	return &RawView{StatelessWidget: NewStatelessWidgetBase(WidgetOpts(o)), view: view, child: child}
}

// Build makes the internal widget that owns the render view.
//
// Flutter: widgets/view.dart RawView.build
func (w *RawView) Build(BuildContext) Widget {
	view, child := w.view, w.child
	return newRawViewInternal(view, func(_ BuildContext, owner *rendering.PipelineOwner) Widget {
		return newViewScope(view, newPipelineOwnerScope(owner, child))
	})
}

// rawViewInternal is the render object widget whose render object is the
// RenderView.
//
// Flutter: widgets/view.dart _RawViewInternal
type rawViewInternal struct {
	LeafRenderObjectWidget
	view    *ui.View
	builder func(bc BuildContext, owner *rendering.PipelineOwner) Widget
}

func newRawViewInternal(view *ui.View, builder func(BuildContext, *rendering.PipelineOwner) Widget) *rawViewInternal {
	return &rawViewInternal{
		LeafRenderObjectWidget: NewLeafRenderObjectWidgetBase(WidgetOpts{Key: foundation.NewObjectKey(view)}),
		view:                   view, builder: builder,
	}
}

func (w *rawViewInternal) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderView(w.view)
}

func (w *rawViewInternal) CreateElement() Element {
	e := &rawViewElement{}
	InitRenderTreeRootElement(&e.RenderTreeRootElement, e, w)
	return e
}

// rawViewElement is the element of a rawViewInternal: the root of the render
// tree of a view. It attaches the render view to the pipeline owner of the
// closest View above (or the root one) and builds the child.
//
// Flutter: widgets/view.dart _RawViewElement
type rawViewElement struct {
	RenderTreeRootElement
	pipelineOwner       *rendering.PipelineOwner
	parentPipelineOwner *rendering.PipelineOwner
	child               Element
}

func (e *rawViewElement) ownerOfRender() *rendering.PipelineOwner {
	if e.pipelineOwner == nil {
		e.pipelineOwner = rendering.NewViewPipelineOwner()
	}
	return e.pipelineOwner
}

func (e *rawViewElement) renderView() *rendering.RenderView {
	return e.renderObject.(*rendering.RenderView)
}

func (e *rawViewElement) updateChild() {
	panicked := foundation.Guard(e.owner.OnError, foundation.BuildPanicked, "while building "+e.String(), func() []any {
		return []any{"library", "widgets library", "element", e.String(), "creatorChain", e.DebugGetCreatorChain(12)}
	}, func() {
		child := e.widget.(*rawViewInternal).builder(e, e.ownerOfRender())
		e.child = e.UpdateChild(e.child, child, nil)
	})
	if panicked {
		e.child = e.UpdateChild(nil, e.owner.errorWidgetFor(errors.New("ViewBuildFailed", "the content of the view could not be built")), e.slot)
	}
}

func (e *rawViewElement) Mount(parent Element, slot any) {
	e.RenderObjectElement.Mount(parent, slot)
	owner := e.ownerOfRender()
	owner.SetRootNode(e.renderView())
	e.attachView(nil)
	e.updateChild()
	e.renderView().PrepareInitialFrame()
	if owner.SemanticsOwner() != nil {
		e.renderView().ScheduleInitialSemantics()
	}
}

func (e *rawViewElement) attachView(parentOwner *rendering.PipelineOwner) {
	if parentOwner == nil {
		parentOwner = PipelineOwnerOf(e)
	}
	parentOwner.AdoptChild(e.ownerOfRender())
	WidgetsBindingOf(e).Renderer().AddRenderView(e.renderView())
	e.parentPipelineOwner = parentOwner
}

func (e *rawViewElement) detachView() {
	if e.parentPipelineOwner == nil {
		return
	}
	WidgetsBindingOf(e).Renderer().RemoveRenderView(e.renderView())
	e.parentPipelineOwner.DropChild(e.ownerOfRender())
	e.parentPipelineOwner = nil
}

func (e *rawViewElement) DidChangeDependencies() {
	e.RenderObjectElement.DidChangeDependencies()
	if e.parentPipelineOwner == nil {
		return
	}
	if newParent := PipelineOwnerOf(e); newParent != e.parentPipelineOwner {
		e.detachView()
		e.attachView(newParent)
	}
}

func (e *rawViewElement) PerformRebuild() {
	e.RenderObjectElement.PerformRebuild()
	e.updateChild()
}

func (e *rawViewElement) Activate() {
	e.RenderObjectElement.Activate()
	e.ownerOfRender().SetRootNode(e.renderView())
	e.attachView(nil)
}

func (e *rawViewElement) Deactivate() {
	e.detachView()
	e.ownerOfRender().SetRootNode(nil)
	e.RenderObjectElement.Deactivate()
}

func (e *rawViewElement) Update(newWidget Widget) {
	e.RenderObjectElement.Update(newWidget)
	e.updateChild()
}

func (e *rawViewElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
}

func (e *rawViewElement) ForgetChild(child Element) {
	e.child = nil
	e.RenderObjectElement.ForgetChild(child)
}

func (e *rawViewElement) InsertRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(singleChildHolder).SetChildObject(child)
}

func (e *rawViewElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {
	if foundation.DebugMode {
		panic(errors.New("RawViewChildMoved", "the child of a view cannot be moved"))
	}
}

func (e *rawViewElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	e.renderObject.(singleChildHolder).SetChildObject(nil)
}

func (e *rawViewElement) Unmount() {
	e.ownerOfRender().Dispose()
	e.RenderObjectElement.Unmount()
}

// viewScope tells the descendants which view they are in.
//
// Flutter: widgets/view.dart _ViewScope
type viewScope struct {
	InheritedWidget
	view *ui.View
}

func newViewScope(view *ui.View, child Widget) *viewScope {
	return &viewScope{NewInheritedWidgetBase(child), view}
}

func (w *viewScope) UpdateShouldNotify(old Widget) bool { return w.view != old.(*viewScope).view }

// pipelineOwnerScope tells the descendants which pipeline owner their render
// tree belongs to.
//
// Flutter: widgets/view.dart _PipelineOwnerScope
type pipelineOwnerScope struct {
	InheritedWidget
	pipelineOwner *rendering.PipelineOwner
}

func newPipelineOwnerScope(owner *rendering.PipelineOwner, child Widget) *pipelineOwnerScope {
	return &pipelineOwnerScope{NewInheritedWidgetBase(child), owner}
}

func (w *pipelineOwnerScope) UpdateShouldNotify(old Widget) bool {
	return w.pipelineOwner != old.(*pipelineOwnerScope).pipelineOwner
}
