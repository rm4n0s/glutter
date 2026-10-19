package semantics_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// treeWithUpdates makes an owner whose updates are collected.
func treeWithUpdates() (*semantics.SemanticsOwner, *[]*ui.SemanticsUpdate) {
	var updates []*ui.SemanticsUpdate
	owner := semantics.NewSemanticsOwner(func(u *ui.SemanticsUpdate) { updates = append(updates, u) })
	return owner, &updates
}

func rootConfig() *semantics.SemanticsConfiguration {
	c := semantics.NewSemanticsConfiguration()
	c.SetIsSemanticBoundary(true)
	c.SetTextDirection(ui.TextDirectionLTR)
	return c
}

func labeledChild(r ui.Rect, label string) *semantics.SemanticsNode {
	c := semantics.NewSemanticsConfiguration()
	c.SetIsSemanticBoundary(true)
	c.SetLabel(label)
	c.SetTextDirection(ui.TextDirectionLTR)
	n := nodeWithRect(r)
	n.UpdateWith(c, nil)
	return n
}

func TestSendSemanticsUpdateOrdersChildren(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	owner, updates := treeWithUpdates()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 100, 100))
	a := labeledChild(rect(0, 0, 10, 10), "a")
	b := labeledChild(rect(20, 0, 30, 10), "b")
	c := labeledChild(rect(0, 20, 10, 30), "c")
	// The children are in inverse paint order.
	root.UpdateWith(rootConfig(), []*semantics.SemanticsNode{c, b, a})
	owner.SendSemanticsUpdate()
	if len(*updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(*updates))
	}
	update := (*updates)[0]
	var rootUpdate ui.SemanticsNodeUpdate
	labels := map[int]string{}
	for _, n := range update.Nodes {
		labels[n.ID] = n.Label
		if n.ID == 0 {
			rootUpdate = n
		}
	}
	if len(update.Nodes) != 4 {
		t.Fatalf("nodes = %d, want 4", len(update.Nodes))
	}
	var traversal []string
	for _, id := range rootUpdate.ChildrenInTraversalOrder {
		traversal = append(traversal, labels[id])
	}
	if !slices.Equal(traversal, []string{"a", "b", "c"}) {
		t.Errorf("traversal order = %v, want [a b c]", traversal)
	}
	var hit []string
	for _, id := range rootUpdate.ChildrenInHitTestOrder {
		hit = append(hit, labels[id])
	}
	if !slices.Equal(hit, []string{"a", "b", "c"}) {
		t.Errorf("hit test order = %v, want [a b c]", hit)
	}

	// Nothing changed: no new update with nodes.
	owner.SendSemanticsUpdate()
	if len(*updates) != 1 {
		t.Errorf("an unchanged tree sent %d updates", len(*updates))
	}
}

func TestSortKeysOverridePosition(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	owner, updates := treeWithUpdates()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 100, 100))
	keyed := func(r ui.Rect, label string, order float64) *semantics.SemanticsNode {
		c := semantics.NewSemanticsConfiguration()
		c.SetIsSemanticBoundary(true)
		c.SetLabel(label)
		c.SetTextDirection(ui.TextDirectionLTR)
		c.SetSortKey(semantics.NewOrdinalSortKey(order))
		n := nodeWithRect(r)
		n.UpdateWith(c, nil)
		return n
	}
	left := keyed(rect(0, 0, 10, 10), "left", 2)
	right := keyed(rect(20, 0, 30, 10), "right", 1)
	root.UpdateWith(rootConfig(), []*semantics.SemanticsNode{right, left})
	owner.SendSemanticsUpdate()
	update := (*updates)[0]
	labels := map[int]string{}
	var rootUpdate ui.SemanticsNodeUpdate
	for _, n := range update.Nodes {
		labels[n.ID] = n.Label
		if n.ID == 0 {
			rootUpdate = n
		}
	}
	var traversal []string
	for _, id := range rootUpdate.ChildrenInTraversalOrder {
		traversal = append(traversal, labels[id])
	}
	if !slices.Equal(traversal, []string{"right", "left"}) {
		t.Errorf("traversal order = %v, want [right left]", traversal)
	}
}

func TestPerformActionIgnoresMissingHandlers(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	owner, _ := treeWithUpdates()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 100, 100))
	taps := 0
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetOnTap(func() { taps++ })
	root.UpdateWith(config, nil)
	owner.PerformAction(0, ui.SemanticsActionTap, nil)
	owner.PerformAction(0, ui.SemanticsActionLongPress, nil) // no handler: ignored
	owner.PerformAction(42, ui.SemanticsActionTap, nil)      // no node: ignored
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
}

func TestBlockedUserActionsAreLeftOutOfTheUpdate(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	owner, updates := treeWithUpdates()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 100, 100))
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.IsBlockingUserActions = true
	config.SetOnTap(noop)
	config.SetOnDidGainAccessibilityFocus(noop)
	root.UpdateWith(config, nil)
	owner.SendSemanticsUpdate()
	actions := (*updates)[0].Nodes[0].Actions
	if actions != ui.SemanticsActionDidGainAccessibilityFocus {
		t.Errorf("actions = %v, want only didGainAccessibilityFocus", actions)
	}
}

func TestCustomActionsAreSentWithTheUpdate(t *testing.T) {
	semantics.DebugResetSemanticsIDCounter()
	semantics.ResetCustomSemanticsActionsForTests()
	owner, updates := treeWithUpdates()
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rect(0, 0, 100, 100))
	calls := 0
	action := semantics.NewCustomSemanticsAction("Archive")
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetCustomSemanticsActions(map[semantics.CustomSemanticsAction]semantics.VoidCallback{action: func() { calls++ }})
	root.UpdateWith(config, nil)
	owner.SendSemanticsUpdate()
	update := (*updates)[0]
	if len(update.CustomActions) != 1 || update.CustomActions[0].Label != "Archive" {
		t.Fatalf("custom actions = %+v", update.CustomActions)
	}
	id := update.CustomActions[0].ID
	if id != semantics.CustomSemanticsActionGetIdentifier(action) {
		t.Errorf("id = %d", id)
	}
	owner.PerformAction(0, ui.SemanticsActionCustomAction, id)
	if calls != 1 {
		t.Errorf("custom action calls = %d, want 1", calls)
	}
}

func TestSemanticsEventMaps(t *testing.T) {
	tests := []struct {
		name  string
		event semantics.SemanticsEvent
		node  foundation.Opt[int]
		want  map[string]any
	}{
		{"announce", semantics.NewAnnounceSemanticsEvent("hi", ui.TextDirectionRTL, 2), foundation.None[int](),
			map[string]any{"type": "announce", "data": map[string]any{"viewId": 2, "message": "hi", "textDirection": 0}}},
		{"announce assertive", semantics.NewAnnounceSemanticsEvent("hi", ui.TextDirectionLTR, 0, semantics.AssertivenessAssertive), foundation.None[int](),
			map[string]any{"type": "announce", "data": map[string]any{"viewId": 0, "message": "hi", "textDirection": 1, "assertiveness": 1}}},
		{"tooltip", semantics.NewTooltipSemanticsEvent("tip"), foundation.None[int](),
			map[string]any{"type": "tooltip", "data": map[string]any{"message": "tip"}}},
		{"long press", semantics.NewLongPressSemanticsEvent(), foundation.Some(3),
			map[string]any{"type": "longPress", "nodeId": 3, "data": map[string]any{}}},
		{"tap", semantics.NewTapSemanticEvent(), foundation.Some(4),
			map[string]any{"type": "tap", "nodeId": 4, "data": map[string]any{}}},
		{"focus", semantics.NewFocusSemanticEvent(), foundation.Some(5),
			map[string]any{"type": "focus", "nodeId": 5, "data": map[string]any{}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.event.ToMap(tc.node)
			if len(got) != len(tc.want) {
				t.Fatalf("map = %v, want %v", got, tc.want)
			}
			for k, w := range tc.want {
				g := got[k]
				if gm, ok := g.(map[string]any); ok {
					wm := w.(map[string]any)
					if len(gm) != len(wm) {
						t.Errorf("%s = %v, want %v", k, gm, wm)
					}
					for kk, ww := range wm {
						if gm[kk] != ww {
							t.Errorf("%s[%s] = %v, want %v", k, kk, gm[kk], ww)
						}
					}
				} else if g != w {
					t.Errorf("%s = %v, want %v", k, g, w)
				}
			}
		})
	}
}

func TestSendEventGoesThroughTheOwner(t *testing.T) {
	var got []map[string]any
	owner := semantics.NewSemanticsOwner(func(*ui.SemanticsUpdate) {}, semantics.SemanticsOwnerOpts{OnSendEvent: func(m map[string]any) { got = append(got, m) }})
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SendEvent(semantics.NewTapSemanticEvent())
	semantics.NewSemanticsNode().SendEvent(semantics.NewTapSemanticEvent()) // not attached: dropped
	if len(got) != 1 || got[0]["nodeId"] != 0 || got[0]["type"] != "tap" {
		t.Errorf("events = %v", got)
	}
}
