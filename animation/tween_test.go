package animation

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/animation/tween_test.dart. The tests of
// BorderTween, BorderRadiusTween and Matrix4Tween come with those tweens in
// step 8.2 (they live in widgets/implicit_animations.dart).

type noLerp struct{ n int }

func TestTweenWithoutBuiltInLerpPanics(t *testing.T) {
	if tag := panicTag(t, func() { NewTween(noLerp{1}, noLerp{2}) }); tag != "TweenLerpMissing" {
		t.Errorf("panic tag = %q, want TweenLerpMissing", tag)
	}
	if tag := panicTag(t, func() { NewTween(0, 1) }); tag != "TweenLerpMissing" {
		t.Errorf("int: panic tag = %q, want TweenLerpMissing", tag)
	}
}

func TestTweenWithALerpMethodInterpolates(t *testing.T) {
	c := NewTween(ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF))
	if got := c.Lerp(0.5); got != (ui.NewColor(0xFF000000)).Lerp(ui.NewColor(0xFFFFFFFF), 0.5) {
		t.Errorf("Lerp(0.5) = %v", got)
	}
	r := NewTween(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRectFromLTWH(2, 2, 2, 2))
	if got, want := r.Lerp(0.5), (ui.NewRectFromLTWH(0, 0, 10, 10)).Lerp(ui.NewRectFromLTWH(2, 2, 2, 2), 0.5); got != want {
		t.Errorf("Rect Lerp(0.5) = %v, want %v", got, want)
	}
}

func TestCanChainTweens(t *testing.T) {
	tween := NewTween(0.30, 0.50)
	oneLine(t, "tween", tween)
	chain := tween.Chain(NewTween(0.50, 1.0))
	controller := NewAnimationController(newTester())
	moreOrLess(t, "chain.Evaluate", chain.Evaluate(controller), 0.40)
	oneLine(t, "chain", chain.(interface{ String() string }))
}

func TestCanAnimateTweens(t *testing.T) {
	tween := NewTween(0.30, 0.50)
	controller := NewAnimationController(newTester())
	animation := tween.Animate(controller)
	controller.SetValue(0.50)
	moreOrLess(t, "animation.Value", animation.Value(), 0.40)
	oneLine(t, "animation", animation)
	if animation.Status() != AnimationStatusForward {
		t.Errorf("status = %v, want forward", animation.Status())
	}
}

func TestCanDriveTweens(t *testing.T) {
	tween := NewTween(0.30, 0.50)
	controller := NewAnimationController(newTester())
	animation := Drive(controller.View(), Animatable[float64](tween))
	controller.SetValue(0.50)
	moreOrLess(t, "animation.Value", animation.Value(), 0.40)
	oneLine(t, "animation", animation)
}

func TestSizeTween(t *testing.T) {
	tween := NewSizeTween(foundation.Some(ui.Size{}), foundation.Some(ui.Size{Width: 20, Height: 30}))
	if got := tween.Lerp(0.5); got != foundation.Some(ui.Size{Width: 10, Height: 15}) {
		t.Errorf("Lerp(0.5) = %v", got)
	}
	oneLine(t, "tween", tween)
	none := NewSizeTween(foundation.None[ui.Size](), foundation.None[ui.Size]())
	if none.Lerp(0.5).IsSome() {
		t.Errorf("a size tween of none should stay none")
	}
}

func TestIntTween(t *testing.T) {
	tween := NewIntTween(5, 9)
	if got := tween.Lerp(0.5); got != 7 {
		t.Errorf("Lerp(0.5) = %d, want 7", got)
	}
	if got := tween.Lerp(0.7); got != 8 {
		t.Errorf("Lerp(0.7) = %d, want 8", got)
	}
}

func TestRectTween(t *testing.T) {
	a, b := ui.NewRectFromLTWH(5, 3, 7, 11), ui.NewRectFromLTWH(8, 12, 14, 18)
	tween := NewRectTween(foundation.Some(a), foundation.Some(b))
	if got := tween.Lerp(0.5); got != foundation.Some(a.Lerp(b, 0.5)) {
		t.Errorf("Lerp(0.5) = %v, want %v", got, a.Lerp(b, 0.5))
	}
	oneLine(t, "tween", tween)
}

func TestConstantTween(t *testing.T) {
	tween := NewConstantTween(100.0)
	if tween.Begin != 100 || tween.End != 100 {
		t.Errorf("begin, end = %v, %v", tween.Begin, tween.End)
	}
	for _, at := range []float64{0, 0.5, 1} {
		if got := tween.Lerp(at); got != 100 {
			t.Errorf("Lerp(%v) = %v", at, got)
		}
	}
}

func TestReverseTween(t *testing.T) {
	tween := NewReverseTween(NewIntTween(5, 9))
	if got := tween.Lerp(0.5); got != 7 {
		t.Errorf("Lerp(0.5) = %d, want 7", got)
	}
	if got := tween.Lerp(0.7); got != 6 {
		t.Errorf("Lerp(0.7) = %d, want 6", got)
	}
}

func TestColorTween(t *testing.T) {
	tween := NewColorTween(foundation.Some(ui.NewColor(0xff000000)), foundation.Some(ui.NewColor(0xffffffff)))
	for _, tt := range []struct {
		at   float64
		want ui.Color
	}{{0, ui.NewColor(0xff000000)}, {0.5, ui.NewColor(0xff7f7f7f)}, {0.7, ui.NewColor(0xffb2b2b2)}, {1, ui.NewColor(0xffffffff)}} {
		got, _ := tween.Lerp(tt.at).Get()
		// isSameColorAs: each channel within 0.004.
		if !sameColor(got, tt.want) {
			t.Errorf("Lerp(%v) = %v, want %v", tt.at, got, tt.want)
		}
	}
}

func TestStepTween(t *testing.T) {
	tween := NewStepTween(5, 9)
	if got := tween.Lerp(0.5); got != 7 {
		t.Errorf("Lerp(0.5) = %d, want 7", got)
	}
	if got := tween.Lerp(0.7); got != 7 {
		t.Errorf("Lerp(0.7) = %d, want 7", got)
	}
}

func TestCurveTween(t *testing.T) {
	tween := NewCurveTween(Curves.EaseIn)
	for _, tt := range []struct{ at, want float64 }{{0, 0}, {0.5, 0.31640625}, {1, 1}} {
		near(t, "CurveTween", tween.Transform(tt.at), tt.want, 1e-8)
	}
}

func TestCurveTweenWithAnInvalidCurveEndpoint(t *testing.T) {
	skipUnlessDebug(t)
	tween := NewCurveTween(bogusCurve{})
	if tag := panicTag(t, func() { tween.Transform(0) }); tag != "CurveEndpointInvalid" {
		t.Errorf("panic tag = %q, want CurveEndpointInvalid", tag)
	}
}

func sameColor(a, b ui.Color) bool {
	d := func(x, y float32) bool { return math.Abs(float64(x-y)) <= 0.004 }
	return d(a.A, b.A) && d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}
