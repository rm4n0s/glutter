// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"reflect"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Flutter: material/bottom_sheet.dart _kBottomSheetEnterDuration
const bottomSheetEnterDuration = 250 * time.Millisecond

// Flutter: material/bottom_sheet.dart _kBottomSheetExitDuration
const bottomSheetExitDuration = 200 * time.Millisecond

// Flutter: material/bottom_sheet.dart _kMinFlingVelocity
const bottomSheetMinFlingVelocity = 700.0

// Flutter: material/bottom_sheet.dart _kCloseProgressThreshold
const bottomSheetCloseProgressThreshold = 0.5

// bottomSheetDefaultScrollControlDisabledMaxHeightRatio is the ratio of the
// height of the screen that a modal bottom sheet that is not scroll controlled
// takes at most.
//
// Flutter: material/bottom_sheet.dart _kDefaultScrollControlDisabledMaxHeightRatio
const bottomSheetDefaultScrollControlDisabledMaxHeightRatio = 9.0 / 16.0

// BottomSheetDragStartHandler is called when the user starts to drag a bottom
// sheet.
//
// Flutter: material/bottom_sheet.dart BottomSheetDragStartHandler
type BottomSheetDragStartHandler func(details gestures.DragStartDetails)

// BottomSheetDragEndHandler is called when the user stops dragging a bottom
// sheet; isClosing is whether the drag closes it.
//
// Flutter: material/bottom_sheet.dart BottomSheetDragEndHandler
type BottomSheetDragEndHandler func(details gestures.DragEndDetails, isClosing bool)

// BottomSheetOpts are the optional parameters of NewBottomSheet.
//
// Flutter: material/bottom_sheet.dart BottomSheet.new
type BottomSheetOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AnimationController controls the sheet when it is dragged; it is required
	// when EnableDrag or ShowDragHandle is true (BottomSheetCreateAnimationController
	// makes one).
	AnimationController *animation.AnimationController
	// NoEnableDrag says that the sheet cannot be dragged down to close it
	// (Flutter's enableDrag: false).
	NoEnableDrag bool
	// ShowDragHandle says whether the sheet shows a drag handle; the default is
	// the theme's, when EnableDrag is true.
	ShowDragHandle foundation.Opt[bool]
	// DragHandleColor is the color of the drag handle, in each state.
	DragHandleColor widgets.WidgetStateProperty[ui.Color]
	// DragHandleSize is the size of the drag handle; the default is 32 by 4.
	DragHandleSize foundation.Opt[ui.Size]
	// OnDragStart is called when the user starts to drag the sheet.
	OnDragStart BottomSheetDragStartHandler
	// OnDragEnd is called when the user stops dragging the sheet.
	OnDragEnd BottomSheetDragEndHandler
	// BackgroundColor is the color of the sheet; the default comes from the theme.
	BackgroundColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the sheet.
	ShadowColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the sheet.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the sheet.
	Shape painting.ShapeBorder
	// ClipBehavior is the clip of the sheet's Material; the default is ClipNone.
	ClipBehavior ui.Clip
	// Constraints are the size constraints of the sheet.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// BottomSheet is a Material Design bottom sheet. It is the content of a modal
// bottom sheet (ShowModalBottomSheet) and of a persistent one (ShowBottomSheet,
// Scaffold.BottomSheet), and the widget gives them the drag to close.
//
// Flutter: material/bottom_sheet.dart BottomSheet
type BottomSheet struct {
	widgets.StatefulWidget
	onClosing func()
	builder   widgets.WidgetBuilder
	o         BottomSheetOpts
}

// NewBottomSheet creates a bottom sheet. onClosing is called when the sheet is
// about to close.
//
// Flutter: material/bottom_sheet.dart BottomSheet.new
func NewBottomSheet(onClosing func(), builder widgets.WidgetBuilder, opts ...BottomSheetOpts) *BottomSheet {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New(tagBottomSheetElevationNegative, "the elevation of a BottomSheet must not be negative", "elevation", o.Elevation.Value()))
	}
	return &BottomSheet{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onClosing: onClosing, builder: builder, o: o}
}

// EnableDrag is whether the sheet can be dragged down to close it.
//
// Flutter: material/bottom_sheet.dart BottomSheet.enableDrag
func (w *BottomSheet) EnableDrag() bool { return !w.o.NoEnableDrag }

// AnimationController is the controller of the sheet.
//
// Flutter: material/bottom_sheet.dart BottomSheet.animationController
func (w *BottomSheet) AnimationController() *animation.AnimationController {
	return w.o.AnimationController
}

// Builder builds the content of the sheet.
//
// Flutter: material/bottom_sheet.dart BottomSheet.builder
func (w *BottomSheet) Builder() widgets.WidgetBuilder { return w.builder }

// CreateState creates the state.
//
// Flutter: material/bottom_sheet.dart BottomSheet.createState
func (w *BottomSheet) CreateState() widgets.State { return &bottomSheetState{} }

// BottomSheetCreateAnimationController creates an animation controller with the
// durations of a bottom sheet (250 ms to open and 200 ms to close), for a sheet
// with a drag.
//
// Flutter: material/bottom_sheet.dart BottomSheet.createAnimationController
func BottomSheetCreateAnimationController(vsync scheduler.TickerProvider, sheetAnimationStyle *animation.AnimationStyle) *animation.AnimationController {
	duration, reverseDuration := foundation.Some(bottomSheetEnterDuration), foundation.Some(bottomSheetExitDuration)
	if sheetAnimationStyle != nil {
		if d, ok := sheetAnimationStyle.Duration.Get(); ok {
			duration = foundation.Some(d)
		}
		if d, ok := sheetAnimationStyle.ReverseDuration.Get(); ok {
			reverseDuration = foundation.Some(d)
		}
	}
	return animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: duration, ReverseDuration: reverseDuration, DebugLabel: "BottomSheet"})
}

type bottomSheetState struct {
	widgets.StateOf[*BottomSheet]
	childKey *widgets.GlobalKey[widgets.State]
	dragged  bool
	hovered  bool
}

func (s *bottomSheetState) InitState() {
	s.childKey = widgets.NewGlobalKey[widgets.State](widgets.GlobalKeyOpts{DebugLabel: "BottomSheet child"})
}

func (s *bottomSheetState) childHeight() float64 {
	box := s.childKey.CurrentContext().FindRenderObject().(rendering.Box)
	return box.Size().Height
}

func (s *bottomSheetState) dismissUnderway() bool {
	return s.Widget().o.AnimationController.Status() == animation.AnimationStatusReverse
}

func (s *bottomSheetState) handleDragStart(details gestures.DragStartDetails) {
	s.SetState(func() { s.dragged = true })
	if cb := s.Widget().o.OnDragStart; cb != nil {
		cb(details)
	}
}

func (s *bottomSheetState) checkController() {
	w := s.Widget()
	if foundation.DebugMode && (w.EnableDrag() || w.o.ShowDragHandle.OrElse(false)) && w.o.AnimationController == nil {
		panic(errors.New(tagBottomSheetNoAnimationController, "'BottomSheet.AnimationController' cannot be nil when 'BottomSheet.EnableDrag' or 'BottomSheet.ShowDragHandle' is true. "+
			"Use 'BottomSheetCreateAnimationController' to create one, or provide another AnimationController."))
	}
}

func (s *bottomSheetState) handleDragUpdate(details gestures.DragUpdateDetails) {
	s.checkController()
	if s.dismissUnderway() {
		return
	}
	c := s.Widget().o.AnimationController
	c.SetValue(c.Value() - details.PrimaryDelta.Value()/s.childHeight())
}

func (s *bottomSheetState) handleDragEnd(details gestures.DragEndDetails) {
	s.checkController()
	if s.dismissUnderway() {
		return
	}
	s.SetState(func() { s.dragged = false })
	w := s.Widget()
	c := w.o.AnimationController
	isClosing := false
	if details.Velocity.PixelsPerSecond.Dy > bottomSheetMinFlingVelocity {
		flingVelocity := -details.Velocity.PixelsPerSecond.Dy / s.childHeight()
		if c.Value() > 0.0 {
			c.Fling(animation.FlingOpts{Velocity: foundation.Some(flingVelocity)})
		}
		if flingVelocity < 0.0 {
			isClosing = true
		}
	} else if c.Value() < bottomSheetCloseProgressThreshold {
		if c.Value() > 0.0 {
			c.Fling(animation.FlingOpts{Velocity: foundation.Some(-1.0)})
		}
		isClosing = true
	} else {
		c.Forward()
	}

	if cb := w.o.OnDragEnd; cb != nil {
		cb(details, isClosing)
	}
	if isClosing {
		w.onClosing()
	}
}

func (s *bottomSheetState) handleDragHandleHover(hovering bool) {
	if hovering != s.hovered {
		s.SetState(func() { s.hovered = hovering })
	}
}

func (s *bottomSheetState) dragHandleStates() widgets.WidgetStates {
	states := widgets.NewWidgetStates()
	if s.dragged {
		states = states.With(widgets.WidgetStateDragged)
	}
	if s.hovered {
		states = states.With(widgets.WidgetStateHovered)
	}
	return states
}

func (s *bottomSheetState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := ThemeOf(bc)
	sheetTheme := theme.BottomSheetTheme
	defaults := bottomSheetDefaults(theme)
	constraints := firstOpt(w.o.Constraints, sheetTheme.Constraints, defaults.Constraints)
	color := firstOpt(w.o.BackgroundColor, sheetTheme.BackgroundColor, defaults.BackgroundColor)
	surfaceTintColor := firstOpt(sheetTheme.SurfaceTintColor, defaults.SurfaceTintColor)
	shadowColor := firstOpt(w.o.ShadowColor, sheetTheme.ShadowColor, defaults.ShadowColor)
	elevation := firstOpt(w.o.Elevation, sheetTheme.Elevation, defaults.Elevation).OrElse(0)
	shape := w.o.Shape
	if shape == nil {
		shape = sheetTheme.Shape
	}
	if shape == nil {
		shape = defaults.Shape
	}
	clipBehavior := w.o.ClipBehavior
	if clipBehavior == ui.ClipUnspecified {
		clipBehavior = sheetTheme.ClipBehavior
	}
	if clipBehavior == ui.ClipUnspecified {
		clipBehavior = ui.ClipNone
	}
	showDragHandle := w.o.ShowDragHandle.OrElse(w.EnableDrag() && sheetTheme.ShowDragHandle.OrElse(false))

	var dragHandle widgets.Widget
	if showDragHandle {
		dragHandle = &bottomSheetDragHandle{
			StatelessWidget: widgets.NewStatelessWidgetBase(), onSemanticsTap: w.onClosing, handleHover: s.handleDragHandleHover,
			states: s.dragHandleStates(), dragHandleColor: w.o.DragHandleColor, dragHandleSize: w.o.DragHandleSize,
		}
		if !w.EnableDrag() {
			dragHandle = newBottomSheetGestureDetector(s.handleDragStart, s.handleDragUpdate, s.handleDragEnd, dragHandle)
		}
	}

	var content widgets.Widget
	if !showDragHandle {
		content = w.builder(bc)
	} else {
		content = widgets.NewStack([]widgets.Widget{
			dragHandle,
			widgets.NewPadding(painting.EdgeInsets{Top: MinInteractiveDimension}, w.builder(bc)),
		}, widgets.StackOpts{Alignment: painting.AlignmentTopCenter})
	}
	var bottomSheet widgets.Widget = NewMaterial(content, MaterialOpts{
		Key: s.childKey, Color: color, Elevation: elevation, SurfaceTintColor: surfaceTintColor, ShadowColor: shadowColor,
		Shape: shape, ClipBehavior: clipBehavior})
	// TODO: the NotificationListener of DraggableScrollableNotification, which
	// closes the sheet at the minimum extent of a DraggableScrollableSheet: the
	// widget is not ported.

	if c, ok := constraints.Get(); ok {
		bottomSheet = widgets.NewAlign(widgets.NewConstrainedBox(c, bottomSheet),
			widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter, HeightFactor: foundation.Some(1.0)})
	}
	if !w.EnableDrag() {
		return bottomSheet
	}
	return newBottomSheetGestureDetector(s.handleDragStart, s.handleDragUpdate, s.handleDragEnd, bottomSheet)
}

// firstOpt is the first value that is set, of the nullable values in the order of
// Dart's `a ?? b ?? c`.
func firstOpt[T any](values ...foundation.Opt[T]) foundation.Opt[T] {
	for _, v := range values {
		if v.IsSome() {
			return v
		}
	}
	return foundation.None[T]()
}

// bottomSheetDefaults are the defaults of the bottom sheet of the theme: the
// Material 3 ones, and none for Material 2.
func bottomSheetDefaults(theme *ThemeData) BottomSheetThemeData {
	if !theme.UseMaterial3 {
		return BottomSheetThemeData{}
	}
	return bottomSheetDefaultsM3(theme)
}

// bottomSheetDefaultsM3 are the Material 3 defaults of the bottom sheet.
//
// Flutter: material/bottom_sheet.dart _BottomSheetDefaultsM3
func bottomSheetDefaultsM3(theme *ThemeData) BottomSheetThemeData {
	colors := theme.ColorScheme
	return BottomSheetThemeData{
		Elevation: foundation.Some(1.0), ModalElevation: foundation.Some(1.0),
		Shape:           painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusVertical(ui.NewRadiusCircular(28), ui.Radius{})}),
		Constraints:     foundation.Some(rendering.BoxConstraints{MaxWidth: 640, MaxHeight: math.Inf(1), MinWidth: 0, MinHeight: 0}),
		BackgroundColor: foundation.Some(colors.SurfaceContainerLow()), SurfaceTintColor: foundation.Some(Colors.Transparent),
		ShadowColor:     foundation.Some(Colors.Transparent),
		DragHandleColor: widgets.NewWidgetStatePropertyAll(colors.OnSurfaceVariant()),
		DragHandleSize:  foundation.Some(ui.Size{Width: 32, Height: 4}),
	}
}

// bottomSheetDragHandle is the drag handle of a bottom sheet.
//
// Flutter: material/bottom_sheet.dart _DragHandle
type bottomSheetDragHandle struct {
	widgets.StatelessWidget
	onSemanticsTap  func()
	handleHover     func(bool)
	states          widgets.WidgetStates
	dragHandleColor widgets.WidgetStateProperty[ui.Color]
	dragHandleSize  foundation.Opt[ui.Size]
}

func (w *bottomSheetDragHandle) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	sheetTheme := theme.BottomSheetTheme
	m3 := bottomSheetDefaultsM3(theme)
	handleSize := firstOpt(w.dragHandleSize, sheetTheme.DragHandleSize, m3.DragHandleSize).Value()
	var color ui.Color
	switch {
	case w.dragHandleColor != nil:
		color = w.dragHandleColor.Resolve(w.states)
	case sheetTheme.DragHandleColor != nil:
		color = sheetTheme.DragHandleColor.Resolve(w.states)
	default:
		color = m3.DragHandleColor.Resolve(w.states)
	}
	return widgets.NewMouseRegion(
		widgets.NewSemantics(
			widgets.NewSizedBox(
				widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
					Height: foundation.Some(handleSize.Height), Width: foundation.Some(handleSize.Width),
					Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
						BorderRadius: painting.NewBorderRadiusCircular(handleSize.Height / 2), Color: foundation.Some(color)}),
				})),
				widgets.SizedBoxOpts{Width: foundation.Some(math.Max(handleSize.Width, MinInteractiveDimension)),
					Height: foundation.Some(math.Max(handleSize.Height, MinInteractiveDimension))}),
			widgets.SemanticsOpts{Label: foundation.Some(MaterialLocalizationsOf(bc).ModalBarrierDismissLabel()), Container: true,
				Button: foundation.Some(true), OnTap: w.onSemanticsTap}),
		widgets.MouseRegionOpts{
			OnEnter: func(*gestures.PointerEnterEvent) { w.handleHover(true) },
			OnExit:  func(*gestures.PointerExitEvent) { w.handleHover(false) },
		})
}

// newBottomSheetGestureDetector makes the detector of the vertical drag of a
// sheet: a drag that is accepted only past the drag threshold.
//
// Flutter: material/bottom_sheet.dart _BottomSheetGestureDetector
func newBottomSheetGestureDetector(onStart gestures.GestureDragStartCallback, onUpdate gestures.GestureDragUpdateCallback,
	onEnd gestures.GestureDragEndCallback, child widgets.Widget) widgets.Widget {
	return &bottomSheetGestureDetector{StatelessWidget: widgets.NewStatelessWidgetBase(), onStart: onStart, onUpdate: onUpdate, onEnd: onEnd, child: child}
}

type bottomSheetGestureDetector struct {
	widgets.StatelessWidget
	onStart  gestures.GestureDragStartCallback
	onUpdate gestures.GestureDragUpdateCallback
	onEnd    gestures.GestureDragEndCallback
	child    widgets.Widget
}

func (w *bottomSheetGestureDetector) Build(bc widgets.BuildContext) widgets.Widget {
	binding := widgets.WidgetsBindingOf(bc).Gestures()
	factory := widgets.NewGestureRecognizerFactoryWithHandlers(
		func() *gestures.VerticalDragGestureRecognizer {
			return gestures.NewVerticalDragGestureRecognizer(binding, gestures.DragGestureRecognizerOpts{DebugOwner: w})
		},
		func(r *gestures.VerticalDragGestureRecognizer) {
			r.OnStart = w.onStart
			r.OnUpdate = w.onUpdate
			r.OnEnd = w.onEnd
			r.OnlyAcceptDragOnThreshold = true
		})
	return widgets.NewRawGestureDetector(w.child, widgets.RawGestureDetectorOpts{
		ExcludeFromSemantics: true,
		Gestures:             map[reflect.Type]widgets.GestureRecognizerFactory{factory.RecognizerType(): factory},
	})
}

// bottomSheetLayoutWithSizeListener lays its child out at the bottom, moved down
// by the animation value, and tells when the size of the child changes.
//
// Flutter: material/bottom_sheet.dart _BottomSheetLayoutWithSizeListener
type bottomSheetLayoutWithSizeListener struct {
	widgets.SingleChildRenderObjectWidget
	onChildSizeChanged                  func(ui.Size)
	animationValue                      float64
	isScrollControlled                  bool
	scrollControlDisabledMaxHeightRatio float64
}

func (w *bottomSheetLayoutWithSizeListener) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderBottomSheetLayoutWithSizeListener{
		onChildSizeChanged: w.onChildSizeChanged, animationValue: w.animationValue,
		isScrollControlled: w.isScrollControlled, scrollControlDisabledMaxHeightRatio: w.scrollControlDisabledMaxHeightRatio}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (w *bottomSheetLayoutWithSizeListener) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderBottomSheetLayoutWithSizeListener)
	// The callback is a new closure at each build: it is not compared.
	r.onChildSizeChanged = w.onChildSizeChanged
	if r.animationValue != w.animationValue || r.isScrollControlled != w.isScrollControlled ||
		r.scrollControlDisabledMaxHeightRatio != w.scrollControlDisabledMaxHeightRatio {
		r.animationValue = w.animationValue
		r.isScrollControlled = w.isScrollControlled
		r.scrollControlDisabledMaxHeightRatio = w.scrollControlDisabledMaxHeightRatio
		r.MarkNeedsLayout()
	}
}

// Flutter: material/bottom_sheet.dart _RenderBottomSheetLayoutWithSizeListener
type renderBottomSheetLayoutWithSizeListener struct {
	rendering.RenderShiftedBox
	lastSize                            ui.Size
	onChildSizeChanged                  func(ui.Size)
	animationValue                      float64
	isScrollControlled                  bool
	scrollControlDisabledMaxHeightRatio float64
}

func (r *renderBottomSheetLayoutWithSizeListener) ComputeMinIntrinsicWidth(float64) float64 { return 0 }
func (r *renderBottomSheetLayoutWithSizeListener) ComputeMaxIntrinsicWidth(float64) float64 { return 0 }
func (r *renderBottomSheetLayoutWithSizeListener) ComputeMinIntrinsicHeight(float64) float64 {
	return 0
}
func (r *renderBottomSheetLayoutWithSizeListener) ComputeMaxIntrinsicHeight(float64) float64 {
	return 0
}

func (r *renderBottomSheetLayoutWithSizeListener) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Biggest()
}

func (r *renderBottomSheetLayoutWithSizeListener) childConstraints(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	maxHeight := constraints.MaxHeight
	if !r.isScrollControlled {
		maxHeight = constraints.MaxHeight * r.scrollControlDisabledMaxHeightRatio
	}
	return rendering.BoxConstraints{MinWidth: constraints.MaxWidth, MaxWidth: constraints.MaxWidth, MaxHeight: maxHeight}
}

func (r *renderBottomSheetLayoutWithSizeListener) positionForChild(size, childSize ui.Size) ui.Offset {
	return ui.Offset{Dx: 0.0, Dy: size.Height - childSize.Height*r.animationValue}
}

func (r *renderBottomSheetLayoutWithSizeListener) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	cc := r.childConstraints(constraints)
	result, ok := child.GetDryBaseline(cc, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := cc.Smallest()
	if !cc.IsTight() {
		childSize = child.GetDryLayout(cc)
	}
	return foundation.Some(result + r.positionForChild(constraints.Biggest(), childSize).Dy)
}

func (r *renderBottomSheetLayoutWithSizeListener) PerformLayout() {
	constraints := r.BoxConstraints()
	r.SetSize(constraints.Biggest())
	child := r.Child()
	if child == nil {
		return
	}
	cc := r.childConstraints(constraints)
	child.Layout(cc, rendering.LayoutOpts{ParentUsesSize: !cc.IsTight()})
	childSize := cc.Smallest()
	if !cc.IsTight() {
		childSize = child.Size()
	}
	rendering.BoxParentDataOf(child).Offset = r.positionForChild(r.Size(), childSize)
	if r.lastSize != childSize {
		r.lastSize = childSize
		r.onChildSizeChanged(r.lastSize)
	}
}

// modalBottomSheet is the page of a modal bottom sheet route.
//
// Flutter: material/bottom_sheet.dart _ModalBottomSheet
type modalBottomSheet struct {
	widgets.StatefulWidget
	route                               *ModalBottomSheetRoute
	backgroundColor                     foundation.Opt[ui.Color]
	elevation                           foundation.Opt[float64]
	shape                               painting.ShapeBorder
	clipBehavior                        ui.Clip
	constraints                         foundation.Opt[rendering.BoxConstraints]
	isScrollControlled                  bool
	scrollControlDisabledMaxHeightRatio float64
	enableDrag                          bool
	showDragHandle                      bool
	animationStyle                      *animation.AnimationStyle
}

func (w *modalBottomSheet) CreateState() widgets.State { return &modalBottomSheetState{} }

type modalBottomSheetState struct {
	widgets.StateOf[*modalBottomSheet]
	sheetAnimation       *animation.ProxyAnimation
	curvedSheetAnimation *animation.CurvedAnimation
}

func (s *modalBottomSheetState) curve() animation.Curve {
	if st := s.Widget().animationStyle; st != nil && st.Curve != nil {
		return st.Curve
	}
	return Easing.LegacyDecelerate
}

func (s *modalBottomSheetState) reverseCurve() animation.Curve {
	if st := s.Widget().animationStyle; st != nil && st.ReverseCurve != nil {
		return st.ReverseCurve
	}
	return Easing.LegacyDecelerate
}

func (s *modalBottomSheetState) InitState() {
	s.curvedSheetAnimation = animation.NewCurvedAnimation(s.Widget().route.Animation(), s.curve(), s.reverseCurve())
	s.sheetAnimation = animation.NewProxyAnimation(s.curvedSheetAnimation)
}

func (s *modalBottomSheetState) Dispose() {
	s.sheetAnimation.SetParent(animation.AlwaysDismissed)
	s.curvedSheetAnimation.Dispose()
}

func (s *modalBottomSheetState) handleDragStart(gestures.DragStartDetails) {
	s.sheetAnimation.SetParent(s.Widget().route.Animation())
}

func (s *modalBottomSheetState) handleDragEnd(_ gestures.DragEndDetails, _ bool) {
	currentProgress := s.Widget().route.Animation().Value()
	s.sheetAnimation.SetParent(animation.NewCurvedAnimation(s.Widget().route.Animation(),
		animation.NewSplit(currentProgress, animation.SplitOpts{EndCurve: s.curve()}),
		animation.NewSplit(currentProgress, animation.SplitOpts{EndCurve: s.reverseCurve()})))
}

func (s *modalBottomSheetState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	widgets.DebugCheckHasMediaQuery(bc)
	localizations := MaterialLocalizationsOf(bc)
	// On the Apple platforms the route has no label.
	routeLabel := localizations.DialogLabel()
	sheet := NewBottomSheet(func() {
		if w.route.IsCurrent() {
			widgets.NavigatorPop(bc)
		}
	}, w.route.builder, BottomSheetOpts{
		AnimationController: w.route.animationController, BackgroundColor: w.backgroundColor, Elevation: w.elevation, Shape: w.shape,
		ClipBehavior: w.clipBehavior, Constraints: w.constraints, NoEnableDrag: !w.enableDrag, ShowDragHandle: foundation.Some(w.showDragHandle),
		OnDragStart: s.handleDragStart, OnDragEnd: s.handleDragEnd,
	})
	return widgets.NewAnimatedBuilder(s.sheetAnimation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewSemantics(
			widgets.NewClipRect(&bottomSheetLayoutWithSizeListener{
				SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child),
				onChildSizeChanged: func(size ui.Size) {
					w.route.didChangeBarrierSemanticsClip(painting.NewEdgeInsetsFromLTRB(0, 0, 0, size.Height))
				},
				animationValue: s.sheetAnimation.Value(), isScrollControlled: w.isScrollControlled,
				scrollControlDisabledMaxHeightRatio: w.scrollControlDisabledMaxHeightRatio,
			}),
			widgets.SemanticsOpts{ScopesRoute: foundation.Some(true), NamesRoute: foundation.Some(true), Label: foundation.Some(routeLabel), ExplicitChildNodes: true})
	}, sheet)
}

// ModalBottomSheetRouteOpts are the optional parameters of
// NewModalBottomSheetRoute.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.new
type ModalBottomSheetRouteOpts struct {
	widgets.ModalRouteOpts
	// CapturedThemes are the inherited themes of the context that shows the sheet,
	// which wrap the page.
	CapturedThemes *widgets.CapturedThemes
	// BarrierLabel is the semantic label of the barrier.
	BarrierLabel string
	// BarrierOnTapHint is the hint of the tap action of the barrier.
	BarrierOnTapHint string
	// BackgroundColor is the color of the sheet.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the sheet.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the sheet.
	Shape painting.ShapeBorder
	// ClipBehavior is the clip of the sheet.
	ClipBehavior ui.Clip
	// Constraints are the size constraints of the sheet.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// ModalBarrierColor is the color of the barrier; the default is black54.
	ModalBarrierColor foundation.Opt[ui.Color]
	// NoDismissible says that tapping the barrier does not dismiss the sheet
	// (Flutter's isDismissible: false).
	NoDismissible bool
	// NoEnableDrag says that the sheet cannot be dragged down to close it.
	NoEnableDrag bool
	// ShowDragHandle says whether the sheet shows a drag handle; the default is
	// the theme's.
	ShowDragHandle foundation.Opt[bool]
	// IsScrollControlled says that the sheet may take the whole height of the
	// screen; the default is false, which limits it to 9/16 of the height.
	IsScrollControlled bool
	// ScrollControlDisabledMaxHeightRatio is the ratio of the height of the screen
	// that the sheet takes at most when it is not scroll controlled; the default is
	// 9/16.
	ScrollControlDisabledMaxHeightRatio foundation.Opt[float64]
	// TransitionAnimationController controls the animation of the sheet; the route
	// does not dispose it.
	TransitionAnimationController *animation.AnimationController
	// UseSafeArea says that the sheet avoids the system intrusions at the top.
	UseSafeArea bool
	// SheetAnimationStyle is the duration and curves of the animation.
	SheetAnimationStyle *animation.AnimationStyle
}

// ModalBottomSheetRoute is a modal route with a bottom sheet in it: it blocks the
// rest of the app until the sheet is dismissed.
//
// Deviation: Flutter's anchorPoint, which the DisplayFeatureSubScreen uses, is
// not ported: Glutter has no display features (SPEC §12.10).
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute
type ModalBottomSheetRoute struct {
	widgets.PopupRouteBase
	builder             widgets.WidgetBuilder
	o                   ModalBottomSheetRouteOpts
	clipDetails         *foundation.ValueNotifier[painting.EdgeInsets]
	animationController *animation.AnimationController
}

// NewModalBottomSheetRoute creates the route; builder builds the content of the
// sheet.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.new
func NewModalBottomSheetRoute(builder widgets.WidgetBuilder, opts ...ModalBottomSheetRouteOpts) *ModalBottomSheetRoute {
	o := firstOpts(opts)
	if !o.ScrollControlDisabledMaxHeightRatio.IsSome() {
		o.ScrollControlDisabledMaxHeightRatio = foundation.Some(bottomSheetDefaultScrollControlDisabledMaxHeightRatio)
	}
	r := &ModalBottomSheetRoute{builder: builder, o: o, clipDetails: foundation.NewValueNotifier(painting.EdgeInsets{})}
	widgets.InitPopupRouteBase(&r.PopupRouteBase, r, o.ModalRouteOpts)
	return r
}

// Dispose releases the notifier of the clip of the barrier.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.dispose
func (r *ModalBottomSheetRoute) Dispose() {
	r.clipDetails.Dispose()
	r.PopupRouteBase.Dispose()
}

func (r *ModalBottomSheetRoute) didChangeBarrierSemanticsClip(newClipDetails painting.EdgeInsets) bool {
	if r.clipDetails.Value() == newClipDetails {
		return false
	}
	r.clipDetails.SetValue(newClipDetails)
	return true
}

// TransitionDuration is the duration of the transition of the controller, or of
// the sheet animation style.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.transitionDuration
func (r *ModalBottomSheetRoute) TransitionDuration() time.Duration {
	if c := r.o.TransitionAnimationController; c != nil && c.Duration.IsSome() {
		return c.Duration.Value()
	}
	if st := r.o.SheetAnimationStyle; st != nil && st.Duration.IsSome() {
		return st.Duration.Value()
	}
	return bottomSheetEnterDuration
}

// ReverseTransitionDuration is the duration of the reverse transition.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.reverseTransitionDuration
func (r *ModalBottomSheetRoute) ReverseTransitionDuration() time.Duration {
	if c := r.o.TransitionAnimationController; c != nil {
		if c.ReverseDuration.IsSome() {
			return c.ReverseDuration.Value()
		}
		if c.Duration.IsSome() {
			return c.Duration.Value()
		}
	}
	if st := r.o.SheetAnimationStyle; st != nil && st.ReverseDuration.IsSome() {
		return st.ReverseDuration.Value()
	}
	return bottomSheetExitDuration
}

// BarrierDismissible is whether tapping the barrier pops the route.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.barrierDismissible
func (r *ModalBottomSheetRoute) BarrierDismissible() bool { return !r.o.NoDismissible }

// BarrierLabel is the semantic label of the barrier.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.barrierLabel
func (r *ModalBottomSheetRoute) BarrierLabel() string { return r.o.BarrierLabel }

// BarrierColor is the color of the barrier: black54 by default.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.barrierColor
func (r *ModalBottomSheetRoute) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.Some(r.o.ModalBarrierColor.OrElse(Colors.Black54))
}

// CreateAnimationController creates the controller of the route: the one that was
// given, or one with the durations of a bottom sheet.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.createAnimationController
func (r *ModalBottomSheetRoute) CreateAnimationController() *animation.AnimationController {
	if r.animationController != nil {
		panic(errors.New(tagModalBottomSheetRouteControllerExists, "the animation controller of a ModalBottomSheetRoute was created twice"))
	}
	if r.o.TransitionAnimationController != nil {
		r.animationController = r.o.TransitionAnimationController
		r.SetWillDisposeAnimationController(false)
	} else {
		r.animationController = BottomSheetCreateAnimationController(r.Navigator(), r.o.SheetAnimationStyle)
	}
	return r.animationController
}

// BuildPage builds the sheet in its page.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.buildPage
func (r *ModalBottomSheetRoute) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	content := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		theme := ThemeOf(bc)
		sheetTheme := theme.BottomSheetTheme
		defaults := bottomSheetDefaults(theme)
		enableDrag := !r.o.NoEnableDrag
		return &modalBottomSheet{
			StatefulWidget: widgets.NewStatefulWidgetBase(), route: r, animationStyle: r.o.SheetAnimationStyle,
			backgroundColor: firstOpt(r.o.BackgroundColor, sheetTheme.ModalBackgroundColor, sheetTheme.BackgroundColor, defaults.BackgroundColor),
			elevation:       firstOpt(r.o.Elevation, sheetTheme.ModalElevation, sheetTheme.Elevation, defaults.ModalElevation),
			shape:           r.o.Shape, clipBehavior: r.o.ClipBehavior, constraints: r.o.Constraints,
			isScrollControlled:                  r.o.IsScrollControlled,
			scrollControlDisabledMaxHeightRatio: r.o.ScrollControlDisabledMaxHeightRatio.Value(),
			enableDrag:                          enableDrag,
			showDragHandle:                      r.o.ShowDragHandle.OrElse(enableDrag && sheetTheme.ShowDragHandle.OrElse(false)),
		}
	})
	var bottomSheet widgets.Widget
	if r.o.UseSafeArea {
		bottomSheet = widgets.NewSafeArea(content, widgets.SafeAreaOpts{Bottom: foundation.Some(false)})
	} else {
		bottomSheet = widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true}, content)
	}
	bottomSheet = widgets.NewSemantics(bottomSheet, widgets.SemanticsOpts{HitTestBehavior: foundation.Some(ui.SemanticsHitTestBehaviorOpaque)})
	if r.o.CapturedThemes != nil {
		return r.o.CapturedThemes.Wrap(bottomSheet)
	}
	return bottomSheet
}

// BuildModalBarrier builds the barrier, which semantics are clipped to the
// part of the screen that the sheet does not cover.
//
// Flutter: material/bottom_sheet.dart ModalBottomSheetRoute.buildModalBarrier
func (r *ModalBottomSheetRoute) BuildModalBarrier() widgets.Widget {
	barrierColor := r.BarrierColor().Value()
	if barrierColor.Alpha() != 0 && !r.Offstage() {
		// ChangedInternalState is called if the barrier color, curve, dismissible,
		// label or hint updates.
		colorAnimation := animation.Drive[foundation.Opt[ui.Color]](r.Animation(),
			animation.NewColorTween(foundation.Some(barrierColor.WithOpacity(0)), foundation.Some(barrierColor)).
				Chain(animation.NewCurveTween(r.BarrierCurve())))
		return widgets.NewAnimatedModalBarrier(colorAnimation, widgets.AnimatedModalBarrierOpts{
			Dismissible: foundation.Some(r.BarrierDismissible()), SemanticsLabel: r.BarrierLabel(),
			BarrierSemanticsDismissible: foundation.Some(r.SemanticsDismissible()), ClipDetailsNotifier: r.clipDetails,
			SemanticsOnTapHint: r.o.BarrierOnTapHint})
	}
	return widgets.NewModalBarrier(widgets.ModalBarrierOpts{
		Dismissible: foundation.Some(r.BarrierDismissible()), SemanticsLabel: r.BarrierLabel(),
		BarrierSemanticsDismissible: foundation.Some(r.SemanticsDismissible()), ClipDetailsNotifier: r.clipDetails,
		SemanticsOnTapHint: r.o.BarrierOnTapHint})
}

// ShowModalBottomSheetOpts are the optional parameters of ShowModalBottomSheet.
//
// Flutter: material/bottom_sheet.dart showModalBottomSheet
type ShowModalBottomSheetOpts struct {
	// BackgroundColor is the color of the sheet.
	BackgroundColor foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier; the default is the
	// scrimLabel of the localizations.
	BarrierLabel string
	// Elevation is the z-coordinate of the sheet.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the sheet.
	Shape painting.ShapeBorder
	// ClipBehavior is the clip of the sheet.
	ClipBehavior ui.Clip
	// Constraints are the size constraints of the sheet.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// BarrierColor is the color of the barrier.
	BarrierColor foundation.Opt[ui.Color]
	// IsScrollControlled says that the sheet may take the whole height.
	IsScrollControlled bool
	// ScrollControlDisabledMaxHeightRatio is the ratio of the height that the sheet
	// takes at most when it is not scroll controlled; the default is 9/16.
	ScrollControlDisabledMaxHeightRatio foundation.Opt[float64]
	// UseRootNavigator pushes the sheet onto the root navigator.
	UseRootNavigator bool
	// NoDismissible says that tapping the barrier does not dismiss the sheet.
	NoDismissible bool
	// NoEnableDrag says that the sheet cannot be dragged down to close it.
	NoEnableDrag bool
	// ShowDragHandle says whether the sheet shows a drag handle.
	ShowDragHandle foundation.Opt[bool]
	// UseSafeArea says that the sheet avoids the system intrusions at the top.
	UseSafeArea bool
	// RouteSettings are the settings of the route.
	RouteSettings widgets.RouteSettings
	// TransitionAnimationController controls the animation of the sheet.
	TransitionAnimationController *animation.AnimationController
	// SheetAnimationStyle is the duration and curves of the animation.
	SheetAnimationStyle *animation.AnimationStyle
	// RequestFocus is whether the sheet requests the focus.
	RequestFocus foundation.Opt[bool]
}

// ShowModalBottomSheet shows a modal Material Design bottom sheet. It returns a
// channel that gets the value that the sheet is popped with, then is closed.
//
// Flutter: material/bottom_sheet.dart showModalBottomSheet
func ShowModalBottomSheet(bc widgets.BuildContext, builder widgets.WidgetBuilder, opts ...ShowModalBottomSheetOpts) <-chan any {
	o := firstOpts(opts)
	widgets.DebugCheckHasMediaQuery(bc)
	DebugCheckHasMaterialLocalizations(bc)
	navigator := widgets.NavigatorOf(bc, widgets.NavigatorOfOpts{RootNavigator: o.UseRootNavigator})
	localizations := MaterialLocalizationsOf(bc)
	barrierLabel := o.BarrierLabel
	if barrierLabel == "" {
		barrierLabel = localizations.ScrimLabel()
	}
	return navigator.Push(NewModalBottomSheetRoute(builder, ModalBottomSheetRouteOpts{
		ModalRouteOpts:     widgets.ModalRouteOpts{RouteOpts: widgets.RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus}},
		CapturedThemes:     widgets.CaptureThemes(bc, navigator.Context()),
		IsScrollControlled: o.IsScrollControlled, ScrollControlDisabledMaxHeightRatio: o.ScrollControlDisabledMaxHeightRatio,
		BarrierLabel: barrierLabel, BarrierOnTapHint: localizations.ScrimOnTapHint(localizations.BottomSheetLabel()),
		BackgroundColor: o.BackgroundColor, Elevation: o.Elevation, Shape: o.Shape, ClipBehavior: o.ClipBehavior,
		Constraints: o.Constraints, NoDismissible: o.NoDismissible,
		ModalBarrierColor: firstOpt(o.BarrierColor, ThemeOf(bc).BottomSheetTheme.ModalBarrierColor),
		NoEnableDrag:      o.NoEnableDrag, ShowDragHandle: o.ShowDragHandle, TransitionAnimationController: o.TransitionAnimationController,
		UseSafeArea: o.UseSafeArea, SheetAnimationStyle: o.SheetAnimationStyle,
	}))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBottomSheetElevationNegative          = "BottomSheetElevationNegative"
	tagBottomSheetNoAnimationController      = "BottomSheetNoAnimationController"
	tagModalBottomSheetRouteControllerExists = "ModalBottomSheetRouteControllerExists"
)
