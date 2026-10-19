package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/implicit_animations_test.dart. The
// SliverAnimatedOpacity cases are in sliver_transitions_test.go.

const (
	animationDuration = 1000 * time.Millisecond
	additionalDelay   = 1 * time.Millisecond
)

func TestBoxConstraintsTweenControl(t *testing.T) {
	tween := NewBoxConstraintsTween(rendering.NewBoxConstraintsTight(ui.Size{Width: 20, Height: 50}), rendering.NewBoxConstraintsTight(ui.Size{Width: 10, Height: 30}))
	r := tween.Lerp(0.25)
	if r.MinWidth != 17.5 || r.MaxWidth != 17.5 || r.MinHeight != 45 || r.MaxHeight != 45 {
		t.Errorf("lerp(0.25) = %v, want tight(17.5, 45)", r)
	}
}

func TestDecorationTweenControl(t *testing.T) {
	tween := NewDecorationTween(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFF00FF00))}),
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFFFF00))}))
	r, ok := tween.Lerp(0.25).(*painting.BoxDecoration)
	if !ok {
		t.Fatalf("lerp = %T, want *painting.BoxDecoration", tween.Lerp(0.25))
	}
	if c, _ := r.Color.Get(); c.Alpha() != 0xFF || c.Red() < 0x3F || c.Red() > 0x40 || c.Green() != 0xFF || c.Blue() != 0 {
		t.Errorf("color = %v, want 0xFF3FFF00", r.Color)
	}
}

func TestEdgeInsetsTweenControl(t *testing.T) {
	tween := NewEdgeInsetsTween(painting.NewEdgeInsetsSymmetric(50, 0), painting.NewEdgeInsetsFromLTRB(0, 10, 0, 30))
	r := tween.Lerp(0.25)
	if r.Left != 0 || r.Right != 0 || r.Top != 40 || r.Bottom != 45 {
		t.Errorf("lerp(0.25) = %v, want (0, 40, 0, 45)", r)
	}
}

func TestMatrix4TweenControl(t *testing.T) {
	tween := NewMatrix4Tween(ui.NewMatrix4TranslationValues(10, 20, 30), ui.NewMatrix4TranslationValues(14, 24, 34))
	r := tween.Lerp(0.25)
	if want := ui.NewMatrix4TranslationValues(11, 21, 31); !r.Equals(want) {
		t.Errorf("lerp(0.25) = %v, want %v", r, want)
	}
}

func TestMatrix4TweenWithoutEndsPanics(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "Matrix4TweenNull", func() { NewMatrix4Tween(nil, ui.NewMatrix4TranslationValues(1, 1, 1)).Lerp(0.5) })
}

// rebuildState is the state of TestAnimatedWidget: it counts its builds and
// builds a Stack with the animated widget and a toggle.
type rebuildState struct {
	StateOf[*testAnimated]
	builds   int
	toggle   bool
	callback func()
	switchK  foundation.Key
	// animated is getAnimatedWidget.
	animated func(s *rebuildState) Widget
}

// testAnimated is Flutter's TestAnimatedWidget: it returns the state it was
// given from CreateState.
type testAnimated struct {
	StatefulWidget
	state *rebuildState
}

func newTestAnimated(s *rebuildState) *testAnimated {
	return &testAnimated{NewStatefulWidgetBase(), s}
}

func (w *testAnimated) CreateState() State { return w.state }

func (s *rebuildState) Build(BuildContext) Widget {
	s.builds++
	return NewStack([]Widget{s.animated(s), s.buildToggle()})
}

func (s *rebuildState) buildToggle() Widget {
	return NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(48.0)}), GestureDetectorOpts{
		Key:      s.switchK,
		Behavior: rendering.HitTestBehaviorOpaque,
		OnTap:    func() { s.SetState(func() { s.toggle = !s.toggle }) },
	})
}

func newRebuildState(callback func(), animated func(s *rebuildState) Widget) *rebuildState {
	return &rebuildState{callback: callback, switchK: foundation.NewKey("switchKey"), animated: animated}
}

func wrap(child Widget) Widget {
	return NewDirectionality(ui.TextDirectionLTR, NewCenter(child))
}

// pumpAnimated pumps the TestAnimatedWidget of the state inside the wrapper.
func pumpAnimated(ts *gluttertest.Tester, s *rebuildState) {
	ts.PumpWidget(wrap(newTestAnimated(s)))
}

func tapTest2and3(t *testing.T, ts *gluttertest.Tester, f gluttertest.Finder, called *int) {
	t.Helper()
	for want := 2; want <= 3; want++ {
		ts.Tap(f)
		ts.Pump()
		ts.Pump(animationDuration + additionalDelay)
		if *called != want {
			t.Errorf("onEnd called %d times, want %d", *called, want)
		}
	}
}

func TestImplicitAnimationsOnEnd(t *testing.T) {
	child := func() Widget { return NewPlaceholder(nil) }
	d := animationDuration
	tests := []struct {
		name string
		make func(toggle bool, cb func()) Widget
	}{
		{"AnimatedContainer", func(toggle bool, cb func()) Widget {
			o := AnimatedContainerOpts{OnEnd: cb, Width: foundation.Some(20.0)}
			if toggle {
				o.Width = foundation.Some(10.0)
				o.ForegroundDecoration = painting.NewBoxDecoration()
			}
			return NewAnimatedContainer(d, child(), o)
		}},
		{"AnimatedPadding", func(toggle bool, cb func()) Widget {
			p := painting.NewEdgeInsetsAll(16)
			if toggle {
				p = painting.NewEdgeInsetsAll(8)
			}
			return NewAnimatedPadding(p, d, child(), AnimatedPaddingOpts{OnEnd: cb})
		}},
		{"AnimatedAlign", func(toggle bool, cb func()) Widget {
			a := painting.AlignmentBottomRight
			if toggle {
				a = painting.AlignmentTopLeft
			}
			return NewAnimatedAlign(a, d, child(), AnimatedAlignOpts{OnEnd: cb})
		}},
		{"AnimatedPositioned", func(toggle bool, cb func()) Widget {
			left := 20.0
			if toggle {
				left = 10
			}
			return NewAnimatedPositioned(d, child(), AnimatedPositionedOpts{OnEnd: cb, Left: foundation.Some(left)})
		}},
		{"AnimatedPositionedDirectional", func(toggle bool, cb func()) Widget {
			start := 20.0
			if toggle {
				start = 10
			}
			return NewAnimatedPositionedDirectional(d, child(), AnimatedPositionedDirectionalOpts{OnEnd: cb, Start: foundation.Some(start)})
		}},
		{"AnimatedSlide", func(toggle bool, cb func()) Widget {
			o := ui.Offset{}
			if toggle {
				o = ui.Offset{Dx: 1, Dy: 1}
			}
			return NewAnimatedSlide(o, d, child(), AnimatedSlideOpts{OnEnd: cb})
		}},
		{"AnimatedScale", func(toggle bool, cb func()) Widget {
			s := 1.0
			if toggle {
				s = 2
			}
			return NewAnimatedScale(s, d, child(), AnimatedScaleOpts{OnEnd: cb})
		}},
		{"AnimatedRotation", func(toggle bool, cb func()) Widget {
			r := 0.0
			if toggle {
				r = 1.5
			}
			return NewAnimatedRotation(r, d, child(), AnimatedRotationOpts{OnEnd: cb})
		}},
		{"AnimatedOpacity", func(toggle bool, cb func()) Widget {
			o := 0.0
			if toggle {
				o = 1
			}
			return NewAnimatedOpacity(o, d, child(), AnimatedOpacityOpts{OnEnd: cb})
		}},
		{"AnimatedFractionallySizedBox", func(toggle bool, cb func()) Widget {
			f := 0.75
			if toggle {
				f = 0.25
			}
			return NewAnimatedFractionallySizedBox(d, child(), AnimatedFractionallySizedBoxOpts{OnEnd: cb, HeightFactor: foundation.Some(f), WidthFactor: foundation.Some(f)})
		}},
		{"AnimatedDefaultTextStyle", func(toggle bool, cb func()) Widget {
			style := painting.NewTextStyle(painting.TextStyleOpts{FontStyle: ui.FontStyleNormal})
			if toggle {
				style = painting.NewTextStyle(painting.TextStyleOpts{FontStyle: ui.FontStyleItalic})
			}
			return NewAnimatedDefaultTextStyle(style, d, child(), AnimatedDefaultTextStyleOpts{OnEnd: cb})
		}},
		{"AnimatedPhysicalModel", func(toggle bool, cb func()) Widget {
			c := ui.NewColor(0xFF00FF00)
			if toggle {
				c = ui.NewColor(0xFFFF0000)
			}
			return NewAnimatedPhysicalModel(c, ui.NewColor(0xFF0000FF), d, child(), AnimatedPhysicalModelOpts{OnEnd: cb})
		}},
		{"TweenAnimationBuilder", func(toggle bool, cb func()) Widget {
			tween := animation.NewTween(2.0, 1.0)
			if toggle {
				tween = animation.NewTween(1.0, 2.0)
			}
			return NewTweenAnimationBuilder(tween, d, func(_ BuildContext, size float64, c Widget) Widget {
				return NewSizedBox(c, SizedBoxOpts{Width: foundation.Some(size), Height: foundation.Some(size)})
			}, child(), TweenAnimationBuilderOpts[float64]{OnEnd: cb})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			called := 0
			s := newRebuildState(func() { called++ }, func(s *rebuildState) Widget { return tc.make(s.toggle, s.callback) })
			pumpAnimated(ts, s)
			f := gluttertest.ByKey(s.switchK)
			ts.Tap(f)
			ts.Pump()
			if called != 0 {
				t.Errorf("after the tap: called = %d, want 0", called)
			}
			ts.Pump(animationDuration)
			if called != 0 {
				t.Errorf("at the end of the duration: called = %d, want 0", called)
			}
			ts.Pump(additionalDelay)
			if called != 1 {
				t.Errorf("after the delay: called = %d, want 1", called)
			}
			tapTest2and3(t, ts, f, &called)
		})
	}
}

// transitionStep is one pump of a transition test and the value expected after it.
type transitionStep struct {
	pump time.Duration
	want float64
}

func TestImplicitAnimationsTransitionValues(t *testing.T) {
	steps := func(v ...float64) []transitionStep {
		return []transitionStep{{0, v[0]}, {500 * time.Millisecond, v[1]}, {250 * time.Millisecond, v[2]}, {250 * time.Millisecond, v[3]}}
	}
	tests := []struct {
		name  string
		make  func(toggle bool) Widget
		value func(ts *gluttertest.Tester) float64
		steps []transitionStep
	}{
		{"AnimatedScale", func(toggle bool) Widget {
			s := 1.0
			if toggle {
				s = 2
			}
			return NewAnimatedScale(s, animationDuration, NewPlaceholder(nil))
		}, func(ts *gluttertest.Tester) float64 {
			return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*Placeholder](), gluttertest.ByType[*ScaleTransition]()))).(*ScaleTransition).Scale().Value()
		}, steps(1, 1.5, 1.75, 2)},
		{"AnimatedRotation", func(toggle bool) Widget {
			r := 0.0
			if toggle {
				r = 1.5
			}
			return NewAnimatedRotation(r, animationDuration, NewPlaceholder(nil))
		}, func(ts *gluttertest.Tester) float64 {
			return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*Placeholder](), gluttertest.ByType[*RotationTransition]()))).(*RotationTransition).Turns().Value()
		}, steps(0, 0.75, 1.125, 1.5)},
		{"AnimatedOpacity", func(toggle bool) Widget {
			o := 0.0
			if toggle {
				o = 1
			}
			return NewAnimatedOpacity(o, animationDuration, NewPlaceholder(nil))
		}, func(ts *gluttertest.Tester) float64 {
			return ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*Placeholder](), gluttertest.ByType[*FadeTransition]()))).(*FadeTransition).Opacity().Value()
		}, steps(0, 0.5, 0.75, 1)},
		{"AnimatedSlide", func(toggle bool) Widget {
			o := ui.Offset{}
			if toggle {
				o = ui.Offset{Dx: 1, Dy: 1}
			}
			return NewAnimatedSlide(o, animationDuration, NewPlaceholder(nil))
		}, func(ts *gluttertest.Tester) float64 {
			p := ts.Widget(gluttertest.First(gluttertest.Ancestor(gluttertest.ByType[*Placeholder](), gluttertest.ByType[*SlideTransition]()))).(*SlideTransition).Position().Value()
			if p.Dx != p.Dy {
				t.Errorf("offset = %v, want equal components", p)
			}
			return p.Dx
		}, steps(0, 0.5, 0.75, 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			s := newRebuildState(nil, func(s *rebuildState) Widget { return tc.make(s.toggle) })
			pumpAnimated(ts, s)
			if s.builds != 1 {
				t.Fatalf("builds = %d, want 1", s.builds)
			}
			ts.Tap(gluttertest.ByKey(s.switchK))
			if s.builds != 1 {
				t.Fatalf("builds after the tap = %d, want 1", s.builds)
			}
			for i, step := range tc.steps {
				if step.pump == 0 {
					ts.Pump()
				} else {
					ts.Pump(step.pump)
				}
				if got := tc.value(ts); got != step.want {
					t.Errorf("step %d: value = %v, want %v", i, got, step.want)
				}
				if s.builds != 2 {
					t.Errorf("step %d: builds = %d, want 2", i, s.builds)
				}
			}
		})
	}
}

// animatedState is what the state of an implicit animation gives a test.
type animatedState interface {
	State
	Animation() animation.Animation[float64]
}

func TestImplicitAnimationDisposesItsCurvedAnimationOnChange(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[animatedState]()
	curve := foundation.NewValueNotifier[animation.Curve](animation.NewInterval(0, 0.5))
	ts.PumpWidget(wrap(NewValueListenableBuilder[animation.Curve](curve, func(_ BuildContext, c animation.Curve, _ Widget) Widget {
		return NewAnimatedOpacity(1, time.Second, NewContainer(nil, ContainerOpts{Color: foundation.Some(ui.NewColor(0xFF00FF00))}), AnimatedOpacityOpts{Key: key, Curve: c})
	}, nil)))
	first, ok := key.CurrentState()
	if !ok {
		t.Fatal("no state")
	}
	firstAnim, ok := first.Animation().(*animation.CurvedAnimation)
	if !ok {
		t.Fatalf("animation = %T, want *CurvedAnimation", first.Animation())
	}
	if firstAnim.IsDisposed {
		t.Error("the first animation is disposed at the start")
	}
	curve.SetValue(animation.NewInterval(0, 0.6))
	ts.PumpAndSettle()
	second, _ := key.CurrentState()
	secondAnim := second.Animation().(*animation.CurvedAnimation)
	if first != second {
		t.Error("the state changed")
	}
	if firstAnim == secondAnim {
		t.Error("the animation was not replaced")
	}
	if !firstAnim.IsDisposed {
		t.Error("the first animation is not disposed")
	}
	if secondAnim.IsDisposed {
		t.Error("the second animation is disposed")
	}
	ts.PumpWidget(wrap(NewOffstage(nil)))
	ts.PumpAndSettle()
	if !secondAnim.IsDisposed {
		t.Error("the second animation is not disposed after the widget left")
	}
}

func TestAnimatedPhysicalModelDefaultArgsMatchPhysicalModel(t *testing.T) {
	color := ui.NewColor(0)
	animated := NewAnimatedPhysicalModel(color, color, 0, NewSizedBoxShrink(nil))
	plain := NewPhysicalModel(color, NewSizedBoxShrink(nil), PhysicalModelOpts{ShadowColor: foundation.Some(color)})
	if animated.Shape() != plain.Shape() {
		t.Errorf("shape = %v, want %v", animated.Shape(), plain.Shape())
	}
	if animated.ClipBehavior() != plain.ClipBehavior() {
		t.Errorf("clipBehavior = %v, want %v", animated.ClipBehavior(), plain.ClipBehavior())
	}
	if animated.BorderRadius() != plain.BorderRadius() {
		t.Errorf("borderRadius = %v, want %v", animated.BorderRadius(), plain.BorderRadius())
	}
}

func TestImplicitAnimationsDoNotCrashAtZeroArea(t *testing.T) {
	text := func() Widget { return NewText("X") }
	tests := []struct {
		name  string
		inner func() Widget
		find  gluttertest.Finder
		box   func(Widget) Widget
	}{
		{"AnimatedSlide", func() Widget { return NewAnimatedSlide(ui.Offset{Dx: 100}, 300*time.Millisecond, nil) }, gluttertest.ByType[*AnimatedSlide](), NewSizedBoxShrinkWidget},
		{"AnimatedScale", func() Widget { return NewAnimatedScale(2, 300*time.Millisecond, text()) }, gluttertest.ByType[*AnimatedScale](), NewSizedBoxShrinkWidget},
		{"AnimatedRotation", func() Widget { return NewAnimatedRotation(0.75, 300*time.Millisecond, text()) }, gluttertest.ByType[*AnimatedRotation](), NewSizedBoxShrinkWidget},
		{"AnimatedOpacity", func() Widget { return NewAnimatedOpacity(0.5, 300*time.Millisecond, nil) }, gluttertest.ByType[*AnimatedOpacity](), func(c Widget) Widget { return NewSizedBox(c) }},
		{"AnimatedDefaultTextStyle", func() Widget {
			return NewAnimatedDefaultTextStyle(painting.NewTextStyle(painting.TextStyleOpts{FontStyle: ui.FontStyleItalic}), 300*time.Millisecond, text())
		}, gluttertest.ByType[*AnimatedDefaultTextStyle](), NewSizedBoxShrinkWidget},
		{"AnimatedPhysicalModel", func() Widget {
			return NewAnimatedPhysicalModel(ui.NewColor(0xFF009688), ui.NewColor(0xFF64FFDA), 300*time.Millisecond, text())
		}, gluttertest.ByType[*AnimatedPhysicalModel](), NewSizedBoxShrinkWidget},
		{"AnimatedFractionallySizedBox", func() Widget {
			return NewAnimatedFractionallySizedBox(300*time.Millisecond, nil, AnimatedFractionallySizedBoxOpts{WidthFactor: foundation.Some(0.5), HeightFactor: foundation.Some(0.5)})
		}, gluttertest.ByType[*AnimatedFractionallySizedBox](), NewSizedBoxShrinkWidget},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(tc.box(tc.inner()))))
			ts.PumpAndSettle()
			if size := ts.Size(tc.find); size != (ui.Size{}) {
				t.Errorf("size = %v, want zero", size)
			}
		})
	}
}

// NewSizedBoxShrinkWidget is SizedBox.shrink as a function of the child.
func NewSizedBoxShrinkWidget(child Widget) Widget { return NewSizedBoxShrink(child) }

// vsyncOf makes the tickers of the scheduler of the tester (Flutter's TestVSync).
type vsyncOf struct{ s *scheduler.SchedulerBinding }

func (v vsyncOf) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	return scheduler.NewTicker(v.s, onTick)
}

func TestSlideTransitionDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{})
	controller := animation.NewAnimationController(vsyncOf{ts.Binding.Scheduler()}, animation.AnimationControllerOpts{
		Value: foundation.Some(1.0), Duration: foundation.Some(2 * time.Second)})
	defer controller.Dispose()
	curved := animation.NewCurvedAnimation(controller, animation.Curves.Linear)
	defer curved.Dispose()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(
		NewSlideTransition(animation.NewTween(ui.Offset{}, ui.Offset{Dx: 1.5}).Animate(curved), NewPlaceholder(nil)))))
	if size := ts.Size(gluttertest.ByType[*SlideTransition]()); size != (ui.Size{}) {
		t.Errorf("size = %v, want zero", size)
	}
}
