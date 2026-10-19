package rendering_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// allOverlapSliver is _RenderAllOverlapSliver of viewport_test.dart: a sliver that
// overlaps the others as far as possible, and does nothing else but log its paint
// and add itself to a hit test.
type allOverlapSliver struct {
	rendering.RenderSliver
	id       int
	paintLog *[]int
}

func newAllOverlapSliver(id int, paintLog *[]int) *allOverlapSliver {
	s := &allOverlapSliver{id: id, paintLog: paintLog}
	rendering.InitRenderSliver(&s.RenderSliver, s)
	return s
}

func (s *allOverlapSliver) PerformLayout() {
	c := s.SliverConstraints()
	s.SetGeometry(rendering.NewSliverGeometry(rendering.SliverGeometryOpts{
		PaintExtent: c.RemainingPaintExtent, MaxPaintExtent: c.RemainingPaintExtent, LayoutExtent: foundation.Some(0.0),
	}))
}

func (s *allOverlapSliver) Paint(*rendering.PaintingContext, ui.Offset) {
	*s.paintLog = append(*s.paintLog, s.id)
}

func (s *allOverlapSliver) HitTest(result *rendering.SliverHitTestResult, mainAxisPosition, crossAxisPosition float64) bool {
	if mainAxisPosition >= 0 && mainAxisPosition < s.Geometry().HitTestExtent && crossAxisPosition >= 0 && crossAxisPosition < s.SliverConstraints().CrossAxisExtent {
		result.Add(rendering.NewSliverHitTestEntry(s, mainAxisPosition, crossAxisPosition))
	}
	return false
}

func overlapViewport(order rendering.SliverPaintOrder, paintLog *[]int) *rendering.RenderViewport {
	var slivers []rendering.Sliver
	for i := 0; i < 5; i++ {
		slivers = append(slivers, newAllOverlapSliver(i, paintLog))
	}
	return rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{PaintOrder: order},
		Anchor:                 0.5, Center: slivers[2], Children: slivers,
	})
}

func TestViewportPaintOrder(t *testing.T) {
	tests := []struct {
		name  string
		order rendering.SliverPaintOrder
		want  []int
	}{
		// First sliver paints last, over other slivers; last sliver paints first.
		{"default (firstIsTop)", rendering.SliverPaintOrderUnspecified, []int{4, 3, 2, 1, 0}},
		{"firstIsTop", rendering.SliverPaintOrderFirstIsTop, []int{4, 3, 2, 1, 0}},
		// Last sliver paints last, over other slivers; first sliver paints first.
		{"lastIsTop", rendering.SliverPaintOrderLastIsTop, []int{0, 1, 2, 3, 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log []int
			layout(t, newHarness(t), overlapViewport(tc.order, &log), atPhase(layoutOpts{}, phasePaint))
			if !equalInts(log, tc.want) {
				t.Errorf("paint order = %v, want %v", log, tc.want)
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestViewportHitTestOrder(t *testing.T) {
	tests := []struct {
		name  string
		order rendering.SliverPaintOrder
		want  []int
	}{
		{"default (firstIsTop)", rendering.SliverPaintOrderUnspecified, []int{0, 1, 2, 3, 4}},
		{"lastIsTop", rendering.SliverPaintOrderLastIsTop, []int{4, 3, 2, 1, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log []int
			root := overlapViewport(tc.order, &log)
			layout(t, newHarness(t), root)
			var got []int
			for _, target := range hitTargets(root, off(400, 300)) {
				if s, ok := target.(*allOverlapSliver); ok {
					got = append(got, s.id)
				}
			}
			if !equalInts(got, tc.want) {
				t.Errorf("hit test order = %v, want %v", got, tc.want)
			}
			var _ gestures.HitTestTarget = (*allOverlapSliver)(nil)
		})
	}
}

func TestViewportAnchorAndCenter(t *testing.T) {
	var log []int
	boxes := sizedBoxes(3, 100, 100)
	var slivers []rendering.Sliver
	for _, b := range boxes {
		slivers = append(slivers, rendering.NewRenderSliverToBoxAdapter(b))
	}
	// the center is the second sliver, at the anchor 0.5 of the 600 high viewport
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
		Anchor: 0.5, Center: slivers[1], Children: slivers,
	})
	layout(t, newHarness(t), root)
	_ = log
	want := []float64{200, 300, 400}
	for i, b := range boxes {
		if got := b.LocalToGlobal(ui.Offset{}, root).Dy; got != want[i] {
			t.Errorf("box %d at %v, want %v", i, got, want[i])
		}
	}
	if root.Center() != slivers[1] || root.Anchor() != 0.5 {
		t.Errorf("center %v anchor %v", root.Center(), root.Anchor())
	}
	if got := root.IndexOfFirstChild(); got != -1 {
		t.Errorf("IndexOfFirstChild = %d, want -1", got)
	}
	if got := root.LabelForChild(0); got != "center child" {
		t.Errorf("LabelForChild(0) = %q", got)
	}
	if got := root.LabelForChild(-1); got != "child -1" {
		t.Errorf("LabelForChild(-1) = %q", got)
	}
	root.SetCenter(slivers[0])
	pumpFrame(newHarness(t), phaseLayout)
}

func TestViewportApproximatePaintClipRespectsClipBehavior(t *testing.T) {
	tests := []struct {
		name  string
		clip  ui.Clip
		clips bool
	}{
		{"default", ui.ClipUnspecified, true},
		{"none", ui.ClipNone, false},
		{"hard edge", ui.ClipHardEdge, true},
		{"anti alias", ui.ClipAntiAlias, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sliver := rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(20, 20))
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ClipBehavior: tc.clip}, Children: []rendering.Sliver{sliver},
			})
			layout(t, newHarness(t), root)
			wantBehavior := tc.clip
			if wantBehavior == ui.ClipUnspecified {
				wantBehavior = ui.ClipHardEdge
			}
			if root.ClipBehavior() != wantBehavior {
				t.Errorf("clipBehavior = %v, want %v", root.ClipBehavior(), wantBehavior)
			}
			want := foundation.None[ui.Rect]()
			if tc.clips {
				want = foundation.Some(ui.OffsetZero.And(root.Size()))
			}
			if got := root.DescribeApproximatePaintClip(sliver); got != want {
				t.Errorf("DescribeApproximatePaintClip = %v, want %v", got, want)
			}
		})
	}
}

func TestViewportClipsOverflowingContent(t *testing.T) {
	tests := []struct {
		name   string
		clip   ui.Clip
		offset float64
		want   ui.Clip
	}{
		{"content that overflows", ui.ClipUnspecified, 0, ui.ClipHardEdge},
		{"content that overflows, anti aliased", ui.ClipAntiAlias, 0, ui.ClipAntiAlias},
		{"content that overflows, clip none", ui.ClipNone, 0, ui.ClipNone},
		{"content that fits", ui.ClipUnspecified, -1, ui.ClipNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			height := 1000.0
			if tc.offset < 0 {
				height = 100
			}
			sliver := rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(800, height))
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ClipBehavior: tc.clip}, Children: []rendering.Sliver{sliver},
			})
			h := newHarness(t)
			layout(t, h, root, atPhase(layoutOpts{}, phasePaint))
			if got := observedClip(t, h, root); got != tc.want {
				t.Errorf("clip = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestViewportIntrinsicsAreNotSupported(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		box  rendering.Box
		want []string
	}{
		{"RenderViewport", rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()), []string{
			"RenderViewport does not support returning intrinsic dimensions.",
			"Calculating the intrinsic dimensions would require instantiating every child of the viewport, which defeats the point of viewports being lazy.",
			"consider a RenderShrinkWrappingViewport render object (ShrinkWrappingViewport widget), which achieves that effect without implementing the intrinsic dimension API.",
		}},
		{"RenderShrinkWrappingViewport", rendering.NewRenderShrinkWrappingViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero()), []string{
			"RenderShrinkWrappingViewport does not support returning intrinsic dimensions.",
			"you should be able to achieve that effect by just giving the viewport loose constraints, without needing to measure its intrinsic dimensions.",
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for name, call := range map[string]func(){
				"min width":  func() { tc.box.ComputeMinIntrinsicWidth(0) },
				"max width":  func() { tc.box.ComputeMaxIntrinsicWidth(0) },
				"min height": func() { tc.box.ComputeMinIntrinsicHeight(0) },
				"max height": func() { tc.box.ComputeMaxIntrinsicHeight(0) },
			} {
				err, ok := panicOf(call).(error)
				if !ok {
					t.Errorf("%s: no panic", name)
					continue
				}
				appErr, _ := errors.FromError(err)
				if appErr == nil || appErr.Tag != "ViewportIntrinsicsUnsupported" {
					t.Errorf("%s: error = %v", name, err)
					continue
				}
				for _, w := range tc.want {
					if !strings.Contains(appErr.Message, w) {
						t.Errorf("%s: message lacks %q:\n%s", name, w, appErr.Message)
					}
				}
			}
		})
	}
}

// implicitScrollOffset is a viewport offset that allows implicit scrolling and
// records the jumps.
type implicitScrollOffset struct {
	*viewportOffsetSpy
	jumps    []float64
	animates []float64
}

func newImplicitScrollOffset(pixels float64) *implicitScrollOffset {
	return &implicitScrollOffset{viewportOffsetSpy: newViewportOffsetSpy(pixels)}
}
func (o *implicitScrollOffset) AllowImplicitScrolling() bool { return true }
func (o *implicitScrollOffset) JumpTo(to float64) {
	o.jumps = append(o.jumps, to)
	o.pixels = to
	o.NotifyListeners()
}
func (o *implicitScrollOffset) AnimateTo(to float64, _ time.Duration, _ animation.Curve) *scheduler.TickerFuture {
	o.animates = append(o.animates, to)
	o.pixels = to
	o.NotifyListeners()
	return scheduler.NewTickerFutureComplete()
}

func TestViewportShowOnScreen(t *testing.T) {
	boxes := sizedBoxes(20, 300, 100)
	manager := newTestChildManager(asBoxes(boxes)...)
	offset := newImplicitScrollOffset(0)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, offset, rendering.RenderViewportOpts{Children: []rendering.Sliver{manager.createList()}})
	h := newHarness(t)
	layout(t, h, root, withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 300, Height: 200})))
	// item 4 is at 400..500, below the viewport of 200 (but in its cache area):
	// showing it scrolls to 300
	boxes[4].ShowOnScreen(rendering.ShowOnScreenOpts{})
	if len(offset.jumps) != 1 || offset.jumps[0] != 300 {
		t.Errorf("jumps = %v, want [300]", offset.jumps)
	}
	// with a duration the offset animates
	pumpFrame(h, phaseLayout)
	boxes[7].ShowOnScreen(rendering.ShowOnScreenOpts{Duration: 100 * time.Millisecond})
	if len(offset.animates) != 1 || offset.animates[0] != 600 {
		t.Errorf("animates = %v, want [600]", offset.animates)
	}
	// an item that is on screen already does not scroll
	pumpFrame(h, phaseLayout)
	jumps := len(offset.jumps)
	boxes[6].ShowOnScreen(rendering.ShowOnScreenOpts{})
	if len(offset.jumps) != jumps {
		t.Errorf("showing a visible item jumped to %v", offset.jumps[jumps:])
	}
}

func TestRevealedOffsetClampOffset(t *testing.T) {
	leading := rendering.RevealedOffset{Offset: 100}
	trailing := rendering.RevealedOffset{Offset: 200}
	tests := []struct {
		current float64
		want    foundation.Opt[rendering.RevealedOffset]
	}{
		{50, foundation.Some(leading)},
		{150, foundation.None[rendering.RevealedOffset]()},
		{250, foundation.Some(trailing)},
	}
	for _, tc := range tests {
		if got := rendering.RevealedOffsetClampOffset(leading, trailing, tc.current); got != tc.want {
			t.Errorf("RevealedOffsetClampOffset(current %v) = %v, want %v", tc.current, got, tc.want)
		}
		// the edges can come in either order
		if got := rendering.RevealedOffsetClampOffset(trailing, leading, tc.current); got != tc.want {
			t.Errorf("inverted: RevealedOffsetClampOffset(current %v) = %v, want %v", tc.current, got, tc.want)
		}
	}
	if got := (rendering.RevealedOffset{Offset: 1, Rect: ui.NewRectFromLTWH(0, 0, 2, 3)}).String(); got != "RevealedOffset(offset: 1.0, rect: Rect.fromLTRB(0.0, 0.0, 2.0, 3.0))" {
		t.Errorf("String() = %q", got)
	}
}

func TestViewportSemanticsChildren(t *testing.T) {
	boxes := sizedBoxes(5, 800, 300)
	var slivers []rendering.Sliver
	for _, b := range boxes {
		slivers = append(slivers, rendering.NewRenderSliverToBoxAdapter(b))
	}
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0))},
		Children:               slivers,
	})
	layout(t, newHarness(t), root)
	// The viewport shows 2 slivers and has no cache
	var visited []rendering.RenderObject
	root.VisitChildrenForSemantics(func(c rendering.RenderObject) { visited = append(visited, c) })
	if len(visited) != 2 {
		t.Fatalf("visited %d slivers for semantics, want 2", len(visited))
	}
	// in the order they are painted
	if visited[0] != rendering.RenderObject(slivers[1]) || visited[1] != rendering.RenderObject(slivers[0]) {
		t.Errorf("visited %v", visited)
	}
}
