package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are the Navigator cases of modal_barrier_test.dart, left to
// step 10.2: a barrier in the second route of an app pops it when dismissed.
// Not ported: the two cases with WillPopScope (removed from Flutter).

var barrierKey = foundation.NewKey("barrier")

func barrierApp(second func() Widget) Widget {
	return navTestApp(navTestAppOpts{routes: map[string]WidgetBuilder{
		"/": func(bc BuildContext) Widget {
			return NewGestureDetector(navText("X"), GestureDetectorOpts{OnTap: func() { NavigatorPushNamed(bc, "/modal") }})
		},
		"/modal": func(BuildContext) Widget { return second() },
	}})
}

func TestModalBarrierPopsTheNavigator(t *testing.T) {
	seconds := map[string]func(onDismiss func()) Widget{
		"ModalBarrier": func(onDismiss func()) Widget {
			return NewModalBarrier(ModalBarrierOpts{Key: barrierKey, OnDismiss: onDismiss})
		},
		"AnimatedModalBarrier": func(onDismiss func()) Widget {
			return NewAnimatedModalBarrier(animation.NewAlwaysStoppedAnimation(foundation.Some(ui.NewColor(0xFFFF0000))),
				AnimatedModalBarrierOpts{Key: barrierKey, OnDismiss: onDismiss})
		},
	}
	open := func(t *testing.T, ts *gluttertest.Tester, second func() Widget) {
		t.Helper()
		ts.PumpWidget(barrierApp(second))
		if ts.Count(gluttertest.ByKey(barrierKey)) != 0 {
			t.Fatal("the barrier should not be visible at first")
		}
		// Tapping on X routes to the barrier.
		ts.Tap(gluttertest.ByText("X"))
		ts.Pump()            // begin transition
		ts.Pump(time.Second) // end transition
	}
	for name, second := range seconds {
		for _, button := range []struct {
			name string
			opts gluttertest.GestureOpts
		}{
			{"primary tap", gluttertest.GestureOpts{}},
			{"non-primary tap", gluttertest.GestureOpts{Buttons: foundation.Some(gestures.SecondaryButton)}},
		} {
			t.Run(name+" pops the Navigator when dismissed by a "+button.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				open(t, ts, func() Widget { return second(nil) })
				// Presses the barrier; it does not dismiss yet.
				gesture := ts.StartGesture(ts.Center(gluttertest.ByKey(barrierKey)), button.opts)
				ts.PumpAndSettle()
				if ts.Count(gluttertest.ByKey(barrierKey)) != 1 {
					t.Fatal("the barrier should still be there")
				}
				// Releases the pointer; the barrier is dismissed.
				gesture.Up()
				ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
				if ts.Count(gluttertest.ByKey(barrierKey)) != 0 {
					t.Error("the route should have been dismissed by tapping the barrier")
				}
			})
		}
		t.Run(name+" calls onDismiss and does not pop", func(t *testing.T) {
			calls := 0
			ts := gluttertest.New(t)
			open(t, ts, func() Widget { return second(func() { calls++ }) })
			if ts.Count(gluttertest.ByKey(barrierKey)) != 1 {
				t.Fatal("the barrier should show")
			}
			ts.Tap(gluttertest.ByKey(barrierKey))
			ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
			if calls != 1 {
				t.Errorf("onDismiss calls = %d, want 1", calls)
			}
			if ts.Count(gluttertest.ByKey(barrierKey)) != 1 {
				t.Error("the route should not have been dismissed: there is an onDismiss callback")
			}
		})
		t.Run(name+" may pop the Navigator when competing with other gestures", func(t *testing.T) {
			ts := gluttertest.New(t)
			open(t, ts, func() Widget {
				return NewStack([]Widget{
					second(nil),
					NewGestureDetector(NewContainer(nil), GestureDetectorOpts{
						OnVerticalDragStart: func(gestures.DragStartDetails) {},
						Behavior:            rendering.HitTestBehaviorTranslucent,
					}),
				}, StackOpts{TextDirection: ui.TextDirectionLTR})
			})
			ts.Tap(gluttertest.ByKey(barrierKey))
			ts.Pump()            // begin transition
			ts.Pump(time.Second) // end transition
			if ts.Count(gluttertest.ByKey(barrierKey)) != 0 {
				t.Error("the route should have been dismissed by tapping the barrier")
			}
		})
	}
}
