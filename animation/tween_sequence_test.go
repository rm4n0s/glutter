package animation

import (
	"math"
	"testing"
)

func TestTweenSequenceContract(t *testing.T) {
	if tag := panicTag(t, func() { NewTweenSequence[float64](nil) }); tag != "TweenSequenceInvalid" {
		t.Errorf("empty: panic tag = %q, want TweenSequenceInvalid", tag)
	}
	for _, w := range []float64{0, -1, math.NaN()} {
		items := []TweenSequenceItem[float64]{{Tween: NewTween(0.0, 1.0), Weight: w}}
		if tag := panicTag(t, func() { NewTweenSequence(items) }); tag != "TweenSequenceInvalid" {
			t.Errorf("weight %v: panic tag = %q, want TweenSequenceInvalid", w, tag)
		}
	}
	seq := NewTweenSequence([]TweenSequenceItem[float64]{{Tween: NewTween(0.0, 1.0), Weight: 1}})
	wantPanic(t, "TweenSequenceOutOfRange", func() { seq.Transform(1.5) })
	wantPanic(t, "TweenSequenceOutOfRange", func() { seq.Transform(-0.5) })
	wantPanic(t, "TweenSequenceNoInterval", func() { seq.Transform(math.NaN()) })
}

func TestFlippedTweenSequence(t *testing.T) {
	seq := NewFlippedTweenSequence([]TweenSequenceItem[float64]{
		{Tween: NewTween(0.0, 1.0), Weight: 1},
		{Tween: NewConstantTween(1.0), Weight: 1},
	})
	for _, tt := range []struct{ at, want float64 }{{0, 0}, {0.25, 0}, {0.5, 0}, {0.75, 0.5}, {1, 1}} {
		moreOrLess(t, "flipped", seq.Transform(tt.at), tt.want)
	}
	controller := NewAnimationController(newTester())
	animation := seq.Animate(controller)
	controller.SetValue(0.75)
	moreOrLess(t, "animated flipped", animation.Value(), 0.5)
}
