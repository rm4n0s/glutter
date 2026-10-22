package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
	"golang.org/x/image/font/gofont/gomono"
)

// These tests are ported from packages/flutter/test/widgets/sliver_tree_test.dart. Not ported: the test of
// TreeSliver together with PinnedHeaderSliver, which Glutter does not have yet; the golden of the scrolling
// test is a golden of this repository (testdata/goldens/sliver_tree_scrolling.png).

// simpleNodeSet is the tree most of the tests use; each call makes a new one.
func simpleNodeSet() []*TreeSliverNode[string] {
	n := func(s string, opts ...TreeSliverNodeOpts[string]) *TreeSliverNode[string] {
		return NewTreeSliverNode(s, opts...)
	}
	return []*TreeSliverNode[string]{
		n("Root 0"),
		n("Root 1", TreeSliverNodeOpts[string]{Expanded: true, Children: []*TreeSliverNode[string]{n("Child 1:0"), n("Child 1:1")}}),
		n("Root 2", TreeSliverNodeOpts[string]{Children: []*TreeSliverNode[string]{n("Child 2:0"), n("Child 2:1")}}),
		n("Root 3"),
	}
}

// newTreeTester is a tester whose default font is Go Mono: the engine of a test has Roboto and Material Icons only
// and no font for a style that names no family, so the text of the default node builder, and the arrow
// (U+25BA, which Go Mono has), would have no width and a tap on them would miss. A font loaded under "sans-serif" is
// the default font.
func newTreeTester(t *testing.T, opts ...gluttertest.Opts) *gluttertest.Tester {
	t.Helper()
	var o gluttertest.Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Fonts = []gluttertest.Font{{Family: "sans-serif", Data: gomono.TTF}}
	return gluttertest.New(t, o)
}

// treeApp shows the slivers in a scroll view.
func treeApp(slivers ...Widget) Widget {
	return treeText(NewCustomScrollView(slivers))
}

func treeText(child Widget) Widget {
	return ltr(NewDefaultTextStyle(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(14.0)}), child))
}

func treeCount(t *testing.T, ts *gluttertest.Tester, text string, want int) {
	t.Helper()
	if got := ts.Count(gluttertest.ByText(text)); got != want {
		t.Errorf("%q is found %d times, want %d", text, got, want)
	}
}

func TestTreeSliverNode(t *testing.T) {
	t.Run("getters, String", func(t *testing.T) {
		children := []*TreeSliverNode[string]{NewTreeSliverNode("child")}
		node := NewTreeSliverNode("parent", TreeSliverNodeOpts[string]{Children: children, Expanded: true})
		first := node.Children()[0]
		if node.Content() != "parent" || len(node.Children()) != 1 || first != children[0] || !node.IsExpanded() {
			t.Errorf("node = %v", node)
		}
		if first.Content() != "child" || len(first.Children()) != 0 || first.IsExpanded() {
			t.Errorf("child = %v", first)
		}
		// Set by TreeSliver when built, for tree integrity.
		if node.Depth().IsSome() || node.Parent() != nil || first.Depth().IsSome() || first.Parent() != nil {
			t.Error("depth and parent are set before the node was built")
		}
		if got, want := node.String(), "TreeSliverNode: parent, depth: null, parent, expanded: true"; got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
		if got, want := first.String(), "TreeSliverNode: child, depth: null, leaf"; got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
	t.Run("a node without children is never expanded", func(t *testing.T) {
		if NewTreeSliverNode("leaf", TreeSliverNodeOpts[string]{Expanded: true}).IsExpanded() {
			t.Error("a leaf is expanded")
		}
	})
	t.Run("TreeSliverNode sets up parent and depth properties", func(t *testing.T) {
		ts := newTreeTester(t)
		children := []*TreeSliverNode[string]{NewTreeSliverNode("child")}
		node := NewTreeSliverNode("parent", TreeSliverNodeOpts[string]{Children: children, Expanded: true})
		ts.PumpWidget(treeApp(NewTreeSliver([]*TreeSliverNode[string]{node})))
		first := node.Children()[0]
		if node.Depth() != foundation.Some(0) || node.Parent() != nil || first.Depth() != foundation.Some(1) || first.Parent() != node {
			t.Errorf("depth and parent after the build: %v, %v", node, first)
		}
		if got, want := node.String(), "TreeSliverNode: parent, depth: root, parent, expanded: true"; got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
		if got, want := first.String(), "TreeSliverNode: child, depth: 1, leaf"; got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
}

func TestTreeSliverController(t *testing.T) {
	// pump shows a tree with the controller.
	pump := func(t *testing.T) (*gluttertest.Tester, *TreeSliverController[string], []*TreeSliverNode[string]) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		nodes := simpleNodeSet()
		ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{Controller: controller})))
		return ts, controller, nodes
	}
	t.Run("Can set controller on TreeSliver", func(t *testing.T) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		var returned *TreeSliverController[string]
		ts.PumpWidget(treeApp(NewTreeSliver(simpleNodeSet(), TreeSliverOpts[string]{
			Controller: controller,
			TreeNodeBuilder: func(bc BuildContext, node *TreeSliverNode[string], style animation.AnimationStyle) Widget {
				if returned == nil {
					returned = TreeSliverControllerOf[string](bc)
				}
				return TreeSliverDefaultTreeNodeBuilder(bc, node, style)
			},
		})))
		if returned != controller {
			t.Error("the controller below the tree is not the one that was given")
		}
	})
	t.Run("Can get default controller on TreeSliver", func(t *testing.T) {
		ts := newTreeTester(t)
		var returned *TreeSliverController[string]
		ts.PumpWidget(treeApp(NewTreeSliver(simpleNodeSet(), TreeSliverOpts[string]{
			TreeNodeBuilder: func(bc BuildContext, node *TreeSliverNode[string], style animation.AnimationStyle) Widget {
				if returned == nil {
					returned, _ = TreeSliverControllerMaybeOf[string](bc)
				}
				return TreeSliverDefaultTreeNodeBuilder(bc, node, style)
			},
		})))
		if returned == nil {
			t.Error("the tree has no controller of its own")
		}
	})
	t.Run("Can get node for TreeSliverNode.content", func(t *testing.T) {
		_, controller, nodes := pump(t)
		if got := controller.GetNodeFor("Root 0"); got != nodes[0] {
			t.Errorf("GetNodeFor(Root 0) = %v", got)
		}
		if got := controller.GetNodeFor("Child 1:1"); got != nodes[1].Children()[1] {
			t.Errorf("GetNodeFor(Child 1:1) = %v", got)
		}
		if got := controller.GetNodeFor("Nobody"); got != nil {
			t.Errorf("GetNodeFor(Nobody) = %v, want nil", got)
		}
	})
	t.Run("Can get isExpanded for a node", func(t *testing.T) {
		_, controller, nodes := pump(t)
		if controller.IsExpanded(nodes[0]) || !controller.IsExpanded(nodes[1]) {
			t.Errorf("IsExpanded: %v %v", controller.IsExpanded(nodes[0]), controller.IsExpanded(nodes[1]))
		}
	})
	t.Run("Can get isActive for a node", func(t *testing.T) {
		_, controller, nodes := pump(t)
		if !controller.IsActive(nodes[0]) || !controller.IsActive(nodes[1]) {
			t.Error("the root nodes are active")
		}
		// The parent 'Root 2' is not expanded, so its children are not active.
		if controller.IsExpanded(nodes[2]) || controller.IsActive(nodes[2].Children()[0]) {
			t.Error("the children of a collapsed node are active")
		}
		if got := controller.GetActiveIndexFor(nodes[3]); got != foundation.Some(5) {
			t.Errorf("GetActiveIndexFor(Root 3) = %v, want 5", got)
		}
		if got := controller.GetActiveIndexFor(nodes[2].Children()[0]); got.IsSome() {
			t.Errorf("GetActiveIndexFor(Child 2:0) = %v, want none", got)
		}
	})
	t.Run("Can toggleNode, to collapse or expand", func(t *testing.T) {
		ts, controller, nodes := pump(t)
		if controller.IsExpanded(nodes[2]) || controller.IsActive(nodes[2].Children()[0]) {
			t.Fatal("Root 2 starts collapsed")
		}
		controller.ToggleNode(nodes[2]) // expand
		if !controller.IsExpanded(nodes[2]) || !controller.IsActive(nodes[2].Children()[0]) {
			t.Error("Root 2 did not expand")
		}
		if !controller.IsExpanded(nodes[1]) || !controller.IsActive(nodes[1].Children()[0]) {
			t.Fatal("Root 1 starts expanded")
		}
		controller.ToggleNode(nodes[1]) // collapse
		if controller.IsExpanded(nodes[1]) || !controller.IsActive(nodes[1].Children()[0]) {
			t.Error("the children of a collapsing node stay active until the animation is over")
		}
		ts.PumpAndSettle()
		if controller.IsExpanded(nodes[1]) || controller.IsActive(nodes[1].Children()[0]) {
			t.Error("the children of Root 1 are still active")
		}
	})
	t.Run("Can expandNode, then collapseAll", func(t *testing.T) {
		ts, controller, nodes := pump(t)
		controller.ExpandNode(nodes[2])
		if !controller.IsExpanded(nodes[2]) || !controller.IsActive(nodes[2].Children()[0]) {
			t.Error("Root 2 did not expand")
		}
		controller.ExpandNode(nodes[2]) // already expanded: nothing
		if !controller.IsExpanded(nodes[2]) {
			t.Error("ExpandNode of an expanded node collapsed it")
		}
		if !controller.IsExpanded(nodes[1]) || !controller.IsExpanded(nodes[2]) {
			t.Fatal("both parents are expanded")
		}
		controller.CollapseAll()
		ts.PumpAndSettle()
		if controller.IsExpanded(nodes[1]) || controller.IsExpanded(nodes[2]) {
			t.Error("a parent is still expanded after CollapseAll")
		}
	})
	t.Run("Can collapseNode, then expandAll", func(t *testing.T) {
		ts, controller, nodes := pump(t)
		if !controller.IsExpanded(nodes[1]) || !controller.IsActive(nodes[1].Children()[0]) {
			t.Fatal("Root 1 starts expanded")
		}
		controller.CollapseNode(nodes[1])
		if controller.IsExpanded(nodes[1]) || !controller.IsActive(nodes[1].Children()[0]) {
			t.Error("the children of a collapsing node stay active until the animation is over")
		}
		ts.PumpAndSettle()
		if controller.IsActive(nodes[1].Children()[0]) {
			t.Error("the children of Root 1 are still active")
		}
		controller.CollapseNode(nodes[1]) // already collapsed: nothing
		if controller.IsExpanded(nodes[1]) || controller.IsExpanded(nodes[2]) {
			t.Fatal("both parents are collapsed")
		}
		controller.ExpandAll()
		if !controller.IsExpanded(nodes[1]) || !controller.IsExpanded(nodes[2]) {
			t.Error("a parent is not expanded after ExpandAll")
		}
		ts.PumpAndSettle()
		treeCount(t, ts, "Child 1:0", 1)
		treeCount(t, ts, "Child 2:1", 1)
	})
}

func TestTreeSliverControllerFailures(t *testing.T) {
	t.Run("a controller that is not attached", func(t *testing.T) {
		defer func() {
			appErr, ok := errors.FromError(asError(recover()))
			if !ok || appErr.Tag != "TreeSliverControllerNotAttached" {
				t.Errorf("panic = %v, want TreeSliverControllerNotAttached", appErr)
			}
		}()
		NewTreeSliverController[string]().IsExpanded(NewTreeSliverNode("x"))
		t.Error("no panic")
	})
	t.Run("a controller of another tree", func(t *testing.T) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		ts.PumpWidget(treeApp(NewTreeSliver(simpleNodeSet(), TreeSliverOpts[string]{Controller: controller}),
			NewTreeSliver(simpleNodeSet(), TreeSliverOpts[string]{Controller: controller})))
		err := ts.TakeError()
		if err == nil {
			t.Fatal("two trees with one controller reported nothing")
		}
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "TreeSliverControllerAttached" {
			t.Errorf("error = %v, want TreeSliverControllerAttached", err)
		}
	})
	t.Run("a node that is not active", func(t *testing.T) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		nodes := simpleNodeSet()
		ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{Controller: controller})))
		defer func() {
			appErr, ok := errors.FromError(asError(recover()))
			if !ok || appErr.Tag != "TreeSliverNodeNotActive" {
				t.Errorf("panic = %v, want TreeSliverNodeNotActive", appErr)
			}
		}()
		controller.ToggleNode(nodes[2].Children()[0])
		t.Error("no panic")
	})
	t.Run("Of without a tree", func(t *testing.T) {
		ts := newTreeTester(t)
		var got any
		ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) (w Widget) {
			w = NewSizedBox(nil)
			defer func() { got = recover() }()
			TreeSliverControllerOf[string](bc)
			return w
		})))
		appErr, ok := errors.FromError(asError(got))
		if !ok || appErr.Tag != "TreeSliverControllerNotFound" {
			t.Errorf("panic = %v, want TreeSliverControllerNotFound", got)
		}
	})
}

func asError(v any) error {
	err, _ := v.(error)
	return err
}

func TestTreeSliverIndentationTypeValues(t *testing.T) {
	tests := []struct {
		name string
		got  rendering.TreeSliverIndentationType
		want float64
	}{
		{"standard", rendering.TreeSliverIndentationTypeStandard, 10.0},
		{"none", rendering.TreeSliverIndentationTypeNone, 0.0},
		{"custom", rendering.TreeSliverIndentationTypeCustom(50.0), 50.0},
	}
	for _, tt := range tests {
		if got := tt.got.Value(); got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
		}
	}
	defer func() {
		appErr, ok := errors.FromError(asError(recover()))
		if !ok || appErr.Tag != "TreeSliverIndentationNegative" {
			t.Errorf("panic = %v, want TreeSliverIndentationNegative", appErr)
		}
	}()
	rendering.TreeSliverIndentationTypeCustom(-1)
	t.Error("no panic")
}

func TestTreeSliverToggleNodeWithOnNodeToggle(t *testing.T) {
	ts := newTreeTester(t)
	nodes := simpleNodeSet()
	controller := NewTreeSliverController[string]()
	toggled := false
	var toggledNode *TreeSliverNode[string]
	onToggle := func(node *TreeSliverNode[string]) { toggled, toggledNode = true, node }
	// The default node builder wraps the leading icon with TreeSliverWrapChildToToggleNode.
	ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{Controller: controller, OnNodeToggle: onToggle})))
	if !controller.IsExpanded(nodes[1]) {
		t.Fatal("Root 1 starts expanded")
	}
	ts.Tap(gluttertest.First(gluttertest.ByType[*Icon]()))
	ts.Pump()
	if controller.IsExpanded(nodes[1]) || !toggled || toggledNode != nodes[1] {
		t.Errorf("after the tap on the icon: expanded = %v, toggled = %v, node = %v", controller.IsExpanded(nodes[1]), toggled, toggledNode)
	}
	ts.PumpAndSettle()
	if controller.IsExpanded(nodes[1]) {
		t.Error("Root 1 expanded again")
	}
	toggled, toggledNode = false, nil

	// TreeSliverWrapChildToToggleNode makes the whole row toggle the node.
	ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{
		Controller: controller, OnNodeToggle: onToggle,
		TreeNodeBuilder: func(_ BuildContext, node *TreeSliverNode[string], style animation.AnimationStyle) Widget {
			duration := style.Duration.OrElse(TreeSliverDefaultAnimationDuration)
			curve := style.Curve
			if curve == nil {
				curve = TreeSliverDefaultAnimationCurve
			}
			var arrow Widget
			if len(node.Children()) > 0 {
				turns := 0.0
				if node.IsExpanded() {
					turns = 0.25
				}
				arrow = NewAnimatedRotation(turns, duration, NewIcon(NewIconData(0x25BA), IconOpts{Size: foundation.Some(14.0)}), AnimatedRotationOpts{Curve: curve})
			}
			return TreeSliverWrapChildToToggleNode(node, NewPadding(painting.NewEdgeInsetsAll(8.0), NewRow([]Widget{
				NewSizedBoxSquare(foundation.Some(30.0), arrow), NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(8.0)}), NewText(node.Content()),
			})))
		},
	})))
	if controller.IsExpanded(nodes[1]) { // still collapsed from earlier
		t.Fatal("Root 1 is expanded")
	}
	// A tap on the text, not on the icon.
	ts.Tap(gluttertest.ByText("Root 1"))
	ts.Pump()
	if !controller.IsExpanded(nodes[1]) || !toggled || toggledNode != nodes[1] {
		t.Errorf("after the tap on the text: expanded = %v, toggled = %v, node = %v", controller.IsExpanded(nodes[1]), toggled, toggledNode)
	}
}

func TestTreeSliverAnimationStylePipedToTheNodeBuilder(t *testing.T) {
	ts := newTreeTester(t)
	nodes := simpleNodeSet()
	var style *animation.AnimationStyle
	build := func(opts TreeSliverOpts[string]) Widget {
		opts.TreeNodeBuilder = func(_ BuildContext, node *TreeSliverNode[string], s animation.AnimationStyle) Widget {
			style = &s
			return NewText(node.Content())
		}
		return treeApp(NewTreeSliver(nodes, opts))
	}
	// The default.
	ts.PumpWidget(build(TreeSliverOpts[string]{}))
	if style == nil || style.Duration != TreeSliverDefaultToggleAnimationStyle.Duration || style.Curve != TreeSliverDefaultToggleAnimationStyle.Curve {
		t.Errorf("default style = %+v, want %+v", style, TreeSliverDefaultToggleAnimationStyle)
	}
	// No animation.
	style = nil
	ts.PumpWidget(build(TreeSliverOpts[string]{ToggleAnimationStyle: foundation.Some(animation.AnimationStyleNoAnimation)}))
	if style == nil || style.Curve != nil || style.Duration != foundation.Some(time.Duration(0)) {
		t.Errorf("no animation style = %+v", style)
	}
	// Another curve and duration.
	style = nil
	ts.PumpWidget(build(TreeSliverOpts[string]{ToggleAnimationStyle: foundation.Some(animation.AnimationStyle{
		Curve: animation.Curves.EaseIn, Duration: foundation.Some(200 * time.Millisecond)})}))
	if style == nil || style.Curve != animation.Curves.EaseIn || style.Duration != foundation.Some(200*time.Millisecond) {
		t.Errorf("custom style = %+v", style)
	}
}

func TestTreeSliverReflectsAddedNodes(t *testing.T) {
	tests := []struct {
		name string
		// add changes the tree.
		add func(nodes []*TreeSliverNode[string]) []*TreeSliverNode[string]
		// label is the text that appears.
		label string
	}{
		{"more root nodes", func(nodes []*TreeSliverNode[string]) []*TreeSliverNode[string] {
			return append(nodes, NewTreeSliverNode("Added root"))
		}, "Added root"},
		{"more nodes below the root", func(nodes []*TreeSliverNode[string]) []*TreeSliverNode[string] {
			nodes[1].AddChild(NewTreeSliverNode("Added child"))
			return nodes
		}, "Added child"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTreeTester(t)
			nodes := simpleNodeSet()
			controller := NewTreeSliverController[string]()
			ts.PumpWidget(treeText(NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
				return NewColumn([]Widget{
					NewGestureDetector(NewText("Add"), GestureDetectorOpts{OnTap: func() {
						setState(func() { nodes = tt.add(nodes) })
					}}),
					NewExpanded(NewCustomScrollView([]Widget{NewTreeSliver(nodes, TreeSliverOpts[string]{Controller: controller})})),
				})
			})))
			ts.Pump()
			for text, want := range map[string]int{"Root 0": 1, "Root 1": 1, "Child 1:0": 1, "Child 1:1": 1, "Root 2": 1, "Child 2:0": 0, "Child 2:1": 0, "Root 3": 1, tt.label: 0} {
				treeCount(t, ts, text, want)
			}
			ts.Tap(gluttertest.ByText("Add"))
			ts.Pump()
			for text, want := range map[string]int{"Root 0": 1, "Root 1": 1, "Child 1:0": 1, "Child 1:1": 1, "Root 2": 1, "Child 2:0": 0, "Child 2:1": 0, "Root 3": 1, tt.label: 1} {
				treeCount(t, ts, text, want)
			}
		})
	}
}

func TestTreeSliverNodeClosesItsChildrenWhenCollapsed(t *testing.T) {
	newTree := func(expanded bool) []*TreeSliverNode[string] {
		n := func(s string, opts ...TreeSliverNodeOpts[string]) *TreeSliverNode[string] {
			return NewTreeSliverNode(s, opts...)
		}
		if expanded {
			return []*TreeSliverNode[string]{n("First", TreeSliverNodeOpts[string]{Expanded: true, Children: []*TreeSliverNode[string]{
				n("alpha", TreeSliverNodeOpts[string]{Expanded: true, Children: []*TreeSliverNode[string]{n("uno"), n("dos"), n("tres")}}),
				n("beta"), n("kappa"),
			}})}
		}
		return []*TreeSliverNode[string]{
			n("First"),
			n("Second", TreeSliverNodeOpts[string]{Children: []*TreeSliverNode[string]{
				n("alpha", TreeSliverNodeOpts[string]{Children: []*TreeSliverNode[string]{n("uno"), n("dos"), n("tres")}}),
				n("beta"), n("kappa"),
			}}),
			n("Third", TreeSliverNodeOpts[string]{Expanded: true, Children: []*TreeSliverNode[string]{n("gamma"), n("delta"), n("epsilon")}}),
			n("Fourth"),
		}
	}
	tappableRow := func(controller *TreeSliverController[string]) TreeSliverNodeBuilder[string] {
		return func(bc BuildContext, node *TreeSliverNode[string], style animation.AnimationStyle) Widget {
			return NewGestureDetector(TreeSliverDefaultTreeNodeBuilder(bc, node, style), GestureDetectorOpts{
				Key: foundation.NewValueKey(node.Content()), Behavior: rendering.HitTestBehaviorTranslucent,
				OnTap: func() { controller.ToggleNode(node) },
			})
		}
	}
	t.Run("when the animation is disabled", func(t *testing.T) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		ts.PumpWidget(treeApp(NewTreeSliver(newTree(false), TreeSliverOpts[string]{
			Controller: controller, ToggleAnimationStyle: foundation.Some(animation.AnimationStyleNoAnimation), TreeNodeBuilder: tappableRow(controller),
		})))
		for text, want := range map[string]int{"First": 1, "Second": 1, "Third": 1, "Fourth": 1, "alpha": 0, "beta": 0, "kappa": 0, "gamma": 1, "delta": 1, "epsilon": 1, "uno": 0, "dos": 0, "tres": 0} {
			treeCount(t, ts, text, want)
		}
		ts.Tap(gluttertest.ByText("Second"))
		ts.PumpAndSettle()
		treeCount(t, ts, "alpha", 1)
		ts.Tap(gluttertest.ByText("alpha"))
		ts.PumpAndSettle()
		for _, text := range []string{"uno", "dos", "tres"} {
			treeCount(t, ts, text, 1)
		}
		ts.Tap(gluttertest.ByText("alpha"))
		ts.PumpAndSettle()
		for _, text := range []string{"uno", "dos", "tres"} {
			treeCount(t, ts, text, 0)
		}
	})
	t.Run("when the animation is completed", func(t *testing.T) {
		ts := newTreeTester(t)
		controller := NewTreeSliverController[string]()
		ts.PumpWidget(treeText(NewCustomScrollView([]Widget{NewTreeSliver(newTree(true), TreeSliverOpts[string]{
			Controller: controller, TreeNodeBuilder: tappableRow(controller),
			ToggleAnimationStyle: foundation.Some(animation.AnimationStyle{Curve: animation.Curves.EaseInOut, Duration: foundation.Some(200 * time.Millisecond)}),
		})}, CustomScrollViewOpts{ShrinkWrap: true})))
		for _, text := range []string{"alpha", "uno", "dos", "tres"} {
			treeCount(t, ts, text, 1)
		}
		ts.Tap(gluttertest.ByText("alpha"))
		ts.PumpAndSettle()
		for _, text := range []string{"uno", "dos", "tres"} {
			treeCount(t, ts, text, 0)
		}
		treeCount(t, ts, "alpha", 1)
	})
}

func TestTreeSliverChildNodePositions(t *testing.T) {
	ts := newTreeTester(t)
	k0, k1 := foundation.NewValueKey(0), foundation.NewValueKey(1)
	tree := []*TreeSliverNode[foundation.Key]{NewTreeSliverNode[foundation.Key](k0, TreeSliverNodeOpts[foundation.Key]{
		Expanded: true, Children: []*TreeSliverNode[foundation.Key]{NewTreeSliverNode[foundation.Key](k1)},
	})}
	ts.PumpWidget(treeApp(NewTreeSliver(tree, TreeSliverOpts[foundation.Key]{
		Indentation:          foundation.Some(rendering.TreeSliverIndentationTypeCustom(20)),
		TreeRowExtentBuilder: func(*TreeSliverNode[foundation.Key], rendering.SliverLayoutDimensions) float64 { return 20 },
		TreeNodeBuilder: func(_ BuildContext, node *TreeSliverNode[foundation.Key], _ animation.AnimationStyle) Widget {
			return NewContainer(nil, ContainerOpts{Key: node.Content()})
		},
	})))
	if got, want := ts.TopLeft(gluttertest.ByKey(k1)), (ui.Offset{Dx: 20, Dy: 20}); got != want {
		t.Errorf("the top left of the child node = %v, want %v", got, want)
	}
}

func TestTreeSliverRendersAfterScrolling(t *testing.T) {
	ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 20, Height: 20}})
	controller := NewScrollController()
	row := NewColoredBox(ui.NewColor(0xFFF44336), nil)
	ts.PumpWidget(ltr(NewAlign(NewSizedBoxSquare(foundation.Some(20.0), NewCustomScrollView([]Widget{
		NewTreeSliver([]*TreeSliverNode[any]{NewTreeSliverNode[any](struct{}{})}, TreeSliverOpts[any]{
			TreeRowExtentBuilder: func(*TreeSliverNode[any], rendering.SliverLayoutDimensions) float64 { return 10 },
			TreeNodeBuilder:      func(BuildContext, *TreeSliverNode[any], animation.AnimationStyle) Widget { return row },
		}),
		NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(20.0)})),
	}, CustomScrollViewOpts{Controller: controller})), AlignOpts{Alignment: painting.AlignmentTopLeft})))
	controller.JumpTo(5)
	ts.PumpAndSettle()
	ts.MatchesGolden("sliver_tree_scrolling")
	if got, want := ts.TopLeft(gluttertest.ByType[*ColoredBox]()), (ui.Offset{Dx: 0, Dy: -5}); got != want {
		t.Errorf("the top left of the row = %v, want %v", got, want)
	}
}

func TestTreeSliverAnimatesTheRowsBelowANodeThatToggles(t *testing.T) {
	rowY := func(ts *gluttertest.Tester, text string) float64 { return ts.TopLeft(gluttertest.ByText(text)).Dy }
	newApp := func(t *testing.T) (*gluttertest.Tester, *TreeSliverController[string], []*TreeSliverNode[string]) {
		ts := newTreeTester(t, gluttertest.Opts{Size: ui.Size{Width: 300, Height: 400}})
		controller := NewTreeSliverController[string]()
		nodes := simpleNodeSet()
		ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{Controller: controller})))
		return ts, controller, nodes
	}
	const eps = 0.01
	t.Run("collapsing Root 1 pulls Root 2 up over the 150 ms of the animation", func(t *testing.T) {
		ts, controller, nodes := newApp(t)
		start := rowY(ts, "Root 2")
		controller.ToggleNode(nodes[1]) // two rows of 40 leave
		ts.Pump()                       // the frame that starts the animation
		if got := rowY(ts, "Root 2"); got != start {
			t.Errorf("at the start Root 2 is at %v, want %v", got, start)
		}
		ts.Pump(75 * time.Millisecond)
		if got, want := rowY(ts, "Root 2"), start-40; got < want-eps || got > want+eps {
			t.Errorf("half way Root 2 is at %v, want %v", got, want)
		}
		ts.MatchesGolden("sliver_tree_collapsing")
		ts.PumpAndSettle()
		if got, want := rowY(ts, "Root 2"), start-80; got < want-eps || got > want+eps {
			t.Errorf("at the end Root 2 is at %v, want %v", got, want)
		}
		treeCount(t, ts, "Child 1:0", 0)
	})
	t.Run("expanding Root 2 pushes Root 3 down", func(t *testing.T) {
		ts, controller, nodes := newApp(t)
		start := rowY(ts, "Root 3")
		controller.ToggleNode(nodes[2])
		ts.Pump()
		ts.Pump(75 * time.Millisecond)
		if got, want := rowY(ts, "Root 3"), start+40; got < want-eps || got > want+eps {
			t.Errorf("half way Root 3 is at %v, want %v", got, want)
		}
		ts.PumpAndSettle()
		if got, want := rowY(ts, "Root 3"), start+80; got < want-eps || got > want+eps {
			t.Errorf("at the end Root 3 is at %v, want %v", got, want)
		}
		treeCount(t, ts, "Child 2:1", 1)
	})
	t.Run("a toggle in the middle of an animation turns it around", func(t *testing.T) {
		ts, controller, nodes := newApp(t)
		start := rowY(ts, "Root 3")
		controller.ToggleNode(nodes[2])
		ts.Pump()
		ts.Pump(75 * time.Millisecond)
		controller.ToggleNode(nodes[2]) // collapse again, half way
		ts.PumpAndSettle()
		if got := rowY(ts, "Root 3"); got < start-eps || got > start+eps {
			t.Errorf("Root 3 is at %v after expanding and collapsing, want %v", got, start)
		}
		treeCount(t, ts, "Child 2:0", 0)
	})
	t.Run("two nodes animate together", func(t *testing.T) {
		ts, controller, nodes := newApp(t)
		start := rowY(ts, "Root 3")
		controller.ToggleNode(nodes[1]) // -80
		controller.ToggleNode(nodes[2]) // +80
		ts.Pump()
		ts.Pump(150 * time.Millisecond)
		ts.PumpAndSettle()
		if got := rowY(ts, "Root 3"); got < start-eps || got > start+eps {
			t.Errorf("Root 3 is at %v, want %v", got, start)
		}
		treeCount(t, ts, "Child 1:0", 0)
		treeCount(t, ts, "Child 2:0", 1)
	})
}

func TestTreeSliverWithoutAnimationUpdatesAtOnce(t *testing.T) {
	ts := newTreeTester(t)
	controller := NewTreeSliverController[string]()
	nodes := simpleNodeSet()
	ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{
		Controller: controller, ToggleAnimationStyle: foundation.Some(animation.AnimationStyle{Duration: foundation.Some(time.Duration(0))}),
	})))
	controller.ToggleNode(nodes[1])
	ts.Pump()
	treeCount(t, ts, "Child 1:0", 0) // gone in the first frame
	controller.ToggleNode(nodes[2])
	ts.Pump()
	treeCount(t, ts, "Child 2:0", 1)
}

func TestTreeSliverIndentationAndRowExtents(t *testing.T) {
	ts := newTreeTester(t)
	nodes := simpleNodeSet()
	extents := func(node *TreeSliverNode[string], _ rendering.SliverLayoutDimensions) float64 {
		if len(node.Children()) > 0 {
			return 60
		}
		return 30
	}
	ts.PumpWidget(treeApp(NewTreeSliver(nodes, TreeSliverOpts[string]{
		TreeRowExtentBuilder: extents, Indentation: foundation.Some(rendering.TreeSliverIndentationTypeNone),
		TreeNodeBuilder: func(_ BuildContext, node *TreeSliverNode[string], _ animation.AnimationStyle) Widget {
			return NewContainer(nil, ContainerOpts{Key: foundation.NewValueKey(node.Content())})
		},
	})))
	for key, want := range map[string]ui.Offset{"Root 0": {Dx: 0, Dy: 0}, "Root 1": {Dx: 0, Dy: 30}, "Child 1:0": {Dx: 0, Dy: 90}, "Child 1:1": {Dx: 0, Dy: 120}, "Root 2": {Dx: 0, Dy: 150}} {
		if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(key))); got != want {
			t.Errorf("%s is at %v, want %v", key, got, want)
		}
	}
	if got := ts.Size(gluttertest.ByKey(foundation.NewValueKey("Root 1"))); got != (ui.Size{Width: 800, Height: 60}) {
		t.Errorf("Root 1 is %v, want 800x60", got)
	}
}

func TestTreeSliverOnTheWrongAxisDirection(t *testing.T) {
	ts := newTreeTester(t)
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{NewTreeSliver(simpleNodeSet())}, CustomScrollViewOpts{Reverse: true})))
	err := ts.TakeError()
	for ts.TakeError() != nil { // what follows from a sliver that did not lay out
	}
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("TreeSliverAxisDirection") {
		t.Errorf("error = %v, want TreeSliverAxisDirection", err)
	}
}
