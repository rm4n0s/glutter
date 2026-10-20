// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// FloatingActionButtonMargin is the margin between a floating action button
// and the edges of the scaffold.
//
// Flutter: material/floating_action_button_location.dart kFloatingActionButtonMargin
const FloatingActionButtonMargin = 16.0

// scaffoldSlot is the id of a child of the layout of a scaffold.
//
// Flutter: material/scaffold.dart _ScaffoldSlot
type scaffoldSlot int

const (
	scaffoldSlotBody scaffoldSlot = iota
	scaffoldSlotAppBar
	scaffoldSlotSnackBar
	scaffoldSlotMaterialBanner
	scaffoldSlotPersistentFooter
	scaffoldSlotBottomNavigationBar
	scaffoldSlotFloatingActionButton
	// TODO(step 13.2): bodyScrim, bottomSheet, drawer and endDrawer. TODO(step
	// 13.2 or later): statusBar, the tap target for the status bar of iOS.
)

// ScaffoldPrelayoutGeometry holds the geometry that a FloatingActionButtonLocation
// uses to place the button, before the scaffold lays its children out.
//
// Flutter: material/scaffold.dart ScaffoldPrelayoutGeometry
type ScaffoldPrelayoutGeometry struct {
	// FloatingActionButtonSize is the size of the floating action button.
	FloatingActionButtonSize ui.Size
	// BottomSheetSize is the size of the bottom sheet.
	BottomSheetSize ui.Size
	// ContentBottom is the distance from the top of the scaffold to the bottom of
	// the content: the body, above the bottom widgets and the on-screen keyboard.
	ContentBottom float64
	// ContentTop is the distance from the top of the scaffold to the top of the
	// content: the bottom of the app bar.
	ContentTop float64
	// MinInsets are the minimum padding the content must leave around itself.
	MinInsets painting.EdgeInsets
	// MinViewPadding is the minimum view padding.
	MinViewPadding painting.EdgeInsets
	// ScaffoldSize is the size of the scaffold.
	ScaffoldSize ui.Size
	// SnackBarSize is the size of the snack bar.
	SnackBarSize ui.Size
	// MaterialBannerSize is the size of the material banner.
	MaterialBannerSize ui.Size
	// TextDirection is the direction of the scaffold.
	TextDirection ui.TextDirection
}

// FloatingActionButtonLocation places the floating action button in a scaffold.
//
// TODO(step 13.3): the other locations, the animator and the transition.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation
type FloatingActionButtonLocation interface {
	// GetOffset is the position of the top left of the button for the geometry.
	//
	// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.getOffset
	GetOffset(geometry ScaffoldPrelayoutGeometry) ui.Offset
}

type fabLocationEndFloat struct{}

// FloatingActionButtonLocationEndFloat places the button at the end of the
// screen, floating above the bottom widgets, with a margin. It is the default.
//
// Flutter: material/floating_action_button_location.dart FloatingActionButtonLocation.endFloat
var FloatingActionButtonLocationEndFloat FloatingActionButtonLocation = fabLocationEndFloat{}

func (fabLocationEndFloat) GetOffset(g ScaffoldPrelayoutGeometry) ui.Offset {
	var x float64
	if g.TextDirection == ui.TextDirectionRTL {
		// Left offset.
		x = FloatingActionButtonMargin + g.MinInsets.Left
	} else {
		// Right offset.
		x = g.ScaffoldSize.Width - FloatingActionButtonMargin - g.MinInsets.Right - g.FloatingActionButtonSize.Width
	}
	// FabFloatOffsetY.
	contentBottom := g.ContentBottom
	bottomContentHeight := g.ScaffoldSize.Height - contentBottom
	bottomSheetHeight := g.BottomSheetSize.Height
	fabHeight := g.FloatingActionButtonSize.Height
	snackBarHeight := g.SnackBarSize.Height
	safeMargin := math.Max(FloatingActionButtonMargin, g.MinViewPadding.Bottom-bottomContentHeight+FloatingActionButtonMargin)
	fabY := contentBottom - fabHeight - safeMargin
	if snackBarHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-snackBarHeight-fabHeight-FloatingActionButtonMargin)
	}
	if bottomSheetHeight > 0.0 {
		fabY = math.Min(fabY, contentBottom-bottomSheetHeight-fabHeight/2.0)
	}
	return ui.Offset{Dx: x, Dy: fabY}
}

// ScaffoldGeometry holds the geometry of the parts of a scaffold that its
// descendants, such as a BottomAppBar, need to know about.
//
// Flutter: material/scaffold.dart ScaffoldGeometry
type ScaffoldGeometry struct {
	// BottomNavigationBarTop is the distance from the scaffold's top to the top of
	// the bottom navigation bar, when there is one.
	BottomNavigationBarTop foundation.Opt[float64]
	// FloatingActionButtonArea is the rectangle of the floating action button, when
	// there is one, in the coordinate system of the scaffold.
	FloatingActionButtonArea foundation.Opt[ui.Rect]
}

// CopyWith returns the geometry with the set fields replaced.
//
// Flutter: material/scaffold.dart ScaffoldGeometry.copyWith
func (g ScaffoldGeometry) CopyWith(bottomNavigationBarTop foundation.Opt[float64], floatingActionButtonArea foundation.Opt[ui.Rect]) ScaffoldGeometry {
	if bottomNavigationBarTop.IsSome() {
		g.BottomNavigationBarTop = bottomNavigationBarTop
	}
	if floatingActionButtonArea.IsSome() {
		g.FloatingActionButtonArea = floatingActionButtonArea
	}
	return g
}

func (g ScaffoldGeometry) scaleFloatingActionButton(scaleFactor float64) ScaffoldGeometry {
	if scaleFactor == 1.0 {
		return g
	}
	if scaleFactor == 0.0 {
		return ScaffoldGeometry{BottomNavigationBarTop: g.BottomNavigationBarTop}
	}
	area := g.FloatingActionButtonArea.Value()
	center := area.Center()
	scaled := ui.RectLerp(&ui.Rect{Left: center.Dx, Top: center.Dy, Right: center.Dx, Bottom: center.Dy}, &area, scaleFactor)
	return g.CopyWith(foundation.None[float64](), foundation.Some(*scaled))
}

// scaffoldGeometryNotifier notifies the listeners of the geometry of a scaffold
// when its layout changes it.
//
// Deviation: Flutter asserts that value is read during the paint phase; Glutter
// does not check (SPEC §12.10).
//
// Flutter: material/scaffold.dart _ScaffoldGeometryNotifier
type scaffoldGeometryNotifier struct {
	foundation.ChangeNotifier
	floatingActionButtonScale float64
	geometry                  ScaffoldGeometry
}

func (n *scaffoldGeometryNotifier) Value() ScaffoldGeometry {
	return n.geometry.scaleFloatingActionButton(n.floatingActionButtonScale)
}

func (n *scaffoldGeometryNotifier) updateWith(bottomNavigationBarTop foundation.Opt[float64], floatingActionButtonArea foundation.Opt[ui.Rect]) {
	n.geometry = n.geometry.CopyWith(bottomNavigationBarTop, floatingActionButtonArea)
	n.NotifyListeners()
}

// scaffoldBodyInfo is what the layout of the scaffold tells the builder of the
// body about the other slots: Flutter passes it in a BoxConstraints subclass.
//
// Flutter: material/scaffold.dart _BodyBoxConstraints
type scaffoldBodyInfo struct {
	bottomWidgetsHeight  float64
	appBarHeight         float64
	materialBannerHeight float64
}

// scaffoldBodyBuilder wraps the body in a MediaQuery whose padding accounts for
// the height of the bottom widgets and the app bar, when the body extends under
// them.
//
// Flutter: material/scaffold.dart _BodyBuilder
func scaffoldBodyBuilder(extendBody, extendBodyBehindAppBar bool, info *scaffoldBodyInfo, body widgets.Widget) widgets.Widget {
	if !extendBody && !extendBodyBehindAppBar {
		return body
	}
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, _ rendering.BoxConstraints) widgets.Widget {
		metrics := widgets.MediaQueryOf(bc)
		bottom := metrics.Padding.Bottom
		if extendBody {
			bottom = math.Max(metrics.Padding.Bottom, info.bottomWidgetsHeight)
		}
		top := metrics.Padding.Top
		if extendBodyBehindAppBar {
			top = math.Max(metrics.Padding.Top, info.appBarHeight+info.materialBannerHeight)
		}
		return widgets.NewMediaQuery(metrics.CopyWith(widgets.MediaQueryDataCopyOpts{
			Padding: foundation.Some(metrics.Padding.CopyWith(painting.EdgeInsetsCopy{Top: foundation.Some(top), Bottom: foundation.Some(bottom)}))}), body)
	}, widgets.LayoutBuilderOpts{Extra: func() any { return *info }})
}

// scaffoldLayout positions the slots of a scaffold.
//
// Flutter: material/scaffold.dart _ScaffoldLayout
type scaffoldLayout struct {
	rendering.MultiChildLayoutDelegateBase
	extendBody                     bool
	extendBodyBehindAppBar         bool
	minInsets                      painting.EdgeInsets
	minViewPadding                 painting.EdgeInsets
	textDirection                  ui.TextDirection
	geometryNotifier               *scaffoldGeometryNotifier
	floatingActionButtonLocation   FloatingActionButtonLocation
	isSnackBarFloating             bool
	snackBarWidth                  foundation.Opt[float64]
	extendBodyBehindMaterialBanner bool
	bodyInfo                       *scaffoldBodyInfo
}

func (d *scaffoldLayout) PerformLayout(size ui.Size) {
	looseConstraints := rendering.NewBoxConstraintsLoose(size)

	// This part of the layout has the same effect as putting the app bar and the
	// body in a column and making the body flexible. What's different is that in
	// this case the app bar appears _after_ the body in the stacking order, so the
	// app bar's shadow is drawn on top of the body.
	fullWidthConstraints := looseConstraints.Tighten(rendering.TightenOpts{Width: foundation.Some(size.Width)})
	bottom := size.Height
	contentTop := 0.0
	bottomWidgetsHeight := 0.0
	appBarHeight := 0.0

	if d.HasChild(scaffoldSlotAppBar) {
		appBarHeight = d.LayoutChild(scaffoldSlotAppBar, fullWidthConstraints).Height
		if !d.extendBodyBehindAppBar {
			contentTop = appBarHeight
		}
		d.PositionChild(scaffoldSlotAppBar, ui.Offset{})
	}

	bottomNavigationBarTop := foundation.None[float64]()
	if d.HasChild(scaffoldSlotBottomNavigationBar) {
		h := d.LayoutChild(scaffoldSlotBottomNavigationBar, fullWidthConstraints).Height
		bottomWidgetsHeight += h
		top := math.Max(0.0, bottom-bottomWidgetsHeight)
		bottomNavigationBarTop = foundation.Some(top)
		d.PositionChild(scaffoldSlotBottomNavigationBar, ui.Offset{Dx: 0, Dy: top})
	}

	if d.HasChild(scaffoldSlotPersistentFooter) {
		footerConstraints := rendering.BoxConstraints{MaxWidth: fullWidthConstraints.MaxWidth, MaxHeight: math.Max(0.0, bottom-bottomWidgetsHeight-contentTop)}
		h := d.LayoutChild(scaffoldSlotPersistentFooter, footerConstraints).Height
		bottomWidgetsHeight += h
		d.PositionChild(scaffoldSlotPersistentFooter, ui.Offset{Dx: 0, Dy: math.Max(0.0, bottom-bottomWidgetsHeight)})
	}

	materialBannerSize := ui.Size{}
	if d.HasChild(scaffoldSlotMaterialBanner) {
		materialBannerSize = d.LayoutChild(scaffoldSlotMaterialBanner, fullWidthConstraints)
		d.PositionChild(scaffoldSlotMaterialBanner, ui.Offset{Dx: 0, Dy: appBarHeight})
		// Push content down only if elevation is 0.
		if !d.extendBodyBehindMaterialBanner {
			contentTop += materialBannerSize.Height
		}
	}

	// Set the content bottom to account for the greater of the height of any bottom
	// widgets or of the on-screen keyboard.
	contentBottom := math.Max(0.0, bottom-math.Max(d.minInsets.Bottom, bottomWidgetsHeight))

	if d.HasChild(scaffoldSlotBody) {
		bodyMaxHeight := math.Max(0.0, contentBottom-contentTop)
		if d.extendBody && d.minInsets.Bottom <= bottomWidgetsHeight {
			bodyMaxHeight += bottomWidgetsHeight
			bodyMaxHeight = ui.ClampDouble(bodyMaxHeight, 0.0, looseConstraints.MaxHeight-contentTop)
		} else {
			bottomWidgetsHeight = 0.0
		}
		*d.bodyInfo = scaffoldBodyInfo{bottomWidgetsHeight: bottomWidgetsHeight, appBarHeight: appBarHeight, materialBannerHeight: materialBannerSize.Height}
		d.LayoutChild(scaffoldSlotBody, rendering.BoxConstraints{MaxWidth: fullWidthConstraints.MaxWidth, MaxHeight: bodyMaxHeight})
		d.PositionChild(scaffoldSlotBody, ui.Offset{Dx: 0, Dy: contentTop})
	}

	// TODO(step 13.2): the body scrim and the bottom sheet.
	bottomSheetSize := ui.Size{}
	snackBarSize := ui.Size{}
	if d.HasChild(scaffoldSlotSnackBar) && !d.isSnackBarFloating {
		snackBarSize = d.LayoutChild(scaffoldSlotSnackBar, fullWidthConstraints)
	}

	floatingActionButtonRect := ui.Rect{}
	if d.HasChild(scaffoldSlotFloatingActionButton) {
		fabSize := d.LayoutChild(scaffoldSlotFloatingActionButton, looseConstraints)
		// TODO(step 13.3): the previous location and the animator move the button
		// from the old position to the new one.
		currentGeometry := ScaffoldPrelayoutGeometry{
			BottomSheetSize: bottomSheetSize, ContentBottom: contentBottom, ContentTop: appBarHeight, FloatingActionButtonSize: fabSize,
			MinInsets: d.minInsets, ScaffoldSize: size, SnackBarSize: snackBarSize, MaterialBannerSize: materialBannerSize,
			TextDirection: d.textDirection, MinViewPadding: d.minViewPadding,
		}
		fabOffset := d.floatingActionButtonLocation.GetOffset(currentGeometry)
		d.PositionChild(scaffoldSlotFloatingActionButton, fabOffset)
		floatingActionButtonRect = ui.NewRectFromLTWH(fabOffset.Dx, fabOffset.Dy, fabSize.Width, fabSize.Height)
	}

	if d.HasChild(scaffoldSlotSnackBar) {
		hasCustomWidth := d.snackBarWidth.IsSome() && d.snackBarWidth.Value() < size.Width
		if snackBarSize == (ui.Size{}) {
			c := fullWidthConstraints
			if hasCustomWidth {
				c = looseConstraints
			}
			snackBarSize = d.LayoutChild(scaffoldSlotSnackBar, c)
		}
		var snackBarYOffsetBase float64
		// TODO(step 13.3): the locations at the top of the screen do not show the
		// snack bar above the button; endFloat does.
		showAboveFab := true
		if floatingActionButtonRect.Size() != (ui.Size{}) && d.isSnackBarFloating && showAboveFab {
			if top, ok := bottomNavigationBarTop.Get(); ok {
				snackBarYOffsetBase = math.Min(top, floatingActionButtonRect.Top)
			} else {
				snackBarYOffsetBase = floatingActionButtonRect.Top
			}
		} else {
			// SnackBarBehavior.fixed applies a SafeArea automatically.
			// SnackBarBehavior.floating does not since the positioning is affected
			// if there is a FloatingActionButton (see condition above). If there is
			// no FAB, make sure we account for safe space when the SnackBar is
			// floating.
			safeYOffsetBase := size.Height - d.minViewPadding.Bottom
			if d.isSnackBarFloating {
				snackBarYOffsetBase = math.Min(contentBottom, safeYOffsetBase)
			} else {
				snackBarYOffsetBase = contentBottom
			}
		}
		xOffset := 0.0
		if hasCustomWidth {
			xOffset = (size.Width - d.snackBarWidth.Value()) / 2
		}
		d.PositionChild(scaffoldSlotSnackBar, ui.Offset{Dx: xOffset, Dy: snackBarYOffsetBase - snackBarSize.Height})
		if foundation.DebugMode && d.isSnackBarFloating {
			// Whether a floating SnackBar has been offset too high. To improve the
			// developer experience, this check is done after the call to
			// PositionChild: if we checked sooner the SnackBar is visible because its
			// default position is (0, 0) and it can cause confusion to the user.
			if !((snackBarYOffsetBase - snackBarSize.Height) >= 0) {
				panic(floatingSnackBarOffScreen())
			}
		}
	}

	// TODO(step 13.2): the status bar, drawer and end drawer slots.

	var fabArea foundation.Opt[ui.Rect]
	if d.HasChild(scaffoldSlotFloatingActionButton) {
		fabArea = foundation.Some(floatingActionButtonRect)
	}
	d.geometryNotifier.updateWith(bottomNavigationBarTop, fabArea)
}

func (d *scaffoldLayout) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	o := old.(*scaffoldLayout)
	return o.minInsets != d.minInsets || o.minViewPadding != d.minViewPadding || o.textDirection != d.textDirection ||
		o.floatingActionButtonLocation != d.floatingActionButtonLocation || o.extendBody != d.extendBody ||
		o.extendBodyBehindAppBar != d.extendBodyBehindAppBar
}

// ScaffoldOpts are the optional parameters of NewScaffold.
//
// Flutter: material/scaffold.dart Scaffold.new
type ScaffoldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AppBar is the app bar, displayed at the top of the scaffold.
	AppBar widgets.PreferredSizeWidget
	// Body is the primary content of the scaffold, displayed below the app bar.
	Body widgets.Widget
	// FloatingActionButton is displayed above the body, in the corner that the
	// location says (the end of the bottom by default).
	FloatingActionButton widgets.Widget
	// FloatingActionButtonLocation places the floating action button; the default
	// is FloatingActionButtonLocationEndFloat.
	//
	// TODO(step 13.3): changing it animates the button, with the animator.
	FloatingActionButtonLocation FloatingActionButtonLocation
	// PersistentFooterButtons are the buttons that are displayed at the bottom of
	// the scaffold, below the body.
	PersistentFooterButtons []widgets.Widget
	// PersistentFooterAlignment is the alignment of the persistent footer buttons;
	// the default is AlignmentDirectional.centerEnd.
	PersistentFooterAlignment painting.AlignmentGeometry
	// PersistentFooterDecoration is the decoration of the footer; the default is a
	// divider on the top.
	PersistentFooterDecoration *painting.BoxDecoration
	// BottomNavigationBar is displayed at the bottom of the scaffold.
	BottomNavigationBar widgets.Widget
	// BackgroundColor is the color of the Material of the scaffold; the default
	// is the scaffold background color of the theme.
	BackgroundColor foundation.Opt[ui.Color]
	// ResizeToAvoidBottomInset says whether the body and the floating widgets
	// resize to avoid the on-screen keyboard; the default is true.
	ResizeToAvoidBottomInset foundation.Opt[bool]
	// NoPrimary says that the scaffold is not displayed at the top of the screen,
	// so the app bar does not pad itself with the top padding (Flutter's
	// primary: false).
	NoPrimary bool
	// ExtendBody says that the body extends to the bottom of the scaffold, behind
	// the bottom navigation bar and the persistent footer buttons.
	ExtendBody bool
	// ExtendBodyBehindAppBar says that the body extends to the top of the
	// scaffold, behind the app bar.
	ExtendBodyBehindAppBar bool

	// TODO(step 13.2): Drawer, EndDrawer, OnDrawerChanged, OnEndDrawerChanged,
	// DrawerDragStartBehavior, DrawerBarrierDismissible, DrawerScrimColor,
	// DrawerEdgeDragWidth, the drawer drag flags, BottomSheet and
	// BottomSheetScrimBuilder. TODO: RestorationId.
}

// Scaffold implements the basic Material Design visual layout structure: an app
// bar, a body, a floating action button, a bottom navigation bar and persistent
// footer buttons, plus the snack bars and the banners that a ScaffoldMessenger
// shows.
//
// TODO(step 13.2): the drawers, the bottom sheets (Scaffold.bottomSheet and
// showBottomSheet) and the status bar tap. TODO(step 13.3): the floating action
// button locations, animator and transition.
//
// Flutter: material/scaffold.dart Scaffold
type Scaffold struct {
	widgets.StatefulWidget
	o ScaffoldOpts
}

// NewScaffold creates a scaffold.
//
// Flutter: material/scaffold.dart Scaffold.new
func NewScaffold(opts ...ScaffoldOpts) *Scaffold {
	o := firstOpts(opts)
	if o.FloatingActionButtonLocation == nil {
		o.FloatingActionButtonLocation = FloatingActionButtonLocationEndFloat
	}
	return &Scaffold{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// AppBar is the app bar.
//
// Flutter: material/scaffold.dart Scaffold.appBar
func (w *Scaffold) AppBar() widgets.PreferredSizeWidget { return w.o.AppBar }

// Body is the primary content of the scaffold.
//
// Flutter: material/scaffold.dart Scaffold.body
func (w *Scaffold) Body() widgets.Widget { return w.o.Body }

// FloatingActionButton is the button displayed above the body.
//
// Flutter: material/scaffold.dart Scaffold.floatingActionButton
func (w *Scaffold) FloatingActionButton() widgets.Widget { return w.o.FloatingActionButton }

// BottomNavigationBar is displayed at the bottom of the scaffold.
//
// Flutter: material/scaffold.dart Scaffold.bottomNavigationBar
func (w *Scaffold) BottomNavigationBar() widgets.Widget { return w.o.BottomNavigationBar }

// PersistentFooterButtons are the buttons at the bottom of the scaffold.
//
// Flutter: material/scaffold.dart Scaffold.persistentFooterButtons
func (w *Scaffold) PersistentFooterButtons() []widgets.Widget { return w.o.PersistentFooterButtons }

// ExtendBody is whether the body extends behind the bottom widgets.
//
// Flutter: material/scaffold.dart Scaffold.extendBody
func (w *Scaffold) ExtendBody() bool { return w.o.ExtendBody }

// ExtendBodyBehindAppBar is whether the body extends behind the app bar.
//
// Flutter: material/scaffold.dart Scaffold.extendBodyBehindAppBar
func (w *Scaffold) ExtendBodyBehindAppBar() bool { return w.o.ExtendBodyBehindAppBar }

// Primary is whether the scaffold is displayed at the top of the screen.
//
// Flutter: material/scaffold.dart Scaffold.primary
func (w *Scaffold) Primary() bool { return !w.o.NoPrimary }

// CreateState creates the state.
//
// Flutter: material/scaffold.dart Scaffold.createState
func (w *Scaffold) CreateState() widgets.State { return &ScaffoldState{} }

// ScaffoldOf returns the state of the closest Scaffold above the context. It
// panics with the tag ScaffoldMissing in debug builds when there is none.
//
// Flutter: material/scaffold.dart Scaffold.of
func ScaffoldOf(bc widgets.BuildContext) *ScaffoldState {
	if s, ok := widgets.FindAncestorStateOfType[*ScaffoldState](bc); ok {
		return s
	}
	panic(scaffoldMissingError(bc))
}

// ScaffoldMaybeOf returns the state of the closest Scaffold above the context, or
// nil when there is none.
//
// Flutter: material/scaffold.dart Scaffold.maybeOf
func ScaffoldMaybeOf(bc widgets.BuildContext) *ScaffoldState {
	s, _ := widgets.FindAncestorStateOfType[*ScaffoldState](bc)
	return s
}

// ScaffoldGeometryOf returns a listenable of the geometry of the closest
// Scaffold above the context, which is updated by the layout of the scaffold: a
// BottomAppBar uses it to make the notch of the floating action button. It
// panics with the tag ScaffoldGeometryMissing in debug builds when there is no
// Scaffold.
//
// Flutter: material/scaffold.dart Scaffold.geometryOf
func ScaffoldGeometryOf(bc widgets.BuildContext) foundation.ValueListenable[ScaffoldGeometry] {
	scope, ok := widgets.DependOn[*scaffoldScope](bc)
	if !ok {
		panic(scaffoldGeometryMissingError(bc))
	}
	return scope.geometryNotifier
}

// ScaffoldHasDrawer is whether the Scaffold that most tightly encloses the given
// context has a drawer. When registerForUpdates is true (Flutter's default), the
// context is rebuilt when the answer changes.
//
// Flutter: material/scaffold.dart Scaffold.hasDrawer
func ScaffoldHasDrawer(bc widgets.BuildContext, registerForUpdates bool) bool {
	if registerForUpdates {
		scope, ok := widgets.DependOn[*scaffoldScope](bc)
		return ok && scope.hasDrawer
	}
	s := ScaffoldMaybeOf(bc)
	return s != nil && s.HasDrawer()
}

// ScaffoldState is the state of a Scaffold: it manages the snack bars and the
// banners that the ScaffoldMessenger above shows and the geometry of the
// floating action button.
//
// Flutter: material/scaffold.dart ScaffoldState
type ScaffoldState struct {
	widgets.StateOf[*Scaffold]

	appBarMaxHeight  foundation.Opt[float64]
	geometryNotifier *scaffoldGeometryNotifier
	bodyKey          *widgets.GlobalKey[widgets.State]
	bodyInfo         scaffoldBodyInfo

	scaffoldMessenger       *ScaffoldMessengerState
	messengerSnackBar       *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]
	messengerMaterialBanner *ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason]
}

// HasAppBar is whether the scaffold has an app bar.
//
// Flutter: material/scaffold.dart ScaffoldState.hasAppBar
func (s *ScaffoldState) HasAppBar() bool { return s.Widget().o.AppBar != nil }

// HasDrawer is whether the scaffold has a drawer. It is false until step 13.2.
//
// Flutter: material/scaffold.dart ScaffoldState.hasDrawer
func (s *ScaffoldState) HasDrawer() bool { return false }

// HasEndDrawer is whether the scaffold has an end drawer. It is false until step
// 13.2.
//
// Flutter: material/scaffold.dart ScaffoldState.hasEndDrawer
func (s *ScaffoldState) HasEndDrawer() bool { return false }

// HasFloatingActionButton is whether the scaffold has a floating action button.
//
// Flutter: material/scaffold.dart ScaffoldState.hasFloatingActionButton
func (s *ScaffoldState) HasFloatingActionButton() bool {
	return s.Widget().o.FloatingActionButton != nil
}

// AppBarMaxHeight is the max height of the app bar of the last build, when the
// scaffold has one.
//
// Flutter: material/scaffold.dart ScaffoldState.appBarMaxHeight
func (s *ScaffoldState) AppBarMaxHeight() foundation.Opt[float64] { return s.appBarMaxHeight }

// IsDrawerOpen is whether the drawer is opened. It is false until step 13.2.
//
// Flutter: material/scaffold.dart ScaffoldState.isDrawerOpen
func (s *ScaffoldState) IsDrawerOpen() bool { return false }

// IsEndDrawerOpen is whether the end drawer is opened. It is false until step
// 13.2.
//
// Flutter: material/scaffold.dart ScaffoldState.isEndDrawerOpen
func (s *ScaffoldState) IsEndDrawerOpen() bool { return false }

func (s *ScaffoldState) updateSnackBar() {
	var current *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]
	if len(s.scaffoldMessenger.snackBars) > 0 {
		current = s.scaffoldMessenger.snackBars[0]
	}
	if s.messengerSnackBar != current {
		s.SetState(func() { s.messengerSnackBar = current })
	}
}

func (s *ScaffoldState) updateMaterialBanner() {
	var current *ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason]
	if len(s.scaffoldMessenger.materialBanners) > 0 {
		current = s.scaffoldMessenger.materialBanners[0]
	}
	if s.messengerMaterialBanner != current {
		s.SetState(func() { s.messengerMaterialBanner = current })
	}
}

func (s *ScaffoldState) resizeToAvoidBottomInset() bool {
	return s.Widget().o.ResizeToAvoidBottomInset.OrElse(true)
}

// InitState creates the geometry notifier.
//
// Flutter: material/scaffold.dart ScaffoldState.initState
func (s *ScaffoldState) InitState() {
	s.geometryNotifier = &scaffoldGeometryNotifier{}
	s.bodyKey = widgets.NewGlobalKey[widgets.State]()
	// TODO(step 13.3): the floating action button transition scales the button in
	// and out; until then it is fully shown.
	s.updateGeometryScale()
}

func (s *ScaffoldState) updateGeometryScale() {
	scale := 0.0
	if s.Widget().o.FloatingActionButton != nil {
		scale = 1.0
	}
	s.geometryNotifier.floatingActionButtonScale = scale
}

// DidUpdateWidget keeps the geometry scale of the button.
//
// Flutter: material/scaffold.dart ScaffoldState.didUpdateWidget
func (s *ScaffoldState) DidUpdateWidget(*Scaffold) { s.updateGeometryScale() }

// DidChangeDependencies registers the scaffold with the closest
// ScaffoldMessenger.
//
// Flutter: material/scaffold.dart ScaffoldState.didChangeDependencies
func (s *ScaffoldState) DidChangeDependencies() {
	current := ScaffoldMessengerMaybeOf(s.Context())
	if s.scaffoldMessenger != nil && (current == nil || s.scaffoldMessenger != current) {
		s.scaffoldMessenger.unregister(s)
	}
	s.scaffoldMessenger = current
	if s.scaffoldMessenger != nil {
		s.scaffoldMessenger.register(s)
	}
}

// Dispose unregisters the scaffold.
//
// Flutter: material/scaffold.dart ScaffoldState.dispose
func (s *ScaffoldState) Dispose() {
	s.geometryNotifier.Dispose()
	if s.scaffoldMessenger != nil {
		s.scaffoldMessenger.unregister(s)
	}
}

// addSlot adds the child of a slot to the children, wrapped in the media query
// that has the padding removed that the other slots take.
//
// Flutter: material/scaffold.dart ScaffoldState._addIfNonNull
type slotOpts struct {
	removeLeftPadding, removeTopPadding, removeRightPadding, removeBottomPadding bool
	removeBottomInset                                                            bool
	maintainBottomViewPadding                                                    bool
}

func (s *ScaffoldState) addSlot(children []widgets.Widget, child widgets.Widget, slot scaffoldSlot, o slotOpts) []widgets.Widget {
	if child == nil {
		return children
	}
	data := widgets.MediaQueryOf(s.Context()).RemovePadding(widgets.RemoveSides{
		Left: o.removeLeftPadding, Top: o.removeTopPadding, Right: o.removeRightPadding, Bottom: o.removeBottomPadding})
	if o.removeBottomInset {
		data = data.RemoveViewInsets(widgets.RemoveSides{Bottom: true})
	}
	if o.maintainBottomViewPadding && data.ViewInsets.Bottom != 0.0 {
		data = data.CopyWith(widgets.MediaQueryDataCopyOpts{
			Padding: foundation.Some(data.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(data.ViewPadding.Bottom)}))})
	}
	return append(children, widgets.NewLayoutId(slot, widgets.NewMediaQuery(data, child)))
}

// Build builds the scaffold.
//
// Flutter: material/scaffold.dart ScaffoldState.build
func (s *ScaffoldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	widgets.DebugCheckHasMediaQuery(bc)
	widgets.DebugCheckHasDirectionality(bc)
	theme := ThemeOf(bc)
	textDirection := widgets.DirectionalityOf(bc)
	resize := s.resizeToAvoidBottomInset()
	mq := widgets.MediaQueryOf(bc)

	var children []widgets.Widget
	var body widgets.Widget
	if o.Body != nil {
		body = scaffoldBodyBuilder(o.ExtendBody, o.ExtendBodyBehindAppBar, &s.bodyInfo, widgets.NewKeyedSubtree(o.Body, widgets.WidgetOpts{Key: s.bodyKey}))
	}
	children = s.addSlot(children, body, scaffoldSlotBody, slotOpts{
		removeTopPadding: o.AppBar != nil, removeBottomPadding: o.BottomNavigationBar != nil || o.PersistentFooterButtons != nil,
		removeBottomInset: resize})

	// TODO(step 13.2): the body scrim.

	if o.AppBar != nil {
		topPadding := 0.0
		if !o.NoPrimary {
			topPadding = mq.Padding.Top
		}
		s.appBarMaxHeight = foundation.Some(AppBarPreferredHeightFor(bc, o.AppBar) + topPadding)
		children = s.addSlot(children,
			widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: s.appBarMaxHeight.Value()},
				FlexibleSpaceBarCreateSettings(s.appBarMaxHeight.Value(), o.AppBar)),
			scaffoldSlotAppBar, slotOpts{removeBottomPadding: true})
	}

	// TODO(step 13.2): the bottom sheets.

	isSnackBarFloating := false
	snackBarWidth := foundation.None[float64]()
	if s.messengerSnackBar != nil {
		snackBarTheme := SnackBarThemeOf(bc)
		sb := s.messengerSnackBar.widget
		behavior := sb.o.Behavior
		if behavior == SnackBarBehaviorUnspecified {
			behavior = snackBarTheme.Behavior
		}
		if behavior == SnackBarBehaviorUnspecified {
			behavior = SnackBarBehaviorFixed
		}
		isSnackBarFloating = behavior == SnackBarBehaviorFloating
		snackBarWidth = sb.o.Width
		if snackBarWidth.IsNone() {
			snackBarWidth = snackBarTheme.Width
		}
		children = s.addSlot(children, sb, scaffoldSlotSnackBar, slotOpts{
			removeTopPadding: true, removeBottomPadding: o.BottomNavigationBar != nil || o.PersistentFooterButtons != nil,
			maintainBottomViewPadding: !resize})
	}

	extendBodyBehindMaterialBanner := false
	if s.messengerMaterialBanner != nil {
		bannerTheme := MaterialBannerThemeOf(bc)
		banner := s.messengerMaterialBanner.widget
		elevation := banner.o.Elevation.OrElse(bannerTheme.Elevation.OrElse(0.0))
		extendBodyBehindMaterialBanner = elevation != 0.0
		children = s.addSlot(children, banner, scaffoldSlotMaterialBanner, slotOpts{
			removeTopPadding: o.AppBar != nil, removeBottomPadding: true, maintainBottomViewPadding: !resize})
	}

	if o.PersistentFooterButtons != nil {
		decoration := o.PersistentFooterDecoration
		if decoration == nil {
			decoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.Border{Top: createDividerBorderSide(bc, 1.0)}})
		}
		alignment := o.PersistentFooterAlignment
		if alignment == nil {
			alignment = painting.AlignmentDirectionalCenterEnd
		}
		children = s.addSlot(children,
			widgets.NewContainer(
				widgets.NewSafeArea(
					widgets.NewIntrinsicHeight(
						widgets.NewPadding(painting.NewEdgeInsetsAll(8),
							widgets.NewAlign(widgets.NewOverflowBar(o.PersistentFooterButtons, widgets.OverflowBarOpts{
								Spacing: 8, OverflowAlignment: rendering.OverflowBarAlignmentEnd}), widgets.AlignOpts{Alignment: alignment}))),
					widgets.SafeAreaOpts{Top: foundation.Some(false)}),
				widgets.ContainerOpts{Decoration: decoration}),
			scaffoldSlotPersistentFooter, slotOpts{removeTopPadding: true, removeBottomPadding: o.BottomNavigationBar != nil, maintainBottomViewPadding: !resize})
	}

	if o.BottomNavigationBar != nil {
		children = s.addSlot(children, o.BottomNavigationBar, scaffoldSlotBottomNavigationBar, slotOpts{removeTopPadding: true, maintainBottomViewPadding: !resize})
	}

	// TODO(step 13.3): the floating action button transition wraps the child.
	children = s.addSlot(children, o.FloatingActionButton, scaffoldSlotFloatingActionButton, slotOpts{
		removeLeftPadding: true, removeTopPadding: true, removeRightPadding: true, removeBottomPadding: true})

	// TODO(step 13.2): the status bar tap target (iOS and macOS) and the drawers.

	minInsets := mq.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(0.0)})
	if resize {
		minInsets = mq.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(mq.ViewInsets.Bottom)})
	}
	minViewPadding := mq.ViewPadding
	if resize && mq.ViewInsets.Bottom != 0.0 {
		minViewPadding = mq.ViewPadding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(0.0)})
	}

	return &scaffoldScope{
		InheritedWidget: widgets.NewInheritedWidgetBase(
			widgets.NewScrollNotificationObserver(
				NewMaterial(
					// TODO(step 13.2): an Actions widget with a DismissIntent action that
					// closes the drawers wraps the layout.
					widgets.NewCustomMultiChildLayout(&scaffoldLayout{
						MultiChildLayoutDelegateBase:   rendering.NewMultiChildLayoutDelegateBase(nil),
						extendBody:                     o.ExtendBody,
						extendBodyBehindAppBar:         o.ExtendBodyBehindAppBar,
						minInsets:                      minInsets,
						minViewPadding:                 minViewPadding,
						floatingActionButtonLocation:   o.FloatingActionButtonLocation,
						geometryNotifier:               s.geometryNotifier,
						textDirection:                  textDirection,
						isSnackBarFloating:             isSnackBarFloating,
						extendBodyBehindMaterialBanner: extendBodyBehindMaterialBanner,
						snackBarWidth:                  snackBarWidth,
						bodyInfo:                       &s.bodyInfo,
					}, children),
					MaterialOpts{Color: foundation.Some(o.BackgroundColor.OrElse(theme.ScaffoldBackgroundColor))}))),
		hasDrawer: s.HasDrawer(), geometryNotifier: s.geometryNotifier,
	}
}

// scaffoldScope gives the descendants of a scaffold its geometry and whether it
// has a drawer.
//
// Flutter: material/scaffold.dart _ScaffoldScope
type scaffoldScope struct {
	widgets.InheritedWidget
	hasDrawer        bool
	geometryNotifier *scaffoldGeometryNotifier
}

func (w *scaffoldScope) UpdateShouldNotify(old widgets.Widget) bool {
	return w.hasDrawer != old.(*scaffoldScope).hasDrawer
}

// createDividerBorderSide is Divider.createBorderSide for the Material 3 and
// Material 2 defaults of a Divider.
//
// TODO(step 13.9): Divider.createBorderSide, with the divider theme.
func createDividerBorderSide(bc widgets.BuildContext, width float64) painting.BorderSide {
	theme := ThemeOf(bc)
	color := theme.DividerColor
	if theme.UseMaterial3 {
		color = theme.ColorScheme.OutlineVariant()
	}
	return painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(color), Width: foundation.Some(width)})
}

func scaffoldMissingError(bc widgets.BuildContext) error {
	return errors.New("ScaffoldMissing", "Scaffold.of() called with a context that does not contain a Scaffold.",
		"widget", foundation.RuntimeTypeName(bc.Widget()))
}

func scaffoldGeometryMissingError(bc widgets.BuildContext) error {
	return errors.New("ScaffoldGeometryMissing", "Scaffold.geometryOf() called with a context that does not contain a Scaffold.",
		"widget", foundation.RuntimeTypeName(bc.Widget()))
}

func floatingSnackBarOffScreen() error {
	return errors.New("FloatingSnackBarOffScreen", "Floating SnackBar presented off screen. A SnackBar with behavior: SnackBarBehavior.floating is positioned above the floating action button or the bottom of the screen and does not fit.")
}
