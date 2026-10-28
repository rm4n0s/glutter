package widgets_test

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/focus_manager_test.dart.
// Not ported: the deprecated onKey (RawKeyEvent) parts, the memory allocation
// events, the variants that run on every key transit mode, and the cases that
// need a FocusNode subclass (the node is a plain struct, SPEC §12.10). A
// microtask is a function posted to the binding, which Pump runs.

// focusContext pumps a widget and returns its context, the setupWidget of the
// Dart test.
func focusContext(ts *gluttertest.Tester) BuildContext {
	key := NewGlobalKey[State]()
	ts.PumpWidget(NewContainer(nil, ContainerOpts{Key: key}))
	return key.CurrentContext()
}

// attached makes a node, attaches it to the context and puts it below parent
// (the root scope when parent is nil).
type attachedNode struct {
	node       *FocusNode
	attachment *FocusAttachment
}

func attach(t *testing.T, ctx BuildContext, node *FocusNode) *attachedNode {
	t.Helper()
	t.Cleanup(node.Dispose)
	return &attachedNode{node, node.Attach(ctx)}
}

func nodes(ns ...*FocusNode) []*FocusNode { return ns }

func TestFocusNodeCanAddChildren(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	parent, child1, child2 := NewFocusNode(), NewFocusNode(), NewFocusNode()
	pa, c1a, c2a := attach(t, ctx, parent), attach(t, ctx, child1), attach(t, ctx, child2)
	pa.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	c1a.attachment.Reparent(parent)
	if child1.Parent() != parent || parent.Children()[0] != child1 || parent.Children()[len(parent.Children())-1] != child1 {
		t.Fatalf("children = %v", parent.Children())
	}
	c2a.attachment.Reparent(parent)
	if child1.Parent() != parent || child2.Parent() != parent {
		t.Error("the children are not below the parent")
	}
	if got := parent.Children(); !slices.Equal(got, nodes(child1, child2)) {
		t.Errorf("children = %v, want child1, child2", got)
	}
}

func TestFocusNodeCanRemoveChildren(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	parent, child1, child2 := NewFocusNode(), NewFocusNode(), NewFocusNode()
	pa, c1a, c2a := attach(t, ctx, parent), attach(t, ctx, child1), attach(t, ctx, child2)
	pa.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	c1a.attachment.Reparent(parent)
	c2a.attachment.Reparent(parent)
	if !slices.Equal(parent.Children(), nodes(child1, child2)) {
		t.Fatalf("children = %v", parent.Children())
	}
	c1a.attachment.Detach()
	if child1.Parent() != nil || child2.Parent() != parent || !slices.Equal(parent.Children(), nodes(child2)) {
		t.Errorf("after detaching child1: parent %v, %v, children %v", child1.Parent(), child2.Parent(), parent.Children())
	}
	c2a.attachment.Detach()
	if child1.Parent() != nil || child2.Parent() != nil || len(parent.Children()) != 0 {
		t.Error("the children were not removed")
	}
}

func TestFocusNodeGeometryIsTransformedProperly(t *testing.T) {
	ts := gluttertest.New(t)
	focusNode1 := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node 1"})
	focusNode2 := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node 2"})
	defer focusNode1.Dispose()
	defer focusNode2.Dispose()
	box := func() Widget {
		return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)})
	}
	ts.PumpWidget(NewPadding(painting.NewEdgeInsetsAll(8), NewColumn([]Widget{
		NewFocus(box(), FocusOpts{FocusNode: focusNode1}),
		NewTransformTranslate(ui.Offset{Dx: 10, Dy: 20},
			NewTransformScale(
				NewTransformRotate(math.Pi, NewFocus(box(), FocusOpts{FocusNode: focusNode2})),
				TransformScaleOpts{Scale: foundation.Some(0.33)})),
	})))
	focusNode2.RequestFocus()
	ts.Pump()
	// The matrices keep float32 (SPEC §12.10), so the transformed numbers are
	// exact only to that precision.
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-4 }
	nearRect := func(a, b ui.Rect) bool {
		return near(a.Left, b.Left) && near(a.Top, b.Top) && near(a.Right, b.Right) && near(a.Bottom, b.Bottom)
	}
	if r := focusNode1.Rect(); r != ui.NewRectFromLTRB(300, 8, 500, 108) {
		t.Errorf("rect 1 = %v", r)
	}
	if r := focusNode2.Rect(); !nearRect(r, ui.NewRectFromLTRB(443, 194.5, 377, 161.5)) {
		t.Errorf("rect 2 = %v", r)
	}
	if got := focusNode1.Size(); got != (ui.Size{Width: 200, Height: 100}) {
		t.Errorf("size 1 = %v", got)
	}
	if got := focusNode2.Size(); !near(got.Width, -66) || !near(got.Height, -33) {
		t.Errorf("size 2 = %v", got)
	}
	if got := focusNode1.Offset(); got != (ui.Offset{Dx: 300, Dy: 8}) {
		t.Errorf("offset 1 = %v", got)
	}
	if got := focusNode2.Offset(); !near(got.Dx, 443) || !near(got.Dy, 194.5) {
		t.Errorf("offset 2 = %v", got)
	}
}

// focusTree is the tree most scope tests use:
//
//	root -> scope -> parent1 -> child1
//	               \-> parent2 -> child2
type focusTree struct {
	scope                            *FocusScopeNode
	parent1, parent2, child1, child2 *FocusNode
	attachments                      map[*FocusNode]*FocusAttachment
}

func newFocusTree(t *testing.T, ts *gluttertest.Tester, ctx BuildContext) *focusTree {
	t.Helper()
	ft := &focusTree{
		scope:   NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}}),
		parent1: NewFocusNode(FocusNodeOpts{DebugLabel: "Parent 1"}), parent2: NewFocusNode(FocusNodeOpts{DebugLabel: "Parent 2"}),
		child1: NewFocusNode(FocusNodeOpts{DebugLabel: "Child 1"}), child2: NewFocusNode(FocusNodeOpts{DebugLabel: "Child 2"}),
		attachments: map[*FocusNode]*FocusAttachment{},
	}
	for _, n := range []*FocusNode{ft.scope.Node(), ft.parent1, ft.parent2, ft.child1, ft.child2} {
		ft.attachments[n] = attach(t, ctx, n).attachment
	}
	ft.attachments[ft.scope.Node()].Reparent(ts.Binding.FocusManager().RootScope().Node())
	ft.attachments[ft.parent1].Reparent(ft.scope.Node())
	ft.attachments[ft.parent2].Reparent(ft.scope.Node())
	ft.attachments[ft.child1].Reparent(ft.parent1)
	ft.attachments[ft.child2].Reparent(ft.parent2)
	return ft
}

func TestFocusNodeDescendantsAreFocusableDisablesFocusForDescendants(t *testing.T) {
	ts := gluttertest.New(t)
	ft := newFocusTree(t, ts, focusContext(ts))
	fm := ts.Binding.FocusManager()
	ft.child1.RequestFocus()
	ts.Pump()
	if fm.PrimaryFocus() != ft.child1 || ft.scope.FocusedChild() != ft.child1 {
		t.Fatalf("primary focus = %v, focused child = %v", fm.PrimaryFocus(), ft.scope.FocusedChild())
	}
	if !slices.Contains(ft.scope.TraversalDescendants(), ft.child1) || !slices.Contains(ft.scope.TraversalDescendants(), ft.child2) {
		t.Fatal("both children should be traversal descendants")
	}

	ft.parent2.SetDescendantsAreFocusable(false)
	// The node itself should still be focusable, even if the descendants are not.
	ft.parent2.RequestFocus()
	ts.Pump()
	if !ft.parent2.HasPrimaryFocus() {
		t.Fatal("parent2 should have the primary focus")
	}
	ft.child2.RequestFocus()
	ts.Pump()
	if fm.PrimaryFocus() == ft.child2 || fm.PrimaryFocus() != ft.parent2 || ft.scope.FocusedChild() != ft.parent2 {
		t.Errorf("primary focus = %v, want parent2", fm.PrimaryFocus())
	}
	if !slices.Contains(ft.scope.TraversalDescendants(), ft.child1) || slices.Contains(ft.scope.TraversalDescendants(), ft.child2) {
		t.Error("only child1 should be a traversal descendant")
	}

	ft.parent1.SetDescendantsAreFocusable(false)
	ts.Pump()
	if fm.PrimaryFocus() == ft.child2 || fm.PrimaryFocus() == ft.child1 || ft.scope.FocusedChild() != ft.parent2 {
		t.Errorf("primary focus = %v, focused child = %v", fm.PrimaryFocus(), ft.scope.FocusedChild())
	}
	if slices.Contains(ft.scope.TraversalDescendants(), ft.child1) || slices.Contains(ft.scope.TraversalDescendants(), ft.child2) {
		t.Error("no child should be a traversal descendant")
	}
}

func TestFocusNodeDescendantsAreTraversableDisablesTraversalForDescendants(t *testing.T) {
	ts := gluttertest.New(t)
	ft := newFocusTree(t, ts, focusContext(ts))
	want := func(ns ...*FocusNode) {
		t.Helper()
		if got := ft.scope.TraversalDescendants(); !slices.Equal(got, ns) {
			t.Errorf("traversal descendants = %v, want %v", got, ns)
		}
	}
	want(ft.child1, ft.parent1, ft.child2, ft.parent2)
	ft.parent2.SetDescendantsAreTraversable(false)
	want(ft.child1, ft.parent1, ft.parent2)
	ft.parent1.SetDescendantsAreTraversable(false)
	want(ft.parent1, ft.parent2)
	ft.parent1.SetDescendantsAreTraversable(true)
	ft.parent2.SetDescendantsAreTraversable(true)
	ft.scope.SetDescendantsAreTraversable(false)
	want()
}

func TestFocusNodeCanRequestFocusDoesNotAffectTraversalChildren(t *testing.T) {
	ts := gluttertest.New(t)
	ft := newFocusTree(t, ts, focusContext(ts))
	ft.child1.RequestFocus()
	ts.Pump()
	if ts.Binding.FocusManager().PrimaryFocus() != ft.child1 || ft.scope.FocusedChild() != ft.child1 {
		t.Fatal("child1 should have the focus")
	}
	if !slices.Contains(ft.parent2.TraversalChildren(), ft.child2) || !slices.Contains(ft.scope.TraversalChildren(), ft.parent2) {
		t.Fatal("child2 and parent2 should be traversal children")
	}
	ft.parent2.SetCanRequestFocus(false)
	ts.Pump()
	if !slices.Contains(ft.parent2.TraversalChildren(), ft.child2) {
		t.Error("child2 should still be a traversal child of parent2")
	}
	if slices.Contains(ft.scope.TraversalChildren(), ft.parent2) {
		t.Error("parent2 should not be a traversal child of the scope")
	}
}

func TestFocusNodeImplementsDebugFillProperties(t *testing.T) {
	skipUnlessDebug(t)
	builder := foundation.NewDiagnosticPropertiesBuilder()
	focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Label"})
	defer focusNode.Dispose()
	focusNode.DebugFillProperties(builder)
	var description []string
	for _, n := range builder.Properties {
		description = append(description, n.String())
	}
	want := []string{
		"descendantsAreFocusable: true",
		"descendantsAreTraversable: true",
		"canRequestFocus: true",
		"hasFocus: false",
		"hasPrimaryFocus: false",
	}
	// Flutter lists "context: null" first; a property without a value is left out.
	if !slices.Equal(description, want) {
		t.Errorf("description = %q, want %q", description, want)
	}
}

// setLifecycle changes the lifecycle state and runs the microtasks that follow,
// as the await of the Dart helper does.
func setLifecycle(ts *gluttertest.Tester, state ui.AppLifecycleState) {
	ts.Dispatcher().SetAppLifecycleState(state)
	ts.FlushPosted()
}

// digit1 is LogicalKeyboardKey.digit1.
const digit1 = services.LogicalKeyboardKeyDigit1

func TestFocusNodeOnKeyEventHandlersAreCalledFromThePrimaryFocusUp(t *testing.T) {
	ts := gluttertest.New(t)
	focusNode1, focusNode2, focusNode3 := NewFocusNode(), NewFocusNode(), NewFocusNode()
	defer focusNode1.Dispose()
	defer focusNode2.Dispose()
	defer focusNode3.Dispose()
	results := []KeyEventResult{KeyEventResultIgnored, KeyEventResultIgnored, KeyEventResultIgnored}
	var logs []int
	handler := func(id int) FocusOnKeyEventCallback {
		return func(*FocusNode, services.KeyEvent) KeyEventResult {
			logs = append(logs, id)
			return results[id/10]
		}
	}
	ts.PumpWidget(NewFocus(
		NewFocus(
			NewFocus(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}),
				FocusOpts{FocusNode: focusNode3, OnKeyEvent: handler(20)}),
			FocusOpts{FocusNode: focusNode2, OnKeyEvent: handler(10)}),
		FocusOpts{FocusNode: focusNode1, OnKeyEvent: handler(0)}))
	focusNode3.RequestFocus()
	ts.Pump()

	steps := []struct {
		name    string
		results []KeyEventResult
		up      bool
		handled bool
		logs    []int
	}{
		{"all ignored", []KeyEventResult{KeyEventResultIgnored, KeyEventResultIgnored, KeyEventResultIgnored}, false, false, []int{20, 10, 0}},
		{"a handler can stop the propagation", []KeyEventResult{KeyEventResultIgnored, KeyEventResultHandled, KeyEventResultIgnored}, true, true, []int{20, 10}},
		{"skipRemainingHandlers works", []KeyEventResult{KeyEventResultIgnored, KeyEventResultSkipRemainingHandlers, KeyEventResultIgnored}, true, false, []int{20, 10}},
	}
	for _, s := range steps {
		results = s.results
		logs = nil
		var handled bool
		if s.up {
			handled = ts.SendKeyUpEvent(digit1)
		} else {
			handled = ts.SendKeyDownEvent(digit1)
		}
		if handled != s.handled || !slices.Equal(logs, s.logs) {
			t.Errorf("%s: handled = %v, logs = %v; want %v, %v", s.name, handled, logs, s.handled, s.logs)
		}
		if !s.up {
			ts.SendKeyUpEvent(digit1)
		}
	}
}

func TestFocusManagerIgnoresAppLifecycleChangesOnAndroidAndIOS(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ctx := focusContext(ts)
			scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}})
			focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Focus Node"})
			sa, fa := attach(t, ctx, scope.Node()), attach(t, ctx, focusNode)
			sa.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
			fa.attachment.Reparent(scope.Node())
			focusNode.RequestFocus()
			ts.Pump()
			if !focusNode.HasPrimaryFocus() {
				t.Fatal("focusNode should have the primary focus")
			}
			setLifecycle(ts, ui.AppLifecycleStatePaused)
			if !focusNode.HasPrimaryFocus() {
				t.Error("the focus should stay when paused")
			}
			setLifecycle(ts, ui.AppLifecycleStateResumed)
			if !focusNode.HasPrimaryFocus() {
				t.Error("the focus should stay when resumed")
			}
		})
	}
}

func TestFocusManagerRespondsToAppLifecycleChanges(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ctx := focusContext(ts)
			scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}})
			focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Focus Node"})
			sa, fa := attach(t, ctx, scope.Node()), attach(t, ctx, focusNode)
			sa.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
			fa.attachment.Reparent(scope.Node())
			focusNode.RequestFocus()
			ts.Pump()
			if !focusNode.HasPrimaryFocus() {
				t.Fatal("focusNode should have the primary focus")
			}
			setLifecycle(ts, ui.AppLifecycleStatePaused)
			if focusNode.HasPrimaryFocus() {
				t.Error("the focus should be suspended when paused")
			}
			setLifecycle(ts, ui.AppLifecycleStateResumed)
			if !focusNode.HasPrimaryFocus() {
				t.Error("the focus should be restored when resumed")
			}
		})
	}
}

func TestFocusManagerNodeIsRemovedCompletelyEvenIfAppIsPaused(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}})
	focusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Focus Node"})
	sa, fa := attach(t, ctx, scope.Node()), attach(t, ctx, focusNode)
	sa.attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	fa.attachment.Reparent(scope.Node())
	focusNode.RequestFocus()
	ts.Pump()
	if !focusNode.HasPrimaryFocus() {
		t.Fatal("focusNode should have the primary focus")
	}
	setLifecycle(ts, ui.AppLifecycleStatePaused)
	fa.attachment.Detach()
	if focusNode.HasPrimaryFocus() {
		t.Error("the detached node should not have the focus")
	}
	setLifecycle(ts, ui.AppLifecycleStateResumed)
	if focusNode.HasPrimaryFocus() {
		t.Error("the detached node should not get the focus back")
	}
}

func TestFocusManagerSuspendedFocusIsNotRestoredWhenAnotherNodeClaimedFocus(t *testing.T) {
	ts := gluttertest.New(t)
	ctx := focusContext(ts)
	scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}})
	nodeA, nodeB := NewFocusNode(FocusNodeOpts{DebugLabel: "A"}), NewFocusNode(FocusNodeOpts{DebugLabel: "B"})
	attach(t, ctx, scope.Node()).attachment.Reparent(ts.Binding.FocusManager().RootScope().Node())
	attach(t, ctx, nodeA).attachment.Reparent(scope.Node())
	attach(t, ctx, nodeB).attachment.Reparent(scope.Node())
	nodeA.RequestFocus()
	ts.Pump()
	if !nodeA.HasPrimaryFocus() {
		t.Fatal("nodeA should have the primary focus")
	}
	// The app goes inactive: nodeA's focus is suspended and parked.
	setLifecycle(ts, ui.AppLifecycleStateInactive)
	if nodeA.HasPrimaryFocus() {
		t.Fatal("nodeA should have lost the focus")
	}
	// Another node requests the focus before the app resumes (a different window
	// was activated).
	nodeB.RequestFocus()
	// On resume, nodeA must not steal the focus back from nodeB.
	setLifecycle(ts, ui.AppLifecycleStateResumed)
	ts.Pump()
	if !nodeB.HasPrimaryFocus() || nodeA.HasPrimaryFocus() {
		t.Error("nodeB should have the focus, not nodeA")
	}
}

var _ = strings.Contains

// focusEnv builds focus trees by hand: Put attaches a node to the context and
// reparents it, as the Dart tests do with attach(context).reparent(parent:).
type focusEnv struct {
	t   *testing.T
	ts  *gluttertest.Tester
	ctx BuildContext
	att map[*FocusNode]*FocusAttachment
}

func newFocusEnv(t *testing.T) *focusEnv {
	t.Helper()
	ts := gluttertest.New(t)
	return &focusEnv{t: t, ts: ts, ctx: focusContext(ts), att: map[*FocusNode]*FocusAttachment{}}
}

func (e *focusEnv) fm() *FocusManager { return e.ts.Binding.FocusManager() }
func (e *focusEnv) root() *FocusNode  { return e.fm().RootScope().Node() }

func (e *focusEnv) node(label string) *FocusNode {
	n := NewFocusNode(FocusNodeOpts{DebugLabel: label})
	e.attach(n)
	return n
}

func (e *focusEnv) scope(label string) *FocusScopeNode {
	s := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: label}})
	e.attach(s.Node())
	return s
}

func (e *focusEnv) attach(n *FocusNode) *FocusAttachment {
	e.t.Cleanup(n.Dispose)
	a := n.Attach(e.ctx)
	e.att[n] = a
	return a
}

// put reparents the node below the parent (the root scope when it is nil).
func (e *focusEnv) put(n, parent *FocusNode) {
	e.t.Helper()
	if parent == nil {
		parent = e.root()
	}
	a, ok := e.att[n]
	if !ok {
		a = e.attach(n)
	}
	a.Reparent(parent)
}

func (e *focusEnv) pump() { e.ts.Pump() }

func TestFocusScopeNodeCanSetFirstFocusOnAScopeWithNoManager(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent, child1, child2 := e.scope("Scope"), e.scope("Parent"), e.scope("Child 1"), e.scope("Child 2")
	scope.SetFirstFocus(parent)
	parent.SetFirstFocus(child1)
	parent.SetFirstFocus(child2)
	child1.RequestFocus()
	e.pump()
	if scope.HasFocus() || child1.HasFocus() || child1.HasPrimaryFocus() {
		t.Error("nothing should have the focus: the scopes are not in the tree")
	}
	if scope.FocusedChild() != parent.Node() || parent.FocusedChild() != child1.Node() {
		t.Errorf("focused children = %v, %v", scope.FocusedChild(), parent.FocusedChild())
	}
	e.att[child1.Node()].Detach()
	if scope.HasFocus() || scope.FocusedChild() != parent.Node() {
		t.Error("detaching child1 should not change the scope")
	}
}

func TestFocusScopeNodeRemovingANodeRemovesItFromScope(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent, child1, child2 := e.scope(""), e.node(""), e.node(""), e.node("")
	e.put(scope.Node(), nil)
	e.put(parent, scope.Node())
	e.put(child1, parent)
	e.put(child2, parent)
	child1.RequestFocus()
	e.pump()
	if !scope.HasFocus() || !child1.HasFocus() || !child1.HasPrimaryFocus() || scope.FocusedChild() != child1 {
		t.Fatal("child1 should have the focus")
	}
	e.att[child1].Detach()
	if scope.HasFocus() || scope.FocusedChild() != nil {
		t.Errorf("after the detach: has focus %v, focused child %v", scope.HasFocus(), scope.FocusedChild())
	}
}

func TestFocusScopeNodeCanAddChildrenToScopeAndFocus(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent, child1, child2 := e.scope(""), e.node(""), e.node(""), e.node("")
	e.put(scope.Node(), nil)
	e.put(parent, scope.Node())
	e.put(child1, parent)
	e.put(child2, parent)
	if scope.Children()[0] != parent || parent.Parent() != scope.Node() || child1.Parent() != parent || child2.Parent() != parent ||
		!slices.Equal(parent.Children(), nodes(child1, child2)) {
		t.Fatal("the tree is wrong")
	}
	child1.RequestFocus()
	e.pump()
	if scope.FocusedChild() != child1 || !parent.HasFocus() || parent.HasPrimaryFocus() || !child1.HasFocus() ||
		!child1.HasPrimaryFocus() || child2.HasFocus() || child2.HasPrimaryFocus() {
		t.Error("child1 should have the focus")
	}
	child2.RequestFocus()
	e.pump()
	if scope.FocusedChild() != child2 || !parent.HasFocus() || parent.HasPrimaryFocus() || child1.HasFocus() ||
		child1.HasPrimaryFocus() || !child2.HasFocus() || !child2.HasPrimaryFocus() {
		t.Error("child2 should have the focus")
	}
}

// Regression test for https://github.com/flutter/flutter/issues/136758.
func TestFocusScopeNodeRemovingGrandchildrenFromScopeUpdatesFocusedChild(t *testing.T) {
	e := newFocusEnv(t)
	scope1, child1, child2 := e.scope("scope1"), e.node("child1"), e.node("child2")
	e.put(scope1.Node(), nil)
	e.put(child1, scope1.Node())
	e.put(child2, child1)
	if child1.Parent() != scope1.Node() || scope1.Children()[0] != child1 {
		t.Fatal("the tree is wrong")
	}
	child2.RequestFocus()
	e.pump()
	if scope1.FocusedChild() != child2 {
		t.Fatal("child2 should be the focused child")
	}
	// Detaches the middle child: the scope no longer references child2.
	e.att[child1].Detach()
	if scope1.FocusedChild() != nil {
		t.Errorf("focused child = %v, want none", scope1.FocusedChild())
	}
}

func TestFocusScopeNodeRequestingFocusBeforeAddingToTreeResultsInARequestAfterAdding(t *testing.T) {
	e := newFocusEnv(t)
	scope := e.scope("")
	child := NewFocusNode()
	defer child.Dispose()
	child.RequestFocus()
	if child.HasPrimaryFocus() {
		t.Fatal("the child is not attached yet")
	}
	e.put(scope.Node(), nil)
	e.pump()
	if scope.FocusedChild() != nil || child.HasPrimaryFocus() {
		t.Fatal("the child is still not attached")
	}
	childAttachment := child.Attach(e.ctx)
	if child.HasPrimaryFocus() {
		t.Fatal("the child is not parented yet")
	}
	childAttachment.Reparent(scope.Node())
	e.pump()
	if !child.HasPrimaryFocus() {
		t.Error("now attached and parented, so the focus should have happened")
	}
}

func TestFocusScopeNodeAutofocusWorks(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent, child1, child2 := e.scope("Scope"), e.node("Parent"), e.node("Child 1"), e.node("Child 2")
	e.put(scope.Node(), nil)
	e.put(parent, scope.Node())
	e.put(child1, parent)
	e.put(child2, parent)

	scope.Autofocus(child2)
	e.pump()
	if scope.FocusedChild() != child2 || !parent.HasFocus() || child1.HasFocus() || child1.HasPrimaryFocus() ||
		!child2.HasFocus() || !child2.HasPrimaryFocus() {
		t.Fatal("child2 should have the focus")
	}
	child1.RequestFocus()
	scope.Autofocus(child2)
	e.pump()
	if scope.FocusedChild() != child1 || !parent.HasFocus() || !child1.HasFocus() || !child1.HasPrimaryFocus() ||
		child2.HasFocus() || child2.HasPrimaryFocus() {
		t.Error("child1 should have the focus: the scope already had a focused child")
	}
}

func TestFocusScopeNodeAddingAFocusedChildToAScopeSetsScopeAsFocusedChildInParentScope(t *testing.T) {
	e := newFocusEnv(t)
	scope1, scope2, child1, child2 := e.scope(""), e.scope(""), e.node(""), e.node("")
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), scope1.Node())
	e.put(child1, scope1.Node())
	e.put(child2, scope2.Node())
	child2.RequestFocus()
	e.pump()
	if scope2.FocusedChild() != child2 || scope1.FocusedChild() != scope2.Node() || child1.HasFocus() || child1.HasPrimaryFocus() ||
		!child2.HasFocus() || !child2.HasPrimaryFocus() {
		t.Fatal("child2 should have the focus")
	}
	child1.RequestFocus()
	e.pump()
	if scope2.FocusedChild() != child2 || scope1.FocusedChild() != child1 || !child1.HasFocus() || !child1.HasPrimaryFocus() ||
		child2.HasFocus() || child2.HasPrimaryFocus() {
		t.Error("child1 should have the focus now")
	}
}

func TestFocusScopeNodeCanMoveNodeWithFocusWithoutLosingFocus(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent1, parent2, child1, child2 := e.scope("Scope"), e.node("Parent 1"), e.node("Parent 2"), e.node("Child 1"), e.node("Child 2")
	e.put(scope.Node(), nil)
	e.put(parent1, scope.Node())
	e.put(parent2, scope.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	if !slices.Equal(scope.Children(), nodes(parent1, parent2)) || parent1.Parent() != scope.Node() || parent2.Parent() != scope.Node() ||
		child1.Parent() != parent1 || child2.Parent() != parent1 || !slices.Equal(parent1.Children(), nodes(child1, child2)) {
		t.Fatal("the tree is wrong")
	}
	child1.RequestFocus()
	e.pump()
	e.att[child1].Reparent(parent2)
	e.pump()
	if scope.FocusedChild() != child1 || child1.Parent() != parent2 || child2.Parent() != parent1 ||
		parent1.Children()[0] != child2 || parent2.Children()[0] != child1 {
		t.Error("child1 should have moved with the focus")
	}
}

func TestFocusScopeNodeCanRequestFocusAffectsDescendantsAreFocusable(t *testing.T) {
	scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope"}})
	steps := []struct {
		name                    string
		do                      func()
		descendantsAreFocusable bool
		canRequestFocus         bool
	}{
		{"descendants not focusable", func() { scope.SetDescendantsAreFocusable(false) }, false, true},
		{"descendants focusable", func() { scope.SetDescendantsAreFocusable(true) }, true, true},
		{"cannot request focus", func() { scope.SetCanRequestFocus(false) }, false, false},
		{"can request focus", func() { scope.SetCanRequestFocus(true) }, true, true},
	}
	for _, s := range steps {
		s.do()
		if scope.DescendantsAreFocusable() != s.descendantsAreFocusable || scope.CanRequestFocus() != s.canRequestFocus {
			t.Errorf("%s: descendantsAreFocusable %v, canRequestFocus %v", s.name, scope.DescendantsAreFocusable(), scope.CanRequestFocus())
		}
	}
}

func TestFocusScopeNodeCanRequestFocusAffectsChildren(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent1, parent2, child1, child2 := e.scope("Scope"), e.node("Parent 1"), e.node("Parent 2"), e.node("Child 1"), e.node("Child 2")
	e.put(scope.Node(), nil)
	e.put(parent1, scope.Node())
	e.put(parent2, scope.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	child1.RequestFocus()
	e.pump()
	has := func(list []*FocusNode, n *FocusNode) bool { return slices.Contains(list, n) }
	if e.fm().PrimaryFocus() != child1 || scope.FocusedChild() != child1 || !has(scope.TraversalDescendants(), child1) ||
		!has(scope.TraversalDescendants(), child2) || !has(scope.TraversalChildren(), parent1) || !has(parent1.TraversalChildren(), child2) {
		t.Fatal("child1 should have the focus and all be traversable")
	}
	scope.SetCanRequestFocus(false)
	e.pump()
	child2.RequestFocus()
	e.pump()
	if e.fm().PrimaryFocus() == child2 || e.fm().PrimaryFocus() == child1 || scope.FocusedChild() != child1 ||
		has(scope.TraversalDescendants(), child1) || has(scope.TraversalDescendants(), child2) ||
		has(scope.TraversalChildren(), parent1) || has(parent1.TraversalChildren(), child2) {
		t.Errorf("a scope that cannot be focused: primary focus %v, focused child %v", e.fm().PrimaryFocus(), scope.FocusedChild())
	}
}

func TestFocusScopeNodeSkipTraversalDoesNotAffectChildren(t *testing.T) {
	e := newFocusEnv(t)
	scope, parent1, parent2, child1, child2 := e.scope("Scope"), e.node("Parent 1"), e.node("Parent 2"), e.node("Child 1"), e.node("Child 2")
	e.put(scope.Node(), nil)
	e.put(parent1, scope.Node())
	e.put(parent2, scope.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	child1.RequestFocus()
	e.pump()
	root := e.fm().RootScope()
	if e.fm().PrimaryFocus() != child1 || scope.FocusedChild() != child1 || !slices.Contains(root.TraversalDescendants(), scope.Node()) ||
		!slices.Contains(scope.TraversalDescendants(), child1) || !slices.Contains(scope.TraversalDescendants(), child2) {
		t.Fatal("child1 should have the focus")
	}
	scope.SetSkipTraversal(true)
	e.pump()
	if e.fm().PrimaryFocus() != child1 || scope.FocusedChild() != child1 || slices.Contains(root.TraversalDescendants(), scope.Node()) ||
		!slices.Contains(scope.TraversalDescendants(), child1) || !slices.Contains(scope.TraversalDescendants(), child2) {
		t.Error("skipping the scope should not affect its children")
	}
}

func TestFocusScopeNodeCanMoveNodeBetweenScopesAndLoseScopeFocus(t *testing.T) {
	e := newFocusEnv(t)
	scope1, scope2 := e.scope("scope1"), e.scope("scope2")
	parent1, parent2 := e.node("parent1"), e.node("parent2")
	child1, child2, child3, child4 := e.node("child1"), e.node("child2"), e.node("child3"), e.node("child4")
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), nil)
	e.put(parent1, scope1.Node())
	e.put(parent2, scope2.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	e.put(child3, parent2)
	e.put(child4, parent2)
	child1.RequestFocus()
	e.pump()
	if scope1.FocusedChild() != child1 || slices.Contains(parent2.Children(), child1) {
		t.Fatal("child1 should be the focused child of scope1")
	}
	e.att[child1].Reparent(parent2)
	e.pump()
	if scope1.FocusedChild() != nil || !slices.Contains(parent2.Children(), child1) {
		t.Errorf("focused child of scope1 = %v, want none", scope1.FocusedChild())
	}
}

func TestFocusScopeNodeAncestorsAndDescendantsAreComputedAndRecomputedProperly(t *testing.T) {
	e := newFocusEnv(t)
	scope1, scope2 := e.scope("scope1"), e.scope("scope2")
	parent1, parent2 := e.node("parent1"), e.node("parent2")
	child1, child2, child3, child4 := e.node("child1"), e.node("child2"), e.node("child3"), e.node("child4")
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), nil)
	e.put(parent1, scope1.Node())
	e.put(parent2, scope2.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	e.put(child3, parent2)
	e.put(child4, parent2)
	child4.RequestFocus()
	e.pump()
	rootScope := e.fm().RootScope()
	// The View's focus nodes come first: the FocusTraversalGroup above the view
	// and the view scope.
	preamble := nodes(rootScope.Children()[0].Children()[0], rootScope.Children()[0])
	if got := child4.Ancestors(); !slices.Equal(got, nodes(parent2, scope2.Node(), rootScope.Node())) {
		t.Errorf("ancestors = %v", got)
	}
	want := append(slices.Clone(preamble), child1, child2, parent1, scope1.Node(), child3, child4, parent2, scope2.Node())
	if got := rootScope.Descendants(); !slices.Equal(got, want) {
		t.Errorf("descendants = %v, want %v", got, want)
	}
	e.att[scope2.Node()].Reparent(child2)
	e.pump()
	if got := child4.Ancestors(); !slices.Equal(got, nodes(parent2, scope2.Node(), child2, parent1, scope1.Node(), rootScope.Node())) {
		t.Errorf("ancestors after the move = %v", got)
	}
	want = append(slices.Clone(preamble), child1, child3, child4, parent2, scope2.Node(), child2, parent1, scope1.Node())
	if got := rootScope.Descendants(); !slices.Equal(got, want) {
		t.Errorf("descendants after the move = %v, want %v", got, want)
	}
}

func TestFocusScopeNodeCanMoveFocusBetweenScopesAndKeepFocus(t *testing.T) {
	e := newFocusEnv(t)
	scope1, scope2 := e.scope(""), e.scope("")
	parent1, parent2 := e.node(""), e.node("")
	child1, child2, child3, child4 := e.node(""), e.node(""), e.node(""), e.node("")
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), nil)
	e.put(parent1, scope1.Node())
	e.put(parent2, scope2.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	e.put(child3, parent2)
	e.put(child4, parent2)
	child4.RequestFocus()
	e.pump()
	child1.RequestFocus()
	e.pump()
	check := func(step string, got, want []bool) {
		t.Helper()
		if !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", step, got, want)
		}
	}
	state := func() []bool {
		return []bool{child4.HasFocus(), child4.HasPrimaryFocus(), child1.HasFocus(), child1.HasPrimaryFocus(), scope1.HasFocus(),
			scope1.HasPrimaryFocus(), scope2.HasFocus(), scope2.HasPrimaryFocus(), parent1.HasFocus(), parent2.HasFocus()}
	}
	check("child1 focused", state(), []bool{false, false, true, true, true, false, false, false, true, false})
	if scope1.FocusedChild() != child1 || scope2.FocusedChild() != child4 {
		t.Fatalf("focused children = %v, %v", scope1.FocusedChild(), scope2.FocusedChild())
	}
	scope2.RequestFocus()
	e.pump()
	check("scope2 focused", state(), []bool{true, true, false, false, false, false, true, false, false, true})
	if scope1.FocusedChild() != child1 || scope2.FocusedChild() != child4 {
		t.Errorf("focused children = %v, %v", scope1.FocusedChild(), scope2.FocusedChild())
	}
}

// twoScopes is the tree of the unfocus tests:
//
//	root -> scope1 -> parent1 -> child1, child2
//	     \-> scope2 -> parent2 -> child3, child4
type twoScopes struct {
	*focusEnv
	scope1, scope2                 *FocusScopeNode
	parent1, parent2               *FocusNode
	child1, child2, child3, child4 *FocusNode
}

func newTwoScopes(t *testing.T) *twoScopes {
	t.Helper()
	e := newFocusEnv(t)
	s := &twoScopes{focusEnv: e,
		scope1: e.scope("scope1"), scope2: e.scope("scope2"), parent1: e.node("parent1"), parent2: e.node("parent2"),
		child1: e.node("child1"), child2: e.node("child2"), child3: e.node("child3"), child4: e.node("child4")}
	e.put(s.scope1.Node(), nil)
	e.put(s.scope2.Node(), nil)
	e.put(s.parent1, s.scope1.Node())
	e.put(s.parent2, s.scope2.Node())
	e.put(s.child1, s.parent1)
	e.put(s.child2, s.parent1)
	e.put(s.child3, s.parent2)
	e.put(s.child4, s.parent2)
	return s
}

// buildHistory focuses child4, child2, child3 and child1 in turn.
func (s *twoScopes) buildHistory() {
	for _, n := range nodes(s.child4, s.child2, s.child3, s.child1) {
		n.RequestFocus()
		s.pump()
	}
}

func TestFocusScopeNodeUnfocusWithDispositionPreviouslyFocusedChildWorksProperly(t *testing.T) {
	s := newTwoScopes(t)
	s.buildHistory()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 {
		t.Fatal("the history is wrong")
	}
	s.child1.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
	s.pump()
	if s.scope1.FocusedChild() != s.child2 || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasFocus() || s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || !s.child2.HasPrimaryFocus() {
		t.Error("the focus should be on child2")
	}
	// The child can be focused again.
	s.child1.RequestFocus()
	s.pump()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasFocus() || s.scope2.HasFocus() ||
		!s.child1.HasPrimaryFocus() || s.child3.HasPrimaryFocus() {
		t.Error("the focus should be on child1")
	}
	// The same thing happens when unfocusing a second time.
	s.child1.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
	s.pump()
	if s.scope1.FocusedChild() != s.child2 || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasFocus() || s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || !s.child2.HasPrimaryFocus() {
		t.Error("the focus should be on child2 again")
	}
	// When the scope gets unfocused, the sibling scope gets the focus.
	s.child1.RequestFocus()
	s.pump()
	s.scope1.Unfocus(UnfocusDispositionPreviouslyFocusedChild)
	s.pump()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 || s.scope1.HasFocus() || !s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || !s.child3.HasPrimaryFocus() {
		t.Error("the focus should be on child3")
	}
}

func TestFocusScopeNodeUnfocusWithDispositionScopeWorksProperly(t *testing.T) {
	s := newTwoScopes(t)
	s.buildHistory()
	s.child1.Unfocus()
	s.pump()
	// The focused child of scope2 does not change; the focus does.
	if s.scope1.FocusedChild() != nil || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasPrimaryFocus() || s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || s.child2.HasPrimaryFocus() {
		t.Error("the focus should be on scope1")
	}
	s.child1.RequestFocus()
	s.pump()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasFocus() || s.scope2.HasFocus() ||
		!s.child1.HasPrimaryFocus() || s.child3.HasPrimaryFocus() {
		t.Error("the focus should be on child1")
	}
	s.child1.Unfocus()
	s.pump()
	if s.scope1.FocusedChild() != nil || s.scope2.FocusedChild() != s.child3 || !s.scope1.HasPrimaryFocus() || s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || s.child2.HasPrimaryFocus() {
		t.Error("the focus should be on scope1 again")
	}
	// When the scope gets unfocused, its parent scope (the root scope) gets the
	// focus, without changing the focused children.
	s.child1.RequestFocus()
	s.pump()
	s.scope1.Unfocus()
	s.pump()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 || s.scope1.HasFocus() || s.scope2.HasFocus() ||
		s.child1.HasPrimaryFocus() || s.child3.HasPrimaryFocus() || !s.fm().RootScope().HasPrimaryFocus() {
		t.Error("the focus should be on the root scope")
	}
}

func TestFocusScopeNodeUnfocusWorksProperlyWhenSomeNodesAreUnfocusable(t *testing.T) {
	s := newTwoScopes(t)
	s.buildHistory()
	if !s.child1.HasPrimaryFocus() {
		t.Fatal("child1 should have the focus")
	}
	s.scope1.SetCanRequestFocus(false)
	s.pump()
	if s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 || !s.child3.HasPrimaryFocus() {
		t.Fatal("child3 should have the focus")
	}
	for _, d := range []UnfocusDisposition{UnfocusDispositionScope, UnfocusDispositionPreviouslyFocusedChild} {
		s.child1.Unfocus(d)
		s.pump()
		if !s.child3.HasPrimaryFocus() || s.scope1.FocusedChild() != s.child1 || s.scope2.FocusedChild() != s.child3 ||
			s.scope1.HasPrimaryFocus() || !s.scope2.HasFocus() || s.child1.HasPrimaryFocus() || s.child2.HasPrimaryFocus() {
			t.Errorf("disposition %v: the focus should stay on child3", d)
		}
	}
}

func TestFocusScopeNodeRequestingFocusOnAScopeWorksWhenSomeFocusedChildNodesAreUnfocusable(t *testing.T) {
	s := newTwoScopes(t)
	s.buildHistory()
	if !s.child1.HasPrimaryFocus() {
		t.Fatal("child1 should have the focus")
	}
	s.child1.SetCanRequestFocus(false)
	s.child3.SetCanRequestFocus(false)
	s.pump()
	s.scope1.RequestFocus()
	s.pump()
	if s.scope1.FocusedChild() != s.child2 || !s.child2.HasPrimaryFocus() {
		t.Fatalf("child2 should have the focus, scope1's focused child is %v", s.scope1.FocusedChild())
	}
	s.scope2.RequestFocus()
	s.pump()
	if s.scope2.FocusedChild() != s.child4 || !s.child4.HasPrimaryFocus() {
		t.Errorf("child4 should have the focus, scope2's focused child is %v", s.scope2.FocusedChild())
	}
}

func TestFocusScopeNodeKeyHandlingBubblesUpAndTerminatesWhenHandled(t *testing.T) {
	e := newFocusEnv(t)
	receivedAnEvent := map[*FocusNode]bool{}
	shouldHandle := map[*FocusNode]bool{}
	handle := func(node *FocusNode, _ services.KeyEvent) KeyEventResult {
		if shouldHandle[node] {
			receivedAnEvent[node] = true
			return KeyEventResultHandled
		}
		return KeyEventResultIgnored
	}
	withHandler := func(n *FocusNode) *FocusNode { n.SetOnKeyEvent(handle); return n }
	scope1, scope2 := e.scope("Scope 1"), e.scope("Scope 2")
	withHandler(scope1.Node())
	withHandler(scope2.Node())
	parent1, parent2 := withHandler(e.node("Parent 1")), withHandler(e.node("Parent 2"))
	child1, child2, child3, child4 := withHandler(e.node("Child 1")), withHandler(e.node("Child 2")), withHandler(e.node("Child 3")), withHandler(e.node("Child 4"))
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), nil)
	e.put(parent1, scope1.Node())
	e.put(parent2, scope2.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	e.put(child3, parent2)
	e.put(child4, parent2)
	child4.RequestFocus()
	e.pump()
	sendEvent := func() {
		clear(receivedAnEvent)
		e.ts.SendKeyEvent(services.LogicalKeyboardKeyMetaLeft)
	}
	received := func(want ...*FocusNode) {
		t.Helper()
		if len(receivedAnEvent) != len(want) {
			t.Errorf("received = %d nodes, want %d", len(receivedAnEvent), len(want))
			return
		}
		for _, n := range want {
			if !receivedAnEvent[n] {
				t.Errorf("%v did not receive the event", n)
			}
		}
	}
	shouldHandle[scope2.Node()], shouldHandle[parent2], shouldHandle[child2], shouldHandle[child4] = true, true, true, true
	sendEvent()
	received(child4)
	delete(shouldHandle, child4)
	sendEvent()
	received(parent2)
	delete(shouldHandle, parent2)
	sendEvent()
	received(scope2.Node())
	clear(shouldHandle)
	sendEvent()
	received()
	child1.RequestFocus()
	e.pump()
	shouldHandle[scope2.Node()], shouldHandle[parent2], shouldHandle[child2], shouldHandle[child4] = true, true, true, true
	sendEvent()
	// None of the focused nodes handles the event, so nothing receives it.
	received()
}

func TestFocusManagerInitialHighlightModeGuessesCorrectly(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformFuchsia, foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS,
		foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			fm := ts.Binding.FocusManager()
			fm.SetHighlightStrategy(FocusHighlightStrategyAutomatic)
			want := FocusHighlightModeTraditional
			switch platform {
			case foundation.TargetPlatformFuchsia, foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS:
				want = FocusHighlightModeTouch
			}
			if got := fm.HighlightMode(); got != want {
				t.Errorf("highlight mode = %v, want %v", got, want)
			}
		})
	}
}

func TestFocusManagerMouseEventsChangeInitialFocusHighlightModeOnMobile(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia, foundation.TargetPlatformIOS} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewContainer(nil))
			fm := ts.Binding.FocusManager()
			if got := fm.HighlightMode(); got != FocusHighlightModeTouch {
				t.Fatalf("highlight mode = %v, want touch", got)
			}
			ts.Hover(ui.Offset{})
			if got := fm.HighlightMode(); got != FocusHighlightModeTraditional {
				t.Errorf("highlight mode after a mouse event = %v, want traditional", got)
			}
		})
	}
}

func TestFocusManagerMouseEventsChangeInitialFocusHighlightModeOnDesktop(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewContainer(nil))
			fm := ts.Binding.FocusManager()
			if got := fm.HighlightMode(); got != FocusHighlightModeTraditional {
				t.Fatalf("highlight mode = %v, want traditional", got)
			}
			ts.Hover(ui.Offset{})
			if got := fm.HighlightMode(); got != FocusHighlightModeTraditional {
				t.Errorf("highlight mode after a mouse event = %v, want traditional", got)
			}
		})
	}
}

func TestFocusManagerKeyboardEventsChangeInitialFocusHighlightMode(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformLinux, foundation.TargetPlatformIOS} {
		t.Run(platform.String(), func(t *testing.T) {
			usePlatform(t, platform)
			ts := gluttertest.New(t)
			ts.PumpWidget(NewContainer(nil))
			ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
			if got := ts.Binding.FocusManager().HighlightMode(); got != FocusHighlightModeTraditional {
				t.Errorf("highlight mode = %v, want traditional", got)
			}
		})
	}
}

func TestFocusManagerEventsChangeFocusHighlightMode(t *testing.T) {
	// The tests of Flutter run as Android, where the first mode is touch.
	usePlatform(t, foundation.TargetPlatformAndroid)
	ts := gluttertest.New(t)
	ts.PumpWidget(NewContainer(nil))
	fm := ts.Binding.FocusManager()
	callCount := 0
	lastMode := FocusHighlightModeUnspecified
	handle := fm.AddHighlightModeListener(func(mode FocusHighlightMode) { lastMode = mode; callCount++ })
	defer fm.RemoveHighlightModeListener(handle)
	if callCount != 0 || lastMode != FocusHighlightModeUnspecified {
		t.Fatal("the listener was called too early")
	}
	check := func(step string, count int, mode FocusHighlightMode) {
		t.Helper()
		if callCount != count || (lastMode != FocusHighlightModeUnspecified && lastMode != mode) || fm.HighlightMode() != mode {
			t.Errorf("%s: call count %d, last mode %v, mode %v; want %d, %v", step, callCount, lastMode, fm.HighlightMode(), count, mode)
		}
	}
	fm.SetHighlightStrategy(FocusHighlightStrategyAutomatic)
	if fm.HighlightMode() != FocusHighlightModeTouch {
		t.Fatalf("highlight mode = %v, want touch", fm.HighlightMode())
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyMetaLeft)
	check("a key", 1, FocusHighlightModeTraditional)
	ts.TapAt(ui.Offset{})
	check("a touch", 2, FocusHighlightModeTouch)
	g := ts.StartGesture(ui.Offset{}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	g.Up()
	check("a mouse click", 2, FocusHighlightModeTouch)
	ts.TapAt(ui.Offset{})
	check("another touch", 2, FocusHighlightModeTouch)
	fm.SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
	check("always traditional", 3, FocusHighlightModeTraditional)
	fm.SetHighlightStrategy(FocusHighlightStrategyAlwaysTouch)
	check("always touch", 4, FocusHighlightModeTouch)
}

func TestFocusManagerScopesCanBeFocusedWithoutSendingFocusToDescendants(t *testing.T) {
	ts := gluttertest.New(t)
	scopeNode := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope1"}})
	childFocusNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Child1"})
	defer scopeNode.Dispose()
	defer childFocusNode.Dispose()
	ts.PumpWidget(NewFocusScopeWithExternalFocusNode(
		scopeNode,
		NewFocus(
			NewFocusScope(
				NewFocusWithExternalFocusNode(childFocusNode, NewSizedBox(nil)),
				FocusScopeOpts{DebugLabel: "Scope2"}),
			FocusOpts{DebugLabel: "Parent1"})))
	childFocusNode.RequestFocus()
	ts.Pump()
	if !scopeNode.HasFocus() || !childFocusNode.HasPrimaryFocus() {
		t.Fatal("the child should have the focus")
	}
	scopeNode.RequestScopeFocus()
	ts.Pump()
	if !scopeNode.HasPrimaryFocus() || childFocusNode.HasPrimaryFocus() {
		t.Error("the scope should have the primary focus")
	}
}

func TestFocusScopeNodeImplementsDebugFillProperties(t *testing.T) {
	skipUnlessDebug(t)
	builder := foundation.NewDiagnosticPropertiesBuilder()
	scope := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "Scope Label"}})
	defer scope.Dispose()
	scope.DebugFillProperties(builder)
	var description []string
	for _, n := range builder.Properties {
		description = append(description, n.String())
	}
	want := []string{"descendantsAreFocusable: true", "descendantsAreTraversable: true", "canRequestFocus: true", "hasFocus: false", "hasPrimaryFocus: false"}
	if !slices.Equal(description, want) {
		t.Errorf("description = %q, want %q", description, want)
	}
}

func TestFocusManagerDebugDescribeFocusTreeProducesCorrectOutput(t *testing.T) {
	skipUnlessDebug(t)
	e := newFocusEnv(t)
	scope1 := e.scope("Scope 1")
	scope2 := NewFocusScopeNode() // no label, just to test that it works
	e.attach(scope2.Node())
	parent1, parent2 := e.node("Parent 1"), e.node("Parent 2")
	child1, child2 := e.node("Child 1"), NewFocusNode() // no label either
	e.attach(child2)
	child3, child4 := e.node("Child 3"), e.node("Child 4")
	e.put(scope1.Node(), nil)
	e.put(scope2.Node(), nil)
	e.put(parent1, scope1.Node())
	e.put(parent2, scope2.Node())
	e.put(child1, parent1)
	e.put(child2, parent1)
	e.put(child3, parent2)
	e.put(child4, parent2)
	child4.RequestFocus()
	e.pump()
	got := hashRE.ReplaceAllString(DebugDescribeFocusTree(e.ctx), "#00000")
	got = strings.ReplaceAll(got, "Container-[GlobalKey#00000]", "Container-[GlobalKey]")
	// The creator chain depends on the widgets of each implementation: it is
	// checked for its start, and the rest of the dump is compared whole.
	chainEnd := strings.Index(got, " │\n └─rootScope")
	if chainEnd < 0 {
		t.Fatalf("no root scope in\n%s", got)
	}
	head, rest := got[:chainEnd], got[chainEnd:]
	if !strings.HasPrefix(head, "FocusManager#00000\n │ primaryFocus: FocusNode#00000(Child 4 [PRIMARY FOCUS])\n │ primaryFocusCreator: Container-[GlobalKey]") {
		t.Errorf("head =\n%s", head)
	}
	const want = ` │
 └─rootScope: FocusScopeNode#00000(Root Focus Scope [IN FOCUS PATH])
   │ IN FOCUS PATH
   │ focusedChildren: FocusScopeNode#00000([IN FOCUS PATH])
   │
   ├─Child 1: _FocusTraversalGroupNode#00000(FocusTraversalGroup)
   │ │ context: Focus
   │ │ NOT FOCUSABLE
   │ │
   │ └─Child 1: FocusScopeNode#00000(View Scope)
   │     context: FocusScope
   │
   ├─Child 2: FocusScopeNode#00000(Scope 1)
   │ │ context: Container-[GlobalKey]
   │ │
   │ └─Child 1: FocusNode#00000(Parent 1)
   │   │ context: Container-[GlobalKey]
   │   │
   │   ├─Child 1: FocusNode#00000(Child 1)
   │   │   context: Container-[GlobalKey]
   │   │
   │   └─Child 2: FocusNode#00000
   │       context: Container-[GlobalKey]
   │
   └─Child 3: FocusScopeNode#00000([IN FOCUS PATH])
     │ context: Container-[GlobalKey]
     │ IN FOCUS PATH
     │ focusedChildren: FocusNode#00000(Child 4 [PRIMARY FOCUS])
     │
     └─Child 1: FocusNode#00000(Parent 2 [IN FOCUS PATH])
       │ context: Container-[GlobalKey]
       │ IN FOCUS PATH
       │
       ├─Child 1: FocusNode#00000(Child 3)
       │   context: Container-[GlobalKey]
       │
       └─Child 2: FocusNode#00000(Child 4 [PRIMARY FOCUS])
           context: Container-[GlobalKey]
           PRIMARY FOCUS
`
	if rest != want {
		t.Errorf("focus tree =\n%s\nwant\n%s", rest, want)
	}
}

var hashRE = regexp.MustCompile(`#[0-9a-f]{5}`)

func TestAutofocusWorksWhenThePreviousFocusedNodeIsDetached(t *testing.T) {
	ts := gluttertest.New(t)
	node1, node2 := NewFocusNode(), NewFocusNode()
	defer node1.Dispose()
	defer node2.Dispose()
	ts.PumpWidget(NewFocusScope(NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true, FocusNode: node1})))
	ts.Pump()
	if !node1.HasPrimaryFocus() {
		t.Fatal("node1 should have the focus")
	}
	ts.PumpWidget(NewFocusScope(NewSizedBox(NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true, FocusNode: node2}))))
	ts.Pump()
	if !node2.HasPrimaryFocus() {
		t.Error("node2 should have the focus")
	}
}

func TestAutofocusNodeDetachedBeforeAutofocusIsApplied(t *testing.T) {
	ts := gluttertest.New(t)
	scopeNode := NewFocusScopeNode()
	node1 := NewFocusNode()
	defer scopeNode.Dispose()
	defer node1.Dispose()
	ts.PumpWidget(NewFocusScope(NewFocus(NewPlaceholder(nil), FocusOpts{Autofocus: true, FocusNode: node1}), FocusScopeOpts{Node: scopeNode}))
	ts.PumpWidget(NewFocusScope(NewFocus(NewPlaceholder(nil)), FocusScopeOpts{Node: scopeNode}))
	ts.Pump()
	if node1.HasPrimaryFocus() || !scopeNode.HasPrimaryFocus() {
		t.Error("the scope should have the focus: the node was detached before the autofocus")
	}
}

func TestAutofocusTheFirstCandidate(t *testing.T) {
	ts := gluttertest.New(t)
	node1, node2, node3 := NewFocusNode(), NewFocusNode(), NewFocusNode()
	defer node1.Dispose()
	defer node2.Dispose()
	defer node3.Dispose()
	ts.PumpWidget(ltr(NewColumn([]Widget{
		NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: node1}),
		NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: node2}),
		NewFocus(NewSizedBox(nil), FocusOpts{Autofocus: true, FocusNode: node3}),
	})))
	if !node1.HasPrimaryFocus() {
		t.Error("node1 should have the focus")
	}
}

func TestAutofocusWorksWithGlobalKeyReparenting(t *testing.T) {
	ts := gluttertest.New(t)
	node := NewFocusNode()
	scope1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "scope1"}})
	scope2 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "scope2"}})
	defer node.Dispose()
	defer scope1.Dispose()
	defer scope2.Dispose()
	key := NewGlobalKey[State]()
	ts.PumpWidget(ltr(NewColumn([]Widget{
		NewFocusScope(NewFocus(NewSizedBox(nil), FocusOpts{Key: key, FocusNode: node}), FocusScopeOpts{Node: scope1}),
		NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope2}),
	})))
	// The focus changes are applied before the persistent callbacks, so before
	// the context the node is attached to is reparented.
	scope1.Autofocus(node)
	ts.PumpWidget(ltr(NewColumn([]Widget{
		NewFocusScope(NewSizedBox(nil), FocusScopeOpts{Node: scope1}),
		NewFocusScope(NewFocus(NewSizedBox(nil), FocusOpts{Key: key, FocusNode: node}), FocusScopeOpts{Node: scope2}),
	})))
	if !node.HasPrimaryFocus() || !scope2.HasFocus() {
		t.Error("the node should have the focus, in scope2")
	}
}

func TestFocusManagerDoesNotLoseFocusedChildWhenReparentingIfTheNearestScopeDoesNotChange(t *testing.T) {
	e := newFocusEnv(t)
	parent1, parent2 := e.scope("parent1"), e.scope("parent2")
	child1, child2 := e.node("child1"), e.node("child2")
	e.put(parent1.Node(), nil)
	e.put(child1, parent1.Node())
	e.put(child2, child1)
	parent1.Autofocus(child2)
	e.pump()
	e.put(parent2.Node(), nil)
	parent2.RequestFocus()
	e.pump()
	if parent1.FocusedChild() != child2 {
		t.Fatalf("focused child = %v, want child2", parent1.FocusedChild())
	}
	e.att[child2].Reparent(parent1.Node())
	if parent1.FocusedChild() != child2 {
		t.Errorf("focused child after the reparent = %v, want child2", parent1.FocusedChild())
	}
	parent1.RequestFocus()
	e.pump()
	if parent1.FocusedChild() != child2 {
		t.Errorf("focused child after the request = %v, want child2", parent1.FocusedChild())
	}
}

func TestFocusManagerAncestorsGetNotifiedExactlyAsOftenAsNeededIfFocusedChildChangesFocus(t *testing.T) {
	e := newFocusEnv(t)
	top, parent1, parent2 := e.scope("top"), e.scope("parent1"), e.scope("parent2")
	child1, child2 := e.node("child1"), e.node("child2")
	e.put(top.Node(), nil)
	e.put(parent1.Node(), top.Node())
	e.put(parent2.Node(), top.Node())
	e.put(child1, parent1.Node())
	e.put(child2, parent2.Node())
	type seen struct {
		focus  bool
		notify int
	}
	var topS, parent1S, parent2S, child1S, child2S seen
	listen := func(n *FocusNode, s *seen) {
		n.AddListener(func() { s.notify++; s.focus = n.HasFocus() })
	}
	listen(top.Node(), &topS)
	listen(parent1.Node(), &parent1S)
	listen(parent2.Node(), &parent2S)
	listen(child1, &child1S)
	listen(child2, &child2S)
	clearSeen := func() {
		topS, parent1S, parent2S, child1S, child2S = seen{}, seen{}, seen{}, seen{}, seen{}
	}
	check := func(step string, want [5]seen) {
		t.Helper()
		got := [5]seen{topS, parent1S, parent2S, child1S, child2S}
		if got != want {
			t.Errorf("%s: top, parent1, parent2, child1, child2 = %+v, want %+v", step, got, want)
		}
	}
	child1.RequestFocus()
	e.pump()
	check("child1 requested", [5]seen{{true, 1}, {true, 1}, {false, 0}, {true, 1}, {false, 0}})
	clearSeen()
	child1.Unfocus()
	e.pump()
	check("child1 unfocused", [5]seen{{false, 0}, {true, 1}, {false, 0}, {false, 1}, {false, 0}})
	clearSeen()
	child1.RequestFocus()
	e.pump()
	check("child1 requested again", [5]seen{{false, 0}, {true, 1}, {false, 0}, {true, 1}, {false, 0}})
	clearSeen()
	child2.RequestFocus()
	e.pump()
	check("child2 requested", [5]seen{{false, 0}, {false, 1}, {true, 1}, {false, 1}, {true, 1}})
	// Changing the focus back before the pump does not notify.
	clearSeen()
	child1.RequestFocus()
	child2.RequestFocus()
	e.pump()
	check("changed back", [5]seen{})
}

func TestFocusManagerFocusChangesNotifyListeners(t *testing.T) {
	e := newFocusEnv(t)
	parent1, child1, child2 := e.scope("parent1"), e.node("child1"), e.node("child2")
	e.put(parent1.Node(), nil)
	e.put(child1, parent1.Node())
	e.put(child2, child1)
	notifyCount := 0
	handle := e.fm().AddListener(func() { notifyCount++ })
	defer e.fm().RemoveListener(handle)
	steps := []struct {
		name string
		do   func()
	}{
		{"autofocus", func() { parent1.Autofocus(child2) }},
		{"three requests", func() { child1.RequestFocus(); child2.RequestFocus(); child1.RequestFocus() }},
		{"request child2", func() { child2.RequestFocus() }},
		{"unfocus child2", func() { child2.Unfocus() }},
	}
	for i, s := range steps {
		s.do()
		if i == 0 && notifyCount != 0 {
			t.Fatalf("%s: notified before the pump", s.name)
		}
		e.pump()
		if notifyCount != 1 {
			t.Errorf("%s: notify count = %d, want 1", s.name, notifyCount)
		}
		notifyCount = 0
	}
}

// keyHandlerTest is the common part of the tests of the early and late key event
// handlers.
func keyHandlerTest(t *testing.T, add func(*FocusManager, OnKeyEventCallback) FocusHandlerHandle, remove func(*FocusManager, FocusHandlerHandle), early bool) {
	ts := gluttertest.New(t)
	focusNode1 := NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node 1"})
	defer focusNode1.Dispose()
	var logs []int
	focusResult := KeyEventResultIgnored
	ts.PumpWidget(NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: focusNode1,
		OnKeyEvent: func(_ *FocusNode, event services.KeyEvent) KeyEventResult {
			logs = append(logs, 0)
			if _, ok := event.(*services.KeyDownEvent); ok {
				return focusResult
			}
			return KeyEventResultIgnored
		}}))
	focusNode1.RequestFocus()
	ts.Pump()
	result := KeyEventResultIgnored
	handler := func(event services.KeyEvent) KeyEventResult {
		if _, ok := event.(*services.KeyDownEvent); ok {
			return result
		}
		return KeyEventResultIgnored
	}
	press := func(step string, wantDown bool, want []int) {
		t.Helper()
		logs = nil
		down := ts.SendKeyDownEvent(digit1)
		up := ts.SendKeyUpEvent(digit1)
		if down != wantDown || up {
			t.Errorf("%s: down handled %v, up handled %v; want %v, false", step, down, up, wantDown)
		}
		if !slices.Equal(logs, want) {
			t.Errorf("%s: logs = %v, want %v", step, logs, want)
		}
	}
	press("before the handler", false, []int{0, 0})
	handle := add(ts.Binding.FocusManager(), handler)
	// An early handler that handles the event hides it from the nodes.
	handledFirst := []int{0, 0}
	if early {
		handledFirst = []int{0}
	}
	focusResult, result = KeyEventResultIgnored, KeyEventResultHandled
	press("handler handles", true, handledFirst)
	focusResult, result = KeyEventResultIgnored, KeyEventResultIgnored
	press("nobody handles", false, []int{0, 0})
	focusResult, result = KeyEventResultHandled, KeyEventResultIgnored
	press("the node handles", true, []int{0, 0})
	remove(ts.Binding.FocusManager(), handle)
	focusResult, result = KeyEventResultIgnored, KeyEventResultHandled
	press("handler removed", false, []int{0, 0})
	focusResult, result = KeyEventResultHandled, KeyEventResultIgnored
	press("the node handles after the removal", true, []int{0, 0})
}

func TestFocusManagerAddEarlyKeyEventHandlerWorks(t *testing.T) {
	keyHandlerTest(t, (*FocusManager).AddEarlyKeyEventHandler, (*FocusManager).RemoveEarlyKeyEventHandler, true)
}

func TestFocusManagerAddLateKeyEventHandlerWorks(t *testing.T) {
	keyHandlerTest(t, (*FocusManager).AddLateKeyEventHandler, (*FocusManager).RemoveLateKeyEventHandler, false)
}

// Deviation: Flutter ignores a key event when no node has the primary focus;
// here the early and the late key event handlers still see it (SPEC §12.10).
func TestFocusManagerKeyHandlersSeeKeysWithoutPrimaryFocus(t *testing.T) {
	const (
		ignored = KeyEventResultIgnored
		handled = KeyEventResultHandled
		skip    = KeyEventResultSkipRemainingHandlers
	)
	tests := []struct {
		name        string
		early       KeyEventResult
		late        KeyEventResult
		wantHandled bool
		wantEarly   int
		wantLate    int
	}{
		{"early handles", handled, handled, true, 1, 0},
		{"early ignores and late ignores", ignored, ignored, false, 1, 1},
		{"early ignores and late handles", ignored, handled, true, 1, 1},
		{"early skips the remaining handlers", skip, handled, false, 1, 0},
		{"late skips the remaining handlers", ignored, skip, false, 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewSizedBox(nil))
			fm := ts.Binding.FocusManager()
			if fm.PrimaryFocus() != nil {
				t.Fatalf("primary focus = %v, want nil", fm.PrimaryFocus())
			}
			var early, late int
			fm.AddEarlyKeyEventHandler(func(event services.KeyEvent) KeyEventResult {
				if _, ok := event.(*services.KeyDownEvent); ok {
					early++
					return tc.early
				}
				return ignored
			})
			fm.AddLateKeyEventHandler(func(event services.KeyEvent) KeyEventResult {
				if _, ok := event.(*services.KeyDownEvent); ok {
					late++
					return tc.late
				}
				return ignored
			})
			if got := ts.SendKeyDownEvent(digit1); got != tc.wantHandled {
				t.Errorf("HandleKeyData returned %v, want %v", got, tc.wantHandled)
			}
			ts.SendKeyUpEvent(digit1)
			if early != tc.wantEarly || late != tc.wantLate {
				t.Errorf("early handler calls = %d, late handler calls = %d; want %d, %d", early, late, tc.wantEarly, tc.wantLate)
			}
		})
	}
}

func TestFocusManagerWithoutPrimaryFocusAndNoHandlersIgnoresTheKey(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewSizedBox(nil))
	if ts.Binding.FocusManager().PrimaryFocus() != nil {
		t.Fatal("primary focus is not nil")
	}
	if ts.SendKeyDownEvent(digit1) {
		t.Error("a key was handled with no focus and no handler")
	}
	ts.SendKeyUpEvent(digit1)
}

// TestFocusManagerEarlyHandlerClaimsAKeyWhileTheFocusMovesOn is the case of a
// tab that closes while the next one takes its place: the focused field is
// disposed, so nothing has the primary focus until the focus update runs.
func TestFocusManagerEarlyHandlerClaimsAKeyWhileTheFocusMovesOn(t *testing.T) {
	tests := []struct {
		name        string
		result      KeyEventResult
		wantHandled bool
		wantSeenByB int
	}{
		{"claimed", KeyEventResultHandled, true, 0},
		{"not claimed", KeyEventResultIgnored, false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			nodeA := NewFocusNode(FocusNodeOpts{DebugLabel: "A"})
			nodeB := NewFocusNode(FocusNodeOpts{DebugLabel: "B"})
			defer nodeB.Dispose()
			var seenByA, seenByB, early int
			count := func(n *int) FocusOnKeyEventCallback {
				return func(_ *FocusNode, event services.KeyEvent) KeyEventResult {
					if _, ok := event.(*services.KeyDownEvent); ok {
						*n++
					}
					return KeyEventResultIgnored
				}
			}
			ts.PumpWidget(NewColumn([]Widget{
				NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: nodeA, OnKeyEvent: count(&seenByA)}),
				NewFocus(NewSizedBox(nil), FocusOpts{FocusNode: nodeB, OnKeyEvent: count(&seenByB)}),
			}))
			nodeA.RequestFocus()
			ts.Pump()
			fm := ts.Binding.FocusManager()
			if fm.PrimaryFocus() != nodeA {
				t.Fatalf("primary focus = %v, want A", fm.PrimaryFocus())
			}
			fm.AddEarlyKeyEventHandler(func(event services.KeyEvent) KeyEventResult {
				if _, ok := event.(*services.KeyDownEvent); ok {
					early++
					return tc.result
				}
				return KeyEventResultIgnored
			})
			// The field is disposed and the next one asks for the focus, but the
			// focus update has not run.
			nodeA.Dispose()
			nodeB.RequestFocus()
			if fm.PrimaryFocus() != nil {
				t.Fatalf("primary focus = %v, want nil before the focus update", fm.PrimaryFocus())
			}
			if got := ts.SendKeyDownEvent(digit1); got != tc.wantHandled {
				t.Errorf("HandleKeyData returned %v, want %v", got, tc.wantHandled)
			}
			ts.SendKeyUpEvent(digit1)
			if early != 1 {
				t.Errorf("early handler calls = %d, want 1", early)
			}
			// The focus update runs: B has the focus and A saw nothing.
			ts.Pump()
			if fm.PrimaryFocus() != nodeB {
				t.Fatalf("primary focus = %v, want B", fm.PrimaryFocus())
			}
			if seenByA != 0 {
				t.Errorf("A saw %d keys, want 0", seenByA)
			}
			if seenByB != 0 {
				t.Errorf("B saw %d keys of the first press, want 0", seenByB)
			}
			// A key of the next press goes through the early handler and, when it
			// was not claimed, reaches B.
			ts.SendKeyDownEvent(digit1)
			ts.SendKeyUpEvent(digit1)
			if want := tc.wantSeenByB; seenByB != want {
				t.Errorf("B saw %d keys after the next press, want %d", seenByB, want)
			}
		})
	}
}

func TestFocusManagerNotifiesListenersWhenAWidgetLosesFocusBecauseItWasRemoved(t *testing.T) {
	ts := gluttertest.New(t)
	nodeA := NewFocusNode(FocusNodeOpts{DebugLabel: "a"})
	nodeB := NewFocusNode(FocusNodeOpts{DebugLabel: "b"})
	defer nodeA.Dispose()
	defer nodeB.Dispose()
	rtl := func(children ...Widget) Widget { return NewDirectionality(ui.TextDirectionRTL, NewColumn(children)) }
	ts.PumpWidget(rtl(NewFocus(NewText("a"), FocusOpts{FocusNode: nodeA}), NewFocus(NewText("b"), FocusOpts{FocusNode: nodeB})))
	notifyCount := 0
	handle := ts.Binding.FocusManager().AddListener(func() { notifyCount++ })
	defer ts.Binding.FocusManager().RemoveListener(handle)
	nodeA.RequestFocus()
	ts.Pump()
	if !nodeA.HasPrimaryFocus() || notifyCount != 1 {
		t.Fatalf("nodeA has the focus %v, notify count %d", nodeA.HasPrimaryFocus(), notifyCount)
	}
	notifyCount = 0
	ts.PumpWidget(rtl(NewFocus(NewText("b"), FocusOpts{FocusNode: nodeB})))
	ts.Pump()
	if nodeA.HasPrimaryFocus() || nodeB.HasPrimaryFocus() || notifyCount != 1 {
		t.Errorf("nodeA %v, nodeB %v, notify count %d; want false, false, 1", nodeA.HasPrimaryFocus(), nodeB.HasPrimaryFocus(), notifyCount)
	}
}

func TestFocusManagerDebugFocusChangesCausesLoggingOfFocusChanges(t *testing.T) {
	skipUnlessDebug(t)
	var messages strings.Builder
	debug := NewDebugOptions()
	debug.FocusChanges = true
	debug.Print = func(line string) { messages.WriteString(line + "\n") }
	ts := gluttertest.New(t, gluttertest.Opts{Debug: debug})
	ctx := focusContext(ts)
	parent1, child1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "parent1"}}), NewFocusNode(FocusNodeOpts{DebugLabel: "child1"})
	defer parent1.Dispose()
	defer child1.Dispose()
	parent1.Attach(ctx).Reparent(ts.Binding.FocusManager().RootScope().Node())
	child1.Attach(ctx).Reparent(parent1.Node())
	notifyCount := 0
	handle := ts.Binding.FocusManager().AddListener(func() { notifyCount++ })
	defer ts.Binding.FocusManager().RemoveListener(handle)
	parent1.RequestFocus()
	if notifyCount != 0 {
		t.Fatal("notified before the pump")
	}
	ts.Pump()
	if notifyCount != 1 {
		t.Fatalf("notify count = %d, want 1", notifyCount)
	}
	child1.RequestFocus()
	ts.Pump()
	if notifyCount != 2 {
		t.Fatalf("notify count = %d, want 2", notifyCount)
	}
	got := messages.String()
	for _, re := range []string{
		`   └─Child \d+: FocusScopeNode#[a-f0-9]{5}\(parent1 \[PRIMARY FOCUS\]\)`,
		`FOCUS: Notified 2 dirty nodes`,
		`FOCUS: Scheduling update, current focus is null, next focus will be FocusScopeNode#.*parent1`,
	} {
		if !regexp.MustCompile(re).MatchString(got) {
			t.Errorf("the log does not match %q:\n%s", re, got)
		}
	}
}

func TestFocusManagerDoesNotDescribeNodesWhenDebugFocusChangesIsFalse(t *testing.T) {
	skipUnlessDebug(t)
	var messages strings.Builder
	debug := NewDebugOptions()
	debug.Print = func(line string) { messages.WriteString(line + "\n") }
	ts := gluttertest.New(t, gluttertest.Opts{Debug: debug})
	run := func() {
		ctx := focusContext(ts)
		parent1, child1 := NewFocusScopeNode(FocusScopeNodeOpts{FocusNodeOpts: FocusNodeOpts{DebugLabel: "parent1"}}), NewFocusNode(FocusNodeOpts{DebugLabel: "child1"})
		parent1.Attach(ctx).Reparent(ts.Binding.FocusManager().RootScope().Node())
		child1.Attach(ctx).Reparent(parent1.Node())
		child1.RequestFocus()
		ts.Pump()
		child1.Dispose()
		parent1.Dispose()
		ts.Pump()
	}
	run()
	if messages.Len() != 0 {
		t.Fatalf("messages = %q, want none", messages.String())
	}
	debug.FocusChanges = true
	run()
	if !strings.Contains(messages.String(), "FOCUS: Notified 3 dirty nodes:") {
		t.Errorf("messages = %q", messages.String())
	}
}
