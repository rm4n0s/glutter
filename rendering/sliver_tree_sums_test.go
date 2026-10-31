package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// refTree is the code that RenderTreeSliver had before step 29.40: the lookups of
// the index and of the layout offset add up the extents of the builder from row 0 on
// each call. The tests ask it and the sliver the same questions in the same order.
type refTree struct {
	builder rendering.ItemExtentBuilder
	manager rendering.RenderSliverBoxChildManager
	sliver  *rendering.RenderTreeSliver
	active  map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation
	leading map[int]*foundation.UniqueKey
	offsets map[*foundation.UniqueKey]float64
}

func newRefTree(sliver *rendering.RenderTreeSliver, builder rendering.ItemExtentBuilder, manager rendering.RenderSliverBoxChildManager, active map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation) *refTree {
	return &refTree{builder: builder, manager: manager, sliver: sliver, active: active,
		leading: map[int]*foundation.UniqueKey{}, offsets: map[*foundation.UniqueKey]float64{}}
}

func (r *refTree) dims() rendering.SliverLayoutDimensions { return r.sliver.LayoutDimensions() }

func (r *refTree) updateAnimationCache() {
	clear(r.leading)
	for key, animation := range r.active {
		r.leading[animation.FromIndex-1] = key
	}
	for key := range r.offsets {
		if _, ok := r.active[key]; !ok {
			delete(r.offsets, key)
		}
	}
}

func (r *refTree) childIndexForScrollOffset(scrollOffset float64) int {
	if scrollOffset == 0.0 {
		return 0
	}
	position, totalAnimationOffset := 0.0, 0.0
	index := 0
	childCount, hasCount := r.manager.EstimatedChildCount().Get()
	for position < scrollOffset {
		if hasCount && index > childCount-1 {
			break
		}
		itemExtent, ok := r.builder(index, r.dims()).Get()
		if !ok {
			break
		}
		if key, animating := r.leading[index]; animating {
			if _, known := r.offsets[key]; !known {
				r.computeAnimationOffsetFor(key, position)
			}
			totalAnimationOffset += r.offsets[key] * (1 - r.active[key].Value)
		}
		position += itemExtent - totalAnimationOffset
		index++
	}
	return index - 1
}

func (r *refTree) computeAnimationOffsetFor(key *foundation.UniqueKey, position float64) {
	animation := r.active[key]
	c := r.sliver.SliverConstraints()
	targetPosition := c.ScrollOffset + c.RemainingCacheExtent
	currentPosition := position
	totalAnimatingOffset := 0.0
	for index := animation.FromIndex; index <= animation.ToIndex && currentPosition < targetPosition; index++ {
		itemExtent := r.builder(index, r.dims()).Value()
		totalAnimatingOffset += itemExtent
		currentPosition += itemExtent
	}
	r.offsets[key] = totalAnimatingOffset
}

// indexToLayoutOffset reports a panic (an animation that is not worked out) as ok = false.
func (r *refTree) indexToLayoutOffset(index int) (offset float64, ok bool) {
	defer func() {
		if recover() != nil {
			offset, ok = 0, false
		}
	}()
	position, totalAnimationOffset := 0.0, 0.0
	childCount, hasCount := r.manager.EstimatedChildCount().Get()
	for current := 0; current < index; current++ {
		if hasCount && current > childCount-1 {
			break
		}
		itemExtent, got := r.builder(current, r.dims()).Get()
		if !got {
			break
		}
		if key, animating := r.leading[current]; animating {
			if _, known := r.offsets[key]; !known {
				panic("unknown")
			}
			totalAnimationOffset += r.offsets[key] * (1 - r.active[key].Value)
		}
		position += itemExtent
	}
	return position - totalAnimationOffset, true
}

func (r *refTree) computeMaxScrollOffset() float64 {
	offset := 0.0
	for i := 0; i < r.manager.ChildCount(); i++ {
		extent, ok := r.builder(i, r.dims()).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

// treeFixture is a RenderTreeSliver in a viewport of 300 by 300 pixels.
type treeFixture struct {
	sliver  *rendering.RenderTreeSliver
	manager *lazyRowsManager
	root    *rendering.RenderViewport
	stack   *renderhost.Stack
	active  map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation
	ref     *refTree
	calls   map[int]int
	extents []float64
}

func newTreeFixture(extents []float64, count int, known bool, offset float64, animations []rendering.TreeSliverNodesAnimation) (*treeFixture, []*foundation.UniqueKey) {
	f := &treeFixture{manager: newLazyRowsManager(count, known), extents: extents, calls: map[int]int{},
		active: map[*foundation.UniqueKey]rendering.TreeSliverNodesAnimation{}}
	var keys []*foundation.UniqueKey
	for _, a := range animations {
		key := foundation.NewUniqueKey()
		keys = append(keys, key)
		f.active[key] = a
	}
	build := func(i int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		f.calls[i]++
		if i < 0 || i >= len(extents) {
			return foundation.None[float64]()
		}
		return foundation.Some(extents[i])
	}
	f.sliver = rendering.NewRenderTreeSliver(f.manager, build, f.active, 10)
	f.manager.insert, f.manager.remove = f.sliver.Insert, f.sliver.Remove
	f.root = noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(offset), f.sliver)
	f.stack = renderhost.New(renderhost.Opts{Size: ui.Size{Width: 300, Height: 300}})
	f.stack.Host.DiscardScenes = true
	f.ref = newRefTree(f.sliver, answerFrom(extents), f.manager, f.active)
	f.stack.SetRoot(f.root)
	f.layout()
	return f, keys
}

// layout lays out a frame, and does for the reference what PerformLayout does with the lookups.
func (f *treeFixture) layout() {
	f.sliver.SetActiveAnimations(f.active)
	f.stack.Pump()
	f.ref.updateAnimationCache()
	c := f.sliver.SliverConstraints()
	first := c.ScrollOffset + c.CacheOrigin
	f.ref.childIndexForScrollOffset(first)
	if end := first + c.RemainingCacheExtent; !math.IsInf(end, 0) {
		f.ref.childIndexForScrollOffset(end)
	}
}

// TestTreeSliverSumsEqualTheOldAddingUp compares the lookups of the tree, row by row, with the
// code they replace, in every state of the animations: not started, running, almost done, a
// node that collapses, several nodes at once.
func TestTreeSliverSumsEqualTheOldAddingUp(t *testing.T) {
	extents := randomExtents(80, 71)
	for i := range extents {
		extents[i] = float64(int(extents[i])%40 + 5) // 5 to 44
	}
	extents[10], extents[11] = 0, 0
	anim := func(from, to int, value float64) rendering.TreeSliverNodesAnimation {
		return rendering.TreeSliverNodesAnimation{FromIndex: from, ToIndex: to, Value: value}
	}
	tests := []struct {
		name       string
		count      int
		known      bool
		offset     float64
		animations []rendering.TreeSliverNodesAnimation
		// values are the values of the animations at each frame
		values [][]float64
	}{
		{"no animation", 80, true, 0, nil, [][]float64{nil}},
		{"a node that expands", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(3, 6, 0)}, [][]float64{{0}, {0.25}, {0.5}, {0.75}, {1}}},
		{"a node that collapses", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(3, 6, 1)}, [][]float64{{1}, {0.75}, {0.5}, {0.25}, {0}}},
		{"two nodes at once", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(3, 5, 0), anim(10, 14, 1)}, [][]float64{{0, 1}, {0.25, 0.75}, {0.5, 0.5}, {1, 0}}},
		{"nodes that follow one another", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(4, 5, 0.5), anim(6, 9, 0.25), anim(7, 8, 0.75)}, [][]float64{{0.5, 0.25, 0.75}, {0.25, 0.5, 0.5}}},
		{"a node at the first row", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(1, 3, 0.25)}, [][]float64{{0.25}, {0.5}}},
		{"a node near the end", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(75, 79, 0.5)}, [][]float64{{0.5}, {0}}},
		{"scrolled past a node", 80, true, 400, []rendering.TreeSliverNodesAnimation{anim(3, 6, 0.5), anim(30, 33, 0.25)}, [][]float64{{0.5, 0.25}, {0.75, 0.5}}},
		{"scrolled into a node", 80, true, 90, []rendering.TreeSliverNodesAnimation{anim(3, 8, 0.5), anim(9, 12, 0.5)}, [][]float64{{0.5, 0.5}, {1, 0.25}}},
		{"unknown count", 80, false, 0, []rendering.TreeSliverNodesAnimation{anim(3, 6, 0.5), anim(20, 22, 0.25)}, [][]float64{{0.5, 0.25}, {0.25, 0.5}}},
		{"the builder stops before a node", 80, true, 0, []rendering.TreeSliverNodesAnimation{anim(3, 6, 0.5), anim(20, 22, 0.25)}, [][]float64{{0.5, 0.25}}},
		{"the count is before a node", 15, true, 0, []rendering.TreeSliverNodesAnimation{anim(3, 6, 0.5), anim(20, 22, 0.25)}, [][]float64{{0.5, 0.25}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows := extents
			if tc.name == "the builder stops before a node" {
				rows = extents[:15]
			}
			f, keys := newTreeFixture(rows, tc.count, tc.known, tc.offset, tc.animations)
			for frame, values := range tc.values {
				for i, key := range keys {
					a := f.active[key]
					a.Value = values[i]
					f.active[key] = a
				}
				if frame > 0 {
					f.layout()
				}
				// the questions are asked twice: the second time the sums are all there
				for round := 0; round < 2; round++ {
					for _, so := range scrollOffsetsToTry(rows) {
						if so < 0 || math.IsNaN(so) {
							continue
						}
						want := f.ref.childIndexForScrollOffset(so)
						if got := f.sliver.GetMinChildIndexForScrollOffset(so); got != want {
							t.Fatalf("frame %d round %d: GetMinChildIndexForScrollOffset(%v) = %d, want %d", frame, round, so, got, want)
						}
						if got := f.sliver.GetMaxChildIndexForScrollOffset(so); got != want {
							t.Fatalf("frame %d round %d: GetMaxChildIndexForScrollOffset(%v) = %d, want %d", frame, round, so, got, want)
						}
					}
					for i := -1; i <= len(rows)+3; i++ {
						want, _ := f.ref.indexToLayoutOffset(i)
						if got := f.sliver.IndexToLayoutOffset(i); got != want {
							t.Fatalf("frame %d round %d: IndexToLayoutOffset(%d) = %v, want %v", frame, round, i, got, want)
						}
					}
					if got, want := f.sliver.ComputeMaxScrollOffset(f.sliver.SliverConstraints()), f.ref.computeMaxScrollOffset(); got != want {
						t.Fatalf("frame %d round %d: ComputeMaxScrollOffset = %v, want %v", frame, round, got, want)
					}
				}
			}
		})
	}
}

// TestTreeSliverWorksOutAnimationsLikeTheOldLoop asks the offsets before every animation is
// worked out: the old code panics in a debug build for an animation that is not worked out,
// and so does the sliver, at the same moment.
func TestTreeSliverWorksOutAnimationsLikeTheOldLoop(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("needs the debug-only panic")
	}
	extents := fixedExtents(60, 20)
	// the viewport is 300 pixels: the animation at row 40 is far below the cache extent
	f, _ := newTreeFixture(extents, 60, true, 0, []rendering.TreeSliverNodesAnimation{
		{FromIndex: 3, ToIndex: 5, Value: 0.5}, {FromIndex: 41, ToIndex: 44, Value: 0.5}})
	askRef := func(index int) (float64, bool) { return f.ref.indexToLayoutOffset(index) }
	askSliver := func(index int) (v float64, ok bool) {
		defer func() { ok = recover() == nil }()
		return f.sliver.IndexToLayoutOffset(index), true
	}
	for _, index := range []int{2, 5, 20, 41, 50} {
		wv, wok := askRef(index)
		gv, gok := askSliver(index)
		if wok != gok || (wok && wv != gv) {
			t.Errorf("IndexToLayoutOffset(%d) = %v (ok %v), want %v (ok %v)", index, gv, gok, wv, wok)
		}
	}
	// a search that reaches the animation works it out, in both
	if got, want := f.sliver.GetMinChildIndexForScrollOffset(1000), f.ref.childIndexForScrollOffset(1000); got != want {
		t.Errorf("GetMinChildIndexForScrollOffset(1000) = %d, want %d", got, want)
	}
	for _, index := range []int{2, 5, 20, 41, 50} {
		wv, wok := askRef(index)
		gv, gok := askSliver(index)
		if wok != gok || (wok && wv != gv) {
			t.Errorf("after the search: IndexToLayoutOffset(%d) = %v (ok %v), want %v (ok %v)", index, gv, gok, wv, wok)
		}
		if !gok {
			t.Errorf("after the search the offset of an animation at row %d is still missing", index)
		}
	}
}

// TestTreeSliverCallsTheBuilderOncePerRow scrolls through 5,000 rows with a node that
// expands, and asks the builder at most once for each row between two rebuilds.
func TestTreeSliverCallsTheBuilderOncePerRow(t *testing.T) {
	extents := randomExtents(5000, 81)
	for i := range extents {
		extents[i] += 5
	}
	total := sumOf(extents)
	f, keys := newTreeFixture(extents, len(extents), true, 0, []rendering.TreeSliverNodesAnimation{{FromIndex: 4, ToIndex: 9, Value: 0}})
	frames := 0
	for offset := 0.0; offset < total-300; offset += 97 {
		a := f.active[keys[0]]
		a.Value = min(1, a.Value+0.01)
		f.active[keys[0]] = a
		f.root.SetOffset(rendering.NewViewportOffsetFixed(offset))
		f.layout()
		frames++
	}
	if frames < 500 {
		t.Fatalf("only %d frames", frames)
	}
	for row, n := range f.calls {
		if n > 1 {
			t.Fatalf("after %d frames row %d was asked for %d times", frames, row, n)
		}
	}
}
