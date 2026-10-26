package widgets_test

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Step 28.17: TreeSliverNode.SetExpanded.

// plainRow builds the row of a node as its text only: the default builder has an arrow that turns with its own
// animation, which would keep a frame scheduled whatever the rows do.
func plainRow(_ BuildContext, node *TreeSliverNode[treeItem], _ animation.AnimationStyle) Widget {
	return NewText(node.Content().Label)
}

func hasScheduledFrame(ts *gluttertest.Tester) bool {
	return ts.Binding.Scheduler().HasScheduledFrame()
}

func TestTreeSliverNodeSetExpandedOfAHiddenNode(t *testing.T) {
	// a and b are collapsed folders, b is inside a. b is expanded while it is hidden.
	d := item(4, "d")
	c := item(3, "c", collapsedFolder(d))
	b := item(2, "b", collapsedFolder(c))
	a := item(1, "a", collapsedFolder(b))
	ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow})
	c.SetExpanded(true)
	b.SetExpanded(true)
	ts.Pump()
	if hasScheduledFrame(ts) {
		t.Errorf("a frame is scheduled for a node that is not shown")
	}
	if got := rows(ts, labels()...); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("rows = %v", got)
	}
	if !b.IsExpanded() || !c.IsExpanded() || controller.IsActive(b) {
		t.Errorf("b expanded %v, c expanded %v, b active %v", b.IsExpanded(), c.IsExpanded(), controller.IsActive(b))
	}
	controller.ToggleNode(a)
	ts.PumpAndSettle()
	if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("rows after a opened = %v", got)
	}
	if d.Parent() != c || d.Depth() != foundation.Some(3) || c.Parent() != b || b.Parent() != a {
		t.Errorf("parents and depths: b %v, c %v, d %v", b, c, d)
	}
}

func TestTreeSliverNodeSetExpandedOfAShownNode(t *testing.T) {
	tests := []struct {
		name     string
		start    bool // b starts expanded
		set      bool
		wantRows []string
	}{
		{"expand", false, true, []string{"a", "b", "c", "d", "e"}},
		{"collapse", true, false, []string{"a", "b", "e"}},
		{"expand an expanded one", true, true, []string{"a", "b", "c", "d", "e"}},
		{"collapse a collapsed one", false, false, []string{"a", "b", "e"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, d, e := item(3, "c"), item(4, "d"), item(5, "e")
			b := item(2, "b", TreeSliverNodeOpts[treeItem]{Children: []itemNode{c, d}, Expanded: tt.start})
			a := item(1, "a", kids(b, e))
			ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow})
			ts.Pump()
			ey := ts.TopLeft(gluttertest.ByText("e")).Dy
			b.SetExpanded(tt.set)
			ts.Pump() // one frame: nothing animates
			if hasScheduledFrame(ts) {
				t.Errorf("a frame is still scheduled, so something animates")
			}
			if got := rows(ts, labels()...); !slices.Equal(got, tt.wantRows) {
				t.Fatalf("rows = %v, want %v", got, tt.wantRows)
			}
			if b.IsExpanded() != tt.set {
				t.Errorf("b expanded = %v", b.IsExpanded())
			}
			// The row below is at its final place at once: two rows of 40 for c and d.
			want := ey
			if tt.set && !tt.start {
				want += 80
			} else if !tt.set && tt.start {
				want -= 80
			}
			if got := ts.TopLeft(gluttertest.ByText("e")).Dy; got != want {
				t.Errorf("e is at %v, want %v", got, want)
			}
			if tt.set {
				for _, n := range []itemNode{c, d} {
					if n.Parent() != b || n.Depth() != foundation.Some(2) || !controller.IsActive(n) {
						t.Errorf("%v: parent %v, depth %v, active %v", n, n.Parent(), n.Depth(), controller.IsActive(n))
					}
				}
			} else if controller.IsActive(c) || controller.IsActive(d) {
				t.Errorf("c or d is active after b collapsed")
			}
			if e.Parent() != a || e.Depth() != foundation.Some(1) {
				t.Errorf("e: parent %v, depth %v", e.Parent(), e.Depth())
			}
		})
	}
}

func TestTreeSliverNodeSetExpandedEndsTheAnimationOfTheNode(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		wantRows []string
	}{
		{"expand while it collapses", true, []string{"a", "b", "c", "d"}},
		{"collapse while it expands", false, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := item(2, "b", openFolder(tt.set, item(3, "c"), item(4, "d")))
			a := item(1, "a", kids(b))
			ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow})
			controller.ToggleNode(b) // the opposite of set
			ts.Pump()
			ts.Pump(40 * time.Millisecond) // in the middle of the animation
			if !hasScheduledFrame(ts) {
				t.Fatalf("the toggle does not animate")
			}
			b.SetExpanded(tt.set)
			ts.Pump()
			if hasScheduledFrame(ts) {
				t.Errorf("the animation of b goes on")
			}
			if got := rows(ts, labels()...); !slices.Equal(got, tt.wantRows) {
				t.Errorf("rows = %v, want %v", got, tt.wantRows)
			}
		})
	}
}

// openFolder makes the options of a folder with the given expansion.
func openFolder(expanded bool, children ...itemNode) TreeSliverNodeOpts[treeItem] {
	return TreeSliverNodeOpts[treeItem]{Children: children, Expanded: expanded}
}

func TestTreeSliverNodeSetExpandedOfALeafAndALazyNode(t *testing.T) {
	t.Run("a leaf stays collapsed", func(t *testing.T) {
		leaf := item(2, "b")
		a := item(1, "a", kids(leaf))
		ts, _ := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow})
		leaf.SetExpanded(true)
		ts.Pump()
		if leaf.IsExpanded() {
			t.Errorf("a leaf is expanded")
		}
		// Not even when it gets children later: it is a node that was collapsed.
		leaf.AddChild(item(3, "c"))
		ts.Pump()
		if leaf.IsExpanded() {
			t.Errorf("a leaf with a new child is expanded")
		}
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b"}) {
			t.Errorf("rows = %v", got)
		}
		// With a child it expands.
		leaf.SetExpanded(true)
		ts.Pump()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c"}) {
			t.Errorf("rows after the expansion = %v", got)
		}
	})
	t.Run("a lazy node expands, and its children show when they come", func(t *testing.T) {
		toggles := 0
		lazy := item(2, "b", TreeSliverNodeOpts[treeItem]{HasChildren: true})
		a := item(1, "a", kids(lazy))
		ts, _ := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow, OnNodeToggle: func(*TreeSliverNode[treeItem]) { toggles++ }})
		lazy.SetExpanded(true)
		ts.Pump()
		if !lazy.IsExpanded() {
			t.Fatalf("the lazy node is not expanded")
		}
		if toggles != 0 {
			t.Errorf("OnNodeToggle was called %d times by SetExpanded", toggles)
		}
		lazy.SetChildren([]itemNode{item(3, "c")})
		ts.Pump()
		if hasScheduledFrame(ts) {
			t.Errorf("the children of an expanded node animate")
		}
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c"}) {
			t.Errorf("rows = %v", got)
		}
		lazy.SetExpanded(false)
		ts.Pump()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b"}) {
			t.Errorf("rows after the collapse = %v", got)
		}
	})
}

func TestTreeSliverNodeSetExpandedToTheSameValueRebuildsNothing(t *testing.T) {
	builds := 0
	b := item(2, "b", collapsedFolder(item(3, "c")))
	a := item(1, "a", kids(b))
	ts, _ := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{
		TreeNodeBuilder: func(bc BuildContext, node *TreeSliverNode[treeItem], style animation.AnimationStyle) Widget {
			builds++
			return plainRow(bc, node, style)
		},
	})
	ts.Pump()
	b.SetExpanded(false) // it is collapsed already
	ts.Pump()
	if hasScheduledFrame(ts) {
		t.Errorf("a frame is scheduled")
	}
	before := builds
	b.SetExpanded(true)
	ts.Pump()
	if builds == before {
		t.Fatalf("expanding b built no row")
	}
	before = builds
	b.SetExpanded(true)
	b.SetExpanded(true)
	ts.Pump()
	if hasScheduledFrame(ts) || builds != before {
		t.Errorf("the same value twice: %d rows built, frame scheduled %v", builds-before, hasScheduledFrame(ts))
	}
}

func TestTreeSliverControllerStillAnimates(t *testing.T) {
	tests := []struct {
		name string
		do   func(c *TreeSliverController[treeItem], b itemNode)
		open bool // b starts expanded
	}{
		{"ExpandNode", func(c *TreeSliverController[treeItem], b itemNode) { c.ExpandNode(b) }, false},
		{"CollapseNode", func(c *TreeSliverController[treeItem], b itemNode) { c.CollapseNode(b) }, true},
		{"ToggleNode", func(c *TreeSliverController[treeItem], b itemNode) { c.ToggleNode(b) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := item(2, "b", openFolder(tt.open, item(3, "c"), item(4, "d")))
			e := item(5, "e")
			a := item(1, "a", kids(b, e))
			ts, controller := itemTree(t, []itemNode{a}, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow})
			start := ts.TopLeft(gluttertest.ByText("e")).Dy
			tt.do(controller, b)
			ts.Pump()
			ts.Pump(75 * time.Millisecond)
			if !hasScheduledFrame(ts) {
				t.Fatalf("the toggle ends in one frame")
			}
			mid := ts.TopLeft(gluttertest.ByText("e")).Dy
			final := start + 80
			if tt.open {
				final = start - 80
			}
			if mid == start || mid == final {
				t.Errorf("e is at %v half way (from %v to %v)", mid, start, final)
			}
			ts.PumpAndSettle()
			if got := ts.TopLeft(gluttertest.ByText("e")).Dy; got != final {
				t.Errorf("e ends at %v, want %v", got, final)
			}
		})
	}
}

// modelTree makes the nodes of a tree from a model that holds the expansion by id, as Emetos does at each change:
// a(1) { b(2) { c(3) }, d(4) }, e(5) { f(6) }.
func modelTree(set func(n itemNode)) []itemNode {
	node := func(id int, label string, children ...itemNode) itemNode {
		n := item(id, label, TreeSliverNodeOpts[treeItem]{Children: children})
		if set != nil && len(children) > 0 {
			set(n)
		}
		return n
	}
	return []itemNode{
		node(1, "a", node(2, "b", node(3, "c")), node(4, "d")),
		node(5, "e", node(6, "f")),
	}
}

func TestTreeSliverSetExpandedWinsOverNodeKey(t *testing.T) {
	key := func(i treeItem) any { return i.ID }
	// Every node of the model is set; ids of the folders: 1, 2, 5.
	apply := func(model map[int]bool) func(n itemNode) {
		return func(n itemNode) { n.SetExpanded(model[n.Content().ID]) }
	}
	tests := []struct {
		name    string
		nodeKey func(treeItem) any
		before  map[int]bool
		after   map[int]bool
		// setOnly is the list of ids whose nodes the app sets in the second tree; nil is all of them.
		setOnly  []int
		wantRows []string
	}{
		{
			name: "the model wins over the old nodes", nodeKey: key,
			before: map[int]bool{1: true, 2: true, 5: true}, after: map[int]bool{1: true, 2: false, 5: true},
			wantRows: []string{"a", "b", "d", "e", "f"},
		},
		{
			name: "a model that opens what was closed", nodeKey: key,
			before: map[int]bool{1: true, 2: false}, after: map[int]bool{1: true, 2: true, 5: true},
			wantRows: []string{"a", "b", "c", "d", "e", "f"},
		},
		{
			name: "a model that closes everything", nodeKey: key,
			before: map[int]bool{1: true, 2: true, 5: true}, after: map[int]bool{},
			wantRows: []string{"a", "e"},
		},
		{
			name: "the nodes the app does not set keep the old expansion", nodeKey: key,
			before: map[int]bool{1: true, 2: true, 5: true}, after: map[int]bool{1: true, 2: false, 5: false}, setOnly: []int{2},
			wantRows: []string{"a", "b", "d", "e", "f"},
		},
		{
			name: "without a key the model is the only state", nodeKey: nil,
			before: map[int]bool{1: true, 2: true, 5: true}, after: map[int]bool{1: true, 2: false, 5: true},
			wantRows: []string{"a", "b", "d", "e", "f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
			controller := NewTreeSliverController[treeItem]()
			pump := func(nodes []itemNode) {
				ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow, Controller: controller, NodeKey: tt.nodeKey})))
				ts.PumpAndSettle()
			}
			pump(modelTree(apply(tt.before)))
			second := modelTree(func(n itemNode) {
				if tt.setOnly == nil || slices.Contains(tt.setOnly, n.Content().ID) {
					n.SetExpanded(tt.after[n.Content().ID])
				}
			})
			pump(second)
			if got := rows(ts, labels()...); !slices.Equal(got, tt.wantRows) {
				t.Errorf("rows = %v, want %v", got, tt.wantRows)
			}
			// The nodes of the new tree are the ones that answer.
			if n := controller.GetNodeFor(treeItem{ID: 2, Label: "b"}); tt.nodeKey != nil && n != second[0].Children()[0] {
				t.Errorf("GetNodeFor finds %v, which is not the new node", n)
			}
		})
	}
	t.Run("a node set after the tree took it in wins as well", func(t *testing.T) {
		ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
		controller := NewTreeSliverController[treeItem]()
		pump := func(nodes []itemNode) {
			ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow, Controller: controller, NodeKey: key})))
			ts.PumpAndSettle()
		}
		model := map[int]bool{1: true, 2: true}
		pump(modelTree(apply(model)))
		second := modelTree(nil)
		pump(second) // the old expansion is carried over
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "c", "d", "e"}) {
			t.Fatalf("rows with the carried expansion = %v", got)
		}
		second[0].Children()[0].SetExpanded(false)
		ts.Pump()
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "b", "d", "e"}) {
			t.Errorf("rows = %v", got)
		}
	})
	t.Run("the mark ends when a tree takes the node in", func(t *testing.T) {
		ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 600}})
		controller := NewTreeSliverController[treeItem]()
		pump := func(nodes []itemNode) {
			ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[treeItem]{TreeNodeBuilder: plainRow, Controller: controller, NodeKey: key})))
			ts.PumpAndSettle()
		}
		setTo := func(v bool) func(itemNode) { return func(n itemNode) { n.SetExpanded(v) } }
		first := modelTree(setTo(true))
		pump(first)
		pump(modelTree(setTo(false))) // the app says: all closed
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "e"}) {
			t.Fatalf("rows with the model = %v", got)
		}
		// The first nodes are used again, with the expansion they were given when they were new: the tree took
		// them in, so the expansion of the nodes that are in the tree now (all closed) is carried to them.
		pump(first)
		if got := rows(ts, labels()...); !slices.Equal(got, []string{"a", "e"}) {
			t.Errorf("rows with the first nodes again = %v", got)
		}
	})
}
