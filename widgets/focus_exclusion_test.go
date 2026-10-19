package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from the focus cases of animated_cross_fade_test.dart,
// indexed_stack_test.dart and visibility_test.dart (all in
// packages/flutter/test/widgets).

func focusText(s string) Widget { return NewFocus(NewText(s)) }

func TestAnimatedCrossFadeFocus(t *testing.T) {
	for _, tc := range []struct {
		name               string
		excludeBottomFocus foundation.Opt[bool]
		hiddenHasFocus     bool
	}{
		{"test focus", foundation.None[bool](), false},
		{"bottom child can have focus", foundation.Some(false), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(ltr(NewAnimatedCrossFade(focusText("AAA"), focusText("BBB"), CrossFadeStateShowFirst, 50*time.Millisecond,
				AnimatedCrossFadeOpts{ExcludeBottomFocus: tc.excludeBottomFocus})))
			scopeOK := FocusOfOpts{ScopeOK: true}
			visibleNode := FocusOf(ts.Element(gluttertest.ByText("AAA")).(BuildContext), scopeOK)
			visibleNode.RequestFocus()
			ts.Pump()
			if !visibleNode.HasPrimaryFocus() {
				t.Fatal("the visible node should have the primary focus")
			}
			hiddenNode := FocusOf(ts.Element(gluttertest.ByText("BBB")).(BuildContext), scopeOK)
			hiddenNode.RequestFocus()
			ts.Pump()
			if hiddenNode.HasPrimaryFocus() != tc.hiddenHasFocus {
				t.Errorf("hidden node has primary focus = %v, want %v", hiddenNode.HasPrimaryFocus(), tc.hiddenHasFocus)
			}
		})
	}
}

func TestIndexedStackExcludesFocusForHiddenChildren(t *testing.T) {
	ts := gluttertest.New(t)
	pump := func(index int) {
		ts.PumpWidget(ltr(NewIndexedStack([]Widget{focusText("child 0"), focusText("child 1")}, IndexedStackOpts{Index: foundation.Some(index)})))
	}
	pump(0)
	find := func(s string) *FocusNode {
		return FocusOf(ts.Element(gluttertest.ByText(s, gluttertest.FindOpts{IncludeOffstage: true})).(BuildContext))
	}
	child0, child1 := find("child 0"), find("child 1")
	request := func(n *FocusNode) { n.RequestFocus(); ts.Pump() }
	check := func(step string, want0, want1 bool) {
		t.Helper()
		if child0.HasFocus() != want0 || child1.HasFocus() != want1 {
			t.Errorf("%s: focus = (%v, %v), want (%v, %v)", step, child0.HasFocus(), child1.HasFocus(), want0, want1)
		}
	}
	request(child0)
	check("child 0 requested", true, false)
	request(child1)
	check("child 1 requested while hidden", true, false)
	pump(1)
	request(child1)
	check("child 1 requested after showing it", false, true)
	request(child0)
	check("child 0 requested while hidden", false, true)
}

func TestVisibilityFocus(t *testing.T) {
	for _, tc := range []struct {
		name        string
		build       func(visible bool) Widget
		keepsFocus  bool
		unfocusDone bool // the node is requested again without a pump, as in Dart
	}{
		{"maintain* false excludes focus of child when not visible",
			func(v bool) Widget {
				return ltr(NewVisibility(focusText("child"), VisibilityOpts{Visible: foundation.Some(v)}))
			}, false, true},
		{"maintain* true does not exclude focus of child when not visible",
			func(v bool) Widget {
				return ltr(NewVisibilityMaintain(focusText("child"), VisibilityOpts{Visible: foundation.Some(v)}))
			}, true, false},
		{"maintain* true except maintainFocusability excludes focus of child when not visible",
			func(v bool) Widget {
				return ltr(NewVisibility(focusText("child"), VisibilityOpts{Visible: foundation.Some(v), MaintainState: true,
					MaintainAnimation: true, MaintainInteractivity: true, MaintainSemantics: true, MaintainSize: true}))
			}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(tc.build(true))
			node := FocusOf(ts.Element(gluttertest.ByText("child", gluttertest.FindOpts{IncludeOffstage: true})).(BuildContext))
			node.RequestFocus()
			ts.Pump()
			if !node.HasFocus() {
				t.Fatal("the node should have the focus")
			}
			ts.PumpWidget(tc.build(false))
			if tc.unfocusDone {
				node.RequestFocus()
			}
			if node.HasFocus() != tc.keepsFocus {
				t.Errorf("has focus = %v, want %v", node.HasFocus(), tc.keepsFocus)
			}
		})
	}
}

// The keyboardDismissBehavior cases of scroll_view_test.dart and
// single_child_scroll_view_test.dart, with Focus widgets standing in for the
// EditableText fields (which come with step 11.4).
func TestKeyboardDismissBehaviorUnfocusesOnDrag(t *testing.T) {
	boxes := func(nodes []*FocusNode) []Widget {
		var out []Widget
		for _, n := range nodes {
			out = append(out, NewSizedBox(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: n}), SizedBoxOpts{Height: foundation.Some(50.0)}))
		}
		return out
	}
	for _, tc := range []struct {
		name  string
		build func(b ScrollViewKeyboardDismissBehavior, nodes []*FocusNode) Widget
	}{
		{"ListView", func(b ScrollViewKeyboardDismissBehavior, nodes []*FocusNode) Widget {
			return ltr(NewListView(boxes(nodes), ListViewOpts{KeyboardDismissBehavior: b}))
		}},
		{"SingleChildScrollView", func(b ScrollViewKeyboardDismissBehavior, nodes []*FocusNode) Widget {
			return ltr(NewSingleChildScrollView(NewColumn(boxes(nodes)), SingleChildScrollViewOpts{KeyboardDismissBehavior: b}))
		}},
	} {
		for _, b := range []struct {
			name     string
			behavior ScrollViewKeyboardDismissBehavior
			want     bool
		}{
			{"onDrag", ScrollViewKeyboardDismissBehaviorOnDrag, false},
			{"manual", ScrollViewKeyboardDismissBehaviorManual, true},
		} {
			t.Run(tc.name+" "+b.name, func(t *testing.T) {
				ts := gluttertest.New(t)
				nodes := make([]*FocusNode, 50)
				for i := range nodes {
					nodes[i] = NewFocusNode()
				}
				defer func() {
					for _, n := range nodes {
						n.Dispose()
					}
				}()
				ts.PumpWidget(tc.build(b.behavior, nodes))
				nodes[0].RequestFocus()
				ts.Pump()
				if !nodes[0].HasFocus() {
					t.Fatal("the first node should have the focus")
				}
				ts.DragFrom(ui.Offset{Dx: 100, Dy: 25}, ui.Offset{Dy: -40})
				ts.PumpAndSettle()
				if nodes[0].HasFocus() != b.want {
					t.Errorf("has focus = %v, want %v", nodes[0].HasFocus(), b.want)
				}
			})
		}
	}
}
