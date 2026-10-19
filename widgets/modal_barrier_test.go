package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/modal_barrier_test.dart. Not ported:
// the hover tests (the mouse tracker, step 10.4), the system alert sound
// (services), the Navigator tests (step 10.2) and the default mouse cursor
// (SystemMouseCursors, step 10.4). The semantics tree tests read the nodes of the
// tree, as there is no semantics tester.

const testRed = 0xFFFF0000

func TestModalBarrierInteractions(t *testing.T) {
	tapped := false
	tapTarget := func() Widget {
		return NewGestureDetector(NewSizedBoxSquare(foundation.Some(10.0), NewText("target", TextOpts{TextDirection: ui.TextDirectionLTR})),
			GestureDetectorOpts{OnTap: func() { tapped = true }})
	}
	barrier := func() Widget { return NewModalBarrier(ModalBarrierOpts{Dismissible: foundation.Some(false)}) }

	t.Run("prevents interactions with widgets behind it", func(t *testing.T) {
		tapped = false
		ts := gluttertest.New(t)
		ts.PumpWidget(NewStack([]Widget{tapTarget(), barrier()}, StackOpts{TextDirection: ui.TextDirectionLTR}))
		ts.TapAt(ts.TopLeft(gluttertest.ByText("target")).Add(ui.Offset{Dx: 1, Dy: 1}))
		if tapped {
			t.Error("the tap was not prevented by ModalBarrier")
		}
	})
	t.Run("does not prevent interactions with widgets in front of it", func(t *testing.T) {
		tapped = false
		ts := gluttertest.New(t)
		ts.PumpWidget(NewStack([]Widget{barrier(), tapTarget()}, StackOpts{TextDirection: ui.TextDirectionLTR}))
		ts.Tap(gluttertest.ByText("target"))
		if !tapped {
			t.Error("the tap was prevented by ModalBarrier")
		}
	})
	t.Run("does not prevent interactions with translucent widgets in front of it", func(t *testing.T) {
		dragged := false
		ts := gluttertest.New(t)
		ts.PumpWidget(NewStack([]Widget{barrier(), NewGestureDetector(
			NewCenter(NewText("target", TextOpts{TextDirection: ui.TextDirectionLTR})),
			GestureDetectorOpts{Behavior: rendering.HitTestBehaviorTranslucent, OnHorizontalDragStart: func(gestures.DragStartDetails) { dragged = true }})},
			StackOpts{TextDirection: ui.TextDirectionLTR}))
		ts.DragFrom(ts.BottomRight(gluttertest.ByType[*GestureDetector]()).Sub(ui.Offset{Dx: 10, Dy: 10}), ui.Offset{Dx: -20})
		if !dragged {
			t.Error("the drag was prevented by ModalBarrier")
		}
	})
}

func TestModalBarrierDismissal(t *testing.T) {
	t.Run("calls onDismiss when tapped", func(t *testing.T) {
		calls := 0
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewModalBarrier(ModalBarrierOpts{Key: key, OnDismiss: func() { calls++ }}))
		ts.Tap(gluttertest.ByKey(key))
		if calls != 1 {
			t.Errorf("onDismiss calls = %d, want 1", calls)
		}
	})
	t.Run("a tap of any button dismisses", func(t *testing.T) {
		calls := 0
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewModalBarrier(ModalBarrierOpts{Key: key, OnDismiss: func() { calls++ }}))
		ts.Tap(gluttertest.ByKey(key), gluttertest.GestureOpts{Buttons: foundation.Some(gestures.SecondaryButton)})
		if calls != 1 {
			t.Errorf("onDismiss calls = %d, want 1", calls)
		}
	})
	t.Run("does not call onDismiss when it is not dismissible", func(t *testing.T) {
		calls := 0
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewModalBarrier(ModalBarrierOpts{Key: key, Dismissible: foundation.Some(false), OnDismiss: func() { calls++ }}))
		ts.Tap(gluttertest.ByKey(key))
		if calls != 0 {
			t.Errorf("onDismiss calls = %d, want 0", calls)
		}
	})
	t.Run("a failing onDismiss is reported as a gesture error", func(t *testing.T) {
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewModalBarrier(ModalBarrierOpts{Key: key, OnDismiss: func() { panic("deliberate") }}))
		ts.Tap(gluttertest.ByKey(key))
		if err := ts.TakeError(); err == nil {
			t.Error("the error was not reported")
		}
	})
}

func TestModalBarrierSemantics(t *testing.T) {
	t.Run("an undismissible barrier is excluded from the semantics", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewModalBarrier(ModalBarrierOpts{Dismissible: foundation.Some(false)}))
		if w := ts.Widget(gluttertest.ByType[*ExcludeSemantics]()).(*ExcludeSemantics); !w.Excluding() {
			t.Error("the barrier is not excluded from the semantics")
		}
		if n := ts.Count(gluttertest.ByType[*BlockSemantics]()); n != 1 {
			t.Errorf("BlockSemantics = %d, want 1", n)
		}
	})
	t.Run("a dismissible barrier with a label is excluded on desktop", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss"})))
		if w := ts.Widget(gluttertest.ByType[*ExcludeSemantics]()).(*ExcludeSemantics); !w.Excluding() {
			t.Error("the barrier is not excluded on Linux, which does not dismiss with it")
		}
	})
	t.Run("barrierSemanticsDismissible false excludes the barrier on mobile", func(t *testing.T) {
		skipUnlessDebug(t)
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
		defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss"})))
		if w := ts.Widget(gluttertest.ByType[*ExcludeSemantics]()).(*ExcludeSemantics); w.Excluding() {
			t.Error("the barrier is excluded on Android")
		}
		ts.PumpWidget(ltr(NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss", BarrierSemanticsDismissible: foundation.Some(false)})))
		if w := ts.Widget(gluttertest.ByType[*ExcludeSemantics]()).(*ExcludeSemantics); !w.Excluding() {
			t.Error("the barrier is not excluded with barrierSemanticsDismissible false")
		}
	})
	t.Run("clips the semantics with the insets of the notifier", func(t *testing.T) {
		skipUnlessDebug(t)
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformAndroid))
		defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		ts := gluttertest.New(t)
		notifier := foundation.NewValueNotifier(painting.NewEdgeInsetsAll(10))
		defer notifier.Dispose()
		ts.PumpWidget(ltr(NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss", ClipDetailsNotifier: notifier})))
		notifier.SetValue(painting.NewEdgeInsetsAll(20))
		ts.Pump()
		if n := ts.Count(gluttertest.ByType[*ModalBarrier]()); n != 1 {
			t.Errorf("barriers = %d", n)
		}
	})
}

func TestModalBarrierDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewModalBarrier()))))
	if size := ts.Size(gluttertest.ByType[*ModalBarrier]()); size != (ui.Size{}) {
		t.Errorf("size = %v", size)
	}
}

func TestAnimatedModalBarrier(t *testing.T) {
	t.Run("calls onDismiss when tapped", func(t *testing.T) {
		calls := 0
		ts := gluttertest.New(t)
		key := foundation.NewUniqueKey()
		ts.PumpWidget(NewAnimatedModalBarrier(animation.NewAlwaysStoppedAnimation(foundation.Some(ui.NewColor(testRed))),
			AnimatedModalBarrierOpts{Key: key, OnDismiss: func() { calls++ }}))
		ts.Tap(gluttertest.ByKey(key))
		if calls != 1 {
			t.Errorf("onDismiss calls = %d, want 1", calls)
		}
	})
	t.Run("rebuilds with the color of the animation", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := newVsyncController(ts)
		defer controller.Dispose()
		tween := animation.NewColorTween(foundation.Some(ui.NewColor(0xFF000000)), foundation.Some(ui.NewColor(0xFFFFFFFF)))
		ts.PumpWidget(NewAnimatedModalBarrier(tween.Animate(controller)))
		colorOf := func() ui.Color { return ts.Widget(gluttertest.ByType[*ColoredBox]()).(*ColoredBox).Color() }
		if got := colorOf(); got != ui.NewColor(0xFF000000) {
			t.Errorf("color = %v", got)
		}
		controller.SetValue(1)
		ts.Pump()
		if got := colorOf(); got != ui.NewColor(0xFFFFFFFF) {
			t.Errorf("color at 1 = %v", got)
		}
	})
	t.Run("does not crash at zero area", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewAnimatedModalBarrier(animation.NewAlwaysStoppedAnimation(foundation.Some(ui.NewColor(testRed))))))))
		if size := ts.Size(gluttertest.ByType[*AnimatedModalBarrier]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	})
}

// rootSemanticsNode is the root of the semantics tree of the view.
func rootSemanticsNode(ts *gluttertest.Tester) *semantics.SemanticsNode {
	return ts.Binding.Renderer().RenderViews()[0].Owner().SemanticsOwner().RootSemanticsNode()
}

func TestModalBarrierSemanticsTree(t *testing.T) {
	skipUnlessDebug(t) // the semantics nodes are read from the render objects
	size := ui.Size{Width: 800, Height: 600}
	t.Run("an undismissible barrier is hidden in the semantics tree", func(t *testing.T) {
		for name, barrier := range map[string]Widget{
			"ModalBarrier":         NewModalBarrier(ModalBarrierOpts{Dismissible: foundation.Some(false)}),
			"AnimatedModalBarrier": NewAnimatedModalBarrier(animation.NewAlwaysStoppedAnimation(foundation.Some(ui.NewColor(testRed))), AnimatedModalBarrierOpts{Dismissible: foundation.Some(false)}),
		} {
			ts := gluttertest.New(t, gluttertest.Opts{Size: size})
			ts.EnsureSemantics()
			ts.PumpWidget(barrier)
			root := rootSemanticsNode(ts)
			if root == nil || root.ChildrenCount() != 0 {
				t.Errorf("%s: the semantics tree has children", name)
			}
		}
	})
	dismissButton := func(t *testing.T, ts *gluttertest.Tester, rect ui.Rect) {
		t.Helper()
		root := rootSemanticsNode(ts)
		if root == nil || root.ChildrenCount() != 1 {
			t.Fatalf("the root has %d children, want 1", root.ChildrenCount())
		}
		var node *semantics.SemanticsNode
		root.VisitChildren(func(n *semantics.SemanticsNode) bool { node = n; return true })
		data := node.GetSemanticsData()
		if node.Label() != "Dismiss" || node.TextDirection() != ui.TextDirectionLTR || node.Rect() != rect ||
			!data.HasAction(ui.SemanticsActionTap) || !data.HasAction(ui.SemanticsActionDismiss) {
			t.Errorf("node = label %q, direction %v, rect %v, tap %v, dismiss %v; want the Dismiss button at %v",
				node.Label(), node.TextDirection(), node.Rect(), data.HasAction(ui.SemanticsActionTap), data.HasAction(ui.SemanticsActionDismiss), rect)
		}
	}
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS, foundation.TargetPlatformAndroid} {
		t.Run("a dismissible barrier includes a button on "+platform.String(), func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
			defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
			for name, barrier := range map[string]Widget{
				"ModalBarrier": NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss"}),
				"AnimatedModalBarrier": NewAnimatedModalBarrier(animation.NewAlwaysStoppedAnimation(foundation.Some(ui.NewColor(testRed))),
					AnimatedModalBarrierOpts{SemanticsLabel: "Dismiss"}),
			} {
				t.Run(name, func(t *testing.T) {
					ts := gluttertest.New(t, gluttertest.Opts{Size: size})
					ts.EnsureSemantics()
					ts.PumpWidget(ltr(barrier))
					dismissButton(t, ts, ui.NewRectFromLTWH(0, 0, 800, 600))
				})
			}
		})
		t.Run("the semantics clipper clips the rect in four directions on "+platform.String(), func(t *testing.T) {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(platform))
			defer foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
			ts := gluttertest.New(t, gluttertest.Opts{Size: size})
			ts.EnsureSemantics()
			notifier := foundation.NewValueNotifier(painting.NewEdgeInsetsFromLTRB(10, 20, 30, 40))
			defer notifier.Dispose()
			ts.PumpWidget(ltr(NewModalBarrier(ModalBarrierOpts{SemanticsLabel: "Dismiss", ClipDetailsNotifier: notifier})))
			dismissButton(t, ts, ui.NewRectFromLTRB(10, 20, 800-30, 600-40))
		})
	}
}
