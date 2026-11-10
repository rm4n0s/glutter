package cupertino

import (
	"math"
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoMenuEntry is an item of a CupertinoMenuAnchor that tells the menu whether it has a leading widget (so that
// the other items leave room for it) and whether it is a divider.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuEntry
type CupertinoMenuEntry interface {
	// HasLeading is whether the entry has a leading widget.
	//
	// Flutter: cupertino/menu_anchor.dart CupertinoMenuEntry.hasLeading
	HasLeading(bc widgets.BuildContext) bool
	// IsDivider is whether the entry is a divider.
	//
	// Flutter: cupertino/menu_anchor.dart CupertinoMenuEntry.isDivider
	IsDivider() bool
}

// menuAnchorScope tells the items of a menu whether any of them has a leading widget.
//
// Flutter: cupertino/menu_anchor.dart _AnchorScope
type menuAnchorScope struct {
	widgets.InheritedWidget
	hasLeading bool
}

func (w *menuAnchorScope) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(*menuAnchorScope).hasLeading != w.hasLeading
}

// CupertinoMenuAnchorMaybeHasLeadingOf is whether any item of the closest CupertinoMenuAnchor above the context has
// a leading widget, and false as the second result when there is no anchor.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.maybeHasLeadingOf
func CupertinoMenuAnchorMaybeHasLeadingOf(bc widgets.BuildContext) (hasLeading, ok bool) {
	if scope, found := widgets.DependOn[*menuAnchorScope](bc); found {
		return scope.hasLeading, true
	}
	return false, false
}

// CupertinoMenuAnchorOpts are the optional parameters of NewCupertinoMenuAnchor.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.new
type CupertinoMenuAnchorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller opens and closes the menu; the anchor makes its own when there is none.
	Controller *widgets.MenuController
	// OnOpen is called when the menu opens.
	OnOpen foundation.VoidCallback
	// OnClose is called when the menu has closed.
	OnClose foundation.VoidCallback
	// OnAnimationStatusChanged is called when the status of the animation of the menu changes.
	OnAnimationStatusChanged func(status animation.AnimationStatus)
	// Constraints are the constraints of the menu; the default is the width of iOS for the size of the screen.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// ConstrainCrossAxis makes the menu take the width that the overlay gives it, instead of its own.
	ConstrainCrossAxis bool
	// ConsumeOutsideTaps makes a tap outside the menu close it without reaching what is under it.
	ConsumeOutsideTaps bool
	// NotEnableSwipe turns off the swipe from the anchor over the items; the swipe is on by default.
	NotEnableSwipe bool
	// EnableLongPressToOpen opens the menu when the anchor is long pressed, and lets the same touch swipe over the
	// items. It needs the swipe.
	EnableLongPressToOpen bool
	// UseRootOverlay puts the menu in the root overlay.
	UseRootOverlay bool
	// OverlayPadding is the space that the menu keeps to the edges of the overlay; the default is 8 on every side.
	OverlayPadding painting.EdgeInsetsGeometry
	// Builder builds the anchor with the controller of the menu; the default is the child.
	Builder widgets.RawMenuAnchorChildBuilder
	// Child is the anchor, if there is no builder, or the child that the builder gets.
	Child widgets.Widget
	// ChildFocusNode is the focus node of the anchor: it gets the focus when the menu closes.
	ChildFocusNode *widgets.FocusNode
}

// CupertinoMenuAnchor is a menu as iOS 26 shows it: a rounded translucent surface that grows from the anchor, with
// items that can be reached by a swipe that starts at the anchor.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor
type CupertinoMenuAnchor struct {
	widgets.StatefulWidget
	menuChildren []widgets.Widget
	o            CupertinoMenuAnchorOpts
}

// NewCupertinoMenuAnchor creates a menu of the children, usually CupertinoMenuItems and CupertinoMenuDividers. In
// debug builds it panics with the tag CupertinoMenuAnchorLongPressWithoutSwipe when EnableLongPressToOpen is set
// and NotEnableSwipe too.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.new
func NewCupertinoMenuAnchor(menuChildren []widgets.Widget, opts ...CupertinoMenuAnchorOpts) *CupertinoMenuAnchor {
	o := firstOf(opts)
	if foundation.DebugMode && o.NotEnableSwipe && o.EnableLongPressToOpen {
		panic(errors.New(tagCupertinoMenuAnchorLongPressWithoutSwipe, "enableLongPressToOpen cannot be true if enableSwipe is false"))
	}
	if o.OverlayPadding == nil {
		o.OverlayPadding = painting.NewEdgeInsetsAll(8)
	}
	return &CupertinoMenuAnchor{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), menuChildren: menuChildren, o: o}
}

// MenuChildren are the items of the menu.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.menuChildren
func (w *CupertinoMenuAnchor) MenuChildren() []widgets.Widget { return w.menuChildren }

// Controller is the controller of the menu.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.controller
func (w *CupertinoMenuAnchor) Controller() *widgets.MenuController { return w.o.Controller }

// Child is the anchor.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.child
func (w *CupertinoMenuAnchor) Child() widgets.Widget { return w.o.Child }

// CreateState creates the state.
//
// Flutter: cupertino/menu_anchor.dart CupertinoMenuAnchor.createState
func (w *CupertinoMenuAnchor) CreateState() widgets.State { return &cupertinoMenuAnchorState{} }

const menuLongPressToOpenDuration = 400 * time.Millisecond

var (
	menuSpringTolerance = physics.Tolerance{Velocity: 0.1}
	menuForwardSpring   = physics.NewSpringDescriptionWithDurationAndBounce(physics.SpringDescriptionWithDurationAndBounceOpts{
		Duration: foundation.Some(337 * time.Millisecond), Bounce: 0.2})
	menuReverseSpring = physics.NewSpringDescriptionWithDurationAndBounce(physics.SpringDescriptionWithDurationAndBounceOpts{
		Duration: foundation.Some(409 * time.Millisecond)})
)

type cupertinoMenuAnchorState struct {
	widgets.StateOf[*CupertinoMenuAnchor]
	widgets.TickerProvider
	animationController *animation.AnimationController
	menuScopeNode       *widgets.FocusScopeNode
	swipeDistance       *foundation.ValueNotifier[float64]
	hasLeadingWidget    foundation.Opt[bool]
	internalController  *widgets.MenuController
	animationStatus     animation.AnimationStatus
}

func (s *cupertinoMenuAnchorState) menuController() *widgets.MenuController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.internalController
}

func (s *cupertinoMenuAnchorState) isOpenOrOpening() bool {
	return s.animationStatus.IsForwardOrCompleted()
}

func (s *cupertinoMenuAnchorState) enableSwipe() bool {
	return !s.Widget().o.NotEnableSwipe && s.animationStatus != animation.AnimationStatusReverse
}

func (s *cupertinoMenuAnchorState) InitState() {
	if s.Widget().o.Controller == nil {
		s.internalController = widgets.NewMenuController()
	}
	s.animationStatus = animation.AnimationStatusDismissed
	s.menuScopeNode = widgets.NewFocusScopeNode()
	s.swipeDistance = foundation.NewValueNotifier(0.0)
	s.animationController = animation.NewAnimationControllerUnbounded(s)
	s.animationController.AddStatusListener(s.handleAnimationStatusChange)
}

func (s *cupertinoMenuAnchorState) DidUpdateWidget(old *CupertinoMenuAnchor) {
	w := s.Widget()
	if old.o.Controller != w.o.Controller {
		if w.o.Controller != nil {
			s.internalController = nil
		} else {
			s.internalController = widgets.NewMenuController()
		}
	}
	if !sameWidgets(old.menuChildren, w.menuChildren) {
		s.hasLeadingWidget = foundation.Some(s.resolveHasLeading())
	}
}

func sameWidgets(a, b []widgets.Widget) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *cupertinoMenuAnchorState) DidChangeDependencies() {
	if s.hasLeadingWidget.IsNone() {
		s.hasLeadingWidget = foundation.Some(s.resolveHasLeading())
	}
}

func (s *cupertinoMenuAnchorState) Dispose() {
	s.menuScopeNode.Dispose()
	s.animationController.Stop()
	s.animationController.Dispose()
	s.internalController = nil
	s.swipeDistance.Dispose()
}

func (s *cupertinoMenuAnchorState) resolveHasLeading() bool {
	for _, child := range s.Widget().menuChildren {
		if entry, ok := child.(CupertinoMenuEntry); ok && entry.HasLeading(s.Context()) {
			return true
		}
	}
	return false
}

func (s *cupertinoMenuAnchorState) handleAnimationStatusChange(status animation.AnimationStatus) {
	s.SetState(func() { s.animationStatus = status })
	if f := s.Widget().o.OnAnimationStatusChanged; f != nil {
		f(status)
	}
}

func (s *cupertinoMenuAnchorState) handleSwipeDistanceChange(distance float64) {
	if !s.menuController().IsOpen() {
		return
	}
	s.swipeDistance.SetValue(distance)
}

func (s *cupertinoMenuAnchorState) handleAnchorSwipeStart() {
	if s.isOpenOrOpening() || !s.Widget().o.EnableLongPressToOpen {
		return
	}
	s.menuController().Open()
}

func (s *cupertinoMenuAnchorState) handleCloseRequested(hideMenu foundation.VoidCallback) {
	if s.animationStatus == animation.AnimationStatusReverse || s.animationStatus == animation.AnimationStatusDismissed {
		return
	}
	s.animationController.AnimateBackWith(physics.NewClampedSimulation(
		physics.NewSpringSimulation(menuReverseSpring, s.animationController.Value(), 0.0, 0.0, physics.SpringSimulationOpts{Tolerance: foundation.Some(menuSpringTolerance)}),
		physics.ClampedSimulationOpts{XMin: foundation.Some(0.0), XMax: foundation.Some(1.0)})).Then(hideMenu)
}

func (s *cupertinoMenuAnchorState) handleOpenRequested(_ foundation.Opt[ui.Offset], showOverlay foundation.VoidCallback) {
	showOverlay()
	if s.animationStatus == animation.AnimationStatusCompleted || s.animationStatus == animation.AnimationStatusForward {
		return
	}
	s.animationController.AnimateWith(physics.NewSpringSimulation(menuForwardSpring, s.animationController.Value(), 1, 0.5))
	widgets.FocusScopeOf(s.Context()).SetFirstFocus(s.menuScopeNode)
}

func (s *cupertinoMenuAnchorState) buildMenuOverlay(_ widgets.BuildContext, info widgets.RawMenuOverlayInfo) widgets.Widget {
	w := s.Widget()
	open := s.isOpenOrOpening()
	return widgets.NewExcludeSemantics(widgets.NewIgnorePointer(widgets.NewExcludeFocus(&menuOverlay{
		StatefulWidget:      widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
		children:            w.menuChildren,
		focusScopeNode:      s.menuScopeNode,
		consumeOutsideTaps:  w.o.ConsumeOutsideTaps,
		constrainCrossAxis:  w.o.ConstrainCrossAxis,
		constraints:         w.o.Constraints,
		overlaySize:         info.OverlaySize,
		overlayPadding:      w.o.OverlayPadding,
		anchorRect:          info.AnchorRect,
		anchorPosition:      info.Position,
		tapRegionGroupID:    info.TapRegionGroupID,
		visibilityAnimation: s.animationController.View(),
		swipeDistance:       s.swipeDistance,
	}, widgets.ExcludeFocusOpts{NotExcluding: open}), widgets.IgnorePointerOpts{Ignoring: foundation.Some(!open)}),
		widgets.ExcludeSemanticsOpts{Excluding: foundation.Some(!open)})
}

func (s *cupertinoMenuAnchorState) buildChild(bc widgets.BuildContext, _ *widgets.MenuController, _ widgets.Widget) widgets.Widget {
	w := s.Widget()
	var anchor widgets.Widget
	if w.o.Builder != nil {
		anchor = w.o.Builder(bc, s.menuController(), w.o.Child)
	}
	if anchor == nil {
		anchor = w.o.Child
	}
	if anchor == nil {
		anchor = widgets.NewSizedBoxShrink(nil)
	}
	if !w.o.EnableLongPressToOpen || !s.enableSwipe() {
		return anchor
	}
	return newMenuSwipeSurface(anchor, menuLongPressToOpenDuration, s.handleAnchorSwipeStart)
}

func (s *cupertinoMenuAnchorState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return newMenuSwipeRegion(s.enableSwipe(), s.handleSwipeDistanceChange,
		&menuAnchorScope{InheritedWidget: widgets.NewInheritedWidgetBase(widgets.NewRawMenuAnchor(s.menuController(), s.buildMenuOverlay, nil,
			widgets.RawMenuAnchorOpts{
				UseRootOverlay: w.o.UseRootOverlay, OnCloseRequested: s.handleCloseRequested, OnOpenRequested: s.handleOpenRequested,
				Builder: s.buildChild, ChildFocusNode: w.o.ChildFocusNode, ConsumeOutsideTaps: w.o.ConsumeOutsideTaps,
				OnClose: w.o.OnClose, OnOpen: w.o.OnOpen})), hasLeading: s.hasLeadingWidget.Value()})
}

// menuOverlay is the surface of the menu in the overlay: its size animation, its shadow, its position next to the
// anchor and the focus and keyboard handling.
//
// Flutter: cupertino/menu_anchor.dart _MenuOverlay
type menuOverlay struct {
	widgets.StatefulWidget
	children            []widgets.Widget
	focusScopeNode      *widgets.FocusScopeNode
	consumeOutsideTaps  bool
	constrainCrossAxis  bool
	constraints         foundation.Opt[rendering.BoxConstraints]
	overlaySize         ui.Size
	overlayPadding      painting.EdgeInsetsGeometry
	anchorRect          ui.Rect
	anchorPosition      foundation.Opt[ui.Offset]
	tapRegionGroupID    any
	visibilityAnimation animation.Animation[float64]
	swipeDistance       *foundation.ValueNotifier[float64]
}

func (w *menuOverlay) CreateState() widgets.State { return &menuOverlayState{} }

type menuFocusUpIntent struct{ widgets.DirectionalFocusIntent }
type menuFocusDownIntent struct{ widgets.DirectionalFocusIntent }
type menuFocusFirstIntent struct{ widgets.IntentBase }
type menuFocusLastIntent struct{ widgets.IntentBase }

var menuTraversalShortcuts = widgets.ShortcutMap{
	{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: menuFocusUpIntent{widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionUp}}},
	{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: menuFocusDownIntent{widgets.DirectionalFocusIntent{Direction: widgets.TraversalDirectionDown}}},
	{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyHome), Intent: menuFocusFirstIntent{}},
	{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEnd), Intent: menuFocusLastIntent{}},
}

type menuOverlayState struct {
	widgets.StateOf[*menuOverlay]
	widgets.TickerProvider
	widgets.WidgetsBindingObserverBase
	swipeAnimationController *animation.AnimationController
	scrollController         *widgets.ScrollControllerBase
	scaleAnimation           *animation.ProxyAnimation
	fadeAnimation            *animation.ProxyAnimation
	sizeAnimation            *animation.ProxyAnimation
	attachmentPointAlignment painting.Alignment
	attachmentPoint          ui.Offset
	menuAlignment            painting.Alignment
	children                 []widgets.Widget
	textDirection            ui.TextDirection
	hasTextDirection         bool
	swipeTargetDistance      float64
	swipeCurrentDistance     float64
	swipeVelocity            float64
	swipeTicker              *scheduler.Ticker
	swipeHandle              foundation.ListenerHandle
	binding                  *widgets.WidgetsBinding
	actions                  widgets.ActionMap
}

var menuAttachmentOffset = ui.Offset{Dy: 8}

func (s *menuOverlayState) InitState() {
	s.binding = widgets.WidgetsBindingOf(s.Context())
	s.binding.AddObserver(s)
	s.swipeAnimationController = animation.NewAnimationControllerUnbounded(s, animation.AnimationControllerOpts{Value: foundation.Some(1.0)})
	s.scrollController = widgets.NewScrollController()
	s.scaleAnimation = animation.NewProxyAnimation(nil)
	s.fadeAnimation = animation.NewProxyAnimation(nil)
	s.sizeAnimation = animation.NewProxyAnimation(nil)
	s.actions = widgets.ActionMap{
		reflect.TypeFor[menuFocusUpIntent]():    newMenuFocusAction[menuFocusUpIntent](true),
		reflect.TypeFor[menuFocusDownIntent]():  newMenuFocusAction[menuFocusDownIntent](false),
		reflect.TypeFor[menuFocusFirstIntent](): newMenuFocusEdgeAction[menuFocusFirstIntent](true),
		reflect.TypeFor[menuFocusLastIntent]():  newMenuFocusEdgeAction[menuFocusLastIntent](false),
	}
	s.swipeHandle = s.Widget().swipeDistance.AddListener(s.handleSwipeDistanceChanged)
	s.resolveChildren()
}

func (s *menuOverlayState) DidChangeDependencies() {
	newTextDirection := widgets.DirectionalityOf(s.Context())
	if !s.hasTextDirection || s.textDirection != newTextDirection {
		s.textDirection, s.hasTextDirection = newTextDirection, true
		s.resolvePosition()
	}
	s.resolveMotion()
}

func (s *menuOverlayState) DidUpdateWidget(old *menuOverlay) {
	w := s.Widget()
	if old.swipeDistance != w.swipeDistance {
		old.swipeDistance.RemoveListener(s.swipeHandle)
		s.swipeHandle = w.swipeDistance.AddListener(s.handleSwipeDistanceChanged)
	}
	if old.visibilityAnimation != w.visibilityAnimation {
		s.resolveMotion()
	}
	if old.anchorRect != w.anchorRect || old.anchorPosition != w.anchorPosition || old.overlaySize != w.overlaySize {
		s.resolvePosition()
	}
	if !sameWidgets(old.children, w.children) {
		s.resolveChildren()
	}
}

func (s *menuOverlayState) DidChangeAccessibilityFeatures() { s.resolveMotion() }

func (s *menuOverlayState) Dispose() {
	s.scrollController.Dispose()
	s.Widget().swipeDistance.RemoveListener(s.swipeHandle)
	if s.swipeTicker != nil {
		s.swipeTicker.Stop()
		s.swipeTicker.Dispose()
	}
	s.swipeAnimationController.Stop()
	s.swipeAnimationController.Dispose()
	s.scaleAnimation.SetParent(nil)
	s.fadeAnimation.SetParent(nil)
	s.sizeAnimation.SetParent(nil)
	s.binding.RemoveObserver(s)
}

func isMenuDivider(w widgets.Widget) bool {
	e, ok := w.(CupertinoMenuEntry)
	return ok && e.IsDivider()
}

// resolveChildren puts an implicit divider between the items, unless one of the two is a divider.
//
// Flutter: cupertino/menu_anchor.dart _MenuOverlayState._resolveChildren
func (s *menuOverlayState) resolveChildren() {
	src := s.Widget().children
	if len(src) == 0 {
		s.children = nil
		return
	}
	var children []widgets.Widget
	for i, child := range src {
		children = append(children, child)
		if i == len(src)-1 {
			break
		}
		if isMenuDivider(child) || isMenuDivider(src[i+1]) {
			continue
		}
		children = append(children, &menuImplicitDivider{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{})})
	}
	s.children = children
}

func menuFadeAnimation(parent animation.Animation[float64]) animation.Animation[float64] {
	return animation.Drive[float64](parent, animation.NewCurveTween(animation.Curves.EaseIn).Chain(newMenuClampTween(0, 1)))
}

func (s *menuOverlayState) resolveMotion() {
	features := widgets.ViewOf(s.Context()).PlatformDispatcher().AccessibilityFeatures()
	visibility := s.Widget().visibilityAnimation
	swipeScale := animation.NewTween(0.8, 1.0).Animate(s.swipeAnimationController.View())
	switch {
	case features.DisableAnimations:
		s.scaleAnimation.SetParent(animation.AlwaysComplete)
		s.fadeAnimation.SetParent(animation.AlwaysComplete)
		s.sizeAnimation.SetParent(animation.AlwaysComplete)
	case features.ReduceMotion:
		s.scaleAnimation.SetParent(swipeScale)
		s.sizeAnimation.SetParent(animation.AlwaysComplete)
		s.fadeAnimation.SetParent(menuFadeAnimation(visibility))
	default:
		s.scaleAnimation.SetParent(animation.NewCompoundAnimation[float64]("_AnimationProduct", visibility, swipeScale, func(a, b float64) float64 { return a * b }))
		s.sizeAnimation.SetParent(animation.NewTween(0.8, 1.0).Animate(visibility))
		s.fadeAnimation.SetParent(menuFadeAnimation(visibility))
	}
}

func (s *menuOverlayState) resolvePosition() {
	w := s.Widget()
	var anchorMidpoint ui.Offset
	if pos, ok := w.anchorPosition.Get(); ok {
		anchorMidpoint = w.anchorRect.TopLeft().Add(pos)
	} else {
		anchorMidpoint = w.anchorRect.Center()
	}
	xMidpointRatio := anchorMidpoint.Dx / w.overlaySize.Width
	yMidpointRatio := anchorMidpoint.Dy / w.overlaySize.Height
	dy := -1.0
	if yMidpointRatio < 0.55 {
		dy = 1
	}
	dx := 0.0
	switch {
	case xMidpointRatio < 0.4:
		dx = -1
	case xMidpointRatio > 0.6:
		dx = 1
	}
	s.menuAlignment = painting.Alignment{X: dx, Y: -dy}
	var transformOrigin ui.Offset
	if pos, ok := w.anchorPosition.Get(); ok {
		s.attachmentPoint = w.anchorRect.TopLeft().Add(pos)
		transformOrigin = s.attachmentPoint
	} else {
		offset := ui.Offset{Dx: menuAttachmentOffset.Dx * dy, Dy: menuAttachmentOffset.Dy * dy}
		s.attachmentPoint = painting.Alignment{X: dx, Y: dy}.WithinRect(w.anchorRect).Add(offset)
		transformOrigin = painting.Alignment{X: 0, Y: dy}.WithinRect(w.anchorRect).Add(offset)
	}
	xOriginRatio := transformOrigin.Dx / w.overlaySize.Width
	yOriginRatio := transformOrigin.Dy / w.overlaySize.Height
	s.attachmentPointAlignment = painting.Alignment{X: xOriginRatio*2 - 1, Y: yOriginRatio*2 - 1}
}

func (s *menuOverlayState) handleOutsideTap(*gestures.PointerDownEvent) {
	widgets.MenuControllerMaybeOf(s.Context()).Close()
}

func (s *menuOverlayState) handleSwipeDistanceChanged() {
	s.swipeTargetDistance = ui.ClampDouble(s.Widget().swipeDistance.Value(), 0, 150)
	if s.swipeCurrentDistance == s.swipeTargetDistance {
		return
	}
	if s.swipeTicker == nil {
		s.swipeTicker = s.CreateTicker(s.updateSwipeScale)
	}
	if !s.swipeTicker.IsActive() {
		s.swipeTicker.Start()
	}
}

func (s *menuOverlayState) updateSwipeScale(time.Duration) {
	const (
		maxVelocity                    = 20.0
		minVelocity                    = 8.0
		maxSwipeDistance               = 150.0
		accelerationRate               = 0.12
		decelerationDistanceThreshold  = 80.0
		remainingDistanceSnapThreshold = 1.0
		terminationDistanceThreshold   = 5.0
	)
	distance := s.swipeTargetDistance - s.swipeCurrentDistance
	absoluteDistance := math.Abs(distance)
	proximityFactor := math.Min(absoluteDistance/decelerationDistanceThreshold, 1.0)
	s.swipeVelocity += accelerationRate * proximityFactor
	s.swipeVelocity = ui.ClampDouble(s.swipeVelocity, minVelocity, maxVelocity)
	finalVelocity := s.swipeVelocity * proximityFactor
	sign := 0.0
	switch {
	case distance > 0:
		sign = 1
	case distance < 0:
		sign = -1
	}
	s.swipeCurrentDistance += sign * finalVelocity
	if absoluteDistance < remainingDistanceSnapThreshold {
		s.swipeCurrentDistance = s.swipeTargetDistance
		s.swipeVelocity = 0
		if s.swipeTargetDistance < terminationDistanceThreshold {
			s.swipeTicker.Stop()
		}
	}
	s.swipeAnimationController.SetValue(1 - s.swipeCurrentDistance/maxSwipeDistance)
}

func (s *menuOverlayState) buildAlign(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return widgets.NewAlign(child, widgets.AlignOpts{HeightFactor: foundation.Some(s.sizeAnimation.Value()), WidthFactor: foundation.Some(1.0), Alignment: painting.AlignmentTopCenter})
}

func (s *menuOverlayState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var constraints rendering.BoxConstraints
	if c, ok := w.constraints.Get(); ok {
		constraints = c
	} else {
		constraints = rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{
			Width: foundation.Some(menuWidthOf(widgets.MediaQueryWidthOf(bc), menuLargeTextModeEnabled(bc)))})
	}
	brightness := ui.BrightnessLight
	if b, ok := CupertinoThemeMaybeBrightnessOf(bc); ok {
		brightness = b
	}
	var child widgets.Widget = newMenuSwipeSurface(widgets.NewTapRegion(widgets.NewActions(s.actions, widgets.NewShortcuts(menuTraversalShortcuts,
		widgets.NewFocusScope(widgets.NewCustomPaint(widgets.NewFadeTransition(s.fadeAnimation,
			NewCupertinoPopupSurface(widgets.NewAnimatedBuilder(s.sizeAnimation, s.buildAlign, widgets.NewSemantics(
				widgets.NewConstrainedBox(constraints, widgets.NewSingleChildScrollView(
					widgets.NewColumn(s.children, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}),
					widgets.SingleChildScrollViewOpts{ClipBehavior: ui.ClipNone})),
				widgets.SemanticsOpts{ExplicitChildNodes: true, ScopesRoute: foundation.Some(true)}))),
			widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true}),
			widgets.CustomPaintOpts{Painter: newMenuShadowPainter(brightness, s.fadeAnimation)}),
			widgets.FocusScopeOpts{Node: w.focusScopeNode, DescendantsAreFocusable: foundation.Some(true), DescendantsAreTraversable: foundation.Some(true),
				CanRequestFocus: foundation.Some(true)}))),
		widgets.TapRegionOpts{GroupID: w.tapRegionGroupID, ConsumeOutsideTaps: w.consumeOutsideTaps, OnTapOutside: s.handleOutsideTap}), 0, nil)
	if !w.constrainCrossAxis {
		child = widgets.NewUnconstrainedBox(child, widgets.UnconstrainedBoxOpts{ClipBehavior: ui.ClipHardEdge,
			Alignment: painting.AlignmentDirectionalCenterStart, ConstrainedAxis: painting.AxisVertical})
	}
	return widgets.NewConstrainedBox(rendering.NewBoxConstraintsLoose(w.overlaySize),
		widgets.NewScaleTransition(s.scaleAnimation, widgets.NewValueListenableBuilder[float64](s.sizeAnimation, func(bc widgets.BuildContext, _ float64, child widgets.Widget) widgets.Widget {
			effectiveAnchorRect := w.anchorRect
			if w.anchorPosition.IsSome() {
				effectiveAnchorRect = ui.NewRectFromLTWH(s.attachmentPoint.Dx, s.attachmentPoint.Dy, 0, 0)
			}
			return widgets.NewCustomSingleChildLayout(&menuLayoutDelegate{
				anchorRect: effectiveAnchorRect, attachmentPoint: s.attachmentPoint, heightFactor: s.sizeAnimation.Value(),
				menuAlignment: s.menuAlignment, overlayPadding: w.overlayPadding.Resolve(s.textDirection)}, child)
		}, child), widgets.ScaleTransitionOpts{Alignment: s.attachmentPointAlignment}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoMenuAnchorLongPressWithoutSwipe = "CupertinoMenuAnchorLongPressWithoutSwipe"
