// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"iter"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// OverlayPortal inserts its overlay child into an Overlay above it: the child
// of the portal is laid out and painted where the portal is, and the overlay
// child is laid out and painted by the Overlay, above the OverlayEntry that
// the portal is in. It is the way to show a floating widget (a tooltip, a menu,
// a dropdown) without managing an OverlayEntry by hand.
//
// OverlayPortal is inspired by the flutter_portal package. The render object of
// the overlay child is a direct child of the render object of the Overlay (the
// theater), so the overlay child does layout after both the theater and the
// portal's render object (the layout surrogate) have done theirs.

// overlayPortalClock is the source of the time stamps that order the overlay
// children: a later stamp paints above an earlier one.
//
// Deviation: Flutter keeps the clock in a static field of OverlayPortalController;
// Glutter has no such state, so the clock is an atomic counter that only hands
// out increasing numbers (SPEC §12.10), like the one of backdrop keys.
var overlayPortalClock atomic.Int64

// OverlayChildLocation is the Overlay that an OverlayPortal's overlay child goes
// into.
//
// Flutter: widgets/overlay.dart OverlayChildLocation
type OverlayChildLocation int

const (
	// OverlayChildLocationUnspecified is the default: the nearest Overlay above
	// the portal.
	//
	// Flutter: no counterpart (the zero value of the enum)
	OverlayChildLocationUnspecified OverlayChildLocation = iota
	// OverlayChildLocationNearestOverlay is the closest Overlay above the portal.
	//
	// Flutter: widgets/overlay.dart OverlayChildLocation.nearestOverlay
	OverlayChildLocationNearestOverlay
	// OverlayChildLocationRootOverlay is the furthest Overlay above the portal,
	// which is the one of the whole app (or of the closest LookupBoundary).
	//
	// Flutter: widgets/overlay.dart OverlayChildLocation.rootOverlay
	OverlayChildLocationRootOverlay
)

func (l OverlayChildLocation) orDefault() OverlayChildLocation {
	if l == OverlayChildLocationUnspecified {
		return OverlayChildLocationNearestOverlay
	}
	return l
}

// String is the name of the location.
func (l OverlayChildLocation) String() string {
	switch l {
	case OverlayChildLocationRootOverlay:
		return "rootOverlay"
	default:
		return "nearestOverlay"
	}
}

// OverlayChildLayoutInfo is what the builder of an
// OverlayPortal.overlayChildLayoutBuilder gets: the size of the portal's child,
// where it is painted in the Overlay, and the size of the Overlay.
//
// Flutter: widgets/overlay.dart OverlayChildLayoutInfo
type OverlayChildLayoutInfo struct {
	// ChildSize is the size of the portal's child in its own coordinates.
	//
	// Flutter: widgets/overlay.dart OverlayChildLayoutInfo.childSize
	ChildSize ui.Size
	// ChildPaintTransform is the paint transform of the portal's child, in the
	// coordinates of the target Overlay.
	//
	// Flutter: widgets/overlay.dart OverlayChildLayoutInfo.childPaintTransform
	ChildPaintTransform ui.Matrix4
	// OverlaySize is the size of the target Overlay in its own coordinates.
	//
	// Flutter: widgets/overlay.dart OverlayChildLayoutInfo.overlaySize
	OverlaySize ui.Size
}

// OverlayChildLayoutBuilder builds the overlay child from the layout information
// of the portal.
//
// Flutter: widgets/overlay.dart OverlayChildLayoutBuilder
type OverlayChildLayoutBuilder func(bc BuildContext, info OverlayChildLayoutInfo) Widget

// OverlayPortalControllerOpts are the optional parameters of
// NewOverlayPortalController.
//
// Flutter: widgets/overlay.dart OverlayPortalController.new
type OverlayPortalControllerOpts struct {
	// DebugLabel names the controller in messages.
	DebugLabel string
}

// OverlayPortalController shows, hides and brings to the top the overlay child of
// an OverlayPortal.
//
// A controller can be given to at most one OverlayPortal at a time. When it moves
// from one portal to another, its IsShowing state does not carry over. Show and
// Hide can be called before the controller is given to a portal, but should not
// be called while the widget tree is being rebuilt.
//
// Flutter: widgets/overlay.dart OverlayPortalController
type OverlayPortalController struct {
	attachTarget *overlayPortalState
	// zOrderIndex lets Show and Hide be called when the controller is not yet
	// attached. Once it is attached, the zOrderIndex of the portal is the source
	// of truth, and this one is cleared.
	zOrderIndex foundation.Opt[int64]
	debugLabel  string
}

// NewOverlayPortalController creates a controller.
//
// Flutter: widgets/overlay.dart OverlayPortalController.new
func NewOverlayPortalController(opts ...OverlayPortalControllerOpts) *OverlayPortalController {
	return &OverlayPortalController{debugLabel: firstOpts(opts).DebugLabel}
}

// now is a unique and increasing time stamp.
//
// Flutter: widgets/overlay.dart OverlayPortalController._now
func (c *OverlayPortalController) now() int64 { return overlayPortalClock.Add(1) }

// Show shows the overlay child, above the other overlay children that are
// shown, and above the entry of the portal. It has no effect on the position of
// an overlay child that already is shown, except that it brings it to the top.
//
// Flutter: widgets/overlay.dart OverlayPortalController.show
func (c *OverlayPortalController) Show() {
	if state := c.attachTarget; state != nil {
		state.show(c.now())
	} else {
		c.zOrderIndex = foundation.Some(c.now())
	}
}

// Hide hides the overlay child. In debug builds a controller that is not
// attached to a portal panics with the tag OverlayPortalHideWithoutShow when it is
// not showing.
//
// Flutter: widgets/overlay.dart OverlayPortalController.hide
func (c *OverlayPortalController) Hide() {
	if state := c.attachTarget; state != nil {
		state.hide()
		return
	}
	if foundation.DebugMode && c.zOrderIndex.IsNone() {
		panic(errors.New("OverlayPortalHideWithoutShow", "Hide was called on an OverlayPortalController that is not attached and not showing",
			"controller", c.String()))
	}
	c.zOrderIndex = foundation.None[int64]()
}

// IsShowing is whether the overlay child is showing.
//
// Flutter: widgets/overlay.dart OverlayPortalController.isShowing
func (c *OverlayPortalController) IsShowing() bool {
	if state := c.attachTarget; state != nil {
		return state.zOrderIndex.IsSome()
	}
	return c.zOrderIndex.IsSome()
}

// Toggle shows the overlay child if it is hidden, and hides it if it is shown.
//
// Flutter: widgets/overlay.dart OverlayPortalController.toggle
func (c *OverlayPortalController) Toggle() {
	if c.IsShowing() {
		c.Hide()
	} else {
		c.Show()
	}
}

// String describes the controller.
//
// Flutter: widgets/overlay.dart OverlayPortalController.toString
func (c *OverlayPortalController) String() string {
	label := ""
	if c.debugLabel != "" {
		label = "(" + c.debugLabel + ")"
	}
	detached := ""
	if c.attachTarget == nil {
		detached = " DETACHED"
	}
	return "OverlayPortalController" + label + detached
}

// OverlayPortalOpts are the optional parameters of NewOverlayPortal.
//
// Flutter: widgets/overlay.dart OverlayPortal.new
type OverlayPortalOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OverlayLocation is the Overlay that the overlay child goes into; the
	// default is OverlayChildLocationNearestOverlay.
	OverlayLocation OverlayChildLocation
}

// OverlayPortal is a widget that renders its overlay child on an Overlay.
//
// When the controller shows the overlay child, the builder builds it and the
// closest Overlay above the portal (or the root one) paints it above the
// OverlayEntry that the portal is in. The overlay child keeps the context of the
// portal, so it can use the InheritedWidgets above the portal, and is laid out
// with the constraints of the Overlay.
//
// The overlay child must not be a Positioned directly: wrap it in a widget that
// positions it, such as an Align or a Positioned inside a Stack.
//
// Flutter: widgets/overlay.dart OverlayPortal
type OverlayPortal struct {
	StatefulWidget
	controller          *OverlayPortalController
	overlayChildBuilder WidgetBuilder
	child               Widget
	overlayLocation     OverlayChildLocation
}

// NewOverlayPortal creates a portal.
//
// Flutter: widgets/overlay.dart OverlayPortal.new
func NewOverlayPortal(controller *OverlayPortalController, overlayChildBuilder WidgetBuilder, child Widget, opts ...OverlayPortalOpts) *OverlayPortal {
	o := firstOpts(opts)
	return &OverlayPortal{
		StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), controller: controller,
		overlayChildBuilder: overlayChildBuilder, child: child, overlayLocation: o.OverlayLocation.orDefault(),
	}
}

// NewOverlayPortalTargetsRootOverlay creates a portal whose overlay child goes
// into the root Overlay.
//
// Deprecated: Flutter deprecated it in favor of OverlayPortal with
// OverlayChildLocationRootOverlay.
//
// Flutter: widgets/overlay.dart OverlayPortal.targetsRootOverlay
func NewOverlayPortalTargetsRootOverlay(controller *OverlayPortalController, overlayChildBuilder WidgetBuilder, child Widget, opts ...OverlayPortalOpts) *OverlayPortal {
	o := firstOpts(opts)
	o.OverlayLocation = OverlayChildLocationRootOverlay
	return NewOverlayPortal(controller, overlayChildBuilder, child, o)
}

// NewOverlayPortalOverlayChildLayoutBuilder creates a portal whose overlay child
// is built by a builder that gets the size and the paint transform of the
// portal's child, and the size of the Overlay. The builder runs during layout,
// every frame that the layout runs.
//
// Flutter: widgets/overlay.dart OverlayPortal.overlayChildLayoutBuilder
func NewOverlayPortalOverlayChildLayoutBuilder(controller *OverlayPortalController, overlayChildBuilder OverlayChildLayoutBuilder, child Widget, opts ...OverlayPortalOpts) *OverlayPortal {
	return NewOverlayPortal(controller, func(BuildContext) Widget {
		return &overlayChildLayoutBuilderWidget{NewRenderObjectWidgetBase(), overlayChildBuilder}
	}, child, opts...)
}

// Controller is the controller that shows and hides the overlay child.
//
// Flutter: widgets/overlay.dart OverlayPortal.controller
func (w *OverlayPortal) Controller() *OverlayPortalController { return w.controller }

// OverlayChildBuilder builds the overlay child.
//
// Flutter: widgets/overlay.dart OverlayPortal.overlayChildBuilder
func (w *OverlayPortal) OverlayChildBuilder() WidgetBuilder { return w.overlayChildBuilder }

// Child is the widget that the portal lays out and paints where it is.
//
// Flutter: widgets/overlay.dart OverlayPortal.child
func (w *OverlayPortal) Child() Widget { return w.child }

// OverlayLocation is the Overlay that the overlay child goes into.
//
// Flutter: widgets/overlay.dart OverlayPortal.overlayLocation
func (w *OverlayPortal) OverlayLocation() OverlayChildLocation { return w.overlayLocation }

// CreateState creates the state.
//
// Flutter: widgets/overlay.dart OverlayPortal.createState
func (w *OverlayPortal) CreateState() State {
	return &overlayPortalState{childModelMayHaveChanged: true}
}

// overlayPortalState is the state of an OverlayPortal.
//
// Flutter: widgets/overlay.dart _OverlayPortalState
type overlayPortalState struct {
	StateOf[*OverlayPortal]
	zOrderIndex foundation.Opt[int64]
	// childModelMayHaveChanged is set when the location of the overlay child may
	// have changed, so that the cache is checked. The framework compares slots
	// with ==, and an overlayEntryLocation is mutable and so compared by identity:
	// changing the slot can be slow, so the cache is kept when it can be.
	childModelMayHaveChanged bool
	locationCache            *overlayEntryLocation
}

func isTheSameLocation(cache *overlayEntryLocation, marker *renderTheaterMarker) bool {
	return cache.childModel == marker.overlayEntryWidgetState && cache.theater == marker.theater
}

// getLocation is the location of the overlay child within the overlay. The
// developer must call Show first, to get a unique time stamp of the interaction
// that decides the z-index of the overlay child.
//
// Flutter: widgets/overlay.dart _OverlayPortalState._getLocation
func (s *overlayPortalState) getLocation(bc BuildContext, zOrderIndex int64, overlayLocation OverlayChildLocation) *overlayEntryLocation {
	cached := s.locationCache
	var marker *renderTheaterMarker
	getMarker := func() *renderTheaterMarker {
		if marker == nil {
			marker = renderTheaterMarkerOf(bc, overlayLocation == OverlayChildLocationRootOverlay)
		}
		return marker
	}
	isCacheValid := cached != nil && (!s.childModelMayHaveChanged || isTheSameLocation(cached, getMarker()))
	s.childModelMayHaveChanged = false
	if isCacheValid {
		if foundation.DebugMode {
			cached.debugIsLocationValid()
		}
		return cached
	}
	// Otherwise invalidate the cache and create a new location.
	if cached != nil {
		cached.debugMarkLocationInvalid()
	}
	m := getMarker()
	s.locationCache = &overlayEntryLocation{zOrderIndex: zOrderIndex, childModel: m.overlayEntryWidgetState, theater: m.theater}
	return s.locationCache
}

func (s *overlayPortalState) InitState() { s.setupController(s.Widget().controller) }

func (s *overlayPortalState) setupController(controller *OverlayPortalController) {
	if foundation.DebugMode {
		if target := controller.attachTarget; target != nil && target != s {
			if el, ok := target.Context().(Element); ok && el.AsElement().DebugIsActive() {
				panic(errors.New("OverlayPortalControllerAttached", "failed to attach "+controller.String()+" to the OverlayPortal: it is already attached to another one",
					"controller", controller.String()))
			}
		}
	}
	controllerZOrderIndex, hasController := controller.zOrderIndex.Get()
	zOrderIndex, hasState := s.zOrderIndex.Get()
	if !hasState || (hasController && controllerZOrderIndex > zOrderIndex) {
		s.zOrderIndex = controller.zOrderIndex
	}
	controller.zOrderIndex = foundation.None[int64]()
	controller.attachTarget = s
}

func (s *overlayPortalState) DidChangeDependencies() { s.childModelMayHaveChanged = true }

func (s *overlayPortalState) DidUpdateWidget(oldWidget *OverlayPortal) {
	w := s.Widget()
	s.childModelMayHaveChanged = s.childModelMayHaveChanged || oldWidget.overlayLocation != w.overlayLocation
	if oldWidget.controller != w.controller {
		oldWidget.controller.attachTarget = nil
		s.setupController(w.controller)
	}
}

func (s *overlayPortalState) Activate() {
	if foundation.DebugMode && s.Widget().controller.attachTarget != s {
		panic(errors.New("OverlayPortalControllerDetached", "an OverlayPortal was activated while its controller is attached to another one",
			"controller", s.Widget().controller.String()))
	}
}

func (s *overlayPortalState) Dispose() {
	s.Widget().controller.attachTarget = nil
	if s.locationCache != nil {
		s.locationCache.debugMarkLocationInvalid()
	}
	s.locationCache = nil
}

func (s *overlayPortalState) debugCheckNotBuilding(method string) {
	if foundation.DebugMode && WidgetsBindingOf(s.Context()).Scheduler().SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
		panic(errors.New("OverlayPortalShowDuringBuild", "OverlayPortalController."+method+" should not be called during build",
			"controller", s.Widget().controller.String()))
	}
}

func (s *overlayPortalState) show(zOrderIndex int64) {
	s.debugCheckNotBuilding("Show")
	s.SetState(func() { s.zOrderIndex = foundation.Some(zOrderIndex) })
	if s.locationCache != nil {
		s.locationCache.debugMarkLocationInvalid()
	}
	s.locationCache = nil
}

func (s *overlayPortalState) hide() {
	s.debugCheckNotBuilding("Hide")
	s.SetState(func() { s.zOrderIndex = foundation.None[int64]() })
	if s.locationCache != nil {
		s.locationCache.debugMarkLocationInvalid()
	}
	s.locationCache = nil
}

func (s *overlayPortalState) Build(bc BuildContext) Widget {
	w := s.Widget()
	semanticsChild := NewSemantics(w.child, SemanticsOpts{TraversalParentIdentifier: s})
	zOrderIndex, showing := s.zOrderIndex.Get()
	if !showing {
		return &overlayPortalRenderWidget{RenderObjectWidget: NewRenderObjectWidgetBase(), child: semanticsChild}
	}
	overlayLocation := s.getLocation(bc, zOrderIndex, w.overlayLocation)
	overlayData := MediaQueryOf(overlayLocation.childModel.Context())
	data := MediaQueryOf(bc).CopyWith(MediaQueryDataCopyOpts{
		Padding: foundation.Some(overlayData.Padding), ViewInsets: foundation.Some(overlayData.ViewInsets), ViewPadding: foundation.Some(overlayData.ViewPadding),
	})
	return &overlayPortalRenderWidget{
		RenderObjectWidget: NewRenderObjectWidgetBase(),
		overlayLocation:    overlayLocation,
		overlayChild: &deferredLayout{
			SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(NewMediaQuery(data, NewBuilder(w.overlayChildBuilder))),
			childIdentifier:               s,
		},
		child: semanticsChild,
	}
}

// overlayEntryLocation is a cursor that points to a place in the child model of
// an Overlay (the overlay children that are painted above an OverlayEntry) and
// that inserts, removes and moves a renderDeferredLayoutBox to and from its
// theater. The box is painted above the OverlayEntry and below the entry that is
// above that one.
//
// The location is the slot of the overlay child of the portal. Three parts
// identify a place in an overlay: the theater, the child model (the entry) and
// the z-order index. As the location is mutable and compared by identity, one
// instance never stands for more than one place.
//
// Flutter: widgets/overlay.dart _OverlayEntryLocation
type overlayEntryLocation struct {
	zOrderIndex int64
	childModel  *overlayEntryWidgetState
	theater     *renderTheater

	overlayChildRenderBox *renderDeferredLayoutBox
	// prev and next are the neighbors in the list of the child model, which is
	// sorted by zOrderIndex; linked is whether the location is in that list.
	prev, next *overlayEntryLocation
	linked     bool
	// invalid is set by debugMarkLocationInvalid.
	invalid bool
}

func (l *overlayEntryLocation) addToChildModel(child *renderDeferredLayoutBox) {
	if foundation.DebugMode && l.overlayChildRenderBox != nil {
		panic(errors.New("OverlayLocationOccupied", "failed to add an overlay child: the location is occupied already",
			"location", l.String(), "occupant", foundation.DescribeIdentity(l.overlayChildRenderBox)))
	}
	l.overlayChildRenderBox = child
	l.childModel.add(l)
	l.theater.MarkNeedsPaint()
	l.theater.MarkNeedsCompositingBitsUpdate()
	l.theater.MarkNeedsSemanticsUpdate()
}

func (l *overlayEntryLocation) removeFromChildModel(child *renderDeferredLayoutBox) {
	if foundation.DebugMode && child != l.overlayChildRenderBox {
		panic(errors.New("OverlayLocationOccupantMismatch", "failed to remove an overlay child that is not the occupant of the location",
			"location", l.String()))
	}
	l.overlayChildRenderBox = nil
	l.childModel.remove(l)
	l.theater.MarkNeedsPaint()
	l.theater.MarkNeedsCompositingBitsUpdate()
	l.theater.MarkNeedsSemanticsUpdate()
}

func (l *overlayEntryLocation) addChild(child *renderDeferredLayoutBox) {
	l.debugIsLocationValid()
	l.addToChildModel(child)
	l.theater.addDeferredChild(child)
}

// removeChild is allowed even when the location is invalid.
func (l *overlayEntryLocation) removeChild(child *renderDeferredLayoutBox) {
	l.removeFromChildModel(child)
	l.theater.removeDeferredChild(child)
}

func (l *overlayEntryLocation) moveChild(child *renderDeferredLayoutBox, from *overlayEntryLocation) {
	l.debugIsLocationValid()
	fromTheater, fromModel := from.theater, from.childModel
	if fromTheater != l.theater {
		fromTheater.removeDeferredChild(child)
		l.theater.addDeferredChild(child)
	}
	if fromModel != l.childModel || from.zOrderIndex != l.zOrderIndex {
		from.removeFromChildModel(child)
		l.addToChildModel(child)
	}
}

// reattachFromLayoutSurrogate and detachFromLayoutSurrogate are called when the
// portal's render object attaches and detaches (for instance, during global key
// reparenting): the portal removes its overlay child from the theater when it
// detaches and puts it back when it attaches again.
func (l *overlayEntryLocation) reattachFromLayoutSurrogate(child *renderDeferredLayoutBox) {
	if foundation.DebugMode && l.overlayChildRenderBox != nil {
		panic(errors.New("OverlayLocationOccupied", "failed to reattach an overlay child: detachFromLayoutSurrogate must be called first",
			"location", l.String()))
	}
	l.theater.addDeferredChild(child)
	l.overlayChildRenderBox = child
}

func (l *overlayEntryLocation) detachFromLayoutSurrogate(child *renderDeferredLayoutBox) {
	l.theater.removeDeferredChild(child)
	l.overlayChildRenderBox = nil
}

// debugIsLocationValid panics with the tag OverlayLocationInvalid in debug
// builds if the location was marked invalid, which makes it unusable as the
// slot of an overlay child. removeChild and detachFromLayoutSurrogate are
// allowed on an invalid location, as they run when the portal is removed from
// the tree.
func (l *overlayEntryLocation) debugIsLocationValid() bool {
	if foundation.DebugMode && l.invalid {
		panic(errors.New("OverlayLocationInvalid", l.String()+" is already disposed"))
	}
	return true
}

// debugMarkLocationInvalid is not reversible.
func (l *overlayEntryLocation) debugMarkLocationInvalid() {
	l.debugIsLocationValid()
	if foundation.DebugMode {
		l.invalid = true
	}
}

func (l *overlayEntryLocation) String() string {
	s := "_OverlayEntryLocation[" + strconv.FormatInt(l.zOrderIndex, 10) + "]"
	if l.invalid {
		s += " (INVALID)"
	}
	return s
}

// add inserts the location into the list of the child model, which is sorted by
// zOrderIndex: after the last location whose index is not greater.
//
// Flutter: widgets/overlay.dart _OverlayEntryWidgetState._add
func (s *overlayEntryWidgetState) add(child *overlayEntryLocation) {
	if foundation.DebugMode && (!s.Mounted() || child.linked) {
		panic(errors.New("OverlayChildModelInvalid", "an overlay child was added to the child model of an entry that is not mounted, or twice",
			"location", child.String()))
	}
	insertPosition := s.lastSibling
	for insertPosition != nil && insertPosition.zOrderIndex > child.zOrderIndex {
		insertPosition = insertPosition.prev
	}
	child.linked = true
	if insertPosition == nil {
		child.prev, child.next = nil, s.firstSibling
		if s.firstSibling != nil {
			s.firstSibling.prev = child
		} else {
			s.lastSibling = child
		}
		s.firstSibling = child
		return
	}
	child.prev, child.next = insertPosition, insertPosition.next
	if insertPosition.next != nil {
		insertPosition.next.prev = child
	} else {
		s.lastSibling = child
	}
	insertPosition.next = child
}

// remove takes the location out of the list of the child model.
//
// Flutter: widgets/overlay.dart _OverlayEntryWidgetState._remove
func (s *overlayEntryWidgetState) remove(child *overlayEntryLocation) {
	if !child.linked {
		if foundation.DebugMode {
			panic(errors.New("OverlayChildModelInvalid", "an overlay child was removed from a child model that does not have it",
				"location", child.String()))
		}
		return
	}
	if child.prev != nil {
		child.prev.next = child.next
	} else {
		s.firstSibling = child.next
	}
	if child.next != nil {
		child.next.prev = child.prev
	} else {
		s.lastSibling = child.prev
	}
	child.prev, child.next, child.linked = nil, nil, false
}

// childIterable traverses the overlay children in paint order (from the farthest
// to the closest to the user), or in the reverse order. It stays valid while the
// child model is mutated: adding, removing and moving deferred children is
// allowed during the layout of the theater, though the affected children do not
// have to be laid out in the same pass.
//
// Flutter: widgets/overlay.dart _OverlayEntryWidgetState._createChildIterable
func (s *overlayEntryWidgetState) childIterable(reversed bool) iter.Seq[*renderDeferredLayoutBox] {
	return func(yield func(*renderDeferredLayoutBox) bool) {
		candidate := s.firstSibling
		if reversed {
			candidate = s.lastSibling
		}
		for candidate != nil {
			box := candidate.overlayChildRenderBox
			if reversed {
				candidate = candidate.prev
			} else {
				candidate = candidate.next
			}
			if box != nil && !yield(box) {
				return
			}
		}
	}
}

func overlaySlot(l *overlayEntryLocation) any {
	if l == nil {
		return nil
	}
	return l
}

// overlayPortalRenderWidget is the widget of the render object that is the layout
// surrogate of the overlay child: it lays out the child of the portal and tells
// the theater when to lay out the overlay child.
//
// Flutter: widgets/overlay.dart _OverlayPortal
type overlayPortalRenderWidget struct {
	RenderObjectWidget
	overlayChild    Widget
	child           Widget
	overlayLocation *overlayEntryLocation
}

func (w *overlayPortalRenderWidget) CreateElement() Element {
	e := &overlayPortalElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

func (w *overlayPortalRenderWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderLayoutSurrogateProxyBox{overlayLocation: w.overlayLocation}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *overlayPortalRenderWidget) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*renderLayoutSurrogateProxyBox).overlayLocation = w.overlayLocation
}

// overlayPortalElement is the element of overlayPortalRenderWidget. Its child
// is the portal's child, and its overlay child is inserted into the theater.
//
// Flutter: widgets/overlay.dart _OverlayPortalElement
type overlayPortalElement struct {
	RenderObjectElement
	overlayChild Element
	child        Element
}

func (e *overlayPortalElement) surrogate() *renderLayoutSurrogateProxyBox {
	return e.renderObject.(*renderLayoutSurrogateProxyBox)
}

func (e *overlayPortalElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	w := e.widget.(*overlayPortalRenderWidget)
	e.child = e.UpdateChild(e.child, w.child, nil)
	e.overlayChild = e.UpdateChild(e.overlayChild, w.overlayChild, overlaySlot(w.overlayLocation))
}

func (e *overlayPortalElement) Update(newWidget Widget) {
	e.RenderObjectElement.Update(newWidget)
	w := newWidget.(*overlayPortalRenderWidget)
	e.child = e.UpdateChild(e.child, w.child, nil)
	e.overlayChild = e.UpdateChild(e.overlayChild, w.overlayChild, overlaySlot(w.overlayLocation))
}

// ForgetChild drops the child. The overlay child element has no key, as the
// deferredLayout widget does not take one, so only the regular child can be taken
// during global key reparenting.
func (e *overlayPortalElement) ForgetChild(child Element) {
	if foundation.DebugMode && child != e.child {
		panic(errors.New("OverlayPortalForgetChild", "only the child of an OverlayPortal can be forgotten, not its overlay child"))
	}
	e.child = nil
	e.RenderObjectElement.ForgetChild(child)
}

func (e *overlayPortalElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
	if e.overlayChild != nil {
		visitor(e.overlayChild)
	}
}

func (e *overlayPortalElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	if foundation.DebugMode && child.Parent() != nil {
		panic(errors.New("RenderObjectHasParent", "the render object of an OverlayPortal's child already has a parent",
			"child", foundation.DescribeIdentity(child)))
	}
	r := e.surrogate()
	if slot != nil {
		// deferredLayoutChild is assigned in deferredLayout.CreateRenderObject.
		slot.(*overlayEntryLocation).addChild(child.(*renderDeferredLayoutBox))
		r.MarkNeedsSemanticsUpdate()
		return
	}
	r.SetChildObject(child)
}

// MoveRenderObjectChild moves the overlay child to another location. The
// deferredLayout widget has no key, so the overlay child and the regular child
// are never exchanged: both slots are locations.
func (e *overlayPortalElement) MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any) {
	n := newSlot.(*overlayEntryLocation)
	n.debugIsLocationValid()
	n.moveChild(child.(*renderDeferredLayoutBox), oldSlot.(*overlayEntryLocation))
	e.surrogate().MarkNeedsSemanticsUpdate()
}

func (e *overlayPortalElement) RemoveRenderObjectChild(child rendering.RenderObject, slot any) {
	r := e.surrogate()
	if slot == nil {
		r.SetChildObject(nil)
		return
	}
	slot.(*overlayEntryLocation).removeChild(child.(*renderDeferredLayoutBox))
	r.deferredLayoutChild = nil
	r.MarkNeedsSemanticsUpdate()
}

// deferredLayout is the widget of the overlay child's render object: a
// renderDeferredLayoutBox, which lays its child out after the theater and the
// layout surrogate have laid out.
//
// This widget must not be given a key: reparenting between the overlay child
// and the child is not supported.
//
// Flutter: widgets/overlay.dart _DeferredLayout
type deferredLayout struct {
	SingleChildRenderObjectWidget
	childIdentifier any
}

func (w *deferredLayout) CreateElement() Element { return NewSingleChildRenderObjectElement(w) }

func (w *deferredLayout) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	parent, _ := FindAncestorRenderObjectOfType[*renderLayoutSurrogateProxyBox](bc)
	r := &renderDeferredLayoutBox{layoutSurrogate: parent, childIdentifier: w.childIdentifier, needsLayoutFlag: true}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	parent.deferredLayoutChild = r
	return r
}

func (w *deferredLayout) UpdateRenderObject(bc BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderDeferredLayoutBox)
	if foundation.DebugMode {
		parent, _ := FindAncestorRenderObjectOfType[*renderLayoutSurrogateProxyBox](bc)
		if r.layoutSurrogate != parent || parent.deferredLayoutChild != r {
			panic(errors.New("OverlayDeferredLayoutSurrogateChanged", "the layout surrogate of an overlay child changed"))
		}
	}
	// Deviation: Flutter's setter compares the new identifier with the getter,
	// which returns the old one, so it never changes the identifier. The identifier
	// is the state of the portal, which does not change, so setting it is the same.
	r.childIdentifier = w.childIdentifier
}

// renderDeferredLayoutBox is a child of the theater that lays its own child out
// only after the sizes of the render objects from its layout surrogate to the
// theater are known. To that end:
//
//  1. It is a relayout boundary, and calling MarkNeedsLayout on it or adding it to
//     the theater as a child never dirties the theater. Instead, it is always
//     added to the pipeline owner's dirty list when it needs layout (even for the
//     initial layout, when it is first added to the tree).
//  2. Its Layout is overridden such that PerformLayout does nothing when it is
//     called from Layout, which keeps the theater from laying the subtree out too
//     early (the box may still be resized). Instead, MarkNeedsLayout is called from
//     Layout to schedule a layout update of this relayout boundary when needed.
//
// When the pipeline owner's FlushLayout calls it, it behaves like an Overlay that
// has only one entry.
//
// Flutter: widgets/overlay.dart _RenderDeferredLayoutBox
type renderDeferredLayoutBox struct {
	rendering.RenderProxyBox
	layoutSurrogate *renderLayoutSurrogateProxyBox
	childIdentifier any
	// needsLayoutFlag is Flutter's _needsLayout, which is also true before the
	// first layout.
	needsLayoutFlag         bool
	doingLayoutFromTreeWalk bool
}

func (r *renderDeferredLayoutBox) SetupParentData(child rendering.RenderObject) {
	theaterSetupParentData(child)
}

// theaterOf is the parent, which must be a theater.
func (r *renderDeferredLayoutBox) theaterOf() *renderTheater {
	th, ok := r.Parent().(*renderTheater)
	if !ok {
		panic(errors.New("OverlayDeferredBoxParent", "the parent of a deferred layout box is not the theater of an Overlay",
			"parent", foundation.DescribeIdentity(r.Parent())))
	}
	return th
}

func (r *renderDeferredLayoutBox) childrenInPaintOrder() iter.Seq[rendering.Box] {
	return func(yield func(rendering.Box) bool) {
		if child := r.Child(); child != nil {
			yield(child)
		}
	}
}

func (r *renderDeferredLayoutBox) childrenInHitTestOrder() iter.Seq[rendering.Box] {
	return r.childrenInPaintOrder()
}

func (r *renderDeferredLayoutBox) RedepthChildren() {
	// The layout surrogate can be adopted after this box enters the theater. Until
	// then, the surrogate has no owner and cannot redepth this child. Once the
	// surrogate is adopted, its own RedepthChildren restores the depth invariant.
	if r.layoutSurrogate.Attached() {
		r.layoutSurrogate.RedepthChild(r)
	}
	r.RenderProxyBox.RedepthChildren()
}

func (r *renderDeferredLayoutBox) SizedByParent() bool { return true }

func (r *renderDeferredLayoutBox) MarkNeedsLayout() {
	r.needsLayoutFlag = true
	r.RenderProxyBox.MarkNeedsLayout()
}

func (r *renderDeferredLayoutBox) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	return rendering.StackBaselineForChild(child, constraints.Biggest(), constraints, r.theaterOf().resolvedAlignment(), baseline)
}

func (r *renderDeferredLayoutBox) DebugLayoutParent() rendering.RenderObject {
	if !foundation.DebugMode {
		return nil
	}
	return r.layoutSurrogate
}

// doLayoutFrom lays the box out as its parent (the theater, or the layout
// surrogate) does, and then makes the box lay its subtree out through the dirty
// list.
func (r *renderDeferredLayoutBox) doLayoutFrom(treewalkParent rendering.RenderObject, constraints rendering.Constraints) {
	shouldAddToDirtyList := r.needsLayoutFlag || r.Constraints() != constraints
	if foundation.DebugMode && r.doingLayoutFromTreeWalk {
		panic(errors.New("OverlayDeferredLayoutReentrant", "a deferred layout box was laid out while it was laid out from a tree walk"))
	}
	r.doingLayoutFromTreeWalk = true
	r.RenderProxyBox.Layout(constraints)
	r.doingLayoutFromTreeWalk = false
	r.needsLayoutFlag = false
	if shouldAddToDirtyList {
		// Instead of laying this subtree out via the tree walk, add it to the dirty
		// list. This ensures that:
		//
		//  1. this node is laid out by the pipeline owner after the two nodes it
		//     depends on (the theater and the layout surrogate) are laid out, as it
		//     has a greater depth than its dependencies;
		//  2. when the child of this box starts layout, the nodes from the layout
		//     surrogate to the theater (exclusive) have finished layout, so the
		//     child can read their sizes and compute the paint transform of the
		//     regular child within the Overlay.
		//
		// Invoking MarkNeedsLayout as a layout callback lets this node be merged
		// back into the owner's dirty list in the right order, if it is not already
		// dirty, so that the subtree is not laid out twice.
		treewalkParent.AsRenderObject().InvokeLayoutCallback(func(rendering.Constraints) { r.MarkNeedsLayout() })
	}
}

// Layout ignores ParentUsesSize, as the box is sized by its parent.
func (r *renderDeferredLayoutBox) Layout(constraints rendering.Constraints, _ ...rendering.LayoutOpts) {
	r.doLayoutFrom(r.Parent(), constraints)
}

func (r *renderDeferredLayoutBox) PerformResize() { r.SetSize(r.BoxConstraints().Biggest()) }

func (r *renderDeferredLayoutBox) PerformLayout() {
	if r.doingLayoutFromTreeWalk {
		r.needsLayoutFlag = false
		return
	}
	// This method is invoked directly by FlushLayout, or by the PerformLayout of
	// the layout surrogate.
	child := r.Child()
	if child == nil {
		r.needsLayoutFlag = false
		return
	}
	if foundation.DebugMode && !r.BoxConstraints().IsTight() {
		panic(errors.New("OverlayDeferredConstraintsLoose", "a deferred layout box was laid out with constraints that are not tight",
			"constraints", r.BoxConstraints().String()))
	}
	theaterLayoutChild(r, child, r.BoxConstraints())
	r.needsLayoutFlag = false
}

func (r *renderDeferredLayoutBox) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return theaterDistanceToActualBaseline(r, baseline)
}

func (r *renderDeferredLayoutBox) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return theaterHitTestChildren(r, result, position)
}

func (r *renderDeferredLayoutBox) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	theaterPaint(r, context, offset)
}

func (r *renderDeferredLayoutBox) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	if r.childIdentifier != nil {
		config.SetTraversalChildIdentifier(r.childIdentifier)
	}
}

func (r *renderDeferredLayoutBox) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	offset := child.ParentData().(*rendering.StackParentData).Offset
	transform.Translate(float32(offset.Dx), float32(offset.Dy), 0)
}

// renderLayoutSurrogateProxyBox is a proxy box that makes sure its deferred
// layout child has a greater depth than itself. It attaches and detaches the box
// when it attaches and detaches from its pipeline owner, so that the attached
// status of the box is always in sync with both the surrogate and the theater.
//
// Flutter: widgets/overlay.dart _RenderLayoutSurrogateProxyBox
type renderLayoutSurrogateProxyBox struct {
	rendering.RenderProxyBox
	// deferredLayoutChild is set as soon as the deferredLayout widget creates it,
	// and only cleared when the widget is removed from the tree.
	deferredLayoutChild *renderDeferredLayoutBox
	overlayLocation     *overlayEntryLocation

	didDetachDeferredChild bool
}

func (r *renderLayoutSurrogateProxyBox) Attach(owner *rendering.PipelineOwner) {
	r.RenderProxyBox.Attach(owner)
	// If attach is called after didDetachDeferredChild is set then it is always
	// safe to put the deferred child back, as the theater must be an ancestor of
	// both render objects.
	if r.didDetachDeferredChild {
		r.didDetachDeferredChild = false
		r.overlayLocation.reattachFromLayoutSurrogate(r.deferredLayoutChild)
	}
}

func (r *renderLayoutSurrogateProxyBox) Detach() {
	// Detach the deferred child if this node is being detached, but only if the
	// theater is not detached already (the theater then detaches the child).
	if d := r.deferredLayoutChild; d != nil {
		if th, ok := d.Parent().(*renderTheater); ok && th.Attached() {
			r.overlayLocation.detachFromLayoutSurrogate(d)
			r.didDetachDeferredChild = true
		}
	}
	r.RenderProxyBox.Detach()
}

func (r *renderLayoutSurrogateProxyBox) RedepthChildren() {
	r.RenderProxyBox.RedepthChildren()
	// If the child is not attached yet, the theater (its real parent) redepths it
	// when it becomes attached.
	if d := r.deferredLayoutChild; d != nil && d.Attached() {
		r.RedepthChild(d)
	}
}

func (r *renderLayoutSurrogateProxyBox) PerformLayout() {
	r.RenderProxyBox.PerformLayout()
	deferredChild := r.deferredLayoutChild
	if deferredChild == nil {
		return
	}
	// To make sure that all ancestors' PerformLayout calls have returned when the
	// deferred child does layout, the deferred child is put in the dirty list if it
	// is dirty and its subtree is made unreachable by the layout tree walk.
	//
	// The deferred child is a relayout boundary but may not be in the dirty list if
	// it has never been laid out (the framework does not treat it as a boundary
	// yet). The code below makes sure that it is.
	th, ok := deferredChild.Parent().(*renderTheater)
	if !ok {
		return
	}
	// If the theater is laying out its size-determining child, its size is not
	// available yet. As the theater lays that child out first and the deferred child
	// is never size determining, there is nothing to do: the theater updates the
	// constraints of the deferred child and puts it in the dirty list if needed.
	if th.layingOutSizeDeterminingChild {
		return
	}
	theaterConstraints := th.BoxConstraints()
	var boxSize ui.Size
	if theaterConstraints.Biggest().IsFinite() {
		boxSize = theaterConstraints.Biggest()
	} else {
		boxSize = th.Size()
	}
	deferredChild.doLayoutFrom(r, rendering.NewBoxConstraintsTight(boxSize))
}

// overlayChildLayoutBuilderWidget is the widget of OverlayPortal's overlay child
// layout builder: it builds its child during layout from the OverlayChildLayoutInfo.
//
// Flutter: widgets/overlay.dart _OverlayChildLayoutBuilder
type overlayChildLayoutBuilderWidget struct {
	RenderObjectWidget
	builder OverlayChildLayoutBuilder
}

func (w *overlayChildLayoutBuilderWidget) buildLayout(bc BuildContext, info any) Widget {
	return w.builder(bc, info.(OverlayChildLayoutInfo))
}

func (w *overlayChildLayoutBuilderWidget) updateShouldRebuild(Widget) bool { return true }

func (w *overlayChildLayoutBuilderWidget) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	r := &renderOverlayChildLayoutBuilder{scheduler: WidgetsBindingOf(bc).Scheduler()}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	rendering.InitLayoutCallbackMixin(&r.LayoutCallbackMixin, r, r.layoutCallback)
	return r
}

func (w *overlayChildLayoutBuilderWidget) CreateElement() Element {
	e := &layoutBuilderElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// renderOverlayChildLayoutBuilder is a box that:
//   - has the same size and paint transform as its parent and its theater: the
//     three boxes describe the same rectangle on screen;
//   - is a relayout boundary, and is marked dirty for relayout every frame (but
//     only when a frame is already scheduled: the callback does not schedule a new
//     one);
//   - runs a layout callback in PerformLayout.
//
// Like the deferred layout box, it uses the stack layout algorithm, so developers
// can use Positioned.
//
// Flutter: widgets/overlay.dart _RenderLayoutBuilder
type renderOverlayChildLayoutBuilder struct {
	rendering.RenderProxyBox
	rendering.LayoutCallbackMixin
	scheduler *scheduler.SchedulerBinding
	callback  func(rendering.Constraints)
	// info is the layout information, from the last layout callback.
	info       foundation.Opt[OverlayChildLayoutInfo]
	callbackID int
}

func (r *renderOverlayChildLayoutBuilder) SetupParentData(child rendering.RenderObject) {
	theaterSetupParentData(child)
}

func (r *renderOverlayChildLayoutBuilder) updateCallback(f func(rendering.Constraints)) {
	hadCallback := r.callback != nil
	r.callback = f
	if !hadCallback {
		r.ScheduleLayoutCallback()
	}
}

func (r *renderOverlayChildLayoutBuilder) clearCallback()          { r.callback = nil }
func (r *renderOverlayChildLayoutBuilder) scheduleLayoutCallback() { r.ScheduleLayoutCallback() }

// layoutInfo is the information that the last layout callback computed. It
// panics with the tag OverlayChildLayoutInfoMissing before the first one.
func (r *renderOverlayChildLayoutBuilder) layoutInfo() any {
	info, ok := r.info.Get()
	if !ok {
		panic(errors.New("OverlayChildLayoutInfoMissing", "the layout information of an overlay child was read before the layout callback ran"))
	}
	return info
}

func (r *renderOverlayChildLayoutBuilder) childrenInPaintOrder() iter.Seq[rendering.Box] {
	return func(yield func(rendering.Box) bool) {
		if child := r.Child(); child != nil {
			yield(child)
		}
	}
}

func (r *renderOverlayChildLayoutBuilder) childrenInHitTestOrder() iter.Seq[rendering.Box] {
	return r.childrenInPaintOrder()
}

// theaterOf is the theater of the parent, which must be a deferred layout box.
func (r *renderOverlayChildLayoutBuilder) theaterOf() *renderTheater {
	parent, ok := r.Parent().(*renderDeferredLayoutBox)
	if !ok {
		panic(errors.New("OverlayLayoutBuilderParent", "the parent of an overlay child layout builder is not a deferred layout box",
			"parent", foundation.DescribeIdentity(r.Parent())))
	}
	return parent.theaterOf()
}

func (r *renderOverlayChildLayoutBuilder) SizedByParent() bool { return true }

func (r *renderOverlayChildLayoutBuilder) PerformResize() { r.SetSize(r.BoxConstraints().Biggest()) }

func (r *renderOverlayChildLayoutBuilder) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	offset := child.ParentData().(*rendering.StackParentData).Offset
	transform.Translate(float32(offset.Dx), float32(offset.Dy), 0)
}

// computeNewLayoutInfo is the size of the portal, the transform to the theater
// and the size of the theater.
func (r *renderOverlayChildLayoutBuilder) computeNewLayoutInfo() OverlayChildLayoutInfo {
	th := r.theaterOf()
	parent := r.Parent().(*renderDeferredLayoutBox)
	surrogate := parent.layoutSurrogate
	if foundation.DebugMode {
		for node := rendering.RenderObject(surrogate); node != nil && node != rendering.RenderObject(th); node = node.Parent() {
			if follower, ok := node.(*rendering.RenderFollowerLayer); ok {
				panic(errors.New("OverlayPaintTransformUnreliable",
					"The paint transform cannot be reliably computed because of RenderFollowerLayer(s). RenderFollowerLayer establishes its paint transform only after the layout phase; consider replacing the corresponding CompositedTransformFollower with OverlayPortal.overlayChildLayoutBuilder if possible.",
					"follower", foundation.DescribeIdentity(follower)))
			}
		}
	}
	paintTransform := surrogate.GetTransformTo(th)
	return OverlayChildLayoutInfo{ChildSize: surrogate.Size(), ChildPaintTransform: *paintTransform, OverlaySize: r.Size()}
}

func (r *renderOverlayChildLayoutBuilder) layoutCallback() {
	r.info = foundation.Some(r.computeNewLayoutInfo())
	if r.callback != nil {
		r.callback(r.Constraints())
	}
}

func (r *renderOverlayChildLayoutBuilder) PerformLayout() {
	r.RunLayoutCallback()
	if child := r.Child(); child != nil {
		theaterLayoutChild(r, child, r.BoxConstraints())
	}
	if r.callbackID == 0 {
		r.callbackID = r.scheduler.ScheduleFrameCallback(r.frameCallback, scheduler.ScheduleFrameCallbackOpts{NoNewFrame: true})
	}
}

func (r *renderOverlayChildLayoutBuilder) frameCallback(time.Duration) {
	r.callbackID = 0
	r.MarkNeedsLayout()
}

// The layout callback may change the live render tree during layout, so the
// intrinsic dimensions are neither available nor needed: this object is not
// reachable in intrinsic dimension calculations.
func (r *renderOverlayChildLayoutBuilder) ComputeMinIntrinsicWidth(float64) float64 {
	r.DebugCannotComputeDryLayout(speculativeLayoutErrorMessage)
	return 0
}

func (r *renderOverlayChildLayoutBuilder) ComputeMaxIntrinsicWidth(float64) float64 {
	r.DebugCannotComputeDryLayout(speculativeLayoutErrorMessage)
	return 0
}

func (r *renderOverlayChildLayoutBuilder) ComputeMinIntrinsicHeight(float64) float64 {
	r.DebugCannotComputeDryLayout(speculativeLayoutErrorMessage)
	return 0
}

func (r *renderOverlayChildLayoutBuilder) ComputeMaxIntrinsicHeight(float64) float64 {
	r.DebugCannotComputeDryLayout(speculativeLayoutErrorMessage)
	return 0
}

const speculativeLayoutErrorMessage = "This RenderObject should not be reachable in intrinsic dimension calculations."

func (r *renderOverlayChildLayoutBuilder) ComputeDryLayout(rendering.BoxConstraints) ui.Size {
	r.DebugCannotComputeDryLayout(speculativeLayoutErrorMessage)
	return ui.Size{}
}

func (r *renderOverlayChildLayoutBuilder) ComputeDryBaseline(rendering.BoxConstraints, ui.TextBaseline) foundation.Opt[float64] {
	r.DebugCannotComputeDryLayout("Calculating the dry baseline would require running the layout callback speculatively, which might mutate the live render object tree.")
	return foundation.None[float64]()
}

func (r *renderOverlayChildLayoutBuilder) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	return theaterDistanceToActualBaseline(r, baseline)
}

func (r *renderOverlayChildLayoutBuilder) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	return theaterHitTestChildren(r, result, position)
}

func (r *renderOverlayChildLayoutBuilder) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	theaterPaint(r, context, offset)
}

func (r *renderOverlayChildLayoutBuilder) Dispose() {
	if r.callbackID != 0 {
		r.scheduler.CancelFrameCallbackWithID(r.callbackID)
		r.callbackID = 0
	}
	r.RenderProxyBox.Dispose()
}
