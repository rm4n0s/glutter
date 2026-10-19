package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/automatic_keep_alive_test.dart.

type autoLeaf struct {
	StatefulWidget
	child Widget
}

func newAutoLeaf(key foundation.Key, child Widget) *autoLeaf {
	return &autoLeaf{NewStatefulWidgetBase(WidgetOpts{Key: key}), child}
}

func (w *autoLeaf) CreateState() State { return &autoLeafState{} }

type autoLeafState struct {
	StateOf[*autoLeaf]
	keepAlive bool
	handle    *KeepAliveHandle
}

func (s *autoLeafState) Deactivate() {
	if s.handle != nil {
		s.handle.Dispose()
		s.handle = nil
	}
}

func (s *autoLeafState) setKeepAlive(v bool) {
	s.keepAlive = v
	if v {
		if s.handle == nil {
			s.handle = NewKeepAliveHandle()
			DispatchNotification(s.Context(), KeepAliveNotification{Handle: s.handle})
		}
	} else if s.handle != nil {
		s.handle.Dispose()
		s.handle = nil
	}
}

func (s *autoLeafState) Build(BuildContext) Widget {
	if s.keepAlive && s.handle == nil {
		s.handle = NewKeepAliveHandle()
		DispatchNotification(s.Context(), KeepAliveNotification{Handle: s.handle})
	}
	return s.Widget().child
}

func autoLeafKey(i int) foundation.Key { return NewGlobalObjectKey[*autoLeafState](i) }

func generateAutoList(child Widget, impliedMode bool) []Widget {
	out := make([]Widget, 100)
	for i := range out {
		leaf := newAutoLeaf(autoLeafKey(i), child)
		if impliedMode {
			out[i] = leaf
		} else {
			out[i] = NewAutomaticKeepAlive(leaf)
		}
	}
	return out
}

func TestAutomaticKeepAliveInLists(t *testing.T) {
	for _, impliedMode := range []bool{false, true} {
		mode := "explicit"
		if impliedMode {
			mode = "implied"
		}
		opts := ListViewOpts{ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)), AddAutomaticKeepAlives: foundation.Some(impliedMode),
			AddRepaintBoundaries: foundation.Some(impliedMode), AddSemanticIndexes: foundation.Some(false)}
		withExtent := opts
		withExtent.ItemExtent = foundation.Some(12.3)
		tests := []struct {
			name  string
			build func() Widget
			find  func(...gluttertest.FindOpts) gluttertest.Finder
		}{
			{"ListView with itemExtent", func() Widget {
				return NewListView(generateAutoList(NewPlaceholder(nil), impliedMode), withExtent)
			}, gluttertest.ByType[*ListView]},
			{"ListView without itemExtent", func() Widget {
				return NewListView(generateAutoList(NewSizedBox(NewPlaceholder(nil), SizedBoxOpts{Height: foundation.Some(12.3)}), impliedMode), opts)
			}, gluttertest.ByType[*ListView]},
			{"GridView", func() Widget {
				return NewGridViewCount(2, generateAutoList(NewPlaceholder(nil), impliedMode), GridViewOpts{ScrollCacheExtent: opts.ScrollCacheExtent,
					AddAutomaticKeepAlives: opts.AddAutomaticKeepAlives, AddRepaintBoundaries: opts.AddRepaintBoundaries, AddSemanticIndexes: opts.AddSemanticIndexes,
					ChildAspectRatio: foundation.Some(400.0 / 24.6)})
			}, gluttertest.ByType[*GridView]},
		}
		for _, tc := range tests {
			t.Run(mode+" "+tc.name, func(t *testing.T) {
				runKeepAliveScenarioKey(t, tc.build(), tc.find, autoLeafKey, func(ts *gluttertest.Tester, i int, v bool) {
					s, ok := NewGlobalObjectKey[*autoLeafState](i).CurrentStateIn(ts.Binding.BuildOwner())
					if !ok {
						t.Fatalf("no state for key %d", i)
					}
					s.setKeepAlive(v)
				})
			})
		}
	}
}

func TestAutomaticKeepAliveDouble(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	no := foundation.Some(false)
	box := func(i int) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Key: NewGlobalObjectKey[*autoLeafState](i), Height: foundation.Some(400.0)})
	}
	ts.PumpWidget(ltr(NewListView([]Widget{
		NewAutomaticKeepAlive(NewSizedBox(NewStack([]Widget{
			newAutoLeaf(autoLeafKey(0), NewPlaceholder(nil)), newAutoLeaf(autoLeafKey(1), NewPlaceholder(nil))}), SizedBoxOpts{Height: foundation.Some(400.0)})),
		NewAutomaticKeepAlive(box(2)),
		NewAutomaticKeepAlive(box(3)),
	}, ListViewOpts{AddAutomaticKeepAlives: no, AddRepaintBoundaries: no, AddSemanticIndexes: no,
		ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0))})))
	off := gluttertest.FindOpts{IncludeOffstage: true}
	count := func(i int, offstage bool) int {
		if offstage {
			return ts.Count(gluttertest.ByKey(autoLeafKey(i), off))
		}
		return ts.Count(gluttertest.ByKey(autoLeafKey(i)))
	}
	expect := func(step string, want [4][2]int) {
		t.Helper()
		for i, w := range want {
			if got := count(i, false); got != w[0] {
				t.Errorf("%s: key %d onstage = %d, want %d", step, i, got, w[0])
			}
			if got := count(i, true); got != w[1] {
				t.Errorf("%s: key %d with offstage = %d, want %d", step, i, got, w[1])
			}
		}
	}
	expect("initial", [4][2]int{{1, 1}, {1, 1}, {1, 1}, {0, 0}})
	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: -1000})
	ts.Pump()
	expect("bottom", [4][2]int{{0, 0}, {0, 0}, {1, 1}, {1, 1}})
	ts.Drag(gluttertest.ByType[*ListView](), ui.Offset{Dy: 1000})
	ts.Pump()
	expect("top", [4][2]int{{1, 1}, {1, 1}, {1, 1}, {0, 0}})
}
