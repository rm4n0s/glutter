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
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The sizes and the timing of the dropdown buttons.
const (
	dropdownMenuDuration   = 300 * time.Millisecond
	dropdownMenuItemHeight = MinInteractiveDimension
	dropdownDenseHeight    = 24.0
)

// dropdownMenuItemPadding is the padding of the items of the menu.
var dropdownMenuItemPadding = painting.NewEdgeInsetsSymmetric(0, 16.0)

// DropdownButtonBuilder is the type of the function that builds the widgets that
// are shown in the button for each of the items, when the button is closed.
//
// Flutter: material/dropdown.dart DropdownButtonBuilder
type DropdownButtonBuilder func(bc widgets.BuildContext) []widgets.Widget

// dropdownMenuPainter paints the menu: the background, which grows from the
// selected item as the route animates.
//
// Flutter: material/dropdown.dart _DropdownMenuPainter
type dropdownMenuPainter struct {
	rendering.CustomPainterBase
	color                 foundation.Opt[ui.Color]
	elevation             int
	selectedIndex         int
	borderRadius          foundation.Opt[painting.BorderRadius]
	resize                animation.Animation[float64]
	getSelectedItemOffset func() float64
	painter               painting.BoxPainter
}

func newDropdownMenuPainter(color foundation.Opt[ui.Color], elevation, selectedIndex int, borderRadius foundation.Opt[painting.BorderRadius],
	resize animation.Animation[float64], getSelectedItemOffset func() float64) *dropdownMenuPainter {
	var radius painting.BorderRadiusGeometry = painting.NewBorderRadiusAll(ui.NewRadiusCircular(2.0))
	if br, ok := borderRadius.Get(); ok {
		radius = br
	}
	decoration := painting.NewBoxDecoration(painting.BoxDecorationOpts{
		// If you add an image here, you must provide a real configuration in the
		// Paint function and you must provide some sort of onChanged callback.
		Color: color, BorderRadius: radius, BoxShadow: ElevationToShadow[elevation],
	})
	return &dropdownMenuPainter{
		CustomPainterBase: rendering.NewCustomPainterBase(resize), color: color, elevation: elevation, selectedIndex: selectedIndex,
		borderRadius: borderRadius, resize: resize, getSelectedItemOffset: getSelectedItemOffset,
		painter: decoration.CreateBoxPainter(nil),
	}
}

func (p *dropdownMenuPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	selectedItemOffset := p.getSelectedItemOffset()
	topBegin := ui.ClampDouble(selectedItemOffset, 0.0, math.Max(size.Height-dropdownMenuItemHeight, 0.0))
	bottomBegin := ui.ClampDouble(topBegin+dropdownMenuItemHeight, math.Min(dropdownMenuItemHeight, size.Height), size.Height)
	t := p.resize.Value()
	rect := ui.Rect{Left: 0, Top: topBegin * (1 - t), Right: size.Width, Bottom: bottomBegin + (size.Height-bottomBegin)*t}
	p.painter.Paint(canvas, rect.TopLeft(), painting.ImageConfiguration{Size: foundation.Some(rect.Size())})
}

func (p *dropdownMenuPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o := old.(*dropdownMenuPainter)
	return o.color != p.color || o.elevation != p.elevation || o.selectedIndex != p.selectedIndex ||
		o.borderRadius != p.borderRadius || o.resize != p.resize
}

// dropdownRouteResult boxes the result of the route so that it can be none:
// otherwise, canceling the route (which returns nil) would get confused with
// actually returning a real none value.
//
// Flutter: material/dropdown.dart _DropdownRouteResult
type dropdownRouteResult[T comparable] struct{ result foundation.Opt[T] }

// dropdownMenuLimits are the vertical extent of the menu and the initial scroll
// offset of its list.
//
// Flutter: material/dropdown.dart _MenuLimits
type dropdownMenuLimits struct{ top, bottom, height, scrollOffset float64 }

// dropdownMenuItemLayout lets the route look up the size of each menu item.
// These sizes are used to compute the offset of the selected item so that the
// route page can align the vertical center of the selected item with the
// vertical center of the dropdown button, as closely as possible.
//
// Flutter: material/dropdown.dart _MenuItem
type dropdownMenuItemLayout struct {
	widgets.SingleChildRenderObjectWidget
	onLayout func(size ui.Size)
}

func newDropdownMenuItemLayout(onLayout func(size ui.Size), item widgets.Widget) *dropdownMenuItemLayout {
	return &dropdownMenuItemLayout{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(item), onLayout: onLayout}
}

func (w *dropdownMenuItemLayout) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderDropdownMenuItem{onLayout: w.onLayout}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *dropdownMenuItemLayout) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderDropdownMenuItem).onLayout = w.onLayout
}

// renderDropdownMenuItem is the render object of dropdownMenuItemLayout.
//
// Flutter: material/dropdown.dart _RenderMenuItem
type renderDropdownMenuItem struct {
	rendering.RenderProxyBox
	onLayout func(size ui.Size)
}

func (r *renderDropdownMenuItem) PerformLayout() {
	r.RenderProxyBox.PerformLayout()
	r.onLayout(r.Size())
}

// dropdownMenuItemContainer is the container of a menu item created by a
// DropdownButton: it provides the default configuration for DropdownMenuItems,
// as well as for the hint and the disabled hint of a DropdownButton.
//
// Flutter: material/dropdown.dart _DropdownMenuItemContainer
func dropdownMenuItemContainer(alignment painting.AlignmentGeometry, child widgets.Widget) widgets.Widget {
	if alignment == nil {
		alignment = painting.AlignmentDirectionalCenterStart
	}
	return widgets.NewSemantics(
		widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(dropdownMenuItemHeight)}),
			widgets.NewAlign(child, widgets.AlignOpts{Alignment: alignment})),
		widgets.SemanticsOpts{Button: foundation.Some(true)})
}

// DropdownMenuItemOpts are the optional parameters of NewDropdownMenuItem.
//
// Flutter: material/dropdown.dart DropdownMenuItem.new
type DropdownMenuItemOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTap is called when the item is tapped, before the menu closes.
	OnTap func()
	// Value is the value that the dropdown button gives to its OnChanged when
	// this item is selected.
	Value foundation.Opt[T]
	// Disabled makes the item unable to be selected (Flutter's enabled: false).
	Disabled bool
	// Alignment is how the child is aligned in the item; the default is the center
	// of the start.
	Alignment painting.AlignmentGeometry
}

// DropdownMenuItem is an item in the menu created by a DropdownButton. The
// type T is the type of the value that the entry represents.
//
// Flutter: material/dropdown.dart DropdownMenuItem
type DropdownMenuItem[T comparable] struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     DropdownMenuItemOpts[T]
}

// NewDropdownMenuItem creates an item with the child as its label.
//
// Flutter: material/dropdown.dart DropdownMenuItem.new
func NewDropdownMenuItem[T comparable](child widgets.Widget, opts ...DropdownMenuItemOpts[T]) *DropdownMenuItem[T] {
	o := firstOpts(opts)
	return &DropdownMenuItem[T]{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Value is the value of the item.
//
// Flutter: material/dropdown.dart DropdownMenuItem.value
func (w *DropdownMenuItem[T]) Value() foundation.Opt[T] { return w.o.Value }

// Enabled is whether the item can be selected.
//
// Flutter: material/dropdown.dart DropdownMenuItem.enabled
func (w *DropdownMenuItem[T]) Enabled() bool { return !w.o.Disabled }

// Child is the label of the item.
//
// Flutter: material/dropdown.dart DropdownMenuItem.child
func (w *DropdownMenuItem[T]) Child() widgets.Widget { return w.child }

// Build builds the container of the item.
func (w *DropdownMenuItem[T]) Build(widgets.BuildContext) widgets.Widget {
	return dropdownMenuItemContainer(w.o.Alignment, w.child)
}

// dropdownHideUnderline is the widget that hides the underline of the
// DropdownButtons below it.
type dropdownHideUnderline struct {
	widgets.InheritedWidget
}

func (w *dropdownHideUnderline) UpdateShouldNotify(widgets.Widget) bool { return false }

// NewDropdownButtonHideUnderline wraps a widget that hides the underline of the
// DropdownButtons below it. It is a convenience for the case where a dropdown
// button is the child of a widget that already draws one, such as an
// InputDecorator.
//
// Flutter: material/dropdown.dart DropdownButtonHideUnderline
func NewDropdownButtonHideUnderline(child widgets.Widget, opts ...widgets.WidgetOpts) widgets.Widget {
	return &dropdownHideUnderline{InheritedWidget: widgets.NewInheritedWidgetBase(child, opts...)}
}

// DropdownButtonHideUnderlineAt is whether the context is below a
// NewDropdownButtonHideUnderline widget.
//
// Flutter: material/dropdown.dart DropdownButtonHideUnderline.at
func DropdownButtonHideUnderlineAt(bc widgets.BuildContext) bool {
	_, ok := widgets.DependOn[*dropdownHideUnderline](bc)
	return ok
}

// dropdownItem is what the route needs of an item: its widget, its value and
// whether it is enabled, whatever the type of the value.
type dropdownItem[T comparable] struct {
	item   *DropdownMenuItem[T]
	layout widgets.Widget
}

// dropdownRoute is the route of the menu of a DropdownButton.
//
// Flutter: material/dropdown.dart _DropdownRoute
type dropdownRoute[T comparable] struct {
	widgets.PopupRouteBase
	items                       []dropdownItem[T]
	padding                     painting.EdgeInsetsGeometry
	buttonRect                  ui.Rect
	selectedIndex               int
	elevation                   int
	capturedThemes              *widgets.CapturedThemes
	style                       *painting.TextStyle
	barrierLabel                string
	itemHeight                  foundation.Opt[float64]
	menuWidth                   foundation.Opt[float64]
	dropdownColor               foundation.Opt[ui.Color]
	menuMaxHeight               foundation.Opt[float64]
	enableFeedback              bool
	borderRadius                foundation.Opt[painting.BorderRadius]
	barrierDismissible          bool
	dropdownMenuItemMouseCursor services.MouseCursor
	itemHeights                 []float64
}

// TransitionDuration is 300 ms.
func (r *dropdownRoute[T]) TransitionDuration() time.Duration { return dropdownMenuDuration }

// ReverseTransitionDuration is 300 ms.
func (r *dropdownRoute[T]) ReverseTransitionDuration() time.Duration { return dropdownMenuDuration }

// BarrierDismissible is whether tapping outside closes the menu.
func (r *dropdownRoute[T]) BarrierDismissible() bool { return r.barrierDismissible }

// BarrierColor is none: the barrier is transparent.
func (r *dropdownRoute[T]) BarrierColor() foundation.Opt[ui.Color] {
	return foundation.None[ui.Color]()
}

// BarrierLabel is the label of the barrier.
func (r *dropdownRoute[T]) BarrierLabel() string { return r.barrierLabel }

// BuildPage builds the page of the menu.
func (r *dropdownRoute[T]) BuildPage(bc widgets.BuildContext, _, _ animation.Animation[float64]) widgets.Widget {
	return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		return &dropdownRoutePage[T]{StatefulWidget: widgets.NewStatefulWidgetBase(), route: r, constraints: constraints}
	})
}

func (r *dropdownRoute[T]) dismiss() {
	if r.IsActive() {
		r.Navigator().RemoveRoute(r)
	}
}

func (r *dropdownRoute[T]) getItemOffset(index int) float64 {
	offset := MaterialListPadding.Top
	if len(r.items) > 0 && index > 0 {
		for _, h := range r.itemHeights[:index] {
			offset += h
		}
	}
	return offset
}

// getMenuLimits is the vertical extent of the menu and the initial scroll offset
// of the list that contains the menu items. The vertical center of the selected
// item is aligned with the vertical center of the button, as far as that's
// possible given the available height.
func (r *dropdownRoute[T]) getMenuLimits(buttonRect ui.Rect, availableHeight float64, index int) dropdownMenuLimits {
	computedMaxHeight := availableHeight - 2.0*dropdownMenuItemHeight
	if h, ok := r.menuMaxHeight.Get(); ok {
		computedMaxHeight = math.Min(computedMaxHeight, h)
	}
	buttonTop := buttonRect.Top
	buttonBottom := math.Min(buttonRect.Bottom, availableHeight)
	selectedItemOffset := r.getItemOffset(index)

	// If the button is placed on the bottom or top of the screen, its top or
	// bottom may be less than the item height from the edge of the screen. In
	// this case, the menu limits are changed to align with the top or bottom edge
	// of the button.
	topLimit := math.Min(dropdownMenuItemHeight, buttonTop)
	bottomLimit := math.Max(availableHeight-dropdownMenuItemHeight, buttonBottom)

	menuTop := (buttonTop - selectedItemOffset) - (r.itemHeights[r.selectedIndex]-buttonRect.Height())/2.0
	preferredMenuHeight := MaterialListPadding.Top + MaterialListPadding.Bottom
	for _, h := range r.itemHeights {
		preferredMenuHeight += h
	}

	// If there are too many elements in the menu, it is shrunk down so it is at
	// most the computed max height.
	menuHeight := math.Min(computedMaxHeight, preferredMenuHeight)
	menuBottom := menuTop + menuHeight

	// If the computed top or bottom of the menu are outside of the range
	// specified, they are brought into range. If the item height is larger than
	// the button height and the button is at the very bottom or top of the screen,
	// the menu will be aligned with the bottom or top of the button respectively.
	if menuTop < topLimit {
		menuTop = math.Min(buttonTop, topLimit)
		menuBottom = menuTop + menuHeight
	}
	if menuBottom > bottomLimit {
		menuBottom = math.Max(buttonBottom, bottomLimit)
		menuTop = menuBottom - menuHeight
	}
	if menuBottom-r.itemHeights[r.selectedIndex]/2.0 < buttonBottom-buttonRect.Height()/2.0 {
		menuBottom = buttonBottom - buttonRect.Height()/2.0 + r.itemHeights[r.selectedIndex]/2.0
		menuTop = menuBottom - menuHeight
	}

	scrollOffset := 0.0
	// If all of the menu items will not fit within the available height then the
	// scroll offset that will line the selected menu item up with the select item
	// is computed. This is only done when the menu is first shown: subsequently
	// the scroll offset is left where the user left it. This scroll offset is only
	// accurate for fixed height menu items (the default).
	if preferredMenuHeight > computedMaxHeight {
		// The offset should be zero if the selected item is in view at the beginning
		// of the menu. Otherwise, the scroll offset should center the item if
		// possible.
		scrollOffset = math.Max(0.0, selectedItemOffset-(buttonTop-menuTop))
		// If the selected item's scroll offset is greater than the maximum scroll
		// offset, set it instead to the maximum allowed scroll offset.
		scrollOffset = math.Min(scrollOffset, preferredMenuHeight-menuHeight)
	}
	return dropdownMenuLimits{menuTop, menuBottom, menuHeight, scrollOffset}
}

// dropdownRoutePage is the page of the route: the menu in a layout that
// positions it over the button.
//
// Flutter: material/dropdown.dart _DropdownRoutePage
type dropdownRoutePage[T comparable] struct {
	widgets.StatefulWidget
	route       *dropdownRoute[T]
	constraints rendering.BoxConstraints
}

func (w *dropdownRoutePage[T]) CreateState() widgets.State { return &dropdownRoutePageState[T]{} }

type dropdownRoutePageState[T comparable] struct {
	widgets.StateOf[*dropdownRoutePage[T]]
	scrollController *widgets.ScrollControllerBase
}

func (s *dropdownRoutePageState[T]) InitState() {
	// Computing the initial scroll offset now, before the items have been laid out.
	// This only works if the item heights are effectively fixed, i.e. either the
	// itemHeight of the button is specified or it is null and all of the items'
	// intrinsic heights are less than kMinInteractiveDimension. Otherwise the
	// initial scroll offset is just a rough approximation based on treating the
	// items as if their heights were all equal to kMinInteractiveDimension.
	route := s.Widget().route
	limits := route.getMenuLimits(route.buttonRect, s.Widget().constraints.MaxHeight, route.selectedIndex)
	s.scrollController = widgets.NewScrollController(widgets.ScrollControllerOpts{InitialScrollOffset: limits.scrollOffset})
}

func (s *dropdownRoutePageState[T]) Dispose() { s.scrollController.Dispose() }

func (s *dropdownRoutePageState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasDirectionality(bc)
	w := s.Widget()
	route := w.route
	textDirection := widgets.DirectionalityOf(bc)
	menu := &dropdownMenu[T]{
		StatefulWidget: widgets.NewStatefulWidgetBase(), route: route, padding: route.padding.Resolve(textDirection),
		buttonRect: route.buttonRect, constraints: w.constraints, dropdownColor: route.dropdownColor,
		enableFeedback: route.enableFeedback, borderRadius: route.borderRadius, scrollController: s.scrollController,
		mouseCursor: route.dropdownMenuItemMouseCursor,
	}
	return widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Top: true, Bottom: true, Left: true, Right: true},
		widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return widgets.NewCustomSingleChildLayout(&dropdownMenuRouteLayout[T]{
				SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil),
				buttonRect:                    route.buttonRect, route: route, textDirection: textDirection, menuWidth: route.menuWidth,
			}, route.capturedThemes.Wrap(menu))
		}))
}

// dropdownMenuRouteLayout positions the menu over the button.
//
// Flutter: material/dropdown.dart _DropdownMenuRouteLayout
type dropdownMenuRouteLayout[T comparable] struct {
	rendering.SingleChildLayoutDelegateBase
	buttonRect    ui.Rect
	route         *dropdownRoute[T]
	textDirection ui.TextDirection
	menuWidth     foundation.Opt[float64]
}

func (d *dropdownMenuRouteLayout[T]) GetSize(constraints rendering.BoxConstraints) ui.Size {
	return constraints.Biggest()
}

func (d *dropdownMenuRouteLayout[T]) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	// The maximum height of a simple menu should be one or more rows less than the
	// view height. This ensures a tappable area outside of the simple menu with
	// which to dismiss the menu.
	maxHeight := math.Max(0.0, constraints.MaxHeight-2*dropdownMenuItemHeight)
	if h, ok := d.route.menuMaxHeight.Get(); ok && h <= maxHeight {
		maxHeight = h
	}
	// The width of a menu should be at most the view width. This ensures that the
	// menu does not extend past the left and right edges of the screen.
	width := math.Min(constraints.MaxWidth, d.menuWidth.OrElse(d.buttonRect.Width()))
	return rendering.BoxConstraints{MinWidth: width, MaxWidth: width, MaxHeight: maxHeight}
}

func (d *dropdownMenuRouteLayout[T]) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	limits := d.route.getMenuLimits(d.buttonRect, size.Height, d.route.selectedIndex)
	var left float64
	if d.textDirection == ui.TextDirectionRTL {
		left = ui.ClampDouble(d.buttonRect.Right, 0.0, size.Width) - childSize.Width
	} else {
		left = ui.ClampDouble(d.buttonRect.Left, 0.0, size.Width-childSize.Width)
	}
	return ui.Offset{Dx: left, Dy: limits.top}
}

func (d *dropdownMenuRouteLayout[T]) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	o := old.(*dropdownMenuRouteLayout[T])
	return d.buttonRect != o.buttonRect || d.textDirection != o.textDirection
}

// dropdownMenu is the menu: the items in a list, in a background that grows from
// the selected item.
//
// Flutter: material/dropdown.dart _DropdownMenu
type dropdownMenu[T comparable] struct {
	widgets.StatefulWidget
	route            *dropdownRoute[T]
	padding          painting.EdgeInsets
	buttonRect       ui.Rect
	constraints      rendering.BoxConstraints
	dropdownColor    foundation.Opt[ui.Color]
	enableFeedback   bool
	borderRadius     foundation.Opt[painting.BorderRadius]
	scrollController *widgets.ScrollControllerBase
	mouseCursor      services.MouseCursor
}

func (w *dropdownMenu[T]) CreateState() widgets.State { return &dropdownMenuState[T]{} }

type dropdownMenuState[T comparable] struct {
	widgets.StateOf[*dropdownMenu[T]]
	fadeOpacity *animation.CurvedAnimation
	resize      *animation.CurvedAnimation
}

func (s *dropdownMenuState[T]) InitState() {
	// These animations are held as state because of their curve direction. When
	// the animation of the route reverses, recreating the CurvedAnimation objects
	// in build would lose the curve direction.
	a := s.Widget().route.Animation()
	s.fadeOpacity = animation.NewCurvedAnimation(a, animation.NewInterval(0.0, 0.25), animation.NewInterval(0.75, 1.0))
	s.resize = animation.NewCurvedAnimation(a, animation.NewInterval(0.25, 0.5), animation.NewThreshold(0.0))
}

func (s *dropdownMenuState[T]) Dispose() {
	s.fadeOpacity.Dispose()
	s.resize.Dispose()
}

// Build builds the menu, which is shown in three stages (unit timing in
// brackets):
//
//	[0s - 0.25s] - Fade in a rect-sized menu container with the selected item.
//	[0.25s - 0.5s] - Grow the otherwise empty menu container from the center
//	  until it's big enough for as many items as are going to be shown.
//	[0.5s - 1.0s] Fade in the remaining visible items from top to bottom.
//
// When the menu is dismissed the entire thing fades out in the first 0.25s.
//
// Flutter: material/dropdown.dart _DropdownMenuState.build
func (s *dropdownMenuState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	w := s.Widget()
	route := w.route
	localizations := MaterialLocalizationsOf(bc)
	theme := ThemeOf(bc)
	children := make([]widgets.Widget, 0, len(route.items))
	for i := range route.items {
		children = append(children, &dropdownMenuItemButton[T]{
			StatefulWidget: widgets.NewStatefulWidgetBase(), route: route, padding: w.padding, buttonRect: w.buttonRect,
			constraints: w.constraints, itemIndex: i, enableFeedback: w.enableFeedback, scrollController: w.scrollController,
			mouseCursor: w.mouseCursor,
		})
	}
	radius := painting.BorderRadiusGeometry(painting.BorderRadiusZero)
	clip := ui.ClipNone
	if br, ok := w.borderRadius.Get(); ok {
		radius = br
		clip = ui.ClipAntiAlias
	}
	return widgets.NewFadeTransition(s.fadeOpacity,
		widgets.NewCustomPaint(
			widgets.NewSemantics(
				widgets.NewClipRRect(
					NewMaterial(
						widgets.NewScrollConfiguration(
							widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{
								Scrollbars: foundation.Some(false), Overscroll: foundation.Some(false),
								Physics: widgets.NewClampingScrollPhysics(nil), Platform: theme.Platform}),
							widgets.NewPrimaryScrollController(w.scrollController,
								tempScrollbar(bc, true,
									widgets.NewListView(children, widgets.ListViewOpts{
										// Ensure this always inherits the PrimaryScrollController.
										Primary: foundation.Some(true), Padding: MaterialListPadding, ShrinkWrap: true})))),
						MaterialOpts{Type: MaterialTypeTransparency, TextStyle: route.style}),
					widgets.ClipRRectOpts{BorderRadius: radius, ClipBehavior: clip}),
				widgets.SemanticsOpts{
					Role: foundation.Some(ui.SemanticsRoleMenu), ScopesRoute: foundation.Some(true), NamesRoute: foundation.Some(true),
					ExplicitChildNodes: true, Label: foundation.Some(localizations.PopupMenuLabel())}),
			widgets.CustomPaintOpts{Painter: newDropdownMenuPainter(
				foundation.Some(w.dropdownColor.OrElse(theme.CanvasColor)), route.elevation, route.selectedIndex, w.borderRadius, s.resize,
				// This offset is passed as a callback, not a value, because it must be
				// retrieved at paint time (after layout), not at build time.
				func() float64 { return route.getItemOffset(route.selectedIndex) })}))
}

// dropdownMenuItemButton is the button that wraps a menu item.
//
// Flutter: material/dropdown.dart _DropdownMenuItemButton
type dropdownMenuItemButton[T comparable] struct {
	widgets.StatefulWidget
	route            *dropdownRoute[T]
	scrollController *widgets.ScrollControllerBase
	padding          painting.EdgeInsets
	buttonRect       ui.Rect
	constraints      rendering.BoxConstraints
	itemIndex        int
	enableFeedback   bool
	mouseCursor      services.MouseCursor
}

func (w *dropdownMenuItemButton[T]) CreateState() widgets.State {
	return &dropdownMenuItemButtonState[T]{}
}

type dropdownMenuItemButtonState[T comparable] struct {
	widgets.StateOf[*dropdownMenuItemButton[T]]
	opacityAnimation *animation.CurvedAnimation
}

func (s *dropdownMenuItemButtonState[T]) InitState() { s.setOpacityAnimation() }

func (s *dropdownMenuItemButtonState[T]) DidUpdateWidget(old *dropdownMenuItemButton[T]) {
	w := s.Widget()
	if old.itemIndex != w.itemIndex || old.route.Animation() != w.route.Animation() || old.route.selectedIndex != w.route.selectedIndex ||
		len(w.route.items) != len(old.route.items) {
		s.opacityAnimation.Dispose()
		s.setOpacityAnimation()
	}
}

func (s *dropdownMenuItemButtonState[T]) setOpacityAnimation() {
	w := s.Widget()
	unit := 0.5 / (float64(len(w.route.items)) + 1.5)
	if w.itemIndex == w.route.selectedIndex {
		s.opacityAnimation = animation.NewCurvedAnimation(w.route.Animation(), animation.NewThreshold(0.0))
		return
	}
	start := ui.ClampDouble(0.5+float64(w.itemIndex+1)*unit, 0.0, 1.0)
	end := ui.ClampDouble(start+1.5*unit, 0.0, 1.0)
	s.opacityAnimation = animation.NewCurvedAnimation(w.route.Animation(), animation.NewInterval(start, end))
}

func (s *dropdownMenuItemButtonState[T]) Dispose() { s.opacityAnimation.Dispose() }

func (s *dropdownMenuItemButtonState[T]) handleFocusChange(focused bool) {
	w := s.Widget()
	inTraditionalMode := widgets.WidgetsBindingOf(s.Context()).FocusManager().HighlightMode() == widgets.FocusHighlightModeTraditional
	if focused && inTraditionalMode {
		limits := w.route.getMenuLimits(w.buttonRect, w.constraints.MaxHeight, w.itemIndex)
		w.scrollController.AnimateTo(limits.scrollOffset, 100*time.Millisecond, animation.Curves.EaseInOut)
	}
}

func (s *dropdownMenuItemButtonState[T]) handleOnTap() {
	w := s.Widget()
	item := w.route.items[w.itemIndex].item
	if item.o.OnTap != nil {
		item.o.OnTap()
	}
	widgets.NavigatorPop(s.Context(), dropdownRouteResult[T]{result: item.o.Value})
}

func (s *dropdownMenuItemButtonState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	entry := w.route.items[w.itemIndex]
	child := widgets.Widget(entry.layout)
	child = widgets.NewPadding(w.padding, child)
	child = widgets.NewSizedBox(child, widgets.SizedBoxOpts{Height: w.route.itemHeight})
	isSelected := w.itemIndex == w.route.selectedIndex
	highlightMode := widgets.WidgetsBindingOf(bc).FocusManager().HighlightMode()
	// An InkWell is added to the item only if it is enabled.
	if entry.item.Enabled() {
		inner := child
		if highlightMode == widgets.FocusHighlightModeTouch {
			// When the highlight mode is traditional, the InkWell draws the selected
			// item background color. When it is touch, an Ink is inserted to force
			// the selected item background color.
			color := foundation.None[ui.Color]()
			if isSelected {
				color = foundation.Some(ThemeOf(bc).FocusColor)
			}
			inner = NewInk(child, InkOpts{Color: color})
		}
		child = NewInkWell(inner, InkWellOpts{
			Autofocus:      isSelected,
			EnableFeedback: foundation.Some(w.enableFeedback),
			OnTap:          s.handleOnTap,
			OnFocusChange:  s.handleFocusChange,
			MouseCursor:    w.mouseCursor,
		})
	}
	child = widgets.NewFadeTransition(s.opacityAnimation, child)
	return widgets.NewSemantics(child, widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleMenuItem)})
}

// DropdownButtonOpts are the optional parameters of NewDropdownButton.
//
// Flutter: material/dropdown.dart DropdownButton.new
type DropdownButtonOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SelectedItemBuilder builds the widgets that the button shows for the
	// selected item, one for each of the items.
	SelectedItemBuilder DropdownButtonBuilder
	// Value is the value of the currently selected item; none shows the hint.
	Value foundation.Opt[T]
	// Hint is the placeholder displayed before the user chooses a value.
	Hint widgets.Widget
	// DisabledHint is the placeholder displayed when the button is disabled.
	DisabledHint widgets.Widget
	// OnTap is called when the button is tapped, before the menu opens.
	OnTap func()
	// Elevation is the z-coordinate of the menu; the default is 8.
	Elevation foundation.Opt[int]
	// Style is the text style of the items and the hint; the default is the
	// titleMedium style of the theme.
	Style *painting.TextStyle
	// Underline is the widget drawn under the button; the default is a thin line.
	Underline widgets.Widget
	// Icon is the icon at the end of the button; the default is a drop-down arrow.
	Icon widgets.Widget
	// IconDisabledColor is the color of the icon when the button is disabled.
	IconDisabledColor foundation.Opt[ui.Color]
	// IconEnabledColor is the color of the icon when the button is enabled.
	IconEnabledColor foundation.Opt[ui.Color]
	// IconSize is the size of the icon; the default is 24.
	IconSize foundation.Opt[float64]
	// IsDense reduces the height of the button, from 48 to 24 and the height of
	// the text.
	IsDense bool
	// IsExpanded makes the button fill the width that it is given.
	IsExpanded bool
	// ItemHeight is the height of the items; the default is 48, and none lets the
	// items have their own height (it must be at least 48 when given).
	ItemHeight foundation.Opt[float64]
	// NoItemHeight lets the items have their own height (Flutter's itemHeight:
	// null).
	NoItemHeight bool
	// MenuWidth is the width of the menu; the default is the width of the button.
	MenuWidth foundation.Opt[float64]
	// FocusColor is the background color of the button when it has the focus.
	FocusColor foundation.Opt[ui.Color]
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus says that the button focuses itself when it is built.
	Autofocus bool
	// DropdownColor is the background color of the menu; the default is the
	// canvas color of the theme.
	DropdownColor foundation.Opt[ui.Color]
	// MenuMaxHeight is the maximum height of the menu.
	MenuMaxHeight foundation.Opt[float64]
	// EnableFeedback says whether detected gestures give acoustic and haptic
	// feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
	// Alignment is how the selected item is aligned in the button; the default is
	// the center of the start.
	Alignment painting.AlignmentGeometry
	// BorderRadius is the border radius of the focus ink and of the menu.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Padding is the padding around the button.
	Padding painting.EdgeInsetsGeometry
	// NoBarrierDismissible stops a tap outside the menu from closing it
	// (Flutter's barrierDismissible: false).
	NoBarrierDismissible bool
	// MouseCursor is the cursor over the button; the default is the clickable one.
	MouseCursor services.MouseCursor
	// DropdownMenuItemMouseCursor is the cursor over the items of the menu.
	DropdownMenuItemMouseCursor services.MouseCursor
}

// DropdownButton shows the currently selected item and an arrow that opens a
// menu for selecting another item. The type T is the type of the values that the
// items represent.
//
// Flutter: material/dropdown.dart DropdownButton
type DropdownButton[T comparable] struct {
	widgets.StatefulWidget
	items     []*DropdownMenuItem[T]
	onChanged func(value foundation.Opt[T])
	o         DropdownButtonOpts[T]
}

// NewDropdownButton creates the button; a nil list of items or a nil onChanged
// disables it. In debug builds it panics with the tag DropdownButtonValueNotUnique
// when the value is not the value of exactly one item, and with the tag
// DropdownButtonItemHeight when the item height is less than 48.
//
// Flutter: material/dropdown.dart DropdownButton.new
func NewDropdownButton[T comparable](items []*DropdownMenuItem[T], onChanged func(value foundation.Opt[T]), opts ...DropdownButtonOpts[T]) *DropdownButton[T] {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if v, ok := o.Value.Get(); ok && len(items) > 0 {
			n := 0
			for _, it := range items {
				if it.o.Value == foundation.Some(v) {
					n++
				}
			}
			if n != 1 {
				panic(errors.NewErrf("DropdownButtonValueNotUnique", "there should be exactly one item with the value of the DropdownButton: %v; either zero or 2 or more DropdownMenuItems were detected with the same value", v))
			}
		}
		if h, ok := o.ItemHeight.Get(); ok && h < MinInteractiveDimension {
			panic(errors.New("DropdownButtonItemHeight", "the itemHeight of a DropdownButton must be at least kMinInteractiveDimension", "itemHeight", h))
		}
	}
	return &DropdownButton[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), items: items, onChanged: onChanged, o: o}
}

// Items are the items of the menu.
//
// Flutter: material/dropdown.dart DropdownButton.items
func (w *DropdownButton[T]) Items() []*DropdownMenuItem[T] { return w.items }

// Value is the value of the selected item.
//
// Flutter: material/dropdown.dart DropdownButton.value
func (w *DropdownButton[T]) Value() foundation.Opt[T] { return w.o.Value }

// itemHeight is the height of the items: 48 by default, none when
// NoItemHeight.
func (w *DropdownButton[T]) itemHeight() foundation.Opt[float64] {
	if w.o.NoItemHeight {
		return foundation.None[float64]()
	}
	return foundation.Some(w.o.ItemHeight.OrElse(MinInteractiveDimension))
}

func (w *DropdownButton[T]) iconSize() float64 { return w.o.IconSize.OrElse(24.0) }

// CreateState creates the state.
//
// Flutter: material/dropdown.dart DropdownButton.createState
func (w *DropdownButton[T]) CreateState() widgets.State { return &dropdownButtonState[T]{} }

type dropdownButtonState[T comparable] struct {
	widgets.StateOf[*DropdownButton[T]]
	selectedIndex   foundation.Opt[int]
	dropdownRoute   *dropdownRoute[T]
	lastOrientation widgets.Orientation
	hasOrientation  bool
	internalNode    *widgets.FocusNode
	actionMap       widgets.ActionMap
	hasPrimaryFocus bool
	isMenuExpanded  bool
	focusedNode     *widgets.FocusNode
	focusHandle     foundation.ListenerHandle
}

func (s *dropdownButtonState[T]) focusNode() *widgets.FocusNode {
	if n := s.Widget().o.FocusNode; n != nil {
		return n
	}
	return s.internalNode
}

func (s *dropdownButtonState[T]) listen() {
	s.focusedNode = s.focusNode()
	s.focusHandle = s.focusedNode.AddListener(s.handleFocusChanged)
}

func (s *dropdownButtonState[T]) InitState() {
	s.updateSelectedIndex()
	if s.Widget().o.FocusNode == nil {
		s.internalNode = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "DropdownButton"})
	}
	activate := widgets.NewCallbackAction(func(widgets.ActivateIntent) any { s.handleTap(); return nil })
	buttonActivate := widgets.NewCallbackAction(func(widgets.ButtonActivateIntent) any { s.handleTap(); return nil })
	s.actionMap = widgets.ActionMap{activate.IntentType(): activate, buttonActivate.IntentType(): buttonActivate}
	s.listen()
}

func (s *dropdownButtonState[T]) Dispose() {
	s.removeDropdownRoute()
	s.focusedNode.RemoveListener(s.focusHandle)
	if s.internalNode != nil {
		s.internalNode.Dispose()
	}
}

func (s *dropdownButtonState[T]) handleFocusChanged() {
	if s.hasPrimaryFocus != s.focusNode().HasPrimaryFocus() {
		s.SetState(func() { s.hasPrimaryFocus = s.focusNode().HasPrimaryFocus() })
	}
}

func (s *dropdownButtonState[T]) removeDropdownRoute() {
	if s.dropdownRoute != nil {
		s.dropdownRoute.dismiss()
	}
	s.dropdownRoute = nil
	s.hasOrientation = false
}

func (s *dropdownButtonState[T]) DidUpdateWidget(old *DropdownButton[T]) {
	if s.Widget().o.FocusNode != old.o.FocusNode {
		s.focusedNode.RemoveListener(s.focusHandle)
		if s.internalNode != nil && s.Widget().o.FocusNode != nil {
			s.internalNode.Dispose()
			s.internalNode = nil
		}
		if s.Widget().o.FocusNode == nil && s.internalNode == nil {
			s.internalNode = widgets.NewFocusNode(widgets.FocusNodeOpts{DebugLabel: "DropdownButton"})
		}
		s.hasPrimaryFocus = s.focusNode().HasPrimaryFocus()
		s.listen()
	}
	s.updateSelectedIndex()
}

func (s *dropdownButtonState[T]) updateSelectedIndex() {
	w := s.Widget()
	s.selectedIndex = foundation.None[int]()
	if len(w.items) == 0 {
		return
	}
	if w.o.Value.IsNone() {
		hasNoneItem := false
		for _, it := range w.items {
			if it.Enabled() && it.o.Value == w.o.Value {
				hasNoneItem = true
			}
		}
		if !hasNoneItem {
			return
		}
	}
	for i, it := range w.items {
		if it.o.Value == w.o.Value {
			s.selectedIndex = foundation.Some(i)
			return
		}
	}
}

func (s *dropdownButtonState[T]) textStyle(bc widgets.BuildContext) *painting.TextStyle {
	if st := s.Widget().o.Style; st != nil {
		return st
	}
	return ThemeOf(bc).TextTheme.TitleMedium
}

func (s *dropdownButtonState[T]) enabled() bool {
	return len(s.Widget().items) > 0 && s.Widget().onChanged != nil
}

// handleTap opens the menu over the button.
//
// Flutter: material/dropdown.dart _DropdownButtonState._handleTap
func (s *dropdownButtonState[T]) handleTap() {
	w := s.Widget()
	bc := s.Context()
	textDirection := widgets.DirectionalityOf(bc)
	// Deviation: the legacy ButtonTheme, with its alignedDropdown, is not ported:
	// the menu is never aligned with the button.
	menuMargin := painting.EdgeInsetsDirectional{Start: 16.0, End: 24.0}
	heights := make([]float64, len(w.items))
	itemHeight := w.itemHeight().OrElse(MinInteractiveDimension)
	for i := range heights {
		heights[i] = itemHeight
	}
	navigator := widgets.NavigatorOf(bc)
	if foundation.DebugMode && s.dropdownRoute != nil {
		panic(errors.New("DropdownButtonRouteOpen", "the menu of a DropdownButton was opened while it was already open"))
	}
	itemBox := bc.FindRenderObject().(rendering.Box)
	itemRect := ui.Rect{}.Shift(itemBox.LocalToGlobal(ui.Offset{}, navigator.Context().FindRenderObject()))
	itemRect = ui.Rect{Left: itemRect.Left, Top: itemRect.Top, Right: itemRect.Left + itemBox.Size().Width, Bottom: itemRect.Top + itemBox.Size().Height}
	margin := menuMargin.Resolve(textDirection)
	buttonRect := ui.Rect{Left: itemRect.Left - margin.Left, Top: itemRect.Top - margin.Top, Right: itemRect.Right + margin.Right, Bottom: itemRect.Bottom + margin.Bottom}

	route := &dropdownRoute[T]{
		padding: dropdownMenuItemPadding, buttonRect: buttonRect, selectedIndex: s.selectedIndex.OrElse(0),
		elevation: w.o.Elevation.OrElse(8), capturedThemes: widgets.CaptureThemes(bc, navigator.Context()), style: s.textStyle(bc),
		barrierLabel: MaterialLocalizationsOf(bc).ModalBarrierDismissLabel(), itemHeight: w.itemHeight(), menuWidth: w.o.MenuWidth,
		dropdownColor: w.o.DropdownColor, menuMaxHeight: w.o.MenuMaxHeight, enableFeedback: w.o.EnableFeedback.OrElse(true),
		borderRadius: w.o.BorderRadius, barrierDismissible: !w.o.NoBarrierDismissible,
		dropdownMenuItemMouseCursor: w.o.DropdownMenuItemMouseCursor, itemHeights: heights,
	}
	items := make([]dropdownItem[T], len(w.items))
	for index, item := range w.items {
		index := index
		items[index] = dropdownItem[T]{item: item, layout: newDropdownMenuItemLayout(func(size ui.Size) {
			// If the route is nil and onLayout is called, this means that
			// PerformLayout was called on a route that has not left the widget tree
			// but is already on its way out. Since onLayout is used primarily to
			// collect the desired heights of each menu item before laying them out,
			// not having the route collect each item's height to lay out is fine
			// since the route is already on its way out.
			if s.dropdownRoute == nil {
				return
			}
			s.dropdownRoute.itemHeights[index] = size.Height
		}, item)}
	}
	route.items = items
	widgets.InitPopupRouteBase(&route.PopupRouteBase, route, widgets.ModalRouteOpts{})
	s.dropdownRoute = route
	s.focusNode().RequestFocus()
	navigator.Push(route)
	route.PoppedThen(func(result any) {
		s.removeDropdownRoute()
		if s.Mounted() {
			s.SetState(func() { s.isMenuExpanded = false })
		}
		r, ok := result.(dropdownRouteResult[T])
		if !s.Mounted() || !ok {
			return
		}
		if f := s.Widget().onChanged; f != nil {
			f(r.result)
		}
	})
	if w.o.OnTap != nil {
		w.o.OnTap()
	}
	s.SetState(func() { s.isMenuExpanded = true })
}

// denseButtonHeight is the height of the button when it is dense: reduced from
// 48 to 24, but not smaller than the text that it contains or than its icon.
func (s *dropdownButtonState[T]) denseButtonHeight(bc widgets.BuildContext) float64 {
	style := s.textStyle(bc)
	titleMedium := ThemeOf(bc).TextTheme.TitleMedium
	fontSize := style.FontSize.OrElse(titleMedium.FontSize.Value())
	lineHeight := style.Height.OrElse(titleMedium.Height.OrElse(1.0))
	scaledFontSize := widgets.MediaQueryTextScalerOf(bc).Scale(fontSize * lineHeight)
	return math.Max(scaledFontSize, math.Max(s.Widget().iconSize(), dropdownDenseHeight))
}

// iconColor is the color of the icon: these colors are not defined in the
// Material Design spec.
func (s *dropdownButtonState[T]) iconColor(bc widgets.BuildContext) ui.Color {
	brightness := ThemeOf(bc).Brightness()
	w := s.Widget()
	if s.enabled() {
		if c, ok := w.o.IconEnabledColor.Get(); ok {
			return c
		}
		if brightness == ui.BrightnessDark {
			return Colors.White70
		}
		return Colors.Grey.Shade700()
	}
	if c, ok := w.o.IconDisabledColor.Get(); ok {
		return c
	}
	if brightness == ui.BrightnessDark {
		return Colors.White10
	}
	return Colors.Grey.Shade400()
}

// Build builds the button.
//
// Flutter: material/dropdown.dart _DropdownButtonState.build
func (s *dropdownButtonState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterial(bc)
	DebugCheckHasMaterialLocalizations(bc)
	w := s.Widget()
	theme := ThemeOf(bc)
	newOrientation := widgets.MediaQueryOrientationOf(bc)
	if !s.hasOrientation {
		s.lastOrientation, s.hasOrientation = newOrientation, true
	}
	if newOrientation != s.lastOrientation {
		s.removeDropdownRoute()
		s.lastOrientation, s.hasOrientation = newOrientation, true
	}
	textStyle := s.textStyle(bc)

	// The width of the button and the menu are defined by the widest item and the
	// width of the hint.
	var items []widgets.Widget
	if w.o.SelectedItemBuilder != nil {
		selected := w.o.SelectedItemBuilder(bc)
		if foundation.DebugMode && len(w.items) > 0 && len(selected) != len(w.items) {
			panic(errors.NewErrf("DropdownButtonSelectedItemBuilder", "the selectedItemBuilder must return a list of widgets with the same length as the items list; currently, selectedItemBuilder returns a list of length %d, but items has length %d", len(selected), len(w.items)))
		}
		items = append(items, selected...)
	} else {
		for _, it := range w.items {
			items = append(items, it)
		}
	}
	hintIndex := foundation.None[int]()
	if w.o.Hint != nil || (!s.enabled() && w.o.DisabledHint != nil) {
		displayedHint := w.o.Hint
		if !s.enabled() && w.o.DisabledHint != nil {
			displayedHint = w.o.DisabledHint
		}
		hintIndex = foundation.Some(len(items))
		items = append(items, widgets.NewDefaultTextStyle(
			textStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.HintColor)}),
			widgets.NewIgnorePointer(dropdownMenuItemContainer(w.o.Alignment, displayedHint))))
	}

	// Deviation: the legacy ButtonTheme, with its alignedDropdown, is not ported: the
	// button has no padding.
	var padding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero

	// If the value is none (then the selected index is none) the hint is displayed,
	// or nothing at all.
	var innerItems widgets.Widget
	if len(items) == 0 {
		innerItems = widgets.NewSizedBox(nil)
	} else {
		children := items
		if !w.o.IsDense {
			children = make([]widgets.Widget, len(items))
			for i, item := range items {
				if h, ok := w.itemHeight().Get(); ok {
					children[i] = widgets.NewSizedBox(item, widgets.SizedBoxOpts{Height: foundation.Some(h)})
				} else {
					children[i] = widgets.NewColumn([]widgets.Widget{item}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin})
				}
			}
		}
		stackOpts := widgets.IndexedStackOpts{Alignment: w.o.Alignment}
		if idx, ok := s.selectedIndex.Get(); ok {
			stackOpts.Index = foundation.Some(idx)
		} else if h, ok := hintIndex.Get(); ok {
			stackOpts.Index = foundation.Some(h)
		} else {
			stackOpts.NoIndex = true
		}
		innerItems = widgets.NewIndexedStack(children, stackOpts)
	}

	icon := w.o.Icon
	if icon == nil {
		icon = widgets.NewIcon(Icons.ArrowDropDown())
	}
	effectiveSuffixIcon := widgets.NewIconTheme(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
		Color: foundation.Some(s.iconColor(bc)), Size: foundation.Some(w.iconSize())}), icon)

	rowChildren := []widgets.Widget{innerItems}
	if w.o.IsExpanded {
		rowChildren[0] = widgets.NewExpanded(innerItems)
	}
	rowChildren = append(rowChildren, effectiveSuffixIcon)
	style := textStyle
	if !s.enabled() {
		style = textStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.DisabledColor)})
	}
	height := foundation.None[float64]()
	if w.o.IsDense {
		height = foundation.Some(s.denseButtonHeight(bc))
	}
	var result widgets.Widget = widgets.NewDefaultTextStyle(style,
		widgets.NewSizedBox(
			widgets.NewPadding(padding.Resolve(widgets.DirectionalityOf(bc)),
				widgets.NewRow(rowChildren, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween, MainAxisSize: rendering.MainAxisSizeMin})),
			widgets.SizedBoxOpts{Height: height}))

	if !DropdownButtonHideUnderlineAt(bc) {
		bottom := 8.0
		if w.o.IsDense || w.o.NoItemHeight {
			bottom = 0.0
		}
		underline := w.o.Underline
		if underline == nil {
			underline = widgets.NewContainer(nil, widgets.ContainerOpts{
				Height: foundation.Some(1.0),
				Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
					Border: painting.Border{Bottom: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFFBDBDBD)), Width: foundation.Some(0.0)})}}),
			})
		}
		result = widgets.NewStack([]widgets.Widget{
			result,
			widgets.NewPositioned(underline, widgets.PositionedOpts{Left: foundation.Some(0.0), Right: foundation.Some(0.0), Bottom: foundation.Some(bottom)}),
		})
	}

	var cursor services.MouseCursor = widgets.WidgetStateMouseCursorAdaptiveClickable
	if w.o.MouseCursor != nil {
		cursor = w.o.MouseCursor
	}
	states := widgets.NewWidgetStates()
	if !s.enabled() {
		states = states.With(widgets.WidgetStateDisabled)
	}
	effectiveCursor := cursor
	if p, ok := cursor.(widgets.WidgetStateProperty[services.MouseCursor]); ok {
		effectiveCursor = p.Resolve(states)
	}
	var onTap func()
	if s.enabled() {
		onTap = s.handleTap
	}
	inner := widgets.Widget(result)
	if w.o.Padding != nil {
		inner = widgets.NewPadding(w.o.Padding, result)
	}
	// Deviation: Flutter's DropdownButtonFormField, which gives the button an
	// InputDecorator, comes with step 13.5, with the InputDecorator.
	result = NewInkWell(inner, InkWellOpts{
		MouseCursor:     effectiveCursor,
		OnTap:           onTap,
		CanRequestFocus: foundation.Some(s.enabled()),
		BorderRadius:    w.o.BorderRadius,
		FocusNode:       s.focusNode(),
		Autofocus:       w.o.Autofocus,
		FocusColor:      firstOpt(w.o.FocusColor, foundation.Some(theme.FocusColor)),
		EnableFeedback:  foundation.Some(false),
	})
	childHasButtonSemantic := hintIndex.IsSome() || (s.selectedIndex.IsSome() && w.o.SelectedItemBuilder == nil)
	return widgets.NewSemantics(widgets.NewActions(s.actionMap, result), widgets.SemanticsOpts{
		Button: foundation.Some(!childHasButtonSemantic), Expanded: foundation.Some(s.isMenuExpanded)})
}
