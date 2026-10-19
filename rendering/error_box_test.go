package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestRenderErrorBox(t *testing.T) {
	t.Run("takes the room it is given", func(t *testing.T) {
		box := rendering.NewRenderErrorBox("boom")
		layout(t, newHarness(t), box, atPhase(withConstraints(rendering.BoxConstraints{MaxWidth: 300, MaxHeight: 200}), phasePaint))
		if box.Size() != (ui.Size{Width: 300, Height: 200}) {
			t.Errorf("size = %v, want 300x200", box.Size())
		}
		if got := box.GetDryLayout(rendering.BoxConstraints{MaxWidth: 50, MaxHeight: 60}); got != (ui.Size{Width: 50, Height: 60}) {
			t.Errorf("dry layout = %v, want 50x60", got)
		}
		if box.Message() != "boom" {
			t.Errorf("message = %q", box.Message())
		}
	})
	t.Run("intrinsics and hit testing", func(t *testing.T) {
		box := rendering.NewRenderErrorBox("")
		if got := box.ComputeMaxIntrinsicWidth(0); got != 100000 {
			t.Errorf("max intrinsic width = %v", got)
		}
		if got := box.ComputeMaxIntrinsicHeight(0); got != 100000 {
			t.Errorf("max intrinsic height = %v", got)
		}
		if !box.SizedByParent() || !box.HitTestSelf(ui.Offset{}) {
			t.Error("sized by parent and hit self")
		}
	})
	t.Run("colors depend on the build mode", func(t *testing.T) {
		box := rendering.NewRenderErrorBox("x")
		want := ui.NewColor(0xF0C0C0C0)
		if foundation.DebugMode {
			want = ui.NewColor(0xF0900000)
		}
		if box.BackgroundColor != want {
			t.Errorf("background = %v, want %v", box.BackgroundColor, want)
		}
	})
	t.Run("draws its text with the text engine", func(t *testing.T) {
		box := rendering.NewRenderErrorBox("an error happened", rendering.RenderErrorBoxOpts{TextEngine: renderingtest.BundledTextEngine()})
		h := newHarness(t)
		layout(t, h, box, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 400, Height: 300})), phaseComposite))
		if h.LastScene() == nil {
			t.Error("no scene was composited")
		}
	})
}

func TestRenderChildTypeMismatch(t *testing.T) {
	// A render view holds a box; a render object that is not a box is refused.
	h := newHarness(t)
	view := h.RenderView
	other := rendering.RenderObject(h.RenderView)
	requireTag(t, "RenderChildTypeMismatch", func() { view.SetChildObject(other) })
	view.SetChildObject(nil)
	if view.ChildObject() != nil {
		t.Error("the child was not cleared")
	}
	box := newSizedBox(1, 1)
	view.SetChildObject(box)
	if view.ChildObject() != rendering.RenderObject(box) {
		t.Error("ChildObject is not the box that was set")
	}

	flex := rendering.NewRenderFlex(nil)
	a, b := newSizedBox(1, 1), newSizedBox(2, 2)
	flex.InsertObject(a, nil)
	flex.InsertObject(b, a)
	if flex.FirstChild() != rendering.Box(a) || flex.LastChild() != rendering.Box(b) {
		t.Error("InsertObject did not put the children in order")
	}
	flex.MoveObject(b, nil)
	if flex.FirstChild() != rendering.Box(b) {
		t.Error("MoveObject did not move the child to the start")
	}
	flex.RemoveObject(b)
	if flex.ChildCount() != 1 {
		t.Errorf("RemoveObject left %d children", flex.ChildCount())
	}
	requireTag(t, "RenderChildTypeMismatch", func() { flex.InsertObject(other, nil) })
	requireTag(t, "RenderChildTypeMismatch", func() { flex.InsertObject(newSizedBox(1, 1), other) })
}
