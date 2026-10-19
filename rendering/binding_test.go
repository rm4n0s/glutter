package rendering_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func TestPipelineOwnerChildren(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, parent, child *rendering.PipelineOwner)
	}{
		{"adopted children are visited", func(t *testing.T, parent, child *rendering.PipelineOwner) {
			parent.AdoptChild(child)
			var got []*rendering.PipelineOwner
			parent.VisitChildren(func(c *rendering.PipelineOwner) { got = append(got, c) })
			if len(got) != 1 || got[0] != child {
				t.Errorf("visited %v, want the child", got)
			}
		}},
		{"dropped children are not visited", func(t *testing.T, parent, child *rendering.PipelineOwner) {
			parent.AdoptChild(child)
			parent.DropChild(child)
			n := 0
			parent.VisitChildren(func(*rendering.PipelineOwner) { n++ })
			if n != 0 {
				t.Errorf("visited %d children, want 0", n)
			}
		}},
		{"children share the debug options of the parent", func(t *testing.T, parent, child *rendering.PipelineOwner) {
			parent.AdoptChild(child)
			if child.DebugOptions() != parent.DebugOptions() {
				t.Error("the child does not share the parent's debug options")
			}
		}},
		{"a dropped child can be disposed", func(t *testing.T, parent, child *rendering.PipelineOwner) {
			parent.AdoptChild(child)
			parent.DropChild(child)
			child.Dispose()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, rendering.NewPipelineOwner(), rendering.NewPipelineOwner())
		})
	}
}

func TestPipelineOwnerFlushesChildrenOfTheTree(t *testing.T) {
	h := newHarness(t)
	box := newTBox(10, 10)
	h.SetRoot(box)
	h.Pump()
	if !box.HasSize() {
		t.Fatal("the child owner of the root was not laid out")
	}
	box.MarkNeedsLayout()
	if len(h.Owner.NodesNeedingLayout()) == 0 {
		t.Error("a marked node is not in its owner's list")
	}
	h.Pump()
	if len(h.Owner.NodesNeedingLayout()) != 0 {
		t.Error("layout did not clear the list")
	}
}

func TestRendererBindingFirstFrameDeferral(t *testing.T) {
	tests := []struct {
		name       string
		deferCalls int
		allowCalls int
		wantScenes int
	}{
		{"no deferral sends the first frame", 0, 0, 1},
		{"a deferred first frame is not sent", 1, 0, 0},
		{"allowing the first frame sends it", 1, 1, 1},
		{"two deferrals need two allows", 2, 1, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.SetRoot(newTBox(10, 10))
			for i := 0; i < tc.deferCalls; i++ {
				h.Binding.DeferFirstFrame()
			}
			for i := 0; i < tc.allowCalls; i++ {
				h.Binding.AllowFirstFrame()
			}
			h.Pump()
			if got := len(h.Scenes()); got != tc.wantScenes {
				t.Errorf("%d scenes, want %d", got, tc.wantScenes)
			}
		})
	}
}

func TestRendererBindingFramesAfterFirstAreNotDeferred(t *testing.T) {
	h := newHarness(t)
	box := newTBox(10, 10)
	h.SetRoot(box)
	h.Pump()
	h.Binding.DeferFirstFrame()
	box.MarkNeedsPaint()
	h.Pump()
	if got := len(h.Scenes()); got != 2 {
		t.Errorf("%d scenes, want 2: only the first frame can be deferred", got)
	}
	h.Binding.ResetFirstFrameSent()
	if h.Binding.SendFramesToEngine() {
		t.Error("frames go to the engine after ResetFirstFrameSent with a deferral")
	}
}

func TestRendererBindingRenderViews(t *testing.T) {
	h := newHarness(t)
	if got := len(h.Binding.RenderViews()); got != 1 {
		t.Fatalf("%d render views, want 1", got)
	}
	h.Binding.RemoveRenderView(h.RenderView)
	if got := len(h.Binding.RenderViews()); got != 0 {
		t.Errorf("%d render views after remove, want 0", got)
	}
	if got := h.Binding.DebugDumpRenderTree(); !strings.Contains(got, "No render tree root") {
		t.Errorf("dump of no views is %q", got)
	}
}

func TestRendererBindingDumps(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	h.SetRoot(newTPadding(5, newTBox(10, 10)))
	h.Pump()
	if got := h.Binding.DebugDumpRenderTree(); !strings.Contains(got, "tPadding") && !strings.Contains(got, "RenderView") {
		t.Errorf("render tree dump has neither the view nor the padding:\n%s", got)
	}
	if got := h.Binding.DebugDumpLayerTree(); got == "" {
		t.Error("layer tree dump is empty")
	}
	if got := h.Binding.DebugDumpPipelineOwnerTree(); !strings.Contains(got, "PipelineOwner") {
		t.Errorf("pipeline owner tree dump:\n%s", got)
	}
}

func TestRendererBindingMetricsChangeForcesFrame(t *testing.T) {
	h := newHarness(t)
	h.SetRoot(newBiggestBox())
	h.Pump()
	h.Dispatcher.UpdateViewConfiguration(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: ui.Size{Width: 400, Height: 300}, DevicePixelRatio: 1})
	h.Pump()
	if got := h.RenderView.Size(); got != (ui.Size{Width: 400, Height: 300}) {
		t.Errorf("size after a metrics change is %v, want 400x300", got)
	}
}

func TestRendererBindingForceRepaintMarksAll(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	box := newTBox(10, 10)
	h.SetRoot(box)
	h.Pump()
	h.Binding.ForceRepaint()
	found := false
	for _, n := range h.Owner.NodesNeedingPaint() {
		if n == rendering.RenderObject(box) {
			found = true
		}
	}
	if !found && !box.DebugNeedsPaint() {
		t.Error("ForceRepaint did not mark the child")
	}
}

func TestRepaintBoundaryLayersAreReused(t *testing.T) {
	h := newHarness(t)
	box := newTBox(10, 10)
	box.boundary = true
	h.SetRoot(box)
	h.Pump()
	first := box.Layer()
	if first == nil {
		t.Fatal("a repaint boundary has no layer")
	}
	box.MarkNeedsPaint()
	h.Pump()
	if box.Layer() != first {
		t.Error("the layer of a repaint boundary was replaced between frames")
	}
}

// TestRendererBindingWrapDrawFrame checks that the widgets binding can put its
// work before and after the frame of the renderer binding.
func TestRendererBindingWrapDrawFrame(t *testing.T) {
	h := newHarness(t)
	h.SetRoot(newTBox(10, 10))
	var order []string
	h.Binding.WrapDrawFrame(func(super func() error) error {
		order = append(order, "before")
		err := super()
		order = append(order, "after")
		return err
	})
	h.Pump()
	if strings.Join(order, ",") != "before,after" {
		t.Errorf("order = %v, want before and after the frame", order)
	}
	if len(h.Scenes()) != 1 {
		t.Errorf("%d scenes, want the wrapped frame to draw one", len(h.Scenes()))
	}
}
