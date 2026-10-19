package rendering_test

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

var (
	orange     = ui.NewColorFromARGB(0xFF, 0xFF, 0x90, 0x00)
	green      = ui.NewColorFromARGB(0xFF, 0x0F, 0xF9, 0x00)
	orangeRect = ui.NewRectFromLTWH(10, 10, 50, 75)
	greenRect  = ui.NewRectFromLTWH(20, 20, 100, 150)
)

// biggestBox is TestRenderObject of view_test.dart: as big as it can be, with an
// orange rectangle.
type biggestBox struct{ rendering.RenderBox }

func newBiggestBox() *biggestBox {
	b := &biggestBox{}
	rendering.InitRenderBox(&b.RenderBox, b)
	return b
}

func (b *biggestBox) PerformLayout() { b.SetSize(b.BoxConstraints().Biggest()) }
func (b *biggestBox) ComputeDryLayout(c rendering.BoxConstraints) ui.Size {
	return c.Biggest()
}
func (b *biggestBox) Paint(ctx *rendering.PaintingContext, _ ui.Offset) {
	p := ui.NewPaint()
	p.SetColor(orange)
	ctx.Canvas().DrawRect(orangeRect, p)
}

func pixel(t *testing.T, h *renderingtest.Harness, x, y int) color.RGBA {
	t.Helper()
	img, err := h.Image()
	if err != nil {
		t.Fatal(err)
	}
	return img.RGBAAt(x, y)
}

func viewConfig(w, h, dpr float64) rendering.ViewConfiguration {
	c := rendering.NewBoxConstraintsTight(ui.Size{Width: w, Height: h})
	return rendering.NewViewConfiguration(rendering.ViewConfigurationOpts{
		LogicalConstraints:  foundation.Some(c),
		PhysicalConstraints: foundation.Some(c.Mul(dpr)),
		DevicePixelRatio:    foundation.Some(dpr),
	})
}

func TestRenderViewPaintBoundsAccountForPixelRatio(t *testing.T) {
	h := renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 2})
	h.SetRoot(newBiggestBox())
	h.Pump()
	if got := h.RenderView.Size(); got != (ui.Size{Width: 400, Height: 300}) {
		t.Errorf("Size() = %v, want 400x300", got)
	}
	if got, want := h.RenderView.PaintBounds(), ui.NewRectFromLTWH(0, 0, 800, 600); got != want {
		t.Errorf("PaintBounds() = %v, want %v", got, want)
	}
	if got, want := h.RenderView.SemanticBounds(), ui.NewRectFromLTWH(0, 0, 800, 600); got != want {
		t.Errorf("SemanticBounds() = %v, want %v", got, want)
	}
}

func TestRenderViewRootLayerReplacement(t *testing.T) {
	skipUnlessDebug(t)
	newView := func(cfg rendering.ViewConfiguration) *rendering.RenderView {
		h := newHarness(t)
		v := rendering.NewRenderView(h.View, rendering.RenderViewOpts{Configuration: foundation.Some(cfg)})
		v.Attach(rendering.NewPipelineOwner())
		v.PrepareInitialFrame()
		return v
	}
	t.Run("same configuration", func(t *testing.T) {
		v := newView(viewConfig(20, 20, 2))
		first := v.DebugLayer()
		v.SetConfiguration(viewConfig(20, 20, 2))
		if v.DebugLayer() != first {
			t.Error("the root layer was replaced for an equal configuration")
		}
		v.SetConfiguration(viewConfig(20, 20, 5))
		if v.DebugLayer() == first {
			t.Error("the root layer was not replaced for another pixel ratio")
		}
	})
	t.Run("resize", func(t *testing.T) {
		v := newView(viewConfig(100, 100, 2))
		first := v.DebugLayer()
		v.SetConfiguration(viewConfig(100, 1117, 2))
		if v.DebugLayer() != first {
			t.Error("the root layer was replaced when the view resized")
		}
	})
}

func TestViewConfigurationEquality(t *testing.T) {
	a, b, c := viewConfig(20, 20, 2), viewConfig(20, 20, 2), viewConfig(20, 20, 3)
	if a != b {
		t.Error("equal configurations differ")
	}
	if a == c {
		t.Error("different configurations are equal")
	}
	if got, want := a.String(), "BoxConstraints(w=20.0, h=20.0) at 2.0x"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got := a.ToMatrix(); got.Entry(0, 0) != 2 || got.Entry(1, 1) != 2 || got.Entry(2, 2) != 1 {
		t.Errorf("ToMatrix() = %v", got)
	}
	if !a.ShouldUpdateMatrix(c) || a.ShouldUpdateMatrix(b) {
		t.Error("ShouldUpdateMatrix is wrong")
	}
	if got := a.ToPhysicalSize(ui.Size{Width: 5, Height: 6}); got != (ui.Size{Width: 40, Height: 40}) {
		t.Errorf("ToPhysicalSize() = %v", got)
	}
}

func TestRenderViewDebugPaintCallback(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	h.SetRoot(newBiggestBox())
	h.Pump()
	if got := pixel(t, h, 15, 15); got != (color.RGBA{0xFF, 0x90, 0, 0xFF}) {
		t.Fatalf("orange rect pixel = %v", got)
	}
	if got := pixel(t, h, 80, 100); got != (color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}) && got.A != 0 && got != (color.RGBA{}) {
		t.Fatalf("background pixel = %v", got)
	}
	before := pixel(t, h, 80, 100)
	h.Binding.DebugOptions().AddPaintCallback(func(ctx *rendering.PaintingContext, _ ui.Offset, _ *rendering.RenderView) {
		p := ui.NewPaint()
		p.SetColor(green)
		ctx.Canvas().DrawRect(greenRect, p)
	})
	h.RenderView.MarkNeedsPaint()
	h.Pump()
	if got := pixel(t, h, 80, 100); got != (color.RGBA{0x0F, 0xF9, 0, 0xFF}) {
		t.Errorf("green rect pixel = %v, want green (was %v)", got, before)
	}
	if got := pixel(t, h, 25, 25); got != (color.RGBA{0x0F, 0xF9, 0, 0xFF}) {
		t.Errorf("green over orange pixel = %v", got)
	}
	h.Binding.DebugOptions().RemovePaintCallback(0)
	h.RenderView.MarkNeedsPaint()
	h.Pump()
	if got := pixel(t, h, 80, 100); got != before {
		t.Errorf("after removing the callback pixel = %v, want %v", got, before)
	}
}

func TestRenderViewConfigurationChangesBeforePrepare(t *testing.T) {
	h := newHarness(t)
	v := rendering.NewRenderView(h.View)
	v.SetConfiguration(rendering.NewViewConfiguration(rendering.ViewConfigurationOpts{
		LogicalConstraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 100, Height: 200})),
		DevicePixelRatio:   foundation.Some(3.0),
	}))
	v.SetConfiguration(rendering.NewViewConfiguration(rendering.ViewConfigurationOpts{
		LogicalConstraints: foundation.Some(rendering.NewBoxConstraintsTight(ui.Size{Width: 200, Height: 300})),
		DevicePixelRatio:   foundation.Some(2.0),
	}))
	owner := rendering.NewPipelineOwner()
	owner.SetRootNode(v)
	v.PrepareInitialFrame()
}

func TestRenderViewConstraintsFromConfiguration(t *testing.T) {
	h := newHarness(t)
	c := rendering.BoxConstraints{MinWidth: 1, MaxWidth: 2, MinHeight: 3, MaxHeight: 4}
	cfg := rendering.NewViewConfiguration(rendering.ViewConfigurationOpts{
		LogicalConstraints: foundation.Some(c), PhysicalConstraints: foundation.Some(c.Mul(3)), DevicePixelRatio: foundation.Some(3.0),
	})
	v := rendering.NewRenderView(h.View)
	if v.HasConfiguration() {
		t.Error("a new view has a configuration")
	}
	if err, _ := panicOf(func() { v.BoxConstraints() }).(error); err == nil {
		t.Error("constraints without a configuration did not panic")
	} else {
		requireRoute(t, err, "RenderViewNoConfiguration")
	}
	v.SetConfiguration(cfg)
	if got := v.BoxConstraints(); got != c {
		t.Errorf("constraints = %v, want %v", got, c)
	}
	v2 := rendering.NewRenderView(h.View, rendering.RenderViewOpts{Configuration: foundation.Some(cfg)})
	if got := v2.Constraints(); got != rendering.Constraints(c) {
		t.Errorf("constraints = %v, want %v", got, c)
	}
}

func TestRenderViewFromViewConfiguration(t *testing.T) {
	h := renderingtest.New(t, renderingtest.Opts{Size: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 2})
	cfg := rendering.NewViewConfigurationFromView(h.View)
	want := rendering.NewBoxConstraintsTight(ui.Size{Width: 400, Height: 300})
	if cfg.LogicalConstraints != want || cfg.DevicePixelRatio != 2 ||
		cfg.PhysicalConstraints != rendering.NewBoxConstraintsTight(ui.Size{Width: 800, Height: 600}) {
		t.Errorf("configuration = %v / %v / %v", cfg.LogicalConstraints, cfg.PhysicalConstraints, cfg.DevicePixelRatio)
	}
}
