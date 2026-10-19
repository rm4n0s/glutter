package rendering

import (
	"sort"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// EstimateMaxScrollOffsetOpts are the parameters of
// RenderSliverBoxChildManager.EstimateMaxScrollOffset.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.estimateMaxScrollOffset
type EstimateMaxScrollOffsetOpts struct {
	// FirstIndex is the index of the first live child.
	FirstIndex foundation.Opt[int]
	// LastIndex is the index of the last live child.
	LastIndex foundation.Opt[int]
	// LeadingScrollOffset is the scroll offset of the leading edge of the first
	// live child.
	LeadingScrollOffset foundation.Opt[float64]
	// TrailingScrollOffset is the scroll offset of the trailing edge of the last
	// live child.
	TrailingScrollOffset foundation.Opt[float64]
}

// RenderSliverBoxChildManager creates and removes the box children of a
// RenderSliverMultiBoxAdaptor, lazily, while the sliver lays itself out.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager
type RenderSliverBoxChildManager interface {
	// CreateChild creates the child at the index and inserts it after the given
	// child (nil for the start). If it can not create a child, the sliver is at
	// the end of its list. The manager inserts it with the adaptor's Insert.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.createChild
	CreateChild(index int, after Box)
	// RemoveChild removes the child, which was created by CreateChild.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.removeChild
	RemoveChild(child Box)
	// EstimateMaxScrollOffset estimates the total scroll extent of the list, from
	// the extent of the children that are live.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.estimateMaxScrollOffset
	EstimateMaxScrollOffset(constraints SliverConstraints, opts EstimateMaxScrollOffsetOpts) float64
	// ChildCount is the number of children the list has, or the number the
	// manager knows of if it does not know the last.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.childCount
	ChildCount() int
	// EstimatedChildCount is the number of children if it is known, none if the
	// list is infinite or its length is not known.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.estimatedChildCount
	EstimatedChildCount() foundation.Opt[int]
	// DidAdoptChild is called when the adaptor adopts the child, so that the
	// manager can record the index in the parent data.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.didAdoptChild
	DidAdoptChild(child Box)
	// SetDidUnderflow says whether the list ran out of children before it filled
	// the viewport.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.setDidUnderflow
	SetDidUnderflow(value bool)
	// DidStartLayout is called at the start of the layout of the adaptor.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.didStartLayout
	DidStartLayout()
	// DidFinishLayout is called at the end of the layout of the adaptor.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.didFinishLayout
	DidFinishLayout()
	// DebugAssertChildListLocked is true, or panics, in debug builds, when the
	// child list is not locked.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.debugAssertChildListLocked
	DebugAssertChildListLocked() bool
}

// RenderSliverBoxChildManagerDefaults is the default of the methods of a
// RenderSliverBoxChildManager that Flutter gives a body: an embedding manager
// does not have to implement them.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager
type RenderSliverBoxChildManagerDefaults struct{}

// EstimatedChildCount is none.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.estimatedChildCount
func (RenderSliverBoxChildManagerDefaults) EstimatedChildCount() foundation.Opt[int] {
	return foundation.None[int]()
}

// DidStartLayout does nothing.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.didStartLayout
func (RenderSliverBoxChildManagerDefaults) DidStartLayout() {}

// DidFinishLayout does nothing.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.didFinishLayout
func (RenderSliverBoxChildManagerDefaults) DidFinishLayout() {}

// DebugAssertChildListLocked is true.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverBoxChildManager.debugAssertChildListLocked
func (RenderSliverBoxChildManagerDefaults) DebugAssertChildListLocked() bool { return true }

// KeepAliveParentDataMixin is the state of a parent data that lets its child ask
// the parent to keep it alive when it scrolls out of view. Embed it in the
// parent data type.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart KeepAliveParentDataMixin
type KeepAliveParentDataMixin struct {
	// KeepAlive is whether the child asks the parent to keep it alive while it is
	// not visible.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart KeepAliveParentDataMixin.keepAlive
	KeepAlive bool
}

// KeepAliveData is the mixin.
func (k *KeepAliveParentDataMixin) KeepAliveData() *KeepAliveParentDataMixin { return k }

// KeepAliveParentData is a parent data that embeds KeepAliveParentDataMixin and
// knows whether the parent keeps the child alive at the moment.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart KeepAliveParentDataMixin
type KeepAliveParentData interface {
	ParentData
	// KeepAliveData is the embedded mixin.
	KeepAliveData() *KeepAliveParentDataMixin
	// KeptAlive is whether the parent keeps the child alive at the moment, while
	// it is out of view.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart KeepAliveParentDataMixin.keptAlive
	KeptAlive() bool
}

// SliverMultiBoxAdaptorParentData is the parent data of the children of a
// RenderSliverMultiBoxAdaptor: the layout offset, the index and the links to the
// siblings.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart SliverMultiBoxAdaptorParentData
type SliverMultiBoxAdaptorParentData struct {
	SliverLogicalParentData
	ContainerParentData[Box]
	KeepAliveParentDataMixin
	// Index is the index of the child in the list; none before the manager sets it.
	//
	// Flutter: rendering/sliver_multi_box_adaptor.dart SliverMultiBoxAdaptorParentData.index
	Index foundation.Opt[int]

	keptAlive bool
}

// KeptAlive is whether the parent keeps the child alive, out of view.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart SliverMultiBoxAdaptorParentData.keptAlive
func (d *SliverMultiBoxAdaptorParentData) KeptAlive() bool { return d.keptAlive }

// Detach panics in debug builds if the sibling links are still set.
//
// Flutter: rendering/object.dart ContainerParentDataMixin.detach
func (d *SliverMultiBoxAdaptorParentData) Detach() { d.checkDetach() }

// String is the index, the keep alive flag and the layout offset.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart SliverMultiBoxAdaptorParentData.toString
func (d *SliverMultiBoxAdaptorParentData) String() string {
	index := "null"
	if i, ok := d.Index.Get(); ok {
		index = strconv.Itoa(i)
	}
	keepAlive := ""
	if d.KeepAlive {
		keepAlive = "keepAlive; "
	}
	return "index=" + index + "; " + keepAlive + d.SliverLogicalParentData.String()
}

func (d *SliverMultiBoxAdaptorParentData) multiBox() *SliverMultiBoxAdaptorParentData { return d }

// sliverMultiBoxHolder is a ParentData that is or embeds a
// SliverMultiBoxAdaptorParentData.
type sliverMultiBoxHolder interface {
	ParentData
	multiBox() *SliverMultiBoxAdaptorParentData
}

// SliverMultiBoxAdaptorParentDataOf is the SliverMultiBoxAdaptorParentData of a
// child (the parent data of the child is that or embeds it).
//
// Flutter: no counterpart (the cast `child.parentData! as SliverMultiBoxAdaptorParentData`)
func SliverMultiBoxAdaptorParentDataOf(child RenderObject) *SliverMultiBoxAdaptorParentData {
	return child.ParentData().(sliverMultiBoxHolder).multiBox()
}

// keepAliveBucket maps indices to kept alive children, in insertion order, as
// Dart's map does.
type keepAliveBucket struct {
	items map[int]Box
	order []int
}

func (k *keepAliveBucket) get(index int) (Box, bool) {
	b, ok := k.items[index]
	return b, ok
}

func (k *keepAliveBucket) set(index int, child Box) {
	if k.items == nil {
		k.items = map[int]Box{}
	}
	if _, ok := k.items[index]; !ok {
		k.order = append(k.order, index)
	}
	k.items[index] = child
}

func (k *keepAliveBucket) remove(index int) {
	if _, ok := k.items[index]; !ok {
		return
	}
	delete(k.items, index)
	for i, v := range k.order {
		if v == index {
			k.order = append(k.order[:i], k.order[i+1:]...)
			break
		}
	}
}

func (k *keepAliveBucket) values() []Box {
	out := make([]Box, 0, len(k.order))
	for _, i := range k.order {
		out = append(out, k.items[i])
	}
	return out
}

func (k *keepAliveBucket) containsValue(child Box) bool {
	for _, v := range k.items {
		if v == child {
			return true
		}
	}
	return false
}

func (k *keepAliveBucket) clear() {
	k.items = nil
	k.order = nil
}

// RenderSliverMultiBoxAdaptor is the base of the slivers that have a list of box
// children which are created lazily, by a child manager, as the sliver lays
// itself out, and may be kept alive out of view. The concrete type embeds it,
// calls InitRenderSliverMultiBoxAdaptor, and implements PerformLayout.
//
// Flutter's adoptChild override, which tells the manager about a child that is
// not kept alive, runs inside Insert, because Go has no virtual methods.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor
type RenderSliverMultiBoxAdaptor struct {
	RenderSliver
	ContainerRenderObject[Box]

	childManager                RenderSliverBoxChildManager
	keepAliveBucket             keepAliveBucket
	debugDanglingKeepAlives     []Box
	debugChildIntegrityDisabled bool
}

// InitRenderSliverMultiBoxAdaptor sets the outer object and the child manager.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.new
func InitRenderSliverMultiBoxAdaptor(a *RenderSliverMultiBoxAdaptor, self Sliver, childManager RenderSliverBoxChildManager) {
	a.childManager = childManager
	InitRenderSliver(&a.RenderSliver, self)
	InitContainerRenderObject(&a.ContainerRenderObject, self)
}

// SetupParentData gives the child a SliverMultiBoxAdaptorParentData if it has
// none.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.setupParentData
func (a *RenderSliverMultiBoxAdaptor) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(sliverMultiBoxHolder); !ok {
		child.AsRenderObject().SetParentData(&SliverMultiBoxAdaptorParentData{})
	}
}

// ChildManager is the object that creates and removes the children.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.childManager
func (a *RenderSliverMultiBoxAdaptor) ChildManager() RenderSliverBoxChildManager {
	return a.childManager
}

// DebugChildIntegrityEnabled is whether the debug checks of the child list are
// on.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.debugChildIntegrityEnabled
func (a *RenderSliverMultiBoxAdaptor) DebugChildIntegrityEnabled() bool {
	return !a.debugChildIntegrityDisabled
}

// SetDebugChildIntegrityEnabled turns the debug checks of the child list on or
// off; in debug builds it checks that the children are in order and, when they
// are on, that no kept alive child is dangling, and panics with the tags
// SliverChildOrderInvalid and SliverDanglingKeepAlives otherwise.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.debugChildIntegrityEnabled
func (a *RenderSliverMultiBoxAdaptor) SetDebugChildIntegrityEnabled(enabled bool) {
	if !foundation.DebugMode {
		return
	}
	a.debugChildIntegrityDisabled = !enabled
	a.debugVerifyChildOrder()
	if enabled && len(a.debugDanglingKeepAlives) != 0 {
		panic(errors.New("SliverDanglingKeepAlives", "the adaptor has kept alive children that are not in its bucket",
			"object", foundation.DescribeIdentity(a.self)))
	}
}

func (a *RenderSliverMultiBoxAdaptor) debugVerifyChildOrder() {
	if !foundation.DebugMode || a.debugChildIntegrityDisabled {
		return
	}
	child := a.FirstChild()
	for !isNil(child) {
		index := a.IndexOf(child)
		child = a.ChildAfter(child)
		if !isNil(child) && a.IndexOf(child) <= index {
			panic(errors.New("SliverChildOrderInvalid", "the children of the adaptor are not in increasing order of their index",
				"object", foundation.DescribeIdentity(a.self), "index", index))
		}
	}
}

func (a *RenderSliverMultiBoxAdaptor) debugRemoveDangling(child Box) {
	for i, c := range a.debugDanglingKeepAlives {
		if c == child {
			a.debugDanglingKeepAlives = append(a.debugDanglingKeepAlives[:i], a.debugDanglingKeepAlives[i+1:]...)
			return
		}
	}
}

// Insert adds the child after another child, or at the start if after is nil,
// and tells the manager that it adopted it.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.insert
func (a *RenderSliverMultiBoxAdaptor) Insert(child, after Box) {
	if foundation.DebugMode && a.keepAliveBucket.containsValue(child) {
		panic(errors.New("SliverInsertKeptAlive", "a child that is kept alive was inserted again",
			"object", foundation.DescribeIdentity(a.self), "child", foundation.DescribeIdentity(child)))
	}
	a.insertAdopting(child, after, func() {
		// adoptChild override
		if !SliverMultiBoxAdaptorParentDataOf(child).keptAlive {
			a.childManager.DidAdoptChild(child)
		}
	})
	a.debugVerifyChildOrder()
}

// InsertObject is Insert, for the widgets library.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.insertRenderObjectChild
func (a *RenderSliverMultiBoxAdaptor) InsertObject(child, after RenderObject) {
	a.Insert(a.typed(child), a.typedOrZero(after))
}

// Move moves the child to after another child (or the start). A kept alive child
// is only handed to the manager again, to update its slot.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.move
func (a *RenderSliverMultiBoxAdaptor) Move(child, after Box) {
	data := SliverMultiBoxAdaptorParentDataOf(child)
	if !data.keptAlive {
		a.ContainerRenderObject.Move(child, after)
		a.childManager.DidAdoptChild(child) // updates the slot in the parentData
		// Its slot may change even if super.move does not change the position.
		a.MarkNeedsLayout()
		return
	}
	// If the child in the bucket is not current child, that means someone has
	// already moved and replaced current child, and we cannot remove this child.
	if bucketed, ok := a.keepAliveBucket.get(data.Index.Value()); ok && bucketed == child {
		a.keepAliveBucket.remove(data.Index.Value())
	}
	if foundation.DebugMode {
		a.debugRemoveDangling(child)
	}
	a.childManager.DidAdoptChild(child)
	// If there is an existing child in the bucket, that means that the child
	// was kept alive and then replaced; it will be removed in collectGarbage.
	if foundation.DebugMode {
		if existing, ok := a.keepAliveBucket.get(data.Index.Value()); ok {
			a.debugDanglingKeepAlives = append(a.debugDanglingKeepAlives, existing)
		}
	}
	a.keepAliveBucket.set(data.Index.Value(), child)
}

// MoveObject is Move, for the widgets library.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.moveRenderObjectChild
func (a *RenderSliverMultiBoxAdaptor) MoveObject(child, after RenderObject) {
	a.Move(a.typed(child), a.typedOrZero(after))
}

// Remove removes the child, from the list or from the bucket of kept alive
// children.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.remove
func (a *RenderSliverMultiBoxAdaptor) Remove(child Box) {
	data := SliverMultiBoxAdaptorParentDataOf(child)
	if !data.keptAlive {
		a.ContainerRenderObject.Remove(child)
		return
	}
	if foundation.DebugMode {
		a.debugRemoveDangling(child)
	}
	a.keepAliveBucket.remove(data.Index.Value())
	a.DropChild(child)
}

// RemoveObject is Remove, for the widgets library.
//
// Flutter: widgets/sliver.dart SliverMultiBoxAdaptorElement.removeRenderObjectChild
func (a *RenderSliverMultiBoxAdaptor) RemoveObject(child RenderObject) { a.Remove(a.typed(child)) }

// RemoveAll removes every child, the kept alive ones included.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.removeAll
func (a *RenderSliverMultiBoxAdaptor) RemoveAll() {
	a.ContainerRenderObject.RemoveAll()
	for _, child := range a.keepAliveBucket.values() {
		a.DropChild(child)
	}
	a.keepAliveBucket.clear()
}

func (a *RenderSliverMultiBoxAdaptor) createOrObtainChild(index int, after Box) {
	a.InvokeLayoutCallback(func(Constraints) {
		if child, ok := a.keepAliveBucket.get(index); ok {
			a.keepAliveBucket.remove(index)
			data := SliverMultiBoxAdaptorParentDataOf(child)
			a.DropChild(child)
			child.AsRenderObject().SetParentData(data)
			a.Insert(child, after)
			data.keptAlive = false
		} else {
			a.childManager.CreateChild(index, after)
		}
	})
}

func (a *RenderSliverMultiBoxAdaptor) destroyOrCacheChild(child Box) {
	data := SliverMultiBoxAdaptorParentDataOf(child)
	if data.KeepAlive {
		a.Remove(child)
		a.keepAliveBucket.set(data.Index.Value(), child)
		child.AsRenderObject().SetParentData(data)
		a.AdoptChild(child) // not the override that tells the manager: super.adoptChild
		data.keptAlive = true
	} else {
		a.childManager.RemoveChild(child)
	}
}

// Attach attaches the kept alive children too.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.attach
func (a *RenderSliverMultiBoxAdaptor) Attach(owner *PipelineOwner) {
	a.RenderSliver.Attach(owner)
	for _, child := range a.keepAliveBucket.values() {
		child.Attach(owner)
	}
}

// Detach detaches the kept alive children too.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.detach
func (a *RenderSliverMultiBoxAdaptor) Detach() {
	a.RenderSliver.Detach()
	for _, child := range a.keepAliveBucket.values() {
		child.Detach()
	}
}

// RedepthChildren updates the depth of the kept alive children too.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.redepthChildren
func (a *RenderSliverMultiBoxAdaptor) RedepthChildren() {
	a.RenderSliver.RedepthChildren()
	for _, child := range a.keepAliveBucket.values() {
		a.RedepthChild(child)
	}
}

// VisitChildren visits the live children and the kept alive ones.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.visitChildren
func (a *RenderSliverMultiBoxAdaptor) VisitChildren(visitor RenderObjectVisitor) {
	a.RenderSliver.VisitChildren(visitor)
	for _, child := range a.keepAliveBucket.values() {
		visitor(child)
	}
}

// VisitChildrenForSemantics visits the live children, not the kept alive ones.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.visitChildrenForSemantics
func (a *RenderSliverMultiBoxAdaptor) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	a.RenderSliver.VisitChildren(visitor)
}

// SemanticBounds are the paint bounds of the first child if the sliver is not
// visible.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.semanticBounds
func (a *RenderSliverMultiBoxAdaptor) SemanticBounds() ui.Rect {
	first := a.FirstChild()
	if a.HasGeometry() && !a.Geometry().Visible && !isNil(first) && first.HasSize() {
		return first.PaintBounds()
	}
	return a.RenderSliver.SemanticBounds()
}

// AddInitialChildOpts are the optional parameters of AddInitialChild.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.addInitialChild
type AddInitialChildOpts struct {
	// Index is the index of the child to create; the default is 0.
	Index int
	// LayoutOffset is its layout offset; the default is 0.
	LayoutOffset float64
}

// AddInitialChild creates the first child, which must be done when there are
// none. It returns false if the manager has no child at the index, after
// telling it so.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.addInitialChild
func (a *RenderSliverMultiBoxAdaptor) AddInitialChild(opts ...AddInitialChildOpts) bool {
	o := firstOpts(opts)
	a.childManager.DebugAssertChildListLocked()
	a.createOrObtainChild(o.Index, nil)
	if first := a.FirstChild(); !isNil(first) {
		SliverMultiBoxAdaptorParentDataOf(first).LayoutOffset = foundation.Some(o.LayoutOffset)
		return true
	}
	a.childManager.SetDidUnderflow(true)
	return false
}

// InsertAndLayoutLeadingChild creates and lays out the child before the first
// one. It returns nil, after telling the manager, if there is none.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.insertAndLayoutLeadingChild
func (a *RenderSliverMultiBoxAdaptor) InsertAndLayoutLeadingChild(childConstraints BoxConstraints, opts ...LayoutOpts) Box {
	a.childManager.DebugAssertChildListLocked()
	index := a.IndexOf(a.FirstChild()) - 1
	a.createOrObtainChild(index, nil)
	if first := a.FirstChild(); !isNil(first) && a.IndexOf(first) == index {
		first.Layout(childConstraints, firstOpts(opts))
		return first
	}
	a.childManager.SetDidUnderflow(true)
	return nil
}

// InsertAndLayoutChild creates and lays out the child after the given one. It
// returns nil, after telling the manager, if there is none.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.insertAndLayoutChild
func (a *RenderSliverMultiBoxAdaptor) InsertAndLayoutChild(childConstraints BoxConstraints, after Box, opts ...LayoutOpts) Box {
	a.childManager.DebugAssertChildListLocked()
	index := a.IndexOf(after) + 1
	a.createOrObtainChild(index, after)
	if child := a.ChildAfter(after); !isNil(child) && a.IndexOf(child) == index {
		child.Layout(childConstraints, firstOpts(opts))
		return child
	}
	a.childManager.SetDidUnderflow(true)
	return nil
}

// CalculateLeadingGarbage is how many children from the start have an index
// below the first index.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.calculateLeadingGarbage
func (a *RenderSliverMultiBoxAdaptor) CalculateLeadingGarbage(firstIndex int) int {
	walker := a.FirstChild()
	leadingGarbage := 0
	for !isNil(walker) && a.IndexOf(walker) < firstIndex {
		leadingGarbage++
		walker = a.ChildAfter(walker)
	}
	return leadingGarbage
}

// CalculateTrailingGarbage is how many children from the end have an index
// above the last index.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.calculateTrailingGarbage
func (a *RenderSliverMultiBoxAdaptor) CalculateTrailingGarbage(lastIndex int) int {
	walker := a.LastChild()
	trailingGarbage := 0
	for !isNil(walker) && a.IndexOf(walker) > lastIndex {
		trailingGarbage++
		walker = a.ChildBefore(walker)
	}
	return trailingGarbage
}

// CollectGarbage removes the given numbers of children from the start and the end
// of the list, or keeps them alive if they ask to, and removes the kept alive
// children that no longer ask.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.collectGarbage
func (a *RenderSliverMultiBoxAdaptor) CollectGarbage(leadingGarbage, trailingGarbage int) {
	a.childManager.DebugAssertChildListLocked()
	if foundation.DebugMode && a.ChildCount() < leadingGarbage+trailingGarbage {
		panic(errors.New("SliverGarbageInvalid", "more children were collected than the adaptor has",
			"object", foundation.DescribeIdentity(a.self), "leading", leadingGarbage, "trailing", trailingGarbage, "childCount", a.ChildCount()))
	}
	a.InvokeLayoutCallback(func(Constraints) {
		for ; leadingGarbage > 0; leadingGarbage-- {
			a.destroyOrCacheChild(a.FirstChild())
		}
		for ; trailingGarbage > 0; trailingGarbage-- {
			a.destroyOrCacheChild(a.LastChild())
		}
		// Ask the child manager to remove the children that are no longer kept
		// alive. (This should cause _keepAliveBucket to change, so we have to
		// prepare our list ahead of time.)
		for _, child := range a.keepAliveBucket.values() {
			if !SliverMultiBoxAdaptorParentDataOf(child).KeepAlive {
				a.childManager.RemoveChild(child)
			}
		}
	})
}

// IndexOf is the index of the child, which the manager set.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.indexOf
func (a *RenderSliverMultiBoxAdaptor) IndexOf(child Box) int {
	return SliverMultiBoxAdaptorParentDataOf(child).Index.Value()
}

// PaintExtentOf is the extent of the child along the main axis.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.paintExtentOf
func (a *RenderSliverMultiBoxAdaptor) PaintExtentOf(child Box) float64 {
	if a.SliverConstraints().Axis() == painting.AxisHorizontal {
		return child.Size().Width
	}
	return child.Size().Height
}

// HitTestChildren hit tests the children, from the last.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.hitTestChildren
func (a *RenderSliverMultiBoxAdaptor) HitTestChildren(result *SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	boxResult := NewBoxHitTestResultWrap(result.HitTestResult)
	for child := a.LastChild(); !isNil(child); child = a.ChildBefore(child) {
		if SliverHitTestBoxChild(a.debugSelf(), boxResult, child, mainAxisPosition, crossAxisPosition) {
			return true
		}
	}
	return false
}

// ChildMainAxisPosition is the scroll offset of the child minus the scroll
// offset of the sliver.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.childMainAxisPosition
func (a *RenderSliverMultiBoxAdaptor) ChildMainAxisPosition(child RenderObject) float64 {
	return a.ChildScrollOffset(child).Value() - a.SliverConstraints().ScrollOffset
}

// ChildScrollOffset is the layout offset of the child.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.childScrollOffset
func (a *RenderSliverMultiBoxAdaptor) ChildScrollOffset(child RenderObject) foundation.Opt[float64] {
	return SliverMultiBoxAdaptorParentDataOf(child).LayoutOffset
}

// PaintsChild is true for a child that has an index and is not in the bucket of
// kept alive children.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.paintsChild
func (a *RenderSliverMultiBoxAdaptor) PaintsChild(child RenderObject) bool {
	holder, ok := child.ParentData().(sliverMultiBoxHolder)
	if !ok {
		return false
	}
	data := holder.multiBox()
	if !data.Index.IsSome() {
		return false
	}
	_, kept := a.keepAliveBucket.get(data.Index.Value())
	return !kept
}

// ApplyPaintTransform translates by the position of the child; for a child that
// is not painted it is the zero matrix.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.applyPaintTransform
func (a *RenderSliverMultiBoxAdaptor) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if !a.PaintsChild(child) {
		transform.SetZero()
		return
	}
	SliverApplyPaintTransformForBoxChild(a.debugSelf(), child.(Box), transform)
}

// Paint paints the children that are in the viewport.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.paint
func (a *RenderSliverMultiBoxAdaptor) Paint(context *PaintingContext, offset ui.Offset) {
	if isNil(a.FirstChild()) {
		return
	}
	self := a.debugSelf()
	c, g := a.SliverConstraints(), a.Geometry()
	var mainAxisUnit, crossAxisUnit, originOffset ui.Offset
	var addExtent bool
	switch ApplyGrowthDirectionToAxisDirection(c.AxisDirection, c.GrowthDirection) {
	case painting.AxisDirectionUp:
		mainAxisUnit, crossAxisUnit = ui.Offset{Dy: -1}, ui.Offset{Dx: 1}
		originOffset = offset.Add(ui.Offset{Dy: g.PaintExtent})
		addExtent = true
	case painting.AxisDirectionRight:
		mainAxisUnit, crossAxisUnit = ui.Offset{Dx: 1}, ui.Offset{Dy: 1}
		originOffset = offset
	case painting.AxisDirectionLeft:
		mainAxisUnit, crossAxisUnit = ui.Offset{Dx: -1}, ui.Offset{Dy: 1}
		originOffset = offset.Add(ui.Offset{Dx: g.PaintExtent})
		addExtent = true
	default:
		mainAxisUnit, crossAxisUnit = ui.Offset{Dy: 1}, ui.Offset{Dx: 1}
		originOffset = offset
	}
	for child := a.FirstChild(); !isNil(child); child = a.ChildAfter(child) {
		mainAxisDelta := self.ChildMainAxisPosition(child)
		crossAxisDelta := self.ChildCrossAxisPosition(child)
		childOffset := ui.Offset{
			Dx: originOffset.Dx + mainAxisUnit.Dx*mainAxisDelta + crossAxisUnit.Dx*crossAxisDelta,
			Dy: originOffset.Dy + mainAxisUnit.Dy*mainAxisDelta + crossAxisUnit.Dy*crossAxisDelta,
		}
		if addExtent {
			extent := a.PaintExtentOf(child)
			childOffset = childOffset.Add(ui.Offset{Dx: mainAxisUnit.Dx * extent, Dy: mainAxisUnit.Dy * extent})
		}
		// If the child's visible interval (mainAxisDelta, mainAxisDelta +
		// paintExtentOf(child)) does not intersect the paint extent interval
		// (0, constraints.remainingPaintExtent), it's likely that the child
		// hasn't been laid out yet, so don't paint it.
		if mainAxisDelta < c.RemainingPaintExtent && mainAxisDelta+a.PaintExtentOf(child) > 0 {
			context.PaintChild(child, childOffset)
		}
	}
}

// DebugFillProperties adds the indices of the live children.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.debugFillProperties
func (a *RenderSliverMultiBoxAdaptor) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	a.RenderSliver.DebugFillProperties(b)
	message := "no children current live"
	if first := a.FirstChild(); !isNil(first) {
		message = "currently live children: " + strconv.Itoa(a.IndexOf(first)) + " to " + strconv.Itoa(a.IndexOf(a.LastChild()))
	}
	b.Add(foundation.NewDiagnosticsNodeMessage(message))
}

// DebugAssertChildListIsNonEmptyAndContiguous panics, in debug builds, with the
// tag SliverChildListInvalid if there are no children or their indices are not
// consecutive. It returns true.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.debugAssertChildListIsNonEmptyAndContiguous
func (a *RenderSliverMultiBoxAdaptor) DebugAssertChildListIsNonEmptyAndContiguous() bool {
	if !foundation.DebugMode {
		return true
	}
	first := a.FirstChild()
	if isNil(first) {
		panic(errors.New("SliverChildListInvalid", "the child list of the adaptor is empty", "object", foundation.DescribeIdentity(a.self)))
	}
	index := a.IndexOf(first)
	for child := a.ChildAfter(first); !isNil(child); child = a.ChildAfter(child) {
		index++
		if a.IndexOf(child) != index {
			panic(errors.New("SliverChildListInvalid", "the indices of the children of the adaptor are not contiguous",
				"object", foundation.DescribeIdentity(a.self), "index", index))
		}
	}
	return true
}

// DebugDescribeChildren describes the live children by index, and the kept alive
// ones after them.
//
// Flutter: rendering/sliver_multi_box_adaptor.dart RenderSliverMultiBoxAdaptor.debugDescribeChildren
func (a *RenderSliverMultiBoxAdaptor) DebugDescribeChildren() []foundation.DiagnosticsNode {
	var children []foundation.DiagnosticsNode
	for child := a.FirstChild(); !isNil(child); child = a.ChildAfter(child) {
		data := SliverMultiBoxAdaptorParentDataOf(child)
		children = append(children, foundation.ToDiagnosticsNode(child,
			foundation.DiagnosticsNodeOpts{Name: foundation.Some("child with index " + indexString(data.Index))}))
	}
	if len(a.keepAliveBucket.items) > 0 {
		indices := make([]int, 0, len(a.keepAliveBucket.items))
		for i := range a.keepAliveBucket.items {
			indices = append(indices, i)
		}
		sort.Ints(indices)
		for _, index := range indices {
			children = append(children, foundation.ToDiagnosticsNode(a.keepAliveBucket.items[index], foundation.DiagnosticsNodeOpts{
				Name:  foundation.Some("child with index " + strconv.Itoa(index) + " (kept alive but not laid out)"),
				Style: foundation.DiagnosticsTreeStyleOffstage,
			}))
		}
	}
	return children
}

func indexString(i foundation.Opt[int]) string {
	if v, ok := i.Get(); ok {
		return strconv.Itoa(v)
	}
	return "null"
}
