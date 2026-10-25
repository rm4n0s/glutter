package widgets_test

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Step 27.9: changing the children of a node, lazy nodes, and the keys of the nodes.

type treeItem struct {
	ID    int
	Label string
}

func (i treeItem) String() string { return i.Label }

type itemNode = *TreeSliverNode[treeItem]

func item(id int, label string, opts ...TreeSliverNodeOpts[treeItem]) itemNode {
	return NewTreeSliverNode(treeItem{ID: id, Label: label}, opts...)
}

func kids(nodes ...itemNode) TreeSliverNodeOpts[treeItem] {
	return TreeSliverNodeOpts[treeItem]{Children: nodes, Expanded: true}
}

func itemTree(t *testing.T, nodes []itemNode, opts TreeSliverOpts[treeItem]) (*gluttertest.Tester, *TreeSliverController[treeItem]) {
	t.Helper()
	ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
	if opts.Controller == nil {
		opts.Controller = NewTreeSliverController[treeItem]()
	}
	ts.PumpWidget(treeApp(NewTreeSliver(nodes, opts)))
	return ts, opts.Controller
}

// rows are the labels of the rows that show, from the top.
func rows(ts *gluttertest.Tester, labels ...string) []string {
	var shown []string
	for _, l := range labels {
		if ts.Exists(gluttertest.ByText(l)) {
			shown = append(shown, l)
		}
	}
	slices.SortFunc(shown, func(a, b string) int {
		ya, yb := ts.TopLeft(gluttertest.ByText(a)).Dy, ts.TopLeft(gluttertest.ByText(b)).Dy
		switch {
		case ya < yb:
			return -1
		case ya > yb:
			return 1
		}
		return 0
	})
	return shown
}

func labels() []string { return []string{"a", "b", "c", "d", "e", "f", "g", "h"} }

func TestTreeSliverNodeChangesOfChildren(t *testing.T) {
	tests := []struct {
		name string
		// change changes the children of the folder "a"; f is a node that is not in the tree.
		change      func(a itemNode, b, c, d, f itemNode)
		wantRows    []string
		wantExpands bool // whether "a" is still expanded
	}{
		{"insert at the start", func(a, b, c, d, f itemNode) { a.InsertChild(0, f) }, []string{"a", "f", "b", "c", "d"}, true},
		{"insert in the middle", func(a, b, c, d, f itemNode) { a.InsertChild(1, f) }, []string{"a", "b", "f", "c", "d"}, true},
		{"insert beyond the end", func(a, b, c, d, f itemNode) { a.InsertChild(99, f) }, []string{"a", "b", "c", "d", "f"}, true},
		{"insert before the start", func(a, b, c, d, f itemNode) { a.InsertChild(-3, f) }, []string{"a", "f", "b", "c", "d"}, true},
		{"add", func(a, b, c, d, f itemNode) { a.AddChild(f) }, []string{"a", "b", "c", "d", "f"}, true},
		{"remove one", func(a, b, c, d, f itemNode) { a.RemoveChild(c) }, []string{"a", "b", "d"}, true},
		{"remove a child that is not one", func(a, b, c, d, f itemNode) {
			if a.RemoveChild(f) {
				t.Error("RemoveChild reported a node that is not a child")
			}
		}, []string{"a", "b", "c", "d"}, true},
		{"remove all one by one", func(a, b, c, d, f itemNode) { a.RemoveChild(b); a.RemoveChild(c); a.RemoveChild(d) }, []string{"a"}, false},
		{"replace", func(a, b, c, d, f itemNode) { a.SetChildren([]itemNode{f, c}) }, []string{"a", "f", "c"}, true},
		{"clear", func(a, b, c, d, f itemNode) { a.SetChildren(nil) }, []string{"a"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, c, d, f := item(2, "b"), item(3, "c"), item(4, "d"), item(5, "f")
			a := item(1, "a", kids(b, c, d))
			ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{})
			tt.change(a, b, c, d, f)
			ts.Pump()
			if got := rows(ts, labels()...); !slices.Equal(got, tt.wantRows) {
				t.Fatalf("rows = %v, want %v", got, tt.wantRows)
			}
			if a.IsExpanded() != tt.wantExpands {
				t.Errorf("a expanded = %v, want %v", a.IsExpanded(), tt.wantExpands)
			}
			// The depth and the parent of every node that shows are right, and the nodes that left have none.
			for _, n := range a.Children() {
				if n.Parent() != a || n.Depth() != foundation.Some(1) {
					t.Errorf("%v: parent %v, depth %v", n, n.Parent(), n.Depth())
				}
				if !controller.IsActive(n) {
					t.Errorf("%v is not active", n)
				}
			}
			for _, n := range []itemNode{b, c, d, f} {
				if !slices.Contains(a.Children(), n) && (n.Parent() != nil && n != f || n.Depth().IsSome() && n != f) {
					t.Errorf("%v left the tree but has parent %v and depth %v", n, n.Parent(), n.Depth())
				}
			}
		})
	}
}

func TestTreeSliverNodeChangesBelowDeeperNodes(t *testing.T) {
	ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
	leaf := item(4, "d")
	c := item(3, "c", kids(leaf))
	b := item(2, "b", kids(c))
	a := item(1, "a", kids(b))
	controller := NewTreeSliverController[treeItem]()
	ts.PumpWidget(treeApp(NewTreeSliver([]itemNode{a}, TreeSliverOpts[treeItem]{Controller: controller})))
	if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("rows = %v", got)
	}
	// A node with a whole subtree is moved to the root of a: its depth follows.
	a.AddChild(c)
	a.RemoveChild(b)
	ts.Pump()
	if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "c", "d"}) {
		t.Fatalf("rows after the move = %v", got)
	}
	if c.Depth() != foundation.Some(1) || leaf.Depth() != foundation.Some(2) || leaf.Parent() != c || c.Parent() != a {
		t.Errorf("after the move: c %v, d %v", c, leaf)
	}
	if b.Parent() != nil || b.Depth().IsSome() || controller.IsActive(b) {
		t.Errorf("b left the tree: parent %v, depth %v", b.Parent(), b.Depth())
	}
	// Removing an expanded folder takes its expanded subtree with it.
	a.RemoveChild(c)
	ts.Pump()
	if got := rows(ts, labels()...); !slices.Equal(got, []string{"a"}) || a.IsExpanded() {
		t.Errorf("rows after the removal = %v, a expanded = %v", got, a.IsExpanded())
	}
}

func TestTreeSliverNodeChangesDuringAnimations(t *testing.T) {
	tests := []struct {
		name   string
		change func(a, b itemNode)
		want   []string
	}{
		{"a folder that expands is removed", func(a, b itemNode) { a.RemoveChild(b) }, []string{"a"}},
		{"the children of a folder that expands are replaced", func(a, b itemNode) { b.SetChildren([]itemNode{item(9, "g")}) }, []string{"a", "b", "g"}},
		{"the children of a folder that expands are cleared", func(a, b itemNode) { b.SetChildren(nil) }, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := item(2, "b", collapsedFolder(item(3, "c"), item(4, "d")))
			a := item(1, "a", kids(b))
			ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{})
			controller.ToggleNode(b)
			ts.Pump()
			ts.Pump(40 * time.Millisecond) // in the middle of the animation
			tt.change(a, b)
			ts.Pump()
			ts.PumpAndSettle()
			if got := rows(ts, labels()...); !slices.Equal(got, tt.want) {
				t.Errorf("rows = %v, want %v", got, tt.want)
			}
		})
	}
}

// collapsedFolder makes the options of a folder that starts collapsed.
func collapsedFolder(children ...itemNode) TreeSliverNodeOpts[treeItem] {
	return TreeSliverNodeOpts[treeItem]{Children: children}
}

func TestTreeSliverLazyNodes(t *testing.T) {
	t.Run("the children are set in OnNodeToggle and show with the animation", func(t *testing.T) {
		folder := item(1, "a", TreeSliverNodeOpts[treeItem]{HasChildren: true})
		last := item(9, "h")
		var toggled []bool
		ts, controller := itemTree(t, []itemNode{folder, last}, TreeSliverOpts[treeItem]{OnNodeToggle: func(n itemNode) {
			toggled = append(toggled, n.IsExpanded())
			if n.IsExpanded() && len(n.Children()) == 0 {
				n.SetChildren([]itemNode{item(2, "b"), item(3, "c")})
			}
		}})
		if !folder.HasChildren() || folder.IsExpanded() || len(folder.Children()) != 0 {
			t.Fatalf("the lazy node: %v", folder)
		}
		if n := ts.Count(gluttertest.ByType[*AnimatedRotation]()); n != 1 {
			t.Errorf("arrows = %d, want the one of the lazy node", n)
		}
		start := ts.TopLeft(gluttertest.ByText("h")).Dy
		controller.ToggleNode(folder)
		ts.Pump()
		ts.Pump(75 * time.Millisecond)
		if mid := ts.TopLeft(gluttertest.ByText("h")).Dy; mid <= start || mid >= start+80 {
			t.Errorf("half way h is at %v: it moves from %v to %v", mid, start, start+80)
		}
		ts.PumpAndSettle()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c", "h"}) {
			t.Errorf("rows = %v", got)
		}
		if !slices.Equal(toggled, []bool{true}) || !folder.IsExpanded() || folder.Children()[0].Depth() != foundation.Some(1) {
			t.Errorf("toggled = %v, expanded = %v", toggled, folder.IsExpanded())
		}
		// It collapses like any folder, and keeps its children for the next opening.
		controller.ToggleNode(folder)
		ts.PumpAndSettle()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "h"}) || folder.IsExpanded() || len(folder.Children()) != 2 {
			t.Errorf("rows after the collapse = %v, expanded = %v", got, folder.IsExpanded())
		}
	})
	t.Run("the children come after the toggle", func(t *testing.T) {
		folder := item(1, "a", TreeSliverNodeOpts[treeItem]{HasChildren: true})
		toggles := 0
		ts, controller := itemTree(t, []itemNode{folder, item(9, "h")}, TreeSliverOpts[treeItem]{OnNodeToggle: func(itemNode) { toggles++ }})
		controller.ToggleNode(folder)
		ts.PumpAndSettle()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "h"}) || !folder.IsExpanded() || toggles != 1 {
			t.Fatalf("rows = %v, expanded = %v, toggles = %d", got, folder.IsExpanded(), toggles)
		}
		folder.SetChildren([]itemNode{item(2, "b")})
		ts.PumpAndSettle()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "h"}) {
			t.Errorf("rows after the children came = %v", got)
		}
	})
	t.Run("a lazy node can start expanded", func(t *testing.T) {
		folder := item(1, "a", TreeSliverNodeOpts[treeItem]{HasChildren: true, Expanded: true})
		ts, _ := itemTree(t, []itemNode{folder}, TreeSliverOpts[treeItem]{})
		if !folder.IsExpanded() {
			t.Error("a lazy node that starts expanded is not")
		}
		folder.AddChild(item(2, "b"))
		ts.Pump()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b"}) {
			t.Errorf("rows = %v", got)
		}
	})
	t.Run("a leaf is not toggled", func(t *testing.T) {
		leaf := item(1, "a")
		toggles := 0
		ts, controller := itemTree(t, []itemNode{leaf}, TreeSliverOpts[treeItem]{OnNodeToggle: func(itemNode) { toggles++ }})
		controller.ToggleNode(leaf)
		ts.Pump()
		if toggles != 0 || leaf.IsExpanded() || leaf.HasChildren() {
			t.Errorf("a leaf: toggles = %d, expanded = %v", toggles, leaf.IsExpanded())
		}
		if n := ts.Count(gluttertest.ByType[*AnimatedRotation]()); n != 0 {
			t.Errorf("a leaf has %d arrows", n)
		}
	})
}

func TestTreeSliverNodeKey(t *testing.T) {
	key := func(i treeItem) any { return i.ID }
	t.Run("GetNodeFor and IsExpanded", func(t *testing.T) {
		tests := []struct {
			name    string
			nodeKey func(treeItem) any
			probe   treeItem
			want    bool // found
		}{
			{"the same id with another label, with a key", key, treeItem{ID: 3, Label: "renamed"}, true},
			{"the same id with another label, without a key", nil, treeItem{ID: 3, Label: "renamed"}, false},
			{"the same content, without a key", nil, treeItem{ID: 3, Label: "c"}, true},
			{"an id that is not there, with a key", key, treeItem{ID: 7, Label: "c"}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := item(3, "c")
				a := item(1, "a", kids(item(2, "b", kids(c))))
				_, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{NodeKey: tt.nodeKey})
				got := controller.GetNodeFor(tt.probe)
				if (got == c) != tt.want || (got != nil && got != c) {
					t.Errorf("GetNodeFor(%v) = %v, want found %v", tt.probe, got, tt.want)
				}
				// IsExpanded finds the node by the content as well: a copy of the folder that is not
				// the node counts as the node when the keys are the same.
				copyOfA := item(1, "other label")
				if got := controller.IsExpanded(copyOfA); got != (tt.nodeKey != nil) {
					t.Errorf("IsExpanded(a copy of a) = %v, want %v", got, tt.nodeKey != nil)
				}
			})
		}
	})
	t.Run("the expansion survives a tree of new nodes", func(t *testing.T) {
		build := func() []itemNode {
			return []itemNode{
				item(1, "a", kids(item(2, "b"))),
				item(3, "c", TreeSliverNodeOpts[treeItem]{Children: []itemNode{item(4, "d")}}),
				item(5, "e", TreeSliverNodeOpts[treeItem]{Children: []itemNode{item(6, "f")}, Expanded: true}),
			}
		}
		tests := []struct {
			name     string
			nodeKey  func(treeItem) any
			wantRows []string
		}{
			// Before: a is expanded, c is expanded by the test, e is collapsed by the test. The new tree has
			// the flags of build(): a expanded, c collapsed, e expanded.
			{"with a key the old state is kept", key, []string{"a", "b", "c", "d", "e"}},
			{"without a key the new nodes have their own", nil, []string{"a", "b", "c", "e", "f"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
				controller := NewTreeSliverController[treeItem]()
				first := build()
				pump := func(nodes []itemNode) {
					ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[treeItem]{Controller: controller, NodeKey: tt.nodeKey})))
				}
				pump(first)
				controller.ToggleNode(first[1]) // c opens
				controller.ToggleNode(first[2]) // e closes
				ts.PumpAndSettle()
				if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c", "d", "e"}) {
					t.Fatalf("rows before = %v", got)
				}
				second := build()
				pump(second)
				ts.PumpAndSettle()
				if got := rows(ts, labels()...); !slices.Equal(got, tt.wantRows) {
					t.Errorf("rows after = %v, want %v", got, tt.wantRows)
				}
				// The nodes of the new tree are the ones that answer.
				if n := controller.GetNodeFor(treeItem{ID: 3, Label: "c"}); n != second[1] {
					t.Errorf("GetNodeFor finds %v, which is not the new node", n)
				}
			})
		}
	})
	t.Run("a node that is new keeps its own state, and the nodes that went take their animation", func(t *testing.T) {
		b := item(2, "b", TreeSliverNodeOpts[treeItem]{Children: []itemNode{item(3, "c")}})
		ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
		controller := NewTreeSliverController[treeItem]()
		pump := func(nodes []itemNode) {
			ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[treeItem]{Controller: controller, NodeKey: key})))
		}
		pump([]itemNode{b})
		controller.ToggleNode(b)
		ts.Pump()
		ts.Pump(40 * time.Millisecond) // b is opening
		pump([]itemNode{item(7, "g", TreeSliverNodeOpts[treeItem]{Children: []itemNode{item(8, "h")}, Expanded: true})})
		ts.PumpAndSettle()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"g", "h"}) {
			t.Errorf("rows = %v", got)
		}
	})
}
