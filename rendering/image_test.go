package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func testImage(w, h int) *ui.Image { return ui.NewImageFromRGBA(make([]byte, 4*w*h), w, h) }

func TestRenderImageSizing(t *testing.T) {
	square, wide, tall := testImage(10, 10), testImage(20, 10), testImage(10, 20)
	bc := func(minW, minH, maxW, maxH float64) rendering.BoxConstraints {
		return rendering.BoxConstraints{MinWidth: minW, MinHeight: minH, MaxWidth: maxW, MaxHeight: maxH}
	}
	tests := []struct {
		name        string
		opts        rendering.RenderImageOpts
		constraints rendering.BoxConstraints
		want        ui.Size
	}{
		{"square fills the minimum", rendering.RenderImageOpts{Image: square}, bc(25, 25, 100, 100), ui.Size{Width: 25, Height: 25}},
		{"wide with a minimum height", rendering.RenderImageOpts{Image: wide}, bc(5, 30, 100, 100), ui.Size{Width: 60, Height: 30}},
		{"tall with a minimum width", rendering.RenderImageOpts{Image: tall}, bc(50, 5, 75, 75), ui.Size{Width: 50, Height: 75}},
		{"wide at its own size", rendering.RenderImageOpts{Image: wide}, bc(5, 5, 100, 100), ui.Size{Width: 20, Height: 10}},
		{"wide shrunk", rendering.RenderImageOpts{Image: wide}, bc(5, 5, 16, 16), ui.Size{Width: 16, Height: 8}},
		{"tall shrunk", rendering.RenderImageOpts{Image: tall}, bc(5, 5, 16, 16), ui.Size{Width: 8, Height: 16}},
		{"square grown to the maximum", rendering.RenderImageOpts{Image: square}, bc(4, 4, 8, 8), ui.Size{Width: 8, Height: 8}},
		{"wide in a square", rendering.RenderImageOpts{Image: wide}, bc(20, 20, 30, 30), ui.Size{Width: 30, Height: 20}},
		{"tall in a square", rendering.RenderImageOpts{Image: tall}, bc(20, 20, 30, 30), ui.Size{Width: 20, Height: 30}},
		{"no image", rendering.RenderImageOpts{}, bc(25, 25, 100, 100), ui.Size{Width: 25, Height: 25}},
		{"no image, width", rendering.RenderImageOpts{Width: foundation.Some(50.0)}, bc(25, 25, 100, 100), ui.Size{Width: 50, Height: 25}},
		{"no image, height", rendering.RenderImageOpts{Height: foundation.Some(50.0)}, bc(25, 25, 100, 100), ui.Size{Width: 25, Height: 50}},
		{"no image, both, constrained", rendering.RenderImageOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}, bc(25, 25, 75, 75), ui.Size{Width: 75, Height: 75}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			image := rendering.NewRenderImage(tc.opts)
			layout(t, newHarness(t), image, withConstraints(tc.constraints))
			if image.Size() != tc.want {
				t.Errorf("size = %v, want %v", image.Size(), tc.want)
			}
		})
	}
}

func TestRenderImageToStringDeep(t *testing.T) {
	skipUnlessDebug(t)
	img := testImage(10, 10)
	image := rendering.NewRenderImage(rendering.RenderImageOpts{Image: img})
	layout(t, newHarness(t), image, atPhase(withConstraints(rendering.BoxConstraints{MinWidth: 25, MinHeight: 25, MaxWidth: 100, MaxHeight: 100}), phaseLayout))
	got := ignoreHashCodes(image.ToStringDeep(foundation.ToStringDeepOpts{MinLevel: foundation.DiagnosticLevelInfo}))
	want := "RenderImage#00000 relayoutBoundary=up2 NEEDS-PAINT\n" +
		"   parentData: <none> (can use size)\n" +
		"   constraints: BoxConstraints(25.0<=w<=100.0, 25.0<=h<=100.0)\n" +
		"   size: Size(25.0, 25.0)\n" +
		"   image: " + img.String() + "\n" +
		"   alignment: Alignment.center\n" +
		"   invertColors: false\n" +
		"   filterQuality: medium\n"
	if got != want {
		t.Errorf("toStringDeep:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderImageColorBlendMode(t *testing.T) {
	image := rendering.NewRenderImage()
	if image.ColorBlendMode() != ui.BlendModeUnspecified {
		t.Errorf("default blend mode = %v", image.ColorBlendMode())
	}
	image.SetColorBlendMode(ui.BlendModeColor)
	if image.ColorBlendMode() != ui.BlendModeColor {
		t.Errorf("blend mode = %v", image.ColorBlendMode())
	}
}

func TestRenderImageOwnsItsImage(t *testing.T) {
	t.Run("a new image disposes the old one", func(t *testing.T) {
		original := testImage(10, 10)
		first, second := original.Clone(), testImage(20, 10)
		r := rendering.NewRenderImage(rendering.RenderImageOpts{Image: first})
		r.SetImage(second)
		if !first.IsDisposed() || second.IsDisposed() || original.IsDisposed() {
			t.Errorf("disposed: first %v, second %v, original %v", first.IsDisposed(), second.IsDisposed(), original.IsDisposed())
		}
		r.SetImage(nil)
		if !second.IsDisposed() || r.Image() != nil {
			t.Errorf("second disposed %v, image %v", second.IsDisposed(), r.Image())
		}
	})
	t.Run("a clone of the image is disposed instead", func(t *testing.T) {
		original := testImage(10, 10)
		first, second := original.Clone(), original.Clone()
		r := rendering.NewRenderImage(rendering.RenderImageOpts{Image: first})
		r.SetImage(second)
		if first.IsDisposed() || !second.IsDisposed() || r.Image() != first {
			t.Errorf("disposed: first %v, second %v", first.IsDisposed(), second.IsDisposed())
		}
	})
	t.Run("the same image is kept", func(t *testing.T) {
		clone := testImage(10, 10).Clone()
		r := rendering.NewRenderImage(rendering.RenderImageOpts{Image: clone})
		r.SetImage(r.Image())
		if clone.IsDisposed() {
			t.Error("setting the same image disposed it")
		}
		r.SetImage(nil)
		if !clone.IsDisposed() {
			t.Error("clearing the image did not dispose it")
		}
	})
	t.Run("dispose releases the image", func(t *testing.T) {
		clone := testImage(10, 10).Clone()
		r := rendering.NewRenderImage(rendering.RenderImageOpts{Image: clone})
		r.Dispose()
		if !clone.IsDisposed() || r.Image() != nil {
			t.Errorf("disposed %v, image %v", clone.IsDisposed(), r.Image())
		}
	})
}

func TestRenderImageOpacityListener(t *testing.T) {
	skipUnlessDebug(t)
	opacity := foundation.NewValueNotifier(0.5)
	image := rendering.NewRenderImage(rendering.RenderImageOpts{Image: testImage(4, 4), Opacity: opacity})
	h := newHarness(t)
	layout(t, h, image, atPhase(layoutOpts{}, phasePaint))
	if image.AsRenderObject().DebugNeedsPaint() {
		t.Fatal("the image needs paint after the frame")
	}
	opacity.SetValue(0.25)
	if !image.AsRenderObject().DebugNeedsPaint() {
		t.Error("a new opacity must need paint")
	}
	pumpFrame(h, phasePaint)
	h.SetRoot(nil)
	opacity.SetValue(1)
	if image.AsRenderObject().DebugNeedsPaint() {
		t.Error("a detached image listens to the opacity")
	}
}
