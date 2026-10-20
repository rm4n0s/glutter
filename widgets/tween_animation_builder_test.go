// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/tween_animation_builder_test.dart.

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

type intLog struct{ values []int }

func (l *intLog) want(t *testing.T, name string, want ...int) {
	t.Helper()
	if !slices.Equal(l.values, want) {
		t.Errorf("%s: values = %v, want %v", name, l.values, want)
	}
}

// firstFrame checks and removes the second call of the builder that the first
// tick of a controller started while the tree was attached causes: the ticker
// started outside a frame ticks at elapsed 0 in the first frame, and the
// controller notifies its listeners.
//
// Deviation from the Dart test, which expects a single call after pumpWidget
// (SPEC section 12.10): the Go tester attaches the tree outside a frame.
func (l *intLog) firstFrame(t *testing.T, initial int) {
	t.Helper()
	if !slices.Equal(l.values, []int{initial, initial}) {
		t.Fatalf("after the first frame: values = %v, want [%d %d]", l.values, initial, initial)
	}
	l.values = l.values[:1]
}

func (l *intLog) builder() ValueWidgetBuilder[int] {
	return func(_ BuildContext, i int, _ Widget) Widget {
		l.values = append(l.values, i)
		return NewPlaceholder(nil)
	}
}

func intBuilder(l *intLog, tween *animation.Tween[int], d time.Duration, o TweenAnimationBuilderOpts[int]) *TweenAnimationBuilder[int] {
	return NewTweenAnimationBuilder(tween, d, l.builder(), nil, o)
}

func TestTweenAnimationBuilderAnimatesForwardWhenBuilt(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	endCount := 0
	ts.PumpWidget(intBuilder(l, animation.NewIntTween(10, 110), time.Second, TweenAnimationBuilderOpts[int]{OnEnd: func() { endCount++ }}))
	l.firstFrame(t, 10)
	if endCount != 0 {
		t.Errorf("endCount = %d", endCount)
	}
	l.want(t, "built", 10)
	ts.Pump(ms(500))
	l.want(t, "half", 10, 60)
	ts.Pump(ms(501))
	if endCount != 1 {
		t.Errorf("endCount = %d, want 1", endCount)
	}
	l.want(t, "end", 10, 60, 110)
	ts.Pump(ms(500))
	if endCount != 1 {
		t.Errorf("endCount = %d, want 1", endCount)
	}
	l.want(t, "after the end", 10, 60, 110)
}

func TestTweenAnimationBuilderNoInitialAnimationWhenBeginIsNullOrEqualToEnd(t *testing.T) {
	for name, tween := range map[string]*animation.Tween[int]{
		"begin=null": animation.NewIntTween(100, 100).WithoutBegin(),
		"begin=end":  animation.NewIntTween(100, 100),
	} {
		ts := gluttertest.New(t)
		l := &intLog{}
		endCount := 0
		ts.PumpWidget(intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{OnEnd: func() { endCount++ }}))
		l.want(t, name, 100)
		ts.Pump(2 * time.Second)
		if endCount != 0 {
			t.Errorf("%s: endCount = %d", name, endCount)
		}
		l.want(t, name+" later", 100)
	}
}

func TestTweenAnimationBuilderReplaceTweenAnimatesNewTween(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	l.firstFrame(t, 0)
	l.want(t, "first", 0)
	ts.Pump(2 * time.Second) // finish first animation
	l.want(t, "finished", 0, 100)
	ts.PumpWidget(build(animation.NewIntTween(100, 200)))
	l.want(t, "second built", 0, 100, 100)
	ts.Pump(ms(500))
	l.want(t, "half", 0, 100, 100, 150)
	ts.Pump(ms(500))
	l.want(t, "end", 0, 100, 100, 150, 200)
}

func TestTweenAnimationBuilderCurveIsRespected(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int], curve animation.Curve) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{Curve: curve})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100), animation.Curves.EaseInExpo))
	l.firstFrame(t, 0)
	l.want(t, "built", 0)
	ts.Pump(ms(500))
	if last := l.values[len(l.values)-1]; last >= 50 || last <= 0 {
		t.Errorf("last = %d, want between 0 and 50", last)
	}
	ts.Pump(2 * time.Second)
	l.values = nil
	ts.PumpWidget(build(animation.NewIntTween(100, 200), animation.Curves.Linear))
	l.want(t, "second built", 100)
	ts.Pump(ms(500))
	l.want(t, "second half", 100, 150)
}

func TestTweenAnimationBuilderDurationIsRespected(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int], d time.Duration) Widget {
		return intBuilder(l, tween, d, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100), time.Second))
	l.firstFrame(t, 0)
	l.want(t, "built", 0)
	ts.Pump(ms(500))
	l.want(t, "half", 0, 50)
	ts.Pump(2 * time.Second)
	l.values = nil
	ts.PumpWidget(build(animation.NewIntTween(100, 200), 2*time.Second))
	l.want(t, "second built", 100)
	ts.Pump(ms(500))
	l.want(t, "second quarter", 100, 125)
}

func TestTweenAnimationBuilderChildIsIntegratedIntoTheTree(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewTweenAnimationBuilder(animation.NewIntTween(0, 100), time.Second,
		func(_ BuildContext, _ int, child Widget) Widget { return child }, NewText("Hello World"))))
	if n := ts.Count(gluttertest.ByText("Hello World")); n != 1 {
		t.Errorf("found %d texts, want 1", n)
	}
}

func TestTweenAnimationBuilderChangeTweenGaplessWhileRunningForward(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	l.firstFrame(t, 0)
	l.want(t, "built", 0)
	ts.Pump(ms(500))
	l.want(t, "half", 0, 50)
	ts.PumpWidget(build(animation.NewIntTween(200, 300)))
	l.want(t, "changed", 0, 50, 50) // gapless: the animation continues where it left off
	ts.Pump(ms(500))
	l.want(t, "half again", 0, 50, 50, 175) // 175 is halfway between 50 and the new target 300
	ts.Pump(2 * time.Second)
	l.want(t, "end", 0, 50, 50, 175, 300)
}

func TestTweenAnimationBuilderChangeTweenWithTheSameInstancesAgain(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	tween1, tween2 := animation.NewIntTween(0, 100), animation.NewIntTween(200, 300)
	ts.PumpWidget(build(tween1))
	l.firstFrame(t, 0)
	ts.Pump(ms(500))
	ts.PumpWidget(build(tween2))
	ts.Pump(ms(500))
	ts.Pump(2 * time.Second)
	l.want(t, "values", 0, 50, 50, 175, 300)
}

func TestTweenAnimationBuilderChangingTweenWhileAGaplessChangeIsInProgress(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	l.firstFrame(t, 0)
	ts.Pump(ms(500))
	l.want(t, "first", 0, 50)
	l.values = nil
	ts.PumpWidget(build(animation.NewIntTween(200, 300)))
	ts.Pump(ms(500))
	l.want(t, "second", 50, 175)
	l.values = nil
	ts.PumpWidget(build(animation.NewIntTween(400, 501)))
	ts.Pump(ms(500))
	ts.Pump(ms(500))
	l.want(t, "third", 175, 338, 501)
}

func TestTweenAnimationBuilderChangingCurveWithoutARunningAnimationDoesNotAnimate(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(curve animation.Curve) Widget {
		return intBuilder(l, animation.NewIntTween(0, 100), time.Second, TweenAnimationBuilderOpts[int]{Curve: curve})
	}
	ts.PumpWidget(build(animation.Curves.Linear))
	l.firstFrame(t, 0)
	ts.Pump(2 * time.Second)
	l.want(t, "first", 0, 100)
	l.values = nil
	ts.PumpWidget(build(animation.Curves.EaseInExpo))
	l.want(t, "curve changed", 100)
	ts.Pump(2 * time.Second)
	l.want(t, "later", 100)
}

func TestTweenAnimationBuilderSettingTheSameTweenDoesNotAnimate(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	l.firstFrame(t, 0)
	ts.Pump(ms(500))
	ts.Pump(ms(500))
	l.want(t, "first", 0, 50, 100)
	l.values = nil
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	ts.Pump(ms(500))
	ts.Pump(ms(500))
	for _, v := range l.values {
		if v != 100 {
			t.Errorf("values = %v, want all 100", l.values)
			break
		}
	}
}

func TestTweenAnimationBuilderSettingTheSameTweenWhileAGaplessAnimationRuns(t *testing.T) {
	ts := gluttertest.New(t)
	l := &intLog{}
	build := func(tween *animation.Tween[int]) Widget {
		return intBuilder(l, tween, time.Second, TweenAnimationBuilderOpts[int]{})
	}
	ts.PumpWidget(build(animation.NewIntTween(0, 100)))
	l.firstFrame(t, 0)
	ts.Pump(ms(500))
	l.want(t, "first", 0, 50)
	ts.PumpWidget(build(animation.NewIntTween(200, 300)))
	ts.Pump(ms(500))
	l.want(t, "second", 0, 50, 50, 175)
	ts.PumpWidget(build(animation.NewIntTween(200, 300)))
	l.want(t, "same tween", 0, 50, 50, 175, 175)
	ts.Pump(ms(500))
	l.want(t, "end", 0, 50, 50, 175, 175, 300)
	l.values = nil
	ts.Pump(2 * time.Second)
	for _, v := range l.values {
		if v != 300 {
			t.Errorf("values = %v, want all 300", l.values)
			break
		}
	}
}

func TestTweenAnimationBuilderWorksWithNullableTweens(t *testing.T) {
	ts := gluttertest.New(t)
	var values []foundation.Opt[ui.Size]
	ts.PumpWidget(NewTweenAnimationBuilder(animation.NewSizeTween(foundation.None[ui.Size](), foundation.Some(ui.Size{Width: 10, Height: 10})), time.Second,
		func(_ BuildContext, s foundation.Opt[ui.Size], _ Widget) Widget {
			values = append(values, s)
			return NewPlaceholder(nil)
		}, nil))
	want := foundation.Some(ui.Size{Width: 10, Height: 10})
	if len(values) != 1 || values[0] != want {
		t.Errorf("values = %v, want [%v]", values, want)
	}
	ts.Pump(2 * time.Second)
	if len(values) != 1 {
		t.Errorf("values = %v, want a single build", values)
	}
}

func TestTweenAnimationBuilderDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{})
	ts.PumpWidget(NewCenter(NewTweenAnimationBuilder(animation.NewSizeTween(foundation.None[ui.Size](), foundation.Some(ui.Size{Width: 10, Height: 10})), time.Second,
		func(BuildContext, foundation.Opt[ui.Size], Widget) Widget { return NewPlaceholder(nil) }, nil)))
	if got := ts.Size(gluttertest.ByType[*TweenAnimationBuilder[foundation.Opt[ui.Size]]]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
	ts.PumpAndSettle()
}
