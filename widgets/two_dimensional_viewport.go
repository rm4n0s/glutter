package widgets

import (
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// ChildVicinity is the location of a child in the two-dimensional space of a
// two-dimensional viewport: the index along each axis.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity
type ChildVicinity = rendering.ChildVicinity

// ChildVicinityInvalid is the vicinity of no child.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.invalid
var ChildVicinityInvalid = rendering.ChildVicinityInvalid

// NewChildVicinity creates a vicinity.
//
// Flutter: widgets/two_dimensional_viewport.dart ChildVicinity.new
func NewChildVicinity(xIndex, yIndex int) ChildVicinity {
	return rendering.NewChildVicinity(xIndex, yIndex)
}

// TwoDimensionalViewportParentData is the parent data of the children of a
// two-dimensional viewport.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewportParentData
type TwoDimensionalViewportParentData = rendering.TwoDimensionalViewportParentData

// TwoDimensionalIndexedWidgetBuilder builds the widget at the vicinity, or
// returns nil if there is none.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalIndexedWidgetBuilder
type TwoDimensionalIndexedWidgetBuilder func(bc BuildContext, vicinity ChildVicinity) Widget

// TwoDimensionalChildDelegate supplies the children of a two-dimensional
// viewport. It notifies its listeners when the children change.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildDelegate
type TwoDimensionalChildDelegate interface {
	rendering.TwoDimensionalChildDelegate
	// Build builds the widget at the vicinity, or returns nil if there is none.
	//
	// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildDelegate.build
	Build(bc BuildContext, vicinity ChildVicinity) Widget
}

// TwoDimensionalChildBuilderDelegateOpts are the optional parameters of
// NewTwoDimensionalChildBuilderDelegate.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.new
type TwoDimensionalChildBuilderDelegateOpts struct {
	// MaxXIndex is the greatest x index of a child; none means unbounded.
	MaxXIndex foundation.Opt[int]
	// MaxYIndex is the greatest y index of a child; none means unbounded.
	MaxYIndex foundation.Opt[int]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary;
	// the default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddAutomaticKeepAlives is whether to wrap each child in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
}

// TwoDimensionalChildBuilderDelegate supplies the children of a two-dimensional
// viewport lazily, from a builder, up to optional greatest indices.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate
type TwoDimensionalChildBuilderDelegate struct {
	foundation.ChangeNotifier
	builder                TwoDimensionalIndexedWidgetBuilder
	maxXIndex, maxYIndex   foundation.Opt[int]
	addRepaintBoundaries   bool
	addAutomaticKeepAlives bool
}

// NewTwoDimensionalChildBuilderDelegate creates a delegate. In debug builds it
// panics with the tag TwoDimensionalIndexInvalid when a maximum index is less
// than -1.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.new
func NewTwoDimensionalChildBuilderDelegate(builder TwoDimensionalIndexedWidgetBuilder, opts ...TwoDimensionalChildBuilderDelegateOpts) *TwoDimensionalChildBuilderDelegate {
	o := firstOpts(opts)
	checkMaxIndex(o.MaxXIndex)
	checkMaxIndex(o.MaxYIndex)
	return &TwoDimensionalChildBuilderDelegate{builder: builder, maxXIndex: o.MaxXIndex, maxYIndex: o.MaxYIndex,
		addRepaintBoundaries: o.AddRepaintBoundaries.OrElse(true), addAutomaticKeepAlives: o.AddAutomaticKeepAlives.OrElse(true)}
}

func checkMaxIndex(v foundation.Opt[int]) {
	if i, ok := v.Get(); ok && i < -1 && foundation.DebugMode {
		panic(errors.New("TwoDimensionalIndexInvalid", "the maximum index must not be less than -1", "index", i))
	}
}

// MaxXIndex is the greatest x index of a child, or none.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.maxXIndex
func (d *TwoDimensionalChildBuilderDelegate) MaxXIndex() foundation.Opt[int] { return d.maxXIndex }

// SetMaxXIndex changes the greatest x index and notifies the listeners.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.maxXIndex
func (d *TwoDimensionalChildBuilderDelegate) SetMaxXIndex(value foundation.Opt[int]) {
	if value == d.maxXIndex {
		return
	}
	checkMaxIndex(value)
	d.maxXIndex = value
	d.NotifyListeners()
}

// MaxYIndex is the greatest y index of a child, or none.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.maxYIndex
func (d *TwoDimensionalChildBuilderDelegate) MaxYIndex() foundation.Opt[int] { return d.maxYIndex }

// SetMaxYIndex changes the greatest y index and notifies the listeners.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.maxYIndex
func (d *TwoDimensionalChildBuilderDelegate) SetMaxYIndex(value foundation.Opt[int]) {
	if value == d.maxYIndex {
		return
	}
	checkMaxIndex(value)
	d.maxYIndex = value
	d.NotifyListeners()
}

// Build builds the child at the vicinity, or returns nil beyond the greatest
// indices or where the builder has none.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.build
func (d *TwoDimensionalChildBuilderDelegate) Build(bc BuildContext, vicinity ChildVicinity) Widget {
	// If an explicit upper bound has been exceeded, there is no child.
	if vicinity.XIndex < 0 || (d.maxXIndex.IsSome() && vicinity.XIndex > d.maxXIndex.Value()) {
		return nil
	}
	if vicinity.YIndex < 0 || (d.maxYIndex.IsSome() && vicinity.YIndex > d.maxYIndex.Value()) {
		return nil
	}
	child := guardedBuild(bc, func() Widget { return d.builder(bc, vicinity) })
	if child == nil {
		return nil
	}
	return wrapTwoDimensionalChild(child, d.addRepaintBoundaries, d.addAutomaticKeepAlives)
}

// ShouldRebuild is true.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildBuilderDelegate.shouldRebuild
func (d *TwoDimensionalChildBuilderDelegate) ShouldRebuild(rendering.TwoDimensionalChildDelegate) bool {
	return true
}

func wrapTwoDimensionalChild(child Widget, addRepaintBoundaries, addAutomaticKeepAlives bool) Widget {
	if addRepaintBoundaries {
		child = NewRepaintBoundary(child)
	}
	if addAutomaticKeepAlives {
		child = NewAutomaticKeepAlive(newSelectionKeepAlive(child))
	}
	return child
}

// TwoDimensionalChildListDelegateOpts are the optional parameters of
// NewTwoDimensionalChildListDelegate.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate.new
type TwoDimensionalChildListDelegateOpts struct {
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary;
	// the default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddAutomaticKeepAlives is whether to wrap each child in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
}

// TwoDimensionalChildListDelegate supplies the children of a two-dimensional
// viewport from a list of rows (the first index is y, the second one x).
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate
type TwoDimensionalChildListDelegate struct {
	foundation.ChangeNotifier
	children               [][]Widget
	addRepaintBoundaries   bool
	addAutomaticKeepAlives bool
}

// NewTwoDimensionalChildListDelegate creates a delegate of the rows of children.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate.new
func NewTwoDimensionalChildListDelegate(children [][]Widget, opts ...TwoDimensionalChildListDelegateOpts) *TwoDimensionalChildListDelegate {
	o := firstOpts(opts)
	return &TwoDimensionalChildListDelegate{children: children,
		addRepaintBoundaries: o.AddRepaintBoundaries.OrElse(true), addAutomaticKeepAlives: o.AddAutomaticKeepAlives.OrElse(true)}
}

// Children are the rows of children.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate.children
func (d *TwoDimensionalChildListDelegate) Children() [][]Widget { return d.children }

// Build returns the child at the vicinity, or nil beyond the lists.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate.build
func (d *TwoDimensionalChildListDelegate) Build(_ BuildContext, vicinity ChildVicinity) Widget {
	// If an explicit upper bound has been exceeded, there is no child.
	if vicinity.YIndex < 0 || vicinity.YIndex >= len(d.children) {
		return nil
	}
	if vicinity.XIndex < 0 || vicinity.XIndex >= len(d.children[vicinity.YIndex]) {
		return nil
	}
	return wrapTwoDimensionalChild(d.children[vicinity.YIndex][vicinity.XIndex], d.addRepaintBoundaries, d.addAutomaticKeepAlives)
}

// ShouldRebuild is whether the children are not the same lists.
//
// Flutter: widgets/scroll_delegate.dart TwoDimensionalChildListDelegate.shouldRebuild
func (d *TwoDimensionalChildListDelegate) ShouldRebuild(old rendering.TwoDimensionalChildDelegate) bool {
	o, ok := old.(*TwoDimensionalChildListDelegate)
	if !ok || len(o.children) != len(d.children) {
		return true
	}
	if len(d.children) == 0 {
		return false
	}
	// The same slice has the same first element address.
	return &o.children[0] != &d.children[0]
}

// TwoDimensionalViewportOpts are the optional parameters of
// NewTwoDimensionalViewportBase.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.new
type TwoDimensionalViewportOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollCacheExtent is how far beyond the viewport children are built; the
	// default is 250 pixels.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// ClipBehavior is how the viewport clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// TwoDimensionalViewport is the base of the widgets that create a
// RenderTwoDimensionalViewport. The concrete widget embeds it, creates the
// render object in CreateRenderObject (giving the context, which is the child
// manager, to the render object) and updates it in UpdateRenderObject, and
// defines CreateElement as NewTwoDimensionalViewportElement(w).
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport
type TwoDimensionalViewport struct {
	RenderObjectWidget
	verticalOffset          rendering.ViewportOffset
	verticalAxisDirection   painting.AxisDirection
	horizontalOffset        rendering.ViewportOffset
	horizontalAxisDirection painting.AxisDirection
	delegate                TwoDimensionalChildDelegate
	mainAxis                painting.Axis
	scrollCacheExtent       foundation.Opt[rendering.ScrollCacheExtent]
	clipBehavior            ui.Clip
}

// NewTwoDimensionalViewportBase creates the base of a two-dimensional viewport
// widget. In debug builds it panics with the tag TwoDimensionalViewportAxisInvalid
// when a direction is along the wrong axis.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.new
func NewTwoDimensionalViewportBase(verticalOffset rendering.ViewportOffset, verticalAxisDirection painting.AxisDirection,
	horizontalOffset rendering.ViewportOffset, horizontalAxisDirection painting.AxisDirection,
	delegate TwoDimensionalChildDelegate, mainAxis painting.Axis, opts ...TwoDimensionalViewportOpts) TwoDimensionalViewport {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if verticalAxisDirection != painting.AxisDirectionDown && verticalAxisDirection != painting.AxisDirectionUp {
			panic(errors.New("TwoDimensionalViewportAxisInvalid", "TwoDimensionalViewport.verticalAxisDirection is not Axis.vertical"))
		}
		if horizontalAxisDirection != painting.AxisDirectionLeft && horizontalAxisDirection != painting.AxisDirectionRight {
			panic(errors.New("TwoDimensionalViewportAxisInvalid", "TwoDimensionalViewport.horizontalAxisDirection is not Axis.horizontal"))
		}
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	return TwoDimensionalViewport{
		RenderObjectWidget: NewRenderObjectWidgetBase(WidgetOpts{Key: o.Key}),
		verticalOffset:     verticalOffset, verticalAxisDirection: verticalAxisDirection,
		horizontalOffset: horizontalOffset, horizontalAxisDirection: horizontalAxisDirection,
		delegate: delegate, mainAxis: mainAxis, scrollCacheExtent: o.ScrollCacheExtent, clipBehavior: o.ClipBehavior,
	}
}

// VerticalOffset is the offset of the vertical axis.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.verticalOffset
func (w *TwoDimensionalViewport) VerticalOffset() rendering.ViewportOffset { return w.verticalOffset }

// VerticalAxisDirection is the direction in which the vertical offset grows.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.verticalAxisDirection
func (w *TwoDimensionalViewport) VerticalAxisDirection() painting.AxisDirection {
	return w.verticalAxisDirection
}

// HorizontalOffset is the offset of the horizontal axis.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.horizontalOffset
func (w *TwoDimensionalViewport) HorizontalOffset() rendering.ViewportOffset {
	return w.horizontalOffset
}

// HorizontalAxisDirection is the direction in which the horizontal offset grows.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.horizontalAxisDirection
func (w *TwoDimensionalViewport) HorizontalAxisDirection() painting.AxisDirection {
	return w.horizontalAxisDirection
}

// Delegate supplies the children.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.delegate
func (w *TwoDimensionalViewport) Delegate() TwoDimensionalChildDelegate { return w.delegate }

// MainAxis is the axis the children are ordered by.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.mainAxis
func (w *TwoDimensionalViewport) MainAxis() painting.Axis { return w.mainAxis }

// ScrollCacheExtent is how far beyond the viewport children are built, or none
// for the default.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.scrollCacheExtent
func (w *TwoDimensionalViewport) ScrollCacheExtent() foundation.Opt[rendering.ScrollCacheExtent] {
	return w.scrollCacheExtent
}

// ClipBehavior is how the viewport clips.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.clipBehavior
func (w *TwoDimensionalViewport) ClipBehavior() ui.Clip { return w.clipBehavior }

// UpdateViewport copies the properties of the widget to the render object. A
// concrete widget calls it from UpdateRenderObject, then sets its own.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.updateRenderObject
func (w *TwoDimensionalViewport) UpdateViewport(r *rendering.RenderTwoDimensionalViewport) {
	r.SetHorizontalOffset(w.horizontalOffset)
	r.SetHorizontalAxisDirection(w.horizontalAxisDirection)
	r.SetVerticalOffset(w.verticalOffset)
	r.SetVerticalAxisDirection(w.verticalAxisDirection)
	r.SetMainAxis(w.mainAxis)
	r.SetDelegate(w.delegate)
	r.SetScrollCacheExtent(w.scrollCacheExtent.OrElse(rendering.NewScrollCacheExtentPixels(rendering.RenderAbstractViewportDefaultCacheExtent)))
	r.SetClipBehavior(w.clipBehavior)
}

// NewTwoDimensionalViewportElement creates the element of a concrete
// two-dimensional viewport widget.
//
// Flutter: widgets/two_dimensional_viewport.dart TwoDimensionalViewport.createElement
func NewTwoDimensionalViewportElement(w Widget) Element {
	e := &twoDimensionalViewportElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// elementsByVicinity is a map from vicinities to elements that iterates in
// insertion order, as Dart's map does.
type elementsByVicinity struct {
	items map[ChildVicinity]Element
	order []ChildVicinity
}

func (m *elementsByVicinity) get(v ChildVicinity) Element { return m.items[v] }

func (m *elementsByVicinity) set(v ChildVicinity, e Element) {
	if m.items == nil {
		m.items = map[ChildVicinity]Element{}
	}
	if _, ok := m.items[v]; !ok {
		m.order = append(m.order, v)
	}
	m.items[v] = e
}

func (m *elementsByVicinity) remove(v ChildVicinity) {
	if _, ok := m.items[v]; !ok {
		return
	}
	delete(m.items, v)
	for i, x := range m.order {
		if x == v {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

func (m *elementsByVicinity) values() []Element {
	out := make([]Element, 0, len(m.order))
	for _, v := range m.order {
		out = append(out, m.items[v])
	}
	return out
}

// twoDimensionalViewportElement is the element of a two-dimensional viewport: it
// is the child manager of the render object, which builds the children while it
// lays out.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement
type twoDimensionalViewportElement struct {
	RenderObjectElement
	// Contains all children, including those that are keyed.
	vicinityToChild elementsByVicinity
	keyToChild      map[foundation.Key]Element
	// Used between StartLayout and EndLayout to compute the new values.
	newVicinityToChild *elementsByVicinity
	newKeyToChild      map[foundation.Key]Element
}

func (e *twoDimensionalViewportElement) viewport() *rendering.RenderTwoDimensionalViewport {
	return e.renderObject.(interface {
		AsRenderTwoDimensionalViewport() *rendering.RenderTwoDimensionalViewport
	}).AsRenderTwoDimensionalViewport()
}

func (e *twoDimensionalViewportElement) delegate() TwoDimensionalChildDelegate {
	return e.widget.(interface {
		Delegate() TwoDimensionalChildDelegate
	}).Delegate()
}

// OnNotification counts the notification as one that went through a viewport.
//
// Flutter: widgets/scroll_notification.dart ViewportElementMixin.onNotification
func (e *twoDimensionalViewportElement) OnNotification(notification Notification) bool {
	return ViewportElementOnNotification(notification)
}

// PerformRebuild asks the render object to build the children again during its
// layout, as only then it is known which of them are visible.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement.performRebuild
func (e *twoDimensionalViewportElement) PerformRebuild() {
	e.RenderObjectElement.PerformRebuild()
	e.viewport().MarkNeedsLayoutWithDelegateRebuild()
}

func (e *twoDimensionalViewportElement) debugIsDoingLayout() bool {
	return e.newKeyToChild != nil && e.newVicinityToChild != nil
}

// ForgetChild drops the child.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement.forgetChild
func (e *twoDimensionalViewportElement) ForgetChild(child Element) {
	if foundation.DebugMode && e.debugIsDoingLayout() {
		panic(errors.New("TwoDimensionalViewportForgetDuringLayout", "a child was forgotten while the viewport was building"))
	}
	e.RenderObjectElement.ForgetChild(child)
	e.vicinityToChild.remove(child.AsElement().slot.(ChildVicinity))
	if key := child.Widget().Key(); key != nil {
		delete(e.keyToChild, key)
	}
}

// InsertRenderObjectChild adds the child to the render object.
func (e *twoDimensionalViewportElement) InsertRenderObjectChild(child rendering.RenderObject, slot any) {
	e.viewport().InsertChild(child.(rendering.Box), slot.(ChildVicinity))
}

// MoveRenderObjectChild moves the child between slots.
func (e *twoDimensionalViewportElement) MoveRenderObjectChild(child rendering.RenderObject, oldSlot, newSlot any) {
	e.viewport().MoveChild(child.(rendering.Box), oldSlot.(ChildVicinity), newSlot.(ChildVicinity))
}

// RemoveRenderObjectChild removes the child from the render object.
func (e *twoDimensionalViewportElement) RemoveRenderObjectChild(child rendering.RenderObject, slot any) {
	e.viewport().RemoveChild(child.(rendering.Box), slot.(ChildVicinity))
}

// VisitChildren visits the children.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement.visitChildren
func (e *twoDimensionalViewportElement) VisitChildren(visitor ElementVisitor) {
	for _, child := range e.vicinityToChild.values() {
		visitor(child)
	}
}

// DebugDescribeChildren describes the children by their slot, in order.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement.debugDescribeChildren
func (e *twoDimensionalViewportElement) DebugDescribeChildren() []foundation.DiagnosticsNode {
	children := e.vicinityToChild.values()
	sort.SliceStable(children, func(i, j int) bool {
		return children[i].AsElement().slot.(ChildVicinity).CompareTo(children[j].AsElement().slot.(ChildVicinity)) < 0
	})
	out := make([]foundation.DiagnosticsNode, 0, len(children))
	for _, child := range children {
		out = append(out, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some(child.AsElement().slot.(ChildVicinity).String())}))
	}
	return out
}

// StartLayout starts collecting the children of the layout.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement._startLayout
func (e *twoDimensionalViewportElement) StartLayout() {
	if foundation.DebugMode && e.debugIsDoingLayout() {
		panic(errors.New("TwoDimensionalViewportLayoutNested", "the viewport started a layout while another was running"))
	}
	e.newVicinityToChild = &elementsByVicinity{}
	e.newKeyToChild = map[foundation.Key]Element{}
}

// BuildChild builds the child at the vicinity with the delegate.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement._buildChild
func (e *twoDimensionalViewportElement) BuildChild(vicinity ChildVicinity) {
	if foundation.DebugMode && !e.debugIsDoingLayout() {
		panic(errors.New("TwoDimensionalViewportLayoutNotStarted", "a child was built outside of the layout of the viewport"))
	}
	e.owner.BuildScope(e, func() {
		newWidget := e.delegate().Build(e, vicinity)
		if newWidget == nil {
			return
		}
		oldElement := e.retrieveOldElement(newWidget, vicinity)
		newChild := e.UpdateChild(oldElement, newWidget, vicinity)
		if foundation.DebugMode {
			// Ensure the child does not overwrite an existing one.
			if e.newVicinityToChild.get(vicinity) != nil {
				panic(errors.New("TwoDimensionalViewportChildDuplicate", "a child was built twice for the same vicinity", "vicinity", vicinity.String()))
			}
		}
		e.newVicinityToChild.set(vicinity, newChild)
		if key := newWidget.Key(); key != nil {
			if foundation.DebugMode && e.newKeyToChild[key] != nil {
				panic(errors.New("TwoDimensionalViewportKeyDuplicate", "two children have the same key", "key", key.String()))
			}
			e.newKeyToChild[key] = newChild
		}
	})
}

func (e *twoDimensionalViewportElement) retrieveOldElement(newWidget Widget, vicinity ChildVicinity) Element {
	if key := newWidget.Key(); key != nil {
		if result := e.keyToChild[key]; result != nil {
			delete(e.keyToChild, key)
			e.vicinityToChild.remove(result.AsElement().slot.(ChildVicinity))
			return result
		}
		return nil
	}
	if potentialOldElement := e.vicinityToChild.get(vicinity); potentialOldElement != nil && potentialOldElement.Widget().Key() == nil {
		e.vicinityToChild.remove(vicinity)
		return potentialOldElement
	}
	return nil
}

// ReuseChild keeps the child at the vicinity for the layout.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement._reuseChild
func (e *twoDimensionalViewportElement) ReuseChild(vicinity ChildVicinity) {
	if foundation.DebugMode && !e.debugIsDoingLayout() {
		panic(errors.New("TwoDimensionalViewportLayoutNotStarted", "a child was reused outside of the layout of the viewport"))
	}
	elementToReuse := e.vicinityToChild.get(vicinity)
	if foundation.DebugMode && elementToReuse == nil {
		panic(errors.New("TwoDimensionalViewportReuseMissing", "expected to reuse an element, but none was found", "vicinity", vicinity.String()))
	}
	e.vicinityToChild.remove(vicinity)
	e.newVicinityToChild.set(vicinity, elementToReuse)
	if key := elementToReuse.Widget().Key(); key != nil {
		e.newKeyToChild[key] = e.keyToChild[key]
		delete(e.keyToChild, key)
	}
}

// EndLayout unmounts the children that the layout did not build or reuse.
//
// Flutter: widgets/two_dimensional_viewport.dart _TwoDimensionalViewportElement._endLayout
func (e *twoDimensionalViewportElement) EndLayout() {
	if foundation.DebugMode && !e.debugIsDoingLayout() {
		panic(errors.New("TwoDimensionalViewportLayoutNotStarted", "the layout of the viewport ended but did not start"))
	}
	// Unmount all elements that have not been reused in the layout cycle.
	for _, element := range e.vicinityToChild.values() {
		if element.Widget().Key() == nil {
			// If it has a key, it is handled below.
			e.UpdateChild(element, nil, nil)
		}
	}
	for _, element := range e.keyToChild {
		e.UpdateChild(element, nil, nil)
	}
	e.vicinityToChild = *e.newVicinityToChild
	e.keyToChild = e.newKeyToChild
	e.newVicinityToChild = nil
	e.newKeyToChild = nil
}

// TwoDimensionalScrollViewSelf is what TwoDimensionalScrollView needs of the type
// that embeds it.
type TwoDimensionalScrollViewSelf interface {
	// BuildViewport builds the viewport with the offset of each axis.
	//
	// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.buildViewport
	BuildViewport(bc BuildContext, verticalOffset, horizontalOffset rendering.ViewportOffset) Widget
}

// TwoDimensionalScrollViewOpts are the optional parameters of
// InitTwoDimensionalScrollView.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.new
type TwoDimensionalScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Primary is whether this is the primary scroll view for the main axis; the
	// default is true when it has no controller there and the main axis is vertical.
	Primary foundation.Opt[bool]
	// MainAxis is the axis the children are ordered by; the default is vertical.
	MainAxis painting.Axis
	// VerticalDetails are the details of the vertical axis; the default is
	// ScrollableDetails vertical.
	VerticalDetails foundation.Opt[ScrollableDetails]
	// HorizontalDetails are the details of the horizontal axis; the default is
	// ScrollableDetails horizontal.
	HorizontalDetails foundation.Opt[ScrollableDetails]
	// ScrollCacheExtent is how far beyond the viewport children are built.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// DiagonalDragBehavior is how the view responds to a diagonal drag.
	DiagonalDragBehavior DiagonalDragBehavior
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// KeyboardDismissBehavior is how the view dismisses the keyboard.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	// ClipBehavior is how the view clips; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
}

// TwoDimensionalScrollView is the base of the scroll views that scroll along two
// axes: it builds a TwoDimensionalScrollable that builds the viewport. The
// concrete widget embeds it, calls InitTwoDimensionalScrollView, and implements
// BuildViewport.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView
type TwoDimensionalScrollView struct {
	StatelessWidget
	self                    TwoDimensionalScrollViewSelf
	primary                 foundation.Opt[bool]
	mainAxis                painting.Axis
	verticalDetails         ScrollableDetails
	horizontalDetails       ScrollableDetails
	delegate                TwoDimensionalChildDelegate
	scrollCacheExtent       foundation.Opt[rendering.ScrollCacheExtent]
	diagonalDragBehavior    DiagonalDragBehavior
	dragStartBehavior       gestures.DragStartBehavior
	keyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	clipBehavior            ui.Clip
	hitTestBehavior         rendering.HitTestBehavior
}

// InitTwoDimensionalScrollView sets the outer widget and the properties.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.new
func InitTwoDimensionalScrollView(v *TwoDimensionalScrollView, self TwoDimensionalScrollViewSelf, delegate TwoDimensionalChildDelegate, opts ...TwoDimensionalScrollViewOpts) {
	o := firstOpts(opts)
	if o.MainAxis == painting.AxisUnspecified {
		o.MainAxis = painting.AxisVertical
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	v.StatelessWidget = NewStatelessWidgetBase(WidgetOpts{Key: o.Key})
	v.self = self
	v.primary, v.mainAxis = o.Primary, o.MainAxis
	v.verticalDetails = o.VerticalDetails.OrElse(NewScrollableDetailsVertical())
	v.horizontalDetails = o.HorizontalDetails.OrElse(NewScrollableDetailsHorizontal())
	v.delegate = delegate
	v.scrollCacheExtent = o.ScrollCacheExtent
	v.diagonalDragBehavior, v.dragStartBehavior = o.DiagonalDragBehavior, o.DragStartBehavior
	v.keyboardDismissBehavior, v.clipBehavior, v.hitTestBehavior = o.KeyboardDismissBehavior, o.ClipBehavior, o.HitTestBehavior
}

// Delegate supplies the children.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.delegate
func (v *TwoDimensionalScrollView) Delegate() TwoDimensionalChildDelegate { return v.delegate }

// MainAxis is the axis the children are ordered by.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.mainAxis
func (v *TwoDimensionalScrollView) MainAxis() painting.Axis { return v.mainAxis }

// ScrollCacheExtent is how far beyond the viewport children are built, or none.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.scrollCacheExtent
func (v *TwoDimensionalScrollView) ScrollCacheExtent() foundation.Opt[rendering.ScrollCacheExtent] {
	return v.scrollCacheExtent
}

// ClipBehavior is how the view clips.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.clipBehavior
func (v *TwoDimensionalScrollView) ClipBehavior() ui.Clip { return v.clipBehavior }

// VerticalDetails are the details of the vertical axis.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.verticalDetails
func (v *TwoDimensionalScrollView) VerticalDetails() ScrollableDetails { return v.verticalDetails }

// HorizontalDetails are the details of the horizontal axis.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.horizontalDetails
func (v *TwoDimensionalScrollView) HorizontalDetails() ScrollableDetails { return v.horizontalDetails }

// DiagonalDragBehavior is how the view responds to a diagonal drag.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.diagonalDragBehavior
func (v *TwoDimensionalScrollView) DiagonalDragBehavior() DiagonalDragBehavior {
	return v.diagonalDragBehavior
}

// DragStartBehavior is where a drag starts.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.dragStartBehavior
func (v *TwoDimensionalScrollView) DragStartBehavior() gestures.DragStartBehavior {
	return v.dragStartBehavior
}

// Build builds the TwoDimensionalScrollable that builds the viewport.
//
// Flutter: widgets/two_dimensional_scroll_view.dart TwoDimensionalScrollView.build
func (v *TwoDimensionalScrollView) Build(bc BuildContext) Widget {
	if foundation.DebugMode {
		if painting.AxisDirectionToAxis(v.verticalDetails.Direction) != painting.AxisVertical {
			panic(errors.New("TwoDimensionalScrollViewAxisInvalid", "TwoDimensionalScrollView.verticalDetails are not Axis.vertical"))
		}
		if painting.AxisDirectionToAxis(v.horizontalDetails.Direction) != painting.AxisHorizontal {
			panic(errors.New("TwoDimensionalScrollViewAxisInvalid", "TwoDimensionalScrollView.horizontalDetails are not Axis.horizontal"))
		}
	}
	mainAxisDetails := v.verticalDetails
	if v.mainAxis == painting.AxisHorizontal {
		mainAxisDetails = v.horizontalDetails
	}
	effectivePrimary := v.primary.OrElse(isNil(mainAxisDetails.Controller) && PrimaryScrollControllerShouldInherit(bc, v.mainAxis))
	if effectivePrimary {
		// Using the PrimaryScrollController for the main axis.
		if foundation.DebugMode && !isNil(mainAxisDetails.Controller) {
			panic(errors.New("TwoDimensionalScrollViewPrimaryConflict",
				"TwoDimensionalScrollView.primary was explicitly set to true, but a ScrollController was provided in the ScrollableDetails of the TwoDimensionalScrollView.mainAxis"))
		}
		mainAxisDetails = mainAxisDetails.CopyWith(ScrollableDetailsCopy{Controller: PrimaryScrollControllerOf(bc)})
	}
	horizontalDetails, verticalDetails := v.horizontalDetails, v.verticalDetails
	if v.mainAxis == painting.AxisHorizontal {
		horizontalDetails = mainAxisDetails
	} else {
		verticalDetails = mainAxisDetails
	}
	scrollable := NewTwoDimensionalScrollable(horizontalDetails, verticalDetails, v.self.BuildViewport, TwoDimensionalScrollableOpts{
		DiagonalDragBehavior: v.diagonalDragBehavior, DragStartBehavior: v.dragStartBehavior, HitTestBehavior: v.hitTestBehavior,
	})
	var result Widget = scrollable
	if effectivePrimary {
		// Further descendant ScrollViews will not inherit the same
		// PrimaryScrollController.
		result = NewPrimaryScrollControllerNone(scrollable)
	}
	return wrapKeyboardDismiss(bc, v.keyboardDismissBehavior, nil, result)
}
