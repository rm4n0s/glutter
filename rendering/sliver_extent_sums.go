package rendering

import (
	"math"
	"slices"
	"sort"

	"github.com/rm4n0s/glutter/foundation"
)

// maxExactRows is the longest list whose scroll extent is the total of its extents: the
// total needs the extent of every row, and a list that says it has more rows than this
// (an endless list has a huge count) keeps the estimate of its child manager.
const maxExactRows = 1 << 22

// extentSummer is implemented by the slivers (RenderSliverVariedExtentList and
// RenderTreeSliver) whose extents come from a builder and that read them from the
// running sums of RenderSliverFixedExtentBoxAdaptor, not from the builder at each call.
// A sliver with its own FixedExtentLayoutOverrides (the carousel) does not implement it:
// its builder may depend on the scroll offset.
type extentSummer interface{ usesExtentSums() }

// exactExtentSummer is implemented by the slivers that report the total of their
// extents as the scroll extent when the child count is known and the builder gives
// every extent (RenderSliverVariedExtentList).
type exactExtentSummer interface{ exactScrollExtent() }

// maxCachedSumsCapacity is the capacity of the slices of extentSums that is kept
// whatever the child count (see extentSums.release).
const maxCachedSumsCapacity = 1024

// extentSums are the extents of the rows of a list, and their running sums, which
// RenderSliverFixedExtentBoxAdaptor reads instead of adding up the extents of the
// builder from row 0 on every call. They are made lazily, to the highest row asked for,
// and dropped when the builder is set, when the viewport extent or the cross axis
// extent of the layout dimensions change, or when the estimated child count changes
// (SPEC §11.4). They hold the answers of the builder only: the child count limits what a
// query reads, not what is kept.
//
// Deviation: Flutter adds up the extents on each call (SPEC §12.10).
//
// Flutter: no counterpart
type extentSums struct {
	valid bool
	// viewport and cross are the layout dimensions that the extents were made with;
	// the scroll offset and the preceding scroll extent change in every scroll and are
	// not part of the key.
	viewport, cross float64
	// count is the estimated child count when the sums were started.
	count foundation.Opt[int]
	// extents[i] is the extent of the row i; sums[i] is the layout offset of the row i,
	// so sums has one more value than extents and sums[0] is 0. The rows 0 to
	// len(extents)-1 are those the builder gave an extent for.
	extents, sums []float64
	// ended says that the builder gave no extent for the row len(extents): the list
	// ends there.
	ended bool
}

// drop forgets the sums. It keeps the memory of the slices for the next ones: the
// first layout after a rebuild makes them again, and makes them in the same size.
func (s *extentSums) drop() {
	s.valid, s.ended = false, false
	s.extents, s.sums = s.extents[:0], s.sums[:0]
}

// release gives the memory back when it is much more than the list needs now (a list
// that got shorter).
func (s *extentSums) release(count foundation.Opt[int]) {
	limit := maxCachedSumsCapacity
	if n, ok := count.Get(); ok {
		limit += 2 * max(n, 0)
	}
	if cap(s.extents) > limit {
		s.extents, s.sums = nil, nil
	}
}

// sumsEnabled says whether the sliver reads its extents from the sums.
func (a *RenderSliverFixedExtentBoxAdaptor) sumsEnabled() bool {
	if _, ok := a.self.(extentSummer); !ok {
		return false
	}
	return a.hooks().ItemExtentBuilder() != nil
}

// dropExtentSums forgets the sums: the builder changed.
func (a *RenderSliverFixedExtentBoxAdaptor) dropExtentSums() { a.sums.drop() }

// extentCache is the sums, dropped first if the layout dimensions or the child count
// are not those they were made with.
func (a *RenderSliverFixedExtentBoxAdaptor) extentCache() *extentSums {
	s := &a.sums
	d := a.LayoutDimensions()
	count := a.childManager.EstimatedChildCount()
	if s.valid && (math.Float64bits(s.viewport) != math.Float64bits(d.ViewportMainAxisExtent) ||
		math.Float64bits(s.cross) != math.Float64bits(d.CrossAxisExtent) || s.count != count) {
		s.drop()
	}
	if !s.valid {
		s.release(count)
		s.valid, s.viewport, s.cross, s.count = true, d.ViewportMainAxisExtent, d.CrossAxisExtent, count
		s.sums = append(s.sums[:0], 0)
	}
	return s
}

// fillExtentSums asks the builder for the rows up to the row n-1 that it was not asked
// for yet, and stops at the first row that has no extent.
func (a *RenderSliverFixedExtentBoxAdaptor) fillExtentSums(n int) *extentSums {
	s := a.extentCache()
	a.fillTo(s, n)
	return s
}

func (a *RenderSliverFixedExtentBoxAdaptor) fillTo(s *extentSums, n int) {
	if s.ended || len(s.extents) >= n {
		return
	}
	builder := a.hooks().ItemExtentBuilder()
	dims := a.LayoutDimensions()
	for !s.ended && len(s.extents) < n {
		i := len(s.extents)
		extent, ok := builder(i, dims).Get()
		if !ok {
			s.ended = true
			return
		}
		s.extents = append(s.extents, extent)
		s.sums = append(s.sums, s.sums[i]+extent)
	}
}

// prefixSum is the layout offset of the row at the index, as the sum of the extents of
// the rows before it, up to the first row without an extent.
func (a *RenderSliverFixedExtentBoxAdaptor) prefixSum(index int) float64 {
	if index <= 0 {
		return 0
	}
	s := a.fillExtentSums(index)
	return s.sums[min(index, len(s.extents))]
}

// childCountLimit is the estimated child count if the manager knows it, and the largest
// int if not.
func (a *RenderSliverFixedExtentBoxAdaptor) childCountLimit() int {
	if count, ok := a.childManager.EstimatedChildCount().Get(); ok {
		return count
	}
	return int(^uint(0) >> 1)
}

// childIndexAtSum is the index that getChildIndexForScrollOffset finds for a scroll
// offset other than 0: the row before the first whose layout offset is at least the
// scroll offset (-1 for a scroll offset that is not positive, or when there is no
// row), limited to the rows that exist.
func (a *RenderSliverFixedExtentBoxAdaptor) childIndexAtSum(scrollOffset float64) int {
	if !(scrollOffset > 0) {
		return -1
	}
	limit := a.childCountLimit()
	s := a.extentCache()
	// Extend the sums to the first row that reaches the scroll offset: the rows that
	// are filled once are not asked for again.
	for !s.ended && len(s.extents) < limit && s.sums[len(s.extents)] < scrollOffset {
		a.fillTo(s, len(s.extents)+1)
	}
	n := min(len(s.extents), limit)
	// The first k in 1..n with sums[k] >= scrollOffset; n if none.
	i := sort.Search(n, func(i int) bool { return s.sums[i+1] >= scrollOffset })
	if i >= n {
		return n - 1
	}
	return i
}

// extentOf is the main axis extent of the child at the index: the item extent, or what
// the builder gives (0 if it gives none).
func (a *RenderSliverFixedExtentBoxAdaptor) extentOf(index int) float64 {
	builder := a.hooks().ItemExtentBuilder()
	if builder == nil {
		return a.fixedItemExtent()
	}
	if a.sumsEnabled() && index >= 0 {
		if s := a.fillExtentSums(index + 1); index < len(s.extents) {
			return s.extents[index]
		}
	}
	return builder(index, a.LayoutDimensions()).Value()
}

// exactMaxScrollOffset is the total of the extents if the sliver reports it as its
// scroll extent: the child count is known and the builder gives the extent of every
// row.
func (a *RenderSliverFixedExtentBoxAdaptor) exactMaxScrollOffset() (float64, bool) {
	if _, ok := a.self.(exactExtentSummer); !ok || !a.sumsEnabled() {
		return 0, false
	}
	count, ok := a.childManager.EstimatedChildCount().Get()
	if !ok || count < 0 || count > maxExactRows {
		return 0, false
	}
	s := a.extentCache()
	if !s.ended && len(s.extents) < count {
		// The whole list is asked for: make room for it once.
		s.extents = slices.Grow(s.extents, count-len(s.extents))
		s.sums = slices.Grow(s.sums, count-len(s.extents))
		a.fillTo(s, count)
	}
	if len(s.extents) < count {
		return 0, false
	}
	return s.sums[count], true
}
