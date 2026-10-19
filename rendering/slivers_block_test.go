package rendering_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// testSliverBoxChildManager is TestRenderSliverBoxChildManager of
// slivers_block_test.dart: it hands out the given boxes.
type testSliverBoxChildManager struct {
	rendering.RenderSliverBoxChildManagerDefaults
	children []rendering.Box
	// insert, remove and the render object are set by the sliver's constructor
	insert                 func(child, after rendering.Box)
	remove                 func(child rendering.Box)
	currentlyUpdatingIndex foundation.Opt[int]
	didUnderflow           bool
	started, finished      int
}

func (m *testSliverBoxChildManager) CreateChild(index int, after rendering.Box) {
	if index < 0 || index >= len(m.children) {
		return
	}
	m.currentlyUpdatingIndex = foundation.Some(index)
	defer func() { m.currentlyUpdatingIndex = foundation.None[int]() }()
	m.insert(m.children[index], after)
}

func (m *testSliverBoxChildManager) RemoveChild(child rendering.Box) { m.remove(child) }

func (m *testSliverBoxChildManager) EstimateMaxScrollOffset(_ rendering.SliverConstraints, o rendering.EstimateMaxScrollOffsetOpts) float64 {
	return float64(len(m.children)) * (o.TrailingScrollOffset.Value() - o.LeadingScrollOffset.Value()) /
		float64(o.LastIndex.Value()-o.FirstIndex.Value()+1)
}

func (m *testSliverBoxChildManager) ChildCount() int { return len(m.children) }

func (m *testSliverBoxChildManager) DidAdoptChild(child rendering.Box) {
	rendering.SliverMultiBoxAdaptorParentDataOf(child).Index = m.currentlyUpdatingIndex
}

func (m *testSliverBoxChildManager) SetDidUnderflow(value bool) { m.didUnderflow = value }
func (m *testSliverBoxChildManager) DidStartLayout()            { m.started++ }
func (m *testSliverBoxChildManager) DidFinishLayout()           { m.finished++ }

func newTestChildManager(children ...rendering.Box) *testSliverBoxChildManager {
	return &testSliverBoxChildManager{children: children}
}

func (m *testSliverBoxChildManager) createList() *rendering.RenderSliverList {
	r := rendering.NewRenderSliverList(m)
	m.insert, m.remove = r.Insert, r.Remove
	return r
}

func (m *testSliverBoxChildManager) createFixedExtentList(itemExtent float64) *rendering.RenderSliverFixedExtentList {
	r := rendering.NewRenderSliverFixedExtentList(m, itemExtent)
	m.insert, m.remove = r.Insert, r.Remove
	return r
}

func (m *testSliverBoxChildManager) createFillViewport() *rendering.RenderSliverFillViewport {
	r := rendering.NewRenderSliverFillViewport(m)
	m.insert, m.remove = r.Insert, r.Remove
	return r
}

func (m *testSliverBoxChildManager) createGrid(delegate rendering.SliverGridDelegate) *rendering.RenderSliverGrid {
	r := rendering.NewRenderSliverGrid(m, delegate)
	m.insert, m.remove = r.Insert, r.Remove
	return r
}

func sizedBoxes(n int, w, h float64) []*RenderSizedBox {
	boxes := make([]*RenderSizedBox, n)
	for i := range boxes {
		boxes[i] = newRenderSizedBox(w, h)
	}
	return boxes
}

func asBoxes(boxes []*RenderSizedBox) []rendering.Box {
	out := make([]rendering.Box, len(boxes))
	for i, b := range boxes {
		out[i] = b
	}
	return out
}

func noCacheViewport(crossAxisDirection painting.AxisDirection, axisDirection painting.AxisDirection, offset rendering.ViewportOffset, children ...rendering.Sliver) *rendering.RenderViewport {
	return rendering.NewRenderViewport(crossAxisDirection, offset, rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{
			AxisDirection:     axisDirection,
			ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)),
		},
		Children: children,
	})
}

// viewportOffsetSpy is ViewportOffsetSpy of slivers_block_test.dart.
type viewportOffsetSpy struct {
	*foundation.ChangeNotifier
	pixels    float64
	corrected bool
}

func newViewportOffsetSpy(pixels float64) *viewportOffsetSpy {
	return &viewportOffsetSpy{ChangeNotifier: foundation.NewChangeNotifier(), pixels: pixels}
}

func (s *viewportOffsetSpy) Pixels() float64                          { return s.pixels }
func (s *viewportOffsetSpy) HasPixels() bool                          { return true }
func (s *viewportOffsetSpy) ApplyViewportDimension(float64) bool      { return true }
func (s *viewportOffsetSpy) ApplyContentDimensions(_, _ float64) bool { return true }
func (s *viewportOffsetSpy) CorrectBy(correction float64) {
	s.pixels += correction
	s.corrected = true
}
func (s *viewportOffsetSpy) JumpTo(float64) {}
func (s *viewportOffsetSpy) AnimateTo(float64, time.Duration, animation.Curve) *scheduler.TickerFuture {
	return scheduler.NewTickerFutureComplete()
}
func (s *viewportOffsetSpy) UserScrollDirection() rendering.ScrollDirection {
	return rendering.ScrollDirectionIdle
}
func (s *viewportOffsetSpy) AllowImplicitScrolling() bool { return false }

func TestRenderSliverListBasic(t *testing.T) {
	// position: where each box is, or none if it is not attached
	type position struct {
		want ui.Offset
		none bool
	}
	at := func(x, y float64) position { return position{want: off(x, y)} }
	gone := position{none: true}
	tests := []struct {
		name string
		axis painting.AxisDirection
		// each step: the scroll offset, and the positions of a..e
		steps []struct {
			offset float64
			want   [5]position
		}
	}{
		{"down", painting.AxisDirectionDown, []struct {
			offset float64
			want   [5]position
		}{
			{0, [5]position{at(0, 0), at(0, 400), gone, gone, gone}},
			{200, [5]position{at(0, -200), at(0, 200), gone, gone, gone}},
			{600, [5]position{gone, at(0, -200), at(0, 200), gone, gone}},
			{900, [5]position{gone, gone, at(0, -100), at(0, 300), gone}},
			{200, [5]position{at(0, -200), at(0, 200), gone, gone, gone}},
		}},
		{"up", painting.AxisDirectionUp, []struct {
			offset float64
			want   [5]position
		}{
			{0, [5]position{at(0, 200), at(0, -200), gone, gone, gone}},
			{200, [5]position{at(0, 400), at(0, 0), gone, gone, gone}},
			{600, [5]position{gone, at(0, 400), at(0, 0), gone, gone}},
			{900, [5]position{gone, gone, at(0, 300), at(0, -100), gone}},
			{200, [5]position{at(0, 400), at(0, 0), gone, gone, gone}},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boxes := sizedBoxes(5, 100, 400)
			manager := newTestChildManager(asBoxes(boxes)...)
			inner := manager.createList()
			root := noCacheViewport(painting.AxisDirectionRight, tc.axis, rendering.NewViewportOffsetZero(), inner)
			h := newHarness(t)
			layout(t, h, root)
			if root.Size() != (ui.Size{Width: 800, Height: 600}) {
				t.Fatalf("size = %v, want 800x600", root.Size())
			}
			check := func(step int, offset float64, want [5]position) {
				t.Helper()
				for i, b := range boxes {
					if want[i].none {
						if b.Attached() {
							t.Errorf("step %d (offset %v): box %c is attached, want detached", step, offset, 'a'+i)
						}
						continue
					}
					if !b.Attached() {
						t.Errorf("step %d (offset %v): box %c is detached", step, offset, 'a'+i)
						continue
					}
					if got := b.LocalToGlobal(ui.Offset{}, nil); got != want[i].want {
						t.Errorf("step %d (offset %v): box %c at %v, want %v", step, offset, 'a'+i, got, want[i].want)
					}
				}
			}
			check(0, 0, tc.steps[0].want)
			// make sure that layout is stable by laying out again
			inner.MarkNeedsLayout()
			pumpFrame(h, phaseLayout)
			check(0, 0, tc.steps[0].want)
			// now try various scroll offsets
			for i, s := range tc.steps[1:] {
				root.SetOffset(rendering.NewViewportOffsetFixed(s.offset))
				pumpFrame(h, phaseLayout)
				check(i+1, s.offset, s.want)
			}
		})
	}
}

func TestSliverListNoZeroScrollOffsetCorrection(t *testing.T) {
	boxes := sizedBoxes(5, 100, 400)
	manager := newTestChildManager(asBoxes(boxes)...)
	inner := manager.createList()
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{inner}})
	h := newHarness(t)
	layout(t, h, root)

	rendering.SliverMultiBoxAdaptorParentDataOf(boxes[0]).LayoutOffset = foundation.Some(0.001)

	root.SetOffset(rendering.NewViewportOffsetFixed(900))
	pumpFrame(h, phaseLayout)

	root.SetOffset(rendering.NewViewportOffsetFixed(0))
	pumpFrame(h, phaseLayout)

	if c := inner.Geometry().ScrollOffsetCorrection; c.IsSome() {
		t.Errorf("scrollOffsetCorrection = %v, want none", c.Value())
	}
}

func TestSliverListNoCorrectionOnTinyPrecisionError(t *testing.T) {
	boxes := sizedBoxes(5, 100, 400)
	manager := newTestChildManager(asBoxes(boxes)...)
	inner := manager.createList()
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{inner}})
	h := newHarness(t)
	layout(t, h, root)

	// Simulate double precision error.
	rendering.SliverMultiBoxAdaptorParentDataOf(boxes[0]).LayoutOffset = foundation.Some(-0.0000000000001)

	root.SetOffset(rendering.NewViewportOffsetFixed(900))
	pumpFrame(h, phaseLayout)

	spy := newViewportOffsetSpy(0)
	root.SetOffset(spy)
	pumpFrame(h, phaseLayout)

	if spy.corrected {
		t.Error("the viewport offset was corrected")
	}
}

func TestSliverMultiBoxAdaptorParentDataString(t *testing.T) {
	d := &rendering.SliverMultiBoxAdaptorParentData{}
	if d.KeepAlive || d.Index.IsSome() {
		t.Fatalf("new parent data has keepAlive %v and index %v", d.KeepAlive, d.Index)
	}
	steps := []struct {
		apply func()
		want  string
	}{
		{func() {}, "index=null; layoutOffset=None"},
		{func() { d.KeepAlive = true }, "index=null; keepAlive; layoutOffset=None"},
		{func() { d.KeepAlive = false }, "index=null; layoutOffset=None"},
		{func() { d.Index = foundation.Some(0) }, "index=0; layoutOffset=None"},
		{func() { d.Index = foundation.Some(1) }, "index=1; layoutOffset=None"},
		{func() { d.Index = foundation.Some(-1) }, "index=-1; layoutOffset=None"},
		{func() { d.LayoutOffset = foundation.Some(100.0) }, "index=-1; layoutOffset=100.0"},
	}
	for _, s := range steps {
		s.apply()
		if got := d.String(); got != s.want {
			t.Errorf("String() = %q, want %q", got, s.want)
		}
	}
}
