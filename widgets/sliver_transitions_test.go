package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/transitions_test.dart (SliverFadeTransition)
// and implicit_animations_test.dart (SliverAnimatedOpacity).

func TestSliverFadeTransitionAnimates(t *testing.T) {
	ts := gluttertest.New(t)
	controller := newVsyncController(ts)
	defer controller.Dispose()
	anim := animation.NewTween(0.0, 1.0).Animate(controller)
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverFadeTransition(anim, NewSliverToBoxAdapter(NewText("Fade In")))})))
	opacity := func() float64 {
		w := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByText("Fade In"), gluttertest.ByType[*SliverFadeTransition]()))).(*SliverFadeTransition)
		return w.Opacity().Value()
	}
	for _, v := range []float64{0, 0.25, 0.5, 0.75, 1} {
		controller.SetValue(v)
		ts.Pump()
		if got := opacity(); got != v {
			t.Errorf("opacity = %v, want %v", got, v)
		}
	}
}

// sliverAnimatedOpacityApp is TestAnimatedWidget with a SliverAnimatedOpacity:
// the toggle is a tap target after the animated sliver.
func sliverAnimatedOpacityApp(toggle *bool, onEnd func(), builds *int) Widget {
	switchKey := foundation.NewValueKey("switch")
	return NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		*builds++
		opacity := 0.0
		if *toggle {
			opacity = 1
		}
		return ltr(NewCustomScrollView([]Widget{
			NewSliverAnimatedOpacity(opacity, animationDuration, NewSliverToBoxAdapter(NewPlaceholder(nil)), SliverAnimatedOpacityOpts{OnEnd: onEnd}),
			NewSliverToBoxAdapter(NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Key: switchKey, Height: foundation.Some(50.0)}),
				GestureDetectorOpts{Behavior: rendering.HitTestBehaviorOpaque, OnTap: func() { setState(func() { *toggle = !*toggle }) }})),
		}))
	})
}

func TestSliverAnimatedOpacity(t *testing.T) {
	switchFinder := gluttertest.ByKey(foundation.NewValueKey("switch"))
	t.Run("onEnd", func(t *testing.T) {
		ts := gluttertest.New(t)
		toggle, builds, ended := false, 0, 0
		ts.PumpWidget(sliverAnimatedOpacityApp(&toggle, func() { ended++ }, &builds))
		ts.Tap(switchFinder)
		ts.Pump()
		if ended != 0 {
			t.Fatalf("ended = %d, want 0", ended)
		}
		ts.Pump(animationDuration)
		if ended != 0 {
			t.Fatalf("ended = %d after the duration, want 0", ended)
		}
		ts.Pump(additionalDelay)
		if ended != 1 {
			t.Errorf("ended = %d, want 1", ended)
		}
		// Tapping again animates back and ends again.
		ts.Tap(switchFinder)
		ts.Pump()
		ts.Pump(animationDuration + additionalDelay)
		if ended != 2 {
			t.Errorf("ended = %d, want 2", ended)
		}
	})
	t.Run("transition", func(t *testing.T) {
		ts := gluttertest.New(t)
		toggle, builds := false, 0
		ts.PumpWidget(sliverAnimatedOpacityApp(&toggle, nil, &builds))
		opacityWidget := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*Placeholder](), gluttertest.ByType[*SliverFadeTransition]()))).(*SliverFadeTransition)
		if builds != 1 {
			t.Fatalf("builds = %d, want 1", builds)
		}
		ts.Tap(switchFinder)
		if builds != 1 {
			t.Fatalf("builds after the tap = %d, want 1", builds)
		}
		ts.Pump()
		if builds != 2 {
			t.Fatalf("builds after the pump = %d, want 2", builds)
		}
		if got := opacityWidget.Opacity().Value(); got != 0 {
			t.Errorf("opacity = %v, want 0", got)
		}
		for _, step := range []struct {
			pump time.Duration
			want float64
		}{{animationDuration / 2, 0.5}, {animationDuration / 4, 0.75}, {animationDuration / 4, 1}} {
			ts.Pump(step.pump)
			if got := opacityWidget.Opacity().Value(); got != step.want {
				t.Errorf("opacity = %v, want %v", got, step.want)
			}
		}
	})
	t.Run("the opacity must be between 0 and 1", func(t *testing.T) {
		skipUnlessDebug(t)
		requireTag(t, "OpacityOutOfRange", func() { NewSliverAnimatedOpacity(2, time.Second, nil) })
	})
}
