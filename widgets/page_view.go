package widgets

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// PageControllerOpts are the optional parameters of NewPageController.
//
// Flutter: widgets/page_view.dart PageController.new
type PageControllerOpts struct {
	// InitialPage is the page to show when the first view is created.
	InitialPage int
	// KeepPage is whether the page is saved in the PageStorage, and restored when
	// a new view is created; the default is true.
	KeepPage foundation.Opt[bool]
	// ViewportFraction is the fraction of the viewport that each page occupies;
	// the default is 1.
	ViewportFraction foundation.Opt[float64]
	// OnAttach is called when a position is attached.
	OnAttach ScrollControllerCallback
	// OnDetach is called when a position is detached.
	OnDetach ScrollControllerCallback
}

// PageController controls which page is visible in a PageView, and the scroll
// position in pages instead of pixels.
//
// Flutter: widgets/page_view.dart PageController
type PageController struct {
	ScrollControllerBase
	initialPage      int
	keepPage         bool
	viewportFraction float64
}

// NewPageController creates a controller for a page view. In debug builds it
// panics with the tag ViewportFractionInvalid when the fraction is not positive.
//
// Flutter: widgets/page_view.dart PageController.new
func NewPageController(opts ...PageControllerOpts) *PageController {
	o := firstOpts(opts)
	fraction := o.ViewportFraction.OrElse(1)
	if foundation.DebugMode && !(fraction > 0) {
		panic(errors.New("ViewportFractionInvalid", "the viewport fraction must be greater than zero", "viewportFraction", fraction))
	}
	c := &PageController{initialPage: o.InitialPage, keepPage: o.KeepPage.OrElse(true), viewportFraction: fraction}
	InitScrollControllerBase(&c.ScrollControllerBase, c, ScrollControllerOpts{OnAttach: o.OnAttach, OnDetach: o.OnDetach})
	return c
}

// InitialPage is the page to show when the first view is created.
//
// Flutter: widgets/page_view.dart PageController.initialPage
func (c *PageController) InitialPage() int { return c.initialPage }

// KeepPage is whether the page is saved in the PageStorage.
//
// Flutter: widgets/page_view.dart PageController.keepPage
func (c *PageController) KeepPage() bool { return c.keepPage }

// ViewportFraction is the fraction of the viewport that each page occupies.
//
// Flutter: widgets/page_view.dart PageController.viewportFraction
func (c *PageController) ViewportFraction() float64 { return c.viewportFraction }

// Page is the current page, a fraction while the view is scrolling between two,
// or none before the first layout. In debug builds it panics with the tag
// PageControllerNotAttached when no view uses the controller and with
// PageControllerMultipleViews when more than one does.
//
// Flutter: widgets/page_view.dart PageController.page
func (c *PageController) Page() foundation.Opt[float64] {
	if foundation.DebugMode {
		if len(c.positions) == 0 {
			panic(errors.New("PageControllerNotAttached", "PageController.page cannot be accessed before a PageView is built with it"))
		}
		if len(c.positions) != 1 {
			panic(errors.New("PageControllerMultipleViews", "the page property cannot be read when multiple PageViews are attached to the same PageController"))
		}
	}
	return c.Position().(*pagePosition).Page()
}

func (c *PageController) debugCheckAttached() {
	if !foundation.DebugMode {
		return
	}
	if len(c.positions) == 0 {
		panic(errors.New("PageControllerNotAttached", "PageController is not attached to a PageView"))
	}
	if len(c.positions) != 1 {
		panic(errors.New("PageControllerMultipleViews", "multiple PageViews are attached to the same PageController"))
	}
}

// AnimateToPage animates the controlled view to the page. The future completes
// when the animation ends.
//
// Flutter: widgets/page_view.dart PageController.animateToPage
func (c *PageController) AnimateToPage(page int, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	c.debugCheckAttached()
	position := c.Position().(*pagePosition)
	if position.cachedPage.IsSome() {
		position.cachedPage = foundation.Some(float64(page))
		return scheduler.NewTickerFutureComplete()
	}
	if !position.HasViewportDimension() {
		position.pageToUseOnStartup = float64(page)
		return scheduler.NewTickerFutureComplete()
	}
	return position.AnimateTo(position.GetPixelsFromPage(float64(page)), duration, curve)
}

// JumpToPage changes which page is displayed without animation.
//
// Flutter: widgets/page_view.dart PageController.jumpToPage
func (c *PageController) JumpToPage(page int) {
	c.debugCheckAttached()
	position := c.Position().(*pagePosition)
	if position.cachedPage.IsSome() {
		position.cachedPage = foundation.Some(float64(page))
		return
	}
	if !position.HasViewportDimension() {
		position.pageToUseOnStartup = float64(page)
		return
	}
	position.JumpTo(position.GetPixelsFromPage(float64(page)))
}

// NextPage animates to the next page.
//
// Flutter: widgets/page_view.dart PageController.nextPage
func (c *PageController) NextPage(duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	return c.AnimateToPage(int(math.Round(c.Page().Value()))+1, duration, curve)
}

// PreviousPage animates to the previous page.
//
// Flutter: widgets/page_view.dart PageController.previousPage
func (c *PageController) PreviousPage(duration time.Duration, curve animation.Curve) *scheduler.TickerFuture {
	return c.AnimateToPage(int(math.Round(c.Page().Value()))-1, duration, curve)
}

// CreateScrollPosition creates the position that counts in pages.
//
// Flutter: widgets/page_view.dart PageController.createScrollPosition
func (c *PageController) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition {
	return newPagePosition(physics, context, c.initialPage, c.keepPage, c.viewportFraction, oldPosition)
}

// Attach registers the position and gives it the viewport fraction.
//
// Flutter: widgets/page_view.dart PageController.attach
func (c *PageController) Attach(position ScrollPosition) {
	c.ScrollControllerBase.Attach(position)
	position.(*pagePosition).setViewportFraction(c.viewportFraction)
}

// PageMetrics are the scroll metrics of a PageView: they also say the page.
//
// Flutter: widgets/page_view.dart PageMetrics
type PageMetrics interface {
	ScrollMetrics
	// ViewportFraction is the fraction of the viewport that each page occupies.
	//
	// Flutter: widgets/page_view.dart PageMetrics.viewportFraction
	ViewportFraction() float64
	// Page is the current page, or none while the pixels are not known.
	//
	// Flutter: widgets/page_view.dart PageMetrics.page
	Page() foundation.Opt[float64]
}

// FixedPageMetrics is an immutable snapshot of PageMetrics. The scroll
// notifications of a PageView carry one.
//
// Deviation: Dart's PageMetrics is a class; here it is an interface that
// FixedPageMetrics and the page view's position implement (SPEC §12.10).
//
// Flutter: widgets/page_view.dart PageMetrics
type FixedPageMetrics struct {
	FixedScrollMetrics
	viewportFraction float64
}

// NewFixedPageMetrics creates a snapshot. A value that is none is missing: reading
// it panics with the tag ScrollMetricsValueMissing.
//
// Flutter: widgets/page_view.dart PageMetrics.new
func NewFixedPageMetrics(minScrollExtent, maxScrollExtent, pixels, viewportDimension foundation.Opt[float64], axisDirection painting.AxisDirection, viewportFraction, devicePixelRatio float64) *FixedPageMetrics {
	m := &FixedPageMetrics{viewportFraction: viewportFraction}
	m.FixedScrollMetrics = *NewFixedScrollMetrics(minScrollExtent, maxScrollExtent, pixels, viewportDimension, axisDirection, devicePixelRatio)
	InitScrollMetricsMixin(&m.ScrollMetricsMixin, m)
	return m
}

// ViewportFraction is the fraction of the viewport that each page occupies.
//
// Flutter: widgets/page_view.dart PageMetrics.viewportFraction
func (m *FixedPageMetrics) ViewportFraction() float64 { return m.viewportFraction }

// CopyWith is a copy with the changes, and the same viewport fraction.
//
// Flutter: widgets/page_view.dart PageMetrics.copyWith
func (m *FixedPageMetrics) CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics {
	return pageMetricsCopy(m, m.viewportFraction, changes)
}

// Page is the current page.
//
// Flutter: widgets/page_view.dart PageMetrics.page
func (m *FixedPageMetrics) Page() foundation.Opt[float64] { return pageOf(m, m.viewportFraction) }

func pageMetricsCopy(m ScrollMetrics, viewportFraction float64, changes []ScrollMetricsChanges) *FixedPageMetrics {
	f := scrollMetricsCopy(m, changes)
	copied := &FixedPageMetrics{FixedScrollMetrics: *f, viewportFraction: viewportFraction}
	InitScrollMetricsMixin(&copied.ScrollMetricsMixin, copied)
	return copied
}

func pageOf(m ScrollMetrics, viewportFraction float64) foundation.Opt[float64] {
	return foundation.Some(math.Max(0, ui.ClampDouble(m.Pixels(), m.MinScrollExtent(), m.MaxScrollExtent())) /
		math.Max(1, m.ViewportDimension()*viewportFraction))
}

// pagePosition is the scroll position of a PageView: it also works in pages.
//
// Flutter: widgets/page_view.dart _PagePosition
type pagePosition struct {
	ScrollPositionWithSingleContext
	initialPage        int
	pageToUseOnStartup float64
	cachedPage         foundation.Opt[float64]
	viewportFraction   float64
}

func newPagePosition(physics ScrollPhysics, context ScrollContext, initialPage int, keepPage bool, viewportFraction float64, oldPosition ScrollPosition) *pagePosition {
	if foundation.DebugMode && !(viewportFraction > 0) {
		panic(errors.New("ViewportFractionInvalid", "the viewport fraction must be greater than zero", "viewportFraction", viewportFraction))
	}
	p := &pagePosition{initialPage: initialPage, pageToUseOnStartup: float64(initialPage), viewportFraction: viewportFraction}
	InitScrollPositionWithSingleContext(&p.ScrollPositionWithSingleContext, p, physics, context, ScrollPositionWithSingleContextOpts{
		NoInitialPixels: true, KeepScrollOffset: foundation.Some(keepPage), OldPosition: oldPosition})
	return p
}

// ViewportFraction is the fraction of the viewport that each page occupies.
//
// Flutter: widgets/page_view.dart _PagePosition.viewportFraction
func (p *pagePosition) ViewportFraction() float64 { return p.viewportFraction }

func (p *pagePosition) setViewportFraction(value float64) {
	if p.viewportFraction == value {
		return
	}
	oldPage := p.Page()
	p.viewportFraction = value
	if oldPage, ok := oldPage.Get(); ok {
		p.ForcePixels(p.GetPixelsFromPage(oldPage))
	}
}

func (p *pagePosition) initialPageOffset() float64 {
	return math.Max(0, p.ViewportDimension()*(p.viewportFraction-1)/2)
}

// GetPageFromPixels is the page at the pixels, for the viewport dimension.
//
// Flutter: widgets/page_view.dart _PagePosition.getPageFromPixels
func (p *pagePosition) GetPageFromPixels(pixels, viewportDimension float64) float64 {
	if foundation.DebugMode && !(viewportDimension > 0) {
		panic(errors.New("PageViewportDimensionInvalid", "the viewport dimension must be greater than zero", "viewportDimension", viewportDimension))
	}
	actual := math.Max(0, pixels-p.initialPageOffset()) / (viewportDimension * p.viewportFraction)
	round := math.Round(actual)
	if math.Abs(actual-round) < foundation.PrecisionErrorTolerance {
		return round
	}
	return actual
}

// GetPixelsFromPage is the pixels of the page.
//
// Flutter: widgets/page_view.dart _PagePosition.getPixelsFromPage
func (p *pagePosition) GetPixelsFromPage(page float64) float64 {
	return page*p.ViewportDimension()*p.viewportFraction + p.initialPageOffset()
}

// Page is the current page, or none before the pixels are known.
//
// Flutter: widgets/page_view.dart _PagePosition.page
func (p *pagePosition) Page() foundation.Opt[float64] {
	if !p.HasPixels() {
		return foundation.None[float64]()
	}
	if foundation.DebugMode && !p.HasContentDimensions() && p.haveDimensions {
		panic(errors.New("PageBeforeContentDimensions", "page value is only available after content dimensions are established"))
	}
	if !p.HasContentDimensions() && !p.haveDimensions {
		return foundation.None[float64]()
	}
	if p.cachedPage.IsSome() {
		return p.cachedPage
	}
	return foundation.Some(p.GetPageFromPixels(ui.ClampDouble(p.Pixels(), p.MinScrollExtent(), p.MaxScrollExtent()), p.ViewportDimension()))
}

// SaveScrollOffset saves the page in the PageStorage.
//
// Flutter: widgets/page_view.dart _PagePosition.saveScrollOffset
func (p *pagePosition) SaveScrollOffset() {
	if bucket := MaybePageStorageOf(p.context.StorageContext()); bucket != nil {
		page := p.cachedPage.OrElse(0)
		if p.cachedPage.IsNone() {
			page = p.GetPageFromPixels(p.Pixels(), p.ViewportDimension())
		}
		bucket.WriteState(p.context.StorageContext(), page)
	}
}

// RestoreScrollOffset reads the page from the PageStorage.
//
// Flutter: widgets/page_view.dart _PagePosition.restoreScrollOffset
func (p *pagePosition) RestoreScrollOffset() {
	if p.HasPixels() {
		return
	}
	if bucket := MaybePageStorageOf(p.context.StorageContext()); bucket != nil {
		if v, ok := bucket.ReadState(p.context.StorageContext()).(float64); ok {
			p.pageToUseOnStartup = v
		}
	}
}

// SaveOffset gives the page to the scroll context to save.
//
// Flutter: widgets/page_view.dart _PagePosition.saveOffset
func (p *pagePosition) SaveOffset() {
	page := p.cachedPage.OrElse(0)
	if p.cachedPage.IsNone() {
		page = p.GetPageFromPixels(p.Pixels(), p.ViewportDimension())
	}
	p.context.SaveOffset(page)
}

// RestoreOffset restores a saved page.
//
// Flutter: widgets/page_view.dart _PagePosition.restoreOffset
func (p *pagePosition) RestoreOffset(offset float64, initialRestore bool) {
	if initialRestore {
		p.pageToUseOnStartup = offset
	} else {
		p.JumpTo(p.GetPixelsFromPage(offset))
	}
}

// ApplyViewportDimension keeps the page when the viewport changes its size.
//
// Flutter: widgets/page_view.dart _PagePosition.applyViewportDimension
func (p *pagePosition) ApplyViewportDimension(viewportDimension float64) bool {
	var oldViewportDimensions foundation.Opt[float64]
	if p.HasViewportDimension() {
		oldViewportDimensions = foundation.Some(p.ViewportDimension())
	}
	if oldViewportDimensions == foundation.Some(viewportDimension) {
		return true
	}
	result := p.ScrollPositionWithSingleContext.ApplyViewportDimension(viewportDimension)
	var oldPixels foundation.Opt[float64]
	if p.HasPixels() {
		oldPixels = foundation.Some(p.Pixels())
	}
	var page float64
	switch {
	case oldPixels.IsNone():
		page = p.pageToUseOnStartup
	case oldViewportDimensions == foundation.Some(0.0):
		page = p.cachedPage.Value()
	default:
		page = p.GetPageFromPixels(oldPixels.Value(), oldViewportDimensions.Value())
	}
	newPixels := p.GetPixelsFromPage(page)
	if viewportDimension == 0 {
		p.cachedPage = foundation.Some(page)
	} else {
		p.cachedPage = foundation.None[float64]()
	}
	if oldPixels != foundation.Some(newPixels) {
		p.CorrectPixels(newPixels)
		return false
	}
	return result
}

// Absorb takes over the cached page of the old position.
//
// Flutter: widgets/page_view.dart _PagePosition.absorb
func (p *pagePosition) Absorb(other ScrollPosition) {
	p.ScrollPositionWithSingleContext.Absorb(other)
	if foundation.DebugMode && p.cachedPage.IsSome() {
		panic(errors.New("PageCachedPageSet", "a page position absorbed another one while it had a cached page"))
	}
	if o, ok := other.(*pagePosition); ok && o.cachedPage.IsSome() {
		p.cachedPage = o.cachedPage
	}
}

// ApplyContentDimensions makes room for the padding of the ends.
//
// Flutter: widgets/page_view.dart _PagePosition.applyContentDimensions
func (p *pagePosition) ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool {
	newMinScrollExtent := minScrollExtent + p.initialPageOffset()
	return p.ScrollPositionWithSingleContext.ApplyContentDimensions(newMinScrollExtent, math.Max(newMinScrollExtent, maxScrollExtent-p.initialPageOffset()))
}

// CopyWith is a snapshot of the metrics, as page metrics.
//
// Flutter: widgets/page_view.dart _PagePosition.copyWith
func (p *pagePosition) CopyWith(changes ...ScrollMetricsChanges) ScrollMetrics {
	return pageMetricsCopy(p, p.viewportFraction, changes)
}

// forceImplicitScrollPhysics decides whether the view scrolls implicitly.
//
// Flutter: widgets/page_view.dart _ForceImplicitScrollPhysics
type forceImplicitScrollPhysics struct {
	ScrollPhysicsBase
	allowImplicitScrolling bool
}

func newForceImplicitScrollPhysics(allowImplicitScrolling bool, parent ScrollPhysics) *forceImplicitScrollPhysics {
	p := &forceImplicitScrollPhysics{allowImplicitScrolling: allowImplicitScrolling}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

func (p *forceImplicitScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return newForceImplicitScrollPhysics(p.allowImplicitScrolling, p.BuildParent(ancestor))
}

func (p *forceImplicitScrollPhysics) AllowImplicitScrolling() bool { return p.allowImplicitScrolling }

// PageScrollPhysics are the scroll physics used by a PageView: they settle on a
// whole page.
//
// Flutter: widgets/page_view.dart PageScrollPhysics
type PageScrollPhysics struct {
	ScrollPhysicsBase
}

// NewPageScrollPhysics creates the physics with the parent (nil for none).
//
// Flutter: widgets/page_view.dart PageScrollPhysics.new
func NewPageScrollPhysics(parent ScrollPhysics) *PageScrollPhysics {
	p := &PageScrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, parent)
	return p
}

// ApplyTo returns a copy on top of the ancestor.
//
// Flutter: widgets/page_view.dart PageScrollPhysics.applyTo
func (p *PageScrollPhysics) ApplyTo(ancestor ScrollPhysics) ScrollPhysics {
	return NewPageScrollPhysics(p.BuildParent(ancestor))
}

func (p *PageScrollPhysics) getPage(position ScrollMetrics) float64 {
	if pp, ok := position.(*pagePosition); ok {
		return pp.Page().Value()
	}
	return position.Pixels() / position.ViewportDimension()
}

func (p *PageScrollPhysics) getPixels(position ScrollMetrics, page float64) float64 {
	if pp, ok := position.(*pagePosition); ok {
		return pp.GetPixelsFromPage(page)
	}
	return page * position.ViewportDimension()
}

func (p *PageScrollPhysics) getTargetPixels(position ScrollMetrics, tolerance physics.Tolerance, velocity float64) float64 {
	page := p.getPage(position)
	if velocity < -tolerance.Velocity {
		page -= 0.5
	} else if velocity > tolerance.Velocity {
		page += 0.5
	}
	return p.getPixels(position, math.Round(page))
}

// CreateBallisticSimulation is a spring to the page that the fling ends on.
//
// Flutter: widgets/page_view.dart PageScrollPhysics.createBallisticSimulation
func (p *PageScrollPhysics) CreateBallisticSimulation(position ScrollMetrics, velocity float64) physics.Simulation {
	// If we're out of range and not headed back in range, defer to the parent
	// ballistics, which should put us back in range at a page boundary.
	if (velocity <= 0 && position.Pixels() <= position.MinScrollExtent()) || (velocity >= 0 && position.Pixels() >= position.MaxScrollExtent()) {
		return p.ScrollPhysicsBase.CreateBallisticSimulation(position, velocity)
	}
	tolerance := p.ToleranceFor(position)
	target := p.getTargetPixels(position, tolerance, velocity)
	if target != position.Pixels() {
		return physics.NewScrollSpringSimulation(p.Spring(), position.Pixels(), target, velocity,
			physics.SpringSimulationOpts{Tolerance: foundation.Some(tolerance)})
	}
	return nil
}

// AllowImplicitScrolling is false.
//
// Flutter: widgets/page_view.dart PageScrollPhysics.allowImplicitScrolling
func (p *PageScrollPhysics) AllowImplicitScrolling() bool { return false }

// PageViewOpts are the optional parameters of NewPageView and its named
// constructors.
//
// Flutter: widgets/page_view.dart PageView.new
type PageViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollDirection is the axis along which the pages scroll; the default is
	// AxisHorizontal.
	ScrollDirection painting.Axis
	// Reverse is whether the pages scroll against the reading direction.
	Reverse bool
	// Controller controls which page is visible.
	Controller *PageController
	// Physics is how the pages respond to the user.
	Physics ScrollPhysics
	// PageSnapping is whether the view settles on a whole page; the default is
	// true.
	PageSnapping foundation.Opt[bool]
	// OnPageChanged is called when the page changes.
	OnPageChanged func(index int)
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// AllowImplicitScrolling is whether the pages beyond the visible one are kept
	// for accessibility.
	AllowImplicitScrolling bool
	// ScrollCacheExtent is how far beyond the viewport the view builds pages; it
	// must be positive exactly when AllowImplicitScrolling is true. The default
	// is one viewport with it, none without.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// RestorationID is the identifier for restoring the page; it has no effect
	// until state restoration is ported.
	RestorationID string
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// ScrollBehavior describes how the view scrolls; the default is the one of the
	// ScrollConfiguration above, without scrollbars.
	ScrollBehavior ScrollBehavior
	// PadEnds is whether to add padding to both ends of the list so that the first
	// and last pages can be centered; the default is true.
	PadEnds foundation.Opt[bool]
	// FindChildIndexCallback finds the index of the page that has a key
	// (NewPageViewBuilder).
	FindChildIndexCallback ChildIndexGetter
	// ItemCount is the number of pages; none means unbounded (NewPageViewBuilder).
	ItemCount foundation.Opt[int]
}

// PageView is a scrollable list that works page by page.
//
// Flutter: widgets/page_view.dart PageView
type PageView struct {
	StatefulWidget
	childrenDelegate       SliverChildDelegate
	scrollDirection        painting.Axis
	reverse                bool
	controller             *PageController
	physics                ScrollPhysics
	pageSnapping           bool
	onPageChanged          func(int)
	dragStartBehavior      gestures.DragStartBehavior
	allowImplicitScrolling bool
	scrollCacheExtent      rendering.ScrollCacheExtent
	restorationID          string
	clipBehavior           ui.Clip
	hitTestBehavior        rendering.HitTestBehavior
	scrollBehavior         ScrollBehavior
	padEnds                bool
}

func newPageView(delegate SliverChildDelegate, o PageViewOpts) *PageView {
	if foundation.DebugMode {
		if e, ok := o.ScrollCacheExtent.Get(); ok && (e.Value() > 0) != o.AllowImplicitScrolling {
			panic(errors.New("PageViewScrollCacheExtentInconsistent",
				"scrollCacheExtent and allowImplicitScrolling must be consistent: scrollCacheExtent must be greater than 0.0 when allowImplicitScrolling is true, and must be 0.0 when allowImplicitScrolling is false"))
		}
	}
	extent, ok := o.ScrollCacheExtent.Get()
	if !ok {
		fraction := 0.0
		if o.AllowImplicitScrolling {
			fraction = 1
		}
		extent = rendering.NewScrollCacheExtentViewport(fraction)
	}
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisHorizontal
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	return &PageView{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), delegate, o.ScrollDirection, o.Reverse, o.Controller, o.Physics,
		o.PageSnapping.OrElse(true), o.OnPageChanged, o.DragStartBehavior, o.AllowImplicitScrolling, extent, o.RestorationID,
		o.ClipBehavior, o.HitTestBehavior, o.ScrollBehavior, o.PadEnds.OrElse(true)}
}

// NewPageView creates a scrollable list that works page by page from an explicit
// list of widgets.
//
// Flutter: widgets/page_view.dart PageView.new
func NewPageView(children []Widget, opts ...PageViewOpts) *PageView {
	return newPageView(NewSliverChildListDelegate(children), firstOpts(opts))
}

// NewPageViewBuilder creates a scrollable list that works page by page using
// widgets that are created on demand.
//
// Flutter: widgets/page_view.dart PageView.builder
func NewPageViewBuilder(itemBuilder NullableIndexedWidgetBuilder, opts ...PageViewOpts) *PageView {
	o := firstOpts(opts)
	return newPageView(NewSliverChildBuilderDelegate(itemBuilder, SliverChildBuilderDelegateOpts{
		FindChildIndexCallback: o.FindChildIndexCallback, ChildCount: o.ItemCount}), o)
}

// NewPageViewCustom creates a scrollable list that works page by page with a
// custom child model.
//
// Flutter: widgets/page_view.dart PageView.custom
func NewPageViewCustom(childrenDelegate SliverChildDelegate, opts ...PageViewOpts) *PageView {
	return newPageView(childrenDelegate, firstOpts(opts))
}

// ChildrenDelegate supplies the pages.
//
// Flutter: widgets/page_view.dart PageView.childrenDelegate
func (w *PageView) ChildrenDelegate() SliverChildDelegate { return w.childrenDelegate }

// Controller controls which page is visible, or nil.
//
// Flutter: widgets/page_view.dart PageView.controller
func (w *PageView) Controller() *PageController { return w.controller }

// ScrollCacheExtent is how far beyond the viewport the view builds pages.
//
// Flutter: widgets/page_view.dart PageView.scrollCacheExtent
func (w *PageView) ScrollCacheExtent() rendering.ScrollCacheExtent { return w.scrollCacheExtent }

// AllowImplicitScrolling is whether the pages beyond the visible one are kept.
//
// Flutter: widgets/page_view.dart PageView.allowImplicitScrolling
func (w *PageView) AllowImplicitScrolling() bool { return w.allowImplicitScrolling }

// CreateState creates the state.
//
// Flutter: widgets/page_view.dart PageView.createState
func (w *PageView) CreateState() State { return &pageViewState{} }

type pageViewState struct {
	StateOf[*PageView]
	lastReportedPage int
	controller       *PageController
}

func (s *pageViewState) InitState() {
	s.initController()
	s.lastReportedPage = s.controller.initialPage
}

func (s *pageViewState) Dispose() {
	if s.Widget().controller == nil {
		s.controller.Dispose()
	}
}

func (s *pageViewState) initController() {
	if s.Widget().controller != nil {
		s.controller = s.Widget().controller
	} else {
		s.controller = NewPageController()
	}
}

func (s *pageViewState) DidUpdateWidget(oldWidget *PageView) {
	if oldWidget.controller != s.Widget().controller {
		if oldWidget.controller == nil {
			s.controller.Dispose()
		}
		s.initController()
	}
}

func (s *pageViewState) getDirection(bc BuildContext) painting.AxisDirection {
	if s.Widget().scrollDirection == painting.AxisHorizontal {
		if foundation.DebugMode {
			DebugCheckHasDirectionality(bc)
		}
		axisDirection := painting.TextDirectionToAxisDirection(DirectionalityOf(bc))
		if s.Widget().reverse {
			return painting.FlipAxisDirection(axisDirection)
		}
		return axisDirection
	}
	if s.Widget().reverse {
		return painting.AxisDirectionUp
	}
	return painting.AxisDirectionDown
}

func (s *pageViewState) Build(bc BuildContext) Widget {
	w := s.Widget()
	axisDirection := s.getDirection(bc)
	base := w.physics
	if isNil(base) && w.scrollBehavior != nil {
		base = w.scrollBehavior.GetScrollPhysics(bc)
	}
	if w.pageSnapping {
		base = NewPageScrollPhysics(nil).ApplyTo(base)
	}
	physics := newForceImplicitScrollPhysics(w.allowImplicitScrolling, nil).ApplyTo(base)

	scrollBehavior := w.scrollBehavior
	if scrollBehavior == nil {
		scrollBehavior = ScrollConfigurationOf(bc).CopyWith(ScrollBehaviorCopy{Scrollbars: foundation.Some(false)})
	}
	return NewNotificationListener[ScrollNotification](
		func(notification ScrollNotification) bool {
			if _, ok := notification.(*ScrollUpdateNotification); ok && notification.Depth() == 0 && w.onPageChanged != nil {
				metrics := notification.Metrics().(PageMetrics)
				currentPage := int(math.Round(metrics.Page().Value()))
				if currentPage != s.lastReportedPage {
					s.lastReportedPage = currentPage
					w.onPageChanged(currentPage)
				}
			}
			return false
		},
		NewScrollable(func(bc BuildContext, position rendering.ViewportOffset) Widget {
			return NewViewport(position, []Widget{
				NewSliverFillViewport(w.childrenDelegate, SliverFillViewportOpts{
					ViewportFraction: foundation.Some(s.controller.viewportFraction), PadEnds: foundation.Some(w.padEnds),
					AllowImplicitScrolling: foundation.Some(w.allowImplicitScrolling)}),
			}, ViewportOpts{ScrollCacheExtent: foundation.Some(w.scrollCacheExtent), AxisDirection: axisDirection, ClipBehavior: w.clipBehavior})
		}, ScrollableOpts{
			DragStartBehavior: w.dragStartBehavior, AxisDirection: axisDirection, Controller: s.controller, Physics: physics,
			HitTestBehavior: w.hitTestBehavior, ScrollBehavior: scrollBehavior,
		}))
}
