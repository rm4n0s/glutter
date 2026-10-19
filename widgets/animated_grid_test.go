package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/animated_grid_test.dart.

func spacedGridDelegate() rendering.SliverGridDelegate {
	return rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100, rendering.SliverGridDelegateWithMaxCrossAxisExtentOpts{MainAxisSpacing: 10, CrossAxisSpacing: 10})
}

func TestSliverAnimatedGridBuilderRespectsFindChildIndexCallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	finderCalled := false
	itemCount := 7
	var stateSetter StateSetter
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		stateSetter = setState
		return NewCustomScrollView([]Widget{NewSliverAnimatedGrid(func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
			return NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(fmt.Sprint(i)), Height: foundation.Some(2000.0)})
		}, spacedGridDelegate(), SliverAnimatedListOpts{InitialItemCount: itemCount, FindChildIndexCallback: func(foundation.Key) foundation.Opt[int] {
			finderCalled = true
			return foundation.None[int]()
		}})})
	})))
	if finderCalled {
		t.Fatal("the finder was called on the first build")
	}
	stateSetter(func() { itemCount = 77 })
	ts.Pump()
	if !finderCalled {
		t.Error("the finder was not called")
	}
}

func TestAnimatedGrid(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	listKey := NewGlobalKey[*AnimatedGridState]()
	builder := func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
		return itemBox(fmt.Sprintf("item %d", i))
	}
	ts.PumpWidget(ltr(NewAnimatedGrid(builder, spacedGridDelegate(), AnimatedScrollViewOpts{Key: listKey, InitialItemCount: 2})))
	if ts.Count(gluttertest.ByType[*SliverAnimatedGrid]()) != 1 {
		t.Fatal("no SliverAnimatedGrid")
	}
	grid, _ := listKey.CurrentState()
	grid.InsertItem(0)
	ts.Pump()
	expectTexts(t, ts, []string{"item 2"}, nil)
	grid.RemoveItem(2, removedItem("removing item"), AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	expectTexts(t, ts, []string{"removing item"}, []string{"item 2"})
	ts.PumpAndSettle()
	expectTexts(t, ts, nil, []string{"removing item"})
	grid.InsertAllItems(0, 2)
	ts.Pump()
	expectTexts(t, ts, []string{"item 2", "item 3"}, nil)
	grid.RemoveAllItems(removedItem("removing item"), AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	if countTexts(ts, "removing item") == 0 {
		t.Error("no removing item")
	}
	expectTexts(t, ts, nil, []string{"item 0", "item 1", "item 2", "item 3"})
	ts.PumpAndSettle()
	expectTexts(t, ts, nil, []string{"removing item"})
}

func TestSliverAnimatedGridInsertScalesTheNewItems(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	listKey := NewGlobalKey[*SliverAnimatedGridState]()
	offstage := gluttertest.FindOpts{IncludeOffstage: true}
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedGrid(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
		return NewScaleTransition(a, itemBox(fmt.Sprintf("item %d", i)), ScaleTransitionOpts{Key: foundation.NewValueKey(i)})
	}, rendering.NewSliverGridDelegateWithMaxCrossAxisExtent(100), SliverAnimatedListOpts{Key: listKey})})))
	scale := func(i int) float64 {
		return ts.Widget(gluttertest.ByKey(foundation.NewValueKey(i), offstage)).(*ScaleTransition).Scale().Value()
	}
	left := func(i int) float64 { return ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(i), offstage)).Dx }
	right := func(i int) float64 { return ts.BottomRight(gluttertest.ByKey(foundation.NewValueKey(i), offstage)).Dx }
	grid, _ := listKey.CurrentState()
	grid.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	if scale(0) != 0 {
		t.Errorf("scale = %v, want 0", scale(0))
	}
	ts.Pump(50 * time.Millisecond)
	if scale(0) != 0.5 {
		t.Errorf("scale = %v, want 0.5", scale(0))
	}
	ts.Pump(50 * time.Millisecond)
	if scale(0) != 1 || left(0) != 0 || right(0) != 100 {
		t.Errorf("item 0: scale %v at %v..%v", scale(0), left(0), right(0))
	}
	grid.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	grid.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	if scale(0) != 0 || scale(1) != 0 || scale(2) != 1 {
		t.Errorf("scales = %v %v %v", scale(0), scale(1), scale(2))
	}
	ts.Pump(50 * time.Millisecond)
	if scale(0) != 0.5 || scale(1) != 0.5 || scale(2) != 1 {
		t.Errorf("scales = %v %v %v", scale(0), scale(1), scale(2))
	}
	ts.Pump(50 * time.Millisecond)
	for i := 0; i < 3; i++ {
		if scale(i) != 1 || left(i) != float64(100*i) || right(i) != float64(100*(i+1)) {
			t.Errorf("item %d: scale %v at %v..%v", i, scale(i), left(i), right(i))
		}
	}
}
