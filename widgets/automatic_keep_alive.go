package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
)

// KeepAliveOpts are the optional parameters of NewKeepAlive.
//
// Flutter: widgets/sliver.dart KeepAlive.new
type KeepAliveOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// KeepAlive marks a child of a sliver list (or of a two-dimensional viewport)
// as one to keep alive when it scrolls out of view. It is the parent data
// widget that AutomaticKeepAlive builds.
//
// Flutter: widgets/sliver.dart KeepAlive
type KeepAlive struct {
	ParentDataWidgetOf[rendering.KeepAliveParentData]
	keepAlive bool
}

// NewKeepAlive creates a widget that tells the parent whether to keep the child
// alive.
//
// Flutter: widgets/sliver.dart KeepAlive.new
func NewKeepAlive(keepAlive bool, child Widget, opts ...KeepAliveOpts) *KeepAlive {
	return &KeepAlive{NewParentDataWidgetBase[rendering.KeepAliveParentData](child, WidgetOpts{Key: firstOpts(opts).Key}), keepAlive}
}

// KeepAlive is whether the child is kept alive.
//
// Flutter: widgets/sliver.dart KeepAlive.keepAlive
func (w *KeepAlive) KeepAlive() bool { return w.keepAlive }

// ApplyParentData writes the flag to the parent data. When the flag is turned
// off, the parent lays out again, so that it can let the child go.
//
// Flutter: widgets/sliver.dart KeepAlive.applyParentData
func (w *KeepAlive) ApplyParentData(renderObject rendering.RenderObject) {
	parentData := renderObject.ParentData().(rendering.KeepAliveParentData).KeepAliveData()

	if parentData.KeepAlive != w.keepAlive {
		parentData.KeepAlive = w.keepAlive
		if !w.keepAlive {
			if p := renderObject.AsRenderObject().Parent(); p != nil {
				p.MarkNeedsLayout()
			}
		}
	}
}

// DebugCanApplyOutOfTurn is whether the widget keeps the child alive: only that
// can be applied without a rebuild.
//
// Flutter: widgets/sliver.dart KeepAlive.debugCanApplyOutOfTurn
func (w *KeepAlive) DebugCanApplyOutOfTurn() bool { return w.keepAlive }

// DebugTypicalAncestorWidgetClass is not used: DebugTypicalAncestorWidgetDescription
// names the ancestors.
//
// Flutter: widgets/sliver.dart KeepAlive.debugTypicalAncestorWidgetClass
func (w *KeepAlive) DebugTypicalAncestorWidgetClass() string {
	return "SliverWithKeepAliveWidget or TwoDimensionalViewport"
}

// DebugTypicalAncestorWidgetDescription names the widgets that KeepAlive goes
// inside.
//
// Flutter: widgets/sliver.dart KeepAlive.debugTypicalAncestorWidgetDescription
func (w *KeepAlive) DebugTypicalAncestorWidgetDescription() string {
	return "SliverWithKeepAliveWidget or TwoDimensionalViewport"
}

// KeepAliveNotification is dispatched by a widget that wants its subtree kept
// alive; the handle tells the AutomaticKeepAlive above when it does not any more.
//
// Flutter: widgets/automatic_keep_alive.dart KeepAliveNotification
type KeepAliveNotification struct {
	// Handle is notified when the client no longer wants to be kept alive.
	//
	// Flutter: widgets/automatic_keep_alive.dart KeepAliveNotification.handle
	Handle foundation.Listenable
}

// KeepAliveHandle is the Listenable that a client keeps alive with: disposing it
// tells the AutomaticKeepAlive that the client is done.
//
// Flutter: widgets/automatic_keep_alive.dart KeepAliveHandle
type KeepAliveHandle struct {
	foundation.ChangeNotifier
}

// NewKeepAliveHandle creates a handle.
//
// Flutter: widgets/automatic_keep_alive.dart KeepAliveHandle.new
func NewKeepAliveHandle() *KeepAliveHandle { return &KeepAliveHandle{} }

// Dispose notifies the listeners, then drops them.
//
// Flutter: widgets/automatic_keep_alive.dart KeepAliveHandle.dispose
func (h *KeepAliveHandle) Dispose() {
	h.NotifyListeners()
	h.ChangeNotifier.Dispose()
}

// AutomaticKeepAliveOpts are the optional parameters of NewAutomaticKeepAlive.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAlive.new
type AutomaticKeepAliveOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// AutomaticKeepAlive lets the descendants that dispatch a KeepAliveNotification
// keep their subtree alive in a lazy list: while any of them wants it, it
// builds a KeepAlive(true) parent data widget.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAlive
type AutomaticKeepAlive struct {
	StatefulWidget
	child Widget
}

// NewAutomaticKeepAlive creates a widget that keeps its child alive when the
// child asks for it.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAlive.new
func NewAutomaticKeepAlive(child Widget, opts ...AutomaticKeepAliveOpts) *AutomaticKeepAlive {
	return &AutomaticKeepAlive{NewStatefulWidgetBase(WidgetOpts{Key: firstOpts(opts).Key}), child}
}

// Child is the widget below this one.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAlive.child
func (w *AutomaticKeepAlive) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAlive.createState
func (w *AutomaticKeepAlive) CreateState() State { return &automaticKeepAliveState{} }

type keepAliveClient struct {
	handle   foundation.ListenerHandle
	callback foundation.VoidCallback
}

// automaticKeepAliveState is the state of an AutomaticKeepAlive.
//
// Flutter: widgets/automatic_keep_alive.dart _AutomaticKeepAliveState
type automaticKeepAliveState struct {
	StateOf[*AutomaticKeepAlive]
	handles      map[foundation.Listenable]*keepAliveClient
	handleOrder  []foundation.Listenable
	child        Widget
	keepingAlive bool
}

func (s *automaticKeepAliveState) InitState() { s.updateChild() }

func (s *automaticKeepAliveState) DidUpdateWidget(*AutomaticKeepAlive) { s.updateChild() }

func (s *automaticKeepAliveState) updateChild() {
	s.child = NewNotificationListener[KeepAliveNotification](s.addClient, s.Widget().child)
}

func (s *automaticKeepAliveState) Dispose() {
	for _, handle := range s.handleOrder {
		if c, ok := s.handles[handle]; ok {
			handle.RemoveListener(c.handle)
		}
	}
}

func (s *automaticKeepAliveState) addClient(notification KeepAliveNotification) bool {
	handle := notification.Handle
	if s.handles == nil {
		s.handles = map[foundation.Listenable]*keepAliveClient{}
	}
	if foundation.DebugMode {
		if _, ok := s.handles[handle]; ok {
			panic(errors.New(tagKeepAliveHandleAdded, "the same KeepAliveHandle was added to an AutomaticKeepAlive twice"))
		}
	}
	client := &keepAliveClient{callback: s.createCallback(handle)}
	client.handle = handle.AddListener(client.callback)
	s.handles[handle] = client
	s.handleOrder = append(s.handleOrder, handle)
	if !s.keepingAlive {
		s.keepingAlive = true
		if childElement := s.childElement(); childElement != nil {
			s.updateParentDataOfChild(childElement)
		} else {
			s.Context().Owner().binding.Scheduler().AddPostFrameCallback(func(time.Duration) {
				if !s.Mounted() {
					return
				}
				childElement := s.childElement()
				if childElement == nil && foundation.DebugMode {
					panic(errors.New(tagKeepAliveChildMissing, "the AutomaticKeepAlive has no KeepAlive child to update"))
				}
				if childElement != nil {
					s.updateParentDataOfChild(childElement)
				}
			})
		}
	}
	return false
}

func (s *automaticKeepAliveState) childElement() *ParentDataElement {
	var child Element
	// VisitChildren rather than VisitChildElements: this can be called during
	// build, and the latter checks that it is not (see the Dart comment).
	s.Context().(Element).VisitChildren(func(e Element) { child = e })
	if child == nil {
		return nil
	}
	pde, ok := child.(*ParentDataElement)
	if !ok && foundation.DebugMode {
		panic(errors.New(tagKeepAliveChildNotParentData, "the child of an AutomaticKeepAlive must be a KeepAlive"))
	}
	return pde
}

func (s *automaticKeepAliveState) updateParentDataOfChild(childElement *ParentDataElement) {
	childElement.ApplyWidgetOutOfTurn(s.Build(s.Context()))
}

func (s *automaticKeepAliveState) removeHandle(handle foundation.Listenable) {
	delete(s.handles, handle)
	for i, h := range s.handleOrder {
		if h == handle {
			s.handleOrder = append(s.handleOrder[:i:i], s.handleOrder[i+1:]...)
			break
		}
	}
}

func (s *automaticKeepAliveState) createCallback(handle foundation.Listenable) foundation.VoidCallback {
	callback := func() {
		if foundation.DebugMode && !s.Mounted() {
			panic(errors.New(tagKeepAliveHandleAfterDispose,
				"AutomaticKeepAlive handle triggered after AutomaticKeepAlive was disposed; widgets should always trigger their KeepAliveNotification handle when they are deactivated, so that they (or their handle) do not send spurious events later when they are no longer in the tree"))
		}
		client := s.handles[handle]
		s.removeHandle(handle)
		if client != nil {
			handle.RemoveListener(client.handle)
		}
		if len(s.handles) == 0 {
			binding := s.Context().Owner().binding
			if binding.Scheduler().SchedulerPhase() < scheduler.SchedulerPhasePersistentCallbacks {
				s.SetState(func() { s.keepingAlive = false })
			} else {
				s.keepingAlive = false
				binding.Post(func() {
					if s.Mounted() && len(s.handles) == 0 {
						s.SetState(func() {})
					}
				})
			}
		}
	}
	return callback
}

func (s *automaticKeepAliveState) Build(BuildContext) Widget {
	return NewKeepAlive(s.keepingAlive, s.child)
}

// AutomaticKeepAliveClient is what a State embeds to keep its subtree alive in
// a lazy list when its WantKeepAlive method returns true. The state calls
// UpdateKeepAlive when the answer changes, and KeepAliveBuild from its Build.
//
//	type itemState struct {
//		widgets.StateOf[*Item]
//		widgets.AutomaticKeepAliveClient
//	}
//
//	func (s *itemState) WantKeepAlive() bool { return true }
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAliveClientMixin
type AutomaticKeepAliveClient struct {
	state           State
	keepAliveHandle *KeepAliveHandle
}

// wantKeepAliver is what the state of an AutomaticKeepAliveClient implements.
type wantKeepAliver interface{ WantKeepAlive() bool }

func (c *AutomaticKeepAliveClient) attachStateMixin(s State) { c.state = s }

func (c *AutomaticKeepAliveClient) activateStateMixin() {}

func (c *AutomaticKeepAliveClient) initStateMixin() {
	if c.wantKeepAlive() {
		c.ensureKeepAlive()
	}
}

func (c *AutomaticKeepAliveClient) deactivateStateMixin() {
	if c.keepAliveHandle != nil {
		c.releaseKeepAlive()
	}
}

func (c *AutomaticKeepAliveClient) disposeStateMixin() {}

func (c *AutomaticKeepAliveClient) wantKeepAlive() bool {
	w, ok := c.state.(wantKeepAliver)
	if !ok {
		panic(errors.New(tagWantKeepAliveMissing, "a state that embeds AutomaticKeepAliveClient must have a WantKeepAlive() bool method",
			"state", foundation.RuntimeTypeName(c.state)))
	}
	return w.WantKeepAlive()
}

func (c *AutomaticKeepAliveClient) context() BuildContext {
	return c.state.stateInternals().data().element
}

func (c *AutomaticKeepAliveClient) ensureKeepAlive() {
	c.keepAliveHandle = NewKeepAliveHandle()
	DispatchNotification(c.context(), KeepAliveNotification{c.keepAliveHandle})
}

func (c *AutomaticKeepAliveClient) releaseKeepAlive() {
	c.keepAliveHandle.Dispose()
	c.keepAliveHandle = nil
}

// UpdateKeepAlive keeps the subtree alive, or lets it go, as WantKeepAlive says.
// Call it when WantKeepAlive changes.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAliveClientMixin.updateKeepAlive
func (c *AutomaticKeepAliveClient) UpdateKeepAlive() {
	if c.wantKeepAlive() {
		if c.keepAliveHandle == nil {
			c.ensureKeepAlive()
		}
	} else if c.keepAliveHandle != nil {
		c.releaseKeepAlive()
	}
}

// KeepAliveBuild is what a Build method calls first (Dart's super.build): it
// asks to be kept alive if WantKeepAlive says so and it has not yet.
//
// Flutter: widgets/automatic_keep_alive.dart AutomaticKeepAliveClientMixin.build
func (c *AutomaticKeepAliveClient) KeepAliveBuild() {
	if c.wantKeepAlive() && c.keepAliveHandle == nil {
		c.ensureKeepAlive()
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagKeepAliveChildMissing       = "KeepAliveChildMissing"
	tagKeepAliveChildNotParentData = "KeepAliveChildNotParentData"
	tagKeepAliveHandleAdded        = "KeepAliveHandleAdded"
	tagKeepAliveHandleAfterDispose = "KeepAliveHandleAfterDispose"
	tagWantKeepAliveMissing        = "WantKeepAliveMissing"
)
