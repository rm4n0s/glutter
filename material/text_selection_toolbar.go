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
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// textSelectionToolbarHeight is the height of the toolbar when its overflow
	// menu is closed.
	//
	// Flutter: material/text_selection_toolbar.dart _kToolbarHeight
	textSelectionToolbarHeight = 44.0
	// textSelectionToolbarContentDistance is the distance between the content and
	// the toolbar above it.
	//
	// Flutter: material/text_selection_toolbar.dart _kToolbarContentDistance
	textSelectionToolbarContentDistance = 8.0
	// textSelectionToolbarScreenPadding is the minimum space between the toolbar
	// and the edges of the screen.
	//
	// Deviation: Flutter reads CupertinoTextSelectionToolbar.kToolbarScreenPadding,
	// which has the same value; there is no Cupertino library (SPEC §12.10).
	//
	// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.kToolbarScreenPadding
	textSelectionToolbarScreenPadding = 8.0
	// textSelectionToolbarOverflowDuration is the duration of the change of the
	// size of the toolbar when the overflow menu opens or closes. It was eyeballed
	// on a Pixel 2 emulator running Android API 28.
	//
	// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowableState.build
	textSelectionToolbarOverflowDuration = 140 * time.Millisecond
)

const (
	// TextSelectionToolbarHandleSize is the size of the text selection handles.
	//
	// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.kHandleSize
	TextSelectionToolbarHandleSize = 22.0
	// TextSelectionToolbarContentDistanceBelow is the padding between the toolbar
	// and the anchor when the toolbar is below it.
	//
	// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.kToolbarContentDistanceBelow
	TextSelectionToolbarContentDistanceBelow = TextSelectionToolbarHandleSize - 2.0
)

// TextSelectionToolbarOpts are the optional parameters of
// NewTextSelectionToolbar.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.new
type TextSelectionToolbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ToolbarBuilder builds the toolbar container: useful for customizing the
	// high-level background of the toolbar. The given child widget contains all
	// of the children. The default is a Material card in the color of the
	// surface.
	ToolbarBuilder widgets.ToolbarBuilder
}

// TextSelectionToolbar is a fully-functional Material-style text selection
// toolbar.
//
// It tries to position itself above anchorAbove, but if it does not fit, then it
// positions itself below anchorBelow. If any children do not fit in the menu,
// an overflow menu is created automatically.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar
type TextSelectionToolbar struct {
	widgets.StatelessWidget
	anchorAbove    ui.Offset
	anchorBelow    ui.Offset
	toolbarBuilder widgets.ToolbarBuilder
	children       []widgets.Widget
}

// NewTextSelectionToolbar creates a toolbar with the children, typically
// TextSelectionToolbarTextButtons. The anchorAbove is the focal point above
// which the toolbar attempts to position itself (when there is not enough room
// above before reaching the top of the screen, the toolbar positions itself
// below anchorBelow). It panics with the tag TextSelectionToolbarNoChildren (in
// debug builds) when there is no child.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.new
func NewTextSelectionToolbar(anchorAbove, anchorBelow ui.Offset, children []widgets.Widget, opts ...TextSelectionToolbarOpts) *TextSelectionToolbar {
	o := firstOpts(opts)
	if foundation.DebugMode && len(children) == 0 {
		panic(errors.New(tagTextSelectionToolbarNoChildren, "a TextSelectionToolbar needs at least one child"))
	}
	if o.ToolbarBuilder == nil {
		o.ToolbarBuilder = textSelectionToolbarDefaultToolbarBuilder
	}
	return &TextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		anchorAbove: anchorAbove, anchorBelow: anchorBelow, toolbarBuilder: o.ToolbarBuilder, children: children}
}

// AnchorAbove is the focal point above which the toolbar attempts to position
// itself. If there is not enough room above before reaching the top of the
// screen, then the toolbar positions itself below AnchorBelow.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.anchorAbove
func (w *TextSelectionToolbar) AnchorAbove() ui.Offset { return w.anchorAbove }

// AnchorBelow is the focal point below which the toolbar attempts to position
// itself, if it does not fit above AnchorAbove.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.anchorBelow
func (w *TextSelectionToolbar) AnchorBelow() ui.Offset { return w.anchorBelow }

// Children are the children that are displayed in the toolbar: typically
// buttons.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.children
func (w *TextSelectionToolbar) Children() []widgets.Widget { return w.children }

// ToolbarBuilder builds the toolbar container.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.toolbarBuilder
func (w *TextSelectionToolbar) ToolbarBuilder() widgets.ToolbarBuilder { return w.toolbarBuilder }

// textSelectionToolbarDefaultToolbarBuilder builds the default Android Material
// text selection menu toolbar.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar._defaultToolbarBuilder
func textSelectionToolbarDefaultToolbarBuilder(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return &textSelectionToolbarContainer{StatelessWidget: widgets.NewStatelessWidgetBase(), child: child}
}

// Build positions the toolbar above anchorAbove if it fits there, and below
// anchorBelow otherwise.
//
// Flutter: material/text_selection_toolbar.dart TextSelectionToolbar.build
func (w *TextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	// Incorporate the padding distance between the content and toolbar.
	anchorAbovePadded := w.anchorAbove.Sub(ui.Offset{Dy: textSelectionToolbarContentDistance})
	anchorBelowPadded := w.anchorBelow.Add(ui.Offset{Dy: TextSelectionToolbarContentDistanceBelow})

	const screenPadding = textSelectionToolbarScreenPadding
	paddingAbove := widgets.MediaQueryPaddingOf(bc).Top + screenPadding
	availableHeight := anchorAbovePadded.Dy - textSelectionToolbarContentDistance - paddingAbove
	fitsAbove := textSelectionToolbarHeight <= availableHeight
	// Makes up for the Padding above the Stack.
	localAdjustment := ui.Offset{Dx: screenPadding, Dy: paddingAbove}

	return widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(screenPadding, paddingAbove, screenPadding, screenPadding),
		widgets.NewCustomSingleChildLayout(
			widgets.NewTextSelectionToolbarLayoutDelegate(anchorAbovePadded.Sub(localAdjustment), anchorBelowPadded.Sub(localAdjustment),
				widgets.TextSelectionToolbarLayoutDelegateOpts{FitsAbove: foundation.Some(fitsAbove)}),
			&textSelectionToolbarOverflowable{StatefulWidget: widgets.NewStatefulWidgetBase(), isAbove: fitsAbove,
				toolbarBuilder: w.toolbarBuilder, children: w.children}))
}

// textSelectionToolbarOverflowable is a toolbar containing the given children.
// If they overflow the width available, then the overflowing children are
// displayed in an overflow menu.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowable
type textSelectionToolbarOverflowable struct {
	widgets.StatefulWidget
	children []widgets.Widget
	// isAbove: when true, the toolbar fits above its anchor and is positioned
	// there.
	isAbove bool
	// toolbarBuilder builds the toolbar that is populated with the children and
	// fit inside of the layout that adjusts to overflow.
	toolbarBuilder widgets.ToolbarBuilder
}

// CreateState creates the state.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowable.createState
func (w *textSelectionToolbarOverflowable) CreateState() widgets.State {
	return &textSelectionToolbarOverflowableState{containerKey: foundation.NewUniqueKey()}
}

// textSelectionToolbarOverflowableState is the state of the overflowable
// toolbar: it knows whether the overflow menu is open.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowableState
type textSelectionToolbarOverflowableState struct {
	widgets.StateOf[*textSelectionToolbarOverflowable]
	// overflowOpen is whether or not the overflow menu is open. When it is
	// closed, the menu items that do not overflow are shown. When it is open,
	// only the overflowing menu items are shown.
	overflowOpen bool
	// containerKey is the key for textSelectionToolbarTrailingEdgeAlign.
	containerKey foundation.Key
}

// reset closes the menu and resets layout calculations, as in when the menu has
// changed and saved values are no longer relevant. This is called in setState
// or another context where a rebuild is happening.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowableState._reset
func (s *textSelectionToolbarOverflowableState) reset() {
	// Change textSelectionToolbarTrailingEdgeAlign's key when the menu changes in
	// order to cause it to rebuild. This lets it recalculate its saved width for
	// the new set of children, and it prevents AnimatedSize from animating the
	// size change.
	s.containerKey = foundation.NewUniqueKey()
	// If the menu items change, make sure the overflow menu is closed. This
	// prevents getting into a broken state where overflowOpen is true when there
	// are not enough children to cause overflow.
	s.overflowOpen = false
}

// DidUpdateWidget resets the state if the children changed at all.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowableState.didUpdateWidget
func (s *textSelectionToolbarOverflowableState) DidUpdateWidget(old *textSelectionToolbarOverflowable) {
	// If the children are changing at all, the current page should be reset.
	if !widgetListEquals(s.Widget().children, old.children) {
		s.reset()
	}
}

// widgetListEquals is whether the two lists have the same widgets, the same
// instances, in the same order.
//
// Flutter: foundation/collections.dart listEquals
func widgetListEquals(a, b []widgets.Widget) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 || &a[0] == &b[0] {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Build builds the toolbar with its overflow button.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowableState.build
func (s *textSelectionToolbarOverflowableState) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	localizations := MaterialLocalizationsOf(bc)
	textDirection := widgets.DirectionalityOf(bc)
	w := s.Widget()

	// The navButton that shows and hides the overflow menu is the first child.
	var navButton widgets.Widget
	if s.overflowOpen {
		navButton = newTextSelectionToolbarOverflowButton(widgets.StandardComponentTypeBackButton.Key(), widgets.NewIcon(Icons.ArrowBack()),
			s.toggleOverflow, localizations.BackButtonTooltip())
	} else {
		navButton = newTextSelectionToolbarOverflowButton(widgets.StandardComponentTypeMoreButton.Key(), widgets.NewIcon(Icons.MoreVert()),
			s.toggleOverflow, localizations.MoreButtonTooltip())
	}
	items := make([]widgets.Widget, 0, len(w.children)+1)
	items = append(items, navButton)
	items = append(items, w.children...)

	return &textSelectionToolbarTrailingEdgeAlign{
		SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(
			widgets.NewAnimatedSize(textSelectionToolbarOverflowDuration, w.toolbarBuilder(bc, &textSelectionToolbarItemsLayout{
				MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(items),
				isAbove:                      w.isAbove, overflowOpen: s.overflowOpen, textDirection: textDirection})),
			widgets.WidgetOpts{Key: s.containerKey}),
		overflowOpen: s.overflowOpen, textDirection: textDirection,
	}
}

// toggleOverflow opens the overflow menu if it is closed, and closes it if it is
// open.
func (s *textSelectionToolbarOverflowableState) toggleOverflow() {
	s.SetState(func() { s.overflowOpen = !s.overflowOpen })
}

// textSelectionToolbarTrailingEdgeAlign: when the overflow menu is open, it
// tries to align its trailing edge to the trailing edge of the closed menu. This
// widget handles this effect by measuring and maintaining the width of the
// closed menu and aligning the child to that side.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlign
type textSelectionToolbarTrailingEdgeAlign struct {
	widgets.SingleChildRenderObjectWidget
	overflowOpen  bool
	textDirection ui.TextDirection
}

// CreateRenderObject creates the render object.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlign.createRenderObject
func (w *textSelectionToolbarTrailingEdgeAlign) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderTextSelectionToolbarTrailingEdgeAlign{overflowOpen: w.overflowOpen, textDirection: w.textDirection}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlign.updateRenderObject
func (w *textSelectionToolbarTrailingEdgeAlign) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderTextSelectionToolbarTrailingEdgeAlign)
	r.setOverflowOpen(w.overflowOpen)
	r.setTextDirection(w.textDirection)
}

// renderTextSelectionToolbarTrailingEdgeAlign is the render object of
// textSelectionToolbarTrailingEdgeAlign.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox
type renderTextSelectionToolbarTrailingEdgeAlign struct {
	rendering.RenderProxyBox
	// closedWidth is the width of the menu when it was closed. This is used to
	// achieve the behavior where the open menu aligns its trailing edge to the
	// closed menu's trailing edge.
	closedWidth   foundation.Opt[float64]
	overflowOpen  bool
	textDirection ui.TextDirection
}

func (r *renderTextSelectionToolbarTrailingEdgeAlign) setOverflowOpen(value bool) {
	if value == r.overflowOpen {
		return
	}
	r.overflowOpen = value
	r.MarkNeedsLayout()
}

func (r *renderTextSelectionToolbarTrailingEdgeAlign) setTextDirection(value ui.TextDirection) {
	if value == r.textDirection {
		return
	}
	r.textDirection = value
	r.MarkNeedsLayout()
}

// childData is the parent data of the child.
func (r *renderTextSelectionToolbarTrailingEdgeAlign) childData() *widgets.ToolbarItemsParentData {
	return r.Child().ParentData().(*widgets.ToolbarItemsParentData)
}

// PerformLayout lays the child out loosely, remembers its width when the menu is
// closed, and aligns it to the trailing edge of that width.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox.performLayout
func (r *renderTextSelectionToolbarTrailingEdgeAlign) PerformLayout() {
	constraints := r.BoxConstraints()
	child := r.Child()
	child.Layout(constraints.Loosen(), rendering.LayoutOpts{ParentUsesSize: true})

	// Save the width when the menu is closed. If the menu changes, this width is
	// invalid, so it's important that this RenderBox be recreated in that case.
	// Currently, this is achieved by providing a new key to
	// textSelectionToolbarTrailingEdgeAlign.
	if !r.overflowOpen && r.closedWidth.IsNone() {
		r.closedWidth = foundation.Some(child.Size().Width)
	}

	// If the open menu is wider than the closed menu, just use its own width and
	// don't worry about aligning the trailing edges. closedWidth is used even
	// when the menu is closed to allow it to animate its size while keeping the
	// same edge alignment.
	width := child.Size().Width
	if closed, ok := r.closedWidth.Get(); ok && !(child.Size().Width > closed) {
		width = closed
	}
	r.SetSize(constraints.Constrain(ui.Size{Width: width, Height: child.Size().Height}))

	// Set the offset in the parent data such that the child will be aligned to the
	// trailing edge, depending on the text direction.
	dx := r.Size().Width - child.Size().Width
	if r.textDirection == ui.TextDirectionRTL {
		dx = 0.0
	}
	r.childData().Offset = ui.Offset{Dx: dx}
}

// SetupParentData gives the child a ToolbarItemsParentData if it has none.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox.setupParentData
func (r *renderTextSelectionToolbarTrailingEdgeAlign) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*widgets.ToolbarItemsParentData); !ok {
		child.AsRenderObject().SetParentData(&widgets.ToolbarItemsParentData{})
	}
}

// Paint paints the child at the offset set in the parent data.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox.paint
func (r *renderTextSelectionToolbarTrailingEdgeAlign) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	context.PaintChild(r.Child(), r.childData().Offset.Add(offset))
}

// HitTestChildren includes the parent data offset in the hit test.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox.hitTestChildren
func (r *renderTextSelectionToolbarTrailingEdgeAlign) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	// The x, y parameters have the top left of the node's box as the origin.
	child := r.Child()
	offset := r.childData().Offset
	return result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
		return child.HitTest(result, transformed)
	})
}

// ApplyPaintTransform translates by the offset of the child.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarTrailingEdgeAlignRenderBox.applyPaintTransform
func (r *renderTextSelectionToolbarTrailingEdgeAlign) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	offset := child.ParentData().(*widgets.ToolbarItemsParentData).Offset
	transform.Translate(float32(offset.Dx), float32(offset.Dy), 0)
	r.RenderProxyBox.ApplyPaintTransform(child, transform)
}

// textSelectionToolbarItemsLayout renders the menu items in the correct
// positions in the menu and its overflow submenu based on calculating which item
// would first overflow.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayout
type textSelectionToolbarItemsLayout struct {
	widgets.MultiChildRenderObjectWidget
	isAbove       bool
	overflowOpen  bool
	textDirection ui.TextDirection
}

// CreateRenderObject creates the render object.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayout.createRenderObject
func (w *textSelectionToolbarItemsLayout) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderTextSelectionToolbarItemsLayout{lastIndexThatFits: -1, isAbove: w.isAbove, overflowOpen: w.overflowOpen, textDirection: w.textDirection}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayout.updateRenderObject
func (w *textSelectionToolbarItemsLayout) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderTextSelectionToolbarItemsLayout)
	r.setIsAbove(w.isAbove)
	r.setTextDirection(w.textDirection)
	r.setOverflowOpen(w.overflowOpen)
}

// CreateElement creates the element that tells which children are on stage.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayout.createElement
func (w *textSelectionToolbarItemsLayout) CreateElement() widgets.Element {
	e := &textSelectionToolbarItemsLayoutElement{}
	widgets.InitMultiChildRenderObjectElement(&e.MultiChildRenderObjectElement, e, w)
	return e
}

// textSelectionToolbarItemsLayoutElement is the element of
// textSelectionToolbarItemsLayout: only the children that are painted are
// on stage.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayoutElement
type textSelectionToolbarItemsLayoutElement struct {
	widgets.MultiChildRenderObjectElement
}

// DebugVisitOnstageChildren visits the children that are painted.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayoutElement.debugVisitOnstageChildren
func (e *textSelectionToolbarItemsLayoutElement) DebugVisitOnstageChildren(visitor widgets.ElementVisitor) {
	for _, child := range e.Children() {
		if textSelectionToolbarShouldPaint(child) {
			visitor(child)
		}
	}
}

// textSelectionToolbarShouldPaint is whether the child element is painted.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarItemsLayoutElement._shouldPaint
func textSelectionToolbarShouldPaint(child widgets.Element) bool {
	return child.RenderObject().ParentData().(*widgets.ToolbarItemsParentData).ShouldPaint
}

// renderTextSelectionToolbarItemsLayout is the render object of
// textSelectionToolbarItemsLayout.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout
type renderTextSelectionToolbarItemsLayout struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	// lastIndexThatFits is the index of the last item that does not overflow.
	lastIndexThatFits int
	isAbove           bool
	overflowOpen      bool
	textDirection     ui.TextDirection
}

func (r *renderTextSelectionToolbarItemsLayout) setIsAbove(value bool) {
	if value == r.isAbove {
		return
	}
	r.isAbove = value
	r.MarkNeedsLayout()
}

func (r *renderTextSelectionToolbarItemsLayout) setOverflowOpen(value bool) {
	if value == r.overflowOpen {
		return
	}
	r.overflowOpen = value
	r.MarkNeedsLayout()
}

func (r *renderTextSelectionToolbarItemsLayout) setTextDirection(value ui.TextDirection) {
	if value == r.textDirection {
		return
	}
	r.textDirection = value
	r.MarkNeedsLayout()
}

func toolbarItemData(child rendering.Box) *widgets.ToolbarItemsParentData {
	return child.ParentData().(*widgets.ToolbarItemsParentData)
}

// layoutChildren lays out the necessary children, and figures out where the
// children first overflow, if at all.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._layoutChildren
func (r *renderTextSelectionToolbarItemsLayout) layoutChildren() {
	// When overflow is not open, the toolbar is always a specific height.
	sizedConstraints := r.BoxConstraints()
	if !r.overflowOpen {
		sizedConstraints = rendering.NewBoxConstraintsLoose(ui.Size{Width: sizedConstraints.MaxWidth, Height: textSelectionToolbarHeight})
	}

	i := -1
	width := 0.0
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		i++

		// No need to layout children inside the overflow menu when it's closed.
		// The opposite is not true. It is necessary to layout the children that
		// don't overflow when the overflow menu is open in order to calculate
		// lastIndexThatFits.
		if r.lastIndexThatFits != -1 && !r.overflowOpen {
			continue
		}

		child.Layout(sizedConstraints.Loosen(), rendering.LayoutOpts{ParentUsesSize: true})
		width += child.Size().Width

		if width > sizedConstraints.MaxWidth && r.lastIndexThatFits == -1 {
			r.lastIndexThatFits = i - 1
		}
	}

	// If the last child overflows, but only because of the width of the overflow
	// button, then just show it and hide the overflow button.
	navButton := r.FirstChild()
	if r.lastIndexThatFits != -1 && r.lastIndexThatFits == r.ChildCount()-2 &&
		width-navButton.Size().Width <= sizedConstraints.MaxWidth {
		r.lastIndexThatFits = -1
	}
}

// shouldPaintChild returns true when the child should be painted, false
// otherwise.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._shouldPaintChild
func (r *renderTextSelectionToolbarItemsLayout) shouldPaintChild(child rendering.Box, index int) bool {
	// Paint the navButton when there is overflow.
	if child == r.FirstChild() {
		return r.lastIndexThatFits != -1
	}

	// If there is no overflow, all children besides the navButton are painted.
	if r.lastIndexThatFits == -1 {
		return true
	}

	// When there is overflow, paint if the child is in the part of the menu that
	// is currently open. Overflowing children are painted when the overflow menu
	// is open, and the children that fit are painted when the overflow menu is
	// closed.
	return (index > r.lastIndexThatFits) == r.overflowOpen
}

// placeChildrenHorizontally is the horizontal layout.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._placeChildrenHorizontally
func (r *renderTextSelectionToolbarItemsLayout) placeChildrenHorizontally() ui.Size {
	navButton := r.FirstChild()
	isRtl := r.textDirection == ui.TextDirectionRTL

	var contentItems []rendering.Box
	totalWidth := 0.0
	maxHeight := 0.0

	// First pass: calculate dimensions and collect items.
	i := -1
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		data := toolbarItemData(child)
		i++

		if !r.shouldPaintChild(child, i) {
			// There is no need to update children that won't be painted.
			data.ShouldPaint = false
			continue
		}
		data.ShouldPaint = true

		totalWidth += child.Size().Width
		maxHeight = math.Max(maxHeight, child.Size().Height)

		if child != navButton {
			contentItems = append(contentItems, child)
		}
	}

	// Position items based on text direction.
	currentX := 0.0
	showNavButton := r.lastIndexThatFits >= 0

	if isRtl {
		// In RTL, we want the nav button on the left and items right-aligned.
		if showNavButton {
			// (Flutter also advances currentX by the width of the button here, which
			// nothing reads afterwards.)
			toolbarItemData(navButton).Offset = ui.OffsetZero
		}

		// Position content items from right to left.
		rightEdge := totalWidth
		for _, item := range contentItems {
			rightEdge -= item.Size().Width
			toolbarItemData(item).Offset = ui.Offset{Dx: rightEdge}
		}
	} else {
		// LTR: Place content items first, then nav button. First position all
		// content items from left to right.
		for _, item := range contentItems {
			toolbarItemData(item).Offset = ui.Offset{Dx: currentX}
			currentX += item.Size().Width
		}

		// Then place the nav button at the end.
		if showNavButton {
			toolbarItemData(navButton).Offset = ui.Offset{Dx: currentX}
		}
	}

	return ui.Size{Width: totalWidth, Height: maxHeight}
}

// placeChildrenVertically is the vertical layout (overflow menu).
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._placeChildrenVertically
func (r *renderTextSelectionToolbarItemsLayout) placeChildrenVertically() ui.Size {
	navButton := r.FirstChild()

	currentY := 0.0
	maxWidth := 0.0

	navButtonData := toolbarItemData(navButton)

	if r.shouldPaintChild(navButton, 0) {
		navButtonData.ShouldPaint = true
		if !r.isAbove {
			navButtonData.Offset = ui.OffsetZero
			currentY += navButton.Size().Height
			maxWidth = math.Max(maxWidth, navButton.Size().Width)
		}
	} else {
		navButtonData.ShouldPaint = false
	}

	i := -1
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		data := toolbarItemData(child)

		i++

		// Ignore the navigation button.
		if child == navButton {
			continue
		}

		// There is no need to update children that won't be painted.
		if !r.shouldPaintChild(child, i) {
			data.ShouldPaint = false
			continue
		}

		data.ShouldPaint = true
		data.Offset = ui.Offset{Dy: currentY}
		currentY += child.Size().Height
		maxWidth = math.Max(maxWidth, child.Size().Width)
	}

	if r.isAbove && navButtonData.ShouldPaint {
		navButtonData.Offset = ui.Offset{Dy: currentY}
		currentY += navButton.Size().Height
		maxWidth = math.Max(maxWidth, navButton.Size().Width)
	}

	return ui.Size{Width: maxWidth, Height: currentY}
}

// placeChildren decides which children are painted, sets their ShouldPaint, and
// sets the offset that painted children are placed at.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._placeChildren
func (r *renderTextSelectionToolbarItemsLayout) placeChildren() {
	if r.overflowOpen {
		r.SetSize(r.placeChildrenVertically())
		return
	}
	r.SetSize(r.placeChildrenHorizontally())
}

// resizeChildrenWhenOverflow horizontally expands the children when the menu
// overflows so they can react to pointer events into their whole area.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout._resizeChildrenWhenOverflow
func (r *renderTextSelectionToolbarItemsLayout) resizeChildrenWhenOverflow() {
	if !r.overflowOpen {
		return
	}

	navButton := r.FirstChild()
	i := -1

	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		data := toolbarItemData(child)

		i++

		// Ignore the navigation button.
		if child == navButton {
			continue
		}

		// There is no need to update children that won't be painted.
		if !r.shouldPaintChild(child, i) {
			data.ShouldPaint = false
			continue
		}

		child.Layout(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(r.Size().Width)}),
			rendering.LayoutOpts{ParentUsesSize: true})
	}
}

// PerformLayout lays the children out, decides which ones are painted and places
// them.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout.performLayout
func (r *renderTextSelectionToolbarItemsLayout) PerformLayout() {
	r.lastIndexThatFits = -1
	if r.FirstChild() == nil {
		r.SetSize(r.BoxConstraints().Smallest())
		return
	}

	r.layoutChildren()
	r.placeChildren()
	r.resizeChildrenWhenOverflow()
}

// Paint paints the children that should be painted.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout.paint
func (r *renderTextSelectionToolbarItemsLayout) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		data := toolbarItemData(child)
		if !data.ShouldPaint {
			continue
		}
		context.PaintChild(child, data.Offset.Add(offset))
	}
}

// SetupParentData gives the child a ToolbarItemsParentData if it has none.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout.setupParentData
func (r *renderTextSelectionToolbarItemsLayout) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*widgets.ToolbarItemsParentData); !ok {
		child.AsRenderObject().SetParentData(&widgets.ToolbarItemsParentData{})
	}
}

// HitTestChildren hit tests the children that are shown, from the last.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout.hitTestChildren
func (r *renderTextSelectionToolbarItemsLayout) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); child != nil; child = r.ChildBefore(child) {
		// The x, y parameters have the top left of the node's box as the origin.
		data := toolbarItemData(child)

		// Don't hit test children that aren't shown.
		if !data.ShouldPaint {
			continue
		}

		child := child
		offset := data.Offset
		isHit := result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		})
		if isHit {
			return true
		}
	}
	return false
}

// VisitChildrenForSemantics visits only the children that should be painted.
//
// Flutter: material/text_selection_toolbar.dart _RenderTextSelectionToolbarItemsLayout.visitChildrenForSemantics
func (r *renderTextSelectionToolbarItemsLayout) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		if toolbarItemData(child).ShouldPaint {
			visitor(child)
		}
	}
}

// textSelectionToolbarContainer is the Material-styled toolbar outline. Fill it
// with any widgets you want. No overflow ability.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarContainer
type textSelectionToolbarContainer struct {
	widgets.StatelessWidget
	child widgets.Widget
}

var (
	// These colors were taken from a screenshot of a Pixel 6 emulator running
	// Android API level 34.
	//
	// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarContainer._defaultColorLight
	textSelectionToolbarDefaultColorLight = ui.NewColor(0xffffffff)
	// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarContainer._defaultColorDark
	textSelectionToolbarDefaultColorDark = ui.NewColor(0xff424242)
)

// textSelectionToolbarContainerColor is the color of the toolbar: white or dark
// grey, unless the scheme has a surface that is not the default one.
//
// Deviation: Flutter compares the surface to the one of ThemeData() with
// identical(), which holds for the colors of the default theme; Glutter compares
// the colors by value (SPEC §12.10).
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarContainer._getColor
func textSelectionToolbarContainerColor(colorScheme *ColorScheme) ui.Color {
	isDark := colorScheme.Brightness() == ui.BrightnessDark
	defaultScheme := colorSchemeLightM3
	if isDark {
		defaultScheme = colorSchemeDarkM3
	}
	if defaultScheme.Surface() != colorScheme.Surface() {
		return colorScheme.Surface()
	}
	if isDark {
		return textSelectionToolbarDefaultColorDark
	}
	return textSelectionToolbarDefaultColorLight
}

// Build builds a Material card with the color of the toolbar and a shape that is
// a stadium.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarContainer.build
func (w *textSelectionToolbarContainer) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	return NewMaterial(w.child, MaterialOpts{
		// This value was eyeballed to match the native text selection menu on a
		// Pixel 6 emulator running Android API level 34.
		BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(textSelectionToolbarHeight / 2)),
		ClipBehavior: ui.ClipAntiAlias,
		Color:        foundation.Some(textSelectionToolbarContainerColor(theme.ColorScheme)),
		Elevation:    1.0,
		Type:         MaterialTypeCard,
	})
}

// textSelectionToolbarOverflowButton is a button styled like a Material native
// Android text selection overflow menu forward and back controls.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowButton
type textSelectionToolbarOverflowButton struct {
	widgets.StatelessWidget
	icon      *widgets.Icon
	onPressed foundation.VoidCallback
	tooltip   string
}

func newTextSelectionToolbarOverflowButton(key foundation.Key, icon *widgets.Icon, onPressed foundation.VoidCallback, tooltip string) *textSelectionToolbarOverflowButton {
	return &textSelectionToolbarOverflowButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: key}),
		icon: icon, onPressed: onPressed, tooltip: tooltip}
}

// Build builds a transparent Material with an icon button.
//
// Flutter: material/text_selection_toolbar.dart _TextSelectionToolbarOverflowButton.build
func (w *textSelectionToolbarOverflowButton) Build(widgets.BuildContext) widgets.Widget {
	// This should be an AnimatedIcon, but AnimatedIcons does not support
	// arrow_back to more_vert (the TODO of Flutter's code).
	return NewMaterial(NewIconButton(w.onPressed, w.icon, IconButtonOpts{Tooltip: w.tooltip}), MaterialOpts{
		Type:  MaterialTypeCard,
		Color: foundation.Some(ui.NewColor(0x00000000)),
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextSelectionToolbarNoChildren = "TextSelectionToolbarNoChildren"
)
