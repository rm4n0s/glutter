package cupertino

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

type opRect = foundation.Opt[ui.Rect]

// contextMenuRoute is the route of the open menu: the child moves to its place and the sheet of the actions
// grows beside it, and then the static page stays, where the child can be dragged away to dismiss the menu.
//
// Deviation: Flutter shares the tweens of the rectangles between all the routes (static fields); each route has
// its own here, as there is no global mutable state (SPEC §12.10).
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute
type contextMenuRoute struct {
	widgets.PopupRouteBase
	actions           []widgets.Widget
	builder           CupertinoContextMenuBuilder
	childKey          *widgets.GlobalKey[widgets.State]
	sheetKey          *widgets.GlobalKey[widgets.State]
	location          contextMenuLocation
	barrierLabel      string
	externalOffstage  bool
	internalOffstage  bool
	scaleFactor       float64
	lastOrientation   widgets.Orientation
	previousChildRect ui.Rect
	scale             float64
	postFrame         func(func())

	rectTween, rectTweenReverse, sheetRectTween *animation.Tween[opRect]
	rectAnimatable, rectAnimatableReverse       animation.Animatable[opRect]
	sheetRectAnimatable, sheetRectAnimatableRev animation.Animatable[opRect]
	sheetScaleTween                             *animation.Tween[float64]
	sheetScaleAnimatable, sheetScaleAnimatRev   animation.Animatable[float64]
	opacityTween                                *animation.Tween[float64]
	sheetOpacity                                animation.Animation[float64]
	curvedAnimation, sheetOpacityCurved         *animation.CurvedAnimation
}

const contextMenuModalBarrierColor = 0x6604040F

func (r *contextMenuRoute) init() {
	r.childKey = widgets.NewGlobalKey[widgets.State]()
	r.sheetKey = widgets.NewGlobalKey[widgets.State]()
	r.scale = 1.0
	curve := animation.NewCurveTween(animation.Curves.EaseOutBack)
	curveReverse := animation.NewCurveTween(animation.Curves.EaseInBack)
	r.rectTween = animation.NewRectTween(foundation.None[ui.Rect](), foundation.None[ui.Rect]())
	r.rectTweenReverse = animation.NewRectTween(foundation.None[ui.Rect](), foundation.None[ui.Rect]())
	r.sheetRectTween = animation.NewRectTween(foundation.None[ui.Rect](), foundation.None[ui.Rect]())
	r.rectAnimatable = r.rectTween.Chain(curve)
	r.rectAnimatableReverse = r.rectTweenReverse.Chain(curveReverse)
	r.sheetRectAnimatable = r.sheetRectTween.Chain(curve)
	r.sheetRectAnimatableRev = r.sheetRectTween.Chain(curveReverse)
	r.sheetScaleTween = animation.NewTween(0.0, 0.0)
	r.sheetScaleAnimatable = r.sheetScaleTween.Chain(curve)
	r.sheetScaleAnimatRev = r.sheetScaleTween.Chain(curveReverse)
	r.opacityTween = animation.NewTween(0.0, 1.0)
	widgets.InitPopupRouteBase(&r.PopupRouteBase, r, widgets.ModalRouteOpts{Filter: ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 5.0, SigmaY: 5.0})})
}

// BarrierLabel is the label of the barrier.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.barrierLabel
func (r *contextMenuRoute) BarrierLabel() string { return r.barrierLabel }

// BarrierColor is the dark blue of the barrier.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.barrierColor
func (r *contextMenuRoute) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.Some(ui.NewColor(contextMenuModalBarrierColor))
}

// BarrierDismissible is true.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.barrierDismissible
func (r *contextMenuRoute) BarrierDismissible() bool { return true }

// SemanticsDismissible is false.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.semanticsDismissible
func (r *contextMenuRoute) SemanticsDismissible() bool { return false }

// TransitionDuration is 335 ms.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.transitionDuration
func (r *contextMenuRoute) TransitionDuration() time.Duration { return contextMenuModalPopupTransition }

func contextMenuScaledRect(bc widgets.BuildContext, scale float64) ui.Rect {
	childRect := contextMenuRect(bc)
	scaled := ui.Size{Width: childRect.Width() * scale, Height: childRect.Height() * scale}
	offset := ui.Offset{Dx: childRect.Left + (childRect.Width()-scaled.Width)/2, Dy: childRect.Top + (childRect.Height()-scaled.Height)/2}
	return ui.NewRectFromLTWH(offset.Dx, offset.Dy, scaled.Width, scaled.Height)
}

// contextMenuSheetAlignment is where the sheet scales from.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.getSheetAlignment
func contextMenuSheetAlignment(location contextMenuLocation, orientation widgets.Orientation) painting.AlignmentDirectional {
	switch location {
	case contextMenuLocationCenter:
		if orientation == widgets.OrientationLandscape {
			return painting.AlignmentDirectionalTopStart
		}
		return painting.AlignmentDirectionalTopCenter
	case contextMenuLocationRight:
		return painting.AlignmentDirectionalTopEnd
	}
	return painting.AlignmentDirectionalTopStart
}

func contextMenuSheetRectBegin(orientation widgets.Orientation, location contextMenuLocation, childRect, sheetRect ui.Rect) ui.Rect {
	portrait := orientation == widgets.OrientationPortrait
	var target ui.Offset
	var dx float64
	switch location {
	case contextMenuLocationCenter:
		target = childRect.TopCenter()
		if portrait {
			target = childRect.BottomCenter()
		}
		dx = sheetRect.Width() / 2
	case contextMenuLocationRight:
		target = childRect.TopRight()
		if portrait {
			target = childRect.BottomRight()
		}
		dx = sheetRect.Width()
	default:
		target = childRect.TopLeft()
		if portrait {
			target = childRect.BottomLeft()
		}
	}
	return ui.NewRectFromLTWH(target.Dx-dx, target.Dy, sheetRect.Width(), sheetRect.Height())
}

func (r *contextMenuRoute) onDismiss(bc widgets.BuildContext, scale, opacity float64) {
	r.scale = scale
	r.opacityTween.End = opacity
	r.sheetOpacityCurved = animation.NewCurvedAnimation(r.Animation(), animation.NewInterval(0.9, 1.0))
	r.sheetOpacity = r.opacityTween.Animate(r.sheetOpacityCurved)
	widgets.NavigatorOf(bc).Pop()
}

func (r *contextMenuRoute) updateTweenRects() {
	childRect := contextMenuScaledRect(r.childKey.CurrentContext(), r.scale)
	r.rectTween.Begin = foundation.Some(r.previousChildRect)
	r.rectTween.End = foundation.Some(childRect)
	original := ui.NewRectFromCenter(r.previousChildRect.Center(), r.previousChildRect.Width()/r.scaleFactor, r.previousChildRect.Height()/r.scaleFactor)
	sheetRect := contextMenuRect(r.sheetKey.CurrentContext())
	r.sheetRectTween.Begin = foundation.Some(contextMenuSheetRectBegin(r.lastOrientation, r.location, original, sheetRect))
	r.sheetRectTween.End = foundation.Some(sheetRect)
	r.sheetScaleTween.Begin = 0.0
	r.sheetScaleTween.End = r.scale
	r.rectTweenReverse.Begin = foundation.Some(original)
	r.rectTweenReverse.End = foundation.Some(childRect)
}

func (r *contextMenuRoute) setOffstageInternally() {
	r.PopupRouteBase.SetOffstage(r.externalOffstage || r.internalOffstage)
	r.ChangedInternalState()
}

// DidPop sets the rectangles that the route goes back through.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.didPop
func (r *contextMenuRoute) DidPop(result any) bool {
	r.updateTweenRects()
	return r.PopupRouteBase.DidPop(result)
}

// SetOffstage keeps the route offstage until its rectangles are known.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.offstage
func (r *contextMenuRoute) SetOffstage(value bool) {
	r.externalOffstage = value
	r.setOffstageInternally()
}

// DidPush builds the route offstage, and sets the rectangles after the first frame.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.didPush
func (r *contextMenuRoute) DidPush() *scheduler.TickerFuture {
	r.internalOffstage = true
	r.setOffstageInternally()
	r.postFrame(func() {
		r.updateTweenRects()
		r.internalOffstage = false
		r.setOffstageInternally()
	})
	return r.PopupRouteBase.DidPush()
}

// CreateAnimation makes the animation of the route, and the opacity of the sheet that follows it.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.createAnimation
func (r *contextMenuRoute) CreateAnimation() animation.Animation[float64] {
	a := r.PopupRouteBase.CreateAnimation()
	if r.curvedAnimation == nil || r.curvedAnimation.Parent != a {
		if r.curvedAnimation != nil {
			r.curvedAnimation.Dispose()
		}
		r.curvedAnimation = animation.NewCurvedAnimation(a, animation.Curves.Linear)
	}
	r.sheetOpacity = r.opacityTween.Animate(r.curvedAnimation)
	return a
}

// BuildPage builds nothing: the transitions build the page.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.buildPage
func (r *contextMenuRoute) BuildPage(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
	return widgets.NewSizedBoxShrink(nil)
}

func evalRect(a animation.Animatable[opRect], anim animation.Animation[float64]) ui.Rect {
	return a.Evaluate(anim).Value()
}

// BuildTransitions moves the child and the sheet while the route animates, and shows the static page after it.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.buildTransitions
func (r *contextMenuRoute) BuildTransitions(_ widgets.BuildContext, anim, _ animation.Animation[float64], _ widgets.Widget) widgets.Widget {
	return widgets.NewOrientationBuilder(func(bc widgets.BuildContext, orientation widgets.Orientation) widgets.Widget {
		r.lastOrientation = orientation
		if !animation.IsCompleted(anim) {
			reverse := anim.Status() == animation.AnimationStatusReverse
			rect, sheetRect, sheetScale := evalRect(r.rectAnimatable, anim), evalRect(r.sheetRectAnimatable, anim), r.sheetScaleAnimatable.Evaluate(anim)
			if reverse {
				rect, sheetRect, sheetScale = evalRect(r.rectAnimatableReverse, anim), evalRect(r.sheetRectAnimatableRev, anim), r.sheetScaleAnimatRev.Evaluate(anim)
			}
			return widgets.NewStack([]widgets.Widget{
				widgets.NewPositionedFromRect(sheetRect, widgets.NewFadeTransition(r.sheetOpacity,
					widgets.NewTransformScale(newContextMenuSheet(r.sheetKey, r.actions, r.location, orientation),
						widgets.TransformScaleOpts{TransformOpts: widgets.TransformOpts{Alignment: contextMenuSheetAlignment(r.location, orientation)},
							Scale: foundation.Some(sheetScale)}))),
				widgets.NewPositionedFromRect(rect, r.builder(bc, anim), widgets.WidgetOpts{Key: r.childKey}),
			})
		}
		return &contextMenuRouteStatic{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
			actions: r.actions, child: r.builder(bc, anim), childKey: r.childKey, location: r.location, onDismiss: r.onDismiss,
			orientation: orientation, sheetKey: r.sheetKey, childRect: r.previousChildRect}
	})
}

// Dispose frees the animations.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRoute.dispose
func (r *contextMenuRoute) Dispose() {
	if r.curvedAnimation != nil {
		r.curvedAnimation.Dispose()
	}
	if r.sheetOpacityCurved != nil {
		r.sheetOpacityCurved.Dispose()
	}
	r.PopupRouteBase.Dispose()
}

const (
	contextMenuStaticMinScale           = 0.8
	contextMenuStaticSheetScaleTreshold = 0.9
	contextMenuStaticPadding            = 20.0
	contextMenuStaticDamping            = 400.0
	contextMenuStaticMoveDuration       = 600 * time.Millisecond
)

// contextMenuRouteStatic is the page of the open menu: the child and the sheet at rest, and the drag that dismisses
// them.
//
// Flutter: cupertino/context_menu.dart _ContextMenuRouteStatic
type contextMenuRouteStatic struct {
	widgets.StatefulWidget
	actions     []widgets.Widget
	child       widgets.Widget
	childKey    foundation.Key
	location    contextMenuLocation
	onDismiss   func(bc widgets.BuildContext, scale, opacity float64)
	orientation widgets.Orientation
	sheetKey    foundation.Key
	childRect   ui.Rect
}

func (w *contextMenuRouteStatic) CreateState() widgets.State { return &contextMenuRouteStaticState{} }

type contextMenuRouteStaticState struct {
	widgets.StateOf[*contextMenuRouteStatic]
	widgets.TickerProvider
	dragOffset           ui.Offset
	lastScale            float64
	moveController       *animation.AnimationController
	moveCurved           *animation.CurvedAnimation
	sheetController      *animation.AnimationController
	sheetCurved          *animation.CurvedAnimation
	moveAnimation        animation.Animation[ui.Offset]
	sheetScaleAnimation  animation.Animation[float64]
	sheetOpacityAnimator animation.Animation[float64]
	moveListener         foundation.ListenerHandle
	hasMoveListener      bool
	flingListener        animation.StatusListenerHandle
	hasFlingListener     bool
}

func contextMenuStaticScale(maxDragDistance, dy float64) float64 {
	dyDirectional := dy
	if dy > 0 {
		dyDirectional = -dy
	}
	return math.Max(contextMenuStaticMinScale, (maxDragDistance+dyDirectional)/maxDragDistance)
}

func (s *contextMenuRouteStaticState) onPanStart(gestures.DragStartDetails) {
	s.moveController.SetValue(1.0)
	s.setDragOffset(ui.Offset{})
}

func (s *contextMenuRouteStaticState) onPanUpdate(d gestures.DragUpdateDetails) {
	s.setDragOffset(s.dragOffset.Add(d.Delta))
}

func (s *contextMenuRouteStaticState) onPanEnd(d gestures.DragEndDetails) {
	if math.Abs(d.Velocity.PixelsPerSecond.Dy) >= gestures.MinFlingVelocity {
		flingIsAway := d.Velocity.PixelsPerSecond.Dy > 0
		finalPosition := 0.0
		if flingIsAway {
			finalPosition = s.moveAnimation.Value().Dy + 100.0
		}
		if flingIsAway && s.sheetController.Status() != animation.AnimationStatusForward {
			s.sheetController.Forward()
		} else if !flingIsAway && s.sheetController.Status() != animation.AnimationStatusReverse {
			s.sheetController.Reverse()
		}
		s.moveAnimation = animation.NewTween(ui.Offset{Dy: s.moveAnimation.Value().Dy}, ui.Offset{Dy: finalPosition}).Animate(s.moveController)
		s.moveController.Reset()
		s.moveController.Duration = foundation.Some(64 * time.Millisecond)
		s.moveController.Forward()
		if s.hasFlingListener {
			s.moveController.RemoveStatusListener(s.flingListener)
		}
		s.flingListener = s.moveController.AddStatusListener(s.flingStatusListener)
		s.hasFlingListener = true
		return
	}
	if s.lastScale == contextMenuStaticMinScale {
		s.Widget().onDismiss(s.Context(), s.lastScale, s.sheetOpacityAnimator.Value())
		return
	}
	if s.hasMoveListener {
		s.moveController.RemoveListener(s.moveListener)
	}
	s.moveListener = s.moveController.AddListener(s.moveListenerFunc)
	s.hasMoveListener = true
	s.moveController.Reverse()
}

func (s *contextMenuRouteStaticState) moveListenerFunc() {
	if s.lastScale > contextMenuStaticSheetScaleTreshold {
		s.moveController.RemoveListener(s.moveListener)
		s.hasMoveListener = false
		if !animation.IsDismissed[float64](s.sheetController) {
			s.sheetController.Reverse()
		}
	}
}

func (s *contextMenuRouteStaticState) flingStatusListener(status animation.AnimationStatus) {
	if !status.IsCompleted() {
		return
	}
	s.moveController.Duration = foundation.Some(contextMenuStaticMoveDuration)
	s.moveController.RemoveStatusListener(s.flingListener)
	s.hasFlingListener = false
	if s.moveAnimation.Value().Dy == 0.0 {
		return
	}
	s.Widget().onDismiss(s.Context(), s.lastScale, s.sheetOpacityAnimator.Value())
}

func (s *contextMenuRouteStaticState) setDragOffset(dragOffset ui.Offset) {
	endX := contextMenuStaticPadding * dragOffset.Dx / contextMenuStaticDamping
	endY := dragOffset.Dy
	if dragOffset.Dy < 0.0 {
		endY = contextMenuStaticPadding * dragOffset.Dy / contextMenuStaticDamping
	}
	s.SetState(func() {
		s.dragOffset = dragOffset
		s.moveAnimation = animation.NewTween(ui.Offset{}, ui.Offset{Dx: ui.ClampDouble(endX, -contextMenuStaticPadding, contextMenuStaticPadding), Dy: endY}).Animate(s.moveCurved)
		if s.lastScale <= contextMenuStaticSheetScaleTreshold && s.sheetController.Status() != animation.AnimationStatusForward && s.sheetScaleAnimation.Value() != 0.0 {
			s.sheetController.Forward()
		} else if s.lastScale > contextMenuStaticSheetScaleTreshold && s.sheetController.Status() != animation.AnimationStatusReverse && s.sheetScaleAnimation.Value() != 1.0 {
			s.sheetController.Reverse()
		}
	})
}

func (s *contextMenuRouteStaticState) getChild(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	screenSize := widgets.MediaQuerySizeOf(bc)
	padding := widgets.MediaQueryPaddingOf(bc)
	screenBounds := ui.NewRectFromLTWH(0, 0, screenSize.Width-padding.Left-padding.Right, screenSize.Height-padding.Top-padding.Bottom)
	sheet := widgets.NewAnimatedBuilder(s.sheetController, s.buildSheetAnimation,
		newContextMenuSheet(w.sheetKey, w.actions, w.location, w.orientation))
	return widgets.NewCustomMultiChildLayout(
		newContextMenuAlignedChildrenDelegate(w.childRect, screenBounds, w.orientation, w.location),
		[]widgets.Widget{
			widgets.NewLayoutId(contextMenuChildSlot, widgets.NewAnimatedBuilder(s.moveController, s.buildChildAnimation, w.child)),
			widgets.NewLayoutId(contextMenuSheetSlot, sheet),
		})
}

func (s *contextMenuRouteStaticState) buildSheetAnimation(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	w := s.Widget()
	return widgets.NewTransformScale(widgets.NewFadeTransition(s.sheetOpacityAnimator, child), widgets.TransformScaleOpts{
		TransformOpts: widgets.TransformOpts{Alignment: contextMenuSheetAlignment(w.location, w.orientation)}, Scale: foundation.Some(s.sheetScaleAnimation.Value())})
}

func (s *contextMenuRouteStaticState) buildChildAnimation(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
	s.lastScale = contextMenuStaticScale(widgets.MediaQueryHeightOf(bc), s.moveAnimation.Value().Dy)
	return widgets.NewTransformScale(child, widgets.TransformScaleOpts{TransformOpts: widgets.TransformOpts{Key: s.Widget().childKey}, Scale: foundation.Some(s.lastScale)})
}

func (s *contextMenuRouteStaticState) buildAnimation(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return widgets.NewTransformTranslate(s.moveAnimation.Value(), child)
}

func (s *contextMenuRouteStaticState) InitState() {
	s.lastScale = 1.0
	s.moveController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(contextMenuStaticMoveDuration), Value: foundation.Some(1.0)})
	s.moveCurved = animation.NewCurvedAnimation(s.moveController, animation.Curves.ElasticIn)
	s.sheetController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(100 * time.Millisecond), ReverseDuration: foundation.Some(300 * time.Millisecond)})
	s.sheetCurved = animation.NewCurvedAnimation(s.sheetController, animation.Curves.Linear, animation.Curves.EaseInBack)
	s.sheetScaleAnimation = animation.NewTween(1.0, 0.0).Animate(s.sheetCurved)
	s.sheetOpacityAnimator = animation.NewTween(1.0, 0.0).Animate(s.sheetController)
	s.moveAnimation = animation.NewTween(ui.Offset{}, ui.Offset{}).Animate(s.moveCurved)
	s.setDragOffset(ui.Offset{})
}

func (s *contextMenuRouteStaticState) Dispose() {
	s.moveController.Dispose()
	s.moveCurved.Dispose()
	s.sheetController.Dispose()
	s.sheetCurved.Dispose()
}

func (s *contextMenuRouteStaticState) Build(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewSafeArea(widgets.NewAlign(
		widgets.NewGestureDetector(widgets.NewAnimatedBuilder(s.moveController, s.buildAnimation, s.getChild(bc)),
			widgets.GestureDetectorOpts{OnPanEnd: s.onPanEnd, OnPanStart: s.onPanStart, OnPanUpdate: s.onPanUpdate}),
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft}))
}

const (
	contextMenuSheetWidth                   = 250.0
	contextMenuSheetScrollbarMainAxisMargin = 13.0
)

// contextMenuSheet is the actions of the menu in a rounded sheet, with a hairline between them.
//
// Flutter: cupertino/context_menu.dart _ContextMenuSheet
type contextMenuSheet struct {
	widgets.StatefulWidget
	actions     []widgets.Widget
	location    contextMenuLocation
	orientation widgets.Orientation
}

func newContextMenuSheet(key foundation.Key, actions []widgets.Widget, location contextMenuLocation, orientation widgets.Orientation) *contextMenuSheet {
	return &contextMenuSheet{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: key}), actions: actions, location: location, orientation: orientation}
}

func (w *contextMenuSheet) CreateState() widgets.State { return &contextMenuSheetState{} }

type contextMenuSheetState struct {
	widgets.StateOf[*contextMenuSheet]
	controller *widgets.ScrollControllerBase
}

func (s *contextMenuSheetState) InitState() { s.controller = widgets.NewScrollController() }
func (s *contextMenuSheetState) Dispose()   { s.controller.Dispose() }

func (s *contextMenuSheetState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	column := []widgets.Widget{w.actions[0]}
	for _, action := range w.actions[1:] {
		column = append(column, widgets.NewDecoratedBox(
			painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.Border{Top: painting.NewBorderSide(painting.BorderSideOpts{
				Color: foundation.Some(CupertinoDynamicColorResolve(contextMenuBorderColor, bc)), Width: foundation.Some(0.4)})}}),
			action, widgets.DecoratedBoxOpts{Position: rendering.DecorationPositionForeground}))
	}
	return widgets.NewSizedBox(widgets.NewIntrinsicHeight(widgets.NewClipRSuperellipse(
		widgets.NewColoredBox(CupertinoDynamicColorResolve(CupertinoContextMenuKBackgroundColor, bc),
			widgets.NewScrollConfiguration(widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
				NewCupertinoScrollbar(widgets.NewSingleChildScrollView(
					widgets.NewColumn(column, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch}),
					widgets.SingleChildScrollViewOpts{Controller: s.controller}),
					CupertinoScrollbarOpts{MainAxisMargin: foundation.Some(contextMenuSheetScrollbarMainAxisMargin), Controller: s.controller}))),
		widgets.ClipRSuperellipseOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(13.0))})),
		widgets.SizedBoxOpts{Width: foundation.Some(contextMenuSheetWidth)})
}

type contextMenuSlot int

const (
	contextMenuChildSlot contextMenuSlot = iota
	contextMenuSheetSlot
)

// contextMenuAlignedChildrenDelegate lays out the child and the sheet side by side, above each other in a portrait
// view, as close to the place of the pressed child as the screen allows.
//
// Flutter: cupertino/context_menu.dart _ContextMenuAlignedChildrenDelegate
type contextMenuAlignedChildrenDelegate struct {
	rendering.MultiChildLayoutDelegateBase
	targetRect, screenBounds ui.Rect
	orientation              widgets.Orientation
	location                 contextMenuLocation
}

func newContextMenuAlignedChildrenDelegate(targetRect, screenBounds ui.Rect, orientation widgets.Orientation, location contextMenuLocation) *contextMenuAlignedChildrenDelegate {
	return &contextMenuAlignedChildrenDelegate{MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(nil),
		targetRect: targetRect, screenBounds: screenBounds, orientation: orientation, location: location}
}

func (d *contextMenuAlignedChildrenDelegate) PerformLayout(size ui.Size) {
	constraints := rendering.NewBoxConstraintsLoose(size)
	availableHeightForChild := d.screenBounds.Height() - contextMenuStaticPadding
	availableWidth := d.screenBounds.Width() - contextMenuStaticPadding*2
	availableWidthForChild := availableWidth
	if d.orientation == widgets.OrientationLandscape {
		availableWidthForChild = availableWidth - contextMenuSheetWidth
	}
	childSize := d.LayoutChild(contextMenuChildSlot, constraints.CopyWith(rendering.BoxConstraintsCopy{
		MaxHeight: foundation.Some(availableHeightForChild), MaxWidth: foundation.Some(availableWidthForChild)}))
	availableHeightForMenu := availableHeightForChild
	if d.orientation == widgets.OrientationPortrait {
		availableHeightForMenu = availableHeightForChild - (childSize.Height + contextMenuStaticPadding)
	}
	menuSize := d.LayoutChild(contextMenuSheetSlot, constraints.CopyWith(rendering.BoxConstraintsCopy{MaxHeight: foundation.Some(availableHeightForMenu)}))

	var initialChildLeft, initialChildTop, maxClampedLeft, maxClampedTop float64
	var secondChildOffset ui.Offset
	menuBeforeChild := false
	if d.orientation == widgets.OrientationPortrait {
		totalHeight := childSize.Height + menuSize.Height + contextMenuStaticPadding
		totalWidth := childSize.Width + contextMenuStaticPadding
		initialChildLeft = d.targetRect.Center().Dx - childSize.Width/2
		initialChildTop = d.targetRect.Center().Dy - childSize.Height
		var secondChildDx float64
		switch d.location {
		case contextMenuLocationCenter:
			secondChildDx = childSize.Width/2 - menuSize.Width/2
		case contextMenuLocationRight:
			secondChildDx = childSize.Width - menuSize.Width
		}
		secondChildOffset = ui.Offset{Dx: secondChildDx, Dy: childSize.Height + contextMenuStaticPadding}
		maxClampedLeft = d.screenBounds.Right - totalWidth
		maxClampedTop = d.screenBounds.Bottom - totalHeight
	} else {
		menuBeforeChild = d.location == contextMenuLocationRight
		totalWidth := childSize.Width + menuSize.Width + contextMenuStaticPadding
		initialChildLeft = d.screenBounds.Center().Dx - totalWidth/2
		initialChildTop = d.screenBounds.Center().Dy - math.Max(childSize.Height, menuSize.Height)/2
		secondChildDx := childSize.Width
		if menuBeforeChild {
			secondChildDx = menuSize.Width
		}
		secondChildOffset = ui.Offset{Dx: secondChildDx + contextMenuStaticPadding}
		maxClampedLeft = d.screenBounds.Right - totalWidth
		maxClampedTop = d.screenBounds.Bottom
	}
	clampedLeft := ui.ClampDouble(initialChildLeft, d.screenBounds.Left+contextMenuStaticPadding, maxClampedLeft)
	clampedTop := ui.ClampDouble(initialChildTop, d.screenBounds.Top+contextMenuStaticPadding, maxClampedTop)
	firstPosition := ui.Offset{Dx: clampedLeft, Dy: clampedTop}
	secondPosition := firstPosition.Add(secondChildOffset)
	if menuBeforeChild {
		d.PositionChild(contextMenuChildSlot, secondPosition)
		d.PositionChild(contextMenuSheetSlot, firstPosition)
	} else {
		d.PositionChild(contextMenuChildSlot, firstPosition)
		d.PositionChild(contextMenuSheetSlot, secondPosition)
	}
}

func (d *contextMenuAlignedChildrenDelegate) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o, ok := old.(*contextMenuAlignedChildrenDelegate)
	return !ok || o.targetRect != d.targetRect || o.screenBounds != d.screenBounds || o.orientation != d.orientation || o.location != d.location
}
