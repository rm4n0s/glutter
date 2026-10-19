package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/animated_list_test.dart.

func itemBox(text string) Widget {
	return NewSizedBox(NewCenter(NewText(text)), SizedBoxOpts{Height: foundation.Some(100.0)})
}

func removedItem(text string) AnimatedRemovedItemBuilder {
	return func(BuildContext, animation.Animation[float64]) Widget { return itemBox(text) }
}

func countTexts(ts *gluttertest.Tester, text string) int { return ts.Count(gluttertest.ByText(text)) }

func TestSliverAnimatedListBuilderRespectsFindChildIndexCallback(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	finderCalled := false
	itemCount := 7
	var stateSetter StateSetter
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		stateSetter = setState
		return NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
			return NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(fmt.Sprint(i)), Height: foundation.Some(2000.0)})
		}, SliverAnimatedListOpts{InitialItemCount: itemCount, FindChildIndexCallback: func(foundation.Key) foundation.Opt[int] {
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

func TestAnimatedList(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	listKey := NewGlobalKey[*AnimatedListState]()
	builder := func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
		return itemBox(fmt.Sprintf("item %d", i))
	}
	ts.PumpWidget(ltr(NewAnimatedList(builder, AnimatedScrollViewOpts{Key: listKey, InitialItemCount: 2})))
	if sliver := ts.Widget(gluttertest.ByType[*SliverAnimatedList]()).(*SliverAnimatedList); sliver == nil {
		t.Fatal("no SliverAnimatedList")
	}
	list, _ := listKey.CurrentState()
	list.InsertItem(0)
	ts.Pump()
	expectTexts(t, ts, []string{"item 2"}, nil)
	list.RemoveItem(2, removedItem("removing item"), AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	expectTexts(t, ts, []string{"removing item"}, []string{"item 2"})
	ts.PumpAndSettle()
	expectTexts(t, ts, nil, []string{"removing item"})

	// insertAllItems
	list.InsertAllItems(0, 2)
	ts.Pump()
	expectTexts(t, ts, []string{"item 2", "item 3"}, nil)

	// removeAllItems
	list.RemoveAllItems(removedItem("removing item"), AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump()
	if countTexts(ts, "removing item") == 0 {
		t.Error("no removing item")
	}
	expectTexts(t, ts, nil, []string{"item 0", "item 1", "item 2", "item 3"})
	ts.PumpAndSettle()
	expectTexts(t, ts, nil, []string{"removing item"})
}

func TestAnimatedListShouldSafelyExecuteRemoveAllItemsDuringLongRemovalOfOneItem(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	listKey := NewGlobalKey[*AnimatedListState]()
	builder := func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
		return itemBox(fmt.Sprintf("item %d", i))
	}
	ts.PumpWidget(ltr(NewAnimatedList(builder, AnimatedScrollViewOpts{Key: listKey, InitialItemCount: 2})))
	texts := func() int { return ts.Count(gluttertest.ByType[*Text]()) }
	if texts() != 2 {
		t.Fatalf("texts = %d, want 2", texts())
	}
	list, _ := listKey.CurrentState()
	// Insert 1 item (item 0, item 1, item 2).
	list.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(200 * time.Millisecond)})
	ts.Pump(50 * time.Millisecond)
	if texts() != 3 {
		t.Fatalf("texts = %d, want 3", texts())
	}
	expectTexts(t, ts, []string{"item 0", "item 1", "item 2"}, nil)
	// Remove item 2 (item 0, item 1, removing item 2).
	list.RemoveItem(2, removedItem("removing item 2"), AnimatedItemOpts{Duration: foundation.Some(200 * time.Millisecond)})
	ts.Pump(50 * time.Millisecond)
	if texts() != 3 {
		t.Fatalf("texts = %d, want 3", texts())
	}
	expectTexts(t, ts, []string{"item 0", "item 1", "removing item 2"}, []string{"item 2"})
	// removeAllItems (removing all items, removing all items, removing item 2).
	list.RemoveAllItems(removedItem("removing all items"), AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
	ts.Pump(50 * time.Millisecond)
	if texts() != 3 {
		t.Fatalf("texts = %d, want 3", texts())
	}
	if n := countTexts(ts, "removing all items"); n != 2 {
		t.Errorf("removing all items: %d, want 2", n)
	}
	if countTexts(ts, "removing item 2") == 0 {
		t.Error("removing item 2 vanished")
	}
	expectTexts(t, ts, nil, []string{"item 0", "item 1", "item 2"})
	ts.PumpAndSettle()
	if texts() != 0 {
		t.Errorf("texts = %d, want 0", texts())
	}
}

func sizeTransitionFor(key int, text string, a animation.Animation[float64]) Widget {
	return NewSizeTransition(a, itemBox(text), SizeTransitionOpts{Key: foundation.NewValueKey(key)})
}

func TestSliverAnimatedList(t *testing.T) {
	offstage := gluttertest.FindOpts{IncludeOffstage: true}
	item := func(i int) gluttertest.Finder { return gluttertest.ByKey(foundation.NewValueKey(i), offstage) }
	t.Run("initialItemCount", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		animations := map[int]animation.Animation[float64]{}
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
			animations[i] = a
			return itemBox(fmt.Sprintf("item %d", i))
		}, SliverAnimatedListOpts{InitialItemCount: 2})})))
		expectTexts(t, ts, []string{"item 0", "item 1"}, nil)
		for i := 0; i < 2; i++ {
			if animations[i] == nil || animations[i].Value() != 1 {
				t.Errorf("animation %d = %v", i, animations[i])
			}
		}
	})
	t.Run("insert", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		listKey := NewGlobalKey[*SliverAnimatedListState]()
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
			return sizeTransitionFor(i, fmt.Sprintf("item %d", i), a)
		}, SliverAnimatedListOpts{Key: listKey})})))
		height := func(i int) float64 { return ts.Size(item(i)).Height }
		top := func(i int) float64 { return ts.TopLeft(item(i)).Dy }
		bottom := func(i int) float64 { return ts.BottomRight(item(i)).Dy }
		want := func(label string, got, want float64) {
			t.Helper()
			if got != want {
				t.Errorf("%s = %v, want %v", label, got, want)
			}
		}
		list, _ := listKey.CurrentState()
		list.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		ts.Pump()
		want("height 0", height(0), 0)
		ts.Pump(50 * time.Millisecond)
		want("height 0", height(0), 50)
		ts.Pump(50 * time.Millisecond)
		want("height 0", height(0), 100)
		expectTexts(t, ts, []string{"item 0"}, nil)
		want("top 0", top(0), 0)
		want("bottom 0", bottom(0), 100)

		list.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		list.InsertItem(0, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		ts.Pump()
		want("height 0", height(0), 0)
		want("height 1", height(1), 0)
		want("height 2", height(2), 100)
		ts.Pump(50 * time.Millisecond)
		want("height 0", height(0), 50)
		want("height 1", height(1), 50)
		want("height 2", height(2), 100)
		ts.Pump(50 * time.Millisecond)
		for i := 0; i < 3; i++ {
			want(fmt.Sprintf("height %d", i), height(i), 100)
			want(fmt.Sprintf("top %d", i), top(i), float64(100*i))
			want(fmt.Sprintf("bottom %d", i), bottom(i), float64(100*(i+1)))
		}
		expectTexts(t, ts, []string{"item 0", "item 1", "item 2"}, nil)
	})
	t.Run("insertAll", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		listKey := NewGlobalKey[*SliverAnimatedListState]()
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
			return sizeTransitionFor(i, fmt.Sprintf("item %d", i), a)
		}, SliverAnimatedListOpts{Key: listKey})})))
		list, _ := listKey.CurrentState()
		list.InsertAllItems(0, 2, AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		ts.Pump()
		for i := 0; i < 2; i++ {
			if got := ts.Size(item(i)).Height; got != 0 {
				t.Errorf("height %d = %v, want 0", i, got)
			}
		}
		ts.Pump(50 * time.Millisecond)
		for i := 0; i < 2; i++ {
			if got := ts.Size(item(i)).Height; got != 50 {
				t.Errorf("height %d = %v, want 50", i, got)
			}
		}
		ts.Pump(50 * time.Millisecond)
		for i := 0; i < 2; i++ {
			if ts.Size(item(i)).Height != 100 || ts.TopLeft(item(i)).Dy != float64(100*i) {
				t.Errorf("item %d: height %v top %v", i, ts.Size(item(i)).Height, ts.TopLeft(item(i)).Dy)
			}
		}
	})
	t.Run("remove", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		listKey := NewGlobalKey[*SliverAnimatedListState]()
		items := []int{0, 1, 2}
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
			return sizeTransitionFor(items[i], fmt.Sprintf("item %d", items[i]), a)
		}, SliverAnimatedListOpts{Key: listKey, InitialItemCount: 3})})))
		top := func(i int) float64 { return ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(i))).Dy }
		bottom := func(i int) float64 { return ts.BottomRight(gluttertest.ByKey(foundation.NewValueKey(i))).Dy }
		expectTexts(t, ts, []string{"item 0", "item 1", "item 2"}, nil)
		items = items[1:]
		list, _ := listKey.CurrentState()
		list.RemoveItem(0, func(_ BuildContext, a animation.Animation[float64]) Widget { return sizeTransitionFor(0, "item 0", a) },
			AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		for i, want := range [][2]float64{{0, 100}, {100, 200}, {200, 300}} {
			if top(i) != want[0] || bottom(i) != want[1] {
				t.Errorf("item %d at %v..%v, want %v", i, top(i), bottom(i), want)
			}
		}
		ts.Pump()
		ts.Pump(50 * time.Millisecond)
		for i, want := range [][2]float64{{0, 50}, {50, 150}, {150, 250}} {
			if top(i) != want[0] || bottom(i) != want[1] {
				t.Errorf("mid-animation item %d at %v..%v, want %v", i, top(i), bottom(i), want)
			}
		}
		ts.PumpAndSettle()
		for i, want := range [][2]float64{{0, 100}, {100, 200}} {
			if top(i+1) != want[0] || bottom(i+1) != want[1] {
				t.Errorf("settled item %d at %v..%v, want %v", i+1, top(i+1), bottom(i+1), want)
			}
		}
	})
	t.Run("removeAll", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		listKey := NewGlobalKey[*SliverAnimatedListState]()
		items := []int{0, 1, 2}
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverAnimatedList(func(_ BuildContext, i int, a animation.Animation[float64]) Widget {
			return sizeTransitionFor(items[i], fmt.Sprintf("item %d", items[i]), a)
		}, SliverAnimatedListOpts{Key: listKey, InitialItemCount: 3})})))
		expectTexts(t, ts, []string{"item 0", "item 1", "item 2"}, nil)
		items = nil
		list, _ := listKey.CurrentState()
		list.RemoveAllItems(func(_ BuildContext, a animation.Animation[float64]) Widget { return sizeTransitionFor(0, "item 0", a) },
			AnimatedItemOpts{Duration: foundation.Some(100 * time.Millisecond)})
		ts.PumpAndSettle()
		expectTexts(t, ts, nil, []string{"item 0", "item 1", "item 2"})
	})
	t.Run("works in combination with other slivers", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		listKey := NewGlobalKey[*SliverAnimatedListState]()
		ts.PumpWidget(ltr(NewCustomScrollView([]Widget{
			NewSliverListList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)}), NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(100.0)})}),
			NewSliverAnimatedList(func(_ BuildContext, i int, _ animation.Animation[float64]) Widget {
				return NewSizedBox(NewText(fmt.Sprintf("item %d", i)), SizedBoxOpts{Height: foundation.Some(100.0)})
			}, SliverAnimatedListOpts{Key: listKey, InitialItemCount: 3}),
		})))
		topOf := func(text string) float64 { return ts.TopLeft(gluttertest.ByText(text)).Dy }
		if topOf("item 0") != 200 || topOf("item 1") != 300 {
			t.Errorf("tops = %v %v", topOf("item 0"), topOf("item 1"))
		}
		list, _ := listKey.CurrentState()
		list.InsertItem(3)
		ts.PumpAndSettle()
		if topOf("item 3") != 500 {
			t.Errorf("item 3 top = %v", topOf("item 3"))
		}
		list.RemoveItem(0, func(_ BuildContext, a animation.Animation[float64]) Widget {
			return NewSizeTransition(a, NewSizedBox(NewText("removing"), SizedBoxOpts{Height: foundation.Some(100.0)}), SizeTransitionOpts{Key: foundation.NewObjectKey("removing")})
		}, AnimatedItemOpts{Duration: foundation.Some(time.Second)})
		ts.Pump()
		expectTexts(t, ts, nil, []string{"item 3"})
		ts.Pump(500 * time.Millisecond)
		if got := ts.Size(gluttertest.ByKey(foundation.NewObjectKey("removing"))).Height; got != 50 {
			t.Errorf("removing height = %v, want 50", got)
		}
		if topOf("item 0") != 250 {
			t.Errorf("item 0 top = %v, want 250", topOf("item 0"))
		}
		ts.PumpAndSettle()
		expectTexts(t, ts, nil, []string{"removing"})
		if topOf("item 0") != 200 {
			t.Errorf("item 0 top = %v, want 200", topOf("item 0"))
		}
	})
}

func TestAnimatedListStateOf(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var found *AnimatedListState
	ts.PumpWidget(ltr(NewAnimatedList(func(bc BuildContext, i int, _ animation.Animation[float64]) Widget {
		return NewBuilder(func(bc BuildContext) Widget {
			found, _ = AnimatedListMaybeOf(bc)
			return itemBox("x")
		})
	}, AnimatedScrollViewOpts{InitialItemCount: 1})))
	if found == nil {
		t.Error("AnimatedListMaybeOf found nothing")
	}
	ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
		if foundation.DebugMode {
			requireTag(t, "AnimatedListMissing", func() { AnimatedListOf(bc) })
		}
		return NewSizedBox(nil)
	})))
}

var _ = ui.Offset{}
