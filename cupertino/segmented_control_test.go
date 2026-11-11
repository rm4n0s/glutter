package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/cupertino/{segmented_control,sliding_segmented_control}_test.dart.
// The cases that read the calls of the canvas with a `paints` matcher are goldens here
// (TestCupertinoSegmentedControlGoldens), and the cases of the semantics tester are not ported.

func segments(labels ...string) []cupertino.CupertinoSegment[string] {
	var out []cupertino.CupertinoSegment[string]
	for _, l := range labels {
		out = append(out, cupertino.CupertinoSegment[string]{Value: l, Child: widgets.NewText(l)})
	}
	return out
}

func text(s string) gluttertest.Finder { return gluttertest.ByText(s) }

func TestCupertinoSegmentedControl(t *testing.T) {
	control := gluttertest.ByType[*cupertino.CupertinoSegmentedControl[string]]()
	app := func(group *foundation.Opt[string], opts cupertino.CupertinoSegmentedControlOpts[string], labels ...string) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewStatefulBuilder(
			func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				opts.GroupValue = *group
				return widgets.NewCenter(cupertino.NewCupertinoSegmentedControl(func(v string) {
					setState(func() { *group = foundation.Some(v) })
				}, segments(labels...), opts))
			}))})
	}
	t.Run("a tap selects a segment, and the selected one stays", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.None[string]()
		ts.PumpWidget(app(&g, cupertino.CupertinoSegmentedControlOpts[string]{}, "A", "B", "C"))
		ts.Tap(text("B"))
		ts.PumpAndSettle()
		if g != foundation.Some("B") {
			t.Errorf("selected %v", g)
		}
		ts.Tap(text("B"))
		ts.PumpAndSettle()
		if g != foundation.Some("B") {
			t.Errorf("selected %v", g)
		}
		ts.Tap(text("C"))
		ts.PumpAndSettle()
		if g != foundation.Some("C") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("a disabled segment does not answer", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSegmentedControlOpts[string]{DisabledChildren: []string{"B"}}, "A", "B", "C"))
		ts.Tap(text("B"))
		ts.PumpAndSettle()
		if g != foundation.Some("A") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("the segments have the same width, and the height is at least 28", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSegmentedControlOpts[string]{}, "A", "BBBBBBBB", "C"))
		size := ts.Size(control)
		if size.Height < 28 {
			t.Errorf("height = %v", size.Height)
		}
		a, b := ts.Center(text("A")), ts.Center(text("BBBBBBBB"))
		c := ts.Center(text("C"))
		if d1, d2 := b.Dx-a.Dx, c.Dx-b.Dx; d1 < 0 || d2 < 0 || d1-d2 > 0.01 || d2-d1 > 0.01 {
			t.Errorf("centers %v %v %v are not evenly spaced", a, b, c)
		}
	})
	t.Run("a segment in RTL is laid out from the right", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewDirectionality(ui.TextDirectionRTL, cupertino.NewCupertinoPageScaffold(
			widgets.NewCenter(cupertino.NewCupertinoSegmentedControl(func(string) {}, segments("A", "B"), cupertino.CupertinoSegmentedControlOpts[string]{GroupValue: g}))))}))
		if a, b := ts.Center(text("A")), ts.Center(text("B")); a.Dx <= b.Dx {
			t.Errorf("A at %v is not right of B at %v", a.Dx, b.Dx)
		}
	})
	t.Run("the keyboard moves the selection of the group", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSegmentedControlOpts[string]{}, "A", "B", "C"))
		ts.Tap(text("A")) // gives the focus to the segment
		ts.PumpAndSettle()
		ts.SendKeyEvent(services.LogicalKeyboardKeyArrowRight)
		ts.PumpAndSettle()
		if g != foundation.Some("B") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("the errors of the arguments", func(t *testing.T) {
		skipUnlessDebug(t)
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSegmentedControl(func(string) {}, segments("A"))
		}), "NewCupertinoSegmentedControl[...].CupertinoSegmentedControlTooFewChildren")
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSegmentedControl(func(string) {}, segments("A", "A"))
		}), "NewCupertinoSegmentedControl[...].CupertinoSegmentedControlDuplicateValue")
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSegmentedControl(func(string) {}, segments("A", "B"), cupertino.CupertinoSegmentedControlOpts[string]{GroupValue: foundation.Some("Z")})
		}), "NewCupertinoSegmentedControl[...].CupertinoSegmentedControlGroupValueNotAChild")
	})
}

func TestCupertinoSlidingSegmentedControl(t *testing.T) {
	control := gluttertest.ByType[*cupertino.CupertinoSlidingSegmentedControl[string]]()
	app := func(group *foundation.Opt[string], opts cupertino.CupertinoSlidingSegmentedControlOpts[string], labels ...string) widgets.Widget {
		return testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewStatefulBuilder(
			func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				opts.GroupValue = *group
				return widgets.NewCenter(cupertino.NewCupertinoSlidingSegmentedControl(func(v string) {
					setState(func() { *group = foundation.Some(v) })
				}, segments(labels...), opts))
			}))})
	}
	t.Run("a tap selects a segment", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B", "C"))
		ts.Tap(text("C"))
		ts.PumpAndSettle()
		if g != foundation.Some("C") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("a drag of the thumb selects the segment it ends on", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B", "C"))
		from, to := ts.Center(text("A")), ts.Center(text("C"))
		gesture := ts.StartGesture(from)
		gesture.MoveBy(ui.Offset{Dx: (to.Dx - from.Dx) / 2})
		ts.Pump()
		gesture.MoveBy(ui.Offset{Dx: (to.Dx - from.Dx) / 2})
		ts.Pump()
		if g != foundation.Some("A") {
			t.Errorf("the value changed before the end of the drag: %v", g)
		}
		gesture.Up()
		ts.PumpAndSettle()
		if g != foundation.Some("C") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("a drag that started on another segment selects what it ends on", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B", "C"))
		gesture := ts.StartGesture(ts.Center(text("B")))
		ts.Pump()
		gesture.MoveBy(ui.Offset{Dx: 5})
		ts.Pump()
		gesture.Up()
		ts.PumpAndSettle()
		if g != foundation.Some("B") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("a disabled segment does not answer", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{DisabledChildren: []string{"B"}}, "A", "B", "C"))
		ts.Tap(text("B"))
		ts.PumpAndSettle()
		if g != foundation.Some("A") {
			t.Errorf("selected %v", g)
		}
		ts.Tap(text("C"))
		ts.PumpAndSettle()
		if g != foundation.Some("C") {
			t.Errorf("selected %v", g)
		}
	})
	t.Run("a momentary control keeps no thumb but calls back", func(t *testing.T) {
		ts := gluttertest.New(t)
		var got []string
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
			cupertino.NewCupertinoSlidingSegmentedControl(func(v string) { got = append(got, v) }, segments("A", "B"),
				cupertino.CupertinoSlidingSegmentedControlOpts[string]{IsMomentary: true})))}))
		ts.Tap(text("A"))
		ts.PumpAndSettle()
		ts.Tap(text("A"))
		ts.PumpAndSettle()
		if len(got) != 2 {
			t.Errorf("callbacks = %v, want two", got)
		}
	})
	t.Run("the widths are the same, or follow the content", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "BBBBBBBBBB"))
		same := ts.Center(text("BBBBBBBBBB")).Dx - ts.Center(text("A")).Dx
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{ProportionalWidth: true}, "A", "BBBBBBBBBB"))
		proportional := ts.Center(text("BBBBBBBBBB")).Dx - ts.Center(text("A")).Dx
		if proportional >= same {
			t.Errorf("distance with proportional widths %v is not less than with the same widths %v", proportional, same)
		}
	})
	t.Run("a segment in RTL is laid out from the right", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(testApp(cupertino.CupertinoAppOpts{Home: widgets.NewDirectionality(ui.TextDirectionRTL, cupertino.NewCupertinoPageScaffold(widgets.NewCenter(
			cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("A", "B"), cupertino.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: foundation.Some("A")}))))}))
		if a, b := ts.Center(text("A")), ts.Center(text("B")); a.Dx <= b.Dx {
			t.Errorf("A at %v is not right of B at %v", a.Dx, b.Dx)
		}
		ts.Tap(text("B"))
		ts.PumpAndSettle()
	})
	t.Run("the size is at least 28 high", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B"))
		if h := ts.Size(control).Height; h < 28 {
			t.Errorf("height = %v", h)
		}
	})
	t.Run("the group value can change from outside", func(t *testing.T) {
		ts := gluttertest.New(t)
		g := foundation.Some("A")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B"))
		g = foundation.Some("B")
		ts.PumpWidget(app(&g, cupertino.CupertinoSlidingSegmentedControlOpts[string]{}, "A", "B"))
		ts.PumpAndSettle()
	})
	t.Run("the errors of the arguments", func(t *testing.T) {
		skipUnlessDebug(t)
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("A"))
		}), "NewCupertinoSlidingSegmentedControl[...].CupertinoSlidingSegmentedControlTooFewChildren")
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("A", "A"))
		}), "NewCupertinoSlidingSegmentedControl[...].CupertinoSlidingSegmentedControlDuplicateValue")
		checkErrorTag(t, recoverError(func() {
			cupertino.NewCupertinoSlidingSegmentedControl(func(string) {}, segments("A", "B"),
				cupertino.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: foundation.Some("Z")})
		}), "NewCupertinoSlidingSegmentedControl[...].CupertinoSlidingSegmentedControlGroupValueNotAChild")
	})
}
