package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from the tests of BorderTween, BorderRadiusTween and Matrix4Tween in
// packages/flutter/test/animation/tween_test.dart, and the tweens of
// rendering/tweens.dart.

func TestBorderRadiusTweenNullable(t *testing.T) {
	radius := func(r float64) foundation.Opt[painting.BorderRadius] {
		return foundation.Some(painting.NewBorderRadiusAll(ui.NewRadiusCircular(r)))
	}
	none := foundation.None[painting.BorderRadius]()
	tests := []struct {
		name       string
		begin, end foundation.Opt[painting.BorderRadius]
		at         float64
		want       foundation.Opt[painting.BorderRadius]
	}{
		{"none to none", none, none, 0.5, none},
		{"none to some", none, radius(10), 0.5, radius(5)},
		{"some to none", radius(10), none, 0.5, radius(5)},
		{"some to some", radius(10), radius(20), 0.5, radius(15)},
	}
	for _, tt := range tests {
		got := NewBorderRadiusTween(tt.begin, tt.end).Lerp(tt.at)
		if got != tt.want {
			t.Errorf("%s: lerp = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBorderTweenNullable(t *testing.T) {
	side := func(w float64) foundation.Opt[painting.Border] {
		return foundation.Some(painting.NewBorderAll(painting.BorderSideOpts{Width: foundation.Some(w), Color: foundation.Some(ui.NewColor(0xFF000000))}))
	}
	none := foundation.None[painting.Border]()
	if got := NewBorderTween(none, none).Lerp(0.5); got.IsSome() {
		t.Errorf("none to none = %v, want none", got)
	}
	got, ok := NewBorderTween(side(2), side(4)).Lerp(0.5).Get()
	if !ok || got.Top.Width != 3 {
		t.Errorf("some to some = %v, want width 3", got)
	}
	got, ok = NewBorderTween(none, side(4)).Lerp(0.5).Get()
	if !ok || got.Top.Width != 2 {
		t.Errorf("none to some = %v, want width 2", got)
	}
}

func TestMatrix4TweenInterpolatesScaleAndRotation(t *testing.T) {
	scaled := ui.NewMatrix4Compose(ui.Vector3{}, ui.NewQuaternionIdentity(), ui.Vector3{X: 1, Y: 1, Z: 1})
	bigger := ui.NewMatrix4Compose(ui.Vector3{X: 10}, ui.NewQuaternionIdentity(), ui.Vector3{X: 3, Y: 3, Z: 3})
	m := NewMatrix4Tween(scaled, bigger).Lerp(0.5)
	translation, _, scale := m.Decompose()
	if translation.X != 5 || scale.X != 2 || scale.Y != 2 {
		t.Errorf("translation %v and scale %v, want x 5 and scale 2", translation, scale)
	}
}

func TestRenderingTweens(t *testing.T) {
	fo := rendering.NewFractionalOffsetTween(foundation.Some(painting.NewFractionalOffset(0, 0)), foundation.Some(painting.NewFractionalOffset(1, 1)))
	if got, _ := fo.Lerp(0.25).Get(); got.DX() != 0.25 {
		t.Errorf("FractionalOffsetTween at 0.25 = %v", got)
	}
	toNone := rendering.NewFractionalOffsetTween(foundation.None[painting.FractionalOffset](), foundation.Some(painting.NewFractionalOffset(1, 1)))
	if got, ok := toNone.Lerp(1).Get(); !ok || got.DX() != 1 {
		t.Errorf("none to some at 1 = %v", got)
	}
	al := rendering.NewAlignmentTween(painting.AlignmentTopLeft, painting.AlignmentBottomRight).Lerp(0.5)
	if al.X != 0 || al.Y != 0 {
		t.Errorf("AlignmentTween at 0.5 = %v, want the center", al)
	}
	geo := rendering.NewAlignmentGeometryTween(painting.AlignmentTopLeft, nil).Lerp(0.5)
	if a, ok := geo.(painting.Alignment); !ok || a.X != -0.5 || a.Y != -0.5 {
		t.Errorf("AlignmentGeometryTween at 0.5 = %v", geo)
	}
}

func TestImplicitAnimationConstructorsPanicInDebug(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		tag  string
		call func()
	}{
		{"AlignFactorNegative", func() {
			NewAnimatedAlign(painting.AlignmentCenter, 0, nil, AnimatedAlignOpts{WidthFactor: foundation.Some(-1.0)})
		}},
		{"AlignFactorNegative", func() {
			NewAnimatedFractionallySizedBox(0, nil, AnimatedFractionallySizedBoxOpts{HeightFactor: foundation.Some(-1.0)})
		}},
		{"PhysicalModelElevationNegative", func() {
			NewAnimatedPhysicalModel(ui.NewColor(0), ui.NewColor(0), 0, nil, AnimatedPhysicalModelOpts{Elevation: -1})
		}},
	}
	for _, tt := range tests {
		requireTag(t, tt.tag, tt.call)
	}
}
