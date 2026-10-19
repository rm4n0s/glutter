package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// layoutWatcher is RenderLayoutWatcher: a proxy box that counts its layouts.
type layoutWatcher struct {
	rendering.RenderProxyBox
	layouts int
}

func newLayoutWatcher(child rendering.Box) *layoutWatcher {
	w := &layoutWatcher{}
	rendering.InitRenderProxyBox(&w.RenderProxyBox, w, child)
	return w
}

func (w *layoutWatcher) PerformLayout() {
	w.layouts++
	w.RenderProxyBox.PerformLayout()
}

func TestViewportImpactOfLayout(t *testing.T) {
	box := newRenderSizedBox(100, 400)
	sliver := rendering.NewRenderSliverToBoxAdapter(box)
	viewport := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{Children: []rendering.Sliver{sliver}})
	root := newLayoutWatcher(viewport)
	h := newHarness(t)
	if root.layouts != 0 {
		t.Fatalf("layouts = %d, want 0", root.layouts)
	}
	layout(t, h, root)
	centerOf := func(b *RenderSizedBox) ui.Offset {
		return b.LocalToGlobal(ui.Offset{Dx: b.Size().Width / 2, Dy: b.Size().Height / 2}, nil)
	}
	check := func(step string, want ui.Offset) {
		t.Helper()
		if root.layouts != 1 {
			t.Errorf("%s: layouts = %d, want 1", step, root.layouts)
		}
		if got := centerOf(box); got != want {
			t.Errorf("%s: the center of the box is at %v, want %v", step, got, want)
		}
	}
	check("first layout", off(400, 200))

	box = newRenderSizedBox(100, 300)
	sliver.SetChild(box)
	if root.layouts != 1 {
		t.Fatalf("layouts = %d, want 1", root.layouts)
	}
	pumpFrame(h, phaseLayout)
	check("new child", off(400, 150))

	viewport.SetOffset(rendering.NewViewportOffsetFixed(20))
	pumpFrame(h, phaseLayout)
	check("offset 20", off(400, 130))

	viewport.SetOffset(rendering.NewViewportOffsetFixed(-20))
	pumpFrame(h, phaseLayout)
	check("offset -20", off(400, 170))

	viewport.SetAnchor(20.0 / 600.0)
	pumpFrame(h, phaseLayout)
	check("anchor", off(400, 190))

	viewport.SetAxisDirection(painting.AxisDirectionUp)
	pumpFrame(h, phaseLayout)
	check("axis direction up", off(400, 600-190))
}
