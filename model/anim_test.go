package model_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

func TestTextureTransformApply(t *testing.T) {
	tests := []struct {
		name         string
		tr           model.TextureTransform
		u, v         float32
		wantU, wantV float32
	}{
		{"identity", model.TextureTransform{Scale: [2]float32{1, 1}}, 0.25, 0.75, 0.25, 0.75},
		{"offset", model.TextureTransform{Scale: [2]float32{1, 1}, Offset: [2]float32{0.5, -0.25}}, 0.25, 0.75, 0.75, 0.5},
		{"scale", model.TextureTransform{Scale: [2]float32{2, 3}}, 0.5, 0.5, 1, 1.5},
		{"a quarter turn", model.TextureTransform{Scale: [2]float32{1, 1}, Rotation: math.Pi / 2}, 1, 0, 0, -1},
		{"scale, then rotation, then offset", model.TextureTransform{Scale: [2]float32{2, 1}, Rotation: math.Pi / 2, Offset: [2]float32{1, 1}}, 1, 0, 1, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, v := tt.tr.Apply(tt.u, tt.v)
			if math.Abs(float64(u-tt.wantU)) > 1e-6 || math.Abs(float64(v-tt.wantV)) > 1e-6 {
				t.Errorf("(%v, %v), want (%v, %v)", u, v, tt.wantU, tt.wantV)
			}
		})
	}
}

func TestTRS(t *testing.T) {
	id := model.IdentityTRS()
	if m := id.Matrix(); m != vecmath.Mat4Identity() {
		t.Errorf("identity matrix %v", m)
	}
	trs := model.TRS{
		Translation: vecmath.Vec3{X: 1, Y: 2, Z: 3},
		Rotation:    vecmath.NewQuatAxisAngle(vecmath.Vec3{Z: 1}, math.Pi/2),
		Scale:       vecmath.Vec3{X: 2, Y: 2, Z: 2},
	}
	p := trs.Matrix().TransformPoint(vecmath.Vec3{X: 1})
	if math.Abs(p.X-1) > 1e-12 || math.Abs(p.Y-4) > 1e-12 || math.Abs(p.Z-3) > 1e-12 {
		t.Errorf("(1, 0, 0) goes to %v, want (1, 4, 3)", p)
	}
}

func TestAnimation(t *testing.T) {
	tests := []struct {
		name     string
		channel  model.AnimationChannel
		width    int
		duration float32
	}{
		{"translation", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{0, 1}, Values: make([]float32, 6)}, 3, 1},
		{"rotation", model.AnimationChannel{Path: model.PathRotation, Times: []float32{0, 2.5}, Values: make([]float32, 8)}, 4, 2.5},
		{"scale", model.AnimationChannel{Path: model.PathScale, Times: []float32{1}, Values: make([]float32, 3)}, 3, 1},
		{"weights", model.AnimationChannel{Path: model.PathWeights, Times: []float32{0, 1, 2}, Values: make([]float32, 6)}, 2, 2},
		{"cubic spline weights", model.AnimationChannel{Path: model.PathWeights, Interpolation: model.InterpolationCubicSpline, Times: []float32{0, 1}, Values: make([]float32, 12)}, 2, 1},
		{"weights without keyframes", model.AnimationChannel{Path: model.PathWeights}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.channel.Width(); got != tt.width {
				t.Errorf("width %d, want %d", got, tt.width)
			}
			a := model.Animation{Channels: []model.AnimationChannel{{Times: []float32{0.5}}, tt.channel}}
			if got := a.Duration(); got != max(tt.duration, 0.5) {
				t.Errorf("duration %v", got)
			}
		})
	}
	if (&model.Animation{}).Duration() != 0 {
		t.Errorf("an animation with no channel lasts 0")
	}
}

func TestNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{model.PathTranslation.String(), "translation"}, {model.PathRotation.String(), "rotation"}, {model.PathScale.String(), "scale"},
		{model.PathWeights.String(), "weights"}, {model.AnimationPath(9).String(), "unknown"},
		{model.InterpolationLinear.String(), "LINEAR"}, {model.InterpolationStep.String(), "STEP"},
		{model.InterpolationCubicSpline.String(), "CUBICSPLINE"}, {model.Interpolation(9).String(), "unknown"},
		{model.LightDirectional.String(), "directional"}, {model.LightPoint.String(), "point"}, {model.LightSpot.String(), "spot"},
		{model.LightType(9).String(), "unknown"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("name %q, want %q", tt.got, tt.want)
		}
	}
}

func TestPBRMaterialColor(t *testing.T) {
	if got := (&model.Material{}).Color(); got != model.DefaultColor {
		t.Errorf("a material with no color is %v", got)
	}
	black := &model.Material{PBR: true}
	if got := black.Color(); got != ([4]float32{}) {
		t.Errorf("a PBR material keeps its color, even when it is all zero: %v", got)
	}
	if got := (*model.Material)(nil).Color(); got != model.DefaultColor {
		t.Errorf("no material is %v", got)
	}
}
