package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The measures of the toolbar of iOS (the constants of cupertino/text_selection_toolbar.dart).
const (
	toolbarContentDistance = 8.0
	toolbarChevronSize     = 10.0
	toolbarChevronWidth    = 2.0

	toolbarTransitionDuration = 125 * millisecond

	// CupertinoTextSelectionToolbarScreenPadding is the minimum padding between the toolbar and the sides of the
	// screen.
	//
	// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.kToolbarScreenPadding
	CupertinoTextSelectionToolbarScreenPadding = 8.0
)

var (
	toolbarBorderRadius     = ui.NewRadiusCircular(8.0)
	toolbarArrowSize        = ui.Size{Width: 14.0, Height: 7.0}
	toolbarBackgroundColor  = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFF6F6F6), ui.NewColor(0xFF222222))
	toolbarDividerColor     = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFFD6D6D6), ui.NewColor(0xFF424242))
	toolbarChevronTextColor = NewCupertinoDynamicColorWithBrightness(CupertinoColors.Black, CupertinoColors.White)
)

// CupertinoToolbarBuilder builds the toolbar around the items: the default one puts a shape with an arrow, a shadow
// and a background around them.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoToolbarBuilder
type CupertinoToolbarBuilder func(bc widgets.BuildContext, anchorAbove, anchorBelow ui.Offset, child widgets.Widget) widgets.Widget

// CupertinoTextSelectionToolbarOpts are the optional parameters of NewCupertinoTextSelectionToolbar.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.new
type CupertinoTextSelectionToolbarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ToolbarBuilder builds the toolbar around the items; the default is the toolbar of iOS.
	ToolbarBuilder CupertinoToolbarBuilder
}

// CupertinoTextSelectionToolbar is the toolbar of the text selection of iOS: a row of buttons with an arrow that
// points at the selection, and arrows that page through the buttons that do not fit.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar
type CupertinoTextSelectionToolbar struct {
	widgets.StatelessWidget
	anchorAbove, anchorBelow ui.Offset
	children                 []widgets.Widget
	toolbarBuilder           CupertinoToolbarBuilder
}

// NewCupertinoTextSelectionToolbar creates a toolbar of the buttons; it is placed above the anchorAbove if it
// fits, and below the anchorBelow if not. In debug builds it panics with the tag
// CupertinoTextSelectionToolbarNoChildren when there are none.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.new
func NewCupertinoTextSelectionToolbar(anchorAbove, anchorBelow ui.Offset, children []widgets.Widget, opts ...CupertinoTextSelectionToolbarOpts) *CupertinoTextSelectionToolbar {
	o := firstOf(opts)
	if foundation.DebugMode && len(children) == 0 {
		panic(errors.New(tagCupertinoTextSelectionToolbarNoChildren, "a CupertinoTextSelectionToolbar needs at least one child"))
	}
	builder := o.ToolbarBuilder
	if builder == nil {
		builder = cupertinoToolbarDefaultBuilder
	}
	return &CupertinoTextSelectionToolbar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		anchorAbove: anchorAbove, anchorBelow: anchorBelow, children: children, toolbarBuilder: builder}
}

// AnchorAbove is the point above which the toolbar is placed, when it fits.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.anchorAbove
func (w *CupertinoTextSelectionToolbar) AnchorAbove() ui.Offset { return w.anchorAbove }

// AnchorBelow is the point below which the toolbar is placed, when it does not fit above.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.anchorBelow
func (w *CupertinoTextSelectionToolbar) AnchorBelow() ui.Offset { return w.anchorBelow }

// Children are the buttons.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.children
func (w *CupertinoTextSelectionToolbar) Children() []widgets.Widget { return w.children }

// cupertinoToolbarDefaultBuilder puts the child in the shape of the toolbar, on its background.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar._defaultToolbarBuilder
func cupertinoToolbarDefaultBuilder(bc widgets.BuildContext, anchorAbove, anchorBelow ui.Offset, child widgets.Widget) widgets.Widget {
	var shadow foundation.Opt[ui.Color]
	if CupertinoThemeBrightnessOf(bc) == ui.BrightnessLight {
		shadow = foundation.Some(CupertinoColors.Black.WithOpacity(0.2))
	}
	return &cupertinoTextSelectionToolbarShape{
		SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(
			widgets.NewColoredBox(CupertinoDynamicColorResolve(toolbarBackgroundColor, bc), child)),
		anchorAbove: anchorAbove, anchorBelow: anchorBelow, shadowColor: shadow,
	}
}

// Build places the content above or below the anchors, inside the padding of the screen.
//
// Flutter: cupertino/text_selection_toolbar.dart CupertinoTextSelectionToolbar.build
func (w *CupertinoTextSelectionToolbar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	mediaQueryPadding := widgets.MediaQueryPaddingOf(bc)
	paddingAbove := mediaQueryPadding.Top + CupertinoTextSelectionToolbarScreenPadding
	leftMargin := arrowScreenPadding + mediaQueryPadding.Left
	rightMargin := widgets.MediaQueryWidthOf(bc) - mediaQueryPadding.Right - arrowScreenPadding
	anchorAboveAdjusted := ui.Offset{Dx: ui.ClampDouble(w.anchorAbove.Dx, leftMargin, rightMargin), Dy: w.anchorAbove.Dy - toolbarContentDistance - paddingAbove}
	anchorBelowAdjusted := ui.Offset{Dx: ui.ClampDouble(w.anchorBelow.Dx, leftMargin, rightMargin), Dy: w.anchorBelow.Dy + toolbarContentDistance - paddingAbove}
	return widgets.NewPadding(
		painting.NewEdgeInsetsFromLTRB(CupertinoTextSelectionToolbarScreenPadding, paddingAbove, CupertinoTextSelectionToolbarScreenPadding, CupertinoTextSelectionToolbarScreenPadding),
		widgets.NewCustomSingleChildLayout(
			widgets.NewTextSelectionToolbarLayoutDelegate(anchorAboveAdjusted, anchorBelowAdjusted),
			&cupertinoTextSelectionToolbarContent{
				StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}),
				anchorAbove:    anchorAboveAdjusted, anchorBelow: anchorBelowAdjusted, toolbarBuilder: w.toolbarBuilder, children: w.children}))
}

// cupertinoTextSelectionToolbarShape is the widget of the shape of the toolbar.
//
// Flutter: cupertino/text_selection_toolbar.dart _CupertinoTextSelectionToolbarShape
type cupertinoTextSelectionToolbarShape struct {
	widgets.SingleChildRenderObjectWidget
	anchorAbove, anchorBelow ui.Offset
	shadowColor              foundation.Opt[ui.Color]
}

func (w *cupertinoTextSelectionToolbarShape) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderCupertinoTextSelectionToolbarShape{anchorAbove: w.anchorAbove, anchorBelow: w.anchorBelow, shadowColor: w.shadowColor,
		clipPathLayer: rendering.NewLayerHandle[*rendering.ClipPathLayer](nil)}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (w *cupertinoTextSelectionToolbarShape) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderCupertinoTextSelectionToolbarShape)
	r.setAnchorAbove(w.anchorAbove)
	r.setAnchorBelow(w.anchorBelow)
	r.setShadowColor(w.shadowColor)
}

// renderCupertinoTextSelectionToolbarShape clips its child to a rounded rectangle with an arrow at the anchor, and
// draws the shadow behind it.
//
// Flutter: cupertino/text_selection_toolbar.dart _RenderCupertinoTextSelectionToolbarShape
type renderCupertinoTextSelectionToolbarShape struct {
	rendering.RenderShiftedBox
	anchorAbove, anchorBelow ui.Offset
	shadowColor              foundation.Opt[ui.Color]
	clipPathLayer            *rendering.LayerHandle[*rendering.ClipPathLayer]
}

func (r *renderCupertinoTextSelectionToolbarShape) IsRepaintBoundary() bool { return true }

func (r *renderCupertinoTextSelectionToolbarShape) setAnchorAbove(v ui.Offset) {
	if v == r.anchorAbove {
		return
	}
	r.anchorAbove = v
	r.MarkNeedsLayout()
}

func (r *renderCupertinoTextSelectionToolbarShape) setAnchorBelow(v ui.Offset) {
	if v == r.anchorBelow {
		return
	}
	r.anchorBelow = v
	r.MarkNeedsLayout()
}

func (r *renderCupertinoTextSelectionToolbarShape) setShadowColor(v foundation.Opt[ui.Color]) {
	if v == r.shadowColor {
		return
	}
	r.shadowColor = v
	r.MarkNeedsPaint()
}

func (r *renderCupertinoTextSelectionToolbarShape) isAbove(childHeight float64) bool {
	return r.anchorAbove.Dy >= childHeight-toolbarArrowSize.Height
}

func (r *renderCupertinoTextSelectionToolbarShape) constraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return rendering.BoxConstraints{MinWidth: toolbarArrowSize.Width + toolbarBorderRadius.X*2, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}.Enforce(constraints.Loosen())
}

func (r *renderCupertinoTextSelectionToolbarShape) computeChildOffset(childSize ui.Size) ui.Offset {
	if r.isAbove(childSize.Height) {
		return ui.Offset{Dy: -toolbarArrowSize.Height}
	}
	return ui.Offset{}
}

func (r *renderCupertinoTextSelectionToolbarShape) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	enforced := r.constraintsForChild(constraints)
	result, ok := child.GetDryBaseline(enforced, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(result + r.computeChildOffset(child.GetDryLayout(enforced)).Dy)
}

func (r *renderCupertinoTextSelectionToolbarShape) PerformLayout() {
	child := r.Child()
	if child == nil {
		return
	}
	child.Layout(r.constraintsForChild(r.BoxConstraints()), rendering.LayoutOpts{ParentUsesSize: true})
	rendering.BoxParentDataOf(child).Offset = r.computeChildOffset(child.Size())
	r.SetSize(ui.Size{Width: child.Size().Width, Height: child.Size().Height - toolbarArrowSize.Height})
}

func (r *renderCupertinoTextSelectionToolbarShape) shapeRRect(child rendering.Box) ui.RRect {
	rect := ui.NewRectFromLTWH(0.0, toolbarArrowSize.Height, child.Size().Width, child.Size().Height-toolbarArrowSize.Height*2)
	return ui.NewRRectFromRectAndRadius(rect, toolbarBorderRadius).ScaleRadii()
}

// addRRectToPath adds the arcs of the corners of the rounded rectangle to the path, from the corner of the start
// angle (a multiple of a quarter turn).
//
// Flutter: cupertino/text_selection_toolbar.dart _RenderCupertinoTextSelectionToolbarShape._addRRectToPath
func addRRectToPath(path *ui.Path, rrect ui.RRect, startAngle float64) *ui.Path {
	const halfPI = math.Pi / 2
	rect := rrect.OuterRect()
	type corner struct {
		vertex ui.Offset
		radius ui.Radius
	}
	corners := []corner{
		{rect.BottomRight(), rrect.BrRadius().Neg()},
		{rect.BottomLeft(), ui.NewRadiusElliptical(rrect.BlRadius().X, -rrect.BlRadius().Y)},
		{rect.TopLeft(), rrect.TlRadius()},
		{rect.TopRight(), ui.NewRadiusElliptical(-rrect.TrRadius().X, rrect.TrRadius().Y)},
	}
	start := int(startAngle / halfPI)
	for i := start; i < len(corners)+start; i++ {
		c := corners[((i%len(corners))+len(corners))%len(corners)]
		other := ui.Offset{Dx: c.vertex.Dx + 2*c.radius.X, Dy: c.vertex.Dy + 2*c.radius.Y}
		path.ArcTo(ui.NewRectFromPoints(c.vertex, other), halfPI*float64(i), halfPI, false)
	}
	return path
}

func (r *renderCupertinoTextSelectionToolbarShape) clipPath(child rendering.Box, rrect ui.RRect) *ui.Path {
	path := ui.NewPath()
	size := r.Size()
	if toolbarBorderRadius.X*2+toolbarArrowSize.Width > size.Width {
		path.AddRRect(rrect)
		return path
	}
	isAbove := r.isAbove(child.Size().Height)
	anchor := r.anchorBelow
	if isAbove {
		anchor = r.anchorAbove
	}
	localAnchor := r.GlobalToLocal(anchor, nil)
	arrowTipX := ui.ClampDouble(localAnchor.Dx, toolbarBorderRadius.X+toolbarArrowSize.Width/2, size.Width-toolbarArrowSize.Width/2-toolbarBorderRadius.X)
	if isAbove {
		arrowBaseY := child.Size().Height - toolbarArrowSize.Height
		arrowTipY := child.Size().Height
		path.MoveTo(arrowTipX+toolbarArrowSize.Width/2, arrowBaseY) // right side of the arrow triangle
		path.LineTo(arrowTipX, arrowTipY)                           // the tip of the arrow
		path.LineTo(arrowTipX-toolbarArrowSize.Width/2, arrowBaseY) // left side of the arrow triangle
	} else {
		arrowBaseY := toolbarArrowSize.Height
		path.MoveTo(arrowTipX-toolbarArrowSize.Width/2, arrowBaseY) // right side of the arrow triangle
		path.LineTo(arrowTipX, 0.0)                                 // the tip of the arrow
		path.LineTo(arrowTipX+toolbarArrowSize.Width/2, arrowBaseY) // left side of the arrow triangle
	}
	startAngle := -math.Pi / 2
	if isAbove {
		startAngle = math.Pi / 2
	}
	addRRectToPath(path, rrect, startAngle)
	path.Close()
	return path
}

func (r *renderCupertinoTextSelectionToolbarShape) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	child := r.Child()
	if child == nil {
		return
	}
	childOffset := rendering.BoxParentDataOf(child).Offset
	rrect := r.shapeRRect(child)
	clipPath := r.clipPath(child, rrect)
	if color, ok := r.shadowColor.Get(); ok {
		boxShadow := painting.BoxShadow{Shadow: ui.NewShadow(ui.ShadowOpts{Color: &color, BlurRadius: 15.0})}
		shadowRRect := ui.NewRRectFromLTRBR(rrect.Left, rrect.Top, rrect.Right, rrect.Bottom+toolbarArrowSize.Height, toolbarBorderRadius).
			Shift(offset.Add(childOffset).Add(boxShadow.Offset))
		context.Canvas().DrawRRect(shadowRRect, boxShadow.ToPaint())
	}
	r.clipPathLayer.SetLayer(context.PushClipPath(r.NeedsCompositing(), offset.Add(childOffset), ui.OffsetZero.And(child.Size()), clipPath,
		func(inner *rendering.PaintingContext, innerOffset ui.Offset) { inner.PaintChild(child, innerOffset) },
		rendering.PushClipPathOpts{OldLayer: r.clipPathLayer.Layer()}))
}

func (r *renderCupertinoTextSelectionToolbarShape) Dispose() {
	r.clipPathLayer.SetLayer(nil)
	r.RenderShiftedBox.Dispose()
}

func (r *renderCupertinoTextSelectionToolbarShape) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	child := r.Child()
	if child == nil {
		return false
	}
	childOffset := rendering.BoxParentDataOf(child).Offset
	hitBox := ui.NewRectFromLTWH(childOffset.Dx, childOffset.Dy+toolbarArrowSize.Height, child.Size().Width, child.Size().Height-toolbarArrowSize.Height*2)
	if !hitBox.Contains(position) {
		return false
	}
	return r.RenderShiftedBox.HitTestChildren(result, position)
}

// cupertinoTextSelectionToolbarContent is the state of the pages of the toolbar: the page that shows, and the fade
// between two pages.
//
// Flutter: cupertino/text_selection_toolbar.dart _CupertinoTextSelectionToolbarContent
type cupertinoTextSelectionToolbarContent struct {
	widgets.StatefulWidget
	anchorAbove, anchorBelow ui.Offset
	children                 []widgets.Widget
	toolbarBuilder           CupertinoToolbarBuilder
}

func (w *cupertinoTextSelectionToolbarContent) CreateState() widgets.State {
	return &cupertinoTextSelectionToolbarContentState{toolbarItemsKey: widgets.NewGlobalKey[widgets.State](), nextPage: foundation.None[int]()}
}

type cupertinoTextSelectionToolbarContentState struct {
	widgets.StateOf[*cupertinoTextSelectionToolbarContent]
	widgets.TickerProvider
	controller      *animation.AnimationController
	statusHandle    animation.StatusListenerHandle
	listening       bool
	nextPage        foundation.Opt[int]
	page            int
	toolbarItemsKey *widgets.GlobalKey[widgets.State]
}

func (s *cupertinoTextSelectionToolbarContentState) onHorizontalDragEnd(details gestures.DragEndDetails) {
	if velocity, ok := details.PrimaryVelocity.Get(); ok && velocity != 0 {
		if velocity > 0 {
			s.handlePreviousPage()
		} else {
			s.handleNextPage()
		}
	}
}

func (s *cupertinoTextSelectionToolbarContentState) itemsRender() *renderCupertinoTextSelectionToolbarItems {
	e := s.toolbarItemsKey.CurrentElement()
	if e == nil {
		return nil
	}
	r, _ := e.(widgets.BuildContext).FindRenderObject().(*renderCupertinoTextSelectionToolbarItems)
	return r
}

func (s *cupertinoTextSelectionToolbarContentState) startPageChange(next int) {
	s.controller.Reverse()
	s.addStatusListener()
	s.nextPage = foundation.Some(next)
}

func (s *cupertinoTextSelectionToolbarContentState) handleNextPage() {
	if r := s.itemsRender(); r != nil && r.hasNextPage {
		s.startPageChange(s.page + 1)
	}
}

func (s *cupertinoTextSelectionToolbarContentState) handlePreviousPage() {
	if r := s.itemsRender(); r != nil && r.hasPreviousPage {
		s.startPageChange(s.page - 1)
	}
}

func (s *cupertinoTextSelectionToolbarContentState) addStatusListener() {
	s.removeStatusListener()
	s.statusHandle = s.controller.AddStatusListener(s.statusListener)
	s.listening = true
}

func (s *cupertinoTextSelectionToolbarContentState) removeStatusListener() {
	if s.listening {
		s.controller.RemoveStatusListener(s.statusHandle)
		s.listening = false
	}
}

func (s *cupertinoTextSelectionToolbarContentState) statusListener(status animation.AnimationStatus) {
	if status != animation.AnimationStatusDismissed {
		return
	}
	s.SetState(func() {
		s.page = s.nextPage.Value()
		s.nextPage = foundation.None[int]()
	})
	s.controller.Forward()
	s.removeStatusListener()
}

func (s *cupertinoTextSelectionToolbarContentState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Value: foundation.Some(1.0), Duration: foundation.Some(toolbarTransitionDuration)})
}

func (s *cupertinoTextSelectionToolbarContentState) DidUpdateWidget(old *cupertinoTextSelectionToolbarContent) {
	if !cupertinoWidgetListEquals(s.Widget().children, old.children) {
		// If the children are changing at all, the current page should be reset.
		s.page = 0
		s.nextPage = foundation.None[int]()
		s.controller.Forward()
		s.removeStatusListener()
	}
}

// cupertinoWidgetListEquals is whether the two lists have the same widgets, the same instances, in the same order.
//
// Flutter: foundation/collections.dart listEquals
func cupertinoWidgetListEquals(a, b []widgets.Widget) bool {
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

func (s *cupertinoTextSelectionToolbarContentState) Dispose() {
	s.removeStatusListener()
	s.controller.Dispose()
}

func (s *cupertinoTextSelectionToolbarContentState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	chevronColor := CupertinoDynamicColorResolve(toolbarChevronTextColor, bc)
	chevron := func(onPressed func(), isLeft bool) widgets.Widget {
		return widgets.NewCenter(NewCupertinoTextSelectionToolbarButton(widgets.NewIgnorePointer(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{
			Painter: &cupertinoChevronPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), color: chevronColor, isLeft: isLeft},
			Size:    ui.Size{Width: toolbarChevronSize, Height: toolbarChevronSize}})),
			CupertinoTextSelectionToolbarButtonOpts{OnPressed: onPressed}),
			widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)})
	}
	children := make([]widgets.Widget, len(w.children))
	for i, child := range w.children {
		children[i] = widgets.NewCenter(child, widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)})
	}
	return w.toolbarBuilder(bc, w.anchorAbove, w.anchorBelow,
		widgets.NewFadeTransition(s.controller,
			widgets.NewAnimatedSize(toolbarTransitionDuration,
				widgets.NewGestureDetector(&cupertinoTextSelectionToolbarItems{
					MultiChildRenderObjectWidget: widgets.NewMultiChildRenderObjectWidgetBase(
						append([]widgets.Widget{chevron(s.handlePreviousPage, true), chevron(s.handleNextPage, false)}, children...), widgets.WidgetOpts{Key: s.toolbarItemsKey}),
					page: s.page, dividerColor: CupertinoDynamicColorResolve(toolbarDividerColor, bc), dividerWidth: 1.0 / widgets.MediaQueryDevicePixelRatioOf(bc),
				}, widgets.GestureDetectorOpts{OnHorizontalDragEnd: s.onHorizontalDragEnd}),
				widgets.AnimatedSizeOpts{Curve: animation.Curves.Decelerate})))
}

// cupertinoChevronPainter paints the chevron of a button of the toolbar, pointing left or right.
//
// Flutter: cupertino/text_selection_toolbar.dart _CupertinoChevronPainter
type cupertinoChevronPainter struct {
	rendering.CustomPainterBase
	color  ui.Color
	isLeft bool
}

func (p *cupertinoChevronPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	iconSize := size.Height
	sign := -1.0
	if p.isLeft {
		sign = 1
	}
	centerOffset := ui.Offset{Dx: iconSize / 4 * sign}
	firstPoint := ui.Offset{Dx: iconSize / 2}.Add(centerOffset)
	middleX := iconSize
	if p.isLeft {
		middleX = 0
	}
	middlePoint := ui.Offset{Dx: middleX, Dy: iconSize / 2}.Add(centerOffset)
	lowerPoint := ui.Offset{Dx: iconSize / 2, Dy: iconSize}.Add(centerOffset)
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(toolbarChevronWidth)
	paint.SetStrokeCap(ui.StrokeCapRound)
	paint.SetStrokeJoin(ui.StrokeJoinRound)
	canvas.DrawLine(firstPoint, middlePoint, paint)
	canvas.DrawLine(middlePoint, lowerPoint, paint)
}

func (p *cupertinoChevronPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*cupertinoChevronPainter)
	return !ok || o.color != p.color || o.isLeft != p.isLeft
}

func (p *cupertinoChevronPainter) HitTest(ui.Offset) foundation.Opt[bool] {
	return foundation.None[bool]()
}

// cupertinoTextSelectionToolbarItems is the widget of the layout of the buttons in pages. Its first two children are
// the back and the next buttons, and the others are the items.
//
// Deviation: Flutter has the two buttons in slots of a render object element of its own, apart from the list of
// the items; here they are the first two children of the list, which has the same order of visits, so the layout and
// the paint are the same (SPEC §12.10).
//
// Flutter: cupertino/text_selection_toolbar.dart _CupertinoTextSelectionToolbarItems
type cupertinoTextSelectionToolbarItems struct {
	widgets.MultiChildRenderObjectWidget
	page         int
	dividerColor ui.Color
	dividerWidth float64
}

func (w *cupertinoTextSelectionToolbarItems) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	r := &renderCupertinoTextSelectionToolbarItems{dividerColor: w.dividerColor, dividerWidth: w.dividerWidth, page: w.page}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitContainerRenderObject(&r.ContainerRenderObject, r)
	return r
}

func (w *cupertinoTextSelectionToolbarItems) UpdateRenderObject(_ widgets.BuildContext, ro rendering.RenderObject) {
	r := ro.(*renderCupertinoTextSelectionToolbarItems)
	r.setPage(w.page)
	r.setDividerColor(w.dividerColor)
	r.setDividerWidth(w.dividerWidth)
}

// CreateElement creates the element that tells which children are on stage.
func (w *cupertinoTextSelectionToolbarItems) CreateElement() widgets.Element {
	e := &cupertinoTextSelectionToolbarItemsElement{}
	widgets.InitMultiChildRenderObjectElement(&e.MultiChildRenderObjectElement, e, w)
	return e
}

// cupertinoTextSelectionToolbarItemsElement is the element of the items: only the children that are painted are on
// stage.
//
// Flutter: cupertino/text_selection_toolbar.dart _CupertinoTextSelectionToolbarItemsElement
type cupertinoTextSelectionToolbarItemsElement struct {
	widgets.MultiChildRenderObjectElement
}

func (e *cupertinoTextSelectionToolbarItemsElement) DebugVisitOnstageChildren(visitor widgets.ElementVisitor) {
	for _, child := range e.Children() {
		if toolbarItemData(child.RenderObject().(rendering.Box)).ShouldPaint {
			visitor(child)
		}
	}
}

func toolbarItemData(child rendering.Box) *widgets.ToolbarItemsParentData {
	return child.ParentData().(*widgets.ToolbarItemsParentData)
}

// renderCupertinoTextSelectionToolbarItems lays the buttons out in a row, in pages that fit in the width, with the
// back and next buttons around them.
//
// Flutter: cupertino/text_selection_toolbar.dart _RenderCupertinoTextSelectionToolbarItems
type renderCupertinoTextSelectionToolbarItems struct {
	rendering.RenderBox
	rendering.ContainerRenderObject[rendering.Box]
	hasNextPage, hasPreviousPage bool
	page                         int
	dividerColor                 ui.Color
	dividerWidth                 float64
}

func (r *renderCupertinoTextSelectionToolbarItems) setPage(v int) {
	if v == r.page {
		return
	}
	r.page = v
	r.MarkNeedsLayout()
}

func (r *renderCupertinoTextSelectionToolbarItems) setDividerColor(v ui.Color) {
	if v == r.dividerColor {
		return
	}
	r.dividerColor = v
	r.MarkNeedsLayout()
}

func (r *renderCupertinoTextSelectionToolbarItems) setDividerWidth(v float64) {
	if v == r.dividerWidth {
		return
	}
	r.dividerWidth = v
	r.MarkNeedsLayout()
}

func (r *renderCupertinoTextSelectionToolbarItems) backButton() rendering.Box { return r.FirstChild() }
func (r *renderCupertinoTextSelectionToolbarItems) nextButton() rendering.Box {
	return r.ChildAfter(r.FirstChild())
}

func (r *renderCupertinoTextSelectionToolbarItems) PerformLayout() {
	constraints := r.BoxConstraints()
	if r.FirstChild() == nil {
		r.SetSize(constraints.Smallest())
		return
	}
	greatestHeight := 0.0
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		greatestHeight = math.Max(greatestHeight, child.GetMaxIntrinsicHeight(constraints.MaxWidth))
	}
	back, next := r.backButton(), r.nextButton()
	slotted := rendering.BoxConstraints{MaxWidth: constraints.MaxWidth, MinHeight: greatestHeight, MaxHeight: greatestHeight}
	back.Layout(slotted, rendering.LayoutOpts{ParentUsesSize: true})
	next.Layout(slotted, rendering.LayoutOpts{ParentUsesSize: true})
	subsequentPageButtonsWidth := back.Size().Width + next.Size().Width

	currentButtonPosition := 0.0
	toolbarWidth := 0.0 // The width of the whole widget.
	currentPage := 0
	i := -1
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		i++
		pd := toolbarItemData(child)
		pd.ShouldPaint = false
		if child == back || child == next || currentPage > r.page {
			continue
		}
		var paginationButtonsWidth float64
		switch {
		case currentPage != 0:
			paginationButtonsWidth = subsequentPageButtonsWidth
		case i == r.ChildCount()-1:
			paginationButtonsWidth = 0.0
		default:
			paginationButtonsWidth = next.Size().Width
		}
		layoutItem := func() {
			child.Layout(rendering.BoxConstraints{MaxWidth: constraints.MaxWidth - paginationButtonsWidth, MinHeight: greatestHeight, MaxHeight: greatestHeight},
				rendering.LayoutOpts{ParentUsesSize: true})
		}
		layoutItem()
		// If this child causes the current page to overflow, move to the next page and relayout the child.
		currentWidth := currentButtonPosition + paginationButtonsWidth + child.Size().Width
		if currentWidth > constraints.MaxWidth {
			currentPage++
			currentButtonPosition = back.Size().Width + r.dividerWidth
			paginationButtonsWidth = back.Size().Width + next.Size().Width
			layoutItem()
		}
		pd.Offset = ui.Offset{Dx: currentButtonPosition}
		currentButtonPosition += child.Size().Width + r.dividerWidth
		pd.ShouldPaint = currentPage == r.page
		if currentPage == r.page {
			toolbarWidth = currentButtonPosition
		}
	}

	// It shouldn't be possible to navigate beyond the last page.
	if currentPage > 0 {
		nextData, backData := toolbarItemData(next), toolbarItemData(back)
		if r.page != currentPage {
			nextData.Offset = ui.Offset{Dx: toolbarWidth}
			nextData.ShouldPaint = true
			toolbarWidth += next.Size().Width
		}
		if r.page > 0 {
			backData.Offset = ui.Offset{}
			backData.ShouldPaint = true
		}
	} else {
		// Remove the trailing divider of the only page.
		toolbarWidth -= r.dividerWidth
	}
	r.hasNextPage = r.page != currentPage
	r.hasPreviousPage = r.page > 0
	r.SetSize(constraints.Constrain(ui.Size{Width: toolbarWidth, Height: greatestHeight}))
}

func (r *renderCupertinoTextSelectionToolbarItems) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	back, next := r.backButton(), r.nextButton()
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		pd := toolbarItemData(child)
		if !pd.ShouldPaint {
			continue
		}
		childOffset := pd.Offset.Add(offset)
		context.PaintChild(child, childOffset)
		// Draw the divider when it is not the last child of the page.
		if child != next && (pd.NextSibling != nil || child == back) {
			paint := ui.NewPaint()
			paint.SetColor(r.dividerColor)
			context.Canvas().DrawLine(ui.Offset{Dx: child.Size().Width}.Add(childOffset), ui.Offset{Dx: child.Size().Width, Dy: child.Size().Height}.Add(childOffset), paint)
		}
	}
}

func (r *renderCupertinoTextSelectionToolbarItems) SetupParentData(child rendering.RenderObject) {
	if _, ok := child.ParentData().(*widgets.ToolbarItemsParentData); !ok {
		child.AsRenderObject().SetParentData(&widgets.ToolbarItemsParentData{})
	}
}

func (r *renderCupertinoTextSelectionToolbarItems) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	for child := r.LastChild(); child != nil; child = r.ChildBefore(child) {
		pd := toolbarItemData(child)
		if !pd.ShouldPaint {
			continue
		}
		child := child
		offset := pd.Offset
		if result.AddWithPaintOffset(&offset, position, func(result *rendering.BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}

func (r *renderCupertinoTextSelectionToolbarItems) VisitChildrenForSemantics(visitor rendering.RenderObjectVisitor) {
	for child := r.FirstChild(); child != nil; child = r.ChildAfter(child) {
		if toolbarItemData(child).ShouldPaint {
			visitor(child)
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagCupertinoTextSelectionToolbarNoChildren = "CupertinoTextSelectionToolbarNoChildren"
