//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestDecorationDebugPanics(t *testing.T) {
	size := ui.Size{Width: 100, Height: 100}
	img := &fakeImage{}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"blend mode without background", "BoxDecorationBlendModeWithoutBackground", func() {
			NewBoxDecoration(BoxDecorationOpts{BackgroundBlendMode: ui.BlendModeColor})
		}},
		{"circle with radius", "BoxDecorationCircleBorderRadius", func() {
			NewBoxDecoration(BoxDecorationOpts{Shape: BoxShapeCircle, BorderRadius: NewBorderRadiusCircular(5)}).DebugAssertIsValid()
		}},
		{"circle with radius when painting", "BoxDecorationCircleBorderRadius", func() {
			d := NewBoxDecoration(BoxDecorationOpts{Shape: BoxShapeCircle, BorderRadius: NewBorderRadiusCircular(5), Color: foundation.Some(colorR)})
			paintDecoration(t, d, ui.Offset{}, cfgOf(100, 100, ui.TextDirectionLTR))
		}},
		{"hit test outside", "DecorationHitTestOutside", func() { NewBoxDecoration().HitTest(size, ui.Offset{Dx: 200, Dy: 0}, ui.TextDirectionLTR) }},
		{"box image without onChanged", "BoxDecorationOnChangedRequired", func() { NewBoxDecoration(BoxDecorationOpts{Image: img}).CreateBoxPainter(nil) }},
		{"box painter without size", "BoxDecorationNoSize", func() {
			NewBoxDecoration().CreateBoxPainter(nil).Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.Offset{}, ImageConfiguration{})
		}},
		{"clip path of a default decoration", "DecorationNoClipPath", func() { DecorationDefaults{}.GetClipPath(ui.Rect{}, ui.TextDirectionLTR) }},
		{"shape color and gradient", "ShapeDecorationColorAndGradient", func() {
			NewShapeDecoration(Border{}, ShapeDecorationOpts{Color: foundation.Some(colorR), Gradient: NewLinearGradient([]ui.Color{colorR, colorG})})
		}},
		{"shape from non-uniform border", "ShapeDecorationBorderNotUniform", func() {
			NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{Border: Border{Top: widthSide(1), Left: widthSide(2)}, BorderRadius: NewBorderRadiusCircular(5)}))
		}},
		{"shape image without onChanged", "ShapeDecorationOnChangedRequired", func() { NewShapeDecoration(Border{}, ShapeDecorationOpts{Image: img}).CreateBoxPainter(nil) }},
		{"shape painter without size", "ShapeDecorationNoSize", func() {
			NewShapeDecoration(Border{}).CreateBoxPainter(nil).Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.Offset{}, ImageConfiguration{})
		}},
		{"logo invalid", "FlutterLogoInvalid", func() {
			NewFlutterLogoDecoration().DebugAssertIsValid()
			FlutterLogoDecorationLerp(nil, &FlutterLogoDecoration{opacity: 2}, 0.5)
		}},
		{"logo painter without size", "FlutterLogoNoSize", func() {
			NewFlutterLogoDecoration().CreateBoxPainter(nil).Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.Offset{}, ImageConfiguration{})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, tt.tag, tt.call)
		})
	}
}
