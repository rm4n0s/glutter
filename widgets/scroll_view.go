package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollViewOpts are the optional parameters of the base of the scroll views, and
// of NewCustomScrollView.
//
// Flutter: widgets/scroll_view.dart ScrollView.new
type ScrollViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Controller reads and controls the scroll position. A view that has one is
	// not primary.
	Controller ScrollController
	// Primary is whether this is the primary scroll view that uses the
	// PrimaryScrollController above; the default is true for a vertical view
	// with no controller.
	Primary foundation.Opt[bool]
	// Physics is how the view responds to the user; the default is the one of the
	// ScrollBehavior (always scrollable for a primary view).
	Physics ScrollPhysics
	// ShrinkWrap is whether the extent of the view along the main axis is the
	// extent of its content.
	ShrinkWrap bool
	// ScrollCacheExtent is how far beyond the viewport the view builds children;
	// the default is 250 logical pixels.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// KeyboardDismissBehavior is how the view dismisses the keyboard; the default
	// is the one of the ScrollBehavior.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// ScrollBehavior describes how the view scrolls; the default is the one of the
	// ScrollConfiguration above.
	ScrollBehavior ScrollBehavior
	// Center is the key of the sliver at the zero scroll offset; the default is
	// the first sliver. It can not be set for a shrink-wrapping view.
	Center foundation.Key
	// Anchor is the relative position of the zero scroll offset, from 0 to 1.
	Anchor float64
	// PaintOrder is the order in which the slivers paint; the default is the first
	// sliver on top.
	PaintOrder rendering.SliverPaintOrder
	// SemanticChildCount is the number of children that contribute semantic
	// information; none lets the scrollable find them.
	SemanticChildCount foundation.Opt[int]
}

// ScrollViewSelf is what a scroll view needs of the type that embeds it: the
// slivers to scroll. A type may also have a BuildViewport method with the
// signature of ScrollView.BuildViewport to replace the viewport.
//
// Flutter: widgets/scroll_view.dart ScrollView.buildSlivers
type ScrollViewSelf interface {
	// BuildSlivers builds the slivers that the viewport shows.
	//
	// Flutter: widgets/scroll_view.dart ScrollView.buildSlivers
	BuildSlivers(bc BuildContext) []Widget
}

// scrollViewViewportBuilder is the optional method of a scroll view that builds
// the viewport itself.
type scrollViewViewportBuilder interface {
	BuildViewport(bc BuildContext, offset rendering.ViewportOffset, axisDirection painting.AxisDirection, slivers []Widget) Widget
}

// ScrollView is the base of the widgets that create a Scrollable and a Viewport
// for their slivers: CustomScrollView, ListView and GridView. A type that embeds
// it implements ScrollViewSelf and calls InitScrollView.
//
// Flutter: widgets/scroll_view.dart ScrollView
type ScrollView struct {
	StatelessWidget
	self                    ScrollViewSelf
	scrollDirection         painting.Axis
	reverse                 bool
	controller              ScrollController
	primary                 foundation.Opt[bool]
	physics                 ScrollPhysics
	scrollBehavior          ScrollBehavior
	shrinkWrap              bool
	center                  foundation.Key
	anchor                  float64
	scrollCacheExtent       foundation.Opt[rendering.ScrollCacheExtent]
	semanticChildCount      foundation.Opt[int]
	paintOrder              rendering.SliverPaintOrder
	dragStartBehavior       gestures.DragStartBehavior
	keyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	restorationID           string
	clipBehavior            ui.Clip
	hitTestBehavior         rendering.HitTestBehavior
}

// InitScrollView sets up the scroll view; self is the outer type. In debug builds
// it panics with the tag ScrollViewPrimaryWithController when the view is primary
// and has a controller, ScrollViewCenterShrinkWrap when it shrink-wraps and has a
// center, ScrollViewAnchorInvalid when the anchor is not between 0 and 1, and
// ScrollViewSemanticChildCountInvalid when the semantic child count is negative.
//
// Flutter: widgets/scroll_view.dart ScrollView.new
func InitScrollView(v *ScrollView, self ScrollViewSelf, o ScrollViewOpts) {
	if foundation.DebugMode {
		switch {
		case !isNil(o.Controller) && o.Primary.OrElse(false):
			panic(errors.New(tagScrollViewPrimaryWithController,
				"primary scroll views obtain their ScrollController via inheritance from a PrimaryScrollController widget; you cannot both set primary to true and pass an explicit controller"))
		case o.ShrinkWrap && o.Center != nil:
			panic(errors.New(tagScrollViewCenterShrinkWrap, "a scroll view that shrink-wraps its content can not have a center"))
		case !(o.Anchor >= 0 && o.Anchor <= 1):
			panic(errors.New(tagScrollViewAnchorInvalid, "the anchor of a scroll view must be between 0 and 1", "anchor", o.Anchor))
		case o.SemanticChildCount.IsSome() && o.SemanticChildCount.Value() < 0:
			panic(errors.New(tagScrollViewSemanticChildCountInvalid, "the semantic child count of a scroll view must not be negative",
				"semanticChildCount", o.SemanticChildCount.Value()))
		}
	}
	if o.ScrollDirection == painting.AxisUnspecified {
		o.ScrollDirection = painting.AxisVertical
	}
	if o.PaintOrder == rendering.SliverPaintOrderUnspecified {
		o.PaintOrder = rendering.SliverPaintOrderFirstIsTop
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorOpaque
	}
	if o.DragStartBehavior == gestures.DragStartBehaviorUnspecified {
		o.DragStartBehavior = gestures.DragStartBehaviorStart
	}
	physics := o.Physics
	if isNil(physics) && (o.Primary.OrElse(false) || (o.Primary.IsNone() && isNil(o.Controller) && o.ScrollDirection == painting.AxisVertical)) {
		physics = NewAlwaysScrollableScrollPhysics(nil)
	}
	v.StatelessWidget = NewStatelessWidgetBase(WidgetOpts{Key: o.Key})
	v.self = self
	v.scrollDirection, v.reverse, v.controller, v.primary, v.physics = o.ScrollDirection, o.Reverse, o.Controller, o.Primary, physics
	v.scrollBehavior, v.shrinkWrap, v.center, v.anchor = o.ScrollBehavior, o.ShrinkWrap, o.Center, o.Anchor
	v.scrollCacheExtent, v.semanticChildCount, v.paintOrder = o.ScrollCacheExtent, o.SemanticChildCount, o.PaintOrder
	v.dragStartBehavior, v.keyboardDismissBehavior, v.restorationID = o.DragStartBehavior, o.KeyboardDismissBehavior, o.RestorationID
	v.clipBehavior, v.hitTestBehavior = o.ClipBehavior, o.HitTestBehavior
}

// ScrollDirection is the axis along which the view scrolls.
//
// Flutter: widgets/scroll_view.dart ScrollView.scrollDirection
func (v *ScrollView) ScrollDirection() painting.Axis { return v.scrollDirection }

// Reverse is whether the view scrolls against the reading direction.
//
// Flutter: widgets/scroll_view.dart ScrollView.reverse
func (v *ScrollView) Reverse() bool { return v.reverse }

// Controller reads and controls the scroll position.
//
// Flutter: widgets/scroll_view.dart ScrollView.controller
func (v *ScrollView) Controller() ScrollController { return v.controller }

// Primary is whether the view was asked to be primary, or none.
//
// Flutter: widgets/scroll_view.dart ScrollView.primary
func (v *ScrollView) Primary() foundation.Opt[bool] { return v.primary }

// Physics is how the view responds to the user, or nil.
//
// Flutter: widgets/scroll_view.dart ScrollView.physics
func (v *ScrollView) Physics() ScrollPhysics { return v.physics }

// ScrollBehavior describes how the view scrolls, or nil.
//
// Flutter: widgets/scroll_view.dart ScrollView.scrollBehavior
func (v *ScrollView) ScrollBehavior() ScrollBehavior { return v.scrollBehavior }

// ShrinkWrap is whether the view takes the extent of its content.
//
// Flutter: widgets/scroll_view.dart ScrollView.shrinkWrap
func (v *ScrollView) ShrinkWrap() bool { return v.shrinkWrap }

// Center is the key of the sliver at the zero scroll offset, or nil.
//
// Flutter: widgets/scroll_view.dart ScrollView.center
func (v *ScrollView) Center() foundation.Key { return v.center }

// Anchor is the relative position of the zero scroll offset.
//
// Flutter: widgets/scroll_view.dart ScrollView.anchor
func (v *ScrollView) Anchor() float64 { return v.anchor }

// ScrollCacheExtent is how far beyond the viewport the view builds children.
//
// Flutter: widgets/scroll_view.dart ScrollView.scrollCacheExtent
func (v *ScrollView) ScrollCacheExtent() foundation.Opt[rendering.ScrollCacheExtent] {
	return v.scrollCacheExtent
}

// SemanticChildCount is the number of children that contribute semantic
// information, or none.
//
// Flutter: widgets/scroll_view.dart ScrollView.semanticChildCount
func (v *ScrollView) SemanticChildCount() foundation.Opt[int] { return v.semanticChildCount }

// PaintOrder is the order in which the slivers paint.
//
// Flutter: widgets/scroll_view.dart ScrollView.paintOrder
func (v *ScrollView) PaintOrder() rendering.SliverPaintOrder { return v.paintOrder }

// DragStartBehavior is where a drag starts.
//
// Flutter: widgets/scroll_view.dart ScrollView.dragStartBehavior
func (v *ScrollView) DragStartBehavior() gestures.DragStartBehavior { return v.dragStartBehavior }

// KeyboardDismissBehavior is how the view dismisses the keyboard, or unspecified.
//
// Flutter: widgets/scroll_view.dart ScrollView.keyboardDismissBehavior
func (v *ScrollView) KeyboardDismissBehavior() ScrollViewKeyboardDismissBehavior {
	return v.keyboardDismissBehavior
}

// RestorationID is the identifier for restoring the scroll offset.
//
// Flutter: widgets/scroll_view.dart ScrollView.restorationId
func (v *ScrollView) RestorationID() string { return v.restorationID }

// ClipBehavior is how the view clips its content.
//
// Flutter: widgets/scroll_view.dart ScrollView.clipBehavior
func (v *ScrollView) ClipBehavior() ui.Clip { return v.clipBehavior }

// HitTestBehavior is how the view behaves during hit testing.
//
// Flutter: widgets/scroll_view.dart ScrollView.hitTestBehavior
func (v *ScrollView) HitTestBehavior() rendering.HitTestBehavior { return v.hitTestBehavior }

// GetDirection is the axis direction of the view: its axis, reversed or not, and
// the ambient text direction for a horizontal one.
//
// Flutter: widgets/scroll_view.dart ScrollView.getDirection
func (v *ScrollView) GetDirection(bc BuildContext) painting.AxisDirection {
	return AxisDirectionFromAxisReverseAndDirectionality(bc, v.scrollDirection, v.reverse)
}

// BuildViewport builds the viewport, or the shrink-wrapping one, for the slivers.
//
// Flutter: widgets/scroll_view.dart ScrollView.buildViewport
func (v *ScrollView) BuildViewport(bc BuildContext, offset rendering.ViewportOffset, axisDirection painting.AxisDirection, slivers []Widget) Widget {
	if foundation.DebugMode && (axisDirection == painting.AxisDirectionUp || axisDirection == painting.AxisDirectionDown) {
		DebugCheckHasDirectionality(bc)
	}
	if v.shrinkWrap {
		return NewShrinkWrappingViewport(offset, slivers, ShrinkWrappingViewportOpts{AxisDirection: axisDirection, PaintOrder: v.paintOrder,
			ClipBehavior: v.clipBehavior, ScrollCacheExtent: v.scrollCacheExtent})
	}
	return NewViewport(offset, slivers, ViewportOpts{AxisDirection: axisDirection, ScrollCacheExtent: v.scrollCacheExtent, Center: v.center,
		Anchor: v.anchor, PaintOrder: v.paintOrder, ClipBehavior: v.clipBehavior})
}

// Build builds the Scrollable with the viewport of the slivers.
//
// Flutter: widgets/scroll_view.dart ScrollView.build
func (v *ScrollView) Build(bc BuildContext) Widget {
	slivers := v.self.BuildSlivers(bc)
	axisDirection := v.GetDirection(bc)

	effectivePrimary := v.primary.OrElse(isNil(v.controller) && PrimaryScrollControllerShouldInherit(bc, v.scrollDirection))
	scrollController := v.controller
	if effectivePrimary {
		scrollController = PrimaryScrollControllerMaybeOf(bc)
	}
	buildViewport := v.BuildViewport
	if b, ok := v.self.(scrollViewViewportBuilder); ok {
		buildViewport = b.BuildViewport
	}
	scrollable := NewScrollable(func(bc BuildContext, offset rendering.ViewportOffset) Widget {
		return buildViewport(bc, offset, axisDirection, slivers)
	}, ScrollableOpts{
		DragStartBehavior: v.dragStartBehavior, AxisDirection: axisDirection, Controller: scrollController, Physics: v.physics,
		ScrollBehavior: v.scrollBehavior, SemanticChildCount: v.semanticChildCount, HitTestBehavior: v.hitTestBehavior,
		ClipBehavior: v.clipBehavior,
	})

	var result Widget = scrollable
	if effectivePrimary && !isNil(scrollController) {
		// Further descendant ScrollViews will not inherit the same PrimaryScrollController.
		result = NewPrimaryScrollControllerNone(scrollable)
	}
	return wrapKeyboardDismiss(bc, v.keyboardDismissBehavior, v.scrollBehavior, result)
}

// CustomScrollViewOpts are the optional parameters of NewCustomScrollView.
//
// Flutter: widgets/scroll_view.dart CustomScrollView.new
type CustomScrollViewOpts = ScrollViewOpts

// CustomScrollView is a ScrollView that creates custom scroll effects using
// slivers.
//
// Flutter: widgets/scroll_view.dart CustomScrollView
type CustomScrollView struct {
	ScrollView
	slivers []Widget
}

// NewCustomScrollView creates a scroll view of the slivers.
//
// Flutter: widgets/scroll_view.dart CustomScrollView.new
func NewCustomScrollView(slivers []Widget, opts ...CustomScrollViewOpts) *CustomScrollView {
	v := &CustomScrollView{slivers: slivers}
	InitScrollView(&v.ScrollView, v, firstOpts(opts))
	return v
}

// Slivers are the slivers that the viewport shows.
//
// Flutter: widgets/scroll_view.dart CustomScrollView.slivers
func (v *CustomScrollView) Slivers() []Widget { return v.slivers }

// BuildSlivers is the list of slivers.
//
// Flutter: widgets/scroll_view.dart CustomScrollView.buildSlivers
func (v *CustomScrollView) BuildSlivers(BuildContext) []Widget { return v.slivers }

// BoxScrollViewSelf is what a box scroll view needs of the type that embeds it.
//
// Flutter: widgets/scroll_view.dart BoxScrollView.buildChildLayout
type BoxScrollViewSelf interface {
	// BuildChildLayout builds the one sliver that the view shows.
	//
	// Flutter: widgets/scroll_view.dart BoxScrollView.buildChildLayout
	BuildChildLayout(bc BuildContext) Widget
}

// BoxScrollView is the base of the scroll views that have a single sliver of
// boxes, with a padding: ListView and GridView.
//
// Flutter: widgets/scroll_view.dart BoxScrollView
type BoxScrollView struct {
	ScrollView
	boxSelf BoxScrollViewSelf
	padding painting.EdgeInsetsGeometry
}

// InitBoxScrollView sets up the box scroll view; self is the outer type.
//
// Flutter: widgets/scroll_view.dart BoxScrollView.new
func InitBoxScrollView(v *BoxScrollView, self interface {
	ScrollViewSelf
	BoxScrollViewSelf
}, padding painting.EdgeInsetsGeometry, o ScrollViewOpts) {
	v.boxSelf = self
	v.padding = padding
	InitScrollView(&v.ScrollView, self, o)
}

// Padding is the amount of space by which to inset the children, or nil.
//
// Flutter: widgets/scroll_view.dart BoxScrollView.padding
func (v *BoxScrollView) Padding() painting.EdgeInsetsGeometry { return v.padding }

// BuildSlivers builds the sliver, padded.
//
// Flutter: widgets/scroll_view.dart BoxScrollView.buildSlivers
func (v *BoxScrollView) BuildSlivers(bc BuildContext) []Widget {
	sliver := v.boxSelf.BuildChildLayout(bc)
	effectivePadding := v.padding
	if v.padding == nil {
		if mediaQuery, ok := MediaQueryMaybeOf(bc); ok {
			zero := foundation.Some(0.0)
			horizontal := mediaQuery.Padding.CopyWith(painting.EdgeInsetsCopy{Top: zero, Bottom: zero})
			vertical := mediaQuery.Padding.CopyWith(painting.EdgeInsetsCopy{Left: zero, Right: zero})
			if v.scrollDirection == painting.AxisVertical {
				effectivePadding = vertical
				sliver = NewMediaQuery(mediaQuery.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(horizontal)}), sliver)
			} else {
				effectivePadding = horizontal
				sliver = NewMediaQuery(mediaQuery.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(vertical)}), sliver)
			}
		}
	}
	if effectivePadding != nil {
		sliver = NewSliverPadding(effectivePadding, sliver)
	}
	return []Widget{sliver}
}

// ListViewOpts are the optional parameters of NewListView and its named
// constructors. Some fields apply to one of them, as the comments say.
//
// Flutter: widgets/scroll_view.dart ListView.new
type ListViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Controller reads and controls the scroll position. A view that has one is
	// not primary.
	Controller ScrollController
	// Primary is whether this is the primary scroll view that uses the
	// PrimaryScrollController above; the default is true for a vertical view
	// with no controller.
	Primary foundation.Opt[bool]
	// Physics is how the view responds to the user; the default is the one of the
	// ScrollBehavior (always scrollable for a primary view).
	Physics ScrollPhysics
	// ShrinkWrap is whether the extent of the view along the main axis is the
	// extent of its content.
	ShrinkWrap bool
	// ScrollCacheExtent is how far beyond the viewport the view builds children;
	// the default is 250 logical pixels.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// KeyboardDismissBehavior is how the view dismisses the keyboard; the default
	// is the one of the ScrollBehavior.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// Padding is the amount of space by which to inset the children; nil lets the
	// MediaQuery padding apply.
	Padding painting.EdgeInsetsGeometry
	// SemanticChildCount is the number of children that contribute semantic
	// information; the default is the number of items.
	SemanticChildCount foundation.Opt[int]
	// ItemExtent forces the children to have the given extent in the scroll
	// direction; it is more efficient. At most one of ItemExtent, ItemExtentBuilder
	// and PrototypeItem can be given.
	ItemExtent foundation.Opt[float64]
	// ItemExtentBuilder gives the extent of each child in the scroll direction.
	ItemExtentBuilder rendering.ItemExtentBuilder
	// PrototypeItem is a widget whose extent in the scroll direction is the one of
	// all the children.
	PrototypeItem Widget
	// AddAutomaticKeepAlives is whether to wrap each child in an AutomaticKeepAlive;
	// the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary; the
	// default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each child in an IndexedSemantics; the
	// default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// FindChildIndexCallback finds the index of the child that has a key
	// (NewListViewBuilder).
	FindChildIndexCallback ChildIndexGetter
	// FindItemIndexCallback finds the index of the item, not counting the
	// separators, that has a key (NewListViewSeparated).
	FindItemIndexCallback ChildIndexGetter
	// ItemCount is the number of items; none means unbounded
	// (NewListViewBuilder).
	ItemCount foundation.Opt[int]
}

func (o ListViewOpts) scrollViewOpts(semanticChildCount foundation.Opt[int]) ScrollViewOpts {
	return ScrollViewOpts{Key: o.Key, ScrollDirection: o.ScrollDirection, Reverse: o.Reverse, Controller: o.Controller, Primary: o.Primary,
		Physics: o.Physics, ShrinkWrap: o.ShrinkWrap, ScrollCacheExtent: o.ScrollCacheExtent, DragStartBehavior: o.DragStartBehavior,
		KeyboardDismissBehavior: o.KeyboardDismissBehavior, RestorationID: o.RestorationID, ClipBehavior: o.ClipBehavior,
		HitTestBehavior: o.HitTestBehavior, SemanticChildCount: semanticChildCount}
}

// ListView is a scrollable list of widgets arranged linearly.
//
// Flutter: widgets/scroll_view.dart ListView
type ListView struct {
	BoxScrollView
	itemExtent        foundation.Opt[float64]
	itemExtentBuilder rendering.ItemExtentBuilder
	prototypeItem     Widget
	childrenDelegate  SliverChildDelegate
}

func newListView(delegate SliverChildDelegate, o ListViewOpts, semanticChildCount foundation.Opt[int]) *ListView {
	if foundation.DebugMode {
		given := 0
		if o.ItemExtent.IsSome() {
			given++
		}
		if o.ItemExtentBuilder != nil {
			given++
		}
		if o.PrototypeItem != nil {
			given++
		}
		if given > 1 {
			panic(errors.New(tagListViewItemExtentConflict, "you can only pass one of itemExtent, prototypeItem and itemExtentBuilder"))
		}
	}
	v := &ListView{itemExtent: o.ItemExtent, itemExtentBuilder: o.ItemExtentBuilder, prototypeItem: o.PrototypeItem, childrenDelegate: delegate}
	InitBoxScrollView(&v.BoxScrollView, v, o.Padding, o.scrollViewOpts(semanticChildCount))
	return v
}

func (o ListViewOpts) listDelegate(children []Widget) *SliverChildListDelegate {
	return NewSliverChildListDelegate(children, SliverChildListDelegateOpts{AddAutomaticKeepAlives: o.AddAutomaticKeepAlives,
		AddRepaintBoundaries: o.AddRepaintBoundaries, AddSemanticIndexes: o.AddSemanticIndexes})
}

// NewListView creates a scrollable, linear array of widgets from an explicit
// list.
//
// Flutter: widgets/scroll_view.dart ListView.new
func NewListView(children []Widget, opts ...ListViewOpts) *ListView {
	o := firstOpts(opts)
	return newListView(o.listDelegate(children), o, semanticOr(o.SemanticChildCount, foundation.Some(len(children))))
}

// NewListViewBuilder creates a scrollable, linear array of widgets that are
// created on demand.
//
// Flutter: widgets/scroll_view.dart ListView.builder
func NewListViewBuilder(itemBuilder NullableIndexedWidgetBuilder, opts ...ListViewOpts) *ListView {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if n, ok := o.ItemCount.Get(); ok {
			if n < 0 {
				panic(errors.New(tagListViewItemCountInvalid, "the item count of a list view must not be negative", "itemCount", n))
			}
			if s, ok := o.SemanticChildCount.Get(); ok && s > n {
				panic(errors.New(tagListViewSemanticChildCountInvalid, "the semantic child count of a list view must not be greater than the item count",
					"semanticChildCount", s, "itemCount", n))
			}
		}
	}
	delegate := NewSliverChildBuilderDelegate(itemBuilder, SliverChildBuilderDelegateOpts{FindChildIndexCallback: o.FindChildIndexCallback,
		ChildCount: o.ItemCount, AddAutomaticKeepAlives: o.AddAutomaticKeepAlives, AddRepaintBoundaries: o.AddRepaintBoundaries,
		AddSemanticIndexes: o.AddSemanticIndexes})
	return newListView(delegate, o, semanticOr(o.SemanticChildCount, o.ItemCount))
}

// NewListViewSeparated creates a scrollable, linear array of widgets with a
// separator between each pair of items.
//
// Flutter: widgets/scroll_view.dart ListView.separated
func NewListViewSeparated(itemBuilder, separatorBuilder NullableIndexedWidgetBuilder, itemCount int, opts ...ListViewOpts) *ListView {
	o := firstOpts(opts)
	if foundation.DebugMode && itemCount < 0 {
		panic(errors.New(tagListViewItemCountInvalid, "the item count of a list view must not be negative", "itemCount", itemCount))
	}
	delegate := separatedDelegate(itemBuilder, separatorBuilder, SliverSeparatedOpts{FindItemIndexCallback: o.FindItemIndexCallback,
		ItemCount: foundation.Some(itemCount), AddAutomaticKeepAlives: o.AddAutomaticKeepAlives, AddRepaintBoundaries: o.AddRepaintBoundaries,
		AddSemanticIndexes: o.AddSemanticIndexes})
	o.ItemExtent, o.ItemExtentBuilder, o.PrototypeItem = foundation.None[float64](), nil, nil
	return newListView(delegate, o, foundation.Some(itemCount))
}

// NewListViewCustom creates a scrollable, linear array of widgets with a custom
// child model.
//
// Flutter: widgets/scroll_view.dart ListView.custom
func NewListViewCustom(childrenDelegate SliverChildDelegate, opts ...ListViewOpts) *ListView {
	o := firstOpts(opts)
	return newListView(childrenDelegate, o, o.SemanticChildCount)
}

// ItemExtent is the extent the children are forced to have, or none.
//
// Flutter: widgets/scroll_view.dart ListView.itemExtent
func (v *ListView) ItemExtent() foundation.Opt[float64] { return v.itemExtent }

// ItemExtentBuilder gives the extent of each child, or nil.
//
// Flutter: widgets/scroll_view.dart ListView.itemExtentBuilder
func (v *ListView) ItemExtentBuilder() rendering.ItemExtentBuilder { return v.itemExtentBuilder }

// PrototypeItem is the widget that gives the extent of the children, or nil.
//
// Flutter: widgets/scroll_view.dart ListView.prototypeItem
func (v *ListView) PrototypeItem() Widget { return v.prototypeItem }

// ChildrenDelegate supplies the children.
//
// Flutter: widgets/scroll_view.dart ListView.childrenDelegate
func (v *ListView) ChildrenDelegate() SliverChildDelegate { return v.childrenDelegate }

// BuildChildLayout builds the sliver list that matches the extent options.
//
// Flutter: widgets/scroll_view.dart ListView.buildChildLayout
func (v *ListView) BuildChildLayout(BuildContext) Widget {
	switch {
	case v.itemExtent.IsSome():
		return NewSliverFixedExtentList(v.childrenDelegate, v.itemExtent.Value())
	case v.itemExtentBuilder != nil:
		return NewSliverVariedExtentList(v.childrenDelegate, v.itemExtentBuilder)
	case v.prototypeItem != nil:
		return NewSliverPrototypeExtentList(v.childrenDelegate, v.prototypeItem)
	}
	return NewSliverList(v.childrenDelegate)
}

// GridViewOpts are the optional parameters of NewGridView and its named
// constructors. Some fields apply to one of them, as the comments say.
//
// Flutter: widgets/scroll_view.dart GridView.new
type GridViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ScrollDirection is the axis along which the view scrolls; the default is
	// AxisVertical.
	ScrollDirection painting.Axis
	// Reverse is whether the view scrolls against the reading direction.
	Reverse bool
	// Controller reads and controls the scroll position. A view that has one is
	// not primary.
	Controller ScrollController
	// Primary is whether this is the primary scroll view that uses the
	// PrimaryScrollController above; the default is true for a vertical view
	// with no controller.
	Primary foundation.Opt[bool]
	// Physics is how the view responds to the user; the default is the one of the
	// ScrollBehavior (always scrollable for a primary view).
	Physics ScrollPhysics
	// ShrinkWrap is whether the extent of the view along the main axis is the
	// extent of its content.
	ShrinkWrap bool
	// ScrollCacheExtent is how far beyond the viewport the view builds children;
	// the default is 250 logical pixels.
	ScrollCacheExtent foundation.Opt[rendering.ScrollCacheExtent]
	// DragStartBehavior is where a drag starts; the default is
	// DragStartBehaviorStart.
	DragStartBehavior gestures.DragStartBehavior
	// KeyboardDismissBehavior is how the view dismisses the keyboard; the default
	// is the one of the ScrollBehavior.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
	// RestorationID is the identifier for restoring the scroll offset; it has no
	// effect until state restoration is ported.
	RestorationID string
	// ClipBehavior is how the view clips its content; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// HitTestBehavior is how the view behaves during hit testing; the default is
	// HitTestBehaviorOpaque.
	HitTestBehavior rendering.HitTestBehavior
	// Padding is the amount of space by which to inset the children; nil lets the
	// MediaQuery padding apply.
	Padding painting.EdgeInsetsGeometry
	// SemanticChildCount is the number of children that contribute semantic
	// information; the default is the number of items.
	SemanticChildCount foundation.Opt[int]
	// AddAutomaticKeepAlives is whether to wrap each child in an AutomaticKeepAlive;
	// the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary; the
	// default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each child in an IndexedSemantics; the
	// default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// FindChildIndexCallback finds the index of the child that has a key
	// (NewGridViewBuilder).
	FindChildIndexCallback ChildIndexGetter
	// ItemCount is the number of items; none means unbounded (NewGridViewBuilder).
	ItemCount foundation.Opt[int]
	// MainAxisSpacing is the number of logical pixels between each child along the
	// main axis (NewGridViewCount, NewGridViewExtent).
	MainAxisSpacing float64
	// CrossAxisSpacing is the number of logical pixels between each child along
	// the cross axis (NewGridViewCount, NewGridViewExtent).
	CrossAxisSpacing float64
	// ChildAspectRatio is the ratio of the cross axis to the main axis extent of
	// each child; the default is 1 (NewGridViewCount, NewGridViewExtent).
	ChildAspectRatio foundation.Opt[float64]
	// MainAxisExtent is the extent of each tile in the main axis, instead of the
	// aspect ratio (NewGridViewCount, NewGridViewExtent).
	MainAxisExtent foundation.Opt[float64]
}

func (o GridViewOpts) scrollViewOpts(semanticChildCount foundation.Opt[int]) ScrollViewOpts {
	return ScrollViewOpts{Key: o.Key, ScrollDirection: o.ScrollDirection, Reverse: o.Reverse, Controller: o.Controller, Primary: o.Primary,
		Physics: o.Physics, ShrinkWrap: o.ShrinkWrap, ScrollCacheExtent: o.ScrollCacheExtent, DragStartBehavior: o.DragStartBehavior,
		KeyboardDismissBehavior: o.KeyboardDismissBehavior, RestorationID: o.RestorationID, ClipBehavior: o.ClipBehavior,
		HitTestBehavior: o.HitTestBehavior, SemanticChildCount: semanticChildCount}
}

func (o GridViewOpts) listDelegate(children []Widget) *SliverChildListDelegate {
	return NewSliverChildListDelegate(children, SliverChildListDelegateOpts{AddAutomaticKeepAlives: o.AddAutomaticKeepAlives,
		AddRepaintBoundaries: o.AddRepaintBoundaries, AddSemanticIndexes: o.AddSemanticIndexes})
}

// GridView is a scrollable, 2D array of widgets.
//
// Flutter: widgets/scroll_view.dart GridView
type GridView struct {
	BoxScrollView
	gridDelegate     rendering.SliverGridDelegate
	childrenDelegate SliverChildDelegate
}

func newGridView(gridDelegate rendering.SliverGridDelegate, delegate SliverChildDelegate, o GridViewOpts, semanticChildCount foundation.Opt[int]) *GridView {
	v := &GridView{gridDelegate: gridDelegate, childrenDelegate: delegate}
	InitBoxScrollView(&v.BoxScrollView, v, o.Padding, o.scrollViewOpts(semanticChildCount))
	return v
}

// NewGridView creates a scrollable, 2D array of widgets with a custom grid
// delegate, from an explicit list.
//
// Flutter: widgets/scroll_view.dart GridView.new
func NewGridView(gridDelegate rendering.SliverGridDelegate, children []Widget, opts ...GridViewOpts) *GridView {
	o := firstOpts(opts)
	return newGridView(gridDelegate, o.listDelegate(children), o, semanticOr(o.SemanticChildCount, foundation.Some(len(children))))
}

// NewGridViewBuilder creates a scrollable, 2D array of widgets that are created
// on demand.
//
// Flutter: widgets/scroll_view.dart GridView.builder
func NewGridViewBuilder(gridDelegate rendering.SliverGridDelegate, itemBuilder NullableIndexedWidgetBuilder, opts ...GridViewOpts) *GridView {
	o := firstOpts(opts)
	delegate := NewSliverChildBuilderDelegate(itemBuilder, SliverChildBuilderDelegateOpts{FindChildIndexCallback: o.FindChildIndexCallback,
		ChildCount: o.ItemCount, AddAutomaticKeepAlives: o.AddAutomaticKeepAlives, AddRepaintBoundaries: o.AddRepaintBoundaries,
		AddSemanticIndexes: o.AddSemanticIndexes})
	return newGridView(gridDelegate, delegate, o, semanticOr(o.SemanticChildCount, o.ItemCount))
}

// NewGridViewCustom creates a scrollable, 2D array of widgets with both a custom
// grid delegate and a custom child model.
//
// Flutter: widgets/scroll_view.dart GridView.custom
func NewGridViewCustom(gridDelegate rendering.SliverGridDelegate, childrenDelegate SliverChildDelegate, opts ...GridViewOpts) *GridView {
	o := firstOpts(opts)
	return newGridView(gridDelegate, childrenDelegate, o, o.SemanticChildCount)
}

// NewGridViewCount creates a scrollable, 2D array of widgets with a fixed number
// of tiles in the cross axis.
//
// Flutter: widgets/scroll_view.dart GridView.count
func NewGridViewCount(crossAxisCount int, children []Widget, opts ...GridViewOpts) *GridView {
	o := firstOpts(opts)
	gridDelegate := rendering.NewSliverGridDelegateWithFixedCrossAxisCount(crossAxisCount, rendering.SliverGridDelegateWithFixedCrossAxisCountOpts{
		MainAxisSpacing: o.MainAxisSpacing, CrossAxisSpacing: o.CrossAxisSpacing, ChildAspectRatio: o.ChildAspectRatio, MainAxisExtent: o.MainAxisExtent})
	return newGridView(gridDelegate, o.listDelegate(children), o, semanticOr(o.SemanticChildCount, foundation.Some(len(children))))
}

// NewGridViewExtent creates a scrollable, 2D array of widgets with tiles that
// each have a maximum cross axis extent.
//
// Flutter: widgets/scroll_view.dart GridView.extent
func NewGridViewExtent(maxCrossAxisExtent float64, children []Widget, opts ...GridViewOpts) *GridView {
	o := firstOpts(opts)
	gridDelegate := rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(maxCrossAxisExtent, rendering.SliverGridDelegateWithMaxCrossAxisExtentOpts{
		MainAxisSpacing: o.MainAxisSpacing, CrossAxisSpacing: o.CrossAxisSpacing, ChildAspectRatio: o.ChildAspectRatio, MainAxisExtent: o.MainAxisExtent})
	return newGridView(gridDelegate, o.listDelegate(children), o, semanticOr(o.SemanticChildCount, foundation.Some(len(children))))
}

// GridDelegate lays out the tiles.
//
// Flutter: widgets/scroll_view.dart GridView.gridDelegate
func (v *GridView) GridDelegate() rendering.SliverGridDelegate { return v.gridDelegate }

// ChildrenDelegate supplies the children.
//
// Flutter: widgets/scroll_view.dart GridView.childrenDelegate
func (v *GridView) ChildrenDelegate() SliverChildDelegate { return v.childrenDelegate }

// BuildChildLayout builds the sliver grid.
//
// Flutter: widgets/scroll_view.dart GridView.buildChildLayout
func (v *GridView) BuildChildLayout(BuildContext) Widget {
	return NewSliverGrid(v.childrenDelegate, v.gridDelegate)
}

// semanticOr is the first option that is set.
func semanticOr(a, b foundation.Opt[int]) foundation.Opt[int] {
	if a.IsSome() {
		return a
	}
	return b
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagListViewItemCountInvalid            = "ListViewItemCountInvalid"
	tagListViewItemExtentConflict          = "ListViewItemExtentConflict"
	tagListViewSemanticChildCountInvalid   = "ListViewSemanticChildCountInvalid"
	tagScrollViewAnchorInvalid             = "ScrollViewAnchorInvalid"
	tagScrollViewCenterShrinkWrap          = "ScrollViewCenterShrinkWrap"
	tagScrollViewPrimaryWithController     = "ScrollViewPrimaryWithController"
	tagScrollViewSemanticChildCountInvalid = "ScrollViewSemanticChildCountInvalid"
)
