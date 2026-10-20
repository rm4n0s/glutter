package widgets

import (
	"strconv"

	"github.com/rm4n0s/glutter/foundation"
)

// NullableIndexedWidgetBuilder builds the widget at an index, or returns nil when
// there is no widget at that index.
//
// Flutter: widgets/framework.dart NullableIndexedWidgetBuilder
type NullableIndexedWidgetBuilder func(bc BuildContext, index int) Widget

// IndexedWidgetBuilder builds the widget at an index.
//
// Flutter: widgets/framework.dart IndexedWidgetBuilder
type IndexedWidgetBuilder func(bc BuildContext, index int) Widget

// SemanticIndexCallback gives the semantic index of the widget at a local index:
// none for a widget that is not a semantic item (a separator).
//
// Flutter: widgets/scroll_delegate.dart SemanticIndexCallback
type SemanticIndexCallback func(widget Widget, localIndex int) foundation.Opt[int]

// ChildIndexGetter finds the index of the child that has the key, or none.
//
// Flutter: widgets/scroll_delegate.dart ChildIndexGetter
type ChildIndexGetter func(key foundation.Key) foundation.Opt[int]

func defaultSemanticIndexCallback(_ Widget, localIndex int) foundation.Opt[int] {
	return foundation.Some(localIndex)
}

// SliverChildDelegate supplies the children of a sliver list or grid: it builds
// the child at an index, lazily, and knows how many there are.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate
type SliverChildDelegate interface {
	// Build returns the child at the index, or nil if there is none.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.build
	Build(bc BuildContext, index int) Widget
	// EstimatedChildCount is the number of children, or none if it is not known.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.estimatedChildCount
	EstimatedChildCount() foundation.Opt[int]
	// EstimateMaxScrollOffset is an estimate of the maximum scroll extent, or
	// none to let the sliver extrapolate it.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.estimateMaxScrollOffset
	EstimateMaxScrollOffset(firstIndex, lastIndex int, leadingScrollOffset, trailingScrollOffset float64) foundation.Opt[float64]
	// DidFinishLayout is called at the end of the layout of the sliver, with the
	// indices of the first and last children that are live.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.didFinishLayout
	DidFinishLayout(firstIndex, lastIndex int)
	// ShouldRebuild is whether the children must be built again, as this
	// delegate replaces the old one.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.shouldRebuild
	ShouldRebuild(oldDelegate SliverChildDelegate) bool
	// FindIndexByKey is the new index of the child that has the key, or none.
	//
	// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.findIndexByKey
	FindIndexByKey(key foundation.Key) foundation.Opt[int]
}

// SliverChildDelegateBase gives the optional methods of a SliverChildDelegate
// their default: embed it in a delegate of your own.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate
type SliverChildDelegateBase struct{}

// EstimatedChildCount is none.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.estimatedChildCount
func (SliverChildDelegateBase) EstimatedChildCount() foundation.Opt[int] {
	return foundation.None[int]()
}

// EstimateMaxScrollOffset is none.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.estimateMaxScrollOffset
func (SliverChildDelegateBase) EstimateMaxScrollOffset(int, int, float64, float64) foundation.Opt[float64] {
	return foundation.None[float64]()
}

// DidFinishLayout does nothing.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.didFinishLayout
func (SliverChildDelegateBase) DidFinishLayout(int, int) {}

// FindIndexByKey is none.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.findIndexByKey
func (SliverChildDelegateBase) FindIndexByKey(foundation.Key) foundation.Opt[int] {
	return foundation.None[int]()
}

// saltedValueKey is the key that the delegates give to their children, made from
// the key of the child the builder returned, so that it differs from it.
//
// Flutter: widgets/scroll_delegate.dart _SaltedValueKey
type saltedValueKey struct {
	foundation.LocalKeyBase
	value foundation.Key
}

func (k saltedValueKey) String() string { return "[salted " + k.value.String() + "]" }

func unsalted(key foundation.Key) foundation.Key {
	if s, ok := key.(saltedValueKey); ok {
		return s.value
	}
	return key
}

// wrapSliverChild adds what the delegates add around a child: a repaint boundary,
// the semantic index and the keep alive, in a KeyedSubtree with the salted key.
func wrapSliverChild(child Widget, index int, addRepaintBoundaries, addSemanticIndexes, addAutomaticKeepAlives bool,
	semanticIndexCallback SemanticIndexCallback, semanticIndexOffset int) Widget {
	var key foundation.Key
	if k := child.Key(); k != nil {
		key = saltedValueKey{value: k}
	}
	if addRepaintBoundaries {
		child = NewRepaintBoundary(child)
	}
	if addSemanticIndexes {
		if semanticIndex, ok := semanticIndexCallback(child, index).Get(); ok {
			child = NewIndexedSemantics(semanticIndex+semanticIndexOffset, child)
		}
	}
	if addAutomaticKeepAlives {
		child = NewAutomaticKeepAlive(newSelectionKeepAlive(child))
	}
	return NewKeyedSubtree(child, WidgetOpts{Key: key})
}

func describeChildCount(d SliverChildDelegate) string {
	if n, ok := d.EstimatedChildCount().Get(); ok {
		return "estimated child count: " + strconv.Itoa(n)
	}
	return ""
}

// SliverChildBuilderDelegateOpts are the optional parameters of
// NewSliverChildBuilderDelegate.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.new
type SliverChildBuilderDelegateOpts struct {
	// FindChildIndexCallback finds the index of the child that has a key, so that
	// children that move keep their state.
	FindChildIndexCallback ChildIndexGetter
	// ChildCount is the number of children; none means unbounded, until the
	// builder returns nil.
	ChildCount foundation.Opt[int]
	// AddAutomaticKeepAlives is whether to wrap each child in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary;
	// the default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each child in an IndexedSemantics;
	// the default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// SemanticIndexCallback gives the semantic index of a child; the default is
	// its index.
	SemanticIndexCallback SemanticIndexCallback
	// SemanticIndexOffset is added to the semantic index of every child.
	SemanticIndexOffset int
}

// SliverChildBuilderDelegate is a delegate that builds its children with a
// function, lazily, as the list scrolls.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate
type SliverChildBuilderDelegate struct {
	SliverChildDelegateBase
	builder                NullableIndexedWidgetBuilder
	findChildIndexCallback ChildIndexGetter
	childCount             foundation.Opt[int]
	addAutomaticKeepAlives bool
	addRepaintBoundaries   bool
	addSemanticIndexes     bool
	semanticIndexCallback  SemanticIndexCallback
	semanticIndexOffset    int
}

// NewSliverChildBuilderDelegate creates a delegate that builds children with the
// builder.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.new
func NewSliverChildBuilderDelegate(builder NullableIndexedWidgetBuilder, opts ...SliverChildBuilderDelegateOpts) *SliverChildBuilderDelegate {
	o := firstOpts(opts)
	if o.SemanticIndexCallback == nil {
		o.SemanticIndexCallback = defaultSemanticIndexCallback
	}
	return &SliverChildBuilderDelegate{
		builder: builder, findChildIndexCallback: o.FindChildIndexCallback, childCount: o.ChildCount,
		addAutomaticKeepAlives: o.AddAutomaticKeepAlives.OrElse(true), addRepaintBoundaries: o.AddRepaintBoundaries.OrElse(true),
		addSemanticIndexes: o.AddSemanticIndexes.OrElse(true), semanticIndexCallback: o.SemanticIndexCallback,
		semanticIndexOffset: o.SemanticIndexOffset,
	}
}

// Builder is the function that builds the children.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.builder
func (d *SliverChildBuilderDelegate) Builder() NullableIndexedWidgetBuilder { return d.builder }

// ChildCount is the number of children, or none.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.childCount
func (d *SliverChildBuilderDelegate) ChildCount() foundation.Opt[int] { return d.childCount }

// AddAutomaticKeepAlives is whether each child is wrapped in an AutomaticKeepAlive.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.addAutomaticKeepAlives
func (d *SliverChildBuilderDelegate) AddAutomaticKeepAlives() bool { return d.addAutomaticKeepAlives }

// AddRepaintBoundaries is whether each child is wrapped in a RepaintBoundary.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.addRepaintBoundaries
func (d *SliverChildBuilderDelegate) AddRepaintBoundaries() bool { return d.addRepaintBoundaries }

// AddSemanticIndexes is whether each child is wrapped in an IndexedSemantics.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.addSemanticIndexes
func (d *SliverChildBuilderDelegate) AddSemanticIndexes() bool { return d.addSemanticIndexes }

// SemanticIndexOffset is added to the semantic index of every child.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.semanticIndexOffset
func (d *SliverChildBuilderDelegate) SemanticIndexOffset() int { return d.semanticIndexOffset }

// FindIndexByKey asks the callback for the index of the child.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.findIndexByKey
func (d *SliverChildBuilderDelegate) FindIndexByKey(key foundation.Key) foundation.Opt[int] {
	if d.findChildIndexCallback == nil {
		return foundation.None[int]()
	}
	return d.findChildIndexCallback(unsalted(key))
}

// Build builds the child at the index and wraps it. It returns nil for an index
// outside of the child count, and when the builder returns nil. A panic of the
// builder becomes an error widget, once reported.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.build
func (d *SliverChildBuilderDelegate) Build(bc BuildContext, index int) Widget {
	if count, ok := d.childCount.Get(); index < 0 || (ok && index >= count) {
		return nil
	}
	child := guardedBuild(bc, func() Widget { return d.builder(bc, index) })
	if child == nil {
		return nil
	}
	return wrapSliverChild(child, index, d.addRepaintBoundaries, d.addSemanticIndexes, d.addAutomaticKeepAlives,
		d.semanticIndexCallback, d.semanticIndexOffset)
}

// EstimatedChildCount is the child count.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.estimatedChildCount
func (d *SliverChildBuilderDelegate) EstimatedChildCount() foundation.Opt[int] { return d.childCount }

// ShouldRebuild is true.
//
// Flutter: widgets/scroll_delegate.dart SliverChildBuilderDelegate.shouldRebuild
func (d *SliverChildBuilderDelegate) ShouldRebuild(SliverChildDelegate) bool { return true }

// String describes the delegate.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.toString
func (d *SliverChildBuilderDelegate) String() string {
	return "SliverChildBuilderDelegate(" + describeChildCount(d) + ")"
}

// guardedBuild runs a builder of children and turns its panic into an error
// widget, after reporting it, as Dart's builder exceptions are. Without an error
// reporter the panic goes on.
func guardedBuild(bc BuildContext, build func() Widget) Widget {
	owner := bc.Owner()
	if owner == nil || owner.OnError == nil {
		return build()
	}
	var result Widget
	var failure error
	foundation.Guard(func(err error) { failure = err }, foundation.BuildPanicked, "building", nil, func() { result = build() })
	if failure != nil {
		owner.OnError(failure)
		return owner.errorWidgetFor(failure)
	}
	return result
}

// SliverChildListDelegateOpts are the optional parameters of
// NewSliverChildListDelegate.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.new
type SliverChildListDelegateOpts struct {
	// AddAutomaticKeepAlives is whether to wrap each child in an
	// AutomaticKeepAlive; the default is true.
	AddAutomaticKeepAlives foundation.Opt[bool]
	// AddRepaintBoundaries is whether to wrap each child in a RepaintBoundary;
	// the default is true.
	AddRepaintBoundaries foundation.Opt[bool]
	// AddSemanticIndexes is whether to wrap each child in an IndexedSemantics;
	// the default is true.
	AddSemanticIndexes foundation.Opt[bool]
	// SemanticIndexCallback gives the semantic index of a child; the default is
	// its index.
	SemanticIndexCallback SemanticIndexCallback
	// SemanticIndexOffset is added to the semantic index of every child.
	SemanticIndexOffset int
}

// SliverChildListDelegate is a delegate that supplies children from an explicit
// list.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate
type SliverChildListDelegate struct {
	SliverChildDelegateBase
	children               []Widget
	addAutomaticKeepAlives bool
	addRepaintBoundaries   bool
	addSemanticIndexes     bool
	semanticIndexCallback  SemanticIndexCallback
	semanticIndexOffset    int
	// keyToIndex is the lazily filled map from keys to indices; the nil key's
	// entry is the next index to look at. nil for a fixed (constant) delegate.
	keyToIndex map[foundation.Key]int
	scanned    int
}

// NewSliverChildListDelegate creates a delegate that supplies the children. The
// delegate finds children by key lazily, and caches what it learns; the list must
// not change.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.new
func NewSliverChildListDelegate(children []Widget, opts ...SliverChildListDelegateOpts) *SliverChildListDelegate {
	d := NewSliverChildListDelegateFixed(children, opts...)
	d.keyToIndex = map[foundation.Key]int{}
	return d
}

// NewSliverChildListDelegateFixed creates a delegate for a list that never
// changes: it can not find children by key.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.fixed
func NewSliverChildListDelegateFixed(children []Widget, opts ...SliverChildListDelegateOpts) *SliverChildListDelegate {
	o := firstOpts(opts)
	if o.SemanticIndexCallback == nil {
		o.SemanticIndexCallback = defaultSemanticIndexCallback
	}
	return &SliverChildListDelegate{
		children: children, addAutomaticKeepAlives: o.AddAutomaticKeepAlives.OrElse(true),
		addRepaintBoundaries: o.AddRepaintBoundaries.OrElse(true), addSemanticIndexes: o.AddSemanticIndexes.OrElse(true),
		semanticIndexCallback: o.SemanticIndexCallback, semanticIndexOffset: o.SemanticIndexOffset,
	}
}

// Children is the list of children.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.children
func (d *SliverChildListDelegate) Children() []Widget { return d.children }

// AddAutomaticKeepAlives is whether each child is wrapped in an AutomaticKeepAlive.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.addAutomaticKeepAlives
func (d *SliverChildListDelegate) AddAutomaticKeepAlives() bool { return d.addAutomaticKeepAlives }

// AddRepaintBoundaries is whether each child is wrapped in a RepaintBoundary.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.addRepaintBoundaries
func (d *SliverChildListDelegate) AddRepaintBoundaries() bool { return d.addRepaintBoundaries }

// AddSemanticIndexes is whether each child is wrapped in an IndexedSemantics.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.addSemanticIndexes
func (d *SliverChildListDelegate) AddSemanticIndexes() bool { return d.addSemanticIndexes }

func (d *SliverChildListDelegate) findChildIndex(key foundation.Key) foundation.Opt[int] {
	if d.keyToIndex == nil {
		return foundation.None[int]()
	}
	if index, ok := d.keyToIndex[key]; ok {
		return foundation.Some(index)
	}
	index := d.scanned
	for index < len(d.children) {
		child := d.children[index]
		if k := child.Key(); k != nil {
			d.keyToIndex[k] = index
		}
		if child.Key() == key {
			d.scanned = index + 1
			return foundation.Some(index)
		}
		index++
	}
	d.scanned = index
	return foundation.None[int]()
}

// FindIndexByKey looks for the child that has the key.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.findIndexByKey
func (d *SliverChildListDelegate) FindIndexByKey(key foundation.Key) foundation.Opt[int] {
	return d.findChildIndex(unsalted(key))
}

// Build wraps the child at the index, or returns nil outside of the list.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.build
func (d *SliverChildListDelegate) Build(_ BuildContext, index int) Widget {
	if index < 0 || index >= len(d.children) {
		return nil
	}
	return wrapSliverChild(d.children[index], index, d.addRepaintBoundaries, d.addSemanticIndexes, d.addAutomaticKeepAlives,
		d.semanticIndexCallback, d.semanticIndexOffset)
}

// EstimatedChildCount is the length of the list.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.estimatedChildCount
func (d *SliverChildListDelegate) EstimatedChildCount() foundation.Opt[int] {
	return foundation.Some(len(d.children))
}

// ShouldRebuild is whether the list of children is not the same one: Go slices
// are compared by their first element and length.
//
// Flutter: widgets/scroll_delegate.dart SliverChildListDelegate.shouldRebuild
func (d *SliverChildListDelegate) ShouldRebuild(oldDelegate SliverChildDelegate) bool {
	old := oldDelegate.(*SliverChildListDelegate)
	if len(d.children) != len(old.children) {
		return true
	}
	if len(d.children) == 0 {
		return false
	}
	return &d.children[0] != &old.children[0]
}

// String describes the delegate.
//
// Flutter: widgets/scroll_delegate.dart SliverChildDelegate.toString
func (d *SliverChildListDelegate) String() string {
	return "SliverChildListDelegate(" + describeChildCount(d) + ")"
}
