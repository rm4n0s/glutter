package rendering_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func contains2(list []rendering.RenderObject, o rendering.RenderObject) bool {
	for _, x := range list {
		if x == o {
			return true
		}
	}
	return false
}

func TestPipelineOwnerNodesNeedingLayout(t *testing.T) {
	owner := rendering.NewPipelineOwner()
	obj := newTObject(false)
	obj.repaintBoundary = true
	obj.Attach(owner)
	if len(owner.NodesNeedingLayout()) != 0 {
		t.Fatal("a new owner has nodes needing layout")
	}
	obj.Layout(rendering.NewBoxConstraintsTightForFinite())
	obj.MarkNeedsLayout()
	if !contains2(owner.NodesNeedingLayout(), obj) {
		t.Error("the relayout boundary was not added to the nodes needing layout")
	}
	owner.FlushLayout()
	if len(owner.NodesNeedingLayout()) != 0 {
		t.Error("FlushLayout left nodes needing layout")
	}
}

func TestPipelineOwnerNodesNeedingPaint(t *testing.T) {
	owner := rendering.NewPipelineOwner()
	obj := newTObject(true)
	obj.repaintBoundary = true
	layer := rendering.NewOffsetLayer()
	layer.Attach(owner)
	obj.Attach(owner)
	if len(owner.NodesNeedingPaint()) != 0 {
		t.Fatal("a new owner has nodes needing paint")
	}
	obj.MarkNeedsPaint()
	obj.ScheduleInitialPaint(layer)
	if !contains2(owner.NodesNeedingPaint(), obj) {
		t.Error("the repaint boundary was not added to the nodes needing paint")
	}
	owner.FlushPaint()
	if len(owner.NodesNeedingPaint()) != 0 {
		t.Error("FlushPaint left nodes needing paint")
	}
}

func TestPipelineOwnerVisualUpdate(t *testing.T) {
	calls := 0
	owner := rendering.NewPipelineOwner(rendering.PipelineOwnerOpts{OnNeedVisualUpdate: func() { calls++ }})
	obj := newTObject(true)
	obj.repaintBoundary = true
	obj.Attach(owner)
	obj.Layout(rendering.NewBoxConstraintsTightForFinite())
	obj.MarkNeedsLayout()
	if calls != 1 {
		t.Errorf("OnNeedVisualUpdate called %d times after MarkNeedsLayout, want 1", calls)
	}
}

func TestRenderObjectGetTransformTo(t *testing.T) {
	t.Run("siblings and descendants", func(t *testing.T) {
		owner := rendering.NewPipelineOwner()
		o1 := newTObject(false)
		o1.Attach(owner)
		o11, o12 := newTObject(false), newTObject(false)
		o1.Add(o11)
		o1.Add(o12)
		pairs := [][2]*tObject{{o11, o12}, {o1, o11}, {o1, o12}, {o11, o1}, {o12, o1}, {o1, o1}, {o11, o11}, {o12, o12}}
		for i, p := range pairs {
			if got := p[0].GetTransformTo(p[1]); !got.IsIdentity() {
				t.Errorf("pair %d: %v, want identity", i, got)
			}
		}
	})
	t.Run("paint transform", func(t *testing.T) {
		owner := rendering.NewPipelineOwner()
		o0 := newTObject(false)
		o0.Attach(owner)
		o1, o2 := newTObject(false), newTObject(false)
		o0.Add(o1)
		o0.Add(o2)
		o0.paintTransform = ui.NewMatrix4Diagonal3Values(9, 4, 1)
		o11, o21 := newTObject(false), newTObject(false)
		o1.Add(o11)
		o1.paintTransform = ui.NewMatrix4TranslationValues(8, 16, 32)
		o2.Add(o21)
		o2.paintTransform = ui.NewMatrix4TranslationValues(32, 64, 128)
		want := ui.NewMatrix4TranslationValues((8-32)*9, (16-64)*4, 32-128)
		if got := o11.GetTransformTo(o21); !got.Equals(want) {
			t.Errorf("got %v, want %v", got, want)
		}
		singular := ui.NewMatrix4FromList([16]float32{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4})
		o0.paintTransform = singular
		if got := o11.GetTransformTo(o21); !got.Equals(ui.NewMatrix4Zero()) {
			t.Errorf("got %v for a singular transform, want zero", got)
		}
	})
}

func TestRenderObjectDispose(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("debugDisposed", func(t *testing.T) {
		o := newTObject(false)
		if o.DebugDisposed() {
			t.Fatal("disposed before dispose")
		}
		o.Dispose()
		if !o.DebugDisposed() || !strings.Contains(o.ToStringShort(), "DISPOSED") {
			t.Errorf("disposed = %v, short = %q", o.DebugDisposed(), o.ToStringShort())
		}
	})
	t.Run("nulls the layer on a repaint boundary", func(t *testing.T) {
		o := newTObject(true)
		o.repaintBoundary = true
		rendering.RepaintCompositedChild(o, true)
		if _, ok := o.DebugLayer().(rendering.OffsetLayerLike); !ok {
			t.Fatalf("layer = %T, want an OffsetLayer", o.DebugLayer())
		}
		o.Dispose()
		if o.DebugLayer() != nil {
			t.Error("the layer survived Dispose")
		}
	})
	t.Run("nulls the layer when no longer a repaint boundary", func(t *testing.T) {
		o := newTObject(true)
		o.repaintBoundary = true
		rendering.RepaintCompositedChild(o, true)
		o.repaintBoundary = false
		o.Dispose()
		if o.DebugLayer() != nil {
			t.Error("the layer survived Dispose")
		}
	})
}

func TestRenderObjectChangeBoundaryAfterMarks(t *testing.T) {
	h := newHarness(t)
	o := newTObject(true)
	o.repaintBoundary = true
	o.Attach(h.Owner)
	o.Layout(rendering.NewBoxConstraintsTightForFinite())
	rendering.RepaintCompositedChild(o, true)
	o.MarkNeedsCompositedLayerUpdate()
	o.MarkNeedsPaint()
	o.repaintBoundary = false
	o.MarkNeedsCompositingBitsUpdate()
	h.FlushPipeline()
}

func TestPaintingContextRecorder(t *testing.T) {
	ctx := rendering.NewPaintingContext(rendering.NewContainerLayer(), ui.Rect{})
	if !ctx.Recorder().IsRecording() {
		t.Error("the recorder is not recording")
	}
}

func TestPaintingContextCompositionCallback(t *testing.T) {
	root := rendering.NewContainerLayer()
	ctx := rendering.NewPaintingContext(root, ui.Rect{})
	called := false
	ctx.AddCompositionCallback(func(l rendering.Layer) {
		if l != rendering.Layer(root) {
			t.Errorf("layer = %v, want the root", l)
		}
		called = true
	})
	if called {
		t.Fatal("called early")
	}
	root.BuildScene(ui.NewSceneBuilder()).Dispose()
	if !called {
		t.Error("the composition callback was not called")
	}
}

// layerReuse is _testPaintingContextLayerReuse: paint twice and check that the
// second paint was given the first paint's layer, and that the type is right.
func layerReuse[L rendering.ContainerLayerLike](t *testing.T, push func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, offset ui.Offset, old L) rendering.ContainerLayerLike) {
	t.Helper()
	h := newHarness(t)
	box := newTLayerBox(func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, offset ui.Offset, old rendering.ContainerLayerLike) rendering.ContainerLayerLike {
		o, _ := old.(L)
		return push(p, ctx, offset, o)
	})
	layoutBox(t, h, box)
	box.MarkNeedsPaint()
	h.Pump()
	if len(box.painted) != 2 {
		t.Fatalf("painted %d layers, want 2", len(box.painted))
	}
	if _, ok := box.painted[0].(L); !ok {
		t.Errorf("layer type = %T", box.painted[0])
	}
	if box.painted[0] != box.painted[1] {
		t.Error("the second paint did not reuse the layer")
	}
}

func TestPaintingContextReusesLayers(t *testing.T) {
	noop := func(*rendering.PaintingContext, ui.Offset) {}
	_ = noop
	t.Run("pushClipRect", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.ClipRectLayer) rendering.ContainerLayerLike {
			return ctx.PushClipRect(true, o, ui.Rect{}, p, rendering.PushClipRectOpts{OldLayer: old})
		})
	})
	t.Run("pushClipRRect", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.ClipRRectLayer) rendering.ContainerLayerLike {
			return ctx.PushClipRRect(true, o, ui.Rect{}, ui.NewRRectFromRectAndRadius(ui.Rect{}, ui.NewRadiusCircular(1)), p, rendering.PushClipRRectOpts{OldLayer: old})
		})
	})
	t.Run("pushClipRSuperellipse", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.ClipRSuperellipseLayer) rendering.ContainerLayerLike {
			return ctx.PushClipRSuperellipse(true, o, ui.Rect{}, ui.NewRSuperellipseFromRectAndRadius(ui.Rect{}, ui.NewRadiusCircular(1)), p, rendering.PushClipRSuperellipseOpts{OldLayer: old})
		})
	})
	t.Run("pushClipPath", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.ClipPathLayer) rendering.ContainerLayerLike {
			return ctx.PushClipPath(true, o, ui.Rect{}, ui.NewPath(), p, rendering.PushClipPathOpts{OldLayer: old})
		})
	})
	t.Run("pushColorFilter", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.ColorFilterLayer) rendering.ContainerLayerLike {
			return ctx.PushColorFilter(o, ui.NewColorFilterMode(ui.NewColorFromARGB(255, 0, 0, 0), ui.BlendModeClear), p, old)
		})
	})
	t.Run("pushTransform", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.TransformLayer) rendering.ContainerLayerLike {
			return ctx.PushTransform(true, o, ui.NewMatrix4Identity(), p, old)
		})
	})
	t.Run("pushOpacity", func(t *testing.T) {
		layerReuse(t, func(p rendering.PaintingContextCallback, ctx *rendering.PaintingContext, o ui.Offset, old *rendering.OpacityLayer) rendering.ContainerLayerLike {
			return ctx.PushOpacity(o, 100, p, old)
		})
	})
}

func TestPaintingContextPushWithoutCompositing(t *testing.T) {
	rec := func() (*rendering.PaintingContext, *rendering.ContainerLayer) {
		root := rendering.NewContainerLayer()
		return rendering.NewPaintingContext(root, ui.NewRectFromLTWH(0, 0, 100, 100)), root
	}
	painted := 0
	painter := func(ctx *rendering.PaintingContext, o ui.Offset) {
		painted++
		ctx.Canvas().DrawRect(ui.NewRectFromLTWH(0, 0, 1, 1), ui.NewPaint())
	}
	tests := []struct {
		name string
		push func(ctx *rendering.PaintingContext) any
	}{
		{"clipRect", func(ctx *rendering.PaintingContext) any {
			return ctx.PushClipRect(false, ui.OffsetZero, ui.NewRectFromLTWH(0, 0, 10, 10), painter)
		}},
		{"clipRRect", func(ctx *rendering.PaintingContext) any {
			return ctx.PushClipRRect(false, ui.OffsetZero, ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRRectFromRectAndRadius(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRadiusCircular(2)), painter)
		}},
		{"clipPath", func(ctx *rendering.PaintingContext) any {
			return ctx.PushClipPath(false, ui.OffsetZero, ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewPath(), painter)
		}},
		{"transform", func(ctx *rendering.PaintingContext) any {
			return ctx.PushTransform(false, ui.OffsetZero, ui.NewMatrix4Identity(), painter, nil)
		}},
		{"clip none", func(ctx *rendering.PaintingContext) any {
			return ctx.PushClipRect(true, ui.OffsetZero, ui.NewRectFromLTWH(0, 0, 10, 10), painter, rendering.PushClipRectOpts{ClipBehavior: ui.ClipNone})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, root := rec()
			before := painted
			res := tc.push(ctx)
			ctx.StopRecordingIfNeeded()
			if painted != before+1 {
				t.Errorf("painter ran %d times", painted-before)
			}
			switch v := res.(type) {
			case *rendering.ClipRectLayer:
				if v != nil {
					t.Error("a layer was made without compositing")
				}
			case *rendering.ClipRRectLayer:
				if v != nil {
					t.Error("a layer was made without compositing")
				}
			case *rendering.ClipPathLayer:
				if v != nil {
					t.Error("a layer was made without compositing")
				}
			case *rendering.TransformLayer:
				if v != nil {
					t.Error("a layer was made without compositing")
				}
			}
			if _, ok := root.FirstChild().(*rendering.PictureLayer); !ok || root.FirstChild() != root.LastChild() {
				t.Errorf("the painting went into %v, want one picture layer", root.FirstChild())
			}
		})
	}
}

// layerRoot is a render object that paints a leader layer, LeaderLayerRenderObject.
type leaderObject struct {
	rendering.RenderObjectBase
	link *rendering.LayerLink
}

func newLeaderObject(link *rendering.LayerLink) *leaderObject {
	o := &leaderObject{link: link}
	rendering.InitRenderObject(&o.RenderObjectBase, o)
	return o
}

func (o *leaderObject) IsRepaintBoundary() bool { return true }
func (o *leaderObject) PaintBounds() ui.Rect    { return ui.Rect{} }
func (o *leaderObject) SemanticBounds() ui.Rect { return ui.Rect{} }
func (o *leaderObject) PerformLayout()          {}
func (o *leaderObject) PerformResize()          {}
func (o *leaderObject) Paint(ctx *rendering.PaintingContext, offset ui.Offset) {
	if o.link != nil {
		ctx.PushLayer(rendering.NewLeaderLayer(o.link), func(*rendering.PaintingContext, ui.Offset) {}, offset, foundation.None[ui.Rect]())
	}
}

func TestLeaderLayerSwitchesRenderObjectWithinFrame(t *testing.T) {
	h := newHarness(t)
	link := rendering.NewLayerLink()
	o1 := newLeaderObject(link)
	o1.Attach(h.Owner)
	root1 := rendering.NewOffsetLayer()
	root1.Attach(o1)
	o1.ScheduleInitialPaint(root1)
	o1.Layout(rendering.NewBoxConstraintsTightForFinite())

	o2 := newLeaderObject(nil)
	root2 := rendering.NewOffsetLayer()
	root2.Attach(o2)
	o2.Attach(h.Owner)
	o2.ScheduleInitialPaint(root2)
	o2.Layout(rendering.NewBoxConstraintsTightForFinite())
	h.FlushPipeline()

	o1.link, o2.link = nil, link
	o1.MarkNeedsPaint()
	o2.MarkNeedsPaint()
	h.FlushPipeline()
	o1.link, o2.link = link, nil
	o1.MarkNeedsPaint()
	o2.MarkNeedsPaint()
	h.FlushPipeline()
	if link.Leader() == nil {
		t.Error("the link has no leader")
	}
}
