package widgets

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

func TestRootWidgetDescription(t *testing.T) {
	if got := NewRootWidget(nil, RootWidgetOpts{DebugShortDescription: "[root]"}).ToStringShort(); got != "[root]" {
		t.Errorf("ToStringShort = %q, want [root]", got)
	}
	if got := NewRootWidget(nil).ToStringShort(); got != "RootWidget" {
		t.Errorf("ToStringShort = %q, want RootWidget", got)
	}
}

// TestAttachRootWidgetTwiceUpdates checks that a second AttachRootWidget
// updates the existing root element instead of making a new one.
func TestAttachRootWidgetTwiceUpdates(t *testing.T) {
	bt := newBindingTree(t)
	builds := 0
	mk := func(label string) Widget {
		return builderOf(func(BuildContext) Widget { builds++; return newLeaf(WidgetOpts{Key: foundation.NewValueKey(label)}) })
	}
	bt.b.AttachRootWidget(mk("a"))
	bt.frame()
	root := bt.b.RootElement()
	if root == nil || builds != 1 {
		t.Fatalf("root = %v, builds = %d", root, builds)
	}
	bt.b.AttachRootWidget(mk("b"))
	if !bt.b.Scheduler().HasScheduledFrame() {
		t.Error("replacing the root widget did not schedule a frame")
	}
	bt.frame()
	if bt.b.RootElement() != root {
		t.Error("the root element was replaced")
	}
	if builds != 2 {
		t.Errorf("builds = %d, want 2", builds)
	}
}

// TestRootElementReportsAFailureToAttach checks _rebuild of RootElement: a panic
// while the child is created is reported with the context 'attaching to the
// render tree' and leaves the tree empty.
func TestRootElementReportsAFailureToAttach(t *testing.T) {
	bt := newBindingTree(t, bindingTreeOpts{collectPanics: true})
	// A widget whose state panics in InitState makes UpdateChild itself panic.
	bad := newSpy(newLeaf())
	bad.onInitState = func(BuildContext) { panic("cannot attach") }
	bt.b.AttachRootWidget(bad)
	if len(bt.errs) != 1 {
		t.Fatalf("%d errors, want 1: %v", len(bt.errs), bt.errs)
	}
	appErr := requireErrorTag(t, bt.errs[0], foundation.BuildPanicked)
	if appErr.Metadata["context"] != "attaching to the render tree" {
		t.Errorf("context = %v", appErr.Metadata["context"])
	}
	if !strings.Contains(appErr.Error(), "cannot attach") {
		t.Errorf("error = %v", appErr)
	}
	var children int
	bt.b.RootElement().VisitChildren(func(Element) { children++ })
	if children != 0 {
		t.Errorf("the root has %d children, want 0", children)
	}
}

func TestBindingReassemble(t *testing.T) {
	bt := newBindingTree(t)
	builds := 0
	bt.pump(builderOf(func(BuildContext) Widget { builds++; return newBox(10, 10) }))
	before := builds
	bt.b.Reassemble()
	bt.frame()
	if builds <= before {
		t.Errorf("builds = %d after Reassemble, want more than %d", builds, before)
	}
}

func TestReassembleNeedsTheRoot(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newMulti(newLeaf()))
	child := elementOfType[*multi](t, tr)
	requireTag(t, "ReassembleNotRoot", func() { tr.owner.Reassemble(child) })
}

func TestDebugOptionsPrint(t *testing.T) {
	skipUnlessDebug(t)
	var lines []string
	dbg := NewDebugOptions()
	dbg.Print = func(l string) { lines = append(lines, l) }
	dbg.PrintRebuildDirtyWidgets = true
	dbg.PrintBuildScope = true
	var setState func(func())
	bt := newBindingTree(t, bindingTreeOpts{debug: dbg})
	bt.pump(newBuilder(func(_ BuildContext, ss func(func())) Widget { setState = ss; return newLeaf() }))
	has := func(prefix string) int {
		n := 0
		for _, l := range lines {
			if strings.HasPrefix(l, prefix) {
				n++
			}
		}
		return n
	}
	if has("Building builder") == 0 || has("buildScope called with context") == 0 || has("buildScope finished") == 0 {
		t.Errorf("lines = %q, want 'Building', 'buildScope called' and 'buildScope finished'", lines)
	}
	lines = nil
	setState(func() {})
	bt.frame()
	if has("Rebuilding builder") != 1 {
		t.Errorf("lines = %q, want one 'Rebuilding builder'", lines)
	}
}

func TestDebugOptionsStacksAndKeyedLifecycle(t *testing.T) {
	skipUnlessDebug(t)
	var lines []string
	dbg := NewDebugOptions()
	dbg.Print = func(l string) { lines = append(lines, l) }
	dbg.PrintScheduleBuildForStacks = true
	dbg.PrintGlobalKeyedWidgetLifecycle = true
	var setState func(func())
	bt := newBindingTree(t, bindingTreeOpts{debug: dbg})
	key := NewGlobalKey[*builderState]()
	bt.pump(newMulti(newBuilder(func(_ BuildContext, ss func(func())) Widget { setState = ss; return newLeaf() }, WidgetOpts{Key: key})))
	setState(func() {})
	if n := countPrefix(lines, "scheduleBuildFor() called for"); n != 1 {
		t.Errorf("%d scheduleBuildFor lines, want 1: %q", n, lines)
	}
	lines = nil
	bt.pump(newMulti())
	if countPrefix(lines, "Deactivated") != 1 || countPrefix(lines, "Discarding") != 1 {
		t.Errorf("lines = %q, want 'Deactivated' and 'Discarding' for the keyed widget", lines)
	}
}

func countPrefix(lines []string, prefix string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

func TestRootElementMustBeTheRoot(t *testing.T) {
	skipUnlessDebug(t)
	tr := newTree(t)
	tr.pump(newMulti(newLeaf()))
	parent := elementOfType[*multi](t, tr)
	e := NewRootWidget(newLeaf()).CreateElement().(*RootElement)
	e.AssignOwner(tr.owner)
	requireTag(t, "RootElementHasParent", func() { e.Mount(parent, nil) })
}
