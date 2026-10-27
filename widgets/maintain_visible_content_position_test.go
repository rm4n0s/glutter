package widgets_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The tests of the MaintainVisibleContentPosition option (SPEC §11, §12.10):
// 40 keyed rows of 50 pixels in a list of 600 pixels, newest first, and rows
// inserted at item 0, as a live log does.

const (
	liveRowExtent = 50.0
	liveRows      = 40
	liveViewport  = 600.0
)

type liveLogConfig struct {
	kind    string // "list", "fixed", "varied" or "prototype"
	reverse bool
	option  bool
	keyed   bool // each row has a ValueKey
	find    bool // the list has a FindChildIndexCallback
	offset  float64
}

type liveLog struct {
	t      *testing.T
	ts     *gluttertest.Tester
	cfg    liveLogConfig
	ctl    ScrollController
	ids    []int // the id of the row at each index, newest first
	nextID int
	set    StateSetter
}

func newLiveLog(t *testing.T, cfg liveLogConfig) *liveLog {
	t.Helper()
	l := &liveLog{t: t, ts: gluttertest.New(t, gluttertest.Opts{}), cfg: cfg, ctl: NewScrollController()}
	for i := 0; i < liveRows; i++ {
		l.ids = append(l.ids, liveRows-i)
	}
	l.nextID = liveRows + 1
	l.ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		l.set = setState
		return l.list()
	})))
	if cfg.offset != 0 {
		l.ctl.JumpTo(cfg.offset)
		l.ts.Pump()
	}
	return l
}

func (l *liveLog) row(id int) Widget {
	var key foundation.Key
	if l.cfg.keyed {
		key = foundation.NewValueKey(id)
	}
	return NewSizedBox(NewText(fmt.Sprintf("row %d", id)), SizedBoxOpts{Key: key, Height: foundation.Some(liveRowExtent)})
}

func (l *liveLog) list() Widget {
	opts := ListViewOpts{
		Reverse: l.cfg.reverse, Controller: l.ctl, ItemCount: foundation.Some(len(l.ids)),
		MaintainVisibleContentPosition: l.cfg.option,
	}
	if l.cfg.find {
		ids := l.ids
		opts.FindChildIndexCallback = func(key foundation.Key) foundation.Opt[int] {
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
		}
	}
	switch l.cfg.kind {
	case "fixed":
		opts.ItemExtent = foundation.Some(liveRowExtent)
	case "varied":
		opts.ItemExtentBuilder = func(int, rendering.SliverLayoutDimensions) foundation.Opt[float64] {
			return foundation.Some(liveRowExtent)
		}
	case "prototype":
		opts.PrototypeItem = NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(liveRowExtent)})
	}
	ids := l.ids
	return NewListViewBuilder(func(_ BuildContext, i int) Widget {
		if i >= len(ids) {
			return nil
		}
		return l.row(ids[i])
	}, opts)
}

// insertFront adds n rows at item 0 and pumps a frame.
func (l *liveLog) insertFront(n int) {
	l.t.Helper()
	l.set(func() {
		fresh := make([]int, 0, n+len(l.ids))
		for i := 0; i < n; i++ {
			fresh = append(fresh, l.nextID+n-1-i)
		}
		l.nextID += n
		l.ids = append(fresh, l.ids...)
	})
	l.ts.Pump()
}

// positions are the top-left corners of the rows on the screen, by id.
func (l *liveLog) positions() map[int]ui.Offset {
	out := map[int]ui.Offset{}
	for id := 1; id < l.nextID; id++ {
		finder := gluttertest.ByText(fmt.Sprintf("row %d", id))
		if l.ts.Count(finder) != 1 {
			continue
		}
		if p := l.ts.TopLeft(finder); p.Dy > -liveRowExtent+1 && p.Dy < liveViewport-1 {
			out[id] = p
		}
	}
	return out
}

func sameOffset(a, b ui.Offset) bool { return math.Abs(a.Dx-b.Dx) < 1e-6 && math.Abs(a.Dy-b.Dy) < 1e-6 }

// A visible row keeps its top-left corner (to the pixel) when rows are inserted
// before it, while the list is not at its start.
func TestMaintainVisibleContentPositionKeepsVisibleRows(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		for _, reverse := range []bool{true, false} {
			for _, tc := range []struct {
				name           string
				offset         float64
				frames, insert int
			}{
				{"inside the list", 500, 1, 3},
				{"inside the list, several frames", 500, 4, 5},
				{"many rows at once", 500, 1, 25},
				{"close to the start, item 0 alive", 100, 1, 3},
				{"close to the start, many rows", 100, 1, 30},
				{"just below the start", 20, 1, 4},
			} {
				t.Run(fmt.Sprintf("%s reverse=%v %s", kind, reverse, tc.name), func(t *testing.T) {
					l := newLiveLog(t, liveLogConfig{kind: kind, reverse: reverse, option: true, keyed: true, find: true, offset: tc.offset})
					before := l.positions()
					if len(before) < 8 {
						t.Fatalf("only %d rows on the screen before", len(before))
					}
					offsetBefore := l.ctl.Offset()
					for f := 0; f < tc.frames; f++ {
						l.insertFront(tc.insert)
					}
					after := l.positions()
					for id, want := range before {
						got, ok := after[id]
						if !ok || !sameOffset(got, want) {
							t.Errorf("row %d: top-left %v after, want %v (shown: %v)", id, got, want, ok)
						}
					}
					if kind == "list" && tc.offset >= 400 && l.ctl.Offset() != offsetBefore {
						t.Errorf("scroll offset = %v, want it to stay at %v", l.ctl.Offset(), offsetBefore)
					}
				})
			}
		}
	}
}

// Scrolling to the start after the insertion shows every new row, in order, one
// after the other, and the offset at the start is 0.
func TestMaintainVisibleContentPositionStartShowsEveryNewRow(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		for _, reverse := range []bool{true, false} {
			for _, tc := range []struct {
				offset   float64
				stepwise bool // scroll in steps of 40 pixels, as a drag does, instead of one jump
			}{{500, false}, {100, false}, {500, true}, {100, true}} {
				offset := tc.offset
				t.Run(fmt.Sprintf("%s reverse=%v offset=%v stepwise=%v", kind, reverse, offset, tc.stepwise), func(t *testing.T) {
					l := newLiveLog(t, liveLogConfig{kind: kind, reverse: reverse, option: true, keyed: true, find: true, offset: offset})
					l.insertFront(3)
					l.insertFront(4)
					if tc.stepwise {
						for o := l.ctl.Offset() - 40; o > 0; o -= 40 {
							l.ctl.JumpTo(o)
							l.ts.Pump()
						}
					}
					l.ctl.JumpTo(0)
					l.ts.Pump()
					l.ts.Pump()
					if got := l.ctl.Offset(); got != 0 {
						t.Fatalf("offset at the start = %v, want 0", got)
					}
					newest := l.nextID - 1
					for k := 0; k < 11; k++ {
						want := ui.Offset{Dy: float64(k) * liveRowExtent}
						if reverse {
							want.Dy = liveViewport - liveRowExtent - want.Dy
						}
						finder := gluttertest.ByText(fmt.Sprintf("row %d", newest-k))
						if l.ts.Count(finder) != 1 {
							t.Fatalf("row %d is not shown at the start", newest-k)
						}
						if got := l.ts.TopLeft(finder); !sameOffset(got, want) {
							t.Errorf("row %d: top-left %v, want %v", newest-k, got, want)
						}
					}
					// The end of the list: scrolling to the other end reaches
					// exactly the content that there is (47 rows).
					l.ctl.JumpTo(1e6)
					l.ts.PumpAndSettle()
					extent := 47 * liveRowExtent
					if got := l.ctl.Offset(); math.Abs(got-(extent-liveViewport)) > 1e-6 {
						t.Errorf("offset at the end = %v, want %v", got, extent-liveViewport)
					}
				})
			}
		}
	}
}

// At the start of the list the new rows are shown at the start and the others
// move (the list follows), with or without the option.
func TestMaintainVisibleContentPositionFollowsAtTheStart(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		for _, reverse := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s reverse=%v", kind, reverse), func(t *testing.T) {
				var results [2]map[int]ui.Offset
				for i, option := range []bool{false, true} {
					l := newLiveLog(t, liveLogConfig{kind: kind, reverse: reverse, option: option, keyed: true, find: true})
					l.insertFront(3)
					results[i] = l.positions()
					newest := l.nextID - 1
					want := 0.0
					if reverse {
						want = liveViewport - liveRowExtent
					}
					if p, ok := results[i][newest]; !ok || p.Dy != want {
						t.Errorf("option=%v: newest row at %v (shown: %v), want y=%v", option, p, ok, want)
					}
				}
				if len(results[0]) == 0 || len(results[0]) != len(results[1]) {
					t.Fatalf("rows shown: %d without the option, %d with it", len(results[0]), len(results[1]))
				}
				for id, p := range results[0] {
					if q, ok := results[1][id]; !ok || !sameOffset(p, q) {
						t.Errorf("row %d: %v without the option, %v with it", id, p, q)
					}
				}
			})
		}
	}
}

// Rows inserted after the visible ones move nothing.
func TestMaintainVisibleContentPositionRowsAfterTheVisibleOnes(t *testing.T) {
	for _, kind := range []string{"list", "fixed"} {
		t.Run(kind, func(t *testing.T) {
			l := newLiveLog(t, liveLogConfig{kind: kind, reverse: true, option: true, keyed: true, find: true, offset: 500})
			before := l.positions()
			l.set(func() { l.ids = append(l.ids, 100, 101, 102) })
			l.nextID = 103
			l.ts.Pump()
			after := l.positions()
			for id, want := range before {
				if got, ok := after[id]; !ok || !sameOffset(got, want) {
					t.Errorf("row %d: %v after, want %v", id, got, want)
				}
			}
		})
	}
}

// A visible row that is removed leaves the others where they are without the
// option, and with it nothing panics and the same rows are shown.
func TestMaintainVisibleContentPositionRemovedVisibleRow(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		t.Run(kind, func(t *testing.T) {
			var results [2]map[int]ui.Offset
			for i, option := range []bool{false, true} {
				l := newLiveLog(t, liveLogConfig{kind: kind, reverse: true, option: option, keyed: true, find: true, offset: 500})
				removed := l.ids[14]
				l.set(func() { l.ids = append(append([]int{}, l.ids[:14]...), l.ids[15:]...) })
				l.ts.Pump()
				results[i] = l.positions()
				if _, ok := results[i][removed]; ok {
					t.Errorf("option=%v: the removed row %d is still shown", option, removed)
				}
				if len(results[i]) < 8 {
					t.Errorf("option=%v: %d rows shown", option, len(results[i]))
				}
			}
			for id, p := range results[0] {
				if q, ok := results[1][id]; ok && !sameOffset(p, q) {
					t.Errorf("row %d: %v without the option, %v with it", id, p, q)
				}
			}
		})
	}
}

// Without keys, or without a way to find them, the option changes nothing.
func TestMaintainVisibleContentPositionNeedsKeys(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		for _, tc := range []struct {
			name        string
			keyed, find bool
		}{{"no keys", false, false}, {"keys but no finder", true, false}} {
			t.Run(kind+" "+tc.name, func(t *testing.T) {
				var results [2]map[int]ui.Offset
				var offsets [2]float64
				for i, option := range []bool{false, true} {
					l := newLiveLog(t, liveLogConfig{kind: kind, reverse: true, option: option, keyed: tc.keyed, find: tc.find, offset: 500})
					l.insertFront(3)
					results[i], offsets[i] = l.positions(), l.ctl.Offset()
				}
				if offsets[0] != offsets[1] || len(results[0]) != len(results[1]) {
					t.Fatalf("offset %v and %d rows without the option, %v and %d with it", offsets[0], len(results[0]), offsets[1], len(results[1]))
				}
				for id, p := range results[0] {
					if q, ok := results[1][id]; !ok || !sameOffset(p, q) {
						t.Errorf("row %d: %v without the option, %v with it", id, p, q)
					}
				}
			})
		}
	}
}

// Without the option the list does what Flutter does: a visible row moves by the
// height of the rows inserted before it, with and without the finder, at the
// start and inside the list (the numbers measured on 2026-10-27).
func TestMaintainVisibleContentPositionOffIsFlutter(t *testing.T) {
	for _, find := range []bool{true, false} {
		for _, tc := range []struct {
			offset float64
			row    int
			before float64
			after  float64
		}{
			{0, 35, 300, 150},   // the row at index 5 moves up by 150
			{500, 24, 250, 100}, // the row at index 16
		} {
			t.Run(fmt.Sprintf("find=%v offset=%v", find, tc.offset), func(t *testing.T) {
				l := newLiveLog(t, liveLogConfig{kind: "list", reverse: true, keyed: true, find: find, offset: tc.offset})
				finder := gluttertest.ByText(fmt.Sprintf("row %d", tc.row))
				if got := l.ts.TopLeft(finder).Dy; got != tc.before {
					t.Fatalf("row %d at y=%v before, want %v", tc.row, got, tc.before)
				}
				l.insertFront(3)
				if got := l.ts.TopLeft(finder).Dy; got != tc.after {
					t.Errorf("row %d at y=%v after, want %v", tc.row, got, tc.after)
				}
			})
		}
	}
}

// With the option and the keys of a list of children (NewListView), the visible
// rows stay.
func TestMaintainVisibleContentPositionListOfChildren(t *testing.T) {
	ids := []int{}
	for i := liveRows; i >= 1; i-- {
		ids = append(ids, i)
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ctl := NewScrollController()
	var set StateSetter
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		set = setState
		children := make([]Widget, len(ids))
		for i, id := range ids {
			children[i] = NewSizedBox(NewText(fmt.Sprintf("row %d", id)), SizedBoxOpts{Key: foundation.NewValueKey(id), Height: foundation.Some(liveRowExtent)})
		}
		return NewListView(children, ListViewOpts{Reverse: true, Controller: ctl, MaintainVisibleContentPosition: true})
	})))
	ctl.JumpTo(500)
	ts.Pump()
	finder := gluttertest.ByText("row 28")
	before := ts.TopLeft(finder)
	set(func() { ids = append([]int{43, 42, 41}, ids...) })
	ts.Pump()
	if got := ts.TopLeft(finder); !sameOffset(got, before) {
		t.Errorf("row 28 at %v, want %v", got, before)
	}
}

// The option works through the sliver widgets and the separated list.
func TestMaintainVisibleContentPositionSliverWidgets(t *testing.T) {
	ids := []int{}
	for i := liveRows; i >= 1; i-- {
		ids = append(ids, i)
	}
	ts := gluttertest.New(t, gluttertest.Opts{})
	ctl := NewScrollController()
	var set StateSetter
	find := func(key foundation.Key) foundation.Opt[int] {
		for i, id := range ids {
			if foundation.NewValueKey(id) == key {
				return foundation.Some(i)
			}
		}
		return foundation.None[int]()
	}
	ts.PumpWidget(ltr(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		set = setState
		return NewCustomScrollView([]Widget{
			NewSliverListSeparated(func(_ BuildContext, i int) Widget {
				return NewSizedBox(NewText(fmt.Sprintf("row %d", ids[i])), SizedBoxOpts{Key: foundation.NewValueKey(ids[i]), Height: foundation.Some(liveRowExtent)})
			}, func(BuildContext, int) Widget { return NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(10.0)}) },
				SliverSeparatedOpts{ItemCount: foundation.Some(len(ids)), FindItemIndexCallback: find, MaintainVisibleContentPosition: true}),
		}, CustomScrollViewOpts{Reverse: true, Controller: ctl})
	})))
	ctl.JumpTo(500)
	ts.Pump()
	finder := gluttertest.ByText("row 28")
	if ts.Count(finder) != 1 {
		t.Fatal("row 28 is not shown")
	}
	before := ts.TopLeft(finder)
	set(func() { ids = append([]int{43, 42, 41}, ids...) })
	ts.Pump()
	if got := ts.TopLeft(finder); !sameOffset(got, before) {
		t.Errorf("row 28 at %v, want %v", got, before)
	}
}

// Two insertions before one frame are one rebuild.
func TestMaintainVisibleContentPositionTwoInsertionsInOneFrame(t *testing.T) {
	for _, kind := range []string{"list", "fixed", "varied", "prototype"} {
		t.Run(kind, func(t *testing.T) {
			l := newLiveLog(t, liveLogConfig{kind: kind, reverse: true, option: true, keyed: true, find: true, offset: 500})
			before := l.positions()
			for _, n := range []int{3, 4} {
				l.set(func() {
					fresh := make([]int, 0, n+len(l.ids))
					for i := 0; i < n; i++ {
						fresh = append(fresh, l.nextID+n-1-i)
					}
					l.nextID += n
					l.ids = append(fresh, l.ids...)
				})
			}
			l.ts.Pump()
			after := l.positions()
			for id, want := range before {
				if got, ok := after[id]; !ok || !sameOffset(got, want) {
					t.Errorf("row %d: %v after, want %v", id, got, want)
				}
			}
		})
	}
}
