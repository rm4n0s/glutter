// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scrollable_helpers_test.dart. The
// case of the Scrollable without a notification context is not ported (a
// ScrollableState can not be subclassed), and the deprecated clipBehavior has no
// counterpart;
// the autoscroller cases build their list without ListView.builder (step 9.3).

func TestScrollableDetailsCopyWithAndEqual(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	details := NewScrollableDetails(painting.AxisDirectionDown, ScrollableDetailsOpts{
		Controller: controller, Physics: NewAlwaysScrollableScrollPhysics(nil), DecorationClipBehavior: foundation.Some(ui.ClipHardEdge)})
	if copied := details.CopyWith(ScrollableDetailsCopy{}); !details.Equal(copied) {
		t.Fatalf("a copy with no changes differs: %v vs %v", details, copied)
	}
	clamping := NewClampingScrollPhysics(nil)
	copied := details.CopyWith(ScrollableDetailsCopy{Direction: painting.AxisDirectionLeft, Physics: clamping,
		DecorationClipBehavior: foundation.Some(ui.ClipNone)})
	want := NewScrollableDetails(painting.AxisDirectionLeft, ScrollableDetailsOpts{
		Controller: controller, Physics: clamping, DecorationClipBehavior: foundation.Some(ui.ClipNone)})
	if !copied.Equal(want) {
		t.Errorf("copy = %v, want %v", copied, want)
	}
	if copied.Equal(details) {
		t.Error("the changed copy equals the original")
	}
}

func TestScrollableDetailsString(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	bare := NewScrollableDetails(painting.AxisDirectionRight)
	if got, want := bare.String(), "ScrollableDetails(axisDirection: AxisDirection.right)"; got != want {
		t.Errorf("bare = %q, want %q", got, want)
	}
	full := NewScrollableDetails(painting.AxisDirectionDown, ScrollableDetailsOpts{
		Controller: controller, Physics: NewAlwaysScrollableScrollPhysics(nil), DecorationClipBehavior: foundation.Some(ui.ClipHardEdge)})
	for _, part := range []string{"ScrollableDetails(axisDirection: AxisDirection.down, scroll controller: ScrollController", "(no clients), ",
		"scroll physics: AlwaysScrollableScrollPhysics, decorationClipBehavior: Clip.hardEdge)"} {
		if !stringsContain(full.String(), part) {
			t.Errorf("%q does not contain %q", full.String(), part)
		}
	}
}

func TestScrollableDetailsConstructors(t *testing.T) {
	tests := []struct {
		name    string
		details ScrollableDetails
		want    painting.AxisDirection
	}{
		{"vertical", NewScrollableDetailsVertical(), painting.AxisDirectionDown},
		{"vertical reversed", NewScrollableDetailsVertical(ScrollableDetailsOpts{Reverse: true}), painting.AxisDirectionUp},
		{"horizontal", NewScrollableDetailsHorizontal(), painting.AxisDirectionRight},
		{"horizontal reversed", NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Reverse: true}), painting.AxisDirectionLeft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.details.Direction != tt.want {
				t.Errorf("direction = %v, want %v", tt.details.Direction, tt.want)
			}
		})
	}
}

func TestScrollActionDirectionalIncrement(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(50, 100))))
	state := scrollableOf(ts)
	tests := []struct {
		name   string
		intent ScrollIntent
		want   float64
	}{
		{"line down", ScrollIntent{Direction: painting.AxisDirectionDown}, 50},
		{"line up", ScrollIntent{Direction: painting.AxisDirectionUp}, -50},
		{"page down", ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}, 0.8 * 600},
		{"page up", ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}, -0.8 * 600},
		{"the other axis", ScrollIntent{Direction: painting.AxisDirectionRight}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScrollActionGetDirectionalIncrement(state, tt.intent); got != tt.want {
				t.Errorf("increment = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScrollActionUsesTheIncrementCalculator(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var seen []ScrollIncrementType
	ts.PumpWidget(ltr(testList(items(50, 100), listOpts{ScrollableOpts: ScrollableOpts{
		IncrementCalculator: func(d ScrollIncrementDetails) float64 {
			seen = append(seen, d.Type)
			return 7
		}}})))
	state := scrollableOf(ts)
	if got := ScrollActionGetDirectionalIncrement(state, ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}); got != -7 {
		t.Errorf("increment = %v, want -7", got)
	}
	if len(seen) != 1 || seen[0] != ScrollIncrementTypePage {
		t.Errorf("the calculator saw %v", seen)
	}
}

func TestEdgeDraggingAutoScrollerScrollsNearTheEdges(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(testList(items(50, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	scrolled := 0
	scroller := NewEdgeDraggingAutoScroller(scrollableOf(ts), 1, func() { scrolled++ })

	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, 100, 100)) // at the start: nothing to scroll back to
	ts.Pump()
	if scroller.Scrolling() || controller.Position().Pixels() != 0 {
		t.Fatalf("scrolling = %v, pixels = %v at the start", scroller.Scrolling(), controller.Position().Pixels())
	}

	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 520, 100, 100)) // past the bottom edge by 20
	if !scroller.Scrolling() {
		t.Fatal("not scrolling below the bottom edge")
	}
	ts.Pump()            // the animation starts
	ts.Pump(time.Second) // and ends
	if got := controller.Position().Pixels(); got != 20 {
		t.Errorf("pixels = %v, want 20 (the overdrag of one animation)", got)
	}
	ts.Pump(10 * time.Millisecond) // the animation is done after its duration
	if scrolled == 0 {
		t.Error("onScrollViewScrolled was not called")
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle() // the current animation ends
	before := controller.Position().Pixels()
	ts.Pump(time.Second)
	if got := controller.Position().Pixels(); got != before {
		t.Errorf("pixels moved from %v to %v after the stop", before, got)
	}

	// Past the top edge it scrolls back.
	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, -10, 100, 100))
	ts.Pump()
	ts.Pump(time.Second)
	if got := controller.Position().Pixels(); got >= before {
		t.Errorf("pixels = %v, want less than %v", got, before)
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle()
}

func TestEdgeDraggingAutoScrollerHandlesDragTargetSizeWithTransformScale(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewTransformScale(
		NewSizedBoxSquare(foundation.Some(400.0), testList(items(20, 100))),
		TransformScaleOpts{Scale: foundation.Some(0.5)}))))
	ts.PumpAndSettle()
	state := scrollableOf(ts)
	scroller := NewEdgeDraggingAutoScroller(state, 1, nil)
	box := state.Context().FindRenderObject().(rendering.Box)
	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, box.Size().Width, box.Size().Height))
	ts.Pump()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle()
}

func TestEdgeDraggingAutoScrollerDragTargetTooLarge(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(50, 100))))
	scroller := NewEdgeDraggingAutoScroller(scrollableOf(ts), 1, nil)
	requireTag(t, "AutoScrollDragTargetTooLarge", func() {
		scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, 100, 5000))
	})
}

// modifierKey is the modifier of the default scroll shortcuts of the platform:
// Meta on Apple's, Control elsewhere (the test platform is Linux).
const modifierKey = services.LogicalKeyboardKeyControlLeft

func boxByKey(i int) foundation.Key { return foundation.NewValueKey(fmt.Sprintf("Box %d", i)) }

// boxRectOf is the rect of the box with the key, offstage ones included.
func boxRectOf(ts *gluttertest.Tester, key foundation.Key) ui.Rect {
	f := gluttertest.ByKey(key, gluttertest.FindOpts{IncludeOffstage: true})
	tl, br := ts.TopLeft(f), ts.BottomRight(f)
	return ui.NewRectFromLTRB(tl.Dx, tl.Dy, br.Dx, br.Dy)
}

func requireBoxRect(t *testing.T, ts *gluttertest.Tester, key foundation.Key, l, tp, r, b float64) {
	t.Helper()
	if got, want := boxRectOf(ts, key), ui.NewRectFromLTRB(l, tp, r, b); got != want {
		t.Fatalf("rect of %v = %v, want %v", key, got, want)
	}
}

// scrollKeys sends the key with the modifier of the platform held, then lets the
// scrolling settle.
func scrollKeys(ts *gluttertest.Tester, key services.LogicalKeyboardKey, withModifier bool) {
	if withModifier {
		ts.SendKeyDownEvent(modifierKey)
	}
	ts.SendKeyEvent(key)
	if withModifier {
		ts.SendKeyUpEvent(modifierKey)
	}
	ts.PumpAndSettle()
}

// boxSlivers are 20 SliverToBoxAdapters with a box each; the first (or the one
// chosen) has the focus.
func boxSlivers(horizontal bool, focusNode *FocusNode) []Widget {
	slivers := make([]Widget, 20)
	for i := range slivers {
		size := SizedBoxOpts{Key: boxByKey(i), Height: foundation.Some(50.0)}
		if horizontal {
			size = SizedBoxOpts{Key: boxByKey(i), Width: foundation.Some(50.0)}
		}
		focus := FocusOpts{Autofocus: focusNode == nil && i == 0}
		if focusNode != nil {
			focus.FocusNode = focusNode
		}
		slivers[i] = NewSliverToBoxAdapter(NewFocus(NewSizedBox(nil, size), focus))
	}
	return slivers
}

func TestKeyboardScrolling(t *testing.T) {
	t.Run("does not happen if the scroll physics are NeverScrollableScrollPhysics", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(boxSlivers(false, nil),
			CustomScrollViewOpts{Controller: controller, Physics: NewNeverScrollableScrollPhysics(nil)})}))
		ts.PumpAndSettle()
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 800, 50)
		for _, k := range []struct {
			key          services.LogicalKeyboardKey
			withModifier bool
		}{
			{services.LogicalKeyboardKeyArrowDown, true}, {services.LogicalKeyboardKeyArrowUp, true},
			{services.LogicalKeyboardKeyPageDown, false}, {services.LogicalKeyboardKeyPageUp, false},
		} {
			scrollKeys(ts, k.key, k.withModifier)
			if controller.Position().Pixels() != 0 {
				t.Fatalf("%v scrolled to %v", k.key, controller.Position().Pixels())
			}
			requireBoxRect(t, ts, boxByKey(0), 0, 0, 800, 50)
		}
	})

	t.Run("vertical scrollables are scrolled when activated via keyboard", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(boxSlivers(false, nil),
			CustomScrollViewOpts{Controller: controller})}))
		ts.PumpAndSettle()
		if controller.Position().Pixels() != 0 {
			t.Fatalf("pixels = %v, want 0", controller.Position().Pixels())
		}
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 800, 50)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowDown, true)
		requireBoxRect(t, ts, boxByKey(0), 0, -50, 800, 0)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowUp, true)
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 800, 50)
		scrollKeys(ts, services.LogicalKeyboardKeyPageDown, false)
		requireBoxRect(t, ts, boxByKey(0), 0, -400, 800, -350)
		scrollKeys(ts, services.LogicalKeyboardKeyPageUp, false)
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 800, 50)
	})

	t.Run("horizontal scrollables are scrolled when activated via keyboard", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(boxSlivers(true, nil),
			CustomScrollViewOpts{Controller: controller, ScrollDirection: painting.AxisHorizontal})}))
		ts.PumpAndSettle()
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 50, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowRight, true)
		requireBoxRect(t, ts, boxByKey(0), -50, 0, 0, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowLeft, true)
		requireBoxRect(t, ts, boxByKey(0), 0, 0, 50, 600)
	})

	t.Run("horizontal scrollables are scrolled the correct direction in RTL locales", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewDirectionality(ui.TextDirectionRTL, NewCustomScrollView(boxSlivers(true, nil),
			CustomScrollViewOpts{Controller: controller, ScrollDirection: painting.AxisHorizontal}))}))
		ts.PumpAndSettle()
		requireBoxRect(t, ts, boxByKey(0), 750, 0, 800, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowLeft, true)
		requireBoxRect(t, ts, boxByKey(0), 800, 0, 850, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowRight, true)
		requireBoxRect(t, ts, boxByKey(0), 750, 0, 800, 600)
	})

	t.Run("reversed vertical scrollables are scrolled when activated via keyboard", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		node := NewFocusNode(FocusNodeOpts{DebugLabel: "SizedBox"})
		defer node.Dispose()
		slivers := boxSlivers(false, node)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(slivers,
			CustomScrollViewOpts{Controller: controller, Reverse: true})}))
		node.RequestFocus()
		ts.PumpAndSettle()
		if controller.Position().Pixels() != 0 {
			t.Fatalf("pixels = %v, want 0", controller.Position().Pixels())
		}
		requireBoxRect(t, ts, boxByKey(0), 0, 550, 800, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowUp, true)
		requireBoxRect(t, ts, boxByKey(0), 0, 600, 800, 650)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowDown, true)
		requireBoxRect(t, ts, boxByKey(0), 0, 550, 800, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyPageUp, false)
		requireBoxRect(t, ts, boxByKey(0), 0, 950, 800, 1000)
		scrollKeys(ts, services.LogicalKeyboardKeyPageDown, false)
		requireBoxRect(t, ts, boxByKey(0), 0, 550, 800, 600)
	})

	t.Run("reversed horizontal scrollables are scrolled when activated via keyboard", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		node := NewFocusNode(FocusNodeOpts{DebugLabel: "SizedBox"})
		defer node.Dispose()
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(boxSlivers(true, node),
			CustomScrollViewOpts{Controller: controller, ScrollDirection: painting.AxisHorizontal, Reverse: true})}))
		node.RequestFocus()
		ts.PumpAndSettle()
		requireBoxRect(t, ts, boxByKey(0), 750, 0, 800, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowLeft, true)
		requireBoxRect(t, ts, boxByKey(0), 800, 0, 850, 600)
		scrollKeys(ts, services.LogicalKeyboardKeyArrowRight, true)
	})

	t.Run("custom scrollables with a center sliver are scrolled when activated via keyboard", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		centerKey := foundation.NewValueKey("Center")
		slivers := make([]Widget, 20)
		for i := range slivers {
			item := fmt.Sprintf("Item %d", i)
			opts := SliverToBoxAdapterOpts{}
			if i == 10 {
				opts.Key = centerKey
			}
			slivers[i] = NewSliverToBoxAdapter(NewFocus(NewContainer(NewText(item, TextOpts{TextDirection: ui.TextDirectionLTR}),
				ContainerOpts{Key: foundation.NewValueKey(item), Alignment: painting.AlignmentCenter, Height: foundation.Some(100.0)}),
				FocusOpts{Autofocus: i == 10}), opts)
		}
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewCustomScrollView(slivers,
			CustomScrollViewOpts{Controller: controller, Center: centerKey})}))
		ts.PumpAndSettle()
		item10 := foundation.NewValueKey("Item 10")
		if controller.Position().Pixels() != 0 {
			t.Fatalf("pixels = %v, want 0", controller.Position().Pixels())
		}
		requireBoxRect(t, ts, item10, 0, 0, 800, 100)
		for range 10 {
			scrollKeys(ts, services.LogicalKeyboardKeyArrowDown, true)
		}
		// Starts at #10 already, so it doesn't work out to 500 because it hits the bottom.
		if controller.Position().Pixels() != 400 {
			t.Fatalf("pixels = %v, want 400", controller.Position().Pixels())
		}
		requireBoxRect(t, ts, item10, 0, -400, 800, -300)
		for range 10 {
			scrollKeys(ts, services.LogicalKeyboardKeyArrowUp, true)
		}
		// Goes up two past the "center" where it started, so negative.
		if controller.Position().Pixels() != -100 {
			t.Fatalf("pixels = %v, want -100", controller.Position().Pixels())
		}
		requireBoxRect(t, ts, item10, 0, 100, 800, 200)
	})
}

func TestCanScrollUsingIntentsOnly(t *testing.T) {
	ts := gluttertest.New(t)
	page := func(text string) Widget {
		return NewSizedBox(NewText(text, TextOpts{TextDirection: ui.TextDirectionLTR}), SizedBoxOpts{Height: foundation.Some(600.0)})
	}
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewListView([]Widget{
		page("The cow as white as milk"), page("The cape as red as blood"), page("The hair as yellow as corn"),
	})}))
	counts := func(a, b, c int) {
		t.Helper()
		for text, want := range map[string]int{"The cow as white as milk": a, "The cape as red as blood": b, "The hair as yellow as corn": c} {
			if got := ts.Count(gluttertest.ByText(text)); got != want {
				t.Errorf("%q found %d times, want %d", text, got, want)
			}
		}
	}
	counts(1, 0, 0)
	pageDown := ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}
	ActionsInvoke(ts.Element(gluttertest.ByType[*SliverList]()), pageDown)
	ts.Pump()                        // start scroll
	ts.Pump(1000 * time.Millisecond) // end scroll
	counts(1, 1, 0)
	ActionsInvoke(ts.Element(gluttertest.ByType[*SliverList]()), pageDown)
	ts.Pump()
	ts.Pump(1000 * time.Millisecond)
	counts(0, 1, 1)
}

func TestScrollActionFailureTags(t *testing.T) {
	skipUnlessDebug(t)
	intent := ScrollIntent{Direction: painting.AxisDirectionDown}

	t.Run("ScrollActionContextMissing", func(t *testing.T) {
		requireTag(t, "ScrollActionContextMissing", func() { NewScrollAction().InvokeWithContext(intent, nil) })
	})

	t.Run("ScrollActionPositionsInvalid", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := NewScrollController()
		defer controller.Dispose()
		key := NewGlobalKey[State]()
		// No scrollable has the primary controller: it has no position.
		ts.PumpWidget(NewPrimaryScrollController(controller, NewContainer(nil, ContainerOpts{Key: key})))
		requireTag(t, "ScrollActionPositionsInvalid", func() { NewScrollAction().InvokeWithContext(intent, key.CurrentContext()) })
	})

	t.Run("ScrollActionNotLaidOut", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller := &noPixelsController{}
		InitScrollControllerBase(&controller.ScrollControllerBase, controller, ScrollControllerOpts{})
		defer controller.Dispose()
		key := NewGlobalKey[State]()
		// A scrollable whose viewport builder makes no viewport never gets pixels, and
		// the controller makes its position without any.
		ts.PumpWidget(ltr(NewScrollable(func(BuildContext, rendering.ViewportOffset) Widget {
			return NewContainer(nil, ContainerOpts{Key: key})
		}, ScrollableOpts{Controller: controller})))
		requireTag(t, "ScrollActionNotLaidOut", func() { NewScrollAction().InvokeWithContext(intent, key.CurrentContext()) })
	})
}

// noPixelsController is a ScrollController whose positions start without pixels.
type noPixelsController struct{ ScrollControllerBase }

func (c *noPixelsController) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, old ScrollPosition) ScrollPosition {
	return NewScrollPositionWithSingleContext(physics, context, ScrollPositionWithSingleContextOpts{NoInitialPixels: true, OldPosition: old})
}
