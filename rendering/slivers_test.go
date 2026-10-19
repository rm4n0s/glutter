package rendering_test

import (
	"github.com/rm4n0s/errors"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// RenderSizedBox is the class of rendering_tester.dart: a box of a fixed size
// that is sized by its parent.
type RenderSizedBox struct {
	rendering.RenderBox
	preferred ui.Size
}

func newRenderSizedBox(w, h float64) *RenderSizedBox {
	b := &RenderSizedBox{preferred: ui.Size{Width: w, Height: h}}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *RenderSizedBox) SizedByParent() bool { return true }
func (b *RenderSizedBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Constrain(b.preferred)
}
func (b *RenderSizedBox) PerformResize()                            { b.SetSize(b.BoxConstraints().Constrain(b.preferred)) }
func (b *RenderSizedBox) PerformLayout()                            {}
func (b *RenderSizedBox) ComputeMinIntrinsicWidth(float64) float64  { return b.preferred.Width }
func (b *RenderSizedBox) ComputeMaxIntrinsicWidth(float64) float64  { return b.preferred.Width }
func (b *RenderSizedBox) ComputeMinIntrinsicHeight(float64) float64 { return b.preferred.Height }
func (b *RenderSizedBox) ComputeMaxIntrinsicHeight(float64) float64 { return b.preferred.Height }
func (b *RenderSizedBox) HitTestSelf(ui.Offset) bool                { return true }

// sliverBoxes are five slivers that hold a sized box each.
func sliverBoxes(w, h float64) (boxes [5]*RenderSizedBox, slivers []rendering.Sliver) {
	for i := range boxes {
		boxes[i] = newRenderSizedBox(w, h)
		slivers = append(slivers, rendering.NewRenderSliverToBoxAdapter(boxes[i]))
	}
	return boxes, slivers
}

func paintOriginOf(s rendering.RenderObject) ui.Offset {
	return rendering.SliverPhysicalParentDataOf(s).PaintOffset
}

type boxCheck struct {
	box         string
	local, want ui.Offset
}

type originCheck struct {
	sliver string
	want   ui.Offset
}

type viewportStep struct {
	offset  float64
	checks  []boxCheck
	origins []originCheck
}

func TestViewportBasic(t *testing.T) {
	tests := []struct {
		name      string
		shrink    bool
		axis      painting.AxisDirection
		cross     painting.AxisDirection
		box       ui.Size
		steps     []viewportStep
		hit       ui.Offset
		hitTarget string
	}{
		{name: "RenderViewport basic test - down", shrink: false, axis: painting.AxisDirectionDown, cross: painting.AxisDirectionRight, box: ui.Size{Width: 100.0, Height: 400.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 0)}, {"b", off(0, 0), off(0, 400)}, {"c", off(0, 0), off(0, 800)}, {"d", off(0, 0), off(0, 1200)}, {"e", off(0, 0), off(0, 1600)}, {"a", off(800, 400), off(800, 400)}, {"b", off(800, 400), off(800, 800)}, {"c", off(800, 400), off(800, 1200)}, {"d", off(800, 400), off(800, 1600)}, {"e", off(800, 400), off(800, 2000)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(0, -200)}, {"b", off(0, 0), off(0, 200)}, {"c", off(0, 0), off(0, 600)}, {"d", off(0, 0), off(0, 1000)}, {"e", off(0, 0), off(0, 1400)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(0, -600)}, {"b", off(0, 0), off(0, -200)}, {"c", off(0, 0), off(0, 200)}, {"d", off(0, 0), off(0, 600)}, {"e", off(0, 0), off(0, 1000)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(0, -900)}, {"b", off(0, 0), off(0, -500)}, {"c", off(0, 0), off(0, -100)}, {"d", off(0, 0), off(0, 300)}, {"e", off(0, 0), off(0, 700)}}, origins: []originCheck{}},
			},
			hit: off(130.0, 150.0), hitTarget: "c",
		},
		{name: "RenderViewport basic test - up", shrink: false, axis: painting.AxisDirectionUp, cross: painting.AxisDirectionRight, box: ui.Size{Width: 100.0, Height: 400.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 200)}, {"b", off(0, 0), off(0, -200)}, {"c", off(0, 0), off(0, -600)}, {"d", off(0, 0), off(0, -1000)}, {"e", off(0, 0), off(0, -1400)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(0, 400)}, {"b", off(0, 0), off(0, 0)}, {"c", off(0, 0), off(0, -400)}, {"d", off(0, 0), off(0, -800)}, {"e", off(0, 0), off(0, -1200)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(0, 800)}, {"b", off(0, 0), off(0, 400)}, {"c", off(0, 0), off(0, 0)}, {"d", off(0, 0), off(0, -400)}, {"e", off(0, 0), off(0, -800)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(0, 1100)}, {"b", off(0, 0), off(0, 700)}, {"c", off(0, 0), off(0, 300)}, {"d", off(0, 0), off(0, -100)}, {"e", off(0, 0), off(0, -500)}}, origins: []originCheck{}},
			},
			hit: off(150.0, 350.0), hitTarget: "c",
		},
		{name: "RenderViewport basic test - right", shrink: false, axis: painting.AxisDirectionRight, cross: painting.AxisDirectionDown, box: ui.Size{Width: 400.0, Height: 100.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 0)}, {"b", off(0, 0), off(400, 0)}, {"c", off(0, 0), off(800, 0)}, {"d", off(0, 0), off(1200, 0)}, {"e", off(0, 0), off(1600, 0)}}, origins: []originCheck{{"A", off(0, 0)}, {"B", off(400, 0)}, {"C", off(800, 0)}, {"D", off(1200, 0)}, {"E", off(1600, 0)}}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(-200, 0)}, {"b", off(0, 0), off(200, 0)}, {"c", off(0, 0), off(600, 0)}, {"d", off(0, 0), off(1000, 0)}, {"e", off(0, 0), off(1400, 0)}}, origins: []originCheck{{"A", off(0, 0)}, {"B", off(200, 0)}, {"C", off(600, 0)}, {"D", off(1000, 0)}, {"E", off(1400, 0)}}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(-600, 0)}, {"b", off(0, 0), off(-200, 0)}, {"c", off(0, 0), off(200, 0)}, {"d", off(0, 0), off(600, 0)}, {"e", off(0, 0), off(1000, 0)}}, origins: []originCheck{{"A", off(0, 0)}, {"B", off(0, 0)}, {"C", off(200, 0)}, {"D", off(600, 0)}, {"E", off(1000, 0)}}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(-900, 0)}, {"b", off(0, 0), off(-500, 0)}, {"c", off(0, 0), off(-100, 0)}, {"d", off(0, 0), off(300, 0)}, {"e", off(0, 0), off(700, 0)}}, origins: []originCheck{{"A", off(0, 0)}, {"B", off(0, 0)}, {"C", off(0, 0)}, {"D", off(300, 0)}, {"E", off(700, 0)}}},
			},
			hit: off(150.0, 450.0), hitTarget: "c",
		},
		{name: "RenderViewport basic test - left", shrink: false, axis: painting.AxisDirectionLeft, cross: painting.AxisDirectionDown, box: ui.Size{Width: 400.0, Height: 100.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(400, 0)}, {"b", off(0, 0), off(0, 0)}, {"c", off(0, 0), off(-400, 0)}, {"d", off(0, 0), off(-800, 0)}, {"e", off(0, 0), off(-1200, 0)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(600, 0)}, {"b", off(0, 0), off(200, 0)}, {"c", off(0, 0), off(-200, 0)}, {"d", off(0, 0), off(-600, 0)}, {"e", off(0, 0), off(-1000, 0)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(1000, 0)}, {"b", off(0, 0), off(600, 0)}, {"c", off(0, 0), off(200, 0)}, {"d", off(0, 0), off(-200, 0)}, {"e", off(0, 0), off(-600, 0)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(1300, 0)}, {"b", off(0, 0), off(900, 0)}, {"c", off(0, 0), off(500, 0)}, {"d", off(0, 0), off(100, 0)}, {"e", off(0, 0), off(-300, 0)}}, origins: []originCheck{}},
			},
			hit: off(550.0, 150.0), hitTarget: "c",
		},
		{name: "RenderShrinkWrappingViewport basic test - down", shrink: true, axis: painting.AxisDirectionDown, cross: painting.AxisDirectionRight, box: ui.Size{Width: 100.0, Height: 400.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 0)}, {"b", off(0, 0), off(0, 400)}, {"c", off(0, 0), off(0, 800)}, {"d", off(0, 0), off(0, 1200)}, {"e", off(0, 0), off(0, 1600)}, {"a", off(800, 400), off(800, 400)}, {"b", off(800, 400), off(800, 800)}, {"c", off(800, 400), off(800, 1200)}, {"d", off(800, 400), off(800, 1600)}, {"e", off(800, 400), off(800, 2000)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(0, -200)}, {"b", off(0, 0), off(0, 200)}, {"c", off(0, 0), off(0, 600)}, {"d", off(0, 0), off(0, 1000)}, {"e", off(0, 0), off(0, 1400)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(0, -600)}, {"b", off(0, 0), off(0, -200)}, {"c", off(0, 0), off(0, 200)}, {"d", off(0, 0), off(0, 600)}, {"e", off(0, 0), off(0, 1000)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(0, -900)}, {"b", off(0, 0), off(0, -500)}, {"c", off(0, 0), off(0, -100)}, {"d", off(0, 0), off(0, 300)}, {"e", off(0, 0), off(0, 700)}}, origins: []originCheck{}},
			},
			hit: off(130.0, 150.0), hitTarget: "c",
		},
		{name: "RenderShrinkWrappingViewport basic test - up", shrink: true, axis: painting.AxisDirectionUp, cross: painting.AxisDirectionRight, box: ui.Size{Width: 100.0, Height: 400.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 200)}, {"b", off(0, 0), off(0, -200)}, {"c", off(0, 0), off(0, -600)}, {"d", off(0, 0), off(0, -1000)}, {"e", off(0, 0), off(0, -1400)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(0, 400)}, {"b", off(0, 0), off(0, 0)}, {"c", off(0, 0), off(0, -400)}, {"d", off(0, 0), off(0, -800)}, {"e", off(0, 0), off(0, -1200)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(0, 800)}, {"b", off(0, 0), off(0, 400)}, {"c", off(0, 0), off(0, 0)}, {"d", off(0, 0), off(0, -400)}, {"e", off(0, 0), off(0, -800)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(0, 1100)}, {"b", off(0, 0), off(0, 700)}, {"c", off(0, 0), off(0, 300)}, {"d", off(0, 0), off(0, -100)}, {"e", off(0, 0), off(0, -500)}}, origins: []originCheck{}},
			},
			hit: off(150.0, 350.0), hitTarget: "c",
		},
		{name: "RenderShrinkWrappingViewport basic test - right", shrink: true, axis: painting.AxisDirectionRight, cross: painting.AxisDirectionDown, box: ui.Size{Width: 400.0, Height: 100.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(0, 0)}, {"b", off(0, 0), off(400, 0)}, {"c", off(0, 0), off(800, 0)}, {"d", off(0, 0), off(1200, 0)}, {"e", off(0, 0), off(1600, 0)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(-200, 0)}, {"b", off(0, 0), off(200, 0)}, {"c", off(0, 0), off(600, 0)}, {"d", off(0, 0), off(1000, 0)}, {"e", off(0, 0), off(1400, 0)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(-600, 0)}, {"b", off(0, 0), off(-200, 0)}, {"c", off(0, 0), off(200, 0)}, {"d", off(0, 0), off(600, 0)}, {"e", off(0, 0), off(1000, 0)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(-900, 0)}, {"b", off(0, 0), off(-500, 0)}, {"c", off(0, 0), off(-100, 0)}, {"d", off(0, 0), off(300, 0)}, {"e", off(0, 0), off(700, 0)}}, origins: []originCheck{}},
			},
			hit: off(150.0, 450.0), hitTarget: "c",
		},
		{name: "RenderShrinkWrappingViewport basic test - left", shrink: true, axis: painting.AxisDirectionLeft, cross: painting.AxisDirectionDown, box: ui.Size{Width: 400.0, Height: 100.0},
			steps: []viewportStep{
				{offset: 0, checks: []boxCheck{{"a", off(0, 0), off(400, 0)}, {"b", off(0, 0), off(0, 0)}, {"c", off(0, 0), off(-400, 0)}, {"d", off(0, 0), off(-800, 0)}, {"e", off(0, 0), off(-1200, 0)}}, origins: []originCheck{}},
				{offset: 200, checks: []boxCheck{{"a", off(0, 0), off(600, 0)}, {"b", off(0, 0), off(200, 0)}, {"c", off(0, 0), off(-200, 0)}, {"d", off(0, 0), off(-600, 0)}, {"e", off(0, 0), off(-1000, 0)}}, origins: []originCheck{}},
				{offset: 600, checks: []boxCheck{{"a", off(0, 0), off(1000, 0)}, {"b", off(0, 0), off(600, 0)}, {"c", off(0, 0), off(200, 0)}, {"d", off(0, 0), off(-200, 0)}, {"e", off(0, 0), off(-600, 0)}}, origins: []originCheck{}},
				{offset: 900, checks: []boxCheck{{"a", off(0, 0), off(1300, 0)}, {"b", off(0, 0), off(900, 0)}, {"c", off(0, 0), off(500, 0)}, {"d", off(0, 0), off(100, 0)}, {"e", off(0, 0), off(-300, 0)}}, origins: []originCheck{}},
			},
			hit: off(550.0, 150.0), hitTarget: "c",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boxes, slivers := sliverBoxes(tc.box.Width, tc.box.Height)
			var root rendering.Box
			var setOffset func(rendering.ViewportOffset)
			if tc.shrink {
				v := rendering.NewRenderShrinkWrappingViewport(tc.cross, rendering.NewViewportOffsetZero(),
					rendering.RenderShrinkWrappingViewportOpts{RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: tc.axis}, Children: slivers})
				root, setOffset = v, v.SetOffset
			} else {
				v := rendering.NewRenderViewport(tc.cross, rendering.NewViewportOffsetZero(),
					rendering.RenderViewportOpts{RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: tc.axis}, Children: slivers})
				root, setOffset = v, v.SetOffset
			}
			h := newHarness(t)
			layout(t, h, root)
			if got := root.Size(); got != (ui.Size{Width: 800, Height: 600}) {
				t.Fatalf("size = %v, want 800x600", got)
			}
			for i, step := range tc.steps {
				if i > 0 {
					setOffset(rendering.NewViewportOffsetFixed(step.offset))
					pumpFrame(h, phaseLayout)
				}
				for _, c := range step.checks {
					if got := boxes[c.box[0]-'a'].LocalToGlobal(c.local, nil); got != c.want {
						t.Errorf("offset %v: %s.localToGlobal(%v) = %v, want %v", step.offset, c.box, c.local, got, c.want)
					}
				}
				for _, c := range step.origins {
					if got := paintOriginOf(slivers[c.sliver[0]-'A']); got != c.want {
						t.Errorf("offset %v: paint origin of sliver %s = %v, want %v", step.offset, c.sliver, got, c.want)
					}
				}
			}
			result := rendering.NewBoxHitTestResult()
			root.HitTest(result, tc.hit)
			if got := result.Path()[0].Target(); got != gestures.HitTestTarget(boxes[tc.hitTarget[0]-'a']) {
				t.Errorf("hit target = %v, want box %s", got, tc.hitTarget)
			}
		})
	}
}

func TestViewportNoChildren(t *testing.T) {
	skipUnlessDebug(t)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero())
	if got := deepInfo(root); got != viewportNoChildrenBeforeLayout {
		t.Errorf("before layout:\n%s\nwant:\n%s", got, viewportNoChildrenBeforeLayout)
	}
	h := newHarness(t)
	layout(t, h, root)
	root.SetOffset(rendering.NewViewportOffsetFixed(900))
	if got := deepInfo(root); got != viewportNoChildrenAfterLayout {
		t.Errorf("after layout:\n%s\nwant:\n%s", got, viewportNoChildrenAfterLayout)
	}
	pumpFrame(h, phaseLayout)
}

func TestViewportDownDescription(t *testing.T) {
	skipUnlessDebug(t)
	_, slivers := sliverBoxes(100, 400)
	root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: slivers})
	layout(t, newHarness(t), root)
	if got := deepInfo(root); got != viewportDownDescription {
		t.Errorf("description:\n%s\nwant:\n%s", got, viewportDownDescription)
	}
}

func TestShrinkWrappingViewportNoChildren(t *testing.T) {
	root := rendering.NewRenderShrinkWrappingViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero())
	h := newHarness(t)
	layout(t, h, root)
	root.SetOffset(rendering.NewViewportOffsetFixed(900))
	pumpFrame(h, phaseLayout)
}

func TestShrinkWrappingViewportShrinkWraps(t *testing.T) {
	tests := []struct {
		name  string
		axis  painting.AxisDirection
		boxes []ui.Size
		want  ui.Size
	}{
		{"1 child", painting.AxisDirectionLeft, []ui.Size{{Width: 400, Height: 100}}, ui.Size{Width: 400, Height: 600}},
		{"2 children", painting.AxisDirectionRight, []ui.Size{{Width: 300, Height: 100}, {Width: 150, Height: 100}}, ui.Size{Width: 450, Height: 600}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var slivers []rendering.Sliver
			for _, s := range tc.boxes {
				slivers = append(slivers, rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(s.Width, s.Height)))
			}
			child := rendering.NewRenderShrinkWrappingViewport(painting.AxisDirectionDown, rendering.NewViewportOffsetFixed(200),
				rendering.RenderShrinkWrappingViewportOpts{RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: tc.axis}, Children: slivers})
			root := rendering.NewRenderPositionedBox(child)
			layout(t, newHarness(t), root)
			if root.Size() != (ui.Size{Width: 800, Height: 600}) {
				t.Errorf("root size = %v, want 800x600", root.Size())
			}
			if child.Size() != tc.want {
				t.Errorf("child size = %v, want %v", child.Size(), tc.want)
			}
		})
	}
}

func TestSliverGeometryToString(t *testing.T) {
	tests := []struct {
		name string
		g    rendering.SliverGeometry
		want string
	}{
		{"zero", rendering.SliverGeometryZero, "SliverGeometry(scrollExtent: 0.0, hidden, maxPaintExtent: 0.0)"},
		{"visible", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, PaintExtent: 50, LayoutExtent: foundation.Some(20.0), Visible: foundation.Some(true)}),
			"SliverGeometry(scrollExtent: 100.0, paintExtent: 50.0, layoutExtent: 20.0, maxPaintExtent: 0.0, cacheExtent: 20.0)"},
		{"hidden", rendering.NewSliverGeometry(rendering.SliverGeometryOpts{ScrollExtent: 100, LayoutExtent: foundation.Some(20.0)}),
			"SliverGeometry(scrollExtent: 100.0, hidden, layoutExtent: 20.0, maxPaintExtent: 0.0, cacheExtent: 20.0)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSliverBounds(t *testing.T) {
	tests := []struct {
		name  string
		axis  painting.AxisDirection
		cross painting.AxisDirection
		box   ui.Size
		want  func(root ui.Size) ui.Rect
	}{
		{"vertical", painting.AxisDirectionDown, painting.AxisDirectionRight, ui.Size{Width: 400, Height: 150},
			func(root ui.Size) ui.Rect { return ui.NewRectFromLTWH(0, 0, root.Width, 150) }},
		{"horizontal", painting.AxisDirectionRight, painting.AxisDirectionDown, ui.Size{Width: 150, Height: 400},
			func(root ui.Size) ui.Rect { return ui.NewRectFromLTWH(0, 0, 150, root.Height) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sliver := rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(tc.box.Width, tc.box.Height))
			root := rendering.NewRenderViewport(tc.cross, rendering.NewViewportOffsetZero(),
				rendering.RenderViewportOpts{RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: tc.axis}, Children: []rendering.Sliver{sliver}})
			layout(t, newHarness(t), root)
			want := tc.want(root.Size())
			if got := sliver.PaintBounds(); got != want {
				t.Errorf("paintBounds = %v, want %v", got, want)
			}
			if got := sliver.SemanticBounds(); got != want {
				t.Errorf("semanticBounds = %v, want %v", got, want)
			}
		})
	}
}

func TestPrecedingScrollExtent(t *testing.T) {
	tests := []struct {
		name   string
		count  int
		offset float64
		want   float64
	}{
		{"is 0.0 for the first sliver in the list", 1, 0, 0},
		{"accumulates over multiple slivers", 3, 0, 300},
		{"is not impacted by scrollOffset", 3, 100, 300},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var slivers []rendering.Sliver
			for i := 0; i < tc.count; i++ {
				slivers = append(slivers, rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(800, 150)))
			}
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetFixed(tc.offset), rendering.RenderViewportOpts{Children: slivers})
			layout(t, newHarness(t), root)
			if got := slivers[len(slivers)-1].SliverConstraints().PrecedingScrollExtent; got != tc.want {
				t.Errorf("precedingScrollExtent = %v, want %v", got, tc.want)
			}
		})
	}
}

type dummyHitTarget struct{}

func (dummyHitTarget) HandleEvent(gestures.PointerEvent, gestures.HitTestEntry) {}

func TestSliverHitTestResultWrapping(t *testing.T) {
	entry1 := gestures.NewHitTestEntry(dummyHitTarget{})
	entry2 := gestures.NewHitTestEntry(dummyHitTarget{})
	entry3 := gestures.NewHitTestEntry(dummyHitTarget{})
	transform := ui.NewMatrix4TranslationValues(40, 150, 0)

	wrapped := gestures.NewHitTestResult()
	wrapped.PushTransform(transform)
	wrapped.Add(entry1)
	if got := wrapped.Path(); len(got) != 1 || got[0] != gestures.HitTestEntry(entry1) {
		t.Fatalf("path = %v, want [entry1]", got)
	}
	if !entry1.Transform().Equals(transform) {
		t.Errorf("entry1.transform = %v, want %v", entry1.Transform(), transform)
	}

	wrapping := rendering.NewSliverHitTestResultWrap(wrapped)
	if got := wrapping.Path(); len(got) != 1 || got[0] != gestures.HitTestEntry(entry1) {
		t.Fatalf("wrapping path = %v, want [entry1]", got)
	}

	wrapping.Add(entry2)
	if len(wrapping.Path()) != 2 || len(wrapped.Path()) != 2 {
		t.Fatalf("paths = %d and %d entries, want 2 and 2", len(wrapping.Path()), len(wrapped.Path()))
	}
	if !entry2.Transform().Equals(transform) {
		t.Errorf("entry2.transform = %v, want %v", entry2.Transform(), transform)
	}

	wrapped.Add(entry3)
	if len(wrapping.Path()) != 3 || len(wrapped.Path()) != 3 {
		t.Fatalf("paths = %d and %d entries, want 3 and 3", len(wrapping.Path()), len(wrapped.Path()))
	}
	if !entry3.Transform().Equals(transform) {
		t.Errorf("entry3.transform = %v, want %v", entry3.Transform(), transform)
	}
}

func TestSliverAddWithAxisOffset(t *testing.T) {
	tests := []struct {
		name                        string
		mainOffset, crossOffset     float64
		mainPosition, crossPosition float64
		hit                         bool
		wantMain, wantCross         float64
	}{
		{"zero", 0, 0, 0, 0, true, 0, 0},
		{"positive offsets", 5, 6, 10, 20, false, 10 - 5, 20 - 6},
		{"negative offsets", -5, -6, 10, 20, false, 10 + 5, 20 + 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rendering.NewSliverHitTestResult()
			var mains, crosses []float64
			zero := ui.Offset{}
			isHit := result.AddWithAxisOffset(&zero, tc.mainOffset, tc.crossOffset, tc.mainPosition, tc.crossPosition,
				func(_ *rendering.SliverHitTestResult, main, cross float64) bool {
					mains, crosses = append(mains, main), append(crosses, cross)
					return tc.hit
				})
			if isHit != tc.hit {
				t.Errorf("isHit = %v, want %v", isHit, tc.hit)
			}
			if len(mains) != 1 || mains[0] != tc.wantMain || crosses[0] != tc.wantCross {
				t.Errorf("positions = %v, %v; want %v, %v", mains, crosses, tc.wantMain, tc.wantCross)
			}
		})
	}
}

func TestSliverAddWithAxisOffsetNonZeroPaintOffset(t *testing.T) {
	result := rendering.NewSliverHitTestResult()
	entry := gestures.NewHitTestEntry(dummyHitTarget{})
	paintOffset := ui.Offset{Dx: 7, Dy: 11}
	var gotMain, gotCross float64
	isHit := result.AddWithAxisOffset(&paintOffset, 5, 6, 10, 20, func(r *rendering.SliverHitTestResult, main, cross float64) bool {
		gotMain, gotCross = main, cross
		r.Add(entry)
		return true
	})
	if !isHit {
		t.Error("isHit = false, want true")
	}
	if gotMain != 10-5 || gotCross != 20-6 {
		t.Errorf("positions = %v, %v; want 5, 14", gotMain, gotCross)
	}
	m := entry.Transform().Clone()
	m.Translate(float32(paintOffset.Dx), float32(paintOffset.Dy), 0)
	if !m.Equals(ui.NewMatrix4Identity()) {
		t.Errorf("transform with the paint offset added back = %v, want identity", m)
	}
}

func TestSliverConstraintsInvalid(t *testing.T) {
	skipUnlessDebug(t)
	nan := math.NaN()
	tests := []struct {
		name string
		c    rendering.SliverConstraints
		want string
	}{
		{"NaN on all double properties", rendering.SliverConstraints{
			AxisDirection: painting.AxisDirectionDown, CacheOrigin: nan, CrossAxisDirection: painting.AxisDirectionLeft, CrossAxisExtent: nan,
			GrowthDirection: rendering.GrowthDirectionForward, Overlap: nan, PrecedingScrollExtent: nan, RemainingCacheExtent: nan,
			RemainingPaintExtent: nan, ScrollOffset: nan, UserScrollDirection: rendering.ScrollDirectionIdle, ViewportMainAxisExtent: nan,
		}, "SliverConstraints is not valid:\n" +
			"  The \"scrollOffset\" is NaN.\n" +
			"  The \"overlap\" is NaN.\n" +
			"  The \"crossAxisExtent\" is NaN.\n" +
			"  The \"scrollOffset\" is NaN, expected greater than or equal to zero.\n" +
			"  The \"viewportMainAxisExtent\" is NaN, expected greater than or equal to zero.\n" +
			"  The \"remainingPaintExtent\" is NaN, expected greater than or equal to zero.\n" +
			"  The \"remainingCacheExtent\" is NaN, expected greater than or equal to zero.\n" +
			"  The \"cacheOrigin\" is NaN, expected less than or equal to zero.\n" +
			"  The \"precedingScrollExtent\" is NaN, expected greater than or equal to zero.\n" +
			"  The constraints are not normalized.\n" +
			"The offending constraints were:\n" +
			"  SliverConstraints(AxisDirection.down, GrowthDirection.forward, ScrollDirection.idle, scrollOffset: NaN, precedingScrollExtent: NaN, remainingPaintExtent: NaN, overlap: NaN, crossAxisExtent: NaN, crossAxisDirection: AxisDirection.left, viewportMainAxisExtent: NaN, remainingCacheExtent: NaN, cacheOrigin: NaN)"},
		{"sign of relevant double properties", rendering.SliverConstraints{
			AxisDirection: painting.AxisDirectionDown, CacheOrigin: 1, CrossAxisDirection: painting.AxisDirectionLeft,
			GrowthDirection: rendering.GrowthDirectionForward, PrecedingScrollExtent: -1, RemainingCacheExtent: -1,
			RemainingPaintExtent: -1, ScrollOffset: -1, UserScrollDirection: rendering.ScrollDirectionIdle,
		}, "SliverConstraints is not valid:\n" +
			"  The \"scrollOffset\" is negative.\n" +
			"  The \"remainingPaintExtent\" is negative.\n" +
			"  The \"remainingCacheExtent\" is negative.\n" +
			"  The \"cacheOrigin\" is positive.\n" +
			"  The \"precedingScrollExtent\" is negative.\n" +
			"  The constraints are not normalized.\n" +
			"The offending constraints were:\n" +
			"  SliverConstraints(AxisDirection.down, GrowthDirection.forward, ScrollDirection.idle, scrollOffset: -1.0, precedingScrollExtent: -1.0, remainingPaintExtent: -1.0, crossAxisExtent: 0.0, crossAxisDirection: AxisDirection.left, viewportMainAxisExtent: 0.0, remainingCacheExtent: -1.0, cacheOrigin: 1.0)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requireTagMessage(t, "SliverConstraintsInvalid", tc.want, func() { tc.c.DebugAssertIsValid() })
		})
	}
}

func TestSliverLayoutDimensionsComparable(t *testing.T) {
	a := rendering.SliverLayoutDimensions{ScrollOffset: 1, PrecedingScrollExtent: 2, ViewportMainAxisExtent: 3, CrossAxisExtent: 4}
	b := rendering.SliverLayoutDimensions{ScrollOffset: 1, PrecedingScrollExtent: 2, ViewportMainAxisExtent: 3, CrossAxisExtent: 4}
	if a != b {
		t.Errorf("equal dimensions compare unequal: %v and %v", a, b)
	}
}

// requireTagMessage fails unless call panics with an error that has the tag and
// the message.
func requireTagMessage(t *testing.T, tag, message string, call func()) {
	t.Helper()
	v := panicOf(call)
	err, ok := v.(error)
	if !ok {
		t.Fatalf("expected a panic with tag %s, got %v", tag, v)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if !appErr.HasRoute(tag) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), tag)
	}
	if appErr.Message != message {
		t.Errorf("message:\n%s\nwant:\n%s", appErr.Message, message)
	}
}
