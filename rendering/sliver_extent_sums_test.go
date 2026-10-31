package rendering_test

import (
	"math"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// The reference functions below are the loops that RenderSliverFixedExtentBoxAdaptor had
// before step 29.40 (they add up the extents of the builder from row 0 on each call). The
// tests compare the sliver, which now reads running sums, with them.

func refIndexToLayoutOffset(builder rendering.ItemExtentBuilder, m rendering.RenderSliverBoxChildManager, dims rendering.SliverLayoutDimensions, index int) float64 {
	offset := 0.0
	for i := 0; i < index; i++ {
		if childCount, ok := m.EstimatedChildCount().Get(); ok && i > childCount-1 {
			break
		}
		extent, ok := builder(i, dims).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

func refChildIndexForScrollOffset(builder rendering.ItemExtentBuilder, m rendering.RenderSliverBoxChildManager, dims rendering.SliverLayoutDimensions, scrollOffset float64) int {
	if scrollOffset == 0 {
		return 0
	}
	position := 0.0
	index := 0
	for position < scrollOffset {
		if childCount, ok := m.EstimatedChildCount().Get(); ok && index > childCount-1 {
			break
		}
		extent, ok := builder(index, dims).Get()
		if !ok {
			break
		}
		position += extent
		index++
	}
	return index - 1
}

func refComputeMaxScrollOffset(builder rendering.ItemExtentBuilder, m rendering.RenderSliverBoxChildManager, dims rendering.SliverLayoutDimensions) float64 {
	offset := 0.0
	for i := 0; i < m.ChildCount(); i++ {
		extent, ok := builder(i, dims).Get()
		if !ok {
			break
		}
		offset += extent
	}
	return offset
}

// variedFixture is a varied extent list in a viewport of 300 by 300 pixels (inside a
// constrained box, so that the dimensions can change), laid out once. The builder
// answers from source(dimensions) and counts its calls.
type variedFixture struct {
	sliver  *rendering.RenderSliverVariedExtentList
	manager *lazyRowsManager
	root    *rendering.RenderViewport
	box     *rendering.RenderConstrainedBox
	stack   *renderhost.Stack
	source  func(d rendering.SliverLayoutDimensions) []float64
	calls   map[int]int
	total   int
}

func newVariedFixture(extents []float64, count int, known bool) *variedFixture {
	f := &variedFixture{manager: newLazyRowsManager(count, known), calls: map[int]int{}}
	f.source = func(rendering.SliverLayoutDimensions) []float64 { return extents }
	f.sliver = rendering.NewRenderSliverVariedExtentList(f.manager, f.build)
	f.manager.insert, f.manager.remove = f.sliver.Insert, f.sliver.Remove
	f.root = noCacheViewport(painting.AxisDirectionRight, painting.AxisDirectionDown, rendering.NewViewportOffsetZero(), f.sliver)
	f.box = rendering.NewRenderConstrainedBox(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 300}), f.root)
	f.stack = renderhost.New(renderhost.Opts{Size: ui.Size{Width: 800, Height: 600}})
	f.stack.Host.DiscardScenes = true
	f.stack.SetRoot(rendering.NewRenderPositionedBox(f.box))
	f.stack.Pump()
	return f
}

func (f *variedFixture) build(index int, d rendering.SliverLayoutDimensions) foundation.Opt[float64] {
	f.calls[index]++
	f.total++
	extents := f.source(d)
	if index < 0 || index >= len(extents) {
		return foundation.None[float64]()
	}
	return foundation.Some(extents[index])
}

// scrollTo moves the viewport to the offset and lays out a frame.
func (f *variedFixture) scrollTo(offset float64) {
	f.root.SetOffset(rendering.NewViewportOffsetFixed(offset))
	f.stack.Pump()
}

// resize changes the size of the viewport and lays out a frame.
func (f *variedFixture) resize(width, height float64) {
	f.box.SetAdditionalConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: width, Height: height}))
	f.stack.Pump()
}

// maxCalls is the most calls any one row got, and the row.
func (f *variedFixture) maxCalls() (row, calls int) {
	for r, n := range f.calls {
		if n > calls {
			row, calls = r, n
		}
	}
	return row, calls
}

func (f *variedFixture) resetCalls() {
	f.calls = map[int]int{}
	f.total = 0
}

// scrollOffsetsToTry are the scroll offsets that the equivalence tests ask about: on
// each boundary of the rows, a little before and after it, inside the rows, 0, negative,
// past the end and NaN.
func scrollOffsetsToTry(extents []float64) []float64 {
	out := []float64{0, -3, 0.25, math.NaN(), math.Inf(1)}
	position := 0.0
	for i, e := range extents {
		position += e
		out = append(out, position, position-0.5, position+0.5, position-e/2)
		if i > 400 {
			break
		}
	}
	return append(out, position+1000, position*2+1)
}

func sameFloat(a, b float64) bool {
	return a == b || (math.IsNaN(a) && math.IsNaN(b))
}

func answerFrom(extents []float64) rendering.ItemExtentBuilder {
	return func(i int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
		if i < 0 || i >= len(extents) {
			return foundation.None[float64]()
		}
		return foundation.Some(extents[i])
	}
}

func TestVariedExtentListSumsEqualTheOldAddingUp(t *testing.T) {
	tests := []struct {
		name    string
		extents []float64
		// count and known are what the manager says: known is EstimatedChildCount
		count int
		known bool
	}{
		{"fixed extents, known count", fixedExtents(40, 25), 40, true},
		{"fixed extents, unknown count", fixedExtents(40, 25), 40, false},
		{"alternating extents", alternatingExtents(60, 10, 35), 60, true},
		{"alternating extents, unknown count", alternatingExtents(60, 10, 35), 60, false},
		{"random extents", randomExtents(300, 1), 300, true},
		{"random extents, another seed", randomExtents(300, 99), 300, true},
		{"random extents, unknown count", randomExtents(300, 5), 300, false},
		{"rows of 0 at the start, the middle and the end", []float64{0, 0, 10, 0, 0, 0, 20, 5, 0, 30, 0, 0}, 12, true},
		{"all rows of 0", fixedExtents(8, 0), 8, true},
		{"one row", []float64{42}, 1, true},
		{"no rows", nil, 0, true},
		{"the builder gives none part way, known count", randomExtents(50, 3)[:20], 50, true},
		{"the builder gives none part way, unknown count", randomExtents(50, 3)[:20], 50, false},
		{"the count is less than the rows of the builder", randomExtents(50, 4), 17, true},
		{"the count is less than the rows of the builder, unknown", randomExtents(50, 4), 17, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newVariedFixture(tc.extents, tc.count, tc.known)
			dims := f.sliver.LayoutDimensions()
			reference := answerFrom(tc.extents)
			// the queries are made twice: the second round reads the sums that the first made
			for round := 0; round < 2; round++ {
				for i := -2; i <= len(tc.extents)+3; i++ {
					want := refIndexToLayoutOffset(reference, f.manager, dims, i)
					if got := f.sliver.IndexToLayoutOffset(i); !sameFloat(got, want) {
						t.Errorf("round %d: IndexToLayoutOffset(%d) = %v, want %v", round, i, got, want)
					}
				}
				for _, so := range scrollOffsetsToTry(tc.extents) {
					want := refChildIndexForScrollOffset(reference, f.manager, dims, so)
					if got := f.sliver.GetMinChildIndexForScrollOffset(so); got != want {
						t.Errorf("round %d: GetMinChildIndexForScrollOffset(%v) = %d, want %d", round, so, got, want)
					}
					if got := f.sliver.GetMaxChildIndexForScrollOffset(so); got != want {
						t.Errorf("round %d: GetMaxChildIndexForScrollOffset(%v) = %d, want %d", round, so, got, want)
					}
				}
				want := refComputeMaxScrollOffset(reference, f.manager, dims)
				if got := f.sliver.ComputeMaxScrollOffset(f.sliver.SliverConstraints()); !sameFloat(got, want) {
					t.Errorf("round %d: ComputeMaxScrollOffset = %v, want %v", round, got, want)
				}
			}
		})
	}
}

// TestVariedExtentListSumsInEveryOrder asks the same questions in a random order, so that
// the sums are filled in steps that go up and down.
func TestVariedExtentListSumsInEveryOrder(t *testing.T) {
	extents := randomExtents(500, 31)
	f := newVariedFixture(extents, len(extents), true)
	dims := f.sliver.LayoutDimensions()
	reference := answerFrom(extents)
	queries := scrollOffsetsToTry(extents)
	order := randomExtents(len(queries)*2, 17)
	for _, v := range order {
		so := queries[int(v*7919)%len(queries)]
		want := refChildIndexForScrollOffset(reference, f.manager, dims, so)
		if got := f.sliver.GetMinChildIndexForScrollOffset(so); got != want {
			t.Fatalf("GetMinChildIndexForScrollOffset(%v) = %d, want %d", so, got, want)
		}
		index := int(v) * 5
		if got, want := f.sliver.IndexToLayoutOffset(index), refIndexToLayoutOffset(reference, f.manager, dims, index); got != want {
			t.Fatalf("IndexToLayoutOffset(%d) = %v, want %v", index, got, want)
		}
	}
}

// TestVariedExtentListCallsTheBuilderOncePerRow scrolls through a list of 10,000 rows
// frame by frame and asks the builder at most once for each row between two rebuilds.
func TestVariedExtentListCallsTheBuilderOncePerRow(t *testing.T) {
	const rows = 10_000
	extents := randomExtents(rows, 11)
	total := sumOf(extents)
	tests := []struct {
		name  string
		known bool
	}{
		{"known count", true},
		{"unknown count", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newVariedFixture(extents, rows, tc.known)
			frames := 0
			// forward by 137 pixels, to the end, and back by 401 pixels
			for offset := 0.0; offset < total-300; offset += 137 {
				f.scrollTo(offset)
				frames++
			}
			for offset := total - 300; offset > 0; offset -= 401 {
				f.scrollTo(offset)
				frames++
			}
			if frames < 1000 {
				t.Fatalf("only %d frames", frames)
			}
			if row, n := f.maxCalls(); n > 1 {
				t.Fatalf("after %d frames row %d was asked for %d times (%d calls for %d rows)", frames, row, n, f.total, rows)
			}
			// after a rebuild the rows are asked for again, once each
			f.resetCalls()
			f.sliver.SetItemExtentBuilder(f.build)
			for offset := 0.0; offset < total-300; offset += 733 {
				f.scrollTo(offset)
			}
			if row, n := f.maxCalls(); n > 1 {
				t.Errorf("after a rebuild row %d was asked for %d times", row, n)
			}
			if f.total == 0 {
				t.Error("a new builder was not asked for any row")
			}
		})
	}
}

// TestVariedExtentListInvalidation checks that what the sliver keeps is dropped when the
// builder, the dimensions and the child count change.
func TestVariedExtentListInvalidation(t *testing.T) {
	first, second := randomExtents(200, 21), randomExtents(200, 22)
	tests := []struct {
		name string
		// change makes the change and gives the extents and the child count after it
		change func(f *variedFixture) (extents []float64, count int)
	}{
		{"a new builder", func(f *variedFixture) ([]float64, int) {
			f.source = func(rendering.SliverLayoutDimensions) []float64 { return second }
			f.sliver.SetItemExtentBuilder(f.build)
			return second, 200
		}},
		{"a new cross axis extent", func(f *variedFixture) ([]float64, int) {
			f.source = func(d rendering.SliverLayoutDimensions) []float64 {
				if d.CrossAxisExtent == 300 {
					return first
				}
				return second
			}
			f.resize(200, 300)
			return second, 200
		}},
		{"a new viewport extent", func(f *variedFixture) ([]float64, int) {
			f.source = func(d rendering.SliverLayoutDimensions) []float64 {
				if d.ViewportMainAxisExtent == 300 {
					return first
				}
				return second
			}
			f.resize(300, 250)
			return second, 200
		}},
		{"a new child count", func(f *variedFixture) ([]float64, int) {
			f.source = func(rendering.SliverLayoutDimensions) []float64 { return second }
			f.manager.count = 150
			f.sliver.MarkNeedsLayout()
			f.stack.Pump()
			return second, 150
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newVariedFixture(first, 200, true)
			f.scrollTo(100)
			if got, want := f.sliver.IndexToLayoutOffset(50), sumOf(first[:50]); got != want {
				t.Fatalf("before the change IndexToLayoutOffset(50) = %v, want %v", got, want)
			}
			extents, count := tc.change(f)
			f.scrollTo(120)
			dims := f.sliver.LayoutDimensions()
			reference := answerFrom(extents)
			if got, want := f.sliver.IndexToLayoutOffset(50), refIndexToLayoutOffset(reference, f.manager, dims, 50); got != want {
				t.Errorf("after the change IndexToLayoutOffset(50) = %v, want %v (the old sums were kept)", got, want)
			}
			if got, want := f.sliver.ComputeMaxScrollOffset(f.sliver.SliverConstraints()), sumOf(extents[:count]); got != want {
				t.Errorf("after the change ComputeMaxScrollOffset = %v, want %v", got, want)
			}
			so := sumOf(extents[:80]) + 1
			if got, want := f.sliver.GetMinChildIndexForScrollOffset(so), refChildIndexForScrollOffset(reference, f.manager, dims, so); got != want {
				t.Errorf("after the change GetMinChildIndexForScrollOffset(%v) = %d, want %d", so, got, want)
			}
		})
	}
}

// TestVariedExtentListExactScrollExtent checks the scroll extent: the total of the extents
// when the child count is known and the builder gives every extent, the estimate of the child
// manager otherwise.
func TestVariedExtentListExactScrollExtent(t *testing.T) {
	extents := randomExtents(400, 41)
	tests := []struct {
		name    string
		extents []float64
		count   int
		known   bool
		exact   bool
	}{
		{"known count, every extent", extents, 400, true, true},
		{"known count, the builder stops early", extents[:300], 400, true, false},
		{"unknown count", extents, 400, false, false},
		{"known count, the builder goes on past it", extents, 100, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newVariedFixture(tc.extents, tc.count, tc.known)
			got := f.sliver.Geometry().ScrollExtent
			if tc.exact {
				if want := sumOf(tc.extents[:tc.count]); got != want {
					t.Errorf("scroll extent = %v, want the total %v", got, want)
				}
				if f.manager.estimates != 0 {
					t.Errorf("the child manager was asked for the estimate %d times", f.manager.estimates)
				}
				return
			}
			total := sumOf(tc.extents)
			if got == total {
				t.Errorf("scroll extent = %v, the total of a list that is not known to the end; the estimate of the child manager was expected", got)
			}
			if f.manager.estimates == 0 {
				t.Error("the child manager was not asked for the estimate")
			}
		})
	}
}

// TestVariedExtentListJumpToTheEnd jumps to the end of 100,000 rows in one frame, and the
// last row is laid out and visible in that frame.
func TestVariedExtentListJumpToTheEnd(t *testing.T) {
	const rows = 100_000
	extents := randomExtents(rows, 51)
	for i := range extents {
		extents[i] += 1 // no row of 0, so that the last row has a size
	}
	total := sumOf(extents)
	f := newVariedFixture(extents, rows, true)
	if got := f.sliver.Geometry().ScrollExtent; got != total {
		t.Fatalf("scroll extent = %v, want %v", got, total)
	}
	f.scrollTo(total - 300)
	last := f.sliver.LastChild()
	if last == nil {
		t.Fatal("no child after the jump")
	}
	if got := f.sliver.IndexOf(last); got != rows-1 {
		t.Fatalf("the last child is row %d, want %d", got, rows-1)
	}
	offset := f.sliver.IndexToLayoutOffset(rows - 1)
	if want := total - extents[rows-1]; offset != want {
		t.Errorf("last row at %v, want %v", offset, want)
	}
	if got := last.Size().Height; got != extents[rows-1] {
		t.Errorf("last row has height %v, want %v", got, extents[rows-1])
	}
	// the viewport ends where the list ends: the last row is the last thing in it
	if end := f.sliver.Geometry().ScrollExtent; end != total {
		t.Errorf("scroll extent after the jump = %v, want %v", end, total)
	}
	if row, n := f.maxCalls(); n > 1 {
		t.Errorf("row %d was asked for %d times", row, n)
	}
}

// liveHeap is the bytes of the heap that the program holds after a collection.
func liveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// TestVariedExtentListSumsDoNotKeepRowsOfAShorterList checks that the memory of a long
// list is given back when the list gets short (SPEC §15, "Memory over time").
func TestVariedExtentListSumsDoNotKeepRowsOfAShorterList(t *testing.T) {
	const longRows = 1_000_000 // 16 MB of extents and sums
	before := liveHeap()
	var f *variedFixture
	func() {
		long := randomExtents(longRows, 61)
		f = newVariedFixture(long, longRows, true)
		if got, want := f.sliver.ComputeMaxScrollOffset(f.sliver.SliverConstraints()), sumOf(long); got != want {
			t.Fatalf("ComputeMaxScrollOffset = %v, want %v", got, want)
		}
	}()
	f.resetCalls()
	full := liveHeap()
	if full < before+8<<20 {
		t.Fatalf("the sums of %d rows hold only %d bytes", longRows, full-before)
	}
	short := randomExtents(20, 62)
	f.source = func(rendering.SliverLayoutDimensions) []float64 { return short }
	f.manager.count = len(short)
	f.sliver.SetItemExtentBuilder(f.build)
	f.stack.Pump()
	if got, want := f.sliver.ComputeMaxScrollOffset(f.sliver.SliverConstraints()), sumOf(short); got != want {
		t.Errorf("ComputeMaxScrollOffset = %v, want %v", got, want)
	}
	f.resetCalls() // the map of the calls of the builder is the test's, not the sliver's
	after := liveHeap()
	if after > before+4<<20 {
		t.Errorf("a list of %d rows that was %d rows holds %d bytes more than before it was made", len(short), longRows, after-before)
	}
	runtime.KeepAlive(f)
}
