package widgets

import (
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// menuTraversalShortcuts are the shortcuts for traversing the menus.
//
// Flutter: widgets/raw_menu_anchor.dart _kMenuTraversalShortcuts
func menuTraversalShortcuts() ShortcutMap {
	return ShortcutMap{
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyGameButtonA), Intent: ActivateIntent{}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyEscape), Intent: DismissIntent{}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: DirectionalFocusIntent{Direction: TraversalDirectionDown}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: DirectionalFocusIntent{Direction: TraversalDirectionUp}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: DirectionalFocusIntent{Direction: TraversalDirectionLeft}},
		{Activator: NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: DirectionalFocusIntent{Direction: TraversalDirectionRight}},
	}
}

// RawMenuOverlayInfo is the layout information of a menu overlay: the anchor
// rectangle, the size of the overlay, the position that the menu was opened at
// and the group of the tap regions of the menu.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuOverlayInfo
type RawMenuOverlayInfo struct {
	// AnchorRect is the position and size of the anchor widget relative to the
	// overlay.
	//
	// Flutter: widgets/raw_menu_anchor.dart RawMenuOverlayInfo.anchorRect
	AnchorRect ui.Rect
	// OverlaySize is the size of the overlay.
	//
	// Flutter: widgets/raw_menu_anchor.dart RawMenuOverlayInfo.overlaySize
	OverlaySize ui.Size
	// Position is the position passed to MenuController.Open, if any, relative
	// to the anchor.
	//
	// Flutter: widgets/raw_menu_anchor.dart RawMenuOverlayInfo.position
	Position foundation.Opt[ui.Offset]
	// TapRegionGroupID is the group ID of the tap regions of the menu: a menu
	// should wrap its content in a TapRegion with this group ID so taps inside
	// it are not outside taps.
	//
	// Flutter: widgets/raw_menu_anchor.dart RawMenuOverlayInfo.tapRegionGroupId
	TapRegionGroupID any
}

// RawMenuAnchorOverlayBuilder is the type of the function that builds the
// overlay of a RawMenuAnchor.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorOverlayBuilder
type RawMenuAnchorOverlayBuilder func(bc BuildContext, info RawMenuOverlayInfo) Widget

// RawMenuAnchorChildBuilder is the type of the function that builds the anchor
// child of a RawMenuAnchor; child may be nil.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorChildBuilder
type RawMenuAnchorChildBuilder func(bc BuildContext, controller *MenuController, child Widget) Widget

// RawMenuAnchorOpenRequestedCallback is the type of the callback that is called
// when a request to open the menu is made; calling showOverlay opens it.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorOpenRequestedCallback
type RawMenuAnchorOpenRequestedCallback func(position foundation.Opt[ui.Offset], showOverlay foundation.VoidCallback)

// RawMenuAnchorCloseRequestedCallback is the type of the callback that is
// called when a request to close the menu is made; calling hideOverlay closes it.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorCloseRequestedCallback
type RawMenuAnchorCloseRequestedCallback func(hideOverlay foundation.VoidCallback)

// menuControllerScope notifies anchor descendants when a menu opens and closes,
// and passes the controller of the anchor to descendants.
//
// Flutter: widgets/raw_menu_anchor.dart _MenuControllerScope
type menuControllerScope struct {
	InheritedWidget
	isOpen     bool
	controller *MenuController
}

func (w *menuControllerScope) UpdateShouldNotify(old Widget) bool {
	return w.isOpen != old.(*menuControllerScope).isOpen
}

// menuAnchor is what the menu controller and the anchors need of an anchor
// state: the shared base and the methods that the two kinds of anchors
// implement.
type menuAnchor interface {
	State
	anchorBase() *rawMenuAnchorBase
	isOpen() bool
	menuController() *MenuController
	open(position foundation.Opt[ui.Offset])
	close(inDispose bool)
	handleOpenRequest(position foundation.Opt[ui.Offset])
	handleCloseRequest()
	buildAnchor(bc BuildContext) Widget
	mountedState() bool
	setStateOf(fn func())
	contextOf() BuildContext
}

// rawMenuAnchorBase is the common interface and state for both types of
// anchors, RawMenuAnchor and RawMenuAnchorGroup.
//
// Flutter: widgets/raw_menu_anchor.dart _RawMenuAnchorBaseMixin
type rawMenuAnchorBase struct {
	self           menuAnchor
	anchorChildren []menuAnchor
	parent         menuAnchor
	scrollPosition ScrollPosition
	scrollHandle   foundation.ListenerHandle
	viewSize       foundation.Opt[ui.Size]
}

func (b *rawMenuAnchorBase) isRoot() bool { return b.parent == nil }

func (b *rawMenuAnchorBase) root() menuAnchor {
	anchor := b.self
	for anchor.anchorBase().parent != nil {
		anchor = anchor.anchorBase().parent
	}
	return anchor
}

func (b *rawMenuAnchorBase) initState() { b.self.menuController().attach(b.self) }

func (b *rawMenuAnchorBase) didChangeDependencies() {
	var newParent menuAnchor
	if c := MenuControllerMaybeOf(b.self.contextOf()); c != nil {
		newParent = c.anchor
	}
	if newParent != b.parent {
		if foundation.DebugMode && newParent == b.self {
			panic(errors.New(tagMenuControllerMultipleAnchors, "a MenuController should only be attached to one anchor at a time"))
		}
		if b.parent != nil {
			b.parent.anchorBase().removeChild(b.self)
		}
		b.parent = newParent
		if b.parent != nil {
			b.parent.anchorBase().addChild(b.self)
		}
	}
	if b.isRoot() {
		if b.scrollPosition != nil {
			b.scrollPosition.IsScrollingNotifier().RemoveListener(b.scrollHandle)
			b.scrollPosition = nil
		}
		if s := ScrollableMaybeOf(b.self.contextOf()); s != nil {
			b.scrollPosition = s.Position()
			b.scrollHandle = b.scrollPosition.IsScrollingNotifier().AddListener(b.handleScroll)
		}
		newSize := MediaQuerySizeOf(b.self.contextOf())
		if old, ok := b.viewSize.Get(); ok && newSize != old && b.self.isOpen() {
			// Close the menus if the view changes size.
			b.self.handleCloseRequest()
		}
		b.viewSize = foundation.Some(newSize)
	}
}

func (b *rawMenuAnchorBase) dispose() {
	if b.self.isOpen() {
		b.self.close(true)
	}
	if b.isRoot() && b.scrollPosition != nil {
		b.scrollPosition.IsScrollingNotifier().RemoveListener(b.scrollHandle)
		b.scrollPosition = nil
	}
	if b.parent != nil {
		b.parent.anchorBase().removeChild(b.self)
	}
	b.parent = nil
	b.anchorChildren = nil
	b.self.menuController().detach(b.self)
}

func (b *rawMenuAnchorBase) addChild(child menuAnchor) {
	if foundation.DebugMode {
		for _, c := range b.anchorChildren {
			if c == child {
				panic(errors.New(tagMenuAnchorChildAlreadyAdded, "the child anchor is already in this anchor"))
			}
		}
	}
	b.anchorChildren = append(b.anchorChildren, child)
}

func (b *rawMenuAnchorBase) removeChild(child menuAnchor) {
	for i, c := range b.anchorChildren {
		if c == child {
			b.anchorChildren = append(b.anchorChildren[:i], b.anchorChildren[i+1:]...)
			return
		}
	}
	if foundation.DebugMode {
		panic(errors.New(tagMenuAnchorChildNotFound, "the child anchor is not in this anchor"))
	}
}

// handleScroll closes the menus when an ancestor scrolls and this is a root
// anchor. It doesn't close on any scroll, since the menus themselves must be
// able to scroll if they're too big for the view.
func (b *rawMenuAnchorBase) handleScroll() {
	if b.self.isOpen() {
		b.self.handleCloseRequest()
	}
}

func (b *rawMenuAnchorBase) childChangedOpenState() {
	if b.parent != nil {
		b.parent.anchorBase().childChangedOpenState()
	}
	binding := WidgetsBindingOf(b.self.contextOf())
	if binding.Scheduler().SchedulerPhase() != scheduler.SchedulerPhasePersistentCallbacks {
		// Mark dirty now, but only if not in a build.
		b.self.setStateOf(func() {})
	} else {
		binding.RunPostFrame(func(time.Duration) { b.self.setStateOf(func() {}) })
	}
}

func (b *rawMenuAnchorBase) closeChildren(inDispose bool) {
	for _, c := range append([]menuAnchor(nil), b.anchorChildren...) {
		c.close(inDispose)
	}
}

func (b *rawMenuAnchorBase) requestChildrenClose() {
	for _, c := range append([]menuAnchor(nil), b.anchorChildren...) {
		c.handleCloseRequest()
	}
}

func (b *rawMenuAnchorBase) handleOutsideTap(*gestures.PointerDownEvent) {
	if b.self.isOpen() {
		b.requestChildrenClose()
	}
}

func (b *rawMenuAnchorBase) build() Widget {
	actions := ActionMap{}
	// Check if open to allow DismissIntent to bubble when the menu is closed.
	if b.self.isOpen() {
		a := newDismissMenuAction(b.self.menuController())
		actions[a.IntentType()] = a
	}
	return &menuControllerScope{
		InheritedWidget: NewInheritedWidgetBase(NewActions(actions, NewBuilder(b.self.buildAnchor))),
		isOpen:          b.self.isOpen(), controller: b.self.menuController(),
	}
}

// RawMenuAnchorOpts are the optional parameters of NewRawMenuAnchor.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchor.new
type RawMenuAnchorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ChildFocusNode is the focus node of the anchor child, which receives the
	// focus when the menu opens.
	ChildFocusNode *FocusNode
	// ConsumeOutsideTaps stops a tap outside the menu from doing anything else
	// while the menu is open.
	ConsumeOutsideTaps bool
	// OnOpen is called when the menu is opened.
	OnOpen foundation.VoidCallback
	// OnClose is called when the menu is closed.
	OnClose foundation.VoidCallback
	// OnOpenRequested is called when a request to open the menu is made; the
	// default shows the overlay.
	OnOpenRequested RawMenuAnchorOpenRequestedCallback
	// OnCloseRequested is called when a request to close the menu is made; the
	// default hides the overlay.
	OnCloseRequested RawMenuAnchorCloseRequestedCallback
	// UseRootOverlay puts the menu in the root overlay, instead of the nearest.
	UseRootOverlay bool
	// Builder builds the anchor child; the default is the child.
	Builder RawMenuAnchorChildBuilder
}

// RawMenuAnchor is a widget that wraps a child and shows a menu overlay at the
// request of a MenuController. It has no style: it handles the logic of the
// menu, and leaves the look of the menu to the overlay builder.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchor
type RawMenuAnchor struct {
	StatefulWidget
	controller     *MenuController
	overlayBuilder RawMenuAnchorOverlayBuilder
	child          Widget
	o              RawMenuAnchorOpts
}

// NewRawMenuAnchor creates a menu anchor. The child may be nil.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchor.new
func NewRawMenuAnchor(controller *MenuController, overlayBuilder RawMenuAnchorOverlayBuilder, child Widget, opts ...RawMenuAnchorOpts) *RawMenuAnchor {
	o := firstOpts(opts)
	if o.OnOpenRequested == nil {
		o.OnOpenRequested = func(_ foundation.Opt[ui.Offset], showOverlay foundation.VoidCallback) { showOverlay() }
	}
	if o.OnCloseRequested == nil {
		o.OnCloseRequested = func(hideOverlay foundation.VoidCallback) { hideOverlay() }
	}
	return &RawMenuAnchor{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), controller: controller, overlayBuilder: overlayBuilder, child: child, o: o}
}

// Controller is the controller of the anchor.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchor.controller
func (w *RawMenuAnchor) Controller() *MenuController { return w.controller }

// CreateState creates the state.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchor.createState
func (w *RawMenuAnchor) CreateState() State {
	s := &rawMenuAnchorState{overlayController: NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "MenuAnchor controller"})}
	s.base.self = s
	return s
}

type rawMenuAnchorState struct {
	StateOf[*RawMenuAnchor]
	base              rawMenuAnchorBase
	overlayController *OverlayPortalController
	menuPosition      foundation.Opt[ui.Offset]
}

func (s *rawMenuAnchorState) anchorBase() *rawMenuAnchorBase  { return &s.base }
func (s *rawMenuAnchorState) mountedState() bool              { return s.Mounted() }
func (s *rawMenuAnchorState) setStateOf(fn func())            { s.SetState(fn) }
func (s *rawMenuAnchorState) contextOf() BuildContext         { return s.Context() }
func (s *rawMenuAnchorState) menuController() *MenuController { return s.Widget().controller }
func (s *rawMenuAnchorState) isOpen() bool                    { return s.overlayController.IsShowing() }
func (s *rawMenuAnchorState) isRootOverlayAnchor() bool {
	_, ok := s.base.parent.(*rawMenuAnchorState)
	return !ok
}

// useRootOverlay: if this is a nested menu, it uses the same overlay as the
// root menu.
func (s *rawMenuAnchorState) useRootOverlay() bool {
	if p, ok := s.base.parent.(*rawMenuAnchorState); ok {
		return p.useRootOverlay()
	}
	return s.Widget().o.UseRootOverlay
}

func (s *rawMenuAnchorState) InitState()             { s.base.initState() }
func (s *rawMenuAnchorState) DidChangeDependencies() { s.base.didChangeDependencies() }
func (s *rawMenuAnchorState) Dispose()               { s.base.dispose() }

func (s *rawMenuAnchorState) DidUpdateWidget(old *RawMenuAnchor) {
	if old.controller != s.Widget().controller {
		old.controller.detach(s)
		s.Widget().controller.attach(s)
	}
}

func (s *rawMenuAnchorState) open(position foundation.Opt[ui.Offset]) {
	if !s.Mounted() {
		return
	}
	if s.isOpen() {
		// The menu is already open, but we need to move to another location, so
		// close it first.
		s.close(false)
	}
	// Close all siblings.
	if s.base.parent != nil {
		s.base.parent.anchorBase().requestChildrenClose()
	}
	s.menuPosition = position
	if s.base.parent != nil {
		s.base.parent.anchorBase().childChangedOpenState()
	}
	s.overlayController.Show()
	if s.isRootOverlayAnchor() && s.Widget().o.ChildFocusNode != nil {
		s.Widget().o.ChildFocusNode.RequestFocus()
	}
	if f := s.Widget().o.OnOpen; f != nil {
		f()
	}
	// Mark dirty to notify MenuController dependents.
	s.SetState(func() {})
}

func (s *rawMenuAnchorState) close(inDispose bool) {
	if !s.isOpen() {
		return
	}
	// Parents will close after children do.
	s.base.closeChildren(inDispose)
	binding := WidgetsBindingOf(s.Context())
	inBuild := binding.Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks
	// Don't hide if we're in the middle of a build.
	if !inBuild {
		s.overlayController.Hide()
	} else if !inDispose {
		binding.RunPostFrame(func(time.Duration) { s.overlayController.Hide() })
	}
	if !inDispose {
		// Notify that the open state changed, but only if not currently disposing.
		if s.base.parent != nil {
			s.base.parent.anchorBase().childChangedOpenState()
		}
		if f := s.Widget().o.OnClose; f != nil {
			f()
		}
		if s.Mounted() && !inBuild {
			// Mark dirty, but only if mounted and not in a build.
			s.SetState(func() {})
		}
	}
}

func (s *rawMenuAnchorState) handleOpenRequest(position foundation.Opt[ui.Offset]) {
	s.Widget().o.OnOpenRequested(position, func() { s.open(position) })
}

func (s *rawMenuAnchorState) handleCloseRequest() {
	// Changes in the size of the media query cause the anchor to close during
	// DidChangeDependencies. When this happens, calling SetState during the
	// closing sequence would schedule a build during a build. This is avoided by
	// checking if we're in a build, and if so, scheduling the close for the next
	// frame.
	binding := WidgetsBindingOf(s.Context())
	if binding.Scheduler().SchedulerPhase() != scheduler.SchedulerPhasePersistentCallbacks {
		s.Widget().o.OnCloseRequested(func() { s.close(false) })
	} else {
		binding.RunPostFrame(func(time.Duration) {
			if s.Mounted() {
				s.Widget().o.OnCloseRequested(func() { s.close(false) })
			}
		})
	}
	// Parents will request close before children do.
	s.base.requestChildrenClose()
}

func (s *rawMenuAnchorState) buildOverlay(bc BuildContext, layoutInfo OverlayChildLayoutInfo) Widget {
	// Transform the anchor rectangle using the full transform matrix.
	anchorRect := layoutInfo.ChildPaintTransform.TransformRect(ui.NewRectFromLTWH(0, 0, layoutInfo.ChildSize.Width, layoutInfo.ChildSize.Height))
	info := RawMenuOverlayInfo{
		AnchorRect: anchorRect, OverlaySize: layoutInfo.OverlaySize, Position: s.menuPosition, TapRegionGroupID: s.base.root().menuController(),
	}
	return s.Widget().overlayBuilder(bc, info)
}

func (s *rawMenuAnchorState) buildAnchor(bc BuildContext) Widget {
	w := s.Widget()
	rootAnchor := s.base.root()
	child := NewShortcuts(menuTraversalShortcuts(),
		NewTapRegion(NewBuilder(func(bc BuildContext) Widget {
			if w.o.Builder != nil {
				if r := w.o.Builder(bc, s.menuController(), w.child); r != nil {
					return r
				}
			}
			if w.child != nil {
				return w.child
			}
			return NewSizedBox(nil)
		}), TapRegionOpts{GroupID: rootAnchor.menuController(), ConsumeOutsideTaps: rootAnchor.isOpen() && w.o.ConsumeOutsideTaps, OnTapOutside: s.base.handleOutsideTap}),
		ShortcutsOpts{ExcludeSemantics: true})
	location := OverlayChildLocationNearestOverlay
	if s.useRootOverlay() {
		location = OverlayChildLocationRootOverlay
	}
	return NewOverlayPortalOverlayChildLayoutBuilder(s.overlayController, s.buildOverlay, child, OverlayPortalOpts{OverlayLocation: location})
}

func (s *rawMenuAnchorState) Build(BuildContext) Widget { return s.base.build() }

// RawMenuAnchorGroup is a widget that groups menu anchors, so that one tap
// outside closes all the menus in the group. It has no menu of its own.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorGroup
type RawMenuAnchorGroup struct {
	StatefulWidget
	controller *MenuController
	child      Widget
}

// NewRawMenuAnchorGroup creates a group of menu anchors.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorGroup.new
func NewRawMenuAnchorGroup(controller *MenuController, child Widget, opts ...WidgetOpts) *RawMenuAnchorGroup {
	return &RawMenuAnchorGroup{NewStatefulWidgetBase(opts...), controller, child}
}

// CreateState creates the state.
//
// Flutter: widgets/raw_menu_anchor.dart RawMenuAnchorGroup.createState
func (w *RawMenuAnchorGroup) CreateState() State {
	s := &rawMenuAnchorGroupState{}
	s.base.self = s
	return s
}

type rawMenuAnchorGroupState struct {
	StateOf[*RawMenuAnchorGroup]
	base rawMenuAnchorBase
}

func (s *rawMenuAnchorGroupState) anchorBase() *rawMenuAnchorBase  { return &s.base }
func (s *rawMenuAnchorGroupState) mountedState() bool              { return s.Mounted() }
func (s *rawMenuAnchorGroupState) setStateOf(fn func())            { s.SetState(fn) }
func (s *rawMenuAnchorGroupState) contextOf() BuildContext         { return s.Context() }
func (s *rawMenuAnchorGroupState) menuController() *MenuController { return s.Widget().controller }
func (s *rawMenuAnchorGroupState) isOpen() bool {
	for _, c := range s.base.anchorChildren {
		if c.isOpen() {
			return true
		}
	}
	return false
}

func (s *rawMenuAnchorGroupState) InitState()             { s.base.initState() }
func (s *rawMenuAnchorGroupState) DidChangeDependencies() { s.base.didChangeDependencies() }
func (s *rawMenuAnchorGroupState) Dispose()               { s.base.dispose() }

func (s *rawMenuAnchorGroupState) DidUpdateWidget(old *RawMenuAnchorGroup) {
	if old.controller != s.Widget().controller {
		old.controller.detach(s)
		s.Widget().controller.attach(s)
	}
}

func (s *rawMenuAnchorGroupState) close(inDispose bool) {
	if !s.isOpen() {
		return
	}
	s.base.closeChildren(inDispose)
	if inDispose {
		return
	}
	binding := WidgetsBindingOf(s.Context())
	if binding.Scheduler().SchedulerPhase() != scheduler.SchedulerPhasePersistentCallbacks {
		s.SetState(func() {})
	} else {
		binding.RunPostFrame(func(time.Duration) {
			if s.Mounted() {
				s.SetState(func() {})
			}
		})
	}
}

// open does nothing: menu nodes are always open.
func (s *rawMenuAnchorGroupState) open(foundation.Opt[ui.Offset]) {}

func (s *rawMenuAnchorGroupState) handleCloseRequest() { s.base.requestChildrenClose() }

func (s *rawMenuAnchorGroupState) handleOpenRequest(position foundation.Opt[ui.Offset]) {
	s.open(position)
}

func (s *rawMenuAnchorGroupState) buildAnchor(BuildContext) Widget {
	return NewTapRegion(s.Widget().child, TapRegionOpts{GroupID: s.base.root().menuController(), OnTapOutside: s.base.handleOutsideTap})
}

func (s *rawMenuAnchorGroupState) Build(BuildContext) Widget { return s.base.build() }

// MenuController is the controller of a RawMenuAnchor or RawMenuAnchorGroup: it
// opens and closes the menu of the anchor it is attached to.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController
type MenuController struct {
	// anchor is the anchor that this controller controls; it is set when the
	// controller is attached to an anchor.
	anchor menuAnchor
}

// NewMenuController creates a controller.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.new
func NewMenuController() *MenuController { return &MenuController{} }

// IsOpen is whether the menu is open: the anchor's overlay is showing, or for a
// group, any of its anchors'.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.isOpen
func (c *MenuController) IsOpen() bool { return c.anchor != nil && c.anchor.isOpen() }

// Open requests that the menu open, at the position relative to the anchor if
// given. It panics with the tag MenuControllerNotAttached in debug builds when
// the controller isn't attached to an anchor.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.open
func (c *MenuController) Open(position ...ui.Offset) {
	c.assertAttached("Open")
	if c.anchor == nil {
		return
	}
	p := foundation.None[ui.Offset]()
	if len(position) > 0 {
		p = foundation.Some(position[0])
	}
	c.anchor.handleOpenRequest(p)
}

// Close requests that the menu and its submenus close. It does nothing when the
// controller isn't attached.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.close
func (c *MenuController) Close() {
	if c.anchor != nil {
		c.anchor.handleCloseRequest()
	}
}

// CloseChildren requests that the submenus of the menu close.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.closeChildren
func (c *MenuController) CloseChildren() {
	c.assertAttached("CloseChildren")
	if c.anchor != nil {
		c.anchor.anchorBase().requestChildrenClose()
	}
}

func (c *MenuController) assertAttached(what string) {
	if foundation.DebugMode && c.anchor == nil {
		panic(errors.New(tagMenuControllerNotAttached, what+" was called on a MenuController that is not attached to an anchor"))
	}
}

func (c *MenuController) attach(anchor menuAnchor) { c.anchor = anchor }

func (c *MenuController) detach(anchor menuAnchor) {
	if c.anchor == anchor {
		c.anchor = nil
	}
}

// MenuControllerMaybeOf returns the MenuController of the closest anchor that
// encloses the context, or nil. It doesn't create a dependency.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.maybeOf
func MenuControllerMaybeOf(bc BuildContext) *MenuController {
	if s, ok := GetInherited[*menuControllerScope](bc); ok {
		return s.controller
	}
	return nil
}

// MenuControllerMaybeIsOpenOf returns whether the menu of the closest anchor
// that encloses the context is open, and whether there is such an anchor. It
// creates a dependency, so the context rebuilds when the menu opens or closes.
//
// Flutter: widgets/raw_menu_anchor.dart MenuController.maybeIsOpenOf
func MenuControllerMaybeIsOpenOf(bc BuildContext) (isOpen, ok bool) {
	if s, found := DependOn[*menuControllerScope](bc); found {
		return s.isOpen, true
	}
	return false, false
}

// DismissMenuAction is an action that closes the menu: it handles the
// DismissIntent for the root of the menus of a controller.
//
// Flutter: widgets/raw_menu_anchor.dart DismissMenuAction
type DismissMenuAction struct {
	DismissAction
	controller *MenuController
}

// NewDismissMenuAction creates the action.
//
// Flutter: widgets/raw_menu_anchor.dart DismissMenuAction.new
func NewDismissMenuAction(controller *MenuController) *DismissMenuAction {
	return newDismissMenuAction(controller)
}

func newDismissMenuAction(controller *MenuController) *DismissMenuAction {
	a := &DismissMenuAction{controller: controller}
	InitActionBase(&a.ActionBase, a)
	return a
}

// IntentType is DismissIntent.
//
// Flutter: widgets/raw_menu_anchor.dart DismissMenuAction
func (a *DismissMenuAction) IntentType() reflect.Type { return reflect.TypeFor[DismissIntent]() }

// Invoke closes the menus of the controller.
//
// Flutter: widgets/raw_menu_anchor.dart DismissMenuAction.invoke
func (a *DismissMenuAction) Invoke(Intent) any {
	a.controller.anchor.anchorBase().root().handleCloseRequest()
	return nil
}

// IsEnabled is whether the controller is attached to an anchor.
//
// Flutter: widgets/raw_menu_anchor.dart DismissMenuAction.isEnabled
func (a *DismissMenuAction) IsEnabled(Intent) bool { return a.controller.anchor != nil }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMenuAnchorChildAlreadyAdded   = "MenuAnchorChildAlreadyAdded"
	tagMenuAnchorChildNotFound       = "MenuAnchorChildNotFound"
	tagMenuControllerMultipleAnchors = "MenuControllerMultipleAnchors"
	tagMenuControllerNotAttached     = "MenuControllerNotAttached"
)
