package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/sliver_prototype_item_extent_test.dart.

// prototypeTestItem is TestItem of the Dart test: a container with the text of
// the item; its key is the one of the item.
func prototypeTestItem(item int, width, height foundation.Opt[float64], key foundation.Key) Widget {
	if key == nil {
		key = foundation.NewValueKey(item)
	}
	return NewContainer(NewText(fmt.Sprintf("Item %d", item)), ContainerOpts{Key: key, Width: width, Height: height, Alignment: painting.AlignmentCenter})
}

func prototypeFrame(count int, width, height float64, axis painting.Axis, prototypeKey foundation.Key) Widget {
	opt := func(v float64) foundation.Opt[float64] {
		if v == 0 {
			return foundation.None[float64]()
		}
		return foundation.Some(v)
	}
	if axis == painting.AxisUnspecified {
		axis = painting.AxisVertical
	}
	return ltr(NewCustomScrollView([]Widget{NewSliverPrototypeExtentList(
		NewSliverChildBuilderDelegate(func(_ BuildContext, i int) Widget {
			return prototypeTestItem(i, foundation.None[float64](), foundation.None[float64](), nil)
		}, SliverChildBuilderDelegateOpts{ChildCount: foundation.Some(count)}),
		prototypeTestItem(-1, opt(width), opt(height), prototypeKey))}, CustomScrollViewOpts{ScrollDirection: axis}))
}

func TestSliverPrototypeExtentListBuilder(t *testing.T) {
	for _, count := range []int{20, 8} {
		t.Run(fmt.Sprint(count, " items"), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverPrototypeExtentListBuilder(func(_ BuildContext, i int) Widget {
				return prototypeTestItem(i, foundation.None[float64](), foundation.None[float64](), nil)
			}, prototypeTestItem(-1, foundation.None[float64](), foundation.Some(100.0), nil), SliverBuilderOpts{ItemCount: foundation.Some(count)})})))
			// The viewport is 600 pixels high, the lazily created items are 100 pixels high.
			for i := 0; i < 6; i++ {
				item := gluttertest.ByKey(foundation.NewValueKey(i))
				if ts.TopLeft(item).Dy != float64(i*100) || ts.Size(item).Height != 100 {
					t.Errorf("item %d at %v, size %v", i, ts.TopLeft(item), ts.Size(item))
				}
			}
			for i := 7; i < count; i++ {
				if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 0 {
					t.Errorf("item %d is built", i)
				}
			}
		})
	}
}

func TestSliverPrototypeExtentListScrollingBasics(t *testing.T) {
	t.Run("vertical", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(prototypeFrame(20, 0, 100, painting.AxisVertical, nil))
		for i := 0; i < 6; i++ {
			item := gluttertest.ByKey(foundation.NewValueKey(i))
			if ts.TopLeft(item).Dy != float64(i*100) || ts.Size(item).Height != 100 {
				t.Errorf("item %d at %v, size %v", i, ts.TopLeft(item), ts.Size(item))
			}
		}
		for i := 7; i < 20; i++ {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 0 {
				t.Errorf("item %d is built", i)
			}
		}
		ts.Fling(gluttertest.ByText("Item 2"), ui.Offset{Dy: -200}, 5000)
		ts.PumpAndSettle()
		for i := 19; i >= 14; i-- {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 1 {
				t.Errorf("item %d is missing", i)
			}
		}
		for i := 13; i >= 0; i-- {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 0 {
				t.Errorf("item %d is built", i)
			}
		}
	})
	t.Run("horizontal", func(t *testing.T) {
		useAndroid(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(prototypeFrame(20, 100, 0, painting.AxisHorizontal, nil))
		// The viewport is 800 pixels wide, the lazily created items are 100 pixels wide.
		for i := 0; i < 8; i++ {
			item := gluttertest.ByKey(foundation.NewValueKey(i))
			if ts.TopLeft(item).Dx != float64(i*100) || ts.Size(item).Width != 100 {
				t.Errorf("item %d at %v, size %v", i, ts.TopLeft(item), ts.Size(item))
			}
		}
		for i := 9; i < 20; i++ {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 0 {
				t.Errorf("item %d is built", i)
			}
		}
		ts.Fling(gluttertest.ByText("Item 3"), ui.Offset{Dx: -200}, 5000)
		ts.PumpAndSettle()
		for i := 19; i >= 12; i-- {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 1 {
				t.Errorf("item %d is missing", i)
			}
		}
		for i := 11; i >= 0; i-- {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 0 {
				t.Errorf("item %d is built", i)
			}
		}
	})
}

func TestSliverPrototypeExtentListChangingThePrototypeItem(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	visible := func(first, last int) {
		t.Helper()
		for i := 0; i < 10; i++ {
			want := 0
			if i >= first && i < last {
				want = 1
			}
			if got := ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))); got != want {
				t.Errorf("item %d: count %d, want %d", i, got, want)
			}
		}
	}
	ts.PumpWidget(prototypeFrame(10, 0, 60, painting.AxisVertical, nil))
	visible(0, 10) // 600 pixels high, 60 pixels per item
	ts.PumpWidget(prototypeFrame(10, 0, 120, painting.AxisVertical, nil))
	visible(0, 5)
	ts.PumpWidget(prototypeFrame(10, 0, 60, painting.AxisVertical, nil))
	visible(0, 10)
}

func TestSliverPrototypeExtentListFirstItemIsAlsoThePrototype(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	items := make([]Widget, 10)
	for i := range items {
		height := foundation.None[float64]()
		if i == 0 {
			height = foundation.Some(60.0)
		}
		items[i] = NewContainer(NewText(fmt.Sprintf("Item %d", i)), ContainerOpts{Key: foundation.NewValueKey(i), Height: height, Alignment: painting.AlignmentCenter})
	}
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewSliverPrototypeExtentList(
		NewSliverChildBuilderDelegate(func(_ BuildContext, i int) Widget { return items[i] }, SliverChildBuilderDelegateOpts{ChildCount: foundation.Some(10)}),
		items[0])})))
	// Item 0 exists in the list and as the prototype item.
	if got := ts.Count(gluttertest.ByText("Item 0", gluttertest.FindOpts{IncludeOffstage: true})); got != 2 {
		t.Errorf("Item 0 is built %d times, want 2", got)
	}
	for i := 1; i < 10; i++ {
		if ts.Count(gluttertest.ByText(fmt.Sprintf("Item %d", i))) != 1 {
			t.Errorf("item %d is missing", i)
		}
	}
}

func TestSliverPrototypeExtentListPrototypePaintTransformIsZero(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/67117
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[State]()
	ts.PumpWidget(prototypeFrame(20, 0, 100, painting.AxisVertical, key))
	scrollView := ts.RenderObject(gluttertest.ByType[*CustomScrollView]())
	prototype := key.CurrentContext().FindRenderObject()
	if !prototype.GetTransformTo(scrollView).Equals(ui.NewMatrix4Zero()) {
		t.Errorf("the transform of the prototype is %v", prototype.GetTransformTo(scrollView))
	}
}
