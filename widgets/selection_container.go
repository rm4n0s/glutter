package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// SelectionContainer is a container that allows Selectable widgets to
// participate in the selection of a SelectionArea or a SelectableRegion above
// it, and lets its delegate decide how the selection of the widgets below it is
// handled and what content the container gives.
//
// Flutter: widgets/selection_container.dart SelectionContainer
type SelectionContainer struct {
	StatefulWidget
	registrar rendering.SelectionRegistrar
	delegate  SelectionContainerDelegate
	child     Widget
}

// SelectionContainerOpts are the optional parameters of NewSelectionContainer.
//
// Flutter: widgets/selection_container.dart SelectionContainer.new
type SelectionContainerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Registrar is the SelectionRegistrar the container registers with; the
	// default is the nearest registrar above it (SelectionContainerMaybeOf).
	Registrar rendering.SelectionRegistrar
}

// NewSelectionContainer creates a selection container whose delegate handles
// the selection of the widgets below the child.
//
// Flutter: widgets/selection_container.dart SelectionContainer.new
func NewSelectionContainer(delegate SelectionContainerDelegate, child Widget, opts ...SelectionContainerOpts) *SelectionContainer {
	o := firstOpts(opts)
	return &SelectionContainer{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), registrar: o.Registrar, delegate: delegate, child: child}
}

// NewSelectionContainerDisabled creates a selection container that disables
// the selection below it: the Selectables under it do not register.
//
// Flutter: widgets/selection_container.dart SelectionContainer.disabled
func NewSelectionContainerDisabled(child Widget, opts ...SelectionContainerOpts) *SelectionContainer {
	o := firstOpts(opts)
	return &SelectionContainer{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child}
}

// SelectionContainerMaybeOf gets the SelectionRegistrar from the context, or
// nil if there is none or the selection is disabled.
//
// Flutter: widgets/selection_container.dart SelectionContainer.maybeOf
func SelectionContainerMaybeOf(bc BuildContext) rendering.SelectionRegistrar {
	if scope, ok := DependOn[*SelectionRegistrarScope](bc); ok {
		return scope.registrar
	}
	return nil
}

func (w *SelectionContainer) disabled() bool { return w.delegate == nil }

// CreateState creates the state.
//
// Flutter: widgets/selection_container.dart SelectionContainer.createState
func (w *SelectionContainer) CreateState() State {
	return &selectionContainerState{notifier: foundation.NewChangeNotifier()}
}

// disabledGeometry is the geometry of a disabled container: it has content so
// that the selection passes by it.
var disabledSelectionGeometry = rendering.SelectionGeometry{Status: rendering.SelectionStatusNone, HasContent: true}

// selectionContainerState is the state of a SelectionContainer; it is the
// Selectable that the container registers with the registrar above it.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState
type selectionContainerState struct {
	StateOf[*SelectionContainer]
	rendering.SelectionRegistrant

	// notifier fans the delegate's changes out to the listeners of the state.
	notifier       *foundation.ChangeNotifier
	delegateHandle foundation.ListenerHandle
	hasHandle      bool
}

func (s *selectionContainerState) subscribe(d SelectionContainerDelegate) {
	if d == nil || s.hasHandle {
		return
	}
	s.delegateHandle, s.hasHandle = d.AddListener(s.notifier.NotifyListeners), true
}

func (s *selectionContainerState) unsubscribe(d SelectionContainerDelegate) {
	if d != nil && s.hasHandle {
		d.RemoveListener(s.delegateHandle)
	}
	s.hasHandle = false
}

func (s *selectionContainerState) InitState() {
	rendering.InitSelectionRegistrant(&s.SelectionRegistrant, s)
	w := s.Widget()
	if !w.disabled() {
		w.delegate.selectionContainerBase().attach(s.Context())
		s.subscribe(w.delegate)
		if w.registrar != nil {
			s.SetRegistrar(w.registrar)
		}
	}
}

func (s *selectionContainerState) DidUpdateWidget(old *SelectionContainer) {
	w := s.Widget()
	if old.delegate != w.delegate {
		if !old.disabled() {
			old.delegate.selectionContainerBase().attach(nil)
			s.unsubscribe(old.delegate)
		}
		if !w.disabled() {
			w.delegate.selectionContainerBase().attach(s.Context())
			s.subscribe(w.delegate)
		}
		oldValue, newValue := foundation.None[rendering.SelectionGeometry](), foundation.None[rendering.SelectionGeometry]()
		if !old.disabled() {
			oldValue = foundation.Some(old.delegate.Value())
		}
		if !w.disabled() {
			newValue = foundation.Some(w.delegate.Value())
		}
		if oldValue.IsSome() != newValue.IsSome() || (oldValue.IsSome() && !oldValue.Value().Equal(newValue.Value())) {
			s.notifier.NotifyListeners()
		}
	}
	if w.disabled() {
		s.SetRegistrar(nil)
	} else if w.registrar != nil {
		s.SetRegistrar(w.registrar)
	}
}

func (s *selectionContainerState) DidChangeDependencies() {
	w := s.Widget()
	if w.registrar == nil && !w.disabled() {
		s.SetRegistrar(SelectionContainerMaybeOf(s.Context()))
	}
}

// AddListener adds a listener to the delegate's changes.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.addListener
func (s *selectionContainerState) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	s.assertEnabled()
	return s.notifier.AddListener(listener)
}

// RemoveListener removes a listener.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.removeListener
func (s *selectionContainerState) RemoveListener(handle foundation.ListenerHandle) {
	s.notifier.RemoveListener(handle)
}

func (s *selectionContainerState) assertEnabled() {
	if foundation.DebugMode && s.Widget().disabled() {
		panic(errors.New("SelectionContainerDisabled", "the SelectionContainer is disabled"))
	}
}

// PushHandleLayers gives the layer links to the delegate.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.pushHandleLayers
func (s *selectionContainerState) PushHandleLayers(startHandle, endHandle *rendering.LayerLink) {
	s.assertEnabled()
	s.Widget().delegate.PushHandleLayers(startHandle, endHandle)
}

// GetSelectedContent is the content that the delegate selected.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.getSelectedContent
func (s *selectionContainerState) GetSelectedContent() *rendering.SelectedContent {
	s.assertEnabled()
	return s.Widget().delegate.GetSelectedContent()
}

// GetSelection is the range of the selection in the delegate.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.getSelection
func (s *selectionContainerState) GetSelection() *rendering.SelectedContentRange {
	s.assertEnabled()
	return s.Widget().delegate.GetSelection()
}

// DispatchSelectionEvent gives the event to the delegate.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.dispatchSelectionEvent
func (s *selectionContainerState) DispatchSelectionEvent(event rendering.SelectionEvent) rendering.SelectionResult {
	s.assertEnabled()
	return s.Widget().delegate.DispatchSelectionEvent(event)
}

// Value is the geometry of the selection of the delegate.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.value
func (s *selectionContainerState) Value() rendering.SelectionGeometry {
	if s.Widget().disabled() {
		return disabledSelectionGeometry
	}
	return s.Widget().delegate.Value()
}

// GetTransformTo is the transform of the container to the ancestor.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.getTransformTo
func (s *selectionContainerState) GetTransformTo(ancestor rendering.RenderObject) *ui.Matrix4 {
	s.assertEnabled()
	return s.Context().FindRenderObject().GetTransformTo(ancestor)
}

// ContentLength is the length of the content of the delegate.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.contentLength
func (s *selectionContainerState) ContentLength() int { return s.Widget().delegate.ContentLength() }

// Size is the size of the container.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.size
func (s *selectionContainerState) Size() ui.Size {
	return s.Context().FindRenderObject().(rendering.Box).Size()
}

// BoundingBoxes is the box of the container.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.boundingBoxes
func (s *selectionContainerState) BoundingBoxes() []ui.Rect {
	return []ui.Rect{s.Context().FindRenderObject().(rendering.Box).PaintBounds()}
}

// Dispose releases the delegate and the registration.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.dispose
func (s *selectionContainerState) Dispose() {
	w := s.Widget()
	if !w.disabled() {
		w.delegate.selectionContainerBase().attach(nil)
		s.unsubscribe(w.delegate)
	}
	s.SelectionRegistrant.Dispose()
	s.notifier.Dispose()
}

// Build wraps the child in the scope of the registrar.
//
// Flutter: widgets/selection_container.dart _SelectionContainerState.build
func (s *selectionContainerState) Build(BuildContext) Widget {
	w := s.Widget()
	if w.disabled() {
		return &SelectionRegistrarScope{InheritedWidget: NewInheritedWidgetBase(w.child)}
	}
	return NewSelectionRegistrarScope(w.delegate, w.child)
}

// SelectionRegistrarScope is the inherited widget that gives the registrar of a
// SelectionContainer to the Selectables below it.
//
// Flutter: widgets/selection_container.dart SelectionRegistrarScope
type SelectionRegistrarScope struct {
	InheritedWidget
	registrar rendering.SelectionRegistrar
}

// NewSelectionRegistrarScope creates a scope that gives the registrar.
//
// Flutter: widgets/selection_container.dart SelectionRegistrarScope.new
func NewSelectionRegistrarScope(registrar rendering.SelectionRegistrar, child Widget, opts ...WidgetOpts) *SelectionRegistrarScope {
	return &SelectionRegistrarScope{InheritedWidget: NewInheritedWidgetBase(child, opts...), registrar: registrar}
}

// UpdateShouldNotify is whether the registrar is another one.
//
// Flutter: widgets/selection_container.dart SelectionRegistrarScope.updateShouldNotify
func (w *SelectionRegistrarScope) UpdateShouldNotify(old Widget) bool {
	return old.(*SelectionRegistrarScope).registrar != w.registrar
}

// SelectionContainerDelegate is the delegate of a SelectionContainer: it
// handles the selection events of the container and the Selectables that
// register with it. Embed SelectionContainerDelegateBase in the implementation.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate
type SelectionContainerDelegate interface {
	rendering.SelectionHandler
	rendering.SelectionRegistrar
	selectionContainerBase() *SelectionContainerDelegateBase
}

// SelectionContainerDelegateBase has what the delegates of a container share:
// the context of the container, and the transforms and the size relative to it.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate
type SelectionContainerDelegateBase struct {
	context BuildContext
	// bindingOfContext is the binding of the last context, kept so that the
	// delegate can still schedule work after its container unmounts.
	bindingOfContext *WidgetsBinding
}

// attach sets the context of the container; a nil context detaches the delegate
// but keeps its binding.
func (b *SelectionContainerDelegateBase) attach(bc BuildContext) {
	b.context = bc
	if bc != nil {
		b.bindingOfContext = WidgetsBindingOf(bc)
	}
}

func (b *SelectionContainerDelegateBase) selectionContainerBase() *SelectionContainerDelegateBase {
	return b
}

func (b *SelectionContainerDelegateBase) containerBox(what string) rendering.Box {
	var ro rendering.RenderObject
	if b.context != nil {
		ro = b.context.FindRenderObject()
	}
	if ro == nil {
		panic(errors.New("SelectionContainerNotLaidOut", what+" cannot be called before the SelectionContainer is built"))
	}
	return ro.(rendering.Box)
}

// GetTransformFrom is the transform from the child to the container.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate.getTransformFrom
func (b *SelectionContainerDelegateBase) GetTransformFrom(child rendering.Selectable) *ui.Matrix4 {
	return child.GetTransformTo(b.containerBox("GetTransformFrom"))
}

// GetTransformTo is the transform from the container to the ancestor.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate.getTransformTo
func (b *SelectionContainerDelegateBase) GetTransformTo(ancestor rendering.RenderObject) *ui.Matrix4 {
	return b.containerBox("GetTransformTo").GetTransformTo(ancestor)
}

// HasSize reports whether the container has been laid out.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate.hasSize
func (b *SelectionContainerDelegateBase) HasSize() bool {
	return b.containerBox("HasSize").HasSize()
}

// ContainerSize is the size of the container; it must have been laid out.
//
// Flutter: widgets/selection_container.dart SelectionContainerDelegate.containerSize
func (b *SelectionContainerDelegateBase) ContainerSize() ui.Size {
	box := b.containerBox("ContainerSize")
	if foundation.DebugMode && !box.HasSize() {
		panic(errors.New("SelectionContainerNotLaidOut", "ContainerSize cannot be called before the SelectionContainer is laid out"))
	}
	return box.Size()
}
