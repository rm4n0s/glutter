// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DrawerAlignment is the side of the screen that a drawer is at.
//
// Flutter: material/drawer.dart DrawerAlignment
type DrawerAlignment int

const (
	// DrawerAlignmentUnspecified is the zero value: the start.
	DrawerAlignmentUnspecified DrawerAlignment = iota
	// DrawerAlignmentStart places the drawer at the start of the screen: the left
	// for left to right text.
	//
	// Flutter: material/drawer.dart DrawerAlignment.start
	DrawerAlignmentStart
	// DrawerAlignmentEnd places the drawer at the end of the screen.
	//
	// Flutter: material/drawer.dart DrawerAlignment.end
	DrawerAlignmentEnd
)

// Name is the name of the value, as Dart's Enum.name.
func (a DrawerAlignment) Name() string { return enumName(int(a), "start", "end") }

// String is "DrawerAlignment." and the name.
func (a DrawerAlignment) String() string { return "DrawerAlignment." + a.Name() }

// IsEnd is whether the alignment is the end.
func (a DrawerAlignment) IsEnd() bool { return a == DrawerAlignmentEnd }

// Flutter: material/drawer.dart _kWidth, _kEdgeDragWidth, _kMinFlingVelocity and
// _kBaseSettleDuration
const (
	drawerWidth                   = 304.0
	drawerEdgeDragWidth           = 20.0
	drawerMinFlingVelocity        = 365.0
	drawerBaseSettleDurationValue = 246 * time.Millisecond
)

// DrawerOpts are the optional parameters of NewDrawer.
//
// Flutter: material/drawer.dart Drawer.new
type DrawerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the color of the drawer's Material.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the drawer.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the drawer.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the drawer (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the drawer.
	Shape painting.ShapeBorder
	// Width is the width of the drawer; the default is 304.
	Width foundation.Opt[float64]
	// Child is the content of the drawer.
	Child widgets.Widget
	// SemanticLabel is the semantic label of the dialog the drawer is; the default
	// is the drawerLabel of the localizations.
	SemanticLabel string
	// ClipBehavior is the clip of the drawer's Material, when it has a shape.
	ClipBehavior ui.Clip
}

// Drawer is a Material Design panel that slides in horizontally from the edge of
// a Scaffold to show navigation links in an application.
//
// Flutter: material/drawer.dart Drawer
type Drawer struct {
	widgets.StatelessWidget
	o DrawerOpts
}

// NewDrawer creates a drawer. In debug builds it panics with the tag
// DrawerElevationNegative when the elevation is negative.
//
// Flutter: material/drawer.dart Drawer.new
func NewDrawer(opts ...DrawerOpts) *Drawer {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New("DrawerElevationNegative", "the elevation of a Drawer must not be negative", "elevation", o.Elevation.Value()))
	}
	return &Drawer{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Child is the content of the drawer.
//
// Flutter: material/drawer.dart Drawer.child
func (w *Drawer) Child() widgets.Widget { return w.o.Child }

// Build builds the drawer.
//
// Flutter: material/drawer.dart Drawer.build
func (w *Drawer) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	drawerTheme := DrawerThemeOf(bc)
	theme := ThemeOf(bc)
	label := w.o.SemanticLabel
	switch theme.Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
	default:
		if label == "" {
			label = MaterialLocalizationsOf(bc).DrawerLabel()
		}
	}
	isDrawerStart := true
	if c := DrawerControllerMaybeOf(bc); c != nil && c.Alignment().IsEnd() {
		isDrawerStart = false
	}
	defaults := drawerDefaults(bc)
	var effectiveShape painting.ShapeBorder
	switch {
	case w.o.Shape != nil:
		effectiveShape = w.o.Shape
	case isDrawerStart && drawerTheme.Shape != nil:
		effectiveShape = drawerTheme.Shape
	case isDrawerStart:
		effectiveShape = defaults.Shape
	case drawerTheme.EndShape != nil:
		effectiveShape = drawerTheme.EndShape
	default:
		effectiveShape = defaults.EndShape
	}
	clip := ui.ClipNone
	if effectiveShape != nil {
		clip = w.o.ClipBehavior
		if clip == ui.ClipUnspecified {
			clip = drawerTheme.ClipBehavior
		}
		if clip == ui.ClipUnspecified {
			clip = defaults.ClipBehavior
		}
	}
	return widgets.NewSemantics(
		widgets.NewConstrainedBox(
			rendering.NewBoxConstraintsExpand(rendering.ExpandOpts{Width: foundation.Some(firstOpt(w.o.Width, drawerTheme.Width).OrElse(drawerWidth))}),
			NewMaterial(w.o.Child, MaterialOpts{
				Color:            firstOpt(w.o.BackgroundColor, drawerTheme.BackgroundColor, defaults.BackgroundColor),
				Elevation:        firstOpt(w.o.Elevation, drawerTheme.Elevation, defaults.Elevation).Value(),
				ShadowColor:      firstOpt(w.o.ShadowColor, drawerTheme.ShadowColor, defaults.ShadowColor),
				SurfaceTintColor: firstOpt(w.o.SurfaceTintColor, drawerTheme.SurfaceTintColor, defaults.SurfaceTintColor),
				Shape:            effectiveShape, ClipBehavior: clip,
			})),
		widgets.SemanticsOpts{ScopesRoute: foundation.Some(true), NamesRoute: foundation.Some(true), ExplicitChildNodes: true, Label: foundation.Some(label)})
}

// drawerDefaults are the defaults of the drawer of the theme of the context.
func drawerDefaults(bc widgets.BuildContext) DrawerThemeData {
	theme := ThemeOf(bc)
	if !theme.UseMaterial3 {
		return drawerDefaultsM2(theme)
	}
	return drawerDefaultsM3(theme, widgets.DirectionalityOf(bc))
}

// drawerDefaultsM2 are the hand coded defaults for Material 2.
//
// Flutter: material/drawer.dart _DrawerDefaultsM2
func drawerDefaultsM2(theme *ThemeData) DrawerThemeData {
	return DrawerThemeData{Elevation: foundation.Some(16.0), ClipBehavior: ui.ClipHardEdge, ShadowColor: foundation.Some(theme.ShadowColor)}
}

// drawerDefaultsM3 are the Material 3 defaults.
//
// Flutter: material/drawer.dart _DrawerDefaultsM3
func drawerDefaultsM3(theme *ThemeData, direction ui.TextDirection) DrawerThemeData {
	// There isn't currently a token for the shapes, but they are shown in the spec,
	// so they are hard coded here for now.
	end := painting.NewBorderRadiusDirectionalHorizontal(ui.Radius{}, ui.NewRadiusCircular(16)).Resolve(direction)
	start := painting.NewBorderRadiusDirectionalHorizontal(ui.NewRadiusCircular(16), ui.Radius{}).Resolve(direction)
	return DrawerThemeData{
		Elevation: foundation.Some(1.0), ClipBehavior: ui.ClipHardEdge,
		BackgroundColor:  foundation.Some(theme.ColorScheme.SurfaceContainerLow()),
		SurfaceTintColor: foundation.Some(Colors.Transparent), ShadowColor: foundation.Some(Colors.Transparent),
		Shape:    painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: end}),
		EndShape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: start}),
	}
}

// DrawerCallback is called when the drawer opens or closes.
//
// Flutter: material/drawer.dart DrawerCallback
type DrawerCallback func(isOpened bool)

// drawerControllerScope gives the descendants of a drawer its controller widget.
//
// Flutter: material/drawer.dart _DrawerControllerScope
type drawerControllerScope struct {
	widgets.InheritedWidget
	controller *DrawerController
}

func (w *drawerControllerScope) UpdateShouldNotify(old widgets.Widget) bool {
	return w.controller != old.(*drawerControllerScope).controller
}

// DrawerControllerOpts are the optional parameters of NewDrawerController.
//
// Flutter: material/drawer.dart DrawerController.new
type DrawerControllerOpts struct {
	// Key is the key of the widget, typically a GlobalKey of the DrawerControllerState
	// that opens and closes the drawer.
	Key foundation.Key
	// IsDrawerOpen is whether the drawer starts open.
	IsDrawerOpen bool
	// DrawerCallback is called when the drawer is opened or closed.
	DrawerCallback DrawerCallback
	// DragStartBehavior is how the drags start; the default is start.
	DragStartBehavior gestures.DragStartBehavior
	// ScrimColor is the color of the scrim behind the drawer.
	ScrimColor foundation.Opt[ui.Color]
	// EdgeDragWidth is the width of the area of the edge of the screen that opens
	// the drawer by a drag; the default is 20 plus the padding of the side.
	EdgeDragWidth foundation.Opt[float64]
	// NoEnableOpenDragGesture says that the drawer cannot be opened by dragging
	// from the edge (Flutter's enableOpenDragGesture: false).
	NoEnableOpenDragGesture bool
	// NoDrawerBarrierDismissible says that tapping the scrim does not close the
	// drawer (Flutter's drawerBarrierDismissible: false).
	NoDrawerBarrierDismissible bool
}

// DrawerController provides interactive behavior for Drawer widgets: it slides
// the drawer in and out, and closes it with the barrier and the back action.
//
// Flutter: material/drawer.dart DrawerController
type DrawerController struct {
	widgets.StatefulWidget
	child     widgets.Widget
	alignment DrawerAlignment
	o         DrawerControllerOpts
}

// NewDrawerController creates the controller of a drawer.
//
// Flutter: material/drawer.dart DrawerController.new
func NewDrawerController(alignment DrawerAlignment, child widgets.Widget, opts ...DrawerControllerOpts) *DrawerController {
	o := firstOpts(opts)
	return &DrawerController{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, alignment: alignment, o: o}
}

// Child is the drawer.
//
// Flutter: material/drawer.dart DrawerController.child
func (w *DrawerController) Child() widgets.Widget { return w.child }

// Alignment is the side of the screen of the drawer.
//
// Flutter: material/drawer.dart DrawerController.alignment
func (w *DrawerController) Alignment() DrawerAlignment { return w.alignment }

// DrawerBarrierDismissible is whether tapping the scrim closes the drawer.
//
// Flutter: material/drawer.dart DrawerController.drawerBarrierDismissible
func (w *DrawerController) DrawerBarrierDismissible() bool { return !w.o.NoDrawerBarrierDismissible }

// EnableOpenDragGesture is whether the drawer can be opened by a drag from the
// edge.
//
// Flutter: material/drawer.dart DrawerController.enableOpenDragGesture
func (w *DrawerController) EnableOpenDragGesture() bool { return !w.o.NoEnableOpenDragGesture }

// DrawerControllerMaybeOf returns the DrawerController of the closest drawer above
// the context, or nil. The context depends on it.
//
// Flutter: material/drawer.dart DrawerController.maybeOf
func DrawerControllerMaybeOf(bc widgets.BuildContext) *DrawerController {
	if scope, ok := widgets.DependOn[*drawerControllerScope](bc); ok {
		return scope.controller
	}
	return nil
}

// DrawerControllerOf is like DrawerControllerMaybeOf but panics with the tag
// DrawerControllerMissing in debug builds when there is none.
//
// Flutter: material/drawer.dart DrawerController.of
func DrawerControllerOf(bc widgets.BuildContext) *DrawerController {
	c := DrawerControllerMaybeOf(bc)
	if c == nil {
		panic(errors.New("DrawerControllerMissing", "DrawerController.of() was called with a context that does not contain a DrawerController widget. "+
			"No DrawerController widget ancestor could be found starting from the context that was passed to DrawerController.of().",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return c
}

// CreateState creates the state.
//
// Flutter: material/drawer.dart DrawerController.createState
func (w *DrawerController) CreateState() widgets.State { return &DrawerControllerState{} }

// DrawerControllerState is the state of a DrawerController: Open and Close slide
// the drawer.
//
// Flutter: material/drawer.dart DrawerControllerState
type DrawerControllerState struct {
	widgets.StateOf[*DrawerController]
	widgets.SingleTickerProvider

	controller         *animation.AnimationController
	historyEntry       *widgets.LocalHistoryEntry
	focusScopeNode     *widgets.FocusScopeNode
	drawerKey          *widgets.GlobalKey[widgets.State]
	gestureDetectorKey *widgets.GlobalKey[widgets.State]
	previouslyOpened   bool
}

// InitState creates the animation controller.
//
// Flutter: material/drawer.dart DrawerControllerState.initState
func (s *DrawerControllerState) InitState() {
	s.drawerKey = widgets.NewGlobalKey[widgets.State]()
	s.gestureDetectorKey = widgets.NewGlobalKey[widgets.State]()
	s.focusScopeNode = widgets.NewFocusScopeNode()
	value := 0.0
	if s.Widget().o.IsDrawerOpen {
		value = 1.0
	}
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Value: foundation.Some(value), Duration: foundation.Some(drawerBaseSettleDurationValue)})
	s.controller.AddListener(s.animationChanged)
	s.controller.AddStatusListener(s.animationStatusChanged)
}

// Dispose releases the history entry, the controller and the focus scope.
//
// Flutter: material/drawer.dart DrawerControllerState.dispose
func (s *DrawerControllerState) Dispose() {
	if s.historyEntry != nil {
		s.historyEntry.Remove()
	}
	s.controller.Dispose()
	s.focusScopeNode.Dispose()
}

// DidUpdateWidget snaps the drawer open or shut when the widget says so and the
// user is not dragging it.
//
// Flutter: material/drawer.dart DrawerControllerState.didUpdateWidget
func (s *DrawerControllerState) DidUpdateWidget(old *DrawerController) {
	if s.controller.Status().IsAnimating() {
		return // Don't snap the drawer open or shut while the user is dragging.
	}
	if s.Widget().o.IsDrawerOpen != old.o.IsDrawerOpen {
		if s.Widget().o.IsDrawerOpen {
			s.controller.SetValue(1.0)
		} else {
			s.controller.SetValue(0.0)
		}
	}
}

func (s *DrawerControllerState) animationChanged() {
	// The animation controller's state is our build state, and it changed already.
	s.SetState(func() {})
}

func (s *DrawerControllerState) ensureHistoryEntry() {
	if s.historyEntry != nil {
		return
	}
	if route := widgets.ModalRouteOf(s.Context()); route != nil {
		s.historyEntry = widgets.NewLocalHistoryEntry(widgets.LocalHistoryEntryOpts{
			OnRemove: s.handleHistoryEntryRemoved, ImpliesAppBarDismissal: foundation.Some(false)})
		route.AddLocalHistoryEntry(s.historyEntry)
		widgets.FocusScopeOf(s.Context()).SetFirstFocus(s.focusScopeNode)
	}
}

func (s *DrawerControllerState) animationStatusChanged(status animation.AnimationStatus) {
	switch status {
	case animation.AnimationStatusForward:
		s.ensureHistoryEntry()
	case animation.AnimationStatusReverse:
		if s.historyEntry != nil {
			s.historyEntry.Remove()
		}
		s.historyEntry = nil
	}
}

func (s *DrawerControllerState) handleHistoryEntryRemoved() {
	s.historyEntry = nil
	s.Close()
}

func (s *DrawerControllerState) handleDragDown(gestures.DragDownDetails) {
	s.controller.Stop()
	s.ensureHistoryEntry()
}

func (s *DrawerControllerState) handleDragCancel() {
	if s.controller.Status().IsDismissed() || s.controller.IsAnimating() {
		return
	}
	if s.controller.Value() < 0.5 {
		s.Close()
	} else {
		s.Open()
	}
}

func (s *DrawerControllerState) width() float64 {
	// drawerWidth if the drawer is not being shown currently.
	if bc := s.drawerKey.CurrentContext(); bc != nil {
		if box, ok := bc.FindRenderObject().(rendering.Box); ok {
			return box.Size().Width
		}
	}
	return drawerWidth
}

func (s *DrawerControllerState) directionFactor() float64 {
	rtl := widgets.DirectionalityOf(s.Context()) == ui.TextDirectionRTL
	start := !s.Widget().alignment.IsEnd()
	if rtl == start {
		return -1
	}
	return 1
}

func (s *DrawerControllerState) move(details gestures.DragUpdateDetails) {
	s.controller.SetValue(s.controller.Value() + details.PrimaryDelta.Value()/s.width()*s.directionFactor())
	opened := s.controller.Value() > 0.5
	if cb := s.Widget().o.DrawerCallback; opened != s.previouslyOpened && cb != nil {
		cb(opened)
	}
	s.previouslyOpened = opened
}

func (s *DrawerControllerState) settle(details gestures.DragEndDetails) {
	if s.controller.Status().IsDismissed() {
		return
	}
	xVelocity := details.Velocity.PixelsPerSecond.Dx
	switch {
	case xVelocity >= drawerMinFlingVelocity || xVelocity <= -drawerMinFlingVelocity:
		visualVelocity := xVelocity / s.width() * s.directionFactor()
		s.controller.Fling(animation.FlingOpts{Velocity: foundation.Some(visualVelocity)})
		if cb := s.Widget().o.DrawerCallback; cb != nil {
			cb(visualVelocity > 0.0)
		}
	case s.controller.Value() < 0.5:
		s.Close()
	default:
		s.Open()
	}
}

// Open opens the drawer.
//
// Flutter: material/drawer.dart DrawerControllerState.open
func (s *DrawerControllerState) Open() {
	s.controller.Fling()
	if cb := s.Widget().o.DrawerCallback; cb != nil {
		cb(true)
	}
}

// Close closes the drawer.
//
// Flutter: material/drawer.dart DrawerControllerState.close
func (s *DrawerControllerState) Close() {
	s.controller.Fling(animation.FlingOpts{Velocity: foundation.Some(-1.0)})
	if cb := s.Widget().o.DrawerCallback; cb != nil {
		cb(false)
	}
}

func (s *DrawerControllerState) drawerOuterAlignment() painting.AlignmentDirectional {
	if s.Widget().alignment.IsEnd() {
		return painting.AlignmentDirectionalCenterEnd
	}
	return painting.AlignmentDirectionalCenterStart
}

func (s *DrawerControllerState) drawerInnerAlignment() painting.AlignmentDirectional {
	if s.Widget().alignment.IsEnd() {
		return painting.AlignmentDirectionalCenterStart
	}
	return painting.AlignmentDirectionalCenterEnd
}

func (s *DrawerControllerState) buildDrawer(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	isDesktop := false
	switch ThemeOf(bc).Platform {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformLinux, foundation.TargetPlatformWindows:
		isDesktop = true
	}
	var dragAreaWidth float64
	if w.o.EdgeDragWidth.IsSome() {
		dragAreaWidth = w.o.EdgeDragWidth.Value()
	} else {
		padding := widgets.MediaQueryPaddingOf(bc)
		side := padding.Left
		ltr := widgets.DirectionalityOf(bc) == ui.TextDirectionLTR
		if ltr == w.alignment.IsEnd() {
			side = padding.Right
		}
		dragAreaWidth = drawerEdgeDragWidth + side
	}

	if s.controller.Status().IsDismissed() {
		if w.EnableOpenDragGesture() && !isDesktop {
			return widgets.NewAlign(
				widgets.NewGestureDetector(
					widgets.NewLimitedBox(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(dragAreaWidth), Height: foundation.Some(infinity())}),
						widgets.LimitedBoxOpts{MaxHeight: foundation.Some(0.0)}),
					widgets.GestureDetectorOpts{
						Key: s.gestureDetectorKey, OnHorizontalDragUpdate: s.move, OnHorizontalDragEnd: s.settle,
						Behavior: rendering.HitTestBehaviorTranslucent, ExcludeFromSemantics: true, DragStartBehavior: w.o.DragStartBehavior}),
				widgets.AlignOpts{Alignment: s.drawerOuterAlignment()})
		}
		return widgets.NewSizedBoxShrink(nil)
	}

	// On Android, the back button is used to dismiss a modal.
	platformHasBackButton := ThemeOf(bc).Platform == foundation.TargetPlatformAndroid
	scrimColor := firstOpt(w.o.ScrimColor, DrawerThemeOf(bc).ScrimColor).OrElse(Colors.Black54)
	effectiveScrimColor := scrimColor.WithOpacity(float64(scrimColor.Alpha()) / 255 * s.controller.Value())
	drawerScrim := widgets.NewColoredBox(effectiveScrimColor,
		widgets.NewLimitedBox(widgets.NewSizedBoxExpand(nil), widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0), MaxHeight: foundation.Some(0.0)}))

	var onTap func()
	if w.DrawerBarrierDismissible() {
		onTap = s.Close
	}
	child := widgets.Widget(&drawerControllerScope{
		InheritedWidget: widgets.NewInheritedWidgetBase(widgets.NewRepaintBoundary(widgets.NewStack([]widgets.Widget{
			widgets.NewBlockSemantics(widgets.NewExcludeSemantics(
				widgets.NewGestureDetector(
					widgets.NewSemantics(drawerScrim, widgets.SemanticsOpts{Label: foundation.Some(MaterialLocalizationsOf(bc).ModalBarrierDismissLabel())}),
					widgets.GestureDetectorOpts{OnTap: onTap}),
				widgets.ExcludeSemanticsOpts{Excluding: foundation.Some(platformHasBackButton)})),
			widgets.NewAlign(
				widgets.NewAlign(
					widgets.NewRepaintBoundary(widgets.NewFocusScope(w.child, widgets.FocusScopeOpts{Key: s.drawerKey, Node: s.focusScopeNode})),
					widgets.AlignOpts{Alignment: s.drawerInnerAlignment(), WidthFactor: foundation.Some(s.controller.Value())}),
				widgets.AlignOpts{Alignment: s.drawerOuterAlignment()}),
		}))),
		controller: w,
	})
	if isDesktop {
		return child
	}
	return widgets.NewGestureDetector(child, widgets.GestureDetectorOpts{
		Key: s.gestureDetectorKey, OnHorizontalDragDown: s.handleDragDown, OnHorizontalDragUpdate: s.move, OnHorizontalDragEnd: s.settle,
		OnHorizontalDragCancel: s.handleDragCancel, ExcludeFromSemantics: true, DragStartBehavior: w.o.DragStartBehavior})
}

// Build builds the drawer: the scrim and the sliding drawer, or the area of the
// edge that opens it.
//
// Flutter: material/drawer.dart DrawerControllerState.build
func (s *DrawerControllerState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	return ListTileThemeMerge(ListTileThemeData{Style: ListTileStyleDrawer}, s.buildDrawer(bc))
}

func infinity() float64 { return math.Inf(1) }
