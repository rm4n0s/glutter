package rendering

import (
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// TreeSliverNodesAnimation is an animation of the children of a node that expands or
// collapses: the index of the first and of the last of them (in the list of the active nodes) and
// the value of the animation, from 0 (collapsed) to 1 (expanded).
//
// Flutter: rendering/sliver_tree.dart TreeSliverNodesAnimation
type TreeSliverNodesAnimation struct {
	FromIndex, ToIndex int
	Value              float64
}

// TreeSliverNodeParentData is the parent data of the children of a RenderTreeSliver: the
// depth of the node in the tree.
//
// Flutter: rendering/sliver_tree.dart TreeSliverNodeParentData
type TreeSliverNodeParentData struct {
	SliverMultiBoxAdaptorParentData
	// Depth is the depth of the node in the tree: 0 for the root nodes.
	//
	// Flutter: rendering/sliver_tree.dart TreeSliverNodeParentData.depth
	Depth int
}

// TreeSliverIndentationType is how far the children of a node are indented from their
// parent. Make one with TreeSliverIndentationTypeCustom, or use the two that are given.
//
// Flutter: rendering/sliver_tree.dart TreeSliverIndentationType
type TreeSliverIndentationType struct{ value float64 }

// Value is the number of pixels each level is indented by.
//
// Flutter: rendering/sliver_tree.dart TreeSliverIndentationType.value
func (t TreeSliverIndentationType) Value() float64 { return t.value }

// TreeSliverIndentationTypeStandard is the standard indentation, 10 pixels a level.
//
// Flutter: rendering/sliver_tree.dart TreeSliverIndentationType.standard
var TreeSliverIndentationTypeStandard = TreeSliverIndentationType{10.0}

// TreeSliverIndentationTypeNone does not indent the children.
//
// Flutter: rendering/sliver_tree.dart TreeSliverIndentationType.none
var TreeSliverIndentationTypeNone = TreeSliverIndentationType{0.0}

// TreeSliverIndentationTypeCustom indents each level by a number of pixels. It panics with
// TreeSliverIndentationNegative in debug builds when the value is negative.
//
// Flutter: rendering/sliver_tree.dart TreeSliverIndentationType.custom
func TreeSliverIndentationTypeCustom(value float64) TreeSliverIndentationType {
	if foundation.DebugMode && value < 0 {
		panic(errors.New(tagTreeSliverIndentationNegative, "the indentation of a tree must not be negative", "value", value))
	}
	return TreeSliverIndentationType{value}
}

// RenderTreeSliver is the sliver of a TreeSliver: a list of rows of varied extents, each indented by
// its depth, and an animation of the rows that appear or leave when a node expands or collapses.
//
// Deviation: the sliver embeds RenderSliverFixedExtentBoxAdaptor and not RenderSliverVariedExtentList, because
// Go has no virtual dispatch from an embedded base to the outer type (SPEC §12.10); the extents come from
// the same builder.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver
type RenderTreeSliver struct {
	RenderSliverFixedExtentBoxAdaptor
	itemExtentBuilder ItemExtentBuilder
	activeAnimations  map[*foundation.UniqueKey]TreeSliverNodesAnimation
	indentation       float64

	// animationLeadingIndices maps the index of a parent to the key of the animation of its children.
	animationLeadingIndices map[int]*foundation.UniqueKey
	// animationOffsets maps the key of an animation to the fixed distance that it covers, which is
	// worked out when it starts.
	animationOffsets map[*foundation.UniqueKey]float64
	clipHandles      map[*foundation.UniqueKey]*LayerHandle[*ClipRectLayer]
}

var (
	_ FixedExtentLayoutOverrides = (*RenderTreeSliver)(nil)
	_ Sliver                     = (*RenderTreeSliver)(nil)
)

// NewRenderTreeSliver creates the sliver. The map of the active animations is the one that the
// TreeSliver changes (and tells the sliver of with SetActiveAnimations).
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.new
func NewRenderTreeSliver(childManager RenderSliverBoxChildManager, itemExtentBuilder ItemExtentBuilder, activeAnimations map[*foundation.UniqueKey]TreeSliverNodesAnimation, indentation float64) *RenderTreeSliver {
	r := &RenderTreeSliver{
		itemExtentBuilder: itemExtentBuilder, activeAnimations: activeAnimations, indentation: indentation,
		animationLeadingIndices: map[int]*foundation.UniqueKey{},
		animationOffsets:        map[*foundation.UniqueKey]float64{},
		clipHandles:             map[*foundation.UniqueKey]*LayerHandle[*ClipRectLayer]{},
	}
	InitRenderSliverFixedExtentBoxAdaptor(&r.RenderSliverFixedExtentBoxAdaptor, r, childManager)
	return r
}

// ItemExtent is none: the extents come from the builder.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtent
func (r *RenderTreeSliver) ItemExtent() foundation.Opt[float64] { return foundation.None[float64]() }

// ItemExtentBuilder is the function that gives the extent of a row.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtentBuilder
func (r *RenderTreeSliver) ItemExtentBuilder() ItemExtentBuilder { return r.itemExtentBuilder }

// SetItemExtentBuilder changes the builder and marks the sliver as needing layout. Go can
// not compare functions, so it always does.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverVariedExtentList.itemExtentBuilder
func (r *RenderTreeSliver) SetItemExtentBuilder(value ItemExtentBuilder) {
	r.itemExtentBuilder = value
	r.MarkNeedsLayout()
}

// ActiveAnimations are the animations of the nodes that are expanding or collapsing, by the key that the
// TreeSliver gave each.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.activeAnimations
func (r *RenderTreeSliver) ActiveAnimations() map[*foundation.UniqueKey]TreeSliverNodesAnimation {
	return r.activeAnimations
}

// SetActiveAnimations replaces the animations and marks the sliver as needing layout. Go can not compare
// maps, so it always does; the TreeSliver changes the map in place and gives it again.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.activeAnimations
func (r *RenderTreeSliver) SetActiveAnimations(value map[*foundation.UniqueKey]TreeSliverNodesAnimation) {
	r.activeAnimations = value
	r.MarkNeedsLayout()
}

// Indentation is how many pixels each level is indented by.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.indentation
func (r *RenderTreeSliver) Indentation() float64 { return r.indentation }

// SetIndentation changes the indentation and marks the sliver as needing layout when it changed.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.indentation
func (r *RenderTreeSliver) SetIndentation(value float64) {
	if r.indentation == value {
		return
	}
	if foundation.DebugMode && value < 0 {
		panic(errors.New(tagTreeSliverIndentationNegative, "the indentation of a tree must not be negative", "value", value))
	}
	r.indentation = value
	r.MarkNeedsLayout()
}

func (r *RenderTreeSliver) updateAnimationCache() {
	clear(r.animationLeadingIndices)
	for key, animation := range r.activeAnimations {
		r.animationLeadingIndices[animation.FromIndex-1] = key
	}
	// Remove the offsets and the clip layers of the animations that are over.
	for key := range r.animationOffsets {
		if _, ok := r.activeAnimations[key]; !ok {
			delete(r.animationOffsets, key)
		}
	}
	for key, handle := range r.clipHandles {
		if _, ok := r.activeAnimations[key]; !ok {
			handle.SetLayer(nil)
			delete(r.clipHandles, key)
		}
	}
}

// SetupParentData gives the child a TreeSliverNodeParentData if it has none.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.setupParentData
func (r *RenderTreeSliver) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*TreeSliverNodeParentData); !ok {
		child.AsRenderObject().SetParentData(&TreeSliverNodeParentData{})
	}
}

// Dispose lets go of the clip layers.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.dispose
func (r *RenderTreeSliver) Dispose() {
	for key, handle := range r.clipHandles {
		handle.SetLayer(nil)
		delete(r.clipHandles, key)
	}
	r.RenderSliverFixedExtentBoxAdaptor.Dispose()
}

// PerformLayout works out where the animations are, then lays out like a varied extent list. It
// panics with TreeSliverAxisDirection in debug builds when the viewport does not go down.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.performLayout
func (r *RenderTreeSliver) PerformLayout() {
	if foundation.DebugMode && r.SliverConstraints().AxisDirection != painting.AxisDirectionDown {
		panic(errors.New(tagTreeSliverAxisDirection, "TreeSliver is only supported in Viewports with an AxisDirection.down",
			"axisDirection", r.SliverConstraints().AxisDirection))
	}
	r.updateAnimationCache()
	r.RenderSliverFixedExtentBoxAdaptor.PerformLayout()
}

// OverrideMinChildIndexForScrollOffset is the index of the first row at the scroll offset, counting the
// space that the animations take.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.getMinChildIndexForScrollOffset
func (r *RenderTreeSliver) OverrideMinChildIndexForScrollOffset(scrollOffset float64) int {
	return r.childIndexForScrollOffset(scrollOffset)
}

// OverrideMaxChildIndexForScrollOffset is the index of the last row at the scroll offset, counting
// the space that the animations take.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.getMaxChildIndexForScrollOffset
func (r *RenderTreeSliver) OverrideMaxChildIndexForScrollOffset(scrollOffset float64) int {
	return r.childIndexForScrollOffset(scrollOffset)
}

func (r *RenderTreeSliver) childIndexForScrollOffset(scrollOffset float64) int {
	if scrollOffset == 0.0 {
		return 0
	}
	position, totalAnimationOffset := 0.0, 0.0
	index := 0
	childCount, hasCount := r.childManager.EstimatedChildCount().Get()
	for position < scrollOffset {
		if hasCount && index > childCount-1 {
			break
		}
		itemExtent, ok := r.itemExtentBuilder(index, r.LayoutDimensions()).Get()
		if !ok {
			break
		}
		if key, animating := r.animationLeadingIndices[index]; animating {
			if _, known := r.animationOffsets[key]; !known {
				// The distance that this block covers over the life of the animation is not worked out yet.
				r.computeAnimationOffsetFor(key, position)
			}
			// The offset, as far as the animation has gone.
			totalAnimationOffset += r.animationOffsets[key] * (1 - r.activeAnimations[key].Value)
		}
		position += itemExtent - totalAnimationOffset
		index++
	}
	return index - 1
}

func (r *RenderTreeSliver) computeAnimationOffsetFor(key *foundation.UniqueKey, position float64) {
	animation := r.activeAnimations[key]
	c := r.SliverConstraints()
	targetPosition := c.ScrollOffset + c.RemainingCacheExtent
	currentPosition := position
	totalAnimatingOffset := 0.0
	// Only the rows that would be visible, or in the cache extent, are animated, unless all of them fit.
	for index := animation.FromIndex; index <= animation.ToIndex && currentPosition < targetPosition; index++ {
		itemExtent := r.itemExtentBuilder(index, r.LayoutDimensions()).Value()
		totalAnimatingOffset += itemExtent
		currentPosition += itemExtent
	}
	// For the life of the animation, every row after the first one is offset by this distance times
	// the animation value, even if more rows scroll into view: the distance has to stay the same for a smooth
	// animation.
	r.animationOffsets[key] = totalAnimatingOffset
}

// ChildCrossAxisPosition is the indentation of the row: its depth times the indentation.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.childCrossAxisPosition
func (r *RenderTreeSliver) ChildCrossAxisPosition(child RenderObject) float64 {
	return float64(child.ParentData().(*TreeSliverNodeParentData).Depth) * r.indentation
}

// OverrideIndexToLayoutOffset is the layout offset of the row at the index, less what the animations
// that start before it have not covered yet.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.indexToLayoutOffset
func (r *RenderTreeSliver) OverrideIndexToLayoutOffset(index int) float64 {
	position, totalAnimationOffset := 0.0, 0.0
	childCount, hasCount := r.childManager.EstimatedChildCount().Get()
	for current := 0; current < index; current++ {
		if hasCount && current > childCount-1 {
			break
		}
		itemExtent, ok := r.itemExtentBuilder(current, r.LayoutDimensions()).Get()
		if !ok {
			break
		}
		if key, animating := r.animationLeadingIndices[current]; animating {
			if _, known := r.animationOffsets[key]; foundation.DebugMode && !known {
				panic(errors.New(tagTreeSliverAnimationOffsetMissing, "the distance of an animation is worked out before the layout offsets are asked for", "index", current))
			}
			totalAnimationOffset += r.animationOffsets[key] * (1 - r.activeAnimations[key].Value)
		}
		position += itemExtent
	}
	return position - totalAnimationOffset
}

// OverrideComputeMaxScrollOffset is the extent of all the rows.
//
// Flutter: rendering/sliver_fixed_extent_list.dart RenderSliverFixedExtentBoxAdaptor.computeMaxScrollOffset
func (r *RenderTreeSliver) OverrideComputeMaxScrollOffset(SliverConstraints) float64 {
	offset := 0.0
	for i := 0; i < r.childManager.ChildCount(); i++ {
		extent, ok := r.itemExtentBuilder(i, r.LayoutDimensions()).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

// paintSegment is a run of rows that is painted together, from leadingIndex to trailingIndex.
type paintSegment struct{ leadingIndex, trailingIndex int }

// Paint paints the rows indented by their depth. While nodes expand or collapse, the rows that
// follow an animation are clipped to the space that is left, so that they do not draw over the rows
// that are leaving.
//
// Flutter: rendering/sliver_tree.dart RenderTreeSliver.paint
func (r *RenderTreeSliver) Paint(context *PaintingContext, offset ui.Offset) {
	if isNil(r.FirstChild()) {
		return
	}
	c := r.SliverConstraints()
	nextChild := r.FirstChild()
	paintUpTo := func(index int, startWith Box, context *PaintingContext, offset ui.Offset) {
		child := startWith
		for !isNil(child) && r.IndexOf(child) <= index {
			mainAxisDelta := r.ChildMainAxisPosition(child)
			parentData := child.ParentData().(*TreeSliverNodeParentData)
			childOffset := ui.Offset{
				Dx: float64(parentData.Depth) * r.indentation,
				Dy: parentData.LayoutOffset.Value() - c.ScrollOffset,
			}.Add(offset)
			// A row whose visible interval does not meet the paint extent is hidden.
			if mainAxisDelta < c.RemainingPaintExtent && mainAxisDelta+r.PaintExtentOf(child) > 0 {
				context.PaintChild(child, childOffset)
			}
			child = r.ChildAfter(child)
		}
		nextChild = child
	}

	if len(r.animationLeadingIndices) == 0 {
		paintUpTo(r.IndexOf(r.LastChild()), r.FirstChild(), context, offset)
		return
	}

	// Animating: separate the segments to clip for any overlap.
	leadingIndex := r.IndexOf(r.FirstChild())
	animationIndices := make([]int, 0, len(r.animationLeadingIndices))
	for index := range r.animationLeadingIndices {
		animationIndices = append(animationIndices, index)
	}
	slices.Sort(animationIndices)
	var segments []paintSegment
	for _, trailingIndex := range animationIndices {
		segments = append(segments, paintSegment{leadingIndex, trailingIndex})
		leadingIndex = trailingIndex + 1
	}
	segments = append(segments, paintSegment{leadingIndex, r.IndexOf(r.LastChild())})

	// The first segment is painted without a clip, the others with a clip layer.
	paintUpTo(segments[0].trailingIndex, nextChild, context, offset)
	for _, segment := range segments[1:] {
		// The rectangle goes from the trailing edge of the parent (the row before the segment) to
		// the trailing edge of the segment. The leading edge of the segment cannot be used, because it is moving.
		parentIndex := max(segment.leadingIndex-1, 0)
		leadingOffset := r.OverrideIndexToLayoutOffset(parentIndex)
		if parentIndex != 0 {
			leadingOffset += r.itemExtentBuilder(parentIndex, r.LayoutDimensions()).Value()
		}
		trailingOffset := r.OverrideIndexToLayoutOffset(segment.trailingIndex) + r.itemExtentBuilder(segment.trailingIndex, r.LayoutDimensions()).Value()
		rect := ui.NewRectFromPoints(ui.Offset{Dx: 0, Dy: leadingOffset}, ui.Offset{Dx: c.CrossAxisExtent, Dy: trailingOffset})
		// The clip layer is kept under the key of the animation of the parent, except for the odd segment.
		key := r.animationLeadingIndices[parentIndex]
		handle := r.clipHandles[key]
		if handle == nil {
			handle = NewLayerHandle[*ClipRectLayer](nil)
			r.clipHandles[key] = handle
		}
		first := nextChild
		last := segment.trailingIndex
		handle.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, rect, func(context *PaintingContext, offset ui.Offset) {
			paintUpTo(last, first, context, offset)
		}, PushClipRectOpts{OldLayer: handle.Layer()}))
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTreeSliverAnimationOffsetMissing = "TreeSliverAnimationOffsetMissing"
	tagTreeSliverAxisDirection          = "TreeSliverAxisDirection"
	tagTreeSliverIndentationNegative    = "TreeSliverIndentationNegative"
)
