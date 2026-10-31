package rendering_test

import (
	"math/rand/v2"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
)

// lazyRowsManager is a child manager of a list of count rows that makes a row when the
// sliver asks for it (a list of 100,000 rows does not make 100,000 render objects up
// front). Its scroll extent estimate is the one of the widgets' element: the average
// extent of the rows laid out times the count.
type lazyRowsManager struct {
	rendering.RenderSliverBoxChildManagerDefaults
	count    int
	known    bool // EstimatedChildCount is some(count)
	insert   func(child, after rendering.Box)
	remove   func(child rendering.Box)
	updating foundation.Opt[int]
	// estimates counts the calls of EstimateMaxScrollOffset.
	estimates int
}

func newLazyRowsManager(count int, known bool) *lazyRowsManager {
	return &lazyRowsManager{count: count, known: known}
}

func (m *lazyRowsManager) CreateChild(index int, after rendering.Box) {
	if index < 0 || index >= m.count {
		return
	}
	m.updating = foundation.Some(index)
	defer func() { m.updating = foundation.None[int]() }()
	m.insert(newRenderSizedBox(10, 10), after)
}

func (m *lazyRowsManager) RemoveChild(child rendering.Box) { m.remove(child) }

func (m *lazyRowsManager) EstimateMaxScrollOffset(_ rendering.SliverConstraints, o rendering.EstimateMaxScrollOffsetOpts) float64 {
	m.estimates++
	return float64(m.count) * (o.TrailingScrollOffset.Value() - o.LeadingScrollOffset.Value()) /
		float64(o.LastIndex.Value()-o.FirstIndex.Value()+1)
}

func (m *lazyRowsManager) ChildCount() int { return m.count }

func (m *lazyRowsManager) EstimatedChildCount() foundation.Opt[int] {
	if m.known {
		return foundation.Some(m.count)
	}
	return foundation.None[int]()
}

func (m *lazyRowsManager) DidAdoptChild(child rendering.Box) {
	rendering.SliverMultiBoxAdaptorParentDataOf(child).Index = m.updating
}

func (m *lazyRowsManager) SetDidUnderflow(bool) {}
func (m *lazyRowsManager) DidStartLayout()      {}
func (m *lazyRowsManager) DidFinishLayout()     {}

func fixedExtents(n int, extent float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = extent
	}
	return out
}

func alternatingExtents(n int, a, b float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		if i%2 == 0 {
			out[i] = a
		} else {
			out[i] = b
		}
	}
	return out
}

// randomExtents are whole numbers from 0 to 99, with a seed, so that sums are exact.
func randomExtents(n int, seed uint64) []float64 {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	out := make([]float64, n)
	for i := range out {
		out[i] = float64(r.IntN(100))
	}
	return out
}

func sumOf(extents []float64) float64 {
	total := 0.0
	for _, e := range extents {
		total += e
	}
	return total
}
