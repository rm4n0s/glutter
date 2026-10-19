package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func cachedViewport(children ...rendering.Sliver) *rendering.RenderViewport {
	return rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
		RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(250))},
		Children:               children,
	})
}

func TestRenderSliverOpacityCompositing(t *testing.T) {
	tests := []struct {
		name    string
		opacity foundation.Opt[float64]
		want    bool
	}{
		{"does not composite if it is transparent", foundation.Some(0.0), false},
		{"does composite if it is opaque", foundation.None[float64](), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sliver := rendering.NewRenderSliverOpacity(rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(1, 1)),
				rendering.RenderSliverOpacityOpts{Opacity: tc.opacity})
			layout(t, newHarness(t), cachedViewport(sliver), atPhase(layoutOpts{}, phaseComposite))
			if got := sliver.NeedsCompositing(); got != tc.want {
				t.Errorf("needsCompositing = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRenderSliverAnimatedOpacityCompositing(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  bool
	}{
		{"does not composite if it is transparent", 0, false},
		{"does composite if it is partially opaque", 0.5, true},
		{"does composite if it is opaque", 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			controller := animation.NewAnimationController(tickerProvider{h}, animation.AnimationControllerOpts{Value: foundation.Some(tc.value)})
			defer controller.Dispose()
			sliver := rendering.NewRenderSliverAnimatedOpacity(controller, rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(1, 1)))
			layout(t, h, cachedViewport(sliver), atPhase(layoutOpts{}, phaseComposite))
			if got := sliver.NeedsCompositing(); got != tc.want {
				t.Errorf("needsCompositing = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRenderSliverReusesItsLayer(t *testing.T) {
	skipUnlessDebug(t)
	tight := rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10})
	h := newHarness(t)
	sliver := rendering.NewRenderSliverOpacity(rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(1, 1)),
		rendering.RenderSliverOpacityOpts{Opacity: foundation.Some(0.5)})
	root := cachedViewport(sliver)
	if sliver.DebugLayer() != nil {
		t.Fatal("the sliver has a layer before layout")
	}
	layout(t, h, root, atPhase(withConstraints(tight), phasePaint))
	layer := sliver.DebugLayer()
	if layer == nil {
		t.Fatal("the sliver has no layer")
	}
	// Mark for repaint otherwise pumpFrame is a noop.
	sliver.MarkNeedsPaint()
	if !sliver.DebugNeedsPaint() {
		t.Error("the sliver does not need paint")
	}
	pumpFrame(h, phasePaint)
	if sliver.DebugNeedsPaint() {
		t.Error("the sliver needs paint after the frame")
	}
	if sliver.DebugLayer() != layer {
		t.Error("the layer was not reused")
	}
}

func TestRenderSliverAnimatedOpacityReusesItsLayer(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	// must not be 0 or 1. Otherwise, it won't create a layer
	controller := animation.NewAnimationController(tickerProvider{h}, animation.AnimationControllerOpts{Value: foundation.Some(0.5)})
	defer controller.Dispose()
	sliver := rendering.NewRenderSliverAnimatedOpacity(controller, rendering.NewRenderSliverToBoxAdapter(newRenderSizedBox(1, 1)))
	root := cachedViewport(sliver)
	if sliver.DebugLayer() != nil {
		t.Fatal("the sliver has a layer before layout")
	}
	layout(t, h, root, atPhase(withConstraints(rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 10})), phasePaint))
	layer := sliver.DebugLayer()
	if layer == nil {
		t.Fatal("the sliver has no layer")
	}
	sliver.MarkNeedsPaint()
	if !sliver.DebugNeedsPaint() {
		t.Error("the sliver does not need paint")
	}
	pumpFrame(h, phasePaint)
	if sliver.DebugNeedsPaint() {
		t.Error("the sliver needs paint after the frame")
	}
	if sliver.DebugLayer() != layer {
		t.Error("the layer was not reused")
	}
}
