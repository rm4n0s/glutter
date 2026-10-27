package widgets_test

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// The tests of the EstimateItemOffsets option (SPEC §11.4, §12.10): 3,000 items
// of varied heights in a list of 600 pixels, and jumps far from the items that
// the list has.

const (
	estimateItems    = 3000
	estimateViewport = 600.0
)

// estimateHeight is the height of item i: from 30 to 119 pixels, the same on
// every run.
func estimateHeight(i int) float64 { return 30 + float64((i*37)%90) }

type estimateList struct {
	t       testing.TB
	ts      *gluttertest.Tester
	ctl     ScrollController
	built   int
	reverse bool
	setIDs  StateSetter
	// ids is the id of the item at each index; the item with id n is as tall as
	// estimateHeight(n).
	ids []int
}

type estimateConfig struct {
	estimate bool
	keep     bool
	reverse  bool
	// discard drops the scenes that the frames make, for the tests of the memory (leak_test.go).
	discard bool
}

func newEstimateList(t testing.TB, cfg estimateConfig) *estimateList {
	t.Helper()
	l := &estimateList{t: t, ts: gluttertest.New(t, gluttertest.Opts{DiscardScenes: cfg.discard}), ctl: NewScrollController(), reverse: cfg.reverse}
	for i := 0; i < estimateItems; i++ {
		l.ids = append(l.ids, i)
	}
	l.ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		l.setIDs = setState
		ids := l.ids
		return NewListViewBuilder(func(_ BuildContext, i int) Widget {
			if i >= len(ids) {
				return nil
			}
			l.built++
			return NewSizedBox(NewText(fmt.Sprintf("item %d", ids[i])),
				SizedBoxOpts{Key: foundation.NewValueKey(ids[i]), Height: foundation.Some(estimateHeight(ids[i]))})
		}, ListViewOpts{
			Controller: l.ctl, ItemCount: foundation.Some(len(ids)), Reverse: cfg.reverse,
			EstimateItemOffsets: cfg.estimate, MaintainVisibleContentPosition: cfg.keep,
			FindChildIndexCallback: func(key foundation.Key) foundation.Opt[int] {
				vk, ok := key.(foundation.ValueKey[int])
				if !ok {
					return foundation.None[int]()
				}
				for i, id := range ids {
					if id == vk.Value() {
						return foundation.Some(i)
					}
				}
				return foundation.None[int]()
			},
		})
	})))
	return l
}

// shown are the ids of the items that are built now, in order.
func (l *estimateList) shown() []int {
	var ids []int
	for _, e := range gluttertest.ByType[*SizedBox]().Find(l.ts) {
		if vk, ok := e.Widget().Key().(foundation.ValueKey[int]); ok {
			ids = append(ids, vk.Value())
		}
	}
	sort.Ints(ids)
	return ids
}

// checkContiguous fails if two built items overlap or leave a gap between them,
// and returns the top of the first one.
func (l *estimateList) checkContiguous() float64 {
	l.t.Helper()
	ids := l.shown()
	if len(ids) == 0 {
		l.t.Fatal("no item is built")
	}
	// The items are in the list order, which is the order of their ids here.
	top := func(id int) float64 {
		return l.ts.TopLeft(gluttertest.ByText(fmt.Sprintf("item %d", id))).Dy
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[i-1]+1 {
			l.t.Fatalf("the built items are not consecutive: %v", ids)
		}
		want := top(ids[i-1]) + estimateHeight(ids[i-1])
		if l.reverse {
			// The items grow upwards: item i ends where item i-1 starts.
			want = top(ids[i-1]) - estimateHeight(ids[i])
		}
		if got := top(ids[i]); math.Abs(got-want) > 1e-3 {
			l.t.Errorf("item %d is at %v, want %v (item %d ends there): a gap or an overlap", ids[i], got, want, ids[i-1])
		}
	}
	return top(ids[0])
}

// settle pumps until the scroll offset stops changing (a correction takes
// another layout).
func (l *estimateList) settle() {
	l.t.Helper()
	for i := 0; i < 8; i++ {
		before := l.ctl.Offset()
		l.ts.Pump()
		if l.ctl.Offset() == before {
			return
		}
	}
}

func TestEstimateItemOffsetsLaysOutOneWindow(t *testing.T) {
	for _, tc := range []struct {
		name     string
		estimate bool
		reverse  bool
		target   func(max float64) float64
		// window: the most items the jump may build with the option; manyAtLeast:
		// the fewest without it.
		window, manyAtLeast int
	}{
		{"middle, option on", true, false, func(max float64) float64 { return max / 2 }, 40, 0},
		{"end, option on", true, false, func(max float64) float64 { return max }, 40, 0},
		{"middle, option off", false, false, func(max float64) float64 { return max / 2 }, 0, 1000},
		{"end, option off", false, false, func(max float64) float64 { return max }, 0, 2000},
		{"middle, reverse, option on", true, true, func(max float64) float64 { return max / 2 }, 40, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newEstimateList(t, estimateConfig{estimate: tc.estimate, reverse: tc.reverse})
			l.ts.Pump()
			l.built = 0
			l.ctl.JumpTo(tc.target(l.ctl.Position().MaxScrollExtent()))
			l.settle()
			t.Logf("builder calls: %d, first item shown: %v", l.built, l.shown()[:1])
			if tc.estimate && l.built > tc.window {
				t.Errorf("the jump built %d items, want at most %d (one window)", l.built, tc.window)
			}
			if !tc.estimate && l.built < tc.manyAtLeast {
				t.Errorf("the jump built %d items, want at least %d (Flutter builds all in between)", l.built, tc.manyAtLeast)
			}
			l.checkContiguous()
		})
	}
}

func TestEstimateItemOffsetsScrollingBackEndsAtItemZero(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target func(max float64) float64
	}{
		{"from the middle", func(max float64) float64 { return max / 2 }},
		{"from the end", func(max float64) float64 { return max }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newEstimateList(t, estimateConfig{estimate: true})
			l.ts.Pump()
			l.ctl.JumpTo(tc.target(l.ctl.Position().MaxScrollExtent()))
			l.settle()
			l.checkContiguous()
			for i := 0; i < 20000 && l.ctl.Offset() > 0; i++ {
				l.ctl.JumpTo(math.Max(0, l.ctl.Offset()-400))
				l.settle()
				l.checkContiguous()
			}
			if got := l.ctl.Offset(); got != 0 {
				t.Fatalf("the scroll offset is %v after scrolling back, want 0", got)
			}
			ids := l.shown()
			if ids[0] != 0 {
				t.Fatalf("the first item shown is %d, want 0", ids[0])
			}
			if top := l.checkContiguous(); math.Abs(top) > 1e-6 {
				t.Errorf("item 0 is at %v, want 0", top)
			}
		})
	}
}

func TestEstimateItemOffsetsJumpsBackwardsFromTheEnd(t *testing.T) {
	l := newEstimateList(t, estimateConfig{estimate: true})
	l.ts.Pump()
	l.ctl.JumpTo(l.ctl.Position().MaxScrollExtent())
	l.settle()
	l.built = 0
	l.ctl.JumpTo(l.ctl.Position().MaxScrollExtent() / 3)
	l.settle()
	if l.built > 40 {
		t.Errorf("the backward jump built %d items, want at most 40", l.built)
	}
	l.checkContiguous()
	ids := l.shown()
	if ids[0] < 500 || ids[len(ids)-1] > 1500 {
		t.Errorf("the jump to a third of the list shows items %d to %d, want items near 1000", ids[0], ids[len(ids)-1])
	}
}

func TestEstimateItemOffsetsWithMaintainVisibleContentPosition(t *testing.T) {
	l := newEstimateList(t, estimateConfig{estimate: true, keep: true})
	l.ts.Pump()
	l.ctl.JumpTo(l.ctl.Position().MaxScrollExtent() / 2)
	l.settle()
	l.checkContiguous()
	pos := map[int]float64{}
	for _, id := range l.shown() {
		pos[id] = l.ts.TopLeft(gluttertest.ByText(fmt.Sprintf("item %d", id))).Dy
	}
	// Five new items at the start.
	l.setIDs(func() {
		fresh := []int{estimateItems, estimateItems + 1, estimateItems + 2, estimateItems + 3, estimateItems + 4}
		l.ids = append(fresh, l.ids...)
	})
	l.ts.Pump()
	l.settle()
	kept := 0
	for id, was := range pos {
		if l.ts.Count(gluttertest.ByText(fmt.Sprintf("item %d", id))) != 1 {
			continue
		}
		if got := l.ts.TopLeft(gluttertest.ByText(fmt.Sprintf("item %d", id))).Dy; math.Abs(got-was) > 1e-6 {
			t.Errorf("item %d moved from %v to %v when items were inserted before it", id, was, got)
		}
		kept++
	}
	if kept < 3 {
		t.Errorf("only %d of the items shown before the insert are still built", kept)
	}
}

// BenchmarkEstimateItemOffsetsJump is a jump to the middle of a list of 3,000
// items of different heights, and back to the start, with and without the option.
func BenchmarkEstimateItemOffsetsJump(b *testing.B) {
	for _, estimate := range []bool{false, true} {
		b.Run(fmt.Sprintf("estimate=%v", estimate), func(b *testing.B) {
			l := newEstimateList(b, estimateConfig{estimate: estimate})
			l.ts.Pump()
			middle := l.ctl.Position().MaxScrollExtent() / 2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				l.ctl.JumpTo(middle)
				l.ts.Pump()
				l.ctl.JumpTo(0)
				l.ts.Pump()
			}
		})
	}
}
