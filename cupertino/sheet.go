package cupertino

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// sheetDeviceCornerRadiusSmoothingFactor is applied to the top padding of the device, which approximates
	// the radius of its corners, to get a smoother end to the animation of the radius.
	sheetDeviceCornerRadiusSmoothingFactor = 0.9
	// sheetRoundedDeviceCornersThreshold is the radius (in logical pixels, after the smoothing) under which
	// the animation of the radius starts from zero.
	sheetRoundedDeviceCornersThreshold = 20.0
	// sheetTopGapRatio is the distance from the top of the open sheet to the top of the screen, as a ratio of
	// the height of the screen.
	sheetTopGapRatio = 0.08
	// sheetStretchedTopGapRatio is the least distance from the top of the sheet to the top of the screen: how
	// far the sheet can be pulled up before it snaps back.
	sheetStretchedTopGapRatio = 0.072
	// sheetMinFlingVelocity is the least velocity of a drag downwards that dismisses the sheet, in screen
	// heights per second.
	sheetMinFlingVelocity = 2.0
	// sheetDroppedDragAnimationDuration is how long the sheet animates when it is let go mid-drag.
	sheetDroppedDragAnimationDuration = 300 * time.Millisecond
	// sheetScaleFactor is the amount that the sheet in the background scales down.
	sheetScaleFactor = 0.0835
)

// The tweens of the sheet, as functions so that the package keeps no state.
func sheetBottomUpTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{Dx: 0, Dy: 1}, ui.Offset{})
}

func sheetBottomUpTweenWhenCoveringOtherSheet() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{Dx: 0, Dy: 1}, ui.Offset{Dx: 0, Dy: -0.02})
}

func sheetMidUpTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{}, ui.Offset{Dx: 0, Dy: -0.005})
}

func sheetTopDownTween() *animation.Tween[ui.Offset] {
	return animation.NewTween(ui.Offset{}, ui.Offset{Dx: 0, Dy: 0.07})
}

func sheetOpacityTween() *animation.Tween[float64] { return animation.NewTween(0.0, 0.10) }

func sheetScaleTween() *animation.Tween[float64] {
	return animation.NewTween(1.0, 1.0-sheetScaleFactor)
}

// ShowCupertinoSheetOpts are the optional parameters of ShowCupertinoSheet.
//
// Flutter: cupertino/sheet.dart showCupertinoSheet
type ShowCupertinoSheetOpts struct {
	// Builder builds the content of the sheet.
	Builder widgets.WidgetBuilder
	// ScrollableBuilder builds the content of the sheet with the controller of the scrollable that the sheet
	// drags. It is used instead of Builder when it is set.
	ScrollableBuilder widgets.ScrollableWidgetBuilder
	// UseNestedNavigation puts a Navigator in the sheet, so that the sheet can have routes of its own.
	UseNestedNavigation bool
	// NotEnableDrag is true to keep the user from dismissing the sheet by dragging it down.
	NotEnableDrag bool
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// TopGap is the gap between the top of the screen and the top of the sheet, as a ratio of the height
	// of the screen, from 0 to 0.9; the default is 0.08. A sheet with a gap of its own does not move the
	// route below it.
	TopGap foundation.Opt[float64]
	// ShowDragHandle is whether the sheet has a drag handle at the top.
	ShowDragHandle bool
}

// ShowCupertinoSheet presents a sheet on the root navigator: the content slides up from the bottom, and the
// route below it shrinks into the background. It returns a channel that gets the value the sheet is popped
// with, then is closed. In debug builds it panics with the tag CupertinoSheetTopGapOutOfRange when the top
// gap is not from 0 to 0.9, and with CupertinoSheetBuilderMissing when there is no builder.
//
// Flutter: cupertino/sheet.dart showCupertinoSheet
func ShowCupertinoSheet(bc widgets.BuildContext, opts ShowCupertinoSheetOpts) <-chan any {
	if foundation.DebugMode {
		if opts.TopGap.IsSome() && (opts.TopGap.Value() < 0 || opts.TopGap.Value() > 0.9) {
			panic(errors.New(tagCupertinoSheetTopGapOutOfRange, "topGap must be between 0.0 and 0.9", "topGap", opts.TopGap.Value()))
		}
		if opts.Builder == nil && opts.ScrollableBuilder == nil {
			panic(errors.New(tagCupertinoSheetBuilderMissing, "either Builder or ScrollableBuilder must be given"))
		}
	}
	route := NewCupertinoSheetRoute(CupertinoSheetRouteOpts{
		Builder: opts.Builder, ScrollableBuilder: opts.ScrollableBuilder, RouteSettings: opts.RouteSettings,
		NotEnableDrag: opts.NotEnableDrag, ShowDragHandle: opts.ShowDragHandle, TopGap: opts.TopGap,
	})
	navigator := widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: true})
	if !opts.UseNestedNavigation {
		return navigator.Push(route)
	}
	nestedKey := widgets.NewGlobalKey[widgets.State]()
	content := func(builder widgets.WidgetBuilder) widgets.Widget {
		return widgets.NewNavigatorPopHandler(
			widgets.NewNavigator(widgets.NavigatorOpts{
				Key:          nestedKey,
				InitialRoute: "/",
				OnGenerateInitialRoutes: func(*widgets.NavigatorState, string) []widgets.Route {
					return []widgets.Route{NewCupertinoPageRoute(func(bc widgets.BuildContext) widgets.Widget {
						return widgets.NewPopScope(builder(bc), widgets.PopScopeOpts{
							CanPop: foundation.Some(false),
							OnPopInvokedWithResult: func(didPop bool, result any) {
								if didPop {
									return
								}
								widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: true}).Pop(result)
							},
						})
					})}
				},
			}),
			widgets.NavigatorPopHandlerOpts{OnPopWithResult: func(any) {
				if cur, _ := nestedKey.CurrentState(); cur != nil {
					state, ok := cur.(*widgets.NavigatorState)
					if !ok {
						return
					}
					state.MaybePop()
				}
			}})
	}
	nested := NewCupertinoSheetRoute(CupertinoSheetRouteOpts{
		ScrollableBuilder: func(bc widgets.BuildContext, controller widgets.ScrollController) widgets.Widget {
			if opts.ScrollableBuilder != nil {
				return content(func(bc widgets.BuildContext) widgets.Widget { return opts.ScrollableBuilder(bc, controller) })
			}
			return content(opts.Builder)
		},
		RouteSettings: opts.RouteSettings, NotEnableDrag: opts.NotEnableDrag, ShowDragHandle: opts.ShowDragHandle, TopGap: opts.TopGap,
	})
	return navigator.Push(nested)
}

// CupertinoSheetTransition provides an iOS style sheet transition: the sheet slides up from the bottom
// and the route below shrinks and moves up (or down, for a fullscreen route) behind it.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition
type CupertinoSheetTransition struct {
	widgets.StatefulWidget
	primaryRouteAnimation   animation.Animation[float64]
	secondaryRouteAnimation animation.Animation[float64]
	child                   widgets.Widget
	linearTransition        bool
	topGap                  float64
}

// CupertinoSheetTransitionOpts are the optional parameters of NewCupertinoSheetTransition.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.new
type CupertinoSheetTransitionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TopGap is the gap between the top of the screen and the top of the sheet, as a ratio of the height
	// of the screen; the default is 0.08.
	TopGap foundation.Opt[float64]
}

// NewCupertinoSheetTransition creates the transition of the child. linearTransition is true while the
// user drags the sheet: the animation then follows the finger, with no easing.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.new
func NewCupertinoSheetTransition(primaryRouteAnimation, secondaryRouteAnimation animation.Animation[float64], linearTransition bool, child widgets.Widget, opts ...CupertinoSheetTransitionOpts) *CupertinoSheetTransition {
	o := firstOf(opts)
	return &CupertinoSheetTransition{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), primaryRouteAnimation,
		secondaryRouteAnimation, child, linearTransition, o.TopGap.OrElse(sheetTopGapRatio)}
}

// Child is the sheet that slides.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.child
func (w *CupertinoSheetTransition) Child() widgets.Widget { return w.child }

// TopGap is the gap between the top of the screen and the top of the sheet, as a ratio of the height of the
// screen.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.topGap
func (w *CupertinoSheetTransition) TopGap() float64 { return w.topGap }

// CupertinoSheetTransitionDelegatedTransition is the transition that the route below a sheet takes: the
// route shrinks, moves down a little, and gets rounded corners and a dim layer, and a sheet below the new one
// moves up and shrinks instead.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.delegateTransition
func CupertinoSheetTransitionDelegatedTransition(bc widgets.BuildContext, _, secondaryAnimation animation.Animation[float64], _ bool, child widgets.Widget) widgets.Widget {
	if CupertinoSheetRouteHasParentSheet(bc) {
		return delegatedCoverSheetSecondaryTransition(secondaryAnimation, child)
	}
	linear := widgets.NavigatorOf(bc).UserGestureInProgress()
	var curve, reverseCurve animation.Curve = animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear
	if linear {
		curve, reverseCurve = animation.Curves.Linear, animation.Curves.Linear
	}
	curved := animation.NewCurvedAnimation(secondaryAnimation, curve, reverseCurve)
	topPadding := 0.0
	if padding, ok := widgets.MediaQueryMaybeViewPaddingOf(bc); ok {
		topPadding = padding.Top
	}
	deviceCornerRadius := topPadding * sheetDeviceCornerRadiusSmoothingFactor
	roundedDeviceCorners := deviceCornerRadius > sheetRoundedDeviceCornersThreshold
	beginRadius := 0.0
	if roundedDeviceCorners {
		beginRadius = deviceCornerRadius
	}
	var begin, end painting.BorderRadiusGeometry = painting.NewBorderRadiusVertical(ui.NewRadiusCircular(beginRadius), ui.Radius{}), painting.NewBorderRadiusCircular(12)
	decorationTween := animation.NewTweenFunc(begin, end, painting.BorderRadiusGeometryLerp)
	radiusAnimation := animation.Drive[painting.BorderRadiusGeometry](curved, decorationTween)
	opacityAnimation := animation.Drive(curved, sheetOpacityTween())
	slideAnimation := animation.Drive(curved, sheetTopDownTween())
	scaleAnimation := animation.Drive(curved, sheetScaleTween())
	curved.Dispose()

	overlayColor := ui.NewColor(0xFF000000)
	if CupertinoThemeBrightnessOf(bc) == ui.BrightnessDark {
		overlayColor = ui.NewColor(0xFFc8c8c8)
	}
	contrastedChild := child
	if child != nil && !animation.IsDismissed(secondaryAnimation) {
		contrastedChild = widgets.NewStack([]widgets.Widget{
			child,
			widgets.NewFadeTransition(opacityAnimation, widgets.NewColoredBox(overlayColor, widgets.NewSizedBoxExpand(nil))),
		})
	}
	topGapHeight := widgets.MediaQuerySizeOf(bc).Height * sheetTopGapRatio
	return widgets.NewStack([]widgets.Widget{
		widgets.NewAnnotatedRegion(services.SystemUiOverlayStyle{StatusBarBrightness: ui.BrightnessDark, StatusBarIconBrightness: ui.BrightnessLight},
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(topGapHeight), Width: foundation.Some(infinity)})),
		widgets.NewSlideTransition(slideAnimation,
			widgets.NewScaleTransition(scaleAnimation,
				widgets.NewAnimatedBuilder(radiusAnimation, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
					radius := painting.BorderRadiusGeometry(painting.BorderRadiusZero)
					if !animation.IsDismissed(secondaryAnimation) {
						radius = radiusAnimation.Value()
					}
					return widgets.NewClipRSuperellipse(contrastedChild, widgets.ClipRSuperellipseOpts{BorderRadius: radius})
				}, child),
				widgets.ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityMedium), Alignment: painting.AlignmentTopCenter})),
	})
}

// delegatedCoverSheetSecondaryTransition is what a sheet below another sheet does: it moves up a little and
// shrinks.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition._delegatedCoverSheetSecondaryTransition
func delegatedCoverSheetSecondaryTransition(secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	curved := animation.NewCurvedAnimation(secondaryAnimation, animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear)
	slideAnimation := animation.Drive(curved, sheetMidUpTween())
	scaleAnimation := animation.Drive(curved, sheetScaleTween())
	curved.Dispose()
	return widgets.NewSlideTransition(slideAnimation,
		widgets.NewScaleTransition(scaleAnimation,
			widgets.NewClipRSuperellipse(child, widgets.ClipRSuperellipseOpts{
				BorderRadius: painting.NewBorderRadiusVertical(ui.NewRadiusCircular(12), ui.Radius{})}),
			widgets.ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityMedium), Alignment: painting.AlignmentTopCenter}),
		widgets.SlideTransitionOpts{TransformHitTests: foundation.Some(false)})
}

// CreateState creates the state.
//
// Flutter: cupertino/sheet.dart CupertinoSheetTransition.createState
func (w *CupertinoSheetTransition) CreateState() widgets.State {
	return &cupertinoSheetTransitionState{}
}

type cupertinoSheetTransitionState struct {
	widgets.StateOf[*CupertinoSheetTransition]
	widgets.SingleTickerProvider
	// stretchDragController controls the top padding when the sheet is stretched a little upward.
	stretchDragController *animation.AnimationController
	stretchDragAnimation  animation.Animation[float64]
	// secondaryPositionAnimation and secondaryScaleAnimation are for when another sheet covers this one.
	secondaryPositionAnimation animation.Animation[ui.Offset]
	secondaryScaleAnimation    animation.Animation[float64]
	primaryPositionCurve       *animation.CurvedAnimation
	secondaryPositionCurve     *animation.CurvedAnimation
}

func (s *cupertinoSheetTransitionState) InitState() {
	s.stretchDragController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(time.Microsecond)})
	s.setupAnimation()
}

func (s *cupertinoSheetTransitionState) DidUpdateWidget(old *CupertinoSheetTransition) {
	w := s.Widget()
	if old.primaryRouteAnimation != w.primaryRouteAnimation || old.secondaryRouteAnimation != w.secondaryRouteAnimation {
		s.disposeCurve()
		s.setupAnimation()
	}
}

func (s *cupertinoSheetTransitionState) Dispose() {
	s.disposeCurve()
	s.stretchDragController.Dispose()
}

func (s *cupertinoSheetTransitionState) setupAnimation() {
	w := s.Widget()
	s.primaryPositionCurve = animation.NewCurvedAnimation(w.primaryRouteAnimation, animation.Curves.FastEaseInToSlowEaseOut, animation.Curves.FastEaseInToSlowEaseOut.Flipped())
	s.secondaryPositionCurve = animation.NewCurvedAnimation(w.secondaryRouteAnimation, animation.Curves.LinearToEaseOut, animation.Curves.EaseInToLinear)
	// The stretch distance is 0.008 of the height of the screen, whatever the top gap is.
	const stretchDistance = sheetTopGapRatio - sheetStretchedTopGapRatio
	stretchedTopGap := w.topGap - stretchDistance
	s.stretchDragAnimation = animation.Drive[float64](s.stretchDragController, animation.NewTween(w.topGap, stretchedTopGap))
	s.secondaryPositionAnimation = animation.Drive(s.secondaryPositionCurve, sheetMidUpTween())
	s.secondaryScaleAnimation = animation.Drive(s.secondaryPositionCurve, sheetScaleTween())
}

func (s *cupertinoSheetTransitionState) disposeCurve() {
	if s.primaryPositionCurve != nil {
		s.primaryPositionCurve.Dispose()
		s.primaryPositionCurve = nil
	}
	if s.secondaryPositionCurve != nil {
		s.secondaryPositionCurve.Dispose()
		s.secondaryPositionCurve = nil
	}
}

// coverSheetPrimaryTransition slides the sheet in from the bottom.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetTransitionState._coverSheetPrimaryTransition
func (s *cupertinoSheetTransitionState) coverSheetPrimaryTransition(bc widgets.BuildContext, a animation.Animation[float64], linearTransition bool, child widgets.Widget) widgets.Widget {
	offsetTween := sheetBottomUpTween()
	if CupertinoSheetRouteHasParentSheet(bc) {
		offsetTween = sheetBottomUpTweenWhenCoveringOtherSheet()
	}
	var curve, reverseCurve animation.Curve = animation.Curves.FastEaseInToSlowEaseOut, animation.Curves.FastEaseInToSlowEaseOut.Flipped()
	if linearTransition {
		curve, reverseCurve = animation.Curves.Linear, animation.Curves.Linear
	}
	curved := animation.NewCurvedAnimation(a, curve, reverseCurve)
	positionAnimation := animation.Drive(curved, offsetTween)
	curved.Dispose()
	return widgets.NewSlideTransition(positionAnimation, child)
}

// coverSheetSecondaryTransition moves this sheet up and shrinks it when another sheet covers it.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetTransitionState._coverSheetSecondaryTransition
func (s *cupertinoSheetTransitionState) coverSheetSecondaryTransition(child widgets.Widget) widgets.Widget {
	return widgets.NewSlideTransition(s.secondaryPositionAnimation,
		widgets.NewScaleTransition(s.secondaryScaleAnimation, child,
			widgets.ScaleTransitionOpts{FilterQuality: foundation.Some(ui.FilterQualityMedium), Alignment: painting.AlignmentTopCenter}),
		widgets.SlideTransitionOpts{TransformHitTests: foundation.Some(false)})
}

func (s *cupertinoSheetTransitionState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return newStretchDragControllerProvider(s.stretchDragController,
		widgets.NewSizedBoxExpand(widgets.NewAnimatedBuilder(s.stretchDragAnimation, func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
			return widgets.NewPadding(painting.EdgeInsets{Top: widgets.MediaQueryHeightOf(bc) * s.stretchDragAnimation.Value()},
				s.coverSheetSecondaryTransition(s.coverSheetPrimaryTransition(bc, w.primaryRouteAnimation, w.linearTransition, w.child)))
		}, nil)))
}

// stretchDragControllerProvider gives the controller of the upward stretch to the drag detector of the
// sheet.
//
// Flutter: cupertino/sheet.dart _StretchDragControllerProvider
type stretchDragControllerProvider struct {
	widgets.InheritedWidget
	controller *animation.AnimationController
}

func newStretchDragControllerProvider(controller *animation.AnimationController, child widgets.Widget) *stretchDragControllerProvider {
	return &stretchDragControllerProvider{widgets.NewInheritedWidgetBase(child), controller}
}

func (w *stretchDragControllerProvider) UpdateShouldNotify(widgets.Widget) bool { return false }

// CupertinoSheetRouteOpts are the optional parameters of NewCupertinoSheetRoute.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.new
type CupertinoSheetRouteOpts struct {
	// Builder builds the content of the sheet.
	Builder widgets.WidgetBuilder
	// ScrollableBuilder builds the content of the sheet with the controller of the scrollable that the sheet
	// drags; it is used instead of Builder when it is set.
	ScrollableBuilder widgets.ScrollableWidgetBuilder
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// NotEnableDrag is true to keep the user from dismissing the sheet by dragging it down.
	NotEnableDrag bool
	// ShowDragHandle is whether the sheet has a drag handle at the top.
	ShowDragHandle bool
	// TopGap is the gap between the top of the screen and the top of the sheet, as a ratio of the height of
	// the screen, from 0 to 0.9; the default is 0.08.
	TopGap foundation.Opt[float64]
}

// CupertinoSheetRoute is a modal route that shows a sheet in the style of iOS: it slides up from the bottom
// over the route below, which shrinks behind it, and it is dismissed by dragging it down.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute
type CupertinoSheetRoute struct {
	widgets.PageRouteBase
	builder           widgets.WidgetBuilder
	scrollableBuilder widgets.ScrollableWidgetBuilder
	enableDrag        bool
	showDragHandle    bool
	topGap            foundation.Opt[float64]
}

// NewCupertinoSheetRoute creates the route. In debug builds it panics with the tag
// CupertinoSheetTopGapOutOfRange when the top gap is not from 0 to 0.9, and with CupertinoSheetBuilderMissing
// when there is no builder.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.new
func NewCupertinoSheetRoute(opts ...CupertinoSheetRouteOpts) *CupertinoSheetRoute {
	o := firstOf(opts)
	if foundation.DebugMode {
		if o.TopGap.IsSome() && (o.TopGap.Value() < 0 || o.TopGap.Value() > 0.9) {
			panic(errors.New(tagCupertinoSheetTopGapOutOfRange, "topGap must be between 0.0 and 0.9", "topGap", o.TopGap.Value()))
		}
		if o.Builder == nil && o.ScrollableBuilder == nil {
			panic(errors.New(tagCupertinoSheetBuilderMissing, "either ScrollableBuilder or Builder must not be nil"))
		}
	}
	r := &CupertinoSheetRoute{builder: o.Builder, scrollableBuilder: o.ScrollableBuilder, enableDrag: !o.NotEnableDrag,
		showDragHandle: o.ShowDragHandle, topGap: o.TopGap}
	widgets.InitPageRouteBase(&r.PageRouteBase, r, widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: o.RouteSettings}})
	return r
}

// Builder builds the content of the sheet; it is nil when there is a ScrollableBuilder.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.builder
func (r *CupertinoSheetRoute) Builder() widgets.WidgetBuilder { return r.builder }

// ScrollableBuilder builds the content of the sheet with the controller of its scrollable.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.scrollableBuilder
func (r *CupertinoSheetRoute) ScrollableBuilder() widgets.ScrollableWidgetBuilder {
	return r.scrollableBuilder
}

// EnableDrag is whether the user can dismiss the sheet by dragging it down.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.enableDrag
func (r *CupertinoSheetRoute) EnableDrag() bool { return r.enableDrag }

// ShowDragHandle is whether the sheet has a drag handle at the top.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.showDragHandle
func (r *CupertinoSheetRoute) ShowDragHandle() bool { return r.showDragHandle }

// TopGap is the gap between the top of the screen and the top of the sheet, as a ratio of the height of the
// screen.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.topGap
func (r *CupertinoSheetRoute) TopGap() float64 { return r.topGap.OrElse(sheetTopGapRatio) }

func (r *CupertinoSheetRoute) hasCustomTopGap() bool { return r.topGap.IsSome() }

func (r *CupertinoSheetRoute) effectiveBuilder() widgets.ScrollableWidgetBuilder {
	if r.scrollableBuilder != nil {
		return r.scrollableBuilder
	}
	return func(bc widgets.BuildContext, _ widgets.ScrollController) widgets.Widget { return r.builder(bc) }
}

// sheetWithDragHandle builds the content, with the drag handle over it when the route has one.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute._sheetWithDragHandle
func (r *CupertinoSheetRoute) sheetWithDragHandle(bc widgets.BuildContext, controller widgets.ScrollController) widgets.Widget {
	if !r.showDragHandle {
		return r.effectiveBuilder()(bc, controller)
	}
	// The values come from the Figma files of Apple and a simulator that runs iOS 18.2.
	const (
		dragHandleTopPadding = 5.0
		dragHandleHeight     = 5.0
		dragHandleWidth      = 36.0
		dragHandlePadding    = 15.0
	)
	data := widgets.MediaQueryOf(bc)
	data.Padding = painting.EdgeInsets{Top: dragHandlePadding}
	return widgets.NewStack([]widgets.Widget{
		widgets.NewMediaQuery(data, r.effectiveBuilder()(bc, controller)),
		widgets.NewAlign(widgets.NewPadding(painting.EdgeInsets{Top: dragHandleTopPadding},
			widgets.NewDecoratedBox(painting.NewShapeDecoration(
				painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(dragHandleWidth / 2))}),
				painting.ShapeDecorationOpts{Color: foundation.Some(CupertinoColors.TertiaryLabel.EffectiveColor())}),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(dragHandleHeight), Width: foundation.Some(dragHandleWidth)}))),
			widgets.AlignOpts{Alignment: painting.AlignmentTopCenter}),
	}, widgets.StackOpts{Fit: rendering.StackFitExpand})
}

// BuildContent builds the sheet: the content with rounded corners at the top, at the elevated level of the
// interface, that the user can drag down.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.buildContent
func (r *CupertinoSheetRoute) BuildContent(bc widgets.BuildContext) widgets.Widget {
	return widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true},
		widgets.NewClipRSuperellipse(
			NewCupertinoUserInterfaceLevel(CupertinoUserInterfaceLevelElevated,
				newCupertinoSheetScope(
					newCupertinoDraggableScrollableSheet(
						func() bool { return r.enableDrag },
						func() *sheetDragGestureController { return startSheetPopGesture(r, r.TopGap()) },
						r.sheetWithDragHandle))),
			widgets.ClipRSuperellipseOpts{BorderRadius: painting.NewBorderRadiusVertical(ui.NewRadiusCircular(12), ui.Radius{})}))
}

// CupertinoSheetRouteHasParentSheet is whether the context is in a sheet.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.hasParentSheet
func CupertinoSheetRouteHasParentSheet(bc widgets.BuildContext) bool {
	_, ok := widgets.GetInherited[*cupertinoSheetScope](bc)
	return ok
}

// CupertinoSheetRoutePopSheet pops the root navigator when the context is in a sheet.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.popSheet
func CupertinoSheetRoutePopSheet(bc widgets.BuildContext) {
	if CupertinoSheetRouteHasParentSheet(bc) {
		widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: true}).Pop()
	}
}

// BarrierColor is transparent.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.barrierColor
func (r *CupertinoSheetRoute) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.Some(ui.NewColor(0x00000000))
}

// BarrierDismissible is false.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.barrierDismissible
func (r *CupertinoSheetRoute) BarrierDismissible() bool { return false }

// BarrierLabel is empty.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.barrierLabel
func (r *CupertinoSheetRoute) BarrierLabel() string { return "" }

// MaintainState is true.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.maintainState
func (r *CupertinoSheetRoute) MaintainState() bool { return true }

// Opaque is false: the route below shows behind the sheet.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.opaque
func (r *CupertinoSheetRoute) Opaque() bool { return false }

// TransitionDuration is 500 ms.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.transitionDuration
func (r *CupertinoSheetRoute) TransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// ReverseTransitionDuration is 500 ms.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.reverseTransitionDuration
func (r *CupertinoSheetRoute) ReverseTransitionDuration() time.Duration {
	return CupertinoRouteTransitionDuration
}

// DelegatedTransition is the transition of the route below, none when the sheet has a top gap of its own.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.delegatedTransition
func (r *CupertinoSheetRoute) DelegatedTransition() widgets.DelegatedTransitionBuilder {
	if r.hasCustomTopGap() {
		return nil
	}
	return CupertinoSheetTransitionDelegatedTransition
}

// CanTransitionFrom is false when the sheet has a top gap of its own.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.canTransitionFrom
func (r *CupertinoSheetRoute) CanTransitionFrom(widgets.TransitionRoute) bool {
	return !r.hasCustomTopGap()
}

// CanTransitionTo is whether the next route is a sheet, and this one has no top gap of its own.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.canTransitionTo
func (r *CupertinoSheetRoute) CanTransitionTo(nextRoute widgets.TransitionRoute) bool {
	if r.hasCustomTopGap() {
		return false
	}
	_, ok := nextRoute.(*CupertinoSheetRoute)
	return ok
}

// BuildPage builds the content.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.buildPage
func (r *CupertinoSheetRoute) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	return r.BuildContent(bc)
}

// BuildTransitions builds the transition of the sheet, with the gesture that drags it down.
//
// Flutter: cupertino/sheet.dart CupertinoSheetRoute.buildTransitions
func (r *CupertinoSheetRoute) BuildTransitions(bc widgets.BuildContext, a, secondaryAnimation animation.Animation[float64], child widgets.Widget) widgets.Widget {
	linearTransition := r.PopGestureInProgress()
	return NewCupertinoSheetTransition(a, secondaryAnimation, linearTransition,
		newSheetDragGestureDetector(func() bool { return r.enableDrag },
			func() *sheetDragGestureController { return startSheetPopGesture(r, r.TopGap()) }, child),
		CupertinoSheetTransitionOpts{TopGap: foundation.Some(r.TopGap())})
}

// cupertinoSheetScope tells the content of a sheet that it is in one.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScope
type cupertinoSheetScope struct{ widgets.InheritedWidget }

func newCupertinoSheetScope(child widgets.Widget) *cupertinoSheetScope {
	return &cupertinoSheetScope{widgets.NewInheritedWidgetBase(child)}
}

func (w *cupertinoSheetScope) UpdateShouldNotify(widgets.Widget) bool { return false }

func startSheetPopGesture(route widgets.ModalRoute, topGap float64) *sheetDragGestureController {
	return newSheetDragGestureController(route.Navigator(), route.Controller(), route.IsCurrent, route.IsActive, topGap)
}

// sheetDragGestureDetector is the layer over the sheet that starts the drag that dismisses it.
//
// Flutter: cupertino/sheet.dart _CupertinoDragGestureDetector
type sheetDragGestureDetector struct {
	widgets.StatefulWidget
	enabledCallback   func() bool
	onStartPopGesture func() *sheetDragGestureController
	child             widgets.Widget
}

func newSheetDragGestureDetector(enabled func() bool, onStart func() *sheetDragGestureController, child widgets.Widget) *sheetDragGestureDetector {
	return &sheetDragGestureDetector{widgets.NewStatefulWidgetBase(), enabled, onStart, child}
}

func (w *sheetDragGestureDetector) CreateState() widgets.State {
	return &sheetDragGestureDetectorState{}
}

type sheetDragGestureDetectorState struct {
	widgets.StateOf[*sheetDragGestureDetector]
	controller        *sheetDragGestureController
	recognizer        *gestures.VerticalDragGestureRecognizer
	stretchController *stretchDragControllerProvider
}

func (s *sheetDragGestureDetectorState) sheetHeight() float64 {
	size, _ := s.Context().Size()
	return size.Height
}

func (s *sheetDragGestureDetectorState) InitState() {
	s.stretchController, _ = widgets.GetInherited[*stretchDragControllerProvider](s.Context())
	binding := widgets.WidgetsBindingOf(s.Context())
	s.recognizer = gestures.NewVerticalDragGestureRecognizer(binding.Gestures(), gestures.DragGestureRecognizerOpts{DebugOwner: s})
	clock := binding.Gestures().Clock()
	s.recognizer.VelocityTrackerBuilder = func(event gestures.PointerEvent) *gestures.VelocityTracker {
		return gestures.NewIOSScrollViewFlingVelocityTracker(clock, event.Kind())
	}
	s.recognizer.OnStart = s.handleDragStart
	s.recognizer.OnUpdate = s.handleDragUpdate
	s.recognizer.OnEnd = s.handleDragEnd
	s.recognizer.OnCancel = s.handleDragCancel
}

func (s *sheetDragGestureDetectorState) DidChangeDependencies() {
	s.stretchController, _ = widgets.GetInherited[*stretchDragControllerProvider](s.Context())
}

func (s *sheetDragGestureDetectorState) Dispose() {
	s.recognizer.Dispose()
	// If this is disposed during a drag, tell the navigator after the frame that the gesture is over.
	if c := s.controller; c != nil {
		binding := widgets.WidgetsBindingOf(s.Context())
		binding.RunPostFrame(func(time.Duration) {
			if c.navigator.Mounted() {
				c.navigator.DidStopUserGesture()
			}
		})
		s.controller = nil
	}
}

func (s *sheetDragGestureDetectorState) handleDragStart(gestures.DragStartDetails) {
	s.controller = s.Widget().onStartPopGesture()
}

func (s *sheetDragGestureDetectorState) handleDragUpdate(details gestures.DragUpdateDetails) {
	if s.stretchController == nil || s.controller == nil {
		return
	}
	delta := 0.0
	if height := s.sheetHeight(); height > 0 {
		delta = details.PrimaryDelta.Value() / height
	}
	s.controller.dragUpdate(delta, s.stretchController.controller)
}

func (s *sheetDragGestureDetectorState) handleDragEnd(details gestures.DragEndDetails) {
	if s.stretchController == nil || s.controller == nil {
		s.controller = nil
		return
	}
	velocity := 0.0
	if height := s.sheetHeight(); height > 0 {
		velocity = details.Velocity.PixelsPerSecond.Dy / height
	}
	s.controller.dragEnd(velocity, s.stretchController.controller)
	s.controller = nil
}

func (s *sheetDragGestureDetectorState) handleDragCancel() {
	// This can be called even when the start was not, paired with the down event.
	if s.stretchController == nil {
		s.controller = nil
		return
	}
	if s.controller != nil {
		s.controller.dragEnd(0, s.stretchController.controller)
	}
	s.controller = nil
}

func (s *sheetDragGestureDetectorState) handlePointerDown(event *gestures.PointerDownEvent) {
	if s.Widget().enabledCallback() {
		s.recognizer.AddPointer(event)
	}
}

func (s *sheetDragGestureDetectorState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewListener(s.Widget().child, widgets.ListenerOpts{OnPointerDown: s.handlePointerDown, Behavior: rendering.HitTestBehaviorTranslucent})
}

// sheetDragGestureController moves the sheet with the finger of the drag, and lets it settle or go when the
// finger is lifted.
//
// Flutter: cupertino/sheet.dart _CupertinoDragGestureController
type sheetDragGestureController struct {
	popDragController *animation.AnimationController
	navigator         *widgets.NavigatorState
	getIsActive       func() bool
	getIsCurrent      func() bool
	topGap            float64
}

func newSheetDragGestureController(navigator *widgets.NavigatorState, popDragController *animation.AnimationController, isCurrent, isActive func() bool, topGap float64) *sheetDragGestureController {
	c := &sheetDragGestureController{popDragController: popDragController, navigator: navigator,
		getIsActive: isActive, getIsCurrent: isCurrent, topGap: topGap}
	navigator.DidStartUserGesture()
	return c
}

// dragUpdate moves the sheet by the fraction of its height that the finger moved. Pulling up the sheet that is
// open to its limit stretches it instead, through upController when there is one.
func (c *sheetDragGestureController) dragUpdate(delta float64, upController *animation.AnimationController) {
	if upController != nil && c.popDragController.Value() == 1.0 && (upController.Value() > 0 || delta < 0) {
		// The stretchable range is the same whatever the top gap is.
		const stretchDistance = sheetTopGapRatio - sheetStretchedTopGapRatio
		upController.SetValue(upController.Value() - delta/stretchDistance)
	} else {
		c.popDragController.SetValue(c.popDragController.Value() - delta)
	}
}

func (c *sheetDragGestureController) isDragged() bool { return c.popDragController.Value() != 1.0 }

// dragEnd lets go of the sheet: a stretched sheet goes back to its height; otherwise the sheet pops when the
// finger was lifted past the middle or flinging down, and comes back otherwise.
func (c *sheetDragGestureController) dragEnd(velocity float64, upController *animation.AnimationController) {
	// If the sheet is stretched upward, the stretch goes back.
	if upController != nil && upController.Value() > 0 {
		upController.AnimateBack(0.0, animation.AnimateOpts{Duration: foundation.Some(180 * time.Millisecond), Curve: animation.Curves.EaseOut})
		c.navigator.DidStopUserGesture()
		return
	}
	// This curve was found by eyeballing the animations of iOS 18.0 on a simulator.
	var animationCurve animation.Curve = animation.Curves.EaseOut
	isCurrent := c.getIsCurrent()
	var animateForward bool
	switch {
	case !isCurrent:
		// The route was navigated away from: whether it is still in the stack decides, whatever the velocity
		// or the position were.
		animateForward = c.getIsActive()
	case velocity >= sheetMinFlingVelocity || velocity <= -sheetMinFlingVelocity:
		animateForward = velocity <= 0
	default:
		// A drag that is dropped slowly pops the sheet a little past the halfway point.
		animateForward = c.popDragController.Value() > 0.52
	}
	opts := animation.AnimateOpts{Duration: foundation.Some(sheetDroppedDragAnimationDuration), Curve: animationCurve}
	if animateForward {
		c.popDragController.AnimateTo(1.0, opts)
	} else {
		if isCurrent {
			// This route is destined to pop at this point: reuse the pop of the navigator.
			c.navigator.Pop()
		}
		if c.popDragController.IsAnimating() {
			c.popDragController.AnimateBack(0.0, opts)
		}
	}
	if c.popDragController.IsAnimating() {
		// Keep the user gesture in progress so that the curve of the transition does not change mid-flight.
		var handle animation.StatusListenerHandle
		handle = c.popDragController.AddStatusListener(func(animation.AnimationStatus) {
			c.navigator.DidStopUserGesture()
			c.popDragController.RemoveStatusListener(handle)
		})
	} else {
		c.navigator.DidStopUserGesture()
	}
}

// sheetScrollController is the controller of the scrollable in a sheet: its position hands the drag to the
// sheet when the content is at its start.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollController
type sheetScrollController struct {
	widgets.ScrollControllerBase
	onDragStart        func()
	onDragUpdate       func(delta float64)
	onDragEnd          func(velocity float64)
	sheetIsDraggedDown func() bool
}

func newSheetScrollController(onDragStart func(), onDragUpdate, onDragEnd func(float64), sheetIsDraggedDown func() bool) *sheetScrollController {
	c := &sheetScrollController{onDragStart: onDragStart, onDragUpdate: onDragUpdate, onDragEnd: onDragEnd, sheetIsDraggedDown: sheetIsDraggedDown}
	widgets.InitScrollControllerBase(&c.ScrollControllerBase, c, widgets.ScrollControllerOpts{})
	return c
}

// CreateScrollPosition makes the position that hands the drag to the sheet.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollController.createScrollPosition
func (c *sheetScrollController) CreateScrollPosition(physics widgets.ScrollPhysics, context widgets.ScrollContext, oldPosition widgets.ScrollPosition) widgets.ScrollPosition {
	return newSheetScrollPosition(physics.ApplyTo(widgets.NewAlwaysScrollableScrollPhysics(nil)), context, oldPosition,
		c.onDragStart, c.onDragUpdate, c.onDragEnd, c.sheetIsDraggedDown)
}

// sheetScrollPosition is a modified version of the position of DraggableScrollableSheet: a drag down on a
// list at its start moves the sheet, and a fling down at the start dismisses it.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition
type sheetScrollPosition struct {
	widgets.ScrollPositionWithSingleContext
	onDragStart          func()
	onDragUpdate         func(delta float64)
	onDragEnd            func(velocity float64)
	sheetIsDraggedDown   func() bool
	dragCancelCallback   func()
	ballisticControllers map[*animation.AnimationController]struct{}
}

func newSheetScrollPosition(physics widgets.ScrollPhysics, context widgets.ScrollContext, oldPosition widgets.ScrollPosition,
	onDragStart func(), onDragUpdate, onDragEnd func(float64), sheetIsDraggedDown func() bool) *sheetScrollPosition {
	p := &sheetScrollPosition{onDragStart: onDragStart, onDragUpdate: onDragUpdate, onDragEnd: onDragEnd,
		sheetIsDraggedDown: sheetIsDraggedDown, ballisticControllers: map[*animation.AnimationController]struct{}{}}
	widgets.InitScrollPositionWithSingleContext(&p.ScrollPositionWithSingleContext, p, physics, context,
		widgets.ScrollPositionWithSingleContextOpts{OldPosition: oldPosition})
	return p
}

func (p *sheetScrollPosition) listShouldScroll() bool { return p.Pixels() > 0.0 }

// Absorb takes the cancel callback of the other position.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.absorb
func (p *sheetScrollPosition) Absorb(other widgets.ScrollPosition) {
	p.ScrollPositionWithSingleContext.Absorb(other)
	o, ok := other.(*sheetScrollPosition)
	if !ok {
		return
	}
	if o.dragCancelCallback != nil {
		p.dragCancelCallback = o.dragCancelCallback
		o.dragCancelCallback = nil
	}
}

// BeginActivity stops the ballistic simulations that run.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.beginActivity
func (p *sheetScrollPosition) BeginActivity(newActivity widgets.ScrollActivity) {
	for c := range p.ballisticControllers {
		c.Stop()
	}
	p.ScrollPositionWithSingleContext.BeginActivity(newActivity)
}

// Dispose disposes the ballistic controllers.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.dispose
func (p *sheetScrollPosition) Dispose() {
	for c := range p.ballisticControllers {
		c.Dispose()
	}
	clear(p.ballisticControllers)
	p.ScrollPositionWithSingleContext.Dispose()
}

// ApplyUserOffset moves the sheet when the list is at its start and the drag is down, or the sheet is
// dragged down already; otherwise it scrolls the list.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.applyUserOffset
func (p *sheetScrollPosition) ApplyUserOffset(delta float64) {
	p.onDragStart()
	if !p.listShouldScroll() && (delta > 0 || p.sheetIsDraggedDown()) {
		p.onDragUpdate(delta)
	} else {
		p.ScrollPositionWithSingleContext.ApplyUserOffset(delta)
	}
}

// GoBallistic ends the drag of the sheet, and lets the list fling or the sheet go with the velocity.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.goBallistic
func (p *sheetScrollPosition) GoBallistic(velocity float64) {
	// End the drag gesture.
	if velocity == 0.0 || (velocity < 0.0 && p.listShouldScroll()) || (velocity > 0.0 && p.Pixels() != p.MaxScrollExtent()) {
		p.onDragEnd(0.0)
		p.ScrollPositionWithSingleContext.GoBallistic(velocity)
		return
	}
	if p.dragCancelCallback != nil {
		p.dragCancelCallback()
		p.dragCancelCallback = nil
	}
	if velocity < 0.0 && !p.listShouldScroll() {
		p.onDragEnd(velocity)
		p.ScrollPositionWithSingleContext.GoBallistic(0)
		return
	}
	p.onDragEnd(0.0)
	p.ScrollPositionWithSingleContext.GoBallistic(velocity)
}

// Drag keeps the cancel callback, to call it when the position goes ballistic by itself.
//
// Flutter: cupertino/sheet.dart _CupertinoSheetScrollPosition.drag
func (p *sheetScrollPosition) Drag(details gestures.DragStartDetails, dragCancelCallback func()) gestures.Drag {
	p.dragCancelCallback = dragCancelCallback
	return p.ScrollPositionWithSingleContext.Drag(details, dragCancelCallback)
}

// cupertinoDraggableScrollableSheet gives its builder the controller that hands the drag of its
// scrollable to the route.
//
// Flutter: cupertino/sheet.dart _CupertinoDraggableScrollableSheet
type cupertinoDraggableScrollableSheet struct {
	widgets.StatefulWidget
	builder           widgets.ScrollableWidgetBuilder
	enabledCallback   func() bool
	onStartPopGesture func() *sheetDragGestureController
}

func newCupertinoDraggableScrollableSheet(enabled func() bool, onStart func() *sheetDragGestureController, builder widgets.ScrollableWidgetBuilder) *cupertinoDraggableScrollableSheet {
	return &cupertinoDraggableScrollableSheet{widgets.NewStatefulWidgetBase(), builder, enabled, onStart}
}

func (w *cupertinoDraggableScrollableSheet) CreateState() widgets.State {
	return &cupertinoDraggableScrollableSheetState{}
}

type cupertinoDraggableScrollableSheetState struct {
	widgets.StateOf[*cupertinoDraggableScrollableSheet]
	scrollController *sheetScrollController
	controller       *sheetDragGestureController
}

func (s *cupertinoDraggableScrollableSheetState) InitState() {
	s.scrollController = newSheetScrollController(s.dragStart, s.dragUpdate, s.handleDragEnd,
		func() bool { return s.controller != nil && s.controller.isDragged() })
}

func (s *cupertinoDraggableScrollableSheetState) Dispose() {
	// If this is disposed during a drag, tell the navigator after the frame that the gesture is over.
	if c := s.controller; c != nil {
		binding := widgets.WidgetsBindingOf(s.Context())
		binding.RunPostFrame(func(time.Duration) {
			if c.navigator.Mounted() {
				c.navigator.DidStopUserGesture()
			}
		})
		s.controller = nil
	}
	s.scrollController.Dispose()
}

func (s *cupertinoDraggableScrollableSheetState) dragStart() {
	if s.controller == nil {
		s.controller = s.Widget().onStartPopGesture()
	}
}

func (s *cupertinoDraggableScrollableSheetState) dragUpdate(delta float64) {
	if s.controller != nil {
		size, _ := s.Context().Size()
		s.controller.dragUpdate(delta/(size.Height-size.Height*sheetTopGapRatio), nil)
	}
}

func (s *cupertinoDraggableScrollableSheetState) handleDragEnd(velocity float64) {
	if s.controller != nil {
		size, _ := s.Context().Size()
		s.controller.dragEnd(-velocity/size.Height, nil)
		s.controller = nil
	}
}

func (s *cupertinoDraggableScrollableSheetState) Build(bc widgets.BuildContext) widgets.Widget {
	return s.Widget().builder(bc, s.scrollController)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSheetTopGapOutOfRange = "CupertinoSheetTopGapOutOfRange"
	tagCupertinoSheetBuilderMissing   = "CupertinoSheetBuilderMissing"
)
