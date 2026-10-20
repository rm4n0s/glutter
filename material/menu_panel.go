// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// menuStyleOpt is the first value that the styles give for a property whose
// states resolve to an Opt, in the order of the styles; the properties are
// resolved with no states.
func menuStyleOpt[T any](styles []*MenuStyle, get func(*MenuStyle) widgets.WidgetStateProperty[foundation.Opt[T]]) foundation.Opt[T] {
	states := widgets.NewWidgetStates()
	for _, st := range styles {
		if st == nil {
			continue
		}
		if p := get(st); p != nil {
			if v := p.Resolve(states); v.IsSome() {
				return v
			}
		}
	}
	return foundation.None[T]()
}

// menuStyleNil is the first non-nil value that the styles give for a property
// whose states resolve to an interface or a pointer.
func menuStyleNil[T any](styles []*MenuStyle, get func(*MenuStyle) widgets.WidgetStateProperty[T]) (T, bool) {
	states := widgets.NewWidgetStates()
	for _, st := range styles {
		if st == nil {
			continue
		}
		if p := get(st); p != nil {
			if v := p.Resolve(states); any(v) != nil {
				return v, true
			}
		}
	}
	var zero T
	return zero, false
}

// menuPanelOpts are the parameters of newMenuPanel.
type menuPanelOpts struct {
	menuStyle              *MenuStyle
	clipBehavior           ui.Clip
	crossAxisUnconstrained bool
	orientation            painting.Axis
	heightAnimation        animation.Animation[float64]
	children               []widgets.Widget
}

// menuPanel is the panel of a menu: a Material with the items in a scrollable
// that lays them out along the orientation of the menu.
//
// Flutter: material/menu_anchor.dart _MenuPanel
type menuPanel struct {
	widgets.StatefulWidget
	o menuPanelOpts
}

func newMenuPanel(o menuPanelOpts) *menuPanel {
	return &menuPanel{StatefulWidget: widgets.NewStatefulWidgetBase(), o: o}
}

func (w *menuPanel) CreateState() widgets.State { return &menuPanelState{} }

type menuPanelState struct {
	widgets.StateOf[*menuPanel]
	scrollController *widgets.ScrollControllerBase
}

func (s *menuPanelState) InitState() { s.scrollController = widgets.NewScrollController() }

func (s *menuPanelState) Dispose() { s.scrollController.Dispose() }

// Build builds the panel with the style of the widget, then of the theme, then
// of the defaults.
//
// Flutter: material/menu_anchor.dart _MenuPanelState.build
func (s *menuPanelState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var themeStyle, defaultStyle *MenuStyle
	if w.o.orientation == painting.AxisHorizontal {
		themeStyle, defaultStyle = MenuBarThemeOf(bc).Style, menuBarDefaultsM3(bc)
	} else {
		themeStyle, defaultStyle = MenuThemeOf(bc).Style, menuDefaultsM3(bc)
	}
	styles := []*MenuStyle{w.o.menuStyle, themeStyle, defaultStyle}

	backgroundColor := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.BackgroundColor })
	shadowColor := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.ShadowColor })
	surfaceTintColor := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] { return st.SurfaceTintColor })
	elevation := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[float64]] { return st.Elevation }).OrElse(0)
	minimumSize := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.MinimumSize })
	fixedSize := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.FixedSize })
	maximumSize := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] { return st.MaximumSize })
	side := menuStyleOpt(styles, func(st *MenuStyle) widgets.WidgetStateProperty[foundation.Opt[painting.BorderSide]] { return st.Side })
	shape, _ := menuStyleNil(styles, func(st *MenuStyle) widgets.WidgetStateProperty[painting.OutlinedBorder] { return st.Shape })
	if sd, ok := side.Get(); ok {
		shape = shape.CopyWithSide(sd)
	}
	visualDensity := VisualDensityStandard()
	for _, st := range styles {
		if st != nil && st.VisualDensity.IsSome() {
			visualDensity = st.VisualDensity.OrElse(visualDensity)
			break
		}
	}
	padding, ok := menuStyleNil(styles, func(st *MenuStyle) widgets.WidgetStateProperty[painting.EdgeInsetsGeometry] { return st.Padding })
	if !ok {
		padding = painting.EdgeInsetsZero
	}
	densityAdjustment := visualDensity.BaseSizeAdjustment()

	// Per the Material Design team: don't allow the VisualDensity adjustment to
	// reduce the width of the left/right padding. If it did,
	// VisualDensity.compact, the default for desktop/web, would reduce the
	// horizontal padding to zero. Vertical padding is not affected at all.
	dx := math.Max(0, densityAdjustment.Dx)
	resolvedPadding := padding.Add(painting.NewEdgeInsetsSymmetric(0, dx)).Clamp(painting.EdgeInsetsZero, painting.EdgeInsetsGeometryInfinity)

	min := minimumSize.OrElse(ui.Size{})
	max := maximumSize.OrElse(ui.Size{Width: math.Inf(1), Height: math.Inf(1)})
	effectiveConstraints := visualDensity.EffectiveConstraints(rendering.BoxConstraints{
		MinWidth: min.Width, MinHeight: min.Height, MaxWidth: max.Width, MaxHeight: max.Height})
	if fixed, ok := fixedSize.Get(); ok {
		size := effectiveConstraints.Constrain(fixed)
		if !math.IsInf(size.Width, 0) && !math.IsNaN(size.Width) {
			effectiveConstraints = effectiveConstraints.CopyWith(rendering.BoxConstraintsCopy{MinWidth: foundation.Some(size.Width), MaxWidth: foundation.Some(size.Width)})
		}
		if !math.IsInf(size.Height, 0) && !math.IsNaN(size.Height) {
			effectiveConstraints = effectiveConstraints.CopyWith(rendering.BoxConstraintsCopy{MinHeight: foundation.Some(size.Height), MaxHeight: foundation.Some(size.Height)})
		}
	}

	// If the menu panel is horizontal, the children are wrapped in an
	// IntrinsicWidth widget to ensure that the children are as wide as the
	// widest child.
	children := w.o.children
	if w.o.orientation == painting.AxisHorizontal {
		children = make([]widgets.Widget, 0, len(w.o.children))
		for _, c := range w.o.children {
			children = append(children, widgets.NewIntrinsicWidth(c))
		}
	}

	status, _ := maybeMenuAnimationStatusOf(bc)
	displayScrollbar := status == animation.AnimationStatusCompleted

	var panel widgets.Widget = widgets.NewPadding(resolvedPadding,
		widgets.NewScrollConfiguration(
			widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{
				Scrollbars: foundation.Some(false), Overscroll: foundation.Some(false), Physics: widgets.NewClampingScrollPhysics(nil)}),
			widgets.NewPrimaryScrollController(s.scrollController,
				tempScrollbar(bc, displayScrollbar,
					widgets.NewSingleChildScrollView(
						widgets.NewFlex(w.o.orientation, children, widgets.FlexOpts{
							CrossAxisAlignment: rendering.CrossAxisAlignmentStart,
							TextDirection:      widgets.DirectionalityOf(bc),
							MainAxisSize:       rendering.MainAxisSizeMin,
						}),
						widgets.SingleChildScrollViewOpts{Controller: s.scrollController, ScrollDirection: w.o.orientation})))))
	if w.o.heightAnimation != nil {
		// An AnimatedBuilder is used instead of SizeTransition because Material
		// already introduces its own ClipRRect for the shape.
		panel = widgets.NewAnimatedBuilder(w.o.heightAnimation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
			return widgets.NewAlign(child, widgets.AlignOpts{
				Alignment:    painting.AlignmentDirectionalTopStart,
				HeightFactor: foundation.Some(w.o.heightAnimation.Value()),
				WidthFactor:  foundation.Some(1.0),
			})
		}, panel)
	}
	matType := MaterialTypeCanvas
	if !backgroundColor.IsSome() {
		matType = MaterialTypeTransparency
	}
	panel = s.intrinsicCrossSize(NewMaterial(panel, MaterialOpts{
		Elevation:        elevation,
		Shape:            shape,
		Color:            backgroundColor,
		ShadowColor:      shadowColor,
		SurfaceTintColor: surfaceTintColor,
		Type:             matType,
		ClipBehavior:     w.o.clipBehavior,
	}))
	if w.o.crossAxisUnconstrained {
		panel = widgets.NewUnconstrainedBox(panel, widgets.UnconstrainedBoxOpts{
			ConstrainedAxis: w.o.orientation,
			ClipBehavior:    ui.ClipHardEdge,
			Alignment:       painting.AlignmentDirectionalCenterStart,
		})
	}
	return widgets.NewConstrainedBox(effectiveConstraints, panel)
}

func (s *menuPanelState) intrinsicCrossSize(child widgets.Widget) widgets.Widget {
	if s.Widget().o.orientation == painting.AxisHorizontal {
		return widgets.NewIntrinsicHeight(child)
	}
	return widgets.NewIntrinsicWidth(child)
}

// submenu is the widget that defines the menu drawn in the overlay: the panel in
// a layout that positions it next to the anchor.
//
// Flutter: material/menu_anchor.dart _Submenu
type submenu struct {
	widgets.StatelessWidget
	anchor                 *menuAnchorState
	layerLink              *rendering.LayerLink
	menuStyle              *MenuStyle
	menuPosition           widgets.RawMenuOverlayInfo
	alignmentOffset        ui.Offset
	consumeOutsideTaps     bool
	clipBehavior           ui.Clip
	crossAxisUnconstrained bool
	menuChildren           []widgets.Widget
	menuScopeNode          *widgets.FocusScopeNode
	fadeAnimation          animation.Animation[float64]
	heightAnimation        animation.Animation[float64]
	reservedPadding        painting.EdgeInsetsGeometry
}

// Build builds the menu panel and the layout that positions it.
//
// Flutter: material/menu_anchor.dart _Submenu.build
func (w *submenu) Build(bc widgets.BuildContext) widgets.Widget {
	// Use the text direction of the context where the button is.
	textDirection := widgets.DirectionalityOf(bc)
	var themeStyle, defaultStyle *MenuStyle
	if p := w.anchor.parent(); p == nil || p.orientation() == painting.AxisHorizontal {
		themeStyle, defaultStyle = MenuBarThemeOf(bc).Style, menuBarDefaultsM3(bc)
	} else {
		themeStyle, defaultStyle = MenuThemeOf(bc).Style, menuDefaultsM3(bc)
	}
	styles := []*MenuStyle{w.menuStyle, themeStyle, defaultStyle}

	mouseCursor := services.MouseCursor(services.MouseCursorUncontrolled)
	if c, ok := menuStyleNil(styles, func(st *MenuStyle) widgets.WidgetStateProperty[services.MouseCursor] { return st.MouseCursor }); ok {
		mouseCursor = c
	}
	visualDensity := ThemeOf(bc).VisualDensity
	for _, st := range styles {
		if st != nil && st.VisualDensity.IsSome() {
			visualDensity = st.VisualDensity.OrElse(visualDensity)
			break
		}
	}
	var alignment painting.AlignmentGeometry
	for _, st := range styles {
		if st != nil && st.Alignment != nil {
			alignment = st.Alignment
			break
		}
	}
	padding, ok := menuStyleNil(styles, func(st *MenuStyle) widgets.WidgetStateProperty[painting.EdgeInsetsGeometry] { return st.Padding })
	if !ok {
		padding = painting.EdgeInsetsZero
	}
	densityAdjustment := visualDensity.BaseSizeAdjustment()
	// Per the Material Design team: don't allow the VisualDensity adjustment to
	// reduce the width of the left/right padding. If it did,
	// VisualDensity.compact, the default for desktop/web, would reduce the
	// horizontal padding to zero.
	dx := math.Max(0, densityAdjustment.Dx)
	resolvedPadding := padding.Add(painting.NewEdgeInsetsFromLTRB(dx, 0, dx, 0)).Clamp(painting.EdgeInsetsZero, painting.EdgeInsetsGeometryInfinity)

	anchorRect := ui.Rect{}
	if w.layerLink == nil {
		r := w.menuPosition.AnchorRect
		anchorRect = ui.Rect{Left: r.Left + dx, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
	}

	anchor := w.anchor
	panel := widgets.Widget(widgets.NewTapRegion(
		widgets.NewMouseRegion(
			widgets.NewFocusScope(
				widgets.NewActions(dismissMenuActions(anchor.menuController()),
					widgets.NewShortcuts(menuTraversalShortcuts(),
						widgets.NewFadeTransition(w.fadeAnimation,
							newMenuPanel(menuPanelOpts{
								menuStyle:              w.menuStyle,
								clipBehavior:           w.clipBehavior,
								orientation:            anchor.orientation(),
								crossAxisUnconstrained: w.crossAxisUnconstrained,
								heightAnimation:        w.heightAnimation,
								children:               w.menuChildren,
							}), widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true}))),
				widgets.FocusScopeOpts{Node: anchor.menuScopeNode, SkipTraversal: foundation.Some(true)}),
			widgets.MouseRegionOpts{Cursor: mouseCursor, HitTestBehavior: rendering.HitTestBehaviorDeferToChild}),
		widgets.TapRegionOpts{
			GroupID:            w.menuPosition.TapRegionGroupID,
			ConsumeOutsideTaps: anchor.root().menuController().IsOpen() && anchor.Widget().o.ConsumeOutsideTap,
			OnTapOutside:       func(*gestures.PointerDownEvent) { anchor.menuController().Close() },
		}))

	themeData := ThemeOf(bc).CopyWith(ThemeDataCopy{VisualDensity: foundation.Some(visualDensity)})
	parentOrientation := painting.AxisHorizontal
	if p := anchor.parent(); p != nil {
		parentOrientation = p.orientation()
	}
	var layout widgets.Widget = NewTheme(themeData,
		widgets.NewConstrainedBox(rendering.NewBoxConstraintsLoose(w.menuPosition.OverlaySize),
			widgets.NewAnimatedBuilder(w.heightAnimation, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
				mediaQuery := widgets.MediaQueryOf(bc)
				return widgets.NewCustomSingleChildLayout(&menuLayout{
					SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil),
					anchorRect:                    anchorRect,
					textDirection:                 textDirection,
					menuPadding:                   resolvedPadding,
					alignment:                     alignment,
					alignmentOffset:               w.alignmentOffset,
					menuPosition:                  w.menuPosition.Position,
					orientation:                   anchor.orientation(),
					parentOrientation:             parentOrientation,
					reservedPadding:               w.reservedPadding,
					heightFactor:                  w.heightAnimation.Value(),
					mediaQueryData:                mediaQuery,
				}, child)
			}, panel)))
	if w.layerLink == nil {
		return layout
	}
	return widgets.NewCompositedTransformFollower(w.layerLink, layout, widgets.CompositedTransformFollowerOpts{TargetAnchor: foundation.Some(painting.AlignmentBottomLeft)})
}

// menuLayout positions the menu in the view while trying to keep as much as
// possible visible in the view.
//
// Deviation: Flutter avoids the display features (the hinge of a foldable) with
// DisplayFeatureSubScreen; there are none on the desktop, so the whole overlay
// is the allowed area.
//
// Flutter: material/menu_anchor.dart _MenuLayout
type menuLayout struct {
	rendering.SingleChildLayoutDelegateBase
	// anchorRect is the rectangle of the underlying button, relative to the
	// overlay's dimensions.
	anchorRect ui.Rect
	// textDirection is whether to prefer going to the left or to the right.
	textDirection ui.TextDirection
	// alignment is the alignment to use when finding the ideal location for the
	// menu.
	alignment painting.AlignmentGeometry
	// alignmentOffset is the offset from the alignment position to find the ideal
	// location for the menu.
	alignmentOffset ui.Offset
	// menuPosition is the position passed to the open method, if any.
	menuPosition foundation.Opt[ui.Offset]
	// menuPadding is the padding on the inside of the menu, so it can be accounted
	// for when positioning.
	menuPadding painting.EdgeInsetsGeometry
	// orientation is the orientation of this menu.
	orientation painting.Axis
	// parentOrientation is the orientation of the parent of this menu.
	parentOrientation painting.Axis
	// reservedPadding is how close to the edge of the safe area the menu will be
	// placed.
	reservedPadding painting.EdgeInsetsGeometry
	// heightFactor is the factor by which the height of the menu is scaled.
	heightFactor float64
	// mediaQueryData is used to ensure the menu is positioned within the safe area
	// and respects view insets such as the software keyboard.
	mediaQueryData widgets.MediaQueryData
}

func (d *menuLayout) GetSize(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Biggest()
}

func (d *menuLayout) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	// The menu can be at most the size of the overlay minus the view padding in
	// each direction.
	return rendering.NewBoxConstraintsLoose(constraints.Biggest()).Deflate(d.reservedPadding)
}

func (d *menuLayout) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	// size: the size of the overlay. childSize: the size of the menu, when fully
	// open, as determined by GetConstraintsForChild.
	overlayRect := d.mediaQueryData.Padding.DeflateRect(d.mediaQueryData.ViewInsets.DeflateRect(ui.Rect{Right: size.Width, Bottom: size.Height}))
	unconstrainedHeight := 0.0
	if d.heightFactor > 0.01 {
		unconstrainedHeight = childSize.Height / d.heightFactor
	}
	childHeightEstimate := math.Min(unconstrainedHeight, size.Height)
	childSizeEstimate := ui.Size{Width: childSize.Width, Height: childHeightEstimate}
	finalPosition := d.positionChild(childSizeEstimate, overlayRect)
	if d.menuPosition.IsSome() {
		return finalPosition
	}
	// If the menu sits above the anchor when fully open, grow upward. Keep the
	// bottom (attachment) fixed by shifting the top-left during animation.
	growsUp := finalPosition.Dy+childSizeEstimate.Height <= d.anchorRect.Center().Dy
	if growsUp {
		dy := childHeightEstimate - childSize.Height
		return ui.Offset{Dx: finalPosition.Dx, Dy: finalPosition.Dy + dy}
	}
	initialPosition := ui.Offset{Dx: finalPosition.Dx, Dy: d.anchorRect.Bottom}
	return ui.Offset{
		Dx: initialPosition.Dx + (finalPosition.Dx-initialPosition.Dx)*d.heightFactor,
		Dy: initialPosition.Dy + (finalPosition.Dy-initialPosition.Dy)*d.heightFactor,
	}
}

func (d *menuLayout) positionChild(childSize ui.Size, overlayRect ui.Rect) ui.Offset {
	var x, y float64
	if !d.menuPosition.IsSome() {
		desiredPosition := d.alignment.Resolve(d.textDirection).WithinRect(d.anchorRect)
		directionalOffset := d.alignmentOffset
		if _, ok := d.alignment.(painting.AlignmentDirectional); ok && d.textDirection == ui.TextDirectionRTL {
			directionalOffset = ui.Offset{Dx: -d.alignmentOffset.Dx, Dy: d.alignmentOffset.Dy}
		}
		desiredPosition = desiredPosition.Add(directionalOffset)
		x, y = desiredPosition.Dx, desiredPosition.Dy
		if d.textDirection == ui.TextDirectionRTL {
			x -= childSize.Width
		}
	} else {
		adjusted := d.menuPosition.OrElse(ui.Offset{}).Add(d.anchorRect.TopLeft())
		x, y = adjusted.Dx, adjusted.Dy
	}
	allowedRect := overlayRect
	offLeftSide := func(x float64) bool { return x < allowedRect.Left }
	offRightSide := func(x float64) bool { return x+childSize.Width > allowedRect.Right }
	offTop := func(y float64) bool { return y < allowedRect.Top }
	offBottom := func(y float64) bool { return y+childSize.Height > allowedRect.Bottom }

	// Avoid going outside an area defined as the rectangle offset from the edge
	// of the screen by the button padding. If the menu is off of the screen, move
	// the menu to the other side of the button first, and then if it doesn't fit
	// there, then just move it over as much as needed to make it fit.
	if childSize.Width >= allowedRect.Width() {
		// It just doesn't fit, so put as much on the screen as possible.
		x = allowedRect.Left
	} else {
		if offLeftSide(x) {
			// If the parent is a different orientation than the current one, then
			// just push it over instead of trying the other side.
			if d.parentOrientation != d.orientation {
				x = allowedRect.Left
			} else {
				newX := d.anchorRect.Right + d.alignmentOffset.Dx
				if !offRightSide(newX) {
					x = newX
				} else {
					x = allowedRect.Left
				}
			}
		} else if offRightSide(x) {
			if d.parentOrientation != d.orientation {
				x = allowedRect.Right - childSize.Width
			} else {
				newX := d.anchorRect.Left - childSize.Width - d.alignmentOffset.Dx
				if !offLeftSide(newX) {
					x = newX
				} else {
					x = allowedRect.Right - childSize.Width
				}
			}
		}
	}
	if childSize.Height >= allowedRect.Height() {
		// Too tall to fit, fit as much on as possible.
		y = allowedRect.Top
	} else {
		if offTop(y) {
			newY := d.anchorRect.Bottom
			if !offBottom(newY) {
				y = newY
			} else {
				y = allowedRect.Top
			}
		} else if offBottom(y) {
			newY := d.anchorRect.Top - childSize.Height
			if !offTop(newY) {
				// Only move the menu up if its parent is horizontal
				// (MenuAnchor/MenuBar).
				if d.parentOrientation == painting.AxisHorizontal {
					y = newY - d.alignmentOffset.Dy
				} else {
					y = newY
				}
			} else {
				y = allowedRect.Bottom - childSize.Height
			}
		}
	}
	return ui.Offset{Dx: x, Dy: y}
}

func (d *menuLayout) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o := old.(*menuLayout)
	return d.anchorRect != o.anchorRect ||
		d.textDirection != o.textDirection ||
		!foundation.ValuesEqual(d.alignment, o.alignment) ||
		d.alignmentOffset != o.alignmentOffset ||
		d.menuPosition != o.menuPosition ||
		!foundation.ValuesEqual(d.menuPadding, o.menuPadding) ||
		d.orientation != o.orientation ||
		d.parentOrientation != o.parentOrientation ||
		!foundation.ValuesEqual(d.reservedPadding, o.reservedPadding) ||
		d.heightFactor != o.heightFactor
}

// dismissMenuActions is the actions that close the menus of the controller on
// the DismissIntent.
func dismissMenuActions(controller *widgets.MenuController) widgets.ActionMap {
	a := widgets.NewDismissMenuAction(controller)
	return widgets.ActionMap{a.IntentType(): a}
}
