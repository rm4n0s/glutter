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
// Deviation: the key of the internal widget is an ObjectKey of the view, not a
// GlobalKey (SPEC §12.10).
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
		panic(errors.New(tagViewOfWithoutView, msg, "context", bc.(Element).String()))
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
	WidgetsBindingObserverBase
	scopeNode   *FocusScopeNode
	policy      FocusTraversalPolicy
	viewFocus   bool
	scopeHandle foundation.ListenerHandle
	goneHandle  foundation.ListenerHandle
}

func (s *viewState) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

// implicitGone is whether this is the View of the implicit view and the first
// window is gone, closed by the platform or destroyed with its controller.
func (s *viewState) implicitGone() bool {
	return s.Widget().view.ViewID() == ui.ImplicitViewID && s.binding().WindowingOwner().implicitGone.Value()
}

func (s *viewState) InitState() {
	if s.Widget().view.ViewID() == ui.ImplicitViewID {
		s.goneHandle = s.binding().WindowingOwner().implicitGone.AddListener(func() {
			if s.Mounted() {
				s.SetState(func() {})
			}
		})
	}
	label := ""
	if foundation.DebugMode {
		label = "View Scope"
	}
	s.scopeNode = NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: label}})
	s.policy = NewReadingOrderTraversalPolicy()
	s.binding().AddObserver(s)
	s.scopeHandle = s.scopeNode.AddListener(s.scopeFocusChanged)
}

func (s *viewState) Dispose() {
	if s.Widget().view.ViewID() == ui.ImplicitViewID {
		s.binding().WindowingOwner().implicitGone.RemoveListener(s.goneHandle)
	}
	s.binding().RemoveObserver(s)
	s.scopeNode.RemoveListener(s.scopeHandle)
	s.scopeNode.Dispose()
}

// scopeFocusChanged tells the view that its scope gained the focus, when the view
// itself does not have the focus of the platform yet, so that the platform moves
// the focus to the window of the view.
//
// Flutter: widgets/view.dart _ViewState._scopeFocusChangeListener
func (s *viewState) scopeFocusChanged() {
	if s.viewFocus == s.scopeNode.HasFocus() || !s.scopeNode.HasFocus() {
		return
	}
	s.binding().PlatformDispatcher().RequestViewFocusChange(s.Widget().view.ViewID(), ui.ViewFocusStateFocused, ui.ViewFocusDirectionForward)
}

// DidChangeViewFocus moves the focus into the scope of the view when the window of
// the view got the focus of the platform, and parks it on the root scope when the
// window lost it.
//
// Flutter: widgets/view.dart _ViewState.didChangeViewFocus
func (s *viewState) DidChangeViewFocus(event ui.ViewFocusEvent) {
	id := s.Widget().view.ViewID()
	s.viewFocus = event.State == ui.ViewFocusStateFocused && event.ViewID == id
	if event.ViewID != id {
		return
	}
	switch event.State {
	case ui.ViewFocusStateFocused:
		var next *FocusNode
		switch event.Direction {
		case ui.ViewFocusDirectionForward:
			if next = s.policy.FindFirstFocus(&s.scopeNode.FocusNode, true); next == nil {
				next = &s.scopeNode.FocusNode
			}
		case ui.ViewFocusDirectionBackward:
			next = s.policy.FindLastFocus(&s.scopeNode.FocusNode, true)
		default:
			next = &s.scopeNode.FocusNode
		}
		next.RequestFocus()
	case ui.ViewFocusStateUnfocused:
		// Focusing the root scope "parks" the focus: no descendant has it, so no
		// widget receives the keyboard events.
		s.Context().Owner().FocusManager().rootScope.RequestScopeFocus()
	}
}

// Build shows the child in a RawView with the MediaQuery of the view, a
// FocusTraversalGroup and the view's focus scope.
func (s *viewState) Build(bc BuildContext) Widget {
	w := s.Widget()
	if s.implicitGone() {
		// The first window was closed or destroyed while other windows go on: its
		// content is taken out of the tree, and its states are disposed.
		// A ViewCollection of no views is a widget with no render object, which is
		// right wherever a View is (at the root, or in a collection of views).
		return NewViewCollection(nil)
	}
	return NewRawView(w.view, NewMediaQueryFromView(w.view,
		// This view's focus subtree is attached directly to the root scope rather
		// than nested under the scope of an enclosing view (which happens when a
		// view is rendered inside another via a ViewAnchor): each view is an
		// independent focus root.
		NewFocusTraversalGroup(
			NewFocusScopeWithExternalFocusNode(s.scopeNode, w.child, FocusScopeWithExternalFocusNodeOpts{ExcludeSemantics: true}),
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

func (w *rawViewInternal) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := rendering.NewRenderView(w.view)
	// The style that the annotated regions ask for goes to SystemChrome after each frame.
	r.OnSystemUIOverlayStyle = WidgetsBindingOf(bc).SystemChrome().SetSystemUIOverlayStyle
	return r
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
	var report foundation.ErrorReporter
	taken := false
	if e.owner.OnError != nil {
		// An ErrorBoundary above the view is told first.
		report = func(err error) { taken = e.owner.reportBuildError(e, err) }
	}
	panicked := foundation.Guard(report, foundation.BuildPanicked, "while building "+e.String(), func() []any {
		return []any{"library", "widgets library", "element", e.String(), "creatorChain", e.DebugGetCreatorChain(12)}
	}, func() {
		child := e.widget.(*rawViewInternal).builder(e, e.ownerOfRender())
		e.child = e.UpdateChild(e.child, child, nil)
	})
	if panicked {
		if taken {
			dropStaleChild(e, e.child)
			e.child = nil
		}
		e.child = e.UpdateChild(nil, e.owner.errorWidgetFor(errors.New(tagViewBuildFailed, "the content of the view could not be built")), e.slot)
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
		panic(errors.New(tagRawViewChildMoved, "the child of a view cannot be moved"))
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRawViewChildMoved = "RawViewChildMoved"
	tagViewBuildFailed   = "ViewBuildFailed"
	tagViewOfWithoutView = "ViewOfWithoutView"
)
