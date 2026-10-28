package widgets_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover what Flutter does not do (SPEC §11.7): a focus node that
// leaves the tree while it has the focus notifies its listeners that it lost it.

// tabHost is a widget tree with two tabs: tab 0 is built by content and tab 1 is
// an empty box. hostSet rebuilds the host, which stays in the tree.
type tabHost struct {
	tab     int
	hostSet StateSetter
	// hostBuilds counts the builds of the host.
	hostBuilds int
}

func (h *tabHost) widget(content func() Widget) Widget {
	return NewStatefulBuilder(func(bc BuildContext, set StateSetter) Widget {
		h.hostSet = set
		h.hostBuilds++
		if h.tab != 0 {
			return NewSizedBox(nil)
		}
		return content()
	})
}

func (h *tabHost) show(ts *gluttertest.Tester, tab int) {
	h.hostSet(func() { h.tab = tab })
	ts.Pump()
}

// settle pumps a few frames, so that every focus update has run.
func settle(ts *gluttertest.Tester) {
	for range 3 {
		ts.Pump()
	}
}

func TestFocusNodeOfTheAppThatLeavesTheTreeWithTheFocusIsNotified(t *testing.T) {
	tests := []struct {
		name string
		// descendant puts the primary focus on a Focus below the node.
		descendant bool
		// setState makes the listener call SetState on the host, which stays in
		// the tree.
		setState bool
	}{
		{name: "primary focus"},
		{name: "a descendant has the primary focus", descendant: true},
		{name: "the listener calls SetState on a state that stays", setState: true},
		{name: "a descendant has the primary focus, the listener calls SetState on a state that stays", descendant: true, setState: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			h := &tabHost{}
			node := NewFocusNode()
			defer node.Dispose()
			var heard []bool
			node.AddListener(func() {
				heard = append(heard, node.HasFocus())
				if tt.setState {
					h.hostSet(func() {})
				}
			})
			ts.PumpWidget(h.widget(func() Widget {
				if tt.descendant {
					return NewFocus(NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true}), FocusOpts{FocusNode: node})
				}
				return NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: node})
			}))
			settle(ts)
			if !node.HasFocus() {
				t.Fatal("the node should have the focus")
			}
			if !slices.Equal(heard, []bool{true}) {
				t.Fatalf("before the tab changes, the listener heard %v, want [true]", heard)
			}
			builds := h.hostBuilds
			h.show(ts, 1)
			if node.HasFocus() {
				t.Error("the node still has the focus after it left the tree")
			}
			settle(ts)
			if !slices.Equal(heard, []bool{true, false}) {
				t.Errorf("the listener heard %v, want [true false]", heard)
			}
			settle(ts)
			if len(heard) != 2 {
				t.Errorf("the listener heard %v, want no more after the next frames", heard)
			}
			if tt.setState && h.hostBuilds <= builds+1 {
				t.Errorf("the host was built %d times after the tab changed, want a rebuild for the SetState of the listener", h.hostBuilds-builds)
			}
			if errs := ts.TakeErrors(); len(errs) != 0 {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestFocusWithANodeOfTheAppThatLeavesTheTreeWithTheFocusOnlyNotifiesTheNode(t *testing.T) {
	ts := gluttertest.New(t)
	h := &tabHost{}
	node := NewFocusNode()
	defer node.Dispose()
	var heard, called []bool
	node.AddListener(func() { heard = append(heard, node.HasFocus()) })
	ts.PumpWidget(h.widget(func() Widget {
		return NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: node,
			OnFocusChange: func(has bool) { called = append(called, has) }})
	}))
	settle(ts)
	h.show(ts, 1)
	settle(ts)
	if !slices.Equal(heard, []bool{true, false}) {
		t.Errorf("the node's listener heard %v, want [true false]", heard)
	}
	if !slices.Equal(called, []bool{true}) {
		t.Errorf("OnFocusChange heard %v, want [true] only: the widget is gone when the node is told", called)
	}
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestFocusNodeDetachedByAttachmentWhileFocusedIsNotifiedOnce(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	node := NewFocusNode()
	a := attach(t, ctx, node)
	a.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	var heard []bool
	node.AddListener(func() { heard = append(heard, node.HasFocus()) })
	node.RequestFocus()
	settle(ts)
	if !slices.Equal(heard, []bool{true}) {
		t.Fatalf("heard %v, want [true]", heard)
	}
	a.attachment.Detach()
	settle(ts)
	if !slices.Equal(heard, []bool{true, false}) {
		t.Errorf("after Detach, the listener heard %v, want [true false]", heard)
	}
}

func TestFocusNodeDisposedAfterLeavingTheTreeWithTheFocusIsNotNotified(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	node := NewFocusNode()
	attachment := node.Attach(ctx)
	attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	var heard []bool
	node.AddListener(func() { heard = append(heard, node.HasFocus()) })
	node.RequestFocus()
	settle(ts)
	attachment.Detach()
	// Disposing the node before the focus update: it must not be notified, and
	// the update must not touch the disposed node.
	node.Dispose()
	settle(ts)
	if !slices.Equal(heard, []bool{true}) {
		t.Errorf("a disposed node heard %v, want [true]", heard)
	}
}

func TestFocusNodeThatLeavesTheTreeWithoutTheFocusIsNotNotified(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	node, other := NewFocusNode(), NewFocusNode()
	a, b := attach(t, ctx, node), attach(t, ctx, other)
	root := ts.Binding.FocusManager().RootScope().Node()
	a.attachment.Reparent(root)
	b.attachment.Reparent(root)
	var heard []bool
	node.AddListener(func() { heard = append(heard, node.HasFocus()) })
	other.RequestFocus()
	settle(ts)
	heard = nil
	a.attachment.Detach()
	settle(ts)
	if len(heard) != 0 {
		t.Errorf("a node without the focus heard %v when it left the tree, want nothing", heard)
	}
}

func TestFocusWidgetMovedWithAGlobalKeyKeepsTheFocusAndIsNotNotified(t *testing.T) {
	ts := gluttertest.New(t)
	key := NewGlobalKey[State]()
	node := NewFocusNode()
	defer node.Dispose()
	var called []bool
	focus := func() Widget {
		return NewFocus(NewSizedBox(nil), FocusOpts{Key: key, Autofocus: true, FocusNode: node,
			OnFocusChange: func(has bool) { called = append(called, has) }})
	}
	var heard []bool
	node.AddListener(func() { heard = append(heard, node.HasFocus()) })
	ts.PumpWidget(NewColumn([]Widget{NewPadding(painting.NewEdgeInsetsAll(1), focus()), NewSizedBox(nil)}))
	settle(ts)
	if !node.HasPrimaryFocus() {
		t.Fatal("the Focus should have the primary focus")
	}
	ts.PumpWidget(NewColumn([]Widget{NewSizedBox(nil), NewPadding(painting.NewEdgeInsetsAll(1), focus())}))
	settle(ts)
	if !node.HasPrimaryFocus() {
		t.Error("the Focus moved with a GlobalKey lost the focus")
	}
	if !slices.Equal(heard, []bool{true}) {
		t.Errorf("the node's listener heard %v, want [true] only", heard)
	}
	if !slices.Equal(called, []bool{true}) {
		t.Errorf("OnFocusChange heard %v, want [true] only", called)
	}
}
