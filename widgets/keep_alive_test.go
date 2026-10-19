package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/keep_alive_test.dart.
// The 'KeepAlive render tree description' test compares a toStringDeep dump of
// the render tree and is not ported: the dump format is not Flutter's.

type keepAliveLeaf struct {
	StatefulWidget
	child Widget
}

func newKeepAliveLeaf(index int, child Widget) *keepAliveLeaf {
	return &keepAliveLeaf{NewStatefulWidgetBase(WidgetOpts{Key: NewGlobalObjectKey[*keepAliveLeafState](index)}), child}
}

func (w *keepAliveLeaf) CreateState() State { return &keepAliveLeafState{} }

type keepAliveLeafState struct {
	StateOf[*keepAliveLeaf]
	keepAlive bool
}

func (s *keepAliveLeafState) setKeepAlive(v bool) {
	s.SetState(func() { s.keepAlive = v })
}

func (s *keepAliveLeafState) Build(BuildContext) Widget {
	return NewKeepAlive(s.keepAlive, s.Widget().child)
}

func generateKeepAliveList(child Widget) []Widget {
	out := make([]Widget, 100)
	for i := range out {
		out[i] = newKeepAliveLeaf(i, child)
	}
	return out
}

func TestKeepAliveDebugTypicalAncestorWidgetClass(t *testing.T) {
	k := NewKeepAlive(false, NewContainer(nil))
	if got := k.DebugTypicalAncestorWidgetDescription(); got != "SliverWithKeepAliveWidget or TwoDimensionalViewport" {
		t.Errorf("description = %q", got)
	}
}

func TestKeepAliveInLists(t *testing.T) {
	no := foundation.Some(false)
	tests := []struct {
		name  string
		build func() Widget
		type_ func(...gluttertest.FindOpts) gluttertest.Finder
	}{
		{"ListView with itemExtent", func() Widget {
			return NewListView(generateKeepAliveList(NewPlaceholder(nil)), ListViewOpts{
				ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)), AddAutomaticKeepAlives: no, AddRepaintBoundaries: no,
				AddSemanticIndexes: no, ItemExtent: foundation.Some(12.3)})
		}, gluttertest.ByType[*ListView]},
		{"ListView without itemExtent", func() Widget {
			return NewListView(generateKeepAliveList(NewSizedBox(NewPlaceholder(nil), SizedBoxOpts{Height: foundation.Some(12.3)})), ListViewOpts{
				ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)), AddAutomaticKeepAlives: no, AddRepaintBoundaries: no,
				AddSemanticIndexes: no})
		}, gluttertest.ByType[*ListView]},
		{"GridView", func() Widget {
			return NewGridViewCount(2, generateKeepAliveList(NewPlaceholder(nil)), GridViewOpts{
				ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)), AddAutomaticKeepAlives: no, AddRepaintBoundaries: no,
				AddSemanticIndexes: no, ChildAspectRatio: foundation.Some(400.0 / 24.6)})
		}, gluttertest.ByType[*GridView]},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runKeepAliveScenario(t, tc.build(), tc.type_, func(ts *gluttertest.Tester, i int, v bool) {
				s, ok := NewGlobalObjectKey[*keepAliveLeafState](i).CurrentStateIn(ts.Binding.BuildOwner())
				if !ok {
					t.Fatalf("no state for key %d", i)
				}
				s.setKeepAlive(v)
			})
		})
	}
}

// runKeepAliveScenario is the sequence that the list tests of keep_alive_test.dart
// and automatic_keep_alive_test.dart share: scroll down by 300, keep the item 60
// alive, scroll back, release it.
func runKeepAliveScenario(t *testing.T, widget Widget, scrollable func(...gluttertest.FindOpts) gluttertest.Finder,
	setKeepAlive func(ts *gluttertest.Tester, i int, v bool)) {
	t.Helper()
	runKeepAliveScenarioKey(t, widget, scrollable, func(i int) foundation.Key { return NewGlobalObjectKey[*keepAliveLeafState](i) }, setKeepAlive)
}

func runKeepAliveScenarioKey(t *testing.T, widget Widget, scrollable func(...gluttertest.FindOpts) gluttertest.Finder,
	leafKey func(int) foundation.Key, setKeepAlive func(ts *gluttertest.Tester, i int, v bool)) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{})
	off := gluttertest.FindOpts{IncludeOffstage: true}
	count := func(i int, offstage bool) int {
		if offstage {
			return ts.Count(gluttertest.ByKey(leafKey(i), off))
		}
		return ts.Count(gluttertest.ByKey(leafKey(i)))
	}
	check := func(step string, want map[int][2]int) {
		t.Helper()
		for i, w := range want {
			if got := count(i, false); got != w[0] {
				t.Errorf("%s: key %d onstage count = %d, want %d", step, i, got, w[0])
			}
			if got := count(i, true); got != w[1] {
				t.Errorf("%s: key %d offstage-inclusive count = %d, want %d", step, i, got, w[1])
			}
		}
	}
	ts.PumpWidget(ltr(widget))
	check("initial", map[int][2]int{3: {1, 1}, 30: {1, 1}, 59: {0, 0}, 60: {0, 0}, 61: {0, 0}, 90: {0, 0}})
	ts.Drag(scrollable(), ui.Offset{Dy: -300})
	ts.Pump()
	check("scrolled", map[int][2]int{3: {0, 0}, 30: {1, 1}, 59: {1, 1}, 60: {1, 1}, 61: {1, 1}, 90: {0, 0}})
	setKeepAlive(ts, 60, true)
	ts.Drag(scrollable(), ui.Offset{Dy: 300})
	ts.Pump()
	check("kept alive", map[int][2]int{3: {1, 1}, 30: {1, 1}, 59: {0, 0}, 60: {0, 1}, 61: {0, 0}, 90: {0, 0}})
	setKeepAlive(ts, 60, false)
	ts.Pump()
	check("released", map[int][2]int{3: {1, 1}, 30: {1, 1}, 59: {0, 0}, 60: {0, 0}, 61: {0, 0}, 90: {0, 0}})
}
